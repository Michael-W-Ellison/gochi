package social

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"math"
	"sort"
	"sync"
	"time"

	"github.com/Michael-W-Ellison/gochi/pkg/types"
)

// CloudServiceStatus represents the cloud service connection state
type CloudServiceStatus int

const (
	CloudStatusDisconnected CloudServiceStatus = iota
	CloudStatusConnecting
	CloudStatusConnected
	CloudStatusError
)

// String returns the string representation of CloudServiceStatus
func (s CloudServiceStatus) String() string {
	return [...]string{"Disconnected", "Connecting", "Connected", "Error"}[s]
}

// Location represents a geographic coordinate
type Location struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	Accuracy  float64 `json:"accuracy"` // Meters
	Timestamp time.Time `json:"timestamp"`
}

// PetProfile represents a public-facing pet profile for social features
type PetProfile struct {
	PetID       types.PetID `json:"pet_id"`
	OwnerID     string      `json:"owner_id"`
	Name        string      `json:"name"`
	Species     string      `json:"species"`
	Age         float64     `json:"age"` // In game days
	Personality string      `json:"personality"`
	AvatarURL   string      `json:"avatar_url"`
	Bio         string      `json:"bio"`
	Badges      []string    `json:"badges"`
	IsOnline    bool        `json:"is_online"`
	LastSeen    time.Time   `json:"last_seen"`
	Privacy     PrivacyLevel `json:"privacy"`
}

// PrivacyLevel controls what information is shared
type PrivacyLevel int

const (
	PrivacyPublic PrivacyLevel = iota
	PrivacyFriendsOnly
	PrivacyPrivate
)

// String returns the string representation of PrivacyLevel
func (p PrivacyLevel) String() string {
	return [...]string{"Public", "Friends Only", "Private"}[p]
}

// NearbyPet represents a pet discovered in the vicinity
type NearbyPet struct {
	Profile   *PetProfile `json:"profile"`
	Distance  float64     `json:"distance"` // Meters
	Direction float64     `json:"direction"` // Degrees from north
	LastSeen  time.Time   `json:"last_seen"`
	Signal    float64     `json:"signal"` // Signal strength 0-1
}

// SocialInteractionType represents types of social interactions
type SocialInteractionType int

const (
	SocialInteractionWave SocialInteractionType = iota
	SocialInteractionGreet
	SocialInteractionPlay
	SocialInteractionGift
	SocialInteractionTrade
	SocialInteractionBattle
	SocialInteractionVisit
)

// String returns the string representation of SocialInteractionType
func (s SocialInteractionType) String() string {
	return [...]string{"Wave", "Greet", "Play", "Gift", "Trade", "Battle", "Visit"}[s]
}

// SocialInteraction represents an interaction between pets
type SocialInteraction struct {
	ID            string                `json:"id"`
	Type          SocialInteractionType `json:"type"`
	InitiatorID   types.PetID           `json:"initiator_id"`
	TargetID      types.PetID           `json:"target_id"`
	Timestamp     time.Time             `json:"timestamp"`
	Outcome       string                `json:"outcome"`
	ExperienceGain float64              `json:"experience_gain"`
	Data          map[string]interface{} `json:"data"`
}

// FriendRequest represents a friend request between pet owners
type FriendRequest struct {
	ID          string    `json:"id"`
	FromPetID   types.PetID `json:"from_pet_id"`
	ToPetID     types.PetID `json:"to_pet_id"`
	Message     string    `json:"message"`
	SentAt      time.Time `json:"sent_at"`
	Status      FriendRequestStatus `json:"status"`
	RespondedAt time.Time `json:"responded_at"`
}

// FriendRequestStatus represents the status of a friend request
type FriendRequestStatus int

const (
	FriendRequestPending FriendRequestStatus = iota
	FriendRequestAccepted
	FriendRequestDeclined
	FriendRequestExpired
)

// String returns the string representation of FriendRequestStatus
func (s FriendRequestStatus) String() string {
	return [...]string{"Pending", "Accepted", "Declined", "Expired"}[s]
}

// SocialFeed represents the activity feed
type SocialFeed struct {
	Items     []*FeedItem `json:"items"`
	LastCheck time.Time   `json:"last_check"`
}

// FeedItem represents an item in the social feed
type FeedItem struct {
	ID        string    `json:"id"`
	Type      string    `json:"type"`
	PetID     types.PetID `json:"pet_id"`
	Content   string    `json:"content"`
	Timestamp time.Time `json:"timestamp"`
	Likes     int       `json:"likes"`
	HasLiked  bool      `json:"has_liked"`
	Data      map[string]interface{} `json:"data"`
}

// SocialEvent represents a social event that pets can participate in
type SocialEvent struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Type        string    `json:"type"`
	Location    *Location `json:"location,omitempty"`
	StartTime   time.Time `json:"start_time"`
	EndTime     time.Time `json:"end_time"`
	MaxParticipants int   `json:"max_participants"`
	Participants []types.PetID `json:"participants"`
	Rewards     []string  `json:"rewards"`
	IsActive    bool      `json:"is_active"`
}

