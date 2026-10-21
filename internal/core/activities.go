package core

import (
	"errors"
	"fmt"
	"math"
	"math/rand"
	"strings"

	"github.com/Michael-W-Ellison/gochi/internal/ai"
	"github.com/Michael-W-Ellison/gochi/internal/biology"
	"github.com/Michael-W-Ellison/gochi/pkg/types"
)

// Activity errors
var (
	ErrUnknownTrick  = errors.New("unknown trick")
	ErrTrickNotKnown = errors.New("pet has not learned this trick yet")
	ErrTrickRusty    = errors.New("pet is rusty at this trick")
	ErrTooTired      = errors.New("pet is too tired")
	ErrExploring     = errors.New("pet is out exploring")
	ErrNeedsCare     = errors.New("pet needs food, water and energy first")
	ErrNoGame        = errors.New("no game in progress")
	ErrInvalidSpot   = errors.New("invalid hiding spot")
	ErrNotExploring  = errors.New("pet is not out exploring")
	ErrInjured       = errors.New("pet is injured")
	ErrHibernating   = errors.New("pet is hibernating")
	ErrSleeping      = errors.New("pet is sleeping")
)

// RefusedError reports that the pet turned an activity down, with the
// reason in its own words (for example "Biscuit is too stressed for
// training.")
type RefusedError struct {
	Reason string
}

func (e RefusedError) Error() string {
	return e.Reason
}

// randIntn picks random numbers for games and souvenirs (replaced in tests)
var randIntn = rand.Intn

// ---------------------------------------------------------------------------
// Tricks

// minTrainingEnergy is the energy a pet needs to train or play games
const minTrainingEnergy = 0.25

// TrickProgress is a trick with the pet's skill at it
type TrickProgress struct {
	Trick
	Skill   float64 `json:"skill"`            // 0.0 to 1.0
	Learned bool    `json:"learned"`          // Can perform it now
	Rusty   bool    `json:"rusty"`            // Learned it once, but needs practice before performing it again
	Locked  string  `json:"locked,omitempty"` // Why it can't be started yet (a trick to learn first)
}

// Trick skills fade without practice. A trick practiced within
// trickRustGrace game days keeps its skill; after that the skill falls by
// trickRustRate per game day. A trick the pet has learned never falls below
// rustyTrickFloor, so a session or two of practice brings it back. At the
// default pace (one game day per real minute, counted only while the game
// is running) a trick stays sharp for two hours of play and is rusty after
// about four.
const (
	trickRustGrace  = 120.0
	trickRustRate   = 0.25 / 120
	rustyTrickFloor = 0.75
)

// decayTricksLocked lets unpracticed tricks get rusty over deltaTime game
// days. Caller must hold p.mu.
func (p *DigitalPet) decayTricksLocked(deltaTime float64) {
	if len(p.TrickSkills) == 0 {
		return
	}
	age := p.Biology.GetAgeInDays()
	if p.TrickPracticedAge == nil {
		p.TrickPracticedAge = make(map[string]float64)
	}
	for id, skill := range p.TrickSkills {
		if skill >= 1.0 {
			p.markTrickLearnedLocked(id) // Saved before learning was recorded
		}
		practiced, ok := p.TrickPracticedAge[id]
		if !ok {
			// Saved before practice was recorded: the grace period starts now
			p.TrickPracticedAge[id] = age
			continue
		}
		// Only the part of deltaTime past the grace period counts
		rusting := min(deltaTime, age-practiced-trickRustGrace)
		if rusting <= 0 {
			continue
		}
		floor := 0.0
		if p.TricksLearned[id] {
			floor = rustyTrickFloor
		}
		if skill > floor {
			p.TrickSkills[id] = max(skill-trickRustRate*rusting, floor)
		}
	}
}

// markTrickLearnedLocked records that the pet has learned a trick. Caller
// must hold p.mu.
func (p *DigitalPet) markTrickLearnedLocked(id string) {
	if p.TricksLearned == nil {
		p.TricksLearned = make(map[string]bool)
	}
	p.TricksLearned[id] = true
}

