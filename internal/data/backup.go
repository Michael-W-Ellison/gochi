package data

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/Michael-W-Ellison/gochi/pkg/types"
)

// BackupManager handles backup creation and restoration
type BackupManager struct {
	mu sync.RWMutex

	backupPath      string
	maxBackups      int
	autoBackupInterval time.Duration
	storage         *LocalStorage
}

// BackupInfo contains metadata about a backup
type BackupInfo struct {
	Filename    string
	CreatedAt   time.Time
	PetIDs      []types.PetID
	Size        int64
	Compressed  bool
}

// NewBackupManager creates a new backup manager
func NewBackupManager(backupPath string, storage *LocalStorage) *BackupManager {
	return &BackupManager{
		backupPath:         backupPath,
		maxBackups:         10, // Keep last 10 backups
		autoBackupInterval: 24 * time.Hour, // Daily backups
		storage:            storage,
	}
}

// SetMaxBackups sets the maximum number of backups to retain
func (bm *BackupManager) SetMaxBackups(max int) {
	bm.mu.Lock()
	defer bm.mu.Unlock()
	bm.maxBackups = max
}

// SetAutoBackupInterval sets how often automatic backups occur
func (bm *BackupManager) SetAutoBackupInterval(interval time.Duration) {
	bm.mu.Lock()
	defer bm.mu.Unlock()
	bm.autoBackupInterval = interval
}

// CreateBackup creates a backup of all pets
func (bm *BackupManager) CreateBackup() (string, error) {
	bm.mu.Lock()
	defer bm.mu.Unlock()

	// Ensure backup directory exists
	if err := os.MkdirAll(bm.backupPath, 0755); err != nil {
		return "", fmt.Errorf("failed to create backup directory: %w", err)
	}

	// Create backup filename with timestamp
	timestamp := time.Now().Format("20060102_150405")
	backupFilename := filepath.Join(bm.backupPath, fmt.Sprintf("backup_%s.zip", timestamp))

	// Create zip file
	zipFile, err := os.Create(backupFilename)
	if err != nil {
		return "", fmt.Errorf("failed to create backup file: %w", err)
	}
	defer zipFile.Close()

	zipWriter := zip.NewWriter(zipFile)
	defer zipWriter.Close()

	// Get all pet files
	petIDs, err := bm.storage.ListPets()
	if err != nil {
		return "", fmt.Errorf("failed to list pets: %w", err)
	}

	// Add each pet file to the zip
	for _, petID := range petIDs {
		filename := bm.storage.getFilename(petID)
		if err := bm.addFileToZip(zipWriter, filename); err != nil {
			return "", fmt.Errorf("failed to add %s to backup: %w", filename, err)
		}
	}

	// Cleanup old backups
	if err := bm.cleanupOldBackups(); err != nil {
		// Log error but don't fail the backup
		fmt.Printf("Warning: failed to cleanup old backups: %v\n", err)
	}

	return backupFilename, nil
}

// CreateBackupForPet creates a backup of a specific pet
func (bm *BackupManager) CreateBackupForPet(petID types.PetID) (string, error) {
	bm.mu.Lock()
	defer bm.mu.Unlock()

	// Ensure backup directory exists
	if err := os.MkdirAll(bm.backupPath, 0755); err != nil {
		return "", fmt.Errorf("failed to create backup directory: %w", err)
	}

	// Create backup filename
	timestamp := time.Now().Format("20060102_150405")
	backupFilename := filepath.Join(bm.backupPath,
		fmt.Sprintf("backup_%s_%s.zip", string(petID), timestamp))

	// Create zip file
	zipFile, err := os.Create(backupFilename)
	if err != nil {
		return "", fmt.Errorf("failed to create backup file: %w", err)
	}
	defer zipFile.Close()

	zipWriter := zip.NewWriter(zipFile)
	defer zipWriter.Close()

	// Add pet file to zip
	filename := bm.storage.getFilename(petID)
	if err := bm.addFileToZip(zipWriter, filename); err != nil {
		return "", fmt.Errorf("failed to add pet file to backup: %w", err)
	}

	return backupFilename, nil
}

