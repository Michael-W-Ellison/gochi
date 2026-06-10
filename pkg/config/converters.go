package config

import (
	"time"
)

// ToDataConfig converts AppConfig to data package configuration
// This bridges the centralized config to the data package's Config struct
func (c *AppConfig) ToDataConfig() DataConfig {
	return DataConfig{
		BasePath:           c.Data.BasePath,
		BackupPath:         c.Data.BackupPath,
		CacheSizeMB:        c.Data.CacheSizeMB,
		CacheTTLMinutes:    c.Data.CacheTTLMinutes,
		EnableEncryption:   c.Data.EnableEncryption,
		EncryptionKey:      c.Data.EncryptionKey,
		MaxBackups:         c.Data.MaxBackups,
		EnableAutoSave:     c.Data.EnableAutoSave,
		AutoSaveInterval:   c.Data.AutoSaveInterval,
		EnableAutoBackup:   c.Data.EnableAutoBackup,
		AutoBackupInterval: c.Data.AutoBackupInterval,
	}
}

// ToGameLoopConfig converts AppConfig to game loop configuration
func (c *AppConfig) ToGameLoopConfig() GameLoopConfig {
	return GameLoopConfig{
		TargetFPS:          c.Game.TargetFPS,
		AutoSaveInterval:   c.Game.AutoSaveInterval,
		AutoBackupInterval: c.Game.AutoBackupInterval,
		EnableAutoSave:     c.Game.EnableAutoSave,
		EnableAutoBackup:   c.Game.EnableAutoBackup,
	}
}

// ToEnvironmentConfig converts AppConfig to environment configuration
func (c *AppConfig) ToEnvironmentConfig() EnvironmentConfig {
	return EnvironmentConfig{
		StartSeason:           c.Environment.StartSeason,
		SeasonLengthHours:     c.Environment.SeasonLengthHours,
		SeasonalIntensity:     c.Environment.SeasonalIntensity,
		EnableTransitions:     c.Environment.EnableTransitions,
		WeatherVolatility:     c.Environment.WeatherVolatility,
		WeatherChangeInterval: c.Environment.WeatherChangeInterval,
		StartingBiome:         c.Environment.StartingBiome,
	}
}

// DataConfig is an intermediate config structure for the data package
type DataConfig struct {
	BasePath           string
	BackupPath         string
	CacheSizeMB        int
	CacheTTLMinutes    int
	EnableEncryption   bool
	EncryptionKey      string
	MaxBackups         int
	EnableAutoSave     bool
	AutoSaveInterval   time.Duration
	EnableAutoBackup   bool
	AutoBackupInterval time.Duration
}

// GameLoopConfig is an intermediate config structure for the game loop
type GameLoopConfig struct {
	TargetFPS          int
	AutoSaveInterval   time.Duration
	AutoBackupInterval time.Duration
	EnableAutoSave     bool
	EnableAutoBackup   bool
}

// EnvironmentConfig is an intermediate config structure for the environment
type EnvironmentConfig struct {
	StartSeason           string
	SeasonLengthHours     float64
	SeasonalIntensity     float64
	EnableTransitions     bool
	WeatherVolatility     float64
	WeatherChangeInterval float64
	StartingBiome         string
}
