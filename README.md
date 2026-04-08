# ChisaBot — WhatsApp Bot (Go + Whatsmeow)

A modular, high-performance WhatsApp bot built with Go and [whatsmeow](https://github.com/tulir/whatsmeow), optimized for low-resource servers.

## Features

| Category             | Commands                                           |
| -------------------- | -------------------------------------------------- |
| **Sticker**          | `.s` — Image/Video/GIF → WebP sticker              |
| **Text Sticker**     | `.ts <text>` — Add meme text to sticker/image      |
| **Brat Sticker**     | `.brat <text>` — Create a brat-style text sticker  |
| **Sticker to Image** | `.toimg` — WebP sticker → PNG / View Once retrieval |
| **Video Downloader** | `.dl <url>` — Download TikTok/IG/YouTube Video     |
| **Audio Downloader** | `.mp3 <url>` — Download YouTube Audio              |
| **Group Admin**      | `.tagall`, `.warn`, `.resetwarn`, `.kick`, `.autotag` |
| **Anti-Sticker**     | `.bansticker`, `.unbansticker`, `.liststicker`     |
| **User Ban**         | `.banuser`, `.unbanuser`, `.listuser`              |
| **Welcome/Goodbye**  | Auto-message on group join/leave                   |
| **Auto-Tag**         | Auto-tag everyone on TikTok link detection         |
| **System**           | `.menu`, `.stat`                                   |

**Prefixes:** `.` `!` `/` (all work interchangeably)

## Prerequisites

1. **Go 1.24+** — https://go.dev/dl/
2. **FFmpeg** — Required for base sticker conversion
3. **ImageMagick** — Required for `.brat` text sticker generation
4. **GCC** — Required for SQLite (CGO)
5. **yt-dlp** — Required for media downloading

### Install FFmpeg & ImageMagick

```bash
# Ubuntu/Debian
sudo apt update && sudo apt install -y ffmpeg imagemagick

# Arch
sudo pacman -S ffmpeg imagemagick

# macOS
brew install ffmpeg imagemagick
```

Verify: `ffmpeg -version` and `magick -version` (or `convert -version`)

### Install GCC (if not present)

```bash
# Ubuntu/Debian
sudo apt install -y build-essential

# Arch
sudo pacman -S base-devel
```

### Install yt-dlp

```bash
# pip (recommended)
pip install -U yt-dlp

# Or download binary
sudo curl -L https://github.com/yt-dlp/yt-dlp/releases/latest/download/yt-dlp -o /usr/local/bin/yt-dlp
sudo chmod a+rx /usr/local/bin/yt-dlp
```

Verify: `yt-dlp --version`

## Quick Start

```bash
# Clone and enter project
cd chisa_bot

# Download dependencies
go mod tidy

# Build
CGO_ENABLED=1 go build -o chisabot ./cmd/bot/

# Run
./chisabot
```

On first run, a **QR code** will be printed in the terminal. Scan it with WhatsApp (Linked Devices → Link a Device).

After linking, the session is saved to `session.db` and subsequent runs reconnect automatically.

## Project Structure

```
chisa_bot/
├── cmd/bot/main.go              # Entry point, QR login, event loop
├── internal/
│   ├── config/
│   │   ├── config.go            # Prefixes, sticker metadata, all constants
│   │   └── messages.go          # Bot message templates
│   ├── router/router.go         # Multi-prefix command parser
│   ├── handlers/
│   │   ├── antisticker.go       # .bansticker, .banuser, etc.
│   │   ├── downloader.go        # .dl, .mp3
│   │   ├── group.go             # Welcome/Goodbye, .tagall, .warn, .kick
│   │   ├── media.go             # .s, .toimg, .ts, .brat
│   │   ├── menu.go              # .menu
│   │   ├── registry.go          # Command routing mapping
│   │   └── system.go            # .stat
│   └── services/
│       ├── autotagstore.go      # Auto-tag preference persistence
│       ├── bannedstickers.go    # Banned sticker hash management
│       ├── bannedstickerusers.go# Banned sticker user management
│       ├── cleanup.go           # Temp files auto-cleaner
│       ├── downloader.go        # yt-dlp wrapper with URL validation
│       ├── ffmpeg.go            # FFmpeg & ImageMagick wrapper
│       └── warnstore.go         # Warning count persistence
├── pkg/
│   ├── ratelimit/ratelimit.go   # Per-user/per-chat rate limiter
│   └── utils/
│       ├── message.go           # Reply helpers, media download
│       └── sticker.go           # WebP Exif metadata writer
├── go.mod
└── go.sum
```

## Architecture

- **Goroutine per command**: Every incoming message is dispatched in its own goroutine to prevent blocking.
- **Panic recovery**: All goroutines have `recover()` wrappers — the bot never crashes.
- **Graceful shutdown**: `Ctrl+C` triggers clean disconnection.
- **Memory limits**: Media downloads are capped at 100MB. Video stickers limited to 8s.
- **Rate limiting**: Per-user cooldown (3s) and per-chat sliding window (10 commands/min).
- **URL validation**: All user-supplied URLs are validated before being passed to external tools.

## Configuration

All configuration is centralized in `internal/config/config.go`:

```go
var Prefixes = []string{".", "!", "/"}

const (
    StickerPackName   = "ChisaBot"
    StickerAuthorName = "chisa_bot"
)

// Rate limiting, media limits, warning thresholds, file paths, etc.
```

## Stopping the Bot

Press `Ctrl+C` for graceful shutdown, or send `SIGTERM`.

## Troubleshooting

| Issue                       | Fix                                             |
| --------------------------- | ----------------------------------------------- |
| `ffmpeg: command not found` | Install ffmpeg (see above)                      |
| `yt-dlp: command not found` | Install yt-dlp (see above)                      |
| `CGO_ENABLED` error         | Install GCC: `sudo apt install build-essential` |
| QR code timeout             | Restart the bot and scan faster                 |
| Session expired             | Delete `session.db` and re-scan QR              |
