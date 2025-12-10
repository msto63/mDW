// File: validationx_additional_test.go
// Title: Additional Validation Tests for Coverage Improvement
// Description: Tests for previously untested functions to achieve 90%+ coverage
// Author: msto63 with Claude Sonnet 4.0
// Version: v0.1.0
// Created: 2025-01-25

package validationx

import (
	"testing"
	"time"

	"github.com/msto63/mDW/foundation/core/validation"
)

// ===============================
// Tests for 0% Coverage Functions
// ===============================

func TestStartsWith(t *testing.T) {
	validator := StartsWith("prefix")
	
	t.Run("valid prefix", func(t *testing.T) {
		result := validator.Validate("prefix_test")
		if !result.Valid {
			t.Errorf("StartsWith() returned error for valid input: %v", result.ErrorMessages())
		}
	})
	
	t.Run("invalid prefix", func(t *testing.T) {
		result := validator.Validate("test_prefix")
		if result.Valid {
			t.Error("StartsWith() should return error for invalid prefix")
		}
	})
	
	t.Run("empty string", func(t *testing.T) {
		result := validator.Validate("")
		if result.Valid {
			t.Error("StartsWith() should return error for empty string")
		}
	})
	
	t.Run("exact match", func(t *testing.T) {
		result := validator.Validate("prefix")
		if !result.Valid {
			t.Errorf("StartsWith() returned error for exact match: %v", result.ErrorMessages())
		}
	})
	
	t.Run("case sensitive", func(t *testing.T) {
		result := validator.Validate("PREFIX_test")
		if result.Valid {
			t.Error("StartsWith() should be case sensitive")
		}
	})
}

func TestEndsWith(t *testing.T) {
	validator := EndsWith("suffix")
	
	t.Run("valid suffix", func(t *testing.T) {
		result := validator.Validate("test_suffix")
		if !result.Valid {
			t.Errorf("EndsWith() returned error for valid input: %v", result.ErrorMessages())
		}
	})
	
	t.Run("invalid suffix", func(t *testing.T) {
		result := validator.Validate("suffix_test")
		if result.Valid {
			t.Error("EndsWith() should return error for invalid suffix")
		}
	})
	
	t.Run("empty string", func(t *testing.T) {
		result := validator.Validate("")
		if result.Valid {
			t.Error("EndsWith() should return error for empty string")
		}
	})
	
	t.Run("exact match", func(t *testing.T) {
		result := validator.Validate("suffix")
		if !result.Valid {
			t.Errorf("EndsWith() returned error for exact match: %v", result.ErrorMessages())
		}
	})
	
	t.Run("case sensitive", func(t *testing.T) {
		result := validator.Validate("test_SUFFIX")
		if result.Valid {
			t.Error("EndsWith() should be case sensitive")
		}
	})
}

func TestIPv4(t *testing.T) {
	t.Run("valid IPv4", func(t *testing.T) {
		validIPs := []string{
			"192.168.1.1",
			"127.0.0.1",
			"10.0.0.1",
			"255.255.255.255",
			"0.0.0.0",
		}
		
		for _, ip := range validIPs {
			result := IPv4.Validate(ip)
			if !result.Valid {
				t.Errorf("IPv4() returned error for valid IP %s: %v", ip, result.ErrorMessages())
			}
		}
	})
	
	t.Run("invalid IPv4", func(t *testing.T) {
		invalidIPs := []string{
			"256.1.1.1",    // Out of range
			"192.168.1",    // Missing octet
			"192.168.1.1.1", // Extra octet
			"192.168.01.1", // Leading zero
			"192.168.-1.1", // Negative
			"192.168.1.a",  // Non-numeric
			"not.an.ip.address",
			"",
			"::1", // IPv6
		}
		
		for _, ip := range invalidIPs {
			result := IPv4.Validate(ip)
			if result.Valid {
				t.Errorf("IPv4() should return error for invalid IP %s", ip)
			}
		}
	})
}