// Leaderboard represents a competitive leaderboard
type Leaderboard struct {
	ID      string            `json:"id"`
	Name    string            `json:"name"`
	Type    string            `json:"type"`
	Period  string            `json:"period"` // daily, weekly, monthly, all-time
	Entries []*LeaderboardEntry `json:"entries"`
	UpdatedAt time.Time       `json:"updated_at"`
}

// LeaderboardEntry represents an entry in the leaderboard
type LeaderboardEntry struct {
	Rank    int         `json:"rank"`
	PetID   types.PetID `json:"pet_id"`
	PetName string      `json:"pet_name"`
	Score   float64     `json:"score"`
	Change  int         `json:"change"` // Position change from previous period
}

// CloudServiceConfig holds configuration for the cloud service
type CloudServiceConfig struct {
	ServerURL        string        `json:"server_url"`
	APIKey           string        `json:"api_key"`
	NearbyRadius     float64       `json:"nearby_radius"` // Meters
	ScanInterval     time.Duration `json:"scan_interval"`
	HeartbeatInterval time.Duration `json:"heartbeat_interval"`
	EnableNearby     bool          `json:"enable_nearby"`
	EnableSocialFeed bool          `json:"enable_social_feed"`
	EnableEvents     bool          `json:"enable_events"`
}

// DefaultCloudServiceConfig returns default configuration
func DefaultCloudServiceConfig() *CloudServiceConfig {
	return &CloudServiceConfig{
		ServerURL:        "https://api.gochi.example.com",
		NearbyRadius:     500.0, // 500 meters
		ScanInterval:     30 * time.Second,
		HeartbeatInterval: 60 * time.Second,
		EnableNearby:     true,
		EnableSocialFeed: true,
		EnableEvents:     true,
	}
}

// CloudService handles all cloud-based social features
type CloudService struct {
	mu sync.RWMutex

	// Configuration
	Config *CloudServiceConfig

	// Connection state
	Status    CloudServiceStatus
	LastError error
	UserID    string
	SessionID string

	// Local pet info
	LocalPet    *PetProfile
	LocalLocation *Location

	// Social data
	Friends       map[types.PetID]*PetProfile
	NearbyPets    []*NearbyPet
	PendingRequests []*FriendRequest
	SocialFeed    *SocialFeed
	ActiveEvents  []*SocialEvent
	Leaderboards  map[string]*Leaderboard

	// Interaction history
	InteractionHistory []*SocialInteraction
	LastInteractionTime time.Time

	// Callbacks
	OnNearbyPetFound    func(pet *NearbyPet)
	OnNearbyPetLost     func(petID types.PetID)
	OnFriendRequest     func(request *FriendRequest)
	OnSocialInteraction func(interaction *SocialInteraction)
	OnEventNotification func(event *SocialEvent)
	OnStatusChange      func(status CloudServiceStatus)

	// Backend provider (interface for actual network calls)
	Provider SocialCloudProvider

	// Internal
	stopChan      chan struct{}
	nearbyScanTicker *time.Ticker
	heartbeatTicker  *time.Ticker
}

// SocialCloudProvider interface for the actual cloud backend
type SocialCloudProvider interface {
	// Connection
	Connect(userID, sessionID string) error
	Disconnect() error
	IsConnected() bool

	// Profile management
	UpdateProfile(profile *PetProfile) error
	GetProfile(petID types.PetID) (*PetProfile, error)
	SearchProfiles(query string, limit int) ([]*PetProfile, error)

	// Location/Nearby
	UpdateLocation(location *Location) error
	GetNearbyPets(location *Location, radius float64) ([]*NearbyPet, error)

	// Friends
	GetFriends() (map[types.PetID]*PetProfile, error)
	SendFriendRequest(toPetID types.PetID, message string) (*FriendRequest, error)
	GetPendingFriendRequests() ([]*FriendRequest, error)
	RespondToFriendRequest(requestID string, accept bool) error
	RemoveFriend(petID types.PetID) error

	// Social interactions
	SendInteraction(interaction *SocialInteraction) error
	GetInteractionHistory(limit int) ([]*SocialInteraction, error)

	// Feed
	GetSocialFeed(limit int) (*SocialFeed, error)
	PostToFeed(content string, data map[string]interface{}) (*FeedItem, error)
	LikeFeedItem(itemID string) error

	// Events
	GetActiveEvents() ([]*SocialEvent, error)
	JoinEvent(eventID string) error
	LeaveEvent(eventID string) error

	// Leaderboards
	GetLeaderboard(leaderboardID string) (*Leaderboard, error)
	SubmitScore(leaderboardID string, score float64) error
}

