package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/Michael-W-Ellison/gochi/internal/core"
	"github.com/Michael-W-Ellison/gochi/internal/environment"
	"github.com/Michael-W-Ellison/gochi/pkg/types"
)

// Display handles terminal output formatting
type Display struct {
	width  int
	height int
}

// NewDisplay creates a new display
func NewDisplay() *Display {
	return &Display{
		width:  80,
		height: 24,
	}
}

// Clear clears the terminal screen
func (d *Display) Clear() {
	fmt.Print("\033[H\033[2J")
}

// PrintHeader prints a styled header
func (d *Display) PrintHeader(text string) {
	d.printSeparator()
	fmt.Printf("║ %-*s ║\n", d.width-4, text)
	d.printSeparator()
}

// PrintPetStatus displays comprehensive pet information
func (d *Display) PrintPetStatus(pet *core.DigitalPet) {
	if pet == nil {
		fmt.Println("No pet loaded")
		return
	}

	d.PrintHeader(fmt.Sprintf("🐾 %s", pet.Name))

	// Basic info
	fmt.Printf("Age: %.1f days | Born: %s\n",
		pet.Biology.GetAgeInDays(),
		pet.Biology.BirthTime.Format("Jan 2"))

	if !pet.Biology.IsAlive {
		fmt.Println("💀 Status: DECEASED")
		return
	}

	// Vitals bar
	fmt.Println("\n📊 Vitals:")
	d.printBar("Health", pet.Biology.Vitals.Health, 100)
	d.printBar("Hunger", 1.0-pet.Biology.Vitals.Nutrition, 100)
	d.printBar("Thirst", 1.0-pet.Biology.Vitals.Hydration, 100)
	d.printBar("Energy", pet.Biology.Vitals.Energy, 100)
	d.printBar("Clean", pet.Biology.Vitals.Cleanliness, 100)

	// Emotions
	fmt.Println("\n😊 Emotions:")
	d.printBar("Joy", pet.Emotions.Joy, 100)
	d.printBar("Excitement", pet.Emotions.Excitement, 100)
	d.printBar("Affection", pet.Emotions.Affection, 100)

	// Current behavior
	fmt.Printf("\n🎭 Behavior: %s\n", pet.CurrentBehavior.String())

	// Personality traits
	fmt.Println("\n✨ Personality:")
	traits := pet.Personality.Traits
	fmt.Printf("  Playful: %s | Affectionate: %s | Independent: %s\n",
		d.formatTrait(traits.Playfulness),
		d.formatTrait(traits.Affectionate),
		d.formatTrait(traits.Independence))
	fmt.Printf("  Curious: %s  | Energetic: %s    | Calm: %s\n",
		d.formatTrait(traits.Curiosity),
		d.formatTrait(traits.EnergyLevel),
		d.formatTrait(1.0-traits.Neuroticism))
}

// PrintEnvironment displays environmental information
func (d *Display) PrintEnvironment(env *environment.EnvironmentManager) {
	if env == nil {
		return
	}

	fmt.Println("\n🌍 Environment:")

	// Season
	season := env.GetCurrentSeason()
	effects := env.CalculateEnvironmentalEffects()
	fmt.Printf("  Season: %s (%.0f%% through)\n",
		season.String(),
		env.GetSeasonProgress()*100)

	// Weather
	weather := env.GetWeatherConditions()
	fmt.Printf("  Weather: %s, %.1f°C\n",
		weather.Type.String(),
		weather.Temperature)

	// Location
	location := env.GetCurrentLocation()
	if location != nil {
		fmt.Printf("  Location: %s (%s)\n",
			location.Name,
			location.Biome.String())
	}

	// Environmental effects
	fmt.Printf("  Food: %.0f%% | Comfort: %.0f%%\n",
		effects.FoodAvailability*100,
		effects.OverallComfort*100)
}

// PrintMenu displays available commands
func (d *Display) PrintMenu() {
	fmt.Println("\n📋 Commands:")
	fmt.Println("  feed    - Feed your pet")
	fmt.Println("  water   - Give water to your pet")
	fmt.Println("  play    - Play with your pet")
	fmt.Println("  pet     - Pet your pet")
	fmt.Println("  clean   - Clean your pet")
	fmt.Println("  sleep   - Let your pet rest")
	fmt.Println("  status  - Show full status")
	fmt.Println("  travel  - Travel to a new location")
	fmt.Println("  wait    - Wait and let time pass")
	fmt.Println("  save    - Save your pet")
	fmt.Println("  quit    - Exit game")
	fmt.Print("\n> ")
}

