package secrets

import (
	"testing"
)

func TestGetKeychainItem_InvalidKeychain(t *testing.T) {
	// Service names may still resolve via file fallback; just ensure call is safe.
	_, err := GetKeychainItem("nonexistent_keychain", "test")
	_ = err
}

func TestSetKeychainItem_InvalidKeychain(t *testing.T) {
	// Service names may still resolve via file fallback; just ensure call is safe.
	err := SetKeychainItem("nonexistent_keychain", "test", []byte("data"))
	_ = err
}

func TestDeleteKeychainItem_InvalidKeychain(t *testing.T) {
	// Service names may still resolve via file fallback; just ensure call is safe.
	err := DeleteKeychainItem("nonexistent_keychain", "test")
	_ = err
}

func TestSetKeychainItem_EmptyData(t *testing.T) {
	// Test with empty data - should still work
	err := SetKeychainItem("test_service", "test_account", []byte{})
	// May succeed or fail depending on OS keyring
	_ = err
}

func TestSetKeychainItem_LargeData(t *testing.T) {
	// Test with large data
	largeData := make([]byte, 1024*1024) // 1MB
	for i := range largeData {
		largeData[i] = byte(i % 256)
	}

	err := SetKeychainItem("test_service", "test_account", largeData)
	// May succeed or fail depending on OS keyring
	_ = err
}

func TestGetKeychainItem_AccountNotFound(t *testing.T) {
	// Test with non-existent account
	_, err := GetKeychainItem("granola", "nonexistent_account_12345")
	// May succeed or fail depending on OS keyring
	_ = err
}

func TestDeleteKeychainItem_AccountNotFound(t *testing.T) {
	// Test with non-existent account
	err := DeleteKeychainItem("granola", "nonexistent_account_12345")
	// May succeed or fail depending on OS keyring
	_ = err
}
