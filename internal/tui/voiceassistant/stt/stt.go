// ============================================================================
// meinDENKWERK (mDW) - Lokale KI-Plattform
// ============================================================================
//
// Package:     stt
// Description: Speech-to-Text interface
// Author:      Mike Stoffels with Claude
// Created:     2025-12-07
// License:     MIT
// ============================================================================

package stt

import (
	"context"
)

// Transcriber is the interface for speech-to-text engines
type Transcriber interface {
	// Transcribe converts audio samples to text
	Transcribe(ctx context.Context, samples []float32) (Result, error)

	// TranscribeFile transcribes audio from a file
	TranscribeFile(ctx context.Context, path string) (Result, error)

	// Close releases resources
	Close() error
}

// Result holds the transcription result
type Result struct {
	// Text is the transcribed text
	Text string

	// Language is the detected language
	Language string

	// Confidence is the confidence score (0-1)
	Confidence float32

	// Segments are the individual segments with timestamps
	Segments []Segment

	// Duration is the audio duration in seconds
	Duration float32
}

// Segment is a transcription segment with timing
type Segment struct {
	// Text is the segment text
	Text string

	// Start is the start time in seconds
	Start float32

	// End is the end time in seconds
	End float32

	// Confidence is the segment confidence
	Confidence float32
}

// Config holds STT configuration
type Config struct {
	// ModelPath is the path to the model file
	ModelPath string

	// Language is the target language (e.g., "de", "en", "auto")
	Language string

	// SampleRate is the expected audio sample rate
	SampleRate int

	// Translate enables translation to English
	Translate bool

	// NumThreads is the number of threads to use
	NumThreads int
}

// DefaultConfig returns default STT configuration
func DefaultConfig() Config {
	return Config{
		Language:   "de",
		SampleRate: 16000,
		Translate:  false,
		NumThreads: 4,
	}
}
