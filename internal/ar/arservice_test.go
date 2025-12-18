package ar

import (
	"math"
	"testing"
	"time"

	"github.com/Michael-W-Ellison/gochi/pkg/types"
)

// ============================================================================
// Vector3 Tests
// ============================================================================

func TestVector3Add(t *testing.T) {
	v1 := Vector3{X: 1, Y: 2, Z: 3}
	v2 := Vector3{X: 4, Y: 5, Z: 6}

	result := v1.Add(v2)

	if result.X != 5 || result.Y != 7 || result.Z != 9 {
		t.Errorf("Expected {5, 7, 9}, got {%v, %v, %v}", result.X, result.Y, result.Z)
	}
}

func TestVector3Sub(t *testing.T) {
	v1 := Vector3{X: 5, Y: 7, Z: 9}
	v2 := Vector3{X: 1, Y: 2, Z: 3}

	result := v1.Sub(v2)

	if result.X != 4 || result.Y != 5 || result.Z != 6 {
		t.Errorf("Expected {4, 5, 6}, got {%v, %v, %v}", result.X, result.Y, result.Z)
	}
}

func TestVector3Scale(t *testing.T) {
	v := Vector3{X: 2, Y: 3, Z: 4}

	result := v.Scale(2)

	if result.X != 4 || result.Y != 6 || result.Z != 8 {
		t.Errorf("Expected {4, 6, 8}, got {%v, %v, %v}", result.X, result.Y, result.Z)
	}
}

func TestVector3Magnitude(t *testing.T) {
	v := Vector3{X: 3, Y: 4, Z: 0}

	result := v.Magnitude()

	if result != 5 {
		t.Errorf("Expected 5, got %v", result)
	}
}

func TestVector3Normalize(t *testing.T) {
	v := Vector3{X: 3, Y: 4, Z: 0}

	result := v.Normalize()

	if math.Abs(result.X-0.6) > 0.001 || math.Abs(result.Y-0.8) > 0.001 || result.Z != 0 {
		t.Errorf("Expected {0.6, 0.8, 0}, got {%v, %v, %v}", result.X, result.Y, result.Z)
	}
}

func TestVector3NormalizeZero(t *testing.T) {
	v := Vector3{X: 0, Y: 0, Z: 0}

	result := v.Normalize()

	if result.X != 0 || result.Y != 0 || result.Z != 0 {
		t.Errorf("Expected {0, 0, 0}, got {%v, %v, %v}", result.X, result.Y, result.Z)
	}
}

func TestVector3Distance(t *testing.T) {
	v1 := Vector3{X: 0, Y: 0, Z: 0}
	v2 := Vector3{X: 3, Y: 4, Z: 0}

	result := v1.Distance(v2)

	if result != 5 {
		t.Errorf("Expected 5, got %v", result)
	}
}

// ============================================================================
// Transform Tests
// ============================================================================

func TestNewTransform(t *testing.T) {
	transform := NewTransform()

	if transform.Position.X != 0 || transform.Position.Y != 0 || transform.Position.Z != 0 {
		t.Error("Position should be at origin")
	}

	if transform.Scale.X != 1 || transform.Scale.Y != 1 || transform.Scale.Z != 1 {
		t.Error("Scale should be 1, 1, 1")
	}

	if transform.Rotation.W != 1 {
		t.Error("Rotation W should be 1 (identity quaternion)")
	}
}

func TestQuaternionIdentity(t *testing.T) {
	q := QuaternionIdentity()

	if q.X != 0 || q.Y != 0 || q.Z != 0 || q.W != 1 {
		t.Errorf("Expected identity quaternion {0, 0, 0, 1}, got {%v, %v, %v, %v}", q.X, q.Y, q.Z, q.W)
	}
}

// ============================================================================
// DetectedPlane Tests
// ============================================================================

func TestDetectedPlaneArea(t *testing.T) {
	plane := &DetectedPlane{
		ID:     "test-plane",
		Type:   PlaneTypeHorizontal,
		Extent: Vector3{X: 2, Y: 0, Z: 3},
	}

	area := plane.Area()

	if area != 6 {
		t.Errorf("Expected area 6, got %v", area)
	}
}

