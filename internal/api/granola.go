package api

import (
	"encoding/json"
	"fmt"
	"net/mail"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/ShaneOxM/granola-cli-go/internal/auth"
)

var (
	uuidPattern         = regexp.MustCompile(`^[a-f0-9]{8}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{12}$`)
	alphanumericPattern = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)
	emailPattern        = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	searchQueryPattern  = regexp.MustCompile(`^[a-zA-Z0-9\s\-_.,'!?]+$`)
)

func validateID(id string) error {
	if id == "" {
		return fmt.Errorf("ID cannot be empty")
	}
	if !uuidPattern.MatchString(id) {
		return fmt.Errorf("invalid ID format: %s", id)
	}
	return nil
}

func validateWorkspace(workspace string) error {
	if workspace == "" {
		return nil
	}
	if !alphanumericPattern.MatchString(workspace) {
		return fmt.Errorf("invalid workspace ID: %s", workspace)
	}
	return nil
}

func validateFolder(folder string) error {
	if folder == "" {
		return nil
	}
	if !alphanumericPattern.MatchString(folder) {
		return fmt.Errorf("invalid folder path: %s", folder)
	}
	return nil
}

func validateEmail(email string) error {
	if email == "" {
		return nil
	}
	_, err := mail.ParseAddress(email)
	if err != nil {
		return fmt.Errorf("invalid email address: %w", err)
	}
	return nil
}

func validateSearch(query string) error {
	if query == "" {
		return nil
	}
	if len(query) > 500 {
		return fmt.Errorf("search query too long (max 500 characters)")
	}
	// More restrictive pattern - only allow alphanumeric and safe punctuation
	searchQueryPattern := regexp.MustCompile(`^[a-zA-Z0-9\s\-_.,'!?]+$`)
	if !searchQueryPattern.MatchString(query) {
		return fmt.Errorf("search query contains invalid characters")
	}
	// Additional check for potential injection patterns
	injectionPatterns := []string{"'", `"`, ";", "--", "/*", "*/", "UNION", "SELECT", "DROP"}
	for _, pattern := range injectionPatterns {
		if strings.Contains(strings.ToUpper(query), pattern) {
			return fmt.Errorf("search query contains potentially dangerous characters")
		}
	}
	return nil
}

type Client struct {
	BinaryPath string
}

func NewClient() *Client {
	c := &Client{}
	c.BinaryPath = c.findGranolaBinary()
	return c
}

func (c *Client) findGranolaBinary() string {
	if path := os.Getenv("GRANOLA_LEGACY_BINARY"); path != "" {
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}

	if path, err := exec.LookPath("granola-node"); err == nil {
		return path
	}

	if path, err := exec.LookPath("granola"); err == nil {
		return path
	}

	locations := []string{
		"/opt/homebrew/bin/granola-node",
		"/opt/homebrew/bin/granola",
		"/usr/local/bin/granola",
		"/usr/bin/granola",
		filepath.Join(os.Getenv("HOME"), "go/bin/granola"),
	}

	for _, loc := range locations {
		if _, err := os.Stat(loc); err == nil {
			return loc
		}
	}

	return "granola"
}

func (c *Client) ListMeetings(limit int, workspace, folder, search, attendee string) ([]Meeting, error) {
	if limit <= 0 || limit > 1000 {
		return nil, fmt.Errorf("invalid limit: must be between 1 and 1000")
	}
	if err := validateWorkspace(workspace); err != nil {
		return nil, err
	}
	if err := validateFolder(folder); err != nil {
		return nil, err
	}
	if err := validateSearch(search); err != nil {
		return nil, err
	}
	if err := validateEmail(attendee); err != nil {
		return nil, err
	}

	h, err := NewHttpClient()
	if err != nil {
		return nil, err
	}
	body := map[string]interface{}{
		"limit":                     limit,
		"offset":                    0,
		"include_last_viewed_panel": false,
	}
	if workspace != "" {
		body["workspace_id"] = workspace
	}
	resp, err := h.post("/v2/get-documents", body)
	if err != nil {
		return nil, fmt.Errorf("granola API list meetings: %w", err)
	}
	var wrapped struct {
		Docs []Meeting `json:"docs"`
	}
	if err := decodeResult(resp, &wrapped); err != nil {
		return nil, err
	}
	meetings := wrapped.Docs
	if search != "" || attendee != "" {
		meetings = filterMeetings(meetings, search, attendee)
	}
	if folder != "" {
		meetings, err = c.listMeetingsForFolder(h, folder, workspace, search, attendee, limit)
		if err != nil {
			return nil, err
		}
	}
	if len(meetings) > limit {
		meetings = meetings[:limit]
	}
	return meetings, nil
}