// NewCloudService creates a new cloud service
func NewCloudService(config *CloudServiceConfig) *CloudService {
	if config == nil {
		config = DefaultCloudServiceConfig()
	}

	return &CloudService{
		Config:             config,
		Status:             CloudStatusDisconnected,
		Friends:            make(map[types.PetID]*PetProfile),
		NearbyPets:         make([]*NearbyPet, 0),
		PendingRequests:    make([]*FriendRequest, 0),
		SocialFeed:         &SocialFeed{Items: make([]*FeedItem, 0)},
		ActiveEvents:       make([]*SocialEvent, 0),
		Leaderboards:       make(map[string]*Leaderboard),
		InteractionHistory: make([]*SocialInteraction, 0),
	}
}

// SetProvider sets the cloud provider implementation
func (cs *CloudService) SetProvider(provider SocialCloudProvider) {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	cs.Provider = provider
}

// Connect establishes connection to the cloud service
func (cs *CloudService) Connect(userID string) error {
	cs.mu.Lock()
	if cs.Provider == nil {
		cs.mu.Unlock()
		return fmt.Errorf("no cloud provider configured")
	}

	cs.Status = CloudStatusConnecting
	cs.UserID = userID
	cs.SessionID = generateSessionID()
	cs.mu.Unlock()

	if cs.OnStatusChange != nil {
		cs.OnStatusChange(CloudStatusConnecting)
	}

	err := cs.Provider.Connect(userID, cs.SessionID)

	cs.mu.Lock()
	if err != nil {
		cs.Status = CloudStatusError
		cs.LastError = err
		cs.mu.Unlock()
		if cs.OnStatusChange != nil {
			cs.OnStatusChange(CloudStatusError)
		}
		return err
	}

	cs.Status = CloudStatusConnected
	cs.mu.Unlock()

	if cs.OnStatusChange != nil {
		cs.OnStatusChange(CloudStatusConnected)
	}

	// Start background services
	cs.startBackgroundServices()

	return nil
}

// Disconnect closes the cloud service connection
func (cs *CloudService) Disconnect() error {
	cs.stopBackgroundServices()

	cs.mu.Lock()
	defer cs.mu.Unlock()

	if cs.Provider != nil {
		cs.Provider.Disconnect()
	}

	cs.Status = CloudStatusDisconnected
	cs.SessionID = ""

	if cs.OnStatusChange != nil {
		cs.OnStatusChange(CloudStatusDisconnected)
	}

	return nil
}

// startBackgroundServices starts background scanning and heartbeat
func (cs *CloudService) startBackgroundServices() {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	cs.stopChan = make(chan struct{})

	// Start nearby pet scanning
	if cs.Config.EnableNearby {
		cs.nearbyScanTicker = time.NewTicker(cs.Config.ScanInterval)
		go cs.nearbyScanLoop()
	}

	// Start heartbeat
	cs.heartbeatTicker = time.NewTicker(cs.Config.HeartbeatInterval)
	go cs.heartbeatLoop()
}

// stopBackgroundServices stops all background services
func (cs *CloudService) stopBackgroundServices() {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	if cs.stopChan != nil {
		close(cs.stopChan)
		cs.stopChan = nil
	}

	if cs.nearbyScanTicker != nil {
		cs.nearbyScanTicker.Stop()
		cs.nearbyScanTicker = nil
	}

	if cs.heartbeatTicker != nil {
		cs.heartbeatTicker.Stop()
		cs.heartbeatTicker = nil
	}
}

// nearbyScanLoop periodically scans for nearby pets
func (cs *CloudService) nearbyScanLoop() {
	cs.mu.RLock()
	ticker := cs.nearbyScanTicker
	stopChan := cs.stopChan
	cs.mu.RUnlock()

	if ticker == nil || stopChan == nil {
		return
	}

	for {
		select {
		case _, ok := <-ticker.C:
			if !ok {
				return
			}
			cs.scanForNearbyPets()
		case <-stopChan:
			return
		}
	}
}

// heartbeatLoop sends periodic heartbeats to keep connection alive
func (cs *CloudService) heartbeatLoop() {
	cs.mu.RLock()
	ticker := cs.heartbeatTicker
	stopChan := cs.stopChan
	cs.mu.RUnlock()

	if ticker == nil || stopChan == nil {
		return
	}

	for {
		select {
		case _, ok := <-ticker.C:
			if !ok {
				return
			}
			cs.sendHeartbeat()
		case <-stopChan:
			return
		}
	}
}

// sendHeartbeat sends a heartbeat to the server
func (cs *CloudService) sendHeartbeat() {
	cs.mu.RLock()
	if cs.Provider == nil || !cs.Provider.IsConnected() {
		cs.mu.RUnlock()
		return
	}

	// Update profile to signal online status
	if cs.LocalPet != nil {
		profile := cs.LocalPet
		profile.IsOnline = true
		profile.LastSeen = time.Now()
		cs.mu.RUnlock()
		cs.Provider.UpdateProfile(profile)
		return
	}
	cs.mu.RUnlock()
}

