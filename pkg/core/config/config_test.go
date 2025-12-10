package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestDuration_UnmarshalText(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected time.Duration
		wantErr  bool
	}{
		{"seconds", "30s", 30 * time.Second, false},
		{"minutes", "5m", 5 * time.Minute, false},
		{"hours", "2h", 2 * time.Hour, false},
		{"complex", "1h30m", 90 * time.Minute, false},
		{"milliseconds", "100ms", 100 * time.Millisecond, false},
		{"invalid", "invalid", 0, true},
		{"empty", "", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var d Duration
			err := d.UnmarshalText([]byte(tt.input))

			if (err != nil) != tt.wantErr {
				t.Errorf("UnmarshalText() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && d.Duration != tt.expected {
				t.Errorf("UnmarshalText() = %v, want %v", d.Duration, tt.expected)
			}
		})
	}
}

func TestDuration_MarshalText(t *testing.T) {
	tests := []struct {
		name     string
		duration time.Duration
		expected string
	}{
		{"seconds", 30 * time.Second, "30s"},
		{"minutes", 5 * time.Minute, "5m0s"},
		{"hours", 2 * time.Hour, "2h0m0s"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := Duration{tt.duration}
			result, err := d.MarshalText()

			if err != nil {
				t.Errorf("MarshalText() error = %v", err)
				return
			}

			if string(result) != tt.expected {
				t.Errorf("MarshalText() = %v, want %v", string(result), tt.expected)
			}
		})
	}
}

func TestConfig_applyDefaults(t *testing.T) {
	cfg := &Config{}
	cfg.applyDefaults()

	// General defaults
	if cfg.General.Name != "meinDENKWERK" {
		t.Errorf("General.Name = %v, want meinDENKWERK", cfg.General.Name)
	}
	if cfg.General.Environment != "development" {
		t.Errorf("General.Environment = %v, want development", cfg.General.Environment)
	}
	if cfg.General.DataDir != "./data" {
		t.Errorf("General.DataDir = %v, want ./data", cfg.General.DataDir)
	}
	if cfg.General.LogLevel != "info" {
		t.Errorf("General.LogLevel = %v, want info", cfg.General.LogLevel)
	}

	// Kant defaults
	if cfg.Kant.Port != 8080 {
		t.Errorf("Kant.Port = %v, want 8080", cfg.Kant.Port)
	}
	if cfg.Kant.Host != "0.0.0.0" {
		t.Errorf("Kant.Host = %v, want 0.0.0.0", cfg.Kant.Host)
	}
	if cfg.Kant.ReadTimeout.Duration != 30*time.Second {
		t.Errorf("Kant.ReadTimeout = %v, want 30s", cfg.Kant.ReadTimeout.Duration)
	}

	// Russell defaults
	if cfg.Russell.Port != 9100 {
		t.Errorf("Russell.Port = %v, want 9100", cfg.Russell.Port)
	}

	// Turing defaults
	if cfg.Turing.Port != 9200 {
		t.Errorf("Turing.Port = %v, want 9200", cfg.Turing.Port)
	}
	if cfg.Turing.DefaultProvider != "ollama" {
		t.Errorf("Turing.DefaultProvider = %v, want ollama", cfg.Turing.DefaultProvider)
	}
	if cfg.Turing.DefaultModel != "mistral:7b" {
		t.Errorf("Turing.DefaultModel = %v, want mistral:7b", cfg.Turing.DefaultModel)
	}

	// Hypatia defaults
	if cfg.Hypatia.Port != 9220 {
		t.Errorf("Hypatia.Port = %v, want 9220", cfg.Hypatia.Port)
	}
	if cfg.Hypatia.DefaultTopK != 5 {
		t.Errorf("Hypatia.DefaultTopK = %v, want 5", cfg.Hypatia.DefaultTopK)
	}
	if cfg.Hypatia.Chunking.DefaultSize != 512 {
		t.Errorf("Hypatia.Chunking.DefaultSize = %v, want 512", cfg.Hypatia.Chunking.DefaultSize)
	}

	// Leibniz defaults
	if cfg.Leibniz.Port != 9140 {
		t.Errorf("Leibniz.Port = %v, want 9140", cfg.Leibniz.Port)
	}
	if cfg.Leibniz.MaxIterations != 10 {
		t.Errorf("Leibniz.MaxIterations = %v, want 10", cfg.Leibniz.MaxIterations)
	}

	// Babbage defaults
	if cfg.Babbage.Port != 9150 {
		t.Errorf("Babbage.Port = %v, want 9150", cfg.Babbage.Port)
	}
	if cfg.Babbage.DefaultLanguage != "de" {
		t.Errorf("Babbage.DefaultLanguage = %v, want de", cfg.Babbage.DefaultLanguage)
	}

	// Bayes defaults
	if cfg.Bayes.Port != 9120 {
		t.Errorf("Bayes.Port = %v, want 9120", cfg.Bayes.Port)
	}
	if cfg.Bayes.RetentionDays != 30 {
		t.Errorf("Bayes.RetentionDays = %v, want 30", cfg.Bayes.RetentionDays)
	}
}