// practiceTrickLocked records that the pet practiced a trick now. Caller
// must hold p.mu.
func (p *DigitalPet) practiceTrickLocked(id string) {
	if p.TrickPracticedAge == nil {
		p.TrickPracticedAge = make(map[string]float64)
	}
	p.TrickPracticedAge[id] = p.Biology.GetAgeInDays()
}

// TrickResult reports a training session or performance
type TrickResult struct {
	InteractionResult
	Trick       Trick
	Skill       float64 // Skill after training
	JustLearned bool    // The trick was learned in this session
	Text        string  // What happened, e.g. "Biscuit learned Spin!"
}

// checkActiveLocked returns why the pet cannot do an activity, or nil.
// Caller must hold p.mu.
func (p *DigitalPet) checkActiveLocked() error {
	switch {
	case !p.Biology.IsAlive:
		return ErrPetNotAvailable
	case p.Hibernating:
		return ErrHibernating
	case p.isExploringLocked():
		return ErrExploring
	case p.CurrentBehavior == types.BehaviorSleeping:
		return ErrSleeping
	case p.Biology.Vitals.Energy < minTrainingEnergy:
		return ErrTooTired
	}
	return nil
}

// trainingProgressLocked returns how much one session improves a trick:
// faster for intelligent pets and pets that like training, slower for
// harder tricks. Caller must hold p.mu.
func (p *DigitalPet) trainingProgressLocked(t Trick, reactionMultiplier float64) float64 {
	intelligence := p.Personality.Traits.Intelligence
	liking := p.Memory.GetLearnedAffinity(types.InteractionTraining)
	base := 0.15 + 0.25*intelligence + 0.1*liking
	// Young pets learn fastest; old age and lack of sleep slow learning
	cognition := p.Biology.Processes.CognitiveCapacity / 0.8
	if cognition <= 0 {
		cognition = 1
	}
	return max(base*reactionMultiplier*cognition/(1.0+0.3*float64(t.Difficulty-1)), 0.05)
}

// TrainTrick practices a trick with the pet
func (p *DigitalPet) TrainTrick(trickID string) (TrickResult, error) {
	return p.TrainTrickWith(trickID, "")
}

// TrainTrickWith trains a trick with a training method (see
// trainingMethods; "" trains without a particular method). Treats and
// clicker training use a treat from the pantry; clicker training also
// needs a clicker.
func (p *DigitalPet) TrainTrickWith(trickID, methodID string) (TrickResult, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	t, ok := findTrick(trickID)
	if !ok {
		return TrickResult{}, ErrUnknownTrick
	}
	if err := p.checkActiveLocked(); err != nil {
		return TrickResult{}, err
	}
	if reason := p.trickLockedReasonLocked(t); reason != "" && p.TrickSkills[t.ID] == 0 {
		return TrickResult{}, RefusedError{reason}
	}

	choice, speed := careChoice{}, 1.0
	if methodID != "" {
		m, ok := findMethod(methodID)
		if !ok {
			return TrickResult{}, ErrUnknownChoice
		}
		if err := p.methodAvailableLocked(m); err != nil {
			return TrickResult{}, err
		}
		choice.method, speed = &m, p.methodProgressLocked(m)
	}

	result := TrickResult{Trick: t}
	result.InteractionResult = p.interactWithLocked(types.InteractionTraining, choice)
	if !result.Applied {
		return TrickResult{}, RefusedError{result.Refusal}
	}
	switch methodID {
	case "treats", "clicker", "signals":
		p.useSupplyLocked(p.defLocked().Treats)
		p.countLocked("reward_training", 1)
	case "punishment":
		p.countLocked("scolding", 1)
	}

	if p.TrickSkills == nil {
		p.TrickSkills = make(map[string]float64)
	}
	before := p.TrickSkills[t.ID]
	speed *= p.effectLocked(effTraining)
	if p.TrainingBoost { // A freeze-dried treat was waiting
		speed *= 1.5
		p.TrainingBoost = false
	}
	result.Skill = min(before+p.trainingProgressLocked(t, result.ReactionMultiplier)*speed, 1.0)
	p.TrickSkills[t.ID] = result.Skill
	p.practiceTrickLocked(t.ID)
	if before >= 1.0 {
		p.markTrickLearnedLocked(t.ID) // Learned before learning was recorded
	}
	relearned := p.TricksLearned[t.ID] && before < 1.0
	result.JustLearned = before < 1.0 && result.Skill >= 1.0 && !relearned
	if result.Skill >= 1.0 {
		p.markTrickLearnedLocked(t.ID)
	}
	if result.JustLearned {
		p.weeklyEventLocked(GoalLearn)
	}
	p.goalEventLocked(GoalPractice, t.ID)

	switch {
	case relearned && result.Skill >= 1.0:
		result.Text = fmt.Sprintf("%s remembers %s again! %s", p.Name, t.Name, t.Icon)
	case relearned:
		result.Text = fmt.Sprintf("%s is brushing up on %s (%.0f%%).", p.Name, t.Name, result.Skill*100)
	case result.JustLearned:
		result.Text = fmt.Sprintf("%s learned %s! %s", p.Name, t.Name, t.Icon)
	case before >= 1.0:
		result.Text = fmt.Sprintf("%s practices %s. It already knows it well.", p.Name, t.Name)
	default:
		result.Text = fmt.Sprintf("%s is getting better at %s (%.0f%%).", p.Name, t.Name, result.Skill*100)
	}
	return result, nil
}

