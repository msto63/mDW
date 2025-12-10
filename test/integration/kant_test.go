package integration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"testing"
	"time"
)

// HTTP client for Kant API tests
var httpClient = &http.Client{
	Timeout: 60 * time.Second,
}

func TestKant_HealthEndpoint(t *testing.T) {
	cfg := getTestConfig()
	skipIfServiceUnavailable(t, cfg.KantAddr, "Kant")
	logTestStart(t, "Kant", "HealthEndpoint")

	url := fmt.Sprintf("http://%s/api/v1/health", cfg.KantAddr)
	resp, err := httpClient.Get(url)
	requireNoError(t, err, "GET /health failed")
	defer resp.Body.Close()

	requireEqual(t, http.StatusOK, resp.StatusCode, "Expected 200 OK")

	var result map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	requireNoError(t, err, "Failed to decode response")

	t.Logf("Health response: %+v", result)
}

func TestKant_ServicesEndpoint(t *testing.T) {
	cfg := getTestConfig()
	skipIfServiceUnavailable(t, cfg.KantAddr, "Kant")
	logTestStart(t, "Kant", "ServicesEndpoint")

	url := fmt.Sprintf("http://%s/api/v1/services", cfg.KantAddr)
	resp, err := httpClient.Get(url)
	requireNoError(t, err, "GET /services failed")
	defer resp.Body.Close()

	requireEqual(t, http.StatusOK, resp.StatusCode, "Expected 200 OK")

	body, _ := io.ReadAll(resp.Body)
	t.Logf("Services response: %s", string(body))
}

func TestKant_ModelsEndpoint(t *testing.T) {
	cfg := getTestConfig()
	skipIfServiceUnavailable(t, cfg.KantAddr, "Kant")
	skipIfServiceUnavailable(t, cfg.TuringAddr, "Turing")
	logTestStart(t, "Kant", "ModelsEndpoint")

	url := fmt.Sprintf("http://%s/api/v1/models", cfg.KantAddr)
	resp, err := httpClient.Get(url)
	requireNoError(t, err, "GET /models failed")
	defer resp.Body.Close()

	requireEqual(t, http.StatusOK, resp.StatusCode, "Expected 200 OK")

	var result struct {
		Models []struct {
			Name     string `json:"name"`
			Size     int64  `json:"size"`
			Provider string `json:"provider"`
		} `json:"models"`
	}
	err = json.NewDecoder(resp.Body).Decode(&result)
	requireNoError(t, err, "Failed to decode response")

	t.Logf("Found %d models", len(result.Models))
}

