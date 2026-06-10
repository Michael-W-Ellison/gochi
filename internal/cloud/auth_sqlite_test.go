package cloud

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestNewSQLiteAuthProvider(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	provider, err := NewSQLiteAuthProvider(dbPath)
	if err != nil {
		t.Fatalf("Failed to create provider: %v", err)
	}
	defer provider.Close()

	// Verify database file was created
	if _, err := os.Stat(dbPath); err != nil {
		t.Errorf("Database file not created: %v", err)
	}
}

func TestSQLiteAuthProvider_RegisterAndAuthenticate(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")
	provider, err := NewSQLiteAuthProvider(dbPath)
	if err != nil {
		t.Fatalf("Failed to create provider: %v", err)
	}
	defer provider.Close()

	// Register user
	creds := &Credentials{
		Username: "testuser",
		Password: "TestPass123!",
		Email:    "test@example.com",
	}

	user, err := provider.Register(creds)
	if err != nil {
		t.Fatalf("Failed to register user: %v", err)
	}

	if user.Username != creds.Username {
		t.Errorf("Expected username %s, got %s", creds.Username, user.Username)
	}
	if user.Email != creds.Email {
		t.Errorf("Expected email %s, got %s", creds.Email, user.Email)
	}
	if user.IsPremium {
		t.Error("New user should not be premium")
	}

	// Authenticate
	session, err := provider.Authenticate(creds.Username, creds.Password)
	if err != nil {
		t.Fatalf("Failed to authenticate: %v", err)
	}

	if session.UserID != user.ID {
		t.Errorf("Expected user ID %s, got %s", user.ID, session.UserID)
	}
	if session.Token == "" {
		t.Error("Session token should not be empty")
	}
}

func TestSQLiteAuthProvider_AuthenticateInvalidCredentials(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")
	provider, err := NewSQLiteAuthProvider(dbPath)
	if err != nil {
		t.Fatalf("Failed to create provider: %v", err)
	}
	defer provider.Close()

	// Register user
	creds := &Credentials{
		Username: "testuser",
		Password: "TestPass123!",
		Email:    "test@example.com",
	}
	_, err = provider.Register(creds)
	if err != nil {
		t.Fatalf("Failed to register user: %v", err)
	}

	// Try wrong password
	_, err = provider.Authenticate(creds.Username, "wrongpassword")
	if err == nil {
		t.Error("Expected authentication to fail with wrong password")
	}

	// Try non-existent user
	_, err = provider.Authenticate("nonexistent", "password123")
	if err == nil {
		t.Error("Expected authentication to fail for non-existent user")
	}
}

func TestSQLiteAuthProvider_DuplicateUsername(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")
	provider, err := NewSQLiteAuthProvider(dbPath)
	if err != nil {
		t.Fatalf("Failed to create provider: %v", err)
	}
	defer provider.Close()

	// Register first user
	creds := &Credentials{
		Username: "testuser",
		Password: "TestPass123!",
		Email:    "test1@example.com",
	}
	_, err = provider.Register(creds)
	if err != nil {
		t.Fatalf("Failed to register user: %v", err)
	}

	// Try to register with same username
	creds2 := &Credentials{
		Username: "testuser",
		Password: "password456",
		Email:    "test2@example.com",
	}
	_, err = provider.Register(creds2)
	if err == nil {
		t.Error("Expected registration to fail with duplicate username")
	}
}

func TestSQLiteAuthProvider_DuplicateEmail(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")
	provider, err := NewSQLiteAuthProvider(dbPath)
	if err != nil {
		t.Fatalf("Failed to create provider: %v", err)
	}
	defer provider.Close()

	// Register first user
	creds := &Credentials{
		Username: "testuser1",
		Password: "TestPass123!",
		Email:    "test@example.com",
	}
	_, err = provider.Register(creds)
	if err != nil {
		t.Fatalf("Failed to register user: %v", err)
	}

	// Try to register with same email
	creds2 := &Credentials{
		Username: "testuser2",
		Password: "password456",
		Email:    "test@example.com",
	}
	_, err = provider.Register(creds2)
	if err == nil {
		t.Error("Expected registration to fail with duplicate email")
	}
}

