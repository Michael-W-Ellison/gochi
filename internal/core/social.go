package core

import (
	"fmt"

	"github.com/Michael-W-Ellison/gochi/internal/ai"
	"github.com/Michael-W-Ellison/gochi/internal/biology"
	"github.com/Michael-W-Ellison/gochi/internal/social"
	"github.com/Michael-W-Ellison/gochi/pkg/types"
)

// Social introductions: the player introduces the active pet to another of
// their pets. How a meeting goes depends on both personalities, their
// species and how earlier meetings went; repeated meetings build a
// friendship or a rivalry in both pets' relationships.

// meetingOutcome is how a meeting went
type meetingOutcome struct {
	quality  float64 // 0.0 (a fight) to 1.0 (instant friends)
	reaction float64 // How the active pet takes it (reaction score)
	lesson   string
}

// Meeting reports an introduction
type Meeting struct {
	InteractionResult
	Text         string  `json:"text"`
	Quality      float64 `json:"quality"`      // 0.0 to 1.0
	Relationship string  `json:"relationship"` // How the two get along now
}

// Meeting quality levels
const (
	goodMeeting     = 0.6  // Played together
	cautiousMeeting = 0.45 // Sniffed each other and kept their distance
)

// socialInfo snapshots a pet for a meeting, so two pets' locks are never
// held at once
type socialInfo struct {
	id                         types.PetID
	name                       string
	species                    Species
	traits                     ai.Traits
	alive, hibernating, asleep bool
	exploring                  bool
	energy                     float64
	young                      bool
	relations                  map[types.PetID]float64 // How it gets along with pets it knows (relationScore)
}

func (p *DigitalPet) socialInfo() socialInfo {
	p.mu.Lock()
	defer p.mu.Unlock()
	info := socialInfo{
		id: p.ID, name: p.Name, species: p.speciesLocked(), traits: *p.Personality.Traits,
		alive: p.Biology.IsAlive, hibernating: p.Hibernating, exploring: p.isExploringLocked(),
		asleep: p.CurrentBehavior == types.BehaviorSleeping, energy: p.Biology.Vitals.Energy,
		young: p.Biology.LifeStage() == biology.StageYoung, relations: make(map[types.PetID]float64),
	}
	for id, rel := range p.Relationships.Relationships {
		info.relations[id] = relationScore(rel)
	}
	return info
}

// unavailable explains why a pet can't meet another now, or returns ""
func (s socialInfo) unavailable() string {
	switch {
	case !s.alive:
		return s.name + " has passed away"
	case s.hibernating:
		return s.name + " is hibernating"
	case s.exploring:
		return s.name + " is out exploring"
	case s.asleep:
		return s.name + " is asleep"
	}
	return ""
}

// sociability is how readily a pet takes to others (0.0 to 1.0)
func sociability(t ai.Traits) float64 {
	return (t.Extraversion + t.Agreeableness + (1 - t.Territoriality) + (1 - t.Neuroticism)) / 4
}

// speciesMeeting is how well each pairing of species tends to get along
// at first: dogs are social animals; cats are territorial; a cat and a dog
// need the most care
func speciesMeeting(a, b Species) float64 {
	switch {
	case a == SpeciesDog && b == SpeciesDog:
		return 0.1
	case a == SpeciesCat && b == SpeciesCat:
		return -0.1
	}
	return -0.15
}

// introductionLessons teach how to introduce each pairing of species
func introductionLesson(a, b Species) string {
	switch {
	case a == SpeciesCat && b == SpeciesCat:
		return "Introduce cats slowly: keep them apart at first, swap bedding so they learn each other's scent, " +
			"then let them see each other before short, supervised meetings."
	case a == SpeciesDog && b == SpeciesDog:
		return "Introduce dogs on neutral ground, on loose leashes, with a walk side by side before they meet face to face."
	}
	return "Keep the dog on a leash for early meetings and give the cat high places and a way out; never let the dog chase the cat."
}

