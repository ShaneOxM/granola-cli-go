package auth

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/ShaneOxM/granola-cli-go/internal/secrets"
	"golang.org/x/oauth2"
)

func TestGenerateSecureState(t *testing.T) {
	a := generateSecureState()
	b := generateSecureState()
	if a == "" || b == "" || a == b {
		t.Fatalf("expected unique non-empty states")
	}
}

func TestGetOAuthConfigFromEnv(t *testing.T) {
	oldID, oldSecret := os.Getenv("GRANOLA_OAUTH_CLIENT_ID"), os.Getenv("GRANOLA_OAUTH_CLIENT_SECRET")
	oldAuth, oldToken := os.Getenv("GRANOLA_OAUTH_AUTH_URL"), os.Getenv("GRANOLA_OAUTH_TOKEN_URL")
	defer func() {
		_ = os.Setenv("GRANOLA_OAUTH_CLIENT_ID", oldID)
		_ = os.Setenv("GRANOLA_OAUTH_CLIENT_SECRET", oldSecret)
		_ = os.Setenv("GRANOLA_OAUTH_AUTH_URL", oldAuth)
		_ = os.Setenv("GRANOLA_OAUTH_TOKEN_URL", oldToken)
		defaultOAuthConfig = nil
	}()
	defaultOAuthConfig = nil
	_ = os.Setenv("GRANOLA_OAUTH_CLIENT_ID", "id")
	_ = os.Setenv("GRANOLA_OAUTH_CLIENT_SECRET", "secret")
	_ = os.Setenv("GRANOLA_OAUTH_AUTH_URL", "https://example.com/auth")
	_ = os.Setenv("GRANOLA_OAUTH_TOKEN_URL", "https://example.com/token")
	cfg := getOAuthConfig()
	if cfg == nil || cfg.ClientID != "id" {
		t.Fatalf("expected config from env")
	}
}

func TestLoginWithBrowserRequiresConfig(t *testing.T) {
	old := defaultOAuthConfig
	defer func() { defaultOAuthConfig = old }()
	defaultOAuthConfig = nil
	if err := LoginWithBrowser("user@example.com"); err == nil {
		t.Fatalf("expected missing config error")
	}
}

func TestGetOAuthToken(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"access","refresh_token":"refresh2","token_type":"Bearer","expires_in":3600}`))
	}))
	defer server.Close()
	oldCfg := defaultOAuthConfig
	oldAuth := defaultAuth
	defer func() {
		defaultOAuthConfig = oldCfg
		defaultAuth = oldAuth
	}()
	defaultOAuthConfig = &oauth2.Config{Endpoint: oauth2.Endpoint{TokenURL: server.URL}}
	store := secrets.NewMockTokenStore()
	_ = store.SetToken("granola-user", &secrets.Token{Email: "granola-user", RefreshToken: "refresh", CreatedAt: time.Now()})
	defaultAuth = &Auth{store: store, cache: map[string]string{}}
	tok, err := GetOAuthToken()
	if err != nil {
		t.Fatalf("GetOAuthToken error: %v", err)
	}
	if tok.AccessToken != "access" {
		t.Fatalf("unexpected token: %+v", tok)
	}
	stored, err := store.GetToken("granola-user")
	if err != nil || stored.RefreshToken != "refresh2" {
		t.Fatalf("expected refreshed token persisted: %+v err=%v", stored, err)
	}
}
