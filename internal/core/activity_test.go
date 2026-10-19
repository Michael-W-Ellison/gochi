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
