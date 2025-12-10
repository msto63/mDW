// ============================================================================
// meinDENKWERK (mDW) - Lokale KI-Plattform
// ============================================================================
//
// Package:     websearch
// Description: Unified web search with fallback chain
// Author:      Mike Stoffels with Claude
// Created:     2025-12-10
// License:     MIT
// ============================================================================

package websearch

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/msto63/mDW/pkg/core/logging"
)

// SearchResult represents a unified search result
type SearchResult struct {
	Title       string    `json:"title"`
	URL         string    `json:"url"`
	Description string    `json:"description"`
	Source      string    `json:"source"`       // Which search engine provided this
	PublishedAt string    `json:"published_at"` // Publication date if available
	Score       float64   `json:"score"`        // Relevance score
}

// SearchResponse represents the response from a search operation
type SearchResponse struct {
	Query       string         `json:"query"`
	Results     []SearchResult `json:"results"`
	TotalFound  int            `json:"total_found"`
	SearchTime  time.Duration  `json:"search_time"`
	Source      string         `json:"source"` // Which backend was used
	Suggestions []string       `json:"suggestions"`
}

// SearchEngine defines the interface for search backends
type SearchEngine interface {
	Search(ctx context.Context, query string, count int) (*SearchResponse, error)
	Name() string
	Available(ctx context.Context) bool
}

// WebSearchClient provides unified web search with fallback
type WebSearchClient struct {
	searxng    *SearXNGClient
	httpClient *http.Client
	logger     *logging.Logger
}

// Config holds configuration for the web search client
type Config struct {
	SearXNGInstances []string      // SearXNG instance URLs
	Timeout          time.Duration // Request timeout
}

// DefaultConfig returns default configuration
func DefaultConfig() Config {
	return Config{
		SearXNGInstances: DefaultSearXNGConfig().Instances,
		Timeout:          20 * time.Second,
	}
}

// NewWebSearchClient creates a new web search client
func NewWebSearchClient(cfg Config) *WebSearchClient {
	searxngCfg := DefaultSearXNGConfig()
	if len(cfg.SearXNGInstances) > 0 {
		searxngCfg.Instances = cfg.SearXNGInstances
	}
	searxngCfg.Timeout = cfg.Timeout

	return &WebSearchClient{
		searxng:    NewSearXNGClient(searxngCfg),
		httpClient: &http.Client{Timeout: cfg.Timeout},
		logger:     logging.New("websearch"),
	}
}

// Search performs a web search using the fallback chain:
// 1. SearXNG (local instance if available, then public)
// 2. DuckDuckGo HTML scraping (always available)
func (c *WebSearchClient) Search(ctx context.Context, query string, count int) (*SearchResponse, error) {
	if count <= 0 {
		count = 5
	}
	if count > 20 {
		count = 20
	}

	start := time.Now()

	// Try SearXNG first (preferred - digital sovereignty)
	resp, err := c.searchSearXNG(ctx, query, count)
	if err == nil && len(resp.Results) > 0 {
		resp.SearchTime = time.Since(start)
		c.logger.Debug("Search completed via SearXNG", "query", query, "results", len(resp.Results))
		return resp, nil
	}
	if err != nil {
		c.logger.Debug("SearXNG search failed", "error", err)
	}

	// Fallback to DuckDuckGo HTML scraping
	resp, err = c.searchDuckDuckGo(ctx, query, count)
	if err != nil {
		return nil, fmt.Errorf("all search engines failed: %w", err)
	}

	resp.SearchTime = time.Since(start)
	c.logger.Debug("Search completed via DuckDuckGo", "query", query, "results", len(resp.Results))
	return resp, nil
}

// searchSearXNG performs a search using SearXNG
func (c *WebSearchClient) searchSearXNG(ctx context.Context, query string, count int) (*SearchResponse, error) {
	opts := DefaultSearchOptions()

	result, err := c.searxng.Search(ctx, query, opts)
	if err != nil {
		return nil, err
	}

	var results []SearchResult
	for i, r := range result.Results {
		if i >= count {
			break
		}
		results = append(results, SearchResult{
			Title:       r.Title,
			URL:         r.URL,
			Description: r.Content,
			Source:      fmt.Sprintf("SearXNG (%s)", r.Engine),
			PublishedAt: r.PublishedDate,
			Score:       r.Score,
		})
	}

	return &SearchResponse{
		Query:       query,
		Results:     results,
		TotalFound:  result.NumberOfResults,
		Source:      "SearXNG",
		Suggestions: result.Suggestions,
	}, nil
}

// searchDuckDuckGo performs a search using DuckDuckGo HTML
func (c *WebSearchClient) searchDuckDuckGo(ctx context.Context, query string, count int) (*SearchResponse, error) {
	searchURL := fmt.Sprintf("https://html.duckduckgo.com/html/?q=%s", url.QueryEscape(query))

	req, err := http.NewRequestWithContext(ctx, "GET", searchURL, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; mDW/1.0)")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	results := c.parseDuckDuckGoResults(string(body), count)

	return &SearchResponse{
		Query:      query,
		Results:    results,
		TotalFound: len(results),
		Source:     "DuckDuckGo",
	}, nil
}

