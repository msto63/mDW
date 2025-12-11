// Package enrichment provides prompt enrichment for the Aristoteles service
package enrichment

import (
	"context"
	"fmt"
	"time"

	pb "github.com/msto63/mDW/api/gen/aristoteles"
	hypatiapb "github.com/msto63/mDW/api/gen/hypatia"
	"github.com/msto63/mDW/internal/aristoteles/pipeline"
	"github.com/msto63/mDW/pkg/core/logging"
)

// HypatiaClient is the interface for Hypatia (RAG) service calls
type HypatiaClient interface {
	Search(ctx context.Context, req *hypatiapb.SearchRequest) (*hypatiapb.SearchResponse, error)
}

// WebSearchFunc is a function type for web search
type WebSearchFunc func(ctx context.Context, query string) (string, error)

// Enricher coordinates prompt enrichment from various sources
type Enricher struct {
	hypatiaClient HypatiaClient
	webSearchFunc WebSearchFunc
	logger        *logging.Logger
	config        *Config
}

// Config holds enricher configuration
type Config struct {
	EnableWebSearch bool
	EnableRAG       bool
	MaxResults      int
	MinRelevance    float32
	WebTimeout      time.Duration
	RAGTimeout      time.Duration
}

// DefaultConfig returns default enricher configuration
func DefaultConfig() *Config {
	return &Config{
		EnableWebSearch: true,
		EnableRAG:       true,
		MaxResults:      5,
		MinRelevance:    0.5,
		WebTimeout:      10 * time.Second,
		RAGTimeout:      5 * time.Second,
	}
}

// NewEnricher creates a new enricher
func NewEnricher(cfg *Config) *Enricher {
	if cfg == nil {
		cfg = DefaultConfig()
	}
	return &Enricher{
		logger: logging.New("aristoteles-enrichment"),
		config: cfg,
	}
}

// SetHypatiaClient sets the Hypatia client
func (e *Enricher) SetHypatiaClient(client HypatiaClient) {
	e.hypatiaClient = client
}

// SetWebSearchFunc sets the web search function
func (e *Enricher) SetWebSearchFunc(fn WebSearchFunc) {
	e.webSearchFunc = fn
}

// Enrich enriches the prompt with relevant context
func (e *Enricher) Enrich(ctx context.Context, pctx *pipeline.Context) error {
	if pctx.Strategy == nil {
		return nil
	}

	if !pctx.Strategy.RequiresEnrichment {
		return nil
	}

	for _, enrichType := range pctx.Strategy.Enrichments {
		var step *pb.EnrichmentStep
		var err error

		switch enrichType {
		case pb.EnrichmentType_ENRICHMENT_RAG:
			step, err = e.enrichRAG(ctx, pctx.Prompt)
		case pb.EnrichmentType_ENRICHMENT_WEB_SEARCH:
			step, err = e.enrichWebSearch(ctx, pctx.Prompt)
		case pb.EnrichmentType_ENRICHMENT_CONTEXT:
			step, err = e.enrichContext(ctx, pctx)
		default:
			continue
		}

		if err != nil {
			e.logger.Warn("Enrichment failed",
				"type", enrichType.String(),
				"error", err)
			step = &pb.EnrichmentStep{
				Type:    enrichType,
				Success: false,
				Error:   err.Error(),
			}
		}

		if step != nil {
			pctx.AddEnrichment(step)
		}
	}

	return nil
}

