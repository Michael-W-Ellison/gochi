package main

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"mime"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// The game's server only answers the player. By default it listens on
// this computer alone. Every request is checked by guard:
//
//   - The Host header must be "localhost" or an IP address, so a web page
//     can't reach the server through a name it controls (DNS rebinding).
//   - Requests that change something must come from the game's own page:
//     a JSON body, and no Origin (or Sec-Fetch-Site) from another site. A
//     web page the player visits can't use the game; and since responses
//     carry no CORS headers, it can't read them either.
//   - Requests from other devices (when the server listens on the
//     network, for AR on a phone) need the device to have been paired
//     with a code shown on this computer. Paired devices get a cookie.
//
// The REST API (/api/v1, /health, /ready) has its own bearer token and is
// subject only to the Host check.

// Pairing settings
const (
	pairingCodeLength = 8
	pairingCodeLife   = 10 * time.Minute
	maxPairingTries   = 5               // Wrong codes from one address before it waits
	pairingLockout    = 5 * time.Minute // The wait
	maxPairingGuesser = 1000            // Addresses whose wrong codes are remembered
	deviceCookie      = "gochi_device"
	deviceCookieAge   = 365 * 24 * 60 * 60 // Seconds
)

// pairingAlphabet has no letters or digits that look alike
const pairingAlphabet = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"

// devicePairing holds the current pairing code and the paired devices
type devicePairing struct {
	mu       sync.Mutex
	path     string          // devices.json; "" keeps them in memory only
	devices  map[string]bool // Hashes of paired devices' tokens
	code     string          // The current pairing code, or ""; it pairs one device
	expires  time.Time       // When the code stops working
	guessers map[string]*pairingGuesser
	now      func() time.Time
}

// pairingGuesser is an address that has sent wrong codes. Each address
// waits after its own wrong codes, so one device can't stop others
// pairing; with 8 characters from 32, a code can't be guessed in the 10
// minutes it lasts.
type pairingGuesser struct {
	failures int       // Wrong codes in a row
	lockedTo time.Time // No codes accepted from it until then
	last     time.Time // Its last wrong code
}

// loadDevicePairing reads the paired devices from devices.json in dir
func loadDevicePairing(dir string) (*devicePairing, error) {
	d := &devicePairing{devices: make(map[string]bool), guessers: make(map[string]*pairingGuesser), now: time.Now}
	if dir == "" {
		return d, nil
	}
	d.path = filepath.Join(dir, "devices.json")
	data, err := os.ReadFile(d.path)
	switch {
	case errors.Is(err, os.ErrNotExist):
		return d, nil
	case err != nil:
		return nil, err
	}
	var hashes []string
	if err := json.Unmarshal(data, &hashes); err != nil {
		return nil, fmt.Errorf("%s: %w", d.path, err)
	}
	for _, h := range hashes {
		d.devices[h] = true
	}
	return d, nil
}

// saveLocked writes devices.json. Caller must hold d.mu.
func (d *devicePairing) saveLocked() error {
	if d.path == "" {
		return nil
	}
	hashes := make([]string, 0, len(d.devices))
	for h := range d.devices {
		hashes = append(hashes, h)
	}
	data, err := json.Marshal(hashes)
	if err != nil {
		return err
	}
	tmp := d.path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		_ = os.Remove(tmp) // Cut short (a full disk); the write's error is the one to report
		return err
	}
	return os.Rename(tmp, d.path)
}

// hashDeviceToken returns the stored form of a device's token
func hashDeviceToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

// NewCode makes a new pairing code, valid for pairingCodeLife
func (d *devicePairing) NewCode() (string, time.Time, error) {
	var b strings.Builder
	for i := 0; i < pairingCodeLength; i++ {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(pairingAlphabet))))
		if err != nil {
			return "", time.Time{}, err
		}
		b.WriteByte(pairingAlphabet[n.Int64()])
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	d.code, d.expires = b.String(), d.now().Add(pairingCodeLife)
	return d.code, d.expires, nil
}

