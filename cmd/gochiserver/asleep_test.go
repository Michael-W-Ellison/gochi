package main

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/Michael-W-Ellison/gochi/internal/core"
	"github.com/Michael-W-Ellison/gochi/pkg/types"
)

// asleep puts the active pet to sleep
func (h *harness) asleep() {
	h.t.Helper()
	h.server.mu.Lock()
	defer h.server.mu.Unlock()
	pet := h.server.pet
	pet.Biology.Vitals.Fatigue, pet.Biology.Vitals.Energy = 0.9, 0.3
	pet.Update(0.01)
	if pet.CurrentBehavior != types.BehaviorSleeping {
		h.t.Fatalf("The pet should be asleep: %s", pet.CurrentBehavior)
	}
}

// Care and activities that wait for a sleeping pet to wake tell the page
// it is asleep, and about when it wakes; food still works
func TestCareForASleepingPet(t *testing.T) {
	h := newHarness(t)
	h.createPet("Mochi", core.SpeciesCat)
	h.asleep()

	var status struct {
		Data struct {
			Behavior    string `json:"behavior"`
			WakeSeconds int    `json:"wake_seconds"`
		} `json:"data"`
	}
	if err := json.Unmarshal(h.do(http.MethodGet, "/api/status", "").Body.Bytes(), &status); err != nil {
		t.Fatal(err)
	}
	if status.Data.Behavior != "Sleeping" || status.Data.WakeSeconds <= 0 {
		t.Errorf("The status should say when it wakes: %+v", status.Data)
	}

	type reply struct {
		Error string `json:"error"`
		Data  struct {
			Asleep      bool `json:"asleep"`
			WakeSeconds int  `json:"wake_seconds"`
		} `json:"data"`
	}
	for _, req := range [][2]string{
		{"/api/interact", `{"action":"pet"}`},
		{"/api/interact", `{"action":"groom"}`},
		{"/api/interact", `{"action":"play"}`},
		{"/api/interact", `{"action":"pet","region":"cheeks"}`},
		{"/api/hideseek/start", `{}`},
		{"/api/explore", `{"place":"park"}`},
	} {
		rec := h.post(req[0], req[1])
		var r reply
		if err := json.Unmarshal(rec.Body.Bytes(), &r); err != nil {
			t.Fatal(err)
		}
		if rec.Code != http.StatusConflict || !r.Data.Asleep {
			t.Errorf("%s %s while asleep: %d %s", req[0], req[1], rec.Code, rec.Body.String())
		}
	}
	h.server.mu.Lock()
	h.server.pet.Biology.Vitals.Nutrition = 0.3
	h.server.mu.Unlock()
	if rec := h.post("/api/interact", `{"action":"feed"}`); rec.Code != http.StatusOK {
		t.Errorf("A sleeping pet can still be fed: %d %s", rec.Code, rec.Body.String())
	}
}
