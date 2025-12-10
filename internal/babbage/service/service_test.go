package service

import (
	"context"
	"strings"
	"testing"
)

func TestNewService(t *testing.T) {
	svc, err := NewService(Config{})
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}
	if svc == nil {
		t.Fatal("NewService() returned nil")
	}
}

func TestService_SetLLMFunc(t *testing.T) {
	svc, _ := NewService(Config{})

	svc.SetLLMFunc(func(ctx context.Context, prompt string) (string, error) {
		return "response", nil
	})

	if svc.llmFunc == nil {
		t.Error("llmFunc should be set")
	}

	// Verify function works
	result, err := svc.llmFunc(context.Background(), "test")
	if err != nil || result != "response" {
		t.Error("llmFunc should return response")
	}
}

func TestService_Analyze(t *testing.T) {
	svc, _ := NewService(Config{})
	ctx := context.Background()

	text := "Dies ist ein Test. Der Text enthält mehrere Sätze. Er ist auf Deutsch."
	result, err := svc.Analyze(ctx, text)

	if err != nil {
		t.Fatalf("Analyze() error = %v", err)
	}

	if result.WordCount == 0 {
		t.Error("WordCount should be > 0")
	}
	if result.CharCount != len(text) {
		t.Errorf("CharCount = %d, want %d", result.CharCount, len(text))
	}
	if result.Sentences < 3 {
		t.Errorf("Sentences = %d, want >= 3", result.Sentences)
	}
	if result.Language != "de" {
		t.Errorf("Language = %v, want de", result.Language)
	}
}

func TestService_Analyze_EmptyText(t *testing.T) {
	svc, _ := NewService(Config{})
	ctx := context.Background()

	_, err := svc.Analyze(ctx, "")
	if err == nil {
		t.Error("Analyze() should return error for empty text")
	}
}

func TestService_Summarize_Extractive(t *testing.T) {
	svc, _ := NewService(Config{})
	ctx := context.Background()

	text := "Erster Satz ist wichtig. Zweiter Satz folgt. Dritter Satz beendet den Text."
	req := &SummarizeRequest{
		Text:      text,
		MaxLength: 10,
	}

	summary, err := svc.Summarize(ctx, req)
	if err != nil {
		t.Fatalf("Summarize() error = %v", err)
	}

	if summary == "" {
		t.Error("Summary should not be empty")
	}
	if !strings.HasSuffix(summary, ".") {
		t.Error("Summary should end with period")
	}
}

func TestService_Summarize_EmptyText(t *testing.T) {
	svc, _ := NewService(Config{})
	ctx := context.Background()

	_, err := svc.Summarize(ctx, &SummarizeRequest{Text: ""})
	if err == nil {
		t.Error("Summarize() should return error for empty text")
	}
}

func TestService_Summarize_WithLLM(t *testing.T) {
	svc, _ := NewService(Config{})
	svc.SetLLMFunc(func(ctx context.Context, prompt string) (string, error) {
		return "LLM Zusammenfassung", nil
	})

	ctx := context.Background()
	summary, err := svc.Summarize(ctx, &SummarizeRequest{
		Text:      "Ein langer Text der zusammengefasst werden soll.",
		MaxLength: 20,
	})

	if err != nil {
		t.Fatalf("Summarize() error = %v", err)
	}
	if summary != "LLM Zusammenfassung" {
		t.Errorf("Summary = %v, want 'LLM Zusammenfassung'", summary)
	}
}

func TestService_Classify_NoLLM(t *testing.T) {
	svc, _ := NewService(Config{})
	ctx := context.Background()

	result, err := svc.Classify(ctx, &ClassifyRequest{
		Text:   "Ein Text zum Klassifizieren",
		Labels: []string{"Sport", "Politik", "Wirtschaft"},
	})

	if err != nil {
		t.Fatalf("Classify() error = %v", err)
	}

	// Without LLM, should return first label
	if result.Label != "Sport" {
		t.Errorf("Label = %v, want Sport", result.Label)
	}
}

