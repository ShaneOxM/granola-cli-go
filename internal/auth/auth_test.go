package auth

import (
	"testing"
)

func TestCredentials(t *testing.T) {
	// Test Credentials struct creation
	creds := &Credentials{
		EmailAddress: "test@example.com",
		RefreshToken: "refresh_token_123",
		AccessToken:  "access_token_123",
		ClientID:     "client_123",
	}

	if creds.EmailAddress != "test@example.com" {
		t.Errorf("Email = %v, want test@example.com", creds.EmailAddress)
	}

	if creds.RefreshToken != "refresh_token_123" {
		t.Errorf("RefreshToken = %v, want refresh_token_123", creds.RefreshToken)
	}

	if creds.AccessToken != "access_token_123" {
		t.Errorf("AccessToken = %v, want access_token_123", creds.AccessToken)
	}

	if creds.ClientID != "client_123" {
		t.Errorf("ClientID = %v, want client_123", creds.ClientID)
	}
}

func TestCredentials_Empty(t *testing.T) {
	creds := &Credentials{}

	if creds.EmailAddress != "" {
		t.Errorf("Email = %v, want empty", creds.EmailAddress)
	}

	if creds.RefreshToken != "" {
		t.Errorf("RefreshToken = %v, want empty", creds.RefreshToken)
	}

	if creds.AccessToken != "" {
		t.Errorf("AccessToken = %v, want empty", creds.AccessToken)
	}

	if creds.ClientID != "" {
		t.Errorf("ClientID = %v, want empty", creds.ClientID)
	}
}

func TestHasValidToken(t *testing.T) {
	tests := []struct {
		name  string
		creds *Credentials
		want  bool
	}{
		{
			name: "has access token",
			creds: &Credentials{
				AccessToken: "token123",
			},
			want: true,
		},
		{
			name: "no access token",
			creds: &Credentials{
				AccessToken: "",
			},
			want: false,
		},
		{
			name:  "empty credentials",
			creds: &Credentials{},
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hasToken := tt.creds.AccessToken != ""
			if hasToken != tt.want {
				t.Errorf("HasValidToken = %v, want %v", hasToken, tt.want)
			}
		})
	}
}

func TestRefreshTokenRequired(t *testing.T) {
	tests := []struct {
		name  string
		creds *Credentials
		want  bool
	}{
		{
			name: "has refresh token",
			creds: &Credentials{
				RefreshToken: "refresh123",
			},
			want: true,
		},
		{
			name: "no refresh token",
			creds: &Credentials{
				RefreshToken: "",
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hasRefresh := tt.creds.RefreshToken != ""
			if hasRefresh != tt.want {
				t.Errorf("HasRefreshToken = %v, want %v", hasRefresh, tt.want)
			}
		})
	}
}

func TestValidateCredentials(t *testing.T) {
	tests := []struct {
		name    string
		creds   *Credentials
		wantErr bool
	}{
		{
			name: "valid credentials",
			creds: &Credentials{
				EmailAddress: "test@example.com",
				RefreshToken: "refresh123",
				AccessToken:  "access123",
				ClientID:     "client123",
			},
			wantErr: false,
		},
		{
			name: "missing client ID",
			creds: &Credentials{
				EmailAddress: "test@example.com",
				RefreshToken: "refresh123",
				AccessToken:  "access123",
				ClientID:     "",
			},
			wantErr: true,
		},
		{
			name: "missing refresh token",
			creds: &Credentials{
				EmailAddress: "test@example.com",
				RefreshToken: "",
				AccessToken:  "access123",
				ClientID:     "client123",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			needsRefresh := tt.creds.RefreshToken == "" || tt.creds.ClientID == ""
			if needsRefresh != tt.wantErr {
				t.Errorf("NeedsRefresh = %v, wantErr %v", needsRefresh, tt.wantErr)
			}
		})
	}
}
