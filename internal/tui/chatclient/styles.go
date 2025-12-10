// ============================================================================
// meinDENKWERK (mDW) - Lokale KI-Plattform
// ============================================================================
//
// Package:     chatclient
// Description: Styles for the ChatClient TUI with ChatGPT-like appearance
// Author:      Mike Stoffels with Claude
// Created:     2025-12-07
// License:     MIT
// ============================================================================

package chatclient

import (
	"github.com/charmbracelet/lipgloss"
)

// Color Palette - Same as ControlCenter for consistency
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
	ColorBg          = lipgloss.Color("#0F172A") // Slate 900
	ColorBgPanel     = lipgloss.Color("#1E293B") // Slate 800
	ColorBgHover     = lipgloss.Color("#334155") // Slate 700
	ColorBgSelected  = lipgloss.Color("#3B0764") // Purple 950
	ColorBgUser      = lipgloss.Color("#1E3A5F") // User message background
	ColorBgAssistant = lipgloss.Color("#1E293B") // Assistant message background

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

// Chat message styles - ChatGPT-like bubbles
var (
	UserMessageStyle = lipgloss.NewStyle().
				Foreground(ColorText).
				Background(ColorBgUser).
				Padding(1, 2).
				MarginBottom(1).
				BorderStyle(lipgloss.RoundedBorder()).
				BorderForeground(ColorSecondary)

	AssistantMessageStyle = lipgloss.NewStyle().
				Foreground(ColorText).
				Background(ColorBgAssistant).
				Padding(1, 2).
				MarginBottom(1).
				BorderStyle(lipgloss.RoundedBorder()).
				BorderForeground(ColorDimmed)

	SystemMessageStyle = lipgloss.NewStyle().
				Foreground(ColorTextMuted).
				Italic(true).
				Padding(0, 2).
				MarginBottom(1)

	ErrorMessageStyle = lipgloss.NewStyle().
				Foreground(ColorError).
				Padding(0, 2).
				MarginBottom(1)

	RoleLabelUserStyle = lipgloss.NewStyle().
				Foreground(ColorSecondary).
				Bold(true)

	RoleLabelAssistantStyle = lipgloss.NewStyle().
				Foreground(ColorPrimary).
				Bold(true)
)

// Panel/Box styles
var (
	ChatPanelStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorDimmed).
			Padding(0, 1)

	FocusedChatPanelStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(ColorPrimary).
				Padding(0, 1)

	ModelSelectorStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(ColorDimmed).
				Padding(0, 1).
				MarginBottom(1)

	FocusedModelSelectorStyle = lipgloss.NewStyle().
					Border(lipgloss.DoubleBorder()).
					BorderForeground(ColorAccent).
					Background(ColorBgPanel).
					Foreground(ColorText).
					Padding(1, 2).
					MarginTop(1).
					MarginBottom(1)
)

// Input styles
var (
	InputStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorDimmed).
			Padding(0, 1)

	FocusedInputStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(ColorPrimary).
				Padding(0, 1)

	PlaceholderStyle = lipgloss.NewStyle().
				Foreground(ColorTextDim).
				Italic(true)
)

// Model selector styles
var (
	ModelItemStyle = lipgloss.NewStyle().
			Foreground(ColorText).
			Padding(0, 1)

	SelectedModelItemStyle = lipgloss.NewStyle().
				Foreground(ColorText).
				Background(ColorBgSelected).
				Bold(true).
				Padding(0, 1)

	ModelLabelStyle = lipgloss.NewStyle().
			Foreground(ColorTextMuted)
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

	StatusLoadingStyle = lipgloss.NewStyle().
				Foreground(ColorAccent)
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

// Loading/Spinner styles
var (
	SpinnerStyle = lipgloss.NewStyle().
			Foreground(ColorPrimary)

	ThinkingStyle = lipgloss.NewStyle().
			Foreground(ColorTextMuted).
			Italic(true)
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
	IconUser      = "  "
	IconAssistant = "  "
	IconSystem    = "  "
	IconModel     = "  "
	IconOnline    = "  "
	IconOffline   = "  "
	IconLoading   = "  "
	IconSend      = "  "
	IconClear     = "  "
	IconArrowDown = "  "
	IconCheck     = "  "
)

// Logo
const Logo = "mDW ChatClient"

// RenderKeyHint renders a keyboard shortcut hint
func RenderKeyHint(key, description string) string {
	return HelpKeyStyle.Render(key) + " " + HelpDescStyle.Render(description)
}

// RenderUserLabel renders the user role label
func RenderUserLabel() string {
	return RoleLabelUserStyle.Render(IconUser + "Du")
}

// RenderAssistantLabel renders the assistant role label
func RenderAssistantLabel(model string) string {
	return RoleLabelAssistantStyle.Render(IconAssistant + model)
}

// RenderModelBadge renders a model badge
func RenderModelBadge(model string, selected bool) string {
	if selected {
		return SelectedModelItemStyle.Render(IconCheck + model)
	}
	return ModelItemStyle.Render("  " + model)
}
