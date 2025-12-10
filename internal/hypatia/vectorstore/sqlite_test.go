package vectorstore

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultSQLiteConfig(t *testing.T) {
	cfg := DefaultSQLiteConfig()

	if cfg.Path != "./data/vectors.db" {
		t.Errorf("Path = %v, want ./data/vectors.db", cfg.Path)
	}
	if cfg.Dimensions != 768 {
		t.Errorf("Dimensions = %v, want 768", cfg.Dimensions)
	}
}

func TestNewSQLiteStore(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	store, err := NewSQLiteStore(SQLiteConfig{
		Path:       dbPath,
		Dimensions: 384,
	})
	if err != nil {
		t.Fatalf("NewSQLiteStore() error = %v", err)
	}
	defer store.Close()

	if store.db == nil {
		t.Error("db should not be nil")
	}
	if store.dimensions != 384 {
		t.Errorf("dimensions = %v, want 384", store.dimensions)
	}
}

func TestSQLiteStore_Insert(t *testing.T) {
	store := createTestSQLiteStore(t)
	defer store.Close()
	ctx := context.Background()

	doc := &Document{
		ID:         "doc1",
		Content:    "Test content",
		Collection: "test",
		Embedding:  []float64{0.1, 0.2, 0.3},
		Metadata:   map[string]string{"key": "value"},
	}

	err := store.Insert(ctx, doc)
	if err != nil {
		t.Fatalf("Insert() error = %v", err)
	}

	// Verify document was inserted
	retrieved, err := store.Get(ctx, "doc1")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	if retrieved.Content != "Test content" {
		t.Errorf("Content = %v, want 'Test content'", retrieved.Content)
	}
	if retrieved.Collection != "test" {
		t.Errorf("Collection = %v, want 'test'", retrieved.Collection)
	}
	if len(retrieved.Embedding) != 3 {
		t.Errorf("Embedding length = %v, want 3", len(retrieved.Embedding))
	}
}

func TestSQLiteStore_Insert_EmptyID(t *testing.T) {
	store := createTestSQLiteStore(t)
	defer store.Close()
	ctx := context.Background()

	err := store.Insert(ctx, &Document{Content: "No ID"})
	if err == nil {
		t.Error("Insert() should return error for empty ID")
	}
}

func TestSQLiteStore_Insert_DefaultCollection(t *testing.T) {
	store := createTestSQLiteStore(t)
	defer store.Close()
	ctx := context.Background()

	doc := &Document{
		ID:      "doc1",
		Content: "Test",
	}
	store.Insert(ctx, doc)

	retrieved, _ := store.Get(ctx, "doc1")
	if retrieved.Collection != "default" {
		t.Errorf("Collection = %v, want 'default'", retrieved.Collection)
	}
}

func TestSQLiteStore_Search(t *testing.T) {
	store := createTestSQLiteStore(t)
	defer store.Close()
	ctx := context.Background()

	// Insert test documents
	docs := []*Document{
		{ID: "doc1", Content: "First", Collection: "test", Embedding: []float64{1.0, 0.0, 0.0}},
		{ID: "doc2", Content: "Second", Collection: "test", Embedding: []float64{0.9, 0.1, 0.0}},
		{ID: "doc3", Content: "Third", Collection: "test", Embedding: []float64{0.0, 1.0, 0.0}},
	}
	store.Insert(ctx, docs...)

	// Search for similar to doc1
	results, err := store.Search(ctx, []float64{1.0, 0.0, 0.0}, "test", 2, 0.5)
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}

	if len(results) != 2 {
		t.Errorf("Results count = %v, want 2", len(results))
	}

	// First result should be doc1 (perfect match)
	if len(results) > 0 && results[0].Document.ID != "doc1" {
		t.Errorf("First result ID = %v, want doc1", results[0].Document.ID)
	}
}

func TestSQLiteStore_Search_EmptyCollection(t *testing.T) {
	store := createTestSQLiteStore(t)
	defer store.Close()
	ctx := context.Background()

	results, err := store.Search(ctx, []float64{1.0, 0.0}, "nonexistent", 5, 0.0)
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}

	if len(results) != 0 {
		t.Errorf("Results count = %v, want 0", len(results))
	}
}

func TestSQLiteStore_Get_NotFound(t *testing.T) {
	store := createTestSQLiteStore(t)
	defer store.Close()
	ctx := context.Background()

	_, err := store.Get(ctx, "nonexistent")
	if err == nil {
		t.Error("Get() should return error for nonexistent document")
	}
}

func TestSQLiteStore_Delete(t *testing.T) {
	store := createTestSQLiteStore(t)
	defer store.Close()
	ctx := context.Background()

	store.Insert(ctx, &Document{ID: "doc1", Content: "Test"})

	err := store.Delete(ctx, "doc1")
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	_, err = store.Get(ctx, "doc1")
	if err == nil {
		t.Error("Document should be deleted")
	}
}

