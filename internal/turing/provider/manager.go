// ============================================================================
// meinDENKWERK (mDW) - Lokale KI-Plattform
// ============================================================================
//
// Package:     provider
// Description: Provider manager for multi-provider support
// Author:      Mike Stoffels with Claude
// Created:     2025-12-06
// License:     MIT
// ============================================================================

package provider

import (
	"context"
	"fmt"
	"sync"

	"github.com/msto63/mDW/pkg/core/logging"
)

// Manager manages multiple LLM providers
type Manager struct {
	providers       map[ProviderType]Provider
	defaultProvider ProviderType
	embedProvider   ProviderType
	logger          *logging.Logger
	mu              sync.RWMutex
}

// ManagerConfig holds manager configuration
type ManagerConfig struct {
	// Ollama config (always enabled as local default)
	OllamaURL     string
	OllamaTimeout int // seconds
	OllamaModel   string
	OllamaEmbed   string

	// OpenAI config (optional)
	OpenAIKey   string
	OpenAIModel string
	OpenAIEmbed string

	// Anthropic config (optional)
	AnthropicKey   string
	AnthropicModel string

	// Mistral config (optional)
	MistralKey   string
	MistralModel string
	MistralEmbed string

	// Default provider
	DefaultProvider string
	EmbedProvider   string
}

// NewManager creates a new provider manager
func NewManager(cfg ManagerConfig) (*Manager, error) {
	logger := logging.New("provider-manager")
	m := &Manager{
		providers:       make(map[ProviderType]Provider),
		defaultProvider: ProviderOllama,
		embedProvider:   ProviderOllama,
		logger:          logger,
	}

	// Always initialize Ollama (local, no API key required)
	ollamaCfg := DefaultOllamaConfig()
	if cfg.OllamaURL != "" {
		ollamaCfg.BaseURL = cfg.OllamaURL
	}
	if cfg.OllamaModel != "" {
		ollamaCfg.DefaultModel = cfg.OllamaModel
	}
	if cfg.OllamaEmbed != "" {
		ollamaCfg.EmbedModel = cfg.OllamaEmbed
	}

	ollamaProvider, err := NewOllamaProvider(ollamaCfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create Ollama provider: %w", err)
	}
	m.providers[ProviderOllama] = ollamaProvider
	logger.Info("Ollama provider initialized", "url", ollamaCfg.BaseURL)

	// Initialize OpenAI if API key provided
	if cfg.OpenAIKey != "" {
		openAICfg := DefaultOpenAIConfig()
		openAICfg.APIKey = cfg.OpenAIKey
		if cfg.OpenAIModel != "" {
			openAICfg.DefaultModel = cfg.OpenAIModel
		}
		if cfg.OpenAIEmbed != "" {
			openAICfg.EmbedModel = cfg.OpenAIEmbed
		}

		openAIProvider, err := NewOpenAIProvider(openAICfg)
		if err != nil {
			logger.Warn("Failed to create OpenAI provider", "error", err)
		} else {
			m.providers[ProviderOpenAI] = openAIProvider
			logger.Info("OpenAI provider initialized", "model", openAICfg.DefaultModel)
		}
	}

	// Initialize Anthropic if API key provided
	if cfg.AnthropicKey != "" {
		anthropicCfg := DefaultAnthropicConfig()
		anthropicCfg.APIKey = cfg.AnthropicKey
		if cfg.AnthropicModel != "" {
			anthropicCfg.DefaultModel = cfg.AnthropicModel
		}

		anthropicProvider, err := NewAnthropicProvider(anthropicCfg)
		if err != nil {
			logger.Warn("Failed to create Anthropic provider", "error", err)
		} else {
			m.providers[ProviderAnthropic] = anthropicProvider
			logger.Info("Anthropic provider initialized", "model", anthropicCfg.DefaultModel)
		}
	}

	// Initialize Mistral if API key provided
	if cfg.MistralKey != "" {
		mistralCfg := DefaultMistralConfig()
		mistralCfg.APIKey = cfg.MistralKey
		if cfg.MistralModel != "" {
			mistralCfg.DefaultModel = cfg.MistralModel
		}
		if cfg.MistralEmbed != "" {
			mistralCfg.EmbedModel = cfg.MistralEmbed
		}

		mistralProvider, err := NewMistralProvider(mistralCfg)
		if err != nil {
			logger.Warn("Failed to create Mistral provider", "error", err)
		} else {
			m.providers[ProviderMistral] = mistralProvider
			logger.Info("Mistral provider initialized", "model", mistralCfg.DefaultModel)
		}
	}

	// Set default provider
	if cfg.DefaultProvider != "" {
		switch cfg.DefaultProvider {
		case "openai":
			if _, ok := m.providers[ProviderOpenAI]; ok {
				m.defaultProvider = ProviderOpenAI
			}
		case "anthropic":
			if _, ok := m.providers[ProviderAnthropic]; ok {
				m.defaultProvider = ProviderAnthropic
			}
		case "mistral":
			if _, ok := m.providers[ProviderMistral]; ok {
				m.defaultProvider = ProviderMistral
			}
		case "ollama":
			m.defaultProvider = ProviderOllama
		}
	}

	// Set embed provider
	if cfg.EmbedProvider != "" {
		switch cfg.EmbedProvider {
		case "openai":
			if _, ok := m.providers[ProviderOpenAI]; ok {
				m.embedProvider = ProviderOpenAI
			}
		case "mistral":
			if _, ok := m.providers[ProviderMistral]; ok {
				m.embedProvider = ProviderMistral
			}
		case "ollama":
			m.embedProvider = ProviderOllama
		}
	}

	logger.Info("Provider manager initialized",
		"providers", len(m.providers),
		"default", m.defaultProvider,
		"embed", m.embedProvider,
	)

	return m, nil
}

