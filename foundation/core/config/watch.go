// File: watch.go
// Title: Configuration File Watching Implementation
// Description: Implements file system watching for configuration files to
//              support hot-reloading and automatic configuration updates.
// Author: msto63 with Claude Sonnet 4.0
// Version: v0.1.0
// Created: 2025-01-25
// Modified: 2025-01-25
//
// Change History:
// - 2025-01-25 v0.1.0: Initial implementation of file watching

package config

import (
	"os"
	"time"

	mdwerror "github.com/msto63/mDW/foundation/core/error"
	"github.com/msto63/mDW/foundation/utils/stringx"
)

// startWatching starts monitoring the configuration file for changes
func (c *Config) startWatching() error {
	if stringx.IsBlank(c.filePath) {
		return mdwerror.New("file path required for watching").
			WithCode(mdwerror.CodeValidationFailed).
			WithOperation("config.startWatching")
	}

	// Simple polling-based watcher (can be enhanced with fsnotify later)
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		if !c.watching {
			break
		}

		// Check if file was modified
		fileInfo, err := os.Stat(c.filePath)
		if err != nil {
			// File might have been deleted or moved
			continue
		}

		c.mu.RLock()
		lastModified := c.lastModified
		c.mu.RUnlock()

		if fileInfo.ModTime().After(lastModified) {
			// File was modified - reload configuration
			if err := c.reload(); err != nil {
				// Log error but continue watching
				continue
			}
		}
	}

	return nil
}

// reload reloads the configuration from the file and notifies watchers
func (c *Config) reload() error {
	// Read and parse the updated file
	content, err := os.ReadFile(c.filePath)
	if err != nil {
		return mdwerror.Wrap(err, "failed to read config file during reload").
			WithCode(mdwerror.CodeConfigError).
			WithOperation("config.reload").
			WithDetail("filePath", c.filePath)
	}

	newData, err := parseContent(content, c.format)
	if err != nil {
		return mdwerror.Wrap(err, "failed to parse config file during reload").
			WithCode(mdwerror.CodeInvalidInput).
			WithOperation("config.reload").
			WithDetail("filePath", c.filePath).
			WithDetail("format", c.format.String())
	}

	// Create a copy of the old configuration for comparison
	c.mu.Lock()
	oldConfig := &Config{
		data:   c.deepCopyMap(c.data),
		format: c.format,
	}

	// Update the configuration
	c.data = newData
	fileInfo, _ := os.Stat(c.filePath)
	if fileInfo != nil {
		c.lastModified = fileInfo.ModTime()
	}

	// Get watchers (copy to avoid holding lock during callbacks)
	watchers := make([]ChangeHandler, len(c.watchers))
	copy(watchers, c.watchers)
	c.mu.Unlock()

	// Notify all watchers
	newConfig := &Config{
		data:   c.deepCopyMap(c.data),
		format: c.format,
	}

	for _, handler := range watchers {
		if handler != nil {
			go handler(oldConfig, newConfig)
		}
	}

	return nil
}

// StopWatching stops file monitoring
func (c *Config) StopWatching() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.watching = false
}

// IsWatching returns whether file monitoring is active
func (c *Config) IsWatching() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.watching
}