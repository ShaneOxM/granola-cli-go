//go:build integration
// +build integration

package api

import (
	"os"
	"testing"
)

// TestIntegrationClientCreation tests that the client can be created
func TestIntegrationClientCreation(t *testing.T) {
	if os.Getenv("CI") == "true" {
		t.Skip("Skipping integration test in CI")
	}

	client := NewClient()
	if client == nil {
		t.Fatal("NewClient() returned nil")
	}

	// Note: BinaryPath may be empty if granola CLI is not installed
	t.Logf("Client created. BinaryPath: %s", client.BinaryPath)
}

// TestIntegrationValidateFunctions tests all validation functions
func TestIntegrationValidateFunctions(t *testing.T) {
	tests := []struct {
		name      string
		validator func(string) error
		valid     string
		invalid   string
	}{
		{
			name:      "ID validation",
			validator: validateID,
			valid:     "550e8400-e29b-41d4-a716-446655440000",
			invalid:   "",
		},
		{
			name:      "Workspace validation",
			validator: validateWorkspace,
			valid:     "my-workspace",
			invalid:   "my/workspace",
		},
		{
			name:      "Folder validation",
			validator: validateFolder,
			valid:     "my-folder",
			invalid:   "my folder",
		},
		{
			name:      "Email validation",
			validator: validateEmail,
			valid:     "user@example.com",
			invalid:   "no-at-sign",
		},
		{
			name:      "Search validation",
			validator: validateSearch,
			valid:     "meeting discussion",
			invalid:   "meeting;DROP TABLE",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test valid input
			err := tt.validator(tt.valid)
			if err != nil {
				t.Errorf("validate(%s) = %v, want nil", tt.valid, err)
			}

			// Test invalid input
			err = tt.validator(tt.invalid)
			if err == nil {
				t.Errorf("validate(%s) = nil, want error", tt.invalid)
			}
		})
	}
}
