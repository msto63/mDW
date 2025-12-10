package cmd

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	babbagepb "github.com/msto63/mDW/api/gen/babbage"
	turingpb "github.com/msto63/mDW/api/gen/turing"
	"github.com/msto63/mDW/internal/babbage/service"
	"github.com/msto63/mDW/internal/turing/ollama"
	"github.com/spf13/cobra"
)

var (
	analyzeOperations []string
	analyzeSummarize  bool
	analyzeMaxWords   int
	analyzeDirect     bool
)

var analyzeCmd = &cobra.Command{
	Use:   "analyze [text]",
	Short: "Textanalyse mit NLP",
	Long: `Analysiert Text mit verschiedenen NLP-Operationen.

Operationen:
  sentiment  - Sentiment-Analyse
  entities   - Named Entity Recognition
  keywords   - Keyword-Extraktion
  language   - Spracherkennung

Beispiele:
  mdw analyze "Das Produkt ist ausgezeichnet!"
  mdw analyze --summarize README.md
  mdw analyze --operations sentiment,keywords "Text hier"
  mdw analyze --direct "Text"  # Direkt ohne Babbage Service
  echo "Text" | mdw analyze`,
	RunE: runAnalyze,
}

var summarizeCmd = &cobra.Command{
	Use:   "summarize [text|datei]",
	Short: "Text zusammenfassen",
	Long: `Erstellt eine Zusammenfassung des Textes.

Beispiele:
  mdw summarize "Langer Text hier..."
  mdw summarize README.md
  mdw summarize --max-words 50 dokument.txt
  cat artikel.txt | mdw summarize`,
	RunE: runSummarize,
}

func init() {
	rootCmd.AddCommand(analyzeCmd)
	rootCmd.AddCommand(summarizeCmd)

	analyzeCmd.Flags().StringSliceVarP(&analyzeOperations, "operations", "o", []string{"all"}, "Operationen (sentiment,entities,keywords,language,all)")
	analyzeCmd.Flags().BoolVarP(&analyzeSummarize, "summarize", "s", false, "Auch zusammenfassen")
	analyzeCmd.Flags().IntVar(&analyzeMaxWords, "max-words", 100, "Max. Wörter für Zusammenfassung")
	analyzeCmd.Flags().BoolVar(&analyzeDirect, "direct", false, "Direkt ohne Babbage Service")

	summarizeCmd.Flags().IntVar(&analyzeMaxWords, "max-words", 100, "Max. Wörter")
	summarizeCmd.Flags().BoolVar(&analyzeDirect, "direct", false, "Direkt ohne Babbage Service")
}

func runAnalyze(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	text, err := getInputText(args)
	if err != nil {
		return err
	}

	if text == "" {
		return fmt.Errorf("kein Text zum Analysieren")
	}

	if analyzeDirect {
		return runAnalyzeDirect(ctx, text)
	}

	return runAnalyzeGRPC(ctx, text)
}

func runAnalyzeGRPC(ctx context.Context, text string) error {
	addrs := DefaultServiceAddresses()
	client, conn, err := NewBabbageClient(addrs.Babbage)
	if err != nil {
		return fmt.Errorf("Babbage-Service nicht erreichbar: %v\nStarte den Service mit: mdw serve babbage", err)
	}
	defer conn.Close()

	runAll := contains(analyzeOperations, "all")

	fmt.Println("Textanalyse (gRPC)")
	fmt.Println(strings.Repeat("=", 50))
	fmt.Printf("Textlänge: %d Zeichen\n\n", len(text))

	grpcCtx, cancel := context.WithTimeout(ctx, gRPCTimeout)
	defer cancel()

	resp, err := client.Analyze(grpcCtx, &babbagepb.AnalyzeRequest{
		Text: text,
	})
	if err != nil {
		return fmt.Errorf("Analyse fehlgeschlagen: %v", err)
	}

	// Display results based on requested operations
	if runAll || contains(analyzeOperations, "language") {
		fmt.Printf("Sprache: %s (%.0f%% Konfidenz)\n", getLanguageName(resp.Language), resp.LanguageConfidence*100)
	}

	if runAll || contains(analyzeOperations, "sentiment") {
		if resp.Sentiment != nil {
			sentimentStr := resp.Sentiment.Sentiment.String()
			emoji := getSentimentEmoji(sentimentStr)
			fmt.Printf("Sentiment: %s %s (Score: %.2f)\n",
				emoji, sentimentStr, resp.Sentiment.Score)
		}
	}

	if runAll || contains(analyzeOperations, "keywords") {
		if len(resp.Keywords) > 0 {
			displayKeywords := resp.Keywords
			if len(displayKeywords) > 10 {
				displayKeywords = displayKeywords[:10]
			}
			var keywordStrs []string
			for _, kw := range displayKeywords {
				keywordStrs = append(keywordStrs, kw.Word)
			}
			fmt.Printf("\nKeywords: %s\n", strings.Join(keywordStrs, ", "))
		}
	}

	if runAll || contains(analyzeOperations, "entities") {
		if len(resp.Entities) > 0 {
			fmt.Println("\nEntitäten:")
			for _, e := range resp.Entities {
				fmt.Printf("  - %s (%s)\n", e.Text, e.Type.String())
			}
		}
	}

	// Statistics - use basic text stats since proto doesn't have them
	words := len(strings.Fields(text))
	fmt.Printf("\nStatistiken:\n")
	fmt.Printf("  Wörter: %d\n", words)
	fmt.Printf("  Zeichen: %d\n", len(text))

	// Summarize if requested
	if analyzeSummarize {
		fmt.Println("\n" + strings.Repeat("-", 50))
		fmt.Println("Zusammenfassung:")

		summary, err := summarizeGRPC(ctx, client, text)
		if err != nil {
			fmt.Printf("Zusammenfassung fehlgeschlagen: %v\n", err)
		} else {
			fmt.Println(summary)
		}
	}

	return nil
}

