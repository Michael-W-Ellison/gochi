// Package hub is the cloud service: a Gochi server can run as a hub that
// other Gochi servers connect to. Players register with the hub's join
// code and get an account token. The hub keeps a copy of each player's
// pet saves, so they can be backed up and synced between devices; it
// shows players whose pets are in the same area (nearby pets), when they
// choose to share; and it keeps a feed of achievements players share.
//
// The hub is self-hosted and stores everything as files in its directory.
// Tokens are stored only as hashes.
package hub

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"
)

// Limits
const (
	MaxPetsPerAccount = 50
	MaxSaveBytes      = 8 << 20 // A pet's save file
	maxFeed           = 200     // Achievements kept in the feed
	maxAreaLength     = 60
)

// Errors
var (
	ErrRegistrationClosed = errors.New("this hub has no join code, so it doesn't accept new players")
	ErrWrongJoinCode      = errors.New("wrong join code")
	ErrBadUsername        = errors.New("usernames are 2 to 24 letters, digits, '-' or '_'")
	ErrUsernameTaken      = errors.New("that username is taken")
	ErrUnauthorized       = errors.New("unknown account token")
	ErrBadKey             = errors.New("invalid pet key")
	ErrNotFound           = errors.New("no such pet")
	ErrVersionConflict    = errors.New("the hub has a different version of this pet")
	ErrTooManyPets        = errors.New("too many pets for one account")
)

var (
	usernamePattern = regexp.MustCompile(`^[A-Za-z0-9_-]{2,24}$`)
	keyPattern      = regexp.MustCompile(`^[a-f0-9]{32}$`)
)

// PetKey returns the hub's key for a pet: a hash of its ID, so pet names
// don't appear in file names
func PetKey(petID string) string {
	sum := sha256.Sum256([]byte(petID))
	return hex.EncodeToString(sum[:16])
}

// PetInfo describes a pet stored on the hub
type PetInfo struct {
	Key     string    `json:"key"`
	Version int       `json:"version"`
	SavedAt time.Time `json:"saved_at"` // When the pet was last played (time last passed for it)
	Deleted bool      `json:"deleted"`  // Deleted by the player; its save is gone
	Name    string    `json:"name"`
	Species string    `json:"species"`
	Stage   string    `json:"stage"`
	Alive   bool      `json:"alive"`
}

// storedPet is a pet's file on the hub
type storedPet struct {
	PetInfo
	Save json.RawMessage `json:"save,omitempty"`
}

// Profile is a player's social settings
type Profile struct {
	Username string `json:"username"`
	Area     string `json:"area"`    // Town or area, typed by the player
	Sharing  bool   `json:"sharing"` // Pets appear to nearby players and achievements can be shared
}

// account is a registered player
type account struct {
	Profile
	TokenHash string    `json:"token_hash"`
	Created   time.Time `json:"created"`
}

// NearbyPet is another player's pet in the same area
type NearbyPet struct {
	Owner   string    `json:"owner"`
	Name    string    `json:"name"`
	Species string    `json:"species"`
	Stage   string    `json:"stage"`
	SavedAt time.Time `json:"saved_at"`
}

// SharedAchievement is an achievement a player shared
type SharedAchievement struct {
	Owner       string    `json:"owner"`
	PetName     string    `json:"pet_name"`
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Icon        string    `json:"icon"`
	Description string    `json:"description"`
	SharedAt    time.Time `json:"shared_at"`
}

// Hub is the hub's state
type Hub struct {
	dir      string
	joinCode string
	now      func() time.Time

	mu       sync.Mutex
	accounts map[string]*account // By lowercase username
	byToken  map[string]string   // Token hash -> lowercase username
	feed     []SharedAchievement
}

// New opens (or creates) a hub keeping its files in dir. Players need
// joinCode to register; with an empty join code registration is closed.
func New(dir, joinCode string) (*Hub, error) {
	if err := os.MkdirAll(filepath.Join(dir, "pets"), 0o700); err != nil {
		return nil, err
	}
	h := &Hub{dir: dir, joinCode: joinCode, now: time.Now,
		accounts: make(map[string]*account), byToken: make(map[string]string)}

	var accounts []*account
	if err := readJSON(filepath.Join(dir, "accounts.json"), &accounts); err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	for _, a := range accounts {
		h.accounts[strings.ToLower(a.Username)] = a
		h.byToken[a.TokenHash] = strings.ToLower(a.Username)
	}
	if err := readJSON(filepath.Join(dir, "feed.json"), &h.feed); err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	return h, nil
}

// RegistrationOpen reports whether new players can register
func (h *Hub) RegistrationOpen() bool {
	return h.joinCode != ""
}

// hashToken returns the stored form of a token
func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

