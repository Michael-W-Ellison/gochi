package main

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/Michael-W-Ellison/gochi/internal/core"
)

// The route sweep sends every page route, in every state the game can be
// in, requests that are empty, malformed or nonsense. Whatever arrives, a
// route must answer with JSON and a client error, never crash the server
// or fail with a server error. (A route that assumed an active pet once
// crashed when there was none.)

// sweepBodies are the request bodies each route is sent
var sweepBodies = []string{
	``,
	`{}`,
	`not json`,
	`[]`,
	`null`,
	`{"slot":"../../etc/passwd","id":"../household","to_slot":"..","friend_slot":"/","child_slot":"\u0000"}`,
	`{"place":"moon","game":"chess","choice":"dance","show":"cats","level":99,"routine":["fly","fly","fly"],` +
		`"souvenir":"gold","items":["a","b","c","d","e","f","g"],"activity":"sing","change":"lava"}`,
	`{"slot":7,"place":{},"level":"high","quality":"good","items":"x","partner_slots":"x","lit":"yes","paused":3}`,
	`{"quality":-1e308,"amount":-5,"points":999999999999,"level":-1,"spot":-1,"intensity":1e308}`,
	`{"name":"` + strings.Repeat("x", 500) + `","species":"dragon","breed":"unicorn"}`,
}

// sweepState sets up one state of the game to sweep
type sweepState struct {
	name  string
	setup func(h *harness)
}

// withPets creates an active pet and a second, saved one
func withPets(h *harness) {
	h.createPet("Biscuit", core.SpeciesDog)
	h.createPet("Mochi", core.SpeciesCat) // Biscuit is now saved; Mochi is active
}

// activePet returns the active pet
func (h *harness) activePet() *core.DigitalPet {
	h.server.mu.RLock()
	defer h.server.mu.RUnlock()
	return h.server.pet
}

var sweepStates = []sweepState{
	{"no pet", func(h *harness) {}},
	{"young pet", withPets},
	{"grown pet with a past", func(h *harness) {
		withPets(h)
		pet := h.activePet()
		pet.Biology.Processes.Age = 300
		pet.FriendshipPoints = 5000
		pet.Souvenirs = []string{"pebble", "acorn", "pearl"}
		pet.TricksLearned = map[string]bool{"sit": true, "come": true, "touch": true}
		pet.TrickSkills = map[string]float64{"sit": 1, "come": 1, "touch": 1}
		pet.Happening = &core.PendingHappening{ID: "thunderstorm", Since: time.Now()}
		h.tick(5 * time.Second)
	}},
	{"dead pet", func(h *harness) {
		withPets(h)
		h.activePet().Biology.IsAlive = false
	}},
	{"pet away exploring", func(h *harness) {
		withPets(h)
		if rec := h.post("/api/explore", `{"place":"park"}`); rec.Code != http.StatusOK {
			h.t.Fatalf("Exploring: %s", rec.Body.String())
		}
	}},
	{"hibernating pet", func(h *harness) {
		withPets(h)
		h.activePet().Hibernating = true
	}},
	{"game paused", func(h *harness) {
		withPets(h)
		h.post("/api/pause", `{"paused":true}`)
	}},
}

func TestRouteSweep(t *testing.T) {
	for _, state := range sweepStates {
		t.Run(state.name, func(t *testing.T) {
			h := newHarness(t)
			state.setup(h)
			succeeded := 0
			for _, r := range h.server.routes() {
				for _, method := range []string{http.MethodGet, http.MethodPost, http.MethodDelete} {
					bodies := sweepBodies
					if method != http.MethodPost {
						bodies = []string{``}
					}
					for _, body := range bodies {
						rec := h.do(method, r.pattern, body)
						where := method + " " + r.pattern + " " + body
						switch rec.Code {
						case http.StatusOK:
							succeeded++
						case http.StatusUnsupportedMediaType, http.StatusMisdirectedRequest:
							t.Fatalf("%s: the guard turned the test's request away: %s", where, rec.Body.String())
						}
						if rec.Code >= 500 {
							t.Errorf("%s: server error %d %s", where, rec.Code, rec.Body.String())
						}
						if !json.Valid(rec.Body.Bytes()) {
							t.Errorf("%s: not JSON: %.200s", where, rec.Body.String())
						}
						h.tick(time.Second) // Time passes between requests
						h.checkPet(where)
						h.checkHousehold(where)
					}
				}
			}
			if succeeded < 20 {
				t.Errorf("Only %d requests succeeded; the sweep isn't reaching the routes", succeeded)
			}
		})
	}
}
