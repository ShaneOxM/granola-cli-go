package output

import (
	"os"
	"testing"
)

func TestFromEnv_JSON(t *testing.T) {
	original := os.Getenv("GRANOLA_JSON")
	defer os.Setenv("GRANOLA_JSON", original)

	os.Setenv("GRANOLA_JSON", "1")
	mode := FromEnv()

	if mode != ModeJSON {
		t.Errorf("FromEnv() = %v, want ModeJSON", mode)
	}
}

func TestFromEnv_TSV(t *testing.T) {
	originalJSON := os.Getenv("GRANOLA_JSON")
	originalTSV := os.Getenv("GRANOLA_TSV")
	defer func() {
		os.Setenv("GRANOLA_JSON", originalJSON)
		os.Setenv("GRANOLA_TSV", originalTSV)
	}()

	os.Unsetenv("GRANOLA_JSON")
	os.Setenv("GRANOLA_TSV", "1")
	mode := FromEnv()

	if mode != ModeTSV {
		t.Errorf("FromEnv() = %v, want ModeTSV", mode)
	}
}

func TestFromEnv_Auto(t *testing.T) {
	originalJSON := os.Getenv("GRANOLA_JSON")
	originalTSV := os.Getenv("GRANOLA_TSV")
	defer func() {
		os.Setenv("GRANOLA_JSON", originalJSON)
		os.Setenv("GRANOLA_TSV", originalTSV)
	}()

	os.Unsetenv("GRANOLA_JSON")
	os.Unsetenv("GRANOLA_TSV")
	mode := FromEnv()

	if mode != ModeAuto {
		t.Errorf("FromEnv() = %v, want ModeAuto", mode)
	}
}

func TestJSON(t *testing.T) {
	// Test JSON encoding with simple struct
	type TestStruct struct {
		Name  string `json:"name"`
		Value int    `json:"value"`
	}

	data := TestStruct{Name: "test", Value: 42}
	err := JSON(data)
	// Just verify function exists and runs
	_ = err
}

func TestJSON_EmptyStruct(t *testing.T) {
	type EmptyStruct struct{}
	data := EmptyStruct{}
	err := JSON(data)
	_ = err
}

func TestJSON_Slice(t *testing.T) {
	data := []string{"item1", "item2", "item3"}
	err := JSON(data)
	_ = err
}

func TestJSON_Map(t *testing.T) {
	data := map[string]int{"a": 1, "b": 2}
	err := JSON(data)
	_ = err
}

func TestTSV_Empty(t *testing.T) {
	err := TSV([]string{}, [][]string{})
	_ = err
}

func TestTSV_SingleRow(t *testing.T) {
	headers := []string{"ID", "Name"}
	rows := [][]string{{"1", "Item 1"}}

	err := TSV(headers, rows)
	_ = err
}

func TestTSV_MultipleRows(t *testing.T) {
	headers := []string{"ID", "Name", "Value"}
	rows := [][]string{
		{"1", "Item 1", "100"},
		{"2", "Item 2", "200"},
		{"3", "Item 3", "300"},
	}

	err := TSV(headers, rows)
	_ = err
}

func TestTSV_EmptyHeaders(t *testing.T) {
	headers := []string{}
	rows := [][]string{{"1", "2", "3"}}

	err := TSV(headers, rows)
	_ = err
}

func TestTSV_EmptyRows(t *testing.T) {
	headers := []string{"ID", "Name"}
	rows := [][]string{}

	err := TSV(headers, rows)
	_ = err
}

func TestTable_Empty(t *testing.T) {
	err := Table([]string{}, [][]string{})
	_ = err
}

func TestTable_SingleRow(t *testing.T) {
	headers := []string{"ID", "Name"}
	rows := [][]string{{"1", "Item 1"}}

	err := Table(headers, rows)
	_ = err
}

func TestTable_MultipleRows(t *testing.T) {
	headers := []string{"ID", "Name", "Value"}
	rows := [][]string{
		{"1", "Item 1", "100"},
		{"2", "Item 2", "200"},
		{"3", "Item 3", "300"},
	}

	err := Table(headers, rows)
	_ = err
}

func TestTable_EmptyHeaders(t *testing.T) {
	headers := []string{}
	rows := [][]string{{"1", "2", "3"}}

	err := Table(headers, rows)
	_ = err
}

func TestTable_EmptyRows(t *testing.T) {
	headers := []string{"ID", "Name"}
	rows := [][]string{}

	err := Table(headers, rows)
	_ = err
}

