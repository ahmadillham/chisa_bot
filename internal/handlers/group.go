package handlers

import (
	"context"
	"fmt"
	"log"

	"go.mau.fi/whatsmeow"
	waProto "go.mau.fi/whatsmeow/binary/proto"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	"google.golang.org/protobuf/proto"

	"chisa_bot/pkg/utils"
)

// GroupHandler handles group management features.
type GroupHandler struct{}

// NewGroupHandler creates a new GroupHandler.
func NewGroupHandler() *GroupHandler {
	return &GroupHandler{}
}

// IsAdmin checks if the user is an admin in the group.
func (h *GroupHandler) IsAdmin(client *whatsmeow.Client, chatJID types.JID, userJID types.JID) bool {
	groupInfo, err := client.GetGroupInfo(context.Background(), chatJID)
	if err != nil {
		log.Printf("[group] failed to get info: %v", err)
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
		log.Printf("[group] User joined: %s in %s", join.String(), evt.JID.String())
		welcomeMsg := fmt.Sprintf(
			"👋 Halo @%s!\nSelamat datang di grup! 🎉\n\nSemoga betah ya~ 😊",
			join.User,
		)
		h.sendGroupMention(client, evt.JID, welcomeMsg, []string{join.String()})
	}

	for _, leave := range evt.Leave {
		log.Printf("[group] User left: %s from %s", leave.String(), evt.JID.String())
		goodbyeMsg := fmt.Sprintf(
			"👋 Sampai jumpa @%s!\nTerima kasih sudah meramaikan grup. 👋",
			leave.User,
		)
		h.sendGroupMention(client, evt.JID, goodbyeMsg, []string{leave.String()})
	}
}

// HandleTagAll mentions all group members (admin only).
func (h *GroupHandler) HandleTagAll(client *whatsmeow.Client, evt *events.Message) {
	if !evt.Info.IsGroup {
		utils.ReplyText(client, evt, "⚠️ Perintah ini hanya bisa digunakan di grup.")
		return
	}

	if !h.IsAdmin(client, evt.Info.Chat, evt.Info.Sender) {
		utils.ReplyText(client, evt, "⚠️ Hanya admin yang bisa menggunakan perintah ini.")
		return
	}

	groupInfo, err := client.GetGroupInfo(context.Background(), evt.Info.Chat)
	if err != nil {
		log.Printf("[tagall] failed to get group info: %v", err)
		utils.ReplyText(client, evt, "❌ Gagal mendapatkan info grup.")
		return
	}

	var mentionJIDs []string
	for _, p := range groupInfo.Participants {
		mentionJIDs = append(mentionJIDs, p.JID.String())
	}

	text := "📢 *Tag All Members*"

	msg := &waProto.Message{
		ExtendedTextMessage: &waProto.ExtendedTextMessage{
			Text: proto.String(text),
			ContextInfo: &waProto.ContextInfo{
				StanzaID:      proto.String(evt.Info.ID),
				Participant:   proto.String(evt.Info.Sender.String()),
				QuotedMessage: evt.Message,
				MentionedJID:  mentionJIDs,
			},
		},
	}

	if _, err := client.SendMessage(context.Background(), evt.Info.Chat, msg); err != nil {
		log.Printf("[tagall] failed to send: %v", err)
		utils.ReplyText(client, evt, "❌ Gagal mengirim tag all.")
	}
}

// HandleKick kicks a member (admin only).
func (h *GroupHandler) HandleKick(client *whatsmeow.Client, evt *events.Message, args []string) {
	if !evt.Info.IsGroup {
		utils.ReplyText(client, evt, "⚠️ Perintah ini hanya bisa digunakan di grup.")
		return
	}

	if !h.IsAdmin(client, evt.Info.Chat, evt.Info.Sender) {
		utils.ReplyText(client, evt, "⚠️ Hanya admin yang bisa menggunakan perintah ini.")
		return
	}

	var targetJID types.JID
	found := false

	// Check mentions in the message itself
	if evt.Message.GetExtendedTextMessage() != nil {
		mentionList := evt.Message.GetExtendedTextMessage().GetContextInfo().GetMentionedJID()
		if len(mentionList) > 0 {
			targetJID, _ = types.ParseJID(mentionList[0])
			found = true
		} else {
			// Check quoted message sender
			ctxInfo := evt.Message.GetExtendedTextMessage().GetContextInfo()
			if ctxInfo != nil && ctxInfo.Participant != nil {
				targetJID, _ = types.ParseJID(*ctxInfo.Participant)
				found = true
			}
		}
	}

	if !found {
		utils.ReplyText(client, evt, "⚠️ Tag atau reply user yang ingin di-kick.")
		return
	}

	// preventing kicking self (bot) or admins should be handled by WhatsApp anyway (admins can kick admins unless creator, bot can't kick admins if not admin etc). 
	// But let's just try.

	// Use "remove" string literal which is standard for UpdateGroupParticipants
	_, err := client.UpdateGroupParticipants(context.Background(), evt.Info.Chat, []types.JID{targetJID}, whatsmeow.ParticipantChangeRemove)
	if err != nil {
		log.Printf("[kick] failed: %v", err)
		utils.ReplyText(client, evt, "❌ Gagal kick member. Pastikan bot adalah admin.")
		return
	}
	utils.ReplyText(client, evt, "👋 Sayonara!")
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
		log.Printf("[group] failed to send mention message: %v", err)
	}
}
