// File: error_test.go
// Title: Error Module Tests
// Description: Comprehensive tests for the error module covering all functionality
//              including error creation, wrapping, codes, severity, and metadata.
// Author: msto63 with Claude Sonnet 4.0
// Version: v0.1.0
// Created: 2025-01-24
// Modified: 2025-01-24
//
// Change History:
// - 2025-01-24 v0.1.0: Initial implementation with comprehensive test coverage

package error

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

func TestNew(t *testing.T) {
	msg := "test error message"
	err := New(msg)
	
	if err == nil {
		t.Fatal("New() returned nil")
	}
	
	if err.Error() != msg {
		t.Errorf("Error() = %q, want %q", err.Error(), msg)
	}
	
	if err.Code() != CodeUnknown {
		t.Errorf("Code() = %v, want %v", err.Code(), CodeUnknown)
	}
	
	if err.Severity() != SeverityMedium {
		t.Errorf("Severity() = %v, want %v", err.Severity(), SeverityMedium)
	}
	
	if err.Timestamp().IsZero() {
		t.Error("Timestamp() should not be zero")
	}
	
	if len(err.StackTrace()) == 0 {
		t.Error("StackTrace() should not be empty")
	}
}

func TestWrap(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		message  string
		wantNil  bool
		wantMsg  string
	}{
		{
			name:    "wrap nil error",
			err:     nil,
			message: "wrapper message",
			wantNil: true,
		},
		{
			name:    "wrap standard error",
			err:     errors.New("original error"),
			message: "wrapper message",
			wantMsg: "wrapper message: original error",
		},
		{
			name:    "wrap mDW error",
			err:     New("original mDW error").WithCode(CodeDatabaseError),
			message: "wrapper message",
			wantMsg: "wrapper message: original mDW error",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wrapped := Wrap(tt.err, tt.message)
			
			if tt.wantNil {
				if wrapped != nil {
					t.Errorf("Wrap() = %v, want nil", wrapped)
				}
				return
			}
			
			if wrapped == nil {
				t.Fatal("Wrap() returned nil")
			}
			
			if wrapped.Error() != tt.wantMsg {
				t.Errorf("Error() = %q, want %q", wrapped.Error(), tt.wantMsg)
			}
			
			// Test that mDW error properties are preserved
			if mdwErr, ok := tt.err.(*Error); ok {
				if wrapped.Code() != mdwErr.Code() {
					t.Errorf("Code() = %v, want %v", wrapped.Code(), mdwErr.Code())
				}
			}
		})
	}
}

func TestErrorChaining(t *testing.T) {
	original := errors.New("root cause")
	middle := Wrap(original, "middle layer")
	top := Wrap(middle, "top layer")
	
	// Test error messages
	expected := "top layer: middle layer: root cause"
	if top.Error() != expected {
		t.Errorf("Error() = %q, want %q", top.Error(), expected)
	}
	
	// Test unwrapping
	if !errors.Is(top, middle) {
		t.Error("errors.Is() should find middle layer")
	}
	
	if !errors.Is(top, original) {
		t.Error("errors.Is() should find original error")
	}
	
	// Test root cause
	rootCause := top.RootCause()
	if rootCause != original {
		t.Errorf("RootCause() = %v, want %v", rootCause, original)
	}
}

func TestWithCode(t *testing.T) {
	err := New("test error").WithCode(CodeDatabaseError)
	
	if err.Code() != CodeDatabaseError {
		t.Errorf("Code() = %v, want %v", err.Code(), CodeDatabaseError)
	}
	
	// Should auto-set severity based on code
	expectedSeverity := GetSeverityFromCode(CodeDatabaseError)
	if err.Severity() != expectedSeverity {
		t.Errorf("Severity() = %v, want %v", err.Severity(), expectedSeverity)
	}
}

func TestWithSeverity(t *testing.T) {
	err := New("test error").WithSeverity(SeverityCritical)
	
	if err.Severity() != SeverityCritical {
		t.Errorf("Severity() = %v, want %v", err.Severity(), SeverityCritical)
	}
}