// RestoreBackup restores all pets from a backup file
func (bm *BackupManager) RestoreBackup(backupFilename string) ([]types.PetID, error) {
	bm.mu.Lock()
	defer bm.mu.Unlock()

	// Open zip file
	zipReader, err := zip.OpenReader(backupFilename)
	if err != nil {
		return nil, fmt.Errorf("failed to open backup file: %w", err)
	}
	defer zipReader.Close()

	restoredPets := make([]types.PetID, 0)

	// Extract each file
	for _, file := range zipReader.File {
		if file.FileInfo().IsDir() {
			continue
		}

		// Only restore .json files
		if !strings.HasSuffix(file.Name, ".json") {
			continue
		}

		// Open file in zip
		rc, err := file.Open()
		if err != nil {
			return nil, fmt.Errorf("failed to open file in backup: %w", err)
		}

		// Create destination file
		destPath := filepath.Join(bm.storage.basePath, filepath.Base(file.Name))
		destFile, err := os.Create(destPath)
		if err != nil {
			rc.Close()
			return nil, fmt.Errorf("failed to create destination file: %w", err)
		}

		// Copy data
		_, err = io.Copy(destFile, rc)
		rc.Close()
		destFile.Close()

		if err != nil {
			return nil, fmt.Errorf("failed to copy file data: %w", err)
		}

		// Extract pet ID from filename
		baseName := filepath.Base(file.Name)
		petID := types.PetID(baseName[:len(baseName)-5]) // Remove ".json"
		restoredPets = append(restoredPets, petID)
	}

	return restoredPets, nil
}

// ListBackups returns information about all available backups
func (bm *BackupManager) ListBackups() ([]BackupInfo, error) {
	bm.mu.RLock()
	defer bm.mu.RUnlock()

	entries, err := os.ReadDir(bm.backupPath)
	if err != nil {
		if os.IsNotExist(err) {
			return []BackupInfo{}, nil
		}
		return nil, fmt.Errorf("failed to read backup directory: %w", err)
	}

	backups := make([]BackupInfo, 0)
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".zip") {
			continue
		}

		info, err := entry.Info()
		if err != nil {
			continue
		}

		backupInfo := BackupInfo{
			Filename:   filepath.Join(bm.backupPath, entry.Name()),
			CreatedAt:  info.ModTime(),
			Size:       info.Size(),
			Compressed: true,
		}

		// Try to read pet IDs from the zip
		petIDs, err := bm.getPetIDsFromBackup(backupInfo.Filename)
		if err == nil {
			backupInfo.PetIDs = petIDs
		}

		backups = append(backups, backupInfo)
	}

	// Sort by creation time (newest first)
	sort.Slice(backups, func(i, j int) bool {
		return backups[i].CreatedAt.After(backups[j].CreatedAt)
	})

	return backups, nil
}

// DeleteBackup removes a backup file
func (bm *BackupManager) DeleteBackup(filename string) error {
	bm.mu.Lock()
	defer bm.mu.Unlock()

	if err := os.Remove(filename); err != nil {
		return fmt.Errorf("failed to delete backup: %w", err)
	}

	return nil
}

// GetLatestBackup returns the most recent backup
func (bm *BackupManager) GetLatestBackup() (*BackupInfo, error) {
	backups, err := bm.ListBackups()
	if err != nil {
		return nil, err
	}

	if len(backups) == 0 {
		return nil, fmt.Errorf("no backups found")
	}

	return &backups[0], nil
}

// Helper methods

func (bm *BackupManager) addFileToZip(zipWriter *zip.Writer, filename string) error {
	// Open source file
	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	// Get file info
	info, err := file.Stat()
	if err != nil {
		return err
	}

	// Create header
	header, err := zip.FileInfoHeader(info)
	if err != nil {
		return err
	}

	// Set compression method
	header.Method = zip.Deflate

	// Use base name for the file in the zip
	header.Name = filepath.Base(filename)

	// Create writer for this file
	writer, err := zipWriter.CreateHeader(header)
	if err != nil {
		return err
	}

	// Copy file contents
	_, err = io.Copy(writer, file)
	return err
}

func (bm *BackupManager) cleanupOldBackups() error {
	backups, err := bm.ListBackups()
	if err != nil {
		return err
	}

	// Remove backups beyond max count
	if len(backups) > bm.maxBackups {
		for i := bm.maxBackups; i < len(backups); i++ {
			if err := os.Remove(backups[i].Filename); err != nil {
				return err
			}
		}
	}

	return nil
}

func (bm *BackupManager) getPetIDsFromBackup(filename string) ([]types.PetID, error) {
	zipReader, err := zip.OpenReader(filename)
	if err != nil {
		return nil, err
	}
	defer zipReader.Close()

	petIDs := make([]types.PetID, 0)
	for _, file := range zipReader.File {
		if !strings.HasSuffix(file.Name, ".json") {
			continue
		}

		baseName := filepath.Base(file.Name)
		petID := types.PetID(baseName[:len(baseName)-5])
		petIDs = append(petIDs, petID)
	}

	return petIDs, nil
}
