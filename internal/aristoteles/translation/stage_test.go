// Package translation provides tests for the translation stage
package translation

import (
	"context"
	"errors"
	"testing"

	"github.com/msto63/mDW/internal/aristoteles/pipeline"
)

func TestNewStage(t *testing.T) {
	stage := NewStage(nil)
	if stage == nil {
		t.Fatal("NewStage returned nil")
	}
	if stage.model != "llama3.2:3b" {
		t.Errorf("Expected default model llama3.2:3b, got %s", stage.model)
	}
}

func TestNewStageWithConfig(t *testing.T) {
	cfg := &Config{Model: "custom-model"}
	stage := NewStage(cfg)
	if stage.model != "custom-model" {
		t.Errorf("Expected model custom-model, got %s", stage.model)
	}
}

func TestStageName(t *testing.T) {
	stage := NewStage(nil)
	if stage.Name() != "translation" {
		t.Errorf("Expected name 'translation', got '%s'", stage.Name())
	}
}

func TestDetectLanguageEnglish(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		expected string
	}{
		{"simple english", "Hello, how are you?", "en"},
		{"english question", "What is the latest news about AI?", "en"},
		{"english code request", "Write a Python function to sort a list", "en"},
		{"ascii only", "Search for current information", "en"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := detectLanguage(tt.text)
			if result != tt.expected {
				t.Errorf("detectLanguage(%q) = %s, expected %s", tt.text, result, tt.expected)
			}
		})
	}
}

func TestDetectLanguageGerman(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		expected string
	}{
		{"german with umlaut", "Was sind die aktuellen Nachrichten?", "de"},
		{"german question", "Wie ist das Wetter heute?", "de"},
		{"german greeting", "Hallo, wie geht es dir?", "de"},
		{"german with ß", "Ich weiß nicht", "de"},
		{"german article", "Der Mann ist groß", "de"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := detectLanguage(tt.text)
			if result != tt.expected {
				t.Errorf("detectLanguage(%q) = %s, expected %s", tt.text, result, tt.expected)
			}
		})
	}
}

func TestDetectLanguageFrench(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		expected string
	}{
		{"french with accent", "C'est très bien", "fr"},
		{"french question", "Où est la bibliothèque?", "fr"},
		{"french greeting with accent", "Bonjour, comment ça va?", "fr"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := detectLanguage(tt.text)
			if result != tt.expected {
				t.Errorf("detectLanguage(%q) = %s, expected %s", tt.text, result, tt.expected)
			}
		})
	}
}

func TestDetectLanguageSpanish(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		expected string
	}{
		{"spanish with ñ", "El niño está jugando", "es"},
		{"spanish question", "¿Dónde está el banco?", "es"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := detectLanguage(tt.text)
			if result != tt.expected {
				t.Errorf("detectLanguage(%q) = %s, expected %s", tt.text, result, tt.expected)
			}
		})
	}
}

func TestIsASCIIText(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		expected bool
	}{
		{"ascii only", "Hello World", true},
		{"with numbers", "Test 123", true},
		{"with punctuation", "Hello, World!", true},
		{"with umlaut", "Hällö", false},
		{"with accent", "café", false},
		{"with special char", "niño", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isASCIIText(tt.text)
			if result != tt.expected {
				t.Errorf("isASCIIText(%q) = %v, expected %v", tt.text, result, tt.expected)
			}
		})
	}
}

func TestExecuteEnglishSkipsTranslation(t *testing.T) {
	stage := NewStage(nil)
	ctx := context.Background()
	pctx := pipeline.NewContext("test-1", "Hello, how are you?", "", nil, nil)

	err := stage.Execute(ctx, pctx)
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}

	if pctx.SourceLanguage != "en" {
		t.Errorf("Expected SourceLanguage 'en', got '%s'", pctx.SourceLanguage)
	}

	// For English, PromptForAnalysis should be same as Prompt
	if pctx.PromptForAnalysis != pctx.Prompt {
		t.Errorf("Expected PromptForAnalysis to equal Prompt for English")
	}
}

func TestExecuteWithoutLLMFallsBack(t *testing.T) {
	stage := NewStage(nil)
	// No LLM func set
	ctx := context.Background()
	pctx := pipeline.NewContext("test-1", "Was sind die aktuellen Nachrichten?", "", nil, nil)

	err := stage.Execute(ctx, pctx)
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}

	if pctx.SourceLanguage != "de" {
		t.Errorf("Expected SourceLanguage 'de', got '%s'", pctx.SourceLanguage)
	}

	// Without LLM, should fall back to original prompt
	if pctx.PromptForAnalysis != pctx.Prompt {
		t.Errorf("Expected PromptForAnalysis to equal Prompt when no LLM available")
	}
}

