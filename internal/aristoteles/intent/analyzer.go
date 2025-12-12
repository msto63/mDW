// Package intent provides intent analysis for the Aristoteles service
package intent

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	pb "github.com/msto63/mDW/api/gen/aristoteles"
	"github.com/msto63/mDW/pkg/core/logging"
)

// LLMFunc is a function type for calling the LLM
type LLMFunc func(ctx context.Context, model string, systemPrompt string, userPrompt string) (string, error)

// cacheEntry represents a cached intent result
type cacheEntry struct {
	result    *pb.IntentResult
	timestamp time.Time
}

// Analyzer analyzes user prompts to determine intent
type Analyzer struct {
	llmFunc             LLMFunc
	model               string
	confidenceThreshold float32
	logger              *logging.Logger

	// Intent caching
	cache     map[string]*cacheEntry
	cacheMu   sync.RWMutex
	cacheTTL  time.Duration
	cacheSize int
}

// Config holds analyzer configuration
type Config struct {
	Model               string
	ConfidenceThreshold float32
}

// DefaultConfig returns default analyzer configuration
func DefaultConfig() *Config {
	return &Config{
		Model:               "mistral:7b",
		ConfidenceThreshold: 0.7,
	}
}

// NewAnalyzer creates a new intent analyzer
func NewAnalyzer(cfg *Config) *Analyzer {
	if cfg == nil {
		cfg = DefaultConfig()
	}
	return &Analyzer{
		model:               cfg.Model,
		confidenceThreshold: cfg.ConfidenceThreshold,
		logger:              logging.New("aristoteles-intent"),
		cache:               make(map[string]*cacheEntry),
		cacheTTL:            5 * time.Minute,
		cacheSize:           100,
	}
}

// SetLLMFunc sets the LLM function
func (a *Analyzer) SetLLMFunc(fn LLMFunc) {
	a.llmFunc = fn
}

// Analyze analyzes a prompt and returns the intent result
func (a *Analyzer) Analyze(ctx context.Context, prompt string, conversationID string) (*pb.IntentResult, error) {
	// Check cache first
	cacheKey := a.getCacheKey(prompt)
	if cached := a.getFromCache(cacheKey); cached != nil {
		a.logger.Debug("Intent cache hit", "key", cacheKey[:8])
		return cached, nil
	}

	var result *pb.IntentResult
	var err error

	if a.llmFunc == nil {
		result, err = a.analyzeLocal(prompt)
	} else {
		start := time.Now()

		systemPrompt := intentSystemPrompt
		userPrompt := fmt.Sprintf(intentUserPrompt, prompt)

		response, llmErr := a.llmFunc(ctx, a.model, systemPrompt, userPrompt)
		if llmErr != nil {
			a.logger.Warn("LLM intent analysis failed, using fallback", "error", llmErr)
			result, err = a.analyzeLocal(prompt)
		} else {
			result, err = a.parseResponse(response)
			if err != nil {
				a.logger.Warn("Failed to parse LLM response, using fallback", "error", err)
				result, err = a.analyzeLocal(prompt)
			} else {
				a.logger.Debug("Intent analyzed via LLM",
					"primary", result.Primary.String(),
					"confidence", result.Confidence,
					"duration", time.Since(start))
			}
		}
	}

	if err != nil {
		return nil, err
	}

	// Cache the result
	a.putInCache(cacheKey, result)

	return result, nil
}

// getCacheKey generates a cache key for the prompt
func (a *Analyzer) getCacheKey(prompt string) string {
	// Normalize prompt by trimming and lowercasing for better cache hits
	normalized := strings.ToLower(strings.TrimSpace(prompt))
	hash := sha256.Sum256([]byte(normalized))
	return hex.EncodeToString(hash[:])
}

// getFromCache retrieves a cached result if valid
func (a *Analyzer) getFromCache(key string) *pb.IntentResult {
	a.cacheMu.RLock()
	defer a.cacheMu.RUnlock()

	entry, ok := a.cache[key]
	if !ok {
		return nil
	}

	// Check TTL
	if time.Since(entry.timestamp) > a.cacheTTL {
		return nil
	}

	return entry.result
}

