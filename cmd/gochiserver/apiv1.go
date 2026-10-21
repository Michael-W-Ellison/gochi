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
	return a.findSaved(id, s.slot)
}

// findSaved loads the saved pet with the given ID, other than the one in
// skip (the active pet's slot)
func (a apiPetStore) findSaved(id types.PetID, skip string) (*core.DigitalPet, string, error) {
	s := a.s
	if s.store == nil {
		return nil, "", os.ErrNotExist
	}
	saved, err := s.store.List()
	if err != nil {
		return nil, "", err
	}
	for _, entry := range saved {
		if entry.Slot != skip && entry.ID == string(id) {
			pet, _, _, err := s.store.Load(entry.Slot)
			return pet, entry.Slot, err
		}
	}
	return nil, "", os.ErrNotExist
}

// activeCopy returns a copy of the active pet (nil if there is none) and
// its slot
func (a apiPetStore) activeCopy() (*core.DigitalPet, string, error) {
	s := a.s
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.pet == nil {
		return nil, "", nil
	}
	pet, err := copyPet(s.pet)
	return pet, s.slot, err
}

// GetPet returns a copy of a pet. Saved pets are read without holding up
// the game.
func (a apiPetStore) GetPet(id types.PetID) (*core.DigitalPet, error) {
	active, slot, err := a.activeCopy()
	if err != nil {
		return nil, err
	}
	if active != nil && active.ID == id {
		return active, nil
	}
	pet, _, err := a.findSaved(id, slot)
	return pet, err
}

func (a apiPetStore) GetPetsByOwner(owner types.UserID) ([]*core.DigitalPet, error) {
	s := a.s
	active, slot, err := a.activeCopy()
	if err != nil {
		return nil, err
	}
	var pets []*core.DigitalPet
	if active != nil {
		pets = append(pets, active)
	}
	if s.store != nil {
		saved, err := s.store.List()
		if err != nil {
			return nil, err
		}
		for _, entry := range saved {
			if entry.Slot == slot || entry.Unreadable {
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

	if s.store.Full() {
		return api.ErrTooManyPets
	}
	if err := s.store.SavePet(s.store.NewSlot(pet), pet); err != nil {
		return err
	}
	s.observePet(pet, false)
	return nil
}

// ModifyPet changes a pet and saves it: the active pet itself, under the
// game's lock, so nothing the game does meanwhile is lost, or a saved pet.
// Renaming needs the parent PIN, which the API can't give, so it is
// refused while a PIN is set.
func (a apiPetStore) ModifyPet(id types.PetID, change func(*core.DigitalPet) error) (*core.DigitalPet, error) {
	s := a.s
	s.saveMu.Lock()
	defer s.saveMu.Unlock()
	s.mu.Lock()
	defer s.mu.Unlock()

	pet, slot, active := s.pet, s.slot, s.pet != nil && s.pet.ID == id
	if !active {
		var err error
		if pet, slot, err = a.findLocked(id); errors.Is(err, os.ErrNotExist) {
			return nil, api.ErrNotFound
		} else if err != nil {
			return nil, err
		}
	}
	name, location := pet.Name, pet.Location
	if err := change(pet); err != nil {
		return nil, err
	}
	if pet.Name != name && s.parental != nil && s.parental.Locked() {
		pet.Name, pet.Location = name, location
		return nil, api.ErrNotPermitted
	}

	if active {
		s.markActiveLocked(time.Now())
		if err := s.saveActiveLocked(); err != nil {
			return nil, err
		}
		return copyPet(pet)
	}
	if err := s.store.SavePet(slot, pet); err != nil {
		return nil, err
	}
	s.observePet(pet, false)
	return pet, nil
}

// DeletePet is not allowed: deleting a pet needs the confirmations the
// game asks for
func (a apiPetStore) DeletePet(types.PetID) error {
	return api.ErrNotPermitted
}
