// File: validationx_test.go
// Title: Validation Utilities Tests
// Description: Comprehensive test suite for all validationx utility functions including
//              unit tests, edge cases, and integration scenarios.
// Author: msto63 with Claude Sonnet 4.0
// Version: v0.1.0
// Created: 2025-01-25
// Modified: 2025-01-25
//
// Change History:
// - 2025-01-25 v0.1.0: Initial test implementation with comprehensive coverage

package validationx

import (
	"strings"
	"testing"
	"time"

	"github.com/msto63/mDW/foundation/core/validation"
)

// ===============================
// Basic Validation Tests
// ===============================

func TestRequired(t *testing.T) {
	testCases := []struct {
		name    string
		value   interface{}
		isValid bool
	}{
		{"nil value", nil, false},
		{"empty string", "", false},
		{"whitespace string", "   ", false},
		{"valid string", "hello", true},
		{"empty slice", []interface{}{}, false},
		{"valid slice", []interface{}{1, 2, 3}, true},
		{"empty map", map[string]interface{}{}, false},
		{"valid map", map[string]interface{}{"key": "value"}, true},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := Required.Validate(tc.value)
			if result.Valid != tc.isValid {
				t.Errorf("Required(%v) = %v, want %v", tc.value, result.Valid, tc.isValid)
			}
		})
	}
}

func TestOptional(t *testing.T) {
	validator := Optional(MinLength(5))
	
	testCases := []struct {
		name    string
		value   interface{}
		isValid bool
	}{
		{"nil value", nil, true},
		{"empty string", "", true},
		{"short string", "abc", false},
		{"valid string", "hello world", true},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := validator.Validate(tc.value)
			if result.Valid != tc.isValid {
				t.Errorf("Optional validator(%v) = %v, want %v", tc.value, result.Valid, tc.isValid)
			}
		})
	}
}

// ===============================
// String Validation Tests
// ===============================

func TestMinLength(t *testing.T) {
	validator := MinLength(5)
	
	testCases := []struct {
		name    string
		value   interface{}
		isValid bool
	}{
		{"too short", "abc", false},
		{"exact length", "hello", true},
		{"longer than required", "hello world", true},
		{"non-string", 123, false},
		{"unicode string", "héllo", true}, // 5 characters
		{"empty string", "", false},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := validator.Validate(tc.value)
			if result.Valid != tc.isValid {
				t.Errorf("MinLength(5)(%v) = %v, want %v", tc.value, result.Valid, tc.isValid)
			}
		})
	}
}

func TestMaxLength(t *testing.T) {
	validator := MaxLength(5)
	
	testCases := []struct {
		name    string
		value   interface{}
		isValid bool
	}{
		{"shorter than max", "abc", true},
		{"exact length", "hello", true},
		{"longer than max", "hello world", false},
		{"non-string", 123, false},
		{"unicode string", "hé", true}, // 2 characters
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := validator.Validate(tc.value)
			if result.Valid != tc.isValid {
				t.Errorf("MaxLength(5)(%v) = %v, want %v", tc.value, result.Valid, tc.isValid)
			}
		})
	}
}

func TestLength(t *testing.T) {
	validator := Length(5)
	
	testCases := []struct {
		name    string
		value   interface{}
		isValid bool
	}{
		{"too short", "abc", false},
		{"exact length", "hello", true},
		{"too long", "hello world", false},
		{"non-string", 123, false},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := validator.Validate(tc.value)
			if result.Valid != tc.isValid {
				t.Errorf("Length(5)(%v) = %v, want %v", tc.value, result.Valid, tc.isValid)
			}
		})
	}
}

func TestContains(t *testing.T) {
	validator := Contains("world")
	
	testCases := []struct {
		name    string
		value   interface{}
		isValid bool
	}{
		{"contains substring", "hello world", true},
		{"does not contain", "hello there", false},
		{"non-string", 123, false},
		{"empty string", "", false},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := validator.Validate(tc.value)
			if result.Valid != tc.isValid {
				t.Errorf("Contains('world')(%v) = %v, want %v", tc.value, result.Valid, tc.isValid)
			}
		})
	}
}

