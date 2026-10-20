package data

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

// SaveData tests

func TestNewSaveData(t *testing.T) {
	sd := NewSaveData("user123", "TestUser")

	if sd.UserID != "user123" {
		t.Errorf("Expected UserID 'user123', got '%s'", sd.UserID)
	}

	if sd.Username != "TestUser" {
		t.Errorf("Expected Username 'TestUser', got '%s'", sd.Username)
	}

	if sd.SaveID == "" {
		t.Error("SaveID should not be empty")
	}

	if sd.Version != "1.0.0" {
		t.Errorf("Expected Version '1.0.0', got '%s'", sd.Version)
	}
}

func TestDefaultUserPreferences(t *testing.T) {
	prefs := DefaultUserPreferences()

	if !prefs.NotificationsEnabled {
		t.Error("Notifications should be enabled by default")
	}

	if prefs.Language != "en" {
		t.Errorf("Expected language 'en', got '%s'", prefs.Language)
	}

	if prefs.AutoSaveInterval != 300 {
		t.Errorf("Expected auto save interval 300, got %d", prefs.AutoSaveInterval)
	}
}

func TestSetGetPetData(t *testing.T) {
	sd := NewSaveData("user123", "TestUser")

	petData := map[string]interface{}{
		"name":   "Fluffy",
		"health": 100,
	}

	err := sd.SetPetData(petData)
	if err != nil {
		t.Errorf("Failed to set pet data: %v", err)
	}

	var retrieved map[string]interface{}
	err = sd.GetPetData(&retrieved)
	if err != nil {
		t.Errorf("Failed to get pet data: %v", err)
	}

	if retrieved["name"] != "Fluffy" {
		t.Errorf("Expected name 'Fluffy', got '%v'", retrieved["name"])
	}
}

func TestAddHistoryEntry(t *testing.T) {
	sd := NewSaveData("user123", "TestUser")

	sd.AddHistoryEntry("feed", "Fed the pet", "success", 1.5)

	if len(sd.CareHistory) != 1 {
		t.Errorf("Expected 1 history entry, got %d", len(sd.CareHistory))
	}

	entry := sd.CareHistory[0]
	if entry.ActionType != "feed" {
		t.Errorf("Expected action type 'feed', got '%s'", entry.ActionType)
	}
}

func TestHistoryEntryLimit(t *testing.T) {
	sd := NewSaveData("user123", "TestUser")

	// Add more than 1000 entries
	for i := 0; i < 1100; i++ {
		sd.AddHistoryEntry("test", "Test action", "success", float64(i))
	}

	if len(sd.CareHistory) != 1000 {
		t.Errorf("Expected history to be limited to 1000 entries, got %d", len(sd.CareHistory))
	}
}

func TestAddAchievement(t *testing.T) {
	sd := NewSaveData("user123", "TestUser")

	added := sd.AddAchievement("first_feed", "First Feeding", "Fed your pet for the first time", "common", 10)
	if !added {
		t.Error("Achievement should be added")
	}

	// Try adding same achievement again
	added = sd.AddAchievement("first_feed", "First Feeding", "Fed your pet for the first time", "common", 10)
	if added {
		t.Error("Duplicate achievement should not be added")
	}

	if len(sd.Achievements) != 1 {
		t.Errorf("Expected 1 achievement, got %d", len(sd.Achievements))
	}
}

func TestHasAchievement(t *testing.T) {
	sd := NewSaveData("user123", "TestUser")

	sd.AddAchievement("test_ach", "Test", "Test achievement", "common", 5)

	if !sd.HasAchievement("test_ach") {
		t.Error("Should have test_ach achievement")
	}

	if sd.HasAchievement("nonexistent") {
		t.Error("Should not have nonexistent achievement")
	}
}

func TestUnlockContent(t *testing.T) {
	sd := NewSaveData("user123", "TestUser")

	unlocked := sd.UnlockContent("location", "beach")
	if !unlocked {
		t.Error("Beach location should be unlocked")
	}

	// Try unlocking again
	unlocked = sd.UnlockContent("location", "beach")
	if unlocked {
		t.Error("Already unlocked content should return false")
	}
}

