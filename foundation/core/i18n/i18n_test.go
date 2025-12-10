// File: i18n_test.go
// Title: Internationalization Module Tests
// Description: Comprehensive tests for the i18n module covering TOML/YAML
//              parsing, locale detection, translation templates, pluralization,
//              and all core internationalization functionality.
// Author: msto63 with Claude Sonnet 4.0
// Version: v0.1.0
// Created: 2025-01-25
// Modified: 2025-01-25
//
// Change History:
// - 2025-01-25 v0.1.0: Initial test implementation

package i18n

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestNew(t *testing.T) {
	// Create temporary directory for test files
	tempDir := t.TempDir()

	t.Run("create with valid options", func(t *testing.T) {
		// Create test locale files
		enContent := `
[messages]
welcome = "Welcome"
goodbye = "Goodbye"

[errors]
not_found = "Not found"
`
		enPath := filepath.Join(tempDir, "en.toml")
		if err := os.WriteFile(enPath, []byte(enContent), 0644); err != nil {
			t.Fatalf("Failed to write en.toml: %v", err)
		}

		manager, err := New(Options{
			DefaultLocale: "en",
			LocalesDir:    tempDir,
			Format:        FormatTOML,
		})
		if err != nil {
			t.Fatalf("Failed to create i18n manager: %v", err)
		}

		if manager.GetDefaultLocale() != "en" {
			t.Errorf("Expected default locale 'en', got '%s'", manager.GetDefaultLocale())
		}

		if manager.GetCurrentLocale() != "en" {
			t.Errorf("Expected current locale 'en', got '%s'", manager.GetCurrentLocale())
		}
	})

	t.Run("empty default locale", func(t *testing.T) {
		_, err := New(Options{
			DefaultLocale: "",
			LocalesDir:    tempDir,
		})
		if err == nil {
			t.Error("Expected error for empty default locale")
		}
	})

	t.Run("nonexistent locales directory", func(t *testing.T) {
		_, err := New(Options{
			DefaultLocale: "en",
			LocalesDir:    "/nonexistent/directory",
		})
		if err == nil {
			t.Error("Expected error for nonexistent locales directory")
		}
	})
}

func TestTranslation(t *testing.T) {
	tempDir := t.TempDir()

	// Create test locale files
	enContent := `
[messages]
welcome = "Welcome, {{.Name}}!"
simple = "Hello"

[nested]
[nested.deep]
value = "Deep value"

[plurals]
item_count = ["{{.Count}} item", "{{.Count}} items"]
`
	enPath := filepath.Join(tempDir, "en.toml")
	if err := os.WriteFile(enPath, []byte(enContent), 0644); err != nil {
		t.Fatalf("Failed to write en.toml: %v", err)
	}

	deContent := `
[messages]
welcome = "Willkommen, {{.Name}}!"
simple = "Hallo"

[nested]
[nested.deep]
value = "Tiefer Wert"
`
	dePath := filepath.Join(tempDir, "de.toml")
	if err := os.WriteFile(dePath, []byte(deContent), 0644); err != nil {
		t.Fatalf("Failed to write de.toml: %v", err)
	}

	manager, err := New(Options{
		DefaultLocale: "en",
		LocalesDir:    tempDir,
		Format:        FormatTOML,
	})
	if err != nil {
		t.Fatalf("Failed to create i18n manager: %v", err)
	}

	t.Run("simple translation", func(t *testing.T) {
		result := manager.T("messages.simple")
		if result != "Hello" {
			t.Errorf("Expected 'Hello', got '%s'", result)
		}
	})

	t.Run("template translation", func(t *testing.T) {
		result := manager.T("messages.welcome", map[string]interface{}{
			"Name": "John",
		})
		if result != "Welcome, John!" {
			t.Errorf("Expected 'Welcome, John!', got '%s'", result)
		}
	})

	t.Run("nested translation", func(t *testing.T) {
		result := manager.T("nested.deep.value")
		if result != "Deep value" {
			t.Errorf("Expected 'Deep value', got '%s'", result)
		}
	})

	t.Run("missing translation", func(t *testing.T) {
		result := manager.T("missing.key")
		if result != "" {
			t.Errorf("Expected empty string for missing key, got '%s'", result)
		}
	})

	t.Run("TryT with existing key", func(t *testing.T) {
		result, err := manager.TryT("messages.simple")
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if result != "Hello" {
			t.Errorf("Expected 'Hello', got '%s'", result)
		}
	})

	t.Run("TryT with missing key", func(t *testing.T) {
		_, err := manager.TryT("missing.key")
		if err == nil {
			t.Error("Expected error for missing key")
		}
	})

	t.Run("TWithFallback", func(t *testing.T) {
		result := manager.TWithFallback("missing.key", "Default message")
		if result != "Default message" {
			t.Errorf("Expected 'Default message', got '%s'", result)
		}

		result = manager.TWithFallback("messages.simple", "Default message")
		if result != "Hello" {
			t.Errorf("Expected 'Hello', got '%s'", result)
		}
	})

	t.Run("locale switching", func(t *testing.T) {
		// Switch to German
		if err := manager.SetLocale("de"); err != nil {
			t.Fatalf("Failed to set locale to de: %v", err)
		}

		result := manager.T("messages.simple")
		if result != "Hallo" {
			t.Errorf("Expected 'Hallo', got '%s'", result)
		}

		// Switch back to English
		if err := manager.SetLocale("en"); err != nil {
			t.Fatalf("Failed to set locale to en: %v", err)
		}

		result = manager.T("messages.simple")
		if result != "Hello" {
			t.Errorf("Expected 'Hello', got '%s'", result)
		}
	})

	t.Run("invalid locale", func(t *testing.T) {
		err := manager.SetLocale("invalid")
		if err == nil {
			t.Error("Expected error for invalid locale")
		}
	})
}

