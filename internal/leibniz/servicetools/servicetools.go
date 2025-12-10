// ============================================================================
// meinDENKWERK (mDW) - Lokale KI-Plattform
// ============================================================================
//
// Package:     servicetools
// Description: Service integration tools for Leibniz agent (RAG, NLP, etc.)
// Author:      Mike Stoffels with Claude
// Created:     2025-12-06
// License:     MIT
// ============================================================================

package servicetools

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	babbagepb "github.com/msto63/mDW/api/gen/babbage"
	hypatiapb "github.com/msto63/mDW/api/gen/hypatia"
	"github.com/msto63/mDW/internal/leibniz/agent"
	"github.com/msto63/mDW/pkg/core/logging"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// ServiceTools provides tools that call other mDW services
type ServiceTools struct {
	hypatiaAddr string
	babbageAddr string
	logger      *logging.Logger
}

// Config holds service tools configuration
type Config struct {
	HypatiaAddr string
	BabbageAddr string
}

// DefaultConfig returns default configuration
func DefaultConfig() Config {
	return Config{
		HypatiaAddr: "localhost:9220",
		BabbageAddr: "localhost:9150",
	}
}

// New creates a new ServiceTools instance
func New(cfg Config) *ServiceTools {
	return &ServiceTools{
		hypatiaAddr: cfg.HypatiaAddr,
		babbageAddr: cfg.BabbageAddr,
		logger:      logging.New("leibniz-servicetools"),
	}
}

// RegisterAll registers all service tools with the agent
func (st *ServiceTools) RegisterAll(ag *agent.Agent) {
	// RAG Tools (Hypatia)
	st.registerRAGSearchTool(ag)
	st.registerRAGAugmentTool(ag)

	// NLP Tools (Babbage)
	st.registerSummarizeTool(ag)
	st.registerAnalyzeSentimentTool(ag)
	st.registerExtractKeywordsTool(ag)
	st.registerExtractEntitiesTool(ag)
	st.registerDetectLanguageTool(ag)

	st.logger.Info("Service tools registered",
		"rag_tools", 2,
		"nlp_tools", 5,
	)
}

// dialHypatia creates a connection to Hypatia
func (st *ServiceTools) dialHypatia(ctx context.Context) (*grpc.ClientConn, hypatiapb.HypatiaServiceClient, error) {
	dialCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	conn, err := grpc.DialContext(dialCtx, st.hypatiaAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to connect to Hypatia: %w", err)
	}

	return conn, hypatiapb.NewHypatiaServiceClient(conn), nil
}

// dialBabbage creates a connection to Babbage
func (st *ServiceTools) dialBabbage(ctx context.Context) (*grpc.ClientConn, babbagepb.BabbageServiceClient, error) {
	dialCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	conn, err := grpc.DialContext(dialCtx, st.babbageAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to connect to Babbage: %w", err)
	}

	return conn, babbagepb.NewBabbageServiceClient(conn), nil
}

// ============================================================================
// RAG Tools (Hypatia)
// ============================================================================

// registerRAGSearchTool registers the knowledge base search tool
func (st *ServiceTools) registerRAGSearchTool(ag *agent.Agent) {
	ag.RegisterTool(&agent.Tool{
		Name:        "knowledge_search",
		Description: "Durchsucht die Wissensdatenbank nach relevanten Informationen zu einer Anfrage",
		Parameters: map[string]agent.ParameterDef{
			"query": {
				Type:        "string",
				Description: "Die Suchanfrage",
				Required:    true,
			},
			"collection": {
				Type:        "string",
				Description: "Name der zu durchsuchenden Sammlung (optional, default: 'default')",
				Required:    false,
			},
			"top_k": {
				Type:        "number",
				Description: "Anzahl der zurückzugebenden Ergebnisse (optional, default: 5)",
				Required:    false,
			},
		},
		Handler: st.handleRAGSearch,
	})
}

func (st *ServiceTools) handleRAGSearch(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	query, ok := params["query"].(string)
	if !ok || query == "" {
		return nil, fmt.Errorf("query parameter required")
	}

	collection := "default"
	if c, ok := params["collection"].(string); ok && c != "" {
		collection = c
	}

	topK := int32(5)
	if k, ok := params["top_k"].(float64); ok {
		topK = int32(k)
	}

	conn, client, err := st.dialHypatia(ctx)
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	resp, err := client.Search(ctx, &hypatiapb.SearchRequest{
		Query:      query,
		Collection: collection,
		TopK:       topK,
		MinScore:   0.5,
	})
	if err != nil {
		return nil, fmt.Errorf("search failed: %w", err)
	}

	// Format results
	if len(resp.Results) == 0 {
		return "Keine relevanten Dokumente gefunden.", nil
	}

	var results []map[string]interface{}
	for _, r := range resp.Results {
		results = append(results, map[string]interface{}{
			"content":     r.Content,
			"score":       r.Score,
			"document_id": r.DocumentId,
		})
	}

	resultJSON, _ := json.MarshalIndent(results, "", "  ")
	return fmt.Sprintf("Gefundene Dokumente (%d):\n%s", len(results), string(resultJSON)), nil
}

