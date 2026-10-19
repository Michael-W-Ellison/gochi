package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Michael-W-Ellison/gochi/internal/simulation"
	"github.com/Michael-W-Ellison/gochi/pkg/types"
)

func statusOf(t *testing.T, server *Server) map[string]interface{} {
	t.Helper()
	rec := httptest.NewRecorder()
	server.handleStatus(rec, httptest.NewRequest(http.MethodGet, "/api/status", nil))
	var resp struct {
		Data map[string]interface{} `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	return resp.Data
}

func TestTrainAndPerformThroughServer(t *testing.T) {
	server := newTestServer(t)
	if err := server.createPet("Biscuit", false, false); err != nil {
		t.Fatal(err)
	}

	if rec := post(t, server.handleTrick(true), `{"trick":"sit"}`); rec.Code != http.StatusConflict {
		t.Errorf("Performing an unlearned trick: status %d", rec.Code)
	}
	if rec := post(t, server.handleTrick(false), `{"trick":"fly"}`); rec.Code != http.StatusBadRequest {
		t.Errorf("Unknown trick: status %d", rec.Code)
	}

	learned := false
	for i := 0; i < 20 && !learned; i++ {
		server.pet.Biology.Vitals.Energy = 1.0
		rec := post(t, server.handleTrick(false), `{"trick":"sit"}`)
		if rec.Code != http.StatusOK {
			t.Fatalf("Train: status %d, body %s", rec.Code, rec.Body.String())
		}
		learned = strings.Contains(rec.Body.String(), `"just_learned":true`)
	}
	if !learned {
		t.Fatal("Sit should be learned through the server")
	}

	rec := post(t, server.handleTrick(true), `{"trick":"sit"}`)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "Biscuit sits down neatly.") {
		t.Errorf("Perform: status %d, body %s", rec.Code, rec.Body.String())
	}

	tricks := statusOf(t, server)["tricks"].([]interface{})
	if first := tricks[0].(map[string]interface{}); first["id"] != "sit" || first["learned"] != true {
		t.Errorf("Status should show Sit learned: %+v", first)
	}

	server.pet.Biology.Vitals.Energy = 0.1
	server.pet.Biology.Vitals.Nutrition = 0.9
	if rec := post(t, server.handleTrick(false), `{"trick":"paw"}`); rec.Code != http.StatusConflict ||
		!strings.Contains(rec.Body.String(), "too tired") {
		t.Errorf("Training a tired pet: status %d, body %s", rec.Code, rec.Body.String())
	}
}

func TestHideAndSeekThroughServer(t *testing.T) {
	server := newTestServer(t)
	if err := server.createPet("Biscuit", false, false); err != nil {
		t.Fatal(err)
	}

	if rec := post(t, server.handleHideSeekGuess, `{"spot":0}`); rec.Code != http.StatusConflict {
		t.Errorf("Guess without a game: status %d", rec.Code)
	}

	rec := post(t, server.handleHideSeekStart, "")
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"spots":[`) {
		t.Fatalf("Start: status %d, body %s", rec.Code, rec.Body.String())
	}
	if statusOf(t, server)["hide_and_seek"] != true {
		t.Error("Status should show the game in progress")
	}

	// Guess until the game ends
	var data map[string]interface{}
	for spot := 0; spot < 4; spot++ {
		rec := post(t, server.handleHideSeekGuess, `{"spot":`+string(rune('0'+spot))+`}`)
		var resp struct {
			Data map[string]interface{} `json:"data"`
		}
		json.Unmarshal(rec.Body.Bytes(), &resp)
		data = resp.Data
		if data["over"] == true {
			break
		}
	}
	if data["over"] != true || data["spot"] == "" || data["text"] == "" {
		t.Errorf("Game should end with the hiding spot revealed: %+v", data)
	}
}