func TestService_Classify_EmptyText(t *testing.T) {
	svc, _ := NewService(Config{})
	ctx := context.Background()

	_, err := svc.Classify(ctx, &ClassifyRequest{Text: "", Labels: []string{"A"}})
	if err == nil {
		t.Error("Classify() should return error for empty text")
	}
}

func TestService_Classify_NoLabels(t *testing.T) {
	svc, _ := NewService(Config{})
	ctx := context.Background()

	_, err := svc.Classify(ctx, &ClassifyRequest{Text: "Test", Labels: []string{}})
	if err == nil {
		t.Error("Classify() should return error for empty labels")
	}
}

func TestService_ExtractKeywords(t *testing.T) {
	svc, _ := NewService(Config{})
	ctx := context.Background()

	text := "Machine Learning und Deep Learning sind wichtige Themen. Künstliche Intelligenz verändert die Welt."
	keywords, err := svc.ExtractKeywords(ctx, text, 5)

	if err != nil {
		t.Fatalf("ExtractKeywords() error = %v", err)
	}
	if len(keywords) > 5 {
		t.Errorf("Keywords count = %d, want <= 5", len(keywords))
	}
}

func TestService_ExtractKeywords_EmptyText(t *testing.T) {
	svc, _ := NewService(Config{})
	ctx := context.Background()

	_, err := svc.ExtractKeywords(ctx, "", 10)
	if err == nil {
		t.Error("ExtractKeywords() should return error for empty text")
	}
}

func TestService_DetectLanguage(t *testing.T) {
	svc, _ := NewService(Config{})
	ctx := context.Background()

	tests := []struct {
		text     string
		expected string
	}{
		{"Dies ist ein deutscher Text mit Wörtern und Sätzen", "de"},
		{"This is an English text with words and sentences", "en"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			lang, err := svc.DetectLanguage(ctx, tt.text)
			if err != nil {
				t.Fatalf("DetectLanguage() error = %v", err)
			}
			if lang != tt.expected {
				t.Errorf("Language = %v, want %v", lang, tt.expected)
			}
		})
	}
}

func TestService_DetectLanguage_EmptyText(t *testing.T) {
	svc, _ := NewService(Config{})
	ctx := context.Background()

	_, err := svc.DetectLanguage(ctx, "")
	if err == nil {
		t.Error("DetectLanguage() should return error for empty text")
	}
}

func TestService_HealthCheck(t *testing.T) {
	svc, _ := NewService(Config{})
	err := svc.HealthCheck(context.Background())
	if err != nil {
		t.Errorf("HealthCheck() error = %v", err)
	}
}

func TestCountWords(t *testing.T) {
	tests := []struct {
		text     string
		expected int
	}{
		{"Ein Wort", 2},
		{"Eins Zwei Drei Vier", 4},
		{"", 0},
		{"   Leerzeichen   ", 1},
	}

	for _, tt := range tests {
		result := countWords(tt.text)
		if result != tt.expected {
			t.Errorf("countWords(%q) = %d, want %d", tt.text, result, tt.expected)
		}
	}
}

func TestCountSentences(t *testing.T) {
	tests := []struct {
		text     string
		expected int
	}{
		{"Ein Satz.", 1},
		{"Satz eins. Satz zwei.", 2},
		{"Frage? Antwort!", 2},
		{"Kein Satzzeichen", 1},
		{"", 0},
	}

	for _, tt := range tests {
		result := countSentences(tt.text)
		if result != tt.expected {
			t.Errorf("countSentences(%q) = %d, want %d", tt.text, result, tt.expected)
		}
	}
}

