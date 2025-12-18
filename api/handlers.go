package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/Michael-W-Ellison/gochi/internal/core"
	"github.com/Michael-W-Ellison/gochi/pkg/types"
)

// Handler provides HTTP handler methods for the API
type Handler struct {
	petStore      PetStore
	userStore     UserStore
	socialService SocialService
	syncService   SyncService
}

// PetStore defines the interface for pet data storage
type PetStore interface {
	GetPet(id types.PetID) (*core.DigitalPet, error)
	GetPetsByOwner(ownerID types.UserID) ([]*core.DigitalPet, error)
	CreatePet(pet *core.DigitalPet) error
	UpdatePet(pet *core.DigitalPet) error
	DeletePet(id types.PetID) error
}

// UserStore defines the interface for user data storage
type UserStore interface {
	GetUser(id types.UserID) (*User, error)
	GetUserByEmail(email string) (*User, error)
	CreateUser(user *User) error
	UpdateUser(user *User) error
	ValidateCredentials(email, password string) (*User, error)
}

// User represents a user in the system
type User struct {
	ID          types.UserID `json:"id"`
	Email       string       `json:"email"`
	DisplayName string       `json:"display_name"`
	CreatedAt   time.Time    `json:"created_at"`
}

// SocialService defines the interface for social features
type SocialService interface {
	GetNearbyPets(location *Location, radiusMeters float64) ([]*NearbyPetResponse, error)
	GetFriends(petID types.PetID) ([]*FriendResponse, error)
	SendFriendRequest(fromPetID, toPetID types.PetID, message string) error
	AcceptFriendRequest(requestID string) error
	DeclineFriendRequest(requestID string) error
	RemoveFriend(petID, friendID types.PetID) error
}

// Location represents a geographic location
type Location struct {
	Latitude  float64
	Longitude float64
}

// SyncService defines the interface for cloud sync
type SyncService interface {
	Sync(userID types.UserID, request *SyncRequest) (*SyncResponse, error)
	ResolveConflict(userID types.UserID, request *ConflictResolutionRequest) error
}

// NewHandler creates a new API handler
func NewHandler(petStore PetStore, userStore UserStore, socialService SocialService, syncService SyncService) *Handler {
	return &Handler{
		petStore:      petStore,
		userStore:     userStore,
		socialService: socialService,
		syncService:   syncService,
	}
}

// ============================================================================
// Helper Functions
// ============================================================================

// writeJSON writes a JSON response
func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

// writeSuccess writes a successful API response
func writeSuccess(w http.ResponseWriter, data interface{}) {
	writeJSON(w, http.StatusOK, APIResponse{
		Success:   true,
		Data:      data,
		Timestamp: time.Now(),
	})
}

// writeError writes an error API response
func writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, APIResponse{
		Success: false,
		Error: &APIError{
			Code:    code,
			Message: message,
		},
		Timestamp: time.Now(),
	})
}

// parseJSON parses JSON from request body
func parseJSON(r *http.Request, v interface{}) error {
	return json.NewDecoder(r.Body).Decode(v)
}

// getPathParam extracts a path parameter (placeholder - actual implementation depends on router)
func getPathParam(r *http.Request, name string) string {
	// This would be implemented based on the actual router being used
	// For example, with gorilla/mux: mux.Vars(r)[name]
	return r.PathValue(name)
}

// getQueryParam extracts a query parameter with default value
func getQueryParam(r *http.Request, name, defaultValue string) string {
	value := r.URL.Query().Get(name)
	if value == "" {
		return defaultValue
	}
	return value
}

// getQueryParamInt extracts an integer query parameter with default value
func getQueryParamInt(r *http.Request, name string, defaultValue int) int {
	value := r.URL.Query().Get(name)
	if value == "" {
		return defaultValue
	}
	intValue, err := strconv.Atoi(value)
	if err != nil {
		return defaultValue
	}
	return intValue
}

// getQueryParamFloat extracts a float query parameter with default value
func getQueryParamFloat(r *http.Request, name string, defaultValue float64) float64 {
	value := r.URL.Query().Get(name)
	if value == "" {
		return defaultValue
	}
	floatValue, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return defaultValue
	}
	return floatValue
}

// getUserIDFromContext extracts user ID from request context (set by auth middleware)
func getUserIDFromContext(r *http.Request) types.UserID {
	// This would be implemented based on your auth middleware
	// For example: return r.Context().Value("userID").(types.UserID)
	return ""
}

