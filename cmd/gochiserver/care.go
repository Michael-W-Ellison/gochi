package main

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"slices"
	"time"

	"github.com/Michael-W-Ellison/gochi/internal/core"
)

// Care variants, supplies, environment changes and introductions

// handleCareOptions handles GET /api/care/options - the foods, treats,
// toys, petting spots, training methods and environment changes the
// active pet can be given now
func (s *Server) handleCareOptions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, APIResponse{Success: false, Error: "Method not allowed"})
		return
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.pet == nil {
		writeJSON(w, http.StatusNotFound, APIResponse{Success: false, Error: "No pet found"})
		return
	}
	writeJSON(w, http.StatusOK, APIResponse{Success: true, Data: s.pet.GetCareOptions()})
}

// interactWithChoice gives care with a chosen variant: {"food": id} for
// feed and treat, {"toy": id} for play, {"region": id} for pet. It returns
// false if the request has no variant. Caller must hold s.mu.
func (s *Server) interactWithChoice(action, food, toy, region string) (core.InteractionResult, bool, error) {
	switch {
	case action == "feed" && food != "":
		result, err := s.pet.Feed(food)
		return result, true, err
	case action == "treat" && food != "":
		result, err := s.pet.GiveTreat(food)
		return result, true, err
	case action == "play" && toy != "":
		result, err := s.pet.PlayWith(toy)
		return result, true, err
	case action == "pet" && region != "":
		result, err := s.pet.PetOn(region)
		return result, true, err
	}
	return core.InteractionResult{}, false, nil
}

// handleSupplies handles GET /api/supplies - the supplies the active pet's
// species can use, with prices, rarities and today's discounts
func (s *Server) handleSupplies(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, APIResponse{Success: false, Error: "Method not allowed"})
		return
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.pet == nil {
		writeJSON(w, http.StatusNotFound, APIResponse{Success: false, Error: "No pet found"})
		return
	}
	writeJSON(w, http.StatusOK, APIResponse{Success: true, Data: s.suppliesDataLocked()})
}

// suppliesDataLocked describes the supply shop. Caller must hold s.mu.
func (s *Server) suppliesDataLocked() map[string]interface{} {
	return map[string]interface{}{
		"points":        s.pet.FriendshipPoints,
		"points_earned": s.pet.PointsEarned,
		"supplies":      s.pet.SupplyOffers(),
	}
}

// handleBuySupply handles POST /api/supplies/buy with {"id": supply}
func (s *Server) handleBuySupply(w http.ResponseWriter, r *http.Request) {
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
	err := s.pet.BuySupply(req.ID)
	data := s.suppliesDataLocked()
	s.mu.Unlock()

	switch {
	case errors.Is(err, core.ErrUnknownSupply):
		writeJSON(w, http.StatusBadRequest, APIResponse{Success: false, Error: "Unknown supply"})
	case errors.Is(err, core.ErrSupplyLocked):
		writeJSON(w, http.StatusConflict, APIResponse{Success: false,
			Error: name + " needs to earn more Friendship Points to unlock this"})
	case errors.Is(err, core.ErrNotEnoughPoints):
		writeJSON(w, http.StatusConflict, APIResponse{Success: false, Error: "Not enough Friendship Points"})
	case errors.Is(err, core.ErrPantryFull):
		writeJSON(w, http.StatusConflict, APIResponse{Success: false, Error: "The pantry is full"})
	case err != nil:
		writeJSON(w, http.StatusConflict, APIResponse{Success: false, Error: name + " can't use supplies right now"})
	default:
		if _, err := s.saveNow(); err != nil {
			log.Printf("Failed to save after buying supplies: %v", err)
		}
		writeJSON(w, http.StatusOK, APIResponse{Success: true, Data: data})
	}
}

// handleEnrich handles POST /api/enrich with {"change": id} - changes
// something in the pet's surroundings
func (s *Server) handleEnrich(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Change string `json:"change"`
	}
	if r.Method == http.MethodPost {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, APIResponse{Success: false, Error: "Invalid request"})
			return
		}
	}
	s.withActivePet(w, r, func() {
		if s.pet.IsHibernating() {
			writeActivityError(w, s.pet.Name, core.ErrHibernating)
			return
		}
		if s.pet.IsExploring() {
			writeActivityError(w, s.pet.Name, core.ErrExploring)
			return
		}
		result, err := s.pet.ChangeEnvironment(req.Change)
		s.writeCareResult(w, result, err)
	})
}

// writeCareResult reports care given with a variant. Caller must hold
// s.mu.
func (s *Server) writeCareResult(w http.ResponseWriter, result core.InteractionResult, err error) {
	if err != nil {
		writeActivityError(w, s.pet.Name, err)
		return
	}
	if !result.Applied {
		message := result.Refusal
		if message == "" {
			message = s.pet.Name + " can't do that right now."
		}
		writeJSON(w, http.StatusConflict, APIResponse{Success: false, Error: message})
		return
	}
	writeJSON(w, http.StatusOK, APIResponse{Success: true, Data: s.interactionData(result)})
}

// introductionPartner is a saved pet the active pet can be introduced to
type introductionPartner struct {
	Slot         string `json:"slot"`
	Name         string `json:"name"`
	Species      string `json:"species"`
	Relationship string `json:"relationship"` // How they get along, or "" if they haven't met
}

