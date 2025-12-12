package cmd

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	turingpb "github.com/msto63/mDW/api/gen/turing"
	"github.com/msto63/mDW/internal/turing/ollama"
	"github.com/spf13/cobra"
)

var (
	chatModel       string
	chatSystem      string
	chatTemperature float64
	chatMaxTokens   int
	chatStream      bool
	chatDirect      bool // Use Ollama directly instead of gRPC
)

var chatCmd = &cobra.Command{
	Use:   "chat [nachricht]",
	Short: "Chat mit dem LLM",
	Long: `Startet eine Chat-Sitzung mit dem LLM.

Ohne Argument wird ein interaktiver Chat gestartet.
Mit Argument wird eine einzelne Nachricht gesendet.

Beispiele:
  mdw chat "Was ist die Hauptstadt von Deutschland?"
  mdw chat --model llama3.2 "Erkläre Quantencomputing"
  mdw chat --direct  # Direkt mit Ollama (ohne Turing Service)
  mdw chat  # Interaktiver Modus`,
	RunE: runChat,
}

func init() {
	rootCmd.AddCommand(chatCmd)

	chatCmd.Flags().StringVarP(&chatModel, "model", "m", "", "LLM-Modell (leer = Turing Default)")
	chatCmd.Flags().StringVarP(&chatSystem, "system", "s", "", "System-Prompt")
	chatCmd.Flags().Float64VarP(&chatTemperature, "temperature", "t", 0.7, "Temperatur (0.0-2.0)")
	chatCmd.Flags().IntVar(&chatMaxTokens, "max-tokens", 2048, "Maximale Anzahl Tokens")
	chatCmd.Flags().BoolVar(&chatStream, "stream", true, "Streaming-Ausgabe")
	chatCmd.Flags().BoolVar(&chatDirect, "direct", false, "Direkt mit Ollama kommunizieren (ohne Turing Service)")
}

func runChat(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	// If direct mode, use Ollama client
	if chatDirect {
		return runChatDirect(ctx, args)
	}

	// Use gRPC Turing service
	return runChatGRPC(ctx, args)
}

func runChatGRPC(ctx context.Context, args []string) error {
	addrs := DefaultServiceAddresses()
	client, conn, err := NewTuringClient(addrs.Turing)
	if err != nil {
		return fmt.Errorf("Turing-Service nicht erreichbar: %v\nStarte den Service mit: mdw serve turing", err)
	}
	defer conn.Close()

	// Get default model from Turing if not specified
	if chatModel == "" {
		configResp, err := client.GetConfig(ctx, &turingpb.GetConfigRequest{})
		if err != nil {
			// Fallback to a reasonable default if config can't be fetched
			chatModel = "mistral:7b"
		} else {
			chatModel = configResp.DefaultModel
		}
	}

	// Single message mode
	if len(args) > 0 {
		message := strings.Join(args, " ")
		return sendChatMessageGRPC(ctx, client, message)
	}

	// Interactive mode
	return runInteractiveChatGRPC(ctx, client)
}

func sendChatMessageGRPC(ctx context.Context, client turingpb.TuringServiceClient, message string) error {
	messages := []*turingpb.Message{
		{Role: "user", Content: message},
	}

	if chatSystem != "" {
		messages = append([]*turingpb.Message{
			{Role: "system", Content: chatSystem},
		}, messages...)
	}

	req := &turingpb.ChatRequest{
		Messages:    messages,
		Model:       chatModel,
		MaxTokens:   int32(chatMaxTokens),
		Temperature: float32(chatTemperature),
	}

	if chatStream {
		return streamChatResponseGRPC(ctx, client, req)
	}

	grpcCtx, cancel := context.WithTimeout(ctx, gRPCTimeout)
	defer cancel()

	resp, err := client.Chat(grpcCtx, req)
	if err != nil {
		return fmt.Errorf("Chat-Fehler: %v", err)
	}

	fmt.Println(resp.Content)
	return nil
}

func streamChatResponseGRPC(ctx context.Context, client turingpb.TuringServiceClient, req *turingpb.ChatRequest) error {
	grpcCtx, cancel := context.WithTimeout(ctx, gRPCTimeout)
	defer cancel()

	stream, err := client.StreamChat(grpcCtx, req)
	if err != nil {
		return fmt.Errorf("Streaming-Fehler: %v", err)
	}

	for {
		chunk, err := stream.Recv()
		if err == io.EOF {
			fmt.Println()
			return nil
		}
		if err != nil {
			return fmt.Errorf("Stream-Fehler: %v", err)
		}

		fmt.Print(chunk.Delta)
		if chunk.Done {
			fmt.Println()
			return nil
		}
	}
}

