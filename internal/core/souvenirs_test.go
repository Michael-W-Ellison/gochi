package core

import (
	"errors"
	"testing"

	"github.com/Michael-W-Ellison/gochi/pkg/types"
)

func TestSouvenirSets(t *testing.T) {
	sets := SouvenirSets()
	if len(sets) != 12 {
		t.Errorf("%d places have souvenirs, want 12", len(sets))
	}
	for place, set := range sets {
		rare := 0
		for _, s := range set {
			if s.Rare {
				rare++
			}
		}
		if len(set) != 5 || rare != 1 {
			t.Errorf("%s: %d souvenirs, %d rare", place, len(set), rare)
		}
	}
	// Souvenirs from before sets keep their place
	for id, place := range map[string]string{"pebble": "park", "shell": "beach", "feather": "farm", "leaf": "forest",
		"flower": "meadow", "crystal": "forest", "acorn": "park"} {
		if s, ok := findSouvenir(id); !ok || s.Place != place {
			t.Errorf("%s should be from the %s", id, place)
		}
	}
	pet := NewDigitalPet("Mochi", "o")
	for i := 0; i < 20; i++ {
		if s := pet.pickSouvenirLocked("beach"); s.Place != "beach" {
			t.Fatalf("A beach trip found %s", s.ID)
		}
	}
}

func TestGiftsAndRareFinds(t *testing.T) {
	a, b := NewDigitalPet("Mochi", "o"), NewDigitalPet("Pip", "o")
	a.Souvenirs = []string{"pearl", "fossil", "crystal"}
	if _, err := GiveSouvenir(a, b, "pearl"); !errors.Is(err, ErrNotFriends) {
		t.Errorf("A gift to a stranger: %v", err)
	}
	b.Relationships.AddRelationship(a.ID, types.RelationshipFriend)
	rel, _ := b.Relationships.GetRelationship(a.ID)
	before := rel.BondStrength
	if _, err := GiveSouvenir(a, b, "shell"); !errors.Is(err, ErrNoSouvenir) {
		t.Errorf("Giving what it doesn't have: %v", err)
	}
	earned := map[string]bool{}
	for _, ach := range a.CheckAchievements() {
		earned[ach.ID] = true
	}
	if !earned["rare_find"] {
		t.Error("Three rare souvenirs earn Rare Find")
	}
	if _, err := GiveSouvenir(a, b, "pearl"); err != nil {
		t.Fatal(err)
	}
	if len(a.Souvenirs) != 2 || len(b.Souvenirs) != 1 || rel.BondStrength <= before {
		t.Errorf("After a gift: %v %v bond %.2f", a.Souvenirs, b.Souvenirs, rel.BondStrength)
	}
}

func TestShelfSetCheersPetsUp(t *testing.T) {
	pet := NewDigitalPet("Mochi", "o")
	before := pet.effectLocked(effHappinessDrop)
	SetShelfSet(true)
	t.Cleanup(func() { SetShelfSet(false) })
	if after := pet.effectLocked(effHappinessDrop); after >= before {
		t.Errorf("A full set on the shelf: %.3f vs %.3f", after, before)
	}
}