func TestDetectedPlaneContainsPoint(t *testing.T) {
	plane := &DetectedPlane{
		ID:     "test-plane",
		Type:   PlaneTypeHorizontal,
		Center: Vector3{X: 0, Y: 0, Z: 0},
		Extent: Vector3{X: 2, Y: 0, Z: 2},
	}

	// Point inside
	inside := Vector3{X: 0.5, Y: 0, Z: 0.5}
	if !plane.ContainsPoint(inside) {
		t.Error("Point should be inside plane")
	}

	// Point outside
	outside := Vector3{X: 2, Y: 0, Z: 2}
	if plane.ContainsPoint(outside) {
		t.Error("Point should be outside plane")
	}
}

// ============================================================================
// ARService Tests
// ============================================================================

func TestNewARService(t *testing.T) {
	service := NewARService(nil)

	if service == nil {
		t.Fatal("Service should not be nil")
	}

	if service.State != ARSessionStateInactive {
		t.Errorf("Expected inactive state, got %v", service.State)
	}

	if service.Config == nil {
		t.Error("Config should not be nil")
	}
}

func TestNewARServiceWithConfig(t *testing.T) {
	config := &ARSessionConfig{
		TrackingMode:     ARTrackingModeOrientation,
		PlaneDetection:   false,
		MaxTrackedPlanes: 5,
	}

	service := NewARService(config)

	if service.Config.TrackingMode != ARTrackingModeOrientation {
		t.Errorf("Expected orientation tracking mode, got %v", service.Config.TrackingMode)
	}

	if service.Config.PlaneDetection != false {
		t.Error("Plane detection should be false")
	}
}

func TestDefaultARSessionConfig(t *testing.T) {
	config := DefaultARSessionConfig()

	if config.TrackingMode != ARTrackingModeWorld {
		t.Errorf("Expected world tracking mode, got %v", config.TrackingMode)
	}

	if !config.PlaneDetection {
		t.Error("Plane detection should be enabled by default")
	}

	if !config.LightEstimation {
		t.Error("Light estimation should be enabled by default")
	}
}

func TestStartSession(t *testing.T) {
	service := NewARService(nil)

	err := service.StartSession()
	if err != nil {
		t.Fatalf("StartSession failed: %v", err)
	}

	if service.State != ARSessionStateInitializing {
		t.Errorf("Expected initializing state, got %v", service.State)
	}

	if service.sessionID == "" {
		t.Error("Session ID should be set")
	}
}

func TestStartSessionAlreadyActive(t *testing.T) {
	service := NewARService(nil)

	_ = service.StartSession()
	err := service.StartSession()

	if err == nil {
		t.Error("Expected error when starting already active session")
	}
}

func TestPauseSession(t *testing.T) {
	service := NewARService(nil)
	_ = service.StartSession()

	// Wait for tracking state
	time.Sleep(600 * time.Millisecond)

	service.PauseSession()

	if service.State != ARSessionStatePaused {
		t.Errorf("Expected paused state, got %v", service.State)
	}
}

func TestResumeSession(t *testing.T) {
	service := NewARService(nil)
	_ = service.StartSession()

	time.Sleep(600 * time.Millisecond)

	service.PauseSession()
	service.ResumeSession()

	if service.State != ARSessionStateTracking {
		t.Errorf("Expected tracking state, got %v", service.State)
	}
}

func TestStopSession(t *testing.T) {
	service := NewARService(nil)
	_ = service.StartSession()

	service.StopSession()

	if service.State != ARSessionStateInactive {
		t.Errorf("Expected inactive state, got %v", service.State)
	}

	if service.sessionID != "" {
		t.Error("Session ID should be cleared")
	}
}

// ============================================================================
// Plane Detection Tests
// ============================================================================

func TestAddDetectedPlane(t *testing.T) {
	service := NewARService(nil)

	plane := &DetectedPlane{
		ID:     "plane-1",
		Type:   PlaneTypeHorizontal,
		Center: Vector3{X: 0, Y: 0, Z: 0},
		Extent: Vector3{X: 1, Y: 0, Z: 1},
	}

	service.AddDetectedPlane(plane)

	if len(service.DetectedPlanes) != 1 {
		t.Errorf("Expected 1 plane, got %d", len(service.DetectedPlanes))
	}
}

func TestAddDetectedPlaneTooSmall(t *testing.T) {
	service := NewARService(nil)

	plane := &DetectedPlane{
		ID:     "small-plane",
		Type:   PlaneTypeHorizontal,
		Extent: Vector3{X: 0.1, Y: 0, Z: 0.1}, // 0.01 sq meters, below default threshold
	}

	service.AddDetectedPlane(plane)

	if len(service.DetectedPlanes) != 0 {
		t.Error("Small plane should not be added")
	}
}