func TestWithDetail(t *testing.T) {
	err := New("test error").
		WithDetail("key1", "value1").
		WithDetail("key2", 42)
	
	details := err.Details()
	
	if len(details) != 2 {
		t.Errorf("Details() length = %d, want 2", len(details))
	}
	
	if details["key1"] != "value1" {
		t.Errorf("Details()[\"key1\"] = %v, want \"value1\"", details["key1"])
	}
	
	if details["key2"] != 42 {
		t.Errorf("Details()[\"key2\"] = %v, want 42", details["key2"])
	}
}

func TestWithDetails(t *testing.T) {
	details := map[string]interface{}{
		"key1": "value1",
		"key2": 42,
		"key3": true,
	}
	
	err := New("test error").WithDetails(details)
	
	errDetails := err.Details()
	if len(errDetails) != 3 {
		t.Errorf("Details() length = %d, want 3", len(errDetails))
	}
	
	for k, v := range details {
		if errDetails[k] != v {
			t.Errorf("Details()[%q] = %v, want %v", k, errDetails[k], v)
		}
	}
}

func TestWithContext(t *testing.T) {
	context := "user-service.CreateUser"
	err := New("test error").WithContext(context)
	
	if err.Context() != context {
		t.Errorf("Context() = %q, want %q", err.Context(), context)
	}
}

func TestWithOperation(t *testing.T) {
	operation := "INSERT INTO users"
	err := New("test error").WithOperation(operation)
	
	if err.Operation() != operation {
		t.Errorf("Operation() = %q, want %q", err.Operation(), operation)
	}
}

func TestWithUserID(t *testing.T) {
	userID := "user123"
	err := New("test error").WithUserID(userID)
	
	if err.UserID() != userID {
		t.Errorf("UserID() = %q, want %q", err.UserID(), userID)
	}
}

func TestWithRequestID(t *testing.T) {
	requestID := "req-abc-123"
	err := New("test error").WithRequestID(requestID)
	
	if err.RequestID() != requestID {
		t.Errorf("RequestID() = %q, want %q", err.RequestID(), requestID)
	}
}

func TestWithMessage(t *testing.T) {
	key := "error.database.connection"
	args := map[string]interface{}{
		"host": "localhost",
		"port": 5432,
	}
	
	err := New("test error").WithMessage(key, args)
	
	if err.MessageKey() != key {
		t.Errorf("MessageKey() = %q, want %q", err.MessageKey(), key)
	}
	
	msgArgs := err.MessageArgs()
	if len(msgArgs) != 2 {
		t.Errorf("MessageArgs() length = %d, want 2", len(msgArgs))
	}
	
	if msgArgs["host"] != "localhost" {
		t.Errorf("MessageArgs()[\"host\"] = %v, want \"localhost\"", msgArgs["host"])
	}
}

