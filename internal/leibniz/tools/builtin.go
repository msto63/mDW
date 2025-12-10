// ============================================================================
// meinDENKWERK (mDW) - Lokale KI-Plattform
// ============================================================================
//
// Package:     tools
// Description: Built-in tools for Leibniz agent service
// Author:      Mike Stoffels with Claude
// Created:     2025-12-06
// License:     MIT
// ============================================================================

package tools

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/msto63/mDW/internal/leibniz/agent"
	"github.com/msto63/mDW/pkg/core/logging"
)

// BuiltinTools provides built-in tools for agents
type BuiltinTools struct {
	logger       *logging.Logger
	allowedPaths []string
	httpClient   *http.Client
}

// Config holds configuration for built-in tools
type Config struct {
	AllowedPaths    []string // Paths where file operations are allowed
	HTTPTimeout     time.Duration
	EnableNetwork   bool
	EnableShell     bool
	EnableWebSearch bool // Enable web search tool (deprecated, use WebResearchAgent)
}

// DefaultConfig returns default configuration
func DefaultConfig() Config {
	homeDir, _ := os.UserHomeDir()
	return Config{
		AllowedPaths:    []string{homeDir},
		HTTPTimeout:     30 * time.Second,
		EnableNetwork:   true,
		EnableShell:     false, // Disabled by default for security
		EnableWebSearch: false, // Deprecated, use WebResearchAgent instead
	}
}

// NewBuiltinTools creates new built-in tools
func NewBuiltinTools(cfg Config) *BuiltinTools {
	return &BuiltinTools{
		logger:       logging.New("builtin-tools"),
		allowedPaths: cfg.AllowedPaths,
		httpClient:   &http.Client{Timeout: cfg.HTTPTimeout},
	}
}

// RegisterAll registers all built-in tools with the agent
func (b *BuiltinTools) RegisterAll(ag *agent.Agent, cfg Config) {
	// File operations
	b.registerFileTools(ag)

	// Network operations (if enabled)
	if cfg.EnableNetwork {
		b.registerNetworkTools(ag)
	}

	// Web search (if enabled)
	if cfg.EnableWebSearch {
		b.registerWebSearchTools(ag)
	}

	// Shell operations (if enabled)
	if cfg.EnableShell {
		b.registerShellTools(ag)
	}

	// Utility tools
	b.registerUtilityTools(ag)

	b.logger.Info("Built-in tools registered",
		"file", true,
		"network", cfg.EnableNetwork,
		"websearch", cfg.EnableWebSearch,
		"shell", cfg.EnableShell,
	)
}

// registerFileTools registers file system tools
func (b *BuiltinTools) registerFileTools(ag *agent.Agent) {
	// Read file
	ag.RegisterTool(&agent.Tool{
		Name:        "read_file",
		Description: "Liest den Inhalt einer Datei",
		Parameters: map[string]agent.ParameterDef{
			"path": {Type: "string", Description: "Pfad zur Datei", Required: true},
		},
		Handler: b.readFile,
	})

	// Write file
	ag.RegisterTool(&agent.Tool{
		Name:        "write_file",
		Description: "Schreibt Inhalt in eine Datei (ACHTUNG: ändert Dateien!)",
		Parameters: map[string]agent.ParameterDef{
			"path":    {Type: "string", Description: "Pfad zur Datei", Required: true},
			"content": {Type: "string", Description: "Inhalt der Datei", Required: true},
		},
		Handler: b.writeFile,
	})

	// List directory
	ag.RegisterTool(&agent.Tool{
		Name:        "list_directory",
		Description: "Listet Dateien und Verzeichnisse auf",
		Parameters: map[string]agent.ParameterDef{
			"path": {Type: "string", Description: "Pfad zum Verzeichnis", Required: true},
		},
		Handler: b.listDirectory,
	})

	// Search files
	ag.RegisterTool(&agent.Tool{
		Name:        "search_files",
		Description: "Sucht nach Dateien mit einem Muster",
		Parameters: map[string]agent.ParameterDef{
			"path":    {Type: "string", Description: "Startverzeichnis", Required: true},
			"pattern": {Type: "string", Description: "Suchmuster (z.B. *.txt)", Required: true},
		},
		Handler: b.searchFiles,
	})

	// Create directory
	ag.RegisterTool(&agent.Tool{
		Name:        "create_directory",
		Description: "Erstellt ein Verzeichnis",
		Parameters: map[string]agent.ParameterDef{
			"path": {Type: "string", Description: "Pfad zum Verzeichnis", Required: true},
		},
		Handler: b.createDirectory,
	})

	// Delete file
	ag.RegisterTool(&agent.Tool{
		Name:        "delete_file",
		Description: "Löscht eine Datei oder ein leeres Verzeichnis (ACHTUNG: unwiderruflich!)",
		Parameters: map[string]agent.ParameterDef{
			"path": {Type: "string", Description: "Pfad zur Datei/Verzeichnis", Required: true},
		},
		Handler: b.deleteFile,
	})

	// File info
	ag.RegisterTool(&agent.Tool{
		Name:        "file_info",
		Description: "Gibt Informationen über eine Datei zurück",
		Parameters: map[string]agent.ParameterDef{
			"path": {Type: "string", Description: "Pfad zur Datei", Required: true},
		},
		Handler: b.fileInfo,
	})
}

