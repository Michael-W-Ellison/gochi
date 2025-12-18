package social

import (
	"testing"
	"time"

	"github.com/Michael-W-Ellison/gochi/pkg/types"
)

// =============== Relationship Tests ===============

// TestNewRelationship tests relationship creation
func TestNewRelationship(t *testing.T) {
	petID := types.PetID("pet_123")
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

	// Check initial values
	if rel.BondStrength != 0.3 {
		t.Errorf("Expected initial BondStrength 0.3, got %f", rel.BondStrength)
	}

	if rel.Trust != 0.5 {
		t.Errorf("Expected initial Trust 0.5, got %f", rel.Trust)
	}

	if rel.Affection != 0.3 {
		t.Errorf("Expected initial Affection 0.3, got %f", rel.Affection)
	}

	if rel.Rivalry != 0.1 {
		t.Errorf("Expected initial Rivalry 0.1, got %f", rel.Rivalry)
	}

	if rel.TotalInteractions != 0 {
		t.Errorf("Expected 0 total interactions, got %d", rel.TotalInteractions)
	}

	if rel.History == nil {
		t.Error("History should not be nil")
	}
}

// TestNewRelationshipTypes tests creation of different relationship types
func TestNewRelationshipTypes(t *testing.T) {
	relTypes := []types.RelationshipType{
		types.RelationshipFriend,
		types.RelationshipRival,
		types.RelationshipMate,
		types.RelationshipParent,
		types.RelationshipOffspring,
	}

	for _, relType := range relTypes {
		rel := NewRelationship(types.PetID("test"), relType)
		if rel.Type != relType {
			t.Errorf("Expected type %v, got %v", relType, rel.Type)
		}
	}
}

// TestRelationshipUpdatePositive tests positive interactions
func TestRelationshipUpdatePositive(t *testing.T) {
	rel := NewRelationship(types.PetID("pet_123"), types.RelationshipFriend)

	initialBond := rel.BondStrength
	initialTrust := rel.Trust
	initialAffection := rel.Affection
	initialRivalry := rel.Rivalry

	rel.Update(1.0, 100.0) // Positive interaction

	if rel.BondStrength <= initialBond {
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
		t.Errorf("Expected 1 total interaction, got %d", rel.TotalInteractions)
	}
}

// TestRelationshipUpdateNegative tests negative interactions
func TestRelationshipUpdateNegative(t *testing.T) {
	rel := NewRelationship(types.PetID("pet_123"), types.RelationshipFriend)

	initialBond := rel.BondStrength
	initialTrust := rel.Trust
	initialRivalry := rel.Rivalry

	rel.Update(-1.0, 100.0) // Negative interaction

	if rel.BondStrength >= initialBond {
		t.Error("Negative interaction should decrease BondStrength")
	}

	if rel.Trust >= initialTrust {
		t.Error("Negative interaction should decrease Trust")
	}

	if rel.Rivalry <= initialRivalry {
		t.Error("Negative interaction should increase Rivalry")
	}

	if rel.TotalInteractions != 1 {
		t.Errorf("Expected 1 total interaction, got %d", rel.TotalInteractions)
	}
}

// TestRelationshipUpdateNeutral tests neutral interactions
func TestRelationshipUpdateNeutral(t *testing.T) {
	rel := NewRelationship(types.PetID("pet_123"), types.RelationshipFriend)

	initialBond := rel.BondStrength
	initialTrust := rel.Trust

	rel.Update(0.0, 100.0) // Neutral interaction

	// Neutral should not change stats significantly
	if rel.BondStrength != initialBond {
		t.Error("Neutral interaction should not change BondStrength")
	}

	if rel.Trust != initialTrust {
		t.Error("Neutral interaction should not change Trust")
	}

	// But should still count as an interaction
	if rel.TotalInteractions != 1 {
		t.Errorf("Expected 1 total interaction, got %d", rel.TotalInteractions)
	}
}

// TestRelationshipUpdateMultiple tests cumulative interactions
func TestRelationshipUpdateMultiple(t *testing.T) {
	rel := NewRelationship(types.PetID("pet_123"), types.RelationshipFriend)

	// Multiple positive interactions
	for i := 0; i < 10; i++ {
		rel.Update(0.8, float64(i*10))
	}

	if rel.TotalInteractions != 10 {
		t.Errorf("Expected 10 total interactions, got %d", rel.TotalInteractions)
	}

	// Bond should have increased significantly
	if rel.BondStrength < 0.5 {
		t.Error("Multiple positive interactions should significantly increase bond")
	}
}