func TestConfig_GetServiceAddress(t *testing.T) {
	cfg := &Config{}
	cfg.applyDefaults()

	tests := []struct {
		service  string
		expected string
	}{
		{"kant", "0.0.0.0:8080"},
		{"russell", "0.0.0.0:9100"},
		{"turing", "0.0.0.0:9200"},
		{"hypatia", "0.0.0.0:9220"},
		{"leibniz", "0.0.0.0:9140"},
		{"babbage", "0.0.0.0:9150"},
		{"bayes", "0.0.0.0:9120"},
		{"unknown", ""},
	}

	for _, tt := range tests {
		t.Run(tt.service, func(t *testing.T) {
			result := cfg.GetServiceAddress(tt.service)
			if result != tt.expected {
				t.Errorf("GetServiceAddress(%q) = %v, want %v", tt.service, result, tt.expected)
			}
		})
	}
}

func TestLoad_FileNotFound(t *testing.T) {
	_, err := Load("/nonexistent/path/config.toml")
	if err == nil {
		t.Error("Load() expected error for non-existent file")
	}
}

func TestLoad_ValidConfig(t *testing.T) {
	// Create temp config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.toml")

	configContent := `
[general]
name = "TestDENKWERK"
environment = "test"

[kant]
port = 9999
host = "127.0.0.1"

[turing]
default_model = "test-model"

[turing.providers.ollama]
enabled = true
base_url = "http://localhost:11434"
`

	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.General.Name != "TestDENKWERK" {
		t.Errorf("General.Name = %v, want TestDENKWERK", cfg.General.Name)
	}
	if cfg.Kant.Port != 9999 {
		t.Errorf("Kant.Port = %v, want 9999", cfg.Kant.Port)
	}
	if cfg.Kant.Host != "127.0.0.1" {
		t.Errorf("Kant.Host = %v, want 127.0.0.1", cfg.Kant.Host)
	}
	if cfg.Turing.DefaultModel != "test-model" {
		t.Errorf("Turing.DefaultModel = %v, want test-model", cfg.Turing.DefaultModel)
	}

	// Check defaults were applied for missing values
	if cfg.Russell.Port != 9100 {
		t.Errorf("Russell.Port = %v, want 9100 (default)", cfg.Russell.Port)
	}
}

func TestConfig_expandEnvVars(t *testing.T) {
	os.Setenv("TEST_API_KEY", "secret-key-123")
	defer os.Unsetenv("TEST_API_KEY")

	cfg := &Config{
		Turing: TuringConfig{
			Providers: ProvidersConfig{
				OpenAI: ProviderConfig{
					APIKey: "$TEST_API_KEY",
				},
			},
		},
	}

	cfg.expandEnvVars()

	if cfg.Turing.Providers.OpenAI.APIKey != "secret-key-123" {
		t.Errorf("APIKey = %v, want secret-key-123", cfg.Turing.Providers.OpenAI.APIKey)
	}
}

func TestLoadFromEnv_NoConfigFound(t *testing.T) {
	// Temporarily unset MDW_CONFIG
	original := os.Getenv("MDW_CONFIG")
	os.Unsetenv("MDW_CONFIG")
	defer func() {
		if original != "" {
			os.Setenv("MDW_CONFIG", original)
		}
	}()

	// Change to a temp directory without config files
	originalWd, _ := os.Getwd()
	tmpDir := t.TempDir()
	os.Chdir(tmpDir)
	defer os.Chdir(originalWd)

	_, err := LoadFromEnv()
	if err == nil {
		t.Error("LoadFromEnv() expected error when no config found")
	}
}
