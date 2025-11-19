package cloud

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/Michael-W-Ellison/gochi/pkg/types"
)

// Friend represents a friend relationship
type Friend struct {
	UserID      string
	Username    string
	FriendSince time.Time
	PetCount    int
	LastOnline  time.Time
	IsPremium   bool
}

// FriendRequest represents a pending friend request
type FriendRequest struct {
	FromUserID string
	FromUsername string
	ToUserID   string
	SentAt     time.Time
	Message    string
}

// Achievement represents a game achievement
type Achievement struct {
	ID          string
	Name        string
	Description string
	IconURL     string
	Points      int
	UnlockedAt  *time.Time
	Progress    float64 // 0-1
	MaxProgress float64
}

// LeaderboardEntry represents an entry in a leaderboard
type LeaderboardEntry struct {
	Rank      int
	UserID    string
	Username  string
	Score     int64
	PetName   string
	UpdatedAt time.Time
}

// SocialService handles social features
type SocialService struct {
	mu sync.RWMutex

	authManager *AuthManager

	// Local caches
	friends        map[string]*Friend
	friendRequests []*FriendRequest
	achievements   map[string]*Achievement
	lastUpdate     time.Time
}

// NewSocialService creates a new social service
func NewSocialService(authManager *AuthManager) *SocialService {
	return &SocialService{
		authManager:    authManager,
		friends:        make(map[string]*Friend),
		friendRequests: make([]*FriendRequest, 0),
		achievements:   make(map[string]*Achievement),
		lastUpdate:     time.Now(),
	}
}

// SendFriendRequest sends a friend request to another user
func (ss *SocialService) SendFriendRequest(username string, message string) error {
	ss.mu.Lock()
	defer ss.mu.Unlock()

	if !ss.authManager.IsLoggedIn() {
		return errors.New("not logged in")
	}

	currentUser, err := ss.authManager.GetCurrentUser()
	if err != nil {
		return err
	}

	// Check if already friends
	for _, friend := range ss.friends {
		if friend.Username == username {
			return errors.New("already friends with this user")
		}
	}

	// In a real implementation, this would call an API
	// For testing/mock purposes, we simulate receiving a request from the target user
	// This allows single-user testing
	request := &FriendRequest{
		FromUserID:   username, // Simulate as if it came from them
		FromUsername: username,
		ToUserID:     currentUser.ID,
		SentAt:       time.Now(),
		Message:      message,
	}

	ss.friendRequests = append(ss.friendRequests, request)

	return nil
}

// AcceptFriendRequest accepts a friend request
func (ss *SocialService) AcceptFriendRequest(fromUsername string) error {
	ss.mu.Lock()
	defer ss.mu.Unlock()

	if !ss.authManager.IsLoggedIn() {
		return errors.New("not logged in")
	}

	// Find the request
	var requestIndex = -1
	var request *FriendRequest
	for i, req := range ss.friendRequests {
		if req.FromUsername == fromUsername {
			requestIndex = i
			request = req
			break
		}
	}

	if requestIndex == -1 {
		return errors.New("friend request not found")
	}

	// Add as friend
	friend := &Friend{
		UserID:      request.FromUserID,
		Username:    request.FromUsername,
		FriendSince: time.Now(),
		LastOnline:  time.Now(),
	}

	ss.friends[friend.UserID] = friend

	// Remove request
	ss.friendRequests = append(ss.friendRequests[:requestIndex], ss.friendRequests[requestIndex+1:]...)

	return nil
}

// RejectFriendRequest rejects a friend request
func (ss *SocialService) RejectFriendRequest(fromUsername string) error {
	ss.mu.Lock()
	defer ss.mu.Unlock()

	if !ss.authManager.IsLoggedIn() {
		return errors.New("not logged in")
	}

	// Find and remove the request
	for i, req := range ss.friendRequests {
		if req.FromUsername == fromUsername {
			ss.friendRequests = append(ss.friendRequests[:i], ss.friendRequests[i+1:]...)
			return nil
		}
	}

	return errors.New("friend request not found")
}

// RemoveFriend removes a friend
func (ss *SocialService) RemoveFriend(username string) error {
	ss.mu.Lock()
	defer ss.mu.Unlock()

	if !ss.authManager.IsLoggedIn() {
		return errors.New("not logged in")
	}

	// Find and remove friend
	for userID, friend := range ss.friends {
		if friend.Username == username {
			delete(ss.friends, userID)
			return nil
		}
	}

	return errors.New("friend not found")
}

// GetFriends returns the list of friends
func (ss *SocialService) GetFriends() []*Friend {
	ss.mu.RLock()
	defer ss.mu.RUnlock()

	friends := make([]*Friend, 0, len(ss.friends))
	for _, friend := range ss.friends {
		friends = append(friends, friend)
	}

	return friends
}

// GetFriendRequests returns pending friend requests
func (ss *SocialService) GetFriendRequests() []*FriendRequest {
	ss.mu.RLock()
	defer ss.mu.RUnlock()

	requests := make([]*FriendRequest, len(ss.friendRequests))
	copy(requests, ss.friendRequests)

	return requests
}

// UnlockAchievement unlocks an achievement
func (ss *SocialService) UnlockAchievement(achievementID string) error {
	ss.mu.Lock()
	defer ss.mu.Unlock()

	if !ss.authManager.IsLoggedIn() {
		return errors.New("not logged in")
	}

	achievement, exists := ss.achievements[achievementID]
	if !exists {
		return fmt.Errorf("achievement %s not found", achievementID)
	}

	if achievement.UnlockedAt != nil {
		return errors.New("achievement already unlocked")
	}

	now := time.Now()
	achievement.UnlockedAt = &now
	achievement.Progress = 1.0

	return nil
}

