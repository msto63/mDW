package vectorstore

import (
	"context"
	"fmt"
	"math"
	"sync"
)

// Document represents a stored document
type Document struct {
	ID         string
	Content    string
	Embedding  []float64
	Metadata   map[string]string
	Collection string
}

// SearchResult represents a search result
type SearchResult struct {
	Document *Document
	Score    float64
}

// Store is the interface for vector stores
type Store interface {
	// Insert adds documents to the store
	Insert(ctx context.Context, docs ...*Document) error

	// Search performs similarity search
	Search(ctx context.Context, embedding []float64, collection string, topK int, minScore float64) ([]SearchResult, error)

	// Get retrieves a document by ID
	Get(ctx context.Context, id string) (*Document, error)

	// Delete removes a document by ID
	Delete(ctx context.Context, id string) error

	// CreateCollection creates an empty collection
	CreateCollection(ctx context.Context, collection string) error

	// ListCollections returns all collection names
	ListCollections(ctx context.Context) ([]string, error)

	// DeleteCollection removes an entire collection
	DeleteCollection(ctx context.Context, collection string) error

	// Count returns the number of documents in a collection
	Count(ctx context.Context, collection string) (int64, error)

	// Close closes the store
	Close() error
}

// MemoryStore is an in-memory vector store for development
type MemoryStore struct {
	mu          sync.RWMutex
	documents   map[string]*Document
	collections map[string][]string // collection -> document IDs
	norms       map[string]float64  // document ID -> pre-computed norm
}

// NewMemoryStore creates a new in-memory store
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		documents:   make(map[string]*Document),
		collections: make(map[string][]string),
		norms:       make(map[string]float64),
	}
}

// Insert adds documents to the store
func (s *MemoryStore) Insert(ctx context.Context, docs ...*Document) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, doc := range docs {
		if doc.ID == "" {
			return fmt.Errorf("document ID is required")
		}
		if doc.Collection == "" {
			doc.Collection = "default"
		}

		s.documents[doc.ID] = doc

		// Pre-compute and cache the embedding norm
		if len(doc.Embedding) > 0 {
			s.norms[doc.ID] = vectorNorm(doc.Embedding)
		}

		// Add to collection index
		found := false
		for _, id := range s.collections[doc.Collection] {
			if id == doc.ID {
				found = true
				break
			}
		}
		if !found {
			s.collections[doc.Collection] = append(s.collections[doc.Collection], doc.ID)
		}
	}

	return nil
}

// scoredDocMem holds a document with its similarity score for the memory store
type scoredDocMem struct {
	doc   *Document
	score float64
}

// Search performs similarity search using cosine similarity with optimized top-k
func (s *MemoryStore) Search(ctx context.Context, embedding []float64, collection string, topK int, minScore float64) ([]SearchResult, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if collection == "" {
		collection = "default"
	}
	if topK <= 0 {
		topK = 10
	}

	docIDs, ok := s.collections[collection]
	if !ok {
		return []SearchResult{}, nil
	}

	// Pre-compute query norm once
	queryNorm := vectorNorm(embedding)
	if queryNorm == 0 {
		return []SearchResult{}, nil
	}

	// Use min-heap for efficient top-k selection
	heap := make([]scoredDocMem, 0, topK+1)

	for _, id := range docIDs {
		doc := s.documents[id]
		if doc == nil || len(doc.Embedding) == 0 {
			continue
		}

		// Use pre-computed norm if available
		docNorm := s.norms[id]
		if docNorm == 0 {
			docNorm = vectorNorm(doc.Embedding)
		}

		score := cosineSimilarityWithNorms(embedding, doc.Embedding, queryNorm, docNorm)
		if score >= minScore {
			// Add to heap
			heap = append(heap, scoredDocMem{doc: doc, score: score})
			heapifyUpMem(heap, len(heap)-1)

			// Remove minimum if over capacity
			if len(heap) > topK {
				heap[0] = heap[len(heap)-1]
				heap = heap[:len(heap)-1]
				heapifyDownMem(heap, 0)
			}
		}
	}

	// Sort results descending by score
	for i := 0; i < len(heap)-1; i++ {
		for j := i + 1; j < len(heap); j++ {
			if heap[j].score > heap[i].score {
				heap[i], heap[j] = heap[j], heap[i]
			}
		}
	}

	// Convert to SearchResult
	searchResults := make([]SearchResult, len(heap))
	for i, r := range heap {
		searchResults[i] = SearchResult{
			Document: r.doc,
			Score:    r.score,
		}
	}

	return searchResults, nil
}