// ============================================================================
// Pet Handlers
// ============================================================================

// HandleCreatePet handles POST /api/v1/pets
func (h *Handler) HandleCreatePet(w http.ResponseWriter, r *http.Request) {
	var req CreatePetRequest
	if err := parseJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
		return
	}

	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "MISSING_NAME", "Pet name is required")
		return
	}

	userID := getUserIDFromContext(r)

	var pet *core.DigitalPet
	if req.RandomizeTraits {
		pet = core.NewDigitalPetRandom(req.Name, userID)
	} else {
		pet = core.NewDigitalPet(req.Name, userID)
	}

	if err := h.petStore.CreatePet(pet); err != nil {
		writeError(w, http.StatusInternalServerError, "CREATE_FAILED", "Failed to create pet")
		return
	}

	writeSuccess(w, convertPetToResponse(pet))
}

// HandleGetPets handles GET /api/v1/pets
func (h *Handler) HandleGetPets(w http.ResponseWriter, r *http.Request) {
	userID := getUserIDFromContext(r)

	pets, err := h.petStore.GetPetsByOwner(userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "FETCH_FAILED", "Failed to fetch pets")
		return
	}

	var responses []PetResponse
	for _, pet := range pets {
		responses = append(responses, *convertPetToResponse(pet))
	}

	writeSuccess(w, responses)
}

// HandleGetPet handles GET /api/v1/pets/{petId}
func (h *Handler) HandleGetPet(w http.ResponseWriter, r *http.Request) {
	petID := types.PetID(getPathParam(r, "petId"))

	pet, err := h.petStore.GetPet(petID)
	if err != nil {
		writeError(w, http.StatusNotFound, "PET_NOT_FOUND", "Pet not found")
		return
	}

	writeSuccess(w, convertPetToResponse(pet))
}

// HandleUpdatePet handles PUT /api/v1/pets/{petId}
func (h *Handler) HandleUpdatePet(w http.ResponseWriter, r *http.Request) {
	petID := types.PetID(getPathParam(r, "petId"))

	var req UpdatePetRequest
	if err := parseJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
		return
	}

	pet, err := h.petStore.GetPet(petID)
	if err != nil {
		writeError(w, http.StatusNotFound, "PET_NOT_FOUND", "Pet not found")
		return
	}

	if req.Name != "" {
		pet.Name = req.Name
	}
	if req.Location != "" {
		pet.Location = req.Location
	}

	if err := h.petStore.UpdatePet(pet); err != nil {
		writeError(w, http.StatusInternalServerError, "UPDATE_FAILED", "Failed to update pet")
		return
	}

	writeSuccess(w, convertPetToResponse(pet))
}

// HandleDeletePet handles DELETE /api/v1/pets/{petId}
func (h *Handler) HandleDeletePet(w http.ResponseWriter, r *http.Request) {
	petID := types.PetID(getPathParam(r, "petId"))

	if err := h.petStore.DeletePet(petID); err != nil {
		writeError(w, http.StatusNotFound, "PET_NOT_FOUND", "Pet not found")
		return
	}

	writeSuccess(w, map[string]bool{"deleted": true})
}

// HandleGetPetStatus handles GET /api/v1/pets/{petId}/status
func (h *Handler) HandleGetPetStatus(w http.ResponseWriter, r *http.Request) {
	petID := types.PetID(getPathParam(r, "petId"))

	pet, err := h.petStore.GetPet(petID)
	if err != nil {
		writeError(w, http.StatusNotFound, "PET_NOT_FOUND", "Pet not found")
		return
	}

	status := pet.GetCurrentStatus()
	writeSuccess(w, PetStatusResponse{
		ID:              pet.ID,
		Name:            pet.Name,
		IsAlive:         status.IsAlive,
		CurrentBehavior: status.CurrentBehavior.String(),
		Health:          status.Health,
		Energy:          status.Energy,
		Happiness:       status.Happiness,
		Wellbeing:       status.Wellbeing,
		MoodDescription: status.MoodDescription,
		CriticalNeeds:   status.CriticalNeeds,
	})
}

// ============================================================================
// Interaction Handlers
// ============================================================================

