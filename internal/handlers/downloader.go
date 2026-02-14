package handlers

import (
	"log"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types/events"

	"chisa_bot/internal/services"
	"chisa_bot/pkg/utils"
)

// DownloaderHandler handles media download commands.
type DownloaderHandler struct {
	ytdlp *services.YtDlpService
}

// NewDownloaderHandler creates a new DownloaderHandler.
func NewDownloaderHandler() *DownloaderHandler {
	return &DownloaderHandler{
		ytdlp: services.NewYtDlpService(),
	}
}

// HandleVideo downloads video from any supported platform (IG, TikTok, FB, YouTube, etc).
func (h *DownloaderHandler) HandleVideo(client *whatsmeow.Client, evt *events.Message, args []string) {
	if len(args) == 0 {
		utils.ReplyText(client, evt, "⚠️ Penggunaan: .dl <url>\nSupport: IG, TikTok, FB, YouTube, Twitter, dll.")
		return
	}

	url := args[0]
	utils.ReplyText(client, evt, "⏳ Sedang memproses media...")

	// Use the smart "DownloadAny" service.
	result, err := h.ytdlp.DownloadAny(url)
	if err != nil {
		log.Printf("[dl] download failed: %v", err)
		utils.ReplyText(client, evt, "❌ Gagal mendownload media. Pastikan link publik dan valid.")
		return
	}

	caption := result.Title
	if len(caption) > 200 {
		caption = caption[:197] + "..."
	}

	// Determine if it's video or image
	if result.Type == "image" {
		if err := utils.ReplyImage(client, evt, result.Data, result.Mimetype, caption); err != nil {
			log.Printf("[dl] failed to send image: %v", err)
			utils.ReplyText(client, evt, "❌ Gagal mengirim gambar ke WhatsApp.")
		}
	} else {
		// Default to video
		if err := utils.ReplyVideo(client, evt, result.Data, result.Mimetype, caption); err != nil {
			log.Printf("[dl] failed to send video: %v", err)
			utils.ReplyText(client, evt, "❌ Gagal mengirim media ke WhatsApp (mungkin file terlalu besar).")
		}
	}
}

// HandleAudio downloads audio (MP3) from YouTube/TikTok/etc.
func (h *DownloaderHandler) HandleAudio(client *whatsmeow.Client, evt *events.Message, args []string) {
	if len(args) == 0 {
		utils.ReplyText(client, evt, "⚠️ Penggunaan: .mp3 <url>")
		return
	}

	url := args[0]
	utils.ReplyText(client, evt, "⏳ Sedang mengambil audio...")

	result, err := h.ytdlp.DownloadAudio(url)
	if err != nil {
		log.Printf("[mp3] download failed: %v", err)
		utils.ReplyText(client, evt, "❌ Gagal mendownload audio.")
		return
	}

	if err := utils.ReplyAudio(client, evt, result.Data, result.Mimetype); err != nil {
		log.Printf("[mp3] failed to send audio: %v", err)
		utils.ReplyText(client, evt, "❌ Gagal mengirim audio.")
	}
}