func TestYAMLFormat(t *testing.T) {
	tempDir := t.TempDir()

	// Create YAML locale file
	yamlContent := `
messages:
  welcome: "Welcome, {{.Name}}!"
  simple: "Hello"

nested:
  deep:
    value: "Deep value"
`
	yamlPath := filepath.Join(tempDir, "en.yaml")
	if err := os.WriteFile(yamlPath, []byte(yamlContent), 0644); err != nil {
		t.Fatalf("Failed to write en.yaml: %v", err)
	}

	manager, err := New(Options{
		DefaultLocale: "en",
		LocalesDir:    tempDir,
		Format:        FormatYAML,
	})
	if err != nil {
		t.Fatalf("Failed to create i18n manager: %v", err)
	}

	t.Run("YAML simple translation", func(t *testing.T) {
		result := manager.T("messages.simple")
		if result != "Hello" {
			t.Errorf("Expected 'Hello', got '%s'", result)
		}
	})

	t.Run("YAML template translation", func(t *testing.T) {
		result := manager.T("messages.welcome", map[string]interface{}{
			"Name": "Jane",
		})
		if result != "Welcome, Jane!" {
			t.Errorf("Expected 'Welcome, Jane!', got '%s'", result)
		}
	})

	t.Run("YAML nested translation", func(t *testing.T) {
		result := manager.T("nested.deep.value")
		if result != "Deep value" {
			t.Errorf("Expected 'Deep value', got '%s'", result)
		}
	})
}

func TestPluralization(t *testing.T) {
	tempDir := t.TempDir()

	enContent := `
[plurals]
item_count = ["{{.Count}} item", "{{.Count}} items"]
simple_count = ["one", "many"]
`
	enPath := filepath.Join(tempDir, "en.toml")
	if err := os.WriteFile(enPath, []byte(enContent), 0644); err != nil {
		t.Fatalf("Failed to write en.toml: %v", err)
	}

	manager, err := New(Options{
		DefaultLocale: "en",
		LocalesDir:    tempDir,
		Format:        FormatTOML,
	})
	if err != nil {
		t.Fatalf("Failed to create i18n manager: %v", err)
	}

	t.Run("singular form", func(t *testing.T) {
		result := manager.Plural("plurals.item_count", 1, map[string]interface{}{
			"Count": 1,
		})
		if result != "1 item" {
			t.Errorf("Expected '1 item', got '%s'", result)
		}
	})

	t.Run("plural form", func(t *testing.T) {
		result := manager.Plural("plurals.item_count", 5, map[string]interface{}{
			"Count": 5,
		})
		if result != "5 items" {
			t.Errorf("Expected '5 items', got '%s'", result)
		}
	})

	t.Run("simple plural without template", func(t *testing.T) {
		result := manager.Plural("plurals.simple_count", 1, nil)
		if result != "one" {
			t.Errorf("Expected 'one', got '%s'", result)
		}

		result = manager.Plural("plurals.simple_count", 5, nil)
		if result != "many" {
			t.Errorf("Expected 'many', got '%s'", result)
		}
	})

	t.Run("missing plural key", func(t *testing.T) {
		result := manager.Plural("missing.key", 1, nil)
		if !strings.Contains(result, "missing.key") {
			t.Errorf("Expected result to contain key name, got '%s'", result)
		}
	})
}

