package social

import (
	"testing"
	"time"

	"github.com/Michael-W-Ellison/gochi/pkg/types"
)

func TestCloudServiceStatusString(t *testing.T) {
	tests := []struct {
		status   CloudServiceStatus
		expected string
	}{
		{CloudStatusDisconnected, "Disconnected"},
		{CloudStatusConnecting, "Connecting"},
		{CloudStatusConnected, "Connected"},
		{CloudStatusError, "Error"},
	}

	for _, tt := range tests {
		if got := tt.status.String(); got != tt.expected {
			t.Errorf("CloudServiceStatus.String() = %v, want %v", got, tt.expected)
		}
	}
}

func TestPrivacyLevelString(t *testing.T) {
	tests := []struct {
		level    PrivacyLevel
		expected string
	}{
		{PrivacyPublic, "Public"},
		{PrivacyFriendsOnly, "Friends Only"},
		{PrivacyPrivate, "Private"},
	}

	for _, tt := range tests {
		if got := tt.level.String(); got != tt.expected {
			t.Errorf("PrivacyLevel.String() = %v, want %v", got, tt.expected)
		}
	}
}

func TestSocialInteractionTypeString(t *testing.T) {
	tests := []struct {
		interactionType SocialInteractionType
		expected        string
	}{
		{SocialInteractionWave, "Wave"},
		{SocialInteractionGreet, "Greet"},
		{SocialInteractionPlay, "Play"},
		{SocialInteractionGift, "Gift"},
		{SocialInteractionTrade, "Trade"},
		{SocialInteractionBattle, "Battle"},
		{SocialInteractionVisit, "Visit"},
	}

	for _, tt := range tests {
		if got := tt.interactionType.String(); got != tt.expected {
			t.Errorf("SocialInteractionType.String() = %v, want %v", got, tt.expected)
		}
	}
}

func TestFriendRequestStatusString(t *testing.T) {
	tests := []struct {
		status   FriendRequestStatus
		expected string
	}{
		{FriendRequestPending, "Pending"},
		{FriendRequestAccepted, "Accepted"},
		{FriendRequestDeclined, "Declined"},
		{FriendRequestExpired, "Expired"},
	}

	for _, tt := range tests {
		if got := tt.status.String(); got != tt.expected {
			t.Errorf("FriendRequestStatus.String() = %v, want %v", got, tt.expected)
		}
	}
}

func TestDefaultCloudServiceConfig(t *testing.T) {
	config := DefaultCloudServiceConfig()

	if config.NearbyRadius != 500.0 {
		t.Errorf("Expected nearby radius 500.0, got %f", config.NearbyRadius)
	}

	if config.ScanInterval != 30*time.Second {
		t.Errorf("Expected scan interval 30s, got %v", config.ScanInterval)
	}

	if config.HeartbeatInterval != 60*time.Second {
		t.Errorf("Expected heartbeat interval 60s, got %v", config.HeartbeatInterval)
	}

	if !config.EnableNearby {
		t.Error("Expected EnableNearby to be true")
	}

	if !config.EnableSocialFeed {
		t.Error("Expected EnableSocialFeed to be true")
	}

	if !config.EnableEvents {
		t.Error("Expected EnableEvents to be true")
	}
}

func TestNewCloudService(t *testing.T) {
	cs := NewCloudService(nil)

	if cs.Config == nil {
		t.Error("Expected config to be set")
	}

	if cs.Status != CloudStatusDisconnected {
		t.Errorf("Expected status Disconnected, got %v", cs.Status)
	}

	if cs.Friends == nil {
		t.Error("Expected Friends map to be initialized")
	}

	if cs.NearbyPets == nil {
		t.Error("Expected NearbyPets slice to be initialized")
	}

	if cs.SocialFeed == nil {
		t.Error("Expected SocialFeed to be initialized")
	}
}

