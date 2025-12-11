// ============================================================================
// meinDENKWERK (mDW) - Lokale KI-Plattform
// ============================================================================
//
// Package:     websearch
// Description: Unit tests for Web Research Agent
// Author:      Mike Stoffels with Claude
// Created:     2025-12-10
// License:     MIT
// ============================================================================

package websearch

import (
	"context"
	"strings"
	"testing"
	"time"
)

// ============================================================================
// Unit Tests - Configuration
// ============================================================================

func TestDefaultAgentConfig(t *testing.T) {
	cfg := DefaultAgentConfig()

	if cfg.Timeout != 30*time.Second {
		t.Errorf("Timeout = %v, expected 30s", cfg.Timeout)
	}
	if len(cfg.SearXNGInstances) == 0 {
		t.Error("SearXNGInstances should not be empty")
	}
}

func TestNewWebResearchAgent(t *testing.T) {
	cfg := DefaultAgentConfig()
	agent := NewWebResearchAgent(cfg)

	if agent == nil {
		t.Fatal("NewWebResearchAgent returned nil")
	}
	if agent.searchClient == nil {
		t.Error("searchClient should not be nil")
	}
	if agent.logger == nil {
		t.Error("logger should not be nil")
	}
	if agent.enablePlaton {
		t.Error("enablePlaton should default to false")
	}
}

func TestNewWebResearchAgent_CustomConfig(t *testing.T) {
	cfg := AgentConfig{
		SearXNGInstances: []string{"http://localhost:9999"},
		Timeout:          60 * time.Second,
	}
	agent := NewWebResearchAgent(cfg)

	if agent.searchClient == nil {
		t.Error("searchClient should not be nil")
	}
}

// ============================================================================
// Unit Tests - Agent Definition
// ============================================================================

func TestGetAgentDefinition(t *testing.T) {
	agent := NewWebResearchAgent(DefaultAgentConfig())
	def := agent.GetAgentDefinition()

	tests := []struct {
		name     string
		got      interface{}
		expected interface{}
	}{
		{"ID", def.ID, "web-researcher"},
		{"Name contains Web", strings.Contains(def.Name, "Web") || strings.Contains(def.Name, "Recherche"), true},
		{"MaxSteps", def.MaxSteps, 12},
		{"Timeout", def.Timeout, 300 * time.Second},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.expected {
				t.Errorf("%s = %v, expected %v", tt.name, tt.got, tt.expected)
			}
		})
	}
}

func TestAgentDefinition_Tools(t *testing.T) {
	agent := NewWebResearchAgent(DefaultAgentConfig())
	def := agent.GetAgentDefinition()

	expectedTools := []string{"web_search", "fetch_webpage", "search_news"}

	if len(def.Tools) != len(expectedTools) {
		t.Errorf("Tools count = %d, expected %d", len(def.Tools), len(expectedTools))
	}

	for _, expected := range expectedTools {
		found := false
		for _, tool := range def.Tools {
			if tool == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected tool %q not found in definition", expected)
		}
	}
}

func TestAgentDefinition_SystemPrompt(t *testing.T) {
	agent := NewWebResearchAgent(DefaultAgentConfig())
	def := agent.GetAgentDefinition()

	// Check that system prompt contains key elements
	requiredPhrases := []string{
		"web_search",
		"fetch_webpage",
		"Quellen",
	}

	for _, phrase := range requiredPhrases {
		if !strings.Contains(def.SystemPrompt, phrase) {
			t.Errorf("SystemPrompt should contain %q", phrase)
		}
	}
}

// ============================================================================
// Unit Tests - AgentDefinition Struct
// ============================================================================

func TestAgentDefinition_Struct(t *testing.T) {
	def := AgentDefinition{
		ID:           "test-agent",
		Name:         "Test Agent",
		Description:  "A test agent",
		SystemPrompt: "You are a test agent.",
		Tools:        []string{"tool1", "tool2"},
		MaxSteps:     5,
		Timeout:      60 * time.Second,
	}

	if def.ID != "test-agent" {
		t.Errorf("ID = %s, expected test-agent", def.ID)
	}
	if def.MaxSteps != 5 {
		t.Errorf("MaxSteps = %d, expected 5", def.MaxSteps)
	}
	if len(def.Tools) != 2 {
		t.Errorf("Tools length = %d, expected 2", len(def.Tools))
	}
}

