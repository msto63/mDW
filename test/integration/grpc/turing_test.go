// ============================================================================
// meinDENKWERK (mDW) - Lokale KI-Plattform
// ============================================================================
//
// Package:     grpc
// Description: Integration tests for Turing gRPC service
// Author:      Mike Stoffels with Claude
// Created:     2025-12-06
// License:     MIT
// ============================================================================

//go:build integration

package grpc

import (
	"context"
	"io"
	"testing"
	"time"

	"github.com/msto63/mDW/api/gen/common"
	turingpb "github.com/msto63/mDW/api/gen/turing"
)

// TuringTestClient wraps the Turing gRPC client for testing
type TuringTestClient struct {
	conn   *TestConnection
	client turingpb.TuringServiceClient
}

// NewTuringTestClient creates a new Turing test client
func NewTuringTestClient() (*TuringTestClient, error) {
	configs := DefaultServiceConfigs()
	cfg := configs["turing"]

	conn, err := NewTestConnection(cfg)
	if err != nil {
		return nil, err
	}

	return &TuringTestClient{
		conn:   conn,
		client: turingpb.NewTuringServiceClient(conn.Conn()),
	}, nil
}

// Close closes the test client connection
func (tc *TuringTestClient) Close() error {
	return tc.conn.Close()
}

// Client returns the underlying gRPC client
func (tc *TuringTestClient) Client() turingpb.TuringServiceClient {
	return tc.client
}

// Context returns a context with the configured timeout
func (tc *TuringTestClient) Context() context.Context {
	return tc.conn.Context()
}

// ContextWithTimeout returns a context with a custom timeout
func (tc *TuringTestClient) ContextWithTimeout(timeout time.Duration) (context.Context, context.CancelFunc) {
	return tc.conn.ContextWithTimeout(timeout)
}

// TestTuringHealthCheck tests the health check endpoint
func TestTuringHealthCheck(t *testing.T) {
	client, err := NewTuringTestClient()
	if err != nil {
		t.Fatalf("Failed to create Turing client: %v", err)
	}
	defer client.Close()

	ctx, cancel := client.ContextWithTimeout(10 * time.Second)
	defer cancel()

	resp, err := client.Client().HealthCheck(ctx, &common.HealthCheckRequest{})
	if err != nil {
		t.Fatalf("HealthCheck failed: %v", err)
	}

	t.Logf("Health Check Response:")
	t.Logf("  Status: %s", resp.GetStatus())
	t.Logf("  Service: %s", resp.GetService())
	t.Logf("  Version: %s", resp.GetVersion())
	t.Logf("  Uptime: %d seconds", resp.GetUptimeSeconds())

	if resp.GetStatus() != "healthy" {
		t.Errorf("Expected status 'healthy', got '%s'", resp.GetStatus())
	}

	if resp.GetService() != "turing" {
		t.Errorf("Expected service 'turing', got '%s'", resp.GetService())
	}
}

// TestTuringListModels tests listing available models
func TestTuringListModels(t *testing.T) {
	client, err := NewTuringTestClient()
	if err != nil {
		t.Fatalf("Failed to create Turing client: %v", err)
	}
	defer client.Close()

	ctx, cancel := client.ContextWithTimeout(30 * time.Second)
	defer cancel()

	resp, err := client.Client().ListModels(ctx, &common.Empty{})
	if err != nil {
		t.Fatalf("ListModels failed: %v", err)
	}

	t.Logf("Found %d models:", len(resp.GetModels()))
	for _, model := range resp.GetModels() {
		t.Logf("  - %s (provider: %s, size: %d bytes)", model.GetName(), model.GetProvider(), model.GetSize())
	}

	if len(resp.GetModels()) == 0 {
		t.Error("Expected at least one model to be available")
	}
}

// TestTuringChat tests the chat completion endpoint
func TestTuringChat(t *testing.T) {
	client, err := NewTuringTestClient()
	if err != nil {
		t.Fatalf("Failed to create Turing client: %v", err)
	}
	defer client.Close()

	tests := []struct {
		name    string
		model   string
		message string
	}{
		{
			name:    "Mistral 7B",
			model:   "ollama:mistral:7b",
			message: "Say 'Hello from Mistral' in exactly 3 words.",
		},
		{
			name:    "Qwen 2.5 7B",
			model:   "ollama:qwen2.5:7b",
			message: "Say 'Hello from Qwen' in exactly 3 words.",
		},
		{
			name:    "Ministral 3 8B",
			model:   "ollama:ministral-3:8b",
			message: "Say 'Hello from Ministral' in exactly 3 words.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := client.ContextWithTimeout(60 * time.Second)
			defer cancel()

			req := &turingpb.ChatRequest{
				Model: tt.model,
				Messages: []*turingpb.Message{
					{
						Role:    "user",
						Content: tt.message,
					},
				},
				Temperature: 0.7,
				MaxTokens:   100,
			}

			resp, err := client.Client().Chat(ctx, req)
			if err != nil {
				t.Fatalf("Chat failed for %s: %v", tt.model, err)
			}

			t.Logf("Response from %s:", tt.model)
			t.Logf("  Content: %s", resp.GetContent())
			t.Logf("  Model: %s", resp.GetModel())
			t.Logf("  Tokens: prompt=%d, completion=%d, total=%d",
				resp.GetPromptTokens(), resp.GetCompletionTokens(), resp.GetTotalTokens())

			if resp.GetContent() == "" {
				t.Error("Expected non-empty response content")
			}
		})
	}
}

