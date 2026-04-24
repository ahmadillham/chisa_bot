package handlers

import (
	"context"
	"log/slog"

	"go.mau.fi/whatsmeow"
	waProto "go.mau.fi/whatsmeow/binary/proto"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	"google.golang.org/protobuf/proto"

	"chisa_bot/pkg/utils"
)

// GroupHandler handles group management features.
type GroupHandler struct {
}

// NewGroupHandler creates a new GroupHandler.
func NewGroupHandler() *GroupHandler {
	return &GroupHandler{}
}

// IsAdmin checks if the user is an admin in the group.
func (h *GroupHandler) IsAdmin(client *whatsmeow.Client, chatJID types.JID, userJID types.JID) bool {
	groupInfo, err := client.GetGroupInfo(context.Background(), chatJID)
	if err != nil {
		slog.Error("failed to get info", "error", err)
		return false
	}
	for _, p := range groupInfo.Participants {
		if p.JID.User == userJID.User {
			return p.IsAdmin || p.IsSuperAdmin
		}
	}
	return false
}

// HandleGroupParticipants handles join/leave events in groups.
func (h *GroupHandler) HandleGroupParticipants(client *whatsmeow.Client, evt *events.GroupInfo) {
	if evt.JID.Server != types.GroupServer {
		return
	}

	for _, join := range evt.Join {
		slog.Info("User joined in", "user", join.String(), "group", evt.JID.String())
		welcomeMsg := "Selamat datang member baru"
		h.sendGroupMention(client, evt.JID, welcomeMsg, []string{join.String()})
	}

	for _, leave := range evt.Leave {
		slog.Info("User left from", "user", leave.String(), "group", evt.JID.String())
		goodbyeMsg := "Good Bye"
		h.sendGroupMention(client, evt.JID, goodbyeMsg, []string{leave.String()})
	}
}

// HandleTagAll mentions all group members (admin only).
func (h *GroupHandler) HandleTagAll(client *whatsmeow.Client, evt *events.Message) {
	if !evt.Info.IsGroup {
		utils.ReplyTextDirect(client, evt, "Perintah ini hanya bisa digunakan di grup.")
		return
	}

	if !h.IsAdmin(client, evt.Info.Chat, evt.Info.Sender) {
		utils.ReplyTextDirect(client, evt, "Hanya admin yang bisa menggunakan perintah ini.")
		return
	}

	h.TagAll(client, evt.Info.Chat, evt.Message, evt.Info.ID, evt.Info.Sender, "📢 *Tag All Members*")
}

// TagAll mentions all group members with a custom message.
func (h *GroupHandler) TagAll(client *whatsmeow.Client, chatJID types.JID, quotedMsg *waProto.Message, stanzaID string, senderJID types.JID, title string) {
	groupInfo, err := client.GetGroupInfo(context.Background(), chatJID)
	if err != nil {
		slog.Error("failed to get group info", "error", err)
		return
	}

	var mentionJIDs []string
	for _, p := range groupInfo.Participants {
		mentionJIDs = append(mentionJIDs, p.JID.String())
	}

	msg := &waProto.Message{
		ExtendedTextMessage: &waProto.ExtendedTextMessage{
			Text: proto.String(title),
			ContextInfo: &waProto.ContextInfo{
				StanzaID:      proto.String(stanzaID),
				Participant:   proto.String(senderJID.String()),
				QuotedMessage: quotedMsg,
				MentionedJID:  mentionJIDs,
			},
		},
	}

	if _, err := client.SendMessage(context.Background(), chatJID, msg); err != nil {
		slog.Error("failed to send", "error", err)
	}
}

// HandleKick kicks a member (admin only).
func (h *GroupHandler) HandleKick(client *whatsmeow.Client, evt *events.Message, args []string) {
	if !evt.Info.IsGroup {
		utils.ReplyTextDirect(client, evt, "Perintah ini hanya bisa digunakan di grup.")
		return
	}

	if !h.IsAdmin(client, evt.Info.Chat, evt.Info.Sender) {
		utils.ReplyTextDirect(client, evt, "Hanya admin yang bisa menggunakan perintah ini.")
		return
	}

	targetJID, found := utils.GetTargetJID(evt)

	if !found {
		utils.ReplyTextDirect(client, evt, "Tag atau reply user yang ingin di-kick.")
		return
	}

	// preventing kicking self (bot) or admins should be handled by WhatsApp anyway (admins can kick admins unless creator, bot can't kick admins if not admin etc).
	// But let's just try.

	// Use "remove" string literal which is standard for UpdateGroupParticipants
	_, err := client.UpdateGroupParticipants(context.Background(), evt.Info.Chat, []types.JID{targetJID}, whatsmeow.ParticipantChangeRemove)
	if err != nil {
		slog.Error("failed", "error", err)
		utils.ReplyTextDirect(client, evt, "Gagal kick member. Pastikan bot adalah admin.")
		return
	}
}

func (h *GroupHandler) sendGroupMention(client *whatsmeow.Client, chatJID types.JID, text string, mentionJIDs []string) {
	msg := &waProto.Message{
		ExtendedTextMessage: &waProto.ExtendedTextMessage{
			Text: proto.String(text),
			ContextInfo: &waProto.ContextInfo{
				MentionedJID: mentionJIDs,
			},
		},
	}

	if _, err := client.SendMessage(context.Background(), chatJID, msg); err != nil {
		slog.Error("failed to send mention message", "error", err)
	}
}