// heapifyUpMem maintains min-heap property going up
func heapifyUpMem(heap []scoredDocMem, i int) {
	for i > 0 {
		parent := (i - 1) / 2
		if heap[i].score >= heap[parent].score {
			break
		}
		heap[i], heap[parent] = heap[parent], heap[i]
		i = parent
	}
}

// heapifyDownMem maintains min-heap property going down
func heapifyDownMem(heap []scoredDocMem, i int) {
	n := len(heap)
	for {
		smallest := i
		left := 2*i + 1
		right := 2*i + 2

		if left < n && heap[left].score < heap[smallest].score {
			smallest = left
		}
		if right < n && heap[right].score < heap[smallest].score {
			smallest = right
		}
		if smallest == i {
			break
		}
		heap[i], heap[smallest] = heap[smallest], heap[i]
		i = smallest
	}
}

// Get retrieves a document by ID
func (s *MemoryStore) Get(ctx context.Context, id string) (*Document, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	doc, ok := s.documents[id]
	if !ok {
		return nil, fmt.Errorf("document not found: %s", id)
	}
	return doc, nil
}

// Delete removes a document by ID
func (s *MemoryStore) Delete(ctx context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	doc, ok := s.documents[id]
	if !ok {
		return nil
	}

	// Remove from collection index
	if ids, exists := s.collections[doc.Collection]; exists {
		for i, docID := range ids {
			if docID == id {
				s.collections[doc.Collection] = append(ids[:i], ids[i+1:]...)
				break
			}
		}
	}

	delete(s.documents, id)
	delete(s.norms, id)
	return nil
}

// CreateCollection creates an empty collection
func (s *MemoryStore) CreateCollection(ctx context.Context, collection string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if collection == "" {
		return fmt.Errorf("collection name is required")
	}

	// Only create if doesn't exist
	if _, exists := s.collections[collection]; !exists {
		s.collections[collection] = []string{}
	}
	return nil
}

// ListCollections returns all collection names
func (s *MemoryStore) ListCollections(ctx context.Context) ([]string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	collections := make([]string, 0, len(s.collections))
	for name := range s.collections {
		collections = append(collections, name)
	}
	return collections, nil
}

// DeleteCollection removes an entire collection
func (s *MemoryStore) DeleteCollection(ctx context.Context, collection string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	docIDs, ok := s.collections[collection]
	if !ok {
		return nil
	}

	for _, id := range docIDs {
		delete(s.documents, id)
	}
	delete(s.collections, collection)

	return nil
}

// Count returns the number of documents in a collection
func (s *MemoryStore) Count(ctx context.Context, collection string) (int64, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if collection == "" {
		return int64(len(s.documents)), nil
	}

	ids, ok := s.collections[collection]
	if !ok {
		return 0, nil
	}
	return int64(len(ids)), nil
}

// Close closes the store
func (s *MemoryStore) Close() error {
	return nil
}

// vectorNorm calculates the L2 norm of a vector
func vectorNorm(v []float64) float64 {
	var sum float64
	for _, x := range v {
		sum += x * x
	}
	return math.Sqrt(sum)
}

// cosineSimilarityWithNorms calculates cosine similarity using pre-computed norms
func cosineSimilarityWithNorms(a, b []float64, normA, normB float64) float64 {
	if len(a) != len(b) || len(a) == 0 {
		return 0
	}
	if normA == 0 || normB == 0 {
		return 0
	}

	var dotProduct float64
	for i := range a {
		dotProduct += a[i] * b[i]
	}

	return dotProduct / (normA * normB)
}

// cosineSimilarity calculates cosine similarity between two vectors (fallback)
func cosineSimilarity(a, b []float64) float64 {
	if len(a) != len(b) {
		return 0
	}

	var dotProduct, normA, normB float64
	for i := range a {
		dotProduct += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}

	if normA == 0 || normB == 0 {
		return 0
	}

	return dotProduct / (math.Sqrt(normA) * math.Sqrt(normB))
}
