package social

import (
	"time"

	"github.com/Michael-W-Ellison/gochi/pkg/types"
)

// SharedExperience represents an experience shared between two pets
type SharedExperience struct {
	Description string
	Timestamp   time.Time
	GameTime    float64
	Impact      float64 // How much it affected the relationship
}

// Relationship represents the bond between two pets
type Relationship struct {
	PetID            types.PetID
	Type             types.RelationshipType
	BondStrength     float64 // 0.0 to 1.0
	Trust            float64 // 0.0 to 1.0
	Affection        float64 // 0.0 to 1.0
	Rivalry          float64 // 0.0 to 1.0 (conflict level)
	History          []SharedExperience
	LastInteraction  time.Time
	TotalInteractions int
	FirstMet         time.Time
}

// NewRelationship creates a new relationship with neutral starting values
func NewRelationship(petID types.PetID, relType types.RelationshipType) *Relationship {
	now := time.Now()
	return &Relationship{
		PetID:            petID,
		Type:             relType,
		BondStrength:     0.3,
		Trust:            0.5,
		Affection:        0.3,
		Rivalry:          0.1,
		History:          make([]SharedExperience, 0),
		LastInteraction:  now,
		TotalInteractions: 0,
		FirstMet:         now,
	}
}

// Update modifies the relationship based on interaction quality
func (r *Relationship) Update(interactionQuality float64, gameTime float64) {
	r.LastInteraction = time.Now()
	r.TotalInteractions++

	// Positive interactions strengthen bond
	if interactionQuality > 0 {
		r.BondStrength += interactionQuality * 0.05
		r.Trust += interactionQuality * 0.03
		r.Affection += interactionQuality * 0.04
		r.Rivalry -= interactionQuality * 0.02
	} else {
		// Negative interactions increase rivalry and decrease other stats
		r.Rivalry += (-interactionQuality) * 0.05
		r.Trust -= (-interactionQuality) * 0.04
		r.BondStrength -= (-interactionQuality) * 0.03
	}

	r.Clamp()
}

// AddSharedExperience records a shared experience
func (r *Relationship) AddSharedExperience(description string, gameTime float64, impact float64) {
	exp := SharedExperience{
		Description: description,
		Timestamp:   time.Now(),
		GameTime:    gameTime,
		Impact:      impact,
	}

	r.History = append(r.History, exp)

	// Keep only last 50 experiences
	if len(r.History) > 50 {
		r.History = r.History[1:]
	}
}

// Decay weakens relationships over time without interaction
func (r *Relationship) Decay(deltaTime float64) {
	timeSinceInteraction := time.Since(r.LastInteraction).Hours() / 24.0 // Days

	if timeSinceInteraction > 1.0 {
		decayRate := 0.01 * deltaTime
		r.BondStrength -= decayRate
		r.Affection -= decayRate
		r.Trust -= decayRate * 0.5
	}

	r.Clamp()
}

// Clamp ensures all values stay within valid range
func (r *Relationship) Clamp() {
	r.BondStrength = clamp(r.BondStrength, 0.0, 1.0)
	r.Trust = clamp(r.Trust, 0.0, 1.0)
	r.Affection = clamp(r.Affection, 0.0, 1.0)
	r.Rivalry = clamp(r.Rivalry, 0.0, 1.0)
}

// GetRelationshipQuality returns overall quality of relationship
func (r *Relationship) GetRelationshipQuality() float64 {
	positive := (r.BondStrength + r.Trust + r.Affection) / 3.0
	negative := r.Rivalry
	return (positive - negative + 1.0) / 2.0 // Normalize to 0-1
}

// GetDescription returns a text description of the relationship
func (r *Relationship) GetDescription() string {
	quality := r.GetRelationshipQuality()

	switch r.Type {
	case types.RelationshipFriend:
		if quality > 0.8 {
			return "best friends"
		} else if quality > 0.6 {
			return "good friends"
		} else if quality > 0.4 {
			return "friends"
		} else {
			return "acquaintances"
		}

	case types.RelationshipRival:
		if r.Rivalry > 0.7 {
			return "bitter rivals"
		} else if r.Rivalry > 0.4 {
			return "rivals"
		} else {
			return "competitive"
		}

	case types.RelationshipMate:
		if quality > 0.8 {
			return "devoted mates"
		} else if quality > 0.6 {
			return "mates"
		} else {
			return "partners"
		}

	case types.RelationshipParent:
		return "parent"

	case types.RelationshipOffspring:
		return "offspring"

	default:
		return "acquaintance"
	}
}

