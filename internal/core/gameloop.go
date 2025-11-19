package core

import (
	"fmt"
	"sync"
	"time"

	"github.com/Michael-W-Ellison/gochi/internal/data"
	"github.com/Michael-W-Ellison/gochi/internal/interaction"
	"github.com/Michael-W-Ellison/gochi/internal/simulation"
	"github.com/Michael-W-Ellison/gochi/pkg/types"
)

// GameState represents the current state of the game
type GameState int

const (
	GameStateInitializing GameState = iota
	GameStateRunning
	GameStatePaused
	GameStateStopped
)

func (gs GameState) String() string {
	return [...]string{
		"Initializing", "Running", "Paused", "Stopped",
	}[gs]
}

// GameLoopConfig holds configuration for the game loop
type GameLoopConfig struct {
	TargetFPS          int
	AutoSaveInterval   time.Duration
	AutoBackupInterval time.Duration
	EnableAutoSave     bool
	EnableAutoBackup   bool
	DataManagerConfig  *data.Config
}

// DefaultGameLoopConfig returns default game loop configuration
func DefaultGameLoopConfig() *GameLoopConfig {
	return &GameLoopConfig{
		TargetFPS:          60,
		AutoSaveInterval:   5 * time.Minute,
		AutoBackupInterval: 24 * time.Hour,
		EnableAutoSave:     true,
		EnableAutoBackup:   true,
		DataManagerConfig:  data.DefaultConfig(),
	}
}

// GameLoop is the main game loop that orchestrates all systems
type GameLoop struct {
	mu sync.RWMutex

	// Configuration
	config *GameLoopConfig

	// State
	state           GameState
	running         bool
	paused          bool
	startTime       time.Time
	totalGameTime   float64 // Total game time in seconds
	frameCount      int64
	lastFrameTime   time.Time
	currentFPS      float64

	// Core Systems
	timeManager       *simulation.TimeManager
	eventSystem       *EventSystem
	dataManager       *data.DataManager
	interactionSystem *interaction.InteractionProcessor

	// Active Pets
	activePets map[types.PetID]*DigitalPet

	// Auto-save tracking
	lastAutoSave   time.Time
	lastAutoBackup time.Time

	// Statistics
	totalUpdates   int64
	totalSaves     int64
	totalBackups   int64
	averageFrameMS float64
}

// NewGameLoop creates a new game loop with the given configuration
func NewGameLoop(config *GameLoopConfig) (*GameLoop, error) {
	if config == nil {
		config = DefaultGameLoopConfig()
	}

	// Create data manager
	dataManager, err := data.NewDataManager(config.DataManagerConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create data manager: %w", err)
	}

	gl := &GameLoop{
		config:            config,
		state:             GameStateInitializing,
		running:           false,
		paused:            false,
		timeManager:       simulation.NewTimeManager(types.TimeScaleRealTime),
		eventSystem:       NewEventSystem(),
		dataManager:       dataManager,
		interactionSystem: interaction.NewInteractionProcessor(),
		activePets:        make(map[types.PetID]*DigitalPet),
		currentFPS:        0,
	}

	// Register event handlers
	gl.setupEventHandlers()

	return gl, nil
}

// Start starts the game loop
func (gl *GameLoop) Start() error {
	gl.mu.Lock()
	defer gl.mu.Unlock()

	if gl.running {
		return fmt.Errorf("game loop is already running")
	}

	gl.running = true
	gl.paused = false
	gl.state = GameStateRunning
	gl.startTime = time.Now()
	gl.lastFrameTime = time.Now()
	gl.lastAutoSave = time.Now()
	gl.lastAutoBackup = time.Now()

	// Start the time manager
	gl.timeManager.SetTimeScale(types.TimeScaleRealTime)

	// Emit event
	gl.eventSystem.Emit(CreateEvent(EventAutoSave, "", "Game loop started"))

	return nil
}

// Stop stops the game loop
func (gl *GameLoop) Stop() error {
	gl.mu.Lock()
	defer gl.mu.Unlock()

	if !gl.running {
		return fmt.Errorf("game loop is not running")
	}

	// Save all pets before stopping
	for petID, pet := range gl.activePets {
		if err := gl.dataManager.SavePet(petID, pet); err != nil {
			fmt.Printf("Warning: failed to save pet %s: %v\n", petID, err)
		}
	}

	gl.running = false
	gl.state = GameStateStopped

	return nil
}

// Pause pauses the game loop
func (gl *GameLoop) Pause() {
	gl.mu.Lock()
	defer gl.mu.Unlock()

	gl.paused = true
	gl.state = GameStatePaused
	gl.timeManager.SetTimeScale(types.TimeScalePaused)
}

