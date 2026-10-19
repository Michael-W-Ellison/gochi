package core

import (
	"errors"
	"testing"
	"time"

	"github.com/Michael-W-Ellison/gochi/pkg/types"
)

// useRandom makes randIntn return fixed values for one test
func useRandom(t *testing.T, values ...int) {
	t.Helper()
	i := 0
	randIntn = func(n int) int {
		v := values[i%len(values)] % n
		i++
		return v
	}
	t.Cleanup(func() { randIntn = defaultRandIntn })
}

var defaultRandIntn = randIntn

func TestTrainingLearnsTrick(t *testing.T) {
	pet := NewDigitalPet("Biscuit", "user123")

	sessions := 0
	var result TrickResult
	for !result.JustLearned {
		setCalmHealthyState(pet)
		var err error
		result, err = pet.TrainTrick("sit")
		if err != nil {
			t.Fatalf("TrainTrick failed: %v", err)
		}
		sessions++
		if sessions > 20 {
			t.Fatal("Sit should be learned within 20 sessions")
		}
	}

	if result.Text != "Biscuit learned Sit! 🐾" {
		t.Errorf("Learned text = %q", result.Text)
	}
	if tricks := pet.Tricks(); !tricks[0].Learned || tricks[0].Skill != 1.0 {
		t.Errorf("Sit should be learned: %+v", tricks[0])
	}

	performed, err := pet.PerformTrick("sit")
	if err != nil || performed.Text != "Biscuit sits down neatly." {
		t.Errorf("PerformTrick = %q, %v", performed.Text, err)
	}
}

func TestHarderTricksTakeLonger(t *testing.T) {
	sessionsToLearn := func(trick string) int {
		pet := NewDigitalPet("Biscuit", "user123")
		for n := 1; n <= 100; n++ {
			setCalmHealthyState(pet)
			if r, _ := pet.TrainTrick(trick); r.JustLearned {
				return n
			}
		}
		return 100
	}

	if easy, hard := sessionsToLearn("sit"), sessionsToLearn("play_dead"); hard <= easy {
		t.Errorf("Play dead (%d sessions) should take longer than Sit (%d)", hard, easy)
	}
}

func TestSmarterPetsLearnFaster(t *testing.T) {
	smart := NewDigitalPet("Smart", "user123")
	smart.Personality.Traits.Intelligence = 1.0
	slow := NewDigitalPet("Slow", "user123")
	slow.Personality.Traits.Intelligence = 0.0

	a, _ := smart.TrainTrick("spin")
	b, _ := slow.TrainTrick("spin")
	if a.Skill <= b.Skill {
		t.Errorf("An intelligent pet should learn faster: %f vs %f", a.Skill, b.Skill)
	}
}

func TestTrickErrors(t *testing.T) {
	pet := NewDigitalPet("Biscuit", "user123")

	if _, err := pet.TrainTrick("fly"); !errors.Is(err, ErrUnknownTrick) {
		t.Errorf("Unknown trick: %v", err)
	}
	if _, err := pet.PerformTrick("spin"); !errors.Is(err, ErrTrickNotKnown) {
		t.Errorf("Performing an unlearned trick: %v", err)
	}

	pet.Biology.Vitals.Energy = 0.2
	if _, err := pet.TrainTrick("sit"); !errors.Is(err, ErrTooTired) {
		t.Errorf("Training a tired pet: %v", err)
	}
}

func TestTrainingEarnsPointsAndLearning(t *testing.T) {
	useClock(t, time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC))
	pet := NewDigitalPet("Biscuit", "user123")

	r, _ := pet.TrainTrick("sit")
	if r.PointsEarned != PointsPerCare || r.ReactionText == "" {
		t.Errorf("Training should earn points and react: %+v", r.InteractionResult)
	}
	if pet.Memory.GetLearnedAffinity(types.InteractionTraining) <= 0 {
		t.Error("Training should be learned as a positive experience")
	}
}

func TestHideAndSeekFound(t *testing.T) {
	useClock(t, time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC))
	useRandom(t, 2) // Hides in the toy box
	pet := NewDigitalPet("Biscuit", "user123")

	if _, err := pet.GuessHidingSpot(0); !errors.Is(err, ErrNoGame) {
		t.Errorf("Guessing without a game: %v", err)
	}
	if err := pet.StartHideAndSeek(); err != nil {
		t.Fatal(err)
	}

	miss, _ := pet.GuessHidingSpot(0)
	if miss.Found || miss.Over || miss.GuessesLeft != 1 {
		t.Errorf("First wrong guess should leave one guess: %+v", miss)
	}

	hit, err := pet.GuessHidingSpot(2)
	if err != nil || !hit.Found || !hit.Over || hit.Spot != "in the toy box" {
		t.Fatalf("Second guess should find the pet: %+v, %v", hit, err)
	}
	if hit.BonusPoints != GameBonusPoints || hit.PointsEarned != PointsPerCare || !hit.Applied {
		t.Errorf("Finding the pet should count as play and earn the bonus: %+v", hit)
	}
	if pet.PlayingHideAndSeek() {
		t.Error("The game should be over")
	}

	// The bonus has a cooldown
	pet.StartHideAndSeek()
	again, _ := pet.GuessHidingSpot(2)
	if again.BonusPoints != 0 {
		t.Error("The game bonus should not be earned again within the cooldown")
	}
}

