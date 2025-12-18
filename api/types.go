package api

import (
	"time"

	"github.com/Michael-W-Ellison/gochi/pkg/types"
)

// ============================================================================
// Common Types
// ============================================================================

// APIResponse is the standard response wrapper for all API endpoints
type APIResponse struct {
	Success   bool        `json:"success"`
	Data      interface{} `json:"data,omitempty"`
	Error     *APIError   `json:"error,omitempty"`
	Timestamp time.Time   `json:"timestamp"`
}

// APIError represents an error response
type APIError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}

// PaginatedResponse wraps paginated data
type PaginatedResponse struct {
	Items      interface{} `json:"items"`
	TotalCount int         `json:"total_count"`
	Page       int         `json:"page"`
	PageSize   int         `json:"page_size"`
	HasMore    bool        `json:"has_more"`
}

// ============================================================================
// Authentication Types
// ============================================================================

// LoginRequest represents a login request
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// LoginResponse represents a successful login response
type LoginResponse struct {
	UserID       string    `json:"user_id"`
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	ExpiresAt    time.Time `json:"expires_at"`
}

// RegisterRequest represents a user registration request
type RegisterRequest struct {
	Email       string `json:"email"`
	Password    string `json:"password"`
	DisplayName string `json:"display_name"`
}

// TokenRefreshRequest represents a token refresh request
type TokenRefreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

// ============================================================================
// Pet Types
// ============================================================================

// CreatePetRequest represents a request to create a new pet
type CreatePetRequest struct {
	Name           string            `json:"name"`
	Species        string            `json:"species,omitempty"`
	Appearance     *AppearanceConfig `json:"appearance,omitempty"`
	RandomizeTraits bool             `json:"randomize_traits"`
}

// AppearanceConfig represents pet appearance customization
type AppearanceConfig struct {
	PrimaryColor   string `json:"primary_color"`
	SecondaryColor string `json:"secondary_color"`
	PatternType    string `json:"pattern_type,omitempty"`
	EyeColor       string `json:"eye_color"`
	Size           string `json:"size"` // small, medium, large
}

// PetResponse represents a pet in API responses
type PetResponse struct {
	ID              types.PetID         `json:"id"`
	Name            string              `json:"name"`
	Owner           types.UserID        `json:"owner"`
	Species         string              `json:"species"`
	Age             float64             `json:"age_days"`
	IsAlive         bool                `json:"is_alive"`
	CurrentBehavior string              `json:"current_behavior"`
	Location        string              `json:"location"`
	Vitals          *VitalsResponse     `json:"vitals"`
	Emotions        *EmotionsResponse   `json:"emotions"`
	Personality     *PersonalityResponse `json:"personality,omitempty"`
	Statistics      *PetStatistics      `json:"statistics"`
	CreatedAt       time.Time           `json:"created_at"`
	LastUpdatedAt   time.Time           `json:"last_updated_at"`
}

// VitalsResponse represents pet vital statistics
type VitalsResponse struct {
	Health      float64 `json:"health"`
	Energy      float64 `json:"energy"`
	Happiness   float64 `json:"happiness"`
	Nutrition   float64 `json:"nutrition"`
	Hydration   float64 `json:"hydration"`
	Cleanliness float64 `json:"cleanliness"`
	Stress      float64 `json:"stress"`
	Fatigue     float64 `json:"fatigue"`
	Wellbeing   float64 `json:"overall_wellbeing"`
}

// EmotionsResponse represents pet emotional state
type EmotionsResponse struct {
	Joy             float64 `json:"joy"`
	Sadness         float64 `json:"sadness"`
	Anger           float64 `json:"anger"`
	Fear            float64 `json:"fear"`
	Excitement      float64 `json:"excitement"`
	Contentment     float64 `json:"contentment"`
	Affection       float64 `json:"affection"`
	Loneliness      float64 `json:"loneliness"`
	DominantEmotion string  `json:"dominant_emotion"`
	MoodScore       float64 `json:"mood_score"`
	MoodDescription string  `json:"mood_description"`
	MoodTrend       string  `json:"mood_trend"`
}

// PersonalityResponse represents pet personality traits
type PersonalityResponse struct {
	Openness          float64 `json:"openness"`
	Conscientiousness float64 `json:"conscientiousness"`
	Extraversion      float64 `json:"extraversion"`
	Agreeableness     float64 `json:"agreeableness"`
	Neuroticism       float64 `json:"neuroticism"`
	Playfulness       float64 `json:"playfulness"`
	Independence      float64 `json:"independence"`
	Loyalty           float64 `json:"loyalty"`
	Intelligence      float64 `json:"intelligence"`
	EnergyLevel       float64 `json:"energy_level"`
	Adaptability      float64 `json:"adaptability"`
	Description       string  `json:"description"`
}

