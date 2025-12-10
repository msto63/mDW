// ============================================================================
// meinDENKWERK (mDW) - Lokale KI-Plattform
// ============================================================================
//
// Package:     provider
// Description: Anthropic provider implementation
// Author:      Mike Stoffels with Claude
// Created:     2025-12-06
// License:     MIT
// ============================================================================

package provider

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

// AnthropicProvider implements the Provider interface for Anthropic
type AnthropicProvider struct {
	apiKey       string
	baseURL      string
	httpClient   *http.Client
	defaultModel string
}

// AnthropicConfig holds Anthropic provider configuration
type AnthropicConfig struct {
	APIKey       string
	BaseURL      string
	Timeout      time.Duration
	DefaultModel string
}

// DefaultAnthropicConfig returns default Anthropic configuration
func DefaultAnthropicConfig() AnthropicConfig {
	return AnthropicConfig{
		BaseURL:      "https://api.anthropic.com/v1",
		Timeout:      120 * time.Second,
		DefaultModel: "claude-3-5-sonnet-20241022",
	}
}

// NewAnthropicProvider creates a new Anthropic provider
func NewAnthropicProvider(cfg AnthropicConfig) (*AnthropicProvider, error) {
	if cfg.APIKey == "" {
		return nil, fmt.Errorf("Anthropic API key is required")
	}

	if cfg.BaseURL == "" {
		cfg.BaseURL = DefaultAnthropicConfig().BaseURL
	}
	if cfg.Timeout == 0 {
		cfg.Timeout = DefaultAnthropicConfig().Timeout
	}
	if cfg.DefaultModel == "" {
		cfg.DefaultModel = DefaultAnthropicConfig().DefaultModel
	}

	return &AnthropicProvider{
		apiKey:       cfg.APIKey,
		baseURL:      cfg.BaseURL,
		httpClient:   &http.Client{Timeout: cfg.Timeout},
		defaultModel: cfg.DefaultModel,
	}, nil
}

// Name returns the provider name
func (p *AnthropicProvider) Name() string {
	return "anthropic"
}

// Anthropic API types
type anthropicMessage struct {
	Role    string              `json:"role"`
	Content []anthropicContent `json:"content"`
}

type anthropicContent struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
}

type anthropicRequest struct {
	Model       string             `json:"model"`
	Messages    []anthropicMessage `json:"messages"`
	MaxTokens   int                `json:"max_tokens"`
	System      string             `json:"system,omitempty"`
	Temperature float64            `json:"temperature,omitempty"`
	TopP        float64            `json:"top_p,omitempty"`
	Stream      bool               `json:"stream,omitempty"`
}

type anthropicResponse struct {
	ID         string             `json:"id"`
	Type       string             `json:"type"`
	Role       string             `json:"role"`
	Content    []anthropicContent `json:"content"`
	Model      string             `json:"model"`
	StopReason string             `json:"stop_reason"`
	Usage      struct {
		InputTokens  int `json:"input_tokens"`
		OutputTokens int `json:"output_tokens"`
	} `json:"usage"`
}

