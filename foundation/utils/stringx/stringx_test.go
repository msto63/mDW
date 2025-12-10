// File: stringx_test.go
// Title: Unit Tests for Core String Utilities
// Description: Comprehensive unit tests for the core string utility functions
//              in the stringx package. Tests cover edge cases, Unicode handling,
//              and expected behavior for all public functions.
// Author: msto63 with Claude Sonnet 4.0
// Version: v0.1.0
// Created: 2025-01-24
// Modified: 2025-01-24
//
// Change History:
// - 2025-01-24 v0.1.0: Initial test implementation

package stringx

import (
	"testing"
)

func TestIsEmpty(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{"empty string", "", true},
		{"single space", " ", false},
		{"normal string", "hello", false},
		{"unicode string", "こんにちは", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsEmpty(tt.input)
			if result != tt.expected {
				t.Errorf("IsEmpty(%q) = %v; want %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestIsBlank(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{"empty string", "", true},
		{"single space", " ", true},
		{"multiple spaces", "   ", true},
		{"tab and spaces", " \t ", true},
		{"newline", "\n", true},
		{"mixed whitespace", " \t\n\r ", true},
		{"string with content", "hello", false},
		{"string with spaces around", " hello ", false},
		{"unicode content", "こんにちは", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsBlank(tt.input)
			if result != tt.expected {
				t.Errorf("IsBlank(%q) = %v; want %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestIsNotEmpty(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{"empty string", "", false},
		{"single space", " ", true},
		{"normal string", "hello", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsNotEmpty(tt.input)
			if result != tt.expected {
				t.Errorf("IsNotEmpty(%q) = %v; want %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestIsNotBlank(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{"empty string", "", false},
		{"single space", " ", false},
		{"multiple spaces", "   ", false},
		{"string with content", "hello", true},
		{"string with spaces around", " hello ", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsNotBlank(tt.input)
			if result != tt.expected {
				t.Errorf("IsNotBlank(%q) = %v; want %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestTruncate(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		maxLen   int
		ellipsis string
		expected string
	}{
		{"short string no truncation", "hello", 10, "...", "hello"},
		{"exact length no truncation", "hello", 5, "...", "hello"},
		{"basic truncation", "hello world", 8, "...", "hello..."},
		{"unicode truncation", "こんにちは世界", 4, "...", "こ..."},
		{"zero length", "hello", 0, "...", ""},
		{"negative length", "hello", -1, "...", ""},
		{"ellipsis longer than maxLen", "hello", 2, "...", "he"},
		{"empty ellipsis", "hello world", 5, "", "hello"},
		{"custom ellipsis", "hello world", 8, " more", "hel more"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Truncate(tt.input, tt.maxLen, tt.ellipsis)
			if result != tt.expected {
				t.Errorf("Truncate(%q, %d, %q) = %q; want %q", tt.input, tt.maxLen, tt.ellipsis, result, tt.expected)
			}
		})
	}
}

func TestReverse(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"empty string", "", ""},
		{"single character", "a", "a"},
		{"simple string", "hello", "olleh"},
		{"unicode string", "こんにちは", "はちにんこ"},
		{"mixed unicode", "a🌟b", "b🌟a"},
		{"palindrome", "racecar", "racecar"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Reverse(tt.input)
			if result != tt.expected {
				t.Errorf("Reverse(%q) = %q; want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestContainsIgnoreCase(t *testing.T) {
	tests := []struct {
		name     string
		s        string
		substr   string
		expected bool
	}{
		{"exact match", "hello", "hello", true},
		{"case insensitive match", "Hello World", "WORLD", true},
		{"partial match", "Hello World", "wor", true},
		{"no match", "hello", "xyz", false},
		{"empty substr", "hello", "", true},
		{"empty string", "", "hello", false},
		{"both empty", "", "", true},
		{"unicode case insensitive", "ÑOÑO", "ñoño", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ContainsIgnoreCase(tt.s, tt.substr)
			if result != tt.expected {
				t.Errorf("ContainsIgnoreCase(%q, %q) = %v; want %v", tt.s, tt.substr, result, tt.expected)
			}
		})
	}
}

func TestPadLeft(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		width    int
		pad      rune
		expected string
	}{
		{"pad with spaces", "hello", 10, ' ', "     hello"},
		{"pad with zeros", "123", 5, '0', "00123"},
		{"no padding needed", "hello", 3, ' ', "hello"},
		{"exact width", "hello", 5, ' ', "hello"},
		{"unicode input", "こんにちは", 7, '*', "**こんにちは"},
		{"unicode pad", "test", 6, '★', "★★test"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := PadLeft(tt.input, tt.width, tt.pad)
			if result != tt.expected {
				t.Errorf("PadLeft(%q, %d, %q) = %q; want %q", tt.input, tt.width, tt.pad, result, tt.expected)
			}
		})
	}
}

func TestPadRight(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		width    int
		pad      rune
		expected string
	}{
		{"pad with spaces", "hello", 10, ' ', "hello     "},
		{"pad with dashes", "test", 8, '-', "test----"},
		{"no padding needed", "hello", 3, ' ', "hello"},
		{"exact width", "hello", 5, ' ', "hello"},
		{"unicode input", "こんにちは", 7, '*', "こんにちは**"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := PadRight(tt.input, tt.width, tt.pad)
			if result != tt.expected {
				t.Errorf("PadRight(%q, %d, %q) = %q; want %q", tt.input, tt.width, tt.pad, result, tt.expected)
			}
		})
	}
}

func TestCenter(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		width    int
		pad      rune
		expected string
	}{
		{"center odd width", "test", 9, ' ', "  test   "},
		{"center even width", "test", 8, ' ', "  test  "},
		{"center with stars", "hi", 6, '*', "**hi**"},
		{"no padding needed", "hello", 3, ' ', "hello"},
		{"exact width", "hello", 5, ' ', "hello"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Center(tt.input, tt.width, tt.pad)
			if result != tt.expected {
				t.Errorf("Center(%q, %d, %q) = %q; want %q", tt.input, tt.width, tt.pad, result, tt.expected)
			}
		})
	}
}

func TestSplitLines(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{"unix line endings", "line1\nline2\nline3", []string{"line1", "line2", "line3"}},
		{"windows line endings", "line1\r\nline2\r\nline3", []string{"line1", "line2", "line3"}},
		{"mac line endings", "line1\rline2\rline3", []string{"line1", "line2", "line3"}},
		{"mixed line endings", "line1\nline2\r\nline3\rline4", []string{"line1", "line2", "line3", "line4"}},
		{"single line", "single", []string{"single"}},
		{"empty string", "", []string{""}},
		{"only newlines", "\n\n", []string{"", "", ""}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SplitLines(tt.input)
			if len(result) != len(tt.expected) {
				t.Errorf("SplitLines(%q) returned %d lines; want %d", tt.input, len(result), len(tt.expected))
				return
			}
			for i, line := range result {
				if line != tt.expected[i] {
					t.Errorf("SplitLines(%q)[%d] = %q; want %q", tt.input, i, line, tt.expected[i])
				}
			}
		})
	}
}

func TestFirstNonEmpty(t *testing.T) {
	tests := []struct {
		name     string
		inputs   []string
		expected string
	}{
		{"first is non-empty", []string{"hello", "world"}, "hello"},
		{"second is first non-empty", []string{"", "world", "test"}, "world"},
		{"all empty", []string{"", "", ""}, ""},
		{"single non-empty", []string{"hello"}, "hello"},
		{"single empty", []string{""}, ""},
		{"no inputs", []string{}, ""},
		{"space is not empty", []string{" ", "world"}, " "},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FirstNonEmpty(tt.inputs...)
			if result != tt.expected {
				t.Errorf("FirstNonEmpty(%v) = %q; want %q", tt.inputs, result, tt.expected)
			}
		})
	}
}

func TestFirstNonBlank(t *testing.T) {
	tests := []struct {
		name     string
		inputs   []string
		expected string
	}{
		{"first is non-blank", []string{"hello", "world"}, "hello"},
		{"second is first non-blank", []string{"", "world", "test"}, "world"},
		{"skip whitespace", []string{"", " ", "\t", "hello"}, "hello"},
		{"all blank", []string{"", " ", "\t"}, ""},
		{"single non-blank", []string{"hello"}, "hello"},
		{"single blank", []string{" "}, ""},
		{"no inputs", []string{}, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FirstNonBlank(tt.inputs...)
			if result != tt.expected {
				t.Errorf("FirstNonBlank(%v) = %q; want %q", tt.inputs, result, tt.expected)
			}
		})
	}
}

