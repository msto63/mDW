// File: config.go
// Title: Core Configuration Management Implementation
// Description: Implements the main Config type and core functionality for
//              loading, parsing, and accessing configuration data from TOML
//              and YAML files with environment variable support.
// Author: msto63 with Claude Sonnet 4.0
// Version: v0.1.0
// Created: 2025-01-25
// Modified: 2025-01-25
//
// Change History:
// - 2025-01-25 v0.1.0: Initial implementation with TOML/YAML support

package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/BurntSushi/toml"
	"gopkg.in/yaml.v3"

	mdwerror "github.com/msto63/mDW/foundation/core/error"
	mdwstringx "github.com/msto63/mDW/foundation/utils/stringx"
)

// Format represents the configuration file format
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

// Config represents a configuration instance with thread-safe access
type Config struct {
	mu           sync.RWMutex
	data         map[string]interface{}
	filePath     string
	format       Format
	envPrefix    string
	watchers     []ChangeHandler
	watching     bool
	lastModified time.Time
	
	// Context information for better error reporting and tracing
	requestID    string
	userID       string
	correlationID string
	
	// Performance optimizations
	envCache     map[string]string // Cache for environment variable lookups
	envCacheMu   sync.RWMutex     // Separate mutex for env cache
	cacheExpiry  time.Time        // Cache expiration time
	cacheTimeout time.Duration    // Cache timeout duration (default 5 minutes)
	pathCache    map[string][]string // Cache for dot notation paths
	pathCacheMu  sync.RWMutex        // Separate mutex for path cache
}

// ChangeHandler is called when configuration changes are detected
type ChangeHandler func(oldConfig, newConfig *Config)

// LoadOptions defines options for loading configuration
type LoadOptions struct {
	Format    Format            // File format (default: auto-detect)
	EnvPrefix string            // Environment variable prefix (default: none)
	Defaults  map[string]interface{} // Default values
	Watch     bool              // Enable file watching (default: false)
}

// ValidationRule defines validation criteria for configuration values
type ValidationRule struct {
	Required bool        // Whether the field is required
	Type     string      // Expected type: "string", "int", "bool", "float", "duration", "[]string", etc.
	Min      interface{} // Minimum value (for numbers) or length (for strings/slices)
	Max      interface{} // Maximum value (for numbers) or length (for strings/slices)
	Default  interface{} // Default value if not present
	Pattern  string      // Regex pattern for string validation
}

// ValidationRules maps configuration keys to their validation rules
type ValidationRules map[string]ValidationRule

// Load loads configuration from a file with default options
func Load(filePath string) (*Config, error) {
	return LoadWithOptions(filePath, LoadOptions{
		Format: FormatAuto,
	})
}

