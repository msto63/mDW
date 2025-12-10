package cmd

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	hypatiapb "github.com/msto63/mDW/api/gen/hypatia"
	turingpb "github.com/msto63/mDW/api/gen/turing"
	"github.com/msto63/mDW/internal/hypatia/chunking"
	"github.com/msto63/mDW/internal/hypatia/service"
	"github.com/msto63/mDW/internal/hypatia/vectorstore"
	"github.com/msto63/mDW/internal/turing/ollama"
	"github.com/spf13/cobra"
)

var (
	searchCollection string
	searchTopK       int
	searchMinScore   float64
	searchWithAnswer bool
	searchDirect     bool
)

var searchCmd = &cobra.Command{
	Use:   "search <query>",
	Short: "Semantische Suche in Dokumenten",
	Long: `Führt eine semantische Suche in den indizierten Dokumenten durch.

Beispiele:
  mdw search "Wie funktioniert RAG?"
  mdw search --collection docs "API Authentifizierung"
  mdw search --top-k 10 "Machine Learning"
  mdw search --answer "Was sind die Vorteile von Go?"
  mdw search --direct "Query"  # Direkt ohne Hypatia Service`,
	Args: cobra.MinimumNArgs(1),
	RunE: runSearch,
}

var indexCmd = &cobra.Command{
	Use:   "index <datei|verzeichnis>",
	Short: "Dokumente indizieren",
	Long: `Indiziert Dokumente für die semantische Suche.

Unterstützte Formate: .txt, .md, .go, .py, .js, .ts, .json, .yaml, .toml

Beispiele:
  mdw index README.md
  mdw index ./docs/
  mdw index --collection code ./src/`,
	Args: cobra.MinimumNArgs(1),
	RunE: runIndex,
}

func init() {
	rootCmd.AddCommand(searchCmd)
	rootCmd.AddCommand(indexCmd)

	searchCmd.Flags().StringVarP(&searchCollection, "collection", "c", "default", "Collection-Name")
	searchCmd.Flags().IntVarP(&searchTopK, "top-k", "k", 5, "Anzahl Ergebnisse")
	searchCmd.Flags().Float64Var(&searchMinScore, "min-score", 0.7, "Minimaler Relevanz-Score")
	searchCmd.Flags().BoolVarP(&searchWithAnswer, "answer", "a", false, "Mit LLM-generierter Antwort")
	searchCmd.Flags().BoolVar(&searchDirect, "direct", false, "Direkt ohne Hypatia Service")

	indexCmd.Flags().StringVarP(&searchCollection, "collection", "c", "default", "Collection-Name")
	indexCmd.Flags().BoolVar(&searchDirect, "direct", false, "Direkt ohne Hypatia Service")
}

func runSearch(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	query := strings.Join(args, " ")

	if searchDirect {
		return runSearchDirect(ctx, query)
	}

	return runSearchGRPC(ctx, query)
}

