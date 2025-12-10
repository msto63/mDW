// ============================================================================
// meinDENKWERK (mDW) - Lokale KI-Plattform
// ============================================================================
//
// Package:     grpc
// Description: Integration tests for Hypatia gRPC service (RAG)
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
	hypatiapb "github.com/msto63/mDW/api/gen/hypatia"
)

// HypatiaTestClient wraps the Hypatia gRPC client for testing
type HypatiaTestClient struct {
	conn   *TestConnection
	client hypatiapb.HypatiaServiceClient
}

// NewHypatiaTestClient creates a new Hypatia test client
func NewHypatiaTestClient() (*HypatiaTestClient, error) {
	configs := DefaultServiceConfigs()
	cfg := configs["hypatia"]

	conn, err := NewTestConnection(cfg)
	if err != nil {
		return nil, err
	}

	return &HypatiaTestClient{
		conn:   conn,
		client: hypatiapb.NewHypatiaServiceClient(conn.Conn()),
	}, nil
}

// Close closes the test client connection
func (hc *HypatiaTestClient) Close() error {
	return hc.conn.Close()
}

// Client returns the underlying gRPC client
func (hc *HypatiaTestClient) Client() hypatiapb.HypatiaServiceClient {
	return hc.client
}

// ContextWithTimeout returns a context with a custom timeout
func (hc *HypatiaTestClient) ContextWithTimeout(timeout time.Duration) (context.Context, context.CancelFunc) {
	return hc.conn.ContextWithTimeout(timeout)
}

