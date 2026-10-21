// Package api defines the HTTP API handlers and routes for the Gochi digital pet system.
//
// This package provides:
//   - RESTful endpoints for pet management (create, read, update, delete)
//   - Interaction endpoints for feeding, playing and other care
//
// There are no accounts, social or messaging endpoints: the API works only
// on the player's own pets. It follows RESTful conventions with JSON
// request/response bodies. Authentication is handled via a Bearer token in
// the Authorization header.
package api
