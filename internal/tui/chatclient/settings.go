// ============================================================================
// meinDENKWERK (mDW) - Lokale KI-Plattform
// ============================================================================
//
// Package:     chatclient
// Description: Settings persistence for ChatClient
// Author:      Mike Stoffels with Claude
// Created:     2025-12-07
// License:     MIT
// ============================================================================

package chatclient

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// Settings holds persistent ChatClient settings
type Settings struct {
	LastModel    string   `json:"last_model"`
	InputHistory []string `json:"input_history,omitempty"`
}

// settingsDir returns the directory for settings files
func settingsDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ".mdw"
	}
	return filepath.Join(home, ".mdw")
}

// settingsFile returns the path to the settings file
func settingsFile() string {
	return filepath.Join(settingsDir(), "chatclient.json")
}

// LoadSettings loads settings from disk
func LoadSettings() (*Settings, error) {
	data, err := os.ReadFile(settingsFile())
	if err != nil {
		if os.IsNotExist(err) {
			return &Settings{}, nil
		}
		return nil, err
	}

	var settings Settings
	if err := json.Unmarshal(data, &settings); err != nil {
		return &Settings{}, nil
	}

	return &settings, nil
}

// SaveSettings saves settings to disk
func SaveSettings(settings *Settings) error {
	dir := settingsDir()
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(settingsFile(), data, 0644)
}

// SaveLastModel saves the last used model
func SaveLastModel(model string) error {
	settings, _ := LoadSettings()
	settings.LastModel = model
	return SaveSettings(settings)
}

// LoadLastModel loads the last used model
func LoadLastModel() string {
	settings, err := LoadSettings()
	if err != nil || settings.LastModel == "" {
		return ""
	}
	return settings.LastModel
}

// SaveInputHistory saves the input history
func SaveInputHistory(history []string) error {
	settings, _ := LoadSettings()
	// Maximal 100 Einträge speichern
	if len(history) > 100 {
		history = history[len(history)-100:]
	}
	settings.InputHistory = history
	return SaveSettings(settings)
}

// LoadInputHistory loads the input history
func LoadInputHistory() []string {
	settings, err := LoadSettings()
	if err != nil || len(settings.InputHistory) == 0 {
		return []string{}
	}
	return settings.InputHistory
}
