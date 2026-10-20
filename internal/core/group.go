package core

import (
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/Michael-W-Ellison/gochi/internal/ai"
	"github.com/Michael-W-Ellison/gochi/internal/biology"
	"github.com/Michael-W-Ellison/gochi/pkg/types"
)

// Group activities: the active pet spends time with up to three of the
// player's other pets - playing, napping or (dogs) going for a walk. How
// it goes depends on each pair: their sociability, species and history,
// with cats territorial in a crowd and dogs at their best as a pack. The
// pets sort out a pecking order: in a disagreement a clearly lower-ranked
// pet backs down, while two of similar rank can scuffle. The pets signal
// to each other the way real cats and dogs do, and every pair's
// relationship records the time together.

// Group activity errors
var (
	ErrGroupSize    = errors.New("a group activity needs 2 to 4 different pets")
	ErrUnknownGroup = errors.New("unknown group activity")
	ErrDogsOnlyWalk = errors.New("only dogs go for walks")
)

// Group activities
const (
	GroupPlay = "play"
	GroupWalk = "walk"
	GroupNap  = "nap"
)

// maxGroupSize is the most pets in a group activity
const maxGroupSize = 4

// groupActivity describes what a group does together
type groupActivity struct {
	name     string // "played together"
	icon     string
	energy   float64 // Energy needed to join, and used
	fatigue  float64 // Tiredness it relieves (napping)
	exercise float64 // Play sessions' worth of exercise
	kind     biology.ExerciseType
	bonus    func(a, b socialInfo) float64
	lesson   map[Species]string
}

var groupActivities = map[string]groupActivity{
	GroupPlay: {name: "played together", icon: "🎾", energy: 0.1, exercise: 1,
		bonus: func(a, b socialInfo) float64 {
			return 0.1 * ((a.traits.Playfulness+b.traits.Playfulness)/2 - 0.5)
		},
		lesson: map[Species]string{
			SpeciesCat: "Cats play best with cats they know. Give each cat its own toy, and stop the game if ears go flat or tails lash.",
			SpeciesDog: "Watch dogs at play: play bows, loose bodies and taking turns chasing are good signs; stiff bodies and pinning mean it's time for a break.",
		}},
	GroupWalk: {name: "went for a walk together", icon: "🦮", energy: 0.15, exercise: 1.2, kind: biology.ExerciseWalk,
		bonus: func(a, b socialInfo) float64 { return 0.1 }, // Dogs are pack animals
		lesson: map[Species]string{
			SpeciesDog: "Walking dogs together is good exercise and builds their bond; walk them side by side and keep leashes from tangling.",
		}},
	GroupNap: {name: "napped together", icon: "💤", fatigue: 0.15,
		bonus: func(a, b socialInfo) float64 {
			// Only close companions share a sleeping spot
			if a.relations[b.id] >= 0.5 && b.relations[a.id] >= 0.5 {
				return 0.1
			}
			return -0.15
		},
		lesson: map[Species]string{
			SpeciesCat: "Cats that sleep curled up together are bonded; cats that don't get along need separate beds and resting places.",
			SpeciesDog: "Dogs that rest together trust each other. Give each dog its own bed too, so it can choose.",
		}},
}

// conflictLevel is the pair harmony below which two pets clash
const conflictLevel = cautiousMeeting

// clearRank is the difference in dominance at which one pet simply backs
// down instead of fighting
const clearRank = 0.15

// Dominance returns where the pet stands in a pecking order, 0.0 to 1.0:
// outgoing, territorial, confident and less agreeable pets rank higher,
// and the young lower
func dominance(info socialInfo) float64 {
	t := info.traits
	d := 0.35*t.Extraversion + 0.3*t.Territoriality + 0.2*(1-t.Neuroticism) + 0.15*(1-t.Agreeableness)
	if info.young {
		d -= 0.2
	}
	return max(min(d, 1), 0)
}

// pairHarmony is how well two pets get along in an activity, 0.0 to 1.0:
// like a meeting (social.go), plus the activity and the crowd. Territorial
// cats get on worse the more pets there are.
func pairHarmony(a, b socialInfo, activity groupActivity, groupSize int) float64 {
	h := (sociability(a.traits)+sociability(b.traits))/2 + speciesMeeting(a.species, b.species)
	history := 0
	for _, side := range []struct{ from, to socialInfo }{{a, b}, {b, a}} {
		if score, ok := side.from.relations[side.to.id]; ok {
			h += (score - neutralRelation) * 0.25
			history++
		}
	}
	if history == 0 {
		h -= 0.05 // Strangers are cautious
	}
	h += activity.bonus(a, b)
	h += 0.1 * ((a.traits.Agreeableness+b.traits.Agreeableness)/2 - 0.5) // Cooperation
	for _, pet := range []socialInfo{a, b} {
		if pet.species == SpeciesCat && groupSize > 2 {
			h -= 0.05 * float64(groupSize-2) * pet.traits.Territoriality
		}
	}
	return max(min(h, 1), 0)
}

