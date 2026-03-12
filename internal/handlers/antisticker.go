package handlers

import (
	"context"
	"encoding/hex"
	"fmt"
	"log"
	"strings"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types/events"

	"chisa_bot/internal/config"
	"chisa_bot/internal/services"
	"chisa_bot/pkg/utils"
)

// AntiStickerHandler handles auto-deletion of banned stickers.
type AntiStickerHandler struct {
	store        *services.BannedStickerStore
	groupHandler *GroupHandler
}

// NewAntiStickerHandler creates a new AntiStickerHandler.
func NewAntiStickerHandler(store *services.BannedStickerStore, groupHandler *GroupHandler) *AntiStickerHandler {
	return &AntiStickerHandler{store: store, groupHandler: groupHandler}
}

// CheckAndRevoke checks if a message contains a banned sticker and revokes it.
// Returns true if the sticker was banned and revoked, false otherwise.
// This should be called for every group message BEFORE command routing.
func (h *AntiStickerHandler) CheckAndRevoke(client *whatsmeow.Client, evt *events.Message) bool {
	// Only check group messages.
	if !evt.Info.IsGroup {
		return false
	}

	// Skip messages from the bot itself.
	if evt.Info.IsFromMe {
		return false
	}

	// Check if the message contains a sticker.
	stickerMsg := evt.Message.GetStickerMessage()
	if stickerMsg == nil {
		return false
	}

	// Get the file SHA256 hash from the sticker message.
	fileSHA256 := stickerMsg.GetFileSHA256()
	fileEncSHA256 := stickerMsg.GetFileEncSHA256()

	// Debug: log every sticker hash for troubleshooting.
	hashHex := hex.EncodeToString(fileSHA256)
	encHashHex := hex.EncodeToString(fileEncSHA256)
	log.Printf("[anti-sticker] Sticker received — FileSHA256: %s | FileEncSHA256: %s | from: %s | chat: %s",
		hashHex, encHashHex, evt.Info.Sender.User, evt.Info.Chat.String())

	if len(fileSHA256) == 0 {
		return false
	}

	if !h.store.IsBanned(hashHex) && !h.store.IsBanned(encHashHex) {
		return false
	}

	// Sticker is banned! Revoke the message.
	log.Printf("[anti-sticker] Banned sticker detected (hash: %s) from %s in %s — revoking",
		hashHex[:16]+"...", evt.Info.Sender.User, evt.Info.Chat.String())

	// BuildRevoke with sender JID (admin revocation of someone else's message).
	revokeMsg := client.BuildRevoke(evt.Info.Chat, evt.Info.Sender, evt.Info.ID)

	if _, err := client.SendMessage(context.Background(), evt.Info.Chat, revokeMsg); err != nil {
		log.Printf("[anti-sticker] failed to revoke message: %v", err)
		return false
	}

	return true
}

// HandleBanSticker adds a sticker's hash to the banned list (admin only).
// Usage: reply to a sticker with .bansticker, or .bansticker <hash>
func (h *AntiStickerHandler) HandleBanSticker(client *whatsmeow.Client, evt *events.Message, args []string) {
	if !evt.Info.IsGroup {
		utils.ReplyText(client, evt, config.MsgOnlyGroup)
		return
	}

	// Check admin.
	if !h.groupHandler.IsAdmin(client, evt.Info.Chat, evt.Info.Sender) {
		utils.ReplyText(client, evt, config.MsgOnlyAdmin)
		return
	}

	var hashHex string

	// Option 1: Reply to a sticker.
	quoted := utils.GetQuotedMessage(evt)
	if quoted != nil && quoted.GetStickerMessage() != nil {
		fileSHA256 := quoted.GetStickerMessage().GetFileSHA256()
		if len(fileSHA256) > 0 {
			hashHex = hex.EncodeToString(fileSHA256)
		}
	}

	// Option 2: Hash provided as argument.
	if hashHex == "" && len(args) > 0 {
		hashHex = strings.ToLower(args[0])
	}

	if hashHex == "" {
		utils.ReplyText(client, evt, "⚠️ Reply sticker yang ingin di-ban, atau kirim:\n.bansticker <sha256_hash>")
		return
	}

	if h.store.Add(hashHex) {
		utils.ReplyText(client, evt, fmt.Sprintf("✅ Sticker berhasil di-ban.\nHash: `%s`\nTotal banned: %d", hashHex, h.store.Count()))
	} else {
		utils.ReplyText(client, evt, "⚠️ Sticker ini sudah ada di daftar banned.")
	}
}

// HandleUnbanSticker removes a sticker's hash from the banned list (admin only).
// Usage: reply to a sticker with .unbansticker, or .unbansticker <hash>
func (h *AntiStickerHandler) HandleUnbanSticker(client *whatsmeow.Client, evt *events.Message, args []string) {
	if !evt.Info.IsGroup {
		utils.ReplyText(client, evt, config.MsgOnlyGroup)
		return
	}

	if !h.groupHandler.IsAdmin(client, evt.Info.Chat, evt.Info.Sender) {
		utils.ReplyText(client, evt, config.MsgOnlyAdmin)
		return
	}

	var hashHex string

	// Option 1: Reply to a sticker.
	quoted := utils.GetQuotedMessage(evt)
	if quoted != nil && quoted.GetStickerMessage() != nil {
		fileSHA256 := quoted.GetStickerMessage().GetFileSHA256()
		if len(fileSHA256) > 0 {
			hashHex = hex.EncodeToString(fileSHA256)
		}
	}

	// Option 2: Hash provided as argument.
	if hashHex == "" && len(args) > 0 {
		hashHex = strings.ToLower(args[0])
	}

	if hashHex == "" {
		utils.ReplyText(client, evt, "⚠️ Reply sticker yang ingin di-unban, atau kirim:\n.unbansticker <sha256_hash>")
		return
	}

	if h.store.Remove(hashHex) {
		utils.ReplyText(client, evt, fmt.Sprintf("✅ Sticker berhasil di-unban.\nHash: `%s`\nTotal banned: %d", hashHex, h.store.Count()))
	} else {
		utils.ReplyText(client, evt, "⚠️ Hash tidak ditemukan di daftar banned.")
	}
}
