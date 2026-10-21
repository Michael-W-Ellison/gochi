package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
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
	// Preferences recorded in each save: the game speed and how often the
	// active pet is saved (seconds; 0 for off)
	timeScale       string
	autosaveSeconds int

	mu    sync.Mutex // Serializes saves and loads
	dm    *data.DataManager
	saves map[string]*data.SaveData // Save metadata per slot
	cache *data.Cache               // *cachedSave per slot
}

// saveCacheSize is how much pet data (in bytes) the store keeps in memory,
// so listing pets and loading partners for breeding and introductions
// doesn't read and check every save file each time
const saveCacheSize = 32 << 20

// cachedSave is a save file's contents as last read or written, with the
// file's modification time and size then, which tell whether it changed
type cachedSave struct {
	modTime  time.Time
	fileSize int64
	saveData *data.SaveData
	petData  []byte
	summary  savedPet
}

// maxSavedPets is how many pets the player can have saved at once
const maxSavedPets = 100

// errTooManyPets is returned when making a pet with maxSavedPets saved
var errTooManyPets = fmt.Errorf("you have %d pets, the most there can be; delete one to make room", maxSavedPets)

// Full reports whether as many pets as there can be are saved
func (ps *petStore) Full() bool {
	files, err := ps.dm.ListSaves()
	return err == nil && len(files) >= maxSavedPets
}

// savedPet summarizes a saved pet for the pet list
type savedPet struct {
	ID          string        `json:"-"`
	Slot        string        `json:"slot"`
	Name        string        `json:"name"`
	Species     string        `json:"species"` // cat or dog
	Age         float64       `json:"age"`
	IsAlive     bool          `json:"is_alive"`
	Hibernating bool          `json:"hibernating"`
	Paused      bool          `json:"paused,omitempty"` // Kept as it is instead of living on with the household
	SavedAt     time.Time     `json:"saved_at"`
	Unreadable  bool          `json:"unreadable,omitempty"` // Neither the save nor its backups can be read
	Outfit      string        `json:"outfit,omitempty"`     // Icons of what it wears
	Breed       string        `json:"breed,omitempty"`
	Look        *core.PetLook `json:"look,omitempty"` // nil for pets from before breeds
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
	return &petStore{dm: dm, saves: make(map[string]*data.SaveData), cache: data.NewCache(saveCacheSize, 0)}, nil
}

// cached returns the cached contents of slot's save file, if it hasn't
// changed since they were cached
func (ps *petStore) cached(slot string) (*cachedSave, bool) {
	value, ok := ps.cache.Get(slot)
	if !ok {
		return nil, false
	}
	entry := value.(*cachedSave)
	info, err := os.Stat(ps.SavePath(slot))
	if err != nil || !info.ModTime().Equal(entry.modTime) || info.Size() != entry.fileSize {
		ps.cache.Delete(slot)
		return nil, false
	}
	return entry, true
}

// cacheLocked caches the contents of slot's save file, just read or
// written. Caller must hold ps.mu.
func (ps *petStore) cacheLocked(slot string, saveData *data.SaveData) {
	info, err := os.Stat(ps.SavePath(slot))
	if err != nil {
		ps.cache.Delete(slot)
		return
	}
	petData := bytes.Clone(saveData.PetData)
	summary, err := summarize(slot, petData, saveData.LastSaveTime)
	if err != nil {
		ps.cache.Delete(slot)
		return
	}
	ps.cache.Set(slot, &cachedSave{modTime: info.ModTime(), fileSize: info.Size(), saveData: saveData,
		petData: petData, summary: summary}, int64(len(petData)))
}

// Load restores the pet in slot from its most recent valid save or backup.
// It returns the path loaded from and a warning if a newer save was
// skipped as unusable. If the slot has no save, the error wraps
// os.ErrNotExist.
func (ps *petStore) Load(slot string) (pet *core.DigitalPet, path string, warning error, err error) {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	if entry, ok := ps.cached(slot); ok {
		if pet, err := core.Load(entry.petData); err == nil {
			ps.saves[slot] = entry.saveData
			return pet, ps.SavePath(slot), nil, nil
		}
	}

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
	if path == ps.SavePath(slot) && warning == nil {
		ps.cacheLocked(slot, saveData) // Not a backup, which the next save replaces
	}
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
	return ps.SaveWithRecord(slot, petData, nil)
}

