// ============================================================================
// meinDENKWERK (mDW) - Lokale KI-Plattform
// ============================================================================
//
// Package:     client
// Description: mDW API Client
// Author:      Mike Stoffels with Claude
// Created:     2025-12-07
// License:     MIT
// ============================================================================

package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// MDWClient is the client for mDW API
type MDWClient struct {
	baseURL    string
	wsURL      string
	model      string
	httpClient *http.Client
}

// Config holds mDW client configuration
type Config struct {
	BaseURL        string
	WebSocketURL   string
	Model          string
	TimeoutSeconds int
}

// DefaultConfig returns default client configuration
func DefaultConfig() Config {
	return Config{
		BaseURL:        "http://localhost:8080",
		WebSocketURL:   "ws://localhost:8080/api/v1/chat/ws",
		Model:          "mistral:7b",
		TimeoutSeconds: 60,
	}
}

// NewMDWClient creates a new mDW client
func NewMDWClient(cfg Config) *MDWClient {
	return &MDWClient{
		baseURL: cfg.BaseURL,
		wsURL:   cfg.WebSocketURL,
		model:   cfg.Model,
		httpClient: &http.Client{
			Timeout: time.Duration(cfg.TimeoutSeconds) * time.Second,
		},
	}
}

// ChatRequest represents a chat request
type ChatRequest struct {
	Messages []Message `json:"messages"`
	Model    string    `json:"model"`
	Stream   bool      `json:"stream,omitempty"`
}

// Message represents a chat message
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ChatResponse represents a chat response
type ChatResponse struct {
	Message   Message `json:"message"`
	Model     string  `json:"model"`
	CreatedAt string  `json:"created_at"`
	Done      bool    `json:"done"`
}

// Chat sends a chat request and returns the response
func (c *MDWClient) Chat(ctx context.Context, userMessage string) (string, error) {
	req := ChatRequest{
		Messages: []Message{
			{Role: "user", Content: userMessage},
		},
		Model:  c.model,
		Stream: false,
	}

	body, err := json.Marshal(req)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	url := c.baseURL + "/api/v1/chat"
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("server returned %d: %s", resp.StatusCode, string(respBody))
	}

	var chatResp ChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&chatResp); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	return chatResp.Message.Content, nil
}

// ChatWithHistory sends a chat request with history and returns the response
func (c *MDWClient) ChatWithHistory(ctx context.Context, messages []Message) (string, error) {
	req := ChatRequest{
		Messages: messages,
		Model:    c.model,
		Stream:   false,
	}

	body, err := json.Marshal(req)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	url := c.baseURL + "/api/v1/chat"
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("server returned %d: %s", resp.StatusCode, string(respBody))
	}

	var chatResp ChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&chatResp); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	return chatResp.Message.Content, nil
}

// HealthStatus represents the detailed health status of the mDW backend
type HealthStatus struct {
	Online       bool              // Overall backend availability
	Status       string            // Overall status ("healthy", "degraded", etc.)
	Version      string            // Backend version
	Uptime       string            // Backend uptime
	Services     map[string]string // Individual service statuses
	ErrorMessage string            // Error message if health check failed
}

// HealthCheck checks if the mDW API is healthy (simple check)
func (c *MDWClient) HealthCheck(ctx context.Context) error {
	status := c.GetHealthStatus(ctx)
	if !status.Online {
		if status.ErrorMessage != "" {
			return fmt.Errorf("unhealthy: %s", status.ErrorMessage)
		}
		return fmt.Errorf("unhealthy: backend not available")
	}
	return nil
}

// GetHealthStatus returns detailed health status of the mDW backend
func (c *MDWClient) GetHealthStatus(ctx context.Context) HealthStatus {
	url := c.baseURL + "/api/v1/health"
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return HealthStatus{
			Online:       false,
			ErrorMessage: fmt.Sprintf("failed to create request: %v", err),
		}
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return HealthStatus{
			Online:       false,
			ErrorMessage: fmt.Sprintf("connection failed: %v", err),
		}
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return HealthStatus{
			Online:       false,
			ErrorMessage: fmt.Sprintf("unhealthy: status %d", resp.StatusCode),
		}
	}

	var healthResp struct {
		Status   string            `json:"status"`
		Version  string            `json:"version"`
		Uptime   string            `json:"uptime"`
		Services map[string]string `json:"services"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&healthResp); err != nil {
		return HealthStatus{
			Online:       false,
			ErrorMessage: fmt.Sprintf("failed to decode response: %v", err),
		}
	}

	// Count healthy vs unhealthy services
	healthyCount := 0
	totalCount := 0
	for _, status := range healthResp.Services {
		totalCount++
		if status == "healthy" {
			healthyCount++
		}
	}

	// Determine overall status
	overallStatus := healthResp.Status
	if healthyCount < totalCount {
		if healthyCount == 0 {
			overallStatus = "unhealthy"
		} else {
			overallStatus = "degraded"
		}
	}

	return HealthStatus{
		Online:   true,
		Status:   overallStatus,
		Version:  healthResp.Version,
		Uptime:   healthResp.Uptime,
		Services: healthResp.Services,
	}
}

// SetModel sets the model to use for chat
func (c *MDWClient) SetModel(model string) {
	c.model = model
}

// Model returns the current model
func (c *MDWClient) Model() string {
	return c.model
}

// Analyze sends text to Babbage for NLP analysis
func (c *MDWClient) Analyze(ctx context.Context, text string) (*AnalyzeResponse, error) {
	req := map[string]string{"text": text}
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	url := c.baseURL + "/api/v1/analyze"
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("server returned %d: %s", resp.StatusCode, string(respBody))
	}

	var analyzeResp AnalyzeResponse
	if err := json.NewDecoder(resp.Body).Decode(&analyzeResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &analyzeResp, nil
}

// AnalyzeResponse represents the analyze response
type AnalyzeResponse struct {
	Sentiment struct {
		Label      string  `json:"label"`
		Score      float64 `json:"score"`
		Confidence float64 `json:"confidence"`
	} `json:"sentiment"`
	Entities []struct {
		Text string `json:"text"`
		Type string `json:"type"`
	} `json:"entities"`
	Keywords []struct {
		Word  string  `json:"word"`
		Score float64 `json:"score"`
	} `json:"keywords"`
	Language struct {
		Code       string  `json:"code"`
		Confidence float64 `json:"confidence"`
	} `json:"language"`
}