// Errors from pairing
var (
	errWrongPairingCode = errors.New("that code isn't right, or has expired; get a new one on the computer running the game")
	errPairingLockedOut = fmt.Errorf("too many wrong codes; try again in %d minutes", int(pairingLockout.Minutes()))
)

// Pair checks a pairing code sent from an address and returns a token for
// the device
func (d *devicePairing) Pair(code, address string) (string, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	now := d.now()
	guesser := d.guessers[address]
	if guesser != nil && now.Before(guesser.lockedTo) {
		return "", errPairingLockedOut
	}
	code = strings.ToUpper(strings.TrimSpace(code))
	if d.code == "" || now.After(d.expires) || subtle.ConstantTimeCompare([]byte(code), []byte(d.code)) != 1 {
		if guesser == nil {
			d.forgetGuessersLocked(now)
			guesser = &pairingGuesser{}
			d.guessers[address] = guesser
		}
		guesser.failures++
		guesser.last = now
		if guesser.failures >= maxPairingTries {
			guesser.failures, guesser.lockedTo = 0, now.Add(pairingLockout)
		}
		return "", errWrongPairingCode
	}
	delete(d.guessers, address)
	d.code = "" // Each code pairs one device
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	token := hex.EncodeToString(raw)
	d.devices[hashDeviceToken(token)] = true
	if err := d.saveLocked(); err != nil {
		delete(d.devices, hashDeviceToken(token))
		return "", err
	}
	return token, nil
}

// forgetGuessersLocked forgets addresses that are no longer waiting and
// haven't guessed lately, and if there are still too many, all of them.
// Caller must hold d.mu.
func (d *devicePairing) forgetGuessersLocked(now time.Time) {
	if len(d.guessers) < maxPairingGuesser {
		return
	}
	for address, g := range d.guessers {
		if now.After(g.lockedTo) && now.Sub(g.last) > pairingLockout {
			delete(d.guessers, address)
		}
	}
	if len(d.guessers) >= maxPairingGuesser {
		d.guessers = make(map[string]*pairingGuesser)
	}
}

