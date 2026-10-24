package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/Michael-W-Ellison/gochi/internal/ai"
	"github.com/Michael-W-Ellison/gochi/internal/ar"
	"github.com/Michael-W-Ellison/gochi/internal/core"
	"github.com/Michael-W-Ellison/gochi/internal/hub"
	"github.com/Michael-W-Ellison/gochi/internal/simulation"
	"github.com/Michael-W-Ellison/gochi/pkg/types"
	"github.com/Michael-W-Ellison/gochi/web"
)

// Server holds the server state
type Server struct {
	// Held for reading by every request and background step, and for
	// writing while the player switches accounts (accounts.go); acquire
	// before every other lock
	accountMu sync.RWMutex
	accounts  *accountRegistry // The players on this device; nil for one player, without accounts

	saveMu sync.Mutex // Orders saves and pet switches; acquire before mu
	mu     sync.RWMutex
	pet    *core.DigitalPet        // The active pet; only its time passes
	slot   string                  // Save slot of the active pet
	static fs.FS                   // The page's static files
	clock  *simulation.TimeManager // Game clock; its scale sets the game's speed
	store  *petStore               // nil disables saving
	cloud  *cloud                  // Connection to a cloud hub; nil disables cloud features
	shares chan sharedAchievement  // Achievements to share through the hub, from the update loop; nil to not share
	hub    *hub.Hub                // This server's own hub, if it runs one
	events *core.EventSystem       // What happened to the pets while time passed

	// When the player is usually around, learned from when they use the
	// page; nil disables it
	owner     *ai.OwnerSchedule
	ownerPath string // Where owner is saved; "" to not save it

	// The player's own statistics, kept on this computer only; nil
	// disables them
	stats *playerStats

	// The household: Home Stars, house upgrades and settings; nil
	// disables them. holidayBought is set once a holiday item has been
	// bought this session (one per session).
	house         *household
	holidayBought bool

	// Messages the page shows in front of everything until dismissed
	popups []popup

	// When the household achievements were last checked
	lastHouseholdAwards time.Time

	// Game days played since the saved pets were last brought up to date,
	// and when that was (householdtime.go)
	householdDays     float64
	lastHouseholdTime time.Time

	// The AR playdate the player is at, through the cloud hub
	playdates playdateState

	// The parent PIN, if one is set; nil means none
	parental *parentalControls

	// Devices other than this computer that may use the game, and where
	// the server listens (see guard.go)
	pairing    *devicePairing
	listenAddr string
	onNetwork  bool
	usesTLS    bool

	// A hash of the static files, added to their URLs so they can be
	// cached until they change
	staticVersion string

	// Time stops while the game is paused, so the pet cannot decline
	manualPause  bool          // Paused with the Pause button
	lastActivity time.Time     // Last time the player was present
	awayTimeout  time.Duration // Pause after this long without the player (0 disables)

	// Augmented reality state, for AR clients. Never held together with
	// mu.
	arMu      sync.Mutex
	arService *ar.ARService // Created on first use
	arTouches []arTouch     // Pet interactions from the gesture being processed
}

// Pause reasons reported by /api/status
const (
	pauseReasonManual = "manual" // Player pressed Pause
	pauseReasonAway   = "away"   // Player has not been present for awayTimeout
)

// APIResponse is a standard API response wrapper
type APIResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

// PetStatus represents the pet's current state for the frontend
type PetStatus struct {
	ID       types.PetID      `json:"id"`
	Name     string           `json:"name"`
	Age      float64          `json:"age"`
	IsAlive  bool             `json:"is_alive"`
	Behavior string           `json:"behavior"`
	Activity string           `json:"activity"` // What the pet is doing, e.g. "is playing"
	Request  *core.PetRequest `json:"request"`  // What the pet is asking for, or null
	Species  core.Species     `json:"species"`

	// Health care: organ systems, the living area, the pet's illness ("" if
	// well), its warning signs, the care guide topics they relate to, and
	// today's care checklist
	Body        []core.BodySystem    `json:"body"`
	LivingArea  core.LivingArea      `json:"living_area"`
	Illness     string               `json:"illness"`
	Symptoms    []core.Symptom       `json:"symptoms"`
	GuideTopics []string             `json:"guide_topics"`
	Checklist   []core.ChecklistItem `json:"checklist"`

	// Today's goals and their difficulty (1 to 5), what the pet will soon
	// need, and when the player usually plays ("" until it is known)
	Goals        []core.Goal      `json:"goals"`
	Weekly       core.WeeklyGoal  `json:"weekly"`
	Titles       []string         `json:"titles"`
	Shelf        []core.Souvenir  `json:"shelf"`
	Happening    *core.Happening  `json:"happening"`
	TipsSeen     []string         `json:"tips_seen"`
	Theme        core.Theme       `json:"theme"`     // This week's care theme
	SoundOff     bool             `json:"sound_off"` // A parent has turned the game's sounds off
	GoalLevel    float64          `json:"goal_level"`
	Forecasts    []forecastStatus `json:"forecasts"`
	ThingsToDo   []core.ThingToDo `json:"things_to_do"` // Activities, those ready now first
	PlaySchedule string           `json:"play_schedule"`

	Tricks           []core.TrickProgress   `json:"tricks"`
	Exploring        bool                   `json:"exploring"`
	ExploringSeconds int                    `json:"exploring_seconds"` // Real seconds until home
	HideAndSeek      bool                   `json:"hide_and_seek"`     // A game is in progress
	Souvenirs        []core.SouvenirCount   `json:"souvenirs"`
	Needs            []core.NeedLevel       `json:"needs"` // Needs not shown as stat bars
	Environment      core.EnvironmentStatus `json:"environment"`
	CareTips         map[string]string      `json:"care_tips"` // What each care button does
	Generation       int                    `json:"generation"`
	LastAdventure    *core.Adventure        `json:"last_adventure"` // Most recent trip, if it ended recently
	Mood             string                 `json:"mood"`
	Health           float64                `json:"health"`
	Energy           float64                `json:"energy"`
	Happiness        float64                `json:"happiness"`
	Hunger           float64                `json:"hunger"`
	Water            float64                `json:"water"`
	Wellbeing        float64                `json:"wellbeing"`
	Personality      string                 `json:"personality"`
	Changes          []string               `json:"personality_changes"` // How its personality has changed
	Rewarded         []string               `json:"rewarded"`            // Behaviors it has learned you reward, by mood
	Trends           []string               `json:"trends"`              // Which way its recent experiences move its personality
	Life             core.LifeStatus        `json:"life"`
	Achievements     []core.Achievement     `json:"achievements"`
	AchievementTotal int                    `json:"achievement_total"`
	LifeSeconds      int                    `json:"life_seconds"` // Real seconds of play left at the default pace
	Attention        string                 `json:"attention"`    // How much it enjoys attention by nature
	Tolerates        []string               `json:"tolerates"`    // Care it neither likes nor dislikes
	Relations        []core.PetRelation     `json:"relations"`    // Pets it knows and how they get along
	Likes            []string               `json:"likes"`
	Dislikes         []string               `json:"dislikes"`
	Slot             string                 `json:"slot"`
	Hibernating      bool                   `json:"hibernating"`
	HibernationOn    bool                   `json:"hibernation_enabled"` // Hibernate instead of dying
	CauseOfDeath     string                 `json:"cause_of_death,omitempty"`
	Paused           bool                   `json:"paused"`
	PauseReason      string                 `json:"pause_reason,omitempty"`

	// How the pet looks (cosmetics by slot) and its home looks (house
	// cosmetics by slot), upkeep that needs doing, first-aid kit uses,
	// Home Stars, and messages to show in front of everything
	Outfit       map[string]core.Cosmetic  `json:"outfit"`
	HouseLook    map[string]core.HouseLook `json:"house_look"`
	Upkeep       []core.UpkeepStatus       `json:"upkeep"`
	FirstAidUses int                       `json:"first_aid_uses"`
	Injured      bool                      `json:"injured"`
	HomeStars    int                       `json:"home_stars"`
	Popups       []popup                   `json:"popups"`

	// Its breed, how it looks and what it is prone to, and where it
	// comes from
	Breed core.BreedInfo `json:"breed"`

	FriendshipPoints int             `json:"friendship_points"`
	Items            []core.ShopItem `json:"items"`   // Owned items
	Effects          []core.ShopItem `json:"effects"` // Items whose effects currently apply
}

func main() {
	defaults := defaultConfig()
	configPath := flag.String("config", "", "settings file (default configs/config.yaml in the current folder or beside the game, if present)")
	dataDir := flag.String("data-dir", defaults.Server.DataDir, "directory for saves and backups (default ~/.gochi)")
	autosaveInterval := flag.Duration("autosave", defaults.Simulation.Autosave, "how often to save the pet automatically (0 disables)")
	awayTimeout := flag.Duration("away-pause", defaults.Simulation.AwayPause, "pause time when the player has been away this long (0 disables)")
	addr := flag.String("addr", defaults.Server.Addr, "address to listen on")
	apiToken := flag.String("api-token", defaults.Server.APIToken, "enable the REST API at /api/v1 for requests with this bearer token")
	scaleName := flag.String("time-scale", defaults.Simulation.TimeScale, "game speed: game (1 game day per real minute), 24x, 4x or realtime")
	hubEnabled := flag.Bool("hub", defaults.Hub.Enabled, "also run as a cloud hub other Gochi servers can connect to")
	hubJoinCode := flag.String("hub-join-code", defaults.Hub.JoinCode, "the code players need to register with this hub")
	hubAddr := flag.String("hub-addr", defaults.Hub.Addr, "address the hub listens on (always HTTPS)")
	tlsCert := flag.String("tls-cert", defaults.Server.TLSCert, "serve HTTPS with this certificate (PEM; needs -tls-key)")
	tlsKey := flag.String("tls-key", defaults.Server.TLSKey, "the HTTPS certificate's private key (PEM)")
	webDir := flag.String("web-dir", "", "serve the page's static files from this folder instead of the built-in ones (for working on the page)")
	flag.Parse()

	// The page's static files: built in, or from a folder
	static, builtIn := web.Static(), true
	if *webDir != "" {
		static, builtIn = os.DirFS(*webDir), false
		log.Printf("Serving static files from: %s", *webDir)
	}

	// Settings: defaults, then the config file, then command-line options
	config := defaults
	if *configPath == "" {
		*configPath = defaultConfigPath()
	}
	if *configPath != "" {
		var err error
		if config, err = loadConfig(*configPath); err != nil {
			log.Fatalf("Could not read settings: %v", err)
		}
		log.Printf("Settings from %s", *configPath)
	}
	flag.Visit(func(f *flag.Flag) {
		switch f.Name {
		case "data-dir":
			config.Server.DataDir = *dataDir
		case "autosave":
			config.Simulation.Autosave = *autosaveInterval
		case "away-pause":
			config.Simulation.AwayPause = *awayTimeout
		case "addr":
			config.Server.Addr = *addr
		case "api-token":
			config.Server.APIToken = *apiToken
		case "time-scale":
			config.Simulation.TimeScale = *scaleName
		case "hub":
			config.Hub.Enabled = *hubEnabled
		case "hub-join-code":
			config.Hub.JoinCode = *hubJoinCode
		case "hub-addr":
			config.Hub.Addr = *hubAddr
		case "tls-cert":
			config.Server.TLSCert = *tlsCert
		case "tls-key":
			config.Server.TLSKey = *tlsKey
		}
	})
	if err := config.validate(); err != nil {
		log.Fatalf("Invalid settings: %v", err)
	}
	timeScale := timeScales[config.Simulation.TimeScale]
	log.Printf("Game speed: %s", config.Simulation.TimeScale)

	// The device's files are in the data folder; each player account's
	// are in its own folder in it (accounts.go)
	dataPath := dataRoot(config.Server.DataDir)
	accounts, err := openAccounts(dataPath)
	if err != nil {
		log.Fatalf("Could not read the player accounts: %v\nTo start afresh with the files from before accounts, "+
			"move %s aside (the accounts' folders stay in %s)", err, filepath.Join(dataPath, accountsFileName),
			filepath.Join(dataPath, "accounts"))
	}
	player := accounts.current()
	parts, err := openAccountParts(accounts.dir(player.ID), config.Simulation.TimeScale,
		int(config.Simulation.Autosave.Seconds()))
	if err != nil {
		log.Fatalf("Could not open %s's files: %v", player.Name, err)
	}
	players, _ := accounts.list()
	if len(players) == 1 {
		log.Printf("Playing as %s", player.Name)
	} else {
		log.Printf("Playing as %s (%d players on this device)", player.Name, len(players))
	}
	parental, err := loadParentalControls(dataPath)
	if err != nil {
		// Not started without the PIN, so damaging the file doesn't
		// remove it
		log.Fatalf("Could not read the parent PIN: %v\nTo start without a PIN (it can be set again in Privacy), remove %s",
			err, filepath.Join(dataPath, "parental.json"))
	}
	// The holiday and sound settings were the household's before they
	// were the device's
	if err := parental.adoptHouseSettings(parts.house.snapshot()); err != nil {
		log.Printf("Warning: could not move the holiday and sound settings to %s: %v",
			filepath.Join(dataPath, "parental.json"), err)
	}
	parental.apply()
	store := parts.store
	pairing := loadOrSetAside("the paired devices", filepath.Join(dataPath, "devices.json"),
		func() (*devicePairing, error) { return loadDevicePairing(dataPath) })
	removeTempFiles(dataPath)
	var hubServer *hub.Hub
	if config.Hub.Enabled {
		if hubServer, err = hub.New(filepath.Join(dataPath, "hub"), config.Hub.JoinCode); err != nil {
			log.Fatalf("Could not set up the cloud hub: %v", err)
		}
	}

	// Create server
	server := &Server{
		static:       static,
		clock:        simulation.NewTimeManager(timeScale),
		accounts:     accounts,
		store:        store,
		lastActivity: time.Now(),
		awayTimeout:  config.Simulation.AwayPause,
		events:       newEventSystem(),
		cloud:        parts.cloud,
		shares:       make(chan sharedAchievement, shareQueue),
		hub:          hubServer,
		owner:        parts.owner,
		ownerPath:    parts.ownerPath,
		stats:        parts.stats,
		house:        parts.house,
		parental:     parental,
		pairing:      pairing,
		listenAddr:   config.Server.Addr,
		onNetwork:    listensOnNetwork(config.Server.Addr),
		usesTLS:      config.usesTLS(),
	}

	// Sync with the cloud hub first, so the newest saves are restored
	if server.cloudStatus().Connected {
		// Offline, the game starts after a short wait and syncs later
		syncCtx, cancel := context.WithTimeout(context.Background(), startupSyncTimeout)
		if report, err := server.cloudSync(syncCtx); err != nil {
			log.Printf("Cloud sync failed; playing with the saves on this device: %v", err)
		} else {
			log.Printf("Cloud sync: %s", report.summary())
		}
		cancel()
	}

	// Restore the most recently saved pet. Time does not pass while the
	// program is closed: the pet resumes exactly as it was saved.
	pet, slot, path, warning, err := store.LoadMostRecent()
	switch {
	case err == nil:
		server.pet, server.slot = pet, slot
		log.Printf("Restored %s from %s", pet.Name, path)
		if warning != nil {
			log.Printf("Warning: %v", warning)
		}
	case errors.Is(err, os.ErrNotExist):
		log.Printf("No saved pets found; pets will be saved in %s", store.SaveDir())
	default:
		log.Printf("Warning: could not restore a saved pet: %v", err)
		log.Printf("Starting without a pet; the save files have been left in place in %s", store.SaveDir())
	}
	server.recordSavedPets()
	server.scanAchievements()
	server.observePet(server.pet, false)
	if config.Simulation.AwayPause > 0 {
		log.Printf("Time pauses after %s without the player", config.Simulation.AwayPause)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Start background update and autosave loops
	go server.updateLoop()
	if config.Simulation.Autosave > 0 {
		go server.autosaveLoop(ctx, config.Simulation.Autosave)
		log.Printf("Autosaving every %s", config.Simulation.Autosave)
	}
	go server.syncLoop(ctx)
	go server.shareLoop(ctx)

	// Setup routes
	mux := http.NewServeMux()

	// The REST API from the api package, for other clients
	if config.Server.APIToken != "" {
		apiRoutes := server.apiHandler(config.Server.APIToken, config.Server.APIRateLimit)
		for _, pattern := range []string{"/api/v1", "/api/v1/", "/health", "/ready"} {
			mux.Handle(pattern, apiRoutes)
		}
		log.Printf("REST API enabled at /api/v1")
		if config.Server.APIRateLimit > 0 {
			log.Printf("REST API clients may make %d requests a minute", config.Server.APIRateLimit)
		}
	}

	// The cloud hub, for other Gochi servers, on its own HTTPS address
	var hubHTTP *http.Server
	serverErr := make(chan error, 2)
	if hubServer != nil {
		certFile, keyFile, fingerprint, err := hub.ServerCertificate(filepath.Join(dataPath, "hub"),
			config.Server.TLSCert, config.Server.TLSKey)
		if err != nil {
			log.Fatalf("Could not set up the cloud hub's certificate: %v", err)
		}
		hubMux := http.NewServeMux()
		hubMux.Handle("/hub/v1/", hubServer.Handler())
		hubHTTP = &http.Server{Addr: config.Hub.Addr, Handler: hubMux, TLSConfig: hub.TLSConfig()}
		hub.SetTimeouts(hubHTTP)
		go func() { serverErr <- hubHTTP.ListenAndServeTLS(certFile, keyFile) }()
		log.Printf("Cloud hub at https://<this computer>%s/hub/v1", config.Hub.Addr)
		log.Printf("Hub certificate fingerprint (players can compare it with the one their game shows): %s", fingerprint)
		if !hubServer.RegistrationOpen() {
			log.Printf("Registration is closed (no join code)")
		}
	}
	for _, r := range server.routes() {
		mux.HandleFunc(r.pattern, r.handler)
	}

	// Static files, cacheable by browsers and proxies (see staticHandler)
	if server.staticVersion, err = staticVersion(static); err != nil {
		log.Printf("Warning: could not read the static files: %v", err)
	}
	mux.Handle("/static/", server.staticHandler(static, builtIn))

	// Templates/pages
	mux.HandleFunc("/", server.handleIndex)

	// Start server
	httpServer := &http.Server{Addr: config.Server.Addr, Handler: server.guard(server.accountGate(securityHeaders(mux))),
		TLSConfig:         tlsConfig(),
		ReadHeaderTimeout: 10 * time.Second, ReadTimeout: 30 * time.Second, WriteTimeout: 60 * time.Second,
		IdleTimeout: 2 * time.Minute}
	scheme := "http"
	go func() {
		if config.usesTLS() {
			serverErr <- httpServer.ListenAndServeTLS(config.Server.TLSCert, config.Server.TLSKey)
		} else {
			serverErr <- httpServer.ListenAndServe()
		}
	}()
	if config.usesTLS() {
		scheme = "https"
		log.Printf("Serving HTTPS with %s", config.Server.TLSCert)
	}
	_, port, _ := net.SplitHostPort(config.Server.Addr)
	log.Printf("Starting Gochi server on %s", config.Server.Addr)
	log.Printf("Open your browser to %s://localhost:%s to play! Press Ctrl+C to save and exit.", scheme, port)
	if server.onNetwork {
		log.Printf("Other devices on the network can play after pairing (🔒 Privacy on this computer shows the code)")
	}

	select {
	case err := <-serverErr:
		log.Fatal(err)
	case <-ctx.Done():
		stop() // A second Ctrl+C exits immediately
	}

	// Save and exit
	log.Printf("Shutting down...")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		log.Printf("Warning: HTTP shutdown: %v", err)
	}
	if hubHTTP != nil {
		if err := hubHTTP.Shutdown(shutdownCtx); err != nil {
			log.Printf("Warning: hub shutdown: %v", err)
		}
	}

	// The loops keep running; no switch can start once the page is shut
	// down, so the account stays the one being played
	server.accountMu.RLock()
	defer server.accountMu.RUnlock()
	if err := server.saveOwner(); err != nil {
		log.Printf("Could not save the play schedule: %v", err)
	}
	server.mu.RLock()
	server.observePet(server.pet, false)
	server.mu.RUnlock()
	if err := server.stats.save(); err != nil {
		log.Printf("Could not save your statistics: %v", err)
	}
	if err := server.house.save(); err != nil {
		log.Printf("Could not save the household: %v", err)
	}
	if saved, err := server.saveNow(); err != nil {
		log.Printf("Failed to save pet on exit: %v", err)
		os.Exit(1)
	} else if saved {
		log.Printf("Saved pet to %s", server.store.SavePath(server.slot))
	}
	if server.cloudStatus().Connected {
		syncCtx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		if report, err := server.cloudSync(syncCtx); err != nil {
			log.Printf("Cloud sync on exit failed: %v", err)
		} else {
			log.Printf("Cloud sync: %s", report.summary())
		}
		cancel()
	}
}

func (s *Server) updateLoop() {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for now := range ticker.C {
		s.safeTick(now)
	}
}

// safeTick runs tick, and if the pet's simulation fails (a damaged save
// that loading did not catch) logs it instead of stopping the game
func (s *Server) safeTick(now time.Time) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Error advancing the pet: %v", r)
		}
	}()
	s.accountMu.RLock()
	defer s.accountMu.RUnlock()
	s.tick(now)
	s.shareTripWithFriend()
	s.householdTime(now)
}

// tick advances the active pet to now, unless the game is paused
func (s *Server) tick(now time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Goals completed by care since the last tick
	if s.pet != nil {
		s.announceGoalsLocked()
	}

	// No time passes while paused; resuming continues from now
	paused := s.pet == nil || !s.pet.IsAlive() || s.pauseReasonLocked(now) != ""
	s.clock.SetPaused(paused, now)
	gameSeconds := s.clock.UpdateAt(now)
	s.houseTickLocked(now, !paused, gameSeconds)
	if paused {
		return
	}
	s.householdDays += gameSeconds / secondsPerGameDay

	// The pet's simulation counts time in game days
	before := s.pet.Conditions()
	s.pet.Update(gameSeconds / secondsPerGameDay)
	for _, event := range core.ConditionEvents(before, s.pet.Conditions()) {
		s.events.Emit(event)
		s.pet.RecordEvent(event)
		if event.Type == core.EventTripReturned {
			s.starsForTripLocked()
		}
	}
	s.householdAwardsLocked(now)
	s.checkHappeningLocked(now)
	s.announceAchievementsLocked()
	s.observePet(s.pet, false)
	s.checkGrowingUpLocked()
}

