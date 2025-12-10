package ollama

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Client is the Ollama API client
type Client struct {
	baseURL    string
	httpClient *http.Client
}

// Config holds client configuration
type Config struct {
	BaseURL string
	Timeout time.Duration
}

// DefaultConfig returns default configuration
func DefaultConfig() Config {
	return Config{
		BaseURL: "http://localhost:11434",
		Timeout: 120 * time.Second,
	}
}

// NewClient creates a new Ollama client
func NewClient(cfg Config) *Client {
	return &Client{
		baseURL: cfg.BaseURL,
		httpClient: &http.Client{
			Timeout: cfg.Timeout,
		},
	}
}

// GenerateRequest represents a generate request
type GenerateRequest struct {
	Model     string                 `json:"model"`
	Prompt    string                 `json:"prompt"`
	System    string                 `json:"system,omitempty"`
	Template  string                 `json:"template,omitempty"`
	Context   []int                  `json:"context,omitempty"`
	Stream    bool                   `json:"stream"`
	Raw       bool                   `json:"raw,omitempty"`
	Format    string                 `json:"format,omitempty"`
	Options   map[string]interface{} `json:"options,omitempty"`
	KeepAlive string                 `json:"keep_alive,omitempty"`
}

// GenerateResponse represents a generate response
type GenerateResponse struct {
	Model              string    `json:"model"`
	CreatedAt          time.Time `json:"created_at"`
	Response           string    `json:"response"`
	Done               bool      `json:"done"`
	Context            []int     `json:"context,omitempty"`
	TotalDuration      int64     `json:"total_duration,omitempty"`
	LoadDuration       int64     `json:"load_duration,omitempty"`
	PromptEvalCount    int       `json:"prompt_eval_count,omitempty"`
	PromptEvalDuration int64     `json:"prompt_eval_duration,omitempty"`
	EvalCount          int       `json:"eval_count,omitempty"`
	EvalDuration       int64     `json:"eval_duration,omitempty"`
}

// ChatRequest represents a chat request
type ChatRequest struct {
	Model     string                 `json:"model"`
	Messages  []ChatMessage          `json:"messages"`
	Stream    bool                   `json:"stream"`
	Format    string                 `json:"format,omitempty"`
	Options   map[string]interface{} `json:"options,omitempty"`
	KeepAlive string                 `json:"keep_alive,omitempty"`
}

// ChatMessage represents a chat message
type ChatMessage struct {
	Role    string   `json:"role"`
	Content string   `json:"content"`
	Images  []string `json:"images,omitempty"`
}

// ChatResponse represents a chat response
type ChatResponse struct {
	Model              string      `json:"model"`
	CreatedAt          time.Time   `json:"created_at"`
	Message            ChatMessage `json:"message"`
	Done               bool        `json:"done"`
	TotalDuration      int64       `json:"total_duration,omitempty"`
	LoadDuration       int64       `json:"load_duration,omitempty"`
	PromptEvalCount    int         `json:"prompt_eval_count,omitempty"`
	PromptEvalDuration int64       `json:"prompt_eval_duration,omitempty"`
	EvalCount          int         `json:"eval_count,omitempty"`
	EvalDuration       int64       `json:"eval_duration,omitempty"`
}

// EmbeddingRequest represents an embedding request
type EmbeddingRequest struct {
	Model     string                 `json:"model"`
	Input     interface{}            `json:"input"` // string or []string
	Truncate  bool                   `json:"truncate,omitempty"`
	Options   map[string]interface{} `json:"options,omitempty"`
	KeepAlive string                 `json:"keep_alive,omitempty"`
}

// EmbeddingResponse represents an embedding response
type EmbeddingResponse struct {
	Model      string      `json:"model"`
	Embeddings [][]float64 `json:"embeddings"`
}

// ModelInfo represents model information
type ModelInfo struct {
	Name       string    `json:"name"`
	ModifiedAt time.Time `json:"modified_at"`
	Size       int64     `json:"size"`
	Digest     string    `json:"digest"`
	Details    struct {
		Format            string   `json:"format"`
		Family            string   `json:"family"`
		Families          []string `json:"families"`
		ParameterSize     string   `json:"parameter_size"`
		QuantizationLevel string   `json:"quantization_level"`
	} `json:"details"`
}

// ListModelsResponse represents a list models response
type ListModelsResponse struct {
	Models []ModelInfo `json:"models"`
}

// Generate generates text from a prompt
func (c *Client) Generate(ctx context.Context, req *GenerateRequest) (*GenerateResponse, error) {
	req.Stream = false

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/api/generate", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("request failed with status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var result GenerateResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &result, nil
}

