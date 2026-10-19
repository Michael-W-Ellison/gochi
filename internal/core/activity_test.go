package core

import (
	"testing"

	"github.com/Michael-W-Ellison/gochi/pkg/types"
)

func TestEveryBehaviorHasActivityDescription(t *testing.T) {
	for b := types.BehaviorIdle; b <= types.BehaviorHibernating; b++ {
		if _, ok := activityDescriptions[b]; !ok {
			t.Errorf("No activity description for %s", b)
		}
	}
}

func TestLearnedBehaviorIsDescribed(t *testing.T) {
	pet := NewDigitalPet("TestPet", "user123")
	for i := 0; i < 10; i++ {
		pet.ProcessUserInteraction(types.InteractionPlaying, 1.0)
		setCalmHealthyState(pet)
	}
	pet.updateBehavior()

	if got := ActivityDescription(pet.CurrentBehavior); got != "is playing" {
		t.Errorf("A pet that learned to love play should be described as playing, got %q", got)
	}
}

func TestLikesAndDislikes(t *testing.T) {
	pet := NewDigitalPet("Biscuit", "user123")

	if likes, dislikes := pet.LikesAndDislikes(); len(likes) != 0 || len(dislikes) != 0 {
		t.Errorf("A new pet should have no likes or dislikes, got %v / %v", likes, dislikes)
	}

	for i := 0; i < 10; i++ {
		pet.ProcessUserInteraction(types.InteractionPlaying, 1.0)
		setCalmHealthyState(pet)
	}
	for i := 0; i < 3; i++ {
		pet.ProcessUserInteraction(types.InteractionPetting, 1.0)
		setCalmHealthyState(pet)
	}
	for i := 0; i < 10; i++ {
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