// TestTuringStreamChat tests the streaming chat endpoint
func TestTuringStreamChat(t *testing.T) {
	client, err := NewTuringTestClient()
	if err != nil {
		t.Fatalf("Failed to create Turing client: %v", err)
	}
	defer client.Close()

	ctx, cancel := client.ContextWithTimeout(60 * time.Second)
	defer cancel()

	req := &turingpb.ChatRequest{
		Model: "ollama:mistral:7b",
		Messages: []*turingpb.Message{
			{
				Role:    "user",
				Content: "Count from 1 to 5, one number per line.",
			},
		},
		Temperature: 0.7,
		MaxTokens:   100,
	}

	stream, err := client.Client().StreamChat(ctx, req)
	if err != nil {
		t.Fatalf("StreamChat failed: %v", err)
	}

	var fullContent string
	chunkCount := 0

	for {
		chunk, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("Failed to receive chunk: %v", err)
		}

		chunkCount++
		fullContent += chunk.GetDelta()
	}

	t.Logf("Received %d chunks", chunkCount)
	t.Logf("Full content: %s", fullContent)

	if chunkCount == 0 {
		t.Error("Expected at least one chunk from stream")
	}

	if fullContent == "" {
		t.Error("Expected non-empty content from stream")
	}
}

// TestTuringEmbed tests the embedding endpoint
func TestTuringEmbed(t *testing.T) {
	client, err := NewTuringTestClient()
	if err != nil {
		t.Fatalf("Failed to create Turing client: %v", err)
	}
	defer client.Close()

	ctx, cancel := client.ContextWithTimeout(30 * time.Second)
	defer cancel()

	req := &turingpb.EmbedRequest{
		Model: "ollama:nomic-embed-text",
		Input: "This is a test sentence for embedding.",
	}

	resp, err := client.Client().Embed(ctx, req)
	if err != nil {
		// Embedding might not be available with all models
		t.Skipf("Embed not available: %v", err)
	}

	t.Logf("Embedding dimensions: %d", len(resp.GetEmbedding()))

	if len(resp.GetEmbedding()) == 0 {
		t.Error("Expected non-empty embedding vector")
	}
}

// TestTuringGetModel tests getting a specific model's info
func TestTuringGetModel(t *testing.T) {
	client, err := NewTuringTestClient()
	if err != nil {
		t.Fatalf("Failed to create Turing client: %v", err)
	}
	defer client.Close()

	ctx, cancel := client.ContextWithTimeout(10 * time.Second)
	defer cancel()

	req := &turingpb.GetModelRequest{
		Name: "mistral:7b",
	}

	resp, err := client.Client().GetModel(ctx, req)
	if err != nil {
		t.Fatalf("GetModel failed: %v", err)
	}

	t.Logf("Model Info:")
	t.Logf("  Name: %s", resp.GetName())
	t.Logf("  Provider: %s", resp.GetProvider())
	t.Logf("  Size: %d bytes", resp.GetSize())
	t.Logf("  Available: %v", resp.GetAvailable())
	t.Logf("  Modified At: %s", resp.GetModifiedAt())

	if resp.GetName() == "" {
		t.Error("Expected non-empty model name")
	}
}

// TestTuringChatWithSystemPrompt tests chat with a system prompt
func TestTuringChatWithSystemPrompt(t *testing.T) {
	client, err := NewTuringTestClient()
	if err != nil {
		t.Fatalf("Failed to create Turing client: %v", err)
	}
	defer client.Close()

	ctx, cancel := client.ContextWithTimeout(60 * time.Second)
	defer cancel()

	req := &turingpb.ChatRequest{
		Model: "ollama:mistral:7b",
		Messages: []*turingpb.Message{
			{
				Role:    "user",
				Content: "What is your name?",
			},
		},
		SystemPrompt: "You are a helpful AI assistant named 'TestBot'. Always introduce yourself by name.",
		Temperature:  0.7,
		MaxTokens:    200,
	}

	resp, err := client.Client().Chat(ctx, req)
	if err != nil {
		t.Fatalf("Chat with system prompt failed: %v", err)
	}

	t.Logf("Response with system prompt:")
	t.Logf("  Content: %s", resp.GetContent())

	if resp.GetContent() == "" {
		t.Error("Expected non-empty response content")
	}
}

// RunTuringTestSuite runs all Turing tests and returns a test suite with results
func RunTuringTestSuite(t *testing.T) *TestSuite {
	suite := NewTestSuite("turing")

	tests := []struct {
		name string
		fn   func(*testing.T)
	}{
		{"HealthCheck", TestTuringHealthCheck},
		{"ListModels", TestTuringListModels},
		{"Chat", TestTuringChat},
		{"StreamChat", TestTuringStreamChat},
		{"Embed", TestTuringEmbed},
		{"GetModel", TestTuringGetModel},
		{"ChatWithSystemPrompt", TestTuringChatWithSystemPrompt},
	}

	for _, tt := range tests {
		start := time.Now()
		passed := t.Run(tt.name, tt.fn)
		duration := time.Since(start)

		result := TestResult{
			Name:     tt.name,
			Passed:   passed,
			Duration: duration,
		}
		suite.AddResult(result)
	}

	suite.Finish()
	t.Log(suite.Summary())

	return suite
}
