// File: error_integration_test.go
// Title: Error Handling Integration Tests
// Description: Comprehensive tests for error handling patterns across all
//              mDW foundation modules to ensure consistent error behavior.
// Author: msto63 with Claude Sonnet 4.0
// Version: v0.1.0
// Created: 2025-01-25
// Modified: 2025-01-25
//
// Change History:
// - 2025-01-25 v0.1.0: Initial implementation of error integration tests

package integration

import (
	"errors"
	"strings"
	"testing"

	mdwerror "github.com/msto63/mDW/foundation/core/error"
	mdwerrors "github.com/msto63/mDW/foundation/core/errors"
	"github.com/msto63/mDW/foundation/utils/mathx"
	"github.com/msto63/mDW/foundation/utils/stringx"
)

// TestStandardizedErrorFormats verifies all modules use consistent error formats
func TestStandardizedErrorFormats(t *testing.T) {
	t.Run("all modules use mDW error types", func(t *testing.T) {
		testCases := []struct {
			name     string
			errorFunc func() error
			module   string
		}{
			{
				name: "stringx validation error",
				errorFunc: func() error {
					return mdwerrors.ValidationFailed("stringx", "email", "invalid", "must be valid email")
				},
				module: "stringx",
			},
			{
				name: "mathx invalid input error",
				errorFunc: func() error {
					return mdwerrors.InvalidInput("mathx", "parse", "abc", "numeric value")
				},
				module: "mathx",
			},
			{
				name: "filex operation error",
				errorFunc: func() error {
					cause := errors.New("file not found")
					return mdwerrors.OperationFailed("filex", "read", cause)
				},
				module: "filex",
			},
		}
		
		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				err := tc.errorFunc()
				
				// Should be a mDW error
				mdwErr, ok := err.(*mdwerror.Error)
				if !ok {
					t.Fatalf("Error should be *mdwerror.Error, got %T", err)
				}
				
				// Should have module in details
				details := mdwErr.Details()
				if details == nil {
					t.Fatal("Error should have details")
				}
				
				if details["module"] != tc.module {
					t.Errorf("Expected module '%s', got '%v'", tc.module, details["module"])
				}
				
				// Should have a severity
				severity := mdwErr.Severity()
				if severity < mdwerror.SeverityLow || severity > mdwerror.SeverityCritical {
					t.Error("Error should have a valid severity")
				}
				
				// Should have a code
				code := mdwErr.Code()
				if string(code) == "" {
					t.Error("Error should have a code")
				}
			})
		}
	})
}

// TestErrorSeverityConsistency verifies severity levels are used consistently
func TestErrorSeverityConsistency(t *testing.T) {
	t.Run("validation errors are low severity", func(t *testing.T) {
		validationErrors := []error{
			mdwerrors.ValidationFailed("stringx", "email", "invalid", "must be valid email"),
			mdwerrors.ValidationFailed("mathx", "amount", "-5", "must be positive"),
			mdwerrors.ValidationFailed("timex", "date", "invalid", "must be valid date"),
		}
		
		for i, err := range validationErrors {
			if mdwErr, ok := err.(*mdwerror.Error); ok {
				if mdwErr.Severity() != mdwerror.SeverityLow {
					t.Errorf("Validation error %d should have low severity, got %v", 
						i, mdwErr.Severity())
				}
			}
		}
	})
	
	t.Run("operation failures are high severity", func(t *testing.T) {
		operationErrors := []error{
			mdwerrors.OperationFailed("filex", "read", errors.New("permission denied")),
			mdwerrors.OperationFailed("mathx", "divide", errors.New("division by zero")),
			mdwerrors.OperationFailed("timex", "parse", errors.New("invalid format")),
		}
		
		for i, err := range operationErrors {
			if mdwErr, ok := err.(*mdwerror.Error); ok {
				if mdwErr.Severity() != mdwerror.SeverityHigh {
					t.Errorf("Operation error %d should have high severity, got %v", 
						i, mdwErr.Severity())
				}
			}
		}
	})
	
	t.Run("input errors are medium severity", func(t *testing.T) {
		inputErrors := []error{
			mdwerrors.InvalidInput("stringx", "validate", "", "non-empty string"),
			mdwerrors.InvalidInput("mathx", "parse", "abc", "numeric value"),
			mdwerrors.InvalidFormat("timex", "2023-13-45", "YYYY-MM-DD"),
		}
		
		for i, err := range inputErrors {
			if mdwErr, ok := err.(*mdwerror.Error); ok {
				if mdwErr.Severity() != mdwerror.SeverityMedium {
					t.Errorf("Input error %d should have medium severity, got %v", 
						i, mdwErr.Severity())
				}
			}
		}
	})
}

