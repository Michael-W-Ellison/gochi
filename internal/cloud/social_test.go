package cloud

import (
	"testing"

	"github.com/Michael-W-Ellison/gochi/pkg/types"
)

func TestNewSocialService(t *testing.T) {
	provider := NewMockAuthProvider()
	am := NewAuthManager(provider)
	ss := NewSocialService(am)

	if ss == nil {
		t.Fatal("NewSocialService returned nil")
	}
}

func TestSendFriendRequest(t *testing.T) {
	provider := NewMockAuthProvider()
	am := NewAuthManager(provider)
	ss := NewSocialService(am)

	// Register and login
	creds := &Credentials{
		Username: "testuser",
		Password: "password123",
		Email:    "test@example.com",
	}
	am.Register(creds)

	// Send friend request
	err := ss.SendFriendRequest("friend1", "Let's be friends!")
	if err != nil {
		t.Fatalf("SendFriendRequest failed: %v", err)
	}

	// Check requests
	requests := ss.GetFriendRequests()
	if len(requests) != 1 {
		t.Errorf("Expected 1 friend request, got %d", len(requests))
	}
}

func TestAcceptFriendRequest(t *testing.T) {
	provider := NewMockAuthProvider()
	am := NewAuthManager(provider)
	ss := NewSocialService(am)

	// Register and login
	creds := &Credentials{
		Username: "testuser",
		Password: "password123",
		Email:    "test@example.com",
	}
	am.Register(creds)

	// Send friend request
	ss.SendFriendRequest("friend1", "Hello")

	// Accept request
	err := ss.AcceptFriendRequest("friend1")
	if err != nil {
		t.Fatalf("AcceptFriendRequest failed: %v", err)
	}

	// Check friends
	friends := ss.GetFriends()
	if len(friends) != 1 {
		t.Errorf("Expected 1 friend, got %d", len(friends))
	}

	// Request should be removed
	requests := ss.GetFriendRequests()
	if len(requests) != 0 {
		t.Errorf("Expected 0 requests after accepting, got %d", len(requests))
	}
}

func TestRejectFriendRequest(t *testing.T) {
	provider := NewMockAuthProvider()
	am := NewAuthManager(provider)
	ss := NewSocialService(am)

	// Register and login
	creds := &Credentials{
		Username: "testuser",
		Password: "password123",
		Email:    "test@example.com",
	}
	am.Register(creds)

	// Send friend request
	ss.SendFriendRequest("friend1", "Hello")

	// Reject request
	err := ss.RejectFriendRequest("friend1")
	if err != nil {
		t.Fatalf("RejectFriendRequest failed: %v", err)
	}

	// Should have no friends and no requests
	friends := ss.GetFriends()
	requests := ss.GetFriendRequests()

	if len(friends) != 0 {
		t.Error("Expected no friends after rejecting")
	}

	if len(requests) != 0 {
		t.Error("Expected no requests after rejecting")
	}
}

func TestRemoveFriend(t *testing.T) {
	provider := NewMockAuthProvider()
	am := NewAuthManager(provider)
	ss := NewSocialService(am)

	// Register and login
	creds := &Credentials{
		Username: "testuser",
		Password: "password123",
		Email:    "test@example.com",
	}
	am.Register(creds)

	// Add friend
	ss.SendFriendRequest("friend1", "Hello")
	ss.AcceptFriendRequest("friend1")

	// Remove friend
	err := ss.RemoveFriend("friend1")
	if err != nil {
		t.Fatalf("RemoveFriend failed: %v", err)
	}

	// Check friends
	friends := ss.GetFriends()
	if len(friends) != 0 {
		t.Errorf("Expected 0 friends after removing, got %d", len(friends))
	}
}

func TestInitializeAchievements(t *testing.T) {
	provider := NewMockAuthProvider()
	am := NewAuthManager(provider)
	ss := NewSocialService(am)

	// Initialize achievements
	ss.InitializeAchievements()

	achievements := ss.GetAchievements()
	if len(achievements) == 0 {
		t.Error("Expected achievements to be initialized")
	}

	// Check for specific achievements
	foundFirstPet := false
	for _, ach := range achievements {
		if ach.ID == "first_pet" {
			foundFirstPet = true
			break
		}
	}

	if !foundFirstPet {
		t.Error("Expected 'first_pet' achievement")
	}
}

func TestUnlockAchievement(t *testing.T) {
	provider := NewMockAuthProvider()
	am := NewAuthManager(provider)
	ss := NewSocialService(am)

	// Register and login
	creds := &Credentials{
		Username: "testuser",
		Password: "password123",
		Email:    "test@example.com",
	}
	am.Register(creds)

	// Initialize achievements
	ss.InitializeAchievements()

	// Unlock achievement
	err := ss.UnlockAchievement("first_pet")
	if err != nil {
		t.Fatalf("UnlockAchievement failed: %v", err)
	}

	// Check unlocked
	unlocked := ss.GetUnlockedAchievements()
	if len(unlocked) != 1 {
		t.Errorf("Expected 1 unlocked achievement, got %d", len(unlocked))
	}

	// Try to unlock again
	err = ss.UnlockAchievement("first_pet")
	if err == nil {
		t.Error("Expected error when unlocking already unlocked achievement")
	}
}

