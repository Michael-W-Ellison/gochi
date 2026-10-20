package core

import (
	"math"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/Michael-W-Ellison/gochi/internal/ai"
	"github.com/Michael-W-Ellison/gochi/pkg/types"
)

func TestStateMachineChoosesBehavior(t *testing.T) {
	pet := NewDigitalPet("Biscuit", "user123")
	setCalmHealthyState(pet)
	pet.updateBehavior()
	if pet.CurrentBehavior != types.BehaviorIdle {
		t.Fatalf("A calm, balanced pet should idle, got %s", pet.CurrentBehavior)
	}

	pet.Biology.Vitals.Energy = 0.2
	pet.updateBehavior()
	if pet.CurrentBehavior != types.BehaviorSleeping {
		t.Fatalf("A tired pet should sleep, got %s", pet.CurrentBehavior)
	}
	// Hysteresis: it keeps sleeping until rested
	pet.Biology.Vitals.Energy = 0.45
	pet.updateBehavior()
	if pet.CurrentBehavior != types.BehaviorSleeping {
		t.Errorf("A pet should sleep until rested, got %s", pet.CurrentBehavior)
	}

	setCalmHealthyState(pet)
	pet.updateBehavior() // Rested, it wakes up
	if err := pet.SendExploring(); err != nil {
		t.Fatal(err)
	}
	pet.updateBehavior()
	if pet.CurrentBehavior != types.BehaviorExploring {
		t.Errorf("A pet on a trip should be exploring, got %s", pet.CurrentBehavior)
	}

	history := pet.behavior.GetTransitionHistory(0)
	if len(history) < 2 || history[0].ToState != types.BehaviorSleeping {
		t.Errorf("The state machine should record its transitions: %+v", history)
	}
}

// playfulPet returns a calm pet just playful enough to choose to play
func playfulPet() *DigitalPet {
	pet := NewDigitalPet("Biscuit", "user123")
	setCalmHealthyState(pet)
	traits := pet.Personality.Traits
	traits.Playfulness, traits.EnergyLevel, traits.Extraversion = 0.7, 0.6, 0.6
	return pet
}

func TestPetLearnsWhatOwnerDiscourages(t *testing.T) {
	pet := playfulPet()
	if got := pet.chooseDiscretionaryBehavior(); got != types.BehaviorPlaying {
		t.Fatalf("The pet should start out choosing to play, got %s", got)
	}

	// The owner scolds it whenever it plays
	context := pet.learningContextLocked()
	for i := 0; i < 30; i++ {
		pet.learnFromResponseLocked(types.BehaviorPlaying, context, -1)
	}
	if pet.learnedRewardLocked(types.BehaviorPlaying) >= 0 {
		t.Fatal("Playing should have a negative learned reward")
	}
	if got := pet.chooseDiscretionaryBehavior(); got == types.BehaviorPlaying {
		t.Error("A pet scolded for playing should stop choosing to play")
	}
}

func TestPetLearnsWhatOwnerRewards(t *testing.T) {
	pet := NewDigitalPet("Biscuit", "user123")
	setCalmHealthyState(pet)
	if got := pet.chooseDiscretionaryBehavior(); got != types.BehaviorIdle {
		t.Fatalf("A balanced pet should idle, got %s", got)
	}

	// Each time it seeks attention, the owner pets it
	context := pet.learningContextLocked()
	for i := 0; i < 30; i++ {
		pet.learnFromResponseLocked(types.BehaviorSocialInteraction, context, 0.9)
	}
	if got := pet.chooseDiscretionaryBehavior(); got != types.BehaviorSocialInteraction {
		t.Errorf("A pet rewarded for seeking attention should seek it more, got %s", got)
	}

	// Only behaviors the pet chooses are learned
	pet.learnFromResponseLocked(types.BehaviorSleeping, context, 1)
	if pet.Learning.GetQValue(context, types.BehaviorSleeping) != 0 {
		t.Error("Behaviors the pet doesn't choose should not be learned")
	}
}

func TestCareRewardsCurrentBehavior(t *testing.T) {
	pet := NewDigitalPet("Biscuit", "user123")
	setCalmHealthyState(pet)
	pet.CurrentBehavior = types.BehaviorSocialInteraction
	context := pet.learningContextLocked()
	pet.Interact(types.InteractionPetting, 1.0)
	if pet.Learning == nil || pet.Learning.GetQValue(context, types.BehaviorSocialInteraction) <= 0 {
		t.Error("Petting a pet that wants attention should reward seeking attention")
	}
}

func TestPersonalityChangesWithCare(t *testing.T) {
	clock := useClock(t, time.Date(2026, 10, 19, 9, 0, 0, 0, time.Local))
	pet := NewDigitalPet("Biscuit", "user123")
	start := *pet.Personality.Traits
	for i := 0; i < 40; i++ {
		*clock = clock.Add(time.Hour) // Care spread out over time
		setCalmHealthyState(pet)
		pet.Interact(types.InteractionPlaying, 1.0)
	}
	if pet.Personality.Traits.Playfulness <= start.Playfulness+0.05 {
		t.Errorf("Enjoyed play should make a pet more playful: %f -> %f", start.Playfulness, pet.Personality.Traits.Playfulness)
	}
	if changes := pet.PersonalityChanges(); !slices.Contains(changes, "more playful") {
		t.Errorf("The change should be described: %v", changes)
	}

	scolded := NewDigitalPet("Mochi", "user123")
	for i := 0; i < 20; i++ {
		*clock = clock.Add(time.Hour)
		setCalmHealthyState(scolded)
		scolded.Interact(types.InteractionDiscipline, 1.0)
	}
	if changes := scolded.PersonalityChanges(); !slices.Contains(changes, "more anxious") {
		t.Errorf("A pet scolded often should become more anxious: %v", changes)
	}
}

