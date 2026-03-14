// Package storage provides SQLite database operations for granola-cli.

package storage

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/ShaneOxM/granola-cli-go/internal/logger"

	_ "modernc.org/sqlite"
)

type DB struct {
	db *sql.DB
}

func NewDB(path string) (*DB, error) {
	logger.Debug("Opening database", "path", path)
	db, err := sql.Open("sqlite", path)
	if err != nil {
		logger.Error("Failed to open database", "path", path, "error", err)
		return nil, err
	}

	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	db.SetConnMaxLifetime(0)

	d := &DB{db: db}
	if err := d.init(); err != nil {
		logger.Error("Database initialization failed", "path", path, "error", err)
		return nil, err
	}

	logger.Debug("Database opened successfully", "path", path)
	return d, nil
}

func (d *DB) init() error {
	if _, err := d.db.Exec(`PRAGMA busy_timeout = 5000`); err != nil {
		return err
	}
	if _, err := d.db.Exec(`PRAGMA journal_mode = WAL`); err != nil {
		return err
	}
	if _, err := d.db.Exec(`PRAGMA synchronous = NORMAL`); err != nil {
		return err
	}

	if _, err := d.db.Exec(`
		CREATE TABLE IF NOT EXISTS meetings (
			id TEXT PRIMARY KEY,
			title TEXT,
			created_at TEXT,
			updated_at TEXT,
			workspace_id TEXT,
			attendees TEXT,
			creator TEXT
		)
	`); err != nil {
		return err
	}

	if _, err := d.db.Exec(`
		CREATE TABLE IF NOT EXISTS context (
			id TEXT PRIMARY KEY,
			meeting_id TEXT,
			type TEXT,
			content TEXT,
			source_ref TEXT,
			created_at DATETIME
		)
	`); err != nil {
		return err
	}

	if _, err := d.db.Exec(`
		CREATE TABLE IF NOT EXISTS progression (
			id TEXT PRIMARY KEY,
			meeting_id TEXT,
			stage TEXT,
			description TEXT,
			created_at DATETIME
		)
	`); err != nil {
		return err
	}

	if _, err := d.db.Exec(`
		CREATE TABLE IF NOT EXISTS email_links (
			id TEXT PRIMARY KEY,
			meeting_id TEXT,
			email_id TEXT,
			thread_id TEXT,
			reason TEXT,
			score INTEGER,
			created_at DATETIME
		)
	`); err != nil {
		return err
	}

	if _, err := d.db.Exec(`
		CREATE TABLE IF NOT EXISTS calendar_links (
			id TEXT PRIMARY KEY,
			meeting_id TEXT,
			event_id TEXT,
			event_summary TEXT,
			reason TEXT,
			score INTEGER,
			created_at DATETIME
		)
	`); err != nil {
		return err
	}

	// Initialize embeddings
	return d.InitEmbeddings()
}

func (d *DB) Close() error {
	return d.db.Close()
}

type Meeting struct {
	ID          string
	Title       string
	CreatedAt   string
	UpdatedAt   string
	WorkspaceID string
	Attendees   sql.NullString
	Creator     sql.NullString
}

