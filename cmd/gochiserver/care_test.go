package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Michael-W-Ellison/gochi/internal/core"
)

func getJSON(t *testing.T, handler http.HandlerFunc, path string) map[string]interface{} {
	t.Helper()
	rec := httptest.NewRecorder()
	handler(rec, httptest.NewRequest(http.MethodGet, path, nil))
	var resp struct {
		Data interface{} `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("%s: %v: %s", path, err, rec.Body.String())
	}
	if m, ok := resp.Data.(map[string]interface{}); ok {
		return m
	}
	return map[string]interface{}{"list": resp.Data}
}

func TestSuppliesAndChoicesThroughServer(t *testing.T) {
	server := newTestServer(t)
	if err := server.createPet("Rex", core.SpeciesDog, false, false); err != nil {
		t.Fatal(err)
	}
	server.pet.FriendshipPoints, server.pet.PointsEarned = 200, 200

	supplies, _ := getJSON(t, server.handleSupplies, "/api/supplies")["supplies"].([]interface{})
	if len(supplies) == 0 {
		t.Fatal("The supply shop should list supplies")
	}
	for _, id := range []string{"tennis_ball", "dog_wet", "dog_treats"} {
		if rec := post(t, server.handleBuySupply, `{"id":"`+id+`"}`); rec.Code != http.StatusOK {
			t.Fatalf("Buying %s: %d %s", id, rec.Code, rec.Body.String())
		}
	}
	if rec := post(t, server.handleBuySupply, `{"id":"cat_wet"}`); rec.Code != http.StatusBadRequest {
		t.Errorf("A dog's shop shouldn't sell cat food: %d", rec.Code)
	}

	options := getJSON(t, server.handleCareOptions, "/api/care/options")
	if toys, _ := options["toys"].([]interface{}); len(toys) != 1 {
		t.Errorf("The ball should be among the toys: %v", options["toys"])
	}

	v := server.pet.Biology.Vitals
	v.Nutrition, v.Energy, v.Fatigue, v.Stress = 0.3, 1, 0, 0
	rec := post(t, server.handleInteract, `{"action":"feed","food":"dog_wet"}`)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "Wet food") {
		t.Errorf("Feeding wet food: %d %s", rec.Code, rec.Body.String())
	}
	rec = post(t, server.handleInteract, `{"action":"play","toy":"tennis_ball"}`)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "Fetch") {
		t.Errorf("Playing fetch: %d %s", rec.Code, rec.Body.String())
	}
	rec = post(t, server.handleInteract, `{"action":"pet","region":"chest"}`)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "chest") {
		t.Errorf("Petting the chest: %d %s", rec.Code, rec.Body.String())
	}
	if rec := post(t, server.handleInteract, `{"action":"play","toy":"wand_toy"}`); rec.Code != http.StatusBadRequest {
		t.Errorf("A dog can't play with a cat's wand: %d %s", rec.Code, rec.Body.String())
	}
	rec = post(t, server.handleTrick(false), `{"trick":"sit","method":"treats"}`)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "treat") {
		t.Errorf("Training with treats: %d %s", rec.Code, rec.Body.String())
	}
	if rec := post(t, server.handleTrick(false), `{"trick":"sit","method":"clicker"}`); rec.Code != http.StatusConflict ||
		!strings.Contains(rec.Body.String(), "needs a clicker") {
		t.Errorf("Clicker training without a clicker: %d %s", rec.Code, rec.Body.String())
	}
	rec = post(t, server.handleEnrich, `{"change":"sniff_game"}`)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "Nose work") {
		t.Errorf("A sniffing game: %d %s", rec.Code, rec.Body.String())
	}
}

func TestIntroductionThroughServer(t *testing.T) {
	server := newTestServer(t)
	if err := server.createPet("Biscuit", core.SpeciesCat, false, false); err != nil {
		t.Fatal(err)
	}
	partnerSlot := server.slot
	if err := server.createPet("Rex", core.SpeciesDog, false, false); err != nil {
		t.Fatal(err)
	}

	partners, _ := getJSON(t, server.handleIntroducePartners, "/api/introduce/partners")["list"].([]interface{})
	if len(partners) != 1 || !strings.Contains(partners[0].(map[string]interface{})["name"].(string), "Biscuit") {
		t.Fatalf("Biscuit should be listed to meet: %v", partners)
	}

	rec := post(t, server.handleIntroduce, `{"partner_slot":"`+partnerSlot+`"}`)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "leash") {
		t.Fatalf("Introducing a dog to a cat: %d %s", rec.Code, rec.Body.String())
	}
	if relations, _ := statusOf(t, server)["relations"].([]interface{}); len(relations) != 1 {
		t.Errorf("Rex should now know Biscuit: %v", relations)
	}

	// The partner was saved with the meeting
	partner, _, _, err := server.store.Load(partnerSlot)
	if err != nil {
		t.Fatal(err)
	}
	if relations := partner.Relations(); len(relations) != 1 || relations[0].Name != "Rex" {
		t.Errorf("Biscuit's save should record meeting Rex: %+v", relations)
	}
	if rec := post(t, server.handleIntroduce, `{"partner_slot":"`+server.slot+`"}`); rec.Code != http.StatusBadRequest {
		t.Errorf("A pet can't meet itself: %d", rec.Code)
	}
}

func TestGroupActivityThroughServer(t *testing.T) {
	server := newTestServer(t)
	var slots []string
	for _, name := range []string{"Max", "Bella"} {
		if err := server.createPet(name, core.SpeciesDog, false, false); err != nil {
			t.Fatal(err)
		}
		slots = append(slots, server.slot)
	}
	if err := server.createPet("Rex", core.SpeciesDog, false, false); err != nil {
		t.Fatal(err)
	}

	body := `{"activity":"walk","partner_slots":["` + slots[0] + `","` + slots[1] + `"]}`
	rec := post(t, server.handleGroup, body)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "Rex, Max and Bella went for a walk together") ||
		!strings.Contains(rec.Body.String(), `"order":[`) {
		t.Fatalf("A group walk: %d %s", rec.Code, rec.Body.String())
	}

	// Every pet was saved knowing the others
	for _, slot := range slots {
		partner, _, _, err := server.store.Load(slot)
		if err != nil {
			t.Fatal(err)
		}
		if relations := partner.Relations(); len(relations) != 2 || relations[0].LastShared != "went for a walk together" {
			t.Errorf("%s's save should record the walk: %+v", partner.Name, relations)
		}
	}

	for _, bad := range []string{
		`{"activity":"walk","partner_slots":[]}`,
		`{"activity":"walk","partner_slots":["` + slots[0] + `","` + slots[0] + `"]}`,
		`{"activity":"walk","partner_slots":["a","b","c","d"]}`,
		`{"activity":"walk","partner_slots":["` + server.slot + `"]}`,
		`{"activity":"dance","partner_slots":["` + slots[0] + `"]}`,
	} {
		if rec := post(t, server.handleGroup, bad); rec.Code != http.StatusBadRequest {
			t.Errorf("%s: expected 400, got %d %s", bad, rec.Code, rec.Body.String())
		}
	}
}

func TestSpeciesList(t *testing.T) {
	server := newTestServer(t)
	list, _ := getJSON(t, server.handleSpecies, "/api/species")["list"].([]interface{})
	if len(list) < 2 {
		t.Fatalf("Species: %v", list)
	}
	first := list[0].(map[string]interface{})
	if first["id"] != "cat" || first["icon"] == "" || first["walks"] != false {
		t.Errorf("The default species should be first: %v", first)
	}
	for _, item := range list {
		s := item.(map[string]interface{})
		if sprites, _ := s["sprites"].(map[string]interface{}); sprites["idle"] == "" || sprites["dead"] == "" {
			t.Errorf("%v has no sprites", s["id"])
		}
	}
	if rec := post(t, server.handleCreate, `{"name":"Nemo","species":"fish"}`); rec.Code != http.StatusBadRequest {
		t.Errorf("Creating an unknown species: %d %s", rec.Code, rec.Body)
	}
}
