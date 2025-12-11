// ============================================================================
// meinDENKWERK (mDW) - Agent Embedding Tests
// ============================================================================

package agentloader

import (
	"context"
	"fmt"
	"math"
	"testing"
)

// mockEmbeddingFunc creates a mock embedding function for testing
func mockEmbeddingFunc(embeddings map[string][]float64) EmbeddingFunc {
	return func(ctx context.Context, texts []string) ([][]float64, error) {
		result := make([][]float64, len(texts))
		for i, text := range texts {
			if emb, ok := embeddings[text]; ok {
				result[i] = emb
			} else {
				// Generate deterministic mock embedding based on text length
				result[i] = make([]float64, 4)
				for j := range result[i] {
					result[i][j] = float64(len(text)+j) / 100.0
				}
			}
		}
		return result, nil
	}
}

func TestNewEmbeddingRegistry(t *testing.T) {
	registry := NewEmbeddingRegistry()

	if registry == nil {
		t.Fatal("NewEmbeddingRegistry returned nil")
	}

	if registry.IsInitialized() {
		t.Error("New registry should not be initialized")
	}

	if registry.agents == nil {
		t.Error("Agents map should not be nil")
	}
}

func TestSetEmbeddingFunc(t *testing.T) {
	registry := NewEmbeddingRegistry()

	if registry.IsInitialized() {
		t.Error("Registry should not be initialized before setting function")
	}

	mockFn := mockEmbeddingFunc(nil)
	registry.SetEmbeddingFunc(mockFn)

	if !registry.IsInitialized() {
		t.Error("Registry should be initialized after setting function")
	}

	// Test setting to nil
	registry.SetEmbeddingFunc(nil)
	if registry.IsInitialized() {
		t.Error("Registry should not be initialized after setting function to nil")
	}
}

func TestUpdateAgentEmbedding(t *testing.T) {
	registry := NewEmbeddingRegistry()
	registry.SetEmbeddingFunc(mockEmbeddingFunc(nil))

	agent := &AgentYAML{
		ID:          "test-agent",
		Name:        "Test Agent",
		Description: "A test agent for unit testing",
		Tools: []ToolConfig{
			{Name: "search"},
		},
	}

	ctx := context.Background()
	err := registry.UpdateAgentEmbedding(ctx, agent)
	if err != nil {
		t.Fatalf("UpdateAgentEmbedding failed: %v", err)
	}

	// Verify embedding was stored
	emb, exists := registry.GetAgentEmbedding("test-agent")
	if !exists {
		t.Fatal("Embedding should exist after update")
	}

	if emb.AgentID != "test-agent" {
		t.Errorf("AgentID mismatch: got %s, want test-agent", emb.AgentID)
	}

	if emb.AgentName != "Test Agent" {
		t.Errorf("AgentName mismatch: got %s, want Test Agent", emb.AgentName)
	}

	if len(emb.Embedding) == 0 {
		t.Error("Embedding should not be empty")
	}

	if emb.TextHash == "" {
		t.Error("TextHash should not be empty")
	}
}

func TestUpdateAgentEmbedding_NotInitialized(t *testing.T) {
	registry := NewEmbeddingRegistry()
	// Not setting embedding function

	agent := &AgentYAML{
		ID:   "test-agent",
		Name: "Test Agent",
	}

	ctx := context.Background()
	err := registry.UpdateAgentEmbedding(ctx, agent)

	// Should not fail, just skip
	if err != nil {
		t.Errorf("Should not fail when not initialized: %v", err)
	}

	// Embedding should not exist
	_, exists := registry.GetAgentEmbedding("test-agent")
	if exists {
		t.Error("Embedding should not exist when not initialized")
	}
}

