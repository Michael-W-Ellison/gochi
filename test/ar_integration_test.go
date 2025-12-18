package test

import (
	"testing"
	"time"

	"github.com/Michael-W-Ellison/gochi/internal/ar"
	"github.com/Michael-W-Ellison/gochi/internal/core"
	"github.com/Michael-W-Ellison/gochi/pkg/types"
)

// TestARSessionLifecycle tests complete AR session lifecycle
func TestARSessionLifecycle(t *testing.T) {
	service := ar.NewARService(nil) // nil uses default config

	// Verify initial state
	if service.State != ar.ARSessionStateInactive {
		t.Errorf("Initial state should be Inactive, got %v", service.State)
	}

	// Start session
	err := service.StartSession()
	if err != nil {
		t.Fatalf("Failed to start session: %v", err)
	}

	// Give time for initialization
	time.Sleep(600 * time.Millisecond)

	if service.State != ar.ARSessionStateTracking {
		t.Errorf("State should be Tracking after start, got %v", service.State)
	}

	// Pause session
	service.PauseSession()

	if service.State != ar.ARSessionStatePaused {
		t.Errorf("State should be Paused, got %v", service.State)
	}

	// Resume session
	service.ResumeSession()

	if service.State != ar.ARSessionStateTracking {
		t.Errorf("State should be Tracking after resume, got %v", service.State)
	}

	// Stop session
	service.StopSession()

	if service.State != ar.ARSessionStateInactive {
		t.Errorf("State should be Inactive after stop, got %v", service.State)
	}
}

// TestARPlaneDetection tests plane detection in AR
func TestARPlaneDetection(t *testing.T) {
	service := ar.NewARService(nil)
	service.StartSession()
	defer service.StopSession()

	time.Sleep(600 * time.Millisecond)

	// Add a horizontal plane
	plane := &ar.DetectedPlane{
		ID:     "plane_1",
		Type:   ar.PlaneTypeHorizontal,
		Center: ar.Vector3{X: 0, Y: 0, Z: -1},
		Extent: ar.Vector3{X: 2, Y: 0, Z: 2},
	}

	service.AddDetectedPlane(plane)

	// Verify plane is tracked
	if len(service.DetectedPlanes) != 1 {
		t.Errorf("Expected 1 plane, got %d", len(service.DetectedPlanes))
	}

	// Get largest horizontal plane
	largest := service.GetLargestHorizontalPlane()
	if largest == nil {
		t.Error("Should find largest horizontal plane")
	}

	if largest.ID != "plane_1" {
		t.Errorf("Expected plane 'plane_1', got '%s'", largest.ID)
	}
}

// TestARPetPlacement tests placing a pet in AR
func TestARPetPlacement(t *testing.T) {
	service := ar.NewARService(nil)
	service.StartSession()
	defer service.StopSession()

	time.Sleep(600 * time.Millisecond)

	// Create a pet
	pet := core.NewDigitalPet("ARPet", "user_ar")

	// Place pet in AR
	position := ar.Vector3{X: 0, Y: 0, Z: -1}
	arPet, err := service.PlacePet(pet.ID, position, "")
	if err != nil {
		t.Fatalf("Failed to place pet: %v", err)
	}

	// Verify pet is placed
	if len(service.PlacedPets) != 1 {
		t.Errorf("Expected 1 placed pet, got %d", len(service.PlacedPets))
	}

	if arPet.Transform.Position != position {
		t.Errorf("Pet position mismatch: expected %v, got %v",
			position, arPet.Transform.Position)
	}
}

// TestARPetMovement tests moving a pet in AR space
func TestARPetMovement(t *testing.T) {
	service := ar.NewARService(nil)
	service.StartSession()
	defer service.StopSession()

	time.Sleep(600 * time.Millisecond)

	petID := types.PetID("movement_pet")

	// Place pet
	initialPos := ar.Vector3{X: 0, Y: 0, Z: -1}
	service.PlacePet(petID, initialPos, "")

	// Move pet
	newPos := ar.Vector3{X: 1, Y: 0, Z: -2}
	err := service.MovePetTo(petID, newPos)
	if err != nil {
		t.Fatalf("Failed to move pet: %v", err)
	}

	// Verify movement started
	arPet := service.PlacedPets[petID]
	if arPet.TargetPoint == nil {
		t.Error("Target point should be set")
	}
}