func TestLocaleDetection(t *testing.T) {
	tempDir := t.TempDir()

	// Create multiple locale files
	for _, locale := range []string{"en", "de", "fr"} {
		content := `[test]
value = "test"
`
		path := filepath.Join(tempDir, locale+".toml")
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to write %s.toml: %v", locale, err)
		}
	}

	manager, err := New(Options{
		DefaultLocale: "en",
		LocalesDir:    tempDir,
		Format:        FormatTOML,
	})
	if err != nil {
		t.Fatalf("Failed to create i18n manager: %v", err)
	}

	tests := []struct {
		acceptLanguage string
		expected       string
	}{
		{"en-US,en;q=0.9,de;q=0.8", "en"},
		{"de-DE,de;q=0.9,en;q=0.8", "de"},
		{"fr-FR,fr;q=0.9", "fr"},
		{"es-ES,es;q=0.9", "en"}, // Fallback to default
		{"", "en"},                // Empty header
		{"invalid", "en"},         // Invalid header
	}

	for _, test := range tests {
		t.Run("detect_"+test.acceptLanguage, func(t *testing.T) {
			result := manager.DetectLocale(test.acceptLanguage)
			if result != test.expected {
				t.Errorf("For Accept-Language '%s', expected '%s', got '%s'",
					test.acceptLanguage, test.expected, result)
			}
		})
	}
}

func TestLocaleNormalization(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"en", "en"},
		{"en-US", "en-US"},
		{"en_US", "en-US"},
		{"DE-de", "de-DE"},
		{"fr_FR", "fr-FR"},
		{"", ""},
		{"invalid-toolong", ""},
		{"x", ""},
	}

	for _, test := range tests {
		t.Run("normalize_"+test.input, func(t *testing.T) {
			result := NormalizeLocale(test.input)
			if result != test.expected {
				t.Errorf("For input '%s', expected '%s', got '%s'",
					test.input, test.expected, result)
			}
		})
	}
}

func TestLocaleValidation(t *testing.T) {
	tests := []struct {
		locale string
		valid  bool
	}{
		{"en", true},
		{"en-US", true},
		{"de-DE", true},
		{"", false},
		{"invalid-toolong", false},
		{"x", false},
	}

	for _, test := range tests {
		t.Run("validate_"+test.locale, func(t *testing.T) {
			err := ValidateLocale(test.locale)
			if test.valid && err != nil {
				t.Errorf("Expected locale '%s' to be valid, got error: %v", test.locale, err)
			}
			if !test.valid && err == nil {
				t.Errorf("Expected locale '%s' to be invalid", test.locale)
			}
		})
	}
}

func TestSplitLocale(t *testing.T) {
	tests := []struct {
		locale   string
		language string
		country  string
	}{
		{"en", "en", ""},
		{"en-US", "en", "US"},
		{"de-DE", "de", "DE"},
		{"fr_CA", "fr", "CA"},
		{"", "", ""},
	}

	for _, test := range tests {
		t.Run("split_"+test.locale, func(t *testing.T) {
			language, country := SplitLocale(test.locale)
			if language != test.language {
				t.Errorf("Expected language '%s', got '%s'", test.language, language)
			}
			if country != test.country {
				t.Errorf("Expected country '%s', got '%s'", test.country, country)
			}
		})
	}
}

