package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/Michael-W-Ellison/gochi/internal/core"
	"github.com/Michael-W-Ellison/gochi/internal/data"
)

// petStore saves and restores pets through a DataManager. Each pet has its
// own save slot: a save file named <slot>.json with its own backups.
type petStore struct {
	mu    sync.Mutex // Serializes saves and loads
	dm    *data.DataManager
	saves map[string]*data.SaveData // Save metadata per slot
}

// savedPet summarizes a saved pet for the pet list
type savedPet struct {
	Slot        string    `json:"slot"`
	Name        string    `json:"name"`
	Age         float64   `json:"age"`
	IsAlive     bool      `json:"is_alive"`
	Hibernating bool      `json:"hibernating"`
	SavedAt     time.Time `json:"saved_at"`
}

// newPetStore creates a store that keeps saves and backups under dir
func newPetStore(dir string) (*petStore, error) {
	config := data.DefaultDataManagerConfig()
	if dir != "" {
		config.SavePath = filepath.Join(dir, "saves")
		config.BackupPath = filepath.Join(dir, "backups")
		config.CachePath = filepath.Join(dir, "cache")
	}

	dm, err := data.NewDataManagerWithConfig(config)
	if err != nil {
		return nil, err
	}
	return &petStore{dm: dm, saves: make(map[string]*data.SaveData)}, nil
}

// Load restores the pet in slot from its most recent valid save or backup.
// It returns the path loaded from and a warning if a newer save was
// skipped as unusable. If the slot has no save, the error wraps
// os.ErrNotExist.
func (ps *petStore) Load(slot string) (pet *core.DigitalPet, path string, warning error, err error) {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	saveData, path, err := ps.dm.LoadLatest(slot)
	if err != nil {
		return nil, "", nil, err
	}
	warning = ps.dm.GetLastError()

	pet, err = core.Load(saveData.PetData)
	if err != nil {
		return nil, "", nil, fmt.Errorf("failed to restore pet from %s: %w", path, err)
	}

	ps.saves[slot] = saveData
	return pet, path, warning, nil
}

// LoadMostRecent restores the most recently saved pet that can be loaded.
// If there are no saves, the error wraps os.ErrNotExist.
func (ps *petStore) LoadMostRecent() (pet *core.DigitalPet, slot, path string, warning error, err error) {
	files, err := ps.dm.ListSaves() // Newest first
	if err != nil {
		return nil, "", "", nil, err
	}

	var failures []string
	for _, file := range files {
		slot := strings.TrimSuffix(file.Filename, ".json")
		pet, path, warning, err := ps.Load(slot)
		if err != nil {
			failures = append(failures, fmt.Sprintf("%s: %v", slot, err))
			continue
		}
		if len(failures) > 0 {
			warning = errors.Join(warning, fmt.Errorf("skipped unusable saves: %s", strings.Join(failures, "; ")))
		}
		return pet, slot, path, warning, nil
	}

	if len(failures) == 0 {
		return nil, "", "", nil, fmt.Errorf("no saved pets: %w", os.ErrNotExist)
	}
	return nil, "", "", nil, fmt.Errorf("no usable saves: %s", strings.Join(failures, "; "))
}

// Save writes serialized pet data (from DigitalPet.Save) to slot.
// The previous save of that slot is kept as a backup.
func (ps *petStore) Save(slot string, petData []byte) error {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	saveData, ok := ps.saves[slot]
	if !ok {
		saveData = data.NewSaveData("web_user", "web_user")
		ps.saves[slot] = saveData
	}

	// Compact the pet JSON before embedding it in the save file
	if err := saveData.SetPetData(json.RawMessage(petData)); err != nil {
		return err
	}
	return ps.dm.Save(saveData, slot)
}

// List returns all saved pets, most recently saved first. Saves that
// cannot be read are left out.
func (ps *petStore) List() ([]savedPet, error) {
	files, err := ps.dm.ListSaves() // Newest first
	if err != nil {
		return nil, err
	}

	pets := make([]savedPet, 0, len(files))
	for _, file := range files {
		pet, err := readSummary(strings.TrimSuffix(file.Filename, ".json"), file.Path)
		if err != nil {
			continue
		}
		pets = append(pets, pet)
	}
	return pets, nil
}

// Summary returns the list entry for the pet saved in slot
func (ps *petStore) Summary(slot string) (savedPet, error) {
	return readSummary(slot, ps.SavePath(slot))
}

// readSummary reads only the fields the pet list shows, not the whole pet
func readSummary(slot, path string) (savedPet, error) {
	contents, err := os.ReadFile(path)
	if err != nil {
		return savedPet{}, err
	}
	saveData, err := data.FromJSON(contents)
	if err != nil {
		return savedPet{}, err
	}

	var summary struct {
		Name        string `json:"name"`
		Hibernating bool   `json:"hibernating"`
		Biology     struct {
			IsAlive   bool
			Processes struct {
				Age float64 `json:"age"`
			}
		} `json:"biology"`
	}
	if err := json.Unmarshal(saveData.PetData, &summary); err != nil {
		return savedPet{}, err
	}

	return savedPet{
		Slot:        slot,
		Name:        summary.Name,
		Age:         summary.Biology.Processes.Age,
		IsAlive:     summary.Biology.IsAlive,
		Hibernating: summary.Hibernating,
		SavedAt:     saveData.LastSaveTime,
	}, nil
}

// Delete permanently removes the pet in slot and all of its backups.
// If the slot has no save, the error wraps os.ErrNotExist.
func (ps *petStore) Delete(slot string) error {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	if err := ps.dm.DeleteWithBackups(slot); err != nil {
		return err
	}
	delete(ps.saves, slot)
	return nil
}

// NewSlot returns an unused slot name for a pet, derived from its ID
func (ps *petStore) NewSlot(pet *core.DigitalPet) string {
	base := sanitizeSlot(string(pet.ID))
	slot := base
	for i := 2; ps.dm.Exists(slot); i++ {
		slot = fmt.Sprintf("%s-%d", base, i)
	}
	return slot
}

// ValidSlot reports whether slot is a well-formed slot name
func ValidSlot(slot string) bool {
	return slot != "" && slot == sanitizeSlot(slot)
}

// sanitizeSlot keeps letters, digits, '_' and '-' so a slot is always a
// plain file name
func sanitizeSlot(s string) string {
	var b strings.Builder
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '_', r == '-':
			b.WriteRune(r)
		default:
			b.WriteRune('-')
		}
	}
	if b.Len() == 0 {
		return "pet"
	}
	return b.String()
}

// SaveDir returns the directory pets are saved in
func (ps *petStore) SaveDir() string {
	return ps.dm.SavePath
}

// SavePath returns where the pet in slot is saved
func (ps *petStore) SavePath(slot string) string {
	return filepath.Join(ps.dm.SavePath, slot+".json")
}
