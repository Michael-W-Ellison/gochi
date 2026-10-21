package hub

import (
	"errors"
	"math"
	"net/http"
	"strings"
	"testing"
	"time"
)

// card returns a pet card request for tests
func card(pet string) CardRequest {
	return CardRequest{PetKey: PetKey(pet), Species: "dog", Sociability: 0.8, Look: "idle"}
}

func TestPlaydateThroughAPI(t *testing.T) {
	_, server := testHub(t)
	alice, bob, carol := register(t, server), register(t, server), register(t, server)
	aliceHandle, bobHandle := handleOf(t, alice), handleOf(t, bob)

	started, err := alice.StartPlaydate(ctx, card("Biscuit"))
	if err != nil {
		t.Fatal(err)
	}
	if len(started.Code) != playdateCodeLength || started.Host != aliceHandle || len(started.Members) != 1 {
		t.Fatalf("Started playdate: %+v", started)
	}
	for _, r := range started.Code {
		if !strings.ContainsRune(playdateAlphabet, r) {
			t.Errorf("Code %q uses %q", started.Code, r)
		}
	}
	// The hub names the pet; the card's ID means nothing outside the hub
	me := started.Members[0].Card
	if me.Name != aliceHandle+"'s dog" || !strings.HasPrefix(me.ID, "playmate_") || strings.Contains(me.ID, PetKey("Biscuit")) {
		t.Errorf("Card: %+v", me)
	}

	joined, err := bob.JoinPlaydate(ctx, " "+strings.ToLower(started.Code)+" ", card("Rex"))
	if err != nil {
		t.Fatal(err)
	}
	if len(joined.Members) != 2 || joined.Members[1].Handle != bobHandle || joined.Members[1].Slot != 1 || joined.You != bobHandle {
		t.Errorf("Joined playdate: %+v", joined)
	}
	if _, err := carol.JoinPlaydate(ctx, "ZZZZZZ", card("Tom")); !errors.Is(err, ErrNoPlaydate) {
		t.Errorf("Joining an unknown playdate: %v", err)
	}
	if _, err := carol.UpdatePlaydate(ctx, started.Code, nil, Offset{}, 0); !errors.Is(err, ErrNotInPlaydate) {
		t.Errorf("Updating a playdate you haven't joined: %v", err)
	}

	// Positions are shared on the floor, rounded, and move at most
	// maxPlaydateMove at a time
	if _, err := bob.UpdatePlaydate(ctx, started.Code, nil, Offset{X: 0.3123, Y: 2, Z: 99}, 0); err != nil {
		t.Fatal(err)
	}
	happy := card("Biscuit")
	happy.Look = "happy"
	state, err := alice.UpdatePlaydate(ctx, started.Code, &happy, Offset{X: -0.5}, 0)
	if err != nil {
		t.Fatal(err)
	}
	if p := state.Members[1].Position; math.Abs(p.X-0.3) > 1e-9 || p.Y != 0 || p.Z != maxPlaydateMove {
		t.Errorf("Bob's position = %+v", p)
	}
	if state.Members[0].Card.Look != "happy" {
		t.Errorf("Alice's card should be updated: %+v", state.Members[0].Card)
	}

	// Bob's pet plays with Alice's; the hub writes the event
	event, err := bob.ReportPlay(ctx, started.Code, PlayReport{To: aliceHandle, Quality: 0.7, Bond: 3})
	if err != nil {
		t.Fatal(err)
	}
	if event.From != bobHandle || event.To != aliceHandle || event.Seq != 1 || event.Bond != 1 ||
		event.ToPet != me.ID || event.FromCard.Name != bobHandle+"'s dog" ||
		event.Text != bobHandle+"'s dog and "+aliceHandle+"'s dog played together happily!" {
		t.Errorf("Event = %+v", event)
	}
	if _, err := bob.ReportPlay(ctx, started.Code, PlayReport{To: handleOf(t, carol), Quality: 0.5}); err == nil {
		t.Error("Plays are only with members")
	}
	if _, err := bob.ReportPlay(ctx, started.Code, PlayReport{To: bobHandle, Quality: 0.5}); err == nil {
		t.Error("A pet can't play with itself")
	}
	state, _ = alice.UpdatePlaydate(ctx, started.Code, nil, Offset{}, 0)
	if len(state.Events) != 1 || state.Last != 1 {
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

func TestPlaydatesCarryNoText(t *testing.T) {
	_, server := testHub(t)
	alice := register(t, server)
	for name, bad := range map[string]CardRequest{
		"species": {PetKey: PetKey("a"), Species: "meet me at 4", Look: "idle"},
		"look":    {PetKey: PetKey("a"), Species: "cat", Look: "call 555 0100"},
		"reason":  {PetKey: PetKey("a"), Species: "cat", Look: "idle", Unavailable: "at the park"},
		"pet key": {PetKey: "Biscuit", Species: "cat", Look: "idle"},
		"outfit":  {PetKey: PetKey("a"), Species: "cat", Look: "idle", Outfit: map[string]string{"head": "meet me at 4"}},
		"outfit slot": {PetKey: PetKey("a"), Species: "cat", Look: "idle",
			Outfit: map[string]string{"call 555 0100": "beret"}},
		"outfit slot mismatch": {PetKey: PetKey("a"), Species: "cat", Look: "idle",
			Outfit: map[string]string{"body": "beret"}},
		"breed":         {PetKey: PetKey("a"), Species: "cat", Look: "idle", Breed: "meet me at 4"},
		"breed species": {PetKey: PetKey("a"), Species: "cat", Look: "idle", Breed: "labrador"},
	} {
		if _, err := alice.StartPlaydate(ctx, bad); !strings.Contains(errString(err), "invalid pet card") {
			t.Errorf("A card with a bad %s should be refused: %v", name, err)
		}
	}
	nan := card("a")
	nan.Sociability = math.NaN()
	if _, err := makeCard("x", "Blue Otter 42", nan); err == nil {
		t.Error("A card whose sociability isn't a number should be refused")
	}
	dressed := card("Biscuit")
	dressed.Outfit = map[string]string{"head": "beret", "neck": "bow_tie"}
	dressed.Species, dressed.Breed = "cat", "siamese"
	pd, err := alice.StartPlaydate(ctx, dressed)
	if err != nil {
		t.Fatal(err)
	}
	// A modified game sending extra fields or text
	for path, body := range map[string]string{
		"/hub/v1/playdates":                        `{"card":{"pet_key":"` + PetKey("a") + `","species":"cat","look":"idle","name":"Sam"}}`,
		"/hub/v1/playdates/" + pd.Code + "/play":   `{"to":"x","quality":0.5,"bond":0,"text":"hello"}`,
		"/hub/v1/playdates/" + pd.Code + "/events": `{"kind":"note","text":"hello"}`,
	} {
		if code, _ := rawPost(t, http.MethodPost, server.URL+path, alice.Token, body); code == http.StatusOK || code == http.StatusCreated {
			t.Errorf("%s with text should be refused: %d", path, code)
		}
	}
}

// errString returns an error's text, or ""
func errString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

func TestPlaydateLimits(t *testing.T) {
	h, err := New(t.TempDir(), "secret")
	if err != nil {
		t.Fatal(err)
	}
	clock := time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)
	h.now = func() time.Time { return clock }
	var ids []string
	for i := 0; i < 7; i++ {
		token, _, err := h.Register("secret")
		if err != nil {
			t.Fatal(err)
		}
		id, _ := h.Authenticate(token)
		ids = append(ids, id)
	}

	pd, err := h.StartPlaydate(ids[0], card("p0"))
	if err != nil {
		t.Fatal(err)
	}
	for i, id := range ids[1:MaxPlaydateMembers] {
		if _, err := h.JoinPlaydate(id, pd.Code, card(string(rune('a'+i)))); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := h.JoinPlaydate(ids[6], pd.Code, card("p6")); !errors.Is(err, ErrPlaydateFull) {
		t.Errorf("A seventh player should find it full: %v", err)
	}

	// Positions that aren't numbers are ignored
	state, err := h.UpdatePlaydate(ids[1], pd.Code, nil, Offset{X: math.NaN(), Y: math.Inf(1)}, 0)
	if err != nil {
		t.Fatal(err)
	}
	if p := state.Members[1].Position; p.X != 0 || p.Y != 0 {
		t.Errorf("Position = %+v", p)
	}

	// Players who stop updating leave; their slot is given to the next
	// player to join
	clock = clock.Add(memberTimeout / 2)
	if _, err := h.UpdatePlaydate(ids[0], pd.Code, nil, Offset{}, 0); err != nil {
		t.Fatal(err)
	}
	clock = clock.Add(memberTimeout/2 + time.Second)
	state, err = h.JoinPlaydate(ids[6], pd.Code, card("p6"))
	if err != nil {
		t.Fatal(err)
	}
	if len(state.Members) != 2 || state.Members[1].Slot != 1 {
		t.Errorf("Only the host should remain, with the new player in slot 1: %+v", state.Members)
	}

	// Starting a playdate leaves the old one
	if _, err := h.StartPlaydate(ids[6], card("p6")); err != nil {
		t.Fatal(err)
	}
	if state, _ := h.UpdatePlaydate(ids[0], pd.Code, nil, Offset{}, 0); len(state.Members) != 1 {
		t.Errorf("The player should have left the first playdate: %+v", state.Members)
	}
}

func TestPlaydateRateLimitIsPerPlayer(t *testing.T) {
	_, server := testHub(t)
	alice, bob := register(t, server), register(t, server)
	dressed := card("Biscuit")
	dressed.Outfit = map[string]string{"head": "beret", "neck": "bow_tie"}
	dressed.Species, dressed.Breed = "cat", "siamese"
	pd, err := alice.StartPlaydate(ctx, dressed)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := bob.JoinPlaydate(ctx, pd.Code, card("Rex")); err != nil {
		t.Fatal(err)
	}
	// Both players share an address; each has their own limit
	for i := 0; i < requestsPerMinute-10; i++ {
		if _, err := alice.UpdatePlaydate(ctx, pd.Code, nil, Offset{}, 0); err != nil {
			t.Fatalf("Request %d: %v", i, err)
		}
	}
	if _, err := bob.UpdatePlaydate(ctx, pd.Code, nil, Offset{}, 0); err != nil {
		t.Errorf("Bob shouldn't share Alice's limit: %v", err)
	}
	// Requests without a valid key are limited by address
	stranger := NewClient(server.URL, "not-a-token", "")
	req, _ := http.NewRequest(http.MethodGet, server.URL+"/hub/v1/status", nil)
	req.Header.Set("Authorization", "Bearer not-a-token")
	for i := 0; i < requestsPerMinute; i++ {
		resp, err := stranger.HTTP.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		if err := resp.Body.Close(); err != nil {
			t.Fatal(err)
		}
	}
	resp, err := stranger.HTTP.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	if err := resp.Body.Close(); err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusTooManyRequests {
		t.Errorf("An unknown key shouldn't escape the address limit: %d", resp.StatusCode)
	}
}

// Numbers pass between players only as coarsely as the game needs, so they
// can't carry messages
func TestPlaydateNumbersAreCoarse(t *testing.T) {
	h, err := New(t.TempDir(), "secret")
	if err != nil {
		t.Fatal(err)
	}
	clock := time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)
	h.now = func() time.Time { return clock }
	var ids []string
	for i := 0; i < 2; i++ {
		token, _, err := h.Register("secret")
		if err != nil {
			t.Fatal(err)
		}
		id, _ := h.Authenticate(token)
		ids = append(ids, id)
	}
	secret := card("a")
	secret.Sociability = 0.123456789
	pd, err := h.StartPlaydate(ids[0], secret)
	if err != nil {
		t.Fatal(err)
	}
	state, err := h.JoinPlaydate(ids[1], pd.Code, card("b"))
	if err != nil {
		t.Fatal(err)
	}
	if s := state.Members[0].Card.Sociability; math.Abs(s-0.1) > 1e-9 {
		t.Errorf("Sociability = %v, want 0.1", s)
	}
	event, err := h.ReportPlay(ids[1], pd.Code, PlayReport{To: state.Members[0].Handle, Quality: 0.7777, Bond: -0.3333})
	if err != nil {
		t.Fatal(err)
	}
	if math.Abs(event.Quality-0.8) > 1e-9 || math.Abs(event.Bond+0.3) > 1e-9 {
		t.Errorf("Quality %v, bond %v", event.Quality, event.Bond)
	}

	// Positions change at most about once a second, by at most
	// maxPlaydateMove
	move := func(x float64) Offset {
		state, err := h.UpdatePlaydate(ids[1], pd.Code, nil, Offset{X: x}, 0)
		if err != nil {
			t.Fatal(err)
		}
		return state.Members[1].Position
	}
	if p := move(0.3); math.Abs(p.X-0.3) > 1e-9 {
		t.Errorf("First move = %+v", p)
	}
	if p := move(-0.2); math.Abs(p.X-0.3) > 1e-9 {
		t.Errorf("A second change straight away should wait: %+v", p)
	}
	clock = clock.Add(time.Second)
	if p := move(4); math.Abs(p.X-0.8) > 1e-9 {
		t.Errorf("A far move should go maxPlaydateMove: %+v", p)
	}
}
