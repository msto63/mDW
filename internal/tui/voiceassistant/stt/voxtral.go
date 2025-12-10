// ============================================================================
// meinDENKWERK (mDW) - Lokale KI-Plattform
// ============================================================================
//
// Package:     stt
// Description: Voxtral STT client (via vLLM OpenAI-compatible API)
// Author:      Mike Stoffels with Claude
// Created:     2025-12-08
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
	"mime/multipart"
	"net/http"
	"os"
	"time"

	"github.com/msto63/mDW/pkg/core/logging"
)

// VoxtralClient implements the Transcriber interface using Voxtral via vLLM
type VoxtralClient struct {
	baseURL    string
	model      string
	language   string
	sampleRate int
	client     *http.Client
	logger     *logging.Logger
}

// VoxtralConfig holds Voxtral-specific configuration
type VoxtralConfig struct {
	// BaseURL is the vLLM server URL (e.g., "http://localhost:8100")
	BaseURL string

	// Model is the model name (default: "mistralai/Voxtral-Mini-3B-2507")
	Model string

	// Language is the target language (e.g., "de", "en", "auto")
	Language string

	// SampleRate is the audio sample rate
	SampleRate int

	// TimeoutSeconds is the request timeout
	TimeoutSeconds int
}

// DefaultVoxtralConfig returns default Voxtral configuration
func DefaultVoxtralConfig() VoxtralConfig {
	return VoxtralConfig{
		BaseURL:        "http://localhost:8100",
		Model:          "mistralai/Voxtral-Mini-3B-2507",
		Language:       "de",
		SampleRate:     16000,
		TimeoutSeconds: 60,
	}
}

// NewVoxtralClient creates a new Voxtral client
func NewVoxtralClient(cfg VoxtralConfig) *VoxtralClient {
	if cfg.BaseURL == "" {
		cfg.BaseURL = "http://localhost:8100"
	}
	if cfg.Model == "" {
		cfg.Model = "mistralai/Voxtral-Mini-3B-2507"
	}
	if cfg.SampleRate == 0 {
		cfg.SampleRate = 16000
	}
	if cfg.TimeoutSeconds == 0 {
		cfg.TimeoutSeconds = 60
	}

	return &VoxtralClient{
		baseURL:    cfg.BaseURL,
		model:      cfg.Model,
		language:   cfg.Language,
		sampleRate: cfg.SampleRate,
		client: &http.Client{
			Timeout: time.Duration(cfg.TimeoutSeconds) * time.Second,
		},
		logger: logging.New("voxtral-stt"),
	}
}

// Transcribe converts audio samples to text using Voxtral
func (c *VoxtralClient) Transcribe(ctx context.Context, samples []float32) (Result, error) {
	if len(samples) == 0 {
		return Result{}, fmt.Errorf("no audio samples provided")
	}

	// Convert samples to WAV format
	wavData, err := c.samplesToWAV(samples)
	if err != nil {
		return Result{}, fmt.Errorf("failed to convert samples to WAV: %w", err)
	}

	return c.transcribeWAV(ctx, wavData)
}

// TranscribeFile transcribes audio from a file
func (c *VoxtralClient) TranscribeFile(ctx context.Context, path string) (Result, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Result{}, fmt.Errorf("failed to read audio file: %w", err)
	}

	return c.transcribeWAV(ctx, data)
}

