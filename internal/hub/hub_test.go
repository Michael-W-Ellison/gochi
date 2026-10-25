package hub

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// testHub returns a hub with join code "secret" and a client for it. Its
// players all register from this computer, so it takes more new accounts
// from one address than a hub does.
func testHub(t *testing.T) (*Hub, *httptest.Server) {
	t.Helper()
	h, err := New(t.TempDir(), "secret")
	if err != nil {
		t.Fatal(err)
	}
	h.newAccounts = newEventLimiter(1000, newAccountWindow)
	server := httptest.NewServer(h.Handler())
	t.Cleanup(server.Close)
	return h, server
}

// register registers a player through the API and returns their client
func register(t *testing.T, server *httptest.Server) *Client {
	t.Helper()
	c := NewClient(server.URL, "", "")
	token, handle, err := c.Register(context.Background(), "secret", 0)
	if err != nil {
		t.Fatalf("Register: %v", err)
	}
	if !ValidHandle(handle) {
		t.Fatalf("The hub should give a handle, got %q", handle)
	}
	c.Token = token
	return c
}

// handleOf returns a client's handle
func handleOf(t *testing.T, c *Client) string {
	t.Helper()
	profile, err := c.GetProfile(ctx)
	if err != nil {
		t.Fatal(err)
	}
	return profile.Handle
}

var ctx = context.Background()

// rawPost sends a request with any body, as a modified game could
func rawPost(t *testing.T, method, url, token, body string) (int, string) {
	t.Helper()
	req, _ := http.NewRequest(method, url, strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }() // Nothing to report once the body is read
	var out strings.Builder
	buf := make([]byte, 4096)
	n, _ := resp.Body.Read(buf)
	out.Write(buf[:n])
	return resp.StatusCode, out.String()
}

func TestRegistrationGivesHandles(t *testing.T) {
	h, server := testHub(t)
	c := NewClient(server.URL, "", "")
	if status, err := c.Status(ctx); err != nil || !status.RegistrationOpen {
		t.Fatalf("Status: %+v, %v", status, err)
	}
	if _, _, err := c.Register(ctx, "wrong", 0); err == nil || !strings.Contains(err.Error(), "wrong join code") {
		t.Errorf("Wrong join code: %v", err)
	}
	// Players can't choose a name
	if code, _ := rawPost(t, http.MethodPost, server.URL+"/hub/v1/register", "", `{"join_code":"secret","username":"Sam Smith"}`); code != http.StatusBadRequest {
		t.Errorf("A chosen username should be refused: %d", code)
	}
	handles := map[string]bool{}
	for i := 0; i < 20; i++ {
		token, handle, err := c.Register(ctx, "secret", 0)
		if err != nil || len(token) != 64 || !ValidHandle(handle) || handles[handle] {
			t.Fatalf("Register: %q %q %v", token, handle, err)
		}
		handles[handle] = true
		if id, err := h.Authenticate(token); err != nil || !idPattern.MatchString(id) {
			t.Errorf("Authenticate: %q, %v", id, err)
		}
	}

	// Tokens are stored only as hashes, and no name at all
	data, _ := os.ReadFile(filepath.Join(h.dir, "accounts.json"))
	if strings.Contains(string(data), `"username"`) {
		t.Errorf("Accounts shouldn't hold usernames: %s", data)
	}

	closed, _ := New(t.TempDir(), "")
	if _, _, err := closed.Register("", 0); !errors.Is(err, ErrRegistrationClosed) {
		t.Errorf("A hub without a join code is closed: %v", err)
	}
}

