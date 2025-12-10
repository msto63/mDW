// ============================================================================
// meinDENKWERK (mDW) - Lokale KI-Plattform
// ============================================================================
//
// Package:     cmd
// Description: CLI command for mDW LogViewer TUI
// Author:      Mike Stoffels with Claude
// Created:     2025-12-07
// License:     MIT
// ============================================================================

package cmd

import (
	"github.com/msto63/mDW/internal/tui/logviewer"
	"github.com/spf13/cobra"
)

var (
	logViewerBayesAddr   string
	logViewerMaxLogCount int
)

var logViewerCmd = &cobra.Command{
	Use:     "logs",
	Aliases: []string{"logviewer", "log", "bayes-logs"},
	Short:   "Startet den mDW Log Viewer",
	Long: `Startet den interaktiven mDW Log Viewer.

Der Log Viewer zeigt Logs aus dem Bayes-Service in einer
eleganten Terminal-UI an:

  - Echtzeit-Aktualisierung der Logs
  - Filterung nach Log-Level (1-5)
  - Pause/Resume-Funktion
  - Auto-Scroll zum neuesten Log

Tastenkuerzel:
  1-5         Log-Level togglen (1=DEBUG, 2=INFO, 3=WARN, 4=ERROR, 5=FATAL)
  0           Alle Level anzeigen
  p / Space   Pause/Resume
  r           Refresh
  a           Auto-Scroll togglen
  g / G       Zum Anfang / Ende springen
  PgUp/PgDn   Scrollen
  c           Logs leeren
  Ctrl+C      Beenden`,
	RunE: runLogViewer,
}

func init() {
	rootCmd.AddCommand(logViewerCmd)

	logViewerCmd.Flags().StringVar(&logViewerBayesAddr, "bayes-addr", "localhost:9120",
		"Adresse des Bayes-Service")
	logViewerCmd.Flags().IntVar(&logViewerMaxLogCount, "max-logs", 1000,
		"Maximale Anzahl der angezeigten Logs")
}

func runLogViewer(cmd *cobra.Command, args []string) error {
	cfg := logviewer.Config{
		BayesAddr:   logViewerBayesAddr,
		MaxLogCount: logViewerMaxLogCount,
	}

	return logviewer.Run(cfg)
}
