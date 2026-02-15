package  main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/mdp/qrterminal/v3"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types/events"
	waLog "go.mau.fi/whatsmeow/util/log"

	"chisa_bot/internal/handlers"
	"chisa_bot/internal/router"
	"chisa_bot/pkg/ratelimit"
	"chisa_bot/pkg/utils"
)

func main() {
	// Initialize SQLite store for sessions.
	dbLog := waLog.Stdout("Database", "WARN", true)
	container, err := sqlstore.New(context.Background(), "sqlite3", "file:session.db?_journal_mode=WAL&_busy_timeout=5000&_foreign_keys=on", dbLog)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}

	// Get the first device from the store, or create a new one.
	deviceStore, err := container.GetFirstDevice(context.Background())
	if err != nil {
		log.Fatalf("Failed to get device: %v", err)
	}

	clientLog := waLog.Stdout("Client", "WARN", true)
	client := whatsmeow.NewClient(deviceStore, clientLog)

	// Initialize handlers.
	mediaHandler := handlers.NewMediaHandler()
	dlHandler := handlers.NewDownloaderHandler()
	groupHandler := handlers.NewGroupHandler()
	menuHandler := handlers.NewMenuHandler()
	sysHandler := handlers.NewSystemHandler()
	limiter := ratelimit.New(3*time.Second, 20, time.Minute)

	// Register the main event handler.
	client.AddEventHandler(func(rawEvt interface{}) {
		switch evt := rawEvt.(type) {

		case *events.Message:
			// Process message commands in a goroutine to avoid blocking.
			go func() {
				defer func() {
					if r := recover(); r != nil {
						log.Printf("[PANIC RECOVERED] %v", r)
					}
				}()
				handleMessage(client, evt, mediaHandler, dlHandler, groupHandler, menuHandler, sysHandler, limiter)
			}()

		case *events.GroupInfo:
			// Handle group join/leave events in a goroutine.
			go func() {
				defer func() {
					if r := recover(); r != nil {
						log.Printf("[PANIC RECOVERED] %v", r)
					}
				}()
				groupHandler.HandleGroupParticipants(client, evt)
			}()

		case *events.Connected:
			log.Println("✅ Bot connected successfully!")

		case *events.LoggedOut:
			log.Println("⚠️ Bot logged out. Please re-authenticate.")

		case *events.StreamReplaced:
			log.Println("⚠️ Stream replaced (another device connected).")
		}
	})

	// Connect to WhatsApp.
	if client.Store.ID == nil {
		// No session found, generate QR code for login.
		qrChan, _ := client.GetQRChannel(context.Background())
		if err := client.Connect(); err != nil {
			log.Fatalf("Failed to connect: %v", err)
		}

		for evt := range qrChan {
			switch evt.Event {
			case "code":
				fmt.Println("\n📱 Scan QR Code below to login:")
				qrterminal.GenerateHalfBlock(evt.Code, qrterminal.L, os.Stdout)
				fmt.Println()
			case "login":
				log.Println("✅ Login successful!")
			case "timeout":
				log.Println("❌ QR code timed out. Please restart the bot.")
				os.Exit(1)
			}
		}
	} else {
		// Session exists, connect directly.
		if err := client.Connect(); err != nil {
			log.Fatalf("Failed to connect: %v", err)
		}
		log.Println("🔄 Reconnected using saved session.")
	}



	// Graceful shutdown on OS signals.
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	log.Println("🤖 Bot is now running. Press Ctrl+C to stop.")
	<-sigChan

	log.Println("🛑 Shutting down gracefully...")
	client.Disconnect()
	log.Println("👋 Bot stopped. Goodbye!")
}

// handleMessage parses and routes incoming messages to the appropriate handler.
func handleMessage(
	client *whatsmeow.Client,
	evt *events.Message,
	media *handlers.MediaHandler,
	dl *handlers.DownloaderHandler,
	group *handlers.GroupHandler,
	menu *handlers.MenuHandler,
	sys *handlers.SystemHandler,
	limiter *ratelimit.Limiter,
) {
	// Ignore messages from self.
	if evt.Info.IsFromMe {
		return
	}

	// Extract text from various message types.
	text := utils.GetTextFromMessage(evt)
	if text == "" {
		return
	}

	// Parse the command.
	parsed := router.Parse(text)
	if parsed == nil {
		return
	}

	// Rate limit check.
	switch limiter.Check(evt.Info.Sender.String(), evt.Info.Chat.String()) {
	case ratelimit.UserCooldown:
		return
	case ratelimit.ChatRateLimit:
		return
	}

	log.Printf("[CMD] %s | from: %s | chat: %s", parsed.Command, evt.Info.Sender.User, evt.Info.Chat.String())

	// Route to appropriate handler.
	switch parsed.Command {
	case "sticker", "s":
		media.HandleSticker(client, evt)
	case "toimg":
		media.HandleStickerToImage(client, evt)
	case "show", "showimg", "rv":
		media.HandleRetrieveViewOnce(client, evt)
	case "dl", "tiktok", "tt", "ig", "instagram", "ytmp4":
		dl.HandleVideo(client, evt, parsed.Args)
	case "mp3", "ytmp3":
		dl.HandleAudio(client, evt, parsed.Args)
	case "tagall":
		group.HandleTagAll(client, evt)
	case "kick", "usir":
		group.HandleKick(client, evt, parsed.Args)
	case "menu", "help":
		menu.HandleMenu(client, evt)
	case "stats", "server", "stat":
		sys.HandleStats(client, evt)
	default:
		// Unknown command — silently ignore.
	}
}