// UpdateLocalPet updates the local pet profile
func (cs *CloudService) UpdateLocalPet(profile *PetProfile) error {
	cs.mu.Lock()
	cs.LocalPet = profile
	cs.mu.Unlock()

	if cs.Provider != nil && cs.Provider.IsConnected() {
		return cs.Provider.UpdateProfile(profile)
	}

	return nil
}

// UpdateLocation updates the local pet's location
func (cs *CloudService) UpdateLocation(lat, lon, accuracy float64) error {
	cs.mu.Lock()
	cs.LocalLocation = &Location{
		Latitude:  lat,
		Longitude: lon,
		Accuracy:  accuracy,
		Timestamp: time.Now(),
	}
	location := cs.LocalLocation
	cs.mu.Unlock()

	if cs.Provider != nil && cs.Provider.IsConnected() {
		return cs.Provider.UpdateLocation(location)
	}

	return nil
}

// scanForNearbyPets scans for nearby pets
func (cs *CloudService) scanForNearbyPets() {
	cs.mu.RLock()
	if cs.Provider == nil || !cs.Provider.IsConnected() || cs.LocalLocation == nil {
		cs.mu.RUnlock()
		return
	}
	location := cs.LocalLocation
	radius := cs.Config.NearbyRadius
	cs.mu.RUnlock()

	nearbyPets, err := cs.Provider.GetNearbyPets(location, radius)
	if err != nil {
		return
	}

	cs.mu.Lock()
	// Track which pets we had before
	oldPetIDs := make(map[types.PetID]bool)
	for _, pet := range cs.NearbyPets {
		oldPetIDs[pet.Profile.PetID] = true
	}

	// Track which pets we have now
	newPetIDs := make(map[types.PetID]bool)
	for _, pet := range nearbyPets {
		newPetIDs[pet.Profile.PetID] = true
	}

	cs.NearbyPets = nearbyPets
	cs.mu.Unlock()

	// Notify about new pets
	for _, pet := range nearbyPets {
		if !oldPetIDs[pet.Profile.PetID] {
			if cs.OnNearbyPetFound != nil {
				cs.OnNearbyPetFound(pet)
			}
		}
	}

	// Notify about lost pets
	for petID := range oldPetIDs {
		if !newPetIDs[petID] {
			if cs.OnNearbyPetLost != nil {
				cs.OnNearbyPetLost(petID)
			}
		}
	}
}

// GetNearbyPets returns the current list of nearby pets
func (cs *CloudService) GetNearbyPets() []*NearbyPet {
	cs.mu.RLock()
	defer cs.mu.RUnlock()

	result := make([]*NearbyPet, len(cs.NearbyPets))
	copy(result, cs.NearbyPets)
	return result
}

// GetNearbyPetsSorted returns nearby pets sorted by distance
func (cs *CloudService) GetNearbyPetsSorted() []*NearbyPet {
	pets := cs.GetNearbyPets()
	sort.Slice(pets, func(i, j int) bool {
		return pets[i].Distance < pets[j].Distance
	})
	return pets
}

// FindNearbyPet finds a specific nearby pet by ID
func (cs *CloudService) FindNearbyPet(petID types.PetID) *NearbyPet {
	cs.mu.RLock()
	defer cs.mu.RUnlock()

	for _, pet := range cs.NearbyPets {
		if pet.Profile.PetID == petID {
			return pet
		}
	}
	return nil
}

// SendFriendRequest sends a friend request to another pet
func (cs *CloudService) SendFriendRequest(toPetID types.PetID, message string) error {
	if cs.Provider == nil || !cs.Provider.IsConnected() {
		return fmt.Errorf("not connected to cloud service")
	}

	request, err := cs.Provider.SendFriendRequest(toPetID, message)
	if err != nil {
		return err
	}

	cs.mu.Lock()
	cs.PendingRequests = append(cs.PendingRequests, request)
	cs.mu.Unlock()

	return nil
}

// GetPendingFriendRequests returns pending friend requests
func (cs *CloudService) GetPendingFriendRequests() []*FriendRequest {
	cs.mu.RLock()
	defer cs.mu.RUnlock()

	result := make([]*FriendRequest, len(cs.PendingRequests))
	copy(result, cs.PendingRequests)
	return result
}

// RefreshFriendRequests fetches the latest friend requests from server
func (cs *CloudService) RefreshFriendRequests() error {
	if cs.Provider == nil || !cs.Provider.IsConnected() {
		return fmt.Errorf("not connected to cloud service")
	}

	requests, err := cs.Provider.GetPendingFriendRequests()
	if err != nil {
		return err
	}

	cs.mu.Lock()
	cs.PendingRequests = requests
	cs.mu.Unlock()

	// Notify about new requests
	for _, request := range requests {
		if request.Status == FriendRequestPending {
			if cs.OnFriendRequest != nil {
				cs.OnFriendRequest(request)
			}
		}
	}

	return nil
}

