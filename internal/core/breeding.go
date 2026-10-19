package core

import (
	"errors"
	"time"

	"github.com/Michael-W-Ellison/gochi/internal/ai"
	"github.com/Michael-W-Ellison/gochi/internal/genetics"
	"github.com/Michael-W-Ellison/gochi/pkg/types"
)

// Breeding uses the genetics package. Each pet has a genome whose
// behavioral genes match its personality. Two pets that meet the genetic
// system's requirements (age, health, happiness, cooldown, compatibility)
// can have a baby: its genome comes from crossover and mutation of its
// parents' genomes, its genetically based traits from that genome, and its
// other traits from ai.PersonalityMatrix.InheritTraits.

// Breeding errors
var (
	ErrSamePet = errors.New("a pet cannot breed with itself")
)

// BreedingError reports why two pets could not have a baby
type BreedingError struct {
	Reason string
}

func (e BreedingError) Error() string {
	return e.Reason
}

// childMutationRate is the chance each inherited personality trait mutates
const childMutationRate = 0.05

// genePersonality links behavioral genes to the personality traits they
// underlie
var genePersonality = []struct {
	gene genetics.TraitType
	get  func(*ai.Traits) float64
	set  func(*ai.Traits, float64)
}{
	{genetics.TraitBasePlayfulness, func(t *ai.Traits) float64 { return t.Playfulness }, func(t *ai.Traits, v float64) { t.Playfulness = v }},
	{genetics.TraitBaseIntelligence, func(t *ai.Traits) float64 { return t.Intelligence }, func(t *ai.Traits, v float64) { t.Intelligence = v }},
	{genetics.TraitBaseSociability, func(t *ai.Traits) float64 { return t.Extraversion }, func(t *ai.Traits, v float64) { t.Extraversion = v }},
	{genetics.TraitBaseAnxiety, func(t *ai.Traits) float64 { return t.Neuroticism }, func(t *ai.Traits, v float64) { t.Neuroticism = v }},
	{genetics.TraitBaseEnergy, func(t *ai.Traits) float64 { return t.EnergyLevel }, func(t *ai.Traits, v float64) { t.EnergyLevel = v }},
	{genetics.TraitBaseCuriosity, func(t *ai.Traits) float64 { return t.Curiosity }, func(t *ai.Traits, v float64) { t.Curiosity = v }},
	{genetics.TraitBaseAffection, func(t *ai.Traits) float64 { return t.Affectionate }, func(t *ai.Traits, v float64) { t.Affectionate = v }},
	{genetics.TraitBaseAggression, func(t *ai.Traits) float64 { return 1 - t.Agreeableness }, func(t *ai.Traits, v float64) { t.Agreeableness = 1 - v }},
}

// geneValue returns a gene's value. GetTraitValue also multiplies by the
// gene's expression (0.8 to 1.0); using that for personality would make
// every generation's traits drift lower, so the value is used directly.
func geneValue(g *genetics.Genome, gene genetics.TraitType) float64 {
	if allele, ok := g.Traits[gene]; ok {
		return allele.Value
	}
	return 0.5
}

// ensureGenomeLocked gives pets created before genomes existed one that
// matches their personality. Caller must hold p.mu.
func (p *DigitalPet) ensureGenomeLocked() {
	if p.Genome != nil {
		return
	}
	values := make(map[genetics.TraitType]float64, len(genePersonality)+1)
	for _, link := range genePersonality {
		values[link.gene] = link.get(p.Personality.Traits)
	}
	values[genetics.TraitImmuneStrength] = p.Biology.Processes.ImmuneStrength
	p.Genome = genetics.NewGenomeWithTraits(values)
}

// breedingInfo is what breeding needs to know about one parent
type breedingInfo struct {
	id                                 types.PetID
	name                               string
	alive, hibernating, exploring      bool
	age, health, happiness, lastBredAt float64
	genome                             *genetics.Genome
	personality                        ai.PersonalityMatrix
	hibernationEnabled                 bool
	owner                              types.UserID
}

// breedingInfo snapshots the pet for breeding, so two pets' locks are
// never held at once
func (p *DigitalPet) breedingInfo() breedingInfo {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.ensureGenomeLocked()

	traits := *p.Personality.Traits
	personality := *p.Personality
	personality.Traits = &traits
	return breedingInfo{
		id:                 p.ID,
		name:               p.Name,
		alive:              p.Biology.IsAlive,
		hibernating:        p.Hibernating,
		exploring:          p.isExploringLocked(),
		age:                p.Biology.GetAgeInDays(),
		health:             p.Biology.Vitals.Health,
		happiness:          p.Biology.Vitals.Happiness,
		lastBredAt:         p.LastBredAge,
		genome:             p.Genome.Clone(),
		personality:        personality,
		hibernationEnabled: p.HibernationEnabled,
		owner:              p.Owner,
	}
}

