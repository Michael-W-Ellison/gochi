package data

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"
)

// CloudSyncStatus represents the state of cloud synchronization
type CloudSyncStatus int

const (
	SyncStatusIdle CloudSyncStatus = iota
	SyncStatusSyncing
	SyncStatusSuccess
	SyncStatusError
	SyncStatusConflict
)

// String returns the string representation of CloudSyncStatus
func (s CloudSyncStatus) String() string {
	return [...]string{"Idle", "Syncing", "Success", "Error", "Conflict"}[s]
}

// CloudProvider interface for cloud storage backends
type CloudProvider interface {
	// Upload uploads data to cloud storage
	Upload(userID string, data []byte) error

	// Download retrieves data from cloud storage
	Download(userID string) ([]byte, error)

	// GetMetadata retrieves metadata about stored data
	GetMetadata(userID string) (*CloudMetadata, error)

	// Delete removes data from cloud storage
	Delete(userID string) error

	// IsAvailable checks if the cloud service is reachable
	IsAvailable() bool
}

// CloudMetadata contains information about cloud-stored data
type CloudMetadata struct {
	UserID        string    `json:"user_id"`
	LastModified  time.Time `json:"last_modified"`
	Version       int       `json:"version"`
	Size          int64     `json:"size"`
	Checksum      string    `json:"checksum"`
}

// SyncConflict represents a conflict between local and cloud data
type SyncConflict struct {
	LocalData     *SaveData
	CloudData     *SaveData
	LocalVersion  int
	CloudVersion  int
	DetectedAt    time.Time
}

// ConflictResolution specifies how to resolve a sync conflict
type ConflictResolution int

const (
	ResolveKeepLocal ConflictResolution = iota
	ResolveKeepCloud
	ResolveMerge
)

// CloudSyncService handles cloud synchronization
type CloudSyncService struct {
	mu sync.RWMutex

	// Configuration
	Provider      CloudProvider
	UserID        string
	Enabled       bool
	AutoSync      bool
	SyncInterval  time.Duration

	// State
	Status        CloudSyncStatus
	LastSync      time.Time
	LastError     error
	PendingSync   bool
	CurrentConflict *SyncConflict

	// Callbacks
	OnSyncComplete func(success bool, err error)
	OnConflict     func(conflict *SyncConflict)

	// Internal
	stopChan      chan struct{}
	syncTicker    *time.Ticker
}

// NewCloudSyncService creates a new cloud sync service
func NewCloudSyncService(provider CloudProvider, userID string) *CloudSyncService {
	return &CloudSyncService{
		Provider:     provider,
		UserID:       userID,
		Enabled:      false,
		AutoSync:     false,
		SyncInterval: 5 * time.Minute,
		Status:       SyncStatusIdle,
	}
}

// Enable enables cloud synchronization
func (cs *CloudSyncService) Enable() error {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	if cs.Provider == nil {
		return fmt.Errorf("no cloud provider configured")
	}

	if !cs.Provider.IsAvailable() {
		return fmt.Errorf("cloud service is not available")
	}

	cs.Enabled = true
	return nil
}

// Disable disables cloud synchronization
func (cs *CloudSyncService) Disable() {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	cs.Enabled = false
	cs.StopAutoSync()
}

// StartAutoSync starts automatic background synchronization
func (cs *CloudSyncService) StartAutoSync() {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	if cs.syncTicker != nil {
		return // Already running
	}

	cs.AutoSync = true
	cs.syncTicker = time.NewTicker(cs.SyncInterval)
	cs.stopChan = make(chan struct{})

	go func() {
		for {
			select {
			case <-cs.syncTicker.C:
				if cs.PendingSync {
					cs.Sync(nil)
				}
			case <-cs.stopChan:
				return
			}
		}
	}()
}

// StopAutoSync stops automatic synchronization
func (cs *CloudSyncService) StopAutoSync() {
	if cs.syncTicker != nil {
		cs.syncTicker.Stop()
		cs.syncTicker = nil
	}

	if cs.stopChan != nil {
		close(cs.stopChan)
		cs.stopChan = nil
	}

	cs.AutoSync = false
}

