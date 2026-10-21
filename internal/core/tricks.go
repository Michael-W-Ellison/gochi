package core

import (
	"fmt"
	"slices"

	"github.com/Michael-W-Ellison/gochi/internal/biology"
)

// Tricks. Cats and dogs learn different tricks, and harder ones build on
// easier ones: "touch" (nose to hand) is the start of many, and "come"
// (recall) matters most for safety. Cats can learn to roll over and play
// dead only when taught young, and only if they are playful and relaxed;
// these aren't part of shows.

// Trick is something a pet can be trained to do
type Trick struct {
	ID         string    `json:"id"`
	Name       string    `json:"name"`
	Icon       string    `json:"icon"`
	Difficulty int       `json:"difficulty"`         // 1 (easiest) to 5
	Requires   string    `json:"requires,omitempty"` // A trick to learn first
	Special    bool      `json:"special,omitempty"`  // For cats: only when young and suited, and not in shows
	Lesson     string    `json:"lesson,omitempty"`
	Performed  string    `json:"-"` // What the pet does; %s is its name
	species    []Species // Who learns it; nil for both
	noJoints   bool      // Not for seniors or breeds with joint problems
}

// tricks lists every trick, easiest first
var tricks = []Trick{
	{ID: "sit", Name: "Sit", Icon: "🐾", Difficulty: 1, Performed: "%s sits down neatly."},
	{ID: "come", Name: "Come when called", Icon: "📣", Difficulty: 1, Performed: "%s comes running to you!",
		Lesson: "Recall is the most useful thing a pet can learn: it keeps it safe outdoors. Always reward coming back."},
	{ID: "touch", Name: "Touch", Icon: "👆", Difficulty: 2, Performed: "%s boops your hand with its nose.",
		Lesson: "Target training (touching a hand or stick) is the start of many tricks, and a calm way to move a pet without pulling."},
	{ID: "paw", Name: "Shake a paw", Icon: "🤝", Difficulty: 2, Requires: "sit", Performed: "%s offers you a paw to shake."},
	{ID: "down", Name: "Down", Icon: "⬇️", Difficulty: 2, Requires: "sit", species: dogs, Performed: "%s lies down."},
	{ID: "stay", Name: "Stay", Icon: "✋", Difficulty: 3, Requires: "down", species: dogs, Performed: "%s stays put until you say so.",
		Lesson: "Build a stay slowly: a second, then two, then a step away. Always go back to reward it."},
	{ID: "high_five", Name: "High five", Icon: "🙌", Difficulty: 3, Requires: "paw", Performed: "%s gives you a high five!"},
	{ID: "spin", Name: "Spin", Icon: "🌀", Difficulty: 3, Requires: "touch", Performed: "%s spins around in a happy circle!"},
	{ID: "bow", Name: "Bow", Icon: "🙇", Difficulty: 3, Requires: "down", species: dogs, Performed: "%s takes a bow."},
	{ID: "hoop", Name: "Jump through a hoop", Icon: "⭕", Difficulty: 4, Requires: "touch", Performed: "%s leaps through the hoop!",
		Lesson: "Keep jumps low for young pets, seniors and breeds with joint problems."},
	{ID: "bell", Name: "Ring a bell", Icon: "🔔", Difficulty: 4, Requires: "touch", species: cats,
		Performed: "%s rings the bell with a paw.", Lesson: "A bell gives a pet a polite way to ask: hang it by the door or bowl."},
	{ID: "fetch", Name: "Fetch", Icon: "🎾", Difficulty: 4, Requires: "come", Performed: "%s brings the toy back!",
		Lesson: "Many cats fetch too. Use soft toys, never sticks, which splinter."},
	{ID: "sit_pretty", Name: "Sit pretty", Icon: "🐿️", Difficulty: 4, Requires: "sit", species: cats, noJoints: true,
		Performed: "%s sits up on its back legs."},
	{ID: "weave", Name: "Weave through legs", Icon: "〰️", Difficulty: 4, Requires: "spin", species: dogs,
		Performed: "%s weaves in and out between your legs."},
	{ID: "roll_over", Name: "Roll over", Icon: "🔄", Difficulty: 4, Requires: "down", Performed: "%s rolls all the way over!"},
	{ID: "play_dead", Name: "Play dead", Icon: "🎭", Difficulty: 5, Requires: "roll_over",
		Performed: "%s flops over dramatically... then peeks at you!"},
	{ID: "wave", Name: "Wave", Icon: "👋", Difficulty: 5, Requires: "high_five", species: cats, Performed: "%s waves a paw at you."},
}

