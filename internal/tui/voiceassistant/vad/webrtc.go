// ============================================================================
// meinDENKWERK (mDW) - Lokale KI-Plattform
// ============================================================================
//
// Package:     vad
// Description: WebRTC VAD implementation
// Author:      Mike Stoffels with Claude
// Created:     2025-12-07
// License:     MIT
// ============================================================================

package vad

import (
	"fmt"

	webrtcvad "github.com/maxhawkins/go-webrtcvad"
)

// WebRTCVAD implements voice activity detection using WebRTC's VAD
type WebRTCVAD struct {
	vad        *webrtcvad.VAD
	sampleRate int
	mode       int
}

// NewWebRTCVAD creates a new WebRTC VAD instance
func NewWebRTCVAD(cfg Config) (*WebRTCVAD, error) {
	vad, err := webrtcvad.New()
	if err != nil {
		return nil, fmt.Errorf("failed to create WebRTC VAD: %w", err)
	}

	// Set aggressiveness mode (0-3)
	mode := cfg.Mode
	if mode < 0 {
		mode = 0
	}
	if mode > 3 {
		mode = 3
	}

	if err := vad.SetMode(mode); err != nil {
		return nil, fmt.Errorf("failed to set VAD mode: %w", err)
	}

	// Validate sample rate
	validRates := []int{8000, 16000, 32000, 48000}
	validRate := false
	for _, r := range validRates {
		if cfg.SampleRate == r {
			validRate = true
			break
		}
	}
	if !validRate {
		return nil, fmt.Errorf("invalid sample rate %d, must be one of %v", cfg.SampleRate, validRates)
	}

	return &WebRTCVAD{
		vad:        vad,
		sampleRate: cfg.SampleRate,
		mode:       mode,
	}, nil
}

// Process processes float32 audio samples and returns whether speech is detected
func (w *WebRTCVAD) Process(samples []float32) (bool, error) {
	// Convert float32 to int16
	int16Samples := make([]int16, len(samples))
	for i, s := range samples {
		// Clamp to valid range
		if s > 1.0 {
			s = 1.0
		}
		if s < -1.0 {
			s = -1.0
		}
		int16Samples[i] = int16(s * 32767)
	}

	return w.ProcessInt16(int16Samples)
}

// ProcessInt16 processes 16-bit integer samples
func (w *WebRTCVAD) ProcessInt16(samples []int16) (bool, error) {
	// WebRTC VAD requires specific frame sizes:
	// 10ms, 20ms, or 30ms at the sample rate
	// For 16kHz: 160, 320, or 480 samples
	frameSize := w.getFrameSize()

	if len(samples) < frameSize {
		// Pad with zeros if too short
		padded := make([]int16, frameSize)
		copy(padded, samples)
		samples = padded
	}

	// Process in frames and return true if any frame has speech
	for i := 0; i+frameSize <= len(samples); i += frameSize {
		frame := samples[i : i+frameSize]

		// Convert to bytes (little-endian)
		frameBytes := int16ToBytes(frame)

		active, err := w.vad.Process(w.sampleRate, frameBytes)
		if err != nil {
			return false, fmt.Errorf("VAD processing failed: %w", err)
		}

		if active {
			return true, nil
		}
	}

	return false, nil
}

// getFrameSize returns the frame size for 10ms at the configured sample rate
func (w *WebRTCVAD) getFrameSize() int {
	// 10ms frame size
	return w.sampleRate / 100
}

// int16ToBytes converts int16 slice to bytes (little-endian)
func int16ToBytes(samples []int16) []byte {
	bytes := make([]byte, len(samples)*2)
	for i, s := range samples {
		bytes[i*2] = byte(s)
		bytes[i*2+1] = byte(s >> 8)
	}
	return bytes
}

// Close releases resources
func (w *WebRTCVAD) Close() error {
	// WebRTC VAD doesn't require explicit cleanup
	return nil
}

// SetMode sets the VAD aggressiveness mode (0-3)
func (w *WebRTCVAD) SetMode(mode int) error {
	if mode < 0 || mode > 3 {
		return fmt.Errorf("mode must be between 0 and 3")
	}
	if err := w.vad.SetMode(mode); err != nil {
		return fmt.Errorf("failed to set mode: %w", err)
	}
	w.mode = mode
	return nil
}

// Mode returns the current aggressiveness mode
func (w *WebRTCVAD) Mode() int {
	return w.mode
}

// SampleRate returns the sample rate
func (w *WebRTCVAD) SampleRate() int {
	return w.sampleRate
}
