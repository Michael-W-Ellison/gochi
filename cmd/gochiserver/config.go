package main

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"time"

	"gopkg.in/yaml.v3"
)

// Config holds the server's settings. They come from the defaults, then
// the config file, then any command-line options, each overriding the
// one before.
type Config struct {
	Server struct {
		Addr         string `yaml:"addr"`           // Address and port to listen on
		DataDir      string `yaml:"data_dir"`       // Saves and backups; "" means ~/.gochi
		APIToken     string `yaml:"api_token"`      // Turns on the REST API; "" leaves it off
		APIRateLimit int    `yaml:"api_rate_limit"` // REST API requests per minute per client; 0 turns the limit off
	} `yaml:"server"`

	Hub struct {
		Enabled  bool   `yaml:"enabled"`   // Also run as a cloud hub
		JoinCode string `yaml:"join_code"` // Players need it to register; "" closes registration
	} `yaml:"hub"`

	Simulation struct {
		TimeScale string        `yaml:"time_scale"` // game, 24x, 4x or realtime
		Autosave  time.Duration `yaml:"autosave"`   // 0 turns autosave off
		AwayPause time.Duration `yaml:"away_pause"` // 0 turns the away pause off
	} `yaml:"simulation"`
}

// defaultConfig returns the settings used when neither the config file
// nor a command-line option sets them
func defaultConfig() Config {
	var c Config
	c.Server.Addr = ":8080"
	c.Server.APIRateLimit = 120
	c.Simulation.TimeScale = "game"
	c.Simulation.Autosave = time.Minute
	c.Simulation.AwayPause = 2 * time.Minute
	return c
}

// unknownField matches the decoder's message for a setting with no field
var unknownField = regexp.MustCompile(`field (\S+) not found in type .*`)

// loadConfig reads a config file over the defaults. Settings the file
// leaves out keep their defaults; unknown settings are an error, so a
// misspelled setting is not silently ignored. A relative data_dir is
// relative to the file's directory.
func loadConfig(path string) (Config, error) {
	c := defaultConfig()
	data, err := os.ReadFile(path)
	if err != nil {
		return c, err
	}

	decoder := yaml.NewDecoder(bytes.NewReader(data))
	decoder.KnownFields(true)
	if err := decoder.Decode(&c); err != nil && !errors.Is(err, io.EOF) {
		var typeErr *yaml.TypeError
		if errors.As(err, &typeErr) {
			// Name unknown settings instead of the Go types they don't fit
			for i, e := range typeErr.Errors {
				typeErr.Errors[i] = unknownField.ReplaceAllString(e, `unknown setting "$1"`)
			}
		}
		return c, fmt.Errorf("%s: %w", path, err)
	}

	if c.Server.DataDir != "" && !filepath.IsAbs(c.Server.DataDir) {
		c.Server.DataDir = filepath.Join(filepath.Dir(path), c.Server.DataDir)
	}
	if err := c.validate(); err != nil {
		return c, fmt.Errorf("%s: %w", path, err)
	}
	return c, nil
}

// validate checks settings that have a fixed set of values or a range
func (c Config) validate() error {
	if _, ok := timeScales[c.Simulation.TimeScale]; !ok {
		return fmt.Errorf("unknown time_scale %q (use game, 24x, 4x or realtime)", c.Simulation.TimeScale)
	}
	if c.Simulation.Autosave < 0 || c.Simulation.AwayPause < 0 {
		return errors.New("autosave and away_pause can't be negative")
	}
	if c.Server.APIRateLimit < 0 {
		return errors.New("api_rate_limit can't be negative")
	}
	return nil
}

// defaultConfigPath returns configs/config.yaml in the project directory
// (the one holding the web directory), or "" if there is none
func defaultConfigPath(webDir string) string {
	path := filepath.Join(filepath.Dir(webDir), "configs", "config.yaml")
	if info, err := os.Stat(path); err == nil && !info.IsDir() {
		return path
	}
	return ""
}
