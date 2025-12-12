// Package aristoteles provides integration tests for translation pipeline
package aristoteles

import (
	"context"
	"testing"

	pb "github.com/msto63/mDW/api/gen/aristoteles"
	"github.com/msto63/mDW/internal/aristoteles/intent"
	"github.com/msto63/mDW/internal/aristoteles/pipeline"
	"github.com/msto63/mDW/internal/aristoteles/translation"
)

// TestTranslationIntentPipelineIntegration tests the integration of Translation and Intent stages
func TestTranslationIntentPipelineIntegration(t *testing.T) {
	// Create stages
	translationStage := translation.NewStage(nil)
	intentAnalyzer := intent.NewAnalyzer(nil)
	intentStage := intent.NewStage(intentAnalyzer)

	// Mock translation function
	translationStage.SetLLMFunc(func(ctx context.Context, model, systemPrompt, userPrompt string) (string, error) {
		// Simulate translations
		translations := map[string]string{
			"Was sind die aktuellen Nachrichten?":                "What are the current news?",
			"Wie ist das aktuelle Wetter?":                       "What is the current weather?",
			"Ich möchte eine Python Funktion schreiben":          "Write a Python function",
			"Übersetze diesen Text ins Englische":                "Translate this text to English",
			"Bitte fasse diesen Artikel für mich zusammen":       "Please summarize this article for me",
			"Schreibe mir eine Geschichte über einen Ritter":     "Write a story about a knight",
			"Hallo, wie geht es dir?":                            "Hello, how are you?",
		}
		if trans, ok := translations[userPrompt]; ok {
			return trans, nil
		}
		return userPrompt, nil
	})

	tests := []struct {
		name           string
		prompt         string
		expectedIntent pb.IntentType
		expectedLang   string
	}{
		{
			name:           "German web research -> WEB_RESEARCH",
			prompt:         "Was sind die aktuellen Nachrichten?",
			expectedIntent: pb.IntentType_INTENT_TYPE_WEB_RESEARCH,
			expectedLang:   "de",
		},
		{
			name:           "German weather query -> WEB_RESEARCH",
			prompt:         "Wie ist das aktuelle Wetter?",
			expectedIntent: pb.IntentType_INTENT_TYPE_WEB_RESEARCH,
			expectedLang:   "de",
		},
		{
			name:           "German code request -> CODE_GENERATION",
			prompt:         "Ich möchte eine Python Funktion schreiben",
			expectedIntent: pb.IntentType_INTENT_TYPE_CODE_GENERATION,
			expectedLang:   "de",
		},
		{
			name:           "German translation request -> TRANSLATION",
			prompt:         "Übersetze diesen Text ins Englische",
			expectedIntent: pb.IntentType_INTENT_TYPE_TRANSLATION,
			expectedLang:   "de",
		},
		{
			name:           "German summary request -> SUMMARIZATION",
			prompt:         "Bitte fasse diesen Artikel für mich zusammen",
			expectedIntent: pb.IntentType_INTENT_TYPE_SUMMARIZATION,
			expectedLang:   "de",
		},
		{
			name:           "German creative request -> CREATIVE",
			prompt:         "Schreibe mir eine Geschichte über einen Ritter",
			expectedIntent: pb.IntentType_INTENT_TYPE_CREATIVE,
			expectedLang:   "de",
		},
		{
			name:           "German greeting -> DIRECT_LLM",
			prompt:         "Hallo, wie geht es dir?",
			expectedIntent: pb.IntentType_INTENT_TYPE_DIRECT_LLM,
			expectedLang:   "de",
		},
		{
			name:           "English web research -> WEB_RESEARCH",
			prompt:         "What are the current news?",
			expectedIntent: pb.IntentType_INTENT_TYPE_WEB_RESEARCH,
			expectedLang:   "en",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			pctx := pipeline.NewContext("test-"+tt.name, tt.prompt, "", nil, nil)

			// Execute translation stage
			err := translationStage.Execute(ctx, pctx)
			if err != nil {
				t.Fatalf("Translation stage failed: %v", err)
			}

			// Verify language detection
			if pctx.SourceLanguage != tt.expectedLang {
				t.Errorf("Expected language %s, got %s", tt.expectedLang, pctx.SourceLanguage)
			}

			// Execute intent stage
			err = intentStage.Execute(ctx, pctx)
			if err != nil {
				t.Fatalf("Intent stage failed: %v", err)
			}

			// Verify intent
			if pctx.Intent.Primary != tt.expectedIntent {
				t.Errorf("Expected intent %s, got %s", tt.expectedIntent.String(), pctx.Intent.Primary.String())
			}

			// Verify original prompt preserved
			if pctx.Prompt != tt.prompt {
				t.Errorf("Original prompt modified: expected %q, got %q", tt.prompt, pctx.Prompt)
			}
		})
	}
}

