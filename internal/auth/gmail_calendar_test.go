package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ShaneOxM/granola-cli-go/internal/config"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

func useTempConfigPath(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.json")
	old := os.Getenv("GRANOLA_CONFIG_PATH")
	t.Cleanup(func() { _ = os.Setenv("GRANOLA_CONFIG_PATH", old) })
	_ = os.Setenv("GRANOLA_CONFIG_PATH", path)
	if err := config.Init(); err != nil {
		t.Fatalf("config init failed: %v", err)
	}
	return path
}

func TestTokenCacheLoadSave(t *testing.T) {
	dir := t.TempDir()
	cache := &TokenCache{path: filepath.Join(dir, "oauth_cache.json")}
	cache.token = &oauth2.Token{AccessToken: "abc", RefreshToken: "ref", Expiry: time.Now().Add(time.Hour)}
	if err := cache.save(); err != nil {
		t.Fatalf("save failed: %v", err)
	}

	loaded := &TokenCache{path: cache.path}
	if err := loaded.load(); err != nil {
		t.Fatalf("load failed: %v", err)
	}
	if loaded.token == nil || loaded.token.AccessToken != "abc" {
		t.Fatalf("unexpected token after load")
	}
}

func TestTokenCacheLoadMissingFile(t *testing.T) {
	cache := &TokenCache{path: filepath.Join(t.TempDir(), "missing.json")}
	if err := cache.load(); err != nil {
		t.Fatalf("expected nil error for missing file, got %v", err)
	}
}

func TestNewGmailCalendarAuthWithoutEnv(t *testing.T) {
	oldID := os.Getenv("GRANOLA_OAUTH_CLIENT_ID")
	oldSecret := os.Getenv("GRANOLA_OAUTH_CLIENT_SECRET")
	defer func() {
		_ = os.Setenv("GRANOLA_OAUTH_CLIENT_ID", oldID)
		_ = os.Setenv("GRANOLA_OAUTH_CLIENT_SECRET", oldSecret)
	}()
	_ = os.Unsetenv("GRANOLA_OAUTH_CLIENT_ID")
	_ = os.Unsetenv("GRANOLA_OAUTH_CLIENT_SECRET")

	a, err := NewGmailCalendarAuth()
	if err != nil {
		t.Fatalf("expected auth constructor to succeed without env, got %v", err)
	}
	if len(a.scopes) != 2 {
		t.Fatalf("expected default scopes to be set")
	}
}

func TestTokenCacheFileFormat(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "oauth_cache.json")
	cache := &TokenCache{path: path, token: &oauth2.Token{AccessToken: "tok"}}
	if err := cache.save(); err != nil {
		t.Fatalf("save failed: %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read failed: %v", err)
	}
	var payload map[string]any
	if err := json.Unmarshal(data, &payload); err != nil {
		t.Fatalf("invalid json: %v", err)
	}
	if payload["access_token"] == nil {
		t.Fatalf("missing access_token field")
	}
}

func TestInvalidateTokenRemovesCacheFile(t *testing.T) {
	oldID := os.Getenv("GRANOLA_OAUTH_CLIENT_ID")
	oldSecret := os.Getenv("GRANOLA_OAUTH_CLIENT_SECRET")
	defer func() {
		_ = os.Setenv("GRANOLA_OAUTH_CLIENT_ID", oldID)
		_ = os.Setenv("GRANOLA_OAUTH_CLIENT_SECRET", oldSecret)
	}()
	_ = os.Setenv("GRANOLA_OAUTH_CLIENT_ID", "id")
	_ = os.Setenv("GRANOLA_OAUTH_CLIENT_SECRET", "secret")

	a, err := NewGmailCalendarAuth()
	if err != nil {
		t.Fatalf("new auth failed: %v", err)
	}
	a.cache.path = filepath.Join(t.TempDir(), "oauth_cache.json")
	a.cache.token = &oauth2.Token{AccessToken: "tok", Expiry: time.Now().Add(time.Hour)}
	if err := a.cache.save(); err != nil {
		t.Fatalf("save failed: %v", err)
	}
	if err := a.InvalidateToken(); err != nil {
		t.Fatalf("invalidate failed: %v", err)
	}
	if _, err := os.Stat(a.cache.path); err == nil {
		t.Fatalf("expected cache file to be removed")
	}
}

