package core

import (
	"errors"
	"fmt"
	"math"
	"math/rand"

	"github.com/Michael-W-Ellison/gochi/internal/ai"
	"github.com/Michael-W-Ellison/gochi/pkg/types"
)

// Activity errors
var (
	ErrUnknownTrick  = errors.New("unknown trick")
	ErrTrickNotKnown = errors.New("pet has not learned this trick yet")
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

// Trick is something a pet can be trained to do
type Trick struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Icon       string `json:"icon"`
	Difficulty int    `json:"difficulty"` // 1 (easiest) to 5
	Performed  string `json:"-"`          // What the pet does; %s is its name
}

// tricks lists every trick from easiest to hardest
var tricks = []Trick{
	{ID: "sit", Name: "Sit", Icon: "🐾", Difficulty: 1, Performed: "%s sits down neatly."},
	{ID: "paw", Name: "Shake a paw", Icon: "🤝", Difficulty: 2, Performed: "%s offers you a paw to shake."},
	{ID: "spin", Name: "Spin", Icon: "🌀", Difficulty: 3, Performed: "%s spins around in a happy circle!"},
	{ID: "roll_over", Name: "Roll over", Icon: "🔄", Difficulty: 4, Performed: "%s rolls all the way over!"},
	{ID: "play_dead", Name: "Play dead", Icon: "🎭", Difficulty: 5, Performed: "%s flops over dramatically... then peeks at you!"},
}

// minTrainingEnergy is the energy a pet needs to train or play games
const minTrainingEnergy = 0.25

// TrickProgress is a trick with the pet's skill at it
type TrickProgress struct {
	Trick
	Skill   float64 `json:"skill"` // 0.0 to 1.0
	Learned bool    `json:"learned"`
}

// TrickResult reports a training session or performance
type TrickResult struct {
	InteractionResult
	Trick       Trick
	Skill       float64 // Skill after training
	JustLearned bool    // The trick was learned in this session
	Text        string  // What happened, e.g. "Biscuit learned Spin!"
}

// findTrick returns the trick with the given ID
func findTrick(id string) (Trick, bool) {
	for _, t := range tricks {
		if t.ID == id {
			return t, true
		}
	}
	return Trick{}, false
}

// Tricks returns every trick with the pet's progress
func (p *DigitalPet) Tricks() []TrickProgress {
	p.mu.Lock()
	defer p.mu.Unlock()

	progress := make([]TrickProgress, len(tricks))
	for i, t := range tricks {
		skill := p.TrickSkills[t.ID]
		progress[i] = TrickProgress{Trick: t, Skill: skill, Learned: skill >= 1.0}
	}
	return progress
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
	return max(base*reactionMultiplier/(1.0+0.3*float64(t.Difficulty-1)), 0.05)
}

