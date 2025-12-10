// ============================================================================
// meinDENKWERK (mDW) - Lokale KI-Plattform
// ============================================================================
//
// Package:     reranker
// Description: Reranking for improved RAG quality
// Author:      Mike Stoffels with Claude
// Created:     2025-12-06
// License:     MIT
// ============================================================================

package reranker

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/msto63/mDW/pkg/core/logging"
)

// Document represents a document to be reranked
type Document struct {
	ID       string
	Content  string
	Score    float64
	Metadata map[string]string
}

// RerankResult represents a reranked document
type RerankResult struct {
	Document      *Document
	OriginalScore float64
	RerankScore   float64
	FinalScore    float64
}

// Reranker interface for different reranking strategies
type Reranker interface {
	// Rerank reorders documents by relevance to the query
	Rerank(ctx context.Context, query string, docs []*Document, topK int) ([]*RerankResult, error)
}

// LLMFunc is a function that generates text from a prompt
type LLMFunc func(ctx context.Context, prompt string) (string, error)

// CrossEncoderReranker uses an LLM for cross-encoder style reranking
type CrossEncoderReranker struct {
	llmFunc LLMFunc
	logger  *logging.Logger
}

// NewCrossEncoderReranker creates a new cross-encoder reranker
func NewCrossEncoderReranker(llmFunc LLMFunc) *CrossEncoderReranker {
	return &CrossEncoderReranker{
		llmFunc: llmFunc,
		logger:  logging.New("reranker"),
	}
}

// Rerank reorders documents using LLM-based scoring
func (r *CrossEncoderReranker) Rerank(ctx context.Context, query string, docs []*Document, topK int) ([]*RerankResult, error) {
	if r.llmFunc == nil {
		return nil, fmt.Errorf("LLM function not set")
	}

	if len(docs) == 0 {
		return []*RerankResult{}, nil
	}

	r.logger.Info("Reranking documents", "query", query, "docs", len(docs), "topK", topK)

	results := make([]*RerankResult, len(docs))

	// Score each document against the query
	for i, doc := range docs {
		score, err := r.scoreDocument(ctx, query, doc)
		if err != nil {
			r.logger.Warn("Failed to score document", "id", doc.ID, "error", err)
			score = doc.Score * 0.5 // Fallback to reduced original score
		}

		results[i] = &RerankResult{
			Document:      doc,
			OriginalScore: doc.Score,
			RerankScore:   score,
			FinalScore:    combineScores(doc.Score, score),
		}
	}

	// Sort by final score descending
	sort.Slice(results, func(i, j int) bool {
		return results[i].FinalScore > results[j].FinalScore
	})

	// Take top K
	if topK > 0 && len(results) > topK {
		results = results[:topK]
	}

	r.logger.Info("Reranking completed", "results", len(results))
	return results, nil
}

// scoreDocument scores a single document against the query
func (r *CrossEncoderReranker) scoreDocument(ctx context.Context, query string, doc *Document) (float64, error) {
	prompt := fmt.Sprintf(`Rate the relevance of the following document to the query on a scale of 0.0 to 1.0.
Only respond with a single number between 0.0 and 1.0.

Query: %s

Document:
%s

Relevance score (0.0-1.0):`, query, truncateText(doc.Content, 1000))

	response, err := r.llmFunc(ctx, prompt)
	if err != nil {
		return 0, err
	}

	// Parse the score from response
	score, err := parseScore(strings.TrimSpace(response))
	if err != nil {
		return 0, err
	}

	return score, nil
}

// BatchCrossEncoderReranker batches multiple documents for efficient LLM calls
type BatchCrossEncoderReranker struct {
	llmFunc   LLMFunc
	logger    *logging.Logger
	batchSize int
}

// NewBatchCrossEncoderReranker creates a batch reranker
func NewBatchCrossEncoderReranker(llmFunc LLMFunc, batchSize int) *BatchCrossEncoderReranker {
	if batchSize <= 0 {
		batchSize = 5
	}
	return &BatchCrossEncoderReranker{
		llmFunc:   llmFunc,
		logger:    logging.New("batch-reranker"),
		batchSize: batchSize,
	}
}

