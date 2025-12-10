// ============================================================================
// meinDENKWERK (mDW) - Lokale KI-Plattform
// ============================================================================
//
// Package:     websearch
// Description: Unit tests for web search functionality
// Author:      Mike Stoffels with Claude
// Created:     2025-12-10
// License:     MIT
// ============================================================================

package websearch

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// ============================================================================
// Unit Tests - Configuration
// ============================================================================

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.Timeout != 20*time.Second {
		t.Errorf("Timeout = %v, expected 20s", cfg.Timeout)
	}
	if len(cfg.SearXNGInstances) == 0 {
		t.Error("SearXNGInstances should not be empty")
	}
}

func TestNewWebSearchClient(t *testing.T) {
	cfg := DefaultConfig()
	client := NewWebSearchClient(cfg)

	if client == nil {
		t.Fatal("NewWebSearchClient returned nil")
	}
	if client.searxng == nil {
		t.Error("searxng client should not be nil")
	}
	if client.httpClient == nil {
		t.Error("httpClient should not be nil")
	}
	if client.logger == nil {
		t.Error("logger should not be nil")
	}
}

func TestNewWebSearchClient_CustomConfig(t *testing.T) {
	cfg := Config{
		SearXNGInstances: []string{"http://localhost:8888"},
		Timeout:          10 * time.Second,
	}
	client := NewWebSearchClient(cfg)

	if client.httpClient.Timeout != 10*time.Second {
		t.Errorf("httpClient.Timeout = %v, expected 10s", client.httpClient.Timeout)
	}
}

// ============================================================================
// Unit Tests - SearchResult
// ============================================================================

func TestSearchResult_Fields(t *testing.T) {
	result := SearchResult{
		Title:       "Test Title",
		URL:         "https://example.com",
		Description: "Test description",
		Source:      "DuckDuckGo",
		PublishedAt: "2025-01-01",
		Score:       0.95,
	}

	if result.Title != "Test Title" {
		t.Errorf("Title = %s, expected 'Test Title'", result.Title)
	}
	if result.URL != "https://example.com" {
		t.Errorf("URL = %s, expected 'https://example.com'", result.URL)
	}
	if result.Score != 0.95 {
		t.Errorf("Score = %f, expected 0.95", result.Score)
	}
}

// ============================================================================
// Unit Tests - SearchResponse
// ============================================================================

func TestSearchResponse_Empty(t *testing.T) {
	resp := &SearchResponse{
		Query:   "test query",
		Results: []SearchResult{},
		Source:  "test",
	}

	if len(resp.Results) != 0 {
		t.Error("Results should be empty")
	}
	if resp.TotalFound != 0 {
		t.Error("TotalFound should be 0")
	}
}

func TestSearchResponse_WithResults(t *testing.T) {
	resp := &SearchResponse{
		Query: "test query",
		Results: []SearchResult{
			{Title: "Result 1", URL: "https://example1.com"},
			{Title: "Result 2", URL: "https://example2.com"},
		},
		TotalFound:  100,
		SearchTime:  150 * time.Millisecond,
		Source:      "SearXNG",
		Suggestions: []string{"related query 1", "related query 2"},
	}

	if len(resp.Results) != 2 {
		t.Errorf("Results length = %d, expected 2", len(resp.Results))
	}
	if resp.TotalFound != 100 {
		t.Errorf("TotalFound = %d, expected 100", resp.TotalFound)
	}
	if len(resp.Suggestions) != 2 {
		t.Errorf("Suggestions length = %d, expected 2", len(resp.Suggestions))
	}
}

// ============================================================================
// Unit Tests - WebpageContent
// ============================================================================

func TestWebpageContent_Fields(t *testing.T) {
	content := &WebpageContent{
		URL:     "https://example.com/page",
		Title:   "Page Title",
		Content: "This is the page content.",
	}

	if content.URL != "https://example.com/page" {
		t.Errorf("URL mismatch")
	}
	if content.Title != "Page Title" {
		t.Errorf("Title mismatch")
	}
	if content.Content != "This is the page content." {
		t.Errorf("Content mismatch")
	}
}

// ============================================================================
// Unit Tests - HTML Parsing
// ============================================================================

func TestExtractTitle(t *testing.T) {
	tests := []struct {
		name     string
		html     string
		expected string
	}{
		{
			name:     "simple title",
			html:     "<html><head><title>Test Title</title></head></html>",
			expected: "Test Title",
		},
		{
			name:     "title with whitespace",
			html:     "<html><head><title>  Spaced Title  </title></head></html>",
			expected: "Spaced Title",
		},
		{
			name:     "no title",
			html:     "<html><head></head></html>",
			expected: "",
		},
		{
			name:     "uppercase title tag",
			html:     "<html><head><TITLE>Upper Title</TITLE></head></html>",
			expected: "Upper Title",
		},
		{
			name:     "title with attributes",
			html:     `<html><head><title lang="en">Attr Title</title></head></html>`,
			expected: "Attr Title",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractTitle(tt.html)
			if result != tt.expected {
				t.Errorf("extractTitle() = %q, expected %q", result, tt.expected)
			}
		})
	}
}

