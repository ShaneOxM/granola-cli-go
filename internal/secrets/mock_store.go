// Package secrets provides mock implementations for testing.
package secrets

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/99designs/keyring"
)

// MockStoreError is a custom error for mock store operations
type MockStoreError struct {
	Message string
	Key     string
}

func (e *MockStoreError) Error() string {
	return fmt.Sprintf("mock store error: %s (key: %s)", e.Message, e.Key)
}

// KeyringMockStore is a thread-safe in-memory keyring store for testing
type KeyringMockStore struct {
	mu    sync.RWMutex
	items map[string]keyring.Item
}

// NewKeyringMockStore creates a new mock store instance
func NewKeyringMockStore() *KeyringMockStore {
	return &KeyringMockStore{
		items: make(map[string]keyring.Item),
	}
}

// Reset clears all items from the mock store
func (m *KeyringMockStore) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.items = make(map[string]keyring.Item)
}

// Get retrieves an item by account key
func (m *KeyringMockStore) Get(account string) (keyring.Item, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	item, ok := m.items[account]
	if !ok {
		return keyring.Item{}, &MockStoreError{
			Message: "item not found",
			Key:     account,
		}
	}
	return item, nil
}

// Set stores an item with the given key
func (m *KeyringMockStore) Set(item keyring.Item) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.items[item.Key] = item
	return nil
}

// Remove deletes an item by key
func (m *KeyringMockStore) Remove(account string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.items[account]; !ok {
		return &MockStoreError{
			Message: "item not found",
			Key:     account,
		}
	}
	delete(m.items, account)
	return nil
}

// List returns all item keys (for testing/debugging)
func (m *KeyringMockStore) List() ([]string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	keys := make([]string, 0, len(m.items))
	for key := range m.items {
		keys = append(keys, key)
	}
	return keys, nil
}

// MockTokenStore is a mock implementation of secrets.Store for auth tests
type MockTokenStore struct {
	mu     sync.RWMutex
	tokens map[string]*Token
	err    error
}

// NewMockTokenStore creates a new mock token store
func NewMockTokenStore() *MockTokenStore {
	return &MockTokenStore{
		tokens: make(map[string]*Token),
	}
}

// SetToken stores a token for the given email
func (m *MockTokenStore) SetToken(email string, token *Token) error {
	if m.err != nil {
		return m.err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.tokens[email] = token
	return nil
}

// GetToken retrieves a token for the given email
func (m *MockTokenStore) GetToken(email string) (*Token, error) {
	if m.err != nil {
		return nil, m.err
	}
	m.mu.RLock()
	defer m.mu.RUnlock()

	token, ok := m.tokens[email]
	if !ok {
		return nil, &MockStoreError{
			Message: "token not found",
			Key:     email,
		}
	}
	return token, nil
}

// DeleteToken removes a token for the given email
func (m *MockTokenStore) DeleteToken(email string) error {
	if m.err != nil {
		return m.err
	}
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.tokens[email]; !ok {
		return &MockStoreError{
			Message: "token not found",
			Key:     email,
		}
	}
	delete(m.tokens, email)
	return nil
}

// ListTokens returns all stored tokens
func (m *MockTokenStore) ListTokens() ([]*Token, error) {
	if m.err != nil {
		return nil, m.err
	}
	m.mu.RLock()
	defer m.mu.RUnlock()

	tokens := make([]*Token, 0, len(m.tokens))
	for _, token := range m.tokens {
		tokens = append(tokens, token)
	}
	return tokens, nil
}

// SetError sets a fake error to test error handling
func (m *MockTokenStore) SetError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.err = err
}

// ClearError clears any set error
func (m *MockTokenStore) ClearError() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.err = nil
}

// MockKeyringItem creates a mock keyring item for testing
func MockKeyringItem(key, label string, data []byte) keyring.Item {
	return keyring.Item{
		Key:   key,
		Label: label,
		Data:  data,
	}
}

// MockToken creates a mock token for testing
func MockToken(email, refreshToken string) *Token {
	return &Token{
		Email:        email,
		RefreshToken: refreshToken,
		CreatedAt:    time.Now(),
	}
}

// TokenToJSON converts a token to JSON bytes for keyring storage
func TokenToJSON(token *Token) ([]byte, error) {
	return json.Marshal(token)
}

// JSONToToken converts JSON bytes to a token
func JSONToToken(data []byte) (*Token, error) {
	var token Token
	if err := json.Unmarshal(data, &token); err != nil {
		return nil, err
	}
	return &token, nil
}
