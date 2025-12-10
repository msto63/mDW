// ============================================================================
// meinDENKWERK (mDW) - Lokale KI-Plattform
// ============================================================================
//
// Package:     tts
// Description: Piper TTS implementation
// Author:      Mike Stoffels with Claude
// Created:     2025-12-07
// License:     MIT
// ============================================================================

package tts

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// PiperTTS implements text-to-speech using Piper
type PiperTTS struct {
	binaryPath  string
	modelPath   string
	configPath  string
	sampleRate  int
	espeakData  string
}

// NewPiperTTS creates a new Piper TTS synthesizer
func NewPiperTTS(cfg Config) (*PiperTTS, error) {
	// Verify binary exists
	if cfg.BinaryPath == "" {
		return nil, fmt.Errorf("piper binary path is required")
	}
	if _, err := os.Stat(cfg.BinaryPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("piper binary not found: %s", cfg.BinaryPath)
	}

	// Verify model exists
	if cfg.ModelPath == "" {
		return nil, fmt.Errorf("model path is required")
	}
	if _, err := os.Stat(cfg.ModelPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("model file not found: %s", cfg.ModelPath)
	}

	// Config file should be next to model
	configPath := cfg.ModelPath + ".json"
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("model config not found: %s", configPath)
	}

	// Find espeak-ng-data directory (relative to binary)
	espeakData := filepath.Join(filepath.Dir(cfg.BinaryPath), "espeak-ng-data")
	if _, err := os.Stat(espeakData); os.IsNotExist(err) {
		espeakData = "" // Will use default
	}

	return &PiperTTS{
		binaryPath:  cfg.BinaryPath,
		modelPath:   cfg.ModelPath,
		configPath:  configPath,
		sampleRate:  cfg.SampleRate,
		espeakData:  espeakData,
	}, nil
}

// Synthesize converts text to audio (raw PCM 16-bit signed)
func (p *PiperTTS) Synthesize(ctx context.Context, text string) ([]byte, error) {
	// Build command
	args := []string{
		"--model", p.modelPath,
		"--config", p.configPath,
		"--output_raw",
	}

	if p.espeakData != "" {
		args = append(args, "--espeak_data", p.espeakData)
	}

	cmd := exec.CommandContext(ctx, p.binaryPath, args...)
	cmd.Stdin = strings.NewReader(text)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Set working directory to piper directory for library paths
	cmd.Dir = filepath.Dir(p.binaryPath)

	// Set library path for macOS
	cmd.Env = append(os.Environ(),
		fmt.Sprintf("DYLD_LIBRARY_PATH=%s", filepath.Dir(p.binaryPath)),
	)

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("piper failed: %w, stderr: %s", err, stderr.String())
	}

	return stdout.Bytes(), nil
}

// SynthesizeToFile converts text to audio and saves to a WAV file
func (p *PiperTTS) SynthesizeToFile(ctx context.Context, text, path string) error {
	args := []string{
		"--model", p.modelPath,
		"--config", p.configPath,
		"--output_file", path,
	}

	if p.espeakData != "" {
		args = append(args, "--espeak_data", p.espeakData)
	}

	cmd := exec.CommandContext(ctx, p.binaryPath, args...)
	cmd.Stdin = strings.NewReader(text)

	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	cmd.Dir = filepath.Dir(p.binaryPath)
	cmd.Env = append(os.Environ(),
		fmt.Sprintf("DYLD_LIBRARY_PATH=%s", filepath.Dir(p.binaryPath)),
	)

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("piper failed: %w, stderr: %s", err, stderr.String())
	}

	return nil
}

// SampleRate returns the output sample rate
func (p *PiperTTS) SampleRate() int {
	if p.sampleRate > 0 {
		return p.sampleRate
	}
	return 22050 // Piper default
}

// Close releases resources
func (p *PiperTTS) Close() error {
	return nil
}

// GetAvailableVoices returns a list of available voices
// This would require parsing the piper voices directory
func GetAvailableVoices(voicesDir string) ([]VoiceInfo, error) {
	entries, err := os.ReadDir(voicesDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read voices directory: %w", err)
	}

	var voices []VoiceInfo
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if strings.HasSuffix(entry.Name(), ".onnx") {
			name := strings.TrimSuffix(entry.Name(), ".onnx")
			voices = append(voices, VoiceInfo{
				Name:      name,
				ModelPath: filepath.Join(voicesDir, entry.Name()),
			})
		}
	}

	return voices, nil
}

// VoiceInfo holds information about a TTS voice
type VoiceInfo struct {
	Name        string
	Language    string
	Description string
	ModelPath   string
	SampleRate  int
}
