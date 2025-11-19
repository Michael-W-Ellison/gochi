package core

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/Michael-W-Ellison/gochi/internal/ai"
	"github.com/Michael-W-Ellison/gochi/internal/biology"
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

// Update processes all system updates for a given time delta
func (p *DigitalPet) Update(deltaTime float64) {
	if !p.Biology.IsAlive {
		return
	}

	// Update biological systems
	p.Biology.Update(deltaTime)

	// Update emotions
	p.Emotions.Update(deltaTime)

	// Decay memories
	p.Memory.DecayMemories(deltaTime)

	// Periodically consolidate memories
	if p.TotalInteractions%10 == 0 {
		p.Memory.ConsolidateMemories()
	}

	// Update relationships
	p.Relationships.UpdateAll(deltaTime)

	// Update current behavior based on state
	p.updateBehavior()

	// Track time
	p.LastUpdateAt = time.Now()
	p.TotalPlayTime += deltaTime / 60.0 // Convert to hours
}

// ProcessUserInteraction handles user actions and applies effects
func (p *DigitalPet) ProcessUserInteraction(interactionType types.InteractionType, intensity float64) {
	if !p.Biology.IsAlive {
		return
	}

	p.TotalInteractions++

	// Apply biological effects
	p.applyInteractionEffects(interactionType, intensity)

	// Apply emotional effects
	stimulus := ai.CreateStimulusFromInteraction(interactionType, intensity)
	p.Emotions.ApplyEmotionalStimulus(stimulus)

	// Record memory
	p.Memory.RecordInteraction(interactionType, p.Biology.GetAgeInDays(), intensity, p.Emotions.DominantEmotion)

	// Update behavior
	p.updateBehavior()
}

// applyInteractionEffects modifies vital stats based on interaction type
func (p *DigitalPet) applyInteractionEffects(interactionType types.InteractionType, intensity float64) {
	vitals := p.Biology.Vitals

	switch interactionType {
	case types.InteractionFeeding:
		vitals.Nutrition += 0.3 * intensity
		vitals.Energy += 0.1 * intensity
		vitals.Happiness += 0.05 * intensity

	case types.InteractionPetting:
		vitals.Happiness += 0.15 * intensity
		vitals.Stress -= 0.1 * intensity

	case types.InteractionPlaying:
		vitals.Happiness += 0.2 * intensity
		vitals.Energy -= 0.1 * intensity
		vitals.Stress -= 0.05 * intensity

	case types.InteractionGrooming:
		vitals.Cleanliness += 0.3 * intensity
		vitals.Happiness += 0.05 * intensity

	case types.InteractionMedicalCare:
		vitals.Health += 0.2 * intensity
		vitals.Stress += 0.05 * intensity // Medical care can be stressful

	case types.InteractionRewards:
		vitals.Happiness += 0.25 * intensity

	case types.InteractionDiscipline:
		vitals.Stress += 0.15 * intensity
		vitals.Happiness -= 0.1 * intensity
	}

	vitals.Clamp()
}

// updateBehavior determines the current behavior based on state
func (p *DigitalPet) updateBehavior() {
	vitals := p.Biology.Vitals

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
	} else if p.Personality.GetTraitInfluence("play") > 0.6 && vitals.Energy > 0.4 {
		p.CurrentBehavior = types.BehaviorPlaying
	} else if p.Personality.GetTraitInfluence("explore") > 0.6 && vitals.Energy > 0.4 {
		p.CurrentBehavior = types.BehaviorExploring
	} else {
		p.CurrentBehavior = types.BehaviorIdle
	}
}

// GetCurrentStatus returns a comprehensive status report
func (p *DigitalPet) GetCurrentStatus() PetStatus {
	return PetStatus{
		PetID:            p.ID,
		Name:             p.Name,
		IsAlive:          p.Biology.IsAlive,
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
