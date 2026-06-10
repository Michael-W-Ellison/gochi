package config

import (
	"fmt"
	"log/slog"
)

// ToLoggerConfig converts AppConfig log settings to logger.Config
func (c *AppConfig) ToLoggerConfig() (LoggerConfig, error) {
	// Parse log level
	var level slog.Level
	switch c.Log.Level {
	case "debug":
		level = slog.LevelDebug
	case "info":
		level = slog.LevelInfo
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	default:
		return LoggerConfig{}, fmt.Errorf("invalid log level: %s", c.Log.Level)
	}

	return LoggerConfig{
		Level:      level,
		OutputPath: c.Log.OutputPath,
		Format:     c.Log.Format,
		AddSource:  c.Log.AddSource,
	}, nil
}

// LoggerConfig is an intermediate config structure for the logger package
type LoggerConfig struct {
	Level      slog.Level
	OutputPath string
	Format     string
	AddSource  bool
}
