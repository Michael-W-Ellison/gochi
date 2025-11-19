package data

import (
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/Michael-W-Ellison/gochi/pkg/types"
)

// SyncStatus represents the synchronization state
type SyncStatus int

const (
	SyncStatusIdle SyncStatus = iota
	SyncStatusSyncing
	SyncStatusSuccess
	SyncStatusFailed
)

func (s SyncStatus) String() string {
	return [...]string{
		"Idle", "Syncing", "Success", "Failed",
	}[s]
}

// SyncResult contains the outcome of a sync operation
type SyncResult struct {
	Status        SyncStatus
	SyncedPets    []types.PetID
	FailedPets    []types.PetID
	StartTime     time.Time
	EndTime       time.Time
	BytesUploaded int64
	BytesDownloaded int64
	Errors        []string
}

// CloudProvider defines the interface for cloud storage providers
type CloudProvider interface {
	// Upload uploads pet data to the cloud
	Upload(petID types.PetID, data []byte) error

	// Download retrieves pet data from the cloud
	Download(petID types.PetID) ([]byte, error)

	// Delete removes pet data from the cloud
	Delete(petID types.PetID) error

	// List returns all pet IDs stored in the cloud
	List() ([]types.PetID, error)

	// GetLastModified returns when a pet was last modified in the cloud
	GetLastModified(petID types.PetID) (time.Time, error)

	// IsConnected checks if the cloud service is reachable
	IsConnected() bool
}

// CloudSyncManager handles synchronization between local and cloud storage
type CloudSyncManager struct {
	mu sync.RWMutex

	provider       CloudProvider
	localStorage   *LocalStorage
	autoSyncEnabled bool
	syncInterval   time.Duration
	lastSyncTime   time.Time
	lastSyncResult *SyncResult
}

// NewCloudSyncManager creates a new cloud sync manager
func NewCloudSyncManager(provider CloudProvider, localStorage *LocalStorage) *CloudSyncManager {
	return &CloudSyncManager{
		provider:       provider,
		localStorage:   localStorage,
		autoSyncEnabled: false,
		syncInterval:   30 * time.Minute,
	}
}

// EnableAutoSync enables automatic synchronization
func (csm *CloudSyncManager) EnableAutoSync(interval time.Duration) {
	csm.mu.Lock()
	defer csm.mu.Unlock()

	csm.autoSyncEnabled = true
	csm.syncInterval = interval
}

// DisableAutoSync disables automatic synchronization
func (csm *CloudSyncManager) DisableAutoSync() {
	csm.mu.Lock()
	defer csm.mu.Unlock()

	csm.autoSyncEnabled = false
}

// SyncAll synchronizes all pets between local and cloud
func (csm *CloudSyncManager) SyncAll() *SyncResult {
	csm.mu.Lock()
	defer csm.mu.Unlock()

	result := &SyncResult{
		Status:         SyncStatusSyncing,
		StartTime:      time.Now(),
		SyncedPets:     make([]types.PetID, 0),
		FailedPets:     make([]types.PetID, 0),
		Errors:         make([]string, 0),
	}

	// Check connection
	if !csm.provider.IsConnected() {
		result.Status = SyncStatusFailed
		result.Errors = append(result.Errors, "Cloud provider not connected")
		result.EndTime = time.Now()
		csm.lastSyncResult = result
		return result
	}

	// Get local pets
	localPets, err := csm.localStorage.ListPets()
	if err != nil {
		result.Status = SyncStatusFailed
		result.Errors = append(result.Errors, fmt.Sprintf("Failed to list local pets: %v", err))
		result.EndTime = time.Now()
		csm.lastSyncResult = result
		return result
	}

	// Get cloud pets
	cloudPets, err := csm.provider.List()
	if err != nil {
		result.Status = SyncStatusFailed
		result.Errors = append(result.Errors, fmt.Sprintf("Failed to list cloud pets: %v", err))
		result.EndTime = time.Now()
		csm.lastSyncResult = result
		return result
	}

	// Create maps for easy lookup
	localMap := make(map[types.PetID]bool)
	for _, petID := range localPets {
		localMap[petID] = true
	}

	cloudMap := make(map[types.PetID]bool)
	for _, petID := range cloudPets {
		cloudMap[petID] = true
	}

	// Sync pets that exist in both locations
	for _, petID := range localPets {
		if cloudMap[petID] {
			if err := csm.syncPet(petID, result); err != nil {
				result.FailedPets = append(result.FailedPets, petID)
				result.Errors = append(result.Errors,
					fmt.Sprintf("Failed to sync %s: %v", petID, err))
			} else {
				result.SyncedPets = append(result.SyncedPets, petID)
			}
		} else {
			// Upload new local pet to cloud
			if err := csm.uploadPet(petID, result); err != nil {
				result.FailedPets = append(result.FailedPets, petID)
				result.Errors = append(result.Errors,
					fmt.Sprintf("Failed to upload %s: %v", petID, err))
			} else {
				result.SyncedPets = append(result.SyncedPets, petID)
			}
		}
	}

	// Download cloud-only pets
	for _, petID := range cloudPets {
		if !localMap[petID] {
			if err := csm.downloadPet(petID, result); err != nil {
				result.FailedPets = append(result.FailedPets, petID)
				result.Errors = append(result.Errors,
					fmt.Sprintf("Failed to download %s: %v", petID, err))
			} else {
				result.SyncedPets = append(result.SyncedPets, petID)
			}
		}
	}

	result.EndTime = time.Now()
	if len(result.FailedPets) == 0 {
		result.Status = SyncStatusSuccess
	} else {
		result.Status = SyncStatusFailed
	}

	csm.lastSyncTime = time.Now()
	csm.lastSyncResult = result

	return result
}