func TestMaxTrackedPlanes(t *testing.T) {
	config := &ARSessionConfig{
		MaxTrackedPlanes: 2,
		MinPlaneArea:     0.01,
	}
	service := NewARService(config)

	for i := 0; i < 5; i++ {
		plane := &DetectedPlane{
			ID:     string(rune('a' + i)),
			Type:   PlaneTypeHorizontal,
			Extent: Vector3{X: 1, Y: 0, Z: 1},
		}
		service.AddDetectedPlane(plane)
	}

	if len(service.DetectedPlanes) > 2 {
		t.Errorf("Expected max 2 planes, got %d", len(service.DetectedPlanes))
	}
}

func TestUpdateDetectedPlane(t *testing.T) {
	service := NewARService(nil)

	plane := &DetectedPlane{
		ID:     "plane-1",
		Type:   PlaneTypeHorizontal,
		Center: Vector3{X: 0, Y: 0, Z: 0},
		Extent: Vector3{X: 1, Y: 0, Z: 1},
	}
	service.AddDetectedPlane(plane)

	// Update plane
	updatedPlane := &DetectedPlane{
		ID:     "plane-1",
		Type:   PlaneTypeHorizontal,
		Center: Vector3{X: 1, Y: 0, Z: 1},
		Extent: Vector3{X: 2, Y: 0, Z: 2},
	}
	service.UpdateDetectedPlane(updatedPlane)

	stored := service.DetectedPlanes["plane-1"]
	if stored.Center.X != 1 {
		t.Error("Plane should be updated")
	}
}

func TestRemoveDetectedPlane(t *testing.T) {
	service := NewARService(nil)

	plane := &DetectedPlane{
		ID:     "plane-1",
		Type:   PlaneTypeHorizontal,
		Extent: Vector3{X: 1, Y: 0, Z: 1},
	}
	service.AddDetectedPlane(plane)

	service.RemoveDetectedPlane("plane-1")

	if len(service.DetectedPlanes) != 0 {
		t.Error("Plane should be removed")
	}
}

func TestGetLargestHorizontalPlane(t *testing.T) {
	service := NewARService(nil)
	service.Config.MinPlaneArea = 0.01

	// Add planes of different sizes
	small := &DetectedPlane{
		ID:     "small",
		Type:   PlaneTypeHorizontal,
		Extent: Vector3{X: 1, Y: 0, Z: 1},
	}
	large := &DetectedPlane{
		ID:     "large",
		Type:   PlaneTypeHorizontal,
		Extent: Vector3{X: 3, Y: 0, Z: 3},
	}
	vertical := &DetectedPlane{
		ID:     "vertical",
		Type:   PlaneTypeVertical,
		Extent: Vector3{X: 5, Y: 5, Z: 0},
	}

	service.AddDetectedPlane(small)
	service.AddDetectedPlane(large)
	service.AddDetectedPlane(vertical)

	largest := service.GetLargestHorizontalPlane()

	if largest == nil || largest.ID != "large" {
		t.Error("Should return largest horizontal plane")
	}
}

// ============================================================================
// Pet Placement Tests
// ============================================================================

func TestPlacePet(t *testing.T) {
	service := NewARService(nil)
	_ = service.StartSession()
	time.Sleep(600 * time.Millisecond) // Wait for tracking

	petID := types.PetID("pet-123")
	position := Vector3{X: 0, Y: 0, Z: -1}

	arPet, err := service.PlacePet(petID, position, "")
	if err != nil {
		t.Fatalf("PlacePet failed: %v", err)
	}

	if arPet == nil {
		t.Fatal("AR pet should not be nil")
	}

	if arPet.PetID != petID {
		t.Errorf("Expected pet ID %s, got %s", petID, arPet.PetID)
	}

	if arPet.Transform.Position != position {
		t.Error("Pet position should match")
	}

	if !arPet.IsVisible {
		t.Error("Pet should be visible")
	}
}

func TestPlacePetNotTracking(t *testing.T) {
	service := NewARService(nil)
	// Don't start session

	petID := types.PetID("pet-123")
	position := Vector3{X: 0, Y: 0, Z: -1}

	_, err := service.PlacePet(petID, position, "")
	if err == nil {
		t.Error("Expected error when not tracking")
	}
}

func TestRemovePet(t *testing.T) {
	service := NewARService(nil)
	_ = service.StartSession()
	time.Sleep(600 * time.Millisecond)

	petID := types.PetID("pet-123")
	service.PlacePet(petID, Vector3{}, "")

	service.RemovePet(petID)

	if service.GetPlacedPet(petID) != nil {
		t.Error("Pet should be removed")
	}
}

