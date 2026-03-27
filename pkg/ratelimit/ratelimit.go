package ratelimit

import (
	"sync"
	"time"
)

// Limiter provides per-user and per-chat rate limiting.
type Limiter struct {
	mu sync.Mutex

	// Per-user: tracks last command time per user JID.
	userLast map[string]time.Time
	userCooldown time.Duration

	// Per-chat: tracks command timestamps within the sliding window.
	chatCmds   map[string][]time.Time
	chatLimit  int
	chatWindow time.Duration

	// Cleanup interval.
	lastCleanup time.Time
}

// New creates a new Limiter.
// userCooldown: minimum time between commands for a single user.
// chatLimit: max commands per chatWindow per chat.
func New(userCooldown time.Duration, chatLimit int, chatWindow time.Duration) *Limiter {
	return &Limiter{
		userLast:     make(map[string]time.Time),
		userCooldown: userCooldown,
		chatCmds:     make(map[string][]time.Time),
		chatLimit:    chatLimit,
		chatWindow:   chatWindow,
		lastCleanup:  time.Now(),
	}
}

// Result describes why a request was denied.
type Result int

const (
	Allowed       Result = iota
	UserCooldown         // user is sending too fast
	ChatRateLimit        // chat has too many commands this minute
)

// Check tests whether a command from userJID in chatJID is allowed.
// Returns Allowed if OK, or the reason it was denied.
func (l *Limiter) Check(userJID, chatJID string) Result {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()

	// Periodic cleanup every 5 minutes to free memory.
	if now.Sub(l.lastCleanup) > 5*time.Minute {
		l.cleanup(now)
		l.lastCleanup = now
	}

	// 1. Per-user cooldown check.
	if last, ok := l.userLast[userJID]; ok {
		if now.Sub(last) < l.userCooldown {
			return UserCooldown
		}
	}

	// 2. Per-chat rate limit check (sliding window).
	chatKey := chatJID
	cmds := l.chatCmds[chatKey]

	// Remove timestamps outside the window.
	cutoff := now.Add(-l.chatWindow)
	start := 0
	for start < len(cmds) && cmds[start].Before(cutoff) {
		start++
	}
	cmds = cmds[start:]

	if len(cmds) >= l.chatLimit {
		l.chatCmds[chatKey] = cmds
		return ChatRateLimit
	}

	// Allowed — record this command.
	l.userLast[userJID] = now
	l.chatCmds[chatKey] = append(cmds, now)

	return Allowed
}

// cleanup removes stale entries to prevent memory leaks.
func (l *Limiter) cleanup(now time.Time) {
	// Clean user entries older than 1 minute.
	for k, v := range l.userLast {
		if now.Sub(v) > time.Minute {
			delete(l.userLast, k)
		}
	}

	// Clean chat entries with no recent commands.
	cutoff := now.Add(-l.chatWindow)
	for k, cmds := range l.chatCmds {
		start := 0
		for start < len(cmds) && cmds[start].Before(cutoff) {
			start++
		}
		if start >= len(cmds) {
			delete(l.chatCmds, k)
		} else {
			l.chatCmds[k] = cmds[start:]
		}
	}
}
