package services

import (
	"database/sql"
	"encoding/json"
	"log/slog"
	"os"
)

// AutoTagStore manages persistent auto-tag preferences for groups using SQLite.
type AutoTagStore struct {
	db *sql.DB
}

// NewAutoTagStore creates a new AutoTagStore and ensures the table exists.
func NewAutoTagStore(db *sql.DB) *AutoTagStore {
	store := &AutoTagStore{db: db}

	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS autotag_prefs (
			group_jid TEXT PRIMARY KEY,
			disabled BOOLEAN
		)
	`)
	if err != nil {
		slog.Error("Failed to create autotag_prefs table", "error", err)
		os.Exit(1)
	}

	store.migrateLegacyJSON()
	return store
}

// IsDisabled checks if auto-tag is disabled for a specific group.
// Returns true if disabled, false if enabled (default).
func (s *AutoTagStore) IsDisabled(groupJID string) bool {
	var disabled bool
	err := s.db.QueryRow(`SELECT disabled FROM autotag_prefs WHERE group_jid = ?`, groupJID).Scan(&disabled)
	if err != nil && err != sql.ErrNoRows {
		slog.Error("error getting pref", "error", err)
	}
	// If no rows, default is false (enabled)
	return disabled
}

// SetDisabled updates the auto-tag preference for a specific group.
func (s *AutoTagStore) SetDisabled(groupJID string, disabled bool) {
	if disabled {
		_, err := s.db.Exec(`
			INSERT INTO autotag_prefs (group_jid, disabled) 
			VALUES (?, 1)
			ON CONFLICT(group_jid) DO UPDATE SET disabled = 1
		`, groupJID)
		if err != nil {
			slog.Error("error setting pref", "error", err)
		}
	} else {
		_, err := s.db.Exec(`DELETE FROM autotag_prefs WHERE group_jid = ?`, groupJID)
		if err != nil {
			slog.Error("error removing pref", "error", err)
		}
	}
}

// migrateLegacyJSON attempts to read autotag.json and insert them to DB.
func (s *AutoTagStore) migrateLegacyJSON() {
	legacyFile := "autotag.json"
	data, err := os.ReadFile(legacyFile)
	if err != nil {
		return
	}

	slog.Info("[autotagstore] Running legacy JSON migration...")
	var disabledGroups map[string]bool
	if err := json.Unmarshal(data, &disabledGroups); err == nil {
		for groupJID, disabled := range disabledGroups {
			if disabled {
				_, _ = s.db.Exec(`
					INSERT OR IGNORE INTO autotag_prefs (group_jid, disabled) 
					VALUES (?, 1)
				`, groupJID)
			}
		}
	}
	_ = os.Rename(legacyFile, legacyFile+".bak")
}