func TestCloudServiceConnect(t *testing.T) {
	cs := NewCloudService(nil)
	provider := NewMockSocialCloudProvider()
	cs.SetProvider(provider)

	err := cs.Connect("test_user")
	if err != nil {
		t.Errorf("Connect failed: %v", err)
	}

	if cs.Status != CloudStatusConnected {
		t.Errorf("Expected status Connected, got %v", cs.Status)
	}

	if cs.UserID != "test_user" {
		t.Errorf("Expected user ID 'test_user', got '%s'", cs.UserID)
	}

	if cs.SessionID == "" {
		t.Error("Expected session ID to be set")
	}
}

func TestCloudServiceConnectNoProvider(t *testing.T) {
	cs := NewCloudService(nil)

	err := cs.Connect("test_user")
	if err == nil {
		t.Error("Expected error when no provider configured")
	}
}

func TestCloudServiceDisconnect(t *testing.T) {
	cs := NewCloudService(nil)
	provider := NewMockSocialCloudProvider()
	cs.SetProvider(provider)
	cs.Connect("test_user")

	cs.Disconnect()

	if cs.Status != CloudStatusDisconnected {
		t.Errorf("Expected status Disconnected, got %v", cs.Status)
	}

	if cs.SessionID != "" {
		t.Error("Expected session ID to be cleared")
	}
}

func TestUpdateLocalPet(t *testing.T) {
	cs := NewCloudService(nil)
	provider := NewMockSocialCloudProvider()
	cs.SetProvider(provider)
	cs.Connect("test_user")

	profile := &PetProfile{
		PetID:   types.PetID("pet_123"),
		Name:    "Fluffy",
		Species: "Dog",
	}

	err := cs.UpdateLocalPet(profile)
	if err != nil {
		t.Errorf("UpdateLocalPet failed: %v", err)
	}

	if cs.LocalPet.Name != "Fluffy" {
		t.Errorf("Expected name 'Fluffy', got '%s'", cs.LocalPet.Name)
	}

	// Check it was uploaded to provider
	storedProfile, err := provider.GetProfile(types.PetID("pet_123"))
	if err != nil {
		t.Errorf("Profile not found in provider: %v", err)
	}
	if storedProfile.Name != "Fluffy" {
		t.Errorf("Expected stored name 'Fluffy', got '%s'", storedProfile.Name)
	}
}

func TestUpdateLocation(t *testing.T) {
	cs := NewCloudService(nil)
	provider := NewMockSocialCloudProvider()
	cs.SetProvider(provider)
	cs.Connect("test_user")

	err := cs.UpdateLocation(40.7128, -74.0060, 10.0)
	if err != nil {
		t.Errorf("UpdateLocation failed: %v", err)
	}

	if cs.LocalLocation == nil {
		t.Fatal("Expected LocalLocation to be set")
	}

	if cs.LocalLocation.Latitude != 40.7128 {
		t.Errorf("Expected latitude 40.7128, got %f", cs.LocalLocation.Latitude)
	}

	if cs.LocalLocation.Longitude != -74.0060 {
		t.Errorf("Expected longitude -74.0060, got %f", cs.LocalLocation.Longitude)
	}

	if cs.LocalLocation.Accuracy != 10.0 {
		t.Errorf("Expected accuracy 10.0, got %f", cs.LocalLocation.Accuracy)
	}
}

func TestGetNearbyPets(t *testing.T) {
	cs := NewCloudService(nil)
	provider := NewMockSocialCloudProvider()
	cs.SetProvider(provider)
	cs.Connect("test_user")

	// Add mock nearby pets
	provider.AddMockNearbyPet(&PetProfile{
		PetID: types.PetID("nearby_1"),
		Name:  "Rex",
	}, 100.0, 45.0)

	provider.AddMockNearbyPet(&PetProfile{
		PetID: types.PetID("nearby_2"),
		Name:  "Whiskers",
	}, 200.0, 90.0)

	// Update location to trigger scan
	cs.UpdateLocation(40.7128, -74.0060, 10.0)

	// Manually scan (in real scenario this is automatic)
	cs.scanForNearbyPets()

	nearbyPets := cs.GetNearbyPets()
	if len(nearbyPets) != 2 {
		t.Errorf("Expected 2 nearby pets, got %d", len(nearbyPets))
	}
}

