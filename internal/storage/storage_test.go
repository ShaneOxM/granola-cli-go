package storage

import (
	"context"
	"testing"
	"time"

	"github.com/ShaneOxM/granola-cli-go/internal/embeddings"
)

func TestNewDB(t *testing.T) {
	// Use a temporary database file
	tempFile := t.TempDir() + "/test.db"

	db, err := NewDB(tempFile)
	if err != nil {
		t.Fatalf("NewDB() error = %v", err)
	}
	defer db.Close()

	if db == nil {
		t.Fatal("NewDB() returned nil")
	}
	if _, err := db.db.Exec(`PRAGMA journal_mode`); err != nil {
		t.Fatalf("expected usable db pragma: %v", err)
	}
}

func TestMeeting(t *testing.T) {
	meeting := Meeting{
		ID:          "550e8400-e29b-41d4-a716-446655440000",
		Title:       "Test Meeting",
		CreatedAt:   "2024-01-01T00:00:00Z",
		UpdatedAt:   "2024-01-02T00:00:00Z",
		WorkspaceID: "workspace-123",
	}

	if meeting.ID != "550e8400-e29b-41d4-a716-446655440000" {
		t.Errorf("ID = %v, want 550e8400-e29b-41d4-a716-446655440000", meeting.ID)
	}

	if meeting.Title != "Test Meeting" {
		t.Errorf("Title = %v, want Test Meeting", meeting.Title)
	}

	if meeting.WorkspaceID != "workspace-123" {
		t.Errorf("WorkspaceID = %v, want workspace-123", meeting.WorkspaceID)
	}
}

func TestEmptyMeeting(t *testing.T) {
	meeting := Meeting{}

	if meeting.ID != "" {
		t.Errorf("ID = %v, want empty", meeting.ID)
	}

	if meeting.Title != "" {
		t.Errorf("Title = %v, want empty", meeting.Title)
	}
}

func TestContext(t *testing.T) {
	ctx := Context{
		ID:        "meeting-123:note-1",
		MeetingID: "meeting-123",
		Type:      "note",
		Content:   "This is a test note",
	}

	if ctx.ID != "meeting-123:note-1" {
		t.Errorf("ID = %v, want meeting-123:note-1", ctx.ID)
	}

	if ctx.Type != "note" {
		t.Errorf("Type = %v, want note", ctx.Type)
	}

	if ctx.Content != "This is a test note" {
		t.Errorf("Content = %v, want This is a test note", ctx.Content)
	}
}

func TestProgression(t *testing.T) {
	prog := Progression{
		ID:          "meeting-123:stage-1",
		MeetingID:   "meeting-123",
		Stage:       "planning",
		Description: "Initial planning phase",
	}

	if prog.ID != "meeting-123:stage-1" {
		t.Errorf("ID = %v, want meeting-123:stage-1", prog.ID)
	}

	if prog.Stage != "planning" {
		t.Errorf("Stage = %v, want planning", prog.Stage)
	}
}

func TestEmailLinks(t *testing.T) {
	tempFile := t.TempDir() + "/test.db"
	db, err := NewDB(tempFile)
	if err != nil {
		t.Fatalf("NewDB() error = %v", err)
	}
	defer db.Close()

	if err := db.SaveEmailLink("meeting-1", "email-1", "thread-1", "around_meeting", 2); err != nil {
		t.Fatalf("SaveEmailLink() error = %v", err)
	}
	links, err := db.GetEmailLinks("meeting-1")
	if err != nil {
		t.Fatalf("GetEmailLinks() error = %v", err)
	}
	if len(links) != 1 || links[0].EmailID != "email-1" {
		t.Fatalf("unexpected email links: %+v", links)
	}
}

func TestCalendarLinks(t *testing.T) {
	db, err := NewDB(t.TempDir() + "/test.db")
	if err != nil {
		t.Fatalf("NewDB() error = %v", err)
	}
	defer db.Close()
	if err := db.SaveCalendarLink("meeting-1", "event-1", "Summary", "title+attendees", 2); err != nil {
		t.Fatalf("SaveCalendarLink() error = %v", err)
	}
	links, err := db.GetCalendarLinks("meeting-1")
	if err != nil {
		t.Fatalf("GetCalendarLinks() error = %v", err)
	}
	if len(links) != 1 || links[0].EventID != "event-1" {
		t.Fatalf("unexpected calendar links: %+v", links)
	}
}

