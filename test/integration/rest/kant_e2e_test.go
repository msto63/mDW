//go:build integration

package rest

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"
)

// Test configuration
var (
	kantBaseURL   = getEnvOrDefault("KANT_URL", "http://localhost:8080")
	defaultModel  = getEnvOrDefault("TEST_MODEL", "qwen2.5:7b") // Use available model
	testTimeout   = 60 * time.Second
	agentTimeout  = 150 * time.Second // Agent execution needs longer timeout
)

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// HTTP client for tests
func newTestClient() *http.Client {
	return &http.Client{
		Timeout: testTimeout,
	}
}

// HTTP client for agent tests (longer timeout)
func newAgentTestClient() *http.Client {
	return &http.Client{
		Timeout: agentTimeout,
	}
}

// ============================================================================
// Request/Response Types (matching handler.go)
// ============================================================================

type ChatRequest struct {
	Messages    []Message         `json:"messages"`
	Model       string            `json:"model,omitempty"`
	MaxTokens   int               `json:"max_tokens,omitempty"`
	Temperature float64           `json:"temperature,omitempty"`
	Stream      bool              `json:"stream,omitempty"`
	Context     map[string]string `json:"context,omitempty"`
}

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ChatResponse struct {
	ID      string  `json:"id"`
	Model   string  `json:"model"`
	Created int64   `json:"created"`
	Message Message `json:"message"`
	Usage   Usage   `json:"usage,omitempty"`
}

type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

type SearchRequest struct {
	Query      string  `json:"query"`
	Collection string  `json:"collection,omitempty"`
	TopK       int     `json:"top_k,omitempty"`
	MinScore   float64 `json:"min_score,omitempty"`
}

type SearchResult struct {
	ID       string            `json:"id"`
	Content  string            `json:"content"`
	Score    float64           `json:"score"`
	Metadata map[string]string `json:"metadata,omitempty"`
}

type SearchResponse struct {
	Query   string         `json:"query"`
	Results []SearchResult `json:"results"`
	Total   int            `json:"total"`
}

type IngestRequest struct {
	Content    string            `json:"content"`
	Title      string            `json:"title,omitempty"`
	Source     string            `json:"source,omitempty"`
	Collection string            `json:"collection,omitempty"`
	Metadata   map[string]string `json:"metadata,omitempty"`
}

type IngestResponse struct {
	DocumentID string `json:"document_id"`
	Success    bool   `json:"success"`
}

type AnalyzeRequest struct {
	Text string `json:"text"`
}

type AnalyzeResponse struct {
	Language  string           `json:"language,omitempty"`
	Sentiment *SentimentResult `json:"sentiment,omitempty"`
	Entities  []Entity         `json:"entities,omitempty"`
	Keywords  []Keyword        `json:"keywords,omitempty"`
}

type SentimentResult struct {
	Label      string  `json:"label"`
	Confidence float64 `json:"confidence"`
}

type Entity struct {
	Text  string `json:"text"`
	Type  string `json:"type"`
	Start int    `json:"start"`
	End   int    `json:"end"`
}

type Keyword struct {
	Word  string  `json:"word"`
	Score float64 `json:"score"`
}

type SummarizeRequest struct {
	Text      string `json:"text"`
	MaxLength int    `json:"max_length,omitempty"`
	Style     string `json:"style,omitempty"`
}

type SummarizeResponse struct {
	Summary        string `json:"summary"`
	OriginalLength int    `json:"original_length"`
	SummaryLength  int    `json:"summary_length"`
}

type AgentRequest struct {
	Task     string   `json:"task"`
	Tools    []string `json:"tools,omitempty"`
	MaxSteps int      `json:"max_steps,omitempty"`
}

type AgentResponse struct {
	ID        string      `json:"id"`
	Status    string      `json:"status"`
	Result    string      `json:"result"`
	Steps     []AgentStep `json:"steps,omitempty"`
	ToolsUsed []string    `json:"tools_used,omitempty"`
}

