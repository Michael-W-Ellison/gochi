package core

import (
	"errors"
	"math"
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
		pet.Species = SpeciesDog
		pet.TricksLearned = map[string]bool{"roll_over": true} // Play dead builds on it
		for n := 1; n <= 100; n++ {
			setCalmHealthyState(pet)
			if r, _ := pet.TrainTrick(trick); r.JustLearned {
				return n
			}
		}
		return 100
	}

	if easy, hard := sessionsToLearn("sit"), sessionsToLearn("play_dead"); hard <= easy || hard == 100 {
		t.Errorf("Play dead (%d sessions) should take longer than Sit (%d)", hard, easy)
	}
}

func TestSmarterPetsLearnFaster(t *testing.T) {
	smart := NewDigitalPet("Smart", "user123")
	smart.Personality.Traits.Intelligence = 1.0
	slow := NewDigitalPet("Slow", "user123")
	slow.Personality.Traits.Intelligence = 0.0

	a, _ := smart.TrainTrick("touch")
	b, _ := slow.TrainTrick("touch")
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
	useRandom(t, 50, 0) // Treasure outcome, first souvenir
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
	if pet.LastAdventure == nil || pet.LastAdventure.Outcome != OutcomeTreasure ||
		pet.LastAdventure.Souvenir == nil || pet.LastAdventure.Souvenir.ID != "pebble" ||
		pet.LastAdventure.Text != "Biscuit came home from a harness walk at the park with a shiny pebble! 🪨 Biscuit made some friends along the way." {
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
	useRandom(t, 50, 0) // Every trip goes well
	pet := NewDigitalPet("Biscuit", "user123")
	for i := 0; i < 12; i++ {
		setCalmHealthyState(pet)
		pet.RecentCare = nil // Trips spread out over time
		pet.SendExploring()
		pet.Update(exploreDuration + 0.1)
	}
	setCalmHealthyState(pet)
	pet.updateBehavior()

	if pet.CurrentBehavior != types.BehaviorInvestigating {
		t.Errorf("A pet that loves exploring should be curious at home, got %s", pet.CurrentBehavior)
	}
	if pet.IsExploring() || pet.Activity() != "is sniffing around the house" {
		t.Errorf("A curious pet at home is not on a trip: exploring %v, %q", pet.IsExploring(), pet.Activity())
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

// exploreOnce sends a healthy pet on a full trip
func exploreOnce(t *testing.T, pet *DigitalPet) *Adventure {
	t.Helper()
	setCalmHealthyState(pet)
	if err := pet.SendExploring(); err != nil {
		t.Fatalf("SendExploring: %v", err)
	}
	pet.Update(exploreDuration + 0.1)
	if pet.IsExploring() {
		t.Fatal("Pet should be home")
	}
	return pet.LastAdventure
}

func TestTripCanFindNothing(t *testing.T) {
	useClock(t, time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC))
	useRandom(t, 99, 0) // Nothing of value: a muddy stick
	pet := NewDigitalPet("Biscuit", "user123")

	a := exploreOnce(t, pet)
	if a.Outcome != OutcomeNothing || a.Souvenir != nil || a.Text != "Biscuit came home from a harness walk at the park with a muddy stick. Not much of a treasure! Biscuit made some friends along the way." {
		t.Errorf("Unexpected adventure: %+v", a)
	}
	if len(pet.SouvenirCollection()) != 0 {
		t.Error("Junk should not be added to the treasures")
	}
	if a.PointsEarned != PointsPerCare {
		t.Error("A trip that finds nothing is still fun and earns points")
	}

	useRandom(t, 99, 3) // Nothing at all; the second trip discovers the beach
	if a := exploreOnce(t, pet); a.Text != "Biscuit came home from a harness walk at the park empty-handed. Biscuit made some friends along the way. 🗺️ Biscuit can now explore the beach!" {
		t.Errorf("Text = %q", a.Text)
	}
}

func TestTripCanInjurePet(t *testing.T) {
	useRandom(t, 0) // Injury
	pet := NewDigitalPet("Biscuit", "user123")
	setCalmHealthyState(pet)
	points := pet.FriendshipPoints

	a := exploreOnce(t, pet)
	if a.Outcome != OutcomeInjured || a.Souvenir != nil || a.PointsEarned != 0 {
		t.Fatalf("Unexpected adventure: %+v", a)
	}
	if got, _ := changeOf(a.Changes, "Health"); math.Abs(got+0.25) > 1e-9 {
		t.Errorf("Injury should cost 0.25 health, got %f", got)
	}
	if !pet.Injured || pet.FriendshipPoints != points {
		t.Errorf("Pet should be injured and earn nothing: injured=%v points=%d", pet.Injured, pet.FriendshipPoints)
	}
	if pet.Memory.GetLearnedAffinity(types.InteractionEnvironmentalEnrichment) >= 0 {
		t.Error("Getting hurt should make the pet wary of exploring")
	}

	if r := pet.CurrentRequest(); r == nil || r.Action != "medic" {
		t.Errorf("An injured pet should ask to be healed, got %+v", r)
	}
	setCalmHealthyState(pet)
	if err := pet.SendExploring(); !errors.Is(err, ErrInjured) {
		t.Errorf("An injured pet should not go exploring: %v", err)
	}

	pet.ProcessUserInteraction(types.InteractionMedicalCare, 1.0)
	if pet.Injured {
		t.Error("Medical care should heal the injury")
	}
	if r := pet.CurrentRequest(); r != nil && r.Action == "medic" {
		t.Error("A healed pet should stop asking to be healed")
	}
}

func TestSevereInjuryHibernatesInsteadOfDying(t *testing.T) {
	useRandom(t, 0) // Injury
	pet := NewDigitalPet("Biscuit", "user123")
	pet.HibernationEnabled = true
	setCalmHealthyState(pet)
	pet.SendExploring()
	pet.Biology.Vitals.Health = 0.3

	pet.Update(exploreDuration + 0.1)

	if !pet.IsAlive() || !pet.IsHibernating() {
		t.Errorf("A badly hurt pet should hibernate, alive=%v hibernating=%v", pet.IsAlive(), pet.IsHibernating())
	}
}

func TestPersonalityChangesTripOdds(t *testing.T) {
	outcome := func(curiosity, adaptability float64, roll int) string {
		useRandom(t, roll)
		pet := NewDigitalPet("Biscuit", "user123")
		setCalmHealthyState(pet) // Fine weather
		pet.Personality.Traits.Curiosity = curiosity
		pet.Personality.Traits.Adaptability = adaptability
		return pet.exploreOutcomeLocked()
	}

	// A roll of 60 finds treasure for a curious pet but not an incurious one
	if got := outcome(1.0, 0.5, 60); got != OutcomeTreasure {
		t.Errorf("Curious pet: %s", got)
	}
	if got := outcome(0.0, 0.5, 60); got != OutcomeNothing {
		t.Errorf("Incurious pet: %s", got)
	}

	// A roll of 10 injures a clumsy pet but not an adaptable one
	if got := outcome(0.5, 0.0, 10); got != OutcomeInjured {
		t.Errorf("Unadaptable pet: %s", got)
	}
	if got := outcome(0.5, 1.0, 10); got == OutcomeInjured {
		t.Error("Adaptable pet should not be injured by this roll")
	}
}

func TestRecallFromExploringLosesGains(t *testing.T) {
	pet := NewDigitalPet("Biscuit", "user123")
	if _, err := pet.RecallFromExploring(); !errors.Is(err, ErrNotExploring) {
		t.Errorf("Recalling a pet at home: %v", err)
	}

	setCalmHealthyState(pet)
	pet.SendExploring()
	pet.Update(exploreDuration / 2)
	points, interactions, affinity := pet.FriendshipPoints, pet.TotalInteractions,
		pet.Memory.GetLearnedAffinity(types.InteractionEnvironmentalEnrichment)

	text, err := pet.RecallFromExploring()
	if err != nil || text != "Biscuit heads home early and brings nothing back." {
		t.Fatalf("Recall = %q, %v", text, err)
	}
	if pet.IsExploring() || pet.CurrentBehavior == types.BehaviorExploring {
		t.Error("Pet should be home")
	}
	if pet.LastAdventure != nil || len(pet.Souvenirs) != 0 || pet.FriendshipPoints != points ||
		pet.TotalInteractions != interactions ||
		pet.Memory.GetLearnedAffinity(types.InteractionEnvironmentalEnrichment) != affinity {
		t.Error("Calling a pet home early should bring no souvenir, points, fun or learning")
	}

	// The trip does not finish later
	pet.Update(exploreDuration)
	if pet.LastAdventure != nil {
		t.Error("A recalled trip should not finish")
	}
}

func TestQuitHideAndSeekLosesGains(t *testing.T) {
	useRandom(t, 1)
	pet := NewDigitalPet("Biscuit", "user123")
	if _, err := pet.QuitHideAndSeek(); !errors.Is(err, ErrNoGame) {
		t.Errorf("Quitting without a game: %v", err)
	}

	pet.StartHideAndSeek()
	pet.GuessHidingSpot(0) // One wrong guess
	points, interactions := pet.FriendshipPoints, pet.TotalInteractions

	text, err := pet.QuitHideAndSeek()
	if err != nil || text != "Biscuit comes out from behind the couch. Maybe next time! 🛋️" {
		t.Fatalf("Quit = %q, %v", text, err)
	}
	if pet.PlayingHideAndSeek() || pet.FriendshipPoints != points || pet.TotalInteractions != interactions {
		t.Error("Quitting should end the game with no gains")
	}
	if _, err := pet.GuessHidingSpot(1); !errors.Is(err, ErrNoGame) {
		t.Errorf("Guessing after quitting: %v", err)
	}
}

func TestHibernatingPetActivityError(t *testing.T) {
	pet := NewDigitalPet("Biscuit", "user123")
	pet.HibernationEnabled = true
	pet.Update(10000)

	if _, err := pet.TrainTrick("sit"); !errors.Is(err, ErrHibernating) {
		t.Errorf("Training a hibernating pet: %v", err)
	}

	pet.Biology.IsAlive = false
	if err := pet.StartHideAndSeek(); !errors.Is(err, ErrPetNotAvailable) {
		t.Errorf("Playing with a pet that is not alive: %v", err)
	}
}

func TestSevereInjuryCanKillWithoutHibernation(t *testing.T) {
	useRandom(t, 0) // Injury
	pet := NewDigitalPet("Biscuit", "user123")
	setCalmHealthyState(pet)
	pet.SendExploring()
	pet.Biology.Vitals.Health = 0.2

	pet.Update(exploreDuration + 0.1)

	if pet.IsAlive() || pet.Biology.CauseOfDeath != "Health depleted" {
		t.Errorf("A fatal injury without hibernation should kill the pet, alive=%v cause=%q",
			pet.IsAlive(), pet.Biology.CauseOfDeath)
	}
}