func TestSQLiteAuthProvider_ValidateSession(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")
	provider, err := NewSQLiteAuthProvider(dbPath)
	if err != nil {
		t.Fatalf("Failed to create provider: %v", err)
	}
	defer provider.Close()

	// Register and authenticate
	creds := &Credentials{
		Username: "testuser",
		Password: "TestPass123!",
		Email:    "test@example.com",
	}
	user, _ := provider.Register(creds)
	session, err := provider.Authenticate(creds.Username, creds.Password)
	if err != nil {
		t.Fatalf("Failed to authenticate: %v", err)
	}

	// Validate session
	validatedSession, err := provider.ValidateSession(session.Token)
	if err != nil {
		t.Fatalf("Failed to validate session: %v", err)
	}

	if validatedSession.UserID != user.ID {
		t.Errorf("Expected user ID %s, got %s", user.ID, validatedSession.UserID)
	}

	// Try invalid token
	_, err = provider.ValidateSession("invalid-token")
	if err == nil {
		t.Error("Expected validation to fail for invalid token")
	}
}

func TestSQLiteAuthProvider_RevokeSession(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")
	provider, err := NewSQLiteAuthProvider(dbPath)
	if err != nil {
		t.Fatalf("Failed to create provider: %v", err)
	}
	defer provider.Close()

	// Register and authenticate
	creds := &Credentials{
		Username: "testuser",
		Password: "TestPass123!",
		Email:    "test@example.com",
	}
	_, _ = provider.Register(creds)
	session, _ := provider.Authenticate(creds.Username, creds.Password)

	// Revoke session
	err = provider.RevokeSession(session.Token)
	if err != nil {
		t.Fatalf("Failed to revoke session: %v", err)
	}

	// Verify session is invalid
	_, err = provider.ValidateSession(session.Token)
	if err == nil {
		t.Error("Expected validation to fail after revocation")
	}
}

func TestSQLiteAuthProvider_GetUser(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")
	provider, err := NewSQLiteAuthProvider(dbPath)
	if err != nil {
		t.Fatalf("Failed to create provider: %v", err)
	}
	defer provider.Close()

	// Register user
	creds := &Credentials{
		Username: "testuser",
		Password: "TestPass123!",
		Email:    "test@example.com",
	}
	user, _ := provider.Register(creds)

	// Get user
	retrievedUser, err := provider.GetUser(user.ID)
	if err != nil {
		t.Fatalf("Failed to get user: %v", err)
	}

	if retrievedUser.Username != user.Username {
		t.Errorf("Expected username %s, got %s", user.Username, retrievedUser.Username)
	}
	if retrievedUser.Email != user.Email {
		t.Errorf("Expected email %s, got %s", user.Email, retrievedUser.Email)
	}

	// Try non-existent user
	_, err = provider.GetUser("nonexistent")
	if err == nil {
		t.Error("Expected error for non-existent user")
	}
}

func TestSQLiteAuthProvider_UpdateUser(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")
	provider, err := NewSQLiteAuthProvider(dbPath)
	if err != nil {
		t.Fatalf("Failed to create provider: %v", err)
	}
	defer provider.Close()

	// Register user
	creds := &Credentials{
		Username: "testuser",
		Password: "TestPass123!",
		Email:    "test@example.com",
	}
	user, _ := provider.Register(creds)

	// Update user
	user.PetCount = 5
	user.IsPremium = true
	user.TotalPlayTime = 3600 * time.Second

	err = provider.UpdateUser(user)
	if err != nil {
		t.Fatalf("Failed to update user: %v", err)
	}

	// Retrieve and verify
	updatedUser, _ := provider.GetUser(user.ID)
	if updatedUser.PetCount != 5 {
		t.Errorf("Expected pet count 5, got %d", updatedUser.PetCount)
	}
	if !updatedUser.IsPremium {
		t.Error("Expected user to be premium")
	}
	if updatedUser.TotalPlayTime != 3600*time.Second {
		t.Errorf("Expected play time 3600s, got %v", updatedUser.TotalPlayTime)
	}
}

