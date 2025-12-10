// ============================================================================
// meinDENKWERK (mDW) - Lokale KI-Plattform
// ============================================================================
//
// Package:     websearch
// Description: SearXNG client for privacy-respecting web search
// Author:      Mike Stoffels with Claude
// Created:     2025-12-10
// License:     MIT
// ============================================================================

package websearch

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// SearXNGClient is a client for SearXNG instances
type SearXNGClient struct {
	instances  []string
	httpClient *http.Client
	userAgent  string
}

// SearXNGConfig holds SearXNG client configuration
type SearXNGConfig struct {
	Instances      []string      // List of SearXNG instance URLs
	Timeout        time.Duration // Request timeout
	UserAgent      string        // User agent string
}

// DefaultSearXNGConfig returns default configuration with public instances
func DefaultSearXNGConfig() SearXNGConfig {
	return SearXNGConfig{
		Instances: []string{
			// Lokale Instanz hat Priorität (falls konfiguriert)
			// "http://localhost:8888",
			// Öffentliche Instanzen als Fallback
			"https://searx.be",
			"https://search.sapti.me",
			"https://searx.tiekoetter.com",
			"https://search.bus-hit.me",
		},
		Timeout:   15 * time.Second,
		UserAgent: "mDW/1.0 (Web Research Agent)",
	}
}

// NewSearXNGClient creates a new SearXNG client
func NewSearXNGClient(cfg SearXNGConfig) *SearXNGClient {
	return &SearXNGClient{
		instances:  cfg.Instances,
		httpClient: &http.Client{Timeout: cfg.Timeout},
		userAgent:  cfg.UserAgent,
	}
}

// SearXNGResponse represents the JSON response from SearXNG
type SearXNGResponse struct {
	Query           string           `json:"query"`
	NumberOfResults int              `json:"number_of_results"`
	Results         []SearXNGResult  `json:"results"`
	Suggestions     []string         `json:"suggestions"`
	Infoboxes       []SearXNGInfobox `json:"infoboxes"`
}

// SearXNGResult represents a single search result
type SearXNGResult struct {
	URL           string   `json:"url"`
	Title         string   `json:"title"`
	Content       string   `json:"content"`
	Engine        string   `json:"engine"`
	Engines       []string `json:"engines"`
	PublishedDate string   `json:"publishedDate,omitempty"`
	Score         float64  `json:"score"`
}

// SearXNGInfobox represents an infobox from search results
type SearXNGInfobox struct {
	Infobox string `json:"infobox"`
	Content string `json:"content"`
	Engine  string `json:"engine"`
}

// SearchOptions holds options for search requests
type SearchOptions struct {
	Categories []string // Categories: general, images, news, etc.
	Engines    []string // Specific engines to use
	Language   string   // Language code (de, en, etc.)
	TimeRange  string   // Time range: day, week, month, year
	SafeSearch int      // Safe search: 0=off, 1=moderate, 2=strict
	PageNo     int      // Page number (starts at 1)
}

// DefaultSearchOptions returns default search options
func DefaultSearchOptions() SearchOptions {
	return SearchOptions{
		Categories: []string{"general"},
		Language:   "de-DE",
		SafeSearch: 1,
		PageNo:     1,
	}
}

// Search performs a search across configured SearXNG instances
func (c *SearXNGClient) Search(ctx context.Context, query string, opts SearchOptions) (*SearXNGResponse, error) {
	var lastErr error

	// Try each instance until one succeeds
	for _, instance := range c.instances {
		result, err := c.searchInstance(ctx, instance, query, opts)
		if err != nil {
			lastErr = err
			continue
		}
		return result, nil
	}

	if lastErr != nil {
		return nil, fmt.Errorf("all SearXNG instances failed, last error: %w", lastErr)
	}
	return nil, fmt.Errorf("no SearXNG instances configured")
}

// searchInstance performs a search on a specific instance
func (c *SearXNGClient) searchInstance(ctx context.Context, instance, query string, opts SearchOptions) (*SearXNGResponse, error) {
	// Build URL with parameters
	params := url.Values{}
	params.Set("q", query)
	params.Set("format", "json")

	if len(opts.Categories) > 0 {
		params.Set("categories", strings.Join(opts.Categories, ","))
	}
	if len(opts.Engines) > 0 {
		params.Set("engines", strings.Join(opts.Engines, ","))
	}
	if opts.Language != "" {
		params.Set("language", opts.Language)
	}
	if opts.TimeRange != "" {
		params.Set("time_range", opts.TimeRange)
	}
	if opts.SafeSearch > 0 {
		params.Set("safesearch", fmt.Sprintf("%d", opts.SafeSearch))
	}
	if opts.PageNo > 1 {
		params.Set("pageno", fmt.Sprintf("%d", opts.PageNo))
	}

	searchURL := fmt.Sprintf("%s/search?%s", strings.TrimSuffix(instance, "/"), params.Encode())

	req, err := http.NewRequestWithContext(ctx, "GET", searchURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("User-Agent", c.userAgent)
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("instance %s returned %d: %s", instance, resp.StatusCode, string(body))
	}

	var result SearXNGResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &result, nil
}

// AddInstance adds a SearXNG instance to the client
func (c *SearXNGClient) AddInstance(url string) {
	// Add at the beginning for priority
	c.instances = append([]string{url}, c.instances...)
}

// SetInstances replaces all instances
func (c *SearXNGClient) SetInstances(instances []string) {
	c.instances = instances
}

// GetInstances returns configured instances
func (c *SearXNGClient) GetInstances() []string {
	return c.instances
}

// CheckInstance checks if a SearXNG instance is available
func (c *SearXNGClient) CheckInstance(ctx context.Context, instance string) error {
	// Try a simple search to verify the instance works
	_, err := c.searchInstance(ctx, instance, "test", DefaultSearchOptions())
	return err
}
