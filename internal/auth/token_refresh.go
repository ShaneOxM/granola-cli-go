package auth

import (
	"bytes"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

const (
	lockFileName      = "granola_token_refresh.lock"
	maxRefreshRetries = 3
)

var workOSAuthURL = "https://api.workos.com/user_management/authenticate"

// refreshRequest matches the WorkOS refresh token request format
type refreshRequest struct {
	ClientID     string `json:"client_id"`
	GrantType    string `json:"grant_type"`
	RefreshToken string `json:"refresh_token"`
}

// refreshResponse matches the WorkOS refresh token response format
type refreshResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
	TokenType    string `json:"token_type"`
}

// RefreshAccessToken refreshes the access token using the stored refresh token
// Uses file-based locking to prevent race conditions when multiple CLI processes
// attempt to refresh simultaneously
func RefreshAccessToken() (*Credentials, error) {
	// Try multiple times with exponential backoff
	var lastErr error
	for attempt := 0; attempt < maxRefreshRetries; attempt++ {
		creds, err := tryRefreshWithLock()
		if err == nil {
			return creds, nil
		}
		lastErr = err

		// Wait before retry (exponential backoff: 250ms, 500ms, 1s)
		time.Sleep(time.Duration(250*(1<<uint(attempt))) * time.Millisecond)
	}

	return nil, fmt.Errorf("failed to refresh token after %d attempts: %w", maxRefreshRetries, lastErr)
}

// tryRefreshWithLock acquires a file lock and performs token refresh
func tryRefreshWithLock() (*Credentials, error) {
	// Read current credentials
	creds, err := GetCredentials()
	if err != nil {
		return nil, fmt.Errorf("failed to read credentials: %w", err)
	}

	if creds.RefreshToken == "" || creds.ClientID == "" {
		return nil, fmt.Errorf("cannot refresh: missing refreshToken or clientId")
	}

	// Create lock file path in temp directory
	lockPath, err := getLockPath()
	if err != nil {
		return nil, fmt.Errorf("failed to get lock path: %w", err)
	}

	// Acquire file lock with PID tracking
	lockFile, err := os.OpenFile(lockPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return nil, fmt.Errorf("failed to acquire lock (another process may be refreshing): %w", err)
	}

	// Write PID to lock file for debugging and stale lock detection
	fmt.Fprintf(lockFile, "%d\n", os.Getpid())
	lockFile.Sync()

	defer lockFile.Close()

	// Re-read credentials inside the lock - another process may have updated them
	creds, err = GetCredentials()
	if err != nil || creds.RefreshToken == "" || creds.ClientID == "" {
		return nil, fmt.Errorf("cannot refresh: missing refreshToken or clientId after lock")
	}

	// Prepare refresh request
	reqBody := refreshRequest{
		ClientID:     creds.ClientID,
		GrantType:    "refresh_token",
		RefreshToken: creds.RefreshToken,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal refresh request: %w", err)
	}

	// Call WorkOS API
	resp, err := callWorkOSRefresh(jsonBody)
	if err != nil {
		return nil, fmt.Errorf("WorkOS refresh failed: %w", err)
	}

	// Create new credentials
	newCreds := &Credentials{
		RefreshToken: resp.RefreshToken,
		AccessToken:  resp.AccessToken,
		ClientID:     creds.ClientID,
	}

	// Save new credentials immediately (WorkOS tokens are single-use)
	if err := SaveCredentials(newCreds); err != nil {
		return nil, fmt.Errorf("failed to save new credentials: %w", err)
	}

	return newCreds, nil
}

// callWorkOSRefresh makes the HTTP request to WorkOS
func callWorkOSRefresh(body []byte) (*refreshResponse, error) {
	resp, err := http.Post(workOSAuthURL, "application/json", bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		buf := new(bytes.Buffer)
		buf.ReadFrom(resp.Body)
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, buf.String())
	}

	var respData refreshResponse
	if err := json.NewDecoder(resp.Body).Decode(&respData); err != nil {
		return nil, fmt.Errorf("failed to parse WorkOS response: %w", err)
	}

	if respData.AccessToken == "" {
		return nil, fmt.Errorf("WorkOS response missing access_token")
	}

	return &respData, nil
}

// getLockPath returns a unique lock file path in the temp directory
func getLockPath() (string, error) {
	tempDir := os.TempDir()
	if tempDir == "" {
		tempDir = os.Getenv("TEMP")
		if tempDir == "" {
			tempDir = os.Getenv("TMP")
		}
	}

	if tempDir == "" {
		return "", fmt.Errorf("could not determine temp directory")
	}

	// Generate unique lock file name with PID and random bytes
	pid := os.Getpid()
	random := make([]byte, 8)
	if _, err := rand.Read(random); err != nil {
		return "", fmt.Errorf("failed to generate random bytes: %w", err)
	}
	hash := fmt.Sprintf("%d-%x", pid, random)

	return filepath.Join(tempDir, fmt.Sprintf(".granola_lock_%s", hash)), nil
}
