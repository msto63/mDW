// ============================================================================
// meinDENKWERK (mDW) - Lokale KI-Plattform
// ============================================================================
//
// Package:     cmd
// Description: CLI command for mDW Agent Builder TUI
// Author:      Mike Stoffels with Claude
// Created:     2025-12-07
// License:     MIT
// ============================================================================

package cmd

import (
	"github.com/msto63/mDW/internal/tui/agentbuilder"
	"github.com/spf13/cobra"
)

var (
	agentBuilderLeibnizAddr string
	agentBuilderTuringAddr  string
)

var agentBuilderCmd = &cobra.Command{
	Use:     "agents",
	Aliases: []string{"agentbuilder", "agent-builder", "ab"},
	Short:   "Startet den mDW Agent Builder",
	Long: `Startet den interaktiven mDW Agent Builder.

Der Agent Builder ermoeglicht die Erstellung, Verwaltung und
das Testen von KI-Agenten im Leibniz-Service:

  - Agenten erstellen und bearbeiten
  - Alle Parameter konfigurieren
  - Tools auswaehlen
  - Agenten testen

Ansichten:
  - Listen-Ansicht: Uebersicht aller Agenten
  - Editor-Ansicht: Agent bearbeiten
  - Tool-Picker: Tools auswaehlen
  - Test-Ansicht: Agent interaktiv testen

Tastenkuerzel (Listen-Ansicht):
  n           Neuer Agent
  e/Enter     Agent bearbeiten
  d           Agent loeschen
  c           Agent klonen
  t           Agent testen
  1-3         Vorlage laden
  r           Aktualisieren
  q           Beenden

Tastenkuerzel (Editor):
  Tab         Naechstes Feld
  Ctrl+S      Speichern
  Esc         Abbrechen

Tastenkuerzel (Test):
  Enter       Nachricht senden
  Ctrl+R      Konversation zuruecksetzen
  Ctrl+X      Test abbrechen
  Esc         Zurueck`,
	RunE: runAgentBuilder,
}

func init() {
	rootCmd.AddCommand(agentBuilderCmd)

	agentBuilderCmd.Flags().StringVar(&agentBuilderLeibnizAddr, "leibniz-addr", "localhost:9140",
		"Adresse des Leibniz-Service")
	agentBuilderCmd.Flags().StringVar(&agentBuilderTuringAddr, "turing-addr", "localhost:9200",
		"Adresse des Turing-Service")
}

func runAgentBuilder(cmd *cobra.Command, args []string) error {
	cfg := agentbuilder.Config{
		LeibnizAddr: agentBuilderLeibnizAddr,
		TuringAddr:  agentBuilderTuringAddr,
	}

	return agentbuilder.Run(cfg)
}
