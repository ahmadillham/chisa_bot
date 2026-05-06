package services

import (
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"strings"
)

// BannedImageUserStore manages a persistent list of user JIDs who are forbidden from sending images, using SQLite.
type BannedImageUserStore struct {
	db *sql.DB
}

// NewBannedImageUserStore creates a new store and ensures the table exists.
func NewBannedImageUserStore(db *sql.DB) *BannedImageUserStore {
	store := &BannedImageUserStore{db: db}

	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS banned_image_users (
			jid TEXT PRIMARY KEY
		)
	`)
	if err != nil {
		slog.Error("Failed to create banned_image_users table", "error", err)
		os.Exit(1)
	}

	return store
}

// IsBanned checks if a user is in the banned image list.
func (s *BannedImageUserStore) IsBanned(jid string) bool {
	var count int
	err := s.db.QueryRow(`SELECT 1 FROM banned_image_users WHERE jid = ?`, jid).Scan(&count)
	return err == nil
}

// Add adds a user to the banned list. Returns true if newly added, false if already in list.
func (s *BannedImageUserStore) Add(jid string) bool {
	res, err := s.db.Exec(`INSERT OR IGNORE INTO banned_image_users (jid) VALUES (?)`, jid)
	if err != nil {
		slog.Error("Error adding user to image ban list", "error", err)
		return false
	}
	rows, _ := res.RowsAffected()
	return rows > 0
}

// Remove removes a user from the banned list. Returns true if removed, false if they weren't in the list.
func (s *BannedImageUserStore) Remove(jid string) bool {
	res, err := s.db.Exec(`DELETE FROM banned_image_users WHERE jid = ?`, jid)
	if err != nil {
		slog.Error("Error removing user from image ban list", "error", err)
		return false
	}
	rows, _ := res.RowsAffected()
	return rows > 0
}

// Count returns the number of banned users.
func (s *BannedImageUserStore) Count() int {
	var count int
	_ = s.db.QueryRow(`SELECT COUNT(*) FROM banned_image_users`).Scan(&count)
	return count
}

// ListFormatted returns a formatted list of all banned users.
func (s *BannedImageUserStore) ListFormatted() string {
	rows, err := s.db.Query(`SELECT jid FROM banned_image_users ORDER BY jid ASC`)
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

	if len(lines) == 0 {
		return "Tidak ada user yang di-ban pengiriman gambarnya."
	}
	return strings.Join(lines, "\n")
}