func TestIPv6(t *testing.T) {
	t.Run("valid IPv6", func(t *testing.T) {
		validIPs := []string{
			"2001:0db8:85a3:0000:0000:8a2e:0370:7334",
			"2001:db8:85a3:0:0:8a2e:370:7334",
			"2001:db8:85a3::8a2e:370:7334",
			"::1",
			"::",
			"fe80::1",
			"2001:db8::1",
		}
		
		for _, ip := range validIPs {
			result := IPv6.Validate(ip)
			if !result.Valid {
				t.Errorf("IPv6() returned error for valid IP %s: %v", ip, result.ErrorMessages())
			}
		}
	})
	
	t.Run("invalid IPv6", func(t *testing.T) {
		invalidIPs := []string{
			"2001:0db8:85a3::8a2e:370g:7334", // Invalid character
			"2001:0db8:85a3:0000:0000:8a2e:0370:7334:extra", // Too many groups
			"192.168.1.1", // IPv4
			"not.an.ip.address",
			"",
			"12345::abcd", // Group too long
		}
		
		for _, ip := range invalidIPs {
			result := IPv6.Validate(ip)
			if result.Valid {
				t.Errorf("IPv6() should return error for invalid IP %s", ip)
			}
		}
	})
}

func TestDateBefore(t *testing.T) {
	// Test date: 2023-06-15
	testDate := time.Date(2023, 6, 15, 0, 0, 0, 0, time.UTC)
	validator := DateBefore(testDate)
	
	t.Run("valid date before", func(t *testing.T) {
		validDates := []string{
			"2023-06-14",
			"2023-06-01",
			"2022-12-31",
			"2020-01-01",
		}
		
		for _, dateStr := range validDates {
			result := validator.Validate(dateStr)
			if !result.Valid {
				t.Errorf("DateBefore() returned error for valid date %s: %v", dateStr, result.ErrorMessages())
			}
		}
	})
	
	t.Run("invalid date after", func(t *testing.T) {
		invalidDates := []string{
			"2023-06-15", // Same date
			"2023-06-16", // After
			"2023-12-31", // After
			"2024-01-01", // After
		}
		
		for _, dateStr := range invalidDates {
			result := validator.Validate(dateStr)
			if result.Valid {
				t.Errorf("DateBefore() should return error for date %s", dateStr)
			}
		}
	})
	
	t.Run("invalid date format", func(t *testing.T) {
		invalidFormats := []string{
			"not-a-date",
			"2023/06/14",
			"06-14-2023",
			"",
			"2023-13-01", // Invalid month
			"2023-06-32", // Invalid day
		}
		
		for _, dateStr := range invalidFormats {
			result := validator.Validate(dateStr)
			if result.Valid {
				t.Errorf("DateBefore() should return error for invalid format %s", dateStr)
			}
		}
	})
	
	t.Run("with time component", func(t *testing.T) {
		// Test with time included
		result := validator.Validate("2023-06-14T23:59:59Z")
		if !result.Valid {
			t.Errorf("DateBefore() should handle ISO datetime format: %v", result.ErrorMessages())
		}
	})
}

func TestValidateStruct(t *testing.T) {
	type TestStruct struct {
		Name  string `validate:"required,min_length:2"`
		Email string `validate:"required,email"`
		Age   int    `validate:"min:18,max:100"`
	}
	
	t.Run("valid struct", func(t *testing.T) {
		valid := TestStruct{
			Name:  "John Doe",
			Email: "john@example.com",
			Age:   25,
		}
		
		result := ValidateStruct(valid)
		if !result.Valid {
			t.Errorf("ValidateStruct() returned errors for valid struct: %v", result.ErrorMessages())
		}
	})
	
	t.Run("invalid struct", func(t *testing.T) {
		invalid := TestStruct{
			Name:  "J", // Too short
			Email: "invalid-email", // Invalid email
			Age:   15, // Too young
		}
		
		result := ValidateStruct(invalid)
		if result.Valid {
			t.Error("ValidateStruct() should return errors for invalid struct")
		}
		
		// Should have multiple errors
		errors := result.ErrorMessages()
		if len(errors) == 0 {
			t.Error("ValidateStruct() should return specific error messages")
		}
	})
	
	t.Run("struct with no validation tags", func(t *testing.T) {
		type NoValidation struct {
			Name string
			Age  int
		}
		
		test := NoValidation{Name: "", Age: 0}
		result := ValidateStruct(test)
		
		if !result.Valid {
			t.Error("ValidateStruct() should not return errors for struct without validation tags")
		}
	})
	
	t.Run("empty struct", func(t *testing.T) {
		type EmptyStruct struct{}
		
		test := EmptyStruct{}
		result := ValidateStruct(test)
		
		if !result.Valid {
			t.Error("ValidateStruct() should not return errors for empty struct")
		}
	})
	
	t.Run("struct with pointer fields", func(t *testing.T) {
		type PointerStruct struct {
			Name  *string `validate:"required"`
			Email *string `validate:"email"`
		}
		
		name := "John"
		email := "john@example.com"
		test := PointerStruct{
			Name:  &name,
			Email: &email,
		}
		
		result := ValidateStruct(test)
		if !result.Valid {
			t.Errorf("ValidateStruct() returned errors for valid pointer struct: %v", result.ErrorMessages())
		}
	})
}

