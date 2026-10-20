package main

import (
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/Michael-W-Ellison/gochi/internal/ai"
	"github.com/Michael-W-Ellison/gochi/internal/core"
)

func TestParentPIN(t *testing.T) {
	dir := t.TempDir()
	p, err := loadParentalControls(dir)
	if err != nil {
		t.Fatal(err)
	}
	if p.Locked() || p.Check("") != nil {
		t.Fatal("Without a PIN everything is allowed")
	}
	if err := p.SetPIN("", "12"); err != errBadPIN {
		t.Errorf("A short PIN: %v", err)
	}
	if err := p.SetPIN("", "2468"); err != nil {
		t.Fatal(err)
	}

	// Kept, hashed, across restarts
	p, _ = loadParentalControls(dir)
	if !p.Locked() || p.Check("2468") != nil || p.Check("") != errParentPIN || p.Check("1111") != errWrongPIN {
		t.Error("The PIN should be needed and checked after a restart")
	}
	if err := p.SetPIN("1111", ""); err != errWrongPIN {
		t.Errorf("Removing the PIN needs the PIN: %v", err)
	}

	// Too many wrong tries lock it for a while
	clock := time.Now()
	p.now = func() time.Time { return clock }
	p.failures = 0
	for i := 0; i < maxPINTries; i++ {
		p.Check("0000")
	}
	if err := p.Check("2468"); err != errPINLockedOut {
		t.Errorf("After %d wrong PINs even the right one waits: %v", maxPINTries, err)
	}
	clock = clock.Add(pinLockout + time.Second)
	if err := p.Check("2468"); err != nil {
		t.Errorf("After the wait: %v", err)
	}

	if err := p.SetPIN("2468", ""); err != nil || p.Locked() {
		t.Errorf("Removing the PIN: %v", err)
	}
}

func TestParentPINGuardsSocialFeatures(t *testing.T) {
	s := cloudServer(t)
	s.parental, _ = loadParentalControls("")
	s.owner = &ai.OwnerSchedule{}
	if err := s.createPet("Biscuit", core.SpeciesCat, false, false); err != nil {
		t.Fatal(err)
	}
	hubURL := testHubURL(t)

	if rec := post(t, s.handlePrivacy, `{"new_pin":"2468"}`); rec.Code != http.StatusOK {
		t.Fatalf("Setting the PIN: %d %s", rec.Code, rec.Body)
	}
	connect := `{"hub_url":"` + hubURL + `","username":"kid","join_code":"code"`
	if rec := post(t, s.handleCloudConnect, connect+`}`); rec.Code != http.StatusForbidden ||
		!strings.Contains(rec.Body.String(), "parent_pin_needed") {
		t.Errorf("Connecting without the PIN: %d %s", rec.Code, rec.Body)
	}
	if rec := post(t, s.handleCloudConnect, connect+`,"parent_pin":"2468"}`); rec.Code != http.StatusOK {
		t.Fatalf("Connecting with the PIN: %d %s", rec.Code, rec.Body)
	}
	for name, try := range map[string]func() int{
		"sharing":          func() int { return post(t, s.handleCloudProfile, `{"area":"Leeds","show_nearby":true}`).Code },
		"a playdate":       func() int { return post(t, s.handlePlaydate, `{"action":"start"}`).Code },
		"privacy":          func() int { return post(t, s.handlePrivacy, `{"learn_schedule":false}`).Code },
		"removing the PIN": func() int { return post(t, s.handlePrivacy, `{"remove_pin":true}`).Code },
		"disconnecting":    func() int { return post(t, s.handleCloudDisconnect, `{}`).Code },
	} {
		if code := try(); code != http.StatusForbidden {
			t.Errorf("Changing %s without the PIN: %d", name, code)
		}
	}
	if rec := post(t, s.handleCloudProfile, `{"area":"Leeds","show_nearby":true,"parent_pin":"2468"}`); rec.Code != http.StatusOK {
		t.Errorf("Sharing with the PIN: %d %s", rec.Code, rec.Body)
	}
	if st := s.cloudStatus(); !st.ShowNearby || st.ShareFeats {
		t.Errorf("Only nearby pets should be shared: %+v", st)
	}
	// Leaving a playdate never needs the PIN
	if rec := post(t, s.handlePlaydate, `{"action":"leave"}`); rec.Code != http.StatusOK {
		t.Errorf("Leaving a playdate: %d", rec.Code)
	}
}

func TestScheduleLearningSwitch(t *testing.T) {
	s := newTestServer(t)
	s.parental, _ = loadParentalControls("")
	s.owner = &ai.OwnerSchedule{}
	s.markActiveLocked(time.Now())
	if s.owner.Visits != 1 {
		t.Fatal("Presence should be learned by default")
	}
	status := getJSON(t, s.handlePrivacy, "/api/privacy")
	if status["learn_schedule"] != true || status["parent_lock"] != false {
		t.Errorf("Privacy: %v", status)
	}
	if rec := post(t, s.handlePrivacy, `{"learn_schedule":false,"erase_schedule":true}`); rec.Code != http.StatusOK {
		t.Fatalf("%d %s", rec.Code, rec.Body)
	}
	s.markActiveLocked(time.Now().Add(2 * time.Hour))
	if s.owner.Visits != 0 || s.owner.Learning() {
		t.Errorf("Learning should be off and erased: %d visits", s.owner.Visits)
	}
}
