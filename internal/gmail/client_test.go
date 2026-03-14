package gmail

import (
	"context"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ShaneOxM/granola-cli-go/internal/auth"
	"github.com/ShaneOxM/granola-cli-go/internal/config"
)

func TestToMessage(t *testing.T) {
	raw := messageResponse{ID: "m1", ThreadID: "t1", Snippet: "hello"}
	raw.Payload.Headers = append(raw.Payload.Headers,
		struct {
			Name  string `json:"name"`
			Value string `json:"value"`
		}{Name: "From", Value: "alice@example.com"},
		struct {
			Name  string `json:"name"`
			Value string `json:"value"`
		}{Name: "Subject", Value: "Subject"},
		struct {
			Name  string `json:"name"`
			Value string `json:"value"`
		}{Name: "Date", Value: "Mon"},
	)
	m := toMessage(raw)
	if m.ID != "m1" || m.From == "" || m.Subject == "" {
		t.Fatalf("unexpected message conversion: %+v", m)
	}
}

func TestToMessageWithoutPayload(t *testing.T) {
	m := toMessage(messageResponse{ID: "x"})
	if m.ID != "x" {
		t.Fatalf("expected id x, got %s", m.ID)
	}
}

func TestBuildAroundMeetingQuery(t *testing.T) {
	q := BuildAroundMeetingQuery([]string{"a@example.com", "", "b@example.com"}, time.Date(2026, 3, 7, 12, 0, 0, 0, time.UTC))
	if q == "" {
		t.Fatalf("query should not be empty")
	}
	if !(strings.Contains(q, "from:a@example.com") && strings.Contains(q, "from:b@example.com")) {
		t.Fatalf("expected attendee filters in query: %s", q)
	}
}

func TestListMessagesAndGetMessage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasPrefix(r.URL.Path, "/users/me/messages") && r.Method == http.MethodGet && r.URL.Query().Get("format") == "":
			_, _ = w.Write([]byte(`{"messages":[{"id":"m1"}]}`))
		case strings.HasPrefix(r.URL.Path, "/users/me/messages/m1"):
			_, _ = w.Write([]byte(`{"id":"m1","threadId":"t1","snippet":"hello","payload":{"headers":[{"name":"From","value":"a@example.com"},{"name":"Subject","value":"Hi"},{"name":"Date","value":"Mon"}]}}`))
		default:
			http.Error(w, "not found", http.StatusNotFound)
		}
	}))
	defer server.Close()

	old := baseURL
	baseURL = server.URL
	defer func() { baseURL = old }()

	c := &Client{http: server.Client()}
	msgs, err := c.ListMessages(context.Background(), "", 1)
	if err != nil {
		t.Fatalf("list failed: %v", err)
	}
	if len(msgs) != 1 || msgs[0].ID != "m1" {
		t.Fatalf("unexpected list response: %+v", msgs)
	}

	m, err := c.GetMessage(context.Background(), "m1", "metadata")
	if err != nil {
		t.Fatalf("get failed: %v", err)
	}
	if m.Subject != "Hi" {
		t.Fatalf("unexpected subject: %+v", m)
	}
}

func TestListMessagesHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusBadRequest)
	}))
	defer server.Close()

	old := baseURL
	baseURL = server.URL
	defer func() { baseURL = old }()

	c := &Client{http: server.Client()}
	_, err := c.ListMessages(context.Background(), "", 1)
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestListMessagesRetriesOn429(t *testing.T) {
	count := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/users/me/messages") && r.URL.Query().Get("format") == "" {
			count++
			if count == 1 {
				http.Error(w, "rate limited", http.StatusTooManyRequests)
				return
			}
			_, _ = w.Write([]byte(`{"messages":[]}`))
			return
		}
		http.Error(w, "not found", http.StatusNotFound)
	}))
	defer server.Close()

	old := baseURL
	baseURL = server.URL
	defer func() { baseURL = old }()

	c := &Client{http: server.Client()}
	msgs, err := c.ListMessages(context.Background(), "", 1)
	if err != nil {
		t.Fatalf("expected retry success, got %v", err)
	}
	if len(msgs) != 0 || count < 2 {
		t.Fatalf("expected retry behavior, count=%d", count)
	}
}