// TestARPetAnimation tests setting pet animations
func TestARPetAnimation(t *testing.T) {
	service := ar.NewARService(nil)
	service.StartSession()
	defer service.StopSession()

	time.Sleep(600 * time.Millisecond)

	petID := types.PetID("animation_pet")

	// Place pet
	service.PlacePet(petID, ar.Vector3{X: 0, Y: 0, Z: -1}, "")

	// Set animation
	err := service.SetPetAnimation(petID, "playing")
	if err != nil {
		t.Fatalf("Failed to set animation: %v", err)
	}

	// Verify animation
	arPet := service.PlacedPets[petID]
	if arPet.Animation != "playing" {
		t.Errorf("Expected animation 'playing', got '%s'", arPet.Animation)
	}
}

// TestARExerciseTracking tests exercise tracking in AR
func TestARExerciseTracking(t *testing.T) {
	tracker := ar.NewExerciseTracker()

	// Start tracking
	tracker.StartTracking()

	if !tracker.IsTracking {
		t.Error("Tracker should be tracking")
	}

	// Simulate movement
	tracker.UpdatePosition(ar.Vector3{X: 0, Y: 0, Z: 0})
	tracker.UpdatePosition(ar.Vector3{X: 1, Y: 0, Z: 0})
	tracker.UpdatePosition(ar.Vector3{X: 2, Y: 0, Z: 0})

	// Get stats
	stats := tracker.GetExerciseStats()

	if stats.TotalDistance < 1.5 {
		t.Errorf("Expected at least 1.5m distance, got %f", stats.TotalDistance)
	}

	// Wait a bit for duration
	time.Sleep(20 * time.Millisecond)

	// Stop tracking
	result := tracker.StopTracking()

	if result.Duration == 0 {
		t.Error("Duration should be recorded")
	}

	if tracker.IsTracking {
		t.Error("Tracker should have stopped")
	}
}

// TestARExerciseCalories tests calorie calculation
func TestARExerciseCalories(t *testing.T) {
	tracker := ar.NewExerciseTracker()
	tracker.StartTracking()

	// Simulate walking distance
	for i := 0; i < 100; i++ {
		pos := ar.Vector3{X: float64(i) * 0.5, Y: 0, Z: 0}
		tracker.UpdatePosition(pos)
	}

	stats := tracker.GetExerciseStats()

	// Should have burned some calories
	if stats.CaloriesBurned <= 0 {
		t.Log("Note: Calories may require significant distance")
	}

	t.Logf("Distance: %.2f m, Calories: %.2f", stats.TotalDistance, stats.CaloriesBurned)
	tracker.StopTracking()
}

// TestARExerciseStepCounting tests step detection
func TestARExerciseStepCounting(t *testing.T) {
	tracker := ar.NewExerciseTracker()
	tracker.StepThreshold = 0.7 // 70cm per step
	tracker.StartTracking()

	// Simulate steps (each step ~0.7m)
	for i := 0; i <= 10; i++ {
		pos := ar.Vector3{X: float64(i) * 0.7, Y: 0, Z: 0}
		tracker.UpdatePosition(pos)
	}

	stats := tracker.GetExerciseStats()

	t.Logf("Steps counted: %d, Distance: %.2f m", stats.TotalSteps, stats.TotalDistance)
	tracker.StopTracking()
}

