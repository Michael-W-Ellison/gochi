package core

import (
	"maps"
	"slices"
)

// Breeding and breeds. Two pets of the same breed have a baby of that
// breed. Any other pairing has a mixed breed (a domestic shorthair or
// longhair cat, or a mixed breed dog): each body feature comes from one
// parent or the other, and it inherits a breed tendency only if both
// parents have it, at half strength. A simplification of the finding that
// crossbreeds have fewer recessive inherited disorders, not a claim that
// mixed breeds are always healthier.
//
// Coat patterns are inherited from a parent. Two merle parents can have a
// "double merle" puppy, mostly white and often deaf with eye problems,
// which is why merle is never bred to merle.

// Lineage is where a bred pet comes from
type Lineage struct {
	Parents []LineageParent   `json:"parents"`
	From    map[string]string `json:"from,omitempty"` // For a mixed breed: each feature, and the parent it came from
	Notes   []string          `json:"notes,omitempty"`
}

// LineageParent is one parent: its name and breed, and its own parents'
// breeds if it was a mixed breed
type LineageParent struct {
	Name         string   `json:"name"`
	Breed        string   `json:"breed"`      // ID
	BreedName    string   `json:"breed_name"` // Name
	ParentBreeds []string `json:"parent_breeds,omitempty"`
}

// lineageParent describes a parent for its baby's lineage
func lineageParent(info breedingInfo) LineageParent {
	b, _ := FindBreed(info.breed)
	lp := LineageParent{Name: info.name, Breed: info.breed, BreedName: b.Name}
	if b.Mixed && info.lineage != nil {
		for _, gp := range info.lineage.Parents {
			lp.ParentBreeds = append(lp.ParentBreeds, gp.BreedName)
		}
	}
	return lp
}

// coinFlip picks one of two parents
func coinFlip() bool { return randIntn(2) == 0 }

// inheritBreed gives a baby its breed, looks and tendencies from its
// parents. Caller owns the child alone.
func inheritBreed(child *DigitalPet, a, b breedingInfo) {
	lineage := &Lineage{Parents: []LineageParent{lineageParent(a), lineageParent(b)}}
	breedA, _ := FindBreed(a.breed)
	pattern := inheritPattern(a.look.Pattern, b.look.Pattern)

	if a.breed == b.breed && !breedA.Mixed {
		child.setBreedLocked(breedA, child.Genome)
		if pattern != "" && (pattern == PatternDoubleMerl || slices.Contains(breedA.Patterns, pattern)) {
			child.Look.Pattern = pattern
		}
	} else {
		child.mixBreedLocked(a, b, lineage)
		if pattern != "" {
			child.Look.Pattern = pattern
		}
	}
	child.Look.normalize(child.Species)
	if child.Look.Pattern == PatternDoubleMerl {
		child.Tendencies[TendEyes] = max(child.Tendencies[TendEyes], 1)
		lineage.Notes = append(lineage.Notes, "Double merle: both parents are merle. Double merles are often deaf and have "+
			"eye problems, which is why merle is never bred to merle.")
	}
	child.Biology.Physiology.Hearing = congenitalHearing(*child.Look)
	child.Lineage = lineage
}

// inheritPattern returns the pattern a baby inherits from parents with
// these patterns: one parent's, except that two merles can have a double
// merle (one in four) or a non-merle (one in four, "" to keep the breed's)
func inheritPattern(a, b string) string {
	merle := func(p string) bool { return p == PatternMerle || p == PatternDoubleMerl }
	if merle(a) && merle(b) {
		switch randIntn(4) {
		case 0:
			return PatternDoubleMerl
		case 1:
			return ""
		}
		return PatternMerle
	}
	if coinFlip() {
		return a
	}
	return b
}

// mixBreedLocked makes the child a mixed breed of parents a and b. Caller
// owns the child alone.
func (child *DigitalPet) mixBreedLocked(a, b breedingInfo, lineage *Lineage) {
	from := map[string]string{}
	choose := func(feature, x, y string) string {
		if coinFlip() {
			from[feature] = a.name
			return x
		}
		from[feature] = b.name
		return y
	}
	la, lb := a.look, b.look
	look := PetLook{}
	look.Build = choose("build", la.Build, lb.Build)
	look.Ears = choose("ears", la.Ears, lb.Ears)
	look.Muzzle = choose("muzzle", la.Muzzle, lb.Muzzle)
	look.Coat = choose("coat", la.Coat, lb.Coat)
	look.Legs = choose("legs", la.Legs, lb.Legs)
	look.Tail = choose("tail", la.Tail, lb.Tail)
	look.Color = choose("color", la.Color, lb.Color)
	look.Color2 = choose("second color", la.Color2, lb.Color2)
	look.Pattern = choose("pattern", la.Pattern, lb.Pattern)
	look.Eye = choose("eyes", la.Eye, lb.Eye)
	look.Eye2 = look.Eye
	look.Size = (la.Size + lb.Size) / 2 * (0.95 + float64(randIntn(101))/1000)
	look.Body.Size = [2]float64{min(la.Size, lb.Size), max(la.Size, lb.Size)}

	id := BreedMixed
	if child.Species == SpeciesCat {
		id = BreedDomesticShorthair
		if look.Coat == CoatLong || look.Coat == CoatSemi || look.Coat == CoatShaggy {
			id = BreedDomesticLonghair
		}
	}
	child.Breed = id
	child.Look = &look
	child.fillHomeLookLocked()
	lineage.From = from

	// Tendencies both parents have pass on at half strength
	child.Tendencies = map[string]float64{}
	for t, sa := range a.tendencies {
		if sb := b.tendencies[t]; sa > 0 && sb > 0 {
			child.Tendencies[t] = (sa + sb) / 4
		}
	}
	child.BreedLifespan = (lifespanOr1(a.lifespan) + lifespanOr1(b.lifespan)) / 2
	child.Biology.EnsurePhysiology()
}

func lifespanOr1(l float64) float64 {
	if l <= 0 {
		return 1
	}
	return l
}

// cloneLineage copies a lineage
func cloneLineage(l *Lineage) *Lineage {
	if l == nil {
		return nil
	}
	c := *l
	c.Parents = append([]LineageParent(nil), l.Parents...)
	c.From = maps.Clone(l.From)
	c.Notes = append([]string(nil), l.Notes...)
	return &c
}