func runSearchGRPC(ctx context.Context, query string) error {
	fmt.Printf("Suche: %s\n", query)
	fmt.Println(strings.Repeat("-", 50))

	addrs := DefaultServiceAddresses()
	client, conn, err := NewHypatiaClient(addrs.Hypatia)
	if err != nil {
		return fmt.Errorf("Hypatia-Service nicht erreichbar: %v\nStarte den Service mit: mdw serve hypatia", err)
	}
	defer conn.Close()

	grpcCtx, cancel := context.WithTimeout(ctx, gRPCTimeout)
	defer cancel()

	resp, err := client.Search(grpcCtx, &hypatiapb.SearchRequest{
		Query:      query,
		Collection: searchCollection,
		TopK:       int32(searchTopK),
		MinScore:   float32(searchMinScore),
	})
	if err != nil {
		return fmt.Errorf("Suche fehlgeschlagen: %v", err)
	}

	if len(resp.Results) == 0 {
		fmt.Println("Keine Ergebnisse gefunden.")
		fmt.Println("\nTipp: Indiziere zuerst Dokumente mit 'mdw index <datei>'")
		return nil
	}

	// Display results
	for i, result := range resp.Results {
		fmt.Printf("\n[%d] Score: %.2f\n", i+1, result.Score)
		if result.DocumentId != "" {
			fmt.Printf("    Dokument: %s\n", result.DocumentId)
		}
		if result.Metadata != nil && result.Metadata.Title != "" {
			fmt.Printf("    Titel: %s\n", result.Metadata.Title)
		}

		// Truncate content for display
		content := result.Content
		if len(content) > 200 {
			content = content[:200] + "..."
		}
		fmt.Printf("    %s\n", content)
	}

	// Generate answer if requested
	if searchWithAnswer && len(resp.Results) > 0 {
		fmt.Println("\n" + strings.Repeat("-", 50))
		fmt.Println("Generiere Antwort...")

		contextText := buildContextFromGRPC(resp.Results)
		answer, err := generateAnswerGRPC(ctx, query, contextText)
		if err != nil {
			fmt.Printf("Antwort-Generierung fehlgeschlagen: %v\n", err)
		} else {
			fmt.Printf("\nAntwort:\n%s\n", answer)
		}
	}

	return nil
}

func buildContextFromGRPC(results []*hypatiapb.SearchResult) string {
	var parts []string
	for i, r := range results {
		parts = append(parts, fmt.Sprintf("[Dokument %d]\n%s", i+1, r.Content))
	}
	return strings.Join(parts, "\n\n")
}

func generateAnswerGRPC(ctx context.Context, query, contextText string) (string, error) {
	addrs := DefaultServiceAddresses()
	client, conn, err := NewTuringClient(addrs.Turing)
	if err != nil {
		return "", fmt.Errorf("Turing-Service nicht erreichbar: %v", err)
	}
	defer conn.Close()

	prompt := fmt.Sprintf(`Basierend auf den folgenden Dokumenten, beantworte die Frage.

Dokumente:
%s

Frage: %s

Antworte präzise und zitiere relevante Stellen aus den Dokumenten.`, contextText, query)

	grpcCtx, cancel := context.WithTimeout(ctx, gRPCTimeout)
	defer cancel()

	var fullResponse strings.Builder
	stream, err := client.StreamChat(grpcCtx, &turingpb.ChatRequest{
		Messages: []*turingpb.Message{
			{Role: "user", Content: prompt},
		},
		Model: chatModel,
	})
	if err != nil {
		return "", err
	}

	for {
		chunk, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", err
		}
		fmt.Print(chunk.Delta)
		fullResponse.WriteString(chunk.Delta)
		if chunk.Done {
			break
		}
	}
	fmt.Println()

	return fullResponse.String(), nil
}

func runIndex(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	if searchDirect {
		return runIndexDirect(ctx, args)
	}

	return runIndexGRPC(ctx, args)
}

func runIndexGRPC(ctx context.Context, args []string) error {
	addrs := DefaultServiceAddresses()
	client, conn, err := NewHypatiaClient(addrs.Hypatia)
	if err != nil {
		return fmt.Errorf("Hypatia-Service nicht erreichbar: %v\nStarte den Service mit: mdw serve hypatia", err)
	}
	defer conn.Close()

	var totalDocs int
	for _, path := range args {
		info, err := os.Stat(path)
		if err != nil {
			fmt.Printf("Fehler: %s nicht gefunden\n", path)
			continue
		}

		if info.IsDir() {
			count, err := indexDirectoryGRPC(ctx, client, path)
			if err != nil {
				fmt.Printf("Fehler beim Indizieren von %s: %v\n", path, err)
				continue
			}
			totalDocs += count
		} else {
			if err := indexFileGRPC(ctx, client, path); err != nil {
				fmt.Printf("Fehler beim Indizieren von %s: %v\n", path, err)
				continue
			}
			totalDocs++
		}
	}

	fmt.Printf("\n%d Dokument(e) indiziert in Collection '%s'\n", totalDocs, searchCollection)
	return nil
}

