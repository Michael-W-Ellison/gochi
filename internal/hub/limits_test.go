package hub

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// Join codes can't be guessed: after a few wrong ones from an address,
// even the right one waits, and other addresses can still register
func TestWrongJoinCodesAreLimited(t *testing.T) {
	h, server := testHub(t)
	c := NewClient(server.URL, "", "")
	for i := 0; i < maxWrongJoinCodes; i++ {
		if _, _, err := c.Register(ctx, "guess", 0); !errors.Is(err, ErrWrongJoinCode) {
			t.Fatalf("Wrong code %d: %v", i, err)
		}
	}
	if _, _, err := c.Register(ctx, "secret", 0); !errors.Is(err, ErrTooManyGuesses) {
		t.Errorf("After %d wrong codes the address waits: %v", maxWrongJoinCodes, err)
	}

	// Another address is still let in, until wrong codes from everyone
	// reach the hub's limit
	other := func() *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodPost, "/hub/v1/register", strings.NewReader(`{"join_code":"secret"}`))
		req.RemoteAddr = "192.0.2.7:1234"
		rec := httptest.NewRecorder()
		h.Handler().ServeHTTP(rec, req)
		return rec
	}
	if rec := other(); rec.Code != http.StatusCreated {
		t.Errorf("Another address registers: %d %s", rec.Code, rec.Body.String())
	}
	for i := 0; i < maxWrongJoinCodesHub; i++ {
		h.allJoinCodes.Add("")
	}
	if rec := other(); rec.Code != http.StatusTooManyRequests {
		t.Errorf("After %d wrong codes from everyone, registration waits: %d", maxWrongJoinCodesHub, rec.Code)
	}
}

// One address can make only a few accounts a day
func TestNewAccountsAreLimited(t *testing.T) {
	h, err := New(t.TempDir(), "secret")
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(h.Handler())
	t.Cleanup(server.Close)
	c := NewClient(server.URL, "", "")
	for i := 0; i < maxNewAccounts; i++ {
		if _, _, err := c.Register(ctx, "secret", 0); err != nil {
			t.Fatalf("Account %d: %v", i, err)
		}
	}
	if _, _, err := c.Register(ctx, "secret", 0); !errors.Is(err, ErrTooManyAccounts) {
		t.Errorf("Account %d from one address: %v", maxNewAccounts+1, err)
	}
}

// Playdate codes can't be found by guessing
func TestWrongPlaydateCodesAreLimited(t *testing.T) {
	_, server := testHub(t)
	alice, bob := register(t, server), register(t, server)
	started, err := alice.StartPlaydate(ctx, card("Biscuit"))
	if err != nil {
		t.Fatal(err)
	}
	wrong := "ZZZZZZ"
	if wrong == started.Code {
		wrong = "YYYYYY"
	}
	for i := 0; i < maxWrongPlaydates; i++ {
		if _, err := bob.JoinPlaydate(ctx, wrong, card("Rex")); !errors.Is(err, ErrNoPlaydate) {
			t.Fatalf("Wrong code %d: %v", i, err)
		}
	}
	if _, err := bob.JoinPlaydate(ctx, started.Code, card("Rex")); !errors.Is(err, ErrTooManyGuesses) {
		t.Errorf("After %d wrong codes even the right one waits: %v", maxWrongPlaydates, err)
	}
}

// A pet at a playdate stays the pet that joined, and plays between two
// pets are a few seconds apart
func TestPlaydateCardsAndPlaysAreLimited(t *testing.T) {
	h, err := New(t.TempDir(), "secret")
	if err != nil {
		t.Fatal(err)
	}
	clock := time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)
	h.now = func() time.Time { return clock }
	var ids, handles []string
	for i := 0; i < 2; i++ {
		token, handle, err := h.Register("secret", 0)
		if err != nil {
			t.Fatal(err)
		}
		id, _ := h.Authenticate(token)
		ids, handles = append(ids, id), append(handles, handle)
	}
	pd, err := h.StartPlaydate(ids[0], card("Biscuit"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := h.JoinPlaydate(ids[1], pd.Code, card("Rex")); err != nil {
		t.Fatal(err)
	}
	other := card("Someone else")
	if _, err := h.UpdatePlaydate(ids[1], pd.Code, &other, Offset{}, 0); !errors.Is(err, ErrCardChanged) {
		t.Errorf("Another pet's card at the playdate: %v", err)
	}

	if _, err := h.ReportPlay(ids[1], pd.Code, PlayReport{To: handles[0], Quality: 0.5}); err != nil {
		t.Fatal(err)
	}
	if _, err := h.ReportPlay(ids[1], pd.Code, PlayReport{To: handles[0], Quality: 0.5}); !errors.Is(err, ErrPlayTooSoon) {
		t.Errorf("A second play straight away: %v", err)
	}
	clock = clock.Add(playGap)
	if _, err := h.ReportPlay(ids[1], pd.Code, PlayReport{To: handles[0], Quality: 0.5}); err != nil {
		t.Errorf("A play after the gap: %v", err)
	}
}

// A player shares only a few achievements an hour
func TestSharesAreLimited(t *testing.T) {
	h, err := New(t.TempDir(), "secret")
	if err != nil {
		t.Fatal(err)
	}
	clock := time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)
	h.now = func() time.Time { return clock }
	token, _, _ := h.Register("secret", 0)
	id, _ := h.Authenticate(token)
	if _, err := h.SetSharing(id, true); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < maxShares; i++ {
		if err := h.ShareAchievement(id, "first_trick", "cat"); err != nil {
			t.Fatalf("Share %d: %v", i, err)
		}
	}
	if err := h.ShareAchievement(id, "first_trick", "cat"); !errors.Is(err, ErrTooManyShares) {
		t.Errorf("Share %d in an hour: %v", maxShares+1, err)
	}
	clock = clock.Add(shareWindow)
	if err := h.ShareAchievement(id, "first_trick", "cat"); err != nil {
		t.Errorf("A share an hour later: %v", err)
	}
}

