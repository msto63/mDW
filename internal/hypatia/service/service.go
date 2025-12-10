package service

import (
	"context"
	"fmt"
	"time"

	"github.com/msto63/mDW/internal/hypatia/chunking"
	"github.com/msto63/mDW/internal/hypatia/expansion"
	"github.com/msto63/mDW/internal/hypatia/reranker"
	"github.com/msto63/mDW/internal/hypatia/vectorstore"
	"github.com/msto63/mDW/pkg/core/logging"
)

// EmbeddingFunc is a function that generates embeddings
type EmbeddingFunc func(ctx context.Context, texts []string) ([][]float64, error)

// LLMFunc is a function that generates text from a prompt (for reranking)
type LLMFunc = reranker.LLMFunc

// IndexRequest represents a document indexing request
type IndexRequest struct {
	ID         string
	Content    string
	Collection string
	Metadata   map[string]string
}

// SearchRequest represents a search request
type SearchRequest struct {
	Query      string
	Collection string
	TopK       int
	MinScore   float64
}

// SearchResult represents a search result
type SearchResult struct {
	ID       string
	Content  string
	Score    float64
	Metadata map[string]string
}

// CollectionInfo represents collection information
type CollectionInfo struct {
	Name          string
	DocumentCount int64
}

// Service is the Hypatia RAG service
type Service struct {
	store            vectorstore.Store
	chunker          *chunking.Chunker
	embedFunc        EmbeddingFunc
	llmFunc          LLMFunc
	reranker         reranker.Reranker
	expander         expansion.Expander
	logger           *logging.Logger
	defaultTopK      int
	minScore         float64
	enableReranking  bool
	enableExpansion  bool
}

// RerankStrategy defines the reranking strategy
type RerankStrategy string

const (
	RerankStrategyNone        RerankStrategy = "none"
	RerankStrategyKeyword     RerankStrategy = "keyword"
	RerankStrategyCrossEncoder RerankStrategy = "cross_encoder"
	RerankStrategyBatch       RerankStrategy = "batch"
	RerankStrategyComposite   RerankStrategy = "composite"
)

// ExpansionStrategy defines the query expansion strategy
type ExpansionStrategy string

const (
	ExpansionStrategyNone      ExpansionStrategy = "none"
	ExpansionStrategySynonym   ExpansionStrategy = "synonym"
	ExpansionStrategyLLM       ExpansionStrategy = "llm"
	ExpansionStrategyHyDE      ExpansionStrategy = "hyde"
	ExpansionStrategyComposite ExpansionStrategy = "composite"
)

// Config holds service configuration
type Config struct {
	ChunkSize       int
	ChunkOverlap    int
	ChunkStrategy   chunking.Strategy
	DefaultTopK     int
	MinRelevance    float64
	EmbeddingFunc   EmbeddingFunc
	LLMFunc         LLMFunc
	EnableReranking bool
	RerankStrategy  RerankStrategy

	// Query expansion configuration
	EnableExpansion   bool
	ExpansionStrategy ExpansionStrategy
	ExpansionLanguage string // "de" or "en"
	MaxExpandedQueries int
}

// DefaultConfig returns default configuration
func DefaultConfig() Config {
	return Config{
		ChunkSize:          1000,
		ChunkOverlap:       200,
		ChunkStrategy:      chunking.StrategyRecursive,
		DefaultTopK:        5,
		MinRelevance:       0.7,
		EnableReranking:    true,
		RerankStrategy:     RerankStrategyKeyword,
		EnableExpansion:    true,
		ExpansionStrategy:  ExpansionStrategySynonym,
		ExpansionLanguage:  "de",
		MaxExpandedQueries: 5,
	}
}

