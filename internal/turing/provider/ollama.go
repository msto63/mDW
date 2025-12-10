// ============================================================================
// meinDENKWERK (mDW) - Lokale KI-Plattform
// ============================================================================
//
// Package:     provider
// Description: Ollama provider implementation (wraps existing ollama client)
// Author:      Mike Stoffels with Claude
// Created:     2025-12-06
// License:     MIT
// ============================================================================

package provider

import (
	"context"
	"time"

	"github.com/msto63/mDW/internal/turing/ollama"
)

// OllamaProvider implements the Provider interface for Ollama
type OllamaProvider struct {
	client       *ollama.Client
	defaultModel string
	embedModel   string
}

// OllamaConfig holds Ollama provider configuration
type OllamaConfig struct {
	BaseURL      string
	Timeout      time.Duration
	DefaultModel string
	EmbedModel   string
}

// DefaultOllamaConfig returns default Ollama configuration
func DefaultOllamaConfig() OllamaConfig {
	return OllamaConfig{
		BaseURL:      "http://localhost:11434",
		Timeout:      120 * time.Second,
		DefaultModel: "llama3.2",
		EmbedModel:   "nomic-embed-text",
	}
}

// NewOllamaProvider creates a new Ollama provider
func NewOllamaProvider(cfg OllamaConfig) (*OllamaProvider, error) {
	if cfg.BaseURL == "" {
		cfg.BaseURL = DefaultOllamaConfig().BaseURL
	}
	if cfg.Timeout == 0 {
		cfg.Timeout = DefaultOllamaConfig().Timeout
	}
	if cfg.DefaultModel == "" {
		cfg.DefaultModel = DefaultOllamaConfig().DefaultModel
	}
	if cfg.EmbedModel == "" {
		cfg.EmbedModel = DefaultOllamaConfig().EmbedModel
	}

	client := ollama.NewClient(ollama.Config{
		BaseURL: cfg.BaseURL,
		Timeout: cfg.Timeout,
	})

	return &OllamaProvider{
		client:       client,
		defaultModel: cfg.DefaultModel,
		embedModel:   cfg.EmbedModel,
	}, nil
}

// Name returns the provider name
func (p *OllamaProvider) Name() string {
	return "ollama"
}

// Chat performs a chat completion
func (p *OllamaProvider) Chat(ctx context.Context, req *ChatRequest) (*ChatResponse, error) {
	model := req.Model
	if model == "" {
		model = p.defaultModel
	}

	messages := make([]ollama.ChatMessage, 0, len(req.Messages)+1)

	// Add system message if provided
	if req.System != "" {
		messages = append(messages, ollama.ChatMessage{
			Role:    "system",
			Content: req.System,
		})
	}

	for _, msg := range req.Messages {
		messages = append(messages, ollama.ChatMessage{
			Role:    msg.Role,
			Content: msg.Content,
		})
	}

	options := make(map[string]interface{})
	if req.MaxTokens > 0 {
		options["num_predict"] = req.MaxTokens
	}
	if req.Temperature > 0 {
		options["temperature"] = req.Temperature
	}
	if req.TopP > 0 {
		options["top_p"] = req.TopP
	}

	ollamaReq := &ollama.ChatRequest{
		Model:    model,
		Messages: messages,
		Options:  options,
	}

	resp, err := p.client.Chat(ctx, ollamaReq)
	if err != nil {
		return nil, err
	}

	return &ChatResponse{
		Message: Message{
			Role:    resp.Message.Role,
			Content: resp.Message.Content,
		},
		Model:         resp.Model,
		PromptTokens:  resp.PromptEvalCount,
		OutputTokens:  resp.EvalCount,
		TotalDuration: time.Duration(resp.TotalDuration),
		Done:          resp.Done,
	}, nil
}

