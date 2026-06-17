package services

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"strings"
)

// BannedStickerUserStore manages a persistent list of user JIDs who are forbidden from sending stickers, using SQLite.
// Bans are per-group: a user banned in one group is not affected in others.
type BannedStickerUserStore struct {
	db *sql.DB
}

// NewBannedStickerUserStore creates a new store and ensures the table exists.
func NewBannedStickerUserStore(db *sql.DB) *BannedStickerUserStore {
	store := &BannedStickerUserStore{db: db}

	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS banned_sticker_users (
			jid TEXT NOT NULL,
			group_jid TEXT NOT NULL,
			PRIMARY KEY (jid, group_jid)
		)
	`)
	if err != nil {
		slog.Error("Failed to create banned_sticker_users table", "error", err)
		os.Exit(1)
	}

	// Migrate old schema (jid-only PK) to new schema (jid + group_jid).
	migrateBanTable(db, "banned_sticker_users")

	store.migrateLegacyJSON()
	return store
}

// IsBanned checks if a user is banned from sending stickers in a specific group.
func (s *BannedStickerUserStore) IsBanned(jid string, groupJID string) bool {
	var count int
	err := s.db.QueryRow(`SELECT 1 FROM banned_sticker_users WHERE jid = ? AND group_jid = ?`, jid, groupJID).Scan(&count)
	return err == nil
}

// Add adds a user to the banned list for a specific group. Returns true if newly added, false if already in list.
func (s *BannedStickerUserStore) Add(jid string, groupJID string) bool {
	res, err := s.db.Exec(`INSERT OR IGNORE INTO banned_sticker_users (jid, group_jid) VALUES (?, ?)`, jid, groupJID)
	if err != nil {
		slog.Error("Error adding user", "error", err)
		return false
	}
	rows, _ := res.RowsAffected()
	return rows > 0
}

// Remove removes a user from the banned list for a specific group. Returns true if removed, false if they weren't in the list.
func (s *BannedStickerUserStore) Remove(jid string, groupJID string) bool {
	res, err := s.db.Exec(`DELETE FROM banned_sticker_users WHERE jid = ? AND group_jid = ?`, jid, groupJID)
	if err != nil {
		slog.Error("Error removing user", "error", err)
		return false
	}
	rows, _ := res.RowsAffected()
	return rows > 0
}

// Count returns the number of banned users in a specific group.
func (s *BannedStickerUserStore) Count(groupJID string) int {
	var count int
	_ = s.db.QueryRow(`SELECT COUNT(*) FROM banned_sticker_users WHERE group_jid = ?`, groupJID).Scan(&count)
	return count
}

// ListFormatted returns a formatted list of all banned users in a specific group.
func (s *BannedStickerUserStore) ListFormatted(groupJID string) string {
	rows, err := s.db.Query(`SELECT jid FROM banned_sticker_users WHERE group_jid = ? ORDER BY jid ASC`, groupJID)
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
		slog.Error("Error iterating banned sticker users", "error", err)
	}

	if len(lines) == 0 {
		return "Tidak ada user yang di-ban pengiriman stickernya."
	}
	return strings.Join(lines, "\n")
}

// migrateLegacyJSON reads banned_sticker_users.json and inserts to DB.
// Legacy entries don't have group_jid, so they are skipped (user must re-ban per group).
func (s *BannedStickerUserStore) migrateLegacyJSON() {
	legacyFile := "banned_sticker_users.json"
	data, err := os.ReadFile(legacyFile)
	if err != nil {
		return
	}

	slog.Info("[bannedstickerusers] Legacy JSON found — cannot migrate without group context. Renaming to .bak.")
	// Legacy JSON doesn't have group_jid, so we can't migrate meaningfully.
	// We just back it up. Admins need to re-ban users per group.

	var wrappedData struct {
		JIDs map[string]bool `json:"jids"`
	}
	if err := json.Unmarshal(data, &wrappedData); err == nil && wrappedData.JIDs != nil {
		count := 0
		for _, isBanned := range wrappedData.JIDs {
			if isBanned {
				count++
			}
		}
		if count > 0 {
			slog.Warn("[bannedstickerusers] Legacy JSON had entries that could not be migrated (no group context)", "count", count)
		}
	}

	_ = os.Rename(legacyFile, legacyFile+".bak")
}
