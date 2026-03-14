package main

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	api "github.com/ShaneOxM/granola-cli-go/internal/api"
	calpkg "github.com/ShaneOxM/granola-cli-go/internal/calendar"
	"github.com/ShaneOxM/granola-cli-go/internal/embeddings"
	gm "github.com/ShaneOxM/granola-cli-go/internal/gmail"
	meet "github.com/ShaneOxM/granola-cli-go/internal/meeting"
	"github.com/ShaneOxM/granola-cli-go/internal/output"
)

type gmailListOptions struct {
	query      string
	person     string
	jsonOutput bool
	max        int64
}

type gmailGetOptions struct {
	jsonOutput bool
	bodyOutput bool
	format     string
}

type calendarListOptions struct {
	calendarID string
	jsonOutput bool
	max        int64
}

func parseGmailListOptions(args []string) gmailListOptions {
	opts := gmailListOptions{max: 20}
	for _, a := range args {
		if a == "--json" {
			opts.jsonOutput = true
			continue
		}
		if strings.HasPrefix(a, "--max=") {
			if n, err := strconv.ParseInt(strings.TrimPrefix(a, "--max="), 10, 64); err == nil {
				opts.max = n
			}
			continue
		}
		if strings.HasPrefix(a, "--person=") {
			opts.person = strings.TrimSpace(strings.TrimPrefix(a, "--person="))
			continue
		}
		if opts.query == "" {
			opts.query = a
		}
	}
	return opts
}

func parseGmailGetOptions(args []string) gmailGetOptions {
	opts := gmailGetOptions{format: "metadata"}
	for _, a := range args {
		if a == "--json" {
			opts.jsonOutput = true
		}
		if a == "--body" || a == "--full" {
			opts.bodyOutput = true
			opts.format = "full"
		}
	}
	return opts
}

func parseCalendarListOptions(args []string) calendarListOptions {
	opts := calendarListOptions{calendarID: "primary", max: 20}
	for _, a := range args {
		if a == "--json" {
			opts.jsonOutput = true
			continue
		}
		if strings.HasPrefix(a, "--calendar=") {
			opts.calendarID = strings.TrimPrefix(a, "--calendar=")
			continue
		}
		if strings.HasPrefix(a, "--max=") {
			if n, err := strconv.ParseInt(strings.TrimPrefix(a, "--max="), 10, 64); err == nil {
				opts.max = n
			}
		}
	}
	return opts
}

func gmailRows(msgs []gm.Message) [][]string {
	rows := make([][]string, 0, len(msgs))
	for _, m := range msgs {
		rows = append(rows, []string{output.Truncate(m.Date, 31), output.Truncate(m.From, 36), output.Truncate(m.Subject, 72)})
	}
	return rows
}

func gmailSubjectRows(msgs []gm.Message) [][]string {
	rows := make([][]string, 0, len(msgs))
	for _, m := range msgs {
		rows = append(rows, []string{output.Truncate(m.Date, 31), output.Truncate(m.Subject, 80)})
	}
	return rows
}

func calendarRows(events []calpkg.Event) [][]string {
	rows := make([][]string, 0, len(events))
	for _, e := range events {
		rows = append(rows, []string{output.Truncate(e.Start, 25), output.Truncate(e.Summary, 60), output.Truncate(e.ID, 28)})
	}
	return rows
}

func enrichedMeetingRows(items []meet.EnrichedMeeting) [][]string {
	rows := make([][]string, 0, len(items))
	for _, item := range items {
		matched := "No"
		if item.MatchedEvent != nil {
			matched = output.Truncate(item.MatchedEvent.Summary, 42)
		}
		status := "ok"
		if item.NeedsReview {
			status = "review"
		}
		rows = append(rows, []string{output.Truncate(item.Meeting.Title, 42), matched, fmt.Sprintf("%d", item.MatchScore), status})
	}
	return rows
}

type groupedSearchResult struct {
	MeetingID    string
	MeetingTitle string
	BestScore    float64
	Matches      []embeddings.SearchResult
}

func groupSearchResults(results []embeddings.SearchResult, titles map[string]string) []groupedSearchResult {
	grouped := map[string]*groupedSearchResult{}
	for _, r := range results {
		g := grouped[r.MeetingID]
		if g == nil {
			title := titles[r.MeetingID]
			if title == "" {
				title = r.MeetingTitle
			}
			g = &groupedSearchResult{MeetingID: r.MeetingID, MeetingTitle: title}
			grouped[r.MeetingID] = g
		}
		if r.Score > g.BestScore {
			g.BestScore = r.Score
		}
		g.Matches = append(g.Matches, r)
	}
	out := make([]groupedSearchResult, 0, len(grouped))
	for _, g := range grouped {
		sort.Slice(g.Matches, func(i, j int) bool { return g.Matches[i].Score > g.Matches[j].Score })
		out = append(out, *g)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].BestScore > out[j].BestScore })
	return out
}

func renderProseMirrorText(doc *api.ProseMirrorDoc) string {
	if doc == nil {
		return ""
	}
	parts := make([]string, 0)
	for _, node := range doc.Content {
		if text := strings.TrimSpace(renderProseMirrorNode(node)); text != "" {
			parts = append(parts, text)
		}
	}
	return strings.TrimSpace(strings.Join(parts, "\n\n"))
}

func renderProseMirrorNode(node api.ProseMirrorNode) string {
	if strings.TrimSpace(node.Text) != "" {
		return node.Text
	}
	childParts := make([]string, 0)
	for _, child := range node.Content {
		if text := strings.TrimSpace(renderProseMirrorNode(child)); text != "" {
			childParts = append(childParts, text)
		}
	}
	text := strings.Join(childParts, " ")
	switch node.Type {
	case "heading":
		return strings.TrimSpace(text)
	case "bulletList", "orderedList":
		return strings.Join(childParts, "\n")
	case "listItem":
		if text == "" {
			return ""
		}
		return "- " + strings.TrimSpace(text)
	default:
		return strings.TrimSpace(text)
	}
}
