//go:build voice
// +build voice

// ============================================================================
// meinDENKWERK (mDW) - Lokale KI-Plattform
// ============================================================================
//
// Package:     cmd
// Description: CLI command for mDW Voice Assistant
// Author:      Mike Stoffels with Claude
// Created:     2025-12-07
// License:     MIT
// ============================================================================

package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/msto63/mDW/internal/tui/voiceassistant"
	"github.com/spf13/cobra"
)

var (
	vaModel         string
	vaLanguage      string
	vaWhisperModel  string
	vaDisableTTS    bool
	vaAPIURL        string
	vaOllamaURL     string
	vaSilenceMs     int
	vaUseMDW        bool
)

var voiceAssistantCmd = &cobra.Command{
	Use:     "voice",
	Aliases: []string{"va", "voice-assistant"},
	Short:   "Startet den mDW Voice Assistant",
	Long: `Startet den mDW Voice Assistant als Menüleisten-Anwendung.

Der Voice Assistant ermöglicht Sprachinteraktion mit Ollama/LLMs:

  - Aktivierung via Tastenkürzel (Cmd+Shift+M auf macOS, Ctrl+Shift+M sonst)
  - Automatische Erkennung von Sprechpausen (VAD)
  - Lokale Spracherkennung mit Whisper
  - Optionale Sprachausgabe mit Piper TTS
  - Streaming-Antworten direkt von Ollama

Standard: Direkte Verbindung zu Ollama (kein mDW-Backend nötig)

Voraussetzungen:
  - Ollama muss laufen (ollama serve)
  - PortAudio installiert (brew install portaudio)
  - Whisper-Modell in tools/whisper/
  - Piper in tools/piper/ (optional, für TTS)

Beispiele:
  mdw voice                       # Direkt mit Ollama
  mdw voice --model qwen2.5:7b    # Mit bestimmtem Modell
  mdw voice --no-tts              # Ohne Sprachausgabe
  mdw voice --use-mdw             # Über mDW-Backend (erfordert mdw serve)`,
	RunE: runVoiceAssistant,
}

func init() {
	rootCmd.AddCommand(voiceAssistantCmd)

	voiceAssistantCmd.Flags().StringVarP(&vaModel, "model", "m", "mistral:7b",
		"LLM-Modell für den Chat")
	voiceAssistantCmd.Flags().StringVarP(&vaLanguage, "language", "l", "de",
		"Sprache für STT (de, en, auto)")
	voiceAssistantCmd.Flags().StringVar(&vaWhisperModel, "whisper-model", "base",
		"Whisper-Modell (tiny, base, small, medium)")
	voiceAssistantCmd.Flags().BoolVar(&vaDisableTTS, "no-tts", false,
		"Sprachausgabe deaktivieren")
	voiceAssistantCmd.Flags().StringVar(&vaOllamaURL, "ollama-url", "http://localhost:11434",
		"Ollama API URL")
	voiceAssistantCmd.Flags().StringVar(&vaAPIURL, "api-url", "http://localhost:8080",
		"mDW API URL (nur mit --use-mdw)")
	voiceAssistantCmd.Flags().BoolVar(&vaUseMDW, "use-mdw", false,
		"mDW-Backend verwenden statt direkter Ollama-Verbindung")
	voiceAssistantCmd.Flags().IntVar(&vaSilenceMs, "silence-ms", 3000,
		"Millisekunden Stille bis Aufnahme endet")
}

