// ============================================================================
// meinDENKWERK (mDW) - Lokale KI-Plattform
// ============================================================================
//
// Package:     audio
// Description: Audio capture using PortAudio
// Author:      Mike Stoffels with Claude
// Created:     2025-12-07
// License:     MIT
// ============================================================================

package audio

import (
	"context"
	"fmt"
	"sync"

	"github.com/gordonklaus/portaudio"
)

const (
	// DefaultSampleRate is the default sample rate for audio capture (16kHz for Whisper)
	DefaultSampleRate = 16000

	// DefaultFramesPerBuffer is the default buffer size
	DefaultFramesPerBuffer = 512

	// DefaultChannels is mono audio
	DefaultChannels = 1
)

// Capture handles audio input from microphone
type Capture struct {
	mu          sync.RWMutex
	stream      *portaudio.Stream
	sampleRate  float64
	bufferSize  int
	channels    int
	deviceName  string
	running     bool
	outputChan  chan []float32
	initialized bool
}

// CaptureConfig holds configuration for audio capture
type CaptureConfig struct {
	SampleRate float64
	BufferSize int
	Channels   int
	DeviceName string // Name of the input device (empty = default)
}

// DefaultCaptureConfig returns default capture configuration
func DefaultCaptureConfig() CaptureConfig {
	return CaptureConfig{
		SampleRate: DefaultSampleRate,
		BufferSize: DefaultFramesPerBuffer,
		Channels:   DefaultChannels,
	}
}

// NewCapture creates a new audio capture instance
func NewCapture(cfg CaptureConfig) (*Capture, error) {
	if err := portaudio.Initialize(); err != nil {
		return nil, fmt.Errorf("failed to initialize PortAudio: %w", err)
	}

	return &Capture{
		sampleRate:  cfg.SampleRate,
		bufferSize:  cfg.BufferSize,
		channels:    cfg.Channels,
		deviceName:  cfg.DeviceName,
		outputChan:  make(chan []float32, 100),
		initialized: true,
	}, nil
}

// Start begins audio capture
func (c *Capture) Start(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.running {
		return fmt.Errorf("capture already running")
	}

	// Create buffer for audio samples
	buffer := make([]float32, c.bufferSize)

	var stream *portaudio.Stream
	var err error

	// Check if a specific device is requested
	if c.deviceName != "" && c.deviceName != "default" {
		// Find the device by name
		device, findErr := c.findDeviceByName(c.deviceName)
		if findErr != nil {
			// Fall back to default if device not found
			stream, err = portaudio.OpenDefaultStream(
				c.channels, // input channels
				0,          // output channels (none)
				c.sampleRate,
				c.bufferSize,
				buffer,
			)
		} else {
			// Open stream with specific device
			streamParams := portaudio.StreamParameters{
				Input: portaudio.StreamDeviceParameters{
					Device:   device,
					Channels: c.channels,
					Latency:  device.DefaultLowInputLatency,
				},
				SampleRate:      c.sampleRate,
				FramesPerBuffer: c.bufferSize,
			}
			stream, err = portaudio.OpenStream(streamParams, buffer)
		}
	} else {
		// Open default input stream
		stream, err = portaudio.OpenDefaultStream(
			c.channels, // input channels
			0,          // output channels (none)
			c.sampleRate,
			c.bufferSize,
			buffer,
		)
	}

	if err != nil {
		return fmt.Errorf("failed to open audio stream: %w", err)
	}

	c.stream = stream

	// Start the stream
	if err := stream.Start(); err != nil {
		stream.Close()
		return fmt.Errorf("failed to start audio stream: %w", err)
	}

	c.running = true

	// Start capture goroutine
	go c.captureLoop(ctx, buffer)

	return nil
}

// findDeviceByName finds a PortAudio device by name
func (c *Capture) findDeviceByName(name string) (*portaudio.DeviceInfo, error) {
	devices, err := portaudio.Devices()
	if err != nil {
		return nil, err
	}

	for _, dev := range devices {
		if dev.Name == name && dev.MaxInputChannels > 0 {
			return dev, nil
		}
	}

	return nil, fmt.Errorf("device not found: %s", name)
}

// captureLoop continuously reads audio from the stream
func (c *Capture) captureLoop(ctx context.Context, buffer []float32) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			c.mu.RLock()
			if !c.running || c.stream == nil {
				c.mu.RUnlock()
				return
			}
			stream := c.stream
			c.mu.RUnlock()

			// Read audio data
			err := stream.Read()
			if err != nil {
				// Check if we're still supposed to be running
				c.mu.RLock()
				stillRunning := c.running
				c.mu.RUnlock()
				if !stillRunning {
					return
				}
				continue
			}

			// Copy buffer and send to channel
			samples := make([]float32, len(buffer))
			copy(samples, buffer)

			select {
			case c.outputChan <- samples:
			default:
				// Channel full, skip this buffer
			}
		}
	}
}

// Stop stops audio capture
func (c *Capture) Stop() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.running {
		return nil
	}

	c.running = false

	if c.stream != nil {
		if err := c.stream.Stop(); err != nil {
			// Log but don't fail
		}
		if err := c.stream.Close(); err != nil {
			return fmt.Errorf("failed to close audio stream: %w", err)
		}
		c.stream = nil
	}

	return nil
}

// Close cleans up resources
func (c *Capture) Close() error {
	if err := c.Stop(); err != nil {
		return err
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if c.initialized {
		if err := portaudio.Terminate(); err != nil {
			return fmt.Errorf("failed to terminate PortAudio: %w", err)
		}
		c.initialized = false
	}

	close(c.outputChan)
	return nil
}

// Output returns the channel that receives audio samples
func (c *Capture) Output() <-chan []float32 {
	return c.outputChan
}

// IsRunning returns whether capture is currently running
func (c *Capture) IsRunning() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.running
}

// SampleRate returns the sample rate
func (c *Capture) SampleRate() float64 {
	return c.sampleRate
}

// BufferSize returns the buffer size
func (c *Capture) BufferSize() int {
	return c.bufferSize
}

// SetDeviceName sets the device name for future captures
func (c *Capture) SetDeviceName(name string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.deviceName = name
}

// ListInputDevices returns a list of available input devices
func ListInputDevices() ([]DeviceInfo, error) {
	if err := portaudio.Initialize(); err != nil {
		return nil, fmt.Errorf("failed to initialize PortAudio: %w", err)
	}
	defer portaudio.Terminate()

	devices, err := portaudio.Devices()
	if err != nil {
		return nil, fmt.Errorf("failed to get devices: %w", err)
	}

	defaultInput, _ := portaudio.DefaultInputDevice()
	var defaultInputName string
	if defaultInput != nil {
		defaultInputName = defaultInput.Name
	}

	var inputDevices []DeviceInfo
	for _, dev := range devices {
		if dev.MaxInputChannels > 0 {
			inputDevices = append(inputDevices, DeviceInfo{
				Name:              dev.Name,
				MaxInputChannels:  dev.MaxInputChannels,
				DefaultSampleRate: dev.DefaultSampleRate,
				IsDefault:         dev.Name == defaultInputName,
			})
		}
	}

	return inputDevices, nil
}

// DeviceInfo holds information about an audio device
type DeviceInfo struct {
	Name              string
	MaxInputChannels  int
	DefaultSampleRate float64
	IsDefault         bool
}