// TestErrorCodeConsistency verifies error codes follow consistent patterns
func TestErrorCodeConsistency(t *testing.T) {
	t.Run("module-specific error codes", func(t *testing.T) {
		testCases := []struct {
			module       string
			operation    string
			expectedCode string
		}{
			{"stringx", "format", "STRINGX_INVALID_FORMAT"},
			{"mathx", "divide", "MATHX_DIVISION_BY_ZERO"},
			{"filex", "read", "FILEX_READ_FAILED"},
			{"timex", "parse", "TIMEX_PARSE_ERROR"},
		}
		
		for _, tc := range testCases {
			t.Run(tc.module+"_"+tc.operation, func(t *testing.T) {
				var err error
				
				switch tc.module {
				case "stringx":
					err = mdwerrors.InvalidFormat(tc.module, "invalid", "valid format")
				case "mathx":
					err = mdwerrors.OperationFailed(tc.module, tc.operation, errors.New("division by zero"))
				case "filex":
					err = mdwerrors.OperationFailed(tc.module, tc.operation, errors.New("read failed"))
				case "timex":
					err = mdwerrors.OperationFailed(tc.module, tc.operation, errors.New("parse error"))
				}
				
				if mdwErr, ok := err.(*mdwerror.Error); ok {
					code := string(mdwErr.Code())
					if !strings.Contains(code, strings.ToUpper(tc.module)) {
						t.Errorf("Error code '%s' should contain module name '%s'", 
							code, strings.ToUpper(tc.module))
					}
				}
			})
		}
	})
}

// TestErrorWrappingAndUnwrapping verifies error wrapping works correctly
func TestErrorWrappingAndUnwrapping(t *testing.T) {
	t.Run("error wrapping preserves original error", func(t *testing.T) {
		originalErr := errors.New("original error message")
		wrappedErr := mdwerrors.OperationFailed("testmodule", "test_op", originalErr)
		
		// Should be able to unwrap to original error
		if !errors.Is(wrappedErr, originalErr) {
			t.Error("Wrapped error should be detectable with errors.Is")
		}
		
		// Should contain original error message
		if !strings.Contains(wrappedErr.Error(), "original error message") {
			t.Error("Wrapped error should contain original error message")
		}
	})
	
	t.Run("multiple levels of wrapping", func(t *testing.T) {
		// Level 1: Original error
		originalErr := errors.New("file not found")
		
		// Level 2: Module-specific error
		moduleErr := mdwerrors.OperationFailed("filex", "read", originalErr)
		
		// Level 3: Higher-level operation error
		serviceErr := mdwerrors.OperationFailed("service", "process_file", moduleErr)
		
		// Should be able to find original error
		if !errors.Is(serviceErr, originalErr) {
			t.Error("Should be able to unwrap through multiple levels")
		}
		
		// Should preserve context from all levels
		errMsg := serviceErr.Error()
		if !strings.Contains(errMsg, "service") {
			t.Error("Error should contain service context")
		}
		if !strings.Contains(errMsg, "filex") {
			t.Error("Error should contain filex context")
		}
	})
}

