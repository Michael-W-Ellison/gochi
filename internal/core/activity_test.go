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
