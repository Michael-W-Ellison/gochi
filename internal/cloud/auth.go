package cloud

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"sync"
	"time"
)

// User represents a cloud service user
type User struct {
	ID            string
	Username      string
	Email         string
	CreatedAt     time.Time
	LastLoginAt   time.Time
	TotalPlayTime time.Duration
	PetCount      int
	IsPremium     bool
}

// Session represents an authenticated user session
type Session struct {
	Token     string
	UserID    string
	CreatedAt time.Time
	ExpiresAt time.Time
	IPAddress string
	DeviceID  string
}

// Credentials holds login information
type Credentials struct {
	Username string
	Password string
	Email    string
}

// AuthProvider defines the interface for authentication backends
type AuthProvider interface {
	// Authenticate verifies credentials and returns a session
	Authenticate(username, password string) (*Session, error)

	// Register creates a new user account
	Register(creds *Credentials) (*User, error)

	// ValidateSession checks if a session token is valid
	ValidateSession(token string) (*Session, error)

	// RevokeSession invalidates a session token
	RevokeSession(token string) error

	// GetUser retrieves user information
	GetUser(userID string) (*User, error)

	// UpdateUser updates user information
	UpdateUser(user *User) error
}

// AuthManager handles authentication and session management
type AuthManager struct {
	mu sync.RWMutex

	provider       AuthProvider
	currentSession *Session
	currentUser    *User
	sessionTimeout time.Duration
}

// NewAuthManager creates a new authentication manager
func NewAuthManager(provider AuthProvider) *AuthManager {
	return &AuthManager{
		provider:       provider,
		sessionTimeout: 24 * time.Hour,
	}
}

// Login authenticates a user and creates a session
func (am *AuthManager) Login(username, password string) error {
	am.mu.Lock()
	defer am.mu.Unlock()

	// Authenticate with provider
	session, err := am.provider.Authenticate(username, password)
	if err != nil {
		return fmt.Errorf("authentication failed: %w", err)
	}

	// Get user information
	user, err := am.provider.GetUser(session.UserID)
	if err != nil {
		return fmt.Errorf("failed to get user info: %w", err)
	}

	am.currentSession = session
	am.currentUser = user

	return nil
}

// Register creates a new user account
func (am *AuthManager) Register(creds *Credentials) error {
	am.mu.Lock()
	defer am.mu.Unlock()

	// Validate credentials
	if err := validateCredentials(creds); err != nil {
		return err
	}

	// Register with provider
	user, err := am.provider.Register(creds)
	if err != nil {
		return fmt.Errorf("registration failed: %w", err)
	}

	// Auto-login after registration
	session, err := am.provider.Authenticate(creds.Username, creds.Password)
	if err != nil {
		// Registration succeeded but auto-login failed
		return nil
	}

	am.currentSession = session
	am.currentUser = user

	return nil
}

// Logout ends the current session
func (am *AuthManager) Logout() error {
	am.mu.Lock()
	defer am.mu.Unlock()

	if am.currentSession == nil {
		return errors.New("not logged in")
	}

	// Revoke session with provider
	if err := am.provider.RevokeSession(am.currentSession.Token); err != nil {
		// Log error but continue with local logout
	}

	am.currentSession = nil
	am.currentUser = nil

	return nil
}

// IsLoggedIn returns whether there's an active session
func (am *AuthManager) IsLoggedIn() bool {
	am.mu.RLock()
	defer am.mu.RUnlock()

	if am.currentSession == nil {
		return false
	}

	// Check if session has expired
	if time.Now().After(am.currentSession.ExpiresAt) {
		return false
	}

	return true
}

// GetCurrentUser returns the currently logged-in user
func (am *AuthManager) GetCurrentUser() (*User, error) {
	am.mu.RLock()
	defer am.mu.RUnlock()

	if am.currentUser == nil {
		return nil, errors.New("not logged in")
	}

	return am.currentUser, nil
}

// GetSession returns the current session
func (am *AuthManager) GetSession() (*Session, error) {
	am.mu.RLock()
	defer am.mu.RUnlock()

	if am.currentSession == nil {
		return nil, errors.New("no active session")
	}

	return am.currentSession, nil
}

// RefreshSession extends the current session
func (am *AuthManager) RefreshSession() error {
	am.mu.Lock()
	defer am.mu.Unlock()

	if am.currentSession == nil {
		return errors.New("no active session")
	}

	// Validate current session
	session, err := am.provider.ValidateSession(am.currentSession.Token)
	if err != nil {
		// Session invalid, clear it
		am.currentSession = nil
		am.currentUser = nil
		return fmt.Errorf("session invalid: %w", err)
	}

	am.currentSession = session
	return nil
}

