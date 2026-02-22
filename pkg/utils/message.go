package utils

import (
	"context"
	"fmt"

	"math/rand"
	"time"

	"go.mau.fi/whatsmeow"
	waProto "go.mau.fi/whatsmeow/binary/proto"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	"google.golang.org/protobuf/proto"
)

// GetTextFromMessage extracts the text body from various message types.
func GetTextFromMessage(msg *events.Message) string {
	if msg.Message.GetConversation() != "" {
		return msg.Message.GetConversation()
	}
	if msg.Message.GetExtendedTextMessage() != nil {
		return msg.Message.GetExtendedTextMessage().GetText()
	}
	if msg.Message.GetImageMessage() != nil {
		return msg.Message.GetImageMessage().GetCaption()
	}
	if msg.Message.GetVideoMessage() != nil {
		return msg.Message.GetVideoMessage().GetCaption()
	}
	if msg.Message.GetDocumentMessage() != nil {
		return msg.Message.GetDocumentMessage().GetCaption()
	}
	return ""
}

// newContextInfo builds a ContextInfo that quotes the triggering message.
func newContextInfo(evt *events.Message) *waProto.ContextInfo {
	return &waProto.ContextInfo{
		StanzaID:      proto.String(evt.Info.ID),
		Participant:   proto.String(evt.Info.Sender.String()),
		QuotedMessage: evt.Message,
	}
}

// SimulateTyping adds a random delay (0.5s - 1.5s) to mimic human behavior.
// It also sends a "coding/recording" presence update.
func SimulateTyping(client *whatsmeow.Client, chatJID types.JID) {
	// Send "typing" presence
	client.SendChatPresence(context.Background(), chatJID, types.ChatPresenceComposing, types.ChatPresenceMediaText)

	// Random delay 3000ms - 5000ms (3s - 5s)
	ms := 3000 + rand.Intn(2000)
	time.Sleep(time.Duration(ms) * time.Millisecond)

	// Send "paused" presence
	client.SendChatPresence(context.Background(), chatJID, types.ChatPresencePaused, types.ChatPresenceMediaText)
}

// ReplyText sends a text reply to the message that triggered it.
func ReplyText(client *whatsmeow.Client, evt *events.Message, text string) error {
	SimulateTyping(client, evt.Info.Chat)
	msg := &waProto.Message{
		ExtendedTextMessage: &waProto.ExtendedTextMessage{
			Text:        proto.String(text),
			ContextInfo: newContextInfo(evt),
		},
	}
	_, err := client.SendMessage(context.Background(), evt.Info.Chat, msg)
	return err
}

// ReplyTextWithMentions sends a text reply with specific mentions.
func ReplyTextWithMentions(client *whatsmeow.Client, evt *events.Message, text string, mentions []string) error {
	SimulateTyping(client, evt.Info.Chat)
	ctxInfo := newContextInfo(evt)
	ctxInfo.MentionedJID = mentions

	msg := &waProto.Message{
		ExtendedTextMessage: &waProto.ExtendedTextMessage{
			Text:        proto.String(text),
			ContextInfo: ctxInfo,
		},
	}
	_, err := client.SendMessage(context.Background(), evt.Info.Chat, msg)
	return err
}

// ReplyImage sends an image reply.
func ReplyImage(client *whatsmeow.Client, evt *events.Message, imageData []byte, mimetype string, caption string) error {
	SimulateTyping(client, evt.Info.Chat)
	uploaded, err := client.Upload(context.Background(), imageData, whatsmeow.MediaImage)
	if err != nil {
		return fmt.Errorf("failed to upload image: %w", err)
	}

	msg := &waProto.Message{
		ImageMessage: &waProto.ImageMessage{
			URL:           proto.String(uploaded.URL),
			DirectPath:    proto.String(uploaded.DirectPath),
			MediaKey:      uploaded.MediaKey,
			FileEncSHA256: uploaded.FileEncSHA256,
			FileSHA256:    uploaded.FileSHA256,
			FileLength:    proto.Uint64(uint64(len(imageData))),
			Mimetype:      proto.String(mimetype),
			Caption:       proto.String(caption),
			ContextInfo:   newContextInfo(evt),
		},
	}
	_, err = client.SendMessage(context.Background(), evt.Info.Chat, msg)
	return err
}

// ReplyVideo sends a video reply.
func ReplyVideo(client *whatsmeow.Client, evt *events.Message, videoData []byte, mimetype string, caption string) error {
	SimulateTyping(client, evt.Info.Chat)
	uploaded, err := client.Upload(context.Background(), videoData, whatsmeow.MediaVideo)
	if err != nil {
		return fmt.Errorf("failed to upload video: %w", err)
	}

	msg := &waProto.Message{
		VideoMessage: &waProto.VideoMessage{
			URL:           proto.String(uploaded.URL),
			DirectPath:    proto.String(uploaded.DirectPath),
			MediaKey:      uploaded.MediaKey,
			FileEncSHA256: uploaded.FileEncSHA256,
			FileSHA256:    uploaded.FileSHA256,
			FileLength:    proto.Uint64(uint64(len(videoData))),
			Mimetype:      proto.String(mimetype),
			Caption:       proto.String(caption),
			ContextInfo:   newContextInfo(evt),
		},
	}
	_, err = client.SendMessage(context.Background(), evt.Info.Chat, msg)
	return err
}

