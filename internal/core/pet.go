package core

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"reflect"
	"strings"
	"sync"
	"time"

	"github.com/Michael-W-Ellison/gochi/internal/ai"
	"github.com/Michael-W-Ellison/gochi/internal/biology"
	"github.com/Michael-W-Ellison/gochi/internal/environment"
	"github.com/Michael-W-Ellison/gochi/internal/genetics"
	"github.com/Michael-W-Ellison/gochi/internal/interaction"
	"github.com/Michael-W-Ellison/gochi/internal/simulation"
	"github.com/Michael-W-Ellison/gochi/internal/social"
	"github.com/Michael-W-Ellison/gochi/pkg/types"
)

// DigitalPet represents a complete digital pet with all its systems
type DigitalPet struct {
	// Identification
	ID      types.PetID `json:"id"`
	Name    string      `json:"name"`
	Species Species     `json:"species"` // "" in saves made before species existed (a cat)

	// Core Systems
	Biology       *biology.BiologicalSystems  `json:"biology"`
	Personality   *ai.PersonalityMatrix       `json:"personality"`
	Memory        *ai.MemorySystem            `json:"memory"`
	Emotions      *ai.EmotionState            `json:"emotions"`
	Relationships *social.SocialRelationships `json:"relationships"`
	Needs         *simulation.NeedsManager    `json:"needs"`
	Environment   *environment.Environment    `json:"environment"`
	Genome        *genetics.Genome            `json:"genome"`

	// Current State
	CurrentBehavior types.BehaviorState `json:"current_behavior"`
	Location        string              `json:"location"`

	// Metadata
	CreatedAt    time.Time    `json:"created_at"`
	LastUpdateAt time.Time    `json:"last_update_at"`
	Owner        types.UserID `json:"owner"`

	// Statistics
	TotalInteractions int            `json:"total_interactions"`
	TotalPlayTime     float64        `json:"total_play_time"` // In hours
	CareCounts        map[string]int `json:"care_counts"`     // Interactions by kind, e.g. "Feeding" (records.go)
	Hibernations      int            `json:"hibernations"`    // Times it has gone into hibernation

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
	PointsEarned     int                                 `json:"points_earned"` // Over its whole life, which unlocks rarer supplies
	Supplies         *interaction.Inventory              `json:"supplies"`      // Pantry of food, treats and toys

	// Upkeep: when (pet age, game days) each upkeep timer runs out, such as
	// a fountain filter or flea collar (upkeep.go); first-aid kit uses left;
	// the calendar day each once-a-day supply was last given; the day the
	// living area was last cleaned; and whether freeze-dried treats are
	// waiting to make the next training count more
	Upkeep         map[string]float64 `json:"upkeep,omitempty"`
	FirstAidUses   int                `json:"first_aid_uses,omitempty"`
	DailyUse       map[string]string  `json:"daily_use,omitempty"`
	LastCleanedDay string             `json:"last_cleaned_day,omitempty"`
	TrainingBoost  bool               `json:"training_boost,omitempty"`

	// How the pet looks: cosmetics owned, and what it wears in each slot
	// (cosmetics.go)
	Cosmetics []string          `json:"cosmetics,omitempty"`
	Outfit    map[string]string `json:"outfit,omitempty"`

	// Family (family.go): the pet's parents, Friendship Points given by a
	// parent (spent like any, but not counted towards unlocking rarer
	// items), the child its belongings were passed on to after it died,
	// and whether the player was warned that it is about to grow up
	Parents      []types.PetID `json:"parents,omitempty"`
	GiftedPoints int           `json:"gifted_points,omitempty"`
	PassedOnTo   types.PetID   `json:"passed_on_to,omitempty"`
	AdultWarned  bool          `json:"adult_warned,omitempty"`

	// Breed (breeds.go): the breed, how the pet looks, the tendencies it
	// has (by how strongly), its lifespan relative to its species, and for
	// a mixed breed where it comes from
	Breed         string             `json:"breed,omitempty"`
	Look          *PetLook           `json:"look,omitempty"`
	Tendencies    map[string]float64 `json:"tendencies,omitempty"`
	BreedLifespan float64            `json:"breed_lifespan,omitempty"`
	Lineage       *Lineage           `json:"lineage,omitempty"`

	// Care records for care achievements: days all of the day's goals
	// were finished, the last of them, and illnesses treated
	GoalDays int `json:"goal_days,omitempty"`
	// Goal variety: the last few days' activity goal kinds, the last
	// wellbeing goal, the week's goal, and trips to each place
	RecentGoalKinds   []string       `json:"recent_goal_kinds,omitempty"`
	LastWellbeingGoal string         `json:"last_wellbeing_goal,omitempty"`
	Weekly            *WeeklyGoal    `json:"weekly,omitempty"`
	PlaceVisits       map[string]int `json:"place_visits,omitempty"`

	// Care records for achievements (carerecords.go): counts and streaks,
	// and marks such as the last day something was counted
	Counters map[string]float64 `json:"counters,omitempty"`
	Marks    map[string]string  `json:"marks,omitempty"`

	// Shows (shows.go): ribbons won, titles, and its age at its last show
	Ribbons          []Ribbon `json:"ribbons,omitempty"`
	Titles           []string `json:"titles,omitempty"`
	LastShowAge      float64  `json:"last_show_age,omitempty"`
	LastGoalDay      string   `json:"last_goal_day,omitempty"`
	IllnessesTreated int      `json:"illnesses_treated,omitempty"`

	// Home happenings (happenings.go): the one waiting for a choice, when
	// the last came on the play clock, and when one was last tried for
	// (not saved)
	Happening        *PendingHappening `json:"happening,omitempty"`
	LastHappening    time.Time         `json:"last_happening,omitempty"`
	happeningChecked time.Time

	// Kept as it is while it isn't the active pet, instead of living on
	// with the household (householdtime.go)
	HouseholdPaused bool `json:"household_paused,omitempty"`

	// The pages of its book (keepsakes.go)
	Keepsakes []Keepsake `json:"keepsakes,omitempty"`

	// Age (in days) at the last interaction, used to tell when the pet
	// has been left alone
	LastInteractionAge float64 `json:"last_interaction_age"`

	// The play clock (playclock.go), and the reading it was last advanced
	// from (not saved)
	PlayClock time.Time `json:"play_clock"`
	playRef   time.Time

	// Activities: trick skills (0.0 to 1.0, learned at 1.0), an exploring
	// trip that ends at ExploringUntilAge (0 when home), souvenirs found
	// while exploring, the most recent trip, and the last game bonus
	TrickSkills       map[string]float64   `json:"trick_skills"`
	TrickPracticedAge map[string]float64   `json:"trick_practiced_age"` // Age each trick was last trained or performed
	TricksLearned     map[string]bool      `json:"tricks_learned"`      // Tricks the pet has ever learned
	ExploringUntilAge float64              `json:"exploring_until_age"`
	ExploringTo       string               `json:"exploring_to"`          // Where the current or last trip went
	LongTrip          bool                 `json:"long_trip,omitempty"`   // The current or last trip is a long one
	TripFriend        *TripFriend          `json:"trip_friend,omitempty"` // A friend on the current trip, until it shares the finds
	ParasiteExposure  float64              `json:"parasite_exposure"`     // Builds with trips to risky places (places.go)
	TripsTaken        int                  `json:"trips_taken"`           // Trips finished, which discover new places
	Souvenirs         []string             `json:"souvenirs"`
	LastAdventure     *Adventure           `json:"last_adventure"`
	LastGameReward    time.Time            `json:"last_game_reward"`
	Injured           bool                 `json:"injured"`       // Hurt while exploring; healed by medical care
	History           []HistoryEntry       `json:"history"`       // What happened to the pet, oldest first (records.go)
	Goals             *GoalBoard           `json:"goals"`         // Today's adaptive goals (goals.go)
	Achievements      map[string]time.Time `json:"achievements"`  // Achievements earned, by ID, with when
	LastBredAge       float64              `json:"last_bred_age"` // Age at the last breeding (0 if never)
	BabiesWith        map[types.PetID]int  `json:"babies_with"`   // Babies had with each mate

	// Health care: how dirty the living area is (0.0 clean to 1.0
	// filthy), the illness the pet has (nil if well), recent health
	// events, and today's care checklist
	LivingAreaWaste float64        `json:"living_area_waste"`
	Illness         *Illness       `json:"illness"`
	HealthEvents    []HealthEvent  `json:"health_events"`
	Checklist       *CareChecklist `json:"checklist"`
	game            *hideAndSeek   // Game in progress (not saved)

	// What the pet has learned about how its owner responds to its
	// behaviors (reinforcement learning), and the state machine that
	// chooses its behavior (built when needed, not saved)
	Learning       *ai.LearningSystem                    `json:"learning"`
	RecentCare     map[types.InteractionType]*RecentCare `json:"recent_care"`     // Repeated care, which counts less
	EnrichmentUsed map[string]float64                    `json:"enrichment_used"` // Age each environment change was last made
	KnownPets      map[types.PetID]string                `json:"known_pets"`      // Names of the pets it has met or is related to
	choice         *careChoice                           // Variant of the care being given (not saved)
	behavior       *ai.BehaviorStateMachine

	// completedGoals are goals completed since TakeCompletedGoals was last
	// called (not saved)
	completedGoals []Goal

	// What UpcomingNeeds last showed, and the care given by then (not saved)
	shownNeeds []shownNeed
	shownAfter careGiven

	// mu guards concurrent updates, interactions and reads of pet state
	mu sync.Mutex
}

