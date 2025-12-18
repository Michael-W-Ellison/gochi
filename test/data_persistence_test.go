package test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/Michael-W-Ellison/gochi/internal/core"
	"github.com/Michael-W-Ellison/gochi/internal/data"
	"github.com/Michael-W-Ellison/gochi/pkg/types"
)

// TestDataManagerCreation tests data manager initialization
func TestDataManagerCreation(t *testing.T) {
	manager, err := data.NewDataManager()
	if err != nil {
		t.Fatalf("Failed to create DataManager: %v", err)
	}

	if manager == nil {
		t.Fatal("DataManager should not be nil")
	}
}

// TestDataManagerWithConfig tests custom configuration
func TestDataManagerWithConfig(t *testing.T) {
	tempDir := t.TempDir()

	config := data.DataManagerConfig{
		SavePath:   filepath.Join(tempDir, "saves"),
		BackupPath: filepath.Join(tempDir, "backups"),
		CachePath:  filepath.Join(tempDir, "cache"),
		MaxBackups: 3,
		AutoBackup: true,
	}

	manager, err := data.NewDataManagerWithConfig(config)
	if err != nil {
		t.Fatalf("Failed to create DataManager with config: %v", err)
	}

	if manager.SavePath != config.SavePath {
		t.Errorf("SavePath mismatch: expected %s, got %s",
			config.SavePath, manager.SavePath)
	}
}

// TestSaveDataCreation tests SaveData creation
func TestSaveDataCreation(t *testing.T) {
	saveData := data.NewSaveData("user_123", "TestUser")

	if saveData.UserID != "user_123" {
		t.Errorf("UserID mismatch: expected 'user_123', got '%s'", saveData.UserID)
	}

	if saveData.Username != "TestUser" {
		t.Errorf("Username mismatch: expected 'TestUser', got '%s'", saveData.Username)
	}

	if saveData.Version == "" {
		t.Error("Version should be set")
	}

	if saveData.SaveID == "" {
		t.Error("SaveID should be generated")
	}
}

// TestSaveDataWithPet tests saving pet data
func TestSaveDataWithPet(t *testing.T) {
	pet := core.NewDigitalPet("TestPet", "user_savedata")

	saveData := data.NewSaveData("user_123", "TestUser")
	err := saveData.SetPetData(pet)
	if err != nil {
		t.Fatalf("Failed to set pet data: %v", err)
	}

	// Retrieve pet data
	var loadedPet core.DigitalPet
	err = saveData.GetPetData(&loadedPet)
	if err != nil {
		t.Fatalf("Failed to get pet data: %v", err)
	}

	if loadedPet.Name != pet.Name {
		t.Errorf("Pet name mismatch: expected '%s', got '%s'", pet.Name, loadedPet.Name)
	}
}

// TestSaveAndLoad tests saving and loading save data
func TestSaveAndLoad(t *testing.T) {
	tempDir := t.TempDir()

	config := data.DataManagerConfig{
		SavePath:   filepath.Join(tempDir, "saves"),
		BackupPath: filepath.Join(tempDir, "backups"),
		CachePath:  filepath.Join(tempDir, "cache"),
		MaxBackups: 3,
	}

	manager, err := data.NewDataManagerWithConfig(config)
	if err != nil {
		t.Fatalf("Failed to create DataManager: %v", err)
	}

	// Create save data with a pet
	pet := core.NewDigitalPet("LoadTest", "user_load")
	pet.TotalInteractions = 42

	saveData := data.NewSaveData("user_load", "LoadUser")
	saveData.SetPetData(pet)

	// Save
	filename := "test_save.json"
	err = manager.Save(saveData, filename)
	if err != nil {
		t.Fatalf("Failed to save: %v", err)
	}

	// Verify file exists
	savePath := filepath.Join(config.SavePath, filename)
	if _, err := os.Stat(savePath); os.IsNotExist(err) {
		t.Error("Save file should exist")
	}

	// Load
	loadedData, err := manager.Load(filename)
	if err != nil {
		t.Fatalf("Failed to load: %v", err)
	}

	if loadedData.UserID != saveData.UserID {
		t.Errorf("UserID mismatch: expected '%s', got '%s'",
			saveData.UserID, loadedData.UserID)
	}
}

