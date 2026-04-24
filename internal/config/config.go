package config

import (
	"log/slog"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

// Config variables
var (
	Prefixes                 = []string{".", "!", "/"}
	BotDatabaseFile          = "bot.db"
	RateLimitUserCooldownSec = 3
	RateLimitChatMax         = 10
	RateLimitChatWindowSec   = 60
	MaxFileSizeMB            = 100
	MaxAudioSizeMB           = 50
	MaxVideoStickerSec       = 8
	MaxConcurrentMediaTasks  = 4
)

// Bot metadata for sticker packs.
const (
	StickerPackName   = "ChisaBot"
	StickerAuthorName = "chisa_bot"
)

const (
	MemeFontPath = "/usr/share/fonts/TTF/DejaVuSans-Bold.ttf"
)

var MemeFontCandidates = []string{
	"/usr/share/fonts/TTF/DejaVuSans-Bold.ttf",
	"/usr/share/fonts/truetype/dejavu/DejaVuSans-Bold.ttf",
	"/usr/share/fonts/TTF/LiberationSans-Bold.ttf",
	"/System/Library/Fonts/Supplemental/Arial Bold.ttf",
	"C:\\Windows\\Fonts\\arialbd.ttf",
}

// Rate limit and system messages.
const (
	MsgQueueLimit    = "Permintaan sedang diproses, mohon tunggu antrean..."
	MsgRateLimitUser = "Terlalu cepat, tunggu beberapa detik."
	MsgRateLimitChat = "Terlalu banyak perintah di chat ini, coba lagi nanti."
)

// Load reads configuration from .env and environment variables.
func Load() {
	if err := godotenv.Load(); err != nil {
		slog.Info("No .env file found or error loading, using default/env values", "error", err)
	}

	if p := os.Getenv("PREFIXES"); p != "" {
		Prefixes = strings.Split(p, ",")
	}
	if v := os.Getenv("BOT_DATABASE_FILE"); v != "" {
		BotDatabaseFile = v
	}
	if v := os.Getenv("RATE_LIMIT_USER_COOLDOWN_SEC"); v != "" {
		if val, err := strconv.Atoi(v); err == nil {
			RateLimitUserCooldownSec = val
		}
	}
	if v := os.Getenv("RATE_LIMIT_CHAT_MAX"); v != "" {
		if val, err := strconv.Atoi(v); err == nil {
			RateLimitChatMax = val
		}
	}
	if v := os.Getenv("RATE_LIMIT_CHAT_WINDOW_SEC"); v != "" {
		if val, err := strconv.Atoi(v); err == nil {
			RateLimitChatWindowSec = val
		}
	}
	if v := os.Getenv("MAX_FILE_SIZE_MB"); v != "" {
		if val, err := strconv.Atoi(v); err == nil {
			MaxFileSizeMB = val
		}
	}
	if v := os.Getenv("MAX_AUDIO_SIZE_MB"); v != "" {
		if val, err := strconv.Atoi(v); err == nil {
			MaxAudioSizeMB = val
		}
	}
	if v := os.Getenv("MAX_VIDEO_STICKER_SEC"); v != "" {
		if val, err := strconv.Atoi(v); err == nil {
			MaxVideoStickerSec = val
		}
	}

	if v := os.Getenv("MAX_CONCURRENT_MEDIA_TASKS"); v != "" {
		if val, err := strconv.Atoi(v); err == nil {
			MaxConcurrentMediaTasks = val
		}
	}
}

// ValidateURL checks that a URL is safe to pass to external tools.
// Returns true if valid, false otherwise.
func ValidateURL(rawURL string) bool {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		return false
	}

	// Must start with http:// or https://
	if !strings.HasPrefix(rawURL, "http://") && !strings.HasPrefix(rawURL, "https://") {
		return false
	}

	// Reject flag injection (strings starting with -)
	if strings.Contains(rawURL, " ") {
		return false
	}

	// Must parse as valid URL
	u, err := url.Parse(rawURL)
	if err != nil {
		return false
	}

	// Must have a host
	if u.Host == "" {
		return false
	}

	return true
}
