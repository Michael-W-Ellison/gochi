package core

import (
	"testing"
	"time"

	"github.com/Michael-W-Ellison/gochi/pkg/types"
)

func TestNewGameLoop(t *testing.T) {
	config := DefaultGameLoopConfig()
	gl, err := NewGameLoop(config)

	if err != nil {
		t.Fatalf("NewGameLoop failed: %v", err)
	}

	if gl == nil {
		t.Fatal("NewGameLoop returned nil")
	}

	if gl.state != GameStateInitializing {
		t.Errorf("Initial state = %v, want %v", gl.state, GameStateInitializing)
	}

	if gl.running {
		t.Error("Game loop should not be running initially")
	}
}

func TestGameLoopStartStop(t *testing.T) {
	config := DefaultGameLoopConfig()
	gl, _ := NewGameLoop(config)

	// Start
	err := gl.Start()
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	if !gl.IsRunning() {
		t.Error("Game loop should be running after Start")
	}

	if gl.GetState() != GameStateRunning {
		t.Errorf("State = %v, want %v", gl.GetState(), GameStateRunning)
	}

	// Stop
	err = gl.Stop()
	if err != nil {
		t.Fatalf("Stop failed: %v", err)
	}

	if gl.IsRunning() {
		t.Error("Game loop should not be running after Stop")
	}

	if gl.GetState() != GameStateStopped {
		t.Errorf("State = %v, want %v", gl.GetState(), GameStateStopped)
	}
}

func TestGameLoopPauseResume(t *testing.T) {
	config := DefaultGameLoopConfig()
	gl, _ := NewGameLoop(config)

	gl.Start()

	// Pause
	gl.Pause()

	if !gl.IsPaused() {
		t.Error("Game loop should be paused")
	}

	if gl.GetState() != GameStatePaused {
		t.Errorf("State = %v, want %v", gl.GetState(), GameStatePaused)
	}

	// Resume
	gl.Resume()

	if gl.IsPaused() {
		t.Error("Game loop should not be paused after Resume")
	}

	if gl.GetState() != GameStateRunning {
		t.Errorf("State = %v, want %v", gl.GetState(), GameStateRunning)
	}

	gl.Stop()
}

func TestGameLoopUpdate(t *testing.T) {
	config := DefaultGameLoopConfig()
	gl, _ := NewGameLoop(config)

	gl.Start()

	// Run a few updates
	for i := 0; i < 5; i++ {
		err := gl.Update()
		if err != nil {
			t.Fatalf("Update failed: %v", err)
		}
	}

	stats := gl.GetStatistics()
	totalUpdates := stats["total_updates"].(int64)

	if totalUpdates != 5 {
		t.Errorf("Total updates = %d, want 5", totalUpdates)
	}

	gl.Stop()
}

func TestGameLoopRunOnce(t *testing.T) {
	config := DefaultGameLoopConfig()
	gl, _ := NewGameLoop(config)

	err := gl.RunOnce()
	if err != nil {
		t.Fatalf("RunOnce failed: %v", err)
	}

	if !gl.IsRunning() {
		t.Error("Game loop should be running after RunOnce")
	}

	stats := gl.GetStatistics()
	totalUpdates := stats["total_updates"].(int64)

	if totalUpdates != 1 {
		t.Errorf("Total updates = %d, want 1", totalUpdates)
	}

	gl.Stop()
}

func TestAddPet(t *testing.T) {
	config := DefaultGameLoopConfig()
	gl, _ := NewGameLoop(config)

	pet := NewDigitalPet("Fluffy", "user_123")

	err := gl.AddPet(pet)
	if err != nil {
		t.Fatalf("AddPet failed: %v", err)
	}

	// Retrieve pet
	retrievedPet, err := gl.GetPet(pet.ID)
	if err != nil {
		t.Fatalf("GetPet failed: %v", err)
	}

	if retrievedPet.ID != pet.ID {
		t.Errorf("Pet ID = %v, want %v", retrievedPet.ID, pet.ID)
	}

	if retrievedPet.Name != pet.Name {
		t.Errorf("Pet name = %s, want %s", retrievedPet.Name, pet.Name)
	}
}

func TestAddNilPet(t *testing.T) {
	config := DefaultGameLoopConfig()
	gl, _ := NewGameLoop(config)

	err := gl.AddPet(nil)
	if err == nil {
		t.Error("AddPet should fail with nil pet")
	}
}

func TestRemovePet(t *testing.T) {
	config := DefaultGameLoopConfig()
	gl, _ := NewGameLoop(config)

	pet := NewDigitalPet("Fluffy", "user_123")
	gl.AddPet(pet)

	// Remove pet
	err := gl.RemovePet(pet.ID)
	if err != nil {
		t.Fatalf("RemovePet failed: %v", err)
	}

	// Should not be found
	_, err = gl.GetPet(pet.ID)
	if err == nil {
		t.Error("GetPet should fail after removal")
	}
}

