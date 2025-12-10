package vectorstore

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"
	"sync"

	_ "github.com/mattn/go-sqlite3"
)

// SQLiteStore is a SQLite-based vector store
// It can optionally use the sqlite-vec extension for efficient vector search
type SQLiteStore struct {
	db         *sql.DB
	mu         sync.RWMutex
	hasVecExt  bool
	dimensions int
}

// SQLiteConfig holds SQLite store configuration
type SQLiteConfig struct {
	Path       string
	Dimensions int
	VecExtPath string // Optional path to sqlite-vec extension
}

// DefaultSQLiteConfig returns default SQLite configuration
func DefaultSQLiteConfig() SQLiteConfig {
	return SQLiteConfig{
		Path:       "./data/vectors.db",
		Dimensions: 768, // Default for nomic-embed-text
	}
}

// NewSQLiteStore creates a new SQLite-based vector store
func NewSQLiteStore(cfg SQLiteConfig) (*SQLiteStore, error) {
	// Ensure directory exists
	dir := filepath.Dir(cfg.Path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create directory: %w", err)
	}

	// Open database with WAL mode for better concurrent access
	db, err := sql.Open("sqlite3", cfg.Path+"?_journal_mode=WAL&_synchronous=NORMAL")
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	store := &SQLiteStore{
		db:         db,
		dimensions: cfg.Dimensions,
	}

	// Try to load sqlite-vec extension if path provided
	if cfg.VecExtPath != "" {
		store.tryLoadVecExtension(cfg.VecExtPath)
	}

	// Initialize schema
	if err := store.initSchema(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to initialize schema: %w", err)
	}

	return store, nil
}

// tryLoadVecExtension attempts to load the sqlite-vec extension
func (s *SQLiteStore) tryLoadVecExtension(path string) {
	// sqlite-vec extension loading would go here
	// For now, we use pure Go implementation as fallback
	s.hasVecExt = false
}

// initSchema creates the necessary tables
func (s *SQLiteStore) initSchema() error {
	schema := `
	-- Documents table
	CREATE TABLE IF NOT EXISTS documents (
		id TEXT PRIMARY KEY,
		content TEXT NOT NULL,
		collection TEXT NOT NULL DEFAULT 'default',
		metadata TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	-- Embeddings table with pre-computed norm for faster search
	CREATE TABLE IF NOT EXISTS embeddings (
		document_id TEXT PRIMARY KEY,
		embedding BLOB NOT NULL,
		dimensions INTEGER NOT NULL,
		norm REAL NOT NULL DEFAULT 0,
		FOREIGN KEY (document_id) REFERENCES documents(id) ON DELETE CASCADE
	);

	-- Indices for faster queries
	CREATE INDEX IF NOT EXISTS idx_documents_collection ON documents(collection);
	CREATE INDEX IF NOT EXISTS idx_embeddings_doc ON embeddings(document_id);
	`

	if _, err := s.db.Exec(schema); err != nil {
		return err
	}

	// Migration: add norm column if it doesn't exist (for existing databases)
	s.db.Exec(`ALTER TABLE embeddings ADD COLUMN norm REAL NOT NULL DEFAULT 0`)

	return nil
}