// ReplyAudio sends an audio reply.
func ReplyAudio(client *whatsmeow.Client, evt *events.Message, audioData []byte, mimetype string) error {
	SimulateTyping(client, evt.Info.Chat)
	uploaded, err := client.Upload(context.Background(), audioData, whatsmeow.MediaAudio)
	if err != nil {
		return fmt.Errorf("failed to upload audio: %w", err)
	}

	msg := &waProto.Message{
		AudioMessage: &waProto.AudioMessage{
			URL:           proto.String(uploaded.URL),
			DirectPath:    proto.String(uploaded.DirectPath),
			MediaKey:      uploaded.MediaKey,
			FileEncSHA256: uploaded.FileEncSHA256,
			FileSHA256:    uploaded.FileSHA256,
			FileLength:    proto.Uint64(uint64(len(audioData))),
			Mimetype:      proto.String(mimetype),
			ContextInfo:   newContextInfo(evt),
		},
	}
	_, err = client.SendMessage(context.Background(), evt.Info.Chat, msg)
	return err
}

// ReplySticker sends a WebP sticker reply.
func ReplySticker(client *whatsmeow.Client, evt *events.Message, stickerData []byte, animated bool) error {
	SimulateTyping(client, evt.Info.Chat)
	uploaded, err := client.Upload(context.Background(), stickerData, whatsmeow.MediaImage)
	if err != nil {
		return fmt.Errorf("failed to upload sticker: %w", err)
	}

	msg := &waProto.Message{
		StickerMessage: &waProto.StickerMessage{
			URL:           proto.String(uploaded.URL),
			DirectPath:    proto.String(uploaded.DirectPath),
			MediaKey:      uploaded.MediaKey,
			FileEncSHA256: uploaded.FileEncSHA256,
			FileSHA256:    uploaded.FileSHA256,
			FileLength:    proto.Uint64(uint64(len(stickerData))),
			Mimetype:      proto.String("image/webp"),
			IsAnimated:    proto.Bool(animated),
			ContextInfo:   newContextInfo(evt),
		},
	}
	_, err = client.SendMessage(context.Background(), evt.Info.Chat, msg)
	return err
}

// UnwrapViewOnce unwraps all View Once variants (V1, V2, V2Extension)
// and returns the inner message. Returns the original message if not View Once.
func UnwrapViewOnce(msg *waProto.Message) *waProto.Message {
	if vo := msg.GetViewOnceMessage(); vo != nil {
		return vo.GetMessage()
	}
	if vo := msg.GetViewOnceMessageV2(); vo != nil {
		return vo.GetMessage()
	}
	if vo := msg.GetViewOnceMessageV2Extension(); vo != nil {
		return vo.GetMessage()
	}
	return msg
}

// IsViewOnceMessage checks if the message is any variant of View Once.
func IsViewOnceMessage(msg *waProto.Message) bool {
	if msg.GetViewOnceMessage() != nil ||
		msg.GetViewOnceMessageV2() != nil ||
		msg.GetViewOnceMessageV2Extension() != nil {
		return true
	}
	if img := msg.GetImageMessage(); img != nil && img.GetViewOnce() {
		return true
	}
	if vid := msg.GetVideoMessage(); vid != nil && vid.GetViewOnce() {
		return true
	}
	return false
}

// DownloadMediaFromMessage downloads media bytes from a message.
func DownloadMediaFromMessage(client *whatsmeow.Client, msg *waProto.Message) ([]byte, error) {
	// Handle all View Once variants
	msg = UnwrapViewOnce(msg)

	if img := msg.GetImageMessage(); img != nil {
		return client.Download(context.Background(), img)
	}
	if vid := msg.GetVideoMessage(); vid != nil {
		return client.Download(context.Background(), vid)
	}
	if stk := msg.GetStickerMessage(); stk != nil {
		return client.Download(context.Background(), stk)
	}
	if doc := msg.GetDocumentMessage(); doc != nil {
		return client.Download(context.Background(), doc)
	}
	if aud := msg.GetAudioMessage(); aud != nil {
		return client.Download(context.Background(), aud)
	}
	return nil, fmt.Errorf("no downloadable media found in message")
}

// GetQuotedMessage returns the quoted message if the event is a reply.
func GetQuotedMessage(evt *events.Message) *waProto.Message {
	if ext := evt.Message.GetExtendedTextMessage(); ext != nil {
		if ctx := ext.GetContextInfo(); ctx != nil {
			return ctx.GetQuotedMessage()
		}
	}
	return nil
}

// IsMediaMessage checks if the message contains any media.
func IsMediaMessage(msg *waProto.Message) bool {
	// Unwrap all View Once variants
	msg = UnwrapViewOnce(msg)

	return msg.GetImageMessage() != nil ||
		msg.GetVideoMessage() != nil ||
		msg.GetStickerMessage() != nil ||
		msg.GetDocumentMessage() != nil ||
		msg.GetAudioMessage() != nil
}