// A player's number picks their handle: the same number gives the same
// handle on another hub, and the next free one where it is taken
// A player's number picks their handle on a hub, mixed with the hub's own
// secret: the next free one where it is taken, the same after the hub
// restarts, and (almost always) a different one on another hub, so
// nobody can choose the number for someone else's handle
func TestHandleFromANumber(t *testing.T) {
	dir := t.TempDir()
	h, err := New(dir, "secret")
	if err != nil {
		t.Fatal(err)
	}
	_, handle, err := h.Register("secret", 4821)
	if err != nil || !ValidHandle(handle) {
		t.Fatalf("Register: %q %v", handle, err)
	}
	_, taken, err := h.Register("secret", 4821)
	if err != nil || taken == handle || !ValidHandle(taken) {
		t.Errorf("A taken handle should step to a free one: %q %v", taken, err)
	}
	if _, near, _ := h.Register("secret", 4822); near == handle || near == taken {
		t.Errorf("The next number should give another handle: %q", near)
	}
	if key, err := os.ReadFile(filepath.Join(dir, "handle.key")); err != nil || len(key) != 32 {
		t.Fatalf("The hub's secret should be kept: %d bytes, %v", len(key), err)
	}
	restarted, err := New(dir, "secret")
	if err != nil {
		t.Fatal(err)
	}
	free := func(string) bool { return false }
	same, _ := seededHandle(restarted.handleKey, 4821, free)
	if same != handle {
		t.Errorf("After a restart the number should still give %q: %q", handle, same)
	}
	differ := 0
	for seed := uint32(1); seed <= 20; seed++ {
		other, err := New(t.TempDir(), "secret")
		if err != nil {
			t.Fatal(err)
		}
		a, _ := seededHandle(h.handleKey, seed, free)
		b, _ := seededHandle(other.handleKey, seed, free)
		if a != b {
			differ++
		}
	}
	if differ < 18 {
		t.Errorf("The same number should give different handles on different hubs: only %d of 20 differ", differ)
	}

	// Every handle can be given, once
	seen := map[string]bool{}
	for i := 0; i < handleCount; i++ {
		h := handleAt(i)
		if !ValidHandle(h) || seen[h] {
			t.Fatalf("Handle %d: %q", i, h)
		}
		seen[h] = true
	}
	if _, err := seededHandle(h.handleKey, 1, func(string) bool { return true }); err == nil {
		t.Error("A hub with every handle taken should say so")
	}
}

// A hub from before numbers refuses the number; the game registers
// without it and gets a random handle
func TestRegisteringWithAnOlderHub(t *testing.T) {
	_, server := testHub(t)
	old := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			JoinCode string `json:"join_code"`
		}
		decoder := json.NewDecoder(r.Body)
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&req); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"error":"invalid request"}`))
			return
		}
		body, _ := json.Marshal(req)
		forward, _ := http.NewRequest(http.MethodPost, server.URL+r.URL.Path, strings.NewReader(string(body)))
		resp, err := http.DefaultClient.Do(forward)
		if err != nil {
			t.Error(err)
			return
		}
		defer func() { _ = resp.Body.Close() }()
		w.WriteHeader(resp.StatusCode)
		buf := make([]byte, 4096)
		n, _ := resp.Body.Read(buf)
		_, _ = w.Write(buf[:n])
	}))
	t.Cleanup(old.Close)
	token, handle, err := NewClient(old.URL, "", "").Register(ctx, "secret", 4821)
	if err != nil || len(token) != 64 || !ValidHandle(handle) {
		t.Errorf("Registering with an older hub: %q %q %v", token, handle, err)
	}
}

func TestValidHandle(t *testing.T) {
	for _, good := range []string{"Blue Otter 42", "Teal Wren 10", "Amber Badger 99"} {
		if !ValidHandle(good) {
			t.Errorf("%q should be valid", good)
		}
	}
	for _, bad := range []string{"", "Sam", "Blue Otter", "Blue Otter 9", "Blue Otter 100", "Blue Sam 42",
		"blue otter 42", "Blue  Otter 42", "Blue Otter 42 hi"} {
		if ValidHandle(bad) {
			t.Errorf("%q should be invalid", bad)
		}
	}
}

func TestOldAccountsAreMigrated(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "pets", "sam_smith"), 0o700); err != nil {
		t.Fatal(err)
	}
	key := PetKey("pet_1")
	if err := os.WriteFile(filepath.Join(dir, "pets", "sam_smith", key+".json"),
		[]byte(`{"key":"`+key+`","version":3,"name":"Biscuit","species":"cat","save":{"x":1}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "accounts.json"), []byte(`[{"username":"Sam_Smith","area":"Leeds",
		"show_nearby":true,"token_hash":"`+hashToken("old-token")+`"}]`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "feed.json"), []byte(`[{"owner":"Sam_Smith","pet_name":"Biscuit",
		"id":"first_trick","name":"call me","description":"555 0100"}]`), 0o600); err != nil {
		t.Fatal(err)
	}

	h, err := New(dir, "secret")
	if err != nil {
		t.Fatal(err)
	}
	id, err := h.Authenticate("old-token")
	if err != nil {
		t.Fatal(err)
	}
	profile, _ := h.GetProfile(id)
	if !ValidHandle(profile.Handle) || profile.ShareAchievements {
		t.Errorf("A migrated account gets a handle and shares nothing: %+v", profile)
	}
	if pets, _ := h.ListPets(id); len(pets) != 1 || pets[0].Version != 3 {
		t.Errorf("Its pets should come along: %+v", pets)
	}
	if _, err := os.Stat(filepath.Join(dir, "pets", "sam_smith")); !os.IsNotExist(err) {
		t.Error("The directory named after the username should be gone")
	}
	data, _ := os.ReadFile(filepath.Join(dir, "accounts.json"))
	for _, gone := range []string{"Sam_Smith", "Leeds", "area"} {
		if strings.Contains(string(data), gone) {
			t.Errorf("accounts.json still holds %q: %s", gone, data)
		}
	}
	if feed := h.Feed(10); len(feed) != 0 {
		t.Errorf("Old feed entries held text players typed and should be dropped: %+v", feed)
	}
}

