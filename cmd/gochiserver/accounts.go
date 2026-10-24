package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math/big"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/Michael-W-Ellison/gochi/internal/ai"
	"github.com/Michael-W-Ellison/gochi/internal/core"
	"github.com/Michael-W-Ellison/gochi/internal/data"
)

// Player accounts: several players can play on one device, one at a time,
// each with their own pets, Home Stars, house, collection, statistics,
// play schedule, keepsake books and cloud connection. Each account has a
// name the player types, which never leaves this device, and a random
// number, which picks the handle a cloud hub gives the account (see
// hub.Register) without the name being sent. Accounts aren't locked: any
// player can open any account. The parent PIN, the holiday and sound
// settings, paired devices and the hub this server runs are the device's,
// shared by every account.
//
// The data folder holds accounts.json, the device's files and an
// accounts folder with a folder for each account:
//
//	accounts.json, parental.json, devices.json, hub/
//	accounts/<id>/saves, backups, cache, books, household.json,
//	              stats.json, owner.json, cloud.json

// accountsFileName is the list of accounts in the data folder
const accountsFileName = "accounts.json"

// maxAccounts is how many player accounts a device can have
const maxAccounts = 20

// accountFiles are the files and folders each account keeps in its own
// folder. Versions before accounts kept them in the data folder; they are
// moved into the first account.
var accountFiles = []string{"saves", "backups", "cache", "books", householdFileName, statsFileName, "owner.json",
	"cloud.json"}

// accountIDPattern matches an account's ID, which names its folder
var accountIDPattern = regexp.MustCompile(`^[0-9a-f]{16}$`)

// Account errors
var (
	errAccountName    = fmt.Errorf("a player's name is 1 to %d letters, numbers or symbols", core.MaxPetNameLength)
	errAccountTaken   = errors.New("another player has that name")
	errNoAccount      = errors.New("there is no such player")
	errTooManyPlayers = fmt.Errorf("there can be %d players on this device", maxAccounts)
	errAccountOpen    = errors.New("that player is playing now; switch to another player first")
)

// playerAccount is one player on this device
type playerAccount struct {
	ID      string    `json:"id"`     // Random; names the account's folder
	Name    string    `json:"name"`   // Typed by the player; kept on this device only
	Number  uint32    `json:"number"` // Random, 1000 to 9999; picks the account's handle on a cloud hub
	Created time.Time `json:"created"`
}

// accountsData is accounts.json
type accountsData struct {
	Accounts []playerAccount `json:"accounts"`
	Current  string          `json:"current"` // The account being played

	// The account the files from before accounts were moved into
	MovedInto string `json:"moved_into,omitempty"`
}

// accountRegistry is the device's list of accounts
type accountRegistry struct {
	mu   sync.Mutex // Taken after any other lock and never held while taking one
	root string     // The data folder
	data accountsData
}

// dataRoot returns the data folder: dir, or ~/.gochi for ""
func dataRoot(dir string) string {
	if dir != "" {
		return dir
	}
	return filepath.Dir(data.DefaultDataManagerConfig().SavePath)
}

// openAccounts reads accounts.json from the data folder root. The first
// time, it makes the first account, "Player 1", and moves the files from
// before accounts into it; moving is repeated at every start until it is
// done, so a crash while moving loses nothing.
func openAccounts(root string) (*accountRegistry, error) {
	if err := os.MkdirAll(root, 0o700); err != nil {
		return nil, err
	}
	reg := &accountRegistry{root: root}
	contents, err := os.ReadFile(reg.path())
	switch {
	case errors.Is(err, os.ErrNotExist):
	case err != nil:
		return nil, err
	default:
		if err := json.Unmarshal(contents, &reg.data); err != nil {
			return nil, fmt.Errorf("%s: %w", reg.path(), err)
		}
	}
	reg.clean()
	if len(reg.data.Accounts) == 0 {
		first, err := reg.newAccount("Player 1")
		if err != nil {
			return nil, err
		}
		reg.data.Accounts = []playerAccount{first}
		reg.data.Current, reg.data.MovedInto = first.ID, first.ID
		if err := reg.saveLocked(); err != nil {
			return nil, err
		}
	}
	if err := reg.moveOldFiles(); err != nil {
		return nil, err
	}
	return reg, nil
}

