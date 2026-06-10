package cloud

import (
	"errors"
	"sync"
	"time"
)

// MockAuthProvider is a simple in-memory auth provider for testing
type MockAuthProvider struct {
	mu        sync.RWMutex
	users     map[string]*User
	passwords map[string]string
	sessions  map[string]*Session
}

// NewMockAuthProvider creates a new mock auth provider
func NewMockAuthProvider() *MockAuthProvider {
	return &MockAuthProvider{
		users:     make(map[string]*User),
		passwords: make(map[string]string),
		sessions:  make(map[string]*Session),
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
