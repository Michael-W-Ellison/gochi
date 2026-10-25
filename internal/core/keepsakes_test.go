package core

import (
	"strings"
	"testing"
	"time"

	"github.com/Michael-W-Ellison/gochi/internal/biology"
	"github.com/Michael-W-Ellison/gochi/internal/environment"
	"github.com/Michael-W-Ellison/gochi/pkg/types"
)

func kinds(p *DigitalPet) map[string]int {
	m := map[string]int{}
	for _, k := range p.BookPages() {
		m[k.Kind]++
	}
	return m
}

func TestKeepsakePages(t *testing.T) {
	useClock(t, time.Date(2026, 3, 10, 9, 0, 0, 0, time.UTC))
	pet := NewPetOfBreed("Mochi", "owner", SpeciesCat, "siamese")
	pages := pet.BookPages()
	if len(pages) != 1 || pages[0].Kind != KeepsakeHome || pages[0].Look == nil ||
		KeepsakeText(pages[0], "Mochi") != "Mochi came home." {
		t.Fatalf("A new pet's first page: %+v", pages)
	}

	// The first trick, once
	pet.mu.Lock()
	pet.markTrickLearnedLocked("sit")
	pet.markTrickLearnedLocked("come")
	pet.mu.Unlock()
	// Growing up, during play
	pet.Biology.Processes.Age = biology.AdultAge - 0.01
	setCalmHealthyState(pet)
	pet.Update(0.05)
	// A best friend
	friend := NewDigitalPet("Pip", "owner")
	pet.mu.Lock()
	pet.Relationships.AddRelationship(friend.ID, types.RelationshipFriend).BondStrength = 0.9
	pet.rememberPetLocked(friend.ID, "Pip")
	pet.keepMomentsLocked()
	pet.mu.Unlock()

	got := kinds(pet)
	if got[KeepsakeTrick] != 1 || got[KeepsakeGrewUp] != 1 || got[KeepsakeBestFriend] != 1 {
		t.Errorf("Pages: %v", got)
	}
	for _, k := range pet.BookPages() {
		if text := KeepsakeText(k, "Mochi"); text == "" || !strings.Contains(text, "Mochi") {
			t.Errorf("A page's text: %q for %+v", text, k)
		}
	}

	// A holiday played together, once a year
	useClock(t, time.Date(2026, 12, 25, 9, 0, 0, 0, time.UTC))
	pet.mu.Lock()
	pet.keepMomentsLocked()
	pet.keepMomentsLocked()
	pet.mu.Unlock()
	if kinds(pet)[KeepsakeHoliday] != 1 {
		t.Errorf("Christmas together: %v", kinds(pet))
	}

	// A memorial for a pet that has died
	if pet.MemorialOf() != nil {
		t.Error("A living pet has no memorial")
	}
	pet.Biology.IsAlive = false
	if m := pet.MemorialOf(); m == nil || m.Tricks != 2 || m.BestFriend != "Pip" {
		t.Errorf("Memorial: %+v", m)
	}
}

func TestBabiesInTheBook(t *testing.T) {
	a, b := adultPet("Mum"), adultPet("Dad")
	child := breedUntilSuccess(t, a, b, "Kit")
	if pages := child.BookPages(); len(pages) == 0 || pages[0].Detail != "born" || KeepsakeText(pages[0], "Kit") != "Kit was born." {
		t.Errorf("A baby's first page: %+v", pages)
	}
	if kinds(a)[KeepsakeBaby] != 1 {
		t.Errorf("A parent's first baby: %v", kinds(a))
	}
}

func TestThemesOfTheWeek(t *testing.T) {
	seen := map[string]bool{}
	start := time.Date(2026, 1, 5, 12, 0, 0, 0, time.UTC) // Week 2
	for w := 0; w < 52; w++ {
		day := start.AddDate(0, 0, 7*w)
		theme := themeFor(day)
		seen[theme.ID] = true
		_, n := day.ISOWeek()
		plain := themes[n%len(themes)]
		season := environment.SeasonOn(day, environment.Northern)
		if plain.ID == "hydration" && season == environment.SeasonWinter && theme.ID != "paws" {
			t.Errorf("Week %d: hydration in winter should swap with paw care", n)
		}
		if plain.ID == "paws" && season == environment.SeasonSummer && theme.ID != "hydration" {
			t.Errorf("Week %d: paw care in summer should swap with hydration", n)
		}
		if plain.ID != "hydration" && plain.ID != "paws" && theme.ID != plain.ID {
			t.Errorf("Week %d: %s became %s", n, plain.ID, theme.ID)
		}
	}
	if len(seen) < 11 {
		t.Errorf("Themes in a year: %v", seen)
	}

	// Every theme's topic is in the care guide for both species
	for _, species := range []Species{SpeciesCat, SpeciesDog} {
		topics := map[string]bool{}
		for _, g := range CareGuide(species) {
			topics[g.ID] = true
		}
		for _, th := range themes {
			if !topics[th.Topic] {
				t.Errorf("%s: no %s guide topic %s", th.ID, species, th.Topic)
			}
			for _, id := range th.happenings {
				if _, ok := findHappening(id); !ok {
					t.Errorf("%s: no happening %s", th.ID, id)
				}
			}
		}
	}
}

func TestThemeShapesTheWeek(t *testing.T) {
	// Find a week whose theme is dental week
	day := time.Date(2026, 1, 5, 12, 0, 0, 0, time.UTC)
	for themeFor(day).ID != "dental" {
		day = day.AddDate(0, 0, 7)
	}
	useClock(t, day)
	pet := NewDigitalPet("Mochi", "owner")
	if w := pet.WeekGoal(); w.Goal.Kind != WeeklyTeethDays {
		t.Errorf("Dental week's goal: %+v", w.Goal)
	}

	// Calm & comfort week brings a thunderstorm first
	for themeFor(day).ID != "calm" {
		day = day.AddDate(0, 0, 7)
	}
	useClock(t, day)
	useRandom(t, 0)
	pet = NewDigitalPet("Rex", "owner")
	setCalmHealthyState(pet)
	if h := checkHappening(pet); h == nil || h.ID != "thunderstorm" {
		t.Errorf("Calm week's happening: %+v", h)
	}
}