func TestGetLocaleDisplayName(t *testing.T) {
	tests := []struct {
		locale   string
		expected string
	}{
		{"en", "English"},
		{"en-US", "English (United States)"},
		{"de", "Deutsch"},
		{"de-DE", "Deutsch (Deutschland)"},
		{"unknown", "unknown"},
	}

	for _, test := range tests {
		t.Run("display_"+test.locale, func(t *testing.T) {
			result := GetLocaleDisplayName(test.locale)
			if result != test.expected {
				t.Errorf("Expected display name '%s' for '%s', got '%s'",
					test.expected, test.locale, result)
			}
		})
	}
}

func TestManager_GetAvailableLocales(t *testing.T) {
	tempDir := t.TempDir()

	// Create multiple locale files
	locales := []string{"en", "de", "fr", "es"}
	for _, locale := range locales {
		content := `[test]
value = "test"
`
		path := filepath.Join(tempDir, locale+".toml")
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to write %s.toml: %v", locale, err)
		}
	}

	manager, err := New(Options{
		DefaultLocale: "en",
		LocalesDir:    tempDir,
		Format:        FormatTOML,
	})
	if err != nil {
		t.Fatalf("Failed to create i18n manager: %v", err)
	}

	available := manager.GetAvailableLocales()
	if len(available) != len(locales) {
		t.Errorf("Expected %d locales, got %d", len(locales), len(available))
	}

	// Check that all expected locales are present
	localeMap := make(map[string]bool)
	for _, locale := range available {
		localeMap[locale] = true
	}

	for _, expected := range locales {
		if !localeMap[expected] {
			t.Errorf("Expected locale '%s' not found in available locales", expected)
		}
	}
}

func TestManager_HasLocale(t *testing.T) {
	tempDir := t.TempDir()

	enContent := `[test]
value = "test"
`
	enPath := filepath.Join(tempDir, "en.toml")
	if err := os.WriteFile(enPath, []byte(enContent), 0644); err != nil {
		t.Fatalf("Failed to write en.toml: %v", err)
	}

	manager, err := New(Options{
		DefaultLocale: "en",
		LocalesDir:    tempDir,
		Format:        FormatTOML,
	})
	if err != nil {
		t.Fatalf("Failed to create i18n manager: %v", err)
	}

	if !manager.HasLocale("en") {
		t.Error("Expected to have locale 'en'")
	}

	if manager.HasLocale("de") {
		t.Error("Expected not to have locale 'de'")
	}
}

func TestManager_HasTranslation(t *testing.T) {
	tempDir := t.TempDir()

	enContent := `
[messages]
welcome = "Welcome"

[nested]
[nested.deep]
value = "Deep value"
`
	enPath := filepath.Join(tempDir, "en.toml")
	if err := os.WriteFile(enPath, []byte(enContent), 0644); err != nil {
		t.Fatalf("Failed to write en.toml: %v", err)
	}

	manager, err := New(Options{
		DefaultLocale: "en",
		LocalesDir:    tempDir,
		Format:        FormatTOML,
	})
	if err != nil {
		t.Fatalf("Failed to create i18n manager: %v", err)
	}

	if !manager.HasTranslation("messages.welcome") {
		t.Error("Expected to have translation 'messages.welcome'")
	}

	if !manager.HasTranslation("nested.deep.value") {
		t.Error("Expected to have translation 'nested.deep.value'")
	}

	if manager.HasTranslation("missing.key") {
		t.Error("Expected not to have translation 'missing.key'")
	}
}

func TestManager_GetTranslationKeys(t *testing.T) {
	tempDir := t.TempDir()

	enContent := `
[messages]
welcome = "Welcome"
goodbye = "Goodbye"

[errors]
not_found = "Not found"

[nested]
[nested.deep]
value = "Deep value"
`
	enPath := filepath.Join(tempDir, "en.toml")
	if err := os.WriteFile(enPath, []byte(enContent), 0644); err != nil {
		t.Fatalf("Failed to write en.toml: %v", err)
	}

	manager, err := New(Options{
		DefaultLocale: "en",
		LocalesDir:    tempDir,
		Format:        FormatTOML,
	})
	if err != nil {
		t.Fatalf("Failed to create i18n manager: %v", err)
	}

	keys := manager.GetTranslationKeys()
	expectedKeys := []string{
		"errors.not_found",
		"messages.goodbye",
		"messages.welcome",
		"nested.deep.value",
	}

	if len(keys) != len(expectedKeys) {
		t.Errorf("Expected %d keys, got %d", len(expectedKeys), len(keys))
	}

	for i, expected := range expectedKeys {
		if i >= len(keys) || keys[i] != expected {
			t.Errorf("Expected key[%d] to be '%s', got '%s'", i, expected, keys[i])
		}
	}
}

