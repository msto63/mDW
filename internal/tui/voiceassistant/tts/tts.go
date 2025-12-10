// ============================================================================
// meinDENKWERK (mDW) - Lokale KI-Plattform
// ============================================================================
//
// Package:     tts
// Description: Text-to-Speech interface
// Author:      Mike Stoffels with Claude
// Created:     2025-12-07
// License:     MIT
// ============================================================================

package tts

import (
	"context"
)

// Synthesizer is the interface for text-to-speech engines
type Synthesizer interface {
	// Synthesize converts text to audio
	Synthesize(ctx context.Context, text string) ([]byte, error)

	// SynthesizeToFile converts text to audio and saves to a file
	SynthesizeToFile(ctx context.Context, text, path string) error

	// SampleRate returns the output sample rate
	SampleRate() int

	// Close releases resources
	Close() error
}

// Config holds TTS configuration
type Config struct {
	// Voice is the voice name/ID to use
	Voice string

	// ModelPath is the path to the model file
	ModelPath string

	// BinaryPath is the path to the TTS binary (for CLI-based engines)
	BinaryPath string

	// SampleRate is the output sample rate
	SampleRate int

	// Speed is the speech speed (1.0 = normal)
	Speed float32
}

// DefaultConfig returns default TTS configuration
func DefaultConfig() Config {
	return Config{
		Voice:      "de_DE-thorsten-high",
		SampleRate: 22050,
		Speed:      1.0,
	}
}