// HandleInteraction handles POST /api/v1/pets/{petId}/interactions
func (h *Handler) HandleInteraction(w http.ResponseWriter, r *http.Request) {
	petID := types.PetID(getPathParam(r, "petId"))

	var req InteractionRequest
	if err := parseJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
		return
	}

	pet, err := h.petStore.GetPet(petID)
	if err != nil {
		writeError(w, http.StatusNotFound, "PET_NOT_FOUND", "Pet not found")
		return
	}

	// Map string type to InteractionType
	interactionType := mapStringToInteractionType(req.Type)
	if interactionType == -1 {
		writeError(w, http.StatusBadRequest, "INVALID_TYPE", "Invalid interaction type")
		return
	}

	// Store initial values for comparison
	initialHealth := pet.Biology.Vitals.Health
	initialEnergy := pet.Biology.Vitals.Energy
	initialHappiness := pet.Biology.Vitals.Happiness
	initialJoy := pet.Emotions.Joy

	// Process the interaction
	pet.ProcessUserInteraction(interactionType, req.Intensity)

	// Save updated pet
	if err := h.petStore.UpdatePet(pet); err != nil {
		writeError(w, http.StatusInternalServerError, "UPDATE_FAILED", "Failed to save interaction")
		return
	}

	writeSuccess(w, InteractionResponse{
		Success:     true,
		PetReaction: pet.CurrentBehavior.String(),
		VitalChanges: map[string]float64{
			"health":    pet.Biology.Vitals.Health - initialHealth,
			"energy":    pet.Biology.Vitals.Energy - initialEnergy,
			"happiness": pet.Biology.Vitals.Happiness - initialHappiness,
		},
		EmotionChanges: map[string]float64{
			"joy": pet.Emotions.Joy - initialJoy,
		},
		MemoryCreated: true,
		Message:       "Interaction successful",
	})
}

// HandleFeed handles POST /api/v1/pets/{petId}/interactions/feed
func (h *Handler) HandleFeed(w http.ResponseWriter, r *http.Request) {
	petID := types.PetID(getPathParam(r, "petId"))

	var req FeedingRequest
	if err := parseJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
		return
	}

	pet, err := h.petStore.GetPet(petID)
	if err != nil {
		writeError(w, http.StatusNotFound, "PET_NOT_FOUND", "Pet not found")
		return
	}

	// Process feeding interaction
	pet.ProcessUserInteraction(types.InteractionFeeding, req.Quantity)

	if err := h.petStore.UpdatePet(pet); err != nil {
		writeError(w, http.StatusInternalServerError, "UPDATE_FAILED", "Failed to save")
		return
	}

	writeSuccess(w, InteractionResponse{
		Success:     true,
		PetReaction: pet.CurrentBehavior.String(),
		Message:     "Pet has been fed",
	})
}

// HandlePlay handles POST /api/v1/pets/{petId}/interactions/play
func (h *Handler) HandlePlay(w http.ResponseWriter, r *http.Request) {
	petID := types.PetID(getPathParam(r, "petId"))

	var req PlayRequest
	if err := parseJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
		return
	}

	pet, err := h.petStore.GetPet(petID)
	if err != nil {
		writeError(w, http.StatusNotFound, "PET_NOT_FOUND", "Pet not found")
		return
	}

	// Process play interaction
	pet.ProcessUserInteraction(types.InteractionPlaying, req.Enthusiasm)

	if err := h.petStore.UpdatePet(pet); err != nil {
		writeError(w, http.StatusInternalServerError, "UPDATE_FAILED", "Failed to save")
		return
	}

	writeSuccess(w, InteractionResponse{
		Success:     true,
		PetReaction: pet.CurrentBehavior.String(),
		Message:     "Played with pet",
	})
}

// ============================================================================
// Social Handlers
// ============================================================================

// HandleUpdateLocation handles PUT /api/v1/social/location
func (h *Handler) HandleUpdateLocation(w http.ResponseWriter, r *http.Request) {
	var req LocationUpdateRequest
	if err := parseJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
		return
	}

	// Validate coordinates
	if req.Latitude < -90 || req.Latitude > 90 || req.Longitude < -180 || req.Longitude > 180 {
		writeError(w, http.StatusBadRequest, "INVALID_COORDINATES", "Invalid latitude or longitude")
		return
	}

	// Update location in social service (implementation depends on social service)
	writeSuccess(w, map[string]interface{}{
		"updated":   true,
		"latitude":  req.Latitude,
		"longitude": req.Longitude,
	})
}

