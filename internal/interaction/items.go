package interaction

// ItemType represents categories of items
type ItemType int

const (
	ItemTypeFood ItemType = iota
	ItemTypeToy
	ItemTypeGroomingTool
	ItemTypeMedicalItem
	ItemTypeTrainingTool
)

// FoodType represents different types of food with nutritional properties
type FoodType int

const (
	FoodTypeBasicKibble FoodType = iota
	FoodTypePremiumKibble
	FoodTypeTreat
	FoodTypeFruit
	FoodTypeVegetable
	FoodTypeMeat
	FoodTypeDairy
	FoodTypeHealthySnack
	FoodTypeJunkFood
	FoodTypeSpecialMeal
)

// FoodProperties defines the nutritional and hedonic properties of food
type FoodProperties struct {
	Name           string
	NutritionValue float64 // How much it satisfies hunger (0-1)
	HydrationValue float64 // How much it satisfies thirst (0-1)
	EnergyBoost    float64 // Immediate energy gain (0-1)
	HealthImpact   float64 // Long-term health effect (-1 to 1)
	HappinessBoost float64 // How much joy it brings (0-1)
	FillAmount     float64 // How filling it is (0-1, affects how quickly hunger returns)
}

// GetFoodProperties returns the properties for a given food type
func GetFoodProperties(foodType FoodType) FoodProperties {
	foodMap := map[FoodType]FoodProperties{
		FoodTypeBasicKibble: {
			Name:           "Basic Kibble",
			NutritionValue: 0.6,
			HydrationValue: 0.1,
			EnergyBoost:    0.3,
			HealthImpact:   0.0,
			HappinessBoost: 0.2,
			FillAmount:     0.7,
		},
		FoodTypePremiumKibble: {
			Name:           "Premium Kibble",
			NutritionValue: 0.8,
			HydrationValue: 0.2,
			EnergyBoost:    0.5,
			HealthImpact:   0.2,
			HappinessBoost: 0.4,
			FillAmount:     0.8,
		},
		FoodTypeTreat: {
			Name:           "Treat",
			NutritionValue: 0.2,
			HydrationValue: 0.0,
			EnergyBoost:    0.2,
			HealthImpact:   -0.1,
			HappinessBoost: 0.8,
			FillAmount:     0.2,
		},
		FoodTypeFruit: {
			Name:           "Fruit",
			NutritionValue: 0.5,
			HydrationValue: 0.6,
			EnergyBoost:    0.4,
			HealthImpact:   0.3,
			HappinessBoost: 0.5,
			FillAmount:     0.4,
		},
		FoodTypeVegetable: {
			Name:           "Vegetable",
			NutritionValue: 0.6,
			HydrationValue: 0.4,
			EnergyBoost:    0.2,
			HealthImpact:   0.4,
			HappinessBoost: 0.3,
			FillAmount:     0.5,
		},
		FoodTypeMeat: {
			Name:           "Meat",
			NutritionValue: 0.9,
			HydrationValue: 0.1,
			EnergyBoost:    0.7,
			HealthImpact:   0.1,
			HappinessBoost: 0.7,
			FillAmount:     0.9,
		},
		FoodTypeDairy: {
			Name:           "Dairy",
			NutritionValue: 0.5,
			HydrationValue: 0.7,
			EnergyBoost:    0.3,
			HealthImpact:   0.1,
			HappinessBoost: 0.4,
			FillAmount:     0.5,
		},
		FoodTypeHealthySnack: {
			Name:           "Healthy Snack",
			NutritionValue: 0.4,
			HydrationValue: 0.2,
			EnergyBoost:    0.3,
			HealthImpact:   0.2,
			HappinessBoost: 0.5,
			FillAmount:     0.3,
		},
		FoodTypeJunkFood: {
			Name:           "Junk Food",
			NutritionValue: 0.3,
			HydrationValue: 0.0,
			EnergyBoost:    0.6,
			HealthImpact:   -0.3,
			HappinessBoost: 0.9,
			FillAmount:     0.5,
		},
		FoodTypeSpecialMeal: {
			Name:           "Special Meal",
			NutritionValue: 1.0,
			HydrationValue: 0.5,
			EnergyBoost:    0.8,
			HealthImpact:   0.5,
			HappinessBoost: 1.0,
			FillAmount:     1.0,
		},
	}
	return foodMap[foodType]
}