// LoadWithOptions loads configuration from a file with custom options
func LoadWithOptions(filePath string, options LoadOptions) (*Config, error) {
	if mdwstringx.IsBlank(filePath) {
		err := mdwerror.New("config file path cannot be empty").
			WithCode(mdwerror.CodeValidationFailed).
			WithOperation("config.LoadWithOptions")
		if options.EnvPrefix != "" {
			err = err.WithDetail("envPrefix", options.EnvPrefix)
		}
		return nil, err
	}

	// Check if file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		err := mdwerror.New(fmt.Sprintf("config file not found: %s", filePath)).
			WithCode(mdwerror.CodeNotFound).
			WithOperation("config.LoadWithOptions").
			WithDetail("filePath", filePath)
		if options.EnvPrefix != "" {
			err = err.WithDetail("envPrefix", options.EnvPrefix)
		}
		return nil, err
	}

	// Determine format
	format := options.Format
	if format == FormatAuto {
		format = detectFormat(filePath)
	}
	if format == FormatAuto {
		// If still auto after detection, default to TOML
		format = FormatTOML
	}

	// Read file content
	content, err := os.ReadFile(filePath)
	if err != nil {
		returnErr := mdwerror.Wrap(err, "failed to read config file").
			WithCode(mdwerror.CodeConfigError).
			WithOperation("config.LoadWithOptions").
			WithDetail("filePath", filePath)
		if options.EnvPrefix != "" {
			returnErr = returnErr.WithDetail("envPrefix", options.EnvPrefix)
		}
		return nil, returnErr
	}

	// Parse content
	data, err := parseContent(content, format)
	if err != nil {
		returnErr := mdwerror.Wrap(err, "failed to parse config file").
			WithCode(mdwerror.CodeInvalidInput).
			WithOperation("config.LoadWithOptions").
			WithDetail("filePath", filePath).
			WithDetail("format", format.String())
		if options.EnvPrefix != "" {
			returnErr = returnErr.WithDetail("envPrefix", options.EnvPrefix)
		}
		return nil, returnErr
	}

	// Apply defaults
	if options.Defaults != nil {
		data = mergeDefaults(data, options.Defaults)
	}

	// Get file modification time
	fileInfo, _ := os.Stat(filePath)
	lastModified := time.Time{}
	if fileInfo != nil {
		lastModified = fileInfo.ModTime()
	}

	config := &Config{
		data:         data,
		filePath:     filePath,
		format:       format,
		envPrefix:    options.EnvPrefix,
		watchers:     make([]ChangeHandler, 0),
		watching:     options.Watch,
		lastModified: lastModified,
		envCache:     make(map[string]string),
		cacheTimeout: 5 * time.Minute, // Default cache timeout
		pathCache:    make(map[string][]string),
	}

	// Start watching if requested
	if options.Watch {
		go config.startWatching()
	}

	return config, nil
}

// LoadFromString loads configuration from a string with specified format
func LoadFromString(content string, format Format) (*Config, error) {
	if format == FormatAuto {
		format = FormatTOML // Default to TOML
	}

	data, err := parseContent([]byte(content), format)
	if err != nil {
		return nil, mdwerror.Wrap(err, "failed to parse config from string").
			WithCode(mdwerror.CodeInvalidInput).
			WithOperation("config.LoadFromString").
			WithDetail("format", format.String())
	}

	return &Config{
		data:         data,
		format:       format,
		watchers:     make([]ChangeHandler, 0),
		watching:     false,
		envCache:     make(map[string]string),
		cacheTimeout: 5 * time.Minute,
		pathCache:    make(map[string][]string),
	}, nil
}

// detectFormat determines the configuration format from file extension
func detectFormat(filePath string) Format {
	ext := strings.ToLower(filepath.Ext(filePath))
	switch ext {
	case ".yaml", ".yml":
		return FormatYAML
	case ".toml":
		return FormatTOML
	default:
		return FormatTOML // Default to TOML
	}
}

// parseContent parses configuration content based on format
func parseContent(content []byte, format Format) (map[string]interface{}, error) {
	var data map[string]interface{}
	
	switch format {
	case FormatTOML:
		if err := toml.Unmarshal(content, &data); err != nil {
			return nil, mdwerror.Wrap(err, "TOML parse error").
				WithCode(mdwerror.CodeInvalidInput).
				WithOperation("config.parseContent")
		}
	case FormatYAML:
		if err := yaml.Unmarshal(content, &data); err != nil {
			return nil, mdwerror.Wrap(err, "YAML parse error").
				WithCode(mdwerror.CodeInvalidInput).
				WithOperation("config.parseContent")
		}
	default:
		return nil, mdwerror.New(fmt.Sprintf("unsupported format: %s", format)).
			WithCode(mdwerror.CodeInvalidInput).
			WithOperation("config.parseContent").
			WithDetail("format", format.String())
	}
	
	return data, nil
}

// mergeDefaults merges default values into configuration data
func mergeDefaults(data, defaults map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})
	
	// Copy defaults first
	for k, v := range defaults {
		result[k] = v
	}
	
	// Override with actual data
	for k, v := range data {
		result[k] = v
	}
	
	return result
}

