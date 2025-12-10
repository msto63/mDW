package service

import (
	"context"
	"fmt"
	"strings"
	"unicode"

	"github.com/msto63/mDW/pkg/core/logging"
)

// SentimentLabel represents sentiment analysis result
type SentimentLabel string

const (
	SentimentPositive SentimentLabel = "positive"
	SentimentNegative SentimentLabel = "negative"
	SentimentNeutral  SentimentLabel = "neutral"
)

// Sentiment represents sentiment analysis result
type Sentiment struct {
	Label SentimentLabel
	Score float64
}

// Entity represents a named entity
type Entity struct {
	Text  string
	Type  string
	Start int
	End   int
}

// AnalysisResult represents the result of text analysis
type AnalysisResult struct {
	Sentiment  *Sentiment
	Entities   []Entity
	Keywords   []string
	Language   string
	WordCount  int
	CharCount  int
	Sentences  int
}

// SummarizeRequest represents a summarization request
type SummarizeRequest struct {
	Text      string
	MaxLength int
	Style     string // "brief", "detailed", "bullet"
}

// ClassifyRequest represents a classification request
type ClassifyRequest struct {
	Text   string
	Labels []string
}

// ClassifyResult represents classification result
type ClassifyResult struct {
	Label string
	Score float64
}

// LLMFunc is a function for LLM-based operations
type LLMFunc func(ctx context.Context, prompt string) (string, error)

// Service is the Babbage NLP service
type Service struct {
	logger  *logging.Logger
	llmFunc LLMFunc
}

// Config holds service configuration
type Config struct {
	LLMFunc LLMFunc
}

// NewService creates a new Babbage service
func NewService(cfg Config) (*Service, error) {
	logger := logging.New("babbage")

	return &Service{
		logger:  logger,
		llmFunc: cfg.LLMFunc,
	}, nil
}

// SetLLMFunc sets the LLM function
func (s *Service) SetLLMFunc(fn LLMFunc) {
	s.llmFunc = fn
}

// Analyze performs comprehensive text analysis
func (s *Service) Analyze(ctx context.Context, text string) (*AnalysisResult, error) {
	if text == "" {
		return nil, fmt.Errorf("text is required")
	}

	s.logger.Info("Analyzing text", "length", len(text))

	result := &AnalysisResult{
		WordCount:  countWords(text),
		CharCount:  len(text),
		Sentences:  countSentences(text),
		Language:   detectLanguage(text),
		Keywords:   extractKeywords(text),
		Entities:   extractEntities(text),
		Sentiment:  analyzeSentiment(text),
	}

	return result, nil
}

// Summarize generates a summary of the text
func (s *Service) Summarize(ctx context.Context, req *SummarizeRequest) (string, error) {
	if req.Text == "" {
		return "", fmt.Errorf("text is required")
	}

	if s.llmFunc == nil {
		// Fallback to simple extractive summary
		return s.extractiveSummary(req.Text, req.MaxLength), nil
	}

	maxLength := req.MaxLength
	if maxLength <= 0 {
		maxLength = 100
	}

	style := req.Style
	if style == "" {
		style = "brief"
	}

	prompt := fmt.Sprintf(
		"Fasse den folgenden Text in maximal %d Wörtern zusammen. Stil: %s.\n\nText:\n%s",
		maxLength, style, req.Text,
	)

	return s.llmFunc(ctx, prompt)
}

// Classify classifies text into given labels
func (s *Service) Classify(ctx context.Context, req *ClassifyRequest) (*ClassifyResult, error) {
	if req.Text == "" {
		return nil, fmt.Errorf("text is required")
	}
	if len(req.Labels) == 0 {
		return nil, fmt.Errorf("labels are required")
	}

	if s.llmFunc == nil {
		// Return first label as fallback
		return &ClassifyResult{
			Label: req.Labels[0],
			Score: 0.5,
		}, nil
	}

	labels := strings.Join(req.Labels, ", ")
	prompt := fmt.Sprintf(
		"Klassifiziere den folgenden Text in eine der Kategorien: %s.\n"+
			"Antworte nur mit dem Kategorienamen.\n\nText:\n%s",
		labels, req.Text,
	)

	result, err := s.llmFunc(ctx, prompt)
	if err != nil {
		return nil, err
	}

	// Clean up result
	result = strings.TrimSpace(result)
	result = strings.ToLower(result)

	// Match to labels
	for _, label := range req.Labels {
		if strings.Contains(strings.ToLower(label), result) ||
			strings.Contains(result, strings.ToLower(label)) {
			return &ClassifyResult{
				Label: label,
				Score: 0.9,
			}, nil
		}
	}

	return &ClassifyResult{
		Label: result,
		Score: 0.7,
	}, nil
}