func TestSQLiteStore_Delete_NonExistent(t *testing.T) {
	store := createTestSQLiteStore(t)
	defer store.Close()
	ctx := context.Background()

	err := store.Delete(ctx, "nonexistent")
	if err != nil {
		t.Errorf("Delete() should not error for nonexistent: %v", err)
	}
}

func TestSQLiteStore_ListCollections(t *testing.T) {
	store := createTestSQLiteStore(t)
	defer store.Close()
	ctx := context.Background()

	store.Insert(ctx,
		&Document{ID: "doc1", Content: "A", Collection: "alpha"},
		&Document{ID: "doc2", Content: "B", Collection: "beta"},
		&Document{ID: "doc3", Content: "C", Collection: "alpha"},
	)

	collections, err := store.ListCollections(ctx)
	if err != nil {
		t.Fatalf("ListCollections() error = %v", err)
	}

	if len(collections) != 2 {
		t.Errorf("Collections count = %v, want 2", len(collections))
	}
}

func TestSQLiteStore_DeleteCollection(t *testing.T) {
	store := createTestSQLiteStore(t)
	defer store.Close()
	ctx := context.Background()

	store.Insert(ctx,
		&Document{ID: "doc1", Content: "A", Collection: "test"},
		&Document{ID: "doc2", Content: "B", Collection: "test"},
		&Document{ID: "doc3", Content: "C", Collection: "other"},
	)

	err := store.DeleteCollection(ctx, "test")
	if err != nil {
		t.Fatalf("DeleteCollection() error = %v", err)
	}

	count, _ := store.Count(ctx, "test")
	if count != 0 {
		t.Errorf("Count after delete = %v, want 0", count)
	}

	count, _ = store.Count(ctx, "other")
	if count != 1 {
		t.Errorf("Other collection count = %v, want 1", count)
	}
}

func TestSQLiteStore_Count(t *testing.T) {
	store := createTestSQLiteStore(t)
	defer store.Close()
	ctx := context.Background()

	store.Insert(ctx,
		&Document{ID: "doc1", Content: "A", Collection: "test"},
		&Document{ID: "doc2", Content: "B", Collection: "test"},
		&Document{ID: "doc3", Content: "C", Collection: "other"},
	)

	count, err := store.Count(ctx, "test")
	if err != nil {
		t.Fatalf("Count() error = %v", err)
	}
	if count != 2 {
		t.Errorf("Count = %v, want 2", count)
	}

	totalCount, _ := store.Count(ctx, "")
	if totalCount != 3 {
		t.Errorf("Total count = %v, want 3", totalCount)
	}
}

func TestSQLiteStore_Statistics(t *testing.T) {
	store := createTestSQLiteStore(t)
	defer store.Close()
	ctx := context.Background()

	store.Insert(ctx,
		&Document{ID: "doc1", Content: "A", Collection: "test", Embedding: []float64{0.1}},
		&Document{ID: "doc2", Content: "B", Collection: "test", Embedding: []float64{0.2}},
	)

	stats, err := store.Statistics(ctx)
	if err != nil {
		t.Fatalf("Statistics() error = %v", err)
	}

	if stats["total_documents"].(int64) != 2 {
		t.Errorf("total_documents = %v, want 2", stats["total_documents"])
	}
	if stats["total_embeddings"].(int64) != 2 {
		t.Errorf("total_embeddings = %v, want 2", stats["total_embeddings"])
	}
}

func TestSQLiteStore_Vacuum(t *testing.T) {
	store := createTestSQLiteStore(t)
	defer store.Close()
	ctx := context.Background()

	err := store.Vacuum(ctx)
	if err != nil {
		t.Errorf("Vacuum() error = %v", err)
	}
}

func TestSerializeDeserializeEmbedding(t *testing.T) {
	original := []float64{0.1, 0.2, 0.3, 0.4, 0.5}
	serialized := serializeEmbedding(original)
	deserialized := deserializeEmbedding(serialized, len(original))

	for i := range original {
		// Allow small floating point difference due to float32 conversion
		diff := original[i] - deserialized[i]
		if diff > 0.0001 || diff < -0.0001 {
			t.Errorf("deserialized[%d] = %v, want ~%v", i, deserialized[i], original[i])
		}
	}
}

func TestCosineSimSQLite(t *testing.T) {
	tests := []struct {
		name     string
		a        []float64
		b        []float64
		expected float64
	}{
		{"identical", []float64{1, 0, 0}, []float64{1, 0, 0}, 1.0},
		{"orthogonal", []float64{1, 0, 0}, []float64{0, 1, 0}, 0.0},
		{"opposite", []float64{1, 0, 0}, []float64{-1, 0, 0}, -1.0},
		{"partial", []float64{1, 1, 0}, []float64{1, 0, 0}, 0.7071}, // sqrt(2)/2
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := cosineSimSQLite(tt.a, tt.b)
			diff := result - tt.expected
			if diff > 0.001 || diff < -0.001 {
				t.Errorf("cosineSimSQLite() = %v, want ~%v", result, tt.expected)
			}
		})
	}
}

