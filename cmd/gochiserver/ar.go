package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/Michael-W-Ellison/gochi/internal/ar"
	"github.com/Michael-W-Ellison/gochi/pkg/types"
)

// These endpoints let an AR client (one that tracks the camera and finds
// surfaces, such as a WebXR page or a phone app) place the active pet in
// the room, care for it with gestures and take it for walks. The client
// reports what it sees; the server keeps the AR scene and the pet.

// arTouch is a pet interaction found while processing a gesture
type arTouch struct {
	petID types.PetID
	kind  ar.InteractionType
}

// arCare maps AR interactions to the care they give
var arCare = map[ar.InteractionType]types.InteractionType{
	ar.InteractionTypePet:       types.InteractionPetting,
	ar.InteractionTypePlay:      types.InteractionPlaying,
	ar.InteractionTypeFeed:      types.InteractionFeeding,
	ar.InteractionTypeThrowBall: types.InteractionPlaying,
}

// arBehaviors maps pet behaviors to the AR package's behavior names,
// which choose the pet's animation
var arBehaviors = map[types.BehaviorState]string{
	types.BehaviorIdle:          "idle",
	types.BehaviorSleeping:      "sleeping",
	types.BehaviorEating:        "eating",
	types.BehaviorPlaying:       "playing",
	types.BehaviorExploring:     "walking",
	types.BehaviorInvestigating: "walking",
	types.BehaviorExercising:    "running",
	types.BehaviorHappy:         "happy",
	types.BehaviorExcited:       "excited",
	types.BehaviorHibernating:   "sleeping",
}

// arServiceLocked returns the AR service, creating it on first use.
// Caller must hold s.arMu.
func (s *Server) arServiceLocked() *ar.ARService {
	if s.arService == nil {
		s.arService = ar.NewARService(nil)
		// Called by ProcessGesture, which runs with s.arMu held
		s.arService.OnPetInteraction = func(petID types.PetID, kind ar.InteractionType, _ interface{}) {
			s.arTouches = append(s.arTouches, arTouch{petID, kind})
		}
	}
	return s.arService
}

// activePetInfo returns the active pet's ID and behavior, or false if
// there is no pet
func (s *Server) activePetInfo() (types.PetID, types.BehaviorState, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.pet == nil {
		return "", 0, false
	}
	return s.pet.ID, s.pet.GetCurrentStatus().CurrentBehavior, true
}

// syncARPet removes pets that are no longer active from the AR scene and
// shows the active pet's behavior. Caller must hold s.arMu.
func (s *Server) syncARPetLocked(svc *ar.ARService, petID types.PetID, behavior types.BehaviorState) {
	for id := range svc.GetSnapshot().PlacedPets {
		if id != petID {
			svc.RemovePet(id)
		}
	}
	if name, ok := arBehaviors[behavior]; ok && svc.GetPlacedPet(petID) != nil {
		_ = svc.SetPetBehavior(petID, name) // Fails only if the pet isn't placed, checked above
	}
}

// decodePost decodes a POST request's JSON body into v, or writes an
// error and returns false
func decodePost(w http.ResponseWriter, r *http.Request, v interface{}) bool {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, APIResponse{Success: false, Error: "Method not allowed"})
		return false
	}
	if err := json.NewDecoder(r.Body).Decode(v); err != nil {
		writeJSON(w, http.StatusBadRequest, APIResponse{Success: false, Error: "Invalid request"})
		return false
	}
	return true
}

// handleARState handles GET /api/ar - the AR scene, with the active pet's
// current behavior
func (s *Server) handleARState(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, APIResponse{Success: false, Error: "Method not allowed"})
		return
	}
	petID, behavior, ok := s.activePetInfo()

	s.arMu.Lock()
	defer s.arMu.Unlock()
	svc := s.arServiceLocked()
	if ok {
		s.syncARPetLocked(svc, petID, behavior)
	}
	writeJSON(w, http.StatusOK, APIResponse{Success: true, Data: map[string]interface{}{
		"scene":             svc.GetSnapshot(),
		"exercise_tracking": svc.IsExerciseTracking(),
		"exercise":          svc.GetExerciseStats(),
	}})
}

