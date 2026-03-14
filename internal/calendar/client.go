package calendar

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/ShaneOxM/granola-cli-go/internal/auth"
)

var baseURL = "https://www.googleapis.com/calendar/v3"

type Event struct {
	ID          string `json:"id"`
	Summary     string `json:"summary"`
	Description string `json:"description,omitempty"`
	Location    string `json:"location,omitempty"`
	Start       string `json:"start"`
	End         string `json:"end"`
	CalendarID  string `json:"calendar_id"`
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
	Items []eventResponse `json:"items"`
}

type eventResponse struct {
	ID          string `json:"id"`
	Summary     string `json:"summary"`
	Description string `json:"description"`
	Location    string `json:"location"`
	Start       struct {
		DateTime string `json:"dateTime"`
		Date     string `json:"date"`
	} `json:"start"`
	End struct {
		DateTime string `json:"dateTime"`
		Date     string `json:"date"`
	} `json:"end"`
}

func (c *Client) ListEvents(ctx context.Context, calendarID string, start, end time.Time, max int64) ([]Event, error) {
	calendarID, start, end, max = normalizeListParams(calendarID, start, end, max)
	vals := url.Values{}
	vals.Set("timeMin", start.Format(time.RFC3339))
	vals.Set("timeMax", end.Format(time.RFC3339))
	vals.Set("singleEvents", "true")
	vals.Set("orderBy", "startTime")
	vals.Set("maxResults", fmt.Sprintf("%d", max))

	endpoint := fmt.Sprintf("%s/calendars/%s/events?%s", baseURL, url.PathEscape(calendarID), vals.Encode())
	var resp listResponse
	if err := c.getJSON(ctx, endpoint, &resp); err != nil {
		return nil, fmt.Errorf("calendar list: %w", err)
	}
	out := make([]Event, 0, len(resp.Items))
	for _, item := range resp.Items {
		out = append(out, toEvent(item, calendarID))
	}
	return out, nil
}

func normalizeListParams(calendarID string, start, end time.Time, max int64) (string, time.Time, time.Time, int64) {
	if calendarID == "" {
		calendarID = "primary"
	}
	if max <= 0 {
		max = 20
	}
	if start.IsZero() {
		start = time.Now()
	}
	if end.IsZero() {
		end = start.Add(7 * 24 * time.Hour)
	}
	return calendarID, start, end, max
}

func (c *Client) GetEvent(ctx context.Context, calendarID, eventID string) (*Event, error) {
	if calendarID == "" {
		calendarID = "primary"
	}
	if eventID == "" {
		return nil, fmt.Errorf("event id is required")
	}
	endpoint := fmt.Sprintf("%s/calendars/%s/events/%s", baseURL, url.PathEscape(calendarID), url.PathEscape(eventID))
	var raw eventResponse
	if err := c.getJSON(ctx, endpoint, &raw); err != nil {
		return nil, fmt.Errorf("calendar get: %w", err)
	}
	ev := toEvent(raw, calendarID)
	return &ev, nil
}

func toEvent(item eventResponse, calendarID string) Event {
	start := item.Start.DateTime
	if start == "" {
		start = item.Start.Date
	}
	end := item.End.DateTime
	if end == "" {
		end = item.End.Date
	}
	return Event{
		ID:          item.ID,
		Summary:     item.Summary,
		Description: item.Description,
		Location:    item.Location,
		Start:       start,
		End:         end,
		CalendarID:  calendarID,
	}
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