func runInteractiveChatGRPC(ctx context.Context, client turingpb.TuringServiceClient) error {
	fmt.Println("meinDENKWERK Chat (gRPC)")
	fmt.Println("========================")
	fmt.Printf("Modell: %s\n", chatModel)
	fmt.Println("Tippe 'exit' oder 'quit' zum Beenden, 'clear' zum Zurücksetzen")
	fmt.Println()

	var history []*turingpb.Message

	if chatSystem != "" {
		history = append(history, &turingpb.Message{
			Role:    "system",
			Content: chatSystem,
		})
	}

	scanner := bufio.NewScanner(os.Stdin)

	for {
		fmt.Print("Du: ")
		if !scanner.Scan() {
			break
		}

		input := strings.TrimSpace(scanner.Text())
		if input == "" {
			continue
		}

		switch strings.ToLower(input) {
		case "exit", "quit", "q":
			fmt.Println("Auf Wiedersehen!")
			return nil
		case "clear":
			history = history[:0]
			if chatSystem != "" {
				history = append(history, &turingpb.Message{
					Role:    "system",
					Content: chatSystem,
				})
			}
			fmt.Println("[Chat zurückgesetzt]")
			continue
		case "help", "?":
			printChatHelp()
			continue
		}

		// Handle special commands
		if strings.HasPrefix(input, "/model ") {
			chatModel = strings.TrimPrefix(input, "/model ")
			fmt.Printf("[Modell gewechselt zu: %s]\n", chatModel)
			continue
		}

		// Add user message to history
		history = append(history, &turingpb.Message{
			Role:    "user",
			Content: input,
		})

		req := &turingpb.ChatRequest{
			Messages:    history,
			Model:       chatModel,
			MaxTokens:   int32(chatMaxTokens),
			Temperature: float32(chatTemperature),
		}

		fmt.Print("\nAssistent: ")

		if chatStream {
			var fullResponse strings.Builder

			grpcCtx, cancel := context.WithTimeout(ctx, gRPCTimeout)
			stream, err := client.StreamChat(grpcCtx, req)
			if err != nil {
				cancel()
				fmt.Printf("\n[Fehler: %v]\n", err)
				continue
			}

			for {
				chunk, err := stream.Recv()
				if err == io.EOF {
					break
				}
				if err != nil {
					fmt.Printf("\n[Stream-Fehler: %v]\n", err)
					break
				}

				fmt.Print(chunk.Delta)
				fullResponse.WriteString(chunk.Delta)

				if chunk.Done {
					break
				}
			}
			cancel()

			// Add assistant response to history
			history = append(history, &turingpb.Message{
				Role:    "assistant",
				Content: fullResponse.String(),
			})
		} else {
			grpcCtx, cancel := context.WithTimeout(ctx, gRPCTimeout)
			resp, err := client.Chat(grpcCtx, req)
			cancel()

			if err != nil {
				fmt.Printf("\n[Fehler: %v]\n", err)
				continue
			}

			fmt.Print(resp.Content)
			history = append(history, &turingpb.Message{
				Role:    "assistant",
				Content: resp.Content,
			})
		}

		fmt.Println()
	}

	return scanner.Err()
}

// runChatDirect uses Ollama directly without gRPC
func runChatDirect(ctx context.Context, args []string) error {
	client := ollama.NewClient(ollama.Config{
		BaseURL: getOllamaURL(),
	})

	if err := client.Ping(ctx); err != nil {
		return fmt.Errorf("Ollama nicht erreichbar: %v\nStarte Ollama mit: ollama serve", err)
	}

	// Use default model if not specified
	if chatModel == "" {
		chatModel = "mistral:7b"
	}

	if len(args) > 0 {
		message := strings.Join(args, " ")
		return sendChatMessageDirect(ctx, client, message)
	}

	return runInteractiveChatDirect(ctx, client)
}