// ConflictResult describes a clash between two pets
type ConflictResult struct {
	Pets    [2]string `json:"pets"`
	Outcome string    `json:"outcome"` // "backed down" or "scuffle"
	Text    string    `json:"text"`
	Lesson  string    `json:"lesson"`
}

// conflictLessons explain how to handle conflict between pets
var conflictLessons = map[Species]string{
	SpeciesCat: "Cats that clash need space: separate them, give each its own food, litter box and high places, and reintroduce them slowly. Never punish hissing; it's a warning.",
	SpeciesDog: "Interrupt a scuffle calmly with a noise or a distraction, never with your hands. Separate the dogs to calm down, and keep later meetings short and positive.",
}

// Who backed down in a clash
const (
	neitherBackedDown = -1
	aBackedDown       = 0
	bBackedDown       = 1
)

// conflict works out how a clash between two pets goes: the lower-ranked
// pet backs down if their ranks are clearly apart, otherwise they scuffle.
// It returns the result, the bond change for both, and who backed down.
func conflict(a, b socialInfo, harmony float64) (ConflictResult, float64, int) {
	result := ConflictResult{Pets: [2]string{a.name, b.name}, Lesson: conflictLessons[a.species]}
	if a.species != b.species {
		result.Lesson = "A cat and a dog that clash need to be kept apart for a while: give the cat high places and a way out, and keep the dog on a leash for the next few meetings."
	}
	high, low, loser := a, b, bBackedDown
	if dominance(b) > dominance(a) {
		high, low, loser = b, a, aBackedDown
	}

	if dominance(high)-dominance(low) >= clearRank {
		result.Outcome = "backed down"
		result.Text = fmt.Sprintf("%s %s %s, and %s backed down.", high.name, threatSignal(high.species), low.name, low.name)
		return result, harmony - conflictLevel, loser // Mild: the order is settled
	}
	result.Outcome = "scuffle"
	result.Text = fmt.Sprintf("%s and %s got into a scuffle and had to be separated.", a.name, b.name)
	return result, (harmony - conflictLevel) * 3, neitherBackedDown
}

// threatSignal is how a species warns another pet off
func threatSignal(s Species) string {
	if s == SpeciesDog {
		return "growled at"
	}
	return "hissed at"
}

// Signals are how pets talk to each other, by species and how they feel
var friendlySignals = map[Species][]string{
	SpeciesCat: {"slow-blinked at", "rubbed against", "touched noses with"},
	SpeciesDog: {"play-bowed to", "wagged its tail at", "licked"},
}
var warySignals = map[Species][]string{
	SpeciesCat: {"watched", "kept its distance from"},
	SpeciesDog: {"sniffed", "circled"},
}

// signal describes how a pet communicated with another, from their harmony
func signal(from, to socialInfo, harmony float64, n int) string {
	switch {
	case harmony >= goodMeeting:
		options := friendlySignals[from.species]
		return fmt.Sprintf("%s %s %s.", from.name, options[n%len(options)], to.name)
	case harmony >= conflictLevel:
		options := warySignals[from.species]
		return fmt.Sprintf("%s %s %s.", from.name, options[n%len(options)], to.name)
	}
	return fmt.Sprintf("%s %s %s.", from.name, threatSignal(from.species), to.name)
}

// GroupResult reports a group activity
type GroupResult struct {
	InteractionResult                  // For the active pet
	Activity          string           `json:"activity"`
	Text              string           `json:"text"`
	Signals           []string         `json:"signals"`   // What the pets said to each other
	Conflicts         []ConflictResult `json:"conflicts"` // Clashes, if any
	Leader            string           `json:"leader"`    // Top of the pecking order
	Order             []string         `json:"order"`     // The pecking order, highest first
	Harmony           float64          `json:"harmony"`   // How well the group got on, 0.0 to 1.0
	Happiness         float64          `json:"happiness"` // Group happiness afterwards (CalculateGroupHappiness)
}

