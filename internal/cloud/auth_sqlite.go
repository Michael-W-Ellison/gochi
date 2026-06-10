package cloud

import (
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"errors"
	"fmt"
	"regexp"
	"sync"
	"time"

	"github.com/Michael-W-Ellison/gochi/pkg/security"
	"golang.org/x/crypto/pbkdf2"
	_ "modernc.org/sqlite"
)

// SQLiteAuthProvider implements AuthProvider using SQLite database
type SQLiteAuthProvider struct {
	mu          sync.RWMutex
	db          *sql.DB
	rateLimiter *security.RateLimiter
}

// NewSQLiteAuthProvider creates a new SQLite-based auth provider
func NewSQLiteAuthProvider(dbPath string) (*SQLiteAuthProvider, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Test connection
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	// Create rate limiter: 5 attempts per 15 minutes
	rateLimiter := security.NewRateLimiter(5, 15*time.Minute)

	provider := &SQLiteAuthProvider{
		db:          db,
		rateLimiter: rateLimiter,
	}

	// Initialize schema
	if err := provider.initSchema(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to initialize schema: %w", err)
	}

	return provider, nil
}

// Close closes the database connection
func (s *SQLiteAuthProvider) Close() error {
	return s.db.Close()
}

// initSchema creates the necessary tables
func (s *SQLiteAuthProvider) initSchema() error {
	schema := `
	CREATE TABLE IF NOT EXISTS users (
		id TEXT PRIMARY KEY,
		username TEXT UNIQUE NOT NULL,
		email TEXT UNIQUE NOT NULL,
		password_hash TEXT NOT NULL,
		password_salt TEXT NOT NULL,
		created_at INTEGER NOT NULL,
		last_login_at INTEGER NOT NULL,
		total_play_time INTEGER NOT NULL DEFAULT 0,
		pet_count INTEGER NOT NULL DEFAULT 0,
		is_premium INTEGER NOT NULL DEFAULT 0
	);

	CREATE INDEX IF NOT EXISTS idx_users_username ON users(username);
	CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);

	CREATE TABLE IF NOT EXISTS sessions (
		token TEXT PRIMARY KEY,
		user_id TEXT NOT NULL,
		created_at INTEGER NOT NULL,
		expires_at INTEGER NOT NULL,
		ip_address TEXT,
		device_id TEXT,
		FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
	);

	CREATE INDEX IF NOT EXISTS idx_sessions_user_id ON sessions(user_id);
	CREATE INDEX IF NOT EXISTS idx_sessions_expires_at ON sessions(expires_at);
	`

	_, err := s.db.Exec(schema)
	return err
}