func TestGetNearbyPetsSorted(t *testing.T) {
	cs := NewCloudService(nil)
	provider := NewMockSocialCloudProvider()
	cs.SetProvider(provider)
	cs.Connect("test_user")

	provider.AddMockNearbyPet(&PetProfile{
		PetID: types.PetID("far_pet"),
		Name:  "Far",
	}, 300.0, 0.0)

	provider.AddMockNearbyPet(&PetProfile{
		PetID: types.PetID("close_pet"),
		Name:  "Close",
	}, 50.0, 0.0)

	cs.UpdateLocation(40.7128, -74.0060, 10.0)
	cs.scanForNearbyPets()

	sorted := cs.GetNearbyPetsSorted()
	if len(sorted) < 2 {
		t.Fatal("Expected at least 2 pets")
	}

	if sorted[0].Distance > sorted[1].Distance {
		t.Error("Pets should be sorted by distance ascending")
	}
}

func TestFindNearbyPet(t *testing.T) {
	cs := NewCloudService(nil)
	provider := NewMockSocialCloudProvider()
	cs.SetProvider(provider)
	cs.Connect("test_user")

	provider.AddMockNearbyPet(&PetProfile{
		PetID: types.PetID("findme"),
		Name:  "FindMe",
	}, 100.0, 0.0)

	cs.UpdateLocation(40.7128, -74.0060, 10.0)
	cs.scanForNearbyPets()

	found := cs.FindNearbyPet(types.PetID("findme"))
	if found == nil {
		t.Error("Expected to find nearby pet")
	}

	if found.Profile.Name != "FindMe" {
		t.Errorf("Expected name 'FindMe', got '%s'", found.Profile.Name)
	}

	notFound := cs.FindNearbyPet(types.PetID("nonexistent"))
	if notFound != nil {
		t.Error("Should not find nonexistent pet")
	}
}

func TestSendFriendRequest(t *testing.T) {
	cs := NewCloudService(nil)
	provider := NewMockSocialCloudProvider()
	cs.SetProvider(provider)
	cs.Connect("test_user")

	err := cs.SendFriendRequest(types.PetID("friend_pet"), "Let's be friends!")
	if err != nil {
		t.Errorf("SendFriendRequest failed: %v", err)
	}

	requests := cs.GetPendingFriendRequests()
	if len(requests) != 1 {
		t.Errorf("Expected 1 pending request, got %d", len(requests))
	}

	if requests[0].Message != "Let's be friends!" {
		t.Errorf("Expected message 'Let's be friends!', got '%s'", requests[0].Message)
	}
}

func TestSendFriendRequestNotConnected(t *testing.T) {
	cs := NewCloudService(nil)

	err := cs.SendFriendRequest(types.PetID("friend_pet"), "Hello")
	if err == nil {
		t.Error("Expected error when not connected")
	}
}

func TestAcceptFriendRequest(t *testing.T) {
	cs := NewCloudService(nil)
	provider := NewMockSocialCloudProvider()
	cs.SetProvider(provider)
	cs.Connect("test_user")

	// Send a friend request
	cs.SendFriendRequest(types.PetID("friend_pet"), "Hello")

	// Get the request ID
	requests := cs.GetPendingFriendRequests()
	if len(requests) == 0 {
		t.Fatal("Expected pending request")
	}

	// Accept it
	err := cs.AcceptFriendRequest(requests[0].ID)
	if err != nil {
		t.Errorf("AcceptFriendRequest failed: %v", err)
	}
}

func TestDeclineFriendRequest(t *testing.T) {
	cs := NewCloudService(nil)
	provider := NewMockSocialCloudProvider()
	cs.SetProvider(provider)
	cs.Connect("test_user")

	cs.SendFriendRequest(types.PetID("friend_pet"), "Hello")
	requests := cs.GetPendingFriendRequests()

	err := cs.DeclineFriendRequest(requests[0].ID)
	if err != nil {
		t.Errorf("DeclineFriendRequest failed: %v", err)
	}
}

