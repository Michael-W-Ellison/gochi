package ui

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/Michael-W-Ellison/gochi/internal/core"
	"github.com/Michael-W-Ellison/gochi/internal/environment"
	"github.com/Michael-W-Ellison/gochi/pkg/types"
)

// CommandProcessor handles user input and executes commands
type CommandProcessor struct {
	display     *Display
	gameLoop    *core.GameLoop
	environment *environment.EnvironmentManager
	scanner     *bufio.Scanner
	running     bool
}

// NewCommandProcessor creates a new command processor
func NewCommandProcessor(display *Display, gameLoop *core.GameLoop, env *environment.EnvironmentManager) *CommandProcessor {
	return &CommandProcessor{
		display:     display,
		gameLoop:    gameLoop,
		environment: env,
		scanner:     bufio.NewScanner(os.Stdin),
		running:     true,
	}
}

// ReadCommand reads a command from user input
func (cp *CommandProcessor) ReadCommand() string {
	if cp.scanner.Scan() {
		return strings.TrimSpace(strings.ToLower(cp.scanner.Text()))
	}
	return ""
}

// ProcessCommand processes a user command
func (cp *CommandProcessor) ProcessCommand(command string, pet *core.DigitalPet) bool {
	if pet == nil {
		cp.display.PrintError("No pet loaded")
		return true
	}

	parts := strings.Fields(command)
	if len(parts) == 0 {
		return true
	}

	cmd := parts[0]

	switch cmd {
	case "feed":
		return cp.handleFeed(pet)

	case "water":
		return cp.handleWater(pet)

	case "play":
		return cp.handlePlay(pet)

	case "pet":
		return cp.handlePet(pet)

	case "clean":
		return cp.handleClean(pet)

	case "sleep":
		return cp.handleSleep(pet)

	case "status":
		return cp.handleStatus(pet)

	case "travel":
		return cp.handleTravel(pet)

	case "wait":
		return cp.handleWait(pet)

	case "save":
		return cp.handleSave(pet)

	case "help", "menu":
		cp.display.PrintMenu()
		return true

	case "quit", "exit":
		return cp.handleQuit()

	default:
		cp.display.PrintError(fmt.Sprintf("Unknown command: %s", cmd))
		cp.display.PrintMessage("Type 'help' for available commands")
		return true
	}
}

// handleFeed feeds the pet
func (cp *CommandProcessor) handleFeed(pet *core.DigitalPet) bool {
	cp.display.PrintMessage("Feeding your pet...")

	result, err := cp.gameLoop.ProcessInteraction(
		pet.ID,
		types.InteractionFeeding,
		1.0,
		nil,
	)

	if err != nil {
		cp.display.PrintError(fmt.Sprintf("Failed to feed: %v", err))
		return true
	}

	cp.display.PrintSuccess(result.Feedback)

	// Show any vital changes
	for vital, change := range result.VitalChanges {
		if change != 0 {
			cp.display.PrintMessage(fmt.Sprintf("%s: %+.1f", vital, change))
		}
	}

	// Show warnings
	for _, warning := range result.Warnings {
		cp.display.PrintError(warning)
	}

	return true
}

// handleWater gives water to the pet
func (cp *CommandProcessor) handleWater(pet *core.DigitalPet) bool {
	cp.display.PrintMessage("Giving water to your pet...")

	// Use feeding interaction with lower intensity for water
	result, err := cp.gameLoop.ProcessInteraction(
		pet.ID,
		types.InteractionFeeding,
		0.3, // Lower intensity for just water
		"water",
	)

	if err != nil {
		cp.display.PrintError(fmt.Sprintf("Failed to give water: %v", err))
		return true
	}

	cp.display.PrintSuccess(result.Feedback)

	for vital, change := range result.VitalChanges {
		if change != 0 {
			cp.display.PrintMessage(fmt.Sprintf("%s: %+.1f", vital, change))
		}
	}

	for _, warning := range result.Warnings {
		cp.display.PrintError(warning)
	}

	return true
}

// handlePlay plays with the pet
func (cp *CommandProcessor) handlePlay(pet *core.DigitalPet) bool {
	cp.display.PrintMessage("Playing with your pet...")

	result, err := cp.gameLoop.ProcessInteraction(
		pet.ID,
		types.InteractionPlaying,
		0.8,
		nil,
	)

	if err != nil {
		cp.display.PrintError(fmt.Sprintf("Failed to play: %v", err))
		return true
	}

	cp.display.PrintSuccess(result.Feedback)

	for vital, change := range result.VitalChanges {
		if change != 0 {
			cp.display.PrintMessage(fmt.Sprintf("%s: %+.1f", vital, change))
		}
	}

	for emotion, change := range result.EmotionChanges {
		if change != 0 {
			cp.display.PrintMessage(fmt.Sprintf("%s: %+.1f", emotion, change))
		}
	}

	for _, warning := range result.Warnings {
		cp.display.PrintError(warning)
	}

	return true
}

