package interaction

import (
	"testing"
)

func TestGetFoodProperties(t *testing.T) {
	tests := []struct {
		name         string
		foodType     FoodType
		wantName     string
		minNutrition float64
		minHappiness float64
	}{
		{"Basic Kibble", FoodTypeBasicKibble, "Basic Kibble", 0.5, 0.1},
		{"Premium Kibble", FoodTypePremiumKibble, "Premium Kibble", 0.7, 0.3},
		{"Special Meal", FoodTypeSpecialMeal, "Special Meal", 0.9, 0.9},
		{"Treat", FoodTypeTreat, "Treat", 0.1, 0.7},
		{"Junk Food", FoodTypeJunkFood, "Junk Food", 0.2, 0.8},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			props := GetFoodProperties(tt.foodType)

			if props.Name != tt.wantName {
				t.Errorf("Name = %s, want %s", props.Name, tt.wantName)
			}

			if props.NutritionValue < tt.minNutrition {
				t.Errorf("NutritionValue = %f, want >= %f", props.NutritionValue, tt.minNutrition)
			}

			if props.HappinessBoost < tt.minHappiness {
				t.Errorf("HappinessBoost = %f, want >= %f", props.HappinessBoost, tt.minHappiness)
			}

			// All values should be in valid ranges
			if props.NutritionValue < 0 || props.NutritionValue > 1 {
				t.Errorf("NutritionValue %f out of range [0,1]", props.NutritionValue)
			}

			if props.HydrationValue < 0 || props.HydrationValue > 1 {
				t.Errorf("HydrationValue %f out of range [0,1]", props.HydrationValue)
			}

			if props.EnergyBoost < 0 || props.EnergyBoost > 1 {
				t.Errorf("EnergyBoost %f out of range [0,1]", props.EnergyBoost)
			}

			if props.HealthImpact < -1 || props.HealthImpact > 1 {
				t.Errorf("HealthImpact %f out of range [-1,1]", props.HealthImpact)
			}

			if props.HappinessBoost < 0 || props.HappinessBoost > 1 {
				t.Errorf("HappinessBoost %f out of range [0,1]", props.HappinessBoost)
			}

			if props.FillAmount < 0 || props.FillAmount > 1 {
				t.Errorf("FillAmount %f out of range [0,1]", props.FillAmount)
			}
		})
	}
}

func TestFoodTypeHealthImpact(t *testing.T) {
	// Test that unhealthy foods have negative health impact
	junkProps := GetFoodProperties(FoodTypeJunkFood)
	if junkProps.HealthImpact >= 0 {
		t.Error("Junk food should have negative health impact")
	}

	// Healthy foods should have positive impact
	vegProps := GetFoodProperties(FoodTypeVegetable)
	if vegProps.HealthImpact <= 0 {
		t.Error("Vegetables should have positive health impact")
	}

	fruitProps := GetFoodProperties(FoodTypeFruit)
	if fruitProps.HealthImpact <= 0 {
		t.Error("Fruit should have positive health impact")
	}
}

func TestFoodTypeHydration(t *testing.T) {
	// Fruits should provide good hydration
	fruitProps := GetFoodProperties(FoodTypeFruit)
	if fruitProps.HydrationValue < 0.5 {
		t.Error("Fruit should provide good hydration")
	}

	// Dairy should provide good hydration
	dairyProps := GetFoodProperties(FoodTypeDairy)
	if dairyProps.HydrationValue < 0.5 {
		t.Error("Dairy should provide good hydration")
	}

	// Dry foods should provide less hydration
	kibbleProps := GetFoodProperties(FoodTypeBasicKibble)
	if kibbleProps.HydrationValue > 0.3 {
		t.Error("Dry kibble should provide minimal hydration")
	}
}

