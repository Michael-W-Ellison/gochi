package social

import (
	"testing"
	"time"

	"github.com/Michael-W-Ellison/gochi/pkg/types"
)

func TestNewRelationship(t *testing.T) {
	petID := types.PetID("test-pet-1")
	rel := NewRelationship(petID, types.RelationshipFriend)

	if rel == nil {
		t.Fatal("NewRelationship returned nil")
	}

	if rel.PetID != petID {
		t.Errorf("Expected PetID %s, got %s", petID, rel.PetID)
	}

	if rel.Type != types.RelationshipFriend {
		t.Errorf("Expected type Friend, got %v", rel.Type)
	}

	if rel.BondStrength != 0.3 {
		t.Errorf("Expected BondStrength 0.3, got %f", rel.BondStrength)
	}

	if rel.Trust != 0.5 {
		t.Errorf("Expected Trust 0.5, got %f", rel.Trust)
	}

	if rel.Affection != 0.3 {
		t.Errorf("Expected Affection 0.3, got %f", rel.Affection)
	}

	if rel.Rivalry != 0.1 {
		t.Errorf("Expected Rivalry 0.1, got %f", rel.Rivalry)
	}

	if len(rel.History) != 0 {
		t.Errorf("Expected empty history, got %d items", len(rel.History))
	}
}

func TestRelationshipUpdate(t *testing.T) {
	petID := types.PetID("test-pet-1")
	rel := NewRelationship(petID, types.RelationshipFriend)

	initialBondStrength := rel.BondStrength
	initialTrust := rel.Trust
	initialAffection := rel.Affection
	initialRivalry := rel.Rivalry

	// Positive interaction
	rel.Update(1.0, 0.0)

	if rel.BondStrength <= initialBondStrength {
		t.Error("Positive interaction should increase BondStrength")
	}

	if rel.Trust <= initialTrust {
		t.Error("Positive interaction should increase Trust")
	}

	if rel.Affection <= initialAffection {
		t.Error("Positive interaction should increase Affection")
	}

	if rel.Rivalry >= initialRivalry {
		t.Error("Positive interaction should decrease Rivalry")
	}

	if rel.TotalInteractions != 1 {
		t.Errorf("Expected 1 interaction, got %d", rel.TotalInteractions)
	}
}

func TestRelationshipNegativeUpdate(t *testing.T) {
	petID := types.PetID("test-pet-1")
	rel := NewRelationship(petID, types.RelationshipFriend)

	initialBondStrength := rel.BondStrength
	initialTrust := rel.Trust
	initialRivalry := rel.Rivalry

	// Negative interaction
	rel.Update(-1.0, 0.0)

	if rel.BondStrength >= initialBondStrength {
		t.Error("Negative interaction should decrease BondStrength")
	}

	if rel.Trust >= initialTrust {
		t.Error("Negative interaction should decrease Trust")
	}

	if rel.Rivalry <= initialRivalry {
		t.Error("Negative interaction should increase Rivalry")
	}
}

func TestRelationshipClamp(t *testing.T) {
	petID := types.PetID("test-pet-1")
	rel := NewRelationship(petID, types.RelationshipFriend)

	// Force values outside valid range
	rel.BondStrength = 1.5
	rel.Trust = -0.5
	rel.Affection = 2.0
	rel.Rivalry = -1.0

	rel.Clamp()

	if rel.BondStrength > 1.0 {
		t.Errorf("BondStrength should be clamped to 1.0, got %f", rel.BondStrength)
	}

	if rel.Trust < 0.0 {
		t.Errorf("Trust should be clamped to 0.0, got %f", rel.Trust)
	}

	if rel.Affection > 1.0 {
		t.Errorf("Affection should be clamped to 1.0, got %f", rel.Affection)
	}

	if rel.Rivalry < 0.0 {
		t.Errorf("Rivalry should be clamped to 0.0, got %f", rel.Rivalry)
	}
}

func TestAddSharedExperience(t *testing.T) {
	petID := types.PetID("test-pet-1")
	rel := NewRelationship(petID, types.RelationshipFriend)

	rel.AddSharedExperience("Went on an adventure", 100.0, 0.5)

	if len(rel.History) != 1 {
		t.Errorf("Expected 1 experience, got %d", len(rel.History))
	}

	exp := rel.History[0]
	if exp.Description != "Went on an adventure" {
		t.Errorf("Expected description 'Went on an adventure', got %s", exp.Description)
	}

	if exp.GameTime != 100.0 {
		t.Errorf("Expected GameTime 100.0, got %f", exp.GameTime)
	}

	if exp.Impact != 0.5 {
		t.Errorf("Expected Impact 0.5, got %f", exp.Impact)
	}
}

