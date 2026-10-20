package api

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// testLimiter returns a limiter on a clock the test moves
func testLimiter(limit int, window time.Duration) (*SlidingWindowLimiter, *time.Time) {
	l := NewSlidingWindowLimiter(limit, window)
	clock := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	l.now = func() time.Time { return clock }
	return l, &clock
}

func TestSlidingWindowLimiterAllowsUpToLimit(t *testing.T) {
	l, clock := testLimiter(3, time.Minute)
	for i := 0; i < 3; i++ {
		if !l.Allow("a") {
			t.Fatalf("Request %d should be allowed", i+1)
		}
		*clock = clock.Add(10 * time.Second)
	}
	if l.Allow("a") {
		t.Error("A fourth request in the window should be refused")
	}
	if !l.Allow("b") {
		t.Error("Another client has its own limit")
	}
	if l.Remaining("a") != 0 || l.Remaining("b") != 2 || l.Limit() != 3 {
		t.Errorf("Remaining: a=%d b=%d", l.Remaining("a"), l.Remaining("b"))
	}

	// The oldest request (at 12:00:00) leaves the window at 12:01:00
	if want := time.Date(2026, 1, 1, 12, 1, 0, 0, time.UTC); !l.ResetTime("a").Equal(want) {
		t.Errorf("ResetTime = %s, want %s", l.ResetTime("a"), want)
	}
	*clock = time.Date(2026, 1, 1, 12, 1, 0, 1, time.UTC)
	if !l.Allow("a") {
		t.Error("A request should be allowed once the oldest leaves the window")
	}
	if l.Allow("a") {
		t.Error("Only one request left the window")
	}
}

func TestSlidingWindowLimiterDoesNotCountRefusals(t *testing.T) {
	l, clock := testLimiter(1, time.Minute)
	l.Allow("a")
	for i := 0; i < 5; i++ {
		*clock = clock.Add(10 * time.Second)
		l.Allow("a") // Refused
	}
	*clock = clock.Add(11 * time.Second) // 61s after the only counted request
	if !l.Allow("a") {
		t.Error("Refused requests should not count against the client")
	}
}

func TestSlidingWindowLimiterForgetsIdleClients(t *testing.T) {
	l, clock := testLimiter(5, time.Minute)
	for _, key := range []string{"a", "b", "c"} {
		l.Allow(key)
	}
	if l.Clients() != 3 {
		t.Fatalf("Clients = %d, want 3", l.Clients())
	}
	*clock = clock.Add(2 * time.Minute)
	l.Allow("d")
	if l.Clients() != 1 {
		t.Errorf("Idle clients should be forgotten, tracking %d", l.Clients())
	}
	if l.ResetTime("a") != *clock || l.Remaining("a") != 5 {
		t.Error("A forgotten client starts with its full limit")
	}
}

func TestRateLimitMiddleware(t *testing.T) {
	l, _ := testLimiter(2, time.Minute)
	handler := RateLimitMiddleware(l)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))

	request := func(remoteAddr string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.RemoteAddr = remoteAddr
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		return rec
	}

	// Each connection from a client uses a new port; the limit is per address
	first := request("10.0.0.1:5000")
	if first.Code != http.StatusOK || first.Header().Get("X-RateLimit-Limit") != "2" ||
		first.Header().Get("X-RateLimit-Remaining") != "1" {
		t.Errorf("First request: %d, headers %v", first.Code, first.Header())
	}
	request("10.0.0.1:5001")
	refused := request("10.0.0.1:5002")
	if refused.Code != http.StatusTooManyRequests || refused.Header().Get("X-RateLimit-Remaining") != "0" ||
		refused.Header().Get("X-RateLimit-Reset") == "" {
		t.Errorf("Third request: %d, headers %v", refused.Code, refused.Header())
	}
	if other := request("10.0.0.2:5000"); other.Code != http.StatusOK {
		t.Errorf("Another address has its own limit, got %d", other.Code)
	}
}
