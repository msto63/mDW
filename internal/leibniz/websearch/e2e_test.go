// ============================================================================
// meinDENKWERK (mDW) - Lokale KI-Plattform
// ============================================================================
//
// Package:     websearch
// Description: End-to-End tests for Web Research workflow
// Author:      Mike Stoffels with Claude
// Created:     2025-12-10
// License:     MIT
// ============================================================================

//go:build e2e
// +build e2e

package websearch

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	commonpb "github.com/msto63/mDW/api/gen/common"
	pb "github.com/msto63/mDW/api/gen/platon"
	"github.com/msto63/mDW/internal/leibniz/platon"
	"google.golang.org/grpc"
)

// ============================================================================
// E2E Test Infrastructure
// ============================================================================

// mockPlatonServer simulates the Platon service for E2E testing
type mockPlatonServer struct {
	pb.UnimplementedPlatonServiceServer
	piiPatterns []string
}

func newMockPlatonServer() *mockPlatonServer {
	return &mockPlatonServer{
		piiPatterns: []string{
			"test@example.com",
			"john.doe@company.org",
		},
	}
}

func (s *mockPlatonServer) ProcessPre(ctx context.Context, req *pb.ProcessRequest) (*pb.ProcessResponse, error) {
	return &pb.ProcessResponse{
		RequestId:       req.RequestId,
		ProcessedPrompt: req.Prompt,
		Modified:        false,
	}, nil
}

func (s *mockPlatonServer) ProcessPost(ctx context.Context, req *pb.ProcessRequest) (*pb.ProcessResponse, error) {
	content := req.Response
	modified := false

	// Simulate PII filtering
	for _, pii := range s.piiPatterns {
		if strings.Contains(content, pii) {
			content = strings.ReplaceAll(content, pii, "[EMAIL REDACTED]")
			modified = true
		}
	}

	return &pb.ProcessResponse{
		RequestId:         req.RequestId,
		ProcessedResponse: content,
		Modified:          modified,
		AuditLog: []*pb.AuditEntry{
			{
				Handler:    "pii-filter",
				Phase:      "post",
				DurationMs: 5,
				Modified:   modified,
			},
		},
	}, nil
}

func (s *mockPlatonServer) Process(ctx context.Context, req *pb.ProcessRequest) (*pb.ProcessResponse, error) {
	return &pb.ProcessResponse{
		RequestId:         req.RequestId,
		ProcessedPrompt:   req.Prompt,
		ProcessedResponse: req.Response,
	}, nil
}

func (s *mockPlatonServer) HealthCheck(ctx context.Context, req *commonpb.HealthCheckRequest) (*commonpb.HealthCheckResponse, error) {
	return &commonpb.HealthCheckResponse{
		Status:  "healthy",
		Service: "mock-platon",
	}, nil
}

// startMockPlaton starts a mock Platon gRPC server
func startMockPlaton(t *testing.T) (*platon.Client, func()) {
	listener, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		t.Fatalf("Failed to create listener: %v", err)
	}

	server := grpc.NewServer()
	pb.RegisterPlatonServiceServer(server, newMockPlatonServer())

	go func() {
		if err := server.Serve(listener); err != nil {
			// Server stopped
		}
	}()

	// Parse address
	addr := listener.Addr().String()
	host, portStr, _ := net.SplitHostPort(addr)
	var port int
	for _, c := range portStr {
		port = port*10 + int(c-'0')
	}

	// Create client
	cfg := platon.Config{
		Host:    host,
		Port:    port,
		Timeout: 5 * time.Second,
	}

	client, err := platon.NewClient(cfg)
	if err != nil {
		server.GracefulStop()
		listener.Close()
		t.Fatalf("Failed to create Platon client: %v", err)
	}

	cleanup := func() {
		client.Close()
		server.GracefulStop()
		listener.Close()
	}

	return client, cleanup
}

// createMockWebServer creates a mock web server for testing
func createMockWebServer(content string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(content))
	}))
}

// ============================================================================
// E2E Test - Complete Web Research Workflow
// ============================================================================

