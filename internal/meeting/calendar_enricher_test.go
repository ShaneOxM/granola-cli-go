package meeting

import (
	"context"
	"fmt"
	"testing"
	"time"

	api "github.com/ShaneOxM/granola-cli-go/internal/api"
	cal "github.com/ShaneOxM/granola-cli-go/internal/calendar"
)

type fakeCalendarClient struct {
	events []cal.Event
	err    error
}

func (f *fakeCalendarClient) ListEvents(ctx context.Context, calendarID string, start, end time.Time, max int64) ([]cal.Event, error) {
	return f.events, f.err
}

func TestScoreMatch(t *testing.T) {
	m := api.Meeting{Title: "Weekly Standup", Attendees: []api.Person{{Email: "a@example.com"}, {Email: "b@example.com"}}}
	e := cal.Event{Summary: "weekly standup sync"}
	s := scoreMatch(m, e)
	if s < 2 {
		t.Fatalf("expected score >=2, got %d", s)
	}
}

func TestBestMatchNoEnoughScore(t *testing.T) {
	m := api.Meeting{Title: "Unique Meeting"}
	events := []cal.Event{{Summary: "Other"}}
	if bestMatch(m, events) != nil {
		t.Fatalf("expected nil match")
	}
}

func TestScoreMatchAttendeeOnly(t *testing.T) {
	m := api.Meeting{Title: "No title match", Attendees: []api.Person{{Email: "a@example.com"}, {Email: "b@example.com"}}}
	e := cal.Event{Summary: "different event"}
	if s := scoreMatch(m, e); s != 1 {
		t.Fatalf("expected attendee-only score 1, got %d", s)
	}
}

func TestBestMatchChoosesHighestScore(t *testing.T) {
	m := api.Meeting{Title: "Weekly Standup", Attendees: []api.Person{{Email: "a@example.com"}, {Email: "b@example.com"}}}
	events := []cal.Event{{Summary: "Other Event"}, {Summary: "Weekly Standup Sync"}}
	best := bestMatch(m, events)
	if best == nil {
		t.Fatalf("expected a best match")
	}
	if best.Summary != "Weekly Standup Sync" {
		t.Fatalf("unexpected best match: %+v", best)
	}
}

func TestEnrichMeetings(t *testing.T) {
	client := &fakeCalendarClient{events: []cal.Event{{ID: "ev1", Summary: "Weekly Standup Sync"}}}
	meetings := []api.Meeting{{ID: "m1", Title: "Weekly Standup", Attendees: []api.Person{{Email: "a@example.com"}, {Email: "b@example.com"}}}}
	items, err := EnrichMeetings(context.Background(), client, meetings)
	if err != nil {
		t.Fatalf("enrich failed: %v", err)
	}
	if len(items) != 1 || items[0].MatchedEvent == nil {
		t.Fatalf("expected matched event: %+v", items)
	}
}

func TestEnrichMeetingsNoMatch(t *testing.T) {
	client := &fakeCalendarClient{events: []cal.Event{{ID: "ev1", Summary: "Other"}}}
	meetings := []api.Meeting{{ID: "m1", Title: "Unique Meeting"}}
	items, err := EnrichMeetings(context.Background(), client, meetings)
	if err != nil {
		t.Fatalf("enrich failed: %v", err)
	}
	if !items[0].NeedsReview {
		t.Fatalf("expected needs review item: %+v", items[0])
	}
}

func TestEnrichMeetingsError(t *testing.T) {
	client := &fakeCalendarClient{err: fmt.Errorf("boom")}
	_, err := EnrichMeetings(context.Background(), client, []api.Meeting{{ID: "m1"}})
	if err == nil {
		t.Fatal("expected error")
	}
}
