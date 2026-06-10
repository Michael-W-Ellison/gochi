package data

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/Michael-W-Ellison/gochi/pkg/types"
)

// DataManager is the central manager for all data operations
type DataManager struct {
	mu sync.RWMutex

	localStorage  *LocalStorage
	cache         *PetCache
	backupManager *BackupManager
	cloudSync     *CloudSyncManager

	autoSaveEnabled bool
	autoSaveInterval time.Duration
	lastAutoSave    time.Time

	autoBackupEnabled bool
	autoBackupInterval time.Duration
	lastAutoBackup time.Time
}

// Config holds configuration for the DataManager
type Config struct {
	BasePath          string
	BackupPath        string
	CacheSizeMB       int
	CacheTTLMinutes   int
	EncryptionKey     string
	EnableEncryption  bool
	EnableAutoSave    bool
	AutoSaveInterval  time.Duration
	EnableAutoBackup  bool
	AutoBackupInterval time.Duration
	MaxBackups        int
	CloudProvider     CloudProvider
}

// DefaultConfig returns a sensible default configuration
func DefaultConfig() *Config {
	return &Config{
		BasePath:          "./data/pets",
		BackupPath:        "./data/backups",
		CacheSizeMB:       100,
		CacheTTLMinutes:   30,
		EnableEncryption:  false,
		EnableAutoSave:    true,
		AutoSaveInterval:  5 * time.Minute,
		EnableAutoBackup:  true,
		AutoBackupInterval: 24 * time.Hour,
		MaxBackups:        10,
	}
}

// NewDataManager creates a new data manager with the given configuration
func NewDataManager(config *Config) (*DataManager, error) {
	// Create local storage
	localStorage := NewLocalStorage(config.BasePath)
	if config.EnableEncryption && config.EncryptionKey != "" {
		if err := localStorage.SetEncryptionKey(config.EncryptionKey); err != nil {
			return nil, fmt.Errorf("failed to set encryption key: %w", err)
		}
	}

	// Create cache
	cache := NewPetCache(config.CacheSizeMB, config.CacheTTLMinutes)

	// Create backup manager
	backupManager := NewBackupManager(config.BackupPath, localStorage)
	backupManager.SetMaxBackups(config.MaxBackups)
	backupManager.SetAutoBackupInterval(config.AutoBackupInterval)

	// Create cloud sync manager (optional)
	var cloudSync *CloudSyncManager
	if config.CloudProvider != nil {
		cloudSync = NewCloudSyncManager(config.CloudProvider, localStorage)
	}

	dm := &DataManager{
		localStorage:       localStorage,
		cache:              cache,
		backupManager:      backupManager,
		cloudSync:          cloudSync,
		autoSaveEnabled:    config.EnableAutoSave,
		autoSaveInterval:   config.AutoSaveInterval,
		autoBackupEnabled:  config.EnableAutoBackup,
		autoBackupInterval: config.AutoBackupInterval,
	}

	return dm, nil
}

// SavePet saves a pet to local storage and cache
func (dm *DataManager) SavePet(petID types.PetID, petData interface{}) error {
	dm.mu.Lock()
	defer dm.mu.Unlock()

	// Save to local storage
	if err := dm.localStorage.Save(petID, petData); err != nil {
		return fmt.Errorf("failed to save to local storage: %w", err)
	}

	// Update cache
	// In a real implementation, we'd calculate actual size
	dm.cache.CachePet(petID, petData, 10240) // Estimate 10KB

	dm.lastAutoSave = time.Now()

	return nil
}

// LoadPet loads a pet from cache or local storage
func (dm *DataManager) LoadPet(petID types.PetID, target interface{}) error {
	dm.mu.RLock()
	defer dm.mu.RUnlock()

	// Load from local storage
	// Note: Cache is currently not used for loads due to type assertion complexity
	// In a production system, we would use proper serialization/deserialization
	if err := dm.localStorage.Load(petID, target); err != nil {
		return fmt.Errorf("failed to load from local storage: %w", err)
	}

	return nil
}