func TestGetAllPets(t *testing.T) {
	config := DefaultGameLoopConfig()
	gl, _ := NewGameLoop(config)

	// Add multiple pets
	pet1 := NewDigitalPet("Pet1", "user_123")
	pet2 := NewDigitalPet("Pet2", "user_123")
	pet3 := NewDigitalPet("Pet3", "user_123")

	gl.AddPet(pet1)
	gl.AddPet(pet2)
	gl.AddPet(pet3)

	pets := gl.GetAllPets()

	if len(pets) != 3 {
		t.Errorf("GetAllPets returned %d pets, want 3", len(pets))
	}
}

func TestSavePet(t *testing.T) {
	config := DefaultGameLoopConfig()
	config.DataManagerConfig.BasePath = t.TempDir()
	gl, _ := NewGameLoop(config)

	pet := NewDigitalPet("SaveTest", "user_123")
	gl.AddPet(pet)

	err := gl.SavePet(pet.ID)
	if err != nil {
		t.Fatalf("SavePet failed: %v", err)
	}

	// Verify file exists
	if !gl.dataManager.PetExists(pet.ID) {
		t.Error("Pet save file should exist")
	}
}

func TestSaveAllPets(t *testing.T) {
	config := DefaultGameLoopConfig()
	config.DataManagerConfig.BasePath = t.TempDir()
	gl, _ := NewGameLoop(config)

	pet1 := NewDigitalPet("Pet1", "user_123")
	pet2 := NewDigitalPet("Pet2", "user_123")

	gl.AddPet(pet1)
	gl.AddPet(pet2)

	err := gl.SaveAllPets()
	if err != nil {
		t.Fatalf("SaveAllPets failed: %v", err)
	}

	// Verify both files exist
	if !gl.dataManager.PetExists(pet1.ID) {
		t.Error("Pet1 save file should exist")
	}

	if !gl.dataManager.PetExists(pet2.ID) {
		t.Error("Pet2 save file should exist")
	}
}

func TestLoadPet(t *testing.T) {
	config := DefaultGameLoopConfig()
	config.DataManagerConfig.BasePath = t.TempDir()
	gl, _ := NewGameLoop(config)

	// Create and save a pet
	pet := NewDigitalPet("LoadTest", "user_123")
	gl.dataManager.SavePet(pet.ID, pet)

	// Load the pet
	err := gl.LoadPet(pet.ID)
	if err != nil {
		t.Fatalf("LoadPet failed: %v", err)
	}

	// Verify pet is loaded
	loadedPet, err := gl.GetPet(pet.ID)
	if err != nil {
		t.Fatalf("GetPet failed: %v", err)
	}

	if loadedPet.Name != pet.Name {
		t.Errorf("Loaded pet name = %s, want %s", loadedPet.Name, pet.Name)
	}
}

func TestCreateBackup(t *testing.T) {
	config := DefaultGameLoopConfig()
	config.DataManagerConfig.BasePath = t.TempDir()
	config.DataManagerConfig.BackupPath = t.TempDir()
	gl, _ := NewGameLoop(config)

	pet := NewDigitalPet("BackupTest", "user_123")
	gl.AddPet(pet)
	gl.SavePet(pet.ID)

	filename, err := gl.CreateBackup()
	if err != nil {
		t.Fatalf("CreateBackup failed: %v", err)
	}

	if filename == "" {
		t.Error("Backup filename should not be empty")
	}

	stats := gl.GetStatistics()
	totalBackups := stats["total_backups"].(int64)

	if totalBackups != 1 {
		t.Errorf("Total backups = %d, want 1", totalBackups)
	}
}

func TestSetTimeScale(t *testing.T) {
	config := DefaultGameLoopConfig()
	gl, _ := NewGameLoop(config)

	gl.Start()

	// Set to accelerated
	gl.SetTimeScale(types.TimeScaleAccelerated4X)

	scale := gl.GetTimeScale()
	if scale != types.TimeScaleAccelerated4X {
		t.Errorf("Time scale = %v, want %v", scale, types.TimeScaleAccelerated4X)
	}

	gl.Stop()
}

func TestGetStatistics(t *testing.T) {
	config := DefaultGameLoopConfig()
	gl, _ := NewGameLoop(config)

	gl.Start()
	gl.Update()

	stats := gl.GetStatistics()

	if stats["state"] != "Running" {
		t.Errorf("State = %v, want Running", stats["state"])
	}

	if stats["frame_count"].(int64) < 1 {
		t.Error("Frame count should be at least 1")
	}

	if stats["active_pets"].(int) != 0 {
		t.Errorf("Active pets = %d, want 0", stats["active_pets"])
	}

	gl.Stop()
}

func TestPetUpdate(t *testing.T) {
	config := DefaultGameLoopConfig()
	gl, _ := NewGameLoop(config)

	pet := NewDigitalPet("UpdateTest", "user_123")
	gl.AddPet(pet)

	gl.Start()

	initialPlayTime := pet.TotalPlayTime

	// Run several updates
	for i := 0; i < 10; i++ {
		gl.Update()
	}

	// Play time should have increased
	if pet.TotalPlayTime <= initialPlayTime {
		t.Error("Pet play time should have increased")
	}

	gl.Stop()
}

