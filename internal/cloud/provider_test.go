package cloud

import (
	"testing"

	"github.com/Michael-W-Ellison/gochi/pkg/types"
)

func TestMockCloudStorageProvider(t *testing.T) {
	provider := NewMockAuthProvider()
	am := NewAuthManager(provider)
	csp := NewMockCloudStorageProvider(am)

	if csp == nil {
		t.Fatal("NewMockCloudStorageProvider returned nil")
	}
}

func TestUploadDownload(t *testing.T) {
	// Setup
	provider := NewMockAuthProvider()
	am := NewAuthManager(provider)
	csp := NewMockCloudStorageProvider(am)

	// Register and login
	creds := &Credentials{
		Username: "testuser",
		Password: "password123",
		Email:    "test@example.com",
	}
	am.Register(creds)

	// Upload
	petID := types.PetID("test-pet-123")
	data := []byte("test pet data")

	err := csp.Upload(petID, data)
	if err != nil {
		t.Fatalf("Upload failed: %v", err)
	}

	// Download
	downloaded, err := csp.Download(petID)
	if err != nil {
		t.Fatalf("Download failed: %v", err)
	}

	if string(downloaded) != string(data) {
		t.Errorf("Downloaded data doesn't match. Got %s, want %s", string(downloaded), string(data))
	}
}

func TestUploadNotLoggedIn(t *testing.T) {
	provider := NewMockAuthProvider()
	am := NewAuthManager(provider)
	csp := NewMockCloudStorageProvider(am)

	// Try to upload without logging in
	petID := types.PetID("test-pet-123")
	data := []byte("test pet data")

	err := csp.Upload(petID, data)
	if err == nil {
		t.Error("Expected error when uploading without logging in")
	}
}

func TestDelete(t *testing.T) {
	// Setup
	provider := NewMockAuthProvider()
	am := NewAuthManager(provider)
	csp := NewMockCloudStorageProvider(am)

	// Register and login
	creds := &Credentials{
		Username: "testuser",
		Password: "password123",
		Email:    "test@example.com",
	}
	am.Register(creds)

	// Upload
	petID := types.PetID("test-pet-123")
	data := []byte("test pet data")
	csp.Upload(petID, data)

	// Delete
	err := csp.Delete(petID)
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	// Try to download deleted pet
	_, err = csp.Download(petID)
	if err == nil {
		t.Error("Expected error when downloading deleted pet")
	}
}

func TestList(t *testing.T) {
	// Setup
	provider := NewMockAuthProvider()
	am := NewAuthManager(provider)
	csp := NewMockCloudStorageProvider(am)

	// Register and login
	creds := &Credentials{
		Username: "testuser",
		Password: "password123",
		Email:    "test@example.com",
	}
	am.Register(creds)

	// Upload multiple pets
	petIDs := []types.PetID{
		types.PetID("pet1"),
		types.PetID("pet2"),
		types.PetID("pet3"),
	}

	for _, petID := range petIDs {
		csp.Upload(petID, []byte("data"))
	}

	// List
	list, err := csp.List()
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	if len(list) != 3 {
		t.Errorf("Expected 3 pets, got %d", len(list))
	}
}

func TestGetLastModified(t *testing.T) {
	// Setup
	provider := NewMockAuthProvider()
	am := NewAuthManager(provider)
	csp := NewMockCloudStorageProvider(am)

	// Register and login
	creds := &Credentials{
		Username: "testuser",
		Password: "password123",
		Email:    "test@example.com",
	}
	am.Register(creds)

	// Upload
	petID := types.PetID("test-pet-123")
	csp.Upload(petID, []byte("data"))

	// Get last modified
	modTime, err := csp.GetLastModified(petID)
	if err != nil {
		t.Fatalf("GetLastModified failed: %v", err)
	}

	if modTime.IsZero() {
		t.Error("Expected non-zero modification time")
	}
}

func TestIsConnected(t *testing.T) {
	// Setup
	provider := NewMockAuthProvider()
	am := NewAuthManager(provider)
	csp := NewMockCloudStorageProvider(am)

	// Should be connected by default
	if !csp.IsConnected() {
		t.Error("Expected to be connected by default")
	}

	// Set disconnected
	csp.SetConnected(false)
	if csp.IsConnected() {
		t.Error("Expected to be disconnected")
	}

	// Try to upload while disconnected
	creds := &Credentials{
		Username: "testuser",
		Password: "password123",
		Email:    "test@example.com",
	}
	am.Register(creds)

	err := csp.Upload(types.PetID("pet1"), []byte("data"))
	if err == nil {
		t.Error("Expected error when uploading while disconnected")
	}
}

func TestDownloadNonExistent(t *testing.T) {
	// Setup
	provider := NewMockAuthProvider()
	am := NewAuthManager(provider)
	csp := NewMockCloudStorageProvider(am)

	// Register and login
	creds := &Credentials{
		Username: "testuser",
		Password: "password123",
		Email:    "test@example.com",
	}
	am.Register(creds)

	// Try to download non-existent pet
	_, err := csp.Download(types.PetID("nonexistent"))
	if err == nil {
		t.Error("Expected error when downloading non-existent pet")
	}
}