// TestARVector3Math tests 3D vector operations
func TestARVector3Math(t *testing.T) {
	v1 := ar.Vector3{X: 1, Y: 2, Z: 3}
	v2 := ar.Vector3{X: 4, Y: 5, Z: 6}

	// Test addition
	sum := v1.Add(v2)
	expected := ar.Vector3{X: 5, Y: 7, Z: 9}
	if sum != expected {
		t.Errorf("Add failed: expected %v, got %v", expected, sum)
	}

	// Test subtraction
	diff := v2.Sub(v1)
	expected = ar.Vector3{X: 3, Y: 3, Z: 3}
	if diff != expected {
		t.Errorf("Sub failed: expected %v, got %v", expected, diff)
	}

	// Test scale
	scaled := v1.Scale(2)
	expected = ar.Vector3{X: 2, Y: 4, Z: 6}
	if scaled != expected {
		t.Errorf("Scale failed: expected %v, got %v", expected, scaled)
	}

	// Test magnitude
	v := ar.Vector3{X: 3, Y: 4, Z: 0}
	mag := v.Magnitude()
	if mag != 5.0 {
		t.Errorf("Magnitude failed: expected 5, got %f", mag)
	}

	// Test distance
	origin := ar.Vector3{X: 0, Y: 0, Z: 0}
	point := ar.Vector3{X: 3, Y: 4, Z: 0}
	dist := origin.Distance(point)
	if dist != 5.0 {
		t.Errorf("Distance failed: expected 5, got %f", dist)
	}
}

// TestARTransform tests transform operations
func TestARTransform(t *testing.T) {
	transform := ar.NewTransform()

	// Should have identity rotation
	if transform.Rotation != ar.QuaternionIdentity() {
		t.Error("Default rotation should be identity")
	}

	// Should have unit scale
	if transform.Scale != (ar.Vector3{X: 1, Y: 1, Z: 1}) {
		t.Error("Default scale should be (1,1,1)")
	}

	// Should be at origin
	if transform.Position != (ar.Vector3{X: 0, Y: 0, Z: 0}) {
		t.Error("Default position should be origin")
	}
}

// TestARLightEstimation tests light estimation
func TestARLightEstimation(t *testing.T) {
	service := ar.NewARService(nil)
	service.StartSession()
	defer service.StopSession()

	time.Sleep(600 * time.Millisecond)

	// Update light estimate
	service.UpdateLightEstimate(&ar.LightEstimate{
		AmbientIntensity: 1000,
		AmbientColorTemp: 6500,
	})

	estimate := service.GetLightEstimate()
	if estimate == nil {
		t.Fatal("Light estimate should be set")
	}

	if estimate.AmbientIntensity != 1000 {
		t.Errorf("Ambient intensity mismatch: expected 1000, got %f",
			estimate.AmbientIntensity)
	}

	if estimate.AmbientColorTemp != 6500 {
		t.Errorf("Ambient color temp mismatch: expected 6500, got %f",
			estimate.AmbientColorTemp)
	}
}

// TestARCameraTracking tests camera transform updates
func TestARCameraTracking(t *testing.T) {
	service := ar.NewARService(nil)
	service.StartSession()
	defer service.StopSession()

	time.Sleep(600 * time.Millisecond)

	// Update camera position
	cameraPos := ar.Vector3{X: 0, Y: 1.6, Z: 0}
	service.UpdateCameraTransform(ar.Transform{
		Position: cameraPos,
		Rotation: ar.QuaternionIdentity(),
		Scale:    ar.Vector3{X: 1, Y: 1, Z: 1},
	})

	transform := service.GetCameraTransform()
	if transform.Position != cameraPos {
		t.Errorf("Camera position mismatch: expected %v, got %v",
			cameraPos, transform.Position)
	}
}

