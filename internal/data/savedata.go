package data

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"time"
)

// SaveData represents a complete save file structure
type SaveData struct {
	// Metadata
	Version      string    `json:"version"`
	SaveID       string    `json:"save_id"`
	CreatedAt    time.Time `json:"created_at"`
	LastSaveTime time.Time `json:"last_save_time"`
	SaveCount    int       `json:"save_count"`

	// User information
	UserID       string           `json:"user_id"`
	Username     string           `json:"username"`
	Preferences  UserPreferences  `json:"preferences"`

	// Pet data (stored as raw JSON to avoid circular dependencies)
	PetData      json.RawMessage  `json:"pet_data"`

	// Game state
	CareHistory  []HistoryEntry   `json:"care_history"`
	Achievements []Achievement    `json:"achievements"`
	Unlocks      UnlockableContent `json:"unlocks"`
	Statistics   GameStatistics   `json:"statistics"`

	// Sync information
	SyncStatus   SyncStatus       `json:"sync_status"`
	Checksum     string           `json:"checksum"`
}

// UserPreferences stores user settings
type UserPreferences struct {
	NotificationsEnabled bool    `json:"notifications_enabled"`
	SoundEnabled         bool    `json:"sound_enabled"`
	MusicVolume          float64 `json:"music_volume"`
	SFXVolume            float64 `json:"sfx_volume"`
	TimeScale            string  `json:"time_scale"`
	Language             string  `json:"language"`
	Theme                string  `json:"theme"`
	AutoSaveInterval     int     `json:"auto_save_interval"` // seconds
}

// DefaultUserPreferences returns default preference values
func DefaultUserPreferences() UserPreferences {
	return UserPreferences{
		NotificationsEnabled: true,
		SoundEnabled:         true,
		MusicVolume:          0.7,
		SFXVolume:            0.8,
		TimeScale:            "ACCELERATED_4X",
		Language:             "en",
		Theme:                "default",
		AutoSaveInterval:     300,
	}
}

// HistoryEntry records a care action
type HistoryEntry struct {
	Timestamp   time.Time `json:"timestamp"`
	ActionType  string    `json:"action_type"`
	Description string    `json:"description"`
	PetAge      float64   `json:"pet_age"`
	Outcome     string    `json:"outcome"`
}

// Achievement represents an unlocked achievement
type Achievement struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	Description  string    `json:"description"`
	UnlockedAt   time.Time `json:"unlocked_at"`
	Rarity       string    `json:"rarity"` // common, rare, epic, legendary
	Points       int       `json:"points"`
}

// UnlockableContent tracks unlocked features
type UnlockableContent struct {
	Locations     []string `json:"locations"`
	Accessories   []string `json:"accessories"`
	Foods         []string `json:"foods"`
	Toys          []string `json:"toys"`
	Backgrounds   []string `json:"backgrounds"`
	PetVariants   []string `json:"pet_variants"`
}

// GameStatistics tracks overall game stats
type GameStatistics struct {
	TotalPlayTime      float64   `json:"total_play_time"`      // hours
	PetsCreated        int       `json:"pets_created"`
	PetsLost           int       `json:"pets_lost"`
	TotalInteractions  int       `json:"total_interactions"`
	LongestLifespan    float64   `json:"longest_lifespan"`     // days
	HighestHappiness   float64   `json:"highest_happiness"`
	FoodFed            int       `json:"food_fed"`
	GamesPlayed        int       `json:"games_played"`
	AchievementsEarned int       `json:"achievements_earned"`
	FirstPlayDate      time.Time `json:"first_play_date"`
	CurrentStreak      int       `json:"current_streak"`       // days
	LongestStreak      int       `json:"longest_streak"`       // days
}

