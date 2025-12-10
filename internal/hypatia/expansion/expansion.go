// ============================================================================
// meinDENKWERK (mDW) - Lokale KI-Plattform
// ============================================================================
//
// Package:     expansion
// Description: Query expansion for improved RAG retrieval
// Author:      Mike Stoffels with Claude
// Created:     2025-12-06
// License:     MIT
// ============================================================================

package expansion

import (
	"context"
	"fmt"
	"strings"

	"github.com/msto63/mDW/pkg/core/logging"
)

// LLMFunc is a function that generates text from a prompt
type LLMFunc func(ctx context.Context, prompt string) (string, error)

// ExpansionResult represents the result of query expansion
type ExpansionResult struct {
	OriginalQuery   string
	ExpandedQueries []string
	Synonyms        []string
	RelatedTerms    []string
}

// Expander interface for different expansion strategies
type Expander interface {
	// Expand expands the query into multiple variants
	Expand(ctx context.Context, query string) (*ExpansionResult, error)
}

// Config holds configuration for query expansion
type Config struct {
	// EnableSynonyms enables synonym-based expansion
	EnableSynonyms bool

	// EnableLLM enables LLM-based expansion
	EnableLLM bool

	// MaxExpandedQueries limits the number of expanded queries
	MaxExpandedQueries int

	// Language for synonym expansion (de, en)
	Language string

	// LLMFunc for generating expanded queries
	LLMFunc LLMFunc
}

// DefaultConfig returns default configuration
func DefaultConfig() Config {
	return Config{
		EnableSynonyms:     true,
		EnableLLM:          false, // LLM expansion requires function
		MaxExpandedQueries: 5,
		Language:           "de",
	}
}

// MultiStrategyExpander combines multiple expansion strategies
type MultiStrategyExpander struct {
	config  Config
	logger  *logging.Logger
	llmFunc LLMFunc
}

// NewMultiStrategyExpander creates a new multi-strategy expander
func NewMultiStrategyExpander(cfg Config) *MultiStrategyExpander {
	return &MultiStrategyExpander{
		config:  cfg,
		logger:  logging.New("query-expander"),
		llmFunc: cfg.LLMFunc,
	}
}

// SetLLMFunc sets the LLM function for expansion
func (e *MultiStrategyExpander) SetLLMFunc(fn LLMFunc) {
	e.llmFunc = fn
	e.config.EnableLLM = fn != nil
}

// Expand expands the query using all enabled strategies
func (e *MultiStrategyExpander) Expand(ctx context.Context, query string) (*ExpansionResult, error) {
	result := &ExpansionResult{
		OriginalQuery:   query,
		ExpandedQueries: []string{query}, // Always include original
		Synonyms:        []string{},
		RelatedTerms:    []string{},
	}

	e.logger.Info("Expanding query", "query", query)

	// Apply synonym expansion
	if e.config.EnableSynonyms {
		synonymExpanded := e.expandWithSynonyms(query)
		result.Synonyms = synonymExpanded.Synonyms
		result.ExpandedQueries = append(result.ExpandedQueries, synonymExpanded.ExpandedQueries...)
	}

	// Apply LLM expansion
	if e.config.EnableLLM && e.llmFunc != nil {
		llmExpanded, err := e.expandWithLLM(ctx, query)
		if err != nil {
			e.logger.Warn("LLM expansion failed", "error", err)
		} else {
			result.RelatedTerms = llmExpanded.RelatedTerms
			result.ExpandedQueries = append(result.ExpandedQueries, llmExpanded.ExpandedQueries...)
		}
	}

	// Deduplicate and limit
	result.ExpandedQueries = deduplicateStrings(result.ExpandedQueries)
	if len(result.ExpandedQueries) > e.config.MaxExpandedQueries {
		result.ExpandedQueries = result.ExpandedQueries[:e.config.MaxExpandedQueries]
	}

	e.logger.Info("Query expanded",
		"original", query,
		"expanded_count", len(result.ExpandedQueries),
	)

	return result, nil
}

