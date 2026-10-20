package main

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sync"
	"time"
)

// Privacy and parental controls. A parent can set a PIN; once set, it is
// needed to connect to or disconnect from a cloud hub, to change what is
// shared there, to start or join a playdate, and to change these
// settings. The player can also stop the game learning when they play
// (owner.json) and erase what it has learned. The PIN is kept in
// <data dir>/parental.json as a salted, slow hash; it keeps a child from
// changing settings through the game, not someone with access to the
// computer's files.

// PIN rules
var pinPattern = regexp.MustCompile(`^[0-9]{4,8}$`)

const (
	pinHashRounds = 100_000         // Iterations of the PIN hash
	maxPINTries   = 5               // Wrong PINs before a wait
	pinLockout    = 5 * time.Minute // The wait
)

// Parental control errors
var (
	errParentPIN    = errors.New("a parent's PIN is needed for this")
	errWrongPIN     = errors.New("that PIN isn't right")
	errPINLockedOut = fmt.Errorf("too many wrong PINs; try again in %d minutes", int(pinLockout.Minutes()))
	errBadPIN       = errors.New("a PIN is 4 to 8 digits")
)

// parentalSettings is parental.json
type parentalSettings struct {
	Salt []byte `json:"salt,omitempty"`
	Hash []byte `json:"hash,omitempty"`
}

// parentalControls holds the parent PIN
type parentalControls struct {
	mu       sync.Mutex
	path     string // "" keeps it in memory only
	settings parentalSettings
	failures int
	lockedTo time.Time
	now      func() time.Time
}

// loadParentalControls reads parental.json from dir ("" for none)
func loadParentalControls(dir string) (*parentalControls, error) {
	p := &parentalControls{now: time.Now}
	if dir == "" {
		return p, nil
	}
	p.path = filepath.Join(dir, "parental.json")
	data, err := os.ReadFile(p.path)
	switch {
	case errors.Is(err, os.ErrNotExist):
		return p, nil
	case err != nil:
		return nil, err
	}
	if err := json.Unmarshal(data, &p.settings); err != nil {
		return nil, fmt.Errorf("%s: %w", p.path, err)
	}
	return p, nil
}

// hashPIN hashes a PIN with a salt
func hashPIN(pin string, salt []byte) []byte {
	sum := sha256.Sum256(append(append([]byte{}, salt...), pin...))
	for i := 1; i < pinHashRounds; i++ {
		sum = sha256.Sum256(append(sum[:], salt...))
	}
	return sum[:]
}

