package services

import (
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"strings"
)

// BannedImageUserStore manages a persistent list of user JIDs who are forbidden from sending image/video/GIF media, using SQLite.
// Bans are per-group: a user banned in one group is not affected in others.
type BannedImageUserStore struct {
	db *sql.DB
}

// NewBannedImageUserStore creates a new store and ensures the table exists.
func NewBannedImageUserStore(db *sql.DB) *BannedImageUserStore {
	store := &BannedImageUserStore{db: db}

	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS banned_image_users (
			jid TEXT NOT NULL,
			group_jid TEXT NOT NULL,
			PRIMARY KEY (jid, group_jid)
		)
	`)
	if err != nil {
		slog.Error("Failed to create banned_image_users table", "error", err)
		os.Exit(1)
	}

	// Migrate old schema (jid-only PK) to new schema (jid + group_jid).
	migrateBanTable(db, "banned_image_users")

	return store
}

// IsBanned checks if a user is banned from sending image/video/GIF media in a specific group.
func (s *BannedImageUserStore) IsBanned(jid string, groupJID string) bool {
	var count int
	err := s.db.QueryRow(`SELECT 1 FROM banned_image_users WHERE jid = ? AND group_jid = ?`, jid, groupJID).Scan(&count)
	return err == nil
}

// Add adds a user to the banned list for a specific group. Returns true if newly added, false if already in list.
func (s *BannedImageUserStore) Add(jid string, groupJID string) bool {
	res, err := s.db.Exec(`INSERT OR IGNORE INTO banned_image_users (jid, group_jid) VALUES (?, ?)`, jid, groupJID)
	if err != nil {
		slog.Error("Error adding user to image ban list", "error", err)
		return false
	}
	rows, _ := res.RowsAffected()
	return rows > 0
}

// Remove removes a user from the banned list for a specific group. Returns true if removed, false if they weren't in the list.
func (s *BannedImageUserStore) Remove(jid string, groupJID string) bool {
	res, err := s.db.Exec(`DELETE FROM banned_image_users WHERE jid = ? AND group_jid = ?`, jid, groupJID)
	if err != nil {
		slog.Error("Error removing user from image ban list", "error", err)
		return false
	}
	rows, _ := res.RowsAffected()
	return rows > 0
}

// Count returns the number of banned users in a specific group.
func (s *BannedImageUserStore) Count(groupJID string) int {
	var count int
	_ = s.db.QueryRow(`SELECT COUNT(*) FROM banned_image_users WHERE group_jid = ?`, groupJID).Scan(&count)
	return count
}

// ListFormatted returns a formatted list of all image/video/GIF-banned users in a specific group.
func (s *BannedImageUserStore) ListFormatted(groupJID string) string {
	rows, err := s.db.Query(`SELECT jid FROM banned_image_users WHERE group_jid = ? ORDER BY jid ASC`, groupJID)
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
		slog.Error("Error iterating banned image users", "error", err)
	}

	if len(lines) == 0 {
		return "Tidak ada user yang di-ban pengiriman gambar/video/GIF."
	}
	return strings.Join(lines, "\n")
}
