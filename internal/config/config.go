// Package config provides configuration management for granola-cli.
// It handles profile management, environment variables, and JSON configuration files.
package config

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"

	"github.com/ShaneOxM/granola-cli-go/internal/logger"
)

var configPath string

// Profile represents a configured inference endpoint
type Profile struct {
	Type    string `json:"type"`
	Host    string `json:"host,omitempty"`
	RigIP   string `json:"rig_ip,omitempty"`
	Port    int    `json:"port"`
	BaseURL string `json:"base_url"`
	Model   string `json:"model"`
}

// Config represents the granola-cli configuration
type Config struct {
	Provider            string             `json:"provider"`
	BaseURL             string             `json:"base_url"`
	APIKey              string             `json:"api_key"`
	Model               string             `json:"model"`
	Timeout             int                `json:"timeout"`
	GoogleClientID      string             `json:"google_client_id,omitempty"`
	GoogleClientSecret  string             `json:"google_client_secret,omitempty"`
	GoogleAuthMode      string             `json:"google_auth_mode,omitempty"`
	GoogleActiveAccount string             `json:"google_active_account,omitempty"`
	GoogleAccounts      map[string]string  `json:"google_accounts,omitempty"`
	Profiles            map[string]Profile `json:"profiles"`
	DefaultProfile      string             `json:"default_profile"`
	DiscoveryTimeout    int                `json:"discovery_timeout"`
}

func Path() (string, error) {
	// Check environment variable first
	if path := os.Getenv("GRANOLA_CONFIG_PATH"); path != "" {
		return path, nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "granola-cli", "config.json"), nil
}

func Init() error {
	path, err := Path()
	if err != nil {
		return err
	}
	configPath = path

	// Create config dir if not exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	// Create default config if not exists
	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		logger.Info("Creating default config", "path", path)
		cfg := &Config{
			Provider:         "openai",
			Timeout:          300,
			DiscoveryTimeout: 10,
			Profiles: map[string]Profile{
				"ssh": {
					Type:    "ssh",
					Host:    "rig",
					Port:    8080,
					BaseURL: "http://192.168.1.118:8080/v1",
					Model:   "qwen35",
				},
				"tailscale": {
					Type:    "tailscale",
					RigIP:   "100.127.102.71",
					Port:    8080,
					BaseURL: "http://100.127.102.71:8080/v1",
					Model:   "qwen35",
				},
				"ollama": {
					Type:    "local",
					Host:    "localhost",
					Port:    11434,
					BaseURL: "http://localhost:11434/v1",
					Model:   "llama3.2",
				},
				"lmstudio": {
					Type:    "local",
					Host:    "localhost",
					Port:    1234,
					BaseURL: "http://localhost:1234/v1",
					Model:   "auto",
				},
			},
			DefaultProfile: "ssh",
		}
		if err := Write(cfg); err != nil {
			return err
		}
	}

	return nil
}

func Read() (*Config, error) {
	path, err := Path()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

func Write(cfg *Config) error {
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}

	path, err := Path()
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0600)
}

func Set(key, value string) error {
	cfg, err := Read()
	if err != nil {
		return err
	}

	switch key {
	case "base_url":
		cfg.BaseURL = value
	case "model":
		cfg.Model = value
	case "timeout":
		if t, err := parseTimeout(value); err == nil {
			cfg.Timeout = t
		} else {
			return err
		}
	case "google_client_id":
		cfg.GoogleClientID = value
	case "google_client_secret":
		cfg.GoogleClientSecret = value
	default:
		return errors.New("unknown config key: " + key)
	}

	return Write(cfg)
}

func Unset(key string) error {
	cfg, err := Read()
	if err != nil {
		return err
	}

	switch key {
	case "base_url":
		cfg.BaseURL = ""
	case "model":
		cfg.Model = ""
	case "timeout":
		cfg.Timeout = 300
	case "google_client_id":
		cfg.GoogleClientID = ""
	case "google_client_secret":
		cfg.GoogleClientSecret = ""
	default:
		return errors.New("unknown config key: " + key)
	}

	return Write(cfg)
}

// GetProfile returns a profile by name
func GetProfile(name string) (*Profile, error) {
	cfg, err := Read()
	if err != nil {
		return nil, err
	}

	profile, ok := cfg.Profiles[name]
	if !ok {
		return nil, errors.New("profile not found: " + name)
	}

	return &profile, nil
}

// SetProfile updates or creates a profile
func SetProfile(name string, profile Profile) error {
	cfg, err := Read()
	if err != nil {
		return err
	}

	if cfg.Profiles == nil {
		cfg.Profiles = make(map[string]Profile)
	}

	cfg.Profiles[name] = profile
	return Write(cfg)
}

// ListProfiles returns all available profile names
func ListProfiles() []string {
	cfg, _ := Read()
	profiles := make([]string, 0, len(cfg.Profiles))
	for name := range cfg.Profiles {
		profiles = append(profiles, name)
	}
	return profiles
}

// GetDefaultProfile returns the default profile name
func GetDefaultProfile() string {
	cfg, _ := Read()
	return cfg.DefaultProfile
}

// SetDefaultProfile sets the default profile
func SetDefaultProfile(name string) error {
	cfg, err := Read()
	if err != nil {
		return err
	}

	if _, ok := cfg.Profiles[name]; !ok {
		return errors.New("profile not found: " + name)
	}

	cfg.DefaultProfile = name
	return Write(cfg)
}

// GetCurrentProfile returns the currently active profile
func GetCurrentProfile() (*Profile, error) {
	// Check env var first
	if envProfile := os.Getenv("GRANOLA_PROFILE"); envProfile != "" {
		return GetProfile(envProfile)
	}

	// Fall back to config
	cfg, err := Read()
	if err != nil {
		return nil, err
	}

	return GetProfile(cfg.DefaultProfile)
}

// UseProfile switches to a profile
func UseProfile(name string) error {
	return SetDefaultProfile(name)
}

// parseTimeout parses timeout string to seconds
func parseTimeout(s string) (int, error) {
	if s == "" {
		return 300, nil
	}

	// Try parsing as plain seconds
	var timeout int
	_, err := parseDuration(s)
	if err != nil {
		// Try parsing as plain integer
		// For now, just return default
		return 300, nil
	}
	return timeout, nil
}

// parseDuration parses duration string (simplified)
func parseDuration(s string) (int, error) {
	// For now, just return default
	return 300, nil
}