// ProcessGroupActivity has pets do an activity together. pets[0] is the
// active pet, which receives it as care (with its reaction, learning and
// points); every pet's mood, needs and relationships follow how it went.
// The caller saves the other pets.
func ProcessGroupActivity(activityID string, pets ...*DigitalPet) (GroupResult, error) {
	activity, ok := groupActivities[activityID]
	if !ok {
		return GroupResult{}, ErrUnknownGroup
	}
	if len(pets) < 2 || len(pets) > maxGroupSize {
		return GroupResult{}, ErrGroupSize
	}
	infos := make([]socialInfo, len(pets))
	seen := make(map[*DigitalPet]bool)
	for i, pet := range pets {
		if seen[pet] {
			return GroupResult{}, ErrGroupSize
		}
		seen[pet] = true
		infos[i] = pet.socialInfo()
		if reason := infos[i].unavailable(); reason != "" {
			return GroupResult{}, RefusedError{reason + "."}
		}
		if activityID == GroupWalk && infos[i].species != SpeciesDog {
			return GroupResult{}, ErrDogsOnlyWalk
		}
		if infos[i].energy < activity.energy+0.1 {
			return GroupResult{}, RefusedError{infos[i].name + " is too tired to join in."}
		}
	}

	// Every pair: how well they got on, what they signalled, any clash
	result := GroupResult{Activity: activityID, Signals: []string{}, Conflicts: []ConflictResult{}}
	n := len(pets)
	harmony := make([]float64, n) // Each pet's average harmony with the others
	scuffled := make([]bool, n)
	backedDown := make([]bool, n)
	bonds := make([][]float64, n)
	for i := range bonds {
		bonds[i] = make([]float64, n)
	}
	total, pairs := 0.0, 0
	for i := 0; i < n; i++ {
		for j := i + 1; j < n; j++ {
			h := pairHarmony(infos[i], infos[j], activity, n)
			bond := (h - cautiousMeeting) * 2
			if h < conflictLevel {
				clash, b, loser := conflict(infos[i], infos[j], h)
				result.Conflicts = append(result.Conflicts, clash)
				bond = b
				switch loser {
				case aBackedDown:
					backedDown[i] = true
				case bBackedDown:
					backedDown[j] = true
				default:
					scuffled[i], scuffled[j] = true, true
				}
				h /= 2 // A clash spoils the activity for both
			} else {
				result.Signals = append(result.Signals, signal(infos[i], infos[j], h, pairs))
			}
			bonds[i][j], bonds[j][i] = bond, bond
			harmony[i] += h / float64(n-1)
			harmony[j] += h / float64(n-1)
			total += h
			pairs++
		}
	}
	result.Harmony = total / float64(pairs)

	// The pecking order
	order := make([]socialInfo, n)
	copy(order, infos)
	sort.SliceStable(order, func(i, j int) bool { return dominance(order[i]) > dominance(order[j]) })
	for _, info := range order {
		result.Order = append(result.Order, info.name)
	}
	result.Leader = order[0].name

	// Each pet takes part; the active pet as care
	lesson := activity.lesson[infos[0].species]
	if len(result.Conflicts) > 0 {
		lesson = result.Conflicts[0].Lesson
	}
	for i, pet := range pets {
		pet.mu.Lock()
		if i == 0 {
			outcome := meetingOutcome{quality: harmony[0], reaction: (harmony[0] - cautiousMeeting) * 1.5, lesson: lesson}
			care := pet.interactWithLocked(types.InteractionSocialIntroduction, careChoice{meeting: &outcome})
			if !care.Applied {
				pet.mu.Unlock()
				return GroupResult{}, RefusedError{care.Refusal}
			}
			result.InteractionResult = care
		}
		pet.groupActivityLocked(activity, harmony[i], i == 0)
		pet.conflictLocked(scuffled[i], backedDown[i])
		for j, other := range infos {
			if j != i {
				pet.recordMeetingLocked(other, bonds[i][j], activity.name)
			}
		}
		pet.mu.Unlock()
	}

	result.Happiness = CalculateGroupHappiness(pets...)
	result.Text = groupText(infos, activity, result)
	return result, nil
}

