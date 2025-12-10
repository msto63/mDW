// File: i18n.go
// Title: Core Internationalization Implementation
// Description: Implements the main i18n Manager and core functionality for
//              loading, parsing, and managing translations from TOML and YAML
//              language files with template interpolation and pluralization.
// Author: msto63 with Claude Sonnet 4.0
// Version: v0.1.1
// Created: 2025-01-25
// Modified: 2025-07-26
//
// Change History:
// - 2025-01-25 v0.1.0: Initial implementation with TOML/YAML support
// - 2025-07-26 v0.1.1: Fixed template cache collision issue in pluralization,
//                       improved cache key uniqueness for plural forms

package i18n

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"text/template"

	"github.com/BurntSushi/toml"
	"gopkg.in/yaml.v3"

	mdwerror "github.com/msto63/mDW/foundation/core/error"
	mdwstringx "github.com/msto63/mDW/foundation/utils/stringx"
)

// Format represents the language file format
type Format int

const (
	// FormatTOML represents TOML format (default)
	FormatTOML Format = iota
	
	// FormatYAML represents YAML format
	FormatYAML
	
	// FormatAuto auto-detects format from file extension
	FormatAuto
)

// String returns the string representation of the format
func (f Format) String() string {
	switch f {
	case FormatTOML:
		return "toml"
	case FormatYAML:
		return "yaml"
	case FormatAuto:
		return "auto"
	default:
		return "unknown"
	}
}

// Options defines configuration options for the i18n manager
type Options struct {
	DefaultLocale string // Default locale (e.g., "en")
	LocalesDir    string // Directory containing language files
	Format        Format // File format (default: auto-detect)
	Watch         bool   // Enable file watching for hot-reloading
	Fallback      bool   // Enable fallback to default locale (default: true)
}

// Manager manages internationalization for an application
type Manager struct {
	mu              sync.RWMutex
	defaultLocale   string
	currentLocale   string
	localesDir      string
	format          Format
	fallback        bool
	translations    map[string]map[string]interface{} // locale -> translations
	templates       map[string]*template.Template     // key -> compiled template
	watchers        []LocaleChangeHandler
	watching        bool
	
	// Context information for better error reporting and tracing
	requestID       string
	userID          string
	correlationID   string
}

// LocaleChangeHandler is called when locale files change
type LocaleChangeHandler func(locale string, translations map[string]interface{})

// TranslationData represents the structure of a translation file
type TranslationData map[string]interface{}

// New creates a new i18n manager with the specified options
func New(options Options) (*Manager, error) {
	if mdwstringx.IsBlank(options.DefaultLocale) {
		return nil, mdwerror.New("default locale cannot be empty").WithCode(mdwerror.CodeValidationFailed).WithOperation("i18n.New")
	}

	if mdwstringx.IsBlank(options.LocalesDir) {
		options.LocalesDir = "./locales"
	}

	if options.Format == FormatAuto {
		options.Format = FormatTOML // Default to TOML
	}

	// Check if locales directory exists
	if _, err := os.Stat(options.LocalesDir); os.IsNotExist(err) {
		return nil, mdwerror.New("locales directory not found").WithCode(mdwerror.CodeNotFound).WithOperation("i18n.New").WithDetail("directory", options.LocalesDir)
	}

	manager := &Manager{
		defaultLocale: options.DefaultLocale,
		currentLocale: options.DefaultLocale,
		localesDir:    options.LocalesDir,
		format:        options.Format,
		fallback:      true, // Enable fallback by default
		translations:  make(map[string]map[string]interface{}),
		templates:     make(map[string]*template.Template),
		watchers:      make([]LocaleChangeHandler, 0),
		watching:      options.Watch,
	}

	// Load all available locales
	if err := manager.loadAllLocales(); err != nil {
		return nil, mdwerror.Wrap(err, "failed to load locales").WithCode(mdwerror.CodeInvalidOperation).WithOperation("i18n.loadAllLocales")
	}

	// Start watching if requested
	if options.Watch {
		go manager.startWatching()
	}

	return manager, nil
}

