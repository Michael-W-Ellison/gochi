package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	if config == nil {
		t.Fatal("DefaultConfig returned nil")
	}

	// Validate App settings
	if config.App.Name != "Gochi" {
		t.Errorf("Expected app name 'Gochi', got %s", config.App.Name)
	}
	if config.App.Version == "" {
		t.Error("App version should not be empty")
	}

	// Validate Game settings
	if config.Game.TargetFPS != 60 {
		t.Errorf("Expected target FPS 60, got %d", config.Game.TargetFPS)
	}
	if config.Game.AutoSaveInterval != 5*time.Minute {
		t.Errorf("Expected auto save interval 5m, got %v", config.Game.AutoSaveInterval)
	}

	// Validate Data settings
	if config.Data.BasePath == "" {
		t.Error("Data base path should not be empty")
	}
	if config.Data.CacheSizeMB != 100 {
		t.Errorf("Expected cache size 100MB, got %d", config.Data.CacheSizeMB)
	}

	// Validate Environment settings
	if config.Environment.StartSeason != "spring" {
		t.Errorf("Expected start season 'spring', got %s", config.Environment.StartSeason)
	}

	// Validate UI settings
	if config.UI.Width != 80 {
		t.Errorf("Expected UI width 80, got %d", config.UI.Width)
	}
}

func TestConfigValidation(t *testing.T) {
	tests := []struct {
		name      string
		mutate    func(*AppConfig)
		shouldErr bool
		errMsg    string
	}{
		{
			name:      "valid config",
			mutate:    func(c *AppConfig) {},
			shouldErr: false,
		},
		{
			name: "empty app name",
			mutate: func(c *AppConfig) {
				c.App.Name = ""
			},
			shouldErr: true,
			errMsg:    "app.name cannot be empty",
		},
		{
			name: "empty app version",
			mutate: func(c *AppConfig) {
				c.App.Version = ""
			},
			shouldErr: true,
			errMsg:    "app.version cannot be empty",
		},
		{
			name: "invalid FPS too low",
			mutate: func(c *AppConfig) {
				c.Game.TargetFPS = 0
			},
			shouldErr: true,
			errMsg:    "game.target_fps must be between 1 and 300",
		},
		{
			name: "invalid FPS too high",
			mutate: func(c *AppConfig) {
				c.Game.TargetFPS = 400
			},
			shouldErr: true,
			errMsg:    "game.target_fps must be between 1 and 300",
		},
		{
			name: "negative auto save interval",
			mutate: func(c *AppConfig) {
				c.Game.AutoSaveInterval = -1 * time.Second
			},
			shouldErr: true,
			errMsg:    "game.auto_save_interval cannot be negative",
		},
		{
			name: "empty data base path",
			mutate: func(c *AppConfig) {
				c.Data.BasePath = ""
			},
			shouldErr: true,
			errMsg:    "data.base_path cannot be empty",
		},
		{
			name: "empty backup path",
			mutate: func(c *AppConfig) {
				c.Data.BackupPath = ""
			},
			shouldErr: true,
			errMsg:    "data.backup_path cannot be empty",
		},
		{
			name: "negative cache size",
			mutate: func(c *AppConfig) {
				c.Data.CacheSizeMB = -10
			},
			shouldErr: true,
			errMsg:    "data.cache_size_mb cannot be negative",
		},
		{
			name: "encryption enabled without key",
			mutate: func(c *AppConfig) {
				c.Data.EnableEncryption = true
				c.Data.EncryptionKey = ""
			},
			shouldErr: true,
			errMsg:    "data.encryption_key required when encryption is enabled",
		},
		{
			name: "invalid season",
			mutate: func(c *AppConfig) {
				c.Environment.StartSeason = "invalid"
			},
			shouldErr: true,
			errMsg:    "environment.start_season must be spring, summer, autumn, or winter",
		},
		{
			name: "negative season length",
			mutate: func(c *AppConfig) {
				c.Environment.SeasonLengthHours = -10
			},
			shouldErr: true,
			errMsg:    "environment.season_length_hours must be positive",
		},
		{
			name: "seasonal intensity too high",
			mutate: func(c *AppConfig) {
				c.Environment.SeasonalIntensity = 1.5
			},
			shouldErr: true,
			errMsg:    "environment.seasonal_intensity must be between 0 and 1",
		},
		{
			name: "weather volatility too low",
			mutate: func(c *AppConfig) {
				c.Environment.WeatherVolatility = -0.1
			},
			shouldErr: true,
			errMsg:    "environment.weather_volatility must be between 0 and 1",
		},
		{
			name: "invalid biome",
			mutate: func(c *AppConfig) {
				c.Environment.StartingBiome = "invalid"
			},
			shouldErr: true,
			errMsg:    "environment.starting_biome must be a valid biome type",
		},
		{
			name: "UI width too small",
			mutate: func(c *AppConfig) {
				c.UI.Width = 20
			},
			shouldErr: true,
			errMsg:    "ui.width must be between 40 and 500",
		},
		{
			name: "UI height too large",
			mutate: func(c *AppConfig) {
				c.UI.Height = 300
			},
			shouldErr: true,
			errMsg:    "ui.height must be between 10 and 200",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := DefaultConfig()
			tt.mutate(config)

			err := config.Validate()
			if tt.shouldErr {
				if err == nil {
					t.Errorf("Expected validation error, got nil")
				} else if tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("Expected error message to contain '%s', got '%s'", tt.errMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected validation error: %v", err)
				}
			}
		})
	}
}