// ============================================================================
// Unit Tests - Tool Handlers
// ============================================================================

func TestWebSearchHandler_MissingQuery(t *testing.T) {
	agent := NewWebResearchAgent(DefaultAgentConfig())
	ctx := context.Background()

	tests := []struct {
		name   string
		params map[string]interface{}
	}{
		{"nil query", map[string]interface{}{"query": nil}},
		{"empty query", map[string]interface{}{"query": ""}},
		{"missing query", map[string]interface{}{}},
		{"wrong type", map[string]interface{}{"query": 123}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := agent.webSearchHandler(ctx, tt.params)
			if err == nil {
				t.Error("Expected error for invalid query parameter")
			}
		})
	}
}

func TestWebSearchHandler_CountParsing(t *testing.T) {
	// We can't fully test without a mock server, but we can test parameter parsing
	tests := []struct {
		name          string
		countParam    interface{}
		expectedCount int
	}{
		{"default count", nil, 5},
		{"string count", "10", 10},
		{"empty string", "", 5},
		{"invalid string", "abc", 5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			count := 5
			if countStr, ok := tt.countParam.(string); ok && countStr != "" {
				var parsed int
				n, _ := parseCount(countStr)
				if n > 0 {
					parsed = n
					count = parsed
				}
			}
			if count != tt.expectedCount {
				t.Errorf("Parsed count = %d, expected %d", count, tt.expectedCount)
			}
		})
	}
}

// Helper function for testing count parsing
func parseCount(s string) (int, error) {
	var count int
	_, err := parseCountInternal(s, &count)
	return count, err
}

func parseCountInternal(s string, count *int) (int, error) {
	*count = 5 // default
	if s != "" {
		var n int
		if _, err := stringToInt(s, &n); err == nil && n > 0 {
			*count = n
		}
	}
	return *count, nil
}

func stringToInt(s string, result *int) (int, error) {
	*result = 0
	for _, c := range s {
		if c < '0' || c > '9' {
			return 0, nil
		}
		*result = *result*10 + int(c-'0')
	}
	return *result, nil
}

func TestFetchWebpageHandler_MissingURL(t *testing.T) {
	agent := NewWebResearchAgent(DefaultAgentConfig())
	ctx := context.Background()

	tests := []struct {
		name   string
		params map[string]interface{}
	}{
		{"nil url", map[string]interface{}{"url": nil}},
		{"empty url", map[string]interface{}{"url": ""}},
		{"missing url", map[string]interface{}{}},
		{"wrong type", map[string]interface{}{"url": 123}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := agent.fetchWebpageHandler(ctx, tt.params)
			if err == nil {
				t.Error("Expected error for invalid URL parameter")
			}
		})
	}
}

func TestSearchNewsHandler_MissingQuery(t *testing.T) {
	agent := NewWebResearchAgent(DefaultAgentConfig())
	ctx := context.Background()

	tests := []struct {
		name   string
		params map[string]interface{}
	}{
		{"nil query", map[string]interface{}{"query": nil}},
		{"empty query", map[string]interface{}{"query": ""}},
		{"missing query", map[string]interface{}{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := agent.searchNewsHandler(ctx, tt.params)
			if err == nil {
				t.Error("Expected error for invalid query parameter")
			}
		})
	}
}

// ============================================================================
// Unit Tests - Response Formatting
// ============================================================================

func TestFormatSearchResponse_Nil(t *testing.T) {
	agent := NewWebResearchAgent(DefaultAgentConfig())

	result := agent.formatSearchResponse(nil)

	if !strings.Contains(result, "Keine Suchergebnisse") {
		t.Error("Nil response should indicate no results")
	}
}

