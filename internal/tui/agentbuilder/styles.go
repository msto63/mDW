// ============================================================================
// meinDENKWERK (mDW) - Lokale KI-Plattform
// ============================================================================
//
// Package:     agentbuilder
// Description: Styles for the Agent Builder TUI
// Author:      Mike Stoffels with Claude
// Created:     2025-12-07
// License:     MIT
// ============================================================================

package agentbuilder

import (
	"github.com/charmbracelet/lipgloss"
)

// Color Palette - Consistent with other TUI components
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

// Panel styles
var (
	PanelStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorDimmed).
			Padding(0, 1)

	FocusedPanelStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(ColorPrimary).
				Padding(0, 1)

	TitlePanelStyle = lipgloss.NewStyle().
			Border(lipgloss.DoubleBorder()).
			BorderForeground(ColorPrimary).
			Padding(0, 2).
			MarginBottom(1)
)

// List styles
var (
	ListItemStyle = lipgloss.NewStyle().
			Foreground(ColorText).
			PaddingLeft(2)

	SelectedItemStyle = lipgloss.NewStyle().
				Foreground(ColorPrimary).
				Bold(true).
				PaddingLeft(2)

	ListTitleStyle = lipgloss.NewStyle().
			Foreground(ColorSecondary).
			Bold(true).
			MarginBottom(1)
)

// Form/Editor styles
var (
	LabelStyle = lipgloss.NewStyle().
			Foreground(ColorTextMuted).
			Width(16).
			MarginBottom(0)

	InputStyle = lipgloss.NewStyle().
			Foreground(ColorText).
			Background(ColorBgPanel).
			Padding(0, 1)

	FocusedInputStyle = lipgloss.NewStyle().
				Foreground(ColorText).
				Background(ColorBgHover).
				Border(lipgloss.NormalBorder()).
				BorderForeground(ColorPrimary).
				Padding(0, 1)

	// Border styles for text inputs
	InputBorderStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(ColorDimmed).
				Padding(0, 1).
				Width(64)

	InputBorderFocusedStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(ColorPrimary).
				Padding(0, 1).
				Width(64)

	// Border styles for textarea (system prompt)
	TextAreaBorderStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(ColorDimmed).
				Padding(0, 1)

	TextAreaBorderFocusedStyle = lipgloss.NewStyle().
					Border(lipgloss.RoundedBorder()).
					BorderForeground(ColorPrimary).
					Padding(0, 1)

	TextAreaStyle = lipgloss.NewStyle().
			Foreground(ColorText).
			Background(ColorBgPanel).
			Padding(1)

	// Tools field styles
	ToolsFieldStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorDimmed).
			Padding(0, 1).
			Width(64)

	ToolsFieldFocusedStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(ColorPrimary).
				Padding(0, 1).
				Width(64)

	SliderTrackStyle = lipgloss.NewStyle().
				Foreground(ColorDimmed)

	SliderThumbStyle = lipgloss.NewStyle().
				Foreground(ColorPrimary).
				Bold(true)

	SliderValueStyle = lipgloss.NewStyle().
				Foreground(ColorSecondary).
				Bold(true)
)

// Tool picker styles
var (
	ToolItemStyle = lipgloss.NewStyle().
			Foreground(ColorText)

	ToolSelectedStyle = lipgloss.NewStyle().
				Foreground(ColorSuccess).
				Bold(true)

	ToolCategoryStyle = lipgloss.NewStyle().
				Foreground(ColorSecondary).
				Bold(true).
				MarginTop(1).
				MarginBottom(1)

	CheckboxUnchecked = lipgloss.NewStyle().
				Foreground(ColorTextDim).
				Render("[ ]")

	CheckboxChecked = lipgloss.NewStyle().
			Foreground(ColorSuccess).
			Bold(true).
			Render("[x]")
)

// Test view styles
var (
	TestUserStyle = lipgloss.NewStyle().
			Foreground(ColorSecondary).
			Bold(true)

	TestAgentStyle = lipgloss.NewStyle().
			Foreground(ColorPrimary).
			Bold(true)

	TestThinkingStyle = lipgloss.NewStyle().
				Foreground(ColorTextMuted).
				Italic(true)

	TestToolCallStyle = lipgloss.NewStyle().
				Foreground(ColorAccent).
				Bold(true)

	TestToolResultStyle = lipgloss.NewStyle().
				Foreground(ColorSuccess)

	TestErrorStyle = lipgloss.NewStyle().
			Foreground(ColorError).
			Bold(true)

	TestStatsStyle = lipgloss.NewStyle().
			Background(ColorBgPanel).
			Foreground(ColorText).
			Padding(0, 1)
)

// Status styles
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

	StatusRunningStyle = lipgloss.NewStyle().
				Foreground(ColorAccent).
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

// Button styles
var (
	ButtonStyle = lipgloss.NewStyle().
			Background(ColorBgPanel).
			Foreground(ColorText).
			Padding(0, 2).
			MarginRight(1)

	ButtonFocusedStyle = lipgloss.NewStyle().
				Background(ColorPrimary).
				Foreground(ColorText).
				Bold(true).
				Padding(0, 2).
				MarginRight(1)

	ButtonDangerStyle = lipgloss.NewStyle().
				Background(ColorError).
				Foreground(ColorText).
				Bold(true).
				Padding(0, 2).
				MarginRight(1)
)

// Icons
const (
	IconAgent    = "  "
	IconTool     = "  "
	IconTest     = "  "
	IconEdit     = "  "
	IconDelete   = "  "
	IconClone    = "  "
	IconNew      = "  "
	IconSave     = "  "
	IconOnline   = "  "
	IconOffline  = "  "
	IconRunning  = "  "
	IconSuccess  = "  "
	IconError    = "  "
	IconThinking = "  "
)

// Logo
const Logo = "mDW Agent Builder"

// RenderKeyHint renders a keyboard shortcut hint
func RenderKeyHint(key, description string) string {
	return HelpKeyStyle.Render(key) + " " + HelpDescStyle.Render(description)
}

// RenderCheckbox renders a checkbox state
func RenderCheckbox(checked bool) string {
	if checked {
		return CheckboxChecked
	}
	return CheckboxUnchecked
}

// RenderStatusBadge renders a status badge
func RenderStatusBadge(status string) string {
	switch status {
	case "online":
		return StatusOnlineStyle.Render(IconOnline + "Online")
	case "offline":
		return StatusOfflineStyle.Render(IconOffline + "Offline")
	case "running":
		return StatusRunningStyle.Render(IconRunning + "Running")
	default:
		return status
	}
}
