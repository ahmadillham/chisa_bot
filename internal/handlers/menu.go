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

━━━ 🖼️ *Media* ━━━
• .sticker (.s)
  _Ubah gambar/video/GIF jadi sticker_
• .toimg
  _Ubah sticker jadi gambar_
• .showimg (.rv)
  _Ambil media View Once (Reply pesan)_

━━━ 📥 *Downloader* ━━━
• .dl <link>
  _Download IG, TikTok, FB, YouTube_
• .mp3 <link>
  _Download Audio (YouTube/TikTok)_

━━━ 👥 *Grup* ━━━
• .tagall
  _Mention semua anggota (Admin only)_
• .kick <member>
  _Kick member (Admin only)_

━━━ 🛠️ *Lainnya* ━━━
• .short <link>
  _Pendekkan link (TinyURL)_
• .pick <opsi1> | <opsi2>
  _Pilih opsi random_
• .stats
  _Status server bot_
• .menu
  _Tampilkan pesan ini_`

	utils.ReplyText(client, evt, menu)
}
