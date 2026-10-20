package main

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/Michael-W-Ellison/gochi/internal/core"
)

// interactionData is the response data for anything that cares for the
// pet: what changed, how it reacted and the points it earned. Caller must
// hold s.mu.
func (s *Server) interactionData(result core.InteractionResult) map[string]interface{} {
	return map[string]interface{}{
		"message":           "Interaction successful",
		"points_earned":     result.PointsEarned,
		"friendship_points": s.pet.FriendshipPoints,
		"changes":           result.Changes,
		"boosted_by":        result.BoostedBy,
		"reaction":          result.Reaction,
		"reaction_text":     result.ReactionText,
		"lesson":            result.Lesson,
		"note":              result.Note,
		"repeated":          result.Repeated,
	}
}

// handleGuide handles GET /api/guide - the care guide for the active pet's
// species, or ?species= another (see /api/species)
func (s *Server) handleGuide(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, APIResponse{Success: false, Error: "Method not allowed"})
		return
	}
	species, ok := core.ParseSpecies(r.URL.Query().Get("species"))
	if !ok {
		writeJSON(w, http.StatusBadRequest, APIResponse{Success: false, Error: "Unknown kind of pet"})
		return
	}
	if r.URL.Query().Get("species") == "" {
		s.mu.RLock()
		if s.pet != nil {
			species = s.pet.GetSpecies()
		}
		s.mu.RUnlock()
	}
	writeJSON(w, http.StatusOK, APIResponse{Success: true, Data: map[string]interface{}{
		"species": species,
		"topics":  core.CareGuide(species),
	}})
}

// speciesSummary is a species as the page sees it
type speciesSummary struct {
	ID         core.Species      `json:"id"`
	Name       string            `json:"name"`
	Plural     string            `json:"plural"`
	Icon       string            `json:"icon"`
	Sprites    map[string]string `json:"sprites"`
	CleanLabel string            `json:"clean_label"`
	Walks      bool              `json:"walks"`
}

// handleSpecies handles GET /api/species - the kinds of pet, the default
// first
func (s *Server) handleSpecies(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, APIResponse{Success: false, Error: "Method not allowed"})
		return
	}
	list := []speciesSummary{}
	for _, def := range core.SpeciesList() {
		list = append(list, speciesSummary{ID: def.ID, Name: def.Name, Plural: def.Plural, Icon: def.Icon,
			Sprites: def.Sprites, CleanLabel: def.CleanLabel, Walks: def.Walks})
	}
	writeJSON(w, http.StatusOK, APIResponse{Success: true, Data: list})
}

// writeActivityError reports why the pet cannot do an activity
func writeActivityError(w http.ResponseWriter, name string, err error) {
	status, message := http.StatusConflict, ""
	var refused core.RefusedError
	switch {
	case errors.As(err, &refused):
		message = refused.Reason
	case errors.Is(err, core.ErrSleeping):
		message = name + " is sleeping. Let it rest."
	case errors.Is(err, core.ErrHibernating):
		message = name + " is hibernating. Revive it first."
	case errors.Is(err, core.ErrPetNotAvailable):
		message = name + " has passed away."
	case errors.Is(err, core.ErrExploring):
		message = name + " is out exploring and will be back soon."
	case errors.Is(err, core.ErrTooTired):
		message = name + " is too tired. Let it rest or give it some food first."
	case errors.Is(err, core.ErrNeedsCare):
		message = name + " needs food, water and energy before an adventure."
	case errors.Is(err, core.ErrInjured):
		message = name + " is hurt. Heal it before it goes exploring again."
	case errors.Is(err, core.ErrNotExploring):
		message = name + " is already home."
	case errors.Is(err, core.ErrTrickNotKnown):
		message = name + " hasn't learned that trick yet."
	case errors.Is(err, core.ErrUnknownPlace):
		status, message = http.StatusBadRequest, "Unknown place"
	case errors.Is(err, core.ErrPlaceLocked):
		message = name + " hasn't discovered that place yet. Keep exploring!"
	case errors.Is(err, core.ErrTrickRusty):
		message = name + " is rusty at that trick. Practice it with Train first."
	case errors.Is(err, core.ErrNoGame):
		message = "Start a game of hide and seek first."
	case errors.Is(err, core.ErrUnknownTrick):
		status, message = http.StatusBadRequest, "Unknown trick"
	case errors.Is(err, core.ErrNoSupply):
		message = "There's none of that left in the pantry. Buy more in the shop."
	case errors.Is(err, core.ErrNoTreats):
		message = "Training with treats needs treats from the shop."
	case errors.Is(err, core.ErrNoClicker):
		message = "Clicker training needs a clicker from the shop."
	case errors.Is(err, core.ErrWrongSpecies):
		status, message = http.StatusBadRequest, "That isn't for "+name+"'s species"
	case errors.Is(err, core.ErrUnknownChoice):
		status, message = http.StatusBadRequest, "Unknown choice"
	case errors.Is(err, core.ErrInvalidSpot):
		status, message = http.StatusBadRequest, "Unknown hiding spot"
	default:
		status, message = http.StatusInternalServerError, "Something went wrong"
	}
	writeJSON(w, status, APIResponse{Success: false, Error: message})
}

