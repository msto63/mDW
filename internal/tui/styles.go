package tui

import (
	"github.com/charmbracelet/lipgloss"
)

// Colors
var (
	colorPrimary   = lipgloss.Color("#7C3AED")
	colorSecondary = lipgloss.Color("#10B981")
	colorAccent    = lipgloss.Color("#F59E0B")
	colorError     = lipgloss.Color("#EF4444")
	colorMuted     = lipgloss.Color("#6B7280")
	colorBg        = lipgloss.Color("#1F2937")
	colorFg        = lipgloss.Color("#F9FAFB")
)

// Styles
var (
	// Title styles
	TitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorPrimary).
			MarginBottom(1)

	SubtitleStyle = lipgloss.NewStyle().
			Foreground(colorMuted).
			Italic(true)

	// Box styles
	BoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorMuted).
			Padding(1, 2)

	FocusedBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorPrimary).
			Padding(1, 2)

	// Message styles
	UserMessageStyle = lipgloss.NewStyle().
				Foreground(colorSecondary).
				Bold(true)

	AssistantMessageStyle = lipgloss.NewStyle().
				Foreground(colorFg)

	SystemMessageStyle = lipgloss.NewStyle().
				Foreground(colorMuted).
				Italic(true)

	ErrorMessageStyle = lipgloss.NewStyle().
				Foreground(colorError)

	// Status styles
	StatusBarStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("#374151")).
			Foreground(colorFg).
			Padding(0, 1)

	StatusOKStyle = lipgloss.NewStyle().
			Foreground(colorSecondary)

	StatusErrorStyle = lipgloss.NewStyle().
				Foreground(colorError)

	// Menu styles
	MenuItemStyle = lipgloss.NewStyle().
			PaddingLeft(2)

	SelectedMenuItemStyle = lipgloss.NewStyle().
				Foreground(colorPrimary).
				Bold(true).
				PaddingLeft(2)

	// Help style
	HelpStyle = lipgloss.NewStyle().
			Foreground(colorMuted).
			MarginTop(1)

	// Input style
	InputStyle = lipgloss.NewStyle().
			Border(lipgloss.NormalBorder()).
			BorderForeground(colorMuted).
			Padding(0, 1)

	FocusedInputStyle = lipgloss.NewStyle().
				Border(lipgloss.NormalBorder()).
				BorderForeground(colorPrimary).
				Padding(0, 1)

	// Tab styles
	TabStyle = lipgloss.NewStyle().
			Padding(0, 2).
			Foreground(colorMuted)

	ActiveTabStyle = lipgloss.NewStyle().
			Padding(0, 2).
			Foreground(colorPrimary).
			Bold(true).
			Underline(true)
)

// Helper functions
func RenderTitle(title string) string {
	return TitleStyle.Render(title)
}

func RenderError(err string) string {
	return ErrorMessageStyle.Render("Fehler: " + err)
}

func RenderHelp(help string) string {
	return HelpStyle.Render(help)
}