func TestCosineSimSQLite_EdgeCases(t *testing.T) {
	// Different lengths
	result := cosineSimSQLite([]float64{1, 2}, []float64{1, 2, 3})
	if result != 0 {
		t.Errorf("Different lengths should return 0, got %v", result)
	}

	// Empty vectors
	result = cosineSimSQLite([]float64{}, []float64{})
	if result != 0 {
		t.Errorf("Empty vectors should return 0, got %v", result)
	}

	// Zero vectors
	result = cosineSimSQLite([]float64{0, 0}, []float64{0, 0})
	if result != 0 {
		t.Errorf("Zero vectors should return 0, got %v", result)
	}
}

func TestNewStore(t *testing.T) {
	tmpDir := t.TempDir()

	// Test memory store
	memStore, err := NewStore("memory", "", 0)
	if err != nil {
		t.Fatalf("NewStore(memory) error = %v", err)
	}
	memStore.Close()

	// Test SQLite store
	sqlStore, err := NewStore("sqlite", filepath.Join(tmpDir, "test.db"), 384)
	if err != nil {
		t.Fatalf("NewStore(sqlite) error = %v", err)
	}
	sqlStore.Close()

	// Test invalid type
	_, err = NewStore("invalid", "", 0)
	if err == nil {
		t.Error("NewStore(invalid) should return error")
	}
}

func TestSQLiteStore_Upsert(t *testing.T) {
	store := createTestSQLiteStore(t)
	defer store.Close()
	ctx := context.Background()

	// Insert initial document
	store.Insert(ctx, &Document{
		ID:      "doc1",
		Content: "Original",
	})

	// Upsert (should replace)
	store.Insert(ctx, &Document{
		ID:      "doc1",
		Content: "Updated",
	})

	doc, _ := store.Get(ctx, "doc1")
	if doc.Content != "Updated" {
		t.Errorf("Content = %v, want 'Updated'", doc.Content)
	}

	// Count should still be 1
	count, _ := store.Count(ctx, "default")
	if count != 1 {
		t.Errorf("Count = %v, want 1", count)
	}
}

func TestSQLiteStore_Persistence(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "persist.db")
	ctx := context.Background()

	// Create and populate store
	store1, _ := NewSQLiteStore(SQLiteConfig{Path: dbPath})
	store1.Insert(ctx, &Document{ID: "doc1", Content: "Persistent"})
	store1.Close()

	// Reopen and verify data
	store2, _ := NewSQLiteStore(SQLiteConfig{Path: dbPath})
	defer store2.Close()

	doc, err := store2.Get(ctx, "doc1")
	if err != nil {
		t.Fatalf("Data should persist: %v", err)
	}
	if doc.Content != "Persistent" {
		t.Errorf("Content = %v, want 'Persistent'", doc.Content)
	}
}

// Helper function
func createTestSQLiteStore(t *testing.T) *SQLiteStore {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	store, err := NewSQLiteStore(SQLiteConfig{
		Path:       dbPath,
		Dimensions: 384,
	})
	if err != nil {
		t.Fatalf("Failed to create test store: %v", err)
	}

	return store
}

func BenchmarkSQLiteStore_Insert(b *testing.B) {
	tmpDir := os.TempDir()
	dbPath := filepath.Join(tmpDir, "bench_insert.db")
	defer os.Remove(dbPath)

	store, _ := NewSQLiteStore(SQLiteConfig{Path: dbPath, Dimensions: 768})
	defer store.Close()

	ctx := context.Background()
	embedding := make([]float64, 768)
	for i := range embedding {
		embedding[i] = float64(i) / 768.0
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		store.Insert(ctx, &Document{
			ID:        "doc" + string(rune(i)),
			Content:   "Benchmark content",
			Embedding: embedding,
		})
	}
}

func BenchmarkSQLiteStore_Search(b *testing.B) {
	tmpDir := os.TempDir()
	dbPath := filepath.Join(tmpDir, "bench_search.db")
	defer os.Remove(dbPath)

	store, _ := NewSQLiteStore(SQLiteConfig{Path: dbPath, Dimensions: 768})
	defer store.Close()

	ctx := context.Background()
	embedding := make([]float64, 768)
	for i := range embedding {
		embedding[i] = float64(i) / 768.0
	}

	// Insert 100 documents
	for i := 0; i < 100; i++ {
		store.Insert(ctx, &Document{
			ID:        "doc" + string(rune(i)),
			Content:   "Benchmark content",
			Embedding: embedding,
		})
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		store.Search(ctx, embedding, "default", 5, 0.5)
	}
}
