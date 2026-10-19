package core

import (
	"slices"
	"testing"

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
	pet := NewDigitalPet("Biscuit", "user123")
	start := *pet.Personality.Traits
	for i := 0; i < 40; i++ {
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