// Insert adds documents to the store
func (s *SQLiteStore) Insert(ctx context.Context, docs ...*Document) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	docStmt, err := tx.PrepareContext(ctx, `
		INSERT OR REPLACE INTO documents (id, content, collection, metadata)
		VALUES (?, ?, ?, ?)
	`)
	if err != nil {
		return fmt.Errorf("failed to prepare document statement: %w", err)
	}
	defer docStmt.Close()

	embStmt, err := tx.PrepareContext(ctx, `
		INSERT OR REPLACE INTO embeddings (document_id, embedding, dimensions, norm)
		VALUES (?, ?, ?, ?)
	`)
	if err != nil {
		return fmt.Errorf("failed to prepare embedding statement: %w", err)
	}
	defer embStmt.Close()

	for _, doc := range docs {
		if doc.ID == "" {
			return fmt.Errorf("document ID is required")
		}
		if doc.Collection == "" {
			doc.Collection = "default"
		}

		// Serialize metadata
		var metadataJSON []byte
		if doc.Metadata != nil {
			metadataJSON, _ = json.Marshal(doc.Metadata)
		}

		// Insert document
		_, err = docStmt.ExecContext(ctx, doc.ID, doc.Content, doc.Collection, metadataJSON)
		if err != nil {
			return fmt.Errorf("failed to insert document: %w", err)
		}

		// Insert embedding with pre-computed norm
		if len(doc.Embedding) > 0 {
			embBytes := serializeEmbedding(doc.Embedding)
			norm := computeNorm(doc.Embedding)
			_, err = embStmt.ExecContext(ctx, doc.ID, embBytes, len(doc.Embedding), norm)
			if err != nil {
				return fmt.Errorf("failed to insert embedding: %w", err)
			}
		}
	}

	return tx.Commit()
}

// Search performs similarity search with optimized top-k selection
func (s *SQLiteStore) Search(ctx context.Context, embedding []float64, collection string, topK int, minScore float64) ([]SearchResult, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if collection == "" {
		collection = "default"
	}
	if topK <= 0 {
		topK = 10
	}

	// Pre-compute query embedding norm
	queryNorm := computeNorm(embedding)
	if queryNorm == 0 {
		return []SearchResult{}, nil
	}

	// Query documents with pre-computed norms for faster similarity calculation
	rows, err := s.db.QueryContext(ctx, `
		SELECT d.id, d.content, d.collection, d.metadata, e.embedding, e.dimensions, e.norm
		FROM documents d
		JOIN embeddings e ON d.id = e.document_id
		WHERE d.collection = ?
	`, collection)
	if err != nil {
		return nil, fmt.Errorf("failed to query documents: %w", err)
	}
	defer rows.Close()

	// Use a min-heap to maintain top-k results efficiently
	heap := newScoreHeap(topK)

	for rows.Next() {
		var id, content, col string
		var metadataJSON sql.NullString
		var embBytes []byte
		var dimensions int
		var docNorm float64

		if err := rows.Scan(&id, &content, &col, &metadataJSON, &embBytes, &dimensions, &docNorm); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}

		// Skip if norm is zero (shouldn't happen but safety check)
		if docNorm == 0 {
			continue
		}

		// Deserialize embedding
		docEmbedding := deserializeEmbedding(embBytes, dimensions)

		// Calculate similarity using pre-computed norms
		score := cosineSimWithNorms(embedding, docEmbedding, queryNorm, docNorm)
		if score >= minScore {
			doc := &Document{
				ID:         id,
				Content:    content,
				Collection: col,
				Embedding:  docEmbedding,
			}

			// Parse metadata
			if metadataJSON.Valid {
				json.Unmarshal([]byte(metadataJSON.String), &doc.Metadata)
			}

			heap.push(scoredDoc{doc: doc, score: score})
		}
	}

	// Extract results sorted by score descending
	return heap.toResults(), nil
}

// scoredDoc holds a document with its similarity score
type scoredDoc struct {
	doc   *Document
	score float64
}

// scoreHeap is a min-heap for efficient top-k selection
type scoreHeap struct {
	items []scoredDoc
	k     int
}

func newScoreHeap(k int) *scoreHeap {
	return &scoreHeap{
		items: make([]scoredDoc, 0, k+1),
		k:     k,
	}
}

func (h *scoreHeap) push(item scoredDoc) {
	h.items = append(h.items, item)
	h.heapifyUp(len(h.items) - 1)

	// If over capacity, remove minimum (root)
	if len(h.items) > h.k {
		h.items[0] = h.items[len(h.items)-1]
		h.items = h.items[:len(h.items)-1]
		h.heapifyDown(0)
	}
}