func TestExecuteWithMockLLM(t *testing.T) {
	stage := NewStage(nil)

	// Set up mock LLM that returns translated text
	stage.SetLLMFunc(func(ctx context.Context, model, systemPrompt, userPrompt string) (string, error) {
		return "What are the current news?", nil
	})

	ctx := context.Background()
	pctx := pipeline.NewContext("test-1", "Was sind die aktuellen Nachrichten?", "", nil, nil)

	err := stage.Execute(ctx, pctx)
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}

	if pctx.SourceLanguage != "de" {
		t.Errorf("Expected SourceLanguage 'de', got '%s'", pctx.SourceLanguage)
	}

	if pctx.PromptForAnalysis != "What are the current news?" {
		t.Errorf("Expected translated prompt, got '%s'", pctx.PromptForAnalysis)
	}

	// Original prompt should be unchanged
	if pctx.Prompt != "Was sind die aktuellen Nachrichten?" {
		t.Errorf("Original prompt should be unchanged")
	}
}

func TestExecuteWithLLMError(t *testing.T) {
	stage := NewStage(nil)

	// Set up mock LLM that returns an error
	stage.SetLLMFunc(func(ctx context.Context, model, systemPrompt, userPrompt string) (string, error) {
		return "", context.DeadlineExceeded
	})

	ctx := context.Background()
	pctx := pipeline.NewContext("test-1", "Was sind die aktuellen Nachrichten?", "", nil, nil)

	// Should not return error - gracefully falls back
	err := stage.Execute(ctx, pctx)
	if err != nil {
		t.Fatalf("Execute should not return error on LLM failure: %v", err)
	}

	// Should fall back to original prompt
	if pctx.PromptForAnalysis != pctx.Prompt {
		t.Errorf("Expected fallback to original prompt on LLM error")
	}
}

func TestTranslateToEnglishCleansResponse(t *testing.T) {
	stage := NewStage(nil)

	// Mock LLM that returns response with quotes
	stage.SetLLMFunc(func(ctx context.Context, model, systemPrompt, userPrompt string) (string, error) {
		return "  \"What are the current news?\"  ", nil
	})

	ctx := context.Background()
	result, err := stage.translateToEnglish(ctx, "Was sind die aktuellen Nachrichten?")
	if err != nil {
		t.Fatalf("translateToEnglish returned error: %v", err)
	}

	expected := "What are the current news?"
	if result != expected {
		t.Errorf("Expected cleaned response '%s', got '%s'", expected, result)
	}
}

// Additional Unit Tests

func TestSetLLMFunc(t *testing.T) {
	stage := NewStage(nil)

	if stage.llmFunc != nil {
		t.Error("llmFunc should be nil initially")
	}

	called := false
	stage.SetLLMFunc(func(ctx context.Context, model, systemPrompt, userPrompt string) (string, error) {
		called = true
		return "test", nil
	})

	if stage.llmFunc == nil {
		t.Error("llmFunc should be set after SetLLMFunc")
	}

	// Verify the function is callable
	_, _ = stage.llmFunc(context.Background(), "", "", "")
	if !called {
		t.Error("llmFunc was not called")
	}
}

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg == nil {
		t.Fatal("DefaultConfig returned nil")
	}

	if cfg.Model != "llama3.2:3b" {
		t.Errorf("Expected default model llama3.2:3b, got %s", cfg.Model)
	}
}

func TestDetectLanguageMixedContent(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		expected string
	}{
		// Mixed content tests
		{"english with code", "Write a Python function def hello():", "en"},
		{"german with english words", "Ich möchte ein Python Script erstellen", "de"},
		{"mostly ascii with one umlaut", "Hello Wörld", "de"},
		{"numbers only", "12345", "en"},
		{"empty string", "", "en"},
		{"punctuation only", "!?.,;:", "en"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := detectLanguage(tt.text)
			if result != tt.expected {
				t.Errorf("detectLanguage(%q) = %s, expected %s", tt.text, result, tt.expected)
			}
		})
	}
}

func TestDetectLanguageEdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		expected string
	}{
		// Edge cases
		{"single german word", "Übung", "de"},
		{"single french word", "café", "fr"},
		{"single spanish word", "niño", "es"},
		{"URL-like text", "https://example.com/path", "en"},
		{"email-like text", "user@example.com", "en"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := detectLanguage(tt.text)
			if result != tt.expected {
				t.Errorf("detectLanguage(%q) = %s, expected %s", tt.text, result, tt.expected)
			}
		})
	}
}

func TestExecutePreservesOriginalPrompt(t *testing.T) {
	stage := NewStage(nil)

	originalPrompt := "Was sind die aktuellen Nachrichten?"
	translatedPrompt := "What are the current news?"

	stage.SetLLMFunc(func(ctx context.Context, model, systemPrompt, userPrompt string) (string, error) {
		return translatedPrompt, nil
	})

	ctx := context.Background()
	pctx := pipeline.NewContext("test-1", originalPrompt, "", nil, nil)

	err := stage.Execute(ctx, pctx)
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}

	// Original prompt must remain unchanged
	if pctx.Prompt != originalPrompt {
		t.Errorf("Original prompt was modified: expected %q, got %q", originalPrompt, pctx.Prompt)
	}

	// PromptForAnalysis should be the translation
	if pctx.PromptForAnalysis != translatedPrompt {
		t.Errorf("PromptForAnalysis incorrect: expected %q, got %q", translatedPrompt, pctx.PromptForAnalysis)
	}
}

