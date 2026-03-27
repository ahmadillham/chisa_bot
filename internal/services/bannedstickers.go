package services

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
)

// BannedStickerEntry holds a hash and its alias.
type BannedStickerEntry struct {
	Hash  string `json:"hash"`
	Alias string `json:"alias"`
}

// BannedStickerStore manages a global set of banned sticker SHA256 hashes with aliases.
type BannedStickerStore struct {
	Entries []BannedStickerEntry `json:"entries"`
	mu      sync.RWMutex
	file    string
}

// NewBannedStickerStore creates a new store and loads data from file.
// If the file doesn't exist yet, it seeds with the provided default hashes.
func NewBannedStickerStore(file string, defaults []BannedStickerEntry) *BannedStickerStore {
	store := &BannedStickerStore{
		Entries: []BannedStickerEntry{},
		file:    file,
	}
	store.load()

	// Seed defaults if store is empty (first run).
	if len(store.Entries) == 0 && len(defaults) > 0 {
		store.mu.Lock()
		store.Entries = append(store.Entries, defaults...)
		store.mu.Unlock()
		store.save()
	}

	return store
}

// IsBanned checks if a sticker hash is in the banned list.
func (s *BannedStickerStore) IsBanned(hash string) bool {
	h := strings.ToLower(hash)
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, e := range s.Entries {
		if e.Hash == h {
			return true
		}
	}
	return false
}

// Add adds a sticker hash with an alias. Auto-generates alias if empty.
// Returns the alias used and true if newly added.
func (s *BannedStickerStore) Add(hash string, alias string) (string, bool) {
	h := strings.ToLower(hash)
	s.mu.Lock()
	defer s.mu.Unlock()

	// Check if already exists.
	for _, e := range s.Entries {
		if e.Hash == h {
			return e.Alias, false
		}
	}

	// Auto-generate alias if not provided.
	if alias == "" {
		alias = fmt.Sprintf("sticker%d", len(s.Entries)+1)
	}

	s.Entries = append(s.Entries, BannedStickerEntry{Hash: h, Alias: alias})
	s.save()
	return alias, true
}

// Remove removes a banned sticker by alias OR hash. Returns true if found and removed.
func (s *BannedStickerStore) Remove(identifier string) bool {
	id := strings.ToLower(identifier)
	s.mu.Lock()
	defer s.mu.Unlock()

	for i, e := range s.Entries {
		if strings.ToLower(e.Alias) == id || e.Hash == id {
			s.Entries = append(s.Entries[:i], s.Entries[i+1:]...)
			s.save()
			return true
		}
	}
	return false
}

// Count returns the number of banned hashes.
func (s *BannedStickerStore) Count() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.Entries)
}

// ListFormatted returns a formatted list of all banned stickers.
func (s *BannedStickerStore) ListFormatted() string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if len(s.Entries) == 0 {
		return "Tidak ada sticker yang di-ban."
	}

	var lines []string
	for i, e := range s.Entries {
		lines = append(lines, fmt.Sprintf("%d. %s — `%s`", i+1, e.Alias, e.Hash[:16]+"..."))
	}
	return strings.Join(lines, "\n")
}

func (s *BannedStickerStore) load() {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := os.ReadFile(s.file)
	if err != nil {
		if !os.IsNotExist(err) {
			log.Printf("[bannedstickers] failed to read %s: %v", s.file, err)
		}
		return
	}

	if err := json.Unmarshal(data, &s.Entries); err != nil {
		log.Printf("[bannedstickers] failed to parse %s: %v", s.file, err)
	}
}

func (s *BannedStickerStore) save() {
	// Must be called with lock held.
	data, err := json.MarshalIndent(s.Entries, "", "  ")
	if err != nil {
		log.Printf("[bannedstickers] failed to marshal: %v", err)
		return
	}
	if err := os.WriteFile(s.file, data, 0644); err != nil {
		log.Printf("[bannedstickers] failed to write %s: %v", s.file, err)
	}
}
