// Package intent provides tests for the intent analyzer
package intent

import (
	"context"
	"testing"
	"time"

	pb "github.com/msto63/mDW/api/gen/aristoteles"
)

func TestNewAnalyzer(t *testing.T) {
	cfg := DefaultConfig()
	analyzer := NewAnalyzer(cfg)

	if analyzer == nil {
		t.Fatal("NewAnalyzer returned nil")
	}

	if analyzer.model != cfg.Model {
		t.Errorf("Expected model=%s, got %s", cfg.Model, analyzer.model)
	}

	if analyzer.confidenceThreshold != cfg.ConfidenceThreshold {
		t.Errorf("Expected confidenceThreshold=%f, got %f", cfg.ConfidenceThreshold, analyzer.confidenceThreshold)
	}

	if analyzer.cache == nil {
		t.Error("Cache should be initialized")
	}
}

func TestNewAnalyzerWithNilConfig(t *testing.T) {
	analyzer := NewAnalyzer(nil)

	if analyzer == nil {
		t.Fatal("NewAnalyzer with nil config returned nil")
	}

	// Should use defaults
	if analyzer.model != "mistral:7b" {
		t.Errorf("Expected default model, got %s", analyzer.model)
	}
}

func TestAnalyzeLocalDirectLLM(t *testing.T) {
	analyzer := NewAnalyzer(nil)

	ctx := context.Background()
	// Avoid keywords that trigger other intents
	result, err := analyzer.Analyze(ctx, "Hello, how are you?", "")

	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}

	if result == nil {
		t.Fatal("Analyze returned nil result")
	}

	// Simple greeting should be DIRECT_LLM
	if result.Primary != pb.IntentType_INTENT_TYPE_DIRECT_LLM {
		t.Errorf("Expected DIRECT_LLM, got %s", result.Primary.String())
	}
}

func TestAnalyzeLocalCodeGeneration(t *testing.T) {
	analyzer := NewAnalyzer(nil)

	ctx := context.Background()
	result, err := analyzer.Analyze(ctx, "Write a Python function to sort a list", "")

	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}

	if result.Primary != pb.IntentType_INTENT_TYPE_CODE_GENERATION {
		t.Errorf("Expected CODE_GENERATION, got %s", result.Primary.String())
	}
}

func TestAnalyzeLocalCodeAnalysis(t *testing.T) {
	analyzer := NewAnalyzer(nil)

	ctx := context.Background()
	result, err := analyzer.Analyze(ctx, "Analyze this Python code and explain what it does", "")

	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}

	if result.Primary != pb.IntentType_INTENT_TYPE_CODE_ANALYSIS {
		t.Errorf("Expected CODE_ANALYSIS, got %s", result.Primary.String())
	}
}

func TestAnalyzeLocalWebResearch(t *testing.T) {
	analyzer := NewAnalyzer(nil)

	ctx := context.Background()
	result, err := analyzer.Analyze(ctx, "Search for the latest news about AI in 2025", "")

	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}

	if result.Primary != pb.IntentType_INTENT_TYPE_WEB_RESEARCH {
		t.Errorf("Expected WEB_RESEARCH, got %s", result.Primary.String())
	}
}

func TestAnalyzeLocalSummarization(t *testing.T) {
	analyzer := NewAnalyzer(nil)

	ctx := context.Background()
	result, err := analyzer.Analyze(ctx, "Please summarize this article for me", "")

	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}

	if result.Primary != pb.IntentType_INTENT_TYPE_SUMMARIZATION {
		t.Errorf("Expected SUMMARIZATION, got %s", result.Primary.String())
	}
}

func TestAnalyzeLocalTranslation(t *testing.T) {
	analyzer := NewAnalyzer(nil)

	ctx := context.Background()
	result, err := analyzer.Analyze(ctx, "Translate this text to German", "")

	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}

	if result.Primary != pb.IntentType_INTENT_TYPE_TRANSLATION {
		t.Errorf("Expected TRANSLATION, got %s", result.Primary.String())
	}
}

func TestAnalyzeLocalCreative(t *testing.T) {
	analyzer := NewAnalyzer(nil)

	ctx := context.Background()
	result, err := analyzer.Analyze(ctx, "Write a story about a magical forest", "")

	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}

	if result.Primary != pb.IntentType_INTENT_TYPE_CREATIVE {
		t.Errorf("Expected CREATIVE, got %s", result.Primary.String())
	}
}

