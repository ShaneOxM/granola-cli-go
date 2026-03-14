package secrets

import (
	"testing"
)

func TestToken(t *testing.T) {
	token := Token{
		Email:        "test@example.com",
		RefreshToken: "refresh_token_123",
	}

	if token.Email != "test@example.com" {
		t.Errorf("Email = %v, want test@example.com", token.Email)
	}

	if token.RefreshToken != "refresh_token_123" {
		t.Errorf("RefreshToken = %v, want refresh_token_123", token.RefreshToken)
	}
}

func TestEmptyToken(t *testing.T) {
	token := Token{}

	if token.Email != "" {
		t.Errorf("Email = %v, want empty", token.Email)
	}

	if token.RefreshToken != "" {
		t.Errorf("RefreshToken = %v, want empty", token.RefreshToken)
	}
}

func TestToken_WithValidFields(t *testing.T) {
	// Just verify Token can be created with minimal fields
	token := Token{
		Email:        "test@example.com",
		RefreshToken: "refresh_token_123",
	}

	if token.Email != "test@example.com" {
		t.Errorf("Email = %v, want test@example.com", token.Email)
	}

	if token.RefreshToken != "refresh_token_123" {
		t.Errorf("RefreshToken = %v, want refresh_token_123", token.RefreshToken)
	}
}

func TestStoreInterface(t *testing.T) {
	// Just verify the interface exists and can be referenced
	var _ Store = (*KeyringStore)(nil)
}
