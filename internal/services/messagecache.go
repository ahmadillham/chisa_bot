package services

import (
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"time"

	waProto "go.mau.fi/whatsmeow/binary/proto"
	"google.golang.org/protobuf/proto"
)

// MessageCacheStore temporarily stores incoming messages in SQLite for retrieval by anti-delete features.
type MessageCacheStore struct {
	db *sql.DB
}

// NewMessageCacheStore creates a new message cache store and ensures the table exists.
func NewMessageCacheStore(db *sql.DB) *MessageCacheStore {
	store := &MessageCacheStore{db: db}

	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS message_cache (
			stanza_id TEXT PRIMARY KEY,
			protobuf BLOB,
			created_at INTEGER
		)
	`)
	if err != nil {
		slog.Error("Failed to create message_cache table", "error", err)
		os.Exit(1)
	}

	return store
}

// Save marshals and saves a single protobuf message (up to 24h).
func (s *MessageCacheStore) Save(stanzaID string, msg *waProto.Message) {
	if msg == nil || stanzaID == "" {
		return
	}

	bytes, err := proto.Marshal(msg)
	if err != nil {
		slog.Error("Failed to marshal message", "val", stanzaID, "error", err)
		return
	}

	now := time.Now().Unix()
	_, err = s.db.Exec(`
		INSERT OR IGNORE INTO message_cache (stanza_id, protobuf, created_at) 
		VALUES (?, ?, ?)
	`, stanzaID, bytes, now)
	if err != nil {
		slog.Error("Failed to save message", "val", stanzaID, "error", err)
	}
}

// Get retrieves a cached message by its stanza ID.
func (s *MessageCacheStore) Get(stanzaID string) (*waProto.Message, error) {
	var bytes []byte
	err := s.db.QueryRow(`SELECT protobuf FROM message_cache WHERE stanza_id = ?`, stanzaID).Scan(&bytes)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("message not found in cache")
		}
		return nil, err
	}

	msg := &waProto.Message{}
	if err := proto.Unmarshal(bytes, msg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal message: %v", err)
	}
	return msg, nil
}

// Clean deletes messages older than 24 hours.
func (s *MessageCacheStore) Clean() {
	deadline := time.Now().Add(-24 * time.Hour).Unix()
	res, err := s.db.Exec(`DELETE FROM message_cache WHERE created_at < ?`, deadline)
	if err != nil {
		slog.Error("failed to clean old messages", "error", err)
	} else if rows, _ := res.RowsAffected(); rows > 0 {
		slog.Info("deleted %d old messages", "val", rows)
	}
}
