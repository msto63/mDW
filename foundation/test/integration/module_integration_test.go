// File: module_integration_test.go
// Title: mDW Foundation Module Integration Tests
// Description: Tests for cross-module interactions to ensure consistent behavior
//              across different foundation modules and error handling patterns.
// Author: msto63 with Claude Sonnet 4.0
// Version: v0.1.0
// Created: 2025-01-25
// Modified: 2025-01-25
//
// Change History:
// - 2025-01-25 v0.1.0: Initial implementation of integration tests

package integration

import (
	"errors"
	"strings"
	"testing"
	"time"

	mdwerror "github.com/msto63/mDW/foundation/core/error"
	mdwerrors "github.com/msto63/mDW/foundation/core/errors"
	"github.com/msto63/mDW/foundation/utils/mathx"
	"github.com/msto63/mDW/foundation/utils/stringx"
	"github.com/msto63/mDW/foundation/utils/timex"
)

// TestErrorHandlingIntegration verifies consistent error handling across modules
func TestErrorHandlingIntegration(t *testing.T) {
	t.Run("consistent error patterns", func(t *testing.T) {
		// Test that all modules use standardized error patterns
		
		// stringx error
		err1 := mdwerrors.InvalidInput("stringx", "validate", "", "non-empty string")
		if !mdwerrors.IsModuleOperation(err1, "stringx", "validate") {
			t.Error("stringx error doesn't match expected module/operation")
		}
		
		// mathx error
		err2 := mdwerrors.InvalidFormat("mathx", "invalid.decimal", "valid decimal format")
		module := mdwerrors.ExtractModule(err2)
		if module != "mathx" {
			t.Errorf("Expected module 'mathx', got '%s'", module)
		}
		
		// timex error  
		err3 := mdwerrors.ValidationFailed("timex", "timezone", "invalid/zone", "must be valid timezone")
		details := mdwerrors.ExtractDetails(err3)
		if details["field"] != "timezone" {
			t.Errorf("Expected field 'timezone', got %v", details["field"])
		}
	})
	
	t.Run("error severity consistency", func(t *testing.T) {
		// Validation errors should be low severity
		valErr := mdwerrors.ValidationFailed("stringx", "email", "invalid", "must be valid email")
		if valErr.Severity() != mdwerror.SeverityLow {
			t.Error("Validation errors should have low severity")
		}
		
		// Operation failures should be high severity
		opErr := mdwerrors.OperationFailed("filex", "read_file", errors.New("file read failed"))
		if opErr.Severity() != mdwerror.SeverityHigh {
			t.Error("Operation failures should have high severity")
		}
		
		// Input errors should be medium severity
		inputErr := mdwerrors.InvalidInput("mathx", "parse", "abc", "numeric value")
		if inputErr.Severity() != mdwerror.SeverityMedium {
			t.Error("Input errors should have medium severity")
		}
	})
}

// TestCrossModuleDataFlow tests data flow between modules
func TestCrossModuleDataFlow(t *testing.T) {
	t.Run("stringx to mathx conversion", func(t *testing.T) {
		// Test string validation followed by decimal conversion
		input := "123.45"
		
		// Step 1: Validate string format
		if err := stringx.ValidateRequired(input); err != nil {
			t.Fatalf("String validation failed: %v", err)
		}
		
		// Step 2: Convert to decimal using mathx
		decimal, err := mathx.NewDecimal(input)
		if err != nil {
			t.Fatalf("Decimal conversion failed: %v", err)
		}
		
		// Step 3: Verify the conversion
		if decimal.String() != "123.45" {
			t.Errorf("Expected '123.45', got '%s'", decimal.String())
		}
	})
	
	t.Run("stringx to timex parsing", func(t *testing.T) {
		// Test string validation followed by time parsing
		timeStr := "2023-12-25T10:30:00Z"
		
		// Step 1: Validate string is not blank
		if err := stringx.ValidateNotBlank(timeStr); err != nil {
			t.Fatalf("String validation failed: %v", err)
		}
		
		// Step 2: Parse time using timex
		parsedTime, err := timex.Parse(timeStr)
		if err != nil {
			t.Fatalf("Time parsing failed: %v", err)
		}
		
		// Step 3: Verify parsing
		expectedTime := time.Date(2023, 12, 25, 10, 30, 0, 0, time.UTC)
		if !parsedTime.Equal(expectedTime) {
			t.Errorf("Expected %v, got %v", expectedTime, parsedTime)
		}
	})
	
	t.Run("error propagation through modules", func(t *testing.T) {
		// Test error propagation from stringx through mathx
		invalidInput := "not-a-number"
		
		// Step 1: String validation passes (it's not empty)
		if err := stringx.ValidateRequired(invalidInput); err != nil {
			t.Fatalf("Unexpected string validation failure: %v", err)
		}
		
		// Step 2: Decimal conversion should fail with proper error
		_, err := mathx.NewDecimal(invalidInput)
		if err == nil {
			t.Fatal("Expected decimal conversion to fail")
		}
		
		// Step 3: Verify error contains context
		if !strings.Contains(err.Error(), "invalid") {
			t.Errorf("Error should indicate invalid input: %v", err)
		}
	})
}

