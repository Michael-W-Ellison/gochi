package main

import (
	"net/http"
	"strings"
	"testing"

	"github.com/Michael-W-Ellison/gochi/internal/core"
)

func TestHouseholdTimeThroughServer(t *testing.T) {
	server, pipSlot, pip := family(t)
	before := pip.Biology.GetAgeInDays()
	old := core.NewDigitalPet("Old", "test")
	old.Biology.Processes.Age = 1e6
	savePet(t, server.store, "old", old)

	server.liveInHousehold(10)
	saved, _, _, err := server.store.Load(pipSlot)
	if err != nil {
		t.Fatal(err)
	}
	if got := saved.Biology.GetAgeInDays(); got != before+10 {
		t.Errorf("Pip's age after 10 days in the household: %.1f, was %.1f", got, before)
	}
	if gone, _, _, _ := server.store.Load("old"); gone.IsAlive() {
		t.Error("A very old saved pet should reach the end of its life")
	}
	server.mu.Lock()
	popups := append([]popup{}, server.popups...)
	server.mu.Unlock()
	if len(popups) == 0 || !strings.HasPrefix(popups[len(popups)-1].ID, "died_") {
		t.Errorf("The player hears of it: %+v", popups)
	}

	// A paused pet stays as it is
	if rec := post(t, server.handlePausePet, `{"slot":"`+pipSlot+`","paused":true}`); rec.Code != http.StatusOK {
		t.Fatalf("Pausing Pip: %d", rec.Code)
	}
	server.liveInHousehold(10)
	saved, _, _, _ = server.store.Load(pipSlot)
	if got := saved.Biology.GetAgeInDays(); got != before+10 || !saved.IsHouseholdPaused() {
		t.Errorf("A paused pet doesn't age: %.1f", got)
	}
	list, _ := server.store.List()
	for _, entry := range list {
		if entry.Slot == pipSlot && !entry.Paused {
			t.Error("The pet list should show Pip is paused")
		}
	}
}