// AcceptFriendRequest accepts a friend request
func (cs *CloudService) AcceptFriendRequest(requestID string) error {
	if cs.Provider == nil || !cs.Provider.IsConnected() {
		return fmt.Errorf("not connected to cloud service")
	}

	err := cs.Provider.RespondToFriendRequest(requestID, true)
	if err != nil {
		return err
	}

	// Refresh friends list
	return cs.RefreshFriends()
}

// DeclineFriendRequest declines a friend request
func (cs *CloudService) DeclineFriendRequest(requestID string) error {
	if cs.Provider == nil || !cs.Provider.IsConnected() {
		return fmt.Errorf("not connected to cloud service")
	}

	return cs.Provider.RespondToFriendRequest(requestID, false)
}

// RefreshFriends refreshes the friends list from server
func (cs *CloudService) RefreshFriends() error {
	if cs.Provider == nil || !cs.Provider.IsConnected() {
		return fmt.Errorf("not connected to cloud service")
	}

	friends, err := cs.Provider.GetFriends()
	if err != nil {
		return err
	}

	cs.mu.Lock()
	cs.Friends = friends
	cs.mu.Unlock()

	return nil
}

// GetFriends returns the current friends list
func (cs *CloudService) GetFriends() map[types.PetID]*PetProfile {
	cs.mu.RLock()
	defer cs.mu.RUnlock()

	result := make(map[types.PetID]*PetProfile)
	for k, v := range cs.Friends {
		result[k] = v
	}
	return result
}

// GetOnlineFriends returns friends that are currently online
func (cs *CloudService) GetOnlineFriends() []*PetProfile {
	cs.mu.RLock()
	defer cs.mu.RUnlock()

	var online []*PetProfile
	for _, friend := range cs.Friends {
		if friend.IsOnline {
			online = append(online, friend)
		}
	}
	return online
}

// RemoveFriend removes a friend
func (cs *CloudService) RemoveFriend(petID types.PetID) error {
	if cs.Provider == nil || !cs.Provider.IsConnected() {
		return fmt.Errorf("not connected to cloud service")
	}

	err := cs.Provider.RemoveFriend(petID)
	if err != nil {
		return err
	}

	cs.mu.Lock()
	delete(cs.Friends, petID)
	cs.mu.Unlock()

	return nil
}

// IsFriend checks if a pet is a friend
func (cs *CloudService) IsFriend(petID types.PetID) bool {
	cs.mu.RLock()
	defer cs.mu.RUnlock()
	_, exists := cs.Friends[petID]
	return exists
}

// InitiateSocialInteraction starts a social interaction with another pet
func (cs *CloudService) InitiateSocialInteraction(targetID types.PetID, interactionType SocialInteractionType) (*SocialInteraction, error) {
	if cs.Provider == nil || !cs.Provider.IsConnected() {
		return nil, fmt.Errorf("not connected to cloud service")
	}

	cs.mu.RLock()
	localPet := cs.LocalPet
	cs.mu.RUnlock()

	if localPet == nil {
		return nil, fmt.Errorf("local pet not configured")
	}

	interaction := &SocialInteraction{
		ID:          generateInteractionID(),
		Type:        interactionType,
		InitiatorID: localPet.PetID,
		TargetID:    targetID,
		Timestamp:   time.Now(),
		Data:        make(map[string]interface{}),
	}

	err := cs.Provider.SendInteraction(interaction)
	if err != nil {
		return nil, err
	}

	cs.mu.Lock()
	cs.InteractionHistory = append(cs.InteractionHistory, interaction)
	cs.LastInteractionTime = time.Now()
	// Keep only last 100 interactions
	if len(cs.InteractionHistory) > 100 {
		cs.InteractionHistory = cs.InteractionHistory[1:]
	}
	cs.mu.Unlock()

	if cs.OnSocialInteraction != nil {
		cs.OnSocialInteraction(interaction)
	}

	return interaction, nil
}

// WaveAtPet sends a wave to another pet (simple greeting)
func (cs *CloudService) WaveAtPet(targetID types.PetID) error {
	_, err := cs.InitiateSocialInteraction(targetID, SocialInteractionWave)
	return err
}

// PlayWithPet initiates play with another pet
func (cs *CloudService) PlayWithPet(targetID types.PetID) (*SocialInteraction, error) {
	return cs.InitiateSocialInteraction(targetID, SocialInteractionPlay)
}

// VisitPet visits another pet's space
func (cs *CloudService) VisitPet(targetID types.PetID) (*SocialInteraction, error) {
	return cs.InitiateSocialInteraction(targetID, SocialInteractionVisit)
}

// GetInteractionHistory returns recent social interactions
func (cs *CloudService) GetInteractionHistory() []*SocialInteraction {
	cs.mu.RLock()
	defer cs.mu.RUnlock()

	result := make([]*SocialInteraction, len(cs.InteractionHistory))
	copy(result, cs.InteractionHistory)
	return result
}

