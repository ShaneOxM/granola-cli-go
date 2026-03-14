package auth

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCallWorkOSRefresh(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req refreshRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if req.ClientID != "client" || req.RefreshToken != "refresh" {
			t.Fatalf("unexpected request: %+v", req)
		}
		_, _ = w.Write([]byte(`{"access_token":"new-access","refresh_token":"new-refresh","token_type":"Bearer","expires_in":3600}`))
	}))
	defer server.Close()
	orig := workOSAuthURL
	defer func() { workOSAuthURL = orig }()
	workOSAuthURL = server.URL
	resp, err := callWorkOSRefresh([]byte(`{"client_id":"client","grant_type":"refresh_token","refresh_token":"refresh"}`))
	if err != nil || resp.AccessToken != "new-access" {
		t.Fatalf("unexpected response %+v err=%v", resp, err)
	}
}

func TestCallWorkOSRefreshHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "bad request", http.StatusBadRequest)
	}))
	defer server.Close()
	orig := workOSAuthURL
	defer func() { workOSAuthURL = orig }()
	workOSAuthURL = server.URL
	_, err := callWorkOSRefresh([]byte(`{}`))
	if err == nil || !strings.Contains(err.Error(), "HTTP 400") {
		t.Fatalf("expected HTTP 400 error, got %v", err)
	}
}

func TestGetLockPath(t *testing.T) {
	path, err := getLockPath()
	if err != nil || !strings.Contains(path, ".granola_lock_") {
		t.Fatalf("unexpected path %q err=%v", path, err)
	}
}
