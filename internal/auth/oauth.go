package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"time"

	"github.com/ShaneOxM/granola-cli-go/internal/secrets"
	"golang.org/x/oauth2"
)

var defaultOAuthConfig *oauth2.Config

func validateOAuthConfig() error {
	clientID := os.Getenv("GRANOLA_OAUTH_CLIENT_ID")
	if clientID == "" {
		return fmt.Errorf("GRANOLA_OAUTH_CLIENT_ID environment variable is required")
	}

	clientSecret := os.Getenv("GRANOLA_OAUTH_CLIENT_SECRET")
	if clientSecret == "" {
		return fmt.Errorf("GRANOLA_OAUTH_CLIENT_SECRET environment variable is required")
	}

	authURL := os.Getenv("GRANOLA_OAUTH_AUTH_URL")
	tokenURL := os.Getenv("GRANOLA_OAUTH_TOKEN_URL")

	if authURL == "" || tokenURL == "" {
		return fmt.Errorf("GRANOLA_OAUTH_AUTH_URL and GRANOLA_OAUTH_TOKEN_URL environment variables are required")
	}

	defaultOAuthConfig = &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		Endpoint: oauth2.Endpoint{
			AuthURL:  authURL,
			TokenURL: tokenURL,
		},
		Scopes: []string{"meetings:read", "meetings:write", "transcripts:read"},
	}

	return nil
}

func generateSecureState() string {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		panic("failed to generate secure state: " + err.Error())
	}
	return base64.URLEncoding.EncodeToString(bytes)
}

func getOAuthConfig() *oauth2.Config {
	if defaultOAuthConfig != nil {
		return defaultOAuthConfig
	}
	// Fallback for tests before Init() is called
	clientID := os.Getenv("GRANOLA_OAUTH_CLIENT_ID")
	clientSecret := os.Getenv("GRANOLA_OAUTH_CLIENT_SECRET")
	authURL := os.Getenv("GRANOLA_OAUTH_AUTH_URL")
	tokenURL := os.Getenv("GRANOLA_OAUTH_TOKEN_URL")

	if clientID != "" && clientSecret != "" && authURL != "" && tokenURL != "" {
		return &oauth2.Config{
			ClientID:     clientID,
			ClientSecret: clientSecret,
			Endpoint: oauth2.Endpoint{
				AuthURL:  authURL,
				TokenURL: tokenURL,
			},
			Scopes: []string{"meetings:read", "meetings:write", "transcripts:read"},
		}
	}
	return nil
}

func LoginWithBrowser(email string) error {
	cfg := getOAuthConfig()
	if cfg == nil {
		return fmt.Errorf("OAuth config not initialized")
	}

	state := generateSecureState()
	authURL := cfg.AuthCodeURL("state-"+state, oauth2.AccessTypeOffline)

	fmt.Printf("Opening browser for authentication...\n")
	fmt.Printf("Please visit: %s\n\n", authURL)

	if err := openBrowser(authURL); err != nil {
		fmt.Printf("Could not open browser: %v\n", err)
		fmt.Printf("Please manually visit the URL and copy the callback\n\n")
	}

	return handleCallback(state)
}

func openBrowser(url string) error {
	cmd := exec.Command("open", url)
	if err := cmd.Start(); err != nil {
		return err
	}
	return nil
}

func handleCallback(expectedState string) error {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return fmt.Errorf("failed to start server: %v", err)
	}
	defer listener.Close()

	port := listener.Addr().(*net.TCPAddr).Port
	callbackURL := fmt.Sprintf("http://127.0.0.1:%d", port)

	fmt.Printf("Waiting for callback at: %s\n", callbackURL)

	resultChan := make(chan error, 1)

	http.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		code := r.URL.Query().Get("code")
		if code == "" {
			http.Error(w, "No code in callback", http.StatusBadRequest)
			return
		}

		returnedState := r.URL.Query().Get("state")
		if returnedState != expectedState {
			http.Error(w, "Invalid state parameter", http.StatusBadRequest)
			resultChan <- fmt.Errorf("CSRF protection: state mismatch")
			return
		}

		cfg := getOAuthConfig()
		if cfg == nil {
			http.Error(w, "OAuth config not initialized", http.StatusInternalServerError)
			resultChan <- fmt.Errorf("OAuth config not initialized")
			return
		}

		token, err := cfg.Exchange(context.Background(), code)
		if err != nil {
			http.Error(w, fmt.Sprintf("Token exchange failed: %v", err), http.StatusInternalServerError)
			resultChan <- err
			return
		}

		fmt.Printf("Token received! Storing in keyring...\n")

		tokenObj := &secrets.Token{
			Email:        "granola-user",
			RefreshToken: token.RefreshToken,
			CreatedAt:    time.Now(),
		}

		if defaultAuth != nil {
			if err := defaultAuth.store.SetToken("granola-user", tokenObj); err != nil {
				http.Error(w, fmt.Sprintf("Failed to store token: %v", err), http.StatusInternalServerError)
				resultChan <- err
				return
			}
		} else {
			http.Error(w, "Auth not initialized", http.StatusInternalServerError)
			resultChan <- fmt.Errorf("auth not initialized")
			return
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Authentication successful! You can close this window."))
		resultChan <- nil
	})

	go func() {
		if err := http.Serve(listener, nil); err != nil && err != http.ErrServerClosed {
			resultChan <- err
		}
	}()

	select {
	case err := <-resultChan:
		return err
	case <-time.After(5 * time.Minute):
		return fmt.Errorf("timeout waiting for callback")
	}
}

func GetOAuthToken() (*oauth2.Token, error) {
	if defaultAuth == nil {
		return nil, fmt.Errorf("auth not initialized")
	}

	token, err := defaultAuth.store.GetToken("granola-user")
	if err != nil {
		return nil, fmt.Errorf("no token found: %v", err)
	}

	if token.RefreshToken == "" {
		return nil, fmt.Errorf("no refresh token stored")
	}

	cfg := getOAuthConfig()
	if cfg == nil {
		return nil, fmt.Errorf("OAuth config not initialized")
	}

	tokenObj := &oauth2.Token{
		RefreshToken: token.RefreshToken,
		TokenType:    "Bearer",
		Expiry:       time.Now().Add(-time.Hour),
	}

	newToken, err := cfg.TokenSource(context.Background(), tokenObj).Token()
	if err != nil {
		return nil, fmt.Errorf("failed to refresh token: %v", err)
	}

	updatedToken := &secrets.Token{
		Email:        token.Email,
		RefreshToken: newToken.RefreshToken,
		CreatedAt:    token.CreatedAt,
	}
	defaultAuth.store.SetToken("granola-user", updatedToken)

	return newToken, nil
}