// NewService creates a new Hypatia service
func NewService(cfg Config, store vectorstore.Store) (*Service, error) {
	logger := logging.New("hypatia")

	chunkerCfg := chunking.Config{
		Strategy:     cfg.ChunkStrategy,
		ChunkSize:    cfg.ChunkSize,
		ChunkOverlap: cfg.ChunkOverlap,
	}
	chunker := chunking.NewChunker(chunkerCfg)

	// Initialize reranker based on strategy
	var rerankImpl reranker.Reranker
	if cfg.EnableReranking && cfg.RerankStrategy != RerankStrategyNone {
		switch cfg.RerankStrategy {
		case RerankStrategyKeyword:
			rerankImpl = reranker.NewKeywordBoostReranker(0.2)
			logger.Info("Keyword reranker enabled")
		case RerankStrategyCrossEncoder:
			if cfg.LLMFunc != nil {
				rerankImpl = reranker.NewCrossEncoderReranker(cfg.LLMFunc)
				logger.Info("Cross-encoder reranker enabled")
			} else {
				logger.Warn("Cross-encoder reranker requires LLM function, falling back to keyword")
				rerankImpl = reranker.NewKeywordBoostReranker(0.2)
			}
		case RerankStrategyBatch:
			if cfg.LLMFunc != nil {
				rerankImpl = reranker.NewBatchCrossEncoderReranker(cfg.LLMFunc, 5)
				logger.Info("Batch reranker enabled")
			} else {
				logger.Warn("Batch reranker requires LLM function, falling back to keyword")
				rerankImpl = reranker.NewKeywordBoostReranker(0.2)
			}
		case RerankStrategyComposite:
			rerankers := []reranker.Reranker{
				reranker.NewKeywordBoostReranker(0.2),
			}
			if cfg.LLMFunc != nil {
				rerankers = append(rerankers, reranker.NewBatchCrossEncoderReranker(cfg.LLMFunc, 5))
			}
			rerankImpl = reranker.NewCompositeReranker(rerankers...)
			logger.Info("Composite reranker enabled", "stages", len(rerankers))
		}
	}

	// Initialize query expander based on strategy
	var expanderImpl expansion.Expander
	if cfg.EnableExpansion && cfg.ExpansionStrategy != ExpansionStrategyNone {
		maxQueries := cfg.MaxExpandedQueries
		if maxQueries <= 0 {
			maxQueries = 5
		}

		switch cfg.ExpansionStrategy {
		case ExpansionStrategySynonym:
			expanderImpl = expansion.NewSynonymExpander(cfg.ExpansionLanguage)
			logger.Info("Synonym expander enabled", "language", cfg.ExpansionLanguage)
		case ExpansionStrategyLLM:
			if cfg.LLMFunc != nil {
				expanderImpl = expansion.NewLLMExpander(expansion.LLMFunc(cfg.LLMFunc), maxQueries)
				logger.Info("LLM expander enabled")
			} else {
				logger.Warn("LLM expander requires LLM function, falling back to synonym")
				expanderImpl = expansion.NewSynonymExpander(cfg.ExpansionLanguage)
			}
		case ExpansionStrategyHyDE:
			if cfg.LLMFunc != nil {
				expanderImpl = expansion.NewHypothesisExpander(expansion.LLMFunc(cfg.LLMFunc), maxQueries)
				logger.Info("HyDE expander enabled")
			} else {
				logger.Warn("HyDE expander requires LLM function, falling back to synonym")
				expanderImpl = expansion.NewSynonymExpander(cfg.ExpansionLanguage)
			}
		case ExpansionStrategyComposite:
			expanders := []expansion.Expander{
				expansion.NewSynonymExpander(cfg.ExpansionLanguage),
			}
			if cfg.LLMFunc != nil {
				expanders = append(expanders, expansion.NewLLMExpander(expansion.LLMFunc(cfg.LLMFunc), maxQueries))
			}
			expanderImpl = expansion.NewCompositeExpander(maxQueries, expanders...)
			logger.Info("Composite expander enabled", "stages", len(expanders))
		}
	}

	return &Service{
		store:            store,
		chunker:          chunker,
		embedFunc:        cfg.EmbeddingFunc,
		llmFunc:          cfg.LLMFunc,
		reranker:         rerankImpl,
		expander:         expanderImpl,
		logger:           logger,
		defaultTopK:      cfg.DefaultTopK,
		minScore:         cfg.MinRelevance,
		enableReranking:  cfg.EnableReranking && rerankImpl != nil,
		enableExpansion:  cfg.EnableExpansion && expanderImpl != nil,
	}, nil
}