// TestAddSharedExperience tests adding shared experiences
func TestAddSharedExperience(t *testing.T) {
	rel := NewRelationship(types.PetID("pet_123"), types.RelationshipFriend)

	rel.AddSharedExperience("Played together", 100.0, 0.5)

	if len(rel.History) != 1 {
		t.Errorf("Expected 1 history entry, got %d", len(rel.History))
	}

	exp := rel.History[0]
	if exp.Description != "Played together" {
		t.Errorf("Expected description 'Played together', got '%s'", exp.Description)
	}

	if exp.GameTime != 100.0 {
		t.Errorf("Expected GameTime 100.0, got %f", exp.GameTime)
	}

	if exp.Impact != 0.5 {
		t.Errorf("Expected Impact 0.5, got %f", exp.Impact)
	}
}

// TestAddSharedExperienceLimit tests history limit
func TestAddSharedExperienceLimit(t *testing.T) {
	rel := NewRelationship(types.PetID("pet_123"), types.RelationshipFriend)

	// Add more than 50 experiences
	for i := 0; i < 60; i++ {
		rel.AddSharedExperience("Experience", float64(i), 0.1)
	}

	// Should be capped at 50
	if len(rel.History) != 50 {
		t.Errorf("Expected max 50 history entries, got %d", len(rel.History))
	}

	// First experience should be the 11th one (index 10)
	if rel.History[0].GameTime != 10.0 {
		t.Errorf("Expected first experience GameTime 10.0, got %f", rel.History[0].GameTime)
	}
}

// TestRelationshipClamp tests value clamping
func TestRelationshipClamp(t *testing.T) {
	rel := NewRelationship(types.PetID("pet_123"), types.RelationshipFriend)

	// Set values out of range
	rel.BondStrength = 1.5
	rel.Trust = -0.5
	rel.Affection = 2.0
	rel.Rivalry = -1.0

	rel.Clamp()

	if rel.BondStrength != 1.0 {
		t.Errorf("BondStrength should be clamped to 1.0, got %f", rel.BondStrength)
	}

	if rel.Trust != 0.0 {
		t.Errorf("Trust should be clamped to 0.0, got %f", rel.Trust)
	}

	if rel.Affection != 1.0 {
		t.Errorf("Affection should be clamped to 1.0, got %f", rel.Affection)
	}

	if rel.Rivalry != 0.0 {
		t.Errorf("Rivalry should be clamped to 0.0, got %f", rel.Rivalry)
	}
}

// TestGetRelationshipQuality tests quality calculation
func TestGetRelationshipQuality(t *testing.T) {
	rel := NewRelationship(types.PetID("pet_123"), types.RelationshipFriend)

	// With default values, should be in the middle range
	quality := rel.GetRelationshipQuality()
	if quality < 0.0 || quality > 1.0 {
		t.Errorf("Quality should be between 0 and 1, got %f", quality)
	}

	// Set to perfect relationship
	rel.BondStrength = 1.0
	rel.Trust = 1.0
	rel.Affection = 1.0
	rel.Rivalry = 0.0

	quality = rel.GetRelationshipQuality()
	if quality != 1.0 {
		t.Errorf("Perfect relationship should have quality 1.0, got %f", quality)
	}

	// Set to worst relationship
	rel.BondStrength = 0.0
	rel.Trust = 0.0
	rel.Affection = 0.0
	rel.Rivalry = 1.0

	quality = rel.GetRelationshipQuality()
	if quality != 0.0 {
		t.Errorf("Worst relationship should have quality 0.0, got %f", quality)
	}
}