// TestTranslationPipelinePreservesOriginalPrompt tests that original prompt is always preserved
func TestTranslationPipelinePreservesOriginalPrompt(t *testing.T) {
	translationStage := translation.NewStage(nil)
	intentAnalyzer := intent.NewAnalyzer(nil)
	intentStage := intent.NewStage(intentAnalyzer)

	translationStage.SetLLMFunc(func(ctx context.Context, model, systemPrompt, userPrompt string) (string, error) {
		return "Translated: " + userPrompt, nil
	})

	originalPrompts := []string{
		"Was sind die aktuellen Nachrichten über KI?",
		"Wie funktioniert maschinelles Lernen?",
		"Erkläre mir die Relativitätstheorie",
		"Was ist der Sinn des Lebens?",
	}

	for _, original := range originalPrompts {
		t.Run(original[:20], func(t *testing.T) {
			ctx := context.Background()
			pctx := pipeline.NewContext("test", original, "", nil, nil)

			// Execute both stages
			_ = translationStage.Execute(ctx, pctx)
			_ = intentStage.Execute(ctx, pctx)

			// Original MUST be unchanged
			if pctx.Prompt != original {
				t.Errorf("Original prompt was modified!\nExpected: %q\nGot: %q", original, pctx.Prompt)
			}
		})
	}
}

// TestTranslationPipelineWithoutLLM tests fallback behavior when no LLM is available
func TestTranslationPipelineWithoutLLM(t *testing.T) {
	translationStage := translation.NewStage(nil)
	// No LLM function set
	intentAnalyzer := intent.NewAnalyzer(nil)
	intentStage := intent.NewStage(intentAnalyzer)

	ctx := context.Background()

	// German prompt without LLM translation available
	pctx := pipeline.NewContext("test", "Was sind die aktuellen Nachrichten?", "", nil, nil)

	// Execute translation stage - should fall back to original
	err := translationStage.Execute(ctx, pctx)
	if err != nil {
		t.Fatalf("Translation stage failed: %v", err)
	}

	// Without LLM, PromptForAnalysis should be same as Prompt
	if pctx.PromptForAnalysis != pctx.Prompt {
		t.Errorf("Without LLM, PromptForAnalysis should equal Prompt")
	}

	// Execute intent stage
	err = intentStage.Execute(ctx, pctx)
	if err != nil {
		t.Fatalf("Intent stage failed: %v", err)
	}

	// Intent should still be detected (based on German keywords still partially working)
	// But since we removed German keywords, it should be DIRECT_LLM
	if pctx.Intent == nil {
		t.Error("Intent should be set")
	}
}

// TestTranslationPipelineEnglishPassthrough tests that English prompts are passed through without translation
func TestTranslationPipelineEnglishPassthrough(t *testing.T) {
	translationStage := translation.NewStage(nil)

	// LLM should NOT be called for English
	llmCalled := false
	translationStage.SetLLMFunc(func(ctx context.Context, model, systemPrompt, userPrompt string) (string, error) {
		llmCalled = true
		return "Should not be called", nil
	})

	intentAnalyzer := intent.NewAnalyzer(nil)
	intentStage := intent.NewStage(intentAnalyzer)

	ctx := context.Background()

	englishPrompts := []string{
		"What are the current news?",
		"Write a Python function to sort a list",
		"Hello, how are you doing?",
		"Search for information about AI",
	}

	for _, prompt := range englishPrompts {
		// Safely truncate for test name
		testName := prompt
		if len(testName) > 20 {
			testName = testName[:20]
		}
		t.Run(testName, func(t *testing.T) {
			llmCalled = false
			pctx := pipeline.NewContext("test", prompt, "", nil, nil)

			// Execute both stages
			_ = translationStage.Execute(ctx, pctx)
			_ = intentStage.Execute(ctx, pctx)

			// LLM should NOT have been called for English
			if llmCalled {
				t.Errorf("LLM was called for English prompt: %q", prompt)
			}

			// SourceLanguage should be English
			if pctx.SourceLanguage != "en" {
				t.Errorf("Expected SourceLanguage 'en', got %q", pctx.SourceLanguage)
			}

			// PromptForAnalysis should be same as Prompt
			if pctx.PromptForAnalysis != pctx.Prompt {
				t.Errorf("For English, PromptForAnalysis should equal Prompt")
			}
		})
	}
}