// RefreshInteractionHistory fetches interaction history from server
func (cs *CloudService) RefreshInteractionHistory(limit int) error {
	if cs.Provider == nil || !cs.Provider.IsConnected() {
		return fmt.Errorf("not connected to cloud service")
	}

	history, err := cs.Provider.GetInteractionHistory(limit)
	if err != nil {
		return err
	}

	cs.mu.Lock()
	cs.InteractionHistory = history
	cs.mu.Unlock()

	return nil
}

// RefreshSocialFeed fetches the social feed from server
func (cs *CloudService) RefreshSocialFeed() error {
	if cs.Provider == nil || !cs.Provider.IsConnected() {
		return fmt.Errorf("not connected to cloud service")
	}

	feed, err := cs.Provider.GetSocialFeed(50)
	if err != nil {
		return err
	}

	cs.mu.Lock()
	cs.SocialFeed = feed
	cs.mu.Unlock()

	return nil
}

// GetSocialFeed returns the current social feed
func (cs *CloudService) GetSocialFeed() *SocialFeed {
	cs.mu.RLock()
	defer cs.mu.RUnlock()
	return cs.SocialFeed
}

// PostToFeed posts content to the social feed
func (cs *CloudService) PostToFeed(content string, data map[string]interface{}) (*FeedItem, error) {
	if cs.Provider == nil || !cs.Provider.IsConnected() {
		return nil, fmt.Errorf("not connected to cloud service")
	}

	return cs.Provider.PostToFeed(content, data)
}

// LikeFeedItem likes a feed item
func (cs *CloudService) LikeFeedItem(itemID string) error {
	if cs.Provider == nil || !cs.Provider.IsConnected() {
		return fmt.Errorf("not connected to cloud service")
	}

	return cs.Provider.LikeFeedItem(itemID)
}

// RefreshEvents fetches active events from server
func (cs *CloudService) RefreshEvents() error {
	if cs.Provider == nil || !cs.Provider.IsConnected() {
		return fmt.Errorf("not connected to cloud service")
	}

	events, err := cs.Provider.GetActiveEvents()
	if err != nil {
		return err
	}

	cs.mu.Lock()
	cs.ActiveEvents = events
	cs.mu.Unlock()

	return nil
}

// GetActiveEvents returns active social events
func (cs *CloudService) GetActiveEvents() []*SocialEvent {
	cs.mu.RLock()
	defer cs.mu.RUnlock()

	result := make([]*SocialEvent, len(cs.ActiveEvents))
	copy(result, cs.ActiveEvents)
	return result
}

// GetNearbyEvents returns events near the current location
func (cs *CloudService) GetNearbyEvents(maxDistance float64) []*SocialEvent {
	cs.mu.RLock()
	defer cs.mu.RUnlock()

	if cs.LocalLocation == nil {
		return nil
	}

	var nearby []*SocialEvent
	for _, event := range cs.ActiveEvents {
		if event.Location != nil {
			distance := calculateDistance(
				cs.LocalLocation.Latitude, cs.LocalLocation.Longitude,
				event.Location.Latitude, event.Location.Longitude,
			)
			if distance <= maxDistance {
				nearby = append(nearby, event)
			}
		}
	}
	return nearby
}

// JoinEvent joins a social event
func (cs *CloudService) JoinEvent(eventID string) error {
	if cs.Provider == nil || !cs.Provider.IsConnected() {
		return fmt.Errorf("not connected to cloud service")
	}

	return cs.Provider.JoinEvent(eventID)
}

// LeaveEvent leaves a social event
func (cs *CloudService) LeaveEvent(eventID string) error {
	if cs.Provider == nil || !cs.Provider.IsConnected() {
		return fmt.Errorf("not connected to cloud service")
	}

	return cs.Provider.LeaveEvent(eventID)
}

// GetLeaderboard fetches a leaderboard
func (cs *CloudService) GetLeaderboard(leaderboardID string) (*Leaderboard, error) {
	if cs.Provider == nil || !cs.Provider.IsConnected() {
		return nil, fmt.Errorf("not connected to cloud service")
	}

	leaderboard, err := cs.Provider.GetLeaderboard(leaderboardID)
	if err != nil {
		return nil, err
	}

	cs.mu.Lock()
	cs.Leaderboards[leaderboardID] = leaderboard
	cs.mu.Unlock()

	return leaderboard, nil
}

// SubmitScore submits a score to a leaderboard
func (cs *CloudService) SubmitScore(leaderboardID string, score float64) error {
	if cs.Provider == nil || !cs.Provider.IsConnected() {
		return fmt.Errorf("not connected to cloud service")
	}

	return cs.Provider.SubmitScore(leaderboardID, score)
}

// GetStatus returns the current cloud service status
func (cs *CloudService) GetStatus() CloudServiceStatus {
	cs.mu.RLock()
	defer cs.mu.RUnlock()
	return cs.Status
}

// IsConnected returns whether the cloud service is connected
func (cs *CloudService) IsConnected() bool {
	cs.mu.RLock()
	defer cs.mu.RUnlock()
	return cs.Status == CloudStatusConnected
}

