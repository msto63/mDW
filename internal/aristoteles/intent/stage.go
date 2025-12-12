package intent

import (
	"context"
	"time"

	"github.com/msto63/mDW/internal/aristoteles/pipeline"
)

// Stage is the intent analysis pipeline stage
type Stage struct {
	analyzer *Analyzer
}

// NewStage creates a new intent stage
func NewStage(analyzer *Analyzer) *Stage {
	return &Stage{analyzer: analyzer}
}

// Name returns the stage name
func (s *Stage) Name() string {
	return "intent"
}

// Execute runs the intent analysis stage
func (s *Stage) Execute(ctx context.Context, pctx *pipeline.Context) error {
	// Skip if disabled
	if pctx.Options != nil && pctx.Options.SkipIntentAnalysis {
		return nil
	}

	start := time.Now()

	// Use translated prompt for analysis if available, otherwise use original
	// This enables language-agnostic intent analysis (all analysis in English)
	promptToAnalyze := pctx.PromptForAnalysis
	if promptToAnalyze == "" {
		promptToAnalyze = pctx.Prompt
	}

	result, err := s.analyzer.Analyze(ctx, promptToAnalyze, pctx.ConversationID)
	if err != nil {
		return err
	}

	pctx.Intent = result
	pctx.Metrics.IntentDurationMs = time.Since(start).Milliseconds()

	return nil
}
