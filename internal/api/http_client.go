// Package api provides HTTP client functionality for Granola API communications.
// It handles authentication, retries, and TLS verification.
package api

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/ShaneOxM/granola-cli-go/internal/auth"
)

const (
	baseURL    = "https://api.granola.ai"
	appVersion = "7.0.0"
	maxRetries = 3
	baseDelay  = 250 * time.Millisecond
)

// ApiError represents an API error response
// ApiErrorDetail represents structured error details
type ApiErrorDetail struct {
	Code    string                 `json:"code,omitempty"`
	Field   string                 `json:"field,omitempty"`
	Message string                 `json:"message,omitempty"`
	Extra   map[string]interface{} `json:"extra,omitempty"`
}

type ApiError struct {
	Message string          `json:"error,omitempty"`
	Detail  *ApiErrorDetail `json:"detail,omitempty"`
	Status  int             `json:"-"`
}

func (e *ApiError) Error() string {
	if e.Message != "" {
		return e.Message
	}
	return fmt.Sprintf("HTTP %d", e.Status)
}

// HttpClient handles HTTP requests to Granola API
type HttpClient struct {
	token  string
	client *http.Client
}

// NewHttpClient creates a new HTTP client with authentication
func NewHttpClient() (*HttpClient, error) {
	// Get auth env vars
	envVars, err := auth.GetAuthEnvVars()
	if err != nil {
		return nil, fmt.Errorf("failed to get authentication: %w", err)
	}

	token := envVars["GRANOLA_ACCESS_TOKEN"]
	if token == "" {
		return nil, fmt.Errorf("no access token found")
	}

	// Create HTTP client with TLS configuration
	// Verify certificates and use secure defaults
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{
			MinVersion:         tls.VersionTLS12,
			InsecureSkipVerify: false,
		},
		MaxIdleConns:        10,
		MaxIdleConnsPerHost: 10,
		IdleConnTimeout:     90 * time.Second,
		DisableKeepAlives:   false,
	}

	return &HttpClient{
		token: token,
		client: &http.Client{
			Timeout:   60 * time.Second,
			Transport: transport,
		},
	}, nil
}

// SetToken updates the access token (useful after refresh)
func (h *HttpClient) SetToken(token string) {
	h.token = token
}

// post sends a POST request to the Granola API
func (h *HttpClient) post(endpoint string, body interface{}) (interface{}, error) {
	var lastErr error
	var responseData interface{}

	for attempt := 0; attempt < maxRetries; attempt++ {
		responseData, lastErr = h.tryPost(endpoint, body)
		if lastErr == nil {
			return responseData, nil
		}

		// Check if error is retryable
		if apiErr, ok := lastErr.(*ApiError); ok {
			if !isRetryableStatus(apiErr.Status) {
				return nil, lastErr
			}
		}

		// Wait before retry (exponential backoff)
		if attempt < maxRetries-1 {
			delay := baseDelay * time.Duration(1<<uint(attempt))
			time.Sleep(delay)

			// Try to refresh token on retry
			if isAuthError(lastErr) {
				fmt.Fprintf(os.Stderr, "Token expired, refreshing...\n")
				if err := refreshAndRetry(); err != nil {
					return nil, fmt.Errorf("token refresh failed: %w", err)
				}
			}
		}
	}

	return nil, fmt.Errorf("request failed after %d attempts: %w", maxRetries, lastErr)
}

// tryPost makes a single POST request
func (h *HttpClient) tryPost(endpoint string, body interface{}) (interface{}, error) {
	var bodyBytes []byte
	var err error

	if body != nil {
		bodyBytes, err = json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
	}

	url := baseURL + endpoint
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+h.token)
	req.Header.Set("X-App-Version", appVersion)
	req.Header.Set("X-Client-Version", appVersion)
	req.Header.Set("X-Client-Type", "cli")
	req.Header.Set("X-Client-Platform", os.Getenv("GOOS"))
	req.Header.Set("X-Client-Architecture", os.Getenv("GOARCH"))
	req.Header.Set("X-Client-Id", "granola-cli-go")

	// Set User-Agent
	goos := os.Getenv("GOOS")
	goarch := os.Getenv("GOARCH")
	if goos == "" {
		goos = "darwin"
	}
	if goarch == "" {
		goarch = "arm64"
	}
	req.Header.Set("User-Agent", fmt.Sprintf("Granola/%s granola-cli-go/0.1.0 (%s %s)",
		appVersion, goos, goarch))

	// Make request
	resp, err := h.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Check status
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		var apiErr ApiError
		if err := json.Unmarshal(respBody, &apiErr); err != nil {
			apiErr.Message = fmt.Sprintf("HTTP %d: %s", resp.StatusCode, string(respBody))
		}
		apiErr.Status = resp.StatusCode
		return nil, &apiErr
	}

	// Parse response
	var result interface{}
	if len(respBody) > 0 {
		if err := json.Unmarshal(respBody, &result); err != nil {
			return nil, fmt.Errorf("failed to parse response: %w", err)
		}
	}

	return result, nil
}

// isRetryableStatus checks if the status code is retryable
func isRetryableStatus(status int) bool {
	retryable := map[int]bool{
		429: true, // Too Many Requests
		500: true, // Internal Server Error
		502: true, // Bad Gateway
		503: true, // Service Unavailable
		504: true, // Gateway Timeout
	}
	return retryable[status]
}

// isAuthError checks if the error is authentication-related
func isAuthError(err error) bool {
	if apiErr, ok := err.(*ApiError); ok {
		return apiErr.Status == 401 || apiErr.Status == 403
	}
	return false
}

// refreshAndRetry refreshes the token and updates the HTTP client
func refreshAndRetry() error {
	// Refresh the access token
	creds, err := auth.RefreshAccessToken()
	if err != nil {
		return fmt.Errorf("failed to refresh token: %w", err)
	}

	// Update global token
	// Note: In a real implementation, we'd want to update the HttpClient instance
	// For now, we'll set it as an environment variable for child processes
	os.Setenv("GRANOLA_ACCESS_TOKEN", creds.AccessToken)
	os.Setenv("GRANOLA_REFRESH_TOKEN", creds.RefreshToken)

	return nil
}