func TestAnalyzeLocalRAG(t *testing.T) {
	analyzer := NewAnalyzer(nil)

	ctx := context.Background()
	// Use explicit RAG keywords without web search keywords
	result, err := analyzer.Analyze(ctx, "What does it say in the knowledge base about project X?", "")

	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}

	if result.Primary != pb.IntentType_INTENT_TYPE_RAG_QUERY {
		t.Errorf("Expected RAG_QUERY, got %s", result.Primary.String())
	}
}

func TestAnalyzeCaching(t *testing.T) {
	analyzer := NewAnalyzer(nil)

	ctx := context.Background()
	prompt := "Hello, world!"

	// First call - should analyze
	result1, err := analyzer.Analyze(ctx, prompt, "")
	if err != nil {
		t.Fatalf("First analyze returned error: %v", err)
	}

	// Second call with same prompt - should hit cache
	result2, err := analyzer.Analyze(ctx, prompt, "")
	if err != nil {
		t.Fatalf("Second analyze returned error: %v", err)
	}

	// Results should be the same
	if result1.Primary != result2.Primary {
		t.Error("Cached result should match original")
	}

	// Check that cache has an entry
	cacheKey := analyzer.getCacheKey(prompt)
	cached := analyzer.getFromCache(cacheKey)
	if cached == nil {
		t.Error("Expected cache entry to exist")
	}
}

func TestAnalyzeCacheTTL(t *testing.T) {
	analyzer := NewAnalyzer(nil)
	analyzer.cacheTTL = 1 * time.Millisecond // Very short TTL for testing

	ctx := context.Background()
	prompt := "Test TTL"

	// First call
	_, err := analyzer.Analyze(ctx, prompt, "")
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}

	// Wait for TTL to expire
	time.Sleep(10 * time.Millisecond)

	// Cache should be expired
	cacheKey := analyzer.getCacheKey(prompt)
	cached := analyzer.getFromCache(cacheKey)
	if cached != nil {
		t.Error("Expected cache entry to be expired")
	}
}

func TestAnalyzeCacheEviction(t *testing.T) {
	analyzer := NewAnalyzer(nil)
	analyzer.cacheSize = 2 // Small cache for testing

	ctx := context.Background()

	// Fill cache
	analyzer.Analyze(ctx, "Prompt 1", "")
	analyzer.Analyze(ctx, "Prompt 2", "")
	analyzer.Analyze(ctx, "Prompt 3", "") // Should trigger eviction

	// Cache should have at most 2 entries
	analyzer.cacheMu.RLock()
	size := len(analyzer.cache)
	analyzer.cacheMu.RUnlock()

	if size > 2 {
		t.Errorf("Expected cache size <= 2, got %d", size)
	}
}

func TestStringToIntentType(t *testing.T) {
	tests := []struct {
		input    string
		expected pb.IntentType
	}{
		{"DIRECT_LLM", pb.IntentType_INTENT_TYPE_DIRECT_LLM},
		{"direct", pb.IntentType_INTENT_TYPE_DIRECT_LLM},
		{"CODE_GENERATION", pb.IntentType_INTENT_TYPE_CODE_GENERATION},
		{"coding", pb.IntentType_INTENT_TYPE_CODE_GENERATION},
		{"WEB_RESEARCH", pb.IntentType_INTENT_TYPE_WEB_RESEARCH},
		{"search", pb.IntentType_INTENT_TYPE_WEB_RESEARCH},
		{"SUMMARIZATION", pb.IntentType_INTENT_TYPE_SUMMARIZATION},
		{"unknown_type", pb.IntentType_INTENT_TYPE_DIRECT_LLM}, // Default
	}

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			result := stringToIntentType(tc.input)
			if result != tc.expected {
				t.Errorf("stringToIntentType(%s) = %s, expected %s",
					tc.input, result.String(), tc.expected.String())
			}
		})
	}
}

func TestStringToComplexity(t *testing.T) {
	tests := []struct {
		input    string
		expected pb.ComplexityLevel
	}{
		{"SIMPLE", pb.ComplexityLevel_COMPLEXITY_SIMPLE},
		{"easy", pb.ComplexityLevel_COMPLEXITY_SIMPLE},
		{"MODERATE", pb.ComplexityLevel_COMPLEXITY_MODERATE},
		{"medium", pb.ComplexityLevel_COMPLEXITY_MODERATE},
		{"COMPLEX", pb.ComplexityLevel_COMPLEXITY_COMPLEX},
		{"hard", pb.ComplexityLevel_COMPLEXITY_COMPLEX},
		{"EXPERT", pb.ComplexityLevel_COMPLEXITY_EXPERT},
		{"unknown", pb.ComplexityLevel_COMPLEXITY_SIMPLE}, // Default
	}

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			result := stringToComplexity(tc.input)
			if result != tc.expected {
				t.Errorf("stringToComplexity(%s) = %s, expected %s",
					tc.input, result.String(), tc.expected.String())
			}
		})
	}
}