// GetLastError returns the last error
func (cs *CloudService) GetLastError() error {
	cs.mu.RLock()
	defer cs.mu.RUnlock()
	return cs.LastError
}

// SearchPets searches for pets by name or other criteria
func (cs *CloudService) SearchPets(query string, limit int) ([]*PetProfile, error) {
	if cs.Provider == nil || !cs.Provider.IsConnected() {
		return nil, fmt.Errorf("not connected to cloud service")
	}

	return cs.Provider.SearchProfiles(query, limit)
}

// GetPetProfile fetches a pet's profile
func (cs *CloudService) GetPetProfile(petID types.PetID) (*PetProfile, error) {
	if cs.Provider == nil || !cs.Provider.IsConnected() {
		return nil, fmt.Errorf("not connected to cloud service")
	}

	return cs.Provider.GetProfile(petID)
}

// Helper functions

// calculateDistance calculates distance between two coordinates in meters (Haversine formula)
func calculateDistance(lat1, lon1, lat2, lon2 float64) float64 {
	const earthRadius = 6371000 // meters

	dLat := (lat2 - lat1) * math.Pi / 180
	dLon := (lon2 - lon1) * math.Pi / 180

	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(lat1*math.Pi/180)*math.Cos(lat2*math.Pi/180)*
			math.Sin(dLon/2)*math.Sin(dLon/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	return earthRadius * c
}

// calculateBearing calculates bearing from point 1 to point 2 in degrees
func calculateBearing(lat1, lon1, lat2, lon2 float64) float64 {
	dLon := (lon2 - lon1) * math.Pi / 180
	lat1Rad := lat1 * math.Pi / 180
	lat2Rad := lat2 * math.Pi / 180

	y := math.Sin(dLon) * math.Cos(lat2Rad)
	x := math.Cos(lat1Rad)*math.Sin(lat2Rad) -
		math.Sin(lat1Rad)*math.Cos(lat2Rad)*math.Cos(dLon)

	bearing := math.Atan2(y, x) * 180 / math.Pi
	return math.Mod(bearing+360, 360)
}

// generateSessionID generates a unique session ID
func generateSessionID() string {
	bytes := make([]byte, 16)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

// generateInteractionID generates a unique interaction ID
func generateInteractionID() string {
	bytes := make([]byte, 8)
	rand.Read(bytes)
	return fmt.Sprintf("int_%s_%d", hex.EncodeToString(bytes), time.Now().UnixNano())
}

// MockSocialCloudProvider is a mock implementation for testing
type MockSocialCloudProvider struct {
	mu sync.RWMutex

	Connected bool
	Profiles  map[types.PetID]*PetProfile
	Friends   map[types.PetID]*PetProfile
	Requests  []*FriendRequest
	Feed      *SocialFeed
	Events    []*SocialEvent
	Leaderboards map[string]*Leaderboard
	NearbyPets []*NearbyPet
	Interactions []*SocialInteraction
}

// NewMockSocialCloudProvider creates a new mock provider
func NewMockSocialCloudProvider() *MockSocialCloudProvider {
	return &MockSocialCloudProvider{
		Profiles:     make(map[types.PetID]*PetProfile),
		Friends:      make(map[types.PetID]*PetProfile),
		Requests:     make([]*FriendRequest, 0),
		Feed:         &SocialFeed{Items: make([]*FeedItem, 0)},
		Events:       make([]*SocialEvent, 0),
		Leaderboards: make(map[string]*Leaderboard),
		NearbyPets:   make([]*NearbyPet, 0),
		Interactions: make([]*SocialInteraction, 0),
	}
}

func (m *MockSocialCloudProvider) Connect(userID, sessionID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Connected = true
	return nil
}

func (m *MockSocialCloudProvider) Disconnect() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Connected = false
	return nil
}

func (m *MockSocialCloudProvider) IsConnected() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.Connected
}

func (m *MockSocialCloudProvider) UpdateProfile(profile *PetProfile) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Profiles[profile.PetID] = profile
	return nil
}

func (m *MockSocialCloudProvider) GetProfile(petID types.PetID) (*PetProfile, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if profile, exists := m.Profiles[petID]; exists {
		return profile, nil
	}
	return nil, fmt.Errorf("profile not found")
}

func (m *MockSocialCloudProvider) SearchProfiles(query string, limit int) ([]*PetProfile, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var results []*PetProfile
	for _, profile := range m.Profiles {
		if len(results) >= limit {
			break
		}
		// Simple name matching
		if query == "" || containsIgnoreCase(profile.Name, query) {
			results = append(results, profile)
		}
	}
	return results, nil
}

func (m *MockSocialCloudProvider) UpdateLocation(location *Location) error {
	return nil
}

func (m *MockSocialCloudProvider) GetNearbyPets(location *Location, radius float64) ([]*NearbyPet, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]*NearbyPet, len(m.NearbyPets))
	copy(result, m.NearbyPets)
	return result, nil
}