// Authenticate verifies credentials and creates a session
func (s *SQLiteAuthProvider) Authenticate(username, password string) (*Session, error) {
	// Check rate limit before acquiring lock
	if s.rateLimiter.CheckLimit(username) {
		security.LogRateLimitExceeded(username, "")
		return nil, errors.New("too many authentication attempts, please try again later")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// Get user
	var userID, storedHash, salt string
	var lastLoginUnix int64
	query := "SELECT id, password_hash, password_salt, last_login_at FROM users WHERE username = ?"
	err := s.db.QueryRow(query, username).Scan(&userID, &storedHash, &salt, &lastLoginUnix)
	if err == sql.ErrNoRows {
		security.LogLoginAttempt(username, "", false, "user not found")
		return nil, errors.New("invalid credentials")
	}
	if err != nil {
		security.LogLoginAttempt(username, "", false, "database error")
		return nil, fmt.Errorf("database error: %w", err)
	}

	// Verify password
	if !verifyPassword(password, storedHash, salt) {
		security.LogLoginAttempt(username, "", false, "invalid password")
		return nil, errors.New("invalid credentials")
	}

	// Reset rate limit on successful authentication
	s.rateLimiter.Reset(username)

	// Create session
	session := &Session{
		Token:     GenerateSessionToken(),
		UserID:    userID,
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}

	// Store session
	insertSession := `INSERT INTO sessions (token, user_id, created_at, expires_at, ip_address, device_id)
	                  VALUES (?, ?, ?, ?, ?, ?)`
	_, err = s.db.Exec(insertSession,
		session.Token,
		session.UserID,
		session.CreatedAt.Unix(),
		session.ExpiresAt.Unix(),
		session.IPAddress,
		session.DeviceID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	// Update last login
	_, err = s.db.Exec("UPDATE users SET last_login_at = ? WHERE id = ?", time.Now().Unix(), userID)
	if err != nil {
		return nil, fmt.Errorf("failed to update last login: %w", err)
	}

	// Log successful authentication
	security.LogLoginAttempt(username, session.IPAddress, true, "authentication successful")

	return session, nil
}

// Register creates a new user account
func (s *SQLiteAuthProvider) Register(creds *Credentials) (*User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Validate credentials with enhanced security checks
	if err := security.ValidateUsername(creds.Username); err != nil {
		security.LogRegistration(creds.Username, "", false)
		return nil, err
	}
	if err := security.ValidateEmail(creds.Email); err != nil {
		security.LogRegistration(creds.Username, "", false)
		return nil, err
	}
	if err := security.ValidatePassword(creds.Password); err != nil {
		security.LogRegistration(creds.Username, "", false)
		return nil, err
	}

	// Check if username exists
	var exists int
	err := s.db.QueryRow("SELECT COUNT(*) FROM users WHERE username = ?", creds.Username).Scan(&exists)
	if err != nil {
		security.LogRegistration(creds.Username, "", false)
		return nil, fmt.Errorf("database error: %w", err)
	}
	if exists > 0 {
		security.LogRegistration(creds.Username, "", false)
		return nil, errors.New("username already exists")
	}

	// Check if email exists
	err = s.db.QueryRow("SELECT COUNT(*) FROM users WHERE email = ?", creds.Email).Scan(&exists)
	if err != nil {
		security.LogRegistration(creds.Username, "", false)
		return nil, fmt.Errorf("database error: %w", err)
	}
	if exists > 0 {
		security.LogRegistration(creds.Username, "", false)
		return nil, errors.New("email already exists")
	}

	// Generate user ID
	userID := GenerateSessionToken()[:16]

	// Hash password
	salt := generateSalt()
	passwordHash := hashPassword(creds.Password, salt)

	// Create user
	now := time.Now()
	insertUser := `INSERT INTO users (id, username, email, password_hash, password_salt, created_at, last_login_at)
	               VALUES (?, ?, ?, ?, ?, ?, ?)`
	_, err = s.db.Exec(insertUser, userID, creds.Username, creds.Email, passwordHash, salt, now.Unix(), now.Unix())
	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	user := &User{
		ID:          userID,
		Username:    creds.Username,
		Email:       creds.Email,
		CreatedAt:   now,
		LastLoginAt: now,
		PetCount:    0,
		IsPremium:   false,
	}

	// Log successful registration
	security.LogRegistration(creds.Username, "", true)

	return user, nil
}

// ValidateSession checks if a session token is valid
func (s *SQLiteAuthProvider) ValidateSession(token string) (*Session, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var userID, ipAddress, deviceID string
	var createdAtUnix, expiresAtUnix int64

	query := "SELECT user_id, created_at, expires_at, ip_address, device_id FROM sessions WHERE token = ?"
	err := s.db.QueryRow(query, token).Scan(&userID, &createdAtUnix, &expiresAtUnix, &ipAddress, &deviceID)
	if err == sql.ErrNoRows {
		return nil, errors.New("session not found")
	}
	if err != nil {
		return nil, fmt.Errorf("database error: %w", err)
	}

	expiresAt := time.Unix(expiresAtUnix, 0)
	if time.Now().After(expiresAt) {
		return nil, errors.New("session expired")
	}

	session := &Session{
		Token:     token,
		UserID:    userID,
		CreatedAt: time.Unix(createdAtUnix, 0),
		ExpiresAt: expiresAt,
		IPAddress: ipAddress,
		DeviceID:  deviceID,
	}

	return session, nil
}

// RevokeSession invalidates a session token
func (s *SQLiteAuthProvider) RevokeSession(token string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	_, err := s.db.Exec("DELETE FROM sessions WHERE token = ?", token)
	return err
}

// GetUser retrieves user information
func (s *SQLiteAuthProvider) GetUser(userID string) (*User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var username, email string
	var createdAtUnix, lastLoginUnix, totalPlayTime int64
	var petCount, isPremiumInt int

	query := `SELECT username, email, created_at, last_login_at, total_play_time, pet_count, is_premium
	          FROM users WHERE id = ?`
	err := s.db.QueryRow(query, userID).Scan(
		&username, &email, &createdAtUnix, &lastLoginUnix,
		&totalPlayTime, &petCount, &isPremiumInt,
	)
	if err == sql.ErrNoRows {
		return nil, errors.New("user not found")
	}
	if err != nil {
		return nil, fmt.Errorf("database error: %w", err)
	}

	user := &User{
		ID:            userID,
		Username:      username,
		Email:         email,
		CreatedAt:     time.Unix(createdAtUnix, 0),
		LastLoginAt:   time.Unix(lastLoginUnix, 0),
		TotalPlayTime: time.Duration(totalPlayTime) * time.Second,
		PetCount:      petCount,
		IsPremium:     isPremiumInt != 0,
	}

	return user, nil
}

// UpdateUser updates user information
func (s *SQLiteAuthProvider) UpdateUser(user *User) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	isPremiumInt := 0
	if user.IsPremium {
		isPremiumInt = 1
	}

	query := `UPDATE users SET username = ?, email = ?, last_login_at = ?,
	          total_play_time = ?, pet_count = ?, is_premium = ?
	          WHERE id = ?`
	result, err := s.db.Exec(query,
		user.Username,
		user.Email,
		user.LastLoginAt.Unix(),
		int64(user.TotalPlayTime.Seconds()),
		user.PetCount,
		isPremiumInt,
		user.ID,
	)
	if err != nil {
		return fmt.Errorf("database error: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rows == 0 {
		return errors.New("user not found")
	}

	return nil
}

// CleanupExpiredSessions removes expired sessions from the database
func (s *SQLiteAuthProvider) CleanupExpiredSessions() (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	result, err := s.db.Exec("DELETE FROM sessions WHERE expires_at < ?", time.Now().Unix())
	if err != nil {
		return 0, fmt.Errorf("failed to cleanup sessions: %w", err)
	}

	count, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to get rows affected: %w", err)
	}

	return count, nil
}

// Password hashing functions using PBKDF2

const (
	pbkdf2Iterations = 100000
	pbkdf2KeyLen     = 32
)

func generateSalt() string {
	salt := make([]byte, 16)
	_, err := rand.Read(salt)
	if err != nil {
		// Fallback to timestamp-based salt if crypto/rand fails
		salt = []byte(fmt.Sprintf("%d", time.Now().UnixNano()))
	}
	return base64.StdEncoding.EncodeToString(salt)
}

func hashPassword(password, salt string) string {
	saltBytes, _ := base64.StdEncoding.DecodeString(salt)
	hash := pbkdf2.Key([]byte(password), saltBytes, pbkdf2Iterations, pbkdf2KeyLen, sha256.New)
	return base64.StdEncoding.EncodeToString(hash)
}

func verifyPassword(password, storedHash, salt string) bool {
	computedHash := hashPassword(password, salt)
	return computedHash == storedHash
}

// ValidateCredentialsStrict performs strict validation on credentials
func ValidateCredentialsStrict(creds *Credentials) error {
	// Username validation
	if len(creds.Username) < 3 {
		return errors.New("username must be at least 3 characters")
	}
	if len(creds.Username) > 20 {
		return errors.New("username must be at most 20 characters")
	}

	// Only allow alphanumeric and underscore in username
	usernameRegex := regexp.MustCompile(`^[a-zA-Z0-9_]+$`)
	if !usernameRegex.MatchString(creds.Username) {
		return errors.New("username can only contain letters, numbers, and underscores")
	}

	// Password validation
	if len(creds.Password) < 8 {
		return errors.New("password must be at least 8 characters")
	}
	if len(creds.Password) > 128 {
		return errors.New("password must be at most 128 characters")
	}

	// Email validation (RFC 5322 simplified)
	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)
	if !emailRegex.MatchString(creds.Email) {
		return errors.New("invalid email address format")
	}

	return nil
}