// GetString returns a string configuration value with optional default
func (c *Config) GetString(key string, defaultValue ...string) string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	// Check environment variable first
	if envValue := c.getEnvValue(key); envValue != "" {
		return envValue
	}
	
	value := c.getValue(key)
	if value == nil {
		if len(defaultValue) > 0 {
			return defaultValue[0]
		}
		return ""
	}
	
	switch v := value.(type) {
	case string:
		return v
	case fmt.Stringer:
		return v.String()
	default:
		return fmt.Sprintf("%v", v)
	}
}

// GetInt returns an integer configuration value with optional default
func (c *Config) GetInt(key string, defaultValue ...int) int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	// Check environment variable first
	if envValue := c.getEnvValue(key); envValue != "" {
		if intVal, err := strconv.Atoi(envValue); err == nil {
			return intVal
		}
	}
	
	value := c.getValue(key)
	if value == nil {
		if len(defaultValue) > 0 {
			return defaultValue[0]
		}
		return 0
	}
	
	switch v := value.(type) {
	case int:
		return v
	case int64:
		return int(v)
	case float64:
		return int(v)
	case string:
		if intVal, err := strconv.Atoi(v); err == nil {
			return intVal
		}
	}
	
	if len(defaultValue) > 0 {
		return defaultValue[0]
	}
	return 0
}

// GetBool returns a boolean configuration value with optional default
func (c *Config) GetBool(key string, defaultValue ...bool) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	// Check environment variable first
	if envValue := c.getEnvValue(key); envValue != "" {
		if boolVal, err := strconv.ParseBool(envValue); err == nil {
			return boolVal
		}
	}
	
	value := c.getValue(key)
	if value == nil {
		if len(defaultValue) > 0 {
			return defaultValue[0]
		}
		return false
	}
	
	switch v := value.(type) {
	case bool:
		return v
	case string:
		if boolVal, err := strconv.ParseBool(v); err == nil {
			return boolVal
		}
	}
	
	if len(defaultValue) > 0 {
		return defaultValue[0]
	}
	return false
}

// GetFloat returns a float64 configuration value with optional default
func (c *Config) GetFloat(key string, defaultValue ...float64) float64 {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	// Check environment variable first
	if envValue := c.getEnvValue(key); envValue != "" {
		if floatVal, err := strconv.ParseFloat(envValue, 64); err == nil {
			return floatVal
		}
	}
	
	value := c.getValue(key)
	if value == nil {
		if len(defaultValue) > 0 {
			return defaultValue[0]
		}
		return 0.0
	}
	
	switch v := value.(type) {
	case float64:
		return v
	case float32:
		return float64(v)
	case int:
		return float64(v)
	case int64:
		return float64(v)
	case string:
		if floatVal, err := strconv.ParseFloat(v, 64); err == nil {
			return floatVal
		}
	}
	
	if len(defaultValue) > 0 {
		return defaultValue[0]
	}
	return 0.0
}

// GetDuration returns a time.Duration configuration value with optional default
func (c *Config) GetDuration(key string, defaultValue ...time.Duration) time.Duration {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	// Check environment variable first
	if envValue := c.getEnvValue(key); envValue != "" {
		if duration, err := time.ParseDuration(envValue); err == nil {
			return duration
		}
	}
	
	value := c.getValue(key)
	if value == nil {
		if len(defaultValue) > 0 {
			return defaultValue[0]
		}
		return 0
	}
	
	switch v := value.(type) {
	case string:
		if duration, err := time.ParseDuration(v); err == nil {
			return duration
		}
	case time.Duration:
		return v
	case int:
		return time.Duration(v)
	case int64:
		return time.Duration(v)
	}
	
	if len(defaultValue) > 0 {
		return defaultValue[0]
	}
	return 0
}

// GetStringSlice returns a string slice configuration value with optional default
func (c *Config) GetStringSlice(key string, defaultValue ...[]string) []string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	value := c.getValue(key)
	if value == nil {
		if len(defaultValue) > 0 {
			return defaultValue[0]
		}
		return nil
	}
	
	switch v := value.(type) {
	case []string:
		return v
	case []interface{}:
		result := make([]string, len(v))
		for i, item := range v {
			result[i] = fmt.Sprintf("%v", item)
		}
		return result
	case string:
		// Single string -> slice with one element
		return []string{v}
	}
	
	if len(defaultValue) > 0 {
		return defaultValue[0]
	}
	return nil
}

