// ============================================================================
// meinDENKWERK (mDW) - Lokale KI-Plattform
// ============================================================================
//
// Package:     voiceassistant
// Description: Voice Assistant - Configuration
// Author:      Mike Stoffels with Claude
// Created:     2025-12-07
// License:     MIT
// ============================================================================

package voiceassistant

import (
	"os"
	"path/filepath"
	"runtime"
)

// Config holds the voice assistant configuration
type Config struct {
	// General
	Language string
	LogLevel string

	// Activation
	ActivationMode      string // "shortcut", "wakeword", "both"
	Shortcut            string
	WakeWord            string
	WakeWordSensitivity float32
	WakeWordEnabled     bool

	// Audio
	InputDevice  string
	OutputDevice string
	SampleRate   int
	BufferSize   int

	// VAD
	VADEngine           string // "webrtc", "silero"
	VADThreshold        float32
	SilenceDurationMs   int
	MinSpeechDurationMs int

	// STT
	STTEngine         string // "whisper", "voxtral"
	WhisperModel      string // "tiny", "base", "small", "medium"
	WhisperModelPath  string
	StreamingSTT      bool // Enable real-time transcription during speech
	StreamingInterval int  // Interval in ms between streaming transcriptions (default: 2000)

	// Voxtral (alternative STT via vLLM)
	VoxtralURL   string // vLLM server URL (e.g., "http://localhost:8100")
	VoxtralModel string // Model name (e.g., "mistralai/Voxtral-Mini-3B-2507")

	// TTS
	TTSEnabled     bool
	TTSEngine      string // "piper", "macos"
	TTSVoice       string
	TTSRate        int
	PiperBinary    string
	PiperModelPath string

	// mDW / Ollama
	MDWAPIURL       string
	MDWWebSocketURL string
	OllamaURL       string
	MDWModel        string
	TimeoutSeconds  int
	UseDirect       bool // Use direct Ollama connection instead of mDW backend

	// UI
	PopupWidth          int
	PopupHeight         int
	NotificationSounds  bool

	// Dialog Mode
	DialogMode          bool // Auto-continue conversation after TTS
	DialogTimeout       int  // Timeout in seconds to wait for next speech (0 = infinite)
}

// DefaultConfig returns the default configuration
func DefaultConfig() Config {
	homeDir, _ := os.UserHomeDir()
	toolsDir := filepath.Join(homeDir, "Projects", "mDW", "tools")

	return Config{
		// General
		Language: "de",
		LogLevel: "info",

		// Activation
		ActivationMode:      "shortcut",
		Shortcut:            "ctrl+shift+m",
		WakeWord:            "mein denkwerk",
		WakeWordSensitivity: 0.5,
		WakeWordEnabled:     false, // Wake word detection disabled by default (high CPU)

		// Audio
		InputDevice:  "default",
		OutputDevice: "default",
		SampleRate:   16000,
		BufferSize:   512,

		// VAD
		VADEngine:           "webrtc",
		VADThreshold:        0.5,
		SilenceDurationMs:   3000,
		MinSpeechDurationMs: 500,

		// STT
		STTEngine:         "whisper",
		WhisperModel:      "medium",
		WhisperModelPath:  filepath.Join(toolsDir, "whisper", "ggml-medium.bin"),
		StreamingSTT:      false, // Disabled by default (higher CPU usage)
		StreamingInterval: 2000,  // Transcribe every 2 seconds

		// Voxtral
		VoxtralURL:   "http://localhost:8100",
		VoxtralModel: "mistralai/Voxtral-Mini-3B-2507",

		// TTS
		TTSEnabled:     true,
		TTSEngine:      "macos", // Use macOS say as default (Piper has issues on ARM)
		TTSVoice:       "Anna",
		TTSRate:        220,
		PiperBinary:    filepath.Join(toolsDir, "piper", "piper", "piper"),
		PiperModelPath: filepath.Join(toolsDir, "piper", "models", "de_DE-thorsten-high.onnx"),

		// mDW / Ollama
		MDWAPIURL:       "http://localhost:8080",
		MDWWebSocketURL: "ws://localhost:8080/api/v1/chat/ws",
		OllamaURL:       "http://localhost:11434",
		MDWModel:        "mistral:7b",
		TimeoutSeconds:  120,
		UseDirect:       true, // Direct Ollama connection by default

		// UI
		PopupWidth:         400,
		PopupHeight:        300,
		NotificationSounds: true,

		// Dialog Mode
		DialogMode:    false, // Disabled by default
		DialogTimeout: 10,    // Wait 10 seconds for next speech
	}
}

// GetShortcutModifiers returns the platform-specific modifier description
func (c *Config) GetShortcutDescription() string {
	if runtime.GOOS == "darwin" {
		return "Cmd+Shift+M"
	}
	return "Ctrl+Shift+M"
}