func (h *scoreHeap) heapifyUp(i int) {
	for i > 0 {
		parent := (i - 1) / 2
		if h.items[i].score >= h.items[parent].score {
			break
		}
		h.items[i], h.items[parent] = h.items[parent], h.items[i]
		i = parent
	}
}

func (h *scoreHeap) heapifyDown(i int) {
	n := len(h.items)
	for {
		smallest := i
		left := 2*i + 1
		right := 2*i + 2

		if left < n && h.items[left].score < h.items[smallest].score {
			smallest = left
		}
		if right < n && h.items[right].score < h.items[smallest].score {
			smallest = right
		}
		if smallest == i {
			break
		}
		h.items[i], h.items[smallest] = h.items[smallest], h.items[i]
		i = smallest
	}
}

func (h *scoreHeap) toResults() []SearchResult {
	// Sort descending by score
	n := len(h.items)
	for i := 0; i < n-1; i++ {
		for j := i + 1; j < n; j++ {
			if h.items[j].score > h.items[i].score {
				h.items[i], h.items[j] = h.items[j], h.items[i]
			}
		}
	}

	results := make([]SearchResult, n)
	for i, item := range h.items {
		results[i] = SearchResult{
			Document: item.doc,
			Score:    item.score,
		}
	}
	return results
}

// Get retrieves a document by ID
func (s *SQLiteStore) Get(ctx context.Context, id string) (*Document, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	row := s.db.QueryRowContext(ctx, `
		SELECT d.id, d.content, d.collection, d.metadata, e.embedding, e.dimensions
		FROM documents d
		LEFT JOIN embeddings e ON d.id = e.document_id
		WHERE d.id = ?
	`, id)

	var docID, content, collection string
	var metadataJSON sql.NullString
	var embBytes []byte
	var dimensions sql.NullInt64

	if err := row.Scan(&docID, &content, &collection, &metadataJSON, &embBytes, &dimensions); err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("document not found: %s", id)
		}
		return nil, fmt.Errorf("failed to scan document: %w", err)
	}

	doc := &Document{
		ID:         docID,
		Content:    content,
		Collection: collection,
	}

	if metadataJSON.Valid {
		json.Unmarshal([]byte(metadataJSON.String), &doc.Metadata)
	}

	if dimensions.Valid && len(embBytes) > 0 {
		doc.Embedding = deserializeEmbedding(embBytes, int(dimensions.Int64))
	}

	return doc, nil
}

// Delete removes a document by ID
func (s *SQLiteStore) Delete(ctx context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	_, err := s.db.ExecContext(ctx, `DELETE FROM documents WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("failed to delete document: %w", err)
	}

	// Embeddings are deleted automatically due to CASCADE
	return nil
}

// CreateCollection creates an empty collection (or ensures it exists)
func (s *SQLiteStore) CreateCollection(ctx context.Context, collection string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if collection == "" {
		return fmt.Errorf("collection name is required")
	}

	// Insert a marker row to register the collection
	// We use a special document ID prefix to mark empty collections
	_, err := s.db.ExecContext(ctx, `
		INSERT OR IGNORE INTO documents (id, content, collection, created_at)
		VALUES (?, '', ?, datetime('now'))
	`, "__collection_marker__"+collection, collection)
	if err != nil {
		return fmt.Errorf("failed to create collection: %w", err)
	}

	return nil
}

// ListCollections returns all collection names
func (s *SQLiteStore) ListCollections(ctx context.Context) ([]string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	rows, err := s.db.QueryContext(ctx, `
		SELECT DISTINCT collection FROM documents ORDER BY collection
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to list collections: %w", err)
	}
	defer rows.Close()

	var collections []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, fmt.Errorf("failed to scan collection: %w", err)
		}
		collections = append(collections, name)
	}

	return collections, nil
}