// getValue retrieves a configuration value by key (supports dot notation) with path caching
func (c *Config) getValue(key string) interface{} {
	// Check path cache first
	c.pathCacheMu.RLock()
	keys, cached := c.pathCache[key]
	c.pathCacheMu.RUnlock()
	
	// Parse and cache path if not cached
	if !cached {
		keys = strings.Split(key, ".")
		c.pathCacheMu.Lock()
		if c.pathCache == nil {
			c.pathCache = make(map[string][]string)
		}
		c.pathCache[key] = keys
		c.pathCacheMu.Unlock()
	}
	
	current := c.data
	
	for i, k := range keys {
		if i == len(keys)-1 {
			// Last key - return the value
			return current[k]
		}
		
		// Navigate deeper into the structure
		if next, ok := current[k].(map[string]interface{}); ok {
			current = next
		} else {
			return nil
		}
	}
	
	return nil
}

// getEnvValue retrieves environment variable value for a configuration key with caching
func (c *Config) getEnvValue(key string) string {
	envKey := c.formatEnvKey(key)
	
	// Check cache first
	c.envCacheMu.RLock()
	if time.Now().Before(c.cacheExpiry) {
		if cached, exists := c.envCache[envKey]; exists {
			c.envCacheMu.RUnlock()
			return cached
		}
	}
	c.envCacheMu.RUnlock()
	
	// Cache miss or expired - get value from environment
	value := os.Getenv(envKey)
	
	// Update cache
	c.envCacheMu.Lock()
	if c.envCache == nil {
		c.envCache = make(map[string]string)
	}
	c.envCache[envKey] = value
	c.cacheExpiry = time.Now().Add(c.cacheTimeout)
	c.envCacheMu.Unlock()
	
	return value
}

// formatEnvKey converts a config key to environment variable format
func (c *Config) formatEnvKey(key string) string {
	// Convert key to environment variable format
	// database.host -> DATABASE_HOST (with prefix MYAPP_ -> MYAPP_DATABASE_HOST)
	envKey := strings.ToUpper(strings.ReplaceAll(key, ".", "_"))
	if c.envPrefix != "" {
		envKey = strings.ToUpper(c.envPrefix) + "_" + envKey
	}
	return envKey
}

// Has checks if a configuration key exists
func (c *Config) Has(key string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	return c.getValue(key) != nil
}

// Set sets a configuration value (runtime only, not persisted)
func (c *Config) Set(key string, value interface{}) {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	keys := strings.Split(key, ".")
	current := c.data
	
	for i, k := range keys {
		if i == len(keys)-1 {
			// Last key - set the value
			current[k] = value
			return
		}
		
		// Navigate deeper or create nested maps
		if next, ok := current[k].(map[string]interface{}); ok {
			current = next
		} else {
			next = make(map[string]interface{})
			current[k] = next
			current = next
		}
	}
}

// GetAll returns all configuration data as a map
func (c *Config) GetAll() map[string]interface{} {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	// Return a deep copy to prevent external modifications
	return c.deepCopyMap(c.data)
}

// deepCopyMap creates a deep copy of a map
func (c *Config) deepCopyMap(src map[string]interface{}) map[string]interface{} {
	dst := make(map[string]interface{})
	
	for k, v := range src {
		switch val := v.(type) {
		case map[string]interface{}:
			dst[k] = c.deepCopyMap(val)
		case []interface{}:
			dst[k] = append([]interface{}(nil), val...)
		default:
			dst[k] = v
		}
	}
	
	return dst
}

// FilePath returns the path of the loaded configuration file
func (c *Config) FilePath() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.filePath
}

// Format returns the configuration file format
func (c *Config) Format() Format {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.format
}