// putInCache stores a result in the cache
func (a *Analyzer) putInCache(key string, result *pb.IntentResult) {
	a.cacheMu.Lock()
	defer a.cacheMu.Unlock()

	// Simple eviction: if cache is full, remove oldest entries
	if len(a.cache) >= a.cacheSize {
		oldest := time.Now()
		var oldestKey string
		for k, v := range a.cache {
			if v.timestamp.Before(oldest) {
				oldest = v.timestamp
				oldestKey = k
			}
		}
		if oldestKey != "" {
			delete(a.cache, oldestKey)
		}
	}

	a.cache[key] = &cacheEntry{
		result:    result,
		timestamp: time.Now(),
	}
}

// analyzeLocal performs local heuristic-based intent analysis
func (a *Analyzer) analyzeLocal(prompt string) (*pb.IntentResult, error) {
	lower := strings.ToLower(prompt)
	result := &pb.IntentResult{
		Primary:    pb.IntentType_INTENT_TYPE_DIRECT_LLM,
		Secondary:  make([]pb.IntentType, 0),
		Confidence: 0.7,
		Scores:     make(map[string]float32),
		Complexity: pb.ComplexityLevel_COMPLEXITY_SIMPLE,
	}

	// Code-related keywords
	codeKeywords := []string{"code", "function", "class", "implement", "debug", "fix bug", "error", "exception", "compile", "syntax", "program", "script", "python", "javascript", "go ", "golang", "java", "typescript", "html", "css", "sql", "api", "endpoint"}
	for _, kw := range codeKeywords {
		if strings.Contains(lower, kw) {
			if strings.Contains(lower, "analyze") || strings.Contains(lower, "explain") || strings.Contains(lower, "review") {
				result.Primary = pb.IntentType_INTENT_TYPE_CODE_ANALYSIS
			} else {
				result.Primary = pb.IntentType_INTENT_TYPE_CODE_GENERATION
			}
			result.Confidence = 0.85
			result.Complexity = pb.ComplexityLevel_COMPLEXITY_MODERATE
			break
		}
	}

	// Web research keywords (English only - prompts are translated before analysis)
	// Note: Avoid short keywords like "now" that can match within other words
	webKeywords := []string{
		"search", "find", "look up", "what is the latest", "current", "news",
		"recent", "today", "yesterday", "this week", "this month", "breaking",
		"latest", "currently", "at the moment", "right now", "happening now",
	}
	webResearchDetected := false
	for _, kw := range webKeywords {
		if strings.Contains(lower, kw) {
			webResearchDetected = true
			break
		}
	}

	// Dynamic year detection (current year ± 1)
	if !webResearchDetected {
		currentYear := time.Now().Year()
		for year := currentYear - 1; year <= currentYear+1; year++ {
			if strings.Contains(lower, fmt.Sprintf("%d", year)) {
				webResearchDetected = true
				break
			}
		}
	}

	if webResearchDetected {
		if result.Primary == pb.IntentType_INTENT_TYPE_DIRECT_LLM {
			result.Primary = pb.IntentType_INTENT_TYPE_WEB_RESEARCH
		} else {
			result.Secondary = append(result.Secondary, pb.IntentType_INTENT_TYPE_WEB_RESEARCH)
		}
		result.Confidence = 0.8
	}

	// RAG keywords
	ragKeywords := []string{"in my documents", "in the knowledge base", "from my files", "in my notes", "based on"}
	for _, kw := range ragKeywords {
		if strings.Contains(lower, kw) {
			if result.Primary == pb.IntentType_INTENT_TYPE_DIRECT_LLM {
				result.Primary = pb.IntentType_INTENT_TYPE_RAG_QUERY
			} else {
				result.Secondary = append(result.Secondary, pb.IntentType_INTENT_TYPE_RAG_QUERY)
			}
			result.Confidence = 0.85
			break
		}
	}

	// Task decomposition keywords
	taskKeywords := []string{"step by step", "plan", "how do i", "tutorial", "guide me", "walk me through"}
	for _, kw := range taskKeywords {
		if strings.Contains(lower, kw) {
			result.Secondary = append(result.Secondary, pb.IntentType_INTENT_TYPE_TASK_DECOMPOSITION)
			result.Complexity = pb.ComplexityLevel_COMPLEXITY_MODERATE
			break
		}
	}

	// Summarization keywords
	if strings.Contains(lower, "summarize") || strings.Contains(lower, "summary") || strings.Contains(lower, "tldr") || strings.Contains(lower, "brief") {
		result.Primary = pb.IntentType_INTENT_TYPE_SUMMARIZATION
		result.Confidence = 0.9
	}

	// Translation keywords
	if strings.Contains(lower, "translate") || strings.Contains(lower, "translation") || strings.Contains(lower, "in german") || strings.Contains(lower, "in english") || strings.Contains(lower, "auf deutsch") {
		result.Primary = pb.IntentType_INTENT_TYPE_TRANSLATION
		result.Confidence = 0.9
	}

	// Creative keywords
	creativeKeywords := []string{"write a story", "write a poem", "creative", "imagine", "fiction", "compose"}
	for _, kw := range creativeKeywords {
		if strings.Contains(lower, kw) {
			result.Primary = pb.IntentType_INTENT_TYPE_CREATIVE
			result.Confidence = 0.85
			break
		}
	}

	// Multi-step detection
	if strings.Count(prompt, "?") > 1 || strings.Contains(lower, "and then") || strings.Contains(lower, "after that") {
		result.Secondary = append(result.Secondary, pb.IntentType_INTENT_TYPE_MULTI_STEP)
		result.Complexity = pb.ComplexityLevel_COMPLEXITY_COMPLEX
	}

	// Set scores based on detection
	result.Scores[result.Primary.String()] = result.Confidence
	for _, sec := range result.Secondary {
		result.Scores[sec.String()] = result.Confidence * 0.7
	}

	return result, nil
}