// NewDigitalPet creates a new digital pet with default systems
func NewDigitalPet(name string, owner types.UserID) *DigitalPet {
	now := time.Now()
	id := newPetID()

	pet := &DigitalPet{
		ID:                id,
		Name:              name,
		Biology:           biology.NewBiologicalSystems(),
		Personality:       ai.NewPersonalityMatrix(),
		Memory:            ai.NewMemorySystem(100),
		Emotions:          ai.NewEmotionState(),
		Relationships:     social.NewSocialRelationships(20),
		Needs:             simulation.NewNeedsManager(),
		Environment:       environment.NewEnvironment(),
		CurrentBehavior:   types.BehaviorIdle,
		Location:          "home",
		CreatedAt:         now,
		LastUpdateAt:      now,
		Owner:             owner,
		TotalInteractions: 0,
		TotalPlayTime:     0,
	}
	pet.Memory.SetLongTermCapacity(maxLongTermMemories)
	pet.ensureGenomeLocked()
	pet.keepLocked(KeepsakeHome, "") // A new pet; it owns itself alone
	return pet
}

// NewDigitalPetRandom creates a pet with randomized personality
func NewDigitalPetRandom(name string, owner types.UserID) *DigitalPet {
	pet := NewDigitalPet(name, owner)
	pet.Personality = ai.NewPersonalityMatrixRandom()
	pet.Genome = nil
	pet.ensureGenomeLocked() // Match the genome to the new personality
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

	stage := p.Biology.LifeStage()
	defer p.keepLifeStageLocked(stage)

	// Belongings such as bowls and beds slow the pet's needs, and the body
	// responds to its surroundings, needs and age
	p.Biology.DecayMultipliers = p.decayMultipliersLocked()
	p.prepareBodyLocked()

	// Update biological systems in small steps, stopping if the pet
	// hibernates or dies; the rest of the pet lives only that long
	elapsed := 0.0
	for elapsed < deltaTime && p.Biology.IsAlive {
		step := min(maxBiologyStep, deltaTime-elapsed)
		p.updateHealthLocked(step)
		p.Biology.Update(step)
		elapsed += step
		if p.shouldHibernateLocked() {
			p.hibernateLocked()
			break
		}
	}
	deltaTime = elapsed

	if !p.Biology.IsAlive {
		if name := p.illnessNameLocked(); name != "" && p.Biology.CauseOfDeath != causeOldAge {
			p.Biology.CauseOfDeath = "Untreated " + strings.ToLower(name)
		}
		// A pet that dies on a trip or mid-game does not come back; record
		// the time it lived
		p.ExploringUntilAge = 0
		p.game = nil
		p.LastUpdateAt = time.Now()
		p.TotalPlayTime += deltaTime / 60.0
		return
	}

	// A sleeping pet rests and recovers
	if p.CurrentBehavior == types.BehaviorSleeping {
		p.Biology.ProcessRest(deltaTime, p.sleepQualityLocked())
	}

	// Tricks the pet hasn't practiced in a while get rusty
	p.decayTricksLocked(deltaTime)

	// Grooming itself and running around do the pet some good, and so do
	// things to watch and play with on its own
	p.selfCareLocked(deltaTime)
	p.idleEnrichmentLocked(deltaTime)

	// Needs build up over time, and the weather and season affect the pet
	p.updateNeedsLocked(deltaTime)
	p.updateEnvironmentLocked(deltaTime)

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
	p.updateRelationshipsLocked(deltaTime)

	// Goals kept up over time, and records for care achievements
	p.updateTimedGoalsLocked()
	p.updateCareRecordsLocked(deltaTime)
	p.keepMomentsLocked()

	// Loneliness and neglect change the pet's personality
	p.evolveFromNeglectLocked(deltaTime)

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
	if !p.Hibernating {
		p.Hibernations++
	}
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
	// Care is given at an intensity from 0 to 1
	if !(intensity >= 0) { // Also NaN
		intensity = 0
	}
	intensity = min(intensity, 1)
	// The living area can be cleaned while the pet is out exploring
	away := p.isExploringLocked() && interactionType != types.InteractionCleaning
	if !p.Biology.IsAlive || p.Hibernating || away {
		return InteractionResult{}
	}
	p.Biology.EnsurePhysiology()
	if refusal := p.careRefusalLocked(interactionType); refusal != "" {
		return InteractionResult{Refusal: refusal, Suggestion: p.suggestionLocked()}
	}
	behaviorBefore, contextBefore := p.CurrentBehavior, p.learningContextLocked()

	// The interaction rules can turn care down (a full pet is not hungry,
	// a sleeping pet won't play); nothing happens and nothing is earned
	if ok, reason := interaction.ValidateInteraction(interactionType, p.interactionContextLocked()); !ok {
		return InteractionResult{Refusal: strings.Replace(reason, "Pet", p.Name, 1) + ".", Suggestion: p.suggestionLocked()}
	}

	p.TotalInteractions++
	p.countCareLocked(interactionType)
	p.cleanedLocked(interactionType)
	p.LastInteractionAge = p.Biology.GetAgeInDays()
	treated := ""
	if interactionType == types.InteractionMedicalCare {
		p.Injured = false
		treated = p.treatIllnessLocked()
		p.giveFluidsLocked()
	}
	before := *p.Biology.Vitals

	// How much the pet enjoys this, from what it has learned, its
	// personality and its needs, scales the benefits
	result := InteractionResult{Applied: true, ReactionMultiplier: 1}
	involved := interactionType != types.InteractionCleaning // The pet isn't part of cleaning
	innate, weight := 0.0, 0.0
	if involved {
		result.Reaction, result.ReactionMultiplier = p.reactionLocked(interactionType)
		result.ReactionText = reactionText(p.speciesLocked(), interactionType, result.Reaction, p.Name)
		innate = p.innateReactionLocked(interactionType)
		weight = p.repeatWeightLocked(interactionType) // After the reaction, which counts earlier repeats
	}
	// Attention a pet doesn't want stresses it instead of pleasing it
	unwanted := attentionCare[interactionType] && result.Reaction == ReactionReluctant
	boost := result.ReactionMultiplier

	// Belongings such as toys and brushes make care more effective
	if extra, item := p.careBoostLocked(interactionType); extra != 1 {
		boost *= extra
		result.BoostedBy = item
	}

	// Apply biological effects; the boost strengthens the benefits only
	p.applyInteractionEffects(interactionType, intensity, boost)
	p.careStimulationLocked(interactionType)
	p.careHormonesLocked(interactionType, boost, unwanted)
	choiceLesson := p.applyChoiceLocked(boost) // The variant chosen, such as the food or toy
	intensity *= boost

	// Apply emotional effects
	stimulus := ai.CreateStimulusFromInteraction(interactionType, intensity)
	if unwanted {
		p.Biology.Vitals.Stress += 0.1
		p.Biology.Vitals.Happiness -= 0.12
		p.Biology.Vitals.Clamp()
		stimulus = ai.EmotionalStimulus{AngerDelta: 0.1, FearDelta: 0.05, ContentmentDelta: -0.05,
			Source: interactionType.String()}
	}
	p.Emotions.ApplyEmotionalStimulus(stimulus)

	// Record memory. The learned preference for this kind of care comes
	// from how the pet itself took it (its nature and state, not what it
	// already learned), with care repeated in quick succession counting
	// less; it shapes future reactions and behavior
	valence := stimulus.Valence()
	p.Memory.RecordInteractionExperience(interactionType, p.Biology.GetAgeInDays(), intensity, p.Emotions.DominantEmotion,
		valence, math.Abs(valence), experiencedValence(innate), weight)

	// The pet learns how its owner responds to what it was doing, and its
	// personality slowly changes with how it is treated (cleaning the
	// living area doesn't involve the pet)
	if involved {
		p.learnFromResponseLocked(behaviorBefore, contextBefore, valence*weight)
		p.recordExperienceLocked(interactionType, contextBefore, valence, weight)
		p.evolveFromCareLocked(interactionType, valence, weight)
		if weight < diminishedWeight {
			result.Repeated = fmt.Sprintf("%s has had plenty of this just now; more right away won't change how %s grows.",
				p.Name, p.Name)
		}
	}

	p.meetNeedsLocked(interactionType, intensity)
	result.PointsEarned = p.awardCareLocked(interactionType)
	p.checkOffLocked(interactionType)
	result.Lesson = p.lessonLocked(interactionType, treated)
	// The most specific lesson: about the illness treated, then about the
	// food, toy, spot or method chosen, then about unwanted attention
	switch {
	case treated != "":
	case choiceLesson != "":
		result.Lesson = choiceLesson
	case unwanted:
		result.Lesson = p.defLocked().UnwantedAttentionLesson
	}

	// Update behavior
	p.updateBehavior()

	result.Changes = statChanges(before, *p.Biology.Vitals)
	description, outcome := careHistoryText(interactionType, result)
	p.addHistoryLocked(interactionType.String(), description, outcome)
	p.progressGoalsLocked(interactionType)
	if result.Reaction == ReactionReluctant && interactionType != types.InteractionDiscipline { // Never enjoyed
		result.Suggestion = p.suggestionLocked()
	}
	return result
}

