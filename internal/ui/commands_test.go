package ui

import (
	"testing"

	"github.com/Michael-W-Ellison/gochi/internal/core"
	"github.com/Michael-W-Ellison/gochi/internal/environment"
)

func TestNewCommandProcessor(t *testing.T) {
	display := NewDisplay()
	envConfig := environment.DefaultEnvironmentConfig()
	env := environment.NewEnvironmentManager(envConfig)

	gameLoopConfig := core.DefaultGameLoopConfig()
	gameLoop, err := core.NewGameLoop(gameLoopConfig)
	if err != nil {
		t.Fatalf("Failed to create game loop: %v", err)
	}

	cmdProcessor := NewCommandProcessor(display, gameLoop, env)

	if cmdProcessor == nil {
		t.Fatal("NewCommandProcessor returned nil")
	}

	if cmdProcessor.display != display {
		t.Error("Display not set correctly")
	}

	if cmdProcessor.gameLoop != gameLoop {
		t.Error("GameLoop not set correctly")
	}

	if cmdProcessor.environment != env {
		t.Error("Environment not set correctly")
	}

	if !cmdProcessor.IsRunning() {
		t.Error("Command processor should be running initially")
	}
}

func TestProcessCommandWithNoPet(t *testing.T) {
	display := NewDisplay()
	envConfig := environment.DefaultEnvironmentConfig()
	env := environment.NewEnvironmentManager(envConfig)

	gameLoopConfig := core.DefaultGameLoopConfig()
	gameLoop, err := core.NewGameLoop(gameLoopConfig)
	if err != nil {
		t.Fatalf("Failed to create game loop: %v", err)
	}

	cmdProcessor := NewCommandProcessor(display, gameLoop, env)

	// Test with nil pet
	result := cmdProcessor.ProcessCommand("feed", nil)
	if !result {
		t.Error("ProcessCommand should return true for nil pet")
	}
}

func TestProcessCommandWithPet(t *testing.T) {
	display := NewDisplay()
	envConfig := environment.DefaultEnvironmentConfig()
	env := environment.NewEnvironmentManager(envConfig)

	gameLoopConfig := core.DefaultGameLoopConfig()
	gameLoop, err := core.NewGameLoop(gameLoopConfig)
	if err != nil {
		t.Fatalf("Failed to create game loop: %v", err)
	}

	// Start game loop
	err = gameLoop.Start()
	if err != nil {
		t.Fatalf("Failed to start game loop: %v", err)
	}
	defer gameLoop.Stop()

	// Create and add pet
	pet := createTestPet()
	err = gameLoop.AddPet(pet)
	if err != nil {
		t.Fatalf("Failed to add pet: %v", err)
	}

	cmdProcessor := NewCommandProcessor(display, gameLoop, env)

	tests := []struct {
		command string
		returns bool
	}{
		{"help", true},
		{"menu", true},
		{"status", true},
		{"feed", true},
		{"water", true},
		{"play", true},
		{"pet", true},
		{"clean", true},
		{"save", true},
		{"unknown", true},
	}

	for _, test := range tests {
		result := cmdProcessor.ProcessCommand(test.command, pet)
		if result != test.returns {
			t.Errorf("ProcessCommand(%s) = %v, expected %v", test.command, result, test.returns)
		}
	}
}

func TestHandleQuit(t *testing.T) {
	display := NewDisplay()
	envConfig := environment.DefaultEnvironmentConfig()
	env := environment.NewEnvironmentManager(envConfig)

	gameLoopConfig := core.DefaultGameLoopConfig()
	gameLoop, err := core.NewGameLoop(gameLoopConfig)
	if err != nil {
		t.Fatalf("Failed to create game loop: %v", err)
	}

	cmdProcessor := NewCommandProcessor(display, gameLoop, env)

	if !cmdProcessor.IsRunning() {
		t.Error("Command processor should be running initially")
	}

	// Note: handleQuit requires user input, so we can't easily test it
	// We just verify the initial state
}