// DeletePet removes a pet from storage and cache
func (dm *DataManager) DeletePet(petID types.PetID) error {
	dm.mu.Lock()
	defer dm.mu.Unlock()

	// Delete from local storage
	if err := dm.localStorage.Delete(petID); err != nil {
		return fmt.Errorf("failed to delete from local storage: %w", err)
	}

	// Remove from cache
	dm.cache.InvalidatePet(petID)

	// Delete from cloud if available
	if dm.cloudSync != nil && dm.cloudSync.provider != nil {
		ctx := context.Background()
		if dm.cloudSync.provider.IsConnected(ctx) {
			dm.cloudSync.provider.Delete(ctx, petID)
		}
	}

	return nil
}

// ListPets returns all saved pet IDs
func (dm *DataManager) ListPets() ([]types.PetID, error) {
	dm.mu.RLock()
	defer dm.mu.RUnlock()

	return dm.localStorage.ListPets()
}

// PetExists checks if a pet save file exists
func (dm *DataManager) PetExists(petID types.PetID) bool {
	dm.mu.RLock()
	defer dm.mu.RUnlock()

	return dm.localStorage.Exists(petID)
}

// CreateBackup creates a backup of all pets
func (dm *DataManager) CreateBackup() (string, error) {
	dm.mu.Lock()
	defer dm.mu.Unlock()

	filename, err := dm.backupManager.CreateBackup()
	if err != nil {
		return "", err
	}

	dm.lastAutoBackup = time.Now()
	return filename, nil
}

// CreateBackupForPet creates a backup of a specific pet
func (dm *DataManager) CreateBackupForPet(petID types.PetID) (string, error) {
	dm.mu.Lock()
	defer dm.mu.Unlock()

	return dm.backupManager.CreateBackupForPet(petID)
}

// RestoreBackup restores pets from a backup file
func (dm *DataManager) RestoreBackup(backupFilename string) ([]types.PetID, error) {
	dm.mu.Lock()
	defer dm.mu.Unlock()

	// Restore from backup
	restoredPets, err := dm.backupManager.RestoreBackup(backupFilename)
	if err != nil {
		return nil, err
	}

	// Clear cache to force reload
	dm.cache.cache.Clear()

	return restoredPets, nil
}

// ListBackups returns all available backups
func (dm *DataManager) ListBackups() ([]BackupInfo, error) {
	dm.mu.RLock()
	defer dm.mu.RUnlock()

	return dm.backupManager.ListBackups()
}

// SyncToCloud synchronizes all pets to cloud storage
func (dm *DataManager) SyncToCloud() (*SyncResult, error) {
	if dm.cloudSync == nil {
		return nil, fmt.Errorf("cloud sync not configured")
	}

	return dm.cloudSync.SyncAll(), nil
}

// SyncPetToCloud synchronizes a specific pet to cloud
func (dm *DataManager) SyncPetToCloud(petID types.PetID) error {
	if dm.cloudSync == nil {
		return fmt.Errorf("cloud sync not configured")
	}

	return dm.cloudSync.SyncPetByID(petID)
}

// EnableAutoSave enables automatic saving
func (dm *DataManager) EnableAutoSave(interval time.Duration) {
	dm.mu.Lock()
	defer dm.mu.Unlock()

	dm.autoSaveEnabled = true
	dm.autoSaveInterval = interval
}

// DisableAutoSave disables automatic saving
func (dm *DataManager) DisableAutoSave() {
	dm.mu.Lock()
	defer dm.mu.Unlock()

	dm.autoSaveEnabled = false
}

// EnableAutoBackup enables automatic backups
func (dm *DataManager) EnableAutoBackup(interval time.Duration) {
	dm.mu.Lock()
	defer dm.mu.Unlock()

	dm.autoBackupEnabled = true
	dm.autoBackupInterval = interval
}

// DisableAutoBackup disables automatic backups
func (dm *DataManager) DisableAutoBackup() {
	dm.mu.Lock()
	defer dm.mu.Unlock()

	dm.autoBackupEnabled = false
}

