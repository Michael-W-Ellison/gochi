package data

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/Michael-W-Ellison/gochi/pkg/types"
)

// DataVersion represents the version of the data format
const CurrentDataVersion = "1.0.0"

// PetData wraps pet data with metadata for persistence
type PetData struct {
	Version     string      `json:"version"`
	PetID       types.PetID `json:"pet_id"`
	SavedAt     time.Time   `json:"saved_at"`
	Checksum    string      `json:"checksum"`
	Encrypted   bool        `json:"encrypted"`
	Data        []byte      `json:"data"` // JSON-encoded pet data
	Metadata    SaveMetadata `json:"metadata"`
}

// SaveMetadata contains additional information about the save
type SaveMetadata struct {
	DeviceID     string  `json:"device_id"`
	AppVersion   string  `json:"app_version"`
	DataSize     int64   `json:"data_size"`
	CompressionUsed bool `json:"compression_used"`
}

// LocalStorage manages local file-based persistence
type LocalStorage struct {
	mu sync.RWMutex

	basePath    string
	encryptionKey []byte
	useEncryption bool
}

// NewLocalStorage creates a new local storage manager
func NewLocalStorage(basePath string) *LocalStorage {
	return &LocalStorage{
		basePath:      basePath,
		useEncryption: false,
	}
}

// SetEncryptionKey enables encryption for saved data
func (ls *LocalStorage) SetEncryptionKey(key string) error {
	if len(key) == 0 {
		return fmt.Errorf("encryption key cannot be empty")
	}

	// Derive a 32-byte key from the provided string
	hash := sha256.Sum256([]byte(key))
	ls.encryptionKey = hash[:]
	ls.useEncryption = true

	return nil
}

// DisableEncryption turns off encryption
func (ls *LocalStorage) DisableEncryption() {
	ls.useEncryption = false
	ls.encryptionKey = nil
}

// Save persists pet data to disk
func (ls *LocalStorage) Save(petID types.PetID, data interface{}) error {
	ls.mu.Lock()
	defer ls.mu.Unlock()

	// Serialize pet data to JSON
	jsonData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal pet data: %w", err)
	}

	// Calculate checksum
	checksum := calculateChecksum(jsonData)

	// Create pet data wrapper
	petData := &PetData{
		Version:   CurrentDataVersion,
		PetID:     petID,
		SavedAt:   time.Now(),
		Checksum:  checksum,
		Encrypted: ls.useEncryption,
		Data:      jsonData,
		Metadata: SaveMetadata{
			DeviceID:        getDeviceID(),
			AppVersion:      "0.1.0-alpha",
			DataSize:        int64(len(jsonData)),
			CompressionUsed: false,
		},
	}

	// Encrypt if enabled
	if ls.useEncryption {
		encryptedData, err := ls.encrypt(jsonData)
		if err != nil {
			return fmt.Errorf("failed to encrypt data: %w", err)
		}
		petData.Data = encryptedData
	}

	// Marshal the wrapper
	finalData, err := json.MarshalIndent(petData, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal pet data wrapper: %w", err)
	}

	// Ensure directory exists
	if err := os.MkdirAll(ls.basePath, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Write to file
	filename := ls.getFilename(petID)
	if err := os.WriteFile(filename, finalData, 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

// Load retrieves pet data from disk
func (ls *LocalStorage) Load(petID types.PetID, target interface{}) error {
	ls.mu.RLock()
	defer ls.mu.RUnlock()

	filename := ls.getFilename(petID)

	// Read file
	fileData, err := os.ReadFile(filename)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("save file not found: %w", err)
		}
		return fmt.Errorf("failed to read file: %w", err)
	}

	// Unmarshal wrapper
	var petData PetData
	if err := json.Unmarshal(fileData, &petData); err != nil {
		return fmt.Errorf("failed to unmarshal pet data wrapper: %w", err)
	}

	// Version check
	if petData.Version != CurrentDataVersion {
		return fmt.Errorf("data version mismatch: expected %s, got %s",
			CurrentDataVersion, petData.Version)
	}

	// Decrypt if needed
	data := petData.Data
	if petData.Encrypted {
		if !ls.useEncryption {
			return fmt.Errorf("data is encrypted but no encryption key provided")
		}
		decryptedData, err := ls.decrypt(data)
		if err != nil {
			return fmt.Errorf("failed to decrypt data: %w", err)
		}
		data = decryptedData
	}

	// Verify checksum
	actualChecksum := calculateChecksum(data)
	if actualChecksum != petData.Checksum {
		return fmt.Errorf("checksum mismatch: data may be corrupted")
	}

	// Unmarshal pet data
	if err := json.Unmarshal(data, target); err != nil {
		return fmt.Errorf("failed to unmarshal pet data: %w", err)
	}

	return nil
}

