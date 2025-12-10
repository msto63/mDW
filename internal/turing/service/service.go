package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	mdwerror "github.com/msto63/mDW/foundation/core/error"
	ctxmgr "github.com/msto63/mDW/internal/turing/context"
	"github.com/msto63/mDW/internal/turing/provider"
	"github.com/msto63/mDW/internal/turing/store"
	"github.com/msto63/mDW/pkg/core/cache"
	"github.com/msto63/mDW/pkg/core/logging"
)

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

// ChatRequest represents a chat completion request
type ChatRequest struct {
	Messages       []Message
	Model          string
	MaxTokens      int
	Temperature    float64
	TopP           float64
	Stream         bool
	ConversationID string // Optional: for conversation memory
	SaveToHistory  bool   // Whether to save messages to history
}

// Message represents a chat message
type Message struct {
	Role    string
	Content string
}

// ChatResponse represents a chat completion response
type ChatResponse struct {
	Message       Message
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
}

// Service is the Turing LLM service
type Service struct {
	providers    *provider.Manager
	logger       *logging.Logger
	defaultModel string
	embedModel   string
	cache        *cache.ModelsCache
	convStore    store.ConversationStore
	ctxManager   *ctxmgr.Manager
}

// Config holds service configuration
type Config struct {
	// Ollama configuration
	OllamaURL      string
	OllamaTimeout  time.Duration
	DefaultModel   string
	EmbeddingModel string

	// Multi-Provider configuration
	OpenAIKey        string
	OpenAIModel      string
	OpenAIEmbedModel string
	AnthropicKey     string
	AnthropicModel   string
	DefaultProvider  string // "ollama", "openai", "anthropic"
	EmbedProvider    string // "ollama", "openai"

	// Cache configuration
	EnableCache    bool
	ModelsCacheTTL time.Duration
	EmbedCacheTTL  time.Duration

	// Conversation memory
	ConversationStorePath string
	EnableConversations   bool

	// Context window management
	EnableContextManagement bool
	MaxContextTokens        int
	ContextReserveTokens    int
	SummarizeThreshold      float64
	MinMessagesToKeep       int
}

// DefaultConfig returns default configuration
func DefaultConfig() Config {
	return Config{
		OllamaURL:             "http://localhost:11434",
		OllamaTimeout:         120 * time.Second,
		DefaultModel:          "llama3.2",
		EmbeddingModel:        "nomic-embed-text",
		DefaultProvider:       "ollama",
		EmbedProvider:         "ollama",
		EnableCache:           true,
		ModelsCacheTTL:        1 * time.Hour,
		EmbedCacheTTL:         24 * time.Hour,
		ConversationStorePath: "./data/conversations.db",
		EnableConversations:   true,
		// Context window defaults
		EnableContextManagement: true,
		MaxContextTokens:        8192,
		ContextReserveTokens:    1024,
		SummarizeThreshold:      0.75,
		MinMessagesToKeep:       4,
	}
}

