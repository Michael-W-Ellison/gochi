package core

import (
	"errors"
	"math"
	"testing"
	"time"

	"github.com/Michael-W-Ellison/gochi/pkg/types"
)

func TestMiniGames(t *testing.T) {
	useClock(t, time.Date(2026, 3, 10, 9, 0, 0, 0, time.UTC))
	cat, dog := NewDigitalPet("Mochi", "owner"), NewDigitalPet("Rex", "owner")
	dog.Species = SpeciesDog
	ids := func(p *DigitalPet) map[string]MiniGame {
		m := map[string]MiniGame{}
		for _, g := range p.MiniGames() {
			m[g.ID] = g
		}
		return m
	}
	catGames, dogGames := ids(cat), ids(dog)
	if len(catGames) != 4 || len(dogGames) != 4 || catGames[GameFetch].ID != "" || dogGames[GameWand].ID != "" {
		t.Errorf("Games: cat %v, dog %v", catGames, dogGames)
	}
	if g := catGames[GamePuzzle]; g.Tiles < 3 || g.Tiles > 6 || g.Unavailable == "" {
		t.Errorf("A new pet's puzzle feeder: %+v", g)
	}
	if g := catGames[GameGrooming]; len(g.Regions) != len(SpeciesCat.Def().PetRegions) {
		t.Errorf("Grooming regions: %+v", g.Regions)
	}

	// Locked until the pet has had that kind of care
	setCalmHealthyState(dog)
	var refused RefusedError
	if _, err := dog.FinishMiniGame(GameFetch, 1); !errors.As(err, &refused) {
		t.Errorf("Fetch before any play: %v", err)
	}
	if _, err := dog.FinishMiniGame(GameWand, 1); !errors.Is(err, ErrUnknownGame) {
		t.Errorf("A dog with the cats' feather wand: %v", err)
	}
	dog.CareCounts = map[string]int{types.InteractionPlaying.String(): 1}
	result, err := dog.FinishMiniGame(GameFetch, 1)
	if err != nil || !result.Applied || result.Enjoyment != "loved" || result.Lesson == "" {
		t.Errorf("A great game of fetch: %+v %v", result, err)
	}
	setCalmHealthyState(dog)
	if result, _ := dog.FinishMiniGame(GameFetch, math.NaN()); result.Enjoyment != "bored" {
		t.Errorf("A game that went badly: %+v", result)
	}
}