func TestSharedExperienceLimit(t *testing.T) {
	petID := types.PetID("test-pet-1")
	rel := NewRelationship(petID, types.RelationshipFriend)

	// Add more than 50 experiences
	for i := 0; i < 60; i++ {
		rel.AddSharedExperience("Experience", float64(i), 0.1)
	}

	if len(rel.History) != 50 {
		t.Errorf("Expected history limited to 50, got %d", len(rel.History))
	}

	// Check that oldest experiences were removed
	if rel.History[0].GameTime != 10.0 {
		t.Errorf("Expected oldest experience to be from iteration 10, got %f", rel.History[0].GameTime)
	}
}

func TestRelationshipDecay(t *testing.T) {
	petID := types.PetID("test-pet-1")
	rel := NewRelationship(petID, types.RelationshipFriend)

	// Set last interaction to 2 days ago
	rel.LastInteraction = time.Now().Add(-48 * time.Hour)

	initialBondStrength := rel.BondStrength
	initialAffection := rel.Affection
	initialTrust := rel.Trust

	// Apply decay
	rel.Decay(1.0)

	if rel.BondStrength >= initialBondStrength {
		t.Error("Decay should decrease BondStrength")
	}

	if rel.Affection >= initialAffection {
		t.Error("Decay should decrease Affection")
	}

	if rel.Trust >= initialTrust {
		t.Error("Decay should decrease Trust")
	}
}

func TestGetRelationshipQuality(t *testing.T) {
	petID := types.PetID("test-pet-1")
	rel := NewRelationship(petID, types.RelationshipFriend)

	quality := rel.GetRelationshipQuality()

	if quality < 0.0 || quality > 1.0 {
		t.Errorf("Quality should be between 0 and 1, got %f", quality)
	}

	// Set to maximum positive values
	rel.BondStrength = 1.0
	rel.Trust = 1.0
	rel.Affection = 1.0
	rel.Rivalry = 0.0

	quality = rel.GetRelationshipQuality()
	if quality < 0.9 {
		t.Errorf("Expected high quality (>0.9), got %f", quality)
	}

	// Set to maximum negative values
	rel.BondStrength = 0.0
	rel.Trust = 0.0
	rel.Affection = 0.0
	rel.Rivalry = 1.0

	quality = rel.GetRelationshipQuality()
	if quality > 0.1 {
		t.Errorf("Expected low quality (<0.1), got %f", quality)
	}
}

func TestGetDescription(t *testing.T) {
	tests := []struct {
		relType  types.RelationshipType
		quality  float64
		rivalry  float64
		expected string
	}{
		{types.RelationshipFriend, 0.9, 0.0, "best friends"},
		{types.RelationshipFriend, 0.7, 0.0, "good friends"},
		{types.RelationshipFriend, 0.5, 0.0, "friends"},
		{types.RelationshipFriend, 0.3, 0.4, "acquaintances"},
		{types.RelationshipRival, 0.5, 0.8, "bitter rivals"},
		{types.RelationshipRival, 0.5, 0.5, "rivals"},
		{types.RelationshipRival, 0.5, 0.2, "competitive"},
		{types.RelationshipMate, 0.9, 0.0, "devoted mates"},
		{types.RelationshipMate, 0.7, 0.0, "mates"},
		{types.RelationshipMate, 0.5, 0.0, "partners"},
		{types.RelationshipParent, 0.5, 0.0, "parent"},
		{types.RelationshipOffspring, 0.5, 0.0, "offspring"},
	}

	for _, test := range tests {
		petID := types.PetID("test-pet")
		rel := NewRelationship(petID, test.relType)

		// Set values to achieve desired quality
		// quality = (positive - negative + 1.0) / 2.0
		// where positive = (BondStrength + Trust + Affection) / 3.0
		if test.quality > 0.8 {
			// Need quality > 0.8, so (positive - rivalry + 1) / 2 > 0.8
			// With rivalry = 0: positive > 0.6
			rel.BondStrength = 0.9
			rel.Trust = 0.9
			rel.Affection = 0.9
		} else if test.quality > 0.6 {
			// Need 0.6 < quality <= 0.8, so (positive - rivalry + 1) / 2 <= 0.8
			// With rivalry = 0: positive <= 0.6
			rel.BondStrength = 0.5
			rel.Trust = 0.5
			rel.Affection = 0.5
		} else if test.quality > 0.4 {
			// Need 0.4 < quality <= 0.6
			// With rivalry = 0: positive <= 0.2
			rel.BondStrength = 0.1
			rel.Trust = 0.1
			rel.Affection = 0.1
		} else {
			// Need quality <= 0.4
			// Use rivalry to lower quality
			rel.BondStrength = 0.0
			rel.Trust = 0.0
			rel.Affection = 0.0
		}

		rel.Rivalry = test.rivalry

		description := rel.GetDescription()
		if description != test.expected {
			t.Errorf("For type %v with quality %f and rivalry %f, expected '%s', got '%s'",
				test.relType, test.quality, test.rivalry, test.expected, description)
		}
	}
}

