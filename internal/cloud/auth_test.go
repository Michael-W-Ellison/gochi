package cloud

import (
	"testing"
	"time"
)

func TestNewAuthManager(t *testing.T) {
	provider := NewMockAuthProvider()
	am := NewAuthManager(provider)

	if am == nil {
		t.Fatal("NewAuthManager returned nil")
	}

	if am.IsLoggedIn() {
		t.Error("Expected not logged in initially")
	}
}

func TestRegisterAndLogin(t *testing.T) {
	provider := NewMockAuthProvider()
	am := NewAuthManager(provider)

	// Register
	creds := &Credentials{
		Username: "testuser",
		Password: "password123",
		Email:    "test@example.com",
	}

	err := am.Register(creds)
	if err != nil {
		t.Fatalf("Registration failed: %v", err)
	}

	// Should be auto-logged in
	if !am.IsLoggedIn() {
		t.Error("Expected to be logged in after registration")
	}

	// Get user
	user, err := am.GetCurrentUser()
	if err != nil {
		t.Fatalf("Failed to get current user: %v", err)
	}

	if user.Username != "testuser" {
		t.Errorf("Expected username 'testuser', got %s", user.Username)
	}
}

func TestLogin(t *testing.T) {
	provider := NewMockAuthProvider()
	am := NewAuthManager(provider)

	// Register first
	creds := &Credentials{
		Username: "testuser",
		Password: "password123",
		Email:    "test@example.com",
	}
	provider.Register(creds)

	// Logout
	am.Logout()

	// Login
	err := am.Login("testuser", "password123")
	if err != nil {
		t.Fatalf("Login failed: %v", err)
	}

	if !am.IsLoggedIn() {
		t.Error("Expected to be logged in")
	}
}

func TestLoginInvalidCredentials(t *testing.T) {
	provider := NewMockAuthProvider()
	am := NewAuthManager(provider)

	// Try to login with non-existent user
	err := am.Login("nonexistent", "password")
	if err == nil {
		t.Error("Expected login to fail with invalid credentials")
	}
}

func TestLogout(t *testing.T) {
	provider := NewMockAuthProvider()
	am := NewAuthManager(provider)

	// Register and login
	creds := &Credentials{
		Username: "testuser",
		Password: "password123",
		Email:    "test@example.com",
	}
	am.Register(creds)

	// Logout
	err := am.Logout()
	if err != nil {
		t.Fatalf("Logout failed: %v", err)
	}

	if am.IsLoggedIn() {
		t.Error("Expected not logged in after logout")
	}

	// Try to logout again
	err = am.Logout()
	if err == nil {
		t.Error("Expected error when logging out while not logged in")
	}
}

func TestGetSession(t *testing.T) {
	provider := NewMockAuthProvider()
	am := NewAuthManager(provider)

	// Register
	creds := &Credentials{
		Username: "testuser",
		Password: "password123",
		Email:    "test@example.com",
	}
	am.Register(creds)

	// Get session
	session, err := am.GetSession()
	if err != nil {
		t.Fatalf("Failed to get session: %v", err)
	}

	if session.Token == "" {
		t.Error("Expected non-empty session token")
	}

	if session.UserID == "" {
		t.Error("Expected non-empty user ID")
	}
}

func TestRefreshSession(t *testing.T) {
	provider := NewMockAuthProvider()
	am := NewAuthManager(provider)

	// Register
	creds := &Credentials{
		Username: "testuser",
		Password: "password123",
		Email:    "test@example.com",
	}
	am.Register(creds)

	// Refresh session
	err := am.RefreshSession()
	if err != nil {
		t.Fatalf("Failed to refresh session: %v", err)
	}
}

func TestUpdatePlayTime(t *testing.T) {
	provider := NewMockAuthProvider()
	am := NewAuthManager(provider)

	// Register
	creds := &Credentials{
		Username: "testuser",
		Password: "password123",
		Email:    "test@example.com",
	}
	am.Register(creds)

	// Update play time
	duration := 2 * time.Hour
	err := am.UpdatePlayTime(duration)
	if err != nil {
		t.Fatalf("Failed to update play time: %v", err)
	}

	// Check updated
	user, _ := am.GetCurrentUser()
	if user.TotalPlayTime != duration {
		t.Errorf("Expected play time %v, got %v", duration, user.TotalPlayTime)
	}
}

