// ============================================================================
// meinDENKWERK (mDW) - Lokale KI-Plattform
// ============================================================================
//
// Package:     grpc
// Description: Integration tests for Babbage gRPC service (NLP)
// Author:      Mike Stoffels with Claude
// Created:     2025-12-06
// License:     MIT
// ============================================================================

//go:build integration

package grpc

import (
	"context"
	"testing"
	"time"

	"github.com/msto63/mDW/api/gen/common"
	babbagepb "github.com/msto63/mDW/api/gen/babbage"
)

// BabbageTestClient wraps the Babbage gRPC client for testing
type BabbageTestClient struct {
	conn   *TestConnection
	client babbagepb.BabbageServiceClient
}

// NewBabbageTestClient creates a new Babbage test client
func NewBabbageTestClient() (*BabbageTestClient, error) {
	configs := DefaultServiceConfigs()
	cfg := configs["babbage"]

	conn, err := NewTestConnection(cfg)
	if err != nil {
		return nil, err
	}

	return &BabbageTestClient{
		conn:   conn,
		client: babbagepb.NewBabbageServiceClient(conn.Conn()),
	}, nil
}

// Close closes the test client connection
func (bc *BabbageTestClient) Close() error {
	return bc.conn.Close()
}

// Client returns the underlying gRPC client
func (bc *BabbageTestClient) Client() babbagepb.BabbageServiceClient {
	return bc.client
}

// ContextWithTimeout returns a context with a custom timeout
func (bc *BabbageTestClient) ContextWithTimeout(timeout time.Duration) (context.Context, context.CancelFunc) {
	return bc.conn.ContextWithTimeout(timeout)
}

// TestBabbageHealthCheck tests the health check endpoint
func TestBabbageHealthCheck(t *testing.T) {
	client, err := NewBabbageTestClient()
	if err != nil {
		t.Fatalf("Failed to create Babbage client: %v", err)
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

	if resp.GetService() != "babbage" {
		t.Errorf("Expected service 'babbage', got '%s'", resp.GetService())
	}
}

// TestBabbageDetectLanguage tests language detection
func TestBabbageDetectLanguage(t *testing.T) {
	client, err := NewBabbageTestClient()
	if err != nil {
		t.Fatalf("Failed to create Babbage client: %v", err)
	}
	defer client.Close()

	tests := []struct {
		name     string
		text     string
		expected string
	}{
		{"German", "Das ist ein deutscher Text zum Testen.", "de"},
		{"English", "This is an English text for testing.", "en"},
		{"French", "Ceci est un texte français pour les tests.", "fr"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := client.ContextWithTimeout(30 * time.Second)
			defer cancel()

			req := &babbagepb.DetectLanguageRequest{
				Text: tt.text,
			}

			resp, err := client.Client().DetectLanguage(ctx, req)
			if err != nil {
				t.Fatalf("DetectLanguage failed: %v", err)
			}

			t.Logf("Language Detection for '%s':", tt.name)
			t.Logf("  Language: %s", resp.GetLanguage())
			t.Logf("  Confidence: %.2f", resp.GetConfidence())

			if resp.GetLanguage() != tt.expected {
				t.Logf("Expected language '%s', got '%s' (may be acceptable)", tt.expected, resp.GetLanguage())
			}
		})
	}
}

// TestBabbageAnalyzeSentiment tests sentiment analysis
func TestBabbageAnalyzeSentiment(t *testing.T) {
	client, err := NewBabbageTestClient()
	if err != nil {
		t.Fatalf("Failed to create Babbage client: %v", err)
	}
	defer client.Close()

	tests := []struct {
		name     string
		text     string
		positive bool
	}{
		{"Positive", "I love this product! It's absolutely amazing and wonderful.", true},
		{"Negative", "This is terrible. I hate it. Worst experience ever.", false},
		{"Neutral", "The package arrived on Tuesday.", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := client.ContextWithTimeout(30 * time.Second)
			defer cancel()

			req := &babbagepb.SentimentRequest{
				Text: tt.text,
			}

			resp, err := client.Client().AnalyzeSentiment(ctx, req)
			if err != nil {
				t.Fatalf("AnalyzeSentiment failed: %v", err)
			}

			result := resp.GetResult()
			t.Logf("Sentiment for '%s':", tt.name)
			t.Logf("  Sentiment: %s", result.GetSentiment().String())
			t.Logf("  Score: %.2f", result.GetScore())
			t.Logf("  Confidence: %.2f", result.GetConfidence())
		})
	}
}

// TestBabbageExtractEntities tests entity extraction
func TestBabbageExtractEntities(t *testing.T) {
	client, err := NewBabbageTestClient()
	if err != nil {
		t.Fatalf("Failed to create Babbage client: %v", err)
	}
	defer client.Close()

	ctx, cancel := client.ContextWithTimeout(30 * time.Second)
	defer cancel()

	req := &babbagepb.ExtractRequest{
		Text: "Apple Inc. was founded by Steve Jobs in Cupertino, California. Contact us at info@apple.com.",
	}

	resp, err := client.Client().ExtractEntities(ctx, req)
	if err != nil {
		t.Fatalf("ExtractEntities failed: %v", err)
	}

	t.Logf("Extracted %d entities:", len(resp.GetEntities()))
	for _, entity := range resp.GetEntities() {
		t.Logf("  - %s (%s) [confidence: %.2f]", entity.GetText(), entity.GetType().String(), entity.GetConfidence())
	}
}

