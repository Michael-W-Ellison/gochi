package data

import (
	"errors"
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
	SavePath   string
	BackupPath string
	CachePath  string

	// Configuration
	EncryptionEnabled bool
	EncryptionKey     []byte
	MaxBackups        int
	AutoBackup        bool

	// State
	CurrentSave *SaveData
	LastError   error
	IsDirty     bool
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

// Saves are private: files and directories are readable only by the
// user running the game (on systems with Unix permissions)
const (
	filePerm = 0o600
	dirPerm  = 0o700
)

// ensureDirectories creates necessary directories, readable only by their
// owner (directories made by older versions are made private too)
func (dm *DataManager) ensureDirectories() error {
	dirs := []string{dm.SavePath, dm.BackupPath, dm.CachePath}
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, dirPerm); err != nil {
			return err
		}
		if err := os.Chmod(dir, dirPerm); err != nil {
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
		if err := dm.createBackupLocked(filename); err != nil {
			dm.LastError = err // The save is still written; its backups are only older
		}
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
	if err := writeFileAtomic(savePath, data, filePerm); err != nil {
		dm.LastError = err
		return fmt.Errorf("failed to write save file: %w", err)
	}

	dm.CurrentSave = saveData
	dm.IsDirty = false
	return nil
}

// writeFileAtomic writes data to a temporary file in the same directory and
// renames it over path, so an interrupted write never leaves a partial file
func writeFileAtomic(path string, data []byte, perm os.FileMode) error {
	tmp, err := os.CreateTemp(filepath.Dir(path), filepath.Base(path)+".tmp-*")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()

	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close() // The write's error is the one to report
		_ = os.Remove(tmpPath)
		return err
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close() // The write's error is the one to report
		_ = os.Remove(tmpPath)
		return err
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpPath) // The first error is the one to report
		return err
	}
	if err := os.Chmod(tmpPath, perm); err != nil {
		_ = os.Remove(tmpPath) // The first error is the one to report
		return err
	}
	if err := os.Rename(tmpPath, path); err != nil {
		_ = os.Remove(tmpPath) // The first error is the one to report
		return err
	}
	return nil
}

// LoadLatest loads the most recent valid save for filename. It tries the
// save file first, then its backups from newest to oldest, skipping any that
// cannot be read, cannot be parsed or fail checksum validation. It returns
// the save data and the path it was loaded from. If neither the save file
// nor any backup exists, the error wraps os.ErrNotExist.
func (dm *DataManager) LoadLatest(filename string) (*SaveData, string, error) {
	return dm.LoadLatestUsable(filename, nil)
}

// LoadLatestUsable is LoadLatest that also skips saves usable rejects
// (for one whose contents are damaged although the file is intact). A
// nil usable accepts every save that reads.
func (dm *DataManager) LoadLatestUsable(filename string, usable func(*SaveData) error) (*SaveData, string, error) {
	dm.mu.Lock()
	defer dm.mu.Unlock()

	candidates := []string{dm.getSavePath(filename)}
	backups, err := dm.listBackupsLocked(filename)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, "", err
	}
	for _, backup := range backups {
		candidates = append(candidates, backup.Path)
	}

	var failures []string
	for _, path := range candidates {
		saveData, err := dm.loadPathLocked(path)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err == nil && usable != nil {
			err = usable(saveData)
		}
		if err != nil {
			failures = append(failures, fmt.Sprintf("%s: %v", path, err))
			continue
		}

		dm.CurrentSave = saveData
		dm.LastError = nil
		if len(failures) > 0 {
			dm.LastError = fmt.Errorf("skipped unusable saves: %s", strings.Join(failures, "; "))
		}
		return saveData, path, nil
	}

	if len(failures) == 0 {
		return nil, "", fmt.Errorf("no save found for %s: %w", filename, os.ErrNotExist)
	}
	dm.LastError = fmt.Errorf("no usable save found: %s", strings.Join(failures, "; "))
	return nil, "", dm.LastError
}

