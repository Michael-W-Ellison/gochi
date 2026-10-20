package core

import (
	"testing"

	"github.com/Michael-W-Ellison/gochi/pkg/types"
)

func TestEveryBehaviorHasActivityDescription(t *testing.T) {
	for b := types.BehaviorIdle; b <= types.BehaviorInvestigating; b++ {
		if _, ok := activityDescriptions[b]; !ok {
			t.Errorf("No activity description for %s", b)
		}
	}
}

func TestLearnedBehaviorIsDescribed(t *testing.T) {
	pet := NewDigitalPet("TestPet", "user123")
	for i := 0; i < 10; i++ {
		pet.RecentCare = nil // Care spread out over time
		pet.ProcessUserInteraction(types.InteractionPlaying, 1.0)
		setCalmHealthyState(pet)
	}
	pet.updateBehavior()

	if got := ActivityDescription(pet.CurrentBehavior); got != "is playing by itself" {
		t.Errorf("A pet that learned to love play should be described as playing, got %q", got)
	}
}

func TestActivityLines(t *testing.T) {
	tests := []struct {
		name  string
		setup func(p *DigitalPet)
		want  string
	}{
		{"content", func(p *DigitalPet) { p.updateBehavior() }, "is relaxing"},
		{"tired", func(p *DigitalPet) { p.Biology.Vitals.Energy = 0.2; p.updateBehavior() }, "is sleeping"},
		{"sick", func(p *DigitalPet) { p.Biology.Vitals.Health = 0.2; p.updateBehavior() }, "is feeling sick"},
		{"hungry", func(p *DigitalPet) {
			p.Biology.Vitals.Nutrition = 0.2
			p.Emotions.Joy, p.Emotions.Contentment = 0, 0
			p.updateBehavior()
		}, "is looking for food"},
		{"on a trip", func(p *DigitalPet) {
			if err := p.SendExploring(); err != nil {
				panic(err)
			}
		}, "is out exploring"},
		{"hiding", func(p *DigitalPet) {
			if err := p.StartHideAndSeek(); err != nil {
				panic(err)
			}
		}, "is hiding somewhere"},
		{"hibernating", func(p *DigitalPet) {
			p.HibernationEnabled = true
			p.Biology.Vitals.Hydration = 0.05
			p.updateBehavior()
		}, "is hibernating"},
		{"dead", func(p *DigitalPet) {
			p.Biology.Vitals.Hydration = 0
			p.Biology.CheckDeathConditions()
		}, "has passed away"},
		{"curious at home", func(p *DigitalPet) { p.CurrentBehavior = types.BehaviorInvestigating }, "is sniffing around the house"},
	}
	for _, tt := range tests {
		pet := NewDigitalPet("Biscuit", "user123")
		setCalmHealthyState(pet)
		tt.setup(pet)
		if got := pet.Activity(); got != tt.want {
			t.Errorf("%s: Activity() = %q, want %q (behavior %s)", tt.name, got, tt.want, pet.CurrentBehavior)
		}
	}
}

func TestLikesAndDislikes(t *testing.T) {
	pet := NewDigitalPet("Biscuit", "user123")

	if likes, dislikes := pet.LikesAndDislikes(); len(likes) != 0 || len(dislikes) != 0 {
		t.Errorf("A new pet should have no likes or dislikes, got %v / %v", likes, dislikes)
	}

	for i := 0; i < 10; i++ {
		pet.RecentCare = nil // Care spread out over time
		pet.ProcessUserInteraction(types.InteractionPlaying, 1.0)
		setCalmHealthyState(pet)
	}
	for i := 0; i < 3; i++ {
		pet.RecentCare = nil // Care spread out over time
		pet.ProcessUserInteraction(types.InteractionPetting, 1.0)
		setCalmHealthyState(pet)
	}
	for i := 0; i < 10; i++ {
		pet.RecentCare = nil // Care spread out over time
		pet.ProcessUserInteraction(types.InteractionDiscipline, 1.0)
		setCalmHealthyState(pet)
	}

	likes, dislikes := pet.LikesAndDislikes()
	if len(likes) == 0 || likes[0] != "playing" {
		t.Errorf("Playing should be the strongest like, got %v", likes)
	}
	if len(dislikes) != 1 || dislikes[0] != "being scolded" {
		t.Errorf("Dislikes = %v, want [being scolded]", dislikes)
	}
}

