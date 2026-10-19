package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
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
	if err := server.createPet("Biscuit", false); err != nil {
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
	if err := server.createPet("Biscuit", false); err != nil {
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
	if err := server.createPet("Biscuit", false); err != nil {
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

	s := statusOf(t, server)
	adventure, _ := s["last_adventure"].(map[string]interface{})
	if s["exploring"] != false || adventure == nil || !strings.Contains(adventure["text"].(string), "came home from exploring") {
		t.Errorf("Pet should be home with an adventure: exploring=%v adventure=%+v", s["exploring"], adventure)
	}
	if souvenirs := s["souvenirs"].([]interface{}); len(souvenirs) != 1 {
		t.Errorf("Status should list one souvenir, got %v", souvenirs)
	}
}