// TestValidationIntegration tests validation patterns across modules
func TestValidationIntegration(t *testing.T) {
	t.Run("consistent validation error handling", func(t *testing.T) {
		// Test validation failures produce consistent error types
		var validationErrors []error
		
		// stringx validation
		if err := stringx.ValidateLength("x", 5, 10); err != nil {
			validationErrors = append(validationErrors, err)
		}
		
		// mathx validation (through error utils)
		if err := mdwerrors.ValidateRange("mathx", "value", -5, 0, 100); err != nil {
			validationErrors = append(validationErrors, err)
		}
		
		// timex validation (custom)
		if err := mdwerrors.ValidationFailed("timex", "date", "invalid", "must be valid date"); err != nil {
			validationErrors = append(validationErrors, err)
		}
		
		// Verify all validation errors have consistent structure
		for i, err := range validationErrors {
			if err == nil {
				continue
			}
			
			// All should contain validation context
			errStr := err.Error()
			if !strings.Contains(errStr, "validation") && !strings.Contains(errStr, "invalid") {
				t.Errorf("Validation error %d should contain validation context: %v", i, err)
			}
		}
	})
	
	t.Run("validation chain integration", func(t *testing.T) {
		// Test chaining validations from multiple modules
		testCases := []struct {
			name     string
			input    string
			minLen   int
			maxLen   int
			asDecimal bool
			expectError bool
		}{
			{"valid string and decimal", "123.45", 3, 10, true, false},
			{"too short string", "1", 3, 10, false, true},
			{"too long string", "very long string", 3, 10, false, true},
			{"valid string but invalid decimal", "abc", 3, 10, true, true},
			{"empty string", "", 1, 10, false, true},
		}
		
		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				// Step 1: String length validation
				err1 := stringx.ValidateLength(tc.input, tc.minLen, tc.maxLen)
				
				// Step 2: Decimal validation if requested
				var err2 error
				if tc.asDecimal && err1 == nil {
					_, err2 = mathx.NewDecimal(tc.input)
				}
				
				// Determine if any error occurred
				hasError := err1 != nil || err2 != nil
				
				if hasError != tc.expectError {
					t.Errorf("Expected error: %v, got error: %v (err1: %v, err2: %v)", 
						tc.expectError, hasError, err1, err2)
				}
			})
		}
	})
}

// TestPerformanceIntegration tests performance characteristics across modules
func TestPerformanceIntegration(t *testing.T) {
	t.Run("string processing performance", func(t *testing.T) {
		// Test performance of string operations that might be used together
		largeString := strings.Repeat("Hello World! ", 1000)
		
		start := time.Now()
		
		// Multiple string operations
		for i := 0; i < 100; i++ {
			// Validate
			_ = stringx.ValidateLength(largeString, 1, 20000)
			
			// Truncate
			truncated := stringx.Truncate(largeString, 50, "...")
			
			// Pad
			_ = stringx.PadLeft(truncated, 60, ' ')
		}
		
		duration := time.Since(start)
		
		// Should complete in reasonable time (less than 100ms for this workload)
		if duration > 100*time.Millisecond {
			t.Errorf("String operations took too long: %v", duration)
		}
	})
	
	t.Run("decimal calculation performance", func(t *testing.T) {
		// Test decimal operations that might be chained
		start := time.Now()
		
		d1, _ := mathx.NewDecimal("123.45")
		d2, _ := mathx.NewDecimal("67.89")
		
		for i := 0; i < 1000; i++ {
			result := d1.Add(d2)
			result = result.Multiply(mathx.NewDecimalFromFloat(1.1))
			_ = result.String() // Force string conversion
		}
		
		duration := time.Since(start)
		
		// Should complete in reasonable time (less than 50ms for this workload)
		if duration > 50*time.Millisecond {
			t.Errorf("Decimal operations took too long: %v", duration)
		}
	})
}

