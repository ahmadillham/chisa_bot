package services

import (
	"database/sql"
	"log/slog"
	"os"
)

// BannedChatUserStore manages a persistent global list of user JIDs who are forbidden from sending any chat messages.
type BannedChatUserStore struct {
	db *sql.DB
}

// NewBannedChatUserStore creates a new store and ensures the table exists.
func NewBannedChatUserStore(db *sql.DB) *BannedChatUserStore {
	store := &BannedChatUserStore{db: db}

	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS banned_chat_users (
			jid TEXT PRIMARY KEY
		)
	`)
	if err != nil {
		slog.Error("Failed to create banned_chat_users table", "error", err)
		os.Exit(1)
	}

	if err := ensureGlobalBanTable(db, "banned_chat_users"); err != nil {
		slog.Error("Failed to migrate banned_chat_users table", "error", err)
		os.Exit(1)
	}

	return store
}

// IsBanned checks if a user is globally banned from chatting.
func (s *BannedChatUserStore) IsBanned(jid string) bool {
	var count int
	err := s.db.QueryRow(`SELECT 1 FROM banned_chat_users WHERE jid = ?`, jid).Scan(&count)
	return err == nil
}

// Add adds a user to the global banned list. Returns true if newly added, false if already in list.
func (s *BannedChatUserStore) Add(jid string) bool {
	res, err := s.db.Exec(`INSERT OR IGNORE INTO banned_chat_users (jid) VALUES (?)`, jid)
	if err != nil {
		slog.Error("Error adding user to chat ban list", "error", err)
		return false
	}
	rows, _ := res.RowsAffected()
	return rows > 0
}

// Remove removes a user from the global banned list. Returns true if removed, false if they weren't in the list.
func (s *BannedChatUserStore) Remove(jid string) bool {
	res, err := s.db.Exec(`DELETE FROM banned_chat_users WHERE jid = ?`, jid)
	if err != nil {
		slog.Error("Error removing user from chat ban list", "error", err)
		return false
	}
	rows, _ := res.RowsAffected()
	return rows > 0
}
