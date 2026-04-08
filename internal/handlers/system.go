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

// HandleRecover extracts and forwards a deeply nested quoted message (i.e. to retrieve deleted messages).
func (h *SystemHandler) HandleRecover(client *whatsmeow.Client, evt *events.Message) {
	var bStanzaId string
	if ext := evt.Message.GetExtendedTextMessage(); ext != nil && ext.GetContextInfo() != nil {
		bStanzaId = ext.GetContextInfo().GetStanzaID()
	}

	if bStanzaId == "" {
		utils.ReplyTextDirect(client, evt, "Gagal mendapatkan ID pesan yang di-reply.")
		return
	}

	fullB, err := h.msgCache.Get(bStanzaId)
	if err != nil {
		utils.ReplyTextDirect(client, evt, "Pesan tidak ada di riwayat bot (Mungkin bot sedang mati saat pesan tersebut dikirim, atau sudah kadaluwarsa >24 jam).")
		return
	}

	// Extract the grandparent quoted message (the one that got deleted) from the unstripped fullB
	nestedMsg := utils.GetQuotedMessage(&events.Message{Message: fullB})
	if nestedMsg == nil {
		utils.ReplyTextDirect(client, evt, "Pesan yang di-reply ternyata tidak membalas (reply) pesan orang lain sama sekali.")
		return
	}

	// Case 1: Pure text
	if !utils.IsMediaMessage(nestedMsg) {
		// Wrap it in events.Message to reuse GetTextFromMessage
		fakeEvt := &events.Message{Message: nestedMsg}
		text := utils.GetTextFromMessage(fakeEvt)
		if text == "" {
			utils.ReplyTextDirect(client, evt, "Pesan kosong atau format tidak didukung.")
			return
		}
		utils.ReplyText(client, evt, "[Pesan yang Ditarik/Berlalu]:\n\n"+text)
		return
	}

	// Case 2: Media
	data, err := utils.DownloadMediaFromMessage(client, nestedMsg)
	if err != nil {
		utils.ReplyTextDirect(client, evt, "Gagal mengunduh media dari pesan tersebut (mungkin file asli sudah kedaluwarsa dari server cache WhatsApp).")
		return
	}

	nestedMsg = utils.UnwrapViewOnce(nestedMsg)

	if img := nestedMsg.GetImageMessage(); img != nil {
		caption := img.GetCaption()
		if caption != "" {
			caption = "[Pesan yang Ditarik]:\n\n" + caption
		}
		err = utils.ReplyImage(client, evt, data, img.GetMimetype(), caption)
	} else if vid := nestedMsg.GetVideoMessage(); vid != nil {
		caption := vid.GetCaption()
		if caption != "" {
			caption = "[Pesan yang Ditarik]:\n\n" + caption
		}
		err = utils.ReplyVideo(client, evt, data, vid.GetMimetype(), caption)
	} else if stk := nestedMsg.GetStickerMessage(); stk != nil {
		err = utils.ReplySticker(client, evt, data, stk.GetIsAnimated())
	} else if aud := nestedMsg.GetAudioMessage(); aud != nil {
		err = utils.ReplyAudio(client, evt, data, aud.GetMimetype())
	} else if doc := nestedMsg.GetDocumentMessage(); doc != nil {
		utils.ReplyTextDirect(client, evt, "Fitur pengambilan dokumen belum didukung secara utuh.")
		return
	}

	if err != nil {
		utils.ReplyTextDirect(client, evt, "Gagal memforward ulang media.")
	}
}
