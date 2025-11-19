package data

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Michael-W-Ellison/gochi/pkg/types"
)

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	if config == nil {
		t.Fatal("DefaultConfig returned nil")
	}

	if config.BasePath != "./data/pets" {
		t.Errorf("Expected BasePath './data/pets', got %s", config.BasePath)
	}

	if config.BackupPath != "./data/backups" {
		t.Errorf("Expected BackupPath './data/backups', got %s", config.BackupPath)
	}

	if config.CacheSizeMB != 100 {
		t.Errorf("Expected CacheSizeMB 100, got %d", config.CacheSizeMB)
	}

	if config.AutoSaveInterval != 5*time.Minute {
		t.Errorf("Expected AutoSaveInterval 5min, got %v", config.AutoSaveInterval)
	}

	if !config.EnableAutoSave {
		t.Error("EnableAutoSave should be true")
	}

	if !config.EnableAutoBackup {
		t.Error("EnableAutoBackup should be true")
	}
}

func TestNewDataManager(t *testing.T) {
	tempDir := t.TempDir()

	config := &Config{
		BasePath:        filepath.Join(tempDir, "pets"),
		BackupPath:      filepath.Join(tempDir, "backups"),
		CacheSizeMB:     10,
		CacheTTLMinutes: 5,
		MaxBackups:      5,
	}

	dm, err := NewDataManager(config)
	if err != nil {
		t.Fatalf("NewDataManager failed: %v", err)
	}

	if dm == nil {
		t.Fatal("NewDataManager returned nil")
	}

	if dm.localStorage == nil {
		t.Error("localStorage not initialized")
	}

	if dm.cache == nil {
		t.Error("cache not initialized")
	}

	if dm.backupManager == nil {
		t.Error("backupManager not initialized")
	}
}

func TestNewDataManagerWithEncryption(t *testing.T) {
	tempDir := t.TempDir()

	config := &Config{
		BasePath:         filepath.Join(tempDir, "pets"),
		BackupPath:       filepath.Join(tempDir, "backups"),
		CacheSizeMB:      10,
		EnableEncryption: true,
		EncryptionKey:    "test-encryption-key-32-chars!",
	}

	dm, err := NewDataManager(config)
	if err != nil {
		t.Fatalf("NewDataManager with encryption failed: %v", err)
	}

	if dm == nil {
		t.Fatal("NewDataManager returned nil")
	}
}

func TestSaveAndLoadPet(t *testing.T) {
	tempDir := t.TempDir()

	config := &Config{
		BasePath:    filepath.Join(tempDir, "pets"),
		BackupPath:  filepath.Join(tempDir, "backups"),
		CacheSizeMB: 10,
	}

	dm, err := NewDataManager(config)
	if err != nil {
		t.Fatalf("NewDataManager failed: %v", err)
	}

	// Test data
	petID := types.PetID("test-pet-1")
	petData := map[string]interface{}{
		"name": "TestPet",
		"age":  5,
	}

	// Save pet
	err = dm.SavePet(petID, petData)
	if err != nil {
		t.Fatalf("SavePet failed: %v", err)
	}

	// Check if pet exists
	if !dm.PetExists(petID) {
		t.Error("Pet should exist after saving")
	}

	// Load pet
	var loaded map[string]interface{}
	err = dm.LoadPet(petID, &loaded)
	if err != nil {
		t.Fatalf("LoadPet failed: %v", err)
	}

	if loaded["name"] != "TestPet" {
		t.Errorf("Expected name 'TestPet', got %v", loaded["name"])
	}
}

func TestDeletePet(t *testing.T) {
	tempDir := t.TempDir()

	config := &Config{
		BasePath:    filepath.Join(tempDir, "pets"),
		BackupPath:  filepath.Join(tempDir, "backups"),
		CacheSizeMB: 10,
	}

	dm, err := NewDataManager(config)
	if err != nil {
		t.Fatalf("NewDataManager failed: %v", err)
	}

	petID := types.PetID("test-pet-delete")
	petData := map[string]interface{}{"name": "ToDelete"}

	// Save then delete
	dm.SavePet(petID, petData)

	err = dm.DeletePet(petID)
	if err != nil {
		t.Fatalf("DeletePet failed: %v", err)
	}

	// Verify deleted
	if dm.PetExists(petID) {
		t.Error("Pet should not exist after deletion")
	}
}

