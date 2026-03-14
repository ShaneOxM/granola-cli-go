package gmail

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/ShaneOxM/granola-cli-go/internal/auth"
)

var baseURL = "https://gmail.googleapis.com/gmail/v1"

type Message struct {
	ID       string `json:"id"`
	ThreadID string `json:"thread_id"`
	From     string `json:"from"`
	To       string `json:"to,omitempty"`
	Cc       string `json:"cc,omitempty"`
	Subject  string `json:"subject"`
	Date     string `json:"date"`
	Snippet  string `json:"snippet"`
	Body     string `json:"body,omitempty"`
	Internal string `json:"internal_date,omitempty"`
}

type Thread struct {
	ID       string    `json:"id"`
	Messages []Message `json:"messages"`
}

type Client struct {
	auth *auth.GmailCalendarAuth
	http *http.Client
}

func NewClient(ctx context.Context, a *auth.GmailCalendarAuth) (*Client, error) {
	hc, err := a.HTTPClient(ctx)
	if err != nil {
		return nil, err
	}
	return &Client{auth: a, http: hc}, nil
}

type listResponse struct {
	Messages []struct {
		ID string `json:"id"`
	} `json:"messages"`
	NextPageToken string `json:"nextPageToken"`
}

type messageResponse struct {
	ID           string `json:"id"`
	ThreadID     string `json:"threadId"`
	Snippet      string `json:"snippet"`
	InternalDate string `json:"internalDate"`
	Payload      struct {
		MimeType string `json:"mimeType"`
		Body     struct {
			Data string `json:"data"`
		} `json:"body"`
		Headers []struct {
			Name  string `json:"name"`
			Value string `json:"value"`
		} `json:"headers"`
		Parts []messagePart `json:"parts"`
	} `json:"payload"`
}

type messagePart struct {
	MimeType string `json:"mimeType"`
	Body     struct {
		Data string `json:"data"`
	} `json:"body"`
	Parts []messagePart `json:"parts"`
}

type threadResponse struct {
	ID       string            `json:"id"`
	Messages []messageResponse `json:"messages"`
}

func (c *Client) ListMessages(ctx context.Context, query string, max int64) ([]Message, error) {
	if max <= 0 {
		max = 20
	}
	vals := url.Values{}
	if query != "" {
		vals.Set("q", query)
	}
	vals.Set("maxResults", fmt.Sprintf("%d", max))

	var listed listResponse
	if err := c.getJSON(ctx, baseURL+"/users/me/messages?"+vals.Encode(), &listed); err != nil {
		return nil, fmt.Errorf("gmail list: %w", err)
	}

	out := make([]Message, 0, len(listed.Messages))
	for _, m := range listed.Messages {
		msg, err := c.GetMessage(ctx, m.ID, "metadata")
		if err != nil {
			continue
		}
		out = append(out, *msg)
	}
	return out, nil
}

func (c *Client) GetMessage(ctx context.Context, id, format string) (*Message, error) {
	if id == "" {
		return nil, fmt.Errorf("message id is required")
	}
	if format == "" {
		format = "metadata"
	}
	vals := url.Values{}
	vals.Set("format", format)
	vals.Add("metadataHeaders", "From")
	vals.Add("metadataHeaders", "Subject")
	vals.Add("metadataHeaders", "Date")

	var raw messageResponse
	if err := c.getJSON(ctx, baseURL+"/users/me/messages/"+url.PathEscape(id)+"?"+vals.Encode(), &raw); err != nil {
		return nil, fmt.Errorf("gmail get: %w", err)
	}
	m := toMessage(raw)
	return &m, nil
}

func (c *Client) GetThread(ctx context.Context, threadID string) (*Thread, error) {
	if threadID == "" {
		return nil, fmt.Errorf("thread id is required")
	}
	vals := url.Values{}
	vals.Set("format", "full")
	var raw threadResponse
	if err := c.getJSON(ctx, baseURL+"/users/me/threads/"+url.PathEscape(threadID)+"?"+vals.Encode(), &raw); err != nil {
		return nil, fmt.Errorf("gmail thread: %w", err)
	}
	out := &Thread{ID: raw.ID, Messages: make([]Message, 0, len(raw.Messages))}
	for _, msg := range raw.Messages {
		out.Messages = append(out.Messages, toMessage(msg))
	}
	return out, nil
}

func (c *Client) ListFromAttendee(ctx context.Context, attendeeEmail string, max int64) ([]Message, error) {
	if attendeeEmail == "" {
		return nil, fmt.Errorf("attendee email is required")
	}
	q := fmt.Sprintf("from:%s", attendeeEmail)
	return c.ListMessages(ctx, q, max)
}

func (c *Client) ListInvolvingPerson(ctx context.Context, person string, max int64) ([]Message, error) {
	if strings.TrimSpace(person) == "" {
		return nil, fmt.Errorf("person is required")
	}
	return c.ListMessages(ctx, BuildPersonQuery(person), max)
}

func (c *Client) AroundMeeting(ctx context.Context, attendeeEmails []string, around time.Time, max int64) ([]Message, error) {
	if len(attendeeEmails) == 0 {
		return []Message{}, nil
	}
	q := BuildAroundMeetingQuery(attendeeEmails, around)
	msgs, err := c.ListMessages(ctx, q, max)
	if err != nil {
		return nil, err
	}
	sort.SliceStable(msgs, func(i, j int) bool { return msgs[i].Date < msgs[j].Date })
	return msgs, nil
}

