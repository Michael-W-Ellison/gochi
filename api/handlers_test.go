package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Michael-W-Ellison/gochi/internal/core"
	"github.com/Michael-W-Ellison/gochi/pkg/types"
)

// ============================================================================
// Mock Implementations
// ============================================================================

// MockPetStore is a mock implementation of PetStore
type MockPetStore struct {
	pets map[types.PetID]*core.DigitalPet
}

func NewMockPetStore() *MockPetStore {
	return &MockPetStore{
		pets: make(map[types.PetID]*core.DigitalPet),
	}
}

func (m *MockPetStore) GetPet(id types.PetID) (*core.DigitalPet, error) {
	pet, ok := m.pets[id]
	if !ok {
		return nil, http.ErrNoLocation
	}
	return pet, nil
}

func (m *MockPetStore) GetPetsByOwner(ownerID types.UserID) ([]*core.DigitalPet, error) {
	var result []*core.DigitalPet
	for _, pet := range m.pets {
		if pet.Owner == ownerID {
			result = append(result, pet)
		}
	}
	return result, nil
}

func (m *MockPetStore) CreatePet(pet *core.DigitalPet) error {
	m.pets[pet.ID] = pet
	return nil
}

func (m *MockPetStore) ModifyPet(id types.PetID, change func(*core.DigitalPet) error) (*core.DigitalPet, error) {
	pet, ok := m.pets[id]
	if !ok {
		return nil, ErrNotFound
	}
	if err := change(pet); err != nil {
		return nil, err
	}
	return pet, nil
}

func (m *MockPetStore) DeletePet(id types.PetID) error {
	if _, ok := m.pets[id]; !ok {
		return http.ErrNoLocation
	}
	delete(m.pets, id)
	return nil
}

// ============================================================================
// Helper Functions
// ============================================================================

func createTestHandler() *Handler {
	return NewHandler(NewMockPetStore())
}

func makeRequest(handler http.HandlerFunc, method, path string, body interface{}) *httptest.ResponseRecorder {
	var reqBody *bytes.Buffer
	if body != nil {
		jsonBody, _ := json.Marshal(body)
		reqBody = bytes.NewBuffer(jsonBody)
	} else {
		reqBody = bytes.NewBuffer(nil)
	}

	req := httptest.NewRequest(method, path, reqBody)
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	return rr
}

func parseResponse(rr *httptest.ResponseRecorder) APIResponse {
	var response APIResponse
	json.Unmarshal(rr.Body.Bytes(), &response)
	return response
}

// ============================================================================
// Tests
// ============================================================================

func TestWriteJSON(t *testing.T) {
	w := httptest.NewRecorder()
	data := map[string]string{"key": "value"}

	writeJSON(w, http.StatusOK, data)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	contentType := w.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("Expected Content-Type application/json, got %s", contentType)
	}
}

func TestWriteSuccess(t *testing.T) {
	w := httptest.NewRecorder()
	data := map[string]string{"message": "success"}

	writeSuccess(w, data)

	response := parseResponse(w)
	if !response.Success {
		t.Error("Expected success to be true")
	}
}

func TestWriteError(t *testing.T) {
	w := httptest.NewRecorder()

	writeError(w, http.StatusBadRequest, "TEST_ERROR", "Test error message")

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}

	response := parseResponse(w)
	if response.Success {
		t.Error("Expected success to be false")
	}
	if response.Error == nil {
		t.Error("Expected error to be present")
	}
	if response.Error.Code != "TEST_ERROR" {
		t.Errorf("Expected error code TEST_ERROR, got %s", response.Error.Code)
	}
}

func TestHandleCreatePet(t *testing.T) {
	handler := createTestHandler()

	// Test with valid request
	req := CreatePetRequest{
		Name:            "TestPet",
		RandomizeTraits: false,
	}

	rr := makeRequest(handler.HandleCreatePet, "POST", "/api/v1/pets", req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rr.Code)
	}

	response := parseResponse(rr)
	if !response.Success {
		t.Error("Expected success to be true")
	}
}

