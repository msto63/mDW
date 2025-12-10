package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	cfgFile string
	verbose bool
)

var rootCmd = &cobra.Command{
	Use:   "mdw",
	Short: "meinDENKWERK - Lokale AI-Plattform",
	Long: `meinDENKWERK ist eine leichtgewichtige, lokal installierbare
AI-Plattform für den Einzelarbeitsplatz.

Services:
  kant     - API Gateway (HTTP/SSE)
  russell  - Service Discovery & Orchestration
  turing   - LLM Management
  hypatia  - RAG Service
  leibniz  - Agentic AI
  babbage  - NLP Service
  bayes    - Logging Service`,
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "Config-Datei (default: ./configs/config.toml)")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Verbose Output")
}

func printError(msg string, err error) {
	fmt.Fprintf(os.Stderr, "Fehler: %s: %v\n", msg, err)
}