func TestGetTokenFromEnv(t *testing.T) {
	old := os.Getenv("GOOGLE_WORKSPACE_CLI_TOKEN")
	defer func() { _ = os.Setenv("GOOGLE_WORKSPACE_CLI_TOKEN", old) }()
	_ = os.Setenv("GOOGLE_WORKSPACE_CLI_TOKEN", "env-token")

	a, err := NewGmailCalendarAuth()
	if err != nil {
		t.Fatalf("new auth failed: %v", err)
	}
	tok, err := a.GetToken(t.Context())
	if err != nil {
		t.Fatalf("get token failed: %v", err)
	}
	if tok.AccessToken != "env-token" {
		t.Fatalf("unexpected env token")
	}
}

func TestClearCache(t *testing.T) {
	a, err := NewGmailCalendarAuth()
	if err != nil {
		t.Fatalf("new auth failed: %v", err)
	}
	a.cache.path = filepath.Join(t.TempDir(), "oauth_cache.json")
	a.cache.token = &oauth2.Token{AccessToken: "tok"}
	if err := a.cache.save(); err != nil {
		t.Fatalf("save failed: %v", err)
	}
	if err := a.ClearCache(); err != nil {
		t.Fatalf("clear cache failed: %v", err)
	}
	if _, err := os.Stat(a.cache.path); err == nil {
		t.Fatalf("expected cache file to be removed")
	}
}

func TestHTTPClientWithEnvToken(t *testing.T) {
	old := os.Getenv("GOOGLE_WORKSPACE_CLI_TOKEN")
	defer func() { _ = os.Setenv("GOOGLE_WORKSPACE_CLI_TOKEN", old) }()
	_ = os.Setenv("GOOGLE_WORKSPACE_CLI_TOKEN", "env-token")

	a, err := NewGmailCalendarAuth()
	if err != nil {
		t.Fatalf("new auth failed: %v", err)
	}
	hc, err := a.HTTPClient(t.Context())
	if err != nil {
		t.Fatalf("http client failed: %v", err)
	}
	if hc == nil {
		t.Fatalf("http client is nil")
	}
}

func TestGetTokenFromUnexpiredCache(t *testing.T) {
	a, err := NewGmailCalendarAuth()
	if err != nil {
		t.Fatalf("new auth failed: %v", err)
	}
	a.cache.token = &oauth2.Token{AccessToken: "cached", Expiry: time.Now().Add(time.Hour)}
	tok, err := a.GetToken(t.Context())
	if err != nil {
		t.Fatalf("get token failed: %v", err)
	}
	if tok.AccessToken != "cached" {
		t.Fatalf("expected cached token")
	}
}