// ToyType represents different types of toys with different play properties
type ToyType int

const (
	ToyTypeBall ToyType = iota
	ToyTypeRope
	ToyTypePuzzle
	ToyTypeSqueaky
	ToyTypeInteractive
	ToyTypeChewToy
	ToyTypePlush
	ToyTypeFeather
	ToyTypeLaser
	ToyTypeAgility
)

// ToyProperties defines the play characteristics of a toy
type ToyProperties struct {
	Name               string
	EnergyDrain        float64 // How much energy playing drains (0-1)
	MentalStimulation  float64 // How mentally engaging (0-1)
	PhysicalExercise   float64 // How much physical activity (0-1)
	SocialEngagement   float64 // How much it encourages social play (0-1)
	FunFactor          float64 // Base happiness boost (0-1)
	SkillDevelopment   float64 // How much it helps develop skills (0-1)
	DurabilityRequired float64 // How sturdy the pet needs to be (0-1)
}

// GetToyProperties returns the properties for a given toy type
func GetToyProperties(toyType ToyType) ToyProperties {
	toyMap := map[ToyType]ToyProperties{
		ToyTypeBall: {
			Name:               "Ball",
			EnergyDrain:        0.7,
			MentalStimulation:  0.3,
			PhysicalExercise:   0.9,
			SocialEngagement:   0.5,
			FunFactor:          0.7,
			SkillDevelopment:   0.4,
			DurabilityRequired: 0.5,
		},
		ToyTypeRope: {
			Name:               "Rope",
			EnergyDrain:        0.6,
			MentalStimulation:  0.3,
			PhysicalExercise:   0.7,
			SocialEngagement:   0.8,
			FunFactor:          0.6,
			SkillDevelopment:   0.3,
			DurabilityRequired: 0.6,
		},
		ToyTypePuzzle: {
			Name:               "Puzzle",
			EnergyDrain:        0.3,
			MentalStimulation:  0.9,
			PhysicalExercise:   0.2,
			SocialEngagement:   0.2,
			FunFactor:          0.5,
			SkillDevelopment:   0.9,
			DurabilityRequired: 0.3,
		},
		ToyTypeSqueaky: {
			Name:               "Squeaky Toy",
			EnergyDrain:        0.5,
			MentalStimulation:  0.4,
			PhysicalExercise:   0.5,
			SocialEngagement:   0.4,
			FunFactor:          0.8,
			SkillDevelopment:   0.3,
			DurabilityRequired: 0.4,
		},
		ToyTypeInteractive: {
			Name:               "Interactive Toy",
			EnergyDrain:        0.6,
			MentalStimulation:  0.8,
			PhysicalExercise:   0.6,
			SocialEngagement:   0.9,
			FunFactor:          0.9,
			SkillDevelopment:   0.7,
			DurabilityRequired: 0.5,
		},
		ToyTypeChewToy: {
			Name:               "Chew Toy",
			EnergyDrain:        0.2,
			MentalStimulation:  0.2,
			PhysicalExercise:   0.2,
			SocialEngagement:   0.1,
			FunFactor:          0.6,
			SkillDevelopment:   0.2,
			DurabilityRequired: 0.8,
		},
		ToyTypePlush: {
			Name:               "Plush Toy",
			EnergyDrain:        0.4,
			MentalStimulation:  0.3,
			PhysicalExercise:   0.4,
			SocialEngagement:   0.3,
			FunFactor:          0.7,
			SkillDevelopment:   0.2,
			DurabilityRequired: 0.3,
		},
		ToyTypeFeather: {
			Name:               "Feather Toy",
			EnergyDrain:        0.8,
			MentalStimulation:  0.6,
			PhysicalExercise:   0.8,
			SocialEngagement:   0.5,
			FunFactor:          0.9,
			SkillDevelopment:   0.5,
			DurabilityRequired: 0.4,
		},
		ToyTypeLaser: {
			Name:               "Laser Pointer",
			EnergyDrain:        0.9,
			MentalStimulation:  0.5,
			PhysicalExercise:   0.9,
			SocialEngagement:   0.4,
			FunFactor:          0.8,
			SkillDevelopment:   0.4,
			DurabilityRequired: 0.2,
		},
		ToyTypeAgility: {
			Name:               "Agility Equipment",
			EnergyDrain:        0.8,
			MentalStimulation:  0.7,
			PhysicalExercise:   0.9,
			SocialEngagement:   0.6,
			FunFactor:          0.7,
			SkillDevelopment:   0.8,
			DurabilityRequired: 0.7,
		},
	}
	return toyMap[toyType]
}

