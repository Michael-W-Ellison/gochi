package logger

import (
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
)

var (
	// defaultLogger is the global logger instance
	defaultLogger *slog.Logger
	mu            sync.RWMutex
)

// Config holds logger configuration
type Config struct {
	Level      slog.Level
	OutputPath string
	Format     string // "json" or "text"
	AddSource  bool
}

// DefaultConfig returns default logger configuration
func DefaultConfig() *Config {
	return &Config{
		Level:      slog.LevelInfo,
		OutputPath: "",
		Format:     "text",
		AddSource:  false,
	}
}

func init() {
	// Initialize with default text logger to stdout
	defaultLogger = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
}

// Initialize sets up the global logger with the given configuration
func Initialize(config *Config) error {
	if config == nil {
		config = DefaultConfig()
	}

	var writer io.Writer = os.Stdout

	// If output path is specified, create log file
	if config.OutputPath != "" {
		// Ensure log directory exists
		dir := filepath.Dir(config.OutputPath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}

		// Open log file (append mode)
		file, err := os.OpenFile(config.OutputPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			return err
		}

		// Use multi-writer to write to both stdout and file
		writer = io.MultiWriter(os.Stdout, file)
	}

	// Create handler based on format
	var handler slog.Handler
	opts := &slog.HandlerOptions{
		Level:     config.Level,
		AddSource: config.AddSource,
	}

	if config.Format == "json" {
		handler = slog.NewJSONHandler(writer, opts)
	} else {
		handler = slog.NewTextHandler(writer, opts)
	}

	// Create and set the logger
	logger := slog.New(handler)

	mu.Lock()
	defaultLogger = logger
	mu.Unlock()

	return nil
}

// Get returns the global logger instance
func Get() *slog.Logger {
	mu.RLock()
	defer mu.RUnlock()
	return defaultLogger
}

// Debug logs a debug message
func Debug(msg string, args ...any) {
	Get().Debug(msg, args...)
}

// Info logs an info message
func Info(msg string, args ...any) {
	Get().Info(msg, args...)
}

// Warn logs a warning message
func Warn(msg string, args ...any) {
	Get().Warn(msg, args...)
}

// Error logs an error message
func Error(msg string, args ...any) {
	Get().Error(msg, args...)
}

// With returns a logger with the given attributes
func With(args ...any) *slog.Logger {
	return Get().With(args...)
}

// WithGroup returns a logger with a group
func WithGroup(name string) *slog.Logger {
	return Get().WithGroup(name)
}
