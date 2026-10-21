package core

import (
	"errors"
	"strings"
	"testing"

	"github.com/Michael-W-Ellison/gochi/internal/biology"
	"github.com/Michael-W-Ellison/gochi/internal/environment"
	"github.com/Michael-W-Ellison/gochi/pkg/types"
)

func TestLifeStatus(t *testing.T) {
	pet := NewDigitalPet("Biscuit", "user123")
	life := pet.LifeStatus()
	if life.Stage != "Young" || !life.OldAgeDeath || life.Potions != 0 || life.Remaining <= 0 ||
		life.Lifespan != life.Age+life.Remaining {
		t.Errorf("A new pet: %+v", life)
	}

	// Cats live a little longer than dogs
	dog := NewDigitalPet("Rex", "user123")
	dog.Species = SpeciesDog
	dog.Genome = pet.Genome.Clone() // The same genes, so only the species differs
	if dog.LifeStatus().Lifespan >= life.Lifespan {
		t.Errorf("Dog lifespan %.0f should be shorter than a cat's %.0f", dog.LifeStatus().Lifespan, life.Lifespan)
	}

	pet.HibernationEnabled = true
	if pet.LifeStatus().OldAgeDeath {
		t.Error("Gentle mode pets don't die of old age")
	}
}

func TestOldAgeEndsLifeExceptInGentleMode(t *testing.T) {
	for _, gentle := range []bool{false, true} {
		pet := NewDigitalPet("Biscuit", "user123")
		pet.HibernationEnabled = gentle
		pet.Biology.Processes.Age = pet.LifeStatus().Lifespan + 50
		setCalmHealthyState(pet)
		before := pet.Conditions()
		pet.Update(0.01)

		if gentle {
			if !pet.IsAlive() {
				t.Error("A gentle mode pet should live on past its lifespan")
			}
			continue
		}
		if pet.IsAlive() || pet.Biology.CauseOfDeath != "Old age" {
			t.Fatalf("An old pet should die of old age: alive %v, %q", pet.IsAlive(), pet.Biology.CauseOfDeath)
		}
		events := ConditionEvents(before, pet.Conditions())
		if len(events) != 1 || events[0].Message != "Biscuit died of old age, after a long life." {
			t.Errorf("Death event: %+v", events)
		}
	}
}

func TestOldAgeOutranksIllnessAsCauseOfDeath(t *testing.T) {
	pet := NewDigitalPet("Biscuit", "user123")
	pet.Illness = &Illness{Kind: IllnessRespiratory, Severity: 0.1}
	pet.Biology.Processes.Age = pet.LifeStatus().Lifespan + 50
	setCalmHealthyState(pet)
	pet.Update(0.01)
	if pet.Biology.CauseOfDeath != "Old age" {
		t.Errorf("Cause of death = %q", pet.Biology.CauseOfDeath)
	}
}

func TestLifePotion(t *testing.T) {
	pet := NewDigitalPet("Biscuit", "user123")
	if _, err := pet.GiveLifePotion(); !errors.Is(err, ErrNoPotion) {
		t.Errorf("No potion: %v", err)
	}

	pet.FriendshipPoints, pet.PointsEarned = 500, 500
	if err := pet.BuySupply(LifePotionID); !errors.Is(err, ErrSupplyLocked) {
		t.Errorf("The potion is legendary and needs 600 lifetime points: %v", err)
	}
	pet.PointsEarned = 600
	if err := pet.BuySupply(LifePotionID); !errors.Is(err, ErrNotEnoughPoints) {
		t.Errorf("The potion costs 1,000 points: %v", err)
	}
	pet.FriendshipPoints = 2000
	if err := pet.BuySupply(LifePotionID); err != nil {
		t.Fatal(err)
	}
	if pet.LifeStatus().Potions != 1 {
		t.Fatal("The potion should be in the pantry")
	}

	before := pet.LifeStatus().Lifespan
	text, err := pet.GiveLifePotion()
	if err != nil || !strings.HasPrefix(text, "Biscuit drinks the potion") {
		t.Fatalf("GiveLifePotion = %q, %v", text, err)
	}
	life := pet.LifeStatus()
	if life.Lifespan != before+lifePotionDays || life.Extension != lifePotionDays || life.Potions != 0 {
		t.Errorf("After the potion: %+v (before %.0f)", life, before)
	}
}

