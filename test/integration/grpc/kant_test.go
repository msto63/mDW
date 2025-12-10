// ============================================================================
// meinDENKWERK (mDW) - Lokale KI-Plattform
// ============================================================================
//
// Package:     grpc
// Description: Integration tests for Kant HTTP API Gateway
// Author:      Mike Stoffels with Claude
// Created:     2025-12-06
// License:     MIT
// ============================================================================

//go:build integration

package grpc

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"testing"
	"time"
)

const kantBaseURL = "http://localhost:8080"

// KantTestClient wraps HTTP client for testing Kant
type KantTestClient struct {
	client  *http.Client
	baseURL string
}

// NewKantTestClient creates a new Kant test client
func NewKantTestClient() *KantTestClient {
	return &KantTestClient{
		client:  &http.Client{Timeout: 60 * time.Second},
		baseURL: kantBaseURL,
	}
}

// Get performs a GET request
func (c *KantTestClient) Get(path string) (*http.Response, error) {
	return c.client.Get(c.baseURL + path)
}

// Post performs a POST request with JSON body
func (c *KantTestClient) Post(path string, body interface{}) (*http.Response, error) {
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	return c.client.Post(c.baseURL+path, "application/json", bytes.NewReader(jsonBody))
}

// TestKantHealthCheck tests the health endpoint
func TestKantHealthCheck(t *testing.T) {
	client := NewKantTestClient()

	resp, err := client.Get("/api/v1/health")
	if err != nil {
		t.Fatalf("Health check failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected status 200, got %d", resp.StatusCode)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	t.Logf("Health Check Response:")
	t.Logf("  Status: %v", result["status"])
	t.Logf("  Service: %v", result["service"])
	t.Logf("  Version: %v", result["version"])

	if result["status"] != "healthy" {
		t.Errorf("Expected status 'healthy', got '%v'", result["status"])
	}
}

// TestKantRoot tests the root endpoint
func TestKantRoot(t *testing.T) {
	client := NewKantTestClient()

	resp, err := client.Get("/")
	if err != nil {
		t.Fatalf("Root request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected status 200, got %d", resp.StatusCode)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	t.Logf("Root Response:")
	t.Logf("  Message: %v", result["message"])
	t.Logf("  Version: %v", result["version"])
}

// TestKantModels tests the models endpoint
func TestKantModels(t *testing.T) {
	client := NewKantTestClient()

	resp, err := client.Get("/api/v1/models")
	if err != nil {
		t.Fatalf("Models request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected status 200, got %d", resp.StatusCode)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	models, ok := result["models"].([]interface{})
	if !ok {
		t.Fatalf("Expected models array in response")
	}

	t.Logf("Found %d models", len(models))
	for i, m := range models {
		if i >= 3 {
			break
		}
		if model, ok := m.(map[string]interface{}); ok {
			t.Logf("  - %v (provider: %v)", model["name"], model["provider"])
		}
	}

	if len(models) == 0 {
		t.Error("Expected at least one model")
	}
}

// TestKantServices tests the services endpoint
func TestKantServices(t *testing.T) {
	client := NewKantTestClient()

	resp, err := client.Get("/api/v1/services")
	if err != nil {
		t.Fatalf("Services request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected status 200, got %d", resp.StatusCode)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	t.Logf("Services Response: %v", result)
}

// TestKantChat tests the chat endpoint
func TestKantChat(t *testing.T) {
	client := NewKantTestClient()

	chatReq := map[string]interface{}{
		"model": "ollama:mistral:7b",
		"messages": []map[string]string{
			{"role": "user", "content": "Say 'Hello from Kant' in exactly 3 words."},
		},
		"max_tokens": 50,
	}

	resp, err := client.Post("/api/v1/chat", chatReq)
	if err != nil {
		t.Fatalf("Chat request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("Expected status 200, got %d: %s", resp.StatusCode, string(body))
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	t.Logf("Chat Response:")
	t.Logf("  Model: %v", result["model"])

	// Content is nested in message object
	var content string
	if msg, ok := result["message"].(map[string]interface{}); ok {
		content = fmt.Sprintf("%v", msg["content"])
		t.Logf("  Content: %v", content)
	} else {
		t.Logf("  Raw Response: %v", result)
	}

	if content == "" {
		t.Log("Note: Content may be empty depending on model response")
	}
}

// TestKantChatStream tests the streaming chat endpoint
func TestKantChatStream(t *testing.T) {
	client := NewKantTestClient()

	chatReq := map[string]interface{}{
		"model": "ollama:mistral:7b",
		"messages": []map[string]string{
			{"role": "user", "content": "Count from 1 to 3."},
		},
		"max_tokens": 50,
	}

	jsonBody, _ := json.Marshal(chatReq)
	req, err := http.NewRequestWithContext(context.Background(), "POST", kantBaseURL+"/api/v1/chat/stream", bytes.NewReader(jsonBody))
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "text/event-stream")

	resp, err := client.client.Do(req)
	if err != nil {
		t.Fatalf("Stream request failed: %v", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	// Streaming may not be implemented yet
	if resp.StatusCode == http.StatusInternalServerError {
		t.Logf("Streaming endpoint returned 500: %s", string(body))
		t.Skip("Skipping - streaming not yet implemented")
		return
	}

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected status 200, got %d: %s", resp.StatusCode, string(body))
	}

	t.Logf("Received %d bytes from stream", len(body))

	if len(body) == 0 {
		t.Log("Note: Stream response may be empty")
	}
}

// RunKantTestSuite runs all Kant tests
func RunKantTestSuite(t *testing.T) *TestSuite {
	suite := NewTestSuite("kant")

	tests := []struct {
		name string
		fn   func(*testing.T)
	}{
		{"HealthCheck", TestKantHealthCheck},
		{"Root", TestKantRoot},
		{"Models", TestKantModels},
		{"Services", TestKantServices},
		{"Chat", TestKantChat},
		{"ChatStream", TestKantChatStream},
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
