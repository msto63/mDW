// File: validation_test.go
// Title: Core Validation Framework Tests
// Description: Tests for the validation framework infrastructure including interfaces,
//              chains, error handling, and orchestration components. Does NOT test
//              concrete validators - those belong in utils/validationx.
// Author: msto63 with Claude Sonnet 4.0
// Version: v0.1.0
// Created: 2025-01-25
// Modified: 2025-01-25

package validation

import (
	"context"
	"strings"
	"testing"
)

func TestValidationResult(t *testing.T) {
	t.Run("NewValidationResult creates valid result", func(t *testing.T) {
		result := NewValidationResult()
		if !result.Valid {
			t.Error("Expected valid result")
		}
		if len(result.Errors) != 0 {
			t.Error("Expected no errors")
		}
	})

	t.Run("NewValidationError creates invalid result", func(t *testing.T) {
		result := NewValidationError(CodeRequired, "value required")
		if result.Valid {
			t.Error("Expected invalid result")
		}
		if len(result.Errors) != 1 {
			t.Errorf("Expected 1 error, got %d", len(result.Errors))
		}
		if result.Errors[0].Code != CodeRequired {
			t.Errorf("Expected code %s, got %s", CodeRequired, result.Errors[0].Code)
		}
	})

	t.Run("AddError adds error to result", func(t *testing.T) {
		result := NewValidationResult()
		result.AddError(CodeFormat, "invalid format")
		
		if result.Valid {
			t.Error("Expected invalid result after adding error")
		}
		if len(result.Errors) != 1 {
			t.Errorf("Expected 1 error, got %d", len(result.Errors))
		}
	})

	t.Run("FirstError returns first error", func(t *testing.T) {
		result := NewValidationResult()
		result.AddError(CodeRequired, "first error")
		result.AddError(CodeFormat, "second error")
		
		firstError := result.FirstError()
		if firstError == nil {
			t.Fatal("Expected first error")
		}
		if firstError.Message != "first error" {
			t.Errorf("Expected 'first error', got %s", firstError.Message)
		}
	})

	t.Run("ToError converts to standard error", func(t *testing.T) {
		result := NewValidationErrorWithField(CodeEmail, "email", "invalid email", "invalid@")
		err := result.ToError()
		
		if err == nil {
			t.Fatal("Expected error")
		}
		if !strings.Contains(err.Error(), "invalid email") {
			t.Errorf("Error should contain message: %s", err.Error())
		}
	})
}

func TestFrameworkUtilities(t *testing.T) {
	t.Run("GetValueLength function", func(t *testing.T) {
		tests := []struct {
			name     string
			value    interface{}
			expected int
		}{
			{"string", "hello", 5},
			{"empty string", "", 0},
			{"slice", []int{1, 2, 3}, 3},
			{"empty slice", []string{}, 0},
			{"map", map[string]int{"a": 1, "b": 2}, 2},
			{"empty map", map[string]interface{}{}, 0},
			{"nil", nil, 0},
			{"invalid type", 123, -1},
		}
		
		for _, test := range tests {
			t.Run(test.name, func(t *testing.T) {
				result := GetValueLength(test.value)
				if result != test.expected {
					t.Errorf("Expected length %d, got %d", test.expected, result)
				}
			})
		}
	})

	t.Run("ConvertToFloat64 function", func(t *testing.T) {
		tests := []struct {
			name     string
			value    interface{}
			expected float64
			shouldErr bool
		}{
			{"float64", 123.45, 123.45, false},
			{"int", 42, 42.0, false},
			{"string number", "99.9", 99.9, false},
			{"invalid string", "abc", 0, true},
			{"invalid type", []int{1, 2}, 0, true},
		}
		
		for _, test := range tests {
			t.Run(test.name, func(t *testing.T) {
				result, err := ConvertToFloat64(test.value)
				if test.shouldErr {
					if err == nil {
						t.Error("Expected error but got none")
					}
				} else {
					if err != nil {
						t.Errorf("Unexpected error: %v", err)
					}
					if result != test.expected {
						t.Errorf("Expected %f, got %f", test.expected, result)
					}
				}
			})
		}
	})

	t.Run("IsNilOrEmpty function", func(t *testing.T) {
		tests := []struct {
			name     string
			value    interface{}
			expected bool
		}{
			{"nil", nil, true},
			{"empty string", "", true},
			{"non-empty string", "hello", false},
			{"empty slice", []int{}, true},
			{"non-empty slice", []int{1, 2}, false},
			{"number", 42, false},
		}
		
		for _, test := range tests {
			t.Run(test.name, func(t *testing.T) {
				result := IsNilOrEmpty(test.value)
				if result != test.expected {
					t.Errorf("Expected %v, got %v", test.expected, result)
				}
			})
		}
	})
}

