package handlers

import (
	"fmt"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types/events"

	"chisa_bot/internal/config"
	"chisa_bot/internal/services"
	"chisa_bot/pkg/utils"
)

// VIPHandler handles granting and revoking VIP rights.
type VIPHandler struct {
	vipStore *services.VIPUserStore
}

// NewVIPHandler creates a new VIPHandler.
func NewVIPHandler(vipStore *services.VIPUserStore) *VIPHandler {
	return &VIPHandler{vipStore: vipStore}
}

// isOwner checks if the sender is the owner of the bot.
func (h *VIPHandler) isOwner(evt *events.Message) bool {
	senderStr := evt.Info.Sender.ToNonAD().String()

	for _, owner := range config.OwnerJIDs {
		if senderStr == owner {
			return true
		}
	}

	// Debugging log if they don't match
	fmt.Printf("[DEBUG VIP] Sender: '%s', Expected Owners: %v\n", senderStr, config.OwnerJIDs)
	return false
}

// HandleAddVIP adds a user to the VIP list (owner only).
func (h *VIPHandler) HandleAddVIP(client *whatsmeow.Client, evt *events.Message, args []string) {
	if !h.isOwner(evt) {
		utils.ReplyTextDirect(client, evt, "Hanya pemilik bot yang bisa menggunakan perintah ini.")
		return
	}

	targetJID, found := utils.GetTargetJID(evt)
	if !found {
		utils.ReplyTextDirect(client, evt, "Reply pesan atau tag user yang ingin dijadikan VIP.\nContoh: .addvip @user")
		return
	}

	targetStr := targetJID.ToNonAD().String()
	mentionText := fmt.Sprintf("@%s sekarang memiliki hak VIP (kebal kick & ban).", targetJID.ToNonAD().User)
	if !h.vipStore.Add(targetStr) {
		mentionText = fmt.Sprintf("@%s sudah menjadi VIP.", targetJID.ToNonAD().User)
	}
	utils.ReplyTextDirectWithMentions(client, evt, mentionText, []string{targetStr})
}

// HandleRemoveVIP removes a user from the VIP list (owner only).
func (h *VIPHandler) HandleRemoveVIP(client *whatsmeow.Client, evt *events.Message, args []string) {
	if !h.isOwner(evt) {
		utils.ReplyTextDirect(client, evt, "Hanya pemilik bot yang bisa menggunakan perintah ini.")
		return
	}

	targetJID, found := utils.GetTargetJID(evt)
	if !found {
		utils.ReplyTextDirect(client, evt, "Reply pesan atau tag user yang hak VIP-nya ingin dicabut.\nContoh: .rmvip @user")
		return
	}

	targetStr := targetJID.ToNonAD().String()
	mentionText := fmt.Sprintf("Hak VIP @%s telah dicabut.", targetJID.ToNonAD().User)
	if !h.vipStore.Remove(targetStr) {
		mentionText = fmt.Sprintf("@%s bukan VIP.", targetJID.ToNonAD().User)
	}
	utils.ReplyTextDirectWithMentions(client, evt, mentionText, []string{targetStr})
}

// HandleListVIP shows all VIP users (owner only).
func (h *VIPHandler) HandleListVIP(client *whatsmeow.Client, evt *events.Message, args []string) {
	if !h.isOwner(evt) {
		utils.ReplyTextDirect(client, evt, "Hanya pemilik bot yang bisa menggunakan perintah ini.")
		return
	}

	list := h.vipStore.ListFormatted()
	utils.ReplyTextDirect(client, evt, fmt.Sprintf("*Daftar User VIP*\n\n%s", list))
}