// TestPetSerialization tests direct pet serialization
func TestPetSerialization(t *testing.T) {
	// Create and modify a pet
	originalPet := core.NewDigitalPet("SerializeTest", "user_serialize")
	originalPet.ProcessUserInteraction(types.InteractionFeeding, 1.0)
	originalPet.TotalInteractions = 25
	originalPet.Location = "park"

	// Serialize
	jsonData, err := originalPet.Save()
	if err != nil {
		t.Fatalf("Failed to serialize pet: %v", err)
	}

	if len(jsonData) == 0 {
		t.Error("Serialized data should not be empty")
	}

	// Deserialize
	loadedPet, err := core.Load(jsonData)
	if err != nil {
		t.Fatalf("Failed to deserialize pet: %v", err)
	}

	// Verify data matches
	if loadedPet.Name != originalPet.Name {
		t.Errorf("Name mismatch: expected '%s', got '%s'",
			originalPet.Name, loadedPet.Name)
	}

	if loadedPet.ID != originalPet.ID {
		t.Errorf("ID mismatch: expected '%s', got '%s'",
			originalPet.ID, loadedPet.ID)
	}

	if loadedPet.Location != originalPet.Location {
		t.Errorf("Location mismatch: expected '%s', got '%s'",
			originalPet.Location, loadedPet.Location)
	}

	if loadedPet.TotalInteractions != originalPet.TotalInteractions {
		t.Errorf("TotalInteractions mismatch: expected %d, got %d",
			originalPet.TotalInteractions, loadedPet.TotalInteractions)
	}
}

// TestSaveDataFormat tests the save data JSON format
func TestSaveDataFormat(t *testing.T) {
	pet := core.NewDigitalPet("FormatTest", "user_format_test")
	pet.TotalInteractions = 100
	pet.Location = "beach"

	// Serialize
	jsonData, err := pet.Save()
	if err != nil {
		t.Fatalf("Failed to serialize pet: %v", err)
	}

	// Verify it's valid JSON
	var rawData map[string]interface{}
	err = json.Unmarshal(jsonData, &rawData)
	if err != nil {
		t.Fatalf("Save data is not valid JSON: %v", err)
	}

	// Check required fields exist
	requiredFields := []string{"id", "name", "owner", "biology", "personality",
		"memory", "emotions", "relationships", "current_behavior", "location",
		"created_at", "last_update_at", "total_interactions", "total_play_time"}

	for _, field := range requiredFields {
		if _, exists := rawData[field]; !exists {
			t.Errorf("Required field '%s' missing from save data", field)
		}
	}
}