// TestErrorContextPreservation verifies error context is preserved across module boundaries
func TestErrorContextPreservation(t *testing.T) {
	t.Run("context accumulation", func(t *testing.T) {
		// Start with a validation error
		validationErr := mdwerrors.ValidationFailed("stringx", "email", "invalid@", "must be valid email")
		
		// Wrap in processing error
		processingErr := mdwerrors.OperationFailed("processor", "validate_input", validationErr)
		
		// Extract details from final error
		details := mdwerrors.ExtractDetails(processingErr)
		if details == nil {
			t.Fatal("Error should have details")
		}
		
		// Should have current module context
		if details["module"] != "processor" {
			t.Errorf("Expected current module 'processor', got %v", details["module"])
		}
		
		// Should still be able to find original validation context
		if !strings.Contains(processingErr.Error(), "stringx") {
			t.Error("Should preserve original stringx context")
		}
		
		if !strings.Contains(processingErr.Error(), "email") {
			t.Error("Should preserve original field context")
		}
	})
}

// TestErrorBuilderIntegration verifies the error builder works across modules
func TestErrorBuilderIntegration(t *testing.T) {
	t.Run("fluent error building", func(t *testing.T) {
		err := mdwerrors.NewErrorBuilder("integration_test").
			Operation("test_operation").
			Message("test error message").
			Detail("test_key", "test_value").
			Detail("numeric_value", 42).
			Severity(mdwerror.SeverityHigh).
			Code("TEST_ERROR").
			Build()
		
		// Verify all properties are set correctly
		if err == nil {
			t.Fatal("Builder should create an error")
		}
		
		if err.Error() != "test error message" {
			t.Errorf("Expected 'test error message', got '%s'", err.Error())
		}
		
		if err.Code() != mdwerror.Code("TEST_ERROR") {
			t.Errorf("Expected code 'TEST_ERROR', got '%s'", err.Code())
		}
		
		if err.Severity() != mdwerror.SeverityHigh {
			t.Errorf("Expected high severity, got %v", err.Severity())
		}
		
		details := err.Details()
		if details["module"] != "integration_test" {
			t.Errorf("Expected module 'integration_test', got %v", details["module"])
		}
		
		if details["operation"] != "test_operation" {
			t.Errorf("Expected operation 'test_operation', got %v", details["operation"])
		}
		
		if details["test_key"] != "test_value" {
			t.Errorf("Expected test_key 'test_value', got %v", details["test_key"])
		}
		
		if details["numeric_value"] != 42 {
			t.Errorf("Expected numeric_value 42, got %v", details["numeric_value"])
		}
	})
	
	t.Run("auto-generated properties", func(t *testing.T) {
		// Test auto-generation of message and code
		err := mdwerrors.NewErrorBuilder("auto_test").
			Operation("auto_operation").
			Build()
		
		if err == nil {
			t.Fatal("Builder should create an error")
		}
		
		// Should auto-generate message
		expectedMessage := "auto_test.auto_operation failed"
		if err.Error() != expectedMessage {
			t.Errorf("Expected auto-generated message '%s', got '%s'", 
				expectedMessage, err.Error())
		}
		
		// Should auto-generate code
		code := string(err.Code())
		if code == "" {
			t.Error("Should auto-generate error code")
		}
	})
}

