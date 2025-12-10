// ============================================================================
// meinDENKWERK (mDW) - Lokale KI-Plattform
// ============================================================================
//
// Package:     stt
// Description: Whisper STT implementation using whisper.cpp CLI
// Author:      Mike Stoffels with Claude
// Created:     2025-12-07
// License:     MIT
// ============================================================================

package stt

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// WhisperCLI implements speech-to-text using whisper.cpp CLI
type WhisperCLI struct {
	binaryPath string
	modelPath  string
	language   string
	sampleRate int
	tempDir    string
}

// NewWhisperCLI creates a new Whisper CLI transcriber
func NewWhisperCLI(cfg Config) (*WhisperCLI, error) {
	// Find whisper binary
	binaryPath := findWhisperBinary()
	if binaryPath == "" {
		return nil, fmt.Errorf("whisper binary not found")
	}

	// Verify model exists
	if cfg.ModelPath == "" {
		return nil, fmt.Errorf("model path is required")
	}
	if _, err := os.Stat(cfg.ModelPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("model file not found: %s", cfg.ModelPath)
	}

	// Create temp directory
	tempDir, err := os.MkdirTemp("", "whisper-")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp directory: %w", err)
	}

	return &WhisperCLI{
		binaryPath: binaryPath,
		modelPath:  cfg.ModelPath,
		language:   cfg.Language,
		sampleRate: cfg.SampleRate,
		tempDir:    tempDir,
	}, nil
}

// findWhisperBinary finds the whisper binary
func findWhisperBinary() string {
	// Check common locations - try whisper-cli first (Homebrew), then whisper
	locations := []string{
		"/opt/homebrew/bin/whisper-cli",
		"/opt/homebrew/Cellar/whisper-cpp/1.8.2/bin/whisper-cli",
		"/opt/homebrew/bin/whisper",
		"/usr/local/bin/whisper-cli",
		"/usr/local/bin/whisper",
		"/usr/bin/whisper",
	}

	// Also check PATH for whisper-cli first
	if path, err := exec.LookPath("whisper-cli"); err == nil {
		return path
	}
	if path, err := exec.LookPath("whisper"); err == nil {
		return path
	}

	for _, loc := range locations {
		if _, err := os.Stat(loc); err == nil {
			return loc
		}
	}

	return ""
}

// Transcribe converts audio samples to text
func (w *WhisperCLI) Transcribe(ctx context.Context, samples []float32) (Result, error) {
	// Write samples to temp WAV file
	wavPath := filepath.Join(w.tempDir, fmt.Sprintf("audio_%d.wav", time.Now().UnixNano()))
	if err := writeWAV(wavPath, samples, w.sampleRate); err != nil {
		return Result{}, fmt.Errorf("failed to write WAV file: %w", err)
	}
	defer os.Remove(wavPath)

	return w.TranscribeFile(ctx, wavPath)
}

// TranscribeFile transcribes audio from a file
func (w *WhisperCLI) TranscribeFile(ctx context.Context, path string) (Result, error) {
	// Build command arguments for whisper-cli
	args := []string{
		"--model", w.modelPath,
		"--language", w.language,
		"--no-prints",      // suppress info output
		"--output-txt",     // output as text
		"--output-file", "-", // output to stdout
		path,               // input file as last argument
	}

	// Run whisper-cli
	cmd := exec.CommandContext(ctx, w.binaryPath, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		// Try alternative argument format
		args2 := []string{
			"-m", w.modelPath,
			"-l", w.language,
			"-np",  // no prints
			path,
		}
		cmd2 := exec.CommandContext(ctx, w.binaryPath, args2...)
		stdout.Reset()
		stderr.Reset()
		cmd2.Stdout = &stdout
		cmd2.Stderr = &stderr

		if err2 := cmd2.Run(); err2 != nil {
			return Result{}, fmt.Errorf("whisper failed: %w, stderr: %s", err, stderr.String())
		}
	}

	// Parse output - whisper outputs transcription directly
	text := strings.TrimSpace(stdout.String())

	// Clean up timestamps if present (format: [00:00:00.000 --> 00:00:05.000] text)
	lines := strings.Split(text, "\n")
	var cleanLines []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		// Skip timestamp lines
		if strings.HasPrefix(line, "[") && strings.Contains(line, "-->") {
			// Extract text after timestamp
			if idx := strings.Index(line, "]"); idx != -1 {
				line = strings.TrimSpace(line[idx+1:])
			}
		}
		if line != "" {
			cleanLines = append(cleanLines, line)
		}
	}
	text = strings.Join(cleanLines, " ")

	return Result{
		Text:       text,
		Language:   w.language,
		Confidence: 0.9, // Whisper doesn't provide confidence
		Duration:   float32(len(text)) / 10, // Rough estimate
	}, nil
}

// Close releases resources
func (w *WhisperCLI) Close() error {
	if w.tempDir != "" {
		os.RemoveAll(w.tempDir)
	}
	return nil
}

// SetLanguage updates the transcription language
func (w *WhisperCLI) SetLanguage(language string) {
	w.language = language
}

// Language returns the current language
func (w *WhisperCLI) Language() string {
	return w.language
}