func TestHandleCreatePetMissingName(t *testing.T) {
	handler := createTestHandler()

	req := CreatePetRequest{
		Name: "",
	}

	rr := makeRequest(handler.HandleCreatePet, "POST", "/api/v1/pets", req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", rr.Code)
	}

	response := parseResponse(rr)
	if response.Success {
		t.Error("Expected success to be false")
	}
	if response.Error.Code != "MISSING_NAME" {
		t.Errorf("Expected error code MISSING_NAME, got %s", response.Error.Code)
	}
}

func TestHandleCreatePetInvalidJSON(t *testing.T) {
	handler := createTestHandler()

	req := httptest.NewRequest("POST", "/api/v1/pets", bytes.NewBufferString("invalid json"))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	handler.HandleCreatePet(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", rr.Code)
	}
}

func TestHandleGetPets(t *testing.T) {
	handler := createTestHandler()

	rr := makeRequest(handler.HandleGetPets, "GET", "/api/v1/pets", nil)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rr.Code)
	}

	response := parseResponse(rr)
	if !response.Success {
		t.Error("Expected success to be true")
	}
}

func TestHandleGetPetNotFound(t *testing.T) {
	handler := createTestHandler()

	req := httptest.NewRequest("GET", "/api/v1/pets/nonexistent", nil)
	req.SetPathValue("petId", "nonexistent")
	rr := httptest.NewRecorder()

	handler.HandleGetPet(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", rr.Code)
	}
}

func TestMapStringToInteractionType(t *testing.T) {
	tests := []struct {
		input    string
		expected types.InteractionType
	}{
		{"feeding", types.InteractionFeeding},
		{"feed", types.InteractionFeeding},
		{"petting", types.InteractionPetting},
		{"pet", types.InteractionPetting},
		{"playing", types.InteractionPlaying},
		{"play", types.InteractionPlaying},
		{"training", types.InteractionTraining},
		{"train", types.InteractionTraining},
		{"grooming", types.InteractionGrooming},
		{"groom", types.InteractionGrooming},
		{"medical", types.InteractionMedicalCare},
		{"medical_care", types.InteractionMedicalCare},
		{"reward", types.InteractionRewards},
		{"rewards", types.InteractionRewards},
		{"discipline", types.InteractionDiscipline},
		{"invalid", types.InteractionType(-1)},
	}

	for _, tt := range tests {
		result := mapStringToInteractionType(tt.input)
		if result != tt.expected {
			t.Errorf("mapStringToInteractionType(%s) = %d, expected %d", tt.input, result, tt.expected)
		}
	}
}

func TestGetQueryParam(t *testing.T) {
	req := httptest.NewRequest("GET", "/test?foo=bar", nil)

	value := getQueryParam(req, "foo", "default")
	if value != "bar" {
		t.Errorf("Expected 'bar', got '%s'", value)
	}

	defaultValue := getQueryParam(req, "missing", "default")
	if defaultValue != "default" {
		t.Errorf("Expected 'default', got '%s'", defaultValue)
	}
}

func TestGetQueryParamInt(t *testing.T) {
	req := httptest.NewRequest("GET", "/test?num=42&invalid=abc", nil)

	value := getQueryParamInt(req, "num", 0)
	if value != 42 {
		t.Errorf("Expected 42, got %d", value)
	}

	defaultValue := getQueryParamInt(req, "missing", 10)
	if defaultValue != 10 {
		t.Errorf("Expected 10, got %d", defaultValue)
	}

	invalidValue := getQueryParamInt(req, "invalid", 5)
	if invalidValue != 5 {
		t.Errorf("Expected 5 for invalid int, got %d", invalidValue)
	}
}

func TestGetQueryParamFloat(t *testing.T) {
	req := httptest.NewRequest("GET", "/test?num=3.14&invalid=abc", nil)

	value := getQueryParamFloat(req, "num", 0)
	if value != 3.14 {
		t.Errorf("Expected 3.14, got %f", value)
	}

	defaultValue := getQueryParamFloat(req, "missing", 1.5)
	if defaultValue != 1.5 {
		t.Errorf("Expected 1.5, got %f", defaultValue)
	}

	invalidValue := getQueryParamFloat(req, "invalid", 2.5)
	if invalidValue != 2.5 {
		t.Errorf("Expected 2.5 for invalid float, got %f", invalidValue)
	}
}

// ============================================================================
// API Response Tests
// ============================================================================