// SetEmbeddingFunc sets the embedding function
func (s *Service) SetEmbeddingFunc(fn EmbeddingFunc) {
	s.embedFunc = fn
}

// SetLLMFunc sets the LLM function for reranking
func (s *Service) SetLLMFunc(fn LLMFunc) {
	s.llmFunc = fn
	// Update reranker if it's an LLM-based one
	if s.reranker != nil {
		if ce, ok := s.reranker.(*reranker.CrossEncoderReranker); ok {
			*ce = *reranker.NewCrossEncoderReranker(fn)
		} else if be, ok := s.reranker.(*reranker.BatchCrossEncoderReranker); ok {
			*be = *reranker.NewBatchCrossEncoderReranker(fn, 5)
		}
	}
}

// SetReranker sets a custom reranker
func (s *Service) SetReranker(r reranker.Reranker) {
	s.reranker = r
	s.enableReranking = r != nil
}

// SetExpander sets a custom query expander
func (s *Service) SetExpander(exp expansion.Expander) {
	s.expander = exp
	s.enableExpansion = exp != nil
}

// Index indexes a document
func (s *Service) Index(ctx context.Context, req *IndexRequest) error {
	if req.ID == "" {
		return fmt.Errorf("document ID is required")
	}
	if req.Content == "" {
		return fmt.Errorf("content is required")
	}
	if s.embedFunc == nil {
		return fmt.Errorf("embedding function not set")
	}

	s.logger.Info("Indexing document",
		"id", req.ID,
		"collection", req.Collection,
		"content_length", len(req.Content),
	)

	// Split into chunks
	chunks := s.chunker.Split(req.Content, req.ID)
	s.logger.Debug("Document chunked", "chunks", len(chunks))

	// Generate embeddings for all chunks
	texts := make([]string, len(chunks))
	for i, chunk := range chunks {
		texts[i] = chunk.Content
	}

	embeddings, err := s.embedFunc(ctx, texts)
	if err != nil {
		return fmt.Errorf("failed to generate embeddings: %w", err)
	}

	// Create documents and store
	docs := make([]*vectorstore.Document, len(chunks))
	for i, chunk := range chunks {
		metadata := make(map[string]string)
		for k, v := range req.Metadata {
			metadata[k] = v
		}
		metadata["chunk_index"] = fmt.Sprintf("%d", chunk.Index)
		metadata["parent_id"] = req.ID

		docs[i] = &vectorstore.Document{
			ID:         chunk.ID,
			Content:    chunk.Content,
			Embedding:  embeddings[i],
			Metadata:   metadata,
			Collection: req.Collection,
		}
	}

	if err := s.store.Insert(ctx, docs...); err != nil {
		return fmt.Errorf("failed to store documents: %w", err)
	}

	// Store a parent document record for GetDocument lookups
	parentMetadata := make(map[string]string)
	for k, v := range req.Metadata {
		parentMetadata[k] = v
	}
	parentMetadata["_type"] = "parent"
	parentMetadata["_chunk_count"] = fmt.Sprintf("%d", len(chunks))

	parentDoc := &vectorstore.Document{
		ID:         req.ID,
		Content:    req.Content,
		Collection: req.Collection,
		Metadata:   parentMetadata,
		// No embedding for parent - it's just for retrieval
	}
	if err := s.store.Insert(ctx, parentDoc); err != nil {
		s.logger.Warn("Failed to store parent document record", "error", err)
		// Don't fail - chunks are already stored
	}

	s.logger.Info("Document indexed",
		"id", req.ID,
		"chunks", len(chunks),
	)

	return nil
}