func TestLocaleChangeHandler(t *testing.T) {
	tempDir := t.TempDir()

	enContent := `[test]
value = "test"
`
	enPath := filepath.Join(tempDir, "en.toml")
	if err := os.WriteFile(enPath, []byte(enContent), 0644); err != nil {
		t.Fatalf("Failed to write en.toml: %v", err)
	}

	manager, err := New(Options{
		DefaultLocale: "en",
		LocalesDir:    tempDir,
		Format:        FormatTOML,
		Watch:         false, // Disable automatic watching for this test
	})
	if err != nil {
		t.Fatalf("Failed to create i18n manager: %v", err)
	}

	// Test handler registration
	called := false
	var receivedLocale string
	var receivedTranslations map[string]interface{}

	manager.OnLocaleChange(func(locale string, translations map[string]interface{}) {
		called = true
		receivedLocale = locale
		receivedTranslations = translations
	})

	// Manually trigger a reload (simulates file change)
	if err := manager.reloadLocale("en"); err != nil {
		t.Fatalf("Failed to reload locale: %v", err)
	}

	// Give goroutine time to execute
	time.Sleep(100 * time.Millisecond)

	if !called {
		t.Error("Expected locale change handler to be called")
	}

	if receivedLocale != "en" {
		t.Errorf("Expected locale 'en', got '%s'", receivedLocale)
	}

	if receivedTranslations == nil {
		t.Error("Expected translations to be provided")
	}
}

func TestFormatString(t *testing.T) {
	tests := []struct {
		format   Format
		expected string
	}{
		{FormatTOML, "toml"},
		{FormatYAML, "yaml"},
		{FormatAuto, "auto"},
		{Format(999), "unknown"},
	}

	for _, test := range tests {
		t.Run("format_"+test.expected, func(t *testing.T) {
			result := test.format.String()
			if result != test.expected {
				t.Errorf("Expected '%s', got '%s'", test.expected, result)
			}
		})
	}
}

func BenchmarkTranslation(b *testing.B) {
	tempDir := b.TempDir()

	enContent := `
[messages]
welcome = "Welcome, {{.Name}}!"
simple = "Hello"

[nested]
[nested.deep]
value = "Deep value"
`
	enPath := filepath.Join(tempDir, "en.toml")
	if err := os.WriteFile(enPath, []byte(enContent), 0644); err != nil {
		b.Fatalf("Failed to write en.toml: %v", err)
	}

	manager, err := New(Options{
		DefaultLocale: "en",
		LocalesDir:    tempDir,
		Format:        FormatTOML,
	})
	if err != nil {
		b.Fatalf("Failed to create i18n manager: %v", err)
	}

	b.Run("simple translation", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = manager.T("messages.simple")
		}
	})

	b.Run("template translation", func(b *testing.B) {
		data := map[string]interface{}{
			"Name": "John",
		}
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = manager.T("messages.welcome", data)
		}
	})

	b.Run("nested translation", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = manager.T("nested.deep.value")
		}
	})
}

func BenchmarkLocaleDetection(b *testing.B) {
	tempDir := b.TempDir()

	// Create multiple locale files
	for _, locale := range []string{"en", "de", "fr", "es"} {
		content := `[test]
value = "test"
`
		path := filepath.Join(tempDir, locale+".toml")
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			b.Fatalf("Failed to write %s.toml: %v", locale, err)
		}
	}

	manager, err := New(Options{
		DefaultLocale: "en",
		LocalesDir:    tempDir,
		Format:        FormatTOML,
	})
	if err != nil {
		b.Fatalf("Failed to create i18n manager: %v", err)
	}

	acceptLanguage := "en-US,en;q=0.9,de;q=0.8,fr;q=0.7"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = manager.DetectLocale(acceptLanguage)
	}
}