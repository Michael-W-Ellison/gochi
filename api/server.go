package api

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

// ServerConfig holds server configuration
type ServerConfig struct {
	Host            string
	Port            int
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	IdleTimeout     time.Duration
	ShutdownTimeout time.Duration
	TLSCertFile     string
	TLSKeyFile      string
}

// DefaultServerConfig returns default server configuration
func DefaultServerConfig() ServerConfig {
	return ServerConfig{
		Host:            "",
		Port:            8080,
		ReadTimeout:     15 * time.Second,
		WriteTimeout:    15 * time.Second,
		IdleTimeout:     60 * time.Second,
		ShutdownTimeout: 30 * time.Second,
	}
}

// Server represents the HTTP API server
type Server struct {
	config     ServerConfig
	handler    *Handler
	httpServer *http.Server
	validator  TokenValidator
}

// NewServer creates a new API server
func NewServer(config ServerConfig, handler *Handler, validator TokenValidator) *Server {
	return &Server{
		config:    config,
		handler:   handler,
		validator: validator,
	}
}

// SetupRoutes configures all API routes
func (s *Server) SetupRoutes() http.Handler {
	mux := http.NewServeMux()

	// Apply default middleware
	defaultMiddleware := DefaultMiddlewareChain(s.validator)
	authMiddleware := AuthenticatedMiddlewareChain(s.validator)

	// Health check endpoint (no auth required)
	mux.Handle("GET /health", defaultMiddleware(http.HandlerFunc(s.handleHealth)))
	mux.Handle("GET /ready", defaultMiddleware(http.HandlerFunc(s.handleReady)))

	// API version info
	mux.Handle("GET /api/v1", defaultMiddleware(http.HandlerFunc(s.handleAPIInfo)))

	// Pet routes
	mux.Handle("POST /api/v1/pets", authMiddleware(http.HandlerFunc(s.handler.HandleCreatePet)))
	mux.Handle("GET /api/v1/pets", authMiddleware(http.HandlerFunc(s.handler.HandleGetPets)))
	mux.Handle("GET /api/v1/pets/{petId}", authMiddleware(http.HandlerFunc(s.handler.HandleGetPet)))
	mux.Handle("PUT /api/v1/pets/{petId}", authMiddleware(http.HandlerFunc(s.handler.HandleUpdatePet)))
	mux.Handle("DELETE /api/v1/pets/{petId}", authMiddleware(http.HandlerFunc(s.handler.HandleDeletePet)))
	mux.Handle("GET /api/v1/pets/{petId}/status", authMiddleware(http.HandlerFunc(s.handler.HandleGetPetStatus)))

	// Interaction routes
	mux.Handle("POST /api/v1/pets/{petId}/interactions", authMiddleware(http.HandlerFunc(s.handler.HandleInteraction)))
	mux.Handle("POST /api/v1/pets/{petId}/interactions/feed", authMiddleware(http.HandlerFunc(s.handler.HandleFeed)))
	mux.Handle("POST /api/v1/pets/{petId}/interactions/play", authMiddleware(http.HandlerFunc(s.handler.HandlePlay)))

	// Social routes
	mux.Handle("PUT /api/v1/social/location", authMiddleware(http.HandlerFunc(s.handler.HandleUpdateLocation)))
	mux.Handle("GET /api/v1/social/nearby", authMiddleware(http.HandlerFunc(s.handler.HandleGetNearbyPets)))
	mux.Handle("GET /api/v1/social/friends", authMiddleware(http.HandlerFunc(s.handler.HandleGetFriends)))
	mux.Handle("POST /api/v1/social/friends/request", authMiddleware(http.HandlerFunc(s.handler.HandleSendFriendRequest)))
	mux.Handle("POST /api/v1/social/friends/requests/{requestId}/accept", authMiddleware(http.HandlerFunc(s.handler.HandleAcceptFriendRequest)))
	mux.Handle("POST /api/v1/social/friends/requests/{requestId}/decline", authMiddleware(http.HandlerFunc(s.handler.HandleDeclineFriendRequest)))
	mux.Handle("DELETE /api/v1/social/friends/{petId}", authMiddleware(http.HandlerFunc(s.handler.HandleRemoveFriend)))

	// Sync routes
	mux.Handle("POST /api/v1/sync", authMiddleware(http.HandlerFunc(s.handler.HandleSync)))
	mux.Handle("POST /api/v1/sync/resolve", authMiddleware(http.HandlerFunc(s.handler.HandleResolveConflict)))

	return mux
}

// handleHealth handles GET /health
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeSuccess(w, map[string]interface{}{
		"status":  "healthy",
		"version": "1.0.0",
		"time":    time.Now().Format(time.RFC3339),
	})
}

// handleReady handles GET /ready
func (s *Server) handleReady(w http.ResponseWriter, r *http.Request) {
	// Check if all dependencies are ready
	// For now, just return ready
	writeSuccess(w, map[string]interface{}{
		"status": "ready",
		"checks": map[string]string{
			"database": "ok",
			"cache":    "ok",
		},
	})
}