// TrainTrick practices a trick with the pet
func (p *DigitalPet) TrainTrick(trickID string) (TrickResult, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	t, ok := findTrick(trickID)
	if !ok {
		return TrickResult{}, ErrUnknownTrick
	}
	if err := p.checkActiveLocked(); err != nil {
		return TrickResult{}, err
	}

	result := TrickResult{Trick: t}
	result.InteractionResult = p.interactLocked(types.InteractionTraining, 1.0)
	if !result.Applied {
		return TrickResult{}, RefusedError{result.Refusal}
	}

	if p.TrickSkills == nil {
		p.TrickSkills = make(map[string]float64)
	}
	before := p.TrickSkills[t.ID]
	result.Skill = min(before+p.trainingProgressLocked(t, result.ReactionMultiplier), 1.0)
	p.TrickSkills[t.ID] = result.Skill
	result.JustLearned = before < 1.0 && result.Skill >= 1.0

	switch {
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
		if t := now(); t.Sub(p.LastGameReward) >= CareRewardCooldown {
			p.LastGameReward = t
			p.FriendshipPoints += GameBonusPoints
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

// Souvenir is something a pet can bring back from exploring
type Souvenir struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Icon string `json:"icon"`
	Rare bool   `json:"rare"`
}

// souvenirs a pet can find while exploring
var souvenirs = []Souvenir{
	{ID: "pebble", Name: "a shiny pebble", Icon: "🪨"},
	{ID: "feather", Name: "a feather", Icon: "🪶"},
	{ID: "flower", Name: "a flower", Icon: "🌸"},
	{ID: "leaf", Name: "a colorful leaf", Icon: "🍁"},
	{ID: "shell", Name: "a seashell", Icon: "🐚"},
	{ID: "acorn", Name: "an acorn", Icon: "🌰"},
	{ID: "crystal", Name: "a sparkly crystal", Icon: "💎", Rare: true},
}

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
	Souvenir     *Souvenir     `json:"souvenir"` // Only for OutcomeTreasure
	Text         string        `json:"text"`
	ReactionText string        `json:"reaction_text"`
	PointsEarned int           `json:"points_earned"`
	Changes      []StatChange  `json:"changes"`
	ReturnedAge  float64       `json:"returned_age"` // Pet's age when it came home
	Reaction     ReactionLevel `json:"reaction"`
}

const (
	// exploreDuration is how long a trip takes, in game days of the pet's
	// age (the server runs one game day per real minute)
	exploreDuration = 2.0

	// minExploreLevel is the energy, water and food a pet needs to set out
	minExploreLevel = 0.4
)

// SendExploring sends the pet out on a trip. It returns home on its own
// after exploreDuration with a souvenir.
func (p *DigitalPet) SendExploring() error {
	p.mu.Lock()
	defer p.mu.Unlock()

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

	p.game = nil
	p.ExploringUntilAge = p.Biology.GetAgeInDays() + exploreDuration
	p.CurrentBehavior = types.BehaviorExploring
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

// pickSouvenirLocked chooses what the pet found; curious pets find rare
// souvenirs more often. Caller must hold p.mu.
func (p *DigitalPet) pickSouvenirLocked() Souvenir {
	weights := make([]int, len(souvenirs))
	total := 0
	for i, s := range souvenirs {
		weights[i] = 3
		if s.Rare {
			weights[i] = 1 + int(2*p.Personality.Traits.Curiosity)
		}
		total += weights[i]
	}

	pick := randIntn(total)
	for i, w := range weights {
		if pick < w {
			return souvenirs[i]
		}
		pick -= w
	}
	return souvenirs[0]
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
	adventure := &Adventure{Outcome: p.exploreOutcomeLocked()}

	switch adventure.Outcome {
	case OutcomeInjured:
		adventure.Text = fmt.Sprintf("%s got hurt while exploring and came home limping. 🩹 It needs to be healed.", p.Name)
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
			souvenir := p.pickSouvenirLocked()
			p.Souvenirs = append(p.Souvenirs, souvenir.ID)
			adventure.Souvenir = &souvenir
			adventure.Text = fmt.Sprintf("%s came home from exploring with %s! %s", p.Name, souvenir.Name, souvenir.Icon)
		} else if junk := junkFinds[randIntn(len(junkFinds))]; junk != "" {
			adventure.Text = fmt.Sprintf("%s came home with %s. Not much of a treasure!", p.Name, junk)
		} else {
			adventure.Text = fmt.Sprintf("%s came home empty-handed.", p.Name)
		}
	}

	adventure.ReturnedAge = p.Biology.GetAgeInDays()
	p.LastAdventure = adventure
}

// injureLocked hurts the pet: it loses health and happiness, is frightened,
// and remembers exploring as a bad experience. It returns the stat
// changes. Caller must hold p.mu.
func (p *DigitalPet) injureLocked() []StatChange {
	v := p.Biology.Vitals
	before := *v
	v.Health -= 0.25
	v.Happiness -= 0.15
	v.Stress += 0.2
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
	p.updateBehavior()
	return fmt.Sprintf("%s heads home early and brings nothing back.", p.Name), nil
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
