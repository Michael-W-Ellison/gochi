package hub

import (
	"context"
	"encoding/json"
	"errors"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// testHub returns a hub with join code "secret" and a client for it
func testHub(t *testing.T) (*Hub, *httptest.Server) {
	t.Helper()
	h, err := New(t.TempDir(), "secret")
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(h.Handler())
	t.Cleanup(server.Close)
	return h, server
}

// register registers a player through the API and returns their client
func register(t *testing.T, server *httptest.Server, username string) *Client {
	t.Helper()
	c := NewClient(server.URL, "")
	token, err := c.Register(context.Background(), username, "secret")
	if err != nil {
		t.Fatalf("Register %s: %v", username, err)
	}
	c.Token = token
	return c
}

var ctx = context.Background()

func TestRegistration(t *testing.T) {
	h, server := testHub(t)
	c := NewClient(server.URL, "")
	if status, err := c.Status(ctx); err != nil || !status.RegistrationOpen {
		t.Fatalf("Status: %+v, %v", status, err)
	}
	if _, err := c.Register(ctx, "alice", "wrong"); err == nil || !strings.Contains(err.Error(), "wrong join code") {
		t.Errorf("Wrong join code: %v", err)
	}
	if _, err := c.Register(ctx, "a b", "secret"); err == nil {
		t.Error("Usernames can't have spaces")
	}
	token, err := c.Register(ctx, "Alice", "secret")
	if err != nil || len(token) != 64 {
		t.Fatalf("Register: %q, %v", token, err)
	}
	if _, err := c.Register(ctx, "alice", "secret"); err == nil || !strings.Contains(err.Error(), "taken") {
		t.Errorf("Usernames are unique regardless of case: %v", err)
	}
	if user, err := h.Authenticate(token); err != nil || user != "Alice" {
		t.Errorf("Authenticate: %q, %v", user, err)
	}

	// Only a hash of the token is stored
	data, _ := os.ReadFile(filepath.Join(h.dir, "accounts.json"))
	if strings.Contains(string(data), token) || !strings.Contains(string(data), hashToken(token)) {
		t.Error("The token should be stored only as a hash")
	}

	// Accounts survive a restart
	again, err := New(h.dir, "secret")
	if err != nil {
		t.Fatal(err)
	}
	if user, err := again.Authenticate(token); err != nil || user != "Alice" {
		t.Errorf("After restart: %q, %v", user, err)
	}

	closed, _ := New(t.TempDir(), "")
	if _, err := closed.Register("bob", ""); !errors.Is(err, ErrRegistrationClosed) {
		t.Errorf("Without a join code registration is closed: %v", err)
	}
}

func TestUnauthorized(t *testing.T) {
	_, server := testHub(t)
	c := NewClient(server.URL, "not-a-token")
	if _, err := c.ListPets(ctx); !errors.Is(err, ErrUnauthorized) {
		t.Errorf("Unknown token: %v", err)
	}
}

func TestPetVersions(t *testing.T) {
	_, server := testHub(t)
	alice := register(t, server, "alice")
	key := PetKey("pet_1_Biscuit")
	info := PetInfo{Key: key, Name: "Biscuit", Species: "cat", Stage: "Adult", Alive: true}

	v, err := alice.UploadPetData(ctx, info, json.RawMessage(`{"n":1}`), 0)
	if err != nil || v != 1 {
		t.Fatalf("First upload: %d, %v", v, err)
	}
	// A device that didn't see version 1 conflicts
	if _, err := alice.UploadPetData(ctx, info, json.RawMessage(`{"n":2}`), 0); !errors.As(err, new(ConflictError)) {
		t.Errorf("Stale upload: %v", err)
	}
	if v, err = alice.UploadPetData(ctx, info, json.RawMessage(`{"n":2}`), 1); err != nil || v != 2 {
		t.Fatalf("Second upload: %d, %v", v, err)
	}

	got, save, err := alice.DownloadPetData(ctx, key)
	if err != nil || got.Version != 2 || string(save) != `{"n":2}` || got.Name != "Biscuit" {
		t.Errorf("Download: %+v %s %v", got, save, err)
	}
	pets, err := alice.ListPets(ctx)
	if err != nil || len(pets) != 1 || pets[0].Version != 2 {
		t.Errorf("List: %+v %v", pets, err)
	}

	// Another player can't see it
	bob := register(t, server, "bob")
	if _, _, err := bob.DownloadPetData(ctx, key); !errors.Is(err, ErrNotFound) {
		t.Errorf("Bob reading Alice's pet: %v", err)
	}

	// Deleting leaves a marker so other devices don't upload it again
	if v, err := alice.DeletePetData(ctx, key, 2); err != nil || v != 3 {
		t.Fatalf("Delete: %d %v", v, err)
	}
	got, save, _ = alice.DownloadPetData(ctx, key)
	if !got.Deleted || len(save) != 0 {
		t.Errorf("A deleted pet keeps no save: %+v %s", got, save)
	}

	if _, err := alice.UploadPetData(ctx, PetInfo{Key: "../../etc"}, json.RawMessage(`{}`), 0); err == nil {
		t.Error("Invalid keys should be rejected")
	}
	if _, err := alice.UploadPetData(ctx, PetInfo{Key: PetKey("x")}, json.RawMessage(`not json`), 0); err == nil {
		t.Error("A save must be JSON")
	}
}

func TestNearbyAndFeed(t *testing.T) {
	_, server := testHub(t)
	alice, bob, carol := register(t, server, "alice"), register(t, server, "bob"), register(t, server, "carol")
	for i, c := range []*Client{alice, bob, carol} {
		name := []string{"Biscuit", "Rex", "Mochi"}[i]
		if _, err := c.UploadPetData(ctx, PetInfo{Key: PetKey(name), Name: name, Species: "cat", Alive: true},
			json.RawMessage(`{}`), 0); err != nil {
			t.Fatal(err)
		}
	}

	// Nobody shares yet
	if pets, _ := alice.GetNearbyPets(ctx); len(pets) != 0 {
		t.Errorf("No one shares: %+v", pets)
	}
	alice.EnableSocialFeatures(ctx, "Leeds", true)
	bob.EnableSocialFeatures(ctx, "  leeds ", true)
	carol.EnableSocialFeatures(ctx, "York", true)

	pets, err := alice.GetNearbyPets(ctx)
	if err != nil || len(pets) != 1 || pets[0].Name != "Rex" || pets[0].Owner != "bob" {
		t.Errorf("Alice should see Bob's pet in Leeds only: %+v %v", pets, err)
	}
	if profile, _ := bob.GetProfile(ctx); profile.Area != "leeds" || !profile.Sharing {
		t.Errorf("Bob's profile: %+v", profile)
	}

	// Turning sharing off hides you and your view
	bob.EnableSocialFeatures(ctx, "Leeds", false)
	if pets, _ := alice.GetNearbyPets(ctx); len(pets) != 0 {
		t.Errorf("Bob stopped sharing: %+v", pets)
	}
	if pets, _ := bob.GetNearbyPets(ctx); len(pets) != 0 {
		t.Errorf("A player who doesn't share sees no one: %+v", pets)
	}

	if err := bob.ShareAchievement(ctx, SharedAchievement{PetName: "Rex", Name: "Explorer"}); err == nil {
		t.Error("Only players who share can post achievements")
	}
	if err := alice.ShareAchievement(ctx, SharedAchievement{PetName: "Biscuit", ID: "first_trick", Name: "Good Pet", Icon: "🐾",
		Description: "Learned a first trick"}); err != nil {
		t.Fatal(err)
	}
	feed, err := bob.Feed(ctx)
	if err != nil || len(feed) != 1 || feed[0].Owner != "alice" || feed[0].PetName != "Biscuit" {
		t.Errorf("Feed: %+v %v", feed, err)
	}
}

func TestNormalizeURL(t *testing.T) {
	for in, want := range map[string]string{
		"hub.example.com":          "http://hub.example.com",
		"https://hub.example.com/": "https://hub.example.com",
		" 192.168.1.10:8080 ":      "http://192.168.1.10:8080",
		"http://host:1/gochi/":     "http://host:1/gochi",
	} {
		if got, err := NormalizeURL(in); err != nil || got != want {
			t.Errorf("NormalizeURL(%q) = %q, %v; want %q", in, got, err, want)
		}
	}
	for _, bad := range []string{"", "ftp://x", "http://"} {
		if _, err := NormalizeURL(bad); err == nil {
			t.Errorf("%q should be rejected", bad)
		}
	}
}

func TestStatusRejectsOtherServers(t *testing.T) {
	other := httptest.NewServer(nil) // Answers 404
	defer other.Close()
	if _, err := NewClient(other.URL, "").Status(ctx); err == nil {
		t.Error("A server that isn't a hub should be rejected")
	}
}
