package api

import (
	"net/http"
)

// Router defines the interface for HTTP routing
type Router interface {
	Handle(pattern string, handler http.Handler)
	HandleFunc(pattern string, handler func(http.ResponseWriter, *http.Request))
	ServeHTTP(http.ResponseWriter, *http.Request)
}

// Route represents a single API route
type Route struct {
	Method      string
	Path        string
	Handler     http.HandlerFunc
	Middleware  []Middleware
	RequiresAuth bool
	Description string
}

// RouteGroup represents a group of routes with a common prefix
type RouteGroup struct {
	Prefix     string
	Middleware []Middleware
	Routes     []Route
}

// Middleware is a function that wraps an HTTP handler
type Middleware func(http.Handler) http.Handler

// GetRoutes returns all API routes organized by group
func GetRoutes() []RouteGroup {
	return []RouteGroup{
		authRoutes(),
		petRoutes(),
		interactionRoutes(),
		socialRoutes(),
		feedRoutes(),
		eventRoutes(),
		leaderboardRoutes(),
		memoryRoutes(),
		breedingRoutes(),
		syncRoutes(),
		achievementRoutes(),
		environmentRoutes(),
	}
}

// ============================================================================
// Authentication Routes
// ============================================================================

func authRoutes() RouteGroup {
	return RouteGroup{
		Prefix: "/api/v1/auth",
		Routes: []Route{
			{
				Method:      "POST",
				Path:        "/register",
				Description: "Register a new user account",
				RequiresAuth: false,
			},
			{
				Method:      "POST",
				Path:        "/login",
				Description: "Authenticate and receive access tokens",
				RequiresAuth: false,
			},
			{
				Method:      "POST",
				Path:        "/logout",
				Description: "Invalidate current session",
				RequiresAuth: true,
			},
			{
				Method:      "POST",
				Path:        "/refresh",
				Description: "Refresh access token using refresh token",
				RequiresAuth: false,
			},
			{
				Method:      "POST",
				Path:        "/password/reset",
				Description: "Request password reset email",
				RequiresAuth: false,
			},
			{
				Method:      "PUT",
				Path:        "/password",
				Description: "Change password",
				RequiresAuth: true,
			},
			{
				Method:      "GET",
				Path:        "/me",
				Description: "Get current user profile",
				RequiresAuth: true,
			},
			{
				Method:      "PUT",
				Path:        "/me",
				Description: "Update current user profile",
				RequiresAuth: true,
			},
		},
	}
}

// ============================================================================
// Pet Routes
// ============================================================================

func petRoutes() RouteGroup {
	return RouteGroup{
		Prefix: "/api/v1/pets",
		Routes: []Route{
			{
				Method:      "POST",
				Path:        "",
				Description: "Create a new pet",
				RequiresAuth: true,
			},
			{
				Method:      "GET",
				Path:        "",
				Description: "List all pets owned by user",
				RequiresAuth: true,
			},
			{
				Method:      "GET",
				Path:        "/{petId}",
				Description: "Get detailed pet information",
				RequiresAuth: true,
			},
			{
				Method:      "PUT",
				Path:        "/{petId}",
				Description: "Update pet properties (name, location)",
				RequiresAuth: true,
			},
			{
				Method:      "DELETE",
				Path:        "/{petId}",
				Description: "Delete/release a pet",
				RequiresAuth: true,
			},
			{
				Method:      "GET",
				Path:        "/{petId}/status",
				Description: "Get lightweight pet status for polling",
				RequiresAuth: true,
			},
			{
				Method:      "GET",
				Path:        "/{petId}/vitals",
				Description: "Get detailed vital statistics",
				RequiresAuth: true,
			},
			{
				Method:      "GET",
				Path:        "/{petId}/emotions",
				Description: "Get detailed emotional state",
				RequiresAuth: true,
			},
			{
				Method:      "GET",
				Path:        "/{petId}/personality",
				Description: "Get personality traits and description",
				RequiresAuth: true,
			},
			{
				Method:      "POST",
				Path:        "/{petId}/update",
				Description: "Trigger simulation update for delta time",
				RequiresAuth: true,
			},
		},
	}
}

