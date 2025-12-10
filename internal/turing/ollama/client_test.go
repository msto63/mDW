package ollama

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.BaseURL != "http://localhost:11434" {
		t.Errorf("BaseURL = %v, want http://localhost:11434", cfg.BaseURL)
	}
	if cfg.Timeout != 120*time.Second {
		t.Errorf("Timeout = %v, want 120s", cfg.Timeout)
	}
}

func TestNewClient(t *testing.T) {
	cfg := DefaultConfig()
	client := NewClient(cfg)

	if client == nil {
		t.Fatal("NewClient() returned nil")
	}
	if client.baseURL != cfg.BaseURL {
		t.Errorf("baseURL = %v, want %v", client.baseURL, cfg.BaseURL)
	}
}

func TestClient_Generate(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/generate" {
			t.Errorf("Path = %v, want /api/generate", r.URL.Path)
		}
		if r.Method != "POST" {
			t.Errorf("Method = %v, want POST", r.Method)
		}

		var req GenerateRequest
		json.NewDecoder(r.Body).Decode(&req)

		if req.Model != "llama3.2" {
			t.Errorf("Model = %v, want llama3.2", req.Model)
		}

		resp := GenerateResponse{
			Model:    "llama3.2",
			Response: "Generated response",
			Done:     true,
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient(Config{BaseURL: server.URL, Timeout: 5 * time.Second})

	resp, err := client.Generate(context.Background(), &GenerateRequest{
		Model:  "llama3.2",
		Prompt: "Hello",
	})

	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	if resp.Response != "Generated response" {
		t.Errorf("Response = %v, want 'Generated response'", resp.Response)
	}
}

func TestClient_Generate_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Internal error"))
	}))
	defer server.Close()

	client := NewClient(Config{BaseURL: server.URL, Timeout: 5 * time.Second})

	_, err := client.Generate(context.Background(), &GenerateRequest{
		Model:  "llama3.2",
		Prompt: "Hello",
	})

	if err == nil {
		t.Error("Generate() should return error for 500 status")
	}
}

func TestClient_Chat(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/chat" {
			t.Errorf("Path = %v, want /api/chat", r.URL.Path)
		}

		var req ChatRequest
		json.NewDecoder(r.Body).Decode(&req)

		if len(req.Messages) == 0 {
			t.Error("Messages should not be empty")
		}

		resp := ChatResponse{
			Model: "llama3.2",
			Message: ChatMessage{
				Role:    "assistant",
				Content: "Hello! How can I help?",
			},
			Done: true,
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient(Config{BaseURL: server.URL, Timeout: 5 * time.Second})

	resp, err := client.Chat(context.Background(), &ChatRequest{
		Model: "llama3.2",
		Messages: []ChatMessage{
			{Role: "user", Content: "Hello"},
		},
	})

	if err != nil {
		t.Fatalf("Chat() error = %v", err)
	}
	if resp.Message.Role != "assistant" {
		t.Errorf("Role = %v, want assistant", resp.Message.Role)
	}
	if resp.Message.Content != "Hello! How can I help?" {
		t.Errorf("Content = %v, want 'Hello! How can I help?'", resp.Message.Content)
	}
}

func TestClient_Embed(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/embed" {
			t.Errorf("Path = %v, want /api/embed", r.URL.Path)
		}

		resp := EmbeddingResponse{
			Model:      "nomic-embed-text",
			Embeddings: [][]float64{{0.1, 0.2, 0.3}},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient(Config{BaseURL: server.URL, Timeout: 5 * time.Second})

	resp, err := client.Embed(context.Background(), &EmbeddingRequest{
		Model: "nomic-embed-text",
		Input: "Test text",
	})

	if err != nil {
		t.Fatalf("Embed() error = %v", err)
	}
	if len(resp.Embeddings) != 1 {
		t.Errorf("Embeddings count = %d, want 1", len(resp.Embeddings))
	}
	if len(resp.Embeddings[0]) != 3 {
		t.Errorf("Embedding dimension = %d, want 3", len(resp.Embeddings[0]))
	}
}