// enrichRAG performs RAG enrichment via Hypatia
func (e *Enricher) enrichRAG(ctx context.Context, query string) (*pb.EnrichmentStep, error) {
	if !e.config.EnableRAG || e.hypatiaClient == nil {
		return nil, nil
	}

	start := time.Now()

	ctx, cancel := context.WithTimeout(ctx, e.config.RAGTimeout)
	defer cancel()

	resp, err := e.hypatiaClient.Search(ctx, &hypatiapb.SearchRequest{
		Query:    query,
		TopK:     int32(e.config.MaxResults),
		MinScore: e.config.MinRelevance,
	})
	if err != nil {
		return nil, fmt.Errorf("RAG search failed: %w", err)
	}

	if len(resp.Results) == 0 {
		return &pb.EnrichmentStep{
			Id:       fmt.Sprintf("rag-%d", time.Now().UnixNano()),
			Type:     pb.EnrichmentType_ENRICHMENT_RAG,
			Source:   "hypatia",
			Content:  "",
			Success:  true,
			DurationMs: time.Since(start).Milliseconds(),
		}, nil
	}

	// Build enrichment content from results
	var content string
	var relevanceSum float32
	for _, r := range resp.Results {
		content += fmt.Sprintf("- %s\n", r.Content)
		relevanceSum += r.Score
	}
	avgRelevance := relevanceSum / float32(len(resp.Results))

	return &pb.EnrichmentStep{
		Id:             fmt.Sprintf("rag-%d", time.Now().UnixNano()),
		Type:           pb.EnrichmentType_ENRICHMENT_RAG,
		Source:         "hypatia",
		Content:        content,
		RelevanceScore: avgRelevance,
		Success:        true,
		DurationMs:     time.Since(start).Milliseconds(),
	}, nil
}

// enrichWebSearch performs web search enrichment
func (e *Enricher) enrichWebSearch(ctx context.Context, query string) (*pb.EnrichmentStep, error) {
	if !e.config.EnableWebSearch || e.webSearchFunc == nil {
		return nil, nil
	}

	start := time.Now()

	ctx, cancel := context.WithTimeout(ctx, e.config.WebTimeout)
	defer cancel()

	content, err := e.webSearchFunc(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("web search failed: %w", err)
	}

	return &pb.EnrichmentStep{
		Id:             fmt.Sprintf("web-%d", time.Now().UnixNano()),
		Type:           pb.EnrichmentType_ENRICHMENT_WEB_SEARCH,
		Source:         "web",
		Content:        content,
		RelevanceScore: 0.8,
		Success:        true,
		DurationMs:     time.Since(start).Milliseconds(),
	}, nil
}

// enrichContext adds conversation context
func (e *Enricher) enrichContext(ctx context.Context, pctx *pipeline.Context) (*pb.EnrichmentStep, error) {
	// For now, just return nil - conversation context would require
	// loading previous messages from a conversation store
	return nil, nil
}

// Stage is the enrichment pipeline stage
type Stage struct {
	enricher      *Enricher
	maxIterations int
}

// NewStage creates a new enrichment stage
func NewStage(enricher *Enricher, maxIterations int) *Stage {
	if maxIterations <= 0 {
		maxIterations = 3
	}
	return &Stage{
		enricher:      enricher,
		maxIterations: maxIterations,
	}
}

// Name returns the stage name
func (s *Stage) Name() string {
	return "enrichment"
}

// Execute runs the enrichment stage
func (s *Stage) Execute(ctx context.Context, pctx *pipeline.Context) error {
	// Skip if disabled
	if pctx.Options != nil && pctx.Options.SkipEnrichment {
		return nil
	}

	start := time.Now()

	// Override max iterations if specified
	maxIterations := s.maxIterations
	if pctx.Options != nil && pctx.Options.MaxIterations > 0 {
		maxIterations = int(pctx.Options.MaxIterations)
	}

	// For now, single iteration enrichment
	// Future: iterative enrichment with quality checks
	for i := 0; i < maxIterations; i++ {
		if err := s.enricher.Enrich(ctx, pctx); err != nil {
			return err
		}

		// Check if we have enough enrichment
		if len(pctx.Enrichments) > 0 {
			break
		}
	}

	pctx.Metrics.EnrichmentDurationMs = time.Since(start).Milliseconds()
	pctx.Metrics.EnrichmentIterations = int32(len(pctx.Enrichments))

	return nil
}