// Resume resumes the game loop
func (gl *GameLoop) Resume() {
	gl.mu.Lock()
	defer gl.mu.Unlock()

	gl.paused = false
	gl.state = GameStateRunning
	gl.timeManager.SetTimeScale(types.TimeScaleRealTime)
	gl.lastFrameTime = time.Now() // Reset to avoid large delta
}

// Update performs one update cycle
func (gl *GameLoop) Update() error {
	gl.mu.Lock()

	if !gl.running || gl.paused {
		gl.mu.Unlock()
		return nil
	}

	// Calculate delta time
	now := time.Now()
	deltaTime := now.Sub(gl.lastFrameTime).Seconds()
	gl.lastFrameTime = now

	// Update frame statistics
	gl.frameCount++
	gl.totalUpdates++

	// Calculate FPS
	if deltaTime > 0 {
		instantFPS := 1.0 / deltaTime
		// Smooth FPS with exponential moving average
		gl.currentFPS = gl.currentFPS*0.9 + instantFPS*0.1
		gl.averageFrameMS = deltaTime * 1000.0
	}

	// Update time manager
	scaledDelta := gl.timeManager.Update()
	gl.totalGameTime += scaledDelta

	// Update all active pets
	petUpdates := make(map[types.PetID]*DigitalPet)
	for petID, pet := range gl.activePets {
		petUpdates[petID] = pet
	}

	gl.mu.Unlock()

	// Update pets without holding the main lock
	for _, pet := range petUpdates {
		pet.Update(scaledDelta)
		gl.checkPetConditions(pet)
	}

	gl.mu.Lock()

	// Update interaction system (skill decay)
	gl.interactionSystem.UpdateSkillDecay(scaledDelta)

	// Check auto-save
	if gl.config.EnableAutoSave && time.Since(gl.lastAutoSave) > gl.config.AutoSaveInterval {
		gl.mu.Unlock()
		gl.performAutoSave()
		gl.mu.Lock()
	}

	// Check auto-backup
	if gl.config.EnableAutoBackup && time.Since(gl.lastAutoBackup) > gl.config.AutoBackupInterval {
		gl.mu.Unlock()
		gl.performAutoBackup()
		gl.mu.Lock()
	}

	// Perform data manager maintenance
	gl.dataManager.PerformMaintenance()

	gl.mu.Unlock()

	return nil
}

// Run runs the game loop at the target FPS
func (gl *GameLoop) Run() error {
	if err := gl.Start(); err != nil {
		return err
	}

	targetFrameTime := time.Duration(1000000000 / gl.config.TargetFPS)

	for gl.IsRunning() {
		frameStart := time.Now()

		if err := gl.Update(); err != nil {
			return err
		}

		// Sleep to maintain target FPS
		frameDuration := time.Since(frameStart)
		if frameDuration < targetFrameTime {
			time.Sleep(targetFrameTime - frameDuration)
		}
	}

	return nil
}

// RunOnce runs a single update cycle (useful for testing)
func (gl *GameLoop) RunOnce() error {
	if !gl.running {
		if err := gl.Start(); err != nil {
			return err
		}
	}

	return gl.Update()
}

// AddPet adds a pet to the active game
func (gl *GameLoop) AddPet(pet *DigitalPet) error {
	gl.mu.Lock()
	defer gl.mu.Unlock()

	if pet == nil {
		return fmt.Errorf("pet cannot be nil")
	}

	gl.activePets[pet.ID] = pet

	// Emit event
	gl.eventSystem.Emit(CreateEvent(EventPetCreated, pet.ID,
		fmt.Sprintf("Pet %s added to game", pet.Name)))

	return nil
}

// RemovePet removes a pet from the active game
func (gl *GameLoop) RemovePet(petID types.PetID) error {
	gl.mu.Lock()
	defer gl.mu.Unlock()

	pet, exists := gl.activePets[petID]
	if !exists {
		return fmt.Errorf("pet %s not found", petID)
	}

	// Save before removing
	if err := gl.dataManager.SavePet(petID, pet); err != nil {
		return fmt.Errorf("failed to save pet before removing: %w", err)
	}

	delete(gl.activePets, petID)

	return nil
}

// GetPet retrieves an active pet
func (gl *GameLoop) GetPet(petID types.PetID) (*DigitalPet, error) {
	gl.mu.RLock()
	defer gl.mu.RUnlock()

	pet, exists := gl.activePets[petID]
	if !exists {
		return nil, fmt.Errorf("pet %s not found", petID)
	}

	return pet, nil
}