func TestHideAndSeekOutOfGuesses(t *testing.T) {
	useRandom(t, 3)
	pet := NewDigitalPet("Biscuit", "user123")
	pet.StartHideAndSeek()

	pet.GuessHidingSpot(0)
	last, _ := pet.GuessHidingSpot(1)
	if last.Found || !last.Over || last.Spot != "behind the curtain" || last.BonusPoints != 0 {
		t.Errorf("After two misses the pet should reveal itself without a bonus: %+v", last)
	}
	if _, err := pet.GuessHidingSpot(5); !errors.Is(err, ErrNoGame) {
		t.Errorf("Game should be over: %v", err)
	}
}

func TestExploringTrip(t *testing.T) {
	useRandom(t, 0) // First souvenir
	pet := NewDigitalPet("Biscuit", "user123")
	setCalmHealthyState(pet)

	if err := pet.SendExploring(); err != nil {
		t.Fatal(err)
	}
	if !pet.IsExploring() || pet.CurrentBehavior != types.BehaviorExploring {
		t.Fatal("Pet should be out exploring")
	}
	if r := pet.Interact(types.InteractionFeeding, 1.0); r.Applied {
		t.Error("A pet out exploring cannot be fed")
	}
	if err := pet.SendExploring(); !errors.Is(err, ErrExploring) {
		t.Errorf("Sending again: %v", err)
	}
	if pet.CurrentRequest() != nil {
		t.Error("A pet out exploring should not ask for anything")
	}

	// Halfway: still out
	pet.Update(exploreDuration / 2)
	if !pet.IsExploring() {
		t.Fatal("Pet should still be exploring halfway through")
	}

	pet.Update(exploreDuration)
	if pet.IsExploring() {
		t.Fatal("Pet should be home after the trip")
	}
	if pet.LastAdventure == nil || pet.LastAdventure.Souvenir.ID != "pebble" ||
		pet.LastAdventure.Text != "Biscuit came home from exploring with a shiny pebble! 🪨" {
		t.Errorf("Unexpected adventure: %+v", pet.LastAdventure)
	}
	if c := pet.SouvenirCollection(); len(c) != 1 || c[0].ID != "pebble" || c[0].Count != 1 {
		t.Errorf("Collection = %+v", c)
	}
	if pet.Memory.GetLearnedAffinity(types.InteractionEnvironmentalEnrichment) <= 0 {
		t.Error("Exploring should be learned as a positive experience")
	}
}

func TestExploringNeedsCare(t *testing.T) {
	pet := NewDigitalPet("Biscuit", "user123")
	setCalmHealthyState(pet)
	pet.Biology.Vitals.Hydration = 0.3

	if err := pet.SendExploring(); !errors.Is(err, ErrNeedsCare) {
		t.Errorf("A thirsty pet should not go exploring: %v", err)
	}
}

func TestExploringPausesWithTime(t *testing.T) {
	pet := NewDigitalPet("Biscuit", "user123")
	setCalmHealthyState(pet)
	pet.SendExploring()
	remaining := pet.ExploringRemaining()

	// No updates (time paused): the trip does not progress
	if pet.ExploringRemaining() != remaining || remaining != exploreDuration {
		t.Errorf("Remaining = %f, want %f", pet.ExploringRemaining(), exploreDuration)
	}
}

func TestLovingExploringMakesPetAdventurous(t *testing.T) {
	pet := NewDigitalPet("Biscuit", "user123")
	for i := 0; i < 12; i++ {
		setCalmHealthyState(pet)
		pet.SendExploring()
		pet.Update(exploreDuration + 0.1)
	}
	setCalmHealthyState(pet)
	pet.updateBehavior()

	if pet.CurrentBehavior != types.BehaviorExploring {
		t.Errorf("A pet that loves exploring should choose to explore, got %s", pet.CurrentBehavior)
	}
}

func TestActivitiesSurviveSaveLoad(t *testing.T) {
	pet := NewDigitalPet("Biscuit", "user123")
	setCalmHealthyState(pet)
	pet.TrainTrick("sit")
	pet.SendExploring()

	data, err := pet.Save()
	if err != nil {
		t.Fatal(err)
	}
	loaded, err := Load(data)
	if err != nil {
		t.Fatal(err)
	}

	if loaded.Tricks()[0].Skill != pet.Tricks()[0].Skill || !loaded.IsExploring() {
		t.Error("Trick skills and an exploring trip should be restored")
	}
}