// Search performs semantic search with optional query expansion and reranking
func (s *Service) Search(ctx context.Context, req *SearchRequest) ([]SearchResult, error) {
	if req.Query == "" {
		return nil, fmt.Errorf("query is required")
	}
	if s.embedFunc == nil {
		return nil, fmt.Errorf("embedding function not set")
	}

	topK := req.TopK
	if topK <= 0 {
		topK = s.defaultTopK
	}

	minScore := req.MinScore
	if minScore <= 0 {
		minScore = s.minScore
	}

	s.logger.Info("Searching",
		"query", req.Query,
		"collection", req.Collection,
		"top_k", topK,
		"reranking", s.enableReranking,
		"expansion", s.enableExpansion,
	)

	// Apply query expansion if enabled
	queries := []string{req.Query}
	if s.enableExpansion && s.expander != nil {
		expansionResult, err := s.expander.Expand(ctx, req.Query)
		if err != nil {
			s.logger.Warn("Query expansion failed, using original query", "error", err)
		} else if len(expansionResult.ExpandedQueries) > 0 {
			queries = expansionResult.ExpandedQueries
			s.logger.Info("Query expanded",
				"original", req.Query,
				"expanded_count", len(queries),
			)
		}
	}

	// Generate embeddings for all queries
	embeddings, err := s.embedFunc(ctx, queries)
	if err != nil {
		return nil, fmt.Errorf("failed to generate query embeddings: %w", err)
	}

	// Fetch more results if reranking is enabled (to allow reranker to improve selection)
	fetchK := topK
	if s.enableReranking && s.reranker != nil {
		fetchK = topK * 3 // Fetch 3x for reranking pool
		if fetchK > 100 {
			fetchK = 100
		}
	}

	// Search for each query and merge results
	allResults := make(map[string]vectorstore.SearchResult)
	for i, embedding := range embeddings {
		queryResults, err := s.store.Search(ctx, embedding, req.Collection, fetchK, minScore*0.5)
		if err != nil {
			s.logger.Warn("Search failed for expanded query",
				"query_index", i,
				"error", err,
			)
			continue
		}

		// Merge results, taking the best score for each document
		for _, r := range queryResults {
			existing, exists := allResults[r.Document.ID]
			if !exists || r.Score > existing.Score {
				allResults[r.Document.ID] = r
			}
		}
	}

	// Convert map to slice
	storeResults := make([]vectorstore.SearchResult, 0, len(allResults))
	for _, r := range allResults {
		storeResults = append(storeResults, r)
	}

	// Sort by score descending
	for i := 0; i < len(storeResults)-1; i++ {
		for j := i + 1; j < len(storeResults); j++ {
			if storeResults[j].Score > storeResults[i].Score {
				storeResults[i], storeResults[j] = storeResults[j], storeResults[i]
			}
		}
	}

	s.logger.Info("Multi-query search completed",
		"queries", len(queries),
		"unique_results", len(storeResults),
	)

	// Apply reranking if enabled
	if s.enableReranking && s.reranker != nil && len(storeResults) > 0 {
		// Convert to reranker documents
		docs := make([]*reranker.Document, len(storeResults))
		for i, r := range storeResults {
			docs[i] = &reranker.Document{
				ID:       r.Document.ID,
				Content:  r.Document.Content,
				Score:    r.Score,
				Metadata: r.Document.Metadata,
			}
		}

		// Rerank
		rerankedResults, err := s.reranker.Rerank(ctx, req.Query, docs, topK)
		if err != nil {
			s.logger.Warn("Reranking failed, using original results", "error", err)
		} else {
			// Convert reranked results back
			results := make([]SearchResult, len(rerankedResults))
			for i, r := range rerankedResults {
				if r.FinalScore >= minScore {
					results[i] = SearchResult{
						ID:       r.Document.ID,
						Content:  r.Document.Content,
						Score:    r.FinalScore,
						Metadata: r.Document.Metadata,
					}
				}
			}
			// Filter out zero-score results
			filtered := make([]SearchResult, 0, len(results))
			for _, r := range results {
				if r.Score >= minScore {
					filtered = append(filtered, r)
				}
			}
			s.logger.Info("Search with reranking completed",
				"initial", len(storeResults),
				"reranked", len(filtered),
			)
			return filtered, nil
		}
	}

	// Convert results (no reranking or reranking failed)
	results := make([]SearchResult, 0, len(storeResults))
	for _, r := range storeResults {
		if r.Score >= minScore {
			results = append(results, SearchResult{
				ID:       r.Document.ID,
				Content:  r.Document.Content,
				Score:    r.Score,
				Metadata: r.Document.Metadata,
			})
		}
	}

	// Trim to topK
	if len(results) > topK {
		results = results[:topK]
	}

	s.logger.Info("Search completed",
		"results", len(results),
	)

	return results, nil
}