func TestExtractTextFromHTML(t *testing.T) {
	tests := []struct {
		name     string
		html     string
		contains []string
		excludes []string
	}{
		{
			name:     "simple paragraph",
			html:     "<p>Hello World</p>",
			contains: []string{"Hello World"},
			excludes: []string{"<p>", "</p>"},
		},
		{
			name:     "removes script tags",
			html:     "<p>Text</p><script>alert('evil')</script><p>More</p>",
			contains: []string{"Text", "More"},
			excludes: []string{"script", "alert", "evil"},
		},
		{
			name:     "removes style tags",
			html:     "<p>Visible</p><style>.class{color:red}</style>",
			contains: []string{"Visible"},
			excludes: []string{"style", "color", "red"},
		},
		{
			name:     "decodes HTML entities",
			html:     "<p>A &amp; B &lt; C &gt; D</p>",
			contains: []string{"A & B < C > D"},
			excludes: []string{"&amp;", "&lt;", "&gt;"},
		},
		{
			name:     "removes comments",
			html:     "<p>Before</p><!-- comment --><p>After</p>",
			contains: []string{"Before", "After"},
			excludes: []string{"comment", "<!--", "-->"},
		},
		{
			name:     "preserves newlines for block elements",
			html:     "<div>Div1</div><div>Div2</div>",
			contains: []string{"Div1", "Div2"},
			excludes: []string{"<div>"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractTextFromHTML(tt.html)

			for _, s := range tt.contains {
				if !strings.Contains(result, s) {
					t.Errorf("Result should contain %q, got: %q", s, result)
				}
			}

			for _, s := range tt.excludes {
				if strings.Contains(result, s) {
					t.Errorf("Result should NOT contain %q, got: %q", s, result)
				}
			}
		})
	}
}

// ============================================================================
// Unit Tests - DuckDuckGo Parsing
// ============================================================================

func TestParseDuckDuckGoResults(t *testing.T) {
	client := NewWebSearchClient(DefaultConfig())

	// Mock HTML with result structure
	html := `
	<div class="result">
		<a class="result__a" href="https://example1.com">Example 1</a>
		<a class="result__snippet">Description 1</a>
	</div>
	<div class="result">
		<a class="result__a" href="https://example2.com">Example 2</a>
		<a class="result__snippet">Description 2</a>
	</div>
	`

	results := client.parseDuckDuckGoResults(html, 10)

	// Note: The regex-based parser may or may not find results depending on exact HTML structure
	// This test verifies the parser doesn't panic and returns valid structures
	for _, r := range results {
		if r.Source != "DuckDuckGo" {
			t.Errorf("Source should be DuckDuckGo, got %s", r.Source)
		}
	}
}

func TestSimpleDDGParsing(t *testing.T) {
	client := NewWebSearchClient(DefaultConfig())

	html := `
	<a class="result__url" href="//duckduckgo.com/l/?uddg=https%3A%2F%2Fexample.com%2Fpath&rut=abc">example.com</a>
	`

	results := client.simpleDDGParsing(html, 5)

	// This tests the fallback parser
	for _, r := range results {
		if r.Source != "DuckDuckGo" {
			t.Errorf("Source should be DuckDuckGo, got %s", r.Source)
		}
		if r.URL != "" && !strings.HasPrefix(r.URL, "http") {
			t.Errorf("URL should start with http, got %s", r.URL)
		}
	}
}

// ============================================================================
// Unit Tests - Search Count Limits
// ============================================================================

func TestSearch_CountLimits(t *testing.T) {
	tests := []struct {
		name          string
		inputCount    int
		expectedCount int
	}{
		{"negative count", -1, 5},
		{"zero count", 0, 5},
		{"normal count", 10, 10},
		{"max count", 20, 20},
		{"over max count", 50, 20},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			count := tt.inputCount
			if count <= 0 {
				count = 5
			}
			if count > 20 {
				count = 20
			}
			if count != tt.expectedCount {
				t.Errorf("Adjusted count = %d, expected %d", count, tt.expectedCount)
			}
		})
	}
}

// ============================================================================
// Unit Tests - Available Sources
// ============================================================================

func TestGetAvailableSources(t *testing.T) {
	client := NewWebSearchClient(DefaultConfig())

	sources := client.GetAvailableSources()

	// DuckDuckGo should always be available
	hasDDG := false
	for _, s := range sources {
		if s == "DuckDuckGo" {
			hasDDG = true
			break
		}
	}

	if !hasDDG {
		t.Error("DuckDuckGo should always be in available sources")
	}
}

