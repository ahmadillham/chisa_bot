package main

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
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

	"chisa_bot/internal/config"
	"chisa_bot/internal/handlers"
	"chisa_bot/internal/router"
	"chisa_bot/internal/services"
	"chisa_bot/pkg/ratelimit"
	"chisa_bot/pkg/utils"
)

func main() {
	// Load configuration
	config.Load()

	// Initialize Logger
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	// Initialize SQLite store for sessions.
	dbLog := waLog.Stdout("Database", "WARN", true)
	container, err := sqlstore.New(context.Background(), "sqlite3", "file:session.db?_journal_mode=WAL&_busy_timeout=5000&_foreign_keys=on", dbLog)
	if err != nil {
		slog.Error("Failed to initialize database", "error", err)
		os.Exit(1)
	}

	// Get the first device from the store, or create a new one.
	deviceStore, err := container.GetFirstDevice(context.Background())
	if err != nil {
		slog.Error("Failed to get device", "error", err)
		os.Exit(1)
	}

	clientLog := waLog.Stdout("Client", "WARN", true)
	client := whatsmeow.NewClient(deviceStore, clientLog)

	botDB, err := sql.Open("sqlite3", "file:"+config.BotDatabaseFile+"?_journal_mode=WAL&_foreign_keys=on")
	if err != nil {
		slog.Error("Failed to initialize bot DB", "error", err)
		os.Exit(1)
	}

	// Initialize handlers using the bot SQLite DB.
	bannedStickerUserStore := services.NewBannedStickerUserStore(botDB)
	msgCache := services.NewMessageCacheStore(botDB)

	pool := services.NewWorkerPool(config.MaxConcurrentMediaTasks)

	mediaHandler := handlers.NewMediaHandler(pool)
	dlHandler := handlers.NewDownloaderHandler(pool)
	groupHandler := handlers.NewGroupHandler()
	menuHandler := handlers.NewMenuHandler()
	sysHandler := handlers.NewSystemHandler(msgCache)
	antiStickerHandler := handlers.NewAntiStickerHandler(bannedStickerUserStore, groupHandler)

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
	registry.Register("brat", mediaHandler.HandleBrat)

	registry.Register("dl", dlHandler.HandleVideo)
	registry.Register("mp3", dlHandler.HandleAudio)

	registry.Register("tagall", wrap(groupHandler.HandleTagAll))
	registry.Register("kick", groupHandler.HandleKick)

	registry.Register("banuser", antiStickerHandler.HandleBanStickerUser)
	registry.Register("unbanuser", antiStickerHandler.HandleUnbanStickerUser)
	registry.Register("listuser", antiStickerHandler.HandleListBannedUsers)

	registry.Register("menu", wrap(menuHandler.HandleMenu))
	registry.Register("stat", wrap(sysHandler.HandleStats))
	registry.Register("read", wrap(sysHandler.HandleRecover))

	// Register the main event handler.
	client.AddEventHandler(func(rawEvt interface{}) {
		switch evt := rawEvt.(type) {

		case *events.Message:
			// Save the original intact message to SQLite for Anti-Delete features
			msgCache.Save(evt.Info.ID, evt.Message)

			// Process message commands in a goroutine to avoid blocking.
			go func() {
				defer func() {
					if r := recover(); r != nil {
						slog.Error("PANIC RECOVERED", "panic", r)
					}
				}()
				handleMessage(client, evt, registry, antiStickerHandler, limiter)
			}()

		case *events.GroupInfo:
			// Handle group join/leave events in a goroutine.
			go func() {
				defer func() {
					if r := recover(); r != nil {
						slog.Error("PANIC RECOVERED in group handler", "panic", r)
					}
				}()
				groupHandler.HandleGroupParticipants(client, evt)
			}()

		case *events.Connected:
			slog.Info("Bot connected successfully!")

		case *events.LoggedOut:
			slog.Info("Bot logged out. Please re-authenticate.")

		case *events.StreamReplaced:
			slog.Info("Stream replaced (another device connected).")
		}
	})

	// Start temporary files auto-cleaner (hourly scan, delete files older than 1 hour)
	services.StartTempCleaner(1*time.Hour, 1*time.Hour)

	// Background cleanup for message cache (once a day)
	go func() {
		ticker := time.NewTicker(24 * time.Hour)
		defer ticker.Stop()
		for range ticker.C {
			msgCache.Clean()
		}
	}()

	// Connect to WhatsApp.
	if client.Store.ID == nil {
		// No session found, generate QR code for login.
		qrChan, _ := client.GetQRChannel(context.Background())
		if err := client.Connect(); err != nil {
			slog.Error("Failed to connect", "error", err)
			os.Exit(1)
		}

		for evt := range qrChan {
			switch evt.Event {
			case "code":
				fmt.Println("\n📱 Scan QR Code below to login:")
				qrterminal.GenerateHalfBlock(evt.Code, qrterminal.L, os.Stdout)
				fmt.Println()
			case "login":
				slog.Info("Login successful!")
			case "timeout":
				slog.Error("QR code timed out. Please restart the bot.")
				os.Exit(1)
			}
		}
	} else {
		// Session exists, connect directly.
		if err := client.Connect(); err != nil {
			slog.Error("Failed to connect", "error", err)
			os.Exit(1)
		}
		slog.Info("🔄 Reconnected using saved session.")
	}

	// Graceful shutdown on OS signals.
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	slog.Info("🤖 Bot is now running. Press Ctrl+C to stop.")
	<-sigChan

	slog.Info("🛑 Shutting down gracefully...")
	client.Disconnect()
	slog.Info("👋 Bot stopped. Goodbye!")
}

// handleMessage parses and routes incoming messages to the appropriate handler.
func handleMessage(
	client *whatsmeow.Client,
	evt *events.Message,
	registry *handlers.Registry,
	antiStickerHandler *handlers.AntiStickerHandler,
	limiter *ratelimit.Limiter,
) {
	// Anti-sticker check: revoke stickers from banned users BEFORE anything else.
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

		slog.Info("Command executed", "cmd", parsed.Command, "sender", evt.Info.Sender.User, "chat", evt.Info.Chat.String())
		registry.Execute(client, evt, parsed.Command, parsed.Args)
		return
	}

	// Ignore non-command messages from self to prevent infinite loops (e.g. from Auto-Tag)
	if evt.Info.IsFromMe {
		return
	}
}