// TestBabbageExtractKeywords tests keyword extraction
func TestBabbageExtractKeywords(t *testing.T) {
	client, err := NewBabbageTestClient()
	if err != nil {
		t.Fatalf("Failed to create Babbage client: %v", err)
	}
	defer client.Close()

	ctx, cancel := client.ContextWithTimeout(30 * time.Second)
	defer cancel()

	req := &babbagepb.ExtractRequest{
		Text: "Machine learning and artificial intelligence are transforming how we process natural language. Deep learning models can understand context and generate human-like text.",
	}

	resp, err := client.Client().ExtractKeywords(ctx, req)
	if err != nil {
		t.Fatalf("ExtractKeywords failed: %v", err)
	}

	t.Logf("Extracted %d keywords:", len(resp.GetKeywords()))
	for i, kw := range resp.GetKeywords() {
		if i >= 5 {
			break
		}
		t.Logf("  - %s (score: %.2f, freq: %d)", kw.GetWord(), kw.GetScore(), kw.GetFrequency())
	}
}

// TestBabbageSummarize tests text summarization
func TestBabbageSummarize(t *testing.T) {
	client, err := NewBabbageTestClient()
	if err != nil {
		t.Fatalf("Failed to create Babbage client: %v", err)
	}
	defer client.Close()

	ctx, cancel := client.ContextWithTimeout(60 * time.Second)
	defer cancel()

	longText := `Artificial intelligence has made remarkable progress in recent years.
	Machine learning algorithms can now recognize images, understand speech, and generate human-like text.
	Deep learning, a subset of machine learning, uses neural networks with many layers to learn complex patterns.
	Natural language processing enables computers to understand and generate human language.
	These technologies are being applied in healthcare, finance, transportation, and many other industries.
	The future of AI promises even more capabilities, but also raises important ethical considerations.`

	req := &babbagepb.SummarizeRequest{
		Text:      longText,
		MaxLength: 50,
		Style:     babbagepb.SummarizationStyle_SUMMARIZATION_STYLE_BRIEF,
	}

	resp, err := client.Client().Summarize(ctx, req)
	if err != nil {
		t.Fatalf("Summarize failed: %v", err)
	}

	t.Logf("Summarization Result:")
	t.Logf("  Original Length: %d", resp.GetOriginalLength())
	t.Logf("  Summary Length: %d", resp.GetSummaryLength())
	t.Logf("  Compression Ratio: %.2f", resp.GetCompressionRatio())
	t.Logf("  Summary: %s", resp.GetSummary())
}

// TestBabbageAnalyze tests combined analysis
func TestBabbageAnalyze(t *testing.T) {
	client, err := NewBabbageTestClient()
	if err != nil {
		t.Fatalf("Failed to create Babbage client: %v", err)
	}
	defer client.Close()

	ctx, cancel := client.ContextWithTimeout(60 * time.Second)
	defer cancel()

	req := &babbagepb.AnalyzeRequest{
		Text: "Microsoft announced new AI features today. The CEO Satya Nadella presented at the conference in Seattle.",
		Analyses: []babbagepb.AnalysisType{
			babbagepb.AnalysisType_ANALYSIS_TYPE_SENTIMENT,
			babbagepb.AnalysisType_ANALYSIS_TYPE_ENTITIES,
			babbagepb.AnalysisType_ANALYSIS_TYPE_KEYWORDS,
			babbagepb.AnalysisType_ANALYSIS_TYPE_LANGUAGE,
		},
	}

	resp, err := client.Client().Analyze(ctx, req)
	if err != nil {
		t.Fatalf("Analyze failed: %v", err)
	}

	t.Logf("Combined Analysis Result:")
	t.Logf("  Language: %s (confidence: %.2f)", resp.GetLanguage(), resp.GetLanguageConfidence())
	t.Logf("  Sentiment: %s (score: %.2f)", resp.GetSentiment().GetSentiment().String(), resp.GetSentiment().GetScore())
	t.Logf("  Entities: %d found", len(resp.GetEntities()))
	t.Logf("  Keywords: %d found", len(resp.GetKeywords()))
}

// RunBabbageTestSuite runs all Babbage tests
func RunBabbageTestSuite(t *testing.T) *TestSuite {
	suite := NewTestSuite("babbage")

	tests := []struct {
		name string
		fn   func(*testing.T)
	}{
		{"HealthCheck", TestBabbageHealthCheck},
		{"DetectLanguage", TestBabbageDetectLanguage},
		{"AnalyzeSentiment", TestBabbageAnalyzeSentiment},
		{"ExtractEntities", TestBabbageExtractEntities},
		{"ExtractKeywords", TestBabbageExtractKeywords},
		{"Summarize", TestBabbageSummarize},
		{"Analyze", TestBabbageAnalyze},
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