// PetStatistics represents pet statistics
type PetStatistics struct {
	TotalInteractions   int     `json:"total_interactions"`
	TotalPlayTimeHours  float64 `json:"total_play_time_hours"`
	DaysAlive           float64 `json:"days_alive"`
	FriendCount         int     `json:"friend_count"`
	MemoryCount         int     `json:"memory_count"`
	AchievementCount    int     `json:"achievement_count"`
}

// UpdatePetRequest represents a request to update pet properties
type UpdatePetRequest struct {
	Name     string `json:"name,omitempty"`
	Location string `json:"location,omitempty"`
}

// PetStatusResponse is a lightweight status response for polling
type PetStatusResponse struct {
	ID              types.PetID `json:"id"`
	Name            string      `json:"name"`
	IsAlive         bool        `json:"is_alive"`
	CurrentBehavior string      `json:"current_behavior"`
	Health          float64     `json:"health"`
	Energy          float64     `json:"energy"`
	Happiness       float64     `json:"happiness"`
	Wellbeing       float64     `json:"wellbeing"`
	MoodDescription string      `json:"mood_description"`
	CriticalNeeds   []string    `json:"critical_needs,omitempty"`
}

// ============================================================================
// Interaction Types
// ============================================================================

// InteractionRequest represents a user interaction with a pet
type InteractionRequest struct {
	Type      string  `json:"type"` // feeding, petting, playing, training, grooming, medical, rewards, discipline
	Intensity float64 `json:"intensity"` // 0.0 to 1.0
	Duration  float64 `json:"duration,omitempty"` // seconds
	ItemID    string  `json:"item_id,omitempty"` // for feeding specific food, using specific toy
}

// InteractionResponse represents the result of an interaction
type InteractionResponse struct {
	Success        bool              `json:"success"`
	PetReaction    string            `json:"pet_reaction"`
	VitalChanges   map[string]float64 `json:"vital_changes"`
	EmotionChanges map[string]float64 `json:"emotion_changes"`
	MemoryCreated  bool              `json:"memory_created"`
	Experience     float64           `json:"experience_gained,omitempty"`
	Message        string            `json:"message"`
}

// FeedingRequest represents a feeding interaction
type FeedingRequest struct {
	FoodType string  `json:"food_type"` // basic, premium, treat, medicine
	Quantity float64 `json:"quantity"`  // 0.0 to 1.0
}

// PlayRequest represents a play interaction
type PlayRequest struct {
	ToyType    string  `json:"toy_type,omitempty"`
	Duration   float64 `json:"duration"`   // seconds
	Enthusiasm float64 `json:"enthusiasm"` // 0.0 to 1.0
}

// TrainingRequest represents a training interaction
type TrainingRequest struct {
	SkillType string `json:"skill_type"`
	Method    string `json:"method"` // positive_reinforcement, repetition, challenge
}

// GroomingRequest represents a grooming interaction
type GroomingRequest struct {
	GroomType string  `json:"groom_type"` // brush, bath, nail_trim, dental
	Duration  float64 `json:"duration"`
}

// ============================================================================
// Social Types
// ============================================================================

// NearbyPetResponse represents a nearby pet
type NearbyPetResponse struct {
	PetID        types.PetID `json:"pet_id"`
	Name         string      `json:"name"`
	OwnerID      string      `json:"owner_id"`
	OwnerName    string      `json:"owner_name"`
	Distance     float64     `json:"distance_meters"`
	Direction    string      `json:"direction"`
	IsOnline     bool        `json:"is_online"`
	LastSeen     time.Time   `json:"last_seen"`
	Compatibility float64    `json:"compatibility,omitempty"`
}

// FriendRequest represents a friend request
type FriendRequest struct {
	TargetPetID types.PetID `json:"target_pet_id"`
	Message     string      `json:"message,omitempty"`
}

// FriendRequestResponse represents a friend request in responses
type FriendRequestResponse struct {
	ID           string      `json:"id"`
	FromPetID    types.PetID `json:"from_pet_id"`
	FromPetName  string      `json:"from_pet_name"`
	ToPetID      types.PetID `json:"to_pet_id"`
	ToPetName    string      `json:"to_pet_name"`
	Message      string      `json:"message,omitempty"`
	Status       string      `json:"status"` // pending, accepted, declined, expired
	CreatedAt    time.Time   `json:"created_at"`
	ExpiresAt    time.Time   `json:"expires_at"`
}

// FriendResponse represents a friend in responses
type FriendResponse struct {
	PetID         types.PetID `json:"pet_id"`
	Name          string      `json:"name"`
	OwnerID       string      `json:"owner_id"`
	OwnerName     string      `json:"owner_name"`
	BondStrength  float64     `json:"bond_strength"`
	FriendsSince  time.Time   `json:"friends_since"`
	LastInteraction time.Time `json:"last_interaction"`
	IsOnline      bool        `json:"is_online"`
}