func TestListMessagesSetsQuotaHeaderWhenAvailable(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.json")
	oldCfg := os.Getenv("GRANOLA_CONFIG_PATH")
	defer func() { _ = os.Setenv("GRANOLA_CONFIG_PATH", oldCfg) }()
	_ = os.Setenv("GRANOLA_CONFIG_PATH", configPath)
	if err := config.Init(); err != nil {
		t.Fatalf("config init failed: %v", err)
	}
	cfg, _ := config.Read()
	cfg.GoogleAuthMode = "adc"
	cfg.GoogleActiveAccount = ""
	cfg.GoogleAccounts = nil
	_ = config.Write(cfg)

	adcPath := filepath.Join(t.TempDir(), "adc.json")
	_ = os.WriteFile(adcPath, []byte(`{"quota_project_id":"qp-test"}`), 0600)
	oldADC := os.Getenv("GOOGLE_APPLICATION_CREDENTIALS")
	defer func() { _ = os.Setenv("GOOGLE_APPLICATION_CREDENTIALS", oldADC) }()
	_ = os.Setenv("GOOGLE_APPLICATION_CREDENTIALS", adcPath)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("x-goog-user-project"); got != "qp-test" {
			t.Fatalf("expected quota header qp-test, got %q", got)
		}
		_, _ = w.Write([]byte(`{"messages":[]}`))
	}))
	defer server.Close()

	old := baseURL
	baseURL = server.URL
	defer func() { baseURL = old }()

	c := &Client{auth: &auth.GmailCalendarAuth{}, http: server.Client()}
	_, err := c.ListMessages(context.Background(), "", 1)
	if err != nil {
		t.Fatalf("expected success, got %v", err)
	}
}

func TestBuildPersonQuery(t *testing.T) {
	q := BuildPersonQuery("a@example.com")
	if !strings.Contains(q, "from:a@example.com") || !strings.Contains(q, "to:a@example.com") {
		t.Fatalf("unexpected query: %s", q)
	}
}

func TestGetThread(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/threads/t1") {
			_, _ = w.Write([]byte(`{"id":"t1","messages":[{"id":"m1","threadId":"t1","snippet":"hello","payload":{"headers":[{"name":"From","value":"a@example.com"},{"name":"Subject","value":"Hi"}]}}]}`))
			return
		}
		http.Error(w, "not found", http.StatusNotFound)
	}))
	defer server.Close()
	old := baseURL
	baseURL = server.URL
	defer func() { baseURL = old }()
	c := &Client{http: server.Client()}
	thread, err := c.GetThread(context.Background(), "t1")
	if err != nil || thread.ID != "t1" || len(thread.Messages) != 1 {
		t.Fatalf("unexpected thread response: %+v err=%v", thread, err)
	}
}

func TestToMessageExtractsBody(t *testing.T) {
	body := base64.RawURLEncoding.EncodeToString([]byte("hello plain text"))
	raw := messageResponse{ID: "m1", ThreadID: "t1", Snippet: "hello"}
	raw.Payload.MimeType = "text/plain"
	raw.Payload.Body.Data = body
	m := toMessage(raw)
	if m.Body != "hello plain text" {
		t.Fatalf("expected decoded body, got %q", m.Body)
	}
}

func TestListInvolvingPerson(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if q := r.URL.Query().Get("q"); !strings.Contains(q, "to:a@example.com") {
			t.Fatalf("expected involving-person query, got %q", q)
		}
		if strings.HasPrefix(r.URL.Path, "/users/me/messages") && r.URL.Query().Get("format") == "" {
			_, _ = w.Write([]byte(`{"messages":[]}`))
			return
		}
		http.Error(w, "not found", http.StatusNotFound)
	}))
	defer server.Close()
	old := baseURL
	baseURL = server.URL
	defer func() { baseURL = old }()
	c := &Client{http: server.Client()}
	_, err := c.ListInvolvingPerson(context.Background(), "a@example.com", 5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