func TestPersonalityDescriptionUsesName(t *testing.T) {
	pet := NewDigitalPet("Biscuit", "user123")

	if got := pet.PersonalityDescription(); got != "Biscuit is well-balanced." {
		t.Errorf("PersonalityDescription = %q", got)
	}
}

func TestRequestsByUrgency(t *testing.T) {
	pet := NewDigitalPet("Biscuit", "user123")
	setCalmHealthyState(pet)
	if r := pet.CurrentRequest(); r != nil {
		t.Errorf("A content pet should not ask for anything, got %+v", r)
	}

	pet.Biology.Vitals.Nutrition = 0.2
	pet.Biology.Vitals.Hydration = 0.2
	if r := pet.CurrentRequest(); r == nil || r.Action != "drink" {
		t.Errorf("Thirst should come first, got %+v", r)
	}

	pet.ProcessUserInteraction(types.InteractionDrinking, 1.0)
	if r := pet.CurrentRequest(); r == nil || r.Action != "feed" {
		t.Errorf("After drinking the pet should ask for food, got %+v", r)
	}

	pet.ProcessUserInteraction(types.InteractionFeeding, 1.0)
	pet.Biology.Vitals.Cleanliness = 0.1
	if r := pet.CurrentRequest(); r == nil || r.Action != "groom" {
		t.Errorf("A messy pet should ask to be groomed, got %+v", r)
	}
}

func TestLonelyPetAsksForFavoriteAttention(t *testing.T) {
	pet := NewDigitalPet("Biscuit", "user123")
	setCalmHealthyState(pet)

	// Left alone longer than lonelyAfter
	pet.Biology.Processes.Age = lonelyAfter + 1
	if r := pet.CurrentRequest(); r == nil || r.Action != "pet" {
		t.Errorf("A lonely pet should ask to be petted by default, got %+v", r)
	}

	// A pet that has learned to love play asks to play instead
	for i := 0; i < 10; i++ {
		pet.RecentCare = nil // Care spread out over time
		pet.ProcessUserInteraction(types.InteractionPlaying, 1.0)
		setCalmHealthyState(pet)
	}
	pet.Biology.Processes.Age += lonelyAfter + 1
	if r := pet.CurrentRequest(); r == nil || r.Action != "play" {
		t.Errorf("A lonely pet that loves play should ask to play, got %+v", r)
	}
}

func TestSleepingPetOnlyAsksForFoodAndWater(t *testing.T) {
	pet := NewDigitalPet("Biscuit", "user123")
	setCalmHealthyState(pet)
	pet.CurrentBehavior = types.BehaviorSleeping
	pet.Biology.Vitals.Cleanliness = 0.1
	pet.Biology.Processes.Age = lonelyAfter + 1

	if r := pet.CurrentRequest(); r != nil {
		t.Errorf("A sleeping pet should not ask for grooming or attention, got %+v", r)
	}

	pet.Biology.Vitals.Hydration = 0.2
	if r := pet.CurrentRequest(); r == nil || r.Action != "drink" {
		t.Errorf("A sleeping pet should still ask for water, got %+v", r)
	}
}

func TestHibernatingPetAsksToBeRevived(t *testing.T) {
	pet := NewDigitalPet("Biscuit", "user123")
	pet.HibernationEnabled = true
	pet.Update(10000)

	if r := pet.CurrentRequest(); r == nil || r.Action != "revive" {
		t.Errorf("A hibernating pet should ask to be revived, got %+v", r)
	}
}
