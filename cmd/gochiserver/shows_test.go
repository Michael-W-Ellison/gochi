package main

import (
	"net/http"
	"testing"

	"github.com/Michael-W-Ellison/gochi/internal/core"
)

func TestShowsThroughServer(t *testing.T) {
	server := newTestServer(t)
	server.house = newTestHousehold(t, "")
	if err := server.createPet("Rex", core.SpeciesDog, "", false, false); err != nil {
		t.Fatal(err)
	}
	server.mu.Lock()
	server.pet.Biology.Processes.Age = 200
	server.pet.Biology.Vitals.Energy = 1
	server.mu.Unlock()

	data := getJSON(t, server.handleShows, "/api/shows")
	shows, _ := data["shows"].([]interface{})
	if len(shows) != 4 { // Tricks, grooming, agility and happiest; not the cats' playtime
		t.Errorf("A dog's shows: %d", len(shows))
	}
	if rec := post(t, server.handleEnterShow, `{"show":"playtime","level":1}`); rec.Code != http.StatusBadRequest {
		t.Errorf("The cats' show: %d", rec.Code)
	}
	if rec := post(t, server.handleEnterShow, `{"show":"tricks","level":1,"routine":["sit","come","touch"]}`); rec.Code != http.StatusConflict {
		t.Errorf("A routine of tricks it doesn't know: %d %s", rec.Code, rec.Body.String())
	}
	before := balance(server.house)
	rec := post(t, server.handleEnterShow, `{"show":"happiest","level":1}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("Happiest pet: %d %s", rec.Code, rec.Body.String())
	}
	ribbons, _ := server.pet.ShowRecord()
	if len(ribbons) > 0 && ribbons[0].Place == 1 && balance(server.house) != before+5 {
		t.Errorf("A Novice win earns 5 Home Stars: %d", balance(server.house)-before)
	}
}
