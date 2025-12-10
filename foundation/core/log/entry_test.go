// File: entry_test.go
// Title: Log Entry Tests
// Description: Tests for log entry structure and field manipulation functions.
// Author: msto63 with Claude Sonnet 4.0
// Version: v0.1.0
// Created: 2025-01-24
// Modified: 2025-01-24
//
// Change History:
// - 2025-01-24 v0.1.0: Initial implementation with comprehensive entry tests

package log

import (
	"errors"
	"testing"
	"time"
)

func TestNewEntry(t *testing.T) {
	level := LevelInfo
	message := "test message"
	
	entry := NewEntry(level, message)
	
	if entry.Level != level {
		t.Errorf("NewEntry() level = %v, want %v", entry.Level, level)
	}
	
	if entry.Message != message {
		t.Errorf("NewEntry() message = %v, want %v", entry.Message, message)
	}
	
	if entry.Timestamp.IsZero() {
		t.Error("NewEntry() timestamp should not be zero")
	}
	
	if entry.Fields == nil {
		t.Error("NewEntry() fields should be initialized")
	}
}

func TestFieldHelpers(t *testing.T) {
	tests := []struct {
		name     string
		field    Fields
		key      string
		expected interface{}
	}{
		{"Field", Field("test", "value"), "test", "value"},
		{"Int", Int("count", 42), "count", 42},
		{"Int64", Int64("id", int64(123)), "id", int64(123)},
		{"Float64", Float64("rate", 3.14), "rate", 3.14},
		{"String", String("name", "test"), "name", "test"},
		{"Bool", Bool("enabled", true), "enabled", true},
		{"Any", Any("data", "any_value"), "data", "any_value"},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if len(tt.field) != 1 {
				t.Errorf("Field helper should return exactly one field, got %d", len(tt.field))
			}
			
			if value, exists := tt.field[tt.key]; !exists {
				t.Errorf("Field helper should contain key %s", tt.key)
			} else if value != tt.expected {
				t.Errorf("Field helper value = %v, want %v", value, tt.expected)
			}
		})
	}
}

func TestErrField(t *testing.T) {
	err := errors.New("test error")
	field := Err(err)
	
	if len(field) != 1 {
		t.Errorf("Err() should return exactly one field, got %d", len(field))
	}
	
	if value, exists := field["error"]; !exists {
		t.Error("Err() should contain 'error' key")
	} else if value != err {
		t.Errorf("Err() value = %v, want %v", value, err)
	}
}

func TestDurationField(t *testing.T) {
	duration := time.Second
	field := Duration("elapsed", duration)
	
	if len(field) != 1 {
		t.Errorf("Duration() should return exactly one field, got %d", len(field))
	}
	
	if value, exists := field["elapsed"]; !exists {
		t.Error("Duration() should contain 'elapsed' key")
	} else if value != duration {
		t.Errorf("Duration() value = %v, want %v", value, duration)
	}
}

func TestTimeField(t *testing.T) {
	now := time.Now()
	field := Time("created", now)
	
	if len(field) != 1 {
		t.Errorf("Time() should return exactly one field, got %d", len(field))
	}
	
	if value, exists := field["created"]; !exists {
		t.Error("Time() should contain 'created' key")
	} else if value != now {
		t.Errorf("Time() value = %v, want %v", value, now)
	}
}

func TestFieldsMerge(t *testing.T) {
	fields1 := Fields{"key1": "value1", "key2": "value2"}
	fields2 := Fields{"key2": "overwrite", "key3": "value3"}
	
	merged := fields1.Merge(fields2)
	
	if len(merged) != 3 {
		t.Errorf("Merge() should return 3 fields, got %d", len(merged))
	}
	
	if merged["key1"] != "value1" {
		t.Errorf("Merge() key1 = %v, want value1", merged["key1"])
	}
	
	if merged["key2"] != "overwrite" {
		t.Errorf("Merge() key2 = %v, want overwrite", merged["key2"])
	}
	
	if merged["key3"] != "value3" {
		t.Errorf("Merge() key3 = %v, want value3", merged["key3"])
	}
	
	// Original fields should not be modified
	if fields1["key2"] != "value2" {
		t.Error("Merge() should not modify original fields")
	}
}

