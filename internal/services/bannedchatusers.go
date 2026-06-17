package services

import (
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"strings"
)

// BannedChatUserStore manages a persistent list of user JIDs who are forbidden from sending any chat messages, using SQLite.
// Bans are per-group: a user banned in one group is not affected in others.
type BannedChatUserStore struct {
	db *sql.DB
}

// NewBannedChatUserStore creates a new store and ensures the table exists.
func NewBannedChatUserStore(db *sql.DB) *BannedChatUserStore {
	store := &BannedChatUserStore{db: db}

	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS banned_chat_users (
			jid TEXT NOT NULL,
			group_jid TEXT NOT NULL,
			PRIMARY KEY (jid, group_jid)
		)
	`)
	if err != nil {
		slog.Error("Failed to create banned_chat_users table", "error", err)
		os.Exit(1)
	}

	// Migrate old schema (jid-only PK) to new schema (jid + group_jid).
	migrateBanTable(db, "banned_chat_users")

	return store
}

// IsBanned checks if a user is banned from chatting in a specific group.
func (s *BannedChatUserStore) IsBanned(jid string, groupJID string) bool {
	var count int
	err := s.db.QueryRow(`SELECT 1 FROM banned_chat_users WHERE jid = ? AND group_jid = ?`, jid, groupJID).Scan(&count)
	return err == nil
}

// Add adds a user to the banned list for a specific group. Returns true if newly added, false if already in list.
func (s *BannedChatUserStore) Add(jid string, groupJID string) bool {
	res, err := s.db.Exec(`INSERT OR IGNORE INTO banned_chat_users (jid, group_jid) VALUES (?, ?)`, jid, groupJID)
	if err != nil {
		slog.Error("Error adding user to chat ban list", "error", err)
		return false
	}
	rows, _ := res.RowsAffected()
	return rows > 0
}

// Remove removes a user from the banned list for a specific group. Returns true if removed, false if they weren't in the list.
func (s *BannedChatUserStore) Remove(jid string, groupJID string) bool {
	res, err := s.db.Exec(`DELETE FROM banned_chat_users WHERE jid = ? AND group_jid = ?`, jid, groupJID)
	if err != nil {
		slog.Error("Error removing user from chat ban list", "error", err)
		return false
	}
	rows, _ := res.RowsAffected()
	return rows > 0
}

// Count returns the number of banned users in a specific group.
func (s *BannedChatUserStore) Count(groupJID string) int {
	var count int
	_ = s.db.QueryRow(`SELECT COUNT(*) FROM banned_chat_users WHERE group_jid = ?`, groupJID).Scan(&count)
	return count
}

// ListFormatted returns a formatted list of all banned users in a specific group.
func (s *BannedChatUserStore) ListFormatted(groupJID string) string {
	rows, err := s.db.Query(`SELECT jid FROM banned_chat_users WHERE group_jid = ? ORDER BY jid ASC`, groupJID)
	if err != nil {
		return "Error membaca database user."
	}
	defer rows.Close()

	var lines []string
	i := 1
	for rows.Next() {
		var jid string
		if err := rows.Scan(&jid); err == nil {
			displayJID := strings.Split(jid, "@")[0]
			lines = append(lines, fmt.Sprintf("%d. @%s", i, displayJID))
			i++
		}
	}
	if err := rows.Err(); err != nil {
		slog.Error("Error iterating banned chat users", "error", err)
	}

	if len(lines) == 0 {
		return "Tidak ada user yang di-ban pengiriman chatnya."
	}
	return strings.Join(lines, "\n")
}
