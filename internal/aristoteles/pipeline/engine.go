package pipeline

import (
	"context"
	"fmt"
	"sync"
	"time"

	pb "github.com/msto63/mDW/api/gen/aristoteles"
	"github.com/msto63/mDW/pkg/core/logging"
)

// Engine is the pipeline execution engine
type Engine struct {
	stages       []Stage
	logger       *logging.Logger
	mu           sync.RWMutex
	activePipes  map[string]*Context
	config       *Config
}

// Config holds engine configuration
type Config struct {
	MaxIterations           int
	QualityThreshold        float32
	EnableWebSearch         bool
	EnableRAG               bool
	DefaultTimeoutSeconds   int
	IntentModel             string
	IntentConfidenceThreshold float32
}

// DefaultConfig returns default engine configuration
func DefaultConfig() *Config {
	return &Config{
		MaxIterations:           3,
		QualityThreshold:        0.8,
		EnableWebSearch:         true,
		EnableRAG:               true,
		DefaultTimeoutSeconds:   180, // Increased for agent tasks like web research
		IntentModel:             "llama3.2:3b",
		IntentConfidenceThreshold: 0.7,
	}
}

// NewEngine creates a new pipeline engine
func NewEngine(cfg *Config) *Engine {
	if cfg == nil {
		cfg = DefaultConfig()
	}
	return &Engine{
		stages:      make([]Stage, 0),
		logger:      logging.New("aristoteles-engine"),
		activePipes: make(map[string]*Context),
		config:      cfg,
	}
}

// AddStage adds a stage to the pipeline
func (e *Engine) AddStage(stage Stage) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.stages = append(e.stages, stage)
	e.logger.Info("Stage added", "stage", stage.Name())
}

// Execute runs the pipeline with all stages
func (e *Engine) Execute(ctx context.Context, pctx *Context) error {
	e.mu.Lock()
	e.activePipes[pctx.RequestID] = pctx
	e.mu.Unlock()

	defer func() {
		e.mu.Lock()
		delete(e.activePipes, pctx.RequestID)
		e.mu.Unlock()
	}()

	e.logger.Info("Starting pipeline execution",
		"request_id", pctx.RequestID,
		"stages", len(e.stages))

	for _, stage := range e.stages {
		// Check for cancellation
		select {
		case <-ctx.Done():
			pctx.Cancelled = true
			return ctx.Err()
		default:
		}

		// Check if pipeline was cancelled
		if pctx.Cancelled {
			return fmt.Errorf("pipeline cancelled")
		}

		// Check if pipeline was blocked
		if pctx.Blocked {
			e.logger.Info("Pipeline blocked",
				"request_id", pctx.RequestID,
				"reason", pctx.BlockReason)
			return nil
		}

		// Execute stage
		stageStart := time.Now()
		e.logger.Debug("Executing stage",
			"request_id", pctx.RequestID,
			"stage", stage.Name())

		if err := stage.Execute(ctx, pctx); err != nil {
			e.logger.Error("Stage failed",
				"request_id", pctx.RequestID,
				"stage", stage.Name(),
				"error", err)
			return fmt.Errorf("stage %s failed: %w", stage.Name(), err)
		}

		e.logger.Debug("Stage completed",
			"request_id", pctx.RequestID,
			"stage", stage.Name(),
			"duration", time.Since(stageStart))
	}

	e.logger.Info("Pipeline execution completed",
		"request_id", pctx.RequestID,
		"duration", pctx.ElapsedTime())

	return nil
}

// ExecuteStream runs the pipeline with streaming output
func (e *Engine) ExecuteStream(ctx context.Context, pctx *Context, chunkCh chan<- *pb.ProcessChunk) error {
	e.mu.Lock()
	e.activePipes[pctx.RequestID] = pctx
	e.mu.Unlock()

	defer func() {
		e.mu.Lock()
		delete(e.activePipes, pctx.RequestID)
		e.mu.Unlock()
		close(chunkCh)
	}()

	e.logger.Info("Starting streaming pipeline execution",
		"request_id", pctx.RequestID,
		"stages", len(e.stages))

	for _, stage := range e.stages {
		// Check for cancellation
		select {
		case <-ctx.Done():
			pctx.Cancelled = true
			return ctx.Err()
		default:
		}

		if pctx.Cancelled || pctx.Blocked {
			break
		}

		// Execute stage
		if err := stage.Execute(ctx, pctx); err != nil {
			return fmt.Errorf("stage %s failed: %w", stage.Name(), err)
		}

		// Send appropriate chunk based on stage
		switch stage.Name() {
		case "intent":
			if pctx.Intent != nil {
				chunkCh <- &pb.ProcessChunk{
					Type:   pb.ChunkType_CHUNK_TYPE_INTENT,
					Intent: pctx.Intent,
				}
			}
		case "strategy":
			if pctx.Strategy != nil {
				chunkCh <- &pb.ProcessChunk{
					Type:     pb.ChunkType_CHUNK_TYPE_STRATEGY,
					Strategy: pctx.Strategy,
				}
			}
		case "enrichment":
			for _, e := range pctx.Enrichments {
				chunkCh <- &pb.ProcessChunk{
					Type:       pb.ChunkType_CHUNK_TYPE_ENRICHMENT,
					Enrichment: e,
				}
			}
		case "router":
			if pctx.Response != "" {
				chunkCh <- &pb.ProcessChunk{
					Type:    pb.ChunkType_CHUNK_TYPE_RESPONSE,
					Content: pctx.Response,
				}
			}
		}
	}

	// Send final chunk with metrics
	pctx.Metrics.TotalDurationMs = pctx.ElapsedTime().Milliseconds()
	chunkCh <- &pb.ProcessChunk{
		Type:    pb.ChunkType_CHUNK_TYPE_FINAL,
		Done:    true,
		Metrics: pctx.Metrics,
	}

	return nil
}

// Cancel cancels an active pipeline
func (e *Engine) Cancel(requestID string) bool {
	e.mu.Lock()
	defer e.mu.Unlock()

	pctx, ok := e.activePipes[requestID]
	if !ok {
		return false
	}
	pctx.Cancelled = true
	return true
}

// GetStatus returns the status of an active pipeline
func (e *Engine) GetStatus(requestID string) (*pb.PipelineStatusResponse, bool) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	pctx, ok := e.activePipes[requestID]
	if !ok {
		return nil, false
	}

	return &pb.PipelineStatusResponse{
		RequestId:  pctx.RequestID,
		ElapsedMs:  pctx.ElapsedTime().Milliseconds(),
		Completed:  false,
		Cancelled:  pctx.Cancelled,
	}, true
}

// Config returns the engine configuration
func (e *Engine) Config() *Config {
	return e.config
}