func TestAlphaOnly(t *testing.T) {
	testCases := []struct {
		name    string
		value   interface{}
		isValid bool
	}{
		{"only letters", "hello", true},
		{"letters with numbers", "hello123", false},
		{"letters with spaces", "hello world", false},
		{"letters with symbols", "hello!", false},
		{"non-string", 123, false},
		{"empty string", "", true}, // Empty string is valid (no non-alpha chars)
		{"unicode letters", "héllo", true},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := AlphaOnly.Validate(tc.value)
			if result.Valid != tc.isValid {
				t.Errorf("AlphaOnly(%v) = %v, want %v", tc.value, result.Valid, tc.isValid)
			}
		})
	}
}

func TestAlphaNumeric(t *testing.T) {
	testCases := []struct {
		name    string
		value   interface{}
		isValid bool
	}{
		{"letters and numbers", "hello123", true},
		{"only letters", "hello", true},
		{"only numbers", "123", true},
		{"with spaces", "hello 123", false},
		{"with symbols", "hello!", false},
		{"non-string", 123, false},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := AlphaNumeric.Validate(tc.value)
			if result.Valid != tc.isValid {
				t.Errorf("AlphaNumeric(%v) = %v, want %v", tc.value, result.Valid, tc.isValid)
			}
		})
	}
}

func TestNumericOnly(t *testing.T) {
	testCases := []struct {
		name    string
		value   interface{}
		isValid bool
	}{
		{"only numbers", "123", true},
		{"numbers with letters", "123abc", false},
		{"numbers with spaces", "1 2 3", false},
		{"numbers with symbols", "123!", false},
		{"non-string", 123, false},
		{"empty string", "", true}, // Empty string is valid (no non-numeric chars)
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := NumericOnly.Validate(tc.value)
			if result.Valid != tc.isValid {
				t.Errorf("NumericOnly(%v) = %v, want %v", tc.value, result.Valid, tc.isValid)
			}
		})
	}
}

// ===============================
// Pattern Validation Tests
// ===============================

func TestPattern(t *testing.T) {
	validator := Pattern(`^\d{3}-\d{2}-\d{4}$`) // SSN pattern
	
	testCases := []struct {
		name    string
		value   interface{}
		isValid bool
	}{
		{"valid SSN", "123-45-6789", true},
		{"invalid SSN", "123-456-789", false},
		{"non-matching", "hello", false},
		{"non-string", 123, false},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := validator.Validate(tc.value)
			if result.Valid != tc.isValid {
				t.Errorf("Pattern SSN(%v) = %v, want %v", tc.value, result.Valid, tc.isValid)
			}
		})
	}
}

func TestEmail(t *testing.T) {
	testCases := []struct {
		name    string
		value   interface{}
		isValid bool
	}{
		{"valid email", "user@example.com", true},
		{"valid email with subdomain", "user@mail.example.com", true},
		{"invalid email - no @", "userexample.com", false},
		{"invalid email - no domain", "user@", false},
		{"invalid email - no user", "@example.com", false},
		{"non-string", 123, false},
		{"empty string", "", false},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := Email.Validate(tc.value)
			if result.Valid != tc.isValid {
				t.Errorf("Email(%v) = %v, want %v", tc.value, result.Valid, tc.isValid)
			}
		})
	}
}

func TestURL(t *testing.T) {
	testCases := []struct {
		name    string
		value   interface{}
		isValid bool
	}{
		{"valid HTTP URL", "http://example.com", true},
		{"valid HTTPS URL", "https://example.com", true},
		{"valid URL with path", "https://example.com/path", true},
		{"valid URL with query", "https://example.com?query=value", true},
		{"invalid URL - no scheme", "example.com", false},
		{"invalid URL - malformed", "http://", false},
		{"non-string", 123, false},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := URL.Validate(tc.value)
			if result.Valid != tc.isValid {
				t.Errorf("URL(%v) = %v, want %v", tc.value, result.Valid, tc.isValid)
			}
		})
	}
}