// PerformTrick asks the pet to show off a trick it has learned
func (p *DigitalPet) PerformTrick(trickID string) (TrickResult, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	t, ok := findTrick(trickID)
	if !ok {
		return TrickResult{}, ErrUnknownTrick
	}
	if p.TrickSkills[t.ID] < 1.0 {
		if p.TricksLearned[t.ID] {
			return TrickResult{}, ErrTrickRusty
		}
		return TrickResult{}, ErrTrickNotKnown
	}
	if err := p.checkActiveLocked(); err != nil {
		return TrickResult{}, err
	}

	result := TrickResult{Trick: t, Skill: 1.0}
	result.InteractionResult = p.interactLocked(types.InteractionTraining, 0.5)
	if !result.Applied {
		return TrickResult{}, RefusedError{result.Refusal}
	}
	p.practiceTrickLocked(t.ID)
	result.Text = fmt.Sprintf(t.Performed, p.Name)
	return result, nil
}

// ---------------------------------------------------------------------------
// Hide and seek

// HidingSpot is a place a pet can hide
type HidingSpot struct {
	Name string `json:"name"`
	Icon string `json:"icon"`
}

// hidingSpots are where a pet can hide in hide and seek
var hidingSpots = []HidingSpot{
	{"under the bed", "🛏️"},
	{"behind the couch", "🛋️"},
	{"in the toy box", "🧸"},
	{"behind the curtain", "🪟"},
	{"in the laundry basket", "🧺"},
	{"under the table", "🪑"},
}

// hideAndSeekGuesses is how many guesses the player gets
const hideAndSeekGuesses = 2

// GameBonusPoints is the bonus for finding the pet, earned at most once per
// CareRewardCooldown
const GameBonusPoints = 5

// hideAndSeek is a game in progress
type hideAndSeek struct {
	spot        int
	guessesLeft int
}

// GameResult reports a guess in hide and seek
type GameResult struct {
	InteractionResult
	Found       bool   // The guess found the pet
	Over        bool   // The game has ended
	GuessesLeft int    // Guesses remaining if not over
	Spot        string // Where the pet was hiding, once the game is over
	BonusPoints int    // Game bonus for finding the pet
	Text        string
}

// HidingSpots returns the places a pet can hide
func HidingSpots() []HidingSpot {
	return append([]HidingSpot(nil), hidingSpots...)
}

// StartHideAndSeek has the pet hide somewhere
func (p *DigitalPet) StartHideAndSeek() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if err := p.checkActiveLocked(); err != nil {
		return err
	}
	p.game = &hideAndSeek{spot: randIntn(len(hidingSpots)), guessesLeft: hideAndSeekGuesses}
	return nil
}

