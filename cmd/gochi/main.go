package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Michael-W-Ellison/gochi/internal/core"
	"github.com/Michael-W-Ellison/gochi/internal/environment"
	"github.com/Michael-W-Ellison/gochi/internal/ui"
	"github.com/Michael-W-Ellison/gochi/pkg/logger"
	"github.com/Michael-W-Ellison/gochi/pkg/types"
)

func main() {
	// Create shutdown context
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Setup graceful shutdown handler
	shutdownComplete := make(chan struct{})
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Initialize display
	display := ui.NewDisplay()
	display.PrintWelcome()

	// Initialize environment
	envConfig := environment.DefaultEnvironmentConfig()
	env := environment.NewEnvironmentManager(envConfig)
	env.CreateStandardWorld()

	// Initialize game loop
	gameLoopConfig := core.DefaultGameLoopConfig()
	gameLoopConfig.TargetFPS = 10 // Lower FPS for terminal app
	gameLoopConfig.AutoSaveInterval = 5 * time.Minute

	gameLoop, err := core.NewGameLoop(gameLoopConfig)
	if err != nil {
		fmt.Printf("Failed to create game loop: %v\n", err)
		os.Exit(1)
	}

	// Start game loop in background
	err = gameLoop.Start()
	if err != nil {
		fmt.Printf("Failed to start game loop: %v\n", err)
		os.Exit(1)
	}

	// Signal handler goroutine
	go func() {
		sig := <-sigChan
		logger.Info("received shutdown signal", "signal", sig.String())
		cancel() // Cancel context to signal shutdown
	}()

	// Check for existing pet or create new one
	pet := loadOrCreatePet(gameLoop, display)
	if pet == nil {
		fmt.Println("Failed to create pet")
		os.Exit(1)
	}

	// Add pet to game loop
	err = gameLoop.AddPet(pet)
	if err != nil {
		fmt.Printf("Failed to add pet to game loop: %v\n", err)
		os.Exit(1)
	}

	// Main game loop
	display.PrintMessage("Welcome to Gochi! Your digital pet awaits...")
	time.Sleep(2 * time.Second)

	cmdProcessor := ui.NewCommandProcessor(display, gameLoop, env)

	// Game update ticker
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	lastDisplayTime := time.Now()
	displayInterval := 5 * time.Second

	// Main game loop with context-aware shutdown
	gameRunning := true
	for gameRunning && cmdProcessor.IsRunning() {
		select {
		case <-ctx.Done():
			// Shutdown signal received
			logger.Info("shutdown initiated")
			gameRunning = false
			continue

		default:
			// Periodically update environment
			env.Update(1.0) // 1 second of game time

			// Periodically redisplay status
			if time.Since(lastDisplayTime) > displayInterval {
				display.Clear()
				display.PrintPetStatus(pet)
				display.PrintEnvironment(env)
				lastDisplayTime = time.Now()
			}

			// Show menu and wait for command
			display.PrintMenu()

			// Read and process command
			command := cmdProcessor.ReadCommand()
			if command == "" {
				continue
			}

			continuing := cmdProcessor.ProcessCommand(command, pet)
			if !continuing {
				gameRunning = false
				break
			}

			// Small delay to let game loop update
			time.Sleep(100 * time.Millisecond)
		}
	}

	// Perform graceful shutdown with timeout
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	go func() {
		if err := performShutdown(gameLoop, display, env); err != nil {
			logger.Error("shutdown error", "error", err)
		}
		close(shutdownComplete)
	}()

	// Wait for shutdown to complete or timeout
	select {
	case <-shutdownComplete:
		logger.Info("shutdown completed successfully")
	case <-shutdownCtx.Done():
		logger.Error("shutdown timed out after 10 seconds")
	}

	fmt.Println("\nGoodbye! 👋")
}

// loadOrCreatePet loads an existing pet or creates a new one
func loadOrCreatePet(gameLoop *core.GameLoop, display *ui.Display) *core.DigitalPet {
	fmt.Print("\nDo you have an existing pet? (y/n): ")

	var response string
	fmt.Scanln(&response)

	if response == "y" || response == "yes" {
		fmt.Print("Enter pet ID: ")
		var petID string
		fmt.Scanln(&petID)

		// Try to load
		err := gameLoop.LoadPet(types.PetID(petID))
		if err != nil {
			display.PrintError("Failed to load pet, creating new one...")
		} else {
			pet, err := gameLoop.GetPet(types.PetID(petID))
			if err == nil && pet != nil {
				display.PrintSuccess(fmt.Sprintf("Loaded pet: %s", pet.Name))
				return pet
			}
		}
	}

	// Create new pet
	return createNewPet(display)
}

// createNewPet creates a new pet with user input
func createNewPet(display *ui.Display) *core.DigitalPet {
	fmt.Println("\n🐾 Creating a new pet!")

	fmt.Print("Enter pet name: ")
	var name string
	fmt.Scanln(&name)
	if name == "" {
		name = "Gochi"
	}

	// Create pet with default owner
	pet := core.NewDigitalPet(name, types.UserID("default-user"))

	display.PrintSuccess(fmt.Sprintf("Created %s!", name))
	display.PrintMessage(fmt.Sprintf("Pet ID: %s (save this to load your pet later!)", pet.ID))

	time.Sleep(2 * time.Second)

	return pet
}

// performShutdown performs graceful shutdown of all components
func performShutdown(gameLoop *core.GameLoop, display *ui.Display, env *environment.EnvironmentManager) error {
	logger.Info("starting graceful shutdown")
	display.PrintMessage("\nSaving game...")

	// Save all pets
	err := gameLoop.SaveAllPets()
	if err != nil {
		display.PrintError(fmt.Sprintf("Failed to save: %v", err))
		logger.Error("failed to save pets during shutdown", "error", err)
	} else {
		display.PrintSuccess("Game saved!")
		logger.Info("all pets saved successfully")
	}

	// Print summary before shutdown
	pets := gameLoop.GetAllPets()
	if len(pets) > 0 {
		for _, pet := range pets {
			stats := gameLoop.GetStatistics()
			display.PrintSummary(pet, stats["total_game_time"].(float64))
			break // Just show first pet
		}
	}

	// Shutdown game loop (stops and closes resources)
	if err := gameLoop.Shutdown(); err != nil {
		logger.Error("failed to shutdown game loop", "error", err)
		return fmt.Errorf("game loop shutdown failed: %w", err)
	}

	// Shutdown logger (flush and close log files)
	if err := logger.Shutdown(); err != nil {
		// Can't log this error since logger is shutting down
		fmt.Fprintf(os.Stderr, "Failed to shutdown logger: %v\n", err)
		return fmt.Errorf("logger shutdown failed: %w", err)
	}

	return nil
}
