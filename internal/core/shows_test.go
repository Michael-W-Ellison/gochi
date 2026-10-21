package core

import (
	"errors"
	"strings"
	"testing"
)

// showReady returns an adult pet ready for a show, knowing the tricks
func showReady(t *testing.T, species Species, tricksKnown ...string) *DigitalPet {
	t.Helper()
	pet := NewDigitalPet("Mochi", "owner")
	pet.Species = species
	pet.Biology.Processes.Age = 200
	pet.FriendshipPoints = 1000
	pet.TricksLearned, pet.TrickSkills = map[string]bool{}, map[string]float64{}
	for _, id := range tricksKnown {
		pet.TricksLearned[id], pet.TrickSkills[id] = true, 1
	}
	setCalmHealthyState(pet)
	return pet
}

func TestSpeciesTricks(t *testing.T) {
	cat, dog := NewDigitalPet("Cat", "o"), NewDigitalPet("Dog", "o")
	dog.Species = SpeciesDog
	if n := len(cat.Tricks()); n != 13 {
		t.Errorf("Cats have %d tricks, want 11 plus 2 special", n)
	}
	if n := len(dog.Tricks()); n != 14 {
		t.Errorf("Dogs have %d tricks", n)
	}
	// Prerequisites
	if _, err := dog.TrainTrick("stay"); !strings.Contains(errString(err), "Down first") {
		t.Errorf("Stay before down: %v", err)
	}
	if _, err := cat.TrainTrick("down"); err == nil {
		t.Error("Cats aren't taught down")
	}
}

func errString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

func TestCatsRollOverOnlyYoungAndSuited(t *testing.T) {
	kitten := NewDigitalPet("Kit", "o")
	kitten.TricksLearned = map[string]bool{"sit": true}
	kitten.TrickSkills = map[string]float64{"sit": 1}
	kitten.Personality.Traits.Playfulness, kitten.Personality.Traits.Neuroticism = 0.8, 0.2
	setCalmHealthyState(kitten)
	if _, err := kitten.TrainTrick("roll_over"); err != nil {
		t.Errorf("A playful, relaxed kitten can learn to roll over: %v", err)
	}

	old := NewDigitalPet("Old", "o")
	old.TricksLearned, old.TrickSkills = map[string]bool{"sit": true}, map[string]float64{"sit": 1}
	old.Personality.Traits.Playfulness, old.Personality.Traits.Neuroticism = 0.8, 0.2
	old.Biology.Processes.Age = catSpecialAge + 1
	setCalmHealthyState(old)
	if _, err := old.TrainTrick("roll_over"); !strings.Contains(errString(err), "young cats") {
		t.Errorf("An older cat: %v", err)
	}

	shy := NewDigitalPet("Shy", "o")
	shy.TricksLearned, shy.TrickSkills = map[string]bool{"sit": true}, map[string]float64{"sit": 1}
	shy.Personality.Traits.Playfulness, shy.Personality.Traits.Neuroticism = 0.3, 0.7
	setCalmHealthyState(shy)
	if _, err := shy.TrainTrick("roll_over"); !strings.Contains(errString(err), "playful, relaxed") {
		t.Errorf("A shy cat: %v", err)
	}

	// Rolling over doesn't count in a cat's show routine
	cat := showReady(t, SpeciesCat, "sit", "touch", "come", "roll_over")
	if _, err := cat.EnterShow(ShowTricks, LevelNovice, []string{"sit", "touch", "roll_over"}); !errors.Is(err, ErrShowRoutine) {
		t.Errorf("A cat's roll over in a show: %v", err)
	}
}

