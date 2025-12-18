package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/Michael-W-Ellison/gochi/internal/core"
	"github.com/Michael-W-Ellison/gochi/pkg/types"
)

// Server holds the server state
type Server struct {
	mu         sync.RWMutex
	pet        *core.DigitalPet
	webDir     string
	lastUpdate time.Time
}

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
}

func main() {
	// Find web directory
	webDir := findWebDir()
	if webDir == "" {
		log.Fatal("Could not find web directory")
	}
	log.Printf("Serving web content from: %s", webDir)

	// Create server
	server := &Server{
		webDir:     webDir,
		lastUpdate: time.Now(),
	}

	// Start background update loop
	go server.updateLoop()

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
	addr := ":8080"
	log.Printf("Starting Gochi server at http://localhost%s", addr)
	log.Printf("Open your browser to http://localhost:8080 to play!")

	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatal(err)
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

	for range ticker.C {
		s.mu.Lock()
		if s.pet != nil && s.pet.IsAlive() {
			deltaTime := time.Since(s.lastUpdate).Seconds()
			s.lastUpdate = time.Now()
			// Convert real seconds to game time (1 real second = 1/60 game minute)
			gameTime := deltaTime / 60.0
			s.pet.Update(gameTime)
		}
		s.mu.Unlock()
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
            <p class="subtitle">Create your virtual pet companion</p>
            <div class="pet-display">🐣</div>
            <div class="form-group">
                <label for="petNameInput">What will you name your pet?</label>
                <input type="text" id="petNameInput" placeholder="Enter a name..." maxlength="20" autofocus>
            </div>
            <button class="create-btn" id="createBtn" onclick="createPet()">Create My Pet!</button>
            <div class="error-msg hidden" id="errorMsg"></div>
        </div>

        <!-- Pet View -->
        <div id="petView" class="hidden">
            <h1>Gochi</h1>
            <div class="pet-name" id="petName">Loading...</div>
            <div class="pet-display" id="petSprite">🐱</div>
            <div class="mood" id="mood">Mood: Happy</div>

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

        function showCreateView() {
            document.getElementById('createView').classList.remove('hidden');
            document.getElementById('petView').classList.add('hidden');
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
        }

        function fetchStatus() {
            fetch('/api/status')
                .then(r => r.json())
                .then(data => {
                    if (data.success) {
                        hasPet = true;
                        showPetView();
                        updateUI(data.data);
                    } else {
                        hasPet = false;
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

        // Handle Enter key in name input
        document.getElementById('petNameInput').addEventListener('keypress', function(e) {
            if (e.key === 'Enter') {
                e.preventDefault();
                createPet();
            }
        });

        // Initial check for existing pet
        fetchStatus();
    </script>
</body>
</html>`
	w.Write([]byte(html))
}

func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	defer s.mu.RUnlock()

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
	}

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

	if req.Name == "" {
		req.Name = "Gochi"
	}

	s.mu.Lock()
	if req.Randomize {
		s.pet = core.NewDigitalPetRandom(req.Name, "web_user")
	} else {
		s.pet = core.NewDigitalPet(req.Name, "web_user")
	}
	s.lastUpdate = time.Now()
	s.mu.Unlock()

	log.Printf("Created new pet: %s", req.Name)
	writeJSON(w, http.StatusOK, APIResponse{Success: true, Data: s.getPetData()})
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

		if req.Name == "" {
			req.Name = "Gochi"
		}

		s.mu.Lock()
		if req.Randomize {
			s.pet = core.NewDigitalPetRandom(req.Name, "web_user")
		} else {
			s.pet = core.NewDigitalPet(req.Name, "web_user")
		}
		s.lastUpdate = time.Now()
		s.mu.Unlock()

		log.Printf("Created new pet: %s", req.Name)
		writeJSON(w, http.StatusOK, APIResponse{Success: true, Data: s.getPetData()})

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