// TestTranslationPipelineWithLLMError tests graceful degradation on LLM errors
func TestTranslationPipelineWithLLMError(t *testing.T) {
	translationStage := translation.NewStage(nil)
	translationStage.SetLLMFunc(func(ctx context.Context, model, systemPrompt, userPrompt string) (string, error) {
		return "", context.DeadlineExceeded // Simulate timeout
	})

	intentAnalyzer := intent.NewAnalyzer(nil)
	intentStage := intent.NewStage(intentAnalyzer)

	ctx := context.Background()
	pctx := pipeline.NewContext("test", "Was sind die Nachrichten?", "", nil, nil)

	// Execute translation stage - should NOT fail
	err := translationStage.Execute(ctx, pctx)
	if err != nil {
		t.Fatalf("Translation stage should not fail on LLM error: %v", err)
	}

	// Should fall back to original prompt
	if pctx.PromptForAnalysis != pctx.Prompt {
		t.Error("Should fall back to original prompt on LLM error")
	}

	// Intent stage should still work
	err = intentStage.Execute(ctx, pctx)
	if err != nil {
		t.Fatalf("Intent stage failed: %v", err)
	}

	if pctx.Intent == nil {
		t.Error("Intent should be set even after translation fallback")
	}
}

// TestFullPipelineEngineWithTranslation tests the complete pipeline engine
func TestFullPipelineEngineWithTranslation(t *testing.T) {
	// Create pipeline engine
	engine := pipeline.NewEngine(nil)

	// Create and configure stages
	translationStage := translation.NewStage(nil)
	translationStage.SetLLMFunc(func(ctx context.Context, model, systemPrompt, userPrompt string) (string, error) {
		if userPrompt == "Was sind die aktuellen Nachrichten?" {
			return "What are the current news?", nil
		}
		return userPrompt, nil
	})

	intentAnalyzer := intent.NewAnalyzer(nil)
	intentStage := intent.NewStage(intentAnalyzer)

	// Add stages in order
	engine.AddStage(translationStage)
	engine.AddStage(intentStage)

	ctx := context.Background()
	pctx := pipeline.NewContext("test-full", "Was sind die aktuellen Nachrichten?", "", nil, nil)

	// Execute full pipeline
	err := engine.Execute(ctx, pctx)
	if err != nil {
		t.Fatalf("Pipeline execution failed: %v", err)
	}

	// Verify results
	if pctx.SourceLanguage != "de" {
		t.Errorf("Expected SourceLanguage 'de', got %q", pctx.SourceLanguage)
	}

	if pctx.PromptForAnalysis != "What are the current news?" {
		t.Errorf("Expected translated prompt, got %q", pctx.PromptForAnalysis)
	}

	if pctx.Intent == nil {
		t.Fatal("Intent should be set")
	}

	if pctx.Intent.Primary != pb.IntentType_INTENT_TYPE_WEB_RESEARCH {
		t.Errorf("Expected WEB_RESEARCH, got %s", pctx.Intent.Primary.String())
	}

	// Original prompt must be preserved
	if pctx.Prompt != "Was sind die aktuellen Nachrichten?" {
		t.Errorf("Original prompt was modified")
	}
}

