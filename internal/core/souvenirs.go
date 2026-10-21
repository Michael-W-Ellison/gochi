package core

import (
	"errors"
	"fmt"
	"slices"
)

// Souvenirs. Each place has a set of four souvenirs and a rare one. What
// a pet brings home comes from the place it went; curious pets find the
// rare one more often. A few finds carry a lesson (wild mushrooms and
// berries are never to be eaten). The game server keeps the household's
// collection, so a set is complete once the household's pets together
// have found all of it. Souvenirs can be put on the house's shelf, or
// given to another pet as a gift.

// Souvenir is something a pet brings home from a trip
type Souvenir struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Icon   string `json:"icon"`
	Place  string `json:"place"`
	Rare   bool   `json:"rare"`
	Lesson string `json:"lesson,omitempty"`
}

// souvenirs are every souvenir, in sets by place
var souvenirs = []Souvenir{
	{ID: "pebble", Name: "a shiny pebble", Icon: "🪨", Place: "park"},
	{ID: "acorn", Name: "an acorn", Icon: "🌰", Place: "park"},
	{ID: "pinecone", Name: "a pinecone", Icon: "🌲", Place: "park"},
	{ID: "dandelion", Name: "a dandelion", Icon: "🌼", Place: "park"},
	{ID: "four_leaf_clover", Name: "a four-leaf clover", Icon: "🍀", Place: "park", Rare: true},

	{ID: "shell", Name: "a seashell", Icon: "🐚", Place: "beach"},
	{ID: "sea_glass", Name: "a piece of sea glass", Icon: "🔷", Place: "beach"},
	{ID: "driftwood", Name: "a piece of driftwood", Icon: "🪵", Place: "beach"},
	{ID: "sand_dollar", Name: "a sand dollar shell", Icon: "⚪", Place: "beach"},
	{ID: "pearl", Name: "a pearl", Icon: "🦪", Place: "beach", Rare: true},

	{ID: "feather", Name: "a feather", Icon: "🪶", Place: "farm"},
	{ID: "corn_cob", Name: "a corn cob", Icon: "🌽", Place: "farm"},
	{ID: "horseshoe", Name: "an old horseshoe", Icon: "🧲", Place: "farm"},
	{ID: "hay_twist", Name: "a twist of hay", Icon: "🌾", Place: "farm"},
	{ID: "golden_egg", Name: "a painted golden egg", Icon: "🥚", Place: "farm", Rare: true},

	{ID: "leaf", Name: "a colorful leaf", Icon: "🍁", Place: "forest"},
	{ID: "mushroom", Name: "a mushroom (just to look at)", Icon: "🍄", Place: "forest",
		Lesson: "Never let a pet eat wild mushrooms: some are deadly, and they are hard to tell apart."},
	{ID: "moss", Name: "a clump of moss", Icon: "🌿", Place: "forest"},
	{ID: "jay_feather", Name: "a blue jay feather", Icon: "🪶", Place: "forest"},
	{ID: "crystal", Name: "a sparkly crystal", Icon: "💎", Place: "forest", Rare: true},

	{ID: "bottle_cap", Name: "a bottle cap", Icon: "🔘", Place: "city"},
	{ID: "postcard", Name: "a postcard", Icon: "🖼️", Place: "city"},
	{ID: "button", Name: "a lost button", Icon: "🔵", Place: "city"},
	{ID: "toy_car", Name: "a toy car", Icon: "🚗", Place: "city"},
	{ID: "old_coin", Name: "an old coin", Icon: "🪙", Place: "city", Rare: true},

	{ID: "quartz", Name: "a piece of quartz", Icon: "🔹", Place: "mountain"},
	{ID: "edelweiss", Name: "an edelweiss flower", Icon: "🤍", Place: "mountain"},
	{ID: "eagle_feather", Name: "an eagle feather", Icon: "🪶", Place: "mountain"},
	{ID: "smooth_stone", Name: "a smooth stone", Icon: "🪨", Place: "mountain"},
	{ID: "fossil", Name: "a fossil", Icon: "🐚", Place: "mountain", Rare: true},

	{ID: "flower", Name: "a flower", Icon: "🌸", Place: "meadow"},
	{ID: "seed_head", Name: "a dandelion clock", Icon: "🌬️", Place: "meadow"},
	{ID: "wheat_stalk", Name: "a stalk of wheat", Icon: "🌾", Place: "meadow"},
	{ID: "blue_feather", Name: "a little blue feather", Icon: "🪶", Place: "meadow"},
	{ID: "amber", Name: "a drop of amber", Icon: "🟠", Place: "meadow", Rare: true},

	{ID: "river_stone", Name: "a river stone", Icon: "🪨", Place: "river"},
	{ID: "reed", Name: "a reed", Icon: "🎋", Place: "river"},
	{ID: "willow_leaf", Name: "a willow leaf", Icon: "🍃", Place: "river"},
	{ID: "twig_boat", Name: "a twig shaped like a boat", Icon: "🛶", Place: "river"},
	{ID: "gold_flake", Name: "a tiny flake of gold", Icon: "✨", Place: "river", Rare: true},

	{ID: "tennis_ball", Name: "a tennis ball", Icon: "🎾", Place: "dog_park"},
	{ID: "frisbee", Name: "a flying disc", Icon: "🥏", Place: "dog_park"},
	{ID: "rope_knot", Name: "a rope knot", Icon: "🪢", Place: "dog_park"},
	{ID: "bandana", Name: "a lost bandana", Icon: "🧣", Place: "dog_park"},
	{ID: "golden_ball", Name: "a golden ball", Icon: "🟡", Place: "dog_park", Rare: true},

	{ID: "rose_petal", Name: "a rose petal", Icon: "🌹", Place: "garden"},
	{ID: "herb_sprig", Name: "a sprig of mint", Icon: "🌱", Place: "garden"},
	{ID: "snail_shell", Name: "an empty snail shell", Icon: "🐌", Place: "garden"},
	{ID: "seed_packet", Name: "a seed packet", Icon: "🌻", Place: "garden"},
	{ID: "marble", Name: "a glass marble", Icon: "🔮", Place: "garden", Rare: true},

	{ID: "snowy_pinecone", Name: "a snowy pinecone", Icon: "❄️", Place: "snowy_hill"},
	{ID: "red_berries", Name: "a sprig of red berries (not for eating)", Icon: "🍒", Place: "snowy_hill",
		Lesson: "Many winter berries, such as holly and mistletoe, are poisonous to pets."},
	{ID: "mitten", Name: "a lost mitten", Icon: "🧤", Place: "snowy_hill"},
	{ID: "birch_bark", Name: "a curl of birch bark", Icon: "📜", Place: "snowy_hill"},
	{ID: "snowflake", Name: "a snowflake ornament", Icon: "❄️", Place: "snowy_hill", Rare: true},

	{ID: "lake_pebble", Name: "a lake pebble", Icon: "🪨", Place: "lakeside"},
	{ID: "cattail", Name: "a cattail", Icon: "🌾", Place: "lakeside"},
	{ID: "fishing_float", Name: "a fishing float", Icon: "🎈", Place: "lakeside"},
	{ID: "lily", Name: "a water lily", Icon: "🪷", Place: "lakeside"},
	{ID: "map_bottle", Name: "a bottle with a treasure map", Icon: "🗺️", Place: "lakeside", Rare: true},
}