// ============================================================================
// Interaction Routes
// ============================================================================

func interactionRoutes() RouteGroup {
	return RouteGroup{
		Prefix: "/api/v1/pets/{petId}/interactions",
		Routes: []Route{
			{
				Method:      "POST",
				Path:        "",
				Description: "Perform a generic interaction",
				RequiresAuth: true,
			},
			{
				Method:      "POST",
				Path:        "/feed",
				Description: "Feed the pet",
				RequiresAuth: true,
			},
			{
				Method:      "POST",
				Path:        "/pet",
				Description: "Pet/stroke the pet",
				RequiresAuth: true,
			},
			{
				Method:      "POST",
				Path:        "/play",
				Description: "Play with the pet",
				RequiresAuth: true,
			},
			{
				Method:      "POST",
				Path:        "/train",
				Description: "Train the pet",
				RequiresAuth: true,
			},
			{
				Method:      "POST",
				Path:        "/groom",
				Description: "Groom the pet",
				RequiresAuth: true,
			},
			{
				Method:      "POST",
				Path:        "/medical",
				Description: "Provide medical care",
				RequiresAuth: true,
			},
			{
				Method:      "POST",
				Path:        "/reward",
				Description: "Give the pet a reward",
				RequiresAuth: true,
			},
			{
				Method:      "POST",
				Path:        "/discipline",
				Description: "Discipline the pet",
				RequiresAuth: true,
			},
			{
				Method:      "GET",
				Path:        "/history",
				Description: "Get interaction history",
				RequiresAuth: true,
			},
		},
	}
}

// ============================================================================
// Social Routes
// ============================================================================

func socialRoutes() RouteGroup {
	return RouteGroup{
		Prefix: "/api/v1/social",
		Routes: []Route{
			// Location
			{
				Method:      "PUT",
				Path:        "/location",
				Description: "Update user's location for nearby features",
				RequiresAuth: true,
			},
			// Nearby Pets
			{
				Method:      "GET",
				Path:        "/nearby",
				Description: "Get nearby pets within radius",
				RequiresAuth: true,
			},
			// Friends
			{
				Method:      "GET",
				Path:        "/friends",
				Description: "Get list of friends",
				RequiresAuth: true,
			},
			{
				Method:      "POST",
				Path:        "/friends/request",
				Description: "Send a friend request",
				RequiresAuth: true,
			},
			{
				Method:      "GET",
				Path:        "/friends/requests",
				Description: "Get pending friend requests",
				RequiresAuth: true,
			},
			{
				Method:      "POST",
				Path:        "/friends/requests/{requestId}/accept",
				Description: "Accept a friend request",
				RequiresAuth: true,
			},
			{
				Method:      "POST",
				Path:        "/friends/requests/{requestId}/decline",
				Description: "Decline a friend request",
				RequiresAuth: true,
			},
			{
				Method:      "DELETE",
				Path:        "/friends/{petId}",
				Description: "Remove a friend",
				RequiresAuth: true,
			},
			// Social Interactions
			{
				Method:      "POST",
				Path:        "/interact",
				Description: "Interact with another pet (wave, play, etc.)",
				RequiresAuth: true,
			},
			{
				Method:      "GET",
				Path:        "/interactions/history",
				Description: "Get social interaction history",
				RequiresAuth: true,
			},
			// Pet Search
			{
				Method:      "GET",
				Path:        "/search",
				Description: "Search for pets by name",
				RequiresAuth: true,
			},
			// Pet Profile (public view)
			{
				Method:      "GET",
				Path:        "/pets/{petId}",
				Description: "Get public profile of another pet",
				RequiresAuth: true,
			},
		},
	}
}

// ============================================================================
// Social Feed Routes
// ============================================================================