// NewWithWatch creates a new i18n manager with file watching enabled
func NewWithWatch(options Options) (*Manager, error) {
	options.Watch = true
	return New(options)
}

// loadAllLocales loads all available locale files from the locales directory
func (m *Manager) loadAllLocales() error {
	// Scan locales directory
	entries, err := os.ReadDir(m.localesDir)
	if err != nil {
		return fmt.Errorf("failed to read locales directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		fileName := entry.Name()
		
		// Check if file has a supported extension based on configured format
		ext := strings.ToLower(filepath.Ext(fileName))
		supportedExtensions := []string{".toml", ".yaml", ".yml"}
		if m.format == FormatTOML {
			supportedExtensions = []string{".toml"}
		} else if m.format == FormatYAML {
			supportedExtensions = []string{".yaml", ".yml"}
		}
		
		isSupported := false
		for _, supportedExt := range supportedExtensions {
			if ext == supportedExt {
				isSupported = true
				break
			}
		}
		if !isSupported {
			continue
		}

		// Extract locale from filename (e.g., "en.toml" -> "en")
		locale := strings.TrimSuffix(fileName, ext)
		if mdwstringx.IsBlank(locale) {
			continue
		}

		// Load the locale file
		if err := m.loadLocale(locale); err != nil {
			// Log error but continue loading other locales
			continue
		}
	}

	// Ensure default locale is loaded
	if _, exists := m.translations[m.defaultLocale]; !exists {
		return fmt.Errorf("default locale '%s' not found", m.defaultLocale)
	}

	return nil
}

// loadLocale loads translations for a specific locale
func (m *Manager) loadLocale(locale string) error {
	// Determine file extensions to try based on configured format
	var extensions []string
	switch m.format {
	case FormatTOML:
		extensions = []string{".toml"}
	case FormatYAML:
		extensions = []string{".yaml", ".yml"}
	case FormatAuto:
		// Auto-detect: try all, prefer TOML
		extensions = []string{".toml", ".yaml", ".yml"}
	default:
		// Default fallback (same as auto)
		extensions = []string{".toml", ".yaml", ".yml"}
	}
	
	var filePath string
	var format Format
	
	for _, ext := range extensions {
		testPath := filepath.Join(m.localesDir, locale+ext)
		if _, err := os.Stat(testPath); err == nil {
			filePath = testPath
			if ext == ".toml" {
				format = FormatTOML
			} else {
				format = FormatYAML
			}
			break
		}
	}

	if mdwstringx.IsBlank(filePath) {
		return fmt.Errorf("no translation file found for locale '%s'", locale)
	}

	// Read file content
	content, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read locale file %s: %w", filePath, err)
	}

	// Parse content
	var data TranslationData
	switch format {
	case FormatTOML:
		if err := toml.Unmarshal(content, &data); err != nil {
			return fmt.Errorf("failed to parse TOML file %s: %w", filePath, err)
		}
	case FormatYAML:
		if err := yaml.Unmarshal(content, &data); err != nil {
			return fmt.Errorf("failed to parse YAML file %s: %w", filePath, err)
		}
	default:
		return fmt.Errorf("unsupported format for file %s", filePath)
	}

	// Store translations
	m.mu.Lock()
	m.translations[locale] = data
	m.mu.Unlock()


	return nil
}

// T translates a key with optional template data
func (m *Manager) T(key string, data ...map[string]interface{}) string {
	translation, _ := m.TryT(key, data...)
	return translation
}

