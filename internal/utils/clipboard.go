package utils

import (
	"fmt"
	"github.com/atotto/clipboard"
)

// CopyToClipboard copies text to the system clipboard
func CopyToClipboard(text string) error {
	if err := clipboard.WriteAll(text); err != nil {
		return fmt.Errorf("failed to copy to clipboard: %w", err)
	}
	return nil
}