func TestValidatorChain(t *testing.T) {
	// Mock validators for testing framework components
	alwaysValid := ValidatorFunc(func(value interface{}) ValidationResult {
		return NewValidationResult()
	})
	
	alwaysInvalid := ValidatorFunc(func(value interface{}) ValidationResult {
		return NewValidationError(CodeCustom, "always fails")
	})
	
	requiredValidator := ValidatorFunc(func(value interface{}) ValidationResult {
		if value == nil || value == "" {
			return NewValidationError(CodeRequired, "value is required")
		}
		return NewValidationResult()
	})

	t.Run("Basic validator chain", func(t *testing.T) {
		chain := NewValidatorChain("test-chain").
			Add(alwaysValid).
			Add(alwaysValid)
		
		result := chain.Validate("test-value")
		if !result.Valid {
			t.Errorf("Expected valid result, got: %s", result.String())
		}
		
		// Test chain with failure
		failingChain := NewValidatorChain("failing-chain").
			Add(alwaysValid).
			Add(alwaysInvalid)
		
		result = failingChain.Validate("test-value")
		if result.Valid {
			t.Error("Expected invalid result")
		}
		if !result.HasError(CodeCustom) {
			t.Error("Expected CodeCustom error")
		}
	})

	t.Run("Chain with required validator", func(t *testing.T) {
		chain := NewValidatorChain("required-chain").
			Add(requiredValidator).
			Add(alwaysValid)
		
		// Valid case
		result := chain.Validate("valid-value")
		if !result.Valid {
			t.Errorf("Expected valid result, got: %s", result.String())
		}
		
		// Invalid case - empty value
		result = chain.Validate("")
		if result.Valid {
			t.Error("Expected invalid result for empty value")
		}
		if !result.HasError(CodeRequired) {
			t.Error("Expected CodeRequired error")
		}
	})

	t.Run("StopOnFirstError behavior", func(t *testing.T) {
		chain := NewValidatorChain("stop-on-first").
			StopOnFirstError(true).
			Add(alwaysInvalid).
			Add(alwaysInvalid)
		
		// Should stop at first validator
		result := chain.Validate("test-value")
		if result.Valid {
			t.Error("Expected validation to fail")
		}
		
		// Should have only one error due to StopOnFirstError
		if len(result.Errors) != 1 {
			t.Errorf("Expected exactly 1 error with StopOnFirstError, got %d", len(result.Errors))
		}
		
		// Test without StopOnFirstError - should collect all errors
		chain2 := NewValidatorChain("collect-all").
			StopOnFirstError(false).
			Add(alwaysInvalid).
			Add(alwaysInvalid)
		
		result2 := chain2.Validate("test-value")
		if len(result2.Errors) != 2 {
			t.Errorf("Expected 2 errors without StopOnFirstError, got %d", len(result2.Errors))
		}
	})

	t.Run("Context propagation", func(t *testing.T) {
		chain := NewValidatorChain("context-test").
			WithContext("testKey", "testValue")
		
		ctx := context.WithValue(context.Background(), "requestId", "req-123")
		result := chain.ValidateWithContext(ctx, "test@example.com")
		
		if result.Context == nil {
			t.Fatal("Expected context in result")
		}
		
		if result.Context["validatorChain"] != "context-test" {
			t.Error("Expected chain name in context")
		}
	})
}

func TestConditionalValidator(t *testing.T) {
	// Condition that checks if value is a string starting with "test"
	condition := func(value interface{}) bool {
		if str, ok := value.(string); ok {
			return strings.HasPrefix(str, "test")
		}
		return false
	}
	
	// Validator that always fails
	alwaysInvalid := ValidatorFunc(func(value interface{}) ValidationResult {
		return NewValidationError(CodeCustom, "conditional validation failed")
	})
	
	conditional := NewConditionalValidator(condition, alwaysInvalid, "test-conditional")
	
	// Should pass validation when condition is not met
	result := conditional.Validate("hello")
	if !result.Valid {
		t.Error("Expected validation to pass when condition not met")
		}
	
	// Should execute validator when condition is met
	result = conditional.Validate("test-value")
	if result.Valid {
		t.Error("Expected validation to fail when condition met")
	}
	if !result.HasError(CodeCustom) {
		t.Error("Expected CodeCustom error when condition met")
	}
	
	// Check context is added
	if result.Context["conditionalValidator"] != "test-conditional" {
		t.Error("Expected conditional validator name in context")
	}
	if result.Context["conditionMet"] != true {
		t.Error("Expected conditionMet=true in context")
	}
}

