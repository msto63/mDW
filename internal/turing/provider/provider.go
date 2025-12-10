// ============================================================================
// meinDENKWERK (mDW) - Lokale KI-Plattform
// ============================================================================
//
// Package:     provider
// Description: LLM provider abstraction layer for multi-provider support
// Author:      Mike Stoffels with Claude
// Created:     2025-12-06
// License:     MIT
// ============================================================================

package provider

import (
	"context"
	"time"
)

// Provider represents an LLM provider interface
type Provider interface {
	// Name returns the provider name
	Name() string

	// Chat performs a chat completion
	Chat(ctx context.Context, req *ChatRequest) (*ChatResponse, error)

	// ChatStream performs a streaming chat completion
	ChatStream(ctx context.Context, req *ChatRequest) (<-chan *ChatResponse, <-chan error)

	// Generate generates text from a prompt
	Generate(ctx context.Context, req *GenerateRequest) (*GenerateResponse, error)

	// GenerateStream generates text with streaming
	GenerateStream(ctx context.Context, req *GenerateRequest) (<-chan *GenerateResponse, <-chan error)

	// Embed generates embeddings
	Embed(ctx context.Context, req *EmbeddingRequest) (*EmbeddingResponse, error)

	// ListModels lists available models
	ListModels(ctx context.Context) ([]ModelInfo, error)

	// HealthCheck checks if the provider is healthy
	HealthCheck(ctx context.Context) error
}

// Message represents a chat message
type Message struct {
	Role    string
	Content string
}

// ChatRequest represents a chat request
type ChatRequest struct {
	Messages    []Message
	Model       string
	MaxTokens   int
	Temperature float64
	TopP        float64
	Stream      bool
	System      string // System prompt (for providers that support it separately)
}

// ChatResponse represents a chat response
type ChatResponse struct {
	Message       Message
	Model         string
	PromptTokens  int
	OutputTokens  int
	TotalDuration time.Duration
	Done          bool
}

// GenerateRequest represents a text generation request
type GenerateRequest struct {
	Prompt      string
	System      string
	Model       string
	MaxTokens   int
	Temperature float64
	TopP        float64
	Stream      bool
}

// GenerateResponse represents a text generation response
type GenerateResponse struct {
	Text          string
	Model         string
	PromptTokens  int
	OutputTokens  int
	TotalDuration time.Duration
	Done          bool
}

// EmbeddingRequest represents an embedding request
type EmbeddingRequest struct {
	Input []string
	Model string
}

// EmbeddingResponse represents an embedding response
type EmbeddingResponse struct {
	Embeddings [][]float64
	Model      string
}

// ModelInfo represents model information
type ModelInfo struct {
	Name          string
	Size          int64
	ParameterSize string
	Family        string
	Provider      string
}

// ProviderType represents the type of provider
type ProviderType string

const (
	ProviderOllama    ProviderType = "ollama"
	ProviderOpenAI    ProviderType = "openai"
	ProviderAnthropic ProviderType = "anthropic"
	ProviderMistral   ProviderType = "mistral"
)

// ParseProviderModel parses a model string like "openai:gpt-4" into provider and model
func ParseProviderModel(modelStr string) (ProviderType, string) {
	for i, c := range modelStr {
		if c == ':' {
			providerName := modelStr[:i]
			model := modelStr[i+1:]
			switch providerName {
			case "openai":
				return ProviderOpenAI, model
			case "anthropic":
				return ProviderAnthropic, model
			case "ollama":
				return ProviderOllama, model
			case "mistral":
				return ProviderMistral, model
			}
			break
		}
	}
	// Default to Ollama for backward compatibility
	return ProviderOllama, modelStr
}