func runVoiceAssistant(cmd *cobra.Command, args []string) error {
	// Get home directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	// Build paths
	toolsDir := filepath.Join(homeDir, "Projects", "mDW", "tools")
	whisperModelPath := filepath.Join(toolsDir, "whisper", fmt.Sprintf("ggml-%s.bin", vaWhisperModel))
	piperBinary := filepath.Join(toolsDir, "piper", "piper", "piper")
	piperModelPath := filepath.Join(toolsDir, "piper", "models", "de_DE-thorsten-high.onnx")

	// Check if whisper model exists
	if _, err := os.Stat(whisperModelPath); os.IsNotExist(err) {
		fmt.Printf("Whisper-Modell nicht gefunden: %s\n", whisperModelPath)
		fmt.Println("Bitte laden Sie das Modell herunter:")
		fmt.Printf("  curl -L https://huggingface.co/ggerganov/whisper.cpp/resolve/main/ggml-%s.bin -o %s\n",
			vaWhisperModel, whisperModelPath)
		return fmt.Errorf("whisper model not found")
	}

	// Build config with defaults
	cfg := voiceassistant.Config{
		Language:          vaLanguage,
		LogLevel:          "info",
		ActivationMode:    "shortcut",
		SampleRate:        16000,
		BufferSize:        512,
		VADEngine:         "webrtc",
		VADThreshold:      0.5,
		SilenceDurationMs: vaSilenceMs,
		MinSpeechDurationMs: 500,
		STTEngine:         "whisper",
		WhisperModel:      vaWhisperModel,
		WhisperModelPath:  whisperModelPath,
		TTSEnabled:        !vaDisableTTS,
		TTSEngine:         "piper",
		TTSVoice:          "de_DE-thorsten-high",
		PiperBinary:       piperBinary,
		PiperModelPath:    piperModelPath,
		MDWAPIURL:         vaAPIURL,
		MDWWebSocketURL:   vaAPIURL + "/api/v1/chat/ws",
		OllamaURL:         vaOllamaURL,
		MDWModel:          vaModel,
		TimeoutSeconds:    120,
		UseDirect:         !vaUseMDW, // Default: direct Ollama connection
		PopupWidth:         450,
		PopupHeight:        350,
		NotificationSounds: true,
	}

	// Load saved settings (overrides defaults but not CLI flags)
	if err := voiceassistant.LoadSettingsFromFile(&cfg); err != nil {
		fmt.Printf("Hinweis: Einstellungen konnten nicht geladen werden: %v\n", err)
	}

	// CLI flags override saved settings
	if cmd.Flags().Changed("model") {
		cfg.MDWModel = vaModel
	}
	if cmd.Flags().Changed("language") {
		cfg.Language = vaLanguage
	}
	if cmd.Flags().Changed("ollama-url") {
		cfg.OllamaURL = vaOllamaURL
	}
	if cmd.Flags().Changed("use-mdw") {
		cfg.UseDirect = !vaUseMDW
	}
	if cmd.Flags().Changed("no-tts") {
		cfg.TTSEnabled = !vaDisableTTS
	}
	if cmd.Flags().Changed("silence-ms") {
		cfg.SilenceDurationMs = vaSilenceMs
	}

	// Print startup info
	backendInfo := fmt.Sprintf("Ollama direkt (%s)", vaOllamaURL)
	if vaUseMDW {
		backendInfo = fmt.Sprintf("mDW Backend (%s)", vaAPIURL)
	}

	fmt.Println("mDW Voice Assistant")
	fmt.Println("==================")
	fmt.Printf("Modell:      %s\n", vaModel)
	fmt.Printf("Backend:     %s\n", backendInfo)
	fmt.Printf("Sprache:     %s\n", vaLanguage)
	fmt.Printf("Whisper:     %s\n", vaWhisperModel)
	fmt.Printf("TTS:         %v\n", !vaDisableTTS)

	if runtime.GOOS == "darwin" {
		fmt.Println("Aktivierung: Klick auf Menüleisten-Icon")
		fmt.Println()
		fmt.Println("Der Voice Assistant läuft jetzt in der Menüleiste.")
		fmt.Println("Klicken Sie auf das Icon und wählen Sie 'Aktivieren' für eine Sprachaufnahme.")
	} else {
		fmt.Println("Aktivierung: Ctrl+Shift+M")
		fmt.Println()
		fmt.Println("Der Voice Assistant läuft jetzt in der Menüleiste.")
		fmt.Println("Drücken Sie Ctrl+Shift+M um eine Sprachaufnahme zu starten.")
	}
	fmt.Println()

	// Create and run app
	app, err := voiceassistant.New(cfg)
	if err != nil {
		return fmt.Errorf("failed to create voice assistant: %w", err)
	}

	return app.Run()
}