func TestMovePetTo(t *testing.T) {
	service := NewARService(nil)
	_ = service.StartSession()
	time.Sleep(600 * time.Millisecond)

	petID := types.PetID("pet-123")
	service.PlacePet(petID, Vector3{X: 0, Y: 0, Z: 0}, "")

	target := Vector3{X: 1, Y: 0, Z: 1}
	err := service.MovePetTo(petID, target)
	if err != nil {
		t.Fatalf("MovePetTo failed: %v", err)
	}

	pet := service.GetPlacedPet(petID)
	if !pet.IsMoving {
		t.Error("Pet should be moving")
	}

	if pet.TargetPoint == nil {
		t.Error("Target point should be set")
	}

	if pet.Animation != "walking" {
		t.Errorf("Expected walking animation, got %s", pet.Animation)
	}
}

func TestSetPetAnimation(t *testing.T) {
	service := NewARService(nil)
	_ = service.StartSession()
	time.Sleep(600 * time.Millisecond)

	petID := types.PetID("pet-123")
	service.PlacePet(petID, Vector3{}, "")

	err := service.SetPetAnimation(petID, "happy")
	if err != nil {
		t.Fatalf("SetPetAnimation failed: %v", err)
	}

	pet := service.GetPlacedPet(petID)
	if pet.Animation != "happy" {
		t.Errorf("Expected happy animation, got %s", pet.Animation)
	}
}

func TestSetPetBehavior(t *testing.T) {
	service := NewARService(nil)
	_ = service.StartSession()
	time.Sleep(600 * time.Millisecond)

	petID := types.PetID("pet-123")
	service.PlacePet(petID, Vector3{}, "")

	err := service.SetPetBehavior(petID, "playing")
	if err != nil {
		t.Fatalf("SetPetBehavior failed: %v", err)
	}

	pet := service.GetPlacedPet(petID)
	if pet.Behavior != "playing" {
		t.Errorf("Expected playing behavior, got %s", pet.Behavior)
	}

	if pet.Animation != "play" {
		t.Errorf("Expected play animation, got %s", pet.Animation)
	}
}

// ============================================================================
// Exercise Tracker Tests
// ============================================================================

func TestNewExerciseTracker(t *testing.T) {
	tracker := NewExerciseTracker()

	if tracker == nil {
		t.Fatal("Tracker should not be nil")
	}

	if tracker.IsTracking {
		t.Error("Tracker should not be tracking initially")
	}
}

func TestExerciseTrackerStartStop(t *testing.T) {
	tracker := NewExerciseTracker()

	tracker.StartTracking()

	if !tracker.IsTracking {
		t.Error("Tracker should be tracking")
	}

	if tracker.StartTime.IsZero() {
		t.Error("Start time should be set")
	}

	// Small delay to ensure duration > 0
	time.Sleep(10 * time.Millisecond)

	result := tracker.StopTracking()

	if tracker.IsTracking {
		t.Error("Tracker should not be tracking")
	}

	if result.Duration == 0 {
		t.Error("Duration should be recorded")
	}
}

func TestExerciseTrackerUpdatePosition(t *testing.T) {
	tracker := NewExerciseTracker()
	tracker.StartTracking()

	// First position
	tracker.UpdatePosition(Vector3{X: 0, Y: 0, Z: 0})

	// Move 1 meter
	tracker.UpdatePosition(Vector3{X: 1, Y: 0, Z: 0})

	stats := tracker.GetExerciseStats()

	if stats.TotalDistance < 0.9 || stats.TotalDistance > 1.1 {
		t.Errorf("Expected ~1m distance, got %v", stats.TotalDistance)
	}
}

func TestExerciseTrackerStepCounting(t *testing.T) {
	tracker := NewExerciseTracker()
	tracker.StepThreshold = 0.7 // 70cm per step
	tracker.StartTracking()

	// Initial position
	tracker.UpdatePosition(Vector3{X: 0, Y: 0, Z: 0})

	// Walk 2.1 meters (should be ~3 steps)
	tracker.UpdatePosition(Vector3{X: 2.1, Y: 0, Z: 0})

	stats := tracker.GetExerciseStats()

	if stats.TotalSteps != 3 {
		t.Errorf("Expected 3 steps, got %d", stats.TotalSteps)
	}
}