// GetAllPets returns all active pets
func (gl *GameLoop) GetAllPets() []*DigitalPet {
	gl.mu.RLock()
	defer gl.mu.RUnlock()

	pets := make([]*DigitalPet, 0, len(gl.activePets))
	for _, pet := range gl.activePets {
		pets = append(pets, pet)
	}

	return pets
}

// LoadPet loads a pet from storage and adds it to the game
func (gl *GameLoop) LoadPet(petID types.PetID) error {
	gl.mu.Lock()
	defer gl.mu.Unlock()

	// Check if already loaded
	if _, exists := gl.activePets[petID]; exists {
		return fmt.Errorf("pet %s is already loaded", petID)
	}

	// Load from data manager
	var pet DigitalPet
	if err := gl.dataManager.LoadPet(petID, &pet); err != nil {
		return fmt.Errorf("failed to load pet: %w", err)
	}

	gl.activePets[petID] = &pet

	return nil
}

// SavePet saves a specific pet
func (gl *GameLoop) SavePet(petID types.PetID) error {
	gl.mu.RLock()
	pet, exists := gl.activePets[petID]
	gl.mu.RUnlock()

	if !exists {
		return fmt.Errorf("pet %s not found", petID)
	}

	if err := gl.dataManager.SavePet(petID, pet); err != nil {
		return fmt.Errorf("failed to save pet: %w", err)
	}

	gl.totalSaves++

	return nil
}

// SaveAllPets saves all active pets
func (gl *GameLoop) SaveAllPets() error {
	gl.mu.RLock()
	petsCopy := make(map[types.PetID]*DigitalPet)
	for petID, pet := range gl.activePets {
		petsCopy[petID] = pet
	}
	gl.mu.RUnlock()

	for petID, pet := range petsCopy {
		if err := gl.dataManager.SavePet(petID, pet); err != nil {
			return fmt.Errorf("failed to save pet %s: %w", petID, err)
		}
		gl.totalSaves++
	}

	return nil
}

// CreateBackup creates a backup of all game data
func (gl *GameLoop) CreateBackup() (string, error) {
	// Save all pets first
	if err := gl.SaveAllPets(); err != nil {
		return "", fmt.Errorf("failed to save pets before backup: %w", err)
	}

	filename, err := gl.dataManager.CreateBackup()
	if err != nil {
		return "", fmt.Errorf("failed to create backup: %w", err)
	}

	gl.totalBackups++

	// Emit event
	gl.eventSystem.Emit(CreateEvent(EventAutoBackup, "",
		fmt.Sprintf("Backup created: %s", filename)))

	return filename, nil
}

// ProcessInteraction handles a user interaction with a pet
func (gl *GameLoop) ProcessInteraction(petID types.PetID, interactionType types.InteractionType,
	intensity float64, itemData interface{}) (*interaction.InteractionResult, error) {

	gl.mu.RLock()
	pet, exists := gl.activePets[petID]
	gl.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("pet %s not found", petID)
	}

	// Create interaction context
	context := &interaction.InteractionContext{
		Vitals:         pet.Biology.Vitals,
		Emotions:       pet.Emotions,
		Personality:    pet.Personality,
		Needs:          nil, // Would need to add NeedsManager to pet
		Memory:         pet.Memory,
		Learning:       nil, // Would need to add LearningSystem to pet
		Energy:         pet.Biology.Vitals.Energy,
		CurrentMood:    pet.Emotions.GetMoodScore(),
		Fatigue:        pet.Biology.Vitals.Fatigue,
		Age:            int(pet.Biology.GetAgeInDays()),
		UserID:         pet.Owner,
		UserSkillLevel: 0.5,
		BondStrength:   0.7,
	}

	// Process interaction
	result := gl.interactionSystem.ProcessInteraction(interactionType, intensity, context, itemData)

	// Update pet stats based on result
	for vital, change := range result.VitalChanges {
		// Apply changes to vitals
		switch vital {
		case "Health":
			pet.Biology.Vitals.Health += change
		case "Energy":
			pet.Biology.Vitals.Energy += change
		case "Happiness":
			pet.Biology.Vitals.Happiness += change
		case "Stress":
			pet.Biology.Vitals.Stress += change
		}
	}

	pet.Biology.Vitals.Clamp()

	// Emit event
	gl.eventSystem.Emit(CreateEventWithData(EventInteraction, petID,
		fmt.Sprintf("Interaction: %s", interactionType.String()),
		map[string]interface{}{
			"type":          interactionType.String(),
			"intensity":     intensity,
			"effectiveness": result.Effectiveness,
		}))

	return result, nil
}