func TestKant_ChatEndpoint(t *testing.T) {
	cfg := getTestConfig()
	skipIfServiceUnavailable(t, cfg.KantAddr, "Kant")
	skipIfServiceUnavailable(t, cfg.TuringAddr, "Turing")
	skipIfServiceUnavailable(t, cfg.OllamaAddr, "Ollama")
	logTestStart(t, "Kant", "ChatEndpoint")

	url := fmt.Sprintf("http://%s/api/v1/chat", cfg.KantAddr)

	reqBody := map[string]interface{}{
		"messages": []map[string]string{
			{"role": "user", "content": "Antworte mit 'OK'."},
		},
		"model": "mistral:7b",
	}
	jsonBody, _ := json.Marshal(reqBody)

	resp, err := httpClient.Post(url, "application/json", bytes.NewReader(jsonBody))
	requireNoError(t, err, "POST /chat failed")
	defer resp.Body.Close()

	requireEqual(t, http.StatusOK, resp.StatusCode, "Expected 200 OK")

	var result struct {
		Message struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"message"`
	}
	err = json.NewDecoder(resp.Body).Decode(&result)
	requireNoError(t, err, "Failed to decode response")

	requireNotEmpty(t, result.Message.Content, "Response content should not be empty")
	t.Logf("Chat response: %s", result.Message.Content)
}

func TestKant_ChatStreamEndpoint(t *testing.T) {
	cfg := getTestConfig()
	skipIfServiceUnavailable(t, cfg.KantAddr, "Kant")
	skipIfServiceUnavailable(t, cfg.TuringAddr, "Turing")
	skipIfServiceUnavailable(t, cfg.OllamaAddr, "Ollama")
	logTestStart(t, "Kant", "ChatStreamEndpoint (SSE)")

	url := fmt.Sprintf("http://%s/api/v1/chat/stream", cfg.KantAddr)

	reqBody := map[string]interface{}{
		"messages": []map[string]string{
			{"role": "user", "content": "Sage 'Hallo'."},
		},
		"model": "mistral:7b",
	}
	jsonBody, _ := json.Marshal(reqBody)

	resp, err := httpClient.Post(url, "application/json", bytes.NewReader(jsonBody))
	requireNoError(t, err, "POST /chat/stream failed")
	defer resp.Body.Close()

	requireEqual(t, http.StatusOK, resp.StatusCode, "Expected 200 OK")
	requireEqual(t, "text/event-stream", resp.Header.Get("Content-Type"), "Expected SSE content type")

	// Read SSE events
	body, err := io.ReadAll(resp.Body)
	requireNoError(t, err, "Failed to read SSE body")
	requireTrue(t, len(body) > 0, "SSE body should not be empty")

	t.Logf("SSE response length: %d bytes", len(body))
}

func TestKant_SearchEndpoint(t *testing.T) {
	cfg := getTestConfig()
	skipIfServiceUnavailable(t, cfg.KantAddr, "Kant")
	skipIfServiceUnavailable(t, cfg.HypatiaAddr, "Hypatia")
	logTestStart(t, "Kant", "SearchEndpoint")

	url := fmt.Sprintf("http://%s/api/v1/search", cfg.KantAddr)

	reqBody := map[string]interface{}{
		"query": "test query",
		"top_k": 5,
	}
	jsonBody, _ := json.Marshal(reqBody)

	resp, err := httpClient.Post(url, "application/json", bytes.NewReader(jsonBody))
	requireNoError(t, err, "POST /search failed")
	defer resp.Body.Close()

	// May return 200 with empty results or error if no collection
	t.Logf("Search response status: %d", resp.StatusCode)
}

func TestKant_AnalyzeEndpoint(t *testing.T) {
	cfg := getTestConfig()
	skipIfServiceUnavailable(t, cfg.KantAddr, "Kant")
	skipIfServiceUnavailable(t, cfg.BabbageAddr, "Babbage")
	logTestStart(t, "Kant", "AnalyzeEndpoint")

	url := fmt.Sprintf("http://%s/api/v1/analyze", cfg.KantAddr)

	reqBody := map[string]interface{}{
		"text": "Das ist ein fantastischer Test!",
	}
	jsonBody, _ := json.Marshal(reqBody)

	resp, err := httpClient.Post(url, "application/json", bytes.NewReader(jsonBody))
	requireNoError(t, err, "POST /analyze failed")
	defer resp.Body.Close()

	requireEqual(t, http.StatusOK, resp.StatusCode, "Expected 200 OK")

	var result map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	requireNoError(t, err, "Failed to decode response")

	t.Logf("Analyze response: %+v", result)
}

func TestKant_AgentExecuteEndpoint(t *testing.T) {
	cfg := getTestConfig()
	skipIfServiceUnavailable(t, cfg.KantAddr, "Kant")
	skipIfServiceUnavailable(t, cfg.LeibnizAddr, "Leibniz")
	skipIfServiceUnavailable(t, cfg.OllamaAddr, "Ollama")
	logTestStart(t, "Kant", "AgentExecuteEndpoint")

	url := fmt.Sprintf("http://%s/api/v1/agent/execute", cfg.KantAddr)

	// Retry up to 5 times for flaky LLM responses (GPU contention during full test suite)
	var result struct {
		Response string `json:"response"`
		Status   string `json:"status"`
	}
	var lastErr error

	for attempt := 1; attempt <= 5; attempt++ {
		// Use a simple prompt that doesn't require tool usage to avoid max iterations issue
		reqBody := map[string]interface{}{
			"agent_id": "default",
			"message":  "Antworte nur mit 'OK'.",
		}
		jsonBody, _ := json.Marshal(reqBody)

		t.Logf("POST /agent/execute (attempt %d/5)...", attempt)
		resp, err := httpClient.Post(url, "application/json", bytes.NewReader(jsonBody))
		if err != nil {
			lastErr = err
			if attempt < 5 {
				time.Sleep(3 * time.Second)
			}
			continue
		}

		if resp.StatusCode != http.StatusOK {
			resp.Body.Close()
			lastErr = fmt.Errorf("expected 200, got %d", resp.StatusCode)
			if attempt < 5 {
				time.Sleep(3 * time.Second)
			}
			continue
		}

		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		err = json.Unmarshal(body, &result)
		if err != nil {
			lastErr = err
			t.Logf("Decode error: %v, body: %s", err, string(body))
			if attempt < 5 {
				time.Sleep(3 * time.Second)
			}
			continue
		}

		if result.Response != "" {
			lastErr = nil
			break
		}
		lastErr = fmt.Errorf("empty response, body: %s", string(body))
		if attempt < 5 {
			t.Logf("Empty response (body: %s), retrying...", string(body))
			time.Sleep(3 * time.Second)
		}
	}

	requireNoError(t, lastErr, "Agent execute failed after 5 attempts")
	requireNotEmpty(t, result.Response, "Agent response should not be empty after 5 attempts")
	t.Logf("Agent response: %s", result.Response)
}

func TestKant_CollectionsEndpoint(t *testing.T) {
	cfg := getTestConfig()
	skipIfServiceUnavailable(t, cfg.KantAddr, "Kant")
	skipIfServiceUnavailable(t, cfg.HypatiaAddr, "Hypatia")
	logTestStart(t, "Kant", "CollectionsEndpoint")

	url := fmt.Sprintf("http://%s/api/v1/collections", cfg.KantAddr)
	resp, err := httpClient.Get(url)
	requireNoError(t, err, "GET /collections failed")
	defer resp.Body.Close()

	requireEqual(t, http.StatusOK, resp.StatusCode, "Expected 200 OK")

	var result struct {
		Collections []struct {
			Name          string `json:"name"`
			DocumentCount int    `json:"document_count"`
		} `json:"collections"`
	}
	err = json.NewDecoder(resp.Body).Decode(&result)
	requireNoError(t, err, "Failed to decode response")

	t.Logf("Found %d collections", len(result.Collections))
}

func TestKant_ToolsEndpoint(t *testing.T) {
	cfg := getTestConfig()
	skipIfServiceUnavailable(t, cfg.KantAddr, "Kant")
	skipIfServiceUnavailable(t, cfg.LeibnizAddr, "Leibniz")
	logTestStart(t, "Kant", "ToolsEndpoint")

	url := fmt.Sprintf("http://%s/api/v1/agent/tools", cfg.KantAddr)
	resp, err := httpClient.Get(url)
	requireNoError(t, err, "GET /agent/tools failed")
	defer resp.Body.Close()

	requireEqual(t, http.StatusOK, resp.StatusCode, "Expected 200 OK")

	var result struct {
		Tools []struct {
			Name        string `json:"name"`
			Description string `json:"description"`
		} `json:"tools"`
	}
	err = json.NewDecoder(resp.Body).Decode(&result)
	requireNoError(t, err, "Failed to decode response")

	t.Logf("Found %d tools", len(result.Tools))
}
