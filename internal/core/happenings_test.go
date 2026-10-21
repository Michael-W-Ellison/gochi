package core

import (
	"errors"
	"slices"
	"testing"
	"time"

	"github.com/Michael-W-Ellison/gochi/internal/environment"
)

func TestHappeningCatalog(t *testing.T) {
	if len(happenings) != 44 {
		t.Errorf("%d happenings", len(happenings))
	}
	ids, rightFirst := map[string]bool{}, 0
	for _, h := range happenings {
		if ids[h.id] {
			t.Errorf("%s twice", h.id)
		}
		ids[h.id] = true
		if h.lesson == "" || h.title == "" || h.text == "" || len(h.choices) < 2 || len(h.choices) > 3 {
			t.Errorf("%s is incomplete", h.id)
		}
		right, choices := 0, map[string]bool{}
		for _, c := range h.choices {
			if c.right {
				right++
			}
			if choices[c.id] || c.text == "" {
				t.Errorf("%s: choice %q", h.id, c.id)
			}
			choices[c.id] = true
		}
		if right != 1 {
			t.Errorf("%s has %d right choices", h.id, right)
		}
		if view := h.view("Mochi"); h.choices[slices.IndexFunc(h.choices, func(c happeningChoice) bool {
			return c.id == view.Choices[0].ID
		})].right {
			rightFirst++
		}
		for _, id := range h.holidays {
			if _, ok := FindHoliday(id); !ok {
				t.Errorf("%s: no holiday %s", h.id, id)
			}
		}
	}
	if rightFirst > len(happenings)*2/3 {
		t.Errorf("The right choice is listed first in %d of %d", rightFirst, len(happenings))
	}
}

// pendingHappening gives the pet a happening to choose in
func pendingHappening(pet *DigitalPet, id string) {
	pet.mu.Lock()
	pet.Happening = &PendingHappening{ID: id, Since: pet.playNowLocked()}
	pet.mu.Unlock()
}

func TestHappeningsComeNowAndThen(t *testing.T) {
	clock := useClock(t, time.Date(2026, 3, 10, 9, 0, 0, 0, time.UTC))
	useRandom(t, 0) // Always lucky
	pet := NewDigitalPet("Mochi", "owner")
	setCalmHealthyState(pet)
	first := pet.CheckHappening()
	if first == nil || len(first.Choices) < 2 {
		t.Fatalf("A happening: %+v", first)
	}
	if pet.CurrentHappening().ID != first.ID {
		t.Error("The happening waits for a choice")
	}
	pet.DismissHappening()
	*clock = clock.Add(10 * time.Minute)
	if pet.CheckHappening() != nil {
		t.Error("Another happening within 20 minutes")
	}
	*clock = clock.Add(11 * time.Minute)
	second := pet.CheckHappening()
	if second == nil || second.ID == first.ID {
		t.Fatalf("The next happening: %+v", second)
	}

	// Unanswered, it goes away after 10 minutes
	*clock = clock.Add(11 * time.Minute)
	pet.CheckHappening()
	if pet.CurrentHappening() != nil {
		t.Error("An unanswered happening passes")
	}

	// Not while the pet is unwell
	pet = NewDigitalPet("Rex", "owner")
	setCalmHealthyState(pet)
	pet.Injured = true
	if pet.CheckHappening() != nil {
		t.Error("A happening for an injured pet")
	}
}

func TestHappeningsFitThePet(t *testing.T) {
	useClock(t, time.Date(2026, 3, 10, 9, 0, 0, 0, time.UTC))
	cat, dog := NewDigitalPet("Mochi", "owner"), NewDigitalPet("Rex", "owner")
	dog.Species = SpeciesDog
	fits := func(p *DigitalPet, id string) bool {
		h, _ := findHappening(id)
		p.mu.Lock()
		defer p.mu.Unlock()
		return p.happeningFitsLocked(h)
	}
	if !fits(cat, "kneading") || fits(dog, "kneading") || fits(cat, "shoe") {
		t.Error("Species happenings")
	}
	cat.LivingAreaWaste = 0
	if fits(cat, "smell") {
		t.Error("A smell from a clean home")
	}
	cat.LivingAreaWaste = 0.5
	if !fits(cat, "smell") {
		t.Error("A smell from a dirty home")
	}
	dog.mu.Lock()
	dog.ensureEnvironmentLocked()
	dog.Environment.Seasons.SetSeason(environment.SeasonWinter, 1)
	dog.mu.Unlock()
	if !fits(dog, "salt") || fits(dog, "hot_pavement") {
		t.Error("Winter happenings")
	}
	if fits(dog, "festive_tree") { // March
		t.Error("A festive tree out of season")
	}

	// Once a week
	cat.Marks = map[string]string{"happening:kneading": today()}
	if fits(cat, "kneading") {
		t.Error("The same happening twice in a week")
	}
}

func TestHappeningChoices(t *testing.T) {
	useClock(t, time.Date(2026, 3, 10, 9, 0, 0, 0, time.UTC))
	pet := NewDigitalPet("Mochi", "owner")
	setCalmHealthyState(pet)
	if _, err := pet.ChooseHappening("comfort"); !errors.Is(err, ErrNoHappening) {
		t.Errorf("A choice with nothing happening: %v", err)
	}
	pendingHappening(pet, "thunderstorm")
	if _, err := pet.ChooseHappening("dance"); !errors.Is(err, ErrBadChoice) {
		t.Errorf("A choice that isn't offered: %v", err)
	}
	before := pet.FriendshipPoints
	result, err := pet.ChooseHappening("comfort")
	if err != nil || !result.Right || result.Points != happeningPoints || result.Lesson == "" ||
		pet.FriendshipPoints != before+happeningPoints {
		t.Errorf("Comforting a frightened pet: %+v %v", result, err)
	}

	// Points from happenings are limited each day
	total := result.Points
	for i := 0; i < 6; i++ {
		pendingHappening(pet, "thunderstorm")
		r, _ := pet.ChooseHappening("comfort")
		total += r.Points
	}
	if total != happeningPointsPerDay {
		t.Errorf("%d points in a day", total)
	}

	// A wrong choice: leaving chocolate out poisons the pet
	pendingHappening(pet, "chocolate")
	result, _ = pet.ChooseHappening("leave")
	if result.Right || result.Points != 0 || pet.Illness == nil || pet.Illness.Kind != IllnessPoisoning {
		t.Errorf("Chocolate left out: %+v %+v", result, pet.Illness)
	}
	earned := false
	pet.Counters["good_calls"] = 25
	for _, a := range pet.CheckAchievements() {
		earned = earned || a.ID == "happy_helper"
	}
	if !earned {
		t.Error("Happy Helper after 25 good calls")
	}
}