// Rerank reorders documents using batched LLM scoring
func (r *BatchCrossEncoderReranker) Rerank(ctx context.Context, query string, docs []*Document, topK int) ([]*RerankResult, error) {
	if r.llmFunc == nil {
		return nil, fmt.Errorf("LLM function not set")
	}

	if len(docs) == 0 {
		return []*RerankResult{}, nil
	}

	r.logger.Info("Batch reranking documents", "query", query, "docs", len(docs), "topK", topK)

	results := make([]*RerankResult, len(docs))

	// Process in batches
	for batchStart := 0; batchStart < len(docs); batchStart += r.batchSize {
		batchEnd := batchStart + r.batchSize
		if batchEnd > len(docs) {
			batchEnd = len(docs)
		}

		batchDocs := docs[batchStart:batchEnd]
		scores, err := r.scoreBatch(ctx, query, batchDocs)
		if err != nil {
			r.logger.Warn("Batch scoring failed", "error", err)
			// Fallback to original scores
			for i, doc := range batchDocs {
				results[batchStart+i] = &RerankResult{
					Document:      doc,
					OriginalScore: doc.Score,
					RerankScore:   doc.Score * 0.5,
					FinalScore:    doc.Score * 0.75,
				}
			}
			continue
		}

		for i, doc := range batchDocs {
			results[batchStart+i] = &RerankResult{
				Document:      doc,
				OriginalScore: doc.Score,
				RerankScore:   scores[i],
				FinalScore:    combineScores(doc.Score, scores[i]),
			}
		}
	}

	// Sort by final score descending
	sort.Slice(results, func(i, j int) bool {
		return results[i].FinalScore > results[j].FinalScore
	})

	// Take top K
	if topK > 0 && len(results) > topK {
		results = results[:topK]
	}

	r.logger.Info("Batch reranking completed", "results", len(results))
	return results, nil
}

// scoreBatch scores multiple documents in one LLM call
func (r *BatchCrossEncoderReranker) scoreBatch(ctx context.Context, query string, docs []*Document) ([]float64, error) {
	// Build batch prompt
	var sb strings.Builder
	sb.WriteString("Rate the relevance of each document to the query on a scale of 0.0 to 1.0.\n")
	sb.WriteString("Respond with one score per line, in order.\n\n")
	sb.WriteString("Query: ")
	sb.WriteString(query)
	sb.WriteString("\n\n")

	for i, doc := range docs {
		sb.WriteString(fmt.Sprintf("Document %d:\n%s\n\n", i+1, truncateText(doc.Content, 500)))
	}

	sb.WriteString("Scores (one per line, 0.0-1.0):")

	response, err := r.llmFunc(ctx, sb.String())
	if err != nil {
		return nil, err
	}

	// Parse scores from response
	scores := make([]float64, len(docs))
	lines := strings.Split(strings.TrimSpace(response), "\n")

	for i := range docs {
		if i < len(lines) {
			score, err := parseScore(strings.TrimSpace(lines[i]))
			if err != nil {
				scores[i] = docs[i].Score * 0.5 // Fallback
			} else {
				scores[i] = score
			}
		} else {
			scores[i] = docs[i].Score * 0.5 // Fallback
		}
	}

	return scores, nil
}

// KeywordBoostReranker boosts documents containing query keywords
type KeywordBoostReranker struct {
	logger     *logging.Logger
	boostFactor float64
}

// NewKeywordBoostReranker creates a keyword-based reranker
func NewKeywordBoostReranker(boostFactor float64) *KeywordBoostReranker {
	if boostFactor <= 0 {
		boostFactor = 0.2
	}
	return &KeywordBoostReranker{
		logger:     logging.New("keyword-reranker"),
		boostFactor: boostFactor,
	}
}