func TestPotionKeepsAnOldPetAlive(t *testing.T) {
	pet := NewDigitalPet("Biscuit", "user123")
	pet.Biology.Processes.Age = pet.LifeStatus().Lifespan - 1
	pet.Biology.LifeExtension = lifePotionDays
	setCalmHealthyState(pet)
	pet.Update(2)
	if !pet.IsAlive() {
		t.Error("An extended lifespan should keep the pet alive")
	}
}

func TestGrowingUpAndAging(t *testing.T) {
	pet := NewDigitalPet("Biscuit", "user123")
	pet.HibernationEnabled = true
	pet.Biology.Processes.Age = biology.AdultAge - 0.01
	setCalmHealthyState(pet)
	before := pet.Conditions()
	pet.Update(0.05)
	var grownUp bool
	for _, e := range ConditionEvents(before, pet.Conditions()) {
		grownUp = grownUp || (e.Type == EventLifeStage && e.Message == "Biscuit is all grown up!")
	}
	if !grownUp {
		t.Error("Becoming an adult should be an event")
	}

	pet.Biology.Processes.Age = pet.LifeStatus().Lifespan - 100
	pet.HibernationEnabled = false
	pet.prepareBodyLocked()
	var frail bool
	for _, s := range pet.Symptoms() {
		frail = frail || s.Sign == "Very old and frail"
	}
	if !frail {
		t.Errorf("A pet near the end of its life should show it: %+v", pet.Symptoms())
	}
}

func TestOldPetsLearnTricksMoreSlowly(t *testing.T) {
	sessions := func(age float64) float64 {
		pet := NewDigitalPet("Biscuit", "user123")
		pet.HibernationEnabled = true
		pet.Biology.Processes.Age = age
		pet.Biology.Update(0.001) // Work out its abilities for its age
		setCalmHealthyState(pet)
		result, err := pet.TrainTrick("sit")
		if err != nil {
			t.Fatal(err)
		}
		return result.Skill
	}
	if young, old := sessions(10), sessions(biology.BaseLifespan-10); old >= young {
		t.Errorf("A young pet should learn faster: %.3f vs %.3f", young, old)
	}
}

func TestBreedingAges(t *testing.T) {
	young, adult := NewDigitalPet("Young", "u"), adultPet("Adult")
	young.Biology.Processes.Age = biology.AdultAge - 1
	if _, reason := BreedingCompatibility(young, adult); !strings.Contains(reason, "too young") {
		t.Errorf("Young pets can't breed: %q", reason)
	}
}

func TestProcessEnvironmentalInput(t *testing.T) {
	pet := NewDigitalPet("Biscuit", "user123")
	pet.ProcessEnvironmentalInput(EnvironmentData{TimeOfDay: 2, Daylight: 0, AirQuality: 0.2})
	if b := pet.Biology; b.Inputs.Daylight != 0 || b.Inputs.TimeOfDay != 2 || b.AirPollution < 0.79 {
		t.Errorf("Inputs not applied: %+v, pollution %.2f", b.Inputs, b.AirPollution)
	}

	// The pet's own environment: daylight follows the time of day
	pet.Environment.SetTimeOfDay(13)
	pet.Environment.ForceWeather(environment.WeatherClear)
	if data := pet.environmentDataLocked(); data.Daylight != 1 || data.TimeOfDay != 13 {
		t.Errorf("Midday: %+v", data)
	}
	pet.Environment.SetTimeOfDay(1)
	if data := pet.environmentDataLocked(); data.Daylight != 0 {
		t.Errorf("Night: %+v", data)
	}
}

func TestPetsFallAsleepMoreEasilyAtNight(t *testing.T) {
	sleepsAt := func(hour float64) bool {
		pet := NewDigitalPet("Biscuit", "user123")
		setCalmHealthyState(pet)
		pet.Biology.Vitals.Fatigue = 0.6
		pet.Environment.SetTimeOfDay(hour)
		pet.updateBehavior()
		return pet.CurrentBehavior == types.BehaviorSleeping
	}
	if sleepsAt(14) || !sleepsAt(2) {
		t.Errorf("A fairly tired pet should sleep at 2am (%v) but not 2pm (%v)", sleepsAt(2), sleepsAt(14))
	}
}

func TestAffectionReleasesOxytocin(t *testing.T) {
	pet := NewDigitalPet("Biscuit", "user123")
	pet.Personality.Traits.Affectionate, pet.Personality.Traits.Independence = 0.9, 0.1
	setCalmHealthyState(pet)
	before := pet.Biology.Hormones.Oxytocin
	pet.Interact(types.InteractionPetting, 1)
	if pet.Biology.Hormones.Oxytocin <= before {
		t.Error("Petting an affectionate pet should release oxytocin")
	}

	before = pet.Biology.Hormones.Cortisol
	pet.Interact(types.InteractionDiscipline, 1)
	if pet.Biology.Hormones.Cortisol <= before {
		t.Error("Scolding should release cortisol")
	}
}