// expandWithSynonyms expands the query using synonym dictionaries
func (e *MultiStrategyExpander) expandWithSynonyms(query string) *ExpansionResult {
	result := &ExpansionResult{
		OriginalQuery:   query,
		ExpandedQueries: []string{},
		Synonyms:        []string{},
	}

	// Tokenize query
	words := tokenize(query)
	if len(words) == 0 {
		return result
	}

	// Find synonyms for each word
	synonymMap := make(map[string][]string)
	for _, word := range words {
		syns := getSynonyms(word, e.config.Language)
		if len(syns) > 0 {
			synonymMap[word] = syns
			result.Synonyms = append(result.Synonyms, syns...)
		}
	}

	// Generate expanded queries by substituting synonyms
	if len(synonymMap) > 0 {
		expanded := generateSynonymVariants(query, words, synonymMap, 3)
		result.ExpandedQueries = expanded
	}

	return result
}

// expandWithLLM expands the query using LLM
func (e *MultiStrategyExpander) expandWithLLM(ctx context.Context, query string) (*ExpansionResult, error) {
	result := &ExpansionResult{
		OriginalQuery:   query,
		ExpandedQueries: []string{},
		RelatedTerms:    []string{},
	}

	prompt := fmt.Sprintf(`Given the search query: "%s"

Generate 3 alternative search queries that would find similar relevant documents.
Also list 3-5 related terms or concepts.

Format your response exactly as:
QUERIES:
1. [first alternative query]
2. [second alternative query]
3. [third alternative query]

RELATED TERMS:
- [term1]
- [term2]
- [term3]`, query)

	response, err := e.llmFunc(ctx, prompt)
	if err != nil {
		return nil, err
	}

	// Parse response
	lines := strings.Split(response, "\n")
	inQueries := false
	inTerms := false

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		if strings.HasPrefix(strings.ToUpper(line), "QUERIES") {
			inQueries = true
			inTerms = false
			continue
		}
		if strings.HasPrefix(strings.ToUpper(line), "RELATED") {
			inQueries = false
			inTerms = true
			continue
		}

		if inQueries {
			// Parse numbered query
			query := parseNumberedLine(line)
			if query != "" {
				result.ExpandedQueries = append(result.ExpandedQueries, query)
			}
		} else if inTerms {
			// Parse bullet point term
			term := parseBulletLine(line)
			if term != "" {
				result.RelatedTerms = append(result.RelatedTerms, term)
			}
		}
	}

	return result, nil
}

// SynonymExpander is a simple synonym-based expander
type SynonymExpander struct {
	language string
	logger   *logging.Logger
}

// NewSynonymExpander creates a new synonym expander
func NewSynonymExpander(language string) *SynonymExpander {
	return &SynonymExpander{
		language: language,
		logger:   logging.New("synonym-expander"),
	}
}

// Expand expands the query using synonyms
func (e *SynonymExpander) Expand(ctx context.Context, query string) (*ExpansionResult, error) {
	result := &ExpansionResult{
		OriginalQuery:   query,
		ExpandedQueries: []string{query},
		Synonyms:        []string{},
	}

	words := tokenize(query)
	synonymMap := make(map[string][]string)

	for _, word := range words {
		syns := getSynonyms(word, e.language)
		if len(syns) > 0 {
			synonymMap[word] = syns
			result.Synonyms = append(result.Synonyms, syns...)
		}
	}

	if len(synonymMap) > 0 {
		expanded := generateSynonymVariants(query, words, synonymMap, 3)
		result.ExpandedQueries = append(result.ExpandedQueries, expanded...)
	}

	return result, nil
}

// LLMExpander uses an LLM for query expansion
type LLMExpander struct {
	llmFunc          LLMFunc
	logger           *logging.Logger
	maxQueries       int
	expansionPrompt  string
}

// NewLLMExpander creates a new LLM-based expander
func NewLLMExpander(llmFunc LLMFunc, maxQueries int) *LLMExpander {
	return &LLMExpander{
		llmFunc:    llmFunc,
		logger:     logging.New("llm-expander"),
		maxQueries: maxQueries,
		expansionPrompt: `Generate %d alternative search queries for: "%s"
Respond with one query per line, no numbering or bullets.`,
	}
}

// Expand expands the query using LLM
func (e *LLMExpander) Expand(ctx context.Context, query string) (*ExpansionResult, error) {
	result := &ExpansionResult{
		OriginalQuery:   query,
		ExpandedQueries: []string{query},
	}

	if e.llmFunc == nil {
		return result, nil
	}

	prompt := fmt.Sprintf(e.expansionPrompt, e.maxQueries, query)
	response, err := e.llmFunc(ctx, prompt)
	if err != nil {
		return nil, err
	}

	// Parse response
	lines := strings.Split(response, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" && line != query {
			// Remove any numbering or bullets
			cleaned := parseNumberedLine(line)
			if cleaned == "" {
				cleaned = parseBulletLine(line)
			}
			if cleaned == "" {
				cleaned = line
			}
			result.ExpandedQueries = append(result.ExpandedQueries, cleaned)
		}
	}

	return result, nil
}

