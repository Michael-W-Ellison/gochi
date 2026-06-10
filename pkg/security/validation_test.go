package security

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateUsername(t *testing.T) {
	tests := []struct {
		username string
		valid    bool
	}{
		{"validuser", true},
		{"user123", true},
		{"user_name", true},
		{"user-name", true},
		{"ab", false},              // too short
		{"a", false},               // too short
		{"verylongusernamehere123", false}, // too long
		{"user name", false},       // space
		{"user@name", false},       // invalid char
		{"admin", false},           // reserved
		{"root", false},            // reserved
		{"", false},                // empty
	}

	for _, tt := range tests {
		err := ValidateUsername(tt.username)
		if tt.valid && err != nil {
			t.Errorf("ValidateUsername(%q) should be valid, got error: %v", tt.username, err)
		}
		if !tt.valid && err == nil {
			t.Errorf("ValidateUsername(%q) should be invalid, got no error", tt.username)
		}
	}
}

func TestValidateEmail(t *testing.T) {
	tests := []struct {
		email string
		valid bool
	}{
		{"user@example.com", true},
		{"test.user@example.co.uk", true},
		{"user+tag@example.com", true},
		{"invalid", false},
		{"@example.com", false},
		{"user@", false},
		{"user@.com", false},
		{"", false},
		{strings.Repeat("a", 250) + "@example.com", false}, // too long
	}

	for _, tt := range tests {
		err := ValidateEmail(tt.email)
		if tt.valid && err != nil {
			t.Errorf("ValidateEmail(%q) should be valid, got error: %v", tt.email, err)
		}
		if !tt.valid && err == nil {
			t.Errorf("ValidateEmail(%q) should be invalid, got no error", tt.email)
		}
	}
}

func TestValidatePassword(t *testing.T) {
	tests := []struct {
		password string
		valid    bool
	}{
		{"StrongP@ss1", true},
		{"MyP@ssw0rd", true},
		{"Complex1ty!", true},
		{"short", false},           // too short
		{"12345678", false},        // weak
		{"password1", false},       // common
		{"Password1", false},       // no special char (needs 3/4)
		{"PASSWORD!", false},       // no lowercase (needs 3/4)
		{"password!", false},       // no uppercase (needs 3/4)
		{strings.Repeat("A", 129), false}, // too long
	}

	for _, tt := range tests {
		err := ValidatePassword(tt.password)
		if tt.valid && err != nil {
			t.Errorf("ValidatePassword(%q) should be valid, got error: %v", tt.password, err)
		}
		if !tt.valid && err == nil {
			t.Errorf("ValidatePassword(%q) should be invalid, got no error", tt.password)
		}
	}
}

func TestValidatePetID(t *testing.T) {
	tests := []struct {
		petID string
		valid bool
	}{
		{"pet123", true},
		{"my-pet_01", true},
		{"PETID", true},
		{"", false},                // empty
		{"pet/id", false},          // invalid char
		{"pet id", false},          // space
		{"pet@123", false},         // invalid char
		{strings.Repeat("a", 65), false}, // too long
	}

	for _, tt := range tests {
		err := ValidatePetID(tt.petID)
		if tt.valid && err != nil {
			t.Errorf("ValidatePetID(%q) should be valid, got error: %v", tt.petID, err)
		}
		if !tt.valid && err == nil {
			t.Errorf("ValidatePetID(%q) should be invalid, got no error", tt.petID)
		}
	}
}

func TestValidateFilePath(t *testing.T) {
	basePath := "/var/data/pets"

	tests := []struct {
		path  string
		valid bool
	}{
		{"/var/data/pets/pet1.json", true},
		{"/var/data/pets/subdir/pet2.json", true},
		{"/var/data/pets/../pets/pet1.json", false}, // traversal
		{"/etc/passwd", false},                       // outside base
		{"/var/data/pets/\x00evil", false},          // null byte
	}

	for _, tt := range tests {
		err := ValidateFilePath(basePath, tt.path)
		if tt.valid && err != nil {
			t.Errorf("ValidateFilePath(%q) should be valid, got error: %v", tt.path, err)
		}
		if !tt.valid && err == nil {
			t.Errorf("ValidateFilePath(%q) should be invalid, got no error", tt.path)
		}
	}
}

func TestSanitizeFilename(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"valid-file_name.json", "valid-file_name.json"},
		{"../../../etc/passwd", "passwd"},
		{"/path/to/file.txt", "file.txt"},
		{"file with spaces.txt", "filewithspaces.txt"},
		{"file@#$%.txt", "file.txt"},
		{"", ""},
		{".", ""},
		{"..", ""},
		{strings.Repeat("a", 300), strings.Repeat("a", 255)},
	}

	for _, tt := range tests {
		result := SanitizeFilename(tt.input)
		if result != tt.expected {
			t.Errorf("SanitizeFilename(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func TestValidateSessionToken(t *testing.T) {
	tests := []struct {
		token string
		valid bool
	}{
		{strings.Repeat("a", 43), true}, // typical base64 token
		{"valid_token-12345678901234567890123", true},
		{"", false},                              // empty
		{strings.Repeat("a", 20), false},         // too short
		{strings.Repeat("a", 300), false},        // too long
		{"invalid token with spaces", false},     // invalid chars
		{"invalid@token", false},                 // invalid char
	}

	for _, tt := range tests {
		err := ValidateSessionToken(tt.token)
		if tt.valid && err != nil {
			t.Errorf("ValidateSessionToken(%q) should be valid, got error: %v", tt.token, err)
		}
		if !tt.valid && err == nil {
			t.Errorf("ValidateSessionToken(%q) should be invalid, got no error", tt.token)
		}
	}
}

func TestValidateFilePathWithClean(t *testing.T) {
	// Test with cleaned paths
	basePath := filepath.Clean("/var/data/pets")
	validPath := filepath.Join(basePath, "pet1.json")

	err := ValidateFilePath(basePath, validPath)
	if err != nil {
		t.Errorf("ValidateFilePath with clean paths should succeed: %v", err)
	}
}

func TestPasswordComplexity(t *testing.T) {
	// Test different complexity combinations
	tests := []struct {
		password string
		valid    bool
		desc     string
	}{
		{"Abc123!@", true, "has all 4 types"},
		{"Strong9Pass", true, "has 3 types (upper, lower, digit)"},
		{"Abcdefg!", true, "has 3 types (upper, lower, special)"},
		{"secure9!@", true, "has 3 types (lower, digit, special)"},
		{"SECURE9!@", true, "has 3 types (upper, digit, special)"},
		{"abcdefgh", false, "only lowercase"},
		{"ABCDEFGH", false, "only uppercase"},
		{"12345678", false, "only digits"},
		{"!@#$%^&*", false, "only special"},
		{"Abcdefgh", false, "only 2 types"},
	}

	for _, tt := range tests {
		err := ValidatePassword(tt.password)
		if tt.valid && err != nil {
			t.Errorf("%s: ValidatePassword(%q) should be valid, got error: %v", tt.desc, tt.password, err)
		}
		if !tt.valid && err == nil {
			t.Errorf("%s: ValidatePassword(%q) should be invalid, got no error", tt.desc, tt.password)
		}
	}
}