// handlePet pets the pet (affection)
func (cp *CommandProcessor) handlePet(pet *core.DigitalPet) bool {
	cp.display.PrintMessage("Petting your pet...")

	result, err := cp.gameLoop.ProcessInteraction(
		pet.ID,
		types.InteractionPetting,
		1.0,
		nil,
	)

	if err != nil {
		cp.display.PrintError(fmt.Sprintf("Failed to pet: %v", err))
		return true
	}

	cp.display.PrintSuccess(result.Feedback)

	for emotion, change := range result.EmotionChanges {
		if change != 0 {
			cp.display.PrintMessage(fmt.Sprintf("%s: %+.1f", emotion, change))
		}
	}

	for _, warning := range result.Warnings {
		cp.display.PrintError(warning)
	}

	return true
}

// handleClean cleans the pet
func (cp *CommandProcessor) handleClean(pet *core.DigitalPet) bool {
	cp.display.PrintMessage("Cleaning your pet...")

	result, err := cp.gameLoop.ProcessInteraction(
		pet.ID,
		types.InteractionGrooming,
		1.0,
		nil,
	)

	if err != nil {
		cp.display.PrintError(fmt.Sprintf("Failed to clean: %v", err))
		return true
	}

	cp.display.PrintSuccess(result.Feedback)

	for vital, change := range result.VitalChanges {
		if change != 0 {
			cp.display.PrintMessage(fmt.Sprintf("%s: %+.1f", vital, change))
		}
	}

	for _, warning := range result.Warnings {
		cp.display.PrintError(warning)
	}

	return true
}

// handleSleep lets the pet rest
func (cp *CommandProcessor) handleSleep(pet *core.DigitalPet) bool {
	cp.display.PrintMessage("Your pet is resting...")

	// Let time pass
	cp.display.AnimateDots(3*1000000000, "Sleeping")

	// Several updates will happen during this time
	cp.display.PrintSuccess("Your pet feels more rested!")

	return true
}

// handleStatus shows full status
func (cp *CommandProcessor) handleStatus(pet *core.DigitalPet) bool {
	cp.display.Clear()
	cp.display.PrintPetStatus(pet)
	cp.display.PrintEnvironment(cp.environment)
	cp.display.PrintGameStats(cp.gameLoop)
	return true
}

// handleTravel allows traveling to new locations
func (cp *CommandProcessor) handleTravel(pet *core.DigitalPet) bool {
	// Get nearby locations
	locations := cp.environment.GetNearbyLocations(200.0)

	if len(locations) == 0 {
		cp.display.PrintMessage("No nearby locations available")
		return true
	}

	cp.display.PrintLocationsList(locations)
	fmt.Print("\nEnter location number (or 0 to cancel): ")

	choice := cp.ReadCommand()
	if choice == "0" || choice == "" {
		return true
	}

	// Parse choice (simplified - would need better parsing)
	var index int
	fmt.Sscanf(choice, "%d", &index)

	if index < 1 || index > len(locations) {
		cp.display.PrintError("Invalid choice")
		return true
	}

	location := locations[index-1]
	success := cp.environment.TravelToLocation(location.Name)

	if success {
		cp.display.PrintSuccess(fmt.Sprintf("Traveled to %s!", location.Name))
		cp.display.PrintMessage(location.GetDescription())
	} else {
		cp.display.PrintError("Failed to travel")
	}

	return true
}

// handleWait waits and lets time pass
func (cp *CommandProcessor) handleWait(pet *core.DigitalPet) bool {
	cp.display.PrintMessage("Time passes...")
	cp.display.AnimateDots(2*1000000000, "Waiting")
	cp.display.PrintSuccess("A moment has passed")
	return true
}

// handleSave saves the pet
func (cp *CommandProcessor) handleSave(pet *core.DigitalPet) bool {
	cp.display.PrintMessage("Saving...")

	err := cp.gameLoop.SavePet(pet.ID)
	if err != nil {
		cp.display.PrintError(fmt.Sprintf("Failed to save: %v", err))
		return true
	}

	cp.display.PrintSuccess("Pet saved successfully!")
	return true
}

// handleQuit quits the game
func (cp *CommandProcessor) handleQuit() bool {
	fmt.Print("\nAre you sure you want to quit? (y/n): ")
	response := cp.ReadCommand()

	if response == "y" || response == "yes" {
		cp.running = false
		return false
	}

	return true
}

// IsRunning returns whether the command processor is still running
func (cp *CommandProcessor) IsRunning() bool {
	return cp.running
}
