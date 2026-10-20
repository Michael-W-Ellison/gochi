package hub

import (
	"encoding/json"
	"errors"
	"math"
	"net/http"
	"strings"
	"testing"
	"time"
)

// card returns a pet card for tests
func card(name string) json.RawMessage {
	return json.RawMessage(`{"name":"` + name + `"}`)
}

func TestPlaydateThroughAPI(t *testing.T) {
	_, server := testHub(t)
	alice, bob, carol := register(t, server, "alice"), register(t, server, "bob"), register(t, server, "carol")

	started, err := alice.StartPlaydate(ctx, card("Mochi"))
	if err != nil {
		t.Fatal(err)
	}
	if len(started.Code) != playdateCodeLength || started.Host != "alice" || len(started.Members) != 1 {
		t.Fatalf("Started playdate: %+v", started)
	}
	for _, r := range started.Code {
		if !strings.ContainsRune(playdateAlphabet, r) {
			t.Errorf("Code %q uses %q", started.Code, r)
		}
	}

	// Codes can be typed in lower case with spaces around them
	joined, err := bob.JoinPlaydate(ctx, " "+strings.ToLower(started.Code)+" ", card("Rex"))
	if err != nil {
		t.Fatal(err)
	}
	if len(joined.Members) != 2 || joined.Members[1].Owner != "bob" || joined.Members[1].Slot != 1 || joined.You != "bob" {
		t.Errorf("Joined playdate: %+v", joined)
	}
	if _, err := carol.JoinPlaydate(ctx, "ZZZZZZ", card("Tom")); !errors.Is(err, ErrNoPlaydate) {
		t.Errorf("Joining an unknown playdate: %v", err)
	}
	if _, err := carol.UpdatePlaydate(ctx, started.Code, nil, Offset{}, 0); !errors.Is(err, ErrNotInPlaydate) {
		t.Errorf("Updating a playdate you haven't joined: %v", err)
	}

	// Positions are shared, kept near the center
	if _, err := bob.UpdatePlaydate(ctx, started.Code, nil, Offset{X: 0.5, Z: 99}, 0); err != nil {
		t.Fatal(err)
	}
	state, err := alice.UpdatePlaydate(ctx, started.Code, card("Mochi!"), Offset{X: -0.5}, 0)
	if err != nil {
		t.Fatal(err)
	}
	if p := state.Members[1].Position; p.X != 0.5 || p.Z != maxPlaydateOffset {
		t.Errorf("Bob's position = %+v", p)
	}
	if string(state.Members[0].Card) != string(card("Mochi!")) {
		t.Errorf("Alice's card should be updated, got %s", state.Members[0].Card)
	}

	// Events reach everyone, once
	event, err := bob.PostPlaydateEvent(ctx, started.Code, PlaydateEvent{Kind: "play", To: "ALICE", Text: "Rex played with Mochi",
		Data: json.RawMessage(`{"quality":0.7}`)})
	if err != nil {
		t.Fatal(err)
	}
	if event.From != "bob" || event.To != "alice" || event.Seq != 1 {
		t.Errorf("Event = %+v", event)
	}
	if _, err := bob.PostPlaydateEvent(ctx, started.Code, PlaydateEvent{Kind: "play", To: "carol"}); err == nil {
		t.Error("Events can only be sent to members")
	}
	state, _ = alice.UpdatePlaydate(ctx, started.Code, nil, Offset{}, 0)
	if len(state.Events) != 1 || state.Last != 1 || string(state.Events[0].Data) != `{"quality":0.7}` {
		t.Errorf("Alice should see the event: %+v", state)
	}
	if state, _ = alice.UpdatePlaydate(ctx, started.Code, nil, Offset{}, state.Last); len(state.Events) != 0 {
		t.Errorf("Events already seen shouldn't come again: %+v", state.Events)
	}

	// Leaving
	if err := bob.LeavePlaydate(ctx, started.Code); err != nil {
		t.Fatal(err)
	}
	if state, _ = alice.UpdatePlaydate(ctx, started.Code, nil, Offset{}, 0); len(state.Members) != 1 {
		t.Errorf("Bob should have left: %+v", state.Members)
	}
	if err := alice.LeavePlaydate(ctx, started.Code); err != nil {
		t.Fatal(err)
	}
	if _, err := bob.JoinPlaydate(ctx, started.Code, card("Rex")); !errors.Is(err, ErrNoPlaydate) {
		t.Errorf("An empty playdate should end: %v", err)
	}
}

