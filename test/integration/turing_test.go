package integration

import (
	"io"
	"testing"
	"time"

	commonpb "github.com/msto63/mDW/api/gen/common"
	turingpb "github.com/msto63/mDW/api/gen/turing"
)

func TestTuring_HealthCheck(t *testing.T) {
	cfg := getTestConfig()
	skipIfServiceUnavailable(t, cfg.TuringAddr, "Turing")
	logTestStart(t, "Turing", "HealthCheck")

	conn := dialGRPC(t, cfg.TuringAddr)
	client := turingpb.NewTuringServiceClient(conn)

	ctx, cancel := testContext(t, 10*time.Second)
	defer cancel()

	resp, err := client.HealthCheck(ctx, &commonpb.HealthCheckRequest{})
	requireNoError(t, err, "HealthCheck failed")
	requireEqual(t, "healthy", resp.Status, "Service should be healthy")
	requireNotEmpty(t, resp.Version, "Version should not be empty")

	t.Logf("Turing health: status=%s version=%s", resp.Status, resp.Version)
}

func TestTuring_ListModels(t *testing.T) {
	cfg := getTestConfig()
	skipIfServiceUnavailable(t, cfg.TuringAddr, "Turing")
	skipIfServiceUnavailable(t, cfg.OllamaAddr, "Ollama")
	logTestStart(t, "Turing", "ListModels")

	conn := dialGRPC(t, cfg.TuringAddr)
	client := turingpb.NewTuringServiceClient(conn)

	ctx, cancel := testContext(t, 30*time.Second)
	defer cancel()

	resp, err := client.ListModels(ctx, &commonpb.Empty{})
	requireNoError(t, err, "ListModels failed")

	t.Logf("Found %d models", len(resp.Models))
	for _, m := range resp.Models {
		t.Logf("  - %s (size: %d, provider: %s)", m.Name, m.Size, m.Provider)
	}
}

func TestTuring_Chat(t *testing.T) {
	cfg := getTestConfig()
	skipIfServiceUnavailable(t, cfg.TuringAddr, "Turing")
	skipIfServiceUnavailable(t, cfg.OllamaAddr, "Ollama")
	logTestStart(t, "Turing", "Chat")

	conn := dialGRPC(t, cfg.TuringAddr)
	client := turingpb.NewTuringServiceClient(conn)

	ctx, cancel := testContext(t, 60*time.Second)
	defer cancel()

	resp, err := client.Chat(ctx, &turingpb.ChatRequest{
		Messages: []*turingpb.Message{
			{Role: "user", Content: "Antworte nur mit 'Hallo'. Nichts anderes."},
		},
		Model: "mistral:7b",
	})
	requireNoError(t, err, "Chat failed")
	requireNotEmpty(t, resp.Content, "Response content should not be empty")

	t.Logf("Chat response: %s", resp.Content)
}

func TestTuring_StreamChat(t *testing.T) {
	cfg := getTestConfig()
	skipIfServiceUnavailable(t, cfg.TuringAddr, "Turing")
	skipIfServiceUnavailable(t, cfg.OllamaAddr, "Ollama")
	logTestStart(t, "Turing", "StreamChat")

	conn := dialGRPC(t, cfg.TuringAddr)
	client := turingpb.NewTuringServiceClient(conn)

	ctx, cancel := testContext(t, 60*time.Second)
	defer cancel()

	stream, err := client.StreamChat(ctx, &turingpb.ChatRequest{
		Messages: []*turingpb.Message{
			{Role: "user", Content: "Zähle von 1 bis 3. Nur die Zahlen."},
		},
		Model: "mistral:7b",
	})
	requireNoError(t, err, "StreamChat failed")

	var fullResponse string
	chunkCount := 0
	for {
		chunk, err := stream.Recv()
		if err == io.EOF {
			break
		}
		requireNoError(t, err, "Stream receive failed")
		fullResponse += chunk.Delta
		chunkCount++
		if chunk.Done {
			break
		}
	}

	requireNotEmpty(t, fullResponse, "Streamed response should not be empty")
	t.Logf("Received %d chunks, full response: %s", chunkCount, fullResponse)
}

func TestTuring_Embed(t *testing.T) {
	cfg := getTestConfig()
	skipIfServiceUnavailable(t, cfg.TuringAddr, "Turing")
	skipIfServiceUnavailable(t, cfg.OllamaAddr, "Ollama")
	logTestStart(t, "Turing", "Embed")

	conn := dialGRPC(t, cfg.TuringAddr)
	client := turingpb.NewTuringServiceClient(conn)

	ctx, cancel := testContext(t, 30*time.Second)
	defer cancel()

	resp, err := client.Embed(ctx, &turingpb.EmbedRequest{
		Input: "Dies ist ein Testtext für Embeddings.",
		Model: "nomic-embed-text",
	})
	requireNoError(t, err, "Embed failed")
	requireTrue(t, len(resp.Embedding) > 0, "Embedding should not be empty")

	t.Logf("Embedding dimensions: %d", len(resp.Embedding))
}

func TestTuring_BatchEmbed(t *testing.T) {
	cfg := getTestConfig()
	skipIfServiceUnavailable(t, cfg.TuringAddr, "Turing")
	skipIfServiceUnavailable(t, cfg.OllamaAddr, "Ollama")
	logTestStart(t, "Turing", "BatchEmbed")

	conn := dialGRPC(t, cfg.TuringAddr)
	client := turingpb.NewTuringServiceClient(conn)

	ctx, cancel := testContext(t, 60*time.Second)
	defer cancel()

	inputs := []string{
		"Erster Testtext.",
		"Zweiter Testtext.",
		"Dritter Testtext.",
	}

	resp, err := client.BatchEmbed(ctx, &turingpb.BatchEmbedRequest{
		Inputs: inputs,
		Model:  "nomic-embed-text",
	})
	requireNoError(t, err, "BatchEmbed failed")
	requireEqual(t, len(inputs), len(resp.Embeddings), "Should return same number of embeddings")

	for i, emb := range resp.Embeddings {
		requireTrue(t, len(emb.Embedding) > 0, "Each embedding should have values")
		t.Logf("Embedding %d: %d dimensions", i, len(emb.Embedding))
	}
}