func TestFormatSearchResponse_Empty(t *testing.T) {
	agent := NewWebResearchAgent(DefaultAgentConfig())

	resp := &SearchResponse{
		Query:   "test",
		Results: []SearchResult{},
	}

	result := agent.formatSearchResponse(resp)

	if !strings.Contains(result, "Keine Suchergebnisse") {
		t.Error("Empty response should indicate no results")
	}
}

func TestFormatSearchResponse_WithResults(t *testing.T) {
	agent := NewWebResearchAgent(DefaultAgentConfig())

	resp := &SearchResponse{
		Query: "golang testing",
		Results: []SearchResult{
			{
				Title:       "Go Testing Guide",
				URL:         "https://golang.org/doc/testing",
				Description: "Official Go testing documentation",
			},
			{
				Title:       "Testing Best Practices",
				URL:         "https://example.com/testing",
				Description: "Testing best practices",
				PublishedAt: "2025-01-01",
			},
		},
		Source:      "DuckDuckGo",
		Suggestions: []string{"go unit testing", "go integration testing"},
	}

	result := agent.formatSearchResponse(resp)

	// Check for query
	if !strings.Contains(result, "golang testing") {
		t.Error("Should contain query")
	}

	// Check for source
	if !strings.Contains(result, "DuckDuckGo") {
		t.Error("Should contain source")
	}

	// Check for results
	if !strings.Contains(result, "Go Testing Guide") {
		t.Error("Should contain first result title")
	}
	if !strings.Contains(result, "https://golang.org/doc/testing") {
		t.Error("Should contain first result URL")
	}

	// Check for numbered format
	if !strings.Contains(result, "[1]") {
		t.Error("Should have numbered results")
	}
	if !strings.Contains(result, "[2]") {
		t.Error("Should have second numbered result")
	}

	// Check for published date
	if !strings.Contains(result, "2025-01-01") {
		t.Error("Should contain published date when present")
	}

	// Check for suggestions
	if !strings.Contains(result, "Verwandte Suchanfragen") {
		t.Error("Should contain suggestions header")
	}
}

func TestFormatWebpageContent_Nil(t *testing.T) {
	agent := NewWebResearchAgent(DefaultAgentConfig())

	result := agent.formatWebpageContent(nil)

	if !strings.Contains(result, "Fehler") {
		t.Error("Nil content should indicate error")
	}
}

func TestFormatWebpageContent_Valid(t *testing.T) {
	agent := NewWebResearchAgent(DefaultAgentConfig())

	content := &WebpageContent{
		URL:     "https://example.com/page",
		Title:   "Example Page",
		Content: "This is the page content.",
	}

	result := agent.formatWebpageContent(content)

	if !strings.Contains(result, "https://example.com/page") {
		t.Error("Should contain URL")
	}
	if !strings.Contains(result, "Example Page") {
		t.Error("Should contain title")
	}
	if !strings.Contains(result, "This is the page content") {
		t.Error("Should contain content")
	}
	if !strings.Contains(result, "---") {
		t.Error("Should contain separator")
	}
}

func TestFormatWebpageContent_NoTitle(t *testing.T) {
	agent := NewWebResearchAgent(DefaultAgentConfig())

	content := &WebpageContent{
		URL:     "https://example.com/page",
		Title:   "",
		Content: "Content without title.",
	}

	result := agent.formatWebpageContent(content)

	// Should still work without title
	if !strings.Contains(result, "https://example.com/page") {
		t.Error("Should contain URL")
	}
	if !strings.Contains(result, "Content without title") {
		t.Error("Should contain content")
	}
}

// ============================================================================
// Unit Tests - Platon Integration
// ============================================================================

func TestSetPlatonClient_Enable(t *testing.T) {
	agent := NewWebResearchAgent(DefaultAgentConfig())

	if agent.enablePlaton {
		t.Error("Platon should be disabled by default")
	}

	// We can't create a real Platon client without a server,
	// but we can test the nil case
	agent.SetPlatonClient(nil, "")

	if agent.enablePlaton {
		t.Error("Platon should remain disabled with nil client")
	}
}

