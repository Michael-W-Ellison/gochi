package ar

import (
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/Michael-W-Ellison/gochi/pkg/types"
)

// ============================================================================
// Types and Constants
// ============================================================================

// ARSessionState represents the state of an AR session
type ARSessionState string

const (
	ARSessionStateInactive     ARSessionState = "inactive"
	ARSessionStateInitializing ARSessionState = "initializing"
	ARSessionStateTracking     ARSessionState = "tracking"
	ARSessionStateLimited      ARSessionState = "limited"
	ARSessionStatePaused       ARSessionState = "paused"
	ARSessionStateError        ARSessionState = "error"
)

// ARTrackingMode represents the tracking mode
type ARTrackingMode string

const (
	ARTrackingModeWorld       ARTrackingMode = "world"        // Full 6DOF tracking
	ARTrackingModeOrientation ARTrackingMode = "orientation"  // 3DOF orientation only
	ARTrackingModeFace        ARTrackingMode = "face"         // Face tracking
	ARTrackingModeImage       ARTrackingMode = "image"        // Image/marker tracking
	ARTrackingModeNone        ARTrackingMode = "none"         // 2D overlay mode
)

// PlaneType represents detected plane types
type PlaneType string

const (
	PlaneTypeHorizontal PlaneType = "horizontal"
	PlaneTypeVertical   PlaneType = "vertical"
	PlaneTypeUnknown    PlaneType = "unknown"
)

// GestureType represents AR gesture types
type GestureType string

const (
	GestureTypeTap       GestureType = "tap"
	GestureTypeDoubleTap GestureType = "double_tap"
	GestureTypeLongPress GestureType = "long_press"
	GestureTypePan       GestureType = "pan"
	GestureTypePinch     GestureType = "pinch"
	GestureTypeRotate    GestureType = "rotate"
	GestureTypeSwipe     GestureType = "swipe"
)

// InteractionType represents AR interaction types
type InteractionType string

const (
	InteractionTypePet       InteractionType = "pet"
	InteractionTypeThrowBall InteractionType = "throw_ball"
	InteractionTypeFeed      InteractionType = "feed"
	InteractionTypePlay      InteractionType = "play"
	InteractionTypeCall      InteractionType = "call"
	InteractionTypePoint     InteractionType = "point"
)

// ============================================================================
// Core Structures
// ============================================================================

// Vector3 represents a 3D position or direction
type Vector3 struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
	Z float64 `json:"z"`
}

// Add adds two vectors
func (v Vector3) Add(other Vector3) Vector3 {
	return Vector3{X: v.X + other.X, Y: v.Y + other.Y, Z: v.Z + other.Z}
}

// Sub subtracts two vectors
func (v Vector3) Sub(other Vector3) Vector3 {
	return Vector3{X: v.X - other.X, Y: v.Y - other.Y, Z: v.Z - other.Z}
}

// Scale scales a vector by a scalar
func (v Vector3) Scale(s float64) Vector3 {
	return Vector3{X: v.X * s, Y: v.Y * s, Z: v.Z * s}
}

// Magnitude returns the length of the vector
func (v Vector3) Magnitude() float64 {
	return math.Sqrt(v.X*v.X + v.Y*v.Y + v.Z*v.Z)
}

// Normalize returns a unit vector
func (v Vector3) Normalize() Vector3 {
	mag := v.Magnitude()
	if mag == 0 {
		return Vector3{}
	}
	return v.Scale(1 / mag)
}

// Distance calculates distance to another point
func (v Vector3) Distance(other Vector3) float64 {
	return v.Sub(other).Magnitude()
}

// Quaternion represents a rotation in 3D space
type Quaternion struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
	Z float64 `json:"z"`
	W float64 `json:"w"`
}

// Identity returns an identity quaternion (no rotation)
func QuaternionIdentity() Quaternion {
	return Quaternion{X: 0, Y: 0, Z: 0, W: 1}
}

// Transform represents a 3D transform (position and rotation)
type Transform struct {
	Position Vector3    `json:"position"`
	Rotation Quaternion `json:"rotation"`
	Scale    Vector3    `json:"scale"`
}

