// Package api defines the HTTP API handlers and routes for the Gochi digital pet system.
//
// This package provides:
//   - RESTful endpoints for pet management (create, read, update, delete)
//   - Interaction endpoints for feeding, playing, training, etc.
//   - Social feature endpoints for friends, nearby pets, events
//   - Cloud synchronization endpoints
//   - WebSocket support for real-time updates
//
// The API follows RESTful conventions with JSON request/response bodies.
// Authentication is handled via Bearer tokens in the Authorization header.
package api