func TestGetToyProperties(t *testing.T) {
	tests := []struct {
		name             string
		toyType          ToyType
		wantName         string
		expectHighEnergy bool
		expectHighMental bool
	}{
		{"Ball", ToyTypeBall, "Ball", true, false},
		{"Puzzle", ToyTypePuzzle, "Puzzle", false, true},
		{"Interactive", ToyTypeInteractive, "Interactive Toy", false, true}, // Moderate physical, high mental
		{"Laser", ToyTypeLaser, "Laser Pointer", true, false},
		{"Chew Toy", ToyTypeChewToy, "Chew Toy", false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			props := GetToyProperties(tt.toyType)

			if props.Name != tt.wantName {
				t.Errorf("Name = %s, want %s", props.Name, tt.wantName)
			}

			if tt.expectHighEnergy && props.PhysicalExercise < 0.7 {
				t.Errorf("Expected high physical exercise for %s", tt.name)
			}

			if tt.expectHighMental && props.MentalStimulation < 0.7 {
				t.Errorf("Expected high mental stimulation for %s", tt.name)
			}

			// All values should be in valid ranges
			if props.EnergyDrain < 0 || props.EnergyDrain > 1 {
				t.Errorf("EnergyDrain %f out of range [0,1]", props.EnergyDrain)
			}

			if props.MentalStimulation < 0 || props.MentalStimulation > 1 {
				t.Errorf("MentalStimulation %f out of range [0,1]", props.MentalStimulation)
			}

			if props.PhysicalExercise < 0 || props.PhysicalExercise > 1 {
				t.Errorf("PhysicalExercise %f out of range [0,1]", props.PhysicalExercise)
			}

			if props.SocialEngagement < 0 || props.SocialEngagement > 1 {
				t.Errorf("SocialEngagement %f out of range [0,1]", props.SocialEngagement)
			}

			if props.FunFactor < 0 || props.FunFactor > 1 {
				t.Errorf("FunFactor %f out of range [0,1]", props.FunFactor)
			}

			if props.SkillDevelopment < 0 || props.SkillDevelopment > 1 {
				t.Errorf("SkillDevelopment %f out of range [0,1]", props.SkillDevelopment)
			}

			if props.DurabilityRequired < 0 || props.DurabilityRequired > 1 {
				t.Errorf("DurabilityRequired %f out of range [0,1]", props.DurabilityRequired)
			}
		})
	}
}

func TestToyTypePuzzleCharacteristics(t *testing.T) {
	props := GetToyProperties(ToyTypePuzzle)

	// Puzzle toys should be mentally stimulating
	if props.MentalStimulation < 0.8 {
		t.Error("Puzzle toys should have high mental stimulation")
	}

	// But not very physical
	if props.PhysicalExercise > 0.3 {
		t.Error("Puzzle toys should have low physical exercise")
	}

	// High skill development
	if props.SkillDevelopment < 0.8 {
		t.Error("Puzzle toys should have high skill development")
	}

	// Low energy drain
	if props.EnergyDrain > 0.4 {
		t.Error("Puzzle toys should have low energy drain")
	}
}

func TestToyTypeBallCharacteristics(t *testing.T) {
	props := GetToyProperties(ToyTypeBall)

	// Ball should be physically demanding
	if props.PhysicalExercise < 0.8 {
		t.Error("Ball should have high physical exercise")
	}

	// High energy drain
	if props.EnergyDrain < 0.6 {
		t.Error("Ball should have significant energy drain")
	}

	// Moderate fun
	if props.FunFactor < 0.6 {
		t.Error("Ball should be fun")
	}
}