func indexDirectoryGRPC(ctx context.Context, client hypatiapb.HypatiaServiceClient, dir string) (int, error) {
	var count int

	supportedExts := map[string]bool{
		".txt": true, ".md": true, ".go": true, ".py": true,
		".js": true, ".ts": true, ".json": true, ".yaml": true,
		".yml": true, ".toml": true, ".html": true, ".css": true,
	}

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			if strings.HasPrefix(info.Name(), ".") && info.Name() != "." {
				return filepath.SkipDir
			}
			return nil
		}

		ext := strings.ToLower(filepath.Ext(path))
		if !supportedExts[ext] {
			return nil
		}

		if err := indexFileGRPC(ctx, client, path); err != nil {
			fmt.Printf("  Warnung: %s - %v\n", path, err)
			return nil
		}

		count++
		return nil
	})

	return count, err
}

func indexFileGRPC(ctx context.Context, client hypatiapb.HypatiaServiceClient, path string) error {
	content, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	fmt.Printf("Indiziere: %s\n", path)

	grpcCtx, cancel := context.WithTimeout(ctx, gRPCTimeout)
	defer cancel()

	_, err = client.IngestDocument(grpcCtx, &hypatiapb.IngestDocumentRequest{
		Title:      filepath.Base(path),
		Content:    string(content),
		Collection: searchCollection,
		Source:     path,
		Metadata: map[string]string{
			"file": path,
		},
	})

	return err
}

// Direct mode functions (without gRPC)

func runSearchDirect(ctx context.Context, query string) error {
	fmt.Printf("Suche (Direkt): %s\n", query)
	fmt.Println(strings.Repeat("-", 50))

	store := vectorstore.NewMemoryStore()
	ollamaClient := ollama.NewClient(ollama.DefaultConfig())

	embedFunc := func(ctx context.Context, texts []string) ([][]float64, error) {
		resp, err := ollamaClient.Embed(ctx, &ollama.EmbeddingRequest{
			Model: "nomic-embed-text",
			Input: texts,
		})
		if err != nil {
			return nil, err
		}
		return resp.Embeddings, nil
	}

	svcCfg := service.DefaultConfig()
	svcCfg.EmbeddingFunc = embedFunc
	svc, err := service.NewService(svcCfg, store)
	if err != nil {
		return err
	}
	svc.SetEmbeddingFunc(embedFunc)

	results, err := svc.Search(ctx, &service.SearchRequest{
		Query:      query,
		Collection: searchCollection,
		TopK:       searchTopK,
		MinScore:   searchMinScore,
	})
	if err != nil {
		return fmt.Errorf("Suche fehlgeschlagen: %v", err)
	}

	if len(results) == 0 {
		fmt.Println("Keine Ergebnisse gefunden.")
		fmt.Println("\nTipp: Indiziere zuerst Dokumente mit 'mdw index <datei>'")
		return nil
	}

	for i, result := range results {
		fmt.Printf("\n[%d] Score: %.2f\n", i+1, result.Score)
		if id, ok := result.Metadata["parent_id"]; ok {
			fmt.Printf("    Dokument: %s\n", id)
		}

		content := result.Content
		if len(content) > 200 {
			content = content[:200] + "..."
		}
		fmt.Printf("    %s\n", content)
	}

	if searchWithAnswer && len(results) > 0 {
		fmt.Println("\n" + strings.Repeat("-", 50))
		fmt.Println("Generiere Antwort...")

		contextText := buildContext(results)
		answer, err := generateAnswer(ctx, ollamaClient, query, contextText)
		if err != nil {
			fmt.Printf("Antwort-Generierung fehlgeschlagen: %v\n", err)
		} else {
			fmt.Printf("\nAntwort:\n%s\n", answer)
		}
	}

	return nil
}

