package ui

import (
	"testing"
	"time"

	"github.com/Michael-W-Ellison/gochi/internal/core"
	"github.com/Michael-W-Ellison/gochi/internal/environment"
	"github.com/Michael-W-Ellison/gochi/pkg/types"
)

func TestNewDisplay(t *testing.T) {
	display := NewDisplay()

	if display == nil {
		t.Fatal("NewDisplay returned nil")
	}

	if display.width != 80 {
		t.Errorf("Expected width 80, got %d", display.width)
	}

	if display.height != 24 {
		t.Errorf("Expected height 24, got %d", display.height)
	}
}

func TestFormatTrait(t *testing.T) {
	display := NewDisplay()

	tests := []struct {
		value    float64
		expected string
	}{
		{0.9, "⭐⭐⭐"},
		{0.71, "⭐⭐⭐"},
		{0.7, "⭐⭐"},
		{0.6, "⭐⭐"},
		{0.41, "⭐⭐"},
		{0.4, "⭐"},
		{0.3, "⭐"},
		{0.0, "⭐"},
	}

	for _, test := range tests {
		result := display.formatTrait(test.value)
		if result != test.expected {
			t.Errorf("formatTrait(%.1f) = %s, expected %s", test.value, result, test.expected)
		}
	}
}

func TestPrintPetStatus(t *testing.T) {
	display := NewDisplay()

	// Test with nil pet
	display.PrintPetStatus(nil)

	// Test with valid pet
	pet := createTestPet()
	display.PrintPetStatus(pet)
}

func TestPrintEnvironment(t *testing.T) {
	display := NewDisplay()

	// Test with nil environment
	display.PrintEnvironment(nil)

	// Test with valid environment
	envConfig := environment.DefaultEnvironmentConfig()
	env := environment.NewEnvironmentManager(envConfig)
	display.PrintEnvironment(env)
}

func TestPrintLocationsList(t *testing.T) {
	display := NewDisplay()

	locations := []*environment.Location{
		{
			Name:       "Forest Clearing",
			Biome:      environment.BiomeForest,
			Discovered: true,
		},
		{
			Name:       "Mountain Peak",
			Biome:      environment.BiomeMountain,
			Discovered: false,
		},
	}

	display.PrintLocationsList(locations)
}

func TestPrintPetList(t *testing.T) {
	display := NewDisplay()

	// Test with empty map
	emptyPets := make(map[types.PetID]*core.DigitalPet)
	display.PrintPetList(emptyPets)

	// Test with pets
	pets := make(map[types.PetID]*core.DigitalPet)
	pet1 := createTestPet()
	pet2 := createTestPet()
	pet2.Name = "Buddy"

	pets[pet1.ID] = pet1
	pets[pet2.ID] = pet2

	display.PrintPetList(pets)
}

func TestPrintMessages(t *testing.T) {
	display := NewDisplay()

	display.PrintMessage("Test message")
	display.PrintError("Test error")
	display.PrintSuccess("Test success")
	display.PrintWarning("Test warning")
	display.PrintInteractionResult("Test interaction result")
}

func TestPrintWelcome(t *testing.T) {
	display := NewDisplay()
	display.PrintWelcome()
}

func TestPrintMenu(t *testing.T) {
	display := NewDisplay()
	display.PrintMenu()
}

func TestPrintGameStats(t *testing.T) {
	display := NewDisplay()

	gameLoopConfig := core.DefaultGameLoopConfig()
	gameLoop, err := core.NewGameLoop(gameLoopConfig)
	if err != nil {
		t.Fatalf("Failed to create game loop: %v", err)
	}

	display.PrintGameStats(gameLoop)
}

func TestPrintSummary(t *testing.T) {
	display := NewDisplay()

	pet := createTestPet()
	display.PrintSummary(pet, 3600.0) // 1 hour

	display.PrintSummary(nil, 3600.0)
}

func TestAnimateDots(t *testing.T) {
	display := NewDisplay()

	// Test animating dots with short duration
	done := make(chan bool)
	go func() {
		display.AnimateDots(100*time.Millisecond, "Testing")
		done <- true
	}()

	select {
	case <-done:
		// Animation completed successfully
	case <-time.After(500 * time.Millisecond):
		t.Error("AnimateDots took too long to complete")
	}
}

func TestPrintHeaderAndClear(t *testing.T) {
	display := NewDisplay()

	// Test Clear
	display.Clear()

	// Test PrintHeader
	display.PrintHeader("Test Header")
}

func TestPrintSeparator(t *testing.T) {
	display := NewDisplay()

	display.printSeparator()
}