func TestParallelValidator(t *testing.T) {
	// Mock validators for parallel testing
	alwaysValid := ValidatorFunc(func(value interface{}) ValidationResult {
		return NewValidationResult()
	})
	
	alwaysInvalid1 := ValidatorFunc(func(value interface{}) ValidationResult {
		return NewValidationError(CodeCustom, "first error")
	})
	
	alwaysInvalid2 := ValidatorFunc(func(value interface{}) ValidationResult {
		return NewValidationError(CodeFormat, "second error")
	})
	
	parallel := NewParallelValidator("parallel-test").
		Add(alwaysValid).
		Add(alwaysValid)
	
	// Valid case - all validators pass
	result := parallel.Validate("test-value")
	if !result.Valid {
		t.Error("Expected valid result from parallel validation")
	}
	
	// Invalid case - multiple validators fail
	parallelFailing := NewParallelValidator("parallel-failing").
		Add(alwaysInvalid1).
		Add(alwaysInvalid2)
	
	result = parallelFailing.Validate("test-value")
	if result.Valid {
		t.Error("Expected invalid result from parallel validation")
	}
	
	// Should have errors from both validators
	if len(result.Errors) != 2 {
		t.Errorf("Expected 2 validation errors, got %d", len(result.Errors))
	}
	
	// Check context indicates parallel execution
	if result.Context == nil || !result.Context["parallelExecution"].(bool) {
		t.Error("Expected parallel execution context")
	}
	
	if result.Context["totalValidators"] != 2 {
		t.Error("Expected totalValidators=2 in context")
	}
}

func TestCombineResults(t *testing.T) {
	result1 := NewValidationResult()
	result2 := NewValidationError(CodeRequired, "required error")
	result3 := NewValidationError(CodeFormat, "format error")
	
	combined := Combine(result1, result2, result3)
	
	if combined.Valid {
		t.Error("Expected combined result to be invalid")
	}
	
	if len(combined.Errors) != 2 {
		t.Errorf("Expected 2 errors in combined result, got %d", len(combined.Errors))
	}
	
	// Check that both error codes are present
	if !combined.HasError(CodeRequired) {
		t.Error("Expected combined result to have required error")
	}
	if !combined.HasError(CodeFormat) {
		t.Error("Expected combined result to have format error")
	}
}

func TestErrorCodes(t *testing.T) {
	// Test that all standard error codes are defined
	codes := []string{
		CodeRequired, CodeFormat, CodeLength, CodeRange, CodeType, CodePattern, CodeCustom,
		CodeEmail, CodeURL, CodePhoneNumber, CodePassword, CodeNumeric, CodeDate, CodeTime,
		CodeJSON, CodeXML, CodePath, CodeFileExists, CodeFileType, CodePermission,
		CodeLocale, CodeLanguage, CodeCountry, CodeCurrency,
	}
	
	for _, code := range codes {
		if code == "" {
			t.Errorf("Error code should not be empty")
		}
		if !strings.HasPrefix(code, "VALIDATION_") {
			t.Errorf("Error code %s should start with VALIDATION_", code)
		}
	}
}

// Benchmark tests for framework components
func BenchmarkValidationResult(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result := NewValidationResult()
		result.AddError(CodeRequired, "test error")
	}
}

func BenchmarkValidatorChain(b *testing.B) {
	alwaysValid := ValidatorFunc(func(value interface{}) ValidationResult {
		return NewValidationResult()
	})
	
	chain := NewValidatorChain("benchmark").
		Add(alwaysValid).
		Add(alwaysValid).
		Add(alwaysValid)
	
	value := "test-value"
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		chain.Validate(value)
	}
}

func BenchmarkParallelValidator(b *testing.B) {
	alwaysValid := ValidatorFunc(func(value interface{}) ValidationResult {
		return NewValidationResult()
	})
	
	parallel := NewParallelValidator("benchmark").
		Add(alwaysValid).
		Add(alwaysValid).
		Add(alwaysValid).
		Add(alwaysValid)
	
	value := "test-value"
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		parallel.Validate(value)
	}
}

func BenchmarkUtilityFunctions(b *testing.B) {
	b.Run("GetValueLength", func(b *testing.B) {
		value := "hello world"
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			GetValueLength(value)
		}
	})
	
	b.Run("ConvertToFloat64", func(b *testing.B) {
		value := "123.45"
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			ConvertToFloat64(value)
		}
	})
	
	b.Run("IsNilOrEmpty", func(b *testing.B) {
		value := "hello"
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			IsNilOrEmpty(value)
		}
	})
}