// announceAchievementsLocked records achievements the active pet has newly
// earned as events (and shares them through the cloud hub, if the player
// has chosen to). Caller must hold s.mu.
func (s *Server) announceAchievementsLocked() {
	for _, a := range s.pet.CheckAchievements() {
		s.events.Emit(core.GameEvent{Type: core.EventAchievement, PetID: s.pet.ID, Good: true,
			Message: fmt.Sprintf("🏆 %s earned an achievement: %s %s (%s).", s.pet.Name, a.Icon, a.Name,
				strings.ToLower(a.Description[:1])+a.Description[1:])})
		s.shareAchievementLocked(a)
		s.starsForAchievementLocked(a)
	}
}

// newEventSystem creates the server's event system, which also logs the
// most serious events
func newEventSystem() *core.EventSystem {
	events := core.NewEventSystem()
	logEvent := func(event core.GameEvent) { log.Print(event.Message) }
	for _, eventType := range []core.EventType{core.EventPetDied, core.EventPetHibernated, core.EventPetSick} {
		events.RegisterHandler(eventType, logEvent)
	}
	return events
}

// careActions maps the page's care buttons to interactions
var careActions = map[string]types.InteractionType{
	"feed":  types.InteractionFeeding,
	"drink": types.InteractionDrinking,
	"pet":   types.InteractionPetting,
	"play":  types.InteractionPlaying,
	"groom": types.InteractionGrooming,
	"treat": types.InteractionRewards,
	"medic": types.InteractionMedicalCare,
	"clean": types.InteractionCleaning,
	"walk":  types.InteractionWalking,
}

// careTips describes what each care button does
func careTips() map[string]string {
	tips := make(map[string]string, len(careActions))
	for action, interactionType := range careActions {
		tips[action] = core.CareDescription(interactionType)
	}
	return tips
}

// secondsPerGameDay converts the clock's game seconds to game days
const secondsPerGameDay = 24 * 60 * 60

// timeScales are the -time-scale choices
var timeScales = map[string]types.TimeScale{
	"game":     types.TimeScaleGame,
	"24x":      types.TimeScaleAccelerated24X,
	"4x":       types.TimeScaleAccelerated4X,
	"realtime": types.TimeScaleRealTime,
}

// startupSyncTimeout is how long the program waits for a cloud sync
// before restoring the last pet
const startupSyncTimeout = 15 * time.Second

// pauseReasonLocked returns why time is paused, or "" if it is running.
// Caller must hold s.mu.
func (s *Server) pauseReasonLocked(now time.Time) string {
	if s.manualPause {
		return pauseReasonManual
	}
	if s.awayTimeout > 0 && now.Sub(s.lastActivity) > s.awayTimeout {
		return pauseReasonAway
	}
	return ""
}

// markActiveLocked records that the player is present, resuming time if it
// was paused because they were away. Caller must hold s.mu for writing.
func (s *Server) markActiveLocked(now time.Time) {
	s.lastActivity = now
	if s.owner != nil {
		s.owner.RecordPresence(now)
	}
}

func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	// Always serve the simple working page
	s.serveDefaultPage(w)
}