// Sync synchronizes local data with cloud
func (cs *CloudSyncService) Sync(localData *SaveData) error {
	cs.mu.Lock()
	if !cs.Enabled {
		cs.mu.Unlock()
		return fmt.Errorf("cloud sync is not enabled")
	}

	if cs.Status == SyncStatusSyncing {
		cs.mu.Unlock()
		return fmt.Errorf("sync already in progress")
	}

	cs.Status = SyncStatusSyncing
	cs.mu.Unlock()

	// Perform sync
	err := cs.performSync(localData)

	cs.mu.Lock()
	if err != nil {
		cs.Status = SyncStatusError
		cs.LastError = err
	} else {
		cs.Status = SyncStatusSuccess
		cs.LastSync = time.Now()
		cs.PendingSync = false
	}
	cs.mu.Unlock()

	// Call completion callback
	if cs.OnSyncComplete != nil {
		cs.OnSyncComplete(err == nil, err)
	}

	return err
}

// performSync does the actual synchronization work
func (cs *CloudSyncService) performSync(localData *SaveData) error {
	if localData == nil {
		return fmt.Errorf("no local data to sync")
	}

	// Get cloud metadata
	cloudMeta, err := cs.Provider.GetMetadata(cs.UserID)
	if err != nil {
		// No cloud data exists, upload local
		return cs.uploadToCloud(localData)
	}

	// Check versions
	localVersion := localData.SyncStatus.LocalVersion
	cloudVersion := cloudMeta.Version

	if localVersion > cloudVersion {
		// Local is newer, upload
		return cs.uploadToCloud(localData)
	} else if cloudVersion > localVersion {
		// Cloud is newer, we have a potential conflict
		cloudData, err := cs.downloadFromCloud()
		if err != nil {
			return err
		}

		// Check if local has pending changes
		if localData.SyncStatus.PendingChanges {
			// Conflict detected
			cs.handleConflict(localData, cloudData, localVersion, cloudVersion)
			return fmt.Errorf("sync conflict detected")
		}

		// No local changes, cloud wins
		return nil
	}

	// Versions match, check checksums
	if localData.Checksum != cloudMeta.Checksum {
		// Data differs despite same version - conflict
		cloudData, err := cs.downloadFromCloud()
		if err != nil {
			return err
		}
		cs.handleConflict(localData, cloudData, localVersion, cloudVersion)
		return fmt.Errorf("sync conflict detected")
	}

	// Everything in sync
	return nil
}

// uploadToCloud uploads save data to cloud
func (cs *CloudSyncService) uploadToCloud(data *SaveData) error {
	jsonData, err := data.ToJSON()
	if err != nil {
		return fmt.Errorf("failed to serialize data: %w", err)
	}

	if err := cs.Provider.Upload(cs.UserID, jsonData); err != nil {
		return fmt.Errorf("failed to upload to cloud: %w", err)
	}

	// Update sync status
	data.SyncStatus.LastSyncTime = time.Now()
	data.SyncStatus.CloudVersion = data.SyncStatus.LocalVersion
	data.SyncStatus.PendingChanges = false

	return nil
}

// downloadFromCloud retrieves save data from cloud
func (cs *CloudSyncService) downloadFromCloud() (*SaveData, error) {
	data, err := cs.Provider.Download(cs.UserID)
	if err != nil {
		return nil, fmt.Errorf("failed to download from cloud: %w", err)
	}

	saveData, err := FromJSON(data)
	if err != nil {
		return nil, fmt.Errorf("failed to parse cloud data: %w", err)
	}

	return saveData, nil
}

// handleConflict creates and stores a sync conflict
func (cs *CloudSyncService) handleConflict(localData, cloudData *SaveData, localVersion, cloudVersion int) {
	cs.mu.Lock()
	cs.CurrentConflict = &SyncConflict{
		LocalData:    localData,
		CloudData:    cloudData,
		LocalVersion: localVersion,
		CloudVersion: cloudVersion,
		DetectedAt:   time.Now(),
	}
	cs.Status = SyncStatusConflict
	cs.mu.Unlock()

	if cs.OnConflict != nil {
		cs.OnConflict(cs.CurrentConflict)
	}
}