// Delete removes a pet's save file
func (ls *LocalStorage) Delete(petID types.PetID) error {
	ls.mu.Lock()
	defer ls.mu.Unlock()

	filename := ls.getFilename(petID)
	if err := os.Remove(filename); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("save file not found: %w", err)
		}
		return fmt.Errorf("failed to delete file: %w", err)
	}

	return nil
}

// Exists checks if a save file exists for a pet
func (ls *LocalStorage) Exists(petID types.PetID) bool {
	ls.mu.RLock()
	defer ls.mu.RUnlock()

	filename := ls.getFilename(petID)
	_, err := os.Stat(filename)
	return err == nil
}

// ListPets returns a list of all saved pet IDs
func (ls *LocalStorage) ListPets() ([]types.PetID, error) {
	ls.mu.RLock()
	defer ls.mu.RUnlock()

	entries, err := os.ReadDir(ls.basePath)
	if err != nil {
		if os.IsNotExist(err) {
			return []types.PetID{}, nil
		}
		return nil, fmt.Errorf("failed to read directory: %w", err)
	}

	petIDs := make([]types.PetID, 0)
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if filepath.Ext(entry.Name()) == ".json" {
			// Extract pet ID from filename
			name := entry.Name()
			petID := types.PetID(name[:len(name)-5]) // Remove ".json"
			petIDs = append(petIDs, petID)
		}
	}

	return petIDs, nil
}

// GetSaveInfo returns metadata about a save file without loading the full data
func (ls *LocalStorage) GetSaveInfo(petID types.PetID) (*PetData, error) {
	ls.mu.RLock()
	defer ls.mu.RUnlock()

	filename := ls.getFilename(petID)
	fileData, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	var petData PetData
	if err := json.Unmarshal(fileData, &petData); err != nil {
		return nil, fmt.Errorf("failed to unmarshal pet data: %w", err)
	}

	// Clear the actual data to save memory
	petData.Data = nil

	return &petData, nil
}

// Helper methods

func (ls *LocalStorage) getFilename(petID types.PetID) string {
	return filepath.Join(ls.basePath, string(petID)+".json")
}

func (ls *LocalStorage) encrypt(data []byte) ([]byte, error) {
	block, err := aes.NewCipher(ls.encryptionKey)
	if err != nil {
		return nil, err
	}

	// Create GCM mode
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	// Create nonce
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}

	// Encrypt
	ciphertext := gcm.Seal(nonce, nonce, data, nil)
	return ciphertext, nil
}

func (ls *LocalStorage) decrypt(data []byte) ([]byte, error) {
	block, err := aes.NewCipher(ls.encryptionKey)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return nil, fmt.Errorf("ciphertext too short")
	}

	nonce, ciphertext := data[:nonceSize], data[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, err
	}

	return plaintext, nil
}

func calculateChecksum(data []byte) string {
	hash := sha256.Sum256(data)
	return fmt.Sprintf("%x", hash)
}

func getDeviceID() string {
	// In a real implementation, this would get a unique device identifier
	// For now, we'll use a simple placeholder
	hostname, err := os.Hostname()
	if err != nil {
		hostname = "unknown"
	}
	return hostname
}