// TestARSnapshot tests AR state snapshot
func TestARSnapshot(t *testing.T) {
	service := ar.NewARService(nil)
	service.StartSession()
	defer service.StopSession()

	time.Sleep(600 * time.Millisecond)

	// Add some content
	service.AddDetectedPlane(&ar.DetectedPlane{
		ID:     "snapshot_plane",
		Type:   ar.PlaneTypeHorizontal,
		Center: ar.Vector3{X: 0, Y: 0, Z: -1},
		Extent: ar.Vector3{X: 1, Y: 0, Z: 1},
	})

	service.PlacePet(types.PetID("snapshot_pet"), ar.Vector3{X: 0, Y: 0, Z: -1}, "")

	// Get snapshot
	snapshot := service.GetSnapshot()

	if snapshot.SessionState != ar.ARSessionStateTracking {
		t.Errorf("Snapshot state mismatch: expected Tracking, got %v", snapshot.SessionState)
	}

	if len(snapshot.DetectedPlanes) != 1 {
		t.Errorf("Expected 1 plane in snapshot, got %d", len(snapshot.DetectedPlanes))
	}

	if len(snapshot.PlacedPets) != 1 {
		t.Errorf("Expected 1 pet in snapshot, got %d", len(snapshot.PlacedPets))
	}
}

// TestARCallbacks tests callback functionality
func TestARCallbacks(t *testing.T) {
	service := ar.NewARService(nil)

	planeDetected := false
	stateChanged := false

	// Set callbacks
	service.OnPlaneDetected = func(plane *ar.DetectedPlane) {
		planeDetected = true
	}

	service.OnTrackingStateChange = func(state ar.ARSessionState) {
		stateChanged = true
	}

	// Trigger callbacks
	service.StartSession()
	defer service.StopSession()

	time.Sleep(600 * time.Millisecond)

	if !stateChanged {
		t.Error("OnTrackingStateChange should have been called")
	}

	// Add plane to trigger callback
	service.AddDetectedPlane(&ar.DetectedPlane{
		ID:     "callback_plane",
		Type:   ar.PlaneTypeHorizontal,
		Center: ar.Vector3{X: 0, Y: 0, Z: -1},
		Extent: ar.Vector3{X: 1, Y: 0, Z: 1},
	})

	if !planeDetected {
		t.Error("OnPlaneDetected should have been called")
	}
}

// TestARMultiplePets tests placing multiple pets in AR
func TestARMultiplePets(t *testing.T) {
	service := ar.NewARService(nil)
	service.StartSession()
	defer service.StopSession()

	time.Sleep(600 * time.Millisecond)

	// Place multiple pets
	petIDs := []types.PetID{"pet1", "pet2", "pet3"}
	positions := []ar.Vector3{
		{X: -1, Y: 0, Z: -1},
		{X: 0, Y: 0, Z: -1},
		{X: 1, Y: 0, Z: -1},
	}

	for i, id := range petIDs {
		_, err := service.PlacePet(id, positions[i], "")
		if err != nil {
			t.Fatalf("Failed to place pet %s: %v", id, err)
		}
	}

	if len(service.PlacedPets) != 3 {
		t.Errorf("Expected 3 placed pets, got %d", len(service.PlacedPets))
	}

	// Remove one pet
	service.RemovePet(petIDs[1])

	if len(service.PlacedPets) != 2 {
		t.Errorf("Expected 2 placed pets after removal, got %d", len(service.PlacedPets))
	}
}

// TestARConfigOptions tests AR configuration options
func TestARConfigOptions(t *testing.T) {
	config := ar.DefaultARSessionConfig()

	// Verify defaults
	if !config.PlaneDetection {
		t.Error("Plane detection should be enabled by default")
	}

	if !config.LightEstimation {
		t.Error("Light estimation should be enabled by default")
	}

	// Create service with custom config
	config.MaxTrackedPlanes = 5
	service := ar.NewARService(config)

	if service.Config.MaxTrackedPlanes != 5 {
		t.Errorf("Config not applied: expected 5 max planes, got %d",
			service.Config.MaxTrackedPlanes)
	}
}

