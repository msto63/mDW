// Package intent provides tests for the intent stage
package intent

import (
	"context"
	"testing"

	pb "github.com/msto63/mDW/api/gen/aristoteles"
	"github.com/msto63/mDW/internal/aristoteles/pipeline"
)

func TestNewStage(t *testing.T) {
	analyzer := NewAnalyzer(nil)
	stage := NewStage(analyzer)

	if stage == nil {
		t.Fatal("NewStage returned nil")
	}

	if stage.analyzer != analyzer {
		t.Error("Stage analyzer not set correctly")
	}
}

func TestStageName(t *testing.T) {
	stage := NewStage(NewAnalyzer(nil))

	if stage.Name() != "intent" {
		t.Errorf("Expected stage name 'intent', got '%s'", stage.Name())
	}
}

func TestStageExecuteWithPrompt(t *testing.T) {
	analyzer := NewAnalyzer(nil)
	stage := NewStage(analyzer)

	ctx := context.Background()
	pctx := pipeline.NewContext("test-1", "Hello, how are you?", "", nil, nil)

	err := stage.Execute(ctx, pctx)
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}

	if pctx.Intent == nil {
		t.Fatal("Intent should be set after Execute")
	}

	// Simple greeting should be DIRECT_LLM
	if pctx.Intent.Primary != pb.IntentType_INTENT_TYPE_DIRECT_LLM {
		t.Errorf("Expected DIRECT_LLM, got %s", pctx.Intent.Primary.String())
	}
}

func TestStageExecuteWithPromptForAnalysis(t *testing.T) {
	analyzer := NewAnalyzer(nil)
	stage := NewStage(analyzer)

	ctx := context.Background()

	// German original prompt, English translated prompt for analysis
	pctx := pipeline.NewContext("test-1", "Was sind die aktuellen Nachrichten?", "", nil, nil)
	pctx.PromptForAnalysis = "What are the current news?"
	pctx.SourceLanguage = "de"

	err := stage.Execute(ctx, pctx)
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}

	if pctx.Intent == nil {
		t.Fatal("Intent should be set after Execute")
	}

	// Should detect WEB_RESEARCH based on English translation (contains "current", "news")
	if pctx.Intent.Primary != pb.IntentType_INTENT_TYPE_WEB_RESEARCH {
		t.Errorf("Expected WEB_RESEARCH based on translated prompt, got %s", pctx.Intent.Primary.String())
	}
}

func TestStageExecuteUsesPromptForAnalysisOverPrompt(t *testing.T) {
	analyzer := NewAnalyzer(nil)
	stage := NewStage(analyzer)

	ctx := context.Background()

	// Original prompt would be detected as code generation
	// But translated prompt should be detected as web research
	pctx := pipeline.NewContext("test-1", "Implementiere eine Python Funktion", "", nil, nil)
	pctx.PromptForAnalysis = "What is the current weather?"
	pctx.SourceLanguage = "de"

	err := stage.Execute(ctx, pctx)
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}

	// Should use PromptForAnalysis, so WEB_RESEARCH (contains "current")
	if pctx.Intent.Primary != pb.IntentType_INTENT_TYPE_WEB_RESEARCH {
		t.Errorf("Expected WEB_RESEARCH from PromptForAnalysis, got %s", pctx.Intent.Primary.String())
	}
}

func TestStageExecuteFallsBackToPromptWhenNoPromptForAnalysis(t *testing.T) {
	analyzer := NewAnalyzer(nil)
	stage := NewStage(analyzer)

	ctx := context.Background()

	// No PromptForAnalysis set - should fall back to Prompt
	pctx := pipeline.NewContext("test-1", "Write a Python function to sort a list", "", nil, nil)

	err := stage.Execute(ctx, pctx)
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}

	// Should use Prompt, so CODE_GENERATION
	if pctx.Intent.Primary != pb.IntentType_INTENT_TYPE_CODE_GENERATION {
		t.Errorf("Expected CODE_GENERATION from Prompt fallback, got %s", pctx.Intent.Primary.String())
	}
}

func TestStageExecuteSkipsWhenDisabled(t *testing.T) {
	analyzer := NewAnalyzer(nil)
	stage := NewStage(analyzer)

	ctx := context.Background()
	pctx := pipeline.NewContext("test-1", "Hello", "", nil, &pb.ProcessOptions{
		SkipIntentAnalysis: true,
	})

	err := stage.Execute(ctx, pctx)
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}

	// Intent should not be set when skipped
	if pctx.Intent != nil {
		t.Error("Intent should be nil when analysis is skipped")
	}
}

