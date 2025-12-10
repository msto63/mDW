// File: random_test.go
// Title: Unit Tests for Random String Generation
// Description: Comprehensive unit tests for secure random string generation
//              functions. Tests validate character sets, length requirements,
//              and security properties of generated strings.
// Author: msto63 with Claude Sonnet 4.0
// Version: v0.1.0
// Created: 2025-01-24
// Modified: 2025-01-24
//
// Change History:
// - 2025-01-24 v0.1.0: Initial test implementation for random generation

package stringx

import (
	"strings"
	"testing"
)

func TestRandomString(t *testing.T) {
	tests := []struct {
		name    string
		length  int
		charset string
		wantErr bool
	}{
		{"normal case", 10, Alphanumeric, false},
		{"zero length", 0, Alphanumeric, false},
		{"negative length", -1, Alphanumeric, false},
		{"empty charset uses default", 5, "", false},
		{"single char charset", 5, "a", false},
		{"custom charset", 8, "xyz123", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := RandomString(tt.length, tt.charset)
			
			if (err != nil) != tt.wantErr {
				t.Errorf("RandomString() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			
			if tt.length <= 0 {
				if result != "" {
					t.Errorf("RandomString() with length %d should return empty string, got %q", tt.length, result)
				}
				return
			}
			
			if len(result) != tt.length {
				t.Errorf("RandomString() length = %d, want %d", len(result), tt.length)
			}
			
			// Verify all characters are from the expected charset
			expectedCharset := tt.charset
			if expectedCharset == "" {
				expectedCharset = Alphanumeric
			}
			
			for _, char := range result {
				if !strings.ContainsRune(expectedCharset, char) {
					t.Errorf("RandomString() contains unexpected character %q, charset: %q", char, expectedCharset)
				}
			}
		})
	}
}

func TestRandomAlphanumeric(t *testing.T) {
	tests := []struct {
		name   string
		length int
	}{
		{"short string", 5},
		{"medium string", 16},
		{"zero length", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := RandomAlphanumeric(tt.length)
			
			if err != nil {
				t.Errorf("RandomAlphanumeric() error = %v", err)
				return
			}
			
			if tt.length == 0 {
				if result != "" {
					t.Errorf("RandomAlphanumeric(0) should return empty string, got %q", result)
				}
				return
			}
			
			if len(result) != tt.length {
				t.Errorf("RandomAlphanumeric() length = %d, want %d", len(result), tt.length)
			}
			
			// Verify all characters are alphanumeric
			for _, char := range result {
				if !strings.ContainsRune(Alphanumeric, char) {
					t.Errorf("RandomAlphanumeric() contains non-alphanumeric character %q", char)
				}
			}
		})
	}
}

func TestRandomHex(t *testing.T) {
	tests := []int{0, 1, 8, 16, 32}
	
	for _, length := range tests {
		t.Run("length "+string(rune(length+'0')), func(t *testing.T) {
			result, err := RandomHex(length)
			
			if err != nil {
				t.Errorf("RandomHex() error = %v", err)
				return
			}
			
			if length == 0 {
				if result != "" {
					t.Errorf("RandomHex(0) should return empty string, got %q", result)
				}
				return
			}
			
			if len(result) != length {
				t.Errorf("RandomHex() length = %d, want %d", len(result), length)
			}
			
			// Verify all characters are hex
			hexChars := "0123456789abcdef"
			for _, char := range result {
				if !strings.ContainsRune(hexChars, char) {
					t.Errorf("RandomHex() contains non-hex character %q", char)
				}
			}
		})
	}
}

func TestRandomURLSafe(t *testing.T) {
	result, err := RandomURLSafe(20)
	
	if err != nil {
		t.Errorf("RandomURLSafe() error = %v", err)
		return
	}
	
	if len(result) != 20 {
		t.Errorf("RandomURLSafe() length = %d, want 20", len(result))
	}
	
	// Verify all characters are URL-safe
	for _, char := range result {
		if !strings.ContainsRune(URLSafe, char) {
			t.Errorf("RandomURLSafe() contains non-URL-safe character %q", char)
		}
	}
}

func TestRandomHumanReadable(t *testing.T) {
	result, err := RandomHumanReadable(15)
	
	if err != nil {
		t.Errorf("RandomHumanReadable() error = %v", err)
		return
	}
	
	if len(result) != 15 {
		t.Errorf("RandomHumanReadable() length = %d, want 15", len(result))
	}
	
	// Verify all characters are human-readable
	for _, char := range result {
		if !strings.ContainsRune(HumanReadable, char) {
			t.Errorf("RandomHumanReadable() contains non-human-readable character %q", char)
		}
	}
	
	// Verify no ambiguous characters are present
	ambiguousChars := "0Ol1"
	for _, char := range result {
		if strings.ContainsRune(ambiguousChars, char) {
			t.Errorf("RandomHumanReadable() contains ambiguous character %q", char)
		}
	}
}

func TestRandomPassword(t *testing.T) {
	tests := []struct {
		name   string
		length int
	}{
		{"short password", 3},
		{"minimum secure", 8},
		{"medium password", 16},
		{"long password", 32},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := RandomPassword(tt.length)
			
			if err != nil {
				t.Errorf("RandomPassword() error = %v", err)
				return
			}
			
			if len(result) != tt.length {
				t.Errorf("RandomPassword() length = %d, want %d", len(result), tt.length)
			}
			
			// Verify all characters are from allowed sets
			allowedChars := Alphanumeric + SpecialChars
			for _, char := range result {
				if !strings.ContainsRune(allowedChars, char) {
					t.Errorf("RandomPassword() contains disallowed character %q", char)
				}
			}
			
			// For passwords >= 4 chars, verify character diversity
			if tt.length >= 4 {
				hasLower := false
				hasUpper := false
				hasDigit := false
				hasSpecial := false
				
				for _, char := range result {
					switch {
					case strings.ContainsRune(LettersLowercase, char):
						hasLower = true
					case strings.ContainsRune(LettersUppercase, char):
						hasUpper = true
					case strings.ContainsRune(Digits, char):
						hasDigit = true
					case strings.ContainsRune(SpecialChars, char):
						hasSpecial = true
					}
				}
				
				if !hasLower {
					t.Errorf("RandomPassword() missing lowercase letters")
				}
				if !hasUpper {
					t.Errorf("RandomPassword() missing uppercase letters")
				}
				if !hasDigit {
					t.Errorf("RandomPassword() missing digits")
				}
				if !hasSpecial {
					t.Errorf("RandomPassword() missing special characters")
				}
			}
		})
	}
}

// Test that random functions actually produce different results
func TestRandomnessUniqueness(t *testing.T) {
	const iterations = 100
	const length = 10
	
	results := make(map[string]bool)
	
	for i := 0; i < iterations; i++ {
		result, err := RandomAlphanumeric(length)
		if err != nil {
			t.Errorf("RandomAlphanumeric() error = %v", err)
			return
		}
		
		if results[result] {
			t.Errorf("RandomAlphanumeric() produced duplicate result: %q", result)
		}
		results[result] = true
	}
	
	// With 62 possible characters and length 10, duplicates should be extremely rare
	if len(results) < iterations/2 {
		t.Errorf("RandomAlphanumeric() produced too many duplicates: %d unique out of %d", len(results), iterations)
	}
}

// Benchmark tests for performance measurement
func BenchmarkRandomString(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, _ = RandomString(16, Alphanumeric)
	}
}

func BenchmarkRandomAlphanumeric(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, _ = RandomAlphanumeric(16)
	}
}

func BenchmarkRandomPassword(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, _ = RandomPassword(16)
	}
}