// applyInteractionEffects modifies vital stats based on interaction type.
// boost (from the pet's reaction and its items) multiplies the benefits of
// an interaction but not its costs, such as the energy play uses.
func (p *DigitalPet) applyInteractionEffects(interactionType types.InteractionType, intensity, boost float64) {
	vitals := p.Biology.Vitals

	switch interactionType {
	case types.InteractionFeeding:
		portion := 1.0 // A meal's share of a normal meal of basic food
		if p.choice != nil && p.choice.food != nil {
			portion = p.choice.food.food.nutrition
		}
		vitals.Energy += 0.1 * intensity * boost
		vitals.Happiness += 0.05 * intensity * boost
		p.digestLocked(biology.FoodItem{Bulk: 0.3 * portion * intensity, Nutrition: 0.3 * portion * intensity * boost})

	case types.InteractionPetting:
		vitals.Happiness += 0.15 * intensity * boost
		vitals.Stress -= 0.1 * intensity * boost

	case types.InteractionPlaying:
		vitals.Happiness += 0.2 * intensity * boost
		vitals.Energy -= 0.1 * intensity
		vitals.Stress -= 0.05 * intensity * boost
		p.Biology.ProcessExercise(biology.ExercisePlay, intensity)

	case types.InteractionDrinking:
		vitals.Hydration += 0.4 * intensity * boost
		vitals.Happiness += 0.02 * intensity * boost
		p.Biology.Drink(waterToBladder * intensity)

	case types.InteractionTraining:
		vitals.Happiness += 0.05 * intensity * boost
		vitals.Energy -= 0.05 * intensity

	case types.InteractionEnvironmentalEnrichment:
		vitals.Happiness += 0.15 * intensity * boost
		vitals.Stress -= 0.05 * intensity * boost
		vitals.Energy -= 0.15 * intensity
		p.Biology.ProcessExercise(biology.ExerciseExplore, 1.5*intensity)

	case types.InteractionGrooming:
		vitals.Cleanliness += 0.3 * intensity * boost
		vitals.Happiness += 0.05 * intensity * boost

	case types.InteractionMedicalCare:
		vitals.Health += 0.2 * intensity * boost
		vitals.Stress += 0.05 * intensity // Medical care can be stressful

	case types.InteractionRewards:
		vitals.Happiness += 0.25 * intensity * boost
		p.digestLocked(biology.FoodItem{Bulk: 0.1 * intensity}) // Treats are food too

	case types.InteractionCleaning:
		p.LivingAreaWaste = 0
		vitals.Stress -= 0.03 // A clean home is calmer
		vitals.Happiness += 0.02

	case types.InteractionWalking:
		ph := p.Biology.Physiology
		ph.Bowel, ph.Bladder = 0, 0 // Relieved outside, and picked up after
		vitals.Happiness += 0.15 * intensity * boost
		vitals.Stress -= 0.05 * intensity * boost
		vitals.Energy -= 0.08 * intensity
		p.Biology.ProcessExercise(biology.ExerciseWalk, intensity)

	case types.InteractionDiscipline:
		vitals.Stress += 0.15 * intensity
		vitals.Happiness -= 0.1 * intensity
	}

	vitals.Clamp()
}