// GetProvider returns a provider by type
func (m *Manager) GetProvider(providerType ProviderType) (Provider, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	provider, ok := m.providers[providerType]
	if !ok {
		return nil, fmt.Errorf("provider not available: %s", providerType)
	}

	return provider, nil
}

// GetDefaultProvider returns the default provider
func (m *Manager) GetDefaultProvider() Provider {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.providers[m.defaultProvider]
}

// GetEmbedProvider returns the embedding provider
func (m *Manager) GetEmbedProvider() Provider {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.providers[m.embedProvider]
}

// ResolveProvider resolves provider and model from a model string
// Format: "provider:model" or just "model" (uses default provider)
func (m *Manager) ResolveProvider(modelStr string) (Provider, string, error) {
	providerType, model := ParseProviderModel(modelStr)

	provider, err := m.GetProvider(providerType)
	if err != nil {
		// Fall back to default provider
		provider = m.GetDefaultProvider()
		model = modelStr
	}

	return provider, model, nil
}

// Chat performs a chat using the appropriate provider
func (m *Manager) Chat(ctx context.Context, req *ChatRequest) (*ChatResponse, error) {
	provider, model, err := m.ResolveProvider(req.Model)
	if err != nil {
		return nil, err
	}

	req.Model = model
	return provider.Chat(ctx, req)
}

// ChatStream performs a streaming chat
func (m *Manager) ChatStream(ctx context.Context, req *ChatRequest) (<-chan *ChatResponse, <-chan error) {
	provider, model, _ := m.ResolveProvider(req.Model)
	req.Model = model
	return provider.ChatStream(ctx, req)
}

// Generate generates text
func (m *Manager) Generate(ctx context.Context, req *GenerateRequest) (*GenerateResponse, error) {
	provider, model, err := m.ResolveProvider(req.Model)
	if err != nil {
		return nil, err
	}

	req.Model = model
	return provider.Generate(ctx, req)
}

// GenerateStream generates text with streaming
func (m *Manager) GenerateStream(ctx context.Context, req *GenerateRequest) (<-chan *GenerateResponse, <-chan error) {
	provider, model, _ := m.ResolveProvider(req.Model)
	req.Model = model
	return provider.GenerateStream(ctx, req)
}

// Embed generates embeddings using the embed provider
func (m *Manager) Embed(ctx context.Context, req *EmbeddingRequest) (*EmbeddingResponse, error) {
	// Check if model specifies a provider
	if req.Model != "" {
		providerType, model := ParseProviderModel(req.Model)
		if provider, err := m.GetProvider(providerType); err == nil {
			req.Model = model
			return provider.Embed(ctx, req)
		}
	}

	// Use default embed provider
	return m.GetEmbedProvider().Embed(ctx, req)
}

// ListModels lists models from all providers
func (m *Manager) ListModels(ctx context.Context) ([]ModelInfo, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var allModels []ModelInfo

	for _, provider := range m.providers {
		models, err := provider.ListModels(ctx)
		if err != nil {
			m.logger.Warn("Failed to list models", "provider", provider.Name(), "error", err)
			continue
		}
		allModels = append(allModels, models...)
	}

	return allModels, nil
}

// ListProviders returns all available providers
func (m *Manager) ListProviders() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	providers := make([]string, 0, len(m.providers))
	for p := range m.providers {
		providers = append(providers, string(p))
	}
	return providers
}

// HealthCheck checks all providers
func (m *Manager) HealthCheck(ctx context.Context) map[string]error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	results := make(map[string]error)
	for name, provider := range m.providers {
		results[string(name)] = provider.HealthCheck(ctx)
	}

	return results
}

// GetOllamaProvider returns the Ollama provider (for backward compatibility)
func (m *Manager) GetOllamaProvider() *OllamaProvider {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if p, ok := m.providers[ProviderOllama].(*OllamaProvider); ok {
		return p
	}
	return nil
}
