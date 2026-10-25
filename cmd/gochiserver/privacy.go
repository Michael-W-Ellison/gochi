package main

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"sync"
	"time"

	"github.com/Michael-W-Ellison/gochi/internal/core"
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
	pinLockout    = 5 * time.Minute // The first wait; each wait after it is twice as long
	maxPINLockout = 24 * time.Hour  // The longest wait
)

// Parental control errors
var (
	errParentPIN    = errors.New("a parent's PIN is needed for this")
	errWrongPIN     = errors.New("that PIN isn't right")
	errPINLockedOut = errors.New("too many wrong PINs; try again later")
	errBadPIN       = errors.New("a PIN is 4 to 8 digits")
)

// pinTries counts wrong PINs from one source: wrong PINs since the last
// right one, waits since then, and when the current wait ends
type pinTries struct {
	Failures int       `json:"failures,omitempty"`
	Lockouts int       `json:"lockouts,omitempty"`
	LockedTo time.Time `json:"locked_to"`
}

// parentalSettings is parental.json
type parentalSettings struct {
	Salt []byte `json:"salt,omitempty"`
	Hash []byte `json:"hash,omitempty"`

	// Wrong PINs from this computer, and from paired devices together,
	// counted apart so a phone can't lock the parent out of the computer;
	// kept so restarting the game doesn't reset them
	pinTries
	Remote pinTries `json:"remote,omitempty"`

	// The holidays a parent has turned off and whether they turned the
	// game's sounds off, for every player on this device. They were kept
	// in each household before; HouseMoved is set once they were moved
	// here (see adoptHouseSettings).
	HolidaysOff    []string `json:"holidays_off,omitempty"`
	AllHolidaysOff bool     `json:"all_holidays_off,omitempty"`
	SoundOff       bool     `json:"sound_off,omitempty"`
	HouseMoved     bool     `json:"house_moved,omitempty"`
}

// parentalControls holds the parent PIN and the settings a parent makes
// for every player on this device
type parentalControls struct {
	mu       sync.Mutex
	path     string // "" keeps it in memory only
	settings parentalSettings
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
	p.settings.HolidaysOff = knownHolidays(p.settings.HolidaysOff)
	return p, nil
}

// knownHolidays returns the holidays in ids the game knows, once each
func knownHolidays(ids []string) []string {
	var known []string
	for _, id := range ids {
		if _, ok := core.FindHoliday(id); ok && !slices.Contains(known, id) {
			known = append(known, id)
		}
	}
	return known
}

