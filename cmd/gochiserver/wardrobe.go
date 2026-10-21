package main

import (
	"errors"
	"log"
	"net/http"
	"sort"
	"time"

	"github.com/Michael-W-Ellison/gochi/internal/calendar"
	"github.com/Michael-W-Ellison/gochi/internal/core"
)

// Cosmetics, care supplies, the first-aid kit and holidays

// handleBuyCosmetic handles POST /api/cosmetics/buy with {"id"}
func (s *Server) handleBuyCosmetic(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ID string `json:"id"`
	}
	if !decodePost(w, r, &req) {
		return
	}
	s.buyLocked(w, req.ID, func() error { return s.pet.BuyCosmetic(req.ID) })
}

// handleWear handles POST /api/wear with {"slot", "id"}: puts on a
// cosmetic the active pet owns ("" takes off what is in slot)
func (s *Server) handleWear(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Slot string `json:"slot"`
		ID   string `json:"id"`
	}
	if !decodePost(w, r, &req) {
		return
	}
	s.mu.Lock()
	if s.pet == nil {
		s.mu.Unlock()
		writeJSON(w, http.StatusNotFound, APIResponse{Success: false, Error: "No pet found"})
		return
	}
	s.markActiveLocked(time.Now())
	err := s.pet.Wear(req.Slot, req.ID)
	worn := s.pet.Worn()
	s.mu.Unlock()
	switch {
	case errors.Is(err, core.ErrNotOwned):
		writeJSON(w, http.StatusConflict, APIResponse{Success: false, Error: "Buy it first"})
	case errors.Is(err, core.ErrHolidayOff):
		writeJSON(w, http.StatusConflict, APIResponse{Success: false, Error: "That holiday is turned off"})
	case err != nil:
		writeJSON(w, http.StatusBadRequest, APIResponse{Success: false, Error: "That can't be worn there"})
	default:
		if _, err := s.saveNow(); err != nil {
			log.Printf("Failed to save the outfit: %v", err)
		}
		writeJSON(w, http.StatusOK, APIResponse{Success: true, Data: map[string]interface{}{"outfit": worn}})
	}
}

// handleGiveCare handles POST /api/care/give with {"id"}: gives the
// active pet a care supply from the pantry (a vaccination, a filter for
// its fountain, and so on), or with {"id": "first_aid"} treats an injury
// with the first-aid kit
func (s *Server) handleGiveCare(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ID string `json:"id"`
	}
	if !decodePost(w, r, &req) {
		return
	}
	s.mu.Lock()
	if s.pet == nil {
		s.mu.Unlock()
		writeJSON(w, http.StatusNotFound, APIResponse{Success: false, Error: "No pet found"})
		return
	}
	s.markActiveLocked(time.Now())
	name := s.pet.Name
	var text string
	var err error
	if req.ID == "first_aid" {
		text, err = s.pet.UseFirstAid()
	} else {
		text, err = s.pet.GiveCareSupply(req.ID)
	}
	s.mu.Unlock()

	switch {
	case errors.Is(err, core.ErrNoSupply):
		writeJSON(w, http.StatusConflict, APIResponse{Success: false, Error: "There's none in the pantry. Buy some in the shop."})
	case errors.Is(err, core.ErrNoFirstAid):
		writeJSON(w, http.StatusConflict, APIResponse{Success: false, Error: "The first-aid kit is empty, or you don't have one."})
	case errors.Is(err, core.ErrNotInjured):
		writeJSON(w, http.StatusConflict, APIResponse{Success: false, Error: name + " isn't hurt."})
	case errors.Is(err, core.ErrNotNeeded):
		writeJSON(w, http.StatusConflict, APIResponse{Success: false, Error: capitalize(err.Error()) + "."})
	case errors.Is(err, core.ErrPetNotAvailable):
		writeJSON(w, http.StatusConflict, APIResponse{Success: false, Error: name + " has passed away."})
	case err != nil:
		writeJSON(w, http.StatusBadRequest, APIResponse{Success: false, Error: "Unknown supply"})
	default:
		if _, err := s.saveNow(); err != nil {
			log.Printf("Failed to save after care: %v", err)
		}
		writeJSON(w, http.StatusOK, APIResponse{Success: true, Data: map[string]string{"text": text}})
	}
}

// holidayStatus is a holiday as the page shows it
type holidayStatus struct {
	core.Holiday
	On    bool   `json:"on"`    // Not turned off by a parent
	Open  bool   `json:"open"`  // Its items are on sale now
	Start string `json:"start"` // Its next (or current) dates
	End   string `json:"end"`
}

// holidaysStatus lists every holiday, with whether it is on and when it is
func holidaysStatus() []holidayStatus {
	today := calendar.DateOf(time.Now())
	list := []holidayStatus{}
	for _, h := range core.Holidays() {
		start, end := h.Next(today)
		list = append(list, holidayStatus{Holiday: h, On: core.HolidayOn(h.ID), Open: core.HolidayOpen(h.ID),
			Start: start.Time().Format("2006-01-02"), End: end.Time().Format("2006-01-02")})
	}
	// Soonest first
	sort.SliceStable(list, func(i, j int) bool { return list[i].Start < list[j].Start })
	return list
}

// handleHolidays handles GET /api/holidays - every holiday, when it is
// next, and whether it is on
func (s *Server) handleHolidays(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, APIResponse{Success: false, Error: "Method not allowed"})
		return
	}
	allOff := false
	if s.house != nil {
		allOff = s.house.snapshot().AllHolidaysOff
	}
	writeJSON(w, http.StatusOK, APIResponse{Success: true, Data: map[string]interface{}{
		"holidays": holidaysStatus(), "all_off": allOff}})
}