func TestDataManagerListPets(t *testing.T) {
	tempDir := t.TempDir()

	config := &Config{
		BasePath:    filepath.Join(tempDir, "pets"),
		BackupPath:  filepath.Join(tempDir, "backups"),
		CacheSizeMB: 10,
	}

	dm, err := NewDataManager(config)
	if err != nil {
		t.Fatalf("NewDataManager failed: %v", err)
	}

	// Save multiple pets
	for i := 0; i < 3; i++ {
		petID := types.PetID("pet-" + string(rune('1'+i)))
		dm.SavePet(petID, map[string]interface{}{"id": i})
	}

	// List pets
	pets, err := dm.ListPets()
	if err != nil {
		t.Fatalf("ListPets failed: %v", err)
	}

	if len(pets) != 3 {
		t.Errorf("Expected 3 pets, got %d", len(pets))
	}
}

func TestCreateBackup(t *testing.T) {
	tempDir := t.TempDir()

	config := &Config{
		BasePath:    filepath.Join(tempDir, "pets"),
		BackupPath:  filepath.Join(tempDir, "backups"),
		CacheSizeMB: 10,
		MaxBackups:  5,
	}

	dm, err := NewDataManager(config)
	if err != nil {
		t.Fatalf("NewDataManager failed: %v", err)
	}

	// Save a pet first
	petID := types.PetID("test-pet-backup")
	dm.SavePet(petID, map[string]interface{}{"name": "Backup"})

	// Create backup
	filename, err := dm.CreateBackup()
	if err != nil {
		t.Fatalf("CreateBackup failed: %v", err)
	}

	if filename == "" {
		t.Error("Backup filename should not be empty")
	}

	// Verify backup file exists (filename is full path)
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		t.Errorf("Backup file should exist at %s", filename)
	}
}

func TestCreateBackupForPet(t *testing.T) {
	tempDir := t.TempDir()

	config := &Config{
		BasePath:    filepath.Join(tempDir, "pets"),
		BackupPath:  filepath.Join(tempDir, "backups"),
		CacheSizeMB: 10,
		MaxBackups:  5,
	}

	dm, err := NewDataManager(config)
	if err != nil {
		t.Fatalf("NewDataManager failed: %v", err)
	}

	// Save a pet
	petID := types.PetID("test-pet-single-backup")
	dm.SavePet(petID, map[string]interface{}{"name": "SingleBackup"})

	// Create backup for specific pet
	filename, err := dm.CreateBackupForPet(petID)
	if err != nil {
		t.Fatalf("CreateBackupForPet failed: %v", err)
	}

	if filename == "" {
		t.Error("Backup filename should not be empty")
	}
}

func TestRestoreBackup(t *testing.T) {
	tempDir := t.TempDir()

	config := &Config{
		BasePath:    filepath.Join(tempDir, "pets"),
		BackupPath:  filepath.Join(tempDir, "backups"),
		CacheSizeMB: 10,
		MaxBackups:  5,
	}

	dm, err := NewDataManager(config)
	if err != nil {
		t.Fatalf("NewDataManager failed: %v", err)
	}

	// Save pets
	petID1 := types.PetID("pet-restore-1")
	petID2 := types.PetID("pet-restore-2")
	dm.SavePet(petID1, map[string]interface{}{"name": "Pet1"})
	dm.SavePet(petID2, map[string]interface{}{"name": "Pet2"})

	// Create backup
	filename, err := dm.CreateBackup()
	if err != nil {
		t.Fatalf("CreateBackup failed: %v", err)
	}

	// Delete pets
	dm.DeletePet(petID1)
	dm.DeletePet(petID2)

	// Restore from backup
	restored, err := dm.RestoreBackup(filename)
	if err != nil {
		t.Fatalf("RestoreBackup failed: %v", err)
	}

	if len(restored) != 2 {
		t.Errorf("Expected 2 restored pets, got %d", len(restored))
	}

	// Verify pets exist after restore
	if !dm.PetExists(petID1) {
		t.Error("pet-restore-1 should exist after restore")
	}

	if !dm.PetExists(petID2) {
		t.Error("pet-restore-2 should exist after restore")
	}
}

func TestListBackups(t *testing.T) {
	tempDir := t.TempDir()

	config := &Config{
		BasePath:    filepath.Join(tempDir, "pets"),
		BackupPath:  filepath.Join(tempDir, "backups"),
		CacheSizeMB: 10,
		MaxBackups:  5,
	}

	dm, err := NewDataManager(config)
	if err != nil {
		t.Fatalf("NewDataManager failed: %v", err)
	}

	// Save a pet
	petID := types.PetID("test-pet-list-backups")
	dm.SavePet(petID, map[string]interface{}{"name": "Test"})

	// Create some backups
	for i := 0; i < 3; i++ {
		dm.CreateBackup()
		time.Sleep(1 * time.Second) // Ensure different timestamps (format is down to seconds)
	}

	// List backups
	backups, err := dm.ListBackups()
	if err != nil {
		t.Fatalf("ListBackups failed: %v", err)
	}

	if len(backups) < 3 {
		t.Errorf("Expected at least 3 backups, got %d", len(backups))
	}
}