// OnChange registers a change handler for configuration updates
func (c *Config) OnChange(handler ChangeHandler) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.watchers = append(c.watchers, handler)
}

// WithRequestID creates a copy of the config with a request ID for tracing
func (c *Config) WithRequestID(requestID string) *Config {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	clone := &Config{
		data:          c.deepCopyMap(c.data),
		filePath:      c.filePath,
		format:        c.format,
		envPrefix:     c.envPrefix,
		watchers:      append([]ChangeHandler(nil), c.watchers...),
		watching:      c.watching,
		lastModified:  c.lastModified,
		requestID:     requestID,
		userID:        c.userID,
		correlationID: c.correlationID,
		envCache:      make(map[string]string),
		cacheTimeout:  c.cacheTimeout,
		pathCache:     make(map[string][]string),
	}
	return clone
}

// WithUserID creates a copy of the config with a user ID for tracing
func (c *Config) WithUserID(userID string) *Config {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	clone := &Config{
		data:          c.deepCopyMap(c.data),
		filePath:      c.filePath,
		format:        c.format,
		envPrefix:     c.envPrefix,
		watchers:      append([]ChangeHandler(nil), c.watchers...),
		watching:      c.watching,
		lastModified:  c.lastModified,
		requestID:     c.requestID,
		userID:        userID,
		correlationID: c.correlationID,
		envCache:      make(map[string]string),
		cacheTimeout:  c.cacheTimeout,
		pathCache:     make(map[string][]string),
	}
	return clone
}

// WithCorrelationID creates a copy of the config with a correlation ID for tracing
func (c *Config) WithCorrelationID(correlationID string) *Config {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	clone := &Config{
		data:          c.deepCopyMap(c.data),
		filePath:      c.filePath,
		format:        c.format,
		envPrefix:     c.envPrefix,
		watchers:      append([]ChangeHandler(nil), c.watchers...),
		watching:      c.watching,
		lastModified:  c.lastModified,
		requestID:     c.requestID,
		userID:        c.userID,
		correlationID: correlationID,
		envCache:      make(map[string]string),
		cacheTimeout:  c.cacheTimeout,
		pathCache:     make(map[string][]string),
	}
	return clone
}

// Convenience methods for shorter access patterns

// S is a short alias for GetString
func (c *Config) S(key string, defaultValue ...string) string {
	return c.GetString(key, defaultValue...)
}

// I is a short alias for GetInt
func (c *Config) I(key string, defaultValue ...int) int {
	return c.GetInt(key, defaultValue...)
}

// B is a short alias for GetBool
func (c *Config) B(key string, defaultValue ...bool) bool {
	return c.GetBool(key, defaultValue...)
}

// F is a short alias for GetFloat
func (c *Config) F(key string, defaultValue ...float64) float64 {
	return c.GetFloat(key, defaultValue...)
}

// D is a short alias for GetDuration
func (c *Config) D(key string, defaultValue ...time.Duration) time.Duration {
	return c.GetDuration(key, defaultValue...)
}

// SS is a short alias for GetStringSlice
func (c *Config) SS(key string, defaultValue ...[]string) []string {
	return c.GetStringSlice(key, defaultValue...)
}

// String provides a readable representation of the configuration
func (c *Config) String() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	parts := []string{
		fmt.Sprintf("Config{format: %s", c.format.String()),
	}
	
	if c.filePath != "" {
		parts = append(parts, fmt.Sprintf("path: %s", c.filePath))
	}
	
	if c.envPrefix != "" {
		parts = append(parts, fmt.Sprintf("envPrefix: %s", c.envPrefix))
	}
	
	if c.watching {
		parts = append(parts, "watching: true")
	}
	
	if c.requestID != "" {
		parts = append(parts, fmt.Sprintf("requestID: %s", c.requestID))
	}
	
	if c.userID != "" {
		parts = append(parts, fmt.Sprintf("userID: %s", c.userID))
	}
	
	parts = append(parts, fmt.Sprintf("keys: %d}", len(c.data)))
	
	return strings.Join(parts, ", ")
}