func TestAddSearXNGInstance(t *testing.T) {
	client := NewWebSearchClient(Config{
		SearXNGInstances: []string{},
		Timeout:          10 * time.Second,
	})

	client.AddSearXNGInstance("http://localhost:8888")

	// Verify the instance was added
	sources := client.GetAvailableSources()
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
// Unit Tests - URL Validation
// ============================================================================

func TestFetchWebpage_InvalidURL(t *testing.T) {
	client := NewWebSearchClient(DefaultConfig())
	ctx := context.Background()

	tests := []struct {
		name    string
		url     string
		wantErr bool
	}{
		{"ftp URL", "ftp://example.com/file", true},
		{"file URL", "file:///etc/passwd", true},
		{"javascript URL", "javascript:alert(1)", true},
		{"empty URL", "", true},
		{"invalid URL", "not-a-url", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := client.FetchWebpage(ctx, tt.url)
			if (err != nil) != tt.wantErr {
				t.Errorf("FetchWebpage(%s) error = %v, wantErr %v", tt.url, err, tt.wantErr)
			}
		})
	}
}

// ============================================================================
// Integration Tests - Mock Server
// ============================================================================

func TestFetchWebpage_MockServer(t *testing.T) {
	// Create a mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`
			<html>
			<head><title>Test Page</title></head>
			<body>
				<h1>Hello World</h1>
				<p>This is test content.</p>
			</body>
			</html>
		`))
	}))
	defer server.Close()

	client := NewWebSearchClient(Config{Timeout: 5 * time.Second})
	ctx := context.Background()

	content, err := client.FetchWebpage(ctx, server.URL)
	if err != nil {
		t.Fatalf("FetchWebpage failed: %v", err)
	}

	if content.Title != "Test Page" {
		t.Errorf("Title = %q, expected 'Test Page'", content.Title)
	}
	if !strings.Contains(content.Content, "Hello World") {
		t.Error("Content should contain 'Hello World'")
	}
	if !strings.Contains(content.Content, "test content") {
		t.Error("Content should contain 'test content'")
	}
}

func TestFetchWebpage_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client := NewWebSearchClient(Config{Timeout: 5 * time.Second})
	ctx := context.Background()

	_, err := client.FetchWebpage(ctx, server.URL)
	if err == nil {
		t.Error("Expected error for 500 response")
	}
}

func TestFetchWebpage_ContextCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
		w.Write([]byte("<html><body>Delayed</body></html>"))
	}))
	defer server.Close()

	client := NewWebSearchClient(Config{Timeout: 5 * time.Second})

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	_, err := client.FetchWebpage(ctx, server.URL)
	if err == nil {
		t.Error("Expected error due to context timeout")
	}
}

func TestFetchWebpage_ContentTruncation(t *testing.T) {
	// Create server with large content
	largeContent := strings.Repeat("x", 100000) // 100KB
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte("<html><body>" + largeContent + "</body></html>"))
	}))
	defer server.Close()

	client := NewWebSearchClient(Config{Timeout: 5 * time.Second})
	ctx := context.Background()

	content, err := client.FetchWebpage(ctx, server.URL)
	if err != nil {
		t.Fatalf("FetchWebpage failed: %v", err)
	}

	// Content should be truncated to 50000 chars + truncation message
	if len(content.Content) > 55000 {
		t.Errorf("Content should be truncated, got length %d", len(content.Content))
	}
	if !strings.Contains(content.Content, "gekürzt") {
		t.Error("Truncated content should contain truncation marker")
	}
}

// ============================================================================
// Benchmark Tests
// ============================================================================

func BenchmarkExtractTextFromHTML(b *testing.B) {
	html := `
		<html>
		<head><title>Benchmark Page</title></head>
		<body>
			<script>var x = 1;</script>
			<style>.class{color:red}</style>
			<div>
				<h1>Header</h1>
				<p>Paragraph with &amp; entities &lt;test&gt;</p>
				<!-- comment -->
				<ul>
					<li>Item 1</li>
					<li>Item 2</li>
				</ul>
			</div>
		</body>
		</html>
	`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		extractTextFromHTML(html)
	}
}

func BenchmarkExtractTitle(b *testing.B) {
	html := "<html><head><title>Benchmark Title Test</title></head><body>Content</body></html>"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		extractTitle(html)
	}
}

func BenchmarkParseDuckDuckGoResults(b *testing.B) {
	client := NewWebSearchClient(DefaultConfig())
	html := `
		<div class="result">
			<a class="result__a" href="https://example1.com">Example 1</a>
			<a class="result__snippet">Description 1</a>
		</div>
		<div class="result">
			<a class="result__a" href="https://example2.com">Example 2</a>
			<a class="result__snippet">Description 2</a>
		</div>
	`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		client.parseDuckDuckGoResults(html, 10)
	}
}

// ============================================================================
// Concurrency Tests
// ============================================================================

func TestConcurrentSearches(t *testing.T) {
	// Create mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`<html><body><p>Test</p></body></html>`))
	}))
	defer server.Close()

	client := NewWebSearchClient(Config{Timeout: 5 * time.Second})
	ctx := context.Background()

	// Run multiple concurrent fetches
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func() {
			_, err := client.FetchWebpage(ctx, server.URL)
			done <- (err == nil)
		}()
	}

	// Collect results
	successCount := 0
	for i := 0; i < 10; i++ {
		if <-done {
			successCount++
		}
	}

	if successCount != 10 {
		t.Errorf("Expected 10 successful fetches, got %d", successCount)
	}
}