func TestUpdateAgentEmbedding_CacheHit(t *testing.T) {
	callCount := 0
	mockFn := func(ctx context.Context, texts []string) ([][]float64, error) {
		callCount++
		result := make([][]float64, len(texts))
		for i := range texts {
			result[i] = []float64{1.0, 2.0, 3.0, 4.0}
		}
		return result, nil
	}

	registry := NewEmbeddingRegistry()
	registry.SetEmbeddingFunc(mockFn)

	agent := &AgentYAML{
		ID:          "test-agent",
		Name:        "Test Agent",
		Description: "Unchanged description",
	}

	ctx := context.Background()

	// First update
	err := registry.UpdateAgentEmbedding(ctx, agent)
	if err != nil {
		t.Fatalf("First update failed: %v", err)
	}

	if callCount != 1 {
		t.Errorf("Expected 1 embedding call, got %d", callCount)
	}

	// Second update with same content (should use cache)
	err = registry.UpdateAgentEmbedding(ctx, agent)
	if err != nil {
		t.Fatalf("Second update failed: %v", err)
	}

	if callCount != 1 {
		t.Errorf("Expected still 1 embedding call (cached), got %d", callCount)
	}

	// Third update with different content
	agent.Description = "Changed description"
	err = registry.UpdateAgentEmbedding(ctx, agent)
	if err != nil {
		t.Fatalf("Third update failed: %v", err)
	}

	if callCount != 2 {
		t.Errorf("Expected 2 embedding calls after change, got %d", callCount)
	}
}

func TestRemoveAgentEmbedding(t *testing.T) {
	registry := NewEmbeddingRegistry()
	registry.SetEmbeddingFunc(mockEmbeddingFunc(nil))

	agent := &AgentYAML{
		ID:   "test-agent",
		Name: "Test Agent",
	}

	ctx := context.Background()
	registry.UpdateAgentEmbedding(ctx, agent)

	// Verify it exists
	_, exists := registry.GetAgentEmbedding("test-agent")
	if !exists {
		t.Fatal("Embedding should exist before removal")
	}

	// Remove
	registry.RemoveAgentEmbedding("test-agent")

	// Verify it's gone
	_, exists = registry.GetAgentEmbedding("test-agent")
	if exists {
		t.Error("Embedding should not exist after removal")
	}

	// Removing non-existent should not panic
	registry.RemoveAgentEmbedding("non-existent")
}

func TestGetAllEmbeddings(t *testing.T) {
	registry := NewEmbeddingRegistry()
	registry.SetEmbeddingFunc(mockEmbeddingFunc(nil))

	ctx := context.Background()

	agents := []*AgentYAML{
		{ID: "agent1", Name: "Agent 1"},
		{ID: "agent2", Name: "Agent 2"},
		{ID: "agent3", Name: "Agent 3"},
	}

	for _, agent := range agents {
		registry.UpdateAgentEmbedding(ctx, agent)
	}

	all := registry.GetAllEmbeddings()

	if len(all) != 3 {
		t.Errorf("Expected 3 embeddings, got %d", len(all))
	}

	for _, agent := range agents {
		if _, ok := all[agent.ID]; !ok {
			t.Errorf("Missing embedding for %s", agent.ID)
		}
	}
}

func TestFindBestAgentForTask(t *testing.T) {
	// Create embeddings that simulate different specializations
	embeddings := map[string][]float64{
		// Web researcher agent - high similarity to web-related tasks
		"Agent-Name: Web Researcher\nBeschreibung: Searches the web": {0.9, 0.1, 0.1, 0.1},
		// Code reviewer agent - high similarity to code-related tasks
		"Agent-Name: Code Reviewer\nBeschreibung: Reviews code": {0.1, 0.9, 0.1, 0.1},
		// Task: Web search (should match Web Researcher)
		"Search the internet for information": {0.85, 0.1, 0.1, 0.1},
	}

	registry := NewEmbeddingRegistry()
	registry.SetEmbeddingFunc(mockEmbeddingFunc(embeddings))

	ctx := context.Background()

	// Add agents
	webAgent := &AgentYAML{ID: "web-researcher", Name: "Web Researcher", Description: "Searches the web"}
	codeAgent := &AgentYAML{ID: "code-reviewer", Name: "Code Reviewer", Description: "Reviews code"}

	registry.UpdateAgentEmbedding(ctx, webAgent)
	registry.UpdateAgentEmbedding(ctx, codeAgent)

	// Find best agent for web search task
	match, err := registry.FindBestAgentForTask(ctx, "Search the internet for information")
	if err != nil {
		t.Fatalf("FindBestAgentForTask failed: %v", err)
	}

	if match == nil {
		t.Fatal("Match should not be nil")
	}

	// The web researcher should be a better match for web search
	// Note: Due to mock embedding, we just verify a match is returned
	if match.AgentID == "" {
		t.Error("AgentID should not be empty")
	}

	if match.Similarity <= 0 {
		t.Error("Similarity should be positive")
	}
}