func (s *Server) serveDefaultPage(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	html := `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Gochi - Virtual Pet</title>
    <link rel="stylesheet" href="/static/css/ar.css?v={{static}}">
    <style>
        * { box-sizing: border-box; margin: 0; padding: 0; }
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            min-height: 100vh;
            display: flex;
            justify-content: center;
            align-items: center;
            padding: 20px;
        }
        .container {
            background: white;
            border-radius: 20px;
            padding: 30px;
            max-width: 400px;
            width: 100%;
            box-shadow: 0 20px 60px rgba(0,0,0,0.3);
        }
        h1 { text-align: center; color: #333; margin-bottom: 10px; }
        .subtitle { text-align: center; color: #666; margin-bottom: 25px; }
        .pet-name { text-align: center; color: #764ba2; font-size: 1.5em; margin-bottom: 20px; }
        .pet-display {
            text-align: center;
            font-size: 80px;
            margin: 20px 0;
            animation: bounce 1s ease infinite;
        }
        /* Animations stop on low battery and for players who prefer less motion */
        body.power-saving *, body.power-saving *::before, body.power-saving *::after { animation: none !important; transition: none !important; }
        @media (prefers-reduced-motion: reduce) {
            .pet-display { animation: none; }
        }
        @keyframes bounce {
            0%, 100% { transform: translateY(0); }
            50% { transform: translateY(-10px); }
        }
        .stats { margin: 20px 0; }
        .stat {
            display: flex;
            justify-content: space-between;
            align-items: center;
            margin: 10px 0;
        }
        .stat-label { font-weight: 500; color: #555; }
        .stat-bar {
            width: 60%;
            height: 20px;
            background: #eee;
            border-radius: 10px;
            overflow: hidden;
        }
        .stat-fill {
            height: 100%;
            border-radius: 10px;
            transition: width 0.3s ease;
        }
        .stat-fill.health { background: linear-gradient(90deg, #ff6b6b, #ee5a5a); }
        .stat-fill.energy { background: linear-gradient(90deg, #ffd93d, #f9c74f); }
        .stat-fill.happiness { background: linear-gradient(90deg, #6bcb77, #4caf50); }
        .stat-fill.hunger { background: linear-gradient(90deg, #ff9f43, #ee8500); }
        .stat-fill.water { background: linear-gradient(90deg, #74c0fc, #339af0); }
        .mood { text-align: center; font-size: 1.2em; color: #666; margin: 15px 0; }
        .actions {
            display: grid;
            grid-template-columns: repeat(2, 1fr);
            gap: 10px;
            margin-top: 20px;
        }
        .action-btn {
            padding: 15px;
            border: none;
            border-radius: 12px;
            font-size: 16px;
            font-weight: 600;
            cursor: pointer;
            transition: transform 0.1s, box-shadow 0.1s;
            display: flex;
            align-items: center;
            justify-content: center;
            gap: 8px;
        }
        .action-btn:hover { transform: translateY(-2px); box-shadow: 0 5px 20px rgba(0,0,0,0.2); }
        .action-btn:active { transform: translateY(0); }
        .action-btn.feed { background: #ffd93d; color: #333; }
        .action-btn.pet { background: #ff6b9d; color: white; }
        .action-btn.play { background: #6bcb77; color: white; }
        .action-btn.groom { background: #74b9ff; color: white; }
        .action-btn.drink { background: #4dabf7; color: white; }
        .action-btn.medic { background: #ffa8a8; color: #862e2e; }
        .action-btn.clean { background: #b2f2bb; color: #2b8a3e; }
        .action-btn.walk { background: #ffd8a8; color: #a14c00; }
        .actions .action-btn:last-child:nth-child(odd) { grid-column: span 2; }
        .speech { position: relative; margin: 0 auto 6px; max-width: 280px; padding: 8px 14px; border-radius: 16px; background: #fff3bf; color: #5c4400; font-size: 0.95em; text-align: center; }
        .speech::after { content: ''; position: absolute; left: 50%; bottom: -8px; margin-left: -8px; border: 8px solid transparent; border-top-color: #fff3bf; border-bottom: 0; }
        @keyframes nudge { 0%, 100% { box-shadow: 0 0 0 0 rgba(255, 193, 7, 0.9); } 50% { box-shadow: 0 0 0 6px rgba(255, 193, 7, 0); } }
        .requested { animation: nudge 1.2s ease-in-out infinite; outline: 3px solid #ffc107; }
        .activities { display: grid; grid-template-columns: repeat(2, 1fr); gap: 8px; margin-top: 10px; }
        .activity-btn { padding: 10px 6px; border: 2px solid #b197fc; border-radius: 12px; background: #f3f0ff; color: #5f3dc4; font-weight: 600; font-size: 0.9em; cursor: pointer; }
        .activity-btn:disabled { opacity: 0.4; cursor: not-allowed; }
        .activity-panel { margin-top: 10px; padding: 12px; border-radius: 12px; background: #f8f9fa; border: 2px solid #e9ecef; }
        .panel-head { display: flex; justify-content: space-between; align-items: center; margin-bottom: 8px; font-weight: 600; color: #333; }
        .panel-close { background: none; border: none; font-size: 1.2em; cursor: pointer; color: #888; }
        .panel-text { text-align: center; color: #444; margin-bottom: 10px; }
        .trick-row { display: flex; flex-wrap: wrap; align-items: center; gap: 8px; margin-bottom: 8px; }
        .trick-name { width: 120px; font-size: 0.9em; }
        .trick-bar { flex: 1; height: 8px; background: #e9ecef; border-radius: 4px; overflow: hidden; }
        .trick-fill { height: 100%; background: linear-gradient(90deg, #9775fa, #7048e8); }
        .spots { display: grid; grid-template-columns: repeat(2, 1fr); gap: 8px; }
        .spot-btn { padding: 10px; border: 2px solid #dee2e6; border-radius: 12px; background: white; cursor: pointer; font-size: 0.9em; }
        .spot-btn:disabled { opacity: 0.4; cursor: not-allowed; }
        .explore-banner, .adventure-banner { text-align: center; margin: 10px 0; padding: 12px; border-radius: 12px; font-size: 0.9em; }
        .explore-banner { background: #e7f5ff; color: #1864ab; }
        .adventure-banner { background: #ebfbee; color: #2b8a3e; }
        .adventure-banner.hurt { background: #fff5f5; color: #c92a2a; }
        .adventure-banner.plain { background: #f8f9fa; color: #495057; }
        .explore-banner button { display: block; margin: 8px auto 0; }
        .adventure-banner button { display: block; margin: 8px auto 0; }
        .setting { display: flex; align-items: flex-start; gap: 8px; margin: 12px 0 0; font-size: 0.85em; color: #555; text-align: left; cursor: pointer; }
        .setting input { margin-top: 2px; }
        .species-choice { display: flex; gap: 10px; justify-content: center; margin: 4px 0 10px; }
        .species-choice label { display: flex; align-items: center; gap: 6px; padding: 6px 14px; border: 2px solid #ddd; border-radius: 12px; cursor: pointer; }
        .species-choice input:checked + span { font-weight: 700; color: #5f3dc4; }
        .health-panel { margin-top: 14px; padding: 10px 12px; border-radius: 12px; background: #f8f9fa; border: 2px solid #e9ecef; }
        .section-title { font-weight: 700; color: #333; margin-bottom: 6px; font-size: 0.95em; }
        .body-systems { display: grid; grid-template-columns: repeat(4, 1fr); gap: 6px; }
        .living-area { display: flex; align-items: center; gap: 8px; margin-top: 8px; font-size: 0.85em; color: #555; }
        .living-area .stat-bar { flex: 1; height: 10px; }
        .stat-fill.area { background: linear-gradient(90deg, #74c0fc, #4dabf7); }
        .stat-fill.area.low { background: #fab005; }
        .stat-fill.area.critical { background: #fa5252; }
        .illness { margin-top: 8px; font-weight: 700; color: #c92a2a; }
        .life-line, .rhythm-line { margin-top: 8px; font-size: 0.85em; color: #555; }
        .cloud-field { display: block; margin-top: 8px; font-size: 0.85em; color: #555; }
        .cloud-field input { display: block; width: 100%; padding: 7px 9px; margin-top: 3px; border: 2px solid #ddd; border-radius: 10px; }
        .cloud-note { color: #777; font-size: 0.85em; }
        .stats-table { width: 100%; border-collapse: collapse; margin-bottom: 12px; font-size: 0.9em; }
        .stats-table td { padding: 3px 6px; border-bottom: 1px solid #eee; color: #444; }
        .stats-table td:last-child { text-align: right; font-weight: 600; }
        .cloud-key { word-break: break-all; font-family: monospace; font-size: 0.8em; }
        .rhythm-line { color: #777; }
        .potion-btn { margin-top: 8px; }
        .symptom { margin-top: 8px; font-size: 0.88em; color: #5c3c00; background: #fff4e6; border-radius: 10px; padding: 6px 10px; }
        .symptom-hint { color: #666; margin-top: 2px; }
        .symptom.good { color: #2b8a3e; background: #ebfbee; }
        .link-btn { border: none; background: none; color: #5f3dc4; text-decoration: underline; cursor: pointer; padding: 0; font-size: 1em; }
        .checklist { margin-top: 10px; padding: 10px 12px; border-radius: 12px; background: #ebfbee; border: 2px solid #d3f9d8; font-size: 0.9em; }
        .checklist-item { margin-top: 3px; color: #2b8a3e; }
        .checklist-item.todo { color: #555; }
        .goals { margin-top: 10px; padding: 10px 12px; border-radius: 12px; background: #fff9db; border: 2px solid #ffec99; font-size: 0.9em; }
        .goal-item { margin-top: 3px; color: #555; }
        .goal-item.done { color: #2b8a3e; }
        .goal-reward { color: #e67700; font-size: 0.85em; white-space: nowrap; }
        .forecast { margin-top: 10px; padding: 10px 12px; border-radius: 12px; background: #e7f5ff; border: 2px solid #d0ebff; font-size: 0.9em; color: #1864ab; }
        .forecast-item { margin-top: 3px; }
        .play-schedule { margin-top: 4px; color: #555; font-size: 0.85em; }
        .things-to-do { margin-top: 10px; padding: 10px 12px; border-radius: 12px; background: #f3f0ff; border: 2px solid #e5dbff; font-size: 0.9em; }
        .thing-item { display: flex; gap: 8px; align-items: baseline; margin-top: 5px; }
        .thing-btn { flex: none; border: 2px solid #b197fc; background: #fff; color: #5f3dc4; border-radius: 10px; padding: 3px 8px; font-weight: 600; cursor: pointer; }
        .thing-name { flex: none; color: #868e96; font-weight: 600; padding: 0 10px; }
        .thing-why { color: #555; }
        .thing-item.waiting .thing-why { color: #868e96; }
        .event-log { margin-top: 10px; padding: 10px 12px; border-radius: 12px; background: #f8f9fa; border: 2px solid #e9ecef; font-size: 0.88em; }
        .event-log-item { margin-top: 3px; color: #5c3c00; }
        .event-log-item.good { color: #2b8a3e; }
        .event-log-time { color: #868e96; margin-right: 6px; }
        .guide-topic { margin-top: 6px; border-radius: 10px; background: #fff; border: 1px solid #e9ecef; padding: 6px 10px; }
        .guide-topic summary { cursor: pointer; font-weight: 600; color: #333; }
        .guide-topic.relevant { border-color: #fab005; background: #fff9db; }
        .guide-topic p { margin: 6px 0 2px; color: #444; font-size: 0.92em; line-height: 1.4; }
        .points-toast .lesson { color: #1864ab; }
        .points-toast .repeated { color: #868e96; font-size: 0.95em; }
        .needs { display: grid; grid-template-columns: repeat(5, 1fr); gap: 6px; margin-top: 10px; }
        .need { text-align: center; font-size: 0.7em; color: #666; }
        .need-bar { height: 6px; margin-top: 3px; background: #e9ecef; border-radius: 3px; overflow: hidden; }
        .need-fill { height: 100%; background: #20c997; }
        .need-fill.low { background: #fab005; }
        .need-fill.critical { background: #fa5252; }
        .about { margin-top: 16px; padding: 10px 12px; border-radius: 12px; background: #f8f9fa; font-size: 0.85em; color: #555; }
        .about div + div { margin-top: 4px; }
        .about-label { font-weight: 600; color: #333; }
        .environment { text-align: center; color: #777; font-size: 0.85em; margin-bottom: 4px; }
        .activity { text-align: center; color: #555; font-size: 1.05em; margin-top: -8px; margin-bottom: 4px; }
        .friendship { text-align: center; color: #d6336c; font-weight: 600; margin-top: -12px; margin-bottom: 6px; }
        .belongings { text-align: center; font-size: 1.6em; min-height: 1.2em; letter-spacing: 4px; }
        .points-toast { text-align: center; min-height: 1.3em; margin-top: 10px; font-size: 0.9em; color: #d6336c; }
        .points-toast .changes { color: #444; font-weight: 600; }
        .points-toast .reaction { color: #333; font-size: 1.05em; }
        .points-toast .boost { color: #2b8a3e; }
        .effects { margin: 6px auto 0; font-size: 0.8em; color: #2b8a3e; text-align: center; }
        .effects-title { color: #777; font-weight: 600; }
        .shop-header { text-align: center; margin-bottom: 16px; }
        .shop-header .friendship { margin: 6px 0 0; }
        .shop-category { margin: 14px 0 6px; color: #555; font-size: 0.95em; font-weight: 600; }
        .shop-item { display: flex; align-items: center; gap: 10px; padding: 10px 12px; margin-bottom: 8px; border: 2px solid #eee; border-radius: 12px; }
        .shop-item.owned { border-color: #b2f2bb; background: #f4fdf5; }
        .shop-icon { font-size: 1.8em; flex-shrink: 0; }
        .shop-info { flex: 1; min-width: 0; }
        .shop-name { font-weight: bold; color: #333; }
        .shop-desc { font-size: 0.8em; color: #777; }
        .badge { display: inline-block; margin-left: 6px; padding: 0 6px; border-radius: 8px; font-size: 0.72em; font-weight: 600; vertical-align: middle; }
        .badge.common { background: #e9ecef; color: #495057; }
        .badge.uncommon { background: #d3f9d8; color: #2b8a3e; }
        .badge.rare { background: #d0ebff; color: #1864ab; }
        .badge.epic, .badge.legendary { background: #f3d9fa; color: #862e9c; }
        .badge.sale { background: #ffe3e3; color: #c92a2a; }
        .shop-item.locked { opacity: 0.6; }
        .shop-tabs { display: flex; flex-wrap: wrap; gap: 6px; margin: 4px 0 10px; }
        .shop-tab { border: 2px solid #e0e0e0; background: white; border-radius: 16px; padding: 6px 10px; cursor: pointer; font-size: 0.9em; }
        .shop-tab.active { border-color: #667eea; background: #eef0ff; font-weight: 600; }
        .occasion-banner { background: #fff8e1; border-radius: 10px; padding: 8px 10px; font-size: 0.85em; color: #6d5400; margin-bottom: 6px; }
        .shop-lesson { color: #2b8a3e; font-size: 0.82em; margin-top: 2px; }
        .badge.season { background: #fff3bf; color: #8a6d00; }
        .badge.holiday { background: #ffe8cc; color: #a34d00; }
        .cosmetic-swatch { position: relative; }
        .cosmetic-swatch[style*="--c"]::after { content: ''; position: absolute; right: 2px; bottom: 2px; width: 10px; height: 10px; border-radius: 50%; background: var(--c); border: 1px solid #fff; }
        .pet-scene { position: relative; margin: 0 auto; max-width: 320px; border-radius: 18px; padding: 6px 0 10px; }
        .pet-scene.decorated { background: linear-gradient(to bottom, transparent 70%, rgba(0,0,0,0.05) 70%), var(--wall); }
        .pet-scene.decorated::after { content: ''; position: absolute; left: 0; right: 0; bottom: 0; height: 28%; border-radius: 0 0 18px 18px; background: var(--floor); opacity: 0.9; z-index: 0; }
        .pet-figure { position: relative; display: block; z-index: 1; }
        .scene-window { position: absolute; top: 8px; right: 12px; font-size: 1.6em; z-index: 1; }
        .scene-window:not(:empty) { background: #e3f2fd; border: 3px solid #fff; border-radius: 8px; padding: 0 4px; box-shadow: 0 1px 3px rgba(0,0,0,0.15); }
        .scene-nameplate { position: relative; z-index: 1; display: inline-block; color: #fff; font-size: 0.8em; padding: 0 10px; border-radius: 6px; }
        .scene-nameplate:empty { display: none; }
        .upkeep { font-size: 0.85em; color: #2b8a3e; }
        .upkeep.due { color: #c92a2a; }
        .family-row { display: flex; flex-wrap: wrap; align-items: center; gap: 6px; margin: 6px 0; }
        .family-name { flex: 1 1 160px; }
        .scene-shelf { position: absolute; top: 10px; left: 10px; z-index: 1; font-size: 1.2em; background: #c8a27a; border-bottom: 4px solid #8d6e4c; border-radius: 4px; padding: 0 4px; letter-spacing: 2px; }
        .collection-set { border: 1px solid #eee; border-radius: 10px; padding: 6px 8px; margin: 6px 0; text-align: left; }
        .collection-set.complete { border-color: #8bc34a; background: #f4fbec; }
        .collection-items { font-size: 1.5em; letter-spacing: 4px; }
        .collection-items .missing { filter: grayscale(1); opacity: 0.25; }
        .shelf-pick { display: inline-flex; align-items: center; gap: 2px; margin: 2px 6px 2px 0; font-size: 1.1em; }
        .show-box { border: 1px solid #eee; border-radius: 10px; padding: 8px; margin: 8px 0; text-align: left; }
        .game-box { margin: 8px 0; }
        .game-meter { position: relative; height: 22px; border-radius: 11px; background: #eee; margin: 8px 0; overflow: hidden; }
        .game-zone { position: absolute; left: 55%; width: 25%; top: 0; bottom: 0; background: #b7e4b0; }
        .game-marker { position: absolute; top: 0; bottom: 0; width: 4px; margin-left: -2px; background: #667eea; }
        .game-field { min-height: 2.6em; margin: 6px 0; }
        .game-area { position: relative; height: 180px; border-radius: 10px; background: #f6f3ea; touch-action: none; overflow: hidden; }
        .game-area:focus { outline: 2px solid #667eea; }
        .game-sprite { position: absolute; font-size: 1.6em; pointer-events: none; }
        .game-tiles { display: grid; grid-template-columns: repeat(3, 1fr); gap: 6px; margin: 8px 0; }
        .game-tile { font-size: 1.5em; padding: 10px 0; border-radius: 8px; border: 1px solid #ddd; background: #fff; cursor: pointer; }
        .game-tile.wide { font-size: 0.95em; }
        .show-title { font-weight: 600; }
        .show-row { display: flex; flex-wrap: wrap; gap: 6px; align-items: center; margin-top: 6px; }
        .show-row select { padding: 4px; border-radius: 6px; border: 1px solid #ccc; max-width: 100%; }
        .show-results { text-align: left; margin: 6px 0; padding-left: 22px; }
        .show-results li.you { font-weight: 700; color: #5a4fcf; }
        .trick-locked { flex-basis: 100%; font-size: 0.8em; }
        .goal-item.weekly { border-top: 1px dashed #ddd; margin-top: 4px; padding-top: 4px; }
        .pet-display svg { display: block; margin: 0 auto; }
        #petSprite { width: fit-content; margin-left: auto; margin-right: auto; }
        .breed-picker { display: grid; grid-template-columns: repeat(auto-fill, minmax(84px, 1fr)); gap: 6px; margin: 12px 0 6px; }
        .breed-tier { grid-column: 1 / -1; font-weight: 600; color: #555; font-size: 0.85em; margin-top: 4px; text-align: left; }
        .breed-card { display: flex; flex-direction: column; align-items: center; gap: 2px; padding: 4px; border: 2px solid #e0e0e0; border-radius: 10px; background: #fff; cursor: pointer; font: inherit; font-size: 0.75em; color: #333; }
        .breed-card.chosen { border-color: #667eea; background: #eef0fd; }
        .breed-card.locked { opacity: 0.5; }
        .breed-art { width: 64px; height: 64px; display: flex; align-items: center; justify-content: center; }
        .breed-surprise { font-size: 40px; }
        .breed-detail { text-align: left; font-size: 0.85em; background: #fafafa; border-radius: 10px; padding: 8px 10px; margin-bottom: 10px; }
        .breed-title { font-weight: 600; margin-bottom: 4px; }
        .breed-locked { color: #8a6d00; background: #fff8e1; border-radius: 8px; padding: 4px 8px; margin-top: 6px; }
        .tendency-list { margin: 4px 0; padding-left: 18px; text-align: left; }
        .tendency-list li { margin: 3px 0; }
        .saved-pet-art { display: inline-block; width: 44px; height: 44px; vertical-align: middle; margin-right: 6px; flex: none; }
        .family-amount { width: 80px; padding: 4px 6px; border: 2px solid #ddd; border-radius: 8px; }
        .popup-icon { font-size: 2.4em; text-align: center; }
        .shop-warning { font-size: 0.8em; color: #c92a2a; }
        .choice-row { display: flex; align-items: center; gap: 8px; margin-top: 6px; }
        .choice-name { flex: 1; color: #333; }
        .choice-sub { font-size: 0.8em; color: #777; }
        .method-row { display: flex; align-items: center; gap: 8px; margin-bottom: 8px; font-size: 0.9em; color: #555; }
        .method-row select { flex: 1; padding: 4px 6px; border-radius: 8px; border: 2px solid #ddd; }
        .points-toast .note { color: #868e96; }
        .shop-msg { text-align: center; min-height: 1.3em; color: #c94a4a; font-size: 0.9em; margin-top: 8px; }
        .save-row { display: flex; flex-wrap: wrap; align-items: center; justify-content: center; gap: 8px 12px; margin-top: 16px; }
        .save-btn { padding: 8px 20px; border: 2px solid #667eea; border-radius: 20px; background: white; color: #667eea; font-weight: bold; cursor: pointer; }
        .save-btn:disabled { opacity: 0.6; cursor: default; }
        .save-status { font-size: 0.85em; color: #666; }
        .save-row .save-btn { padding: 8px 14px; }
        .pause-banner { text-align: center; margin: 10px 0; padding: 8px 12px; border-radius: 12px; background: #eef1ff; color: #4a4fa8; font-size: 0.9em; }
        .saved-pets { text-align: left; margin-bottom: 25px; }
        .saved-pets h2 { font-size: 1.1em; color: #555; margin-bottom: 10px; text-align: center; }
        .saved-pet-list { list-style: none; }
        .saved-pet-list li { display: flex; align-items: center; justify-content: space-between; gap: 10px; padding: 10px 12px; margin-bottom: 8px; border: 2px solid #eee; border-radius: 12px; }
        .saved-pet-list li.active { border-color: #667eea; }
        .saved-pet-info { min-width: 0; }
        .saved-pet-name { font-weight: bold; color: #333; overflow-wrap: anywhere; }
        .saved-pet-meta { font-size: 0.8em; color: #888; }
        .saved-pet-actions { display: flex; gap: 6px; flex-shrink: 0; }
        .players-bar { display: flex; justify-content: flex-end; margin-bottom: 6px; }
        .player-row { display: flex; flex-wrap: wrap; align-items: center; justify-content: space-between; gap: 8px; padding: 6px 0; border-bottom: 1px solid #eee; }
        .player-name { font-weight: 600; color: #333; overflow-wrap: anywhere; }
        .player-row .cloud-field { flex-basis: 100%; }
        .delete-btn { border-color: #e17070; color: #c94a4a; }
        .hibernate-banner { text-align: center; margin: 10px 0; padding: 12px; border-radius: 12px; background: #fff4e0; color: #8a5a00; font-size: 0.9em; }
        .hibernate-banner button { display: block; margin: 10px auto 0; }
        .action-btn:disabled { opacity: 0.4; cursor: not-allowed; transform: none; box-shadow: none; }
        .modal { position: fixed; inset: 0; background: rgba(0,0,0,0.45); display: flex; align-items: center; justify-content: center; padding: 16px; z-index: 10; }
        .modal-box { background: white; border-radius: 16px; padding: 24px; max-width: 360px; width: 100%; text-align: center; }
        .modal-box h3 { color: #333; margin-bottom: 10px; overflow-wrap: anywhere; }
        .modal-box p { color: #555; font-size: 0.95em; margin-bottom: 14px; overflow-wrap: anywhere; }
        .modal-box input { width: 100%; padding: 10px 12px; border: 2px solid #ddd; border-radius: 10px; font-size: 1em; margin-bottom: 14px; }
        .modal-actions { display: flex; flex-wrap: wrap; gap: 10px; justify-content: center; }
        #popupText { white-space: pre-line; }
        .tour-box { position: fixed; left: 16px; right: 16px; bottom: 16px; max-width: 420px; margin: 0 auto; z-index: 10002;
            background: #fff; border-radius: 14px; padding: 12px 14px; box-shadow: 0 6px 24px rgba(0, 0, 0, 0.25); }
        .theme-line { margin: 8px 0; padding: 8px 10px; border-radius: 10px; background: #f3f0ff; font-size: 0.9em; text-align: left; }
        .book-cover { text-align: center; }
        .book-page { display: flex; gap: 10px; align-items: center; border-top: 1px solid #eee; padding: 6px 0; text-align: left; }
        .book-family { display: flex; flex-wrap: wrap; gap: 6px; align-items: center; margin: 4px 0; }
        .book-relative { border: 1px solid #ddd; border-radius: 10px; background: #fff; padding: 2px 8px; cursor: pointer; }
        .book-memorial { margin-top: 10px; padding: 10px; border-radius: 10px; background: #f7f5ef; text-align: center; }
        .tour-highlight { outline: 3px solid #667eea; outline-offset: 4px; border-radius: 10px; }
        .danger-btn { padding: 8px 16px; border: none; border-radius: 20px; background: #d9534f; color: white; font-weight: bold; cursor: pointer; }
        .danger-btn:disabled { opacity: 0.5; cursor: not-allowed; }
        .back-btn { margin-top: 12px; background: none; border: none; color: #667eea; cursor: pointer; font-size: 0.95em; }
        .status-text { text-align: center; margin-top: 15px; color: #888; font-size: 0.9em; }
        .dead { filter: grayscale(100%); }
        .dead .pet-display { animation: none; }
        .hidden { display: none !important; }

        /* Create pet form styles */
        .create-form { text-align: center; }
        .create-form .pet-display { font-size: 100px; margin: 30px 0; }
        .form-group { margin: 20px 0; }
        .form-group label { display: block; margin-bottom: 8px; color: #555; font-weight: 500; }
        .form-group input {
            width: 100%;
            padding: 12px 15px;
            border: 2px solid #ddd;
            border-radius: 10px;
            font-size: 16px;
            transition: border-color 0.2s;
        }
        .form-group input:focus { outline: none; border-color: #764ba2; }
        /* Radio buttons and checkboxes keep their size and sit beside their text */
        .form-group .species-choice label, .form-group .setting { display: flex; margin-bottom: 0; }
        .form-group .species-choice input, .form-group .setting input { width: auto; padding: 0; margin: 0; border: none; }
        .form-group .setting input { margin-top: 2px; }
        .create-btn {
            width: 100%;
            padding: 15px;
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            color: white;
            border: none;
            border-radius: 12px;
            font-size: 18px;
            font-weight: 600;
            cursor: pointer;
            margin-top: 20px;
            transition: transform 0.1s, box-shadow 0.1s;
        }
        .create-btn:hover { transform: translateY(-2px); box-shadow: 0 5px 20px rgba(102,126,234,0.4); }
        .create-btn:active { transform: translateY(0); }
        .create-btn:disabled { opacity: 0.7; cursor: not-allowed; transform: none; }
        .error-msg { color: #ee5a5a; margin-top: 10px; font-size: 14px; }

        /* AR view (ar.css); these are the design values it expects */
        .ar-container {
            --primary: #6c5ce7; --primary-light: #a29bfe; --success: #00b894; --warning: #fdcb6e; --danger: #d63031;
            --gray-400: #ced4da; --gray-500: #adb5bd; --gray-900: #212529;
            --font-size-xs: 0.75rem; --font-size-sm: 0.875rem; --font-size-md: 1rem; --font-size-xl: 1.5rem;
            --spacing-xs: 0.25rem; --spacing-sm: 0.5rem; --spacing-md: 1rem; --spacing-lg: 1.5rem; --spacing-xl: 2rem;
            --radius-sm: 4px; --radius-md: 8px; --radius-lg: 12px; --radius-xl: 20px; --radius-full: 9999px;
            --shadow-xl: 0 20px 25px rgba(0, 0, 0, 0.15);
        }
        /* While AR is open, messages show above the AR view */
        body.ar-open .points-toast { position: fixed; left: 16px; right: 16px; top: 72px; z-index: 10001; margin: 0;
            padding: 8px 12px; border-radius: 12px; background: rgba(255, 255, 255, 0.92); }
        body.ar-open .points-toast:empty { display: none; }
    </style>
</head>
<body>
    <div class="container" id="app">
        <!-- Players on this device -->
        <div class="players-bar">
            <button class="save-btn" id="playersBtn" onclick="togglePlayers()">👤 Players</button>
        </div>
        <div class="activity-panel hidden" id="playersPanel"></div>
        <!-- Create Pet View -->
        <div id="createView" class="create-form">
            <h1>Welcome to Gochi!</h1>
            <div class="saved-pets hidden" id="savedPets">
                <h2>Your pets</h2>
                <ul class="saved-pet-list" id="savedPetList"></ul>
            </div>
            <p class="subtitle">Create a new virtual pet companion</p>
            <div class="pet-display" id="createPreview"><span style="font-size: 100px">🐣</span></div>
            <div class="form-group">
                <label for="petNameInput">What will you name your pet?</label>
                <input type="text" id="petNameInput" placeholder="Enter a name..." maxlength="24" autofocus>
                <div class="species-choice" id="speciesChoice" role="radiogroup" aria-label="Kind of pet">
                    <label><input type="radio" name="species" value="cat" checked><span>🐱 Cat</span></label>
                </div>
                <div class="breed-picker" id="breedPicker" aria-label="Breed"></div>
                <div class="breed-detail" id="breedDetail"></div>
                <label class="setting">
                    <input type="checkbox" id="hibernationInput">
                    <span>Gentle mode: hibernate instead of dying if neglected</span>
                </label>
            </div>
            <button class="create-btn" id="createBtn" onclick="createPet()">Create My Pet!</button>
            <div class="error-msg hidden" id="errorMsg"></div>
            <button class="back-btn hidden" id="backBtn" onclick="closeChooser()">Back to my pet</button>
        </div>

        <!-- Pet View -->
        <div id="petView" class="hidden">
            <h1>Gochi</h1>
            <div class="pet-name" id="petName">Loading...</div>
            <div class="friendship" id="friendship"></div>
            <div class="belongings" id="belongings"></div>
            <div class="effects hidden" id="effects"></div>
            <div class="speech hidden" id="speech"></div>
            <div class="pet-scene" id="petScene">
                <div class="scene-window" id="sceneWindow" aria-hidden="true"></div>
                <div class="scene-shelf hidden" id="sceneShelf"></div>
                <div class="pet-figure">
                    <div class="pet-display" id="petSprite"></div>
                </div>
                <div class="scene-nameplate" id="sceneNameplate"></div>
            </div>
            <div class="activity" id="activity"></div>
            <div class="environment" id="environment"></div>
            <div class="mood" id="mood">Mood: Happy</div>
            <div class="pause-banner hidden" id="pauseBanner"></div>
            <div class="activities">
                <button class="activity-btn train" onclick="openTricks()">🎓 Train</button>
                <button class="activity-btn" onclick="startHideSeek()">🙈 Hide &amp; Seek</button>
                <button class="activity-btn explore" onclick="explore()">🧭 Explore</button>
                <button class="activity-btn" onclick="openBreeding()">💞 Breed</button>
                <button class="activity-btn enrich" onclick="openChoices('enrich')">🏠 Enrich</button>
                <button class="activity-btn" onclick="openMeet()">🤝 Meet</button>
                <button class="activity-btn" onclick="openShows()">🏅 Shows</button>
                <button class="activity-btn" onclick="openCollection()">🧺 Collection</button>
                <button class="activity-btn" onclick="openGames()">🎮 Games</button>
                <button class="activity-btn ar" onclick="openAR()">📷 See in your room (AR)</button>
            </div>
            <div class="activity-panel hidden" id="activityPanel"></div>
            <div class="explore-banner hidden" id="exploreBanner">
                <div id="exploreText"></div>
                <button class="save-btn" onclick="recallPet()">Call home early</button>
            </div>
            <div class="adventure-banner hidden" id="adventureBanner">
                <div id="adventureText"></div>
                <button class="save-btn" onclick="dismissAdventure()">Yay!</button>
            </div>
            <div class="points-toast" id="pointsToast"></div>
            <div class="hibernate-banner hidden" id="hibernateBanner">
                <span id="hibernateText"></span>
                <button class="save-btn" id="reviveBtn" onclick="revivePet()">Revive</button>
            </div>

            <div class="stats">
                <div class="stat">
                    <span class="stat-label">Health</span>
                    <div class="stat-bar"><div class="stat-fill health" id="healthBar" style="width: 100%"></div></div>
                </div>
                <div class="stat">
                    <span class="stat-label">Energy</span>
                    <div class="stat-bar"><div class="stat-fill energy" id="energyBar" style="width: 100%"></div></div>
                </div>
                <div class="stat">
                    <span class="stat-label">Happiness</span>
                    <div class="stat-bar"><div class="stat-fill happiness" id="happinessBar" style="width: 100%"></div></div>
                </div>
                <div class="stat">
                    <span class="stat-label">Hunger</span>
                    <div class="stat-bar"><div class="stat-fill hunger" id="hungerBar" style="width: 50%"></div></div>
                </div>
                <div class="stat">
                    <span class="stat-label">Water</span>
                    <div class="stat-bar"><div class="stat-fill water" id="waterBar" style="width: 100%"></div></div>
                </div>
            </div>
            <div class="needs" id="needs"></div>

            <div class="actions">
                <button class="action-btn feed" onclick="openChoices('feed')">Feed ▾</button>
                <button class="action-btn drink" onclick="interact('drink')">Drink</button>
                <button class="action-btn pet" onclick="openChoices('pet')">Pet ▾</button>
                <button class="action-btn play" onclick="openChoices('play')">Play ▾</button>
                <button class="action-btn groom" onclick="interact('groom')">Groom</button>
                <button class="action-btn medic" onclick="interact('medic')">🩹 Heal</button>
                <button class="action-btn clean" id="cleanBtn" onclick="interact('clean')">🧹 Scoop litter</button>
                <button class="action-btn walk hidden" id="walkBtn" onclick="interact('walk')">🦮 Walk</button>
            </div>
            <div class="things-to-do hidden" id="thingsToDo"></div>

            <div class="health-panel">
                <div class="section-title">Body</div>
                <div class="body-systems" id="bodySystems"></div>
                <div class="life-line" id="lifeLine"></div>
                <div class="rhythm-line" id="rhythmLine"></div>
                <button class="save-btn potion-btn hidden" id="potionBtn" onclick="givePotion()">🧪 Give life-extension potion</button>
                <button class="save-btn potion-btn hidden" id="firstAidBtn" onclick="giveCare('first_aid')">🩹 Use first-aid kit</button>
                <div id="upkeepLine"></div>
                <button class="save-btn potion-btn" onclick="openFamily()">👪 Family</button>
                <div class="living-area">
                    <span id="livingAreaLabel">Litter box</span>
                    <div class="stat-bar"><div class="stat-fill area" id="livingAreaBar" style="width: 100%"></div></div>
                </div>
                <div class="illness hidden" id="illnessLine"></div>
                <div id="symptoms"></div>
            </div>
            <div class="theme-line" id="themeLine"></div>
            <div class="checklist" id="checklist"></div>
            <div class="goals hidden" id="goals"></div>
            <div class="forecast hidden" id="forecast"></div>
            <div class="event-log hidden" id="eventLog">
                <div class="section-title">Recently</div>
                <div id="eventLogItems"></div>
            </div>

            <div class="save-row">
                <button class="save-btn" id="saveBtn" onclick="savePet()">Save</button>
                <button class="save-btn" id="pauseBtn" onclick="togglePause()">Pause</button>
                <button class="save-btn" onclick="openChooser()">Switch Pet</button>
                <button class="save-btn" onclick="openShop()">Shop</button>
                <button class="save-btn" onclick="openGuide()">📖 Care guide</button>
                <button class="save-btn" onclick="openCloud()">☁️ Cloud</button>
                <button class="save-btn" onclick="openPrivacy()">🔒 Privacy</button>
                <button class="save-btn" onclick="openStats()">📊 My stats</button>
                <button class="save-btn" onclick="openSettings()">⚙️ Settings</button>
                <button class="save-btn" onclick="openBook()">📔 Book</button>
                <button class="save-btn" onclick="openProjects()">🎯 Projects</button>
                <button class="save-btn" onclick="openHelp()">❓ Help</button>
            </div>
            <div class="save-row">
                <span class="save-status" id="saveStatus">Progress is saved automatically</span>
            </div>
            <label class="setting">
                <input type="checkbox" id="hibernationToggle" onchange="setHibernation(this.checked)">
                <span id="hibernationLabel">Gentle mode: hibernate instead of dying if neglected</span>
            </label>

            <div class="about" id="about">
                <div id="aboutBreed"></div>
                <div id="aboutPersonality"></div>
                <div id="aboutAttention"></div>
                <div id="aboutLikes"></div>
                <div id="aboutTolerates"></div>
                <div id="aboutDislikes"></div>
                <div id="aboutChanges"></div>
                <div id="aboutTrends"></div>
                <div id="aboutRewarded"></div>
                <div id="aboutRelations"></div>
                <div id="aboutTreasures"></div>
                <div id="aboutAchievements"></div>
                <div id="aboutSeason"></div>
                <div id="aboutGeneration"></div>
            </div>

            <div class="status-text" id="statusText">Age: 0 days | Wellbeing: 100%</div>
        </div>
    </div>

    <div class="container hidden" id="shopView">
        <div class="shop-header">
            <h1 id="shopTitle">Shop</h1>
            <div class="friendship" id="shopPoints"></div>
        </div>
        <div id="shopList"></div>
        <div class="shop-msg" id="shopMsg"></div>
        <div style="text-align: center;">
            <button class="back-btn" id="shopBackBtn" onclick="closeShop()">Back to my pet</button>
        </div>
    </div>

    <div class="modal hidden" id="popupModal" role="dialog" aria-modal="true" aria-labelledby="popupTitle">
        <div class="modal-box">
            <div class="popup-icon" id="popupIcon" aria-hidden="true"></div>
            <h3 id="popupTitle"></h3>
            <p id="popupText"></p>
            <div class="modal-actions" id="popupActions"></div>
        </div>
    </div>

    <div class="modal hidden" id="deleteModal" role="dialog" aria-modal="true">
        <div class="modal-box">
            <div id="deleteStep1">
                <h3 id="deleteTitle"></h3>
                <p id="deleteWarning"></p>
                <div class="modal-actions">
                    <button class="save-btn" onclick="closeDeleteModal()">Cancel</button>
                    <button class="danger-btn" onclick="showDeleteStep2()">Delete</button>
                </div>
            </div>
            <div id="deleteStep2" class="hidden">
                <h3>Are you sure?</h3>
                <p id="deletePrompt"></p>
                <input type="text" id="deleteConfirmInput" autocomplete="off" oninput="updateDeleteButton()">
                <label class="setting"><input type="checkbox" id="deleteBookInput"> <span>Delete its book too</span></label>
                <div class="modal-actions">
                    <button class="save-btn" onclick="closeDeleteModal()">Cancel</button>
                    <button class="danger-btn" id="deleteFinalBtn" onclick="confirmDelete()" disabled>Delete forever</button>
                </div>
            </div>
        </div>
    </div>

    <script src="/static/js/pet-art.js?v={{static}}"></script>
    <script src="/static/js/ar.js?v={{static}}"></script>
    <script src="/static/js/ar-link.js?v={{static}}"></script>
    <script src="/static/js/shop.js?v={{static}}"></script>
    <script src="/static/js/breeds.js?v={{static}}"></script>
    <script src="/static/js/shows.js?v={{static}}"></script>
    <script src="/static/js/collection.js?v={{static}}"></script>
    <script src="/static/js/minigames.js?v={{static}}"></script>
    <script src="/static/js/sounds.js?v={{static}}"></script>
    <script src="/static/js/book.js?v={{static}}"></script>
    <script>
        // The kinds of pet the game has (from /api/species): their names and
        // icons, and which care buttons each uses. The cat, the default species, is
        // known before they load.
        let speciesList = [{
            id: 'cat', name: 'Cat', icon: '🐱', clean_label: '🧹 Scoop litter', walks: false
        }];

        function speciesInfo(id) {
            return speciesList.find(s => s.id === id) || speciesList[0];
        }

        // An icon and a space, to put before a pet's name
        function speciesIcon(id) {
            return speciesInfo(id).icon + ' ';
        }

        // The drawing of a pet (SVG from GochiArt) in a state
        function petArt(status, state, size) {
            return GochiArt.draw({species: status.species, look: status.breed && status.breed.look, state: state,
                outfit: status.outfit, size: size, label: status.name});
        }

        // Draws the active pet in a state (its current one if none given)
        function drawPet(state, status) {
            status = status || lastStatus;
            if (!status) return;
            document.getElementById('petSprite').innerHTML = petArt(status, state || spriteState(status), 170);
        }

        // Which state the pet is drawn in now (one of each species' sprite states)
        function spriteState(status) {
            if (!status.is_alive) return 'dead';
            if (status.hibernating) return 'hibernating';
            const byBehavior = {Sleeping: 'sleeping', Sick: 'sick', Playing: 'playing', Exploring: 'exploring',
                'Social Interaction': 'social', Eating: 'eating'};
            if (byBehavior[status.behavior]) return byBehavior[status.behavior];
            return status.happiness > 0.7 ? 'happy' : 'idle';
        }

        // Load the species and offer each in the new pet form
        function loadSpecies() {
            return fetch('/api/species').then(r => r.json()).then(data => {
                if (!data.success || data.data.length === 0) return;
                speciesList = data.data;
                const choice = document.getElementById('speciesChoice');
                choice.replaceChildren();
                speciesList.forEach((s, i) => {
                    const label = document.createElement('label');
                    const input = document.createElement('input');
                    input.type = 'radio';
                    input.name = 'species';
                    input.value = s.id;
                    input.checked = i === 0;
                    const span = document.createElement('span');
                    span.textContent = s.icon + ' ' + s.name;
                    label.append(input, span);
                    input.addEventListener('change', () => loadBreeds(s.id));
                    choice.append(label);
                });
                loadBreeds(speciesList[0].id);
            }).catch(console.error);
        }

        let hasPet = false;
        let updateInterval = null;
        let choosing = false;      // Pet list is open
        let paused = false;
        let manualPause = false;

        // The player counts as present if they used the page in the last
        // minute while it was visible. The server pauses time when they
        // have been away for a while, so a tab left open does not count.
        let lastInput = Date.now();
        ['mousemove', 'mousedown', 'keydown', 'touchstart', 'wheel', 'scroll'].forEach(type =>
            window.addEventListener(type, () => { lastInput = Date.now(); }, {passive: true}));

        function playerPresent() {
            return !document.hidden && Date.now() - lastInput < 60000;
        }

        // Hidden pages don't update; a page coming back updates at once
        document.addEventListener('visibilitychange', () => {
            if (!document.hidden) fetchStatus();
            startUpdates();
        });

        function showCreateView() {
            document.getElementById('createView').classList.remove('hidden');
            document.getElementById('petView').classList.add('hidden');
            document.getElementById('backBtn').classList.toggle('hidden', !hasPet);
            loadSavedPets();
        }

        function openChooser() {
            choosing = true;
            showCreateView();
        }

        function closeChooser() {
            choosing = false;
            showPetView();
            fetchStatus();
        }

        function formatAge(days) {
            return days.toFixed(1) + ' days old';
        }

        function loadSavedPets() {
            fetch('/api/saves')
                .then(r => r.json())
                .then(data => {
                    const container = document.getElementById('savedPets');
                    const list = document.getElementById('savedPetList');
                    list.replaceChildren();
                    const pets = (data.success && data.data.pets) || [];
                    container.classList.toggle('hidden', pets.length === 0);

                    pets.forEach(pet => {
                        const item = document.createElement('li');
                        const isActive = hasPet && pet.slot === data.data.active_slot;
                        if (isActive) item.classList.add('active');

                        const info = document.createElement('div');
                        info.className = 'saved-pet-info';
                        const name = document.createElement('div');
                        name.className = 'saved-pet-name';
                        name.textContent = pet.name + (pet.outfit ? ' ' + pet.outfit : '');
                        name.prepend(pet.unreadable ? speciesIcon(pet.species) : savedPetArt(pet));
                        const meta = document.createElement('div');
                        meta.className = 'saved-pet-meta';
                        const state = pet.unreadable ? 'This save is damaged and can\'t be loaded' :
                            !pet.is_alive ? 'Passed away' :
                            pet.hibernating ? 'Hibernating · ' + formatAge(pet.age) :
                            formatAge(pet.age) + (pet.paused ? ' · paused' : '');
                        meta.textContent = state + ' · saved ' + new Date(pet.saved_at).toLocaleString();
                        info.append(name, meta);

                        const btn = document.createElement('button');
                        btn.className = 'save-btn';
                        btn.textContent = isActive ? 'Playing' : 'Load';
                        btn.disabled = isActive || pet.unreadable;
                        btn.onclick = () => loadPet(pet.slot, btn);

                        const del = document.createElement('button');
                        del.className = 'save-btn delete-btn';
                        del.textContent = 'Delete';
                        del.onclick = () => openDeleteModal(pet);

                        const actions = document.createElement('div');
                        actions.className = 'saved-pet-actions';
                        actions.append(btn);
                        // While it isn't the active pet, a pet lives on with the household
                        // unless it is paused
                        if (pet.is_alive && !pet.unreadable) {
                            const pause = document.createElement('button');
                            pause.className = 'save-btn';
                            pause.textContent = pet.paused ? '▶ Live on' : '⏸ Pause';
                            pause.title = pet.paused ? 'Let it age and be looked after by the household while you play another pet' :
                                'Keep it as it is while you play another pet';
                            pause.onclick = () => postJSON('/api/saves/pause', {slot: pet.slot, paused: !pet.paused})
                                .then(() => loadSavedPets()).catch(console.error);
                            actions.append(pause);
                        }
                        actions.append(del);

                        item.append(info, actions);
                        list.append(item);
                    });
                })
                .catch(err => console.error('Saved pets error:', err));
        }

        function loadPet(slot, btn) {
            hideError();
            btn.disabled = true;
            btn.textContent = 'Loading...';

            fetch('/api/load', {
                method: 'POST',
                headers: {'Content-Type': 'application/json'},
                body: JSON.stringify({slot: slot})
            })
            .then(r => r.json())
            .then(data => {
                if (data.success) {
                    hasPet = true;
                    choosing = false;
                    showPetView();
                    fetchStatus();
                    startUpdates();
                } else {
                    showError(data.error || 'Could not load pet');
                    loadSavedPets();
                }
            })
            .catch(() => {
                showError('Connection error. Please try again.');
                loadSavedPets();
            });
        }

        // Deleting takes two confirmations: a warning, then typing the name
        let deleteTarget = null;

        function openDeleteModal(pet) {
            deleteTarget = pet;
            document.getElementById('deleteTitle').textContent = 'Delete ' + pet.name + '?';
            document.getElementById('deleteWarning').textContent =
                'This permanently deletes ' + pet.name + ' and all of its saved backups. It cannot be undone.';
            document.getElementById('deleteStep1').classList.remove('hidden');
            document.getElementById('deleteBookInput').checked = false;
            document.getElementById('deleteStep2').classList.add('hidden');
            document.getElementById('deleteModal').classList.remove('hidden');
        }

        function showDeleteStep2() {
            document.getElementById('deletePrompt').textContent =
                'Type ' + deleteTarget.name + ' below to delete it forever.';
            const input = document.getElementById('deleteConfirmInput');
            input.value = '';
            updateDeleteButton();
            document.getElementById('deleteStep1').classList.add('hidden');
            document.getElementById('deleteStep2').classList.remove('hidden');
            input.focus();
        }

        function updateDeleteButton() {
            const typed = document.getElementById('deleteConfirmInput').value.trim();
            document.getElementById('deleteFinalBtn').disabled = !deleteTarget || typed !== deleteTarget.name;
        }

        function closeDeleteModal() {
            deleteTarget = null;
            document.getElementById('deleteModal').classList.add('hidden');
        }

        function confirmDelete() {
            if (!deleteTarget) return;
            const btn = document.getElementById('deleteFinalBtn');
            btn.disabled = true;

            postJSON('/api/delete', {
                slot: deleteTarget.slot,
                confirm_name: document.getElementById('deleteConfirmInput').value,
                delete_book: document.getElementById('deleteBookInput').checked
            })
            .then(data => {
                closeDeleteModal();
                if (data.success) {
                    if (data.data.was_active) hasPet = false;
                    showCreateView();
                } else {
                    showError(data.error || 'Could not delete pet');
                }
            })
            .catch(() => {
                closeDeleteModal();
                showError('Connection error. Please try again.');
            });
        }

        function setHibernation(enabled) {
            postJSON('/api/hibernation', {enabled: enabled})
                .then(data => {
                    if (!data.success) showPointsToast(data.error || 'Something went wrong');
                    fetchStatus();
                })
                .catch(console.error);
        }

        function revivePet() {
            const btn = document.getElementById('reviveBtn');
            btn.disabled = true;
            postJSON('/api/revive', {})
                .then(() => fetchStatus())
                .catch(console.error)
                .finally(() => { btn.disabled = false; });
        }

        function togglePause() {
            fetch('/api/pause', {
                method: 'POST',
                headers: {'Content-Type': 'application/json'},
                body: JSON.stringify({paused: !manualPause})
            })
            .then(r => r.json())
            .then(() => fetchStatus())
            .catch(console.error);
        }

        function updatePauseUI(status) {
            paused = status.paused;
            manualPause = status.pause_reason === 'manual';
            const banner = document.getElementById('pauseBanner');
            if (status.pause_reason === 'manual') {
                banner.textContent = '⏸ Time is paused. ' + status.name + ' is safe until you press Resume.';
            } else if (status.pause_reason === 'away') {
                banner.textContent = '⏸ Time paused while you were away.';
            }
            banner.classList.toggle('hidden', !paused);
            document.getElementById('pauseBtn').textContent = manualPause ? 'Resume' : 'Pause';
        }

        function showPetView() {
            document.getElementById('createView').classList.add('hidden');
            document.getElementById('petView').classList.remove('hidden');
        }

        function formatPoints(points) {
            return '❤️ ' + points + ' Friendship Point' + (points === 1 ? '' : 's');
        }

        // Shop: spend the active pet's Friendship Points on items for it
        let shopping = false;
        let toastTimer = null;
        // lines: array of [text, className]
        function showPointsToast(lines) {
            if (typeof lines === 'string') lines = [[lines, '']];
            const toast = document.getElementById('pointsToast');
            toast.replaceChildren();
            lines.forEach(([text, className]) => {
                const line = document.createElement('div');
                line.textContent = text;
                if (className) line.className = className;
                toast.append(line);
            });
            if (toastTimer) clearTimeout(toastTimer);
            // Lessons take longer to read
            const lesson = lines.some(([, className]) => className === 'lesson');
            toastTimer = setTimeout(() => { toast.replaceChildren(); }, lesson ? 10000 : 5000);
        }

        function formatChanges(changes) {
            return changes.map(c => {
                const percent = Math.round(c.change * 100);
                return c.stat + ' ' + (percent > 0 ? '+' : '−') + Math.abs(percent) + '%';
            }).join(' · ');
        }

        function showError(msg) {
            const errEl = document.getElementById('errorMsg');
            errEl.textContent = msg;
            errEl.classList.remove('hidden');
        }

        function hideError() {
            document.getElementById('errorMsg').classList.add('hidden');
        }

        function createPet() {
            const nameInput = document.getElementById('petNameInput');
            const name = nameInput.value.trim();
            const btn = document.getElementById('createBtn');

            hideError();

            if (!name) {
                showError('Please enter a name for your pet');
                nameInput.focus();
                return;
            }

            btn.disabled = true;
            btn.textContent = 'Creating...';

            fetch('/api/create', {
                method: 'POST',
                headers: {'Content-Type': 'application/json'},
                body: JSON.stringify({name: name, hibernation: document.getElementById('hibernationInput').checked,
                    species: document.querySelector('input[name="species"]:checked').value, breed: selectedBreed()})
            })
            .then(r => r.json())
            .then(data => {
                if (data.success) {
                    hasPet = true;
                    choosing = false;
                    nameInput.value = '';
                    btn.disabled = false;
                    btn.textContent = 'Create My Pet!';
                    showPetView();
                    fetchStatus();
                    startUpdates();
                } else {
                    showError(data.error || 'Failed to create pet');
                    btn.disabled = false;
                    btn.textContent = 'Create My Pet!';
                }
            })
            .catch(err => {
                showError('Connection error. Please try again.');
                btn.disabled = false;
                btn.textContent = 'Create My Pet!';
            });
        }

        function updateUI(status) {
            document.getElementById('petName').textContent = status.name +
                ((status.titles || []).includes('grand_champion') ? ' 👑' : '');
            // The pet's request: speech bubble, highlighted button, tab title
            const speech = document.getElementById('speech');
            const request = status.request;
            speech.textContent = request ? request.icon + ' ' + request.message : '';
            speech.classList.toggle('hidden', !request);
            document.querySelectorAll('.requested').forEach(el => el.classList.remove('requested'));
            if (request) {
                const target = request.action === 'revive' ?
                    document.getElementById('reviveBtn') :
                    document.querySelector('.action-btn.' + request.action + ', .activity-btn.' + request.action);
                if (target) target.classList.add('requested');
            }
            document.title = request ? '💬 ' + status.name + ' needs you - Gochi' : status.name + ' - Gochi';
            const backLabel = 'Back to ' + status.name;
            document.getElementById('backBtn').textContent = backLabel;
            document.getElementById('shopBackBtn').textContent = backLabel;
            document.getElementById('shopTitle').textContent = 'Shop for ' + status.name;

            Object.entries(status.care_tips || {}).forEach(([action, tip]) => {
                const btn = document.querySelector('.action-btn.' + action);
                if (btn) btn.title = tip;
            });

            const env = status.environment;
            document.getElementById('environment').textContent = env ?
                env.icon + ' ' + env.weather + ', ' + env.temperature + '°C · ' + env.season + ' ' + env.day_phase.toLowerCase() +
                (env.outdoor_safe ? '' : ' · risky for exploring') : '';

            renderBreedAbout(status.breed, status.name);
            document.getElementById('aboutPersonality').textContent = status.personality;
            document.getElementById('aboutGeneration').textContent = 'Generation ' + (status.generation || 1);
            document.getElementById('aboutSeason').textContent = env && env.favorite_season ?
                'Favorite season: ' + env.favorite_season + (env.favorite_season === env.season ? ' (now!)' : '') : '';
            const setList = (id, label, items, none) => {
                const el = document.getElementById(id);
                el.replaceChildren();
                const strong = document.createElement('span');
                strong.className = 'about-label';
                strong.textContent = label + ': ';
                el.append(strong, items.length ? items.join(', ') : none);
            };
            setList('aboutLikes', 'Likes', status.likes || [], 'nothing yet');
            setList('aboutTolerates', 'Puts up with', status.tolerates || [], 'nothing yet');
            setList('aboutDislikes', 'Dislikes', status.dislikes || [], 'nothing yet');
            setList('aboutAttention', 'Attention', status.attention ? [status.name + ' ' + status.attention] : [], 'unknown');
            setList('aboutChanges', 'Changed since adoption', status.personality_changes || [], 'not yet');
            setList('aboutTrends', 'Lately becoming', status.trends || [], 'no clear change');
            setList('aboutAchievements', 'Achievements (' + (status.achievements || []).length + ' of ' +
                status.achievement_total + ')', (status.achievements || []).map(a => a.icon + ' ' + a.name), 'none yet');
            setList('aboutRewarded', 'Has learned you respond to', status.rewarded || [], 'nothing yet');
            setList('aboutRelations', 'Knows', (status.relations || []).map(r => r.name + ' (' + r.description +
                (r.last_shared ? '; last: ' + r.last_shared : '') + ')'),
                'no other pets yet');
            document.getElementById('activity').textContent =
                status.is_alive ? status.name + ' ' + status.activity :
                status.name + ' has passed away' + (status.cause_of_death ? ' (' + status.cause_of_death.toLowerCase() + ')' : '') + '.';
            document.getElementById('hibernationToggle').checked = status.hibernation_enabled;
            document.getElementById('hibernationToggle').disabled = !status.is_alive;
            document.getElementById('hibernationLabel').textContent =
                'Gentle mode: ' + status.name + ' hibernates instead of dying if neglected';
            document.getElementById('healthBar').style.width = (status.health * 100) + '%';
            document.getElementById('energyBar').style.width = (status.energy * 100) + '%';
            document.getElementById('happinessBar').style.width = (status.happiness * 100) + '%';
            document.getElementById('hungerBar').style.width = (status.hunger * 100) + '%';
            document.getElementById('waterBar').style.width = (status.water * 100) + '%';
            const needLabels = {'Exercise': 'Exercise', 'Social': 'Company', 'Mental Stimulation': 'Learning',
                'Affection': 'Affection', 'Exploration': 'Adventure'};
            const needs = document.getElementById('needs');
            needs.replaceChildren();
            (status.needs || []).forEach(n => {
                const cell = document.createElement('div');
                cell.className = 'need';
                cell.title = n.need + ': ' + Math.round(n.level * 100) + '%';
                cell.textContent = needLabels[n.need] || n.need;
                const bar = document.createElement('div');
                bar.className = 'need-bar';
                const fill = document.createElement('div');
                fill.className = 'need-fill' + (n.level < 0.2 ? ' critical' : n.level < 0.4 ? ' low' : '');
                fill.style.width = Math.round(n.level * 100) + '%';
                bar.append(fill);
                cell.append(bar);
                needs.append(cell);
            });
            document.getElementById('friendship').textContent = formatPoints(status.friendship_points) +
                (status.home_stars !== undefined ? ' · ⭐ ' + status.home_stars + ' Home Stars' : '');
            const belongings = document.getElementById('belongings');
            belongings.replaceChildren();
            (status.items || []).forEach(item => {
                const span = document.createElement('span');
                span.textContent = item.icon;
                span.title = item.name;
                belongings.append(span);
            });

            const effects = document.getElementById('effects');
            effects.replaceChildren();
            const active = status.effects || [];
            if (active.length > 0) {
                const title = document.createElement('div');
                title.className = 'effects-title';
                title.textContent = status.hibernating ? 'Item effects (paused while hibernating)' : 'Active effects';
                effects.append(title);
                active.forEach(item => {
                    const line = document.createElement('div');
                    line.textContent = item.icon + ' ' + item.name + ': ' + item.description;
                    effects.append(line);
                });
            }
            effects.classList.toggle('hidden', active.length === 0);
            document.getElementById('mood').textContent = 'Mood: ' + status.mood;
            document.getElementById('statusText').textContent =
                (status.life ? status.life.stage : 'Age') + ': ' + status.age.toFixed(1) + ' days | Wellbeing: ' +
                (status.wellbeing * 100).toFixed(0) + '%';

            document.getElementById('petView').classList.toggle('dead', !status.is_alive);
            drawPet(spriteState(status), status);
            renderShelf(status.shelf);
            renderHouseLook(status.house_look, status.name);
            renderUpkeep(status);
            showPopups(status.popups);
            showHappening(status.happening);
            updatePauseUI(status);

            const hibernating = status.is_alive && status.hibernating;
            document.getElementById('hibernateText').textContent = hibernating ?
                '💤 ' + status.name + ' got too weak and is hibernating to stay safe. Its stats are paused. Revive it to keep playing.' : '';
            document.getElementById('hibernateBanner').classList.toggle('hidden', !hibernating);
            document.querySelectorAll('.actions .action-btn').forEach(b => {
                b.disabled = hibernating || (status.exploring && !b.classList.contains('clean'));
            });
            updateHealth(status);
            updateActivities(status);
        }

        // Health care: the pet's organ systems, its living area, illness,
        // warning signs and today's checklist
        const bodyLabels = {Digestion: 'Digestion', Heart: 'Heart', Lungs: 'Lungs', Nerves: 'Nerves'};
        function updateHealth(status) {
            const kind = speciesInfo(status.species);
            document.getElementById('cleanBtn').textContent = kind.clean_label;
            document.getElementById('walkBtn').classList.toggle('hidden', !kind.walks);

            const body = document.getElementById('bodySystems');
            body.replaceChildren();
            (status.body || []).forEach(system => {
                const cell = document.createElement('div');
                cell.className = 'need';
                cell.title = system.name + ': ' + Math.round(system.level * 100) + '%';
                cell.textContent = bodyLabels[system.name] || system.name;
                const bar = document.createElement('div');
                bar.className = 'need-bar';
                const fill = document.createElement('div');
                fill.className = 'need-fill' + (system.level < 0.3 ? ' critical' : system.level < 0.6 ? ' low' : '');
                fill.style.width = Math.round(system.level * 100) + '%';
                bar.append(fill);
                cell.append(bar);
                body.append(cell);
            });

            // Age, life stage and lifespan; the sleep cycle and hormones
            const life = status.life;
            if (life) {
                let text = '🎂 ' + life.stage + ', ' + Math.floor(life.age) + ' game days old';
                if (!status.is_alive) {
                    text += '.';
                } else if (life.old_age_death) {
                    text += ' · expected to live about ' + Math.round(life.lifespan).toLocaleString() + ' game days (' +
                        formatPlayTime(status.life_seconds) + ' of play left)';
                } else {
                    text += ' · gentle mode: will not die of old age';
                }
                document.getElementById('lifeLine').textContent = text;
                document.getElementById('lifeLine').title = 'Lifespan depends on species, genes and how well ' + status.name +
                    ' has been cared for over its life (care record ' + Math.round(life.care_quality * 100) + '%)' +
                    (life.extension > 0 ? '; potions have added ' + Math.round(life.extension).toLocaleString() + ' game days' : '') + '.';
                const h = life.hormones;
                document.getElementById('rhythmLine').textContent = status.is_alive ?
                    '🌙 ' + life.sleep_phase + (life.sleep_debt >= 1 ? ' · owes ' + life.sleep_debt.toFixed(0) + ' h of sleep' : '') +
                    ' · stress hormone ' + Math.round(h.cortisol * 100) + '% · mood ' + Math.round(h.serotonin * 100) +
                    '% · bonding ' + Math.round(h.oxytocin * 100) + '% · growth ' + Math.round(h.growth * 100) +
                    '% · sleep hormone ' + Math.round(h.melatonin * 100) + '%' : '';
                const potion = document.getElementById('potionBtn');
                potion.classList.toggle('hidden', !status.is_alive || life.potions === 0);
                potion.textContent = '🧪 Give life-extension potion (' + life.potions + ' in the pantry)';
            }

            const area = status.living_area || {name: 'Litter box', cleanliness: 1};
            document.getElementById('livingAreaLabel').textContent = area.name;
            const areaBar = document.getElementById('livingAreaBar');
            areaBar.style.width = Math.round(area.cleanliness * 100) + '%';
            areaBar.className = 'stat-fill area' + (area.cleanliness < 0.4 ? ' critical' : area.cleanliness < 0.6 ? ' low' : '');
            areaBar.parentElement.title = area.name + ': ' + Math.round(area.cleanliness * 100) + '% clean';

            const illness = document.getElementById('illnessLine');
            illness.textContent = status.illness ? '🤒 ' + status.name + ' is ill: ' + status.illness : '';
            illness.classList.toggle('hidden', !status.illness);

            const symptoms = document.getElementById('symptoms');
            symptoms.replaceChildren();
            (status.symptoms || []).forEach(symptom => {
                const box = document.createElement('div');
                box.className = 'symptom' + (symptom.good ? ' good' : '');
                const sign = document.createElement('div');
                sign.textContent = (symptom.good ? '💚 ' : '⚠️ ') + symptom.sign;
                box.append(sign);
                if (symptom.hint || symptom.topic) {
                    const hint = document.createElement('div');
                    hint.className = 'symptom-hint';
                    hint.textContent = symptom.hint ? symptom.hint + ' ' : '';
                    if (symptom.topic) {
                        const more = document.createElement('button');
                        more.className = 'link-btn';
                        more.textContent = 'Learn more';
                        more.onclick = () => openGuide(symptom.topic);
                        hint.append(more);
                    }
                    box.append(hint);
                }
                symptoms.append(box);
            });

            const checklist = document.getElementById('checklist');
            checklist.replaceChildren();
            const title = document.createElement('div');
            title.className = 'section-title';
            title.textContent = "Today's care for " + status.name;
            checklist.append(title);
            (status.checklist || []).forEach(item => {
                const line = document.createElement('div');
                const done = item.done >= item.target;
                line.className = 'checklist-item' + (done ? '' : ' todo');
                line.textContent = (done ? '✅ ' : '⬜ ') + item.label +
                    (item.target > 1 ? ' (' + item.done + '/' + item.target + ')' : '');
                checklist.append(line);
            });

            updateGoals(status);
            updateForecast(status);
            updateThingsToDo(status);
            fetchEvents(status.name);
        }

        // Today's goals, with the bonus points each earns
        function updateGoals(status) {
            const box = document.getElementById('goals');
            const goals = status.goals || [];
            box.classList.toggle('hidden', !status.is_alive || goals.length === 0);
            box.replaceChildren();
            const title = document.createElement('div');
            title.className = 'section-title';
            title.textContent = "Today's goals (level " + status.goal_level + ')';
            box.append(title);
            goals.forEach(goal => {
                const line = document.createElement('div');
                line.className = 'goal-item' + (goal.done ? ' done' : '');
                let text = (goal.done ? '✅ ' : goal.icon + ' ') + goal.text;
                if (!goal.done && goal.kind === 'care' && goal.target > 1) {
                    text += ' (' + goal.progress + '/' + goal.target + ')';
                } else if (!goal.done && goal.kind === 'happiness') {
                    text += ' (now ' + goal.progress + '%)';
                } else if (!goal.done && goal.unit === 'seconds') {
                    text += ' (' + Math.floor(goal.progress / 60) + '/' + Math.round(goal.target / 60) + ' min)';
                }
                line.textContent = text + ' ';
                const reward = document.createElement('span');
                reward.className = 'goal-reward';
                reward.textContent = '+' + goal.reward + ' ❤️';
                line.append(reward);
                box.append(line);
            });
            const week = status.weekly && status.weekly.goal;
            if (week && week.kind) {
                const line = document.createElement('div');
                line.className = 'goal-item weekly' + (week.done ? ' done' : '');
                line.textContent = (week.done ? '✅ ' : '🏅 ') + 'This week: ' + week.text +
                    (week.done ? ' ' : ' (' + week.progress + '/' + week.target + ') ');
                const reward = document.createElement('span');
                reward.className = 'goal-reward';
                reward.textContent = '+' + week.reward + ' ❤️ +' + week.stars + ' ⭐';
                line.append(reward);
                box.append(line);
            }
        }

        // What the player can do with the pet now, besides care: each
        // activity ready opens from here; the others say why they wait
        const thingActions = {train: () => openTricks(), hide_seek: () => startHideSeek(), games: () => openGames(),
            explore: () => explore(), shows: () => openShows(), enrich: () => openChoices('enrich')};

        let shownThings = '';

        function updateThingsToDo(status) {
            const box = document.getElementById('thingsToDo');
            const things = status.things_to_do || [];
            // Rebuilt only when it changes, so a button isn't replaced as it is clicked
            const key = JSON.stringify(things);
            if (key === shownThings) return;
            shownThings = key;
            box.classList.toggle('hidden', things.length === 0);
            box.replaceChildren();
            const title = document.createElement('div');
            title.className = 'section-title';
            title.textContent = 'Things to do now';
            box.append(title);
            things.forEach(t => {
                const line = document.createElement('div');
                line.className = 'thing-item' + (t.ready ? '' : ' waiting');
                let name;
                if (t.ready && thingActions[t.id]) {
                    name = document.createElement('button');
                    name.className = 'thing-btn';
                    name.dataset.thing = t.id;
                    name.onclick = thingActions[t.id];
                } else {
                    name = document.createElement('span');
                    name.className = 'thing-name';
                }
                name.textContent = t.icon + ' ' + t.name;
                const why = document.createElement('span');
                why.className = 'thing-why';
                why.textContent = t.why;
                line.append(name, why);
                box.append(line);
            });
        }

        // What the pet will soon need, and when the player usually plays
        function updateForecast(status) {
            const box = document.getElementById('forecast');
            const forecasts = status.forecasts || [];
            box.classList.toggle('hidden', forecasts.length === 0 && !status.play_schedule);
            box.replaceChildren();
            const title = document.createElement('div');
            title.className = 'section-title';
            title.textContent = 'Coming up';
            box.append(title);
            forecasts.forEach(f => {
                const line = document.createElement('div');
                line.className = 'forecast-item';
                line.textContent = f.icon + ' In ' + formatWait(f.in_seconds) + ': ' + f.message;
                box.append(line);
            });
            if (forecasts.length === 0) {
                const line = document.createElement('div');
                line.className = 'forecast-item';
                line.textContent = status.name + " won't need anything new for a while.";
                box.append(line);
            }
            if (status.play_schedule) {
                const line = document.createElement('div');
                line.className = 'play-schedule';
                line.textContent = 'You usually play on ' + status.play_schedule + '.';
                box.append(line);
            }
        }

        // Things that happened to the pet on its own (illness, hunger,
        // trips ending), newest first
        const maxLoggedEvents = 8;
        let lastEventSeq = 0;
        let eventLogPet = null;
        let fetchingEvents = false;

        function fetchEvents(petName) {
            if (petName !== eventLogPet) {
                // Another pet: show its own events from the start
                eventLogPet = petName;
                lastEventSeq = 0;
                document.getElementById('eventLogItems').replaceChildren();
                document.getElementById('eventLog').classList.add('hidden');
            }
            if (fetchingEvents) return; // The next status update asks again
            fetchingEvents = true;
            fetch('/api/events?after=' + lastEventSeq)
                .then(r => r.json())
                .then(data => {
                    if (!data.success || petName !== eventLogPet) return;
                    lastEventSeq = data.data.last;
                    const items = document.getElementById('eventLogItems');
                    data.data.events.forEach(event => {
                        const line = document.createElement('div');
                        line.className = 'event-log-item' + (event.good ? ' good' : '');
                        const time = document.createElement('span');
                        time.className = 'event-log-time';
                        time.textContent = new Date(event.timestamp).toLocaleTimeString([], {hour: 'numeric', minute: '2-digit'});
                        line.append(time, event.message);
                        items.prepend(line);
                    });
                    while (items.children.length > maxLoggedEvents) items.lastChild.remove();
                    document.getElementById('eventLog').classList.toggle('hidden', items.children.length === 0);
                })
                .catch(console.error)
                .finally(() => { fetchingEvents = false; });
        }

        // The care guide for the pet's species; topics related to its
        // warning signs are marked, and openTopic is opened
        function openGuide(openTopic) {
            panelMode = 'guide';
            fetch('/api/guide')
                .then(r => r.json())
                .then(data => {
                    if (!data.success) return;
                    const relevant = (lastStatus && lastStatus.guide_topics) || [];
                    const p = panel();
                    p.replaceChildren(panelHeader('Caring for your ' + data.data.species));
                    data.data.topics.forEach(topic => {
                        const details = document.createElement('details');
                        details.className = 'guide-topic' + (relevant.includes(topic.id) ? ' relevant' : '');
                        details.open = topic.id === openTopic;
                        const summary = document.createElement('summary');
                        summary.textContent = (relevant.includes(topic.id) ? '⚠️ ' : '') + topic.title;
                        const text = document.createElement('p');
                        text.textContent = topic.text;
                        details.append(summary, text);
                        p.append(details);
                    });
                    p.classList.remove('hidden');
                    p.scrollIntoView({behavior: 'smooth', block: 'nearest'});
                })
                .catch(console.error);
        }

        function fetchStatus() {
            fetch('/api/status' + (playerPresent() ? '?active=1' : ''))
                .then(r => r.json())
                .then(data => {
                    if (data.success) {
                        hasPet = true;
                        if (!choosing && !shopping) {
                            showPetView();
                            updateUI(data.data);
                        }
                    } else if (hasPet || !choosing) {
                        hasPet = false;
                        choosing = true;
                        showCreateView();
                    }
                })
                .catch(err => {
                    console.error('Status fetch error:', err);
                });
        }

        // How often the page updates (ms). The game runs on the server, so
        // updating less often only makes the page a little less current.
        // On battery the page updates less often, and on low battery less
        // still and without animations (where the browser reports the
        // battery); a hidden page doesn't update at all.
        const updatePaces = {charging: 2000, battery: 5000, lowBattery: 15000};
        let updatePace = updatePaces.charging;

        function startUpdates() {
            if (updateInterval) clearInterval(updateInterval);
            updateInterval = document.hidden ? null : setInterval(fetchStatus, updatePace);
        }

        function watchBattery() {
            if (!navigator.getBattery) return;
            navigator.getBattery().then(battery => {
                const adapt = () => {
                    const low = !battery.charging && battery.level <= 0.2;
                    updatePace = battery.charging ? updatePaces.charging : low ? updatePaces.lowBattery : updatePaces.battery;
                    document.body.classList.toggle('power-saving', low);
                    startUpdates();
                };
                battery.addEventListener('chargingchange', adapt);
                battery.addEventListener('levelchange', adapt);
                adapt();
            }).catch(() => {});
        }

        function interact(action) {
            const btn = event.target;
            btn.style.transform = 'scale(0.95)';
            setTimeout(() => btn.style.transform = '', 100);

            // Show reaction
            if (action === 'feed' || action === 'drink') drawPet('eating');
            else if (action === 'play') drawPet('playing');
            else if (action === 'pet') drawPet('happy');

            fetch('/api/interact', {
                method: 'POST',
                headers: {'Content-Type': 'application/json'},
                body: JSON.stringify({action: action})
            })
            .then(r => r.json())
            .then(data => {
                if (data.success) {
                    showResult(data.data);
                    setTimeout(fetchStatus, 500);
                } else if (data.error) {
                    showPointsToast(data.error);
                }
            })
            .catch(console.error);
        }

        // Shows what an action did: an optional headline, the pet's
        // reaction, stat changes, item boost and points
        function showResult(result, headline) {
            const lines = [];
            if (headline) lines.push([headline, 'reaction']);
            if (result.reaction_text) lines.push([result.reaction_text, 'reaction']);
            const face = {delighted: 'happy', reluctant: 'reluctant'}[result.reaction];
            if (face) drawPet(face);
            const changes = result.changes || [];
            if (changes.length > 0) lines.push([formatChanges(changes), 'changes']);
            const boost = result.boosted_by;
            if (boost) lines.push([boost.icon + ' ' + boost.name + ': ' + boost.description, 'boost']);
            if (result.bonus_points > 0) lines.push(['+' + result.bonus_points + ' game bonus!', '']);
            if (result.lesson) lines.push(['💡 ' + result.lesson, 'lesson']);
            if (result.repeated) lines.push([result.repeated, 'repeated']);
            if (result.note) lines.push([result.note, 'note']);
            if (result.suggestion) lines.push(['👉 ' + result.suggestion, 'note']);
            if ('points_earned' in result) {
                lines.push(result.points_earned > 0 ?
                    ['+' + result.points_earned + ' Friendship Points!', ''] :
                    [(lastStatus ? lastStatus.name : 'Your pet') + ' earned points for this recently. Try again in a few minutes.', '']);
            }
            showPointsToast(lines);
        }

        // Activities: tricks, hide and seek, exploring
        let lastStatus = null;
        let panelMode = null;   // 'tricks' or 'hideseek' while the panel is open
        let hideSpots = [];
        let dismissedAdventure = null;
        try { dismissedAdventure = localStorage.getItem('gochiDismissedAdventure'); } catch (e) {}

        // POST JSON. When a parent PIN is set, protected requests ask for it
        // and are sent again with it.
        function postJSON(url, body) {
            body = body || {};
            return fetch(url, {
                method: 'POST',
                headers: {'Content-Type': 'application/json'},
                body: JSON.stringify(body)
            }).then(r => r.json()).then(data => {
                if (data.success || !data.data || !data.data.parent_pin_needed || 'parent_pin' in body) return data;
                const pin = prompt(data.error + ' Ask a parent to enter it.');
                if (pin === null) return data;
                return postJSON(url, Object.assign({}, body, {parent_pin: pin}));
            });
        }

        function panel() {
            return document.getElementById('activityPanel');
        }

        function panelHeader(title) {
            const head = document.createElement('div');
            head.className = 'panel-head';
            const label = document.createElement('span');
            label.textContent = title;
            const close = document.createElement('button');
            close.className = 'panel-close';
            close.textContent = '✕';
            close.setAttribute('aria-label', 'Close');
            close.onclick = closePanel;
            head.append(label, close);
            return head;
        }

        function closePanel() {
            if (panelMode === 'hideseek' && lastStatus && lastStatus.hide_and_seek) quitHideSeek();
            panelMode = null;
            panel().classList.add('hidden');
            panel().replaceChildren();
        }

        // Ending a game or trip early loses anything it would have earned
        function quitHideSeek() {
            postJSON('/api/hideseek/quit')
                .then(data => {
                    if (data.success) showPointsToast(data.data.text);
                    fetchStatus();
                })
                .catch(console.error);
        }

        function recallPet() {
            const name = lastStatus ? lastStatus.name : 'your pet';
            if (!confirm('Call ' + name + ' home now? It will come back with nothing from this trip.')) return;
            postJSON('/api/explore/recall')
                .then(data => {
                    showPointsToast(data.success ? data.data.text : (data.error || 'Something went wrong'));
                    fetchStatus();
                })
                .catch(console.error);
        }

        // Care variants: what to feed, how to play, where to pet, and what to
        // change at home
        let careOptions = null;
        let trainingMethod = 'praise';
        function loadCareOptions() {
            return fetch('/api/care/options')
                .then(r => r.json())
                .then(data => { if (data.success) careOptions = data.data; return careOptions; });
        }

        function choiceRow(label, sub, buttonText, onClick) {
            const row = document.createElement('div');
            row.className = 'choice-row';
            const name = document.createElement('div');
            name.className = 'choice-name';
            name.textContent = label;
            if (sub) {
                const detail = document.createElement('div');
                detail.className = 'choice-sub';
                detail.textContent = sub;
                name.append(detail);
            }
            const btn = document.createElement('button');
            btn.className = 'save-btn';
            btn.textContent = buttonText;
            btn.onclick = onClick;
            row.append(name, btn);
            return row;
        }

        const choiceTitles = {feed: 'What to feed', play: 'How to play', pet: 'Where to pet', enrich: 'Change something at home'};
        // openChoices shows the choices for Feed, Pet, Play or Enrich. The
        // panel sits above the buttons, so a button scrolls the page to it;
        // refreshing the choices after one is made leaves the page where it is.
        function openChoices(kind, refresh) {
            panelMode = 'choice';
            loadCareOptions().then(options => {
                if (!options) return;
                renderChoices(kind, options);
                if (!refresh) panel().scrollIntoView({behavior: 'smooth', block: 'nearest'});
            }).catch(console.error);
        }

        function renderChoices(kind, options) {
            const p = panel();
            p.replaceChildren(panelHeader(choiceTitles[kind]));
            const hint = text => {
                const div = document.createElement('div');
                div.className = 'panel-text';
                div.textContent = text;
                p.append(div);
            };
            if (kind === 'feed') {
                options.foods.forEach(f => p.append(choiceRow(f.icon + ' ' + f.name,
                    f.quantity < 0 ? 'Always available' : f.quantity + ' left' + (f.warning ? ' · ⚠️ not recommended' : ''),
                    'Feed', () => giveChoice('feed', {food: f.id}, f.warning, kind))));
                options.treats.forEach(t => p.append(choiceRow(t.icon + ' ' + t.name, t.quantity + ' left', 'Give',
                    () => giveChoice('treat', {food: t.id}, '', kind))));
                if (options.foods.length === 1) hint('Other foods and treats are in the shop.');
            } else if (kind === 'play') {
                p.append(choiceRow('🤲 Play together', '', 'Play', () => giveChoice('play', {}, '', kind)));
                options.toys.forEach(t => p.append(choiceRow(t.icon + ' ' + t.name,
                    t.uses_left ? t.uses_left + ' plays left' + (t.quantity > 1 ? ' · ' + t.quantity + ' owned' : '') : '',
                    'Play', () => giveChoice('play', {toy: t.id}, '', kind))));
                if (options.toys.length === 0) hint('Toys are in the shop.');
            } else if (kind === 'pet') {
                options.regions.forEach(r => p.append(choiceRow(r.name, '', 'Pet',
                    () => giveChoice('pet', {region: r.id}, '', kind))));
            } else if (kind === 'enrich') {
                options.changes.forEach(c => p.append(choiceRow(c.icon + ' ' + c.name, '', 'Do it', () => {
                    postJSON('/api/enrich', {change: c.id}).then(data => afterChoice(data, kind)).catch(console.error);
                })));
            }
            p.classList.remove('hidden');
        }

        function giveChoice(action, extra, warning, kind) {
            if (warning && !confirm(warning)) return;
            postJSON('/api/interact', Object.assign({action: action}, extra))
                .then(data => afterChoice(data, kind))
                .catch(console.error);
        }

        function afterChoice(data, kind) {
            if (data.success) {
                showResult(data.data, data.data.text);
                setTimeout(fetchStatus, 500);
                if (panelMode === 'choice') openChoices(kind, true); // Quantities and uses change
            } else {
                showPointsToast(data.error || 'Something went wrong');
            }
        }

        // Introductions to the player's other pets
        function openMeet() {
            panelMode = 'meet';
            fetch('/api/introduce/partners')
                .then(r => r.json())
                .then(data => {
                    if (!data.success) {
                        showPointsToast(data.error || 'Something went wrong');
                        return;
                    }
                    const p = panel();
                    p.replaceChildren(panelHeader('Meet another pet'));
                    if (data.data.length === 0) {
                        const none = document.createElement('div');
                        none.className = 'panel-text';
                        none.textContent = 'Create or load another pet first to introduce them.';
                        p.append(none);
                    }
                    data.data.forEach(partner => p.append(choiceRow(
                        speciesIcon(partner.species) + partner.name,
                        partner.relationship || "Haven't met yet", 'Meet', () => {
                            postJSON('/api/introduce', {partner_slot: partner.slot})
                                .then(data => {
                                    if (data.success) {
                                        showResult(data.data, data.data.text);
                                        setTimeout(fetchStatus, 500);
                                        openMeet();
                                    } else {
                                        showPointsToast(data.error || 'Something went wrong');
                                    }
                                })
                                .catch(console.error);
                        })));
                    if (data.data.length > 0) renderGroup(p, data.data);
                    p.classList.remove('hidden');
                })
                .catch(console.error);
        }

        // Group activities: the active pet and up to three others play, nap
        // or (species that go for walks) go for a walk together
        const maxGroupPartners = 3;
        function renderGroup(p, partners) {
            const title = document.createElement('div');
            title.className = 'section-title';
            title.textContent = 'Spend time together';
            title.style.marginTop = '14px';
            const intro = document.createElement('div');
            intro.className = 'panel-text';
            intro.textContent = 'Pick up to ' + maxGroupPartners + ' pets to join ' + lastStatus.name +
                '. Pets that don\'t get along may clash, and the group sorts out who is in charge.';
            p.append(title, intro);

            const chosen = new Set();
            partners.forEach(partner => {
                const row = document.createElement('label');
                row.className = 'method-row';
                const box = document.createElement('input');
                box.type = 'checkbox';
                box.onchange = () => {
                    if (box.checked && chosen.size >= maxGroupPartners) {
                        box.checked = false;
                        showPointsToast('At most ' + maxGroupPartners + ' other pets at a time.');
                        return;
                    }
                    box.checked ? chosen.add(partner.slot) : chosen.delete(partner.slot);
                };
                row.append(box, ' ' + speciesIcon(partner.species) + partner.name);
                p.append(row);
            });

            const buttons = document.createElement('div');
            buttons.className = 'save-row';
            const activities = [['play', '🎾 Play'], ['nap', '💤 Nap']];
            if (speciesInfo(lastStatus.species).walks) activities.push(['walk', '🦮 Walk']);
            activities.forEach(([activity, label]) => {
                const btn = document.createElement('button');
                btn.className = 'save-btn';
                btn.textContent = label;
                btn.onclick = () => {
                    if (chosen.size === 0) {
                        showPointsToast('Pick at least one pet to join in.');
                        return;
                    }
                    postJSON('/api/group', {activity: activity, partner_slots: [...chosen]})
                        .then(data => {
                            if (!data.success) {
                                showPointsToast(data.error || 'Something went wrong');
                                return;
                            }
                            const d = data.data;
                            const lines = [d.text].concat(d.signals || [], (d.conflicts || []).map(c => '⚠️ ' + c.text));
                            showResult(d, lines.join(' '));
                            setTimeout(fetchStatus, 500);
                            openMeet();
                        })
                        .catch(console.error);
                };
                buttons.append(btn);
            });
            p.append(buttons);
        }

        // Cloud: back up and sync pets through a Gochi hub, see pets nearby
        // and shared achievements
        function openCloud() {
            panelMode = 'cloud';
            fetch('/api/cloud').then(r => r.json()).then(data => {
                if (data.success) renderCloud(data.data);
            }).catch(console.error);
        }

        function cloudField(p, label, id, value, placeholder, type) {
            const row = document.createElement('label');
            row.className = 'cloud-field';
            row.textContent = label;
            const input = document.createElement('input');
            input.type = type || 'text';
            input.id = id;
            input.value = value || '';
            input.placeholder = placeholder || '';
            row.append(input);
            p.append(row);
            return input;
        }

        function cloudText(p, text, className) {
            const div = document.createElement('div');
            div.className = className || 'panel-text';
            div.textContent = text;
            p.append(div);
            return div;
        }

        function cloudButton(parent, text, onClick) {
            const btn = document.createElement('button');
            btn.className = 'save-btn';
            btn.textContent = text;
            btn.onclick = onClick;
            parent.append(btn);
            return btn;
        }

        function cloudPost(url, body, done) {
            postJSON(url, body).then(data => {
                showPointsToast(data.success ? (data.data && data.data.text) || 'Done.' : (data.error || 'Something went wrong'));
                if (data.success && done) done(data.data);
                openCloud();
                fetchStatus();
            }).catch(console.error);
        }

        function renderCloud(cloud) {
            const p = panel();
            p.replaceChildren(panelHeader('Cloud'));
            if (cloud.hosting) {
                cloudText(p, '🏠 This server is also a cloud hub. Other players connect to its HTTPS address with the join code.',
                    'panel-text cloud-note');
            }

            if (!cloud.connected) {
                cloudText(p, 'Connect to a Gochi hub to back up your pets, play them on your other devices, share ' +
                    'achievements and meet at playdates. The hub gives you a name like "Blue Otter 42"; your own ' +
                    'name and your pets\' names are never sent to it.');
                const url = cloudField(p, 'Hub address', 'cloudUrl', '', 'hub.example.com or 192.168.1.10:8443');
                const code = cloudField(p, 'Join code (new account)', 'cloudCode', '', 'from whoever runs the hub', 'password');
                const key = cloudField(p, 'Or account key (already have an account)', 'cloudKey', '', 'from your other device', 'password');
                const row = document.createElement('div');
                row.className = 'save-row';
                cloudButton(row, 'Connect', () => cloudPost('/api/cloud/connect',
                    {hub_url: url.value, join_code: code.value, account_key: key.value}));
                p.append(row);
                p.classList.remove('hidden');
                return;
            }

            cloudText(p, 'Connected to ' + cloud.hub_url + ' as ' + cloud.handle + '.');
            if (cloud.fingerprint) {
                cloudText(p, 'Hub certificate fingerprint (ask whoever runs the hub to check it matches theirs): ' +
                    cloud.fingerprint, 'panel-text cloud-key');
            }
            cloudText(p, cloud.last_sync ? 'Last synced ' + new Date(cloud.last_sync).toLocaleString() + '. ' + cloud.last_report :
                'Not synced yet.', 'panel-text cloud-note');
            const actions = document.createElement('div');
            actions.className = 'save-row';
            cloudButton(actions, '🔄 Sync now', () => cloudPost('/api/cloud/sync', {}));
            const keyBtn = cloudButton(actions, 'Show account key', () => {
                postJSON('/api/cloud/key', {}).then(data => {
                    if (!data.success) {
                        showPointsToast(data.error || 'Something went wrong');
                        return;
                    }
                    keyLine.textContent = 'Account key for your other devices (keep it secret): ' + data.data.account_key;
                    keyLine.classList.remove('hidden');
                    keyBtn.disabled = true;
                }).catch(console.error);
            });
            cloudButton(actions, 'Disconnect', () => {
                if (confirm('Disconnect from the hub? Your pets stay on this device and on the hub.')) {
                    cloudPost('/api/cloud/disconnect', {});
                }
            });
            p.append(actions);
            const keyLine = cloudText(p, '', 'panel-text cloud-key hidden');

            renderPlaydateSection(p);

            const title = cloudText(p, 'Sharing achievements', 'section-title');
            title.style.marginTop = '14px';
            cloudText(p, 'When this is on, your pets\' achievements are posted to the hub for everyone on it to see, ' +
                'as your hub name and your pet\'s species (for example "Blue Otter 42\'s cat"). It is off until you ' +
                'turn it on.', 'panel-text cloud-note');
            const row = document.createElement('label');
            row.className = 'method-row';
            const feedBox = document.createElement('input');
            feedBox.type = 'checkbox';
            feedBox.checked = cloud.share_achievements;
            feedBox.onchange = () => cloudPost('/api/cloud/profile', {share_achievements: feedBox.checked},
                () => showPointsToast(feedBox.checked ? 'Sharing achievements.' : 'Not sharing achievements.'));
            row.append(feedBox, ' Share my pets\' achievements');
            p.append(row);

            const feed = cloudText(p, '', 'panel-text');
            p.classList.remove('hidden');
            fetch('/api/cloud/feed').then(r => r.json()).then(data => {
                if (!data.success || data.data.length === 0) return;
                feed.textContent = 'Shared achievements: ' + data.data.slice(0, 10).map(a =>
                    a.icon + ' ' + a.handle + '\'s ' + speciesInfo(a.species).name.toLowerCase() + ': ' + a.name).join(' · ');
            }).catch(console.error);
        }

        // AR playdates: players connected to this hub bring their pets to the
        // same place and see them together in AR
        function renderPlaydateSection(p) {
            const title = cloudText(p, 'AR playdate', 'section-title');
            title.style.marginTop = '14px';
            cloudText(p, 'Meet up with other players on this hub and see your pets together in AR. One of you starts ' +
                'a playdate and tells the others its code; everyone then taps the same spot to mark the center.',
                'panel-text cloud-note');
            const code = cloudField(p, 'Playdate code', 'playdateCode', '', 'to join a friend\'s playdate');
            code.autocapitalize = 'characters';
            const row = document.createElement('div');
            row.className = 'save-row';
            cloudButton(row, '🐾 Start a playdate', () => playdateAction({action: 'start'}));
            cloudButton(row, 'Join', () => playdateAction({action: 'join', code: code.value}));
            p.append(row);
        }

        function playdateAction(body) {
            body.look = lastStatus ? spriteState(lastStatus) : 'idle';
            postJSON('/api/playdate', body).then(data => {
                if (!data.success) {
                    showPointsToast(data.error || 'Something went wrong');
                    return;
                }
                closePanel();
                showPointsToast((body.action === 'start' ? 'Playdate started. Its code is ' : 'Joined playdate ') +
                    data.data.code + '.');
                openAR({playdate: true});
            }).catch(console.error);
        }

        // The player's own statistics, kept on this computer only
        function openStats() {
            panelMode = 'stats';
            fetch('/api/stats').then(r => r.json()).then(data => {
                if (data.success) renderStats(data.data);
            }).catch(console.error);
        }

        function statsTable(p, title, rows) {
            cloudText(p, title, 'section-title');
            if (!rows.length) {
                cloudText(p, 'None yet.', 'panel-text cloud-note');
                return;
            }
            const table = document.createElement('table');
            table.className = 'stats-table';
            for (const [name, value] of rows) {
                const tr = table.insertRow();
                tr.insertCell().textContent = name;
                tr.insertCell().textContent = value;
            }
            p.append(table);
        }

        function renderStats(stats) {
            const p = panel();
            p.replaceChildren(panelHeader('My stats'));
            cloudText(p, 'Your own record of every pet you have raised in this game. It is kept only on this ' +
                'computer and is never sent anywhere.', 'panel-text cloud-note');
            statsTable(p, 'Pets', [
                ['Pets raised', stats.raised],
                ['Adopted', stats.adopted],
                ['Born to your pets', stats.bred],
                ['Alive', stats.alive],
                ['Died', stats.died],
                ['Deleted', stats.deleted],
                ['Times hibernated', stats.hibernations],
            ]);
            statsTable(p, 'How they died', stats.causes.map(c => [c.name, c.count]));
            statsTable(p, 'Care given (' + stats.interactions + ' in all)', stats.care.map(c => [c.name, c.count]));
            statsTable(p, 'Kinds of pet', stats.species.map(c => [c.name, c.count]));
            statsTable(p, 'Each pet', stats.pets.map(pet => {
                let state = pet.alive ? 'alive' : 'died' + (pet.cause_of_death ? ': ' + pet.cause_of_death.toLowerCase() : '');
                if (pet.deleted) state += ', deleted';
                return [speciesIcon(pet.species) + pet.name + ' (' + state + ')',
                    Math.floor(pet.age) + ' days, ' + pet.interactions + ' care'];
            }));
            p.classList.remove('hidden');
        }

        // Players: each has their own pets, household and stats; one plays
        // at a time, and switching saves the one playing
        function playerLabel(a) {
            return a.name + ' #' + a.number;
        }

        function loadPlayers() {
            return fetch('/api/accounts').then(r => r.json()).then(data => {
                if (!data.success) return null;
                const current = data.data.accounts.find(a => a.current);
                document.getElementById('playersBtn').textContent = '👤 ' + (current ? playerLabel(current) : 'Players');
                return data.data;
            });
        }

        function togglePlayers() {
            if (document.getElementById('playersPanel').classList.contains('hidden')) openPlayers();
            else closePlayers();
        }

        function openPlayers() {
            loadPlayers().then(players => {
                if (players) renderPlayers(players);
            }).catch(console.error);
        }

        function closePlayers() {
            const p = document.getElementById('playersPanel');
            p.classList.add('hidden');
            p.replaceChildren();
        }

        function playersPost(url, body, done) {
            postJSON(url, body).then(data => {
                showPointsToast(data.success ? done : (data.error || 'Something went wrong'));
                if (data.success) openPlayers();
            }).catch(console.error);
        }

        // Switching saves the player's game, so the page starts afresh
        // with the other player's
        function switchPlayer(id, btn) {
            btn.disabled = true;
            btn.textContent = 'Opening...';
            postJSON('/api/accounts/switch', {id: id}).then(data => {
                if (data.success) {
                    location.reload();
                    return;
                }
                showPointsToast(data.error || 'Something went wrong');
                openPlayers();
            }).catch(console.error);
        }

        function renderPlayers(players) {
            const p = document.getElementById('playersPanel');
            const head = panelHeader('Players');
            head.querySelector('.panel-close').onclick = closePlayers;
            p.replaceChildren(head);
            cloudText(p, 'Each player has their own pets, Home Stars, house, collection, stats, books and cloud ' +
                'connection. One player plays at a time; switching saves the game of the one playing. Names stay on ' +
                'this computer; a player\'s number picks the name a cloud hub gives them.', 'panel-text cloud-note');
            players.accounts.forEach(a => {
                const row = document.createElement('div');
                row.className = 'player-row';
                const name = document.createElement('span');
                name.className = 'player-name';
                name.textContent = playerLabel(a);
                const actions = document.createElement('div');
                actions.className = 'saved-pet-actions';
                const play = cloudButton(actions, a.current ? 'Playing' : 'Play', () => switchPlayer(a.id, play));
                play.disabled = a.current;
                cloudButton(actions, 'Rename', () => {
                    actions.remove();
                    const field = cloudField(row, 'New name', '', a.name, '', 'text');
                    field.maxLength = 24;
                    const save = document.createElement('div');
                    save.className = 'save-row';
                    cloudButton(save, 'Save name', () => playersPost('/api/accounts/rename',
                        {id: a.id, name: field.value.trim()}, 'Name saved.'));
                    cloudButton(save, 'Cancel', openPlayers);
                    row.append(save);
                    field.focus();
                });
                if (!a.current) {
                    const del = cloudButton(actions, 'Delete', () => {
                        actions.remove();
                        const field = cloudField(row, 'Deleting ' + a.name + ' deletes their pets, books and ' +
                            'everything else of theirs on this computer, for good. Type their name to confirm:',
                            '', '', a.name, 'text');
                        const go = document.createElement('div');
                        go.className = 'save-row';
                        const yes = cloudButton(go, 'Delete for good', () => playersPost('/api/accounts/delete',
                            {id: a.id, confirm_name: field.value.trim()}, 'Player deleted.'));
                        yes.classList.add('delete-btn');
                        cloudButton(go, 'Cancel', openPlayers);
                        row.append(go);
                        field.focus();
                    });
                    del.classList.add('delete-btn');
                }
                row.append(name, actions);
                p.append(row);
            });
            if (players.accounts.length < players.max) {
                const name = cloudField(p, 'New player\'s name', 'newPlayerName', '', 'Name', 'text');
                name.maxLength = 24;
                const row = document.createElement('div');
                row.className = 'save-row';
                cloudButton(row, 'Add player', () => playersPost('/api/accounts/create', {name: name.value.trim()},
                    'Player added. Press Play to play as them.'));
                p.append(row);
            }
            p.classList.remove('hidden');
        }

        // Privacy: learning when the player plays, and the parent PIN
        function openPrivacy() {
            panelMode = 'privacy';
            fetch('/api/privacy').then(r => r.json()).then(data => {
                if (data.success) renderPrivacy(data.data);
            }).catch(console.error);
        }

        function privacyPost(body, done) {
            postJSON('/api/privacy', body).then(data => {
                showPointsToast(data.success ? done : (data.error || 'Something went wrong'));
                openPrivacy();
            }).catch(console.error);
        }

        function renderPrivacy(privacy) {
            const p = panel();
            p.replaceChildren(panelHeader('Privacy'));

            cloudText(p, 'When you play', 'section-title');
            cloudText(p, 'The game learns when you usually play, so it can warn you about what your pet will need ' +
                'before you stop. This stays on this computer.', 'panel-text cloud-note');
            cloudText(p, privacy.play_schedule ? 'You usually play on ' + privacy.play_schedule + '.' :
                'Nothing learned yet.', 'panel-text');
            const learnRow = document.createElement('label');
            learnRow.className = 'method-row';
            const learn = document.createElement('input');
            learn.type = 'checkbox';
            learn.checked = privacy.learn_schedule;
            learn.onchange = () => privacyPost({learn_schedule: learn.checked},
                learn.checked ? 'Learning when you play.' : 'No longer learning when you play.');
            learnRow.append(learn, ' Learn when I usually play');
            p.append(learnRow);
            const eraseRow = document.createElement('div');
            eraseRow.className = 'save-row';
            cloudButton(eraseRow, 'Erase what was learned', () => privacyPost({erase_schedule: true}, 'Erased.'));
            p.append(eraseRow);

            const title = cloudText(p, 'Parent PIN', 'section-title');
            title.style.marginTop = '14px';
            cloudText(p, 'With a parent PIN, the PIN is needed to connect to a cloud hub, change what is shared, ' +
                'start or join a playdate, and change these settings.', 'panel-text cloud-note');
            cloudText(p, privacy.parent_lock ? '🔒 A parent PIN is set.' : 'No parent PIN is set.', 'panel-text');
            const pin = cloudField(p, privacy.parent_lock ? 'New PIN' : 'PIN (4 to 8 digits)', 'newPin', '', '', 'password');
            pin.inputMode = 'numeric';
            const pinRow = document.createElement('div');
            pinRow.className = 'save-row';
            cloudButton(pinRow, privacy.parent_lock ? 'Change PIN' : 'Set PIN',
                () => privacyPost({new_pin: pin.value}, 'Parent PIN saved.'));
            if (privacy.parent_lock) {
                cloudButton(pinRow, 'Remove PIN', () => privacyPost({remove_pin: true}, 'Parent PIN removed.'));
            }
            p.append(pinRow);

            const hols = cloudText(p, 'Holidays', 'section-title');
            hols.style.marginTop = '14px';
            cloudText(p, 'Holiday items are sold for a week before and after each holiday. Turn off any holiday ' +
                '(or all of them): its shop closes and its decorations and cosmetics are hidden until it is turned on ' +
                'again. This is for every player on this computer. Changing this needs the parent PIN, if one is set.',
                'panel-text cloud-note');
            const allRow = document.createElement('label');
            allRow.className = 'setting';
            const all = document.createElement('input');
            all.type = 'checkbox';
            all.checked = privacy.all_holidays_off;
            allRow.append(all, document.createTextNode(' Turn off all holidays'));
            p.append(allRow);
            const holidayBoxes = [];
            (privacy.holidays || []).forEach(h => {
                const row = document.createElement('label');
                row.className = 'setting';
                const box = document.createElement('input');
                box.type = 'checkbox';
                box.checked = h.on;
                box.dataset.id = h.id;
                holidayBoxes.push(box);
                row.append(box, document.createTextNode(' ' + h.icon + ' ' + h.name + ' (' +
                    new Date(h.start + 'T12:00:00').toLocaleDateString(undefined, {month: 'short', day: 'numeric'}) + ')'));
                p.append(row);
            });
            const holidayRow = document.createElement('div');
            holidayRow.className = 'save-row';
            cloudButton(holidayRow, 'Save holidays', () => privacyPost({all_holidays_off: all.checked,
                holidays_off: holidayBoxes.filter(b => !b.checked).map(b => b.dataset.id)}, 'Holidays saved.'));
            p.append(holidayRow);

            const soundTitle = cloudText(p, 'Sound', 'section-title');
            soundTitle.style.marginTop = '14px';
            const soundRow = document.createElement('label');
            soundRow.className = 'panel-text';
            const soundOff = document.createElement('input');
            soundOff.type = 'checkbox';
            soundOff.checked = privacy.sound_off;
            soundOff.onchange = () => privacyPost({sound_off: soundOff.checked},
                soundOff.checked ? 'Sound is off for every player.' : 'Sound is back on.');
            soundRow.append(soundOff, document.createTextNode(' Turn the game\'s sounds off (players can\'t turn them back on)'));
            p.append(soundRow);

            const devices = cloudText(p, 'Play on a phone or tablet', 'section-title');
            devices.style.marginTop = '14px';
            const pairing = cloudText(p, '', 'panel-text');
            p.classList.remove('hidden');
            fetch('/api/pairing').then(r => r.json()).then(data => {
                if (!data.success) {
                    pairing.textContent = data.error || '';
                    return;
                }
                const d = data.data;
                if (!d.on_network) {
                    pairing.textContent = 'The game answers only this computer. To play on a phone or tablet on the ' +
                        'same Wi-Fi (for AR), start it with -addr :8080; each device then needs a code from here.';
                    return;
                }
                pairing.textContent = 'On the phone or tablet, open ' + (d.addresses.join(' or ') || 'this computer\'s address') +
                    ' and enter the code ' + d.code + ' (it works for 10 minutes). Paired devices: ' + d.devices + '.';
                if (d.devices > 0) {
                    const row = document.createElement('div');
                    row.className = 'save-row';
                    cloudButton(row, 'Unpair all devices', () => postJSON('/api/pairing', {forget: true}).then(r => {
                        showPointsToast(r.success ? 'Every device must pair again.' : (r.error || 'Something went wrong'));
                        openPrivacy();
                    }));
                    p.append(row);
                }
            }).catch(console.error);
        }

        function openTricks() {
            panelMode = 'tricks';
            loadCareOptions().then(renderTricks).catch(renderTricks);
        }

        function renderTricks() {
            if (!lastStatus) return;
            const p = panel();
            p.replaceChildren(panelHeader('Tricks'));
            const methods = (careOptions && careOptions.methods) || [];
            if (methods.length > 0) {
                const row = document.createElement('label');
                row.className = 'method-row';
                row.textContent = 'Training method';
                const select = document.createElement('select');
                select.id = 'methodSelect';
                methods.forEach(m => {
                    const option = document.createElement('option');
                    option.value = m.id;
                    option.textContent = m.name + (m.unavailable ? ' (' + m.unavailable.toLowerCase() + ')' : '');
                    option.disabled = !!m.unavailable;
                    select.append(option);
                });
                if (methods.some(m => m.id === trainingMethod && !m.unavailable)) select.value = trainingMethod;
                else select.value = trainingMethod = 'praise';
                select.onchange = () => { trainingMethod = select.value; };
                row.append(select);
                p.append(row);
            }
            lastStatus.tricks.forEach(trick => {
                const row = document.createElement('div');
                row.className = 'trick-row';
                const name = document.createElement('span');
                name.className = 'trick-name';
                name.textContent = trick.icon + ' ' + trick.name + (trick.rusty ? ' (rusty)' : '') + (trick.locked ? ' 🔒' : '');
                if (trick.rusty) name.title = lastStatus.name + ' needs practice before doing this trick again';
                if (trick.locked) name.title = trick.locked;
                const bar = document.createElement('div');
                bar.className = 'trick-bar';
                const fill = document.createElement('div');
                fill.className = 'trick-fill';
                fill.style.width = Math.round(trick.skill * 100) + '%';
                bar.append(fill);
                const btn = document.createElement('button');
                btn.className = 'save-btn';
                btn.textContent = trick.learned ? 'Do it!' : (trick.rusty ? 'Practice' : 'Train');
                btn.onclick = () => doTrick(trick.id, trick.learned);
                if (trick.locked) {
                    btn.disabled = true;
                    btn.title = trick.locked;
                }
                row.append(name, bar, btn);
                if (trick.locked) {
                    const why = document.createElement('div');
                    why.className = 'cloud-note trick-locked';
                    why.textContent = trick.locked;
                    row.append(why);
                }
                p.append(row);
            });
            p.classList.remove('hidden');
        }

        function doTrick(id, perform) {
            postJSON(perform ? '/api/perform' : '/api/train', {trick: id, method: trainingMethod})
                .then(data => {
                    if (data.success) {
                        showResult(data.data, data.data.text);
                        fetchStatus();
                        if (!perform) loadCareOptions().then(() => { if (panelMode === 'tricks') renderTricks(); });
                    } else {
                        showPointsToast(data.error || 'Something went wrong');
                    }
                })
                .catch(console.error);
        }

        function startHideSeek() {
            postJSON('/api/hideseek/start')
                .then(data => {
                    if (!data.success) {
                        showPointsToast(data.error || 'Something went wrong');
                        return;
                    }
                    panelMode = 'hideseek';
                    hideSpots = data.data.spots;
                    if (lastStatus) lastStatus.hide_and_seek = true;
                    renderHideSeek(data.data.text, false);
                })
                .catch(console.error);
        }

        function renderHideSeek(text, over) {
            const p = panel();
            p.replaceChildren(panelHeader('Hide & Seek'));
            const message = document.createElement('div');
            message.className = 'panel-text';
            message.id = 'hideSeekText';
            message.textContent = text;
            p.append(message);

            if (over) {
                const again = document.createElement('button');
                again.className = 'save-btn';
                again.textContent = 'Play again';
                again.onclick = startHideSeek;
                const wrap = document.createElement('div');
                wrap.style.textAlign = 'center';
                wrap.append(again);
                p.append(wrap);
            } else {
                const spots = document.createElement('div');
                spots.className = 'spots';
                hideSpots.forEach((spot, i) => {
                    const btn = document.createElement('button');
                    btn.className = 'spot-btn';
                    btn.textContent = spot.icon + ' ' + spot.name;
                    btn.onclick = () => guessSpot(i, btn);
                    spots.append(btn);
                });
                p.append(spots);

                const stop = document.createElement('button');
                stop.className = 'back-btn';
                stop.textContent = 'Stop playing (no points)';
                stop.onclick = closePanel;
                const wrap = document.createElement('div');
                wrap.style.textAlign = 'center';
                wrap.append(stop);
                p.append(wrap);
            }
            p.classList.remove('hidden');
        }

        function guessSpot(i, btn) {
            btn.disabled = true;
            postJSON('/api/hideseek/guess', {spot: i})
                .then(data => {
                    if (!data.success) {
                        showPointsToast(data.error || 'Something went wrong');
                        return;
                    }
                    if (data.data.over) {
                        if (lastStatus) lastStatus.hide_and_seek = false;
                        renderHideSeek(data.data.text, true);
                        showResult(data.data);
                        fetchStatus();
                    } else {
                        document.getElementById('hideSeekText').textContent = data.data.text;
                    }
                })
                .catch(console.error);
        }

        // Exploring: with only the park discovered and nothing else to
        // choose, the pet goes straight there; otherwise the player picks
        // a place, and can choose a long trip or a friend to take along
        function explore() {
            Promise.all([fetch('/api/explore/places').then(r => r.json()), fetch('/api/saves').then(r => r.json())])
                .then(([data, saves]) => {
                    if (!data.success) {
                        showPointsToast(data.error || 'Something went wrong');
                        return;
                    }
                    const places = data.data.places;
                    const species = (lastStatus && lastStatus.species) || 'cat';
                    const friends = ((saves.success && saves.data.pets) || []).filter(pet =>
                        pet.slot !== saves.data.active_slot && pet.is_alive && !pet.unreadable && !pet.hibernating &&
                        (pet.species || 'cat') === species);
                    const discovered = places.filter(place => place.discovered);
                    if (discovered.length <= 1 && data.data.long_trip && !friends.length) exploreTo('park');
                    else renderPlaces(places, data.data.long_trip, friends);
                })
                .catch(console.error);
        }

        function renderPlaces(places, longTrip, friends) {
            panelMode = 'places';
            const p = panel();
            p.replaceChildren(panelHeader('Explore'));
            const name = (lastStatus && lastStatus.name) || 'Your pet';
            const cat = lastStatus && lastStatus.species !== 'dog';
            const intro = document.createElement('div');
            intro.className = 'panel-text';
            intro.textContent = 'Where should ' + name + ' explore? Wilder places hold more to find and more danger. ' +
                'What ' + name + ' finds is taken from the place and grows back over time.' +
                (cat ? ' Cats go exploring on a harness walk with you, the safe way for a cat to be outdoors.' : '');
            p.append(intro);

            // A long trip and a friend
            const options = el('div', 'show-row');
            const length = el('select', 'show-level');
            length.setAttribute('aria-label', 'Trip length');
            [['short', 'Short trip'], ['long', 'Long trip (two chances to find something)']].forEach(([v, text]) => {
                const o = el('option', '', text);
                o.value = v;
                o.disabled = v === 'long' && !!longTrip;
                length.append(o);
            });
            const friend = el('select', 'show-level');
            friend.setAttribute('aria-label', 'Friend');
            const alone = el('option', '', 'On its own');
            alone.value = '';
            friend.append(alone);
            friends.forEach(pet => {
                const o = el('option', '', 'With ' + pet.name);
                o.value = pet.slot;
                friend.append(o);
            });
            options.append(length, friend);
            p.append(options);
            if (longTrip) cloudText(p, 'Long trips: ' + longTrip, 'panel-text cloud-note');

            places.forEach(place => {
                const row = document.createElement('div');
                row.className = 'trick-row';
                const label = document.createElement('span');
                label.className = 'trick-name';
                label.textContent = place.icon + ' ' + place.name;
                if (!place.discovered || place.unavailable) {
                    const info = document.createElement('span');
                    info.style.cssText = 'flex: 1; font-size: 0.8em; color: #777;';
                    info.textContent = place.discovered ? place.unavailable : 'Discovered after ' + place.trips_needed + ' trips';
                    row.append(label, info);
                } else {
                    const bar = document.createElement('div');
                    bar.className = 'trick-bar';
                    bar.title = Math.round(place.resources * 100) + '% left to find' +
                        (place.danger >= 0.25 ? ' · risky' : '');
                    const fill = document.createElement('div');
                    fill.className = 'trick-fill';
                    fill.style.width = Math.round(place.resources * 100) + '%';
                    bar.append(fill);
                    const btn = document.createElement('button');
                    btn.className = 'save-btn';
                    btn.textContent = 'Go';
                    btn.onclick = () => exploreTo(place.id, length.value === 'long', friend.value);
                    row.append(label, bar, btn);
                }
                p.append(row);
            });
            p.classList.remove('hidden');
        }

        function exploreTo(place, long, friendSlot) {
            postJSON('/api/explore', {place: place, long: !!long, friend_slot: friendSlot || ''})
                .then(data => {
                    if (data.success) {
                        closePanel();
                        showPointsToast(data.data.text);
                        fetchStatus();
                    } else {
                        showPointsToast(data.error || 'Something went wrong');
                    }
                })
                .catch(console.error);
        }

        // AR: places the pet in the room through the camera (or a drawn
        // room on devices without AR); see web/static/js/ar-link.js
        let arReady = false;
        async function openAR(mode) {
            if (typeof GochiAR === 'undefined' || typeof GochiARLink === 'undefined') {
                showPointsToast('AR could not be loaded.');
                return;
            }
            if (!arReady) {
                await GochiAR.init();
                arReady = true;
            }
            document.body.classList.add('ar-open');
            const started = await GochiARLink.start({
                onMessage: text => showPointsToast(text),
                onCare: result => { showResult(result, result.text); setTimeout(fetchStatus, 500); },
                petSprite: () => lastStatus ? petArt(lastStatus, spriteState(lastStatus), 200) : '',
                petLook: () => lastStatus ? spriteState(lastStatus) : 'idle',
                spriteFor: mate => GochiArt.draw({species: mate.species, look: mate.typical, state: mate.look,
                    outfit: mate.outfit, size: 200, label: mate.name}),
                statusText: () => lastStatus ? lastStatus.name + ' ' + lastStatus.activity : '',
                onEnd: () => { document.body.classList.remove('ar-open'); fetchStatus(); },
                playdate: !!(mode && mode.playdate)
            });
            if (!started) document.body.classList.remove('ar-open');
        }

        function openBreeding() {
            panelMode = 'breeding';
            fetch('/api/breed/partners')
                .then(r => r.json())
                .then(data => {
                    if (!data.success) {
                        showPointsToast(data.error || 'Something went wrong');
                        return;
                    }
                    renderBreeding(data.data);
                })
                .catch(console.error);
        }

        function renderBreeding(partners) {
            const p = panel();
            p.replaceChildren(panelHeader('Breed'));
            const name = (lastStatus && lastStatus.name) || 'Your pet';

            if (partners.length === 0) {
                const none = document.createElement('div');
                none.className = 'panel-text';
                none.textContent = 'Create or load another pet first. ' + name + ' needs a partner from your saved pets.';
                p.append(none);
                p.classList.remove('hidden');
                return;
            }

            const input = document.createElement('input');
            input.type = 'text';
            input.id = 'babyNameInput';
            input.maxLength = 24;
            input.placeholder = "Baby's name";
            input.style.cssText = 'width: 100%; padding: 8px 10px; border: 2px solid #ddd; border-radius: 10px; margin-bottom: 10px;';
            p.append(input);

            const env = lastStatus && lastStatus.environment;
            if (env && (env.breeding_season || env.season === 'Winter' || env.season === 'Fall')) {
                const season = document.createElement('div');
                season.className = 'panel-text';
                season.textContent = env.breeding_season ? 'It\'s ' + env.season.toLowerCase() + ': babies are more likely now.' :
                    'It\'s ' + env.season.toLowerCase() + ': babies are less likely until spring.';
                p.append(season);
            }

            partners.forEach(partner => {
                const row = document.createElement('div');
                row.className = 'trick-row';
                const label = document.createElement('span');
                label.className = 'trick-name';
                label.textContent = partner.name;
                const info = document.createElement('span');
                info.style.cssText = 'flex: 1; font-size: 0.8em; color: #777;';
                info.textContent = Math.round(partner.compatibility * 100) + '% compatible' +
                    (partner.reason ? ' · ' + partner.reason : '');
                const btn = document.createElement('button');
                btn.className = 'save-btn';
                btn.textContent = 'Breed';
                btn.disabled = !!partner.reason;
                btn.onclick = () => breedWith(partner.slot, btn);
                row.append(label, info, btn);
                p.append(row);
            });
            p.classList.remove('hidden');
        }

        function breedWith(slot, btn) {
            btn.disabled = true;
            postJSON('/api/breed', {partner_slot: slot, name: document.getElementById('babyNameInput').value})
                .then(data => {
                    showPointsToast(data.success ? data.data.text : (data.error || 'Something went wrong'));
                    openBreeding();
                    fetchStatus();
                })
                .catch(console.error);
        }

        function adventureKey(status) {
            return status.slot + ':' + status.last_adventure.returned_age;
        }

        function dismissAdventure() {
            if (lastStatus && lastStatus.last_adventure) {
                dismissedAdventure = adventureKey(lastStatus);
                try { localStorage.setItem('gochiDismissedAdventure', dismissedAdventure); } catch (e) {}
            }
            document.getElementById('adventureBanner').classList.add('hidden');
        }

        // Play time, from real seconds at the default pace
        function formatPlayTime(seconds) {
            const hours = seconds / 3600;
            return hours >= 2 ? 'about ' + Math.round(hours).toLocaleString() + ' hours' : formatWait(seconds);
        }

        function givePotion() {
            postJSON('/api/potion')
                .then(data => {
                    showPointsToast(data.success ? data.data.text : (data.error || 'Something went wrong'));
                    fetchStatus();
                })
                .catch(console.error);
        }

        function formatWait(seconds) {
            return seconds >= 90 ? 'about ' + Math.round(seconds / 60) + ' minutes' : seconds + ' seconds';
        }

        function updateActivities(status) {
            lastStatus = status;
            const away = status.exploring;
            const busy = status.hibernating || away;

            const banner = document.getElementById('exploreBanner');
            document.getElementById('exploreText').textContent = away ?
                '🧭 ' + status.name + ' is out exploring. Back in ' + formatWait(status.exploring_seconds) + '.' : '';
            banner.classList.toggle('hidden', !away);

            document.querySelectorAll('.activity-btn').forEach(b => { b.disabled = busy; });
            if (busy && panelMode && panelMode !== 'guide') closePanel();
            if (panelMode === 'tricks') renderTricks();

            const adventure = status.last_adventure;
            const adventureBanner = document.getElementById('adventureBanner');
            if (adventure && adventureKey(status) !== dismissedAdventure) {
                const text = document.getElementById('adventureText');
                text.replaceChildren();
                [adventure.text, adventure.reaction_text, formatChanges(adventure.changes || []),
                 adventure.points_earned > 0 ? '+' + adventure.points_earned + ' Friendship Points!' : '']
                    .filter(Boolean).forEach(line => {
                        const div = document.createElement('div');
                        div.textContent = line;
                        text.append(div);
                    });
                adventureBanner.classList.toggle('hurt', adventure.outcome === 'injured');
                adventureBanner.classList.toggle('plain', adventure.outcome === 'nothing');
                adventureBanner.querySelector('button').textContent = adventure.outcome === 'treasure' ? 'Yay!' : 'OK';
                adventureBanner.classList.remove('hidden');
            } else {
                adventureBanner.classList.add('hidden');
            }

            const treasures = (status.souvenirs || []).map(s => s.icon + (s.count > 1 ? '×' + s.count : ''));
            const el = document.getElementById('aboutTreasures');
            el.replaceChildren();
            const label = document.createElement('span');
            label.className = 'about-label';
            label.textContent = 'Treasures: ';
            el.append(label, treasures.length ? treasures.join(' ') : 'none yet');
        }

        function savePet() {
            const btn = document.getElementById('saveBtn');
            const statusEl = document.getElementById('saveStatus');
            btn.disabled = true;

            postJSON('/api/save', {})
                .then(data => {
                    if (data.success) {
                        statusEl.textContent = 'Saved at ' + new Date(data.data.saved_at).toLocaleTimeString();
                    } else {
                        statusEl.textContent = 'Save failed: ' + (data.error || 'unknown error');
                    }
                })
                .catch(() => { statusEl.textContent = 'Save failed: connection error'; })
                .finally(() => { btn.disabled = false; });
        }

        // Handle Enter key in name input
        document.getElementById('petNameInput').addEventListener('keypress', function(e) {
            if (e.key === 'Enter') {
                e.preventDefault();
                createPet();
            }
        });

        // Initial check for existing pet; keep polling so the server knows
        // whether the player is present
        loadSpecies().then(fetchStatus);
        loadPlayers().catch(console.error);
        startUpdates();
        watchBattery();
    </script>
    <!-- Adds to functions in the script above -->
    <script src="/static/js/help.js?v={{static}}"></script>
</body>
</html>`
	_, _ = w.Write([]byte(strings.ReplaceAll(html, "{{static}}", s.staticVersion))) // Fails only if the client has gone
}

