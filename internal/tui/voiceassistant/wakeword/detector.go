// ============================================================================
// meinDENKWERK (mDW) - Lokale KI-Plattform
// ============================================================================
//
// Package:     wakeword
// Description: Wake word detection using Whisper-based keyword spotting
// Author:      Mike Stoffels with Claude
// Created:     2025-12-07
// License:     MIT
// ============================================================================

package wakeword

import (
	"context"
	"strings"
	"sync"
	"time"

	"github.com/msto63/mDW/internal/tui/voiceassistant/audio"
	"github.com/msto63/mDW/internal/tui/voiceassistant/stt"
	"github.com/msto63/mDW/pkg/core/logging"
)

// Config holds configuration for the wake word detector
type Config struct {
	Keywords       []string      // Keywords to listen for (e.g., ["mein denkwerk", "hey mDW"])
	Sensitivity    float32       // Sensitivity threshold (0.0-1.0)
	SampleRate     int           // Audio sample rate
	BufferDuration time.Duration // Duration of audio buffer for detection
	CheckInterval  time.Duration // How often to check for wake word
}

// DefaultConfig returns default wake word configuration
func DefaultConfig() Config {
	return Config{
		Keywords:       []string{"mein denkwerk", "hey denkwerk"},
		Sensitivity:    0.5,
		SampleRate:     16000,
		BufferDuration: 2 * time.Second,
		CheckInterval:  500 * time.Millisecond,
	}
}

// Detector implements wake word detection using Whisper
type Detector struct {
	mu          sync.RWMutex
	config      Config
	transcriber stt.Transcriber
	logger      *logging.Logger
	running     bool
	onDetected  func()
}

// NewDetector creates a new wake word detector
func NewDetector(cfg Config, transcriber stt.Transcriber) *Detector {
	return &Detector{
		config:      cfg,
		transcriber: transcriber,
		logger:      logging.New("wakeword"),
	}
}

// SetOnDetected sets the callback for when a wake word is detected
func (d *Detector) SetOnDetected(callback func()) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.onDetected = callback
}

// SetKeywords updates the keywords to listen for
func (d *Detector) SetKeywords(keywords []string) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.config.Keywords = keywords
}

// Start begins wake word detection with the provided audio capture
func (d *Detector) Start(ctx context.Context, capture *audio.Capture) error {
	d.mu.Lock()
	if d.running {
		d.mu.Unlock()
		return nil
	}
	d.running = true
	d.mu.Unlock()

	d.logger.Info("Wake word detection started", "keywords", d.config.Keywords)

	// Calculate buffer size for detection window
	samplesPerBuffer := int(float64(d.config.SampleRate) * d.config.BufferDuration.Seconds())
	audioBuffer := audio.NewAudioBuffer()

	// Start capture if not already running
	if !capture.IsRunning() {
		if err := capture.Start(ctx); err != nil {
			return err
		}
	}

	go d.detectionLoop(ctx, capture, audioBuffer, samplesPerBuffer)

	return nil
}

// Stop stops wake word detection
func (d *Detector) Stop() {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.running = false
	d.logger.Info("Wake word detection stopped")
}

// IsRunning returns whether detection is currently running
func (d *Detector) IsRunning() bool {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.running
}

// detectionLoop continuously listens for wake words
func (d *Detector) detectionLoop(ctx context.Context, capture *audio.Capture, buffer *audio.AudioBuffer, maxSamples int) {
	ticker := time.NewTicker(d.config.CheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			d.Stop()
			return

		case samples, ok := <-capture.Output():
			if !ok {
				d.Stop()
				return
			}

			// Add samples to buffer
			buffer.Append(samples)

			// Keep buffer at max size
			if buffer.Len() > maxSamples {
				buffer.TrimToSize(maxSamples)
			}

		case <-ticker.C:
			d.mu.RLock()
			running := d.running
			d.mu.RUnlock()

			if !running {
				return
			}

			// Check for wake word
			if buffer.Len() >= maxSamples/2 {
				if d.checkForWakeWord(ctx, buffer.Get()) {
					d.mu.RLock()
					callback := d.onDetected
					d.mu.RUnlock()

					if callback != nil {
						d.logger.Info("Wake word detected!")
						callback()
					}

					// Clear buffer after detection
					buffer.Clear()
				}
			}
		}
	}
}

// checkForWakeWord checks if the audio contains a wake word
func (d *Detector) checkForWakeWord(ctx context.Context, samples []float32) bool {
	if len(samples) == 0 {
		return false
	}

	// Use a short timeout for wake word detection
	detectCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	result, err := d.transcriber.Transcribe(detectCtx, samples)
	if err != nil {
		d.logger.Debug("Wake word transcription failed", "error", err)
		return false
	}

	if result.Text == "" {
		return false
	}

	// Check for keywords
	text := strings.ToLower(result.Text)
	d.logger.Debug("Wake word check", "text", text)

	d.mu.RLock()
	keywords := d.config.Keywords
	d.mu.RUnlock()

	for _, keyword := range keywords {
		if strings.Contains(text, strings.ToLower(keyword)) {
			return true
		}
	}

	return false
}