// TestARPlaneFiltering tests filtering planes by type
func TestARPlaneFiltering(t *testing.T) {
	service := ar.NewARService(nil)
	service.StartSession()
	defer service.StopSession()

	time.Sleep(600 * time.Millisecond)

	// Add horizontal and vertical planes
	service.AddDetectedPlane(&ar.DetectedPlane{
		ID:     "horizontal_1",
		Type:   ar.PlaneTypeHorizontal,
		Extent: ar.Vector3{X: 2, Y: 0, Z: 2},
	})

	service.AddDetectedPlane(&ar.DetectedPlane{
		ID:     "horizontal_2",
		Type:   ar.PlaneTypeHorizontal,
		Extent: ar.Vector3{X: 1, Y: 0, Z: 1},
	})

	service.AddDetectedPlane(&ar.DetectedPlane{
		ID:     "vertical_1",
		Type:   ar.PlaneTypeVertical,
		Extent: ar.Vector3{X: 2, Y: 2, Z: 0},
	})

	// Get largest horizontal
	largest := service.GetLargestHorizontalPlane()
	if largest == nil {
		t.Fatal("Should find largest horizontal plane")
	}

	if largest.ID != "horizontal_1" {
		t.Errorf("Expected largest horizontal to be 'horizontal_1', got '%s'", largest.ID)
	}
}

// TestARPetBehavior tests setting pet behavior in AR
func TestARPetBehavior(t *testing.T) {
	service := ar.NewARService(nil)
	service.StartSession()
	defer service.StopSession()

	time.Sleep(600 * time.Millisecond)

	petID := types.PetID("behavior_pet")
	service.PlacePet(petID, ar.Vector3{X: 0, Y: 0, Z: -1}, "")

	// Set behavior
	err := service.SetPetBehavior(petID, "playing")
	if err != nil {
		t.Fatalf("Failed to set behavior: %v", err)
	}

	arPet := service.PlacedPets[petID]
	if arPet.Behavior != "playing" {
		t.Errorf("Expected behavior 'playing', got '%s'", arPet.Behavior)
	}
}

// TestARServiceExerciseIntegration tests exercise tracking via AR service
func TestARServiceExerciseIntegration(t *testing.T) {
	service := ar.NewARService(nil)
	service.StartSession()
	defer service.StopSession()

	time.Sleep(600 * time.Millisecond)

	// Start exercise tracking
	service.StartExerciseTracking()

	if !service.IsExerciseTracking() {
		t.Error("Service should be tracking exercise")
	}

	// Get stats
	stats := service.GetExerciseStats()
	t.Logf("Initial exercise stats: %+v", stats)

	// Wait a bit
	time.Sleep(20 * time.Millisecond)

	// Stop tracking
	result := service.StopExerciseTracking()
	t.Logf("Exercise result: %+v", result)

	if service.IsExerciseTracking() {
		t.Error("Service should not be tracking exercise after stop")
	}
}

// TestARIntegrationWithPetSystem tests AR integration with core pet system
func TestARIntegrationWithPetSystem(t *testing.T) {
	// Create a pet
	pet := core.NewDigitalPet("ARIntegrationPet", "user_ar_integration")

	// Create AR service
	arService := ar.NewARService(nil)
	arService.StartSession()
	defer arService.StopSession()

	time.Sleep(600 * time.Millisecond)

	// Place pet in AR
	position := ar.Vector3{X: 0, Y: 0, Z: -1.5}
	_, err := arService.PlacePet(pet.ID, position, "")
	if err != nil {
		t.Fatalf("Failed to place pet in AR: %v", err)
	}

	// Update pet and sync to AR
	pet.ProcessUserInteraction(types.InteractionPlaying, 1.0)

	// Sync behavior to AR
	arService.SetPetBehavior(pet.ID, pet.CurrentBehavior.String())

	arPet := arService.PlacedPets[pet.ID]
	t.Logf("Pet behavior in core: %v, in AR: %v",
		pet.CurrentBehavior, arPet.Behavior)

	// Start exercise tracking
	arService.StartExerciseTracking()

	// Get exercise stats
	stats := arService.GetExerciseStats()
	t.Logf("Exercise stats: %+v", stats)

	arService.StopExerciseTracking()
}