func TestEnableDisableAutoSave(t *testing.T) {
	tempDir := t.TempDir()

	config := &Config{
		BasePath:        filepath.Join(tempDir, "pets"),
		BackupPath:      filepath.Join(tempDir, "backups"),
		CacheSizeMB:     10,
		EnableAutoSave:  false,
		AutoSaveInterval: 1 * time.Minute,
	}

	dm, err := NewDataManager(config)
	if err != nil {
		t.Fatalf("NewDataManager failed: %v", err)
	}

	// Enable auto-save
	dm.EnableAutoSave(30 * time.Second)

	if !dm.autoSaveEnabled {
		t.Error("Auto-save should be enabled")
	}

	if dm.autoSaveInterval != 30*time.Second {
		t.Errorf("Expected interval 30s, got %v", dm.autoSaveInterval)
	}

	// Disable auto-save
	dm.DisableAutoSave()

	if dm.autoSaveEnabled {
		t.Error("Auto-save should be disabled")
	}
}

func TestEnableDisableAutoBackup(t *testing.T) {
	tempDir := t.TempDir()

	config := &Config{
		BasePath:           filepath.Join(tempDir, "pets"),
		BackupPath:         filepath.Join(tempDir, "backups"),
		CacheSizeMB:        10,
		EnableAutoBackup:   false,
		AutoBackupInterval: 1 * time.Hour,
	}

	dm, err := NewDataManager(config)
	if err != nil {
		t.Fatalf("NewDataManager failed: %v", err)
	}

	// Enable auto-backup
	dm.EnableAutoBackup(30 * time.Minute)

	if !dm.autoBackupEnabled {
		t.Error("Auto-backup should be enabled")
	}

	if dm.autoBackupInterval != 30*time.Minute {
		t.Errorf("Expected interval 30m, got %v", dm.autoBackupInterval)
	}

	// Disable auto-backup
	dm.DisableAutoBackup()

	if dm.autoBackupEnabled {
		t.Error("Auto-backup should be disabled")
	}
}

func TestGetCacheStats(t *testing.T) {
	tempDir := t.TempDir()

	config := &Config{
		BasePath:    filepath.Join(tempDir, "pets"),
		BackupPath:  filepath.Join(tempDir, "backups"),
		CacheSizeMB: 10,
	}

	dm, err := NewDataManager(config)
	if err != nil {
		t.Fatalf("NewDataManager failed: %v", err)
	}

	stats := dm.GetCacheStats()

	if stats.EntryCount < 0 {
		t.Error("EntryCount should be non-negative")
	}

	if stats.CurrentSize < 0 {
		t.Error("CurrentSize should be non-negative")
	}
}

func TestClearCache(t *testing.T) {
	tempDir := t.TempDir()

	config := &Config{
		BasePath:    filepath.Join(tempDir, "pets"),
		BackupPath:  filepath.Join(tempDir, "backups"),
		CacheSizeMB: 10,
	}

	dm, err := NewDataManager(config)
	if err != nil {
		t.Fatalf("NewDataManager failed: %v", err)
	}

	// Save some pets (which will be cached)
	for i := 0; i < 3; i++ {
		petID := types.PetID("cache-test-" + string(rune('1'+i)))
		dm.SavePet(petID, map[string]interface{}{"id": i})
	}

	// Clear cache
	dm.ClearCache()

	// Verify cache is empty
	stats := dm.GetCacheStats()
	if stats.EntryCount != 0 {
		t.Errorf("Expected 0 cache entries after clear, got %d", stats.EntryCount)
	}
}

func TestInvalidateCache(t *testing.T) {
	tempDir := t.TempDir()

	config := &Config{
		BasePath:    filepath.Join(tempDir, "pets"),
		BackupPath:  filepath.Join(tempDir, "backups"),
		CacheSizeMB: 10,
	}

	dm, err := NewDataManager(config)
	if err != nil {
		t.Fatalf("NewDataManager failed: %v", err)
	}

	petID := types.PetID("invalidate-test")
	dm.SavePet(petID, map[string]interface{}{"name": "Test"})

	// Invalidate specific pet's cache
	dm.InvalidateCache(petID)

	// This is a simple test - in reality we'd verify the cache doesn't contain the pet
	// For now, just ensure it doesn't error
}

