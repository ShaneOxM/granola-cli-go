package meeting

import (
	"context"
	"strings"
	"time"

	api "github.com/ShaneOxM/granola-cli-go/internal/api"
	cal "github.com/ShaneOxM/granola-cli-go/internal/calendar"
)

type CalendarClient interface {
	ListEvents(ctx context.Context, calendarID string, start, end time.Time, max int64) ([]cal.Event, error)
}

type EnrichedMeeting struct {
	Meeting       api.Meeting `json:"meeting"`
	MatchedEvent  *cal.Event  `json:"matched_event,omitempty"`
	MatchedBy     string      `json:"matched_by,omitempty"`
	MatchScore    int         `json:"match_score"`
	NeedsReview   bool        `json:"needs_review"`
	MissingReason string      `json:"missing_reason,omitempty"`
}

func EnrichMeetings(ctx context.Context, client CalendarClient, meetings []api.Meeting) ([]EnrichedMeeting, error) {
	start := time.Now().Add(-30 * 24 * time.Hour)
	end := time.Now().Add(30 * 24 * time.Hour)
	events, err := client.ListEvents(ctx, "primary", start, end, 250)
	if err != nil {
		return nil, err
	}

	out := make([]EnrichedMeeting, 0, len(meetings))
	for _, m := range meetings {
		best := bestMatch(m, events)
		if best == nil {
			out = append(out, EnrichedMeeting{
				Meeting:       m,
				NeedsReview:   true,
				MissingReason: "no matching calendar event",
			})
			continue
		}
		out = append(out, EnrichedMeeting{
			Meeting:      m,
			MatchedEvent: best,
			MatchedBy:    "title+attendees",
			MatchScore:   scoreMatch(m, *best),
		})
	}
	return out, nil
}

func bestMatch(m api.Meeting, events []cal.Event) *cal.Event {
	var best *cal.Event
	bestScore := -1
	for i := range events {
		s := scoreMatch(m, events[i])
		if s > bestScore {
			best = &events[i]
			bestScore = s
		}
	}
	if bestScore < 2 {
		return nil
	}
	return best
}

func scoreMatch(m api.Meeting, e cal.Event) int {
	score := 0
	if strings.TrimSpace(m.Title) != "" && strings.Contains(strings.ToLower(e.Summary), strings.ToLower(m.Title)) {
		score += 2
	}
	if len(AttendeeEmails(m)) >= 2 {
		score += 1
	}
	return score
}
