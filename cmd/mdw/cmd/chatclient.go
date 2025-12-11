// ============================================================================
// meinDENKWERK (mDW) - Lokale KI-Plattform
// ============================================================================
//
// Package:     cmd
// Description: CLI command for mDW ChatClient TUI
// Author:      Mike Stoffels with Claude
// Created:     2025-12-07
// License:     MIT
// ============================================================================

package cmd

import (
	"github.com/msto63/mDW/internal/tui/chatclient"
	"github.com/spf13/cobra"
)

var (
	chatClientModel          string
	chatClientTuringAddr     string
	chatClientAristotelesAddr string
)

var chatClientCmd = &cobra.Command{
	Use:     "chatclient",
	Aliases: []string{"chat-client", "cc-chat"},
	Short:   "Startet den mDW Chat Client",
	Long: `Startet den interaktiven mDW Chat Client.

Der Chat Client bietet eine elegante Terminal-UI zum Chatten
mit KI-Modellen im ChatGPT-Stil:

  - Auswahl verschiedener LLM-Modelle
  - ChatGPT-ähnliche Oberfläche
  - Echtzeit-Streaming der Antworten
  - Unterstützung für Turing-Service und Ollama-Fallback

Tastenkürzel:
  Enter       Nachricht senden
  F2          Modell wählen
  ↑/↓         Im Menü navigieren
  Ctrl+L      Chat leeren
  PgUp/PgDn   Scrollen
  Ctrl+C      Beenden`,
	RunE: runChatClient,
}

func init() {
	rootCmd.AddCommand(chatClientCmd)

	chatClientCmd.Flags().StringVarP(&chatClientModel, "model", "m", "mistral:7b",
		"LLM-Modell für den Chat")
	chatClientCmd.Flags().StringVar(&chatClientTuringAddr, "turing-addr", "localhost:9200",
		"Adresse des Turing-Service")
	chatClientCmd.Flags().StringVar(&chatClientAristotelesAddr, "aristoteles-addr", "localhost:9160",
		"Adresse des Aristoteles-Service")
}

func runChatClient(cmd *cobra.Command, args []string) error {
	cfg := chatclient.Config{
		TuringAddr:      chatClientTuringAddr,
		AristotelesAddr: chatClientAristotelesAddr,
		Model:           chatClientModel,
	}

	return chatclient.Run(cfg)
}