// Register creates an account and returns its token
func (h *Hub) Register(username, joinCode string) (string, error) {
	if h.joinCode == "" {
		return "", ErrRegistrationClosed
	}
	if subtle.ConstantTimeCompare([]byte(joinCode), []byte(h.joinCode)) != 1 {
		return "", ErrWrongJoinCode
	}
	if !usernamePattern.MatchString(username) {
		return "", ErrBadUsername
	}

	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	token := hex.EncodeToString(raw)

	h.mu.Lock()
	defer h.mu.Unlock()
	lower := strings.ToLower(username)
	if _, taken := h.accounts[lower]; taken {
		return "", ErrUsernameTaken
	}
	a := &account{Profile: Profile{Username: username}, TokenHash: hashToken(token), Created: h.now()}
	h.accounts[lower] = a
	h.byToken[a.TokenHash] = lower
	if err := h.saveAccountsLocked(); err != nil {
		delete(h.accounts, lower)
		delete(h.byToken, a.TokenHash)
		return "", err
	}
	if err := os.MkdirAll(h.petDir(lower), 0o700); err != nil {
		return "", err
	}
	return token, nil
}

// Authenticate returns the username a token belongs to
func (h *Hub) Authenticate(token string) (string, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if lower, ok := h.byToken[hashToken(token)]; ok && token != "" {
		return h.accounts[lower].Username, nil
	}
	return "", ErrUnauthorized
}

// petDir is where an account's pets are stored
func (h *Hub) petDir(lower string) string {
	return filepath.Join(h.dir, "pets", lower)
}

// ListPets returns the player's pets on the hub
func (h *Hub) ListPets(username string) ([]PetInfo, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.listPetsLocked(strings.ToLower(username))
}

func (h *Hub) listPetsLocked(lower string) ([]PetInfo, error) {
	entries, err := os.ReadDir(h.petDir(lower))
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	pets := []PetInfo{}
	for _, e := range entries {
		key, ok := strings.CutSuffix(e.Name(), ".json")
		if !ok || !keyPattern.MatchString(key) {
			continue
		}
		var pet storedPet
		if err := readJSON(filepath.Join(h.petDir(lower), e.Name()), &pet); err != nil {
			continue
		}
		pets = append(pets, pet.PetInfo)
	}
	sort.Slice(pets, func(i, j int) bool { return pets[i].Key < pets[j].Key })
	return pets, nil
}

