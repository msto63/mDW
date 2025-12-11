// Package strategy provides strategy selection for the Aristoteles service
package strategy

import (
	"context"
	"time"

	pb "github.com/msto63/mDW/api/gen/aristoteles"
	"github.com/msto63/mDW/internal/aristoteles/pipeline"
	"github.com/msto63/mDW/pkg/core/logging"
)

// Selector selects the optimal strategy based on intent
type Selector struct {
	strategies map[string]*pb.StrategyInfo
	mappings   map[pb.IntentType]string
	logger     *logging.Logger
}

// Config holds selector configuration
type Config struct {
	ModelMappings map[string]string // Intent -> Model
}

// DefaultConfig returns default selector configuration
func DefaultConfig() *Config {
	return &Config{
		ModelMappings: map[string]string{
			"DIRECT_LLM":         "llama3.2:8b",
			"CODE_GENERATION":    "qwen2.5-coder:7b",
			"CODE_ANALYSIS":      "qwen2.5-coder:7b",
			"TASK_DECOMPOSITION": "deepseek-r1:7b",
			"WEB_RESEARCH":       "llama3.2:8b",
			"RAG_QUERY":          "llama3.2:8b",
			"SUMMARIZATION":      "llama3.2:8b",
			"TRANSLATION":        "llama3.2:8b",
			"CREATIVE":           "llama3.2:8b",
			"FACTUAL":            "llama3.2:8b",
			"CONVERSATION":       "llama3.2:8b",
			"MULTI_STEP":         "deepseek-r1:7b",
		},
	}
}

// NewSelector creates a new strategy selector
func NewSelector(cfg *Config) *Selector {
	if cfg == nil {
		cfg = DefaultConfig()
	}

	s := &Selector{
		strategies: make(map[string]*pb.StrategyInfo),
		mappings:   make(map[pb.IntentType]string),
		logger:     logging.New("aristoteles-strategy"),
	}

	s.initDefaultStrategies(cfg)
	return s
}

// initDefaultStrategies initializes the default strategies
func (s *Selector) initDefaultStrategies(cfg *Config) {
	// Direct LLM strategy
	s.strategies["direct_llm"] = &pb.StrategyInfo{
		Id:                 "direct_llm",
		Name:               "Direct LLM",
		Description:        "Direct query to LLM for simple questions",
		Model:              cfg.ModelMappings["DIRECT_LLM"],
		FallbackModel:      "llama3.2:3b",
		Target:             pb.TargetService_TARGET_TURING,
		RequiresEnrichment: false,
		Temperature:        0.7,
		MaxTokens:          2048,
	}
	s.mappings[pb.IntentType_INTENT_TYPE_DIRECT_LLM] = "direct_llm"
	s.mappings[pb.IntentType_INTENT_TYPE_CONVERSATION] = "direct_llm"

	// Code generation strategy
	s.strategies["code_generation"] = &pb.StrategyInfo{
		Id:                 "code_generation",
		Name:               "Code Generation",
		Description:        "Code generation with specialized model",
		Model:              cfg.ModelMappings["CODE_GENERATION"],
		FallbackModel:      "llama3.2:8b",
		Target:             pb.TargetService_TARGET_TURING,
		RequiresEnrichment: false,
		Temperature:        0.3,
		MaxTokens:          4096,
	}
	s.mappings[pb.IntentType_INTENT_TYPE_CODE_GENERATION] = "code_generation"

	// Code analysis strategy
	s.strategies["code_analysis"] = &pb.StrategyInfo{
		Id:                 "code_analysis",
		Name:               "Code Analysis",
		Description:        "Code review and analysis with specialized model",
		Model:              cfg.ModelMappings["CODE_ANALYSIS"],
		FallbackModel:      "llama3.2:8b",
		Target:             pb.TargetService_TARGET_TURING,
		RequiresEnrichment: false,
		Temperature:        0.2,
		MaxTokens:          4096,
	}
	s.mappings[pb.IntentType_INTENT_TYPE_CODE_ANALYSIS] = "code_analysis"

	// Web research strategy - uses Leibniz agent service
	s.strategies["web_research"] = &pb.StrategyInfo{
		Id:                 "web_research",
		Name:               "Web Research",
		Description:        "Web search and research via agent",
		Model:              cfg.ModelMappings["WEB_RESEARCH"],
		FallbackModel:      "mistral:7b",
		Target:             pb.TargetService_TARGET_LEIBNIZ,
		Agents:             []string{"web-researcher"},
		RequiresEnrichment: true,
		Enrichments:        []pb.EnrichmentType{pb.EnrichmentType_ENRICHMENT_WEB_SEARCH},
		Temperature:        0.5,
		MaxTokens:          2048,
	}
	s.mappings[pb.IntentType_INTENT_TYPE_WEB_RESEARCH] = "web_research"

	// RAG query strategy - fallback to Turing if Hypatia unavailable
	s.strategies["rag_query"] = &pb.StrategyInfo{
		Id:                 "rag_query",
		Name:               "RAG Query",
		Description:        "Knowledge base search and augmented generation",
		Model:              cfg.ModelMappings["RAG_QUERY"],
		FallbackModel:      "mistral:7b",
		Target:             pb.TargetService_TARGET_TURING, // TODO: Switch to TARGET_HYPATIA when available
		RequiresEnrichment: false,
		Temperature:        0.5,
		MaxTokens:          2048,
	}
	s.mappings[pb.IntentType_INTENT_TYPE_RAG_QUERY] = "rag_query"
	s.mappings[pb.IntentType_INTENT_TYPE_FACTUAL] = "rag_query"

	// Task decomposition strategy - uses Leibniz agent service
	s.strategies["task_decomposition"] = &pb.StrategyInfo{
		Id:                 "task_decomposition",
		Name:               "Task Decomposition",
		Description:        "Complex task planning and execution via agent",
		Model:              cfg.ModelMappings["TASK_DECOMPOSITION"],
		FallbackModel:      "mistral:7b",
		Target:             pb.TargetService_TARGET_LEIBNIZ,
		Agents:             []string{"task-planner"},
		RequiresEnrichment: true,
		Temperature:        0.4,
		MaxTokens:          4096,
	}
	s.mappings[pb.IntentType_INTENT_TYPE_TASK_DECOMPOSITION] = "task_decomposition"
	s.mappings[pb.IntentType_INTENT_TYPE_MULTI_STEP] = "task_decomposition"

	// Summarization strategy - fallback to Turing if Babbage unavailable
	s.strategies["summarization"] = &pb.StrategyInfo{
		Id:                 "summarization",
		Name:               "Summarization",
		Description:        "Text summarization",
		Model:              cfg.ModelMappings["SUMMARIZATION"],
		FallbackModel:      "mistral:7b",
		Target:             pb.TargetService_TARGET_TURING, // TODO: Switch to TARGET_BABBAGE when available
		RequiresEnrichment: false,
		Temperature:        0.3,
		MaxTokens:          1024,
	}
	s.mappings[pb.IntentType_INTENT_TYPE_SUMMARIZATION] = "summarization"

	// Translation strategy - fallback to Turing if Babbage unavailable
	s.strategies["translation"] = &pb.StrategyInfo{
		Id:                 "translation",
		Name:               "Translation",
		Description:        "Language translation",
		Model:              cfg.ModelMappings["TRANSLATION"],
		FallbackModel:      "mistral:7b",
		Target:             pb.TargetService_TARGET_TURING, // TODO: Switch to TARGET_BABBAGE when available
		RequiresEnrichment: false,
		Temperature:        0.2,
		MaxTokens:          2048,
	}
	s.mappings[pb.IntentType_INTENT_TYPE_TRANSLATION] = "translation"

	// Creative strategy
	s.strategies["creative"] = &pb.StrategyInfo{
		Id:                 "creative",
		Name:               "Creative Writing",
		Description:        "Creative content generation",
		Model:              cfg.ModelMappings["CREATIVE"],
		FallbackModel:      "llama3.2:3b",
		Target:             pb.TargetService_TARGET_TURING,
		RequiresEnrichment: false,
		Temperature:        0.9,
		MaxTokens:          4096,
	}
	s.mappings[pb.IntentType_INTENT_TYPE_CREATIVE] = "creative"
}

