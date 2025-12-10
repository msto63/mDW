// ============================================================================
// meinDENKWERK (mDW) - Lokale KI-Plattform
// ============================================================================
//
// Package:     audio
// Description: Audio playback using PortAudio
// Author:      Mike Stoffels with Claude
// Created:     2025-12-07
// License:     MIT
// ============================================================================

package audio

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"sync"

	"github.com/gordonklaus/portaudio"
)

// Playback handles audio output to speakers
type Playback struct {
	mu         sync.RWMutex
	sampleRate float64
	channels   int
	playing    bool
}

// PlaybackConfig holds configuration for audio playback
type PlaybackConfig struct {
	SampleRate float64
	Channels   int
}

// DefaultPlaybackConfig returns default playback configuration
func DefaultPlaybackConfig() PlaybackConfig {
	return PlaybackConfig{
		SampleRate: 22050, // Piper default
		Channels:   1,
	}
}

// NewPlayback creates a new audio playback instance
func NewPlayback(cfg PlaybackConfig) *Playback {
	return &Playback{
		sampleRate: cfg.SampleRate,
		channels:   cfg.Channels,
	}
}

// PlayRaw plays raw PCM audio data (16-bit signed integers)
func (p *Playback) PlayRaw(data []byte, sampleRate float64) error {
	p.mu.Lock()
	if p.playing {
		p.mu.Unlock()
		return fmt.Errorf("already playing")
	}
	p.playing = true
	p.mu.Unlock()

	defer func() {
		p.mu.Lock()
		p.playing = false
		p.mu.Unlock()
	}()

	// Convert bytes to int16 samples
	reader := bytes.NewReader(data)
	numSamples := len(data) / 2
	samples := make([]int16, numSamples)

	for i := 0; i < numSamples; i++ {
		if err := binary.Read(reader, binary.LittleEndian, &samples[i]); err != nil {
			if err == io.EOF {
				break
			}
			return fmt.Errorf("failed to read samples: %w", err)
		}
	}

	// Convert to float32 for PortAudio
	floatSamples := make([]float32, len(samples))
	for i, s := range samples {
		floatSamples[i] = float32(s) / 32768.0
	}

	return p.playFloat32(floatSamples, sampleRate)
}

// playFloat32 plays float32 audio samples
func (p *Playback) playFloat32(samples []float32, sampleRate float64) error {
	if err := portaudio.Initialize(); err != nil {
		return fmt.Errorf("failed to initialize PortAudio: %w", err)
	}
	defer portaudio.Terminate()

	bufferSize := 1024
	position := 0

	// Create output buffer
	buffer := make([]float32, bufferSize)

	stream, err := portaudio.OpenDefaultStream(
		0,          // input channels (none)
		p.channels, // output channels
		sampleRate,
		bufferSize,
		&buffer,
	)
	if err != nil {
		return fmt.Errorf("failed to open output stream: %w", err)
	}
	defer stream.Close()

	if err := stream.Start(); err != nil {
		return fmt.Errorf("failed to start output stream: %w", err)
	}
	defer stream.Stop()

	// Play all samples
	for position < len(samples) {
		// Fill buffer
		for i := 0; i < bufferSize; i++ {
			if position+i < len(samples) {
				buffer[i] = samples[position+i]
			} else {
				buffer[i] = 0
			}
		}
		position += bufferSize

		if err := stream.Write(); err != nil {
			return fmt.Errorf("failed to write to stream: %w", err)
		}
	}

	return nil
}

// PlayFile plays a WAV file
func (p *Playback) PlayFile(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	// Parse WAV header to get sample rate
	sampleRate, audioData, err := parseWAV(data)
	if err != nil {
		return fmt.Errorf("failed to parse WAV: %w", err)
	}

	return p.PlayRaw(audioData, sampleRate)
}

// parseWAV parses a WAV file and returns sample rate and audio data
func parseWAV(data []byte) (float64, []byte, error) {
	if len(data) < 44 {
		return 0, nil, fmt.Errorf("file too small to be a valid WAV")
	}

	// Check RIFF header
	if string(data[0:4]) != "RIFF" {
		return 0, nil, fmt.Errorf("not a valid RIFF file")
	}

	// Check WAVE format
	if string(data[8:12]) != "WAVE" {
		return 0, nil, fmt.Errorf("not a valid WAVE file")
	}

	// Find fmt chunk
	pos := 12
	var sampleRate uint32
	var dataStart int
	var dataSize int

	for pos < len(data)-8 {
		chunkID := string(data[pos : pos+4])
		chunkSize := binary.LittleEndian.Uint32(data[pos+4 : pos+8])

		switch chunkID {
		case "fmt ":
			if chunkSize >= 16 {
				sampleRate = binary.LittleEndian.Uint32(data[pos+12 : pos+16])
			}
		case "data":
			dataStart = pos + 8
			dataSize = int(chunkSize)
		}

		pos += 8 + int(chunkSize)
		if pos%2 != 0 {
			pos++ // Word alignment
		}
	}

	if sampleRate == 0 || dataStart == 0 {
		return 0, nil, fmt.Errorf("missing required WAV chunks")
	}

	if dataStart+dataSize > len(data) {
		dataSize = len(data) - dataStart
	}

	return float64(sampleRate), data[dataStart : dataStart+dataSize], nil
}

// IsPlaying returns whether audio is currently playing
func (p *Playback) IsPlaying() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.playing
}