func TestSetPremium(t *testing.T) {
	provider := NewMockAuthProvider()
	am := NewAuthManager(provider)

	// Register
	creds := &Credentials{
		Username: "testuser",
		Password: "password123",
		Email:    "test@example.com",
	}
	am.Register(creds)

	// Set premium
	err := am.SetPremium(true)
	if err != nil {
		t.Fatalf("Failed to set premium: %v", err)
	}

	// Check updated
	user, _ := am.GetCurrentUser()
	if !user.IsPremium {
		t.Error("Expected user to be premium")
	}
}

func TestValidateCredentials(t *testing.T) {
	tests := []struct {
		name    string
		creds   *Credentials
		wantErr bool
	}{
		{
			name: "valid credentials",
			creds: &Credentials{
				Username: "testuser",
				Password: "password123",
				Email:    "test@example.com",
			},
			wantErr: false,
		},
		{
			name: "username too short",
			creds: &Credentials{
				Username: "ab",
				Password: "password123",
				Email:    "test@example.com",
			},
			wantErr: true,
		},
		{
			name: "username too long",
			creds: &Credentials{
				Username: "verylongusernamethatexceedslimit",
				Password: "password123",
				Email:    "test@example.com",
			},
			wantErr: true,
		},
		{
			name: "password too short",
			creds: &Credentials{
				Username: "testuser",
				Password: "pass",
				Email:    "test@example.com",
			},
			wantErr: true,
		},
		{
			name: "invalid email",
			creds: &Credentials{
				Username: "testuser",
				Password: "password123",
				Email:    "invalid",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateCredentials(tt.creds)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateCredentials() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestDuplicateRegistration(t *testing.T) {
	provider := NewMockAuthProvider()

	// Register first user
	creds1 := &Credentials{
		Username: "testuser",
		Password: "password123",
		Email:    "test@example.com",
	}
	_, err := provider.Register(creds1)
	if err != nil {
		t.Fatalf("First registration failed: %v", err)
	}

	// Try to register with same username
	creds2 := &Credentials{
		Username: "testuser",
		Password: "password456",
		Email:    "test2@example.com",
	}
	_, err = provider.Register(creds2)
	if err == nil {
		t.Error("Expected error when registering duplicate username")
	}

	// Try to register with same email
	creds3 := &Credentials{
		Username: "testuser2",
		Password: "password456",
		Email:    "test@example.com",
	}
	_, err = provider.Register(creds3)
	if err == nil {
		t.Error("Expected error when registering duplicate email")
	}
}

func TestSessionExpiration(t *testing.T) {
	provider := NewMockAuthProvider()
	am := NewAuthManager(provider)

	// Register
	creds := &Credentials{
		Username: "testuser",
		Password: "password123",
		Email:    "test@example.com",
	}
	am.Register(creds)

	// Get session and manually expire it
	session, _ := am.GetSession()
	am.mu.Lock()
	session.ExpiresAt = time.Now().Add(-1 * time.Hour)
	am.mu.Unlock()

	// Should not be logged in anymore
	if am.IsLoggedIn() {
		t.Error("Expected not logged in with expired session")
	}
}

func TestGenerateSessionToken(t *testing.T) {
	token1 := GenerateSessionToken()
	token2 := GenerateSessionToken()

	if token1 == "" {
		t.Error("Expected non-empty token")
	}

	if token1 == token2 {
		t.Error("Expected different tokens")
	}

	if len(token1) < 32 {
		t.Error("Expected token to have reasonable length")
	}
}

func TestRefreshSessionNotLoggedIn(t *testing.T) {
	provider := NewMockAuthProvider()
	am := NewAuthManager(provider)

	// Try to refresh without logging in
	err := am.RefreshSession()
	if err == nil {
		t.Error("Expected error when refreshing session without being logged in")
	}
}

func TestUpdatePlayTimeNotLoggedIn(t *testing.T) {
	provider := NewMockAuthProvider()
	am := NewAuthManager(provider)

	// Try to update play time without logging in
	err := am.UpdatePlayTime(1 * time.Hour)
	if err == nil {
		t.Error("Expected error when updating play time without being logged in")
	}
}

func TestSetPremiumNotLoggedIn(t *testing.T) {
	provider := NewMockAuthProvider()
	am := NewAuthManager(provider)

	// Try to set premium without logging in
	err := am.SetPremium(true)
	if err == nil {
		t.Error("Expected error when setting premium without being logged in")
	}
}

func TestValidateSession(t *testing.T) {
	provider := NewMockAuthProvider()
	am := NewAuthManager(provider)

	// Register
	creds := &Credentials{
		Username: "testuser",
		Password: "password123",
		Email:    "test@example.com",
	}
	am.Register(creds)

	// Get session
	session, _ := am.GetSession()

	// Validate session
	validatedSession, err := provider.ValidateSession(session.Token)
	if err != nil {
		t.Fatalf("ValidateSession failed: %v", err)
	}

	if validatedSession == nil {
		t.Error("Expected session to be valid")
	}

	// Validate invalid token
	validatedSession, err = provider.ValidateSession("invalid-token")
	if err == nil {
		t.Error("Expected error when validating invalid token")
	}
	if validatedSession != nil {
		t.Error("Expected nil session for invalid token")
	}
}

func TestGetUser(t *testing.T) {
	provider := NewMockAuthProvider()

	// Register
	creds := &Credentials{
		Username: "testuser",
		Password: "password123",
		Email:    "test@example.com",
	}
	user, _ := provider.Register(creds)

	// Get user
	retrievedUser, err := provider.GetUser(user.ID)
	if err != nil {
		t.Fatalf("GetUser failed: %v", err)
	}

	if retrievedUser.Username != "testuser" {
		t.Errorf("Expected username 'testuser', got %s", retrievedUser.Username)
	}

	// Get non-existent user
	_, err = provider.GetUser("nonexistent")
	if err == nil {
		t.Error("Expected error when getting non-existent user")
	}
}

func TestUpdateUser(t *testing.T) {
	provider := NewMockAuthProvider()

	// Register
	creds := &Credentials{
		Username: "testuser",
		Password: "password123",
		Email:    "test@example.com",
	}
	user, _ := provider.Register(creds)

	// Update user
	user.IsPremium = true
	user.TotalPlayTime = 5 * time.Hour

	err := provider.UpdateUser(user)
	if err != nil {
		t.Fatalf("UpdateUser failed: %v", err)
	}

	// Get updated user
	updated, _ := provider.GetUser(user.ID)
	if !updated.IsPremium {
		t.Error("Expected user to be premium")
	}
	if updated.TotalPlayTime != 5*time.Hour {
		t.Errorf("Expected play time %v, got %v", 5*time.Hour, updated.TotalPlayTime)
	}

	// Update non-existent user
	nonExistentUser := &User{ID: "nonexistent"}
	err = provider.UpdateUser(nonExistentUser)
	if err == nil {
		t.Error("Expected error when updating non-existent user")
	}
}

func TestRevokeSession(t *testing.T) {
	provider := NewMockAuthProvider()
	am := NewAuthManager(provider)

	// Register and get session
	creds := &Credentials{
		Username: "testuser",
		Password: "password123",
		Email:    "test@example.com",
	}
	am.Register(creds)
	session, _ := am.GetSession()

	// Revoke session
	err := provider.RevokeSession(session.Token)
	if err != nil {
		t.Fatalf("RevokeSession failed: %v", err)
	}

	// Validate revoked session
	validatedSession, err := provider.ValidateSession(session.Token)
	if err == nil {
		t.Error("Expected error when validating revoked session")
	}
	if validatedSession != nil {
		t.Error("Expected nil session for revoked session")
	}
}

func TestRegisterInvalidCredentials(t *testing.T) {
	provider := NewMockAuthProvider()
	am := NewAuthManager(provider)

	// Try to register with invalid credentials
	invalidCreds := []*Credentials{
		{Username: "ab", Password: "password123", Email: "test@example.com"},
		{Username: "testuser", Password: "pass", Email: "test@example.com"},
		{Username: "testuser", Password: "password123", Email: "invalid"},
	}

	for _, creds := range invalidCreds {
		err := am.Register(creds)
		if err == nil {
			t.Errorf("Expected error when registering with invalid credentials: %+v", creds)
		}
	}
}