func summarizeGRPC(ctx context.Context, client babbagepb.BabbageServiceClient, text string) (string, error) {
	grpcCtx, cancel := context.WithTimeout(ctx, gRPCTimeout)
	defer cancel()

	resp, err := client.Summarize(grpcCtx, &babbagepb.SummarizeRequest{
		Text:      text,
		MaxLength: int32(analyzeMaxWords),
	})
	if err != nil {
		return "", err
	}

	return resp.Summary, nil
}

func runSummarize(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	text, err := getInputText(args)
	if err != nil {
		return err
	}

	if text == "" {
		return fmt.Errorf("kein Text zum Zusammenfassen")
	}

	fmt.Println("Erstelle Zusammenfassung...")
	fmt.Println(strings.Repeat("-", 50))

	if analyzeDirect {
		return runSummarizeDirect(ctx, text)
	}

	return runSummarizeGRPC(ctx, text)
}

func runSummarizeGRPC(ctx context.Context, text string) error {
	// First try Babbage service for extractive summary
	addrs := DefaultServiceAddresses()
	babbageClient, babbageConn, err := NewBabbageClient(addrs.Babbage)
	if err == nil {
		defer babbageConn.Close()

		grpcCtx, cancel := context.WithTimeout(ctx, gRPCTimeout)
		resp, err := babbageClient.Summarize(grpcCtx, &babbagepb.SummarizeRequest{
			Text:      text,
			MaxLength: int32(analyzeMaxWords),
		})
		cancel()

		if err == nil {
			fmt.Println(resp.Summary)
			return nil
		}
	}

	// Fallback to Turing for LLM-based summarization
	turingClient, turingConn, err := NewTuringClient(addrs.Turing)
	if err != nil {
		return fmt.Errorf("Keine Services erreichbar: %v", err)
	}
	defer turingConn.Close()

	prompt := fmt.Sprintf(
		"Fasse den folgenden Text in maximal %d Wörtern zusammen. "+
			"Behalte die wichtigsten Informationen bei.\n\nText:\n%s",
		analyzeMaxWords, text)

	grpcCtx, cancel := context.WithTimeout(ctx, gRPCTimeout)
	defer cancel()

	stream, err := turingClient.StreamChat(grpcCtx, &turingpb.ChatRequest{
		Messages: []*turingpb.Message{
			{Role: "user", Content: prompt},
		},
		Model: chatModel,
	})
	if err != nil {
		return err
	}

	for {
		chunk, err := stream.Recv()
		if err == io.EOF {
			fmt.Println()
			return nil
		}
		if err != nil {
			return err
		}
		fmt.Print(chunk.Delta)
		if chunk.Done {
			fmt.Println()
			return nil
		}
	}
}

// Direct mode functions (without gRPC)

