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

// The puzzle feeder gets harder as the pet solves it well, opening more
// kinds of feeder, and a little easier when it gives up; the set-up
// follows the level
func TestPuzzleFeederLevels(t *testing.T) {
	useClock(t, time.Date(2026, 3, 10, 9, 0, 0, 0, time.UTC))
	pet := NewDigitalPet("Mochi", "owner")
	pet.Personality.Traits.Intelligence = 0.5
	setCalmHealthyState(pet)
	pet.CareCounts = map[string]int{types.InteractionEnvironmentalEnrichment.String(): 1}
	setup := func() PuzzleSetup {
		for _, g := range pet.MiniGames() {
			if g.ID == GamePuzzle {
				if g.Puzzle == nil || g.Tiles != g.Puzzle.Tiles {
					t.Fatalf("The puzzle feeder should carry its set-up: %+v", g)
				}
				return *g.Puzzle
			}
		}
		t.Fatal("No puzzle feeder")
		return PuzzleSetup{}
	}

	first := setup()
	if first.Level != 1 || len(first.Kinds) != 1 || first.Kinds[0] != PuzzleTiles || !first.Exact ||
		first.Tiles != 5 || first.Kibble != 3 || first.Misses != 3 || first.Sniffs != 3 {
		t.Errorf("Level 1: %+v", first)
	}
	previous := first
	for level := 2; level <= maxPuzzleLevel+1; level++ {
		pet.LastGameReward = time.Time{}
		result, err := pet.FinishMiniGame(GamePuzzle, 0.9)
		if err != nil || !result.Applied {
			t.Fatalf("Solving level %d: %+v %v", level-1, result, err)
		}
		s := setup()
		if level > maxPuzzleLevel {
			if s.Level != maxPuzzleLevel || result.Note != "" {
				t.Errorf("The level should stop at %d: %+v %q", maxPuzzleLevel, s, result.Note)
			}
			break
		}
		if s.Level != level || result.Note == "" {
			t.Errorf("Solving it well should raise the level to %d: %+v %q", level, s, result.Note)
		}
		if s.Tiles < previous.Tiles || s.Misses > previous.Misses || s.Sniffs > previous.Sniffs ||
			s.ShowSeconds > previous.ShowSeconds || s.Swaps <= previous.Swaps || s.SwapSeconds > previous.SwapSeconds ||
			len(s.Kinds) < len(previous.Kinds) {
			t.Errorf("Level %d should be no easier than %d: %+v then %+v", level, level-1, previous, s)
		}
		if s.Kibble < 2 || s.Kibble >= s.Tiles || s.Misses < 1 || s.Sniffs < 1 || s.Cups > 5 || s.Tiles > 12 {
			t.Errorf("Level %d is out of bounds: %+v", level, s)
		}
		previous = s
	}
	if hardest := setup(); len(hardest.Kinds) != 4 || hardest.Exact || hardest.Tiles != 12 {
		t.Errorf("The hardest level should have every feeder and only vague hints: %+v", hardest)
	}

	// Giving up makes it easier; a middling game keeps it
	pet.LastGameReward = time.Time{}
	if result, _ := pet.FinishMiniGame(GamePuzzle, 0.2); setup().Level != maxPuzzleLevel-1 || result.Note == "" {
		t.Errorf("Giving up should lower the level: %d %q", setup().Level, result.Note)
	}
	pet.LastGameReward = time.Time{}
	if result, _ := pet.FinishMiniGame(GamePuzzle, 0.5); setup().Level != maxPuzzleLevel-1 || result.Note != "" {
		t.Errorf("A middling game should keep the level: %d %q", setup().Level, result.Note)
	}

	// The level is kept with the pet, and a damaged one is brought into range
	data, err := pet.Save()
	if err != nil {
		t.Fatal(err)
	}
	loaded, err := Load(data)
	if err != nil {
		t.Fatal(err)
	}
	pet = loaded
	setCalmHealthyState(pet)
	if setup().Level != maxPuzzleLevel-1 {
		t.Errorf("The level should be saved: %d", setup().Level)
	}
	pet.PuzzleLevel = 99
	if setup().Level != maxPuzzleLevel {
		t.Errorf("A level out of range: %d", setup().Level)
	}
}
