package hub

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/Michael-W-Ellison/gochi/api"
)

// The hub's HTTP API, under /hub/v1. Every route but status and register
// needs "Authorization: Bearer <token>".
//
//	GET    /hub/v1/status              Whether this is a hub and registration is open
//	POST   /hub/v1/register            {"username", "join_code"} -> {"token"}
//	GET    /hub/v1/pets                The player's pets
//	GET    /hub/v1/pets/{key}          A pet's information and save
//	PUT    /hub/v1/pets/{key}          {"base_version", "info", "save"} -> {"version"}; 409 on a conflict
//	DELETE /hub/v1/pets/{key}?base_version=n
//	GET    /hub/v1/profile             The player's area and sharing
//	PUT    /hub/v1/profile             {"area", "sharing"}
//	GET    /hub/v1/nearby              Other players' pets in the same area
//	GET    /hub/v1/achievements        Shared achievements, newest first
//	POST   /hub/v1/achievements        Share an achievement

// requestsPerMinute is how many requests each client may make
const requestsPerMinute = 120

// Handler returns the hub's HTTP API
func (h *Hub) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /hub/v1/status", h.handleStatus)
	mux.HandleFunc("POST /hub/v1/register", h.handleRegister)
	mux.HandleFunc("GET /hub/v1/pets", h.authed(h.handleListPets))
	mux.HandleFunc("GET /hub/v1/pets/{key}", h.authed(h.handleGetPet))
	mux.HandleFunc("PUT /hub/v1/pets/{key}", h.authed(h.handlePutPet))
	mux.HandleFunc("DELETE /hub/v1/pets/{key}", h.authed(h.handleDeletePet))
	mux.HandleFunc("GET /hub/v1/profile", h.authed(h.handleGetProfile))
	mux.HandleFunc("PUT /hub/v1/profile", h.authed(h.handlePutProfile))
	mux.HandleFunc("GET /hub/v1/nearby", h.authed(h.handleNearby))
	mux.HandleFunc("GET /hub/v1/achievements", h.authed(h.handleFeed))
	mux.HandleFunc("POST /hub/v1/achievements", h.authed(h.handleShare))
	limit := api.RateLimitMiddleware(api.NewSlidingWindowLimiter(requestsPerMinute, time.Minute))
	return limit(mux)
}

// reply writes a JSON response
func reply(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

// errorReply is the body of an error response
type errorReply struct {
	Error   string `json:"error"`
	Version int    `json:"version,omitempty"` // The hub's version, on a conflict
}

// fail writes an error response, with the status that suits the error
func fail(w http.ResponseWriter, err error) {
	status := http.StatusBadRequest
	switch {
	case errors.Is(err, ErrUnauthorized), errors.Is(err, ErrWrongJoinCode):
		status = http.StatusUnauthorized
	case errors.Is(err, ErrRegistrationClosed):
		status = http.StatusForbidden
	case errors.Is(err, ErrNotFound):
		status = http.StatusNotFound
	case errors.Is(err, ErrUsernameTaken), errors.Is(err, ErrVersionConflict), errors.Is(err, ErrTooManyPets):
		status = http.StatusConflict
	}
	reply(w, status, errorReply{Error: err.Error()})
}

// authed wraps a handler that needs the player's username
func (h *Hub) authed(next func(w http.ResponseWriter, r *http.Request, username string)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token, ok := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer ")
		username, err := h.Authenticate(token)
		if !ok || err != nil {
			fail(w, ErrUnauthorized)
			return
		}
		next(w, r, username)
	}
}

// decode reads a JSON request body of at most limit bytes
func decode(w http.ResponseWriter, r *http.Request, limit int64, v interface{}) bool {
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, limit)).Decode(v); err != nil {
		fail(w, errors.New("invalid request"))
		return false
	}
	return true
}

// Status is what GET /hub/v1/status returns
type Status struct {
	Hub              string `json:"hub"` // Always "gochi"
	RegistrationOpen bool   `json:"registration_open"`
}

func (h *Hub) handleStatus(w http.ResponseWriter, r *http.Request) {
	reply(w, http.StatusOK, Status{Hub: "gochi", RegistrationOpen: h.RegistrationOpen()})
}