// SyncPetByID synchronizes a specific pet
func (csm *CloudSyncManager) SyncPetByID(petID types.PetID) error {
	csm.mu.Lock()
	defer csm.mu.Unlock()

	result := &SyncResult{
		StartTime: time.Now(),
	}

	return csm.syncPet(petID, result)
}

// UploadPet uploads a pet to the cloud
func (csm *CloudSyncManager) UploadPet(petID types.PetID) error {
	csm.mu.Lock()
	defer csm.mu.Unlock()

	result := &SyncResult{
		StartTime: time.Now(),
	}

	return csm.uploadPet(petID, result)
}

// DownloadPet downloads a pet from the cloud
func (csm *CloudSyncManager) DownloadPet(petID types.PetID) error {
	csm.mu.Lock()
	defer csm.mu.Unlock()

	result := &SyncResult{
		StartTime: time.Now(),
	}

	return csm.downloadPet(petID, result)
}

// GetLastSyncResult returns the result of the last sync operation
func (csm *CloudSyncManager) GetLastSyncResult() *SyncResult {
	csm.mu.RLock()
	defer csm.mu.RUnlock()

	return csm.lastSyncResult
}

// GetLastSyncTime returns when the last sync occurred
func (csm *CloudSyncManager) GetLastSyncTime() time.Time {
	csm.mu.RLock()
	defer csm.mu.RUnlock()

	return csm.lastSyncTime
}

// Helper methods

func (csm *CloudSyncManager) syncPet(petID types.PetID, result *SyncResult) error {
	// Get local save info
	localInfo, err := csm.localStorage.GetSaveInfo(petID)
	if err != nil {
		return fmt.Errorf("failed to get local save info: %w", err)
	}

	// Get cloud last modified time
	cloudModTime, err := csm.provider.GetLastModified(petID)
	if err != nil {
		// If not found in cloud, upload
		return csm.uploadPet(petID, result)
	}

	// Compare timestamps and sync the newer version
	if localInfo.SavedAt.After(cloudModTime) {
		// Local is newer, upload
		return csm.uploadPet(petID, result)
	} else if cloudModTime.After(localInfo.SavedAt) {
		// Cloud is newer, download
		return csm.downloadPet(petID, result)
	}

	// Files are in sync
	return nil
}

func (csm *CloudSyncManager) uploadPet(petID types.PetID, result *SyncResult) error {
	// Read local file
	filename := csm.localStorage.getFilename(petID)
	data, err := os.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("failed to read local file: %w", err)
	}

	// Upload to cloud
	if err := csm.provider.Upload(petID, data); err != nil {
		return fmt.Errorf("failed to upload to cloud: %w", err)
	}

	result.BytesUploaded += int64(len(data))
	return nil
}

func (csm *CloudSyncManager) downloadPet(petID types.PetID, result *SyncResult) error {
	// Download from cloud
	data, err := csm.provider.Download(petID)
	if err != nil {
		return fmt.Errorf("failed to download from cloud: %w", err)
	}

	// Save locally
	filename := csm.localStorage.getFilename(petID)
	if err := os.WriteFile(filename, data, 0644); err != nil {
		return fmt.Errorf("failed to write local file: %w", err)
	}

	result.BytesDownloaded += int64(len(data))
	return nil
}

// StubCloudProvider is a mock implementation for testing
type StubCloudProvider struct {
	mu        sync.RWMutex
	data      map[types.PetID][]byte
	modTimes  map[types.PetID]time.Time
	connected bool
}

// NewStubCloudProvider creates a stub cloud provider
func NewStubCloudProvider() *StubCloudProvider {
	return &StubCloudProvider{
		data:      make(map[types.PetID][]byte),
		modTimes:  make(map[types.PetID]time.Time),
		connected: true,
	}
}

func (s *StubCloudProvider) Upload(petID types.PetID, data []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.data[petID] = data
	s.modTimes[petID] = time.Now()
	return nil
}

func (s *StubCloudProvider) Download(petID types.PetID) ([]byte, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	data, exists := s.data[petID]
	if !exists {
		return nil, fmt.Errorf("pet not found in cloud")
	}
	return data, nil
}

func (s *StubCloudProvider) Delete(petID types.PetID) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.data, petID)
	delete(s.modTimes, petID)
	return nil
}

func (s *StubCloudProvider) List() ([]types.PetID, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	petIDs := make([]types.PetID, 0, len(s.data))
	for petID := range s.data {
		petIDs = append(petIDs, petID)
	}
	return petIDs, nil
}

func (s *StubCloudProvider) GetLastModified(petID types.PetID) (time.Time, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	modTime, exists := s.modTimes[petID]
	if !exists {
		return time.Time{}, fmt.Errorf("pet not found in cloud")
	}
	return modTime, nil
}

func (s *StubCloudProvider) IsConnected() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.connected
}

func (s *StubCloudProvider) SetConnected(connected bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.connected = connected
}
