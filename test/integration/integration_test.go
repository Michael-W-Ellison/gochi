package integration

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Michael-W-Ellison/gochi/internal/cloud"
	"github.com/Michael-W-Ellison/gochi/internal/core"
	"github.com/Michael-W-Ellison/gochi/internal/data"
	"github.com/Michael-W-Ellison/gochi/pkg/config"
	"github.com/Michael-W-Ellison/gochi/pkg/logger"
	"github.com/Michael-W-Ellison/gochi/pkg/types"
)

// TestFullGameFlow tests the complete game workflow from start to shutdown
func TestFullGameFlow(t *testing.T) {
	// Setup test environment
	tmpDir := t.TempDir()
	dataPath := filepath.Join(tmpDir, "data")

	// Initialize logger
	logConfig := logger.DefaultConfig()
	if err := logger.Initialize(logConfig); err != nil {
		t.Fatalf("Failed to initialize logger: %v", err)
	}
	defer logger.Shutdown()

	// Create game loop with data manager
	gameConfig := core.DefaultGameLoopConfig()
	gameConfig.DataManagerConfig = &data.Config{
		BasePath:         dataPath,
		BackupPath:       filepath.Join(tmpDir, "backups"),
		CacheSizeMB:      10,
		CacheTTLMinutes:  5,
		EnableAutoSave:   false, // Manual save for testing
		EnableAutoBackup: false,
		EnableEncryption: false,
	}

	gameLoop, err := core.NewGameLoop(gameConfig)
	if err != nil {
		t.Fatalf("Failed to create game loop: %v", err)
	}

	err = gameLoop.Start()
	if err != nil {
		t.Fatalf("Failed to start game loop: %v", err)
	}

	// Create and add pet
	pet := core.NewDigitalPet("TestPet", types.UserID("user-123"))
	err = gameLoop.AddPet(pet)
	if err != nil {
		t.Fatalf("Failed to add pet: %v", err)
	}

	// Process interaction
	_, err = gameLoop.ProcessInteraction(pet.ID, types.InteractionFeeding, 0.8, nil)
	if err != nil {
		t.Fatalf("Failed to process interaction: %v", err)
	}

	// Update game state
	err = gameLoop.Update()
	if err != nil {
		t.Fatalf("Failed to update game loop: %v", err)
	}

	// Save pet
	err = gameLoop.SavePet(pet.ID)
	if err != nil {
		t.Fatalf("Failed to save pet: %v", err)
	}

	// Verify pet exists in game loop
	retrievedPet, err := gameLoop.GetPet(pet.ID)
	if err != nil {
		t.Fatalf("Failed to get pet: %v", err)
	}

	if retrievedPet.Name != pet.Name {
		t.Errorf("Pet name mismatch: expected %s, got %s", pet.Name, retrievedPet.Name)
	}

	// Create backup
	backupID, err := gameLoop.CreateBackup()
	if err != nil {
		t.Fatalf("Failed to create backup: %v", err)
	}

	if backupID == "" {
		t.Error("Backup ID should not be empty")
	}

	// Get statistics
	stats := gameLoop.GetStatistics()
	if stats == nil {
		t.Error("Statistics should not be nil")
	}

	// Graceful shutdown
	err = gameLoop.Shutdown()
	if err != nil {
		t.Fatalf("Failed to shutdown game loop: %v", err)
	}
}

// TestAuthenticationFlow tests user registration and login
func TestAuthenticationFlow(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "auth.db")

	// Create auth provider
	authProvider, err := cloud.NewSQLiteAuthProvider(dbPath)
	if err != nil {
		t.Fatalf("Failed to create auth provider: %v", err)
	}
	defer authProvider.Close()

	// Create auth manager
	authManager := cloud.NewAuthManager(authProvider)

	// Register user
	creds := &cloud.Credentials{
		Username: "testuser",
		Password: "SecurePass123!",
		Email:    "test@example.com",
	}

	err = authManager.Register(creds)
	if err != nil {
		t.Fatalf("Failed to register user: %v", err)
	}

	// Login
	err = authManager.Login(creds.Username, creds.Password)
	if err != nil {
		t.Fatalf("Failed to login: %v", err)
	}

	// Verify logged in
	if !authManager.IsLoggedIn() {
		t.Error("Should be logged in after successful login")
	}

	// Get current user
	user, err := authManager.GetCurrentUser()
	if err != nil {
		t.Fatalf("Failed to get current user: %v", err)
	}

	if user.Username != creds.Username {
		t.Errorf("Username mismatch: expected %s, got %s", creds.Username, user.Username)
	}

	// Logout
	err = authManager.Logout()
	if err != nil {
		t.Fatalf("Failed to logout: %v", err)
	}

	// Verify logged out
	if authManager.IsLoggedIn() {
		t.Error("Should not be logged in after logout")
	}
}