// SavePet saves a pet and its record to slot
func (ps *petStore) SavePet(slot string, pet *core.DigitalPet) error {
	petData, err := pet.Save()
	if err != nil {
		return err
	}
	record := pet.SaveRecord()
	return ps.SaveWithRecord(slot, petData, &record)
}

// SaveWithRecord writes serialized pet data to slot, with the pet's record
// (its history, achievements, unlocks and statistics) if record is not
// nil. The previous save of that slot is kept as a backup.
func (ps *petStore) SaveWithRecord(slot string, petData []byte, record *core.SaveRecord) error {
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
	if record != nil {
		applyRecord(saveData, *record)
	}
	saveData.Preferences.TimeScale, saveData.Preferences.AutoSaveInterval = ps.timeScale, ps.autosaveSeconds
	if err := ps.dm.Save(saveData, slot); err != nil {
		ps.cache.Delete(slot)
		return err
	}
	ps.cacheLocked(slot, saveData)
	return nil
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
		slot := strings.TrimSuffix(file.Filename, ".json")
		pet, err := ps.readSummary(slot, file.Path)
		if err != nil {
			// Listed so the player can see it and delete it; it is named
			// by its slot, which deleting it asks for
			pet = savedPet{Slot: slot, Name: slot, Species: string(core.SpeciesCat), SavedAt: file.ModifiedAt,
				Unreadable: true}
		}
		pets = append(pets, pet)
	}
	return pets, nil
}

// Summary returns the list entry for the pet saved in slot
func (ps *petStore) Summary(slot string) (savedPet, error) {
	return ps.readSummary(slot, ps.SavePath(slot))
}

// readSummary returns the list entry for the pet saved at path, from the
// cache if the file hasn't changed
func (ps *petStore) readSummary(slot, path string) (savedPet, error) {
	if path == ps.SavePath(slot) {
		if entry, ok := ps.cached(slot); ok {
			return entry.summary, nil
		}
	}
	contents, err := os.ReadFile(path)
	if err != nil {
		return savedPet{}, err
	}
	saveData, err := data.FromJSON(contents)
	if err != nil {
		return savedPet{}, err
	}
	return summarize(slot, saveData.PetData, saveData.LastSaveTime)
}

// summarize reads only the fields the pet list shows from a saved pet's
// data, not the whole pet
func summarize(slot string, petData []byte, savedAt time.Time) (savedPet, error) {
	var summary struct {
		ID          string            `json:"id"`
		Name        string            `json:"name"`
		Species     string            `json:"species"`
		Hibernating bool              `json:"hibernating"`
		Paused      bool              `json:"household_paused"`
		Outfit      map[string]string `json:"outfit"`
		Breed       string            `json:"breed"`
		Look        *core.PetLook     `json:"look"`
		Biology     struct {
			IsAlive   bool
			Processes struct {
				Age float64 `json:"age"`
			}
		} `json:"biology"`
	}
	if err := json.Unmarshal(petData, &summary); err != nil {
		return savedPet{}, err
	}

	species, ok := core.ParseSpecies(summary.Species)
	if !ok {
		species = core.SpeciesCat
	}
	return savedPet{
		ID:          summary.ID,
		Slot:        slot,
		Name:        summary.Name,
		Species:     string(species),
		Age:         summary.Biology.Processes.Age,
		IsAlive:     summary.Biology.IsAlive,
		Hibernating: summary.Hibernating,
		Paused:      summary.Paused,
		SavedAt:     savedAt,
		Outfit:      outfitIcons(summary.Outfit),
		Breed:       summary.Breed,
		Look:        summary.Look,
	}, nil
}

// Delete permanently removes the pet in slot and all of its backups.
// If the slot has no save, the error wraps os.ErrNotExist.
func (ps *petStore) Delete(slot string) error {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	ps.cache.Delete(slot)
	if err := ps.dm.DeleteWithBackups(slot); err != nil {
		return err
	}
	delete(ps.saves, slot)
	return nil
}

