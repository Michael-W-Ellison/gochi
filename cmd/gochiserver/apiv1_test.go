package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// apiRequest sends a request to the REST API with the test token
func apiRequest(t *testing.T, handler http.Handler, method, path, body, token string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	return rec
}

func TestRESTAPI(t *testing.T) {
	server := newTestServer(t)
	if err := server.createPet("Biscuit", false, false); err != nil {
		t.Fatal(err)
	}
	handler := server.apiHandler("secret")
	id := string(server.pet.ID)

	// Pet routes need the token
	if rec := apiRequest(t, handler, http.MethodGet, "/api/v1/pets", "", ""); rec.Code != http.StatusUnauthorized {
		t.Errorf("No token: expected 401, got %d", rec.Code)
	}
	if rec := apiRequest(t, handler, http.MethodGet, "/api/v1/pets", "", "wrong"); rec.Code != http.StatusUnauthorized {
		t.Errorf("Wrong token: expected 401, got %d", rec.Code)
	}

	// Listing and reading pets
	rec := apiRequest(t, handler, http.MethodGet, "/api/v1/pets", "", "secret")
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "Biscuit") {
		t.Errorf("Listing pets: %d %s", rec.Code, rec.Body.String())
	}
	rec = apiRequest(t, handler, http.MethodGet, "/api/v1/pets/"+id, "", "secret")
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "Biscuit") {
		t.Errorf("Reading the pet: %d %s", rec.Code, rec.Body.String())
	}

	// A refused interaction changes nothing
	server.pet.Biology.Vitals.Nutrition = 0.95
	rec = apiRequest(t, handler, http.MethodPost, "/api/v1/pets/"+id+"/interactions", `{"type":"feed","intensity":1}`, "secret")
	if rec.Code != http.StatusConflict || !strings.Contains(rec.Body.String(), "Biscuit is not hungry.") {
		t.Errorf("Feeding a full pet: %d %s", rec.Code, rec.Body.String())
	}

	// An accepted interaction updates the active pet
	server.pet.Biology.Vitals.Hydration = 0.3
	before := server.pet.TotalInteractions
	rec = apiRequest(t, handler, http.MethodPost, "/api/v1/pets/"+id+"/interactions", `{"type":"drink","intensity":1}`, "secret")
	if rec.Code != http.StatusOK {
		t.Fatalf("Giving a drink: %d %s", rec.Code, rec.Body.String())
	}
	if server.pet.TotalInteractions != before+1 || server.pet.Biology.Vitals.Hydration <= 0.3 {
		t.Errorf("The drink should update the active pet")
	}

	// Creating a pet saves it without switching
	rec = apiRequest(t, handler, http.MethodPost, "/api/v1/pets", `{"name":"Pepper"}`, "secret")
	if rec.Code != http.StatusOK || server.pet.Name != "Biscuit" {
		t.Errorf("Creating a pet: %d, active %s", rec.Code, server.pet.Name)
	}
	rec = apiRequest(t, handler, http.MethodGet, "/api/v1/pets", "", "secret")
	var list struct {
		Data []struct {
			Name string `json:"name"`
		} `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &list); err != nil || len(list.Data) != 2 {
		t.Errorf("Both pets should be listed: %s", rec.Body.String())
	}

	// Deleting needs the game's confirmations
	rec = apiRequest(t, handler, http.MethodDelete, "/api/v1/pets/"+id, "", "secret")
	if rec.Code != http.StatusForbidden || server.pet == nil {
		t.Errorf("Deleting through the API: expected 403, got %d", rec.Code)
	}

	// Health needs no token
	if rec := apiRequest(t, handler, http.MethodGet, "/health", "", ""); rec.Code != http.StatusOK {
		t.Errorf("Health check: %d", rec.Code)
	}
}
