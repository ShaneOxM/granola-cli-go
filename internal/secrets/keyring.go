// Package secrets provides secure credential storage using OS keyring services.
//
// This package supports:
//   - macOS Keychain (login keychain)
//   - Windows Credential Locker
//   - Linux libsecret
//
// All credentials are encrypted at rest using OS-provided encryption.
//
// Types:
//   - Token: Represents an OAuth token with email and refresh token
//   - Store: Interface for token operations
//   - KeyringStore: Implementation using keyring library
//
// Functions:
//   - NewStore: Creates a new keyring store
//   - GetKeychainItem: Retrieves item by service and account
//   - SetKeychainItem: Stores data in keyring
//   - DeleteKeychainItem: Removes item from keyring
package secrets

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/99designs/keyring"
)

var (
	// ErrTokenNotFound is returned when a token is not found
	ErrTokenNotFound = errors.New("token not found")
	// ErrOperationFailed is returned when an operation fails
	ErrOperationFailed = errors.New("operation failed")
)

type Token struct {
	Email        string    `json:"email"`
	RefreshToken string    `json:"refresh_token"`
	CreatedAt    time.Time `json:"created_at"`
}

type Store interface {
	SetToken(email string, token *Token) error
	GetToken(email string) (*Token, error)
	DeleteToken(email string) error
	ListTokens() ([]*Token, error)
}

type KeyringStore struct {
	ring keyring.Keyring
}

func keyringConfig(service string, allowFileFallback bool) keyring.Config {
	config := keyring.Config{
		ServiceName:  service,
		KeychainName: "login",
	}

	if allowFileFallback {
		if cfgDir, err := os.UserConfigDir(); err == nil {
			config.FileDir = filepath.Join(cfgDir, "granola-cli", "keyring")
			config.FilePasswordFunc = func(string) (string, error) {
				return ensureFileKeyringPassword(filepath.Join(cfgDir, "granola-cli", "keyring_password"))
			}
		}
	}

	switch runtime.GOOS {
	case "darwin":
		config.AllowedBackends = []keyring.BackendType{keyring.KeychainBackend}
		if allowFileFallback {
			config.AllowedBackends = append(config.AllowedBackends, keyring.FileBackend)
		}
	case "linux":
		config.AllowedBackends = []keyring.BackendType{keyring.SecretServiceBackend}
		if allowFileFallback {
			config.AllowedBackends = append(config.AllowedBackends, keyring.FileBackend)
		}
	case "windows":
		config.AllowedBackends = []keyring.BackendType{keyring.WinCredBackend}
		if allowFileFallback {
			config.AllowedBackends = append(config.AllowedBackends, keyring.FileBackend)
		}
	default:
		config.AllowedBackends = []keyring.BackendType{keyring.FileBackend}
	}

	return config
}

func NewStore() (Store, error) {
	ring, err := keyring.Open(keyringConfig("granola", true))
	if err != nil {
		return nil, fmt.Errorf("open keyring: %w", err)
	}

	return &KeyringStore{ring: ring}, nil
}

func ensureFileKeyringPassword(path string) (string, error) {
	if data, err := os.ReadFile(path); err == nil && len(data) > 0 {
		return string(data), nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return "", err
	}
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	password := base64.StdEncoding.EncodeToString(buf)
	if err := os.WriteFile(path, []byte(password), 0600); err != nil {
		return "", err
	}
	return password, nil
}

func (s *KeyringStore) SetToken(email string, token *Token) error {
	data, err := json.Marshal(token)
	if err != nil {
		return err
	}

	item := keyring.Item{
		Key:   email,
		Label: "Granola CLI",
		Data:  data,
	}

	return s.ring.Set(item)
}

func (s *KeyringStore) GetToken(email string) (*Token, error) {
	item, err := s.ring.Get(email)
	if err != nil {
		return nil, err
	}

	var token Token
	if err := json.Unmarshal(item.Data, &token); err != nil {
		return nil, err
	}

	return &token, nil
}

func (s *KeyringStore) DeleteToken(email string) error {
	return s.ring.Remove(email)
}

func (s *KeyringStore) ListTokens() ([]*Token, error) {
	return []*Token{}, nil
}

// GetKeychainItem retrieves an item from keyring by service and account
// On macOS, keyring uses: Key=account, Label=service
func GetKeychainItem(service, account string) (keyring.Item, error) {
	ring, err := keyring.Open(keyringConfig(service, true))
	if err != nil {
		return keyring.Item{}, fmt.Errorf("open keyring: %w", err)
	}

	return ring.Get(account)
}

// SetKeychainItem stores data in keyring with service and account
// On macOS, keyring uses: Key=account, Label=service
func SetKeychainItem(service, account string, data []byte) error {
	ring, err := keyring.Open(keyringConfig(service, true))
	if err != nil {
		return fmt.Errorf("open keyring: %w", err)
	}

	item := keyring.Item{
		Key:   account,
		Label: service,
		Data:  data,
	}

	return ring.Set(item)
}

// DeleteKeychainItem removes an item from keyring
func DeleteKeychainItem(service, account string) error {
	ring, err := keyring.Open(keyringConfig(service, true))
	if err != nil {
		return fmt.Errorf("open keyring: %w", err)
	}

	return ring.Remove(account)
}