// autosaveLoop saves the pet every interval until ctx is done
func (s *Server) autosaveLoop(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.autosave()
		}
	}
}

// autosave saves the active pet, the play schedule, the statistics and the
// household
func (s *Server) autosave() {
	s.accountMu.RLock()
	defer s.accountMu.RUnlock()
	if _, err := s.saveNow(); err != nil {
		log.Printf("Autosave failed: %v", err)
	}
	if err := s.saveOwner(); err != nil {
		log.Printf("Could not save the play schedule: %v", err)
	}
	if s.stats != nil {
		if err := s.stats.save(); err != nil {
			log.Printf("Could not save your statistics: %v", err)
		}
	}
	if s.house != nil {
		if err := s.house.save(); err != nil {
			log.Printf("Could not save the household: %v", err)
		}
	}
}

// saveNow writes the current pet to disk. It reports whether a pet was saved;
// with no pet (or saving disabled) it does nothing.
func (s *Server) saveNow() (bool, error) {
	if s.store == nil {
		return false, nil
	}

	s.saveMu.Lock()
	defer s.saveMu.Unlock()

	// Serialize under the lock so the pet is not updated mid-save; write
	// the file after releasing it
	s.mu.RLock()
	if s.pet == nil {
		s.mu.RUnlock()
		return false, nil
	}
	slot, pet := s.slot, s.pet
	petData, err := s.pet.Save()
	record := s.pet.SaveRecord()
	s.mu.RUnlock()
	if err != nil {
		return false, err
	}

	if err := s.store.SaveWithRecord(slot, petData, &record); err != nil {
		return false, err
	}
	if err := s.store.books.write(pet); err != nil {
		log.Printf("Could not write %s's book: %v", pet.Name, err)
	}
	return true, nil
}