// ===============================
// Tests for Low Coverage Functions
// ===============================

func TestFirstErrorEdgeCases(t *testing.T) {
	t.Run("no errors", func(t *testing.T) {
		result := validation.NewValidationResult()
		err := result.FirstError()
		if err != nil {
			t.Errorf("FirstError() should return nil for no errors, got %v", err)
		}
	})
}

func TestPatternEdgeCases(t *testing.T) {
	t.Run("invalid regex pattern", func(t *testing.T) {
		// This should handle invalid regex gracefully
		defer func() {
			if r := recover(); r != nil {
				// Pattern compilation should not panic in production code
				t.Error("Pattern() should handle invalid regex gracefully")
			}
		}()
		
		validator := Pattern("[invalid") // Invalid regex
		result := validator.Validate("test")
		if result.Valid {
			t.Error("Pattern() should return error for invalid regex")
		}
	})
}

func TestURLEdgeCases(t *testing.T) {
	t.Run("URL without scheme", func(t *testing.T) {
		result := URL.Validate("example.com")
		if result.Valid {
			t.Error("URL() should require scheme")
		}
	})
}

func TestIsNumberEdgeCases(t *testing.T) {
	t.Run("scientific notation", func(t *testing.T) {
		result := IsNumber.Validate("1.23e-4")
		if !result.Valid {
			t.Errorf("IsNumber() should accept scientific notation: %v", result.ErrorMessages())
		}
	})
}

func TestIsIntegerEdgeCases(t *testing.T) {
	t.Run("negative zero", func(t *testing.T) {
		result := IsInteger.Validate("-0")
		if !result.Valid {
			t.Errorf("IsInteger() should accept negative zero: %v", result.ErrorMessages())
		}
	})
}

func TestMinEdgeCases(t *testing.T) {
	validator := Min(10)
	
	t.Run("string conversion error", func(t *testing.T) {
		result := validator.Validate("not-a-number")
		if result.Valid {
			t.Error("Min() should return error for non-numeric strings")
		}
	})
	
	t.Run("exact minimum value", func(t *testing.T) {
		result := validator.Validate("10")
		if !result.Valid {
			t.Errorf("Min() should accept exact minimum value: %v", result.ErrorMessages())
		}
	})
	
	t.Run("float input", func(t *testing.T) {
		result := validator.Validate("10.5")
		if !result.Valid {
			t.Errorf("Min() should accept float values: %v", result.ErrorMessages())
		}
	})
}

func TestMaxEdgeCases(t *testing.T) {
	validator := Max(100)
	
	t.Run("string conversion error", func(t *testing.T) {
		result := validator.Validate("not-a-number")
		if result.Valid {
			t.Error("Max() should return error for non-numeric strings")
		}
	})
	
	t.Run("exact maximum value", func(t *testing.T) {
		result := validator.Validate("100")
		if !result.Valid {
			t.Errorf("Max() should accept exact maximum value: %v", result.ErrorMessages())
		}
	})
	
	t.Run("negative values", func(t *testing.T) {
		result := validator.Validate("-50")
		if !result.Valid {
			t.Errorf("Max() should accept negative values within range: %v", result.ErrorMessages())
		}
	})
}

func TestRangeEdgeCases(t *testing.T) {
	validator := Range(10, 100)
	
	t.Run("boundary values", func(t *testing.T) {
		// Test exact boundaries
		result1 := validator.Validate("10")
		result2 := validator.Validate("100")
		
		if !result1.Valid {
			t.Errorf("Range() should accept minimum boundary value: %v", result1.ErrorMessages())
		}
		if !result2.Valid {
			t.Errorf("Range() should accept maximum boundary value: %v", result2.ErrorMessages())
		}
	})
}