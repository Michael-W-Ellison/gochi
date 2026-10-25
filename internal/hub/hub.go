// Package hub is the cloud service: a Gochi server can run as a hub that
// other Gochi servers connect to. Players register with the hub's join
// code and get an account key and a handle the hub chooses (see
// handles.go). The hub keeps an encrypted copy of each player's pet saves,
// so they can be backed up and synced between devices; it keeps a feed of
// achievements players choose to share; and it hosts AR playdates, where
// players bring their pets to the same place.
//
// Players can't send each other anything they write: no names, no
// messages. Everything another player sees is either chosen from fixed
// lists (species, achievements, how a pet looks) or written by the hub
// itself. The hub is self-hosted, stores everything as files in its
// directory, and stores account keys only as hashes.
package hub

import (
	"context"
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

	"github.com/Michael-W-Ellison/gochi/internal/core"
)

// Limits
const (
	MaxPetsPerAccount = 50
	MaxSaveBytes      = 4 << 20  // A pet's (encrypted) save
	MaxAccountBytes   = 64 << 20 // All of an account's saves
	MaxAccounts       = 10000
	maxFeed           = 200 // Achievements kept in the feed
)

// Errors
var (
	ErrRegistrationClosed = errors.New("this hub has no join code, so it doesn't accept new players")
	ErrWrongJoinCode      = errors.New("wrong join code")
	ErrHubFull            = errors.New("this hub has as many players as it can take")
	ErrUnauthorized       = errors.New("unknown account token")
	ErrBadKey             = errors.New("invalid pet key")
	ErrNotFound           = errors.New("no such pet")
	ErrVersionConflict    = errors.New("the hub has a different version of this pet")
	ErrTooManyPets        = errors.New("too many pets for one account")
	ErrAccountFull        = errors.New("this account's saves take up all the space the hub allows")
	ErrUnknownAchievement = errors.New("unknown achievement")
)

var (
	keyPattern = regexp.MustCompile(`^[a-f0-9]{32}$`)
	idPattern  = regexp.MustCompile(`^[a-f0-9]{32}$`)
)

// PetKey returns the hub's key for a pet: a hash of its ID, so nothing
// about the pet appears on the hub
func PetKey(petID string) string {
	sum := sha256.Sum256([]byte(petID))
	return hex.EncodeToString(sum[:16])
}

// PetInfo describes a pet stored on the hub: only what syncing needs. The
// pet itself, name and all, is in its encrypted save.
type PetInfo struct {
	Key     string    `json:"key"`
	Version int       `json:"version"`
	SavedAt time.Time `json:"saved_at"` // When the pet was last played (time last passed for it)
	Deleted bool      `json:"deleted"`  // Deleted by the player; its save is gone
	Size    int       `json:"size"`     // Bytes of its save, set by the hub
}

// storedPet is a pet's file on the hub
type storedPet struct {
	PetInfo
	Save json.RawMessage `json:"save,omitempty"`
}

// Profile is a player's settings on the hub
type Profile struct {
	Handle            string `json:"handle"`             // Chosen by the hub
	ShareAchievements bool   `json:"share_achievements"` // Their pets' achievements go to the feed; off for a new account
}

// account is a registered player
type account struct {
	ID        string    `json:"id"` // Random; names the account's files
	Handle    string    `json:"handle"`
	TokenHash string    `json:"token_hash"`
	Created   time.Time `json:"created"`
	Share     bool      `json:"share_achievements"`

	// Username is the name players chose on hubs before handles; accounts
	// that have one are given an ID and a handle and it is dropped
	Username string `json:"username,omitempty"`
}

// SharedAchievement is an achievement in the feed. The player sends only
// the achievement's ID and their pet's species; the rest is the game's
// own text.
type SharedAchievement struct {
	Handle      string    `json:"handle"`
	Species     string    `json:"species"`
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
	accounts map[string]*account // By ID
	byToken  map[string]string   // Token hash -> ID
	byHandle map[string]string   // Handle -> ID
	feed     []SharedAchievement

	playdates map[string]*playdate // By code; kept in memory only

	// Each account's pets' information, so listing pets and checking
	// versions don't read saves
	pets map[string]map[string]PetInfo // By account ID, then key

	// Uploads being read, so many at once can't use up the hub's memory:
	// a place for each upload across the hub, and the number per account
	uploads   chan struct{}
	uploading map[string]int
}

