// Package auth provides authentication management for granola-cli.
// It handles OAuth2 flows, token refresh, and secure credential storage.
package auth

import (
	"fmt"
	"os"
	"sync"

	"github.com/ShaneOxM/granola-cli-go/internal/logger"
	"github.com/ShaneOxM/granola-cli-go/internal/secrets"
)

// Auth encapsulates authentication state and operations
// This struct allows for dependency injection in tests and advanced usage
type Auth struct {
	store     secrets.Store
	cacheOnce sync.Once
	cache     map[string]string
	cacheErr  error
}

var (
	getCredentialsFn          = GetCredentials
	loadCredentialsFromFileFn = LoadCredentialsFromFile
	saveCredentialsFn         = SaveCredentials
	deleteCredentialsFn       = DeleteCredentials
)

// NewAuth creates a new Auth instance with initialized store
// This is the preferred way to create an Auth instance for testing
func NewAuth() (*Auth, error) {
	s, err := secrets.NewStore()
	if err != nil {
		return nil, err
	}

	a := &Auth{
		store: s,
		cache: make(map[string]string),
	}

	return a, nil
}

// GetAuthEnvVars returns environment variables for granola CLI authentication
// Credentials are cached after first retrieval to avoid repeated keychain prompts
func (a *Auth) GetAuthEnvVars() (map[string]string, error) {
	a.cacheOnce.Do(func() {
		env := make(map[string]string)

		// Get credentials from keychain (only happens once per process)
		creds, keychainErr := getCredentialsFn()
		if keychainErr != nil {
			a.cacheErr = fmt.Errorf("no authentication available: %w", keychainErr)
			return
		}

		// Set environment variables
		env["GRANOLA_ACCESS_TOKEN"] = creds.AccessToken
		env["GRANOLA_REFRESH_TOKEN"] = creds.RefreshToken
		env["GRANOLA_CLIENT_ID"] = creds.ClientID
		env["GRANOLA_AUTHED"] = "true"

		a.cache = env
		a.cacheErr = nil
	})

	if a.cacheErr != nil {
		return nil, a.cacheErr
	}

	return a.cache, nil
}

// Status displays current authentication status
func (a *Auth) Status() error {
	creds, err := getCredentialsFn()
	if err != nil {
		logger.Error("Failed to get credentials", "error", err)
		fmt.Println("\nTo authenticate, run: granola auth login")
		return nil
	}

	fmt.Println("Authenticated account:")
	fmt.Printf("  - Email: %s\n", creds.EmailAddress)
	fmt.Printf("  - Client ID: %s\n", creds.ClientID)
	fmt.Printf("  - Has Access Token: %v\n", creds.AccessToken != "")
	fmt.Printf("  - Has Refresh Token: %v\n", creds.RefreshToken != "")

	return nil
}

// Login imports credentials from the Granola desktop app's supabase.json file
func (a *Auth) Login(args []string) error {
	fmt.Println("Importing credentials from Granola desktop app...")

	creds, err := loadCredentialsFromFileFn()
	if err != nil {
		path := getDefaultSupabasePath()
		logger.Error("Could not load credentials", "path", path, "error", err)
		fmt.Fprintf(os.Stderr, "Make sure the Granola desktop app is installed and you are logged in.\n")
		return err
	}

	fmt.Printf("Found credentials:\n")
	fmt.Printf("  - Email: %s\n", creds.EmailAddress)
	fmt.Printf("  - Client ID: %s\n", creds.ClientID)
	fmt.Printf("  - Has Access Token: %v\n", creds.AccessToken != "")
	fmt.Printf("  - Has Refresh Token: %v\n", creds.RefreshToken != "")

	// Save to keychain
	if err := saveCredentialsFn(creds); err != nil {
		logger.Error("Failed to save credentials", "error", err)
		return err
	}

	fmt.Println("\n✓ Credentials imported successfully")
	fmt.Println("You can now use granola commands that require authentication.")
	return nil
}

// Logout removes credentials from the keychain
func (a *Auth) Logout(args []string) error {
	if err := deleteCredentialsFn(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: Failed to logout: %v\n", err)
		return err
	}

	fmt.Println("✓ Logged out successfully")
	return nil
}

// defaultAuth is the global auth instance for backward compatibility
var defaultAuth *Auth

// Init initializes the default Auth instance
// This is called automatically by the main package
func Init() error {
	a, err := NewAuth()
	if err != nil {
		return err
	}
	defaultAuth = a
	return nil
}

// GetAuthEnvVars returns environment variables (global wrapper for backward compatibility)
func GetAuthEnvVars() (map[string]string, error) {
	if defaultAuth == nil {
		return nil, fmt.Errorf("auth not initialized")
	}
	return defaultAuth.GetAuthEnvVars()
}

// Status displays auth status (global wrapper for backward compatibility)
func Status() error {
	if defaultAuth == nil {
		return fmt.Errorf("auth not initialized")
	}
	return defaultAuth.Status()
}

// Login imports credentials (global wrapper for backward compatibility)
func Login(args []string) error {
	if defaultAuth == nil {
		return fmt.Errorf("auth not initialized")
	}
	return defaultAuth.Login(args)
}

// Logout removes credentials (global wrapper for backward compatibility)
func Logout(args []string) error {
	if defaultAuth == nil {
		return fmt.Errorf("auth not initialized")
	}
	return defaultAuth.Logout(args)
}