// registerRAGAugmentTool registers the prompt augmentation tool
func (st *ServiceTools) registerRAGAugmentTool(ag *agent.Agent) {
	ag.RegisterTool(&agent.Tool{
		Name:        "augment_with_knowledge",
		Description: "Erweitert eine Anfrage mit relevantem Wissen aus der Wissensdatenbank",
		Parameters: map[string]agent.ParameterDef{
			"prompt": {
				Type:        "string",
				Description: "Die zu erweiternde Anfrage",
				Required:    true,
			},
			"collection": {
				Type:        "string",
				Description: "Name der Sammlung (optional, default: 'default')",
				Required:    false,
			},
		},
		Handler: st.handleRAGAugment,
	})
}

func (st *ServiceTools) handleRAGAugment(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	prompt, ok := params["prompt"].(string)
	if !ok || prompt == "" {
		return nil, fmt.Errorf("prompt parameter required")
	}

	collection := "default"
	if c, ok := params["collection"].(string); ok && c != "" {
		collection = c
	}

	conn, client, err := st.dialHypatia(ctx)
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	resp, err := client.AugmentPrompt(ctx, &hypatiapb.AugmentPromptRequest{
		Prompt:     prompt,
		Collection: collection,
		TopK:       3,
	})
	if err != nil {
		return nil, fmt.Errorf("augmentation failed: %w", err)
	}

	if resp.SourcesUsed == 0 {
		return prompt, nil // Return original prompt if no sources found
	}

	return resp.AugmentedPrompt, nil
}

// ============================================================================
// NLP Tools (Babbage)
// ============================================================================

// registerSummarizeTool registers the text summarization tool
func (st *ServiceTools) registerSummarizeTool(ag *agent.Agent) {
	ag.RegisterTool(&agent.Tool{
		Name:        "summarize",
		Description: "Fasst einen Text zusammen",
		Parameters: map[string]agent.ParameterDef{
			"text": {
				Type:        "string",
				Description: "Der zu zusammenfassende Text",
				Required:    true,
			},
			"max_length": {
				Type:        "number",
				Description: "Maximale Länge der Zusammenfassung in Wörtern (optional)",
				Required:    false,
			},
		},
		Handler: st.handleSummarize,
	})
}

func (st *ServiceTools) handleSummarize(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	text, ok := params["text"].(string)
	if !ok || text == "" {
		return nil, fmt.Errorf("text parameter required")
	}

	maxLength := int32(100)
	if m, ok := params["max_length"].(float64); ok {
		maxLength = int32(m)
	}

	conn, client, err := st.dialBabbage(ctx)
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	resp, err := client.Summarize(ctx, &babbagepb.SummarizeRequest{
		Text:      text,
		MaxLength: maxLength,
	})
	if err != nil {
		return nil, fmt.Errorf("summarization failed: %w", err)
	}

	return resp.Summary, nil
}

// registerAnalyzeSentimentTool registers the sentiment analysis tool
func (st *ServiceTools) registerAnalyzeSentimentTool(ag *agent.Agent) {
	ag.RegisterTool(&agent.Tool{
		Name:        "analyze_sentiment",
		Description: "Analysiert die Stimmung/Sentiment eines Textes (positiv, negativ, neutral)",
		Parameters: map[string]agent.ParameterDef{
			"text": {
				Type:        "string",
				Description: "Der zu analysierende Text",
				Required:    true,
			},
		},
		Handler: st.handleAnalyzeSentiment,
	})
}

func (st *ServiceTools) handleAnalyzeSentiment(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	text, ok := params["text"].(string)
	if !ok || text == "" {
		return nil, fmt.Errorf("text parameter required")
	}

	conn, client, err := st.dialBabbage(ctx)
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	resp, err := client.AnalyzeSentiment(ctx, &babbagepb.SentimentRequest{
		Text: text,
	})
	if err != nil {
		return nil, fmt.Errorf("sentiment analysis failed: %w", err)
	}

	result := resp.Result
	sentimentStr := "unbekannt"
	switch result.Sentiment {
	case babbagepb.Sentiment_SENTIMENT_POSITIVE:
		sentimentStr = "positiv"
	case babbagepb.Sentiment_SENTIMENT_NEGATIVE:
		sentimentStr = "negativ"
	case babbagepb.Sentiment_SENTIMENT_NEUTRAL:
		sentimentStr = "neutral"
	case babbagepb.Sentiment_SENTIMENT_MIXED:
		sentimentStr = "gemischt"
	}

	return fmt.Sprintf("Stimmung: %s (Score: %.2f, Konfidenz: %.2f%%)",
		sentimentStr, result.Score, result.Confidence*100), nil
}