// Uploads read at once
const (
	maxUploads        = 16 // Across the hub
	maxAccountUploads = 2  // From one account
	uploadWait        = 30 * time.Second
)

// ErrBusy is returned when the hub is reading too many uploads to take
// another now
var ErrBusy = errors.New("the hub is busy; try again shortly")

// startUpload reserves a place for an upload from an account, waiting a
// while for one. Call the returned function when the upload has been
// handled.
func (h *Hub) startUpload(ctx context.Context, id string) (func(), error) {
	h.mu.Lock()
	if h.uploading[id] >= maxAccountUploads {
		h.mu.Unlock()
		return nil, ErrBusy
	}
	h.uploading[id]++
	h.mu.Unlock()
	release := func() {
		h.mu.Lock()
		if h.uploading[id]--; h.uploading[id] <= 0 {
			delete(h.uploading, id)
		}
		h.mu.Unlock()
	}

	timer := time.NewTimer(uploadWait)
	defer timer.Stop()
	select {
	case h.uploads <- struct{}{}:
		return func() { <-h.uploads; release() }, nil
	case <-ctx.Done():
	case <-timer.C:
	}
	release()
	return nil, ErrBusy
}

// New opens (or creates) a hub keeping its files in dir. Players need
// joinCode to register; with an empty join code registration is closed.
func New(dir, joinCode string) (*Hub, error) {
	if err := os.MkdirAll(filepath.Join(dir, "pets"), 0o700); err != nil {
		return nil, err
	}
	h := &Hub{dir: dir, joinCode: joinCode, now: time.Now,
		accounts: make(map[string]*account), byToken: make(map[string]string), byHandle: make(map[string]string),
		playdates: make(map[string]*playdate), pets: make(map[string]map[string]PetInfo),
		uploads: make(chan struct{}, maxUploads), uploading: make(map[string]int)}

	var accounts []*account
	if err := readJSON(filepath.Join(dir, "accounts.json"), &accounts); err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	migrated := false
	for _, a := range accounts {
		if a.ID == "" || a.Handle == "" || a.Username != "" {
			if err := h.migrateLocked(a); err != nil {
				return nil, err
			}
			migrated = true
		}
		h.addAccountLocked(a)
		if err := h.loadPetIndexLocked(a.ID); err != nil {
			return nil, err
		}
	}
	if migrated {
		if err := h.saveAccountsLocked(); err != nil {
			return nil, err
		}
	}

	// The feed keeps only entries in the current form (older ones held
	// text players typed)
	var feed []SharedAchievement
	if err := readJSON(filepath.Join(dir, "feed.json"), &feed); err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	for _, a := range feed {
		if ValidHandle(a.Handle) && core.Species(a.Species).Known() {
			if known, ok := core.FindAchievement(a.ID); ok {
				a.Name, a.Icon, a.Description = known.Name, known.Icon, known.Description
				h.feed = append(h.feed, a)
			}
		}
	}
	return h, nil
}

// migrateLocked gives an account from an older hub an ID and a handle,
// moves its pets to the ID's directory, and forgets its chosen username.
// Caller must hold h.mu.
func (h *Hub) migrateLocked(a *account) error {
	if a.ID == "" {
		id, err := randomHex(16)
		if err != nil {
			return err
		}
		a.ID = id
	}
	if a.Handle == "" || !ValidHandle(a.Handle) {
		handle, err := newHandle(func(s string) bool { _, ok := h.byHandle[s]; return ok })
		if err != nil {
			return err
		}
		a.Handle = handle
	}
	if a.Username != "" {
		old := filepath.Join(h.dir, "pets", strings.ToLower(a.Username))
		if _, err := os.Stat(old); err == nil {
			if err := os.Rename(old, h.petDir(a.ID)); err != nil {
				return err
			}
		}
		a.Username = ""
	}
	return nil
}

