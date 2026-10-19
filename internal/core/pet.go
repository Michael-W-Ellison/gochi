package core

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/Michael-W-Ellison/gochi/internal/ai"
	"github.com/Michael-W-Ellison/gochi/internal/biology"
	"github.com/Michael-W-Ellison/gochi/internal/simulation"
	"github.com/Michael-W-Ellison/gochi/internal/social"
	"github.com/Michael-W-Ellison/gochi/pkg/types"
)

// DigitalPet represents a complete digital pet with all its systems
type DigitalPet struct {
	// Identification
	ID   types.PetID `json:"id"`
	Name string      `json:"name"`

	// Core Systems
	Biology       *biology.BiologicalSystems   `json:"biology"`
	Personality   *ai.PersonalityMatrix        `json:"personality"`
	Memory        *ai.MemorySystem             `json:"memory"`
	Emotions      *ai.EmotionState             `json:"emotions"`
	Relationships *social.SocialRelationships  `json:"relationships"`
	Needs         *simulation.NeedsManager     `json:"needs"`

	// Current State
	CurrentBehavior types.BehaviorState `json:"current_behavior"`
	Location        string              `json:"location"`

	// Metadata
	CreatedAt    time.Time `json:"created_at"`
	LastUpdateAt time.Time `json:"last_update_at"`
	Owner        types.UserID `json:"owner"`

	// Statistics
	TotalInteractions int     `json:"total_interactions"`
	TotalPlayTime     float64 `json:"total_play_time"` // In hours

	// HibernationEnabled is the owner's choice of whether the pet hibernates
	// instead of dying when a vital stat reaches a critical level (off by
	// default). While Hibernating, the pet's stats are frozen and it ignores
	// interactions until it is revived.
	HibernationEnabled bool `json:"hibernation_enabled"`
	Hibernating        bool `json:"hibernating"`

	// Friendship Points earned by caring for this pet, the items bought
	// with them, and when each kind of care last earned points
	FriendshipPoints int                                 `json:"friendship_points"`
	Inventory        []string                            `json:"inventory"`
	LastCareReward   map[types.InteractionType]time.Time `json:"last_care_reward"`

	// Age (in days) at the last interaction, used to tell when the pet
	// has been left alone
	LastInteractionAge float64 `json:"last_interaction_age"`

	// Activities: trick skills (0.0 to 1.0, learned at 1.0), an exploring
	// trip that ends at ExploringUntilAge (0 when home), souvenirs found
	// while exploring, the most recent trip, and the last game bonus
	TrickSkills       map[string]float64 `json:"trick_skills"`
	ExploringUntilAge float64            `json:"exploring_until_age"`
	Souvenirs         []string           `json:"souvenirs"`
	LastAdventure     *Adventure         `json:"last_adventure"`
	LastGameReward    time.Time          `json:"last_game_reward"`
	Injured           bool               `json:"injured"` // Hurt while exploring; healed by medical care
	game              *hideAndSeek       // Game in progress (not saved)

	// mu guards concurrent updates, interactions and reads of pet state
	mu sync.Mutex
}

// NewDigitalPet creates a new digital pet with default systems
func NewDigitalPet(name string, owner types.UserID) *DigitalPet {
	now := time.Now()
	id := types.PetID(fmt.Sprintf("pet_%d_%s", now.Unix(), name))

	return &DigitalPet{
		ID:              id,
		Name:            name,
		Biology:         biology.NewBiologicalSystems(),
		Personality:     ai.NewPersonalityMatrix(),
		Memory:          ai.NewMemorySystem(100),
		Emotions:        ai.NewEmotionState(),
		Relationships:   social.NewSocialRelationships(20),
		Needs:           simulation.NewNeedsManager(),
		CurrentBehavior: types.BehaviorIdle,
		Location:        "home",
		CreatedAt:       now,
		LastUpdateAt:    now,
		Owner:           owner,
		TotalInteractions: 0,
		TotalPlayTime:     0,
	}
}