func TestTable_VaryingWidths(t *testing.T) {
	headers := []string{"Short", "MediumLength", "VeryLongText"}
	rows := [][]string{
		{"A", "BB", "CCC"},
		{"DDDDDDDD", "EEEE", "FFFFFF"},
		{"G", "HHHHHHHHHH", "IIIIIIIIII"},
	}

	err := Table(headers, rows)
	_ = err
}

func TestPadRight_Shorter(t *testing.T) {
	result := padRight("hello", 10)
	if len(result) != 10 {
		t.Errorf("padRight() length = %v, want 10", len(result))
	}
}

func TestPadRight_Equal(t *testing.T) {
	result := padRight("hello", 5)
	if result != "hello" {
		t.Errorf("padRight() = %v, want hello", result)
	}
}

func TestPadRight_Longer(t *testing.T) {
	result := padRight("hello", 3)
	if result != "hello" {
		t.Errorf("padRight() = %v, want hello", result)
	}
}

func TestPadRight_Empty(t *testing.T) {
	result := padRight("", 5)
	if len(result) != 5 {
		t.Errorf("padRight() length = %v, want 5", len(result))
	}
}

func TestPadRight_Unicode(t *testing.T) {
	result := padRight("你好", 10)
	// Just verify it runs without error
	_ = result
}

func TestTruncate(t *testing.T) {
	if got := Truncate("hello world", 5); got != "he..." {
		t.Errorf("Truncate() = %v, want he...", got)
	}
	if got := Truncate("hi", 5); got != "hi" {
		t.Errorf("Truncate() = %v, want hi", got)
	}
}

func TestMode_Constants(t *testing.T) {
	if ModeAuto != 0 {
		t.Errorf("ModeAuto = %v, want 0", ModeAuto)
	}

	if ModeJSON != 1 {
		t.Errorf("ModeJSON = %v, want 1", ModeJSON)
	}

	if ModeTSV != 2 {
		t.Errorf("ModeTSV = %v, want 2", ModeTSV)
	}

	if ModeTable != 3 {
		t.Errorf("ModeTable = %v, want 3", ModeTable)
	}
}

func TestFromEnv_Priority_JSON(t *testing.T) {
	originalJSON := os.Getenv("GRANOLA_JSON")
	originalTSV := os.Getenv("GRANOLA_TSV")
	defer func() {
		os.Setenv("GRANOLA_JSON", originalJSON)
		os.Setenv("GRANOLA_TSV", originalTSV)
	}()

	// JSON should take priority
	os.Setenv("GRANOLA_JSON", "1")
	os.Setenv("GRANOLA_TSV", "1")
	mode := FromEnv()

	if mode != ModeJSON {
		t.Errorf("FromEnv() = %v, want ModeJSON (JSON has priority)", mode)
	}
}

func TestFromEnv_BothSet(t *testing.T) {
	originalJSON := os.Getenv("GRANOLA_JSON")
	originalTSV := os.Getenv("GRANOLA_TSV")
	defer func() {
		os.Setenv("GRANOLA_JSON", originalJSON)
		os.Setenv("GRANOLA_TSV", originalTSV)
	}()

	os.Setenv("GRANOLA_JSON", "")
	os.Setenv("GRANOLA_TSV", "")
	mode := FromEnv()

	if mode != ModeAuto {
		t.Errorf("FromEnv() = %v, want ModeAuto", mode)
	}
}

func TestTSV_Unicode(t *testing.T) {
	headers := []string{"ID", "名称"}
	rows := [][]string{{"1", "项目 1"}}

	err := TSV(headers, rows)
	_ = err
}

func TestTable_Unicode(t *testing.T) {
	headers := []string{"ID", "名称"}
	rows := [][]string{{"1", "项目 1"}}

	err := Table(headers, rows)
	_ = err
}

func TestTable_SpecialCharacters(t *testing.T) {
	headers := []string{"ID", "Name"}
	rows := [][]string{
		{"1", "Item with | pipe"},
		{"2", "Item with -- dashes"},
	}

	err := Table(headers, rows)
	_ = err
}

func TestTable_EmptyCells(t *testing.T) {
	headers := []string{"ID", "Name"}
	rows := [][]string{
		{"1", ""},
		{"", "Item 2"},
	}

	err := Table(headers, rows)
	_ = err
}

func TestPadRight_EmptyString(t *testing.T) {
	result := padRight("", 0)
	if result != "" {
		t.Errorf("padRight() = %v, want empty", result)
	}
}

func TestPadRight_ZeroWidth(t *testing.T) {
	result := padRight("hello", 0)
	if result != "hello" {
		t.Errorf("padRight() = %v, want hello", result)
	}
}
