package security

import (
	"errors"
	"path/filepath"
	"regexp"
	"strings"
	"unicode"
)

var (
	// Common regex patterns for validation
	usernameRegex = regexp.MustCompile(`^[a-zA-Z0-9_-]{3,20}$`)
	emailRegex    = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)
	petIDRegex    = regexp.MustCompile(`^[a-zA-Z0-9_-]{1,64}$`)
)

// ValidateUsername validates a username for security
func ValidateUsername(username string) error {
	if len(username) < 3 {
		return errors.New("username must be at least 3 characters")
	}
	if len(username) > 20 {
		return errors.New("username must be at most 20 characters")
	}

	if !usernameRegex.MatchString(username) {
		return errors.New("username can only contain letters, numbers, hyphens, and underscores")
	}

	// Check for suspicious patterns
	lower := strings.ToLower(username)
	suspiciousPatterns := []string{"admin", "root", "system", "null", "undefined"}
	for _, pattern := range suspiciousPatterns {
		if lower == pattern {
			return errors.New("username not allowed")
		}
	}

	return nil
}

// ValidateEmail validates an email address
func ValidateEmail(email string) error {
	if len(email) == 0 {
		return errors.New("email cannot be empty")
	}
	if len(email) > 254 {
		return errors.New("email too long")
	}

	if !emailRegex.MatchString(email) {
		return errors.New("invalid email format")
	}

	return nil
}

// ValidatePassword validates password strength
func ValidatePassword(password string) error {
	if len(password) < 8 {
		return errors.New("password must be at least 8 characters")
	}
	if len(password) > 128 {
		return errors.New("password must be at most 128 characters")
	}

	// Check for password complexity
	var hasUpper, hasLower, hasDigit, hasSpecial bool
	for _, ch := range password {
		switch {
		case unicode.IsUpper(ch):
			hasUpper = true
		case unicode.IsLower(ch):
			hasLower = true
		case unicode.IsDigit(ch):
			hasDigit = true
		case unicode.IsPunct(ch) || unicode.IsSymbol(ch):
			hasSpecial = true
		}
	}

	// Require at least 3 of 4 character types
	complexityScore := 0
	if hasUpper {
		complexityScore++
	}
	if hasLower {
		complexityScore++
	}
	if hasDigit {
		complexityScore++
	}
	if hasSpecial {
		complexityScore++
	}

	if complexityScore < 3 {
		return errors.New("password must contain at least 3 of: uppercase, lowercase, digits, special characters")
	}

	// Check for common weak passwords
	weakPasswords := []string{
		"password", "12345678", "qwerty123", "admin123",
		"letmein", "welcome1", "password1", "abc12345",
	}
	lower := strings.ToLower(password)
	for _, weak := range weakPasswords {
		if lower == weak || strings.Contains(lower, weak) {
			return errors.New("password is too common or weak")
		}
	}

	return nil
}

// ValidatePetID validates a pet ID for security
func ValidatePetID(petID string) error {
	if len(petID) == 0 {
		return errors.New("pet ID cannot be empty")
	}
	if len(petID) > 64 {
		return errors.New("pet ID too long")
	}

	if !petIDRegex.MatchString(petID) {
		return errors.New("pet ID contains invalid characters")
	}

	return nil
}

// ValidateFilePath validates a file path for security (prevents path traversal)
func ValidateFilePath(basePath, requestedPath string) error {
	// Clean the paths
	cleanBase := filepath.Clean(basePath)
	cleanRequested := filepath.Clean(requestedPath)

	// Check for path traversal attempts
	if strings.Contains(requestedPath, "..") {
		return errors.New("path traversal attempt detected")
	}

	// Ensure the requested path is within the base path
	if !strings.HasPrefix(cleanRequested, cleanBase) {
		return errors.New("path outside allowed directory")
	}

	// Check for suspicious characters
	if strings.ContainsAny(requestedPath, "\x00") {
		return errors.New("null byte in path")
	}

	return nil
}

// SanitizeFilename removes dangerous characters from filenames
func SanitizeFilename(filename string) string {
	// Extract just the base filename
	base := filepath.Base(filename)

	// Build sanitized filename with only safe characters
	var sanitized strings.Builder
	for _, ch := range base {
		if (ch >= 'a' && ch <= 'z') ||
			(ch >= 'A' && ch <= 'Z') ||
			(ch >= '0' && ch <= '9') ||
			ch == '-' || ch == '_' || ch == '.' {
			sanitized.WriteRune(ch)
		}
	}

	result := sanitized.String()

	// Prevent empty filenames
	if result == "" || result == "." || result == ".." {
		return ""
	}

	// Limit length
	if len(result) > 255 {
		return result[:255]
	}

	return result
}

// ValidateSessionToken validates a session token format
func ValidateSessionToken(token string) error {
	if len(token) == 0 {
		return errors.New("session token cannot be empty")
	}
	if len(token) < 32 {
		return errors.New("session token too short")
	}
	if len(token) > 256 {
		return errors.New("session token too long")
	}

	// Token should be base64 URL-safe characters
	validChars := regexp.MustCompile(`^[A-Za-z0-9_\-=]+$`)
	if !validChars.MatchString(token) {
		return errors.New("invalid session token format")
	}

	return nil
}