func (h *Hub) handleRegister(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Username string `json:"username"`
		JoinCode string `json:"join_code"`
	}
	if !decode(w, r, 4096, &req) {
		return
	}
	token, err := h.Register(req.Username, req.JoinCode)
	if err != nil {
		fail(w, err)
		return
	}
	reply(w, http.StatusCreated, map[string]string{"token": token})
}

func (h *Hub) handleListPets(w http.ResponseWriter, r *http.Request, username string) {
	pets, err := h.ListPets(username)
	if err != nil {
		fail(w, err)
		return
	}
	reply(w, http.StatusOK, pets)
}

// petReply is a pet with its save
type petReply struct {
	Info PetInfo         `json:"info"`
	Save json.RawMessage `json:"save,omitempty"`
}

func (h *Hub) handleGetPet(w http.ResponseWriter, r *http.Request, username string) {
	info, save, err := h.GetPet(username, r.PathValue("key"))
	if err != nil {
		fail(w, err)
		return
	}
	reply(w, http.StatusOK, petReply{Info: info, Save: save})
}

// putRequest is a pet upload
type putRequest struct {
	BaseVersion int             `json:"base_version"`
	Info        PetInfo         `json:"info"`
	Save        json.RawMessage `json:"save"`
}

func (h *Hub) handlePutPet(w http.ResponseWriter, r *http.Request, username string) {
	var req putRequest
	if !decode(w, r, MaxSaveBytes+64<<10, &req) {
		return
	}
	req.Info.Key = r.PathValue("key")
	version, err := h.PutPet(username, req.Info, req.Save, req.BaseVersion)
	if errors.Is(err, ErrVersionConflict) {
		reply(w, http.StatusConflict, errorReply{Error: err.Error(), Version: version})
		return
	}
	if err != nil {
		fail(w, err)
		return
	}
	reply(w, http.StatusOK, map[string]int{"version": version})
}

func (h *Hub) handleDeletePet(w http.ResponseWriter, r *http.Request, username string) {
	base, err := strconv.Atoi(r.URL.Query().Get("base_version"))
	if err != nil {
		fail(w, errors.New("base_version is required"))
		return
	}
	version, err := h.DeletePet(username, r.PathValue("key"), base)
	if errors.Is(err, ErrVersionConflict) {
		reply(w, http.StatusConflict, errorReply{Error: err.Error(), Version: version})
		return
	}
	if err != nil {
		fail(w, err)
		return
	}
	reply(w, http.StatusOK, map[string]int{"version": version})
}

func (h *Hub) handleGetProfile(w http.ResponseWriter, r *http.Request, username string) {
	profile, err := h.GetProfile(username)
	if err != nil {
		fail(w, err)
		return
	}
	reply(w, http.StatusOK, profile)
}

func (h *Hub) handlePutProfile(w http.ResponseWriter, r *http.Request, username string) {
	var req struct {
		Area    string `json:"area"`
		Sharing bool   `json:"sharing"`
	}
	if !decode(w, r, 4096, &req) {
		return
	}
	profile, err := h.SetProfile(username, req.Area, req.Sharing)
	if err != nil {
		fail(w, err)
		return
	}
	reply(w, http.StatusOK, profile)
}

func (h *Hub) handleNearby(w http.ResponseWriter, r *http.Request, username string) {
	nearby, err := h.Nearby(username)
	if err != nil {
		fail(w, err)
		return
	}
	reply(w, http.StatusOK, nearby)
}

// feedSize is how many shared achievements GET /hub/v1/achievements returns
const feedSize = 50

func (h *Hub) handleFeed(w http.ResponseWriter, r *http.Request, username string) {
	reply(w, http.StatusOK, h.Feed(feedSize))
}

func (h *Hub) handleShare(w http.ResponseWriter, r *http.Request, username string) {
	var a SharedAchievement
	if !decode(w, r, 4096, &a) {
		return
	}
	if err := h.ShareAchievement(username, a); err != nil {
		fail(w, err)
		return
	}
	reply(w, http.StatusCreated, map[string]bool{"shared": true})
}