// saveActiveLocked writes the active pet, if any, to its slot.
// Caller must hold s.saveMu and s.mu.
func (s *Server) saveActiveLocked() error {
	if s.store == nil || s.pet == nil {
		return nil
	}
	return s.store.SavePet(s.slot, s.pet)
}

// switchPet saves the active pet, then makes the pet returned by next the
// active one. If the active pet cannot be saved, nothing changes, so a
// pet is never lost by switching. If save is true the new pet is saved
// immediately (for newly created pets).
func (s *Server) switchPet(next func() (*core.DigitalPet, string, error), save bool) error {
	s.saveMu.Lock()
	defer s.saveMu.Unlock()
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.saveActiveLocked(); err != nil {
		return fmt.Errorf("could not save %s: %w", s.pet.Name, err)
	}
	s.observePet(s.pet, false) // Care since the last tick

	pet, slot, err := next()
	if err != nil {
		return err
	}

	now := time.Now()
	s.pet, s.slot = pet, slot
	s.clock.SetPaused(true, now) // The new pet's time starts from now
	s.manualPause = false
	s.markActiveLocked(now)

	if save {
		if err := s.saveActiveLocked(); err != nil {
			log.Printf("Failed to save %s: %v", pet.Name, err)
		}
	}
	s.observePet(pet, false) // Once it is saved, so the statistics never count a pet that wasn't
	s.noteAchievementsLocked(pet)
	return nil
}

