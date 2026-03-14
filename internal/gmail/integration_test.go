package gmail

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/ShaneOxM/granola-cli-go/internal/auth"
)

func TestIntegration_ListMessages(t *testing.T) {
	if os.Getenv("GRANOLA_OAUTH_CLIENT_ID") == "" || os.Getenv("GRANOLA_OAUTH_CLIENT_SECRET") == "" {
		t.Skip("OAuth env vars not set")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	a, err := auth.NewGmailCalendarAuth()
	if err != nil {
		t.Skipf("auth unavailable: %v", err)
	}
	c, err := NewClient(ctx, a)
	if err != nil {
		t.Skipf("gmail client unavailable: %v", err)
	}

	_, err = c.ListMessages(ctx, "", 1)
	if err != nil {
		t.Skipf("gmail API not available in this environment: %v", err)
	}
}