type anthropicStreamEvent struct {
	Type         string `json:"type"`
	Index        int    `json:"index"`
	ContentBlock struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content_block"`
	Delta struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"delta"`
	Message struct {
		ID         string             `json:"id"`
		Model      string             `json:"model"`
		Role       string             `json:"role"`
		Content    []anthropicContent `json:"content"`
		StopReason string             `json:"stop_reason"`
		Usage      struct {
			InputTokens  int `json:"input_tokens"`
			OutputTokens int `json:"output_tokens"`
		} `json:"usage"`
	} `json:"message"`
	Usage struct {
		OutputTokens int `json:"output_tokens"`
	} `json:"usage"`
}

// Chat performs a chat completion
func (p *AnthropicProvider) Chat(ctx context.Context, req *ChatRequest) (*ChatResponse, error) {
	model := req.Model
	if model == "" {
		model = p.defaultModel
	}

	messages := make([]anthropicMessage, 0, len(req.Messages))
	var systemPrompt string

	for _, msg := range req.Messages {
		if msg.Role == "system" {
			systemPrompt = msg.Content
			continue
		}
		messages = append(messages, anthropicMessage{
			Role: msg.Role,
			Content: []anthropicContent{
				{Type: "text", Text: msg.Content},
			},
		})
	}

	// Use explicit system if provided
	if req.System != "" {
		systemPrompt = req.System
	}

	maxTokens := req.MaxTokens
	if maxTokens == 0 {
		maxTokens = 4096
	}

	anthropicReq := anthropicRequest{
		Model:       model,
		Messages:    messages,
		MaxTokens:   maxTokens,
		System:      systemPrompt,
		Temperature: req.Temperature,
		TopP:        req.TopP,
		Stream:      false,
	}

	start := time.Now()
	body, err := json.Marshal(anthropicReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", p.baseURL+"/messages", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", p.apiKey)
	httpReq.Header.Set("anthropic-version", "2023-06-01")

	resp, err := p.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("Anthropic API error: %s - %s", resp.Status, string(bodyBytes))
	}

	var anthropicResp anthropicResponse
	if err := json.NewDecoder(resp.Body).Decode(&anthropicResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Extract text content
	var content string
	for _, c := range anthropicResp.Content {
		if c.Type == "text" {
			content += c.Text
		}
	}

	return &ChatResponse{
		Message: Message{
			Role:    "assistant",
			Content: content,
		},
		Model:         anthropicResp.Model,
		PromptTokens:  anthropicResp.Usage.InputTokens,
		OutputTokens:  anthropicResp.Usage.OutputTokens,
		TotalDuration: time.Since(start),
		Done:          true,
	}, nil
}

// ChatStream performs a streaming chat completion
func (p *AnthropicProvider) ChatStream(ctx context.Context, req *ChatRequest) (<-chan *ChatResponse, <-chan error) {
	respCh := make(chan *ChatResponse, 100)
	errCh := make(chan error, 1)

	go func() {
		defer close(respCh)
		defer close(errCh)

		model := req.Model
		if model == "" {
			model = p.defaultModel
		}

		messages := make([]anthropicMessage, 0, len(req.Messages))
		var systemPrompt string

		for _, msg := range req.Messages {
			if msg.Role == "system" {
				systemPrompt = msg.Content
				continue
			}
			messages = append(messages, anthropicMessage{
				Role: msg.Role,
				Content: []anthropicContent{
					{Type: "text", Text: msg.Content},
				},
			})
		}

		if req.System != "" {
			systemPrompt = req.System
		}

		maxTokens := req.MaxTokens
		if maxTokens == 0 {
			maxTokens = 4096
		}

		anthropicReq := anthropicRequest{
			Model:       model,
			Messages:    messages,
			MaxTokens:   maxTokens,
			System:      systemPrompt,
			Temperature: req.Temperature,
			TopP:        req.TopP,
			Stream:      true,
		}

		body, err := json.Marshal(anthropicReq)
		if err != nil {
			errCh <- fmt.Errorf("failed to marshal request: %w", err)
			return
		}

		httpReq, err := http.NewRequestWithContext(ctx, "POST", p.baseURL+"/messages", bytes.NewReader(body))
		if err != nil {
			errCh <- fmt.Errorf("failed to create request: %w", err)
			return
		}

		httpReq.Header.Set("Content-Type", "application/json")
		httpReq.Header.Set("x-api-key", p.apiKey)
		httpReq.Header.Set("anthropic-version", "2023-06-01")

		resp, err := p.httpClient.Do(httpReq)
		if err != nil {
			errCh <- fmt.Errorf("request failed: %w", err)
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			bodyBytes, _ := io.ReadAll(resp.Body)
			errCh <- fmt.Errorf("Anthropic API error: %s - %s", resp.Status, string(bodyBytes))
			return
		}

		scanner := bufio.NewScanner(resp.Body)
		for scanner.Scan() {
			line := scanner.Text()
			if len(line) < 6 || line[:6] != "data: " {
				continue
			}

			data := line[6:]
			var event anthropicStreamEvent
			if err := json.Unmarshal([]byte(data), &event); err != nil {
				continue
			}

			switch event.Type {
			case "content_block_delta":
				if event.Delta.Type == "text_delta" {
					respCh <- &ChatResponse{
						Message: Message{
							Role:    "assistant",
							Content: event.Delta.Text,
						},
						Model: model,
						Done:  false,
					}
				}
			case "message_stop":
				respCh <- &ChatResponse{
					Model: model,
					Done:  true,
				}
				return
			case "message_delta":
				if event.Usage.OutputTokens > 0 {
					respCh <- &ChatResponse{
						Model:        model,
						OutputTokens: event.Usage.OutputTokens,
						Done:         true,
					}
				}
			}
		}
	}()

	return respCh, errCh
}

// Generate generates text from a prompt
func (p *AnthropicProvider) Generate(ctx context.Context, req *GenerateRequest) (*GenerateResponse, error) {
	chatReq := &ChatRequest{
		Model:       req.Model,
		MaxTokens:   req.MaxTokens,
		Temperature: req.Temperature,
		TopP:        req.TopP,
		System:      req.System,
		Messages:    []Message{{Role: "user", Content: req.Prompt}},
	}

	resp, err := p.Chat(ctx, chatReq)
	if err != nil {
		return nil, err
	}

	return &GenerateResponse{
		Text:          resp.Message.Content,
		Model:         resp.Model,
		PromptTokens:  resp.PromptTokens,
		OutputTokens:  resp.OutputTokens,
		TotalDuration: resp.TotalDuration,
		Done:          true,
	}, nil
}

// GenerateStream generates text with streaming
func (p *AnthropicProvider) GenerateStream(ctx context.Context, req *GenerateRequest) (<-chan *GenerateResponse, <-chan error) {
	respCh := make(chan *GenerateResponse, 100)
	errCh := make(chan error, 1)

	go func() {
		defer close(respCh)
		defer close(errCh)

		chatReq := &ChatRequest{
			Model:       req.Model,
			MaxTokens:   req.MaxTokens,
			Temperature: req.Temperature,
			TopP:        req.TopP,
			System:      req.System,
			Messages:    []Message{{Role: "user", Content: req.Prompt}},
			Stream:      true,
		}

		chatResp, chatErr := p.ChatStream(ctx, chatReq)

		for {
			select {
			case resp, ok := <-chatResp:
				if !ok {
					return
				}
				respCh <- &GenerateResponse{
					Text:  resp.Message.Content,
					Model: resp.Model,
					Done:  resp.Done,
				}
			case err, ok := <-chatErr:
				if ok && err != nil {
					errCh <- err
					return
				}
			case <-ctx.Done():
				return
			}
		}
	}()

	return respCh, errCh
}

// Embed generates embeddings - Anthropic doesn't have a native embedding API
// This returns an error suggesting to use a different provider
func (p *AnthropicProvider) Embed(ctx context.Context, req *EmbeddingRequest) (*EmbeddingResponse, error) {
	return nil, fmt.Errorf("Anthropic does not provide embedding API - use OpenAI or Ollama for embeddings")
}

// ListModels lists available models - Anthropic doesn't have a public models API
func (p *AnthropicProvider) ListModels(ctx context.Context) ([]ModelInfo, error) {
	// Return known Claude models
	models := []ModelInfo{
		{Name: "claude-3-5-sonnet-20241022", Family: "claude-3.5", Provider: "anthropic"},
		{Name: "claude-3-5-haiku-20241022", Family: "claude-3.5", Provider: "anthropic"},
		{Name: "claude-3-opus-20240229", Family: "claude-3", Provider: "anthropic"},
		{Name: "claude-3-sonnet-20240229", Family: "claude-3", Provider: "anthropic"},
		{Name: "claude-3-haiku-20240307", Family: "claude-3", Provider: "anthropic"},
	}
	return models, nil
}

// HealthCheck checks if the provider is healthy
func (p *AnthropicProvider) HealthCheck(ctx context.Context) error {
	// Simple ping by trying to get models
	_, err := p.ListModels(ctx)
	return err
}
