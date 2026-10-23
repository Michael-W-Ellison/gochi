package core

import (
	"strings"
	"testing"

	"github.com/Michael-W-Ellison/gochi/pkg/types"
)

// thingsByID returns ThingsToDo's activities by ID
func thingsByID(pet *DigitalPet) map[string]ThingToDo {
	things := map[string]ThingToDo{}
	for _, t := range pet.ThingsToDo() {
		things[t.ID] = t
	}
	return things
}

func TestThingsToDoForANewPet(t *testing.T) {
	pet := NewDigitalPet("Mochi", "user123")
	setCalmHealthyState(pet)
	list := pet.ThingsToDo()
	things := thingsByID(pet)
	if len(things) != 6 || len(list) != 6 {
		t.Fatalf("Six activities, each once: %+v", list)
	}
	for _, id := range []string{"train", "hide_seek", "explore", "enrich"} {
		if !things[id].Ready {
			t.Errorf("A well pet can %s now: %+v", id, things[id])
		}
	}
	if g := things["games"]; g.Ready || !strings.Contains(g.Why, "open after") {
		t.Errorf("A new pet's games aren't open yet: %+v", g)
	}
	if s := things["shows"]; s.Ready || !strings.Contains(s.Why, "too young") {
		t.Errorf("A young pet can't enter shows: %+v", s)
	}
	waiting := false
	for _, th := range list {
		if waiting && th.Ready {
			t.Errorf("Activities ready now come first: %+v", list)
		}
		waiting = waiting || !th.Ready
	}
}

func TestThingsToDoForASleepingPet(t *testing.T) {
	pet := NewDigitalPet("Mochi", "user123")
	setCalmHealthyState(pet)
	pet.CurrentBehavior = types.BehaviorSleeping
	things := thingsByID(pet)
	for _, id := range []string{"train", "hide_seek", "explore"} {
		if things[id].Ready || things[id].Why != "Mochi is asleep." {
			t.Errorf("A sleeping pet can't %s: %+v", id, things[id])
		}
	}
	if !things["enrich"].Ready {
		t.Errorf("Its home can be changed while it sleeps: %+v", things["enrich"])
	}
}

func TestThingsToDoOnceGamesOpen(t *testing.T) {
	pet := NewDigitalPet("Mochi", "user123")
	setCalmHealthyState(pet)
	pet.CareCounts = map[string]int{types.InteractionPlaying.String(): 1}
	if g := thingsByID(pet)["games"]; !g.Ready || !strings.Contains(g.Why, "Feather wand") {
		t.Errorf("A cat that has played can play with the feather wand: %+v", g)
	}
}

// Care the pet doesn't want suggests what it would do instead
func TestCareNotWantedSuggestsThingsToDo(t *testing.T) {
	pet := NewDigitalPet("Mochi", "user123")
	setCalmHealthyState(pet)
	pet.Biology.Vitals.Hydration, pet.Biology.Vitals.Nutrition = 1, 1

	drink := pet.Interact(types.InteractionDrinking, 1)
	if drink.Reaction != ReactionReluctant {
		t.Fatalf("A pet that isn't thirsty drinks reluctantly: %+v", drink)
	}
	if !strings.HasPrefix(drink.Suggestion, "Meanwhile, try ") || !strings.Contains(drink.Suggestion, "Train") {
		t.Errorf("It should suggest an activity ready now: %q", drink.Suggestion)
	}

	feed := pet.Interact(types.InteractionFeeding, 1)
	if feed.Applied || feed.Refusal == "" || feed.Suggestion == "" {
		t.Errorf("A full pet turns food down, with a suggestion: %+v", feed)
	}

	pet.Biology.Vitals.Nutrition = 0.5
	if fed := pet.Interact(types.InteractionFeeding, 1); fed.Reaction != ReactionReluctant && fed.Suggestion != "" {
		t.Errorf("Care the pet takes gladly suggests nothing: %+v", fed)
	}
}