func TestGetGroomingToolProperties(t *testing.T) {
	tests := []struct {
		name              string
		toolType          GroomingToolType
		wantName          string
		expectHighClean   bool
		expectHighStress  bool
		expectHighSkill   bool
	}{
		{"Brush", GroomingToolBrush, "Brush", true, false, false},
		{"Shampoo", GroomingToolShampoo, "Shampoo", true, false, false},
		{"Nail Clipper", GroomingToolNailClipper, "Nail Clipper", false, true, true},
		{"Toothbrush", GroomingToolToothbrush, "Toothbrush", false, false, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			props := GetGroomingToolProperties(tt.toolType)

			if props.Name != tt.wantName {
				t.Errorf("Name = %s, want %s", props.Name, tt.wantName)
			}

			if tt.expectHighClean && props.CleanlinessBoost < 0.6 {
				t.Errorf("Expected high cleanliness boost for %s", tt.name)
			}

			if tt.expectHighStress && props.StressImpact < 0.4 {
				t.Errorf("Expected high stress for %s", tt.name)
			}

			if tt.expectHighSkill && props.SkillLevelRequired < 0.5 {
				t.Errorf("Expected high skill requirement for %s", tt.name)
			}

			// All values should be in valid ranges
			if props.CleanlinessBoost < 0 || props.CleanlinessBoost > 1 {
				t.Errorf("CleanlinessBoost %f out of range [0,1]", props.CleanlinessBoost)
			}

			if props.HealthBoost < 0 || props.HealthBoost > 1 {
				t.Errorf("HealthBoost %f out of range [0,1]", props.HealthBoost)
			}

			if props.ComfortLevel < 0 || props.ComfortLevel > 1 {
				t.Errorf("ComfortLevel %f out of range [0,1]", props.ComfortLevel)
			}

			if props.StressImpact < -1 || props.StressImpact > 1 {
				t.Errorf("StressImpact %f out of range [-1,1]", props.StressImpact)
			}

			if props.TimeRequired < 0 || props.TimeRequired > 1 {
				t.Errorf("TimeRequired %f out of range [0,1]", props.TimeRequired)
			}

			if props.SkillLevelRequired < 0 || props.SkillLevelRequired > 1 {
				t.Errorf("SkillLevelRequired %f out of range [0,1]", props.SkillLevelRequired)
			}
		})
	}
}

func TestGroomingToolComfort(t *testing.T) {
	// Brush should be comfortable
	brushProps := GetGroomingToolProperties(GroomingToolBrush)
	if brushProps.ComfortLevel < 0.7 {
		t.Error("Brush should be comfortable")
	}

	if brushProps.StressImpact > 0 {
		t.Error("Brush should reduce stress, not increase it")
	}

	// Nail clipper should be less comfortable
	clipperProps := GetGroomingToolProperties(GroomingToolNailClipper)
	if clipperProps.ComfortLevel > 0.5 {
		t.Error("Nail clipper should be less comfortable")
	}

	if clipperProps.StressImpact < 0.3 {
		t.Error("Nail clipper should be moderately stressful")
	}
}

func TestGetMedicalItemProperties(t *testing.T) {
	tests := []struct {
		name              string
		itemType          MedicalItemType
		wantName          string
		expectPreventive  bool
		expectRequireSkill bool
	}{
		{"Vitamins", MedicalItemVitamins, "Vitamins", true, false},
		{"Antibiotics", MedicalItemAntibiotics, "Antibiotics", false, true},
		{"Vaccine", MedicalItemVaccine, "Vaccine", true, true},
		{"Pain Relief", MedicalItemPainRelief, "Pain Relief", false, false},
		{"Bandage", MedicalItemBandage, "Bandage", false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			props := GetMedicalItemProperties(tt.itemType)

			if props.Name != tt.wantName {
				t.Errorf("Name = %s, want %s", props.Name, tt.wantName)
			}

			if props.Preventive != tt.expectPreventive {
				t.Errorf("Preventive = %v, want %v", props.Preventive, tt.expectPreventive)
			}

			if props.RequiresSkill != tt.expectRequireSkill {
				t.Errorf("RequiresSkill = %v, want %v", props.RequiresSkill, tt.expectRequireSkill)
			}

			// All values should be in valid ranges
			if props.HealthBoost < 0 || props.HealthBoost > 1 {
				t.Errorf("HealthBoost %f out of range [0,1]", props.HealthBoost)
			}

			if props.StressImpact < -1 || props.StressImpact > 1 {
				t.Errorf("StressImpact %f out of range [-1,1]", props.StressImpact)
			}

			if props.EffectDuration < 0 {
				t.Errorf("EffectDuration %f should be positive", props.EffectDuration)
			}
		})
	}
}