// UpdateAchievementProgress updates progress towards an achievement
func (ss *SocialService) UpdateAchievementProgress(achievementID string, progress float64) error {
	ss.mu.Lock()
	defer ss.mu.Unlock()

	if !ss.authManager.IsLoggedIn() {
		return errors.New("not logged in")
	}

	achievement, exists := ss.achievements[achievementID]
	if !exists {
		return fmt.Errorf("achievement %s not found", achievementID)
	}

	if achievement.UnlockedAt != nil {
		return errors.New("achievement already unlocked")
	}

	achievement.Progress = progress
	if progress >= 1.0 {
		now := time.Now()
		achievement.UnlockedAt = &now
	}

	return nil
}

// GetAchievements returns all achievements
func (ss *SocialService) GetAchievements() []*Achievement {
	ss.mu.RLock()
	defer ss.mu.RUnlock()

	achievements := make([]*Achievement, 0, len(ss.achievements))
	for _, achievement := range ss.achievements {
		achievements = append(achievements, achievement)
	}

	return achievements
}

// GetUnlockedAchievements returns only unlocked achievements
func (ss *SocialService) GetUnlockedAchievements() []*Achievement {
	ss.mu.RLock()
	defer ss.mu.RUnlock()

	unlocked := make([]*Achievement, 0)
	for _, achievement := range ss.achievements {
		if achievement.UnlockedAt != nil {
			unlocked = append(unlocked, achievement)
		}
	}

	return unlocked
}

// GetTotalAchievementPoints returns total points from unlocked achievements
func (ss *SocialService) GetTotalAchievementPoints() int {
	ss.mu.RLock()
	defer ss.mu.RUnlock()

	total := 0
	for _, achievement := range ss.achievements {
		if achievement.UnlockedAt != nil {
			total += achievement.Points
		}
	}

	return total
}

// InitializeAchievements sets up the achievement system
func (ss *SocialService) InitializeAchievements() {
	ss.mu.Lock()
	defer ss.mu.Unlock()

	// Define achievements
	achievements := []*Achievement{
		{
			ID:          "first_pet",
			Name:        "First Pet",
			Description: "Create your first pet",
			Points:      10,
			MaxProgress: 1,
		},
		{
			ID:          "happy_pet",
			Name:        "Happy Pet",
			Description: "Raise pet happiness to 100%",
			Points:      15,
			MaxProgress: 1,
		},
		{
			ID:          "pet_master",
			Name:        "Pet Master",
			Description: "Own 10 pets",
			Points:      50,
			MaxProgress: 10,
		},
		{
			ID:          "long_term_care",
			Name:        "Long-term Care",
			Description: "Keep a pet alive for 30 days",
			Points:      30,
			MaxProgress: 30,
		},
		{
			ID:          "social_butterfly",
			Name:        "Social Butterfly",
			Description: "Add 5 friends",
			Points:      20,
			MaxProgress: 5,
		},
		{
			ID:          "breeder",
			Name:        "Breeder",
			Description: "Successfully breed 5 pets",
			Points:      25,
			MaxProgress: 5,
		},
		{
			ID:          "explorer",
			Name:        "Explorer",
			Description: "Discover all locations",
			Points:      40,
			MaxProgress: 11, // Number of locations
		},
		{
			ID:          "dedication",
			Name:        "Dedication",
			Description: "Play for 100 hours",
			Points:      100,
			MaxProgress: 100,
		},
	}

	for _, achievement := range achievements {
		ss.achievements[achievement.ID] = achievement
	}
}

// GetLeaderboard retrieves a leaderboard
func (ss *SocialService) GetLeaderboard(category string, limit int) ([]*LeaderboardEntry, error) {
	ss.mu.RLock()
	defer ss.mu.RUnlock()

	if !ss.authManager.IsLoggedIn() {
		return nil, errors.New("not logged in")
	}

	// In a real implementation, this would call an API
	// For now, return mock data
	entries := make([]*LeaderboardEntry, 0)

	// Mock some entries for testing
	if len(entries) == 0 {
		entries = append(entries, &LeaderboardEntry{
			Rank:      1,
			UserID:    "user1",
			Username:  "TopPlayer",
			Score:     1000,
			PetName:   "Champion",
			UpdatedAt: time.Now(),
		})
	}

	if limit > 0 && len(entries) > limit {
		entries = entries[:limit]
	}

	return entries, nil
}

// SubmitScore submits a score to a leaderboard
func (ss *SocialService) SubmitScore(category string, score int64, petID types.PetID) error {
	ss.mu.Lock()
	defer ss.mu.Unlock()

	if !ss.authManager.IsLoggedIn() {
		return errors.New("not logged in")
	}

	// In a real implementation, this would call an API
	// For now, just return success
	return nil
}

// GetStats returns social service statistics
func (ss *SocialService) GetStats() map[string]interface{} {
	ss.mu.RLock()
	defer ss.mu.RUnlock()

	return map[string]interface{}{
		"friend_count":          len(ss.friends),
		"pending_requests":      len(ss.friendRequests),
		"achievement_count":     len(ss.achievements),
		"unlocked_achievements": len(ss.GetUnlockedAchievements()),
		"total_points":          ss.GetTotalAchievementPoints(),
	}
}