func TestGetFriends(t *testing.T) {
	cs := NewCloudService(nil)
	provider := NewMockSocialCloudProvider()
	cs.SetProvider(provider)
	cs.Connect("test_user")

	// Add friends directly to provider
	provider.Friends[types.PetID("friend1")] = &PetProfile{
		PetID:    types.PetID("friend1"),
		Name:     "Friend1",
		IsOnline: true,
	}
	provider.Friends[types.PetID("friend2")] = &PetProfile{
		PetID:    types.PetID("friend2"),
		Name:     "Friend2",
		IsOnline: false,
	}

	err := cs.RefreshFriends()
	if err != nil {
		t.Errorf("RefreshFriends failed: %v", err)
	}

	friends := cs.GetFriends()
	if len(friends) != 2 {
		t.Errorf("Expected 2 friends, got %d", len(friends))
	}
}

func TestGetOnlineFriends(t *testing.T) {
	cs := NewCloudService(nil)
	provider := NewMockSocialCloudProvider()
	cs.SetProvider(provider)
	cs.Connect("test_user")

	provider.Friends[types.PetID("friend1")] = &PetProfile{
		PetID:    types.PetID("friend1"),
		Name:     "Friend1",
		IsOnline: true,
	}
	provider.Friends[types.PetID("friend2")] = &PetProfile{
		PetID:    types.PetID("friend2"),
		Name:     "Friend2",
		IsOnline: false,
	}

	cs.RefreshFriends()

	online := cs.GetOnlineFriends()
	if len(online) != 1 {
		t.Errorf("Expected 1 online friend, got %d", len(online))
	}

	if online[0].Name != "Friend1" {
		t.Errorf("Expected Friend1 to be online, got '%s'", online[0].Name)
	}
}

func TestRemoveFriend(t *testing.T) {
	cs := NewCloudService(nil)
	provider := NewMockSocialCloudProvider()
	cs.SetProvider(provider)
	cs.Connect("test_user")

	provider.Friends[types.PetID("friend1")] = &PetProfile{PetID: types.PetID("friend1")}
	cs.RefreshFriends()

	if !cs.IsFriend(types.PetID("friend1")) {
		t.Error("Expected friend1 to be a friend")
	}

	err := cs.RemoveFriend(types.PetID("friend1"))
	if err != nil {
		t.Errorf("RemoveFriend failed: %v", err)
	}

	if cs.IsFriend(types.PetID("friend1")) {
		t.Error("Expected friend1 to be removed")
	}
}

func TestIsFriend(t *testing.T) {
	cs := NewCloudService(nil)
	provider := NewMockSocialCloudProvider()
	cs.SetProvider(provider)
	cs.Connect("test_user")

	provider.Friends[types.PetID("friend1")] = &PetProfile{PetID: types.PetID("friend1")}
	cs.RefreshFriends()

	if !cs.IsFriend(types.PetID("friend1")) {
		t.Error("Expected friend1 to be a friend")
	}

	if cs.IsFriend(types.PetID("stranger")) {
		t.Error("Expected stranger to not be a friend")
	}
}

func TestInitiateSocialInteraction(t *testing.T) {
	cs := NewCloudService(nil)
	provider := NewMockSocialCloudProvider()
	cs.SetProvider(provider)
	cs.Connect("test_user")

	cs.UpdateLocalPet(&PetProfile{
		PetID: types.PetID("my_pet"),
		Name:  "MyPet",
	})

	interaction, err := cs.InitiateSocialInteraction(types.PetID("other_pet"), SocialInteractionWave)
	if err != nil {
		t.Errorf("InitiateSocialInteraction failed: %v", err)
	}

	if interaction.Type != SocialInteractionWave {
		t.Errorf("Expected Wave interaction, got %v", interaction.Type)
	}

	if interaction.InitiatorID != types.PetID("my_pet") {
		t.Errorf("Expected initiator 'my_pet', got '%s'", interaction.InitiatorID)
	}

	if interaction.TargetID != types.PetID("other_pet") {
		t.Errorf("Expected target 'other_pet', got '%s'", interaction.TargetID)
	}

	// Check interaction is in history
	history := cs.GetInteractionHistory()
	if len(history) != 1 {
		t.Errorf("Expected 1 interaction in history, got %d", len(history))
	}
}