// TestDataPersistence tests data saving and loading
func TestDataPersistence(t *testing.T) {
	tmpDir := t.TempDir()
	dataPath := filepath.Join(tmpDir, "data")

	dataManager, err := data.NewDataManager(&data.Config{
		BasePath:   dataPath,
		BackupPath: filepath.Join(tmpDir, "backups"),
	})
	if err != nil {
		t.Fatalf("Failed to create data manager: %v", err)
	}
	defer dataManager.Close()

	// Save pet data
	petID := types.PetID("test-pet")
	petData := map[string]interface{}{
		"name":      "TestPet",
		"health":    100,
		"hunger":    50,
		"happiness": 75,
	}

	err = dataManager.SavePet(petID, petData)
	if err != nil {
		t.Fatalf("Failed to save pet: %v", err)
	}

	// Load pet data
	var loaded map[string]interface{}
	err = dataManager.LoadPet(petID, &loaded)
	if err != nil {
		t.Fatalf("Failed to load pet: %v", err)
	}

	if loaded["name"] != "TestPet" {
		t.Errorf("Pet name mismatch")
	}

	// Export pet
	exportPath := filepath.Join(tmpDir, "export.json")
	err = dataManager.ExportPet(petID, exportPath)
	if err != nil {
		t.Fatalf("Failed to export pet: %v", err)
	}

	// Verify export file exists
	if _, err := os.Stat(exportPath); err != nil {
		t.Errorf("Export file not created: %v", err)
	}

	// Import pet with new ID
	newPetID, err := dataManager.ImportPet(exportPath)
	if err != nil {
		t.Fatalf("Failed to import pet: %v", err)
	}

	if newPetID == "" {
		t.Error("Imported pet ID should not be empty")
	}
}

// TestConfigurationCreation tests configuration object creation
func TestConfigurationCreation(t *testing.T) {
	// Create default config
	appConfig := config.DefaultConfig()
	if appConfig.App.Name == "" {
		t.Error("App name should not be empty")
	}

	// Create data config from app config
	dataConfig := appConfig.ToDataConfig()
	if dataConfig.BasePath == "" {
		t.Error("Base path should not be empty")
	}

	// Initialize logger with default config
	loggerConfig := logger.DefaultConfig()
	err := logger.Initialize(loggerConfig)
	if err != nil {
		t.Fatalf("Failed to initialize logger: %v", err)
	}
	defer logger.Shutdown()

	// Test logging
	logger.Info("test message")
	logger.Debug("debug message")
}

// TestSecurityFeatures tests security-related functionality
func TestSecurityFeatures(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "auth.db")

	authProvider, err := cloud.NewSQLiteAuthProvider(dbPath)
	if err != nil {
		t.Fatalf("Failed to create auth provider: %v", err)
	}
	defer authProvider.Close()

	authManager := cloud.NewAuthManager(authProvider)

	// Test 1: Weak password rejection
	weakCreds := &cloud.Credentials{
		Username: "testuser",
		Password: "weak",
		Email:    "test@example.com",
	}

	err = authManager.Register(weakCreds)
	if err == nil {
		t.Error("Weak password should be rejected")
	}

	// Test 2: Strong password acceptance
	strongCreds := &cloud.Credentials{
		Username: "testuser",
		Password: "StrongPass123!",
		Email:    "test@example.com",
	}

	err = authManager.Register(strongCreds)
	if err != nil {
		t.Fatalf("Strong password should be accepted: %v", err)
	}

	// Test 3: Successful login
	err = authManager.Login(strongCreds.Username, strongCreds.Password)
	if err != nil {
		t.Fatalf("Valid login should succeed: %v", err)
	}

	if !authManager.IsLoggedIn() {
		t.Error("Should be logged in after successful login")
	}

	// Test 4: Path traversal protection
	dataPath := filepath.Join(tmpDir, "data")
	dataManager, err := data.NewDataManager(&data.Config{
		BasePath:   dataPath,
		BackupPath: filepath.Join(tmpDir, "backups"),
	})
	if err != nil {
		t.Fatalf("Failed to create data manager: %v", err)
	}
	defer dataManager.Close()

	// Try to save with malicious pet ID
	maliciousPetID := types.PetID("../../etc/passwd")
	petData := map[string]interface{}{"name": "Evil"}

	err = dataManager.SavePet(maliciousPetID, petData)
	if err == nil {
		t.Error("Path traversal attempt should be blocked")
	}
}

// TestMultiplePets tests handling multiple pets concurrently
func TestMultiplePets(t *testing.T) {
	tmpDir := t.TempDir()

	gameConfig := core.DefaultGameLoopConfig()
	gameConfig.DataManagerConfig = &data.Config{
		BasePath:   filepath.Join(tmpDir, "data"),
		BackupPath: filepath.Join(tmpDir, "backups"),
	}

	gameLoop, err := core.NewGameLoop(gameConfig)
	if err != nil {
		t.Fatalf("Failed to create game loop: %v", err)
	}

	gameLoop.Start()
	defer gameLoop.Shutdown()

	// Add multiple pets
	petCount := 5
	petIDs := make([]types.PetID, petCount)

	for i := 0; i < petCount; i++ {
		pet := core.NewDigitalPet("Pet"+string(rune('A'+i)), types.UserID("user-123"))
		err = gameLoop.AddPet(pet)
		if err != nil {
			t.Fatalf("Failed to add pet %d: %v", i, err)
		}
		petIDs[i] = pet.ID
	}

	// Update game state
	for i := 0; i < 10; i++ {
		err = gameLoop.Update()
		if err != nil {
			t.Fatalf("Failed to update: %v", err)
		}
		time.Sleep(10 * time.Millisecond)
	}

	// Verify all pets exist
	allPets := gameLoop.GetAllPets()
	if len(allPets) != petCount {
		t.Errorf("Expected %d pets, got %d", petCount, len(allPets))
	}

	// Save all pets
	err = gameLoop.SaveAllPets()
	if err != nil {
		t.Fatalf("Failed to save all pets: %v", err)
	}
}