// NewDigitalPetRandom creates a pet with randomized personality
func NewDigitalPetRandom(name string, owner types.UserID) *DigitalPet {
	pet := NewDigitalPet(name, owner)
	pet.Personality = ai.NewPersonalityMatrixRandom()
	return pet
}

const (
	// criticalLevel is the level at which a vital stat that could cause
	// death sends the pet into hibernation (death occurs at 0.0)
	criticalLevel = 0.1

	// reviveLevel is the minimum level critical stats are restored to on revival
	reviveLevel = 0.5

	// maxBiologyStep is the largest time step the biology is advanced by at
	// once, so a long update cannot skip past the critical level (the
	// fastest stat, hydration, falls 0.05 per unit)
	maxBiologyStep = 0.05
)

// ErrNotHibernating is returned when reviving a pet that is not hibernating
var ErrNotHibernating = errors.New("pet is not hibernating")

// Update processes all system updates for a given time delta
func (p *DigitalPet) Update(deltaTime float64) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.Biology.IsAlive || p.Hibernating {
		return
	}
	if p.shouldHibernateLocked() {
		p.hibernateLocked()
		return
	}

	// Belongings such as bowls and beds slow the pet's needs
	p.Biology.DecayMultipliers = p.decayMultipliersLocked()

	// Update biological systems in small steps, stopping if the pet
	// hibernates or dies; the rest of the pet lives only that long
	elapsed := 0.0
	for elapsed < deltaTime && p.Biology.IsAlive {
		step := min(maxBiologyStep, deltaTime-elapsed)
		p.Biology.Update(step)
		elapsed += step
		if p.shouldHibernateLocked() {
			p.hibernateLocked()
			break
		}
	}
	deltaTime = elapsed

	if !p.Biology.IsAlive {
		// A pet that dies on a trip or mid-game does not come back; record
		// the time it lived
		p.ExploringUntilAge = 0
		p.game = nil
		p.LastUpdateAt = time.Now()
		p.TotalPlayTime += deltaTime / 60.0
		return
	}

	// Needs build up over time
	p.updateNeedsLocked(deltaTime)

	// Come home from exploring once the trip's time has passed
	if p.isExploringLocked() && p.Biology.GetAgeInDays() >= p.ExploringUntilAge {
		p.finishExploringLocked()
	}

	// Update emotions
	p.Emotions.Update(deltaTime)

	// Decay memories
	p.Memory.DecayMemories(deltaTime)

	// Consolidate strong memories before short-term decay erodes them
	p.Memory.ConsolidateMemories()

	// Update relationships
	p.Relationships.UpdateAll(deltaTime)

	// Update current behavior based on state
	p.updateBehavior()

	// Track time
	p.LastUpdateAt = time.Now()
	p.TotalPlayTime += deltaTime / 60.0 // Convert to hours
}

// shouldHibernateLocked reports whether the pet should hibernate now: its
// owner enabled hibernation and a stat is critical. Caller must hold p.mu.
func (p *DigitalPet) shouldHibernateLocked() bool {
	return p.HibernationEnabled && p.isCriticalLocked()
}

// SetHibernationEnabled sets whether the pet hibernates instead of dying.
// Turning it off does not wake a pet that is already hibernating.
func (p *DigitalPet) SetHibernationEnabled(enabled bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.HibernationEnabled = enabled
}

// isCriticalLocked reports whether a stat that could cause death has
// reached the critical level. Caller must hold p.mu.
func (p *DigitalPet) isCriticalLocked() bool {
	v := p.Biology.Vitals
	return v.Health <= criticalLevel ||
		v.Hydration <= criticalLevel ||
		(v.Nutrition <= criticalLevel && v.Energy <= criticalLevel)
}

