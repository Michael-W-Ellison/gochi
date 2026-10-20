package api

import "net/http"

// Middleware is a function that wraps an HTTP handler
type Middleware func(http.Handler) http.Handler