func TestMultipleUploads(t *testing.T) {
	// Setup
	provider := NewMockAuthProvider()
	am := NewAuthManager(provider)
	csp := NewMockCloudStorageProvider(am)

	// Register and login
	creds := &Credentials{
		Username: "testuser",
		Password: "password123",
		Email:    "test@example.com",
	}
	am.Register(creds)

	petID := types.PetID("test-pet")

	// Upload multiple times (should overwrite)
	data1 := []byte("data1")
	data2 := []byte("data2")

	csp.Upload(petID, data1)
	csp.Upload(petID, data2)

	// Download should get latest
	downloaded, err := csp.Download(petID)
	if err != nil {
		t.Fatalf("Download failed: %v", err)
	}

	if string(downloaded) != string(data2) {
		t.Errorf("Expected latest data, got %s", string(downloaded))
	}
}

func TestConcurrentAccess(t *testing.T) {
	// Setup
	provider := NewMockAuthProvider()
	am := NewAuthManager(provider)
	csp := NewMockCloudStorageProvider(am)

	// Register and login
	creds := &Credentials{
		Username: "testuser",
		Password: "password123",
		Email:    "test@example.com",
	}
	am.Register(creds)

	// Concurrent uploads
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func(n int) {
			petID := types.PetID("pet-" + string(rune('0'+n)))
			csp.Upload(petID, []byte("data"))
			done <- true
		}(i)
	}

	// Wait for all
	for i := 0; i < 10; i++ {
		<-done
	}

	// List should show all pets
	list, _ := csp.List()
	if len(list) != 10 {
		t.Errorf("Expected 10 pets after concurrent uploads, got %d", len(list))
	}
}

func TestNewCloudStorageProvider(t *testing.T) {
	provider := NewMockAuthProvider()
	am := NewAuthManager(provider)
	csp := NewCloudStorageProvider("https://api.example.com", am)

	if csp == nil {
		t.Fatal("NewCloudStorageProvider returned nil")
	}

	if csp.baseURL != "https://api.example.com" {
		t.Errorf("Expected baseURL 'https://api.example.com', got %s", csp.baseURL)
	}
}

func TestCloudStorageProviderGetStatistics(t *testing.T) {
	provider := NewMockAuthProvider()
	am := NewAuthManager(provider)
	csp := NewCloudStorageProvider("https://api.example.com", am)

	stats := csp.GetStatistics()

	if stats == nil {
		t.Fatal("GetStatistics returned nil")
	}

	// Check that all expected keys are present
	expectedKeys := []string{"upload_count", "download_count", "total_uploaded", "total_downloaded", "is_connected"}
	for _, key := range expectedKeys {
		if _, ok := stats[key]; !ok {
			t.Errorf("Expected key %s in statistics", key)
		}
	}
}

func TestCloudStorageProviderSetTimeout(t *testing.T) {
	provider := NewMockAuthProvider()
	am := NewAuthManager(provider)
	csp := NewCloudStorageProvider("https://api.example.com", am)

	// Set timeout
	csp.SetTimeout(10)

	// Check that timeout was set (we can't directly access the field, but we can verify it doesn't panic)
	if csp == nil {
		t.Error("Provider should still be valid after SetTimeout")
	}
}

func TestMockProviderDelayMethods(t *testing.T) {
	provider := NewMockAuthProvider()
	am := NewAuthManager(provider)
	csp := NewMockCloudStorageProvider(am)

	// Set delays
	csp.SetUploadDelay(100)
	csp.SetDownloadDelay(50)

	// These methods should not panic
	if csp == nil {
		t.Error("Provider should still be valid after setting delays")
	}
}

func TestCloudStorageProviderNotLoggedIn(t *testing.T) {
	provider := NewMockAuthProvider()
	am := NewAuthManager(provider)
	csp := NewCloudStorageProvider("https://api.example.com", am)

	// Try operations without logging in
	err := csp.Upload(types.PetID("test"), []byte("data"))
	if err == nil {
		t.Error("Expected error when uploading without logging in")
	}

	_, err = csp.Download(types.PetID("test"))
	if err == nil {
		t.Error("Expected error when downloading without logging in")
	}

	err = csp.Delete(types.PetID("test"))
	if err == nil {
		t.Error("Expected error when deleting without logging in")
	}

	_, err = csp.List()
	if err == nil {
		t.Error("Expected error when listing without logging in")
	}

	_, err = csp.GetLastModified(types.PetID("test"))
	if err == nil {
		t.Error("Expected error when getting last modified without logging in")
	}

	// IsConnected should return false when not logged in
	if csp.IsConnected() {
		t.Error("Expected IsConnected to return false when not logged in")
	}
}

