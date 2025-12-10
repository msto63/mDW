// ============================================================================
// meinDENKWERK (mDW) - Lokale KI-Plattform
// ============================================================================
//
// Package:     context
// Description: Context window management for conversations
// Author:      Mike Stoffels with Claude
// Created:     2025-12-06
// License:     MIT
// ============================================================================

package context

import (
	"context"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/msto63/mDW/pkg/core/logging"
)

// Message represents a chat message
type Message struct {
	Role       string
	Content    string
	TokenCount int
}

// SummarizeFunc is a function that summarizes text
type SummarizeFunc func(ctx context.Context, text string, maxTokens int) (string, error)

// WindowConfig holds context window configuration
type WindowConfig struct {
	// MaxTokens is the maximum number of tokens for the context window
	MaxTokens int

	// ReserveTokens is the number of tokens to reserve for response
	ReserveTokens int

	// SummarizeThreshold is the percentage of MaxTokens at which to summarize
	// e.g., 0.8 means summarize when 80% full
	SummarizeThreshold float64

	// MinMessagesToKeep is the minimum number of recent messages to always keep
	MinMessagesToKeep int

	// SummaryMaxTokens is the maximum tokens for the summary
	SummaryMaxTokens int
}

// DefaultWindowConfig returns default configuration
func DefaultWindowConfig() WindowConfig {
	return WindowConfig{
		MaxTokens:          4096,
		ReserveTokens:      512,
		SummarizeThreshold: 0.75,
		MinMessagesToKeep:  4,
		SummaryMaxTokens:   500,
	}
}

// ModelWindowSizes maps model prefixes to their context window sizes
var ModelWindowSizes = map[string]int{
	// Ollama/Local models
	"llama3":   8192,
	"llama3.1": 128000,
	"llama3.2": 128000,
	"mistral":  32768,
	"mixtral":  32768,
	"qwen":     32768,
	"gemma":    8192,
	"phi":      2048,

	// OpenAI models
	"gpt-4":         8192,
	"gpt-4-turbo":   128000,
	"gpt-4o":        128000,
	"gpt-3.5-turbo": 16385,
	"o1":            128000,

	// Anthropic models
	"claude-3":         200000,
	"claude-3.5":       200000,
	"claude-opus":      200000,
	"claude-sonnet":    200000,
	"claude-haiku":     200000,
	"claude-2":         100000,

	// Default
	"default": 4096,
}

// GetModelWindowSize returns the context window size for a model
func GetModelWindowSize(model string) int {
	// Check exact match first
	if size, ok := ModelWindowSizes[model]; ok {
		return size
	}

	// Check prefix match
	modelLower := strings.ToLower(model)
	for prefix, size := range ModelWindowSizes {
		if strings.HasPrefix(modelLower, prefix) {
			return size
		}
	}

	return ModelWindowSizes["default"]
}

// Manager manages context windows for conversations
type Manager struct {
	config        WindowConfig
	summarizeFunc SummarizeFunc
	logger        *logging.Logger
}

// NewManager creates a new context window manager
func NewManager(cfg WindowConfig, summarizeFunc SummarizeFunc) *Manager {
	return &Manager{
		config:        cfg,
		summarizeFunc: summarizeFunc,
		logger:        logging.New("context-manager"),
	}
}

// ConfigForModel returns configuration adjusted for a specific model
func (m *Manager) ConfigForModel(model string) WindowConfig {
	cfg := m.config
	modelSize := GetModelWindowSize(model)

	// Adjust config based on model's actual window size
	if modelSize < cfg.MaxTokens {
		cfg.MaxTokens = modelSize
	}

	return cfg
}

// WindowState represents the current state of a context window
type WindowState struct {
	TotalTokens      int
	AvailableTokens  int
	MessageCount     int
	HasSummary       bool
	SummaryTokens    int
	NeedsSummarize   bool
	ThresholdPercent float64
}

// AnalyzeWindow analyzes the current state of messages in the context window
func (m *Manager) AnalyzeWindow(messages []Message, model string) WindowState {
	cfg := m.ConfigForModel(model)

	totalTokens := 0
	hasSummary := false
	summaryTokens := 0

	for _, msg := range messages {
		tokens := msg.TokenCount
		if tokens == 0 {
			tokens = EstimateTokens(msg.Content)
		}
		totalTokens += tokens

		// Check if this is a summary message
		if msg.Role == "system" && strings.Contains(msg.Content, "[Summary]") {
			hasSummary = true
			summaryTokens = tokens
		}
	}

	availableTokens := cfg.MaxTokens - cfg.ReserveTokens - totalTokens
	threshold := float64(totalTokens) / float64(cfg.MaxTokens-cfg.ReserveTokens)
	needsSummarize := threshold >= cfg.SummarizeThreshold

	return WindowState{
		TotalTokens:      totalTokens,
		AvailableTokens:  availableTokens,
		MessageCount:     len(messages),
		HasSummary:       hasSummary,
		SummaryTokens:    summaryTokens,
		NeedsSummarize:   needsSummarize,
		ThresholdPercent: threshold * 100,
	}
}