// NewTransform creates a new transform at origin
func NewTransform() Transform {
	return Transform{
		Position: Vector3{},
		Rotation: QuaternionIdentity(),
		Scale:    Vector3{X: 1, Y: 1, Z: 1},
	}
}

// DetectedPlane represents a detected surface in AR
type DetectedPlane struct {
	ID        string    `json:"id"`
	Type      PlaneType `json:"type"`
	Center    Vector3   `json:"center"`
	Extent    Vector3   `json:"extent"` // Width, height, depth
	Normal    Vector3   `json:"normal"`
	Transform Transform `json:"transform"`
	Vertices  []Vector3 `json:"vertices,omitempty"`
}

// Area calculates the area of the plane
func (p *DetectedPlane) Area() float64 {
	return p.Extent.X * p.Extent.Z
}

// ContainsPoint checks if a point is within the plane bounds
func (p *DetectedPlane) ContainsPoint(point Vector3) bool {
	// Simplified bounds check
	dx := math.Abs(point.X - p.Center.X)
	dz := math.Abs(point.Z - p.Center.Z)
	return dx <= p.Extent.X/2 && dz <= p.Extent.Z/2
}

// ARObject represents an object placed in AR space
type ARObject struct {
	ID          string      `json:"id"`
	Type        string      `json:"type"`
	Transform   Transform   `json:"transform"`
	AnchorID    string      `json:"anchor_id,omitempty"`
	PlaneID     string      `json:"plane_id,omitempty"`
	IsVisible   bool        `json:"is_visible"`
	IsAnimating bool        `json:"is_animating"`
	Animation   string      `json:"animation,omitempty"`
	Data        interface{} `json:"data,omitempty"`
}

// ARPet represents a pet placed in AR space
type ARPet struct {
	ARObject
	PetID        types.PetID `json:"pet_id"`
	Behavior     string      `json:"behavior"`
	TargetPoint  *Vector3    `json:"target_point,omitempty"`
	MovementSpeed float64    `json:"movement_speed"`
	IsMoving     bool        `json:"is_moving"`
	LookAtCamera bool        `json:"look_at_camera"`
}

// Gesture represents a detected gesture
type Gesture struct {
	Type      GestureType `json:"type"`
	Position  Vector3     `json:"position"`
	StartPos  Vector3     `json:"start_position,omitempty"`
	EndPos    Vector3     `json:"end_position,omitempty"`
	Delta     Vector3     `json:"delta,omitempty"`
	Scale     float64     `json:"scale,omitempty"`
	Rotation  float64     `json:"rotation,omitempty"`
	Velocity  Vector3     `json:"velocity,omitempty"`
	Timestamp time.Time   `json:"timestamp"`
}

// HitTestResult represents a ray cast hit in AR
type HitTestResult struct {
	Position  Vector3   `json:"position"`
	Normal    Vector3   `json:"normal"`
	Distance  float64   `json:"distance"`
	PlaneID   string    `json:"plane_id,omitempty"`
	PlaneType PlaneType `json:"plane_type,omitempty"`
	ObjectID  string    `json:"object_id,omitempty"`
}

// ============================================================================
// AR Session Configuration
// ============================================================================

// ARSessionConfig holds configuration for an AR session
type ARSessionConfig struct {
	TrackingMode         ARTrackingMode `json:"tracking_mode"`
	PlaneDetection       bool           `json:"plane_detection"`
	LightEstimation      bool           `json:"light_estimation"`
	EnvironmentTexturing bool           `json:"environment_texturing"`
	PeopleOcclusion      bool           `json:"people_occlusion"`
	MotionCapture        bool           `json:"motion_capture"`
	AutoFocus            bool           `json:"auto_focus"`
	MaxTrackedPlanes     int            `json:"max_tracked_planes"`
	MinPlaneArea         float64        `json:"min_plane_area"`
}