// TestGetDescriptionFriend tests friend descriptions
func TestGetDescriptionFriend(t *testing.T) {
	rel := NewRelationship(types.PetID("pet_123"), types.RelationshipFriend)

	// Best friends (quality > 0.8)
	// Quality = ((BondStrength + Trust + Affection) / 3 - Rivalry + 1) / 2
	// For quality > 0.8: need positive > 0.6 + Rivalry, so with Rivalry=0: positive > 0.6
	rel.BondStrength = 1.0
	rel.Trust = 1.0
	rel.Affection = 1.0
	rel.Rivalry = 0.0
	desc := rel.GetDescription()
	if desc != "best friends" {
		t.Errorf("Expected 'best friends', got '%s' (quality=%f)", desc, rel.GetRelationshipQuality())
	}

	// Good friends (quality > 0.6)
	// Need ((avg - rivalry + 1) / 2) > 0.6 and <= 0.8
	rel.BondStrength = 0.5
	rel.Trust = 0.5
	rel.Affection = 0.5
	rel.Rivalry = 0.0
	desc = rel.GetDescription()
	if desc != "good friends" {
		t.Errorf("Expected 'good friends', got '%s' (quality=%f)", desc, rel.GetRelationshipQuality())
	}

	// Friends (quality > 0.4)
	rel.BondStrength = 0.3
	rel.Trust = 0.3
	rel.Affection = 0.3
	rel.Rivalry = 0.2
	desc = rel.GetDescription()
	if desc != "friends" {
		t.Errorf("Expected 'friends', got '%s' (quality=%f)", desc, rel.GetRelationshipQuality())
	}

	// Acquaintances (quality <= 0.4)
	rel.BondStrength = 0.0
	rel.Trust = 0.0
	rel.Affection = 0.0
	rel.Rivalry = 0.5
	desc = rel.GetDescription()
	if desc != "acquaintances" {
		t.Errorf("Expected 'acquaintances', got '%s' (quality=%f)", desc, rel.GetRelationshipQuality())
	}
}

// TestGetDescriptionRival tests rival descriptions
func TestGetDescriptionRival(t *testing.T) {
	rel := NewRelationship(types.PetID("pet_123"), types.RelationshipRival)

	// Bitter rivals (high rivalry)
	rel.Rivalry = 0.8
	desc := rel.GetDescription()
	if desc != "bitter rivals" {
		t.Errorf("Expected 'bitter rivals', got '%s'", desc)
	}

	// Rivals
	rel.Rivalry = 0.5
	desc = rel.GetDescription()
	if desc != "rivals" {
		t.Errorf("Expected 'rivals', got '%s'", desc)
	}

	// Competitive (low rivalry)
	rel.Rivalry = 0.2
	desc = rel.GetDescription()
	if desc != "competitive" {
		t.Errorf("Expected 'competitive', got '%s'", desc)
	}
}

// TestGetDescriptionMate tests mate descriptions
func TestGetDescriptionMate(t *testing.T) {
	rel := NewRelationship(types.PetID("pet_123"), types.RelationshipMate)

	// Devoted mates (quality > 0.8)
	rel.BondStrength = 1.0
	rel.Trust = 1.0
	rel.Affection = 1.0
	rel.Rivalry = 0.0
	desc := rel.GetDescription()
	if desc != "devoted mates" {
		t.Errorf("Expected 'devoted mates', got '%s' (quality=%f)", desc, rel.GetRelationshipQuality())
	}

	// Mates (quality > 0.6)
	rel.BondStrength = 0.5
	rel.Trust = 0.5
	rel.Affection = 0.5
	rel.Rivalry = 0.0
	desc = rel.GetDescription()
	if desc != "mates" {
		t.Errorf("Expected 'mates', got '%s' (quality=%f)", desc, rel.GetRelationshipQuality())
	}

	// Partners (quality <= 0.6)
	rel.BondStrength = 0.2
	rel.Trust = 0.2
	rel.Affection = 0.2
	rel.Rivalry = 0.0
	desc = rel.GetDescription()
	if desc != "partners" {
		t.Errorf("Expected 'partners', got '%s' (quality=%f)", desc, rel.GetRelationshipQuality())
	}
}

// TestGetDescriptionFamily tests family descriptions
func TestGetDescriptionFamily(t *testing.T) {
	relParent := NewRelationship(types.PetID("pet_parent"), types.RelationshipParent)
	if relParent.GetDescription() != "parent" {
		t.Errorf("Expected 'parent', got '%s'", relParent.GetDescription())
	}

	relOffspring := NewRelationship(types.PetID("pet_child"), types.RelationshipOffspring)
	if relOffspring.GetDescription() != "offspring" {
		t.Errorf("Expected 'offspring', got '%s'", relOffspring.GetDescription())
	}
}

