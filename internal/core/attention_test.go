package core

import (
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/Michael-W-Ellison/gochi/pkg/types"
)

// withAttentionStyle sets the traits that decide how much a pet enjoys
// attention
func withAttentionStyle(pet *DigitalPet, affectionate, independence, extraversion, neuroticism float64) *DigitalPet {
	traits := pet.Personality.Traits
	traits.Affectionate, traits.Independence, traits.Extraversion, traits.Neuroticism =
		affectionate, independence, extraversion, neuroticism
	return pet
}

func TestAttentionStyleFollowsPersonality(t *testing.T) {
	cuddly := withAttentionStyle(NewDigitalPet("Cuddles", "u"), 0.9, 0.2, 0.7, 0.2)
	balanced := NewDigitalPet("Biscuit", "u")
	aloof := withAttentionStyle(NewDigitalPet("Shadow", "u"), 0.2, 0.9, 0.3, 0.7)
	for pet, want := range map[*DigitalPet]string{cuddly: AttentionLoves, balanced: AttentionTolerates, aloof: AttentionAvoids} {
		if got := pet.AttentionStyle(); got != want {
			t.Errorf("%s: attention style %q, want %q", pet.Name, got, want)
		}
	}

	dog := NewDigitalPet("Rex", "u")
	dog.Species = SpeciesDog
	if dog.AttentionStyle() != AttentionLoves {
		t.Errorf("A balanced dog should love attention, got %q", dog.AttentionStyle())
	}
}

// petOften pets a pet once an hour and returns its likes and dislikes
func petOften(t *testing.T, pet *DigitalPet) (likes, dislikes []string) {
	clock := useClock(t, time.Date(2026, 10, 19, 9, 0, 0, 0, time.Local))
	for i := 0; i < 20; i++ {
		*clock = clock.Add(time.Hour)
		setCalmHealthyState(pet)
		pet.Emotions.Loneliness = 0.2 // Content, not craving company
		pet.Interact(types.InteractionPetting, 1.0)
	}
	return pet.LikesAndDislikes()
}

func TestLikesFollowPersonality(t *testing.T) {
	likes, _ := petOften(t, withAttentionStyle(NewDigitalPet("Cuddles", "u"), 0.9, 0.2, 0.7, 0.2))
	if !slices.Contains(likes, "being petted") {
		t.Errorf("A cuddly pet petted often should learn to like it: %v", likes)
	}

	likes, dislikes := petOften(t, NewDigitalPet("Biscuit", "u"))
	if slices.Contains(likes, "being petted") || slices.Contains(dislikes, "being petted") {
		t.Errorf("A pet that tolerates attention should neither like nor dislike petting: %v / %v", likes, dislikes)
	}
	if tolerates := NewDigitalPet("x", "u").Tolerates(); len(tolerates) != 0 {
		t.Errorf("A new pet tolerates nothing yet: %v", tolerates)
	}

	_, dislikes = petOften(t, withAttentionStyle(NewDigitalPet("Shadow", "u"), 0.2, 0.9, 0.3, 0.7))
	if !slices.Contains(dislikes, "being petted") {
		t.Errorf("An aloof pet petted often should learn to dislike it: %v", dislikes)
	}
}

func TestBalancedPetToleratesPetting(t *testing.T) {
	pet := NewDigitalPet("Biscuit", "u")
	petOften(t, pet)
	if tolerates := pet.Tolerates(); !slices.Contains(tolerates, "being petted") {
		t.Errorf("A pet that tolerates attention should be shown as tolerating petting: %v", tolerates)
	}
}

func TestUnwantedAttentionStresses(t *testing.T) {
	aloof := withAttentionStyle(NewDigitalPet("Shadow", "u"), 0.2, 0.9, 0.3, 0.7)
	setCalmHealthyState(aloof)
	stress, happiness := aloof.Biology.Vitals.Stress, aloof.Biology.Vitals.Happiness
	r := aloof.Interact(types.InteractionPetting, 1.0)
	if r.Reaction != ReactionReluctant {
		t.Fatalf("An aloof pet should be reluctant to be petted, got %s", r.Reaction)
	}
	if aloof.Biology.Vitals.Stress <= stress || aloof.Biology.Vitals.Happiness >= happiness {
		t.Error("Unwanted petting should stress the pet instead of pleasing it")
	}
	if !strings.Contains(r.Lesson, "twitching tail") {
		t.Errorf("Unwanted attention should teach the signs a cat has had enough: %q", r.Lesson)
	}

	cuddly := withAttentionStyle(NewDigitalPet("Cuddles", "u"), 0.9, 0.2, 0.7, 0.2)
	setCalmHealthyState(cuddly)
	happiness = cuddly.Biology.Vitals.Happiness
	if r := cuddly.Interact(types.InteractionPetting, 1.0); r.Reaction == ReactionReluctant || cuddly.Biology.Vitals.Happiness <= happiness {
		t.Errorf("A cuddly pet should enjoy petting: %+v", r)
	}
}

func TestTooMuchPettingOverstimulates(t *testing.T) {
	useClock(t, time.Date(2026, 10, 19, 9, 0, 0, 0, time.Local))
	pet := NewDigitalPet("Biscuit", "u")
	var r InteractionResult
	for i := 0; i < 10; i++ {
		setCalmHealthyState(pet)
		pet.Emotions.Loneliness = 0.5 // Starts out wanting company
		r = pet.Interact(types.InteractionPetting, 1.0)
		if i == 0 && r.Reaction == ReactionReluctant {
			t.Fatal("The first petting should be welcome")
		}
	}
	if r.Reaction != ReactionReluctant {
		t.Errorf("Petting again and again should overstimulate the pet, got %s", r.Reaction)
	}
}
