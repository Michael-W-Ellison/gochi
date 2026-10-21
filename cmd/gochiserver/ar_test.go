package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Michael-W-Ellison/gochi/internal/ar"
	"github.com/Michael-W-Ellison/gochi/internal/core"
)

func TestARThroughServer(t *testing.T) {
	server := newTestServer(t)
	if err := server.createPet("Biscuit", core.SpeciesCat, "", false, false); err != nil {
		t.Fatal(err)
	}

	// Placing needs a tracking session
	if rec := post(t, server.handleARPlace, `{}`); rec.Code != http.StatusConflict {
		t.Errorf("Placing without a session: expected 409, got %d", rec.Code)
	}
	if rec := post(t, server.handleARSession, `{"action":"start"}`); rec.Code != http.StatusOK {
		t.Fatalf("Starting a session: %d %s", rec.Code, rec.Body.String())
	}
	deadline := time.Now().Add(3 * time.Second)
	for {
		server.arMu.Lock()
		state := server.arServiceLocked().GetSessionState()
		server.arMu.Unlock()
		if state == ar.ARSessionStateTracking {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("The AR session never started tracking")
		}
		time.Sleep(50 * time.Millisecond)
	}

	// The client reports a floor, and the pet is placed on it
	if rec := post(t, server.handleARPlace, `{}`); rec.Code != http.StatusConflict {
		t.Errorf("Placing with no floor found: expected 409, got %d", rec.Code)
	}
	rec := post(t, server.handleARPlane, `{"plane":{"id":"floor","type":"horizontal","center":{"x":0,"y":0,"z":-1},"extent":{"x":2,"y":0,"z":2}}}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("Reporting a plane: %d %s", rec.Code, rec.Body.String())
	}
	if rec := post(t, server.handleARPlace, `{}`); rec.Code != http.StatusOK {
		t.Fatalf("Placing the pet: %d %s", rec.Code, rec.Body.String())
	}

	// A tap away from the pet misses; a tap on it pets it
	rec = post(t, server.handleARGesture, `{"type":"tap","position":{"x":5,"y":0,"z":5}}`)
	if !strings.Contains(rec.Body.String(), `"hit":false`) {
		t.Errorf("A tap away from the pet should miss: %s", rec.Body.String())
	}
	before := server.pet.TotalInteractions
	rec = post(t, server.handleARGesture, `{"type":"tap","position":{"x":0,"y":0,"z":-1}}`)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"interaction":"pet"`) {
		t.Fatalf("A tap on the pet: %d %s", rec.Code, rec.Body.String())
	}
	if server.pet.TotalInteractions != before+1 {
		t.Error("A tap on the pet should pet it")
	}

	// The scene shows the pet with its behavior
	rec = httptest.NewRecorder()
	server.handleARState(rec, httptest.NewRequest(http.MethodGet, "/api/ar", nil))
	if !strings.Contains(rec.Body.String(), string(server.pet.ID)) || !strings.Contains(rec.Body.String(), `"behavior"`) {
		t.Errorf("The scene should include the placed pet: %s", rec.Body.String())
	}

	// A tracked walk counts as exercise
	if rec := post(t, server.handleARExercise, `{"action":"stop"}`); rec.Code != http.StatusConflict {
		t.Errorf("Stopping with no walk: expected 409, got %d", rec.Code)
	}
	clock := time.Now()
	server.arMu.Lock()
	server.arServiceLocked().SetClock(func() time.Time {
		clock = clock.Add(100 * time.Second) // Time to walk each stretch
		return clock
	})
	server.arMu.Unlock()
	post(t, server.handleARExercise, `{"action":"start"}`)
	for _, x := range []string{"0", "100", "250"} {
		post(t, server.handleARExercise, `{"action":"position","position":{"x":`+x+`,"y":0,"z":0}}`)
	}
	rec = post(t, server.handleARExercise, `{"action":"stop"}`)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "Biscuit walked 250 m with you.") {
		t.Errorf("Finishing a walk: %d %s", rec.Code, rec.Body.String())
	}

	// A pet that is no longer active leaves the scene
	if err := server.createPet("Pepper", core.SpeciesCat, "", false, false); err != nil {
		t.Fatal(err)
	}
	rec = httptest.NewRecorder()
	server.handleARState(rec, httptest.NewRequest(http.MethodGet, "/api/ar", nil))
	if !strings.Contains(rec.Body.String(), `"placed_pets":{}`) {
		t.Errorf("Switching pets should remove the old pet from the scene: %s", rec.Body.String())
	}
}

func TestStoppingARSessionRemovesPet(t *testing.T) {
	server := newTestServer(t)
	if err := server.createPet("Biscuit", core.SpeciesCat, "", false, false); err != nil {
		t.Fatal(err)
	}
	post(t, server.handleARSession, `{"action":"start"}`)
	deadline := time.Now().Add(3 * time.Second)
	for {
		server.arMu.Lock()
		state := server.arServiceLocked().GetSessionState()
		server.arMu.Unlock()
		if state == ar.ARSessionStateTracking {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("The AR session never started tracking")
		}
		time.Sleep(50 * time.Millisecond)
	}
	if rec := post(t, server.handleARPlace, `{"position":{"x":0,"y":0,"z":-1}}`); rec.Code != http.StatusOK {
		t.Fatalf("Placing the pet: %d %s", rec.Code, rec.Body.String())
	}

	post(t, server.handleARSession, `{"action":"stop"}`)
	rec := httptest.NewRecorder()
	server.handleARState(rec, httptest.NewRequest(http.MethodGet, "/api/ar", nil))
	if !strings.Contains(rec.Body.String(), `"placed_pets":{}`) || !strings.Contains(rec.Body.String(), `"session_state":"inactive"`) {
		t.Errorf("Stopping the session should remove the pet: %s", rec.Body.String())
	}
}