func TestInitiateSocialInteractionNoLocalPet(t *testing.T) {
	cs := NewCloudService(nil)
	provider := NewMockSocialCloudProvider()
	cs.SetProvider(provider)
	cs.Connect("test_user")

	_, err := cs.InitiateSocialInteraction(types.PetID("other_pet"), SocialInteractionWave)
	if err == nil {
		t.Error("Expected error when no local pet configured")
	}
}

func TestWaveAtPet(t *testing.T) {
	cs := NewCloudService(nil)
	provider := NewMockSocialCloudProvider()
	cs.SetProvider(provider)
	cs.Connect("test_user")

	cs.UpdateLocalPet(&PetProfile{PetID: types.PetID("my_pet")})

	err := cs.WaveAtPet(types.PetID("other_pet"))
	if err != nil {
		t.Errorf("WaveAtPet failed: %v", err)
	}
}

func TestPlayWithPet(t *testing.T) {
	cs := NewCloudService(nil)
	provider := NewMockSocialCloudProvider()
	cs.SetProvider(provider)
	cs.Connect("test_user")

	cs.UpdateLocalPet(&PetProfile{PetID: types.PetID("my_pet")})

	interaction, err := cs.PlayWithPet(types.PetID("other_pet"))
	if err != nil {
		t.Errorf("PlayWithPet failed: %v", err)
	}

	if interaction.Type != SocialInteractionPlay {
		t.Errorf("Expected Play interaction, got %v", interaction.Type)
	}
}

func TestVisitPet(t *testing.T) {
	cs := NewCloudService(nil)
	provider := NewMockSocialCloudProvider()
	cs.SetProvider(provider)
	cs.Connect("test_user")

	cs.UpdateLocalPet(&PetProfile{PetID: types.PetID("my_pet")})

	interaction, err := cs.VisitPet(types.PetID("other_pet"))
	if err != nil {
		t.Errorf("VisitPet failed: %v", err)
	}

	if interaction.Type != SocialInteractionVisit {
		t.Errorf("Expected Visit interaction, got %v", interaction.Type)
	}
}

func TestSocialFeed(t *testing.T) {
	cs := NewCloudService(nil)
	provider := NewMockSocialCloudProvider()
	cs.SetProvider(provider)
	cs.Connect("test_user")

	// Post to feed
	item, err := cs.PostToFeed("Hello world!", nil)
	if err != nil {
		t.Errorf("PostToFeed failed: %v", err)
	}

	if item.Content != "Hello world!" {
		t.Errorf("Expected content 'Hello world!', got '%s'", item.Content)
	}

	// Refresh feed
	err = cs.RefreshSocialFeed()
	if err != nil {
		t.Errorf("RefreshSocialFeed failed: %v", err)
	}

	feed := cs.GetSocialFeed()
	if len(feed.Items) != 1 {
		t.Errorf("Expected 1 feed item, got %d", len(feed.Items))
	}
}

func TestLikeFeedItem(t *testing.T) {
	cs := NewCloudService(nil)
	provider := NewMockSocialCloudProvider()
	cs.SetProvider(provider)
	cs.Connect("test_user")

	item, _ := cs.PostToFeed("Test post", nil)

	err := cs.LikeFeedItem(item.ID)
	if err != nil {
		t.Errorf("LikeFeedItem failed: %v", err)
	}
}