func TestSetEncryption(t *testing.T) {
	tempDir := t.TempDir()

	config := &Config{
		BasePath:    filepath.Join(tempDir, "pets"),
		BackupPath:  filepath.Join(tempDir, "backups"),
		CacheSizeMB: 10,
	}

	dm, err := NewDataManager(config)
	if err != nil {
		t.Fatalf("NewDataManager failed: %v", err)
	}

	// Enable encryption
	err = dm.SetEncryption(true, "test-encryption-key-32-chars!")
	if err != nil {
		t.Fatalf("SetEncryption failed: %v", err)
	}

	// Disable encryption
	err = dm.SetEncryption(false, "")
	if err != nil {
		t.Fatalf("SetEncryption(false) failed: %v", err)
	}
}

func TestGetStorageStats(t *testing.T) {
	tempDir := t.TempDir()

	config := &Config{
		BasePath:    filepath.Join(tempDir, "pets"),
		BackupPath:  filepath.Join(tempDir, "backups"),
		CacheSizeMB: 10,
	}

	dm, err := NewDataManager(config)
	if err != nil {
		t.Fatalf("NewDataManager failed: %v", err)
	}

	// Save some pets
	for i := 0; i < 5; i++ {
		petID := types.PetID("stats-test-" + string(rune('1'+i)))
		dm.SavePet(petID, map[string]interface{}{"id": i})
	}

	// Get storage stats
	stats, err := dm.GetStorageStats()
	if err != nil {
		t.Fatalf("GetStorageStats failed: %v", err)
	}

	if stats == nil {
		t.Fatal("Stats should not be nil")
	}

	totalPets, ok := stats["total_pets"].(int)
	if !ok {
		t.Error("total_pets should be an int")
	}

	if totalPets != 5 {
		t.Errorf("Expected 5 total pets, got %d", totalPets)
	}

	if _, ok := stats["cache_stats"]; !ok {
		t.Error("Stats should include cache_stats")
	}
}

func TestPerformMaintenance(t *testing.T) {
	tempDir := t.TempDir()

	config := &Config{
		BasePath:           filepath.Join(tempDir, "pets"),
		BackupPath:         filepath.Join(tempDir, "backups"),
		CacheSizeMB:        10,
		EnableAutoBackup:   true,
		AutoBackupInterval: 1 * time.Millisecond, // Very short for testing
		MaxBackups:         5,
	}

	dm, err := NewDataManager(config)
	if err != nil {
		t.Fatalf("NewDataManager failed: %v", err)
	}

	// Save a pet
	petID := types.PetID("maintenance-test")
	dm.SavePet(petID, map[string]interface{}{"name": "Test"})

	// Set last backup time to past
	dm.lastAutoBackup = time.Now().Add(-1 * time.Hour)

	// Perform maintenance
	err = dm.PerformMaintenance()
	if err != nil {
		t.Fatalf("PerformMaintenance failed: %v", err)
	}

	// Verify a backup was created
	backups, _ := dm.ListBackups()
	if len(backups) == 0 {
		t.Error("Maintenance should have created a backup")
	}
}

func TestSyncToCloudWithoutProvider(t *testing.T) {
	tempDir := t.TempDir()

	config := &Config{
		BasePath:    filepath.Join(tempDir, "pets"),
		BackupPath:  filepath.Join(tempDir, "backups"),
		CacheSizeMB: 10,
	}

	dm, err := NewDataManager(config)
	if err != nil {
		t.Fatalf("NewDataManager failed: %v", err)
	}

	// Try to sync without cloud provider
	_, err = dm.SyncToCloud()
	if err == nil {
		t.Error("SyncToCloud should fail without cloud provider")
	}

	// Try to sync specific pet without cloud provider
	err = dm.SyncPetToCloud(types.PetID("test"))
	if err == nil {
		t.Error("SyncPetToCloud should fail without cloud provider")
	}
}

func TestGetLastSyncResultWithoutProvider(t *testing.T) {
	tempDir := t.TempDir()

	config := &Config{
		BasePath:    filepath.Join(tempDir, "pets"),
		BackupPath:  filepath.Join(tempDir, "backups"),
		CacheSizeMB: 10,
	}

	dm, err := NewDataManager(config)
	if err != nil {
		t.Fatalf("NewDataManager failed: %v", err)
	}

	result := dm.GetLastSyncResult()
	if result != nil {
		t.Error("GetLastSyncResult should return nil without cloud provider")
	}
}

