package services

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
)

// BannedStickerUserStore manages a persistent list of user JIDs who are forbidden from sending stickers, using SQLite.
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
		log.Fatalf("Failed to create banned_sticker_users table: %v", err)
	}

	store.migrateLegacyJSON()
	return store
}

// IsBanned checks if a user is in the banned sticker list.
func (s *BannedStickerUserStore) IsBanned(jid string) bool {
	var count int
	err := s.db.QueryRow(`SELECT 1 FROM banned_sticker_users WHERE jid = ?`, jid).Scan(&count)
	return err == nil
}

// Add adds a user to the banned list. Returns true if newly added, false if already in list.
func (s *BannedStickerUserStore) Add(jid string) bool {
	res, err := s.db.Exec(`INSERT OR IGNORE INTO banned_sticker_users (jid) VALUES (?)`, jid)
	if err != nil {
		log.Printf("[bannedstickerusers] Error adding user: %v", err)
		return false
	}
	rows, _ := res.RowsAffected()
	return rows > 0
}

// Remove removes a user from the banned list. Returns true if removed, false if they weren't in the list.
func (s *BannedStickerUserStore) Remove(jid string) bool {
	res, err := s.db.Exec(`DELETE FROM banned_sticker_users WHERE jid = ?`, jid)
	if err != nil {
		log.Printf("[bannedstickerusers] Error removing user: %v", err)
		return false
	}
	rows, _ := res.RowsAffected()
	return rows > 0
}

// Count returns the number of banned users.
func (s *BannedStickerUserStore) Count() int {
	var count int
	_ = s.db.QueryRow(`SELECT COUNT(*) FROM banned_sticker_users`).Scan(&count)
	return count
}

// ListFormatted returns a formatted list of all banned users.
func (s *BannedStickerUserStore) ListFormatted() string {
	rows, err := s.db.Query(`SELECT jid FROM banned_sticker_users ORDER BY jid ASC`)
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
		return "Tidak ada user yang di-ban pengiriman stickernya."
	}
	return strings.Join(lines, "\n")
}

// migrateLegacyJSON reads banned_sticker_users.json and inserts to DB.
func (s *BannedStickerUserStore) migrateLegacyJSON() {
	legacyFile := "banned_sticker_users.json"
	data, err := os.ReadFile(legacyFile)
	if err != nil {
		return
	}

	log.Println("[bannedstickerusers] Running legacy JSON migration...")
	var wrappedData struct {
		JIDs map[string]bool `json:"jids"`
	}
	
	if err := json.Unmarshal(data, &wrappedData); err == nil && wrappedData.JIDs != nil {
		for jid, isBanned := range wrappedData.JIDs {
			if isBanned {
				_, _ = s.db.Exec(`INSERT OR IGNORE INTO banned_sticker_users (jid) VALUES (?)`, jid)
			}
		}
	} else {
		var flatMap map[string]bool
		if err := json.Unmarshal(data, &flatMap); err == nil {
			for jid, isBanned := range flatMap {
				if isBanned {
					_, _ = s.db.Exec(`INSERT OR IGNORE INTO banned_sticker_users (jid) VALUES (?)`, jid)
				}
			}
		}
	}
	_ = os.Rename(legacyFile, legacyFile+".bak")
}