func TestUnauthorized(t *testing.T) {
	_, server := testHub(t)
	c := NewClient(server.URL, "not-a-token", "")
	if _, err := c.ListPets(ctx); !errors.Is(err, ErrUnauthorized) {
		t.Errorf("Unknown token: %v", err)
	}
}

func TestPetVersions(t *testing.T) {
	_, server := testHub(t)
	alice := register(t, server)
	key := PetKey("pet_1_Biscuit")
	info := PetInfo{Key: key}

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
	if err != nil || got.Version != 2 || string(save) != `{"n":2}` || got.Size == 0 {
		t.Errorf("Download: %+v %s %v", got, save, err)
	}
	pets, err := alice.ListPets(ctx)
	if err != nil || len(pets) != 1 || pets[0].Version != 2 {
		t.Errorf("List: %+v %v", pets, err)
	}

	// Another player can't see it
	bob := register(t, server)
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

func TestPetInfoHoldsNoText(t *testing.T) {
	h, server := testHub(t)
	alice := register(t, server)
	key := PetKey("pet_1")
	// A modified game sending a name and other fields
	code, body := rawPost(t, http.MethodPut, server.URL+"/hub/v1/pets/"+key, alice.Token,
		`{"base_version":0,"info":{"name":"meet me at 4","species":"call 555"},"save":{}}`)
	if code != http.StatusBadRequest {
		t.Errorf("Unknown fields should be refused: %d %s", code, body)
	}
	if _, err := alice.UploadPetData(ctx, PetInfo{Key: key}, json.RawMessage(`{}`), 0); err != nil {
		t.Fatal(err)
	}
	id, _ := h.Authenticate(alice.Token)
	stored, _ := os.ReadFile(filepath.Join(h.petDir(id), key+".json"))
	if strings.Contains(string(stored), `"name"`) || strings.Contains(string(stored), `"species"`) {
		t.Errorf("Stored information should hold no names: %s", stored)
	}
}

func TestAccountQuota(t *testing.T) {
	h, err := New(t.TempDir(), "secret")
	if err != nil {
		t.Fatal(err)
	}
	token, _, _ := h.Register("secret", 0)
	id, _ := h.Authenticate(token)
	save := json.RawMessage(`"` + strings.Repeat("x", MaxSaveBytes-10) + `"`)
	n := MaxAccountBytes / len(save)
	for i := 0; i < n; i++ {
		if _, err := h.PutPet(id, PetInfo{Key: PetKey(string(rune('a' + i)))}, save, 0); err != nil {
			t.Fatalf("Pet %d: %v", i, err)
		}
	}
	if _, err := h.PutPet(id, PetInfo{Key: PetKey("one more")}, save, 0); !errors.Is(err, ErrAccountFull) {
		t.Errorf("Saves beyond the account's space should be refused: %v", err)
	}
	// Replacing a pet with a save of the same size is fine, without a
	// backup when there is no room for one
	if _, err := h.PutPet(id, PetInfo{Key: PetKey("a")}, save, 1); err != nil {
		t.Errorf("Replacing a pet: %v", err)
	}
	if _, err := os.Stat(filepath.Join(h.petDir(id), PetKey("a")+".prev")); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("A full account keeps no backup: %v", err)
	}
	if got := h.accountBytesLocked(id, ""); got > MaxAccountBytes {
		t.Errorf("The account uses %d bytes, more than %d", got, MaxAccountBytes)
	}
}