func TestNewSocialRelationships(t *testing.T) {
	maxRel := 10
	social := NewSocialRelationships(maxRel)

	if social == nil {
		t.Fatal("NewSocialRelationships returned nil")
	}

	if social.MaxRelationships != maxRel {
		t.Errorf("Expected MaxRelationships %d, got %d", maxRel, social.MaxRelationships)
	}

	if len(social.Relationships) != 0 {
		t.Errorf("Expected empty relationships, got %d", len(social.Relationships))
	}
}

func TestAddRelationship(t *testing.T) {
	social := NewSocialRelationships(10)

	petID := types.PetID("test-pet-1")
	rel := social.AddRelationship(petID, types.RelationshipFriend)

	if rel == nil {
		t.Fatal("AddRelationship returned nil")
	}

	if len(social.Relationships) != 1 {
		t.Errorf("Expected 1 relationship, got %d", len(social.Relationships))
	}

	if !social.HasRelationshipWith(petID) {
		t.Error("Relationship should exist")
	}
}

func TestGetRelationship(t *testing.T) {
	social := NewSocialRelationships(10)

	petID := types.PetID("test-pet-1")
	social.AddRelationship(petID, types.RelationshipFriend)

	rel, exists := social.GetRelationship(petID)

	if !exists {
		t.Fatal("Relationship should exist")
	}

	if rel.PetID != petID {
		t.Errorf("Expected PetID %s, got %s", petID, rel.PetID)
	}

	// Test non-existent relationship
	_, exists = social.GetRelationship(types.PetID("non-existent"))
	if exists {
		t.Error("Non-existent relationship should not exist")
	}
}

func TestUpdateRelationship(t *testing.T) {
	social := NewSocialRelationships(10)

	petID := types.PetID("test-pet-1")
	social.AddRelationship(petID, types.RelationshipFriend)

	social.UpdateRelationship(petID, 1.0, 0.0)

	rel, _ := social.GetRelationship(petID)
	if rel.TotalInteractions != 1 {
		t.Errorf("Expected 1 interaction, got %d", rel.TotalInteractions)
	}
}

func TestUpdateAll(t *testing.T) {
	social := NewSocialRelationships(10)

	// Add multiple relationships
	for i := 0; i < 5; i++ {
		petID := types.PetID("test-pet-" + string(rune('1'+i)))
		social.AddRelationship(petID, types.RelationshipFriend)
	}

	// Set all last interactions to 2 days ago
	for _, rel := range social.Relationships {
		rel.LastInteraction = time.Now().Add(-48 * time.Hour)
	}

	// Apply decay to all
	social.UpdateAll(1.0)

	// Check that all were updated
	for _, rel := range social.Relationships {
		if rel.BondStrength >= 0.3 {
			// Should have decayed from initial 0.3
			t.Error("Relationship should have decayed")
		}
	}
}

func TestGetClosestFriends(t *testing.T) {
	social := NewSocialRelationships(10)

	// Add friends with different bond strengths
	for i := 0; i < 5; i++ {
		petID := types.PetID("friend-" + string(rune('1'+i)))
		rel := social.AddRelationship(petID, types.RelationshipFriend)
		rel.BondStrength = float64(i) * 0.2
	}

	// Add a rival (should not be included)
	social.AddRelationship(types.PetID("rival-1"), types.RelationshipRival)

	// Get top 3 friends
	closest := social.GetClosestFriends(3)

	if len(closest) != 3 {
		t.Errorf("Expected 3 closest friends, got %d", len(closest))
	}

	// Check they're in order of strength
	for i := 0; i < len(closest)-1; i++ {
		if closest[i].BondStrength < closest[i+1].BondStrength {
			t.Error("Friends should be ordered by bond strength")
		}
	}

	// Verify all are friends
	for _, rel := range closest {
		if rel.Type != types.RelationshipFriend {
			t.Error("All should be friends")
		}
	}
}

