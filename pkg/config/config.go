package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"gopkg.in/yaml.v3"
)

// AppConfig is the root configuration structure for the entire application
type AppConfig struct {
	App         AppSettings         `yaml:"app"`
	Log         LogSettings         `yaml:"log"`
	Game        GameSettings        `yaml:"game"`
	Data        DataSettings        `yaml:"data"`
	Environment EnvironmentSettings `yaml:"environment"`
	UI          UISettings          `yaml:"ui"`
}

// AppSettings holds application-level configuration
type AppSettings struct {
	Name    string `yaml:"name"`
	Version string `yaml:"version"`
	Debug   bool   `yaml:"debug"`
}

// LogSettings holds logging configuration
type LogSettings struct {
	Level      string `yaml:"level"`       // debug, info, warn, error
	Format     string `yaml:"format"`      // text, json
	OutputPath string `yaml:"output_path"` // empty for stdout only
	AddSource  bool   `yaml:"add_source"`  // add source file/line to logs
}

// GameSettings holds game loop and simulation configuration
type GameSettings struct {
	TargetFPS          int           `yaml:"target_fps"`
	AutoSaveInterval   time.Duration `yaml:"auto_save_interval"`
	AutoBackupInterval time.Duration `yaml:"auto_backup_interval"`
	EnableAutoSave     bool          `yaml:"enable_auto_save"`
	EnableAutoBackup   bool          `yaml:"enable_auto_backup"`
}

// DataSettings holds data persistence and caching configuration
type DataSettings struct {
	BasePath           string        `yaml:"base_path"`
	BackupPath         string        `yaml:"backup_path"`
	CacheSizeMB        int           `yaml:"cache_size_mb"`
	CacheTTLMinutes    int           `yaml:"cache_ttl_minutes"`
	EnableEncryption   bool          `yaml:"enable_encryption"`
	EncryptionKey      string        `yaml:"encryption_key"`
	MaxBackups         int           `yaml:"max_backups"`
	EnableAutoSave     bool          `yaml:"enable_auto_save"`
	AutoSaveInterval   time.Duration `yaml:"auto_save_interval"`
	EnableAutoBackup   bool          `yaml:"enable_auto_backup"`
	AutoBackupInterval time.Duration `yaml:"auto_backup_interval"`
}

// EnvironmentSettings holds environment system configuration
type EnvironmentSettings struct {
	StartSeason           string  `yaml:"start_season"`
	SeasonLengthHours     float64 `yaml:"season_length_hours"`
	SeasonalIntensity     float64 `yaml:"seasonal_intensity"`
	EnableTransitions     bool    `yaml:"enable_transitions"`
	WeatherVolatility     float64 `yaml:"weather_volatility"`
	WeatherChangeInterval float64 `yaml:"weather_change_interval"`
	StartingBiome         string  `yaml:"starting_biome"`
}

// UISettings holds user interface configuration
type UISettings struct {
	Width       int  `yaml:"width"`
	Height      int  `yaml:"height"`
	ShowFPS     bool `yaml:"show_fps"`
	ColorScheme string `yaml:"color_scheme"`
}

// DefaultConfig returns the default application configuration
func DefaultConfig() *AppConfig {
	return &AppConfig{
		App: AppSettings{
			Name:    "Gochi",
			Version: "0.1.0-alpha",
			Debug:   false,
		},
		Log: LogSettings{
			Level:      "info",
			Format:     "text",
			OutputPath: "",
			AddSource:  false,
		},
		Game: GameSettings{
			TargetFPS:          60,
			AutoSaveInterval:   5 * time.Minute,
			AutoBackupInterval: 24 * time.Hour,
			EnableAutoSave:     true,
			EnableAutoBackup:   true,
		},
		Data: DataSettings{
			BasePath:           "./data/pets",
			BackupPath:         "./data/backups",
			CacheSizeMB:        100,
			CacheTTLMinutes:    30,
			EnableEncryption:   false,
			EncryptionKey:      "",
			MaxBackups:         10,
			EnableAutoSave:     true,
			AutoSaveInterval:   5 * time.Minute,
			EnableAutoBackup:   true,
			AutoBackupInterval: 24 * time.Hour,
		},
		Environment: EnvironmentSettings{
			StartSeason:           "spring",
			SeasonLengthHours:     168.0,
			SeasonalIntensity:     0.8,
			EnableTransitions:     true,
			WeatherVolatility:     0.5,
			WeatherChangeInterval: 4.0,
			StartingBiome:         "grassland",
		},
		UI: UISettings{
			Width:       80,
			Height:      24,
			ShowFPS:     false,
			ColorScheme: "default",
		},
	}
}