// QuitHideAndSeek ends a game early. The pet comes out, and the game earns
// nothing.
func (p *DigitalPet) QuitHideAndSeek() (string, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.game == nil {
		return "", ErrNoGame
	}
	spot := hidingSpots[p.game.spot]
	p.game = nil
	return fmt.Sprintf("%s comes out from %s. Maybe next time! %s", p.Name, spot.Name, spot.Icon), nil
}

// PlayingHideAndSeek reports whether a game is in progress
func (p *DigitalPet) PlayingHideAndSeek() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.game != nil
}

// GuessHidingSpot looks for the pet in a spot (an index into HidingSpots).
// Finding it counts as play and earns a game bonus.
func (p *DigitalPet) GuessHidingSpot(spot int) (GameResult, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.game == nil {
		return GameResult{}, ErrNoGame
	}
	if spot < 0 || spot >= len(hidingSpots) {
		return GameResult{}, ErrInvalidSpot
	}

	hiding := hidingSpots[p.game.spot]
	if spot == p.game.spot {
		p.game = nil
		result := GameResult{Found: true, Over: true, Spot: hiding.Name}
		result.InteractionResult = p.interactLocked(types.InteractionPlaying, 1.0)
		if !result.Applied {
			// The pet is too tired to carry on (it may have fallen asleep)
			result.Text = fmt.Sprintf("You found %s %s, but it's too tired to play. %s", p.Name, hiding.Name, hiding.Icon)
			return result, nil
		}
		if t := p.playNowLocked(); t.Sub(p.LastGameReward) >= CareRewardCooldown {
			p.LastGameReward = t
			p.FriendshipPoints += GameBonusPoints
			p.PointsEarned += GameBonusPoints
			result.BonusPoints = GameBonusPoints
		}
		result.Text = fmt.Sprintf("You found %s %s! %s", p.Name, hiding.Name, hiding.Icon)
		return result, nil
	}

	p.game.guessesLeft--
	if p.game.guessesLeft > 0 {
		return GameResult{GuessesLeft: p.game.guessesLeft,
			Text: fmt.Sprintf("%s isn't %s. Try again!", p.Name, hidingSpots[spot].Name)}, nil
	}

	// Out of guesses: the pet comes out, and still enjoyed the game a little
	p.game = nil
	result := GameResult{Over: true, Spot: hiding.Name}
	result.InteractionResult = p.interactLocked(types.InteractionPlaying, 0.5)
	result.Text = fmt.Sprintf("%s pops out from %s! %s", p.Name, hiding.Name, hiding.Icon)
	return result, nil
}

// ---------------------------------------------------------------------------
// Exploring

// junkFinds are worthless things a pet can bring home; "" means nothing
var junkFinds = []string{"a muddy stick", "a soggy leaf", "a pocketful of sand", ""}

// Trip outcomes
const (
	OutcomeTreasure = "treasure" // Found a souvenir for the collection
	OutcomeNothing  = "nothing"  // Found nothing of value
	OutcomeInjured  = "injured"  // Got hurt
)

// Adventure records a finished exploring trip
type Adventure struct {
	Outcome      string        `json:"outcome"`
	Place        string        `json:"place"`    // Where the pet went
	Souvenir     *Souvenir     `json:"souvenir"` // Only for OutcomeTreasure
	Text         string        `json:"text"`
	ReactionText string        `json:"reaction_text"`
	PointsEarned int           `json:"points_earned"`
	Changes      []StatChange  `json:"changes"`
	ReturnedAge  float64       `json:"returned_age"` // Pet's age when it came home
	Reaction     ReactionLevel `json:"reaction"`
	Discovered   string        `json:"discovered,omitempty"` // A place the trip opened up
	Extra        []string      `json:"extra,omitempty"`      // Other souvenirs found: on a long trip, or by a good choice after it
	Long         bool          `json:"long,omitempty"`       // A long trip
	Friend       string        `json:"friend,omitempty"`     // The name of a friend that came along
}

const (
	// exploreDuration is how long a trip takes, in game days of the pet's
	// age (the server runs one game day per real minute), and
	// longExploreDuration a long trip
	exploreDuration     = 2.0
	longExploreDuration = 5.0

	// A long trip adds longTripInjury percentage points of injury chance
	// and costs longTripEnergy more energy
	longTripInjury = 3
	longTripEnergy = 0.1

	// minExploreLevel is the energy, water and food a pet needs to set out
	minExploreLevel = 0.4

	// minFriendBond is how close two pets must be to go exploring together
	minFriendBond = 0.5
)

