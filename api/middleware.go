package api

import (
	"context"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/Michael-W-Ellison/gochi/pkg/types"
)

// ContextKey is a type for context keys
type ContextKey string

const (
	// ContextKeyUserID is the context key for user ID
	ContextKeyUserID ContextKey = "userID"
	// ContextKeyPetID is the context key for pet ID
	ContextKeyPetID ContextKey = "petID"
	// ContextKeyRequestID is the context key for request ID
	ContextKeyRequestID ContextKey = "requestID"
)

// TokenValidator validates authentication tokens
type TokenValidator interface {
	ValidateToken(token string) (types.UserID, error)
}

// ============================================================================
// Authentication Middleware
// ============================================================================

// AuthMiddleware creates an authentication middleware
func AuthMiddleware(validator TokenValidator) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Get token from Authorization header
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				writeError(w, http.StatusUnauthorized, "MISSING_TOKEN", "Authorization header required")
				return
			}

			// Extract Bearer token
			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
				writeError(w, http.StatusUnauthorized, "INVALID_TOKEN", "Invalid authorization format")
				return
			}

			token := parts[1]

			// Validate token
			userID, err := validator.ValidateToken(token)
			if err != nil {
				writeError(w, http.StatusUnauthorized, "INVALID_TOKEN", "Invalid or expired token")
				return
			}

			// Add user ID to context
			ctx := context.WithValue(r.Context(), ContextKeyUserID, userID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// OptionalAuthMiddleware allows requests without authentication but adds user ID if present
func OptionalAuthMiddleware(validator TokenValidator) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader != "" {
				parts := strings.SplitN(authHeader, " ", 2)
				if len(parts) == 2 && strings.ToLower(parts[0]) == "bearer" {
					if userID, err := validator.ValidateToken(parts[1]); err == nil {
						ctx := context.WithValue(r.Context(), ContextKeyUserID, userID)
						r = r.WithContext(ctx)
					}
				}
			}
			next.ServeHTTP(w, r)
		})
	}
}

// ============================================================================
// Logging Middleware
// ============================================================================

// LoggingMiddleware logs HTTP requests
func LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Create a response wrapper to capture status code
		wrapped := &responseWrapper{ResponseWriter: w, statusCode: http.StatusOK}

		next.ServeHTTP(wrapped, r)

		duration := time.Since(start)
		log.Printf("%s %s %d %v", r.Method, r.URL.Path, wrapped.statusCode, duration)
	})
}

// responseWrapper wraps http.ResponseWriter to capture status code
type responseWrapper struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWrapper) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// ============================================================================
// CORS Middleware
// ============================================================================

// CORSConfig holds CORS configuration
type CORSConfig struct {
	AllowedOrigins   []string
	AllowedMethods   []string
	AllowedHeaders   []string
	ExposedHeaders   []string
	AllowCredentials bool
	MaxAge           int
}

// DefaultCORSConfig returns a default CORS configuration
func DefaultCORSConfig() CORSConfig {
	return CORSConfig{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "PATCH", "OPTIONS"},
		AllowedHeaders:   []string{"Origin", "Content-Type", "Accept", "Authorization", "X-Request-ID"},
		ExposedHeaders:   []string{"X-Request-ID"},
		AllowCredentials: false,
		MaxAge:           86400, // 24 hours
	}
}

// CORSMiddleware creates a CORS middleware with the given configuration
func CORSMiddleware(config CORSConfig) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")

			// Check if origin is allowed
			allowed := false
			for _, allowedOrigin := range config.AllowedOrigins {
				if allowedOrigin == "*" || allowedOrigin == origin {
					allowed = true
					break
				}
			}

			if allowed {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				if config.AllowCredentials {
					w.Header().Set("Access-Control-Allow-Credentials", "true")
				}
			}

			// Handle preflight requests
			if r.Method == "OPTIONS" {
				w.Header().Set("Access-Control-Allow-Methods", strings.Join(config.AllowedMethods, ", "))
				w.Header().Set("Access-Control-Allow-Headers", strings.Join(config.AllowedHeaders, ", "))
				w.Header().Set("Access-Control-Max-Age", string(rune(config.MaxAge)))
				w.WriteHeader(http.StatusNoContent)
				return
			}

			if len(config.ExposedHeaders) > 0 {
				w.Header().Set("Access-Control-Expose-Headers", strings.Join(config.ExposedHeaders, ", "))
			}

			next.ServeHTTP(w, r)
		})
	}
}

