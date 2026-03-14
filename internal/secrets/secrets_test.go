package secrets

import (
	"encoding/json"
	"fmt"
	"os"
	"testing"
	"time"
)

func requireSystemKeychain(t *testing.T) {
	t.Helper()
	if os.Getenv("GRANOLA_RUN_KEYCHAIN_TESTS") != "1" {
		t.Skip("skipping keychain-dependent test (set GRANOLA_RUN_KEYCHAIN_TESTS=1 to run)")
	}
}

// MockStore implements Store interface for testing
type MockStore struct {
	tokens map[string]*Token
	err    error
}

func NewMockStore(tokens map[string]*Token) *MockStore {
	return &MockStore{
		tokens: tokens,
		err:    nil,
	}
}

func (m *MockStore) SetToken(email string, token *Token) error {
	if m.err != nil {
		return m.err
	}
	if m.tokens == nil {
		m.tokens = make(map[string]*Token)
	}
	m.tokens[email] = token
	return nil
}

func (m *MockStore) GetToken(email string) (*Token, error) {
	if m.err != nil {
		return nil, m.err
	}
	if m.tokens == nil {
		return nil, ErrTokenNotFound
	}
	token, ok := m.tokens[email]
	if !ok {
		return nil, ErrTokenNotFound
	}
	return token, nil
}

func (m *MockStore) DeleteToken(email string) error {
	if m.err != nil {
		return m.err
	}
	if m.tokens == nil {
		return ErrTokenNotFound
	}
	delete(m.tokens, email)
	return nil
}

func (m *MockStore) ListTokens() ([]*Token, error) {
	return []*Token{}, nil
}

func TestNewStore(t *testing.T) {
	requireSystemKeychain(t)
	// Test NewStore - may fail if keychain not available
	store, err := NewStore()
	// Just verify function exists and runs
	if store == nil && err == nil {
		t.Error("NewStore() returned nil store and nil error")
	}
}

func TestToken_Marshal(t *testing.T) {
	token := &Token{
		Email:        "test@example.com",
		RefreshToken: "refresh_123",
		CreatedAt:    time.Now(),
	}

	data, err := json.Marshal(token)
	if err != nil {
		t.Errorf("json.Marshal() failed: %v", err)
	}

	var token2 Token
	if err := json.Unmarshal(data, &token2); err != nil {
		t.Errorf("json.Unmarshal() failed: %v", err)
	}

	if token2.Email != token.Email {
		t.Errorf("Email mismatch: got %v, want %v", token2.Email, token.Email)
	}

	if token2.RefreshToken != token.RefreshToken {
		t.Errorf("RefreshToken mismatch: got %v, want %v", token2.RefreshToken, token.RefreshToken)
	}
}

func TestMockStore_SetToken(t *testing.T) {
	store := NewMockStore(nil)

	token := &Token{
		Email:        "test@example.com",
		RefreshToken: "refresh_123",
		CreatedAt:    time.Now(),
	}

	err := store.SetToken("test@example.com", token)
	if err != nil {
		t.Errorf("SetToken() failed: %v", err)
	}

	retrieved, err := store.GetToken("test@example.com")
	if err != nil {
		t.Errorf("GetToken() failed: %v", err)
	}

	if retrieved.RefreshToken != token.RefreshToken {
		t.Errorf("RefreshToken mismatch: got %v, want %v", retrieved.RefreshToken, token.RefreshToken)
	}
}

func TestMockStore_GetToken_Found(t *testing.T) {
	initialTokens := map[string]*Token{
		"test@example.com": {
			Email:        "test@example.com",
			RefreshToken: "refresh_123",
			CreatedAt:    time.Now(),
		},
	}

	store := NewMockStore(initialTokens)

	token, err := store.GetToken("test@example.com")
	if err != nil {
		t.Errorf("GetToken() failed: %v", err)
	}

	if token.Email != "test@example.com" {
		t.Errorf("Email mismatch: got %v, want test@example.com", token.Email)
	}
}

func TestMockStore_GetToken_NotFound(t *testing.T) {
	store := NewMockStore(nil)

	_, err := store.GetToken("nonexistent@example.com")
	if err == nil {
		t.Error("GetToken() should return error for nonexistent token")
	}
}

func TestMockStore_DeleteToken(t *testing.T) {
	initialTokens := map[string]*Token{
		"test@example.com": {
			Email:        "test@example.com",
			RefreshToken: "refresh_123",
		},
	}

	store := NewMockStore(initialTokens)

	err := store.DeleteToken("test@example.com")
	if err != nil {
		t.Errorf("DeleteToken() failed: %v", err)
	}

	_, err = store.GetToken("test@example.com")
	if err == nil {
		t.Error("GetToken() should fail after deletion")
	}
}

func TestMockStore_DeleteToken_NotFound(t *testing.T) {
	store := NewMockStore(nil)

	err := store.DeleteToken("nonexistent@example.com")
	if err == nil {
		t.Error("DeleteToken() should return error for nonexistent token")
	}
}