// Trip errors
var (
	ErrLongTrip   = errors.New("a long trip needs a grown-up pet, not a senior, that reliably comes when called")
	ErrTripFriend = errors.New("a friend must be the same species, close to the pet, and well enough to go")
)

// TripOptions are the choices for a trip
type TripOptions struct {
	Place      string      // A place the pet has discovered (see Places)
	Long       bool        // A long trip (see LongTripProblem)
	Friend     *DigitalPet // A saved pet that comes along, if any
	FriendSlot string      // Where the friend is saved, kept for when the trip ends
}

// TripFriend is a saved pet on a trip with the pet
type TripFriend struct {
	ID   types.PetID `json:"id"`
	Name string      `json:"name"`
	Slot string      `json:"slot"`
}

// SendExploring sends the pet out on a trip to the park. It returns home
// on its own after exploreDuration.
func (p *DigitalPet) SendExploring() error {
	return p.SendExploringWith(TripOptions{Place: defaultPlace})
}

// SendExploringTo sends the pet out on a trip to a place it has discovered
// (see Places). It returns home on its own after exploreDuration.
func (p *DigitalPet) SendExploringTo(place string) error {
	return p.SendExploringWith(TripOptions{Place: place})
}

// longTripProblemLocked says why the pet can't go on a long trip, or
// returns "". It must come reliably when called (the "come" trick learned
// and not rusty), and be neither young nor a senior. Caller must hold
// p.mu.
func (p *DigitalPet) longTripProblemLocked() string {
	switch stage := p.Biology.LifeStage(); {
	case stage == biology.StageYoung:
		return p.Name + " is too young for a long trip."
	case stage == biology.StageSenior:
		return p.Name + " is a senior now; short trips suit it better."
	case !p.TricksLearned["come"] || p.TrickSkills["come"] < 1:
		return p.Name + " must come reliably when called (the Come trick, learned and not rusty) to go on a long trip."
	}
	return ""
}

// LongTripProblem says why the pet can't go on a long trip, or returns ""
func (p *DigitalPet) LongTripProblem() string {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.longTripProblemLocked()
}

// tripReadyLocked returns an error if the pet can't set out now. Caller
// must hold p.mu.
func (p *DigitalPet) tripReadyLocked() error {
	if err := p.checkActiveLocked(); err != nil {
		return err
	}
	if p.Injured {
		return ErrInjured
	}
	v := p.Biology.Vitals
	if v.Energy < minExploreLevel || v.Hydration < minExploreLevel || v.Nutrition < minExploreLevel {
		return ErrNeedsCare
	}
	return nil
}

// friendProblemLocked says why friend can't come on the pet's trip, or
// returns "". Caller must hold both pets' locks.
func (p *DigitalPet) friendProblemLocked(friend *DigitalPet, long bool) string {
	rel, ok := p.Relationships.GetRelationship(friend.ID)
	switch {
	case friend.speciesLocked() != p.speciesLocked():
		return "Friends on a trip must be the same species."
	case !ok || rel.BondStrength < minFriendBond:
		return fmt.Sprintf("%s and %s need to be closer friends first.", p.Name, friend.Name)
	case friend.Illness != nil || friend.Injured || friend.Hibernating || !friend.Biology.IsAlive || friend.isExploringLocked():
		return friend.Name + " isn't well enough or isn't here."
	case friend.Biology.Vitals.Energy < minExploreLevel || friend.Biology.Vitals.Hydration < minExploreLevel ||
		friend.Biology.Vitals.Nutrition < minExploreLevel:
		return friend.Name + " needs food, water or rest first."
	case long:
		return friend.longTripProblemLocked()
	}
	return ""
}