// NewService creates a new Turing service
func NewService(cfg Config) (*Service, error) {
	logger := logging.New("turing")

	// Initialize provider manager with multi-provider support
	providerMgr, err := provider.NewManager(provider.ManagerConfig{
		OllamaURL:     cfg.OllamaURL,
		OllamaTimeout: int(cfg.OllamaTimeout.Seconds()),
		OllamaModel:   cfg.DefaultModel,
		OllamaEmbed:   cfg.EmbeddingModel,

		OpenAIKey:   cfg.OpenAIKey,
		OpenAIModel: cfg.OpenAIModel,
		OpenAIEmbed: cfg.OpenAIEmbedModel,

		AnthropicKey:   cfg.AnthropicKey,
		AnthropicModel: cfg.AnthropicModel,

		DefaultProvider: cfg.DefaultProvider,
		EmbedProvider:   cfg.EmbedProvider,
	})
	if err != nil {
		return nil, mdwerror.Wrap(err, "failed to create provider manager").
			WithCode(mdwerror.CodeServiceInitialization).
			WithOperation("service.NewService")
	}

	var modelsCache *cache.ModelsCache
	if cfg.EnableCache {
		modelsCache = cache.NewModelsCache(cache.ModelsConfig{
			ModelsTTL:     cfg.ModelsCacheTTL,
			EmbedTTL:      cfg.EmbedCacheTTL,
			MaxEmbeddings: 10000,
		})
		logger.Info("Cache enabled", "models_ttl", cfg.ModelsCacheTTL, "embed_ttl", cfg.EmbedCacheTTL)
	}

	var convStore store.ConversationStore
	if cfg.EnableConversations {
		convStore, err = store.NewSQLiteConversationStore(store.SQLiteConversationConfig{
			Path: cfg.ConversationStorePath,
		})
		if err != nil {
			return nil, mdwerror.Wrap(err, "failed to create conversation store").
				WithCode(mdwerror.CodeServiceInitialization).
				WithOperation("service.NewService")
		}
		logger.Info("Conversation memory enabled", "path", cfg.ConversationStorePath)
	}

	// Initialize context manager
	var ctxManager *ctxmgr.Manager
	if cfg.EnableContextManagement {
		ctxCfg := ctxmgr.WindowConfig{
			MaxTokens:          cfg.MaxContextTokens,
			ReserveTokens:      cfg.ContextReserveTokens,
			SummarizeThreshold: cfg.SummarizeThreshold,
			MinMessagesToKeep:  cfg.MinMessagesToKeep,
			SummaryMaxTokens:   500,
		}

		// Create summarize function using this service
		// Note: We create the manager first, then set the function after service is created
		ctxManager = ctxmgr.NewManager(ctxCfg, nil)
		logger.Info("Context window management enabled",
			"max_tokens", cfg.MaxContextTokens,
			"summarize_threshold", cfg.SummarizeThreshold,
		)
	}

	svc := &Service{
		providers:    providerMgr,
		logger:       logger,
		defaultModel: cfg.DefaultModel,
		embedModel:   cfg.EmbeddingModel,
		cache:        modelsCache,
		convStore:    convStore,
		ctxManager:   ctxManager,
	}

	// Set summarize function now that service exists
	if ctxManager != nil {
		ctxManager = ctxmgr.NewManager(ctxmgr.WindowConfig{
			MaxTokens:          cfg.MaxContextTokens,
			ReserveTokens:      cfg.ContextReserveTokens,
			SummarizeThreshold: cfg.SummarizeThreshold,
			MinMessagesToKeep:  cfg.MinMessagesToKeep,
			SummaryMaxTokens:   500,
		}, svc.summarizeForContext)
		svc.ctxManager = ctxManager
	}

	return svc, nil
}

// Generate generates text from a prompt
func (s *Service) Generate(ctx context.Context, req *GenerateRequest) (*GenerateResponse, error) {
	model := req.Model
	if model == "" {
		model = s.defaultModel
	}

	s.logger.Info("Generating text",
		"model", model,
		"prompt_length", len(req.Prompt),
	)

	providerReq := &provider.GenerateRequest{
		Prompt:      req.Prompt,
		System:      req.System,
		Model:       model,
		MaxTokens:   req.MaxTokens,
		Temperature: req.Temperature,
		TopP:        req.TopP,
	}

	resp, err := s.providers.Generate(ctx, providerReq)
	if err != nil {
		s.logger.Error("Generation failed", "error", err)
		return nil, mdwerror.Wrap(err, "generation failed").
			WithCode(mdwerror.CodeExternalServiceError).
			WithOperation("service.Generate")
	}

	return &GenerateResponse{
		Text:          resp.Text,
		Model:         resp.Model,
		PromptTokens:  resp.PromptTokens,
		OutputTokens:  resp.OutputTokens,
		TotalDuration: resp.TotalDuration,
		Done:          resp.Done,
	}, nil
}