func TestPlaydateLimits(t *testing.T) {
	h, err := New(t.TempDir(), "secret")
	if err != nil {
		t.Fatal(err)
	}
	clock := time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)
	h.now = func() time.Time { return clock }
	var names []string
	for _, name := range []string{"p1", "p2", "p3", "p4", "p5", "p6", "p7"} {
		if _, err := h.Register(name, "secret"); err != nil {
			t.Fatal(err)
		}
		names = append(names, name)
	}

	if _, err := h.StartPlaydate("p1", json.RawMessage(`not json`)); err == nil {
		t.Error("A card must be JSON")
	}
	if _, err := h.StartPlaydate("p1", json.RawMessage(`"`+strings.Repeat("x", maxCardBytes)+`"`)); err == nil {
		t.Error("A card must be small")
	}
	pd, err := h.StartPlaydate("p1", card("a"))
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range names[1:MaxPlaydateMembers] {
		if _, err := h.JoinPlaydate(name, pd.Code, card(name)); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := h.JoinPlaydate("p7", pd.Code, card("p7")); !errors.Is(err, ErrPlaydateFull) {
		t.Errorf("A seventh player should find it full: %v", err)
	}

	// Positions that aren't numbers are ignored
	state, err := h.UpdatePlaydate("p2", pd.Code, nil, Offset{X: math.NaN(), Y: math.Inf(1)}, 0)
	if err != nil {
		t.Fatal(err)
	}
	if p := state.Members[1].Position; p.X != 0 || p.Y != 0 {
		t.Errorf("Position = %+v", p)
	}

	// Players who stop updating leave; their slot is given to the next
	// player to join
	clock = clock.Add(memberTimeout / 2)
	h.UpdatePlaydate("p1", pd.Code, nil, Offset{}, 0)
	clock = clock.Add(memberTimeout/2 + time.Second)
	state, err = h.JoinPlaydate("p7", pd.Code, card("p7"))
	if err != nil {
		t.Fatal(err)
	}
	if len(state.Members) != 2 || state.Members[1].Owner != "p7" || state.Members[1].Slot != 1 {
		t.Errorf("Only p1 should remain, with p7 joining in slot 1: %+v", state.Members)
	}

	// Starting a playdate leaves the old one
	if _, err := h.StartPlaydate("p7", card("p7")); err != nil {
		t.Fatal(err)
	}
	if state, _ := h.UpdatePlaydate("p1", pd.Code, nil, Offset{}, 0); len(state.Members) != 1 {
		t.Errorf("p7 should have left the first playdate: %+v", state.Members)
	}
}

func TestPlaydateRateLimitIsPerPlayer(t *testing.T) {
	_, server := testHub(t)
	alice, bob := register(t, server, "alice"), register(t, server, "bob")
	pd, err := alice.StartPlaydate(ctx, card("Mochi"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := bob.JoinPlaydate(ctx, pd.Code, card("Rex")); err != nil {
		t.Fatal(err)
	}
	// Both players share an address; each has their own limit
	for i := 0; i < requestsPerMinute-5; i++ {
		if _, err := alice.UpdatePlaydate(ctx, pd.Code, nil, Offset{}, 0); err != nil {
			t.Fatalf("Request %d: %v", i, err)
		}
	}
	if _, err := bob.UpdatePlaydate(ctx, pd.Code, nil, Offset{}, 0); err != nil {
		t.Errorf("Bob shouldn't share Alice's limit: %v", err)
	}
	// Requests without a valid token are limited by address
	stranger := NewClient(server.URL, "not-a-token")
	req, _ := http.NewRequest(http.MethodGet, server.URL+"/hub/v1/status", nil)
	req.Header.Set("Authorization", "Bearer not-a-token")
	for i := 0; i < requestsPerMinute; i++ {
		resp, err := stranger.HTTP.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
	}
	resp, err := stranger.HTTP.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusTooManyRequests {
		t.Errorf("An unknown token shouldn't escape the address limit: %d", resp.StatusCode)
	}
}
