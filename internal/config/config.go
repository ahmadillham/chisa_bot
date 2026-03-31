package config

import (
	"net/url"
	"strings"
)

// Prefixes that the bot will respond to.
var Prefixes = []string{".", "!", "/"}

// Bot metadata for sticker packs.
const (
	StickerPackName   = "ChisaBot"
	StickerAuthorName = "chisa_bot"
)

// Persistence file paths.
const (
	WarnStoreFile              = "warnings.json"
	AutoTagStoreFile           = "autotag.json"
	BannedStickerStoreFile     = "banned_stickers.json"
	BannedStickerUserStoreFile = "banned_sticker_users.json"
)

// Rate limiting defaults.
const (
	RateLimitUserCooldownSec = 3
	RateLimitChatMax         = 10
	RateLimitChatWindowSec   = 60
)

// Media limits.
const (
	MaxFileSizeMB     = 100
	MaxAudioSizeMB    = 50
	MaxVideoStickerSec = 8
)

// Warning system.
const (
	MaxWarningsBeforeKick = 3
)

// FFmpeg font path for text stickers.
const (
	MemeFontPath = "/usr/share/fonts/julietaula-montserrat-fonts/Montserrat-Black.otf"
)

// Rate limit messages.
const (
	MsgRateLimitUser = "⏳ Terlalu cepat, tunggu beberapa detik."
	MsgRateLimitChat = "⏳ Terlalu banyak perintah di chat ini, coba lagi nanti."
)

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