// TestRelationshipDecay tests relationship decay over time
func TestRelationshipDecay(t *testing.T) {
	rel := NewRelationship(types.PetID("pet_123"), types.RelationshipFriend)

	// Set high values
	rel.BondStrength = 0.8
	rel.Affection = 0.8
	rel.Trust = 0.8

	// Simulate time passing (set last interaction to 2 days ago)
	rel.LastInteraction = time.Now().Add(-48 * time.Hour)

	initialBond := rel.BondStrength
	initialAffection := rel.Affection
	initialTrust := rel.Trust

	rel.Decay(1.0) // 1 unit of time

	if rel.BondStrength >= initialBond {
		t.Error("BondStrength should decay after no interaction")
	}

	if rel.Affection >= initialAffection {
		t.Error("Affection should decay after no interaction")
	}

	if rel.Trust >= initialTrust {
		t.Error("Trust should decay after no interaction")
	}
}

// TestRelationshipNoDecayRecent tests no decay for recent interactions
func TestRelationshipNoDecayRecent(t *testing.T) {
	rel := NewRelationship(types.PetID("pet_123"), types.RelationshipFriend)

	// Set values
	rel.BondStrength = 0.8
	rel.Affection = 0.8
	rel.Trust = 0.8

	// Last interaction is now (recent)
	rel.LastInteraction = time.Now()

	initialBond := rel.BondStrength
	initialAffection := rel.Affection
	initialTrust := rel.Trust

	rel.Decay(1.0)

	// Should not decay with recent interaction
	if rel.BondStrength != initialBond {
		t.Error("BondStrength should not decay with recent interaction")
	}

	if rel.Affection != initialAffection {
		t.Error("Affection should not decay with recent interaction")
	}

	if rel.Trust != initialTrust {
		t.Error("Trust should not decay with recent interaction")
	}
}

// =============== SocialRelationships Tests ===============

// TestNewSocialRelationships tests creation
func TestNewSocialRelationships(t *testing.T) {
	sr := NewSocialRelationships(10)

	if sr == nil {
		t.Fatal("NewSocialRelationships returned nil")
	}

	if sr.MaxRelationships != 10 {
		t.Errorf("Expected MaxRelationships 10, got %d", sr.MaxRelationships)
	}

	if sr.Relationships == nil {
		t.Error("Relationships map should not be nil")
	}

	if len(sr.Relationships) != 0 {
		t.Errorf("Expected 0 initial relationships, got %d", len(sr.Relationships))
	}
}

// TestAddRelationship tests adding relationships
func TestAddRelationship(t *testing.T) {
	sr := NewSocialRelationships(10)

	petID := types.PetID("pet_123")
	rel := sr.AddRelationship(petID, types.RelationshipFriend)

	if rel == nil {
		t.Fatal("AddRelationship returned nil")
	}

	if rel.PetID != petID {
		t.Errorf("Expected PetID %s, got %s", petID, rel.PetID)
	}

	if sr.GetRelationshipCount() != 1 {
		t.Errorf("Expected 1 relationship, got %d", sr.GetRelationshipCount())
	}
}

// TestAddRelationshipLimit tests max relationship limit
func TestAddRelationshipLimit(t *testing.T) {
	sr := NewSocialRelationships(3)

	// Add relationships up to limit
	sr.AddRelationship(types.PetID("pet_1"), types.RelationshipFriend)
	sr.AddRelationship(types.PetID("pet_2"), types.RelationshipFriend)
	sr.AddRelationship(types.PetID("pet_3"), types.RelationshipFriend)

	// Strengthen one relationship
	if rel, exists := sr.GetRelationship(types.PetID("pet_2")); exists {
		rel.BondStrength = 0.9
	}

	// Add another - should remove weakest
	sr.AddRelationship(types.PetID("pet_4"), types.RelationshipFriend)

	// Should still only have 3 relationships
	if sr.GetRelationshipCount() != 3 {
		t.Errorf("Expected 3 relationships after exceeding limit, got %d", sr.GetRelationshipCount())
	}

	// The strongest relationship (pet_2) should still exist
	if !sr.HasRelationshipWith(types.PetID("pet_2")) {
		t.Error("Strongest relationship should not be removed")
	}
}

