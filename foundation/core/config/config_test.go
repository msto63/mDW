// File: config_test.go
// Title: Configuration Module Tests
// Description: Comprehensive tests for the config module covering TOML/YAML
//              parsing, environment variable injection, validation, and all
//              core configuration management functionality.
// Author: msto63 with Claude Sonnet 4.0
// Version: v0.1.0
// Created: 2025-01-25
// Modified: 2025-01-25
//
// Change History:
// - 2025-01-25 v0.1.0: Initial test implementation

package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLoad(t *testing.T) {
	// Create temporary directory for test files
	tempDir := t.TempDir()
	
	t.Run("load TOML config", func(t *testing.T) {
		configPath := filepath.Join(tempDir, "test.toml")
		configContent := `
[database]
host = "localhost"
port = 5432
ssl = true

[server]
timeout = "30s"
workers = 4
features = ["auth", "logging", "metrics"]
`
		if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
			t.Fatalf("Failed to write test config: %v", err)
		}

		cfg, err := Load(configPath)
		if err != nil {
			t.Fatalf("Failed to load config: %v", err)
		}

		// Test string values
		if host := cfg.GetString("database.host"); host != "localhost" {
			t.Errorf("Expected host 'localhost', got '%s'", host)
		}

		// Test integer values
		if port := cfg.GetInt("database.port"); port != 5432 {
			t.Errorf("Expected port 5432, got %d", port)
		}

		// Test boolean values
		if ssl := cfg.GetBool("database.ssl"); !ssl {
			t.Errorf("Expected ssl true, got %v", ssl)
		}

		// Test duration values
		if timeout := cfg.GetDuration("server.timeout"); timeout != 30*time.Second {
			t.Errorf("Expected timeout 30s, got %v", timeout)
		}

		// Test string slice values
		features := cfg.GetStringSlice("server.features")
		expectedFeatures := []string{"auth", "logging", "metrics"}
		if len(features) != len(expectedFeatures) {
			t.Errorf("Expected %d features, got %d", len(expectedFeatures), len(features))
		}
		for i, feature := range features {
			if feature != expectedFeatures[i] {
				t.Errorf("Expected feature '%s', got '%s'", expectedFeatures[i], feature)
			}
		}
	})

	t.Run("load YAML config", func(t *testing.T) {
		configPath := filepath.Join(tempDir, "test.yaml")
		configContent := `
database:
  host: localhost
  port: 5432
  ssl: true

server:
  timeout: 30s
  workers: 4
  features:
    - auth
    - logging
    - metrics
`
		if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
			t.Fatalf("Failed to write test config: %v", err)
		}

		cfg, err := Load(configPath)
		if err != nil {
			t.Fatalf("Failed to load config: %v", err)
		}

		// Test values same as TOML test
		if host := cfg.GetString("database.host"); host != "localhost" {
			t.Errorf("Expected host 'localhost', got '%s'", host)
		}

		if port := cfg.GetInt("database.port"); port != 5432 {
			t.Errorf("Expected port 5432, got %d", port)
		}
	})

	t.Run("file not found", func(t *testing.T) {
		_, err := Load("nonexistent.toml")
		if err == nil {
			t.Error("Expected error for nonexistent file")
		}
	})
}

func TestEnvironmentVariables(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "test.toml")
	configContent := `
[database]
host = "localhost"
port = 5432
`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	// Set environment variables
	os.Setenv("DATABASE_HOST", "production-db")
	os.Setenv("DATABASE_PORT", "3306")
	defer func() {
		os.Unsetenv("DATABASE_HOST")
		os.Unsetenv("DATABASE_PORT")
	}()

	cfg, err := LoadWithOptions(configPath, LoadOptions{
		EnvPrefix: "",
	})
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Environment variables should override config values
	if host := cfg.GetString("database.host"); host != "production-db" {
		t.Errorf("Expected host 'production-db' from env var, got '%s'", host)
	}

	if port := cfg.GetInt("database.port"); port != 3306 {
		t.Errorf("Expected port 3306 from env var, got %d", port)
	}
}

func TestDefaults(t *testing.T) {
	t.Run("with default values", func(t *testing.T) {
		tempDir := t.TempDir()
		configPath := filepath.Join(tempDir, "test.toml")
		configContent := `
[database]
host = "localhost"
`
		if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
			t.Fatalf("Failed to write test config: %v", err)
		}

		cfg, err := Load(configPath)
		if err != nil {
			t.Fatalf("Failed to load config: %v", err)
		}

		// Test default values for missing keys
		if port := cfg.GetInt("database.port", 5432); port != 5432 {
			t.Errorf("Expected default port 5432, got %d", port)
		}

		if ssl := cfg.GetBool("database.ssl", true); !ssl {
			t.Errorf("Expected default ssl true, got %v", ssl)
		}

		if timeout := cfg.GetDuration("server.timeout", 30*time.Second); timeout != 30*time.Second {
			t.Errorf("Expected default timeout 30s, got %v", timeout)
		}
	})
}

