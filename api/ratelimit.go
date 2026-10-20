package api

import (
	"sync"
	"time"
)

// SlidingWindowLimiter allows each client up to limit requests in any
// window of time. It implements RateLimiter.
type SlidingWindowLimiter struct {
	mu          sync.Mutex
	limit       int
	window      time.Duration
	requests    map[string][]time.Time // Each client's requests in the window, oldest first
	lastCleanup time.Time
	now         func() time.Time
}

// NewSlidingWindowLimiter creates a limiter that allows limit requests per
// client in any window
func NewSlidingWindowLimiter(limit int, window time.Duration) *SlidingWindowLimiter {
	return &SlidingWindowLimiter{
		limit:    limit,
		window:   window,
		requests: make(map[string][]time.Time),
		now:      time.Now,
	}
}

// Allow records a request from key and reports whether it is within the
// limit. A refused request is not counted.
func (l *SlidingWindowLimiter) Allow(key string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := l.now()
	l.cleanupLocked(now)
	requests := l.recentLocked(key, now)
	if len(requests) >= l.limit {
		return false
	}
	l.requests[key] = append(requests, now)
	return true
}

// Limit returns how many requests a client may make in a window
func (l *SlidingWindowLimiter) Limit() int {
	return l.limit
}

// Remaining returns how many more requests key may make now
func (l *SlidingWindowLimiter) Remaining(key string) int {
	l.mu.Lock()
	defer l.mu.Unlock()
	return max(l.limit-len(l.recentLocked(key, l.now())), 0)
}

// ResetTime returns when key may make another request: now if it is
// within the limit, otherwise when its oldest request leaves the window
func (l *SlidingWindowLimiter) ResetTime(key string) time.Time {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := l.now()
	requests := l.recentLocked(key, now)
	if len(requests) < l.limit {
		return now
	}
	return requests[len(requests)-l.limit].Add(l.window)
}

// recentLocked returns key's requests within the window, dropping older
// ones. Caller must hold l.mu.
func (l *SlidingWindowLimiter) recentLocked(key string, now time.Time) []time.Time {
	requests := l.requests[key]
	cutoff := now.Add(-l.window)
	i := 0
	for i < len(requests) && !requests[i].After(cutoff) {
		i++
	}
	if i == len(requests) {
		delete(l.requests, key)
		return nil
	}
	if i > 0 {
		requests = append(requests[:0], requests[i:]...)
		l.requests[key] = requests
	}
	return requests
}

// cleanupLocked forgets clients with no requests in the window, at most
// once per window, so clients that stop calling don't use memory forever.
// Caller must hold l.mu.
func (l *SlidingWindowLimiter) cleanupLocked(now time.Time) {
	if now.Sub(l.lastCleanup) < l.window {
		return
	}
	l.lastCleanup = now
	for key := range l.requests {
		l.recentLocked(key, now)
	}
}

// Clients returns how many clients the limiter is tracking
func (l *SlidingWindowLimiter) Clients() int {
	l.mu.Lock()
	defer l.mu.Unlock()
	return len(l.requests)
}
