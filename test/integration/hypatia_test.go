package integration

import (
	"fmt"
	"testing"
	"time"

	commonpb "github.com/msto63/mDW/api/gen/common"
	hypatiapb "github.com/msto63/mDW/api/gen/hypatia"
)

func TestHypatia_HealthCheck(t *testing.T) {
	cfg := getTestConfig()
	skipIfServiceUnavailable(t, cfg.HypatiaAddr, "Hypatia")
	logTestStart(t, "Hypatia", "HealthCheck")

	conn := dialGRPC(t, cfg.HypatiaAddr)
	client := hypatiapb.NewHypatiaServiceClient(conn)

	ctx, cancel := testContext(t, 10*time.Second)
	defer cancel()

	resp, err := client.HealthCheck(ctx, &commonpb.HealthCheckRequest{})
	requireNoError(t, err, "HealthCheck failed")
	requireEqual(t, "healthy", resp.Status, "Service should be healthy")

	t.Logf("Hypatia health: status=%s version=%s", resp.Status, resp.Version)
}

func TestHypatia_CollectionLifecycle(t *testing.T) {
	cfg := getTestConfig()
	skipIfServiceUnavailable(t, cfg.HypatiaAddr, "Hypatia")
	logTestStart(t, "Hypatia", "CollectionLifecycle")

	conn := dialGRPC(t, cfg.HypatiaAddr)
	client := hypatiapb.NewHypatiaServiceClient(conn)

	ctx, cancel := testContext(t, 30*time.Second)
	defer cancel()

	collectionName := fmt.Sprintf("test_collection_%d", time.Now().UnixNano())

	// Create collection
	t.Log("Creating collection...")
	createResp, err := client.CreateCollection(ctx, &hypatiapb.CreateCollectionRequest{
		Name:                collectionName,
		Description:         "Integration test collection",
		EmbeddingDimensions: 768,
	})
	requireNoError(t, err, "CreateCollection failed")
	requireEqual(t, collectionName, createResp.Name, "Collection name mismatch")
	t.Logf("Created collection: %s", createResp.Name)

	// List collections
	t.Log("Listing collections...")
	listResp, err := client.ListCollections(ctx, &commonpb.Empty{})
	requireNoError(t, err, "ListCollections failed")

	found := false
	for _, c := range listResp.Collections {
		if c.Name == collectionName {
			found = true
			t.Logf("Found collection: %s (docs: %d, chunks: %d)", c.Name, c.DocumentCount, c.ChunkCount)
		}
	}
	requireTrue(t, found, "Created collection not found in list")

	// Delete collection
	t.Log("Deleting collection...")
	_, err = client.DeleteCollection(ctx, &hypatiapb.DeleteCollectionRequest{
		Name: collectionName,
	})
	requireNoError(t, err, "DeleteCollection failed")
	t.Log("Collection deleted successfully")
}

func TestHypatia_DocumentIngestion(t *testing.T) {
	cfg := getTestConfig()
	skipIfServiceUnavailable(t, cfg.HypatiaAddr, "Hypatia")
	skipIfServiceUnavailable(t, cfg.OllamaAddr, "Ollama") // Needed for embeddings
	logTestStart(t, "Hypatia", "DocumentIngestion")

	conn := dialGRPC(t, cfg.HypatiaAddr)
	client := hypatiapb.NewHypatiaServiceClient(conn)

	ctx, cancel := testContext(t, 60*time.Second)
	defer cancel()

	collectionName := fmt.Sprintf("test_docs_%d", time.Now().UnixNano())

	// Create collection first
	_, err := client.CreateCollection(ctx, &hypatiapb.CreateCollectionRequest{
		Name:                collectionName,
		Description:         "Document ingestion test",
		EmbeddingDimensions: 768,
	})
	requireNoError(t, err, "CreateCollection failed")
	defer func() {
		client.DeleteCollection(ctx, &hypatiapb.DeleteCollectionRequest{Name: collectionName})
	}()

	// Ingest document
	t.Log("Ingesting document...")
	ingestResp, err := client.IngestDocument(ctx, &hypatiapb.IngestDocumentRequest{
		Title:      "Test Document",
		Content:    "Dies ist ein Testdokument für die Integration. Es enthält wichtige Informationen über das meinDENKWERK System.",
		Collection: collectionName,
		Source:     "integration_test",
		Options: &hypatiapb.IngestOptions{
			ChunkSize:    256,
			ChunkOverlap: 64,
			Strategy:     hypatiapb.ChunkingStrategy_CHUNKING_STRATEGY_SENTENCE,
		},
	})
	requireNoError(t, err, "IngestDocument failed")
	requireTrue(t, ingestResp.Success, "Ingestion should succeed")
	requireNotEmpty(t, ingestResp.DocumentId, "Document ID should not be empty")

	t.Logf("Ingested document: id=%s, chunks=%d", ingestResp.DocumentId, ingestResp.ChunksCreated)

	// Get document
	t.Log("Getting document...")
	docResp, err := client.GetDocument(ctx, &hypatiapb.GetDocumentRequest{
		DocumentId: ingestResp.DocumentId,
	})
	requireNoError(t, err, "GetDocument failed")
	requireEqual(t, "Test Document", docResp.Title, "Document title mismatch")

	t.Logf("Retrieved document: title=%s, chunks=%d", docResp.Title, docResp.ChunkCount)

	// List documents
	t.Log("Listing documents...")
	listResp, err := client.ListDocuments(ctx, &hypatiapb.ListDocumentsRequest{
		Collection: collectionName,
		PageSize:   10,
	})
	requireNoError(t, err, "ListDocuments failed")
	requireTrue(t, len(listResp.Documents) > 0, "Should have at least one document")

	// Delete document
	t.Log("Deleting document...")
	_, err = client.DeleteDocument(ctx, &hypatiapb.DeleteDocumentRequest{
		DocumentId: ingestResp.DocumentId,
		Collection: collectionName,
	})
	requireNoError(t, err, "DeleteDocument failed")
	t.Log("Document deleted successfully")
}