// ============================================================================
// Rate Limiting Middleware
// ============================================================================

// RateLimiter defines the interface for rate limiting
type RateLimiter interface {
	Allow(key string) bool
	Limit() int
	Remaining(key string) int
	ResetTime(key string) time.Time
}

// RateLimitMiddleware creates a rate limiting middleware
func RateLimitMiddleware(limiter RateLimiter) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Use IP address or user ID as key
			key := r.RemoteAddr
			if userID, ok := r.Context().Value(ContextKeyUserID).(types.UserID); ok && userID != "" {
				key = string(userID)
			}

			if !limiter.Allow(key) {
				w.Header().Set("X-RateLimit-Limit", string(rune(limiter.Limit())))
				w.Header().Set("X-RateLimit-Remaining", "0")
				w.Header().Set("X-RateLimit-Reset", limiter.ResetTime(key).Format(time.RFC3339))
				writeError(w, http.StatusTooManyRequests, "RATE_LIMITED", "Too many requests")
				return
			}

			w.Header().Set("X-RateLimit-Limit", string(rune(limiter.Limit())))
			w.Header().Set("X-RateLimit-Remaining", string(rune(limiter.Remaining(key))))

			next.ServeHTTP(w, r)
		})
	}
}

// ============================================================================
// Request ID Middleware
// ============================================================================

// RequestIDMiddleware adds a unique request ID to each request
func RequestIDMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check if request ID is provided in header
		requestID := r.Header.Get("X-Request-ID")
		if requestID == "" {
			requestID = generateRequestID()
		}

		// Add to context and response header
		ctx := context.WithValue(r.Context(), ContextKeyRequestID, requestID)
		w.Header().Set("X-Request-ID", requestID)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// generateRequestID generates a unique request ID
func generateRequestID() string {
	return time.Now().Format("20060102150405") + "-" + randomString(8)
}

// randomString generates a random string of given length
func randomString(n int) string {
	const letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, n)
	for i := range b {
		b[i] = letters[time.Now().UnixNano()%int64(len(letters))]
	}
	return string(b)
}

// ============================================================================
// Recovery Middleware
// ============================================================================

// RecoveryMiddleware recovers from panics and returns a 500 error
func RecoveryMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				log.Printf("Panic recovered: %v", err)
				writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "An internal error occurred")
			}
		}()

		next.ServeHTTP(w, r)
	})
}

// ============================================================================
// Content Type Middleware
// ============================================================================

// JSONContentTypeMiddleware ensures responses have JSON content type
func JSONContentTypeMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		next.ServeHTTP(w, r)
	})
}

// RequireJSONMiddleware ensures requests have JSON content type
func RequireJSONMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" && r.Method != "DELETE" && r.Method != "OPTIONS" {
			contentType := r.Header.Get("Content-Type")
			if !strings.Contains(contentType, "application/json") {
				writeError(w, http.StatusUnsupportedMediaType, "INVALID_CONTENT_TYPE", "Content-Type must be application/json")
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}

// ============================================================================
// Timeout Middleware
// ============================================================================

// TimeoutMiddleware adds a timeout to requests
func TimeoutMiddleware(timeout time.Duration) Middleware {
	return func(next http.Handler) http.Handler {
		return http.TimeoutHandler(next, timeout, `{"success":false,"error":{"code":"TIMEOUT","message":"Request timeout"}}`)
	}
}

// ============================================================================
// Middleware Chain
// ============================================================================

// Chain chains multiple middleware together
func Chain(middlewares ...Middleware) Middleware {
	return func(final http.Handler) http.Handler {
		for i := len(middlewares) - 1; i >= 0; i-- {
			final = middlewares[i](final)
		}
		return final
	}
}

// DefaultMiddlewareChain returns the default middleware chain for API routes
func DefaultMiddlewareChain(validator TokenValidator) Middleware {
	return Chain(
		RecoveryMiddleware,
		RequestIDMiddleware,
		LoggingMiddleware,
		CORSMiddleware(DefaultCORSConfig()),
		JSONContentTypeMiddleware,
	)
}

// AuthenticatedMiddlewareChain returns middleware chain with authentication
func AuthenticatedMiddlewareChain(validator TokenValidator) Middleware {
	return Chain(
		RecoveryMiddleware,
		RequestIDMiddleware,
		LoggingMiddleware,
		CORSMiddleware(DefaultCORSConfig()),
		JSONContentTypeMiddleware,
		RequireJSONMiddleware,
		AuthMiddleware(validator),
	)
}