// TryT translates a key and returns an error if translation fails
func (m *Manager) TryT(key string, data ...map[string]interface{}) (string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Get translation
	translation := m.getTranslation(key, m.currentLocale)
	if mdwstringx.IsBlank(translation) {
		return "", mdwerror.New("translation not found").WithCode(mdwerror.CodeNotFound).WithOperation("i18n.TryT").WithDetail("key", key)
	}

	// Render template if data provided
	if len(data) > 0 && data[0] != nil {
		rendered, err := m.renderTemplate(key, translation, data[0])
		if err != nil {
			return translation, mdwerror.Wrap(err, "template rendering failed").WithCode(mdwerror.CodeInvalidOperation).WithOperation("i18n.renderTemplate")
		}
		return rendered, nil
	}

	return translation, nil
}

// TWithFallback translates a key with fallback to default message
func (m *Manager) TWithFallback(key string, fallbackMsg string, data ...map[string]interface{}) string {
	if translation, err := m.TryT(key, data...); err == nil {
		return translation
	}
	
	// Render fallback with template data if provided
	if len(data) > 0 && data[0] != nil {
		if rendered, err := m.renderTemplate(key+"_fallback", fallbackMsg, data[0]); err == nil {
			return rendered
		}
	}
	
	return fallbackMsg
}

// Plural returns the appropriate plural form based on count
func (m *Manager) Plural(key string, count int, data map[string]interface{}) string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Get raw translation value
	translations := m.translations[m.currentLocale]
	if translations == nil {
		return fmt.Sprintf("[%s]", key)
	}

	rawValue := m.getNestedRawValue(translations, key)
	if rawValue == nil {
		return fmt.Sprintf("[%s]", key)
	}

	// Handle plural forms
	forms := m.parsePluralFormsFromRaw(rawValue)
	if len(forms) == 0 {
		return fmt.Sprintf("[%s]", key)
	}

	// Select appropriate form based on count
	formIndex := m.getPluralFormIndex(count, m.currentLocale)
	if formIndex >= len(forms) {
		formIndex = len(forms) - 1
	}
	if formIndex < 0 {
		formIndex = 0
	}

	selectedForm := forms[formIndex]

	// Render template with data
	if data != nil {
		if rendered, err := m.renderTemplate(key+"_plural_"+fmt.Sprintf("%d", formIndex), selectedForm, data); err == nil {
			return rendered
		}
	}

	return selectedForm
}

// getTranslation retrieves a translation for a specific locale with fallback
func (m *Manager) getTranslation(key, locale string) string {
	// Try current locale first
	if translations, exists := m.translations[locale]; exists {
		if value := m.getNestedValue(translations, key); value != "" {
			return value
		}
	}

	// Try fallback to default locale if enabled
	if m.fallback && locale != m.defaultLocale {
		if translations, exists := m.translations[m.defaultLocale]; exists {
			if value := m.getNestedValue(translations, key); value != "" {
				return value
			}
		}
	}

	return ""
}

// getNestedValue retrieves a nested value from translations using dot notation
func (m *Manager) getNestedValue(data map[string]interface{}, key string) string {
	keys := strings.Split(key, ".")
	current := data

	for i, k := range keys {
		if i == len(keys)-1 {
			// Last key - return the value
			if value, ok := current[k]; ok {
				// Handle array values (for plurals)
				if arr, isSlice := value.([]interface{}); isSlice {
					// Return first element for non-plural calls
					if len(arr) > 0 {
						return fmt.Sprintf("%v", arr[0])
					}
					return ""
				}
				return fmt.Sprintf("%v", value)
			}
			return ""
		}

		// Navigate deeper into the structure
		if next, ok := current[k].(map[string]interface{}); ok {
			current = next
		} else if nextData, ok := current[k].(TranslationData); ok {
			current = nextData
		} else {
			return ""
		}
	}

	return ""
}

// getNestedRawValue retrieves a nested raw value (preserving arrays)
func (m *Manager) getNestedRawValue(data map[string]interface{}, key string) interface{} {
	keys := strings.Split(key, ".")
	current := data

	for i, k := range keys {
		if i == len(keys)-1 {
			// Last key - return the raw value
			if value, ok := current[k]; ok {
				return value
			}
			return nil
		}

		// Navigate deeper into the structure
		if next, ok := current[k].(map[string]interface{}); ok {
			current = next
		} else if nextData, ok := current[k].(TranslationData); ok {
			current = nextData
		} else {
			return nil
		}
	}

	return nil
}