func TestHypatia_Search(t *testing.T) {
	cfg := getTestConfig()
	skipIfServiceUnavailable(t, cfg.HypatiaAddr, "Hypatia")
	skipIfServiceUnavailable(t, cfg.OllamaAddr, "Ollama")
	logTestStart(t, "Hypatia", "Search")

	conn := dialGRPC(t, cfg.HypatiaAddr)
	client := hypatiapb.NewHypatiaServiceClient(conn)

	ctx, cancel := testContext(t, 90*time.Second)
	defer cancel()

	collectionName := fmt.Sprintf("test_search_%d", time.Now().UnixNano())

	// Setup: Create collection and ingest documents
	_, err := client.CreateCollection(ctx, &hypatiapb.CreateCollectionRequest{
		Name:                collectionName,
		Description:         "Search test collection",
		EmbeddingDimensions: 768,
	})
	requireNoError(t, err, "CreateCollection failed")
	defer func() {
		client.DeleteCollection(ctx, &hypatiapb.DeleteCollectionRequest{Name: collectionName})
	}()

	// Ingest test documents
	docs := []struct {
		title   string
		content string
	}{
		{"Künstliche Intelligenz", "KI und maschinelles Lernen revolutionieren die Technologiebranche."},
		{"Go Programmierung", "Go ist eine effiziente Programmiersprache für Backend-Systeme."},
		{"Datenbanken", "SQL und NoSQL Datenbanken speichern strukturierte und unstrukturierte Daten."},
	}

	for _, doc := range docs {
		_, err := client.IngestDocument(ctx, &hypatiapb.IngestDocumentRequest{
			Title:      doc.title,
			Content:    doc.content,
			Collection: collectionName,
			Source:     "test",
		})
		requireNoError(t, err, "IngestDocument failed for "+doc.title)
	}

	t.Log("Documents ingested, performing search...")

	// Search
	searchResp, err := client.Search(ctx, &hypatiapb.SearchRequest{
		Query:      "Programmierung und Softwareentwicklung",
		Collection: collectionName,
		TopK:       5,
		MinScore:   0.0,
	})
	requireNoError(t, err, "Search failed")

	t.Logf("Search returned %d results in %dms", len(searchResp.Results), searchResp.SearchTimeMs)
	for i, r := range searchResp.Results {
		title := ""
		if r.Metadata != nil {
			title = r.Metadata.Title
		}
		t.Logf("  %d. %s (score: %.3f)", i+1, title, r.Score)
	}
}

func TestHypatia_AugmentPrompt(t *testing.T) {
	cfg := getTestConfig()
	skipIfServiceUnavailable(t, cfg.HypatiaAddr, "Hypatia")
	skipIfServiceUnavailable(t, cfg.OllamaAddr, "Ollama")
	logTestStart(t, "Hypatia", "AugmentPrompt")

	conn := dialGRPC(t, cfg.HypatiaAddr)
	client := hypatiapb.NewHypatiaServiceClient(conn)

	ctx, cancel := testContext(t, 90*time.Second)
	defer cancel()

	collectionName := fmt.Sprintf("test_rag_%d", time.Now().UnixNano())

	// Setup collection with context
	_, err := client.CreateCollection(ctx, &hypatiapb.CreateCollectionRequest{
		Name:                collectionName,
		Description:         "RAG test collection",
		EmbeddingDimensions: 768,
	})
	requireNoError(t, err, "CreateCollection failed")
	defer func() {
		client.DeleteCollection(ctx, &hypatiapb.DeleteCollectionRequest{Name: collectionName})
	}()

	// Add knowledge
	_, err = client.IngestDocument(ctx, &hypatiapb.IngestDocumentRequest{
		Title:      "meinDENKWERK Architektur",
		Content:    "meinDENKWERK besteht aus 7 Microservices: Kant (API Gateway), Russell (Discovery), Turing (LLM), Hypatia (RAG), Leibniz (Agent), Babbage (NLP), und Bayes (Logging).",
		Collection: collectionName,
		Source:     "docs",
	})
	requireNoError(t, err, "IngestDocument failed")

	// Augment prompt
	t.Log("Augmenting prompt with context...")
	augmentResp, err := client.AugmentPrompt(ctx, &hypatiapb.AugmentPromptRequest{
		Prompt:           "Was ist meinDENKWERK?",
		Collection:       collectionName,
		TopK:             3,
		MaxContextTokens: 1000,
	})
	requireNoError(t, err, "AugmentPrompt failed")
	requireNotEmpty(t, augmentResp.AugmentedPrompt, "Augmented prompt should not be empty")

	t.Logf("Augmented prompt (%d tokens, %d sources):\n%s",
		augmentResp.ContextTokens, augmentResp.SourcesUsed, augmentResp.AugmentedPrompt)
}