func sendChatMessageDirect(ctx context.Context, client *ollama.Client, message string) error {
	messages := []ollama.ChatMessage{
		{Role: "user", Content: message},
	}

	if chatSystem != "" {
		messages = append([]ollama.ChatMessage{
			{Role: "system", Content: chatSystem},
		}, messages...)
	}

	options := make(map[string]interface{})
	if chatMaxTokens > 0 {
		options["num_predict"] = chatMaxTokens
	}
	if chatTemperature > 0 {
		options["temperature"] = chatTemperature
	}

	req := &ollama.ChatRequest{
		Model:    chatModel,
		Messages: messages,
		Options:  options,
	}

	if chatStream {
		respCh, errCh := client.ChatStream(ctx, req)

		for {
			select {
			case resp, ok := <-respCh:
				if !ok {
					fmt.Println()
					return nil
				}
				fmt.Print(resp.Message.Content)
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
	}

	resp, err := client.Chat(ctx, req)
	if err != nil {
		return fmt.Errorf("Chat-Fehler: %v", err)
	}

	fmt.Println(resp.Message.Content)
	return nil
}

func runInteractiveChatDirect(ctx context.Context, client *ollama.Client) error {
	fmt.Println("meinDENKWERK Chat (Direkt)")
	fmt.Println("==========================")
	fmt.Printf("Modell: %s\n", chatModel)
	fmt.Println("Tippe 'exit' oder 'quit' zum Beenden, 'clear' zum Zurücksetzen")
	fmt.Println()

	var history []ollama.ChatMessage

	if chatSystem != "" {
		history = append(history, ollama.ChatMessage{
			Role:    "system",
			Content: chatSystem,
		})
	}

	scanner := bufio.NewScanner(os.Stdin)

	for {
		fmt.Print("Du: ")
		if !scanner.Scan() {
			break
		}

		input := strings.TrimSpace(scanner.Text())
		if input == "" {
			continue
		}

		switch strings.ToLower(input) {
		case "exit", "quit", "q":
			fmt.Println("Auf Wiedersehen!")
			return nil
		case "clear":
			history = history[:0]
			if chatSystem != "" {
				history = append(history, ollama.ChatMessage{
					Role:    "system",
					Content: chatSystem,
				})
			}
			fmt.Println("[Chat zurückgesetzt]")
			continue
		case "help", "?":
			printChatHelp()
			continue
		}

		if strings.HasPrefix(input, "/model ") {
			chatModel = strings.TrimPrefix(input, "/model ")
			fmt.Printf("[Modell gewechselt zu: %s]\n", chatModel)
			continue
		}

		history = append(history, ollama.ChatMessage{
			Role:    "user",
			Content: input,
		})

		options := make(map[string]interface{})
		if chatMaxTokens > 0 {
			options["num_predict"] = chatMaxTokens
		}
		if chatTemperature > 0 {
			options["temperature"] = chatTemperature
		}

		req := &ollama.ChatRequest{
			Model:    chatModel,
			Messages: history,
			Options:  options,
		}

		fmt.Print("\nAssistent: ")

		if chatStream {
			var fullResponse strings.Builder
			respCh, errCh := client.ChatStream(ctx, req)

		streamLoop:
			for {
				select {
				case resp, ok := <-respCh:
					if !ok {
						break streamLoop
					}
					fmt.Print(resp.Message.Content)
					fullResponse.WriteString(resp.Message.Content)
					if resp.Done {
						break streamLoop
					}
				case err, ok := <-errCh:
					if ok && err != nil {
						fmt.Printf("\n[Fehler: %v]\n", err)
						break streamLoop
					}
				case <-ctx.Done():
					return ctx.Err()
				}
			}

			history = append(history, ollama.ChatMessage{
				Role:    "assistant",
				Content: fullResponse.String(),
			})
		} else {
			resp, err := client.Chat(ctx, req)
			if err != nil {
				fmt.Printf("\n[Fehler: %v]\n", err)
				continue
			}
			fmt.Print(resp.Message.Content)
			history = append(history, ollama.ChatMessage{
				Role:    "assistant",
				Content: resp.Message.Content,
			})
		}

		fmt.Println()
	}

	return scanner.Err()
}

func printChatHelp() {
	fmt.Print(`
Befehle:
  exit, quit, q  - Chat beenden
  clear          - Chat-Verlauf löschen
  /model <name>  - Modell wechseln
  help, ?        - Diese Hilfe anzeigen
`)
}

func getOllamaURL() string {
	if url := os.Getenv("OLLAMA_URL"); url != "" {
		return url
	}
	return "http://localhost:11434"
}