func TestIP(t *testing.T) {
	testCases := []struct {
		name    string
		value   interface{}
		isValid bool
	}{
		{"valid IPv4", "192.168.1.1", true},
		{"valid IPv6", "2001:0db8:85a3:0000:0000:8a2e:0370:7334", true},
		{"valid IPv6 short", "::1", true},
		{"invalid IP", "999.999.999.999", false},
		{"invalid format", "not.an.ip", false},
		{"non-string", 123, false},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := IP.Validate(tc.value)
			if result.Valid != tc.isValid {
				t.Errorf("IP(%v) = %v, want %v", tc.value, result.Valid, tc.isValid)
			}
		})
	}
}

func TestUUID(t *testing.T) {
	testCases := []struct {
		name    string
		value   interface{}
		isValid bool
	}{
		{"valid UUID v4", "550e8400-e29b-41d4-a716-446655440000", true},
		{"valid UUID v1", "6ba7b810-9dad-11d1-80b4-00c04fd430c8", true},
		{"invalid UUID - wrong format", "550e8400-e29b-41d4-a716", false},
		{"invalid UUID - wrong characters", "550e8400-e29b-41d4-a716-44665544000g", false},
		{"non-string", 123, false},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := UUID.Validate(tc.value)
			if result.Valid != tc.isValid {
				t.Errorf("UUID(%v) = %v, want %v", tc.value, result.Valid, tc.isValid)
			}
		})
	}
}

// ===============================
// Numeric Validation Tests
// ===============================

func TestIsNumber(t *testing.T) {
	testCases := []struct {
		name    string
		value   interface{}
		isValid bool
	}{
		{"integer", 123, true},
		{"float", 123.45, true},
		{"string number", "123.45", true},
		{"string integer", "123", true},
		{"invalid string", "not a number", false},
		{"boolean", true, false},
		{"nil", nil, false},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := IsNumber.Validate(tc.value)
			if result.Valid != tc.isValid {
				t.Errorf("IsNumber(%v) = %v, want %v", tc.value, result.Valid, tc.isValid)
			}
		})
	}
}

func TestIsInteger(t *testing.T) {
	testCases := []struct {
		name    string
		value   interface{}
		isValid bool
	}{
		{"integer", 123, true},
		{"float", 123.45, false},
		{"string integer", "123", true},
		{"string float", "123.45", false},
		{"invalid string", "not a number", false},
		{"boolean", true, false},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := IsInteger.Validate(tc.value)
			if result.Valid != tc.isValid {
				t.Errorf("IsInteger(%v) = %v, want %v", tc.value, result.Valid, tc.isValid)
			}
		})
	}
}

func TestMin(t *testing.T) {
	validator := Min(10)
	
	testCases := []struct {
		name    string
		value   interface{}
		isValid bool
	}{
		{"above minimum", 15, true},
		{"at minimum", 10, true},
		{"below minimum", 5, false},
		{"string number above", "15", true},
		{"string number below", "5", false},
		{"invalid string", "not a number", false},
		{"non-numeric", true, false},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := validator.Validate(tc.value)
			if result.Valid != tc.isValid {
				t.Errorf("Min(10)(%v) = %v, want %v", tc.value, result.Valid, tc.isValid)
			}
		})
	}
}

func TestMax(t *testing.T) {
	validator := Max(100)
	
	testCases := []struct {
		name    string
		value   interface{}
		isValid bool
	}{
		{"below maximum", 50, true},
		{"at maximum", 100, true},
		{"above maximum", 150, false},
		{"string number below", "50", true},
		{"string number above", "150", false},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := validator.Validate(tc.value)
			if result.Valid != tc.isValid {
				t.Errorf("Max(100)(%v) = %v, want %v", tc.value, result.Valid, tc.isValid)
			}
		})
	}
}

func TestRange(t *testing.T) {
	validator := Range(10, 100)
	
	testCases := []struct {
		name    string
		value   interface{}
		isValid bool
	}{
		{"in range", 50, true},
		{"at minimum", 10, true},
		{"at maximum", 100, true},
		{"below range", 5, false},
		{"above range", 150, false},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := validator.Validate(tc.value)
			if result.Valid != tc.isValid {
				t.Errorf("Range(10, 100)(%v) = %v, want %v", tc.value, result.Valid, tc.isValid)
			}
		})
	}
}