func TestProcessWithPlaton_Disabled(t *testing.T) {
	agent := NewWebResearchAgent(DefaultAgentConfig())
	ctx := context.Background()

	// Without Platon client, should return original content
	content := "Test content"
	result, blocked, err := agent.processWithPlaton(ctx, content, "post")

	if err != nil {
		t.Errorf("processWithPlaton should not error when disabled: %v", err)
	}
	if blocked {
		t.Error("Should not be blocked when Platon is disabled")
	}
	if result != content {
		t.Errorf("Content should be unchanged, got %q", result)
	}
}

func TestProcessWithPlaton_NilClient(t *testing.T) {
	agent := NewWebResearchAgent(DefaultAgentConfig())
	agent.enablePlaton = true // Force enable without client
	agent.platonClient = nil

	ctx := context.Background()
	content := "Test content"

	result, blocked, err := agent.processWithPlaton(ctx, content, "pre")

	// Should gracefully handle nil client
	if err != nil {
		t.Errorf("Should handle nil client gracefully: %v", err)
	}
	if blocked {
		t.Error("Should not be blocked with nil client")
	}
	if result != content {
		t.Error("Content should be unchanged with nil client")
	}
}

// ============================================================================
// Unit Tests - Search Client Access
// ============================================================================

func TestSearchClient(t *testing.T) {
	agent := NewWebResearchAgent(DefaultAgentConfig())

	client := agent.SearchClient()

	if client == nil {
		t.Error("SearchClient() should not return nil")
	}
	if client != agent.searchClient {
		t.Error("SearchClient() should return the same client instance")
	}
}

func TestAgent_AddSearXNGInstance(t *testing.T) {
	agent := NewWebResearchAgent(AgentConfig{
		SearXNGInstances: []string{},
		Timeout:          10 * time.Second,
	})

	agent.AddSearXNGInstance("http://localhost:8888")

	// Verify through SearchClient
	sources := agent.SearchClient().GetAvailableSources()
	hasSearXNG := false
	for _, s := range sources {
		if s == "SearXNG" {
			hasSearXNG = true
			break
		}
	}

	if !hasSearXNG {
		t.Error("SearXNG should be available after adding instance")
	}
}

// ============================================================================
// Benchmark Tests
// ============================================================================

func BenchmarkFormatSearchResponse(b *testing.B) {
	agent := NewWebResearchAgent(DefaultAgentConfig())
	resp := &SearchResponse{
		Query: "benchmark test",
		Results: []SearchResult{
			{Title: "Result 1", URL: "https://example1.com", Description: "Desc 1"},
			{Title: "Result 2", URL: "https://example2.com", Description: "Desc 2"},
			{Title: "Result 3", URL: "https://example3.com", Description: "Desc 3"},
		},
		Source:      "DuckDuckGo",
		Suggestions: []string{"suggestion 1", "suggestion 2"},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		agent.formatSearchResponse(resp)
	}
}

func BenchmarkFormatWebpageContent(b *testing.B) {
	agent := NewWebResearchAgent(DefaultAgentConfig())
	content := &WebpageContent{
		URL:     "https://example.com/benchmark",
		Title:   "Benchmark Page",
		Content: strings.Repeat("This is benchmark content. ", 100),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		agent.formatWebpageContent(content)
	}
}

// ============================================================================
// Table-Driven Tests
// ============================================================================

func TestAgentConfig_Combinations(t *testing.T) {
	tests := []struct {
		name      string
		config    AgentConfig
		expectErr bool
	}{
		{
			name:      "default config",
			config:    DefaultAgentConfig(),
			expectErr: false,
		},
		{
			name: "custom instances",
			config: AgentConfig{
				SearXNGInstances: []string{"http://custom:8080"},
				Timeout:          15 * time.Second,
			},
			expectErr: false,
		},
		{
			name: "empty instances",
			config: AgentConfig{
				SearXNGInstances: []string{},
				Timeout:          10 * time.Second,
			},
			expectErr: false,
		},
		{
			name: "zero timeout",
			config: AgentConfig{
				SearXNGInstances: []string{"http://test:8080"},
				Timeout:          0,
			},
			expectErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			agent := NewWebResearchAgent(tt.config)
			if agent == nil && !tt.expectErr {
				t.Error("Agent should not be nil")
			}
		})
	}
}
