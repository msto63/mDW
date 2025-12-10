package cmd

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	commonpb "github.com/msto63/mDW/api/gen/common"
	leibnizpb "github.com/msto63/mDW/api/gen/leibniz"
	"github.com/msto63/mDW/internal/leibniz/agent"
	"github.com/msto63/mDW/internal/leibniz/service"
	"github.com/msto63/mDW/internal/turing/ollama"
	"github.com/spf13/cobra"
)

var (
	agentMaxSteps int
	agentTimeout  time.Duration
	agentVerbose  bool
	agentDirect   bool
	agentID       string
)

var agentCmd = &cobra.Command{
	Use:   "agent <aufgabe>",
	Short: "Führe eine Aufgabe mit dem AI-Agenten aus",
	Long: `Startet den AI-Agenten, um eine komplexe Aufgabe auszuführen.

Der Agent kann:
- Multi-step Reasoning durchführen
- Tools verwenden (Rechner, Zeitabfrage, etc.)
- MCP-Server für erweiterte Funktionen nutzen

Beispiele:
  mdw agent "Berechne 15% von 250 und addiere 100"
  mdw agent --max-steps 5 "Was ist heute für ein Tag?"
  mdw agent --verbose "Analysiere die aktuelle Systemzeit"
  mdw agent --direct "Aufgabe"  # Direkt ohne Leibniz Service`,
	Args: cobra.MinimumNArgs(1),
	RunE: runAgent,
}

var toolsCmd = &cobra.Command{
	Use:   "tools",
	Short: "Liste verfügbare Agent-Tools",
	Long:  `Zeigt alle verfügbaren Tools für den AI-Agenten an.`,
	RunE:  runListTools,
}

func init() {
	rootCmd.AddCommand(agentCmd)
	rootCmd.AddCommand(toolsCmd)

	agentCmd.Flags().IntVar(&agentMaxSteps, "max-steps", 10, "Maximale Anzahl Schritte")
	agentCmd.Flags().DurationVar(&agentTimeout, "timeout", 2*time.Minute, "Timeout für die Ausführung")
	agentCmd.Flags().BoolVarP(&agentVerbose, "verbose", "v", false, "Zeige detaillierte Schrittinformationen")
	agentCmd.Flags().BoolVar(&agentDirect, "direct", false, "Direkt ohne Leibniz Service")
	agentCmd.Flags().StringVar(&agentID, "agent", "default", "Agent-ID")

	toolsCmd.Flags().BoolVar(&agentDirect, "direct", false, "Direkt ohne Leibniz Service")
}

func runAgent(cmd *cobra.Command, args []string) error {
	ctx, cancel := context.WithTimeout(context.Background(), agentTimeout)
	defer cancel()

	task := strings.Join(args, " ")

	if agentDirect {
		return runAgentDirect(ctx, task)
	}

	return runAgentGRPC(ctx, task)
}

func runAgentGRPC(ctx context.Context, task string) error {
	fmt.Println("meinDENKWERK Agent (gRPC)")
	fmt.Println(strings.Repeat("=", 50))
	fmt.Printf("Aufgabe: %s\n", task)
	fmt.Println()

	addrs := DefaultServiceAddresses()
	client, conn, err := NewLeibnizClient(addrs.Leibniz)
	if err != nil {
		return fmt.Errorf("Leibniz-Service nicht erreichbar: %v\nStarte den Service mit: mdw serve leibniz", err)
	}
	defer conn.Close()

	fmt.Println("Starte Agent...")
	if agentVerbose {
		fmt.Println()
	}

	// Use streaming for verbose mode
	if agentVerbose {
		return runAgentStreamGRPC(ctx, client, task)
	}

	// Non-streaming execution
	grpcCtx, cancel := context.WithTimeout(ctx, gRPCTimeout)
	defer cancel()

	resp, err := client.Execute(grpcCtx, &leibnizpb.ExecuteRequest{
		AgentId: agentID,
		Message: task,
	})
	if err != nil {
		return fmt.Errorf("Agent-Ausführung fehlgeschlagen: %v", err)
	}

	fmt.Println()
	fmt.Printf("Status: %s\n", resp.Status.String())
	fmt.Printf("Iterationen: %d\n", resp.Iterations)

	if len(resp.Actions) > 0 {
		var toolNames []string
		for _, action := range resp.Actions {
			toolNames = append(toolNames, action.Tool)
		}
		fmt.Printf("Verwendete Tools: %s\n", strings.Join(toolNames, ", "))
	}

	if resp.Response != "" {
		fmt.Println()
		fmt.Println("Ergebnis:")
		fmt.Println(strings.Repeat("-", 50))
		fmt.Println(resp.Response)
	}

	return nil
}

