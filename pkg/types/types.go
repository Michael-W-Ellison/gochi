package types

import "time"

// PetID is a unique identifier for a digital pet
type PetID string

// UserID is a unique identifier for a user/caregiver
type UserID string

// TimeScale represents different simulation speed modes
type TimeScale int

const (
	TimeScaleRealTime TimeScale = iota
	TimeScaleAccelerated4X
	TimeScaleAccelerated24X
	TimeScalePaused
)

// BehaviorState represents the current behavioral state of the pet
type BehaviorState int

const (
	BehaviorIdle BehaviorState = iota
	BehaviorSleeping
	BehaviorEating
	BehaviorPlaying
	BehaviorExploring
	BehaviorSocialInteraction
	BehaviorGrooming
	BehaviorExercising
	BehaviorSick
	BehaviorDistressed
	BehaviorHappy
	BehaviorExcited
)

// String returns the string representation of BehaviorState
func (bs BehaviorState) String() string {
	return [...]string{
		"Idle", "Sleeping", "Eating", "Playing", "Exploring",
		"Social Interaction", "Grooming", "Exercising",
		"Sick", "Distressed", "Happy", "Excited",
	}[bs]
}

// InteractionType represents different types of user interactions
type InteractionType int

const (
	InteractionFeeding InteractionType = iota
	InteractionPetting
	InteractionPlaying
	InteractionTraining
	InteractionGrooming
	InteractionMedicalCare
	InteractionEnvironmentalEnrichment
	InteractionSocialIntroduction
	InteractionDiscipline
	InteractionRewards
)

// String returns the string representation of InteractionType
func (it InteractionType) String() string {
	return [...]string{
		"Feeding", "Petting", "Playing", "Training", "Grooming",
		"Medical Care", "Environmental Enrichment", "Social Introduction",
		"Discipline", "Rewards",
	}[it]
}

// NeedType represents different needs the pet has
type NeedType int

const (
	NeedHunger NeedType = iota
	NeedThirst
	NeedSleep
	NeedExercise
	NeedSocial
	NeedMentalStimulation
	NeedAffection
	NeedCleanliness
	NeedMedicalCare
	NeedExploration
)

// String returns the string representation of NeedType
func (nt NeedType) String() string {
	return [...]string{
		"Hunger", "Thirst", "Sleep", "Exercise", "Social",
		"Mental Stimulation", "Affection", "Cleanliness",
		"Medical Care", "Exploration",
	}[nt]
}

// RelationshipType represents the type of relationship between pets
type RelationshipType int

const (
	RelationshipFriend RelationshipType = iota
	RelationshipRival
	RelationshipMate
	RelationshipOffspring
	RelationshipParent
)

// String returns the string representation of RelationshipType
func (rt RelationshipType) String() string {
	return [...]string{
		"Friend", "Rival", "Mate", "Offspring", "Parent",
	}[rt]
}

// Priority represents the importance level of needs or actions
type Priority int

const (
	PriorityLow Priority = iota
	PriorityMedium
	PriorityHigh
	PriorityCritical
)

// Timestamp represents a point in time in the simulation
type Timestamp struct {
	Real       time.Time // Real-world time
	Simulation time.Time // Simulated time (affected by time scale)
}