// ExtractKeywords extracts keywords from text
func (s *Service) ExtractKeywords(ctx context.Context, text string, maxKeywords int) ([]string, error) {
	if text == "" {
		return nil, fmt.Errorf("text is required")
	}

	if maxKeywords <= 0 {
		maxKeywords = 10
	}

	keywords := extractKeywords(text)
	if len(keywords) > maxKeywords {
		keywords = keywords[:maxKeywords]
	}

	return keywords, nil
}

// DetectLanguage detects the language of the text
func (s *Service) DetectLanguage(ctx context.Context, text string) (string, error) {
	if text == "" {
		return "", fmt.Errorf("text is required")
	}
	return detectLanguage(text), nil
}

// TranslateRequest represents a translation request
type TranslateRequest struct {
	Text           string
	SourceLanguage string
	TargetLanguage string
}

// TranslateResult represents a translation result
type TranslateResult struct {
	TranslatedText   string
	SourceLanguage   string
	TargetLanguage   string
	DetectedLanguage string
}

// Translate translates text between languages using LLM
func (s *Service) Translate(ctx context.Context, req *TranslateRequest) (*TranslateResult, error) {
	if req.Text == "" {
		return nil, fmt.Errorf("text is required")
	}
	if req.TargetLanguage == "" {
		return nil, fmt.Errorf("target_language is required")
	}

	// Detect source language if not provided
	sourceLanguage := req.SourceLanguage
	if sourceLanguage == "" {
		sourceLanguage = detectLanguage(req.Text)
	}

	// If source and target are the same, return the original text
	if sourceLanguage == req.TargetLanguage {
		return &TranslateResult{
			TranslatedText:   req.Text,
			SourceLanguage:   sourceLanguage,
			TargetLanguage:   req.TargetLanguage,
			DetectedLanguage: sourceLanguage,
		}, nil
	}

	// Use LLM for translation
	if s.llmFunc == nil {
		return nil, fmt.Errorf("LLM function not available for translation")
	}

	targetLangName := getLanguageName(req.TargetLanguage)
	sourceLangName := getLanguageName(sourceLanguage)

	prompt := fmt.Sprintf(
		"Übersetze den folgenden Text von %s nach %s. "+
			"Gib nur die Übersetzung zurück, ohne Erklärungen.\n\n"+
			"Text:\n%s",
		sourceLangName, targetLangName, req.Text,
	)

	translated, err := s.llmFunc(ctx, prompt)
	if err != nil {
		return nil, fmt.Errorf("translation failed: %w", err)
	}

	return &TranslateResult{
		TranslatedText:   strings.TrimSpace(translated),
		SourceLanguage:   sourceLanguage,
		TargetLanguage:   req.TargetLanguage,
		DetectedLanguage: sourceLanguage,
	}, nil
}

// getLanguageName returns the full name of a language code
func getLanguageName(code string) string {
	names := map[string]string{
		"de": "Deutsch",
		"en": "Englisch",
		"fr": "Französisch",
		"es": "Spanisch",
		"it": "Italienisch",
		"pt": "Portugiesisch",
		"nl": "Niederländisch",
		"pl": "Polnisch",
		"ru": "Russisch",
		"zh": "Chinesisch",
		"ja": "Japanisch",
		"ko": "Koreanisch",
	}
	if name, ok := names[code]; ok {
		return name
	}
	return code
}

// HealthCheck checks if the service is healthy
func (s *Service) HealthCheck(ctx context.Context) error {
	return nil
}

// Helper functions

func countWords(text string) int {
	return len(strings.Fields(text))
}

func countSentences(text string) int {
	count := 0
	for _, r := range text {
		if r == '.' || r == '!' || r == '?' {
			count++
		}
	}
	if count == 0 && len(text) > 0 {
		count = 1
	}
	return count
}

func detectLanguage(text string) string {
	// Simple heuristic based on common words
	lowerText := strings.ToLower(text)

	// German indicators
	germanWords := []string{"und", "der", "die", "das", "ist", "ein", "eine", "nicht", "mit", "für"}
	germanCount := 0
	for _, word := range germanWords {
		if strings.Contains(lowerText, " "+word+" ") {
			germanCount++
		}
	}

	// English indicators
	englishWords := []string{"the", "and", "is", "are", "was", "were", "have", "has", "with", "for"}
	englishCount := 0
	for _, word := range englishWords {
		if strings.Contains(lowerText, " "+word+" ") {
			englishCount++
		}
	}

	if germanCount > englishCount {
		return "de"
	}
	return "en"
}

