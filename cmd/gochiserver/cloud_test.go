package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Michael-W-Ellison/gochi/internal/core"
	"github.com/Michael-W-Ellison/gochi/internal/hub"
	"github.com/Michael-W-Ellison/gochi/pkg/types"
)

// cloudServer returns a test server with a cloud connection kept in its
// own directory
func cloudServer(t *testing.T) *Server {
	t.Helper()
	s := newTestServer(t)
	c, err := newCloud(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	s.cloud = c
	return s
}

// testHubURL starts a hub with join code "code"
func testHubURL(t *testing.T) string {
	t.Helper()
	h, err := hub.New(t.TempDir(), "code")
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(h.Handler())
	t.Cleanup(server.Close)
	return server.URL
}

func TestCloudSyncBetweenDevices(t *testing.T) {
	ctx := context.Background()
	hubURL := testHubURL(t)
	laptop, phone := cloudServer(t), cloudServer(t)

	if err := laptop.createPet("Biscuit", core.SpeciesCat, false, false); err != nil {
		t.Fatal(err)
	}
	if err := laptop.cloudConnect(ctx, hubURL, "code", ""); err != nil {
		t.Fatal(err)
	}
	report, err := laptop.cloudSync(ctx)
	if err != nil || len(report.Uploaded) != 1 || report.Uploaded[0] != "Biscuit" {
		t.Fatalf("First sync should upload Biscuit: %+v %v", report, err)
	}

	// Nothing changed: nothing to do, even though syncing saves the pet again
	if report, _ := laptop.cloudSync(ctx); report.summary() != "Everything is up to date." {
		t.Errorf("Second sync: %s", report.summary())
	}

	// The phone signs in with the account key and gets Biscuit
	key := accountKey(laptop)
	if err := phone.cloudConnect(ctx, hubURL, "", "not-a-key"); err == nil {
		t.Error("An unknown account key should be refused")
	}
	if err := phone.cloudConnect(ctx, hubURL, "", key); err != nil {
		t.Fatal(err)
	}
	if phone.cloudStatus().Handle != laptop.cloudStatus().Handle || !hub.ValidHandle(phone.cloudStatus().Handle) {
		t.Errorf("Both devices should have the account's handle: %q %q", phone.cloudStatus().Handle, laptop.cloudStatus().Handle)
	}
	if report, err := phone.cloudSync(ctx); err != nil || len(report.Downloaded) != 1 {
		t.Fatalf("The phone should download Biscuit: %+v %v", report, err)
	}
	saved, err := phone.store.List()
	if err != nil || len(saved) != 1 || saved[0].Name != "Biscuit" {
		t.Fatalf("The phone should now have Biscuit: %+v %v", saved, err)
	}

	// Play on the phone, then the laptop picks it up
	pet, _, _, err := phone.store.Load(saved[0].Slot)
	if err != nil {
		t.Fatal(err)
	}
	pet.FriendshipPoints = 777
	pet.LastUpdateAt = time.Now().Add(time.Minute) // Played after the laptop
	if err := phone.store.SavePet(saved[0].Slot, pet); err != nil {
		t.Fatal(err)
	}
	if report, _ := phone.cloudSync(ctx); len(report.Uploaded) != 1 {
		t.Fatalf("The phone should upload its changes: %s", report.summary())
	}
	if report, _ := laptop.cloudSync(ctx); len(report.Downloaded) != 1 {
		t.Fatalf("The laptop should download them: %s", report.summary())
	}
	laptop.mu.RLock()
	points := laptop.pet.FriendshipPoints
	laptop.mu.RUnlock()
	if points != 777 {
		t.Errorf("The laptop's active pet should be reloaded with the phone's changes, points %d", points)
	}

	// Changed on both: the copy played most recently wins
	laptop.mu.Lock()
	laptop.pet.FriendshipPoints = 1
	laptop.pet.LastUpdateAt = time.Now().Add(-time.Hour) // An old game left running
	laptop.mu.Unlock()
	pet, _, _, _ = phone.store.Load(saved[0].Slot)
	pet.FriendshipPoints = 2
	pet.LastUpdateAt = time.Now().Add(2 * time.Minute)
	phone.store.SavePet(saved[0].Slot, pet)
	phone.cloudSync(ctx)
	if report, _ := laptop.cloudSync(ctx); len(report.Conflicts) != 1 {
		t.Fatalf("Both changed: expected a conflict, got %s", report.summary())
	}
	laptop.mu.RLock()
	points = laptop.pet.FriendshipPoints
	laptop.mu.RUnlock()
	if points != 2 {
		t.Errorf("The more recently played copy should win, points %d", points)
	}
}

func TestCloudDeletion(t *testing.T) {
	ctx := context.Background()
	hubURL := testHubURL(t)
	laptop, phone := cloudServer(t), cloudServer(t)
	laptop.createPet("Biscuit", core.SpeciesCat, false, false)
	laptop.cloudConnect(ctx, hubURL, "code", "")
	laptop.cloudSync(ctx)
	phone.cloudConnect(ctx, hubURL, "", accountKey(laptop))
	phone.cloudSync(ctx)

	// Deleting on the laptop marks it deleted on the hub
	slot := laptop.slot
	laptop.mu.RLock()
	id := string(laptop.pet.ID)
	laptop.mu.RUnlock()
	laptop.store.Delete(slot)
	laptop.cloudForget(ctx, id)

	report, err := phone.cloudSync(ctx)
	if err != nil || len(report.DeletedElsewhere) != 1 {
		t.Fatalf("The phone should hear Biscuit was deleted: %s %v", report.summary(), err)
	}
	if saved, _ := phone.store.List(); len(saved) != 1 {
		t.Error("The phone keeps its copy; the player decides whether to delete it")
	}
	if report, _ := phone.cloudSync(ctx); report.summary() != "Everything is up to date." {
		t.Errorf("A pet deleted elsewhere is no longer synced: %s", report.summary())
	}
}

func TestCloudSocialThroughServer(t *testing.T) {
	ctx := context.Background()
	hubURL := testHubURL(t)
	alice, bob := cloudServer(t), cloudServer(t)
	alice.createPet("Biscuit", core.SpeciesCat, false, false)
	bob.createPet("Rex", core.SpeciesDog, false, false)
	for _, p := range []struct {
		s    *Server
		name string
	}{{alice, "alice"}, {bob, "bob"}} {
		body := `{"hub_url":"` + hubURL + `","join_code":"code"}`
		if rec := post(t, p.s.handleCloudConnect, body); rec.Code != http.StatusOK {
			t.Fatalf("Connect %s: %d %s", p.name, rec.Code, rec.Body.String())
		}
		if rec := post(t, p.s.handleCloudProfile, `{"share_achievements":true}`); rec.Code != http.StatusOK {
			t.Fatalf("Profile: %d %s", rec.Code, rec.Body.String())
		}
	}
	// The status never holds the account key
	rec := httptest.NewRecorder()
	alice.handleCloud(rec, httptest.NewRequest(http.MethodGet, "/api/cloud", nil))
	if strings.Contains(rec.Body.String(), accountKey(alice)) {
		t.Errorf("The cloud status shouldn't hold the account key: %s", rec.Body.String())
	}

	// An achievement earned in the update loop is shared through the hub
	ctxShare, cancel := context.WithCancel(ctx)
	defer cancel()
	go bob.shareLoop(ctxShare)
	bob.mu.Lock()
	bob.pet.Souvenirs = []string{"pebble"}
	bob.announceAchievementsLocked()
	bob.mu.Unlock()
	deadline := time.Now().Add(5 * time.Second)
	for {
		feed, err := alice.cloudFeed(ctx)
		if err == nil && len(feed) > 0 {
			if feed[0].Handle != bob.cloudStatus().Handle || feed[0].Species != "dog" || feed[0].Name != "Treasure Hunter" {
				t.Errorf("Feed: %+v", feed)
			}
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("The achievement was not shared: %v %v", feed, err)
		}
		time.Sleep(20 * time.Millisecond)
	}

	if events, _ := bob.events.Since(0, bob.pet.ID); len(events) == 0 || events[0].Type != core.EventAchievement {
		t.Errorf("The achievement should be an event: %+v", events)
	}

	if rec := post(t, alice.handleCloudDisconnect, ""); rec.Code != http.StatusOK || alice.cloudStatus().Connected {
		t.Errorf("Disconnect: %d %s", rec.Code, rec.Body.String())
	}
	if rec := post(t, alice.handleCloudSync, ""); rec.Code != http.StatusConflict {
		t.Errorf("Syncing while disconnected: %d", rec.Code)
	}
}

func TestSaveFileRecordsHistory(t *testing.T) {
	s := newTestServer(t)
	if err := s.createPet("Biscuit", core.SpeciesCat, false, false); err != nil {
		t.Fatal(err)
	}
	s.store.timeScale, s.store.autosaveSeconds = "game", 60
	s.pet.Biology.Vitals.Hydration = 0.3
	s.pet.Interact(types.InteractionDrinking, 1)
	s.pet.TricksLearned = map[string]bool{"sit": true}
	s.pet.CheckAchievements()
	if _, err := s.saveNow(); err != nil {
		t.Fatal(err)
	}

	saveData, _, err := s.store.dm.LoadLatest(s.slot)
	if err != nil {
		t.Fatal(err)
	}
	if len(saveData.CareHistory) < 2 || saveData.CareHistory[0].ActionType != "Drinking" {
		t.Errorf("Care history: %+v", saveData.CareHistory)
	}
	if len(saveData.Achievements) != 1 || saveData.Achievements[0].ID != "first_trick" {
		t.Errorf("Achievements: %+v", saveData.Achievements)
	}
	if len(saveData.Unlocks.Locations) != 1 || saveData.Unlocks.Locations[0] != "park" ||
		len(saveData.Unlocks.Tricks) != 1 || saveData.Unlocks.PetVariants[0] != "cat" {
		t.Errorf("Unlocks: %+v", saveData.Unlocks)
	}
	if saveData.Statistics.TotalInteractions != 1 || saveData.Preferences.TimeScale != "game" ||
		saveData.Preferences.AutoSaveInterval != 60 {
		t.Errorf("Statistics %+v, preferences %+v", saveData.Statistics, saveData.Preferences)
	}
}

// accountKey returns a server's hub account key
func accountKey(s *Server) string {
	s.cloud.mu.Lock()
	defer s.cloud.mu.Unlock()
	return s.cloud.state.Token
}

func TestAccountKeyOnlyOnThisComputer(t *testing.T) {
	s := cloudServer(t)
	s.parental, _ = loadParentalControls("")
	if err := s.cloudConnect(context.Background(), testHubURL(t), "code", ""); err != nil {
		t.Fatal(err)
	}
	local := httptest.NewRequest(http.MethodPost, "/api/cloud/key", strings.NewReader(`{}`))
	local.RemoteAddr = "127.0.0.1:5000"
	rec := httptest.NewRecorder()
	s.handleCloudKey(rec, local)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), accountKey(s)) {
		t.Errorf("This computer should get the key: %d %s", rec.Code, rec.Body)
	}
	phone := httptest.NewRequest(http.MethodPost, "/api/cloud/key", strings.NewReader(`{}`))
	phone.RemoteAddr = "192.168.1.20:5000"
	rec = httptest.NewRecorder()
	s.handleCloudKey(rec, phone)
	if rec.Code != http.StatusForbidden {
		t.Errorf("Another device shouldn't get the key: %d", rec.Code)
	}
}