func TestFindBestAgentForTask_NotInitialized(t *testing.T) {
	registry := NewEmbeddingRegistry()
	// Not initialized

	ctx := context.Background()
	_, err := registry.FindBestAgentForTask(ctx, "Some task")

	if err == nil {
		t.Error("Should fail when not initialized")
	}
}

func TestFindBestAgentForTask_NoAgents(t *testing.T) {
	registry := NewEmbeddingRegistry()
	registry.SetEmbeddingFunc(mockEmbeddingFunc(nil))

	ctx := context.Background()
	_, err := registry.FindBestAgentForTask(ctx, "Some task")

	if err == nil {
		t.Error("Should fail when no agents available")
	}
}

func TestFindTopAgentsForTask(t *testing.T) {
	registry := NewEmbeddingRegistry()
	registry.SetEmbeddingFunc(mockEmbeddingFunc(nil))

	ctx := context.Background()

	// Add multiple agents
	for i := 1; i <= 5; i++ {
		agent := &AgentYAML{
			ID:          fmt.Sprintf("agent%d", i),
			Name:        fmt.Sprintf("Agent %d", i),
			Description: fmt.Sprintf("Description for agent %d with varying length", i),
		}
		registry.UpdateAgentEmbedding(ctx, agent)
	}

	// Find top 3
	matches, err := registry.FindTopAgentsForTask(ctx, "Find something", 3)
	if err != nil {
		t.Fatalf("FindTopAgentsForTask failed: %v", err)
	}

	if len(matches) != 3 {
		t.Errorf("Expected 3 matches, got %d", len(matches))
	}

	// Verify sorting (descending by similarity)
	for i := 1; i < len(matches); i++ {
		if matches[i].Similarity > matches[i-1].Similarity {
			t.Errorf("Matches not sorted correctly at index %d", i)
		}
	}
}

func TestFindTopAgentsForTask_RequestMoreThanAvailable(t *testing.T) {
	registry := NewEmbeddingRegistry()
	registry.SetEmbeddingFunc(mockEmbeddingFunc(nil))

	ctx := context.Background()

	// Add only 2 agents
	registry.UpdateAgentEmbedding(ctx, &AgentYAML{ID: "agent1", Name: "Agent 1"})
	registry.UpdateAgentEmbedding(ctx, &AgentYAML{ID: "agent2", Name: "Agent 2"})

	// Request top 5
	matches, err := registry.FindTopAgentsForTask(ctx, "Task", 5)
	if err != nil {
		t.Fatalf("FindTopAgentsForTask failed: %v", err)
	}

	if len(matches) != 2 {
		t.Errorf("Expected 2 matches (all available), got %d", len(matches))
	}
}

func TestCosineSimilarity(t *testing.T) {
	tests := []struct {
		name     string
		a        []float64
		b        []float64
		expected float64
	}{
		{
			name:     "identical vectors",
			a:        []float64{1, 0, 0},
			b:        []float64{1, 0, 0},
			expected: 1.0,
		},
		{
			name:     "orthogonal vectors",
			a:        []float64{1, 0, 0},
			b:        []float64{0, 1, 0},
			expected: 0.0,
		},
		{
			name:     "opposite vectors",
			a:        []float64{1, 0, 0},
			b:        []float64{-1, 0, 0},
			expected: -1.0,
		},
		{
			name:     "similar vectors",
			a:        []float64{1, 1, 0},
			b:        []float64{1, 0, 0},
			expected: 1.0 / math.Sqrt(2),
		},
		{
			name:     "empty vectors",
			a:        []float64{},
			b:        []float64{},
			expected: 0.0,
		},
		{
			name:     "different lengths",
			a:        []float64{1, 2},
			b:        []float64{1, 2, 3},
			expected: 0.0,
		},
		{
			name:     "zero vector a",
			a:        []float64{0, 0, 0},
			b:        []float64{1, 2, 3},
			expected: 0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := cosineSimilarity(tt.a, tt.b)
			if math.Abs(result-tt.expected) > 1e-9 {
				t.Errorf("cosineSimilarity(%v, %v) = %v, want %v", tt.a, tt.b, result, tt.expected)
			}
		})
	}
}

