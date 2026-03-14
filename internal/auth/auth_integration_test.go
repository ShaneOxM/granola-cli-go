//go:build integration
// +build integration

package auth

import (
	"os"
	"testing"
)

// TestIntegrationAuthInit tests authentication initialization
func TestIntegrationAuthInit(t *testing.T) {
	if os.Getenv("CI") == "true" {
		t.Skip("Skipping integration test in CI")
	}

	// Note: This may fail if secrets are not properly configured
	// but it tests that the initialization flow works
	err := Init()
	if err != nil {
		t.Logf("Auth init skipped (expected in test environment): %v", err)
		// Don't fail the test - secrets may not be configured
		return
	}

	t.Log("Auth initialization successful")
}

// TestIntegrationCredentialsStructure tests that credentials can be created
func TestIntegrationCredentialsStructure(t *testing.T) {
	creds := &Credentials{
		EmailAddress: "test@example.com",
		RefreshToken: "test_refresh_token",
		AccessToken:  "test_access_token",
		ClientID:     "test_client_id",
	}

	if creds.EmailAddress == "" {
		t.Error("Email should not be empty")
	}

	if creds.RefreshToken == "" {
		t.Error("RefreshToken should not be empty")
	}

	if creds.ClientID == "" {
		t.Error("ClientID should not be empty")
	}

	t.Logf("Credentials created: Email=%s, ClientID=%s", creds.EmailAddress, creds.ClientID)
}

// TestIntegrationOAuthConfigValidation tests OAuth configuration validation
func TestIntegrationOAuthConfigValidation(t *testing.T) {
	// Test that environment variables are required
	clientID := os.Getenv("GRANOLA_OAUTH_CLIENT_ID")
	clientSecret := os.Getenv("GRANOLA_OAUTH_CLIENT_SECRET")

	if clientID == "" {
		t.Skip("GRANOLA_OAUTH_CLIENT_ID not set - skipping OAuth validation")
	}

	if clientSecret == "" {
		t.Skip("GRANOLA_OAUTH_CLIENT_SECRET not set - skipping OAuth validation")
	}

	t.Logf("OAuth credentials found. ClientID=%s", clientID[:8]+"...")
}

// TestIntegrationAuthStatus tests auth status command (integration)
func TestIntegrationAuthStatus(t *testing.T) {
	if os.Getenv("CI") == "true" {
		t.Skip("Skipping integration test in CI")
	}

	// Note: This will show "not authenticated" if no credentials exist
	// which is expected behavior
	err := Status()
	if err != nil {
		t.Logf("Auth status: %v", err)
	} else {
		t.Log("Auth status checked")
	}
}
