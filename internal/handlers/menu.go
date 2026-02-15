package handlers

import (
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types/events"

	"chisa_bot/pkg/utils"
)

// MenuHandler handles the menu command.
type MenuHandler struct{}

// NewMenuHandler creates a new MenuHandler.
func NewMenuHandler() *MenuHandler {
	return &MenuHandler{}
}

// HandleMenu sends a list of all available commands.
func (h *MenuHandler) HandleMenu(client *whatsmeow.Client, evt *events.Message) {
	menu := `╔══════════════════════╗
║    🤖 *CHISA BOT*    ║
╚══════════════════════╝

📋 *Daftar Perintah*
Prefix: . ! /

• .sticker (.s)
  _Ubah gambar/video/GIF jadi sticker_
• .toimg
  _Ubah sticker jadi gambar_
• .showimg (.rv)
  _Ambil media View Once (Reply pesan)_
• .dl <link>
  _Download IG, TikTok, FB, YouTube_
• .mp3 <link>
  _Download Audio (YouTube/TikTok)_
• .tagall
  _Mention semua anggota (Admin only)_
• .kick <member>
  _Kick member (Admin only)_
• .stats
  _Status server bot_
• .menu
  _Tampilkan pesan ini_`

	utils.ReplyText(client, evt, menu)
}
