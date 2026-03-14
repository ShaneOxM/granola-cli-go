package calendar

import (
	"context"
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

func TestToEventDateTime(t *testing.T) {
	e := toEvent(eventResponse{
		ID:      "ev1",
		Summary: "Standup",
		Start: struct {
			DateTime string `json:"dateTime"`
			Date     string `json:"date"`
		}{DateTime: "2026-03-07T10:00:00Z"},
		End: struct {
			DateTime string `json:"dateTime"`
			Date     string `json:"date"`
		}{DateTime: "2026-03-07T10:30:00Z"},
	}, "primary")
	if e.ID != "ev1" || e.Start == "" || e.End == "" {
		t.Fatalf("unexpected event conversion: %+v", e)
	}
}

func TestToEventAllDay(t *testing.T) {
	e := toEvent(eventResponse{
		ID: "ev2",
		Start: struct {
			DateTime string `json:"dateTime"`
			Date     string `json:"date"`
		}{Date: "2026-03-08"},
		End: struct {
			DateTime string `json:"dateTime"`
			Date     string `json:"date"`
		}{Date: "2026-03-09"},
	}, "primary")
	if e.Start != "2026-03-08" {
		t.Fatalf("expected all-day start date, got %s", e.Start)
	}
}

func TestNormalizeListParams(t *testing.T) {
	calID, start, end, max := normalizeListParams("", time.Time{}, time.Time{}, 0)
	if calID != "primary" {
		t.Fatalf("expected default calendar id")
	}
	if max != 20 {
		t.Fatalf("expected default max=20")
	}
	if start.IsZero() || end.IsZero() || !end.After(start) {
		t.Fatalf("expected normalized start/end")
	}
}

func TestListEventsAndGetEvent(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/events"):
			_, _ = w.Write([]byte(`{"items":[{"id":"ev1","summary":"Standup","start":{"dateTime":"2026-03-07T10:00:00Z"},"end":{"dateTime":"2026-03-07T10:30:00Z"}}]}`))
		case strings.Contains(r.URL.Path, "/events/ev1"):
			_, _ = w.Write([]byte(`{"id":"ev1","summary":"Standup","description":"Daily","start":{"dateTime":"2026-03-07T10:00:00Z"},"end":{"dateTime":"2026-03-07T10:30:00Z"}}`))
		default:
			http.Error(w, "not found", http.StatusNotFound)
		}
	}))
	defer server.Close()

	old := baseURL
	baseURL = server.URL
	defer func() { baseURL = old }()

	c := &Client{http: server.Client()}
	events, err := c.ListEvents(context.Background(), "primary", time.Time{}, time.Time{}, 1)
	if err != nil {
		t.Fatalf("list events failed: %v", err)
	}
	if len(events) != 1 || events[0].ID != "ev1" {
		t.Fatalf("unexpected list response: %+v", events)
	}
	ev, err := c.GetEvent(context.Background(), "primary", "ev1")
	if err != nil {
		t.Fatalf("get event failed: %v", err)
	}
	if ev.Summary != "Standup" {
		t.Fatalf("unexpected event: %+v", ev)
	}
}

func TestGetEventMissingID(t *testing.T) {
	c := &Client{}
	_, err := c.GetEvent(context.Background(), "primary", "")
	if err == nil {
		t.Fatalf("expected error for missing id")
	}
}

func TestListEventsRetriesOn429(t *testing.T) {
	count := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/events") {
			count++
			if count == 1 {
				http.Error(w, "rate limited", http.StatusTooManyRequests)
				return
			}
			_, _ = w.Write([]byte(`{"items":[]}`))
			return
		}
		http.Error(w, "not found", http.StatusNotFound)
	}))
	defer server.Close()

	old := baseURL
	baseURL = server.URL
	defer func() { baseURL = old }()

	c := &Client{http: server.Client()}
	events, err := c.ListEvents(context.Background(), "primary", time.Time{}, time.Time{}, 1)
	if err != nil {
		t.Fatalf("expected retry success, got %v", err)
	}
	if len(events) != 0 || count < 2 {
		t.Fatalf("expected retry behavior, count=%d", count)
	}
}

func TestListEventsSetsQuotaHeaderWhenAvailable(t *testing.T) {
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
		_, _ = w.Write([]byte(`{"items":[]}`))
	}))
	defer server.Close()

	old := baseURL
	baseURL = server.URL
	defer func() { baseURL = old }()

	c := &Client{auth: &auth.GmailCalendarAuth{}, http: server.Client()}
	_, err := c.ListEvents(context.Background(), "primary", time.Time{}, time.Time{}, 1)
	if err != nil {
		t.Fatalf("expected success, got %v", err)
	}
}
