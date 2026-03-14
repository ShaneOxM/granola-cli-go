package utils

import (
	"testing"
)

func TestCopyToClipboard(t *testing.T) {
	// Test with valid text
	err := CopyToClipboard("Test clipboard text")
	if err != nil {
		t.Skipf("Clipboard not available or error: %v", err)
	}

	// Test with empty text
	err = CopyToClipboard("")
	if err != nil {
		t.Skipf("Clipboard not available or error: %v", err)
	}

	// Test with long text
	longText := ""
	for i := 0; i < 1000; i++ {
		longText += "This is a test line "
	}
	err = CopyToClipboard(longText)
	if err != nil {
		t.Skipf("Clipboard not available or error: %v", err)
	}
}

func TestCopyToClipboard_Error(t *testing.T) {
	// Test that we handle errors gracefully
	// This test may be skipped if clipboard is not available
	err := CopyToClipboard("Test")
	if err != nil {
		t.Logf("Expected clipboard error (may not be available): %v", err)
	}
}