func TestIsContentUnlocked(t *testing.T) {
	sd := NewSaveData("user123", "TestUser")

	// Default unlocks
	if !sd.IsContentUnlocked("location", "home") {
		t.Error("Home should be unlocked by default")
	}

	if sd.IsContentUnlocked("location", "beach") {
		t.Error("Beach should not be unlocked by default")
	}

	sd.UnlockContent("location", "beach")
	if !sd.IsContentUnlocked("location", "beach") {
		t.Error("Beach should be unlocked after UnlockContent")
	}
}

func TestUpdateStatistics(t *testing.T) {
	sd := NewSaveData("user123", "TestUser")

	sd.UpdateStatistics(1.5, 10)
	sd.UpdateStatistics(2.0, 5)

	if sd.Statistics.TotalPlayTime != 3.5 {
		t.Errorf("Expected total play time 3.5, got %f", sd.Statistics.TotalPlayTime)
	}

	if sd.Statistics.TotalInteractions != 15 {
		t.Errorf("Expected total interactions 15, got %d", sd.Statistics.TotalInteractions)
	}
}

func TestChecksum(t *testing.T) {
	sd := NewSaveData("user123", "TestUser")
	sd.SetPetData(map[string]string{"name": "Fluffy"})

	checksum1 := sd.CalculateChecksum()
	if checksum1 == "" {
		t.Error("Checksum should not be empty")
	}

	// Same data should produce same checksum
	checksum2 := sd.CalculateChecksum()
	if checksum1 != checksum2 {
		t.Error("Same data should produce same checksum")
	}

	// Different data should produce different checksum
	sd.SaveCount = 100
	checksum3 := sd.CalculateChecksum()
	if checksum1 == checksum3 {
		t.Error("Different data should produce different checksum")
	}
}

func TestValidateChecksum(t *testing.T) {
	sd := NewSaveData("user123", "TestUser")
	sd.Checksum = sd.CalculateChecksum()

	if !sd.ValidateChecksum() {
		t.Error("Checksum should be valid")
	}

	// Modify data
	sd.SaveCount = 999
	if sd.ValidateChecksum() {
		t.Error("Checksum should be invalid after modification")
	}
}

func TestPrepareSave(t *testing.T) {
	sd := NewSaveData("user123", "TestUser")
	initialCount := sd.SaveCount
	initialVersion := sd.SyncStatus.LocalVersion

	sd.PrepareSave()

	if sd.SaveCount != initialCount+1 {
		t.Error("SaveCount should be incremented")
	}

	if sd.SyncStatus.LocalVersion != initialVersion+1 {
		t.Error("LocalVersion should be incremented")
	}

	if !sd.SyncStatus.PendingChanges {
		t.Error("PendingChanges should be true")
	}

	if sd.Checksum == "" {
		t.Error("Checksum should be set")
	}
}

func TestToFromJSON(t *testing.T) {
	sd := NewSaveData("user123", "TestUser")
	sd.AddAchievement("test", "Test", "Test", "common", 10)
	sd.SetPetData(map[string]string{"name": "Fluffy"})

	data, err := sd.ToJSON()
	if err != nil {
		t.Fatalf("ToJSON failed: %v", err)
	}

	loaded, err := FromJSON(data)
	if err != nil {
		t.Fatalf("FromJSON failed: %v", err)
	}

	if loaded.UserID != sd.UserID {
		t.Errorf("UserID mismatch: expected %s, got %s", sd.UserID, loaded.UserID)
	}

	if len(loaded.Achievements) != len(sd.Achievements) {
		t.Error("Achievements count mismatch")
	}
}