// hibernateLocked puts the pet into hibernation. Caller must hold p.mu.
func (p *DigitalPet) hibernateLocked() {
	p.Hibernating = true
	p.CurrentBehavior = types.BehaviorHibernating
	p.ExploringUntilAge = 0 // A trip ends early so the pet is home to be revived
	p.game = nil
}

// IsHibernating reports whether the pet is hibernating
func (p *DigitalPet) IsHibernating() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.Hibernating
}

// Revive wakes a hibernating pet, restoring its critical stats to at least
// reviveLevel so it can be cared for again
func (p *DigitalPet) Revive() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.Biology.IsAlive {
		return errors.New("pet is not alive")
	}
	if !p.Hibernating {
		return ErrNotHibernating
	}

	v := p.Biology.Vitals
	v.Health = max(v.Health, reviveLevel)
	v.Hydration = max(v.Hydration, reviveLevel)
	v.Nutrition = max(v.Nutrition, reviveLevel)
	v.Energy = max(v.Energy, reviveLevel)
	v.Clamp()

	p.Hibernating = false
	p.updateBehavior()
	return nil
}

// ProcessUserInteraction handles user actions and applies effects
func (p *DigitalPet) ProcessUserInteraction(interactionType types.InteractionType, intensity float64) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.interactLocked(interactionType, intensity)
}

// interactLocked applies an interaction and awards Friendship Points.
// Caller must hold p.mu.
func (p *DigitalPet) interactLocked(interactionType types.InteractionType, intensity float64) InteractionResult {
	if !p.Biology.IsAlive || p.Hibernating || p.isExploringLocked() {
		return InteractionResult{}
	}

	p.TotalInteractions++
	p.LastInteractionAge = p.Biology.GetAgeInDays()
	if interactionType == types.InteractionMedicalCare {
		p.Injured = false
	}
	before := *p.Biology.Vitals

	// How much the pet enjoys this, from what it has learned, its
	// personality and its needs, scales the benefits
	result := InteractionResult{Applied: true}
	result.Reaction, result.ReactionMultiplier = p.reactionLocked(interactionType)
	result.ReactionText = reactionText(interactionType, result.Reaction, p.Name)
	boost := result.ReactionMultiplier

	// Belongings such as toys and brushes make care more effective
	if item, ok := p.careBoostLocked(interactionType); ok {
		boost *= item.Effect
		result.BoostedBy = &item
	}

	// Apply biological effects; the boost strengthens the benefits only
	p.applyInteractionEffects(interactionType, intensity, boost)
	intensity *= boost

	// Apply emotional effects
	stimulus := ai.CreateStimulusFromInteraction(interactionType, intensity)
	p.Emotions.ApplyEmotionalStimulus(stimulus)

	// Record memory; its valence also updates the learned preference
	// for this interaction type, which shapes future behavior
	valence := stimulus.Valence()
	p.Memory.RecordInteractionFull(interactionType, p.Biology.GetAgeInDays(), intensity, p.Emotions.DominantEmotion, valence, math.Abs(valence))

	p.meetNeedsLocked(interactionType, intensity)
	result.PointsEarned = p.awardCareLocked(interactionType)

	// Update behavior
	p.updateBehavior()

	result.Changes = statChanges(before, *p.Biology.Vitals)
	return result
}