// PerformMaintenance performs routine maintenance tasks
func (dm *DataManager) PerformMaintenance() error {
	dm.mu.Lock()
	defer dm.mu.Unlock()

	// Cleanup expired cache entries
	dm.cache.CleanupExpired()

	// Check if auto-backup is due
	if dm.autoBackupEnabled && time.Since(dm.lastAutoBackup) > dm.autoBackupInterval {
		if _, err := dm.backupManager.CreateBackup(); err != nil {
			return fmt.Errorf("auto-backup failed: %w", err)
		}
		dm.lastAutoBackup = time.Now()
	}

	// Check if cloud sync is due (if enabled)
	if dm.cloudSync != nil && dm.cloudSync.autoSyncEnabled {
		lastSync := dm.cloudSync.GetLastSyncTime()
		if time.Since(lastSync) > dm.cloudSync.syncInterval {
			dm.cloudSync.SyncAll()
		}
	}

	return nil
}

// GetCacheStats returns cache statistics
func (dm *DataManager) GetCacheStats() CacheStats {
	dm.mu.RLock()
	defer dm.mu.RUnlock()

	return dm.cache.GetStats()
}

// GetLastSyncResult returns the result of the last cloud sync
func (dm *DataManager) GetLastSyncResult() *SyncResult {
	if dm.cloudSync == nil {
		return nil
	}

	return dm.cloudSync.GetLastSyncResult()
}

// ClearCache clears all cached data
func (dm *DataManager) ClearCache() {
	dm.mu.Lock()
	defer dm.mu.Unlock()

	dm.cache.cache.Clear()
}

// InvalidateCache invalidates a specific pet's cache entry
func (dm *DataManager) InvalidateCache(petID types.PetID) {
	dm.mu.Lock()
	defer dm.mu.Unlock()

	dm.cache.InvalidatePet(petID)
}

// SetEncryption enables or disables encryption
func (dm *DataManager) SetEncryption(enabled bool, key string) error {
	dm.mu.Lock()
	defer dm.mu.Unlock()

	if enabled {
		if err := dm.localStorage.SetEncryptionKey(key); err != nil {
			return fmt.Errorf("failed to set encryption key: %w", err)
		}
	} else {
		dm.localStorage.DisableEncryption()
	}

	return nil
}

// ExportPet exports a pet's data to a file
func (dm *DataManager) ExportPet(petID types.PetID, exportPath string) error {
	dm.mu.RLock()
	defer dm.mu.RUnlock()

	// Validate pet exists
	if !dm.localStorage.Exists(petID) {
		return fmt.Errorf("pet %s does not exist", petID)
	}

	// Validate export path is not empty
	if exportPath == "" {
		return fmt.Errorf("export path cannot be empty")
	}

	// Clean and validate export path
	cleanPath := filepath.Clean(exportPath)

	// Ensure export path directory exists
	exportDir := filepath.Dir(cleanPath)
	if err := os.MkdirAll(exportDir, 0755); err != nil {
		return fmt.Errorf("failed to create export directory: %w", err)
	}

	// Read the complete pet data file
	sourceFile := filepath.Join(dm.localStorage.GetBasePath(), string(petID)+".json")
	fileData, err := os.ReadFile(sourceFile)
	if err != nil {
		return fmt.Errorf("failed to read pet file: %w", err)
	}

	// Parse to verify integrity
	var petData PetData
	if err := json.Unmarshal(fileData, &petData); err != nil {
		return fmt.Errorf("failed to parse pet data: %w", err)
	}

	// Verify data integrity with checksum
	currentChecksum := CalculateChecksum(petData.Data)
	if petData.Checksum != currentChecksum {
		return fmt.Errorf("data integrity check failed: checksum mismatch")
	}

	// Write to export location
	if err := os.WriteFile(cleanPath, fileData, 0644); err != nil {
		return fmt.Errorf("failed to write export file: %w", err)
	}

	// Verify the exported file
	exportedData, err := os.ReadFile(cleanPath)
	if err != nil {
		return fmt.Errorf("failed to verify export: %w", err)
	}

	// Parse exported file and verify checksum
	var exportedPetData PetData
	if err := json.Unmarshal(exportedData, &exportedPetData); err != nil {
		os.Remove(cleanPath)
		return fmt.Errorf("export verification failed: corrupted file: %w", err)
	}

	exportChecksum := CalculateChecksum(exportedPetData.Data)
	if exportChecksum != currentChecksum {
		// Clean up the corrupted export
		os.Remove(cleanPath)
		return fmt.Errorf("export verification failed: checksums do not match")
	}

	return nil
}

