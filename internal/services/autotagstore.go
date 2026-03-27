package services

import (
	"encoding/json"
	"os"
	"sync"
)

// AutoTagStore manages persistent auto-tag preferences for groups.
type AutoTagStore struct {
	// Map: GroupJID -> IsDisabled (true means disabled)
	DisabledGroups map[string]bool
	mu             sync.RWMutex
	file           string
}

// NewAutoTagStore creates a new AutoTagStore and loads data from file.
func NewAutoTagStore(file string) *AutoTagStore {
	store := &AutoTagStore{
		DisabledGroups: make(map[string]bool),
		file:           file,
	}
	store.load()
	return store
}

// IsDisabled checks if auto-tag is disabled for a specific group.
// Returns true if disabled, false if enabled (default).
func (s *AutoTagStore) IsDisabled(groupJID string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.DisabledGroups[groupJID]
}

// SetDisabled updates the auto-tag preference for a specific group.
func (s *AutoTagStore) SetDisabled(groupJID string, disabled bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if disabled {
		s.DisabledGroups[groupJID] = true
	} else {
		delete(s.DisabledGroups, groupJID)
	}
	s.save()
}

func (s *AutoTagStore) load() {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := os.ReadFile(s.file)
	if err != nil {
		return
	}

	_ = json.Unmarshal(data, &s.DisabledGroups)
}

func (s *AutoTagStore) save() {
	// Must be called with lock held
	data, err := json.MarshalIndent(s.DisabledGroups, "", "  ")
	if err != nil {
		return
	}
	_ = os.WriteFile(s.file, data, 0644)
}
