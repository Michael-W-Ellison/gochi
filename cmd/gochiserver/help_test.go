package main

import (
	"net/http"
	"slices"
	"testing"

	"github.com/Michael-W-Ellison/gochi/internal/core"
)

func TestTipsAndSoundSetting(t *testing.T) {
	server := newTestServer(t)
	server.house = newTestHousehold(t, "")
	server.parental, _ = loadParentalControls("")
	if err := server.createPet("Mochi", core.SpeciesCat, "", false, false); err != nil {
		t.Fatal(err)
	}
	if rec := post(t, server.handleTipSeen, `{"id":"tour"}`); rec.Code != http.StatusOK {
		t.Errorf("The tour seen: %d", rec.Code)
	}
	if rec := post(t, server.handleTipSeen, `{"id":"anything else"}`); rec.Code != http.StatusBadRequest {
		t.Errorf("A tip that doesn't exist: %d", rec.Code)
	}
	status := statusOf(t, server)
	if seen, _ := status["tips_seen"].([]interface{}); len(seen) != 1 || seen[0] != "tour" || status["sound_off"] != false {
		t.Errorf("Tips seen: %v, sound off %v", status["tips_seen"], status["sound_off"])
	}

	// A parent turns sound off; it merges like the other settings
	if rec := post(t, server.handlePrivacy, `{"sound_off":true}`); rec.Code != http.StatusOK {
		t.Fatalf("Turning sound off: %d %s", rec.Code, rec.Body.String())
	}
	if statusOf(t, server)["sound_off"] != true {
		t.Error("Sound should be off")
	}
	other := newTestHousehold(t, "")
	other.seeTip("shop")
	server.house.merge(other.snapshot())
	if d := server.house.snapshot(); !d.SoundOff || !slices.Equal(d.TipsSeen, []string{"tour", "shop"}) {
		t.Errorf("After a merge: sound off %v, tips %v", d.SoundOff, d.TipsSeen)
	}
}
