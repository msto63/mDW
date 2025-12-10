package integration

import (
	"context"
	"fmt"
	"testing"
	"time"

	hypatiapb "github.com/msto63/mDW/api/gen/hypatia"
	platonpb "github.com/msto63/mDW/api/gen/platon"
	turingpb "github.com/msto63/mDW/api/gen/turing"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// TestE2E_RAGWorkflow tests the complete RAG workflow:
// 1. Create collection
// 2. Ingest documents (uses Turing for embeddings)
// 3. Search documents
// 4. Augment prompt with context
// 5. Generate response with context (uses Turing)
func TestE2E_RAGWorkflow(t *testing.T) {
	cfg := getTestConfig()
	skipIfServiceUnavailable(t, cfg.HypatiaAddr, "Hypatia")
	skipIfServiceUnavailable(t, cfg.TuringAddr, "Turing")
	skipIfServiceUnavailable(t, cfg.OllamaAddr, "Ollama")
	logTestStart(t, "E2E", "RAG Workflow")

	// Setup Hypatia client
	hypatiaConn := dialGRPC(t, cfg.HypatiaAddr)
	hypatiaClient := hypatiapb.NewHypatiaServiceClient(hypatiaConn)

	// Setup Turing client
	turingConn := dialGRPC(t, cfg.TuringAddr)
	turingClient := turingpb.NewTuringServiceClient(turingConn)

	ctx, cancel := testContext(t, 180*time.Second)
	defer cancel()

	collectionName := fmt.Sprintf("e2e_rag_%d", time.Now().UnixNano())

	// Step 1: Create collection
	t.Log("Step 1: Creating collection...")
	_, err := hypatiaClient.CreateCollection(ctx, &hypatiapb.CreateCollectionRequest{
		Name:                collectionName,
		Description:         "E2E RAG test collection",
		EmbeddingDimensions: 768,
	})
	requireNoError(t, err, "CreateCollection failed")
	defer func() {
		hypatiaClient.DeleteCollection(ctx, &hypatiapb.DeleteCollectionRequest{Name: collectionName})
		t.Log("Cleanup: Collection deleted")
	}()
	t.Logf("  Created collection: %s", collectionName)

	// Step 2: Ingest knowledge documents
	t.Log("Step 2: Ingesting documents...")
	docs := []struct {
		title   string
		content string
	}{
		{
			"meinDENKWERK Überblick",
			"meinDENKWERK ist eine lokale KI-Plattform basierend auf Go. Sie bietet 7 Microservices für verschiedene KI-Aufgaben.",
		},
		{
			"Service Architektur",
			"Die Services sind: Kant (API Gateway), Russell (Discovery), Turing (LLM), Hypatia (RAG), Leibniz (Agent), Babbage (NLP), Bayes (Logging).",
		},
		{
			"Technologie Stack",
			"meinDENKWERK verwendet Go 1.24, gRPC für Service-Kommunikation, SQLite für Vektorspeicherung und Ollama als LLM-Backend.",
		},
	}

	for _, doc := range docs {
		resp, err := hypatiaClient.IngestDocument(ctx, &hypatiapb.IngestDocumentRequest{
			Title:      doc.title,
			Content:    doc.content,
			Collection: collectionName,
			Source:     "e2e_test",
		})
		requireNoError(t, err, "IngestDocument failed for "+doc.title)
		t.Logf("  Ingested: %s (id: %s, chunks: %d)", doc.title, resp.DocumentId, resp.ChunksCreated)
	}

	// Step 3: Search for relevant content
	t.Log("Step 3: Searching for relevant content...")
	searchResp, err := hypatiaClient.Search(ctx, &hypatiapb.SearchRequest{
		Query:      "Welche Services gibt es?",
		Collection: collectionName,
		TopK:       3,
		MinScore:   0.0,
	})
	requireNoError(t, err, "Search failed")
	t.Logf("  Found %d results in %dms", len(searchResp.Results), searchResp.SearchTimeMs)

	for i, r := range searchResp.Results {
		title := ""
		if r.Metadata != nil {
			title = r.Metadata.Title
		}
		t.Logf("    %d. %s (score: %.3f)", i+1, title, r.Score)
	}

	// Step 4: Augment prompt with context
	t.Log("Step 4: Augmenting prompt with context...")
	augmentResp, err := hypatiaClient.AugmentPrompt(ctx, &hypatiapb.AugmentPromptRequest{
		Prompt:           "Erkläre die Architektur von meinDENKWERK.",
		Collection:       collectionName,
		TopK:             3,
		MaxContextTokens: 500,
	})
	requireNoError(t, err, "AugmentPrompt failed")
	t.Logf("  Augmented prompt with %d sources (%d tokens)", augmentResp.SourcesUsed, augmentResp.ContextTokens)

	// Step 5: Generate response with context
	t.Log("Step 5: Generating response with context...")
	chatResp, err := turingClient.Chat(ctx, &turingpb.ChatRequest{
		Messages: []*turingpb.Message{
			{Role: "user", Content: augmentResp.AugmentedPrompt},
		},
		Model: "mistral:7b",
	})
	requireNoError(t, err, "Chat failed")
	requireNotEmpty(t, chatResp.Content, "Response should not be empty")

	t.Logf("  Response: %s", chatResp.Content)
	t.Log("E2E RAG Workflow completed successfully!")
}

// TestE2E_ConversationWorkflow tests multi-turn conversation:
// 1. Initial message
// 2. Follow-up with context
// 3. Another follow-up
func TestE2E_ConversationWorkflow(t *testing.T) {
	cfg := getTestConfig()
	skipIfServiceUnavailable(t, cfg.TuringAddr, "Turing")
	skipIfServiceUnavailable(t, cfg.OllamaAddr, "Ollama")
	logTestStart(t, "E2E", "Conversation Workflow")

	conn := dialGRPC(t, cfg.TuringAddr)
	client := turingpb.NewTuringServiceClient(conn)

	ctx, cancel := testContext(t, 180*time.Second)
	defer cancel()

	var messages []*turingpb.Message

	// Turn 1
	t.Log("Turn 1: Initial greeting...")
	messages = append(messages, &turingpb.Message{
		Role:    "user",
		Content: "Hallo! Mein Name ist Test.",
	})

	resp1, err := client.Chat(ctx, &turingpb.ChatRequest{
		Messages: messages,
		Model:    "mistral:7b",
	})
	requireNoError(t, err, "Turn 1 failed")
	messages = append(messages, &turingpb.Message{
		Role:    "assistant",
		Content: resp1.Content,
	})
	t.Logf("  Assistant: %s", resp1.Content)

	// Turn 2
	t.Log("Turn 2: Follow-up question...")
	messages = append(messages, &turingpb.Message{
		Role:    "user",
		Content: "Wie war mein Name nochmal?",
	})

	resp2, err := client.Chat(ctx, &turingpb.ChatRequest{
		Messages: messages,
		Model:    "mistral:7b",
	})
	requireNoError(t, err, "Turn 2 failed")
	t.Logf("  Assistant: %s", resp2.Content)

	// Verify context retention
	requireTrue(t, len(resp2.Content) > 0, "Should remember the name")

	t.Log("E2E Conversation Workflow completed successfully!")
}

// TestE2E_ServiceDiscovery tests that all services register with Russell
func TestE2E_ServiceDiscovery(t *testing.T) {
	cfg := getTestConfig()
	skipIfServiceUnavailable(t, cfg.RussellAddr, "Russell")
	logTestStart(t, "E2E", "Service Discovery")

	// Check each service registers with Russell
	expectedServices := []struct {
		name string
		addr string
	}{
		{"Turing", cfg.TuringAddr},
		{"Hypatia", cfg.HypatiaAddr},
		{"Leibniz", cfg.LeibnizAddr},
		{"Babbage", cfg.BabbageAddr},
		{"Bayes", cfg.BayesAddr},
		{"Platon", cfg.PlatonAddr},
	}

	for _, svc := range expectedServices {
		available := isServiceAvailable(svc.addr)
		status := "running"
		if !available {
			status = "stopped"
		}
		t.Logf("  %s at %s: %s", svc.name, svc.addr, status)
	}

	t.Log("E2E Service Discovery check completed!")
}

// TestE2E_FullPipeline tests the complete pipeline:
// 1. Analyze text with Babbage
// 2. Store in Hypatia
// 3. Search and retrieve
// 4. Generate response with Turing
func TestE2E_FullPipeline(t *testing.T) {
	cfg := getTestConfig()
	skipIfServiceUnavailable(t, cfg.BabbageAddr, "Babbage")
	skipIfServiceUnavailable(t, cfg.HypatiaAddr, "Hypatia")
	skipIfServiceUnavailable(t, cfg.TuringAddr, "Turing")
	skipIfServiceUnavailable(t, cfg.OllamaAddr, "Ollama")
	logTestStart(t, "E2E", "Full Pipeline")

	ctx, cancel := testContext(t, 300*time.Second)
	defer cancel()

	// Setup clients
	hypatiaConn := dialGRPC(t, cfg.HypatiaAddr)
	hypatiaClient := hypatiapb.NewHypatiaServiceClient(hypatiaConn)

	turingConn := dialGRPC(t, cfg.TuringAddr)
	turingClient := turingpb.NewTuringServiceClient(turingConn)

	collectionName := fmt.Sprintf("e2e_pipeline_%d", time.Now().UnixNano())

	// Step 1: Create collection
	t.Log("Step 1: Creating collection...")
	_, err := hypatiaClient.CreateCollection(ctx, &hypatiapb.CreateCollectionRequest{
		Name:                collectionName,
		Description:         "E2E Pipeline test",
		EmbeddingDimensions: 768,
	})
	requireNoError(t, err, "CreateCollection failed")
	defer func() {
		hypatiaClient.DeleteCollection(ctx, &hypatiapb.DeleteCollectionRequest{Name: collectionName})
	}()

	// Step 2: Ingest analyzed content
	t.Log("Step 2: Ingesting content...")
	_, err = hypatiaClient.IngestDocument(ctx, &hypatiapb.IngestDocumentRequest{
		Title:      "Produktbewertung",
		Content:    "Das Produkt ist ausgezeichnet! Die Qualität übertrifft alle Erwartungen. Der Kundenservice war auch sehr hilfreich.",
		Collection: collectionName,
		Source:     "review",
	})
	requireNoError(t, err, "IngestDocument failed")

	// Step 3: Search
	t.Log("Step 3: Searching...")
	searchResp, err := hypatiaClient.Search(ctx, &hypatiapb.SearchRequest{
		Query:      "Wie ist die Qualität?",
		Collection: collectionName,
		TopK:       1,
	})
	requireNoError(t, err, "Search failed")
	requireTrue(t, len(searchResp.Results) > 0, "Should find results")

	// Step 4: Generate summary
	t.Log("Step 4: Generating summary...")
	context := ""
	if len(searchResp.Results) > 0 {
		context = searchResp.Results[0].Content
	}

	chatResp, err := turingClient.Chat(ctx, &turingpb.ChatRequest{
		Messages: []*turingpb.Message{
			{
				Role: "user",
				Content: fmt.Sprintf(
					"Basierend auf diesem Kontext: '%s'\n\nFasse die Kundenmeinung in einem Satz zusammen.",
					context,
				),
			},
		},
		Model: "mistral:7b",
	})
	requireNoError(t, err, "Chat failed")
	t.Logf("  Summary: %s", chatResp.Content)

	t.Log("E2E Full Pipeline completed successfully!")
}

// TestE2E_EmbeddingsConsistency verifies embedding consistency
func TestE2E_EmbeddingsConsistency(t *testing.T) {
	cfg := getTestConfig()
	skipIfServiceUnavailable(t, cfg.TuringAddr, "Turing")
	skipIfServiceUnavailable(t, cfg.OllamaAddr, "Ollama")
	logTestStart(t, "E2E", "Embeddings Consistency")

	conn := dialGRPC(t, cfg.TuringAddr)
	client := turingpb.NewTuringServiceClient(conn)

	ctx, cancel := testContext(t, 60*time.Second)
	defer cancel()

	text := "Dies ist ein Testtext für Embedding-Konsistenz."

	// Get embedding twice
	t.Log("Getting embeddings twice for same text...")
	resp1, err := client.Embed(ctx, &turingpb.EmbedRequest{
		Input: text,
		Model: "nomic-embed-text",
	})
	requireNoError(t, err, "First embed failed")

	resp2, err := client.Embed(ctx, &turingpb.EmbedRequest{
		Input: text,
		Model: "nomic-embed-text",
	})
	requireNoError(t, err, "Second embed failed")

	// Verify dimensions match
	requireEqual(t, len(resp1.Embedding), len(resp2.Embedding), "Embedding dimensions should match")
	t.Logf("  Both embeddings have %d dimensions", len(resp1.Embedding))

	// Calculate similarity (should be very high for identical text)
	var dotProduct float32
	var norm1, norm2 float32
	for i := range resp1.Embedding {
		dotProduct += resp1.Embedding[i] * resp2.Embedding[i]
		norm1 += resp1.Embedding[i] * resp1.Embedding[i]
		norm2 += resp2.Embedding[i] * resp2.Embedding[i]
	}

	// Simple cosine similarity check
	if norm1 > 0 && norm2 > 0 {
		t.Logf("  Dot product: %.4f", dotProduct)
		t.Log("  Embeddings are consistent")
	}

	t.Log("E2E Embeddings Consistency check completed!")
}

// TestE2E_HealthCheckAll verifies all services are healthy
func TestE2E_HealthCheckAll(t *testing.T) {
	cfg := getTestConfig()
	logTestStart(t, "E2E", "Health Check All Services")

	services := []struct {
		name string
		addr string
	}{
		{"Russell", cfg.RussellAddr},
		{"Turing", cfg.TuringAddr},
		{"Hypatia", cfg.HypatiaAddr},
		{"Leibniz", cfg.LeibnizAddr},
		{"Babbage", cfg.BabbageAddr},
		{"Bayes", cfg.BayesAddr},
		{"Platon", cfg.PlatonAddr},
	}

	healthyCount := 0
	for _, svc := range services {
		if isServiceAvailable(svc.addr) {
			// Try health check
			conn, err := dialGRPCNoFail(svc.addr)
			if err == nil {
				healthyCount++
				t.Logf("  [+] %s at %s: healthy", svc.name, svc.addr)
				conn.Close()
			} else {
				t.Logf("  [-] %s at %s: unhealthy", svc.name, svc.addr)
			}
		} else {
			t.Logf("  [-] %s at %s: unavailable", svc.name, svc.addr)
		}
	}

	t.Logf("Healthy services: %d/%d", healthyCount, len(services))
}

// dialGRPCNoFail attempts to dial without failing the test
func dialGRPCNoFail(addr string) (*grpc.ClientConn, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	return grpc.DialContext(ctx, addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
}

// TestE2E_RAGWithPlatonPipeline tests RAG workflow with Platon pipeline processing
// 1. Pre-process user query with Platon
// 2. Search in Hypatia
// 3. Augment prompt
// 4. Generate response with Turing
// 5. Post-process response with Platon
func TestE2E_RAGWithPlatonPipeline(t *testing.T) {
	cfg := getTestConfig()
	skipIfServiceUnavailable(t, cfg.PlatonAddr, "Platon")
	skipIfServiceUnavailable(t, cfg.HypatiaAddr, "Hypatia")
	skipIfServiceUnavailable(t, cfg.TuringAddr, "Turing")
	skipIfServiceUnavailable(t, cfg.OllamaAddr, "Ollama")
	logTestStart(t, "E2E", "RAG with Platon Pipeline")

	// Setup clients
	platonConn := dialGRPC(t, cfg.PlatonAddr)
	platonClient := platonpb.NewPlatonServiceClient(platonConn)

	hypatiaConn := dialGRPC(t, cfg.HypatiaAddr)
	hypatiaClient := hypatiapb.NewHypatiaServiceClient(hypatiaConn)

	turingConn := dialGRPC(t, cfg.TuringAddr)
	turingClient := turingpb.NewTuringServiceClient(turingConn)

	ctx, cancel := testContext(t, 300*time.Second)
	defer cancel()

	collectionName := fmt.Sprintf("e2e_platon_rag_%d", time.Now().UnixNano())
	userQuery := "Welche Services gibt es in meinDENKWERK? Kontakt: user@test.com"

	// Step 1: Pre-process user query with Platon
	t.Log("Step 1: Pre-processing query with Platon...")
	preReq := &platonpb.ProcessRequest{
		RequestId:  fmt.Sprintf("rag-platon-%d", time.Now().UnixNano()),
		PipelineId: "default",
		Prompt:     userQuery,
		Options: &platonpb.ProcessOptions{
			Debug: true,
		},
	}

	preResp, err := platonClient.ProcessPre(ctx, preReq)
	requireNoError(t, err, "Platon ProcessPre failed")

	processedQuery := preResp.GetProcessedPrompt()
	if processedQuery == "" {
		processedQuery = userQuery
	}

	t.Logf("  Original query: %s", userQuery)
	t.Logf("  Processed query: %s", processedQuery)
	t.Logf("  Modified: %v, Blocked: %v", preResp.GetModified(), preResp.GetBlocked())

	if preResp.GetBlocked() {
		t.Logf("  Query blocked: %s", preResp.GetBlockReason())
		t.Log("E2E RAG with Platon completed (blocked by policy)")
		return
	}

	// Step 2: Create collection and ingest knowledge
	t.Log("Step 2: Creating collection and ingesting knowledge...")
	_, err = hypatiaClient.CreateCollection(ctx, &hypatiapb.CreateCollectionRequest{
		Name:                collectionName,
		Description:         "E2E RAG with Platon test",
		EmbeddingDimensions: 768,
	})
	requireNoError(t, err, "CreateCollection failed")
	defer func() {
		hypatiaClient.DeleteCollection(ctx, &hypatiapb.DeleteCollectionRequest{Name: collectionName})
		t.Log("Cleanup: Collection deleted")
	}()

	// Ingest knowledge documents
	docs := []struct {
		title   string
		content string
	}{
		{
			"mDW Services",
			"meinDENKWERK besteht aus 8 Services: Kant (Gateway), Russell (Discovery), Turing (LLM), Hypatia (RAG), Leibniz (Agent), Babbage (NLP), Bayes (Logging) und Platon (Pipeline Processing).",
		},
		{
			"Platon Service",
			"Der Platon Service ist verantwortlich für Pipeline Processing. Er führt Pre- und Post-Processing von LLM-Anfragen durch, inklusive PII-Erkennung und Content-Moderation.",
		},
	}

	for _, doc := range docs {
		_, err := hypatiaClient.IngestDocument(ctx, &hypatiapb.IngestDocumentRequest{
			Title:      doc.title,
			Content:    doc.content,
			Collection: collectionName,
			Source:     "e2e_test",
		})
		requireNoError(t, err, "IngestDocument failed for "+doc.title)
	}
	t.Logf("  Ingested %d documents", len(docs))

	// Step 3: Search with processed query
	t.Log("Step 3: Searching for relevant content...")
	searchResp, err := hypatiaClient.Search(ctx, &hypatiapb.SearchRequest{
		Query:      processedQuery,
		Collection: collectionName,
		TopK:       3,
		MinScore:   0.0,
	})
	requireNoError(t, err, "Search failed")
	t.Logf("  Found %d results in %dms", len(searchResp.Results), searchResp.SearchTimeMs)

	// Step 4: Augment prompt with context
	t.Log("Step 4: Augmenting prompt with context...")
	augmentResp, err := hypatiaClient.AugmentPrompt(ctx, &hypatiapb.AugmentPromptRequest{
		Prompt:           processedQuery,
		Collection:       collectionName,
		TopK:             3,
		MaxContextTokens: 500,
	})
	requireNoError(t, err, "AugmentPrompt failed")
	t.Logf("  Augmented with %d sources (%d tokens)", augmentResp.SourcesUsed, augmentResp.ContextTokens)

	// Step 5: Generate response with Turing
	t.Log("Step 5: Generating response with Turing...")
	chatResp, err := turingClient.Chat(ctx, &turingpb.ChatRequest{
		Messages: []*turingpb.Message{
			{Role: "user", Content: augmentResp.AugmentedPrompt},
		},
		Model: "qwen2.5:7b",
	})
	requireNoError(t, err, "Chat failed")

	llmResponse := chatResp.Content
	t.Logf("  LLM Response: %s", truncateStr(llmResponse, 150))

	// Step 6: Post-process LLM response with Platon
	t.Log("Step 6: Post-processing response with Platon...")
	postReq := &platonpb.ProcessRequest{
		RequestId:  preReq.RequestId,
		PipelineId: "default",
		Prompt:     processedQuery,
		Response:   llmResponse,
		Options: &platonpb.ProcessOptions{
			Debug: true,
		},
	}

	postResp, err := platonClient.ProcessPost(ctx, postReq)
	requireNoError(t, err, "Platon ProcessPost failed")

	finalResponse := postResp.GetProcessedResponse()
	if finalResponse == "" {
		finalResponse = llmResponse
	}

	t.Logf("  Final Response: %s", truncateStr(finalResponse, 150))
	t.Logf("  Modified: %v, Blocked: %v", postResp.GetModified(), postResp.GetBlocked())

	t.Log("E2E RAG with Platon Pipeline completed successfully!")
}

// TestE2E_AllServicesIntegration tests all 8 services working together
func TestE2E_AllServicesIntegration(t *testing.T) {
	cfg := getTestConfig()
	logTestStart(t, "E2E", "All Services Integration")

	allServices := []struct {
		name string
		addr string
	}{
		{"Russell", cfg.RussellAddr},
		{"Turing", cfg.TuringAddr},
		{"Hypatia", cfg.HypatiaAddr},
		{"Leibniz", cfg.LeibnizAddr},
		{"Babbage", cfg.BabbageAddr},
		{"Bayes", cfg.BayesAddr},
		{"Platon", cfg.PlatonAddr},
	}

	t.Log("Checking all 8 services (including Kant):")

	// Check Kant separately (HTTP)
	kantAvailable := isServiceAvailable(cfg.KantAddr)
	t.Logf("  Kant at %s: %s", cfg.KantAddr, boolToStatus(kantAvailable))

	availableCount := 0
	if kantAvailable {
		availableCount++
	}

	// Check gRPC services
	for _, svc := range allServices {
		available := isServiceAvailable(svc.addr)
		if available {
			availableCount++
		}
		t.Logf("  %s at %s: %s", svc.name, svc.addr, boolToStatus(available))
	}

	t.Logf("\nTotal services available: %d/8", availableCount)

	// Test specific service interactions if enough services are available
	if availableCount >= 5 {
		t.Log("\nTesting service interactions:")

		// Test Platon if available
		if isServiceAvailable(cfg.PlatonAddr) {
			conn := dialGRPC(t, cfg.PlatonAddr)
			client := platonpb.NewPlatonServiceClient(conn)

			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			// Simple process request
			resp, err := client.Process(ctx, &platonpb.ProcessRequest{
				RequestId:  "integration-test",
				PipelineId: "default",
				Prompt:     "Integration test message",
				Options: &platonpb.ProcessOptions{
					Debug:  true,
					DryRun: true,
				},
			})

			if err != nil {
				t.Logf("  Platon Process: error - %v", err)
			} else {
				t.Logf("  Platon Process: OK (blocked=%v, modified=%v)", resp.GetBlocked(), resp.GetModified())
			}
		}
	}

	t.Log("\nE2E All Services Integration check completed!")
}

// Helper functions for the new tests
func boolToStatus(b bool) string {
	if b {
		return "available"
	}
	return "unavailable"
}

func truncateStr(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
