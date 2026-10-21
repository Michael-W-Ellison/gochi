package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/Michael-W-Ellison/gochi/internal/core"
	"github.com/Michael-W-Ellison/gochi/internal/hub"
)

// Cloud: the player can connect this server to a Gochi hub (see the hub
// package) to back up and sync their pets between devices and share
// achievements. The hub gives the player a handle; nothing they type is
// sent to it. The connection and what has been synced are kept in
// cloud.json in the data directory.

// cloudSyncInterval is how often pets are synced while connected
const cloudSyncInterval = 5 * time.Minute

// cloudPet is what was last synced for one pet
type cloudPet struct {
	Slot    string `json:"slot"`
	Version int    `json:"version"`  // The hub's version when last synced
	PetHash string `json:"pet_hash"` // The pet's hash when last synced: it has changed here if this differs
	Deleted bool   `json:"deleted"`  // Deleted on another device; no longer synced
}

// cloudState is the cloud connection, saved in cloud.json
type cloudState struct {
	HubURL     string               `json:"hub_url"`
	Handle     string               `json:"handle"` // Given by the hub
	Token      string               `json:"token"`
	Pin        string               `json:"pin"` // The hub's certificate fingerprint, if it isn't from a certificate authority
	ShareFeats bool                 `json:"share_achievements"`
	LastSync   time.Time            `json:"last_sync"`
	LastReport syncReport           `json:"last_report"`
	Pets       map[string]*cloudPet `json:"pets"` // By hub key
}

// syncReport describes what a sync did
type syncReport struct {
	Uploaded         []string `json:"uploaded"`
	Downloaded       []string `json:"downloaded"`
	Conflicts        []string `json:"conflicts"`         // Changed on both sides; the newer save was kept
	DeletedElsewhere []string `json:"deleted_elsewhere"` // Deleted on another device
	Errors           []string `json:"errors"`
}

// summary describes the report in a sentence
func (r syncReport) summary() string {
	var parts []string
	add := func(list []string, what string) {
		if len(list) > 0 {
			parts = append(parts, what+" "+strings.Join(list, ", "))
		}
	}
	add(r.Uploaded, "Backed up")
	add(r.Downloaded, "Downloaded")
	add(r.Conflicts, "Kept the newer save of")
	add(r.DeletedElsewhere, "Deleted on another device:")
	add(r.Errors, "Problems:")
	if len(parts) == 0 {
		return "Everything is up to date."
	}
	return strings.Join(parts, ". ") + "."
}

// cloud is the server's cloud connection
type cloud struct {
	mu    sync.Mutex // Held for syncs and changes; acquire before Server.saveMu
	path  string
	state cloudState

	shares chan sharedAchievement // Achievements to share, from the update loop
}

// sharedAchievement is an achievement waiting to be shared: its ID and
// the species of the pet that earned it
type sharedAchievement struct {
	id      string
	species core.Species
}

// newCloud loads the cloud connection from dir
func newCloud(dir string) (*cloud, error) {
	c := &cloud{path: filepath.Join(dir, "cloud.json"), shares: make(chan sharedAchievement, 32)}
	data, err := os.ReadFile(c.path)
	switch {
	case errors.Is(err, os.ErrNotExist):
	case err != nil:
		return nil, err
	default:
		if err := json.Unmarshal(data, &c.state); err != nil {
			return nil, fmt.Errorf("%s: %w", c.path, err)
		}
	}
	if c.state.Pets == nil {
		c.state.Pets = make(map[string]*cloudPet)
	}
	// Connections from older versions used usernames and plain HTTP; they
	// are forgotten, and the player connects again
	if c.state.HubURL != "" {
		if _, err := hub.NormalizeURL(c.state.HubURL); err != nil || c.state.Handle == "" {
			log.Printf("Forgetting the connection to %s, made by an older version; connect again from the Cloud panel",
				c.state.HubURL)
			c.state = cloudState{Pets: make(map[string]*cloudPet)}
			if err := c.saveLocked(); err != nil {
				return nil, err
			}
		}
	}
	return c, nil
}

// saveLocked writes cloud.json, readable only by its owner since it holds
// the account token. Caller must hold c.mu.
func (c *cloud) saveLocked() error {
	data, err := json.MarshalIndent(c.state, "", "  ")
	if err != nil {
		return err
	}
	tmp := c.path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		os.Remove(tmp) // Cut short (a full disk)
		return err
	}
	return os.Rename(tmp, c.path)
}

