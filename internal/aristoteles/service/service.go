// Package service provides the business logic for the Aristoteles service
package service

import (
	"context"
	"fmt"
	"time"

	pb "github.com/msto63/mDW/api/gen/aristoteles"
	turingpb "github.com/msto63/mDW/api/gen/turing"
	"github.com/msto63/mDW/internal/aristoteles/clients"
	"github.com/msto63/mDW/internal/aristoteles/enrichment"
	"github.com/msto63/mDW/internal/aristoteles/intent"
	"github.com/msto63/mDW/internal/aristoteles/pipeline"
	"github.com/msto63/mDW/internal/aristoteles/quality"
	"github.com/msto63/mDW/internal/aristoteles/router"
	"github.com/msto63/mDW/internal/aristoteles/strategy"
	"github.com/msto63/mDW/internal/aristoteles/translation"
	"github.com/msto63/mDW/pkg/core/logging"
)

// Service is the Aristoteles business logic service
type Service struct {
	engine      *pipeline.Engine
	translation *translation.Stage
	intent      *intent.Analyzer
	strategy    *strategy.Selector
	enricher    *enrichment.Enricher
	quality     *quality.Evaluator
	router      *router.Router
	clients     *clients.ServiceClients
	logger      *logging.Logger
	config      *Config
}

// Config holds service configuration
type Config struct {
	MaxIterations             int
	QualityThreshold          float32
	EnableWebSearch           bool
	EnableRAG                 bool
	DefaultTimeoutSeconds     int
	IntentModel               string
	IntentConfidenceThreshold float32
	ModelMappings             map[string]string
	TranslationModel          string // Model for prompt translation (default: llama3.2:3b)
}

// DefaultConfig returns default service configuration
func DefaultConfig() *Config {
	return &Config{
		MaxIterations:             3,
		QualityThreshold:          0.8,
		EnableWebSearch:           true,
		EnableRAG:                 true,
		DefaultTimeoutSeconds:     180, // Increased for agent tasks like web research
		IntentModel:               "mistral:7b",
		IntentConfidenceThreshold: 0.7,
		TranslationModel:          "llama3.2:3b", // Fast model for translation
		ModelMappings: map[string]string{
			"DIRECT_LLM":         "mistral:7b",
			"CODE_GENERATION":    "qwen2.5:7b",
			"CODE_ANALYSIS":      "qwen2.5:7b",
			"TASK_DECOMPOSITION": "mistral:7b",
			"WEB_RESEARCH":       "mistral:7b",
			"RAG_QUERY":          "mistral:7b",
			"SUMMARIZATION":      "mistral:7b",
			"TRANSLATION":        "mistral:7b",
			"CREATIVE":           "mistral:7b",
		},
	}
}

// NewService creates a new Aristoteles service
func NewService(cfg *Config, serviceClients *clients.ServiceClients) *Service {
	if cfg == nil {
		cfg = DefaultConfig()
	}

	// Create components
	translationModel := cfg.TranslationModel
	if translationModel == "" {
		translationModel = "llama3.2:3b"
	}
	translationStage := translation.NewStage(&translation.Config{
		Model: translationModel,
	})

	intentAnalyzer := intent.NewAnalyzer(&intent.Config{
		Model:               cfg.IntentModel,
		ConfidenceThreshold: cfg.IntentConfidenceThreshold,
	})

	strategySelector := strategy.NewSelector(&strategy.Config{
		ModelMappings: cfg.ModelMappings,
	})

	enricher := enrichment.NewEnricher(&enrichment.Config{
		EnableWebSearch: cfg.EnableWebSearch,
		EnableRAG:       cfg.EnableRAG,
	})

	qualityEval := quality.NewEvaluator(&quality.Config{
		QualityThreshold: cfg.QualityThreshold,
	})

	routerCfg := router.DefaultConfig()
	routerCfg.DefaultTimeout = time.Duration(cfg.DefaultTimeoutSeconds) * time.Second
	routerInst := router.NewRouter(routerCfg)

	// Wire up service clients
	if serviceClients != nil {
		// Create shared LLM function for translation and intent analysis
		if serviceClients.Turing != nil {
			llmFunc := func(ctx context.Context, model, systemPrompt, userPrompt string) (string, error) {
				resp, err := serviceClients.Turing.Chat(ctx, &turingpb.ChatRequest{
					Model: model,
					Messages: []*turingpb.Message{
						{Role: "system", Content: systemPrompt},
						{Role: "user", Content: userPrompt},
					},
					Temperature: 0.1, // Low temperature for classification/translation
					MaxTokens:   500,
				})
				if err != nil {
					return "", err
				}
				return resp.Content, nil
			}

			// Set LLM func for both translation and intent analysis
			translationStage.SetLLMFunc(llmFunc)
			intentAnalyzer.SetLLMFunc(llmFunc)

			routerInst.SetTuringClient(clients.NewTuringWrapper(serviceClients.Turing))
		}

		if serviceClients.Leibniz != nil {
			routerInst.SetLeibnizClient(clients.NewLeibnizWrapper(serviceClients.Leibniz))
		}

		if serviceClients.Hypatia != nil {
			wrapper := clients.NewHypatiaWrapper(serviceClients.Hypatia)
			enricher.SetHypatiaClient(wrapper)
			routerInst.SetHypatiaClient(wrapper)
		}

		if serviceClients.Babbage != nil {
			routerInst.SetBabbageClient(clients.NewBabbageWrapper(serviceClients.Babbage))
		}
	}

	// Create pipeline engine
	engine := pipeline.NewEngine(&pipeline.Config{
		MaxIterations:           cfg.MaxIterations,
		QualityThreshold:        cfg.QualityThreshold,
		EnableWebSearch:         cfg.EnableWebSearch,
		EnableRAG:               cfg.EnableRAG,
		DefaultTimeoutSeconds:   cfg.DefaultTimeoutSeconds,
		IntentModel:             cfg.IntentModel,
		IntentConfidenceThreshold: cfg.IntentConfidenceThreshold,
	})

	// Add stages to pipeline
	// Translation stage first - translates non-English prompts for language-agnostic intent analysis
	engine.AddStage(translationStage)
	engine.AddStage(intent.NewStage(intentAnalyzer))
	engine.AddStage(strategy.NewStage(strategySelector))
	engine.AddStage(enrichment.NewStage(enricher, cfg.MaxIterations))
	engine.AddStage(quality.NewStage(qualityEval))
	engine.AddStage(router.NewStage(routerInst))

	return &Service{
		engine:      engine,
		translation: translationStage,
		intent:      intentAnalyzer,
		strategy:    strategySelector,
		enricher:    enricher,
		quality:     qualityEval,
		router:      routerInst,
		clients:     serviceClients,
		logger:      logging.New("aristoteles-service"),
		config:      cfg,
	}
}