func TestClient_ListModels(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/tags" {
			t.Errorf("Path = %v, want /api/tags", r.URL.Path)
		}
		if r.Method != "GET" {
			t.Errorf("Method = %v, want GET", r.Method)
		}

		resp := ListModelsResponse{
			Models: []ModelInfo{
				{Name: "llama3.2", Size: 1024 * 1024 * 1024},
				{Name: "nomic-embed-text", Size: 512 * 1024 * 1024},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient(Config{BaseURL: server.URL, Timeout: 5 * time.Second})

	resp, err := client.ListModels(context.Background())

	if err != nil {
		t.Fatalf("ListModels() error = %v", err)
	}
	if len(resp.Models) != 2 {
		t.Errorf("Models count = %d, want 2", len(resp.Models))
	}
}

func TestClient_Ping(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			t.Errorf("Path = %v, want /", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewClient(Config{BaseURL: server.URL, Timeout: 5 * time.Second})

	err := client.Ping(context.Background())
	if err != nil {
		t.Errorf("Ping() error = %v", err)
	}
}

func TestClient_Ping_Error(t *testing.T) {
	client := NewClient(Config{BaseURL: "http://localhost:99999", Timeout: 1 * time.Second})

	err := client.Ping(context.Background())
	if err == nil {
		t.Error("Ping() should return error for invalid URL")
	}
}

func TestGenerateRequest_Fields(t *testing.T) {
	req := GenerateRequest{
		Model:   "llama3.2",
		Prompt:  "Hello",
		System:  "You are helpful",
		Stream:  true,
		Format:  "json",
		Options: map[string]interface{}{"temperature": 0.7},
	}

	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}

	var decoded map[string]interface{}
	json.Unmarshal(data, &decoded)

	if decoded["model"] != "llama3.2" {
		t.Errorf("model = %v, want llama3.2", decoded["model"])
	}
	if decoded["prompt"] != "Hello" {
		t.Errorf("prompt = %v, want Hello", decoded["prompt"])
	}
}

func TestChatMessage_Fields(t *testing.T) {
	msg := ChatMessage{
		Role:    "user",
		Content: "Hello",
		Images:  []string{"base64data"},
	}

	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}

	var decoded map[string]interface{}
	json.Unmarshal(data, &decoded)

	if decoded["role"] != "user" {
		t.Errorf("role = %v, want user", decoded["role"])
	}
	if decoded["content"] != "Hello" {
		t.Errorf("content = %v, want Hello", decoded["content"])
	}
}

func TestClient_GenerateStream(t *testing.T) {
	responses := []GenerateResponse{
		{Response: "Hello", Done: false},
		{Response: " World", Done: false},
		{Response: "!", Done: true},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/x-ndjson")
		encoder := json.NewEncoder(w)
		for _, resp := range responses {
			encoder.Encode(resp)
		}
	}))
	defer server.Close()

	client := NewClient(Config{BaseURL: server.URL, Timeout: 5 * time.Second})

	respCh, errCh := client.GenerateStream(context.Background(), &GenerateRequest{
		Model:  "llama3.2",
		Prompt: "Hello",
	})

	var collected []string
	for resp := range respCh {
		collected = append(collected, resp.Response)
	}

	select {
	case err := <-errCh:
		if err != nil {
			t.Errorf("GenerateStream() error = %v", err)
		}
	default:
	}

	if len(collected) != 3 {
		t.Errorf("Collected %d responses, want 3", len(collected))
	}
}

func TestClient_ChatStream(t *testing.T) {
	responses := []ChatResponse{
		{Message: ChatMessage{Content: "Hello"}, Done: false},
		{Message: ChatMessage{Content: " there"}, Done: false},
		{Message: ChatMessage{Content: "!"}, Done: true},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/x-ndjson")
		encoder := json.NewEncoder(w)
		for _, resp := range responses {
			encoder.Encode(resp)
		}
	}))
	defer server.Close()

	client := NewClient(Config{BaseURL: server.URL, Timeout: 5 * time.Second})

	respCh, errCh := client.ChatStream(context.Background(), &ChatRequest{
		Model:    "llama3.2",
		Messages: []ChatMessage{{Role: "user", Content: "Hi"}},
	})

	var collected []string
	for resp := range respCh {
		collected = append(collected, resp.Message.Content)
	}

	select {
	case err := <-errCh:
		if err != nil {
			t.Errorf("ChatStream() error = %v", err)
		}
	default:
	}

	if len(collected) != 3 {
		t.Errorf("Collected %d responses, want 3", len(collected))
	}
}

func TestModelInfo_Fields(t *testing.T) {
	info := ModelInfo{
		Name:       "llama3.2",
		ModifiedAt: time.Now(),
		Size:       1024 * 1024 * 1024,
		Digest:     "abc123",
	}

	if info.Name != "llama3.2" {
		t.Errorf("Name = %v, want llama3.2", info.Name)
	}
	if info.Size != 1024*1024*1024 {
		t.Errorf("Size = %d, want 1GB", info.Size)
	}
}

func TestEmbeddingRequest_Fields(t *testing.T) {
	req := EmbeddingRequest{
		Model:    "nomic-embed-text",
		Input:    "Test",
		Truncate: true,
	}

	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}

	var decoded map[string]interface{}
	json.Unmarshal(data, &decoded)

	if decoded["model"] != "nomic-embed-text" {
		t.Errorf("model = %v, want nomic-embed-text", decoded["model"])
	}
	if decoded["truncate"] != true {
		t.Errorf("truncate = %v, want true", decoded["truncate"])
	}
}
