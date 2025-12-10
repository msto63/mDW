// ============================================================================
// meinDENKWERK (mDW) - Lokale KI-Plattform
// ============================================================================
//
// Package:     provider
// Description: Mistral AI provider implementation
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

// MistralProvider implements the Provider interface for Mistral AI
type MistralProvider struct {
	apiKey       string
	baseURL      string
	httpClient   *http.Client
	defaultModel string
	embedModel   string
}

// MistralConfig holds Mistral provider configuration
type MistralConfig struct {
	APIKey       string
	BaseURL      string
	Timeout      time.Duration
	DefaultModel string
	EmbedModel   string
}

// DefaultMistralConfig returns default Mistral configuration
func DefaultMistralConfig() MistralConfig {
	return MistralConfig{
		BaseURL:      "https://api.mistral.ai/v1",
		Timeout:      120 * time.Second,
		DefaultModel: "mistral-small-latest",
		EmbedModel:   "mistral-embed",
	}
}

// NewMistralProvider creates a new Mistral provider
func NewMistralProvider(cfg MistralConfig) (*MistralProvider, error) {
	if cfg.APIKey == "" {
		return nil, fmt.Errorf("Mistral API key is required")
	}

	if cfg.BaseURL == "" {
		cfg.BaseURL = DefaultMistralConfig().BaseURL
	}
	if cfg.Timeout == 0 {
		cfg.Timeout = DefaultMistralConfig().Timeout
	}
	if cfg.DefaultModel == "" {
		cfg.DefaultModel = DefaultMistralConfig().DefaultModel
	}
	if cfg.EmbedModel == "" {
		cfg.EmbedModel = DefaultMistralConfig().EmbedModel
	}

	return &MistralProvider{
		apiKey:       cfg.APIKey,
		baseURL:      cfg.BaseURL,
		httpClient:   &http.Client{Timeout: cfg.Timeout},
		defaultModel: cfg.DefaultModel,
		embedModel:   cfg.EmbedModel,
	}, nil
}

// Name returns the provider name
func (p *MistralProvider) Name() string {
	return "mistral"
}

// Mistral API types (compatible with OpenAI format)
type mistralChatRequest struct {
	Model       string           `json:"model"`
	Messages    []mistralMessage `json:"messages"`
	MaxTokens   int              `json:"max_tokens,omitempty"`
	Temperature float64          `json:"temperature,omitempty"`
	TopP        float64          `json:"top_p,omitempty"`
	Stream      bool             `json:"stream,omitempty"`
	SafePrompt  bool             `json:"safe_prompt,omitempty"`
}

type mistralMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type mistralChatResponse struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	Model   string `json:"model"`
	Choices []struct {
		Index        int            `json:"index"`
		Message      mistralMessage `json:"message"`
		Delta        mistralMessage `json:"delta"`
		FinishReason string         `json:"finish_reason"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
}

type mistralEmbeddingRequest struct {
	Model          string   `json:"model"`
	Input          []string `json:"input"`
	EncodingFormat string   `json:"encoding_format,omitempty"`
}

type mistralEmbeddingResponse struct {
	ID     string `json:"id"`
	Object string `json:"object"`
	Data   []struct {
		Object    string    `json:"object"`
		Embedding []float64 `json:"embedding"`
		Index     int       `json:"index"`
	} `json:"data"`
	Model string `json:"model"`
	Usage struct {
		PromptTokens int `json:"prompt_tokens"`
		TotalTokens  int `json:"total_tokens"`
	} `json:"usage"`
}

type mistralModelsResponse struct {
	Object string `json:"object"`
	Data   []struct {
		ID      string `json:"id"`
		Object  string `json:"object"`
		Created int64  `json:"created"`
		OwnedBy string `json:"owned_by"`
	} `json:"data"`
}

// Chat performs a chat completion
func (p *MistralProvider) Chat(ctx context.Context, req *ChatRequest) (*ChatResponse, error) {
	model := req.Model
	if model == "" {
		model = p.defaultModel
	}

	messages := make([]mistralMessage, 0, len(req.Messages)+1)

	// Add system message if provided
	if req.System != "" {
		messages = append(messages, mistralMessage{
			Role:    "system",
			Content: req.System,
		})
	}

	for _, msg := range req.Messages {
		messages = append(messages, mistralMessage{
			Role:    msg.Role,
			Content: msg.Content,
		})
	}

	mistralReq := mistralChatRequest{
		Model:       model,
		Messages:    messages,
		MaxTokens:   req.MaxTokens,
		Temperature: req.Temperature,
		TopP:        req.TopP,
		Stream:      false,
	}

	if mistralReq.MaxTokens == 0 {
		mistralReq.MaxTokens = 4096
	}

	start := time.Now()
	body, err := json.Marshal(mistralReq)
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
		return nil, fmt.Errorf("Mistral API error: %s - %s", resp.Status, string(bodyBytes))
	}

	var mistralResp mistralChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&mistralResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if len(mistralResp.Choices) == 0 {
		return nil, fmt.Errorf("no response choices")
	}

	return &ChatResponse{
		Message: Message{
			Role:    mistralResp.Choices[0].Message.Role,
			Content: mistralResp.Choices[0].Message.Content,
		},
		Model:         mistralResp.Model,
		PromptTokens:  mistralResp.Usage.PromptTokens,
		OutputTokens:  mistralResp.Usage.CompletionTokens,
		TotalDuration: time.Since(start),
		Done:          true,
	}, nil
}

// ChatStream performs a streaming chat completion
func (p *MistralProvider) ChatStream(ctx context.Context, req *ChatRequest) (<-chan *ChatResponse, <-chan error) {
	respCh := make(chan *ChatResponse, 100)
	errCh := make(chan error, 1)

	go func() {
		defer close(respCh)
		defer close(errCh)

		model := req.Model
		if model == "" {
			model = p.defaultModel
		}

		messages := make([]mistralMessage, 0, len(req.Messages)+1)
		if req.System != "" {
			messages = append(messages, mistralMessage{Role: "system", Content: req.System})
		}
		for _, msg := range req.Messages {
			messages = append(messages, mistralMessage{Role: msg.Role, Content: msg.Content})
		}

		mistralReq := mistralChatRequest{
			Model:       model,
			Messages:    messages,
			MaxTokens:   req.MaxTokens,
			Temperature: req.Temperature,
			TopP:        req.TopP,
			Stream:      true,
		}

		if mistralReq.MaxTokens == 0 {
			mistralReq.MaxTokens = 4096
		}

		body, err := json.Marshal(mistralReq)
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
			errCh <- fmt.Errorf("Mistral API error: %s - %s", resp.Status, string(bodyBytes))
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

			var streamResp mistralChatResponse
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
func (p *MistralProvider) Generate(ctx context.Context, req *GenerateRequest) (*GenerateResponse, error) {
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
func (p *MistralProvider) GenerateStream(ctx context.Context, req *GenerateRequest) (<-chan *GenerateResponse, <-chan error) {
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
func (p *MistralProvider) Embed(ctx context.Context, req *EmbeddingRequest) (*EmbeddingResponse, error) {
	model := req.Model
	if model == "" {
		model = p.embedModel
	}

	mistralReq := mistralEmbeddingRequest{
		Model:          model,
		Input:          req.Input,
		EncodingFormat: "float",
	}

	body, err := json.Marshal(mistralReq)
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
		return nil, fmt.Errorf("Mistral API error: %s - %s", resp.Status, string(bodyBytes))
	}

	var mistralResp mistralEmbeddingResponse
	if err := json.NewDecoder(resp.Body).Decode(&mistralResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	embeddings := make([][]float64, len(mistralResp.Data))
	for _, d := range mistralResp.Data {
		embeddings[d.Index] = d.Embedding
	}

	return &EmbeddingResponse{
		Embeddings: embeddings,
		Model:      mistralResp.Model,
	}, nil
}

// ListModels lists available models
func (p *MistralProvider) ListModels(ctx context.Context) ([]ModelInfo, error) {
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
		// Fall back to known models if API fails
		return p.getKnownModels(), nil
	}

	var mistralResp mistralModelsResponse
	if err := json.NewDecoder(resp.Body).Decode(&mistralResp); err != nil {
		return p.getKnownModels(), nil
	}

	models := make([]ModelInfo, len(mistralResp.Data))
	for i, m := range mistralResp.Data {
		models[i] = ModelInfo{
			Name:     m.ID,
			Family:   "mistral",
			Provider: "mistral",
		}
	}

	return models, nil
}

// getKnownModels returns statically known Mistral models
func (p *MistralProvider) getKnownModels() []ModelInfo {
	return []ModelInfo{
		// Mistral 3 (Latest - December 2025)
		{Name: "ministral-3b-latest", Family: "ministral-3", Provider: "mistral"},
		{Name: "ministral-8b-latest", Family: "ministral-3", Provider: "mistral"},
		// Premier models
		{Name: "mistral-large-latest", Family: "mistral-large", Provider: "mistral"},
		{Name: "mistral-medium-latest", Family: "mistral-medium", Provider: "mistral"},
		{Name: "mistral-small-latest", Family: "mistral-small", Provider: "mistral"},
		// Free tier
		{Name: "open-mistral-7b", Family: "open-mistral", Provider: "mistral"},
		{Name: "open-mixtral-8x7b", Family: "open-mixtral", Provider: "mistral"},
		{Name: "open-mixtral-8x22b", Family: "open-mixtral", Provider: "mistral"},
		// Specialized
		{Name: "codestral-latest", Family: "codestral", Provider: "mistral"},
		{Name: "mistral-embed", Family: "embedding", Provider: "mistral"},
	}
}

// HealthCheck checks if the provider is healthy
func (p *MistralProvider) HealthCheck(ctx context.Context) error {
	_, err := p.ListModels(ctx)
	return err
}
