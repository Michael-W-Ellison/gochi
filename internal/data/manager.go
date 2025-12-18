package data

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

// DataManager handles all data persistence operations
type DataManager struct {
	mu sync.RWMutex

	// Paths
	SavePath    string
	BackupPath  string
	CachePath   string

	// Configuration
	EncryptionEnabled bool
	EncryptionKey     []byte
	MaxBackups        int
	AutoBackup        bool

	// State
	CurrentSave  *SaveData
	LastError    error
	IsDirty      bool
}

// DataManagerConfig contains configuration for DataManager
type DataManagerConfig struct {
	SavePath          string
	BackupPath        string
	CachePath         string
	EncryptionEnabled bool
	EncryptionKey     []byte
	MaxBackups        int
	AutoBackup        bool
}

// DefaultDataManagerConfig returns default configuration
func DefaultDataManagerConfig() DataManagerConfig {
	homeDir, _ := os.UserHomeDir()
	basePath := filepath.Join(homeDir, ".gochi")

	return DataManagerConfig{
		SavePath:          filepath.Join(basePath, "saves"),
		BackupPath:        filepath.Join(basePath, "backups"),
		CachePath:         filepath.Join(basePath, "cache"),
		EncryptionEnabled: false,
		EncryptionKey:     nil,
		MaxBackups:        5,
		AutoBackup:        true,
	}
}

// NewDataManager creates a new data manager with default configuration
func NewDataManager() (*DataManager, error) {
	return NewDataManagerWithConfig(DefaultDataManagerConfig())
}

// NewDataManagerWithConfig creates a new data manager with custom configuration
func NewDataManagerWithConfig(config DataManagerConfig) (*DataManager, error) {
	dm := &DataManager{
		SavePath:          config.SavePath,
		BackupPath:        config.BackupPath,
		CachePath:         config.CachePath,
		EncryptionEnabled: config.EncryptionEnabled,
		EncryptionKey:     config.EncryptionKey,
		MaxBackups:        config.MaxBackups,
		AutoBackup:        config.AutoBackup,
	}

	// Create directories
	if err := dm.ensureDirectories(); err != nil {
		return nil, fmt.Errorf("failed to create directories: %w", err)
	}

	return dm, nil
}

// ensureDirectories creates necessary directories
func (dm *DataManager) ensureDirectories() error {
	dirs := []string{dm.SavePath, dm.BackupPath, dm.CachePath}
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	}
	return nil
}

// Save writes save data to disk
func (dm *DataManager) Save(saveData *SaveData, filename string) error {
	dm.mu.Lock()
	defer dm.mu.Unlock()

	// Prepare save data
	saveData.PrepareSave()

	// Create backup if enabled
	if dm.AutoBackup {
		dm.createBackupLocked(filename)
	}

	// Serialize
	var data []byte
	var err error

	if dm.EncryptionEnabled && dm.EncryptionKey != nil {
		data, err = saveData.ToEncryptedJSON(dm.EncryptionKey)
	} else {
		data, err = saveData.ToJSON()
	}

	if err != nil {
		dm.LastError = err
		return fmt.Errorf("failed to serialize save data: %w", err)
	}

	// Write to file
	savePath := dm.getSavePath(filename)
	if err := os.WriteFile(savePath, data, 0644); err != nil {
		dm.LastError = err
		return fmt.Errorf("failed to write save file: %w", err)
	}

	dm.CurrentSave = saveData
	dm.IsDirty = false
	return nil
}

// Load reads save data from disk
func (dm *DataManager) Load(filename string) (*SaveData, error) {
	dm.mu.Lock()
	defer dm.mu.Unlock()

	savePath := dm.getSavePath(filename)

	data, err := os.ReadFile(savePath)
	if err != nil {
		dm.LastError = err
		return nil, fmt.Errorf("failed to read save file: %w", err)
	}

	var saveData *SaveData
	if dm.EncryptionEnabled && dm.EncryptionKey != nil {
		saveData, err = FromEncryptedJSON(data, dm.EncryptionKey)
	} else {
		saveData, err = FromJSON(data)
	}

	if err != nil {
		dm.LastError = err
		return nil, fmt.Errorf("failed to parse save data: %w", err)
	}

	// Validate checksum
	if !saveData.ValidateChecksum() {
		// Data may be corrupted, but still return it with warning
		dm.LastError = fmt.Errorf("save data checksum mismatch - data may be corrupted")
	}

	dm.CurrentSave = saveData
	return saveData, nil
}