// clean drops accounts that aren't valid, and picks a current account if
// the one recorded is gone
func (reg *accountRegistry) clean() {
	var kept []playerAccount
	for _, a := range reg.data.Accounts {
		a.Name = strings.TrimSpace(a.Name)
		if !accountIDPattern.MatchString(a.ID) || !core.ValidPetName(a.Name) || a.Number == 0 ||
			slices.ContainsFunc(kept, func(k playerAccount) bool { return k.ID == a.ID }) {
			log.Printf("Warning: skipping a player account that isn't valid in %s", reg.path())
			continue
		}
		kept = append(kept, a)
	}
	reg.data.Accounts = kept
	if _, ok := reg.findLocked(reg.data.Current); !ok && len(kept) > 0 {
		reg.data.Current = kept[0].ID
	}
}

// moveOldFiles moves the files versions before accounts kept in the data
// folder into the account made for them. A file already in the account
// isn't replaced.
func (reg *accountRegistry) moveOldFiles() error {
	if _, ok := reg.findLocked(reg.data.MovedInto); !ok {
		return nil
	}
	dir := reg.dir(reg.data.MovedInto)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	for _, name := range accountFiles {
		from, to := filepath.Join(reg.root, name), filepath.Join(dir, name)
		if _, err := os.Lstat(from); errors.Is(err, os.ErrNotExist) {
			continue
		}
		if _, err := os.Lstat(to); err == nil {
			log.Printf("Warning: %s was left in place: the player account already has its own", from)
			continue
		}
		if err := os.Rename(from, to); err != nil {
			return fmt.Errorf("moving %s into the first player's folder: %w", from, err)
		}
		log.Printf("Moved %s to %s", from, to)
	}
	return nil
}

// path returns accounts.json's path
func (reg *accountRegistry) path() string {
	return filepath.Join(reg.root, accountsFileName)
}

// dir returns the folder of the account with id
func (reg *accountRegistry) dir(id string) string {
	return filepath.Join(reg.root, "accounts", id)
}

// saveLocked writes accounts.json. Caller must hold reg.mu.
func (reg *accountRegistry) saveLocked() error {
	contents, err := json.MarshalIndent(reg.data, "", "  ")
	if err != nil {
		return err
	}
	tmp := reg.path() + ".tmp"
	if err := os.WriteFile(tmp, contents, 0o600); err != nil {
		_ = os.Remove(tmp) // Cut short (a full disk); the write's error is the one to report
		return err
	}
	return os.Rename(tmp, reg.path())
}

// findLocked returns the account with id. Caller must hold reg.mu (or be
// opening the registry).
func (reg *accountRegistry) findLocked(id string) (playerAccount, bool) {
	for _, a := range reg.data.Accounts {
		if a.ID == id {
			return a, true
		}
	}
	return playerAccount{}, false
}

// current returns the account being played
func (reg *accountRegistry) current() playerAccount {
	reg.mu.Lock()
	defer reg.mu.Unlock()
	a, _ := reg.findLocked(reg.data.Current)
	return a
}

// list returns every account, and the ID of the one being played
func (reg *accountRegistry) list() ([]playerAccount, string) {
	reg.mu.Lock()
	defer reg.mu.Unlock()
	return slices.Clone(reg.data.Accounts), reg.data.Current
}

// find returns the account with id
func (reg *accountRegistry) find(id string) (playerAccount, bool) {
	reg.mu.Lock()
	defer reg.mu.Unlock()
	return reg.findLocked(id)
}

// checkNameLocked returns name, trimmed, if it is a valid name no other
// account than except has. Caller must hold reg.mu.
func (reg *accountRegistry) checkNameLocked(name, except string) (string, error) {
	name = strings.TrimSpace(name)
	if !core.ValidPetName(name) {
		return "", errAccountName
	}
	for _, a := range reg.data.Accounts {
		if a.ID != except && strings.EqualFold(a.Name, name) {
			return "", errAccountTaken
		}
	}
	return name, nil
}

// newAccount returns a new account named name, with a random ID and a
// random number no other account has. Caller must hold reg.mu (or be
// opening the registry).
func (reg *accountRegistry) newAccount(name string) (playerAccount, error) {
	raw := make([]byte, 8)
	if _, err := rand.Read(raw); err != nil {
		return playerAccount{}, err
	}
	a := playerAccount{ID: hex.EncodeToString(raw), Name: name, Created: time.Now().UTC().Truncate(time.Second)}
	for a.Number == 0 || slices.ContainsFunc(reg.data.Accounts, func(o playerAccount) bool { return o.Number == a.Number }) {
		n, err := rand.Int(rand.Reader, big.NewInt(9000))
		if err != nil {
			return playerAccount{}, err
		}
		a.Number = uint32(n.Int64()) + 1000
	}
	return a, nil
}