func (c *Client) GetMeeting(id string) (*Meeting, error) {
	if err := validateID(id); err != nil {
		return nil, err
	}

	h, err := NewHttpClient()
	if err != nil {
		return nil, fmt.Errorf("get meeting: %w", err)
	}
	resp, err := h.post("/v1/get-document-metadata", map[string]interface{}{"document_id": id})
	if err != nil {
		return nil, fmt.Errorf("get meeting: %w", err)
	}
	var m Meeting
	if err := decodeResult(resp, &m); err != nil {
		return nil, fmt.Errorf("parse meeting: %v", err)
	}
	m.ID = id
	if m.Title == "" || m.CreatedAt == "" || m.UpdatedAt == "" {
		if enriched, err := c.findMeetingViaDocuments(h, id); err == nil && enriched != nil {
			if m.Title == "" {
				m.Title = enriched.Title
			}
			if m.CreatedAt == "" {
				m.CreatedAt = enriched.CreatedAt
			}
			if m.UpdatedAt == "" {
				m.UpdatedAt = enriched.UpdatedAt
			}
			if m.WorkspaceID == "" {
				m.WorkspaceID = enriched.WorkspaceID
			}
		}
	}
	return &m, nil
}

func (c *Client) GetTranscript(id string) ([]Utterance, error) {
	if err := validateID(id); err != nil {
		return nil, err
	}

	h, err := NewHttpClient()
	if err != nil {
		return nil, err
	}
	resp, err := h.post("/v1/get-document-transcript", map[string]interface{}{"document_id": id})
	if err != nil {
		return nil, fmt.Errorf("get transcript: %w", err)
	}
	var transcript []Utterance
	if err := decodeResult(resp, &transcript); err != nil {
		return nil, fmt.Errorf("parse transcript: %v", err)
	}
	return transcript, nil
}

func (c *Client) GetNotes(id string) (*ProseMirrorDoc, error) {
	if err := validateID(id); err != nil {
		return nil, err
	}

	h, err := NewHttpClient()
	if err != nil {
		return nil, err
	}
	resp, err := h.post("/v1/get-document-metadata", map[string]interface{}{"document_id": id})
	if err != nil {
		return nil, fmt.Errorf("get notes: %v", err)
	}
	var wrapped struct {
		Notes ProseMirrorDoc `json:"notes"`
	}
	if err := decodeResult(resp, &wrapped); err != nil {
		return nil, fmt.Errorf("parse notes: %v", err)
	}
	return &wrapped.Notes, nil
}

func (c *Client) ListWorkspaces() ([]Workspace, error) {
	h, err := NewHttpClient()
	if err != nil {
		return nil, err
	}
	resp, err := h.post("/v1/get-workspaces", map[string]interface{}{})
	if err != nil {
		return nil, fmt.Errorf("list workspaces: %w", err)
	}
	var wrapped struct {
		Workspaces []struct {
			Workspace struct {
				ID        string `json:"workspace_id"`
				Slug      string `json:"slug"`
				Name      string `json:"display_name"`
				CreatedAt string `json:"created_at"`
				UpdatedAt string `json:"updated_at"`
			} `json:"workspace"`
		} `json:"workspaces"`
	}
	if err := decodeResult(resp, &wrapped); err != nil {
		return nil, fmt.Errorf("parse workspaces: %v", err)
	}
	workspaces := make([]Workspace, 0, len(wrapped.Workspaces))
	for _, item := range wrapped.Workspaces {
		workspaces = append(workspaces, Workspace{
			ID:        item.Workspace.ID,
			Name:      item.Workspace.Name,
			Slug:      item.Workspace.Slug,
			CreatedAt: item.Workspace.CreatedAt,
			UpdatedAt: item.Workspace.UpdatedAt,
		})
	}
	return workspaces, nil
}

func (c *Client) GetWorkspace(id string) (*Workspace, error) {
	workspaces, err := c.ListWorkspaces()
	if err != nil {
		return nil, err
	}
	for _, w := range workspaces {
		if w.ID == id {
			return &w, nil
		}
	}
	return nil, fmt.Errorf("workspace not found: %s", id)
}