// parseDuckDuckGoResults extracts search results from DuckDuckGo HTML
func (c *WebSearchClient) parseDuckDuckGoResults(html string, count int) []SearchResult {
	var results []SearchResult

	// Pattern for result links
	resultPattern := regexp.MustCompile(`<a[^>]*class="result__a"[^>]*href="([^"]*)"[^>]*>([^<]*)</a>`)
	snippetPattern := regexp.MustCompile(`<a[^>]*class="result__snippet"[^>]*>([^<]*)</a>`)

	resultMatches := resultPattern.FindAllStringSubmatch(html, count*2)
	snippetMatches := snippetPattern.FindAllStringSubmatch(html, count*2)

	for i := 0; i < len(resultMatches) && len(results) < count; i++ {
		if len(resultMatches[i]) >= 3 {
			result := SearchResult{
				URL:    resultMatches[i][1],
				Title:  strings.TrimSpace(resultMatches[i][2]),
				Source: "DuckDuckGo",
			}

			if i < len(snippetMatches) && len(snippetMatches[i]) >= 2 {
				result.Description = strings.TrimSpace(snippetMatches[i][1])
			}

			// Skip tracking URLs
			if result.URL != "" && !strings.Contains(result.URL, "duckduckgo.com") {
				results = append(results, result)
			}
		}
	}

	// Fallback parsing if primary method fails
	if len(results) == 0 {
		results = c.simpleDDGParsing(html, count)
	}

	return results
}

// simpleDDGParsing is a fallback parser for DuckDuckGo
func (c *WebSearchClient) simpleDDGParsing(html string, count int) []SearchResult {
	var results []SearchResult

	urlPattern := regexp.MustCompile(`<a[^>]*class="[^"]*result__url[^"]*"[^>]*href="([^"]*)"[^>]*>([^<]*)</a>`)
	matches := urlPattern.FindAllStringSubmatch(html, count*2)

	for _, match := range matches {
		if len(match) >= 3 && len(results) < count {
			actualURL := match[1]
			if strings.Contains(actualURL, "uddg=") {
				if parts := strings.Split(actualURL, "uddg="); len(parts) > 1 {
					decoded, err := url.QueryUnescape(parts[1])
					if err == nil {
						actualURL = strings.Split(decoded, "&")[0]
					}
				}
			}

			if actualURL != "" && strings.HasPrefix(actualURL, "http") {
				results = append(results, SearchResult{
					URL:         actualURL,
					Title:       strings.TrimSpace(match[2]),
					Description: "",
					Source:      "DuckDuckGo",
				})
			}
		}
	}

	return results
}

// FetchWebpage fetches and extracts text content from a URL
func (c *WebSearchClient) FetchWebpage(ctx context.Context, urlStr string) (*WebpageContent, error) {
	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		return nil, fmt.Errorf("invalid URL: %w", err)
	}

	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return nil, fmt.Errorf("only http and https URLs are supported")
	}

	req, err := http.NewRequestWithContext(ctx, "GET", urlStr, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; mDW/1.0)")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,text/plain")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch URL: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("server returned %d", resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 200000)) // 200KB limit
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	content := &WebpageContent{
		URL:     urlStr,
		Title:   extractTitle(string(body)),
		Content: extractTextFromHTML(string(body)),
	}

	// Limit content length
	if len(content.Content) > 50000 {
		content.Content = content.Content[:50000] + "\n... (gekürzt)"
	}

	return content, nil
}

// WebpageContent represents extracted webpage content
type WebpageContent struct {
	URL     string `json:"url"`
	Title   string `json:"title"`
	Content string `json:"content"`
}

// extractTitle extracts the title from HTML
func extractTitle(html string) string {
	titlePattern := regexp.MustCompile(`(?i)<title[^>]*>([^<]*)</title>`)
	matches := titlePattern.FindStringSubmatch(html)
	if len(matches) >= 2 {
		return strings.TrimSpace(matches[1])
	}
	return ""
}

// extractTextFromHTML removes HTML tags and extracts readable text
func extractTextFromHTML(html string) string {
	// Remove script and style tags with content
	scriptPattern := regexp.MustCompile(`(?i)<script[^>]*>[\s\S]*?</script>`)
	html = scriptPattern.ReplaceAllString(html, "")

	stylePattern := regexp.MustCompile(`(?i)<style[^>]*>[\s\S]*?</style>`)
	html = stylePattern.ReplaceAllString(html, "")

	// Remove HTML comments
	commentPattern := regexp.MustCompile(`<!--[\s\S]*?-->`)
	html = commentPattern.ReplaceAllString(html, "")

	// Replace block elements with newlines
	blockPattern := regexp.MustCompile(`(?i)</(p|div|br|h[1-6]|li|tr)>`)
	html = blockPattern.ReplaceAllString(html, "\n")

	// Remove all remaining HTML tags
	tagPattern := regexp.MustCompile(`<[^>]*>`)
	text := tagPattern.ReplaceAllString(html, "")

	// Decode common HTML entities
	text = strings.ReplaceAll(text, "&nbsp;", " ")
	text = strings.ReplaceAll(text, "&amp;", "&")
	text = strings.ReplaceAll(text, "&lt;", "<")
	text = strings.ReplaceAll(text, "&gt;", ">")
	text = strings.ReplaceAll(text, "&quot;", "\"")
	text = strings.ReplaceAll(text, "&#39;", "'")

	// Clean up whitespace
	whitespacePattern := regexp.MustCompile(`[ \t]+`)
	text = whitespacePattern.ReplaceAllString(text, " ")

	newlinePattern := regexp.MustCompile(`\n\s*\n+`)
	text = newlinePattern.ReplaceAllString(text, "\n\n")

	return strings.TrimSpace(text)
}

// AddSearXNGInstance adds a SearXNG instance (e.g., local instance)
func (c *WebSearchClient) AddSearXNGInstance(url string) {
	c.searxng.AddInstance(url)
}

// GetAvailableSources returns which search sources are available
func (c *WebSearchClient) GetAvailableSources() []string {
	sources := []string{"DuckDuckGo"} // Always available

	if len(c.searxng.GetInstances()) > 0 {
		sources = append([]string{"SearXNG"}, sources...)
	}

	return sources
}
