package vectorstore

import (
	"context"
	"testing"
)

func TestDocument_Fields(t *testing.T) {
	doc := &Document{
		ID:         "doc1",
		Content:    "Test content",
		Embedding:  []float64{0.1, 0.2, 0.3},
		Metadata:   map[string]string{"key": "value"},
		Collection: "test",
	}

	if doc.ID != "doc1" {
		t.Errorf("ID = %v, want doc1", doc.ID)
	}
	if doc.Content != "Test content" {
		t.Errorf("Content = %v, want 'Test content'", doc.Content)
	}
	if len(doc.Embedding) != 3 {
		t.Errorf("Embedding length = %d, want 3", len(doc.Embedding))
	}
	if doc.Metadata["key"] != "value" {
		t.Errorf("Metadata[key] = %v, want value", doc.Metadata["key"])
	}
	if doc.Collection != "test" {
		t.Errorf("Collection = %v, want test", doc.Collection)
	}
}

func TestSearchResult_Fields(t *testing.T) {
	doc := &Document{ID: "doc1"}
	result := SearchResult{
		Document: doc,
		Score:    0.95,
	}

	if result.Document.ID != "doc1" {
		t.Errorf("Document.ID = %v, want doc1", result.Document.ID)
	}
	if result.Score != 0.95 {
		t.Errorf("Score = %v, want 0.95", result.Score)
	}
}

func TestMemoryStore_Insert(t *testing.T) {
	store := NewMemoryStore()
	ctx := context.Background()

	doc := &Document{
		ID:         "doc1",
		Content:    "Test content",
		Collection: "test",
		Embedding:  []float64{0.1, 0.2, 0.3},
	}

	err := store.Insert(ctx, doc)
	if err != nil {
		t.Fatalf("Insert() error = %v", err)
	}

	retrieved, err := store.Get(ctx, "doc1")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if retrieved.Content != "Test content" {
		t.Errorf("Content = %v, want 'Test content'", retrieved.Content)
	}
}

func TestMemoryStore_Insert_EmptyID(t *testing.T) {
	store := NewMemoryStore()
	ctx := context.Background()

	err := store.Insert(ctx, &Document{Content: "No ID"})
	if err == nil {
		t.Error("Insert() should return error for empty ID")
	}
}

func TestMemoryStore_Insert_DefaultCollection(t *testing.T) {
	store := NewMemoryStore()
	ctx := context.Background()

	store.Insert(ctx, &Document{ID: "doc1", Content: "Test"})

	retrieved, _ := store.Get(ctx, "doc1")
	if retrieved.Collection != "default" {
		t.Errorf("Collection = %v, want 'default'", retrieved.Collection)
	}
}

func TestMemoryStore_Insert_Multiple(t *testing.T) {
	store := NewMemoryStore()
	ctx := context.Background()

	docs := []*Document{
		{ID: "doc1", Content: "First"},
		{ID: "doc2", Content: "Second"},
		{ID: "doc3", Content: "Third"},
	}

	err := store.Insert(ctx, docs...)
	if err != nil {
		t.Fatalf("Insert() error = %v", err)
	}

	count, _ := store.Count(ctx, "")
	if count != 3 {
		t.Errorf("Count = %v, want 3", count)
	}
}

func TestMemoryStore_Search(t *testing.T) {
	store := NewMemoryStore()
	ctx := context.Background()

	docs := []*Document{
		{ID: "doc1", Content: "First", Collection: "test", Embedding: []float64{1.0, 0.0, 0.0}},
		{ID: "doc2", Content: "Second", Collection: "test", Embedding: []float64{0.9, 0.1, 0.0}},
		{ID: "doc3", Content: "Third", Collection: "test", Embedding: []float64{0.0, 1.0, 0.0}},
	}
	store.Insert(ctx, docs...)

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

func TestMemoryStore_Search_EmptyCollection(t *testing.T) {
	store := NewMemoryStore()
	ctx := context.Background()

	results, err := store.Search(ctx, []float64{1.0, 0.0}, "nonexistent", 5, 0.0)
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}

	if len(results) != 0 {
		t.Errorf("Results count = %v, want 0", len(results))
	}
}

func TestMemoryStore_Search_DefaultCollection(t *testing.T) {
	store := NewMemoryStore()
	ctx := context.Background()

	store.Insert(ctx, &Document{
		ID:        "doc1",
		Content:   "Test",
		Embedding: []float64{1.0, 0.0},
	})

	results, _ := store.Search(ctx, []float64{1.0, 0.0}, "", 5, 0.0)
	if len(results) != 1 {
		t.Errorf("Results count = %v, want 1", len(results))
	}
}

func TestMemoryStore_Get_NotFound(t *testing.T) {
	store := NewMemoryStore()
	ctx := context.Background()

	_, err := store.Get(ctx, "nonexistent")
	if err == nil {
		t.Error("Get() should return error for nonexistent document")
	}
}

func TestMemoryStore_Delete(t *testing.T) {
	store := NewMemoryStore()
	ctx := context.Background()

	store.Insert(ctx, &Document{ID: "doc1", Content: "Test"})
	store.Delete(ctx, "doc1")

	_, err := store.Get(ctx, "doc1")
	if err == nil {
		t.Error("Document should be deleted")
	}
}