// HypothesisExpander generates hypothetical answers and uses them as queries
type HypothesisExpander struct {
	llmFunc    LLMFunc
	logger     *logging.Logger
	maxHypotheses int
}

// NewHypothesisExpander creates a new hypothesis-based expander (HyDE)
func NewHypothesisExpander(llmFunc LLMFunc, maxHypotheses int) *HypothesisExpander {
	return &HypothesisExpander{
		llmFunc:       llmFunc,
		logger:        logging.New("hyde-expander"),
		maxHypotheses: maxHypotheses,
	}
}

// Expand generates hypothetical document snippets
func (e *HypothesisExpander) Expand(ctx context.Context, query string) (*ExpansionResult, error) {
	result := &ExpansionResult{
		OriginalQuery:   query,
		ExpandedQueries: []string{query},
	}

	if e.llmFunc == nil {
		return result, nil
	}

	prompt := fmt.Sprintf(`Write a brief paragraph (2-3 sentences) that would be a good answer to the question: "%s"
This text should contain key terms and concepts that would appear in a relevant document.`, query)

	response, err := e.llmFunc(ctx, prompt)
	if err != nil {
		return nil, err
	}

	// Add the hypothesis as a search query
	response = strings.TrimSpace(response)
	if response != "" {
		result.ExpandedQueries = append(result.ExpandedQueries, response)
	}

	return result, nil
}

// CompositeExpander chains multiple expanders
type CompositeExpander struct {
	expanders []Expander
	logger    *logging.Logger
	maxTotal  int
}

// NewCompositeExpander creates a composite expander
func NewCompositeExpander(maxTotal int, expanders ...Expander) *CompositeExpander {
	return &CompositeExpander{
		expanders: expanders,
		logger:    logging.New("composite-expander"),
		maxTotal:  maxTotal,
	}
}

// Expand runs all expanders and combines results
func (e *CompositeExpander) Expand(ctx context.Context, query string) (*ExpansionResult, error) {
	combined := &ExpansionResult{
		OriginalQuery:   query,
		ExpandedQueries: []string{query},
		Synonyms:        []string{},
		RelatedTerms:    []string{},
	}

	for _, expander := range e.expanders {
		result, err := expander.Expand(ctx, query)
		if err != nil {
			e.logger.Warn("Expander failed", "error", err)
			continue
		}

		combined.ExpandedQueries = append(combined.ExpandedQueries, result.ExpandedQueries...)
		combined.Synonyms = append(combined.Synonyms, result.Synonyms...)
		combined.RelatedTerms = append(combined.RelatedTerms, result.RelatedTerms...)
	}

	// Deduplicate
	combined.ExpandedQueries = deduplicateStrings(combined.ExpandedQueries)
	combined.Synonyms = deduplicateStrings(combined.Synonyms)
	combined.RelatedTerms = deduplicateStrings(combined.RelatedTerms)

	// Limit
	if e.maxTotal > 0 && len(combined.ExpandedQueries) > e.maxTotal {
		combined.ExpandedQueries = combined.ExpandedQueries[:e.maxTotal]
	}

	return combined, nil
}

// Helper functions

// tokenize splits text into lowercase tokens
func tokenize(text string) []string {
	words := make([]string, 0)
	current := ""
	for _, r := range strings.ToLower(text) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == 'ä' || r == 'ö' || r == 'ü' || r == 'ß' {
			current += string(r)
		} else if current != "" {
			if len(current) > 2 {
				words = append(words, current)
			}
			current = ""
		}
	}
	if current != "" && len(current) > 2 {
		words = append(words, current)
	}
	return words
}

