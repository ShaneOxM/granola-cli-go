// Package output provides output formatting utilities for CLI responses.
//
// Supported output modes:
//   - ModeAuto: Auto-detect based on environment (default)
//   - ModeJSON: Output as JSON (set GRANOLA_JSON=1)
//   - ModeTSV: Output as tab-separated values (set GRANOLA_TSV=1)
//   - ModeTable: Human-readable table format
//
// Usage:
//
//	// JSON output
//	output.JSON(data)
//
//	// Table output
//	headers := []string{"ID", "Name"}
//	rows := [][]string{{"1", "Item 1"}}
//	output.Table(headers, rows)
package output

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

type Mode int

const (
	ModeAuto Mode = iota
	ModeJSON
	ModeTSV
	ModeTable
)

func FromEnv() Mode {
	if os.Getenv("GRANOLA_JSON") != "" {
		return ModeJSON
	}
	if os.Getenv("GRANOLA_TSV") != "" {
		return ModeTSV
	}
	return ModeAuto
}

func JSON(v interface{}) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

func TSV(headers []string, rows [][]string) error {
	// Print headers
	fmt.Println(strings.Join(headers, "\t"))

	// Print rows
	for _, row := range rows {
		fmt.Println(strings.Join(row, "\t"))
	}

	return nil
}

func Table(headers []string, rows [][]string) error {
	// Calculate column widths
	widths := make([]int, len(headers))
	for i, h := range headers {
		if len(h) > widths[i] {
			widths[i] = len(h)
		}
	}

	for _, row := range rows {
		for i, cell := range row {
			if i < len(widths) && len(cell) > widths[i] {
				widths[i] = len(cell)
			}
		}
	}

	// Print header row
	var headerLine string
	for i, h := range headers {
		if i > 0 {
			headerLine += " | "
		}
		headerLine += padRight(h, widths[i])
	}
	fmt.Println(headerLine)

	// Print separator
	var sepLine string
	for i, w := range widths {
		if i > 0 {
			sepLine += " | "
		}
		sepLine += strings.Repeat("-", w)
	}
	fmt.Println(sepLine)

	// Print data rows
	for _, row := range rows {
		var line string
		for i, cell := range row {
			if i > 0 {
				line += " | "
			}
			if i < len(widths) {
				line += padRight(cell, widths[i])
			} else {
				line += cell
			}
		}
		fmt.Println(line)
	}

	return nil
}

func Truncate(s string, width int) string {
	if width <= 0 || len(s) <= width {
		return s
	}
	if width <= 3 {
		return s[:width]
	}
	return s[:width-3] + "..."
}

func padRight(s string, width int) string {
	if len(s) >= width {
		return s
	}
	return s + strings.Repeat(" ", width-len(s))
}
