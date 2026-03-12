package services

import (
	"encoding/json"
	"log"
	"os"
	"sort"
	"strings"
	"sync"
)

// BannedStickerStore manages a global set of banned sticker SHA256 hashes.
// Hashes are stored as lowercase hex strings and persisted to a JSON file.
type BannedStickerStore struct {
	Hashes map[string]bool `json:"hashes"`
	mu     sync.RWMutex
	file   string
}

// NewBannedStickerStore creates a new store and loads data from file.
// If the file doesn't exist yet, it seeds with the provided default hashes.
func NewBannedStickerStore(file string, defaultHashes []string) *BannedStickerStore {
	store := &BannedStickerStore{
		Hashes: make(map[string]bool),
		file:   file,
	}
	store.load()

	// Seed default hashes if store is empty (first run).
	if len(store.Hashes) == 0 && len(defaultHashes) > 0 {
		store.mu.Lock()
		for _, h := range defaultHashes {
			store.Hashes[strings.ToLower(h)] = true
		}
		store.mu.Unlock()
		store.save()
	}

	return store
}

// IsBanned checks if a sticker hash is in the banned list.
func (s *BannedStickerStore) IsBanned(hash string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.Hashes[strings.ToLower(hash)]
}

// Add adds a sticker hash to the banned list. Returns true if it was newly added.
func (s *BannedStickerStore) Add(hash string) bool {
	h := strings.ToLower(hash)
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.Hashes[h] {
		return false // already banned
	}
	s.Hashes[h] = true
	s.save()
	return true
}

// Remove removes a sticker hash from the banned list. Returns true if it was found and removed.
func (s *BannedStickerStore) Remove(hash string) bool {
	h := strings.ToLower(hash)
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.Hashes[h] {
		return false
	}
	delete(s.Hashes, h)
	s.save()
	return true
}

// List returns all banned hashes sorted alphabetically.
func (s *BannedStickerStore) List() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]string, 0, len(s.Hashes))
	for h := range s.Hashes {
		result = append(result, h)
	}
	sort.Strings(result)
	return result
}

// Count returns the number of banned hashes.
func (s *BannedStickerStore) Count() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.Hashes)
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

	if err := json.Unmarshal(data, &s.Hashes); err != nil {
		log.Printf("[bannedstickers] failed to parse %s: %v", s.file, err)
	}
}

func (s *BannedStickerStore) save() {
	// Must be called with lock held.
	data, err := json.MarshalIndent(s.Hashes, "", "  ")
	if err != nil {
		log.Printf("[bannedstickers] failed to marshal: %v", err)
		return
	}
	if err := os.WriteFile(s.file, data, 0644); err != nil {
		log.Printf("[bannedstickers] failed to write %s: %v", s.file, err)
	}
}
