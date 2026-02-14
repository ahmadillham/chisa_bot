package services

import (
	"encoding/json"
	"log"
	"os"
	"sync"
	"time"
)

type LeaderboardData struct {
	LastReset time.Time      `json:"last_reset"`
	Scores    map[string]int `json:"scores"` // UserJID -> Score
}

type GameStore struct {
	filePath string
	mu       sync.Mutex
	Data     LeaderboardData
}

func NewGameStore(filePath string) *GameStore {
	store := &GameStore{
		filePath: filePath,
		Data: LeaderboardData{
			LastReset: time.Now(),
			Scores:    make(map[string]int),
		},
	}
	// Ensure directory exists
	if err := os.MkdirAll("data", 0755); err != nil {
		log.Printf("[gamestore] failed to create data directory: %v", err)
	}
	store.load()
	go store.autoSaveRoutine()
	return store
}

func (s *GameStore) load() {
	s.mu.Lock()
	defer s.mu.Unlock()

	file, err := os.ReadFile(s.filePath)
	if err != nil {
		if !os.IsNotExist(err) {
			log.Printf("[gamestore] failed to read file: %v", err)
		}
		return
	}

	if err := json.Unmarshal(file, &s.Data); err != nil {
		log.Printf("[gamestore] failed to unmarshal: %v", err)
		return
	}

	// Check for weekly reset
	if time.Since(s.Data.LastReset).Hours() > 24*7 {
		log.Println("[gamestore] Weekly reset triggered.")
		s.Data.Scores = make(map[string]int)
		s.Data.LastReset = time.Now()
		s.save()
	}
}

func (s *GameStore) save() {
	// Must be called with lock held or ensure safety
	data, err := json.MarshalIndent(s.Data, "", "  ")
	if err != nil {
		log.Printf("[gamestore] failed to marshal: %v", err)
		return
	}
	if err := os.WriteFile(s.filePath, data, 0644); err != nil {
		log.Printf("[gamestore] failed to write file: %v", err)
	}
}

func (s *GameStore) autoSaveRoutine() {
	for {
		time.Sleep(5 * time.Minute)
		s.mu.Lock()
		// Check reset again periodically
		if time.Since(s.Data.LastReset).Hours() > 24*7 {
			log.Println("[gamestore] Weekly reset triggered.")
			s.Data.Scores = make(map[string]int)
			s.Data.LastReset = time.Now()
		}
		s.save()
		s.mu.Unlock()
	}
}

func (s *GameStore) AddScore(userJID string, points int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Data.Scores[userJID] += points
	s.save() // Save immediately on score update to prevent data loss on crash
}

func (s *GameStore) GetLeaderboard() map[string]int {
	s.mu.Lock()
	defer s.mu.Unlock()
	// Return copy
	copy := make(map[string]int)
	for k, v := range s.Data.Scores {
		copy[k] = v
	}
	return copy
}
