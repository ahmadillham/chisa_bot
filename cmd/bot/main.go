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

	"chisa_bot/internal/config"
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

	// Initialize handlers using config values for file paths.
	warnStore := services.NewWarnStore(config.WarnStoreFile)
	autoTagStore := services.NewAutoTagStore(config.AutoTagStoreFile)
	bannedStickerStore := services.NewBannedStickerStore(config.BannedStickerStoreFile, nil)
	mediaHandler := handlers.NewMediaHandler()
	dlHandler := handlers.NewDownloaderHandler()
	groupHandler := handlers.NewGroupHandler(warnStore, autoTagStore)
	menuHandler := handlers.NewMenuHandler()
	sysHandler := handlers.NewSystemHandler()
	antiStickerHandler := handlers.NewAntiStickerHandler(bannedStickerStore, groupHandler)
	limiter := ratelimit.New(
		time.Duration(config.RateLimitUserCooldownSec)*time.Second,
		config.RateLimitChatMax,
		time.Duration(config.RateLimitChatWindowSec)*time.Second,
	)

	// Helper to wrap handlers that don't take args
	wrap := func(h func(*whatsmeow.Client, *events.Message)) handlers.CommandHandler {
		return func(c *whatsmeow.Client, e *events.Message, _ []string) {
			h(c, e)
		}
	}

	// Initialize Registry
	registry := handlers.NewRegistry()
	registry.Register("s", wrap(mediaHandler.HandleSticker))
	registry.Register("toimg", wrap(mediaHandler.HandleImage))
	registry.Register("ts", mediaHandler.HandleTextSticker)

	registry.Register("dl", dlHandler.HandleVideo)
	registry.Register("mp3", dlHandler.HandleAudio)

	registry.Register("tagall", wrap(groupHandler.HandleTagAll))
	registry.Register("warn", groupHandler.HandleWarn)
	registry.Register("resetwarn", groupHandler.HandleResetWarn)
	registry.Register("kick", groupHandler.HandleKick)
	registry.Register("autotag", groupHandler.HandleAutoTag)

	registry.Register("bansticker", antiStickerHandler.HandleBanSticker)
	registry.Register("ban", antiStickerHandler.HandleBanSticker) // alias
	registry.Register("unbansticker", antiStickerHandler.HandleUnbanSticker)
	registry.Register("unban", antiStickerHandler.HandleUnbanSticker) // alias
	registry.Register("liststicker", antiStickerHandler.HandleListBanned)

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
				handleMessage(client, evt, registry, groupHandler, antiStickerHandler, limiter, autoTagStore)
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
	antiStickerHandler *handlers.AntiStickerHandler,
	limiter *ratelimit.Limiter,
	autoTagStore *services.AutoTagStore,
) {
	// Anti-sticker check: revoke banned stickers BEFORE anything else.
	if antiStickerHandler.CheckAndRevoke(client, evt) {
		return // Message was revoked, no further processing needed.
	}

	// Extract text from various message types.
	text := utils.GetTextFromMessage(evt)
	if text == "" {
		return
	}

	// 1. Check for commands first.
	parsed := router.Parse(text)
	if parsed != nil {
		// Rate limit check — only for commands, not regular messages.
		if !evt.Info.IsFromMe {
			switch limiter.Check(evt.Info.Sender.String(), evt.Info.Chat.String()) {
			case ratelimit.UserCooldown:
				utils.ReplyTextDirect(client, evt, config.MsgRateLimitUser)
				return
			case ratelimit.ChatRateLimit:
				utils.ReplyTextDirect(client, evt, config.MsgRateLimitChat)
				return
			}
		}

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
			log.Printf("[AUTO-TAG] TikTok link detected in %s", evt.Info.Chat.String())
			groupHandler.TagAll(client, evt.Info.Chat, evt.Message, evt.Info.ID, evt.Info.Sender, text)
		}
	}
}
