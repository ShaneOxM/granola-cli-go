package auth

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/ShaneOxM/granola-cli-go/internal/secrets"
)

const (
	serviceName = "com.granola.cli"
	accountName = "credentials"
)

func getDefaultClientID() string {
	if clientID := os.Getenv("GRANOLA_CLIENT_ID"); clientID != "" {
		return clientID
	}
	// Deprecated: hardcoded fallback will be removed in future version
	fmt.Fprintln(os.Stderr, "Warning: Using deprecated hardcoded client ID. Set GRANOLA_CLIENT_ID environment variable to suppress this warning.")
	return "client_GranolaMac"
}

// Credentials matches the legacy Node.js CLI structure
type Credentials struct {
	EmailAddress string `json:"email"`
	RefreshToken string `json:"refreshToken"`
	AccessToken  string `json:"accessToken"`
	ClientID     string `json:"clientId"`
}

// GetCredentials reads credentials from macOS Keychain
func GetCredentials() (*Credentials, error) {
	item, err := secrets.GetKeychainItem(serviceName, accountName)
	if err != nil {
		return nil, fmt.Errorf("failed to read keychain: %w", err)
	}

	var creds Credentials
	if err := json.Unmarshal(item.Data, &creds); err != nil {
		return nil, fmt.Errorf("failed to parse credentials: %w", err)
	}

	return &creds, nil
}

// SaveCredentials saves credentials to macOS Keychain
func SaveCredentials(creds *Credentials) error {
	data, err := json.Marshal(creds)
	if err != nil {
		return fmt.Errorf("failed to marshal credentials: %w", err)
	}

	return secrets.SetKeychainItem(serviceName, accountName, data)
}

// DeleteCredentials removes credentials from macOS Keychain
func DeleteCredentials() error {
	return secrets.DeleteKeychainItem(serviceName, accountName)
}

// getDefaultSupabasePath returns the platform-specific path to supabase.json
func getDefaultSupabasePath() string {
	home, _ := os.UserHomeDir()

	var path string
	switch runtime.GOOS {
	case "darwin":
		path = filepath.Join(home, "Library", "Application Support", "Granola", "supabase.json")
	case "windows":
		path = filepath.Join(os.Getenv("APPDATA"), "Granola", "supabase.json")
	default:
		path = filepath.Join(home, ".config", "granola", "supabase.json")
	}

	return path
}

// LoadCredentialsFromFile reads and parses credentials from supabase.json
func LoadCredentialsFromFile() (*Credentials, error) {
	path := getDefaultSupabasePath()

	content, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read %s: %w", path, err)
	}

	return parseSupabaseJSON(content)
}

// parseSupabaseJSON parses the supabase.json file content
// Handles WorkOS tokens (newer), Cognito tokens (older), and legacy format
func parseSupabaseJSON(content []byte) (*Credentials, error) {
	var raw map[string]interface{}
	if err := json.Unmarshal(content, &raw); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	var email string

	// Try WorkOS tokens first (newer auth system)
	if workosTokensRaw, ok := raw["workos_tokens"].(string); ok {
		var workosTokens map[string]interface{}
		if err := json.Unmarshal([]byte(workosTokensRaw), &workosTokens); err != nil {
			return nil, fmt.Errorf("failed to parse workos_tokens: %w", err)
		}

		if accessToken, ok := workosTokens["access_token"].(string); ok {
			email = extractEmailFromRaw(raw)
			clientID := getString(workosTokens, "client_id")
			if clientID == "" {
				clientID = getDefaultClientID()
			}
			return &Credentials{
				EmailAddress: email,
				RefreshToken: getString(workosTokens, "refresh_token"),
				AccessToken:  accessToken,
				ClientID:     clientID,
			}, nil
		}
	}

	// Fall back to Cognito tokens (older auth system)
	if cognitoTokensRaw, ok := raw["cognito_tokens"].(string); ok {
		var cognitoTokens map[string]interface{}
		if err := json.Unmarshal([]byte(cognitoTokensRaw), &cognitoTokens); err != nil {
			return nil, fmt.Errorf("failed to parse cognito_tokens: %w", err)
		}

		if refreshToken, ok := cognitoTokens["refresh_token"].(string); ok && refreshToken != "" {
			email = extractEmailFromRaw(raw)
			clientID := getString(cognitoTokens, "client_id")
			if clientID == "" {
				clientID = getDefaultClientID()
			}
			return &Credentials{
				EmailAddress: email,
				RefreshToken: refreshToken,
				AccessToken:  getString(cognitoTokens, "access_token"),
				ClientID:     clientID,
			}, nil
		}
	}

	// Legacy format: refresh_token at root level
	if refreshToken, ok := raw["refresh_token"].(string); ok && refreshToken != "" {
		email = extractEmailFromRaw(raw)
		clientID := getString(raw, "client_id")
		if clientID == "" {
			clientID = getDefaultClientID()
		}
		return &Credentials{
			EmailAddress: email,
			RefreshToken: refreshToken,
			AccessToken:  getString(raw, "access_token"),
			ClientID:     clientID,
		}, nil
	}

	return nil, fmt.Errorf("no valid tokens found in supabase.json")
}

// extractEmailFromRaw extracts email from user_info field
func extractEmailFromRaw(raw map[string]interface{}) string {
	if userInfoRaw, ok := raw["user_info"].(string); ok {
		var userInfo map[string]interface{}
		if err := json.Unmarshal([]byte(userInfoRaw), &userInfo); err == nil {
			if email, ok := userInfo["email"].(string); ok {
				return email
			}
		}
	}
	return "granola-user"
}

// getString safely extracts a string value from a map
func getString(m map[string]interface{}, key string) string {
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

// GetDefaultSupabasePath returns the path to supabase.json (exported for external use)
func GetDefaultSupabasePath() string {
	return getDefaultSupabasePath()
}

// Email returns the email address
func (c *Credentials) Email() string {
	return c.EmailAddress
}
