// ============================================================================
// meinDENKWERK (mDW) - Lokale KI-Plattform
// ============================================================================
//
// Package:     controlcenter
// Description: Styles for the Control Center TUI
// Author:      Mike Stoffels with Claude
// Created:     2025-12-06
// License:     MIT
// ============================================================================

package controlcenter

import (
	"github.com/charmbracelet/lipgloss"
)

// Color Palette - Elegant dark theme with accent colors
var (
	// Primary colors
	ColorPrimary     = lipgloss.Color("#8B5CF6") // Violet
	ColorSecondary   = lipgloss.Color("#06B6D4") // Cyan
	ColorAccent      = lipgloss.Color("#F59E0B") // Amber
	ColorSuccess     = lipgloss.Color("#10B981") // Emerald
	ColorWarning     = lipgloss.Color("#F59E0B") // Amber
	ColorError       = lipgloss.Color("#EF4444") // Red
	ColorMuted       = lipgloss.Color("#6B7280") // Gray
	ColorDimmed      = lipgloss.Color("#374151") // Dark Gray

	// Background colors
	ColorBg        = lipgloss.Color("#0F172A") // Slate 900
	ColorBgPanel   = lipgloss.Color("#1E293B") // Slate 800
	ColorBgHover   = lipgloss.Color("#334155") // Slate 700
	ColorBgSelected = lipgloss.Color("#3B0764") // Purple 950

	// Text colors
	ColorText       = lipgloss.Color("#F8FAFC") // Slate 50
	ColorTextMuted  = lipgloss.Color("#94A3B8") // Slate 400
	ColorTextDim    = lipgloss.Color("#64748B") // Slate 500
)

// Component Styles

// Logo/Header styles
var (
	LogoStyle = lipgloss.NewStyle().
		Foreground(ColorPrimary).
		Bold(true).
		MarginBottom(1)

	HeaderStyle = lipgloss.NewStyle().
		Foreground(ColorText).
		Bold(true)

	SubHeaderStyle = lipgloss.NewStyle().
		Foreground(ColorTextMuted).
		Italic(true)

	VersionStyle = lipgloss.NewStyle().
		Foreground(ColorTextDim).
		Italic(true)
)

// Panel/Box styles
var (
	PanelStyle = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ColorDimmed).
		Padding(1, 2).
		MarginBottom(1)

	FocusedPanelStyle = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ColorPrimary).
		Padding(1, 2).
		MarginBottom(1)

	TitlePanelStyle = lipgloss.NewStyle().
		Border(lipgloss.DoubleBorder()).
		BorderForeground(ColorPrimary).
		Padding(0, 2).
		MarginBottom(1)
)

// Status indicator styles
var (
	StatusRunningStyle = lipgloss.NewStyle().
		Foreground(ColorSuccess).
		Bold(true)

	StatusStoppedStyle = lipgloss.NewStyle().
		Foreground(ColorError).
		Bold(true)

	StatusStartingStyle = lipgloss.NewStyle().
		Foreground(ColorWarning).
		Bold(true)

	StatusUnknownStyle = lipgloss.NewStyle().
		Foreground(ColorMuted)

	StatusOKStyle = lipgloss.NewStyle().
		Foreground(ColorSuccess)

	StatusFailStyle = lipgloss.NewStyle().
		Foreground(ColorError)

	StatusCheckingStyle = lipgloss.NewStyle().
		Foreground(ColorAccent)
)

// Service list styles
var (
	ServiceRowStyle = lipgloss.NewStyle().
		Padding(0, 1)

	ServiceSelectedStyle = lipgloss.NewStyle().
		Background(ColorBgSelected).
		Foreground(ColorText).
		Bold(true).
		Padding(0, 1)

	ServiceNameStyle = lipgloss.NewStyle().
		Foreground(ColorText).
		Width(20)

	ServicePortStyle = lipgloss.NewStyle().
		Foreground(ColorTextMuted).
		Width(10)
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

// Loading styles
var (
	SpinnerStyle = lipgloss.NewStyle().
		Foreground(ColorPrimary)
)

// Dependency check styles
var (
	DependencyStyle = lipgloss.NewStyle().
		Padding(0, 1)

	DependencyNameStyle = lipgloss.NewStyle().
		Foreground(ColorText).
		Width(20)
)

// ASCII Art Logo
const Logo = `
                ______  _    _                       _
               |  ___ \| |  | |                     | |
  _ __ ___   __|  | | || |_ | | _____  _ __ ___  ___| | __
 | '_ ' _ \ / _ |  | | || __|| |/ / _ \| '__/ _ \/ __| |/ /
 | | | | | |  __| |__| || |_||   <  __/| | |  __/\__ \   <
 |_| |_| |_|\___|______/ \__||_|\_\___||_|  \___||___/_|\_\
`

const LogoSmall = `
  ┏┳┓┳┓┓ ┏
  ┃┃┃┃┃┃┃┃
  ┛ ┗┻┛┗┻┛
`

const LogoCompact = "mDW ControlCenter"

// Status icons
const (
	IconRunning  = "●"
	IconStopped  = "○"
	IconStarting = "◐"
	IconError    = "✗"
	IconOK       = "✓"
	IconWarning  = "⚠"
	IconArrow    = "→"
	IconBullet   = "•"
	IconChecked  = "[✓]"
	IconUnchecked = "[ ]"
	IconSpinner  = "◌"
)

// RenderKeyHint renders a keyboard shortcut hint
func RenderKeyHint(key, description string) string {
	return HelpKeyStyle.Render(key) + " " + HelpDescStyle.Render(description)
}