func feedRoutes() RouteGroup {
	return RouteGroup{
		Prefix: "/api/v1/feed",
		Routes: []Route{
			{
				Method:      "GET",
				Path:        "",
				Description: "Get social feed items",
				RequiresAuth: true,
			},
			{
				Method:      "POST",
				Path:        "",
				Description: "Post to social feed",
				RequiresAuth: true,
			},
			{
				Method:      "GET",
				Path:        "/{itemId}",
				Description: "Get a specific feed item",
				RequiresAuth: true,
			},
			{
				Method:      "DELETE",
				Path:        "/{itemId}",
				Description: "Delete a feed post",
				RequiresAuth: true,
			},
			{
				Method:      "POST",
				Path:        "/{itemId}/like",
				Description: "Like a feed item",
				RequiresAuth: true,
			},
			{
				Method:      "DELETE",
				Path:        "/{itemId}/like",
				Description: "Unlike a feed item",
				RequiresAuth: true,
			},
		},
	}
}

// ============================================================================
// Event Routes
// ============================================================================

func eventRoutes() RouteGroup {
	return RouteGroup{
		Prefix: "/api/v1/events",
		Routes: []Route{
			{
				Method:      "GET",
				Path:        "",
				Description: "Get active events",
				RequiresAuth: true,
			},
			{
				Method:      "GET",
				Path:        "/nearby",
				Description: "Get nearby events",
				RequiresAuth: true,
			},
			{
				Method:      "GET",
				Path:        "/{eventId}",
				Description: "Get event details",
				RequiresAuth: true,
			},
			{
				Method:      "POST",
				Path:        "/{eventId}/join",
				Description: "Join an event",
				RequiresAuth: true,
			},
			{
				Method:      "POST",
				Path:        "/{eventId}/leave",
				Description: "Leave an event",
				RequiresAuth: true,
			},
			{
				Method:      "GET",
				Path:        "/{eventId}/participants",
				Description: "Get event participants",
				RequiresAuth: true,
			},
		},
	}
}

// ============================================================================
// Leaderboard Routes
// ============================================================================

func leaderboardRoutes() RouteGroup {
	return RouteGroup{
		Prefix: "/api/v1/leaderboards",
		Routes: []Route{
			{
				Method:      "GET",
				Path:        "",
				Description: "Get list of available leaderboards",
				RequiresAuth: true,
			},
			{
				Method:      "GET",
				Path:        "/{leaderboardId}",
				Description: "Get leaderboard entries",
				RequiresAuth: true,
			},
			{
				Method:      "GET",
				Path:        "/{leaderboardId}/rank",
				Description: "Get user's rank on leaderboard",
				RequiresAuth: true,
			},
		},
	}
}

// ============================================================================
// Memory Routes
// ============================================================================

func memoryRoutes() RouteGroup {
	return RouteGroup{
		Prefix: "/api/v1/pets/{petId}/memories",
		Routes: []Route{
			{
				Method:      "GET",
				Path:        "",
				Description: "Get pet memories (paginated)",
				RequiresAuth: true,
			},
			{
				Method:      "GET",
				Path:        "/summary",
				Description: "Get memory system summary",
				RequiresAuth: true,
			},
			{
				Method:      "GET",
				Path:        "/search",
				Description: "Search memories by tag or description",
				RequiresAuth: true,
			},
			{
				Method:      "GET",
				Path:        "/{memoryId}",
				Description: "Get a specific memory",
				RequiresAuth: true,
			},
			{
				Method:      "GET",
				Path:        "/positive",
				Description: "Get positive memories",
				RequiresAuth: true,
			},
			{
				Method:      "GET",
				Path:        "/traumatic",
				Description: "Get traumatic memories",
				RequiresAuth: true,
			},
		},
	}
}

// ============================================================================
// Breeding Routes
// ============================================================================

