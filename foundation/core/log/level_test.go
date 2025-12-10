// File: level_test.go
// Title: Log Level Tests
// Description: Tests for log level functionality including string representation,
//              parsing, priority, and filtering logic.
// Author: msto63 with Claude Sonnet 4.0
// Version: v0.1.0
// Created: 2025-01-24
// Modified: 2025-01-24
//
// Change History:
// - 2025-01-24 v0.1.0: Initial implementation with comprehensive level tests

package log

import (
	"testing"
)

func TestLevelString(t *testing.T) {
	tests := []struct {
		level Level
		want  string
	}{
		{LevelTrace, "trace"},
		{LevelDebug, "debug"},
		{LevelInfo, "info"},
		{LevelWarn, "warn"},
		{LevelError, "error"},
		{LevelFatal, "fatal"},
		{LevelAudit, "audit"},
		{Level(999), "unknown"},
	}
	
	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := tt.level.String(); got != tt.want {
				t.Errorf("Level.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestLevelShortString(t *testing.T) {
	tests := []struct {
		level Level
		want  string
	}{
		{LevelTrace, "TRC"},
		{LevelDebug, "DBG"},
		{LevelInfo, "INF"},
		{LevelWarn, "WRN"},
		{LevelError, "ERR"},
		{LevelFatal, "FTL"},
		{LevelAudit, "AUD"},
		{Level(999), "???"},
	}
	
	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := tt.level.ShortString(); got != tt.want {
				t.Errorf("Level.ShortString() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestLevelColor(t *testing.T) {
	tests := []struct {
		level Level
		want  string
	}{
		{LevelTrace, "\033[37m"},
		{LevelDebug, "\033[36m"},
		{LevelInfo, "\033[32m"},
		{LevelWarn, "\033[33m"},
		{LevelError, "\033[31m"},
		{LevelFatal, "\033[35m"},
		{LevelAudit, "\033[34m"},
		{Level(999), "\033[0m"},
	}
	
	for _, tt := range tests {
		t.Run(tt.level.String(), func(t *testing.T) {
			if got := tt.level.Color(); got != tt.want {
				t.Errorf("Level.Color() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestLevelPriority(t *testing.T) {
	tests := []struct {
		level Level
		want  int
	}{
		{LevelTrace, 0},
		{LevelDebug, 1},
		{LevelInfo, 2},
		{LevelWarn, 3},
		{LevelError, 4},
		{LevelFatal, 5},
		{LevelAudit, 6},
	}
	
	for _, tt := range tests {
		t.Run(tt.level.String(), func(t *testing.T) {
			if got := tt.level.Priority(); got != tt.want {
				t.Errorf("Level.Priority() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestLevelShouldLog(t *testing.T) {
	tests := []struct {
		name     string
		level    Level
		minLevel Level
		want     bool
	}{
		{"trace vs info", LevelTrace, LevelInfo, false},
		{"debug vs info", LevelDebug, LevelInfo, false},
		{"info vs info", LevelInfo, LevelInfo, true},
		{"warn vs info", LevelWarn, LevelInfo, true},
		{"error vs info", LevelError, LevelInfo, true},
		{"audit vs error", LevelAudit, LevelError, true}, // Audit always logs
		{"audit vs fatal", LevelAudit, LevelFatal, true}, // Audit always logs
		{"fatal vs trace", LevelFatal, LevelTrace, true},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.level.ShouldLog(tt.minLevel); got != tt.want {
				t.Errorf("Level.ShouldLog() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestLevelIsEnabled(t *testing.T) {
	tests := []struct {
		name     string
		level    Level
		minLevel Level
		want     bool
	}{
		{"debug enabled for debug", LevelDebug, LevelDebug, true},
		{"debug disabled for info", LevelDebug, LevelInfo, false},
		{"error enabled for debug", LevelError, LevelDebug, true},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.level.IsEnabled(tt.minLevel); got != tt.want {
				t.Errorf("Level.IsEnabled() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseLevel(t *testing.T) {
	tests := []struct {
		input   string
		want    Level
		wantErr bool
	}{
		{"trace", LevelTrace, false},
		{"TRC", LevelTrace, false},
		{"debug", LevelDebug, false},
		{"DBG", LevelDebug, false},
		{"info", LevelInfo, false},
		{"INF", LevelInfo, false},
		{"information", LevelInfo, false},
		{"warn", LevelWarn, false},
		{"WRN", LevelWarn, false},
		{"warning", LevelWarn, false},
		{"error", LevelError, false},
		{"ERR", LevelError, false},
		{"fatal", LevelFatal, false},
		{"FTL", LevelFatal, false},
		{"audit", LevelAudit, false},
		{"AUD", LevelAudit, false},
		{"  INFO  ", LevelInfo, false}, // Test trimming
		{"INVALID", LevelInfo, true},   // Returns default with error
		{"", LevelInfo, true},          // Empty string
	}
	
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := ParseLevel(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseLevel() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("ParseLevel() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseError(t *testing.T) {
	err := &ParseError{
		Input: "invalid",
		Type:  "level",
	}
	
	want := "invalid level: invalid"
	if got := err.Error(); got != want {
		t.Errorf("ParseError.Error() = %v, want %v", got, want)
	}
}

func TestAllLevels(t *testing.T) {
	levels := AllLevels()
	
	expectedCount := 7
	if len(levels) != expectedCount {
		t.Errorf("AllLevels() returned %d levels, want %d", len(levels), expectedCount)
	}
	
	// Check that all expected levels are present
	expectedLevels := []Level{
		LevelTrace, LevelDebug, LevelInfo, LevelWarn,
		LevelError, LevelFatal, LevelAudit,
	}
	
	for i, expected := range expectedLevels {
		if i >= len(levels) || levels[i] != expected {
			t.Errorf("AllLevels()[%d] = %v, want %v", i, levels[i], expected)
		}
	}
}

func TestDefaultLevel(t *testing.T) {
	if got := DefaultLevel(); got != LevelInfo {
		t.Errorf("DefaultLevel() = %v, want %v", got, LevelInfo)
	}
}

func TestDevelopmentLevel(t *testing.T) {
	if got := DevelopmentLevel(); got != LevelDebug {
		t.Errorf("DevelopmentLevel() = %v, want %v", got, LevelDebug)
	}
}

func TestLevelOrdering(t *testing.T) {
	// Test that levels are properly ordered by priority
	levels := []Level{LevelTrace, LevelDebug, LevelInfo, LevelWarn, LevelError, LevelFatal}
	
	for i := 0; i < len(levels)-1; i++ {
		if levels[i].Priority() >= levels[i+1].Priority() {
			t.Errorf("Level %v should have lower priority than %v", levels[i], levels[i+1])
		}
	}
}

func TestAuditLevelSpecialBehavior(t *testing.T) {
	// Audit should always log regardless of minimum level
	minLevels := []Level{LevelTrace, LevelDebug, LevelInfo, LevelWarn, LevelError, LevelFatal}
	
	for _, minLevel := range minLevels {
		if !LevelAudit.ShouldLog(minLevel) {
			t.Errorf("LevelAudit should always log, but ShouldLog(%v) returned false", minLevel)
		}
	}
}

// Benchmark tests
func BenchmarkLevelString(b *testing.B) {
	level := LevelInfo
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		_ = level.String()
	}
}

func BenchmarkLevelShouldLog(b *testing.B) {
	level := LevelError
	minLevel := LevelInfo
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		_ = level.ShouldLog(minLevel)
	}
}

func BenchmarkParseLevel(b *testing.B) {
	input := "info"
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		_, _ = ParseLevel(input)
	}
}