// ProcessResult represents the result of processing messages
type ProcessResult struct {
	Messages      []Message
	WasTruncated  bool
	WasSummarized bool
	Summary       string
	TokensRemoved int
	TokensKept    int
}

// ProcessMessages processes messages to fit within the context window
// It applies sliding window and optional summarization
func (m *Manager) ProcessMessages(ctx context.Context, messages []Message, model string) (*ProcessResult, error) {
	cfg := m.ConfigForModel(model)
	state := m.AnalyzeWindow(messages, model)

	result := &ProcessResult{
		Messages:   messages,
		TokensKept: state.TotalTokens,
	}

	// If we're within limits, return as-is
	if !state.NeedsSummarize && state.AvailableTokens > 0 {
		return result, nil
	}

	m.logger.Info("Context window processing needed",
		"total_tokens", state.TotalTokens,
		"threshold", fmt.Sprintf("%.1f%%", state.ThresholdPercent),
		"messages", len(messages),
	)

	// Try summarization first if available
	if m.summarizeFunc != nil && len(messages) > cfg.MinMessagesToKeep+2 {
		summarized, err := m.summarizeOldMessages(ctx, messages, cfg)
		if err != nil {
			m.logger.Warn("Summarization failed, falling back to truncation", "error", err)
		} else {
			result.Messages = summarized
			result.WasSummarized = true
			result.TokensRemoved = state.TotalTokens - m.countTokens(summarized)
			result.TokensKept = m.countTokens(summarized)

			// Find the summary
			for _, msg := range summarized {
				if msg.Role == "system" && strings.Contains(msg.Content, "[Summary]") {
					result.Summary = msg.Content
					break
				}
			}

			m.logger.Info("Context summarized",
				"original_messages", len(messages),
				"new_messages", len(summarized),
				"tokens_removed", result.TokensRemoved,
			)
			return result, nil
		}
	}

	// Fall back to sliding window truncation
	truncated := m.applySlidingWindow(messages, cfg)
	result.Messages = truncated
	result.WasTruncated = true
	result.TokensRemoved = state.TotalTokens - m.countTokens(truncated)
	result.TokensKept = m.countTokens(truncated)

	m.logger.Info("Context truncated (sliding window)",
		"original_messages", len(messages),
		"kept_messages", len(truncated),
		"tokens_removed", result.TokensRemoved,
	)

	return result, nil
}

// summarizeOldMessages creates a summary of older messages
func (m *Manager) summarizeOldMessages(ctx context.Context, messages []Message, cfg WindowConfig) ([]Message, error) {
	if len(messages) <= cfg.MinMessagesToKeep {
		return messages, nil
	}

	// Separate messages to summarize from recent ones to keep
	keepCount := cfg.MinMessagesToKeep
	if keepCount > len(messages) {
		keepCount = len(messages)
	}

	toSummarize := messages[:len(messages)-keepCount]
	toKeep := messages[len(messages)-keepCount:]

	// Build text to summarize
	var sb strings.Builder
	sb.WriteString("Summarize this conversation history:\n\n")
	for _, msg := range toSummarize {
		sb.WriteString(fmt.Sprintf("%s: %s\n\n", msg.Role, msg.Content))
	}

	// Generate summary
	summary, err := m.summarizeFunc(ctx, sb.String(), cfg.SummaryMaxTokens)
	if err != nil {
		return nil, fmt.Errorf("failed to summarize: %w", err)
	}

	// Build new message list with summary
	result := make([]Message, 0, len(toKeep)+1)

	// Add summary as system message
	summaryMsg := Message{
		Role:       "system",
		Content:    fmt.Sprintf("[Summary of earlier conversation]\n%s", summary),
		TokenCount: EstimateTokens(summary) + 10, // +10 for prefix
	}
	result = append(result, summaryMsg)

	// Add kept recent messages
	result = append(result, toKeep...)

	return result, nil
}

// applySlidingWindow applies sliding window truncation to keep recent messages
func (m *Manager) applySlidingWindow(messages []Message, cfg WindowConfig) []Message {
	targetTokens := cfg.MaxTokens - cfg.ReserveTokens

	// Always try to keep at least MinMessagesToKeep
	if len(messages) <= cfg.MinMessagesToKeep {
		return messages
	}

	// Start from the end and work backwards
	var result []Message
	currentTokens := 0

	for i := len(messages) - 1; i >= 0; i-- {
		msg := messages[i]
		tokens := msg.TokenCount
		if tokens == 0 {
			tokens = EstimateTokens(msg.Content)
		}

		// Check if adding this message would exceed the limit
		if currentTokens+tokens > targetTokens {
			// If we haven't reached minimum messages, try to include anyway
			if len(result) < cfg.MinMessagesToKeep {
				result = append([]Message{msg}, result...)
				currentTokens += tokens
				continue
			}
			break
		}

		result = append([]Message{msg}, result...)
		currentTokens += tokens
	}

	return result
}