// createPet saves the active pet and replaces it with a new one
func (s *Server) createPet(name string, species core.Species, breed string, randomize, hibernation bool) error {
	if name == "" {
		name = "Gochi"
	}

	err := s.switchPet(func() (*core.DigitalPet, string, error) {
		if s.store != nil && s.store.Full() {
			return nil, "", errTooManyPets
		}
		pet := core.NewDigitalPet(name, localOwner)
		if randomize || breed != "" {
			pet = core.NewPetOfBreed(name, localOwner, species, breed)
		}
		pet.HibernationEnabled = hibernation
		pet.Species = species
		slot := sanitizeSlot(string(pet.ID))
		if s.store != nil {
			slot = s.store.NewSlot(pet)
		}
		return pet, slot, nil
	}, true)
	if err == nil {
		log.Printf("Created new pet: %s", name)
	}
	return err
}

// handleListSaves handles GET /api/saves - lists saved pets
func (s *Server) handleListSaves(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, APIResponse{Success: false, Error: "Method not allowed"})
		return
	}

	pets := []savedPet{}
	if s.store != nil {
		var err error
		if pets, err = s.store.List(); err != nil {
			log.Printf("Listing saves failed: %v", err)
			writeJSON(w, http.StatusInternalServerError, APIResponse{Success: false, Error: "Could not list saved pets"})
			return
		}
	}

	s.mu.RLock()
	active := ""
	if s.pet != nil {
		active = s.slot
	}
	s.mu.RUnlock()

	writeJSON(w, http.StatusOK, APIResponse{Success: true, Data: map[string]interface{}{
		"pets":        pets,
		"active_slot": active,
	}})
}