func TestAPIResponseJSON(t *testing.T) {
	response := APIResponse{
		Success:   true,
		Data:      map[string]string{"key": "value"},
		Timestamp: time.Now(),
	}

	jsonBytes, err := json.Marshal(response)
	if err != nil {
		t.Fatalf("Failed to marshal APIResponse: %v", err)
	}

	var decoded APIResponse
	if err := json.Unmarshal(jsonBytes, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal APIResponse: %v", err)
	}

	if decoded.Success != response.Success {
		t.Error("Success field mismatch")
	}
}

func TestPaginatedResponseJSON(t *testing.T) {
	response := PaginatedResponse{
		Items:      []string{"item1", "item2"},
		TotalCount: 100,
		Page:       1,
		PageSize:   10,
		HasMore:    true,
	}

	jsonBytes, err := json.Marshal(response)
	if err != nil {
		t.Fatalf("Failed to marshal PaginatedResponse: %v", err)
	}

	var decoded PaginatedResponse
	if err := json.Unmarshal(jsonBytes, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal PaginatedResponse: %v", err)
	}

	if decoded.TotalCount != 100 {
		t.Errorf("Expected TotalCount 100, got %d", decoded.TotalCount)
	}
	if !decoded.HasMore {
		t.Error("Expected HasMore to be true")
	}
}

func TestEmptyListsAreNotNull(t *testing.T) {
	handler := createTestHandler()

	for name, run := range map[string]func(*httptest.ResponseRecorder){
		"pets": func(rr *httptest.ResponseRecorder) {
			handler.HandleGetPets(rr, httptest.NewRequest("GET", "/api/v1/pets", nil))
		},
	} {
		rr := httptest.NewRecorder()
		run(rr)
		if rr.Code != http.StatusOK || !strings.Contains(rr.Body.String(), `"data":[]`) {
			t.Errorf("%s: an empty list should be [], got %d %s", name, rr.Code, rr.Body.String())
		}
	}
}

// apiCall sends a request with a body to a handler for one pet
func apiCall(handler http.HandlerFunc, method, petID, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, "/api/v1/pets/"+petID, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("petId", petID)
	rr := httptest.NewRecorder()
	handler(rr, req)
	return rr
}

func TestIntensityMustBeInRange(t *testing.T) {
	store := NewMockPetStore()
	handler := NewHandler(store)
	pet := core.NewDigitalPet("Mochi", "owner")
	store.pets[pet.ID] = pet
	id := string(pet.ID)
	for _, call := range []struct {
		handler http.HandlerFunc
		body    string
	}{
		{handler.HandleInteraction, `{"type":"petting","intensity":1.7e308}`},
		{handler.HandleInteraction, `{"type":"petting","intensity":-1}`},
		{handler.HandleFeed, `{"quantity":2}`},
		{handler.HandlePlay, `{"enthusiasm":-0.5}`},
	} {
		if rr := apiCall(call.handler, "POST", id, call.body); rr.Code != http.StatusBadRequest {
			t.Errorf("%s: %d %s", call.body, rr.Code, rr.Body.String())
		}
	}
	if pet.TotalInteractions != 0 {
		t.Error("Refused interactions should change nothing")
	}
	if rr := apiCall(handler.HandleInteraction, "POST", id, `{"type":"petting","intensity":0.5}`); rr.Code != http.StatusOK {
		t.Errorf("A valid interaction: %d %s", rr.Code, rr.Body.String())
	}
	if rr := apiCall(handler.HandleInteraction, "POST", "pet_missing", `{"type":"petting","intensity":0.5}`); rr.Code != http.StatusNotFound {
		t.Errorf("A missing pet: %d", rr.Code)
	}
}

func TestLocationMustBeAPlace(t *testing.T) {
	store := NewMockPetStore()
	handler := NewHandler(store)
	pet := core.NewDigitalPet("Mochi", "owner")
	store.pets[pet.ID] = pet
	if rr := apiCall(handler.HandleUpdatePet, "PUT", string(pet.ID), `{"location":"<script>"}`); rr.Code != http.StatusBadRequest {
		t.Errorf("An unknown location should be refused: %d", rr.Code)
	}
	if rr := apiCall(handler.HandleUpdatePet, "PUT", string(pet.ID), `{"location":"park"}`); rr.Code != http.StatusOK || pet.Location != "park" {
		t.Errorf("A place should be accepted: %d %q", rr.Code, pet.Location)
	}
}