// withActivePet runs fn with s.mu held and the player marked present, or
// reports that there is no pet. The request must be a POST.
func (s *Server) withActivePet(w http.ResponseWriter, r *http.Request, fn func()) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, APIResponse{Success: false, Error: "Method not allowed"})
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if s.pet == nil || !s.pet.IsAlive() {
		writeJSON(w, http.StatusBadRequest, APIResponse{Success: false, Error: s.unavailableMessageLocked()})
		return
	}
	s.markActiveLocked(time.Now())
	fn()
}

// unavailableMessageLocked explains why there is no pet to care for.
// Caller must hold s.mu.
func (s *Server) unavailableMessageLocked() string {
	if s.pet == nil {
		return "No pet found"
	}
	return s.pet.Name + " has passed away."
}

// trickData is the response data for training or performing a trick.
// Caller must hold s.mu.
func (s *Server) trickData(result core.TrickResult) map[string]interface{} {
	data := s.interactionData(result.InteractionResult)
	data["text"] = result.Text
	data["trick"] = result.Trick
	data["skill"] = result.Skill
	data["just_learned"] = result.JustLearned
	return data
}

// handleTrick handles POST /api/train and /api/perform with {"trick": id}
func (s *Server) handleTrick(perform bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Trick  string `json:"trick"`
			Method string `json:"method"` // Training method (training only); "" for none
		}
		if r.Method == http.MethodPost {
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				writeJSON(w, http.StatusBadRequest, APIResponse{Success: false, Error: "Invalid request"})
				return
			}
		}

		s.withActivePet(w, r, func() {
			train := func(trick string) (core.TrickResult, error) { return s.pet.TrainTrickWith(trick, req.Method) }
			if perform {
				train = s.pet.PerformTrick
			}
			result, err := train(req.Trick)
			if err != nil {
				writeActivityError(w, s.pet.Name, err)
				return
			}
			writeJSON(w, http.StatusOK, APIResponse{Success: true, Data: s.trickData(result)})
		})
	}
}

// handleHideSeekStart handles POST /api/hideseek/start - the pet hides
func (s *Server) handleHideSeekStart(w http.ResponseWriter, r *http.Request) {
	s.withActivePet(w, r, func() {
		if err := s.pet.StartHideAndSeek(); err != nil {
			writeActivityError(w, s.pet.Name, err)
			return
		}
		writeJSON(w, http.StatusOK, APIResponse{Success: true, Data: map[string]interface{}{
			"spots": core.HidingSpots(),
			"text":  s.pet.Name + " is hiding! Where could it be?",
		}})
	})
}

// handleHideSeekGuess handles POST /api/hideseek/guess with {"spot": index}
func (s *Server) handleHideSeekGuess(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Spot int `json:"spot"`
	}
	if r.Method == http.MethodPost {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, APIResponse{Success: false, Error: "Invalid request"})
			return
		}
	}

	s.withActivePet(w, r, func() {
		result, err := s.pet.GuessHidingSpot(req.Spot)
		if err != nil {
			writeActivityError(w, s.pet.Name, err)
			return
		}

		data := map[string]interface{}{}
		if result.Over {
			data = s.interactionData(result.InteractionResult)
		}
		data["found"] = result.Found
		data["over"] = result.Over
		data["guesses_left"] = result.GuessesLeft
		data["spot"] = result.Spot
		data["bonus_points"] = result.BonusPoints
		data["friendship_points"] = s.pet.FriendshipPoints
		data["text"] = result.Text
		writeJSON(w, http.StatusOK, APIResponse{Success: true, Data: data})
	})
}