func TestMemoryStore_Delete_NonExistent(t *testing.T) {
	store := NewMemoryStore()
	ctx := context.Background()

	err := store.Delete(ctx, "nonexistent")
	if err != nil {
		t.Errorf("Delete() should not error for nonexistent: %v", err)
	}
}

func TestMemoryStore_ListCollections(t *testing.T) {
	store := NewMemoryStore()
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

func TestMemoryStore_DeleteCollection(t *testing.T) {
	store := NewMemoryStore()
	ctx := context.Background()

	store.Insert(ctx,
		&Document{ID: "doc1", Content: "A", Collection: "test"},
		&Document{ID: "doc2", Content: "B", Collection: "test"},
		&Document{ID: "doc3", Content: "C", Collection: "other"},
	)

	store.DeleteCollection(ctx, "test")

	count, _ := store.Count(ctx, "test")
	if count != 0 {
		t.Errorf("Count after delete = %v, want 0", count)
	}

	count, _ = store.Count(ctx, "other")
	if count != 1 {
		t.Errorf("Other collection count = %v, want 1", count)
	}
}

func TestMemoryStore_Count(t *testing.T) {
	store := NewMemoryStore()
	ctx := context.Background()

	store.Insert(ctx,
		&Document{ID: "doc1", Content: "A", Collection: "test"},
		&Document{ID: "doc2", Content: "B", Collection: "test"},
		&Document{ID: "doc3", Content: "C", Collection: "other"},
	)

	count, _ := store.Count(ctx, "test")
	if count != 2 {
		t.Errorf("Count = %v, want 2", count)
	}

	totalCount, _ := store.Count(ctx, "")
	if totalCount != 3 {
		t.Errorf("Total count = %v, want 3", totalCount)
	}
}

func TestMemoryStore_Close(t *testing.T) {
	store := NewMemoryStore()
	err := store.Close()
	if err != nil {
		t.Errorf("Close() error = %v", err)
	}
}

func TestCosineSimilarity(t *testing.T) {
	tests := []struct {
		name     string
		a        []float64
		b        []float64
		expected float64
	}{
		{"identical", []float64{1, 0, 0}, []float64{1, 0, 0}, 1.0},
		{"orthogonal", []float64{1, 0, 0}, []float64{0, 1, 0}, 0.0},
		{"opposite", []float64{1, 0, 0}, []float64{-1, 0, 0}, -1.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := cosineSimilarity(tt.a, tt.b)
			diff := result - tt.expected
			if diff > 0.001 || diff < -0.001 {
				t.Errorf("cosineSimilarity() = %v, want ~%v", result, tt.expected)
			}
		})
	}
}

func TestCosineSimilarity_EdgeCases(t *testing.T) {
	// Different lengths
	if cosineSimilarity([]float64{1, 2}, []float64{1, 2, 3}) != 0 {
		t.Error("Different lengths should return 0")
	}

	// Zero vectors
	if cosineSimilarity([]float64{0, 0}, []float64{0, 0}) != 0 {
		t.Error("Zero vectors should return 0")
	}
}

func TestVectorNorm(t *testing.T) {
	tests := []struct {
		input    []float64
		expected float64
	}{
		{[]float64{0, 0}, 0},
		{[]float64{1, 0}, 1},
		{[]float64{3, 4}, 5},
		{[]float64{1, 1, 1, 1}, 2},
	}

	for _, tt := range tests {
		result := vectorNorm(tt.input)
		diff := result - tt.expected
		if diff > 0.001 || diff < -0.001 {
			t.Errorf("vectorNorm(%v) = %v, want ~%v", tt.input, result, tt.expected)
		}
	}
}

func TestMemoryStore_NoDuplicateInCollection(t *testing.T) {
	store := NewMemoryStore()
	ctx := context.Background()

	// Insert same document twice
	store.Insert(ctx, &Document{ID: "doc1", Content: "First", Collection: "test"})
	store.Insert(ctx, &Document{ID: "doc1", Content: "Updated", Collection: "test"})

	// Count should still be 1
	count, _ := store.Count(ctx, "test")
	if count != 1 {
		t.Errorf("Count = %v, want 1 (no duplicates)", count)
	}

	// Content should be updated
	doc, _ := store.Get(ctx, "doc1")
	if doc.Content != "Updated" {
		t.Errorf("Content = %v, want 'Updated'", doc.Content)
	}
}

func BenchmarkMemoryStore_Insert(b *testing.B) {
	store := NewMemoryStore()
	ctx := context.Background()
	embedding := make([]float64, 768)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		store.Insert(ctx, &Document{
			ID:        "doc" + string(rune(i)),
			Content:   "Benchmark",
			Embedding: embedding,
		})
	}
}

func BenchmarkMemoryStore_Search(b *testing.B) {
	store := NewMemoryStore()
	ctx := context.Background()
	embedding := make([]float64, 768)

	// Insert 100 documents
	for i := 0; i < 100; i++ {
		store.Insert(ctx, &Document{
			ID:        "doc" + string(rune(i)),
			Content:   "Benchmark",
			Embedding: embedding,
		})
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		store.Search(ctx, embedding, "default", 5, 0.5)
	}
}