// SocialInteractionRequest represents a social interaction between pets
type SocialInteractionRequest struct {
	TargetPetID types.PetID `json:"target_pet_id"`
	Type        string      `json:"type"` // wave, greet, play, gift, trade, battle, visit
	Data        interface{} `json:"data,omitempty"`
}

// SocialInteractionResponse represents the result of a social interaction
type SocialInteractionResponse struct {
	Success       bool    `json:"success"`
	Type          string  `json:"type"`
	Result        string  `json:"result"`
	BondChange    float64 `json:"bond_change"`
	ExperienceGain float64 `json:"experience_gain,omitempty"`
	Message       string  `json:"message"`
}

// ============================================================================
// Social Feed Types
// ============================================================================

// FeedPostRequest represents a request to post to the social feed
type FeedPostRequest struct {
	Content    string        `json:"content"`
	ImageURL   string        `json:"image_url,omitempty"`
	TaggedPets []types.PetID `json:"tagged_pets,omitempty"`
}

// FeedItemResponse represents a social feed item
type FeedItemResponse struct {
	ID          string        `json:"id"`
	PetID       types.PetID   `json:"pet_id"`
	PetName     string        `json:"pet_name"`
	OwnerID     string        `json:"owner_id"`
	OwnerName   string        `json:"owner_name"`
	Content     string        `json:"content"`
	ImageURL    string        `json:"image_url,omitempty"`
	TaggedPets  []types.PetID `json:"tagged_pets,omitempty"`
	LikeCount   int           `json:"like_count"`
	HasLiked    bool          `json:"has_liked"`
	CreatedAt   time.Time     `json:"created_at"`
}

// ============================================================================
// Event Types
// ============================================================================

// EventResponse represents a social event
type EventResponse struct {
	ID            string      `json:"id"`
	Name          string      `json:"name"`
	Description   string      `json:"description"`
	Type          string      `json:"type"` // meetup, competition, party, training
	Location      *LocationResponse `json:"location,omitempty"`
	StartTime     time.Time   `json:"start_time"`
	EndTime       time.Time   `json:"end_time"`
	MaxParticipants int       `json:"max_participants"`
	CurrentCount  int         `json:"current_participants"`
	HostPetID     types.PetID `json:"host_pet_id"`
	HostName      string      `json:"host_name"`
	Rewards       []string    `json:"rewards,omitempty"`
	IsJoined      bool        `json:"is_joined"`
}

// LocationResponse represents a geographic location
type LocationResponse struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	Name      string  `json:"name,omitempty"`
	Address   string  `json:"address,omitempty"`
}

// LocationUpdateRequest represents a location update
type LocationUpdateRequest struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

// ============================================================================
// Leaderboard Types
// ============================================================================

// LeaderboardResponse represents a leaderboard
type LeaderboardResponse struct {
	ID          string             `json:"id"`
	Name        string             `json:"name"`
	Description string             `json:"description"`
	Period      string             `json:"period"` // daily, weekly, monthly, all_time
	Entries     []LeaderboardEntry `json:"entries"`
	UserRank    *LeaderboardEntry  `json:"user_rank,omitempty"`
	UpdatedAt   time.Time          `json:"updated_at"`
}

// LeaderboardEntry represents a single leaderboard entry
type LeaderboardEntry struct {
	Rank      int         `json:"rank"`
	PetID     types.PetID `json:"pet_id"`
	PetName   string      `json:"pet_name"`
	OwnerName string      `json:"owner_name"`
	Score     int         `json:"score"`
	IsCurrentUser bool    `json:"is_current_user"`
}

// ============================================================================
// Memory Types
// ============================================================================

// MemoryResponse represents a pet memory
type MemoryResponse struct {
	ID          string    `json:"id"`
	Type        string    `json:"type"`
	Description string    `json:"description"`
	Emotion     string    `json:"emotion"`
	Strength    float64   `json:"strength"`
	Importance  string    `json:"importance"`
	IsLongTerm  bool      `json:"is_long_term"`
	CreatedAt   time.Time `json:"created_at"`
	Tags        []string  `json:"tags,omitempty"`
}

// MemorySummaryResponse represents aggregated memory statistics
type MemorySummaryResponse struct {
	TotalMemories   int                `json:"total_memories"`
	ShortTermCount  int                `json:"short_term_count"`
	LongTermCount   int                `json:"long_term_count"`
	AverageStrength float64            `json:"average_strength"`
	DominantEmotion string             `json:"dominant_emotion"`
	MemoriesByType  map[string]int     `json:"memories_by_type"`
	MostRecalled    *MemoryResponse    `json:"most_recalled,omitempty"`
}