// TestSaveDataIntegrity tests data integrity after save/load cycles
func TestSaveDataIntegrity(t *testing.T) {
	tempDir := t.TempDir()

	config := data.DataManagerConfig{
		SavePath:   filepath.Join(tempDir, "saves"),
		BackupPath: filepath.Join(tempDir, "backups"),
		CachePath:  filepath.Join(tempDir, "cache"),
	}

	manager, err := data.NewDataManagerWithConfig(config)
	if err != nil {
		t.Fatalf("Failed to create DataManager: %v", err)
	}

	// Create a complex pet state
	pet := core.NewDigitalPet("IntegrityTest", "user_integrity")
	pet.ProcessUserInteraction(types.InteractionFeeding, 1.0)
	pet.ProcessUserInteraction(types.InteractionPetting, 0.8)
	pet.ProcessUserInteraction(types.InteractionPlaying, 0.7)
	pet.Update(60.0)

	// Record original values
	originalNutrition := pet.Biology.Vitals.Nutrition
	originalInteractions := pet.TotalInteractions

	// Create save data
	saveData := data.NewSaveData("user_integrity", "IntegrityUser")
	saveData.SetPetData(pet)

	filename := "integrity_test.json"

	// Save and load multiple times
	for i := 0; i < 5; i++ {
		err = manager.Save(saveData, filename)
		if err != nil {
			t.Fatalf("Save cycle %d failed: %v", i, err)
		}

		saveData, err = manager.Load(filename)
		if err != nil {
			t.Fatalf("Load cycle %d failed: %v", i, err)
		}
	}

	// Retrieve pet and verify
	var loadedPet core.DigitalPet
	err = saveData.GetPetData(&loadedPet)
	if err != nil {
		t.Fatalf("Failed to get pet data: %v", err)
	}

	if loadedPet.Biology.Vitals.Nutrition != originalNutrition {
		t.Errorf("Nutrition changed after save/load: %.4f -> %.4f",
			originalNutrition, loadedPet.Biology.Vitals.Nutrition)
	}

	if loadedPet.TotalInteractions != originalInteractions {
		t.Errorf("TotalInteractions changed after save/load: %d -> %d",
			originalInteractions, loadedPet.TotalInteractions)
	}
}

// TestListSaves tests listing save files
func TestListSaves(t *testing.T) {
	tempDir := t.TempDir()

	config := data.DataManagerConfig{
		SavePath:   filepath.Join(tempDir, "saves"),
		BackupPath: filepath.Join(tempDir, "backups"),
		CachePath:  filepath.Join(tempDir, "cache"),
	}

	manager, err := data.NewDataManagerWithConfig(config)
	if err != nil {
		t.Fatalf("Failed to create DataManager: %v", err)
	}

	// Create multiple save files
	for i := 0; i < 3; i++ {
		saveData := data.NewSaveData("user_list", "ListUser")
		pet := core.NewDigitalPet("ListPet", "user_list")
		saveData.SetPetData(pet)

		filename := "save_%d.json"
		err = manager.Save(saveData, filename)
		if err != nil {
			t.Logf("Save %d: %v", i, err)
		}
	}

	// List saves
	saves, err := manager.ListSaves()
	if err != nil {
		t.Fatalf("Failed to list saves: %v", err)
	}

	t.Logf("Found %d saves", len(saves))
}

// TestDeleteSave tests deleting a save file
func TestDeleteSave(t *testing.T) {
	tempDir := t.TempDir()

	config := data.DataManagerConfig{
		SavePath:   filepath.Join(tempDir, "saves"),
		BackupPath: filepath.Join(tempDir, "backups"),
		CachePath:  filepath.Join(tempDir, "cache"),
	}

	manager, err := data.NewDataManagerWithConfig(config)
	if err != nil {
		t.Fatalf("Failed to create DataManager: %v", err)
	}

	// Create and save
	saveData := data.NewSaveData("user_delete", "DeleteUser")
	pet := core.NewDigitalPet("DeleteTest", "user_delete")
	saveData.SetPetData(pet)

	filename := "delete_test.json"
	err = manager.Save(saveData, filename)
	if err != nil {
		t.Fatalf("Failed to save: %v", err)
	}

	// Delete
	err = manager.Delete(filename)
	if err != nil {
		t.Fatalf("Failed to delete: %v", err)
	}

	// Verify file is gone
	savePath := filepath.Join(config.SavePath, filename)
	if _, err := os.Stat(savePath); !os.IsNotExist(err) {
		t.Error("Save file should not exist after deletion")
	}
}

