package handlers

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"go.mau.fi/whatsmeow"
	waProto "go.mau.fi/whatsmeow/binary/proto"
	"go.mau.fi/whatsmeow/types/events"

	"chisa_bot/internal/config"
	"chisa_bot/internal/services"
	"chisa_bot/pkg/utils"
)

// AntiImageHandler handles auto-deletion of image/video/GIF media from banned users.
type AntiImageHandler struct {
	userStore    *services.BannedImageUserStore
	groupHandler *GroupHandler
}

// NewAntiImageHandler creates a new AntiImageHandler.
func NewAntiImageHandler(userStore *services.BannedImageUserStore, groupHandler *GroupHandler) *AntiImageHandler {
	return &AntiImageHandler{userStore: userStore, groupHandler: groupHandler}
}

// CheckAndRevoke checks if a message contains image/video/GIF media from a banned user and revokes it.
// Returns true if the media was revoked, false otherwise.
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

	// Check if the message contains image/video/GIF media, including video/GIF documents.
	if !isImageBanMedia(evt.Message) {
		return false
	}

	// Check if the user is banned from sending image/video/GIF media in this group.
	if h.userStore.IsBanned(evt.Info.Sender.ToNonAD().String(), evt.Info.Chat.String()) {
		slog.Info("Banned user sent image/video/GIF media — revoking", "user", evt.Info.Sender.User, "chat", evt.Info.Chat.String())

		// Revoke immediately
		revokeMsg := client.BuildRevoke(evt.Info.Chat, evt.Info.Sender, evt.Info.ID)
		if _, err := client.SendMessage(context.Background(), evt.Info.Chat, revokeMsg); err != nil {
			slog.Error("failed to revoke user's image/video/GIF media message", "error", err)
			return false
		}
		return true
	}

	return false
}

func isImageBanMedia(msg *waProto.Message) bool {
	if msg == nil {
		return false
	}

	msg = utils.UnwrapViewOnce(msg)
	if msg.GetImageMessage() != nil || msg.GetVideoMessage() != nil {
		return true
	}

	doc := msg.GetDocumentMessage()
	if doc == nil {
		return false
	}

	mimetype := strings.ToLower(strings.TrimSpace(doc.GetMimetype()))
	filename := strings.ToLower(strings.TrimSpace(doc.GetFileName()))

	if strings.HasPrefix(mimetype, "video/") || mimetype == "image/gif" || strings.Contains(mimetype, "gif") {
		return true
	}

	videoExts := []string{".gif", ".mp4", ".mov", ".mkv", ".webm", ".avi", ".3gp", ".m4v"}
	for _, ext := range videoExts {
		if strings.HasSuffix(filename, ext) {
			return true
		}
	}

	return false
}

// HandleBanImageUser bans a user from sending image/video/GIF media in the group (admin only).
// Usage: reply or tag user with .banimg @user
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
		utils.ReplyTextDirect(client, evt, "Reply pesan atau tag member yang ingin dilarang mengirim gambar/video/GIF.\nContoh: .banimg @member")
		return
	}

	// Prevent banning the bot itself.
	if client.Store.ID != nil && targetJID.User == client.Store.ID.User {
		utils.ReplyTextDirect(client, evt, "Tidak bisa ban bot sendiri.")
		return
	}

	targetStr := targetJID.ToNonAD().String()
	groupStr := evt.Info.Chat.String()
	mentionText := fmt.Sprintf("@%s sekarang dilarang mengirim gambar/video/GIF.", targetJID.ToNonAD().User)
	if !h.userStore.Add(targetStr, groupStr) {
		mentionText = fmt.Sprintf("@%s sudah ada di daftar larangan kirim gambar/video/GIF.", targetJID.ToNonAD().User)
	}
	utils.ReplyTextDirectWithMentions(client, evt, mentionText, []string{targetStr})
}

// HandleUnbanImageUser unbans a user, allowing them to send image/video/GIF media again (admin only).
// Usage: reply or tag user with .unbanimg @user
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
		utils.ReplyTextDirect(client, evt, "Reply pesan atau tag member yang ingin diizinkan mengirim gambar/video/GIF lagi.\nContoh: .unbanimg @member")
		return
	}

	targetStr := targetJID.ToNonAD().String()
	groupStr := evt.Info.Chat.String()
	mentionText := fmt.Sprintf("@%s sekarang diizinkan mengirim gambar/video/GIF kembali.", targetJID.ToNonAD().User)
	if !h.userStore.Remove(targetStr, groupStr) {
		mentionText = fmt.Sprintf("@%s tidak ada di daftar larangan kirim gambar/video/GIF.", targetJID.ToNonAD().User)
	}
	utils.ReplyTextDirectWithMentions(client, evt, mentionText, []string{targetStr})
}