// TestAnalyzeLocalWebResearchEnglish tests English web research keywords
// Note: German and other languages are handled by the translation stage which
// translates prompts to English before intent analysis
func TestAnalyzeLocalWebResearchEnglish(t *testing.T) {
	analyzer := NewAnalyzer(nil)
	ctx := context.Background()

	testCases := []struct {
		name   string
		prompt string
	}{
		{"current", "What are the current news about AI?"},
		{"latest", "What is the latest on climate change?"},
		{"today", "What happened today in technology?"},
		{"recent", "Show me recent developments in AI"},
		{"search", "Search for information about quantum computing"},
		{"breaking", "What is the breaking news?"},
		{"happening now", "What is happening now in the market?"},
		{"this week", "What happened this week in politics?"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := analyzer.Analyze(ctx, tc.prompt, "")

			if err != nil {
				t.Fatalf("Analyze returned error: %v", err)
			}

			// Should detect WEB_RESEARCH as primary or secondary intent
			hasWebResearch := result.Primary == pb.IntentType_INTENT_TYPE_WEB_RESEARCH
			for _, sec := range result.Secondary {
				if sec == pb.IntentType_INTENT_TYPE_WEB_RESEARCH {
					hasWebResearch = true
					break
				}
			}

			if !hasWebResearch {
				t.Errorf("Expected WEB_RESEARCH intent for prompt %q, got primary=%s, secondary=%v",
					tc.prompt, result.Primary.String(), result.Secondary)
			}
		})
	}
}

// TestAnalyzeLocalWebResearchDynamicYear tests dynamic year detection
func TestAnalyzeLocalWebResearchDynamicYear(t *testing.T) {
	analyzer := NewAnalyzer(nil)
	ctx := context.Background()

	currentYear := time.Now().Year()
	testCases := []struct {
		name   string
		prompt string
	}{
		{"current year", "What were the major events in " + time.Now().Format("2006") + "?"},
		{"next year", "What are the predictions for " + time.Now().AddDate(1, 0, 0).Format("2006") + "?"},
		{"last year", "What happened in " + time.Now().AddDate(-1, 0, 0).Format("2006") + "?"},
	}

	// Add year to test case names for clarity
	_ = currentYear

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := analyzer.Analyze(ctx, tc.prompt, "")

			if err != nil {
				t.Fatalf("Analyze returned error: %v", err)
			}

			// Should detect WEB_RESEARCH
			hasWebResearch := result.Primary == pb.IntentType_INTENT_TYPE_WEB_RESEARCH
			for _, sec := range result.Secondary {
				if sec == pb.IntentType_INTENT_TYPE_WEB_RESEARCH {
					hasWebResearch = true
					break
				}
			}

			if !hasWebResearch {
				t.Errorf("Expected WEB_RESEARCH intent for prompt %q (year detection), got primary=%s",
					tc.prompt, result.Primary.String())
			}
		})
	}
}

// TestAnalyzeNoFalsePositiveWebResearch tests that non-research prompts don't trigger web research
func TestAnalyzeNoFalsePositiveWebResearch(t *testing.T) {
	analyzer := NewAnalyzer(nil)
	ctx := context.Background()

	// These should NOT trigger web research
	testCases := []struct {
		name            string
		prompt          string
		expectedPrimary pb.IntentType
	}{
		{"code generation", "Implement a new function in Python", pb.IntentType_INTENT_TYPE_CODE_GENERATION},
		{"greeting", "Hello, how are you?", pb.IntentType_INTENT_TYPE_DIRECT_LLM},
		{"creative", "Write a story about a knight", pb.IntentType_INTENT_TYPE_CREATIVE},
		{"summarize", "Summarize this article for me", pb.IntentType_INTENT_TYPE_SUMMARIZATION},
		{"translate", "Translate this text to German", pb.IntentType_INTENT_TYPE_TRANSLATION},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := analyzer.Analyze(ctx, tc.prompt, "")

			if err != nil {
				t.Fatalf("Analyze returned error: %v", err)
			}

			if result.Primary != tc.expectedPrimary {
				t.Errorf("Expected primary %s for prompt %q, got %s",
					tc.expectedPrimary.String(), tc.prompt, result.Primary.String())
			}

			// Verify web research is not the primary intent
			if result.Primary == pb.IntentType_INTENT_TYPE_WEB_RESEARCH {
				t.Errorf("Should NOT trigger WEB_RESEARCH as primary for prompt %q", tc.prompt)
			}
		})
	}
}