// SendExploringWith sends the pet out on a trip. It returns home on its
// own after exploreDuration (longExploreDuration for a long trip). A
// friend that comes along shares the finds when the trip ends (see
// ShareTrip); until then it is unchanged.
func (p *DigitalPet) SendExploringWith(opts TripOptions) error {
	if opts.Friend == p {
		return ErrSamePet
	}
	if opts.Friend != nil {
		unlock := lockPair(p, opts.Friend)
		defer unlock()
	} else {
		p.mu.Lock()
		defer p.mu.Unlock()
	}

	place := opts.Place
	loc, ok := p.placeLocked(place)
	if !ok {
		return ErrUnknownPlace
	}
	if !p.discoveredLocked(place, loc) {
		return ErrPlaceLocked
	}
	if reason := p.placeSeasonLocked(place); reason != "" {
		return RefusedError{fmt.Sprintf("%s: %s.", loc.Name, strings.ToLower(reason))}
	}
	if err := p.tripReadyLocked(); err != nil {
		return err
	}
	if opts.Long {
		if reason := p.longTripProblemLocked(); reason != "" {
			return RefusedError{reason}
		}
	}
	if opts.Friend != nil {
		if reason := p.friendProblemLocked(opts.Friend, opts.Long); reason != "" {
			return RefusedError{reason}
		}
	}

	p.game = nil
	p.ExploringTo = place
	p.LongTrip = opts.Long
	duration := exploreDuration
	if opts.Long {
		duration = longExploreDuration
	}
	p.ExploringUntilAge = p.Biology.GetAgeInDays() + duration
	p.TripFriend = nil
	if opts.Friend != nil {
		p.TripFriend = &TripFriend{ID: opts.Friend.ID, Name: opts.Friend.Name, Slot: opts.FriendSlot}
	}
	p.CurrentBehavior = types.BehaviorExploring
	if p.PlaceVisits == nil {
		p.PlaceVisits = make(map[string]int)
	}
	p.PlaceVisits[place]++
	p.goalEventLocked(GoalExplore, place)
	return nil
}

// IsExploring reports whether the pet is out exploring
func (p *DigitalPet) IsExploring() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.isExploringLocked()
}

// ExploringRemaining returns how long (in game days) until the pet is home
func (p *DigitalPet) ExploringRemaining() float64 {
	p.mu.Lock()
	defer p.mu.Unlock()
	if !p.isExploringLocked() {
		return 0
	}
	return max(p.ExploringUntilAge-p.Biology.GetAgeInDays(), 0)
}

// isExploringLocked reports whether the pet is out exploring. Caller must
// hold p.mu.
func (p *DigitalPet) isExploringLocked() bool {
	return p.ExploringUntilAge > 0
}

// exploreOutcomeLocked decides how a trip went. Curious pets find
// souvenirs more often and adaptable pets get hurt less often; bad weather
// and night make trips riskier and less rewarding. Caller must hold p.mu.
func (p *DigitalPet) exploreOutcomeLocked() string {
	traits := p.Personality.Traits
	injuryChance := min(max(int(12+(0.5-traits.Adaptability)*10), 5), 20)
	treasureChance := min(max(int(55+(traits.Curiosity-0.5)*20), 40), 70)

	// Storms, extreme temperatures and darkness make trips worse
	extraInjury, treasureFactor := p.explorationConditionsLocked()
	injuryChance += extraInjury
	treasureChance = int(float64(treasureChance) * treasureFactor)

	// Wilder places are more dangerous and hold more, and a place others
	// have picked over holds less
	placeInjury, findFactor := p.placeOddsLocked()
	injuryChance = max(injuryChance+placeInjury+int(p.effectLocked(effTripInjury)), 0)
	if p.LongTrip {
		injuryChance += longTripInjury
	}
	treasureChance = min(int(float64(treasureChance)*findFactor), maxFindChance)

	roll := randIntn(100)
	switch {
	case roll < injuryChance:
		return OutcomeInjured
	case roll < injuryChance+treasureChance:
		return OutcomeTreasure
	default:
		return OutcomeNothing
	}
}