func TestSQLiteAuthProvider_CleanupExpiredSessions(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")
	provider, err := NewSQLiteAuthProvider(dbPath)
	if err != nil {
		t.Fatalf("Failed to create provider: %v", err)
	}
	defer provider.Close()

	// Register user
	creds := &Credentials{
		Username: "testuser",
		Password: "TestPass123!",
		Email:    "test@example.com",
	}
	_, _ = provider.Register(creds)

	// Create session
	session, _ := provider.Authenticate(creds.Username, creds.Password)

	// Manually expire the session
	provider.mu.Lock()
	_, err = provider.db.Exec("UPDATE sessions SET expires_at = ? WHERE token = ?",
		time.Now().Add(-1*time.Hour).Unix(), session.Token)
	provider.mu.Unlock()
	if err != nil {
		t.Fatalf("Failed to expire session: %v", err)
	}

	// Cleanup
	count, err := provider.CleanupExpiredSessions()
	if err != nil {
		t.Fatalf("Failed to cleanup sessions: %v", err)
	}

	if count != 1 {
		t.Errorf("Expected to cleanup 1 session, got %d", count)
	}

	// Verify session is gone
	_, err = provider.ValidateSession(session.Token)
	if err == nil {
		t.Error("Expected session to be gone after cleanup")
	}
}

func TestValidateCredentialsStrict(t *testing.T) {
	tests := []struct {
		name    string
		creds   *Credentials
		wantErr bool
	}{
		{
			name: "valid credentials",
			creds: &Credentials{
				Username: "testuser",
				Password: "TestPass123!",
				Email:    "test@example.com",
			},
			wantErr: false,
		},
		{
			name: "username too short",
			creds: &Credentials{
				Username: "ab",
				Password: "TestPass123!",
				Email:    "test@example.com",
			},
			wantErr: true,
		},
		{
			name: "username too long",
			creds: &Credentials{
				Username: "abcdefghijklmnopqrstuvwxyz",
				Password: "TestPass123!",
				Email:    "test@example.com",
			},
			wantErr: true,
		},
		{
			name: "username with invalid characters",
			creds: &Credentials{
				Username: "test-user!",
				Password: "TestPass123!",
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
			name: "invalid email - no @",
			creds: &Credentials{
				Username: "testuser",
				Password: "TestPass123!",
				Email:    "testexample.com",
			},
			wantErr: true,
		},
		{
			name: "invalid email - no domain",
			creds: &Credentials{
				Username: "testuser",
				Password: "TestPass123!",
				Email:    "test@",
			},
			wantErr: true,
		},
		{
			name: "invalid email - no TLD",
			creds: &Credentials{
				Username: "testuser",
				Password: "TestPass123!",
				Email:    "test@example",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateCredentialsStrict(tt.creds)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateCredentialsStrict() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestPasswordHashing(t *testing.T) {
	password := "testpassword123"

	// Generate salt
	salt := generateSalt()
	if salt == "" {
		t.Fatal("Salt generation failed")
	}

	// Hash password
	hash1 := hashPassword(password, salt)
	if hash1 == "" {
		t.Fatal("Password hashing failed")
	}

	// Hash same password with same salt should produce same hash
	hash2 := hashPassword(password, salt)
	if hash1 != hash2 {
		t.Error("Same password with same salt should produce same hash")
	}

	// Different salt should produce different hash
	salt2 := generateSalt()
	hash3 := hashPassword(password, salt2)
	if hash1 == hash3 {
		t.Error("Same password with different salt should produce different hash")
	}

	// Verify password
	if !verifyPassword(password, hash1, salt) {
		t.Error("Password verification failed")
	}

	// Wrong password should not verify
	if verifyPassword("wrongpassword", hash1, salt) {
		t.Error("Wrong password should not verify")
	}
}