// handleExplore handles POST /api/explore with an optional {"place": id}
// - sends the pet exploring (to the park if no place is given)
func (s *Server) handleExplore(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Place string `json:"place"`
	}
	if r.Method == http.MethodPost && r.ContentLength != 0 {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, APIResponse{Success: false, Error: "Invalid request"})
			return
		}
	}
	if req.Place == "" {
		req.Place = "park"
	}
	s.withActivePet(w, r, func() {
		if err := s.pet.SendExploringTo(req.Place); err != nil {
			writeActivityError(w, s.pet.Name, err)
			return
		}
		name := req.Place
		for _, place := range s.pet.Places() {
			if place.ID == req.Place {
				name = strings.ToLower(place.Name)
			}
		}
		writeJSON(w, http.StatusOK, APIResponse{Success: true, Data: map[string]interface{}{
			"text":              s.pet.Name + " sets off to explore the " + name + "!",
			"exploring_seconds": gameDaysToSeconds(s.pet.ExploringRemaining()),
		}})
	})
}

// handlePlaces handles GET /api/explore/places - where the active pet can
// explore, how much is left to find there and how dangerous it is
func (s *Server) handlePlaces(w http.ResponseWriter, r *http.Request) {
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
	writeJSON(w, http.StatusOK, APIResponse{Success: true, Data: s.pet.Places()})
}

// handleRecall handles POST /api/explore/recall - calls the pet home
// early; it brings nothing back
func (s *Server) handleRecall(w http.ResponseWriter, r *http.Request) {
	s.withActivePet(w, r, func() {
		text, err := s.pet.RecallFromExploring()
		if err != nil {
			writeActivityError(w, s.pet.Name, err)
			return
		}
		writeJSON(w, http.StatusOK, APIResponse{Success: true, Data: map[string]interface{}{"text": text}})
	})
}

// handleHideSeekQuit handles POST /api/hideseek/quit - ends a game early
// with no gains
func (s *Server) handleHideSeekQuit(w http.ResponseWriter, r *http.Request) {
	s.withActivePet(w, r, func() {
		text, err := s.pet.QuitHideAndSeek()
		if err != nil {
			writeActivityError(w, s.pet.Name, err)
			return
		}
		writeJSON(w, http.StatusOK, APIResponse{Success: true, Data: map[string]interface{}{"text": text}})
	})
}

// recentAdventureDays is how long (in game days) a finished trip is
// reported in the status, so the page can tell the player about it
const recentAdventureDays = 5.0

// gameDaysToSeconds converts game time to real seconds at the server's
// speed of one game day per real minute
func gameDaysToSeconds(days float64) int {
	return int(days*60 + 0.5)
}

// breedingPartner describes another saved pet as a possible mate
type breedingPartner struct {
	Slot          string  `json:"slot"`
	Name          string  `json:"name"`
	Compatibility float64 `json:"compatibility"` // 0.0 to 1.0
	Reason        string  `json:"reason"`        // Why they can't breed now, or ""
}

// handleBreedPartners handles GET /api/breed/partners - lists the other
// saved pets with their compatibility with the active pet
func (s *Server) handleBreedPartners(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, APIResponse{Success: false, Error: "Method not allowed"})
		return
	}
	if s.store == nil {
		writeJSON(w, http.StatusOK, APIResponse{Success: true, Data: []breedingPartner{}})
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
	partners := []breedingPartner{}
	for _, entry := range saved {
		if entry.Slot == s.slot {
			continue
		}
		partner, _, _, err := s.store.Load(entry.Slot)
		if err != nil {
			continue
		}
		compatibility, reason := core.BreedingCompatibility(s.pet, partner)
		partners = append(partners, breedingPartner{
			Slot: entry.Slot, Name: partner.Name, Compatibility: compatibility, Reason: reason,
		})
	}
	writeJSON(w, http.StatusOK, APIResponse{Success: true, Data: partners})
}