func TestMeetingCRUDAndList(t *testing.T) {
	db, err := NewDB(t.TempDir() + "/test.db")
	if err != nil {
		t.Fatalf("NewDB error: %v", err)
	}
	defer db.Close()
	m := &Meeting{ID: "m1", Title: "Meeting 1", CreatedAt: "2026-03-13T12:00:00Z", UpdatedAt: "2026-03-13T12:05:00Z", WorkspaceID: "w1"}
	if err := db.SaveMeeting(context.Background(), m); err != nil {
		t.Fatalf("SaveMeeting error: %v", err)
	}
	got, err := db.GetMeeting(context.Background(), "m1")
	if err != nil || got.Title != "Meeting 1" {
		t.Fatalf("GetMeeting unexpected: %+v err=%v", got, err)
	}
	items, err := db.ListMeetings(context.Background(), 10)
	if err != nil || len(items) != 1 {
		t.Fatalf("ListMeetings unexpected len=%d err=%v", len(items), err)
	}
}

func TestContextAndProgression(t *testing.T) {
	db, err := NewDB(t.TempDir() + "/test.db")
	if err != nil {
		t.Fatalf("NewDB error: %v", err)
	}
	defer db.Close()
	if err := db.AttachContext("m1", "note", "content"); err != nil {
		t.Fatalf("AttachContext error: %v", err)
	}
	ctxs, err := db.GetContext("m1")
	if err != nil || len(ctxs) != 1 {
		t.Fatalf("GetContext unexpected len=%d err=%v", len(ctxs), err)
	}
	if err := db.AddProgression("m1", "stage", "desc"); err != nil {
		t.Fatalf("AddProgression error: %v", err)
	}
	prog, err := db.GetProgression("m1")
	if err != nil || len(prog) != 1 {
		t.Fatalf("GetProgression unexpected len=%d err=%v", len(prog), err)
	}
}

func TestEmbeddingsLifecycle(t *testing.T) {
	db, err := NewDB(t.TempDir() + "/test.db")
	if err != nil {
		t.Fatalf("NewDB error: %v", err)
	}
	defer db.Close()
	if err := db.InitEmbeddings(); err != nil {
		t.Fatalf("InitEmbeddings error: %v", err)
	}
	meeting := &Meeting{ID: "m1", Title: "Semantic Meeting", CreatedAt: "2026-03-13T12:00:00Z", UpdatedAt: "2026-03-13T12:05:00Z"}
	if err := db.SaveMeeting(context.Background(), meeting); err != nil {
		t.Fatalf("SaveMeeting error: %v", err)
	}
	chunk := &EmbeddingChunk{
		ID:         "m1:chunk:0",
		MeetingID:  "m1",
		ChunkIndex: 0,
		ChunkText:  "project planning summary",
		Embedding:  embeddings.EmbedToBlob([]float32{1, 0, 0}),
		Dimensions: 3,
		Provider:   "test",
		Model:      "mock",
		CreatedAt:  time.Now().Unix(),
	}
	if err := db.SaveEmbeddingChunk(chunk); err != nil {
		t.Fatalf("SaveEmbeddingChunk error: %v", err)
	}
	chunks, err := db.GetChunksByMeeting("m1")
	if err != nil || len(chunks) != 1 {
		t.Fatalf("GetChunksByMeeting unexpected len=%d err=%v", len(chunks), err)
	}
	if err := db.UpdateMeetingEmbeddingStatus("m1", "complete", "test", "mock", 1); err != nil {
		t.Fatalf("UpdateMeetingEmbeddingStatus error: %v", err)
	}
	status, provider, model, chunkCount, err := db.GetMeetingEmbeddingStatus("m1")
	if err != nil || status != "complete" || provider != "test" || model != "mock" || chunkCount != 1 {
		t.Fatalf("unexpected embedding status=%q provider=%q model=%q chunkCount=%d err=%v", status, provider, model, chunkCount, err)
	}
	title, err := db.GetMeetingTitle("m1")
	if err != nil || title != "Semantic Meeting" {
		t.Fatalf("unexpected title %q err=%v", title, err)
	}
	stats := db.GetEmbeddingStats()
	if stats.Total != 1 || stats.Embedded != 1 || stats.TotalChunks != 1 {
		t.Fatalf("unexpected stats: %+v", stats)
	}
	all, err := db.GetAllChunks()
	if err != nil || len(all) != 1 {
		t.Fatalf("GetAllChunks unexpected len=%d err=%v", len(all), err)
	}
	res, err := db.SearchChunks([]float32{1, 0, 0}, 0.1, 5)
	if err != nil || len(res) != 1 {
		t.Fatalf("SearchChunks unexpected len=%d err=%v", len(res), err)
	}
	if err := db.ResetAllEmbeddings(); err != nil {
		t.Fatalf("ResetAllEmbeddings error: %v", err)
	}
	stats = db.GetEmbeddingStats()
	if stats.TotalChunks != 0 {
		t.Fatalf("expected chunks reset, stats=%+v", stats)
	}
}
