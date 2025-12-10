package cmd

import (
	"context"
	"fmt"
	"strings"

	commonpb "github.com/msto63/mDW/api/gen/common"
	turingpb "github.com/msto63/mDW/api/gen/turing"
	"github.com/msto63/mDW/internal/turing/ollama"
	"github.com/spf13/cobra"
)

var modelsDirect bool

var modelsCmd = &cobra.Command{
	Use:   "models",
	Short: "LLM-Modelle verwalten",
	Long: `Zeigt verfügbare LLM-Modelle an und ermöglicht deren Verwaltung.

Beispiele:
  mdw models                    # Alle Modelle anzeigen
  mdw models --direct           # Direkt von Ollama
  mdw models pull llama3.2     # Modell herunterladen`,
	RunE: runModels,
}

var modelsPullCmd = &cobra.Command{
	Use:   "pull <modell>",
	Short: "Modell herunterladen",
	Long: `Lädt ein LLM-Modell von Ollama herunter.

Beispiele:
  mdw models pull llama3.2
  mdw models pull nomic-embed-text
  mdw models pull codellama`,
	Args: cobra.ExactArgs(1),
	RunE: runModelsPull,
}

func init() {
	rootCmd.AddCommand(modelsCmd)
	modelsCmd.AddCommand(modelsPullCmd)

	modelsCmd.Flags().BoolVar(&modelsDirect, "direct", false, "Direkt von Ollama abfragen")
}

func runModels(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	if modelsDirect {
		return runModelsDirect(ctx)
	}

	return runModelsGRPC(ctx)
}

func runModelsGRPC(ctx context.Context) error {
	addrs := DefaultServiceAddresses()
	client, conn, err := NewTuringClient(addrs.Turing)
	if err != nil {
		return fmt.Errorf("Turing-Service nicht erreichbar: %v\nStarte den Service mit: mdw serve turing", err)
	}
	defer conn.Close()

	grpcCtx, cancel := context.WithTimeout(ctx, gRPCTimeout)
	defer cancel()

	resp, err := client.ListModels(grpcCtx, &commonpb.Empty{})
	if err != nil {
		return fmt.Errorf("Fehler beim Laden der Modelle: %v", err)
	}

	fmt.Println("Verfügbare LLM-Modelle (via Turing)")
	fmt.Println("===================================")
	fmt.Println()

	if len(resp.Models) == 0 {
		fmt.Println("Keine Modelle installiert.")
		fmt.Println()
		fmt.Println("Empfohlene Modelle:")
		fmt.Println("  mdw models pull llama3.2        # Allzweck-Modell")
		fmt.Println("  mdw models pull nomic-embed-text # Embeddings")
		fmt.Println("  mdw models pull codellama       # Code-Generierung")
		return nil
	}

	fmt.Printf("%-30s %-15s %-10s\n", "MODELL", "GRÖSSE", "FAMILIE")
	fmt.Println(strings.Repeat("-", 60))

	for _, m := range resp.Models {
		size := formatSizeBytes(m.Size)
		provider := m.Provider
		if provider == "" {
			provider = "-"
		}
		fmt.Printf("%-30s %-15s %-10s\n", m.Name, size, provider)
	}

	fmt.Println()
	fmt.Printf("Gesamt: %d Modell(e)\n", len(resp.Models))

	return nil
}

func runModelsDirect(ctx context.Context) error {
	client := ollama.NewClient(ollama.DefaultConfig())

	if err := client.Ping(ctx); err != nil {
		return fmt.Errorf("Ollama nicht erreichbar: %v\nStarte mit: ollama serve", err)
	}

	models, err := client.ListModels(ctx)
	if err != nil {
		return fmt.Errorf("Fehler beim Laden der Modelle: %v", err)
	}

	fmt.Println("Verfügbare LLM-Modelle (Direkt)")
	fmt.Println("===============================")
	fmt.Println()

	if len(models.Models) == 0 {
		fmt.Println("Keine Modelle installiert.")
		fmt.Println()
		fmt.Println("Empfohlene Modelle:")
		fmt.Println("  mdw models pull llama3.2        # Allzweck-Modell")
		fmt.Println("  mdw models pull nomic-embed-text # Embeddings")
		fmt.Println("  mdw models pull codellama       # Code-Generierung")
		return nil
	}

	fmt.Printf("%-30s %-15s %-10s\n", "MODELL", "GRÖSSE", "FAMILIE")
	fmt.Println(strings.Repeat("-", 60))

	for _, m := range models.Models {
		size := formatSize(m.Size)
		family := m.Details.Family
		if family == "" {
			family = "-"
		}
		fmt.Printf("%-30s %-15s %-10s\n", m.Name, size, family)
	}

	fmt.Println()
	fmt.Printf("Gesamt: %d Modell(e)\n", len(models.Models))

	return nil
}

func runModelsPull(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	modelName := args[0]

	fmt.Printf("Lade Modell: %s\n", modelName)
	fmt.Println("Dies kann einige Minuten dauern...")
	fmt.Println()

	// Try via Turing service first
	addrs := DefaultServiceAddresses()
	client, conn, err := NewTuringClient(addrs.Turing)
	if err == nil {
		defer conn.Close()

		grpcCtx, cancel := context.WithTimeout(ctx, 30*60*1000) // 30 min for large models
		defer cancel()

		_, err := client.PullModel(grpcCtx, &turingpb.PullModelRequest{
			Name: modelName,
		})
		if err == nil {
			fmt.Println("Modell erfolgreich geladen.")
			return nil
		}
		fmt.Printf("Turing-Service konnte Modell nicht laden: %v\n", err)
	}

	// Fallback hint
	fmt.Println("Nutze 'ollama pull " + modelName + "' für Fortschrittsanzeige.")

	return nil
}

func formatSize(bytes int64) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
	)

	switch {
	case bytes >= GB:
		return fmt.Sprintf("%.1f GB", float64(bytes)/GB)
	case bytes >= MB:
		return fmt.Sprintf("%.1f MB", float64(bytes)/MB)
	case bytes >= KB:
		return fmt.Sprintf("%.1f KB", float64(bytes)/KB)
	default:
		return fmt.Sprintf("%d B", bytes)
	}
}

func formatSizeBytes(bytes int64) string {
	return formatSize(bytes)
}
