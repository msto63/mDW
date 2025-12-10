// File: discovery.go
// Title: Configuration File Discovery Implementation
// Description: Implements automatic configuration file discovery across
//              multiple paths and formats for flexible deployment scenarios.
// Author: msto63 with Claude Sonnet 4.0
// Version: v0.1.0
// Created: 2025-01-25
// Modified: 2025-01-25
//
// Change History:
// - 2025-01-25 v0.1.0: Initial implementation of file discovery

package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	mdwerror "github.com/msto63/mDW/foundation/core/error"
)

// DiscoveryOptions defines options for automatic configuration file discovery
type DiscoveryOptions struct {
	Paths      []string // Directories to search for config files
	Filenames  []string // Base filenames to look for (without extension)
	Extensions []string // File extensions to try (.toml, .yaml, .yml)
	EnvPrefix  string   // Environment variable prefix for overrides
	Required   bool     // Whether finding a config file is required
}

// DefaultDiscoveryOptions returns sensible default options for config discovery
func DefaultDiscoveryOptions() DiscoveryOptions {
	return DiscoveryOptions{
		Paths:      []string{".", "./config", "/etc", "/usr/local/etc"},
		Filenames:  []string{"config", "app"},
		Extensions: []string{".toml", ".yaml", ".yml"},
		EnvPrefix:  "",
		Required:   true,
	}
}

// Discover automatically discovers and loads configuration files
func Discover(options DiscoveryOptions) (*Config, error) {
	// Use defaults if options are empty
	if len(options.Paths) == 0 {
		options.Paths = []string{"."}
	}
	if len(options.Filenames) == 0 {
		options.Filenames = []string{"config"}
	}
	if len(options.Extensions) == 0 {
		options.Extensions = []string{".toml", ".yaml", ".yml"}
	}

	// Try to find a configuration file
	for _, path := range options.Paths {
		for _, filename := range options.Filenames {
			for _, ext := range options.Extensions {
				configPath := filepath.Join(path, filename+ext)
				
				// Check if file exists and is readable
				if info, err := os.Stat(configPath); err == nil && !info.IsDir() {
					// Found a config file, try to load it
					loadOptions := LoadOptions{
						Format:    FormatAuto,
						EnvPrefix: options.EnvPrefix,
						Watch:     false,
					}
					
					config, err := LoadWithOptions(configPath, loadOptions)
					if err != nil {
						// File exists but couldn't load - this is an error
						return nil, mdwerror.Wrap(err, fmt.Sprintf("found config file %s but failed to load", configPath)).
							WithCode(mdwerror.CodeInvalidOperation).
							WithOperation("config.Discover").
							WithDetail("configPath", configPath)
					}
					
					return config, nil
				}
			}
		}
	}

	// No configuration file found
	if options.Required {
		searchPaths := make([]string, 0, len(options.Paths)*len(options.Filenames)*len(options.Extensions))
		for _, path := range options.Paths {
			for _, filename := range options.Filenames {
				for _, ext := range options.Extensions {
					searchPaths = append(searchPaths, filepath.Join(path, filename+ext))
				}
			}
		}
		
		return nil, mdwerror.New(fmt.Sprintf("no configuration file found in paths: %s", strings.Join(searchPaths, ", "))).
			WithCode(mdwerror.CodeNotFound).
			WithOperation("config.Discover").
			WithDetail("searchPaths", searchPaths)
	}

	// Create empty configuration if not required
	return &Config{
		data:     make(map[string]interface{}),
		format:   FormatTOML,
		watchers: make([]ChangeHandler, 0),
		watching: false,
	}, nil
}

// DiscoverWithDefaults discovers configuration with default options
func DiscoverWithDefaults() (*Config, error) {
	return Discover(DefaultDiscoveryOptions())
}

// FindConfigFile searches for a configuration file without loading it
func FindConfigFile(options DiscoveryOptions) (string, error) {
	for _, path := range options.Paths {
		for _, filename := range options.Filenames {
			for _, ext := range options.Extensions {
				configPath := filepath.Join(path, filename+ext)
				
				if info, err := os.Stat(configPath); err == nil && !info.IsDir() {
					return configPath, nil
				}
			}
		}
	}

	return "", mdwerror.New("configuration file not found").
		WithCode(mdwerror.CodeNotFound).
		WithOperation("config.FindConfigFile")
}

// ListPossibleConfigFiles returns a list of all possible configuration file paths
func ListPossibleConfigFiles(options DiscoveryOptions) []string {
	var paths []string
	
	for _, path := range options.Paths {
		for _, filename := range options.Filenames {
			for _, ext := range options.Extensions {
				configPath := filepath.Join(path, filename+ext)
				paths = append(paths, configPath)
			}
		}
	}
	
	return paths
}

// LoadWithWatch loads configuration with file watching enabled
func LoadWithWatch(filePath string) (*Config, error) {
	return LoadWithOptions(filePath, LoadOptions{
		Format: FormatAuto,
		Watch:  true,
	})
}

// LoadFromEnv loads configuration entirely from environment variables
func LoadFromEnv(envPrefix string) *Config {
	data := make(map[string]interface{})
	
	// Get all environment variables with the prefix
	for _, env := range os.Environ() {
		parts := strings.SplitN(env, "=", 2)
		if len(parts) != 2 {
			continue
		}
		
		key, value := parts[0], parts[1]
		
		// Check if it matches our prefix
		if envPrefix != "" {
			prefix := strings.ToUpper(envPrefix) + "_"
			if !strings.HasPrefix(key, prefix) {
				continue
			}
			key = strings.TrimPrefix(key, prefix)
		}
		
		// Convert environment variable name to config key
		// MYAPP_DATABASE_HOST -> database.host
		configKey := strings.ToLower(strings.ReplaceAll(key, "_", "."))
		
		// Try to parse as different types
		if parsed := parseEnvValue(value); parsed != nil {
			setNestedValue(data, configKey, parsed)
		} else {
			setNestedValue(data, configKey, value)
		}
	}
	
	return &Config{
		data:      data,
		format:    FormatAuto,
		envPrefix: envPrefix,
		watchers:  make([]ChangeHandler, 0),
		watching:  false,
	}
}

// parseEnvValue attempts to parse environment variable values as appropriate types
func parseEnvValue(value string) interface{} {
	// Try boolean
	if value == "true" || value == "false" {
		return value == "true"
	}
	
	// Try integer
	if intVal, err := strconv.Atoi(value); err == nil {
		return intVal
	}
	
	// Try float
	if floatVal, err := strconv.ParseFloat(value, 64); err == nil {
		return floatVal
	}
	
	// Return as string
	return value
}

// setNestedValue sets a nested value in a map using dot notation
func setNestedValue(data map[string]interface{}, key string, value interface{}) {
	keys := strings.Split(key, ".")
	current := data
	
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