func extractKeywords(text string) []string {
	// Simple keyword extraction based on word frequency
	words := strings.Fields(strings.ToLower(text))
	wordCount := make(map[string]int)

	stopWords := map[string]bool{
		// English
		"the": true, "a": true, "an": true, "and": true, "or": true, "but": true,
		"in": true, "on": true, "at": true, "to": true, "for": true, "of": true,
		"with": true, "by": true, "from": true, "is": true, "are": true, "was": true,
		// German
		"der": true, "die": true, "das": true, "und": true, "oder": true, "aber": true,
		"auf": true, "zu": true, "für": true, "von": true,
		"mit": true, "bei": true, "aus": true, "ist": true, "sind": true, "war": true,
	}

	for _, word := range words {
		// Clean word
		word = strings.Trim(word, ".,!?;:\"'()[]{}")
		if len(word) < 3 || stopWords[word] {
			continue
		}
		wordCount[word]++
	}

	// Sort by frequency
	type wordFreq struct {
		word  string
		count int
	}
	var sorted []wordFreq
	for w, c := range wordCount {
		sorted = append(sorted, wordFreq{w, c})
	}
	for i := 0; i < len(sorted)-1; i++ {
		for j := i + 1; j < len(sorted); j++ {
			if sorted[j].count > sorted[i].count {
				sorted[i], sorted[j] = sorted[j], sorted[i]
			}
		}
	}

	// Take top keywords
	var keywords []string
	for i := 0; i < len(sorted) && i < 20; i++ {
		keywords = append(keywords, sorted[i].word)
	}

	return keywords
}

func extractEntities(text string) []Entity {
	var entities []Entity

	// Simple entity extraction: find capitalized words
	words := strings.Fields(text)
	pos := 0

	for _, word := range words {
		cleanWord := strings.Trim(word, ".,!?;:\"'()[]{}")
		if len(cleanWord) > 1 && unicode.IsUpper(rune(cleanWord[0])) {
			// Check if it's not at start of sentence
			start := strings.Index(text[pos:], cleanWord)
			if start >= 0 {
				entityType := "MISC"
				// Simple heuristics
				if isPersonName(cleanWord) {
					entityType = "PERSON"
				} else if isOrganization(cleanWord) {
					entityType = "ORG"
				} else if isLocation(cleanWord) {
					entityType = "LOC"
				}

				entities = append(entities, Entity{
					Text:  cleanWord,
					Type:  entityType,
					Start: pos + start,
					End:   pos + start + len(cleanWord),
				})
			}
		}
		pos += len(word) + 1
	}

	return entities
}

func analyzeSentiment(text string) *Sentiment {
	// Simple sentiment analysis based on word lists
	lowerText := strings.ToLower(text)

	positiveWords := []string{
		"gut", "super", "toll", "ausgezeichnet", "fantastisch", "wunderbar",
		"good", "great", "excellent", "amazing", "wonderful", "fantastic",
		"love", "happy", "best", "perfect", "beautiful", "awesome",
	}
	negativeWords := []string{
		"schlecht", "schrecklich", "furchtbar", "miserabel", "enttäuschend",
		"bad", "terrible", "awful", "horrible", "disappointing", "worst",
		"hate", "sad", "ugly", "poor", "failure", "problem",
	}

	posCount := 0
	negCount := 0

	for _, word := range positiveWords {
		posCount += strings.Count(lowerText, word)
	}
	for _, word := range negativeWords {
		negCount += strings.Count(lowerText, word)
	}

	if posCount > negCount {
		return &Sentiment{
			Label: SentimentPositive,
			Score: float64(posCount) / float64(posCount+negCount+1),
		}
	} else if negCount > posCount {
		return &Sentiment{
			Label: SentimentNegative,
			Score: float64(negCount) / float64(posCount+negCount+1),
		}
	}

	return &Sentiment{
		Label: SentimentNeutral,
		Score: 0.5,
	}
}

func isPersonName(word string) bool {
	// Simple heuristic
	commonNames := map[string]bool{
		"Anna": true, "Peter": true, "Michael": true, "Sarah": true,
		"John": true, "Mary": true, "James": true, "David": true,
	}
	return commonNames[word]
}

func isOrganization(word string) bool {
	suffixes := []string{"GmbH", "AG", "Inc", "Corp", "LLC", "Ltd"}
	for _, suffix := range suffixes {
		if strings.HasSuffix(word, suffix) {
			return true
		}
	}
	return false
}

func isLocation(word string) bool {
	locations := map[string]bool{
		"Berlin": true, "München": true, "Hamburg": true, "Frankfurt": true,
		"London": true, "Paris": true, "New": true, "York": true,
	}
	return locations[word]
}

func (s *Service) extractiveSummary(text string, maxWords int) string {
	sentences := strings.Split(text, ".")
	var summary strings.Builder
	wordCount := 0

	for _, sentence := range sentences {
		sentence = strings.TrimSpace(sentence)
		if sentence == "" {
			continue
		}

		words := len(strings.Fields(sentence))
		if wordCount+words <= maxWords {
			if summary.Len() > 0 {
				summary.WriteString(". ")
			}
			summary.WriteString(sentence)
			wordCount += words
		} else {
			break
		}
	}

	result := summary.String()
	if !strings.HasSuffix(result, ".") {
		result += "."
	}
	return result
}
