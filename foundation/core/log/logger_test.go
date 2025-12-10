// File: logger_test.go
// Title: Logger Tests
// Description: Tests for the main logger functionality including configuration,
//              context management, and integration with formatters.
// Author: msto63 with Claude Sonnet 4.0
// Version: v0.1.0
// Created: 2025-01-24
// Modified: 2025-01-24
//
// Change History:
// - 2025-01-24 v0.1.0: Initial implementation with comprehensive logger tests

package log

import (
	"bytes"
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

func TestNew(t *testing.T) {
	logger := New()
	
	if logger == nil {
		t.Fatal("New() should not return nil")
	}
	
	if logger.GetLevel() != DefaultLevel() {
		t.Errorf("New() level = %v, want %v", logger.GetLevel(), DefaultLevel())
	}
	
	if logger.contextFields == nil {
		t.Error("New() should initialize context fields")
	}
}

func TestNewWithConfig(t *testing.T) {
	var buf bytes.Buffer
	config := Config{
		Level:  LevelError,
		Format: FormatText,
		Output: &buf,
		Name:   "test-logger",
	}
	
	logger := NewWithConfig(config)
	
	if logger.GetLevel() != LevelError {
		t.Errorf("NewWithConfig() level = %v, want %v", logger.GetLevel(), LevelError)
	}
	
	if logger.name != "test-logger" {
		t.Errorf("NewWithConfig() name = %v, want test-logger", logger.name)
	}
	
	if logger.output != &buf {
		t.Error("NewWithConfig() should set custom output")
	}
}

func TestLoggerWithLevel(t *testing.T) {
	logger := New()
	newLogger := logger.WithLevel(LevelDebug)
	
	if newLogger == logger {
		t.Error("WithLevel() should return a new logger instance")
	}
	
	if newLogger.GetLevel() != LevelDebug {
		t.Errorf("WithLevel() level = %v, want %v", newLogger.GetLevel(), LevelDebug)
	}
	
	// Original logger should be unchanged
	if logger.GetLevel() != DefaultLevel() {
		t.Error("WithLevel() should not modify original logger")
	}
}

func TestLoggerWithFormat(t *testing.T) {
	logger := New()
	newLogger := logger.WithFormat(FormatText)
	
	if newLogger == logger {
		t.Error("WithFormat() should return a new logger instance")
	}
	
	// We can't directly test the formatter type, but we can test that it changes
	if newLogger.formatter == logger.formatter {
		t.Error("WithFormat() should change the formatter")
	}
}

func TestLoggerWithName(t *testing.T) {
	logger := New()
	newLogger := logger.WithName("test-logger")
	
	if newLogger == logger {
		t.Error("WithName() should return a new logger instance")
	}
	
	if newLogger.name != "test-logger" {
		t.Errorf("WithName() name = %v, want test-logger", newLogger.name)
	}
}

func TestLoggerWithField(t *testing.T) {
	logger := New()
	newLogger := logger.WithField("service", "user-api")
	
	if newLogger == logger {
		t.Error("WithField() should return a new logger instance")
	}
	
	if newLogger.contextFields["service"] != "user-api" {
		t.Error("WithField() should add context field")
	}
	
	// Original logger should be unchanged
	if _, exists := logger.contextFields["service"]; exists {
		t.Error("WithField() should not modify original logger")
	}
}

func TestLoggerWithFields(t *testing.T) {
	logger := New()
	fields := Fields{"service": "user-api", "version": "1.0"}
	newLogger := logger.WithFields(fields)
	
	if newLogger == logger {
		t.Error("WithFields() should return a new logger instance")
	}
	
	for k, v := range fields {
		if newLogger.contextFields[k] != v {
			t.Errorf("WithFields() should add field %s=%v", k, v)
		}
	}
}

func TestLoggerWithRequestID(t *testing.T) {
	logger := New()
	newLogger := logger.WithRequestID("req-123")
	
	if newLogger == logger {
		t.Error("WithRequestID() should return a new logger instance")
	}
	
	if newLogger.requestID != "req-123" {
		t.Errorf("WithRequestID() requestID = %v, want req-123", newLogger.requestID)
	}
}

func TestLoggerWithUserID(t *testing.T) {
	logger := New()
	newLogger := logger.WithUserID("user-456")
	
	if newLogger == logger {
		t.Error("WithUserID() should return a new logger instance")
	}
	
	if newLogger.userID != "user-456" {
		t.Errorf("WithUserID() userID = %v, want user-456", newLogger.userID)
	}
}

func TestLoggerWithCorrelationID(t *testing.T) {
	logger := New()
	newLogger := logger.WithCorrelationID("corr-789")
	
	if newLogger == logger {
		t.Error("WithCorrelationID() should return a new logger instance")
	}
	
	if newLogger.correlationID != "corr-789" {
		t.Errorf("WithCorrelationID() correlationID = %v, want corr-789", newLogger.correlationID)
	}
}

func TestLoggerWithCaller(t *testing.T) {
	logger := New()
	newLogger := logger.WithCaller(1)
	
	if newLogger == logger {
		t.Error("WithCaller() should return a new logger instance")
	}
	
	if !newLogger.enableCaller {
		t.Error("WithCaller() should enable caller information")
	}
	
	if newLogger.callerSkipFrames != 1 {
		t.Errorf("WithCaller() skip frames = %v, want 1", newLogger.callerSkipFrames)
	}
}

func TestLoggerLogLevels(t *testing.T) {
	var buf bytes.Buffer
	logger := New().WithOutput(&buf).WithFormat(FormatJSON).WithLevel(LevelTrace)
	
	tests := []struct {
		name   string
		logFn  func(string, ...Fields)
		level  Level
		msg    string
	}{
		{"Trace", logger.Trace, LevelTrace, "trace message"},
		{"Debug", logger.Debug, LevelDebug, "debug message"},
		{"Info", logger.Info, LevelInfo, "info message"},
		{"Warn", logger.Warn, LevelWarn, "warn message"},
		{"Error", logger.Error, LevelError, "error message"},
		{"Audit", logger.Audit, LevelAudit, "audit message"},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf.Reset()
			
			tt.logFn(tt.msg, Fields{"test": true})
			
			if buf.Len() == 0 {
				t.Errorf("%s() should write to output", tt.name)
				return
			}
			
			// Parse JSON output
			var result map[string]interface{}
			if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
				t.Fatalf("Failed to parse JSON output: %v", err)
			}
			
			if result["level"] != tt.level.String() {
				t.Errorf("%s() level = %v, want %v", tt.name, result["level"], tt.level.String())
			}
			
			if result["message"] != tt.msg {
				t.Errorf("%s() message = %v, want %v", tt.name, result["message"], tt.msg)
			}
			
			if result["test"] != true {
				t.Errorf("%s() should include provided fields", tt.name)
			}
		})
	}
}