func TestGetTokenRefreshWithConfig(t *testing.T) {
	tokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"new-token","token_type":"Bearer","expires_in":3600}`))
	}))
	defer tokenServer.Close()

	a := &GmailCalendarAuth{
		config: &oauth2.Config{
			ClientID:     "id",
			ClientSecret: "secret",
			Endpoint: oauth2.Endpoint{
				TokenURL: tokenServer.URL,
			},
		},
		cache: &TokenCache{path: filepath.Join(t.TempDir(), "oauth_cache.json")},
	}
	a.cache.token = &oauth2.Token{
		AccessToken:  "old-token",
		RefreshToken: "refresh-token",
		Expiry:       time.Now().Add(-time.Hour),
	}

	tok, err := a.GetToken(context.Background())
	if err != nil {
		t.Fatalf("expected refresh to succeed, got %v", err)
	}
	if tok.AccessToken != "new-token" {
		t.Fatalf("expected refreshed token")
	}
}

func TestGetTokenLoadsFromCacheFile(t *testing.T) {
	a := &GmailCalendarAuth{cache: &TokenCache{path: filepath.Join(t.TempDir(), "oauth_cache.json")}}
	a.cache.token = &oauth2.Token{AccessToken: "cached-file", Expiry: time.Now().Add(time.Hour)}
	if err := a.cache.save(); err != nil {
		t.Fatalf("save failed: %v", err)
	}
	a.cache.token = nil
	tok, err := a.GetToken(context.Background())
	if err != nil {
		t.Fatalf("expected cache load success, got %v", err)
	}
	if tok.AccessToken != "cached-file" {
		t.Fatalf("expected cached token from file")
	}
}

func TestGetTokenFallsBackToADC(t *testing.T) {
	_ = useTempConfigPath(t)
	cfg, _ := config.Read()
	cfg.GoogleAuthMode = "adc"
	cfg.GoogleActiveAccount = ""
	cfg.GoogleAccounts = nil
	_ = config.Write(cfg)

	orig := findDefaultCredentials
	defer func() { findDefaultCredentials = orig }()
	findDefaultCredentials = func(ctx context.Context, scopes ...string) (*google.Credentials, error) {
		return &google.Credentials{TokenSource: oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "adc-token"})}, nil
	}

	a, err := NewGmailCalendarAuth()
	if err != nil {
		t.Fatalf("new auth failed: %v", err)
	}
	a.cache.path = filepath.Join(t.TempDir(), "none.json")
	a.cache.token = nil
	_ = os.Unsetenv("GOOGLE_WORKSPACE_CLI_TOKEN")
	tok, err := a.GetToken(context.Background())
	if err != nil {
		t.Fatalf("expected ADC token, got %v", err)
	}
	if tok.AccessToken != "adc-token" {
		t.Fatalf("unexpected ADC token")
	}
}

func TestGetTokenADCFailure(t *testing.T) {
	_ = useTempConfigPath(t)
	cfg, _ := config.Read()
	cfg.GoogleAuthMode = "adc"
	cfg.GoogleActiveAccount = ""
	cfg.GoogleAccounts = nil
	_ = config.Write(cfg)

	orig := findDefaultCredentials
	defer func() { findDefaultCredentials = orig }()
	findDefaultCredentials = func(ctx context.Context, scopes ...string) (*google.Credentials, error) {
		return nil, fmt.Errorf("adc unavailable")
	}

	a, err := NewGmailCalendarAuth()
	if err != nil {
		t.Fatalf("new auth failed: %v", err)
	}
	a.cache.path = filepath.Join(t.TempDir(), "none.json")
	a.cache.token = nil
	_ = os.Unsetenv("GOOGLE_WORKSPACE_CLI_TOKEN")
	_, err = a.GetToken(context.Background())
	if err == nil {
		t.Fatalf("expected ADC failure")
	}
}

func TestGetTokenOAuthModeDoesNotFallBackToADC(t *testing.T) {
	_ = useTempConfigPath(t)
	cfg, _ := config.Read()
	cfg.GoogleAuthMode = "oauth"
	_ = config.Write(cfg)

	a, err := NewGmailCalendarAuth()
	if err != nil {
		t.Fatalf("new auth failed: %v", err)
	}
	a.cache.path = filepath.Join(t.TempDir(), "missing.json")
	a.cache.token = nil
	_, err = a.GetToken(context.Background())
	if err == nil || !strings.Contains(err.Error(), "cached oauth token") {
		t.Fatalf("expected oauth-mode cache error, got %v", err)
	}
}

func TestTokenCacheLoadInvalidJSON(t *testing.T) {
	path := filepath.Join(t.TempDir(), "oauth_cache.json")
	if err := os.WriteFile(path, []byte("{invalid-json"), 0600); err != nil {
		t.Fatalf("write failed: %v", err)
	}
	cache := &TokenCache{path: path}
	if err := cache.load(); err == nil {
		t.Fatalf("expected json decode error")
	}
}

func TestTokenCacheSaveDirectoryError(t *testing.T) {
	base := filepath.Join(t.TempDir(), "not-a-dir")
	if err := os.WriteFile(base, []byte("x"), 0600); err != nil {
		t.Fatalf("write file failed: %v", err)
	}
	cache := &TokenCache{
		path:  filepath.Join(base, "oauth_cache.json"),
		token: &oauth2.Token{AccessToken: "tok"},
	}
	if err := cache.save(); err == nil {
		t.Fatalf("expected directory creation error")
	}
}

func TestAuthCodeURLRequiresConfig(t *testing.T) {
	a, err := NewGmailCalendarAuth()
	if err != nil {
		t.Fatalf("new auth failed: %v", err)
	}
	a.config = nil
	_, err = a.AuthCodeURL("state")
	if err == nil {
		t.Fatalf("expected error when config missing")
	}
}

func TestAuthCodeURLSuccess(t *testing.T) {
	a, err := NewGmailCalendarAuth()
	if err != nil {
		t.Fatalf("new auth failed: %v", err)
	}
	a.config = &oauth2.Config{ClientID: "id", ClientSecret: "secret", Endpoint: oauth2.Endpoint{AuthURL: "https://example.com/auth"}}
	u, err := a.AuthCodeURL("abc")
	if err != nil {
		t.Fatalf("auth url failed: %v", err)
	}
	if !strings.Contains(u, "state=abc") {
		t.Fatalf("expected state in auth URL")
	}
}

func TestExchangeCodeRequiresConfig(t *testing.T) {
	a, err := NewGmailCalendarAuth()
	if err != nil {
		t.Fatalf("new auth failed: %v", err)
	}
	a.config = nil
	if err := a.ExchangeCode(context.Background(), "code"); err == nil {
		t.Fatalf("expected config error")
	}
}

func TestExchangeCodeEmpty(t *testing.T) {
	a, err := NewGmailCalendarAuth()
	if err != nil {
		t.Fatalf("new auth failed: %v", err)
	}
	a.config = &oauth2.Config{ClientID: "id", ClientSecret: "secret", Endpoint: oauth2.Endpoint{TokenURL: "https://example.com/token"}}
	if err := a.ExchangeCode(context.Background(), "   "); err == nil {
		t.Fatalf("expected empty code error")
	}
}

func TestExchangeCodeSuccess(t *testing.T) {
	tokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"code-token","token_type":"Bearer","expires_in":3600}`))
	}))
	defer tokenServer.Close()

	a, err := NewGmailCalendarAuth()
	if err != nil {
		t.Fatalf("new auth failed: %v", err)
	}
	a.config = &oauth2.Config{ClientID: "id", ClientSecret: "secret", Endpoint: oauth2.Endpoint{TokenURL: tokenServer.URL}}
	a.cache.path = filepath.Join(t.TempDir(), "oauth_cache.json")
	if err := a.ExchangeCode(context.Background(), "valid-code"); err != nil {
		t.Fatalf("exchange failed: %v", err)
	}
	if a.cache.token == nil || a.cache.token.AccessToken != "code-token" {
		t.Fatalf("expected saved exchanged token")
	}
}

