package services

import (
	"database/sql"
	"fmt"
	"log"
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
		log.Fatalf("Failed to create message_cache table: %v", err)
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
		log.Printf("[messagecache] Failed to marshal message %s: %v", stanzaID, err)
		return
	}

	now := time.Now().Unix()
	_, err = s.db.Exec(`
		INSERT OR IGNORE INTO message_cache (stanza_id, protobuf, created_at) 
		VALUES (?, ?, ?)
	`, stanzaID, bytes, now)
	if err != nil {
		log.Printf("[messagecache] Failed to save message %s: %v", stanzaID, err)
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
		log.Printf("[messagecache] failed to clean old messages: %v", err)
	} else if rows, _ := res.RowsAffected(); rows > 0 {
		log.Printf("[messagecache] deleted %d old messages", rows)
	}
}