// Introduce introduces pet a to pet b. Both pets' relationships record the
// meeting; a also receives it as care (with its reaction, learning and
// points).
func Introduce(a, b *DigitalPet) (Meeting, error) {
	if a == b {
		return Meeting{}, ErrSamePet
	}
	infoA, infoB := a.socialInfo(), b.socialInfo()
	for _, info := range []socialInfo{infoA, infoB} {
		if reason := info.unavailable(); reason != "" {
			return Meeting{}, RefusedError{reason + "."}
		}
	}

	meeting, bond, err := a.meet(infoA, infoB, "met")
	if err != nil {
		return Meeting{}, err
	}

	b.mu.Lock()
	b.recordMeetingLocked(infoA, bond, "met")
	b.meetLocked(meeting.Quality)
	b.mu.Unlock()
	return meeting, nil
}

// meet has this pet (a) meet another (b): how the meeting goes depends on
// both pets' sociability, their species and how a's earlier meetings with
// b went. a receives it as care and records it (what they did) in its
// relationships. It returns the meeting and the bond to record on b's side.
func (a *DigitalPet) meet(infoA, infoB socialInfo, what string) (Meeting, float64, error) {
	a.mu.Lock()
	quality := (sociability(infoA.traits)+sociability(infoB.traits))/2 + speciesMeeting(infoA.species, infoB.species)
	if rel, ok := a.Relationships.GetRelationship(infoB.id); ok {
		quality += (rel.BondStrength-0.3)*0.5 - rel.Rivalry*0.3
	} else {
		quality -= 0.05 // First meetings are cautious
	}
	quality = min(max(quality, 0), 1)

	lesson := introductionLesson(infoA.species, infoB.species)
	if quality < cautiousMeeting {
		lesson += " Hissing or growling means slow down: go back a step and try again later."
	}
	outcome := meetingOutcome{quality: quality, reaction: (quality - cautiousMeeting) * 1.5, lesson: lesson}

	// Repeated meetings in quick succession count less
	weight := 1 / (1 + a.recentCareLocked(types.InteractionSocialIntroduction))
	result := a.interactWithLocked(types.InteractionSocialIntroduction, careChoice{meeting: &outcome})
	if !result.Applied {
		a.mu.Unlock()
		return Meeting{}, 0, RefusedError{result.Refusal}
	}
	bond := (quality - cautiousMeeting) * 2 * weight
	description := a.recordMeetingLocked(infoB, bond, what)
	a.mu.Unlock()

	meeting := Meeting{InteractionResult: result, Quality: quality, Relationship: description}
	verb := "growled at"
	if infoA.species == SpeciesCat {
		verb = "hissed at"
	}
	switch {
	case quality >= goodMeeting:
		meeting.Text = fmt.Sprintf("%s and %s played together happily!", infoA.name, infoB.name)
	case quality >= cautiousMeeting:
		meeting.Text = fmt.Sprintf("%s and %s sniffed each other cautiously and kept their distance.", infoA.name, infoB.name)
	default:
		meeting.Text = fmt.Sprintf("%s %s %s. The meeting didn't go well.", infoA.name, verb, infoB.name)
	}
	return meeting, bond, nil
}

// recordMeetingLocked records time spent with another pet (what they did)
// in this pet's relationships, and returns how they get along now. Caller
// must hold p.mu.
func (p *DigitalPet) recordMeetingLocked(other socialInfo, bond float64, what string) string {
	rel, ok := p.Relationships.GetRelationship(other.id)
	if !ok {
		kind := types.RelationshipFriend
		if bond < 0 {
			kind = types.RelationshipRival
		}
		rel = p.Relationships.AddRelationship(other.id, kind)
		// A first meeting sets the tone
		rel.BondStrength += bond * 0.1
		rel.Rivalry -= bond * 0.15
		rel.Clamp()
	}
	rel.Update(max(min(bond, 1), -1), p.Biology.GetAgeInDays())
	rel.RecordShared(what, p.Biology.GetAgeInDays(), bond)

	// Friends and rivals follow how the meetings go; family stays family
	if rel.Type == types.RelationshipFriend || rel.Type == types.RelationshipRival {
		rel.Type = types.RelationshipRival
		if relationScore(rel) >= neutralRelation {
			rel.Type = types.RelationshipFriend
		}
	}
	p.rememberPetLocked(other.id, other.name)
	return describeRelation(rel)
}