func BuildAroundMeetingQuery(attendeeEmails []string, around time.Time) string {
	start := around.Add(-24 * time.Hour).Format("2006/01/02")
	end := around.Add(24 * time.Hour).Format("2006/01/02")
	emails := make([]string, 0, len(attendeeEmails))
	for _, e := range attendeeEmails {
		e = strings.TrimSpace(e)
		if e != "" {
			emails = append(emails, fmt.Sprintf("from:%s", e))
		}
	}
	return fmt.Sprintf("(%s) after:%s before:%s", strings.Join(emails, " OR "), start, end)
}

func BuildPersonQuery(person string) string {
	p := strings.TrimSpace(person)
	return fmt.Sprintf("(from:%s OR to:%s OR cc:%s OR bcc:%s)", p, p, p, p)
}

func toMessage(m messageResponse) Message {
	msg := Message{ID: m.ID, ThreadID: m.ThreadID, Snippet: m.Snippet}
	for _, h := range m.Payload.Headers {
		switch strings.ToLower(h.Name) {
		case "from":
			msg.From = h.Value
		case "to":
			msg.To = h.Value
		case "cc":
			msg.Cc = h.Value
		case "subject":
			msg.Subject = h.Value
		case "date":
			msg.Date = h.Value
		}
	}
	msg.Body = extractBody(m.Payload)
	if m.InternalDate != "" {
		if ms, err := strconv.ParseInt(m.InternalDate, 10, 64); err == nil {
			msg.Internal = time.UnixMilli(ms).Format(time.RFC3339)
		}
	}
	return msg
}

func extractBody(payload struct {
	MimeType string `json:"mimeType"`
	Body     struct {
		Data string `json:"data"`
	} `json:"body"`
	Headers []struct {
		Name  string `json:"name"`
		Value string `json:"value"`
	} `json:"headers"`
	Parts []messagePart `json:"parts"`
}) string {
	if body := decodeBody(payload.Body.Data); body != "" {
		return cleanBody(body, payload.MimeType)
	}
	for _, p := range payload.Parts {
		if body := extractBodyFromPart(p, true); body != "" {
			return body
		}
	}
	for _, p := range payload.Parts {
		if body := extractBodyFromPart(p, false); body != "" {
			return body
		}
	}
	return ""
}

func extractBodyFromPart(part messagePart, preferPlain bool) string {
	if preferPlain && part.MimeType != "text/plain" {
		for _, nested := range part.Parts {
			if body := extractBodyFromPart(nested, preferPlain); body != "" {
				return body
			}
		}
		return ""
	}
	if !preferPlain && part.MimeType != "text/html" && part.MimeType != "text/plain" {
		for _, nested := range part.Parts {
			if body := extractBodyFromPart(nested, preferPlain); body != "" {
				return body
			}
		}
		return ""
	}
	body := decodeBody(part.Body.Data)
	if body != "" {
		return cleanBody(body, part.MimeType)
	}
	for _, nested := range part.Parts {
		if body := extractBodyFromPart(nested, preferPlain); body != "" {
			return body
		}
	}
	return ""
}

func decodeBody(data string) string {
	if data == "" {
		return ""
	}
	decoded, err := base64.URLEncoding.DecodeString(data)
	if err != nil {
		decoded, err = base64.RawURLEncoding.DecodeString(data)
		if err != nil {
			return ""
		}
	}
	return strings.TrimSpace(string(decoded))
}

func cleanBody(body, mimeType string) string {
	if strings.Contains(mimeType, "html") {
		replacer := strings.NewReplacer("<br>", "\n", "<br/>", "\n", "<br />", "\n", "</p>", "\n\n")
		body = replacer.Replace(body)
		var out strings.Builder
		inTag := false
		for _, r := range body {
			switch r {
			case '<':
				inTag = true
			case '>':
				inTag = false
			default:
				if !inTag {
					out.WriteRune(r)
				}
			}
		}
		body = out.String()
	}
	lines := strings.Split(body, "\n")
	trimmed := make([]string, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			trimmed = append(trimmed, line)
		}
	}
	return strings.Join(trimmed, "\n")
}

func (c *Client) getJSON(ctx context.Context, endpoint string, out any) error {
	var lastErr error
	for attempt := 0; attempt < 3; attempt++ {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
		if err != nil {
			return err
		}
		if c.auth != nil {
			if qp := c.auth.QuotaProject(); qp != "" {
				req.Header.Set("x-goog-user-project", qp)
			}
		}
		resp, err := c.http.Do(req)
		if err != nil {
			lastErr = err
			time.Sleep(time.Duration(1<<attempt) * time.Second)
			continue
		}
		defer resp.Body.Close()
		if resp.StatusCode == http.StatusTooManyRequests {
			time.Sleep(time.Duration(1<<attempt) * time.Second)
			lastErr = fmt.Errorf("rate limited")
			continue
		}
		if resp.StatusCode >= 400 {
			body, _ := io.ReadAll(resp.Body)
			return fmt.Errorf("http %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
		}
		return json.NewDecoder(resp.Body).Decode(out)
	}
	if lastErr != nil {
		return lastErr
	}
	return fmt.Errorf("request failed")
}
