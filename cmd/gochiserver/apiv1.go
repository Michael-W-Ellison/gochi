package main

import (
	"crypto/subtle"
	"errors"
	"net/http"
	"os"
	"time"

	"github.com/Michael-W-Ellison/gochi/api"
	"github.com/Michael-W-Ellison/gochi/internal/core"
	"github.com/Michael-W-Ellison/gochi/pkg/types"
)

// localOwner owns every pet created by this server
const localOwner types.UserID = "web_user"

// errOffline is returned for API features that need a cloud service
var errOffline = errors.New("this feature needs the cloud service, which this server does not have")

// apiHandler returns the api package's REST API (/api/v1, /health and
// /ready), backed by this server's saved pets. Requests to pet routes
// must carry "Authorization: Bearer <token>". Each client may make
// rateLimit requests a minute (0 for no limit).
func (s *Server) apiHandler(token string, rateLimit int) http.Handler {
	handler := api.NewHandler(apiPetStore{s})
	config := api.DefaultServerConfig()
	config.RateLimit = rateLimit
	return api.NewServer(config, handler, tokenValidator(token)).SetupRoutes()
}

// tokenValidator accepts a single token, which identifies the local player
type tokenValidator string

func (t tokenValidator) ValidateToken(token string) (types.UserID, error) {
	if t == "" || subtle.ConstantTimeCompare([]byte(token), []byte(t)) != 1 {
		return "", errors.New("invalid token")
	}
	return localOwner, nil
}

// apiPetStore gives the API handlers copies of the saved pets, so they
// never share a pet with the game loop. Updating the active pet replaces
// it; any change the game made to it since it was read (at most a few
// seconds of time passing) is replaced too.
type apiPetStore struct{ s *Server }

// copyPet returns an independent copy of a pet
func copyPet(pet *core.DigitalPet) (*core.DigitalPet, error) {
	data, err := pet.Save()
	if err != nil {
		return nil, err
	}
	return core.Load(data)
}

// findLocked returns a copy of the pet with the given ID and its slot.
// Caller must hold s.mu.
func (a apiPetStore) findLocked(id types.PetID) (*core.DigitalPet, string, error) {
	s := a.s
	if s.pet != nil && s.pet.ID == id {
		pet, err := copyPet(s.pet)
		return pet, s.slot, err
	}
	if s.store == nil {
		return nil, "", os.ErrNotExist
	}
	saved, err := s.store.List()
	if err != nil {
		return nil, "", err
	}
	for _, entry := range saved {
		if entry.Slot == s.slot {
			continue
		}
		pet, _, _, err := s.store.Load(entry.Slot)
		if err == nil && pet.ID == id {
			return pet, entry.Slot, nil
		}
	}
	return nil, "", os.ErrNotExist
}

func (a apiPetStore) GetPet(id types.PetID) (*core.DigitalPet, error) {
	a.s.mu.RLock()
	defer a.s.mu.RUnlock()
	pet, _, err := a.findLocked(id)
	return pet, err
}

func (a apiPetStore) GetPetsByOwner(owner types.UserID) ([]*core.DigitalPet, error) {
	s := a.s
	s.mu.RLock()
	defer s.mu.RUnlock()

	var pets []*core.DigitalPet
	if s.pet != nil {
		pet, err := copyPet(s.pet)
		if err != nil {
			return nil, err
		}
		pets = append(pets, pet)
	}
	if s.store != nil {
		saved, err := s.store.List()
		if err != nil {
			return nil, err
		}
		for _, entry := range saved {
			if entry.Slot == s.slot {
				continue
			}
			if pet, _, _, err := s.store.Load(entry.Slot); err == nil {
				pets = append(pets, pet)
			}
		}
	}

	owned := pets[:0]
	for _, pet := range pets {
		if pet.Owner == owner {
			owned = append(owned, pet)
		}
	}
	return owned, nil
}

// CreatePet saves a new pet in its own slot; the active pet stays active
func (a apiPetStore) CreatePet(pet *core.DigitalPet) error {
	s := a.s
	if s.store == nil {
		return errors.New("saving is disabled")
	}
	s.saveMu.Lock()
	defer s.saveMu.Unlock()

	return s.store.SavePet(s.store.NewSlot(pet), pet)
}

func (a apiPetStore) UpdatePet(pet *core.DigitalPet) error {
	s := a.s
	s.saveMu.Lock()
	defer s.saveMu.Unlock()
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.pet != nil && s.pet.ID == pet.ID {
		s.pet = pet
		s.markActiveLocked(time.Now())
		return s.saveActiveLocked()
	}

	_, slot, err := a.findLocked(pet.ID)
	if err != nil {
		return err
	}
	return s.store.SavePet(slot, pet)
}

// DeletePet is not allowed: deleting a pet needs the confirmations the
// game asks for
func (a apiPetStore) DeletePet(types.PetID) error {
	return api.ErrNotPermitted
}