// parseResponse parses the LLM JSON response into an IntentResult
func (a *Analyzer) parseResponse(response string) (*pb.IntentResult, error) {
	// Extract JSON from response
	jsonStr := response
	if idx := strings.Index(response, "{"); idx != -1 {
		jsonStr = response[idx:]
		if endIdx := strings.LastIndex(jsonStr, "}"); endIdx != -1 {
			jsonStr = jsonStr[:endIdx+1]
		}
	}

	var parsed struct {
		Primary    string             `json:"primary"`
		Secondary  []string           `json:"secondary"`
		Confidence float32            `json:"confidence"`
		Reasoning  string             `json:"reasoning"`
		Complexity string             `json:"complexity"`
		Entities   []string           `json:"entities"`
		Language   string             `json:"language"`
		Scores     map[string]float32 `json:"scores"`
	}

	if err := json.Unmarshal([]byte(jsonStr), &parsed); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	result := &pb.IntentResult{
		Primary:          stringToIntentType(parsed.Primary),
		Secondary:        make([]pb.IntentType, 0, len(parsed.Secondary)),
		Confidence:       parsed.Confidence,
		Reasoning:        parsed.Reasoning,
		DetectedEntities: parsed.Entities,
		Language:         parsed.Language,
		Complexity:       stringToComplexity(parsed.Complexity),
		Scores:           parsed.Scores,
	}

	for _, s := range parsed.Secondary {
		result.Secondary = append(result.Secondary, stringToIntentType(s))
	}

	if result.Scores == nil {
		result.Scores = make(map[string]float32)
	}

	return result, nil
}

