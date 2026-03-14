// Package auth provides Gmail and Calendar OAuth integration for granola-cli.
// It extends the existing auth package with Google API scopes and services.
package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/ShaneOxM/granola-cli-go/internal/config"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

var findDefaultCredentials = google.FindDefaultCredentials
var accountSanitizer = regexp.MustCompile(`[^a-zA-Z0-9_.-]+`)
var userInfoURL = "https://www.googleapis.com/oauth2/v2/userinfo"

// GmailCalendarAuth manages OAuth2 authentication for Gmail and Calendar APIs
type GmailCalendarAuth struct {
	mu     sync.Mutex
	config *oauth2.Config
	cache  *TokenCache
	scopes []string
	token  *oauth2.Token
}

// TokenCache manages OAuth2 token persistence
type TokenCache struct {
	path    string
	token   *oauth2.Token
	mu      sync.Mutex
	loaded  bool
	loadErr error
}

// NewGmailCalendarAuth creates a new Gmail/Calendar auth instance
func NewGmailCalendarAuth() (*GmailCalendarAuth, error) {
	// Gmail and Calendar scopes (full URLs)
	scopes := []string{
		"https://www.googleapis.com/auth/gmail.readonly",
		"https://www.googleapis.com/auth/calendar.readonly",
	}

	clientID := os.Getenv("GRANOLA_OAUTH_CLIENT_ID")
	clientSecret := os.Getenv("GRANOLA_OAUTH_CLIENT_SECRET")

	if clientID == "" || clientSecret == "" {
		if cfg, err := config.Read(); err == nil {
			if clientID == "" {
				clientID = cfg.GoogleClientID
			}
			if clientSecret == "" {
				clientSecret = cfg.GoogleClientSecret
			}
		}
	}

	var oauthConfig *oauth2.Config
	if clientID != "" && clientSecret != "" {
		oauthConfig = &oauth2.Config{
			ClientID:     clientID,
			ClientSecret: clientSecret,
			Scopes:       scopes,
			Endpoint:     google.Endpoint,
			RedirectURL:  "urn:ietf:wg:oauth:2.0:oob",
		}
	}

	// Get user config path
	userConfig, err := os.UserConfigDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get user config dir: %w", err)
	}

	cachePath := filepath.Join(userConfig, "granola-cli", "oauth_cache.json")
	if cfg, err := config.Read(); err == nil && strings.TrimSpace(cfg.GoogleActiveAccount) != "" {
		acctPath := cachePathForAccount(userConfig, cfg.GoogleActiveAccount)
		if _, statErr := os.Stat(acctPath); statErr == nil {
			cachePath = acctPath
		}
	}

	return &GmailCalendarAuth{
		config: oauthConfig,
		cache:  &TokenCache{path: cachePath},
		scopes: scopes,
	}, nil
}

func cachePathForAccount(userConfigDir, account string) string {
	safe := strings.ToLower(strings.TrimSpace(account))
	safe = accountSanitizer.ReplaceAllString(safe, "_")
	if safe == "" {
		safe = "default"
	}
	return filepath.Join(userConfigDir, "granola-cli", fmt.Sprintf("oauth_cache_%s.json", safe))
}

func (a *GmailCalendarAuth) CachePath() string {
	return a.cache.path
}

func (a *GmailCalendarAuth) SetCachePath(path string) {
	a.cache.path = path
	a.cache.loaded = false
	a.cache.loadErr = nil
}

func (a *GmailCalendarAuth) SetRedirectURL(redirectURL string) {
	if a.config != nil {
		a.config.RedirectURL = redirectURL
	}
}

// ActivateAccount switches token cache to an account-specific cache file.
// If a default cache exists, it is copied to the account cache path.
func (a *GmailCalendarAuth) ActivateAccount(email string) error {
	userConfig, err := os.UserConfigDir()
	if err != nil {
		return err
	}
	target := cachePathForAccount(userConfig, email)
	if target == a.cache.path {
		return nil
	}
	if data, err := os.ReadFile(a.cache.path); err == nil && len(data) > 0 {
		dir := filepath.Dir(target)
		if err := os.MkdirAll(dir, 0700); err != nil {
			return err
		}
		if err := os.WriteFile(target, data, 0600); err != nil {
			return err
		}
	}
	a.SetCachePath(target)
	return nil
}