// Errors giving souvenirs
var (
	ErrNoSouvenir = errors.New("doesn't have that souvenir")
	ErrNotFriends = errors.New("they need to have met first")
)

// findSouvenir returns a souvenir
func findSouvenir(id string) (Souvenir, bool) {
	for _, s := range souvenirs {
		if s.ID == id {
			return s, true
		}
	}
	return Souvenir{}, false
}

// FindSouvenir returns a souvenir
func FindSouvenir(id string) (Souvenir, bool) { return findSouvenir(id) }

// SouvenirSets returns each place's souvenirs, in place order
func SouvenirSets() map[string][]Souvenir {
	sets := map[string][]Souvenir{}
	for _, s := range souvenirs {
		sets[s.Place] = append(sets[s.Place], s)
	}
	return sets
}

// pickSouvenirLocked picks what the pet finds at a place: one of its four
// common souvenirs, or its rare one (likelier for curious pets). Caller
// must hold p.mu.
func (p *DigitalPet) pickSouvenirLocked(place string) Souvenir {
	set := SouvenirSets()[place]
	if len(set) == 0 {
		set = SouvenirSets()[defaultPlace]
	}
	weights := make([]int, len(set))
	total := 0
	for i, s := range set {
		weights[i] = 3
		if s.Rare {
			weights[i] = 1 + int(2*p.Personality.Traits.Curiosity)
		}
		total += weights[i]
	}
	pick := randIntn(total)
	for i, w := range weights {
		if pick < w {
			return set[i]
		}
		pick -= w
	}
	return set[0]
}

