package cmd

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/msto63/mDW/internal/tui"
	"github.com/spf13/cobra"
)

var tuiCmd = &cobra.Command{
	Use:   "tui",
	Short: "Startet die interaktive TUI",
	Long: `Startet die Terminal User Interface (TUI) von meinDENKWERK.

Die TUI bietet eine interaktive Oberfläche für:
  - Chat mit LLM
  - RAG-Suche
  - Agent-Ausführung
  - Service-Status

Navigation:
  Tab       - Zwischen Ansichten wechseln
  Enter     - Nachricht senden
  Ctrl+L    - Chat leeren
  Ctrl+C    - Beenden`,
	RunE: runTUI,
}

func init() {
	rootCmd.AddCommand(tuiCmd)
}

func runTUI(cmd *cobra.Command, args []string) error {
	p := tea.NewProgram(
		tui.NewModel(),
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "TUI Fehler: %v\n", err)
		return err
	}

	return nil
}