func TestEncryption(t *testing.T) {
	sd := NewSaveData("user123", "TestUser")
	sd.SetPetData(map[string]string{"name": "Fluffy"})

	key := []byte("test-encryption-key-12345")

	encrypted, err := sd.ToEncryptedJSON(key)
	if err != nil {
		t.Fatalf("Encryption failed: %v", err)
	}

	decrypted, err := FromEncryptedJSON(encrypted, key)
	if err != nil {
		t.Fatalf("Decryption failed: %v", err)
	}

	if decrypted.UserID != sd.UserID {
		t.Error("Decrypted data mismatch")
	}
}

func TestEncryptionWrongKey(t *testing.T) {
	sd := NewSaveData("user123", "TestUser")
	key1 := []byte("correct-key")
	key2 := []byte("wrong-key")

	encrypted, _ := sd.ToEncryptedJSON(key1)

	_, err := FromEncryptedJSON(encrypted, key2)
	if err == nil {
		t.Error("Decryption with wrong key should fail")
	}
}

// DataManager tests

func TestNewDataManager(t *testing.T) {
	// Use temp directory for testing
	tmpDir := t.TempDir()
	config := DataManagerConfig{
		SavePath:   filepath.Join(tmpDir, "saves"),
		BackupPath: filepath.Join(tmpDir, "backups"),
		CachePath:  filepath.Join(tmpDir, "cache"),
		MaxBackups: 3,
		AutoBackup: true,
	}

	dm, err := NewDataManagerWithConfig(config)
	if err != nil {
		t.Fatalf("Failed to create DataManager: %v", err)
	}

	// Check directories were created
	if _, err := os.Stat(config.SavePath); os.IsNotExist(err) {
		t.Error("Save path should be created")
	}

	if _, err := os.Stat(config.BackupPath); os.IsNotExist(err) {
		t.Error("Backup path should be created")
	}

	if dm.MaxBackups != 3 {
		t.Errorf("Expected MaxBackups 3, got %d", dm.MaxBackups)
	}
}

func TestSaveAndLoad(t *testing.T) {
	tmpDir := t.TempDir()
	config := DataManagerConfig{
		SavePath:   filepath.Join(tmpDir, "saves"),
		BackupPath: filepath.Join(tmpDir, "backups"),
		CachePath:  filepath.Join(tmpDir, "cache"),
		AutoBackup: false,
	}

	dm, _ := NewDataManagerWithConfig(config)

	sd := NewSaveData("user123", "TestUser")
	sd.SetPetData(map[string]string{"name": "Fluffy"})

	// Save
	err := dm.Save(sd, "test_save")
	if err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// Load
	loaded, err := dm.Load("test_save")
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if loaded.UserID != sd.UserID {
		t.Error("Loaded data mismatch")
	}
}

func TestExists(t *testing.T) {
	tmpDir := t.TempDir()
	config := DataManagerConfig{
		SavePath:   filepath.Join(tmpDir, "saves"),
		BackupPath: filepath.Join(tmpDir, "backups"),
		CachePath:  filepath.Join(tmpDir, "cache"),
		AutoBackup: false,
	}

	dm, _ := NewDataManagerWithConfig(config)

	if dm.Exists("nonexistent") {
		t.Error("Nonexistent file should not exist")
	}

	sd := NewSaveData("user123", "TestUser")
	dm.Save(sd, "test_save")

	if !dm.Exists("test_save") {
		t.Error("Saved file should exist")
	}
}

func TestDelete(t *testing.T) {
	tmpDir := t.TempDir()
	config := DataManagerConfig{
		SavePath:   filepath.Join(tmpDir, "saves"),
		BackupPath: filepath.Join(tmpDir, "backups"),
		CachePath:  filepath.Join(tmpDir, "cache"),
		AutoBackup: false,
	}

	dm, _ := NewDataManagerWithConfig(config)

	sd := NewSaveData("user123", "TestUser")
	dm.Save(sd, "test_save")

	err := dm.Delete("test_save")
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	if dm.Exists("test_save") {
		t.Error("Deleted file should not exist")
	}
}