func TestPetExistsNonExistent(t *testing.T) {
	tempDir := t.TempDir()

	config := &Config{
		BasePath:    filepath.Join(tempDir, "pets"),
		BackupPath:  filepath.Join(tempDir, "backups"),
		CacheSizeMB: 10,
	}

	dm, err := NewDataManager(config)
	if err != nil {
		t.Fatalf("NewDataManager failed: %v", err)
	}

	if dm.PetExists(types.PetID("non-existent-pet")) {
		t.Error("Non-existent pet should not exist")
	}
}

func TestLoadNonExistentPet(t *testing.T) {
	tempDir := t.TempDir()

	config := &Config{
		BasePath:    filepath.Join(tempDir, "pets"),
		BackupPath:  filepath.Join(tempDir, "backups"),
		CacheSizeMB: 10,
	}

	dm, err := NewDataManager(config)
	if err != nil {
		t.Fatalf("NewDataManager failed: %v", err)
	}

	var data map[string]interface{}
	err = dm.LoadPet(types.PetID("non-existent"), &data)
	if err == nil {
		t.Error("Loading non-existent pet should fail")
	}
}


func TestExportPet(t *testing.T) {
	tempDir := t.TempDir()

	config := &Config{
		BasePath:    filepath.Join(tempDir, "pets"),
		BackupPath:  filepath.Join(tempDir, "backups"),
		CacheSizeMB: 10,
	}

	dm, err := NewDataManager(config)
	if err != nil {
		t.Fatalf("NewDataManager failed: %v", err)
	}

	petID := types.PetID("test-pet")
	exportPath := filepath.Join(tempDir, "export.json")

	// Test export (stub implementation returns nil)
	err = dm.ExportPet(petID, exportPath)
	if err != nil {
		t.Errorf("ExportPet failed: %v", err)
	}
}

func TestImportPet(t *testing.T) {
	tempDir := t.TempDir()

	config := &Config{
		BasePath:    filepath.Join(tempDir, "pets"),
		BackupPath:  filepath.Join(tempDir, "backups"),
		CacheSizeMB: 10,
	}

	dm, err := NewDataManager(config)
	if err != nil {
		t.Fatalf("NewDataManager failed: %v", err)
	}

	importPath := filepath.Join(tempDir, "import.json")

	// Test import (stub implementation returns empty string)
	petID, err := dm.ImportPet(importPath)
	if err != nil {
		t.Errorf("ImportPet failed: %v", err)
	}

	if petID != "" {
		t.Errorf("Expected empty petID from stub, got %s", petID)
	}
}

func TestCloudSyncManager(t *testing.T) {
	tempDir := t.TempDir()

	localStorage := NewLocalStorage(filepath.Join(tempDir, "pets"))
	provider := NewStubCloudProvider()
	csm := NewCloudSyncManager(provider, localStorage)

	if csm == nil {
		t.Fatal("NewCloudSyncManager returned nil")
	}

	// Test EnableAutoSync
	csm.EnableAutoSync(10 * time.Minute)
	if !csm.autoSyncEnabled {
		t.Error("AutoSync should be enabled")
	}
	if csm.syncInterval != 10*time.Minute {
		t.Errorf("Expected sync interval 10min, got %v", csm.syncInterval)
	}

	// Test DisableAutoSync
	csm.DisableAutoSync()
	if csm.autoSyncEnabled {
		t.Error("AutoSync should be disabled")
	}
}

func TestCloudProvider(t *testing.T) {
	provider := NewStubCloudProvider()

	// Test IsConnected
	if !provider.IsConnected() {
		t.Error("Provider should be connected initially")
	}

	// Test SetConnected
	provider.SetConnected(false)
	if provider.IsConnected() {
		t.Error("Provider should be disconnected")
	}
	provider.SetConnected(true)

	// Test Upload
	petID := types.PetID("test-pet")
	data := []byte(`{"name":"TestPet"}`)
	err := provider.Upload(petID, data)
	if err != nil {
		t.Errorf("Upload failed: %v", err)
	}

	// Test Download
	downloaded, err := provider.Download(petID)
	if err != nil {
		t.Errorf("Download failed: %v", err)
	}
	if string(downloaded) != string(data) {
		t.Errorf("Expected %s, got %s", string(data), string(downloaded))
	}

	// Test List
	petIDs, err := provider.List()
	if err != nil {
		t.Errorf("List failed: %v", err)
	}
	if len(petIDs) != 1 {
		t.Errorf("Expected 1 pet, got %d", len(petIDs))
	}

	// Test GetLastModified
	modTime, err := provider.GetLastModified(petID)
	if err != nil {
		t.Errorf("GetLastModified failed: %v", err)
	}
	if modTime.IsZero() {
		t.Error("ModTime should not be zero")
	}

	// Test Delete
	err = provider.Delete(petID)
	if err != nil {
		t.Errorf("Delete failed: %v", err)
	}

	// Verify deletion
	_, err = provider.Download(petID)
	if err == nil {
		t.Error("Download should fail after deletion")
	}
}