func runIndexDirect(ctx context.Context, args []string) error {
	store := vectorstore.NewMemoryStore()
	ollamaClient := ollama.NewClient(ollama.DefaultConfig())

	if err := ollamaClient.Ping(ctx); err != nil {
		return fmt.Errorf("Ollama nicht erreichbar: %v", err)
	}

	embedFunc := func(ctx context.Context, texts []string) ([][]float64, error) {
		resp, err := ollamaClient.Embed(ctx, &ollama.EmbeddingRequest{
			Model: "nomic-embed-text",
			Input: texts,
		})
		if err != nil {
			return nil, err
		}
		return resp.Embeddings, nil
	}

	svcCfg := service.DefaultConfig()
	svcCfg.EmbeddingFunc = embedFunc
	svc, err := service.NewService(svcCfg, store)
	if err != nil {
		return err
	}
	svc.SetEmbeddingFunc(embedFunc)

	var totalDocs int
	for _, path := range args {
		info, err := os.Stat(path)
		if err != nil {
			fmt.Printf("Fehler: %s nicht gefunden\n", path)
			continue
		}

		if info.IsDir() {
			count, err := indexDirectory(ctx, svc, path)
			if err != nil {
				fmt.Printf("Fehler beim Indizieren von %s: %v\n", path, err)
				continue
			}
			totalDocs += count
		} else {
			if err := indexFile(ctx, svc, path); err != nil {
				fmt.Printf("Fehler beim Indizieren von %s: %v\n", path, err)
				continue
			}
			totalDocs++
		}
	}

	fmt.Printf("\n%d Dokument(e) indiziert in Collection '%s'\n", totalDocs, searchCollection)
	return nil
}

func indexDirectory(ctx context.Context, svc *service.Service, dir string) (int, error) {
	var count int

	supportedExts := map[string]bool{
		".txt": true, ".md": true, ".go": true, ".py": true,
		".js": true, ".ts": true, ".json": true, ".yaml": true,
		".yml": true, ".toml": true, ".html": true, ".css": true,
	}

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			if strings.HasPrefix(info.Name(), ".") && info.Name() != "." {
				return filepath.SkipDir
			}
			return nil
		}

		ext := strings.ToLower(filepath.Ext(path))
		if !supportedExts[ext] {
			return nil
		}

		if err := indexFile(ctx, svc, path); err != nil {
			fmt.Printf("  Warnung: %s - %v\n", path, err)
			return nil
		}

		count++
		return nil
	})

	return count, err
}

func indexFile(ctx context.Context, svc *service.Service, path string) error {
	content, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	fmt.Printf("Indiziere: %s\n", path)

	chunker := chunking.NewChunker(chunking.DefaultConfig())
	chunks := chunker.Split(string(content), filepath.Base(path))

	for _, chunk := range chunks {
		req := &service.IndexRequest{
			ID:         chunk.ID,
			Content:    chunk.Content,
			Collection: searchCollection,
			Metadata: map[string]string{
				"file":      path,
				"chunk_idx": fmt.Sprintf("%d", chunk.Index),
				"parent_id": filepath.Base(path),
			},
		}

		if err := svc.Index(ctx, req); err != nil {
			return err
		}
	}

	return nil
}

func buildContext(results []service.SearchResult) string {
	var parts []string
	for i, r := range results {
		parts = append(parts, fmt.Sprintf("[Dokument %d]\n%s", i+1, r.Content))
	}
	return strings.Join(parts, "\n\n")
}

func generateAnswer(ctx context.Context, client *ollama.Client, query, contextText string) (string, error) {
	prompt := fmt.Sprintf(`Basierend auf den folgenden Dokumenten, beantworte die Frage.

Dokumente:
%s

Frage: %s

Antworte präzise und zitiere relevante Stellen aus den Dokumenten.`, contextText, query)

	resp, err := client.Generate(ctx, &ollama.GenerateRequest{
		Model:  chatModel,
		Prompt: prompt,
	})
	if err != nil {
		return "", err
	}

	return resp.Response, nil
}