// DefaultARSessionConfig returns default AR session configuration
func DefaultARSessionConfig() *ARSessionConfig {
	return &ARSessionConfig{
		TrackingMode:         ARTrackingModeWorld,
		PlaneDetection:       true,
		LightEstimation:      true,
		EnvironmentTexturing: false,
		PeopleOcclusion:      false,
		MotionCapture:        false,
		AutoFocus:            true,
		MaxTrackedPlanes:     10,
		MinPlaneArea:         0.1, // 0.1 square meters
	}
}

// ============================================================================
// AR Service
// ============================================================================

// ARService manages AR sessions and interactions
type ARService struct {
	Config         *ARSessionConfig
	State          ARSessionState
	TrackingQuality float64 // 0.0 to 1.0

	// Tracked objects
	DetectedPlanes map[string]*DetectedPlane
	PlacedObjects  map[string]*ARObject
	PlacedPets     map[types.PetID]*ARPet

	// Camera info
	CameraTransform Transform
	LightEstimate   *LightEstimate

	// Callbacks
	OnPlaneDetected      func(*DetectedPlane)
	OnPlaneUpdated       func(*DetectedPlane)
	OnPlaneRemoved       func(string)
	OnTrackingStateChange func(ARSessionState)
	OnGesture            func(*Gesture)
	OnPetInteraction     func(types.PetID, InteractionType, interface{})

	// Internal
	mu              sync.RWMutex
	sessionID       string
	lastUpdateTime  time.Time
	exerciseTracker *ExerciseTracker
}

// LightEstimate represents estimated lighting conditions
type LightEstimate struct {
	AmbientIntensity   float64   `json:"ambient_intensity"`
	AmbientColorTemp   float64   `json:"ambient_color_temp"`
	PrimaryLightDir    Vector3   `json:"primary_light_direction"`
	PrimaryLightIntensity float64 `json:"primary_light_intensity"`
}

// NewARService creates a new AR service
func NewARService(config *ARSessionConfig) *ARService {
	if config == nil {
		config = DefaultARSessionConfig()
	}

	return &ARService{
		Config:          config,
		State:           ARSessionStateInactive,
		TrackingQuality: 0,
		DetectedPlanes:  make(map[string]*DetectedPlane),
		PlacedObjects:   make(map[string]*ARObject),
		PlacedPets:      make(map[types.PetID]*ARPet),
		CameraTransform: NewTransform(),
		exerciseTracker: NewExerciseTracker(),
	}
}

// ============================================================================
// Session Management
// ============================================================================

// StartSession starts an AR session
func (s *ARService) StartSession() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.State != ARSessionStateInactive && s.State != ARSessionStateError {
		return fmt.Errorf("session already active")
	}

	s.State = ARSessionStateInitializing
	s.sessionID = fmt.Sprintf("ar_session_%d", time.Now().UnixNano())
	s.lastUpdateTime = time.Now()

	// Clear previous session data
	s.DetectedPlanes = make(map[string]*DetectedPlane)
	s.TrackingQuality = 0

	if s.OnTrackingStateChange != nil {
		s.OnTrackingStateChange(s.State)
	}

	// Simulate initialization delay
	go func() {
		time.Sleep(500 * time.Millisecond)
		s.mu.Lock()
		if s.State == ARSessionStateInitializing {
			s.State = ARSessionStateTracking
			s.TrackingQuality = 0.5
			if s.OnTrackingStateChange != nil {
				s.OnTrackingStateChange(s.State)
			}
		}
		s.mu.Unlock()
	}()

	return nil
}

// PauseSession pauses the AR session
func (s *ARService) PauseSession() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.State == ARSessionStateTracking || s.State == ARSessionStateLimited {
		s.State = ARSessionStatePaused
		if s.OnTrackingStateChange != nil {
			s.OnTrackingStateChange(s.State)
		}
	}
}

// ResumeSession resumes a paused AR session
func (s *ARService) ResumeSession() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.State == ARSessionStatePaused {
		s.State = ARSessionStateTracking
		if s.OnTrackingStateChange != nil {
			s.OnTrackingStateChange(s.State)
		}
	}
}