func TestShowEntry(t *testing.T) {
	young := showReady(t, SpeciesDog, "sit", "come", "touch")
	young.Biology.Processes.Age = 10
	if _, err := young.EnterShow(ShowTricks, LevelNovice, []string{"sit", "come", "touch"}); err == nil ||
		!strings.Contains(err.Error(), "too young") {
		t.Errorf("A young pet: %v", err)
	}
	dog := showReady(t, SpeciesDog, "sit", "come", "touch")
	if _, err := dog.EnterShow(ShowTricks, LevelOpen, []string{"sit", "come", "touch"}); !errors.Is(err, ErrShowLevel) {
		t.Errorf("Open level before winning Novice: %v", err)
	}
	if _, err := dog.EnterShow(ShowTricks, LevelNovice, []string{"sit", "sit", "touch"}); !errors.Is(err, ErrShowRoutine) {
		t.Errorf("A repeated trick: %v", err)
	}
	if _, err := dog.EnterShow(ShowPlaytime, LevelNovice, nil); !errors.Is(err, ErrShowSpecies) {
		t.Errorf("A dog at the cats' show: %v", err)
	}

	useRandom(t, 0) // Entrants score their lowest, the pet wins
	before := dog.FriendshipPoints
	result, err := dog.EnterShow(ShowTricks, LevelNovice, []string{"sit", "come", "touch"})
	if err != nil {
		t.Fatal(err)
	}
	if result.Place != 1 || result.Points != 60 || result.Stars != 5 || result.Ribbon == nil || len(result.Entrants) != 6 {
		t.Errorf("A win: %+v", result)
	}
	if dog.FriendshipPoints != before-20+60 {
		t.Errorf("Points after entry and prize: %d", dog.FriendshipPoints-before)
	}
	if levels := dog.Shows()[0].Levels; len(levels) != 2 {
		t.Errorf("Winning Novice opens Open: %v", levels)
	}
	if _, err := dog.EnterShow(ShowTricks, LevelNovice, []string{"sit", "come", "touch"}); err == nil ||
		!strings.Contains(err.Error(), "rest") {
		t.Errorf("Straight into another show: %v", err)
	}
}

func TestShowScoring(t *testing.T) {
	dog := showReady(t, SpeciesDog, "sit", "come", "touch", "paw", "down")
	useRandom(t, 100) // No variation
	dog.mu.Lock()
	defer dog.mu.Unlock()
	easyLast, _ := dog.showScoreLocked(showDefs[0], []string{"touch", "paw", "sit"})
	hardLast, _ := dog.showScoreLocked(showDefs[0], []string{"sit", "come", "touch"})
	if hardLast <= easyLast {
		t.Errorf("Ending on the hardest trick earns a bonus: %d vs %d", hardLast, easyLast)
	}
	dog.Biology.Vitals.Energy, dog.Biology.Vitals.Stress = 0.3, 0.9
	tired, _ := dog.showScoreLocked(showDefs[0], []string{"sit", "come", "touch"})
	if tired >= hardLast {
		t.Error("A tired, stressed pet performs worse")
	}
	dog.Biology.Vitals.Cleanliness = 0.2
	dirty, _ := dog.showScoreLocked(showDefs[1], nil)
	dog.Biology.Vitals.Cleanliness = 1
	clean, _ := dog.showScoreLocked(showDefs[1], nil)
	if clean <= dirty {
		t.Error("A clean coat scores better at the grooming show")
	}
}

func TestShowPointsLimitAndGrandChampion(t *testing.T) {
	dog := showReady(t, SpeciesDog, "sit", "come", "touch")
	dog.mu.Lock()
	if a, b := dog.showPointsLocked(200), dog.showPointsLocked(200); a != 200 || b != 100 {
		t.Errorf("Show points a day: %d then %d", a, b)
	}
	dog.mu.Unlock()

	useRandom(t, 100) // No variation; entrants score near the bottom of their range
	dog = showReady(t, SpeciesDog, "sit", "come", "touch", "weave", "hoop")
	for _, show := range []string{ShowTricks, ShowAgility, ShowHappiest} {
		for level := LevelNovice; level <= LevelChampion; level++ {
			dog.LastShowAge = 0
			delete(dog.Marks, "show_free_day")
			setCalmHealthyState(dog)
			if _, err := dog.EnterShow(show, level, []string{"touch", "hoop", "weave"}); err != nil {
				t.Fatalf("%s level %d: %v", show, level, err)
			}
		}
	}
	if !dog.hasTitleLocked(TitleGrandChampion) {
		t.Errorf("Three Champion wins make a Grand Champion: %+v", dog.Ribbons)
	}
	earned := map[string]bool{}
	for _, a := range dog.CheckAchievements() {
		earned[a.ID] = true
	}
	for _, id := range []string{"first_ribbon", "blue_rosette", "champion", "grand_champion"} {
		if !earned[id] {
			t.Errorf("%s should be earned", id)
		}
	}
}