// GenerateStream generates text with streaming
func (c *Client) GenerateStream(ctx context.Context, req *GenerateRequest) (<-chan *GenerateResponse, <-chan error) {
	req.Stream = true
	respCh := make(chan *GenerateResponse, 100)
	errCh := make(chan error, 1)

	go func() {
		defer close(respCh)
		defer close(errCh)

		body, err := json.Marshal(req)
		if err != nil {
			errCh <- fmt.Errorf("failed to marshal request: %w", err)
			return
		}

		httpReq, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/api/generate", bytes.NewReader(body))
		if err != nil {
			errCh <- fmt.Errorf("failed to create request: %w", err)
			return
		}
		httpReq.Header.Set("Content-Type", "application/json")

		resp, err := c.httpClient.Do(httpReq)
		if err != nil {
			errCh <- fmt.Errorf("request failed: %w", err)
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			bodyBytes, _ := io.ReadAll(resp.Body)
			errCh <- fmt.Errorf("request failed with status %d: %s", resp.StatusCode, string(bodyBytes))
			return
		}

		decoder := json.NewDecoder(resp.Body)
		for {
			var result GenerateResponse
			if err := decoder.Decode(&result); err != nil {
				if err == io.EOF {
					return
				}
				errCh <- fmt.Errorf("failed to decode response: %w", err)
				return
			}

			select {
			case respCh <- &result:
			case <-ctx.Done():
				return
			}

			if result.Done {
				return
			}
		}
	}()

	return respCh, errCh
}

// Chat performs a chat completion
func (c *Client) Chat(ctx context.Context, req *ChatRequest) (*ChatResponse, error) {
	req.Stream = false

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/api/chat", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("request failed with status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var result ChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &result, nil
}

// ChatStream performs a chat completion with streaming
func (c *Client) ChatStream(ctx context.Context, req *ChatRequest) (<-chan *ChatResponse, <-chan error) {
	req.Stream = true
	respCh := make(chan *ChatResponse, 100)
	errCh := make(chan error, 1)

	go func() {
		defer close(respCh)
		defer close(errCh)

		body, err := json.Marshal(req)
		if err != nil {
			errCh <- fmt.Errorf("failed to marshal request: %w", err)
			return
		}

		httpReq, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/api/chat", bytes.NewReader(body))
		if err != nil {
			errCh <- fmt.Errorf("failed to create request: %w", err)
			return
		}
		httpReq.Header.Set("Content-Type", "application/json")

		resp, err := c.httpClient.Do(httpReq)
		if err != nil {
			errCh <- fmt.Errorf("request failed: %w", err)
			return
		}
		defer resp.Body.Close()

		decoder := json.NewDecoder(resp.Body)
		for {
			var result ChatResponse
			if err := decoder.Decode(&result); err != nil {
				if err == io.EOF {
					return
				}
				errCh <- fmt.Errorf("failed to decode response: %w", err)
				return
			}

			select {
			case respCh <- &result:
			case <-ctx.Done():
				return
			}

			if result.Done {
				return
			}
		}
	}()

	return respCh, errCh
}

// Embed generates embeddings
func (c *Client) Embed(ctx context.Context, req *EmbeddingRequest) (*EmbeddingResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/api/embed", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("request failed with status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var result EmbeddingResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &result, nil
}

// ListModels lists available models
func (c *Client) ListModels(ctx context.Context) (*ListModelsResponse, error) {
	httpReq, err := http.NewRequestWithContext(ctx, "GET", c.baseURL+"/api/tags", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("request failed with status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var result ListModelsResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &result, nil
}

// Ping checks if Ollama is available
func (c *Client) Ping(ctx context.Context) error {
	httpReq, err := http.NewRequestWithContext(ctx, "GET", c.baseURL+"/", nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("connection failed: %w", err)
	}
	defer resp.Body.Close()

	return nil
}

// PullRequest represents a model pull request
type PullRequest struct {
	Name     string `json:"name"`
	Insecure bool   `json:"insecure,omitempty"`
	Stream   bool   `json:"stream"`
}

// PullProgress represents pull progress
type PullProgress struct {
	Status    string `json:"status"`
	Digest    string `json:"digest,omitempty"`
	Total     int64  `json:"total,omitempty"`
	Completed int64  `json:"completed,omitempty"`
}

// PullModel pulls a model with progress streaming
func (c *Client) PullModel(ctx context.Context, name string) (<-chan *PullProgress, <-chan error) {
	progressCh := make(chan *PullProgress, 100)
	errCh := make(chan error, 1)

	go func() {
		defer close(progressCh)
		defer close(errCh)

		req := &PullRequest{
			Name:   name,
			Stream: true,
		}

		body, err := json.Marshal(req)
		if err != nil {
			errCh <- fmt.Errorf("failed to marshal request: %w", err)
			return
		}

		// Use a longer timeout for model pulling
		client := &http.Client{Timeout: 30 * time.Minute}

		httpReq, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/api/pull", bytes.NewReader(body))
		if err != nil {
			errCh <- fmt.Errorf("failed to create request: %w", err)
			return
		}
		httpReq.Header.Set("Content-Type", "application/json")

		resp, err := client.Do(httpReq)
		if err != nil {
			errCh <- fmt.Errorf("request failed: %w", err)
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			bodyBytes, _ := io.ReadAll(resp.Body)
			errCh <- fmt.Errorf("request failed with status %d: %s", resp.StatusCode, string(bodyBytes))
			return
		}

		decoder := json.NewDecoder(resp.Body)
		for {
			var progress PullProgress
			if err := decoder.Decode(&progress); err != nil {
				if err == io.EOF {
					return
				}
				errCh <- fmt.Errorf("failed to decode response: %w", err)
				return
			}

			select {
			case progressCh <- &progress:
			case <-ctx.Done():
				return
			}

			// Check for completion
			if progress.Status == "success" {
				return
			}
		}
	}()

	return progressCh, errCh
}
