package data

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Michael-W-Ellison/gochi/pkg/types"
)

// Test data structure
type TestPetData struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Age  int    `json:"age"`
}

func TestNewLocalStorage(t *testing.T) {
	tempDir := t.TempDir()
	storage := NewLocalStorage(tempDir)

	if storage == nil {
		t.Fatal("NewLocalStorage returned nil")
	}

	if storage.basePath != tempDir {
		t.Errorf("basePath = %s, want %s", storage.basePath, tempDir)
	}

	if storage.useEncryption {
		t.Error("useEncryption should be false by default")
	}
}

func TestSaveAndLoad(t *testing.T) {
	tempDir := t.TempDir()
	storage := NewLocalStorage(tempDir)

	petID := types.PetID("test_pet_123")
	testData := &TestPetData{
		ID:   "test_pet_123",
		Name: "Fluffy",
		Age:  3,
	}

	// Save
	err := storage.Save(petID, testData)
	if err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// Verify file exists
	if !storage.Exists(petID) {
		t.Error("File should exist after save")
	}

	// Load
	var loadedData TestPetData
	err = storage.Load(petID, &loadedData)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	// Verify data
	if loadedData.ID != testData.ID {
		t.Errorf("ID = %s, want %s", loadedData.ID, testData.ID)
	}

	if loadedData.Name != testData.Name {
		t.Errorf("Name = %s, want %s", loadedData.Name, testData.Name)
	}

	if loadedData.Age != testData.Age {
		t.Errorf("Age = %d, want %d", loadedData.Age, testData.Age)
	}
}

func TestSaveWithEncryption(t *testing.T) {
	tempDir := t.TempDir()
	storage := NewLocalStorage(tempDir)

	// Enable encryption
	err := storage.SetEncryptionKey("test-encryption-key-12345")
	if err != nil {
		t.Fatalf("SetEncryptionKey failed: %v", err)
	}

	if !storage.useEncryption {
		t.Error("useEncryption should be true after setting key")
	}

	petID := types.PetID("encrypted_pet")
	testData := &TestPetData{
		ID:   "encrypted_pet",
		Name: "Secure Pet",
		Age:  5,
	}

	// Save with encryption
	err = storage.Save(petID, testData)
	if err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// Load with encryption
	var loadedData TestPetData
	err = storage.Load(petID, &loadedData)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	// Verify data
	if loadedData.Name != testData.Name {
		t.Errorf("Name = %s, want %s", loadedData.Name, testData.Name)
	}

	// Try to load without encryption key (should fail)
	storage2 := NewLocalStorage(tempDir)
	var failedData TestPetData
	err = storage2.Load(petID, &failedData)
	if err == nil {
		t.Error("Load should fail without encryption key")
	}
}

func TestDelete(t *testing.T) {
	tempDir := t.TempDir()
	storage := NewLocalStorage(tempDir)

	petID := types.PetID("delete_test_pet")
	testData := &TestPetData{
		ID:   "delete_test_pet",
		Name: "Temporary",
		Age:  1,
	}

	// Save
	err := storage.Save(petID, testData)
	if err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// Verify exists
	if !storage.Exists(petID) {
		t.Error("File should exist after save")
	}

	// Delete
	err = storage.Delete(petID)
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	// Verify doesn't exist
	if storage.Exists(petID) {
		t.Error("File should not exist after delete")
	}
}

func TestListPets(t *testing.T) {
	tempDir := t.TempDir()
	storage := NewLocalStorage(tempDir)

	// Save multiple pets
	petIDs := []types.PetID{"pet1", "pet2", "pet3"}
	for _, petID := range petIDs {
		testData := &TestPetData{
			ID:   string(petID),
			Name: string(petID),
			Age:  1,
		}
		err := storage.Save(petID, testData)
		if err != nil {
			t.Fatalf("Save failed for %s: %v", petID, err)
		}
	}

	// List pets
	listedPets, err := storage.ListPets()
	if err != nil {
		t.Fatalf("ListPets failed: %v", err)
	}

	if len(listedPets) != len(petIDs) {
		t.Errorf("ListPets returned %d pets, want %d", len(listedPets), len(petIDs))
	}

	// Verify all pets are in the list
	petMap := make(map[types.PetID]bool)
	for _, petID := range listedPets {
		petMap[petID] = true
	}

	for _, petID := range petIDs {
		if !petMap[petID] {
			t.Errorf("Pet %s not found in list", petID)
		}
	}
}

