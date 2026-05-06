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

// AntiImageHandler handles auto-deletion of images from banned users.
type AntiImageHandler struct {
	userStore    *services.BannedImageUserStore
	groupHandler *GroupHandler
}

// NewAntiImageHandler creates a new AntiImageHandler.
func NewAntiImageHandler(userStore *services.BannedImageUserStore, groupHandler *GroupHandler) *AntiImageHandler {
	return &AntiImageHandler{userStore: userStore, groupHandler: groupHandler}
}

// CheckAndRevoke checks if a message contains an image from a banned user and revokes it.
// Returns true if the image was revoked, false otherwise.
// This should be called for every group message BEFORE command routing.
func (h *AntiImageHandler) CheckAndRevoke(client *whatsmeow.Client, evt *events.Message) bool {
	// Only check group messages.
	if !evt.Info.IsGroup {
		return false
	}

	// Skip messages from the bot itself.
	if evt.Info.IsFromMe {
		return false
	}

	// Check if the message contains an image (including images with captions).
	imageMsg := evt.Message.GetImageMessage()
	if imageMsg == nil {
		return false
	}

	// Check if the user is banned from sending images.
	if h.userStore.IsBanned(evt.Info.Sender.ToNonAD().String()) {
		slog.Info("Banned user sent image — revoking", "user", evt.Info.Sender.User, "chat", evt.Info.Chat.String())

		// Revoke immediately
		revokeMsg := client.BuildRevoke(evt.Info.Chat, evt.Info.Sender, evt.Info.ID)
		if _, err := client.SendMessage(context.Background(), evt.Info.Chat, revokeMsg); err != nil {
			slog.Error("failed to revoke user's image message", "error", err)
			return false
		}
		return true
	}

	return false
}

// HandleBanImageUser bans a user from sending any image in the group (admin only).
// Usage: reply or tag user with .banimguser @user
func (h *AntiImageHandler) HandleBanImageUser(client *whatsmeow.Client, evt *events.Message, args []string) {
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
		utils.ReplyTextDirect(client, evt, "Reply pesan atau tag member yang ingin dilarang mengirim gambar.\nContoh: .banimg @member")
		return
	}

	// Prevent banning the bot itself.
	if client.Store.ID != nil && targetJID.User == client.Store.ID.User {
		utils.ReplyTextDirect(client, evt, "Tidak bisa ban bot sendiri.")
		return
	}

	targetStr := targetJID.ToNonAD().String()
	mentionText := fmt.Sprintf("@%s sekarang dilarang mengirim gambar.", targetJID.ToNonAD().User)
	if !h.userStore.Add(targetStr) {
		mentionText = fmt.Sprintf("@%s sudah ada di daftar larangan kirim gambar.", targetJID.ToNonAD().User)
	}
	utils.ReplyTextDirectWithMentions(client, evt, mentionText, []string{targetStr})
}

// HandleUnbanImageUser unbans a user, allowing them to send images again (admin only).
// Usage: reply or tag user with .unbanimguser @user
func (h *AntiImageHandler) HandleUnbanImageUser(client *whatsmeow.Client, evt *events.Message, args []string) {
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
		utils.ReplyTextDirect(client, evt, "Reply pesan atau tag member yang ingin diizinkan mengirim gambar lagi.\nContoh: .unbanimg @member")
		return
	}

	targetStr := targetJID.ToNonAD().String()
	mentionText := fmt.Sprintf("@%s sekarang diizinkan mengirim gambar kembali.", targetJID.ToNonAD().User)
	if !h.userStore.Remove(targetStr) {
		mentionText = fmt.Sprintf("@%s tidak ada di daftar larangan kirim gambar.", targetJID.ToNonAD().User)
	}
	utils.ReplyTextDirectWithMentions(client, evt, mentionText, []string{targetStr})
}

// HandleListBannedImageUsers shows all users forbidden from sending images (admin only).
func (h *AntiImageHandler) HandleListBannedImageUsers(client *whatsmeow.Client, evt *events.Message, args []string) {
	if !evt.Info.IsGroup {
		utils.ReplyTextDirect(client, evt, config.MsgOnlyGroup)
		return
	}

	if !h.groupHandler.IsAdmin(client, evt.Info.Chat, evt.Info.Sender) {
		utils.ReplyTextDirect(client, evt, config.MsgOnlyAdmin)
		return
	}

	list := h.userStore.ListFormatted()
	utils.ReplyTextDirect(client, evt, fmt.Sprintf("*Daftar User Dilarang Kirim Gambar*\n\n%s", list))
}