func TestMedicalItemHealthEffects(t *testing.T) {
	// Antibiotics should have strong health boost
	antibioticsProps := GetMedicalItemProperties(MedicalItemAntibiotics)
	if antibioticsProps.HealthBoost < 0.5 {
		t.Error("Antibiotics should have strong health boost")
	}

	// Vitamins should have moderate boost
	vitaminsProps := GetMedicalItemProperties(MedicalItemVitamins)
	if vitaminsProps.HealthBoost > 0.3 {
		t.Error("Vitamins should have moderate health boost")
	}

	// Pain relief should reduce stress
	painReliefProps := GetMedicalItemProperties(MedicalItemPainRelief)
	if painReliefProps.StressImpact >= 0 {
		t.Error("Pain relief should reduce stress")
	}
}

func TestMedicalItemDuration(t *testing.T) {
	// Vaccine should have very long duration
	vaccineProps := GetMedicalItemProperties(MedicalItemVaccine)
	if vaccineProps.EffectDuration < 500 {
		t.Error("Vaccine should have long-lasting effects")
	}

	// Pain relief should have short duration
	painReliefProps := GetMedicalItemProperties(MedicalItemPainRelief)
	if painReliefProps.EffectDuration > 12 {
		t.Error("Pain relief should have short-term effects")
	}
}

func TestSkillTypeString(t *testing.T) {
	tests := []struct {
		skill SkillType
		want  string
	}{
		{SkillObedience, "Obedience"},
		{SkillAgility, "Agility"},
		{SkillTricks, "Tricks"},
		{SkillSocialization, "Socialization"},
		{SkillPottyTraining, "Potty Training"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := tt.skill.String()
			if got != tt.want {
				t.Errorf("String() = %s, want %s", got, tt.want)
			}
		})
	}
}

func TestAllFoodTypes(t *testing.T) {
	// Test that all food types have valid properties
	foodTypes := []FoodType{
		FoodTypeBasicKibble,
		FoodTypePremiumKibble,
		FoodTypeTreat,
		FoodTypeFruit,
		FoodTypeVegetable,
		FoodTypeMeat,
		FoodTypeDairy,
		FoodTypeHealthySnack,
		FoodTypeJunkFood,
		FoodTypeSpecialMeal,
	}

	for _, ft := range foodTypes {
		props := GetFoodProperties(ft)
		if props.Name == "" {
			t.Errorf("Food type %v has empty name", ft)
		}
	}
}

func TestAllToyTypes(t *testing.T) {
	// Test that all toy types have valid properties
	toyTypes := []ToyType{
		ToyTypeBall,
		ToyTypeRope,
		ToyTypePuzzle,
		ToyTypeSqueaky,
		ToyTypeInteractive,
		ToyTypeChewToy,
		ToyTypePlush,
		ToyTypeFeather,
		ToyTypeLaser,
		ToyTypeAgility,
	}

	for _, tt := range toyTypes {
		props := GetToyProperties(tt)
		if props.Name == "" {
			t.Errorf("Toy type %v has empty name", tt)
		}
	}
}

func TestAllGroomingTools(t *testing.T) {
	// Test that all grooming tools have valid properties
	toolTypes := []GroomingToolType{
		GroomingToolBrush,
		GroomingToolComb,
		GroomingToolShampoo,
		GroomingToolNailClipper,
		GroomingToolToothbrush,
		GroomingToolEarCleaner,
	}

	for _, gt := range toolTypes {
		props := GetGroomingToolProperties(gt)
		if props.Name == "" {
			t.Errorf("Grooming tool %v has empty name", gt)
		}
	}
}

func TestAllMedicalItems(t *testing.T) {
	// Test that all medical items have valid properties
	itemTypes := []MedicalItemType{
		MedicalItemVitamins,
		MedicalItemAntibiotics,
		MedicalItemPainRelief,
		MedicalItemBandage,
		MedicalItemVaccine,
		MedicalItemSupplements,
	}

	for _, mt := range itemTypes {
		props := GetMedicalItemProperties(mt)
		if props.Name == "" {
			t.Errorf("Medical item %v has empty name", mt)
		}
	}
}