// LoadFromFile loads configuration from a YAML file
func LoadFromFile(path string) (*AppConfig, error) {
	if path == "" {
		return nil, fmt.Errorf("config file path cannot be empty")
	}

	// Check if file exists
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("config file not found: %s", path)
		}
		return nil, fmt.Errorf("failed to access config file: %w", err)
	}

	// Read file
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// Parse YAML
	config := DefaultConfig()
	if err := yaml.Unmarshal(data, config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Apply environment variable overrides
	if err := applyEnvOverrides(config); err != nil {
		return nil, fmt.Errorf("failed to apply environment overrides: %w", err)
	}

	// Validate configuration
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return config, nil
}

// SaveToFile saves configuration to a YAML file
func SaveToFile(config *AppConfig, path string) error {
	if config == nil {
		return fmt.Errorf("config cannot be nil")
	}
	if path == "" {
		return fmt.Errorf("config file path cannot be empty")
	}

	// Validate before saving
	if err := config.Validate(); err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}

	// Marshal to YAML
	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Write file
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// Validate validates the configuration values
func (c *AppConfig) Validate() error {
	// Validate App settings
	if c.App.Name == "" {
		return fmt.Errorf("app.name cannot be empty")
	}
	if c.App.Version == "" {
		return fmt.Errorf("app.version cannot be empty")
	}

	// Validate Log settings
	validLevels := map[string]bool{"debug": true, "info": true, "warn": true, "error": true}
	if !validLevels[c.Log.Level] {
		return fmt.Errorf("log.level must be debug, info, warn, or error, got %s", c.Log.Level)
	}
	validFormats := map[string]bool{"text": true, "json": true}
	if !validFormats[c.Log.Format] {
		return fmt.Errorf("log.format must be text or json, got %s", c.Log.Format)
	}

	// Validate Game settings
	if c.Game.TargetFPS < 1 || c.Game.TargetFPS > 300 {
		return fmt.Errorf("game.target_fps must be between 1 and 300, got %d", c.Game.TargetFPS)
	}
	if c.Game.AutoSaveInterval < 0 {
		return fmt.Errorf("game.auto_save_interval cannot be negative")
	}
	if c.Game.AutoBackupInterval < 0 {
		return fmt.Errorf("game.auto_backup_interval cannot be negative")
	}

	// Validate Data settings
	if c.Data.BasePath == "" {
		return fmt.Errorf("data.base_path cannot be empty")
	}
	if c.Data.BackupPath == "" {
		return fmt.Errorf("data.backup_path cannot be empty")
	}
	if c.Data.CacheSizeMB < 0 {
		return fmt.Errorf("data.cache_size_mb cannot be negative, got %d", c.Data.CacheSizeMB)
	}
	if c.Data.CacheTTLMinutes < 0 {
		return fmt.Errorf("data.cache_ttl_minutes cannot be negative, got %d", c.Data.CacheTTLMinutes)
	}
	if c.Data.MaxBackups < 0 {
		return fmt.Errorf("data.max_backups cannot be negative, got %d", c.Data.MaxBackups)
	}
	if c.Data.EnableEncryption && c.Data.EncryptionKey == "" {
		return fmt.Errorf("data.encryption_key required when encryption is enabled")
	}

	// Validate Environment settings
	validSeasons := map[string]bool{"spring": true, "summer": true, "autumn": true, "winter": true}
	if !validSeasons[c.Environment.StartSeason] {
		return fmt.Errorf("environment.start_season must be spring, summer, autumn, or winter, got %s", c.Environment.StartSeason)
	}
	if c.Environment.SeasonLengthHours <= 0 {
		return fmt.Errorf("environment.season_length_hours must be positive, got %f", c.Environment.SeasonLengthHours)
	}
	if c.Environment.SeasonalIntensity < 0 || c.Environment.SeasonalIntensity > 1 {
		return fmt.Errorf("environment.seasonal_intensity must be between 0 and 1, got %f", c.Environment.SeasonalIntensity)
	}
	if c.Environment.WeatherVolatility < 0 || c.Environment.WeatherVolatility > 1 {
		return fmt.Errorf("environment.weather_volatility must be between 0 and 1, got %f", c.Environment.WeatherVolatility)
	}
	if c.Environment.WeatherChangeInterval <= 0 {
		return fmt.Errorf("environment.weather_change_interval must be positive, got %f", c.Environment.WeatherChangeInterval)
	}

	validBiomes := map[string]bool{
		"grassland": true, "forest": true, "desert": true,
		"tundra": true, "mountain": true, "ocean": true,
	}
	if !validBiomes[c.Environment.StartingBiome] {
		return fmt.Errorf("environment.starting_biome must be a valid biome type, got %s", c.Environment.StartingBiome)
	}

	// Validate UI settings
	if c.UI.Width < 40 || c.UI.Width > 500 {
		return fmt.Errorf("ui.width must be between 40 and 500, got %d", c.UI.Width)
	}
	if c.UI.Height < 10 || c.UI.Height > 200 {
		return fmt.Errorf("ui.height must be between 10 and 200, got %d", c.UI.Height)
	}

	return nil
}

