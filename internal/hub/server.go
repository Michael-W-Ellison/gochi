package hub

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/Michael-W-Ellison/gochi/api"
	"github.com/Michael-W-Ellison/gochi/pkg/types"
)

// The hub's HTTP API, under /hub/v1. Every route but status and register
// needs "Authorization: Bearer <account key>". Nothing a player sends is
// shown to another player except values from fixed lists and numbers.
//
//	GET    /hub/v1/status              Whether this is a hub and registration is open
//	POST   /hub/v1/register            {"join_code"} -> {"token", "handle"}
//	GET    /hub/v1/pets                The player's pets
//	GET    /hub/v1/pets/{key}          A pet's information and (encrypted) save
//	PUT    /hub/v1/pets/{key}          {"base_version", "info", "save"} -> {"version"}; 409 on a conflict
//	DELETE /hub/v1/pets/{key}?base_version=n
//	GET    /hub/v1/profile             The player's handle and whether they share achievements
//	PUT    /hub/v1/profile             {"share_achievements"}
//	GET    /hub/v1/achievements        Shared achievements, newest first
//	POST   /hub/v1/achievements        {"id", "species"}: share an achievement
//	POST   /hub/v1/playdates           {"card"} -> the new playdate, with its code
//	POST   /hub/v1/playdates/{code}/join    {"card"} -> the playdate
//	POST   /hub/v1/playdates/{code}/update  {"card", "position", "after"} -> the playdate, with events after "after"
//	POST   /hub/v1/playdates/{code}/play    {"to", "quality", "bond"} -> the event the hub wrote
//	POST   /hub/v1/playdates/{code}/leave

// requestsPerMinute is how many requests each client may make
const requestsPerMinute = 120

// requestBytes is the most a request other than a pet upload may hold
const requestBytes = 4096

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
	mux.HandleFunc("GET /hub/v1/achievements", h.authed(h.handleFeed))
	mux.HandleFunc("POST /hub/v1/achievements", h.authed(h.handleShare))
	mux.HandleFunc("POST /hub/v1/playdates", h.authed(h.handleStartPlaydate))
	mux.HandleFunc("POST /hub/v1/playdates/{code}/join", h.authed(h.handleJoinPlaydate))
	mux.HandleFunc("POST /hub/v1/playdates/{code}/update", h.authed(h.handleUpdatePlaydate))
	mux.HandleFunc("POST /hub/v1/playdates/{code}/play", h.authed(h.handlePlay))
	mux.HandleFunc("POST /hub/v1/playdates/{code}/leave", h.authed(h.handleLeavePlaydate))
	limit := api.RateLimitMiddleware(api.NewSlidingWindowLimiter(requestsPerMinute, time.Minute))
	return h.identify(limit(mux))
}

// identify marks requests with a valid account key with the account, so
// each player has their own rate limit even when several share an address
// (as players at a playdate often do); other requests are limited by
// address
func (h *Hub) identify(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if token, ok := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer "); ok {
			if id, err := h.Authenticate(token); err == nil {
				ctx := context.WithValue(r.Context(), api.ContextKeyUserID, types.UserID("hub:"+id))
				r = r.WithContext(ctx)
			}
		}
		next.ServeHTTP(w, r)
	})
}

// reply writes a JSON response
func reply(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Content-Type-Options", "nosniff")
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
	case errors.Is(err, ErrNotFound), errors.Is(err, ErrNoPlaydate), errors.Is(err, ErrNotInPlaydate):
		status = http.StatusNotFound
	case errors.Is(err, ErrVersionConflict), errors.Is(err, ErrTooManyPets), errors.Is(err, ErrPlaydateFull),
		errors.Is(err, ErrAccountFull):
		status = http.StatusConflict
	case errors.Is(err, ErrTooManyPlaydate), errors.Is(err, ErrHubFull):
		status = http.StatusServiceUnavailable
	}
	reply(w, status, errorReply{Error: err.Error()})
}

// authed wraps a handler that needs the player's account ID
func (h *Hub) authed(next func(w http.ResponseWriter, r *http.Request, id string)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token, ok := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer ")
		id, err := h.Authenticate(token)
		if !ok || err != nil {
			fail(w, ErrUnauthorized)
			return
		}
		next(w, r, id)
	}
}

