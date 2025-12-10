// ============================================================================
// meinDENKWERK (mDW) - Lokale KI-Plattform
// ============================================================================
//
// Package:     logviewer
// Description: Styles for the LogViewer TUI
// Author:      Mike Stoffels with Claude
// Created:     2025-12-07
// License:     MIT
// ============================================================================

package logviewer

import (
	"github.com/charmbracelet/lipgloss"
)

// Color Palette - Same as other TUI components for consistency
var (
	// Primary colors
	ColorPrimary   = lipgloss.Color("#8B5CF6") // Violet
	ColorSecondary = lipgloss.Color("#06B6D4") // Cyan
	ColorAccent    = lipgloss.Color("#F59E0B") // Amber
	ColorSuccess   = lipgloss.Color("#10B981") // Emerald
	ColorWarning   = lipgloss.Color("#F59E0B") // Amber
	ColorError     = lipgloss.Color("#EF4444") // Red
	ColorMuted     = lipgloss.Color("#6B7280") // Gray
	ColorDimmed    = lipgloss.Color("#374151") // Dark Gray

	// Background colors
	ColorBg         = lipgloss.Color("#0F172A") // Slate 900
	ColorBgPanel    = lipgloss.Color("#1E293B") // Slate 800
	ColorBgHover    = lipgloss.Color("#334155") // Slate 700
	ColorBgSelected = lipgloss.Color("#3B0764") // Purple 950

	// Text colors
	ColorText      = lipgloss.Color("#F8FAFC") // Slate 50
	ColorTextMuted = lipgloss.Color("#94A3B8") // Slate 400
	ColorTextDim   = lipgloss.Color("#64748B") // Slate 500

	// Log level colors
	ColorDebug = lipgloss.Color("#94A3B8") // Gray
	ColorInfo  = lipgloss.Color("#06B6D4") // Cyan
	ColorWarn  = lipgloss.Color("#F59E0B") // Amber
	ColorFatal = lipgloss.Color("#DC2626") // Dark Red
)

// Logo/Header styles
var (
	LogoStyle = lipgloss.NewStyle().
			Foreground(ColorPrimary).
			Bold(true)

	HeaderStyle = lipgloss.NewStyle().
			Foreground(ColorText).
			Bold(true)

	SubHeaderStyle = lipgloss.NewStyle().
			Foreground(ColorTextMuted).
			Italic(true)
)

// Log entry styles
var (
	LogEntryBaseStyle = lipgloss.NewStyle().
				Foreground(ColorText)

	LogTimestampStyle = lipgloss.NewStyle().
				Foreground(ColorTextDim)

	LogServiceStyle = lipgloss.NewStyle().
			Foreground(ColorSecondary).
			Bold(true)

	LogMessageStyle = lipgloss.NewStyle().
			Foreground(ColorText)

	// Level-specific styles
	LogLevelDebugStyle = lipgloss.NewStyle().
				Foreground(ColorDebug).
				Bold(true)

	LogLevelInfoStyle = lipgloss.NewStyle().
				Foreground(ColorInfo).
				Bold(true)

	LogLevelWarnStyle = lipgloss.NewStyle().
				Foreground(ColorWarning).
				Bold(true)

	LogLevelErrorStyle = lipgloss.NewStyle().
				Foreground(ColorError).
				Bold(true)

	LogLevelFatalStyle = lipgloss.NewStyle().
				Foreground(ColorFatal).
				Background(lipgloss.Color("#450A0A")).
				Bold(true)
)

// Panel/Box styles
var (
	LogPanelStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorDimmed).
			Padding(0, 1)

	FocusedLogPanelStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(ColorPrimary).
				Padding(0, 1)

	FilterBarStyle = lipgloss.NewStyle().
			Background(ColorBgPanel).
			Foreground(ColorText).
			Padding(0, 1)
)

// Status bar styles
var (
	StatusBarStyle = lipgloss.NewStyle().
			Background(ColorBgPanel).
			Foreground(ColorText).
			Padding(0, 1)

	StatusOnlineStyle = lipgloss.NewStyle().
				Foreground(ColorSuccess).
				Bold(true)

	StatusOfflineStyle = lipgloss.NewStyle().
				Foreground(ColorError).
				Bold(true)

	StatusPausedStyle = lipgloss.NewStyle().
				Foreground(ColorWarning).
				Bold(true)
)

// Help styles
var (
	HelpStyle = lipgloss.NewStyle().
			Foreground(ColorTextMuted).
			MarginTop(1)

	HelpKeyStyle = lipgloss.NewStyle().
			Foreground(ColorPrimary).
			Bold(true)

	HelpDescStyle = lipgloss.NewStyle().
			Foreground(ColorTextMuted)
)

// Filter badge styles
var (
	FilterActiveStyle = lipgloss.NewStyle().
				Foreground(ColorSuccess).
				Bold(true)

	FilterInactiveStyle = lipgloss.NewStyle().
				Foreground(ColorTextDim)
)

// Title panel style
var (
	TitlePanelStyle = lipgloss.NewStyle().
			Border(lipgloss.DoubleBorder()).
			BorderForeground(ColorPrimary).
			Padding(0, 2).
			MarginBottom(1)
)

// Icons
const (
	IconLog     = "  "
	IconDebug   = "  "
	IconInfo    = "  "
	IconWarn    = "  "
	IconError   = "  "
	IconFatal   = "  "
	IconOnline  = "  "
	IconOffline = "  "
	IconPaused  = "  "
	IconFilter  = "  "
	IconService = "  "
)

// Logo
const Logo = "mDW LogViewer"

// RenderKeyHint renders a keyboard shortcut hint
func RenderKeyHint(key, description string) string {
	return HelpKeyStyle.Render(key) + " " + HelpDescStyle.Render(description)
}

// RenderLevelBadge renders a log level badge with appropriate styling
func RenderLevelBadge(level string) string {
	switch level {
	case "DEBUG":
		return LogLevelDebugStyle.Render("[DEBUG]")
	case "INFO":
		return LogLevelInfoStyle.Render("[INFO] ")
	case "WARN", "WARNING":
		return LogLevelWarnStyle.Render("[WARN] ")
	case "ERROR":
		return LogLevelErrorStyle.Render("[ERROR]")
	case "FATAL":
		return LogLevelFatalStyle.Render("[FATAL]")
	default:
		return LogLevelInfoStyle.Render("[" + level + "]")
	}
}

// RenderFilterStatus renders a filter status indicator
func RenderFilterStatus(name string, active bool) string {
	if active {
		return FilterActiveStyle.Render(name)
	}
	return FilterInactiveStyle.Render(name)
}