// applyInteractionEffects modifies vital stats based on interaction type.
// boost (from the pet's reaction and its items) multiplies the benefits of
// an interaction but not its costs, such as the energy play uses.
func (p *DigitalPet) applyInteractionEffects(interactionType types.InteractionType, intensity, boost float64) {
	vitals := p.Biology.Vitals

	switch interactionType {
	case types.InteractionFeeding:
		vitals.Nutrition += 0.3 * intensity * boost
		vitals.Energy += 0.1 * intensity * boost
		vitals.Happiness += 0.05 * intensity * boost

	case types.InteractionPetting:
		vitals.Happiness += 0.15 * intensity * boost
		vitals.Stress -= 0.1 * intensity * boost

	case types.InteractionPlaying:
		vitals.Happiness += 0.2 * intensity * boost
		vitals.Energy -= 0.1 * intensity
		vitals.Stress -= 0.05 * intensity * boost

	case types.InteractionDrinking:
		vitals.Hydration += 0.4 * intensity * boost
		vitals.Happiness += 0.02 * intensity * boost

	case types.InteractionTraining:
		vitals.Happiness += 0.05 * intensity * boost
		vitals.Energy -= 0.05 * intensity

	case types.InteractionEnvironmentalEnrichment:
		vitals.Happiness += 0.15 * intensity * boost
		vitals.Stress -= 0.05 * intensity * boost
		vitals.Energy -= 0.15 * intensity

	case types.InteractionGrooming:
		vitals.Cleanliness += 0.3 * intensity * boost
		vitals.Happiness += 0.05 * intensity * boost

	case types.InteractionMedicalCare:
		vitals.Health += 0.2 * intensity * boost
		vitals.Stress += 0.05 * intensity // Medical care can be stressful

	case types.InteractionRewards:
		vitals.Happiness += 0.25 * intensity * boost

	case types.InteractionDiscipline:
		vitals.Stress += 0.15 * intensity
		vitals.Happiness -= 0.1 * intensity
	}

	vitals.Clamp()
}

// updateBehavior determines the current behavior based on state
func (p *DigitalPet) updateBehavior() {
	vitals := p.Biology.Vitals

	// An interaction (such as tiring play) can leave a stat critical
	if p.Hibernating || p.shouldHibernateLocked() {
		p.hibernateLocked()
		return
	}
	if p.isExploringLocked() {
		p.CurrentBehavior = types.BehaviorExploring
		return
	}

	// Critical needs override other behaviors
	if vitals.Health < 0.3 {
		p.CurrentBehavior = types.BehaviorSick
		return
	}

	if vitals.GetOverallWellbeing() < 0.3 {
		p.CurrentBehavior = types.BehaviorDistressed
		return
	}

	if vitals.Fatigue > 0.7 || vitals.Energy < 0.3 {
		p.CurrentBehavior = types.BehaviorSleeping
		return
	}

	// Use emotions and personality to determine behavior
	moodScore := p.Emotions.GetMoodScore()

	if moodScore > 0.6 && vitals.Energy > 0.5 {
		if p.Emotions.Excitement > 0.7 {
			p.CurrentBehavior = types.BehaviorExcited
		} else {
			p.CurrentBehavior = types.BehaviorHappy
		}
	} else if vitals.Nutrition < 0.3 {
		p.CurrentBehavior = types.BehaviorEating
	} else if vitals.Energy > 0.4 {
		p.CurrentBehavior = p.chooseDiscretionaryBehavior()
	} else {
		p.CurrentBehavior = types.BehaviorIdle
	}
}

const (
	// learnedBehaviorWeight scales how far learned preferences shift a
	// personality-based tendency (tendencies and the threshold are 0.0-1.0)
	learnedBehaviorWeight = 0.3

	// discretionaryThreshold is the tendency a behavior needs to be chosen
	discretionaryThreshold = 0.6
)

// chooseDiscretionaryBehavior picks the strongest optional behavior from
// personality plus what the pet has learned from past interactions.
// It is deterministic, so behavior stays consistent until interactions
// gradually change the learned preferences.
func (p *DigitalPet) chooseDiscretionaryBehavior() types.BehaviorState {
	learned := p.Memory.GetLearnedAffinity

	candidates := []struct {
		behavior types.BehaviorState
		score    float64
	}{
		{
			// Enjoying play makes a pet more playful
			types.BehaviorPlaying,
			p.Personality.GetTraitInfluence("play") + learnedBehaviorWeight*learned(types.InteractionPlaying),
		},
		{
			// Enjoying petting draws a pet toward its owner; discipline
			// (always negative) makes it more withdrawn
			types.BehaviorSocialInteraction,
			p.Personality.GetTraitInfluence("social") +
				learnedBehaviorWeight*(learned(types.InteractionPetting)+learned(types.InteractionDiscipline)),
		},
		{
			// Enjoying exploring trips makes a pet more adventurous
			types.BehaviorExploring,
			p.Personality.GetTraitInfluence("explore") + learnedBehaviorWeight*learned(types.InteractionEnvironmentalEnrichment),
		},
	}

	best := types.BehaviorIdle
	bestScore := discretionaryThreshold
	for _, c := range candidates {
		if c.score > bestScore {
			best, bestScore = c.behavior, c.score
		}
	}
	return best
}