func TestListSaves(t *testing.T) {
	tmpDir := t.TempDir()
	config := DataManagerConfig{
		SavePath:   filepath.Join(tmpDir, "saves"),
		BackupPath: filepath.Join(tmpDir, "backups"),
		CachePath:  filepath.Join(tmpDir, "cache"),
		AutoBackup: false,
	}

	dm, _ := NewDataManagerWithConfig(config)

	// Create some saves
	sd1 := NewSaveData("user1", "User1")
	sd2 := NewSaveData("user2", "User2")

	dm.Save(sd1, "save1")
	time.Sleep(10 * time.Millisecond)
	dm.Save(sd2, "save2")

	saves, err := dm.ListSaves()
	if err != nil {
		t.Fatalf("ListSaves failed: %v", err)
	}

	if len(saves) != 2 {
		t.Errorf("Expected 2 saves, got %d", len(saves))
	}

	// Should be sorted by modification time (newest first)
	if saves[0].Filename != "save2.json" {
		t.Error("Saves should be sorted by modification time")
	}
}

func TestBackup(t *testing.T) {
	tmpDir := t.TempDir()
	config := DataManagerConfig{
		SavePath:   filepath.Join(tmpDir, "saves"),
		BackupPath: filepath.Join(tmpDir, "backups"),
		CachePath:  filepath.Join(tmpDir, "cache"),
		AutoBackup: false,
		MaxBackups: 3,
	}

	dm, _ := NewDataManagerWithConfig(config)

	sd := NewSaveData("user123", "TestUser")
	dm.Save(sd, "test_save")

	err := dm.CreateBackup("test_save")
	if err != nil {
		t.Fatalf("CreateBackup failed: %v", err)
	}

	backups, err := dm.ListBackups("test_save")
	if err != nil {
		t.Fatalf("ListBackups failed: %v", err)
	}

	if len(backups) != 1 {
		t.Errorf("Expected 1 backup, got %d", len(backups))
	}
}

func TestStorageStats(t *testing.T) {
	tmpDir := t.TempDir()
	config := DataManagerConfig{
		SavePath:   filepath.Join(tmpDir, "saves"),
		BackupPath: filepath.Join(tmpDir, "backups"),
		CachePath:  filepath.Join(tmpDir, "cache"),
		AutoBackup: false,
	}

	dm, _ := NewDataManagerWithConfig(config)

	sd := NewSaveData("user123", "TestUser")
	dm.Save(sd, "test_save")

	stats := dm.GetStorageStats()

	if stats.SavesCount != 1 {
		t.Errorf("Expected 1 save, got %d", stats.SavesCount)
	}

	if stats.SavesSize <= 0 {
		t.Error("SavesSize should be positive")
	}
}

// CloudSync tests

func TestNewCloudSyncService(t *testing.T) {
	provider := NewMockCloudProvider()
	cs := NewCloudSyncService(provider, "user123")

	if cs.UserID != "user123" {
		t.Errorf("Expected UserID 'user123', got '%s'", cs.UserID)
	}

	if cs.Enabled {
		t.Error("Cloud sync should be disabled by default")
	}
}

func TestCloudSyncEnable(t *testing.T) {
	provider := NewMockCloudProvider()
	cs := NewCloudSyncService(provider, "user123")

	err := cs.Enable()
	if err != nil {
		t.Errorf("Enable failed: %v", err)
	}

	if !cs.Enabled {
		t.Error("Cloud sync should be enabled")
	}
}

func TestCloudSyncDisabled(t *testing.T) {
	provider := NewMockCloudProvider()
	provider.Available = false

	cs := NewCloudSyncService(provider, "user123")

	err := cs.Enable()
	if err == nil {
		t.Error("Enable should fail when provider is unavailable")
	}
}