// create adds an account named name, with its folder
func (reg *accountRegistry) create(name string) (playerAccount, error) {
	reg.mu.Lock()
	defer reg.mu.Unlock()
	name, err := reg.checkNameLocked(name, "")
	if err != nil {
		return playerAccount{}, err
	}
	if len(reg.data.Accounts) >= maxAccounts {
		return playerAccount{}, errTooManyPlayers
	}
	a, err := reg.newAccount(name)
	if err != nil {
		return playerAccount{}, err
	}
	if err := os.MkdirAll(reg.dir(a.ID), 0o700); err != nil {
		return playerAccount{}, err
	}
	reg.data.Accounts = append(reg.data.Accounts, a)
	if err := reg.saveLocked(); err != nil {
		reg.data.Accounts = reg.data.Accounts[:len(reg.data.Accounts)-1]
		return playerAccount{}, err
	}
	return a, nil
}

// rename renames the account with id
func (reg *accountRegistry) rename(id, name string) error {
	reg.mu.Lock()
	defer reg.mu.Unlock()
	i := slices.IndexFunc(reg.data.Accounts, func(a playerAccount) bool { return a.ID == id })
	if i < 0 {
		return errNoAccount
	}
	name, err := reg.checkNameLocked(name, id)
	if err != nil {
		return err
	}
	old := reg.data.Accounts[i].Name
	reg.data.Accounts[i].Name = name
	if err := reg.saveLocked(); err != nil {
		reg.data.Accounts[i].Name = old
		return err
	}
	return nil
}

// setCurrent records the account being played
func (reg *accountRegistry) setCurrent(id string) error {
	reg.mu.Lock()
	defer reg.mu.Unlock()
	old := reg.data.Current
	reg.data.Current = id
	if err := reg.saveLocked(); err != nil {
		reg.data.Current = old
		return err
	}
	return nil
}

// remove deletes the account with id and its folder: its pets, books,
// household and statistics. The account being played can't be removed.
func (reg *accountRegistry) remove(id string) error {
	reg.mu.Lock()
	defer reg.mu.Unlock()
	i := slices.IndexFunc(reg.data.Accounts, func(a playerAccount) bool { return a.ID == id })
	switch {
	case i < 0:
		return errNoAccount
	case id == reg.data.Current:
		return errAccountOpen
	}
	accounts := reg.data.Accounts
	reg.data.Accounts = slices.Delete(slices.Clone(accounts), i, i+1)
	if err := reg.saveLocked(); err != nil {
		reg.data.Accounts = accounts
		return err
	}
	// Removed from the list first: if removing the folder fails, what is
	// left is no longer an account
	if err := os.RemoveAll(reg.dir(id)); err != nil {
		log.Printf("Could not remove all of %s: %v", reg.dir(id), err)
	}
	return nil
}

// accountParts are what each account keeps in its own folder
type accountParts struct {
	store     *petStore
	cloud     *cloud
	owner     *ai.OwnerSchedule
	ownerPath string
	stats     *playerStats
	house     *household
}

// openAccountParts reads an account's files from dir. A damaged file
// other than a pet's save is set aside, and the account starts afresh
// without it.
func openAccountParts(dir, timeScale string, autosaveSeconds int) (accountParts, error) {
	var p accountParts
	var err error
	if p.store, err = newPetStore(dir); err != nil {
		return p, fmt.Errorf("could not set up the save folder: %w", err)
	}
	p.store.timeScale, p.store.autosaveSeconds = timeScale, autosaveSeconds
	if p.cloud, err = setAsideOnError("the cloud settings", filepath.Join(dir, "cloud.json"),
		func() (*cloud, error) { return newCloud(dir) }); err != nil {
		return p, err
	}
	if p.owner, p.ownerPath, err = loadOwnerSchedule(dir); err != nil {
		log.Printf("Warning: could not read the play schedule, starting a new one: %v", err)
		p.owner = &ai.OwnerSchedule{}
	}
	if p.stats, err = setAsideOnError("your statistics", filepath.Join(dir, statsFileName),
		func() (*playerStats, error) { return loadPlayerStats(dir) }); err != nil {
		return p, err
	}
	if p.house, err = setAsideOnError("the household", filepath.Join(dir, householdFileName),
		func() (*household, error) { return loadHousehold(dir) }); err != nil {
		return p, err
	}
	return p, nil
}

// accountNumber returns the number of the account being played, which
// picks its handle on a cloud hub; 0 without accounts
func (s *Server) accountNumber() uint32 {
	if s.accounts == nil {
		return 0
	}
	return s.accounts.current().Number
}