// renderTemplate renders a translation template with data
func (m *Manager) renderTemplate(key, template string, data map[string]interface{}) (string, error) {
	// Check if template is cached
	if tmpl, exists := m.templates[key]; exists {
		var result strings.Builder
		if err := tmpl.Execute(&result, data); err != nil {
			return template, fmt.Errorf("template execution failed: %w", err)
		}
		return result.String(), nil
	}

	// Compile and cache template
	tmpl, err := m.compileTemplate(key, template)
	if err != nil {
		return template, fmt.Errorf("template compilation failed: %w", err)
	}

	m.templates[key] = tmpl

	// Execute template
	var result strings.Builder
	if err := tmpl.Execute(&result, data); err != nil {
		return template, fmt.Errorf("template execution failed: %w", err)
	}

	return result.String(), nil
}

// compileTemplate compiles a text template
func (m *Manager) compileTemplate(name, templateStr string) (*template.Template, error) {
	return template.New(name).Parse(templateStr)
}

// parsePluralFormsFromRaw parses plural forms from raw translation value
func (m *Manager) parsePluralFormsFromRaw(value interface{}) []string {
	// Check if value is a slice (from TOML/YAML array)
	if arr, ok := value.([]interface{}); ok {
		forms := make([]string, len(arr))
		for i, v := range arr {
			forms[i] = fmt.Sprintf("%v", v)
		}
		return forms
	}

	// Single form (string value)
	if str, ok := value.(string); ok {
		return []string{str}
	}

	// Fallback
	return []string{fmt.Sprintf("%v", value)}
}

// getPluralFormIndex returns the appropriate plural form index for a count and locale
func (m *Manager) getPluralFormIndex(count int, locale string) int {
	// Simplified plural rules - can be extended with full CLDR support
	switch {
	case strings.HasPrefix(locale, "en"): // English
		if count == 1 {
			return 0 // singular
		}
		return 1 // plural
	case strings.HasPrefix(locale, "de"): // German
		if count == 1 {
			return 0 // singular
		}
		return 1 // plural
	case strings.HasPrefix(locale, "fr"): // French
		if count <= 1 {
			return 0 // singular
		}
		return 1 // plural
	default:
		// Default English rules
		if count == 1 {
			return 0
		}
		return 1
	}
}

// SetLocale changes the current locale
func (m *Manager) SetLocale(locale string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check if locale is available
	if _, exists := m.translations[locale]; !exists {
		return mdwerror.New("locale not available").WithCode(mdwerror.CodeNotFound).WithOperation("i18n.SetLocale").WithDetail("locale", locale)
	}

	m.currentLocale = locale
	return nil
}

// GetCurrentLocale returns the current active locale
func (m *Manager) GetCurrentLocale() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.currentLocale
}

// GetDefaultLocale returns the default locale
func (m *Manager) GetDefaultLocale() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.defaultLocale
}

// GetAvailableLocales returns a list of all available locales
func (m *Manager) GetAvailableLocales() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	locales := make([]string, 0, len(m.translations))
	for locale := range m.translations {
		locales = append(locales, locale)
	}

	sort.Strings(locales)
	return locales
}

// HasLocale checks if a locale is available
func (m *Manager) HasLocale(locale string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	_, exists := m.translations[locale]
	return exists
}

// HasTranslation checks if a translation key exists in the current locale
func (m *Manager) HasTranslation(key string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	translation := m.getTranslation(key, m.currentLocale)
	return translation != ""
}

// GetTranslationKeys returns all available translation keys for the current locale
func (m *Manager) GetTranslationKeys() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	translations := m.translations[m.currentLocale]
	if translations == nil {
		return nil
	}

	return m.collectKeys(translations, "")
}