func TestHasCode(t *testing.T) {
	tests := []struct {
		name string
		err  error
		code Code
		want bool
	}{
		{
			name: "mDW error with matching code",
			err:  New("test").WithCode(CodeDatabaseError),
			code: CodeDatabaseError,
			want: true,
		},
		{
			name: "mDW error with different code",
			err:  New("test").WithCode(CodeDatabaseError),
			code: CodeNotFound,
			want: false,
		},
		{
			name: "standard error",
			err:  errors.New("standard error"),
			code: CodeDatabaseError,
			want: false,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := HasCode(tt.err, tt.code); got != tt.want {
				t.Errorf("HasCode() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetCode(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want Code
	}{
		{
			name: "mDW error",
			err:  New("test").WithCode(CodeDatabaseError),
			want: CodeDatabaseError,
		},
		{
			name: "standard error",
			err:  errors.New("standard error"),
			want: CodeUnknown,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetCode(tt.err); got != tt.want {
				t.Errorf("GetCode() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetSeverity(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want Severity
	}{
		{
			name: "mDW error",
			err:  New("test").WithSeverity(SeverityCritical),
			want: SeverityCritical,
		},
		{
			name: "standard error",
			err:  errors.New("standard error"),
			want: SeverityMedium,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetSeverity(tt.err); got != tt.want {
				t.Errorf("GetSeverity() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestString(t *testing.T) {
	err := New("test error").
		WithCode(CodeDatabaseError).
		WithSeverity(SeverityHigh).
		WithContext("user-service").
		WithOperation("CreateUser").
		WithUserID("user123").
		WithRequestID("req-456").
		WithDetail("host", "localhost")
	
	str := err.String()
	
	// Check that all information is included
	expectedParts := []string{
		"Error: test error",
		"Code: DATABASE_ERROR",
		"Severity: high",
		"Context: user-service",
		"Operation: CreateUser",
		"UserID: user123",
		"RequestID: req-456",
		"Details: {host=localhost}",
	}
	
	for _, part := range expectedParts {
		if !strings.Contains(str, part) {
			t.Errorf("String() should contain %q, got:\n%s", part, str)
		}
	}
}

func TestMarshalJSON(t *testing.T) {
	err := New("test error").
		WithCode(CodeDatabaseError).
		WithSeverity(SeverityHigh).
		WithContext("user-service").
		WithDetail("host", "localhost")
	
	data, jsonErr := json.Marshal(err)
	if jsonErr != nil {
		t.Fatalf("json.Marshal() error = %v", jsonErr)
	}
	
	var result map[string]interface{}
	if jsonErr := json.Unmarshal(data, &result); jsonErr != nil {
		t.Fatalf("json.Unmarshal() error = %v", jsonErr)
	}
	
	// Check required fields
	if result["message"] != "test error" {
		t.Errorf("JSON message = %v, want \"test error\"", result["message"])
	}
	
	if result["code"] != "DATABASE_ERROR" {
		t.Errorf("JSON code = %v, want \"DATABASE_ERROR\"", result["code"])
	}
	
	if result["severity"] != "high" {
		t.Errorf("JSON severity = %v, want \"high\"", result["severity"])
	}
	
	if result["context"] != "user-service" {
		t.Errorf("JSON context = %v, want \"user-service\"", result["context"])
	}
	
	// Check details
	details, ok := result["details"].(map[string]interface{})
	if !ok {
		t.Fatal("JSON details should be a map")
	}
	
	if details["host"] != "localhost" {
		t.Errorf("JSON details.host = %v, want \"localhost\"", details["host"])
	}
}

func TestStackTrace(t *testing.T) {
	err := New("test error")
	
	stackTrace := err.StackTrace()
	if len(stackTrace) == 0 {
		t.Error("StackTrace() should not be empty")
	}
	
	// Check that the first frame contains this test function
	if !strings.Contains(stackTrace[0].Function, "TestStackTrace") {
		t.Errorf("First stack frame should contain TestStackTrace, got %s", stackTrace[0].Function)
	}
	
	if stackTrace[0].Line == 0 {
		t.Error("Stack frame line should not be 0")
	}
	
	if stackTrace[0].File == "" {
		t.Error("Stack frame file should not be empty")
	}
}

// Benchmark tests
func BenchmarkNew(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = New("benchmark error")
	}
}

func BenchmarkWrapStandardError(b *testing.B) {
	stdErr := errors.New("standard error")
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		_ = Wrap(stdErr, "wrapped error")
	}
}

func BenchmarkWrapmDWError(b *testing.B) {
	mdwErr := New("original error").WithCode(CodeDatabaseError)
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		_ = Wrap(mdwErr, "wrapped error")
	}
}

func BenchmarkWithDetails(b *testing.B) {
	details := map[string]interface{}{
		"key1": "value1",
		"key2": 42,
		"key3": true,
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = New("benchmark error").WithDetails(details)
	}
}

func BenchmarkMarshalJSON(b *testing.B) {
	err := New("benchmark error").
		WithCode(CodeDatabaseError).
		WithSeverity(SeverityHigh).
		WithContext("benchmark").
		WithDetail("iteration", 1)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = json.Marshal(err)
	}
}