func TestCloudSyncUpload(t *testing.T) {
	provider := NewMockCloudProvider()
	cs := NewCloudSyncService(provider, "user123")
	cs.Enable()

	sd := NewSaveData("user123", "TestUser")
	sd.SetPetData(map[string]string{"name": "Fluffy"})

	err := cs.Sync(sd)
	if err != nil {
		t.Errorf("Sync failed: %v", err)
	}

	if cs.Status != SyncStatusSuccess {
		t.Errorf("Expected status Success, got %s", cs.Status.String())
	}

	// Check data was uploaded
	if _, exists := provider.Storage["user123"]; !exists {
		t.Error("Data should be uploaded to cloud")
	}
}

func TestCloudSyncStatusString(t *testing.T) {
	tests := []struct {
		status   CloudSyncStatus
		expected string
	}{
		{SyncStatusIdle, "Idle"},
		{SyncStatusSyncing, "Syncing"},
		{SyncStatusSuccess, "Success"},
		{SyncStatusError, "Error"},
		{SyncStatusConflict, "Conflict"},
	}

	for _, test := range tests {
		if test.status.String() != test.expected {
			t.Errorf("Expected %s, got %s", test.expected, test.status.String())
		}
	}
}

func TestMockCloudProvider(t *testing.T) {
	provider := NewMockCloudProvider()

	// Test upload
	data := []byte(`{"test": "data"}`)
	err := provider.Upload("user123", data)
	if err != nil {
		t.Errorf("Upload failed: %v", err)
	}

	// Test download
	downloaded, err := provider.Download("user123")
	if err != nil {
		t.Errorf("Download failed: %v", err)
	}

	if string(downloaded) != string(data) {
		t.Error("Downloaded data mismatch")
	}

	// Test delete
	err = provider.Delete("user123")
	if err != nil {
		t.Errorf("Delete failed: %v", err)
	}

	_, err = provider.Download("user123")
	if err == nil {
		t.Error("Download should fail after delete")
	}
}

func TestMockCloudProviderUnavailable(t *testing.T) {
	provider := NewMockCloudProvider()
	provider.Available = false

	err := provider.Upload("user123", []byte("data"))
	if err == nil {
		t.Error("Upload should fail when unavailable")
	}

	_, err = provider.Download("user123")
	if err == nil {
		t.Error("Download should fail when unavailable")
	}
}

// Helper function tests

func TestUniqueStrings(t *testing.T) {
	input := []string{"a", "b", "a", "c", "b", "d"}
	result := uniqueStrings(input)

	if len(result) != 4 {
		t.Errorf("Expected 4 unique strings, got %d", len(result))
	}
}

func TestMaxFloat(t *testing.T) {
	if maxFloat(1.5, 2.5) != 2.5 {
		t.Error("maxFloat(1.5, 2.5) should be 2.5")
	}

	if maxFloat(3.0, 1.0) != 3.0 {
		t.Error("maxFloat(3.0, 1.0) should be 3.0")
	}
}

func TestMaxInt(t *testing.T) {
	if maxInt(5, 10) != 10 {
		t.Error("maxInt(5, 10) should be 10")
	}

	if maxInt(15, 3) != 15 {
		t.Error("maxInt(15, 3) should be 15")
	}
}

func newTestDataManager(t *testing.T) *DataManager {
	t.Helper()
	tmpDir := t.TempDir()
	dm, err := NewDataManagerWithConfig(DataManagerConfig{
		SavePath:   filepath.Join(tmpDir, "saves"),
		BackupPath: filepath.Join(tmpDir, "backups"),
		CachePath:  filepath.Join(tmpDir, "cache"),
		MaxBackups: 3,
		AutoBackup: true,
	})
	if err != nil {
		t.Fatalf("Failed to create DataManager: %v", err)
	}
	return dm
}

func TestLoadLatestNoSave(t *testing.T) {
	dm := newTestDataManager(t)

	_, _, err := dm.LoadLatest("pet")
	if !errors.Is(err, os.ErrNotExist) {
		t.Errorf("Expected os.ErrNotExist when nothing was saved, got %v", err)
	}
}