// Paired reports whether a device token is paired
func (d *devicePairing) Paired(token string) bool {
	if token == "" {
		return false
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.devices[hashDeviceToken(token)]
}

// Count returns how many devices are paired
func (d *devicePairing) Count() int {
	d.mu.Lock()
	defer d.mu.Unlock()
	return len(d.devices)
}

// Forget unpairs every device
func (d *devicePairing) Forget() error {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.devices = make(map[string]bool)
	d.code = ""
	return d.saveLocked()
}

// onlyThisComputer refuses a request from another device, for settings
// only the computer running the game may change (which hub the game uses
// and what it shares, the parent PIN and privacy settings), and reports
// whether it came from this computer
func onlyThisComputer(w http.ResponseWriter, r *http.Request) bool {
	if fromThisComputer(r) {
		return true
	}
	writeJSON(w, http.StatusForbidden, APIResponse{Success: false,
		Error: "Change this on the computer running the game"})
	return false
}

// remoteHost returns the address a request came from, without its port
func remoteHost(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

// fromThisComputer reports whether a request came from this computer
func fromThisComputer(r *http.Request) bool {
	ip := net.ParseIP(remoteHost(r))
	return ip != nil && ip.IsLoopback()
}

// allowedHost reports whether a Host header names this server in a way no
// web page can control: "localhost" or an IP address
func allowedHost(hostport string) bool {
	host := hostport
	if h, _, err := net.SplitHostPort(hostport); err == nil {
		host = h
	}
	host = strings.Trim(host, "[]")
	return strings.EqualFold(host, "localhost") || net.ParseIP(host) != nil
}

// sameOrigin reports whether a request that changes something came from
// the game's own page
func sameOrigin(r *http.Request) bool {
	if site := r.Header.Get("Sec-Fetch-Site"); site != "" && site != "same-origin" && site != "none" {
		return false
	}
	if origin := r.Header.Get("Origin"); origin != "" {
		return origin == "http://"+r.Host || origin == "https://"+r.Host
	}
	return true
}

// changes reports whether a request method can change something
func changes(method string) bool {
	return method != http.MethodGet && method != http.MethodHead && method != http.MethodOptions
}

// The largest request bodies the game reads: every request the page or the
// REST API makes is far smaller
const (
	maxRequestBytes     = 64 << 10
	maxPairRequestBytes = 1 << 10
)

// guardError writes a refusal
func guardError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(status)
	_, _ = fmt.Fprintln(w, message) // Fails only if the client has gone
}

// guard checks every request to the game (see the top of this file)
func (s *Server) guard(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !allowedHost(r.Host) {
			guardError(w, http.StatusMisdirectedRequest, "Open the game at http://localhost or the computer's IP address.")
			return
		}
		path := r.URL.Path
		limit := int64(maxRequestBytes)
		if path == "/pair" {
			limit = maxPairRequestBytes // Answered for devices that aren't paired
		}
		if r.ContentLength > limit {
			guardError(w, http.StatusRequestEntityTooLarge, "The request is too large.")
			return
		}
		r.Body = http.MaxBytesReader(w, r.Body, limit)
		restAPI := path == "/api/v1" || strings.HasPrefix(path, "/api/v1/") || path == "/health" || path == "/ready"
		if restAPI {
			next.ServeHTTP(w, r)
			return
		}
		if r.Header.Get("Sec-Fetch-Site") == "cross-site" && strings.HasPrefix(path, "/api/") {
			guardError(w, http.StatusForbidden, "Other web sites can't use the game.")
			return
		}
		if changes(r.Method) {
			if !sameOrigin(r) {
				guardError(w, http.StatusForbidden, "Other web sites can't use the game.")
				return
			}
			if mediaType, _, _ := mime.ParseMediaType(r.Header.Get("Content-Type")); mediaType != "application/json" {
				guardError(w, http.StatusUnsupportedMediaType, "Requests to the game must be JSON.")
				return
			}
		}
		if !fromThisComputer(r) && path != "/pair" {
			cookie, err := r.Cookie(deviceCookie)
			if err != nil || !s.pairing.Paired(cookie.Value) {
				if path == "/" {
					servePairingPage(w)
				} else {
					guardError(w, http.StatusUnauthorized, "Pair this device first: open the game's address and enter the code shown on the computer.")
				}
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}

// handlePair handles POST /pair with {"code"} from another device: a right
// code pairs it, setting its cookie
func (s *Server) handlePair(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Code string `json:"code"`
	}
	if !decodePost(w, r, &req) {
		return
	}
	token, err := s.pairing.Pair(req.Code, remoteHost(r))
	if err != nil {
		status := http.StatusForbidden
		if errors.Is(err, errPairingLockedOut) {
			status = http.StatusTooManyRequests
		}
		writeJSON(w, status, APIResponse{Success: false, Error: capitalize(err.Error()) + "."})
		return
	}
	http.SetCookie(w, &http.Cookie{Name: deviceCookie, Value: token, Path: "/", MaxAge: deviceCookieAge,
		HttpOnly: true, SameSite: http.SameSiteStrictMode, Secure: r.TLS != nil})
	writeJSON(w, http.StatusOK, APIResponse{Success: true})
}

// pairingStatus is what GET /api/pairing returns
type pairingStatus struct {
	OnNetwork bool      `json:"on_network"` // The server listens on the network, not just this computer
	Addresses []string  `json:"addresses"`  // Where other devices open the game
	Code      string    `json:"code,omitempty"`
	Expires   time.Time `json:"expires,omitempty"`
	Devices   int       `json:"devices"` // Paired devices
}

// handlePairing handles GET /api/pairing (on this computer only): a new
// pairing code and the addresses to open on the other device; and POST
// /api/pairing with {"forget": true, "parent_pin"} to unpair every device
func (s *Server) handlePairing(w http.ResponseWriter, r *http.Request) {
	if !fromThisComputer(r) {
		writeJSON(w, http.StatusForbidden, APIResponse{Success: false, Error: "Devices are paired from the computer running the game"})
		return
	}
	if r.Method == http.MethodPost {
		var req struct {
			Forget    bool   `json:"forget"`
			ParentPIN string `json:"parent_pin"`
		}
		if !decodePost(w, r, &req) || !s.parentAllows(w, req.ParentPIN) {
			return
		}
		if req.Forget {
			if err := s.pairing.Forget(); err != nil {
				writeJSON(w, http.StatusInternalServerError, APIResponse{Success: false, Error: "Could not save"})
				return
			}
		}
		writeJSON(w, http.StatusOK, APIResponse{Success: true, Data: pairingStatus{OnNetwork: s.onNetwork,
			Addresses: s.networkAddresses(), Devices: s.pairing.Count()}})
		return
	}
	status := pairingStatus{OnNetwork: s.onNetwork, Addresses: s.networkAddresses(), Devices: s.pairing.Count()}
	if s.onNetwork {
		code, expires, err := s.pairing.NewCode()
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, APIResponse{Success: false, Error: "Could not make a code"})
			return
		}
		status.Code, status.Expires = code, expires
	}
	writeJSON(w, http.StatusOK, APIResponse{Success: true, Data: status})
}