func TestProcessEmptyCommand(t *testing.T) {
	display := NewDisplay()
	envConfig := environment.DefaultEnvironmentConfig()
	env := environment.NewEnvironmentManager(envConfig)

	gameLoopConfig := core.DefaultGameLoopConfig()
	gameLoop, err := core.NewGameLoop(gameLoopConfig)
	if err != nil {
		t.Fatalf("Failed to create game loop: %v", err)
	}

	cmdProcessor := NewCommandProcessor(display, gameLoop, env)
	pet := createTestPet()

	// Test empty command
	result := cmdProcessor.ProcessCommand("", pet)
	if !result {
		t.Error("ProcessCommand with empty string should return true")
	}

	// Test whitespace command
	result = cmdProcessor.ProcessCommand("   ", pet)
	if !result {
		t.Error("ProcessCommand with whitespace should return true")
	}
}

func TestHandleSleep(t *testing.T) {
	display := NewDisplay()
	envConfig := environment.DefaultEnvironmentConfig()
	env := environment.NewEnvironmentManager(envConfig)

	gameLoopConfig := core.DefaultGameLoopConfig()
	gameLoop, err := core.NewGameLoop(gameLoopConfig)
	if err != nil {
		t.Fatalf("Failed to create game loop: %v", err)
	}

	// Start game loop
	err = gameLoop.Start()
	if err != nil {
		t.Fatalf("Failed to start game loop: %v", err)
	}
	defer gameLoop.Stop()

	pet := createTestPet()
	err = gameLoop.AddPet(pet)
	if err != nil {
		t.Fatalf("Failed to add pet: %v", err)
	}

	cmdProcessor := NewCommandProcessor(display, gameLoop, env)

	// Test sleep command
	result := cmdProcessor.ProcessCommand("sleep", pet)
	if !result {
		t.Error("ProcessCommand('sleep') should return true")
	}
}

func TestHandleTravel(t *testing.T) {
	display := NewDisplay()
	envConfig := environment.DefaultEnvironmentConfig()
	env := environment.NewEnvironmentManager(envConfig)

	gameLoopConfig := core.DefaultGameLoopConfig()
	gameLoop, err := core.NewGameLoop(gameLoopConfig)
	if err != nil {
		t.Fatalf("Failed to create game loop: %v", err)
	}

	// Start game loop
	err = gameLoop.Start()
	if err != nil {
		t.Fatalf("Failed to start game loop: %v", err)
	}
	defer gameLoop.Stop()

	pet := createTestPet()
	err = gameLoop.AddPet(pet)
	if err != nil {
		t.Fatalf("Failed to add pet: %v", err)
	}

	cmdProcessor := NewCommandProcessor(display, gameLoop, env)

	// Test travel command
	result := cmdProcessor.ProcessCommand("travel", pet)
	if !result {
		t.Error("ProcessCommand('travel') should return true")
	}
}

func TestHandleWait(t *testing.T) {
	display := NewDisplay()
	envConfig := environment.DefaultEnvironmentConfig()
	env := environment.NewEnvironmentManager(envConfig)

	gameLoopConfig := core.DefaultGameLoopConfig()
	gameLoop, err := core.NewGameLoop(gameLoopConfig)
	if err != nil {
		t.Fatalf("Failed to create game loop: %v", err)
	}

	// Start game loop
	err = gameLoop.Start()
	if err != nil {
		t.Fatalf("Failed to start game loop: %v", err)
	}
	defer gameLoop.Stop()

	pet := createTestPet()
	err = gameLoop.AddPet(pet)
	if err != nil {
		t.Fatalf("Failed to add pet: %v", err)
	}

	cmdProcessor := NewCommandProcessor(display, gameLoop, env)

	// Test wait command
	result := cmdProcessor.ProcessCommand("wait", pet)
	if !result {
		t.Error("ProcessCommand('wait') should return true")
	}
}

func createTestPet() *core.DigitalPet {
	pet := core.NewDigitalPet("TestPet", "user-123")
	return pet
}