// GroomingToolType represents different grooming tools
type GroomingToolType int

const (
	GroomingToolBrush GroomingToolType = iota
	GroomingToolComb
	GroomingToolShampoo
	GroomingToolNailClipper
	GroomingToolToothbrush
	GroomingToolEarCleaner
)

// GroomingToolProperties defines grooming tool characteristics
type GroomingToolProperties struct {
	Name               string
	CleanlinessBoost   float64 // How much it improves cleanliness (0-1)
	HealthBoost        float64 // Health benefit (0-1)
	ComfortLevel       float64 // How comfortable/pleasant it is (0-1)
	StressImpact       float64 // Stress caused (-1 to 1)
	TimeRequired       float64 // How long it takes (0-1)
	SkillLevelRequired float64 // User skill needed (0-1)
}

// GetGroomingToolProperties returns properties for a grooming tool
func GetGroomingToolProperties(toolType GroomingToolType) GroomingToolProperties {
	toolMap := map[GroomingToolType]GroomingToolProperties{
		GroomingToolBrush: {
			Name:               "Brush",
			CleanlinessBoost:   0.6,
			HealthBoost:        0.2,
			ComfortLevel:       0.8,
			StressImpact:       -0.1,
			TimeRequired:       0.5,
			SkillLevelRequired: 0.2,
		},
		GroomingToolComb: {
			Name:               "Comb",
			CleanlinessBoost:   0.5,
			HealthBoost:        0.1,
			ComfortLevel:       0.6,
			StressImpact:       0.1,
			TimeRequired:       0.6,
			SkillLevelRequired: 0.3,
		},
		GroomingToolShampoo: {
			Name:               "Shampoo",
			CleanlinessBoost:   0.9,
			HealthBoost:        0.3,
			ComfortLevel:       0.5,
			StressImpact:       0.3,
			TimeRequired:       0.8,
			SkillLevelRequired: 0.4,
		},
		GroomingToolNailClipper: {
			Name:               "Nail Clipper",
			CleanlinessBoost:   0.3,
			HealthBoost:        0.4,
			ComfortLevel:       0.3,
			StressImpact:       0.5,
			TimeRequired:       0.4,
			SkillLevelRequired: 0.7,
		},
		GroomingToolToothbrush: {
			Name:               "Toothbrush",
			CleanlinessBoost:   0.4,
			HealthBoost:        0.5,
			ComfortLevel:       0.4,
			StressImpact:       0.4,
			TimeRequired:       0.3,
			SkillLevelRequired: 0.5,
		},
		GroomingToolEarCleaner: {
			Name:               "Ear Cleaner",
			CleanlinessBoost:   0.5,
			HealthBoost:        0.6,
			ComfortLevel:       0.3,
			StressImpact:       0.6,
			TimeRequired:       0.3,
			SkillLevelRequired: 0.6,
		},
	}
	return toolMap[toolType]
}