// registerExtractKeywordsTool registers the keyword extraction tool
func (st *ServiceTools) registerExtractKeywordsTool(ag *agent.Agent) {
	ag.RegisterTool(&agent.Tool{
		Name:        "extract_keywords",
		Description: "Extrahiert die wichtigsten Schlüsselwörter aus einem Text",
		Parameters: map[string]agent.ParameterDef{
			"text": {
				Type:        "string",
				Description: "Der zu analysierende Text",
				Required:    true,
			},
		},
		Handler: st.handleExtractKeywords,
	})
}

func (st *ServiceTools) handleExtractKeywords(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	text, ok := params["text"].(string)
	if !ok || text == "" {
		return nil, fmt.Errorf("text parameter required")
	}

	conn, client, err := st.dialBabbage(ctx)
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	resp, err := client.ExtractKeywords(ctx, &babbagepb.ExtractRequest{
		Text: text,
	})
	if err != nil {
		return nil, fmt.Errorf("keyword extraction failed: %w", err)
	}

	if len(resp.Keywords) == 0 {
		return "Keine Schlüsselwörter gefunden.", nil
	}

	var keywords []string
	for _, kw := range resp.Keywords {
		keywords = append(keywords, fmt.Sprintf("%s (%.2f)", kw.Word, kw.Score))
	}

	return fmt.Sprintf("Schlüsselwörter: %v", keywords), nil
}

// registerExtractEntitiesTool registers the entity extraction tool
func (st *ServiceTools) registerExtractEntitiesTool(ag *agent.Agent) {
	ag.RegisterTool(&agent.Tool{
		Name:        "extract_entities",
		Description: "Extrahiert benannte Entitäten (Personen, Orte, Organisationen, etc.) aus einem Text",
		Parameters: map[string]agent.ParameterDef{
			"text": {
				Type:        "string",
				Description: "Der zu analysierende Text",
				Required:    true,
			},
		},
		Handler: st.handleExtractEntities,
	})
}

func (st *ServiceTools) handleExtractEntities(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	text, ok := params["text"].(string)
	if !ok || text == "" {
		return nil, fmt.Errorf("text parameter required")
	}

	conn, client, err := st.dialBabbage(ctx)
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	resp, err := client.ExtractEntities(ctx, &babbagepb.ExtractRequest{
		Text: text,
	})
	if err != nil {
		return nil, fmt.Errorf("entity extraction failed: %w", err)
	}

	if len(resp.Entities) == 0 {
		return "Keine Entitäten gefunden.", nil
	}

	var entities []map[string]interface{}
	for _, e := range resp.Entities {
		entities = append(entities, map[string]interface{}{
			"text": e.Text,
			"type": e.Type.String(),
		})
	}

	resultJSON, _ := json.MarshalIndent(entities, "", "  ")
	return fmt.Sprintf("Gefundene Entitäten:\n%s", string(resultJSON)), nil
}

// registerDetectLanguageTool registers the language detection tool
func (st *ServiceTools) registerDetectLanguageTool(ag *agent.Agent) {
	ag.RegisterTool(&agent.Tool{
		Name:        "detect_language",
		Description: "Erkennt die Sprache eines Textes",
		Parameters: map[string]agent.ParameterDef{
			"text": {
				Type:        "string",
				Description: "Der zu analysierende Text",
				Required:    true,
			},
		},
		Handler: st.handleDetectLanguage,
	})
}

func (st *ServiceTools) handleDetectLanguage(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	text, ok := params["text"].(string)
	if !ok || text == "" {
		return nil, fmt.Errorf("text parameter required")
	}

	conn, client, err := st.dialBabbage(ctx)
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	resp, err := client.DetectLanguage(ctx, &babbagepb.DetectLanguageRequest{
		Text: text,
	})
	if err != nil {
		return nil, fmt.Errorf("language detection failed: %w", err)
	}

	return fmt.Sprintf("Erkannte Sprache: %s (Konfidenz: %.2f%%)",
		resp.Language, resp.Confidence*100), nil
}