// getSynonyms returns synonyms for a word
func getSynonyms(word string, language string) []string {
	word = strings.ToLower(word)

	// German synonyms
	if language == "de" {
		synonymsDE := map[string][]string{
			// Common verbs
			"suchen":    {"finden", "recherchieren", "nachschlagen"},
			"finden":    {"suchen", "entdecken", "lokalisieren"},
			"erstellen": {"erzeugen", "anlegen", "generieren"},
			"löschen":   {"entfernen", "beseitigen", "tilgen"},
			"ändern":    {"modifizieren", "anpassen", "bearbeiten"},
			"speichern": {"sichern", "ablegen", "archivieren"},

			// Common nouns
			"dokument":    {"datei", "unterlagen", "schriftstück"},
			"datei":       {"dokument", "file", "daten"},
			"ordner":      {"verzeichnis", "mappe", "folder"},
			"benutzer":    {"anwender", "nutzer", "user"},
			"nachricht":   {"mitteilung", "meldung", "botschaft"},
			"problem":     {"fehler", "schwierigkeit", "störung"},
			"lösung":      {"antwort", "behebung", "abhilfe"},
			"information": {"daten", "angaben", "auskunft"},
			"projekt":     {"vorhaben", "arbeit", "aufgabe"},
			"system":      {"anlage", "plattform", "infrastruktur"},

			// Tech terms
			"api":       {"schnittstelle", "interface"},
			"datenbank": {"database", "datenspeicher"},
			"server":    {"rechner", "host"},
			"software":  {"programm", "anwendung", "app"},
			"hardware":  {"geräte", "technik"},
		}
		if syns, ok := synonymsDE[word]; ok {
			return syns
		}
	}

	// English synonyms
	synonymsEN := map[string][]string{
		// Common verbs
		"search":  {"find", "look", "query", "retrieve"},
		"find":    {"search", "locate", "discover"},
		"create":  {"make", "generate", "build"},
		"delete":  {"remove", "erase", "clear"},
		"modify":  {"change", "edit", "update"},
		"save":    {"store", "preserve", "keep"},

		// Common nouns
		"document":    {"file", "record", "paper"},
		"file":        {"document", "data", "record"},
		"folder":      {"directory", "path", "location"},
		"user":        {"person", "account", "member"},
		"message":     {"notification", "alert", "note"},
		"problem":     {"issue", "error", "bug"},
		"solution":    {"answer", "fix", "resolution"},
		"information": {"data", "details", "facts"},
		"project":     {"task", "work", "initiative"},
		"system":      {"platform", "infrastructure", "setup"},

		// Tech terms
		"api":       {"interface", "endpoint"},
		"database":  {"db", "datastore", "storage"},
		"server":    {"host", "machine", "instance"},
		"software":  {"program", "application", "app"},
		"hardware":  {"device", "equipment"},
		"query":     {"search", "request", "question"},
		"response":  {"answer", "reply", "result"},
	}
	if syns, ok := synonymsEN[word]; ok {
		return syns
	}

	return nil
}

// generateSynonymVariants generates query variants by substituting synonyms
func generateSynonymVariants(query string, words []string, synonymMap map[string][]string, maxVariants int) []string {
	variants := make([]string, 0, maxVariants)

	for word, syns := range synonymMap {
		for i, syn := range syns {
			if i >= maxVariants {
				break
			}
			variant := strings.Replace(strings.ToLower(query), word, syn, 1)
			if variant != strings.ToLower(query) {
				variants = append(variants, variant)
			}
			if len(variants) >= maxVariants {
				return variants
			}
		}
	}

	return variants
}

// deduplicateStrings removes duplicates from a string slice
func deduplicateStrings(strs []string) []string {
	seen := make(map[string]bool)
	result := make([]string, 0, len(strs))

	for _, s := range strs {
		lower := strings.ToLower(strings.TrimSpace(s))
		if !seen[lower] && lower != "" {
			seen[lower] = true
			result = append(result, s)
		}
	}

	return result
}

// parseNumberedLine extracts text from a numbered line like "1. text" or "1) text"
func parseNumberedLine(line string) string {
	line = strings.TrimSpace(line)

	// Try "1. text" format
	for i := 0; i < len(line); i++ {
		if line[i] == '.' || line[i] == ')' {
			if i > 0 && isDigit(line[i-1]) {
				return strings.TrimSpace(line[i+1:])
			}
		}
	}

	return ""
}

// parseBulletLine extracts text from a bullet line like "- text" or "* text"
func parseBulletLine(line string) string {
	line = strings.TrimSpace(line)
	if len(line) < 2 {
		return ""
	}

	runes := []rune(line)
	if len(runes) > 2 && (runes[0] == '-' || runes[0] == '*' || runes[0] == '•') && runes[1] == ' ' {
		return strings.TrimSpace(string(runes[2:]))
	}

	return ""
}

// isDigit checks if a byte is a digit
func isDigit(b byte) bool {
	return b >= '0' && b <= '9'
}