// SyncStatus tracks cloud synchronization state
type SyncStatus struct {
	LastSyncTime     time.Time `json:"last_sync_time"`
	SyncEnabled      bool      `json:"sync_enabled"`
	CloudVersion     int       `json:"cloud_version"`
	LocalVersion     int       `json:"local_version"`
	PendingChanges   bool      `json:"pending_changes"`
	ConflictDetected bool      `json:"conflict_detected"`
}

// NewSaveData creates a new save data structure
func NewSaveData(userID, username string) *SaveData {
	now := time.Now()
	return &SaveData{
		Version:      "1.0.0",
		SaveID:       generateSaveID(),
		CreatedAt:    now,
		LastSaveTime: now,
		SaveCount:    0,
		UserID:       userID,
		Username:     username,
		Preferences:  DefaultUserPreferences(),
		CareHistory:  make([]HistoryEntry, 0),
		Achievements: make([]Achievement, 0),
		Unlocks: UnlockableContent{
			Locations:   []string{"home", "park"},
			Accessories: make([]string, 0),
			Foods:       []string{"basic_food"},
			Toys:        []string{"ball"},
			Backgrounds: []string{"default"},
			PetVariants: make([]string, 0),
		},
		Statistics: GameStatistics{
			FirstPlayDate: now,
		},
		SyncStatus: SyncStatus{
			SyncEnabled:  false,
			LocalVersion: 1,
		},
	}
}

// SetPetData stores pet data as JSON
func (sd *SaveData) SetPetData(petData interface{}) error {
	data, err := json.Marshal(petData)
	if err != nil {
		return fmt.Errorf("failed to marshal pet data: %w", err)
	}
	sd.PetData = data
	return nil
}

// GetPetData unmarshals pet data into the provided interface
func (sd *SaveData) GetPetData(target interface{}) error {
	if sd.PetData == nil {
		return fmt.Errorf("no pet data available")
	}
	return json.Unmarshal(sd.PetData, target)
}

// AddHistoryEntry adds a care history entry
func (sd *SaveData) AddHistoryEntry(actionType, description, outcome string, petAge float64) {
	entry := HistoryEntry{
		Timestamp:   time.Now(),
		ActionType:  actionType,
		Description: description,
		PetAge:      petAge,
		Outcome:     outcome,
	}
	sd.CareHistory = append(sd.CareHistory, entry)

	// Keep history manageable (last 1000 entries)
	if len(sd.CareHistory) > 1000 {
		sd.CareHistory = sd.CareHistory[len(sd.CareHistory)-1000:]
	}
}

// AddAchievement adds an achievement if not already unlocked
func (sd *SaveData) AddAchievement(id, name, description, rarity string, points int) bool {
	// Check if already unlocked
	for _, a := range sd.Achievements {
		if a.ID == id {
			return false
		}
	}

	achievement := Achievement{
		ID:          id,
		Name:        name,
		Description: description,
		UnlockedAt:  time.Now(),
		Rarity:      rarity,
		Points:      points,
	}
	sd.Achievements = append(sd.Achievements, achievement)
	sd.Statistics.AchievementsEarned++
	return true
}

// HasAchievement checks if an achievement is unlocked
func (sd *SaveData) HasAchievement(id string) bool {
	for _, a := range sd.Achievements {
		if a.ID == id {
			return true
		}
	}
	return false
}

// UnlockContent unlocks new content
func (sd *SaveData) UnlockContent(contentType, contentID string) bool {
	var list *[]string

	switch contentType {
	case "location":
		list = &sd.Unlocks.Locations
	case "accessory":
		list = &sd.Unlocks.Accessories
	case "food":
		list = &sd.Unlocks.Foods
	case "toy":
		list = &sd.Unlocks.Toys
	case "background":
		list = &sd.Unlocks.Backgrounds
	case "pet_variant":
		list = &sd.Unlocks.PetVariants
	default:
		return false
	}

	// Check if already unlocked
	for _, item := range *list {
		if item == contentID {
			return false
		}
	}

	*list = append(*list, contentID)
	return true
}

