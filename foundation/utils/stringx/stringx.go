// File: stringx.go
// Title: Core String Utility Functions
// Description: Implements essential string operations that extend the Go
//              standard library. Focuses on Unicode safety, performance,
//              and developer ergonomics for common string manipulation tasks.
// Author: msto63 with Claude Sonnet 4.0
// Version: v0.1.0
// Created: 2025-01-24
// Modified: 2025-01-24
//
// Change History:
// - 2025-01-24 v0.1.0: Initial implementation with core utilities

package stringx

import (
	"fmt"
	"strings"
	"sync"
	"unicode"
	"unicode/utf8"

	"github.com/msto63/mDW/foundation/core/errors"
)

// String interning for commonly used strings to reduce memory allocations
var (
	internCache = make(map[string]string)
	internMu    sync.RWMutex
)

// Intern returns the canonical representation of a string to reduce memory usage
// This is useful for frequently used strings like log levels, error types, etc.
func Intern(s string) string {
	if s == "" {
		return ""
	}
	
	internMu.RLock()
	if interned, exists := internCache[s]; exists {
		internMu.RUnlock()
		return interned
	}
	internMu.RUnlock()
	
	// Make a copy and cache it
	internMu.Lock()
	// Double-check after acquiring write lock
	if interned, exists := internCache[s]; exists {
		internMu.Unlock()
		return interned
	}
	
	// Limit cache size to prevent memory leaks
	if len(internCache) >= 1000 {
		// Clear half the cache (simple eviction strategy)
		for k := range internCache {
			delete(internCache, k)
			if len(internCache) <= 500 {
				break
			}
		}
	}
	
	// Create a copy to ensure we own the memory
	interned := string([]byte(s))
	internCache[s] = interned
	internMu.Unlock()
	
	return interned
}

// IsEmpty returns true if the string is empty (length 0).
// This is a null-safe operation that handles empty strings safely.
func IsEmpty(s string) bool {
	return len(s) == 0
}

// IsBlank returns true if the string is empty or contains only whitespace.
// This is more comprehensive than IsEmpty and commonly needed in validation.
func IsBlank(s string) bool {
	if len(s) == 0 {
		return true
	}
	for _, r := range s {
		if !unicode.IsSpace(r) {
			return false
		}
	}
	return true
}

// IsNotEmpty returns true if the string is not empty.
// Convenience function that's the inverse of IsEmpty.
func IsNotEmpty(s string) bool {
	return len(s) > 0
}

// IsNotBlank returns true if the string is not empty and contains non-whitespace characters.
// Convenience function that's the inverse of IsBlank.
func IsNotBlank(s string) bool {
	return !IsBlank(s)
}

// Truncate truncates a string to the specified length, adding an ellipsis if truncated.
// This function is Unicode-aware and will not break multi-byte characters.
// If the string is shorter than maxLen, it returns the original string.
func Truncate(s string, maxLen int, ellipsis string) string {
	if maxLen <= 0 {
		return ""
	}
	
	// If the string fits, return as-is
	if utf8.RuneCountInString(s) <= maxLen {
		return s
	}
	
	// Calculate space needed for ellipsis
	ellipsisLen := utf8.RuneCountInString(ellipsis)
	if ellipsisLen >= maxLen {
		// If ellipsis is too long, just return truncated string without ellipsis
		return string([]rune(s)[:maxLen])
	}
	
	// Truncate and add ellipsis
	contentLen := maxLen - ellipsisLen
	return string([]rune(s)[:contentLen]) + ellipsis
}

// Reverse reverses a string while preserving Unicode characters.
// This function properly handles multi-byte UTF-8 characters.
func Reverse(s string) string {
	runes := []rune(s)
	for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
		runes[i], runes[j] = runes[j], runes[i]
	}
	return string(runes)
}

// ContainsIgnoreCase returns true if substr is within s, ignoring case.
// This is a case-insensitive version of strings.Contains.
func ContainsIgnoreCase(s, substr string) bool {
	return strings.Contains(strings.ToLower(s), strings.ToLower(substr))
}

// isASCIIString checks if a string contains only ASCII characters
func isASCIIString(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] >= 128 {
			return false
		}
	}
	return true
}

// isASCIIRune checks if a rune is ASCII
func isASCIIRune(r rune) bool {
	return r < 128
}

// PadLeft pads the string s to the specified width with the given pad character.
// If the string is already longer than width, it returns the original string.
func PadLeft(s string, width int, pad rune) string {
	// Fast path for ASCII-only strings and pad characters
	if isASCIIString(s) && isASCIIRune(pad) {
		if len(s) >= width {
			return s
		}
		
		// ASCII optimization: exact allocation
		result := make([]byte, width)
		padCount := width - len(s)
		
		// Fill padding
		for i := 0; i < padCount; i++ {
			result[i] = byte(pad)
		}
		
		// Copy original string
		copy(result[padCount:], s)
		
		return string(result)
	}
	
	// Unicode fallback path
	runeCount := utf8.RuneCountInString(s)
	if runeCount >= width {
		return s
	}
	
	// Optimized version using strings.Builder
	var builder strings.Builder
	padCount := width - runeCount
	builder.Grow(width * 4) // Pre-allocate for worst-case UTF-8
	
	for i := 0; i < padCount; i++ {
		builder.WriteRune(pad)
	}
	builder.WriteString(s)
	
	return builder.String()
}