// handleAPIInfo handles GET /api/v1
func (s *Server) handleAPIInfo(w http.ResponseWriter, r *http.Request) {
	writeSuccess(w, map[string]interface{}{
		"name":        "Gochi API",
		"version":     "1.0.0",
		"description": "Digital Pet System API",
		"endpoints": map[string]string{
			"pets":         "/api/v1/pets",
			"interactions": "/api/v1/pets/{petId}/interactions",
			"social":       "/api/v1/social",
			"feed":         "/api/v1/feed",
			"events":       "/api/v1/events",
			"leaderboards": "/api/v1/leaderboards",
			"sync":         "/api/v1/sync",
			"achievements": "/api/v1/achievements",
		},
		"documentation": "/api/v1/docs",
	})
}

// Start starts the HTTP server
func (s *Server) Start() error {
	addr := fmt.Sprintf("%s:%d", s.config.Host, s.config.Port)

	s.httpServer = &http.Server{
		Addr:         addr,
		Handler:      s.SetupRoutes(),
		ReadTimeout:  s.config.ReadTimeout,
		WriteTimeout: s.config.WriteTimeout,
		IdleTimeout:  s.config.IdleTimeout,
	}

	log.Printf("Starting server on %s", addr)

	if s.config.TLSCertFile != "" && s.config.TLSKeyFile != "" {
		return s.httpServer.ListenAndServeTLS(s.config.TLSCertFile, s.config.TLSKeyFile)
	}

	return s.httpServer.ListenAndServe()
}

// StartWithGracefulShutdown starts the server with graceful shutdown support
func (s *Server) StartWithGracefulShutdown() error {
	// Channel to receive OS signals
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	// Start server in goroutine
	errChan := make(chan error, 1)
	go func() {
		if err := s.Start(); err != nil && err != http.ErrServerClosed {
			errChan <- err
		}
	}()

	// Wait for shutdown signal or error
	select {
	case err := <-errChan:
		return err
	case <-stop:
		log.Println("Shutting down server...")
	}

	// Create shutdown context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), s.config.ShutdownTimeout)
	defer cancel()

	// Attempt graceful shutdown
	if err := s.httpServer.Shutdown(ctx); err != nil {
		log.Printf("Server forced to shutdown: %v", err)
		return err
	}

	log.Println("Server stopped gracefully")
	return nil
}

// Stop stops the HTTP server
func (s *Server) Stop(ctx context.Context) error {
	if s.httpServer == nil {
		return nil
	}
	return s.httpServer.Shutdown(ctx)
}

// ============================================================================
// API Documentation Types
// ============================================================================

// APIDoc represents API documentation
type APIDoc struct {
	Title       string          `json:"title"`
	Version     string          `json:"version"`
	Description string          `json:"description"`
	BaseURL     string          `json:"base_url"`
	Endpoints   []EndpointDoc   `json:"endpoints"`
}

// EndpointDoc represents documentation for a single endpoint
type EndpointDoc struct {
	Method       string            `json:"method"`
	Path         string            `json:"path"`
	Description  string            `json:"description"`
	RequiresAuth bool              `json:"requires_auth"`
	Parameters   []ParameterDoc    `json:"parameters,omitempty"`
	RequestBody  *RequestBodyDoc   `json:"request_body,omitempty"`
	Responses    []ResponseDoc     `json:"responses"`
}

// ParameterDoc represents a parameter in API documentation
type ParameterDoc struct {
	Name        string `json:"name"`
	In          string `json:"in"` // path, query, header
	Type        string `json:"type"`
	Required    bool   `json:"required"`
	Description string `json:"description"`
}

// RequestBodyDoc represents request body documentation
type RequestBodyDoc struct {
	ContentType string      `json:"content_type"`
	Schema      interface{} `json:"schema"`
	Example     interface{} `json:"example,omitempty"`
}

// ResponseDoc represents response documentation
type ResponseDoc struct {
	StatusCode  int         `json:"status_code"`
	Description string      `json:"description"`
	Schema      interface{} `json:"schema,omitempty"`
	Example     interface{} `json:"example,omitempty"`
}

// GenerateAPIDoc generates API documentation
func GenerateAPIDoc(baseURL string) *APIDoc {
	return &APIDoc{
		Title:       "Gochi Digital Pet API",
		Version:     "1.0.0",
		Description: "RESTful API for the Gochi digital pet system",
		BaseURL:     baseURL,
		Endpoints:   generateEndpointDocs(),
	}
}

// generateEndpointDocs generates documentation for all endpoints
func generateEndpointDocs() []EndpointDoc {
	var docs []EndpointDoc

	for _, group := range GetRoutes() {
		for _, route := range group.Routes {
			docs = append(docs, EndpointDoc{
				Method:       route.Method,
				Path:         group.Prefix + route.Path,
				Description:  route.Description,
				RequiresAuth: route.RequiresAuth,
				Responses: []ResponseDoc{
					{StatusCode: 200, Description: "Success"},
					{StatusCode: 400, Description: "Bad Request"},
					{StatusCode: 401, Description: "Unauthorized"},
					{StatusCode: 500, Description: "Internal Server Error"},
				},
			})
		}
	}

	return docs
}