func TestLoadFromFile(t *testing.T) {
	// Create temporary directory
	tmpDir := t.TempDir()

	t.Run("load valid config", func(t *testing.T) {
		configPath := filepath.Join(tmpDir, "valid.yaml")
		yamlContent := `app:
  name: TestApp
  version: 1.0.0
  debug: true
  log_path: ./test-logs
game:
  target_fps: 30
  auto_save_interval: 10m
  auto_backup_interval: 1h
  enable_auto_save: true
  enable_auto_backup: false
data:
  base_path: ./test-data
  backup_path: ./test-backups
  cache_size_mb: 50
  cache_ttl_minutes: 15
  enable_encryption: false
  encryption_key: ""
  max_backups: 5
  enable_auto_save: true
  auto_save_interval: 10m
  enable_auto_backup: false
  auto_backup_interval: 1h
environment:
  start_season: summer
  season_length_hours: 100.0
  seasonal_intensity: 0.9
  enable_transitions: false
  weather_volatility: 0.3
  weather_change_interval: 2.0
  starting_biome: forest
ui:
  width: 100
  height: 30
  show_fps: true
  color_scheme: dark
`
		if err := os.WriteFile(configPath, []byte(yamlContent), 0644); err != nil {
			t.Fatalf("Failed to write test config: %v", err)
		}

		config, err := LoadFromFile(configPath)
		if err != nil {
			t.Fatalf("Failed to load config: %v", err)
		}

		// Verify loaded values
		if config.App.Name != "TestApp" {
			t.Errorf("Expected app name 'TestApp', got %s", config.App.Name)
		}
		if config.Game.TargetFPS != 30 {
			t.Errorf("Expected FPS 30, got %d", config.Game.TargetFPS)
		}
		if !config.App.Debug {
			t.Error("Expected debug to be true")
		}
		if config.Environment.StartSeason != "summer" {
			t.Errorf("Expected season 'summer', got %s", config.Environment.StartSeason)
		}
	})

	t.Run("file not found", func(t *testing.T) {
		_, err := LoadFromFile(filepath.Join(tmpDir, "nonexistent.yaml"))
		if err == nil {
			t.Error("Expected error for nonexistent file")
		}
	})

	t.Run("empty path", func(t *testing.T) {
		_, err := LoadFromFile("")
		if err == nil {
			t.Error("Expected error for empty path")
		}
	})

	t.Run("invalid YAML", func(t *testing.T) {
		configPath := filepath.Join(tmpDir, "invalid.yaml")
		if err := os.WriteFile(configPath, []byte("invalid: [yaml content"), 0644); err != nil {
			t.Fatalf("Failed to write test config: %v", err)
		}

		_, err := LoadFromFile(configPath)
		if err == nil {
			t.Error("Expected error for invalid YAML")
		}
	})

	t.Run("invalid config values", func(t *testing.T) {
		configPath := filepath.Join(tmpDir, "invalid-values.yaml")
		yamlContent := `app:
  name: ""
  version: 1.0.0
`
		if err := os.WriteFile(configPath, []byte(yamlContent), 0644); err != nil {
			t.Fatalf("Failed to write test config: %v", err)
		}

		_, err := LoadFromFile(configPath)
		if err == nil {
			t.Error("Expected validation error for empty app name")
		}
	})
}

func TestSaveToFile(t *testing.T) {
	tmpDir := t.TempDir()

	t.Run("save valid config", func(t *testing.T) {
		config := DefaultConfig()
		config.App.Name = "SaveTest"
		config.Game.TargetFPS = 45

		configPath := filepath.Join(tmpDir, "save-test.yaml")
		err := SaveToFile(config, configPath)
		if err != nil {
			t.Fatalf("Failed to save config: %v", err)
		}

		// Verify file exists
		if _, err := os.Stat(configPath); err != nil {
			t.Errorf("Config file not created: %v", err)
		}

		// Load it back
		loaded, err := LoadFromFile(configPath)
		if err != nil {
			t.Fatalf("Failed to load saved config: %v", err)
		}

		if loaded.App.Name != "SaveTest" {
			t.Errorf("Expected app name 'SaveTest', got %s", loaded.App.Name)
		}
		if loaded.Game.TargetFPS != 45 {
			t.Errorf("Expected FPS 45, got %d", loaded.Game.TargetFPS)
		}
	})

	t.Run("nil config", func(t *testing.T) {
		err := SaveToFile(nil, filepath.Join(tmpDir, "nil.yaml"))
		if err == nil {
			t.Error("Expected error for nil config")
		}
	})

	t.Run("empty path", func(t *testing.T) {
		config := DefaultConfig()
		err := SaveToFile(config, "")
		if err == nil {
			t.Error("Expected error for empty path")
		}
	})

	t.Run("invalid config", func(t *testing.T) {
		config := DefaultConfig()
		config.App.Name = "" // Invalid

		configPath := filepath.Join(tmpDir, "invalid.yaml")
		err := SaveToFile(config, configPath)
		if err == nil {
			t.Error("Expected error for invalid config")
		}
	})

	t.Run("create directory if missing", func(t *testing.T) {
		config := DefaultConfig()
		configPath := filepath.Join(tmpDir, "subdir", "nested", "config.yaml")

		err := SaveToFile(config, configPath)
		if err != nil {
			t.Fatalf("Failed to save config with nested path: %v", err)
		}

		if _, err := os.Stat(configPath); err != nil {
			t.Errorf("Config file not created: %v", err)
		}
	})
}

