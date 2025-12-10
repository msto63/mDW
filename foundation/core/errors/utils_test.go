// File: utils_test.go
// Title: Shared Error Handling Utilities Tests
// Description: Tests for shared error handling utilities to ensure consistent
//              error patterns across all mDW foundation modules.
// Author: msto63 with Claude Sonnet 4.0
// Version: v0.1.0
// Created: 2025-01-25
// Modified: 2025-01-25

package errors

import (
	"errors"
	"testing"

	mdwerror "github.com/msto63/mDW/foundation/core/error"
)

func TestErrorBuilder(t *testing.T) {
	t.Run("basic error creation", func(t *testing.T) {
		err := NewErrorBuilder("testmodule").
			Operation("test_op").
			Message("test error").
			Detail("key", "value").
			Severity(mdwerror.SeverityHigh).
			Build()

		if err == nil {
			t.Fatal("Expected error, got nil")
		}

		details := err.Details()
		if details["module"] != "testmodule" {
			t.Errorf("Expected module 'testmodule', got %v", details["module"])
		}
		if details["operation"] != "test_op" {
			t.Errorf("Expected operation 'test_op', got %v", details["operation"])
		}
		if details["key"] != "value" {
			t.Errorf("Expected detail key 'value', got %v", details["key"])
		}
	})

	t.Run("error with cause", func(t *testing.T) {
		cause := errors.New("underlying error")
		err := NewErrorBuilder("testmodule").
			Operation("test_op").
			Cause(cause).
			Build()

		if err == nil {
			t.Fatal("Expected error, got nil")
		}

		if !errors.Is(err, cause) {
			t.Error("Expected error to wrap the cause")
		}
	})

	t.Run("auto-generated message", func(t *testing.T) {
		err := NewErrorBuilder("testmodule").
			Operation("test_op").
			Build()

		if err == nil {
			t.Fatal("Expected error, got nil")
		}

		expected := "testmodule.test_op failed"
		if err.Error() != expected {
			t.Errorf("Expected message '%s', got '%s'", expected, err.Error())
		}
	})
}

func TestInvalidInput(t *testing.T) {
	err := InvalidInput("testmodule", "test_op", "invalid", "valid string")
	
	if err == nil {
		t.Fatal("Expected error, got nil")
	}

	details := err.Details()
	if details["module"] != "testmodule" {
		t.Errorf("Expected module 'testmodule', got %v", details["module"])
	}
	if details["input"] != "invalid" {
		t.Errorf("Expected input 'invalid', got %v", details["input"])
	}
	if details["expected"] != "valid string" {
		t.Errorf("Expected 'valid string', got %v", details["expected"])
	}
}

func TestInvalidFormat(t *testing.T) {
	err := InvalidFormat("testmodule", "2023-13-45", "YYYY-MM-DD")
	
	if err == nil {
		t.Fatal("Expected error, got nil")
	}

	details := err.Details()
	if details["input"] != "2023-13-45" {
		t.Errorf("Expected input '2023-13-45', got %v", details["input"])
	}
	if details["expected_format"] != "YYYY-MM-DD" {
		t.Errorf("Expected format 'YYYY-MM-DD', got %v", details["expected_format"])
	}
}

func TestOperationFailed(t *testing.T) {
	cause := errors.New("file not found")
	err := OperationFailed("filex", "read_file", cause)
	
	if err == nil {
		t.Fatal("Expected error, got nil")
	}

	if !errors.Is(err, cause) {
		t.Error("Expected error to wrap the cause")
	}

	details := err.Details()
	if details["module"] != "filex" {
		t.Errorf("Expected module 'filex', got %v", details["module"])
	}
	if details["operation"] != "read_file" {
		t.Errorf("Expected operation 'read_file', got %v", details["operation"])
	}
}

func TestValidationFailed(t *testing.T) {
	err := ValidationFailed("stringx", "email", "invalid@", "must be valid email format")
	
	if err == nil {
		t.Fatal("Expected error, got nil")
	}

	details := err.Details()
	if details["field"] != "email" {
		t.Errorf("Expected field 'email', got %v", details["field"])
	}
	if details["value"] != "invalid@" {
		t.Errorf("Expected value 'invalid@', got %v", details["value"])
	}
	if details["reason"] != "must be valid email format" {
		t.Errorf("Expected reason 'must be valid email format', got %v", details["reason"])
	}
}