func (c *Client) ListFolders(workspace string) ([]Folder, error) {
	if err := validateWorkspace(workspace); err != nil {
		return nil, err
	}

	h, err := NewHttpClient()
	if err != nil {
		return nil, err
	}
	resp, err := h.post("/v2/get-document-lists", map[string]interface{}{})
	if err != nil {
		return nil, fmt.Errorf("list folders: %w", err)
	}
	var wrapped struct {
		Lists []struct {
			ID          string `json:"id"`
			Title       string `json:"title"`
			WorkspaceID string `json:"workspace_id"`
			CreatedAt   string `json:"created_at"`
			UpdatedAt   string `json:"updated_at"`
		} `json:"lists"`
	}
	if err := decodeResult(resp, &wrapped); err != nil {
		return nil, fmt.Errorf("parse folders: %v", err)
	}
	folders := make([]Folder, 0, len(wrapped.Lists))
	for _, item := range wrapped.Lists {
		folders = append(folders, Folder{
			ID:          item.ID,
			Name:        item.Title,
			WorkspaceID: item.WorkspaceID,
			CreatedAt:   item.CreatedAt,
			UpdatedAt:   item.UpdatedAt,
		})
	}
	if workspace == "" {
		return folders, nil
	}
	filtered := make([]Folder, 0, len(folders))
	for _, f := range folders {
		if f.WorkspaceID == workspace {
			filtered = append(filtered, f)
		}
	}
	return filtered, nil
}

func (c *Client) GetFolder(id string) (*Folder, error) {
	folders, err := c.ListFolders("")
	if err != nil {
		return nil, err
	}
	for _, f := range folders {
		if f.ID == id {
			return &f, nil
		}
	}
	return nil, fmt.Errorf("folder not found: %s", id)
}

func decodeResult(src interface{}, dst interface{}) error {
	b, err := json.Marshal(src)
	if err != nil {
		return fmt.Errorf("marshal API result: %w", err)
	}
	if err := json.Unmarshal(b, dst); err != nil {
		return fmt.Errorf("decode API result: %w", err)
	}
	return nil
}

func filterMeetings(meetings []Meeting, search, attendee string) []Meeting {
	if search == "" && attendee == "" {
		return meetings
	}
	filtered := make([]Meeting, 0, len(meetings))
	for _, m := range meetings {
		if search != "" && !strings.Contains(strings.ToLower(m.Title), strings.ToLower(search)) {
			continue
		}
		if attendee != "" {
			matched := false
			for _, a := range m.Attendees {
				name := strings.ToLower(a.Name)
				email := strings.ToLower(a.Email)
				q := strings.ToLower(attendee)
				if strings.Contains(name, q) || strings.Contains(email, q) {
					matched = true
					break
				}
			}
			if !matched {
				continue
			}
		}
		filtered = append(filtered, m)
	}
	return filtered
}

func (c *Client) listMeetingsForFolder(h *HttpClient, folder, workspace, search, attendee string, limit int) ([]Meeting, error) {
	resp, err := h.post("/v2/get-document-lists", map[string]interface{}{})
	if err != nil {
		return nil, err
	}
	var folders []struct {
		ID          string   `json:"id"`
		WorkspaceID string   `json:"workspace_id"`
		DocumentIDs []string `json:"document_ids"`
		Documents   []struct {
			ID string `json:"id"`
		} `json:"documents"`
	}
	if err := decodeResult(resp, &folders); err != nil {
		return nil, err
	}
	var ids []string
	for _, f := range folders {
		if f.ID == folder {
			ids = append(ids, f.DocumentIDs...)
			for _, d := range f.Documents {
				ids = append(ids, d.ID)
			}
			break
		}
	}
	if len(ids) == 0 {
		return []Meeting{}, nil
	}
	resp, err = h.post("/v1/get-documents-batch", map[string]interface{}{"document_ids": ids, "include_last_viewed_panel": false})
	if err != nil {
		return nil, err
	}
	var wrapped struct {
		Documents []Meeting `json:"documents"`
		Docs      []Meeting `json:"docs"`
	}
	if err := decodeResult(resp, &wrapped); err != nil {
		return nil, err
	}
	meetings := wrapped.Documents
	if len(meetings) == 0 {
		meetings = wrapped.Docs
	}
	if workspace != "" {
		filtered := meetings[:0]
		for _, m := range meetings {
			if m.WorkspaceID == workspace {
				filtered = append(filtered, m)
			}
		}
		meetings = filtered
	}
	meetings = filterMeetings(meetings, search, attendee)
	if len(meetings) > limit {
		meetings = meetings[:limit]
	}
	return meetings, nil
}

