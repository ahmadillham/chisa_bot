package handlers

import (
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"strings"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types/events"

	"chisa_bot/pkg/utils"
)

// UtilityHandler handles random pick and other utility commands.
type UtilityHandler struct{}

// NewUtilityHandler creates a new UtilityHandler.
func NewUtilityHandler() *UtilityHandler {
	return &UtilityHandler{}
}

// HandlePick randomly picks one option from the given choices.
// Format: .pick opsi1 | opsi2 | opsi3
func (h *UtilityHandler) HandlePick(client *whatsmeow.Client, evt *events.Message, rawArgs string) {
	rawArgs = strings.TrimSpace(rawArgs)
	if rawArgs == "" {
		utils.ReplyText(client, evt, "⚠️ Format: .pick opsi1 | opsi2 | opsi3\nContoh: .pick Makan | Tidur | Ngoding")
		return
	}

	var options []string
	for _, opt := range strings.Split(rawArgs, "|") {
		opt = strings.TrimSpace(opt)
		if opt != "" {
			options = append(options, opt)
		}
	}

	if len(options) < 2 {
		utils.ReplyText(client, evt, "⚠️ Minimal 2 pilihan, pisahkan dengan |\nContoh: .pick Makan | Tidur | Ngoding")
		return
	}

	chosen := options[rand.Intn(len(options))]

	text := fmt.Sprintf(
		"🎲 *Random Pick!*\n\n"+
			"Dari %d pilihan:\n_%s_\n\n"+
			"🎯 Hasilnya: *%s*",
		len(options),
		strings.Join(options, ", "),
		chosen,
	)

	utils.ReplyText(client, evt, text)
}

// HandleShortLink shortens a URL using TinyURL.
func (h *UtilityHandler) HandleShortLink(client *whatsmeow.Client, evt *events.Message, args []string) {
	if len(args) == 0 {
		utils.ReplyText(client, evt, "⚠️ Penggunaan: .short <url>\nContoh: .short https://google.com")
		return
	}

	url := args[0]
	// Basic validation
	if !strings.HasPrefix(url, "http") {
		url = "https://" + url
	}

	utils.ReplyText(client, evt, "⏳ Sedang memendekkan link...")

	// Use TinyURL API (public, no key needed)
	apiURL := fmt.Sprintf("https://tinyurl.com/api-create.php?url=%s", url)
	resp, err := http.Get(apiURL)
	if err != nil {
		log.Printf("[short] request failed: %v", err)
		utils.ReplyText(client, evt, "❌ Gagal menghubungi layanan shortener.")
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		log.Printf("[short] status code: %d", resp.StatusCode)
		utils.ReplyText(client, evt, "❌ Gagal memendekkan link.")
		return
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("[short] read body failed: %v", err)
		utils.ReplyText(client, evt, "❌ Terjadi kesalahan saat membaca respon.")
		return
	}

	shortURL := string(body)
	utils.ReplyText(client, evt, fmt.Sprintf("✅ Link berhasil dipendekkan:\n%s", shortURL))
}
