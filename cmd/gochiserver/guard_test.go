package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// guarded returns a guarded handler for a test server: the game's page
// and a few routes
func guarded(t *testing.T) (*Server, http.Handler) {
	t.Helper()
	s := newTestServer(t)
	s.parental, _ = loadParentalControls("")
	s.pairing, _ = loadDevicePairing("")
	s.onNetwork, s.listenAddr = true, ":8080"
	if err := s.createPet("Biscuit", "cat", false, false); err != nil {
		t.Fatal(err)
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/api/status", s.handleStatus)
	mux.HandleFunc("/api/rename", s.handleRename)
	mux.HandleFunc("/api/pairing", s.handlePairing)
	mux.HandleFunc("/pair", s.handlePair)
	mux.HandleFunc("/api/v1/pets", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusTeapot) })
	mux.HandleFunc("/", s.handleIndex)
	return s, s.guard(mux)
}

// request sends a request to a handler from an address, with headers
func request(h http.Handler, method, path, from string, body string, headers map[string]string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.RemoteAddr = from
	req.Host = "localhost:8080"
	for k, v := range headers {
		if k == "Host" {
			req.Host = v
		} else {
			req.Header.Set(k, v)
		}
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

const (
	local  = "127.0.0.1:50000"
	remote = "192.168.1.20:50000"
)

var jsonHeader = map[string]string{"Content-Type": "application/json"}

func TestGuardRefusesOtherSites(t *testing.T) {
	s, h := guarded(t)

	// The game's own page works
	if rec := request(h, http.MethodPost, "/api/rename", local, `{"name":"Mochi"}`,
		map[string]string{"Content-Type": "application/json", "Origin": "http://localhost:8080", "Sec-Fetch-Site": "same-origin"}); rec.Code != http.StatusOK {
		t.Fatalf("The game's page should work: %d %s", rec.Code, rec.Body)
	}

	for name, headers := range map[string]map[string]string{
		"another origin":      {"Content-Type": "application/json", "Origin": "https://evil.example"},
		"cross-site fetch":    {"Content-Type": "application/json", "Sec-Fetch-Site": "cross-site"},
		"a form post":         {"Content-Type": "text/plain"},
		"no content type":     {},
		"a rebinding name":    {"Content-Type": "application/json", "Host": "evil.example:8080"},
		"a rebinding subname": {"Content-Type": "application/json", "Host": "localhost.evil.example"},
	} {
		if rec := request(h, http.MethodPost, "/api/rename", local, `{"name":"pwned"}`, headers); rec.Code < 400 {
			t.Errorf("A request with %s should be refused: %d", name, rec.Code)
		}
	}
	if s.pet.Name != "Mochi" {
		t.Errorf("The pet was renamed to %q", s.pet.Name)
	}

	// Reading: no CORS headers, and other sites' requests are refused
	rec := request(h, http.MethodGet, "/api/status", local, "", nil)
	if rec.Code != http.StatusOK || rec.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Errorf("Status: %d, CORS %q", rec.Code, rec.Header().Get("Access-Control-Allow-Origin"))
	}
	if rec := request(h, http.MethodGet, "/api/status", local, "", map[string]string{"Sec-Fetch-Site": "cross-site"}); rec.Code != http.StatusForbidden {
		t.Errorf("Another site reading the status: %d", rec.Code)
	}
	if rec := request(h, http.MethodGet, "/api/status", local, "", map[string]string{"Host": "evil.example"}); rec.Code != http.StatusMisdirectedRequest {
		t.Errorf("Rebinding: %d", rec.Code)
	}
	// IP addresses work as the host, for other devices
	if rec := request(h, http.MethodGet, "/api/status", local, "", map[string]string{"Host": "192.168.1.5:8080"}); rec.Code != http.StatusOK {
		t.Errorf("An IP host: %d", rec.Code)
	}
	// The REST API has its own token and isn't held to same-origin rules
	if rec := request(h, http.MethodPost, "/api/v1/pets", local, `{}`, nil); rec.Code != http.StatusTeapot {
		t.Errorf("The REST API: %d", rec.Code)
	}
}

func TestOtherDevicesNeedPairing(t *testing.T) {
	s, h := guarded(t)
	clock := time.Now()
	s.pairing.now = func() time.Time { return clock }

	// An unpaired device gets the pairing page and nothing else
	if rec := request(h, http.MethodGet, "/", remote, "", nil); rec.Code != http.StatusUnauthorized ||
		!strings.Contains(rec.Body.String(), "Pair this device") {
		t.Errorf("Unpaired page: %d", rec.Code)
	}
	if rec := request(h, http.MethodGet, "/api/status", remote, "", nil); rec.Code != http.StatusUnauthorized {
		t.Errorf("Unpaired status: %d", rec.Code)
	}
	// Only this computer can get a code
	if rec := request(h, http.MethodGet, "/api/pairing", remote, "", nil); rec.Code != http.StatusUnauthorized {
		t.Errorf("A device asking for a code: %d", rec.Code)
	}
	rec := request(h, http.MethodGet, "/api/pairing", local, "", nil)
	status := getJSON(t, func(w http.ResponseWriter, r *http.Request) { w.Write(rec.Body.Bytes()) }, "/")
	code, _ := status["code"].(string)
	if len(code) != pairingCodeLength {
		t.Fatalf("Pairing status: %v", status)
	}

	// A wrong code, then the right one
	if rec := request(h, http.MethodPost, "/pair", remote, `{"code":"WRONG"}`, jsonHeader); rec.Code != http.StatusForbidden {
		t.Errorf("A wrong code: %d", rec.Code)
	}
	rec = request(h, http.MethodPost, "/pair", remote, `{"code":"`+strings.ToLower(code)+`"}`, jsonHeader)
	cookies := rec.Result().Cookies()
	if rec.Code != http.StatusOK || len(cookies) != 1 || !cookies[0].HttpOnly || cookies[0].SameSite != http.SameSiteStrictMode {
		t.Fatalf("Pairing: %d %+v", rec.Code, cookies)
	}
	paired := map[string]string{"Cookie": deviceCookie + "=" + cookies[0].Value}
	if rec := request(h, http.MethodGet, "/api/status", remote, "", paired); rec.Code != http.StatusOK {
		t.Errorf("A paired device: %d", rec.Code)
	}
	if rec := request(h, http.MethodGet, "/api/status", remote, "", map[string]string{"Cookie": deviceCookie + "=made-up"}); rec.Code != http.StatusUnauthorized {
		t.Errorf("A made-up cookie: %d", rec.Code)
	}

	// Codes expire
	request(h, http.MethodGet, "/api/pairing", local, "", nil)
	code = s.pairing.code
	clock = clock.Add(pairingCodeLife + time.Second)
	if rec := request(h, http.MethodPost, "/pair", remote, `{"code":"`+code+`"}`, jsonHeader); rec.Code != http.StatusForbidden {
		t.Errorf("An expired code: %d", rec.Code)
	}

	// Guessing locks pairing for a while and throws the code away
	request(h, http.MethodGet, "/api/pairing", local, "", nil)
	code = s.pairing.code
	for i := 0; i < maxPairingTries; i++ {
		request(h, http.MethodPost, "/pair", remote, `{"code":"AAAAAAAA"}`, jsonHeader)
	}
	if rec := request(h, http.MethodPost, "/pair", remote, `{"code":"`+code+`"}`, jsonHeader); rec.Code != http.StatusTooManyRequests {
		t.Errorf("After %d wrong codes: %d", maxPairingTries, rec.Code)
	}

	// Forgetting unpairs every device
	if rec := request(h, http.MethodPost, "/api/pairing", local, `{"forget":true}`, jsonHeader); rec.Code != http.StatusOK {
		t.Fatalf("Forget: %d %s", rec.Code, rec.Body)
	}
	if rec := request(h, http.MethodGet, "/api/status", remote, "", paired); rec.Code != http.StatusUnauthorized {
		t.Errorf("A forgotten device: %d", rec.Code)
	}
}

func TestPairedDevicesAreKept(t *testing.T) {
	dir := t.TempDir()
	d, _ := loadDevicePairing(dir)
	code, _, _ := d.NewCode()
	token, err := d.Pair(code)
	if err != nil {
		t.Fatal(err)
	}
	d, _ = loadDevicePairing(dir)
	if !d.Paired(token) || d.Paired("other") {
		t.Error("Paired devices should survive a restart")
	}
}

func TestListensOnNetwork(t *testing.T) {
	for addr, want := range map[string]bool{
		"127.0.0.1:8080": false, "localhost:8080": false, "[::1]:8080": false,
		":8080": true, "0.0.0.0:8080": true, "192.168.1.5:8080": true,
	} {
		if got := listensOnNetwork(addr); got != want {
			t.Errorf("listensOnNetwork(%q) = %v", addr, got)
		}
	}
	if defaultConfig().Server.Addr != "127.0.0.1:8080" || listensOnNetwork(defaultConfig().Server.Addr) {
		t.Error("By default the game should answer only this computer")
	}
}