// ResolveConflict resolves a sync conflict
func (cs *CloudSyncService) ResolveConflict(resolution ConflictResolution) (*SaveData, error) {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	if cs.CurrentConflict == nil {
		return nil, fmt.Errorf("no conflict to resolve")
	}

	conflict := cs.CurrentConflict
	var result *SaveData

	switch resolution {
	case ResolveKeepLocal:
		result = conflict.LocalData
		// Force upload local version
		result.SyncStatus.LocalVersion = conflict.CloudVersion + 1

	case ResolveKeepCloud:
		result = conflict.CloudData
		result.SyncStatus.LocalVersion = conflict.CloudVersion

	case ResolveMerge:
		// Merge strategy: keep newer individual elements
		result = cs.mergeConflict(conflict)
	}

	cs.CurrentConflict = nil
	cs.Status = SyncStatusIdle

	return result, nil
}

// mergeConflict attempts to merge conflicting data
func (cs *CloudSyncService) mergeConflict(conflict *SyncConflict) *SaveData {
	local := conflict.LocalData
	cloud := conflict.CloudData

	// Use the one with more recent save time as base
	var merged *SaveData
	if local.LastSaveTime.After(cloud.LastSaveTime) {
		merged = local
	} else {
		merged = cloud
	}

	// Merge achievements (union of both)
	achievementMap := make(map[string]Achievement)
	for _, a := range local.Achievements {
		achievementMap[a.ID] = a
	}
	for _, a := range cloud.Achievements {
		if existing, exists := achievementMap[a.ID]; exists {
			// Keep the one unlocked first
			if a.UnlockedAt.Before(existing.UnlockedAt) {
				achievementMap[a.ID] = a
			}
		} else {
			achievementMap[a.ID] = a
		}
	}
	merged.Achievements = make([]Achievement, 0, len(achievementMap))
	for _, a := range achievementMap {
		merged.Achievements = append(merged.Achievements, a)
	}

	// Merge unlocks (union of both)
	merged.Unlocks = mergeUnlocks(local.Unlocks, cloud.Unlocks)

	// Merge statistics (take maximum values)
	merged.Statistics = mergeStatistics(local.Statistics, cloud.Statistics)

	// Set version higher than both
	merged.SyncStatus.LocalVersion = max(local.SyncStatus.LocalVersion, cloud.SyncStatus.LocalVersion) + 1

	return merged
}

// mergeUnlocks combines unlock lists
func mergeUnlocks(local, cloud UnlockableContent) UnlockableContent {
	return UnlockableContent{
		Locations:   uniqueStrings(append(local.Locations, cloud.Locations...)),
		Accessories: uniqueStrings(append(local.Accessories, cloud.Accessories...)),
		Foods:       uniqueStrings(append(local.Foods, cloud.Foods...)),
		Toys:        uniqueStrings(append(local.Toys, cloud.Toys...)),
		Backgrounds: uniqueStrings(append(local.Backgrounds, cloud.Backgrounds...)),
		PetVariants: uniqueStrings(append(local.PetVariants, cloud.PetVariants...)),
	}
}

// mergeStatistics combines statistics taking max values
func mergeStatistics(local, cloud GameStatistics) GameStatistics {
	return GameStatistics{
		TotalPlayTime:      maxFloat(local.TotalPlayTime, cloud.TotalPlayTime),
		PetsCreated:        maxInt(local.PetsCreated, cloud.PetsCreated),
		PetsLost:           maxInt(local.PetsLost, cloud.PetsLost),
		TotalInteractions:  maxInt(local.TotalInteractions, cloud.TotalInteractions),
		LongestLifespan:    maxFloat(local.LongestLifespan, cloud.LongestLifespan),
		HighestHappiness:   maxFloat(local.HighestHappiness, cloud.HighestHappiness),
		FoodFed:            maxInt(local.FoodFed, cloud.FoodFed),
		GamesPlayed:        maxInt(local.GamesPlayed, cloud.GamesPlayed),
		AchievementsEarned: maxInt(local.AchievementsEarned, cloud.AchievementsEarned),
		FirstPlayDate:      minTime(local.FirstPlayDate, cloud.FirstPlayDate),
		CurrentStreak:      maxInt(local.CurrentStreak, cloud.CurrentStreak),
		LongestStreak:      maxInt(local.LongestStreak, cloud.LongestStreak),
	}
}