// PrintGameStats displays game statistics
func (d *Display) PrintGameStats(gameLoop *core.GameLoop) {
	stats := gameLoop.GetStatistics()

	fmt.Println("\n📈 Game Stats:")
	fmt.Printf("  FPS: %.1f | Updates: %d | Game Time: %.1fh\n",
		stats["current_fps"].(float64),
		stats["total_updates"].(int64),
		stats["total_game_time"].(float64)/3600.0)
}

// printBar prints a progress bar
func (d *Display) printBar(label string, value float64, width int) {
	if value < 0 {
		value = 0
	}
	if value > 1 {
		value = 1
	}

	barWidth := 20
	filled := int(value * float64(barWidth))
	empty := barWidth - filled

	bar := strings.Repeat("█", filled) + strings.Repeat("░", empty)

	// Color based on value
	color := ""
	if value > 0.7 {
		color = "\033[32m" // Green
	} else if value > 0.3 {
		color = "\033[33m" // Yellow
	} else {
		color = "\033[31m" // Red
	}

	fmt.Printf("  %-10s [%s%s\033[0m] %3.0f%%\n", label, color, bar, value*100)
}

// formatTrait formats a personality trait for display
func (d *Display) formatTrait(value float64) string {
	if value > 0.7 {
		return "⭐⭐⭐"
	} else if value > 0.4 {
		return "⭐⭐"
	} else {
		return "⭐"
	}
}

// printSeparator prints a horizontal line
func (d *Display) printSeparator() {
	fmt.Println(strings.Repeat("═", d.width))
}

// PrintMessage prints a formatted message
func (d *Display) PrintMessage(msg string) {
	fmt.Printf("💬 %s\n", msg)
}

// PrintError prints an error message
func (d *Display) PrintError(msg string) {
	fmt.Printf("\033[31m❌ Error: %s\033[0m\n", msg)
}

// PrintSuccess prints a success message
func (d *Display) PrintSuccess(msg string) {
	fmt.Printf("\033[32m✅ %s\033[0m\n", msg)
}

// PrintWarning prints a warning message
func (d *Display) PrintWarning(msg string) {
	fmt.Printf("\033[33m⚠️  %s\033[0m\n", msg)
}

// PrintWelcome prints the welcome screen
func (d *Display) PrintWelcome() {
	d.Clear()
	fmt.Println(`
	╔═══════════════════════════════════════════════════════════╗
	║                                                           ║
	║     🐾  G O C H I  -  Digital Pet Simulator  🐾         ║
	║                                                           ║
	║        Advanced Tamagotchi-Style Pet System              ║
	║                                                           ║
	╚═══════════════════════════════════════════════════════════╝
	`)
}

// PrintInteractionResult displays the result of an interaction
func (d *Display) PrintInteractionResult(result string) {
	fmt.Printf("\n🎮 %s\n", result)
}

// PrintLocationsList displays available locations
func (d *Display) PrintLocationsList(locations []*environment.Location) {
	fmt.Println("\n🗺️  Available Locations:")
	for i, loc := range locations {
		status := "Undiscovered"
		if loc.Discovered {
			status = "Discovered"
		}
		fmt.Printf("  %d. %s (%s) - %s\n",
			i+1,
			loc.Name,
			loc.Biome.String(),
			status)
	}
}

// PrintPetList displays a list of pets
func (d *Display) PrintPetList(pets map[types.PetID]*core.DigitalPet) {
	if len(pets) == 0 {
		fmt.Println("No pets found")
		return
	}

	fmt.Println("\n🐾 Your Pets:")
	for petID, pet := range pets {
		status := "Alive"
		if !pet.Biology.IsAlive {
			status = "Deceased"
		}
		fmt.Printf("  %s - %s (Age: %.1f days) - %s\n",
			petID,
			pet.Name,
			pet.Biology.GetAgeInDays(),
			status)
	}
}

// AnimateDots shows animated dots for waiting
func (d *Display) AnimateDots(duration time.Duration, message string) {
	fmt.Print(message)
	dots := 0
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	done := time.After(duration)

	for {
		select {
		case <-done:
			fmt.Println()
			return
		case <-ticker.C:
			fmt.Print(".")
			dots++
			if dots >= 3 {
				fmt.Print("\b\b\b   \b\b\b")
				dots = 0
			}
		}
	}
}

// PrintSummary prints a game session summary
func (d *Display) PrintSummary(pet *core.DigitalPet, gameTime float64) {
	d.printSeparator()
	fmt.Println("📊 Session Summary")
	d.printSeparator()

	if pet != nil {
		fmt.Printf("Pet: %s\n", pet.Name)
		fmt.Printf("Age: %.1f days\n", pet.Biology.GetAgeInDays())
		fmt.Printf("Final Joy: %.1f%%\n", pet.Emotions.Joy*100)
	}

	fmt.Printf("Session Duration: %.1f minutes\n", gameTime/60.0)
	fmt.Println("\nThank you for playing Gochi! 👋")
	d.printSeparator()
}