// SocialRelationships manages all of a pet's relationships
type SocialRelationships struct {
	Relationships map[types.PetID]*Relationship
	MaxRelationships int
}

// NewSocialRelationships creates a new social relationship manager
func NewSocialRelationships(maxRelationships int) *SocialRelationships {
	return &SocialRelationships{
		Relationships: make(map[types.PetID]*Relationship),
		MaxRelationships: maxRelationships,
	}
}

// AddRelationship creates a new relationship with another pet
func (s *SocialRelationships) AddRelationship(petID types.PetID, relType types.RelationshipType) *Relationship {
	if len(s.Relationships) >= s.MaxRelationships {
		// Remove weakest relationship to make room
		s.removeWeakestRelationship()
	}

	rel := NewRelationship(petID, relType)
	s.Relationships[petID] = rel
	return rel
}

// GetRelationship retrieves a relationship with a specific pet
func (s *SocialRelationships) GetRelationship(petID types.PetID) (*Relationship, bool) {
	rel, exists := s.Relationships[petID]
	return rel, exists
}

// UpdateRelationship modifies an existing relationship
func (s *SocialRelationships) UpdateRelationship(petID types.PetID, quality float64, gameTime float64) {
	if rel, exists := s.Relationships[petID]; exists {
		rel.Update(quality, gameTime)
	}
}

// UpdateAll processes decay for all relationships
func (s *SocialRelationships) UpdateAll(deltaTime float64) {
	for _, rel := range s.Relationships {
		rel.Decay(deltaTime)
	}
}

// GetClosestFriends returns the N strongest friendships
func (s *SocialRelationships) GetClosestFriends(count int) []*Relationship {
	var friends []*Relationship

	for _, rel := range s.Relationships {
		if rel.Type == types.RelationshipFriend {
			friends = append(friends, rel)
		}
	}

	// Simple selection of strongest friendships
	var closest []*Relationship
	for i := 0; i < count && i < len(friends); i++ {
		maxIdx := -1
		maxStrength := 0.0

		for j, friend := range friends {
			alreadySelected := false
			for _, c := range closest {
				if c.PetID == friend.PetID {
					alreadySelected = true
					break
				}
			}

			if !alreadySelected && friend.BondStrength > maxStrength {
				maxIdx = j
				maxStrength = friend.BondStrength
			}
		}

		if maxIdx >= 0 {
			closest = append(closest, friends[maxIdx])
		}
	}

	return closest
}

// GetRelationshipCount returns the total number of relationships
func (s *SocialRelationships) GetRelationshipCount() int {
	return len(s.Relationships)
}

// HasRelationshipWith checks if a relationship exists with a pet
func (s *SocialRelationships) HasRelationshipWith(petID types.PetID) bool {
	_, exists := s.Relationships[petID]
	return exists
}

// removeWeakestRelationship removes the relationship with lowest bond strength
func (s *SocialRelationships) removeWeakestRelationship() {
	var weakestID types.PetID
	weakestStrength := 2.0 // Higher than max possible

	for id, rel := range s.Relationships {
		// Don't remove family relationships
		if rel.Type == types.RelationshipParent || rel.Type == types.RelationshipOffspring {
			continue
		}

		if rel.BondStrength < weakestStrength {
			weakestID = id
			weakestStrength = rel.BondStrength
		}
	}

	if weakestStrength < 2.0 {
		delete(s.Relationships, weakestID)
	}
}

// GetAverageBondStrength returns the average bond strength across all relationships
func (s *SocialRelationships) GetAverageBondStrength() float64 {
	if len(s.Relationships) == 0 {
		return 0.0
	}

	total := 0.0
	for _, rel := range s.Relationships {
		total += rel.BondStrength
	}

	return total / float64(len(s.Relationships))
}

// Helper function to clamp values
func clamp(value, min, max float64) float64 {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}