// ImportPet imports a pet from an exported file
func (dm *DataManager) ImportPet(importPath string) (types.PetID, error) {
	dm.mu.Lock()
	defer dm.mu.Unlock()

	// Validate import path
	if importPath == "" {
		return "", fmt.Errorf("import path cannot be empty")
	}

	// Clean the path
	cleanPath := filepath.Clean(importPath)

	// Check if file exists
	if _, err := os.Stat(cleanPath); err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("import file does not exist: %s", cleanPath)
		}
		return "", fmt.Errorf("failed to access import file: %w", err)
	}

	// Read the import file
	data, err := os.ReadFile(cleanPath)
	if err != nil {
		return "", fmt.Errorf("failed to read import file: %w", err)
	}

	// Parse the pet data to validate format
	var petData PetData
	if err := json.Unmarshal(data, &petData); err != nil {
		return "", fmt.Errorf("invalid pet data format: %w", err)
	}

	// Validate data version compatibility
	if petData.Version != CurrentDataVersion {
		return "", fmt.Errorf("incompatible data version: got %s, expected %s",
			petData.Version, CurrentDataVersion)
	}

	// Verify checksum integrity
	calculatedChecksum := CalculateChecksum(petData.Data)
	if petData.Checksum != calculatedChecksum {
		return "", fmt.Errorf("data integrity check failed: checksum mismatch")
	}

	// Check if pet already exists
	originalID := petData.PetID
	finalID := originalID

	if dm.localStorage.Exists(originalID) {
		// Generate a new unique ID by appending timestamp with nanoseconds for uniqueness
		timestamp := time.Now().UnixNano()
		finalID = types.PetID(fmt.Sprintf("%s_imported_%d", originalID, timestamp))

		// Update the pet data with new ID
		petData.PetID = finalID

		// Re-marshal with new ID
		data, err = json.Marshal(petData)
		if err != nil {
			return "", fmt.Errorf("failed to update pet data with new ID: %w", err)
		}
	}

	// Write to storage location
	basePath := dm.localStorage.GetBasePath()
	targetFile := filepath.Join(basePath, string(finalID)+".json")

	// Ensure directory exists
	if err := os.MkdirAll(basePath, 0755); err != nil {
		return "", fmt.Errorf("failed to create storage directory: %w", err)
	}

	if err := os.WriteFile(targetFile, data, 0644); err != nil {
		return "", fmt.Errorf("failed to write imported pet: %w", err)
	}

	// Verify the imported file
	verifyData, err := os.ReadFile(targetFile)
	if err != nil {
		// Cleanup on verification failure
		os.Remove(targetFile)
		return "", fmt.Errorf("failed to verify import: %w", err)
	}

	verifyChecksum := CalculateChecksum(verifyData)
	originalChecksum := CalculateChecksum(data)
	if verifyChecksum != originalChecksum {
		// Cleanup corrupted import
		os.Remove(targetFile)
		return "", fmt.Errorf("import verification failed: file corrupted during write")
	}

	// Invalidate cache for this pet (already holding dm.mu lock, so call cache directly)
	dm.cache.InvalidatePet(finalID)

	return finalID, nil
}

// GetStorageStats returns statistics about storage usage
func (dm *DataManager) GetStorageStats() (map[string]interface{}, error) {
	dm.mu.RLock()
	defer dm.mu.RUnlock()

	pets, err := dm.localStorage.ListPets()
	if err != nil {
		return nil, err
	}

	stats := map[string]interface{}{
		"total_pets":      len(pets),
		"cache_stats":     dm.cache.GetStats(),
		"last_auto_save":  dm.lastAutoSave,
		"last_auto_backup": dm.lastAutoBackup,
	}

	if dm.cloudSync != nil {
		stats["last_cloud_sync"] = dm.cloudSync.GetLastSyncTime()
		stats["last_sync_result"] = dm.cloudSync.GetLastSyncResult()
	}

	return stats, nil
}