// ============================================================================
// Genetics/Breeding Types
// ============================================================================

// BreedingCompatibilityRequest represents a breeding compatibility check
type BreedingCompatibilityRequest struct {
	Pet1ID types.PetID `json:"pet1_id"`
	Pet2ID types.PetID `json:"pet2_id"`
}

// BreedingCompatibilityResponse represents breeding compatibility result
type BreedingCompatibilityResponse struct {
	IsCompatible      bool     `json:"is_compatible"`
	CompatibilityScore float64 `json:"compatibility_score"`
	PotentialTraits   []string `json:"potential_traits"`
	Warnings          []string `json:"warnings,omitempty"`
	CooldownRemaining float64  `json:"cooldown_remaining_hours,omitempty"`
}

// BreedingRequest represents a breeding request
type BreedingRequest struct {
	Partner1ID types.PetID `json:"partner1_id"`
	Partner2ID types.PetID `json:"partner2_id"`
}

// BreedingResponse represents the result of breeding
type BreedingResponse struct {
	Success       bool        `json:"success"`
	OffspringID   types.PetID `json:"offspring_id,omitempty"`
	OffspringName string      `json:"offspring_name,omitempty"`
	InheritedTraits []string  `json:"inherited_traits,omitempty"`
	Mutations     []string    `json:"mutations,omitempty"`
	Message       string      `json:"message"`
}

// ============================================================================
// Cloud Sync Types
// ============================================================================

// SyncRequest represents a sync request
type SyncRequest struct {
	LastSyncTime time.Time   `json:"last_sync_time"`
	LocalVersion int         `json:"local_version"`
	PetData      interface{} `json:"pet_data,omitempty"`
}

// SyncResponse represents a sync response
type SyncResponse struct {
	Status        string      `json:"status"` // up_to_date, updated, conflict
	ServerVersion int         `json:"server_version"`
	PetData       interface{} `json:"pet_data,omitempty"`
	ConflictData  interface{} `json:"conflict_data,omitempty"`
	LastSyncTime  time.Time   `json:"last_sync_time"`
}

// ConflictResolutionRequest represents a conflict resolution request
type ConflictResolutionRequest struct {
	Resolution string      `json:"resolution"` // keep_local, keep_server, merge
	MergedData interface{} `json:"merged_data,omitempty"`
}

// ============================================================================
// Achievement Types
// ============================================================================

// AchievementResponse represents an achievement
type AchievementResponse struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Category    string    `json:"category"`
	Points      int       `json:"points"`
	IsUnlocked  bool      `json:"is_unlocked"`
	UnlockedAt  time.Time `json:"unlocked_at,omitempty"`
	Progress    float64   `json:"progress"` // 0.0 to 1.0
	IconURL     string    `json:"icon_url,omitempty"`
}

// ============================================================================
// Environment Types
// ============================================================================

// EnvironmentResponse represents current environment state
type EnvironmentResponse struct {
	Weather     *WeatherResponse `json:"weather"`
	Season      string           `json:"season"`
	TimeOfDay   string           `json:"time_of_day"`
	Temperature float64          `json:"temperature_celsius"`
	Humidity    float64          `json:"humidity"`
}

// WeatherResponse represents weather conditions
type WeatherResponse struct {
	Condition   string  `json:"condition"` // clear, cloudy, rainy, snowy, stormy
	Temperature float64 `json:"temperature"`
	Humidity    float64 `json:"humidity"`
	WindSpeed   float64 `json:"wind_speed"`
	Description string  `json:"description"`
}

// ============================================================================
// WebSocket Types
// ============================================================================

// WSMessage represents a WebSocket message
type WSMessage struct {
	Type    string      `json:"type"`
	Payload interface{} `json:"payload"`
}

// WSEventType defines WebSocket event types
type WSEventType string

const (
	WSEventPetUpdate      WSEventType = "pet_update"
	WSEventVitalsUpdate   WSEventType = "vitals_update"
	WSEventEmotionUpdate  WSEventType = "emotion_update"
	WSEventNearbyPet      WSEventType = "nearby_pet"
	WSEventFriendRequest  WSEventType = "friend_request"
	WSEventSocialInteraction WSEventType = "social_interaction"
	WSEventFeedItem       WSEventType = "feed_item"
	WSEventEventUpdate    WSEventType = "event_update"
	WSEventAchievement    WSEventType = "achievement"
	WSEventNotification   WSEventType = "notification"
)

// NotificationResponse represents a notification
type NotificationResponse struct {
	ID        string    `json:"id"`
	Type      string    `json:"type"`
	Title     string    `json:"title"`
	Message   string    `json:"message"`
	Data      interface{} `json:"data,omitempty"`
	IsRead    bool      `json:"is_read"`
	CreatedAt time.Time `json:"created_at"`
}
