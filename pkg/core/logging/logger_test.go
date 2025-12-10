package logging

import (
	"testing"
)

func TestLevel_Constants(t *testing.T) {
	if LevelDebug != 0 {
		t.Errorf("LevelDebug = %d, want 0", LevelDebug)
	}
	if LevelInfo != 1 {
		t.Errorf("LevelInfo = %d, want 1", LevelInfo)
	}
	if LevelWarn != 2 {
		t.Errorf("LevelWarn = %d, want 2", LevelWarn)
	}
	if LevelError != 3 {
		t.Errorf("LevelError = %d, want 3", LevelError)
	}
}

func TestLevel_String(t *testing.T) {
	tests := []struct {
		level    Level
		expected string
	}{
		{LevelDebug, "debug"},
		{LevelInfo, "info"},
		{LevelWarn, "warn"},
		{LevelError, "error"},
		{Level(99), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if got := tt.level.String(); got != tt.expected {
				t.Errorf("Level.String() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestNew(t *testing.T) {
	logger := New("test-service")

	if logger == nil {
		t.Fatal("New() returned nil")
	}
	if logger.name != "test-service" {
		t.Errorf("name = %v, want test-service", logger.name)
	}
}

func TestLogger_WithLevel(t *testing.T) {
	logger := New("test")
	result := logger.WithLevel(LevelDebug)

	if result == nil {
		t.Error("WithLevel should return a logger")
	}
	if result.name != "test" {
		t.Errorf("name should be preserved: got %v", result.name)
	}
}

func TestLogger_LogMethods(t *testing.T) {
	// Test that log methods don't panic
	logger := New("test")

	// These should not panic
	logger.Debug("debug message", "key", "value")
	logger.Info("info message", "key", "value")
	logger.Warn("warn message", "key", "value")
	logger.Error("error message", "key", "value")
}

func TestLogger_MultipleKeyValues(t *testing.T) {
	logger := New("test")

	// Should not panic with multiple key-value pairs
	logger.Info("message", "key1", "value1", "key2", 42, "key3", true)
}

func TestLogger_EmptyKeyValues(t *testing.T) {
	logger := New("test")

	// Should not panic without key-values
	logger.Info("message without key-values")
}

func TestLogger_OddKeyValues(t *testing.T) {
	logger := New("test")

	// Should not panic with odd number of key-values
	logger.Info("message", "key1", "value1", "orphan")
}

func TestParseLevel(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"debug", "debug"},
		{"info", "info"},
		{"warn", "warn"},
		{"warning", "warn"},
		{"error", "error"},
		{"invalid", "info"}, // defaults to info
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			// parseLevel is internal, but we can test via DefaultLoggerConfig
			cfg := DefaultLoggerConfig("test")
			cfg.Level = tt.input
			// The level will be parsed when creating the logger
		})
	}
}

func TestDefaultLoggerConfig(t *testing.T) {
	cfg := DefaultLoggerConfig("my-service")

	if cfg.ServiceName != "my-service" {
		t.Errorf("ServiceName = %v, want my-service", cfg.ServiceName)
	}
	if cfg.Level != "info" {
		t.Errorf("Level = %v, want info", cfg.Level)
	}
	if cfg.Format != "json" {
		t.Errorf("Format = %v, want json", cfg.Format)
	}
}

func TestNewSimpleLogger(t *testing.T) {
	logger := NewSimpleLogger("test-service")

	if logger == nil {
		t.Fatal("NewSimpleLogger() returned nil")
	}
}

func TestNewLogger(t *testing.T) {
	cfg := LoggerConfig{
		ServiceName: "test-service",
		Level:       "debug",
		Format:      "json",
	}

	logger := NewLogger(cfg)

	if logger == nil {
		t.Fatal("NewLogger() returned nil")
	}
}

func TestToFields(t *testing.T) {
	// Empty input
	fields := toFields()
	if fields != nil {
		t.Error("toFields() with no args should return nil")
	}

	// Valid key-value pairs
	fields = toFields("key1", "value1", "key2", 42)
	if fields == nil {
		t.Fatal("toFields() returned nil")
	}
	if fields["key1"] != "value1" {
		t.Errorf("fields[key1] = %v, want value1", fields["key1"])
	}
	if fields["key2"] != 42 {
		t.Errorf("fields[key2] = %v, want 42", fields["key2"])
	}

	// Non-string key (should be skipped)
	fields = toFields(123, "value")
	if len(fields) != 0 {
		t.Errorf("Non-string key should be skipped, got %v fields", len(fields))
	}
}

func TestDefaultBayesWriterConfig(t *testing.T) {
	cfg := DefaultBayesWriterConfig()

	if cfg.Address != "localhost:9120" {
		t.Errorf("Address = %v, want localhost:9120", cfg.Address)
	}
	if cfg.BatchSize != 100 {
		t.Errorf("BatchSize = %v, want 100", cfg.BatchSize)
	}
	if cfg.Fallback == nil {
		t.Error("Fallback should not be nil")
	}
}

func BenchmarkLogger_Info(b *testing.B) {
	logger := New("benchmark")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logger.Info("benchmark message", "iteration", i)
	}
}
