package handlers

import (
	"fmt"
	"os"
	"runtime"
	"strings"
	"time"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types/events"

	"chisa_bot/internal/services"
	"chisa_bot/pkg/utils"

	waProto "go.mau.fi/whatsmeow/binary/proto"
)

// SystemHandler handles system information commands.
type SystemHandler struct {
	startTime time.Time
	msgCache  *services.MessageCacheStore
}

// NewSystemHandler creates a new SystemHandler.
func NewSystemHandler(msgCache *services.MessageCacheStore) *SystemHandler {
	return &SystemHandler{
		startTime: time.Now(),
		msgCache:  msgCache,
	}
}

// HandleStats sends server stats (CPU, RAM, Uptime).
func (h *SystemHandler) HandleStats(client *whatsmeow.Client, evt *events.Message) {
	// CPU info
	cpuModel := getCPUModel()
	cpuCores := runtime.NumCPU()

	// RAM info
	totalRAM, usedRAM, _ := getMemoryInfo()

	// System uptime
	sysUptime := getSystemUptime()
	distro := getDistroName()

	// Go runtime stats
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)
	botMemMB := float64(memStats.Alloc) / 1024 / 1024

	stats := fmt.Sprintf(`%s
%s (%d Cores)
%s / %s
%s
%.2f MB`,
		distro,
		cpuModel, cpuCores,
		usedRAM, totalRAM,
		sysUptime,
		botMemMB,
	)

	utils.ReplyText(client, evt, stats)
}

func formatDuration(d time.Duration) string {
	days := int(d.Hours()) / 24
	hours := int(d.Hours()) % 24
	minutes := int(d.Minutes()) % 60
	seconds := int(d.Seconds()) % 60

	if days > 0 {
		return fmt.Sprintf("%dd %dh %dm %ds", days, hours, minutes, seconds)
	}
	if hours > 0 {
		return fmt.Sprintf("%dh %dm %ds", hours, minutes, seconds)
	}
	if minutes > 0 {
		return fmt.Sprintf("%dm %ds", minutes, seconds)
	}
	return fmt.Sprintf("%ds", seconds)
}

func getDistroName() string {
	data, err := os.ReadFile("/etc/os-release")
	if err != nil {
		return "Unknown"
	}
	for _, line := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(line, "PRETTY_NAME=") {
			name := strings.TrimPrefix(line, "PRETTY_NAME=")
			return strings.Trim(name, "\"")
		}
	}
	return "Unknown"
}

func getCPUModel() string {
	data, err := os.ReadFile("/proc/cpuinfo")
	if err != nil {
		return "Unknown"
	}
	for _, line := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(line, "model name") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				return strings.TrimSpace(parts[1])
			}
		}
	}
	return "Unknown"
}

func getMemoryInfo() (total, used, free string) {
	data, err := os.ReadFile("/proc/meminfo")
	if err != nil {
		return "N/A", "N/A", "N/A"
	}

	var totalKB, availKB uint64
	for _, line := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(line, "MemTotal:") {
			fmt.Sscanf(line, "MemTotal: %d kB", &totalKB)
		} else if strings.HasPrefix(line, "MemAvailable:") {
			fmt.Sscanf(line, "MemAvailable: %d kB", &availKB)
		}
	}

	usedKB := totalKB - availKB
	return formatBytes(totalKB * 1024), formatBytes(usedKB * 1024), formatBytes(availKB * 1024)
}

func getSystemUptime() string {
	data, err := os.ReadFile("/proc/uptime")
	if err != nil {
		return "N/A"
	}

	var seconds float64
	fmt.Sscanf(string(data), "%f", &seconds)
	return formatDuration(time.Duration(seconds) * time.Second)
}

func formatBytes(bytes uint64) string {
	const (
		MB = 1024 * 1024
		GB = 1024 * MB
	)
	if bytes >= GB {
		return fmt.Sprintf("%.2f GB", float64(bytes)/float64(GB))
	}
	return fmt.Sprintf("%.2f MB", float64(bytes)/float64(MB))
}

// HandleRecover extracts and forwards a view-once or deleted message.
func (h *SystemHandler) HandleRecover(client *whatsmeow.Client, evt *events.Message) {
	ctxInfo := evt.Message.GetExtendedTextMessage().GetContextInfo()
	if ctxInfo == nil {
		utils.ReplyTextDirect(client, evt, "Gagal mendapatkan ID pesan yang di-reply.")
		return
	}

	bStanzaId := ctxInfo.GetStanzaID()
	quoted := ctxInfo.GetQuotedMessage()
	var targetMsg *waProto.Message

	// Fallback mechanism:
	// 1. Check if the message is in the cache (gives us the raw original message)
	cachedB, err := h.msgCache.Get(bStanzaId)

	// Determine what the user wants to read
	if quoted != nil && utils.IsViewOnceMessage(quoted) {
		// User is replying directly to a View Once message
		targetMsg = quoted
		if err == nil {
			targetMsg = cachedB // Prefer cached for completeness, though quoted is usually enough
		}
	} else if err == nil {
		// If B is in cache, see if it's a text reply to a deleted message A
		if nestedCtx := cachedB.GetExtendedTextMessage().GetContextInfo(); nestedCtx != nil {
			aStanzaId := nestedCtx.GetStanzaID()
			if aStanzaId != "" {
				cachedA, errA := h.msgCache.Get(aStanzaId)
				if errA == nil {
					targetMsg = cachedA // Target is the grandparent (deleted message A)
				}
			}
		}
		// If targetMsg is still nil, maybe user wants to recover B itself (if B was deleted)
		if targetMsg == nil {
			targetMsg = cachedB
		}
	} else {
		// Not a View Once, and not in cache. But we have quoted message.
		targetMsg = quoted
	}

	if targetMsg == nil {
		utils.ReplyTextDirect(client, evt, "Pesan tidak ada di riwayat bot (Mungkin bot sedang mati saat pesan tersebut dikirim, atau sudah kadaluwarsa).")
		return
	}

	// Case 1: Pure text
	if !utils.IsMediaMessage(targetMsg) {
		text := utils.GetTextFromMessage(&events.Message{Message: targetMsg})
		if text == "" {
			utils.ReplyTextDirect(client, evt, "Pesan kosong atau format tidak didukung.")
			return
		}
		utils.ReplyText(client, evt, text)
		return
	}

	// Case 2: Media
	data, err := utils.DownloadMediaFromMessage(client, targetMsg)
	if err != nil {
		utils.ReplyTextDirect(client, evt, "Gagal mengunduh media dari pesan tersebut (mungkin file asli sudah kedaluwarsa dari server WhatsApp).")
		return
	}

	targetMsg = utils.UnwrapViewOnce(targetMsg)

	if img := targetMsg.GetImageMessage(); img != nil {
		err = utils.ReplyImage(client, evt, data, img.GetMimetype(), img.GetCaption())
	} else if vid := targetMsg.GetVideoMessage(); vid != nil {
		err = utils.ReplyVideo(client, evt, data, vid.GetMimetype(), vid.GetCaption())
	} else if stk := targetMsg.GetStickerMessage(); stk != nil {
		err = utils.ReplySticker(client, evt, data, stk.GetIsAnimated())
	} else if aud := targetMsg.GetAudioMessage(); aud != nil {
		err = utils.ReplyAudio(client, evt, data, aud.GetMimetype())
	} else {
		utils.ReplyTextDirect(client, evt, "Format media belum didukung secara utuh.")
		return
	}

	if err != nil {
		utils.ReplyTextDirect(client, evt, "Gagal memforward ulang media.")
	}
}
