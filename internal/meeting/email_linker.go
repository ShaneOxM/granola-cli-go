package meeting

import (
	"context"
	"strings"
	"time"

	api "github.com/ShaneOxM/granola-cli-go/internal/api"
	gm "github.com/ShaneOxM/granola-cli-go/internal/gmail"
	"github.com/ShaneOxM/granola-cli-go/internal/storage"
)

type GmailClient interface {
	AroundMeeting(ctx context.Context, attendeeEmails []string, around time.Time, max int64) ([]gm.Message, error)
}

func AttendeeEmails(m api.Meeting) []string {
	out := make([]string, 0, len(m.Attendees))
	for _, a := range m.Attendees {
		if a.Email != "" {
			out = append(out, a.Email)
		}
	}
	return out
}

func AroundMeetingEmails(ctx context.Context, client GmailClient, m api.Meeting, max int64) ([]gm.Message, error) {
	t := time.Now()
	if m.CreatedAt != "" {
		if parsed, err := time.Parse(time.RFC3339, m.CreatedAt); err == nil {
			t = parsed
		}
	}
	return client.AroundMeeting(ctx, AttendeeEmails(m), t, max)
}

func LinkMeetingEmails(ctx context.Context, db *storage.DB, client GmailClient, m api.Meeting, max int64) ([]gm.Message, error) {
	msgs, err := AroundMeetingEmails(ctx, client, m, max)
	if err != nil {
		return nil, err
	}
	if db == nil {
		return msgs, nil
	}
	for _, msg := range msgs {
		reason := "around_meeting"
		score := 1
		for _, email := range AttendeeEmails(m) {
			if email != "" && strings.Contains(strings.ToLower(msg.From), strings.ToLower(email)) {
				reason = "attendee_sender"
				score = 2
				break
			}
		}
		_ = db.SaveEmailLink(m.ID, msg.ID, msg.ThreadID, reason, score)
	}
	return msgs, nil
}