// NewSlot returns an unused slot name for a pet, derived from its ID
func (ps *petStore) NewSlot(pet *core.DigitalPet) string {
	return ps.NewSlotFor(string(pet.ID))
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

// applyRecord copies a pet's record into its save file
func applyRecord(saveData *data.SaveData, record core.SaveRecord) {
	saveData.CareHistory = make([]data.HistoryEntry, len(record.History))
	for i, h := range record.History {
		saveData.CareHistory[i] = data.HistoryEntry{Timestamp: h.Time, ActionType: h.Action, Description: h.Description,
			PetAge: h.Age, Outcome: h.Outcome}
	}
	saveData.Achievements = make([]data.Achievement, len(record.Achievements))
	for i, a := range record.Achievements {
		saveData.Achievements[i] = data.Achievement{ID: a.ID, Name: a.Name, Description: a.Description,
			UnlockedAt: a.EarnedAt, Rarity: a.Rarity, Points: a.Points}
	}
	u := record.Unlocks
	saveData.Unlocks = data.UnlockableContent{Locations: u.Places, Accessories: u.Items, Foods: u.Foods, Toys: u.Toys,
		Backgrounds: saveData.Unlocks.Backgrounds, PetVariants: u.Variants, Tricks: u.Tricks}
	stats := &saveData.Statistics
	stats.TotalPlayTime = record.Stats.PlayTime
	stats.TotalInteractions = record.Stats.Interactions
	stats.LongestLifespan = max(stats.LongestLifespan, record.Stats.Age)
	stats.HighestHappiness = max(stats.HighestHappiness, record.Stats.Happiness)
	stats.AchievementsEarned = record.Stats.Achievements
	if !record.Stats.Alive && stats.PetsLost == 0 {
		stats.PetsLost = 1
	}
}

// exportedSave is a save file as it is uploaded to the cloud hub
type exportedSave struct {
	JSON       []byte    // The save file's contents (data.SaveData)
	PetHash    string    // Changes when the pet itself changes (not when it is merely saved again)
	LastPlayed time.Time // When time last passed for the pet
}

// petFingerprint returns a hash of a pet's data and when it was last
// played
func petFingerprint(petData []byte) (string, time.Time) {
	sum := sha256.Sum256(petData)
	var pet struct {
		LastUpdateAt time.Time `json:"last_update_at"`
	}
	json.Unmarshal(petData, &pet)
	return hex.EncodeToString(sum[:]), pet.LastUpdateAt
}

// Export returns the save in slot, to upload to the cloud hub
func (ps *petStore) Export(slot string) (exportedSave, error) {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	var saveData *data.SaveData
	if entry, ok := ps.cached(slot); ok {
		saveData = entry.saveData
	} else {
		loaded, path, err := ps.dm.LoadLatest(slot)
		if err != nil {
			return exportedSave{}, err
		}
		if path != ps.SavePath(slot) {
			return exportedSave{}, fmt.Errorf("the save of %s is damaged; load it to restore it from a backup first", slot)
		}
		saveData = loaded
		ps.cacheLocked(slot, saveData)
	}
	contents, err := saveData.ToJSON()
	if err != nil {
		return exportedSave{}, err
	}
	hash, played := petFingerprint(saveData.PetData)
	return exportedSave{JSON: contents, PetHash: hash, LastPlayed: played}, nil
}

// Import replaces the save in slot with one downloaded from the cloud hub
// (the previous save is kept as a backup) and returns the pet's hash (see
// Export)
func (ps *petStore) Import(slot string, contents []byte) (string, error) {
	saveData, err := data.FromJSON(contents)
	if err != nil {
		return "", fmt.Errorf("the downloaded save can't be read: %w", err)
	}
	if _, err := core.Load(saveData.PetData); err != nil {
		return "", fmt.Errorf("the downloaded pet can't be read: %w", err)
	}

	ps.mu.Lock()
	defer ps.mu.Unlock()
	if err := ps.dm.Save(saveData, slot); err != nil {
		ps.cache.Delete(slot)
		return "", err
	}
	ps.saves[slot] = saveData
	ps.cacheLocked(slot, saveData)
	hash, _ := petFingerprint(saveData.PetData)
	return hash, nil
}

// NewSlotFor returns an unused slot for a pet with the given ID and name
func (ps *petStore) NewSlotFor(id string) string {
	base := sanitizeSlot(id)
	slot := base
	for i := 2; ps.dm.Exists(slot); i++ {
		slot = fmt.Sprintf("%s-%d", base, i)
	}
	return slot
}

// outfitIcons returns the icons of what a pet wears, head to body
func outfitIcons(outfit map[string]string) string {
	icons := ""
	for _, slot := range core.CosmeticSlots {
		if c, ok := core.FindCosmetic(outfit[slot]); ok && c.Slot == slot && (c.Holiday == "" || core.HolidayOn(c.Holiday)) {
			icons += c.Icon
		}
	}
	return icons
}