// Backups count towards the account's space, so it can't hold twice its
// share on the hub's disk
func TestBackupsCountTowardsTheQuota(t *testing.T) {
	h, err := New(t.TempDir(), "secret")
	if err != nil {
		t.Fatal(err)
	}
	token, _, _ := h.Register("secret", 0)
	id, _ := h.Authenticate(token)
	save := json.RawMessage(`"` + strings.Repeat("x", MaxSaveBytes-10) + `"`)
	half := MaxAccountBytes / len(save) / 2
	for i := 0; i < half; i++ {
		key := PetKey(string(rune('a' + i)))
		for version := 0; version < 2; version++ { // Saved again: the first save is kept as a backup
			if _, err := h.PutPet(id, PetInfo{Key: key}, save, version); err != nil {
				t.Fatalf("Pet %d: %v", i, err)
			}
		}
	}
	if got := h.accountBytesLocked(id, ""); got != 2*half*len(save) {
		t.Errorf("Saves and backups: %d bytes, want %d", got, 2*half*len(save))
	}
	if _, err := h.PutPet(id, PetInfo{Key: PetKey("one more")}, save, 0); !errors.Is(err, ErrAccountFull) {
		t.Errorf("With the backups, the account is full: %v", err)
	}
	if h.totalBytes != int64(2*half*len(save)) {
		t.Errorf("The hub's total: %d", h.totalBytes)
	}
	// The totals are rebuilt when the hub starts
	again, err := New(h.dir, "secret")
	if err != nil {
		t.Fatal(err)
	}
	if again.totalBytes != h.totalBytes || again.accountBytesLocked(id, "") != h.accountBytesLocked(id, "") {
		t.Errorf("After a restart: %d and %d bytes", again.totalBytes, again.accountBytesLocked(id, ""))
	}
}

func TestAchievementsAreSharedByID(t *testing.T) {
	_, server := testHub(t)
	alice, bob := register(t, server), register(t, server)

	if err := alice.ShareAchievement(ctx, "first_trick", "cat"); err == nil {
		t.Error("Sharing is off for a new account")
	}
	if _, err := alice.EnableSocialFeatures(ctx, true); err != nil {
		t.Fatal(err)
	}
	if err := alice.ShareAchievement(ctx, "first_trick", "cat"); err != nil {
		t.Fatal(err)
	}
	for _, bad := range []struct{ id, species string }{{"meet me at the park", "cat"}, {"first_trick", "call 555"}} {
		if err := alice.ShareAchievement(ctx, bad.id, bad.species); err == nil {
			t.Errorf("%+v should be refused", bad)
		}
	}
	// A modified game sending its own text
	if code, _ := rawPost(t, http.MethodPost, server.URL+"/hub/v1/achievements", alice.Token,
		`{"id":"first_trick","species":"cat","name":"call me","description":"555 0100"}`); code != http.StatusBadRequest {
		t.Errorf("Text with an achievement should be refused: %d", code)
	}

	feed, err := bob.Feed(ctx)
	if err != nil || len(feed) != 1 {
		t.Fatalf("Feed: %+v %v", feed, err)
	}
	if a := feed[0]; a.Handle != handleOf(t, alice) || a.Name != "Good Pet" || a.Species != "cat" || a.Description == "" {
		t.Errorf("The feed should hold the game's text: %+v", a)
	}
}

func TestNormalizeURL(t *testing.T) {
	for in, want := range map[string]string{
		"hub.example.com":           "https://hub.example.com",
		"https://hub.example.com/":  "https://hub.example.com",
		" 192.168.1.10:8443 ":       "https://192.168.1.10:8443",
		"http://localhost:1/gochi/": "http://localhost:1/gochi",
		"http://127.0.0.1:9000":     "http://127.0.0.1:9000",
	} {
		if got, err := NormalizeURL(in); err != nil || got != want {
			t.Errorf("NormalizeURL(%q) = %q, %v; want %q", in, got, err, want)
		}
	}
	for _, bad := range []string{"", "ftp://x", "http://", "http://hub.example.com", "http://192.168.1.10:8080"} {
		if _, err := NormalizeURL(bad); err == nil {
			t.Errorf("%q should be rejected", bad)
		}
	}
}

func TestStatusRejectsOtherServers(t *testing.T) {
	other := httptest.NewServer(nil) // Answers 404
	defer other.Close()
	if _, err := NewClient(other.URL, "", "").Status(ctx); err == nil {
		t.Error("A server that isn't a hub should be rejected")
	}
}