// GenerateStream generates text with streaming
func (s *Service) GenerateStream(ctx context.Context, req *GenerateRequest) (<-chan *GenerateResponse, <-chan error) {
	model := req.Model
	if model == "" {
		model = s.defaultModel
	}

	respCh := make(chan *GenerateResponse, 100)
	errCh := make(chan error, 1)

	go func() {
		defer close(respCh)
		defer close(errCh)

		providerReq := &provider.GenerateRequest{
			Prompt:      req.Prompt,
			System:      req.System,
			Model:       model,
			MaxTokens:   req.MaxTokens,
			Temperature: req.Temperature,
			TopP:        req.TopP,
			Stream:      true,
		}

		streamResp, streamErr := s.providers.GenerateStream(ctx, providerReq)

		for {
			select {
			case resp, ok := <-streamResp:
				if !ok {
					return
				}
				respCh <- &GenerateResponse{
					Text:          resp.Text,
					Model:         resp.Model,
					PromptTokens:  resp.PromptTokens,
					OutputTokens:  resp.OutputTokens,
					TotalDuration: resp.TotalDuration,
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

// Chat performs a chat completion
func (s *Service) Chat(ctx context.Context, req *ChatRequest) (*ChatResponse, error) {
	model := req.Model
	if model == "" {
		model = s.defaultModel
	}

	s.logger.Info("Chat completion",
		"model", model,
		"messages", len(req.Messages),
	)

	messages := make([]provider.Message, len(req.Messages))
	for i, msg := range req.Messages {
		messages[i] = provider.Message{
			Role:    msg.Role,
			Content: msg.Content,
		}
	}

	providerReq := &provider.ChatRequest{
		Messages:    messages,
		Model:       model,
		MaxTokens:   req.MaxTokens,
		Temperature: req.Temperature,
		TopP:        req.TopP,
	}

	resp, err := s.providers.Chat(ctx, providerReq)
	if err != nil {
		s.logger.Error("Chat failed", "error", err)
		return nil, mdwerror.Wrap(err, "chat failed").
			WithCode(mdwerror.CodeExternalServiceError).
			WithOperation("service.Chat")
	}

	return &ChatResponse{
		Message: Message{
			Role:    resp.Message.Role,
			Content: resp.Message.Content,
		},
		Model:         resp.Model,
		PromptTokens:  resp.PromptTokens,
		OutputTokens:  resp.OutputTokens,
		TotalDuration: resp.TotalDuration,
		Done:          resp.Done,
	}, nil
}

// ChatStream performs a chat completion with streaming
func (s *Service) ChatStream(ctx context.Context, req *ChatRequest) (<-chan *ChatResponse, <-chan error) {
	model := req.Model
	if model == "" {
		model = s.defaultModel
	}

	respCh := make(chan *ChatResponse, 100)
	errCh := make(chan error, 1)

	go func() {
		defer close(respCh)
		defer close(errCh)

		messages := make([]provider.Message, len(req.Messages))
		for i, msg := range req.Messages {
			messages[i] = provider.Message{
				Role:    msg.Role,
				Content: msg.Content,
			}
		}

		providerReq := &provider.ChatRequest{
			Messages:    messages,
			Model:       model,
			MaxTokens:   req.MaxTokens,
			Temperature: req.Temperature,
			TopP:        req.TopP,
			Stream:      true,
		}

		streamResp, streamErr := s.providers.ChatStream(ctx, providerReq)

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
					PromptTokens:  resp.PromptTokens,
					OutputTokens:  resp.OutputTokens,
					TotalDuration: resp.TotalDuration,
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

// Embed generates embeddings for text
func (s *Service) Embed(ctx context.Context, req *EmbeddingRequest) (*EmbeddingResponse, error) {
	model := req.Model
	if model == "" {
		model = s.embedModel
	}

	// Check cache for existing embeddings
	var embeddings [][]float64
	var textsToEmbed []string
	var textsToEmbedIdx []int

	if s.cache != nil {
		cached, missing := s.cache.GetBatchEmbeddings(model, req.Input)
		if len(missing) == 0 {
			// All embeddings cached
			s.logger.Debug("Embeddings cache hit", "count", len(req.Input))
			embeddings = make([][]float64, len(req.Input))
			for i, embed := range cached {
				embeddings[i] = embed
			}
			return &EmbeddingResponse{
				Embeddings: embeddings,
				Model:      model,
			}, nil
		}

		// Some embeddings cached, need to fetch missing ones
		embeddings = make([][]float64, len(req.Input))
		for i, embed := range cached {
			embeddings[i] = embed
		}
		for _, idx := range missing {
			textsToEmbed = append(textsToEmbed, req.Input[idx])
			textsToEmbedIdx = append(textsToEmbedIdx, idx)
		}
		s.logger.Debug("Embeddings partial cache hit",
			"cached", len(cached),
			"missing", len(missing),
		)
	} else {
		textsToEmbed = req.Input
		for i := range req.Input {
			textsToEmbedIdx = append(textsToEmbedIdx, i)
		}
		embeddings = make([][]float64, len(req.Input))
	}

	// Fetch missing embeddings with batch sharding
	if len(textsToEmbed) > 0 {
		s.logger.Info("Generating embeddings",
			"model", model,
			"inputs", len(textsToEmbed),
		)

		// Batch sharding: split into chunks to avoid overwhelming the provider
		const batchSize = 256
		for batchStart := 0; batchStart < len(textsToEmbed); batchStart += batchSize {
			batchEnd := batchStart + batchSize
			if batchEnd > len(textsToEmbed) {
				batchEnd = len(textsToEmbed)
			}

			batchTexts := textsToEmbed[batchStart:batchEnd]
			batchIndices := textsToEmbedIdx[batchStart:batchEnd]

			s.logger.Debug("Processing embedding batch",
				"batch_start", batchStart,
				"batch_size", len(batchTexts),
				"total", len(textsToEmbed),
			)

			providerReq := &provider.EmbeddingRequest{
				Model: model,
				Input: batchTexts,
			}

			resp, err := s.providers.Embed(ctx, providerReq)
			if err != nil {
				s.logger.Error("Embedding batch failed", "error", err, "batch_start", batchStart)
				return nil, mdwerror.Wrap(err, "embedding failed").
					WithCode(mdwerror.CodeExternalServiceError).
					WithOperation("service.Embed").
					WithDetail("batch_start", batchStart)
			}

			// Merge fetched embeddings into result and cache them
			for i, embed := range resp.Embeddings {
				idx := batchIndices[i]
				embeddings[idx] = embed
				if s.cache != nil {
					s.cache.SetEmbedding(model, batchTexts[i], embed)
				}
			}
		}
	}

	return &EmbeddingResponse{
		Embeddings: embeddings,
		Model:      model,
	}, nil
}

// GetCacheStats returns cache statistics
func (s *Service) GetCacheStats() map[string]interface{} {
	if s.cache != nil {
		return s.cache.Stats()
	}
	return map[string]interface{}{"enabled": false}
}

// ListModels lists available models from all providers (cached)
func (s *Service) ListModels(ctx context.Context) ([]ModelInfo, error) {
	// Check cache first
	if s.cache != nil {
		if cached, ok := s.cache.GetModels(); ok {
			if models, ok := cached.([]ModelInfo); ok {
				s.logger.Debug("Models cache hit")
				return models, nil
			}
		}
	}

	// Fetch from all providers
	providerModels, err := s.providers.ListModels(ctx)
	if err != nil {
		return nil, mdwerror.Wrap(err, "failed to list models").
			WithCode(mdwerror.CodeExternalServiceError).
			WithOperation("service.ListModels")
	}

	models := make([]ModelInfo, len(providerModels))
	for i, m := range providerModels {
		models[i] = ModelInfo{
			Name:          m.Name,
			Size:          m.Size,
			ParameterSize: m.ParameterSize,
			Family:        m.Family,
		}
	}

	// Store in cache
	if s.cache != nil {
		s.cache.SetModels(models)
		s.logger.Debug("Models cached", "count", len(models))
	}

	return models, nil
}

// InvalidateModelsCache invalidates the models cache
func (s *Service) InvalidateModelsCache() {
	if s.cache != nil {
		s.cache.InvalidateModels()
		s.logger.Debug("Models cache invalidated")
	}
}

// Summarize generates a summary of the text
func (s *Service) Summarize(ctx context.Context, text string, maxLength int) (string, error) {
	prompt := fmt.Sprintf("Summarize the following text concisely in %d words or less:\n\n%s", maxLength, text)

	resp, err := s.Generate(ctx, &GenerateRequest{
		Prompt:    prompt,
		MaxTokens: maxLength * 2, // Rough estimate
	})
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(resp.Text), nil
}

// HealthCheck checks if the service is healthy
func (s *Service) HealthCheck(ctx context.Context) error {
	results := s.providers.HealthCheck(ctx)
	for _, err := range results {
		if err != nil {
			return err
		}
	}
	return nil
}

// HealthCheckAll returns health status of all providers
func (s *Service) HealthCheckAll(ctx context.Context) map[string]error {
	return s.providers.HealthCheck(ctx)
}

// ListProviders returns all available providers
func (s *Service) ListProviders() []string {
	return s.providers.ListProviders()
}

// PullProgress represents model pull progress
type PullProgress struct {
	Status    string
	Digest    string
	Total     int64
	Completed int64
}

// PullModel pulls a model from the Ollama registry with progress streaming
func (s *Service) PullModel(ctx context.Context, name string) (<-chan *PullProgress, <-chan error) {
	s.logger.Info("Pulling model", "name", name)

	progressCh := make(chan *PullProgress, 100)
	errCh := make(chan error, 1)

	go func() {
		defer close(progressCh)
		defer close(errCh)

		// PullModel is Ollama-specific
		ollamaProvider := s.providers.GetOllamaProvider()
		if ollamaProvider == nil {
			errCh <- fmt.Errorf("Ollama provider not available")
			return
		}

		ollamaProgress, ollamaErr := ollamaProvider.PullModel(ctx, name)

		for {
			select {
			case p, ok := <-ollamaProgress:
				if !ok {
					return
				}
				progressCh <- &PullProgress{
					Status:    p.Status,
					Digest:    p.Digest,
					Total:     p.Total,
					Completed: p.Completed,
				}
			case err, ok := <-ollamaErr:
				if ok && err != nil {
					s.logger.Error("Pull failed", "error", err)
					errCh <- err
					return
				}
			case <-ctx.Done():
				return
			}
		}
	}()

	return progressCh, errCh
}

// ============================================================================
// Conversation Memory Methods
// ============================================================================

// Conversation represents a stored conversation
type Conversation = store.Conversation

// CreateConversation creates a new conversation
func (s *Service) CreateConversation(ctx context.Context, title, model string) (*Conversation, error) {
	if s.convStore == nil {
		return nil, mdwerror.New("conversation memory not enabled").
			WithCode(mdwerror.CodeInvalidInput)
	}

	conv := &Conversation{
		ID:    uuid.New().String(),
		Title: title,
		Model: model,
	}

	if err := s.convStore.CreateConversation(ctx, conv); err != nil {
		return nil, mdwerror.Wrap(err, "failed to create conversation").
			WithCode(mdwerror.CodeInternal).
			WithOperation("service.CreateConversation")
	}

	s.logger.Info("Conversation created", "id", conv.ID, "title", title)
	return conv, nil
}

// GetConversation retrieves a conversation by ID
func (s *Service) GetConversation(ctx context.Context, id string) (*Conversation, error) {
	if s.convStore == nil {
		return nil, mdwerror.New("conversation memory not enabled").
			WithCode(mdwerror.CodeInvalidInput)
	}

	return s.convStore.GetConversation(ctx, id)
}

// ListConversations returns all conversations
func (s *Service) ListConversations(ctx context.Context, limit, offset int) ([]*Conversation, error) {
	if s.convStore == nil {
		return nil, mdwerror.New("conversation memory not enabled").
			WithCode(mdwerror.CodeInvalidInput)
	}

	return s.convStore.ListConversations(ctx, limit, offset)
}

// DeleteConversation deletes a conversation and all its messages
func (s *Service) DeleteConversation(ctx context.Context, id string) error {
	if s.convStore == nil {
		return mdwerror.New("conversation memory not enabled").
			WithCode(mdwerror.CodeInvalidInput)
	}

	if err := s.convStore.DeleteConversation(ctx, id); err != nil {
		return mdwerror.Wrap(err, "failed to delete conversation").
			WithCode(mdwerror.CodeInternal).
			WithOperation("service.DeleteConversation")
	}

	s.logger.Info("Conversation deleted", "id", id)
	return nil
}

// GetConversationMessages retrieves messages for a conversation
func (s *Service) GetConversationMessages(ctx context.Context, conversationID string, limit int) ([]Message, error) {
	if s.convStore == nil {
		return nil, mdwerror.New("conversation memory not enabled").
			WithCode(mdwerror.CodeInvalidInput)
	}

	storedMsgs, err := s.convStore.GetMessages(ctx, conversationID, limit)
	if err != nil {
		return nil, mdwerror.Wrap(err, "failed to get messages").
			WithCode(mdwerror.CodeInternal).
			WithOperation("service.GetConversationMessages")
	}

	messages := make([]Message, len(storedMsgs))
	for i, m := range storedMsgs {
		messages[i] = Message{
			Role:    m.Role,
			Content: m.Content,
		}
	}

	return messages, nil
}

// ChatWithConversation performs a chat with conversation history
func (s *Service) ChatWithConversation(ctx context.Context, conversationID string, userMessage string, model string) (*ChatResponse, error) {
	if s.convStore == nil {
		// Fall back to regular chat without history
		return s.Chat(ctx, &ChatRequest{
			Messages: []Message{{Role: "user", Content: userMessage}},
			Model:    model,
		})
	}

	if model == "" {
		model = s.defaultModel
	}

	// Get or create conversation
	conv, err := s.convStore.GetConversation(ctx, conversationID)
	if err != nil {
		return nil, mdwerror.Wrap(err, "failed to get conversation").
			WithCode(mdwerror.CodeInternal).
			WithOperation("service.ChatWithConversation")
	}

	if conv == nil {
		// Create new conversation
		conv = &Conversation{
			ID:    conversationID,
			Title: truncateString(userMessage, 50),
			Model: model,
		}
		if err := s.convStore.CreateConversation(ctx, conv); err != nil {
			return nil, mdwerror.Wrap(err, "failed to create conversation").
				WithCode(mdwerror.CodeInternal).
				WithOperation("service.ChatWithConversation")
		}
	}

	// Load conversation history
	storedMsgs, err := s.convStore.GetMessages(ctx, conversationID, 0)
	if err != nil {
		return nil, mdwerror.Wrap(err, "failed to load conversation history").
			WithCode(mdwerror.CodeInternal).
			WithOperation("service.ChatWithConversation")
	}

	// Build messages with history
	messages := make([]Message, 0, len(storedMsgs)+1)
	for _, m := range storedMsgs {
		messages = append(messages, Message{
			Role:    m.Role,
			Content: m.Content,
		})
	}
	messages = append(messages, Message{Role: "user", Content: userMessage})

	// Apply context window management if enabled
	if s.ctxManager != nil {
		// Convert to context manager messages
		ctxMessages := make([]ctxmgr.Message, len(messages))
		for i, m := range messages {
			ctxMessages[i] = ctxmgr.Message{
				Role:    m.Role,
				Content: m.Content,
			}
		}

		// Process messages through context manager
		result, err := s.ctxManager.ProcessMessages(ctx, ctxMessages, model)
		if err != nil {
			s.logger.Warn("Context processing failed, using original messages", "error", err)
		} else {
			// Convert back to service messages
			messages = make([]Message, len(result.Messages))
			for i, m := range result.Messages {
				messages[i] = Message{
					Role:    m.Role,
					Content: m.Content,
				}
			}

			if result.WasSummarized {
				s.logger.Info("Conversation context was summarized",
					"conversation_id", conversationID,
					"tokens_removed", result.TokensRemoved,
				)
			} else if result.WasTruncated {
				s.logger.Info("Conversation context was truncated",
					"conversation_id", conversationID,
					"tokens_removed", result.TokensRemoved,
				)
			}
		}
	}

	// Perform chat
	resp, err := s.Chat(ctx, &ChatRequest{
		Messages: messages,
		Model:    model,
	})
	if err != nil {
		return nil, err
	}

	// Save user message and assistant response with token counts
	userTokens := ctxmgr.EstimateTokens(userMessage)
	userMsgID := uuid.New().String()
	if err := s.convStore.AddMessage(ctx, &store.Message{
		ID:             userMsgID,
		ConversationID: conversationID,
		Role:           "user",
		Content:        userMessage,
		TokenCount:     userTokens,
	}); err != nil {
		s.logger.Warn("Failed to save user message", "error", err)
	}

	assistantMsgID := uuid.New().String()
	if err := s.convStore.AddMessage(ctx, &store.Message{
		ID:             assistantMsgID,
		ConversationID: conversationID,
		Role:           "assistant",
		Content:        resp.Message.Content,
		TokenCount:     resp.OutputTokens,
	}); err != nil {
		s.logger.Warn("Failed to save assistant message", "error", err)
	}

	return resp, nil
}

// GetConversationStats returns conversation store statistics
func (s *Service) GetConversationStats(ctx context.Context) (map[string]interface{}, error) {
	if s.convStore == nil {
		return map[string]interface{}{"enabled": false}, nil
	}

	stats, err := s.convStore.Statistics(ctx)
	if err != nil {
		return nil, err
	}
	stats["enabled"] = true
	return stats, nil
}

// Close closes the service and releases resources
func (s *Service) Close() error {
	if s.convStore != nil {
		return s.convStore.Close()
	}
	return nil
}

// truncateString truncates a string to maxLen and adds "..." if truncated
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}

// summarizeForContext is a helper for the context manager to summarize text
func (s *Service) summarizeForContext(ctx context.Context, text string, maxTokens int) (string, error) {
	return s.Summarize(ctx, text, maxTokens)
}

// GetContextWindowState returns the context window state for a conversation
func (s *Service) GetContextWindowState(ctx context.Context, conversationID string, model string) (*ctxmgr.WindowState, error) {
	if s.ctxManager == nil {
		return nil, mdwerror.New("context management not enabled").
			WithCode(mdwerror.CodeInvalidInput)
	}

	if s.convStore == nil {
		return nil, mdwerror.New("conversation memory not enabled").
			WithCode(mdwerror.CodeInvalidInput)
	}

	// Load conversation history
	storedMsgs, err := s.convStore.GetMessages(ctx, conversationID, 0)
	if err != nil {
		return nil, mdwerror.Wrap(err, "failed to load conversation history").
			WithCode(mdwerror.CodeInternal).
			WithOperation("service.GetContextWindowState")
	}

	// Convert to context manager messages
	messages := make([]ctxmgr.Message, len(storedMsgs))
	for i, m := range storedMsgs {
		messages[i] = ctxmgr.Message{
			Role:       m.Role,
			Content:    m.Content,
			TokenCount: m.TokenCount,
		}
	}

	if model == "" {
		model = s.defaultModel
	}

	state := s.ctxManager.AnalyzeWindow(messages, model)
	return &state, nil
}

// EstimateTokens estimates token count for text
func (s *Service) EstimateTokens(text string) int {
	return ctxmgr.EstimateTokens(text)
}