// transcribeWAV sends WAV data to the Voxtral API
func (c *VoxtralClient) transcribeWAV(ctx context.Context, wavData []byte) (Result, error) {
	// Create multipart form
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	// Add audio file
	part, err := writer.CreateFormFile("file", "audio.wav")
	if err != nil {
		return Result{}, fmt.Errorf("failed to create form file: %w", err)
	}
	if _, err := part.Write(wavData); err != nil {
		return Result{}, fmt.Errorf("failed to write audio data: %w", err)
	}

	// Add model
	if err := writer.WriteField("model", c.model); err != nil {
		return Result{}, fmt.Errorf("failed to write model field: %w", err)
	}

	// Add language (if not auto)
	if c.language != "" && c.language != "auto" {
		if err := writer.WriteField("language", c.language); err != nil {
			return Result{}, fmt.Errorf("failed to write language field: %w", err)
		}
	}

	// Add response format
	if err := writer.WriteField("response_format", "verbose_json"); err != nil {
		return Result{}, fmt.Errorf("failed to write response_format field: %w", err)
	}

	// Add temperature (0 for transcription)
	if err := writer.WriteField("temperature", "0"); err != nil {
		return Result{}, fmt.Errorf("failed to write temperature field: %w", err)
	}

	if err := writer.Close(); err != nil {
		return Result{}, fmt.Errorf("failed to close multipart writer: %w", err)
	}

	// Create request
	url := c.baseURL + "/v1/audio/transcriptions"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, &buf)
	if err != nil {
		return Result{}, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	// Send request
	c.logger.Debug("Sending transcription request", "url", url, "size", len(wavData))
	start := time.Now()

	resp, err := c.client.Do(req)
	if err != nil {
		return Result{}, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return Result{}, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return Result{}, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
	}

	// Parse response
	var apiResp voxtralResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return Result{}, fmt.Errorf("failed to parse response: %w", err)
	}

	c.logger.Debug("Transcription complete",
		"duration", time.Since(start),
		"text_length", len(apiResp.Text),
		"language", apiResp.Language,
	)

	// Convert to Result
	result := Result{
		Text:       apiResp.Text,
		Language:   apiResp.Language,
		Duration:   apiResp.Duration,
		Confidence: 1.0, // Voxtral doesn't return overall confidence
	}

	// Convert segments if available
	for _, seg := range apiResp.Segments {
		result.Segments = append(result.Segments, Segment{
			Text:  seg.Text,
			Start: seg.Start,
			End:   seg.End,
		})
	}

	return result, nil
}

// samplesToWAV converts float32 samples to WAV format
func (c *VoxtralClient) samplesToWAV(samples []float32) ([]byte, error) {
	// Convert float32 to int16
	int16Samples := make([]int16, len(samples))
	for i, s := range samples {
		// Clamp to [-1, 1] and convert to int16
		if s > 1.0 {
			s = 1.0
		} else if s < -1.0 {
			s = -1.0
		}
		int16Samples[i] = int16(s * 32767)
	}

	// Create WAV buffer
	var buf bytes.Buffer

	// WAV header
	numChannels := uint16(1)
	sampleRate := uint32(c.sampleRate)
	bitsPerSample := uint16(16)
	byteRate := sampleRate * uint32(numChannels) * uint32(bitsPerSample) / 8
	blockAlign := numChannels * bitsPerSample / 8
	dataSize := uint32(len(int16Samples) * 2)

	// RIFF header
	buf.WriteString("RIFF")
	binary.Write(&buf, binary.LittleEndian, uint32(36+dataSize))
	buf.WriteString("WAVE")

	// fmt chunk
	buf.WriteString("fmt ")
	binary.Write(&buf, binary.LittleEndian, uint32(16))          // chunk size
	binary.Write(&buf, binary.LittleEndian, uint16(1))           // audio format (PCM)
	binary.Write(&buf, binary.LittleEndian, numChannels)         // channels
	binary.Write(&buf, binary.LittleEndian, sampleRate)          // sample rate
	binary.Write(&buf, binary.LittleEndian, byteRate)            // byte rate
	binary.Write(&buf, binary.LittleEndian, blockAlign)          // block align
	binary.Write(&buf, binary.LittleEndian, bitsPerSample)       // bits per sample

	// data chunk
	buf.WriteString("data")
	binary.Write(&buf, binary.LittleEndian, dataSize)

	// Write samples
	for _, sample := range int16Samples {
		binary.Write(&buf, binary.LittleEndian, sample)
	}

	return buf.Bytes(), nil
}

// SetLanguage sets the transcription language
func (c *VoxtralClient) SetLanguage(lang string) {
	c.language = lang
}

// SetBaseURL sets the vLLM server URL
func (c *VoxtralClient) SetBaseURL(url string) {
	c.baseURL = url
}

// Close releases resources
func (c *VoxtralClient) Close() error {
	c.client.CloseIdleConnections()
	return nil
}

// IsAvailable checks if the Voxtral server is available
func (c *VoxtralClient) IsAvailable(ctx context.Context) bool {
	url := c.baseURL + "/health"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return false
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	return resp.StatusCode == http.StatusOK
}

// voxtralResponse is the API response structure
type voxtralResponse struct {
	Text     string            `json:"text"`
	Language string            `json:"language"`
	Duration float32           `json:"duration"`
	Segments []voxtralSegment  `json:"segments,omitempty"`
}

type voxtralSegment struct {
	ID    int     `json:"id"`
	Text  string  `json:"text"`
	Start float32 `json:"start"`
	End   float32 `json:"end"`
}
