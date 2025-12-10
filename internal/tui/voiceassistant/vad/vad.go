// ============================================================================
// meinDENKWERK (mDW) - Lokale KI-Plattform
// ============================================================================
//
// Package:     vad
// Description: Voice Activity Detection interface
// Author:      Mike Stoffels with Claude
// Created:     2025-12-07
// License:     MIT
// ============================================================================

package vad

import (
	"time"
)

// Detector is the interface for voice activity detection
type Detector interface {
	// Process processes audio samples and returns whether speech is detected
	Process(samples []float32) (bool, error)

	// ProcessInt16 processes 16-bit integer samples
	ProcessInt16(samples []int16) (bool, error)

	// Close releases resources
	Close() error
}

// Config holds VAD configuration
type Config struct {
	// SampleRate is the audio sample rate (typically 8000, 16000, 32000, or 48000)
	SampleRate int

	// Mode/Aggressiveness (0-3 for WebRTC VAD, higher = more aggressive filtering)
	Mode int

	// SilenceDuration is how long silence must last to end speech
	SilenceDuration time.Duration

	// MinSpeechDuration is the minimum speech duration to be considered valid
	MinSpeechDuration time.Duration
}

// DefaultConfig returns default VAD configuration
func DefaultConfig() Config {
	return Config{
		SampleRate:        16000,
		Mode:              2, // Moderate aggressiveness
		SilenceDuration:   3 * time.Second,
		MinSpeechDuration: 500 * time.Millisecond,
	}
}

// SpeechState tracks the state of speech detection
type SpeechState struct {
	// IsSpeaking indicates if speech is currently detected
	IsSpeaking bool

	// SpeechStartTime is when speech started
	SpeechStartTime time.Time

	// LastSpeechTime is when speech was last detected
	LastSpeechTime time.Time

	// SilenceDuration is the current silence duration
	SilenceDuration time.Duration

	// SpeechDuration is the total speech duration
	SpeechDuration time.Duration
}

// SpeechTracker tracks speech state over time
type SpeechTracker struct {
	config         Config
	state          SpeechState
	speechActive   bool
	speechStarted  bool
	silenceStart   time.Time
}

// NewSpeechTracker creates a new speech tracker
func NewSpeechTracker(cfg Config) *SpeechTracker {
	return &SpeechTracker{
		config: cfg,
	}
}

// Update updates the speech state based on VAD result
func (t *SpeechTracker) Update(isSpeech bool) SpeechState {
	now := time.Now()

	if isSpeech {
		if !t.speechStarted {
			// Speech just started
			t.speechStarted = true
			t.state.SpeechStartTime = now
			t.state.IsSpeaking = true
		}

		t.state.LastSpeechTime = now
		t.state.SilenceDuration = 0
		t.silenceStart = time.Time{}

		// Update speech duration
		t.state.SpeechDuration = now.Sub(t.state.SpeechStartTime)
	} else {
		if t.speechStarted {
			// Was speaking, now silence
			if t.silenceStart.IsZero() {
				t.silenceStart = now
			}
			t.state.SilenceDuration = now.Sub(t.silenceStart)
		}

		// Check if silence duration exceeds threshold
		if t.state.SilenceDuration >= t.config.SilenceDuration && t.speechStarted {
			t.state.IsSpeaking = false
		}
	}

	return t.state
}

// ShouldEndRecording returns true if recording should end (silence threshold reached)
func (t *SpeechTracker) ShouldEndRecording() bool {
	return t.speechStarted &&
		   t.state.SilenceDuration >= t.config.SilenceDuration &&
		   t.state.SpeechDuration >= t.config.MinSpeechDuration
}

// IsValidSpeech returns true if enough speech has been captured
func (t *SpeechTracker) IsValidSpeech() bool {
	return t.state.SpeechDuration >= t.config.MinSpeechDuration
}

// Reset resets the tracker state
func (t *SpeechTracker) Reset() {
	t.state = SpeechState{}
	t.speechStarted = false
	t.silenceStart = time.Time{}
}

// State returns the current speech state
func (t *SpeechTracker) State() SpeechState {
	return t.state
}

// SetSilenceDuration updates the silence duration threshold
func (t *SpeechTracker) SetSilenceDuration(d time.Duration) {
	t.config.SilenceDuration = d
}

// SilenceDuration returns the current silence duration threshold
func (t *SpeechTracker) SilenceDuration() time.Duration {
	return t.config.SilenceDuration
}
