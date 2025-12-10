// ============================================================================
// meinDENKWERK (mDW) - Lokale KI-Plattform
// ============================================================================
//
// Package:     cmd
// Description: CLI command for mDW Control Center TUI
// Author:      Mike Stoffels with Claude
// Created:     2025-12-06
// License:     MIT
// ============================================================================

package cmd

import (
	"github.com/msto63/mDW/internal/tui/controlcenter"
	"github.com/spf13/cobra"
)

var controlCenterCmd = &cobra.Command{
	Use:     "controlcenter",
	Aliases: []string{"cc", "control"},
	Short:   "Startet das mDW Control Center",
	Long: `Startet das interaktive mDW Control Center.

Das Control Center bietet eine elegante Terminal-UI zur Verwaltung
aller mDW Services:

  - Echtzeit-Statusanzeige aller Services
  - Start/Stop einzelner oder aller Services
  - Dependency-Check beim Start
  - Tastenkürzel für schnelle Aktionen

Tastenkürzel:
  a        Alle Services starten
  s        Alle Services stoppen
  Enter    Ausgewählten Service starten/stoppen
  j/k      Navigation
  r        Status aktualisieren
  d        Dependencies prüfen
  ?        Hilfe anzeigen
  q        Beenden`,
	RunE: runControlCenter,
}

func init() {
	rootCmd.AddCommand(controlCenterCmd)
}

func runControlCenter(cmd *cobra.Command, args []string) error {
	return controlcenter.Run()
}