// GetToken returns an OAuth2 token, refreshing if necessary
func (a *GmailCalendarAuth) GetToken(ctx context.Context) (*oauth2.Token, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	authMode := ""
	if cfg, err := config.Read(); err == nil {
		authMode = strings.TrimSpace(cfg.GoogleAuthMode)
		if active := strings.TrimSpace(cfg.GoogleActiveAccount); active != "" && cfg.GoogleAccounts != nil {
			if mode := strings.TrimSpace(cfg.GoogleAccounts[active]); mode != "" {
				authMode = mode
			}
		}
	}
	if authMode == "adc" && a.config != nil {
		if _, err := os.Stat(a.cache.path); err == nil {
			authMode = "oauth"
		}
	}

	if envToken := os.Getenv("GOOGLE_WORKSPACE_CLI_TOKEN"); envToken != "" {
		return &oauth2.Token{AccessToken: envToken}, nil
	}

	// Load cached token if available
	if a.cache.token != nil {
		// Check if token is expired
		if !a.cache.token.Expiry.Before(time.Now()) {
			return a.cache.token, nil
		}

		if a.config != nil {
			// Refresh expired token when OAuth client config exists
			newToken, err := a.config.TokenSource(ctx, a.cache.token).Token()
			if err != nil {
				return nil, fmt.Errorf("failed to refresh token: %w", err)
			}

			a.cache.token = newToken
			if err := a.cache.save(); err != nil {
				return nil, fmt.Errorf("failed to save token: %w", err)
			}
			return newToken, nil
		}
		// Fall through to ADC when no refresh config is available.
	}

	// Try to load from cache file
	if err := a.cache.load(); err != nil {
		return nil, fmt.Errorf("failed to load cached token: %w", err)
	}

	if a.cache.token != nil {
		if !a.cache.token.Expiry.Before(time.Now()) {
			return a.cache.token, nil
		}

		if a.config != nil {
			newToken, err := a.config.TokenSource(ctx, a.cache.token).Token()
			if err != nil {
				return nil, fmt.Errorf("failed to refresh token: %w", err)
			}

			a.cache.token = newToken
			if err := a.cache.save(); err != nil {
				return nil, fmt.Errorf("failed to save token: %w", err)
			}
			return newToken, nil
		}
	}

	if authMode == "oauth" {
		return nil, fmt.Errorf("no valid cached oauth token available")
	}

	creds, err := findDefaultCredentials(ctx, a.scopes...)
	if err != nil {
		return nil, fmt.Errorf("no valid token available: %w", err)
	}
	tok, err := creds.TokenSource.Token()
	if err != nil {
		return nil, fmt.Errorf("failed to get ADC token: %w", err)
	}
	return tok, nil
}

// HTTPClient returns an authenticated HTTP client for Google APIs.
func (a *GmailCalendarAuth) HTTPClient(ctx context.Context) (*http.Client, error) {
	token, err := a.GetToken(ctx)
	if err != nil {
		return nil, err
	}
	return a.httpClient(ctx, token), nil
}