// catSpecialTricks are the tricks a cat can learn only when taught young,
// and only if it is playful and relaxed
var catSpecialTricks = []string{"roll_over", "play_dead"}

// catSpecialAge is the age (game days) by which a cat must start learning
// them
const catSpecialAge = 600

// findTrick returns the trick with the given ID
func findTrick(id string) (Trick, bool) {
	for _, t := range tricks {
		if t.ID == id {
			return t, true
		}
	}
	return Trick{}, false
}

// forSpeciesTrick returns a trick as a species learns it: a cat rolls
// over from a sit (cats aren't taught "down"), and rolling over and
// playing dead are special for cats
func forSpeciesTrick(t Trick, species Species) (Trick, bool) {
	if t.species != nil && !slices.Contains(t.species, species) {
		return Trick{}, false
	}
	if species == SpeciesCat && slices.Contains(catSpecialTricks, t.ID) {
		t.Special = true
		if t.ID == "roll_over" {
			t.Requires = "sit"
		}
	}
	return t, true
}

// tricksLocked returns the tricks the pet's species learns. Caller must
// hold p.mu.
func (p *DigitalPet) tricksLocked() []Trick {
	var list []Trick
	for _, t := range tricks {
		if t, ok := forSpeciesTrick(t, p.speciesLocked()); ok {
			list = append(list, t)
		}
	}
	return list
}

// regularTricksLocked returns the tricks counted for "every trick" and
// shows: all but the special ones. Caller must hold p.mu.
func (p *DigitalPet) regularTricksLocked() []Trick {
	var list []Trick
	for _, t := range p.tricksLocked() {
		if !t.Special {
			list = append(list, t)
		}
	}
	return list
}

// trickLockedReasonLocked says why the pet can't start learning a trick,
// or returns "". Caller must hold p.mu.
func (p *DigitalPet) trickLockedReasonLocked(t Trick) string {
	t, ok := forSpeciesTrick(t, p.speciesLocked())
	if !ok {
		return fmt.Sprintf("%s is a trick for %s.", t.Name, p.defLocked().Plural)
	}
	if t.Requires != "" && !p.TricksLearned[t.Requires] {
		req, _ := findTrick(t.Requires)
		return fmt.Sprintf("%s needs to learn %s first.", p.Name, req.Name)
	}
	if t.noJoints && (p.Biology.LifeStage() == biology.StageSenior || p.Tendencies[TendJoints] > 0) {
		return fmt.Sprintf("%s isn't good for %s's joints.", t.Name, p.Name)
	}
	if t.Special {
		traits := p.Personality.Traits
		switch {
		case p.Biology.GetAgeInDays() > catSpecialAge:
			return fmt.Sprintf("%s can be taught only to young cats; %s started too late.", t.Name, p.Name)
		case traits.Playfulness < 0.6 || traits.Neuroticism > 0.5:
			return fmt.Sprintf("%s isn't the kind of cat to %s: it suits playful, relaxed cats.", p.Name, t.Name)
		}
	}
	return ""
}

// Tricks returns the tricks the pet's species learns, with the pet's
// progress
func (p *DigitalPet) Tricks() []TrickProgress {
	p.mu.Lock()
	defer p.mu.Unlock()

	var progress []TrickProgress
	for _, t := range p.tricksLocked() {
		skill := p.TrickSkills[t.ID]
		tp := TrickProgress{Trick: t, Skill: skill, Learned: skill >= 1.0, Rusty: skill < 1.0 && p.TricksLearned[t.ID]}
		if skill == 0 {
			tp.Locked = p.trickLockedReasonLocked(t)
		}
		progress = append(progress, tp)
	}
	return progress
}