// SetTimeScale sets the game time scale
func (gl *GameLoop) SetTimeScale(scale types.TimeScale) {
	gl.mu.Lock()
	defer gl.mu.Unlock()

	gl.timeManager.SetTimeScale(scale)
}

// GetTimeScale returns the current time scale
func (gl *GameLoop) GetTimeScale() types.TimeScale {
	gl.mu.RLock()
	defer gl.mu.RUnlock()

	return gl.timeManager.GetTimeScale()
}

// GetState returns the current game state
func (gl *GameLoop) GetState() GameState {
	gl.mu.RLock()
	defer gl.mu.RUnlock()

	return gl.state
}

// IsRunning checks if the game loop is running
func (gl *GameLoop) IsRunning() bool {
	gl.mu.RLock()
	defer gl.mu.RUnlock()

	return gl.running
}

// IsPaused checks if the game loop is paused
func (gl *GameLoop) IsPaused() bool {
	gl.mu.RLock()
	defer gl.mu.RUnlock()

	return gl.paused
}

// GetStatistics returns game loop statistics
func (gl *GameLoop) GetStatistics() map[string]interface{} {
	gl.mu.RLock()
	defer gl.mu.RUnlock()

	return map[string]interface{}{
		"state":              gl.state.String(),
		"uptime_seconds":     time.Since(gl.startTime).Seconds(),
		"total_game_time":    gl.totalGameTime,
		"frame_count":        gl.frameCount,
		"current_fps":        gl.currentFPS,
		"average_frame_ms":   gl.averageFrameMS,
		"active_pets":        len(gl.activePets),
		"total_updates":      gl.totalUpdates,
		"total_saves":        gl.totalSaves,
		"total_backups":      gl.totalBackups,
		"cache_stats":        gl.dataManager.GetCacheStats(),
	}
}

// Helper methods

func (gl *GameLoop) performAutoSave() {
	if err := gl.SaveAllPets(); err != nil {
		fmt.Printf("Auto-save failed: %v\n", err)
	} else {
		gl.lastAutoSave = time.Now()
		gl.eventSystem.Emit(CreateEvent(EventAutoSave, "", "Auto-save completed"))
	}
}

func (gl *GameLoop) performAutoBackup() {
	if _, err := gl.CreateBackup(); err != nil {
		fmt.Printf("Auto-backup failed: %v\n", err)
	} else {
		gl.lastAutoBackup = time.Now()
	}
}

func (gl *GameLoop) checkPetConditions(pet *DigitalPet) {
	// Check for critical conditions and emit events

	// Check if pet died
	if !pet.Biology.IsAlive {
		gl.eventSystem.Emit(CreateEvent(EventPetDied, pet.ID,
			fmt.Sprintf("Pet %s has died", pet.Name)))
		return
	}

	// Check hunger
	if pet.Biology.Vitals.Nutrition < 0.2 {
		gl.eventSystem.Emit(CreateEvent(EventPetHungry, pet.ID,
			fmt.Sprintf("Pet %s is very hungry", pet.Name)))
	}

	// Check sickness
	if pet.Biology.Vitals.Health < 0.3 {
		gl.eventSystem.Emit(CreateEvent(EventPetSick, pet.ID,
			fmt.Sprintf("Pet %s is sick", pet.Name)))
	}

	// Check happiness
	if pet.Biology.Vitals.Happiness > 0.8 {
		gl.eventSystem.Emit(CreateEvent(EventPetHappy, pet.ID,
			fmt.Sprintf("Pet %s is very happy!", pet.Name)))
	} else if pet.Biology.Vitals.Happiness < 0.2 {
		gl.eventSystem.Emit(CreateEvent(EventPetSad, pet.ID,
			fmt.Sprintf("Pet %s is sad", pet.Name)))
	}

	// Check behavior changes
	// (would need to track previous behavior)
}

func (gl *GameLoop) setupEventHandlers() {
	// Register default event handlers
	gl.eventSystem.RegisterHandler(EventPetDied, func(event *GameEvent) {
		// Could trigger achievements, save final state, etc.
		fmt.Printf("[EVENT] %s: %s\n", event.Type.String(), event.Message)
	})

	gl.eventSystem.RegisterHandler(EventAutoSave, func(event *GameEvent) {
		fmt.Printf("[EVENT] %s\n", event.Message)
	})
}

// GetEventSystem returns the event system for external handler registration
func (gl *GameLoop) GetEventSystem() *EventSystem {
	return gl.eventSystem
}

// GetDataManager returns the data manager
func (gl *GameLoop) GetDataManager() *data.DataManager {
	return gl.dataManager
}

// GetTimeManager returns the time manager
func (gl *GameLoop) GetTimeManager() *simulation.TimeManager {
	return gl.timeManager
}
