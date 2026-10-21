package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Michael-W-Ellison/gochi/internal/core"
	"github.com/Michael-W-Ellison/gochi/pkg/types"
)

func TestMiniGamesThroughServer(t *testing.T) {
	server := newTestServer(t)
	if err := server.createPet("Mochi", core.SpeciesCat, "", false, false); err != nil {
		t.Fatal(err)
	}
	rec := httptest.NewRecorder()
	server.handleMiniGames(rec, httptest.NewRequest(http.MethodGet, "/api/games", nil))
	var resp struct {
		Data []core.MiniGame `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil || len(resp.Data) != 4 {
		t.Errorf("A cat's games: %s", rec.Body.String())
	}
	if rec := post(t, server.handleFinishMiniGame, `{"game":"fetch","quality":1}`); rec.Code != http.StatusBadRequest {
		t.Errorf("A cat playing fetch: %d", rec.Code)
	}
	server.mu.Lock()
	server.pet.CareCounts = map[string]int{types.InteractionPlaying.String(): 1}
	server.mu.Unlock()
	rec = post(t, server.handleFinishMiniGame, `{"game":"wand","quality":0.6}`)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"enjoyment":"enjoyed"`) {
		t.Errorf("A game with the feather wand: %d %s", rec.Code, rec.Body.String())
	}
}