type AgentStep struct {
	Step      int    `json:"step"`
	Action    string `json:"action"`
	Tool      string `json:"tool,omitempty"`
	Input     string `json:"input,omitempty"`
	Output    string `json:"output,omitempty"`
	Reasoning string `json:"reasoning,omitempty"`
}

type HealthResponse struct {
	Status   string            `json:"status"`
	Version  string            `json:"version"`
	Uptime   string            `json:"uptime"`
	Services map[string]string `json:"services,omitempty"`
}

type ModelsResponse struct {
	Models []ModelInfo `json:"models"`
}

type ModelInfo struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Provider    string `json:"provider"`
	Size        int64  `json:"size,omitempty"`
	Description string `json:"description,omitempty"`
}

type ErrorResponse struct {
	Error   string `json:"error"`
	Code    string `json:"code,omitempty"`
	Details string `json:"details,omitempty"`
}

type CollectionRequest struct {
	Name        string            `json:"name"`
	Description string            `json:"description,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

type CollectionResponse struct {
	Name        string            `json:"name"`
	Description string            `json:"description,omitempty"`
	Count       int64             `json:"count"`
	Metadata    map[string]string `json:"metadata,omitempty"`
	CreatedAt   string            `json:"created_at,omitempty"`
}

type CollectionsResponse struct {
	Collections []CollectionResponse `json:"collections"`
	Total       int                  `json:"total"`
}

type EmbedRequest struct {
	Text  string   `json:"text,omitempty"`
	Texts []string `json:"texts,omitempty"`
	Model string   `json:"model,omitempty"`
}

type EmbedResponse struct {
	Embeddings [][]float64 `json:"embeddings"`
	Model      string      `json:"model"`
	Dimensions int         `json:"dimensions"`
}

// ============================================================================
// Helper Functions
// ============================================================================

func doRequest(t *testing.T, method, url string, body interface{}) (*http.Response, []byte) {
	t.Helper()

	var reqBody io.Reader
	if body != nil {
		jsonData, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("Failed to marshal request body: %v", err)
		}
		reqBody = bytes.NewReader(jsonData)
	}

	req, err := http.NewRequest(method, url, reqBody)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	client := newTestClient()
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}

	respBody, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		t.Fatalf("Failed to read response body: %v", err)
	}

	return resp, respBody
}

func doRequestWithContext(ctx context.Context, t *testing.T, method, url string, body interface{}) (*http.Response, []byte) {
	t.Helper()

	var reqBody io.Reader
	if body != nil {
		jsonData, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("Failed to marshal request body: %v", err)
		}
		reqBody = bytes.NewReader(jsonData)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, reqBody)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	client := newTestClient()
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}

	respBody, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		t.Fatalf("Failed to read response body: %v", err)
	}

	return resp, respBody
}

// ============================================================================
// Core API Tests
// ============================================================================

func TestHealthEndpoint(t *testing.T) {
	resp, body := doRequest(t, http.MethodGet, kantBaseURL+"/api/v1/health", nil)

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected status 200, got %d: %s", resp.StatusCode, string(body))
	}

	var health HealthResponse
	if err := json.Unmarshal(body, &health); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if health.Status != "healthy" {
		t.Errorf("Expected status 'healthy', got '%s'", health.Status)
	}

	if health.Version == "" {
		t.Error("Version should not be empty")
	}

	if health.Uptime == "" {
		t.Error("Uptime should not be empty")
	}

	// Check that kant is marked healthy
	if health.Services["kant"] != "healthy" {
		t.Errorf("Expected kant to be healthy, got '%s'", health.Services["kant"])
	}
}

func TestRootEndpoint(t *testing.T) {
	resp, body := doRequest(t, http.MethodGet, kantBaseURL+"/api/v1/", nil)

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected status 200, got %d: %s", resp.StatusCode, string(body))
	}

	var info map[string]interface{}
	if err := json.Unmarshal(body, &info); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if info["name"] != "meinDENKWERK API" {
		t.Errorf("Expected name 'meinDENKWERK API', got '%v'", info["name"])
	}

	// Check that endpoints are listed
	endpoints, ok := info["endpoints"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected endpoints to be a map")
	}

	expectedCategories := []string{"core", "llm", "rag", "nlp", "agent", "admin"}
	for _, cat := range expectedCategories {
		if _, exists := endpoints[cat]; !exists {
			t.Errorf("Expected endpoint category '%s' to be listed", cat)
		}
	}
}

func TestNotFoundEndpoint(t *testing.T) {
	resp, body := doRequest(t, http.MethodGet, kantBaseURL+"/api/v1/nonexistent", nil)

	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("Expected status 404, got %d: %s", resp.StatusCode, string(body))
	}

	var errResp ErrorResponse
	if err := json.Unmarshal(body, &errResp); err != nil {
		t.Fatalf("Failed to unmarshal error response: %v", err)
	}

	if errResp.Code != "not_found" {
		t.Errorf("Expected code 'not_found', got '%s'", errResp.Code)
	}
}

// ============================================================================
// Chat Use Case Tests
// ============================================================================

func TestChat_SimpleMessage(t *testing.T) {
	req := ChatRequest{
		Messages: []Message{
			{Role: "user", Content: "Hello, respond with just 'Hi' and nothing else."},
		},
		Model:     defaultModel,
		MaxTokens: 10,
	}

	resp, body := doRequest(t, http.MethodPost, kantBaseURL+"/api/v1/chat", req)

	if resp.StatusCode == http.StatusServiceUnavailable {
		t.Skip("Turing service not available")
	}

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected status 200, got %d: %s", resp.StatusCode, string(body))
	}

	var chatResp ChatResponse
	if err := json.Unmarshal(body, &chatResp); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if chatResp.ID == "" {
		t.Error("Chat ID should not be empty")
	}

	if chatResp.Message.Role != "assistant" {
		t.Errorf("Expected role 'assistant', got '%s'", chatResp.Message.Role)
	}

	if chatResp.Message.Content == "" {
		t.Error("Message content should not be empty")
	}
}

func TestChat_WithModel(t *testing.T) {
	req := ChatRequest{
		Messages: []Message{
			{Role: "user", Content: "Say 'OK'"},
		},
		Model:     "qwen2.5:7b",
		MaxTokens: 5,
	}

	resp, body := doRequest(t, http.MethodPost, kantBaseURL+"/api/v1/chat", req)

	if resp.StatusCode == http.StatusServiceUnavailable {
		t.Skip("Turing service not available")
	}

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected status 200, got %d: %s", resp.StatusCode, string(body))
	}

	var chatResp ChatResponse
	if err := json.Unmarshal(body, &chatResp); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	// Model might be different if requested model isn't available
	if chatResp.Model == "" {
		t.Error("Model should not be empty")
	}
}

func TestChat_ConversationContext(t *testing.T) {
	// Multi-turn conversation test
	req := ChatRequest{
		Messages: []Message{
			{Role: "user", Content: "My name is Alice."},
			{Role: "assistant", Content: "Hello Alice! Nice to meet you."},
			{Role: "user", Content: "What is my name?"},
		},
		Model:     defaultModel,
		MaxTokens: 20,
	}

	resp, body := doRequest(t, http.MethodPost, kantBaseURL+"/api/v1/chat", req)

	if resp.StatusCode == http.StatusServiceUnavailable {
		t.Skip("Turing service not available")
	}

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected status 200, got %d: %s", resp.StatusCode, string(body))
	}

	var chatResp ChatResponse
	if err := json.Unmarshal(body, &chatResp); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	// Response should mention Alice
	if !strings.Contains(strings.ToLower(chatResp.Message.Content), "alice") {
		t.Logf("Warning: Expected response to contain 'Alice', got: %s", chatResp.Message.Content)
	}
}

func TestChat_EmptyMessages_Error(t *testing.T) {
	req := ChatRequest{
		Messages: []Message{},
	}

	resp, body := doRequest(t, http.MethodPost, kantBaseURL+"/api/v1/chat", req)

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("Expected status 400, got %d: %s", resp.StatusCode, string(body))
	}

	var errResp ErrorResponse
	if err := json.Unmarshal(body, &errResp); err != nil {
		t.Fatalf("Failed to unmarshal error response: %v", err)
	}

	if errResp.Code != "invalid_request" {
		t.Errorf("Expected code 'invalid_request', got '%s'", errResp.Code)
	}
}

func TestChat_WrongMethod_Error(t *testing.T) {
	resp, body := doRequest(t, http.MethodGet, kantBaseURL+"/api/v1/chat", nil)

	if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Fatalf("Expected status 405, got %d: %s", resp.StatusCode, string(body))
	}
}

func TestChat_TokenUsage(t *testing.T) {
	req := ChatRequest{
		Messages: []Message{
			{Role: "user", Content: "Count to three."},
		},
		Model:     defaultModel,
		MaxTokens: 50,
	}

	resp, body := doRequest(t, http.MethodPost, kantBaseURL+"/api/v1/chat", req)

	if resp.StatusCode == http.StatusServiceUnavailable {
		t.Skip("Turing service not available")
	}

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected status 200, got %d: %s", resp.StatusCode, string(body))
	}

	var chatResp ChatResponse
	if err := json.Unmarshal(body, &chatResp); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	// Check that usage is reported
	if chatResp.Usage.TotalTokens <= 0 {
		t.Logf("Warning: Total tokens should be > 0, got %d", chatResp.Usage.TotalTokens)
	}
}

// ============================================================================
// RAG Use Case Tests
// ============================================================================

func TestRAG_IngestDocument(t *testing.T) {
	req := IngestRequest{
		Content:    "meinDENKWERK is a local AI platform for sovereign data processing. It provides chat, RAG, and agent capabilities.",
		Title:      "About mDW",
		Source:     "test-suite",
		Collection: "test-collection",
		Metadata: map[string]string{
			"category": "documentation",
			"version":  "1.0",
		},
	}

	resp, body := doRequest(t, http.MethodPost, kantBaseURL+"/api/v1/ingest", req)

	if resp.StatusCode == http.StatusServiceUnavailable {
		t.Skip("Hypatia service not available")
	}

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected status 200, got %d: %s", resp.StatusCode, string(body))
	}

	var ingestResp IngestResponse
	if err := json.Unmarshal(body, &ingestResp); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if !ingestResp.Success {
		t.Error("Expected success to be true")
	}

	if ingestResp.DocumentID == "" {
		t.Error("Document ID should not be empty")
	}
}

func TestRAG_IngestEmpty_Error(t *testing.T) {
	req := IngestRequest{
		Content: "",
	}

	resp, body := doRequest(t, http.MethodPost, kantBaseURL+"/api/v1/ingest", req)

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("Expected status 400, got %d: %s", resp.StatusCode, string(body))
	}
}

func TestRAG_Search(t *testing.T) {
	// First ingest a document
	ingestReq := IngestRequest{
		Content:    "Kubernetes is a container orchestration platform. It manages containerized workloads and services.",
		Title:      "Kubernetes Guide",
		Collection: "test-search",
	}

	resp, _ := doRequest(t, http.MethodPost, kantBaseURL+"/api/v1/ingest", ingestReq)
	if resp.StatusCode == http.StatusServiceUnavailable {
		t.Skip("Hypatia service not available")
	}

	// Wait for indexing
	time.Sleep(500 * time.Millisecond)

	// Now search
	searchReq := SearchRequest{
		Query:      "container orchestration",
		Collection: "test-search",
		TopK:       5,
	}

	resp, body := doRequest(t, http.MethodPost, kantBaseURL+"/api/v1/search", searchReq)

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected status 200, got %d: %s", resp.StatusCode, string(body))
	}

	var searchResp SearchResponse
	if err := json.Unmarshal(body, &searchResp); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if searchResp.Query != "container orchestration" {
		t.Errorf("Expected query 'container orchestration', got '%s'", searchResp.Query)
	}

	// Results may or may not be found depending on embeddings
	t.Logf("Found %d results for query '%s'", searchResp.Total, searchResp.Query)
}

func TestRAG_SearchEmptyQuery_Error(t *testing.T) {
	req := SearchRequest{
		Query: "",
	}

	resp, body := doRequest(t, http.MethodPost, kantBaseURL+"/api/v1/search", req)

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("Expected status 400, got %d: %s", resp.StatusCode, string(body))
	}
}

func TestRAG_Collections_List(t *testing.T) {
	resp, body := doRequest(t, http.MethodGet, kantBaseURL+"/api/v1/collections", nil)

	if resp.StatusCode == http.StatusServiceUnavailable {
		t.Skip("Hypatia service not available")
	}

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected status 200, got %d: %s", resp.StatusCode, string(body))
	}

	var colResp CollectionsResponse
	if err := json.Unmarshal(body, &colResp); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	t.Logf("Found %d collections", colResp.Total)
}

func TestRAG_Collection_Create(t *testing.T) {
	collectionName := fmt.Sprintf("test-coll-%d", time.Now().UnixNano())
	req := CollectionRequest{
		Name:        collectionName,
		Description: "Test collection for E2E tests",
	}

	resp, body := doRequest(t, http.MethodPost, kantBaseURL+"/api/v1/collections", req)

	if resp.StatusCode == http.StatusServiceUnavailable {
		t.Skip("Hypatia service not available")
	}

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected status 200/201, got %d: %s", resp.StatusCode, string(body))
	}

	var colResp CollectionResponse
	if err := json.Unmarshal(body, &colResp); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if colResp.Name != collectionName {
		t.Errorf("Expected name '%s', got '%s'", collectionName, colResp.Name)
	}

	// Cleanup: Delete the collection
	_, _ = doRequest(t, http.MethodDelete, kantBaseURL+"/api/v1/collections/"+collectionName, nil)
}

func TestRAG_Collection_CreateEmptyName_Error(t *testing.T) {
	req := CollectionRequest{
		Name: "",
	}

	resp, body := doRequest(t, http.MethodPost, kantBaseURL+"/api/v1/collections", req)

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("Expected status 400, got %d: %s", resp.StatusCode, string(body))
	}
}

// ============================================================================
// NLP Use Case Tests
// ============================================================================

func TestNLP_AnalyzeText(t *testing.T) {
	req := AnalyzeRequest{
		Text: "Apple Inc. announced a new product in Cupertino, California. The CEO was very excited about the innovation.",
	}

	resp, body := doRequest(t, http.MethodPost, kantBaseURL+"/api/v1/analyze", req)

	if resp.StatusCode == http.StatusServiceUnavailable {
		t.Skip("Babbage service not available")
	}

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected status 200, got %d: %s", resp.StatusCode, string(body))
	}

	var analyzeResp AnalyzeResponse
	if err := json.Unmarshal(body, &analyzeResp); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	// Check that some analysis was performed
	if analyzeResp.Language == "" {
		t.Logf("Warning: Language detection returned empty")
	}

	t.Logf("Detected language: %s", analyzeResp.Language)
	t.Logf("Found %d entities and %d keywords", len(analyzeResp.Entities), len(analyzeResp.Keywords))
}

func TestNLP_AnalyzeEmpty_Error(t *testing.T) {
	req := AnalyzeRequest{
		Text: "",
	}

	resp, body := doRequest(t, http.MethodPost, kantBaseURL+"/api/v1/analyze", req)

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("Expected status 400, got %d: %s", resp.StatusCode, string(body))
	}
}

func TestNLP_Summarize(t *testing.T) {
	req := SummarizeRequest{
		Text: `Artificial intelligence (AI) is a branch of computer science that aims to create
		intelligent machines that can perform tasks typically requiring human intelligence.
		These tasks include learning, reasoning, problem-solving, perception, and language understanding.
		AI has made significant strides in recent years, with applications ranging from virtual assistants
		and recommendation systems to autonomous vehicles and medical diagnostics. Machine learning,
		a subset of AI, enables systems to learn from data without being explicitly programmed.
		Deep learning, an advanced form of machine learning, uses neural networks to process complex patterns.`,
		MaxLength: 100,
		Style:     "brief",
	}

	resp, body := doRequest(t, http.MethodPost, kantBaseURL+"/api/v1/summarize", req)

	if resp.StatusCode == http.StatusServiceUnavailable {
		t.Skip("Babbage service not available")
	}

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected status 200, got %d: %s", resp.StatusCode, string(body))
	}

	var sumResp SummarizeResponse
	if err := json.Unmarshal(body, &sumResp); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if sumResp.Summary == "" {
		t.Error("Summary should not be empty")
	}

	if sumResp.SummaryLength >= sumResp.OriginalLength {
		t.Logf("Warning: Summary should be shorter than original (summary: %d, original: %d)",
			sumResp.SummaryLength, sumResp.OriginalLength)
	}

	t.Logf("Original: %d chars, Summary: %d chars", sumResp.OriginalLength, sumResp.SummaryLength)
}

func TestNLP_SummarizeEmpty_Error(t *testing.T) {
	req := SummarizeRequest{
		Text: "",
	}

	resp, body := doRequest(t, http.MethodPost, kantBaseURL+"/api/v1/summarize", req)

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("Expected status 400, got %d: %s", resp.StatusCode, string(body))
	}
}

func TestNLP_SummarizeStyles(t *testing.T) {
	text := "The quick brown fox jumps over the lazy dog. This sentence contains every letter of the English alphabet and is often used for typing practice."

	styles := []string{"brief", "detailed", "bullet"}

	for _, style := range styles {
		t.Run(style, func(t *testing.T) {
			req := SummarizeRequest{
				Text:  text,
				Style: style,
			}

			resp, body := doRequest(t, http.MethodPost, kantBaseURL+"/api/v1/summarize", req)

			if resp.StatusCode == http.StatusServiceUnavailable {
				t.Skip("Babbage service not available")
			}

			if resp.StatusCode != http.StatusOK {
				t.Fatalf("Expected status 200, got %d: %s", resp.StatusCode, string(body))
			}

			var sumResp SummarizeResponse
			if err := json.Unmarshal(body, &sumResp); err != nil {
				t.Fatalf("Failed to unmarshal response: %v", err)
			}

			if sumResp.Summary == "" {
				t.Errorf("Summary with style '%s' should not be empty", style)
			}
		})
	}
}

// ============================================================================
// Agent Use Case Tests
// ============================================================================

func TestAgent_ExecuteTask(t *testing.T) {
	req := AgentRequest{
		Task:     "What is 2 + 2?",
		MaxSteps: 3,
	}

	// Use longer timeout for agent execution
	jsonData, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("Failed to marshal request: %v", err)
	}

	httpReq, err := http.NewRequest(http.MethodPost, kantBaseURL+"/api/v1/agent", bytes.NewReader(jsonData))
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	client := newAgentTestClient()
	resp, err := client.Do(httpReq)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Failed to read response: %v", err)
	}

	if resp.StatusCode == http.StatusServiceUnavailable {
		t.Skip("Leibniz service not available")
	}

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected status 200, got %d: %s", resp.StatusCode, string(body))
	}

	var agentResp AgentResponse
	if err := json.Unmarshal(body, &agentResp); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if agentResp.ID == "" {
		t.Error("Agent execution ID should not be empty")
	}

	if agentResp.Status == "" {
		t.Error("Agent status should not be empty")
	}

	t.Logf("Agent execution ID: %s, Status: %s", agentResp.ID, agentResp.Status)
}

func TestAgent_EmptyTask_Error(t *testing.T) {
	req := AgentRequest{
		Task: "",
	}

	resp, body := doRequest(t, http.MethodPost, kantBaseURL+"/api/v1/agent", req)

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("Expected status 400, got %d: %s", resp.StatusCode, string(body))
	}
}

func TestAgent_ListTools(t *testing.T) {
	resp, body := doRequest(t, http.MethodGet, kantBaseURL+"/api/v1/agent/tools", nil)

	if resp.StatusCode == http.StatusServiceUnavailable {
		t.Skip("Leibniz service not available")
	}

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected status 200, got %d: %s", resp.StatusCode, string(body))
	}

	var toolsResp map[string]interface{}
	if err := json.Unmarshal(body, &toolsResp); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	tools, ok := toolsResp["tools"].([]interface{})
	if !ok {
		t.Fatal("Expected tools to be an array")
	}

	t.Logf("Found %d available tools", len(tools))
}

// ============================================================================
// LLM Model Tests
// ============================================================================

func TestLLM_ListModels(t *testing.T) {
	resp, body := doRequest(t, http.MethodGet, kantBaseURL+"/api/v1/models", nil)

	if resp.StatusCode == http.StatusServiceUnavailable {
		t.Skip("Turing service not available")
	}

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected status 200, got %d: %s", resp.StatusCode, string(body))
	}

	var modelsResp ModelsResponse
	if err := json.Unmarshal(body, &modelsResp); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	t.Logf("Found %d models", len(modelsResp.Models))

	for _, model := range modelsResp.Models {
		if model.Name == "" {
			t.Error("Model name should not be empty")
		}
		t.Logf("  - %s (provider: %s)", model.Name, model.Provider)
	}
}

func TestLLM_Embed(t *testing.T) {
	req := EmbedRequest{
		Text: "Hello, this is a test sentence for embedding.",
	}

	resp, body := doRequest(t, http.MethodPost, kantBaseURL+"/api/v1/embed", req)

	if resp.StatusCode == http.StatusServiceUnavailable {
		t.Skip("Turing service not available")
	}

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected status 200, got %d: %s", resp.StatusCode, string(body))
	}

	var embedResp EmbedResponse
	if err := json.Unmarshal(body, &embedResp); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if len(embedResp.Embeddings) == 0 {
		t.Error("Embeddings should not be empty")
	}

	if embedResp.Dimensions <= 0 {
		t.Error("Dimensions should be > 0")
	}

	t.Logf("Generated embedding with %d dimensions", embedResp.Dimensions)
}

func TestLLM_EmbedBatch(t *testing.T) {
	req := EmbedRequest{
		Texts: []string{
			"First sentence for embedding.",
			"Second sentence for embedding.",
			"Third sentence for embedding.",
		},
	}

	resp, body := doRequest(t, http.MethodPost, kantBaseURL+"/api/v1/embed", req)

	if resp.StatusCode == http.StatusServiceUnavailable {
		t.Skip("Turing service not available")
	}

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected status 200, got %d: %s", resp.StatusCode, string(body))
	}

	var embedResp EmbedResponse
	if err := json.Unmarshal(body, &embedResp); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if len(embedResp.Embeddings) != 3 {
		t.Errorf("Expected 3 embeddings, got %d", len(embedResp.Embeddings))
	}

	t.Logf("Generated %d embeddings with %d dimensions each", len(embedResp.Embeddings), embedResp.Dimensions)
}

func TestLLM_EmbedEmpty_Error(t *testing.T) {
	req := EmbedRequest{}

	resp, body := doRequest(t, http.MethodPost, kantBaseURL+"/api/v1/embed", req)

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("Expected status 400, got %d: %s", resp.StatusCode, string(body))
	}
}

// ============================================================================
// Integration Scenarios
// ============================================================================

func TestIntegration_RAGPipeline(t *testing.T) {
	// Full RAG pipeline: Ingest -> Search -> Use in Chat
	collectionName := fmt.Sprintf("rag-test-%d", time.Now().UnixNano())

	// Step 1: Create collection
	colReq := CollectionRequest{
		Name: collectionName,
	}
	resp, _ := doRequest(t, http.MethodPost, kantBaseURL+"/api/v1/collections", colReq)
	if resp.StatusCode == http.StatusServiceUnavailable {
		t.Skip("Hypatia service not available")
	}

	// Step 2: Ingest documents
	documents := []IngestRequest{
		{
			Content:    "The capital of France is Paris. Paris is known for the Eiffel Tower.",
			Title:      "France Info",
			Collection: collectionName,
		},
		{
			Content:    "The capital of Germany is Berlin. Berlin has the Brandenburg Gate.",
			Title:      "Germany Info",
			Collection: collectionName,
		},
	}

	for _, doc := range documents {
		resp, body := doRequest(t, http.MethodPost, kantBaseURL+"/api/v1/ingest", doc)
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("Failed to ingest document: %s", string(body))
		}
	}

	// Wait for indexing
	time.Sleep(1 * time.Second)

	// Step 3: Search
	searchReq := SearchRequest{
		Query:      "What is the capital of France?",
		Collection: collectionName,
		TopK:       3,
	}
	resp, body := doRequest(t, http.MethodPost, kantBaseURL+"/api/v1/search", searchReq)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Search failed: %s", string(body))
	}

	var searchResp SearchResponse
	if err := json.Unmarshal(body, &searchResp); err != nil {
		t.Fatalf("Failed to unmarshal search response: %v", err)
	}

	t.Logf("RAG Pipeline: Ingested %d documents, found %d results", len(documents), searchResp.Total)

	// Cleanup
	_, _ = doRequest(t, http.MethodDelete, kantBaseURL+"/api/v1/collections/"+collectionName, nil)
}

func TestIntegration_NLPToChat(t *testing.T) {
	// Analyze text, then use analysis in chat

	// Step 1: Analyze text
	analyzeReq := AnalyzeRequest{
		Text: "The new iPhone was announced by Apple in California.",
	}
	resp, body := doRequest(t, http.MethodPost, kantBaseURL+"/api/v1/analyze", analyzeReq)
	if resp.StatusCode == http.StatusServiceUnavailable {
		t.Skip("Babbage service not available")
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Analysis failed: %s", string(body))
	}

	var analyzeResp AnalyzeResponse
	if err := json.Unmarshal(body, &analyzeResp); err != nil {
		t.Fatalf("Failed to unmarshal analysis: %v", err)
	}

	// Step 2: Use entities in chat
	entities := []string{}
	for _, e := range analyzeResp.Entities {
		entities = append(entities, e.Text)
	}

	if len(entities) > 0 {
		chatReq := ChatRequest{
			Messages: []Message{
				{Role: "user", Content: fmt.Sprintf("Tell me more about: %s", strings.Join(entities, ", "))},
			},
			Model:     defaultModel,
			MaxTokens: 50,
		}
		resp, body := doRequest(t, http.MethodPost, kantBaseURL+"/api/v1/chat", chatReq)
		if resp.StatusCode == http.StatusServiceUnavailable {
			t.Skip("Turing service not available")
		}
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("Chat failed: %s", string(body))
		}

		t.Logf("NLP found entities: %v, chat responded successfully", entities)
	}
}

// ============================================================================
// CORS and Headers Tests
// ============================================================================

func TestCORS_Headers(t *testing.T) {
	req, err := http.NewRequest(http.MethodOptions, kantBaseURL+"/api/v1/health", nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	client := newTestClient()
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected status 200 for OPTIONS, got %d", resp.StatusCode)
	}

	// Check CORS headers
	if resp.Header.Get("Access-Control-Allow-Origin") != "*" {
		t.Error("Expected Access-Control-Allow-Origin: *")
	}

	if resp.Header.Get("Access-Control-Allow-Methods") == "" {
		t.Error("Expected Access-Control-Allow-Methods header")
	}
}

func TestContentType_JSON(t *testing.T) {
	resp, _ := doRequest(t, http.MethodGet, kantBaseURL+"/api/v1/health", nil)

	contentType := resp.Header.Get("Content-Type")
	if !strings.Contains(contentType, "application/json") {
		t.Errorf("Expected Content-Type to contain 'application/json', got '%s'", contentType)
	}
}

// ============================================================================
// Concurrent Request Tests
// ============================================================================

func TestConcurrent_MultipleHealthChecks(t *testing.T) {
	const numRequests = 10
	results := make(chan int, numRequests)

	for i := 0; i < numRequests; i++ {
		go func() {
			resp, _ := doRequest(t, http.MethodGet, kantBaseURL+"/api/v1/health", nil)
			results <- resp.StatusCode
		}()
	}

	successCount := 0
	for i := 0; i < numRequests; i++ {
		status := <-results
		if status == http.StatusOK {
			successCount++
		}
	}

	if successCount != numRequests {
		t.Errorf("Expected all %d requests to succeed, got %d", numRequests, successCount)
	}
}