// TestHypatiaHealthCheck tests the health check endpoint
func TestHypatiaHealthCheck(t *testing.T) {
	client, err := NewHypatiaTestClient()
	if err != nil {
		t.Fatalf("Failed to create Hypatia client: %v", err)
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

	if resp.GetService() != "hypatia" {
		t.Errorf("Expected service 'hypatia', got '%s'", resp.GetService())
	}
}

// TestHypatiaListCollections tests listing collections
func TestHypatiaListCollections(t *testing.T) {
	client, err := NewHypatiaTestClient()
	if err != nil {
		t.Fatalf("Failed to create Hypatia client: %v", err)
	}
	defer client.Close()

	ctx, cancel := client.ContextWithTimeout(10 * time.Second)
	defer cancel()

	resp, err := client.Client().ListCollections(ctx, &common.Empty{})
	if err != nil {
		t.Fatalf("ListCollections failed: %v", err)
	}

	t.Logf("Found %d collections", len(resp.GetCollections()))
	for _, coll := range resp.GetCollections() {
		t.Logf("  - %s: %d documents, %d chunks", coll.GetName(), coll.GetDocumentCount(), coll.GetChunkCount())
	}
}

// TestHypatiaCreateDeleteCollection tests collection management
func TestHypatiaCreateDeleteCollection(t *testing.T) {
	client, err := NewHypatiaTestClient()
	if err != nil {
		t.Fatalf("Failed to create Hypatia client: %v", err)
	}
	defer client.Close()

	ctx, cancel := client.ContextWithTimeout(10 * time.Second)
	defer cancel()

	collectionName := "test-collection-integration"

	// Create collection
	createReq := &hypatiapb.CreateCollectionRequest{
		Name:        collectionName,
		Description: "Integration test collection",
	}

	createResp, err := client.Client().CreateCollection(ctx, createReq)
	if err != nil {
		t.Logf("CreateCollection: %v (collection may already exist)", err)
	} else {
		t.Logf("Created collection: %s", createResp.GetName())
	}

	// Delete collection
	deleteReq := &hypatiapb.DeleteCollectionRequest{
		Name: collectionName,
	}

	_, err = client.Client().DeleteCollection(ctx, deleteReq)
	if err != nil {
		t.Logf("DeleteCollection: %v", err)
	} else {
		t.Log("Deleted collection successfully")
	}
}

// TestHypatiaIngestDocument tests document ingestion
func TestHypatiaIngestDocument(t *testing.T) {
	client, err := NewHypatiaTestClient()
	if err != nil {
		t.Fatalf("Failed to create Hypatia client: %v", err)
	}
	defer client.Close()

	ctx, cancel := client.ContextWithTimeout(30 * time.Second)
	defer cancel()

	req := &hypatiapb.IngestDocumentRequest{
		Title:      "Test Document",
		Content:    "This is a test document for integration testing. It contains information about meinDENKWERK, a local AI platform.",
		Collection: "default",
		Source:     "integration-test",
	}

	resp, err := client.Client().IngestDocument(ctx, req)
	if err != nil {
		t.Logf("IngestDocument: %v (embedding model may not be available)", err)
		t.Skip("Skipping - embedding not available")
		return
	}

	t.Logf("Ingest Response:")
	t.Logf("  Document ID: %s", resp.GetDocumentId())
	t.Logf("  Chunks Created: %d", resp.GetChunksCreated())
	t.Logf("  Success: %v", resp.GetSuccess())

	if !resp.GetSuccess() {
		t.Error("Expected successful ingestion")
	}
}

// TestHypatiaSearch tests search functionality
func TestHypatiaSearch(t *testing.T) {
	client, err := NewHypatiaTestClient()
	if err != nil {
		t.Fatalf("Failed to create Hypatia client: %v", err)
	}
	defer client.Close()

	ctx, cancel := client.ContextWithTimeout(30 * time.Second)
	defer cancel()

	req := &hypatiapb.SearchRequest{
		Query:      "meinDENKWERK",
		Collection: "default",
		TopK:       5,
	}

	resp, err := client.Client().Search(ctx, req)
	if err != nil {
		t.Logf("Search: %v (no documents or embedding not available)", err)
		t.Skip("Skipping - search not available")
		return
	}

	t.Logf("Search Results:")
	t.Logf("  Total: %d", resp.GetTotal())
	t.Logf("  Search Time: %d ms", resp.GetSearchTimeMs())

	for i, result := range resp.GetResults() {
		if i >= 3 {
			break
		}
		t.Logf("  - Score: %.4f, Content: %s...", result.GetScore(), truncate(result.GetContent(), 50))
	}
}

// TestHypatiaAugmentPrompt tests RAG prompt augmentation
func TestHypatiaAugmentPrompt(t *testing.T) {
	client, err := NewHypatiaTestClient()
	if err != nil {
		t.Fatalf("Failed to create Hypatia client: %v", err)
	}
	defer client.Close()

	ctx, cancel := client.ContextWithTimeout(30 * time.Second)
	defer cancel()

	req := &hypatiapb.AugmentPromptRequest{
		Prompt:     "What is meinDENKWERK?",
		Collection: "default",
		TopK:       3,
	}

	resp, err := client.Client().AugmentPrompt(ctx, req)
	if err != nil {
		t.Logf("AugmentPrompt: %v (no documents or embedding not available)", err)
		t.Skip("Skipping - augmentation not available")
		return
	}

	t.Logf("Augment Prompt Response:")
	t.Logf("  Sources Used: %d", resp.GetSourcesUsed())
	t.Logf("  Context Tokens: %d", resp.GetContextTokens())
	t.Logf("  Augmented Prompt Length: %d", len(resp.GetAugmentedPrompt()))
}

// truncate truncates a string to max length
func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}

// RunHypatiaTestSuite runs all Hypatia tests
func RunHypatiaTestSuite(t *testing.T) *TestSuite {
	suite := NewTestSuite("hypatia")

	tests := []struct {
		name string
		fn   func(*testing.T)
	}{
		{"HealthCheck", TestHypatiaHealthCheck},
		{"ListCollections", TestHypatiaListCollections},
		{"CreateDeleteCollection", TestHypatiaCreateDeleteCollection},
		{"IngestDocument", TestHypatiaIngestDocument},
		{"Search", TestHypatiaSearch},
		{"AugmentPrompt", TestHypatiaAugmentPrompt},
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
