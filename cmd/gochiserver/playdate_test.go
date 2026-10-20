package main

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/Michael-W-Ellison/gochi/internal/core"
)

// playdateData decodes a playdate response
func playdateData(t *testing.T, body string) PlaydateStatus {
	t.Helper()
	var resp struct {
		Success bool           `json:"success"`
		Data    PlaydateStatus `json:"data"`
		Error   string         `json:"error"`
	}
	if err := json.Unmarshal([]byte(body), &resp); err != nil || !resp.Success {
		t.Fatalf("Playdate request failed: %v %s", err, body)
	}
	return resp.Data
}

// sociableDog makes the active pet a sociable dog, so meetings go well
func sociableDog(t *testing.T, s *Server, name string) {
	t.Helper()
	if err := s.createPet(name, core.SpeciesDog, false, false); err != nil {
		t.Fatal(err)
	}
	tr := s.pet.Personality.Traits
	tr.Extraversion, tr.Agreeableness, tr.Territoriality, tr.Neuroticism = 0.9, 0.9, 0.1, 0.1
	v := s.pet.Biology.Vitals
	v.Energy, v.Fatigue, v.Stress, v.Happiness = 1, 0, 0, 0.7
}

func TestPlaydateBetweenPlayers(t *testing.T) {
	ctx := context.Background()
	hubURL := testHubURL(t)
	alice, bob := cloudServer(t), cloudServer(t)
	sociableDog(t, alice, "Biscuit")
	sociableDog(t, bob, "Rex")
	for name, s := range map[string]*Server{"alice": alice, "bob": bob} {
		if err := s.cloudConnect(ctx, hubURL, name, "code", ""); err != nil {
			t.Fatal(err)
		}
	}

	// Alice starts a playdate; Bob joins with the code
	rec := post(t, alice.handlePlaydate, `{"action":"start","sprite":"🐶"}`)
	started := playdateData(t, rec.Body.String())
	if !started.Active || len(started.Code) != 6 || len(started.Members) != 1 || !started.Members[0].You {
		t.Fatalf("Started: %+v", started)
	}
	if rec := post(t, bob.handlePlaydate, `{"action":"join","code":"nope"}`); rec.Code != http.StatusConflict {
		t.Errorf("Joining with a wrong code: %d %s", rec.Code, rec.Body)
	}
	rec = post(t, bob.handlePlaydate, `{"action":"join","code":"`+strings.ToLower(started.Code)+`","sprite":"🐕"}`)
	joined := playdateData(t, rec.Body.String())
	if len(joined.Members) != 2 || joined.Members[1].Pet.Name != "Rex" || !joined.Members[1].You || joined.Members[0].Sprite != "🐶" {
		t.Fatalf("Joined: %+v", joined)
	}

	// Positions are shared
	post(t, bob.handlePlaydateUpdate, `{"position":{"x":0.5,"y":0,"z":0.2},"sprite":"🐕"}`)
	rec = post(t, alice.handlePlaydateUpdate, `{"position":{"x":-0.5,"y":0,"z":0},"sprite":"🐶"}`)
	state := playdateData(t, rec.Body.String())
	if rex := state.Members[1]; rex.Position.X != 0.5 || rex.Position.Z != 0.2 || rex.Pet.Name != "Rex" || rex.Relation != "" {
		t.Errorf("Alice should see Rex where Bob put him: %+v", rex)
	}

	// Alice's pet plays with Bob's
	rec = post(t, alice.handlePlaydatePlay, `{"owner":"bob"}`)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "Biscuit and Rex played together happily") {
		t.Fatalf("Playing: %d %s", rec.Code, rec.Body)
	}
	if alice.pet.RelationTo(bob.pet.ID) == "" {
		t.Error("Biscuit should know Rex")
	}
	if rec := post(t, alice.handlePlaydatePlay, `{"owner":"alice"}`); rec.Code != http.StatusNotFound {
		t.Errorf("Playing with your own pet: %d", rec.Code)
	}

	// Bob's game records it for Rex on its next update
	rec = post(t, bob.handlePlaydateUpdate, `{"position":{"x":0.5,"y":0,"z":0.2}}`)
	state = playdateData(t, rec.Body.String())
	if bob.pet.RelationTo(alice.pet.ID) == "" {
		t.Error("Rex should know Biscuit")
	}
	if len(state.Log) != 2 || !strings.Contains(state.Log[0], "played together") || !strings.Contains(state.Log[1], "Rex and Biscuit are") {
		t.Errorf("Bob's log: %+v", state.Log)
	}
	if state.Members[0].Relation == "" {
		t.Errorf("Bob should see how Rex gets along with Biscuit: %+v", state.Members[0])
	}
	// ...once
	before := bob.pet.Relations()[0]
	post(t, bob.handlePlaydateUpdate, `{"position":{"x":0.5,"y":0,"z":0.2}}`)
	if after := bob.pet.Relations()[0]; after != before {
		t.Errorf("A play should be recorded once: %+v then %+v", before, after)
	}

	// Leaving
	rec = post(t, bob.handlePlaydate, `{"action":"leave"}`)
	if left := playdateData(t, rec.Body.String()); left.Active {
		t.Errorf("Bob should have left: %+v", left)
	}
	rec = post(t, alice.handlePlaydateUpdate, `{"position":{"x":0,"y":0,"z":0}}`)
	if state := playdateData(t, rec.Body.String()); len(state.Members) != 1 {
		t.Errorf("Alice should be alone: %+v", state.Members)
	}
}

func TestPlaydateNeedsHubAndPet(t *testing.T) {
	s := cloudServer(t)
	if rec := post(t, s.handlePlaydate, `{"action":"start"}`); rec.Code != http.StatusConflict {
		t.Errorf("A playdate needs a hub: %d %s", rec.Code, rec.Body)
	}
	if rec := post(t, s.handlePlaydateUpdate, `{}`); rec.Code != http.StatusConflict {
		t.Errorf("Updating without a playdate: %d %s", rec.Code, rec.Body)
	}
	if err := s.cloudConnect(context.Background(), testHubURL(t), "alice", "code", ""); err != nil {
		t.Fatal(err)
	}
	if rec := post(t, s.handlePlaydate, `{"action":"start"}`); rec.Code != http.StatusConflict || !strings.Contains(rec.Body.String(), "need a pet") {
		t.Errorf("A playdate needs a pet: %d %s", rec.Code, rec.Body)
	}
}

func TestPlaydateEndsWhenPetChanges(t *testing.T) {
	ctx := context.Background()
	s := cloudServer(t)
	sociableDog(t, s, "Biscuit")
	if err := s.cloudConnect(ctx, testHubURL(t), "alice", "code", ""); err != nil {
		t.Fatal(err)
	}
	post(t, s.handlePlaydate, `{"action":"start"}`)
	sociableDog(t, s, "Rex") // A new active pet
	if rec := post(t, s.handlePlaydateUpdate, `{}`); rec.Code != http.StatusConflict || !strings.Contains(rec.Body.String(), "changed pets") {
		t.Errorf("Changing pets should leave the playdate: %d %s", rec.Code, rec.Body)
	}
	if s.playdates.session != nil {
		t.Error("The playdate should be over")
	}
}