// networkAddresses returns the addresses other devices on the network can
// open the game at, if the server listens on the network
func (s *Server) networkAddresses() []string {
	addresses := []string{}
	if !s.onNetwork {
		return addresses
	}
	_, port, _ := net.SplitHostPort(s.listenAddr)
	scheme := "http"
	if s.usesTLS {
		scheme = "https"
	}
	ifaces, _ := net.InterfaceAddrs()
	for _, a := range ifaces {
		if ipNet, ok := a.(*net.IPNet); ok && !ipNet.IP.IsLoopback() && !ipNet.IP.IsLinkLocalUnicast() && ipNet.IP.To4() != nil {
			addresses = append(addresses, fmt.Sprintf("%s://%s", scheme, net.JoinHostPort(ipNet.IP.String(), port)))
		}
	}
	return addresses
}

// listensOnNetwork reports whether an address to listen on reaches beyond
// this computer
func listensOnNetwork(addr string) bool {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		return true
	}
	if strings.EqualFold(host, "localhost") {
		return false
	}
	ip := net.ParseIP(host)
	return ip == nil || !ip.IsLoopback()
}

// servePairingPage serves the page another device sees before it is
// paired: it asks for the code shown on the computer running the game
func servePairingPage(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(http.StatusUnauthorized)
	_, _ = fmt.Fprint(w, pairingPage) // Fails only if the client has gone
}

const pairingPage = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>Gochi - Pair this device</title>
<style>
body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; background: linear-gradient(135deg, #667eea, #764ba2);
  min-height: 100vh; margin: 0; display: flex; align-items: center; justify-content: center; padding: 16px; box-sizing: border-box; }
.box { background: white; border-radius: 20px; padding: 28px; max-width: 360px; width: 100%; text-align: center; }
h1 { margin: 0 0 8px; color: #333; font-size: 1.4em; }
p { color: #555; }
input { font-size: 1.4em; letter-spacing: 0.2em; text-transform: uppercase; text-align: center; width: 100%; box-sizing: border-box;
  padding: 10px; border: 2px solid #ddd; border-radius: 12px; }
button { margin-top: 14px; width: 100%; padding: 12px; font-size: 1.1em; border: none; border-radius: 12px; background: #764ba2; color: white; }
#error { color: #c92a2a; min-height: 1.2em; }
</style>
</head>
<body>
<div class="box">
<h1>Pair this device</h1>
<p>On the computer running Gochi, open 🔒 Privacy and look under "Play on a phone or tablet" for the code.</p>
<input id="code" maxlength="8" autocomplete="off" autocapitalize="characters">
<button id="pair">Pair</button>
<p id="error"></p>
</div>
<script>
document.getElementById('pair').onclick = () => {
  fetch('/pair', {method: 'POST', headers: {'Content-Type': 'application/json'},
    body: JSON.stringify({code: document.getElementById('code').value})})
    .then(r => r.json())
    .then(data => { if (data.success) location.reload(); else document.getElementById('error').textContent = data.error; })
    .catch(() => { document.getElementById('error').textContent = 'Could not reach the game.'; });
};
</script>
</body>
</html>`
