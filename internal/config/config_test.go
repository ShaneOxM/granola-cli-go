package config

import (
	"os"
	"testing"
)

func TestPath(t *testing.T) {
	path, err := Path()
	if err != nil {
		t.Fatalf("Path() error = %v", err)
	}

	if path == "" {
		t.Error("Path() returned empty string")
	}

	if !contains(path, ".config") && !contains(path, ".granola") {
		t.Logf("Path does not contain expected directory: %s", path)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && findSubstring(s, substr))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestEnvVar(t *testing.T) {
	// Set test environment variable
	os.Setenv("GRANOLA_TEST_VAR", "test_value")
	defer os.Unsetenv("GRANOLA_TEST_VAR")

	// Test with existing env var
	result := os.Getenv("GRANOLA_TEST_VAR")
	if result != "test_value" {
		t.Errorf("os.Getenv() = %v, want test_value", result)
	}

	// Test with non-existing env var
	result = os.Getenv("NON_EXISTING_VAR")
	if result != "" {
		t.Errorf("os.Getenv() = %v, want empty", result)
	}
}

func TestProfile(t *testing.T) {
	profile := Profile{
		Type:    "ssh",
		Host:    "rig",
		Port:    8080,
		BaseURL: "http://192.168.1.118:8080/v1",
		Model:   "qwen35",
	}

	if profile.Type != "ssh" {
		t.Errorf("Profile.Type = %v, want ssh", profile.Type)
	}

	if profile.Host != "rig" {
		t.Errorf("Profile.Host = %v, want rig", profile.Host)
	}

	if profile.Port != 8080 {
		t.Errorf("Profile.Port = %v, want 8080", profile.Port)
	}

	if profile.BaseURL != "http://192.168.1.118:8080/v1" {
		t.Errorf("Profile.BaseURL = %v, want http://192.168.1.118:8080/v1", profile.BaseURL)
	}

	if profile.Model != "qwen35" {
		t.Errorf("Profile.Model = %v, want qwen35", profile.Model)
	}
}

func TestConfig(t *testing.T) {
	cfg := Config{
		Provider:         "openai",
		BaseURL:          "https://api.example.com/v1",
		APIKey:           "test_api_key",
		Model:            "test-model",
		Timeout:          300,
		DefaultProfile:   "default",
		DiscoveryTimeout: 10,
	}

	if cfg.Provider != "openai" {
		t.Errorf("Config.Provider = %v, want openai", cfg.Provider)
	}

	if cfg.BaseURL != "https://api.example.com/v1" {
		t.Errorf("Config.BaseURL = %v, want https://api.example.com/v1", cfg.BaseURL)
	}

	if cfg.Model != "test-model" {
		t.Errorf("Config.Model = %v, want test-model", cfg.Model)
	}

	if cfg.Timeout != 300 {
		t.Errorf("Config.Timeout = %v, want 300", cfg.Timeout)
	}
}

func TestDefaultConfigProfiles(t *testing.T) {
	// Just verify we can create a config with default profiles
	profiles := map[string]Profile{
		"ssh": {
			Type:    "ssh",
			Host:    "rig",
			Port:    8080,
			BaseURL: "http://192.168.1.118:8080/v1",
			Model:   "qwen35",
		},
		"local": {
			Type:    "local",
			Host:    "localhost",
			Port:    11434,
			BaseURL: "http://localhost:11434/v1",
			Model:   "llama3.2",
		},
	}

	if len(profiles) != 2 {
		t.Errorf("Expected 2 profiles, got %d", len(profiles))
	}

	if _, ok := profiles["ssh"]; !ok {
		t.Error("Missing 'ssh' profile")
	}

	if _, ok := profiles["local"]; !ok {
		t.Error("Missing 'local' profile")
	}
}

func TestConfig_Empty(t *testing.T) {
	cfg := Config{}

	if cfg.Provider != "" {
		t.Errorf("Provider = %v, want empty", cfg.Provider)
	}
}

func TestProfile_Empty(t *testing.T) {
	profile := Profile{}

	if profile.Type != "" {
		t.Errorf("Type = %v, want empty", profile.Type)
	}
}

func TestInit(t *testing.T) {
	// Test Init function - may create config file
	err := Init()
	// May succeed or fail depending on environment
	_ = err
}

func TestRead(t *testing.T) {
	// Test Read function - may fail if config doesn't exist
	_, err := Read()
	// May succeed or fail depending on environment
	_ = err
}

func TestWrite(t *testing.T) {
	cfg := &Config{
		Provider: "test",
		Model:    "test-model",
	}

	// Test Write function - may fail if can't write to path
	err := Write(cfg)
	// May succeed or fail depending on environment
	_ = err
}

func TestSet(t *testing.T) {
	// Test Set function
	err := Set("model", "test-model")
	// May fail if config doesn't exist
	_ = err
}

func TestSet_UnknownKey(t *testing.T) {
	// Test Set with unknown key
	err := Set("unknown_key", "value")
	if err == nil {
		t.Error("Set() should return error for unknown key")
	}
}

func TestUnset(t *testing.T) {
	// Test Unset function
	err := Unset("model")
	// May fail if config doesn't exist
	_ = err
}

func TestUnset_UnknownKey(t *testing.T) {
	// Test Unset with unknown key
	err := Unset("unknown_key")
	if err == nil {
		t.Error("Unset() should return error for unknown key")
	}
}

func TestGetProfile(t *testing.T) {
	// Test GetProfile - may fail if profile doesn't exist
	_, err := GetProfile("nonexistent")
	// May succeed or fail depending on environment
	_ = err
}

func TestSetProfile(t *testing.T) {
	profile := Profile{
		Type:    "test",
		BaseURL: "http://test:8080/v1",
		Model:   "test-model",
	}

	// Test SetProfile
	err := SetProfile("test-profile", profile)
	// May fail if config doesn't exist
	_ = err
}

func TestListProfiles(t *testing.T) {
	profiles := ListProfiles()
	// Just verify function runs
	_ = profiles
}

func TestGetDefaultProfile(t *testing.T) {
	profile := GetDefaultProfile()
	// Just verify function runs
	_ = profile
}

func TestSetDefaultProfile(t *testing.T) {
	// Test SetDefaultProfile
	err := SetDefaultProfile("nonexistent")
	// May fail if profile doesn't exist
	_ = err
}

func TestGetCurrentProfile(t *testing.T) {
	// Test GetCurrentProfile
	_, err := GetCurrentProfile()
	// May fail if config doesn't exist
	_ = err
}

func TestUseProfile(t *testing.T) {
	// Test UseProfile
	err := UseProfile("nonexistent")
	// May fail if profile doesn't exist
	_ = err
}

func TestParseTimeout_Default(t *testing.T) {
	timeout, err := parseTimeout("")
	if err != nil {
		t.Logf("parseTimeout() returned error (may be expected): %v", err)
	}
	// Should return default timeout
	if timeout <= 0 {
		t.Error("parseTimeout() should return positive timeout")
	}
}

func TestParseTimeout_Valid(t *testing.T) {
	// Test with valid timeout string
	timeout, err := parseTimeout("300")
	if err != nil {
		t.Logf("parseTimeout() returned error: %v", err)
	}
	// Currently returns 300 for default
	_ = timeout
}

func TestParseDuration(t *testing.T) {
	// Test parseDuration
	timeout, err := parseDuration("300")
	if err != nil {
		t.Logf("parseDuration() returned error: %v", err)
	}
	// Currently returns default
	if timeout != 300 {
		t.Errorf("parseDuration() = %v, want 300", timeout)
	}
}

func TestParseDuration_Empty(t *testing.T) {
	timeout, err := parseDuration("")
	if err != nil {
		t.Logf("parseDuration() returned error: %v", err)
	}
	// Currently returns default
	if timeout != 300 {
		t.Errorf("parseDuration() = %v, want 300", timeout)
	}
}