// MedicalItemType represents different medical items
type MedicalItemType int

const (
	MedicalItemVitamins MedicalItemType = iota
	MedicalItemAntibiotics
	MedicalItemPainRelief
	MedicalItemBandage
	MedicalItemVaccine
	MedicalItemSupplements
)

// MedicalItemProperties defines medical item characteristics
type MedicalItemProperties struct {
	Name          string
	HealthBoost   float64 // Immediate health improvement (0-1)
	StressImpact  float64 // Stress from treatment (-1 to 1)
	EffectDuration float64 // How long effects last in game hours
	Preventive    bool    // Whether it prevents future illness
	RequiresSkill bool    // Whether it requires training to use
}

// GetMedicalItemProperties returns properties for a medical item
func GetMedicalItemProperties(itemType MedicalItemType) MedicalItemProperties {
	itemMap := map[MedicalItemType]MedicalItemProperties{
		MedicalItemVitamins: {
			Name:          "Vitamins",
			HealthBoost:   0.2,
			StressImpact:  0.0,
			EffectDuration: 24.0,
			Preventive:    true,
			RequiresSkill: false,
		},
		MedicalItemAntibiotics: {
			Name:          "Antibiotics",
			HealthBoost:   0.6,
			StressImpact:  0.2,
			EffectDuration: 48.0,
			Preventive:    false,
			RequiresSkill: true,
		},
		MedicalItemPainRelief: {
			Name:          "Pain Relief",
			HealthBoost:   0.3,
			StressImpact:  -0.3,
			EffectDuration: 8.0,
			Preventive:    false,
			RequiresSkill: false,
		},
		MedicalItemBandage: {
			Name:          "Bandage",
			HealthBoost:   0.4,
			StressImpact:  0.1,
			EffectDuration: 12.0,
			Preventive:    false,
			RequiresSkill: false,
		},
		MedicalItemVaccine: {
			Name:          "Vaccine",
			HealthBoost:   0.1,
			StressImpact:  0.4,
			EffectDuration: 720.0, // 30 days
			Preventive:    true,
			RequiresSkill: true,
		},
		MedicalItemSupplements: {
			Name:          "Supplements",
			HealthBoost:   0.3,
			StressImpact:  0.0,
			EffectDuration: 24.0,
			Preventive:    true,
			RequiresSkill: false,
		},
	}
	return itemMap[itemType]
}

// TrainingToolType represents different training tools
type TrainingToolType int

const (
	TrainingToolClicker TrainingToolType = iota
	TrainingToolTargetStick
	TrainingToolTreats
	TrainingToolObstacles
	TrainingToolManual
)

// SkillType represents skills that can be trained
type SkillType int

const (
	SkillObedience SkillType = iota
	SkillAgility
	SkillTricks
	SkillSocialization
	SkillPottyTraining
	SkillQuietness
	SkillFocus
	SkillRecall
	SkillLeashWalking
	SkillBoundaryRespect
)

// String returns the string representation of SkillType
func (st SkillType) String() string {
	return [...]string{
		"Obedience", "Agility", "Tricks", "Socialization", "Potty Training",
		"Quietness", "Focus", "Recall", "Leash Walking", "Boundary Respect",
	}[st]
}

// Skill represents a trainable skill with progression
type Skill struct {
	Type       SkillType
	Level      float64 // 0-1, current mastery level
	Experience float64 // Accumulated experience points
	LastTrained int64  // Timestamp of last training
	Decay      float64 // How quickly skill degrades without practice
}

// TrainingSession represents the results of a training interaction
type TrainingSession struct {
	Skill           SkillType
	Success         bool
	ExperienceGain  float64
	LevelChange     float64
	EffortRequired  float64 // How much effort from user
	PetEngagement   float64 // How engaged the pet was
	RecommendedNext string  // Suggestion for next training
}