// TestRealWorldErrorScenarios tests realistic error scenarios
func TestRealWorldErrorScenarios(t *testing.T) {
	t.Run("input validation chain", func(t *testing.T) {
		// Simulate processing user input through multiple validation layers
		userInput := ""
		
		// Layer 1: Basic string validation
		if err := stringx.ValidateRequired(userInput); err != nil {
			// Convert standard error to mDW error
			mdwErr := mdwerrors.ValidationFailed("input_processor", "user_input", userInput, "input is required")
			
			// Verify error structure
			if mdwErr.Severity() != mdwerror.SeverityLow {
				t.Error("Validation errors should be low severity")
			}
			
			details := mdwErr.Details()
			if details["field"] != "user_input" {
				t.Error("Should preserve field context")
			}
		} else {
			t.Error("Should fail validation for empty input")
		}
	})
	
	t.Run("data conversion pipeline", func(t *testing.T) {
		// Simulate data conversion with error propagation
		invalidDecimal := "not-a-number"
		
		// Step 1: String validation passes
		if err := stringx.ValidateRequired(invalidDecimal); err != nil {
			t.Fatalf("String validation should pass: %v", err)
		}
		
		// Step 2: Decimal conversion fails
		_, err := mathx.NewDecimal(invalidDecimal)
		if err == nil {
			t.Fatal("Decimal conversion should fail")
		}
		
		// Step 3: Wrap in higher-level context
		serviceErr := mdwerrors.OperationFailed("financial_service", "process_amount", err)
		
		// Verify error chain
		if !errors.Is(serviceErr, err) {
			t.Error("Should preserve original error in chain")
		}
		
		// Verify context
		if !strings.Contains(serviceErr.Error(), "financial_service") {
			t.Error("Should include service context")
		}
		
		if !strings.Contains(serviceErr.Error(), "process_amount") {
			t.Error("Should include operation context")
		}
	})
	
	t.Run("error recovery patterns", func(t *testing.T) {
		// Test graceful error handling and recovery
		errors := []error{
			mdwerrors.InvalidInput("module1", "op1", "bad", "good"),
			mdwerrors.ValidationFailed("module2", "field", "value", "reason"),
			mdwerrors.OperationFailed("module3", "op3", errors.New("underlying")),
		}
		
		// Process errors with different recovery strategies
		for i, err := range errors {
			if mdwErr, ok := err.(*mdwerror.Error); ok {
				severity := mdwErr.Severity()
				
				switch severity {
				case mdwerror.SeverityLow:
					// Low severity: log and continue
					t.Logf("Low severity error %d: %v", i, err)
				case mdwerror.SeverityMedium:
					// Medium severity: retry with different input
					t.Logf("Medium severity error %d: %v", i, err)
				case mdwerror.SeverityHigh:
					// High severity: escalate
					t.Logf("High severity error %d: %v", i, err)
				case mdwerror.SeverityCritical:
					// Critical: abort
					t.Logf("Critical error %d: %v", i, err)
				}
			}
		}
	})
}

// TestErrorUtilityFunctions verifies error utility functions work correctly
func TestErrorUtilityFunctions(t *testing.T) {
	t.Run("error analysis functions", func(t *testing.T) {
		err := mdwerrors.InvalidInput("testmodule", "test_op", "input", "expected")
		
		// Test module extraction
		module := mdwerrors.ExtractModule(err)
		if module != "testmodule" {
			t.Errorf("Expected module 'testmodule', got '%s'", module)
		}
		
		// Test operation extraction
		operation := mdwerrors.ExtractOperation(err)
		if operation != "test_op" {
			t.Errorf("Expected operation 'test_op', got '%s'", operation)
		}
		
		// Test module/operation checking
		if !mdwerrors.IsModuleOperation(err, "testmodule", "test_op") {
			t.Error("Should match correct module and operation")
		}
		
		if mdwerrors.IsModuleOperation(err, "wrongmodule", "test_op") {
			t.Error("Should not match wrong module")
		}
		
		if mdwerrors.IsModuleOperation(err, "testmodule", "wrong_op") {
			t.Error("Should not match wrong operation")
		}
	})
	
	t.Run("validation utility functions", func(t *testing.T) {
		// Test required validation
		if err := mdwerrors.ValidateRequired("testmodule", "field", nil); err == nil {
			t.Error("Should fail validation for nil value")
		}
		
		if err := mdwerrors.ValidateRequired("testmodule", "field", ""); err == nil {
			t.Error("Should fail validation for empty string")
		}
		
		if err := mdwerrors.ValidateRequired("testmodule", "field", "valid"); err != nil {
			t.Errorf("Should pass validation for valid string: %v", err)
		}
		
		// Test range validation
		if err := mdwerrors.ValidateRange("testmodule", "field", 150, 0, 100); err == nil {
			t.Error("Should fail validation for out of range value")
		}
		
		if err := mdwerrors.ValidateRange("testmodule", "field", 50, 0, 100); err != nil {
			t.Errorf("Should pass validation for in-range value: %v", err)
		}
	})
}