// finishExploringLocked brings the pet home. Caller must hold p.mu.
func (p *DigitalPet) finishExploringLocked() {
	p.ExploringUntilAge = 0
	place, _ := p.tripPlaceLocked()
	adventure := &Adventure{Outcome: p.exploreOutcomeLocked(), Place: place, Long: p.LongTrip}
	if p.TripFriend != nil {
		adventure.Friend = p.TripFriend.Name
	}
	where := p.placePhraseLocked()
	if p.speciesLocked() == SpeciesCat {
		where = "a harness walk at " + where // Cats explore only on a harness, with their owner
	}

	switch adventure.Outcome {
	case OutcomeInjured:
		adventure.Text = fmt.Sprintf("%s got hurt on %s and came home limping. 🩹 It needs to be healed.", p.Name, strings.TrimPrefix(where, "a "))
		adventure.Changes = p.injureLocked()
		adventure.Reaction = ReactionReluctant
	default:
		// The trip was fun and good exercise either way
		result := p.interactLocked(types.InteractionEnvironmentalEnrichment, 1.0)
		adventure.ReactionText = result.ReactionText
		adventure.Reaction = result.Reaction
		adventure.PointsEarned = result.PointsEarned
		adventure.Changes = result.Changes

		if adventure.Outcome == OutcomeTreasure {
			souvenir := p.pickSouvenirLocked(place)
			p.Souvenirs = append(p.Souvenirs, souvenir.ID)
			adventure.Souvenir = &souvenir
			adventure.Text = fmt.Sprintf("%s came home from %s with %s! %s", p.Name, where, souvenir.Name, souvenir.Icon)
		} else if junk := junkFinds[randIntn(len(junkFinds))]; junk != "" {
			adventure.Text = fmt.Sprintf("%s came home from %s with %s. Not much of a treasure!", p.Name, where, junk)
		} else {
			adventure.Text = fmt.Sprintf("%s came home from %s empty-handed.", p.Name, where)
		}

		// A long trip has a second chance of a find, and is more tiring
		if adventure.Long {
			if p.exploreOutcomeLocked() == OutcomeTreasure {
				souvenir := p.pickSouvenirLocked(place)
				p.Souvenirs = append(p.Souvenirs, souvenir.ID)
				adventure.Extra = append(adventure.Extra, souvenir.ID)
				adventure.Text += fmt.Sprintf(" On the long trip it also found %s! %s", souvenir.Name, souvenir.Icon)
			}
			p.Biology.Vitals.Energy -= longTripEnergy
			p.Biology.Vitals.Clamp()
		}
		if adventure.Friend != "" {
			adventure.Text += fmt.Sprintf(" %s came along and shares what they found.", adventure.Friend)
		}
	}
	if adventure.Outcome != OutcomeInjured {
		adventure.Text += p.visitPlaceLocked()
	}
	note := p.takeFromPlaceLocked(adventure.Outcome)
	adventure.Text += note
	if note != "" {
		adventure.Discovered = p.lastDiscoveredLocked()
	}

	adventure.ReturnedAge = p.Biology.GetAgeInDays()
	p.LastAdventure = adventure
	if adventure.Outcome != OutcomeInjured {
		p.startTripEventLocked(place)
	}
}

// injureLocked hurts the pet: it loses health and happiness, is frightened,
// and remembers exploring as a bad experience. It returns the stat
// changes. Caller must hold p.mu.
func (p *DigitalPet) injureLocked() []StatChange {
	v := p.Biology.Vitals
	before := *v
	v.Health -= 0.25
	v.Happiness -= 0.15
	v.Stress += 0.2 * p.effectLocked(effTripStress)
	v.Energy -= 0.15
	v.Clamp()

	stimulus := ai.EmotionalStimulus{FearDelta: 0.3, SadnessDelta: 0.15, JoyDelta: -0.1, Source: "injury"}
	p.Emotions.ApplyEmotionalStimulus(stimulus)
	valence := stimulus.Valence()
	p.Memory.RecordInteractionFull(types.InteractionEnvironmentalEnrichment, p.Biology.GetAgeInDays(), 1.0,
		p.Emotions.DominantEmotion, valence, math.Abs(valence))

	p.Injured = true
	p.evolveFromHardshipLocked(0.5) // A painful scare leaves the pet more anxious
	p.LastInteractionAge = p.Biology.GetAgeInDays()
	if !p.HibernationEnabled {
		p.Biology.CheckDeathConditions() // A severe injury can be fatal
	}
	if p.Biology.IsAlive {
		p.updateBehavior() // Hibernates if enabled and health is critical
	}
	return statChanges(before, *v)
}