// connectedLocked reports whether the server is connected to a hub.
// Caller must hold c.mu.
func (c *cloud) connectedLocked() bool {
	return c.state.HubURL != "" && c.state.Token != ""
}

// clientLocked returns a hub client. Caller must hold c.mu.
func (c *cloud) clientLocked() *hub.Client {
	return hub.NewClient(c.state.HubURL, c.state.Token, c.state.Pin)
}

// errNotConnected is returned when a cloud feature is used without a hub
var errNotConnected = errors.New("not connected to a cloud hub")

// cloudConnect connects to a hub: with an account key (from another
// device), it signs in to that account; otherwise it registers a new
// account with the hub's join code, and the hub gives it a handle. A hub
// whose certificate isn't from a certificate authority has its
// certificate remembered (pinned) now and required from then on.
func (s *Server) cloudConnect(ctx context.Context, address, joinCode, accountKey string) error {
	if s.cloud == nil {
		return errNotConnected
	}
	baseURL, err := hub.NormalizeURL(address)
	if err != nil {
		return err
	}
	pin, err := hub.ProbeCertificate(ctx, baseURL)
	if err != nil {
		return err
	}
	client := hub.NewClient(baseURL, "", pin)
	if _, err := client.Status(ctx); err != nil {
		return err
	}
	token := strings.TrimSpace(accountKey)
	if token == "" {
		if token, _, err = client.Register(ctx, joinCode); err != nil {
			return err
		}
	}
	client.Token = token
	profile, err := client.GetProfile(ctx)
	if errors.Is(err, hub.ErrUnauthorized) {
		return errors.New("that account key isn't one this hub knows")
	}
	if err != nil {
		return err
	}

	c := s.cloud
	c.mu.Lock()
	defer c.mu.Unlock()
	c.state = cloudState{HubURL: baseURL, Handle: profile.Handle, Token: token, Pin: pin,
		ShareFeats: profile.ShareAchievements, Pets: make(map[string]*cloudPet)}
	return c.saveLocked()
}