func TestLoadLatestReturnsMostRecentSave(t *testing.T) {
	dm := newTestDataManager(t)

	sd := NewSaveData("user", "User")
	sd.Username = "first"
	if err := dm.Save(sd, "pet"); err != nil {
		t.Fatal(err)
	}
	sd.Username = "second"
	if err := dm.Save(sd, "pet"); err != nil {
		t.Fatal(err)
	}

	loaded, path, err := dm.LoadLatest("pet")
	if err != nil {
		t.Fatalf("LoadLatest failed: %v", err)
	}
	if loaded.Username != "second" {
		t.Errorf("Expected most recent save, got %q", loaded.Username)
	}
	if path != dm.getSavePath("pet") {
		t.Errorf("Expected load from main save file, got %s", path)
	}
}

func TestLoadLatestFallsBackToBackup(t *testing.T) {
	dm := newTestDataManager(t)

	sd := NewSaveData("user", "User")
	sd.SetPetData(map[string]string{"name": "first"})
	if err := dm.Save(sd, "pet"); err != nil {
		t.Fatal(err)
	}
	sd.SetPetData(map[string]string{"name": "second"})
	if err := dm.Save(sd, "pet"); err != nil { // Backs up the first save
		t.Fatal(err)
	}

	// Corrupt the main save
	if err := os.WriteFile(dm.getSavePath("pet"), []byte("{truncated"), 0644); err != nil {
		t.Fatal(err)
	}

	loaded, path, err := dm.LoadLatest("pet")
	if err != nil {
		t.Fatalf("LoadLatest should fall back to backup: %v", err)
	}
	if filepath.Dir(path) != dm.BackupPath {
		t.Errorf("Expected load from backup directory, got %s", path)
	}

	var pet map[string]string
	if err := loaded.GetPetData(&pet); err != nil || pet["name"] != "first" {
		t.Errorf("Expected backup pet data, got %v (err %v)", pet, err)
	}
	if dm.GetLastError() == nil {
		t.Error("Skipping a corrupt save should be reported in LastError")
	}
}

func TestLoadLatestSkipsChecksumMismatch(t *testing.T) {
	dm := newTestDataManager(t)

	sd := NewSaveData("user", "User")
	sd.SetPetData(map[string]string{"name": "only"})
	if err := dm.Save(sd, "pet"); err != nil {
		t.Fatal(err)
	}

	// Alter pet data without updating the checksum
	sd.SetPetData(map[string]string{"name": "tampered"})
	data, _ := sd.ToJSON()
	os.WriteFile(dm.getSavePath("pet"), data, 0644)

	if _, _, err := dm.LoadLatest("pet"); err == nil || errors.Is(err, os.ErrNotExist) {
		t.Errorf("Expected failure for save with bad checksum and no backups, got %v", err)
	}
}

func TestSaveLeavesNoTemporaryFiles(t *testing.T) {
	dm := newTestDataManager(t)

	sd := NewSaveData("user", "User")
	for i := 0; i < 3; i++ {
		if err := dm.Save(sd, "pet"); err != nil {
			t.Fatal(err)
		}
	}

	entries, err := os.ReadDir(dm.SavePath)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Name() != "pet.json" {
		names := []string{}
		for _, e := range entries {
			names = append(names, e.Name())
		}
		t.Errorf("Expected only pet.json in save directory, got %v", names)
	}
}

func TestBackupsAreSeparatePerSave(t *testing.T) {
	dm := newTestDataManager(t)

	// "pet" is a prefix of "pet_2"; their backups must not be mixed up
	for _, name := range []string{"pet", "pet_2"} {
		sd := NewSaveData("user", name)
		sd.SetPetData(map[string]string{"name": name})
		if err := dm.Save(sd, name); err != nil {
			t.Fatal(err)
		}
		if err := dm.Save(sd, name); err != nil { // Creates a backup
			t.Fatal(err)
		}
	}

	for _, name := range []string{"pet", "pet_2"} {
		backups, err := dm.ListBackups(name)
		if err != nil {
			t.Fatal(err)
		}
		if len(backups) != 1 {
			t.Errorf("Expected 1 backup for %q, got %d", name, len(backups))
		}
	}

	// With "pet" corrupt, its fallback must be its own backup
	os.WriteFile(dm.getSavePath("pet"), []byte("corrupt"), 0644)
	loaded, _, err := dm.LoadLatest("pet")
	if err != nil {
		t.Fatalf("LoadLatest failed: %v", err)
	}
	var pet map[string]string
	loaded.GetPetData(&pet)
	if pet["name"] != "pet" {
		t.Errorf("Fell back to another save's backup: %v", pet)
	}
}