func (c *Client) findMeetingViaDocuments(h *HttpClient, id string) (*Meeting, error) {
	offset := 0
	for page := 0; page < 100; page++ {
		resp, err := h.post("/v2/get-documents", map[string]interface{}{
			"limit":                     50,
			"offset":                    offset,
			"include_last_viewed_panel": false,
		})
		if err != nil {
			return nil, err
		}
		var wrapped struct {
			Docs []Meeting `json:"docs"`
		}
		if err := decodeResult(resp, &wrapped); err != nil {
			return nil, err
		}
		if len(wrapped.Docs) == 0 {
			return nil, nil
		}
		for _, m := range wrapped.Docs {
			if m.ID == id {
				return &m, nil
			}
		}
		offset += 50
	}
	return nil, nil
}

func (c *Client) GMailSearch(query string) ([]Email, error) {
	if err := validateSearch(query); err != nil {
		return nil, err
	}

	cmd, err := c.createCommand([]string{"--no-pager", "gmail", "search", query, "--output", "json"})
	if err != nil {
		return nil, err
	}
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("gmail search: %w", err)
	}

	var emails []Email
	if err := json.Unmarshal(output, &emails); err != nil {
		return nil, fmt.Errorf("parse emails: %v", err)
	}

	return emails, nil
}

func (c *Client) createCommand(args []string) (*exec.Cmd, error) {
	sanitizedArgs, err := sanitizeArgs(args)
	if err != nil {
		return nil, fmt.Errorf("invalid arguments: %w", err)
	}

	cmd := exec.Command(c.BinaryPath, sanitizedArgs...)
	cmd.Env = os.Environ()

	authEnv, err := auth.GetAuthEnvVars()
	if err == nil {
		for k, v := range authEnv {
			cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
		}
	}

	return cmd, nil
}

func sanitizeArgs(args []string) ([]string, error) {
	if len(args) == 0 {
		return args, nil
	}

	sanitized := make([]string, 0, len(args))
	for i, arg := range args {
		if arg == "" {
			continue
		}

		if strings.Contains(arg, ";") || strings.Contains(arg, "|") || strings.Contains(arg, "&") ||
			strings.Contains(arg, "$") || strings.Contains(arg, "`") || strings.Contains(arg, "\n") ||
			strings.Contains(arg, "\r") || strings.Contains(arg, "\x00") {
			return nil, fmt.Errorf("argument %d contains invalid characters: %s", i, arg)
		}

		if strings.Contains(arg, " ") && !strings.HasPrefix(arg, "--") {
			return nil, fmt.Errorf("argument %d contains spaces: %s", i, arg)
		}

		sanitized = append(sanitized, arg)
	}

	return sanitized, nil
}

func GetCurrentUserEmail() (string, error) {
	creds, err := auth.GetCredentials()
	if err != nil {
		return "", err
	}

	return creds.Email(), nil
}

type Meeting struct {
	ID          string   `json:"id"`
	Title       string   `json:"title"`
	CreatedAt   string   `json:"created_at"`
	UpdatedAt   string   `json:"updated_at"`
	WorkspaceID string   `json:"workspace_id,omitempty"`
	Attendees   []Person `json:"attendees,omitempty"`
	Creator     Person   `json:"creator,omitempty"`
}

type Utterance struct {
	Source         string `json:"source"`
	Text           string `json:"text"`
	StartTimestamp string `json:"start_timestamp"`
	EndTimestamp   string `json:"end_timestamp"`
}

type ProseMirrorMark struct {
	Type string `json:"type"`
}

// ProseMirrorAttrs represents typed attributes for ProseMirror nodes
type ProseMirrorAttrs struct {
	Style    string            `json:"style,omitempty"`
	Class    string            `json:"class,omitempty"`
	Heading  int               `json:"heading,omitempty"`
	ListType string            `json:"list_type,omitempty"`
	Custom   map[string]string `json:"custom,omitempty"`
}

// ProseMirrorDoc represents a ProseMirror document

type ProseMirrorDoc struct {
	Type    string            `json:"type"`
	Content []ProseMirrorNode `json:"content"`
}

type ProseMirrorNode struct {
	Type       string            `json:"type"`
	Text       string            `json:"text,omitempty"`
	Mark       []ProseMirrorMark `json:"mark,omitempty"`
	Content    []ProseMirrorNode `json:"content,omitempty"`
	Attributes ProseMirrorAttrs  `json:"attrs,omitempty"`
}

type Person struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

type Workspace struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Slug      string `json:"slug"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

type Folder struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Slug        string `json:"slug"`
	WorkspaceID string `json:"workspace_id"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

type Email struct {
	ID          string   `json:"id"`
	Subject     string   `json:"subject"`
	From        Person   `json:"from"`
	To          []Person `json:"to"`
	Body        string   `json:"body"`
	ThreadID    string   `json:"thread_id"`
	CreatedAt   string   `json:"created_at"`
	Attachments []string `json:"attachments,omitempty"`
}