func (d *DB) SaveMeeting(ctx context.Context, m *Meeting) error {
	_, err := d.db.ExecContext(ctx, `
		INSERT OR REPLACE INTO meetings (id, title, created_at, updated_at, workspace_id, attendees, creator)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, m.ID, m.Title, m.CreatedAt, m.UpdatedAt, m.WorkspaceID, m.Attendees, m.Creator)
	return err
}

func (d *DB) GetMeeting(ctx context.Context, id string) (*Meeting, error) {
	var m Meeting
	err := d.db.QueryRowContext(ctx, `
		SELECT id, title, created_at, updated_at, workspace_id, attendees, creator
		FROM meetings WHERE id = ?
	`, id).Scan(&m.ID, &m.Title, &m.CreatedAt, &m.UpdatedAt, &m.WorkspaceID, &m.Attendees, &m.Creator)
	if err != nil {
		return nil, err
	}
	return &m, nil
}

func (d *DB) ListMeetings(ctx context.Context, limit int) ([]*Meeting, error) {
	rows, err := d.db.QueryContext(ctx, `
		SELECT id, title, created_at, updated_at, workspace_id, attendees, creator
		FROM meetings ORDER BY created_at DESC LIMIT ?
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var meetings []*Meeting
	for rows.Next() {
		var m Meeting
		if err := rows.Scan(&m.ID, &m.Title, &m.CreatedAt, &m.UpdatedAt, &m.WorkspaceID, &m.Attendees, &m.Creator); err != nil {
			return nil, err
		}
		meetings = append(meetings, &m)
	}
	return meetings, rows.Err()
}

type Context struct {
	ID        string
	MeetingID string
	Type      string
	Content   string
	SourceRef string
	CreatedAt time.Time
}

func (d *DB) AttachContext(meetingID, contextType, content string) error {
	id := fmt.Sprintf("%s:%s", meetingID, contextType)
	_, err := d.db.Exec(`
		INSERT OR REPLACE INTO context (id, meeting_id, type, content, source_ref, created_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`, id, meetingID, contextType, content, "", time.Now())
	return err
}

func (d *DB) GetContext(meetingID string) ([]*Context, error) {
	rows, err := d.db.Query(`
		SELECT id, meeting_id, type, content, source_ref, created_at
		FROM context WHERE meeting_id = ?
	`, meetingID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var contexts []*Context
	for rows.Next() {
		var c Context
		if err := rows.Scan(&c.ID, &c.MeetingID, &c.Type, &c.Content, &c.SourceRef, &c.CreatedAt); err != nil {
			return nil, err
		}
		contexts = append(contexts, &c)
	}
	return contexts, rows.Err()
}

type Progression struct {
	ID          string
	MeetingID   string
	Stage       string
	Description string
	CreatedAt   time.Time
}

type EmailLink struct {
	ID        string
	MeetingID string
	EmailID   string
	ThreadID  string
	Reason    string
	Score     int
	CreatedAt time.Time
}

type CalendarLink struct {
	ID           string
	MeetingID    string
	EventID      string
	EventSummary string
	Reason       string
	Score        int
	CreatedAt    time.Time
}

func (d *DB) AddProgression(meetingID, stage, description string) error {
	id := fmt.Sprintf("%s:%s", meetingID, stage)
	_, err := d.db.Exec(`
		INSERT INTO progression (id, meeting_id, stage, description, created_at)
		VALUES (?, ?, ?, ?, ?)
	`, id, meetingID, stage, description, time.Now())
	return err
}

func (d *DB) GetProgression(meetingID string) ([]*Progression, error) {
	rows, err := d.db.Query(`
		SELECT id, meeting_id, stage, description, created_at
		FROM progression WHERE meeting_id = ?
	`, meetingID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var progression []*Progression
	for rows.Next() {
		var p Progression
		if err := rows.Scan(&p.ID, &p.MeetingID, &p.Stage, &p.Description, &p.CreatedAt); err != nil {
			return nil, err
		}
		progression = append(progression, &p)
	}
	return progression, rows.Err()
}

func (d *DB) SaveEmailLink(meetingID, emailID, threadID, reason string, score int) error {
	id := fmt.Sprintf("%s:%s", meetingID, emailID)
	_, err := d.db.Exec(`
		INSERT OR REPLACE INTO email_links (id, meeting_id, email_id, thread_id, reason, score, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, id, meetingID, emailID, threadID, reason, score, time.Now())
	return err
}

func (d *DB) GetEmailLinks(meetingID string) ([]*EmailLink, error) {
	rows, err := d.db.Query(`
		SELECT id, meeting_id, email_id, thread_id, reason, score, created_at
		FROM email_links WHERE meeting_id = ?
		ORDER BY created_at DESC
	`, meetingID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var links []*EmailLink
	for rows.Next() {
		var l EmailLink
		if err := rows.Scan(&l.ID, &l.MeetingID, &l.EmailID, &l.ThreadID, &l.Reason, &l.Score, &l.CreatedAt); err != nil {
			return nil, err
		}
		links = append(links, &l)
	}
	return links, rows.Err()
}

func (d *DB) SaveCalendarLink(meetingID, eventID, eventSummary, reason string, score int) error {
	id := fmt.Sprintf("%s:%s", meetingID, eventID)
	_, err := d.db.Exec(`
		INSERT OR REPLACE INTO calendar_links (id, meeting_id, event_id, event_summary, reason, score, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, id, meetingID, eventID, eventSummary, reason, score, time.Now())
	return err
}

func (d *DB) GetCalendarLinks(meetingID string) ([]*CalendarLink, error) {
	rows, err := d.db.Query(`
		SELECT id, meeting_id, event_id, event_summary, reason, score, created_at
		FROM calendar_links WHERE meeting_id = ?
		ORDER BY created_at DESC
	`, meetingID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var links []*CalendarLink
	for rows.Next() {
		var l CalendarLink
		if err := rows.Scan(&l.ID, &l.MeetingID, &l.EventID, &l.EventSummary, &l.Reason, &l.Score, &l.CreatedAt); err != nil {
			return nil, err
		}
		links = append(links, &l)
	}
	return links, rows.Err()
}