func TestDetectLanguage(t *testing.T) {
	tests := []struct {
		text     string
		expected string
	}{
		{"Der Mann und die Frau ist ein deutscher Text", "de"},
		{"The man and the woman is an English text", "en"},
		{"Eine Mischung with words", "en"}, // Algorithm finds "with" as English
	}

	for _, tt := range tests {
		result := detectLanguage(tt.text)
		if result != tt.expected {
			t.Errorf("detectLanguage(%q) = %v, want %v", tt.text, result, tt.expected)
		}
	}
}

func TestAnalyzeSentiment(t *testing.T) {
	tests := []struct {
		text     string
		expected SentimentLabel
	}{
		{"Das ist super toll und ausgezeichnet!", SentimentPositive},
		{"Das ist schrecklich und furchtbar.", SentimentNegative},
		{"Das ist ein normaler Text.", SentimentNeutral},
	}

	for _, tt := range tests {
		result := analyzeSentiment(tt.text)
		if result.Label != tt.expected {
			t.Errorf("analyzeSentiment(%q).Label = %v, want %v", tt.text, result.Label, tt.expected)
		}
	}
}

func TestExtractKeywords(t *testing.T) {
	text := "Machine Learning ist wichtig. Machine Learning verändert alles. Deep Learning auch."
	keywords := extractKeywords(text)

	if len(keywords) == 0 {
		t.Error("extractKeywords() should return keywords")
	}

	// "machine" should be in keywords (appears twice)
	found := false
	for _, kw := range keywords {
		if kw == "machine" || kw == "learning" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected 'machine' or 'learning' in keywords")
	}
}

func TestExtractEntities(t *testing.T) {
	text := "Berlin ist eine Stadt. Peter wohnt dort."
	entities := extractEntities(text)

	if len(entities) == 0 {
		t.Error("extractEntities() should return entities")
	}

	hasLocation := false
	hasPerson := false
	for _, e := range entities {
		if e.Text == "Berlin" && e.Type == "LOC" {
			hasLocation = true
		}
		if e.Text == "Peter" && e.Type == "PERSON" {
			hasPerson = true
		}
	}

	if !hasLocation {
		t.Error("Expected Berlin as LOC entity")
	}
	if !hasPerson {
		t.Error("Expected Peter as PERSON entity")
	}
}

func TestIsPersonName(t *testing.T) {
	if !isPersonName("Peter") {
		t.Error("Peter should be recognized as person name")
	}
	if isPersonName("Computer") {
		t.Error("Computer should not be recognized as person name")
	}
}

func TestIsOrganization(t *testing.T) {
	if !isOrganization("TestGmbH") {
		t.Error("TestGmbH should be recognized as organization")
	}
	if isOrganization("Test") {
		t.Error("Test should not be recognized as organization")
	}
}

func TestIsLocation(t *testing.T) {
	if !isLocation("Berlin") {
		t.Error("Berlin should be recognized as location")
	}
	if isLocation("Stuhl") {
		t.Error("Stuhl should not be recognized as location")
	}
}

func TestSentimentLabel_Constants(t *testing.T) {
	if SentimentPositive != "positive" {
		t.Errorf("SentimentPositive = %v, want positive", SentimentPositive)
	}
	if SentimentNegative != "negative" {
		t.Errorf("SentimentNegative = %v, want negative", SentimentNegative)
	}
	if SentimentNeutral != "neutral" {
		t.Errorf("SentimentNeutral = %v, want neutral", SentimentNeutral)
	}
}

func TestService_ExtractiveSummary(t *testing.T) {
	svc, _ := NewService(Config{})

	text := "Erster Satz. Zweiter Satz. Dritter Satz."
	summary := svc.extractiveSummary(text, 4)

	if summary == "" {
		t.Error("Summary should not be empty")
	}
	if !strings.HasSuffix(summary, ".") {
		t.Error("Summary should end with period")
	}
	// Should contain at least first sentence
	if !strings.Contains(summary, "Erster Satz") {
		t.Error("Summary should contain first sentence")
	}
}
