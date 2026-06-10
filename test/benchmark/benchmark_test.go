package benchmark

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/Michael-W-Ellison/gochi/internal/cloud"
	"github.com/Michael-W-Ellison/gochi/internal/core"
	"github.com/Michael-W-Ellison/gochi/internal/data"
	"github.com/Michael-W-Ellison/gochi/pkg/security"
	"github.com/Michael-W-Ellison/gochi/pkg/types"
)

// BenchmarkPetCreation benchmarks pet creation performance
func BenchmarkPetCreation(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = core.NewDigitalPet("BenchPet", "user_123")
	}
}

// BenchmarkPetUpdate benchmarks pet update performance
func BenchmarkPetUpdate(b *testing.B) {
	pet := core.NewDigitalPet("BenchPet", "user_123")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		pet.Update(0.016) // 60 FPS delta
	}
}

// BenchmarkDataSave benchmarks pet data saving
func BenchmarkDataSave(b *testing.B) {
	tmpDir := b.TempDir()
	dataManager, err := data.NewDataManager(&data.Config{
		BasePath:   filepath.Join(tmpDir, "data"),
		BackupPath: filepath.Join(tmpDir, "backups"),
	})
	if err != nil {
		b.Fatalf("Failed to create data manager: %v", err)
	}
	defer dataManager.Close()

	petData := map[string]interface{}{
		"name":   "BenchPet",
		"health": 100,
		"hunger": 50,
		"happiness": 75,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		petID := types.PetID("bench-pet")
		err := dataManager.SavePet(petID, petData)
		if err != nil {
			b.Fatalf("Save failed: %v", err)
		}
	}
}

// BenchmarkDataLoad benchmarks pet data loading
func BenchmarkDataLoad(b *testing.B) {
	tmpDir := b.TempDir()
	dataManager, err := data.NewDataManager(&data.Config{
		BasePath:   filepath.Join(tmpDir, "data"),
		BackupPath: filepath.Join(tmpDir, "backups"),
	})
	if err != nil {
		b.Fatalf("Failed to create data manager: %v", err)
	}
	defer dataManager.Close()

	// Setup: save a pet first
	petID := types.PetID("bench-pet")
	petData := map[string]interface{}{
		"name":   "BenchPet",
		"health": 100,
	}
	dataManager.SavePet(petID, petData)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var loaded map[string]interface{}
		err := dataManager.LoadPet(petID, &loaded)
		if err != nil {
			b.Fatalf("Load failed: %v", err)
		}
	}
}

// BenchmarkPasswordHashing benchmarks password hashing
func BenchmarkPasswordHashing(b *testing.B) {
	tmpDir := b.TempDir()
	dbPath := filepath.Join(tmpDir, "auth.db")
	provider, err := cloud.NewSQLiteAuthProvider(dbPath)
	if err != nil {
		b.Fatalf("Failed to create provider: %v", err)
	}
	defer provider.Close()

	creds := &cloud.Credentials{
		Username: "benchuser",
		Password: "BenchPass123!",
		Email:    "bench@example.com",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		provider.Register(creds)
	}
}

// BenchmarkAuthentication benchmarks authentication performance
func BenchmarkAuthentication(b *testing.B) {
	tmpDir := b.TempDir()
	dbPath := filepath.Join(tmpDir, "auth.db")
	provider, err := cloud.NewSQLiteAuthProvider(dbPath)
	if err != nil {
		b.Fatalf("Failed to create provider: %v", err)
	}
	defer provider.Close()

	// Setup: register user first
	creds := &cloud.Credentials{
		Username: "benchuser",
		Password: "BenchPass123!",
		Email:    "bench@example.com",
	}
	provider.Register(creds)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := provider.Authenticate(creds.Username, creds.Password)
		if err != nil {
			b.Fatalf("Authentication failed: %v", err)
		}
	}
}

