// ============================================================================
// meinDENKWERK (mDW) - Lokale KI-Plattform
// ============================================================================
//
// Package:     ui
// Description: Popup notifications using native macOS (no Fyne dependency)
// Author:      Mike Stoffels with Claude
// Created:     2025-12-07
// License:     MIT
// ============================================================================

package ui

import (
	"fmt"
	"os/exec"
	"runtime"
	"sync"
)

// PopupWindow represents the notification system (no actual window needed)
type PopupWindow struct {
	mu            sync.RWMutex
	currentStatus string
	lastUserText  string
	lastResponse  string
	width         float32
	height        float32
}

// PopupConfig holds popup configuration
type PopupConfig struct {
	Width  float32
	Height float32
}

// DefaultPopupConfig returns default popup configuration
func DefaultPopupConfig() PopupConfig {
	return PopupConfig{
		Width:  450,
		Height: 350,
	}
}

// NewPopupWindow creates a new popup window
func NewPopupWindow(cfg PopupConfig) *PopupWindow {
	return &PopupWindow{
		width:  cfg.Width,
		height: cfg.Height,
	}
}

// Initialize initializes the popup (no-op without Fyne)
func (p *PopupWindow) Initialize() {
	// No initialization needed without Fyne
}

// GetApp returns nil (no Fyne app)
func (p *PopupWindow) GetApp() interface{} {
	return nil
}

// Show shows a notification (uses macOS notification center)
func (p *PopupWindow) Show() {
	// No-op - we use terminal output instead
}

// Hide hides the popup
func (p *PopupWindow) Hide() {
	// No-op
}

// SetTitle sets the title (no-op)
func (p *PopupWindow) SetTitle(title string) {
	// Output to terminal instead
	fmt.Printf("\n=== %s ===\n", title)
}

// SetContent sets the content (outputs to terminal)
func (p *PopupWindow) SetContent(text string) {
	// Content is already printed to terminal in app.go
}

// AppendContent appends content
func (p *PopupWindow) AppendContent(text string) {
	// Content is already printed to terminal in app.go
}

// SetStatus sets the status
func (p *PopupWindow) SetStatus(status string) {
	p.mu.Lock()
	p.currentStatus = status
	p.mu.Unlock()

	// Show status in terminal
	fmt.Printf("\r[Status] %s", status)
}

// ShowSpinner shows activity (no-op)
func (p *PopupWindow) ShowSpinner() {
	// No-op
}

// HideSpinner hides activity (no-op)
func (p *PopupWindow) HideSpinner() {
	// No-op
}

// Clear clears content
func (p *PopupWindow) Clear() {
	p.mu.Lock()
	p.currentStatus = ""
	p.lastUserText = ""
	p.lastResponse = ""
	p.mu.Unlock()
}

// ShowListening shows the listening state
func (p *PopupWindow) ShowListening() {
	fmt.Println("\n🎤 Aufnahme gestartet - Sprechen Sie...")
	if runtime.GOOS == "darwin" {
		// Show native notification
		showMacNotification("mDW Voice Assistant", "Aufnahme läuft...", "")
	}
}

// ShowProcessing shows the processing state
func (p *PopupWindow) ShowProcessing(userText string) {
	p.mu.Lock()
	p.lastUserText = userText
	p.mu.Unlock()

	fmt.Println("\n⏳ Verarbeite Anfrage...")
}

// ShowResponse shows the response
func (p *PopupWindow) ShowResponse(userText, response string) {
	p.mu.Lock()
	p.lastUserText = userText
	p.lastResponse = response
	p.mu.Unlock()

	// Response is already printed in app.go
	// Show completion notification on macOS
	if runtime.GOOS == "darwin" {
		// Truncate for notification
		truncated := response
		if len(truncated) > 100 {
			truncated = truncated[:100] + "..."
		}
		showMacNotification("mDW Antwort", truncated, "")
	}
}

// ShowError shows an error
func (p *PopupWindow) ShowError(err error) {
	fmt.Printf("\n❌ Fehler: %v\n", err)
	if runtime.GOOS == "darwin" {
		showMacNotification("mDW Fehler", err.Error(), "")
	}
}

// Run is a no-op (no Fyne event loop)
func (p *PopupWindow) Run() {
	// No-op - systray handles the event loop
}

// Quit is a no-op
func (p *PopupWindow) Quit() {
	// No-op
}

// IsVisible returns false (no window)
func (p *PopupWindow) IsVisible() bool {
	return false
}

// showMacNotification shows a macOS notification using osascript
func showMacNotification(title, message, subtitle string) {
	script := fmt.Sprintf(`display notification "%s" with title "%s"`,
		escapeAppleScript(message),
		escapeAppleScript(title))
	if subtitle != "" {
		script = fmt.Sprintf(`display notification "%s" with title "%s" subtitle "%s"`,
			escapeAppleScript(message),
			escapeAppleScript(title),
			escapeAppleScript(subtitle))
	}
	exec.Command("osascript", "-e", script).Run()
}

// escapeAppleScript escapes special characters for AppleScript strings
func escapeAppleScript(s string) string {
	// Replace backslashes first, then quotes
	result := ""
	for _, c := range s {
		switch c {
		case '\\':
			result += "\\\\"
		case '"':
			result += "\\\""
		case '\n':
			result += " "
		default:
			result += string(c)
		}
	}
	return result
}