// applyEnvOverrides applies environment variable overrides to the configuration
func applyEnvOverrides(config *AppConfig) error {
	// App overrides
	if val := os.Getenv("GOCHI_DEBUG"); val != "" {
		debug, err := strconv.ParseBool(val)
		if err != nil {
			return fmt.Errorf("invalid GOCHI_DEBUG value: %w", err)
		}
		config.App.Debug = debug
	}

	// Log overrides
	if val := os.Getenv("GOCHI_LOG_LEVEL"); val != "" {
		config.Log.Level = val
	}
	if val := os.Getenv("GOCHI_LOG_FORMAT"); val != "" {
		config.Log.Format = val
	}
	if val := os.Getenv("GOCHI_LOG_OUTPUT"); val != "" {
		config.Log.OutputPath = val
	}
	if val := os.Getenv("GOCHI_LOG_SOURCE"); val != "" {
		addSource, err := strconv.ParseBool(val)
		if err != nil {
			return fmt.Errorf("invalid GOCHI_LOG_SOURCE value: %w", err)
		}
		config.Log.AddSource = addSource
	}

	// Game overrides
	if val := os.Getenv("GOCHI_TARGET_FPS"); val != "" {
		fps, err := strconv.Atoi(val)
		if err != nil {
			return fmt.Errorf("invalid GOCHI_TARGET_FPS value: %w", err)
		}
		config.Game.TargetFPS = fps
	}
	if val := os.Getenv("GOCHI_AUTO_SAVE"); val != "" {
		enable, err := strconv.ParseBool(val)
		if err != nil {
			return fmt.Errorf("invalid GOCHI_AUTO_SAVE value: %w", err)
		}
		config.Game.EnableAutoSave = enable
	}

	// Data overrides
	if val := os.Getenv("GOCHI_DATA_PATH"); val != "" {
		config.Data.BasePath = val
	}
	if val := os.Getenv("GOCHI_BACKUP_PATH"); val != "" {
		config.Data.BackupPath = val
	}
	if val := os.Getenv("GOCHI_ENCRYPTION_KEY"); val != "" {
		config.Data.EncryptionKey = val
		config.Data.EnableEncryption = true
	}
	if val := os.Getenv("GOCHI_CACHE_SIZE_MB"); val != "" {
		size, err := strconv.Atoi(val)
		if err != nil {
			return fmt.Errorf("invalid GOCHI_CACHE_SIZE_MB value: %w", err)
		}
		config.Data.CacheSizeMB = size
	}

	// UI overrides
	if val := os.Getenv("GOCHI_UI_WIDTH"); val != "" {
		width, err := strconv.Atoi(val)
		if err != nil {
			return fmt.Errorf("invalid GOCHI_UI_WIDTH value: %w", err)
		}
		config.UI.Width = width
	}
	if val := os.Getenv("GOCHI_UI_HEIGHT"); val != "" {
		height, err := strconv.Atoi(val)
		if err != nil {
			return fmt.Errorf("invalid GOCHI_UI_HEIGHT value: %w", err)
		}
		config.UI.Height = height
	}
	if val := os.Getenv("GOCHI_SHOW_FPS"); val != "" {
		show, err := strconv.ParseBool(val)
		if err != nil {
			return fmt.Errorf("invalid GOCHI_SHOW_FPS value: %w", err)
		}
		config.UI.ShowFPS = show
	}

	return nil
}
