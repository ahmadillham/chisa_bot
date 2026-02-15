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
	menu := `📋 *Daftar Perintah*
Prefix: . ! /

• .sticker (.s)
• .toimg
• .showimg (.rv)
• .dl <link>
• .mp3 <link>
• .tagall
• .kick <member>
• .stats
• .menu`

	utils.ReplyText(client, evt, menu)
}
