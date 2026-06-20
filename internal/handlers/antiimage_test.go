package handlers

import (
	"testing"

	waProto "go.mau.fi/whatsmeow/binary/proto"
	"google.golang.org/protobuf/proto"
)

func TestIsImageBanMedia(t *testing.T) {
	tests := []struct {
		name string
		msg  *waProto.Message
		want bool
	}{
		{
			name: "image message",
			msg:  &waProto.Message{ImageMessage: &waProto.ImageMessage{}},
			want: true,
		},
		{
			name: "video message",
			msg:  &waProto.Message{VideoMessage: &waProto.VideoMessage{}},
			want: true,
		},
		{
			name: "gif document by mimetype",
			msg: &waProto.Message{DocumentMessage: &waProto.DocumentMessage{
				Mimetype: proto.String("image/gif"),
			}},
			want: true,
		},
		{
			name: "video document by mimetype",
			msg: &waProto.Message{DocumentMessage: &waProto.DocumentMessage{
				Mimetype: proto.String("video/mp4"),
			}},
			want: true,
		},
		{
			name: "gif document by filename",
			msg: &waProto.Message{DocumentMessage: &waProto.DocumentMessage{
				Mimetype: proto.String("application/octet-stream"),
				FileName: proto.String("reaction.gif"),
			}},
			want: true,
		},
		{
			name: "video document by filename",
			msg: &waProto.Message{DocumentMessage: &waProto.DocumentMessage{
				Mimetype: proto.String("application/octet-stream"),
				FileName: proto.String("clip.webm"),
			}},
			want: true,
		},
		{
			name: "non-media document",
			msg: &waProto.Message{DocumentMessage: &waProto.DocumentMessage{
				Mimetype: proto.String("application/pdf"),
				FileName: proto.String("notes.pdf"),
			}},
			want: false,
		},
		{
			name: "text message",
			msg:  &waProto.Message{Conversation: proto.String("hello")},
			want: false,
		},
		{
			name: "nil message",
			msg:  nil,
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isImageBanMedia(tt.msg); got != tt.want {
				t.Fatalf("isImageBanMedia() = %v, want %v", got, tt.want)
			}
		})
	}
}
