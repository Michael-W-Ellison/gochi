package core

import (
	"errors"
	"strings"
	"testing"

	"github.com/Michael-W-Ellison/gochi/pkg/types"
)

// sociablePet returns a pet that takes to others easily
func sociablePet(name string, species Species) *DigitalPet {
	pet := NewDigitalPet(name, "u")
	pet.Species = species
	t := pet.Personality.Traits
	t.Extraversion, t.Agreeableness, t.Territoriality, t.Neuroticism = 0.9, 0.9, 0.1, 0.1
	setCalmHealthyState(pet)
	return pet
}

// grumpyPet returns a territorial, nervous pet
func grumpyPet(name string, species Species) *DigitalPet {
	pet := NewDigitalPet(name, "u")
	pet.Species = species
	t := pet.Personality.Traits
	t.Extraversion, t.Agreeableness, t.Territoriality, t.Neuroticism = 0.1, 0.2, 0.9, 0.8
	setCalmHealthyState(pet)
	return pet
}

func TestFriendlyMeeting(t *testing.T) {
	a, b := sociablePet("Rex", SpeciesDog), sociablePet("Bella", SpeciesDog)
	m, err := Introduce(a, b)
	if err != nil {
		t.Fatal(err)
	}
	if m.Quality < goodMeeting || !strings.Contains(m.Text, "played together happily") {
		t.Errorf("Two sociable dogs should get on: %+v", m)
	}
	if !strings.Contains(m.Lesson, "neutral ground") {
		t.Errorf("Dog introductions should teach how to introduce dogs: %q", m.Lesson)
	}
	for _, pet := range []*DigitalPet{a, b} {
		relations := pet.Relations()
		if len(relations) != 1 || relations[0].Kind != "friend" {
			t.Errorf("%s should now have a friend: %+v", pet.Name, relations)
		}
	}
	if m.PointsEarned == 0 {
		t.Error("An introduction is care and earns points")
	}
}

func TestBadMeetingMakesRivals(t *testing.T) {
	a, b := grumpyPet("Tom", SpeciesCat), grumpyPet("Felix", SpeciesCat)
	stress := b.Biology.Vitals.Stress
	m, err := Introduce(a, b)
	if err != nil {
		t.Fatal(err)
	}
	if m.Quality >= cautiousMeeting || !strings.Contains(m.Text, "hissed at") {
		t.Errorf("Two territorial cats should clash: %+v", m)
	}
	if !strings.Contains(m.Lesson, "swap bedding") || !strings.Contains(m.Lesson, "slow down") {
		t.Errorf("A bad cat meeting should teach gradual introductions: %q", m.Lesson)
	}
	if relations := b.Relations(); len(relations) != 1 || relations[0].Kind != "rival" {
		t.Errorf("A bad meeting should make rivals: %+v", relations)
	}
	if b.Biology.Vitals.Stress <= stress {
		t.Error("A bad meeting should stress the other pet")
	}
}

func TestMeetingsBuildOnEachOther(t *testing.T) {
	a, b := NewDigitalPet("Biscuit", "u"), NewDigitalPet("Mochi", "u")
	first, _ := Introduce(a, b)
	for i := 0; i < 10; i++ {
		setCalmHealthyState(a)
		setCalmHealthyState(b)
		a.RecentCare = nil // Meetings spread out over time
		if _, err := Introduce(a, b); err != nil {
			t.Fatal(err)
		}
	}
	setCalmHealthyState(a)
	setCalmHealthyState(b)
	a.RecentCare = nil
	later, _ := Introduce(a, b)
	if later.Quality <= first.Quality {
		t.Errorf("Good meetings should make later ones go better: %f then %f", first.Quality, later.Quality)
	}
}