// ===============================
// Date/Time Validation Tests
// ===============================

func TestIsDate(t *testing.T) {
	testCases := []struct {
		name    string
		value   interface{}
		isValid bool
	}{
		{"ISO date", "2023-12-25", true},
		{"US date", "12/25/2023", true},
		{"RFC3339", "2023-12-25T15:30:45Z", true},
		{"invalid date", "not a date", false},
		{"invalid format", "2023-13-45", false},
		{"non-string", 123, false},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := IsDate.Validate(tc.value)
			if result.Valid != tc.isValid {
				t.Errorf("IsDate(%v) = %v, want %v", tc.value, result.Valid, tc.isValid)
			}
		})
	}
}

func TestDateAfter(t *testing.T) {
	after := time.Date(2023, 12, 25, 0, 0, 0, 0, time.UTC)
	validator := DateAfter(after)
	
	testCases := []struct {
		name    string
		value   interface{}
		isValid bool
	}{
		{"date after", "2023-12-26", true},
		{"date before", "2023-12-24", false},
		{"same date", "2023-12-25", false},
		{"time.Time after", time.Date(2023, 12, 26, 0, 0, 0, 0, time.UTC), true},
		{"time.Time before", time.Date(2023, 12, 24, 0, 0, 0, 0, time.UTC), false},
		{"invalid date", "not a date", false},
		{"non-date", 123, false},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := validator.Validate(tc.value)
			if result.Valid != tc.isValid {
				t.Errorf("DateAfter(%v) = %v, want %v", tc.value, result.Valid, tc.isValid)
			}
		})
	}
}

// ===============================
// Collection Validation Tests
// ===============================

func TestIn(t *testing.T) {
	validator := In("red", "green", "blue")
	
	testCases := []struct {
		name    string
		value   interface{}
		isValid bool
	}{
		{"valid option", "red", true},
		{"another valid option", "blue", true},
		{"invalid option", "yellow", false},
		{"case sensitive", "Red", false},
		{"numeric", 123, false},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := validator.Validate(tc.value)
			if result.Valid != tc.isValid {
				t.Errorf("In('red', 'green', 'blue')(%v) = %v, want %v", tc.value, result.Valid, tc.isValid)
			}
		})
	}
}

func TestNotIn(t *testing.T) {
	validator := NotIn("admin", "root", "system")
	
	testCases := []struct {
		name    string
		value   interface{}
		isValid bool
	}{
		{"allowed value", "user", true},
		{"forbidden value", "admin", false},
		{"another forbidden", "root", false},
		{"case sensitive", "Admin", true},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := validator.Validate(tc.value)
			if result.Valid != tc.isValid {
				t.Errorf("NotIn('admin', 'root', 'system')(%v) = %v, want %v", tc.value, result.Valid, tc.isValid)
			}
		})
	}
}

// ===============================
// Business Validation Tests
// ===============================

func TestCreditCard(t *testing.T) {
	testCases := []struct {
		name    string
		value   interface{}
		isValid bool
	}{
		{"valid Visa", "4532015112830366", true},
		{"valid Visa with spaces", "4532 0151 1283 0366", true},
		{"valid Visa with dashes", "4532-0151-1283-0366", true},
		{"invalid checksum", "4532015112830367", false},
		{"too short", "4532015112", false},
		{"too long", "45320151128303661234", false},
		{"non-numeric", "4532-0151-128a-0366", false},
		{"non-string", 4532015112830366, false},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := CreditCard.Validate(tc.value)
			if result.Valid != tc.isValid {
				t.Errorf("CreditCard(%v) = %v, want %v", tc.value, result.Valid, tc.isValid)
			}
		})
	}
}