func runAgentStreamGRPC(ctx context.Context, client leibnizpb.LeibnizServiceClient, task string) error {
	grpcCtx, cancel := context.WithTimeout(ctx, gRPCTimeout)
	defer cancel()

	stream, err := client.StreamExecute(grpcCtx, &leibnizpb.ExecuteRequest{
		AgentId: agentID,
		Message: task,
	})
	if err != nil {
		return fmt.Errorf("Streaming fehlgeschlagen: %v", err)
	}

	fmt.Println("Schritte:")
	fmt.Println(strings.Repeat("-", 50))

	var toolsUsed []string
	var finalResponse string

	for {
		chunk, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("Stream-Fehler: %v", err)
		}

		switch chunk.Type {
		case leibnizpb.ChunkType_CHUNK_TYPE_THINKING:
			fmt.Printf("\n[Iteration %d] Denke...\n", chunk.Iteration)
			if chunk.Content != "" {
				fmt.Printf("  %s\n", chunk.Content)
			}

		case leibnizpb.ChunkType_CHUNK_TYPE_TOOL_CALL:
			if chunk.Action != nil {
				fmt.Printf("\n[Tool] %s\n", chunk.Action.Tool)
				fmt.Printf("  Input: %s\n", chunk.Action.Input)
				toolsUsed = append(toolsUsed, chunk.Action.Tool)
			}

		case leibnizpb.ChunkType_CHUNK_TYPE_TOOL_RESULT:
			if chunk.Action != nil {
				fmt.Printf("  Output: %s\n", chunk.Action.Output)
			}

		case leibnizpb.ChunkType_CHUNK_TYPE_RESPONSE:
			fmt.Print(chunk.Content)

		case leibnizpb.ChunkType_CHUNK_TYPE_FINAL:
			finalResponse = chunk.Content
		}
	}

	fmt.Println()
	fmt.Println(strings.Repeat("-", 50))
	fmt.Println()

	if len(toolsUsed) > 0 {
		unique := make(map[string]bool)
		var uniqueTools []string
		for _, t := range toolsUsed {
			if !unique[t] {
				unique[t] = true
				uniqueTools = append(uniqueTools, t)
			}
		}
		fmt.Printf("Verwendete Tools: %s\n", strings.Join(uniqueTools, ", "))
	}

	if finalResponse != "" {
		fmt.Println()
		fmt.Println("Ergebnis:")
		fmt.Println(strings.Repeat("-", 50))
		fmt.Println(finalResponse)
	}

	return nil
}

func runListTools(cmd *cobra.Command, args []string) error {
	if agentDirect {
		return runListToolsDirect()
	}

	return runListToolsGRPC()
}

func runListToolsGRPC() error {
	addrs := DefaultServiceAddresses()
	client, conn, err := NewLeibnizClient(addrs.Leibniz)
	if err != nil {
		return fmt.Errorf("Leibniz-Service nicht erreichbar: %v\nStarte den Service mit: mdw serve leibniz", err)
	}
	defer conn.Close()

	ctx, cancel := context.WithTimeout(context.Background(), gRPCTimeout)
	defer cancel()

	resp, err := client.ListTools(ctx, &commonpb.Empty{})
	if err != nil {
		return fmt.Errorf("Tools konnten nicht abgerufen werden: %v", err)
	}

	fmt.Println("Verfügbare Agent-Tools")
	fmt.Println(strings.Repeat("=", 50))

	if len(resp.Tools) == 0 {
		fmt.Println("Keine Tools verfügbar.")
		return nil
	}

	// Group by source
	builtinTools := []*leibnizpb.ToolInfo{}
	mcpTools := []*leibnizpb.ToolInfo{}
	customTools := []*leibnizpb.ToolInfo{}

	for _, tool := range resp.Tools {
		switch tool.Source {
		case leibnizpb.ToolSource_TOOL_SOURCE_BUILTIN:
			builtinTools = append(builtinTools, tool)
		case leibnizpb.ToolSource_TOOL_SOURCE_MCP:
			mcpTools = append(mcpTools, tool)
		case leibnizpb.ToolSource_TOOL_SOURCE_CUSTOM:
			customTools = append(customTools, tool)
		}
	}

	if len(builtinTools) > 0 {
		fmt.Println("\nEingebaute Tools:")
		for _, tool := range builtinTools {
			fmt.Printf("  - %s: %s\n", tool.Name, tool.Description)
		}
	}

	if len(mcpTools) > 0 {
		fmt.Println("\nMCP-Tools:")
		for _, tool := range mcpTools {
			fmt.Printf("  - %s: %s\n", tool.Name, tool.Description)
		}
	}

	if len(customTools) > 0 {
		fmt.Println("\nCustom-Tools:")
		for _, tool := range customTools {
			fmt.Printf("  - %s: %s\n", tool.Name, tool.Description)
		}
	}

	fmt.Printf("\nGesamt: %d Tool(s)\n", len(resp.Tools))

	return nil
}

