package security

import (
	"time"

	"github.com/Michael-W-Ellison/gochi/pkg/logger"
)

// AuditEventType represents the type of security event
type AuditEventType string

const (
	EventLoginSuccess        AuditEventType = "login_success"
	EventLoginFailure        AuditEventType = "login_failure"
	EventRegistration        AuditEventType = "registration"
	EventSessionCreated      AuditEventType = "session_created"
	EventSessionRevoked      AuditEventType = "session_revoked"
	EventSessionExpired      AuditEventType = "session_expired"
	EventRateLimitExceeded   AuditEventType = "rate_limit_exceeded"
	EventInvalidInput        AuditEventType = "invalid_input"
	EventPathTraversalAttempt AuditEventType = "path_traversal_attempt"
	EventPasswordChange      AuditEventType = "password_change"
	EventAccountDeleted      AuditEventType = "account_deleted"
	EventUnauthorizedAccess  AuditEventType = "unauthorized_access"
)

// AuditEvent represents a security-relevant event
type AuditEvent struct {
	Timestamp  time.Time
	EventType  AuditEventType
	UserID     string
	Username   string
	IPAddress  string
	DeviceID   string
	Success    bool
	Message    string
	Metadata   map[string]interface{}
}

// LogAuditEvent logs a security audit event
func LogAuditEvent(event *AuditEvent) {
	if event == nil {
		return
	}

	// Set timestamp if not provided
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now()
	}

	// Build structured log entry
	args := []interface{}{
		"event_type", string(event.EventType),
		"timestamp", event.Timestamp,
		"success", event.Success,
	}

	if event.UserID != "" {
		args = append(args, "user_id", event.UserID)
	}
	if event.Username != "" {
		args = append(args, "username", event.Username)
	}
	if event.IPAddress != "" {
		args = append(args, "ip_address", event.IPAddress)
	}
	if event.DeviceID != "" {
		args = append(args, "device_id", event.DeviceID)
	}
	if event.Message != "" {
		args = append(args, "message", event.Message)
	}

	// Add metadata
	for key, value := range event.Metadata {
		args = append(args, key, value)
	}

	// Log based on severity
	if !event.Success {
		switch event.EventType {
		case EventRateLimitExceeded, EventPathTraversalAttempt, EventUnauthorizedAccess:
			logger.Error("security event", args...)
		case EventLoginFailure, EventInvalidInput:
			logger.Warn("security event", args...)
		default:
			logger.Info("security event", args...)
		}
	} else {
		logger.Info("security event", args...)
	}
}

// LogLoginAttempt logs a login attempt
func LogLoginAttempt(username, ipAddress string, success bool, message string) {
	LogAuditEvent(&AuditEvent{
		EventType: EventLoginSuccess,
		Username:  username,
		IPAddress: ipAddress,
		Success:   success,
		Message:   message,
	})
}

// LogRegistration logs a user registration
func LogRegistration(username, ipAddress string, success bool) {
	LogAuditEvent(&AuditEvent{
		EventType: EventRegistration,
		Username:  username,
		IPAddress: ipAddress,
		Success:   success,
	})
}

// LogRateLimitExceeded logs a rate limit violation
func LogRateLimitExceeded(identifier, ipAddress string) {
	LogAuditEvent(&AuditEvent{
		EventType: EventRateLimitExceeded,
		Username:  identifier,
		IPAddress: ipAddress,
		Success:   false,
		Message:   "rate limit exceeded",
	})
}

// LogPathTraversalAttempt logs a potential path traversal attack
func LogPathTraversalAttempt(userID, path string) {
	LogAuditEvent(&AuditEvent{
		EventType: EventPathTraversalAttempt,
		UserID:    userID,
		Success:   false,
		Message:   "path traversal attempt detected",
		Metadata: map[string]interface{}{
			"attempted_path": path,
		},
	})
}