func TestSyncStatusString(t *testing.T) {
	tests := []struct {
		status   SyncStatus
		expected string
	}{
		{SyncStatusIdle, "Idle"},
		{SyncStatusSyncing, "Syncing"},
		{SyncStatusSuccess, "Success"},
		{SyncStatusFailed, "Failed"},
	}

	for _, test := range tests {
		result := test.status.String()
		if result != test.expected {
			t.Errorf("Expected %s, got %s", test.expected, result)
		}
	}
}

func TestCloudSyncManagerSyncAll(t *testing.T) {
	tempDir := t.TempDir()

	localStorage := NewLocalStorage(filepath.Join(tempDir, "pets"))
	provider := NewStubCloudProvider()
	csm := NewCloudSyncManager(provider, localStorage)

	// Save some local pets
	petID1 := types.PetID("local-pet-1")
	petID2 := types.PetID("local-pet-2")
	localStorage.Save(petID1, map[string]interface{}{"name": "LocalPet1"})
	localStorage.Save(petID2, map[string]interface{}{"name": "LocalPet2"})

	// Upload one pet to cloud
	data, _ := os.ReadFile(localStorage.getFilename(petID1))
	provider.Upload(petID1, data)

	// Run sync
	result := csm.SyncAll()

	if result == nil {
		t.Fatal("SyncAll result should not be nil")
	}

	if result.Status != SyncStatusSuccess {
		t.Errorf("Expected SyncStatusSuccess, got %s. Errors: %v", result.Status.String(), result.Errors)
	}

	if len(result.SyncedPets) != 2 {
		t.Errorf("Expected 2 synced pets, got %d", len(result.SyncedPets))
	}
}

func TestCloudSyncManagerSyncAllNoConnection(t *testing.T) {
	tempDir := t.TempDir()

	localStorage := NewLocalStorage(filepath.Join(tempDir, "pets"))
	provider := NewStubCloudProvider()
	provider.SetConnected(false)
	csm := NewCloudSyncManager(provider, localStorage)

	result := csm.SyncAll()

	if result.Status != SyncStatusFailed {
		t.Errorf("Expected SyncStatusFailed when disconnected, got %s", result.Status.String())
	}

	if len(result.Errors) == 0 {
		t.Error("Expected errors when cloud provider not connected")
	}
}

func TestCloudSyncManagerSyncPetByID(t *testing.T) {
	tempDir := t.TempDir()

	localStorage := NewLocalStorage(filepath.Join(tempDir, "pets"))
	provider := NewStubCloudProvider()
	csm := NewCloudSyncManager(provider, localStorage)

	// Save a local pet
	petID := types.PetID("sync-by-id-test")
	localStorage.Save(petID, map[string]interface{}{"name": "TestPet"})

	// Sync this specific pet
	err := csm.SyncPetByID(petID)
	if err != nil {
		t.Fatalf("SyncPetByID failed: %v", err)
	}

	// Verify it was uploaded to cloud
	cloudData, err := provider.Download(petID)
	if err != nil {
		t.Fatalf("Pet should be in cloud after sync: %v", err)
	}

	if len(cloudData) == 0 {
		t.Error("Cloud data should not be empty")
	}
}

func TestCloudSyncManagerUploadPet(t *testing.T) {
	tempDir := t.TempDir()

	localStorage := NewLocalStorage(filepath.Join(tempDir, "pets"))
	provider := NewStubCloudProvider()
	csm := NewCloudSyncManager(provider, localStorage)

	// Save a local pet
	petID := types.PetID("upload-test")
	petData := map[string]interface{}{"name": "UploadTest", "age": 5}
	localStorage.Save(petID, petData)

	// Upload to cloud
	err := csm.UploadPet(petID)
	if err != nil {
		t.Fatalf("UploadPet failed: %v", err)
	}

	// Verify it's in cloud
	cloudData, err := provider.Download(petID)
	if err != nil {
		t.Fatalf("Pet should be in cloud after upload: %v", err)
	}

	if len(cloudData) == 0 {
		t.Error("Cloud data should not be empty")
	}
}

