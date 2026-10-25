package core

import (
	"strings"
	"testing"
)

// While a pet is away, the cards tell the player about it and how to care
// for it: facts and tips in turn, starting with where it went and what it
// has found there, each with a title and text
func TestTripCards(t *testing.T) {
	for _, species := range []Species{SpeciesCat, SpeciesDog} {
		pet := NewPetOfBreed("Mochi", "owner", species, "")
		pet.Souvenirs = []string{"pebble", "acorn"}
		pet.TricksLearned = map[string]bool{"sit": true}
		pet.TrickSkills = map[string]float64{"sit": 1}
		cards := pet.TripCards()
		if len(cards) < 10 {
			t.Fatalf("%s: only %d cards", species, len(cards))
		}
		first := cards[0]
		if first.Kind != "fact" || !strings.Contains(first.Title, "park") ||
			!strings.Contains(first.Text, "found 2 of the 5 souvenirs") || !strings.Contains(first.Text, "four-leaf clover") {
			t.Errorf("%s: the first card should be where the pet is and what it found: %+v", species, first)
		}
		facts, tips := 0, 0
		guide := map[string]bool{}
		for i, c := range cards {
			if c.Icon == "" || c.Title == "" || c.Text == "" || strings.Contains(c.Text, "{pet}") {
				t.Errorf("%s: card %d is incomplete: %+v", species, i, c)
			}
			switch c.Kind {
			case "fact":
				facts++
			case "tip":
				tips++
				guide[c.Title] = true
			default:
				t.Errorf("%s: card %d has kind %q", species, i, c.Kind)
			}
			if i > 0 && i < 2*min(facts, tips) && c.Kind == cards[i-1].Kind {
				t.Errorf("%s: facts and tips should take turns: cards %d and %d are both %ss", species, i-1, i, c.Kind)
			}
		}
		if facts < 5 {
			t.Errorf("%s: only %d facts about the pet", species, facts)
		}
		for _, g := range CareGuide(species) {
			if !guide[g.Title] {
				t.Errorf("%s: the care guide's %q should be a tip", species, g.Title)
			}
		}
		text := ""
		for _, c := range cards {
			text += c.Text + "\n"
		}
		for _, want := range []string{"Mochi knows Sit", "game days old", "Mochi is at the park"} {
			if !strings.Contains(text, want) {
				t.Errorf("%s: the cards should say %q", species, want)
			}
		}
	}
}