// ChatStream performs a streaming chat completion
func (p *OllamaProvider) ChatStream(ctx context.Context, req *ChatRequest) (<-chan *ChatResponse, <-chan error) {
	respCh := make(chan *ChatResponse, 100)
	errCh := make(chan error, 1)

	go func() {
		defer close(respCh)
		defer close(errCh)

		model := req.Model
		if model == "" {
			model = p.defaultModel
		}

		messages := make([]ollama.ChatMessage, 0, len(req.Messages)+1)
		if req.System != "" {
			messages = append(messages, ollama.ChatMessage{Role: "system", Content: req.System})
		}
		for _, msg := range req.Messages {
			messages = append(messages, ollama.ChatMessage{Role: msg.Role, Content: msg.Content})
		}

		options := make(map[string]interface{})
		if req.MaxTokens > 0 {
			options["num_predict"] = req.MaxTokens
		}
		if req.Temperature > 0 {
			options["temperature"] = req.Temperature
		}

		ollamaReq := &ollama.ChatRequest{
			Model:    model,
			Messages: messages,
			Options:  options,
		}

		streamResp, streamErr := p.client.ChatStream(ctx, ollamaReq)

		for {
			select {
			case resp, ok := <-streamResp:
				if !ok {
					return
				}
				respCh <- &ChatResponse{
					Message: Message{
						Role:    resp.Message.Role,
						Content: resp.Message.Content,
					},
					Model:         resp.Model,
					PromptTokens:  resp.PromptEvalCount,
					OutputTokens:  resp.EvalCount,
					TotalDuration: time.Duration(resp.TotalDuration),
					Done:          resp.Done,
				}
			case err, ok := <-streamErr:
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

// Generate generates text from a prompt
func (p *OllamaProvider) Generate(ctx context.Context, req *GenerateRequest) (*GenerateResponse, error) {
	model := req.Model
	if model == "" {
		model = p.defaultModel
	}

	options := make(map[string]interface{})
	if req.MaxTokens > 0 {
		options["num_predict"] = req.MaxTokens
	}
	if req.Temperature > 0 {
		options["temperature"] = req.Temperature
	}
	if req.TopP > 0 {
		options["top_p"] = req.TopP
	}

	ollamaReq := &ollama.GenerateRequest{
		Model:   model,
		Prompt:  req.Prompt,
		System:  req.System,
		Options: options,
	}

	resp, err := p.client.Generate(ctx, ollamaReq)
	if err != nil {
		return nil, err
	}

	return &GenerateResponse{
		Text:          resp.Response,
		Model:         resp.Model,
		PromptTokens:  resp.PromptEvalCount,
		OutputTokens:  resp.EvalCount,
		TotalDuration: time.Duration(resp.TotalDuration),
		Done:          resp.Done,
	}, nil
}

// GenerateStream generates text with streaming
func (p *OllamaProvider) GenerateStream(ctx context.Context, req *GenerateRequest) (<-chan *GenerateResponse, <-chan error) {
	respCh := make(chan *GenerateResponse, 100)
	errCh := make(chan error, 1)

	go func() {
		defer close(respCh)
		defer close(errCh)

		model := req.Model
		if model == "" {
			model = p.defaultModel
		}

		options := make(map[string]interface{})
		if req.MaxTokens > 0 {
			options["num_predict"] = req.MaxTokens
		}
		if req.Temperature > 0 {
			options["temperature"] = req.Temperature
		}

		ollamaReq := &ollama.GenerateRequest{
			Model:   model,
			Prompt:  req.Prompt,
			System:  req.System,
			Options: options,
		}

		streamResp, streamErr := p.client.GenerateStream(ctx, ollamaReq)

		for {
			select {
			case resp, ok := <-streamResp:
				if !ok {
					return
				}
				respCh <- &GenerateResponse{
					Text:          resp.Response,
					Model:         resp.Model,
					PromptTokens:  resp.PromptEvalCount,
					OutputTokens:  resp.EvalCount,
					TotalDuration: time.Duration(resp.TotalDuration),
					Done:          resp.Done,
				}
			case err, ok := <-streamErr:
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
func (p *OllamaProvider) Embed(ctx context.Context, req *EmbeddingRequest) (*EmbeddingResponse, error) {
	model := req.Model
	if model == "" {
		model = p.embedModel
	}

	ollamaReq := &ollama.EmbeddingRequest{
		Model: model,
		Input: req.Input,
	}

	resp, err := p.client.Embed(ctx, ollamaReq)
	if err != nil {
		return nil, err
	}

	return &EmbeddingResponse{
		Embeddings: resp.Embeddings,
		Model:      resp.Model,
	}, nil
}

// ListModels lists available models
func (p *OllamaProvider) ListModels(ctx context.Context) ([]ModelInfo, error) {
	resp, err := p.client.ListModels(ctx)
	if err != nil {
		return nil, err
	}

	models := make([]ModelInfo, len(resp.Models))
	for i, m := range resp.Models {
		models[i] = ModelInfo{
			Name:          m.Name,
			Size:          m.Size,
			ParameterSize: m.Details.ParameterSize,
			Family:        m.Details.Family,
			Provider:      "ollama",
		}
	}

	return models, nil
}

// HealthCheck checks if the provider is healthy
func (p *OllamaProvider) HealthCheck(ctx context.Context) error {
	return p.client.Ping(ctx)
}

// PullModel pulls a model (Ollama-specific)
func (p *OllamaProvider) PullModel(ctx context.Context, name string) (<-chan *ollama.PullProgress, <-chan error) {
	return p.client.PullModel(ctx, name)
}

// GetClient returns the underlying Ollama client (for backward compatibility)
func (p *OllamaProvider) GetClient() *ollama.Client {
	return p.client
}