func TestExecuteWithContextCancellation(t *testing.T) {
	stage := NewStage(nil)

	stage.SetLLMFunc(func(ctx context.Context, model, systemPrompt, userPrompt string) (string, error) {
		return "", ctx.Err()
	})

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	pctx := pipeline.NewContext("test-1", "Was sind die Nachrichten?", "", nil, nil)

	// Should not return error - gracefully falls back
	err := stage.Execute(ctx, pctx)
	if err != nil {
		t.Fatalf("Execute should not return error on context cancellation: %v", err)
	}

	// Should fall back to original prompt
	if pctx.PromptForAnalysis != pctx.Prompt {
		t.Error("Should fall back to original prompt on context cancellation")
	}
}

func TestExecuteWithVariousLLMErrors(t *testing.T) {
	tests := []struct {
		name string
		err  error
	}{
		{"timeout error", context.DeadlineExceeded},
		{"cancelled error", context.Canceled},
		{"generic error", errors.New("LLM service unavailable")},
		{"connection error", errors.New("connection refused")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stage := NewStage(nil)
			stage.SetLLMFunc(func(ctx context.Context, model, systemPrompt, userPrompt string) (string, error) {
				return "", tt.err
			})

			ctx := context.Background()
			pctx := pipeline.NewContext("test-1", "Deutsche Nachricht", "", nil, nil)

			// Should not fail the pipeline
			err := stage.Execute(ctx, pctx)
			if err != nil {
				t.Errorf("Execute should not return error for %s", tt.name)
			}

			// Should fall back to original
			if pctx.PromptForAnalysis != pctx.Prompt {
				t.Errorf("Should fall back to original prompt for %s", tt.name)
			}
		})
	}
}

func TestTranslateToEnglishUsesCorrectModel(t *testing.T) {
	customModel := "custom-translation-model"
	stage := NewStage(&Config{Model: customModel})

	var usedModel string
	stage.SetLLMFunc(func(ctx context.Context, model, systemPrompt, userPrompt string) (string, error) {
		usedModel = model
		return "translated", nil
	})

	ctx := context.Background()
	_, err := stage.translateToEnglish(ctx, "Test text")
	if err != nil {
		t.Fatalf("translateToEnglish returned error: %v", err)
	}

	if usedModel != customModel {
		t.Errorf("Expected model %s, got %s", customModel, usedModel)
	}
}

func TestTranslateToEnglishSystemPrompt(t *testing.T) {
	stage := NewStage(nil)

	var capturedSystemPrompt string
	stage.SetLLMFunc(func(ctx context.Context, model, systemPrompt, userPrompt string) (string, error) {
		capturedSystemPrompt = systemPrompt
		return "translated", nil
	})

	ctx := context.Background()
	_, _ = stage.translateToEnglish(ctx, "Test")

	// Verify system prompt contains key instructions
	if capturedSystemPrompt == "" {
		t.Error("System prompt should not be empty")
	}

	expectedPhrases := []string{"translator", "English", "translation"}
	for _, phrase := range expectedPhrases {
		found := false
		if len(capturedSystemPrompt) > 0 {
			for i := 0; i < len(capturedSystemPrompt)-len(phrase)+1; i++ {
				if capturedSystemPrompt[i:i+len(phrase)] == phrase {
					found = true
					break
				}
			}
		}
		// Use contains check
		if !found && !containsString(capturedSystemPrompt, phrase) {
			t.Errorf("System prompt should contain '%s'", phrase)
		}
	}
}

// Helper function
func containsString(s, substr string) bool {
	for i := 0; i < len(s)-len(substr)+1; i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestMultipleLanguageTranslations(t *testing.T) {
	stage := NewStage(nil)

	// Use prompts with clear language indicators
	translations := map[string]string{
		"Guten Morgen, wie geht es dir heute?": "Good morning, how are you today?",
		"Bonjour, où est la bibliothèque?":     "Hello, where is the library?",
		"Hola, ¿dónde está el banco?":          "Hello, where is the bank?",
	}

	stage.SetLLMFunc(func(ctx context.Context, model, systemPrompt, userPrompt string) (string, error) {
		if trans, ok := translations[userPrompt]; ok {
			return trans, nil
		}
		return userPrompt, nil
	})

	ctx := context.Background()

	for original, expected := range translations {
		pctx := pipeline.NewContext("test", original, "", nil, nil)

		err := stage.Execute(ctx, pctx)
		if err != nil {
			t.Fatalf("Execute failed for %q: %v", original, err)
		}

		if pctx.PromptForAnalysis != expected {
			t.Errorf("Translation of %q: expected %q, got %q", original, expected, pctx.PromptForAnalysis)
		}

		// Original should be preserved
		if pctx.Prompt != original {
			t.Errorf("Original prompt was modified for %q", original)
		}
	}
}