func TestCatAndDogMeetingNeedsCare(t *testing.T) {
	cat, dog := NewDigitalPet("Biscuit", "u"), NewDigitalPet("Rex", "u")
	dog.Species = SpeciesDog
	cats1, cats2 := NewDigitalPet("A", "u"), NewDigitalPet("B", "u")
	dogs1, dogs2 := NewDigitalPet("C", "u"), NewDigitalPet("D", "u")
	dogs1.Species, dogs2.Species = SpeciesDog, SpeciesDog
	mixed, _ := Introduce(cat, dog)
	sameCats, _ := Introduce(cats1, cats2)
	sameDogs, _ := Introduce(dogs1, dogs2)
	if !(mixed.Quality < sameCats.Quality && sameCats.Quality < sameDogs.Quality) {
		t.Errorf("First meetings: cat and dog %f, two cats %f, two dogs %f", mixed.Quality, sameCats.Quality, sameDogs.Quality)
	}
	if !strings.Contains(mixed.Lesson, "leash") {
		t.Errorf("A cat and dog meeting should teach how to introduce them: %q", mixed.Lesson)
	}
}

func TestMeetingNeedsBothPetsAvailable(t *testing.T) {
	a, b := NewDigitalPet("Biscuit", "u"), NewDigitalPet("Mochi", "u")
	b.CurrentBehavior = types.BehaviorSleeping
	var refused RefusedError
	if _, err := Introduce(a, b); !errors.As(err, &refused) || !strings.Contains(refused.Reason, "Mochi is asleep") {
		t.Errorf("A sleeping pet can't be introduced: %v", err)
	}
	if _, err := Introduce(a, a); !errors.Is(err, ErrSamePet) {
		t.Errorf("A pet can't meet itself: %v", err)
	}
}

func TestRelationsSurviveSaveLoad(t *testing.T) {
	a, b := sociablePet("Rex", SpeciesDog), sociablePet("Bella", SpeciesDog)
	if _, err := Introduce(a, b); err != nil {
		t.Fatal(err)
	}
	data, err := a.Save()
	if err != nil {
		t.Fatal(err)
	}
	loaded, err := Load(data)
	if err != nil {
		t.Fatal(err)
	}
	if relations := loaded.Relations(); len(relations) != 1 || relations[0].Name != "Bella" {
		t.Errorf("Relations should be saved with names: %+v", relations)
	}
}

func TestUnsureFirstMeetingIsNotFriendship(t *testing.T) {
	cat, dog := NewDigitalPet("Biscuit", "u"), NewDigitalPet("Rex", "u")
	dog.Species = SpeciesDog
	m, _ := Introduce(dog, cat)
	if m.Quality >= cautiousMeeting {
		t.Fatalf("A first meeting of a typical cat and dog should be tense, got %f", m.Quality)
	}
	if relations := dog.Relations(); len(relations) != 1 || relations[0].Kind != "rival" ||
		relations[0].Description != "wary of each other" {
		t.Errorf("After a tense first meeting the pets should be wary of each other: %+v", relations)
	}

	// A great first meeting makes friends, but not best friends at once
	a, b := sociablePet("A", SpeciesDog), sociablePet("B", SpeciesDog)
	if _, err := Introduce(a, b); err != nil {
		t.Fatal(err)
	}
	if relations := a.Relations(); relations[0].Kind != "friend" || relations[0].Description == "best friends" {
		t.Errorf("One good meeting should make friends, not best friends: %+v", relations)
	}
}

func TestDogsDontPurr(t *testing.T) {
	dog := withAttentionStyle(NewDigitalPet("Rex", "u"), 0.9, 0.2, 0.7, 0.2)
	dog.Species = SpeciesDog
	setCalmHealthyState(dog)
	dog.Emotions.Loneliness = 0.6
	r := dog.Interact(types.InteractionPetting, 1.0)
	if r.Reaction != ReactionDelighted || strings.Contains(r.ReactionText, "purr") || !strings.Contains(r.ReactionText, "tail") {
		t.Errorf("A delighted dog should wag its tail, not purr: %+v", r)
	}
}
