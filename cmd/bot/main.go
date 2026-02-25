package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
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
	warnStore := services.NewWarnStore("warnings.json")
	autoTagStore := services.NewAutoTagStore("autotag.json")
	mediaHandler := handlers.NewMediaHandler()
	dlHandler := handlers.NewDownloaderHandler()
	groupHandler := handlers.NewGroupHandler(warnStore, autoTagStore)
	menuHandler := handlers.NewMenuHandler()
	sysHandler := handlers.NewSystemHandler()
	limiter := ratelimit.New(3*time.Second, 10, time.Minute)

	// Helper to wrap handlers that don't take args
	wrap := func(h func(*whatsmeow.Client, *events.Message)) handlers.CommandHandler {
		return func(c *whatsmeow.Client, e *events.Message, _ []string) {
			h(c, e)
		}
	}

	// Initialize Registry
	registry := handlers.NewRegistry()
	registry.Register("s", wrap(mediaHandler.HandleSticker))
	// Unified image command
	registry.Register("toimg", wrap(mediaHandler.HandleImage))
	// Removed aliases: .show, .showimg, .rv, .img, .image
	
	// Downloader handlers already take args, so they match CommandHandler sort of? 
	// Wait, DownloaderHandler.HandleVideo takes (client, evt, args). 
	// Let's check signature in internal/handlers/downloader.go...
	// It takes (client, evt, args). So it matches directly.
	
	registry.Register("dl", dlHandler.HandleVideo)
	registry.Register("mp3", dlHandler.HandleAudio)
	
	registry.Register("tagall", wrap(groupHandler.HandleTagAll))
	registry.Register("warn", groupHandler.HandleWarn)
	registry.Register("resetwarn", groupHandler.HandleResetWarn)
	registry.Register("kick", groupHandler.HandleKick) // HandleKick takes args
	registry.Register("autotag", groupHandler.HandleAutoTag)
	
	registry.Register("menu", wrap(menuHandler.HandleMenu))
	registry.Register("stat", wrap(sysHandler.HandleStats))

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
				handleMessage(client, evt, registry, groupHandler, limiter, autoTagStore)
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
	registry *handlers.Registry,
	groupHandler *handlers.GroupHandler,
	limiter *ratelimit.Limiter,
	autoTagStore *services.AutoTagStore,
) {
	// Extract text from various message types.
	text := utils.GetTextFromMessage(evt)
	if text == "" {
		return
	}

	// Rate limit check.
	// Optionally bypass rate limit for self
	if !evt.Info.IsFromMe {
		switch limiter.Check(evt.Info.Sender.String(), evt.Info.Chat.String()) {
		case ratelimit.UserCooldown:
			return
		case ratelimit.ChatRateLimit:
			return
		}
	}

	// 1. Check for commands first.
	parsed := router.Parse(text)
	if parsed != nil {
		log.Printf("[CMD] %s | from: %s | chat: %s", parsed.Command, evt.Info.Sender.User, evt.Info.Chat.String())
		registry.Execute(client, evt, parsed.Command, parsed.Args)
		return
	}

	// Ignore non-command messages from self to prevent infinite loops (e.g. from Auto-Tag)
	if evt.Info.IsFromMe {
		return
	}

	// 2. If not a command, check for TikTok links in group chats.
	if evt.Info.IsGroup && (strings.Contains(text, "tiktok.com/") || strings.Contains(text, "vm.tiktok.com/")) {
		if !autoTagStore.IsDisabled(evt.Info.Chat.String()) {
			// Log found link
			log.Printf("[AUTO-TAG] TikTok link detected in %s", evt.Info.Chat.String())
			
			// Trigger TagAll with custom message
			tagTitle := fmt.Sprintf("%s", text)
			groupHandler.TagAll(client, evt.Info.Chat, evt.Message, evt.Info.ID, evt.Info.Sender, tagTitle)
		}
	}
}