// apply tells the game which holidays are turned off
func (p *parentalControls) apply() {
	if p == nil {
		core.SetHolidays(false, nil)
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	core.SetHolidays(p.settings.AllHolidaysOff, p.settings.HolidaysOff)
}

// adoptHouseSettings moves the holiday and sound settings from a
// household, where versions before accounts kept them, the first time
func (p *parentalControls) adoptHouseSettings(d householdData) error {
	if p == nil {
		return nil
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.settings.HouseMoved {
		return nil
	}
	settings := p.settings
	settings.HolidaysOff, settings.AllHolidaysOff = knownHolidays(d.HolidaysOff), d.AllHolidaysOff
	settings.SoundOff, settings.HouseMoved = d.SoundOff, true
	return p.saveLocked(settings)
}

// holidays returns whether every holiday is off, and which are
func (p *parentalControls) holidays() (bool, []string) {
	if p == nil {
		return false, nil
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.settings.AllHolidaysOff, slices.Clone(p.settings.HolidaysOff)
}

// setHolidays sets which holidays are turned off
func (p *parentalControls) setHolidays(allOff bool, off []string) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	settings := p.settings
	settings.AllHolidaysOff, settings.HolidaysOff = allOff, knownHolidays(off)
	if err := p.saveLocked(settings); err != nil {
		return err
	}
	core.SetHolidays(settings.AllHolidaysOff, settings.HolidaysOff)
	return nil
}

// soundOff reports whether a parent has turned the game's sounds off
func (p *parentalControls) soundOff() bool {
	if p == nil {
		return false
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.settings.SoundOff
}

// setSoundOff sets whether the game's sounds are off
func (p *parentalControls) setSoundOff(off bool) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	settings := p.settings
	settings.SoundOff = off
	return p.saveLocked(settings)
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

// Check returns nil if no PIN is set or pin is right, for a PIN given on
// this computer (see CheckFrom)
func (p *parentalControls) Check(pin string) error {
	return p.CheckFrom(pin, true)
}

// CheckFrom returns nil if no PIN is set or pin is right. Wrong PINs are
// counted, on this computer (local) and on paired devices apart; after
// maxPINTries from one of them, every PIN from it is refused for a wait.
func (p *parentalControls) CheckFrom(pin string, local bool) error {
	if p == nil {
		return nil
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.checkLocked(pin, local)
}

func (p *parentalControls) checkLocked(pin string, local bool) error {
	if len(p.settings.Hash) == 0 {
		return nil
	}
	settings := p.settings
	tries := &settings.Remote
	if local {
		tries = &settings.pinTries
	}
	now := p.now()
	if now.Before(tries.LockedTo) {
		return errPINLockedOut
	}
	if pin == "" {
		return errParentPIN
	}
	if subtle.ConstantTimeCompare(hashPIN(pin, p.settings.Salt), p.settings.Hash) == 1 {
		if tries.Failures != 0 || tries.Lockouts != 0 {
			*tries = pinTries{}
			if err := p.saveLocked(settings); err != nil { // A right PIN is right either way
				log.Printf("Could not save the parental controls: %v", err)
			}
		}
		return nil
	}
	tries.Failures++
	if tries.Failures >= maxPINTries {
		wait := min(pinLockout<<min(tries.Lockouts, 10), maxPINLockout)
		tries.Failures, tries.LockedTo = 0, now.Add(wait)
		tries.Lockouts++
	}
	if err := p.saveLocked(settings); err != nil {
		p.settings = settings // Count it even if it can't be saved
	}
	return errWrongPIN
}

// saveLocked writes parental.json and keeps settings. Caller must hold p.mu.
func (p *parentalControls) saveLocked(settings parentalSettings) error {
	if p.path != "" {
		data, err := json.Marshal(settings)
		if err != nil {
			return err
		}
		tmp := p.path + ".tmp"
		if err := os.WriteFile(tmp, data, 0o600); err != nil {
			_ = os.Remove(tmp) // Cut short; the write's error is the one to report
			return err
		}
		if err := os.Rename(tmp, p.path); err != nil {
			return err
		}
	}
	p.settings = settings
	return nil
}

// SetPIN sets, changes or (with an empty newPIN) removes the PIN. The
// current PIN is needed if one is set.
func (p *parentalControls) SetPIN(current, newPIN string) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if err := p.checkLocked(current, true); err != nil {
		return err
	}
	settings := parentalSettings{HolidaysOff: p.settings.HolidaysOff, AllHolidaysOff: p.settings.AllHolidaysOff,
		SoundOff: p.settings.SoundOff, HouseMoved: p.settings.HouseMoved} // Only the PIN changes, and the wrong PINs reset
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
	return p.saveLocked(settings)
}

// parentAllows checks the parent PIN for a protected request, writing an
// error and returning false if it isn't given or is wrong. Wrong PINs from
// paired devices count apart from this computer's.
func (s *Server) parentAllows(w http.ResponseWriter, r *http.Request, pin string) bool {
	err := s.parental.CheckFrom(pin, fromThisComputer(r))
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
	LearnSchedule  bool            `json:"learn_schedule"`
	PlaySchedule   string          `json:"play_schedule"` // What has been learned, or ""
	ParentLock     bool            `json:"parent_lock"`   // A parent PIN is set
	Holidays       []holidayStatus `json:"holidays"`
	AllHolidaysOff bool            `json:"all_holidays_off"`
	SoundOff       bool            `json:"sound_off"` // A parent has turned the game's sounds off
}

func (s *Server) privacyStatus() privacyStatus {
	status := privacyStatus{ParentLock: s.parental.Locked(), Holidays: holidaysStatus(), SoundOff: s.parental.soundOff()}
	status.AllHolidaysOff, _ = s.parental.holidays()
	if s.owner != nil {
		status.LearnSchedule, status.PlaySchedule = s.owner.Learning(), s.owner.Summary()
	}
	return status
}

// handlePrivacy handles GET /api/privacy and POST /api/privacy with any of
// {"learn_schedule": bool, "erase_schedule": true, "new_pin": "1234" (""
// to remove the PIN), "remove_pin": true, "holidays_off", "all_holidays_off",
// "sound_off": bool}, plus "parent_pin" if a PIN is set
func (s *Server) handlePrivacy(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		writeJSON(w, http.StatusOK, APIResponse{Success: true, Data: s.privacyStatus()})
		return
	}
	var req struct {
		LearnSchedule *bool    `json:"learn_schedule"`
		EraseSchedule bool     `json:"erase_schedule"`
		NewPIN        *string  `json:"new_pin"`
		HolidaysOff   []string `json:"holidays_off"`     // Holidays to turn off; others are turned on
		AllHolidays   *bool    `json:"all_holidays_off"` // Turn every holiday off (or back on)
		SoundOff      *bool    `json:"sound_off"`        // Turn the game's sounds off (or back on)
		RemovePIN     bool     `json:"remove_pin"`
		ParentPIN     string   `json:"parent_pin"`
	}
	if !onlyThisComputer(w, r) || !decodePost(w, r, &req) {
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
	} else if !s.parentAllows(w, r, req.ParentPIN) {
		return
	}
	if (req.HolidaysOff != nil || req.AllHolidays != nil) && s.parental != nil {
		allOff, off := s.parental.holidays()
		if req.AllHolidays != nil {
			allOff = *req.AllHolidays
		}
		if req.HolidaysOff != nil {
			off = req.HolidaysOff
		}
		if err := s.parental.setHolidays(allOff, off); err != nil {
			writeJSON(w, http.StatusInternalServerError, APIResponse{Success: false, Error: "Could not save the setting"})
			return
		}
	}
	if req.SoundOff != nil && s.parental != nil {
		if err := s.parental.setSoundOff(*req.SoundOff); err != nil {
			writeJSON(w, http.StatusInternalServerError, APIResponse{Success: false, Error: "Could not save the setting"})
			return
		}
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
