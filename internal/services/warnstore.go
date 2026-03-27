package services

import (
	"encoding/json"
	"os"
	"sync"
)

// WarnStore manages persistent warning counts for group members.
type WarnStore struct {
	// Map: GroupJID -> UserJID -> Count
	Counts map[string]map[string]int
	mu     sync.RWMutex
	file   string
}

// NewWarnStore creates a new WarnStore and loads data from file.
func NewWarnStore(file string) *WarnStore {
	store := &WarnStore{
		Counts: make(map[string]map[string]int),
		file:   file,
	}
	store.load()
	return store
}

// AddWarning increments the warning count for a user in a group.
// Returns the new count.
func (s *WarnStore) AddWarning(groupJID, userJID string) int {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.Counts[groupJID] == nil {
		s.Counts[groupJID] = make(map[string]int)
	}
	s.Counts[groupJID][userJID]++
	count := s.Counts[groupJID][userJID]
	s.save()
	return count
}

// GetWarning returns the current warning count for a user.
func (s *WarnStore) GetWarning(groupJID, userJID string) int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if group, ok := s.Counts[groupJID]; ok {
		return group[userJID]
	}
	return 0
}

// ResetWarning resets the warning count for a user to 0.
func (s *WarnStore) ResetWarning(groupJID, userJID string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.Counts[groupJID] != nil {
		delete(s.Counts[groupJID], userJID)
		if len(s.Counts[groupJID]) == 0 {
			delete(s.Counts, groupJID)
		}
		s.save()
	}
}

func (s *WarnStore) load() {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := os.ReadFile(s.file)
	if err != nil {
		if !os.IsNotExist(err) {
			// Log error via standard log if possible, but package assumes stdlib log usage elsewhere.
		}
		return
	}

	if err := json.Unmarshal(data, &s.Counts); err != nil {
		// handle error
	}
}

func (s *WarnStore) save() {
	// Must be called with lock held
	data, err := json.MarshalIndent(s.Counts, "", "  ")
	if err != nil {
		return
	}
	_ = os.WriteFile(s.file, data, 0644)
}