// registerNetworkTools registers network-related tools
func (b *BuiltinTools) registerNetworkTools(ag *agent.Agent) {
	// HTTP GET
	ag.RegisterTool(&agent.Tool{
		Name:        "http_get",
		Description: "Führt eine HTTP GET-Anfrage aus",
		Parameters: map[string]agent.ParameterDef{
			"url": {Type: "string", Description: "URL für die Anfrage", Required: true},
		},
		Handler: b.httpGet,
	})

	// HTTP POST
	ag.RegisterTool(&agent.Tool{
		Name:        "http_post",
		Description: "Führt eine HTTP POST-Anfrage aus",
		Parameters: map[string]agent.ParameterDef{
			"url":          {Type: "string", Description: "URL für die Anfrage", Required: true},
			"body":         {Type: "string", Description: "Request-Body", Required: true},
			"content_type": {Type: "string", Description: "Content-Type Header", Required: false},
		},
		Handler: b.httpPost,
	})
}

// registerWebSearchTools registers web search tools
// Note: These are deprecated - use WebResearchAgent instead for better results
func (b *BuiltinTools) registerWebSearchTools(ag *agent.Agent) {
	// Web search via DuckDuckGo (deprecated - use WebResearchAgent)
	ag.RegisterTool(&agent.Tool{
		Name:        "web_search",
		Description: "Durchsucht das Internet nach aktuellen Informationen (deprecated - use WebResearchAgent). Nutze dieses Tool für Fragen zu aktuellen Ereignissen, Nachrichten, Fakten oder wenn du aktuelle Daten benötigst.",
		Parameters: map[string]agent.ParameterDef{
			"query": {Type: "string", Description: "Suchanfrage", Required: true},
			"count": {Type: "string", Description: "Anzahl der Ergebnisse (Standard: 5, Max: 10)", Required: false},
		},
		Handler: b.webSearch,
	})

	// Fetch webpage content
	ag.RegisterTool(&agent.Tool{
		Name:        "fetch_webpage",
		Description: "Lädt den Textinhalt einer Webseite herunter. Nutze dieses Tool um Details von einer spezifischen URL zu erhalten.",
		Parameters: map[string]agent.ParameterDef{
			"url": {Type: "string", Description: "URL der Webseite", Required: true},
		},
		Handler: b.fetchWebpage,
	})
}

// registerShellTools registers shell command tools
func (b *BuiltinTools) registerShellTools(ag *agent.Agent) {
	// Execute command
	ag.RegisterTool(&agent.Tool{
		Name:        "shell_command",
		Description: "Führt einen Shell-Befehl aus (eingeschränkt auf sichere Befehle)",
		Parameters: map[string]agent.ParameterDef{
			"command": {Type: "string", Description: "Auszuführender Befehl", Required: true},
			"args":    {Type: "string", Description: "Argumente (kommasepariert)", Required: false},
		},
		Handler: b.shellCommand,
	})
}