func TestPhone(t *testing.T) {
	testCases := []struct {
		name    string
		value   interface{}
		isValid bool
	}{
		{"US format", "(555) 123-4567", true},
		{"International format", "+1-555-123-4567", true},
		{"Simple format", "5551234567", true},
		{"Dotted format", "555.123.4567", true},
		{"Too short", "123456", false},
		{"Too long", "12345678901234567", false},
		{"Contains letters", "555-ABC-DEFG", false},
		{"Non-string", 5551234567, false},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := Phone.Validate(tc.value)
			if result.Valid != tc.isValid {
				t.Errorf("Phone(%v) = %v, want %v", tc.value, result.Valid, tc.isValid)
			}
		})
	}
}

// ===============================
// Validator Chain Tests
// ===============================

func TestValidatorChain(t *testing.T) {
	chain := NewValidatorChain("email").
		AddFunc(Required).
		AddFunc(Email)
	
	testCases := []struct {
		name    string
		value   interface{}
		isValid bool
	}{
		{"valid email", "user@example.com", true},
		{"empty string", "", false},
		{"invalid email", "not-an-email", false},
		{"nil value", nil, false},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := chain.Validate(tc.value)
			if result.Valid != tc.isValid {
				t.Errorf("Email chain(%v) = %v, want %v", tc.value, result.Valid, tc.isValid)
				if !result.Valid {
					t.Logf("Errors: %v", result.ErrorMessages())
				}
			}
		})
	}
}

func TestValidatorChainOptional(t *testing.T) {
	chain := NewValidatorChain("phone").
		AddFunc(Optional(Phone))
	
	testCases := []struct {
		name    string
		value   interface{}
		isValid bool
	}{
		{"valid phone", "555-123-4567", true},
		{"empty string", "", true},
		{"nil value", nil, true},
		{"invalid phone", "invalid", false},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := chain.Validate(tc.value)
			if result.Valid != tc.isValid {
				t.Errorf("Optional phone chain(%v) = %v, want %v", tc.value, result.Valid, tc.isValid)
			}
		})
	}
}

// ===============================
// Custom Validation Tests
// ===============================

func TestCustom(t *testing.T) {
	// Custom validator that checks if a string is all uppercase
	uppercaseValidator := Custom(func(value interface{}) (bool, string) {
		str, ok := value.(string)
		if !ok {
			return false, "must be a string"
		}
		
		if str != strings.ToUpper(str) {
			return false, "must be uppercase"
		}
		
		return true, ""
	})
	
	testCases := []struct {
		name    string
		value   interface{}
		isValid bool
	}{
		{"uppercase string", "HELLO", true},
		{"lowercase string", "hello", false},
		{"mixed case", "Hello", false},
		{"non-string", 123, false},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := uppercaseValidator.Validate(tc.value)
			if result.Valid != tc.isValid {
				t.Errorf("Custom uppercase(%v) = %v, want %v", tc.value, result.Valid, tc.isValid)
			}
		})
	}
}

// ===============================
// Validation Result Tests
// ===============================

func TestValidationResult(t *testing.T) {
	// Create a validation result with errors using the core framework
	result := validation.NewValidationResult()
	result.AddFieldError(validation.CodeRequired, "field1", "message1", "value1")
	result.AddFieldError(validation.CodeEmail, "field2", "message2", "value2")
	
	if result.Valid {
		t.Error("Expected result to be invalid after adding errors")
	}
	
	if len(result.Errors) != 2 {
		t.Errorf("Expected 2 errors, got %d", len(result.Errors))
	}
	
	// Test ErrorMessages
	messages := result.ErrorMessages()
	if len(messages) != 2 {
		t.Errorf("Expected 2 error messages, got %d", len(messages))
	}
	
	// Test FirstError
	firstError := result.FirstError()
	if firstError == nil {
		t.Error("Expected first error to be non-nil")
	}
	
	if firstError.Field != "field1" {
		t.Errorf("Expected first error field to be 'field1', got '%s'", firstError.Field)
	}
}

// ===============================
// Validate Function Tests
// ===============================

