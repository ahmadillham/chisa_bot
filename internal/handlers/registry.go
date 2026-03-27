package handlers

import (
	"fmt"
	"strings"
	"sync"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types/events"
)

// CommandHandler is a function type for handling commands.
type CommandHandler func(client *whatsmeow.Client, evt *events.Message, args []string)

// Registry manages command handlers.
type Registry struct {
	handlers map[string]CommandHandler
	mu       sync.RWMutex
}

// NewRegistry creates a new Registry.
func NewRegistry() *Registry {
	return &Registry{
		handlers: make(map[string]CommandHandler),
	}
}

// Register adds a new command handler.
func (r *Registry) Register(command string, handler CommandHandler) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.handlers[strings.ToLower(command)] = handler
}

// Execute runs the handler for a given command.
func (r *Registry) Execute(client *whatsmeow.Client, evt *events.Message, command string, args []string) bool {
	r.mu.RLock()
	handler, exists := r.handlers[strings.ToLower(command)]
	r.mu.RUnlock()

	if !exists {
		return false
	}

	// Recover from panics in handlers to prevent crash
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("[PANIC] Command %s: %v\n", command, r)
		}
	}()

	handler(client, evt, args)
	return true
}