// decode reads a JSON request body of at most limit bytes, refusing
// fields it doesn't know
func decode(w http.ResponseWriter, r *http.Request, limit int64, v interface{}) bool {
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, limit))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(v); err != nil {
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

// registration is what POST /hub/v1/register returns
type registration struct {
	Token  string `json:"token"`
	Handle string `json:"handle"`
}

func (h *Hub) handleRegister(w http.ResponseWriter, r *http.Request) {
	var req struct {
		JoinCode string `json:"join_code"`
	}
	if !decode(w, r, requestBytes, &req) {
		return
	}
	token, handle, err := h.Register(req.JoinCode)
	if err != nil {
		fail(w, err)
		return
	}
	reply(w, http.StatusCreated, registration{Token: token, Handle: handle})
}

func (h *Hub) handleListPets(w http.ResponseWriter, r *http.Request, id string) {
	pets, err := h.ListPets(id)
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

func (h *Hub) handleGetPet(w http.ResponseWriter, r *http.Request, id string) {
	info, save, err := h.GetPet(id, r.PathValue("key"))
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

func (h *Hub) handlePutPet(w http.ResponseWriter, r *http.Request, id string) {
	var req putRequest
	if !decode(w, r, MaxSaveBytes+64<<10, &req) {
		return
	}
	req.Info.Key = r.PathValue("key")
	version, err := h.PutPet(id, req.Info, req.Save, req.BaseVersion)
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

func (h *Hub) handleDeletePet(w http.ResponseWriter, r *http.Request, id string) {
	base, err := strconv.Atoi(r.URL.Query().Get("base_version"))
	if err != nil {
		fail(w, errors.New("base_version is required"))
		return
	}
	version, err := h.DeletePet(id, r.PathValue("key"), base)
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

func (h *Hub) handleGetProfile(w http.ResponseWriter, r *http.Request, id string) {
	profile, err := h.GetProfile(id)
	if err != nil {
		fail(w, err)
		return
	}
	reply(w, http.StatusOK, profile)
}

func (h *Hub) handlePutProfile(w http.ResponseWriter, r *http.Request, id string) {
	var req struct {
		ShareAchievements bool `json:"share_achievements"`
	}
	if !decode(w, r, requestBytes, &req) {
		return
	}
	profile, err := h.SetSharing(id, req.ShareAchievements)
	if err != nil {
		fail(w, err)
		return
	}
	reply(w, http.StatusOK, profile)
}

// feedSize is how many shared achievements GET /hub/v1/achievements returns
const feedSize = 50

func (h *Hub) handleFeed(w http.ResponseWriter, r *http.Request, id string) {
	reply(w, http.StatusOK, h.Feed(feedSize))
}

// shareRequest shares an achievement
type shareRequest struct {
	ID      string `json:"id"`
	Species string `json:"species"`
}

func (h *Hub) handleShare(w http.ResponseWriter, r *http.Request, id string) {
	var req shareRequest
	if !decode(w, r, requestBytes, &req) {
		return
	}
	if err := h.ShareAchievement(id, req.ID, req.Species); err != nil {
		fail(w, err)
		return
	}
	reply(w, http.StatusCreated, map[string]bool{"shared": true})
}

// playdateRequest is a playdate request's body
type playdateRequest struct {
	Card     *CardRequest `json:"card"`
	Position Offset       `json:"position"`
	After    int64        `json:"after"`
}

func (h *Hub) handleStartPlaydate(w http.ResponseWriter, r *http.Request, id string) {
	var req playdateRequest
	if !decode(w, r, requestBytes, &req) {
		return
	}
	if req.Card == nil {
		fail(w, ErrBadCard)
		return
	}
	state, err := h.StartPlaydate(id, *req.Card)
	if err != nil {
		fail(w, err)
		return
	}
	reply(w, http.StatusCreated, state)
}

func (h *Hub) handleJoinPlaydate(w http.ResponseWriter, r *http.Request, id string) {
	var req playdateRequest
	if !decode(w, r, requestBytes, &req) {
		return
	}
	if req.Card == nil {
		fail(w, ErrBadCard)
		return
	}
	state, err := h.JoinPlaydate(id, r.PathValue("code"), *req.Card)
	if err != nil {
		fail(w, err)
		return
	}
	reply(w, http.StatusOK, state)
}

func (h *Hub) handleUpdatePlaydate(w http.ResponseWriter, r *http.Request, id string) {
	var req playdateRequest
	if !decode(w, r, requestBytes, &req) {
		return
	}
	state, err := h.UpdatePlaydate(id, r.PathValue("code"), req.Card, req.Position, req.After)
	if err != nil {
		fail(w, err)
		return
	}
	reply(w, http.StatusOK, state)
}

func (h *Hub) handlePlay(w http.ResponseWriter, r *http.Request, id string) {
	var report PlayReport
	if !decode(w, r, requestBytes, &report) {
		return
	}
	event, err := h.ReportPlay(id, r.PathValue("code"), report)
	if err != nil {
		fail(w, err)
		return
	}
	reply(w, http.StatusCreated, event)
}

func (h *Hub) handleLeavePlaydate(w http.ResponseWriter, r *http.Request, id string) {
	if err := h.LeavePlaydate(id, r.PathValue("code")); err != nil {
		fail(w, err)
		return
	}
	reply(w, http.StatusOK, map[string]bool{"left": true})
}