// cloudDisconnect forgets the hub account. Nothing is deleted from the hub.
func (s *Server) cloudDisconnect() error {
	c := s.cloud
	if c == nil {
		return nil
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.state = cloudState{Pets: make(map[string]*cloudPet)}
	return c.saveLocked()
}

// cloudStatus describes the cloud connection for the page. The account
// key isn't part of it (see handleCloudKey).
type cloudStatus struct {
	Connected   bool       `json:"connected"`
	HubURL      string     `json:"hub_url"`
	Handle      string     `json:"handle"`
	Fingerprint string     `json:"fingerprint"`        // The hub's pinned certificate, or "" if from a certificate authority
	ShareFeats  bool       `json:"share_achievements"` // Achievements are posted to the hub
	LastSync    *time.Time `json:"last_sync"`
	LastReport  string     `json:"last_report"`
	Hosting     bool       `json:"hosting"` // This server is a hub too
}

func (s *Server) cloudStatus() cloudStatus {
	c := s.cloud
	if c == nil {
		return cloudStatus{Hosting: s.hub != nil}
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	status := cloudStatus{Connected: c.connectedLocked(), HubURL: c.state.HubURL, Handle: c.state.Handle,
		Fingerprint: c.state.Pin, ShareFeats: c.state.ShareFeats, Hosting: s.hub != nil}
	if !c.state.LastSync.IsZero() {
		at := c.state.LastSync
		status.LastSync = &at
		status.LastReport = c.state.LastReport.summary()
	}
	return status
}

// cloudSetProfile sets whether the player shares achievements
// (enableSocialFeatures)
func (s *Server) cloudSetProfile(ctx context.Context, shareAchievements bool) error {
	c := s.cloud
	if c == nil {
		return errNotConnected
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.connectedLocked() {
		return errNotConnected
	}
	profile, err := c.clientLocked().EnableSocialFeatures(ctx, shareAchievements)
	if err != nil {
		return err
	}
	c.state.ShareFeats = profile.ShareAchievements
	return c.saveLocked()
}

// cloudFeed returns achievements players have shared
func (s *Server) cloudFeed(ctx context.Context) ([]hub.SharedAchievement, error) {
	c := s.cloud
	if c == nil {
		return nil, errNotConnected
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.connectedLocked() {
		return nil, errNotConnected
	}
	return c.clientLocked().Feed(ctx)
}

// shareAchievementLocked queues an achievement the active pet earned, to
// share through the hub if the player shares. It never blocks the update
// loop. Caller must hold s.mu.
func (s *Server) shareAchievementLocked(a core.Achievement) {
	if s.cloud == nil {
		return
	}
	select {
	case s.cloud.shares <- sharedAchievement{id: a.ID, species: s.pet.GetSpecies()}:
	default: // Too many waiting; this one isn't shared
	}
}

// shareLoop shares queued achievements while the player is connected and
// sharing achievements
func (s *Server) shareLoop(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case shared := <-s.cloud.shares:
			c := s.cloud
			c.mu.Lock()
			if c.connectedLocked() && c.state.ShareFeats {
				if err := c.clientLocked().ShareAchievement(ctx, shared.id, string(shared.species)); err != nil {
					log.Printf("Could not share an achievement: %v", err)
				}
			}
			c.mu.Unlock()
		}
	}
}

// syncLoop syncs pets every cloudSyncInterval while connected
func (s *Server) syncLoop(ctx context.Context) {
	ticker := time.NewTicker(cloudSyncInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if _, err := s.cloudSync(ctx); err != nil && !errors.Is(err, errNotConnected) {
				log.Printf("Cloud sync failed: %v", err)
			}
		}
	}
}

// cloudSync saves the active pet, then syncs every pet with the hub
// (saveToCloud, loadFromCloud and synchronize): a pet changed only here is
// uploaded, one changed only on the hub is downloaded, and one changed on
// both keeps the newer save (the other stays as a backup on its side). A
// pet new on the hub is downloaded into a new slot. If the active pet was
// downloaded, it is reloaded.
func (s *Server) cloudSync(ctx context.Context) (syncReport, error) {
	c := s.cloud
	if c == nil {
		return syncReport{}, errNotConnected
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.connectedLocked() || s.store == nil {
		return syncReport{}, errNotConnected
	}
	if _, err := s.saveNow(); err != nil {
		return syncReport{}, fmt.Errorf("could not save before syncing: %w", err)
	}
	client := c.clientLocked()
	remote, err := client.ListPets(ctx)
	if err != nil {
		return syncReport{}, err
	}

	s.saveMu.Lock()
	defer s.saveMu.Unlock()
	local, err := s.store.List()
	if err != nil {
		return syncReport{}, err
	}

	report := syncReport{Uploaded: []string{}, Downloaded: []string{}, Conflicts: []string{}, DeletedElsewhere: []string{},
		Errors: []string{}}
	localByKey := make(map[string]savedPet)
	for _, pet := range local {
		if pet.ID != "" {
			localByKey[hub.PetKey(pet.ID)] = pet
		}
	}
	remoteByKey := make(map[string]hub.PetInfo)
	var householdInfo *hub.PetInfo
	for _, info := range remote {
		if info.Key == householdKey {
			householdInfo = &info // Synced on its own, below
			continue
		}
		remoteByKey[info.Key] = info
	}
	keys := make([]string, 0, len(localByKey)+len(remoteByKey))
	for key := range localByKey {
		keys = append(keys, key)
	}
	for key := range remoteByKey {
		if _, ok := localByKey[key]; !ok {
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)

	reloadActive := false
	for _, key := range keys {
		pet, isLocal := localByKey[key]
		info, isRemote := remoteByKey[key]
		state := c.state.Pets[key]
		name := pet.Name
		if !isLocal {
			name = "a pet from another device" // Its name is only in its encrypted save
		}
		fail := func(err error) { report.Errors = append(report.Errors, name+": "+err.Error()) }

		upload := func(base int) bool {
			saved, err := s.store.Export(pet.Slot)
			if err != nil {
				fail(err)
				return false
			}
			version, err := client.UploadPetData(ctx, hub.PetInfo{Key: key, SavedAt: saved.LastPlayed}, saved.JSON, base)
			if err != nil {
				fail(err)
				return false
			}
			c.state.Pets[key] = &cloudPet{Slot: pet.Slot, Version: version, PetHash: saved.PetHash}
			return true
		}
		download := func(slot string) bool {
			got, contents, err := client.DownloadPetData(ctx, key)
			if err != nil {
				fail(err)
				return false
			}
			var save struct {
				PetData struct {
					ID   string `json:"id"`
					Name string `json:"name"`
				} `json:"pet_data"`
			}
			json.Unmarshal(contents, &save)
			if save.PetData.Name != "" {
				name = save.PetData.Name
			}
			if slot == "" {
				slot = s.store.NewSlotFor(save.PetData.ID)
			}
			hash, err := s.store.Import(slot, contents)
			if err != nil {
				fail(err)
				return false
			}
			c.state.Pets[key] = &cloudPet{Slot: slot, Version: got.Version, PetHash: hash}
			if slot == s.slot {
				reloadActive = true
			}
			return true
		}

		switch {
		case state != nil && state.Deleted:
			continue // Deleted on another device; no longer synced

		case isRemote && info.Deleted:
			if isLocal && (state == nil || state.Version < info.Version) {
				report.DeletedElsewhere = append(report.DeletedElsewhere, name)
				c.state.Pets[key] = &cloudPet{Slot: pet.Slot, Version: info.Version, Deleted: true}
			}

		case !isRemote:
			if upload(0) {
				report.Uploaded = append(report.Uploaded, name)
			}

		case !isLocal:
			if download("") {
				report.Downloaded = append(report.Downloaded, name)
			}

		default:
			saved, err := s.store.Export(pet.Slot)
			if err != nil {
				fail(err)
				continue
			}
			changedHere := state == nil || saved.PetHash != state.PetHash
			changedThere := state == nil || info.Version != state.Version
			switch {
			case changedHere && !changedThere:
				if upload(state.Version) {
					report.Uploaded = append(report.Uploaded, name)
				}
			case changedThere && !changedHere:
				if download(pet.Slot) {
					report.Downloaded = append(report.Downloaded, name)
				}
			case changedHere && changedThere:
				// Changed on both sides: the copy played most recently wins
				var ok bool
				if info.SavedAt.After(saved.LastPlayed) {
					ok = download(pet.Slot)
				} else {
					ok = upload(info.Version)
				}
				if ok {
					report.Conflicts = append(report.Conflicts, name)
				}
			}
		}
	}

	if s.house != nil {
		if err := s.syncHousehold(ctx, client, householdInfo); err != nil {
			report.Errors = append(report.Errors, "Home Stars: "+err.Error())
		}
	}

	c.state.LastSync = time.Now()
	c.state.LastReport = report
	if err := c.saveLocked(); err != nil {
		return report, err
	}

	if reloadActive {
		s.mu.Lock()
		if pet, _, _, err := s.store.Load(s.slot); err == nil {
			s.pet = pet
			log.Printf("Reloaded %s from the cloud", pet.Name)
		} else {
			report.Errors = append(report.Errors, "reloading the active pet: "+err.Error())
		}
		s.mu.Unlock()
	}
	return report, nil
}

// cloudForget is called when a pet is deleted: the hub marks it deleted so
// other devices stop syncing it. Errors are logged; the pet is already
// gone locally.
func (s *Server) cloudForget(ctx context.Context, petID string) {
	if s.cloud == nil {
		return
	}
	c := s.cloud
	c.mu.Lock()
	defer c.mu.Unlock()
	key := hub.PetKey(petID)
	state, ok := c.state.Pets[key]
	if !c.connectedLocked() || !ok || state.Deleted {
		return
	}
	if _, err := c.clientLocked().DeletePetData(ctx, key, state.Version); err != nil {
		log.Printf("Could not delete the pet from the cloud hub: %v", err)
		return
	}
	delete(c.state.Pets, key)
	if err := c.saveLocked(); err != nil {
		log.Printf("Could not save cloud.json: %v", err)
	}
}

// Cloud endpoints

// cloudTimeout is how long a cloud request from the page may take
const cloudTimeout = 30 * time.Second

// handleCloud handles GET /api/cloud - the cloud connection
func (s *Server) handleCloud(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, APIResponse{Success: false, Error: "Method not allowed"})
		return
	}
	writeJSON(w, http.StatusOK, APIResponse{Success: true, Data: s.cloudStatus()})
}

// writeCloudError reports a cloud failure
func writeCloudError(w http.ResponseWriter, err error) {
	status := http.StatusBadGateway // The hub failed
	switch {
	case errors.Is(err, errNotConnected):
		status = http.StatusConflict
	case errors.Is(err, hub.ErrUnauthorized):
		err = errors.New("the hub doesn't recognize this account any more; disconnect and connect again")
	}
	writeJSON(w, status, APIResponse{Success: false, Error: err.Error()})
}

// handleCloudConnect handles POST /api/cloud/connect with {"hub_url",
// "join_code"} or {"hub_url", "account_key"}: registers with a hub (or
// signs in), then syncs
func (s *Server) handleCloudConnect(w http.ResponseWriter, r *http.Request) {
	var req struct {
		HubURL     string `json:"hub_url"`
		JoinCode   string `json:"join_code"`
		AccountKey string `json:"account_key"`
		ParentPIN  string `json:"parent_pin"`
	}
	if !onlyThisComputer(w, r) || !decodePost(w, r, &req) || !s.parentAllows(w, req.ParentPIN) {
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), cloudTimeout)
	defer cancel()
	if err := s.cloudConnect(ctx, req.HubURL, req.JoinCode, req.AccountKey); err != nil {
		writeJSON(w, http.StatusBadRequest, APIResponse{Success: false, Error: err.Error()})
		return
	}
	report, err := s.cloudSync(ctx)
	if err != nil {
		writeCloudError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, APIResponse{Success: true, Data: map[string]interface{}{
		"status": s.cloudStatus(), "report": report, "text": report.summary()}})
}