// GetPet returns a pet's information and save
func (h *Hub) GetPet(username, key string) (PetInfo, json.RawMessage, error) {
	if !keyPattern.MatchString(key) {
		return PetInfo{}, nil, ErrBadKey
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	pet, err := h.readPetLocked(strings.ToLower(username), key)
	return pet.PetInfo, pet.Save, err
}

func (h *Hub) readPetLocked(lower, key string) (storedPet, error) {
	var pet storedPet
	err := readJSON(filepath.Join(h.petDir(lower), key+".json"), &pet)
	if errors.Is(err, os.ErrNotExist) {
		return pet, ErrNotFound
	}
	return pet, err
}

// PutPet stores a new version of a pet. baseVersion is the version the
// player's copy was based on (0 for a pet new to the hub); if the hub has
// another version, nothing changes and ErrVersionConflict is returned with
// the hub's version. The previous version is kept as a backup.
func (h *Hub) PutPet(username string, info PetInfo, save json.RawMessage, baseVersion int) (int, error) {
	if !keyPattern.MatchString(info.Key) {
		return 0, ErrBadKey
	}
	if len(save) == 0 || len(save) > MaxSaveBytes || !json.Valid(save) {
		return 0, fmt.Errorf("the save must be valid JSON of at most %d bytes", MaxSaveBytes)
	}

	h.mu.Lock()
	defer h.mu.Unlock()
	lower := strings.ToLower(username)
	current, err := h.readPetLocked(lower, info.Key)
	switch {
	case errors.Is(err, ErrNotFound):
		pets, err := h.listPetsLocked(lower)
		if err != nil {
			return 0, err
		}
		if len(pets) >= MaxPetsPerAccount {
			return 0, ErrTooManyPets
		}
	case err != nil:
		return 0, err
	}
	if current.Version != baseVersion {
		return current.Version, ErrVersionConflict
	}

	path := filepath.Join(h.petDir(lower), info.Key+".json")
	if current.Version > 0 {
		if err := copyFile(path, filepath.Join(h.petDir(lower), info.Key+".prev")); err != nil {
			return 0, err
		}
	}
	info.Version = current.Version + 1
	info.Deleted = false
	if err := writeJSON(path, storedPet{PetInfo: info, Save: save}); err != nil {
		return 0, err
	}
	return info.Version, nil
}

// DeletePet replaces a pet's save with a record that it was deleted, so
// the player's other devices know not to upload it again
func (h *Hub) DeletePet(username, key string, baseVersion int) (int, error) {
	if !keyPattern.MatchString(key) {
		return 0, ErrBadKey
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	lower := strings.ToLower(username)
	current, err := h.readPetLocked(lower, key)
	if err != nil {
		return 0, err
	}
	if current.Version != baseVersion {
		return current.Version, ErrVersionConflict
	}
	info := current.PetInfo
	info.Version++
	info.Deleted = true
	os.Remove(filepath.Join(h.petDir(lower), key+".prev"))
	return info.Version, writeJSON(filepath.Join(h.petDir(lower), key+".json"), storedPet{PetInfo: info})
}

// normalizeArea makes area names comparable: trimmed, single-spaced and
// lowercase
func normalizeArea(area string) string {
	return strings.ToLower(strings.Join(strings.Fields(area), " "))
}

// GetProfile returns a player's social settings
func (h *Hub) GetProfile(username string) (Profile, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	a, ok := h.accounts[strings.ToLower(username)]
	if !ok {
		return Profile{}, ErrUnauthorized
	}
	return a.Profile, nil
}

// SetProfile changes a player's area and whether they share
func (h *Hub) SetProfile(username, area string, sharing bool) (Profile, error) {
	area = strings.Join(strings.Fields(area), " ")
	if len([]rune(area)) > maxAreaLength {
		return Profile{}, fmt.Errorf("the area name can be at most %d characters", maxAreaLength)
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	a, ok := h.accounts[strings.ToLower(username)]
	if !ok {
		return Profile{}, ErrUnauthorized
	}
	before := a.Profile
	a.Area, a.Sharing = area, sharing
	if err := h.saveAccountsLocked(); err != nil {
		a.Profile = before
		return Profile{}, err
	}
	return a.Profile, nil
}

// Nearby returns living pets of other players in the player's area, for
// players who share. A player who doesn't share sees no one.
func (h *Hub) Nearby(username string) ([]NearbyPet, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	me, ok := h.accounts[strings.ToLower(username)]
	if !ok {
		return nil, ErrUnauthorized
	}
	nearby := []NearbyPet{}
	area := normalizeArea(me.Area)
	if !me.Sharing || area == "" {
		return nearby, nil
	}
	for lower, other := range h.accounts {
		if other == me || !other.Sharing || normalizeArea(other.Area) != area {
			continue
		}
		pets, err := h.listPetsLocked(lower)
		if err != nil {
			return nil, err
		}
		for _, pet := range pets {
			if pet.Alive && !pet.Deleted {
				nearby = append(nearby, NearbyPet{Owner: other.Username, Name: pet.Name, Species: pet.Species,
					Stage: pet.Stage, SavedAt: pet.SavedAt})
			}
		}
	}
	sort.Slice(nearby, func(i, j int) bool { return nearby[i].SavedAt.After(nearby[j].SavedAt) })
	return nearby, nil
}

// ShareAchievement adds an achievement to the feed. Only players who share
// can post.
func (h *Hub) ShareAchievement(username string, a SharedAchievement) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	me, ok := h.accounts[strings.ToLower(username)]
	if !ok {
		return ErrUnauthorized
	}
	if !me.Sharing {
		return errors.New("turn on sharing to share achievements")
	}
	a.Owner, a.SharedAt = me.Username, h.now()
	a.PetName, a.Name, a.Icon, a.Description = clip(a.PetName, 40), clip(a.Name, 60), clip(a.Icon, 8), clip(a.Description, 200)
	h.feed = append(h.feed, a)
	if over := len(h.feed) - maxFeed; over > 0 {
		h.feed = append(h.feed[:0:0], h.feed[over:]...)
	}
	return writeJSON(filepath.Join(h.dir, "feed.json"), h.feed)
}

// Feed returns up to limit of the most recently shared achievements, newest
// first
func (h *Hub) Feed(limit int) []SharedAchievement {
	h.mu.Lock()
	defer h.mu.Unlock()
	feed := []SharedAchievement{}
	for i := len(h.feed) - 1; i >= 0 && len(feed) < limit; i-- {
		feed = append(feed, h.feed[i])
	}
	return feed
}

// clip shortens text to at most n characters
func clip(text string, n int) string {
	if r := []rune(text); len(r) > n {
		return string(r[:n])
	}
	return text
}

func (h *Hub) saveAccountsLocked() error {
	accounts := make([]*account, 0, len(h.accounts))
	for _, a := range h.accounts {
		accounts = append(accounts, a)
	}
	sort.Slice(accounts, func(i, j int) bool { return accounts[i].Username < accounts[j].Username })
	return writeJSON(filepath.Join(h.dir, "accounts.json"), accounts)
}

// readJSON reads a JSON file into v
func readJSON(path string, v interface{}) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, v)
}

// writeJSON writes v to a file atomically, readable only by its owner
func writeJSON(path string, v interface{}) error {
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// copyFile copies a file, replacing the destination
func copyFile(from, to string) error {
	data, err := os.ReadFile(from)
	if err != nil {
		return err
	}
	return os.WriteFile(to, data, 0o600)
}