// ===============================
// API Consistency Tests
// ===============================

func TestValidateRequired(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantError bool
	}{
		{"valid string", "hello", false},
		{"empty string", "", true},
		{"whitespace string", "   ", false}, // Not empty, just whitespace
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateRequired(tt.input)
			if (err != nil) != tt.wantError {
				t.Errorf("ValidateRequired(%q) error = %v, wantError %v", tt.input, err, tt.wantError)
			}
		})
	}
}

func TestValidateNotBlank(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantError bool
	}{
		{"valid string", "hello", false},
		{"empty string", "", true},
		{"whitespace only", "   ", true},
		{"mixed content", "  hello  ", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateNotBlank(tt.input)
			if (err != nil) != tt.wantError {
				t.Errorf("ValidateNotBlank(%q) error = %v, wantError %v", tt.input, err, tt.wantError)
			}
		})
	}
}

func TestValidateLength(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		minLen    int
		maxLen    int
		wantError bool
	}{
		{"valid length", "hello", 3, 10, false},
		{"too short", "hi", 3, 10, true},
		{"too long", "this is too long", 3, 10, true},
		{"exact min", "abc", 3, 10, false},
		{"exact max", "1234567890", 3, 10, false},
		{"no min constraint", "ab", 0, 10, false},
		{"no max constraint", "very long string", 3, 0, false},
		{"unicode length", "こんにちは", 3, 10, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateLength(tt.input, tt.minLen, tt.maxLen)
			if (err != nil) != tt.wantError {
				t.Errorf("ValidateLength(%q, %d, %d) error = %v, wantError %v", 
					tt.input, tt.minLen, tt.maxLen, err, tt.wantError)
			}
		})
	}
}

