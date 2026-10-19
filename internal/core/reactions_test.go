package core

import (
	"strings"
	"testing"

	"github.com/Michael-W-Ellison/gochi/pkg/types"
)

func TestReactionDependsOnNeed(t *testing.T) {
	hungry := NewDigitalPet("Biscuit", "user123")
	hungry.Biology.Vitals.Nutrition = 0.1
	full := NewDigitalPet("Biscuit", "user123")
	full.Biology.Vitals.Nutrition = 1.0

	h := hungry.Interact(types.InteractionFeeding, 1.0)
	f := full.Interact(types.InteractionFeeding, 1.0)

	if h.Reaction != ReactionDelighted || h.ReactionText != "Biscuit gobbles up every bite!" {
		t.Errorf("A hungry pet should be delighted to eat, got %s: %q", h.Reaction, h.ReactionText)
	}
	if f.Reaction != ReactionReluctant || !strings.Contains(f.ReactionText, "isn't hungry") {
		t.Errorf("A full pet should be reluctant to eat, got %s: %q", f.Reaction, f.ReactionText)
	}
	if h.ReactionMultiplier <= f.ReactionMultiplier {
		t.Error("A hungry pet should get more out of a meal than a full one")
	}
}

func TestReactionDependsOnPersonality(t *testing.T) {
	cuddly := NewDigitalPet("Cuddly", "user123")
	cuddly.Personality.Traits.Affectionate = 1.0
	cuddly.Personality.Traits.Independence = 0.0
	aloof := NewDigitalPet("Aloof", "user123")
	aloof.Personality.Traits.Affectionate = 0.0
	aloof.Personality.Traits.Independence = 1.0

	c := cuddly.Interact(types.InteractionPetting, 1.0)
	a := aloof.Interact(types.InteractionPetting, 1.0)

	if c.ReactionMultiplier <= a.ReactionMultiplier {
		t.Errorf("An affectionate pet should enjoy petting more than an independent one: %f vs %f",
			c.ReactionMultiplier, a.ReactionMultiplier)
	}
	if a.Reaction != ReactionReluctant {
		t.Errorf("An independent pet that is not lonely should be reluctant, got %s", a.Reaction)
	}
}

func TestReactionGrowsWithLearnedLiking(t *testing.T) {
	pet := NewDigitalPet("Biscuit", "user123")
	setCalmHealthyState(pet)
	first := pet.Interact(types.InteractionPlaying, 1.0)

	for i := 0; i < 15; i++ {
		setCalmHealthyState(pet)
		pet.Interact(types.InteractionPlaying, 1.0)
	}
	setCalmHealthyState(pet)
	later := pet.Interact(types.InteractionPlaying, 1.0)

	if later.ReactionMultiplier <= first.ReactionMultiplier {
		t.Errorf("A pet that has learned to love play should enjoy it more: %f -> %f",
			first.ReactionMultiplier, later.ReactionMultiplier)
	}
}

func TestTiredPetIsReluctantToPlay(t *testing.T) {
	pet := NewDigitalPet("Biscuit", "user123")
	pet.Biology.Vitals.Energy = 0.05
	pet.Biology.Vitals.Nutrition = 0.8 // Not critical

	r := pet.Interact(types.InteractionPlaying, 1.0)
	if r.Reaction != ReactionReluctant || r.ReactionText != "Biscuit is too tired to play much." {
		t.Errorf("Got %s: %q", r.Reaction, r.ReactionText)
	}
}

func TestReactionMultiplierBounds(t *testing.T) {
	pet := NewDigitalPet("Biscuit", "user123")
	for it := types.InteractionFeeding; it <= types.InteractionDrinking; it++ {
		for _, nutrition := range []float64{0.0, 1.0} {
			pet.Biology.Vitals.Nutrition = nutrition
			_, m := pet.reactionLocked(it)
			if m < 0.6 || m > 1.4 {
				t.Errorf("%s multiplier %f outside [0.6, 1.4]", it, m)
			}
		}
	}

	if level, m := pet.reactionLocked(types.InteractionDiscipline); level != ReactionReluctant || m != 1.0 {
		t.Errorf("Discipline should be reluctant with no scaling, got %s %f", level, m)
	}
}