// Locked reports whether a parent PIN is set
func (p *parentalControls) Locked() bool {
	if p == nil {
		return false
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	return len(p.settings.Hash) > 0
}

// Check returns nil if no PIN is set or pin is right. Wrong PINs are
// counted; after maxPINTries, every PIN is refused for pinLockout.
func (p *parentalControls) Check(pin string) error {
	if p == nil {
		return nil
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.checkLocked(pin)
}

func (p *parentalControls) checkLocked(pin string) error {
	if len(p.settings.Hash) == 0 {
		return nil
	}
	now := p.now()
	if now.Before(p.lockedTo) {
		return errPINLockedOut
	}
	if pin == "" {
		return errParentPIN
	}
	if subtle.ConstantTimeCompare(hashPIN(pin, p.settings.Salt), p.settings.Hash) == 1 {
		p.failures = 0
		return nil
	}
	p.failures++
	if p.failures >= maxPINTries {
		p.failures, p.lockedTo = 0, now.Add(pinLockout)
	}
	return errWrongPIN
}

// SetPIN sets, changes or (with an empty newPIN) removes the PIN. The
// current PIN is needed if one is set.
func (p *parentalControls) SetPIN(current, newPIN string) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if err := p.checkLocked(current); err != nil {
		return err
	}
	settings := parentalSettings{}
	if newPIN != "" {
		if !pinPattern.MatchString(newPIN) {
			return errBadPIN
		}
		settings.Salt = make([]byte, 16)
		if _, err := rand.Read(settings.Salt); err != nil {
			return err
		}
		settings.Hash = hashPIN(newPIN, settings.Salt)
	}
	if p.path != "" {
		data, err := json.Marshal(settings)
		if err != nil {
			return err
		}
		tmp := p.path + ".tmp"
		if err := os.WriteFile(tmp, data, 0o600); err != nil {
			return err
		}
		if err := os.Rename(tmp, p.path); err != nil {
			return err
		}
	}
	p.settings = settings
	return nil
}

// parentAllows checks the parent PIN for a protected request, writing an
// error and returning false if it isn't given or is wrong
func (s *Server) parentAllows(w http.ResponseWriter, pin string) bool {
	err := s.parental.Check(pin)
	if err == nil {
		return true
	}
	writeParentError(w, err)
	return false
}

// writeParentError reports a missing or wrong parent PIN
func writeParentError(w http.ResponseWriter, err error) {
	status := http.StatusForbidden
	if errors.Is(err, errPINLockedOut) {
		status = http.StatusTooManyRequests
	}
	writeJSON(w, status, APIResponse{Success: false, Error: capitalize(err.Error()) + ".",
		Data: map[string]bool{"parent_pin_needed": true}})
}

// capitalize returns text with its first letter in upper case
func capitalize(text string) string {
	if text == "" {
		return text
	}
	return string(bytes.ToUpper([]byte(text[:1]))) + text[1:]
}

// privacyStatus is what GET /api/privacy returns
type privacyStatus struct {
	LearnSchedule bool   `json:"learn_schedule"`
	PlaySchedule  string `json:"play_schedule"` // What has been learned, or ""
	ParentLock    bool   `json:"parent_lock"`   // A parent PIN is set
}

func (s *Server) privacyStatus() privacyStatus {
	status := privacyStatus{ParentLock: s.parental.Locked()}
	if s.owner != nil {
		status.LearnSchedule, status.PlaySchedule = s.owner.Learning(), s.owner.Summary()
	}
	return status
}

// handlePrivacy handles GET /api/privacy and POST /api/privacy with any of
// {"learn_schedule": bool, "erase_schedule": true, "new_pin": "1234" (""
// to remove the PIN), "remove_pin": true}, plus "parent_pin" if a PIN is
// set
func (s *Server) handlePrivacy(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		writeJSON(w, http.StatusOK, APIResponse{Success: true, Data: s.privacyStatus()})
		return
	}
	var req struct {
		LearnSchedule *bool   `json:"learn_schedule"`
		EraseSchedule bool    `json:"erase_schedule"`
		NewPIN        *string `json:"new_pin"`
		RemovePIN     bool    `json:"remove_pin"`
		ParentPIN     string  `json:"parent_pin"`
	}
	if !decodePost(w, r, &req) {
		return
	}
	if req.NewPIN != nil || req.RemovePIN {
		newPIN := ""
		if req.NewPIN != nil && !req.RemovePIN {
			newPIN = *req.NewPIN
		}
		if err := s.parental.SetPIN(req.ParentPIN, newPIN); err != nil {
			if errors.Is(err, errBadPIN) {
				writeJSON(w, http.StatusBadRequest, APIResponse{Success: false, Error: capitalize(err.Error()) + "."})
				return
			}
			writeParentError(w, err)
			return
		}
	} else if !s.parentAllows(w, req.ParentPIN) {
		return
	}
	if s.owner != nil {
		if req.LearnSchedule != nil {
			s.owner.SetLearning(*req.LearnSchedule)
		}
		if req.EraseSchedule {
			s.owner.Clear()
		}
		if err := s.saveOwner(); err != nil {
			writeJSON(w, http.StatusInternalServerError, APIResponse{Success: false, Error: "Could not save the setting"})
			return
		}
	}
	writeJSON(w, http.StatusOK, APIResponse{Success: true, Data: s.privacyStatus()})
}