// GetCurrentStatus returns a comprehensive status report
func (p *DigitalPet) GetCurrentStatus() PetStatus {
	p.mu.Lock()
	defer p.mu.Unlock()

	return PetStatus{
		PetID:            p.ID,
		Name:             p.Name,
		IsAlive:          p.Biology.IsAlive,
		Hibernating:      p.Hibernating,
		Age:              p.Biology.GetAgeInDays(),
		Health:           p.Biology.Vitals.Health,
		Energy:           p.Biology.Vitals.Energy,
		Happiness:        p.Biology.Vitals.Happiness,
		Wellbeing:        p.Biology.Vitals.GetOverallWellbeing(),
		CurrentBehavior:  p.CurrentBehavior,
		MoodDescription:  p.Emotions.GetMoodDescription(),
		StatusDescription: p.Biology.GetStatus(),
		CriticalNeeds:    p.Biology.Vitals.GetCriticalStats(0.3),
	}
}

// PetStatus represents a snapshot of the pet's current state
type PetStatus struct {
	PetID             types.PetID
	Name              string
	IsAlive           bool
	Hibernating       bool
	Age               float64
	Health            float64
	Energy            float64
	Happiness         float64
	Wellbeing         float64
	CurrentBehavior   types.BehaviorState
	MoodDescription   string
	StatusDescription string
	CriticalNeeds     []string
}

// String provides a human-readable status report
func (s PetStatus) String() string {
	status := fmt.Sprintf("=== %s (Age: %.1f days) ===\n", s.Name, s.Age)
	status += fmt.Sprintf("Status: %s | Mood: %s\n", s.StatusDescription, s.MoodDescription)
	status += fmt.Sprintf("Behavior: %s\n", s.CurrentBehavior.String())
	if s.Hibernating {
		status += "💤 Hibernating - revive to continue\n"
	}
	status += fmt.Sprintf("Health: %.0f%% | Energy: %.0f%% | Happiness: %.0f%%\n",
		s.Health*100, s.Energy*100, s.Happiness*100)
	status += fmt.Sprintf("Overall Wellbeing: %.0f%%\n", s.Wellbeing*100)

	if len(s.CriticalNeeds) > 0 {
		status += fmt.Sprintf("⚠️  Critical Needs: %v\n", s.CriticalNeeds)
	}

	return status
}

// Save serializes the pet to JSON
func (p *DigitalPet) Save() ([]byte, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	return json.MarshalIndent(p, "", "  ")
}

// Load deserializes a pet from JSON
func Load(data []byte) (*DigitalPet, error) {
	var pet DigitalPet
	err := json.Unmarshal(data, &pet)
	if err != nil {
		return nil, err
	}
	return &pet, nil
}

// GetPersonalityDescription returns a description of the pet's personality
func (p *DigitalPet) GetPersonalityDescription() string {
	return p.Personality.GetPersonalityDescription()
}

// GetAge returns the pet's age in days
func (p *DigitalPet) GetAge() float64 {
	return p.Biology.GetAgeInDays()
}

// IsAlive returns whether the pet is alive
func (p *DigitalPet) IsAlive() bool {
	return p.Biology.IsAlive
}

// GetName returns the pet's name
func (p *DigitalPet) GetName() string {
	return p.Name
}