// RecallFromExploring calls the pet home before its trip is over. It comes
// back with nothing: no souvenir, fun, points or learning.
func (p *DigitalPet) RecallFromExploring() (string, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.isExploringLocked() {
		return "", ErrNotExploring
	}
	p.ExploringUntilAge = 0
	p.TripFriend = nil // A friend comes home too, with nothing to share
	p.updateBehavior()
	return fmt.Sprintf("%s heads home early and brings nothing back.", p.Name), nil
}

// TripFriendToShare returns where the friend on the pet's last trip is
// saved, once the trip is over and until it has shared the finds
func (p *DigitalPet) TripFriendToShare() (string, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.TripFriend == nil || p.isExploringLocked() {
		return "", false
	}
	return p.TripFriend.Slot, true
}

// ClearTripFriend forgets the friend on the pet's last trip, when it can't
// be found to share the finds
func (p *DigitalPet) ClearTripFriend() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.TripFriend = nil
}

// ShareTrip gives the friend on the pet's last trip its share: a copy of
// each souvenir found, the trip's exercise and fun, and a closer bond
// with the pet (and the pet with it). Call it once the trip is over; the
// caller saves the friend. A friend that has died or gone away since gets
// nothing.
func ShareTrip(p, friend *DigitalPet) (string, error) {
	if p == friend {
		return "", ErrSamePet
	}
	unlock := lockPair(p, friend)
	defer unlock()
	tf, a := p.TripFriend, p.LastAdventure
	if tf == nil || tf.ID != friend.ID || p.isExploringLocked() {
		return "", ErrPetNotAvailable
	}
	p.TripFriend = nil
	if !friend.Biology.IsAlive || friend.Hibernating || friend.isExploringLocked() || a == nil {
		return "", ErrPetNotAvailable
	}

	shared := []string{}
	if a.Souvenir != nil {
		shared = append(shared, a.Souvenir.ID)
	}
	shared = append(shared, a.Extra...)
	friend.Souvenirs = append(friend.Souvenirs, shared...)
	if friend.PlaceVisits == nil {
		friend.PlaceVisits = make(map[string]int)
	}
	friend.PlaceVisits[a.Place]++

	v := friend.Biology.Vitals
	v.Energy -= 0.1
	if a.Long {
		v.Energy -= longTripEnergy
	}
	v.Happiness += 0.05
	v.Clamp()
	friend.ensureNeedsLocked()
	friend.Needs.SatisfyNeed(types.NeedExploration, 0.5, friend.Biology.GetAgeInDays())

	together := "explored " + placeDetails[a.Place].phrase + " together"
	for _, pair := range [][2]*DigitalPet{{p, friend}, {friend, p}} {
		rel, ok := pair[0].Relationships.GetRelationship(pair[1].ID)
		if !ok {
			rel = pair[0].Relationships.AddRelationship(pair[1].ID, types.RelationshipFriend)
		}
		rel.BondStrength += 0.05
		rel.AddSharedExperience(together, pair[0].Biology.GetAgeInDays(), 0.05)
		rel.Clamp()
	}
	return fmt.Sprintf("%s and %s %s.", p.Name, friend.Name, together), nil
}

// SouvenirCollection returns the souvenirs the pet has found, with how many
// of each, in catalog order
func (p *DigitalPet) SouvenirCollection() []SouvenirCount {
	p.mu.Lock()
	defer p.mu.Unlock()

	counts := make(map[string]int)
	for _, id := range p.Souvenirs {
		counts[id]++
	}
	collection := []SouvenirCount{}
	for _, s := range souvenirs {
		if counts[s.ID] > 0 {
			collection = append(collection, SouvenirCount{Souvenir: s, Count: counts[s.ID]})
		}
	}
	return collection
}

// SouvenirCount is a souvenir and how many the pet has found
type SouvenirCount struct {
	Souvenir
	Count int `json:"count"`
}
