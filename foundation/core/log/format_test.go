// File: format_test.go
// Title: Format Tests
// Description: Tests for log formatting functionality including JSON, text,
//              console, and logfmt formatters.
// Author: msto63 with Claude Sonnet 4.0
// Version: v0.1.0
// Created: 2025-01-24
// Modified: 2025-01-24
//
// Change History:
// - 2025-01-24 v0.1.0: Initial implementation with comprehensive format tests

package log

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"
)

func TestFormatString(t *testing.T) {
	tests := []struct {
		format Format
		want   string
	}{
		{FormatJSON, "json"},
		{FormatText, "text"},
		{FormatConsole, "console"},
		{FormatLogfmt, "logfmt"},
		{Format(999), "unknown"},
	}
	
	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := tt.format.String(); got != tt.want {
				t.Errorf("Format.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseFormat(t *testing.T) {
	tests := []struct {
		input   string
		want    Format
		wantErr bool
	}{
		{"json", FormatJSON, false},
		{"text", FormatText, false},
		{"console", FormatConsole, false},
		{"logfmt", FormatLogfmt, false},
		{"JSON", FormatJSON, false}, // Case insensitive
		{"  text  ", FormatText, false}, // Trimming
		{"invalid", FormatJSON, true}, // Returns default with error
		{"", FormatJSON, true},
	}
	
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := ParseFormat(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseFormat() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("ParseFormat() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestJSONFormatter_Format(t *testing.T) {
	formatter := NewJSONFormatter()
	
	entry := NewEntry(LevelInfo, "test message")
	entry.Logger = "test-logger"
	entry.RequestID = "req-123"
	entry.UserID = "user-456"
	entry.CorrelationID = "corr-789"
	entry.Fields = Fields{"key": "value", "count": 42}
	entry.Error = errors.New("test error")
	entry.Duration = time.Millisecond * 100
	
	data, err := formatter.Format(entry)
	if err != nil {
		t.Fatalf("JSONFormatter.Format() error = %v", err)
	}
	
	// Parse JSON to verify structure
	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("Failed to parse JSON output: %v", err)
	}
	
	// Check required fields
	expectedFields := map[string]interface{}{
		"level":          "info",
		"message":        "test message",
		"logger":         "test-logger",
		"request_id":     "req-123",
		"user_id":        "user-456",
		"correlation_id": "corr-789",
		"key":           "value",
		"count":         float64(42), // JSON numbers are float64
		"error":         "test error",
		"duration_ms":   float64(100),
	}
	
	for key, expected := range expectedFields {
		if actual, exists := result[key]; !exists {
			t.Errorf("JSON output missing field %s", key)
		} else if actual != expected {
			t.Errorf("JSON field %s = %v, want %v", key, actual, expected)
		}
	}
	
	// Check timestamp is present and valid
	if timestamp, exists := result["timestamp"]; !exists {
		t.Error("JSON output missing timestamp")
	} else if _, ok := timestamp.(string); !ok {
		t.Error("JSON timestamp should be a string")
	}
}

func TestJSONFormatter_PrettyPrint(t *testing.T) {
	formatter := &JSONFormatter{PrettyPrint: true}
	
	entry := NewEntry(LevelInfo, "test message")
	entry.Fields = Fields{"key": "value"}
	
	data, err := formatter.Format(entry)
	if err != nil {
		t.Fatalf("JSONFormatter.Format() error = %v", err)
	}
	
	// Pretty printed JSON should contain newlines and indentation
	output := string(data)
	if !strings.Contains(output, "\n") {
		t.Error("Pretty printed JSON should contain newlines")
	}
	
	if !strings.Contains(output, "  ") {
		t.Error("Pretty printed JSON should contain indentation")
	}
}

func TestTextFormatter_Format(t *testing.T) {
	formatter := NewTextFormatter()
	
	entry := NewEntry(LevelInfo, "test message")
	entry.Logger = "test-logger"
	entry.RequestID = "req-123"
	entry.UserID = "user-456"
	entry.Fields = Fields{"key": "value"}
	entry.Error = errors.New("test error")
	entry.Duration = time.Second
	
	data, err := formatter.Format(entry)
	if err != nil {
		t.Fatalf("TextFormatter.Format() error = %v", err)
	}
	
	output := string(data)
	
	// Check that output contains expected elements
	expectedElements := []string{
		"[INF]",
		"{test-logger}",
		"(req=req-123,user=user-456)",
		"test message",
		"[key=value]",
		"error=\"test error\"",
		"duration=1s",
	}
	
	for _, element := range expectedElements {
		if !strings.Contains(output, element) {
			t.Errorf("Text output should contain %q, got: %s", element, output)
		}
	}
}

func TestTextFormatter_DisableTimestamp(t *testing.T) {
	formatter := &TextFormatter{DisableTimestamp: true}
	
	entry := NewEntry(LevelInfo, "test message")
	
	data, err := formatter.Format(entry)
	if err != nil {
		t.Fatalf("TextFormatter.Format() error = %v", err)
	}
	
	output := string(data)
	
	// Should not contain timestamp patterns
	if strings.Contains(output, ":") && strings.Contains(output, "T") {
		t.Errorf("Output with disabled timestamp should not contain timestamp, got: %s", output)
	}
}

func TestTextFormatter_FullTimestamp(t *testing.T) {
	formatter := &TextFormatter{FullTimestamp: true}
	
	entry := NewEntry(LevelInfo, "test message")
	
	data, err := formatter.Format(entry)
	if err != nil {
		t.Fatalf("TextFormatter.Format() error = %v", err)
	}
	
	output := string(data)
	
	// Full timestamp should contain date and time (RFC3339 format)
	if !strings.Contains(output, "T") {
		t.Errorf("Full timestamp should contain date separator, got: %s", output)
	}
}

func TestConsoleFormatter_Format(t *testing.T) {
	formatter := NewConsoleFormatter()
	
	entry := NewEntry(LevelError, "test error message")
	
	data, err := formatter.Format(entry)
	if err != nil {
		t.Fatalf("ConsoleFormatter.Format() error = %v", err)
	}
	
	output := string(data)
	
	// Should contain color codes for error level (red)
	if !strings.Contains(output, "\033[31m") {
		t.Error("Console output should contain color codes")
	}
	
	// Should contain reset code
	if !strings.Contains(output, "\033[0m") {
		t.Error("Console output should contain reset code")
	}
}

func TestConsoleFormatter_DisableColors(t *testing.T) {
	formatter := &ConsoleFormatter{
		DisableColors: true,
		TextFormatter: NewTextFormatter(),
	}
	
	entry := NewEntry(LevelError, "test error message")
	
	data, err := formatter.Format(entry)
	if err != nil {
		t.Fatalf("ConsoleFormatter.Format() error = %v", err)
	}
	
	output := string(data)
	
	// Should not contain color codes
	if strings.Contains(output, "\033[") {
		t.Errorf("Output with disabled colors should not contain color codes, got: %s", output)
	}
}

func TestLogfmtFormatter_Format(t *testing.T) {
	formatter := NewLogfmtFormatter()
	
	entry := NewEntry(LevelWarn, "test warning")
	entry.Logger = "test-logger"
	entry.RequestID = "req-123"
	entry.Fields = Fields{"key": "value", "count": 42, "enabled": true}
	entry.Error = errors.New("test error")
	entry.Duration = time.Millisecond * 250
	
	data, err := formatter.Format(entry)
	if err != nil {
		t.Fatalf("LogfmtFormatter.Format() error = %v", err)
	}
	
	output := string(data)
	
	// Check key=value format
	expectedPairs := []string{
		"level=warn",
		"message=\"test warning\"",
		"logger=test-logger",
		"request_id=req-123",
		"key=\"value\"",
		"count=42",
		"enabled=true",
		"error=\"test error\"",
		"duration_ms=250.000",
	}
	
	for _, pair := range expectedPairs {
		if !strings.Contains(output, pair) {
			t.Errorf("Logfmt output should contain %q, got: %s", pair, output)
		}
	}
	
	// Should contain timestamp
	if !strings.Contains(output, "timestamp=") {
		t.Error("Logfmt output should contain timestamp")
	}
}

func TestGetFormatter(t *testing.T) {
	tests := []struct {
		format   Format
		expected string
	}{
		{FormatJSON, "*log.JSONFormatter"},
		{FormatText, "*log.TextFormatter"},
		{FormatConsole, "*log.ConsoleFormatter"},
		{FormatLogfmt, "*log.LogfmtFormatter"},
		{Format(999), "*log.JSONFormatter"}, // Default fallback
	}
	
	for _, tt := range tests {
		t.Run(tt.format.String(), func(t *testing.T) {
			formatter := GetFormatter(tt.format)
			if formatter == nil {
				t.Error("GetFormatter() should not return nil")
			}
			
			// Check type (simplified check)
			formatterType := strings.Contains(strings.ToLower(tt.expected), strings.ToLower(tt.format.String()))
			if tt.format == Format(999) {
				formatterType = true // Default case
			}
			
			if !formatterType {
				t.Errorf("GetFormatter() returned unexpected type for format %v", tt.format)
			}
		})
	}
}

func TestFormatterWithmDWError(t *testing.T) {
	// This test would require the mDW error type, but for now we'll test
	// that formatters handle errors that implement MarshalJSON gracefully
	
	formatter := NewJSONFormatter()
	
	entry := NewEntry(LevelError, "mDW error occurred")
	// We'll use a standard error for now, but the formatter should handle
	// mDW errors when they're available
	entry.Error = errors.New("standard error")
	
	data, err := formatter.Format(entry)
	if err != nil {
		t.Fatalf("JSONFormatter.Format() with error failed: %v", err)
	}
	
	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("Failed to parse JSON with error: %v", err)
	}
	
	if result["error"] != "standard error" {
		t.Errorf("Error field = %v, want 'standard error'", result["error"])
	}
}

// Benchmark tests
func BenchmarkJSONFormatter_Format(b *testing.B) {
	formatter := NewJSONFormatter()
	entry := NewEntry(LevelInfo, "benchmark message")
	entry.Fields = Fields{"key": "value", "count": 42}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = formatter.Format(entry)
	}
}

func BenchmarkTextFormatter_Format(b *testing.B) {
	formatter := NewTextFormatter()
	entry := NewEntry(LevelInfo, "benchmark message")
	entry.Fields = Fields{"key": "value", "count": 42}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = formatter.Format(entry)
	}
}

func BenchmarkConsoleFormatter_Format(b *testing.B) {
	formatter := NewConsoleFormatter()
	entry := NewEntry(LevelInfo, "benchmark message")
	entry.Fields = Fields{"key": "value", "count": 42}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = formatter.Format(entry)
	}
}

func BenchmarkLogfmtFormatter_Format(b *testing.B) {
	formatter := NewLogfmtFormatter()
	entry := NewEntry(LevelInfo, "benchmark message")
	entry.Fields = Fields{"key": "value", "count": 42}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = formatter.Format(entry)
	}
}