// accountGate lets requests run together, but not while the player
// switches accounts, which replaces the account's parts the requests use.
// The switch itself takes the gate on its own.
func (s *Server) accountGate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/accounts/switch" {
			next.ServeHTTP(w, r)
			return
		}
		s.accountMu.RLock()
		defer s.accountMu.RUnlock()
		next.ServeHTTP(w, r)
	})
}

// saveAccountLocked saves everything of the account being played: its
// play schedule, statistics, household and active pet. Caller must hold
// the gate (accountMu) for writing.
func (s *Server) saveAccountLocked() error {
	if err := s.saveOwner(); err != nil {
		log.Printf("Could not save the play schedule: %v", err)
	}
	s.mu.RLock()
	s.observePet(s.pet, false) // Care since the last tick
	s.mu.RUnlock()
	if s.stats != nil {
		if err := s.stats.save(); err != nil {
			log.Printf("Could not save your statistics: %v", err)
		}
	}
	if s.house != nil {
		if err := s.house.save(); err != nil {
			return fmt.Errorf("could not save the household: %w", err)
		}
	}
	if _, err := s.saveNow(); err != nil {
		return fmt.Errorf("could not save the pet: %w", err)
	}
	return nil
}

// switchAccount saves and closes the account being played, and opens the
// account with id, with its most recently saved pet
func (s *Server) switchAccount(id string) error {
	if s.accounts == nil {
		return errNoAccount
	}
	s.accountMu.Lock()
	defer s.accountMu.Unlock()
	account, ok := s.accounts.find(id)
	if !ok {
		return errNoAccount
	}
	if s.accounts.current().ID == id {
		return nil
	}
	if err := s.saveAccountLocked(); err != nil {
		return err
	}
	parts, err := openAccountParts(s.accounts.dir(id), s.store.timeScale, s.store.autosaveSeconds)
	if err != nil {
		s.house.reapply() // Opening the household told the game about it
		return err
	}
	if err := s.accounts.setCurrent(id); err != nil {
		parts.house.reapply()
		s.house.reapply()
		return err
	}
	s.leavePlaydate()

	s.saveMu.Lock()
	s.mu.Lock()
	s.store, s.cloud, s.owner, s.ownerPath, s.stats, s.house = parts.store, parts.cloud, parts.owner,
		parts.ownerPath, parts.stats, parts.house
	s.pet, s.slot, s.popups, s.holidayBought = nil, "", nil, false
	s.lastHouseholdAwards, s.householdDays, s.lastHouseholdTime = time.Time{}, 0, time.Time{}
	now := time.Now()
	s.clock.SetPaused(true, now) // The account's pet's time starts from now
	s.manualPause = false
	s.markActiveLocked(now)
	pet, slot, path, warning, err := s.store.LoadMostRecent()
	switch {
	case err == nil:
		s.pet, s.slot = pet, slot
		log.Printf("%s is playing; restored %s from %s", account.Name, pet.Name, path)
		if warning != nil {
			log.Printf("Warning: %v", warning)
		}
	case errors.Is(err, os.ErrNotExist):
		log.Printf("%s is playing, with no saved pets yet", account.Name)
	default:
		log.Printf("Warning: could not restore %s's saved pet: %v", account.Name, err)
	}
	s.mu.Unlock()
	s.saveMu.Unlock()

	s.arMu.Lock()
	s.arService, s.arTouches = nil, nil // The AR scene was the other player's pet's
	s.arMu.Unlock()

	s.recordSavedPets()
	s.scanAchievements()
	s.mu.Lock()
	s.observePet(s.pet, false)
	if s.pet != nil {
		s.noteAchievementsLocked(s.pet)
	}
	s.mu.Unlock()
	return nil
}

// leavePlaydate leaves the playdate the player is at, if any, before the
// account changes: the playdate was the other player's
func (s *Server) leavePlaydate() {
	s.playdates.mu.Lock()
	defer s.playdates.mu.Unlock()
	pd := s.playdates.session
	if pd == nil {
		return
	}
	s.playdates.session = nil
	client, err := s.hubClient()
	if err != nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), playdateTimeout)
	defer cancel()
	if err := client.LeavePlaydate(ctx, pd.code); err != nil {
		log.Printf("Could not leave the playdate on the hub (it ends there when it times out): %v", err)
	}
}

// accountView is an account as the page shows it
type accountView struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Number  uint32 `json:"number"`
	Current bool   `json:"current"`
}

// accountsStatus is what the accounts routes return
type accountsStatus struct {
	Accounts []accountView `json:"accounts"`
	Max      int           `json:"max"`
}