func TestCloudSyncManagerDownloadPet(t *testing.T) {
	tempDir := t.TempDir()

	petsPath := filepath.Join(tempDir, "pets")
	os.MkdirAll(petsPath, 0755)

	localStorage := NewLocalStorage(petsPath)
	cloudPath := filepath.Join(tempDir, "cloud")
	os.MkdirAll(cloudPath, 0755)
	cloudStorage := NewLocalStorage(cloudPath)

	provider := NewStubCloudProvider()
	csm := NewCloudSyncManager(provider, localStorage)

	// Create a proper pet file in cloud storage first
	petID := types.PetID("download-test")
	cloudStorage.Save(petID, map[string]interface{}{"name": "DownloadTest", "age": 3})

	// Read the properly formatted file and upload to cloud provider
	cloudData, _ := os.ReadFile(cloudStorage.getFilename(petID))
	provider.Upload(petID, cloudData)

	// Download from cloud
	err := csm.DownloadPet(petID)
	if err != nil {
		t.Fatalf("DownloadPet failed: %v", err)
	}

	// Verify it exists locally
	var loadedData map[string]interface{}
	err = localStorage.Load(petID, &loadedData)
	if err != nil {
		t.Fatalf("Pet should exist locally after download: %v", err)
	}

	if loadedData["name"] != "DownloadTest" {
		t.Errorf("Expected name 'DownloadTest', got %v", loadedData["name"])
	}
}

func TestCloudSyncManagerGetLastSyncResult(t *testing.T) {
	tempDir := t.TempDir()

	localStorage := NewLocalStorage(filepath.Join(tempDir, "pets"))
	provider := NewStubCloudProvider()
	csm := NewCloudSyncManager(provider, localStorage)

	// Initially should be nil
	result := csm.GetLastSyncResult()
	if result != nil {
		t.Error("Initial sync result should be nil")
	}

	// Save a pet and sync
	petID := types.PetID("sync-result-test")
	localStorage.Save(petID, map[string]interface{}{"name": "Test"})
	csm.SyncAll()

	// Now should have a result
	result = csm.GetLastSyncResult()
	if result == nil {
		t.Error("Sync result should not be nil after sync")
	}

	if result.Status != SyncStatusSuccess {
		t.Errorf("Expected success status, got %s", result.Status.String())
	}
}

func TestCloudSyncManagerGetLastSyncTime(t *testing.T) {
	tempDir := t.TempDir()

	localStorage := NewLocalStorage(filepath.Join(tempDir, "pets"))
	provider := NewStubCloudProvider()
	csm := NewCloudSyncManager(provider, localStorage)

	// Initially should be zero
	lastTime := csm.GetLastSyncTime()
	if !lastTime.IsZero() {
		t.Error("Initial sync time should be zero")
	}

	// Save a pet and sync
	petID := types.PetID("sync-time-test")
	localStorage.Save(petID, map[string]interface{}{"name": "Test"})

	beforeSync := time.Now()
	csm.SyncAll()
	afterSync := time.Now()

	// Now should have a time
	lastTime = csm.GetLastSyncTime()
	if lastTime.IsZero() {
		t.Error("Sync time should not be zero after sync")
	}

	if lastTime.Before(beforeSync) || lastTime.After(afterSync) {
		t.Errorf("Sync time %v should be between %v and %v", lastTime, beforeSync, afterSync)
	}
}

func TestCloudSyncBidirectional(t *testing.T) {
	tempDir := t.TempDir()

	petsPath := filepath.Join(tempDir, "pets")
	os.MkdirAll(petsPath, 0755)

	localStorage := NewLocalStorage(petsPath)
	cloudPath := filepath.Join(tempDir, "cloud")
	os.MkdirAll(cloudPath, 0755)
	cloudStorage := NewLocalStorage(cloudPath)

	provider := NewStubCloudProvider()
	csm := NewCloudSyncManager(provider, localStorage)

	// Scenario: Local pet exists, cloud pet exists, sync should handle both
	localPetID := types.PetID("local-only")
	cloudPetID := types.PetID("cloud-only")

	// Save local pet
	localStorage.Save(localPetID, map[string]interface{}{"name": "LocalOnly"})

	// Create and upload cloud pet with proper format
	cloudStorage.Save(cloudPetID, map[string]interface{}{"name": "CloudOnly"})
	cloudData, _ := os.ReadFile(cloudStorage.getFilename(cloudPetID))
	provider.Upload(cloudPetID, cloudData)

	// Sync all
	result := csm.SyncAll()

	if result.Status != SyncStatusSuccess {
		t.Errorf("Bidirectional sync failed: %s. Errors: %v", result.Status.String(), result.Errors)
	}

	// Verify both pets exist locally
	var localData, remoteData map[string]interface{}
	if err := localStorage.Load(localPetID, &localData); err != nil {
		t.Error("Local pet should still exist after sync")
	}
	if err := localStorage.Load(cloudPetID, &remoteData); err != nil {
		t.Error("Cloud pet should be downloaded to local after sync")
	}

	// Verify both pets exist in cloud
	if _, err := provider.Download(localPetID); err != nil {
		t.Error("Local pet should be uploaded to cloud after sync")
	}
	if _, err := provider.Download(cloudPetID); err != nil {
		t.Error("Cloud pet should still exist after sync")
	}
}