// GetStatus returns the current sync status
func (cs *CloudSyncService) GetStatus() CloudSyncStatus {
	cs.mu.RLock()
	defer cs.mu.RUnlock()
	return cs.Status
}

// GetLastSyncTime returns the time of last successful sync
func (cs *CloudSyncService) GetLastSyncTime() time.Time {
	cs.mu.RLock()
	defer cs.mu.RUnlock()
	return cs.LastSync
}

// MarkPendingSync marks that there are changes to sync
func (cs *CloudSyncService) MarkPendingSync() {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	cs.PendingSync = true
}

// HasConflict returns whether there's an unresolved conflict
func (cs *CloudSyncService) HasConflict() bool {
	cs.mu.RLock()
	defer cs.mu.RUnlock()
	return cs.CurrentConflict != nil
}

// GetConflict returns the current conflict if any
func (cs *CloudSyncService) GetConflict() *SyncConflict {
	cs.mu.RLock()
	defer cs.mu.RUnlock()
	return cs.CurrentConflict
}

// MockCloudProvider is a mock implementation for testing
type MockCloudProvider struct {
	mu        sync.RWMutex
	Storage   map[string][]byte
	Metadata  map[string]*CloudMetadata
	Available bool
}

// NewMockCloudProvider creates a new mock cloud provider
func NewMockCloudProvider() *MockCloudProvider {
	return &MockCloudProvider{
		Storage:   make(map[string][]byte),
		Metadata:  make(map[string]*CloudMetadata),
		Available: true,
	}
}

func (m *MockCloudProvider) Upload(userID string, data []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.Available {
		return fmt.Errorf("cloud service unavailable")
	}

	m.Storage[userID] = data

	// Parse to get version
	var saveData SaveData
	if err := json.Unmarshal(data, &saveData); err == nil {
		m.Metadata[userID] = &CloudMetadata{
			UserID:       userID,
			LastModified: time.Now(),
			Version:      saveData.SyncStatus.LocalVersion,
			Size:         int64(len(data)),
			Checksum:     saveData.Checksum,
		}
	}

	return nil
}

func (m *MockCloudProvider) Download(userID string) ([]byte, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if !m.Available {
		return nil, fmt.Errorf("cloud service unavailable")
	}

	data, exists := m.Storage[userID]
	if !exists {
		return nil, fmt.Errorf("no data found for user")
	}

	return data, nil
}

func (m *MockCloudProvider) GetMetadata(userID string) (*CloudMetadata, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if !m.Available {
		return nil, fmt.Errorf("cloud service unavailable")
	}

	meta, exists := m.Metadata[userID]
	if !exists {
		return nil, fmt.Errorf("no metadata found for user")
	}

	return meta, nil
}

func (m *MockCloudProvider) Delete(userID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.Available {
		return fmt.Errorf("cloud service unavailable")
	}

	delete(m.Storage, userID)
	delete(m.Metadata, userID)
	return nil
}

func (m *MockCloudProvider) IsAvailable() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.Available
}

// Helper functions

func uniqueStrings(strs []string) []string {
	seen := make(map[string]bool)
	result := make([]string, 0)
	for _, s := range strs {
		if !seen[s] {
			seen[s] = true
			result = append(result, s)
		}
	}
	return result
}

func maxFloat(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func minTime(a, b time.Time) time.Time {
	if a.Before(b) {
		return a
	}
	return b
}
