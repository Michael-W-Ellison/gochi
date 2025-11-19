package cloud

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/Michael-W-Ellison/gochi/pkg/types"
)

// CloudStorageProvider implements cloud storage for pet data
type CloudStorageProvider struct {
	mu sync.RWMutex

	baseURL     string
	authManager *AuthManager
	httpClient  *http.Client
	timeout     time.Duration

	// Statistics
	uploadCount   int64
	downloadCount int64
	totalUploaded int64
	totalDownloaded int64
	lastError     error
}

// NewCloudStorageProvider creates a new cloud storage provider
func NewCloudStorageProvider(baseURL string, authManager *AuthManager) *CloudStorageProvider {
	return &CloudStorageProvider{
		baseURL:     baseURL,
		authManager: authManager,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		timeout: 30 * time.Second,
	}
}

// Upload uploads pet data to the cloud
func (csp *CloudStorageProvider) Upload(petID types.PetID, data []byte) error {
	csp.mu.Lock()
	defer csp.mu.Unlock()

	if !csp.authManager.IsLoggedIn() {
		return errors.New("not logged in")
	}

	session, err := csp.authManager.GetSession()
	if err != nil {
		return err
	}

	// Create request
	url := fmt.Sprintf("%s/pets/%s", csp.baseURL, petID)
	req, err := http.NewRequest("PUT", url, bytes.NewReader(data))
	if err != nil {
		csp.lastError = err
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Add authentication header
	req.Header.Set("Authorization", "Bearer "+session.Token)
	req.Header.Set("Content-Type", "application/json")

	// Send request
	resp, err := csp.httpClient.Do(req)
	if err != nil {
		csp.lastError = err
		return fmt.Errorf("failed to upload: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		csp.lastError = fmt.Errorf("upload failed with status %d: %s", resp.StatusCode, string(body))
		return csp.lastError
	}

	// Update statistics
	csp.uploadCount++
	csp.totalUploaded += int64(len(data))

	return nil
}

// Download retrieves pet data from the cloud
func (csp *CloudStorageProvider) Download(petID types.PetID) ([]byte, error) {
	csp.mu.Lock()
	defer csp.mu.Unlock()

	if !csp.authManager.IsLoggedIn() {
		return nil, errors.New("not logged in")
	}

	session, err := csp.authManager.GetSession()
	if err != nil {
		return nil, err
	}

	// Create request
	url := fmt.Sprintf("%s/pets/%s", csp.baseURL, petID)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		csp.lastError = err
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Add authentication header
	req.Header.Set("Authorization", "Bearer "+session.Token)

	// Send request
	resp, err := csp.httpClient.Do(req)
	if err != nil {
		csp.lastError = err
		return nil, fmt.Errorf("failed to download: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("pet %s not found in cloud", petID)
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		csp.lastError = fmt.Errorf("download failed with status %d: %s", resp.StatusCode, string(body))
		return nil, csp.lastError
	}

	// Read response body
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		csp.lastError = err
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Update statistics
	csp.downloadCount++
	csp.totalDownloaded += int64(len(data))

	return data, nil
}

// Delete removes pet data from the cloud
func (csp *CloudStorageProvider) Delete(petID types.PetID) error {
	csp.mu.Lock()
	defer csp.mu.Unlock()

	if !csp.authManager.IsLoggedIn() {
		return errors.New("not logged in")
	}

	session, err := csp.authManager.GetSession()
	if err != nil {
		return err
	}

	// Create request
	url := fmt.Sprintf("%s/pets/%s", csp.baseURL, petID)
	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		csp.lastError = err
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Add authentication header
	req.Header.Set("Authorization", "Bearer "+session.Token)

	// Send request
	resp, err := csp.httpClient.Do(req)
	if err != nil {
		csp.lastError = err
		return fmt.Errorf("failed to delete: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		csp.lastError = fmt.Errorf("delete failed with status %d: %s", resp.StatusCode, string(body))
		return csp.lastError
	}

	return nil
}

// List returns all pet IDs stored in the cloud
func (csp *CloudStorageProvider) List() ([]types.PetID, error) {
	csp.mu.Lock()
	defer csp.mu.Unlock()

	if !csp.authManager.IsLoggedIn() {
		return nil, errors.New("not logged in")
	}

	session, err := csp.authManager.GetSession()
	if err != nil {
		return nil, err
	}

	// Create request
	url := fmt.Sprintf("%s/pets", csp.baseURL)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		csp.lastError = err
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Add authentication header
	req.Header.Set("Authorization", "Bearer "+session.Token)

	// Send request
	resp, err := csp.httpClient.Do(req)
	if err != nil {
		csp.lastError = err
		return nil, fmt.Errorf("failed to list pets: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		csp.lastError = fmt.Errorf("list failed with status %d: %s", resp.StatusCode, string(body))
		return nil, csp.lastError
	}

	// Parse response
	var petIDs []types.PetID
	if err := json.NewDecoder(resp.Body).Decode(&petIDs); err != nil {
		csp.lastError = err
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return petIDs, nil
}

// GetLastModified returns when a pet was last modified in the cloud
func (csp *CloudStorageProvider) GetLastModified(petID types.PetID) (time.Time, error) {
	csp.mu.Lock()
	defer csp.mu.Unlock()

	if !csp.authManager.IsLoggedIn() {
		return time.Time{}, errors.New("not logged in")
	}

	session, err := csp.authManager.GetSession()
	if err != nil {
		return time.Time{}, err
	}

	// Create request
	url := fmt.Sprintf("%s/pets/%s/metadata", csp.baseURL, petID)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		csp.lastError = err
		return time.Time{}, fmt.Errorf("failed to create request: %w", err)
	}

	// Add authentication header
	req.Header.Set("Authorization", "Bearer "+session.Token)

	// Send request
	resp, err := csp.httpClient.Do(req)
	if err != nil {
		csp.lastError = err
		return time.Time{}, fmt.Errorf("failed to get metadata: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return time.Time{}, fmt.Errorf("metadata not found")
	}

	// Parse response
	var metadata struct {
		LastModified time.Time `json:"last_modified"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&metadata); err != nil {
		csp.lastError = err
		return time.Time{}, fmt.Errorf("failed to parse metadata: %w", err)
	}

	return metadata.LastModified, nil
}

// IsConnected checks if the cloud service is reachable
func (csp *CloudStorageProvider) IsConnected() bool {
	csp.mu.RLock()
	defer csp.mu.RUnlock()

	if !csp.authManager.IsLoggedIn() {
		return false
	}

	// Try to ping the server
	url := fmt.Sprintf("%s/health", csp.baseURL)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return false
	}

	resp, err := csp.httpClient.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	return resp.StatusCode == http.StatusOK
}

// GetStatistics returns cloud storage statistics
func (csp *CloudStorageProvider) GetStatistics() map[string]interface{} {
	csp.mu.RLock()
	defer csp.mu.RUnlock()

	stats := map[string]interface{}{
		"upload_count":      csp.uploadCount,
		"download_count":    csp.downloadCount,
		"total_uploaded":    csp.totalUploaded,
		"total_downloaded":  csp.totalDownloaded,
		"is_connected":      csp.IsConnected(),
	}

	if csp.lastError != nil {
		stats["last_error"] = csp.lastError.Error()
	}

	return stats
}

// SetTimeout sets the HTTP client timeout
func (csp *CloudStorageProvider) SetTimeout(timeout time.Duration) {
	csp.mu.Lock()
	defer csp.mu.Unlock()

	csp.timeout = timeout
	csp.httpClient.Timeout = timeout
}

// MockCloudStorageProvider is an in-memory cloud storage provider for testing
type MockCloudStorageProvider struct {
	mu sync.RWMutex

	authManager  *AuthManager
	storage      map[types.PetID][]byte
	metadata     map[types.PetID]time.Time
	connected    bool
	uploadDelay  time.Duration
	downloadDelay time.Duration
}

// NewMockCloudStorageProvider creates a new mock cloud storage provider
func NewMockCloudStorageProvider(authManager *AuthManager) *MockCloudStorageProvider {
	return &MockCloudStorageProvider{
		authManager: authManager,
		storage:     make(map[types.PetID][]byte),
		metadata:    make(map[types.PetID]time.Time),
		connected:   true,
	}
}

// Upload uploads pet data
func (m *MockCloudStorageProvider) Upload(petID types.PetID, data []byte) error {
	if m.uploadDelay > 0 {
		time.Sleep(m.uploadDelay)
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.connected {
		return errors.New("not connected")
	}

	if !m.authManager.IsLoggedIn() {
		return errors.New("not logged in")
	}

	m.storage[petID] = make([]byte, len(data))
	copy(m.storage[petID], data)
	m.metadata[petID] = time.Now()

	return nil
}

// Download retrieves pet data
func (m *MockCloudStorageProvider) Download(petID types.PetID) ([]byte, error) {
	if m.downloadDelay > 0 {
		time.Sleep(m.downloadDelay)
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	if !m.connected {
		return nil, errors.New("not connected")
	}

	if !m.authManager.IsLoggedIn() {
		return nil, errors.New("not logged in")
	}

	data, exists := m.storage[petID]
	if !exists {
		return nil, fmt.Errorf("pet %s not found", petID)
	}

	result := make([]byte, len(data))
	copy(result, data)

	return result, nil
}

// Delete removes pet data
func (m *MockCloudStorageProvider) Delete(petID types.PetID) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.connected {
		return errors.New("not connected")
	}

	if !m.authManager.IsLoggedIn() {
		return errors.New("not logged in")
	}

	delete(m.storage, petID)
	delete(m.metadata, petID)

	return nil
}

// List returns all pet IDs
func (m *MockCloudStorageProvider) List() ([]types.PetID, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if !m.connected {
		return nil, errors.New("not connected")
	}

	if !m.authManager.IsLoggedIn() {
		return nil, errors.New("not logged in")
	}

	petIDs := make([]types.PetID, 0, len(m.storage))
	for petID := range m.storage {
		petIDs = append(petIDs, petID)
	}

	return petIDs, nil
}

// GetLastModified returns when a pet was last modified
func (m *MockCloudStorageProvider) GetLastModified(petID types.PetID) (time.Time, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if !m.connected {
		return time.Time{}, errors.New("not connected")
	}

	if !m.authManager.IsLoggedIn() {
		return time.Time{}, errors.New("not logged in")
	}

	modTime, exists := m.metadata[petID]
	if !exists {
		return time.Time{}, fmt.Errorf("pet %s not found", petID)
	}

	return modTime, nil
}

// IsConnected returns connection status
func (m *MockCloudStorageProvider) IsConnected() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.connected
}

// SetConnected sets the connection status (for testing)
func (m *MockCloudStorageProvider) SetConnected(connected bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.connected = connected
}

// SetUploadDelay sets artificial delay for uploads (for testing)
func (m *MockCloudStorageProvider) SetUploadDelay(delay time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.uploadDelay = delay
}

// SetDownloadDelay sets artificial delay for downloads (for testing)
func (m *MockCloudStorageProvider) SetDownloadDelay(delay time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.downloadDelay = delay
}