// registerUtilityTools registers utility tools
func (b *BuiltinTools) registerUtilityTools(ag *agent.Agent) {
	// Get current time
	ag.RegisterTool(&agent.Tool{
		Name:        "current_time",
		Description: "Gibt die aktuelle Uhrzeit und das Datum zurück",
		Parameters:  map[string]agent.ParameterDef{},
		Handler: func(ctx context.Context, params map[string]interface{}) (interface{}, error) {
			return time.Now().Format("2006-01-02 15:04:05 MST"), nil
		},
	})

	// Environment variable
	ag.RegisterTool(&agent.Tool{
		Name:        "get_env",
		Description: "Gibt den Wert einer Umgebungsvariable zurück",
		Parameters: map[string]agent.ParameterDef{
			"name": {Type: "string", Description: "Name der Variable", Required: true},
		},
		Handler: func(ctx context.Context, params map[string]interface{}) (interface{}, error) {
			name, ok := params["name"].(string)
			if !ok {
				return nil, fmt.Errorf("name parameter required")
			}
			value := os.Getenv(name)
			if value == "" {
				return fmt.Sprintf("Environment variable '%s' not set", name), nil
			}
			return value, nil
		},
	})

	// Working directory
	ag.RegisterTool(&agent.Tool{
		Name:        "get_cwd",
		Description: "Gibt das aktuelle Arbeitsverzeichnis zurück",
		Parameters:  map[string]agent.ParameterDef{},
		Handler: func(ctx context.Context, params map[string]interface{}) (interface{}, error) {
			return os.Getwd()
		},
	})
}

// Tool implementations

func (b *BuiltinTools) readFile(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	path, ok := params["path"].(string)
	if !ok {
		return nil, fmt.Errorf("path parameter required")
	}

	if !b.isPathAllowed(path) {
		return nil, fmt.Errorf("access denied: path not in allowed directories")
	}

	content, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	// Limit content size
	if len(content) > 100000 {
		return string(content[:100000]) + "\n... (truncated)", nil
	}

	return string(content), nil
}

func (b *BuiltinTools) writeFile(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	path, ok := params["path"].(string)
	if !ok {
		return nil, fmt.Errorf("path parameter required")
	}
	content, ok := params["content"].(string)
	if !ok {
		return nil, fmt.Errorf("content parameter required")
	}

	if !b.isPathAllowed(path) {
		return nil, fmt.Errorf("access denied: path not in allowed directories")
	}

	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create directory: %w", err)
	}

	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return nil, fmt.Errorf("failed to write file: %w", err)
	}

	return fmt.Sprintf("File written: %s (%d bytes)", path, len(content)), nil
}

func (b *BuiltinTools) listDirectory(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	path, ok := params["path"].(string)
	if !ok {
		return nil, fmt.Errorf("path parameter required")
	}

	if !b.isPathAllowed(path) {
		return nil, fmt.Errorf("access denied: path not in allowed directories")
	}

	entries, err := os.ReadDir(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory: %w", err)
	}

	var result strings.Builder
	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil {
			continue
		}

		typeChar := "-"
		if entry.IsDir() {
			typeChar = "d"
		}

		result.WriteString(fmt.Sprintf("%s %10d %s %s\n",
			typeChar,
			info.Size(),
			info.ModTime().Format("2006-01-02 15:04"),
			entry.Name(),
		))
	}

	return result.String(), nil
}

func (b *BuiltinTools) searchFiles(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	path, ok := params["path"].(string)
	if !ok {
		return nil, fmt.Errorf("path parameter required")
	}
	pattern, ok := params["pattern"].(string)
	if !ok {
		return nil, fmt.Errorf("pattern parameter required")
	}

	if !b.isPathAllowed(path) {
		return nil, fmt.Errorf("access denied: path not in allowed directories")
	}

	var matches []string
	err := filepath.Walk(path, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip errors
		}

		if len(matches) >= 100 {
			return filepath.SkipDir // Limit results
		}

		matched, _ := filepath.Match(pattern, filepath.Base(p))
		if matched {
			matches = append(matches, p)
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("search failed: %w", err)
	}

	if len(matches) == 0 {
		return "No files found matching pattern", nil
	}

	return strings.Join(matches, "\n"), nil
}

func (b *BuiltinTools) createDirectory(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	path, ok := params["path"].(string)
	if !ok {
		return nil, fmt.Errorf("path parameter required")
	}

	if !b.isPathAllowed(path) {
		return nil, fmt.Errorf("access denied: path not in allowed directories")
	}

	if err := os.MkdirAll(path, 0755); err != nil {
		return nil, fmt.Errorf("failed to create directory: %w", err)
	}

	return fmt.Sprintf("Directory created: %s", path), nil
}

func (b *BuiltinTools) deleteFile(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	path, ok := params["path"].(string)
	if !ok {
		return nil, fmt.Errorf("path parameter required")
	}

	if !b.isPathAllowed(path) {
		return nil, fmt.Errorf("access denied: path not in allowed directories")
	}

	if err := os.Remove(path); err != nil {
		return nil, fmt.Errorf("failed to delete: %w", err)
	}

	return fmt.Sprintf("Deleted: %s", path), nil
}