// handleARSession handles POST /api/ar/session with {"action": "start",
// "pause", "resume" or "stop"}
func (s *Server) handleARSession(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Action string `json:"action"`
	}
	if !decodePost(w, r, &req) {
		return
	}

	s.arMu.Lock()
	defer s.arMu.Unlock()
	svc := s.arServiceLocked()
	switch req.Action {
	case "start":
		if err := svc.StartSession(); err != nil {
			writeJSON(w, http.StatusConflict, APIResponse{Success: false, Error: "An AR session is already running"})
			return
		}
	case "pause":
		svc.PauseSession()
	case "resume":
		svc.ResumeSession()
	case "stop":
		svc.StopSession()
		for id := range svc.GetSnapshot().PlacedPets {
			svc.RemovePet(id) // The pet leaves the room with the session
		}
	default:
		writeJSON(w, http.StatusBadRequest, APIResponse{Success: false, Error: "Unknown action"})
		return
	}
	writeJSON(w, http.StatusOK, APIResponse{Success: true, Data: map[string]interface{}{
		"state": svc.GetSessionState(),
	}})
}

// handleARPlane handles POST /api/ar/plane with {"plane": {...}} to report
// a surface the client found or updated, or {"remove": id} when it is gone
func (s *Server) handleARPlane(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Plane  *ar.DetectedPlane `json:"plane"`
		Remove string            `json:"remove"`
	}
	if !decodePost(w, r, &req) {
		return
	}

	s.arMu.Lock()
	defer s.arMu.Unlock()
	svc := s.arServiceLocked()
	switch {
	case req.Remove != "":
		svc.RemoveDetectedPlane(req.Remove)
	case req.Plane != nil && req.Plane.ID != "":
		known := false
		for _, plane := range svc.GetDetectedPlanes() {
			known = known || plane.ID == req.Plane.ID
		}
		if known {
			svc.UpdateDetectedPlane(req.Plane)
		} else {
			svc.AddDetectedPlane(req.Plane) // Ignored if too small or too many
		}
	default:
		writeJSON(w, http.StatusBadRequest, APIResponse{Success: false, Error: "Send a plane with an id, or remove"})
		return
	}
	writeJSON(w, http.StatusOK, APIResponse{Success: true, Data: map[string]interface{}{
		"planes": len(svc.GetDetectedPlanes()),
	}})
}

// handleARCamera handles POST /api/ar/camera with the camera's
// {"transform": {...}} and, optionally, a {"light": {...}} estimate
func (s *Server) handleARCamera(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Transform *ar.Transform     `json:"transform"`
		Light     *ar.LightEstimate `json:"light"`
	}
	if !decodePost(w, r, &req) {
		return
	}

	s.arMu.Lock()
	defer s.arMu.Unlock()
	svc := s.arServiceLocked()
	if req.Transform != nil {
		svc.UpdateCameraTransform(*req.Transform)
	}
	if req.Light != nil {
		svc.UpdateLightEstimate(req.Light)
	}
	writeJSON(w, http.StatusOK, APIResponse{Success: true})
}

// handleARPlace handles POST /api/ar/place with {"position": {...},
// "plane_id": id}. Without either, the pet is placed in the middle of the
// largest floor or table found.
func (s *Server) handleARPlace(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Position *ar.Vector3 `json:"position"`
		PlaneID  string      `json:"plane_id"`
	}
	if !decodePost(w, r, &req) {
		return
	}
	petID, behavior, ok := s.activePetInfo()
	if !ok {
		writeJSON(w, http.StatusNotFound, APIResponse{Success: false, Error: "No pet found"})
		return
	}

	s.arMu.Lock()
	defer s.arMu.Unlock()
	svc := s.arServiceLocked()
	if svc.GetSessionState() != ar.ARSessionStateTracking {
		writeJSON(w, http.StatusConflict, APIResponse{Success: false, Error: "Start an AR session and wait for tracking first"})
		return
	}

	position := req.Position
	if position == nil {
		plane := svc.GetLargestHorizontalPlane()
		if plane == nil {
			writeJSON(w, http.StatusConflict, APIResponse{Success: false, Error: "No floor or table found yet"})
			return
		}
		position, req.PlaneID = &plane.Center, plane.ID
	}

	placed, err := svc.PlacePet(petID, *position, req.PlaneID)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, APIResponse{Success: false, Error: "Could not place the pet: " + err.Error()})
		return
	}
	s.syncARPetLocked(svc, petID, behavior)
	writeJSON(w, http.StatusOK, APIResponse{Success: true, Data: placed})
}