// lastBreedingTime converts the age at the last breeding into the real
// time the genetics package expects, so its cooldown counts game days
func (b breedingInfo) lastBreedingTime(now time.Time) time.Time {
	if b.lastBredAt == 0 {
		return time.Time{}
	}
	return now.Add(-time.Duration((b.age - b.lastBredAt) * 24 * float64(time.Hour)))
}

// candidate describes the pet for the genetics package's requirement
// checks
func (b breedingInfo) candidate(now time.Time) genetics.BreedingCandidate {
	return genetics.BreedingCandidate{Name: b.name, Age: b.age, Health: b.health, Happiness: b.happiness,
		LastBreeding: b.lastBreedingTime(now)}
}

// unavailable explains why a pet can't breed right now, or returns ""
func (b breedingInfo) unavailable() string {
	switch {
	case !b.alive:
		return b.name + " has passed away"
	case b.hibernating:
		return b.name + " is hibernating"
	case b.exploring:
		return b.name + " is out exploring"
	}
	return ""
}

// BreedingCompatibility returns how genetically compatible two pets are
// (0.0 to 1.0) and, if they can't breed right now, why
func BreedingCompatibility(a, b *DigitalPet) (float64, string) {
	if a == b {
		return 0, ErrSamePet.Error()
	}
	infoA, infoB := a.breedingInfo(), b.breedingInfo()
	gs := genetics.NewGeneticSystem()
	compatibility := gs.CalculateCompatibility(infoA.genome, infoB.genome)

	for _, info := range []breedingInfo{infoA, infoB} {
		if reason := info.unavailable(); reason != "" {
			return compatibility, reason
		}
	}
	now := time.Now()
	if reason := gs.CheckRequirements(infoA.candidate(now), infoB.candidate(now)); reason != "" {
		return compatibility, reason
	}
	return compatibility, gs.CompatibilityProblem(compatibility)
}

// Breed tries to have a baby named childName with two pets. It can fail
// if the pets don't meet the requirements or by chance (likelier for less
// compatible pairs); a BreedingError gives the reason. On success both
// parents start their breeding cooldown and the family is recorded in
// everyone's relationships.
func Breed(a, b *DigitalPet, childName string) (*DigitalPet, error) {
	if a == b {
		return nil, ErrSamePet
	}
	infoA, infoB := a.breedingInfo(), b.breedingInfo()
	for _, info := range []breedingInfo{infoA, infoB} {
		if reason := info.unavailable(); reason != "" {
			return nil, BreedingError{reason}
		}
	}

	gs := genetics.NewGeneticSystem()
	now := time.Now()
	if reason := gs.CheckRequirements(infoA.candidate(now), infoB.candidate(now)); reason != "" {
		return nil, BreedingError{reason}
	}
	result := gs.Breed(infoA.genome, infoB.genome) // Checks compatibility; can fail by chance
	if !result.Success {
		return nil, BreedingError{result.FailureReason}
	}

	// The baby's personality: inherited from both parents, with the
	// genetically based traits taken from its genome
	child := NewDigitalPet(childName, infoA.owner)
	child.Genome = result.OffspringGenome
	child.Personality.Traits = infoA.personality.InheritTraits(&infoA.personality, &infoB.personality, childMutationRate)
	for _, link := range genePersonality {
		link.set(child.Personality.Traits, geneValue(child.Genome, link.gene))
	}
	child.Personality.Traits.Clamp()
	child.Biology.Processes.ImmuneStrength = geneValue(child.Genome, genetics.TraitImmuneStrength)
	child.HibernationEnabled = infoA.hibernationEnabled
	child.Relationships.AddRelationship(infoA.id, types.RelationshipParent)
	child.Relationships.AddRelationship(infoB.id, types.RelationshipParent)

	for _, parent := range []struct {
		pet  *DigitalPet
		mate types.PetID
	}{{a, infoB.id}, {b, infoA.id}} {
		parent.pet.mu.Lock()
		parent.pet.LastBredAge = parent.pet.Biology.GetAgeInDays()
		parent.pet.Relationships.AddRelationship(parent.mate, types.RelationshipMate)
		parent.pet.Relationships.AddRelationship(child.ID, types.RelationshipOffspring)
		parent.pet.mu.Unlock()
	}
	return child, nil
}

// Generation returns the pet's generation (1 for a pet that was not bred)
func (p *DigitalPet) Generation() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.ensureGenomeLocked()
	return p.Genome.Generation
}