func (b *BuiltinTools) fileInfo(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	path, ok := params["path"].(string)
	if !ok {
		return nil, fmt.Errorf("path parameter required")
	}

	if !b.isPathAllowed(path) {
		return nil, fmt.Errorf("access denied: path not in allowed directories")
	}

	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("failed to get file info: %w", err)
	}

	return map[string]interface{}{
		"name":     info.Name(),
		"size":     info.Size(),
		"mode":     info.Mode().String(),
		"mod_time": info.ModTime().Format(time.RFC3339),
		"is_dir":   info.IsDir(),
	}, nil
}

func (b *BuiltinTools) httpGet(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	url, ok := params["url"].(string)
	if !ok {
		return nil, fmt.Errorf("url parameter required")
	}

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := b.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 50000)) // Limit response size
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	return map[string]interface{}{
		"status": resp.Status,
		"body":   string(body),
	}, nil
}

func (b *BuiltinTools) httpPost(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	url, ok := params["url"].(string)
	if !ok {
		return nil, fmt.Errorf("url parameter required")
	}
	body, ok := params["body"].(string)
	if !ok {
		return nil, fmt.Errorf("body parameter required")
	}

	contentType := "application/json"
	if ct, ok := params["content_type"].(string); ok && ct != "" {
		contentType = ct
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, strings.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", contentType)

	resp, err := b.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, 50000))
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	return map[string]interface{}{
		"status": resp.Status,
		"body":   string(respBody),
	}, nil
}

func (b *BuiltinTools) shellCommand(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	command, ok := params["command"].(string)
	if !ok {
		return nil, fmt.Errorf("command parameter required")
	}

	// Security: only allow certain commands
	allowedCommands := map[string]bool{
		"ls": true, "cat": true, "head": true, "tail": true,
		"grep": true, "find": true, "wc": true, "sort": true,
		"date": true, "pwd": true, "echo": true, "which": true,
	}

	baseCmd := strings.Split(command, " ")[0]
	if !allowedCommands[baseCmd] {
		return nil, fmt.Errorf("command not allowed: %s", baseCmd)
	}

	var args []string
	if argsStr, ok := params["args"].(string); ok && argsStr != "" {
		args = strings.Split(argsStr, ",")
		for i := range args {
			args[i] = strings.TrimSpace(args[i])
		}
	}

	cmd := exec.CommandContext(ctx, command, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Sprintf("Command failed: %s\nOutput: %s", err, string(output)), nil
	}

	return string(output), nil
}

// isPathAllowed checks if a path is in the allowed directories
func (b *BuiltinTools) isPathAllowed(path string) bool {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return false
	}

	for _, allowed := range b.allowedPaths {
		allowedAbs, err := filepath.Abs(allowed)
		if err != nil {
			continue
		}

		if strings.HasPrefix(absPath, allowedAbs) {
			return true
		}
	}

	return false
}

// SearchResult represents a single search result
type SearchResult struct {
	Title       string `json:"title"`
	URL         string `json:"url"`
	Description string `json:"description"`
}

// webSearch performs a web search using DuckDuckGo
// Note: This is deprecated - use WebResearchAgent instead for better results
func (b *BuiltinTools) webSearch(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	query, ok := params["query"].(string)
	if !ok || query == "" {
		return nil, fmt.Errorf("query parameter required")
	}

	count := 5
	if countStr, ok := params["count"].(string); ok && countStr != "" {
		if _, err := fmt.Sscanf(countStr, "%d", &count); err == nil {
			if count > 10 {
				count = 10
			}
			if count < 1 {
				count = 1
			}
		}
	}

	// Use DuckDuckGo HTML scraping
	results, err := b.duckDuckGoSearch(ctx, query, count)
	if err != nil {
		return nil, fmt.Errorf("web search failed: %w", err)
	}

	return b.formatSearchResults(results), nil
}

// duckDuckGoSearch performs a search using DuckDuckGo HTML
func (b *BuiltinTools) duckDuckGoSearch(ctx context.Context, query string, count int) ([]SearchResult, error) {
	// Use DuckDuckGo HTML version
	searchURL := fmt.Sprintf("https://html.duckduckgo.com/html/?q=%s", url.QueryEscape(query))

	req, err := http.NewRequestWithContext(ctx, "GET", searchURL, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; mDW/1.0)")

	resp, err := b.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return b.parseDuckDuckGoResults(string(body), count), nil
}

