package main

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"sync"

	"github.com/Michael-W-Ellison/gochi/internal/core"
	"github.com/Michael-W-Ellison/gochi/internal/data"
)

// petSaveName is the save file name (without extension) for the server's pet
const petSaveName = "pet"

// petStore saves and restores the server's pet through a DataManager
type petStore struct {
	mu   sync.Mutex // Serializes saves (autosave, manual save, exit save)
	dm   *data.DataManager
	save *data.SaveData
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
	return &petStore{dm: dm}, nil
}

// Load restores the pet from the most recent valid save or backup.
// It returns the path loaded from. If no save exists, the error wraps
// os.ErrNotExist.
func (ps *petStore) Load() (*core.DigitalPet, string, error) {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	saveData, path, err := ps.dm.LoadLatest(petSaveName)
	if err != nil {
		return nil, "", err
	}

	pet, err := core.Load(saveData.PetData)
	if err != nil {
		return nil, "", fmt.Errorf("failed to restore pet from %s: %w", path, err)
	}

	ps.save = saveData
	return pet, path, nil
}

// Save writes serialized pet data (from DigitalPet.Save) to disk.
// The previous save is kept as a backup.
func (ps *petStore) Save(petData []byte) error {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	if ps.save == nil {
		ps.save = data.NewSaveData("web_user", "web_user")
	}

	// Compact the pet JSON before embedding it in the save file
	if err := ps.save.SetPetData(json.RawMessage(petData)); err != nil {
		return err
	}
	return ps.dm.Save(ps.save, petSaveName)
}

// SavePath returns where the pet is saved
func (ps *petStore) SavePath() string {
	return filepath.Join(ps.dm.SavePath, petSaveName+".json")
}

// LastWarning returns a non-fatal problem from the last load, such as a
// corrupt save that was skipped in favor of a backup
func (ps *petStore) LastWarning() error {
	return ps.dm.GetLastError()
}