func (s *Server) accountsStatus() accountsStatus {
	status := accountsStatus{Accounts: []accountView{}, Max: maxAccounts}
	if s.accounts == nil {
		return status
	}
	accounts, current := s.accounts.list()
	for _, a := range accounts {
		status.Accounts = append(status.Accounts, accountView{ID: a.ID, Name: a.Name, Number: a.Number,
			Current: a.ID == current})
	}
	return status
}

// writeAccountError reports an error from an accounts route
func writeAccountError(w http.ResponseWriter, err error) {
	status := http.StatusInternalServerError
	switch {
	case errors.Is(err, errAccountName):
		status = http.StatusBadRequest
	case errors.Is(err, errNoAccount):
		status = http.StatusNotFound
	case errors.Is(err, errAccountTaken), errors.Is(err, errTooManyPlayers), errors.Is(err, errAccountOpen):
		status = http.StatusConflict
	default:
		log.Printf("Player accounts: %v", err)
		err = errors.New("could not save the change")
	}
	writeJSON(w, status, APIResponse{Success: false, Error: capitalize(err.Error()) + "."})
}

// handleAccounts handles GET /api/accounts - the players on this device
// and which one is playing
func (s *Server) handleAccounts(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, APIResponse{Success: false, Error: "Method not allowed"})
		return
	}
	writeJSON(w, http.StatusOK, APIResponse{Success: true, Data: s.accountsStatus()})
}

// handleAccountCreate handles POST /api/accounts/create with {"name"}:
// adds a player, on the computer running the game
func (s *Server) handleAccountCreate(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name string `json:"name"`
	}
	if !onlyThisComputer(w, r) || !decodePost(w, r, &req) {
		return
	}
	if s.accounts == nil {
		writeAccountError(w, errNoAccount)
		return
	}
	a, err := s.accounts.create(req.Name)
	if err != nil {
		writeAccountError(w, err)
		return
	}
	log.Printf("Added a player: %s", a.Name)
	writeJSON(w, http.StatusOK, APIResponse{Success: true, Data: map[string]interface{}{
		"id": a.ID, "accounts": s.accountsStatus()}})
}

// handleAccountSwitch handles POST /api/accounts/switch with {"id"}: saves
// and closes the account being played and opens that one
func (s *Server) handleAccountSwitch(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ID string `json:"id"`
	}
	if !decodePost(w, r, &req) {
		return
	}
	if err := s.switchAccount(req.ID); err != nil {
		writeAccountError(w, err)
		return
	}
	s.accountMu.RLock()
	defer s.accountMu.RUnlock()
	writeJSON(w, http.StatusOK, APIResponse{Success: true, Data: s.accountsStatus()})
}

// handleAccountRename handles POST /api/accounts/rename with {"id",
// "name"}, on the computer running the game
func (s *Server) handleAccountRename(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}
	if !onlyThisComputer(w, r) || !decodePost(w, r, &req) {
		return
	}
	if s.accounts == nil {
		writeAccountError(w, errNoAccount)
		return
	}
	if err := s.accounts.rename(req.ID, req.Name); err != nil {
		writeAccountError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, APIResponse{Success: true, Data: s.accountsStatus()})
}

// handleAccountDelete handles POST /api/accounts/delete with {"id",
// "confirm_name", "parent_pin"}: deletes a player who isn't playing, with
// their pets and everything else of theirs on this device, on the
// computer running the game. The name must be typed to confirm it.
func (s *Server) handleAccountDelete(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ID          string `json:"id"`
		ConfirmName string `json:"confirm_name"`
		ParentPIN   string `json:"parent_pin"`
	}
	if !onlyThisComputer(w, r) || !decodePost(w, r, &req) {
		return
	}
	if s.accounts == nil {
		writeAccountError(w, errNoAccount)
		return
	}
	a, ok := s.accounts.find(req.ID)
	if !ok {
		writeAccountError(w, errNoAccount)
		return
	}
	if strings.TrimSpace(req.ConfirmName) != a.Name {
		writeJSON(w, http.StatusBadRequest, APIResponse{Success: false,
			Error: "Type the player's name to confirm deleting them"})
		return
	}
	if !s.parentAllows(w, req.ParentPIN) {
		return
	}
	if err := s.accounts.remove(req.ID); err != nil {
		writeAccountError(w, err)
		return
	}
	log.Printf("Deleted a player: %s", a.Name)
	writeJSON(w, http.StatusOK, APIResponse{Success: true, Data: s.accountsStatus()})
}
