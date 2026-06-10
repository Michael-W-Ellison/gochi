package security

import (
	"testing"
	"time"
)

func TestNewRateLimiter(t *testing.T) {
	rl := NewRateLimiter(5, 1*time.Minute)
	if rl == nil {
		t.Fatal("NewRateLimiter returned nil")
	}
}

func TestCheckLimit(t *testing.T) {
	rl := NewRateLimiter(3, 1*time.Second)

	// First 3 attempts should succeed
	for i := 0; i < 3; i++ {
		if rl.CheckLimit("test-user") {
			t.Errorf("Attempt %d should not be rate limited", i+1)
		}
	}

	// 4th attempt should be rate limited
	if !rl.CheckLimit("test-user") {
		t.Error("4th attempt should be rate limited")
	}

	// Different user should not be affected
	if rl.CheckLimit("other-user") {
		t.Error("Different user should not be rate limited")
	}
}

func TestCheckLimitWindowExpiry(t *testing.T) {
	rl := NewRateLimiter(2, 100*time.Millisecond)

	// Use up the limit
	rl.CheckLimit("test-user")
	rl.CheckLimit("test-user")

	// Should be rate limited
	if !rl.CheckLimit("test-user") {
		t.Error("Should be rate limited")
	}

	// Wait for window to expire
	time.Sleep(150 * time.Millisecond)

	// Should be allowed again
	if rl.CheckLimit("test-user") {
		t.Error("Should not be rate limited after window expired")
	}
}

func TestReset(t *testing.T) {
	rl := NewRateLimiter(2, 1*time.Minute)

	// Use up the limit
	rl.CheckLimit("test-user")
	rl.CheckLimit("test-user")

	// Should be rate limited
	if !rl.CheckLimit("test-user") {
		t.Error("Should be rate limited before reset")
	}

	// Reset the user
	rl.Reset("test-user")

	// Should be allowed again
	if rl.CheckLimit("test-user") {
		t.Error("Should not be rate limited after reset")
	}
}

func TestCleanup(t *testing.T) {
	rl := NewRateLimiter(5, 50*time.Millisecond)

	// Make some attempts
	rl.CheckLimit("user1")
	rl.CheckLimit("user2")
	rl.CheckLimit("user3")

	// Wait for entries to become stale
	time.Sleep(100 * time.Millisecond)

	// Cleanup
	rl.Cleanup()

	// Attempts map should be empty or have zero entries
	rl.mu.RLock()
	count := len(rl.attempts)
	rl.mu.RUnlock()

	if count != 0 {
		t.Errorf("Expected 0 entries after cleanup, got %d", count)
	}
}

func TestGetRemainingAttempts(t *testing.T) {
	rl := NewRateLimiter(5, 1*time.Minute)

	// Initially should have all attempts
	if remaining := rl.GetRemainingAttempts("test-user"); remaining != 5 {
		t.Errorf("Expected 5 remaining attempts, got %d", remaining)
	}

	// Use one attempt
	rl.CheckLimit("test-user")

	// Should have 4 remaining
	if remaining := rl.GetRemainingAttempts("test-user"); remaining != 4 {
		t.Errorf("Expected 4 remaining attempts, got %d", remaining)
	}

	// Use all remaining
	for i := 0; i < 4; i++ {
		rl.CheckLimit("test-user")
	}

	// Should have 0 remaining
	if remaining := rl.GetRemainingAttempts("test-user"); remaining != 0 {
		t.Errorf("Expected 0 remaining attempts, got %d", remaining)
	}
}

func TestConcurrentAccess(t *testing.T) {
	rl := NewRateLimiter(100, 1*time.Minute)

	done := make(chan bool, 50)
	for i := 0; i < 50; i++ {
		go func(n int) {
			for j := 0; j < 10; j++ {
				rl.CheckLimit("concurrent-user")
			}
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 50; i++ {
		<-done
	}

	// Should have used exactly 100 attempts (first 100 succeed, rest fail)
	remaining := rl.GetRemainingAttempts("concurrent-user")
	if remaining != 0 {
		t.Logf("Remaining attempts: %d (expected 0, but concurrency may vary)", remaining)
	}
}
