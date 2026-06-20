package services

import (
	"database/sql"
	"encoding/json"
	"log/slog"
	"os"
)

// BannedStickerUserStore manages a persistent global list of user JIDs who are forbidden from sending stickers.
type BannedStickerUserStore struct {
	db *sql.DB
}

// NewBannedStickerUserStore creates a new store and ensures the table exists.
func NewBannedStickerUserStore(db *sql.DB) *BannedStickerUserStore {
	store := &BannedStickerUserStore{db: db}

	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS banned_sticker_users (
			jid TEXT PRIMARY KEY
		)
	`)
	if err != nil {
		slog.Error("Failed to create banned_sticker_users table", "error", err)
		os.Exit(1)
	}

	if err := ensureGlobalBanTable(db, "banned_sticker_users"); err != nil {
		slog.Error("Failed to migrate banned_sticker_users table", "error", err)
		os.Exit(1)
	}

	store.migrateLegacyJSON()
	return store
}

// IsBanned checks if a user is globally banned from sending stickers.
func (s *BannedStickerUserStore) IsBanned(jid string) bool {
	var count int
	err := s.db.QueryRow(`SELECT 1 FROM banned_sticker_users WHERE jid = ?`, jid).Scan(&count)
	return err == nil
}

// Add adds a user to the global banned list. Returns true if newly added, false if already in list.
func (s *BannedStickerUserStore) Add(jid string) bool {
	res, err := s.db.Exec(`INSERT OR IGNORE INTO banned_sticker_users (jid) VALUES (?)`, jid)
	if err != nil {
		slog.Error("Error adding user", "error", err)
		return false
	}
	rows, _ := res.RowsAffected()
	return rows > 0
}

// Remove removes a user from the global banned list. Returns true if removed, false if they weren't in the list.
func (s *BannedStickerUserStore) Remove(jid string) bool {
	res, err := s.db.Exec(`DELETE FROM banned_sticker_users WHERE jid = ?`, jid)
	if err != nil {
		slog.Error("Error removing user", "error", err)
		return false
	}
	rows, _ := res.RowsAffected()
	return rows > 0
}

// migrateLegacyJSON reads banned_sticker_users.json and inserts global bans to DB.
func (s *BannedStickerUserStore) migrateLegacyJSON() {
	legacyFile := "banned_sticker_users.json"
	data, err := os.ReadFile(legacyFile)
	if err != nil {
		return
	}

	var wrappedData struct {
		JIDs map[string]bool `json:"jids"`
	}
	if err := json.Unmarshal(data, &wrappedData); err == nil && wrappedData.JIDs != nil {
		count := 0
		for jid, isBanned := range wrappedData.JIDs {
			if isBanned && s.Add(jid) {
				count++
			}
		}
		if count > 0 {
			slog.Info("[bannedstickerusers] Migrated legacy JSON bans to global DB", "count", count)
		}
	}

	_ = os.Rename(legacyFile, legacyFile+".bak")
}