// handleLoad handles POST /api/load - saves the active pet and switches to a saved one
func (s *Server) handleLoad(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, APIResponse{Success: false, Error: "Method not allowed"})
		return
	}

	var req struct {
		Slot string `json:"slot"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || !ValidSlot(req.Slot) {
		writeJSON(w, http.StatusBadRequest, APIResponse{Success: false, Error: "Invalid save slot"})
		return
	}
	if s.store == nil {
		writeJSON(w, http.StatusBadRequest, APIResponse{Success: false, Error: "Saving is disabled"})
		return
	}

	err := s.switchPet(func() (*core.DigitalPet, string, error) {
		pet, path, warning, err := s.store.Load(req.Slot)
		if err != nil {
			return nil, "", err
		}
		log.Printf("Loaded %s from %s", pet.Name, path)
		if warning != nil {
			log.Printf("Warning: %v", warning)
		}
		return pet, req.Slot, nil
	}, false)

	switch {
	case errors.Is(err, os.ErrNotExist):
		writeJSON(w, http.StatusNotFound, APIResponse{Success: false, Error: "No saved pet in that slot"})
	case err != nil:
		log.Printf("Load failed: %v", err)
		writeJSON(w, http.StatusInternalServerError, APIResponse{Success: false, Error: "Could not load pet"})
	default:
		s.mu.RLock()
		data := s.getPetData()
		s.mu.RUnlock()
		writeJSON(w, http.StatusOK, APIResponse{Success: true, Data: data})
	}
}

// shopDataLocked returns the shop for the active pet: equipment and
// decorations, cosmetics, and the holidays whose items are on sale. Caller
// must hold s.mu.
func (s *Server) shopDataLocked() map[string]interface{} {
	open := append([]core.Holiday{}, core.OpenHolidays()...)
	return map[string]interface{}{
		"friendship_points": s.pet.FriendshipPoints,
		"points_earned":     s.pet.PointsEarned,
		"items":             s.pet.ShopOffers(),
		"cosmetics":         s.pet.CosmeticOffers(),
		"holidays":          open,
		"season":            core.RealSeason().String(),
		"holiday_bought":    s.holidayBought,
	}
}

// handleShop handles GET /api/shop - lists items and the active pet's points
func (s *Server) handleShop(w http.ResponseWriter, r *http.Request) {
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
	writeJSON(w, http.StatusOK, APIResponse{Success: true, Data: s.shopDataLocked()})
}

// handleBuy handles POST /api/shop/buy - {"item": id} buys an item for the
// active pet with its Friendship Points
func (s *Server) handleBuy(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, APIResponse{Success: false, Error: "Method not allowed"})
		return
	}

	var req struct {
		Item string `json:"item"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, APIResponse{Success: false, Error: "Invalid request"})
		return
	}

	s.buyLocked(w, req.Item, func() error { return s.pet.BuyItem(req.Item) })
}

// holidayLimitMessage explains the one holiday item per session
const holidayLimitMessage = "One holiday item per visit: come back next time for another."

// buyLocked buys something for the active pet with buy, enforcing one
// holiday item per session, and writes the shop or the reason it failed
func (s *Server) buyLocked(w http.ResponseWriter, id string, buy func() error) {
	s.mu.Lock()
	if s.pet == nil {
		s.mu.Unlock()
		writeJSON(w, http.StatusNotFound, APIResponse{Success: false, Error: "No pet found"})
		return
	}
	s.markActiveLocked(time.Now())
	name := s.pet.Name
	holiday := core.HolidayOf(id) != ""
	var err error
	if holiday && s.holidayBought {
		err = errHolidayLimit
	} else if err = buy(); err == nil && holiday {
		s.holidayBought = true
	}
	data := s.shopDataLocked()
	data["supplies"] = s.pet.SupplyOffers()
	s.mu.Unlock()

	if err != nil {
		writeShopError(w, name, err)
		return
	}
	if _, err := s.saveNow(); err != nil {
		log.Printf("Failed to save after purchase: %v", err)
	}
	writeJSON(w, http.StatusOK, APIResponse{Success: true, Data: data})
}

// errHolidayLimit is returned for a second holiday item in a session
var errHolidayLimit = errors.New("one holiday item per session")

// writeShopError explains why something couldn't be bought
func writeShopError(w http.ResponseWriter, name string, err error) {
	status, message := http.StatusConflict, name+" can't use items right now"
	switch {
	case errors.Is(err, core.ErrUnknownItem), errors.Is(err, core.ErrUnknownSupply), errors.Is(err, core.ErrUnknownCosmetic):
		status, message = http.StatusBadRequest, "Unknown item"
	case errors.Is(err, core.ErrAlreadyOwned):
		message = name + " already has this"
	case errors.Is(err, core.ErrNotEnoughPoints):
		message = "Not enough Friendship Points"
	case errors.Is(err, core.ErrItemLocked), errors.Is(err, core.ErrSupplyLocked):
		message = name + " needs to earn more Friendship Points to unlock this"
	case errors.Is(err, core.ErrOutOfSeason):
		message = "This is sold only in its season"
	case errors.Is(err, core.ErrNotHolidayTime):
		message = "This is sold only around its holiday"
	case errors.Is(err, core.ErrWrongSpecies):
		message = "This is for another kind of pet"
	case errors.Is(err, core.ErrPantryFull):
		message = "The pantry is full"
	case errors.Is(err, errHolidayLimit):
		message = holidayLimitMessage
	}
	writeJSON(w, status, APIResponse{Success: false, Error: message})
}

