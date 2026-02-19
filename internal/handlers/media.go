package handlers

import (
	"fmt"
	"log"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types/events"

	"chisa_bot/internal/config"
	"chisa_bot/internal/services"
	"chisa_bot/pkg/utils"
)

// MediaHandler handles sticker creation and conversion commands.
type MediaHandler struct {
	ffmpeg *services.FFmpegService
}

// NewMediaHandler creates a new MediaHandler.
func NewMediaHandler() *MediaHandler {
	return &MediaHandler{
		ffmpeg: services.NewFFmpegService(),
	}
}

// HandleSticker converts an image/video/GIF to a WebP sticker.
func (h *MediaHandler) HandleSticker(client *whatsmeow.Client, evt *events.Message) {
	// Try to get media from the message itself (image/video with caption)
	// or from a quoted message.
	var mediaMsg = evt.Message

	// Check if the message itself has media.
	if !utils.IsMediaMessage(evt.Message) {
		// Check quoted message.
		quoted := utils.GetQuotedMessage(evt)
		if quoted == nil || !utils.IsMediaMessage(quoted) {
			if err := utils.ReplyText(client, evt, "⚠️ Kirim atau reply gambar/video/GIF dengan caption .sticker atau .s"); err != nil {
				log.Printf("[sticker] failed to reply: %v", err)
			}
			return
		}
		mediaMsg = quoted
	}

	// Download the media.
	data, err := utils.DownloadMediaFromMessage(client, mediaMsg)
	if err != nil {
		log.Printf("[sticker] failed to download media: %v", err)
		utils.ReplyText(client, evt, "❌ Gagal download media.")
		return
	}

	var webpData []byte
	isAnimated := false

	// Unwrap view once just in case (utility handles it, but we need type).
	mediaMsg = utils.UnwrapViewOnce(mediaMsg)

	// Determine media type and convert accordingly.
	if mediaMsg.GetImageMessage() != nil {
		webpData, err = h.ffmpeg.ImageToWebP(data)
	} else if mediaMsg.GetVideoMessage() != nil {
		ext := ".mp4"
		if mediaMsg.GetVideoMessage().GetGifPlayback() {
			ext = ".gif"
		}
		webpData, err = h.ffmpeg.VideoToWebP(data, ext)
		isAnimated = true
	} else {
		err = fmt.Errorf("unsupported media type for sticker")
	}

	if err != nil {
		log.Printf("[sticker] conversion failed: %v", err)
		utils.ReplyText(client, evt, "❌ Gagal convert ke sticker.")
		return
	}

	// Add Exif metadata (pack name & author).
	webpData, err = utils.AddStickerExif(webpData, config.StickerPackName, config.StickerAuthorName)
	if err != nil {
		log.Printf("[sticker] exif injection failed: %v", err)
		// Send without exif, it's not critical.
	}

	// Send the sticker.
	if err := utils.ReplySticker(client, evt, webpData, isAnimated); err != nil {
		log.Printf("[sticker] failed to send sticker: %v", err)
		utils.ReplyText(client, evt, "❌ Gagal mengirim sticker.")
	}
}

// HandleStickerToImage converts a sticker back to a PNG image.
func (h *MediaHandler) HandleStickerToImage(client *whatsmeow.Client, evt *events.Message) {
	// Get the sticker from a quoted message.
	quoted := utils.GetQuotedMessage(evt)
	if quoted == nil || quoted.GetStickerMessage() == nil {
		utils.ReplyText(client, evt, "⚠️ Reply sticker dengan caption .toimg")
		return
	}

	// Download the sticker.
	data, err := utils.DownloadMediaFromMessage(client, quoted)
	if err != nil {
		log.Printf("[toimg] failed to download sticker: %v", err)
		utils.ReplyText(client, evt, "❌ Gagal download sticker.")
		return
	}

	// Convert WebP to PNG.
	pngData, err := h.ffmpeg.WebPToImage(data)
	if err != nil {
		log.Printf("[toimg] conversion failed: %v", err)
		utils.ReplyText(client, evt, "❌ Gagal convert sticker ke gambar.")
		return
	}

	// Send the image.
	if err := utils.ReplyImage(client, evt, pngData, "image/png", ""); err != nil {
		log.Printf("[toimg] failed to send image: %v", err)
		utils.ReplyText(client, evt, "❌ Gagal mengirim gambar.")
	}
}

// HandleRetrieveViewOnce resends a view once message as a normal message.
func (h *MediaHandler) HandleRetrieveViewOnce(client *whatsmeow.Client, evt *events.Message) {
	// Get quoted message.
	quoted := utils.GetQuotedMessage(evt)
	if quoted == nil || !utils.IsMediaMessage(quoted) {
		utils.ReplyText(client, evt, "⚠️ Reply pesan View Once (sekali lihat) dengan caption .showimg")
		return
	}

	// Download media.
	data, err := utils.DownloadMediaFromMessage(client, quoted)
	if err != nil {
		log.Printf("[save] failed to download media: %v", err)
		utils.ReplyText(client, evt, "❌ Gagal download media.")
		return
	}

	// Unwrap to check type (utils handles download, but we need type to send).
	msg := utils.UnwrapViewOnce(quoted)

	// Resend as normal message.
	if img := msg.GetImageMessage(); img != nil {
		err = utils.ReplyImage(client, evt, data, img.GetMimetype(), img.GetCaption())
	} else if vid := msg.GetVideoMessage(); vid != nil {
		err = utils.ReplyVideo(client, evt, data, vid.GetMimetype(), vid.GetCaption())
	} else {
		// Should verify if audio works too properly, but View Once is mainly img/vid.
		err = fmt.Errorf("unsupported view once type")
	}

	if err != nil {
		log.Printf("[save] failed to send media: %v", err)
		utils.ReplyText(client, evt, "❌ Gagal mengirim ulang media.")
	}
}

// HandleImage is a smart command that handles both sticker-to-image and view-once-retrieval.
func (h *MediaHandler) HandleImage(client *whatsmeow.Client, evt *events.Message) {
	quoted := utils.GetQuotedMessage(evt)
	if quoted == nil {
		utils.ReplyText(client, evt, "⚠️ Reply sticker atau pesan View Once dengan caption .toimg")
		return
	}

	// Case 1: Sticker -> Image
	if quoted.GetStickerMessage() != nil {
		h.HandleStickerToImage(client, evt)
		return
	}

	// Case 2: View Once -> Image/Video (supports V1, V2, V2Extension)
	if utils.IsViewOnceMessage(quoted) {
		h.HandleRetrieveViewOnce(client, evt)
		return
	}

	utils.ReplyText(client, evt, "⚠️ Pesan yang di-reply bukan sticker atau View Once.")
}