// Select chooses the best strategy for the given intent
func (s *Selector) Select(intent *pb.IntentResult, forceStrategy string) *pb.StrategyInfo {
	// Check for forced strategy
	if forceStrategy != "" {
		if strategy, ok := s.strategies[forceStrategy]; ok {
			return strategy
		}
		s.logger.Warn("Forced strategy not found", "strategy", forceStrategy)
	}

	// Get strategy based on primary intent
	if intent != nil {
		strategyID, ok := s.mappings[intent.Primary]
		if ok {
			if strategy, ok := s.strategies[strategyID]; ok {
				return strategy
			}
		}
	}

	// Default to direct_llm
	return s.strategies["direct_llm"]
}

// GetStrategy returns a strategy by ID
func (s *Selector) GetStrategy(id string) (*pb.StrategyInfo, bool) {
	strategy, ok := s.strategies[id]
	return strategy, ok
}

// ListStrategies returns all available strategies
func (s *Selector) ListStrategies() []*pb.StrategyInfo {
	result := make([]*pb.StrategyInfo, 0, len(s.strategies))
	for _, strategy := range s.strategies {
		result = append(result, strategy)
	}
	return result
}

// Stage is the strategy selection pipeline stage
type Stage struct {
	selector *Selector
}

// NewStage creates a new strategy stage
func NewStage(selector *Selector) *Stage {
	return &Stage{selector: selector}
}

// Name returns the stage name
func (st *Stage) Name() string {
	return "strategy"
}

// Execute runs the strategy selection stage
func (st *Stage) Execute(ctx context.Context, pctx *pipeline.Context) error {
	start := time.Now()

	var forceStrategy string
	if pctx.Options != nil && pctx.Options.ForceStrategy != "" {
		forceStrategy = pctx.Options.ForceStrategy
	}

	strategy := st.selector.Select(pctx.Intent, forceStrategy)

	// Override model if forced
	if pctx.Options != nil && pctx.Options.ForceModel != "" {
		strategyCopy := *strategy
		strategyCopy.Model = pctx.Options.ForceModel
		strategy = &strategyCopy
	}

	pctx.Strategy = strategy
	pctx.Metrics.StrategyDurationMs = time.Since(start).Milliseconds()

	return nil
}