func TestMaxRelationshipsLimit(t *testing.T) {
	maxRel := 5
	social := NewSocialRelationships(maxRel)

	// Add more than max relationships
	for i := 0; i < maxRel+3; i++ {
		petID := types.PetID("pet-" + string(rune('1'+i)))
		social.AddRelationship(petID, types.RelationshipFriend)
	}

	// Should be limited to maxRel
	if len(social.Relationships) > maxRel {
		t.Errorf("Relationships should be limited to %d, got %d", maxRel, len(social.Relationships))
	}
}

func TestRemoveWeakestRelationship(t *testing.T) {
	social := NewSocialRelationships(3)

	// Add relationships with different bond strengths
	rel1 := social.AddRelationship(types.PetID("pet-1"), types.RelationshipFriend)
	rel1.BondStrength = 0.5

	rel2 := social.AddRelationship(types.PetID("pet-2"), types.RelationshipFriend)
	rel2.BondStrength = 0.3

	rel3 := social.AddRelationship(types.PetID("pet-3"), types.RelationshipFriend)
	rel3.BondStrength = 0.7

	// Add a 4th - should remove pet-2 (weakest)
	social.AddRelationship(types.PetID("pet-4"), types.RelationshipFriend)

	if social.HasRelationshipWith(types.PetID("pet-2")) {
		t.Error("Weakest relationship should have been removed")
	}

	if !social.HasRelationshipWith(types.PetID("pet-1")) {
		t.Error("Stronger relationship should remain")
	}

	if !social.HasRelationshipWith(types.PetID("pet-3")) {
		t.Error("Strongest relationship should remain")
	}
}

func TestFamilyRelationshipsNotRemoved(t *testing.T) {
	social := NewSocialRelationships(2)

	// Add parent relationship with low bond
	parent := social.AddRelationship(types.PetID("parent"), types.RelationshipParent)
	parent.BondStrength = 0.1

	// Add friend relationship with high bond
	friend := social.AddRelationship(types.PetID("friend"), types.RelationshipFriend)
	friend.BondStrength = 0.9

	// Try to add another friend - should remove friend, not parent
	social.AddRelationship(types.PetID("new-friend"), types.RelationshipFriend)

	if !social.HasRelationshipWith(types.PetID("parent")) {
		t.Error("Parent relationship should not be removed")
	}
}

func TestGetRelationshipCount(t *testing.T) {
	social := NewSocialRelationships(10)

	if social.GetRelationshipCount() != 0 {
		t.Errorf("Expected 0 relationships, got %d", social.GetRelationshipCount())
	}

	social.AddRelationship(types.PetID("pet-1"), types.RelationshipFriend)
	social.AddRelationship(types.PetID("pet-2"), types.RelationshipFriend)

	if social.GetRelationshipCount() != 2 {
		t.Errorf("Expected 2 relationships, got %d", social.GetRelationshipCount())
	}
}

func TestGetAverageBondStrength(t *testing.T) {
	social := NewSocialRelationships(10)

	// Empty should return 0
	if social.GetAverageBondStrength() != 0.0 {
		t.Errorf("Expected 0.0 for empty, got %f", social.GetAverageBondStrength())
	}

	// Add relationships with known bond strengths
	rel1 := social.AddRelationship(types.PetID("pet-1"), types.RelationshipFriend)
	rel1.BondStrength = 0.4

	rel2 := social.AddRelationship(types.PetID("pet-2"), types.RelationshipFriend)
	rel2.BondStrength = 0.6

	average := social.GetAverageBondStrength()
	expected := 0.5

	if average != expected {
		t.Errorf("Expected average %f, got %f", expected, average)
	}
}

func TestClampFunction(t *testing.T) {
	tests := []struct {
		value    float64
		min      float64
		max      float64
		expected float64
	}{
		{0.5, 0.0, 1.0, 0.5},
		{-0.5, 0.0, 1.0, 0.0},
		{1.5, 0.0, 1.0, 1.0},
		{0.0, 0.0, 1.0, 0.0},
		{1.0, 0.0, 1.0, 1.0},
	}

	for _, test := range tests {
		result := clamp(test.value, test.min, test.max)
		if result != test.expected {
			t.Errorf("clamp(%f, %f, %f) = %f, expected %f",
				test.value, test.min, test.max, result, test.expected)
		}
	}
}
