# ChisaBot — WhatsApp Bot (Go + Whatsmeow)

A modular, high-performance WhatsApp bot built with Go and [whatsmeow](https://github.com/tulir/whatsmeow), optimized for low-resource servers.

## Features

| Category             | Commands                                           |
| -------------------- | -------------------------------------------------- |
| **Sticker**          | `.sticker` / `.s` — Image/Video/GIF → WebP sticker |
| **Sticker to Image** | `.toimg` — WebP sticker → PNG image                |
| **TikTok**           | `.tiktok <url>` — Download without watermark       |
| **Instagram**        | `.ig <url>` — Download Reels/Posts                 |
| **YouTube**          | `.ytmp3 <url>` / `.ytmp4 <url>` — Audio or Video   |
| **Tag All**          | `.tagall` — Mention all group members (admin only) |
| **Welcome/Goodbye**  | Auto-message on group join/leave                   |
| **Kerang Ajaib**     | `.kerangajaib <question>` — Magic Conch Shell      |
| **Cek Khodam**       | `.cekkhodam <name>` — Random spirit generator      |
| **Cek Jodoh**        | `.cekjodoh <name1> <name2>` — Compatibility check  |

**Prefixes:** `.` `!` `/` (all work interchangeably)

## Prerequisites

1. **Go 1.24+** — https://go.dev/dl/
2. **FFmpeg** — Required for sticker conversion
3. **GCC** — Required for SQLite (CGO)

### Install FFmpeg

```bash
# Ubuntu/Debian
sudo apt update && sudo apt install -y ffmpeg

# Arch
sudo pacman -S ffmpeg

# macOS
brew install ffmpeg
```

Verify: `ffmpeg -version`

### Install GCC (if not present)

```bash
# Ubuntu/Debian
sudo apt install -y build-essential

# Arch
sudo pacman -S base-devel
```

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
│   ├── config/config.go         # Prefixes, sticker pack metadata
│   ├── router/router.go         # Multi-prefix command parser
│   ├── handlers/
│   │   ├── media.go             # .sticker, .toimg
│   │   ├── downloader.go        # .tiktok, .ig, .ytmp3, .ytmp4
│   │   ├── group.go             # Welcome/Goodbye, .tagall
│   │   └── fun.go               # .kerangajaib, .cekkhodam, .cekjodoh
│   └── services/
│       ├── ffmpeg.go            # FFmpeg conversion wrapper
│       └── downloader.go        # Downloader API implementations
├── pkg/utils/
│   ├── message.go               # Reply helpers, media download
│   └── sticker.go               # WebP Exif metadata writer
├── go.mod
└── go.sum
```

## Architecture

- **Goroutine per command**: Every incoming message is dispatched in its own goroutine to prevent blocking.
- **Panic recovery**: All goroutines have `recover()` wrappers — the bot never crashes.
- **Graceful shutdown**: `Ctrl+C` triggers clean disconnection.
- **Memory limits**: Media downloads are capped at 100MB. Video stickers limited to 8s.

## Configuration

Edit `internal/config/config.go` to customize:

```go
var Prefixes = []string{".", "!", "/"}

const (
    StickerPackName   = "ChisaBot"
    StickerAuthorName = "chisa_bot"
)
```

## Stopping the Bot

Press `Ctrl+C` for graceful shutdown, or send `SIGTERM`.

## Troubleshooting

| Issue                       | Fix                                             |
| --------------------------- | ----------------------------------------------- |
| `ffmpeg: command not found` | Install ffmpeg (see above)                      |
| `CGO_ENABLED` error         | Install GCC: `sudo apt install build-essential` |
| QR code timeout             | Restart the bot and scan faster                 |
| Session expired             | Delete `session.db` and re-scan QR              |