// DeleteCollection removes an entire collection
func (s *SQLiteStore) DeleteCollection(ctx context.Context, collection string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	_, err := s.db.ExecContext(ctx, `DELETE FROM documents WHERE collection = ?`, collection)
	if err != nil {
		return fmt.Errorf("failed to delete collection: %w", err)
	}

	return nil
}

// Count returns the number of documents in a collection
func (s *SQLiteStore) Count(ctx context.Context, collection string) (int64, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var query string
	var args []interface{}

	if collection == "" {
		query = `SELECT COUNT(*) FROM documents`
	} else {
		query = `SELECT COUNT(*) FROM documents WHERE collection = ?`
		args = append(args, collection)
	}

	var count int64
	err := s.db.QueryRowContext(ctx, query, args...).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count documents: %w", err)
	}

	return count, nil
}

// Close closes the database connection
func (s *SQLiteStore) Close() error {
	return s.db.Close()
}

// Statistics returns store statistics
func (s *SQLiteStore) Statistics(ctx context.Context) (map[string]interface{}, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	stats := make(map[string]interface{})

	// Total documents
	var totalDocs int64
	s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM documents`).Scan(&totalDocs)
	stats["total_documents"] = totalDocs

	// Total embeddings
	var totalEmb int64
	s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM embeddings`).Scan(&totalEmb)
	stats["total_embeddings"] = totalEmb

	// Collections
	collections, _ := s.ListCollections(ctx)
	stats["collections"] = collections
	stats["collection_count"] = len(collections)

	// Has vector extension
	stats["has_vec_extension"] = s.hasVecExt

	return stats, nil
}

// Vacuum performs database optimization
func (s *SQLiteStore) Vacuum(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	_, err := s.db.ExecContext(ctx, `VACUUM`)
	return err
}

// Helper functions

// serializeEmbedding converts a float64 slice to bytes (little-endian float32)
func serializeEmbedding(embedding []float64) []byte {
	bytes := make([]byte, len(embedding)*4)
	for i, v := range embedding {
		f32 := float32(v)
		bits := math.Float32bits(f32)
		bytes[i*4] = byte(bits)
		bytes[i*4+1] = byte(bits >> 8)
		bytes[i*4+2] = byte(bits >> 16)
		bytes[i*4+3] = byte(bits >> 24)
	}
	return bytes
}

// deserializeEmbedding converts bytes back to float64 slice
func deserializeEmbedding(bytes []byte, dimensions int) []float64 {
	embedding := make([]float64, dimensions)
	for i := 0; i < dimensions && i*4+4 <= len(bytes); i++ {
		bits := uint32(bytes[i*4]) |
			uint32(bytes[i*4+1])<<8 |
			uint32(bytes[i*4+2])<<16 |
			uint32(bytes[i*4+3])<<24
		embedding[i] = float64(math.Float32frombits(bits))
	}
	return embedding
}

// computeNorm calculates the L2 norm of a vector
func computeNorm(v []float64) float64 {
	var sum float64
	for _, x := range v {
		sum += x * x
	}
	return math.Sqrt(sum)
}

// cosineSimWithNorms calculates cosine similarity using pre-computed norms
func cosineSimWithNorms(a, b []float64, normA, normB float64) float64 {
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

// cosineSimSQLite calculates cosine similarity using math package (fallback)
func cosineSimSQLite(a, b []float64) float64 {
	if len(a) != len(b) || len(a) == 0 {
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

// NewStore creates a store based on configuration
func NewStore(storeType, path string, dimensions int) (Store, error) {
	switch strings.ToLower(storeType) {
	case "sqlite", "sqlite3":
		return NewSQLiteStore(SQLiteConfig{
			Path:       path,
			Dimensions: dimensions,
		})
	case "memory", "":
		return NewMemoryStore(), nil
	default:
		return nil, fmt.Errorf("unsupported store type: %s", storeType)
	}
}