// handleHibernation handles POST /api/hibernation - {"enabled": true|false}
// chooses whether the active pet hibernates instead of dying from neglect
func (s *Server) handleHibernation(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, APIResponse{Success: false, Error: "Method not allowed"})
		return
	}

	var req struct {
		Enabled bool `json:"enabled"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, APIResponse{Success: false, Error: "Invalid request"})
		return
	}

	s.mu.Lock()
	if s.pet == nil {
		s.mu.Unlock()
		writeJSON(w, http.StatusNotFound, APIResponse{Success: false, Error: "No pet found"})
		return
	}
	s.markActiveLocked(time.Now())
	s.pet.SetHibernationEnabled(req.Enabled)
	s.mu.Unlock()

	if _, err := s.saveNow(); err != nil {
		log.Printf("Failed to save hibernation setting: %v", err)
	}
	writeJSON(w, http.StatusOK, APIResponse{Success: true, Data: map[string]bool{"hibernation_enabled": req.Enabled}})
}

// handleRevive handles POST /api/revive - wakes the active pet from hibernation
func (s *Server) handleRevive(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, APIResponse{Success: false, Error: "Method not allowed"})
		return
	}

	s.mu.Lock()
	if s.pet == nil {
		s.mu.Unlock()
		writeJSON(w, http.StatusBadRequest, APIResponse{Success: false, Error: "No pet found"})
		return
	}
	s.markActiveLocked(time.Now())
	name := s.pet.Name
	err := s.pet.Revive()
	s.mu.Unlock()

	switch {
	case errors.Is(err, core.ErrNotHibernating):
		writeJSON(w, http.StatusConflict, APIResponse{Success: false, Error: name + " is not hibernating"})
	case err != nil:
		writeJSON(w, http.StatusBadRequest, APIResponse{Success: false, Error: "Could not revive " + name})
	default:
		log.Printf("Revived %s", name)
		if _, err := s.saveNow(); err != nil {
			log.Printf("Failed to save after revive: %v", err)
		}
		writeJSON(w, http.StatusOK, APIResponse{Success: true})
	}
}

// handleDelete handles POST /api/delete - permanently deletes a saved pet
// and its backups. The request must include the pet's exact name as
// confirm_name; the page asks for it as the second confirmation.
func (s *Server) handleDelete(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, APIResponse{Success: false, Error: "Method not allowed"})
		return
	}

	var req struct {
		Slot        string `json:"slot"`
		ConfirmName string `json:"confirm_name"`
		ParentPIN   string `json:"parent_pin"`
		DeleteBook  bool   `json:"delete_book"` // Delete its keepsake book too
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || !ValidSlot(req.Slot) {
		writeJSON(w, http.StatusBadRequest, APIResponse{Success: false, Error: "Invalid save slot"})
		return
	}
	if !s.parentAllows(w, req.ParentPIN) {
		return
	}
	if s.store == nil {
		writeJSON(w, http.StatusBadRequest, APIResponse{Success: false, Error: "Saving is disabled"})
		return
	}

	// Hold both locks so an autosave cannot write the pet back
	s.saveMu.Lock()
	defer s.saveMu.Unlock()
	s.mu.Lock()
	defer s.mu.Unlock()

	active := s.pet != nil && s.slot == req.Slot

	// The active pet may have been renamed since it was last saved
	var name, petID string
	if active {
		name, petID = s.pet.Name, string(s.pet.ID)
	} else {
		summary, err := s.store.Summary(req.Slot)
		switch {
		case errors.Is(err, os.ErrNotExist):
			writeJSON(w, http.StatusNotFound, APIResponse{Success: false, Error: "No saved pet in that slot"})
			return
		case err != nil:
			name = req.Slot // Unreadable save: confirm with the slot name
		default:
			name, petID = summary.Name, summary.ID
		}
	}

	if name == "" {
		name = req.Slot
	}
	if strings.TrimSpace(req.ConfirmName) != name {
		writeJSON(w, http.StatusBadRequest, APIResponse{Success: false, Error: "Type the pet's name exactly to confirm"})
		return
	}

	// The statistics first: if the game stops before the save is
	// removed, the pet is counted again when the game starts
	if active {
		s.observePet(s.pet, false) // Care since the last tick
	}
	if s.stats != nil && petID != "" {
		s.stats.deleted(types.PetID(petID))
		if err := s.stats.save(); err != nil {
			log.Printf("Could not save your statistics: %v", err)
		}
	}
	if err := s.store.Delete(req.Slot); err != nil && !errors.Is(err, os.ErrNotExist) {
		log.Printf("Delete failed: %v", err)
		if s.stats != nil && petID != "" {
			s.stats.undelete(types.PetID(petID))
		}
		writeJSON(w, http.StatusInternalServerError, APIResponse{Success: false, Error: "Could not delete pet"})
		return
	}
	if req.DeleteBook && petID != "" {
		if err := s.store.books.remove(petID); err != nil {
			log.Printf("Could not delete %s's book: %v", name, err)
		}
	}
	if active {
		s.pet, s.slot = nil, ""
	}

	log.Printf("Deleted %s (%s)", name, req.Slot)
	if petID != "" {
		go s.cloud.forget(context.Background(), petID) // Other devices stop syncing it
	}
	writeJSON(w, http.StatusOK, APIResponse{Success: true, Data: map[string]interface{}{
		"deleted":    req.Slot,
		"was_active": active,
	}})
}

// handlePause handles POST /api/pause - {"paused": true|false} stops or
// resumes time for the active pet
func (s *Server) handlePause(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, APIResponse{Success: false, Error: "Method not allowed"})
		return
	}

	var req struct {
		Paused bool `json:"paused"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, APIResponse{Success: false, Error: "Invalid request"})
		return
	}

	s.mu.Lock()
	now := time.Now()
	s.manualPause = req.Paused
	s.markActiveLocked(now)
	reason := s.pauseReasonLocked(now)
	s.mu.Unlock()

	writeJSON(w, http.StatusOK, APIResponse{Success: true, Data: map[string]interface{}{
		"paused":       reason != "",
		"pause_reason": reason,
	}})
}

// handleSave handles POST /api/save - saves the pet immediately
func (s *Server) handleSave(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, APIResponse{Success: false, Error: "Method not allowed"})
		return
	}

	saved, err := s.saveNow()
	if err != nil {
		log.Printf("Manual save failed: %v", err)
		writeJSON(w, http.StatusInternalServerError, APIResponse{Success: false, Error: "Save failed"})
		return
	}
	if !saved {
		writeJSON(w, http.StatusBadRequest, APIResponse{Success: false, Error: "No pet to save"})
		return
	}

	writeJSON(w, http.StatusOK, APIResponse{Success: true, Data: map[string]string{
		"saved_at": time.Now().Format(time.RFC3339),
	}})
}

// handleStatus handles GET /api/status. The page adds ?active=1 when the
// player has used it recently, which counts as the player being present.
func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	if r.URL.Query().Get("active") == "1" {
		s.markActiveLocked(now)
	}

	if s.pet == nil {
		writeJSON(w, http.StatusNotFound, APIResponse{Success: false, Error: "No pet found"})
		return
	}

	status := s.pet.GetCurrentStatus()

	petStatus := PetStatus{
		ID:          s.pet.ID,
		Name:        s.pet.Name,
		Age:         status.Age,
		IsAlive:     status.IsAlive,
		Behavior:    s.pet.CurrentBehavior.String(),
		Activity:    s.pet.Activity(),
		Request:     s.pet.CurrentRequest(),
		Mood:        s.pet.Emotions.GetMoodDescription(),
		Health:      s.pet.Biology.Vitals.Health,
		Energy:      s.pet.Biology.Vitals.Energy,
		Happiness:   s.pet.Biology.Vitals.Happiness,
		Hunger:      1.0 - s.pet.Biology.Vitals.Nutrition,
		Water:       s.pet.Biology.Vitals.Hydration,
		Wellbeing:   status.Wellbeing,
		Personality: s.pet.PersonalityDescription(),
		Slot:        s.slot,
		Hibernating: status.Hibernating,

		FriendshipPoints: s.pet.FriendshipPoints,
		Items:            s.pet.OwnedItems(),
		Effects:          s.pet.ActiveEffects(),
	}
	if petStatus.Items == nil {
		petStatus.Items = []core.ShopItem{}
	}
	petStatus.Needs = s.pet.PsychologicalNeeds()
	petStatus.Environment = s.pet.EnvironmentStatus()
	petStatus.CareTips = careTips()
	petStatus.Generation = s.pet.Generation()
	petStatus.HibernationOn = s.pet.HibernationEnabled
	petStatus.CauseOfDeath = s.pet.Biology.CauseOfDeath
	petStatus.Tricks = s.pet.Tricks()
	petStatus.Exploring = s.pet.IsExploring()
	petStatus.ExploringSeconds = gameDaysToSeconds(s.pet.ExploringRemaining())
	petStatus.HideAndSeek = s.pet.PlayingHideAndSeek()
	petStatus.Souvenirs = s.pet.SouvenirCollection()
	petStatus.Species = s.pet.GetSpecies()
	petStatus.Body = s.pet.BodySystems()
	petStatus.LivingArea = s.pet.GetLivingArea()
	petStatus.Illness = s.pet.IllnessName()
	petStatus.Symptoms = s.pet.Symptoms()
	petStatus.GuideTopics = []string{}
	for _, symptom := range petStatus.Symptoms {
		if symptom.Topic != "" && !slices.Contains(petStatus.GuideTopics, symptom.Topic) {
			petStatus.GuideTopics = append(petStatus.GuideTopics, symptom.Topic)
		}
	}
	petStatus.Checklist = s.pet.DailyChecklist()
	petStatus.Goals, petStatus.GoalLevel = s.pet.DailyGoals()
	petStatus.Weekly = s.pet.WeekGoal()
	_, petStatus.Titles = s.pet.ShowRecord()
	petStatus.Forecasts = s.forecastsLocked(now)
	petStatus.ThingsToDo = s.pet.ThingsToDo()
	petStatus.Outfit = s.pet.Worn()
	petStatus.Breed = s.pet.BreedInfo()
	petStatus.Upkeep = s.pet.UpkeepStatuses()
	if petStatus.Upkeep == nil {
		petStatus.Upkeep = []core.UpkeepStatus{}
	}
	petStatus.FirstAidUses = s.pet.FirstAidUses
	petStatus.Injured = s.pet.Injured
	petStatus.HouseLook = map[string]core.HouseLook{}
	petStatus.SoundOff = s.parental.soundOff()
	if s.house != nil {
		d := s.house.snapshot()
		for slot, id := range d.Look {
			if l, ok := core.FindHouseLook(id); ok {
				petStatus.HouseLook[slot] = l
			}
		}
		petStatus.HomeStars = s.house.status(now).Stars
		petStatus.TipsSeen = d.TipsSeen
		petStatus.Shelf = []core.Souvenir{}
		for _, id := range d.Shelf {
			if sv, ok := core.FindSouvenir(id); ok {
				petStatus.Shelf = append(petStatus.Shelf, sv)
			}
		}
	}
	petStatus.Happening = s.pet.CurrentHappening()
	petStatus.Theme = core.WeekTheme()
	petStatus.Popups = append([]popup{}, s.popups...)
	if s.owner != nil {
		petStatus.PlaySchedule = s.owner.Summary()
	}
	if a := s.pet.LastAdventure; a != nil && status.Age-a.ReturnedAge < recentAdventureDays {
		petStatus.LastAdventure = a
	}
	petStatus.Changes = s.pet.PersonalityChanges()
	petStatus.Rewarded, petStatus.Trends = s.pet.LearningSummary()
	petStatus.Life = s.pet.LifeStatus()
	petStatus.Achievements = s.pet.EarnedAchievements()
	petStatus.AchievementTotal = core.AchievementCount()
	petStatus.LifeSeconds = gameDaysToSeconds(petStatus.Life.Remaining)
	petStatus.Attention = s.pet.AttentionStyle()
	petStatus.Relations = s.pet.Relations()
	petStatus.Tolerates = s.pet.Tolerates()
	petStatus.Likes, petStatus.Dislikes = s.pet.LikesAndDislikes()
	if petStatus.Likes == nil {
		petStatus.Likes = []string{}
	}
	if petStatus.Dislikes == nil {
		petStatus.Dislikes = []string{}
	}
	petStatus.PauseReason = s.pauseReasonLocked(now)
	petStatus.Paused = petStatus.PauseReason != ""

	writeJSON(w, http.StatusOK, APIResponse{Success: true, Data: petStatus})
}

func (s *Server) handleInteract(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, APIResponse{Success: false, Error: "Method not allowed"})
		return
	}

	var req struct {
		Action string `json:"action"`
		Food   string `json:"food"`   // Feed and treat: a food or treat from the pantry
		Toy    string `json:"toy"`    // Play: a toy from the pantry
		Region string `json:"region"` // Pet: where to pet
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, APIResponse{Success: false, Error: "Invalid request"})
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.pet == nil || !s.pet.IsAlive() {
		writeJSON(w, http.StatusBadRequest, APIResponse{Success: false, Error: s.unavailableMessageLocked()})
		return
	}
	if s.pet.IsHibernating() {
		writeJSON(w, http.StatusConflict, APIResponse{Success: false, Error: s.pet.Name + " is hibernating. Revive it first."})
		return
	}
	if s.pet.IsExploring() && req.Action != "clean" { // The home can be cleaned while the pet is out
		writeActivityError(w, s.pet.Name, core.ErrExploring)
		return
	}

	interactionType, ok := careActions[req.Action]
	if !ok {
		writeJSON(w, http.StatusBadRequest, APIResponse{Success: false, Error: "Unknown action"})
		return
	}

	s.markActiveLocked(time.Now())
	if result, chosen, err := s.interactWithChoice(req.Action, req.Food, req.Toy, req.Region); chosen {
		s.writeCareResult(w, result, err)
		return
	}
	s.writeCareResult(w, s.pet.Interact(interactionType, 1.0), nil)
}

func (s *Server) handleRename(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, APIResponse{Success: false, Error: "Method not allowed"})
		return
	}

	var req struct {
		Name      string `json:"name"`
		ParentPIN string `json:"parent_pin"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || !core.ValidPetName(req.Name) {
		writeJSON(w, http.StatusBadRequest, APIResponse{Success: false, Error: core.PetNameRule})
		return
	}
	if !s.parentAllows(w, req.ParentPIN) {
		return
	}
	req.Name = strings.TrimSpace(req.Name)

	s.mu.Lock()
	if s.pet == nil {
		s.mu.Unlock()
		writeJSON(w, http.StatusBadRequest, APIResponse{Success: false, Error: "No pet found"})
		return
	}
	s.pet.SetName(req.Name)
	s.mu.Unlock()

	writeJSON(w, http.StatusOK, APIResponse{Success: true})
}

func (s *Server) handleCreate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, APIResponse{Success: false, Error: "Method not allowed"})
		return
	}

	var req struct {
		Name      string `json:"name"`
		Randomize bool   `json:"randomize"`
		// Hibernate instead of dying from neglect (off by default)
		Hibernation bool   `json:"hibernation"`
		Species     string `json:"species"` // See /api/species; "" for the default (cat)
		Breed       string `json:"breed"`   // See /api/breeds; "" for the default, "random" for any unlocked one
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, APIResponse{Success: false, Error: "Invalid request"})
		return
	}
	species, ok := core.ParseSpecies(req.Species)
	if !ok {
		writeJSON(w, http.StatusBadRequest, APIResponse{Success: false, Error: "Unknown kind of pet"})
		return
	}
	if req.Name != "" && !core.ValidPetName(req.Name) {
		writeJSON(w, http.StatusBadRequest, APIResponse{Success: false, Error: core.PetNameRule})
		return
	}

	breed, err := s.chooseBreed(species, req.Breed)
	if err != nil {
		writeJSON(w, http.StatusConflict, APIResponse{Success: false, Error: capitalize(err.Error()) + "."})
		return
	}
	if err := s.createPet(strings.TrimSpace(req.Name), species, breed, req.Randomize, req.Hibernation); errors.Is(err, errTooManyPets) {
		writeJSON(w, http.StatusConflict, APIResponse{Success: false, Error: capitalize(err.Error()) + "."})
		return
	} else if err != nil {
		log.Printf("Create failed: %v", err)
		writeJSON(w, http.StatusInternalServerError, APIResponse{Success: false, Error: "Could not save the current pet, so no new pet was created"})
		return
	}

	s.mu.RLock()
	data := s.getPetData()
	s.mu.RUnlock()
	writeJSON(w, http.StatusOK, APIResponse{Success: true, Data: data})
}

func (s *Server) getPetData() map[string]interface{} {
	if s.pet == nil {
		return nil
	}

	status := s.pet.GetCurrentStatus()
	return map[string]interface{}{
		"id":               s.pet.ID,
		"name":             s.pet.Name,
		"age":              status.Age,
		"is_alive":         status.IsAlive,
		"hibernating":      status.Hibernating,
		"current_behavior": s.pet.CurrentBehavior.String(),
		"wellbeing":        status.Wellbeing,
		"vitals": map[string]float64{
			"health":      s.pet.Biology.Vitals.Health,
			"energy":      s.pet.Biology.Vitals.Energy,
			"happiness":   s.pet.Biology.Vitals.Happiness,
			"nutrition":   s.pet.Biology.Vitals.Nutrition,
			"hydration":   s.pet.Biology.Vitals.Hydration,
			"cleanliness": s.pet.Biology.Vitals.Cleanliness,
		},
		"emotions": map[string]interface{}{
			"mood":       s.pet.Emotions.GetMoodDescription(),
			"mood_score": s.pet.Emotions.GetMoodScore(),
		},
		"personality": s.pet.PersonalityDescription(),
	}
}

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data) // Fails only if the client has gone
}

// route is a page route: its pattern and handler
type route struct {
	pattern string
	handler http.HandlerFunc
}

// routes are the routes the page uses (the REST API, the cloud hub, static
// files and the page itself are added separately)
func (s *Server) routes() []route {
	return []route{
		{"/api/cloud", s.handleCloud},
		{"/api/cloud/connect", s.handleCloudConnect},
		{"/api/cloud/disconnect", s.handleCloudDisconnect},
		{"/api/cloud/sync", s.handleCloudSync},
		{"/api/cloud/profile", s.handleCloudProfile},
		{"/api/cloud/key", s.handleCloudKey},
		{"/api/pairing", s.handlePairing},
		{"/api/accounts", s.handleAccounts},
		{"/api/accounts/create", s.handleAccountCreate},
		{"/api/accounts/switch", s.handleAccountSwitch},
		{"/api/accounts/rename", s.handleAccountRename},
		{"/api/accounts/delete", s.handleAccountDelete},
		{"/pair", s.handlePair},
		{"/api/cloud/feed", s.handleCloudFeed},
		// Simple API routes
		{"/api/status", s.handleStatus},
		{"/api/interact", s.handleInteract},
		{"/api/rename", s.handleRename},
		{"/api/create", s.handleCreate},
		{"/api/save", s.handleSave},
		{"/api/saves", s.handleListSaves},
		{"/api/saves/pause", s.handlePausePet},
		{"/api/books", s.handleBooks},
		{"/api/projects", s.handleProjects},
		{"/api/books/candle", s.handleBookCandle},
		{"/api/books/delete", s.handleDeleteBook},
		{"/api/load", s.handleLoad},
		{"/api/pause", s.handlePause},
		{"/api/revive", s.handleRevive},
		{"/api/hibernation", s.handleHibernation},
		{"/api/delete", s.handleDelete},
		{"/api/shop", s.handleShop},
		{"/api/shop/buy", s.handleBuy},
		{"/api/train", s.handleTrick(false)},
		{"/api/perform", s.handleTrick(true)},
		{"/api/hideseek/start", s.handleHideSeekStart},
		{"/api/hideseek/guess", s.handleHideSeekGuess},
		{"/api/explore", s.handleExplore},
		{"/api/breed", s.handleBreed},
		{"/api/breed/partners", s.handleBreedPartners},
		{"/api/explore/recall", s.handleRecall},
		{"/api/explore/places", s.handlePlaces},
		{"/api/hideseek/quit", s.handleHideSeekQuit},
		{"/api/guide", s.handleGuide},
		{"/api/species", s.handleSpecies},
		{"/api/breeds", s.handleBreeds},
		{"/api/shows", s.handleShows},
		{"/api/shows/enter", s.handleEnterShow},
		{"/api/happening", s.handleHappening},
		{"/api/games", s.handleMiniGames},
		{"/api/games/finish", s.handleFinishMiniGame},
		{"/api/tips/seen", s.handleTipSeen},
		{"/api/collection", s.handleCollection},
		{"/api/home/shelf", s.handleShelf},
		{"/api/souvenirs/gift", s.handleGiveSouvenir},
		{"/api/privacy", s.handlePrivacy},
		{"/api/stats", s.handleStats},
		{"/api/home", s.handleHome},
		{"/api/home/buy", s.handleHomeBuy},
		{"/api/home/look", s.handleHomeLook},
		{"/api/settings", s.handleSettings},
		{"/api/family", s.handleFamily},
		{"/api/cosmetics/buy", s.handleBuyCosmetic},
		{"/api/wear", s.handleWear},
		{"/api/care/give", s.handleGiveCare},
		{"/api/holidays", s.handleHolidays},
		{"/api/family/transfer", s.handleFamilyTransfer},
		{"/api/family/pass_on", s.handleFamilyPassOn},
		{"/api/popup/dismiss", s.handlePopupDismiss},
		{"/api/potion", s.handlePotion},
		{"/api/events", s.handleEvents},
		{"/api/care/options", s.handleCareOptions},
		{"/api/supplies", s.handleSupplies},
		{"/api/supplies/buy", s.handleBuySupply},
		{"/api/enrich", s.handleEnrich},
		{"/api/introduce", s.handleIntroduce},
		{"/api/group", s.handleGroup},
		{"/api/introduce/partners", s.handleIntroducePartners},
		{"/api/ar", s.handleARState},
		{"/api/ar/session", s.handleARSession},
		{"/api/ar/plane", s.handleARPlane},
		{"/api/ar/camera", s.handleARCamera},
		{"/api/ar/place", s.handleARPlace},
		{"/api/ar/gesture", s.handleARGesture},
		{"/api/ar/exercise", s.handleARExercise},
		{"/api/playdate", s.handlePlaydate},
		{"/api/playdate/update", s.handlePlaydateUpdate},
		{"/api/playdate/play", s.handlePlaydatePlay},
	}
}
