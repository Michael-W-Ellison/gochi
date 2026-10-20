package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func writeConfig(t *testing.T, text string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte(text), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestShippedConfigMatchesDefaults(t *testing.T) {
	config, err := loadConfig(filepath.Join("..", "..", "configs", "config.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if config != defaultConfig() {
		t.Errorf("configs/config.yaml should hold the defaults:\n got %+v\nwant %+v", config, defaultConfig())
	}
}

func TestLoadConfig(t *testing.T) {
	path := writeConfig(t, `
server:
  addr: ":9090"
  data_dir: "saves-here"
simulation:
  time_scale: "24x"
  autosave: 30s
`)
	config, err := loadConfig(path)
	if err != nil {
		t.Fatal(err)
	}
	if config.Server.Addr != ":9090" || config.Simulation.TimeScale != "24x" || config.Simulation.Autosave != 30*time.Second {
		t.Errorf("Settings not read: %+v", config)
	}
	if config.Simulation.AwayPause != 2*time.Minute {
		t.Errorf("A setting left out should keep its default, got %s", config.Simulation.AwayPause)
	}
	if want := filepath.Join(filepath.Dir(path), "saves-here"); config.Server.DataDir != want {
		t.Errorf("A relative data_dir should be relative to the file: got %s, want %s", config.Server.DataDir, want)
	}

	if config, err := loadConfig(writeConfig(t, "")); err != nil || config != defaultConfig() {
		t.Errorf("An empty file should give the defaults: %+v, %v", config, err)
	}
}

func TestLoadConfigRejectsBadSettings(t *testing.T) {
	for text, want := range map[string]string{
		"simulation:\n  tick_rate: 60\n":        `unknown setting "tick_rate"`,
		"simulation:\n  time_scale: \"fast\"\n": "unknown time_scale",
		"simulation:\n  autosave: -1m\n":        "can't be negative",
		"simulation:\n  autosave: often\n":      "often",
		"server:\n  api_rate_limit: -5\n":       "api_rate_limit can't be negative",
	} {
		if _, err := loadConfig(writeConfig(t, text)); err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("%q: expected an error mentioning %q, got %v", text, want, err)
		}
	}
	if _, err := loadConfig(filepath.Join(t.TempDir(), "missing.yaml")); err == nil {
		t.Error("A missing file should be an error")
	}
}