// handleARGesture handles POST /api/ar/gesture with a gesture the client
// recognized. A gesture on the pet cares for it: a tap pets it, a double
// tap plays, a long press feeds it and a swipe throws a ball.
func (s *Server) handleARGesture(w http.ResponseWriter, r *http.Request) {
	var gesture ar.Gesture
	if !decodePost(w, r, &gesture) {
		return
	}
	if gesture.Timestamp.IsZero() {
		gesture.Timestamp = time.Now()
	}

	s.arMu.Lock()
	s.arTouches = nil
	s.arServiceLocked().ProcessGesture(&gesture)
	touches := s.arTouches
	s.arTouches = nil
	s.arMu.Unlock()

	var touch *arTouch
	for i := range touches {
		if _, ok := arCare[touches[i].kind]; ok {
			touch = &touches[i]
			break
		}
	}
	if touch == nil {
		writeJSON(w, http.StatusOK, APIResponse{Success: true, Data: map[string]interface{}{"hit": false}})
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if s.pet == nil || s.pet.ID != touch.petID {
		writeJSON(w, http.StatusOK, APIResponse{Success: true, Data: map[string]interface{}{"hit": false}})
		return
	}
	s.markActiveLocked(time.Now())
	result := s.pet.Interact(arCare[touch.kind], 1.0)
	if !result.Applied {
		message := result.Refusal
		if message == "" {
			message = s.unavailableMessageLocked()
		}
		writeJSON(w, http.StatusConflict, APIResponse{Success: false, Error: message})
		return
	}
	data := s.interactionData(result)
	data["hit"] = true
	data["interaction"] = touch.kind
	writeJSON(w, http.StatusOK, APIResponse{Success: true, Data: data})
}

// handleARExercise handles POST /api/ar/exercise. {"action": "start"}
// starts tracking a walk with the pet, {"action": "position", "position":
// {...}} reports where the player is, and {"action": "stop"} ends the walk
// and counts it towards the pet's exercise need.
func (s *Server) handleARExercise(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Action   string      `json:"action"`
		Position *ar.Vector3 `json:"position"`
	}
	if !decodePost(w, r, &req) {
		return
	}

	s.arMu.Lock()
	svc := s.arServiceLocked()
	var walk ar.ExerciseResult
	switch {
	case req.Action == "start":
		svc.StartExerciseTracking()
	case req.Action == "position" && req.Position != nil:
		svc.UpdateExercisePosition(*req.Position)
	case req.Action == "stop" && svc.IsExerciseTracking():
		walk = svc.StopExerciseTracking()
	case req.Action == "stop":
		s.arMu.Unlock()
		writeJSON(w, http.StatusConflict, APIResponse{Success: false, Error: "No walk is being tracked"})
		return
	default:
		s.arMu.Unlock()
		writeJSON(w, http.StatusBadRequest, APIResponse{Success: false, Error: "Unknown action"})
		return
	}
	stats := svc.GetExerciseStats()
	s.arMu.Unlock()

	if req.Action != "stop" {
		writeJSON(w, http.StatusOK, APIResponse{Success: true, Data: stats})
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if s.pet == nil {
		writeJSON(w, http.StatusNotFound, APIResponse{Success: false, Error: "No pet found"})
		return
	}
	s.markActiveLocked(time.Now())
	if _, err := s.pet.Walk(walk.TotalDistance); err != nil {
		writeActivityError(w, s.pet.Name, err)
		return
	}
	writeJSON(w, http.StatusOK, APIResponse{Success: true, Data: map[string]interface{}{
		"walk": walk,
		"text": fmt.Sprintf("%s walked %.0f m with you.", s.pet.Name, walk.TotalDistance),
	}})
}