// HandleGetNearbyPets handles GET /api/v1/social/nearby
func (h *Handler) HandleGetNearbyPets(w http.ResponseWriter, r *http.Request) {
	lat := getQueryParamFloat(r, "latitude", 0)
	lon := getQueryParamFloat(r, "longitude", 0)
	radius := getQueryParamFloat(r, "radius", 1000) // Default 1km

	if lat == 0 && lon == 0 {
		writeError(w, http.StatusBadRequest, "MISSING_LOCATION", "Latitude and longitude are required")
		return
	}

	location := &Location{Latitude: lat, Longitude: lon}
	nearby, err := h.socialService.GetNearbyPets(location, radius)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "FETCH_FAILED", "Failed to fetch nearby pets")
		return
	}

	writeSuccess(w, nearby)
}

// HandleGetFriends handles GET /api/v1/social/friends
func (h *Handler) HandleGetFriends(w http.ResponseWriter, r *http.Request) {
	petID := types.PetID(getQueryParam(r, "pet_id", ""))
	if petID == "" {
		writeError(w, http.StatusBadRequest, "MISSING_PET_ID", "Pet ID is required")
		return
	}

	friends, err := h.socialService.GetFriends(petID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "FETCH_FAILED", "Failed to fetch friends")
		return
	}

	writeSuccess(w, friends)
}

// HandleSendFriendRequest handles POST /api/v1/social/friends/request
func (h *Handler) HandleSendFriendRequest(w http.ResponseWriter, r *http.Request) {
	var req FriendRequest
	if err := parseJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
		return
	}

	// Get current user's pet ID from context
	fromPetID := types.PetID(getQueryParam(r, "pet_id", ""))
	if fromPetID == "" {
		writeError(w, http.StatusBadRequest, "MISSING_PET_ID", "Source pet ID is required")
		return
	}

	if err := h.socialService.SendFriendRequest(fromPetID, req.TargetPetID, req.Message); err != nil {
		writeError(w, http.StatusInternalServerError, "REQUEST_FAILED", "Failed to send friend request")
		return
	}

	writeSuccess(w, map[string]bool{"sent": true})
}

// HandleAcceptFriendRequest handles POST /api/v1/social/friends/requests/{requestId}/accept
func (h *Handler) HandleAcceptFriendRequest(w http.ResponseWriter, r *http.Request) {
	requestID := getPathParam(r, "requestId")

	if err := h.socialService.AcceptFriendRequest(requestID); err != nil {
		writeError(w, http.StatusInternalServerError, "ACCEPT_FAILED", "Failed to accept friend request")
		return
	}

	writeSuccess(w, map[string]bool{"accepted": true})
}

// HandleDeclineFriendRequest handles POST /api/v1/social/friends/requests/{requestId}/decline
func (h *Handler) HandleDeclineFriendRequest(w http.ResponseWriter, r *http.Request) {
	requestID := getPathParam(r, "requestId")

	if err := h.socialService.DeclineFriendRequest(requestID); err != nil {
		writeError(w, http.StatusInternalServerError, "DECLINE_FAILED", "Failed to decline friend request")
		return
	}

	writeSuccess(w, map[string]bool{"declined": true})
}

// HandleRemoveFriend handles DELETE /api/v1/social/friends/{petId}
func (h *Handler) HandleRemoveFriend(w http.ResponseWriter, r *http.Request) {
	friendPetID := types.PetID(getPathParam(r, "petId"))
	myPetID := types.PetID(getQueryParam(r, "my_pet_id", ""))

	if myPetID == "" {
		writeError(w, http.StatusBadRequest, "MISSING_PET_ID", "Your pet ID is required")
		return
	}

	if err := h.socialService.RemoveFriend(myPetID, friendPetID); err != nil {
		writeError(w, http.StatusInternalServerError, "REMOVE_FAILED", "Failed to remove friend")
		return
	}

	writeSuccess(w, map[string]bool{"removed": true})
}

// ============================================================================
// Sync Handlers
// ============================================================================

// HandleSync handles POST /api/v1/sync
func (h *Handler) HandleSync(w http.ResponseWriter, r *http.Request) {
	var req SyncRequest
	if err := parseJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
		return
	}

	userID := getUserIDFromContext(r)
	response, err := h.syncService.Sync(userID, &req)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "SYNC_FAILED", "Failed to sync")
		return
	}

	writeSuccess(w, response)
}