func TestTruncateWithValidation(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		maxLen    int
		ellipsis  string
		want      string
		wantError bool
	}{
		{"valid truncation", "hello world", 8, "...", "hello...", false},
		{"negative maxLen", "hello", -1, "...", "", true},
		{"zero maxLen", "hello", 0, "...", "", false},
		{"no truncation needed", "short", 10, "...", "short", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := TruncateWithValidation(tt.input, tt.maxLen, tt.ellipsis)
			if (err != nil) != tt.wantError {
				t.Errorf("TruncateWithValidation(%q, %d, %q) error = %v, wantError %v", 
					tt.input, tt.maxLen, tt.ellipsis, err, tt.wantError)
				return
			}
			if !tt.wantError && got != tt.want {
				t.Errorf("TruncateWithValidation(%q, %d, %q) = %q, want %q", 
					tt.input, tt.maxLen, tt.ellipsis, got, tt.want)
			}
		})
	}
}

func TestMustTruncate(t *testing.T) {
	t.Run("valid input", func(t *testing.T) {
		result := MustTruncate("hello world", 8, "...")
		expected := "hello..."
		if result != expected {
			t.Errorf("MustTruncate() = %q, want %q", result, expected)
		}
	})

	t.Run("invalid input panics", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Error("MustTruncate should have panicked")
			}
		}()
		MustTruncate("hello", -1, "...")
	})
}

func TestFromDefault(t *testing.T) {
	tests := []struct {
		name         string
		input        string
		defaultValue string
		want         string
	}{
		{"non-empty string", "hello", "default", "hello"},
		{"empty string", "", "default", "default"},
		{"whitespace string", "   ", "default", "   "}, // Whitespace is not empty
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FromDefault(tt.input, tt.defaultValue)
			if got != tt.want {
				t.Errorf("FromDefault(%q, %q) = %q, want %q", tt.input, tt.defaultValue, got, tt.want)
			}
		})
	}
}

func TestFromBlankDefault(t *testing.T) {
	tests := []struct {
		name         string
		input        string
		defaultValue string
		want         string
	}{
		{"non-blank string", "hello", "default", "hello"},
		{"empty string", "", "default", "default"},
		{"whitespace string", "   ", "default", "default"},
		{"mixed content", "  hello  ", "default", "  hello  "},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FromBlankDefault(tt.input, tt.defaultValue)
			if got != tt.want {
				t.Errorf("FromBlankDefault(%q, %q) = %q, want %q", tt.input, tt.defaultValue, got, tt.want)
			}
		})
	}
}