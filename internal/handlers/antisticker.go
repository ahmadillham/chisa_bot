package handlers

import (
	"context"
	"fmt"
	"log/slog"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types/events"

	"chisa_bot/internal/config"
	"chisa_bot/internal/services"
	"chisa_bot/pkg/utils"
)

// AntiStickerHandler handles auto-deletion of stickers from banned users.
type AntiStickerHandler struct {
	userStore    *services.BannedStickerUserStore
	groupHandler *GroupHandler
}

// NewAntiStickerHandler creates a new AntiStickerHandler.
func NewAntiStickerHandler(userStore *services.BannedStickerUserStore, groupHandler *GroupHandler) *AntiStickerHandler {
	return &AntiStickerHandler{userStore: userStore, groupHandler: groupHandler}
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

	stickerMsg := evt.Message.GetStickerMessage()
	if stickerMsg == nil {
		return false
	}

	// 1. Check if the user is banned from sending stickers.
	if h.userStore.IsBanned(evt.Info.Sender.ToNonAD().String()) {
		slog.Info("Banned user sent sticker — revoking", "user", evt.Info.Sender.User, "chat", evt.Info.Chat.String())

		// Revoke immediately
		revokeMsg := client.BuildRevoke(evt.Info.Chat, evt.Info.Sender, evt.Info.ID)
		if _, err := client.SendMessage(context.Background(), evt.Info.Chat, revokeMsg); err != nil {
			slog.Error("failed to revoke user's message", "error", err)
			return false
		}
		return true
	}

	return false
}


// HandleBanStickerUser bans a user from sending any sticker in the group (admin only).
// Usage: reply or tag user with .banuser @user
func (h *AntiStickerHandler) HandleBanStickerUser(client *whatsmeow.Client, evt *events.Message, args []string) {
	if !evt.Info.IsGroup {
		utils.ReplyTextDirect(client, evt, config.MsgOnlyGroup)
		return
	}

	if !h.groupHandler.IsAdmin(client, evt.Info.Chat, evt.Info.Sender) {
		utils.ReplyTextDirect(client, evt, config.MsgOnlyAdmin)
		return
	}

	targetJID, found := utils.GetTargetJID(evt)
	if !found {
		utils.ReplyTextDirect(client, evt, "Reply pesan atau tag member yang ingin dilarang mengirim sticker.\nContoh: .banuser @member")
		return
	}

	targetStr := targetJID.ToNonAD().String()
	mentionText := fmt.Sprintf("@%s sekarang dilarang mengirim sticker.", targetJID.ToNonAD().User)
	if !h.userStore.Add(targetStr) {
		mentionText = fmt.Sprintf("@%s sudah ada di daftar larangan.", targetJID.ToNonAD().User)
	}
	utils.ReplyTextDirectWithMentions(client, evt, mentionText, []string{targetStr})
}

// HandleUnbanStickerUser unbans a user, allowing them to send stickers again (admin only).
// Usage: reply or tag user with .unbanuser @user
func (h *AntiStickerHandler) HandleUnbanStickerUser(client *whatsmeow.Client, evt *events.Message, args []string) {
	if !evt.Info.IsGroup {
		utils.ReplyTextDirect(client, evt, config.MsgOnlyGroup)
		return
	}

	if !h.groupHandler.IsAdmin(client, evt.Info.Chat, evt.Info.Sender) {
		utils.ReplyTextDirect(client, evt, config.MsgOnlyAdmin)
		return
	}

	targetJID, found := utils.GetTargetJID(evt)
	if !found {
		utils.ReplyTextDirect(client, evt, "Reply pesan atau tag member yang ingin diizinkan mengirim sticker lagi.\nContoh: .unbanuser @member")
		return
	}

	targetStr := targetJID.ToNonAD().String()
	mentionText := fmt.Sprintf("@%s sekarang diizinkan mengirim sticker kembali.", targetJID.ToNonAD().User)
	if !h.userStore.Remove(targetStr) {
		mentionText = fmt.Sprintf("@%s tidak ada di daftar larangan.", targetJID.ToNonAD().User)
	}
	utils.ReplyTextDirectWithMentions(client, evt, mentionText, []string{targetStr})
}

// HandleListBannedUsers shows all users forbidden from sending stickers (admin only).
func (h *AntiStickerHandler) HandleListBannedUsers(client *whatsmeow.Client, evt *events.Message, args []string) {
	if !evt.Info.IsGroup {
		utils.ReplyTextDirect(client, evt, config.MsgOnlyGroup)
		return
	}

	if !h.groupHandler.IsAdmin(client, evt.Info.Chat, evt.Info.Sender) {
		utils.ReplyTextDirect(client, evt, config.MsgOnlyAdmin)
		return
	}

	list := h.userStore.ListFormatted()
	utils.ReplyTextDirect(client, evt, fmt.Sprintf("*Daftar User Dilarang Kirim Sticker*\n\n%s", list))
}