// parseDuckDuckGoResults extracts search results from DuckDuckGo HTML
func (b *BuiltinTools) parseDuckDuckGoResults(html string, count int) []SearchResult {
	var results []SearchResult

	// Pattern to match result blocks
	// DuckDuckGo HTML results are in <a class="result__a" href="...">title</a>
	// and <a class="result__snippet">description</a>
	resultPattern := regexp.MustCompile(`<a[^>]*class="result__a"[^>]*href="([^"]*)"[^>]*>([^<]*)</a>`)
	snippetPattern := regexp.MustCompile(`<a[^>]*class="result__snippet"[^>]*>([^<]*)</a>`)

	resultMatches := resultPattern.FindAllStringSubmatch(html, count*2)
	snippetMatches := snippetPattern.FindAllStringSubmatch(html, count*2)

	for i := 0; i < len(resultMatches) && len(results) < count; i++ {
		if len(resultMatches[i]) >= 3 {
			result := SearchResult{
				URL:   resultMatches[i][1],
				Title: strings.TrimSpace(resultMatches[i][2]),
			}

			// Try to find matching snippet
			if i < len(snippetMatches) && len(snippetMatches[i]) >= 2 {
				result.Description = strings.TrimSpace(snippetMatches[i][1])
			}

			// Skip empty or tracking URLs
			if result.URL != "" && !strings.Contains(result.URL, "duckduckgo.com") {
				results = append(results, result)
			}
		}
	}

	// If regex parsing didn't work well, try simpler extraction
	if len(results) == 0 {
		results = b.simpleDDGParsing(html, count)
	}

	return results
}

// simpleDDGParsing is a fallback parser for DuckDuckGo results
func (b *BuiltinTools) simpleDDGParsing(html string, count int) []SearchResult {
	var results []SearchResult

	// Look for result__url class which contains the actual URLs
	urlPattern := regexp.MustCompile(`<a[^>]*class="[^"]*result__url[^"]*"[^>]*href="([^"]*)"[^>]*>([^<]*)</a>`)
	matches := urlPattern.FindAllStringSubmatch(html, count*2)

	for _, match := range matches {
		if len(match) >= 3 && len(results) < count {
			// Extract actual URL from DuckDuckGo redirect
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
					Description: "Suchergebnis von DuckDuckGo",
				})
			}
		}
	}

	return results
}

// formatSearchResults formats search results for display
func (b *BuiltinTools) formatSearchResults(results []SearchResult) string {
	if len(results) == 0 {
		return "Keine Suchergebnisse gefunden."
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Gefunden: %d Ergebnisse\n\n", len(results)))

	for i, r := range results {
		sb.WriteString(fmt.Sprintf("%d. %s\n", i+1, r.Title))
		sb.WriteString(fmt.Sprintf("   URL: %s\n", r.URL))
		if r.Description != "" {
			sb.WriteString(fmt.Sprintf("   %s\n", r.Description))
		}
		sb.WriteString("\n")
	}

	return sb.String()
}

// fetchWebpage fetches and extracts text content from a webpage
func (b *BuiltinTools) fetchWebpage(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	urlStr, ok := params["url"].(string)
	if !ok || urlStr == "" {
		return nil, fmt.Errorf("url parameter required")
	}

	// Validate URL
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

	resp, err := b.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch URL: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("server returned %d", resp.StatusCode)
	}

	// Read and limit content
	body, err := io.ReadAll(io.LimitReader(resp.Body, 100000)) // 100KB limit
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Extract text content from HTML
	text := b.extractTextFromHTML(string(body))

	// Limit text length
	if len(text) > 50000 {
		text = text[:50000] + "\n... (gekürzt)"
	}

	return map[string]interface{}{
		"url":     urlStr,
		"title":   b.extractTitle(string(body)),
		"content": text,
	}, nil
}

// extractTextFromHTML removes HTML tags and extracts readable text
func (b *BuiltinTools) extractTextFromHTML(html string) string {
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

// extractTitle extracts the title from HTML
func (b *BuiltinTools) extractTitle(html string) string {
	titlePattern := regexp.MustCompile(`(?i)<title[^>]*>([^<]*)</title>`)
	matches := titlePattern.FindStringSubmatch(html)
	if len(matches) >= 2 {
		return strings.TrimSpace(matches[1])
	}
	return ""
}