func TestEvents(t *testing.T) {
	cs := NewCloudService(nil)
	provider := NewMockSocialCloudProvider()
	cs.SetProvider(provider)
	cs.Connect("test_user")

	// Add mock event
	provider.AddMockEvent(&SocialEvent{
		ID:          "event_1",
		Name:        "Pet Party",
		Description: "A fun event",
		StartTime:   time.Now(),
		EndTime:     time.Now().Add(time.Hour),
		IsActive:    true,
	})

	err := cs.RefreshEvents()
	if err != nil {
		t.Errorf("RefreshEvents failed: %v", err)
	}

	events := cs.GetActiveEvents()
	if len(events) != 1 {
		t.Errorf("Expected 1 event, got %d", len(events))
	}

	if events[0].Name != "Pet Party" {
		t.Errorf("Expected event name 'Pet Party', got '%s'", events[0].Name)
	}
}

func TestGetNearbyEvents(t *testing.T) {
	cs := NewCloudService(nil)
	provider := NewMockSocialCloudProvider()
	cs.SetProvider(provider)
	cs.Connect("test_user")

	cs.UpdateLocation(40.7128, -74.0060, 10.0)

	// Add event at same location
	provider.AddMockEvent(&SocialEvent{
		ID:       "event_nearby",
		Name:     "Nearby Event",
		Location: &Location{Latitude: 40.7128, Longitude: -74.0060},
		IsActive: true,
	})

	// Add event far away
	provider.AddMockEvent(&SocialEvent{
		ID:       "event_far",
		Name:     "Far Event",
		Location: &Location{Latitude: 41.0, Longitude: -75.0},
		IsActive: true,
	})

	cs.RefreshEvents()

	nearby := cs.GetNearbyEvents(1000) // Within 1km
	if len(nearby) != 1 {
		t.Errorf("Expected 1 nearby event, got %d", len(nearby))
	}
}

func TestJoinLeaveEvent(t *testing.T) {
	cs := NewCloudService(nil)
	provider := NewMockSocialCloudProvider()
	cs.SetProvider(provider)
	cs.Connect("test_user")

	err := cs.JoinEvent("event_1")
	if err != nil {
		t.Errorf("JoinEvent failed: %v", err)
	}

	err = cs.LeaveEvent("event_1")
	if err != nil {
		t.Errorf("LeaveEvent failed: %v", err)
	}
}

func TestLeaderboard(t *testing.T) {
	cs := NewCloudService(nil)
	provider := NewMockSocialCloudProvider()
	cs.SetProvider(provider)
	cs.Connect("test_user")

	provider.AddMockLeaderboard(&Leaderboard{
		ID:   "high_scores",
		Name: "High Scores",
		Entries: []*LeaderboardEntry{
			{Rank: 1, PetID: "pet_1", PetName: "Champ", Score: 1000},
			{Rank: 2, PetID: "pet_2", PetName: "Runner", Score: 900},
		},
	})

	leaderboard, err := cs.GetLeaderboard("high_scores")
	if err != nil {
		t.Errorf("GetLeaderboard failed: %v", err)
	}

	if leaderboard.Name != "High Scores" {
		t.Errorf("Expected name 'High Scores', got '%s'", leaderboard.Name)
	}

	if len(leaderboard.Entries) != 2 {
		t.Errorf("Expected 2 entries, got %d", len(leaderboard.Entries))
	}
}

func TestSubmitScore(t *testing.T) {
	cs := NewCloudService(nil)
	provider := NewMockSocialCloudProvider()
	cs.SetProvider(provider)
	cs.Connect("test_user")

	err := cs.SubmitScore("high_scores", 500.0)
	if err != nil {
		t.Errorf("SubmitScore failed: %v", err)
	}
}

func TestSearchPets(t *testing.T) {
	cs := NewCloudService(nil)
	provider := NewMockSocialCloudProvider()
	cs.SetProvider(provider)
	cs.Connect("test_user")

	provider.Profiles[types.PetID("pet_1")] = &PetProfile{PetID: "pet_1", Name: "Fluffy"}
	provider.Profiles[types.PetID("pet_2")] = &PetProfile{PetID: "pet_2", Name: "Fluff"}
	provider.Profiles[types.PetID("pet_3")] = &PetProfile{PetID: "pet_3", Name: "Rex"}

	results, err := cs.SearchPets("Fluff", 10)
	if err != nil {
		t.Errorf("SearchPets failed: %v", err)
	}

	if len(results) != 2 {
		t.Errorf("Expected 2 results for 'Fluff', got %d", len(results))
	}
}

