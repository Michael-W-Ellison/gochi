package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/Michael-W-Ellison/gochi/internal/core"
	"github.com/Michael-W-Ellison/gochi/internal/simulation"
	"github.com/Michael-W-Ellison/gochi/pkg/types"
)

// Server holds the server state
type Server struct {
	saveMu sync.Mutex // Orders saves and pet switches; acquire before mu
	mu     sync.RWMutex
	pet    *core.DigitalPet // The active pet; only its time passes
	slot   string           // Save slot of the active pet
	webDir string
	clock  *simulation.TimeManager // Game clock; its scale sets the game's speed
	store  *petStore               // nil disables saving

	// Time stops while the game is paused, so the pet cannot decline
	manualPause  bool          // Paused with the Pause button
	lastActivity time.Time     // Last time the player was present
	awayTimeout  time.Duration // Pause after this long without the player (0 disables)
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
	Name     string           `json:"name"`
	Age      float64          `json:"age"`
	IsAlive  bool             `json:"is_alive"`
	Behavior string           `json:"behavior"`
	Activity string           `json:"activity"` // What the pet is doing, e.g. "is playing"
	Request  *core.PetRequest `json:"request"`  // What the pet is asking for, or null

	Tricks           []core.TrickProgress   `json:"tricks"`
	Exploring        bool                   `json:"exploring"`
	ExploringSeconds int                    `json:"exploring_seconds"` // Real seconds until home
	HideAndSeek      bool                   `json:"hide_and_seek"`     // A game is in progress
	Souvenirs        []core.SouvenirCount   `json:"souvenirs"`
	Needs            []core.NeedLevel       `json:"needs"` // Needs not shown as stat bars
	Environment      core.EnvironmentStatus `json:"environment"`
	CareTips         map[string]string      `json:"care_tips"`      // What each care button does
	LastAdventure    *core.Adventure        `json:"last_adventure"` // Most recent trip, if it ended recently
	Mood             string                 `json:"mood"`
	Health           float64                `json:"health"`
	Energy           float64                `json:"energy"`
	Happiness        float64                `json:"happiness"`
	Hunger           float64                `json:"hunger"`
	Water            float64                `json:"water"`
	Wellbeing        float64                `json:"wellbeing"`
	Personality      string                 `json:"personality"`
	Likes            []string               `json:"likes"`
	Dislikes         []string               `json:"dislikes"`
	Slot             string                 `json:"slot"`
	Hibernating      bool                   `json:"hibernating"`
	HibernationOn    bool                   `json:"hibernation_enabled"` // Hibernate instead of dying
	CauseOfDeath     string                 `json:"cause_of_death,omitempty"`
	Paused           bool                   `json:"paused"`
	PauseReason      string                 `json:"pause_reason,omitempty"`

	FriendshipPoints int             `json:"friendship_points"`
	Items            []core.ShopItem `json:"items"`   // Owned items
	Effects          []core.ShopItem `json:"effects"` // Items whose effects currently apply
}