// UpdatePlayTime updates the user's total play time
func (am *AuthManager) UpdatePlayTime(duration time.Duration) error {
	am.mu.Lock()
	defer am.mu.Unlock()

	if am.currentUser == nil {
		return errors.New("not logged in")
	}

	am.currentUser.TotalPlayTime += duration

	// Update with provider
	if err := am.provider.UpdateUser(am.currentUser); err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}

	return nil
}

// SetPremium sets the premium status of the current user
func (am *AuthManager) SetPremium(isPremium bool) error {
	am.mu.Lock()
	defer am.mu.Unlock()

	if am.currentUser == nil {
		return errors.New("not logged in")
	}

	am.currentUser.IsPremium = isPremium

	if err := am.provider.UpdateUser(am.currentUser); err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}

	return nil
}

// validateCredentials checks if credentials are valid
func validateCredentials(creds *Credentials) error {
	if len(creds.Username) < 3 {
		return errors.New("username must be at least 3 characters")
	}

	if len(creds.Username) > 20 {
		return errors.New("username must be at most 20 characters")
	}

	if len(creds.Password) < 8 {
		return errors.New("password must be at least 8 characters")
	}

	if len(creds.Email) < 5 || !containsAt(creds.Email) {
		return errors.New("invalid email address")
	}

	return nil
}

// containsAt checks if a string contains @
func containsAt(s string) bool {
	for _, c := range s {
		if c == '@' {
			return true
		}
	}
	return false
}

// GenerateSessionToken generates a random session token
func GenerateSessionToken() string {
	b := make([]byte, 32)
	rand.Read(b)
	return base64.URLEncoding.EncodeToString(b)
}

// MockAuthProvider is a simple in-memory auth provider for testing
type MockAuthProvider struct {
	mu       sync.RWMutex
	users    map[string]*User
	passwords map[string]string
	sessions map[string]*Session
}

// NewMockAuthProvider creates a new mock auth provider
func NewMockAuthProvider() *MockAuthProvider {
	return &MockAuthProvider{
		users:    make(map[string]*User),
		passwords: make(map[string]string),
		sessions: make(map[string]*Session),
	}
}

// Authenticate verifies credentials
func (m *MockAuthProvider) Authenticate(username, password string) (*Session, error) {
	m.mu.RLock()

	// Find user by username
	var user *User
	for _, u := range m.users {
		if u.Username == username {
			user = u
			break
		}
	}

	if user == nil {
		m.mu.RUnlock()
		return nil, errors.New("invalid credentials")
	}

	// Check password
	if m.passwords[user.ID] != password {
		m.mu.RUnlock()
		return nil, errors.New("invalid credentials")
	}

	// Create session
	session := &Session{
		Token:     GenerateSessionToken(),
		UserID:    user.ID,
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}

	// Upgrade to write lock
	m.mu.RUnlock()
	m.mu.Lock()
	defer m.mu.Unlock()

	m.sessions[session.Token] = session
	user.LastLoginAt = time.Now()

	return session, nil
}

// Register creates a new user
func (m *MockAuthProvider) Register(creds *Credentials) (*User, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check if username exists
	for _, u := range m.users {
		if u.Username == creds.Username {
			return nil, errors.New("username already exists")
		}
	}

	// Check if email exists
	for _, u := range m.users {
		if u.Email == creds.Email {
			return nil, errors.New("email already exists")
		}
	}

	// Create user
	user := &User{
		ID:          GenerateSessionToken()[:16],
		Username:    creds.Username,
		Email:       creds.Email,
		CreatedAt:   time.Now(),
		LastLoginAt: time.Now(),
		PetCount:    0,
		IsPremium:   false,
	}

	m.users[user.ID] = user
	m.passwords[user.ID] = creds.Password

	return user, nil
}

// ValidateSession checks if a token is valid
func (m *MockAuthProvider) ValidateSession(token string) (*Session, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	session, exists := m.sessions[token]
	if !exists {
		return nil, errors.New("session not found")
	}

	if time.Now().After(session.ExpiresAt) {
		return nil, errors.New("session expired")
	}

	return session, nil
}

// RevokeSession invalidates a session
func (m *MockAuthProvider) RevokeSession(token string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.sessions, token)
	return nil
}

// GetUser retrieves user information
func (m *MockAuthProvider) GetUser(userID string) (*User, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	user, exists := m.users[userID]
	if !exists {
		return nil, errors.New("user not found")
	}

	return user, nil
}

// UpdateUser updates user information
func (m *MockAuthProvider) UpdateUser(user *User) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.users[user.ID]; !exists {
		return errors.New("user not found")
	}

	m.users[user.ID] = user
	return nil
}