func TestOutOfRange(t *testing.T) {
	err := OutOfRange("mathx", "calculate", 150, 0, 100)
	
	if err == nil {
		t.Fatal("Expected error, got nil")
	}

	details := err.Details()
	if details["value"] != 150 {
		t.Errorf("Expected value 150, got %v", details["value"])
	}
	if details["min"] != 0 {
		t.Errorf("Expected min 0, got %v", details["min"])
	}
	if details["max"] != 100 {
		t.Errorf("Expected max 100, got %v", details["max"])
	}
}

func TestNotFound(t *testing.T) {
	err := NotFound("mapx", "get_key", "nonexistent")
	
	if err == nil {
		t.Fatal("Expected error, got nil")
	}

	details := err.Details()
	if details["identifier"] != "nonexistent" {
		t.Errorf("Expected identifier 'nonexistent', got %v", details["identifier"])
	}
}

func TestExtractDetails(t *testing.T) {
	err := InvalidInput("testmodule", "test_op", "input", "expected")
	
	details := ExtractDetails(err)
	if details == nil {
		t.Fatal("Expected details, got nil")
	}
	
	if details["module"] != "testmodule" {
		t.Errorf("Expected module 'testmodule', got %v", details["module"])
	}
}

func TestExtractModule(t *testing.T) {
	err := InvalidInput("testmodule", "test_op", "input", "expected")
	
	module := ExtractModule(err)
	if module != "testmodule" {
		t.Errorf("Expected module 'testmodule', got %s", module)
	}
}

func TestExtractOperation(t *testing.T) {
	err := InvalidInput("testmodule", "test_op", "input", "expected")
	
	operation := ExtractOperation(err)
	if operation != "test_op" {
		t.Errorf("Expected operation 'test_op', got %s", operation)
	}
}

func TestIsModuleOperation(t *testing.T) {
	err := InvalidInput("testmodule", "test_op", "input", "expected")
	
	if !IsModuleOperation(err, "testmodule", "test_op") {
		t.Error("Expected IsModuleOperation to return true")
	}
	
	if IsModuleOperation(err, "wrongmodule", "test_op") {
		t.Error("Expected IsModuleOperation to return false for wrong module")
	}
	
	if IsModuleOperation(err, "testmodule", "wrong_op") {
		t.Error("Expected IsModuleOperation to return false for wrong operation")
	}
}

func TestValidateRequired(t *testing.T) {
	tests := []struct {
		name      string
		value     interface{}
		wantError bool
	}{
		{"valid string", "hello", false},
		{"empty string", "", true},
		{"nil value", nil, true},
		{"valid slice", []int{1, 2, 3}, false},
		{"empty slice", []int{}, true},
		{"valid map", map[string]int{"a": 1}, false},
		{"empty map", map[string]int{}, true},
		{"valid int", 42, false},
		{"zero int", 0, false}, // zero is not considered empty for numbers
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateRequired("testmodule", "testfield", tt.value)
			if (err != nil) != tt.wantError {
				t.Errorf("ValidateRequired() error = %v, wantError %v", err, tt.wantError)
			}
		})
	}
}

func TestValidateRange(t *testing.T) {
	tests := []struct {
		name      string
		value     interface{}
		min       interface{}
		max       interface{}
		wantError bool
	}{
		{"valid int in range", 50, 0, 100, false},
		{"int below range", -10, 0, 100, true},
		{"int above range", 150, 0, 100, true},
		{"valid float in range", 50.5, 0.0, 100.0, false},
		{"float below range", -10.5, 0.0, 100.0, true},
		{"float above range", 150.5, 0.0, 100.0, true},
		{"exact min", 0, 0, 100, false},
		{"exact max", 100, 0, 100, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateRange("testmodule", "testfield", tt.value, tt.min, tt.max)
			if (err != nil) != tt.wantError {
				t.Errorf("ValidateRange() error = %v, wantError %v", err, tt.wantError)
			}
		})
	}
}

func TestToFloat64(t *testing.T) {
	tests := []struct {
		name      string
		value     interface{}
		expected  float64
		wantError bool
	}{
		{"int", 42, 42.0, false},
		{"int32", int32(42), 42.0, false},
		{"int64", int64(42), 42.0, false},
		{"float32", float32(42.5), 42.5, false},
		{"float64", 42.5, 42.5, false},
		{"string", "invalid", 0, true},
		{"bool", true, 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := toFloat64(tt.value)
			if (err != nil) != tt.wantError {
				t.Errorf("toFloat64() error = %v, wantError %v", err, tt.wantError)
				return
			}
			if !tt.wantError && result != tt.expected {
				t.Errorf("toFloat64() = %v, want %v", result, tt.expected)
			}
		})
	}
}