// TestBackupCreation tests backup functionality
func TestBackupCreation(t *testing.T) {
	tempDir := t.TempDir()

	config := data.DataManagerConfig{
		SavePath:   filepath.Join(tempDir, "saves"),
		BackupPath: filepath.Join(tempDir, "backups"),
		CachePath:  filepath.Join(tempDir, "cache"),
		MaxBackups: 3,
		AutoBackup: true,
	}

	manager, err := data.NewDataManagerWithConfig(config)
	if err != nil {
		t.Fatalf("Failed to create DataManager: %v", err)
	}

	// Create save
	saveData := data.NewSaveData("user_backup", "BackupUser")
	pet := core.NewDigitalPet("BackupTest", "user_backup")
	saveData.SetPetData(pet)

	filename := "backup_test.json"
	err = manager.Save(saveData, filename)
	if err != nil {
		t.Fatalf("Failed to save: %v", err)
	}

	// Create backup
	err = manager.CreateBackup(filename)
	if err != nil {
		t.Fatalf("Failed to create backup: %v", err)
	}

	// List backups
	backups, err := manager.ListBackups(filename)
	if err != nil {
		t.Fatalf("Failed to list backups: %v", err)
	}

	t.Logf("Found %d backups", len(backups))
}

// TestCorruptedDataHandling tests handling of corrupted files
func TestCorruptedDataHandling(t *testing.T) {
	tempDir := t.TempDir()

	config := data.DataManagerConfig{
		SavePath:   filepath.Join(tempDir, "saves"),
		BackupPath: filepath.Join(tempDir, "backups"),
		CachePath:  filepath.Join(tempDir, "cache"),
	}

	manager, err := data.NewDataManagerWithConfig(config)
	if err != nil {
		t.Fatalf("Failed to create DataManager: %v", err)
	}

	// Create a corrupted save file
	corruptedPath := filepath.Join(config.SavePath, "corrupted.json")
	err = os.WriteFile(corruptedPath, []byte("not valid json {{{"), 0644)
	if err != nil {
		t.Fatalf("Failed to create corrupted file: %v", err)
	}

	// Attempt to load
	_, err = manager.Load("corrupted.json")
	if err == nil {
		t.Error("Loading corrupted data should fail")
	}
}

// TestEmptyDataHandling tests handling of empty files
func TestEmptyDataHandling(t *testing.T) {
	tempDir := t.TempDir()

	config := data.DataManagerConfig{
		SavePath:   filepath.Join(tempDir, "saves"),
		BackupPath: filepath.Join(tempDir, "backups"),
		CachePath:  filepath.Join(tempDir, "cache"),
	}

	manager, err := data.NewDataManagerWithConfig(config)
	if err != nil {
		t.Fatalf("Failed to create DataManager: %v", err)
	}

	// Create an empty save file
	emptyPath := filepath.Join(config.SavePath, "empty.json")
	err = os.WriteFile(emptyPath, []byte(""), 0644)
	if err != nil {
		t.Fatalf("Failed to create empty file: %v", err)
	}

	// Attempt to load
	_, err = manager.Load("empty.json")
	if err == nil {
		t.Error("Loading empty data should fail")
	}
}

// TestSaveDataMetadata tests metadata in saves
func TestSaveDataMetadata(t *testing.T) {
	saveData := data.NewSaveData("user_metadata", "MetadataUser")

	// Check metadata
	if saveData.Version == "" {
		t.Error("Version should be set")
	}

	if saveData.CreatedAt.IsZero() {
		t.Error("CreatedAt should be set")
	}

	if saveData.SaveID == "" {
		t.Error("SaveID should be generated")
	}
}

// TestSaveDataHistoryEntry tests adding history entries
func TestSaveDataHistoryEntry(t *testing.T) {
	saveData := data.NewSaveData("user_history", "HistoryUser")

	// Add history entry
	saveData.AddHistoryEntry("feeding", "Fed the pet", "success", 1.5)

	if len(saveData.CareHistory) != 1 {
		t.Errorf("Expected 1 history entry, got %d", len(saveData.CareHistory))
	}

	entry := saveData.CareHistory[0]
	if entry.ActionType != "feeding" {
		t.Errorf("ActionType mismatch: expected 'feeding', got '%s'", entry.ActionType)
	}
}

