package logger

import (
	"bytes"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	if config == nil {
		t.Fatal("DefaultConfig returned nil")
	}

	if config.Level != slog.LevelInfo {
		t.Errorf("Expected level Info, got %v", config.Level)
	}

	if config.Format != "text" {
		t.Errorf("Expected format 'text', got %s", config.Format)
	}

	if config.AddSource {
		t.Error("Expected AddSource to be false")
	}
}

func TestInitialize(t *testing.T) {
	t.Run("text format", func(t *testing.T) {
		config := &Config{
			Level:  slog.LevelDebug,
			Format: "text",
		}

		err := Initialize(config)
		if err != nil {
			t.Fatalf("Failed to initialize logger: %v", err)
		}

		logger := Get()
		if logger == nil {
			t.Fatal("Logger is nil after initialization")
		}
	})

	t.Run("json format", func(t *testing.T) {
		config := &Config{
			Level:  slog.LevelInfo,
			Format: "json",
		}

		err := Initialize(config)
		if err != nil {
			t.Fatalf("Failed to initialize logger: %v", err)
		}

		logger := Get()
		if logger == nil {
			t.Fatal("Logger is nil after initialization")
		}
	})

	t.Run("with source", func(t *testing.T) {
		config := &Config{
			Level:     slog.LevelWarn,
			Format:    "text",
			AddSource: true,
		}

		err := Initialize(config)
		if err != nil {
			t.Fatalf("Failed to initialize logger: %v", err)
		}
	})

	t.Run("with file output", func(t *testing.T) {
		tmpDir := t.TempDir()
		logPath := filepath.Join(tmpDir, "test.log")

		config := &Config{
			Level:      slog.LevelInfo,
			Format:     "text",
			OutputPath: logPath,
		}

		err := Initialize(config)
		if err != nil {
			t.Fatalf("Failed to initialize logger with file: %v", err)
		}

		// Verify file was created
		if _, err := os.Stat(logPath); err != nil {
			t.Errorf("Log file not created: %v", err)
		}

		// Write a test message
		Info("test message")

		// Verify content was written
		content, err := os.ReadFile(logPath)
		if err != nil {
			t.Fatalf("Failed to read log file: %v", err)
		}

		if !strings.Contains(string(content), "test message") {
			t.Error("Log message not written to file")
		}
	})

	t.Run("nil config", func(t *testing.T) {
		err := Initialize(nil)
		if err != nil {
			t.Errorf("Expected nil config to use defaults, got error: %v", err)
		}
	})

	t.Run("create nested directory", func(t *testing.T) {
		tmpDir := t.TempDir()
		logPath := filepath.Join(tmpDir, "nested", "dir", "test.log")

		config := &Config{
			Level:      slog.LevelInfo,
			Format:     "text",
			OutputPath: logPath,
		}

		err := Initialize(config)
		if err != nil {
			t.Fatalf("Failed to create nested directory: %v", err)
		}

		if _, err := os.Stat(logPath); err != nil {
			t.Errorf("Log file not created in nested directory: %v", err)
		}
	})
}

func TestLoggingFunctions(t *testing.T) {
	// Capture output
	var buf bytes.Buffer

	// Create a test logger that writes to buffer
	testLogger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))

	mu.Lock()
	defaultLogger = testLogger
	mu.Unlock()

	t.Run("debug", func(t *testing.T) {
		buf.Reset()
		Debug("debug message", "key", "value")

		output := buf.String()
		if !strings.Contains(output, "debug message") {
			t.Error("Debug message not logged")
		}
		if !strings.Contains(output, "key=value") {
			t.Error("Debug attributes not logged")
		}
	})

	t.Run("info", func(t *testing.T) {
		buf.Reset()
		Info("info message", "count", 42)

		output := buf.String()
		if !strings.Contains(output, "info message") {
			t.Error("Info message not logged")
		}
		if !strings.Contains(output, "count=42") {
			t.Error("Info attributes not logged")
		}
	})

	t.Run("warn", func(t *testing.T) {
		buf.Reset()
		Warn("warning message", "reason", "test")

		output := buf.String()
		if !strings.Contains(output, "warning message") {
			t.Error("Warn message not logged")
		}
	})

	t.Run("error", func(t *testing.T) {
		buf.Reset()
		Error("error message", "error", "something went wrong")

		output := buf.String()
		if !strings.Contains(output, "error message") {
			t.Error("Error message not logged")
		}
	})
}

func TestWith(t *testing.T) {
	var buf bytes.Buffer

	testLogger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	mu.Lock()
	defaultLogger = testLogger
	mu.Unlock()

	contextLogger := With("component", "test", "version", "1.0")
	contextLogger.Info("test message")

	output := buf.String()
	if !strings.Contains(output, "component=test") {
		t.Error("Context attribute 'component' not found")
	}
	if !strings.Contains(output, "version=1.0") {
		t.Error("Context attribute 'version' not found")
	}
	if !strings.Contains(output, "test message") {
		t.Error("Message not logged")
	}
}

