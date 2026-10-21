package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/Michael-W-Ellison/gochi/internal/core"
	"github.com/Michael-W-Ellison/gochi/pkg/types"
)

// Handler provides HTTP handler methods for the API
type Handler struct {
	petStore PetStore
}

// PetStore defines the interface for pet data storage
type PetStore interface {
	GetPet(id types.PetID) (*core.DigitalPet, error)
	GetPetsByOwner(ownerID types.UserID) ([]*core.DigitalPet, error)
	CreatePet(pet *core.DigitalPet) error
	// ModifyPet applies change to the pet and saves it, unless change
	// returns an error. It returns the pet as changed.
	ModifyPet(id types.PetID, change func(pet *core.DigitalPet) error) (*core.DigitalPet, error)
	DeletePet(id types.PetID) error
}

// ErrTooManyPets is returned by a PetStore that has as many pets as it can
var ErrTooManyPets = errors.New("too many pets")

// ErrNotFound is returned by a PetStore for a pet it doesn't have
var ErrNotFound = errors.New("pet not found")

// ErrNotPermitted is returned by a PetStore that does not allow an
// operation through the API
var ErrNotPermitted = errors.New("not permitted")

// NewHandler creates a new API handler
func NewHandler(petStore PetStore) *Handler {
	return &Handler{petStore: petStore}
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
	userID, _ := r.Context().Value(ContextKeyUserID).(types.UserID)
	return userID
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
	if !core.ValidPetName(req.Name) {
		writeError(w, http.StatusBadRequest, "INVALID_NAME", core.PetNameRule)
		return
	}

	userID := getUserIDFromContext(r)

	species, ok := core.ParseSpecies(req.Species)
	if !ok {
		writeError(w, http.StatusBadRequest, "INVALID_SPECIES", "Species must be cat or dog")
		return
	}

	var pet *core.DigitalPet
	if req.RandomizeTraits {
		pet = core.NewDigitalPetRandom(req.Name, userID)
	} else {
		pet = core.NewDigitalPet(req.Name, userID)
	}
	pet.Species = species

	if err := h.petStore.CreatePet(pet); errors.Is(err, ErrTooManyPets) {
		writeError(w, http.StatusConflict, "TOO_MANY_PETS", "There are as many pets as there can be; delete one in the game")
		return
	} else if err != nil {
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

	responses := []PetResponse{}
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
	if req.Name != "" && !core.ValidPetName(req.Name) {
		writeError(w, http.StatusBadRequest, "INVALID_NAME", core.PetNameRule)
		return
	}
	if req.Location != "" && !core.KnownPlace(req.Location) {
		writeError(w, http.StatusBadRequest, "INVALID_LOCATION", "Unknown location")
		return
	}

	pet, err := h.petStore.ModifyPet(petID, func(pet *core.DigitalPet) error {
		if req.Name != "" {
			pet.Name = strings.TrimSpace(req.Name)
		}
		if req.Location != "" {
			pet.Location = req.Location
		}
		return nil
	})
	if err != nil {
		writeModifyError(w, err, "Failed to update pet")
		return
	}

	writeSuccess(w, convertPetToResponse(pet))
}

// writeModifyError reports why a pet could not be changed
func writeModifyError(w http.ResponseWriter, err error, failed string) {
	var refused refusal
	switch {
	case errors.As(err, &refused):
		writeError(w, http.StatusConflict, "INTERACTION_REFUSED", refused.message)
	case errors.Is(err, ErrNotFound):
		writeError(w, http.StatusNotFound, "PET_NOT_FOUND", "Pet not found")
	case errors.Is(err, ErrNotPermitted):
		writeError(w, http.StatusForbidden, "NOT_PERMITTED", "A parent PIN is set; make this change in the game")
	default:
		writeError(w, http.StatusInternalServerError, "UPDATE_FAILED", failed)
	}
}

// refusal is the pet turning down an interaction
type refusal struct{ message string }

func (r refusal) Error() string { return r.message }

// validLevel reports whether an intensity is in the range 0 to 1
func validLevel(x float64) bool {
	return x >= 0 && x <= 1 // False for NaN
}

// interact applies an interaction to a pet through the store and writes
// the response
func (h *Handler) interact(w http.ResponseWriter, petID types.PetID, interactionType types.InteractionType,
	intensity float64, fallback string) {
	if !validLevel(intensity) {
		writeError(w, http.StatusBadRequest, "INVALID_INTENSITY", "Intensity must be from 0.0 to 1.0")
		return
	}
	var before core.InteractionResult
	var health, energy, happiness, joy float64
	pet, err := h.petStore.ModifyPet(petID, func(pet *core.DigitalPet) error {
		health, energy = pet.Biology.Vitals.Health, pet.Biology.Vitals.Energy
		happiness, joy = pet.Biology.Vitals.Happiness, pet.Emotions.Joy
		before = pet.Interact(interactionType, intensity)
		if !before.Applied {
			message := before.Refusal
			if message == "" {
				message = pet.Name + " can't do that right now."
			}
			return refusal{message}
		}
		return nil
	})
	if err != nil {
		writeModifyError(w, err, "Failed to save interaction")
		return
	}

	writeSuccess(w, InteractionResponse{
		Success:     true,
		PetReaction: pet.CurrentBehavior.String(),
		VitalChanges: map[string]float64{
			"health":    pet.Biology.Vitals.Health - health,
			"energy":    pet.Biology.Vitals.Energy - energy,
			"happiness": pet.Biology.Vitals.Happiness - happiness,
		},
		EmotionChanges: map[string]float64{
			"joy": pet.Emotions.Joy - joy,
		},
		MemoryCreated: true,
		Message:       interactionMessage(before, fallback),
	})
}

// HandleDeletePet handles DELETE /api/v1/pets/{petId}
func (h *Handler) HandleDeletePet(w http.ResponseWriter, r *http.Request) {
	petID := types.PetID(getPathParam(r, "petId"))

	if err := h.petStore.DeletePet(petID); err != nil {
		if errors.Is(err, ErrNotPermitted) {
			writeError(w, http.StatusForbidden, "NOT_PERMITTED", "Pets can't be deleted through the API")
			return
		}
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
	var req InteractionRequest
	if err := parseJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
		return
	}
	interactionType := mapStringToInteractionType(req.Type)
	if interactionType == -1 {
		writeError(w, http.StatusBadRequest, "INVALID_TYPE", "Invalid interaction type")
		return
	}
	h.interact(w, types.PetID(getPathParam(r, "petId")), interactionType, req.Intensity, "Interaction successful")
}

// HandleFeed handles POST /api/v1/pets/{petId}/interactions/feed
func (h *Handler) HandleFeed(w http.ResponseWriter, r *http.Request) {
	var req FeedingRequest
	if err := parseJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
		return
	}
	h.interact(w, types.PetID(getPathParam(r, "petId")), types.InteractionFeeding, req.Quantity, "Pet has been fed")
}