func TestStageExecuteSetsMetrics(t *testing.T) {
	analyzer := NewAnalyzer(nil)
	stage := NewStage(analyzer)

	ctx := context.Background()
	pctx := pipeline.NewContext("test-1", "Hello", "", nil, nil)

	err := stage.Execute(ctx, pctx)
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}

	// IntentDurationMs can be 0 if execution is very fast (sub-millisecond)
	// Just verify Metrics exists and was initialized
	if pctx.Metrics == nil {
		t.Error("Metrics should be initialized")
	}
}

func TestStageExecuteWithEmptyPromptForAnalysis(t *testing.T) {
	analyzer := NewAnalyzer(nil)
	stage := NewStage(analyzer)

	ctx := context.Background()

	// Empty PromptForAnalysis should fall back to Prompt
	pctx := pipeline.NewContext("test-1", "Search for latest news", "", nil, nil)
	pctx.PromptForAnalysis = "" // Explicitly empty

	err := stage.Execute(ctx, pctx)
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}

	// Should use Prompt, so WEB_RESEARCH
	if pctx.Intent.Primary != pb.IntentType_INTENT_TYPE_WEB_RESEARCH {
		t.Errorf("Expected WEB_RESEARCH from Prompt, got %s", pctx.Intent.Primary.String())
	}
}

func TestStageExecuteVariousIntentTypes(t *testing.T) {
	analyzer := NewAnalyzer(nil)
	stage := NewStage(analyzer)

	tests := []struct {
		name              string
		promptForAnalysis string
		expectedIntent    pb.IntentType
	}{
		{
			name:              "web research",
			promptForAnalysis: "What is the current stock price?",
			expectedIntent:    pb.IntentType_INTENT_TYPE_WEB_RESEARCH,
		},
		{
			name:              "code generation",
			promptForAnalysis: "Write a Python function to calculate fibonacci",
			expectedIntent:    pb.IntentType_INTENT_TYPE_CODE_GENERATION,
		},
		{
			name:              "summarization",
			promptForAnalysis: "Please summarize this article for me",
			expectedIntent:    pb.IntentType_INTENT_TYPE_SUMMARIZATION,
		},
		{
			name:              "translation",
			promptForAnalysis: "Translate this text to German",
			expectedIntent:    pb.IntentType_INTENT_TYPE_TRANSLATION,
		},
		{
			name:              "creative",
			promptForAnalysis: "Write a story about a magical forest",
			expectedIntent:    pb.IntentType_INTENT_TYPE_CREATIVE,
		},
		{
			name:              "direct llm",
			promptForAnalysis: "Hello, how are you doing?",
			expectedIntent:    pb.IntentType_INTENT_TYPE_DIRECT_LLM,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			// Use German original, English analysis prompt
			pctx := pipeline.NewContext("test", "Deutsche Anfrage", "", nil, nil)
			pctx.PromptForAnalysis = tt.promptForAnalysis
			pctx.SourceLanguage = "de"

			err := stage.Execute(ctx, pctx)
			if err != nil {
				t.Fatalf("Execute returned error: %v", err)
			}

			if pctx.Intent.Primary != tt.expectedIntent {
				t.Errorf("Expected %s, got %s", tt.expectedIntent.String(), pctx.Intent.Primary.String())
			}
		})
	}
}

func TestStageExecutePreservesOriginalPrompt(t *testing.T) {
	analyzer := NewAnalyzer(nil)
	stage := NewStage(analyzer)

	ctx := context.Background()

	originalPrompt := "Was sind die aktuellen Nachrichten über KI?"
	translatedPrompt := "What are the current news about AI?"

	pctx := pipeline.NewContext("test-1", originalPrompt, "", nil, nil)
	pctx.PromptForAnalysis = translatedPrompt
	pctx.SourceLanguage = "de"

	err := stage.Execute(ctx, pctx)
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}

	// Original prompt must remain unchanged
	if pctx.Prompt != originalPrompt {
		t.Errorf("Original prompt was modified: expected %q, got %q", originalPrompt, pctx.Prompt)
	}

	// PromptForAnalysis should remain unchanged
	if pctx.PromptForAnalysis != translatedPrompt {
		t.Errorf("PromptForAnalysis was modified: expected %q, got %q", translatedPrompt, pctx.PromptForAnalysis)
	}

	// SourceLanguage should remain unchanged
	if pctx.SourceLanguage != "de" {
		t.Errorf("SourceLanguage was modified: expected 'de', got %q", pctx.SourceLanguage)
	}
}
