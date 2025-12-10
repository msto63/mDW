// File: watch.go
// Title: Locale File Watching Implementation
// Description: Implements file system watching for language files to support
//              hot-reloading and automatic translation updates during development.
// Author: msto63 with Claude Sonnet 4.0
// Version: v0.1.0
// Created: 2025-01-25
// Modified: 2025-01-25
//
// Change History:
// - 2025-01-25 v0.1.0: Initial implementation of locale file watching

package i18n

import (
	"os"
	"path/filepath"
	"strings"
	"text/template"
	"time"

	mdwerror "github.com/msto63/mDW/foundation/core/error"
	"github.com/msto63/mDW/foundation/utils/stringx"
)

// watchedFileInfo stores information about watched files
type watchedFileInfo struct {
	path         string
	locale       string
	lastModified time.Time
}

// startWatching starts monitoring locale files for changes
func (m *Manager) startWatching() error {
	if stringx.IsBlank(m.localesDir) {
		return mdwerror.New("invalid locales directory for watching").WithCode(mdwerror.CodeValidationFailed).WithOperation("i18n.startWatching").WithDetail("directory", m.localesDir)
	}

	// Simple polling-based watcher (can be enhanced with fsnotify later)
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	// Track file modification times
	watchedFiles := make(map[string]watchedFileInfo)
	
	// Initial scan
	if err := m.scanLocaleFiles(watchedFiles); err != nil {
		return err
	}

	for range ticker.C {
		if !m.watching {
			break
		}

		// Check for file changes
		if err := m.checkFileChanges(watchedFiles); err != nil {
			// Log error but continue watching
			continue
		}
	}

	return nil
}

// scanLocaleFiles scans the locales directory and tracks all locale files
func (m *Manager) scanLocaleFiles(watchedFiles map[string]watchedFileInfo) error {
	entries, err := os.ReadDir(m.localesDir)
	if err != nil {
		return mdwerror.Wrap(err, "failed to scan locale files").WithCode(mdwerror.CodeInvalidOperation).WithOperation("i18n.scanLocaleFiles")
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		fileName := entry.Name()
		
		// Check if file has a supported extension
		ext := strings.ToLower(filepath.Ext(fileName))
		if ext != ".toml" && ext != ".yaml" && ext != ".yml" {
			continue
		}

		// Extract locale from filename
		locale := strings.TrimSuffix(fileName, ext)
		if stringx.IsBlank(locale) {
			continue
		}

		filePath := filepath.Join(m.localesDir, fileName)
		
		// Get file modification time
		fileInfo, err := os.Stat(filePath)
		if err != nil {
			continue
		}

		watchedFiles[filePath] = watchedFileInfo{
			path:         filePath,
			locale:       locale,
			lastModified: fileInfo.ModTime(),
		}
	}

	return nil
}

// checkFileChanges checks for changes in watched locale files
func (m *Manager) checkFileChanges(watchedFiles map[string]watchedFileInfo) error {
	// Check existing files for modifications
	for filePath, info := range watchedFiles {
		fileInfo, err := os.Stat(filePath)
		if err != nil {
			// File might have been deleted
			delete(watchedFiles, filePath)
			m.handleFileDeleted(info.locale)
			continue
		}

		// Check if file was modified
		if fileInfo.ModTime().After(info.lastModified) {
			// File was modified - reload locale
			if err := m.reloadLocale(info.locale); err != nil {
				// Log error but continue
				continue
			}

			// Update modification time
			info.lastModified = fileInfo.ModTime()
			watchedFiles[filePath] = info
		}
	}

	// Check for new files
	if err := m.scanForNewFiles(watchedFiles); err != nil {
		return err
	}

	return nil
}

// scanForNewFiles scans for newly added locale files
func (m *Manager) scanForNewFiles(watchedFiles map[string]watchedFileInfo) error {
	entries, err := os.ReadDir(m.localesDir)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		fileName := entry.Name()
		filePath := filepath.Join(m.localesDir, fileName)
		
		// Skip if already watching
		if _, exists := watchedFiles[filePath]; exists {
			continue
		}

		// Check if file has a supported extension
		ext := strings.ToLower(filepath.Ext(fileName))
		if ext != ".toml" && ext != ".yaml" && ext != ".yml" {
			continue
		}

		// Extract locale from filename
		locale := strings.TrimSuffix(fileName, ext)
		if stringx.IsBlank(locale) {
			continue
		}

		// Get file modification time
		fileInfo, err := os.Stat(filePath)
		if err != nil {
			continue
		}

		// Add to watched files
		watchedFiles[filePath] = watchedFileInfo{
			path:         filePath,
			locale:       locale,
			lastModified: fileInfo.ModTime(),
		}

		// Load the new locale
		if err := m.reloadLocale(locale); err != nil {
			// Log error but continue
			continue
		}
	}

	return nil
}

// reloadLocale reloads translations for a specific locale and notifies watchers
func (m *Manager) reloadLocale(locale string) error {
	// Load the updated locale first
	m.mu.RLock()
	m.mu.RUnlock()

	// Load the updated locale
	if err := m.loadLocale(locale); err != nil {
		return mdwerror.Wrap(err, "failed to reload locale").WithCode(mdwerror.CodeInvalidOperation).WithOperation("i18n.reloadLocale").WithDetail("locale", locale)
	}

	// Get new translations
	m.mu.RLock()
	newTranslations := m.deepCopyTranslations(m.translations[locale])
	
	// Get watchers (copy to avoid holding lock during callbacks)
	watchers := make([]LocaleChangeHandler, len(m.watchers))
	copy(watchers, m.watchers)
	m.mu.RUnlock()

	// Clear template cache for this locale (templates might have changed)
	m.clearTemplateCache(locale)

	// Notify all watchers
	for _, handler := range watchers {
		if handler != nil {
			go handler(locale, newTranslations)
		}
	}

	return nil
}

// handleFileDeleted handles the deletion of a locale file
func (m *Manager) handleFileDeleted(locale string) {
	m.mu.Lock()
	
	// Remove translations for the deleted locale
	delete(m.translations, locale)
	
	// Clear template cache for this locale
	m.clearTemplateCache(locale)
	
	// Get watchers
	watchers := make([]LocaleChangeHandler, len(m.watchers))
	copy(watchers, m.watchers)
	
	m.mu.Unlock()

	// Notify watchers about locale removal
	for _, handler := range watchers {
		if handler != nil {
			go handler(locale, nil) // nil indicates locale was removed
		}
	}
}

// clearTemplateCache clears cached templates that might reference a specific locale
func (m *Manager) clearTemplateCache(locale string) {
	// Clear all templates (simple approach - could be optimized to only clear
	// templates that are related to the specific locale)
	m.templates = make(map[string]*template.Template)
}

// deepCopyTranslations creates a deep copy of translation data
func (m *Manager) deepCopyTranslations(src map[string]interface{}) map[string]interface{} {
	if src == nil {
		return nil
	}

	dst := make(map[string]interface{})
	
	for k, v := range src {
		switch val := v.(type) {
		case map[string]interface{}:
			dst[k] = m.deepCopyTranslations(val)
		case []interface{}:
			dst[k] = append([]interface{}(nil), val...)
		default:
			dst[k] = v
		}
	}
	
	return dst
}

// StopWatching stops file monitoring
func (m *Manager) StopWatching() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.watching = false
}

// IsWatching returns whether file monitoring is active
func (m *Manager) IsWatching() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.watching
}

// ReloadAll reloads all locale files
func (m *Manager) ReloadAll() error {
	return m.loadAllLocales()
}