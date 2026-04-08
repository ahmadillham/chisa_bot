package services

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
)

// BannedStickerEntry holds a hash and its alias.
type BannedStickerEntry struct {
	Hash  string `json:"hash"`
	Alias string `json:"alias"`
}

// BannedStickerStore manages a global set of banned sticker SHA256 hashes with SQLite.
type BannedStickerStore struct {
	db *sql.DB
}

// NewBannedStickerStore creates a new BannedStickerStore and ensures the table exists.
func NewBannedStickerStore(db *sql.DB, defaults []BannedStickerEntry) *BannedStickerStore {
	store := &BannedStickerStore{db: db}
	
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS banned_stickers (
			hash TEXT PRIMARY KEY,
			alias TEXT NOT NULL
		)
	`)
	if err != nil {
		log.Fatalf("Failed to create banned_stickers table: %v", err)
	}

	store.migrateLegacyJSON()
	
	// Seed defaults if empty
	for _, entry := range defaults {
		h := strings.ToLower(entry.Hash)
		_, _ = db.Exec(`INSERT OR IGNORE INTO banned_stickers (hash, alias) VALUES (?, ?)`, h, entry.Alias)
	}

	return store
}

// IsBanned checks if a sticker hash is in the banned list.
func (s *BannedStickerStore) IsBanned(hash string) bool {
	h := strings.ToLower(hash)
	var count int
	err := s.db.QueryRow(`SELECT 1 FROM banned_stickers WHERE hash = ?`, h).Scan(&count)
	return err == nil
}

// Add adds a sticker hash with an alias. Auto-generates alias if empty.
// Returns the alias used and true if newly added.
func (s *BannedStickerStore) Add(hash string, alias string) (string, bool) {
	h := strings.ToLower(hash)
	
	var existingAlias string
	err := s.db.QueryRow(`SELECT alias FROM banned_stickers WHERE hash = ?`, h).Scan(&existingAlias)
	if err == nil {
		return existingAlias, false // Already exists
	}

	if alias == "" {
		alias = fmt.Sprintf("sticker_%s", h[:8])
	}

	_, err = s.db.Exec(`INSERT INTO banned_stickers (hash, alias) VALUES (?, ?)`, h, alias)
	if err != nil {
		log.Printf("[bannedstickers] Error adding: %v", err)
	}
	return alias, true
}

// Remove removes a banned sticker by alias OR hash. Returns true if found and removed.
func (s *BannedStickerStore) Remove(identifier string) bool {
	id := strings.ToLower(identifier)
	res, err := s.db.Exec(`DELETE FROM banned_stickers WHERE hash = ? OR LOWER(alias) = ?`, id, id)
	if err != nil {
		log.Printf("[bannedstickers] Error removing: %v", err)
		return false
	}
	rows, _ := res.RowsAffected()
	return rows > 0
}

// Count returns the number of banned hashes.
func (s *BannedStickerStore) Count() int {
	var count int
	_ = s.db.QueryRow(`SELECT COUNT(*) FROM banned_stickers`).Scan(&count)
	return count
}

// ListFormatted returns a formatted list of all banned stickers.
func (s *BannedStickerStore) ListFormatted() string {
	rows, err := s.db.Query(`SELECT hash, alias FROM banned_stickers ORDER BY alias ASC`)
	if err != nil {
		return "Error membaca database sticker."
	}
	defer rows.Close()

	var lines []string
	i := 1
	for rows.Next() {
		var hash, alias string
		if err := rows.Scan(&hash, &alias); err == nil {
			displayHash := hash
			if len(hash) > 16 {
				displayHash = hash[:16] + "..."
			}
			lines = append(lines, fmt.Sprintf("%d. %s — `%s`", i, alias, displayHash))
			i++
		}
	}

	if len(lines) == 0 {
		return "Tidak ada sticker yang di-ban."
	}
	return strings.Join(lines, "\n")
}

// migrateLegacyJSON reads banned_stickers.json and inserts to DB.
func (s *BannedStickerStore) migrateLegacyJSON() {
	legacyFile := "banned_stickers.json"
	data, err := os.ReadFile(legacyFile)
	if err != nil {
		return
	}

	log.Println("[bannedstickers] Running legacy JSON migration...")
	
	// Try standard list format
	var legacyData []BannedStickerEntry
	if err := json.Unmarshal(data, &legacyData); err == nil {
		for _, entry := range legacyData {
			h := strings.ToLower(entry.Hash)
			_, _ = s.db.Exec(`INSERT OR IGNORE INTO banned_stickers (hash, alias) VALUES (?, ?)`, h, entry.Alias)
		}
	} else {
		// Try wrapper format {"entries": [...]}
		var wrappedData struct {
			Entries []BannedStickerEntry `json:"entries"`
		}
		if err := json.Unmarshal(data, &wrappedData); err == nil {
			for _, entry := range wrappedData.Entries {
				h := strings.ToLower(entry.Hash)
				_, _ = s.db.Exec(`INSERT OR IGNORE INTO banned_stickers (hash, alias) VALUES (?, ?)`, h, entry.Alias)
			}
		}
	}
	_ = os.Rename(legacyFile, legacyFile+".bak")
}