// Delete deletes a document
func (s *Service) Delete(ctx context.Context, id string) error {
	return s.store.Delete(ctx, id)
}

// CreateCollection creates a new empty collection
func (s *Service) CreateCollection(ctx context.Context, collection string) error {
	return s.store.CreateCollection(ctx, collection)
}

// ListCollections lists all collections
func (s *Service) ListCollections(ctx context.Context) ([]CollectionInfo, error) {
	names, err := s.store.ListCollections(ctx)
	if err != nil {
		return nil, err
	}

	infos := make([]CollectionInfo, len(names))
	for i, name := range names {
		count, _ := s.store.Count(ctx, name)
		infos[i] = CollectionInfo{
			Name:          name,
			DocumentCount: count,
		}
	}

	return infos, nil
}

// DeleteCollection deletes a collection
func (s *Service) DeleteCollection(ctx context.Context, collection string) error {
	return s.store.DeleteCollection(ctx, collection)
}

// GetDocument retrieves a document by ID
func (s *Service) GetDocument(ctx context.Context, id string) (*vectorstore.Document, error) {
	return s.store.Get(ctx, id)
}

// DocumentInfo represents document information
type DocumentInfo struct {
	ID          string
	Title       string
	Source      string
	Collection  string
	ChunkCount  int
	CreatedAt   time.Time
	Metadata    map[string]string
}

// ListDocuments lists documents in a collection with pagination
func (s *Service) ListDocuments(ctx context.Context, collection string, page, pageSize int) ([]DocumentInfo, int64, error) {
	if pageSize <= 0 {
		pageSize = 20
	}
	if page < 1 {
		page = 1
	}

	// Get all document IDs for the collection
	allDocs, err := s.getDocumentsForCollection(ctx, collection)
	if err != nil {
		return nil, 0, err
	}

	total := int64(len(allDocs))

	// Apply pagination
	start := (page - 1) * pageSize
	if start >= len(allDocs) {
		return []DocumentInfo{}, total, nil
	}
	end := start + pageSize
	if end > len(allDocs) {
		end = len(allDocs)
	}

	return allDocs[start:end], total, nil
}

// getDocumentsForCollection gets all documents for a collection, grouped by parent_id
func (s *Service) getDocumentsForCollection(ctx context.Context, collection string) ([]DocumentInfo, error) {
	// Use a dummy search with very low score to get all documents
	// This is a workaround - ideally the store would have a List method
	names, err := s.store.ListCollections(ctx)
	if err != nil {
		return nil, err
	}

	// Find the collection
	if collection == "" {
		collection = "default"
	}
	found := false
	for _, name := range names {
		if name == collection {
			found = true
			break
		}
	}
	if !found {
		return []DocumentInfo{}, nil
	}

	// Get document count and create basic info
	count, err := s.store.Count(ctx, collection)
	if err != nil {
		return nil, err
	}

	// For now, return a simple list based on count
	// A real implementation would iterate through actual documents
	docs := make([]DocumentInfo, 0)
	if count > 0 {
		docs = append(docs, DocumentInfo{
			ID:         collection + "_docs",
			Title:      collection,
			Collection: collection,
			ChunkCount: int(count),
			CreatedAt:  time.Now(),
		})
	}

	return docs, nil
}

// CollectionStats represents collection statistics
type CollectionStats struct {
	Name          string
	DocumentCount int64
	ChunkCount    int64
	TotalTokens   int64
	StorageBytes  int64
}

