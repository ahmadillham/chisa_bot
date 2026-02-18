package handlers

import (
	"context"
	"fmt"
	"log"
	"time"

	"go.mau.fi/whatsmeow"
	waProto "go.mau.fi/whatsmeow/binary/proto"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	"google.golang.org/protobuf/proto"

	"chisa_bot/internal/config"
	"chisa_bot/internal/services"
	"chisa_bot/pkg/utils"
)

// GroupHandler handles group management features.
type GroupHandler struct {
	warnStore *services.WarnStore
}

// NewGroupHandler creates a new GroupHandler.
func NewGroupHandler(warnStore *services.WarnStore) *GroupHandler {
	return &GroupHandler{
		warnStore: warnStore,
	}
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

	h.TagAll(client, evt.Info.Chat, evt.Message, evt.Info.ID, evt.Info.Sender, "📢 *Tag All Members*")
}

// TagAll mentions all group members with a custom message.
func (h *GroupHandler) TagAll(client *whatsmeow.Client, chatJID types.JID, quotedMsg *waProto.Message, stanzaID string, senderJID types.JID, title string) {
	groupInfo, err := client.GetGroupInfo(context.Background(), chatJID)
	if err != nil {
		log.Printf("[tagall] failed to get group info: %v", err)
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
		log.Printf("[tagall] failed to send: %v", err)
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

// HandleWarn warns a user. 3 warnings = kick.
func (h *GroupHandler) HandleWarn(client *whatsmeow.Client, evt *events.Message, args []string) {
	if !evt.Info.IsGroup {
		utils.ReplyText(client, evt, config.MsgOnlyGroup)
		return
	}

	if !h.IsAdmin(client, evt.Info.Chat, evt.Info.Sender) {
		utils.ReplyText(client, evt, config.MsgOnlyAdmin)
		return
	}

	var targetJID types.JID
	found := false

	// Target detection (Reply > Mention > Args)
	if evt.Message.GetExtendedTextMessage() != nil {
		ctxInfo := evt.Message.GetExtendedTextMessage().GetContextInfo()
		
		// 1. Reply
		if ctxInfo != nil && ctxInfo.Participant != nil {
			targetJID, _ = types.ParseJID(*ctxInfo.Participant)
			found = true
		} else {
			// 2. Mention
			mentionList := ctxInfo.GetMentionedJID()
			if len(mentionList) > 0 {
				targetJID, _ = types.ParseJID(mentionList[0])
				found = true
			}
		}
	}

	if !found {
		// 3. Try parsing args if phone number is provided (advanced usage, optional but good)
		// For now simple usage as requested: Reply or Tag.
		utils.ReplyText(client, evt, "⚠️ Reply pesan atau tag member yang ingin di-warn.\nContoh: .warn @member")
		return
	}

	// Increment warning
	count := h.warnStore.AddWarning(evt.Info.Chat.String(), targetJID.String())

	if count >= 3 {
		// KICK
		utils.ReplyText(client, evt, fmt.Sprintf("⚠️ *PERINGATAN KE-%d (FINAL)*\n@%s otomatis di-kick dari grup.", count, targetJID.User))
		
		// Give a moment for the message to send before kicking (optional, but good practice)
		time.Sleep(1 * time.Second)

		// Use "remove" string literal which is standard for UpdateGroupParticipants
		_, err := client.UpdateGroupParticipants(context.Background(), evt.Info.Chat, []types.JID{targetJID}, whatsmeow.ParticipantChangeRemove)
		if err != nil {
			log.Printf("[warn] failed to kick: %v", err)
			utils.ReplyText(client, evt, "❌ Gagal meng-kick member automatically. Pastikan bot adalah admin.")
		} else {
			// Reset warnings on successful kick
			h.warnStore.ResetWarning(evt.Info.Chat.String(), targetJID.String())
		}
	} else {
		// WARNING 1 or 2
		msg := fmt.Sprintf("⚠️ *PERINGATAN KE-%d*\n\n@%s, tolong ikuti aturan grup.\nPeringatan ke-3 = Kick.", count, targetJID.User)
		// Send as mention
		h.sendGroupMention(client, evt.Info.Chat, msg, []string{targetJID.String()})
	}
}