// addAccountLocked adds an account to the indexes. Caller must hold h.mu.
func (h *Hub) addAccountLocked(a *account) {
	h.accounts[a.ID] = a
	h.byToken[a.TokenHash] = a.ID
	h.byHandle[a.Handle] = a.ID
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

// randomHex returns n random bytes in hex
func randomHex(n int) (string, error) {
	raw := make([]byte, n)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	return hex.EncodeToString(raw), nil
}

// Register creates an account, giving it a handle, and returns its token
// and handle. The handle is random, or, with a player's number (seed, 0
// for none), the one that number gives (see seededHandle).
func (h *Hub) Register(joinCode string, seed uint32) (token, handle string, err error) {
	if h.joinCode == "" {
		return "", "", ErrRegistrationClosed
	}
	if subtle.ConstantTimeCompare([]byte(joinCode), []byte(h.joinCode)) != 1 {
		return "", "", ErrWrongJoinCode
	}
	if token, err = randomHex(32); err != nil {
		return "", "", err
	}
	id, err := randomHex(16)
	if err != nil {
		return "", "", err
	}

	h.mu.Lock()
	defer h.mu.Unlock()
	if len(h.accounts) >= MaxAccounts {
		return "", "", ErrHubFull
	}
	taken := func(s string) bool { _, ok := h.byHandle[s]; return ok }
	if seed != 0 {
		handle, err = seededHandle(seed, taken)
	} else {
		handle, err = newHandle(taken)
	}
	if err != nil {
		return "", "", err
	}
	a := &account{ID: id, Handle: handle, TokenHash: hashToken(token), Created: h.now()}
	h.addAccountLocked(a)
	if err := h.saveAccountsLocked(); err != nil {
		delete(h.accounts, id)
		delete(h.byToken, a.TokenHash)
		delete(h.byHandle, handle)
		return "", "", err
	}
	if err := os.MkdirAll(h.petDir(id), 0o700); err != nil {
		return "", "", err
	}
	return token, handle, nil
}

// Authenticate returns the ID of the account a token belongs to
func (h *Hub) Authenticate(token string) (string, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if id, ok := h.byToken[hashToken(token)]; ok && token != "" {
		return id, nil
	}
	return "", ErrUnauthorized
}

// accountLocked returns an account by ID. Caller must hold h.mu.
func (h *Hub) accountLocked(id string) (*account, error) {
	a, ok := h.accounts[id]
	if !ok {
		return nil, ErrUnauthorized
	}
	return a, nil
}

// petDir is where an account's pets are stored
func (h *Hub) petDir(id string) string {
	return filepath.Join(h.dir, "pets", id)
}

// ListPets returns the account's pets on the hub
func (h *Hub) ListPets(id string) ([]PetInfo, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	pets := make([]PetInfo, 0, len(h.pets[id]))
	for _, info := range h.pets[id] {
		pets = append(pets, info)
	}
	sort.Slice(pets, func(i, j int) bool { return pets[i].Key < pets[j].Key })
	return pets, nil
}

// loadPetIndexLocked reads an account's pets' information into the index.
// Caller must hold h.mu.
func (h *Hub) loadPetIndexLocked(id string) error {
	entries, err := os.ReadDir(h.petDir(id))
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	index := make(map[string]PetInfo)
	for _, e := range entries {
		key, ok := strings.CutSuffix(e.Name(), ".json")
		if !ok || !keyPattern.MatchString(key) {
			continue
		}
		var info PetInfo // Decoding only the information skips the save
		if err := readJSON(filepath.Join(h.petDir(id), e.Name()), &info); err != nil {
			continue
		}
		if stat, err := e.Info(); err == nil && info.Size == 0 && !info.Deleted {
			info.Size = int(stat.Size())
		}
		index[key] = PetInfo{Key: key, Version: info.Version, SavedAt: info.SavedAt, Deleted: info.Deleted, Size: info.Size}
	}
	h.pets[id] = index
	return nil
}

// GetPet returns a pet's information and save
func (h *Hub) GetPet(id, key string) (PetInfo, json.RawMessage, error) {
	if !keyPattern.MatchString(key) || !idPattern.MatchString(id) {
		return PetInfo{}, nil, ErrBadKey
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	info, ok := h.pets[id][key]
	if !ok {
		return PetInfo{}, nil, ErrNotFound
	}
	var pet storedPet
	if err := readJSON(filepath.Join(h.petDir(id), key+".json"), &pet); err != nil {
		return PetInfo{}, nil, err
	}
	return info, pet.Save, nil
}

// accountBytesLocked returns the bytes of an account's saves, leaving out
// one pet. Caller must hold h.mu.
func (h *Hub) accountBytesLocked(id, except string) int {
	total := 0
	for key, info := range h.pets[id] {
		if key != except {
			total += info.Size
		}
	}
	return total
}

// PutPet stores a new version of a pet. baseVersion is the version the
// player's copy was based on (0 for a pet new to the hub); if the hub has
// another version, nothing changes and ErrVersionConflict is returned with
// the hub's version. The previous version is kept as a backup. Only the
// key, the version and when it was last played are taken from info.
func (h *Hub) PutPet(id string, info PetInfo, save json.RawMessage, baseVersion int) (int, error) {
	if !keyPattern.MatchString(info.Key) || !idPattern.MatchString(id) {
		return 0, ErrBadKey
	}
	if len(save) == 0 || len(save) > MaxSaveBytes || !json.Valid(save) {
		return 0, fmt.Errorf("the save must be valid JSON of at most %d bytes", MaxSaveBytes)
	}

	h.mu.Lock()
	defer h.mu.Unlock()
	if _, err := h.accountLocked(id); err != nil {
		return 0, err
	}
	current, exists := h.pets[id][info.Key]
	if !exists && len(h.pets[id]) >= MaxPetsPerAccount {
		return 0, ErrTooManyPets
	}
	if h.accountBytesLocked(id, info.Key)+len(save) > MaxAccountBytes {
		return 0, ErrAccountFull
	}
	if current.Version != baseVersion {
		return current.Version, ErrVersionConflict
	}

	path := filepath.Join(h.petDir(id), info.Key+".json")
	if current.Version > 0 && !current.Deleted {
		if err := copyFile(path, filepath.Join(h.petDir(id), info.Key+".prev")); err != nil {
			return 0, err
		}
	}
	stored := PetInfo{Key: info.Key, Version: current.Version + 1, SavedAt: info.SavedAt, Size: len(save)}
	if err := writeJSON(path, storedPet{PetInfo: stored, Save: save}); err != nil {
		return 0, err
	}
	if h.pets[id] == nil {
		h.pets[id] = make(map[string]PetInfo)
	}
	h.pets[id][info.Key] = stored
	return stored.Version, nil
}

// DeletePet replaces a pet's save with a record that it was deleted, so
// the player's other devices know not to upload it again
func (h *Hub) DeletePet(id, key string, baseVersion int) (int, error) {
	if !keyPattern.MatchString(key) || !idPattern.MatchString(id) {
		return 0, ErrBadKey
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	info, ok := h.pets[id][key]
	if !ok {
		return 0, ErrNotFound
	}
	if info.Version != baseVersion {
		return info.Version, ErrVersionConflict
	}
	info.Version++
	info.Deleted, info.Size = true, 0
	_ = os.Remove(filepath.Join(h.petDir(id), key+".prev")) // Usually there is none
	if err := writeJSON(filepath.Join(h.petDir(id), key+".json"), storedPet{PetInfo: info}); err != nil {
		return 0, err
	}
	h.pets[id][key] = info
	return info.Version, nil
}

// GetProfile returns a player's handle and settings
func (h *Hub) GetProfile(id string) (Profile, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	a, err := h.accountLocked(id)
	if err != nil {
		return Profile{}, err
	}
	return Profile{Handle: a.Handle, ShareAchievements: a.Share}, nil
}

// SetSharing changes whether a player's achievements go to the feed
func (h *Hub) SetSharing(id string, shareAchievements bool) (Profile, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	a, err := h.accountLocked(id)
	if err != nil {
		return Profile{}, err
	}
	before := a.Share
	a.Share = shareAchievements
	if err := h.saveAccountsLocked(); err != nil {
		a.Share = before
		return Profile{}, err
	}
	return Profile{Handle: a.Handle, ShareAchievements: a.Share}, nil
}

// ShareAchievement adds an achievement, by its ID, to the feed, with the
// species of the pet that earned it. Only players who share achievements
// can post.
func (h *Hub) ShareAchievement(id, achievementID, species string) error {
	known, ok := core.FindAchievement(achievementID)
	if !ok {
		return ErrUnknownAchievement
	}
	if !core.Species(species).Known() {
		return errors.New("unknown species")
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	me, err := h.accountLocked(id)
	if err != nil {
		return err
	}
	if !me.Share {
		return errors.New("turn on sharing achievements to share them")
	}
	h.feed = append(h.feed, SharedAchievement{Handle: me.Handle, Species: species, ID: known.ID, Name: known.Name,
		Icon: known.Icon, Description: known.Description, SharedAt: h.now()})
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

func (h *Hub) saveAccountsLocked() error {
	accounts := make([]*account, 0, len(h.accounts))
	for _, a := range h.accounts {
		accounts = append(accounts, a)
	}
	sort.Slice(accounts, func(i, j int) bool { return accounts[i].ID < accounts[j].ID })
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
		_ = os.Remove(tmp) // Cut short (a full disk); the write's error is the one to report
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