// TestAddRelationshipLimitPreservesFamily tests family relationships are preserved
func TestAddRelationshipLimitPreservesFamily(t *testing.T) {
	sr := NewSocialRelationships(2)

	// Add a parent relationship
	sr.AddRelationship(types.PetID("parent"), types.RelationshipParent)

	// Add a friend with weak bond
	sr.AddRelationship(types.PetID("friend"), types.RelationshipFriend)
	if rel, exists := sr.GetRelationship(types.PetID("friend")); exists {
		rel.BondStrength = 0.1 // Very weak
	}

	// Add another relationship - should remove the weak friend, not the parent
	sr.AddRelationship(types.PetID("new_friend"), types.RelationshipFriend)

	// Parent should still exist
	if !sr.HasRelationshipWith(types.PetID("parent")) {
		t.Error("Parent relationship should not be removed")
	}
}

// TestGetRelationship tests retrieving relationships
func TestGetRelationship(t *testing.T) {
	sr := NewSocialRelationships(10)

	petID := types.PetID("pet_123")
	sr.AddRelationship(petID, types.RelationshipFriend)

	rel, exists := sr.GetRelationship(petID)
	if !exists {
		t.Error("Relationship should exist")
	}
	if rel == nil {
		t.Error("Relationship should not be nil")
	}

	// Non-existent relationship
	_, exists = sr.GetRelationship(types.PetID("nonexistent"))
	if exists {
		t.Error("Non-existent relationship should return false")
	}
}

// TestUpdateRelationship tests updating relationships
func TestUpdateRelationship(t *testing.T) {
	sr := NewSocialRelationships(10)

	petID := types.PetID("pet_123")
	sr.AddRelationship(petID, types.RelationshipFriend)

	rel, _ := sr.GetRelationship(petID)
	initialBond := rel.BondStrength

	sr.UpdateRelationship(petID, 1.0, 100.0)

	rel, _ = sr.GetRelationship(petID)
	if rel.BondStrength <= initialBond {
		t.Error("UpdateRelationship should increase bond for positive quality")
	}
}

// TestUpdateRelationshipNonExistent tests updating non-existent relationship
func TestUpdateRelationshipNonExistent(t *testing.T) {
	sr := NewSocialRelationships(10)

	// Should not panic on non-existent relationship
	sr.UpdateRelationship(types.PetID("nonexistent"), 1.0, 100.0)

	// Count should still be 0
	if sr.GetRelationshipCount() != 0 {
		t.Error("Updating non-existent relationship should not create one")
	}
}

// TestUpdateAll tests updating all relationships
func TestUpdateAll(t *testing.T) {
	sr := NewSocialRelationships(10)

	sr.AddRelationship(types.PetID("pet_1"), types.RelationshipFriend)
	sr.AddRelationship(types.PetID("pet_2"), types.RelationshipFriend)

	// Set old interaction times
	for _, rel := range sr.Relationships {
		rel.LastInteraction = time.Now().Add(-48 * time.Hour)
		rel.BondStrength = 0.8
	}

	sr.UpdateAll(1.0)

	// All should have decayed
	for id, rel := range sr.Relationships {
		if rel.BondStrength >= 0.8 {
			t.Errorf("Relationship %s should have decayed", id)
		}
	}
}

// TestGetClosestFriends tests getting closest friends
func TestGetClosestFriends(t *testing.T) {
	sr := NewSocialRelationships(10)

	// Add friends with different bond strengths
	sr.AddRelationship(types.PetID("weak_friend"), types.RelationshipFriend)
	sr.AddRelationship(types.PetID("medium_friend"), types.RelationshipFriend)
	sr.AddRelationship(types.PetID("best_friend"), types.RelationshipFriend)

	if rel, exists := sr.GetRelationship(types.PetID("weak_friend")); exists {
		rel.BondStrength = 0.2
	}
	if rel, exists := sr.GetRelationship(types.PetID("medium_friend")); exists {
		rel.BondStrength = 0.5
	}
	if rel, exists := sr.GetRelationship(types.PetID("best_friend")); exists {
		rel.BondStrength = 0.9
	}

	// Get top 2 friends
	closest := sr.GetClosestFriends(2)

	if len(closest) != 2 {
		t.Errorf("Expected 2 closest friends, got %d", len(closest))
	}

	// First should be best friend
	if closest[0].PetID != types.PetID("best_friend") {
		t.Error("First closest friend should be best_friend")
	}

	// Second should be medium friend
	if closest[1].PetID != types.PetID("medium_friend") {
		t.Error("Second closest friend should be medium_friend")
	}
}