// StopSession stops the AR session
func (s *ARService) StopSession() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.State = ARSessionStateInactive
	s.sessionID = ""
	s.TrackingQuality = 0

	if s.OnTrackingStateChange != nil {
		s.OnTrackingStateChange(s.State)
	}
}

// GetSessionState returns current session state
func (s *ARService) GetSessionState() ARSessionState {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.State
}

// GetTrackingQuality returns current tracking quality
func (s *ARService) GetTrackingQuality() float64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.TrackingQuality
}

// ============================================================================
// Plane Detection
// ============================================================================

// AddDetectedPlane adds a newly detected plane
func (s *ARService) AddDetectedPlane(plane *DetectedPlane) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if len(s.DetectedPlanes) >= s.Config.MaxTrackedPlanes {
		return
	}

	if plane.Area() < s.Config.MinPlaneArea {
		return
	}

	s.DetectedPlanes[plane.ID] = plane

	if s.OnPlaneDetected != nil {
		s.OnPlaneDetected(plane)
	}
}

// UpdateDetectedPlane updates an existing plane
func (s *ARService) UpdateDetectedPlane(plane *DetectedPlane) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.DetectedPlanes[plane.ID]; exists {
		s.DetectedPlanes[plane.ID] = plane

		if s.OnPlaneUpdated != nil {
			s.OnPlaneUpdated(plane)
		}
	}
}

// RemoveDetectedPlane removes a plane
func (s *ARService) RemoveDetectedPlane(planeID string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.DetectedPlanes, planeID)

	if s.OnPlaneRemoved != nil {
		s.OnPlaneRemoved(planeID)
	}
}

// GetDetectedPlanes returns all detected planes
func (s *ARService) GetDetectedPlanes() []*DetectedPlane {
	s.mu.RLock()
	defer s.mu.RUnlock()

	planes := make([]*DetectedPlane, 0, len(s.DetectedPlanes))
	for _, plane := range s.DetectedPlanes {
		planes = append(planes, plane)
	}
	return planes
}

// GetLargestHorizontalPlane returns the largest horizontal plane
func (s *ARService) GetLargestHorizontalPlane() *DetectedPlane {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var largest *DetectedPlane
	var maxArea float64

	for _, plane := range s.DetectedPlanes {
		if plane.Type == PlaneTypeHorizontal {
			area := plane.Area()
			if area > maxArea {
				maxArea = area
				largest = plane
			}
		}
	}

	return largest
}

// ============================================================================
// Object Placement
// ============================================================================

// PlaceObject places an object in AR space
func (s *ARService) PlaceObject(obj *ARObject) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.State != ARSessionStateTracking {
		return fmt.Errorf("AR session not tracking")
	}

	obj.IsVisible = true
	s.PlacedObjects[obj.ID] = obj

	return nil
}

// RemoveObject removes an object from AR space
func (s *ARService) RemoveObject(objectID string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.PlacedObjects, objectID)
}

// GetPlacedObject returns a placed object by ID
func (s *ARService) GetPlacedObject(objectID string) *ARObject {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.PlacedObjects[objectID]
}

// ============================================================================
// Pet Placement
// ============================================================================

// PlacePet places a pet in AR space
func (s *ARService) PlacePet(petID types.PetID, position Vector3, planeID string) (*ARPet, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.State != ARSessionStateTracking {
		return nil, fmt.Errorf("AR session not tracking")
	}

	// Verify plane exists if specified
	if planeID != "" {
		if _, exists := s.DetectedPlanes[planeID]; !exists {
			return nil, fmt.Errorf("plane not found: %s", planeID)
		}
	}

	arPet := &ARPet{
		ARObject: ARObject{
			ID:        fmt.Sprintf("ar_pet_%s", petID),
			Type:      "pet",
			Transform: Transform{
				Position: position,
				Rotation: QuaternionIdentity(),
				Scale:    Vector3{X: 1, Y: 1, Z: 1},
			},
			PlaneID:     planeID,
			IsVisible:   true,
			IsAnimating: true,
			Animation:   "idle",
		},
		PetID:         petID,
		Behavior:      "idle",
		MovementSpeed: 0.5, // meters per second
		LookAtCamera:  true,
	}

	s.PlacedPets[petID] = arPet

	return arPet, nil
}