// TestErrorRecoveryIntegration tests error recovery patterns
func TestErrorRecoveryIntegration(t *testing.T) {
	t.Run("graceful error recovery", func(t *testing.T) {
		// Test that errors in one module don't break others
		
		// Step 1: Cause an error in mathx
		_, mathErr := mathx.NewDecimal("invalid")
		if mathErr == nil {
			t.Fatal("Expected mathx error")
		}
		
		// Step 2: Verify stringx still works normally
		if err := stringx.ValidateRequired("test"); err != nil {
			t.Errorf("stringx should work after mathx error: %v", err)
		}
		
		// Step 3: Verify timex still works normally
		now := time.Now()
		if age := timex.Age(now.AddDate(-25, 0, 0), now); age != 25 {
			t.Errorf("timex should work after mathx error, got age: %d", age)
		}
	})
	
	t.Run("error context preservation", func(t *testing.T) {
		// Test that error context is preserved through module boundaries
		
		// Create error with context
		originalErr := mdwerrors.InvalidInput("stringx", "parse", "input", "expected")
		
		// Wrap error in another module context
		wrappedErr := mdwerrors.OperationFailed("mathx", "convert", originalErr)
		
		// Verify both contexts are preserved
		details := mdwerrors.ExtractDetails(wrappedErr)
		if details == nil {
			t.Fatal("Error details should be preserved")
		}
		
		// Should have mathx context
		if details["module"] != "mathx" {
			t.Errorf("Expected outer module 'mathx', got %v", details["module"])
		}
		
		// Should have wrapped original error
		if !strings.Contains(wrappedErr.Error(), "stringx") {
			t.Error("Original stringx error context should be preserved")
		}
	})
}

// TestRealWorldScenarios tests realistic use cases combining multiple modules
func TestRealWorldScenarios(t *testing.T) {
	t.Run("financial calculation scenario", func(t *testing.T) {
		// Scenario: Process a financial transaction with validation
		amountStr := "1234.56"
		
		// Step 1: Validate input string
		if err := stringx.ValidateRequired(amountStr); err != nil {
			t.Fatalf("Amount validation failed: %v", err)
		}
		
		if err := stringx.ValidateLength(amountStr, 1, 20); err != nil {
			t.Fatalf("Amount length validation failed: %v", err)
		}
		
		// Step 2: Convert to decimal for precise calculation
		amount, err := mathx.NewDecimal(amountStr)
		if err != nil {
			t.Fatalf("Amount conversion failed: %v", err)
		}
		
		// Step 3: Calculate tax (8.5%)
		taxRate := mathx.NewDecimalFromFloat(0.085)
		tax := amount.Multiply(taxRate)
		
		// Step 4: Calculate total
		total := amount.Add(tax)
		
		// Step 5: Verify results
		expectedTax := "104.94" // 1234.56 * 0.085 = 104.9376, rounded to 104.94
		expectedTotal := "1339.50" // 1234.56 + 104.94 = 1339.50
		
		if tax.String() != expectedTax {
			t.Errorf("Expected tax %s, got %s", expectedTax, tax.String())
		}
		
		if total.String() != expectedTotal {
			t.Errorf("Expected total %s, got %s", expectedTotal, total.String())
		}
	})
	
	t.Run("date processing scenario", func(t *testing.T) {
		// Scenario: Process a date range for business day calculation
		startDateStr := "2023-12-25" // Christmas Day (holiday)
		endDateStr := "2024-01-05"   // Regular business day
		
		// Step 1: Validate date strings
		if err := stringx.ValidateNotBlank(startDateStr); err != nil {
			t.Fatalf("Start date validation failed: %v", err)
		}
		
		if err := stringx.ValidateNotBlank(endDateStr); err != nil {
			t.Fatalf("End date validation failed: %v", err)
		}
		
		// Step 2: Parse dates
		startDate, err := time.Parse("2006-01-02", startDateStr)
		if err != nil {
			t.Fatalf("Start date parsing failed: %v", err)
		}
		
		endDate, err := time.Parse("2006-01-02", endDateStr)
		if err != nil {
			t.Fatalf("End date parsing failed: %v", err)
		}
		
		// Step 3: Calculate business days
		businessDays := timex.BusinessDaysBetween(startDate, endDate)
		
		// Step 4: Verify calculation (should exclude weekends and potential holidays)
		if businessDays < 1 {
			t.Errorf("Expected at least 1 business day, got %d", businessDays)
		}
		
		// Should be reasonable for this date range (approximately 8-9 business days)
		if businessDays > 15 {
			t.Errorf("Business days calculation seems too high: %d", businessDays)
		}
	})
}