// IsContentUnlocked checks if content is unlocked
func (sd *SaveData) IsContentUnlocked(contentType, contentID string) bool {
	var list []string

	switch contentType {
	case "location":
		list = sd.Unlocks.Locations
	case "accessory":
		list = sd.Unlocks.Accessories
	case "food":
		list = sd.Unlocks.Foods
	case "toy":
		list = sd.Unlocks.Toys
	case "background":
		list = sd.Unlocks.Backgrounds
	case "pet_variant":
		list = sd.Unlocks.PetVariants
	default:
		return false
	}

	for _, item := range list {
		if item == contentID {
			return true
		}
	}
	return false
}

// UpdateStatistics updates game statistics
func (sd *SaveData) UpdateStatistics(playTime float64, interactions int) {
	sd.Statistics.TotalPlayTime += playTime
	sd.Statistics.TotalInteractions += interactions
}

// CalculateChecksum generates a checksum for data integrity
func (sd *SaveData) CalculateChecksum() string {
	// Create a copy without the checksum field
	data, err := json.Marshal(struct {
		Version     string          `json:"version"`
		SaveID      string          `json:"save_id"`
		UserID      string          `json:"user_id"`
		PetData     json.RawMessage `json:"pet_data"`
		SaveCount   int             `json:"save_count"`
	}{
		Version:   sd.Version,
		SaveID:    sd.SaveID,
		UserID:    sd.UserID,
		PetData:   sd.PetData,
		SaveCount: sd.SaveCount,
	})
	if err != nil {
		return ""
	}

	hash := sha256.Sum256(data)
	return fmt.Sprintf("%x", hash[:16]) // First 16 bytes as hex
}

// ValidateChecksum verifies data integrity
func (sd *SaveData) ValidateChecksum() bool {
	return sd.Checksum == sd.CalculateChecksum()
}

// PrepareSave prepares the save data for writing
func (sd *SaveData) PrepareSave() {
	sd.LastSaveTime = time.Now()
	sd.SaveCount++
	sd.SyncStatus.LocalVersion++
	sd.SyncStatus.PendingChanges = true
	sd.Checksum = sd.CalculateChecksum()
}

// ToJSON serializes save data to JSON
func (sd *SaveData) ToJSON() ([]byte, error) {
	return json.MarshalIndent(sd, "", "  ")
}

// FromJSON deserializes save data from JSON
func FromJSON(data []byte) (*SaveData, error) {
	var sd SaveData
	if err := json.Unmarshal(data, &sd); err != nil {
		return nil, fmt.Errorf("failed to unmarshal save data: %w", err)
	}
	return &sd, nil
}

// ToEncryptedJSON serializes and encrypts save data
func (sd *SaveData) ToEncryptedJSON(key []byte) ([]byte, error) {
	plaintext, err := sd.ToJSON()
	if err != nil {
		return nil, err
	}
	return encrypt(plaintext, key)
}

// FromEncryptedJSON decrypts and deserializes save data
func FromEncryptedJSON(data, key []byte) (*SaveData, error) {
	plaintext, err := decrypt(data, key)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt save data: %w", err)
	}
	return FromJSON(plaintext)
}

// Helper functions

func generateSaveID() string {
	b := make([]byte, 8)
	rand.Read(b)
	return fmt.Sprintf("save_%x", b)
}

func encrypt(plaintext, key []byte) ([]byte, error) {
	// Ensure key is 32 bytes (AES-256)
	keyHash := sha256.Sum256(key)

	block, err := aes.NewCipher(keyHash[:])
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}

	return gcm.Seal(nonce, nonce, plaintext, nil), nil
}

func decrypt(ciphertext, key []byte) ([]byte, error) {
	keyHash := sha256.Sum256(key)

	block, err := aes.NewCipher(keyHash[:])
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, fmt.Errorf("ciphertext too short")
	}

	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	return gcm.Open(nil, nonce, ciphertext, nil)
}