func TestFieldsWith(t *testing.T) {
	fields := Fields{"key1": "value1"}
	
	result := fields.With("key2", "value2")
	
	if len(result) != 2 {
		t.Errorf("With() should return 2 fields, got %d", len(result))
	}
	
	if result["key1"] != "value1" {
		t.Errorf("With() key1 = %v, want value1", result["key1"])
	}
	
	if result["key2"] != "value2" {
		t.Errorf("With() key2 = %v, want value2", result["key2"])
	}
}

func TestFieldsWithNil(t *testing.T) {
	var fields Fields
	
	result := fields.With("key", "value")
	
	if len(result) != 1 {
		t.Errorf("With() on nil should return 1 field, got %d", len(result))
	}
	
	if result["key"] != "value" {
		t.Errorf("With() key = %v, want value", result["key"])
	}
}

func TestFieldsClone(t *testing.T) {
	original := Fields{"key1": "value1", "key2": "value2"}
	
	cloned := original.Clone()
	
	if len(cloned) != len(original) {
		t.Errorf("Clone() length = %d, want %d", len(cloned), len(original))
	}
	
	for k, v := range original {
		if cloned[k] != v {
			t.Errorf("Clone() %s = %v, want %v", k, cloned[k], v)
		}
	}
	
	// Modify clone to ensure independence
	cloned["key1"] = "modified"
	if original["key1"] == "modified" {
		t.Error("Clone() should create independent copy")
	}
}

func TestFieldsCloneNil(t *testing.T) {
	var fields Fields
	cloned := fields.Clone()
	
	if cloned != nil {
		t.Error("Clone() of nil should return nil")
	}
}

func TestEntryWithFields(t *testing.T) {
	entry := NewEntry(LevelInfo, "test")
	fields := Fields{"key1": "value1", "key2": "value2"}
	
	result := entry.WithFields(fields)
	
	if result != entry {
		t.Error("WithFields() should return the same entry instance")
	}
	
	if len(entry.Fields) != 2 {
		t.Errorf("WithFields() should add 2 fields, got %d", len(entry.Fields))
	}
	
	for k, v := range fields {
		if entry.Fields[k] != v {
			t.Errorf("WithFields() %s = %v, want %v", k, entry.Fields[k], v)
		}
	}
}

func TestEntryWithField(t *testing.T) {
	entry := NewEntry(LevelInfo, "test")
	
	result := entry.WithField("key", "value")
	
	if result != entry {
		t.Error("WithField() should return the same entry instance")
	}
	
	if entry.Fields["key"] != "value" {
		t.Errorf("WithField() key = %v, want value", entry.Fields["key"])
	}
}

func TestEntryWithError(t *testing.T) {
	entry := NewEntry(LevelInfo, "test")
	err := errors.New("test error")
	
	result := entry.WithError(err)
	
	if result != entry {
		t.Error("WithError() should return the same entry instance")
	}
	
	if entry.Error != err {
		t.Errorf("WithError() error = %v, want %v", entry.Error, err)
	}
}

func TestEntryWithDuration(t *testing.T) {
	entry := NewEntry(LevelInfo, "test")
	duration := time.Second
	
	result := entry.WithDuration(duration)
	
	if result != entry {
		t.Error("WithDuration() should return the same entry instance")
	}
	
	if entry.Duration != duration {
		t.Errorf("WithDuration() duration = %v, want %v", entry.Duration, duration)
	}
}

func TestEntryWithRequestID(t *testing.T) {
	entry := NewEntry(LevelInfo, "test")
	requestID := "req-123"
	
	result := entry.WithRequestID(requestID)
	
	if result != entry {
		t.Error("WithRequestID() should return the same entry instance")
	}
	
	if entry.RequestID != requestID {
		t.Errorf("WithRequestID() requestID = %v, want %v", entry.RequestID, requestID)
	}
}

func TestEntryWithUserID(t *testing.T) {
	entry := NewEntry(LevelInfo, "test")
	userID := "user-456"
	
	result := entry.WithUserID(userID)
	
	if result != entry {
		t.Error("WithUserID() should return the same entry instance")
	}
	
	if entry.UserID != userID {
		t.Errorf("WithUserID() userID = %v, want %v", entry.UserID, userID)
	}
}

