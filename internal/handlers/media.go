package handlers

import (
	"fmt"
	"log/slog"
	"strings"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types/events"

	"chisa_bot/internal/config"
	"chisa_bot/internal/services"
	"chisa_bot/pkg/utils"
)

// MediaHandler handles sticker creation and conversion commands.
type MediaHandler struct {
	ffmpeg *services.FFmpegService
	pool   *services.WorkerPool
}

// NewMediaHandler creates a new MediaHandler.
func NewMediaHandler(pool *services.WorkerPool) *MediaHandler {
	return &MediaHandler{
		ffmpeg: services.NewFFmpegService(),
		pool:   pool,
	}
}

// HandleSticker converts an image/video/GIF to a WebP sticker.
func (h *MediaHandler) HandleSticker(client *whatsmeow.Client, evt *events.Message) {
	h.pool.Acquire()
	defer h.pool.Release()

	// Try to get media from the message itself (image/video with caption)
	// or from a quoted message.
	var mediaMsg = evt.Message

	// Check if the message itself has media.
	if !utils.IsMediaMessage(evt.Message) {
		// Check quoted message.
		quoted := utils.GetQuotedMessage(evt)
		if quoted == nil || !utils.IsMediaMessage(quoted) {
			if err := utils.ReplyTextDirect(client, evt, "Kirim atau reply gambar/video/GIF dengan caption .sticker atau .s"); err != nil {
				slog.Error("failed to reply", "error", err)
			}
			return
		}
		mediaMsg = quoted
	}

	// Download the media.
	data, err := utils.DownloadMediaFromMessage(client, mediaMsg)
	if err != nil {
		slog.Error("failed to download media", "error", err)
		utils.ReplyTextDirect(client, evt, "Gagal download media.")
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
	} else if mediaMsg.GetDocumentMessage() != nil {
		mimetype := mediaMsg.GetDocumentMessage().GetMimetype()
		if strings.HasPrefix(mimetype, "video/") || strings.HasSuffix(mimetype, "gif") {
			webpData, err = h.ffmpeg.VideoToWebP(data, ".mp4")
			isAnimated = true
		} else {
			webpData, err = h.ffmpeg.ImageToWebP(data)
		}
	} else if mediaMsg.GetStickerMessage() != nil {
		// User is trying to re-sticker a sticker. We can just re-send it with new EXIF.
		webpData = data
		isAnimated = mediaMsg.GetStickerMessage().GetIsAnimated()
	} else {
		err = fmt.Errorf("unsupported media type for sticker")
	}

	if err != nil {
		slog.Error("conversion failed", "error", err)
		utils.ReplyTextDirect(client, evt, fmt.Sprintf("Gagal convert ke sticker: %v", err))
		return
	}

	// Add Exif metadata (pack name & author).
	webpData, err = utils.AddStickerExif(webpData, config.StickerPackName, config.StickerAuthorName)
	if err != nil {
		slog.Error("exif injection failed", "error", err)
		// Send without exif, it's not critical.
	}

	// Send the sticker.
	if err := utils.ReplySticker(client, evt, webpData, isAnimated); err != nil {
		slog.Error("failed to send sticker", "error", err)
		utils.ReplyTextDirect(client, evt, "Gagal mengirim sticker.")
	}
}

// HandleStickerToImage converts a sticker back to a PNG image.
func (h *MediaHandler) HandleStickerToImage(client *whatsmeow.Client, evt *events.Message) {
	h.pool.Acquire()
	defer h.pool.Release()

	// Get the sticker from a quoted message.
	quoted := utils.GetQuotedMessage(evt)
	if quoted == nil || quoted.GetStickerMessage() == nil {
		utils.ReplyTextDirect(client, evt, "Reply sticker dengan caption .toimg")
		return
	}

	// Download the sticker.
	data, err := utils.DownloadMediaFromMessage(client, quoted)
	if err != nil {
		slog.Error("failed to download sticker", "error", err)
		utils.ReplyTextDirect(client, evt, "Gagal download sticker.")
		return
	}

	// Convert WebP to PNG.
	pngData, err := h.ffmpeg.WebPToImage(data)
	if err != nil {
		slog.Error("conversion failed", "error", err)
		utils.ReplyTextDirect(client, evt, "Gagal convert sticker ke gambar.")
		return
	}

	// Send the image.
	if err := utils.ReplyImage(client, evt, pngData, "image/png", ""); err != nil {
		slog.Error("failed to send image", "error", err)
		utils.ReplyTextDirect(client, evt, "Gagal mengirim gambar.")
	}
}

// HandleRetrieveViewOnce resends a view once message as a normal message.
func (h *MediaHandler) HandleRetrieveViewOnce(client *whatsmeow.Client, evt *events.Message) {
	h.pool.Acquire()
	defer h.pool.Release()

	// Get quoted message.
	quoted := utils.GetQuotedMessage(evt)
	if quoted == nil || !utils.IsMediaMessage(quoted) {
		utils.ReplyTextDirect(client, evt, "Reply pesan View Once (sekali lihat) dengan caption .showimg")
		return
	}

	// Download media.
	data, err := utils.DownloadMediaFromMessage(client, quoted)
	if err != nil {
		slog.Error("failed to download media", "error", err)
		utils.ReplyTextDirect(client, evt, "Gagal download media.")
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
		slog.Error("failed to send media", "error", err)
		utils.ReplyTextDirect(client, evt, "Gagal mengirim ulang media.")
	}
}