// RemovePet removes a pet from AR space
func (s *ARService) RemovePet(petID types.PetID) {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.PlacedPets, petID)
}

// GetPlacedPet returns a placed pet
func (s *ARService) GetPlacedPet(petID types.PetID) *ARPet {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.PlacedPets[petID]
}

// MovePetTo moves a pet to a target position
func (s *ARService) MovePetTo(petID types.PetID, target Vector3) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	pet, exists := s.PlacedPets[petID]
	if !exists {
		return fmt.Errorf("pet not placed in AR: %s", petID)
	}

	pet.TargetPoint = &target
	pet.IsMoving = true
	pet.Animation = "walking"
	pet.Behavior = "moving"

	return nil
}

// UpdatePetPosition updates pet position during movement
func (s *ARService) UpdatePetPosition(petID types.PetID, deltaTime float64) {
	s.mu.Lock()
	defer s.mu.Unlock()

	pet, exists := s.PlacedPets[petID]
	if !exists || !pet.IsMoving || pet.TargetPoint == nil {
		return
	}

	// Calculate direction and distance
	direction := pet.TargetPoint.Sub(pet.Transform.Position)
	distance := direction.Magnitude()

	// Check if arrived
	if distance < 0.05 { // 5cm threshold
		pet.Transform.Position = *pet.TargetPoint
		pet.TargetPoint = nil
		pet.IsMoving = false
		pet.Animation = "idle"
		pet.Behavior = "idle"
		return
	}

	// Move towards target
	moveDistance := pet.MovementSpeed * deltaTime
	if moveDistance > distance {
		moveDistance = distance
	}

	movement := direction.Normalize().Scale(moveDistance)
	pet.Transform.Position = pet.Transform.Position.Add(movement)
}

// SetPetAnimation sets the pet's current animation
func (s *ARService) SetPetAnimation(petID types.PetID, animation string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	pet, exists := s.PlacedPets[petID]
	if !exists {
		return fmt.Errorf("pet not placed in AR: %s", petID)
	}

	pet.Animation = animation
	pet.IsAnimating = true

	return nil
}

// SetPetBehavior sets the pet's behavior state
func (s *ARService) SetPetBehavior(petID types.PetID, behavior string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	pet, exists := s.PlacedPets[petID]
	if !exists {
		return fmt.Errorf("pet not placed in AR: %s", petID)
	}

	pet.Behavior = behavior

	// Map behavior to animation
	animationMap := map[string]string{
		"idle":      "idle",
		"happy":     "happy",
		"excited":   "excited",
		"sleeping":  "sleep",
		"eating":    "eat",
		"playing":   "play",
		"walking":   "walk",
		"running":   "run",
		"sitting":   "sit",
		"begging":   "beg",
	}

	if anim, ok := animationMap[behavior]; ok {
		pet.Animation = anim
	}

	return nil
}

// ============================================================================
// Hit Testing
// ============================================================================

// HitTest performs a hit test from screen coordinates
func (s *ARService) HitTest(screenX, screenY float64) []*HitTestResult {
	s.mu.RLock()
	defer s.mu.RUnlock()

	results := make([]*HitTestResult, 0)

	// Check hits against planes
	for _, plane := range s.DetectedPlanes {
		// Simplified hit test - in real implementation this would use ray casting
		result := &HitTestResult{
			Position:  plane.Center,
			Normal:    plane.Normal,
			Distance:  1.0,
			PlaneID:   plane.ID,
			PlaneType: plane.Type,
		}
		results = append(results, result)
	}

	return results
}

// HitTestAgainstPlanes performs hit test only against detected planes
func (s *ARService) HitTestAgainstPlanes(screenX, screenY float64, planeTypes []PlaneType) []*HitTestResult {
	allResults := s.HitTest(screenX, screenY)

	if len(planeTypes) == 0 {
		return allResults
	}

	filtered := make([]*HitTestResult, 0)
	for _, result := range allResults {
		for _, pt := range planeTypes {
			if result.PlaneType == pt {
				filtered = append(filtered, result)
				break
			}
		}
	}

	return filtered
}