func TestIsBackupOf(t *testing.T) {
	tests := []struct {
		name, prefix string
		want         bool
	}{
		{"pet_20261018_223851.bak", "pet", true},
		{"pet_2_20261018_223851.bak", "pet", false},
		{"pet_2_20261018_223851.bak", "pet_2", true},
		{"pet_20261018_223851.json", "pet", false},
		{"petx_20261018_223851.bak", "pet", false},
	}
	for _, tt := range tests {
		if got := isBackupOf(tt.name, tt.prefix); got != tt.want {
			t.Errorf("isBackupOf(%q, %q) = %v, want %v", tt.name, tt.prefix, got, tt.want)
		}
	}
}

func TestDeleteWithBackups(t *testing.T) {
	dm := newTestDataManager(t)

	for _, name := range []string{"pet", "pet_2"} {
		sd := NewSaveData("user", name)
		dm.Save(sd, name)
		dm.Save(sd, name) // Creates a backup
	}

	if err := dm.DeleteWithBackups("pet"); err != nil {
		t.Fatalf("DeleteWithBackups failed: %v", err)
	}

	if dm.Exists("pet") {
		t.Error("Save file should be deleted")
	}
	if backups, _ := dm.ListBackups("pet"); len(backups) != 0 {
		t.Errorf("Backups should be deleted, %d remain", len(backups))
	}
	if _, _, err := dm.LoadLatest("pet"); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("Deleted save should not be loadable, got %v", err)
	}

	// The other save is untouched
	if !dm.Exists("pet_2") {
		t.Error("Other save should not be deleted")
	}
	if backups, _ := dm.ListBackups("pet_2"); len(backups) != 1 {
		t.Errorf("Other save's backups should remain, got %d", len(backups))
	}

	if err := dm.DeleteWithBackups("pet"); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("Deleting again should report os.ErrNotExist, got %v", err)
	}
}

func TestSavesArePrivate(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows doesn't use Unix permissions")
	}
	tmpDir := t.TempDir()
	saves := filepath.Join(tmpDir, "saves")
	os.MkdirAll(saves, 0o755) // Made by an older version
	config := DataManagerConfig{SavePath: saves, BackupPath: filepath.Join(tmpDir, "backups"),
		CachePath: filepath.Join(tmpDir, "cache"), MaxBackups: 3, AutoBackup: true}
	dm, err := NewDataManagerWithConfig(config)
	if err != nil {
		t.Fatal(err)
	}
	save := NewSaveData("user123", "pet")
	if err := dm.Save(save, "pet"); err != nil {
		t.Fatal(err)
	}
	if err := dm.Save(save, "pet"); err != nil { // Makes a backup
		t.Fatal(err)
	}
	for _, dir := range []string{config.SavePath, config.BackupPath} {
		info, err := os.Stat(dir)
		if err != nil || info.Mode().Perm() != dirPerm {
			t.Errorf("%s: %v %v", dir, info.Mode(), err)
		}
		entries, _ := os.ReadDir(dir)
		if len(entries) == 0 {
			t.Errorf("%s is empty", dir)
		}
		for _, e := range entries {
			info, _ := e.Info()
			if info.Mode().Perm() != filePerm {
				t.Errorf("%s/%s is %v", dir, e.Name(), info.Mode().Perm())
			}
		}
	}
}