func TestGetPetProfile(t *testing.T) {
	cs := NewCloudService(nil)
	provider := NewMockSocialCloudProvider()
	cs.SetProvider(provider)
	cs.Connect("test_user")

	provider.Profiles[types.PetID("target")] = &PetProfile{
		PetID: "target",
		Name:  "Target Pet",
	}

	profile, err := cs.GetPetProfile(types.PetID("target"))
	if err != nil {
		t.Errorf("GetPetProfile failed: %v", err)
	}

	if profile.Name != "Target Pet" {
		t.Errorf("Expected name 'Target Pet', got '%s'", profile.Name)
	}
}

func TestGetStatus(t *testing.T) {
	cs := NewCloudService(nil)

	if cs.GetStatus() != CloudStatusDisconnected {
		t.Error("Expected initial status to be Disconnected")
	}

	provider := NewMockSocialCloudProvider()
	cs.SetProvider(provider)
	cs.Connect("test_user")

	if cs.GetStatus() != CloudStatusConnected {
		t.Error("Expected status to be Connected after connecting")
	}
}

func TestIsConnected(t *testing.T) {
	cs := NewCloudService(nil)

	if cs.IsConnected() {
		t.Error("Expected IsConnected to be false initially")
	}

	provider := NewMockSocialCloudProvider()
	cs.SetProvider(provider)
	cs.Connect("test_user")

	if !cs.IsConnected() {
		t.Error("Expected IsConnected to be true after connecting")
	}

	cs.Disconnect()

	if cs.IsConnected() {
		t.Error("Expected IsConnected to be false after disconnecting")
	}
}

func TestCalculateDistance(t *testing.T) {
	// Test distance between two points (NYC to LA approximately 3944 km)
	lat1, lon1 := 40.7128, -74.0060 // NYC
	lat2, lon2 := 34.0522, -118.2437 // LA

	distance := calculateDistance(lat1, lon1, lat2, lon2)

	// Should be approximately 3944 km (3,944,000 meters)
	// Allow 5% tolerance
	expectedDistance := 3944000.0
	tolerance := expectedDistance * 0.05

	if distance < expectedDistance-tolerance || distance > expectedDistance+tolerance {
		t.Errorf("Expected distance ~%f meters, got %f", expectedDistance, distance)
	}
}

func TestCalculateBearing(t *testing.T) {
	// Test bearing from NYC to LA (should be approximately 273 degrees - roughly west)
	lat1, lon1 := 40.7128, -74.0060 // NYC
	lat2, lon2 := 34.0522, -118.2437 // LA

	bearing := calculateBearing(lat1, lon1, lat2, lon2)

	// Should be approximately 273 degrees (west-southwest)
	// Allow 10 degree tolerance
	if bearing < 260 || bearing > 290 {
		t.Errorf("Expected bearing ~273 degrees, got %f", bearing)
	}
}

func TestGenerateSessionID(t *testing.T) {
	id1 := generateSessionID()
	id2 := generateSessionID()

	if id1 == id2 {
		t.Error("Session IDs should be unique")
	}

	if len(id1) != 32 {
		t.Errorf("Expected session ID length 32, got %d", len(id1))
	}
}

func TestGenerateInteractionID(t *testing.T) {
	id1 := generateInteractionID()
	id2 := generateInteractionID()

	if id1 == id2 {
		t.Error("Interaction IDs should be unique")
	}

	if len(id1) < 20 {
		t.Error("Interaction ID seems too short")
	}
}