// Rerank boosts documents based on keyword matches
func (r *KeywordBoostReranker) Rerank(ctx context.Context, query string, docs []*Document, topK int) ([]*RerankResult, error) {
	if len(docs) == 0 {
		return []*RerankResult{}, nil
	}

	r.logger.Info("Keyword reranking", "query", query, "docs", len(docs))

	queryTerms := tokenize(query)
	results := make([]*RerankResult, len(docs))

	for i, doc := range docs {
		keywordScore := calculateKeywordScore(doc.Content, queryTerms)
		rerankScore := doc.Score + (keywordScore * r.boostFactor)

		// Normalize to 0-1 range
		if rerankScore > 1.0 {
			rerankScore = 1.0
		}

		results[i] = &RerankResult{
			Document:      doc,
			OriginalScore: doc.Score,
			RerankScore:   keywordScore,
			FinalScore:    rerankScore,
		}
	}

	// Sort by final score descending
	sort.Slice(results, func(i, j int) bool {
		return results[i].FinalScore > results[j].FinalScore
	})

	if topK > 0 && len(results) > topK {
		results = results[:topK]
	}

	return results, nil
}

// CompositeReranker chains multiple rerankers
type CompositeReranker struct {
	rerankers []Reranker
	logger    *logging.Logger
}

// NewCompositeReranker creates a composite reranker
func NewCompositeReranker(rerankers ...Reranker) *CompositeReranker {
	return &CompositeReranker{
		rerankers: rerankers,
		logger:    logging.New("composite-reranker"),
	}
}

// Rerank applies all rerankers in sequence
func (r *CompositeReranker) Rerank(ctx context.Context, query string, docs []*Document, topK int) ([]*RerankResult, error) {
	if len(docs) == 0 {
		return []*RerankResult{}, nil
	}

	currentDocs := docs

	for _, reranker := range r.rerankers {
		results, err := reranker.Rerank(ctx, query, currentDocs, 0) // Don't limit during intermediate steps
		if err != nil {
			r.logger.Warn("Reranker failed", "error", err)
			continue
		}

		// Convert results back to documents for next reranker
		currentDocs = make([]*Document, len(results))
		for i, result := range results {
			currentDocs[i] = &Document{
				ID:       result.Document.ID,
				Content:  result.Document.Content,
				Score:    result.FinalScore,
				Metadata: result.Document.Metadata,
			}
		}
	}

	// Build final results
	results := make([]*RerankResult, len(currentDocs))
	for i, doc := range currentDocs {
		results[i] = &RerankResult{
			Document:      docs[i], // Use original document
			OriginalScore: docs[i].Score,
			FinalScore:    doc.Score,
			RerankScore:   doc.Score,
		}
	}

	// Sort and take topK
	sort.Slice(results, func(i, j int) bool {
		return results[i].FinalScore > results[j].FinalScore
	})

	if topK > 0 && len(results) > topK {
		results = results[:topK]
	}

	return results, nil
}

// Helper functions

// combineScores combines original and rerank scores
func combineScores(original, rerank float64) float64 {
	// Weight: 40% original, 60% rerank
	return original*0.4 + rerank*0.6
}

// parseScore parses a score from a string
func parseScore(s string) (float64, error) {
	// Remove any non-numeric characters except . and -
	cleaned := ""
	for _, c := range s {
		if (c >= '0' && c <= '9') || c == '.' || c == '-' {
			cleaned += string(c)
		}
	}

	if cleaned == "" {
		return 0, fmt.Errorf("no numeric value found")
	}

	score, err := strconv.ParseFloat(cleaned, 64)
	if err != nil {
		return 0, err
	}

	// Clamp to 0-1
	if score < 0 {
		score = 0
	}
	if score > 1 {
		score = 1
	}

	return score, nil
}

// truncateText truncates text to maxLen characters
func truncateText(text string, maxLen int) string {
	if len(text) <= maxLen {
		return text
	}
	return text[:maxLen] + "..."
}

// tokenize splits text into lowercase tokens
func tokenize(text string) []string {
	words := make([]string, 0)
	current := ""
	for _, r := range text {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == 'ä' || r == 'ö' || r == 'ü' || r == 'ß' {
			if r >= 'A' && r <= 'Z' {
				r = r + 32
			}
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

// calculateKeywordScore calculates a keyword match score
func calculateKeywordScore(content string, queryTerms []string) float64 {
	if len(queryTerms) == 0 {
		return 0
	}

	contentTerms := tokenize(content)
	contentSet := make(map[string]int)
	for _, term := range contentTerms {
		contentSet[term]++
	}

	matches := 0
	for _, term := range queryTerms {
		if count, ok := contentSet[term]; ok && count > 0 {
			matches++
		}
	}

	return float64(matches) / float64(len(queryTerms))
}
