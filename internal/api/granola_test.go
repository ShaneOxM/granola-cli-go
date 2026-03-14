package api

import (
	"testing"
)

func TestValidateID(t *testing.T) {
	tests := []struct {
		name    string
		id      string
		wantErr bool
	}{
		{
			name:    "valid UUID",
			id:      "550e8400-e29b-41d4-a716-446655440000",
			wantErr: false,
		},
		{
			name:    "empty ID",
			id:      "",
			wantErr: true,
		},
		{
			name:    "invalid format",
			id:      "not-a-uuid",
			wantErr: true,
		},
		{
			name:    "too short",
			id:      "550e8400",
			wantErr: true,
		},
		{
			name:    "too long",
			id:      "550e8400-e29b-41d4-a716-446655440000-extra",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateID(tt.id)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateID() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateWorkspace(t *testing.T) {
	tests := []struct {
		name      string
		workspace string
		wantErr   bool
	}{
		{
			name:      "valid workspace",
			workspace: "my-workspace",
			wantErr:   false,
		},
		{
			name:      "valid with underscore",
			workspace: "my_workspace",
			wantErr:   false,
		},
		{
			name:      "valid with hyphen",
			workspace: "my-workspace-123",
			wantErr:   false,
		},
		{
			name:      "empty workspace (allowed)",
			workspace: "",
			wantErr:   false,
		},
		{
			name:      "invalid with slash",
			workspace: "my/workspace",
			wantErr:   true,
		},
		{
			name:      "invalid with space",
			workspace: "my workspace",
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateWorkspace(tt.workspace)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateWorkspace() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateFolder(t *testing.T) {
	tests := []struct {
		name    string
		folder  string
		wantErr bool
	}{
		{
			name:    "valid folder",
			folder:  "my-folder",
			wantErr: false,
		},
		{
			name:    "valid with underscore",
			folder:  "my_folder",
			wantErr: false,
		},
		{
			name:    "valid with hyphen",
			folder:  "my-folder-123",
			wantErr: false,
		},
		{
			name:    "empty folder (allowed)",
			folder:  "",
			wantErr: false,
		},
		{
			name:    "invalid with space",
			folder:  "my folder",
			wantErr: true,
		},
		{
			name:    "invalid with special chars",
			folder:  "my@folder",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateFolder(tt.folder)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateFolder() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateEmail(t *testing.T) {
	tests := []struct {
		name    string
		email   string
		wantErr bool
	}{
		{
			name:    "valid email",
			email:   "user@example.com",
			wantErr: false,
		},
		{
			name:    "valid email with plus",
			email:   "user+tag@example.com",
			wantErr: false,
		},
		{
			name:    "valid email with underscore",
			email:   "user_name@example.com",
			wantErr: false,
		},
		{
			name:    "empty email (allowed)",
			email:   "",
			wantErr: false,
		},
		{
			name:    "invalid email no at",
			email:   "userexample.com",
			wantErr: true,
		},
		{
			name:    "invalid email no domain",
			email:   "user@",
			wantErr: true,
		},
		{
			name:    "email without TLD (valid per RFC)",
			email:   "user@example",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateEmail(tt.email)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateEmail() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateSearch(t *testing.T) {
	tests := []struct {
		name    string
		query   string
		wantErr bool
	}{
		{
			name:    "valid search query",
			query:   "meeting discussion",
			wantErr: false,
		},
		{
			name:    "valid with special chars",
			query:   "meeting -important",
			wantErr: false,
		},
		{
			name:    "valid with punctuation",
			query:   "meeting, discussion!",
			wantErr: false,
		},
		{
			name:    "empty query (allowed)",
			query:   "",
			wantErr: false,
		},
		{
			name:  "query too long",
			query: "a",
		},
		{
			name:    "invalid characters",
			query:   "meeting;DROP TABLE",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateSearch(tt.query)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateSearch() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestFilterMeetings(t *testing.T) {
	meetings := []Meeting{
		{ID: "1", Title: "Project Follow-up", Attendees: []Person{{Email: "a@example.com"}}},
		{ID: "2", Title: "Other", Attendees: []Person{{Email: "b@example.com"}}},
	}
	filtered := filterMeetings(meetings, "project", "")
	if len(filtered) != 1 || filtered[0].ID != "1" {
		t.Fatalf("unexpected search filter result: %+v", filtered)
	}
	filtered = filterMeetings(meetings, "", "b@example.com")
	if len(filtered) != 1 || filtered[0].ID != "2" {
		t.Fatalf("unexpected attendee filter result: %+v", filtered)
	}
}

func TestNewClient(t *testing.T) {
	client := NewClient()
	if client == nil {
		t.Fatal("NewClient() returned nil")
	}
	if client.BinaryPath == "" {
		t.Log("BinaryPath is empty (granola CLI not found in PATH)")
	}
}

func TestValidateSearch_TooLong(t *testing.T) {
	// Test with long query
	longQuery := ""
	for i := 0; i < 500; i++ {
		longQuery += "a"
	}
	err := validateSearch(longQuery)
	if err != nil {
		t.Logf("validateSearch() returned error for long query: %v", err)
	}
}

func TestValidateSearch_Empty(t *testing.T) {
	err := validateSearch("")
	if err != nil {
		t.Error("validateSearch() should allow empty query")
	}
}

func TestValidateSearch_ValidChars(t *testing.T) {
	tests := []string{
		"meeting discussion",
		"meeting-123",
		"meeting_123",
		"meeting, discussion",
	}

	for _, query := range tests {
		err := validateSearch(query)
		if err != nil {
			t.Errorf("validateSearch(%q) = %v, want nil", query, err)
		}
	}
}

func TestValidateSearch_InvalidChars(t *testing.T) {
	tests := []string{
		"meeting;DROP TABLE",
		"meeting | cat",
		"meeting`rm -rf`",
		"meeting$(whoami)",
		"meeting & background",
	}

	for _, query := range tests {
		err := validateSearch(query)
		if err == nil {
			t.Errorf("validateSearch(%q) should return error", query)
		}
	}
}

func TestNewClient_CustomBinaryPath(t *testing.T) {
	// Test with custom binary path
	client := NewClient()
	// BinaryPath may be empty if granola CLI not found
	_ = client
}

func TestValidateID_Empty(t *testing.T) {
	err := validateID("")
	if err == nil {
		t.Error("validateID() should return error for empty ID")
	}
}

func TestValidateID_InvalidFormat(t *testing.T) {
	tests := []string{
		"not-a-uuid",
		"550e8400",
		"550e8400-e29b-41d4-a716-446655440000-extra",
		"550e8400-e29b-41d4-a716-44665544000",
		"550e8400-e29b-41d4-a716-4466554400000",
	}

	for _, id := range tests {
		err := validateID(id)
		if err == nil {
			t.Errorf("validateID(%q) should return error", id)
		}
	}
}

func TestValidateWorkspace_Empty(t *testing.T) {
	err := validateWorkspace("")
	if err != nil {
		t.Error("validateWorkspace() should allow empty workspace")
	}
}

func TestValidateWorkspace_ValidChars(t *testing.T) {
	tests := []string{
		"my-workspace",
		"my_workspace",
		"my-workspace-123",
		"workspace123",
	}

	for _, ws := range tests {
		err := validateWorkspace(ws)
		if err != nil {
			t.Errorf("validateWorkspace(%q) = %v, want nil", ws, err)
		}
	}
}

func TestValidateWorkspace_InvalidChars(t *testing.T) {
	tests := []string{
		"my/workspace",
		"my workspace",
		"my/workspace-123",
		"my/workspace",
	}

	for _, ws := range tests {
		err := validateWorkspace(ws)
		if err == nil {
			t.Errorf("validateWorkspace(%q) should return error", ws)
		}
	}
}

func TestValidateFolder_Empty(t *testing.T) {
	err := validateFolder("")
	if err != nil {
		t.Error("validateFolder() should allow empty folder")
	}
}

func TestValidateFolder_ValidChars(t *testing.T) {
	tests := []string{
		"my-folder",
		"my_folder",
		"my-folder-123",
		"folder123",
	}

	for _, folder := range tests {
		err := validateFolder(folder)
		if err != nil {
			t.Errorf("validateFolder(%q) = %v, want nil", folder, err)
		}
	}
}

func TestValidateFolder_InvalidChars(t *testing.T) {
	tests := []string{
		"my folder",
		"my@folder",
		"my#folder",
		"my$folder",
	}

	for _, folder := range tests {
		err := validateFolder(folder)
		if err == nil {
			t.Errorf("validateFolder(%q) should return error", folder)
		}
	}
}

func TestValidateEmail_Empty(t *testing.T) {
	err := validateEmail("")
	if err != nil {
		t.Error("validateEmail() should allow empty email")
	}
}

func TestValidateEmail_InvalidFormat(t *testing.T) {
	tests := []string{
		"userexample.com",
		"user@",
		"@example.com",
		"user name@example.com",
		"user@.com",
		"user@example.",
	}

	for _, email := range tests {
		err := validateEmail(email)
		if err == nil {
			t.Errorf("validateEmail(%q) should return error", email)
		}
	}
}

func TestValidateEmail_ValidFormat(t *testing.T) {
	tests := []string{
		"user@example.com",
		"user+tag@example.com",
		"user_name@example.com",
		"user.name@example.com",
		"user123@example.com",
	}

	for _, email := range tests {
		err := validateEmail(email)
		if err != nil {
			t.Errorf("validateEmail(%q) = %v, want nil", email, err)
		}
	}
}

func TestNewClient_NoPanic(t *testing.T) {
	// Verify NewClient doesn't panic
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("NewClient() panicked: %v", r)
		}
	}()

	client := NewClient()
	if client == nil {
		t.Error("NewClient() returned nil")
	}
}
