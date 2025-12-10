// ============================================================================
// meinDENKWERK (mDW) - Lokale KI-Plattform
// ============================================================================
//
// Package:     websearch
// Description: Specialized Web Research Agent
// Author:      Mike Stoffels with Claude
// Created:     2025-12-10
// License:     MIT
// ============================================================================

package websearch

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/msto63/mDW/internal/leibniz/agent"
	"github.com/msto63/mDW/internal/leibniz/platon"
	"github.com/msto63/mDW/pkg/core/logging"
)

// WebResearchAgent is a specialized agent for web research tasks
type WebResearchAgent struct {
	searchClient  *WebSearchClient
	platonClient  *platon.Client
	pipelineID    string // Pipeline ID to use for processing
	enablePlaton  bool   // Whether to use Platon for pre/post processing
	logger        *logging.Logger
}

// AgentConfig holds configuration for the web research agent
type AgentConfig struct {
	SearXNGInstances []string // SearXNG instance URLs
	Timeout          time.Duration
}

// DefaultAgentConfig returns default configuration
func DefaultAgentConfig() AgentConfig {
	return AgentConfig{
		SearXNGInstances: DefaultSearXNGConfig().Instances,
		Timeout:          30 * time.Second,
	}
}

// NewWebResearchAgent creates a new web research agent
func NewWebResearchAgent(cfg AgentConfig) *WebResearchAgent {
	searchCfg := Config{
		SearXNGInstances: cfg.SearXNGInstances,
		Timeout:          cfg.Timeout,
	}

	return &WebResearchAgent{
		searchClient: NewWebSearchClient(searchCfg),
		logger:       logging.New("web-research-agent"),
	}
}

// GetAgentDefinition returns the agent definition for registration
func (a *WebResearchAgent) GetAgentDefinition() AgentDefinition {
	return AgentDefinition{
		ID:          "web-researcher",
		Name:        "Web-Recherche Agent",
		Description: "Spezialisierter Agent für Internet-Recherchen mit datenschutzfreundlichen Suchmaschinen",
		SystemPrompt: `Du bist ein spezialisierter Web-Recherche-Assistent.

DEINE AUFGABE:
Du führst strukturierte Internet-Recherchen durch und fasst die Ergebnisse verständlich zusammen.

VERFÜGBARE TOOLS:
- web_search: Durchsucht das Internet (SearXNG → DuckDuckGo Fallback)
- fetch_webpage: Lädt den Inhalt einer spezifischen URL
- search_news: Sucht nach aktuellen Nachrichten

RECHERCHE-STRATEGIE:
1. VERSTEHEN: Analysiere die Anfrage und identifiziere Schlüsselbegriffe
2. SUCHEN: Führe eine oder mehrere Suchanfragen durch
3. FILTERN: Wähle die relevantesten Ergebnisse aus
4. VERTIEFEN: Lade bei Bedarf einzelne Seiten für Details
5. ZUSAMMENFASSEN: Fasse die Erkenntnisse strukturiert zusammen

AUSGABEFORMAT:
- Beginne mit einer kurzen Zusammenfassung (2-3 Sätze)
- Liste die wichtigsten Erkenntnisse auf
- Gib IMMER die Quellen an mit URL
- Erwähne wenn Informationen widersprüchlich oder unsicher sind

WICHTIGE REGELN:
- Nutze mehrere Suchanfragen wenn nötig
- Prüfe die Aktualität der Informationen
- Sei kritisch gegenüber den Quellen
- Gib zu wenn du keine zuverlässigen Informationen findest
- Antworte auf Deutsch, es sei denn anders gewünscht`,
		Tools:    []string{"web_search", "fetch_webpage", "search_news"},
		MaxSteps: 8,
		Timeout:  120 * time.Second,
	}
}

// AgentDefinition represents the agent definition structure
type AgentDefinition struct {
	ID           string
	Name         string
	Description  string
	SystemPrompt string
	Tools        []string
	MaxSteps     int
	Timeout      time.Duration
}

// RegisterTools registers all web search tools with the given agent
func (a *WebResearchAgent) RegisterTools(ag *agent.Agent) {
	// Web search tool
	ag.RegisterTool(&agent.Tool{
		Name:        "web_search",
		Description: "Durchsucht das Internet nach Informationen. Nutzt datenschutzfreundliche Suchmaschinen (SearXNG mit DuckDuckGo als Fallback).",
		Parameters: map[string]agent.ParameterDef{
			"query": {Type: "string", Description: "Die Suchanfrage", Required: true},
			"count": {Type: "string", Description: "Anzahl der Ergebnisse (1-20, Standard: 5)", Required: false},
		},
		Handler: a.webSearchHandler,
	})

	// Fetch webpage tool
	ag.RegisterTool(&agent.Tool{
		Name:        "fetch_webpage",
		Description: "Lädt den Textinhalt einer Webseite. Nutze dies um Details von einem Suchergebnis zu erhalten.",
		Parameters: map[string]agent.ParameterDef{
			"url": {Type: "string", Description: "Die URL der Webseite", Required: true},
		},
		Handler: a.fetchWebpageHandler,
	})

	// News search tool
	ag.RegisterTool(&agent.Tool{
		Name:        "search_news",
		Description: "Sucht nach aktuellen Nachrichten zu einem Thema.",
		Parameters: map[string]agent.ParameterDef{
			"query":      {Type: "string", Description: "Das Nachrichtenthema", Required: true},
			"time_range": {Type: "string", Description: "Zeitraum: day, week, month (Standard: week)", Required: false},
		},
		Handler: a.searchNewsHandler,
	})

	a.logger.Info("Web research tools registered")
}