// ============================================================================
// Gesture Handling
// ============================================================================

// ProcessGesture processes a gesture and triggers appropriate actions
func (s *ARService) ProcessGesture(gesture *Gesture) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Fire callback
	if s.OnGesture != nil {
		s.OnGesture(gesture)
	}

	// Check if gesture hit any placed pets
	for petID, pet := range s.PlacedPets {
		if s.isGestureOnPet(gesture, pet) {
			s.handlePetGesture(petID, gesture)
			return
		}
	}
}

// isGestureOnPet checks if a gesture is on a pet
func (s *ARService) isGestureOnPet(gesture *Gesture, pet *ARPet) bool {
	// Simplified check - use distance threshold
	distance := gesture.Position.Distance(pet.Transform.Position)
	return distance < 0.3 // 30cm threshold
}

// handlePetGesture handles gestures on a pet
func (s *ARService) handlePetGesture(petID types.PetID, gesture *Gesture) {
	var interactionType InteractionType
	var data interface{}

	switch gesture.Type {
	case GestureTypeTap:
		interactionType = InteractionTypePet
	case GestureTypeDoubleTap:
		interactionType = InteractionTypePlay
	case GestureTypeLongPress:
		interactionType = InteractionTypeFeed
	case GestureTypePan:
		// Pet is being dragged - could be for moving
		return
	case GestureTypeSwipe:
		// Throw interaction
		interactionType = InteractionTypeThrowBall
		data = map[string]interface{}{
			"direction": gesture.Delta.Normalize(),
			"velocity":  gesture.Velocity.Magnitude(),
		}
	}

	if s.OnPetInteraction != nil {
		s.OnPetInteraction(petID, interactionType, data)
	}
}

// ============================================================================
// Camera and Lighting
// ============================================================================

// UpdateCameraTransform updates the camera transform
func (s *ARService) UpdateCameraTransform(transform Transform) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.CameraTransform = transform
	s.lastUpdateTime = time.Now()
}

// UpdateLightEstimate updates the light estimate
func (s *ARService) UpdateLightEstimate(estimate *LightEstimate) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.LightEstimate = estimate
}

// GetCameraTransform returns the current camera transform
func (s *ARService) GetCameraTransform() Transform {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.CameraTransform
}

// GetLightEstimate returns the current light estimate
func (s *ARService) GetLightEstimate() *LightEstimate {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.LightEstimate
}

// ============================================================================
// Exercise Tracking
// ============================================================================

// ExerciseTracker tracks user exercise through AR
type ExerciseTracker struct {
	IsTracking       bool
	StartTime        time.Time
	TotalDistance    float64 // meters
	TotalSteps       int
	CaloriesBurned   float64
	LastPosition     *Vector3
	Positions        []Vector3
	StepThreshold    float64
	mu               sync.RWMutex
}

// NewExerciseTracker creates a new exercise tracker
func NewExerciseTracker() *ExerciseTracker {
	return &ExerciseTracker{
		Positions:     make([]Vector3, 0),
		StepThreshold: 0.7, // 70cm per step
	}
}

// StartTracking starts exercise tracking
func (e *ExerciseTracker) StartTracking() {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.IsTracking = true
	e.StartTime = time.Now()
	e.TotalDistance = 0
	e.TotalSteps = 0
	e.CaloriesBurned = 0
	e.LastPosition = nil
	e.Positions = make([]Vector3, 0)
}

// StopTracking stops exercise tracking
func (e *ExerciseTracker) StopTracking() ExerciseResult {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.IsTracking = false

	return ExerciseResult{
		Duration:       time.Since(e.StartTime),
		TotalDistance:  e.TotalDistance,
		TotalSteps:     e.TotalSteps,
		CaloriesBurned: e.CaloriesBurned,
		AverageSpeed:   e.TotalDistance / time.Since(e.StartTime).Seconds(),
	}
}