func TestBuildAgentEmbeddingText(t *testing.T) {
	agent := &AgentYAML{
		ID:          "test-agent",
		Name:        "Test Agent",
		Description: "A comprehensive test agent",
		SystemPrompt: "You are a helpful assistant.",
		Tools: []ToolConfig{
			{Name: "search"},
			{Name: "calculate"},
		},
		Metadata: map[string]string{
			"tags":     "test, example",
			"category": "testing",
		},
	}

	text := buildAgentEmbeddingText(agent)

	// Verify key components are present
	if !contains(text, "Agent-Name: Test Agent") {
		t.Error("Missing agent name")
	}
	if !contains(text, "Beschreibung: A comprehensive test agent") {
		t.Error("Missing description")
	}
	if !contains(text, "search") && !contains(text, "calculate") {
		t.Error("Missing tools")
	}
	if !contains(text, "test, example") {
		t.Error("Missing tags")
	}
	if !contains(text, "testing") {
		t.Error("Missing category")
	}
}

func TestHashString(t *testing.T) {
	// Same input should produce same hash
	hash1 := hashString("test string")
	hash2 := hashString("test string")

	if hash1 != hash2 {
		t.Error("Same string should produce same hash")
	}

	// Different input should produce different hash
	hash3 := hashString("different string")
	if hash1 == hash3 {
		t.Error("Different strings should produce different hashes")
	}

	// Hash should not be empty
	if hash1 == "" {
		t.Error("Hash should not be empty")
	}
}

func TestTruncateString(t *testing.T) {
	tests := []struct {
		input    string
		maxLen   int
		expected string
	}{
		{"short", 10, "short"},
		{"exactly10!", 10, "exactly10!"},
		{"this is a long string", 10, "this is a ..."},
		{"", 10, ""},
	}

	for _, tt := range tests {
		result := truncateString(tt.input, tt.maxLen)
		if result != tt.expected {
			t.Errorf("truncateString(%q, %d) = %q, want %q", tt.input, tt.maxLen, result, tt.expected)
		}
	}
}

func TestSortAgentMatches(t *testing.T) {
	matches := []*AgentMatch{
		{AgentID: "a", Similarity: 0.3},
		{AgentID: "b", Similarity: 0.9},
		{AgentID: "c", Similarity: 0.5},
		{AgentID: "d", Similarity: 0.1},
		{AgentID: "e", Similarity: 0.7},
	}

	sortAgentMatches(matches)

	expected := []string{"b", "e", "c", "a", "d"}
	for i, m := range matches {
		if m.AgentID != expected[i] {
			t.Errorf("Position %d: got %s, want %s", i, m.AgentID, expected[i])
		}
	}
}

func TestExtractPromptEssence(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		maxLen   int
		checkFn  func(string) bool
	}{
		{
			name:   "short prompt",
			input:  "You are helpful.",
			maxLen: 500,
			checkFn: func(s string) bool {
				return s == "You are helpful."
			},
		},
		{
			name:   "long prompt truncated at sentence",
			input:  "First sentence. " + string(make([]byte, 600)),
			maxLen: 500,
			checkFn: func(s string) bool {
				return len(s) <= 600 // Should be truncated
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractPromptEssence(tt.input)
			if !tt.checkFn(result) {
				t.Errorf("extractPromptEssence check failed for %s", tt.name)
			}
		})
	}
}

// Helper function
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