func TestExploreThroughServer(t *testing.T) {
	server := newTestServer(t)
	server.awayTimeout = 0
	if err := server.createPet("Biscuit", false, false); err != nil {
		t.Fatal(err)
	}

	rec := post(t, server.handleExplore, "")
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"exploring_seconds":120`) {
		t.Fatalf("Explore: status %d, body %s", rec.Code, rec.Body.String())
	}

	if rec := post(t, server.handleInteract, `{"action":"feed"}`); rec.Code != http.StatusConflict ||
		!strings.Contains(rec.Body.String(), "out exploring") {
		t.Errorf("Feeding while exploring: status %d, body %s", rec.Code, rec.Body.String())
	}
	if s := statusOf(t, server); s["exploring"] != true || s["activity"] != "is exploring" {
		t.Errorf("Status should show the trip: %+v", s)
	}

	// Two real minutes pass
	start := time.Now()
	for elapsed := time.Duration(0); elapsed <= 125*time.Second; elapsed += time.Second {
		server.tick(start.Add(elapsed))
	}

	// The outcome is random: only a treasure adds a souvenir
	s := statusOf(t, server)
	adventure, _ := s["last_adventure"].(map[string]interface{})
	if s["exploring"] != false || adventure == nil || adventure["text"] == "" {
		t.Fatalf("Pet should be home with an adventure: exploring=%v adventure=%+v", s["exploring"], adventure)
	}
	wantSouvenirs := 0
	if adventure["outcome"] == "treasure" {
		wantSouvenirs = 1
	}
	if souvenirs := s["souvenirs"].([]interface{}); len(souvenirs) != wantSouvenirs {
		t.Errorf("Outcome %v: status lists %d souvenirs, want %d", adventure["outcome"], len(souvenirs), wantSouvenirs)
	}
}

func TestRecallAndQuitThroughServer(t *testing.T) {
	server := newTestServer(t)
	if err := server.createPet("Biscuit", false, false); err != nil {
		t.Fatal(err)
	}

	if rec := post(t, server.handleRecall, ""); rec.Code != http.StatusConflict ||
		!strings.Contains(rec.Body.String(), "already home") {
		t.Errorf("Recall at home: status %d, body %s", rec.Code, rec.Body.String())
	}
	post(t, server.handleExplore, "")
	rec := post(t, server.handleRecall, "")
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "brings nothing back") {
		t.Errorf("Recall: status %d, body %s", rec.Code, rec.Body.String())
	}
	if s := statusOf(t, server); s["exploring"] != false || s["last_adventure"] != nil {
		t.Errorf("Recalled pet should be home with no adventure: %+v", s)
	}

	if rec := post(t, server.handleHideSeekQuit, ""); rec.Code != http.StatusConflict {
		t.Errorf("Quit without a game: status %d", rec.Code)
	}
	post(t, server.handleHideSeekStart, "")
	rec = post(t, server.handleHideSeekQuit, "")
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "Maybe next time") {
		t.Errorf("Quit: status %d, body %s", rec.Code, rec.Body.String())
	}
	if statusOf(t, server)["hide_and_seek"] != false {
		t.Error("Game should be over")
	}
}

func TestHealThroughServer(t *testing.T) {
	server := newTestServer(t)
	if err := server.createPet("Biscuit", false, false); err != nil {
		t.Fatal(err)
	}
	server.pet.Injured = true

	if s := statusOf(t, server); s["request"].(map[string]interface{})["action"] != "medic" {
		t.Errorf("Injured pet should ask to be healed: %+v", s["request"])
	}
	if rec := post(t, server.handleExplore, ""); rec.Code != http.StatusConflict ||
		!strings.Contains(rec.Body.String(), "is hurt") {
		t.Errorf("Exploring while injured: status %d, body %s", rec.Code, rec.Body.String())
	}
	if rec := post(t, server.handleInteract, `{"action":"medic"}`); rec.Code != http.StatusOK {
		t.Fatalf("Heal: status %d, body %s", rec.Code, rec.Body.String())
	}
	if server.pet.Injured {
		t.Error("Medical care should heal the injury")
	}
}

func TestMessagesUsePetName(t *testing.T) {
	server := newTestServer(t)
	if err := server.createPet("Biscuit", false, false); err != nil {
		t.Fatal(err)
	}
	server.pet.HibernationEnabled = true

	server.pet.FriendshipPoints = 100
	post(t, server.handleBuy, `{"item":"balloon"}`)
	if rec := post(t, server.handleBuy, `{"item":"balloon"}`); !strings.Contains(rec.Body.String(), "Biscuit already has this") {
		t.Errorf("Duplicate purchase: %s", rec.Body.String())
	}

	server.pet.Update(10000) // Hibernates
	if rec := post(t, server.handleTrick(false), `{"trick":"sit"}`); !strings.Contains(rec.Body.String(), "Biscuit is hibernating") {
		t.Errorf("Training while hibernating: %s", rec.Body.String())
	}

	server.pet.Biology.IsAlive = false
	for name, handler := range map[string]http.HandlerFunc{
		"interact": server.handleInteract,
		"explore":  server.handleExplore,
	} {
		if rec := post(t, handler, `{"action":"feed"}`); !strings.Contains(rec.Body.String(), "Biscuit has passed away") {
			t.Errorf("%s with a pet that is not alive: %s", name, rec.Body.String())
		}
	}
}

func TestHibernationSetting(t *testing.T) {
	server := newTestServer(t)

	// Creating a pet with gentle mode
	if rec := post(t, server.handleCreate, `{"name":"Gentle","hibernation":true}`); rec.Code != http.StatusOK {
		t.Fatalf("Create: %s", rec.Body.String())
	}
	if !server.pet.HibernationEnabled || statusOf(t, server)["hibernation_enabled"] != true {
		t.Error("Pet created with hibernation should have it enabled")
	}

	// The default is off, and can be changed later
	post(t, server.handleCreate, `{"name":"Biscuit"}`)
	if server.pet.HibernationEnabled {
		t.Error("Hibernation should be off by default")
	}
	if rec := post(t, server.handleHibernation, `{"enabled":true}`); rec.Code != http.StatusOK {
		t.Fatalf("Toggle: %s", rec.Body.String())
	}
	if statusOf(t, server)["hibernation_enabled"] != true {
		t.Error("Status should show hibernation enabled")
	}

	saved, _, _, err := server.store.Load(server.slot)
	if err != nil || !saved.HibernationEnabled {
		t.Errorf("The setting should be saved: %v", err)
	}
}

func TestStatusShowsCauseOfDeath(t *testing.T) {
	server := newTestServer(t)
	server.awayTimeout = 0
	if err := server.createPet("Biscuit", false, false); err != nil {
		t.Fatal(err)
	}

	start := time.Now()
	for elapsed := time.Duration(0); elapsed <= 3*time.Hour && server.pet.IsAlive(); elapsed += time.Minute {
		server.tick(start.Add(elapsed))
	}

	s := statusOf(t, server)
	if s["is_alive"] != false || s["cause_of_death"] == nil || s["cause_of_death"] == "" {
		t.Errorf("A neglected pet should die with a cause: %+v", s)
	}
}

func TestTimeScaleSetsGameSpeed(t *testing.T) {
	ages := map[types.TimeScale]float64{}
	for _, scale := range []types.TimeScale{types.TimeScaleGame, types.TimeScaleAccelerated24X} {
		server := newTestServer(t)
		server.awayTimeout = 0
		server.clock = simulation.NewTimeManager(scale)
		if err := server.createPet("Biscuit", false, false); err != nil {
			t.Fatal(err)
		}
		age := server.pet.GetAge()
		start := time.Now()
		server.tick(start)
		server.tick(start.Add(time.Minute))
		ages[scale] = server.pet.GetAge() - age
	}

	if got := ages[types.TimeScaleGame]; got < 0.99 || got > 1.01 {
		t.Errorf("Game scale: one real minute should be one game day, got %f", got)
	}
	if got := ages[types.TimeScaleAccelerated24X]; got < 0.016 || got > 0.017 {
		t.Errorf("24x scale: one real minute should be 24 game minutes (%f days), got %f", 24.0/1440, got)
	}
}

func TestStatusShowsNeeds(t *testing.T) {
	server := newTestServer(t)
	if err := server.createPet("Biscuit", false, false); err != nil {
		t.Fatal(err)
	}
	needs, ok := statusOf(t, server)["needs"].([]interface{})
	if !ok || len(needs) != 5 {
		t.Errorf("Status should list five needs: %v", needs)
	}
}

func TestStatusShowsEnvironment(t *testing.T) {
	server := newTestServer(t)
	if err := server.createPet("Biscuit", false, false); err != nil {
		t.Fatal(err)
	}
	env, ok := statusOf(t, server)["environment"].(map[string]interface{})
	if !ok || env["weather"] == "" || env["season"] == "" || env["day_phase"] == "" {
		t.Errorf("Status should describe the environment: %v", env)
	}
}

func TestRefusedCareThroughServer(t *testing.T) {
	server := newTestServer(t)
	if err := server.createPet("Biscuit", false, false); err != nil {
		t.Fatal(err)
	}
	server.pet.Biology.Vitals.Nutrition = 0.95

	rec := post(t, server.handleInteract, `{"action":"feed"}`)
	if rec.Code != http.StatusConflict || !strings.Contains(rec.Body.String(), "Biscuit is not hungry.") {
		t.Errorf("Feeding a full pet: status %d, body %s", rec.Code, rec.Body.String())
	}

	tips, _ := statusOf(t, server)["care_tips"].(map[string]interface{})
	if tips["drink"] == "" || tips["feed"] == nil {
		t.Errorf("Status should describe each care button: %v", tips)
	}
}

// makeAdult makes the active pet old, healthy and happy enough to breed
func makeAdult(server *Server) {
	server.pet.Biology.Processes.Age = 10
	server.pet.Biology.Vitals.Health = 1.0
	server.pet.Biology.Vitals.Happiness = 0.9
}

func TestBreedingThroughServer(t *testing.T) {
	server := newTestServer(t)
	if err := server.createPet("Biscuit", false, false); err != nil {
		t.Fatal(err)
	}
	makeAdult(server)
	partnerSlot := server.slot
	if err := server.createPet("Pepper", false, false); err != nil {
		t.Fatal(err)
	}
	makeAdult(server)

	// Errors: a bad slot and breeding with itself
	if rec := post(t, server.handleBreed, `{"partner_slot":"../x"}`); rec.Code != http.StatusBadRequest {
		t.Errorf("A bad slot should be refused, got %d", rec.Code)
	}
	if rec := post(t, server.handleBreed, `{"partner_slot":"`+server.slot+`"}`); rec.Code != http.StatusBadRequest {
		t.Errorf("Breeding with itself should be refused, got %d", rec.Code)
	}

	// Breeding can fail by chance, so retry
	var body string
	for i := 0; i < 50; i++ {
		rec := post(t, server.handleBreed, `{"partner_slot":"`+partnerSlot+`","name":"Pip"}`)
		body = rec.Body.String()
		if rec.Code == http.StatusOK {
			break
		}
		if rec.Code != http.StatusConflict || !strings.Contains(body, "random chance") {
			t.Fatalf("Breeding failed for a reason other than chance: %d %s", rec.Code, body)
		}
	}
	if !strings.Contains(body, "Pip") {
		t.Fatalf("Breeding never succeeded: %s", body)
	}

	saved, err := server.store.List()
	if err != nil {
		t.Fatal(err)
	}
	if len(saved) != 3 {
		t.Errorf("The baby should be saved in its own slot: %v", saved)
	}
	if server.pet.Name != "Pepper" {
		t.Errorf("The active pet should stay active, got %s", server.pet.Name)
	}

	// Both parents are now in their cooldown
	rec := httptest.NewRecorder()
	server.handleBreedPartners(rec, httptest.NewRequest(http.MethodGet, "/api/breed/partners", nil))
	var resp struct {
		Data []breedingPartner `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	for _, partner := range resp.Data {
		if partner.Name == "Biscuit" && partner.Reason == "" {
			t.Errorf("Biscuit should be in its breeding cooldown: %+v", partner)
		}
	}
	if len(resp.Data) != 2 {
		t.Errorf("Partners should list the two other pets: %+v", resp.Data)
	}
}