func TestWithGroup(t *testing.T) {
	var buf bytes.Buffer

	testLogger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	mu.Lock()
	defaultLogger = testLogger
	mu.Unlock()

	groupLogger := WithGroup("database")
	groupLogger.Info("query executed", "duration_ms", 150)

	output := buf.String()
	if !strings.Contains(output, "database") {
		t.Error("Group name not found in output")
	}
	if !strings.Contains(output, "query executed") {
		t.Error("Message not logged")
	}
}

func TestGet(t *testing.T) {
	logger := Get()
	if logger == nil {
		t.Fatal("Get() returned nil logger")
	}

	// Should be able to use it directly
	var buf bytes.Buffer
	testLogger := slog.New(slog.NewTextHandler(&buf, nil))

	mu.Lock()
	defaultLogger = testLogger
	mu.Unlock()

	retrieved := Get()
	if retrieved != testLogger {
		t.Error("Get() did not return the set logger")
	}
}

func TestShutdown(t *testing.T) {
	t.Run("shutdown with no file", func(t *testing.T) {
		// Initialize with no file
		config := &Config{
			Level:  slog.LevelInfo,
			Format: "text",
		}

		err := Initialize(config)
		if err != nil {
			t.Fatalf("Failed to initialize logger: %v", err)
		}

		// Shutdown should succeed even with no file
		err = Shutdown()
		if err != nil {
			t.Errorf("Shutdown failed with no file: %v", err)
		}
	})

	t.Run("shutdown with file", func(t *testing.T) {
		tmpDir := t.TempDir()
		logPath := filepath.Join(tmpDir, "shutdown-test.log")

		config := &Config{
			Level:      slog.LevelInfo,
			Format:     "text",
			OutputPath: logPath,
		}

		err := Initialize(config)
		if err != nil {
			t.Fatalf("Failed to initialize logger: %v", err)
		}

		// Write a message
		Info("test message before shutdown")

		// Shutdown should close the file
		err = Shutdown()
		if err != nil {
			t.Errorf("Shutdown failed: %v", err)
		}

		// Verify file exists and has content
		content, err := os.ReadFile(logPath)
		if err != nil {
			t.Fatalf("Failed to read log file after shutdown: %v", err)
		}

		if !strings.Contains(string(content), "test message before shutdown") {
			t.Error("Log message not found in file after shutdown")
		}
	})

	t.Run("multiple shutdowns", func(t *testing.T) {
		tmpDir := t.TempDir()
		logPath := filepath.Join(tmpDir, "multi-shutdown.log")

		config := &Config{
			Level:      slog.LevelInfo,
			Format:     "text",
			OutputPath: logPath,
		}

		err := Initialize(config)
		if err != nil {
			t.Fatalf("Failed to initialize logger: %v", err)
		}

		// First shutdown
		err = Shutdown()
		if err != nil {
			t.Errorf("First shutdown failed: %v", err)
		}

		// Second shutdown should not error
		err = Shutdown()
		if err != nil {
			t.Errorf("Second shutdown failed: %v", err)
		}
	})

	t.Run("reinitialize after shutdown", func(t *testing.T) {
		tmpDir := t.TempDir()
		logPath1 := filepath.Join(tmpDir, "log1.log")
		logPath2 := filepath.Join(tmpDir, "log2.log")

		// First initialization
		config1 := &Config{
			Level:      slog.LevelInfo,
			Format:     "text",
			OutputPath: logPath1,
		}

		err := Initialize(config1)
		if err != nil {
			t.Fatalf("Failed first initialization: %v", err)
		}

		Info("message in log1")

		// Shutdown
		err = Shutdown()
		if err != nil {
			t.Errorf("Shutdown failed: %v", err)
		}

		// Second initialization with different file
		config2 := &Config{
			Level:      slog.LevelInfo,
			Format:     "text",
			OutputPath: logPath2,
		}

		err = Initialize(config2)
		if err != nil {
			t.Fatalf("Failed second initialization: %v", err)
		}

		Info("message in log2")

		// Shutdown again
		err = Shutdown()
		if err != nil {
			t.Errorf("Second shutdown failed: %v", err)
		}

		// Verify both files have their respective messages
		content1, _ := os.ReadFile(logPath1)
		if !strings.Contains(string(content1), "message in log1") {
			t.Error("Log1 doesn't have expected message")
		}

		content2, _ := os.ReadFile(logPath2)
		if !strings.Contains(string(content2), "message in log2") {
			t.Error("Log2 doesn't have expected message")
		}
	})
}