// countTokens counts total tokens in messages
func (m *Manager) countTokens(messages []Message) int {
	total := 0
	for _, msg := range messages {
		if msg.TokenCount > 0 {
			total += msg.TokenCount
		} else {
			total += EstimateTokens(msg.Content)
		}
	}
	return total
}

// EstimateTokens estimates the number of tokens in a text
// This is a simple approximation based on character/word count
// For more accurate counting, use tiktoken or similar library
func EstimateTokens(text string) int {
	if text == "" {
		return 0
	}

	// Rough estimation: ~4 characters per token for English
	// Adjust for other languages (German ~3.5, CJK ~1.5)
	charCount := utf8.RuneCountInString(text)

	// Count words for additional adjustment
	words := strings.Fields(text)
	wordCount := len(words)

	// Estimate based on average of character-based and word-based methods
	charBased := charCount / 4
	wordBased := wordCount * 4 / 3 // ~1.3 tokens per word

	// Use weighted average
	estimate := (charBased*2 + wordBased) / 3

	// Add overhead for message formatting
	estimate += 4 // role, content markers

	if estimate < 1 {
		estimate = 1
	}

	return estimate
}

// TruncateToTokens truncates text to approximately maxTokens
func TruncateToTokens(text string, maxTokens int) string {
	if EstimateTokens(text) <= maxTokens {
		return text
	}

	// Estimate characters per token
	targetChars := maxTokens * 4

	runes := []rune(text)
	if len(runes) <= targetChars {
		return text
	}

	// Truncate and add ellipsis
	return string(runes[:targetChars-3]) + "..."
}

// MessageChunker splits a long message into smaller chunks
type MessageChunker struct {
	MaxTokensPerChunk int
	Overlap           int // Number of tokens to overlap between chunks
}

// NewMessageChunker creates a new message chunker
func NewMessageChunker(maxTokens, overlap int) *MessageChunker {
	return &MessageChunker{
		MaxTokensPerChunk: maxTokens,
		Overlap:           overlap,
	}
}

// ChunkMessage splits a message into smaller chunks
func (c *MessageChunker) ChunkMessage(content string) []string {
	tokens := EstimateTokens(content)
	if tokens <= c.MaxTokensPerChunk {
		return []string{content}
	}

	// Split by sentences first
	sentences := splitSentences(content)
	if len(sentences) == 0 {
		return []string{content}
	}

	var chunks []string
	var currentChunk strings.Builder
	currentTokens := 0

	for _, sentence := range sentences {
		sentenceTokens := EstimateTokens(sentence)

		if currentTokens+sentenceTokens > c.MaxTokensPerChunk {
			if currentChunk.Len() > 0 {
				chunks = append(chunks, strings.TrimSpace(currentChunk.String()))
				currentChunk.Reset()
				currentTokens = 0

				// Add overlap from previous chunk
				if c.Overlap > 0 && len(chunks) > 0 {
					lastChunk := chunks[len(chunks)-1]
					overlapText := getLastNTokens(lastChunk, c.Overlap)
					currentChunk.WriteString(overlapText)
					currentTokens = EstimateTokens(overlapText)
				}
			}
		}

		currentChunk.WriteString(sentence)
		currentChunk.WriteString(" ")
		currentTokens += sentenceTokens
	}

	if currentChunk.Len() > 0 {
		chunks = append(chunks, strings.TrimSpace(currentChunk.String()))
	}

	return chunks
}

// splitSentences splits text into sentences
func splitSentences(text string) []string {
	// Simple sentence splitting
	var sentences []string
	var current strings.Builder

	runes := []rune(text)
	for i, r := range runes {
		current.WriteRune(r)

		// Check for sentence endings
		if r == '.' || r == '!' || r == '?' {
			// Check if followed by space or end of text
			if i == len(runes)-1 || runes[i+1] == ' ' || runes[i+1] == '\n' {
				sentences = append(sentences, strings.TrimSpace(current.String()))
				current.Reset()
			}
		}
	}

	if current.Len() > 0 {
		sentences = append(sentences, strings.TrimSpace(current.String()))
	}

	return sentences
}

// getLastNTokens returns approximately the last n tokens of text
func getLastNTokens(text string, n int) string {
	if n <= 0 {
		return ""
	}

	targetChars := n * 4
	runes := []rune(text)

	if len(runes) <= targetChars {
		return text
	}

	return string(runes[len(runes)-targetChars:])
}