// groupActivityLocked applies a group activity to one pet: how much it
// enjoyed the company (harmony, 0.0 to 1.0) moves its mood, stress and
// social need, and the activity itself exercises or rests it. The active
// pet's care already counts its social need and points. Caller must hold
// p.mu.
func (p *DigitalPet) groupActivityLocked(activity groupActivity, harmony float64, active bool) {
	v := p.Biology.Vitals
	age := p.Biology.GetAgeInDays()
	p.ensureNeedsLocked()
	if !active {
		p.meetLocked(harmony)
	}
	if harmony >= goodMeeting {
		p.Biology.EnsureHormones()
		p.Biology.Hormones.Pulse(biology.HormoneOxytocin, 0.15*harmony) // Good company is bonding
	}
	if activity.exercise > 0 {
		p.Biology.ProcessExercise(activity.kind, activity.exercise)
		p.Needs.SatisfyNeed(types.NeedExercise, activity.exercise, age)
	}
	v.Energy -= activity.energy
	v.Fatigue -= activity.fatigue * harmony // Only restful in good company
	v.Clamp()
	if harmony >= goodMeeting {
		p.evolveLocked(ai.ExperienceData{TraitAffected: "extraversion", Intensity: 0.1},
			ai.ExperienceData{TraitAffected: "agreeableness", Intensity: 0.05})
	}
}

// groupText summarizes how a group activity went
func groupText(infos []socialInfo, activity groupActivity, result GroupResult) string {
	names := make([]string, len(infos))
	for i, info := range infos {
		names[i] = info.name
	}
	who := strings.Join(names[:len(names)-1], ", ") + " and " + names[len(names)-1]
	text := fmt.Sprintf("%s %s %s", activity.icon, who, activity.name)
	switch {
	case len(result.Conflicts) > 0:
		text += ", but not everyone got along."
	case result.Harmony >= goodMeeting:
		text += " happily!"
	default:
		text += ", a little warily."
	}
	if len(infos) > 2 || len(result.Conflicts) > 0 {
		if infos[0].species == SpeciesDog {
			text += fmt.Sprintf(" %s leads the pack.", result.Leader)
		} else {
			text += fmt.Sprintf(" %s is top cat.", result.Leader)
		}
	}
	return text
}

// CalculateGroupHappiness returns how happy a group of pets is together,
// 0.0 to 1.0: their average happiness, lowered when they don't get along
// (bond minus rivalry between each pair that knows each other)
func CalculateGroupHappiness(pets ...*DigitalPet) float64 {
	if len(pets) == 0 {
		return 0
	}
	infos := make([]socialInfo, len(pets))
	happiness := 0.0
	for i, pet := range pets {
		infos[i] = pet.socialInfo()
		pet.mu.Lock()
		happiness += pet.Biology.Vitals.Happiness
		pet.mu.Unlock()
	}
	happiness /= float64(len(pets))

	relations, pairs := 0.0, 0
	for i := range infos {
		for j := range infos {
			if score, ok := infos[i].relations[infos[j].id]; ok && i != j {
				relations += score
				pairs++
			}
		}
	}
	if pairs == 0 {
		return happiness
	}
	harmony := max(min(0.5+(relations/float64(pairs)-neutralRelation), 1), 0)
	return happiness * (0.6 + 0.4*harmony)
}

// HandleConflict settles a clash between two pets: the lower-ranked one
// backs down if their ranks are clearly apart, otherwise they scuffle and
// both are stressed. Both relationships record it.
func HandleConflict(a, b *DigitalPet) (ConflictResult, error) {
	if a == b {
		return ConflictResult{}, ErrSamePet
	}
	infoA, infoB := a.socialInfo(), b.socialInfo()
	result, bond, loser := conflict(infoA, infoB, 0)
	for i, side := range []struct {
		pet   *DigitalPet
		other socialInfo
	}{{a, infoB}, {b, infoA}} {
		side.pet.mu.Lock()
		side.pet.conflictLocked(loser == neitherBackedDown, loser == i)
		side.pet.recordMeetingLocked(side.other, bond, "clashed")
		side.pet.mu.Unlock()
	}
	return result, nil
}

// conflictLocked applies a clash to the pet: a scuffle stresses and
// frightens it; backing down stresses it a little. Caller must hold p.mu.
func (p *DigitalPet) conflictLocked(scuffled, backedDown bool) {
	v := p.Biology.Vitals
	switch {
	case scuffled:
		v.Stress += 0.15
		v.Happiness -= 0.1
		p.evolveFromHardshipLocked(0.2)
	case backedDown:
		v.Stress += 0.05
	}
	v.Clamp()
}

// UpdateRelationships lets the pet's relationships fade over deltaTime game
// days with pets it hasn't spent time with lately (see social.Relationship.Decay).
// Update does this as the pet's time passes.
func (p *DigitalPet) UpdateRelationships(deltaTime float64) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.updateRelationshipsLocked(deltaTime)
}

// updateRelationshipsLocked lets relationships fade. Caller must hold p.mu.
func (p *DigitalPet) updateRelationshipsLocked(deltaTime float64) {
	p.Relationships.UpdateAll(p.Biology.GetAgeInDays(), deltaTime)
}
