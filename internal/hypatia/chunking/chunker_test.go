package chunking

import (
	"strings"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.Strategy != StrategyRecursive {
		t.Errorf("Strategy = %v, want recursive", cfg.Strategy)
	}
	if cfg.ChunkSize != 1000 {
		t.Errorf("ChunkSize = %v, want 1000", cfg.ChunkSize)
	}
	if cfg.ChunkOverlap != 200 {
		t.Errorf("ChunkOverlap = %v, want 200", cfg.ChunkOverlap)
	}
	if len(cfg.Separators) != 4 {
		t.Errorf("Separators count = %v, want 4", len(cfg.Separators))
	}
}

func TestChunker_SplitFixed(t *testing.T) {
	cfg := Config{
		Strategy:     StrategyFixed,
		ChunkSize:    10,
		ChunkOverlap: 2,
	}
	chunker := NewChunker(cfg)

	text := "0123456789ABCDEFGHIJ"
	chunks := chunker.Split(text, "doc1")

	if len(chunks) < 2 {
		t.Fatalf("Expected at least 2 chunks, got %d", len(chunks))
	}

	// First chunk should be 10 chars
	if len(chunks[0].Content) != 10 {
		t.Errorf("First chunk length = %d, want 10", len(chunks[0].Content))
	}

	// Check chunk IDs
	if chunks[0].ID != "doc1_chunk_0" {
		t.Errorf("First chunk ID = %v, want doc1_chunk_0", chunks[0].ID)
	}
	if chunks[1].ID != "doc1_chunk_1" {
		t.Errorf("Second chunk ID = %v, want doc1_chunk_1", chunks[1].ID)
	}

	// Check indices
	if chunks[0].Index != 0 {
		t.Errorf("First chunk Index = %d, want 0", chunks[0].Index)
	}
	if chunks[1].Index != 1 {
		t.Errorf("Second chunk Index = %d, want 1", chunks[1].Index)
	}
}

func TestChunker_SplitSentence(t *testing.T) {
	cfg := Config{
		Strategy:     StrategySentence,
		ChunkSize:    100,
		ChunkOverlap: 0,
	}
	chunker := NewChunker(cfg)

	text := "Dies ist Satz eins. Dies ist Satz zwei. Dies ist Satz drei."
	chunks := chunker.Split(text, "doc1")

	if len(chunks) == 0 {
		t.Fatal("Expected at least 1 chunk")
	}

	// All sentences should be in one chunk since they fit
	content := chunks[0].Content
	if !strings.Contains(content, "Satz eins") {
		t.Error("Chunk should contain 'Satz eins'")
	}
}

func TestChunker_SplitSentence_MultipleChunks(t *testing.T) {
	cfg := Config{
		Strategy:     StrategySentence,
		ChunkSize:    30,
		ChunkOverlap: 0,
	}
	chunker := NewChunker(cfg)

	text := "Dies ist Satz eins. Dies ist Satz zwei. Dies ist Satz drei."
	chunks := chunker.Split(text, "doc1")

	if len(chunks) < 2 {
		t.Errorf("Expected at least 2 chunks for small chunk size, got %d", len(chunks))
	}
}

func TestChunker_SplitParagraph(t *testing.T) {
	cfg := Config{
		Strategy:     StrategyParagraph,
		ChunkSize:    200,
		ChunkOverlap: 0,
	}
	chunker := NewChunker(cfg)

	text := `Erster Absatz mit Text.

Zweiter Absatz mit mehr Text.

Dritter Absatz.`

	chunks := chunker.Split(text, "doc1")

	if len(chunks) == 0 {
		t.Fatal("Expected at least 1 chunk")
	}

	// With large chunk size, all paragraphs should fit in one chunk
	content := chunks[0].Content
	if !strings.Contains(content, "Erster Absatz") {
		t.Error("Chunk should contain 'Erster Absatz'")
	}
}

func TestChunker_SplitRecursive(t *testing.T) {
	cfg := Config{
		Strategy:     StrategyRecursive,
		ChunkSize:    50,
		ChunkOverlap: 10,
		Separators:   []string{"\n\n", "\n", ". ", " "},
	}
	chunker := NewChunker(cfg)

	text := `Erster Absatz mit etwas längeren Text.

Zweiter Absatz der auch etwas länger ist.

Dritter Absatz.`

	chunks := chunker.Split(text, "doc1")

	if len(chunks) < 2 {
		t.Errorf("Expected at least 2 chunks, got %d", len(chunks))
	}

	// Each chunk should be <= chunk size
	for i, chunk := range chunks {
		if len(chunk.Content) > cfg.ChunkSize+cfg.ChunkOverlap {
			t.Errorf("Chunk %d too large: %d chars", i, len(chunk.Content))
		}
	}
}

func TestChunker_ShortText(t *testing.T) {
	cfg := Config{
		Strategy:     StrategyRecursive,
		ChunkSize:    1000,
		ChunkOverlap: 100,
		Separators:   []string{"\n\n", "\n", ". ", " "},
	}
	chunker := NewChunker(cfg)

	text := "Kurzer Text."
	chunks := chunker.Split(text, "doc1")

	if len(chunks) != 1 {
		t.Errorf("Expected 1 chunk for short text, got %d", len(chunks))
	}

	if chunks[0].Content != text {
		t.Errorf("Content = %v, want %v", chunks[0].Content, text)
	}
}