func TestUpdateAchievementProgress(t *testing.T) {
	provider := NewMockAuthProvider()
	am := NewAuthManager(provider)
	ss := NewSocialService(am)

	// Register and login
	creds := &Credentials{
		Username: "testuser",
		Password: "password123",
		Email:    "test@example.com",
	}
	am.Register(creds)

	// Initialize achievements
	ss.InitializeAchievements()

	// Update progress
	err := ss.UpdateAchievementProgress("pet_master", 0.5)
	if err != nil {
		t.Fatalf("UpdateAchievementProgress failed: %v", err)
	}

	// Check progress
	achievements := ss.GetAchievements()
	var petMaster *Achievement
	for _, ach := range achievements {
		if ach.ID == "pet_master" {
			petMaster = ach
			break
		}
	}

	if petMaster == nil {
		t.Fatal("Could not find pet_master achievement")
	}

	if petMaster.Progress != 0.5 {
		t.Errorf("Expected progress 0.5, got %v", petMaster.Progress)
	}

	// Should not be unlocked yet
	if petMaster.UnlockedAt != nil {
		t.Error("Achievement should not be unlocked at 50% progress")
	}

	// Complete it
	ss.UpdateAchievementProgress("pet_master", 1.0)

	// Should be unlocked now
	achievements = ss.GetAchievements()
	for _, ach := range achievements {
		if ach.ID == "pet_master" {
			if ach.UnlockedAt == nil {
				t.Error("Achievement should be unlocked at 100% progress")
			}
			break
		}
	}
}

func TestGetTotalAchievementPoints(t *testing.T) {
	provider := NewMockAuthProvider()
	am := NewAuthManager(provider)
	ss := NewSocialService(am)

	// Register and login
	creds := &Credentials{
		Username: "testuser",
		Password: "password123",
		Email:    "test@example.com",
	}
	am.Register(creds)

	// Initialize achievements
	ss.InitializeAchievements()

	// Unlock some achievements
	ss.UnlockAchievement("first_pet")    // 10 points
	ss.UnlockAchievement("happy_pet")    // 15 points

	points := ss.GetTotalAchievementPoints()
	if points != 25 {
		t.Errorf("Expected 25 points, got %d", points)
	}
}

func TestGetLeaderboard(t *testing.T) {
	provider := NewMockAuthProvider()
	am := NewAuthManager(provider)
	ss := NewSocialService(am)

	// Register and login
	creds := &Credentials{
		Username: "testuser",
		Password: "password123",
		Email:    "test@example.com",
	}
	am.Register(creds)

	// Get leaderboard
	entries, err := ss.GetLeaderboard("happiness", 10)
	if err != nil {
		t.Fatalf("GetLeaderboard failed: %v", err)
	}

	// Should return at least mock data
	if len(entries) == 0 {
		t.Log("Leaderboard is empty (expected for mock implementation)")
	}
}

func TestSocialServiceNotLoggedIn(t *testing.T) {
	provider := NewMockAuthProvider()
	am := NewAuthManager(provider)
	ss := NewSocialService(am)

	// Try operations without logging in
	err := ss.SendFriendRequest("friend1", "Hello")
	if err == nil {
		t.Error("Expected error when sending friend request without login")
	}

	err = ss.UnlockAchievement("first_pet")
	if err == nil {
		t.Error("Expected error when unlocking achievement without login")
	}

	_, err = ss.GetLeaderboard("test", 10)
	if err == nil {
		t.Error("Expected error when getting leaderboard without login")
	}
}

func TestGetSocialStats(t *testing.T) {
	provider := NewMockAuthProvider()
	am := NewAuthManager(provider)
	ss := NewSocialService(am)

	// Register and login
	creds := &Credentials{
		Username: "testuser",
		Password: "password123",
		Email:    "test@example.com",
	}
	am.Register(creds)

	// Initialize and unlock some achievements
	ss.InitializeAchievements()
	ss.UnlockAchievement("first_pet")

	// Add friend
	ss.SendFriendRequest("friend1", "Hello")
	ss.AcceptFriendRequest("friend1")

	// Get stats
	stats := ss.GetStats()

	if stats["friend_count"] != 1 {
		t.Errorf("Expected 1 friend in stats, got %v", stats["friend_count"])
	}

	if stats["unlocked_achievements"] != 1 {
		t.Errorf("Expected 1 unlocked achievement in stats, got %v", stats["unlocked_achievements"])
	}

	if stats["total_points"] != 10 {
		t.Errorf("Expected 10 points in stats, got %v", stats["total_points"])
	}
}

func TestSendDuplicateFriendRequest(t *testing.T) {
	provider := NewMockAuthProvider()
	am := NewAuthManager(provider)
	ss := NewSocialService(am)

	// Register and login
	creds := &Credentials{
		Username: "testuser",
		Password: "password123",
		Email:    "test@example.com",
	}
	am.Register(creds)

	// Add friend
	ss.SendFriendRequest("friend1", "Hello")
	ss.AcceptFriendRequest("friend1")

	// Try to send request to already-friend
	err := ss.SendFriendRequest("friend1", "Hello again")
	if err == nil {
		t.Error("Expected error when sending request to existing friend")
	}
}

func TestSubmitScore(t *testing.T) {
	provider := NewMockAuthProvider()
	am := NewAuthManager(provider)
	ss := NewSocialService(am)

	// Register and login
	creds := &Credentials{
		Username: "testuser",
		Password: "password123",
		Email:    "test@example.com",
	}
	am.Register(creds)

	// Submit score
	err := ss.SubmitScore("daily_care", 1000, types.PetID("pet1"))
	if err != nil {
		t.Fatalf("SubmitScore failed: %v", err)
	}
}

func TestSubmitScoreNotLoggedIn(t *testing.T) {
	provider := NewMockAuthProvider()
	am := NewAuthManager(provider)
	ss := NewSocialService(am)

	// Try to submit score without logging in
	err := ss.SubmitScore("daily_care", 1000, types.PetID("pet1"))
	if err == nil {
		t.Error("Expected error when submitting score without logging in")
	}
}
