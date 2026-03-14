package main

import (
	"strings"
	"testing"

	api "github.com/ShaneOxM/granola-cli-go/internal/api"
	calpkg "github.com/ShaneOxM/granola-cli-go/internal/calendar"
	"github.com/ShaneOxM/granola-cli-go/internal/embeddings"
	gm "github.com/ShaneOxM/granola-cli-go/internal/gmail"
	meet "github.com/ShaneOxM/granola-cli-go/internal/meeting"
)

func TestParseGmailListOptions(t *testing.T) {
	opts := parseGmailListOptions([]string{"newer_than:7d", "--person=a@example.com", "--max=5", "--json"})
	if opts.query != "newer_than:7d" || opts.person != "a@example.com" || opts.max != 5 || !opts.jsonOutput {
		t.Fatalf("unexpected opts: %+v", opts)
	}
}

func TestParseGmailGetOptions(t *testing.T) {
	opts := parseGmailGetOptions([]string{"--body", "--json"})
	if !opts.bodyOutput || !opts.jsonOutput || opts.format != "full" {
		t.Fatalf("unexpected opts: %+v", opts)
	}
}

func TestParseCalendarListOptions(t *testing.T) {
	opts := parseCalendarListOptions([]string{"--calendar=team", "--max=7", "--json"})
	if opts.calendarID != "team" || opts.max != 7 || !opts.jsonOutput {
		t.Fatalf("unexpected opts: %+v", opts)
	}
}

func TestGmailRows(t *testing.T) {
	rows := gmailRows([]gm.Message{{Date: "date", From: "from", Subject: "subject"}})
	if len(rows) != 1 || len(rows[0]) != 3 {
		t.Fatalf("unexpected rows: %+v", rows)
	}
}

func TestCalendarRows(t *testing.T) {
	rows := calendarRows([]calpkg.Event{{Start: "start", Summary: "title", ID: "id"}})
	if len(rows) != 1 || len(rows[0]) != 3 {
		t.Fatalf("unexpected rows: %+v", rows)
	}
}

func TestEnrichedMeetingRows(t *testing.T) {
	rows := enrichedMeetingRows([]meet.EnrichedMeeting{{MatchScore: 2, Meeting: api.Meeting{Title: "meeting"}, NeedsReview: true}})
	if len(rows) != 1 || rows[0][3] != "review" {
		t.Fatalf("unexpected rows: %+v", rows)
	}
}

func TestGroupSearchResults(t *testing.T) {
	results := []embeddings.SearchResult{
		{MeetingID: "m1", MeetingTitle: "Meeting 1", Score: 0.8, ChunkText: "a"},
		{MeetingID: "m1", MeetingTitle: "Meeting 1", Score: 0.6, ChunkText: "b"},
		{MeetingID: "m2", MeetingTitle: "Meeting 2", Score: 0.7, ChunkText: "c"},
	}
	grouped := groupSearchResults(results, map[string]string{"m1": "Meeting 1", "m2": "Meeting 2"})
	if len(grouped) != 2 {
		t.Fatalf("expected 2 groups, got %d", len(grouped))
	}
	if grouped[0].MeetingID != "m1" || grouped[0].BestScore != 0.8 {
		t.Fatalf("unexpected first group: %+v", grouped[0])
	}
}

func TestRenderProseMirrorText(t *testing.T) {
	doc := &api.ProseMirrorDoc{Content: []api.ProseMirrorNode{
		{Type: "paragraph", Content: []api.ProseMirrorNode{{Type: "text", Text: "Hello"}, {Type: "text", Text: "world"}}},
		{Type: "bulletList", Content: []api.ProseMirrorNode{{Type: "listItem", Content: []api.ProseMirrorNode{{Type: "paragraph", Content: []api.ProseMirrorNode{{Type: "text", Text: "Item 1"}}}}}}},
	}}
	out := renderProseMirrorText(doc)
	if out == "" || !containsText(out, "Hello world") || !containsText(out, "- Item 1") {
		t.Fatalf("unexpected rendered notes: %q", out)
	}
}

func containsText(s, part string) bool { return strings.Contains(s, part) }