func breedingRoutes() RouteGroup {
	return RouteGroup{
		Prefix: "/api/v1/breeding",
		Routes: []Route{
			{
				Method:      "POST",
				Path:        "/compatibility",
				Description: "Check breeding compatibility between two pets",
				RequiresAuth: true,
			},
			{
				Method:      "POST",
				Path:        "/breed",
				Description: "Breed two compatible pets",
				RequiresAuth: true,
			},
			{
				Method:      "GET",
				Path:        "/offspring/{petId}",
				Description: "Get offspring of a pet",
				RequiresAuth: true,
			},
			{
				Method:      "GET",
				Path:        "/parents/{petId}",
				Description: "Get parents of a pet",
				RequiresAuth: true,
			},
			{
				Method:      "GET",
				Path:        "/genetics/{petId}",
				Description: "Get genetic information of a pet",
				RequiresAuth: true,
			},
		},
	}
}

// ============================================================================
// Cloud Sync Routes
// ============================================================================

func syncRoutes() RouteGroup {
	return RouteGroup{
		Prefix: "/api/v1/sync",
		Routes: []Route{
			{
				Method:      "POST",
				Path:        "",
				Description: "Synchronize pet data with cloud",
				RequiresAuth: true,
			},
			{
				Method:      "GET",
				Path:        "/status",
				Description: "Get sync status",
				RequiresAuth: true,
			},
			{
				Method:      "POST",
				Path:        "/resolve",
				Description: "Resolve sync conflict",
				RequiresAuth: true,
			},
			{
				Method:      "GET",
				Path:        "/history",
				Description: "Get sync history",
				RequiresAuth: true,
			},
			{
				Method:      "POST",
				Path:        "/backup",
				Description: "Create manual backup",
				RequiresAuth: true,
			},
			{
				Method:      "GET",
				Path:        "/backups",
				Description: "List available backups",
				RequiresAuth: true,
			},
			{
				Method:      "POST",
				Path:        "/restore/{backupId}",
				Description: "Restore from backup",
				RequiresAuth: true,
			},
		},
	}
}

// ============================================================================
// Achievement Routes
// ============================================================================

func achievementRoutes() RouteGroup {
	return RouteGroup{
		Prefix: "/api/v1/achievements",
		Routes: []Route{
			{
				Method:      "GET",
				Path:        "",
				Description: "Get all achievements with unlock status",
				RequiresAuth: true,
			},
			{
				Method:      "GET",
				Path:        "/{achievementId}",
				Description: "Get specific achievement details",
				RequiresAuth: true,
			},
			{
				Method:      "GET",
				Path:        "/categories",
				Description: "Get achievement categories",
				RequiresAuth: true,
			},
			{
				Method:      "GET",
				Path:        "/recent",
				Description: "Get recently unlocked achievements",
				RequiresAuth: true,
			},
			{
				Method:      "POST",
				Path:        "/{achievementId}/share",
				Description: "Share achievement to social feed",
				RequiresAuth: true,
			},
		},
	}
}

// ============================================================================
// Environment Routes
// ============================================================================

func environmentRoutes() RouteGroup {
	return RouteGroup{
		Prefix: "/api/v1/environment",
		Routes: []Route{
			{
				Method:      "GET",
				Path:        "",
				Description: "Get current environment state",
				RequiresAuth: true,
			},
			{
				Method:      "GET",
				Path:        "/weather",
				Description: "Get current weather",
				RequiresAuth: true,
			},
			{
				Method:      "GET",
				Path:        "/forecast",
				Description: "Get weather forecast",
				RequiresAuth: true,
			},
		},
	}
}

// ============================================================================
// WebSocket Routes
// ============================================================================

// WebSocketRoute represents a WebSocket endpoint
type WebSocketRoute struct {
	Path        string
	Description string
}

// GetWebSocketRoutes returns WebSocket endpoints
func GetWebSocketRoutes() []WebSocketRoute {
	return []WebSocketRoute{
		{
			Path:        "/api/v1/ws",
			Description: "Main WebSocket connection for real-time updates",
		},
		{
			Path:        "/api/v1/ws/pet/{petId}",
			Description: "Pet-specific WebSocket for detailed updates",
		},
	}
}