// BenchmarkCloudUpload benchmarks cloud upload performance
func BenchmarkCloudUpload(b *testing.B) {
	tmpDir := b.TempDir()
	dbPath := filepath.Join(tmpDir, "auth.db")
	provider, err := cloud.NewSQLiteAuthProvider(dbPath)
	if err != nil {
		b.Fatalf("Failed to create provider: %v", err)
	}
	defer provider.Close()

	authManager := cloud.NewAuthManager(provider)
	creds := &cloud.Credentials{
		Username: "benchuser",
		Password: "BenchPass123!",
		Email:    "bench@example.com",
	}
	authManager.Register(creds)

	cloudProvider := cloud.NewMockCloudStorageProvider(authManager)
	ctx := context.Background()
	petID := types.PetID("bench-pet")
	data := []byte(`{"name":"BenchPet","health":100}`)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := cloudProvider.Upload(ctx, petID, data)
		if err != nil {
			b.Fatalf("Upload failed: %v", err)
		}
	}
}

// BenchmarkRateLimiter benchmarks rate limiter performance
func BenchmarkRateLimiter(b *testing.B) {
	rl := security.NewRateLimiter(100, 1*time.Minute)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rl.CheckLimit("bench-user")
	}
}

// BenchmarkInputValidation benchmarks input validation
func BenchmarkInputValidation(b *testing.B) {
	b.Run("Username", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			security.ValidateUsername("testuser")
		}
	})

	b.Run("Email", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			security.ValidateEmail("test@example.com")
		}
	})

	b.Run("Password", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			security.ValidatePassword("SecurePass123!")
		}
	})

	b.Run("PetID", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			security.ValidatePetID("test-pet-123")
		}
	})
}

// BenchmarkGameLoopUpdate benchmarks game loop update performance
func BenchmarkGameLoopUpdate(b *testing.B) {
	tmpDir := b.TempDir()
	config := core.DefaultGameLoopConfig()
	config.DataManagerConfig = &data.Config{
		BasePath:   filepath.Join(tmpDir, "data"),
		BackupPath: filepath.Join(tmpDir, "backups"),
	}

	gameLoop, err := core.NewGameLoop(config)
	if err != nil {
		b.Fatalf("Failed to create game loop: %v", err)
	}

	// Add some pets
	for i := 0; i < 10; i++ {
		pet := core.NewDigitalPet("BenchPet", "user_123")
		gameLoop.AddPet(pet)
	}

	gameLoop.Start()
	defer gameLoop.Shutdown()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		gameLoop.Update()
	}
}

// BenchmarkConcurrentPetAccess benchmarks concurrent pet access
func BenchmarkConcurrentPetAccess(b *testing.B) {
	tmpDir := b.TempDir()
	config := core.DefaultGameLoopConfig()
	config.DataManagerConfig = &data.Config{
		BasePath:   filepath.Join(tmpDir, "data"),
		BackupPath: filepath.Join(tmpDir, "backups"),
	}

	gameLoop, err := core.NewGameLoop(config)
	if err != nil {
		b.Fatalf("Failed to create game loop: %v", err)
	}

	pet := core.NewDigitalPet("BenchPet", "user_123")
	gameLoop.AddPet(pet)
	gameLoop.Start()
	defer gameLoop.Shutdown()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, err := gameLoop.GetPet(pet.ID)
			if err != nil {
				b.Fatalf("GetPet failed: %v", err)
			}
		}
	})
}

// BenchmarkCacheHit benchmarks cache hit performance
func BenchmarkCacheHit(b *testing.B) {
	tmpDir := b.TempDir()
	dataManager, err := data.NewDataManager(&data.Config{
		BasePath:   filepath.Join(tmpDir, "data"),
		BackupPath: filepath.Join(tmpDir, "backups"),
		CacheSizeMB: 100,
		CacheTTLMinutes: 30,
	})
	if err != nil {
		b.Fatalf("Failed to create data manager: %v", err)
	}
	defer dataManager.Close()

	// Setup: save and load to populate cache
	petID := types.PetID("bench-pet")
	petData := map[string]interface{}{"name": "BenchPet"}
	dataManager.SavePet(petID, petData)
	var loaded map[string]interface{}
	dataManager.LoadPet(petID, &loaded) // Populate cache

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var loaded map[string]interface{}
		err := dataManager.LoadPet(petID, &loaded)
		if err != nil {
			b.Fatalf("Load failed: %v", err)
		}
	}
}
