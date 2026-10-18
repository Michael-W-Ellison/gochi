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
	"github.com/Michael-W-Ellison/gochi/pkg/types"
)

// Server holds the server state
type Server struct {
	saveMu     sync.Mutex // Orders saves and pet switches; acquire before mu
	mu         sync.RWMutex
	pet        *core.DigitalPet // The active pet; only its time passes
	slot       string           // Save slot of the active pet
	webDir     string
	lastUpdate time.Time
	store      *petStore // nil disables saving

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
	Name       string  `json:"name"`
	Age        float64 `json:"age"`
	IsAlive    bool    `json:"is_alive"`
	Behavior   string  `json:"behavior"`
	Mood       string  `json:"mood"`
	Health     float64 `json:"health"`
	Energy     float64 `json:"energy"`
	Happiness  float64 `json:"happiness"`
	Hunger     float64 `json:"hunger"`
	Wellbeing  float64 `json:"wellbeing"`
	Personality string `json:"personality"`
	Slot        string `json:"slot"`
	Paused      bool   `json:"paused"`
	PauseReason string `json:"pause_reason,omitempty"`
}

func main() {
	dataDir := flag.String("data-dir", "", "directory for saves and backups (default ~/.gochi)")
	autosaveInterval := flag.Duration("autosave", time.Minute, "how often to save the pet automatically (0 disables)")
	awayTimeout := flag.Duration("away-pause", 2*time.Minute, "pause time when the player has been away this long (0 disables)")
	addr := flag.String("addr", ":8080", "address to listen on")
	flag.Parse()

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
		lastUpdate:   time.Now(),
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

	if s.pet == nil || !s.pet.IsAlive() || s.pauseReasonLocked(now) != "" {
		// No time passes; resuming continues from now
		s.lastUpdate = now
		return
	}

	deltaTime := now.Sub(s.lastUpdate).Seconds()
	s.lastUpdate = now
	// Convert real seconds to game time; the simulation treats one unit as
	// one game day, so one real minute is one game day
	s.pet.Update(deltaTime / 60.0)
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
            </div>
            <button class="create-btn" id="createBtn" onclick="createPet()">Create My Pet!</button>
            <div class="error-msg hidden" id="errorMsg"></div>
            <button class="back-btn hidden" id="backBtn" onclick="closeChooser()">Back to my pet</button>
        </div>

        <!-- Pet View -->
        <div id="petView" class="hidden">
            <h1>Gochi</h1>
            <div class="pet-name" id="petName">Loading...</div>
            <div class="pet-display" id="petSprite">🐱</div>
            <div class="mood" id="mood">Mood: Happy</div>
            <div class="pause-banner hidden" id="pauseBanner"></div>

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
            </div>

            <div class="actions">
                <button class="action-btn feed" onclick="interact('feed')">Feed</button>
                <button class="action-btn pet" onclick="interact('pet')">Pet</button>
                <button class="action-btn play" onclick="interact('play')">Play</button>
                <button class="action-btn groom" onclick="interact('groom')">Groom</button>
            </div>

            <div class="save-row">
                <button class="save-btn" id="saveBtn" onclick="savePet()">Save</button>
                <button class="save-btn" id="pauseBtn" onclick="togglePause()">Pause</button>
                <button class="save-btn" onclick="openChooser()">Switch Pet</button>
            </div>
            <div class="save-row">
                <span class="save-status" id="saveStatus">Progress is saved automatically</span>
            </div>

            <div class="status-text" id="statusText">Age: 0 days | Wellbeing: 100%</div>
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
                        meta.textContent = (pet.is_alive ? formatAge(pet.age) : 'Passed away') +
                            ' · saved ' + new Date(pet.saved_at).toLocaleString();
                        info.append(name, meta);

                        const btn = document.createElement('button');
                        btn.className = 'save-btn';
                        btn.textContent = isActive ? 'Playing' : 'Load';
                        btn.disabled = isActive;
                        btn.onclick = () => loadPet(pet.slot, btn);

                        item.append(info, btn);
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
                banner.textContent = '⏸ Time is paused. Your pet is safe until you press Resume.';
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
                body: JSON.stringify({name: name})
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
            document.getElementById('healthBar').style.width = (status.health * 100) + '%';
            document.getElementById('energyBar').style.width = (status.energy * 100) + '%';
            document.getElementById('happinessBar').style.width = (status.happiness * 100) + '%';
            document.getElementById('hungerBar').style.width = (status.hunger * 100) + '%';
            document.getElementById('mood').textContent = 'Mood: ' + status.mood;
            document.getElementById('statusText').textContent =
                'Age: ' + status.age.toFixed(1) + ' days | Wellbeing: ' + (status.wellbeing * 100).toFixed(0) + '%';

            let sprite = sprites.idle;
            if (!status.is_alive) {
                sprite = sprites.dead;
                document.getElementById('petView').classList.add('dead');
            } else {
                document.getElementById('petView').classList.remove('dead');
                if (status.behavior === 'Sleeping') sprite = sprites.sleeping;
                else if (status.behavior === 'Sick') sprite = sprites.sick;
                else if (status.happiness > 0.7) sprite = sprites.happy;
            }
            document.getElementById('petSprite').textContent = sprite;
            updatePauseUI(status);
        }

        function fetchStatus() {
            fetch('/api/status' + (playerPresent() ? '?active=1' : ''))
                .then(r => r.json())
                .then(data => {
                    if (data.success) {
                        hasPet = true;
                        if (!choosing) {
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
            if (action === 'feed') sprite.textContent = sprites.eating;
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
                    setTimeout(fetchStatus, 500);
                }
            })
            .catch(console.error);
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
	s.lastUpdate = now
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
func (s *Server) createPet(name string, randomize bool) error {
	if name == "" {
		name = "Gochi"
	}

	err := s.switchPet(func() (*core.DigitalPet, string, error) {
		pet := core.NewDigitalPet(name, "web_user")
		if randomize {
			pet = core.NewDigitalPetRandom(name, "web_user")
		}
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
		Name:       s.pet.Name,
		Age:        status.Age,
		IsAlive:    status.IsAlive,
		Behavior:   s.pet.CurrentBehavior.String(),
		Mood:       s.pet.Emotions.GetMoodDescription(),
		Health:     s.pet.Biology.Vitals.Health,
		Energy:     s.pet.Biology.Vitals.Energy,
		Happiness:  s.pet.Biology.Vitals.Happiness,
		Hunger:     1.0 - s.pet.Biology.Vitals.Nutrition,
		Wellbeing:  status.Wellbeing,
		Personality: s.pet.GetPersonalityDescription(),
		Slot:        s.slot,
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
		writeJSON(w, http.StatusBadRequest, APIResponse{Success: false, Error: "Pet is not available"})
		return
	}

	var interactionType types.InteractionType
	switch req.Action {
	case "feed":
		interactionType = types.InteractionFeeding
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
	s.pet.ProcessUserInteraction(interactionType, 1.0)
	writeJSON(w, http.StatusOK, APIResponse{Success: true, Data: map[string]string{"message": "Interaction successful"}})
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
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, APIResponse{Success: false, Error: "Invalid request"})
		return
	}

	if err := s.createPet(req.Name, req.Randomize); err != nil {
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
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, APIResponse{Success: false, Error: "Invalid request"})
			return
		}

		if err := s.createPet(req.Name, req.Randomize); err != nil {
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
		writeJSON(w, http.StatusBadRequest, APIResponse{Success: false, Error: "Pet is not available"})
		return
	}

	// Get action from path
	path := r.URL.Path
	var interactionType types.InteractionType
	var actionName string

	if strings.Contains(path, "/feed") {
		interactionType = types.InteractionFeeding
		actionName = "fed"
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
		"personality": s.pet.GetPersonalityDescription(),
	}
}

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}