// UpdatePosition updates position for exercise tracking
func (e *ExerciseTracker) UpdatePosition(position Vector3) {
	e.mu.Lock()
	defer e.mu.Unlock()

	if !e.IsTracking {
		return
	}

	if e.LastPosition != nil {
		distance := position.Distance(*e.LastPosition)

		// Filter out small movements (noise)
		if distance > 0.1 { // 10cm threshold
			e.TotalDistance += distance
			e.Positions = append(e.Positions, position)

			// Estimate steps
			if distance >= e.StepThreshold {
				steps := int(distance / e.StepThreshold)
				e.TotalSteps += steps
			}

			// Estimate calories (very rough: 0.04 calories per step)
			e.CaloriesBurned = float64(e.TotalSteps) * 0.04
		}
	}

	e.LastPosition = &position
}

// GetExerciseStats returns current exercise stats
func (e *ExerciseTracker) GetExerciseStats() ExerciseResult {
	e.mu.RLock()
	defer e.mu.RUnlock()

	duration := time.Duration(0)
	if e.IsTracking {
		duration = time.Since(e.StartTime)
	}

	avgSpeed := 0.0
	if duration.Seconds() > 0 {
		avgSpeed = e.TotalDistance / duration.Seconds()
	}

	return ExerciseResult{
		Duration:       duration,
		TotalDistance:  e.TotalDistance,
		TotalSteps:     e.TotalSteps,
		CaloriesBurned: e.CaloriesBurned,
		AverageSpeed:   avgSpeed,
	}
}

// ExerciseResult represents exercise tracking results
type ExerciseResult struct {
	Duration       time.Duration `json:"duration"`
	TotalDistance  float64       `json:"total_distance"`
	TotalSteps     int           `json:"total_steps"`
	CaloriesBurned float64       `json:"calories_burned"`
	AverageSpeed   float64       `json:"average_speed"`
}

// ============================================================================
// Exercise Integration with AR Service
// ============================================================================

// StartExerciseTracking starts exercise tracking
func (s *ARService) StartExerciseTracking() {
	s.exerciseTracker.StartTracking()
}

// StopExerciseTracking stops exercise tracking and returns results
func (s *ARService) StopExerciseTracking() ExerciseResult {
	return s.exerciseTracker.StopTracking()
}

// GetExerciseStats returns current exercise stats
func (s *ARService) GetExerciseStats() ExerciseResult {
	return s.exerciseTracker.GetExerciseStats()
}

// IsExerciseTracking returns whether exercise tracking is active
func (s *ARService) IsExerciseTracking() bool {
	return s.exerciseTracker.IsTracking
}

// ============================================================================
// AR Snapshot
// ============================================================================

// ARSnapshot represents a snapshot of the AR state
type ARSnapshot struct {
	SessionState    ARSessionState        `json:"session_state"`
	TrackingQuality float64               `json:"tracking_quality"`
	CameraTransform Transform             `json:"camera_transform"`
	DetectedPlanes  []*DetectedPlane      `json:"detected_planes"`
	PlacedPets      map[types.PetID]*ARPet `json:"placed_pets"`
	LightEstimate   *LightEstimate        `json:"light_estimate,omitempty"`
	Timestamp       time.Time             `json:"timestamp"`
}

// GetSnapshot returns a snapshot of the current AR state
func (s *ARService) GetSnapshot() *ARSnapshot {
	s.mu.RLock()
	defer s.mu.RUnlock()

	planes := make([]*DetectedPlane, 0, len(s.DetectedPlanes))
	for _, p := range s.DetectedPlanes {
		planes = append(planes, p)
	}

	pets := make(map[types.PetID]*ARPet)
	for id, pet := range s.PlacedPets {
		pets[id] = pet
	}

	return &ARSnapshot{
		SessionState:    s.State,
		TrackingQuality: s.TrackingQuality,
		CameraTransform: s.CameraTransform,
		DetectedPlanes:  planes,
		PlacedPets:      pets,
		LightEstimate:   s.LightEstimate,
		Timestamp:       time.Now(),
	}
}