// HandlePlay handles POST /api/v1/pets/{petId}/interactions/play
func (h *Handler) HandlePlay(w http.ResponseWriter, r *http.Request) {
	var req PlayRequest
	if err := parseJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
		return
	}
	h.interact(w, types.PetID(getPathParam(r, "petId")), types.InteractionPlaying, req.Enthusiasm, "Played with pet")
}

// convertPetToResponse converts a DigitalPet to a PetResponse
func convertPetToResponse(pet *core.DigitalPet) *PetResponse {
	return &PetResponse{
		ID:              pet.ID,
		Name:            pet.Name,
		Owner:           pet.Owner,
		Species:         string(pet.GetSpecies()),
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

// interactionMessage describes how the pet reacted, or returns fallback
func interactionMessage(result core.InteractionResult, fallback string) string {
	if result.ReactionText != "" {
		return result.ReactionText
	}
	return fallback
}

// mapStringToInteractionType converts a string to InteractionType
func mapStringToInteractionType(s string) types.InteractionType {
	switch s {
	case "feeding", "feed":
		return types.InteractionFeeding
	case "drinking", "drink":
		return types.InteractionDrinking
	case "cleaning", "clean":
		return types.InteractionCleaning
	case "walking", "walk":
		return types.InteractionWalking
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