// collectKeys recursively collects all keys from nested translation data
func (m *Manager) collectKeys(data map[string]interface{}, prefix string) []string {
	var keys []string

	for key, value := range data {
		fullKey := key
		if prefix != "" {
			fullKey = prefix + "." + key
		}


		// Try both map types since YAML might use TranslationData type
		if nestedMap, ok := value.(map[string]interface{}); ok {
			// Recurse into nested structure
			nestedKeys := m.collectKeys(nestedMap, fullKey)
			keys = append(keys, nestedKeys...)
		} else if nestedData, ok := value.(TranslationData); ok {
			// Recurse into nested structure (TranslationData type)
			nestedKeys := m.collectKeys(nestedData, fullKey)
			keys = append(keys, nestedKeys...)
		} else {
			// Leaf value
			keys = append(keys, fullKey)
		}
	}

	sort.Strings(keys)
	return keys
}

// OnLocaleChange registers a handler for locale changes
func (m *Manager) OnLocaleChange(handler LocaleChangeHandler) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.watchers = append(m.watchers, handler)
}

// Context support methods for better tracing and error reporting

// WithRequestID creates a copy of the manager with a request ID for tracing
func (m *Manager) WithRequestID(requestID string) *Manager {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	// Create shallow copy with same data but new context
	clone := &Manager{
		defaultLocale:   m.defaultLocale,
		currentLocale:   m.currentLocale,
		localesDir:      m.localesDir,
		format:          m.format,
		fallback:        m.fallback,
		translations:    m.translations, // Shared data
		templates:       m.templates,    // Shared templates
		watchers:        append([]LocaleChangeHandler(nil), m.watchers...),
		watching:        m.watching,
		requestID:       requestID,
		userID:          m.userID,
		correlationID:   m.correlationID,
	}
	return clone
}

// WithUserID creates a copy of the manager with a user ID for tracing
func (m *Manager) WithUserID(userID string) *Manager {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	clone := &Manager{
		defaultLocale:   m.defaultLocale,
		currentLocale:   m.currentLocale,
		localesDir:      m.localesDir,
		format:          m.format,
		fallback:        m.fallback,
		translations:    m.translations,
		templates:       m.templates,
		watchers:        append([]LocaleChangeHandler(nil), m.watchers...),
		watching:        m.watching,
		requestID:       m.requestID,
		userID:          userID,
		correlationID:   m.correlationID,
	}
	return clone
}

// WithCorrelationID creates a copy of the manager with a correlation ID for tracing
func (m *Manager) WithCorrelationID(correlationID string) *Manager {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	clone := &Manager{
		defaultLocale:   m.defaultLocale,
		currentLocale:   m.currentLocale,
		localesDir:      m.localesDir,
		format:          m.format,
		fallback:        m.fallback,
		translations:    m.translations,
		templates:       m.templates,
		watchers:        append([]LocaleChangeHandler(nil), m.watchers...),
		watching:        m.watching,
		requestID:       m.requestID,
		userID:          m.userID,
		correlationID:   correlationID,
	}
	return clone
}

// String provides a readable representation of the manager
func (m *Manager) String() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	parts := []string{
		fmt.Sprintf("i18n.Manager{defaultLocale: %s, currentLocale: %s", m.defaultLocale, m.currentLocale),
	}
	
	if m.localesDir != "" {
		parts = append(parts, fmt.Sprintf("localesDir: %s", m.localesDir))
	}
	
	parts = append(parts, fmt.Sprintf("format: %s", m.format.String()))
	
	if m.fallback {
		parts = append(parts, "fallback: true")
	}
	
	if m.watching {
		parts = append(parts, "watching: true")
	}
	
	if m.requestID != "" {
		parts = append(parts, fmt.Sprintf("requestID: %s", m.requestID))
	}
	
	if m.userID != "" {
		parts = append(parts, fmt.Sprintf("userID: %s", m.userID))
	}
	
	parts = append(parts, fmt.Sprintf("locales: %d}", len(m.translations)))
	
	return strings.Join(parts, ", ")
}