func main() {
	dataDir := flag.String("data-dir", "", "directory for saves and backups (default ~/.gochi)")
	autosaveInterval := flag.Duration("autosave", time.Minute, "how often to save the pet automatically (0 disables)")
	awayTimeout := flag.Duration("away-pause", 2*time.Minute, "pause time when the player has been away this long (0 disables)")
	addr := flag.String("addr", ":8080", "address to listen on")
	scaleName := flag.String("time-scale", "game", "game speed: game (1 game day per real minute), 24x, 4x or realtime")
	flag.Parse()

	timeScale, ok := timeScales[*scaleName]
	if !ok {
		log.Fatalf("Unknown -time-scale %q (use game, 24x, 4x or realtime)", *scaleName)
	}

	// Find web directory
	webDir := findWebDir()
	if webDir == "" {
		log.Fatal("Could not find web directory")
	}
	log.Printf("Serving web content from: %s", webDir)

	store, err := newPetStore(*dataDir)
	if err != nil {
		log.Fatalf("Could not set up save directory: %v", err)
	}

	// Create server
	server := &Server{
		webDir:       webDir,
		clock:        simulation.NewTimeManager(timeScale),
		store:        store,
		lastActivity: time.Now(),
		awayTimeout:  *awayTimeout,
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
	if *awayTimeout > 0 {
		log.Printf("Time pauses after %s without the player", *awayTimeout)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Start background update and autosave loops
	go server.updateLoop()
	if *autosaveInterval > 0 {
		go server.autosaveLoop(ctx, *autosaveInterval)
		log.Printf("Autosaving every %s", *autosaveInterval)
	}

	// Setup routes
	mux := http.NewServeMux()

	// API v1 routes (for the frontend JS)
	mux.HandleFunc("/api/v1/pets", server.handlePets)
	mux.HandleFunc("/api/v1/pets/", server.handlePetByID)
	mux.HandleFunc("/api/v1/interactions/", server.handleInteractV1)

	// Simple API routes
	mux.HandleFunc("/api/status", server.handleStatus)
	mux.HandleFunc("/api/interact", server.handleInteract)
	mux.HandleFunc("/api/rename", server.handleRename)
	mux.HandleFunc("/api/create", server.handleCreate)
	mux.HandleFunc("/api/save", server.handleSave)
	mux.HandleFunc("/api/saves", server.handleListSaves)
	mux.HandleFunc("/api/load", server.handleLoad)
	mux.HandleFunc("/api/pause", server.handlePause)
	mux.HandleFunc("/api/revive", server.handleRevive)
	mux.HandleFunc("/api/hibernation", server.handleHibernation)
	mux.HandleFunc("/api/delete", server.handleDelete)
	mux.HandleFunc("/api/shop", server.handleShop)
	mux.HandleFunc("/api/shop/buy", server.handleBuy)
	mux.HandleFunc("/api/train", server.handleTrick(false))
	mux.HandleFunc("/api/perform", server.handleTrick(true))
	mux.HandleFunc("/api/hideseek/start", server.handleHideSeekStart)
	mux.HandleFunc("/api/hideseek/guess", server.handleHideSeekGuess)
	mux.HandleFunc("/api/explore", server.handleExplore)
	mux.HandleFunc("/api/explore/recall", server.handleRecall)
	mux.HandleFunc("/api/hideseek/quit", server.handleHideSeekQuit)

	// Static files with no-cache headers for development
	staticHandler := http.StripPrefix("/static/", http.FileServer(http.Dir(filepath.Join(webDir, "static"))))
	mux.HandleFunc("/static/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
		w.Header().Set("Pragma", "no-cache")
		w.Header().Set("Expires", "0")
		staticHandler.ServeHTTP(w, r)
	})

	// Templates/pages
	mux.HandleFunc("/", server.handleIndex)

	// Start server
	httpServer := &http.Server{Addr: *addr, Handler: mux}
	serverErr := make(chan error, 1)
	go func() {
		serverErr <- httpServer.ListenAndServe()
	}()
	log.Printf("Starting Gochi server at http://localhost%s", *addr)
	log.Printf("Open your browser to http://localhost%s to play! Press Ctrl+C to save and exit.", *addr)

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

	if saved, err := server.saveNow(); err != nil {
		log.Printf("Failed to save pet on exit: %v", err)
		os.Exit(1)
	} else if saved {
		log.Printf("Saved pet to %s", store.SavePath(server.slot))
	}
}

func findWebDir() string {
	// Try various locations
	paths := []string{
		"web",
		"../web",
		"../../web",
		filepath.Join(os.Getenv("GOPATH"), "src/github.com/Michael-W-Ellison/gochi/web"),
	}

	// Also try relative to executable
	if exe, err := os.Executable(); err == nil {
		exeDir := filepath.Dir(exe)
		paths = append(paths,
			filepath.Join(exeDir, "web"),
			filepath.Join(exeDir, "..", "web"),
			filepath.Join(exeDir, "..", "..", "web"),
		)
	}

	// Try current working directory paths
	if cwd, err := os.Getwd(); err == nil {
		paths = append(paths,
			filepath.Join(cwd, "web"),
			filepath.Join(cwd, "..", "web"),
		)
	}

	for _, p := range paths {
		if info, err := os.Stat(p); err == nil && info.IsDir() {
			absPath, _ := filepath.Abs(p)
			return absPath
		}
	}

	return ""
}

func (s *Server) updateLoop() {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for now := range ticker.C {
		s.tick(now)
	}
}

// tick advances the active pet to now, unless the game is paused
func (s *Server) tick(now time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// No time passes while paused; resuming continues from now
	paused := s.pet == nil || !s.pet.IsAlive() || s.pauseReasonLocked(now) != ""
	s.clock.SetPaused(paused, now)
	gameSeconds := s.clock.UpdateAt(now)
	if paused {
		return
	}

	// The pet's simulation counts time in game days
	s.pet.Update(gameSeconds / secondsPerGameDay)
}

// careActions maps the page's care buttons to interactions
var careActions = map[string]types.InteractionType{
	"feed":  types.InteractionFeeding,
	"drink": types.InteractionDrinking,
	"pet":   types.InteractionPetting,
	"play":  types.InteractionPlaying,
	"groom": types.InteractionGrooming,
	"medic": types.InteractionMedicalCare,
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
        .actions .action-btn:last-child:nth-child(odd) { grid-column: span 2; }
        .speech { position: relative; margin: 0 auto 6px; max-width: 280px; padding: 8px 14px; border-radius: 16px; background: #fff3bf; color: #5c4400; font-size: 0.95em; text-align: center; }
        .speech::after { content: ''; position: absolute; left: 50%; bottom: -8px; margin-left: -8px; border: 8px solid transparent; border-top-color: #fff3bf; border-bottom: 0; }
        @keyframes nudge { 0%, 100% { box-shadow: 0 0 0 0 rgba(255, 193, 7, 0.9); } 50% { box-shadow: 0 0 0 6px rgba(255, 193, 7, 0); } }
        .requested { animation: nudge 1.2s ease-in-out infinite; outline: 3px solid #ffc107; }
        .activities { display: grid; grid-template-columns: repeat(3, 1fr); gap: 8px; margin-top: 10px; }
        .activity-btn { padding: 10px 6px; border: 2px solid #b197fc; border-radius: 12px; background: #f3f0ff; color: #5f3dc4; font-weight: 600; font-size: 0.9em; cursor: pointer; }
        .activity-btn:disabled { opacity: 0.4; cursor: not-allowed; }
        .activity-panel { margin-top: 10px; padding: 12px; border-radius: 12px; background: #f8f9fa; border: 2px solid #e9ecef; }
        .panel-head { display: flex; justify-content: space-between; align-items: center; margin-bottom: 8px; font-weight: 600; color: #333; }
        .panel-close { background: none; border: none; font-size: 1.2em; cursor: pointer; color: #888; }
        .panel-text { text-align: center; color: #444; margin-bottom: 10px; }
        .trick-row { display: flex; align-items: center; gap: 8px; margin-bottom: 8px; }
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
        .shop-msg { text-align: center; min-height: 1.3em; color: #c94a4a; font-size: 0.9em; margin-top: 8px; }
        .save-row { display: flex; align-items: center; justify-content: center; gap: 12px; margin-top: 16px; }
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
        .delete-btn { border-color: #e17070; color: #c94a4a; }
        .hibernate-banner { text-align: center; margin: 10px 0; padding: 12px; border-radius: 12px; background: #fff4e0; color: #8a5a00; font-size: 0.9em; }
        .hibernate-banner button { display: block; margin: 10px auto 0; }
        .action-btn:disabled { opacity: 0.4; cursor: not-allowed; transform: none; box-shadow: none; }
        .modal { position: fixed; inset: 0; background: rgba(0,0,0,0.45); display: flex; align-items: center; justify-content: center; padding: 16px; z-index: 10; }
        .modal-box { background: white; border-radius: 16px; padding: 24px; max-width: 360px; width: 100%; text-align: center; }
        .modal-box h3 { color: #333; margin-bottom: 10px; overflow-wrap: anywhere; }
        .modal-box p { color: #555; font-size: 0.95em; margin-bottom: 14px; overflow-wrap: anywhere; }
        .modal-box input { width: 100%; padding: 10px 12px; border: 2px solid #ddd; border-radius: 10px; font-size: 1em; margin-bottom: 14px; }
        .modal-actions { display: flex; gap: 10px; justify-content: center; }
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
    </style>
</head>
<body>
    <div class="container" id="app">
        <!-- Create Pet View -->
        <div id="createView" class="create-form">
            <h1>Welcome to Gochi!</h1>
            <div class="saved-pets hidden" id="savedPets">
                <h2>Your pets</h2>
                <ul class="saved-pet-list" id="savedPetList"></ul>
            </div>
            <p class="subtitle">Create a new virtual pet companion</p>
            <div class="pet-display">🐣</div>
            <div class="form-group">
                <label for="petNameInput">What will you name your pet?</label>
                <input type="text" id="petNameInput" placeholder="Enter a name..." maxlength="20" autofocus>
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
            <div class="pet-display" id="petSprite">🐱</div>
            <div class="activity" id="activity"></div>
            <div class="environment" id="environment"></div>
            <div class="mood" id="mood">Mood: Happy</div>
            <div class="pause-banner hidden" id="pauseBanner"></div>
            <div class="activities">
                <button class="activity-btn train" onclick="openTricks()">🎓 Train</button>
                <button class="activity-btn" onclick="startHideSeek()">🙈 Hide &amp; Seek</button>
                <button class="activity-btn explore" onclick="explore()">🧭 Explore</button>
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
                <button class="action-btn feed" onclick="interact('feed')">Feed</button>
                <button class="action-btn drink" onclick="interact('drink')">Drink</button>
                <button class="action-btn pet" onclick="interact('pet')">Pet</button>
                <button class="action-btn play" onclick="interact('play')">Play</button>
                <button class="action-btn groom" onclick="interact('groom')">Groom</button>
                <button class="action-btn medic" onclick="interact('medic')">🩹 Heal</button>
            </div>

            <div class="save-row">
                <button class="save-btn" id="saveBtn" onclick="savePet()">Save</button>
                <button class="save-btn" id="pauseBtn" onclick="togglePause()">Pause</button>
                <button class="save-btn" onclick="openChooser()">Switch Pet</button>
                <button class="save-btn" onclick="openShop()">Shop</button>
            </div>
            <div class="save-row">
                <span class="save-status" id="saveStatus">Progress is saved automatically</span>
            </div>
            <label class="setting">
                <input type="checkbox" id="hibernationToggle" onchange="setHibernation(this.checked)">
                <span id="hibernationLabel">Gentle mode: hibernate instead of dying if neglected</span>
            </label>

            <div class="about" id="about">
                <div id="aboutPersonality"></div>
                <div id="aboutLikes"></div>
                <div id="aboutDislikes"></div>
                <div id="aboutTreasures"></div>
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
                <div class="modal-actions">
                    <button class="save-btn" onclick="closeDeleteModal()">Cancel</button>
                    <button class="danger-btn" id="deleteFinalBtn" onclick="confirmDelete()" disabled>Delete forever</button>
                </div>
            </div>
        </div>
    </div>

    <script>
        const sprites = {
            idle: '🐱',
            happy: '😺',
            eating: '😋',
            playing: '😸',
            sleeping: '😴',
            sick: '🤒',
            exploring: '🐈',
            social: '😻',
            hibernating: '💤',
            dead: '😿'
        };

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

        document.addEventListener('visibilitychange', () => {
            if (!document.hidden) fetchStatus();
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
                        name.textContent = pet.name;
                        const meta = document.createElement('div');
                        meta.className = 'saved-pet-meta';
                        const state = !pet.is_alive ? 'Passed away' :
                            pet.hibernating ? 'Hibernating · ' + formatAge(pet.age) : formatAge(pet.age);
                        meta.textContent = state + ' · saved ' + new Date(pet.saved_at).toLocaleString();
                        info.append(name, meta);

                        const btn = document.createElement('button');
                        btn.className = 'save-btn';
                        btn.textContent = isActive ? 'Playing' : 'Load';
                        btn.disabled = isActive;
                        btn.onclick = () => loadPet(pet.slot, btn);

                        const del = document.createElement('button');
                        del.className = 'save-btn delete-btn';
                        del.textContent = 'Delete';
                        del.onclick = () => openDeleteModal(pet);

                        const actions = document.createElement('div');
                        actions.className = 'saved-pet-actions';
                        actions.append(btn, del);

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

            fetch('/api/delete', {
                method: 'POST',
                headers: {'Content-Type': 'application/json'},
                body: JSON.stringify({
                    slot: deleteTarget.slot,
                    confirm_name: document.getElementById('deleteConfirmInput').value
                })
            })
            .then(r => r.json())
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
            fetch('/api/revive', {method: 'POST'})
                .then(r => r.json())
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
        const shopCategories = [
            ['food_bowl', 'Food containers'], ['water_bowl', 'Water containers'],
            ['toy', 'Toys'], ['bed', 'Beds'], ['brush', 'Grooming'], ['decoration', 'Decorations']
        ];

        function openShop() {
            shopping = true;
            document.getElementById('app').classList.add('hidden');
            document.getElementById('shopView').classList.remove('hidden');
            document.getElementById('shopMsg').textContent = '';
            loadShop();
        }

        function closeShop() {
            shopping = false;
            document.getElementById('shopView').classList.add('hidden');
            document.getElementById('app').classList.remove('hidden');
            fetchStatus();
        }

        function loadShop() {
            fetch('/api/shop')
                .then(r => r.json())
                .then(data => { if (data.success) renderShop(data.data); })
                .catch(err => console.error('Shop error:', err));
        }

        function renderShop(shop) {
            document.getElementById('shopPoints').textContent = formatPoints(shop.friendship_points);
            const list = document.getElementById('shopList');
            list.replaceChildren();

            shopCategories.forEach(([category, title]) => {
                const items = shop.items.filter(item => item.category === category);
                if (items.length === 0) return;

                const heading = document.createElement('div');
                heading.className = 'shop-category';
                heading.textContent = title;
                list.append(heading);

                items.forEach(item => {
                    const row = document.createElement('div');
                    row.className = 'shop-item' + (item.owned ? ' owned' : '');

                    const icon = document.createElement('div');
                    icon.className = 'shop-icon';
                    icon.textContent = item.icon;

                    const info = document.createElement('div');
                    info.className = 'shop-info';
                    const name = document.createElement('div');
                    name.className = 'shop-name';
                    name.textContent = item.name;
                    const desc = document.createElement('div');
                    desc.className = 'shop-desc';
                    desc.textContent = item.description;
                    info.append(name, desc);

                    const btn = document.createElement('button');
                    btn.className = 'save-btn';
                    if (item.owned) {
                        btn.textContent = 'Owned';
                        btn.disabled = true;
                    } else {
                        btn.textContent = '❤️ ' + item.price;
                        btn.disabled = shop.friendship_points < item.price;
                        btn.onclick = () => buyItem(item.id);
                    }

                    row.append(icon, info, btn);
                    list.append(row);
                });
            });
        }

        function buyItem(id) {
            const msg = document.getElementById('shopMsg');
            msg.textContent = '';
            fetch('/api/shop/buy', {
                method: 'POST',
                headers: {'Content-Type': 'application/json'},
                body: JSON.stringify({item: id})
            })
            .then(r => r.json())
            .then(data => {
                if (data.success) {
                    renderShop(data.data);
                } else {
                    msg.textContent = data.error || 'Could not buy this item';
                    loadShop();
                }
            })
            .catch(() => { msg.textContent = 'Connection error. Please try again.'; });
        }

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
            toastTimer = setTimeout(() => { toast.replaceChildren(); }, 5000);
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
                body: JSON.stringify({name: name, hibernation: document.getElementById('hibernationInput').checked})
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
            document.getElementById('petName').textContent = status.name;
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

            document.getElementById('aboutPersonality').textContent = status.personality;
            const setList = (id, label, items, none) => {
                const el = document.getElementById(id);
                el.replaceChildren();
                const strong = document.createElement('span');
                strong.className = 'about-label';
                strong.textContent = label + ': ';
                el.append(strong, items.length ? items.join(', ') : none);
            };
            setList('aboutLikes', 'Likes', status.likes || [], 'nothing yet');
            setList('aboutDislikes', 'Dislikes', status.dislikes || [], 'nothing yet');
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
            document.getElementById('friendship').textContent = formatPoints(status.friendship_points);
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
                'Age: ' + status.age.toFixed(1) + ' days | Wellbeing: ' + (status.wellbeing * 100).toFixed(0) + '%';

            let sprite = sprites.idle;
            if (!status.is_alive) {
                sprite = sprites.dead;
                document.getElementById('petView').classList.add('dead');
            } else {
                document.getElementById('petView').classList.remove('dead');
                if (status.hibernating) sprite = sprites.hibernating;
                else if (status.behavior === 'Sleeping') sprite = sprites.sleeping;
                else if (status.behavior === 'Sick') sprite = sprites.sick;
                else if (status.behavior === 'Playing') sprite = sprites.playing;
                else if (status.behavior === 'Exploring') sprite = sprites.exploring;
                else if (status.behavior === 'Social Interaction') sprite = sprites.social;
                else if (status.behavior === 'Eating') sprite = sprites.eating;
                else if (status.happiness > 0.7) sprite = sprites.happy;
            }
            document.getElementById('petSprite').textContent = sprite;
            updatePauseUI(status);

            const hibernating = status.is_alive && status.hibernating;
            document.getElementById('hibernateText').textContent = hibernating ?
                '💤 ' + status.name + ' got too weak and is hibernating to stay safe. Its stats are paused. Revive it to keep playing.' : '';
            document.getElementById('hibernateBanner').classList.toggle('hidden', !hibernating);
            document.querySelectorAll('.actions .action-btn').forEach(b => { b.disabled = hibernating || status.exploring; });
            updateActivities(status);
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

        function startUpdates() {
            if (updateInterval) clearInterval(updateInterval);
            updateInterval = setInterval(fetchStatus, 2000);
        }

        function interact(action) {
            const btn = event.target;
            btn.style.transform = 'scale(0.95)';
            setTimeout(() => btn.style.transform = '', 100);

            // Show reaction
            const sprite = document.getElementById('petSprite');
            if (action === 'feed' || action === 'drink') sprite.textContent = sprites.eating;
            else if (action === 'play') sprite.textContent = sprites.playing;
            else if (action === 'pet') sprite.textContent = sprites.happy;

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
            const face = {delighted: '😻', reluctant: '😾'}[result.reaction];
            if (face) document.getElementById('petSprite').textContent = face;
            const changes = result.changes || [];
            if (changes.length > 0) lines.push([formatChanges(changes), 'changes']);
            const boost = result.boosted_by;
            if (boost) lines.push([boost.icon + ' ' + boost.name + ': ' + boost.description, 'boost']);
            if (result.bonus_points > 0) lines.push(['+' + result.bonus_points + ' game bonus!', '']);
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

        function postJSON(url, body) {
            return fetch(url, {
                method: 'POST',
                headers: {'Content-Type': 'application/json'},
                body: JSON.stringify(body || {})
            }).then(r => r.json());
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

        function openTricks() {
            panelMode = 'tricks';
            renderTricks();
        }

        function renderTricks() {
            if (!lastStatus) return;
            const p = panel();
            p.replaceChildren(panelHeader('Tricks'));
            lastStatus.tricks.forEach(trick => {
                const row = document.createElement('div');
                row.className = 'trick-row';
                const name = document.createElement('span');
                name.className = 'trick-name';
                name.textContent = trick.icon + ' ' + trick.name;
                const bar = document.createElement('div');
                bar.className = 'trick-bar';
                const fill = document.createElement('div');
                fill.className = 'trick-fill';
                fill.style.width = Math.round(trick.skill * 100) + '%';
                bar.append(fill);
                const btn = document.createElement('button');
                btn.className = 'save-btn';
                btn.textContent = trick.learned ? 'Do it!' : 'Train';
                btn.onclick = () => doTrick(trick.id, trick.learned);
                row.append(name, bar, btn);
                p.append(row);
            });
            p.classList.remove('hidden');
        }

        function doTrick(id, perform) {
            postJSON(perform ? '/api/perform' : '/api/train', {trick: id})
                .then(data => {
                    if (data.success) {
                        showResult(data.data, data.data.text);
                        fetchStatus();
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

        function explore() {
            postJSON('/api/explore')
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
            if (busy && panelMode) closePanel();
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

            fetch('/api/save', {method: 'POST'})
                .then(r => r.json())
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
        fetchStatus();
        startUpdates();
    </script>
</body>
</html>`
	w.Write([]byte(html))
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
			if _, err := s.saveNow(); err != nil {
				log.Printf("Autosave failed: %v", err)
			}
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
	slot := s.slot
	petData, err := s.pet.Save()
	s.mu.RUnlock()
	if err != nil {
		return false, err
	}

	if err := s.store.Save(slot, petData); err != nil {
		return false, err
	}
	return true, nil
}

// saveActiveLocked writes the active pet, if any, to its slot.
// Caller must hold s.saveMu and s.mu.
func (s *Server) saveActiveLocked() error {
	if s.store == nil || s.pet == nil {
		return nil
	}
	petData, err := s.pet.Save()
	if err != nil {
		return err
	}
	return s.store.Save(s.slot, petData)
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
	return nil
}

// createPet saves the active pet and replaces it with a new one
func (s *Server) createPet(name string, randomize, hibernation bool) error {
	if name == "" {
		name = "Gochi"
	}

	err := s.switchPet(func() (*core.DigitalPet, string, error) {
		pet := core.NewDigitalPet(name, "web_user")
		if randomize {
			pet = core.NewDigitalPetRandom(name, "web_user")
		}
		pet.HibernationEnabled = hibernation
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

// shopItemView is a shop item with whether the active pet owns it
type shopItemView struct {
	core.ShopItem
	Owned bool `json:"owned"`
}

// shopDataLocked returns the shop for the active pet. Caller must hold s.mu.
func (s *Server) shopDataLocked() map[string]interface{} {
	catalog := core.ShopCatalog()
	items := make([]shopItemView, len(catalog))
	for i, item := range catalog {
		items[i] = shopItemView{ShopItem: item, Owned: s.pet.Owns(item.ID)}
	}
	return map[string]interface{}{
		"friendship_points": s.pet.FriendshipPoints,
		"items":             items,
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

	s.mu.Lock()
	if s.pet == nil {
		s.mu.Unlock()
		writeJSON(w, http.StatusNotFound, APIResponse{Success: false, Error: "No pet found"})
		return
	}
	s.markActiveLocked(time.Now())
	name := s.pet.Name
	err := s.pet.BuyItem(req.Item)
	data := s.shopDataLocked()
	s.mu.Unlock()

	switch {
	case errors.Is(err, core.ErrUnknownItem):
		writeJSON(w, http.StatusBadRequest, APIResponse{Success: false, Error: "Unknown item"})
	case errors.Is(err, core.ErrAlreadyOwned):
		writeJSON(w, http.StatusConflict, APIResponse{Success: false, Error: name + " already has this"})
	case errors.Is(err, core.ErrNotEnoughPoints):
		writeJSON(w, http.StatusBadRequest, APIResponse{Success: false, Error: "Not enough Friendship Points"})
	case err != nil:
		writeJSON(w, http.StatusBadRequest, APIResponse{Success: false, Error: name + " can't use items right now"})
	default:
		if _, err := s.saveNow(); err != nil {
			log.Printf("Failed to save after purchase: %v", err)
		}
		writeJSON(w, http.StatusOK, APIResponse{Success: true, Data: data})
	}
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
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || !ValidSlot(req.Slot) {
		writeJSON(w, http.StatusBadRequest, APIResponse{Success: false, Error: "Invalid save slot"})
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
	var name string
	if active {
		name = s.pet.Name
	} else {
		summary, err := s.store.Summary(req.Slot)
		switch {
		case errors.Is(err, os.ErrNotExist):
			writeJSON(w, http.StatusNotFound, APIResponse{Success: false, Error: "No saved pet in that slot"})
			return
		case err != nil:
			name = req.Slot // Unreadable save: confirm with the slot name
		default:
			name = summary.Name
		}
	}

	if name == "" {
		name = req.Slot
	}
	if strings.TrimSpace(req.ConfirmName) != name {
		writeJSON(w, http.StatusBadRequest, APIResponse{Success: false, Error: "Type the pet's name exactly to confirm"})
		return
	}

	if err := s.store.Delete(req.Slot); err != nil && !errors.Is(err, os.ErrNotExist) {
		log.Printf("Delete failed: %v", err)
		writeJSON(w, http.StatusInternalServerError, APIResponse{Success: false, Error: "Could not delete pet"})
		return
	}
	if active {
		s.pet, s.slot = nil, ""
	}

	log.Printf("Deleted %s (%s)", name, req.Slot)
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
		Name:        s.pet.Name,
		Age:         status.Age,
		IsAlive:     status.IsAlive,
		Behavior:    s.pet.CurrentBehavior.String(),
		Activity:    core.ActivityDescription(s.pet.CurrentBehavior),
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
	petStatus.HibernationOn = s.pet.HibernationEnabled
	petStatus.CauseOfDeath = s.pet.Biology.CauseOfDeath
	petStatus.Tricks = s.pet.Tricks()
	petStatus.Exploring = s.pet.IsExploring()
	petStatus.ExploringSeconds = gameDaysToSeconds(s.pet.ExploringRemaining())
	petStatus.HideAndSeek = s.pet.PlayingHideAndSeek()
	petStatus.Souvenirs = s.pet.SouvenirCollection()
	if a := s.pet.LastAdventure; a != nil && status.Age-a.ReturnedAge < recentAdventureDays {
		petStatus.LastAdventure = a
	}
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
	if s.pet.IsExploring() {
		writeActivityError(w, s.pet.Name, core.ErrExploring)
		return
	}

	var interactionType types.InteractionType
	switch req.Action {
	case "feed":
		interactionType = types.InteractionFeeding
	case "drink":
		interactionType = types.InteractionDrinking
	case "pet":
		interactionType = types.InteractionPetting
	case "play":
		interactionType = types.InteractionPlaying
	case "groom":
		interactionType = types.InteractionGrooming
	case "treat":
		interactionType = types.InteractionRewards
	case "medic":
		interactionType = types.InteractionMedicalCare
	default:
		writeJSON(w, http.StatusBadRequest, APIResponse{Success: false, Error: "Unknown action"})
		return
	}

	s.markActiveLocked(time.Now())
	result := s.pet.Interact(interactionType, 1.0)
	if !result.Applied {
		message := result.Refusal
		if message == "" {
			message = s.pet.Name + " can't do that right now."
		}
		writeJSON(w, http.StatusConflict, APIResponse{Success: false, Error: message})
		return
	}
	writeJSON(w, http.StatusOK, APIResponse{Success: true, Data: s.interactionData(result)})
}

func (s *Server) handleRename(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, APIResponse{Success: false, Error: "Method not allowed"})
		return
	}

	var req struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Name == "" {
		writeJSON(w, http.StatusBadRequest, APIResponse{Success: false, Error: "Invalid name"})
		return
	}

	s.mu.Lock()
	if s.pet == nil {
		s.mu.Unlock()
		writeJSON(w, http.StatusBadRequest, APIResponse{Success: false, Error: "No pet found"})
		return
	}
	s.pet.Name = req.Name
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
		Hibernation bool `json:"hibernation"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, APIResponse{Success: false, Error: "Invalid request"})
		return
	}

	if err := s.createPet(req.Name, req.Randomize, req.Hibernation); err != nil {
		log.Printf("Create failed: %v", err)
		writeJSON(w, http.StatusInternalServerError, APIResponse{Success: false, Error: "Could not save the current pet, so no new pet was created"})
		return
	}

	s.mu.RLock()
	data := s.getPetData()
	s.mu.RUnlock()
	writeJSON(w, http.StatusOK, APIResponse{Success: true, Data: data})
}

// handlePets handles /api/v1/pets - list and create
func (s *Server) handlePets(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		// List pets
		s.mu.RLock()
		defer s.mu.RUnlock()

		if s.pet == nil {
			writeJSON(w, http.StatusOK, APIResponse{Success: true, Data: []interface{}{}})
			return
		}
		writeJSON(w, http.StatusOK, APIResponse{Success: true, Data: []interface{}{s.getPetData()}})

	case http.MethodPost:
		// Create pet
		var req struct {
			Name      string `json:"name"`
			Randomize bool   `json:"randomize"`
			// Hibernate instead of dying from neglect (off by default)
			Hibernation bool `json:"hibernation"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, APIResponse{Success: false, Error: "Invalid request"})
			return
		}

		if err := s.createPet(req.Name, req.Randomize, req.Hibernation); err != nil {
			log.Printf("Create failed: %v", err)
			writeJSON(w, http.StatusInternalServerError, APIResponse{Success: false, Error: "Could not save the current pet, so no new pet was created"})
			return
		}

		s.mu.RLock()
		data := s.getPetData()
		s.mu.RUnlock()
		writeJSON(w, http.StatusOK, APIResponse{Success: true, Data: data})

	default:
		writeJSON(w, http.StatusMethodNotAllowed, APIResponse{Success: false, Error: "Method not allowed"})
	}
}

// handlePetByID handles /api/v1/pets/{id}/* routes
func (s *Server) handlePetByID(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.pet == nil {
		writeJSON(w, http.StatusNotFound, APIResponse{Success: false, Error: "No pet found"})
		return
	}

	// Parse the path to find what's being requested
	path := r.URL.Path

	if strings.HasSuffix(path, "/status") {
		// Get pet status
		writeJSON(w, http.StatusOK, APIResponse{Success: true, Data: s.getPetData()})
	} else if strings.HasSuffix(path, "/vitals") {
		// Get vitals
		vitals := map[string]float64{
			"health":      s.pet.Biology.Vitals.Health,
			"energy":      s.pet.Biology.Vitals.Energy,
			"happiness":   s.pet.Biology.Vitals.Happiness,
			"nutrition":   s.pet.Biology.Vitals.Nutrition,
			"hydration":   s.pet.Biology.Vitals.Hydration,
			"cleanliness": s.pet.Biology.Vitals.Cleanliness,
		}
		writeJSON(w, http.StatusOK, APIResponse{Success: true, Data: vitals})
	} else if strings.HasSuffix(path, "/emotions") {
		// Get emotions
		emotions := map[string]interface{}{
			"mood":       s.pet.Emotions.GetMoodDescription(),
			"mood_score": s.pet.Emotions.GetMoodScore(),
		}
		writeJSON(w, http.StatusOK, APIResponse{Success: true, Data: emotions})
	} else {
		// Get full pet data
		writeJSON(w, http.StatusOK, APIResponse{Success: true, Data: s.getPetData()})
	}
}

// handleInteractV1 handles /api/v1/interactions/* routes
func (s *Server) handleInteractV1(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, APIResponse{Success: false, Error: "Method not allowed"})
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

	// Get action from path
	path := r.URL.Path
	var interactionType types.InteractionType
	var actionName string

	if strings.Contains(path, "/feed") {
		interactionType = types.InteractionFeeding
		actionName = "fed"
	} else if strings.Contains(path, "/drink") {
		interactionType = types.InteractionDrinking
		actionName = "given a drink by"
	} else if strings.Contains(path, "/pet") {
		interactionType = types.InteractionPetting
		actionName = "petted"
	} else if strings.Contains(path, "/play") {
		interactionType = types.InteractionPlaying
		actionName = "played with"
	} else if strings.Contains(path, "/groom") {
		interactionType = types.InteractionGrooming
		actionName = "groomed"
	} else if strings.Contains(path, "/train") {
		interactionType = types.InteractionTraining
		actionName = "trained"
	} else if strings.Contains(path, "/medical") {
		interactionType = types.InteractionMedicalCare
		actionName = "gave medical care to"
	} else {
		writeJSON(w, http.StatusBadRequest, APIResponse{Success: false, Error: "Unknown interaction"})
		return
	}

	s.markActiveLocked(time.Now())
	s.pet.ProcessUserInteraction(interactionType, 1.0)

	response := map[string]interface{}{
		"message":      "Interaction successful",
		"pet_reaction": s.pet.Name + " enjoyed being " + actionName + "!",
	}
	writeJSON(w, http.StatusOK, APIResponse{Success: true, Data: response})
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
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}
