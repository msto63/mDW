// ============================================================================
// meinDENKWERK (mDW) - Lokale KI-Plattform
// ============================================================================
//
// Package:     audio
// Description: Audio buffer utilities
// Author:      Mike Stoffels with Claude
// Created:     2025-12-07
// License:     MIT
// ============================================================================

package audio

import (
	"sync"
)

// RingBuffer is a thread-safe ring buffer for audio samples
type RingBuffer struct {
	mu       sync.RWMutex
	data     []float32
	size     int
	writePos int
	readPos  int
	count    int
}

// NewRingBuffer creates a new ring buffer with the specified capacity
func NewRingBuffer(capacity int) *RingBuffer {
	return &RingBuffer{
		data: make([]float32, capacity),
		size: capacity,
	}
}

// Write writes samples to the buffer
func (rb *RingBuffer) Write(samples []float32) int {
	rb.mu.Lock()
	defer rb.mu.Unlock()

	written := 0
	for _, s := range samples {
		rb.data[rb.writePos] = s
		rb.writePos = (rb.writePos + 1) % rb.size
		written++

		if rb.count < rb.size {
			rb.count++
		} else {
			// Overwrite oldest data
			rb.readPos = (rb.readPos + 1) % rb.size
		}
	}

	return written
}

// Read reads up to n samples from the buffer
func (rb *RingBuffer) Read(n int) []float32 {
	rb.mu.Lock()
	defer rb.mu.Unlock()

	if n > rb.count {
		n = rb.count
	}

	samples := make([]float32, n)
	for i := 0; i < n; i++ {
		samples[i] = rb.data[rb.readPos]
		rb.readPos = (rb.readPos + 1) % rb.size
		rb.count--
	}

	return samples
}

// Peek reads n samples without removing them
func (rb *RingBuffer) Peek(n int) []float32 {
	rb.mu.RLock()
	defer rb.mu.RUnlock()

	if n > rb.count {
		n = rb.count
	}

	samples := make([]float32, n)
	pos := rb.readPos
	for i := 0; i < n; i++ {
		samples[i] = rb.data[pos]
		pos = (pos + 1) % rb.size
	}

	return samples
}

// ReadAll reads all samples from the buffer
func (rb *RingBuffer) ReadAll() []float32 {
	rb.mu.Lock()
	defer rb.mu.Unlock()

	samples := make([]float32, rb.count)
	for i := 0; i < rb.count; i++ {
		samples[i] = rb.data[rb.readPos]
		rb.readPos = (rb.readPos + 1) % rb.size
	}
	rb.count = 0

	return samples
}

// GetAll returns all samples without removing them (ordered from oldest to newest)
func (rb *RingBuffer) GetAll() []float32 {
	rb.mu.RLock()
	defer rb.mu.RUnlock()

	samples := make([]float32, rb.count)
	pos := rb.readPos
	for i := 0; i < rb.count; i++ {
		samples[i] = rb.data[pos]
		pos = (pos + 1) % rb.size
	}

	return samples
}

// Len returns the number of samples in the buffer
func (rb *RingBuffer) Len() int {
	rb.mu.RLock()
	defer rb.mu.RUnlock()
	return rb.count
}

// Cap returns the capacity of the buffer
func (rb *RingBuffer) Cap() int {
	return rb.size
}

// Clear clears the buffer
func (rb *RingBuffer) Clear() {
	rb.mu.Lock()
	defer rb.mu.Unlock()
	rb.readPos = 0
	rb.writePos = 0
	rb.count = 0
}

// IsFull returns whether the buffer is full
func (rb *RingBuffer) IsFull() bool {
	rb.mu.RLock()
	defer rb.mu.RUnlock()
	return rb.count == rb.size
}

// IsEmpty returns whether the buffer is empty
func (rb *RingBuffer) IsEmpty() bool {
	rb.mu.RLock()
	defer rb.mu.RUnlock()
	return rb.count == 0
}

// AudioBuffer is a growing buffer for collecting audio samples
type AudioBuffer struct {
	mu      sync.RWMutex
	samples []float32
}

// NewAudioBuffer creates a new audio buffer
func NewAudioBuffer() *AudioBuffer {
	return &AudioBuffer{
		samples: make([]float32, 0, 16000*10), // Pre-allocate for ~10 seconds at 16kHz
	}
}

// Append adds samples to the buffer
func (ab *AudioBuffer) Append(samples []float32) {
	ab.mu.Lock()
	defer ab.mu.Unlock()
	ab.samples = append(ab.samples, samples...)
}

// Get returns all samples
func (ab *AudioBuffer) Get() []float32 {
	ab.mu.RLock()
	defer ab.mu.RUnlock()
	result := make([]float32, len(ab.samples))
	copy(result, ab.samples)
	return result
}

// Len returns the number of samples
func (ab *AudioBuffer) Len() int {
	ab.mu.RLock()
	defer ab.mu.RUnlock()
	return len(ab.samples)
}

// DurationSeconds returns the duration in seconds at the given sample rate
func (ab *AudioBuffer) DurationSeconds(sampleRate float64) float64 {
	ab.mu.RLock()
	defer ab.mu.RUnlock()
	return float64(len(ab.samples)) / sampleRate
}

// Clear clears the buffer
func (ab *AudioBuffer) Clear() {
	ab.mu.Lock()
	defer ab.mu.Unlock()
	ab.samples = ab.samples[:0]
}

// Reset resets the buffer with a new capacity hint
func (ab *AudioBuffer) Reset(capacityHint int) {
	ab.mu.Lock()
	defer ab.mu.Unlock()
	ab.samples = make([]float32, 0, capacityHint)
}

// TrimToSize removes oldest samples to keep the buffer at the specified size
func (ab *AudioBuffer) TrimToSize(maxSamples int) {
	ab.mu.Lock()
	defer ab.mu.Unlock()
	if len(ab.samples) > maxSamples {
		// Keep only the newest samples
		ab.samples = ab.samples[len(ab.samples)-maxSamples:]
	}
}