// TestSaveDataAchievements tests achievement tracking
func TestSaveDataAchievements(t *testing.T) {
	saveData := data.NewSaveData("user_achieve", "AchieveUser")

	// Add achievement
	added := saveData.AddAchievement("first_pet", "First Pet", "Create your first pet", "common", 10)
	if !added {
		t.Error("First achievement add should return true")
	}

	if len(saveData.Achievements) != 1 {
		t.Errorf("Expected 1 achievement, got %d", len(saveData.Achievements))
	}

	// Adding same achievement again should return false
	added = saveData.AddAchievement("first_pet", "First Pet", "Create your first pet", "common", 10)
	if added {
		t.Error("Duplicate achievement add should return false")
	}

	// Check has achievement
	if !saveData.HasAchievement("first_pet") {
		t.Error("Should have 'first_pet' achievement")
	}

	if saveData.HasAchievement("nonexistent") {
		t.Error("Should not have 'nonexistent' achievement")
	}
}

// TestSaveDataUnlocks tests content unlocking
func TestSaveDataUnlocks(t *testing.T) {
	saveData := data.NewSaveData("user_unlock", "UnlockUser")

	// Default unlocks
	if !saveData.IsContentUnlocked("location", "home") {
		t.Error("'home' location should be unlocked by default")
	}

	// Unlock new content
	unlocked := saveData.UnlockContent("location", "beach")
	if !unlocked {
		t.Error("First unlock should return true")
	}

	// Unlock again should return false
	unlocked = saveData.UnlockContent("location", "beach")
	if unlocked {
		t.Error("Duplicate unlock should return false")
	}

	// Check unlocked
	if !saveData.IsContentUnlocked("location", "beach") {
		t.Error("'beach' location should be unlocked")
	}
}

// TestSaveDataChecksum tests checksum calculation
func TestSaveDataChecksum(t *testing.T) {
	saveData := data.NewSaveData("user_checksum", "ChecksumUser")

	// Calculate checksum
	checksum1 := saveData.CalculateChecksum()
	if checksum1 == "" {
		t.Error("Checksum should not be empty")
	}

	// Same data should produce same checksum
	checksum2 := saveData.CalculateChecksum()
	if checksum1 != checksum2 {
		t.Error("Same data should produce same checksum")
	}

	// Prepare save and validate
	saveData.PrepareSave()
	if !saveData.ValidateChecksum() {
		t.Error("Checksum should validate after PrepareSave")
	}
}

// TestSaveDataStatistics tests statistics tracking
func TestSaveDataStatistics(t *testing.T) {
	saveData := data.NewSaveData("user_stats", "StatsUser")

	initialPlayTime := saveData.Statistics.TotalPlayTime
	initialInteractions := saveData.Statistics.TotalInteractions

	// Update statistics
	saveData.UpdateStatistics(1.5, 10)

	if saveData.Statistics.TotalPlayTime != initialPlayTime+1.5 {
		t.Error("Play time should be updated")
	}

	if saveData.Statistics.TotalInteractions != initialInteractions+10 {
		t.Error("Interactions should be updated")
	}
}

// TestSaveDataJSON tests JSON serialization
func TestSaveDataJSON(t *testing.T) {
	saveData := data.NewSaveData("user_json", "JSONUser")
	pet := core.NewDigitalPet("JSONPet", "user_json")
	saveData.SetPetData(pet)

	// Serialize to JSON
	jsonBytes, err := saveData.ToJSON()
	if err != nil {
		t.Fatalf("Failed to serialize to JSON: %v", err)
	}

	if len(jsonBytes) == 0 {
		t.Error("JSON output should not be empty")
	}

	// Deserialize from JSON
	loaded, err := data.FromJSON(jsonBytes)
	if err != nil {
		t.Fatalf("Failed to deserialize from JSON: %v", err)
	}

	if loaded.UserID != saveData.UserID {
		t.Errorf("UserID mismatch: expected '%s', got '%s'",
			saveData.UserID, loaded.UserID)
	}
}