// TestGetClosestFriendsExcludesNonFriends tests that only friends are returned
func TestGetClosestFriendsExcludesNonFriends(t *testing.T) {
	sr := NewSocialRelationships(10)

	sr.AddRelationship(types.PetID("friend"), types.RelationshipFriend)
	sr.AddRelationship(types.PetID("rival"), types.RelationshipRival)
	sr.AddRelationship(types.PetID("parent"), types.RelationshipParent)

	closest := sr.GetClosestFriends(10)

	// Should only return the friend
	if len(closest) != 1 {
		t.Errorf("Expected 1 friend, got %d", len(closest))
	}

	if closest[0].PetID != types.PetID("friend") {
		t.Error("Should only return friend type relationships")
	}
}

// TestGetClosestFriendsEmpty tests with no friends
func TestGetClosestFriendsEmpty(t *testing.T) {
	sr := NewSocialRelationships(10)

	closest := sr.GetClosestFriends(5)

	if len(closest) != 0 {
		t.Errorf("Expected 0 friends with empty relationships, got %d", len(closest))
	}
}

// TestGetRelationshipCount tests counting relationships
func TestGetRelationshipCount(t *testing.T) {
	sr := NewSocialRelationships(10)

	if sr.GetRelationshipCount() != 0 {
		t.Error("Initial count should be 0")
	}

	sr.AddRelationship(types.PetID("pet_1"), types.RelationshipFriend)
	if sr.GetRelationshipCount() != 1 {
		t.Error("Count should be 1 after adding one relationship")
	}

	sr.AddRelationship(types.PetID("pet_2"), types.RelationshipRival)
	if sr.GetRelationshipCount() != 2 {
		t.Error("Count should be 2 after adding two relationships")
	}
}

// TestHasRelationshipWith tests checking relationship existence
func TestHasRelationshipWith(t *testing.T) {
	sr := NewSocialRelationships(10)

	petID := types.PetID("pet_123")
	sr.AddRelationship(petID, types.RelationshipFriend)

	if !sr.HasRelationshipWith(petID) {
		t.Error("Should have relationship with pet_123")
	}

	if sr.HasRelationshipWith(types.PetID("nonexistent")) {
		t.Error("Should not have relationship with nonexistent")
	}
}

// TestGetAverageBondStrength tests average calculation
func TestGetAverageBondStrength(t *testing.T) {
	sr := NewSocialRelationships(10)

	// Empty should return 0
	if sr.GetAverageBondStrength() != 0.0 {
		t.Error("Empty relationships should have average 0")
	}

	sr.AddRelationship(types.PetID("pet_1"), types.RelationshipFriend)
	sr.AddRelationship(types.PetID("pet_2"), types.RelationshipFriend)

	if rel, exists := sr.GetRelationship(types.PetID("pet_1")); exists {
		rel.BondStrength = 0.4
	}
	if rel, exists := sr.GetRelationship(types.PetID("pet_2")); exists {
		rel.BondStrength = 0.8
	}

	avg := sr.GetAverageBondStrength()
	expected := (0.4 + 0.8) / 2.0

	// Use tolerance for floating point comparison
	tolerance := 0.0001
	if avg < expected-tolerance || avg > expected+tolerance {
		t.Errorf("Expected average %f, got %f", expected, avg)
	}
}

// TestGetAverageBondStrengthSingle tests with single relationship
func TestGetAverageBondStrengthSingle(t *testing.T) {
	sr := NewSocialRelationships(10)

	sr.AddRelationship(types.PetID("pet_1"), types.RelationshipFriend)
	if rel, exists := sr.GetRelationship(types.PetID("pet_1")); exists {
		rel.BondStrength = 0.6
	}

	avg := sr.GetAverageBondStrength()
	if avg != 0.6 {
		t.Errorf("Expected average 0.6, got %f", avg)
	}
}

// =============== Helper Function Tests ===============

