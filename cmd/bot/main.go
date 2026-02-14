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
	"chisa_bot/internal/services"
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
	funHandler := handlers.NewFunHandler()
	menuHandler := handlers.NewMenuHandler()
	sysHandler := handlers.NewSystemHandler()
	limiter := ratelimit.New(3*time.Second, 20, time.Minute)

	// Game Handler
	gameStore := services.NewGameStore("data/leaderboard.json")
	gameHandler := handlers.NewGameHandler(gameStore)

	// Utility handler.
	utilHandler := handlers.NewUtilityHandler()

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
				handleMessage(client, evt, mediaHandler, dlHandler, groupHandler, funHandler, menuHandler, sysHandler, limiter, gameHandler, utilHandler)
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
	fun *handlers.FunHandler,
	menu *handlers.MenuHandler,
	sys *handlers.SystemHandler,
	limiter *ratelimit.Limiter,
	game *handlers.GameHandler,
	util *handlers.UtilityHandler,
) {
	// Ignore messages from self.
	if evt.Info.IsFromMe {
		return
	}

	// Only work in groups.
	if !evt.Info.IsGroup {
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
		// Check for game answer if no command prefix
		game.HandleAnswer(client, evt)
		return
	}

	// Rate limit check.
	switch limiter.Check(evt.Info.Sender.String(), evt.Info.Chat.String()) {
	case ratelimit.UserCooldown:
		// Silently ignore — don't even reply to avoid more messages.
		return
	case ratelimit.ChatRateLimit:
		// Silently ignore.
		return
	}

	log.Printf("[CMD] %s | from: %s | chat: %s", parsed.Command, evt.Info.Sender.User, evt.Info.Chat.String())

	// Route to appropriate handler.
	switch parsed.Command {
	// Media commands
	case "sticker", "s":
		media.HandleSticker(client, evt)
	case "toimg":
		media.HandleStickerToImage(client, evt)
	case "show", "showimg", "rv":
		media.HandleRetrieveViewOnce(client, evt)

	// Downloader commands (Smart DL)
	case "dl", "tiktok", "tt", "ig", "instagram", "ytmp4":
		dl.HandleVideo(client, evt, parsed.Args)
	case "mp3", "ytmp3":
		dl.HandleAudio(client, evt, parsed.Args)

	// Group commands
	case "tagall":
		group.HandleTagAll(client, evt)
	case "kick", "usir":
		group.HandleKick(client, evt, parsed.Args)

	// Fun commands
	case "kerangajaib":
		fun.HandleKerangAjaib(client, evt, parsed.RawArgs)
	case "cekkhodam":
		fun.HandleCekKhodam(client, evt, parsed.RawArgs)
	case "cekjodoh":
		fun.HandleCekJodoh(client, evt, parsed.Args)
	case "rate":
		fun.HandleRate(client, evt, parsed.RawArgs)
	case "roast":
		fun.HandleRoast(client, evt, parsed.RawArgs)
	case "siapadia":
		fun.HandleSiapaDia(client, evt, parsed.RawArgs)
	case "seberapa":
		fun.HandleSeberapa(client, evt, parsed.RawArgs)


	// Info commands
	case "menu", "help":
		menu.HandleMenu(client, evt)
	case "stats", "server", "stat":
		sys.HandleStats(client, evt)





	// Game commands
	case "tebakkata":
		game.HandleTebakKata(client, evt)
	case "tebakibukota":
		game.HandleTebakIbuKota(client, evt)
	case "tebaknegara":
		game.HandleTebakNegara(client, evt)
	case "tebakbenda":
		game.HandleTebakBenda(client, evt)
	case "tebakbendera":
		game.HandleTebakBendera(client, evt)
	case "tebakangka":
		game.HandleTebakAngka(client, evt)
	case "kuis":
		game.HandleTebakKuis(client, evt)
	case "nyerah", "skip":
		game.HandleNyerah(client, evt)
	case "leaderboard", "lb":
		game.HandleLeaderboard(client, evt)

	// Utility commands
	case "pick", "pilih":
		util.HandlePick(client, evt, parsed.RawArgs)
	case "short", "shorten", "pendek":
		util.HandleShortLink(client, evt, parsed.Args)

	default:
		// Unknown command — silently ignore.
	}
}