func TestHasAndSet(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "test.toml")
	configContent := `
[database]
host = "localhost"
`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Test Has method
	if !cfg.Has("database.host") {
		t.Error("Expected database.host to exist")
	}

	if cfg.Has("database.port") {
		t.Error("Expected database.port to not exist")
	}

	// Test Set method
	cfg.Set("database.port", 5432)
	if !cfg.Has("database.port") {
		t.Error("Expected database.port to exist after Set")
	}

	if port := cfg.GetInt("database.port"); port != 5432 {
		t.Errorf("Expected port 5432 after Set, got %d", port)
	}

	// Test nested Set
	cfg.Set("server.new.nested.value", "test")
	if value := cfg.GetString("server.new.nested.value"); value != "test" {
		t.Errorf("Expected nested value 'test', got '%s'", value)
	}
}

func TestGetAll(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "test.toml")
	configContent := `
[database]
host = "localhost"
port = 5432

[server]
workers = 4
`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	all := cfg.GetAll()
	
	// Check that data structure is preserved
	if database, ok := all["database"].(map[string]interface{}); ok {
		if host, ok := database["host"].(string); !ok || host != "localhost" {
			t.Errorf("Expected host 'localhost', got '%v'", database["host"])
		}
	} else {
		t.Error("Expected database section to be a map")
	}
}

func TestLoadFromString(t *testing.T) {
	t.Run("TOML string", func(t *testing.T) {
		configContent := `
[database]
host = "localhost"
port = 5432
`
		cfg, err := LoadFromString(configContent, FormatTOML)
		if err != nil {
			t.Fatalf("Failed to load config from string: %v", err)
		}

		if host := cfg.GetString("database.host"); host != "localhost" {
			t.Errorf("Expected host 'localhost', got '%s'", host)
		}
	})

	t.Run("YAML string", func(t *testing.T) {
		configContent := `
database:
  host: localhost
  port: 5432
`
		cfg, err := LoadFromString(configContent, FormatYAML)
		if err != nil {
			t.Fatalf("Failed to load config from string: %v", err)
		}

		if host := cfg.GetString("database.host"); host != "localhost" {
			t.Errorf("Expected host 'localhost', got '%s'", host)
		}
	})
}

func TestFormatDetection(t *testing.T) {
	tests := []struct {
		filename string
		expected Format
	}{
		{"config.toml", FormatTOML},
		{"config.yaml", FormatYAML},
		{"config.yml", FormatYAML},
		{"config.txt", FormatTOML}, // Default fallback
		{"config", FormatTOML},     // Default fallback
	}

	for _, test := range tests {
		t.Run(test.filename, func(t *testing.T) {
			format := detectFormat(test.filename)
			if format != test.expected {
				t.Errorf("Expected format %v for %s, got %v", test.expected, test.filename, format)
			}
		})
	}
}

func TestFilePathAndFormat(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "test.toml")
	configContent := `[test]
value = "test"
`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	if cfg.FilePath() != configPath {
		t.Errorf("Expected file path '%s', got '%s'", configPath, cfg.FilePath())
	}

	if cfg.Format() != FormatTOML {
		t.Errorf("Expected format TOML, got %v", cfg.Format())
	}
}

func BenchmarkGetString(b *testing.B) {
	tempDir := b.TempDir()
	configPath := filepath.Join(tempDir, "bench.toml")
	configContent := `
[database]
host = "localhost"
port = 5432

[server]
workers = 4
timeout = "30s"
`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		b.Fatalf("Failed to write test config: %v", err)
	}

	cfg, err := Load(configPath)
	if err != nil {
		b.Fatalf("Failed to load config: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = cfg.GetString("database.host")
	}
}

func BenchmarkGetInt(b *testing.B) {
	tempDir := b.TempDir()
	configPath := filepath.Join(tempDir, "bench.toml")
	configContent := `
[database]
host = "localhost"
port = 5432
`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		b.Fatalf("Failed to write test config: %v", err)
	}

	cfg, err := Load(configPath)
	if err != nil {
		b.Fatalf("Failed to load config: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = cfg.GetInt("database.port")
	}
}