package config

import (
	"log/slog"
	"testing"
)

func TestToLoggerConfig(t *testing.T) {
	tests := []struct {
		name      string
		logLevel  string
		logFormat string
		addSource bool
		wantLevel slog.Level
		wantErr   bool
	}{
		{
			name:      "debug level",
			logLevel:  "debug",
			logFormat: "text",
			addSource: false,
			wantLevel: slog.LevelDebug,
			wantErr:   false,
		},
		{
			name:      "info level",
			logLevel:  "info",
			logFormat: "json",
			addSource: true,
			wantLevel: slog.LevelInfo,
			wantErr:   false,
		},
		{
			name:      "warn level",
			logLevel:  "warn",
			logFormat: "text",
			addSource: false,
			wantLevel: slog.LevelWarn,
			wantErr:   false,
		},
		{
			name:      "error level",
			logLevel:  "error",
			logFormat: "json",
			addSource: true,
			wantLevel: slog.LevelError,
			wantErr:   false,
		},
		{
			name:      "invalid level",
			logLevel:  "invalid",
			logFormat: "text",
			addSource: false,
			wantLevel: slog.LevelInfo,
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := DefaultConfig()
			config.Log.Level = tt.logLevel
			config.Log.Format = tt.logFormat
			config.Log.AddSource = tt.addSource

			loggerConfig, err := config.ToLoggerConfig()

			if tt.wantErr {
				if err == nil {
					t.Error("Expected error but got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if loggerConfig.Level != tt.wantLevel {
				t.Errorf("Expected level %v, got %v", tt.wantLevel, loggerConfig.Level)
			}

			if loggerConfig.Format != tt.logFormat {
				t.Errorf("Expected format %s, got %s", tt.logFormat, loggerConfig.Format)
			}

			if loggerConfig.AddSource != tt.addSource {
				t.Errorf("Expected addSource %v, got %v", tt.addSource, loggerConfig.AddSource)
			}
		})
	}
}

func TestToLoggerConfigWithOutputPath(t *testing.T) {
	config := DefaultConfig()
	config.Log.Level = "info"
	config.Log.Format = "json"
	config.Log.OutputPath = "/tmp/test.log"

	loggerConfig, err := config.ToLoggerConfig()
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if loggerConfig.OutputPath != "/tmp/test.log" {
		t.Errorf("Expected output path '/tmp/test.log', got %s", loggerConfig.OutputPath)
	}
}
