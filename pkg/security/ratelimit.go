package security

import (
	"sync"
	"time"
)

// RateLimiter implements token bucket rate limiting
type RateLimiter struct {
	mu sync.RWMutex

	// Configuration
	maxAttempts int
	windowSize  time.Duration

	// State
	attempts map[string][]time.Time
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(maxAttempts int, windowSize time.Duration) *RateLimiter {
	return &RateLimiter{
		maxAttempts: maxAttempts,
		windowSize:  windowSize,
		attempts:    make(map[string][]time.Time),
	}
}

// CheckLimit returns true if the identifier has exceeded the rate limit
func (rl *RateLimiter) CheckLimit(identifier string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	cutoff := now.Add(-rl.windowSize)

	// Get or create attempts slice for this identifier
	attempts, exists := rl.attempts[identifier]
	if !exists {
		attempts = make([]time.Time, 0)
	}

	// Remove attempts outside the window
	validAttempts := make([]time.Time, 0)
	for _, attempt := range attempts {
		if attempt.After(cutoff) {
			validAttempts = append(validAttempts, attempt)
		}
	}

	// Check if limit exceeded
	if len(validAttempts) >= rl.maxAttempts {
		rl.attempts[identifier] = validAttempts
		return true // Rate limit exceeded
	}

	// Record this attempt
	validAttempts = append(validAttempts, now)
	rl.attempts[identifier] = validAttempts

	return false // Within rate limit
}

// Reset clears rate limit for an identifier
func (rl *RateLimiter) Reset(identifier string) {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	delete(rl.attempts, identifier)
}

// Cleanup removes stale entries older than the window
func (rl *RateLimiter) Cleanup() {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	cutoff := now.Add(-rl.windowSize)

	for identifier, attempts := range rl.attempts {
		validAttempts := make([]time.Time, 0)
		for _, attempt := range attempts {
			if attempt.After(cutoff) {
				validAttempts = append(validAttempts, attempt)
			}
		}

		if len(validAttempts) == 0 {
			delete(rl.attempts, identifier)
		} else {
			rl.attempts[identifier] = validAttempts
		}
	}
}

// GetRemainingAttempts returns how many attempts are left
func (rl *RateLimiter) GetRemainingAttempts(identifier string) int {
	rl.mu.RLock()
	defer rl.mu.RUnlock()

	now := time.Now()
	cutoff := now.Add(-rl.windowSize)

	attempts, exists := rl.attempts[identifier]
	if !exists {
		return rl.maxAttempts
	}

	// Count valid attempts
	validCount := 0
	for _, attempt := range attempts {
		if attempt.After(cutoff) {
			validCount++
		}
	}

	remaining := rl.maxAttempts - validCount
	if remaining < 0 {
		return 0
	}
	return remaining
}