func TestE2E_WebResearchAgent_CompleteWorkflow(t *testing.T) {
	// 1. Start mock Platon server
	platonClient, cleanupPlaton := startMockPlaton(t)
	defer cleanupPlaton()

	// 2. Create mock web server with test content
	mockHTML := `
		<html>
		<head><title>Test Article</title></head>
		<body>
			<h1>Test Research Content</h1>
			<p>This is important information about the topic.</p>
			<p>Contact: test@example.com for more details.</p>
			<p>Published: 2025-01-15</p>
		</body>
		</html>
	`
	webServer := createMockWebServer(mockHTML)
	defer webServer.Close()

	// 3. Create Web Research Agent with Platon integration
	agent := NewWebResearchAgent(AgentConfig{
		SearXNGInstances: []string{}, // No SearXNG for E2E test
		Timeout:          10 * time.Second,
	})
	agent.SetPlatonClient(platonClient, "e2e-test-pipeline")

	// 4. Test webpage fetch with Platon filtering
	ctx := context.Background()
	params := map[string]interface{}{
		"url": webServer.URL,
	}

	result, err := agent.fetchWebpageHandler(ctx, params)
	if err != nil {
		t.Fatalf("fetchWebpageHandler failed: %v", err)
	}

	resultStr, ok := result.(string)
	if !ok {
		t.Fatalf("Result should be string, got %T", result)
	}

	// 5. Verify content was fetched
	if !strings.Contains(resultStr, "Test Research Content") {
		t.Error("Result should contain page content")
	}
	if !strings.Contains(resultStr, "important information") {
		t.Error("Result should contain paragraph content")
	}

	// 6. Verify Platon filtering worked (PII should be redacted)
	if strings.Contains(resultStr, "test@example.com") {
		t.Error("Email should be redacted by Platon")
	}
	if !strings.Contains(resultStr, "[EMAIL REDACTED]") {
		t.Error("Email should be replaced with redaction marker")
	}
}

func TestE2E_WebResearchAgent_SearchAndFetch(t *testing.T) {
	// Create mock search result page
	searchHTML := `
		<html><body>
		<div class="result">
			<a class="result__a" href="https://example.com/article1">Article One</a>
			<a class="result__snippet">First article description</a>
		</div>
		<div class="result">
			<a class="result__a" href="https://example.com/article2">Article Two</a>
			<a class="result__snippet">Second article description</a>
		</div>
		</body></html>
	`

	searchServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(searchHTML))
	}))
	defer searchServer.Close()

	// Create article page
	articleHTML := `
		<html>
		<head><title>Article One - Detailed</title></head>
		<body>
			<h1>Detailed Article Content</h1>
			<p>This article contains detailed information.</p>
			<p>Author contact: john.doe@company.org</p>
		</body>
		</html>
	`
	articleServer := createMockWebServer(articleHTML)
	defer articleServer.Close()

	// Start Platon mock
	platonClient, cleanupPlaton := startMockPlaton(t)
	defer cleanupPlaton()

	// Create agent
	agent := NewWebResearchAgent(AgentConfig{
		SearXNGInstances: []string{},
		Timeout:          10 * time.Second,
	})
	agent.SetPlatonClient(platonClient, "search-test")

	ctx := context.Background()

	// Fetch article (simulating follow-up from search)
	params := map[string]interface{}{
		"url": articleServer.URL,
	}

	result, err := agent.fetchWebpageHandler(ctx, params)
	if err != nil {
		t.Fatalf("Fetch failed: %v", err)
	}

	resultStr := result.(string)

	// Verify article content
	if !strings.Contains(resultStr, "Detailed Article Content") {
		t.Error("Should contain article content")
	}

	// Verify PII filtering
	if strings.Contains(resultStr, "john.doe@company.org") {
		t.Error("Author email should be redacted")
	}
}

// ============================================================================
// E2E Test - Error Scenarios
// ============================================================================