func TestMockStore_SetToken_WithError(t *testing.T) {
	store := NewMockStore(nil)
	store.err = ErrOperationFailed

	token := &Token{
		Email:        "test@example.com",
		RefreshToken: "refresh_123",
	}

	err := store.SetToken("test@example.com", token)
	if err == nil {
		t.Error("SetToken() should return error when mock error is set")
	}
}

func TestMockStore_GetToken_WithError(t *testing.T) {
	store := NewMockStore(map[string]*Token{
		"test@example.com": {
			Email:        "test@example.com",
			RefreshToken: "refresh_123",
		},
	})
	store.err = ErrOperationFailed

	_, err := store.GetToken("test@example.com")
	if err == nil {
		t.Error("GetToken() should return error when mock error is set")
	}
}

func TestMockStore_DeleteToken_WithError(t *testing.T) {
	store := NewMockStore(map[string]*Token{
		"test@example.com": {
			Email:        "test@example.com",
			RefreshToken: "refresh_123",
		},
	})
	store.err = ErrOperationFailed

	err := store.DeleteToken("test@example.com")
	if err == nil {
		t.Error("DeleteToken() should return error when mock error is set")
	}
}

func TestMockStore_ListTokens(t *testing.T) {
	store := NewMockStore(map[string]*Token{
		"test@example.com": {
			Email:        "test@example.com",
			RefreshToken: "refresh_123",
		},
	})

	tokens, err := store.ListTokens()
	if err != nil {
		t.Errorf("ListTokens() failed: %v", err)
	}

	// ListTokens always returns empty slice in current implementation
	if len(tokens) != 0 {
		t.Errorf("ListTokens() returned %d tokens, want 0", len(tokens))
	}
}

func TestToken_WithEmptyFields(t *testing.T) {
	token := &Token{
		Email:        "",
		RefreshToken: "",
		CreatedAt:    time.Time{},
	}

	data, err := json.Marshal(token)
	if err != nil {
		t.Errorf("json.Marshal() failed for empty token: %v", err)
	}

	var token2 Token
	if err := json.Unmarshal(data, &token2); err != nil {
		t.Errorf("json.Unmarshal() failed: %v", err)
	}

	if token2.Email != "" {
		t.Errorf("Email should be empty")
	}
}

func TestToken_WithSpecialCharacters(t *testing.T) {
	token := &Token{
		Email:        "test+tag@example.com",
		RefreshToken: "refresh_123!@#$%",
		CreatedAt:    time.Now(),
	}

	data, err := json.Marshal(token)
	if err != nil {
		t.Errorf("json.Marshal() failed for special characters: %v", err)
	}

	var token2 Token
	if err := json.Unmarshal(data, &token2); err != nil {
		t.Errorf("json.Unmarshal() failed: %v", err)
	}

	if token2.Email != token.Email {
		t.Errorf("Email mismatch: got %v, want %v", token2.Email, token.Email)
	}
}

func TestMockStore_Persistence(t *testing.T) {
	store := NewMockStore(nil)

	// Set multiple tokens
	for i := 0; i < 10; i++ {
		email := fmt.Sprintf("user%d@example.com", i)
		token := &Token{
			Email:        email,
			RefreshToken: fmt.Sprintf("refresh_%d", i),
			CreatedAt:    time.Now(),
		}

		if err := store.SetToken(email, token); err != nil {
			t.Errorf("SetToken(%d) failed: %v", i, err)
		}
	}

	// Verify all tokens can be retrieved
	for i := 0; i < 10; i++ {
		email := fmt.Sprintf("user%d@example.com", i)
		token, err := store.GetToken(email)
		if err != nil {
			t.Errorf("GetToken(%s) failed: %v", email, err)
		}

		expectedToken := fmt.Sprintf("refresh_%d", i)
		if token.RefreshToken != expectedToken {
			t.Errorf("Token %d: got %v, want %v", i, token.RefreshToken, expectedToken)
		}
	}
}

func TestMockStore_ConcurrentAccess(t *testing.T) {
	store := NewMockStore(nil)

	// Simulate concurrent access (not truly concurrent but tests data structure)
	for i := 0; i < 100; i++ {
		email := fmt.Sprintf("user%d@example.com", i)
		token := &Token{
			Email:        email,
			RefreshToken: fmt.Sprintf("refresh_%d", i),
			CreatedAt:    time.Now(),
		}

		if err := store.SetToken(email, token); err != nil {
			t.Errorf("SetToken(%d) failed: %v", i, err)
		}

		retrieved, err := store.GetToken(email)
		if err != nil {
			t.Errorf("GetToken(%s) failed: %v", email, err)
		}

		if retrieved.RefreshToken != token.RefreshToken {
			t.Errorf("Token %d mismatch", i)
		}
	}
}
