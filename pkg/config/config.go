// Package config handles .sade/config.json for project settings.
package config

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
)

const (
	SadeDir        = ".sade"
	ConfigFile     = "config.json"
	ArchFile       = "architecture.json"
	NodesDir       = "nodes"
	PulsePrompt    = "pulse-prompt.md"
	HousekeepPrompt = "housekeeping-prompt.md"
)

// Config holds the SADE project configuration
type Config struct {
	// Agent is the configured coding agent (pi, claude, codex, etc.)
	Agent string `json:"agent"`

	// PulseMs is the silence duration before triggering reactive updates
	PulseMs int `json:"pulse_ms"`

	// HousekeepingMs is the interval between periodic maintenance runs
	HousekeepingMs int `json:"housekeeping_ms"`

	// PulseEnabled controls reactive updates
	PulseEnabled bool `json:"pulse_enabled"`

	// HousekeepingEnabled controls periodic maintenance
	HousekeepingEnabled bool `json:"housekeeping_enabled"`
}

// Default returns a config with sensible defaults
func Default() *Config {
	return &Config{
		Agent:               "",
		PulseMs:             15000,   // 15 seconds
		HousekeepingMs:      1800000, // 30 minutes
		PulseEnabled:        true,
		HousekeepingEnabled: true,
	}
}

// Paths returns common paths for a project
type Paths struct {
	Root        string
	SadeDir     string
	Config      string
	Arch        string
	Nodes       string
	PulsePrompt string
	HousekeepPrompt string
}

// GetPaths returns all paths for a project root
func GetPaths(projectRoot string) Paths {
	sadeDir := filepath.Join(projectRoot, SadeDir)
	return Paths{
		Root:        projectRoot,
		SadeDir:     sadeDir,
		Config:      filepath.Join(sadeDir, ConfigFile),
		Arch:        filepath.Join(sadeDir, ArchFile),
		Nodes:       filepath.Join(sadeDir, NodesDir),
		PulsePrompt: filepath.Join(sadeDir, PulsePrompt),
		HousekeepPrompt: filepath.Join(sadeDir, HousekeepPrompt),
	}
}

// Exists returns true if .sade/ exists in the project
func Exists(projectRoot string) bool {
	info, err := os.Stat(filepath.Join(projectRoot, SadeDir))
	if err != nil {
		return false
	}
	return info.IsDir()
}

// FindRoot walks up from startDir to find a directory containing .sade/
func FindRoot(startDir string) string {
	dir := startDir
	for {
		if Exists(dir) {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return ""
		}
		dir = parent
	}
}

// Load loads the config from .sade/config.json
func Load(projectRoot string) (*Config, error) {
	paths := GetPaths(projectRoot)

	data, err := os.ReadFile(paths.Config)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Default(), nil
		}
		return nil, err
	}

	cfg := Default()
	if err := json.Unmarshal(data, cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}

// Save writes the config to .sade/config.json
func Save(projectRoot string, cfg *Config) error {
	paths := GetPaths(projectRoot)

	if err := os.MkdirAll(paths.SadeDir, 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(paths.Config, data, 0644)
}

// EnsureDirs creates the .sade directory structure
func EnsureDirs(projectRoot string) error {
	paths := GetPaths(projectRoot)

	if err := os.MkdirAll(paths.SadeDir, 0755); err != nil {
		return err
	}
	if err := os.MkdirAll(paths.Nodes, 0755); err != nil {
		return err
	}

	return nil
}