// PadRight pads the string s to the specified width with the given pad character.
// If the string is already longer than width, it returns the original string.
func PadRight(s string, width int, pad rune) string {
	// Fast path for ASCII-only strings and pad characters
	if isASCIIString(s) && isASCIIRune(pad) {
		if len(s) >= width {
			return s
		}
		
		// ASCII optimization: exact allocation
		result := make([]byte, width)
		
		// Copy original string
		copy(result, s)
		
		// Fill padding
		for i := len(s); i < width; i++ {
			result[i] = byte(pad)
		}
		
		return string(result)
	}
	
	// Unicode fallback path
	runeCount := utf8.RuneCountInString(s)
	if runeCount >= width {
		return s
	}
	
	// Optimized version using strings.Builder
	var builder strings.Builder
	padCount := width - runeCount
	builder.Grow(width * 4) // Pre-allocate for worst-case UTF-8
	
	builder.WriteString(s)
	for i := 0; i < padCount; i++ {
		builder.WriteRune(pad)
	}
	
	return builder.String()
}

// Center centers the string s within the specified width using the pad character.
// If the string is already longer than width, it returns the original string.
func Center(s string, width int, pad rune) string {
	// Fast path for ASCII-only strings and pad characters
	if isASCIIString(s) && isASCIIRune(pad) {
		if len(s) >= width {
			return s
		}
		
		// ASCII optimization: exact allocation
		result := make([]byte, width)
		totalPadding := width - len(s)
		leftPadding := totalPadding / 2
		
		// Left padding
		for i := 0; i < leftPadding; i++ {
			result[i] = byte(pad)
		}
		
		// Copy original string
		copy(result[leftPadding:], s)
		
		// Right padding
		for i := leftPadding + len(s); i < width; i++ {
			result[i] = byte(pad)
		}
		
		return string(result)
	}
	
	// Unicode fallback path
	runeCount := utf8.RuneCountInString(s)
	if runeCount >= width {
		return s
	}
	
	// Optimized version using strings.Builder
	var builder strings.Builder
	totalPadding := width - runeCount
	leftPadding := totalPadding / 2
	rightPadding := totalPadding - leftPadding
	builder.Grow(width * 4) // Pre-allocate for worst-case UTF-8
	
	// Left padding
	for i := 0; i < leftPadding; i++ {
		builder.WriteRune(pad)
	}
	
	// Original string
	builder.WriteString(s)
	
	// Right padding
	for i := 0; i < rightPadding; i++ {
		builder.WriteRune(pad)
	}
	
	return builder.String()
}

// SplitLines splits a string into lines, handling different line ending conventions.
// It properly handles \n, \r\n, and \r line endings.
func SplitLines(s string) []string {
	// Normalize line endings to \n
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.ReplaceAll(s, "\r", "\n")
	
	return strings.Split(s, "\n")
}

// FirstNonEmpty returns the first non-empty string from the provided strings.
// This is useful for providing default values in a chain.
func FirstNonEmpty(strings ...string) string {
	for _, s := range strings {
		if IsNotEmpty(s) {
			return s
		}
	}
	return ""
}

// FirstNonBlank returns the first non-blank string from the provided strings.
// This is useful for providing default values while ignoring whitespace-only strings.
func FirstNonBlank(strings ...string) string {
	for _, s := range strings {
		if IsNotBlank(s) {
			return s
		}
	}
	return ""
}

// ===============================
// API Consistency Improvements
// ===============================

// ValidateRequired validates that a string is not empty, following standard error patterns
func ValidateRequired(s string) error {
	if IsEmpty(s) {
		return errors.StringxValidationError("validate_required", s, "non-empty string")
	}
	return nil
}

// ValidateNotBlank validates that a string is not blank, following standard error patterns
func ValidateNotBlank(s string) error {
	if IsBlank(s) {
		return errors.StringxValidationError("validate_not_blank", s, "non-blank string")
	}
	return nil
}

// ValidateLength validates that a string meets length requirements
func ValidateLength(s string, minLen, maxLen int) error {
	length := utf8.RuneCountInString(s)
	
	if minLen > 0 && length < minLen {
		return errors.StringxValidationError("validate_length", 
			fmt.Sprintf("%s (length: %d)", s, length), 
			fmt.Sprintf("at least %d characters", minLen))
	}
	
	if maxLen > 0 && length > maxLen {
		return errors.StringxValidationError("validate_length", 
			fmt.Sprintf("%s (length: %d)", s, length), 
			fmt.Sprintf("at most %d characters", maxLen))
	}
	
	return nil
}

// TruncateWithValidation truncates a string with input validation, following standard error patterns
func TruncateWithValidation(s string, maxLen int, ellipsis string) (string, error) {
	if maxLen < 0 {
		return "", errors.StringxInvalidInput("truncate_with_validation", maxLen)
	}
	
	if maxLen == 0 {
		return "", nil
	}
	
	// Input validation passed, use the existing Truncate function
	return Truncate(s, maxLen, ellipsis), nil
}

// MustTruncate truncates a string, panicking on invalid input (follows Must* pattern)
func MustTruncate(s string, maxLen int, ellipsis string) string {
	result, err := TruncateWithValidation(s, maxLen, ellipsis)
	if err != nil {
		panic(err)
	}
	return result
}

// ParseLength parses a string and validates its length, following standard parsing patterns
func ParseLength(s string, expectedMin, expectedMax int) (string, error) {
	if err := ValidateLength(s, expectedMin, expectedMax); err != nil {
		return "", fmt.Errorf("stringx.parse_length failed: %w", err)
	}
	return s, nil
}

// FromDefault returns the string if not empty, otherwise returns the default value (follows From* pattern)
func FromDefault(s, defaultValue string) string {
	if IsEmpty(s) {
		return defaultValue
	}
	return s
}

// FromBlankDefault returns the string if not blank, otherwise returns the default value
func FromBlankDefault(s, defaultValue string) string {
	if IsBlank(s) {
		return defaultValue
	}
	return s
}