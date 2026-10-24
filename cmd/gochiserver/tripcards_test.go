package main

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/Michael-W-Ellison/gochi/internal/core"
)

// While the pet is out exploring, the page can fetch facts and tips to show
func TestTripCardsThroughThePage(t *testing.T) {
	h := newHarness(t)
	if rec := h.do(http.MethodGet, "/api/explore/cards", ""); rec.Code != http.StatusNotFound {
		t.Errorf("Cards without a pet: %d", rec.Code)
	}
	h.createPet("Rex", core.SpeciesDog)
	if rec := h.post("/api/explore", `{"place":"park"}`); rec.Code != http.StatusOK {
		t.Fatalf("Exploring: %d %s", rec.Code, rec.Body.String())
	}
	rec := h.do(http.MethodGet, "/api/explore/cards", "")
	var reply struct {
		Data struct {
			Cards []core.TripCard `json:"cards"`
		} `json:"data"`
	}
	if rec.Code != http.StatusOK || json.Unmarshal(rec.Body.Bytes(), &reply) != nil || len(reply.Data.Cards) < 10 {
		t.Fatalf("Cards: %d %s", rec.Code, rec.Body.String())
	}
	if c := reply.Data.Cards[0]; c.Kind != "fact" || c.Title != "At the park" {
		t.Errorf("The first card should be where Rex is: %+v", c)
	}
}