func TestEnvOverrides(t *testing.T) {
	// Save original env
	originalEnv := make(map[string]string)
	envVars := []string{
		"GOCHI_DEBUG", "GOCHI_LOG_PATH", "GOCHI_TARGET_FPS",
		"GOCHI_AUTO_SAVE", "GOCHI_DATA_PATH", "GOCHI_BACKUP_PATH",
		"GOCHI_ENCRYPTION_KEY", "GOCHI_CACHE_SIZE_MB",
		"GOCHI_UI_WIDTH", "GOCHI_UI_HEIGHT", "GOCHI_SHOW_FPS",
	}
	for _, key := range envVars {
		originalEnv[key] = os.Getenv(key)
		os.Unsetenv(key)
	}
	defer func() {
		for key, val := range originalEnv {
			if val != "" {
				os.Setenv(key, val)
			} else {
				os.Unsetenv(key)
			}
		}
	}()

	t.Run("debug override", func(t *testing.T) {
		os.Setenv("GOCHI_DEBUG", "true")
		defer os.Unsetenv("GOCHI_DEBUG")

		config := DefaultConfig()
		err := applyEnvOverrides(config)
		if err != nil {
			t.Fatalf("Failed to apply env overrides: %v", err)
		}

		if !config.App.Debug {
			t.Error("Expected debug to be true from env override")
		}
	})

	t.Run("target FPS override", func(t *testing.T) {
		os.Setenv("GOCHI_TARGET_FPS", "120")
		defer os.Unsetenv("GOCHI_TARGET_FPS")

		config := DefaultConfig()
		err := applyEnvOverrides(config)
		if err != nil {
			t.Fatalf("Failed to apply env overrides: %v", err)
		}

		if config.Game.TargetFPS != 120 {
			t.Errorf("Expected FPS 120 from env override, got %d", config.Game.TargetFPS)
		}
	})

	t.Run("data path override", func(t *testing.T) {
		os.Setenv("GOCHI_DATA_PATH", "/custom/path")
		defer os.Unsetenv("GOCHI_DATA_PATH")

		config := DefaultConfig()
		err := applyEnvOverrides(config)
		if err != nil {
			t.Fatalf("Failed to apply env overrides: %v", err)
		}

		if config.Data.BasePath != "/custom/path" {
			t.Errorf("Expected data path '/custom/path', got %s", config.Data.BasePath)
		}
	})

	t.Run("encryption key override", func(t *testing.T) {
		os.Setenv("GOCHI_ENCRYPTION_KEY", "secret-key-123")
		defer os.Unsetenv("GOCHI_ENCRYPTION_KEY")

		config := DefaultConfig()
		err := applyEnvOverrides(config)
		if err != nil {
			t.Fatalf("Failed to apply env overrides: %v", err)
		}

		if config.Data.EncryptionKey != "secret-key-123" {
			t.Errorf("Expected encryption key from env override")
		}
		if !config.Data.EnableEncryption {
			t.Error("Expected encryption to be enabled when key is set via env")
		}
	})

	t.Run("invalid env value", func(t *testing.T) {
		os.Setenv("GOCHI_TARGET_FPS", "invalid")
		defer os.Unsetenv("GOCHI_TARGET_FPS")

		config := DefaultConfig()
		err := applyEnvOverrides(config)
		if err == nil {
			t.Error("Expected error for invalid env value")
		}
	})

	t.Run("UI overrides", func(t *testing.T) {
		os.Setenv("GOCHI_UI_WIDTH", "120")
		os.Setenv("GOCHI_UI_HEIGHT", "40")
		os.Setenv("GOCHI_SHOW_FPS", "true")
		defer func() {
			os.Unsetenv("GOCHI_UI_WIDTH")
			os.Unsetenv("GOCHI_UI_HEIGHT")
			os.Unsetenv("GOCHI_SHOW_FPS")
		}()

		config := DefaultConfig()
		err := applyEnvOverrides(config)
		if err != nil {
			t.Fatalf("Failed to apply env overrides: %v", err)
		}

		if config.UI.Width != 120 {
			t.Errorf("Expected UI width 120, got %d", config.UI.Width)
		}
		if config.UI.Height != 40 {
			t.Errorf("Expected UI height 40, got %d", config.UI.Height)
		}
		if !config.UI.ShowFPS {
			t.Error("Expected show FPS to be true")
		}
	})
}