func TestGetSaveInfo(t *testing.T) {
	tempDir := t.TempDir()
	storage := NewLocalStorage(tempDir)

	petID := types.PetID("info_test_pet")
	testData := &TestPetData{
		ID:   "info_test_pet",
		Name: "Info Pet",
		Age:  2,
	}

	// Save
	err := storage.Save(petID, testData)
	if err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// Get save info
	info, err := storage.GetSaveInfo(petID)
	if err != nil {
		t.Fatalf("GetSaveInfo failed: %v", err)
	}

	if info.PetID != petID {
		t.Errorf("PetID = %s, want %s", info.PetID, petID)
	}

	if info.Version != CurrentDataVersion {
		t.Errorf("Version = %s, want %s", info.Version, CurrentDataVersion)
	}

	if info.Data != nil {
		t.Error("Data should be nil in save info")
	}

	if info.Metadata.DataSize <= 0 {
		t.Error("DataSize should be positive")
	}
}

func TestLoadNonExistent(t *testing.T) {
	tempDir := t.TempDir()
	storage := NewLocalStorage(tempDir)

	petID := types.PetID("nonexistent_pet")
	var data TestPetData

	err := storage.Load(petID, &data)
	if err == nil {
		t.Error("Load should fail for nonexistent pet")
	}
}

func TestDeleteNonExistent(t *testing.T) {
	tempDir := t.TempDir()
	storage := NewLocalStorage(tempDir)

	petID := types.PetID("nonexistent_pet")

	err := storage.Delete(petID)
	if err == nil {
		t.Error("Delete should fail for nonexistent pet")
	}
}

func TestEmptyEncryptionKey(t *testing.T) {
	storage := NewLocalStorage(t.TempDir())

	err := storage.SetEncryptionKey("")
	if err == nil {
		t.Error("SetEncryptionKey should fail with empty key")
	}
}

func TestChecksumVerification(t *testing.T) {
	tempDir := t.TempDir()
	storage := NewLocalStorage(tempDir)

	petID := types.PetID("checksum_test")
	testData := &TestPetData{
		ID:   "checksum_test",
		Name: "Checksum Pet",
		Age:  4,
	}

	// Save
	err := storage.Save(petID, testData)
	if err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// Corrupt the file by modifying it
	filename := storage.getFilename(petID)
	fileData, err := os.ReadFile(filename)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}

	// Modify the data portion (this will break the checksum)
	// Find and replace "Checksum Pet" with "Corrupted!!"
	corruptedData := []byte{}
	for _, b := range fileData {
		corruptedData = append(corruptedData, b)
	}
	// Simple corruption - flip some bits
	if len(corruptedData) > 100 {
		corruptedData[100] ^= 0xFF
	}

	err = os.WriteFile(filename, corruptedData, 0644)
	if err != nil {
		t.Fatalf("Failed to write corrupted file: %v", err)
	}

	// Try to load - should fail checksum
	var loadedData TestPetData
	err = storage.Load(petID, &loadedData)
	if err == nil {
		t.Error("Load should fail with corrupted data")
	}
}

func TestGetFilename(t *testing.T) {
	storage := NewLocalStorage("/test/path")
	petID := types.PetID("test_pet")

	filename := storage.getFilename(petID)
	expected := filepath.Join("/test/path", "test_pet.json")

	if filename != expected {
		t.Errorf("getFilename() = %s, want %s", filename, expected)
	}
}