// loadPathLocked reads, parses and validates a save file at path
func (dm *DataManager) loadPathLocked(path string) (*SaveData, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var saveData *SaveData
	if dm.EncryptionEnabled && dm.EncryptionKey != nil {
		saveData, err = FromEncryptedJSON(data, dm.EncryptionKey)
	} else {
		saveData, err = FromJSON(data)
	}
	if err != nil {
		return nil, err
	}

	if !saveData.ValidateChecksum() {
		return nil, fmt.Errorf("checksum mismatch")
	}
	return saveData, nil
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

// DeleteWithBackups permanently removes a save file and all of its backups.
// It succeeds if the save file is already gone; the error wraps
// os.ErrNotExist only if neither the save nor any backup existed.
func (dm *DataManager) DeleteWithBackups(filename string) error {
	dm.mu.Lock()
	defer dm.mu.Unlock()

	found := false
	if err := os.Remove(dm.getSavePath(filename)); err == nil {
		found = true
	} else if !errors.Is(err, os.ErrNotExist) {
		dm.LastError = err
		return fmt.Errorf("failed to delete save file: %w", err)
	}

	backups, err := dm.listBackupsLocked(filename)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	var failures []string
	for _, backup := range backups {
		if err := os.Remove(backup.Path); err != nil && !errors.Is(err, os.ErrNotExist) {
			failures = append(failures, err.Error())
			continue
		}
		found = true
	}
	if len(failures) > 0 {
		dm.LastError = fmt.Errorf("failed to delete backups: %s", strings.Join(failures, "; "))
		return dm.LastError
	}

	if !found {
		return fmt.Errorf("no save found for %s: %w", filename, os.ErrNotExist)
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
			Filename:   file.Name(),
			Path:       filepath.Join(dm.SavePath, file.Name()),
			Size:       info.Size(),
			ModifiedAt: info.ModTime(),
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
	timestamp := time.Now().Format(backupTimestampFormat)
	backupName := fmt.Sprintf("%s_%s.bak", strings.TrimSuffix(filename, ".json"), timestamp)
	backupPath := filepath.Join(dm.BackupPath, backupName)

	if err := os.WriteFile(backupPath, data, filePerm); err != nil {
		return fmt.Errorf("failed to create backup: %w", err)
	}

	// Clean up old backups
	dm.cleanupOldBackupsLocked(filename)

	return nil
}

// backupTimestampFormat is the timestamp in backup names: <save>_<timestamp>.bak
const backupTimestampFormat = "20060102_150405"

// isBackupOf reports whether name is a backup of the save named prefix.
// The timestamp must match exactly, so backups of "pet" are not confused
// with backups of another save such as "pet_123".
func isBackupOf(name, prefix string) bool {
	rest, ok := strings.CutPrefix(name, prefix+"_")
	if !ok {
		return false
	}
	timestamp, ok := strings.CutSuffix(rest, ".bak")
	if !ok {
		return false
	}
	_, err := time.Parse(backupTimestampFormat, timestamp)
	return err == nil
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
		if isBackupOf(file.Name(), prefix) {
			backups = append(backups, file)
		}
	}

	// Sort by name (which includes timestamp)
	sort.Slice(backups, func(i, j int) bool {
		return backups[i].Name() > backups[j].Name()
	})

	// Remove old backups
	for i := dm.MaxBackups; i < len(backups); i++ {
		_ = os.Remove(filepath.Join(dm.BackupPath, backups[i].Name())) // Left for the next save to try again
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

	if err := os.WriteFile(savePath, data, filePerm); err != nil {
		return fmt.Errorf("failed to restore backup: %w", err)
	}

	return nil
}

// ListBackups returns all backup files for a save
func (dm *DataManager) ListBackups(filename string) ([]SaveFileInfo, error) {
	dm.mu.RLock()
	defer dm.mu.RUnlock()

	return dm.listBackupsLocked(filename)
}

// listBackupsLocked returns backups for a save, newest first; caller must hold dm.mu
func (dm *DataManager) listBackupsLocked(filename string) ([]SaveFileInfo, error) {
	prefix := strings.TrimSuffix(filename, ".json")

	files, err := os.ReadDir(dm.BackupPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read backup directory: %w", err)
	}

	var backups []SaveFileInfo
	for _, file := range files {
		if !isBackupOf(file.Name(), prefix) {
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

	var errs []error
	for _, file := range files {
		if err := os.Remove(filepath.Join(dm.CachePath, file.Name())); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
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

	if err := os.WriteFile(exportPath, data, filePerm); err != nil {
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
	if err := os.WriteFile(savePath, data, filePerm); err != nil {
		return fmt.Errorf("failed to import save: %w", err)
	}

	return nil
}