// updateBehavior determines the current behavior based on state
func (p *DigitalPet) updateBehavior() {
	// An interaction (such as tiring play) can leave a stat critical
	if p.Hibernating || p.shouldHibernateLocked() {
		p.hibernateLocked()
		return
	}
	// The behavior state machine chooses from the pet's state
	m := p.behaviorMachineLocked()
	if m.GetCurrentState() != p.CurrentBehavior {
		m.ForceState(p.CurrentBehavior) // Set elsewhere, such as by hibernation
	}
	m.Update()
	p.CurrentBehavior = m.GetCurrentState()
}

const (
	// learnedBehaviorWeight scales how far learned preferences shift a
	// personality-based tendency (tendencies and the threshold are 0.0-1.0)
	learnedBehaviorWeight = 0.3

	// discretionaryThreshold is the tendency a behavior needs to be chosen
	discretionaryThreshold = 0.6
)

// chooseDiscretionaryBehavior picks the strongest optional behavior from
// personality plus what the pet has learned from past interactions: how
// much it likes each kind of care, and how its owner has responded when
// it behaved this way before (reinforcement learning).
// It is deterministic, so behavior stays consistent until interactions
// gradually change the learned preferences.
func (p *DigitalPet) chooseDiscretionaryBehavior() types.BehaviorState {
	learned := p.Memory.GetLearnedAffinity
	// The season and weather make a pet more or less active, and bad
	// weather draws it to its owner
	drive, comfort := p.activityDriveLocked(), p.comfortSeekingLocked()

	candidates := []struct {
		behavior types.BehaviorState
		score    float64
	}{
		{
			// Enjoying play makes a pet more playful
			types.BehaviorPlaying,
			p.Personality.GetTraitInfluence("play")*drive + learnedBehaviorWeight*learned(types.InteractionPlaying) +
				learnedBehaviorWeight*p.learnedRewardLocked(types.BehaviorPlaying),
		},
		{
			// Enjoying petting draws a pet toward its owner; discipline
			// (always negative) makes it more withdrawn
			types.BehaviorSocialInteraction,
			p.Personality.GetTraitInfluence("social") + comfort +
				learnedBehaviorWeight*(learned(types.InteractionPetting)+learned(types.InteractionDiscipline)) +
				learnedBehaviorWeight*p.learnedRewardLocked(types.BehaviorSocialInteraction),
		},
		{
			// Enjoying exploring trips makes a pet more curious at home
			types.BehaviorInvestigating,
			p.Personality.GetTraitInfluence("explore") + learnedBehaviorWeight*learned(types.InteractionEnvironmentalEnrichment) +
				learnedBehaviorWeight*p.learnedRewardLocked(types.BehaviorInvestigating),
		},
		{
			// A pet grooms itself when its coat needs it; cats groom far
			// more than dogs, and tidy pets more than others
			types.BehaviorGrooming,
			p.selfGroomingScoreLocked() + learnedBehaviorWeight*p.learnedRewardLocked(types.BehaviorGrooming),
		},
		{
			// An energetic pet that hasn't had enough exercise runs around
			// to burn off energy
			types.BehaviorExercising,
			p.selfExerciseScoreLocked()*drive + learnedBehaviorWeight*p.learnedRewardLocked(types.BehaviorExercising),
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
		PetID:             p.ID,
		Name:              p.Name,
		IsAlive:           p.Biology.IsAlive,
		Hibernating:       p.Hibernating,
		Age:               p.Biology.GetAgeInDays(),
		Health:            p.Biology.Vitals.Health,
		Energy:            p.Biology.Vitals.Energy,
		Happiness:         p.Biology.Vitals.Happiness,
		Wellbeing:         p.Biology.Vitals.GetOverallWellbeing(),
		CurrentBehavior:   p.CurrentBehavior,
		MoodDescription:   p.Emotions.GetMoodDescription(),
		StatusDescription: p.Biology.GetStatus(),
		CriticalNeeds:     p.Biology.Vitals.GetCriticalStats(0.3),
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

	data, err := json.MarshalIndent(p, "", "  ")
	var unsupported *json.UnsupportedValueError
	if errors.As(err, &unsupported) {
		// A value that isn't a number (from a damaged save the checks
		// missed); the pet is saved with 0 in its place rather than not
		// at all
		replaceNonFinite(reflect.ValueOf(p).Elem())
		data, err = json.MarshalIndent(p, "", "  ")
	}
	return data, err
}

// Load deserializes a pet from JSON
func Load(data []byte) (loaded *DigitalPet, err error) {
	defer func() {
		if r := recover(); r != nil { // A damaged save the checks missed
			loaded, err = nil, fmt.Errorf("%w: %v", ErrDamagedSave, r)
		}
	}()
	var pet DigitalPet
	if err := json.Unmarshal(data, &pet); err != nil {
		return nil, err
	}
	if err := pet.repair(); err != nil {
		return nil, err
	}
	pet.Memory.SetLongTermCapacity(maxLongTermMemories) // Older saves kept more
	return &pet, nil
}

// newPetID returns a random pet ID, which says nothing about the pet
func newPetID() types.PetID {
	raw := make([]byte, 16)
	if _, err := rand.Read(raw); err != nil {
		panic(err) // The system's random source failing is not recoverable
	}
	return types.PetID("pet_" + hex.EncodeToString(raw))
}

// maxLongTermMemories is how many long-term memories a pet keeps: the
// strongest, which is enough for its behavior and keeps its save under
// about a megabyte however long it is played
const maxLongTermMemories = 1000

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