// TestClamp tests the clamp helper function
func TestClamp(t *testing.T) {
	tests := []struct {
		value    float64
		min      float64
		max      float64
		expected float64
	}{
		{0.5, 0.0, 1.0, 0.5},  // Within range
		{-0.5, 0.0, 1.0, 0.0}, // Below min
		{1.5, 0.0, 1.0, 1.0},  // Above max
		{0.0, 0.0, 1.0, 0.0},  // At min
		{1.0, 0.0, 1.0, 1.0},  // At max
	}

	for _, tt := range tests {
		result := clamp(tt.value, tt.min, tt.max)
		if result != tt.expected {
			t.Errorf("clamp(%f, %f, %f) = %f, want %f", tt.value, tt.min, tt.max, result, tt.expected)
		}
	}
}

// =============== Integration Tests ===============

// TestRelationshipLifecycle tests a complete relationship lifecycle
func TestRelationshipLifecycle(t *testing.T) {
	sr := NewSocialRelationships(10)

	petID := types.PetID("friend_pet")

	// Create relationship
	rel := sr.AddRelationship(petID, types.RelationshipFriend)
	if rel == nil {
		t.Fatal("Failed to create relationship")
	}

	// Initial state - quality is based on starting values
	// BondStrength=0.3, Trust=0.5, Affection=0.3, Rivalry=0.1
	// Quality = ((0.3+0.5+0.3)/3 - 0.1 + 1) / 2 = (0.367 - 0.1 + 1) / 2 = 0.633
	initialQuality := rel.GetRelationshipQuality()
	initialDesc := rel.GetDescription()

	// Build relationship through positive interactions
	for i := 0; i < 20; i++ {
		sr.UpdateRelationship(petID, 0.8, float64(i*10))
		rel.AddSharedExperience("Positive interaction", float64(i*10), 0.5)
	}

	// Should now have improved quality
	finalQuality := rel.GetRelationshipQuality()
	if finalQuality <= initialQuality {
		t.Errorf("Quality should have improved from %f to higher, got %f", initialQuality, finalQuality)
	}

	// Description should have improved
	finalDesc := rel.GetDescription()
	// Verify the relationship description matches quality expectations
	if finalQuality > 0.8 && finalDesc != "best friends" {
		t.Errorf("With quality %f, expected 'best friends', got '%s'", finalQuality, finalDesc)
	}

	// Log for debugging
	t.Logf("Initial: quality=%f desc='%s', Final: quality=%f desc='%s'",
		initialQuality, initialDesc, finalQuality, finalDesc)

	// Verify statistics
	if rel.TotalInteractions != 20 {
		t.Errorf("Expected 20 interactions, got %d", rel.TotalInteractions)
	}

	if len(rel.History) != 20 {
		t.Errorf("Expected 20 history entries, got %d", len(rel.History))
	}
}

// TestMultipleRelationshipTypes tests managing multiple relationship types
func TestMultipleRelationshipTypes(t *testing.T) {
	sr := NewSocialRelationships(10)

	sr.AddRelationship(types.PetID("friend"), types.RelationshipFriend)
	sr.AddRelationship(types.PetID("rival"), types.RelationshipRival)
	sr.AddRelationship(types.PetID("mate"), types.RelationshipMate)
	sr.AddRelationship(types.PetID("parent"), types.RelationshipParent)
	sr.AddRelationship(types.PetID("child"), types.RelationshipOffspring)

	if sr.GetRelationshipCount() != 5 {
		t.Errorf("Expected 5 relationships, got %d", sr.GetRelationshipCount())
	}

	// Verify types
	tests := []struct {
		petID    types.PetID
		expected types.RelationshipType
	}{
		{types.PetID("friend"), types.RelationshipFriend},
		{types.PetID("rival"), types.RelationshipRival},
		{types.PetID("mate"), types.RelationshipMate},
		{types.PetID("parent"), types.RelationshipParent},
		{types.PetID("child"), types.RelationshipOffspring},
	}

	for _, tt := range tests {
		rel, exists := sr.GetRelationship(tt.petID)
		if !exists {
			t.Errorf("Relationship with %s should exist", tt.petID)
			continue
		}
		if rel.Type != tt.expected {
			t.Errorf("Relationship type for %s should be %v, got %v", tt.petID, tt.expected, rel.Type)
		}
	}
}