// Process executes the full pipeline
func (s *Service) Process(ctx context.Context, req *pb.ProcessRequest) (*pb.ProcessResponse, error) {
	s.logger.Info("Processing request",
		"request_id", req.RequestId,
		"prompt_length", len(req.Prompt))

	pctx := pipeline.NewContext(
		req.RequestId,
		req.Prompt,
		req.ConversationId,
		req.Metadata,
		req.Options,
	)

	if err := s.engine.Execute(ctx, pctx); err != nil {
		return nil, fmt.Errorf("pipeline execution failed: %w", err)
	}

	return pctx.ToProcessResponse(), nil
}

// StreamProcess executes the pipeline with streaming output
func (s *Service) StreamProcess(ctx context.Context, req *pb.ProcessRequest, chunkCh chan<- *pb.ProcessChunk) error {
	s.logger.Info("Starting streaming process",
		"request_id", req.RequestId)

	pctx := pipeline.NewContext(
		req.RequestId,
		req.Prompt,
		req.ConversationId,
		req.Metadata,
		req.Options,
	)

	return s.engine.ExecuteStream(ctx, pctx, chunkCh)
}

// AnalyzeIntent analyzes a prompt for intent
func (s *Service) AnalyzeIntent(ctx context.Context, req *pb.IntentRequest) (*pb.IntentResponse, error) {
	start := time.Now()

	result, err := s.intent.Analyze(ctx, req.Prompt, req.ConversationId)
	if err != nil {
		return nil, fmt.Errorf("intent analysis failed: %w", err)
	}

	return &pb.IntentResponse{
		Intent:     result,
		DurationMs: time.Since(start).Milliseconds(),
	}, nil
}

// GetPipelineStatus returns the status of an active pipeline
func (s *Service) GetPipelineStatus(requestID string) (*pb.PipelineStatusResponse, bool) {
	return s.engine.GetStatus(requestID)
}

// CancelPipeline cancels an active pipeline
func (s *Service) CancelPipeline(requestID string) bool {
	return s.engine.Cancel(requestID)
}

// GetConfig returns the current configuration
func (s *Service) GetConfig() *pb.ConfigResponse {
	return &pb.ConfigResponse{
		MaxIterations:           int32(s.config.MaxIterations),
		QualityThreshold:        s.config.QualityThreshold,
		EnableWebSearch:         s.config.EnableWebSearch,
		EnableRag:               s.config.EnableRAG,
		DefaultTimeoutSeconds:   int32(s.config.DefaultTimeoutSeconds),
		IntentModel:             s.config.IntentModel,
		IntentConfidenceThreshold: s.config.IntentConfidenceThreshold,
		ModelMapping:            s.config.ModelMappings,
	}
}

// UpdateConfig updates the configuration
func (s *Service) UpdateConfig(req *pb.UpdateConfigRequest) *pb.ConfigResponse {
	if req.MaxIterations != nil {
		s.config.MaxIterations = int(*req.MaxIterations)
	}
	if req.QualityThreshold != nil {
		s.config.QualityThreshold = *req.QualityThreshold
	}
	if req.EnableWebSearch != nil {
		s.config.EnableWebSearch = *req.EnableWebSearch
	}
	if req.EnableRag != nil {
		s.config.EnableRAG = *req.EnableRag
	}
	if req.DefaultTimeoutSeconds != nil {
		s.config.DefaultTimeoutSeconds = int(*req.DefaultTimeoutSeconds)
	}
	if req.IntentModel != nil {
		s.config.IntentModel = *req.IntentModel
	}
	if req.IntentConfidenceThreshold != nil {
		s.config.IntentConfidenceThreshold = *req.IntentConfidenceThreshold
	}
	if len(req.ModelMapping) > 0 {
		for k, v := range req.ModelMapping {
			s.config.ModelMappings[k] = v
		}
	}

	return s.GetConfig()
}

// ListStrategies returns all available strategies
func (s *Service) ListStrategies() []*pb.StrategyInfo {
	return s.strategy.ListStrategies()
}

// GetStrategy returns a strategy by ID
func (s *Service) GetStrategy(id string) (*pb.StrategyInfo, bool) {
	return s.strategy.GetStrategy(id)
}

// Stats returns service statistics
func (s *Service) Stats() map[string]interface{} {
	return map[string]interface{}{
		"max_iterations":    s.config.MaxIterations,
		"quality_threshold": s.config.QualityThreshold,
		"enable_web_search": s.config.EnableWebSearch,
		"enable_rag":        s.config.EnableRAG,
		"intent_model":      s.config.IntentModel,
	}
}

// Close closes the service
func (s *Service) Close() error {
	if s.clients != nil {
		return s.clients.Close()
	}
	return nil
}
