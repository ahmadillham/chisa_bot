package services

import (
	"database/sql"
	"encoding/json"
	"log"
	"os"
)

// WarnStore manages persistent warning counts for group members using SQLite.
type WarnStore struct {
	db *sql.DB
}

// NewWarnStore creates a new WarnStore and ensures the table exists.
func NewWarnStore(db *sql.DB) *WarnStore {
	store := &WarnStore{db: db}
	
	// Create table
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS warnings (
			group_jid TEXT,
			user_jid TEXT,
			count INTEGER,
			PRIMARY KEY (group_jid, user_jid)
		)
	`)
	if err != nil {
		log.Fatalf("Failed to create warnings table: %v", err)
	}

	store.migrateLegacyJSON()
	return store
}

// AddWarning increments the warning count for a user in a group.
// Returns the new count.
func (s *WarnStore) AddWarning(groupJID, userJID string) int {
	var count int
	// Insert or increment
	_, err := s.db.Exec(`
		INSERT INTO warnings (group_jid, user_jid, count) 
		VALUES (?, ?, 1)
		ON CONFLICT(group_jid, user_jid) 
		DO UPDATE SET count = warnings.count + 1
	`, groupJID, userJID)
	if err != nil {
		log.Printf("[warnstore] error adding warning: %v", err)
		return s.GetWarning(groupJID, userJID) // return current on error
	}
	
	// Fetch new count
	err = s.db.QueryRow(`SELECT count FROM warnings WHERE group_jid = ? AND user_jid = ?`, groupJID, userJID).Scan(&count)
	if err != nil {
		log.Printf("[warnstore] error getting new count: %v", err)
	}
	return count
}

// GetWarning returns the current warning count for a user.
func (s *WarnStore) GetWarning(groupJID, userJID string) int {
	var count int
	err := s.db.QueryRow(`SELECT count FROM warnings WHERE group_jid = ? AND user_jid = ?`, groupJID, userJID).Scan(&count)
	if err != nil && err != sql.ErrNoRows {
		log.Printf("[warnstore] error getting warning: %v", err)
	}
	return count
}

// ResetWarning resets the warning count for a user to 0.
func (s *WarnStore) ResetWarning(groupJID, userJID string) {
	_, err := s.db.Exec(`DELETE FROM warnings WHERE group_jid = ? AND user_jid = ?`, groupJID, userJID)
	if err != nil {
		log.Printf("[warnstore] error resetting warning: %v", err)
	}
}

// migrateLegacyJSON attempts to read warnings.json and insert them to DB if it exists.
func (s *WarnStore) migrateLegacyJSON() {
	legacyFile := "warnings.json"
	data, err := os.ReadFile(legacyFile)
	if err != nil {
		return // File doesn't exist or permissions error, ignore
	}

	log.Println("[warnstore] Running legacy JSON migration...")
	var counts map[string]map[string]int
	if err := json.Unmarshal(data, &counts); err == nil {
		for groupJID, users := range counts {
			for userJID, count := range users {
				_, _ = s.db.Exec(`
					INSERT OR IGNORE INTO warnings (group_jid, user_jid, count) 
					VALUES (?, ?, ?)
				`, groupJID, userJID, count)
			}
		}
	}
	// Rename file to prevent double migration
	_ = os.Rename(legacyFile, legacyFile+".bak")
}