func TestLoggerErrorWithErr(t *testing.T) {
	var buf bytes.Buffer
	logger := New().WithOutput(&buf).WithFormat(FormatJSON)
	
	err := errors.New("test error")
	logger.ErrorWithErr("operation failed", err)
	
	if buf.Len() == 0 {
		t.Error("ErrorWithErr() should write to output")
		return
	}
	
	var result map[string]interface{}
	if jsonErr := json.Unmarshal(buf.Bytes(), &result); jsonErr != nil {
		t.Fatalf("Failed to parse JSON output: %v", jsonErr)
	}
	
	if result["message"] != "operation failed" {
		t.Errorf("ErrorWithErr() message = %v, want 'operation failed'", result["message"])
	}
	
	if result["error"] != "test error" {
		t.Errorf("ErrorWithErr() error = %v, want 'test error'", result["error"])
	}
}

func TestLoggerWarnWithErr(t *testing.T) {
	var buf bytes.Buffer
	logger := New().WithOutput(&buf).WithFormat(FormatJSON)
	
	err := errors.New("test warning")
	logger.WarnWithErr("operation warning", err)
	
	if buf.Len() == 0 {
		t.Error("WarnWithErr() should write to output")
		return
	}
	
	var result map[string]interface{}
	if jsonErr := json.Unmarshal(buf.Bytes(), &result); jsonErr != nil {
		t.Fatalf("Failed to parse JSON output: %v", jsonErr)
	}
	
	if result["level"] != "warn" {
		t.Errorf("WarnWithErr() level = %v, want 'warn'", result["level"])
	}
}

func TestLoggerLogError(t *testing.T) {
	var buf bytes.Buffer
	logger := New().WithOutput(&buf).WithFormat(FormatJSON)
	
	// Test with nil error
	logger.LogError(nil)
	if buf.Len() != 0 {
		t.Error("LogError(nil) should not write to output")
	}
	
	// Test with standard error
	err := errors.New("standard error")
	logger.LogError(err)
	
	if buf.Len() == 0 {
		t.Error("LogError() should write to output")
		return
	}
	
	var result map[string]interface{}
	if jsonErr := json.Unmarshal(buf.Bytes(), &result); jsonErr != nil {
		t.Fatalf("Failed to parse JSON output: %v", jsonErr)
	}
	
	if result["message"] != "standard error" {
		t.Errorf("LogError() message = %v, want 'standard error'", result["message"])
	}
	
	if result["level"] != "error" {
		t.Errorf("LogError() level = %v, want 'error'", result["level"])
	}
}

func TestLoggerIsLevelEnabled(t *testing.T) {
	logger := New().WithLevel(LevelWarn)
	
	tests := []struct {
		level   Level
		enabled bool
	}{
		{LevelTrace, false},
		{LevelDebug, false},
		{LevelInfo, false},
		{LevelWarn, true},
		{LevelError, true},
		{LevelFatal, true},
		{LevelAudit, true}, // Audit is always enabled
	}
	
	for _, tt := range tests {
		t.Run(tt.level.String(), func(t *testing.T) {
			if got := logger.IsLevelEnabled(tt.level); got != tt.enabled {
				t.Errorf("IsLevelEnabled(%v) = %v, want %v", tt.level, got, tt.enabled)
			}
		})
	}
}