// handleCloudDisconnect handles POST /api/cloud/disconnect
func (s *Server) handleCloudDisconnect(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, APIResponse{Success: false, Error: "Method not allowed"})
		return
	}
	if !onlyThisComputer(w, r) {
		return
	}
	var req struct {
		ParentPIN string `json:"parent_pin"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil && !errors.Is(err, io.EOF) { // The body is optional
		writeJSON(w, http.StatusBadRequest, APIResponse{Success: false, Error: "Invalid request"})
		return
	}
	if !s.parentAllows(w, req.ParentPIN) {
		return
	}
	if err := s.cloudDisconnect(); err != nil {
		writeJSON(w, http.StatusInternalServerError, APIResponse{Success: false, Error: err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, APIResponse{Success: true, Data: s.cloudStatus()})
}

// handleCloudSync handles POST /api/cloud/sync - syncs now
func (s *Server) handleCloudSync(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, APIResponse{Success: false, Error: "Method not allowed"})
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), cloudTimeout)
	defer cancel()
	report, err := s.cloudSync(ctx)
	if err != nil {
		writeCloudError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, APIResponse{Success: true, Data: map[string]interface{}{
		"report": report, "text": report.summary()}})
}

// handleCloudProfile handles POST /api/cloud/profile with
// {"share_achievements"}
func (s *Server) handleCloudProfile(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ShareAchievements bool   `json:"share_achievements"`
		ParentPIN         string `json:"parent_pin"`
	}
	if !onlyThisComputer(w, r) || !decodePost(w, r, &req) || !s.parentAllows(w, req.ParentPIN) {
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), cloudTimeout)
	defer cancel()
	if err := s.cloudSetProfile(ctx, req.ShareAchievements); err != nil {
		writeCloudError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, APIResponse{Success: true, Data: s.cloudStatus()})
}

// handleCloudKey handles POST /api/cloud/key (with "parent_pin" if a PIN
// is set): the account key, for connecting the player's other devices.
// Only the computer running the game is given it.
func (s *Server) handleCloudKey(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ParentPIN string `json:"parent_pin"`
	}
	if !fromThisComputer(r) {
		writeJSON(w, http.StatusForbidden, APIResponse{Success: false,
			Error: "The account key is shown only on the computer running the game"})
		return
	}
	if !decodePost(w, r, &req) || !s.parentAllows(w, req.ParentPIN) {
		return
	}
	c := s.cloud
	if c == nil {
		writeCloudError(w, errNotConnected)
		return
	}
	c.mu.Lock()
	key, connected := c.state.Token, c.connectedLocked()
	c.mu.Unlock()
	if !connected {
		writeCloudError(w, errNotConnected)
		return
	}
	writeJSON(w, http.StatusOK, APIResponse{Success: true, Data: map[string]string{"account_key": key}})
}

// handleCloudFeed handles GET /api/cloud/feed
func (s *Server) handleCloudFeed(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, APIResponse{Success: false, Error: "Method not allowed"})
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), cloudTimeout)
	defer cancel()
	feed, err := s.cloudFeed(ctx)
	if err != nil {
		writeCloudError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, APIResponse{Success: true, Data: feed})
}
