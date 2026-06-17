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

// AntiChatHandler handles auto-deletion of all chat messages from banned users.
type AntiChatHandler struct {
	userStore    *services.BannedChatUserStore
	groupHandler *GroupHandler
}

// NewAntiChatHandler creates a new AntiChatHandler.
func NewAntiChatHandler(userStore *services.BannedChatUserStore, groupHandler *GroupHandler) *AntiChatHandler {
	return &AntiChatHandler{userStore: userStore, groupHandler: groupHandler}
}

// CheckAndRevoke checks if a message is from a banned user and revokes it.
// Returns true if the message was revoked, false otherwise.
// This should be called for every group message BEFORE command routing.
func (h *AntiChatHandler) CheckAndRevoke(client *whatsmeow.Client, evt *events.Message) bool {
	// Only check group messages.
	if !evt.Info.IsGroup {
		return false
	}

	// Skip messages from the bot itself.
	if evt.Info.IsFromMe {
		return false
	}

	// Check if the user is banned from sending chat in this group.
	if h.userStore.IsBanned(evt.Info.Sender.ToNonAD().String(), evt.Info.Chat.String()) {
		slog.Info("Banned user sent chat — revoking", "user", evt.Info.Sender.User, "chat", evt.Info.Chat.String())

		// Revoke immediately
		revokeMsg := client.BuildRevoke(evt.Info.Chat, evt.Info.Sender, evt.Info.ID)
		if _, err := client.SendMessage(context.Background(), evt.Info.Chat, revokeMsg); err != nil {
			slog.Error("failed to revoke user's chat message", "error", err)
			return false
		}
		return true
	}

	return false
}

// HandleBanChatUser bans a user from sending any chat in the group (admin only).
// Usage: reply or tag user with .banchat @user
func (h *AntiChatHandler) HandleBanChatUser(client *whatsmeow.Client, evt *events.Message, args []string) {
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
		utils.ReplyTextDirect(client, evt, "Reply pesan atau tag member yang ingin dilarang mengirim chat.\nContoh: .banchat @member")
		return
	}

	// Prevent banning the bot itself.
	if client.Store.ID != nil && targetJID.User == client.Store.ID.User {
		utils.ReplyTextDirect(client, evt, "Tidak bisa ban bot sendiri.")
		return
	}

	targetStr := targetJID.ToNonAD().String()
	groupStr := evt.Info.Chat.String()
	mentionText := fmt.Sprintf("@%s sekarang dilarang mengirim chat.", targetJID.ToNonAD().User)
	if !h.userStore.Add(targetStr, groupStr) {
		mentionText = fmt.Sprintf("@%s sudah ada di daftar larangan kirim chat.", targetJID.ToNonAD().User)
	}
	utils.ReplyTextDirectWithMentions(client, evt, mentionText, []string{targetStr})
}

// HandleUnbanChatUser unbans a user, allowing them to send chat again (admin only).
// Usage: reply or tag user with .unbanchat @user
func (h *AntiChatHandler) HandleUnbanChatUser(client *whatsmeow.Client, evt *events.Message, args []string) {
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
		utils.ReplyTextDirect(client, evt, "Reply pesan atau tag member yang ingin diizinkan mengirim chat lagi.\nContoh: .unbanchat @member")
		return
	}

	targetStr := targetJID.ToNonAD().String()
	groupStr := evt.Info.Chat.String()
	mentionText := fmt.Sprintf("@%s sekarang diizinkan mengirim chat kembali.", targetJID.ToNonAD().User)
	if !h.userStore.Remove(targetStr, groupStr) {
		mentionText = fmt.Sprintf("@%s tidak ada di daftar larangan kirim chat.", targetJID.ToNonAD().User)
	}
	utils.ReplyTextDirectWithMentions(client, evt, mentionText, []string{targetStr})
}