func runAnalyzeDirect(ctx context.Context, text string) error {
	svc, err := service.NewService(service.Config{})
	if err != nil {
		return err
	}

	runAll := contains(analyzeOperations, "all")

	fmt.Println("Textanalyse (Direkt)")
	fmt.Println(strings.Repeat("=", 50))
	fmt.Printf("Textlänge: %d Zeichen\n\n", len(text))

	result, err := svc.Analyze(ctx, text)
	if err != nil {
		return fmt.Errorf("Analyse fehlgeschlagen: %v", err)
	}

	if runAll || contains(analyzeOperations, "language") {
		fmt.Printf("Sprache: %s\n", getLanguageName(result.Language))
	}

	if runAll || contains(analyzeOperations, "sentiment") {
		if result.Sentiment != nil {
			emoji := getSentimentEmoji(string(result.Sentiment.Label))
			fmt.Printf("Sentiment: %s %s (Score: %.2f)\n",
				emoji, result.Sentiment.Label, result.Sentiment.Score)
		}
	}

	if runAll || contains(analyzeOperations, "keywords") {
		displayKeywords := result.Keywords
		if len(displayKeywords) > 10 {
			displayKeywords = displayKeywords[:10]
		}
		fmt.Printf("\nKeywords: %s\n", strings.Join(displayKeywords, ", "))
	}

	if runAll || contains(analyzeOperations, "entities") {
		if len(result.Entities) > 0 {
			fmt.Println("\nEntitäten:")
			for _, e := range result.Entities {
				fmt.Printf("  - %s (%s)\n", e.Text, e.Type)
			}
		}
	}

	fmt.Printf("\nStatistiken:\n")
	fmt.Printf("  Wörter: %d\n", result.WordCount)
	fmt.Printf("  Sätze: %d\n", result.Sentences)
	fmt.Printf("  Zeichen: %d\n", result.CharCount)

	if analyzeSummarize {
		fmt.Println("\n" + strings.Repeat("-", 50))
		fmt.Println("Zusammenfassung:")

		ollamaClient := ollama.NewClient(ollama.DefaultConfig())
		if err := ollamaClient.Ping(ctx); err == nil {
			svc.SetLLMFunc(func(ctx context.Context, prompt string) (string, error) {
				resp, err := ollamaClient.Generate(ctx, &ollama.GenerateRequest{
					Model:  chatModel,
					Prompt: prompt,
				})
				if err != nil {
					return "", err
				}
				return resp.Response, nil
			})

			summary, err := svc.Summarize(ctx, &service.SummarizeRequest{
				Text:      text,
				MaxLength: analyzeMaxWords,
			})
			if err != nil {
				fmt.Printf("Zusammenfassung fehlgeschlagen: %v\n", err)
			} else {
				fmt.Println(summary)
			}
		} else {
			summary, _ := svc.Summarize(ctx, &service.SummarizeRequest{
				Text:      text,
				MaxLength: analyzeMaxWords,
			})
			fmt.Println(summary)
		}
	}

	return nil
}

func runSummarizeDirect(ctx context.Context, text string) error {
	ollamaClient := ollama.NewClient(ollama.DefaultConfig())
	if err := ollamaClient.Ping(ctx); err == nil {
		prompt := fmt.Sprintf(
			"Fasse den folgenden Text in maximal %d Wörtern zusammen. "+
				"Behalte die wichtigsten Informationen bei.\n\nText:\n%s",
			analyzeMaxWords, text)

		if chatStream {
			req := &ollama.GenerateRequest{
				Model:  chatModel,
				Prompt: prompt,
				Stream: true,
			}
			respCh, errCh := ollamaClient.GenerateStream(ctx, req)

			for {
				select {
				case resp, ok := <-respCh:
					if !ok {
						fmt.Println()
						return nil
					}
					fmt.Print(resp.Response)
					if resp.Done {
						fmt.Println()
						return nil
					}
				case err, ok := <-errCh:
					if ok && err != nil {
						return err
					}
				case <-ctx.Done():
					return ctx.Err()
				}
			}
		} else {
			resp, err := ollamaClient.Generate(ctx, &ollama.GenerateRequest{
				Model:  chatModel,
				Prompt: prompt,
			})
			if err != nil {
				return fmt.Errorf("Zusammenfassung fehlgeschlagen: %v", err)
			}
			fmt.Println(resp.Response)
		}
	} else {
		svc, _ := service.NewService(service.Config{})
		summary, err := svc.Summarize(ctx, &service.SummarizeRequest{
			Text:      text,
			MaxLength: analyzeMaxWords,
		})
		if err != nil {
			return err
		}
		fmt.Println(summary)
	}

	return nil
}

func getInputText(args []string) (string, error) {
	stat, _ := os.Stdin.Stat()
	if (stat.Mode() & os.ModeCharDevice) == 0 {
		data, err := io.ReadAll(os.Stdin)
		if err != nil {
			return "", err
		}
		return string(data), nil
	}

	if len(args) > 0 {
		if _, err := os.Stat(args[0]); err == nil {
			data, err := os.ReadFile(args[0])
			if err != nil {
				return "", err
			}
			return string(data), nil
		}
		return strings.Join(args, " "), nil
	}

	return "", nil
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func getLanguageName(code string) string {
	names := map[string]string{
		"de": "Deutsch",
		"en": "Englisch",
		"fr": "Französisch",
		"es": "Spanisch",
		"it": "Italienisch",
	}
	if name, ok := names[code]; ok {
		return name
	}
	return code
}

func getSentimentEmoji(sentiment string) string {
	switch sentiment {
	case "positive":
		return "+"
	case "negative":
		return "-"
	default:
		return "o"
	}
}