func TestContainsIgnoreCaseHelper(t *testing.T) {
	tests := []struct {
		s        string
		substr   string
		expected bool
	}{
		{"Hello World", "hello", true},
		{"Hello World", "WORLD", true},
		{"Hello World", "foo", false},
		{"", "", true},
		{"abc", "", true},
		{"", "abc", false},
		{"Fluffy", "fluff", true},
		{"Fluffy", "FLUFFY", true},
	}

	for _, tt := range tests {
		result := containsIgnoreCase(tt.s, tt.substr)
		if result != tt.expected {
			t.Errorf("containsIgnoreCase(%q, %q) = %v, want %v", tt.s, tt.substr, result, tt.expected)
		}
	}
}

func TestMockProviderMethods(t *testing.T) {
	provider := NewMockSocialCloudProvider()

	// Test connection
	provider.Connect("user", "session")
	if !provider.IsConnected() {
		t.Error("Expected provider to be connected")
	}

	provider.Disconnect()
	if provider.IsConnected() {
		t.Error("Expected provider to be disconnected")
	}

	// Reconnect for other tests
	provider.Connect("user", "session")

	// Test profile operations
	profile := &PetProfile{
		PetID: types.PetID("test_pet"),
		Name:  "TestPet",
	}
	provider.UpdateProfile(profile)

	retrieved, err := provider.GetProfile(types.PetID("test_pet"))
	if err != nil {
		t.Errorf("GetProfile failed: %v", err)
	}
	if retrieved.Name != "TestPet" {
		t.Errorf("Expected name 'TestPet', got '%s'", retrieved.Name)
	}

	// Test profile not found
	_, err = provider.GetProfile(types.PetID("nonexistent"))
	if err == nil {
		t.Error("Expected error for nonexistent profile")
	}
}

func TestStatusChangeCallback(t *testing.T) {
	cs := NewCloudService(nil)
	provider := NewMockSocialCloudProvider()
	cs.SetProvider(provider)

	var statusChanges []CloudServiceStatus
	cs.OnStatusChange = func(status CloudServiceStatus) {
		statusChanges = append(statusChanges, status)
	}

	cs.Connect("test_user")

	// Should have connecting and connected
	if len(statusChanges) < 2 {
		t.Errorf("Expected at least 2 status changes, got %d", len(statusChanges))
	}

	if statusChanges[0] != CloudStatusConnecting {
		t.Errorf("Expected first status to be Connecting, got %v", statusChanges[0])
	}

	if statusChanges[1] != CloudStatusConnected {
		t.Errorf("Expected second status to be Connected, got %v", statusChanges[1])
	}
}

func TestNearbyPetCallbacks(t *testing.T) {
	cs := NewCloudService(nil)
	provider := NewMockSocialCloudProvider()
	cs.SetProvider(provider)
	cs.Connect("test_user")
	cs.UpdateLocation(40.7128, -74.0060, 10.0)

	var foundPets []types.PetID
	cs.OnNearbyPetFound = func(pet *NearbyPet) {
		foundPets = append(foundPets, pet.Profile.PetID)
	}

	// Add a nearby pet
	provider.AddMockNearbyPet(&PetProfile{
		PetID: types.PetID("new_pet"),
		Name:  "NewPet",
	}, 100.0, 0.0)

	// Trigger scan
	cs.scanForNearbyPets()

	if len(foundPets) != 1 {
		t.Errorf("Expected 1 found pet callback, got %d", len(foundPets))
	}

	if foundPets[0] != types.PetID("new_pet") {
		t.Errorf("Expected found pet 'new_pet', got '%s'", foundPets[0])
	}
}

func TestInteractionHistoryLimit(t *testing.T) {
	cs := NewCloudService(nil)
	provider := NewMockSocialCloudProvider()
	cs.SetProvider(provider)
	cs.Connect("test_user")

	cs.UpdateLocalPet(&PetProfile{PetID: types.PetID("my_pet")})

	// Add more than 100 interactions
	for i := 0; i < 110; i++ {
		cs.InitiateSocialInteraction(types.PetID("other_pet"), SocialInteractionWave)
	}

	history := cs.GetInteractionHistory()
	if len(history) > 100 {
		t.Errorf("Expected history to be capped at 100, got %d", len(history))
	}
}
