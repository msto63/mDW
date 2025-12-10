// File: case.go
// Title: String Case Conversion Utilities
// Description: Implements case conversion functions for various naming
//              conventions commonly used in Go development. Supports
//              snake_case, camelCase, PascalCase, and kebab-case conversions.
// Author: msto63 with Claude Sonnet 4.0
// Version: v0.1.0
// Created: 2025-01-24
// Modified: 2025-01-24
//
// Change History:
// - 2025-01-24 v0.1.0: Initial implementation with case conversion utilities

package stringx

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

// ToSnakeCase converts a string to snake_case.
// It handles camelCase, PascalCase, and spaces by converting them to underscores.
// Example: "MyVariableName" -> "my_variable_name"
func ToSnakeCase(s string) string {
	if IsEmpty(s) {
		return s
	}
	
	var result strings.Builder
	for i, r := range s {
		if unicode.IsUpper(r) {
			// Add underscore before uppercase letters (except at the beginning)
			if i > 0 && !unicode.IsUpper(rune(s[i-1])) {
				result.WriteRune('_')
			}
			result.WriteRune(unicode.ToLower(r))
		} else if unicode.IsSpace(r) || r == '-' {
			// Convert spaces and hyphens to underscores
			result.WriteRune('_')
		} else {
			result.WriteRune(r)
		}
	}
	
	// Clean up multiple consecutive underscores
	return strings.ReplaceAll(result.String(), "__", "_")
}

// ToCamelCase converts a string to camelCase.
// It handles snake_case, kebab-case, and spaces by converting them appropriately.
// Example: "my_variable_name" -> "myVariableName"
func ToCamelCase(s string) string {
	if IsEmpty(s) {
		return s
	}
	
	// If the string doesn't contain separators, check if it's already camelCase
	if !strings.ContainsAny(s, "_- ") {
		// If it starts with lowercase, assume it's already camelCase
		firstRune, _ := utf8.DecodeRuneInString(s)
		if unicode.IsLower(firstRune) {
			return s
		}
		// If it starts with uppercase, convert first character to lowercase
		if unicode.IsUpper(firstRune) {
			return strings.ToLower(string(firstRune)) + s[len(string(firstRune)):]
		}
		return s
	}
	
	// Split on common separators
	words := strings.FieldsFunc(s, func(r rune) bool {
		return r == '_' || r == '-' || unicode.IsSpace(r)
	})
	
	if len(words) == 0 {
		return s
	}
	
	var result strings.Builder
	
	// First word stays lowercase
	result.WriteString(strings.ToLower(words[0]))
	
	// Subsequent words are capitalized
	for i := 1; i < len(words); i++ {
		if len(words[i]) > 0 {
			result.WriteString(strings.Title(strings.ToLower(words[i])))
		}
	}
	
	return result.String()
}

// ToPascalCase converts a string to PascalCase.
// It handles snake_case, kebab-case, and spaces by converting them appropriately.
// Example: "my_variable_name" -> "MyVariableName"
func ToPascalCase(s string) string {
	if IsEmpty(s) {
		return s
	}
	
	// If the string doesn't contain separators, check if it's already PascalCase
	if !strings.ContainsAny(s, "_- ") {
		// If it starts with uppercase, assume it's already PascalCase
		firstRune, _ := utf8.DecodeRuneInString(s)
		if unicode.IsUpper(firstRune) {
			return s
		}
		// If it starts with lowercase, convert first character to uppercase
		if unicode.IsLower(firstRune) {
			return strings.ToUpper(string(firstRune)) + s[len(string(firstRune)):]
		}
		return s
	}
	
	// Split on common separators
	words := strings.FieldsFunc(s, func(r rune) bool {
		return r == '_' || r == '-' || unicode.IsSpace(r)
	})
	
	if len(words) == 0 {
		return s
	}
	
	var result strings.Builder
	
	// All words are capitalized
	for _, word := range words {
		if len(word) > 0 {
			result.WriteString(strings.Title(strings.ToLower(word)))
		}
	}
	
	return result.String()
}

// ToKebabCase converts a string to kebab-case.
// It handles camelCase, PascalCase, snake_case, and spaces appropriately.
// Example: "MyVariableName" -> "my-variable-name"
func ToKebabCase(s string) string {
	if IsEmpty(s) {
		return s
	}
	
	var result strings.Builder
	for i, r := range s {
		if unicode.IsUpper(r) {
			// Add hyphen before uppercase letters (except at the beginning)
			if i > 0 && !unicode.IsUpper(rune(s[i-1])) {
				result.WriteRune('-')
			}
			result.WriteRune(unicode.ToLower(r))
		} else if unicode.IsSpace(r) || r == '_' {
			// Convert spaces and underscores to hyphens
			result.WriteRune('-')
		} else {
			result.WriteRune(r)
		}
	}
	
	// Clean up multiple consecutive hyphens
	return strings.ReplaceAll(result.String(), "--", "-")
}

// ToTitleCase converts a string to Title Case.
// It capitalizes the first letter of each word while preserving spaces.
// Example: "hello world" -> "Hello World"
func ToTitleCase(s string) string {
	if IsEmpty(s) {
		return s
	}
	
	return strings.Title(strings.ToLower(s))
}