func TestEntryWithCorrelationID(t *testing.T) {
	entry := NewEntry(LevelInfo, "test")
	correlationID := "corr-789"
	
	result := entry.WithCorrelationID(correlationID)
	
	if result != entry {
		t.Error("WithCorrelationID() should return the same entry instance")
	}
	
	if entry.CorrelationID != correlationID {
		t.Errorf("WithCorrelationID() correlationID = %v, want %v", entry.CorrelationID, correlationID)
	}
}

func TestEntryWithLogger(t *testing.T) {
	entry := NewEntry(LevelInfo, "test")
	logger := "test-logger"
	
	result := entry.WithLogger(logger)
	
	if result != entry {
		t.Error("WithLogger() should return the same entry instance")
	}
	
	if entry.Logger != logger {
		t.Errorf("WithLogger() logger = %v, want %v", entry.Logger, logger)
	}
}

func TestEntryWithCaller(t *testing.T) {
	entry := NewEntry(LevelInfo, "test")
	function := "TestFunction"
	file := "test.go"
	line := 42
	
	result := entry.WithCaller(function, file, line)
	
	if result != entry {
		t.Error("WithCaller() should return the same entry instance")
	}
	
	if entry.Caller == nil {
		t.Fatal("WithCaller() should set caller info")
	}
	
	if entry.Caller.Function != function {
		t.Errorf("WithCaller() function = %v, want %v", entry.Caller.Function, function)
	}
	
	if entry.Caller.File != file {
		t.Errorf("WithCaller() file = %v, want %v", entry.Caller.File, file)
	}
	
	if entry.Caller.Line != line {
		t.Errorf("WithCaller() line = %v, want %v", entry.Caller.Line, line)
	}
}

func TestEntryClone(t *testing.T) {
	original := NewEntry(LevelInfo, "test message")
	original.Logger = "test-logger"
	original.RequestID = "req-123"
	original.UserID = "user-456"
	original.CorrelationID = "corr-789"
	original.Fields = Fields{"key": "value"}
	original.Error = errors.New("test error")
	original.Duration = time.Second
	original.WithCaller("TestFunction", "test.go", 42)
	
	cloned := original.Clone()
	
	if cloned == original {
		t.Error("Clone() should return a different instance")
	}
	
	if cloned.Level != original.Level {
		t.Errorf("Clone() level = %v, want %v", cloned.Level, original.Level)
	}
	
	if cloned.Message != original.Message {
		t.Errorf("Clone() message = %v, want %v", cloned.Message, original.Message)
	}
	
	if cloned.Logger != original.Logger {
		t.Errorf("Clone() logger = %v, want %v", cloned.Logger, original.Logger)
	}
	
	if cloned.RequestID != original.RequestID {
		t.Errorf("Clone() requestID = %v, want %v", cloned.RequestID, original.RequestID)
	}
	
	if cloned.Fields["key"] != "value" {
		t.Error("Clone() should copy fields")
	}
	
	// Ensure fields are independent
	cloned.Fields["key"] = "modified"
	if original.Fields["key"] == "modified" {
		t.Error("Clone() should create independent fields")
	}
	
	// Check caller info
	if cloned.Caller == nil {
		t.Error("Clone() should copy caller info")
	} else if cloned.Caller == original.Caller {
		t.Error("Clone() should create independent caller info")
	}
}

func TestEntryCloneNil(t *testing.T) {
	var entry *Entry
	cloned := entry.Clone()
	
	if cloned != nil {
		t.Error("Clone() of nil should return nil")
	}
}

// Benchmark tests
func BenchmarkNewEntry(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = NewEntry(LevelInfo, "test message")
	}
}

func BenchmarkFieldsMerge(b *testing.B) {
	fields1 := Fields{"key1": "value1", "key2": "value2"}
	fields2 := Fields{"key3": "value3", "key4": "value4"}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = fields1.Merge(fields2)
	}
}

func BenchmarkEntryWithFields(b *testing.B) {
	entry := NewEntry(LevelInfo, "test message")
	fields := Fields{"key1": "value1", "key2": "value2"}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = entry.WithFields(fields)
	}
}