// handleIntroducePartners handles GET /api/introduce/partners
func (s *Server) handleIntroducePartners(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, APIResponse{Success: false, Error: "Method not allowed"})
		return
	}
	if s.store == nil {
		writeJSON(w, http.StatusOK, APIResponse{Success: true, Data: []introductionPartner{}})
		return
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.pet == nil {
		writeJSON(w, http.StatusNotFound, APIResponse{Success: false, Error: "No pet found"})
		return
	}

	saved, err := s.store.List()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, APIResponse{Success: false, Error: "Could not list saved pets"})
		return
	}
	partners := []introductionPartner{}
	for _, entry := range saved {
		if entry.Slot == s.slot || !entry.IsAlive {
			continue
		}
		partner, _, _, err := s.store.Load(entry.Slot)
		if err != nil {
			continue
		}
		partners = append(partners, introductionPartner{Slot: entry.Slot, Name: partner.Name,
			Species: string(partner.GetSpecies()), Relationship: s.pet.RelationTo(partner.ID)})
	}
	writeJSON(w, http.StatusOK, APIResponse{Success: true, Data: partners})
}

// handleIntroduce handles POST /api/introduce with {"partner_slot": slot}:
// the active pet meets another saved pet, and both are saved
func (s *Server) handleIntroduce(w http.ResponseWriter, r *http.Request) {
	var req struct {
		PartnerSlot string `json:"partner_slot"`
	}
	if !decodePost(w, r, &req) {
		return
	}
	if !ValidSlot(req.PartnerSlot) || s.store == nil {
		writeJSON(w, http.StatusBadRequest, APIResponse{Success: false, Error: "Invalid partner"})
		return
	}

	// Hold the save lock so both pets are saved together
	s.saveMu.Lock()
	defer s.saveMu.Unlock()
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.pet == nil {
		writeJSON(w, http.StatusNotFound, APIResponse{Success: false, Error: "No pet found"})
		return
	}
	if req.PartnerSlot == s.slot {
		writeJSON(w, http.StatusBadRequest, APIResponse{Success: false, Error: s.pet.Name + " can't meet itself"})
		return
	}
	s.markActiveLocked(time.Now())

	partner, _, _, err := s.store.Load(req.PartnerSlot)
	if err != nil {
		writeJSON(w, http.StatusNotFound, APIResponse{Success: false, Error: "No saved pet in that slot"})
		return
	}
	meeting, err := core.Introduce(s.pet, partner)
	if err != nil {
		writeActivityError(w, s.pet.Name, err)
		return
	}

	if err := s.store.SavePet(req.PartnerSlot, partner); err != nil {
		log.Printf("Failed to save %s after meeting: %v", partner.Name, err)
	}
	if err := s.saveActiveLocked(); err != nil {
		log.Printf("Failed to save %s after meeting: %v", s.pet.Name, err)
	}

	data := s.interactionData(meeting.InteractionResult)
	data["text"] = meeting.Text
	data["quality"] = meeting.Quality
	data["relationship"] = meeting.Relationship
	writeJSON(w, http.StatusOK, APIResponse{Success: true, Data: data})
}

// handleGroup handles POST /api/group with {"activity": "play", "walk" or
// "nap", "partner_slots": [slot, ...]}: the active pet spends time with up
// to three other saved pets, and all of them are saved
func (s *Server) handleGroup(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Activity     string   `json:"activity"`
		PartnerSlots []string `json:"partner_slots"`
	}
	if !decodePost(w, r, &req) {
		return
	}
	if s.store == nil || len(req.PartnerSlots) == 0 || len(req.PartnerSlots) > 3 {
		writeJSON(w, http.StatusBadRequest, APIResponse{Success: false, Error: "Choose one to three other pets"})
		return
	}
	for i, slot := range req.PartnerSlots {
		if !ValidSlot(slot) || slices.Contains(req.PartnerSlots[:i], slot) {
			writeJSON(w, http.StatusBadRequest, APIResponse{Success: false, Error: "Invalid partner"})
			return
		}
	}

	// Hold the save lock so every pet is saved together
	s.saveMu.Lock()
	defer s.saveMu.Unlock()
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.pet == nil {
		writeJSON(w, http.StatusNotFound, APIResponse{Success: false, Error: "No pet found"})
		return
	}
	s.markActiveLocked(time.Now())

	pets := []*core.DigitalPet{s.pet}
	for _, slot := range req.PartnerSlots {
		if slot == s.slot {
			writeJSON(w, http.StatusBadRequest, APIResponse{Success: false, Error: s.pet.Name + " is already in the group"})
			return
		}
		partner, _, _, err := s.store.Load(slot)
		if err != nil {
			writeJSON(w, http.StatusNotFound, APIResponse{Success: false, Error: "No saved pet in that slot"})
			return
		}
		pets = append(pets, partner)
	}

	result, err := core.ProcessGroupActivity(req.Activity, pets...)
	switch {
	case errors.Is(err, core.ErrUnknownGroup):
		writeJSON(w, http.StatusBadRequest, APIResponse{Success: false, Error: "Unknown activity"})
		return
	case errors.Is(err, core.ErrDogsOnlyWalk):
		writeJSON(w, http.StatusConflict, APIResponse{Success: false, Error: "Only dogs go for walks together"})
		return
	case err != nil:
		writeActivityError(w, s.pet.Name, err)
		return
	}

	for i, partner := range pets[1:] {
		if err := s.store.SavePet(req.PartnerSlots[i], partner); err != nil {
			log.Printf("Failed to save %s after a group activity: %v", partner.Name, err)
		}
	}
	if err := s.saveActiveLocked(); err != nil {
		log.Printf("Failed to save %s after a group activity: %v", s.pet.Name, err)
	}

	data := s.interactionData(result.InteractionResult)
	data["text"] = result.Text
	data["signals"] = result.Signals
	data["conflicts"] = result.Conflicts
	data["leader"] = result.Leader
	data["order"] = result.Order
	data["harmony"] = result.Harmony
	data["group_happiness"] = result.Happiness
	writeJSON(w, http.StatusOK, APIResponse{Success: true, Data: data})
}
