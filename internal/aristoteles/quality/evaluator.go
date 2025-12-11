// Package quality provides quality evaluation for the Aristoteles service
package quality

import (
	"context"
	"time"

	"github.com/msto63/mDW/internal/aristoteles/pipeline"
	"github.com/msto63/mDW/pkg/core/logging"
)

// Evaluator evaluates the quality of pipeline results
type Evaluator struct {
	threshold float32
	logger    *logging.Logger
}

// Config holds evaluator configuration
type Config struct {
	QualityThreshold float32
}

// DefaultConfig returns default evaluator configuration
func DefaultConfig() *Config {
	return &Config{
		QualityThreshold: 0.8,
	}
}

// NewEvaluator creates a new quality evaluator
func NewEvaluator(cfg *Config) *Evaluator {
	if cfg == nil {
		cfg = DefaultConfig()
	}
	return &Evaluator{
		threshold: cfg.QualityThreshold,
		logger:    logging.New("aristoteles-quality"),
	}
}

// Evaluate evaluates the quality of the current pipeline state
func (e *Evaluator) Evaluate(ctx context.Context, pctx *pipeline.Context) (float32, bool) {
	var score float32 = 1.0

	// Check if we have intent
	if pctx.Intent == nil {
		score -= 0.2
	} else {
		// Higher score for higher confidence
		score = pctx.Intent.Confidence
	}

	// Check enrichments if required
	if pctx.Strategy != nil && pctx.Strategy.RequiresEnrichment {
		if len(pctx.Enrichments) == 0 {
			score -= 0.3
		} else {
			// Average relevance score from enrichments
			var relevanceSum float32
			for _, e := range pctx.Enrichments {
				if e.Success {
					relevanceSum += e.RelevanceScore
				}
			}
			if len(pctx.Enrichments) > 0 {
				avgRelevance := relevanceSum / float32(len(pctx.Enrichments))
				if avgRelevance < e.threshold {
					score -= (e.threshold - avgRelevance)
				}
			}
		}
	}

	// Clamp score
	if score < 0 {
		score = 0
	}
	if score > 1 {
		score = 1
	}

	return score, score >= e.threshold
}

// Stage is the quality evaluation pipeline stage
type Stage struct {
	evaluator *Evaluator
}

// NewStage creates a new quality stage
func NewStage(evaluator *Evaluator) *Stage {
	return &Stage{evaluator: evaluator}
}

// Name returns the stage name
func (s *Stage) Name() string {
	return "quality"
}

// Execute runs the quality evaluation stage
func (s *Stage) Execute(ctx context.Context, pctx *pipeline.Context) error {
	start := time.Now()

	score, sufficient := s.evaluator.Evaluate(ctx, pctx)
	pctx.Metrics.QualityScore = score

	s.evaluator.logger.Debug("Quality evaluated",
		"score", score,
		"sufficient", sufficient,
		"duration", time.Since(start))

	return nil
}