// writeWAV writes float32 samples to a WAV file
func writeWAV(path string, samples []float32, sampleRate int) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	// Convert float32 to int16
	int16Samples := make([]int16, len(samples))
	for i, s := range samples {
		if s > 1.0 {
			s = 1.0
		}
		if s < -1.0 {
			s = -1.0
		}
		int16Samples[i] = int16(s * 32767)
	}

	// WAV header
	numChannels := uint16(1)
	bitsPerSample := uint16(16)
	byteRate := uint32(sampleRate) * uint32(numChannels) * uint32(bitsPerSample) / 8
	blockAlign := numChannels * bitsPerSample / 8
	dataSize := uint32(len(int16Samples) * 2)

	// RIFF header
	f.Write([]byte("RIFF"))
	binary.Write(f, binary.LittleEndian, uint32(36+dataSize))
	f.Write([]byte("WAVE"))

	// fmt chunk
	f.Write([]byte("fmt "))
	binary.Write(f, binary.LittleEndian, uint32(16)) // chunk size
	binary.Write(f, binary.LittleEndian, uint16(1))  // audio format (PCM)
	binary.Write(f, binary.LittleEndian, numChannels)
	binary.Write(f, binary.LittleEndian, uint32(sampleRate))
	binary.Write(f, binary.LittleEndian, byteRate)
	binary.Write(f, binary.LittleEndian, blockAlign)
	binary.Write(f, binary.LittleEndian, bitsPerSample)

	// data chunk
	f.Write([]byte("data"))
	binary.Write(f, binary.LittleEndian, dataSize)

	// Write samples
	for _, s := range int16Samples {
		binary.Write(f, binary.LittleEndian, s)
	}

	return nil
}

// WhisperHTTP implements speech-to-text using a Whisper HTTP server
// Compatible with go-whisper server or LocalAI
type WhisperHTTP struct {
	baseURL    string
	language   string
	sampleRate int
	client     *http.Client
}

// NewWhisperHTTP creates a new Whisper HTTP client
func NewWhisperHTTP(baseURL string, cfg Config) *WhisperHTTP {
	return &WhisperHTTP{
		baseURL:    baseURL,
		language:   cfg.Language,
		sampleRate: cfg.SampleRate,
		client: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

// Transcribe converts audio samples to text via HTTP
func (w *WhisperHTTP) Transcribe(ctx context.Context, samples []float32) (Result, error) {
	// Convert to WAV in memory
	var buf bytes.Buffer
	if err := writeWAVToWriter(&buf, samples, w.sampleRate); err != nil {
		return Result{}, fmt.Errorf("failed to create WAV: %w", err)
	}

	// Create multipart request
	url := fmt.Sprintf("%s/v1/audio/transcriptions", w.baseURL)
	req, err := http.NewRequestWithContext(ctx, "POST", url, &buf)
	if err != nil {
		return Result{}, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "audio/wav")

	// Add language parameter
	q := req.URL.Query()
	q.Add("language", w.language)
	req.URL.RawQuery = q.Encode()

	resp, err := w.client.Do(req)
	if err != nil {
		return Result{}, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return Result{}, fmt.Errorf("server returned %d: %s", resp.StatusCode, string(body))
	}

	// Parse response
	var response struct {
		Text string `json:"text"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return Result{}, fmt.Errorf("failed to decode response: %w", err)
	}

	return Result{
		Text:       response.Text,
		Language:   w.language,
		Confidence: 0.9,
	}, nil
}

// TranscribeFile transcribes audio from a file via HTTP
func (w *WhisperHTTP) TranscribeFile(ctx context.Context, path string) (Result, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Result{}, fmt.Errorf("failed to read file: %w", err)
	}

	url := fmt.Sprintf("%s/v1/audio/transcriptions", w.baseURL)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(data))
	if err != nil {
		return Result{}, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "audio/wav")

	resp, err := w.client.Do(req)
	if err != nil {
		return Result{}, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return Result{}, fmt.Errorf("server returned %d: %s", resp.StatusCode, string(body))
	}

	var response struct {
		Text string `json:"text"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return Result{}, fmt.Errorf("failed to decode response: %w", err)
	}

	return Result{
		Text:       response.Text,
		Language:   w.language,
		Confidence: 0.9,
	}, nil
}

// Close releases resources
func (w *WhisperHTTP) Close() error {
	return nil
}

// SetLanguage updates the transcription language
func (w *WhisperHTTP) SetLanguage(language string) {
	w.language = language
}

// Language returns the current language
func (w *WhisperHTTP) Language() string {
	return w.language
}

// writeWAVToWriter writes WAV data to an io.Writer
func writeWAVToWriter(w io.Writer, samples []float32, sampleRate int) error {
	// Convert float32 to int16
	int16Samples := make([]int16, len(samples))
	for i, s := range samples {
		if s > 1.0 {
			s = 1.0
		}
		if s < -1.0 {
			s = -1.0
		}
		int16Samples[i] = int16(s * 32767)
	}

	numChannels := uint16(1)
	bitsPerSample := uint16(16)
	byteRate := uint32(sampleRate) * uint32(numChannels) * uint32(bitsPerSample) / 8
	blockAlign := numChannels * bitsPerSample / 8
	dataSize := uint32(len(int16Samples) * 2)

	// RIFF header
	w.Write([]byte("RIFF"))
	binary.Write(w, binary.LittleEndian, uint32(36+dataSize))
	w.Write([]byte("WAVE"))

	// fmt chunk
	w.Write([]byte("fmt "))
	binary.Write(w, binary.LittleEndian, uint32(16))
	binary.Write(w, binary.LittleEndian, uint16(1))
	binary.Write(w, binary.LittleEndian, numChannels)
	binary.Write(w, binary.LittleEndian, uint32(sampleRate))
	binary.Write(w, binary.LittleEndian, byteRate)
	binary.Write(w, binary.LittleEndian, blockAlign)
	binary.Write(w, binary.LittleEndian, bitsPerSample)

	// data chunk
	w.Write([]byte("data"))
	binary.Write(w, binary.LittleEndian, dataSize)

	for _, s := range int16Samples {
		binary.Write(w, binary.LittleEndian, s)
	}

	return nil
}
