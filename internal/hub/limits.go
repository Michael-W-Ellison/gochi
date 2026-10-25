package hub

import (
	"net"
	"net/http"
	"sync"
	"time"
)

// Limits on guessing and on growth: wrong join codes and wrong playdate
// codes are counted per address (and wrong join codes across the hub
// too), new accounts per address, and every request per address, so
// someone with many accounts gets no more requests from one address than
// a busy household does.

const (
	maxWrongJoinCodes    = 5   // Per address in guessWindow
	maxWrongJoinCodesHub = 100 // From everyone in guessWindow; registration waits after that
	maxWrongPlaydates    = 20  // Wrong playdate codes per account or address in guessWindow
	guessWindow          = time.Hour
	maxNewAccounts       = 5 // New accounts from one address in newAccountWindow
	newAccountWindow     = 24 * time.Hour
	requestsPerAddress   = 1200 // Requests a minute from one address, whatever the accounts
	maxLimiterKeys       = 100000
)

// ErrTooManyGuesses is returned after too many wrong codes
var ErrTooManyGuesses = errorString("too many wrong codes; try again later")

// ErrTooManyAccounts is returned when an address has made too many
// accounts lately
var ErrTooManyAccounts = errorString("too many new players from this network today; try again tomorrow")

// errorString is an error from a fixed text
type errorString string

func (e errorString) Error() string { return string(e) }

// eventLimiter counts events per key in a sliding window
type eventLimiter struct {
	mu     sync.Mutex
	max    int
	window time.Duration
	now    func() time.Time
	events map[string][]time.Time
}

func newEventLimiter(max int, window time.Duration) *eventLimiter {
	return &eventLimiter{max: max, window: window, now: time.Now, events: make(map[string][]time.Time)}
}

// recentLocked returns key's events in the window. Caller must hold l.mu.
func (l *eventLimiter) recentLocked(key string) []time.Time {
	now := l.now()
	list := l.events[key]
	i := 0
	for i < len(list) && now.Sub(list[i]) >= l.window {
		i++
	}
	if i == len(list) {
		delete(l.events, key)
		return nil
	}
	list = list[i:]
	l.events[key] = list
	return list
}

// Allowed reports whether key has had fewer than max events in the window
func (l *eventLimiter) Allowed(key string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	return len(l.recentLocked(key)) < l.max
}

// Add counts an event for key
func (l *eventLimiter) Add(key string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if len(l.events) >= maxLimiterKeys {
		// Too many keys to keep: forget the ones with nothing recent
		for k := range l.events {
			l.recentLocked(k)
		}
	}
	l.events[key] = append(l.recentLocked(key), l.now())
}

// addressKey returns the address a request came from, without its port:
// an IPv6 address by its /64 network, which one household or device
// usually has whole
func addressKey(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr
	}
	ip := net.ParseIP(host)
	if ip == nil {
		return host
	}
	if ip.To4() != nil {
		return ip.String()
	}
	return ip.Mask(net.CIDRMask(64, 128)).String() + "/64"
}

// limitAddresses refuses requests from an address beyond
// requestsPerAddress a minute
func (h *Hub) limitAddresses(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !h.addressRequests.Allow(addressKey(r)) {
			reply(w, http.StatusTooManyRequests, errorReply{Error: "too many requests"})
			return
		}
		next.ServeHTTP(w, r)
	})
}