func TestValidate(t *testing.T) {
	rules := map[string]*ValidatorChain{
		"name": NewValidatorChain("name").
			AddFunc(Required).
			AddFunc(MinLength(2)),
		"email": NewValidatorChain("email").
			AddFunc(Required).
			AddFunc(Email),
		"age": NewValidatorChain("age").
			AddFunc(Optional(Min(0))),
	}
	
	t.Run("valid data", func(t *testing.T) {
		data := map[string]interface{}{
			"name":  "John Doe",
			"email": "john@example.com",
			"age":   25,
		}
		
		result := Validate(data, rules)
		if !result.Valid {
			t.Errorf("Expected validation to pass, got errors: %v", result.ErrorMessages())
		}
	})
	
	t.Run("invalid data", func(t *testing.T) {
		data := map[string]interface{}{
			"name":  "", // Required but empty
			"email": "invalid-email", // Invalid format
			"age":   -5, // Below minimum
		}
		
		result := Validate(data, rules)
		if result.Valid {
			t.Error("Expected validation to fail")
		}
		
		if len(result.Errors) == 0 {
			t.Error("Expected validation errors")
		}
	})
	
	t.Run("missing fields", func(t *testing.T) {
		data := map[string]interface{}{
			// Missing required fields
		}
		
		result := Validate(data, rules)
		if result.Valid {
			t.Error("Expected validation to fail for missing required fields")
		}
	})
}

// ===============================
// Convenience Function Tests
// ===============================

func TestConvenienceFunctions(t *testing.T) {
	t.Run("IsValidEmail", func(t *testing.T) {
		if !IsValidEmail("user@example.com") {
			t.Error("Expected valid email to return true")
		}
		
		if IsValidEmail("invalid-email") {
			t.Error("Expected invalid email to return false")
		}
	})
	
	t.Run("IsValidURL", func(t *testing.T) {
		if !IsValidURL("https://example.com") {
			t.Error("Expected valid URL to return true")
		}
		
		if IsValidURL("not-a-url") {
			t.Error("Expected invalid URL to return false")
		}
	})
	
	t.Run("IsValidIP", func(t *testing.T) {
		if !IsValidIP("192.168.1.1") {
			t.Error("Expected valid IP to return true")
		}
		
		if IsValidIP("999.999.999.999") {
			t.Error("Expected invalid IP to return false")
		}
	})
	
	t.Run("IsValidUUID", func(t *testing.T) {
		if !IsValidUUID("550e8400-e29b-41d4-a716-446655440000") {
			t.Error("Expected valid UUID to return true")
		}
		
		if IsValidUUID("not-a-uuid") {
			t.Error("Expected invalid UUID to return false")
		}
	})
	
	t.Run("IsValidCreditCard", func(t *testing.T) {
		if !IsValidCreditCard("4532015112830366") {
			t.Error("Expected valid credit card to return true")
		}
		
		if IsValidCreditCard("1234567890") {
			t.Error("Expected invalid credit card to return false")
		}
	})
	
	t.Run("IsValidPhone", func(t *testing.T) {
		if !IsValidPhone("555-123-4567") {
			t.Error("Expected valid phone to return true")
		}
		
		if IsValidPhone("123") {
			t.Error("Expected invalid phone to return false")
		}
	})
}

// ===============================
// Error Handling Tests
// ===============================

func TestValidationError(t *testing.T) {
	// Create a validation error using the core framework
	result := validation.NewValidationErrorWithField(
		validation.CodeEmail, 
		"email", 
		"must be a valid email address", 
		"invalid-email",
	)
	
	if result.Valid {
		t.Error("Expected validation result to be invalid")
	}
	
	if len(result.Errors) != 1 {
		t.Errorf("Expected 1 error, got %d", len(result.Errors))
	}
	
	err := result.Errors[0]
	if err.Field != "email" {
		t.Errorf("Expected field 'email', got '%s'", err.Field)
	}
	
	if err.Code != validation.CodeEmail {
		t.Errorf("Expected code '%s', got '%s'", validation.CodeEmail, err.Code)
	}
	
	if err.Message != "must be a valid email address" {
		t.Errorf("Expected message 'must be a valid email address', got '%s'", err.Message)
	}
}