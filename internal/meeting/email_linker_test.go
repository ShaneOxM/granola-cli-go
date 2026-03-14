package meeting

import (
	"context"
	"testing"
	"time"

	api "github.com/ShaneOxM/granola-cli-go/internal/api"
	gm "github.com/ShaneOxM/granola-cli-go/internal/gmail"
	"github.com/ShaneOxM/granola-cli-go/internal/storage"
)

type fakeGmailClient struct {
	messages []gm.Message
	err      error
	emails   []string
	around   time.Time
	max      int64
}

func (f *fakeGmailClient) AroundMeeting(ctx context.Context, attendeeEmails []string, around time.Time, max int64) ([]gm.Message, error) {
	f.emails = attendeeEmails
	f.around = around
	f.max = max
	return f.messages, f.err
}

func TestAttendeeEmails(t *testing.T) {
	m := api.Meeting{Attendees: []api.Person{
		{Name: "A", Email: "a@example.com"},
		{Name: "B", Email: ""},
		{Name: "C", Email: "c@example.com"},
	}}
	emails := AttendeeEmails(m)
	if len(emails) != 2 {
		t.Fatalf("expected 2 emails, got %d", len(emails))
	}
}

func TestLinkMeetingEmailsNoDB(t *testing.T) {
	m := api.Meeting{Attendees: []api.Person{{Email: "a@example.com"}}, CreatedAt: "2026-03-13T12:00:00Z"}
	client := &fakeGmailClient{messages: []gm.Message{{ID: "e1", ThreadID: "t1", From: "a@example.com", Subject: "Hi"}}}
	msgs, err := LinkMeetingEmails(context.Background(), nil, client, m, 5)
	if err != nil || len(msgs) != 1 {
		t.Fatalf("unexpected result err=%v msgs=%+v", err, msgs)
	}
	if len(client.emails) != 1 || client.max != 5 {
		t.Fatalf("client not called as expected: %+v", client)
	}
}

func TestLinkMeetingEmailsStoresToDB(t *testing.T) {
	db, err := storage.NewDB(t.TempDir() + "/test.db")
	if err != nil {
		t.Fatalf("db error: %v", err)
	}
	defer db.Close()
	m := api.Meeting{ID: "m1", Attendees: []api.Person{{Email: "a@example.com"}}, CreatedAt: "2026-03-13T12:00:00Z"}
	client := &fakeGmailClient{messages: []gm.Message{{ID: "e1", ThreadID: "t1", From: "A <a@example.com>", Subject: "Subject"}}}
	_, err = LinkMeetingEmails(context.Background(), db, client, m, 5)
	if err != nil {
		t.Fatalf("link failed: %v", err)
	}
	links, err := db.GetEmailLinks("m1")
	if err != nil || len(links) != 1 {
		t.Fatalf("unexpected links: %+v err=%v", links, err)
	}
	if links[0].Reason != "attendee_sender" || links[0].Score != 2 {
		t.Fatalf("unexpected saved link: %+v", links[0])
	}
}

func TestAroundMeetingEmailsUsesCreatedAt(t *testing.T) {
	client := &fakeGmailClient{}
	m := api.Meeting{Attendees: []api.Person{{Email: "a@example.com"}}, CreatedAt: "2026-03-13T12:00:00Z"}
	_, _ = AroundMeetingEmails(context.Background(), client, m, 9)
	if client.around.IsZero() || client.max != 9 {
		t.Fatalf("expected createdAt-based call, got %+v", client)
	}
}
