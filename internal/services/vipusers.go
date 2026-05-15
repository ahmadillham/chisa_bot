package services

import (
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"strings"
)

// VIPUserStore manages a persistent list of VIP users who are immune to kicks and bans.
type VIPUserStore struct {
	db *sql.DB
}

// NewVIPUserStore creates a new store and ensures the table exists.
func NewVIPUserStore(db *sql.DB) *VIPUserStore {
	store := &VIPUserStore{db: db}

	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS vip_users (
			jid TEXT PRIMARY KEY
		)
	`)
	if err != nil {
		slog.Error("Failed to create vip_users table", "error", err)
		os.Exit(1)
	}

	return store
}

// IsVIP checks if a user is in the VIP list.
func (s *VIPUserStore) IsVIP(jid string) bool {
	var count int
	err := s.db.QueryRow(`SELECT 1 FROM vip_users WHERE jid = ?`, jid).Scan(&count)
	return err == nil
}

// Add adds a user to the VIP list. Returns true if newly added, false if already in list.
func (s *VIPUserStore) Add(jid string) bool {
	res, err := s.db.Exec(`INSERT OR IGNORE INTO vip_users (jid) VALUES (?)`, jid)
	if err != nil {
		slog.Error("Error adding VIP user", "error", err)
		return false
	}
	rows, _ := res.RowsAffected()
	return rows > 0
}

// Remove removes a user from the VIP list. Returns true if removed, false if they weren't in the list.
func (s *VIPUserStore) Remove(jid string) bool {
	res, err := s.db.Exec(`DELETE FROM vip_users WHERE jid = ?`, jid)
	if err != nil {
		slog.Error("Error removing VIP user", "error", err)
		return false
	}
	rows, _ := res.RowsAffected()
	return rows > 0
}

// ListFormatted returns a formatted list of all VIP users.
func (s *VIPUserStore) ListFormatted() string {
	rows, err := s.db.Query(`SELECT jid FROM vip_users ORDER BY jid ASC`)
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
		return "Tidak ada user VIP."
	}
	return strings.Join(lines, "\n")
}
