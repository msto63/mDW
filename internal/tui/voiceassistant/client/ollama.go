// ============================================================================
// meinDENKWERK (mDW) - Lokale KI-Plattform
// ============================================================================
//
// Package:     client
// Description: Direct Ollama API Client (without mDW backend)
// Author:      Mike Stoffels with Claude
// Created:     2025-12-07
// License:     MIT
// ============================================================================

package client

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// OllamaClient is a direct client for Ollama API
type OllamaClient struct {
	baseURL    string
	model      string
	httpClient *http.Client
}

// OllamaConfig holds Ollama client configuration
type OllamaConfig struct {
	BaseURL        string
	Model          string
	TimeoutSeconds int
}

// DefaultOllamaConfig returns default Ollama configuration
func DefaultOllamaConfig() OllamaConfig {
	return OllamaConfig{
		BaseURL:        "http://localhost:11434",
		Model:          "mistral:7b",
		TimeoutSeconds: 120,
	}
}

// NewOllamaClient creates a new Ollama client
func NewOllamaClient(cfg OllamaConfig) *OllamaClient {
	return &OllamaClient{
		baseURL: cfg.BaseURL,
		model:   cfg.Model,
		httpClient: &http.Client{
			Timeout: time.Duration(cfg.TimeoutSeconds) * time.Second,
		},
	}
}

// OllamaChatRequest represents an Ollama chat request
type OllamaChatRequest struct {
	Model    string          `json:"model"`
	Messages []OllamaMessage `json:"messages"`
	Stream   bool            `json:"stream"`
}

// OllamaMessage represents a chat message
type OllamaMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// OllamaChatResponse represents an Ollama chat response
type OllamaChatResponse struct {
	Model     string        `json:"model"`
	CreatedAt string        `json:"created_at"`
	Message   OllamaMessage `json:"message"`
	Done      bool          `json:"done"`
}

// Chat sends a chat request to Ollama and returns the response
func (c *OllamaClient) Chat(ctx context.Context, userMessage string) (string, error) {
	return c.ChatWithHistory(ctx, []Message{{Role: "user", Content: userMessage}})
}

// ChatWithHistory sends a chat request with history
func (c *OllamaClient) ChatWithHistory(ctx context.Context, messages []Message) (string, error) {
	// Convert to Ollama format
	ollamaMessages := make([]OllamaMessage, len(messages))
	for i, m := range messages {
		ollamaMessages[i] = OllamaMessage{Role: m.Role, Content: m.Content}
	}

	req := OllamaChatRequest{
		Model:    c.model,
		Messages: ollamaMessages,
		Stream:   false,
	}

	body, err := json.Marshal(req)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	url := c.baseURL + "/api/chat"
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
		return "", fmt.Errorf("ollama returned %d: %s", resp.StatusCode, string(respBody))
	}

	var chatResp OllamaChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&chatResp); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	return chatResp.Message.Content, nil
}

// ChatStream sends a chat request and streams the response
func (c *OllamaClient) ChatStream(ctx context.Context, userMessage string, onChunk func(chunk string, done bool)) error {
	return c.ChatStreamWithHistory(ctx, []Message{{Role: "user", Content: userMessage}}, onChunk)
}

// ChatStreamWithHistory sends a chat request with history and streams the response
func (c *OllamaClient) ChatStreamWithHistory(ctx context.Context, messages []Message, onChunk func(chunk string, done bool)) error {
	// Convert to Ollama format
	ollamaMessages := make([]OllamaMessage, len(messages))
	for i, m := range messages {
		ollamaMessages[i] = OllamaMessage{Role: m.Role, Content: m.Content}
	}

	req := OllamaChatRequest{
		Model:    c.model,
		Messages: ollamaMessages,
		Stream:   true,
	}

	body, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	url := c.baseURL + "/api/chat"
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	// Use a client without timeout for streaming
	client := &http.Client{}
	resp, err := client.Do(httpReq)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("ollama returned %d: %s", resp.StatusCode, string(respBody))
	}

	// Read streaming response
	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		var chunk OllamaChatResponse
		if err := json.Unmarshal(line, &chunk); err != nil {
			continue
		}

		if onChunk != nil {
			onChunk(chunk.Message.Content, chunk.Done)
		}

		if chunk.Done {
			return nil
		}
	}

	return scanner.Err()
}

// HealthCheck checks if Ollama is available
func (c *OllamaClient) HealthCheck(ctx context.Context) error {
	url := c.baseURL + "/api/tags"
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("health check failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("ollama unhealthy: status %d", resp.StatusCode)
	}

	return nil
}

// ListModels returns available models
func (c *OllamaClient) ListModels(ctx context.Context) ([]string, error) {
	url := c.baseURL + "/api/tags"
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	var result struct {
		Models []struct {
			Name string `json:"name"`
		} `json:"models"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	models := make([]string, len(result.Models))
	for i, m := range result.Models {
		models[i] = m.Name
	}

	return models, nil
}

// SetModel sets the model to use
func (c *OllamaClient) SetModel(model string) {
	c.model = model
}

// SetBaseURL sets the base URL for Ollama
func (c *OllamaClient) SetBaseURL(url string) {
	c.baseURL = url
}

// Model returns the current model
func (c *OllamaClient) Model() string {
	return c.model
}

// BaseURL returns the current base URL
func (c *OllamaClient) BaseURL() string {
	return c.baseURL
}
