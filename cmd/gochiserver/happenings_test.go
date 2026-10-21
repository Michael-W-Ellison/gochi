package main

import (
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/Michael-W-Ellison/gochi/internal/core"
)

func TestHappeningThroughServer(t *testing.T) {
	server := newTestServer(t)
	if err := server.createPet("Mochi", core.SpeciesCat, "", false, false); err != nil {
		t.Fatal(err)
	}
	if rec := post(t, server.handleHappening, `{"choice":"comfort"}`); rec.Code != http.StatusConflict {
		t.Errorf("A choice with nothing happening: %d", rec.Code)
	}
	server.mu.Lock()
	server.pet.Happening = &core.PendingHappening{ID: "thunderstorm", Since: time.Now()}
	server.mu.Unlock()
	if rec := post(t, server.handleHappening, `{"choice":"dance"}`); rec.Code != http.StatusBadRequest {
		t.Errorf("A choice that isn't offered: %d", rec.Code)
	}
	rec := post(t, server.handleHappening, `{"choice":"comfort"}`)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"right":true`) {
		t.Errorf("Comforting Mochi: %d %s", rec.Code, rec.Body.String())
	}

	// Closing it without a choice
	server.mu.Lock()
	server.pet.Happening = &core.PendingHappening{ID: "zoomies", Since: time.Now()}
	server.mu.Unlock()
	if rec := post(t, server.handleHappening, `{"choice":""}`); rec.Code != http.StatusOK || server.pet.CurrentHappening() != nil {
		t.Errorf("Closing a happening: %d", rec.Code)
	}
}