func TestLoggerSetLevel(t *testing.T) {
	logger := New()
	logger.SetLevel(LevelError)
	
	if logger.GetLevel() != LevelError {
		t.Errorf("SetLevel() level = %v, want %v", logger.GetLevel(), LevelError)
	}
}

func TestLoggerStartTimer(t *testing.T) {
	logger := New()
	timer := logger.StartTimer("test-operation")
	
	if timer == nil {
		t.Fatal("StartTimer() should not return nil")
	}
	
	if timer.operation != "test-operation" {
		t.Errorf("Timer operation = %v, want test-operation", timer.operation)
	}
	
	if timer.logger != logger {
		t.Error("Timer should reference the logger")
	}
}

func TestLoggerWithOutput(t *testing.T) {
	var buf bytes.Buffer
	logger := New()
	newLogger := logger.WithOutput(&buf)
	
	if newLogger == logger {
		t.Error("WithOutput() should return a new logger instance")
	}
	
	newLogger.Info("test message")
	
	if buf.Len() == 0 {
		t.Error("Logger with custom output should write to buffer")
	}
}

func TestLoggerLevelFiltering(t *testing.T) {
	var buf bytes.Buffer
	logger := New().WithOutput(&buf).WithLevel(LevelWarn).WithFormat(FormatText)
	
	// These should not be logged
	logger.Trace("trace message")
	logger.Debug("debug message")
	logger.Info("info message")
	
	// These should be logged
	logger.Warn("warn message")
	logger.Error("error message")
	logger.Audit("audit message") // Always logged
	
	output := buf.String()
	
	// Should not contain low-level messages
	if strings.Contains(output, "trace message") ||
		strings.Contains(output, "debug message") ||
		strings.Contains(output, "info message") {
		t.Error("Logger should filter out messages below minimum level")
	}
	
	// Should contain high-level messages
	if !strings.Contains(output, "warn message") ||
		!strings.Contains(output, "error message") ||
		!strings.Contains(output, "audit message") {
		t.Error("Logger should include messages at or above minimum level")
	}
}

func TestLoggerContextFields(t *testing.T) {
	var buf bytes.Buffer
	logger := New().
		WithOutput(&buf).
		WithFormat(FormatJSON).
		WithField("service", "test-service").
		WithRequestID("req-123").
		WithUserID("user-456")
	
	logger.Info("test message", Fields{"additional": "field"})
	
	var result map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("Failed to parse JSON output: %v", err)
	}
	
	// Check context fields are included
	if result["service"] != "test-service" {
		t.Error("Logger should include context fields")
	}
	
	if result["request_id"] != "req-123" {
		t.Error("Logger should include request ID")
	}
	
	if result["user_id"] != "user-456" {
		t.Error("Logger should include user ID")
	}
	
	// Check additional fields are included
	if result["additional"] != "field" {
		t.Error("Logger should include additional fields")
	}
}

func TestLoggerClone(t *testing.T) {
	original := New().
		WithLevel(LevelDebug).
		WithName("original").
		WithField("service", "test")
	
	// WithLevel should create a clone
	clone := original.WithLevel(LevelError)
	
	if clone == original {
		t.Error("Logger operations should create new instances")
	}
	
	// Changes to clone should not affect original
	if original.GetLevel() == LevelError {
		t.Error("Clone modifications should not affect original")
	}
	
	// Clone should have independent context fields
	clone = clone.WithField("version", "1.0")
	if _, exists := original.contextFields["version"]; exists {
		t.Error("Clone context fields should be independent")
	}
}

func TestGlobalLoggerFunctions(t *testing.T) {
	var buf bytes.Buffer
	
	// Set custom default logger
	originalDefault := GetDefault()
	defer SetDefault(originalDefault)
	
	testLogger := New().WithOutput(&buf).WithFormat(FormatJSON)
	SetDefault(testLogger)
	
	if GetDefault() != testLogger {
		t.Error("SetDefault() should set the default logger")
	}
	
	// Test global functions
	Info("test info message")
	
	if buf.Len() == 0 {
		t.Error("Global Info() should write to default logger")
	}
	
	var result map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("Failed to parse JSON output: %v", err)
	}
	
	if result["message"] != "test info message" {
		t.Error("Global function should log correct message")
	}
}

// Benchmark tests
func BenchmarkLoggerInfo(b *testing.B) {
	var buf bytes.Buffer
	logger := New().WithOutput(&buf)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logger.Info("benchmark message", Fields{"iteration": i})
	}
}

func BenchmarkLoggerWithFields(b *testing.B) {
	logger := New()
	fields := Fields{"service": "test", "version": "1.0"}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = logger.WithFields(fields)
	}
}

func BenchmarkLoggerLevelFiltering(b *testing.B) {
	var buf bytes.Buffer
	logger := New().WithOutput(&buf).WithLevel(LevelError)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logger.Debug("filtered debug message")
	}
}