func TestQuotaProjectFromADCFile(t *testing.T) {
	_ = useTempConfigPath(t)
	cfg, _ := config.Read()
	cfg.GoogleAuthMode = "adc"
	cfg.GoogleActiveAccount = ""
	cfg.GoogleAccounts = nil
	_ = config.Write(cfg)

	adcPath := filepath.Join(t.TempDir(), "application_default_credentials.json")
	_ = os.WriteFile(adcPath, []byte(`{"quota_project_id":"test-project"}`), 0600)
	old := os.Getenv("GOOGLE_APPLICATION_CREDENTIALS")
	defer func() { _ = os.Setenv("GOOGLE_APPLICATION_CREDENTIALS", old) }()
	_ = os.Setenv("GOOGLE_APPLICATION_CREDENTIALS", adcPath)

	a, err := NewGmailCalendarAuth()
	if err != nil {
		t.Fatalf("new auth failed: %v", err)
	}
	if got := a.QuotaProject(); got != "test-project" {
		t.Fatalf("expected quota project, got %q", got)
	}
}

func TestSetRedirectURLAndCanInteractiveLogin(t *testing.T) {
	a, err := NewGmailCalendarAuth()
	if err != nil {
		t.Fatalf("new auth failed: %v", err)
	}
	a.config = &oauth2.Config{}
	if !a.CanInteractiveLogin() {
		t.Fatalf("expected interactive login enabled")
	}
	a.SetRedirectURL("http://localhost:9999")
	if a.config.RedirectURL != "http://localhost:9999" {
		t.Fatalf("redirect URL not updated")
	}
}

func TestCurrentEmail(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"email":"test@example.com"}`))
	}))
	defer server.Close()
	origURL := userInfoURL
	defer func() { userInfoURL = origURL }()
	userInfoURL = server.URL
	oldTok := os.Getenv("GOOGLE_WORKSPACE_CLI_TOKEN")
	defer func() { _ = os.Setenv("GOOGLE_WORKSPACE_CLI_TOKEN", oldTok) }()
	_ = os.Setenv("GOOGLE_WORKSPACE_CLI_TOKEN", "env-token")
	a, err := NewGmailCalendarAuth()
	if err != nil {
		t.Fatalf("new auth failed: %v", err)
	}
	email, err := a.CurrentEmail(context.Background())
	if err != nil || email != "test@example.com" {
		t.Fatalf("unexpected email %q err=%v", email, err)
	}
}

func TestActivateAccountCopiesCacheFile(t *testing.T) {
	a, err := NewGmailCalendarAuth()
	if err != nil {
		t.Fatalf("new auth failed: %v", err)
	}
	baseDir := t.TempDir()
	a.SetCachePath(filepath.Join(baseDir, "oauth_cache.json"))
	if err := os.WriteFile(a.CachePath(), []byte(`{"access_token":"abc"}`), 0600); err != nil {
		t.Fatalf("seed cache failed: %v", err)
	}
	if err := a.ActivateAccount("test.user@example.com"); err != nil {
		t.Fatalf("activate account failed: %v", err)
	}
	if !strings.Contains(a.CachePath(), "oauth_cache_test.user_example.com.json") {
		t.Fatalf("unexpected account cache path: %s", a.CachePath())
	}
	if _, err := os.Stat(a.CachePath()); err != nil {
		t.Fatalf("expected account cache file: %v", err)
	}
}