func TestChunker_EmptyText(t *testing.T) {
	cfg := DefaultConfig()
	chunker := NewChunker(cfg)

	chunks := chunker.Split("", "doc1")

	// Empty text should return one chunk with empty content
	if len(chunks) != 1 {
		t.Errorf("Expected 1 chunk for empty text, got %d", len(chunks))
	}
}

func TestChunker_DefaultStrategy(t *testing.T) {
	cfg := Config{
		Strategy:  "unknown",
		ChunkSize: 100,
	}
	chunker := NewChunker(cfg)

	text := "Test text"
	chunks := chunker.Split(text, "doc1")

	// Unknown strategy should fall back to fixed
	if len(chunks) != 1 {
		t.Errorf("Expected 1 chunk, got %d", len(chunks))
	}
}

func TestSplitIntoSentences(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		expected int
	}{
		{"single sentence", "Dies ist ein Satz.", 1},
		{"two sentences", "Satz eins. Satz zwei.", 2},
		{"question and exclamation", "Was ist das? Super!", 2},
		{"no punctuation", "Kein Satzzeichen", 1},
		{"multiple punctuation", "Wow! Wirklich? Ja.", 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sentences := splitIntoSentences(tt.text)
			if len(sentences) != tt.expected {
				t.Errorf("splitIntoSentences(%q) = %d sentences, want %d", tt.text, len(sentences), tt.expected)
			}
		})
	}
}

func TestGenerateChunkID(t *testing.T) {
	tests := []struct {
		docID    string
		index    int
		expected string
	}{
		{"doc1", 0, "doc1_chunk_0"},
		{"doc1", 5, "doc1_chunk_5"},
		{"my doc", 1, "my_doc_chunk_1"},
		{"doc", 10, "doc_chunk_10"},
	}

	for _, tt := range tests {
		t.Run(tt.docID, func(t *testing.T) {
			result := generateChunkID(tt.docID, tt.index)
			if result != tt.expected {
				t.Errorf("generateChunkID(%q, %d) = %q, want %q", tt.docID, tt.index, result, tt.expected)
			}
		})
	}
}

func TestItoa(t *testing.T) {
	tests := []struct {
		n        int
		expected string
	}{
		{0, "0"},
		{1, "1"},
		{10, "10"},
		{123, "123"},
		{-5, "-5"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := itoa(tt.n)
			if result != tt.expected {
				t.Errorf("itoa(%d) = %q, want %q", tt.n, result, tt.expected)
			}
		})
	}
}

func TestChunker_UnicodeText(t *testing.T) {
	cfg := Config{
		Strategy:     StrategyFixed,
		ChunkSize:    5,
		ChunkOverlap: 0,
	}
	chunker := NewChunker(cfg)

	// Test with German umlauts and special characters
	text := "äöüßÄÖÜ"
	chunks := chunker.Split(text, "doc1")

	if len(chunks) != 2 {
		t.Errorf("Expected 2 chunks for unicode text, got %d", len(chunks))
	}

	// Verify content is not corrupted
	totalContent := ""
	for _, chunk := range chunks {
		totalContent += chunk.Content
	}

	// With overlap=0, total content should match (but without overlap subtraction)
	if !strings.Contains(totalContent, "äöü") {
		t.Error("Unicode characters should be preserved")
	}
}

func TestChunk_Fields(t *testing.T) {
	chunk := Chunk{
		ID:       "doc1_chunk_0",
		Content:  "Test content",
		Index:    0,
		Start:    0,
		End:      12,
		Metadata: map[string]string{"source": "test"},
	}

	if chunk.ID != "doc1_chunk_0" {
		t.Errorf("ID = %v, want doc1_chunk_0", chunk.ID)
	}
	if chunk.Content != "Test content" {
		t.Errorf("Content = %v, want 'Test content'", chunk.Content)
	}
	if chunk.Index != 0 {
		t.Errorf("Index = %v, want 0", chunk.Index)
	}
	if chunk.Start != 0 {
		t.Errorf("Start = %v, want 0", chunk.Start)
	}
	if chunk.End != 12 {
		t.Errorf("End = %v, want 12", chunk.End)
	}
	if chunk.Metadata["source"] != "test" {
		t.Errorf("Metadata[source] = %v, want test", chunk.Metadata["source"])
	}
}

func TestStrategy_Constants(t *testing.T) {
	if StrategyFixed != "fixed" {
		t.Errorf("StrategyFixed = %v, want fixed", StrategyFixed)
	}
	if StrategySentence != "sentence" {
		t.Errorf("StrategySentence = %v, want sentence", StrategySentence)
	}
	if StrategyParagraph != "paragraph" {
		t.Errorf("StrategyParagraph = %v, want paragraph", StrategyParagraph)
	}
	if StrategyRecursive != "recursive" {
		t.Errorf("StrategyRecursive = %v, want recursive", StrategyRecursive)
	}
}

func BenchmarkChunker_SplitFixed(b *testing.B) {
	cfg := Config{
		Strategy:     StrategyFixed,
		ChunkSize:    512,
		ChunkOverlap: 64,
	}
	chunker := NewChunker(cfg)
	text := strings.Repeat("Lorem ipsum dolor sit amet. ", 100)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		chunker.Split(text, "benchmark")
	}
}

func BenchmarkChunker_SplitRecursive(b *testing.B) {
	cfg := Config{
		Strategy:     StrategyRecursive,
		ChunkSize:    512,
		ChunkOverlap: 64,
		Separators:   []string{"\n\n", "\n", ". ", " "},
	}
	chunker := NewChunker(cfg)
	text := strings.Repeat("Lorem ipsum dolor sit amet. ", 100)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		chunker.Split(text, "benchmark")
	}
}