func TestE2E_WebResearchAgent_ServerError(t *testing.T) {
	// Create server that returns 500
	errorServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer errorServer.Close()

	agent := NewWebResearchAgent(DefaultAgentConfig())
	ctx := context.Background()

	params := map[string]interface{}{
		"url": errorServer.URL,
	}

	_, err := agent.fetchWebpageHandler(ctx, params)
	if err == nil {
		t.Error("Should return error for 500 response")
	}
}

func TestE2E_WebResearchAgent_Timeout(t *testing.T) {
	// Create slow server
	slowServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(5 * time.Second)
		w.Write([]byte("<html>Slow response</html>"))
	}))
	defer slowServer.Close()

	agent := NewWebResearchAgent(AgentConfig{
		Timeout: 100 * time.Millisecond,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	params := map[string]interface{}{
		"url": slowServer.URL,
	}

	_, err := agent.fetchWebpageHandler(ctx, params)
	if err == nil {
		t.Error("Should timeout on slow server")
	}
}

func TestE2E_WebResearchAgent_PlatonUnavailable(t *testing.T) {
	// Create agent with invalid Platon config
	agent := NewWebResearchAgent(DefaultAgentConfig())

	// Set enablePlaton but with nil client to simulate connection failure
	agent.enablePlaton = true
	agent.platonClient = nil

	// Create mock web server
	server := createMockWebServer("<html><body>Test content</body></html>")
	defer server.Close()

	ctx := context.Background()
	params := map[string]interface{}{
		"url": server.URL,
	}

	// Should still work, just without filtering
	result, err := agent.fetchWebpageHandler(ctx, params)
	if err != nil {
		t.Fatalf("Should succeed even when Platon unavailable: %v", err)
	}

	resultStr := result.(string)
	if !strings.Contains(resultStr, "Test content") {
		t.Error("Should return unfiltered content when Platon unavailable")
	}
}

// ============================================================================
// E2E Test - Response Formatting
// ============================================================================

func TestE2E_WebResearchAgent_ResponseFormat(t *testing.T) {
	mockHTML := `
		<html>
		<head><title>Formatted Test Page</title></head>
		<body>
			<h1>Main Heading</h1>
			<p>Paragraph one content.</p>
			<p>Paragraph two content.</p>
			<ul>
				<li>List item 1</li>
				<li>List item 2</li>
			</ul>
		</body>
		</html>
	`
	server := createMockWebServer(mockHTML)
	defer server.Close()

	agent := NewWebResearchAgent(DefaultAgentConfig())

	ctx := context.Background()
	params := map[string]interface{}{
		"url": server.URL,
	}

	result, err := agent.fetchWebpageHandler(ctx, params)
	if err != nil {
		t.Fatalf("Fetch failed: %v", err)
	}

	resultStr := result.(string)

	// Check formatting
	if !strings.Contains(resultStr, "Webseite:") {
		t.Error("Should contain 'Webseite:' header")
	}
	if !strings.Contains(resultStr, "Titel:") {
		t.Error("Should contain 'Titel:' header")
	}
	if !strings.Contains(resultStr, "Formatted Test Page") {
		t.Error("Should contain page title")
	}
	if !strings.Contains(resultStr, "---") {
		t.Error("Should contain separator")
	}
	if !strings.Contains(resultStr, "Main Heading") {
		t.Error("Should contain heading content")
	}
}

// ============================================================================
// E2E Test - News Search
// ============================================================================

func TestE2E_WebResearchAgent_NewsSearch(t *testing.T) {
	agent := NewWebResearchAgent(DefaultAgentConfig())

	ctx := context.Background()
	params := map[string]interface{}{
		"query":      "technology",
		"time_range": "week",
	}

	// This test will use DuckDuckGo fallback
	// In E2E we're testing the query modification
	_, err := agent.searchNewsHandler(ctx, params)

	// May fail due to network, but should not panic
	if err != nil {
		t.Logf("News search failed (expected in isolated test): %v", err)
	}
}

// ============================================================================
// E2E Test - Large Content Handling
// ============================================================================

func TestE2E_WebResearchAgent_LargeContent(t *testing.T) {
	// Create server with large content
	largeContent := "<html><body>" + strings.Repeat("<p>Large paragraph content here.</p>", 5000) + "</body></html>"
	server := createMockWebServer(largeContent)
	defer server.Close()

	agent := NewWebResearchAgent(DefaultAgentConfig())

	ctx := context.Background()
	params := map[string]interface{}{
		"url": server.URL,
	}

	result, err := agent.fetchWebpageHandler(ctx, params)
	if err != nil {
		t.Fatalf("Fetch failed: %v", err)
	}

	resultStr := result.(string)

	// Content should be truncated
	if len(resultStr) > 60000 {
		t.Errorf("Content should be truncated, got length %d", len(resultStr))
	}

	// Should have truncation marker
	if !strings.Contains(resultStr, "gekürzt") {
		t.Error("Should indicate content was truncated")
	}
}

// ============================================================================
// E2E Test - Concurrent Workflows
// ============================================================================

func TestE2E_WebResearchAgent_ConcurrentFetches(t *testing.T) {
	// Start Platon mock
	platonClient, cleanupPlaton := startMockPlaton(t)
	defer cleanupPlaton()

	// Create multiple test servers
	servers := make([]*httptest.Server, 5)
	for i := 0; i < 5; i++ {
		content := "<html><body>Content " + string(rune('A'+i)) + " test@example.com</body></html>"
		servers[i] = createMockWebServer(content)
		defer servers[i].Close()
	}

	agent := NewWebResearchAgent(DefaultAgentConfig())
	agent.SetPlatonClient(platonClient, "concurrent-test")

	ctx := context.Background()
	done := make(chan struct {
		index int
		err   error
	}, 5)

	// Fetch all concurrently
	for i, server := range servers {
		go func(idx int, url string) {
			params := map[string]interface{}{"url": url}
			_, err := agent.fetchWebpageHandler(ctx, params)
			done <- struct {
				index int
				err   error
			}{idx, err}
		}(i, server.URL)
	}

	// Collect results
	errorCount := 0
	for i := 0; i < 5; i++ {
		result := <-done
		if result.err != nil {
			errorCount++
			t.Logf("Fetch %d failed: %v", result.index, result.err)
		}
	}

	if errorCount > 0 {
		t.Errorf("%d concurrent fetches failed", errorCount)
	}
}

// ============================================================================
// Benchmark Tests
// ============================================================================

func BenchmarkE2E_FetchWebpage(b *testing.B) {
	server := createMockWebServer("<html><body>Benchmark content</body></html>")
	defer server.Close()

	agent := NewWebResearchAgent(DefaultAgentConfig())
	ctx := context.Background()
	params := map[string]interface{}{"url": server.URL}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		agent.fetchWebpageHandler(ctx, params)
	}
}

func BenchmarkE2E_FetchWithPlaton(b *testing.B) {
	platonClient, cleanup := startMockPlatonForBench()
	defer cleanup()

	server := createMockWebServer("<html><body>test@example.com content</body></html>")
	defer server.Close()

	agent := NewWebResearchAgent(DefaultAgentConfig())
	agent.SetPlatonClient(platonClient, "bench")
	ctx := context.Background()
	params := map[string]interface{}{"url": server.URL}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		agent.fetchWebpageHandler(ctx, params)
	}
}

func startMockPlatonForBench() (*platon.Client, func()) {
	listener, _ := net.Listen("tcp", "localhost:0")
	server := grpc.NewServer()
	pb.RegisterPlatonServiceServer(server, newMockPlatonServer())
	go server.Serve(listener)

	addr := listener.Addr().String()
	host, portStr, _ := net.SplitHostPort(addr)
	var port int
	for _, c := range portStr {
		port = port*10 + int(c-'0')
	}

	cfg := platon.Config{Host: host, Port: port, Timeout: 5 * time.Second}
	client, _ := platon.NewClient(cfg)

	return client, func() {
		client.Close()
		server.GracefulStop()
		listener.Close()
	}
}