// Direct mode functions (without gRPC)

func runAgentDirect(ctx context.Context, task string) error {
	fmt.Println("meinDENKWERK Agent (Direkt)")
	fmt.Println(strings.Repeat("=", 50))
	fmt.Printf("Aufgabe: %s\n", task)
	fmt.Println()

	ollamaClient := ollama.NewClient(ollama.DefaultConfig())

	if err := ollamaClient.Ping(ctx); err != nil {
		return fmt.Errorf("Ollama nicht erreichbar: %v", err)
	}

	llmFunc := func(ctx context.Context, messages []agent.Message) (string, error) {
		ollamaMessages := make([]ollama.ChatMessage, len(messages))
		for i, m := range messages {
			ollamaMessages[i] = ollama.ChatMessage{
				Role:    m.Role,
				Content: m.Content,
			}
		}

		resp, err := ollamaClient.Chat(ctx, &ollama.ChatRequest{
			Model:    chatModel,
			Messages: ollamaMessages,
		})
		if err != nil {
			return "", err
		}
		return resp.Message.Content, nil
	}

	svc, err := service.NewService(service.Config{
		MaxSteps: agentMaxSteps,
	})
	if err != nil {
		return err
	}
	svc.SetLLMFunc(llmFunc)

	fmt.Println("Starte Agent...")
	if agentVerbose {
		fmt.Println()
	}

	result, err := svc.Execute(ctx, &service.ExecuteRequest{
		Task:     task,
		MaxSteps: agentMaxSteps,
		Timeout:  agentTimeout,
	})

	if agentVerbose && len(result.Steps) > 0 {
		fmt.Println("Schritte:")
		fmt.Println(strings.Repeat("-", 50))
		for _, step := range result.Steps {
			fmt.Printf("\n[Schritt %d]\n", step.Index+1)
			if step.Thought != "" {
				fmt.Printf("Gedanke: %s\n", step.Thought)
			}
			if step.Action != "" {
				fmt.Printf("Aktion: %s\n", step.Action)
			}
			if step.ToolName != "" {
				fmt.Printf("Tool: %s\n", step.ToolName)
				if step.ToolInput != "" {
					fmt.Printf("  Input: %s\n", step.ToolInput)
				}
				if step.ToolOutput != "" {
					fmt.Printf("  Output: %s\n", step.ToolOutput)
				}
			}
		}
		fmt.Println()
		fmt.Println(strings.Repeat("-", 50))
	}

	fmt.Println()
	fmt.Printf("Status: %s\n", result.Status)
	fmt.Printf("Dauer: %v\n", result.Duration)

	if len(result.ToolsUsed) > 0 {
		fmt.Printf("Verwendete Tools: %s\n", strings.Join(result.ToolsUsed, ", "))
	}

	if result.Error != "" {
		fmt.Printf("\nFehler: %s\n", result.Error)
	}

	if result.Result != "" {
		fmt.Println()
		fmt.Println("Ergebnis:")
		fmt.Println(strings.Repeat("-", 50))
		fmt.Println(result.Result)
	}

	if err != nil {
		return fmt.Errorf("Agent-Ausführung fehlgeschlagen: %v", err)
	}

	return nil
}

func runListToolsDirect() error {
	svc, err := service.NewService(service.DefaultConfig())
	if err != nil {
		return err
	}

	tools := svc.ListTools()

	fmt.Println("Verfügbare Agent-Tools")
	fmt.Println(strings.Repeat("=", 50))

	if len(tools) == 0 {
		fmt.Println("Keine Tools verfügbar.")
		return nil
	}

	builtinTools := []service.ToolInfo{}
	mcpTools := []service.ToolInfo{}

	for _, tool := range tools {
		if tool.Source == "builtin" {
			builtinTools = append(builtinTools, tool)
		} else {
			mcpTools = append(mcpTools, tool)
		}
	}

	if len(builtinTools) > 0 {
		fmt.Println("\nEingebaute Tools:")
		for _, tool := range builtinTools {
			fmt.Printf("  - %s: %s\n", tool.Name, tool.Description)
		}
	}

	if len(mcpTools) > 0 {
		fmt.Println("\nMCP-Tools:")
		for _, tool := range mcpTools {
			fmt.Printf("  - %s [%s]: %s\n", tool.Name, tool.Source, tool.Description)
		}
	}

	fmt.Printf("\nGesamt: %d Tool(s)\n", len(tools))

	return nil
}