// neutralRelation is the relation score of two pets that have just met
// (a new relationship's bond minus its rivalry)
const neutralRelation = 0.2

// relationScore is how well two pets get along: their bond minus their
// rivalry
func relationScore(rel *social.Relationship) float64 {
	return rel.BondStrength - rel.Rivalry
}

// describeRelation describes how two pets get along, from how their
// meetings went
func describeRelation(rel *social.Relationship) string {
	score := relationScore(rel)
	switch rel.Type {
	case types.RelationshipFriend:
		switch {
		case score >= 0.7:
			return "best friends"
		case score >= 0.5:
			return "good friends"
		case score >= 0.3:
			return "friends"
		}
		return "acquaintances"
	case types.RelationshipRival:
		switch {
		case score >= 0.1:
			return "wary of each other"
		case score >= -0.1:
			return "don't get along"
		}
		return "bitter rivals"
	}
	return rel.GetDescription()
}

// meetLocked lets the other pet in a meeting feel how it went. Caller
// must hold p.mu.
func (p *DigitalPet) meetLocked(quality float64) {
	v := p.Biology.Vitals
	if quality >= cautiousMeeting {
		v.Happiness += 0.1 * quality
		p.Emotions.ApplyEmotionalStimulus(ai.EmotionalStimulus{JoyDelta: 0.1 * quality, LonelinessDelta: -0.15 * quality,
			Source: "meeting"})
		p.ensureNeedsLocked()
		p.Needs.SatisfyNeed(types.NeedSocial, quality, p.Biology.GetAgeInDays())
	} else {
		v.Stress += 0.1
		p.Emotions.ApplyEmotionalStimulus(ai.EmotionalStimulus{FearDelta: 0.1, AngerDelta: 0.05, Source: "meeting"})
	}
	v.Clamp()
}

// rememberPetLocked remembers another pet's name, to show its
// relationships. Caller must hold p.mu.
func (p *DigitalPet) rememberPetLocked(id types.PetID, name string) {
	if p.KnownPets == nil {
		p.KnownPets = make(map[types.PetID]string)
	}
	p.KnownPets[id] = name
}

// PetRelation is how the pet gets along with another pet
type PetRelation struct {
	Name        string `json:"name"`
	Kind        string `json:"kind"` // friend, rival, mate, parent or baby
	Description string `json:"description"`
	LastShared  string `json:"last_shared,omitempty"` // What they last did together, such as "napped together"
}

// relationKinds name each relationship type
var relationKinds = map[types.RelationshipType]string{
	types.RelationshipFriend:    "friend",
	types.RelationshipRival:     "rival",
	types.RelationshipMate:      "mate",
	types.RelationshipParent:    "parent",
	types.RelationshipOffspring: "baby",
}

// Relations lists the pets this pet knows, by name
func (p *DigitalPet) Relations() []PetRelation {
	p.mu.Lock()
	defer p.mu.Unlock()

	relations := []PetRelation{}
	for id, rel := range p.Relationships.Relationships {
		name, ok := p.KnownPets[id]
		if !ok {
			continue
		}
		relation := PetRelation{Name: name, Kind: relationKinds[rel.Type], Description: describeRelation(rel)}
		if last, ok := rel.LastShared(); ok {
			relation.LastShared = last.Description
		}
		relations = append(relations, relation)
	}
	sortRelations(relations)
	return relations
}

// sortRelations orders relations by name
func sortRelations(r []PetRelation) {
	for i := 1; i < len(r); i++ {
		for j := i; j > 0 && r[j].Name < r[j-1].Name; j-- {
			r[j], r[j-1] = r[j-1], r[j]
		}
	}
}

// RelationTo describes how the pet gets along with another pet, or ""
// if they haven't met
func (p *DigitalPet) RelationTo(id types.PetID) string {
	p.mu.Lock()
	defer p.mu.Unlock()
	if rel, ok := p.Relationships.GetRelationship(id); ok {
		return describeRelation(rel)
	}
	return ""
}