// HandleResolveConflict handles POST /api/v1/sync/resolve
func (h *Handler) HandleResolveConflict(w http.ResponseWriter, r *http.Request) {
	var req ConflictResolutionRequest
	if err := parseJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
		return
	}

	userID := getUserIDFromContext(r)
	if err := h.syncService.ResolveConflict(userID, &req); err != nil {
		writeError(w, http.StatusInternalServerError, "RESOLVE_FAILED", "Failed to resolve conflict")
		return
	}

	writeSuccess(w, map[string]bool{"resolved": true})
}

// ============================================================================
// Helper Converters
// ============================================================================

// convertPetToResponse converts a DigitalPet to a PetResponse
func convertPetToResponse(pet *core.DigitalPet) *PetResponse {
	return &PetResponse{
		ID:              pet.ID,
		Name:            pet.Name,
		Owner:           pet.Owner,
		Age:             pet.Biology.GetAgeInDays(),
		IsAlive:         pet.Biology.IsAlive,
		CurrentBehavior: pet.CurrentBehavior.String(),
		Location:        pet.Location,
		Vitals: &VitalsResponse{
			Health:      pet.Biology.Vitals.Health,
			Energy:      pet.Biology.Vitals.Energy,
			Happiness:   pet.Biology.Vitals.Happiness,
			Nutrition:   pet.Biology.Vitals.Nutrition,
			Hydration:   pet.Biology.Vitals.Hydration,
			Cleanliness: pet.Biology.Vitals.Cleanliness,
			Stress:      pet.Biology.Vitals.Stress,
			Fatigue:     pet.Biology.Vitals.Fatigue,
			Wellbeing:   pet.Biology.Vitals.GetOverallWellbeing(),
		},
		Emotions: &EmotionsResponse{
			Joy:             pet.Emotions.Joy,
			Sadness:         pet.Emotions.Sadness,
			Anger:           pet.Emotions.Anger,
			Fear:            pet.Emotions.Fear,
			Excitement:      pet.Emotions.Excitement,
			Contentment:     pet.Emotions.Contentment,
			Affection:       pet.Emotions.Affection,
			Loneliness:      pet.Emotions.Loneliness,
			DominantEmotion: pet.Emotions.DominantEmotion,
			MoodScore:       pet.Emotions.GetMoodScore(),
			MoodDescription: pet.Emotions.GetMoodDescription(),
			MoodTrend:       pet.Emotions.GetMoodTrendDescription(),
		},
		Personality: &PersonalityResponse{
			Openness:          pet.Personality.Traits.Openness,
			Conscientiousness: pet.Personality.Traits.Conscientiousness,
			Extraversion:      pet.Personality.Traits.Extraversion,
			Agreeableness:     pet.Personality.Traits.Agreeableness,
			Neuroticism:       pet.Personality.Traits.Neuroticism,
			Playfulness:       pet.Personality.Traits.Playfulness,
			Independence:      pet.Personality.Traits.Independence,
			Loyalty:           pet.Personality.Traits.Loyalty,
			Intelligence:      pet.Personality.Traits.Intelligence,
			EnergyLevel:       pet.Personality.Traits.EnergyLevel,
			Adaptability:      pet.Personality.Traits.Adaptability,
			Description:       pet.Personality.GetPersonalityDescription(),
		},
		Statistics: &PetStatistics{
			TotalInteractions:  pet.TotalInteractions,
			TotalPlayTimeHours: pet.TotalPlayTime,
			DaysAlive:          pet.Biology.GetAgeInDays(),
			FriendCount:        pet.Relationships.GetRelationshipCount(),
		},
		CreatedAt:     pet.CreatedAt,
		LastUpdatedAt: pet.LastUpdateAt,
	}
}

// mapStringToInteractionType converts a string to InteractionType
func mapStringToInteractionType(s string) types.InteractionType {
	switch s {
	case "feeding", "feed":
		return types.InteractionFeeding
	case "petting", "pet":
		return types.InteractionPetting
	case "playing", "play":
		return types.InteractionPlaying
	case "training", "train":
		return types.InteractionTraining
	case "grooming", "groom":
		return types.InteractionGrooming
	case "medical", "medical_care":
		return types.InteractionMedicalCare
	case "reward", "rewards":
		return types.InteractionRewards
	case "discipline":
		return types.InteractionDiscipline
	default:
		return -1
	}
}
