package ratelimit

import (
	"fmt"
	"testing"
	"time"
)

func TestLimiter_AllowsFirstRequest(t *testing.T) {
	l := New(1*time.Second, 10, time.Minute)
	result := l.Check("user1", "chat1")
	if result != Allowed {
		t.Errorf("First request should be Allowed, got %v", result)
	}
}

func TestLimiter_UserCooldown(t *testing.T) {
	l := New(100*time.Millisecond, 100, time.Minute)

	// First request: allowed
	if r := l.Check("user1", "chat1"); r != Allowed {
		t.Fatalf("First request should be Allowed, got %v", r)
	}

	// Immediate second request: should be rate limited
	if r := l.Check("user1", "chat1"); r != UserCooldown {
		t.Errorf("Immediate second request should be UserCooldown, got %v", r)
	}

	// Wait for cooldown to expire
	time.Sleep(150 * time.Millisecond)

	// Should be allowed again
	if r := l.Check("user1", "chat1"); r != Allowed {
		t.Errorf("Request after cooldown should be Allowed, got %v", r)
	}
}

func TestLimiter_DifferentUsersIndependent(t *testing.T) {
	l := New(1*time.Second, 100, time.Minute)

	if r := l.Check("user1", "chat1"); r != Allowed {
		t.Fatalf("user1 first request should be Allowed, got %v", r)
	}

	// Different user should not be affected
	if r := l.Check("user2", "chat1"); r != Allowed {
		t.Errorf("user2 should be Allowed independently, got %v", r)
	}
}

func TestLimiter_ChatRateLimit(t *testing.T) {
	limit := 3
	l := New(0, limit, time.Minute) // No per-user cooldown

	// Fill up the chat limit
	for i := 0; i < limit; i++ {
		user := fmt.Sprintf("user%d", i)
		if r := l.Check(user, "chat1"); r != Allowed {
			t.Fatalf("Request %d should be Allowed, got %v", i, r)
		}
	}

	// Next request should hit chat rate limit
	if r := l.Check("userX", "chat1"); r != ChatRateLimit {
		t.Errorf("Request exceeding chat limit should be ChatRateLimit, got %v", r)
	}
}

func TestLimiter_DifferentChatsIndependent(t *testing.T) {
	limit := 2
	l := New(0, limit, time.Minute)

	// Fill chat1
	for i := 0; i < limit; i++ {
		l.Check(fmt.Sprintf("user%d", i), "chat1")
	}

	// chat2 should still be fine
	if r := l.Check("user1", "chat2"); r != Allowed {
		t.Errorf("Different chat should be Allowed, got %v", r)
	}
}

func TestLimiter_SlidingWindowExpiry(t *testing.T) {
	limit := 2
	window := 100 * time.Millisecond
	l := New(0, limit, window)

	// Fill up
	l.Check("user1", "chat1")
	l.Check("user2", "chat1")

	// Should be rate limited
	if r := l.Check("user3", "chat1"); r != ChatRateLimit {
		t.Errorf("Should be ChatRateLimit, got %v", r)
	}

	// Wait for window to expire
	time.Sleep(150 * time.Millisecond)

	// Should be allowed again
	if r := l.Check("user3", "chat1"); r != Allowed {
		t.Errorf("After window expiry should be Allowed, got %v", r)
	}
}

func TestLimiter_Cleanup(t *testing.T) {
	l := New(1*time.Millisecond, 100, 1*time.Millisecond)

	// Add some entries
	l.Check("user1", "chat1")
	l.Check("user2", "chat2")

	// Wait for entries to become stale
	time.Sleep(10 * time.Millisecond)

	// Force cleanup by setting lastCleanup far in the past
	l.mu.Lock()
	l.lastCleanup = time.Now().Add(-10 * time.Minute)
	l.mu.Unlock()

	// Next check triggers cleanup
	l.Check("user3", "chat3")

	l.mu.Lock()
	defer l.mu.Unlock()

	// Stale user entries should be cleaned
	if _, exists := l.userLast["user1"]; exists {
		t.Error("user1 should have been cleaned up")
	}
}