// GetCollectionStats gets statistics for a collection
func (s *Service) GetCollectionStats(ctx context.Context, collection string) (*CollectionStats, error) {
	count, err := s.store.Count(ctx, collection)
	if err != nil {
		return nil, err
	}

	// Estimate storage based on document count
	// Each chunk is roughly 1KB on average
	estimatedStorage := count * 1024

	return &CollectionStats{
		Name:          collection,
		DocumentCount: count, // This is actually chunk count in current implementation
		ChunkCount:    count,
		TotalTokens:   count * 256, // Rough estimate: 256 tokens per chunk
		StorageBytes:  estimatedStorage,
	}, nil
}

// HybridSearch performs a hybrid search combining vector and keyword search
func (s *Service) HybridSearch(ctx context.Context, req *SearchRequest, vectorWeight, keywordWeight float64) ([]SearchResult, error) {
	if req.Query == "" {
		return nil, fmt.Errorf("query is required")
	}
	if s.embedFunc == nil {
		return nil, fmt.Errorf("embedding function not set")
	}

	// Normalize weights
	totalWeight := vectorWeight + keywordWeight
	if totalWeight == 0 {
		vectorWeight = 0.7
		keywordWeight = 0.3
		totalWeight = 1.0
	}
	vectorWeight = vectorWeight / totalWeight
	keywordWeight = keywordWeight / totalWeight

	topK := req.TopK
	if topK <= 0 {
		topK = s.defaultTopK
	}

	minScore := req.MinScore
	if minScore <= 0 {
		minScore = s.minScore
	}

	s.logger.Info("Hybrid searching",
		"query", req.Query,
		"collection", req.Collection,
		"vector_weight", vectorWeight,
		"keyword_weight", keywordWeight,
	)

	// Perform vector search
	embeddings, err := s.embedFunc(ctx, []string{req.Query})
	if err != nil {
		return nil, fmt.Errorf("failed to generate query embedding: %w", err)
	}

	vectorResults, err := s.store.Search(ctx, embeddings[0], req.Collection, topK*2, 0)
	if err != nil {
		return nil, fmt.Errorf("vector search failed: %w", err)
	}

	// Perform keyword matching on vector results
	queryTerms := tokenize(req.Query)
	scoredResults := make(map[string]struct {
		doc         *vectorstore.Document
		vectorScore float64
		keywordScore float64
	})

	for _, r := range vectorResults {
		keywordScore := calculateKeywordScore(r.Document.Content, queryTerms)
		scoredResults[r.Document.ID] = struct {
			doc         *vectorstore.Document
			vectorScore float64
			keywordScore float64
		}{
			doc:          r.Document,
			vectorScore:  r.Score,
			keywordScore: keywordScore,
		}
	}

	// Combine scores and sort
	results := make([]SearchResult, 0, len(scoredResults))
	for _, scored := range scoredResults {
		combinedScore := scored.vectorScore*vectorWeight + scored.keywordScore*keywordWeight
		if combinedScore >= minScore {
			results = append(results, SearchResult{
				ID:       scored.doc.ID,
				Content:  scored.doc.Content,
				Score:    combinedScore,
				Metadata: scored.doc.Metadata,
			})
		}
	}

	// Sort by score descending
	for i := 0; i < len(results)-1; i++ {
		for j := i + 1; j < len(results); j++ {
			if results[j].Score > results[i].Score {
				results[i], results[j] = results[j], results[i]
			}
		}
	}

	// Take top K
	if topK > 0 && len(results) > topK {
		results = results[:topK]
	}

	s.logger.Info("Hybrid search completed",
		"results", len(results),
	)

	return results, nil
}

// tokenize splits text into lowercase tokens
func tokenize(text string) []string {
	words := make([]string, 0)
	current := ""
	for _, r := range text {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == 'ä' || r == 'ö' || r == 'ü' || r == 'ß' {
			if r >= 'A' && r <= 'Z' {
				r = r + 32 // lowercase
			}
			current += string(r)
		} else if current != "" {
			if len(current) > 2 { // Ignore very short words
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

// HealthCheck checks if the service is healthy
func (s *Service) HealthCheck(ctx context.Context) error {
	// Try to list collections as a simple health check
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	_, err := s.store.ListCollections(ctx)
	return err
}

// Close closes the service
func (s *Service) Close() error {
	return s.store.Close()
}