func TestIndexesSurviveRestart(t *testing.T) {
	dir := t.TempDir()
	h, err := New(dir, "secret")
	if err != nil {
		t.Fatal(err)
	}
	token, handle, _ := h.Register("secret", 0)
	id, _ := h.Authenticate(token)
	key := PetKey("pet_1")
	if _, err := h.PutPet(id, PetInfo{Key: key}, json.RawMessage(`{"big":"save"}`), 0); err != nil {
		t.Fatal(err)
	}
	h, err = New(dir, "secret")
	if err != nil {
		t.Fatal(err)
	}
	if again, _ := h.Authenticate(token); again != id {
		t.Error("The account should survive a restart")
	}
	if profile, _ := h.GetProfile(id); profile.Handle != handle {
		t.Errorf("The handle should survive a restart: %q, want %q", profile.Handle, handle)
	}
	if pets, _ := h.ListPets(id); len(pets) != 1 || pets[0].Version != 1 || pets[0].Size == 0 {
		t.Errorf("Pets after a restart: %+v", pets)
	}
	if _, err := h.PutPet(id, PetInfo{Key: key}, json.RawMessage(`{}`), 0); !errors.Is(err, ErrVersionConflict) {
		t.Errorf("The indexed version should be checked: %v", err)
	}
}

func TestUploadsAreLimited(t *testing.T) {
	h, err := New(t.TempDir(), "secret")
	if err != nil {
		t.Fatal(err)
	}
	var done []func()
	for i := 0; i < maxAccountUploads; i++ {
		d, err := h.startUpload(context.Background(), "a")
		if err != nil {
			t.Fatal(err)
		}
		done = append(done, d)
	}
	if _, err := h.startUpload(context.Background(), "a"); !errors.Is(err, ErrBusy) {
		t.Errorf("An account's third upload at once should wait: %v", err)
	}
	// Across the hub, uploads wait for a place
	for i := maxAccountUploads; i < maxUploads; i++ {
		d, err := h.startUpload(context.Background(), fmt.Sprint("acct", i))
		if err != nil {
			t.Fatal(err)
		}
		done = append(done, d)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	if _, err := h.startUpload(ctx, "b"); !errors.Is(err, ErrBusy) {
		t.Errorf("A full hub should refuse an upload that can't wait: %v", err)
	}
	for _, d := range done {
		d()
	}
	if len(h.uploading) != 0 || len(h.uploads) != 0 {
		t.Errorf("Places should be given back: %v %d", h.uploading, len(h.uploads))
	}
	if d, err := h.startUpload(context.Background(), "a"); err != nil {
		t.Error(err)
	} else {
		d()
	}
}

// Deleted pets don't count toward an account's pets, and only the most
// recent records of deleted pets are kept
func TestDeletedPetsDontCount(t *testing.T) {
	h, err := New(t.TempDir(), "secret")
	if err != nil {
		t.Fatal(err)
	}
	token, _, _ := h.Register("secret", 0)
	id, _ := h.Authenticate(token)
	save := json.RawMessage(`"x"`)
	at := time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)
	put := func(name string, base int) (int, error) {
		at = at.Add(time.Minute)
		return h.PutPet(id, PetInfo{Key: PetKey(name), SavedAt: at}, save, base)
	}
	for i := 0; i < MaxPetsPerAccount; i++ {
		if _, err := put(fmt.Sprint("pet", i), 0); err != nil {
			t.Fatalf("Pet %d: %v", i, err)
		}
	}
	if _, err := put("one more", 0); !errors.Is(err, ErrTooManyPets) {
		t.Errorf("Pet %d: %v", MaxPetsPerAccount+1, err)
	}
	version, err := h.DeletePet(id, PetKey("pet0"), 1)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := put("one more", 0); err != nil {
		t.Errorf("After deleting a pet there is room for another: %v", err)
	}
	if _, err := put("pet0", version); !errors.Is(err, ErrTooManyPets) {
		t.Errorf("A deleted pet uploaded again counts once more: %v", err)
	}

	// Pets saved and deleted over and over leave a bounded number of records
	if _, err := h.DeletePet(id, PetKey("one more"), 1); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < maxDeletedPets+5; i++ {
		name := fmt.Sprint("passing", i)
		if _, err := put(name, 0); err != nil {
			t.Fatalf("Pet %s: %v", name, err)
		}
		if _, err := h.DeletePet(id, PetKey(name), 1); err != nil {
			t.Fatal(err)
		}
	}
	deleted := 0
	for _, info := range h.pets[id] {
		if info.Deleted {
			deleted++
		}
	}
	if deleted != maxDeletedPets {
		t.Errorf("Records of deleted pets: %d, want %d", deleted, maxDeletedPets)
	}
	if _, ok := h.pets[id][PetKey("pet0")]; ok {
		t.Error("The record of the pet deleted longest ago should be forgotten first")
	}
	again, err := New(h.dir, "secret")
	if err != nil {
		t.Fatal(err)
	}
	if len(again.pets[id]) != len(h.pets[id]) {
		t.Errorf("After a restart: %d pets and records, want %d", len(again.pets[id]), len(h.pets[id]))
	}
}