func TestCatsGroomThemselves(t *testing.T) {
	cat, dog := NewDigitalPet("Biscuit", "u"), NewDigitalPet("Rex", "u")
	dog.Species = SpeciesDog
	for _, pet := range []*DigitalPet{cat, dog} {
		setCalmHealthyState(pet)
		pet.Biology.Vitals.Cleanliness = 0.5
		pet.Emotions.Excitement = 0
		pet.Biology.Vitals.Happiness = 0.5 // Not happy enough to be "happy" first
	}
	if got := cat.chooseDiscretionaryBehavior(); got != types.BehaviorGrooming {
		t.Errorf("A messy cat should groom itself, got %s", got)
	}
	if got := dog.chooseDiscretionaryBehavior(); got == types.BehaviorGrooming {
		t.Error("A dog grooms itself much less")
	}

	cat.CurrentBehavior = types.BehaviorGrooming
	cat.selfCareLocked(1)
	if cat.Biology.Vitals.Cleanliness <= 0.5 {
		t.Error("Grooming itself should clean the cat")
	}
}

func TestEnergeticPetsRunAround(t *testing.T) {
	pet := NewDigitalPet("Rex", "u")
	pet.Species = SpeciesDog
	setCalmHealthyState(pet)
	pet.Personality.Traits.EnergyLevel = 0.9
	pet.ensureNeedsLocked()
	pet.Needs.SetNeedLevel(types.NeedExercise, 0.2)
	if got := pet.selfExerciseScoreLocked(); got <= discretionaryThreshold {
		t.Errorf("An energetic dog needing exercise should want to run around, score %.2f", got)
	}

	pet.CurrentBehavior = types.BehaviorExercising
	pet.Biology.Physiology.CardiovascularHealth = 0.5
	energy := pet.Biology.Vitals.Energy
	pet.selfCareLocked(1)
	if pet.Biology.Physiology.CardiovascularHealth <= 0.5 || pet.Biology.Vitals.Energy >= energy {
		t.Error("Running around should build fitness and use energy")
	}
	if pet.Activity() != "is running around" {
		t.Errorf("Activity = %q", pet.Activity())
	}
}

func TestLearningSummaryAndPredictedRequests(t *testing.T) {
	pet := NewDigitalPet("Biscuit", "user123")
	if rewarded, trends := pet.LearningSummary(); len(rewarded) != 0 || len(trends) != 0 {
		t.Errorf("A new pet has learned nothing: %v %v", rewarded, trends)
	}
	pet.Personality.Traits.Playfulness = 0.9
	for i := 0; i < 15; i++ {
		setCalmHealthyState(pet)
		pet.RecentCare = nil
		pet.CurrentBehavior = types.BehaviorPlaying
		pet.Interact(types.InteractionPlaying, 1)
	}
	rewarded, trends := pet.LearningSummary()
	if len(rewarded) == 0 || !strings.Contains(strings.Join(rewarded, ","), "playing") {
		t.Errorf("Rewarded: %v", rewarded)
	}
	if len(trends) == 0 || trends[0] != "more playful" {
		t.Errorf("Trends: %v", trends)
	}
	if pet.expectedEnjoymentLocked(types.InteractionPlaying) <= pet.expectedEnjoymentLocked(types.InteractionDiscipline) {
		t.Error("A pet that enjoyed play should expect to enjoy it")
	}
}

func TestPotionsAreLimited(t *testing.T) {
	pet := NewDigitalPet("Biscuit", "user123")
	pet.PointsEarned, pet.FriendshipPoints = 5000, 5000
	for i := 0; i < MaxLifePotions+1; i++ {
		if err := pet.BuySupply(LifePotionID); err != nil {
			t.Fatal(err)
		}
	}
	for i := 0; i < MaxLifePotions; i++ {
		if _, err := pet.GiveLifePotion(); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := pet.GiveLifePotion(); !errors.Is(err, ErrPotionsUsedUp) {
		t.Errorf("Another potion shouldn't help: %v", err)
	}
	if life := pet.LifeStatus(); life.Potions != 1 || life.Extension != MaxLifePotions*lifePotionDays {
		t.Errorf("The extra potion should stay in the pantry: %+v", life)
	}
}
