// ============================================================================
// meinDENKWERK (mDW) - Lokale KI-Plattform
// ============================================================================
//
// Package:     provider
// Description: OpenAI provider implementation
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

// OpenAIProvider implements the Provider interface for OpenAI
type OpenAIProvider struct {
	apiKey       string
	baseURL      string
	httpClient   *http.Client
	defaultModel string
	embedModel   string
}

// OpenAIConfig holds OpenAI provider configuration
type OpenAIConfig struct {
	APIKey       string
	BaseURL      string
	Timeout      time.Duration
	DefaultModel string
	EmbedModel   string
}

// DefaultOpenAIConfig returns default OpenAI configuration
func DefaultOpenAIConfig() OpenAIConfig {
	return OpenAIConfig{
		BaseURL:      "https://api.openai.com/v1",
		Timeout:      120 * time.Second,
		DefaultModel: "gpt-4o-mini",
		EmbedModel:   "text-embedding-3-small",
	}
}

// NewOpenAIProvider creates a new OpenAI provider
func NewOpenAIProvider(cfg OpenAIConfig) (*OpenAIProvider, error) {
	if cfg.APIKey == "" {
		return nil, fmt.Errorf("OpenAI API key is required")
	}

	if cfg.BaseURL == "" {
		cfg.BaseURL = DefaultOpenAIConfig().BaseURL
	}
	if cfg.Timeout == 0 {
		cfg.Timeout = DefaultOpenAIConfig().Timeout
	}
	if cfg.DefaultModel == "" {
		cfg.DefaultModel = DefaultOpenAIConfig().DefaultModel
	}
	if cfg.EmbedModel == "" {
		cfg.EmbedModel = DefaultOpenAIConfig().EmbedModel
	}

	return &OpenAIProvider{
		apiKey:       cfg.APIKey,
		baseURL:      cfg.BaseURL,
		httpClient:   &http.Client{Timeout: cfg.Timeout},
		defaultModel: cfg.DefaultModel,
		embedModel:   cfg.EmbedModel,
	}, nil
}

// Name returns the provider name
func (p *OpenAIProvider) Name() string {
	return "openai"
}

// OpenAI API types
type openAIChatRequest struct {
	Model       string           `json:"model"`
	Messages    []openAIMessage  `json:"messages"`
	MaxTokens   int              `json:"max_tokens,omitempty"`
	Temperature float64          `json:"temperature,omitempty"`
	TopP        float64          `json:"top_p,omitempty"`
	Stream      bool             `json:"stream,omitempty"`
}

type openAIMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type openAIChatResponse struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	Model   string `json:"model"`
	Choices []struct {
		Index        int           `json:"index"`
		Message      openAIMessage `json:"message"`
		Delta        openAIMessage `json:"delta"`
		FinishReason string        `json:"finish_reason"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
}

type openAIEmbeddingRequest struct {
	Model string   `json:"model"`
	Input []string `json:"input"`
}

type openAIEmbeddingResponse struct {
	Data []struct {
		Embedding []float64 `json:"embedding"`
		Index     int       `json:"index"`
	} `json:"data"`
	Model string `json:"model"`
	Usage struct {
		PromptTokens int `json:"prompt_tokens"`
		TotalTokens  int `json:"total_tokens"`
	} `json:"usage"`
}

type openAIModelsResponse struct {
	Data []struct {
		ID      string `json:"id"`
		Object  string `json:"object"`
		Created int64  `json:"created"`
		OwnedBy string `json:"owned_by"`
	} `json:"data"`
}

// Chat performs a chat completion
func (p *OpenAIProvider) Chat(ctx context.Context, req *ChatRequest) (*ChatResponse, error) {
	model := req.Model
	if model == "" {
		model = p.defaultModel
	}

	messages := make([]openAIMessage, 0, len(req.Messages)+1)

	// Add system message if provided
	if req.System != "" {
		messages = append(messages, openAIMessage{
			Role:    "system",
			Content: req.System,
		})
	}

	for _, msg := range req.Messages {
		messages = append(messages, openAIMessage{
			Role:    msg.Role,
			Content: msg.Content,
		})
	}

	openAIReq := openAIChatRequest{
		Model:       model,
		Messages:    messages,
		MaxTokens:   req.MaxTokens,
		Temperature: req.Temperature,
		TopP:        req.TopP,
		Stream:      false,
	}

	if openAIReq.MaxTokens == 0 {
		openAIReq.MaxTokens = 4096
	}

	start := time.Now()
	body, err := json.Marshal(openAIReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", p.baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+p.apiKey)

	resp, err := p.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("OpenAI API error: %s - %s", resp.Status, string(bodyBytes))
	}

	var openAIResp openAIChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&openAIResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if len(openAIResp.Choices) == 0 {
		return nil, fmt.Errorf("no response choices")
	}

	return &ChatResponse{
		Message: Message{
			Role:    openAIResp.Choices[0].Message.Role,
			Content: openAIResp.Choices[0].Message.Content,
		},
		Model:         openAIResp.Model,
		PromptTokens:  openAIResp.Usage.PromptTokens,
		OutputTokens:  openAIResp.Usage.CompletionTokens,
		TotalDuration: time.Since(start),
		Done:          true,
	}, nil
}

// ChatStream performs a streaming chat completion
func (p *OpenAIProvider) ChatStream(ctx context.Context, req *ChatRequest) (<-chan *ChatResponse, <-chan error) {
	respCh := make(chan *ChatResponse, 100)
	errCh := make(chan error, 1)

	go func() {
		defer close(respCh)
		defer close(errCh)

		model := req.Model
		if model == "" {
			model = p.defaultModel
		}

		messages := make([]openAIMessage, 0, len(req.Messages)+1)
		if req.System != "" {
			messages = append(messages, openAIMessage{Role: "system", Content: req.System})
		}
		for _, msg := range req.Messages {
			messages = append(messages, openAIMessage{Role: msg.Role, Content: msg.Content})
		}

		openAIReq := openAIChatRequest{
			Model:       model,
			Messages:    messages,
			MaxTokens:   req.MaxTokens,
			Temperature: req.Temperature,
			TopP:        req.TopP,
			Stream:      true,
		}

		if openAIReq.MaxTokens == 0 {
			openAIReq.MaxTokens = 4096
		}

		body, err := json.Marshal(openAIReq)
		if err != nil {
			errCh <- fmt.Errorf("failed to marshal request: %w", err)
			return
		}

		httpReq, err := http.NewRequestWithContext(ctx, "POST", p.baseURL+"/chat/completions", bytes.NewReader(body))
		if err != nil {
			errCh <- fmt.Errorf("failed to create request: %w", err)
			return
		}

		httpReq.Header.Set("Content-Type", "application/json")
		httpReq.Header.Set("Authorization", "Bearer "+p.apiKey)

		resp, err := p.httpClient.Do(httpReq)
		if err != nil {
			errCh <- fmt.Errorf("request failed: %w", err)
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			bodyBytes, _ := io.ReadAll(resp.Body)
			errCh <- fmt.Errorf("OpenAI API error: %s - %s", resp.Status, string(bodyBytes))
			return
		}

		scanner := bufio.NewScanner(resp.Body)
		for scanner.Scan() {
			line := scanner.Text()
			if len(line) < 6 || line[:6] != "data: " {
				continue
			}

			data := line[6:]
			if data == "[DONE]" {
				respCh <- &ChatResponse{Done: true, Model: model}
				return
			}

			var streamResp openAIChatResponse
			if err := json.Unmarshal([]byte(data), &streamResp); err != nil {
				continue
			}

			if len(streamResp.Choices) > 0 {
				respCh <- &ChatResponse{
					Message: Message{
						Role:    "assistant",
						Content: streamResp.Choices[0].Delta.Content,
					},
					Model: streamResp.Model,
					Done:  streamResp.Choices[0].FinishReason != "",
				}
			}
		}
	}()

	return respCh, errCh
}

// Generate generates text from a prompt
func (p *OpenAIProvider) Generate(ctx context.Context, req *GenerateRequest) (*GenerateResponse, error) {
	// Convert to chat format for OpenAI
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
func (p *OpenAIProvider) GenerateStream(ctx context.Context, req *GenerateRequest) (<-chan *GenerateResponse, <-chan error) {
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

// Embed generates embeddings
func (p *OpenAIProvider) Embed(ctx context.Context, req *EmbeddingRequest) (*EmbeddingResponse, error) {
	model := req.Model
	if model == "" {
		model = p.embedModel
	}

	openAIReq := openAIEmbeddingRequest{
		Model: model,
		Input: req.Input,
	}

	body, err := json.Marshal(openAIReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", p.baseURL+"/embeddings", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+p.apiKey)

	resp, err := p.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("OpenAI API error: %s - %s", resp.Status, string(bodyBytes))
	}

	var openAIResp openAIEmbeddingResponse
	if err := json.NewDecoder(resp.Body).Decode(&openAIResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	embeddings := make([][]float64, len(openAIResp.Data))
	for _, d := range openAIResp.Data {
		embeddings[d.Index] = d.Embedding
	}

	return &EmbeddingResponse{
		Embeddings: embeddings,
		Model:      openAIResp.Model,
	}, nil
}

// ListModels lists available models
func (p *OpenAIProvider) ListModels(ctx context.Context) ([]ModelInfo, error) {
	httpReq, err := http.NewRequestWithContext(ctx, "GET", p.baseURL+"/models", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Authorization", "Bearer "+p.apiKey)

	resp, err := p.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("OpenAI API error: %s - %s", resp.Status, string(bodyBytes))
	}

	var openAIResp openAIModelsResponse
	if err := json.NewDecoder(resp.Body).Decode(&openAIResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	models := make([]ModelInfo, len(openAIResp.Data))
	for i, m := range openAIResp.Data {
		models[i] = ModelInfo{
			Name:     m.ID,
			Family:   m.OwnedBy,
			Provider: "openai",
		}
	}

	return models, nil
}

// HealthCheck checks if the provider is healthy
func (p *OpenAIProvider) HealthCheck(ctx context.Context) error {
	_, err := p.ListModels(ctx)
	return err
}