// HandleImage is a smart command that handles both sticker-to-image and view-once-retrieval.
func (h *MediaHandler) HandleImage(client *whatsmeow.Client, evt *events.Message) {
	quoted := utils.GetQuotedMessage(evt)
	if quoted == nil {
		utils.ReplyTextDirect(client, evt, "Reply sticker atau pesan View Once dengan caption .toimg")
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

	utils.ReplyTextDirect(client, evt, "Pesan yang di-reply bukan sticker atau View Once.")
}

// HandleTextSticker adds meme-style text to a sticker or image (command: .ts <text>).
// Usage: send/reply sticker or image with .ts TEKS
func (h *MediaHandler) HandleTextSticker(client *whatsmeow.Client, evt *events.Message, args []string) {
	h.pool.Acquire()
	defer h.pool.Release()

	if len(args) == 0 {
		utils.ReplyTextDirect(client, evt, "Penggunaan: kirim/reply gambar atau sticker dengan .ts <teks>\nContoh: .ts MENGANCAM")
		return
	}

	text := strings.Join(args, " ")
	if len(text) > 70 {
		utils.ReplyTextDirect(client, evt, "Teks terlalu panjang! Maksimal 70 karakter.")
		return
	}
	// Sanitize against ImageMagick injection vectors like `@file` or `-format`
	if strings.HasPrefix(text, "@") || strings.HasPrefix(text, "-") {
		text = " " + text
	}

	// Find the media source: current message or quoted message.
	var targetMsg = evt.Message
	if quoted := utils.GetQuotedMessage(evt); quoted != nil {
		targetMsg = quoted
	}

	// Accept sticker, image, video (GIF), or document.
	stk := targetMsg.GetStickerMessage()
	img := targetMsg.GetImageMessage()
	vid := targetMsg.GetVideoMessage()
	doc := targetMsg.GetDocumentMessage()

	if stk == nil && img == nil && vid == nil && doc == nil {
		utils.ReplyTextDirect(client, evt, "Kirim atau reply gambar/video/sticker dengan caption .ts <teks>")
		return
	}

	// Download the media.
	data, err := utils.DownloadMediaFromMessage(client, targetMsg)
	if err != nil {
		slog.Error("failed to download", "error", err)
		utils.ReplyTextDirect(client, evt, "Gagal download media.")
		return
	}

	// Identify input properties.
	var ext string
	isAnimated := false

	if stk != nil {
		ext = ".webp"
		isAnimated = stk.GetIsAnimated()
	} else if img != nil {
		ext = ".jpg" // default for images
		isAnimated = false
	} else if vid != nil {
		ext = ".mp4"
		if vid.GetGifPlayback() {
			ext = ".gif"
		}
		isAnimated = true
	} else if doc != nil {
		mimetype := doc.GetMimetype()
		if strings.HasPrefix(mimetype, "video/") || strings.HasSuffix(mimetype, "gif") {
			ext = ".mp4"
			isAnimated = true
		} else {
			ext = ".jpg"
			isAnimated = false
		}
	}

	// Overlay the text.
	webpData, err := h.ffmpeg.AddTextToWebP(data, text, ext, isAnimated)
	if err != nil {
		slog.Error("failed to add text", "error", err)
		errMsg := "Gagal menambahkan teks ke sticker."
		if strings.Contains(err.Error(), "ffmpeg") {
			errMsg += " Pastikan FFmpeg terinstal di server."
		}
		utils.ReplyTextDirect(client, evt, errMsg)
		return
	}

	// Add Exif metadata.
	webpData, _ = utils.AddStickerExif(webpData, config.StickerPackName, config.StickerAuthorName)

	// Send as sticker.
	if err := utils.ReplySticker(client, evt, webpData, isAnimated); err != nil {
		slog.Error("failed to send sticker", "error", err)
		utils.ReplyTextDirect(client, evt, "Gagal mengirim sticker.")
	}
}

// HandleBrat creates a 'brat' style sticker from text.
func (h *MediaHandler) HandleBrat(client *whatsmeow.Client, evt *events.Message, args []string) {
	h.pool.Acquire()
	defer h.pool.Release()

	if len(args) == 0 {
		utils.ReplyTextDirect(client, evt, "Penggunaan: .brat <teks>")
		return
	}

	text := strings.Join(args, " ")
	if len(text) > 50 {
		utils.ReplyTextDirect(client, evt, "Teks terlalu panjang! Maksimal 50 karakter.")
		return
	}
	// Sanitize against ImageMagick injection vectors
	if strings.HasPrefix(text, "@") || strings.HasPrefix(text, "-") {
		text = " " + text
	}

	// Generate brat sticker image data.
	webpData, err := h.ffmpeg.GenerateBratSticker(text)
	if err != nil {
		slog.Error("failed to generate", "error", err)
		utils.ReplyTextDirect(client, evt, "Gagal membuat sticker brat. Pastikan ImageMagick (magick/convert) terinstal.")
		return
	}

	// Inject Exif metadata constraints for WhatsApp to detect it as a valid sticker.
	webpData, _ = utils.AddStickerExif(webpData, config.StickerPackName, config.StickerAuthorName)

	// Send sticker.
	if err := utils.ReplySticker(client, evt, webpData, false); err != nil {
		slog.Error("failed to send sticker", "error", err)
		utils.ReplyTextDirect(client, evt, "Gagal mengirim sticker brat.")
	}
}