// Delete removes a save file
func (dm *DataManager) Delete(filename string) error {
	dm.mu.Lock()
	defer dm.mu.Unlock()

	savePath := dm.getSavePath(filename)
	if err := os.Remove(savePath); err != nil {
		dm.LastError = err
		return fmt.Errorf("failed to delete save file: %w", err)
	}

	return nil
}

// ListSaves returns all available save files
func (dm *DataManager) ListSaves() ([]SaveFileInfo, error) {
	dm.mu.RLock()
	defer dm.mu.RUnlock()

	files, err := os.ReadDir(dm.SavePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read save directory: %w", err)
	}

	var saves []SaveFileInfo
	for _, file := range files {
		if file.IsDir() || !strings.HasSuffix(file.Name(), ".json") {
			continue
		}

		info, err := file.Info()
		if err != nil {
			continue
		}

		saves = append(saves, SaveFileInfo{
			Filename:    file.Name(),
			Path:        filepath.Join(dm.SavePath, file.Name()),
			Size:        info.Size(),
			ModifiedAt:  info.ModTime(),
		})
	}

	// Sort by modification time (newest first)
	sort.Slice(saves, func(i, j int) bool {
		return saves[i].ModifiedAt.After(saves[j].ModifiedAt)
	})

	return saves, nil
}

// SaveFileInfo contains information about a save file
type SaveFileInfo struct {
	Filename   string
	Path       string
	Size       int64
	ModifiedAt time.Time
}

// Exists checks if a save file exists
func (dm *DataManager) Exists(filename string) bool {
	dm.mu.RLock()
	defer dm.mu.RUnlock()

	savePath := dm.getSavePath(filename)
	_, err := os.Stat(savePath)
	return err == nil
}

// CreateBackup creates a backup of a save file
func (dm *DataManager) CreateBackup(filename string) error {
	dm.mu.Lock()
	defer dm.mu.Unlock()

	return dm.createBackupLocked(filename)
}

func (dm *DataManager) createBackupLocked(filename string) error {
	savePath := dm.getSavePath(filename)

	// Check if file exists
	data, err := os.ReadFile(savePath)
	if err != nil {
		return nil // No file to backup
	}

	// Create backup with timestamp
	timestamp := time.Now().Format("20060102_150405")
	backupName := fmt.Sprintf("%s_%s.bak", strings.TrimSuffix(filename, ".json"), timestamp)
	backupPath := filepath.Join(dm.BackupPath, backupName)

	if err := os.WriteFile(backupPath, data, 0644); err != nil {
		return fmt.Errorf("failed to create backup: %w", err)
	}

	// Clean up old backups
	dm.cleanupOldBackupsLocked(filename)

	return nil
}

// cleanupOldBackupsLocked removes old backups exceeding MaxBackups
func (dm *DataManager) cleanupOldBackupsLocked(filename string) {
	prefix := strings.TrimSuffix(filename, ".json")

	files, err := os.ReadDir(dm.BackupPath)
	if err != nil {
		return
	}

	var backups []os.DirEntry
	for _, file := range files {
		if strings.HasPrefix(file.Name(), prefix) && strings.HasSuffix(file.Name(), ".bak") {
			backups = append(backups, file)
		}
	}

	// Sort by name (which includes timestamp)
	sort.Slice(backups, func(i, j int) bool {
		return backups[i].Name() > backups[j].Name()
	})

	// Remove old backups
	for i := dm.MaxBackups; i < len(backups); i++ {
		os.Remove(filepath.Join(dm.BackupPath, backups[i].Name()))
	}
}

// RestoreBackup restores a save from backup
func (dm *DataManager) RestoreBackup(backupFilename, targetFilename string) error {
	dm.mu.Lock()
	defer dm.mu.Unlock()

	backupPath := filepath.Join(dm.BackupPath, backupFilename)
	savePath := dm.getSavePath(targetFilename)

	data, err := os.ReadFile(backupPath)
	if err != nil {
		return fmt.Errorf("failed to read backup: %w", err)
	}

	if err := os.WriteFile(savePath, data, 0644); err != nil {
		return fmt.Errorf("failed to restore backup: %w", err)
	}

	return nil
}

// ListBackups returns all backup files for a save
func (dm *DataManager) ListBackups(filename string) ([]SaveFileInfo, error) {
	dm.mu.RLock()
	defer dm.mu.RUnlock()

	prefix := strings.TrimSuffix(filename, ".json")

	files, err := os.ReadDir(dm.BackupPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read backup directory: %w", err)
	}

	var backups []SaveFileInfo
	for _, file := range files {
		if !strings.HasPrefix(file.Name(), prefix) || !strings.HasSuffix(file.Name(), ".bak") {
			continue
		}

		info, err := file.Info()
		if err != nil {
			continue
		}

		backups = append(backups, SaveFileInfo{
			Filename:   file.Name(),
			Path:       filepath.Join(dm.BackupPath, file.Name()),
			Size:       info.Size(),
			ModifiedAt: info.ModTime(),
		})
	}

	// Sort by modification time (newest first)
	sort.Slice(backups, func(i, j int) bool {
		return backups[i].ModifiedAt.After(backups[j].ModifiedAt)
	})

	return backups, nil
}