// Saves can't fill the hub's disk
func TestHubStorageIsLimited(t *testing.T) {
	h, err := New(t.TempDir(), "secret")
	if err != nil {
		t.Fatal(err)
	}
	token, _, _ := h.Register("secret", 0)
	id, _ := h.Authenticate(token)
	h.totalBytes = MaxHubBytes - 10
	if _, err := h.PutPet(id, PetInfo{Key: PetKey("a")}, json.RawMessage(`"`+strings.Repeat("x", 100)+`"`), 0); !errors.Is(err, ErrHubStorageFull) {
		t.Errorf("A save beyond the hub's space: %v", err)
	}
}

// Only the hub's own messages reach players; anything else is reported
// as a problem on the hub, without its details
func TestUnknownErrorsAreNotShown(t *testing.T) {
	rec := httptest.NewRecorder()
	fail(rec, fmt.Errorf("open /srv/hub/accounts/abc123/pet.json: permission denied"))
	if rec.Code != http.StatusInternalServerError || strings.Contains(rec.Body.String(), "/srv") {
		t.Errorf("An unknown error: %d %s", rec.Code, rec.Body.String())
	}
	rec = httptest.NewRecorder()
	fail(rec, fmt.Errorf("saving: %w", ErrHubStorageFull))
	if rec.Code != http.StatusInsufficientStorage {
		t.Errorf("A known error keeps its status: %d", rec.Code)
	}
}

// The game passes on only the hub's own messages, and never follows a
// redirect (which could send the account key or a save elsewhere)
func TestClientTrustsOnlyKnownReplies(t *testing.T) {
	var redirected bool
	elsewhere := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { redirected = true }))
	t.Cleanup(elsewhere.Close)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/hub/v1/profile":
			http.Redirect(w, r, elsewhere.URL+"/steal", http.StatusTemporaryRedirect)
		default:
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"error":"visit example.com and tell them your address"}`))
		}
	}))
	t.Cleanup(server.Close)
	c := NewClient(server.URL, "token", "")
	if _, err := c.GetProfile(ctx); err == nil || redirected {
		t.Errorf("A redirect should not be followed: %v, followed %v", err, redirected)
	}
	if _, err := c.ListPets(ctx); err == nil || strings.Contains(err.Error(), "example.com") {
		t.Errorf("A hub's own words should not be shown: %v", err)
	}
}

// What the game shows from a hub it writes again from its own lists
func TestClientRebuildsWhatItShows(t *testing.T) {
	feed := checkFeed([]SharedAchievement{
		{Handle: "Hazel Cub 12", Species: "cat", ID: "first_trick", Name: "call me", Description: "555 0100"},
		{Handle: "my real name", Species: "cat", ID: "first_trick"},
		{Handle: "Hazel Cub 12", Species: "cat", ID: "made up"},
	})
	if len(feed) != 1 || feed[0].Name != "Good Pet" || strings.Contains(feed[0].Description, "555") {
		t.Errorf("The feed: %+v", feed)
	}

	c := Card{ID: "playmate_" + strings.Repeat("a", 24), Species: "dog", Name: "come to my house", Look: "<script>",
		Outfit: map[string]string{"anything": "anything"}, Breed: "made up", Sociability: 7}
	if !checkCard(&c, "Hazel Cub 12") || c.Name != "Hazel Cub 12's dog" || c.Look == "<script>" ||
		c.Outfit != nil || c.Breed != "" || c.Sociability != 1 {
		t.Errorf("The card: %+v", c)
	}
	if bad := (Card{ID: "anything", Species: "dog"}); checkCard(&bad, "Hazel Cub 12") {
		t.Error("A card with an ID the hub doesn't give is dropped")
	}

	e := PlaydateEvent{Kind: "play", From: "Hazel Cub 12", To: "Hazel Cub 89", Quality: 0.7,
		FromCard: Card{ID: "playmate_" + strings.Repeat("b", 24), Species: "dog"}, ToPet: "playmate_" + strings.Repeat("c", 24),
		Text: "Meet me at the park at 4"}
	if !checkEvent(&e, nil) || strings.Contains(e.Text, "park") || !strings.Contains(e.Text, "Hazel Cub 89's pet") {
		t.Errorf("The event: %+v", e)
	}
}