func (m *MockSocialCloudProvider) GetFriends() (map[types.PetID]*PetProfile, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make(map[types.PetID]*PetProfile)
	for k, v := range m.Friends {
		result[k] = v
	}
	return result, nil
}

func (m *MockSocialCloudProvider) SendFriendRequest(toPetID types.PetID, message string) (*FriendRequest, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	request := &FriendRequest{
		ID:        generateInteractionID(),
		ToPetID:   toPetID,
		Message:   message,
		SentAt:    time.Now(),
		Status:    FriendRequestPending,
	}
	m.Requests = append(m.Requests, request)
	return request, nil
}

func (m *MockSocialCloudProvider) GetPendingFriendRequests() ([]*FriendRequest, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]*FriendRequest, len(m.Requests))
	copy(result, m.Requests)
	return result, nil
}

func (m *MockSocialCloudProvider) RespondToFriendRequest(requestID string, accept bool) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for i, req := range m.Requests {
		if req.ID == requestID {
			if accept {
				m.Requests[i].Status = FriendRequestAccepted
				// Add to friends
				m.Friends[req.FromPetID] = &PetProfile{PetID: req.FromPetID}
			} else {
				m.Requests[i].Status = FriendRequestDeclined
			}
			m.Requests[i].RespondedAt = time.Now()
			return nil
		}
	}
	return fmt.Errorf("request not found")
}

func (m *MockSocialCloudProvider) RemoveFriend(petID types.PetID) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.Friends, petID)
	return nil
}

func (m *MockSocialCloudProvider) SendInteraction(interaction *SocialInteraction) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Interactions = append(m.Interactions, interaction)
	return nil
}

func (m *MockSocialCloudProvider) GetInteractionHistory(limit int) ([]*SocialInteraction, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	count := limit
	if count > len(m.Interactions) {
		count = len(m.Interactions)
	}

	result := make([]*SocialInteraction, count)
	copy(result, m.Interactions[len(m.Interactions)-count:])
	return result, nil
}

func (m *MockSocialCloudProvider) GetSocialFeed(limit int) (*SocialFeed, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.Feed, nil
}

func (m *MockSocialCloudProvider) PostToFeed(content string, data map[string]interface{}) (*FeedItem, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	item := &FeedItem{
		ID:        generateInteractionID(),
		Type:      "post",
		Content:   content,
		Timestamp: time.Now(),
		Data:      data,
	}
	m.Feed.Items = append(m.Feed.Items, item)
	return item, nil
}

func (m *MockSocialCloudProvider) LikeFeedItem(itemID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, item := range m.Feed.Items {
		if item.ID == itemID {
			item.Likes++
			item.HasLiked = true
			return nil
		}
	}
	return fmt.Errorf("item not found")
}

func (m *MockSocialCloudProvider) GetActiveEvents() ([]*SocialEvent, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]*SocialEvent, len(m.Events))
	copy(result, m.Events)
	return result, nil
}

func (m *MockSocialCloudProvider) JoinEvent(eventID string) error {
	return nil
}

func (m *MockSocialCloudProvider) LeaveEvent(eventID string) error {
	return nil
}

func (m *MockSocialCloudProvider) GetLeaderboard(leaderboardID string) (*Leaderboard, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if lb, exists := m.Leaderboards[leaderboardID]; exists {
		return lb, nil
	}
	return nil, fmt.Errorf("leaderboard not found")
}

func (m *MockSocialCloudProvider) SubmitScore(leaderboardID string, score float64) error {
	return nil
}

// AddMockNearbyPet adds a mock nearby pet for testing
func (m *MockSocialCloudProvider) AddMockNearbyPet(profile *PetProfile, distance, direction float64) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.NearbyPets = append(m.NearbyPets, &NearbyPet{
		Profile:   profile,
		Distance:  distance,
		Direction: direction,
		LastSeen:  time.Now(),
		Signal:    1.0 - (distance / 1000), // Signal decreases with distance
	})
}

// AddMockEvent adds a mock event for testing
func (m *MockSocialCloudProvider) AddMockEvent(event *SocialEvent) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Events = append(m.Events, event)
}

// AddMockLeaderboard adds a mock leaderboard for testing
func (m *MockSocialCloudProvider) AddMockLeaderboard(leaderboard *Leaderboard) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Leaderboards[leaderboard.ID] = leaderboard
}

// Helper function for case-insensitive string contains
func containsIgnoreCase(s, substr string) bool {
	if len(substr) == 0 {
		return true
	}
	if len(s) < len(substr) {
		return false
	}
	for i := 0; i <= len(s)-len(substr); i++ {
		match := true
		for j := 0; j < len(substr); j++ {
			sc := s[i+j]
			tc := substr[j]
			if sc >= 'A' && sc <= 'Z' {
				sc = sc + 32
			}
			if tc >= 'A' && tc <= 'Z' {
				tc = tc + 32
			}
			if sc != tc {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}