// stringToIntentType converts a string to IntentType
func stringToIntentType(s string) pb.IntentType {
	s = strings.ToUpper(strings.TrimSpace(s))
	s = strings.ReplaceAll(s, " ", "_")
	s = strings.ReplaceAll(s, "-", "_")

	switch s {
	case "DIRECT_LLM", "DIRECT", "SIMPLE":
		return pb.IntentType_INTENT_TYPE_DIRECT_LLM
	case "CODE_GENERATION", "CODE", "CODING":
		return pb.IntentType_INTENT_TYPE_CODE_GENERATION
	case "CODE_ANALYSIS", "CODE_REVIEW":
		return pb.IntentType_INTENT_TYPE_CODE_ANALYSIS
	case "WEB_RESEARCH", "WEB_SEARCH", "SEARCH":
		return pb.IntentType_INTENT_TYPE_WEB_RESEARCH
	case "RAG_QUERY", "RAG", "KNOWLEDGE":
		return pb.IntentType_INTENT_TYPE_RAG_QUERY
	case "TASK_DECOMPOSITION", "TASK", "PLANNING":
		return pb.IntentType_INTENT_TYPE_TASK_DECOMPOSITION
	case "SUMMARIZATION", "SUMMARY":
		return pb.IntentType_INTENT_TYPE_SUMMARIZATION
	case "TRANSLATION", "TRANSLATE":
		return pb.IntentType_INTENT_TYPE_TRANSLATION
	case "MULTI_STEP", "MULTI", "COMPLEX":
		return pb.IntentType_INTENT_TYPE_MULTI_STEP
	case "CREATIVE", "CREATIVE_WRITING":
		return pb.IntentType_INTENT_TYPE_CREATIVE
	case "FACTUAL", "FACT":
		return pb.IntentType_INTENT_TYPE_FACTUAL
	case "CONVERSATION", "CHAT":
		return pb.IntentType_INTENT_TYPE_CONVERSATION
	default:
		return pb.IntentType_INTENT_TYPE_DIRECT_LLM
	}
}

// stringToComplexity converts a string to ComplexityLevel
func stringToComplexity(s string) pb.ComplexityLevel {
	s = strings.ToUpper(strings.TrimSpace(s))
	switch s {
	case "SIMPLE", "EASY":
		return pb.ComplexityLevel_COMPLEXITY_SIMPLE
	case "MODERATE", "MEDIUM":
		return pb.ComplexityLevel_COMPLEXITY_MODERATE
	case "COMPLEX", "HARD":
		return pb.ComplexityLevel_COMPLEXITY_COMPLEX
	case "EXPERT", "SPECIALIZED":
		return pb.ComplexityLevel_COMPLEXITY_EXPERT
	default:
		return pb.ComplexityLevel_COMPLEXITY_SIMPLE
	}
}

// Intent analysis prompts
const intentSystemPrompt = `You are an intent classifier for an AI assistant. Analyze user prompts and determine:
1. The primary intent (what the user mainly wants)
2. Secondary intents (additional needs)
3. Complexity level
4. Confidence score (0.0-1.0)

Intent types:
- DIRECT_LLM: Simple questions, general knowledge, conversation
- CODE_GENERATION: Writing new code, implementing features
- CODE_ANALYSIS: Reviewing, explaining, debugging code
- WEB_RESEARCH: Finding current/recent information online
- RAG_QUERY: Searching user's knowledge base/documents
- TASK_DECOMPOSITION: Complex tasks needing step-by-step planning
- SUMMARIZATION: Condensing text
- TRANSLATION: Language translation
- MULTI_STEP: Tasks requiring multiple operations
- CREATIVE: Creative writing, storytelling
- FACTUAL: Questions requiring verified facts
- CONVERSATION: General chat, small talk

Complexity levels: SIMPLE, MODERATE, COMPLEX, EXPERT

Respond ONLY with valid JSON in this exact format:
{
  "primary": "INTENT_TYPE",
  "secondary": ["INTENT_TYPE"],
  "confidence": 0.85,
  "reasoning": "Brief explanation",
  "complexity": "SIMPLE|MODERATE|COMPLEX|EXPERT",
  "language": "detected language",
  "entities": ["detected entities"]
}`

const intentUserPrompt = `Analyze this user prompt and classify its intent:

"%s"

Respond with JSON only.`
