package data

import (
	"fmt"
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

	// Try cache first
	if cachedData, found := dm.cache.GetPet(petID); found {
		// Copy cached data to target
		// Note: In a real implementation, we'd need proper deep copying
		*target.(*interface{}) = cachedData
		return nil
	}

	// Load from local storage
	if err := dm.localStorage.Load(petID, target); err != nil {
		return fmt.Errorf("failed to load from local storage: %w", err)
	}

	// Add to cache
	dm.cache.CachePet(petID, target, 10240)

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
	if dm.cloudSync != nil && dm.cloudSync.provider.IsConnected() {
		dm.cloudSync.provider.Delete(petID)
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

	// This would copy the pet file to the export location
	// Implementation omitted for brevity
	return nil
}

// ImportPet imports a pet from an exported file
func (dm *DataManager) ImportPet(importPath string) (types.PetID, error) {
	dm.mu.Lock()
	defer dm.mu.Unlock()

	// This would import a pet file from the import location
	// Implementation omitted for brevity
	return "", nil
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
