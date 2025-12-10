// ============================================================================
// meinDENKWERK (mDW) - Lokale KI-Plattform
// ============================================================================
//
// Package:     voiceassistant
// Description: Settings persistence for Voice Assistant
// Author:      Mike Stoffels with Claude
// Created:     2025-12-07
// License:     MIT
// ============================================================================

package voiceassistant

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// SettingsFile holds persistent settings
type SettingsFile struct {
	Model             string  `json:"model"`
	OllamaURL         string  `json:"ollama_url"`
	UseDirect         bool    `json:"use_direct"`
	MDWAPIURL         string  `json:"mdw_api_url"`
	Language          string  `json:"language"`
	WhisperModel      string  `json:"whisper_model"`
	SilenceDurationMs int     `json:"silence_duration_ms"`
	TTSEnabled        bool    `json:"tts_enabled"`
	TTSVoice          string  `json:"tts_voice"`
	TTSRate           int     `json:"tts_rate"`
	VADThreshold      float32 `json:"vad_threshold"`

	// v0.3.0 fields
	DialogMode        bool   `json:"dialog_mode"`
	DialogTimeout     int    `json:"dialog_timeout"`
	InputDevice       string `json:"input_device"`
	WakeWordEnabled   bool   `json:"wakeword_enabled"`
	WakeWord          string `json:"wakeword"`
	StreamingSTT      bool   `json:"streaming_stt"`
	StreamingInterval int    `json:"streaming_interval"`

	// v0.3.2 fields (Voxtral)
	STTEngine    string `json:"stt_engine"`
	VoxtralURL   string `json:"voxtral_url"`
	VoxtralModel string `json:"voxtral_model"`
}

// getSettingsPath returns the path to the settings file
func getSettingsPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	// Create config directory if it doesn't exist
	configDir := filepath.Join(homeDir, ".config", "mdw")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return "", err
	}

	return filepath.Join(configDir, "voice-assistant.json"), nil
}

// saveSettingsToFile saves the current settings to a file
func (a *App) saveSettingsToFile() error {
	path, err := getSettingsPath()
	if err != nil {
		return err
	}

	settings := SettingsFile{
		Model:             a.config.MDWModel,
		OllamaURL:         a.config.OllamaURL,
		UseDirect:         a.config.UseDirect,
		MDWAPIURL:         a.config.MDWAPIURL,
		Language:          a.config.Language,
		WhisperModel:      a.config.WhisperModel,
		SilenceDurationMs: a.config.SilenceDurationMs,
		TTSEnabled:        a.config.TTSEnabled,
		TTSVoice:          a.config.TTSVoice,
		TTSRate:           a.config.TTSRate,
		VADThreshold:      a.config.VADThreshold,
		// v0.3.0 fields
		DialogMode:        a.config.DialogMode,
		DialogTimeout:     a.config.DialogTimeout,
		InputDevice:       a.config.InputDevice,
		WakeWordEnabled:   a.config.WakeWordEnabled,
		WakeWord:          a.config.WakeWord,
		StreamingSTT:      a.config.StreamingSTT,
		StreamingInterval: a.config.StreamingInterval,
		// v0.3.2 fields
		STTEngine:    a.config.STTEngine,
		VoxtralURL:   a.config.VoxtralURL,
		VoxtralModel: a.config.VoxtralModel,
	}

	data, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}

// LoadSettingsFromFile loads settings from a file and applies them to the config
func LoadSettingsFromFile(cfg *Config) error {
	path, err := getSettingsPath()
	if err != nil {
		return err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // No settings file yet, use defaults
		}
		return err
	}

	var settings SettingsFile
	if err := json.Unmarshal(data, &settings); err != nil {
		return err
	}

	// Apply loaded settings
	if settings.Model != "" {
		cfg.MDWModel = settings.Model
	}
	if settings.OllamaURL != "" {
		cfg.OllamaURL = settings.OllamaURL
	}
	cfg.UseDirect = settings.UseDirect
	if settings.MDWAPIURL != "" {
		cfg.MDWAPIURL = settings.MDWAPIURL
	}
	if settings.Language != "" {
		cfg.Language = settings.Language
	}
	if settings.WhisperModel != "" {
		cfg.WhisperModel = settings.WhisperModel
	}
	if settings.SilenceDurationMs > 0 {
		cfg.SilenceDurationMs = settings.SilenceDurationMs
	}
	cfg.TTSEnabled = settings.TTSEnabled
	if settings.TTSVoice != "" {
		cfg.TTSVoice = settings.TTSVoice
	}
	if settings.TTSRate > 0 {
		cfg.TTSRate = settings.TTSRate
	}
	if settings.VADThreshold > 0 {
		cfg.VADThreshold = settings.VADThreshold
	}

	// v0.3.0 fields
	cfg.DialogMode = settings.DialogMode
	if settings.DialogTimeout > 0 {
		cfg.DialogTimeout = settings.DialogTimeout
	}
	if settings.InputDevice != "" {
		cfg.InputDevice = settings.InputDevice
	}
	cfg.WakeWordEnabled = settings.WakeWordEnabled
	if settings.WakeWord != "" {
		cfg.WakeWord = settings.WakeWord
	}
	cfg.StreamingSTT = settings.StreamingSTT
	if settings.StreamingInterval > 0 {
		cfg.StreamingInterval = settings.StreamingInterval
	}

	// v0.3.2 fields (Voxtral)
	if settings.STTEngine != "" {
		cfg.STTEngine = settings.STTEngine
	}
	if settings.VoxtralURL != "" {
		cfg.VoxtralURL = settings.VoxtralURL
	}
	if settings.VoxtralModel != "" {
		cfg.VoxtralModel = settings.VoxtralModel
	}

	return nil
}