// QuotaProject returns ADC quota_project_id when available.
func (a *GmailCalendarAuth) QuotaProject() string {
	if cfg, err := config.Read(); err == nil {
		mode := strings.TrimSpace(cfg.GoogleAuthMode)
		if active := strings.TrimSpace(cfg.GoogleActiveAccount); active != "" && cfg.GoogleAccounts != nil {
			if acctMode := strings.TrimSpace(cfg.GoogleAccounts[active]); acctMode != "" {
				mode = acctMode
			}
		}
		if mode == "adc" && a.config != nil {
			if _, err := os.Stat(a.cache.path); err == nil {
				mode = "oauth"
			}
		}
		if mode != "adc" {
			return ""
		}
	}
	path := os.Getenv("GOOGLE_APPLICATION_CREDENTIALS")
	if path == "" {
		if home, err := os.UserHomeDir(); err == nil {
			path = filepath.Join(home, ".config", "gcloud", "application_default_credentials.json")
		}
	}
	if path == "" {
		return ""
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	var v struct {
		QuotaProjectID string `json:"quota_project_id"`
	}
	if err := json.Unmarshal(data, &v); err != nil {
		return ""
	}
	return strings.TrimSpace(v.QuotaProjectID)
}

// CanInteractiveLogin reports whether interactive OAuth login can run.
func (a *GmailCalendarAuth) CanInteractiveLogin() bool {
	return a.config != nil
}

// AuthCodeURL returns an OAuth authorization URL for interactive login.
func (a *GmailCalendarAuth) AuthCodeURL(state string) (string, error) {
	if a.config == nil {
		return "", fmt.Errorf("interactive login requires GRANOLA_OAUTH_CLIENT_ID and GRANOLA_OAUTH_CLIENT_SECRET")
	}
	return a.config.AuthCodeURL(state, oauth2.AccessTypeOffline), nil
}

// ExchangeCode exchanges an OAuth authorization code and persists the token cache.
func (a *GmailCalendarAuth) ExchangeCode(ctx context.Context, code string) error {
	if a.config == nil {
		return fmt.Errorf("interactive login requires GRANOLA_OAUTH_CLIENT_ID and GRANOLA_OAUTH_CLIENT_SECRET")
	}
	code = strings.TrimSpace(code)
	if code == "" {
		return fmt.Errorf("authorization code is required")
	}
	tok, err := a.config.Exchange(ctx, code)
	if err != nil {
		return fmt.Errorf("oauth code exchange failed: %w", err)
	}
	a.cache.token = tok
	a.cache.loaded = true
	a.cache.loadErr = nil
	if err := a.cache.save(); err != nil {
		return fmt.Errorf("failed saving oauth token cache: %w", err)
	}
	return nil
}

// CurrentEmail fetches the authenticated account email from Google userinfo.
func (a *GmailCalendarAuth) CurrentEmail(ctx context.Context) (string, error) {
	hc, err := a.HTTPClient(ctx)
	if err != nil {
		return "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, userInfoURL, nil)
	if err != nil {
		return "", err
	}
	resp, err := hc.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("userinfo http %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	var data struct {
		Email string `json:"email"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return "", err
	}
	if strings.TrimSpace(data.Email) == "" {
		return "", fmt.Errorf("email missing in userinfo")
	}
	return data.Email, nil
}

func (a *GmailCalendarAuth) httpClient(ctx context.Context, token *oauth2.Token) *http.Client {
	if a.config != nil {
		return a.config.Client(ctx, token)
	}
	return oauth2.NewClient(ctx, oauth2.StaticTokenSource(token))
}

// Load loads token from cache file
func (c *TokenCache) load() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.loaded {
		return c.loadErr
	}

	data, err := os.ReadFile(c.path)
	if err != nil {
		if os.IsNotExist(err) {
			c.loaded = true
			return nil
		}
		c.loadErr = err
		c.loaded = true
		return err
	}

	var token oauth2.Token
	if err := json.Unmarshal(data, &token); err != nil {
		c.loadErr = err
		c.loaded = true
		return err
	}

	c.token = &token
	c.loaded = true
	return nil
}

// save saves token to cache file
func (c *TokenCache) save() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Ensure directory exists
	dir := filepath.Dir(c.path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("failed to create cache directory: %w", err)
	}

	data, err := json.Marshal(c.token)
	if err != nil {
		return fmt.Errorf("failed to marshal token: %w", err)
	}

	if err := os.WriteFile(c.path, data, 0600); err != nil {
		return fmt.Errorf("failed to save token: %w", err)
	}

	return nil
}

// ClearCache clears the token cache
func (a *GmailCalendarAuth) ClearCache() error {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.cache.token = nil
	a.cache.loaded = false
	a.cache.loadErr = nil
	if err := os.Remove(a.cache.path); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

// InvalidateToken removes the cached token
func (a *GmailCalendarAuth) InvalidateToken() error {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.cache.token = nil

	if err := os.Remove(a.cache.path); err != nil && !os.IsNotExist(err) {
		return err
	}

	return nil
}
