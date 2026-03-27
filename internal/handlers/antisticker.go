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

	// Revoke immediately — speed matters for anti-sticker.
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
		utils.ReplyTextDirect(client, evt, config.MsgOnlyGroup)
		return
	}

	// Check admin.
	if !h.groupHandler.IsAdmin(client, evt.Info.Chat, evt.Info.Sender) {
		utils.ReplyTextDirect(client, evt, config.MsgOnlyAdmin)
		return
	}

	var hashHex string
	var alias string

	// Option 1: Reply to a sticker.
	quoted := utils.GetQuotedMessage(evt)
	if quoted != nil && quoted.GetStickerMessage() != nil {
		fileSHA256 := quoted.GetStickerMessage().GetFileSHA256()
		if len(fileSHA256) > 0 {
			hashHex = hex.EncodeToString(fileSHA256)
		}
	}

	// Optional alias from args (e.g. .bansticker myalias while replying).
	if len(args) > 0 {
		if hashHex != "" {
			// Replying to sticker + provided alias name.
			alias = args[0]
		} else {
			// No reply, treat arg as hash.
			hashHex = strings.ToLower(args[0])
			if len(args) > 1 {
				alias = args[1]
			}
		}
	}

	if hashHex == "" {
		utils.ReplyTextDirect(client, evt, "⚠️ Reply sticker yang ingin di-ban, atau kirim:\n.bansticker <hash> [alias]")
		return
	}

	usedAlias, added := h.store.Add(hashHex, alias)
	if added {
		utils.ReplyTextDirect(client, evt, fmt.Sprintf("✅ Sticker berhasil di-ban.\nAlias: *%s*\nTotal banned: %d", usedAlias, h.store.Count()))
	} else {
		utils.ReplyTextDirect(client, evt, fmt.Sprintf("⚠️ Sticker ini sudah ada di daftar banned (alias: *%s*).", usedAlias))
	}
}

// HandleUnbanSticker removes a sticker's hash from the banned list (admin only).
// Usage: reply to a sticker with .unbansticker, or .unbansticker <hash>
func (h *AntiStickerHandler) HandleUnbanSticker(client *whatsmeow.Client, evt *events.Message, args []string) {
	if !evt.Info.IsGroup {
		utils.ReplyTextDirect(client, evt, config.MsgOnlyGroup)
		return
	}

	if !h.groupHandler.IsAdmin(client, evt.Info.Chat, evt.Info.Sender) {
		utils.ReplyTextDirect(client, evt, config.MsgOnlyAdmin)
		return
	}

	var identifier string

	// Option 1: Reply to a sticker.
	quoted := utils.GetQuotedMessage(evt)
	if quoted != nil && quoted.GetStickerMessage() != nil {
		fileSHA256 := quoted.GetStickerMessage().GetFileSHA256()
		if len(fileSHA256) > 0 {
			identifier = hex.EncodeToString(fileSHA256)
		}
	}

	// Option 2: Alias or hash provided as argument.
	if identifier == "" && len(args) > 0 {
		identifier = args[0]
	}

	if identifier == "" {
		utils.ReplyTextDirect(client, evt, "⚠️ Reply sticker atau kirim:\n.unbansticker <alias>\n.unbansticker <hash>")
		return
	}

	if h.store.Remove(identifier) {
		utils.ReplyTextDirect(client, evt, fmt.Sprintf("✅ Sticker berhasil di-unban.\nTotal banned: %d", h.store.Count()))
	} else {
		utils.ReplyTextDirect(client, evt, "⚠️ Alias atau hash tidak ditemukan di daftar banned.")
	}
}

// HandleListBanned shows all banned stickers with their aliases (admin only).
func (h *AntiStickerHandler) HandleListBanned(client *whatsmeow.Client, evt *events.Message, args []string) {
	if !evt.Info.IsGroup {
		utils.ReplyTextDirect(client, evt, config.MsgOnlyGroup)
		return
	}

	if !h.groupHandler.IsAdmin(client, evt.Info.Chat, evt.Info.Sender) {
		utils.ReplyTextDirect(client, evt, config.MsgOnlyAdmin)
		return
	}

	list := h.store.ListFormatted()
	utils.ReplyTextDirect(client, evt, fmt.Sprintf("🚫 *Daftar Sticker Banned*\n\n%s", list))
}