func TestExerciseTrackerCalories(t *testing.T) {
	tracker := NewExerciseTracker()
	tracker.StepThreshold = 0.7
	tracker.StartTracking()

	tracker.UpdatePosition(Vector3{X: 0, Y: 0, Z: 0})
	tracker.UpdatePosition(Vector3{X: 7, Y: 0, Z: 0}) // 10 steps

	stats := tracker.GetExerciseStats()

	expectedCalories := float64(stats.TotalSteps) * 0.04

	if math.Abs(stats.CaloriesBurned-expectedCalories) > 0.01 {
		t.Errorf("Expected ~%v calories, got %v", expectedCalories, stats.CaloriesBurned)
	}
}

func TestExerciseTrackerFilterNoise(t *testing.T) {
	tracker := NewExerciseTracker()
	tracker.StartTracking()

	// Initial position
	tracker.UpdatePosition(Vector3{X: 0, Y: 0, Z: 0})

	// Small movement (noise) - should be filtered
	tracker.UpdatePosition(Vector3{X: 0.05, Y: 0, Z: 0})

	stats := tracker.GetExerciseStats()

	if stats.TotalDistance > 0 {
		t.Error("Small movements should be filtered as noise")
	}
}

// ============================================================================
// AR Snapshot Tests
// ============================================================================

func TestGetSnapshot(t *testing.T) {
	service := NewARService(nil)
	_ = service.StartSession()
	time.Sleep(600 * time.Millisecond)

	// Add some data
	service.Config.MinPlaneArea = 0.01
	service.AddDetectedPlane(&DetectedPlane{
		ID:     "plane-1",
		Type:   PlaneTypeHorizontal,
		Extent: Vector3{X: 1, Y: 0, Z: 1},
	})

	petID := types.PetID("pet-123")
	service.PlacePet(petID, Vector3{}, "")

	snapshot := service.GetSnapshot()

	if snapshot == nil {
		t.Fatal("Snapshot should not be nil")
	}

	if snapshot.SessionState != ARSessionStateTracking {
		t.Errorf("Expected tracking state, got %v", snapshot.SessionState)
	}

	if len(snapshot.DetectedPlanes) != 1 {
		t.Errorf("Expected 1 plane, got %d", len(snapshot.DetectedPlanes))
	}

	if len(snapshot.PlacedPets) != 1 {
		t.Errorf("Expected 1 pet, got %d", len(snapshot.PlacedPets))
	}

	if snapshot.Timestamp.IsZero() {
		t.Error("Timestamp should be set")
	}
}

// ============================================================================
// Camera and Lighting Tests
// ============================================================================

func TestUpdateCameraTransform(t *testing.T) {
	service := NewARService(nil)

	transform := Transform{
		Position: Vector3{X: 1, Y: 2, Z: 3},
		Rotation: QuaternionIdentity(),
		Scale:    Vector3{X: 1, Y: 1, Z: 1},
	}

	service.UpdateCameraTransform(transform)

	result := service.GetCameraTransform()
	if result.Position.X != 1 || result.Position.Y != 2 || result.Position.Z != 3 {
		t.Error("Camera transform should be updated")
	}
}

func TestUpdateLightEstimate(t *testing.T) {
	service := NewARService(nil)

	estimate := &LightEstimate{
		AmbientIntensity: 1000,
		AmbientColorTemp: 6500,
	}

	service.UpdateLightEstimate(estimate)

	result := service.GetLightEstimate()
	if result == nil {
		t.Fatal("Light estimate should not be nil")
	}

	if result.AmbientIntensity != 1000 {
		t.Errorf("Expected intensity 1000, got %v", result.AmbientIntensity)
	}
}

// ============================================================================
// Callback Tests
// ============================================================================

func TestPlaneDetectedCallback(t *testing.T) {
	service := NewARService(nil)
	service.Config.MinPlaneArea = 0.01

	callbackCalled := false
	service.OnPlaneDetected = func(plane *DetectedPlane) {
		callbackCalled = true
	}

	service.AddDetectedPlane(&DetectedPlane{
		ID:     "test",
		Type:   PlaneTypeHorizontal,
		Extent: Vector3{X: 1, Y: 0, Z: 1},
	})

	if !callbackCalled {
		t.Error("OnPlaneDetected callback should be called")
	}
}

func TestTrackingStateChangeCallback(t *testing.T) {
	service := NewARService(nil)

	states := []ARSessionState{}
	service.OnTrackingStateChange = func(state ARSessionState) {
		states = append(states, state)
	}

	_ = service.StartSession()
	time.Sleep(600 * time.Millisecond)
	service.StopSession()

	if len(states) < 2 {
		t.Errorf("Expected at least 2 state changes, got %d", len(states))
	}
}