func TestCloudSyncNewerVersionPriority(t *testing.T) {
	tempDir := t.TempDir()

	petsPath := filepath.Join(tempDir, "pets")
	os.MkdirAll(petsPath, 0755)

	localStorage := NewLocalStorage(petsPath)
	cloudPath := filepath.Join(tempDir, "cloud")
	os.MkdirAll(cloudPath, 0755)
	cloudStorage := NewLocalStorage(cloudPath)

	provider := NewStubCloudProvider()
	csm := NewCloudSyncManager(provider, localStorage)

	petID := types.PetID("version-test")

	// Save local version first (older)
	localStorage.Save(petID, map[string]interface{}{"name": "OldVersion", "version": 1})
	time.Sleep(100 * time.Millisecond)

	// Create and upload newer version to cloud
	cloudStorage.Save(petID, map[string]interface{}{"name": "NewVersion", "version": 2})
	newerData, _ := os.ReadFile(cloudStorage.getFilename(petID))
	provider.Upload(petID, newerData)

	// Sync should download the newer cloud version
	err := csm.SyncPetByID(petID)
	if err != nil {
		t.Fatalf("Sync failed: %v", err)
	}

	// Verify local has newer version
	var loadedData map[string]interface{}
	localStorage.Load(petID, &loadedData)

	// The cloud version should overwrite local
	version, ok := loadedData["version"].(float64)
	if ok && version == 2 {
		// Cloud version was downloaded
	} else {
		t.Error("Cloud's newer version should have been downloaded")
	}
}

func TestDeleteBackup(t *testing.T) {
	tempDir := t.TempDir()

	config := &Config{
		BasePath:    filepath.Join(tempDir, "pets"),
		BackupPath:  filepath.Join(tempDir, "backups"),
		CacheSizeMB: 10,
		MaxBackups:  5,
	}

	dm, err := NewDataManager(config)
	if err != nil {
		t.Fatalf("NewDataManager failed: %v", err)
	}

	// Save a pet
	petID := types.PetID("delete-backup-test")
	dm.SavePet(petID, map[string]interface{}{"name": "Test"})

	// Create a backup
	backupFile, err := dm.CreateBackup()
	if err != nil {
		t.Fatalf("CreateBackup failed: %v", err)
	}

	// Verify backup exists
	if _, err := os.Stat(backupFile); os.IsNotExist(err) {
		t.Fatal("Backup file should exist")
	}

	// Delete the backup
	err = dm.backupManager.DeleteBackup(backupFile)
	if err != nil {
		t.Fatalf("DeleteBackup failed: %v", err)
	}

	// Verify backup no longer exists
	if _, err := os.Stat(backupFile); !os.IsNotExist(err) {
		t.Error("Backup file should not exist after deletion")
	}
}

func TestGetLatestBackup(t *testing.T) {
	tempDir := t.TempDir()

	config := &Config{
		BasePath:    filepath.Join(tempDir, "pets"),
		BackupPath:  filepath.Join(tempDir, "backups"),
		CacheSizeMB: 10,
		MaxBackups:  5,
	}

	dm, err := NewDataManager(config)
	if err != nil {
		t.Fatalf("NewDataManager failed: %v", err)
	}

	// Initially no backups
	latest, err := dm.backupManager.GetLatestBackup()
	if err == nil {
		t.Error("GetLatestBackup should return error when no backups exist")
	}

	// Save a pet
	petID := types.PetID("latest-backup-test")
	dm.SavePet(petID, map[string]interface{}{"name": "Test"})

	// Create multiple backups
	var lastBackup string
	for i := 0; i < 3; i++ {
		backup, err := dm.CreateBackup()
		if err != nil {
			t.Fatalf("CreateBackup failed: %v", err)
		}
		lastBackup = backup
		time.Sleep(1 * time.Second) // Ensure different timestamps
	}

	// Get latest backup
	latest, err = dm.backupManager.GetLatestBackup()
	if err != nil {
		t.Fatalf("GetLatestBackup failed: %v", err)
	}

	if latest.Filename != lastBackup {
		t.Errorf("Expected latest backup %s, got %s", lastBackup, latest.Filename)
	}
}
