// File: case_test.go
// Title: Unit Tests for Case Conversion Functions
// Description: Comprehensive unit tests for case conversion utilities
//              including snake_case, camelCase, PascalCase, and kebab-case
//              conversions. Tests handle edge cases and Unicode characters.
// Author: msto63 with Claude Sonnet 4.0
// Version: v0.1.0
// Created: 2025-01-24
// Modified: 2025-01-24
//
// Change History:
// - 2025-01-24 v0.1.0: Initial test implementation for case conversions

package stringx

import (
	"testing"
)

func TestToSnakeCase(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"empty string", "", ""},
		{"single word", "hello", "hello"},
		{"camelCase", "helloWorld", "hello_world"},
		{"PascalCase", "HelloWorld", "hello_world"},
		{"with spaces", "hello world", "hello_world"},
		{"with hyphens", "hello-world", "hello_world"},
		{"mixed separators", "hello world-test", "hello_world_test"},
		{"already snake_case", "hello_world", "hello_world"},
		{"consecutive capitals", "HTTPServer", "httpserver"},
		{"single letter", "A", "a"},
		{"numbers", "version2API", "version2_api"},
		{"multiple underscores", "hello__world", "hello_world"},
		{"unicode", "helloWörld", "hello_wörld"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ToSnakeCase(tt.input)
			if result != tt.expected {
				t.Errorf("ToSnakeCase(%q) = %q; want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestToCamelCase(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"empty string", "", ""},
		{"single word", "hello", "hello"},
		{"snake_case", "hello_world", "helloWorld"},
		{"kebab-case", "hello-world", "helloWorld"},
		{"with spaces", "hello world", "helloWorld"},
		{"mixed separators", "hello_world-test case", "helloWorldTestCase"},
		{"already camelCase", "helloWorld", "helloWorld"},
		{"PascalCase", "HelloWorld", "helloWorld"},
		{"single letter", "a", "a"},
		{"single letter words", "a_b_c", "aBC"},
		{"numbers", "version_2_api", "version2Api"},
		{"unicode", "hello_wörld", "helloWörld"},
		{"multiple separators", "hello___world", "helloWorld"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ToCamelCase(tt.input)
			if result != tt.expected {
				t.Errorf("ToCamelCase(%q) = %q; want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestToPascalCase(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"empty string", "", ""},
		{"single word", "hello", "Hello"},
		{"snake_case", "hello_world", "HelloWorld"},
		{"kebab-case", "hello-world", "HelloWorld"},
		{"with spaces", "hello world", "HelloWorld"},
		{"mixed separators", "hello_world-test case", "HelloWorldTestCase"},
		{"already PascalCase", "HelloWorld", "HelloWorld"},
		{"camelCase", "helloWorld", "HelloWorld"},
		{"single letter", "a", "A"},
		{"single letter words", "a_b_c", "ABC"},
		{"numbers", "version_2_api", "Version2Api"},
		{"unicode", "hello_wörld", "HelloWörld"},
		{"multiple separators", "hello___world", "HelloWorld"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ToPascalCase(tt.input)
			if result != tt.expected {
				t.Errorf("ToPascalCase(%q) = %q; want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestToKebabCase(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"empty string", "", ""},
		{"single word", "hello", "hello"},
		{"camelCase", "helloWorld", "hello-world"},
		{"PascalCase", "HelloWorld", "hello-world"},
		{"snake_case", "hello_world", "hello-world"},
		{"with spaces", "hello world", "hello-world"},
		{"mixed separators", "hello world_test", "hello-world-test"},
		{"already kebab-case", "hello-world", "hello-world"},
		{"consecutive capitals", "HTTPServer", "httpserver"},
		{"single letter", "A", "a"},
		{"numbers", "version2API", "version2-api"},
		{"multiple hyphens", "hello--world", "hello-world"},
		{"unicode", "helloWörld", "hello-wörld"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ToKebabCase(tt.input)
			if result != tt.expected {
				t.Errorf("ToKebabCase(%q) = %q; want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestToTitleCase(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"empty string", "", ""},
		{"single word", "hello", "Hello"},
		{"multiple words", "hello world", "Hello World"},
		{"already title case", "Hello World", "Hello World"},
		{"all caps", "HELLO WORLD", "Hello World"},
		{"mixed case", "heLLo WoRLd", "Hello World"},
		{"with punctuation", "hello, world!", "Hello, World!"},
		{"numbers", "version 2 api", "Version 2 Api"},
		{"unicode", "hello wörld", "Hello Wörld"},
		{"single letter", "a", "A"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ToTitleCase(tt.input)
			if result != tt.expected {
				t.Errorf("ToTitleCase(%q) = %q; want %q", tt.input, result, tt.expected)
			}
		})
	}
}

// Test roundtrip conversions to ensure consistency
func TestCaseConversionRoundtrip(t *testing.T) {
	testCases := []string{
		"hello_world",
		"helloWorld", 
		"HelloWorld",
		"hello-world",
		"hello world",
	}

	for _, input := range testCases {
		t.Run("snake->camel->snake: "+input, func(t *testing.T) {
			snake := ToSnakeCase(input)
			camel := ToCamelCase(snake)
			backToSnake := ToSnakeCase(camel)
			
			if snake != backToSnake {
				t.Errorf("Roundtrip failed: %q -> %q -> %q -> %q", input, snake, camel, backToSnake)
			}
		})

		t.Run("kebab->pascal->kebab: "+input, func(t *testing.T) {
			kebab := ToKebabCase(input)
			pascal := ToPascalCase(kebab)
			backToKebab := ToKebabCase(pascal)
			
			if kebab != backToKebab {
				t.Errorf("Roundtrip failed: %q -> %q -> %q -> %q", input, kebab, pascal, backToKebab)
			}
		})
	}
}