// TestMultiLanguageIntentDetection tests intent detection for multiple languages
func TestMultiLanguageIntentDetection(t *testing.T) {
	translationStage := translation.NewStage(nil)
	translationStage.SetLLMFunc(func(ctx context.Context, model, systemPrompt, userPrompt string) (string, error) {
		translations := map[string]string{
			// German - with clear indicators (umlauts, articles)
			"Was sind die aktuellen Nachrichten über KI?":          "What are the current news about AI?",
			"Ich möchte eine Python Funktion für das Sortieren":    "Write a Python function for sorting",
			// French - with accents and articles
			"Quelles sont les nouvelles actuelles sur l'économie?":            "What are the current news about the economy?",
			"Je voudrais écrire une fonction pour trier les données":          "Write a function to sort the data",
			// Spanish - with special chars and articles
			"¿Cuáles son las noticias actuales sobre el clima?":    "What are the current news about the climate?",
			"Quiero escribir código en Python":                     "Write code in Python",
		}
		if trans, ok := translations[userPrompt]; ok {
			return trans, nil
		}
		return userPrompt, nil
	})

	intentAnalyzer := intent.NewAnalyzer(nil)
	intentStage := intent.NewStage(intentAnalyzer)

	tests := []struct {
		name           string
		prompt         string
		expectedIntent pb.IntentType
		expectedLang   string
	}{
		// German - clear German indicators
		{"German news", "Was sind die aktuellen Nachrichten über KI?", pb.IntentType_INTENT_TYPE_WEB_RESEARCH, "de"},
		{"German code", "Ich möchte eine Python Funktion für das Sortieren", pb.IntentType_INTENT_TYPE_CODE_GENERATION, "de"},
		// French - clear French indicators (accents, articles)
		{"French news", "Quelles sont les nouvelles actuelles sur l'économie?", pb.IntentType_INTENT_TYPE_WEB_RESEARCH, "fr"},
		{"French code", "Je voudrais écrire une fonction pour trier les données", pb.IntentType_INTENT_TYPE_CODE_GENERATION, "fr"},
		// Spanish - clear Spanish indicators (¿, accents)
		{"Spanish news", "¿Cuáles son las noticias actuales sobre el clima?", pb.IntentType_INTENT_TYPE_WEB_RESEARCH, "es"},
		{"Spanish code", "Quiero escribir código en Python", pb.IntentType_INTENT_TYPE_CODE_GENERATION, "es"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			pctx := pipeline.NewContext("test", tt.prompt, "", nil, nil)

			_ = translationStage.Execute(ctx, pctx)
			_ = intentStage.Execute(ctx, pctx)

			if pctx.SourceLanguage != tt.expectedLang {
				t.Errorf("Expected language %s, got %s", tt.expectedLang, pctx.SourceLanguage)
			}

			if pctx.Intent.Primary != tt.expectedIntent {
				t.Errorf("Expected intent %s, got %s", tt.expectedIntent.String(), pctx.Intent.Primary.String())
			}
		})
	}
}

// TestPipelineContextFields tests that all translation-related context fields work correctly
func TestPipelineContextFields(t *testing.T) {
	pctx := pipeline.NewContext("test", "Original prompt", "conv-123", nil, nil)

	// Initially empty
	if pctx.PromptForAnalysis != "" {
		t.Error("PromptForAnalysis should be empty initially")
	}
	if pctx.SourceLanguage != "" {
		t.Error("SourceLanguage should be empty initially")
	}

	// Set values
	pctx.PromptForAnalysis = "Translated prompt"
	pctx.SourceLanguage = "de"

	// Verify values
	if pctx.PromptForAnalysis != "Translated prompt" {
		t.Error("PromptForAnalysis not set correctly")
	}
	if pctx.SourceLanguage != "de" {
		t.Error("SourceLanguage not set correctly")
	}

	// Original should be unchanged
	if pctx.Prompt != "Original prompt" {
		t.Error("Prompt should be unchanged")
	}
}

// BenchmarkTranslationPipeline benchmarks the translation pipeline performance
func BenchmarkTranslationPipeline(b *testing.B) {
	translationStage := translation.NewStage(nil)
	translationStage.SetLLMFunc(func(ctx context.Context, model, systemPrompt, userPrompt string) (string, error) {
		return "What are the current news?", nil
	})

	intentAnalyzer := intent.NewAnalyzer(nil)
	intentStage := intent.NewStage(intentAnalyzer)

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		pctx := pipeline.NewContext("bench", "Was sind die aktuellen Nachrichten?", "", nil, nil)
		_ = translationStage.Execute(ctx, pctx)
		_ = intentStage.Execute(ctx, pctx)
	}
}

// BenchmarkEnglishPassthrough benchmarks English passthrough (no translation needed)
func BenchmarkEnglishPassthrough(b *testing.B) {
	translationStage := translation.NewStage(nil)
	// No LLM needed for English

	intentAnalyzer := intent.NewAnalyzer(nil)
	intentStage := intent.NewStage(intentAnalyzer)

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		pctx := pipeline.NewContext("bench", "What are the current news?", "", nil, nil)
		_ = translationStage.Execute(ctx, pctx)
		_ = intentStage.Execute(ctx, pctx)
	}
}