// handleBreed handles POST /api/breed with {"partner_slot": slot, "name":
// babyName}. A baby is saved in its own slot; the active pet stays active.
func (s *Server) handleBreed(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, APIResponse{Success: false, Error: "Method not allowed"})
		return
	}
	var req struct {
		PartnerSlot string `json:"partner_slot"`
		Name        string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || !ValidSlot(req.PartnerSlot) {
		writeJSON(w, http.StatusBadRequest, APIResponse{Success: false, Error: "Invalid partner"})
		return
	}
	if s.store == nil {
		writeJSON(w, http.StatusBadRequest, APIResponse{Success: false, Error: "Saving is disabled"})
		return
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		name = "Baby"
	}

	// Hold the save lock so the parents and baby are saved together
	s.saveMu.Lock()
	defer s.saveMu.Unlock()
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.pet == nil {
		writeJSON(w, http.StatusNotFound, APIResponse{Success: false, Error: "No pet found"})
		return
	}
	if req.PartnerSlot == s.slot {
		writeJSON(w, http.StatusBadRequest, APIResponse{Success: false, Error: s.pet.Name + " can't breed with itself"})
		return
	}
	s.markActiveLocked(time.Now())

	partner, _, _, err := s.store.Load(req.PartnerSlot)
	if err != nil {
		writeJSON(w, http.StatusNotFound, APIResponse{Success: false, Error: "No saved pet in that slot"})
		return
	}

	baby, err := core.Breed(s.pet, partner, name)
	var breedingErr core.BreedingError
	switch {
	case errors.As(err, &breedingErr):
		writeJSON(w, http.StatusConflict, APIResponse{Success: false,
			Error: s.pet.Name + " and " + partner.Name + " couldn't have a baby: " + breedingErr.Reason + "."})
		return
	case err != nil:
		writeJSON(w, http.StatusInternalServerError, APIResponse{Success: false, Error: "Something went wrong"})
		return
	}

	babySlot := s.store.NewSlot(baby)
	for _, save := range []struct {
		slot string
		pet  *core.DigitalPet
	}{{req.PartnerSlot, partner}, {babySlot, baby}} {
		if err := s.store.SavePet(save.slot, save.pet); err != nil {
			log.Printf("Failed to save %s after breeding: %v", save.pet.Name, err)
		}
	}
	if err := s.saveActiveLocked(); err != nil {
		log.Printf("Failed to save %s after breeding: %v", s.pet.Name, err)
	}
	s.observePet(baby, true)
	s.observePet(partner, false)

	log.Printf("%s and %s had a baby: %s", s.pet.Name, partner.Name, baby.Name)
	writeJSON(w, http.StatusOK, APIResponse{Success: true, Data: map[string]interface{}{
		"baby_name": baby.Name,
		"baby_slot": babySlot,
		"text":      s.pet.Name + " and " + partner.Name + " had a baby: " + baby.Name + "! Find " + baby.Name + " under Switch Pet.",
	}})
}

// handleEvents handles GET /api/events?after=seq - what happened to the
// active pet on its own (illness, hunger, trips ending) after the event
// numbered seq, oldest first, and the number to ask after next time
func (s *Server) handleEvents(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, APIResponse{Success: false, Error: "Method not allowed"})
		return
	}
	after, err := strconv.ParseInt(r.URL.Query().Get("after"), 10, 64)
	if err != nil && r.URL.Query().Get("after") != "" {
		writeJSON(w, http.StatusBadRequest, APIResponse{Success: false, Error: "Invalid event number"})
		return
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.pet == nil {
		writeJSON(w, http.StatusNotFound, APIResponse{Success: false, Error: "No pet found"})
		return
	}
	events, last := s.events.Since(after, s.pet.ID)
	writeJSON(w, http.StatusOK, APIResponse{Success: true, Data: map[string]interface{}{
		"events": events,
		"last":   last,
	}})
}

// handlePotion handles POST /api/potion - gives the active pet a
// life-extension potion from its pantry, and saves it
func (s *Server) handlePotion(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, APIResponse{Success: false, Error: "Method not allowed"})
		return
	}
	s.mu.Lock()
	if s.pet == nil || !s.pet.IsAlive() {
		message := s.unavailableMessageLocked()
		s.mu.Unlock()
		writeJSON(w, http.StatusBadRequest, APIResponse{Success: false, Error: message})
		return
	}
	s.markActiveLocked(time.Now())
	name := s.pet.Name
	text, err := s.pet.GiveLifePotion()
	s.mu.Unlock()

	switch {
	case errors.Is(err, core.ErrNoPotion):
		writeJSON(w, http.StatusConflict, APIResponse{Success: false,
			Error: "There's no life-extension potion in the pantry. Buy one in the shop."})
	case err != nil:
		writeActivityError(w, name, err)
	default:
		if _, err := s.saveNow(); err != nil {
			log.Printf("Failed to save after giving a potion: %v", err)
		}
		writeJSON(w, http.StatusOK, APIResponse{Success: true, Data: map[string]interface{}{"text": text}})
	}
}
