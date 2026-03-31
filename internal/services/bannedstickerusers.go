package services

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
)

// BannedStickerUserStore manages a persistent list of user JIDs who are forbidden from sending stickers.
type BannedStickerUserStore struct {
	JIDs map[string]bool `json:"jids"` // A set of banned JIDs (without server part or just raw string)
	mu   sync.RWMutex
	file string
}

// NewBannedStickerUserStore creates a new store and loads data from file.
func NewBannedStickerUserStore(file string) *BannedStickerUserStore {
	store := &BannedStickerUserStore{
		JIDs: make(map[string]bool),
		file: file,
	}
	store.load()
	return store
}

// IsBanned checks if a user is in the banned sticker list.
func (s *BannedStickerUserStore) IsBanned(jid string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.JIDs[jid]
}

// Add adds a user to the banned list. Returns true if newly added, false if already in list.
func (s *BannedStickerUserStore) Add(jid string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.JIDs[jid] {
		return false // Already banned
	}

	s.JIDs[jid] = true
	s.save()
	return true
}

// Remove removes a user from the banned list. Returns true if removed, false if they weren't in the list.
func (s *BannedStickerUserStore) Remove(jid string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.JIDs[jid] {
		return false
	}

	delete(s.JIDs, jid)
	s.save()
	return true
}

// Count returns the number of banned users.
func (s *BannedStickerUserStore) Count() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.JIDs)
}

// ListFormatted returns a formatted list of all banned users.
func (s *BannedStickerUserStore) ListFormatted() string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if len(s.JIDs) == 0 {
		return "Tidak ada user yang di-ban pengiriman stickernya."
	}

	var lines []string
	i := 1
	for jid := range s.JIDs {
		// Clean up formatting: e.g. "62812345678" instead of "62812345678@s.whatsapp.net" potentially
		displayJID := strings.Split(jid, "@")[0]
		lines = append(lines, fmt.Sprintf("%d. @%s", i, displayJID))
		i++
	}
	return strings.Join(lines, "\n")
}

func (s *BannedStickerUserStore) load() {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := os.ReadFile(s.file)
	if err != nil {
		if !os.IsNotExist(err) {
			log.Printf("[bannedstickerusers] failed to read %s: %v", s.file, err)
		}
		return
	}

	if err := json.Unmarshal(data, &s.JIDs); err != nil {
		log.Printf("[bannedstickerusers] failed to parse %s: %v", s.file, err)
	}
	
	if s.JIDs == nil {
		s.JIDs = make(map[string]bool)
	}
}

func (s *BannedStickerUserStore) save() {
	// Must be called with lock held.
	data, err := json.MarshalIndent(s.JIDs, "", "  ")
	if err != nil {
		log.Printf("[bannedstickerusers] failed to marshal: %v", err)
		return
	}
	if err := os.WriteFile(s.file, data, 0644); err != nil {
		log.Printf("[bannedstickerusers] failed to write %s: %v", s.file, err)
	}
}