// webSearchHandler handles web search requests
func (a *WebResearchAgent) webSearchHandler(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	query, ok := params["query"].(string)
	if !ok || query == "" {
		return nil, fmt.Errorf("query parameter required")
	}

	count := 5
	if countStr, ok := params["count"].(string); ok && countStr != "" {
		fmt.Sscanf(countStr, "%d", &count)
	}

	resp, err := a.searchClient.Search(ctx, query, count)
	if err != nil {
		return nil, err
	}

	return a.formatSearchResponse(resp), nil
}

// fetchWebpageHandler handles webpage fetch requests
func (a *WebResearchAgent) fetchWebpageHandler(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	urlStr, ok := params["url"].(string)
	if !ok || urlStr == "" {
		return nil, fmt.Errorf("url parameter required")
	}

	content, err := a.searchClient.FetchWebpage(ctx, urlStr)
	if err != nil {
		return nil, err
	}

	formattedContent := a.formatWebpageContent(content)

	// Process through Platon for content filtering (PII, safety, etc.)
	processedContent, blocked, err := a.processWithPlaton(ctx, formattedContent, "post")
	if err != nil && blocked {
		return "Inhalt wurde aufgrund von Sicherheitsrichtlinien gefiltert.", nil
	}

	return processedContent, nil
}

// searchNewsHandler handles news search requests
func (a *WebResearchAgent) searchNewsHandler(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	query, ok := params["query"].(string)
	if !ok || query == "" {
		return nil, fmt.Errorf("query parameter required")
	}

	// Add "news" or "aktuell" to query for better news results
	newsQuery := query + " news aktuell"

	resp, err := a.searchClient.Search(ctx, newsQuery, 8)
	if err != nil {
		return nil, err
	}

	return a.formatSearchResponse(resp), nil
}

// formatSearchResponse formats search results for the agent
func (a *WebResearchAgent) formatSearchResponse(resp *SearchResponse) string {
	if resp == nil || len(resp.Results) == 0 {
		return "Keine Suchergebnisse gefunden."
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Suche: \"%s\"\n", resp.Query))
	sb.WriteString(fmt.Sprintf("Quelle: %s | Gefunden: %d Ergebnisse\n", resp.Source, len(resp.Results)))
	sb.WriteString(strings.Repeat("-", 50) + "\n\n")

	for i, r := range resp.Results {
		sb.WriteString(fmt.Sprintf("[%d] %s\n", i+1, r.Title))
		sb.WriteString(fmt.Sprintf("    URL: %s\n", r.URL))
		if r.Description != "" {
			sb.WriteString(fmt.Sprintf("    %s\n", r.Description))
		}
		if r.PublishedAt != "" {
			sb.WriteString(fmt.Sprintf("    Datum: %s\n", r.PublishedAt))
		}
		sb.WriteString("\n")
	}

	if len(resp.Suggestions) > 0 {
		sb.WriteString("Verwandte Suchanfragen: " + strings.Join(resp.Suggestions, ", ") + "\n")
	}

	return sb.String()
}

// formatWebpageContent formats webpage content for the agent
func (a *WebResearchAgent) formatWebpageContent(content *WebpageContent) string {
	if content == nil {
		return "Fehler: Kein Inhalt geladen."
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Webseite: %s\n", content.URL))
	if content.Title != "" {
		sb.WriteString(fmt.Sprintf("Titel: %s\n", content.Title))
	}
	sb.WriteString(strings.Repeat("-", 50) + "\n\n")
	sb.WriteString(content.Content)

	return sb.String()
}

// SearchClient returns the underlying search client for direct access
func (a *WebResearchAgent) SearchClient() *WebSearchClient {
	return a.searchClient
}

// AddSearXNGInstance adds a SearXNG instance (e.g., local instance)
func (a *WebResearchAgent) AddSearXNGInstance(url string) {
	a.searchClient.AddSearXNGInstance(url)
}

// SetPlatonClient sets the Platon client for pipeline processing
func (a *WebResearchAgent) SetPlatonClient(client *platon.Client, pipelineID string) {
	a.platonClient = client
	a.pipelineID = pipelineID
	a.enablePlaton = client != nil
	if a.enablePlaton {
		a.logger.Info("Platon integration enabled for web research", "pipeline_id", pipelineID)
	}
}

// processWithPlaton processes content through Platon pre-processing pipeline
func (a *WebResearchAgent) processWithPlaton(ctx context.Context, content string, phase string) (string, bool, error) {
	if !a.enablePlaton || a.platonClient == nil {
		return content, false, nil
	}

	req := &platon.ProcessRequest{
		RequestID:  fmt.Sprintf("websearch-%d", time.Now().UnixNano()),
		PipelineID: a.pipelineID,
		Prompt:     content,
		Metadata: map[string]string{
			"source": "web-research-agent",
			"phase":  phase,
		},
	}

	var resp *platon.ProcessResponse
	var err error

	if phase == "pre" {
		resp, err = a.platonClient.ProcessPre(ctx, req)
	} else {
		req.Response = content
		resp, err = a.platonClient.ProcessPost(ctx, req)
	}

	if err != nil {
		a.logger.Warn("Platon processing failed, continuing without filtering",
			"error", err, "phase", phase)
		return content, false, nil
	}

	if resp.Blocked {
		a.logger.Info("Content blocked by Platon",
			"reason", resp.BlockReason, "phase", phase)
		return "", true, fmt.Errorf("content blocked: %s", resp.BlockReason)
	}

	if resp.Modified {
		a.logger.Debug("Content modified by Platon", "phase", phase)
		if phase == "pre" {
			return resp.ProcessedPrompt, false, nil
		}
		return resp.ProcessedResponse, false, nil
	}

	return content, false, nil
}