// GiveSouvenir has one pet give another a souvenir it found. They must
// have met; the gift strengthens the receiver's bond with the giver.
func GiveSouvenir(from, to *DigitalPet, id string) (string, error) {
	if from == to {
		return "", ErrSamePet
	}
	s, ok := findSouvenir(id)
	if !ok {
		return "", ErrNoSouvenir
	}
	unlock := lockPair(from, to)
	defer unlock()
	i := slices.Index(from.Souvenirs, id)
	switch {
	case i < 0:
		return "", ErrNoSouvenir
	case !from.Biology.IsAlive || !to.Biology.IsAlive:
		return "", ErrPetNotAvailable
	}
	rel, ok := to.Relationships.GetRelationship(from.ID)
	if !ok {
		return "", ErrNotFriends
	}
	from.Souvenirs = slices.Delete(from.Souvenirs, i, i+1)
	to.Souvenirs = append(to.Souvenirs, id)
	rel.BondStrength += 0.05
	rel.AddSharedExperience("was given "+s.Name, to.Biology.GetAgeInDays(), 0.05)
	rel.Clamp()
	to.Biology.Vitals.Happiness = min(to.Biology.Vitals.Happiness+0.05, 1)
	return fmt.Sprintf("%s gave %s %s. %s", from.Name, to.Name, s.Name, s.Icon), nil
}

// rareSouvenirsLocked counts the rare souvenirs the pet has. Caller must
// hold p.mu.
func (p *DigitalPet) rareSouvenirsLocked() int {
	n := 0
	for _, id := range p.Souvenirs {
		if s, ok := findSouvenir(id); ok && s.Rare {
			n++
		}
	}
	return n
}

// souvenirAchievements are achievements for collecting. Conditions run with
// p.mu held.
var souvenirAchievements = []struct {
	Achievement
	earned func(p *DigitalPet) bool
}{
	{Achievement{ID: "rare_find", Name: "Rare Find", Icon: "💎", Rarity: "rare", Points: 25,
		Description: "Found 3 rare souvenirs"},
		func(p *DigitalPet) bool { return p.rareSouvenirsLocked() >= 3 }},
	{Achievement{ID: "set_collector", Name: "Set Collector", Icon: "🧺", Rarity: "rare", Points: 25,
		Description: "Your household completed a place's set of souvenirs"},
		func(p *DigitalPet) bool { return p.awarded("set_collector") }},
	{Achievement{ID: "curator", Name: "Curator", Icon: "🏛️", Rarity: "legendary", Points: 100,
		Description: "Your household completed every set of souvenirs"},
		func(p *DigitalPet) bool { return p.awarded("curator") }},
}

func init() {
	achievements = append(achievements, souvenirAchievements...)
}
