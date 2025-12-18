package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
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

func (m *MockPetStore) UpdatePet(pet *core.DigitalPet) error {
	m.pets[pet.ID] = pet
	return nil
}

func (m *MockPetStore) DeletePet(id types.PetID) error {
	if _, ok := m.pets[id]; !ok {
		return http.ErrNoLocation
	}
	delete(m.pets, id)
	return nil
}

// MockUserStore is a mock implementation of UserStore
type MockUserStore struct {
	users map[types.UserID]*User
}

func NewMockUserStore() *MockUserStore {
	return &MockUserStore{
		users: make(map[types.UserID]*User),
	}
}

func (m *MockUserStore) GetUser(id types.UserID) (*User, error) {
	user, ok := m.users[id]
	if !ok {
		return nil, http.ErrNoLocation
	}
	return user, nil
}

func (m *MockUserStore) GetUserByEmail(email string) (*User, error) {
	for _, user := range m.users {
		if user.Email == email {
			return user, nil
		}
	}
	return nil, http.ErrNoLocation
}

func (m *MockUserStore) CreateUser(user *User) error {
	m.users[user.ID] = user
	return nil
}

func (m *MockUserStore) UpdateUser(user *User) error {
	m.users[user.ID] = user
	return nil
}

func (m *MockUserStore) ValidateCredentials(email, password string) (*User, error) {
	for _, user := range m.users {
		if user.Email == email {
			return user, nil
		}
	}
	return nil, http.ErrNoLocation
}

// MockSocialService is a mock implementation of SocialService
type MockSocialService struct {
	nearbyPets []*NearbyPetResponse
	friends    map[types.PetID][]*FriendResponse
}

func NewMockSocialService() *MockSocialService {
	return &MockSocialService{
		nearbyPets: make([]*NearbyPetResponse, 0),
		friends:    make(map[types.PetID][]*FriendResponse),
	}
}

func (m *MockSocialService) GetNearbyPets(location *Location, radiusMeters float64) ([]*NearbyPetResponse, error) {
	return m.nearbyPets, nil
}

func (m *MockSocialService) GetFriends(petID types.PetID) ([]*FriendResponse, error) {
	return m.friends[petID], nil
}

func (m *MockSocialService) SendFriendRequest(fromPetID, toPetID types.PetID, message string) error {
	return nil
}

func (m *MockSocialService) AcceptFriendRequest(requestID string) error {
	return nil
}

func (m *MockSocialService) DeclineFriendRequest(requestID string) error {
	return nil
}

func (m *MockSocialService) RemoveFriend(petID, friendID types.PetID) error {
	return nil
}

// MockSyncService is a mock implementation of SyncService
type MockSyncService struct{}

func NewMockSyncService() *MockSyncService {
	return &MockSyncService{}
}

func (m *MockSyncService) Sync(userID types.UserID, request *SyncRequest) (*SyncResponse, error) {
	return &SyncResponse{
		Status:        "up_to_date",
		ServerVersion: 1,
		LastSyncTime:  time.Now(),
	}, nil
}

func (m *MockSyncService) ResolveConflict(userID types.UserID, request *ConflictResolutionRequest) error {
	return nil
}

// ============================================================================
// Helper Functions
// ============================================================================

func createTestHandler() *Handler {
	return NewHandler(
		NewMockPetStore(),
		NewMockUserStore(),
		NewMockSocialService(),
		NewMockSyncService(),
	)
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

func TestHandleUpdateLocation(t *testing.T) {
	handler := createTestHandler()

	req := LocationUpdateRequest{
		Latitude:  37.7749,
		Longitude: -122.4194,
	}

	rr := makeRequest(handler.HandleUpdateLocation, "PUT", "/api/v1/social/location", req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rr.Code)
	}

	response := parseResponse(rr)
	if !response.Success {
		t.Error("Expected success to be true")
	}
}

func TestHandleUpdateLocationInvalidCoordinates(t *testing.T) {
	handler := createTestHandler()

	// Test invalid latitude
	req := LocationUpdateRequest{
		Latitude:  100, // Invalid
		Longitude: -122.4194,
	}

	rr := makeRequest(handler.HandleUpdateLocation, "PUT", "/api/v1/social/location", req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", rr.Code)
	}

	response := parseResponse(rr)
	if response.Error.Code != "INVALID_COORDINATES" {
		t.Errorf("Expected error code INVALID_COORDINATES, got %s", response.Error.Code)
	}
}

func TestHandleGetNearbyPets(t *testing.T) {
	handler := createTestHandler()

	req := httptest.NewRequest("GET", "/api/v1/social/nearby?latitude=37.7749&longitude=-122.4194", nil)
	rr := httptest.NewRecorder()

	handler.HandleGetNearbyPets(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rr.Code)
	}
}

func TestHandleGetNearbyPetsMissingLocation(t *testing.T) {
	handler := createTestHandler()

	req := httptest.NewRequest("GET", "/api/v1/social/nearby", nil)
	rr := httptest.NewRecorder()

	handler.HandleGetNearbyPets(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", rr.Code)
	}
}

func TestHandleSync(t *testing.T) {
	handler := createTestHandler()

	req := SyncRequest{
		LastSyncTime: time.Now().Add(-1 * time.Hour),
		LocalVersion: 1,
	}

	rr := makeRequest(handler.HandleSync, "POST", "/api/v1/sync", req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rr.Code)
	}

	response := parseResponse(rr)
	if !response.Success {
		t.Error("Expected success to be true")
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