func TestProcessInteraction(t *testing.T) {
	config := DefaultGameLoopConfig()
	gl, _ := NewGameLoop(config)

	pet := NewDigitalPet("InteractionTest", "user_123")
	gl.AddPet(pet)

	initialHappiness := pet.Biology.Vitals.Happiness

	result, err := gl.ProcessInteraction(pet.ID, types.InteractionPetting, 1.0, nil)
	if err != nil {
		t.Fatalf("ProcessInteraction failed: %v", err)
	}

	if result == nil {
		t.Fatal("Result should not be nil")
	}

	// Happiness should have increased from petting
	if pet.Biology.Vitals.Happiness <= initialHappiness {
		t.Error("Happiness should increase from petting")
	}
}

func TestProcessInteractionNonExistentPet(t *testing.T) {
	config := DefaultGameLoopConfig()
	gl, _ := NewGameLoop(config)

	_, err := gl.ProcessInteraction("nonexistent", types.InteractionPetting, 1.0, nil)
	if err == nil {
		t.Error("ProcessInteraction should fail for non-existent pet")
	}
}

func TestGameStateString(t *testing.T) {
	tests := []struct {
		state GameState
		want  string
	}{
		{GameStateInitializing, "Initializing"},
		{GameStateRunning, "Running"},
		{GameStatePaused, "Paused"},
		{GameStateStopped, "Stopped"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := tt.state.String()
			if got != tt.want {
				t.Errorf("String() = %s, want %s", got, tt.want)
			}
		})
	}
}

func TestDefaultGameLoopConfig(t *testing.T) {
	config := DefaultGameLoopConfig()

	if config.TargetFPS != 60 {
		t.Errorf("TargetFPS = %d, want 60", config.TargetFPS)
	}

	if config.AutoSaveInterval != 5*time.Minute {
		t.Errorf("AutoSaveInterval = %v, want 5m", config.AutoSaveInterval)
	}

	if !config.EnableAutoSave {
		t.Error("EnableAutoSave should be true")
	}

	if config.DataManagerConfig == nil {
		t.Error("DataManagerConfig should not be nil")
	}
}

func TestEventEmissionOnPetAdd(t *testing.T) {
	config := DefaultGameLoopConfig()
	gl, _ := NewGameLoop(config)

	eventReceived := false
	gl.eventSystem.RegisterHandler(EventPetCreated, func(event *GameEvent) {
		eventReceived = true
	})

	pet := NewDigitalPet("EventTest", "user_123")
	gl.AddPet(pet)

	// Give time for event handler to execute
	time.Sleep(10 * time.Millisecond)

	if !eventReceived {
		t.Error("EventPetCreated should be emitted when adding pet")
	}
}

func TestGetEventSystem(t *testing.T) {
	config := DefaultGameLoopConfig()
	gl, _ := NewGameLoop(config)

	es := gl.GetEventSystem()

	if es == nil {
		t.Error("GetEventSystem should not return nil")
	}

	if es != gl.eventSystem {
		t.Error("GetEventSystem should return the same instance")
	}
}

func TestGetDataManager(t *testing.T) {
	config := DefaultGameLoopConfig()
	gl, _ := NewGameLoop(config)

	dm := gl.GetDataManager()

	if dm == nil {
		t.Error("GetDataManager should not return nil")
	}

	if dm != gl.dataManager {
		t.Error("GetDataManager should return the same instance")
	}
}

func TestGetTimeManager(t *testing.T) {
	config := DefaultGameLoopConfig()
	gl, _ := NewGameLoop(config)

	tm := gl.GetTimeManager()

	if tm == nil {
		t.Error("GetTimeManager should not return nil")
	}

	if tm != gl.timeManager {
		t.Error("GetTimeManager should return the same instance")
	}
}

func TestDoubleStart(t *testing.T) {
	config := DefaultGameLoopConfig()
	gl, _ := NewGameLoop(config)

	// First start should succeed
	err := gl.Start()
	if err != nil {
		t.Fatalf("First Start failed: %v", err)
	}

	// Second start should fail
	err = gl.Start()
	if err == nil {
		t.Error("Second Start should fail")
	}

	gl.Stop()
}

func TestStopWithoutStart(t *testing.T) {
	config := DefaultGameLoopConfig()
	gl, _ := NewGameLoop(config)

	err := gl.Stop()
	if err == nil {
		t.Error("Stop should fail when not running")
	}
}

func TestUpdateWhilePaused(t *testing.T) {
	config := DefaultGameLoopConfig()
	gl, _ := NewGameLoop(config)

	gl.Start()
	gl.Pause()

	initialUpdates := gl.totalUpdates

	// Update while paused
	gl.Update()

	// Update should return early without incrementing counter
	if gl.totalUpdates != initialUpdates {
		t.Error("Updates should not increment while paused")
	}

	gl.Stop()
}

func TestConcurrentPetAccess(t *testing.T) {
	config := DefaultGameLoopConfig()
	gl, _ := NewGameLoop(config)

	pet := NewDigitalPet("ConcurrentTest", "user_123")
	gl.AddPet(pet)

	gl.Start()

	// Concurrent reads
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func() {
			gl.GetPet(pet.ID)
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	gl.Stop()
}