func TestNeglectChangesPersonality(t *testing.T) {
	pet := NewDigitalPet("Biscuit", "user123")
	start := *pet.Personality.Traits
	for i := 0; i < 60; i++ {
		pet.Emotions.Loneliness = 0.9
		pet.mu.Lock()
		pet.evolveFromNeglectLocked(1)
		pet.mu.Unlock()
	}
	traits := pet.Personality.Traits
	if traits.Independence <= start.Independence || traits.Loyalty >= start.Loyalty || traits.Neuroticism <= start.Neuroticism {
		t.Errorf("A neglected pet should become more independent, less attached and more anxious: %+v", *traits)
	}
	if changes := pet.PersonalityChanges(); !slices.Contains(changes, "more independent") {
		t.Errorf("The change should be described: %v", changes)
	}
}

func TestInjuryMakesPetAnxious(t *testing.T) {
	pet := NewDigitalPet("Biscuit", "user123")
	start := pet.Personality.Traits.Neuroticism
	pet.mu.Lock()
	pet.injureLocked()
	pet.mu.Unlock()
	if pet.Personality.Traits.Neuroticism <= start {
		t.Error("An injury should leave the pet more anxious")
	}
}

func TestMindSurvivesSaveLoad(t *testing.T) {
	pet := playfulPet()
	context := pet.learningContextLocked()
	pet.learnFromResponseLocked(types.BehaviorPlaying, context, -1)
	pet.evolveLocked(ai.ExperienceData{TraitAffected: "playfulness", Intensity: 10})

	data, err := pet.Save()
	if err != nil {
		t.Fatal(err)
	}
	loaded, err := Load(data)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := loaded.Learning.GetQValue(context, types.BehaviorPlaying), pet.Learning.GetQValue(context, types.BehaviorPlaying); got != want {
		t.Errorf("Learned rewards should be saved: %f, want %f", got, want)
	}
	if changes := loaded.PersonalityChanges(); !slices.Contains(changes, "more playful") {
		t.Errorf("The personality baseline should be saved: %v", changes)
	}
}

func TestRepeatedCareDiminishes(t *testing.T) {
	clock := useClock(t, time.Date(2026, 10, 19, 9, 0, 0, 0, time.Local))
	pet := NewDigitalPet("Biscuit", "user123")

	// The nth repeat in a row counts 1/n
	for i, want := range []float64{1, 0.5, 1.0 / 3} {
		if got := pet.repeatWeightLocked(types.InteractionPetting); math.Abs(got-want) > 1e-9 {
			t.Errorf("Repeat %d: weight %f, want %f", i+1, got, want)
		}
	}
	// Other kinds of care are counted separately
	if got := pet.repeatWeightLocked(types.InteractionPlaying); got != 1 {
		t.Errorf("A different kind of care should count fully, got %f", got)
	}
	// Repeats fade with time: after one half-life, three repeats count as 1.5
	*clock = clock.Add(repeatHalfLife)
	if got := pet.repeatWeightLocked(types.InteractionPetting); math.Abs(got-1/2.5) > 1e-9 {
		t.Errorf("After a half-life, weight %f, want %f", got, 1/2.5)
	}
	*clock = clock.Add(time.Hour)
	if got := pet.repeatWeightLocked(types.InteractionPetting); got < 0.99 {
		t.Errorf("Care long after should count fully again, got %f", got)
	}
}

func TestSpamClickingChangesPersonalityLess(t *testing.T) {
	clock := useClock(t, time.Date(2026, 10, 19, 9, 0, 0, 0, time.Local))
	spaced, spammed := NewDigitalPet("Biscuit", "user123"), NewDigitalPet("Mochi", "user123")
	var last InteractionResult
	for i := 0; i < 40; i++ { // Clicks a second apart
		*clock = clock.Add(time.Second)
		setCalmHealthyState(spammed)
		last = spammed.Interact(types.InteractionPlaying, 1.0)
	}
	for i := 0; i < 40; i++ { // Play an hour apart
		*clock = clock.Add(time.Hour)
		setCalmHealthyState(spaced)
		spaced.Interact(types.InteractionPlaying, 1.0)
	}

	start := NewDigitalPet("Start", "user123").Personality.Traits.Playfulness
	spacedGain := spaced.Personality.Traits.Playfulness - start
	spammedGain := spammed.Personality.Traits.Playfulness - start
	t.Logf("Playfulness gained: 40 rapid play sessions %.4f, 40 spread out %.4f", spammedGain, spacedGain)
	if spammedGain <= 0 || spammedGain > spacedGain/4 {
		t.Errorf("40 rapid play sessions should change a pet far less than 40 spread out: %f vs %f", spammedGain, spacedGain)
	}
	if !strings.Contains(last.Repeated, "won't change how Mochi grows") {
		t.Errorf("The player should be told repeated care no longer counts: %q", last.Repeated)
	}
	if spammed.Biology.Vitals.Happiness < 0.8 {
		t.Error("Repeated play should still make the pet happy")
	}

	// Repeats also count less towards learned likes
	spacedExp := spaced.Memory.Preferences[types.InteractionPlaying].Experience
	spammedExp := spammed.Memory.Preferences[types.InteractionPlaying].Experience
	if spacedExp < 39.9 || spammedExp > 6 {
		t.Errorf("Learned likes should count repeats less: %f spread out, %f rapid", spacedExp, spammedExp)
	}
}