// SetEncryption enables or disables encryption
func (dm *DataManager) SetEncryption(enabled bool, key []byte) {
	dm.mu.Lock()
	defer dm.mu.Unlock()

	dm.EncryptionEnabled = enabled
	dm.EncryptionKey = key
}

// GetCurrentSave returns the currently loaded save
func (dm *DataManager) GetCurrentSave() *SaveData {
	dm.mu.RLock()
	defer dm.mu.RUnlock()

	return dm.CurrentSave
}

// MarkDirty marks the current save as having unsaved changes
func (dm *DataManager) MarkDirty() {
	dm.mu.Lock()
	defer dm.mu.Unlock()

	dm.IsDirty = true
}

// HasUnsavedChanges returns whether there are unsaved changes
func (dm *DataManager) HasUnsavedChanges() bool {
	dm.mu.RLock()
	defer dm.mu.RUnlock()

	return dm.IsDirty
}

// GetLastError returns the last error encountered
func (dm *DataManager) GetLastError() error {
	dm.mu.RLock()
	defer dm.mu.RUnlock()

	return dm.LastError
}

// ClearCache removes all cached data
func (dm *DataManager) ClearCache() error {
	dm.mu.Lock()
	defer dm.mu.Unlock()

	files, err := os.ReadDir(dm.CachePath)
	if err != nil {
		return fmt.Errorf("failed to read cache directory: %w", err)
	}

	for _, file := range files {
		os.Remove(filepath.Join(dm.CachePath, file.Name()))
	}

	return nil
}

// GetStorageStats returns storage usage statistics
func (dm *DataManager) GetStorageStats() StorageStats {
	dm.mu.RLock()
	defer dm.mu.RUnlock()

	stats := StorageStats{}

	// Calculate saves size
	if files, err := os.ReadDir(dm.SavePath); err == nil {
		for _, f := range files {
			if info, err := f.Info(); err == nil {
				stats.SavesSize += info.Size()
				stats.SavesCount++
			}
		}
	}

	// Calculate backups size
	if files, err := os.ReadDir(dm.BackupPath); err == nil {
		for _, f := range files {
			if info, err := f.Info(); err == nil {
				stats.BackupsSize += info.Size()
				stats.BackupsCount++
			}
		}
	}

	// Calculate cache size
	if files, err := os.ReadDir(dm.CachePath); err == nil {
		for _, f := range files {
			if info, err := f.Info(); err == nil {
				stats.CacheSize += info.Size()
			}
		}
	}

	stats.TotalSize = stats.SavesSize + stats.BackupsSize + stats.CacheSize

	return stats
}

// StorageStats contains storage usage information
type StorageStats struct {
	SavesSize    int64
	SavesCount   int
	BackupsSize  int64
	BackupsCount int
	CacheSize    int64
	TotalSize    int64
}

// getSavePath returns the full path for a save file
func (dm *DataManager) getSavePath(filename string) string {
	if !strings.HasSuffix(filename, ".json") {
		filename += ".json"
	}
	return filepath.Join(dm.SavePath, filename)
}

// ExportSave exports a save to a specific location
func (dm *DataManager) ExportSave(filename, exportPath string) error {
	dm.mu.RLock()
	defer dm.mu.RUnlock()

	savePath := dm.getSavePath(filename)

	data, err := os.ReadFile(savePath)
	if err != nil {
		return fmt.Errorf("failed to read save file: %w", err)
	}

	if err := os.WriteFile(exportPath, data, 0644); err != nil {
		return fmt.Errorf("failed to export save: %w", err)
	}

	return nil
}

// ImportSave imports a save from an external location
func (dm *DataManager) ImportSave(importPath, filename string) error {
	dm.mu.Lock()
	defer dm.mu.Unlock()

	data, err := os.ReadFile(importPath)
	if err != nil {
		return fmt.Errorf("failed to read import file: %w", err)
	}

	// Validate the data
	_, err = FromJSON(data)
	if err != nil {
		return fmt.Errorf("invalid save data format: %w", err)
	}

	savePath := dm.getSavePath(filename)
	if err := os.WriteFile(savePath, data, 0644); err != nil {
		return fmt.Errorf("failed to import save: %w", err)
	}

	return nil
}
