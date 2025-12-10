// ============================================================================
// meinDENKWERK (mDW) - Lokale KI-Plattform
// ============================================================================
//
// Package:     ui
// Description: System Tray implementation using fyne.io/systray
// Author:      Mike Stoffels with Claude
// Created:     2025-12-07
// License:     MIT
// ============================================================================

package ui

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"runtime"

	"fyne.io/systray"
)

// IconState represents the current state for icon coloring
type IconState string

const (
	IconStateOffline    IconState = "offline"    // Gray - Backend not available
	IconStateIdle       IconState = "idle"       // White - Backend available, ready
	IconStateRecording  IconState = "recording"  // Red - Recording active
	IconStateProcessing IconState = "processing" // Blue - Processing
	IconStateError      IconState = "error"      // Orange - Error occurred
)

// TrayApp represents the system tray application
type TrayApp struct {
	onActivate    func()
	onSettings    func()
	onQuit        func()
	onModelSelect func(model string)

	menuStatus        *systray.MenuItem
	menuBackendStatus *systray.MenuItem
	menuModel         *systray.MenuItem
	menuActivate      *systray.MenuItem
	menuSettings      *systray.MenuItem
	menuQuit          *systray.MenuItem

	currentStatus  string
	currentModel   string
	backendOnline  bool
	currentIcon    IconState
}

// TrayCallbacks holds callback functions for tray events
type TrayCallbacks struct {
	OnActivate    func()
	OnSettings    func()
	OnQuit        func()
	OnModelSelect func(model string)
}

// NewTrayApp creates a new system tray application
func NewTrayApp(callbacks TrayCallbacks) *TrayApp {
	return &TrayApp{
		onActivate:    callbacks.OnActivate,
		onSettings:    callbacks.OnSettings,
		onQuit:        callbacks.OnQuit,
		onModelSelect: callbacks.OnModelSelect,
		currentStatus: "Bereit",
		currentModel:  "mistral:7b",
		backendOnline: false,
		currentIcon:   IconStateOffline,
	}
}

// SetupSystemTray is a no-op for systray version
func (t *TrayApp) SetupSystemTray() bool {
	// Setup is done in onReady during Run()
	return true
}

// Run starts the system tray application (blocking)
func (t *TrayApp) Run() {
	systray.Run(t.onReady, t.onExit)
}

// onReady is called when the system tray is ready
func (t *TrayApp) onReady() {
	// Set initial icon (gray "mDW" text - offline)
	systray.SetIcon(createTextIconBytes(IconStateOffline))
	systray.SetTitle("")
	systray.SetTooltip("mDW Voice Assistant")

	// Backend status (disabled, just for display)
	t.menuBackendStatus = systray.AddMenuItem("Backend: Prüfe...", "Backend-Verfügbarkeit")
	t.menuBackendStatus.Disable()

	// Status (disabled, just for display)
	t.menuStatus = systray.AddMenuItem("Status: "+t.currentStatus, "Aktueller Status")
	t.menuStatus.Disable()

	// Current model
	t.menuModel = systray.AddMenuItem("Modell: "+t.currentModel, "Aktuelles Modell")
	t.menuModel.Disable()

	systray.AddSeparator()

	// Activate button
	shortcut := "Ctrl+Shift+M"
	if runtime.GOOS == "darwin" {
		shortcut = "Menübar-Klick"
	}
	t.menuActivate = systray.AddMenuItem("Aktivieren ("+shortcut+")", "Sprachaufnahme starten")

	systray.AddSeparator()

	// Settings
	t.menuSettings = systray.AddMenuItem("Einstellungen...", "Einstellungen öffnen")

	systray.AddSeparator()

	// Quit
	t.menuQuit = systray.AddMenuItem("Beenden", "Anwendung beenden")

	// Handle menu clicks
	go t.handleClicks()
}

// handleClicks handles menu item clicks
func (t *TrayApp) handleClicks() {
	for {
		select {
		case <-t.menuActivate.ClickedCh:
			if t.onActivate != nil {
				t.onActivate()
			}
		case <-t.menuSettings.ClickedCh:
			if t.onSettings != nil {
				t.onSettings()
			}
		case <-t.menuQuit.ClickedCh:
			if t.onQuit != nil {
				t.onQuit()
			}
			systray.Quit()
			return
		}
	}
}

// onExit is called when the system tray exits
func (t *TrayApp) onExit() {
	// Cleanup if needed
}

// SetStatus updates the status display
func (t *TrayApp) SetStatus(status string) {
	t.currentStatus = status
	if t.menuStatus != nil {
		t.menuStatus.SetTitle("Status: " + status)
	}
}

// SetModel updates the model display
func (t *TrayApp) SetModel(model string) {
	t.currentModel = model
	if t.menuModel != nil {
		t.menuModel.SetTitle("Modell: " + model)
	}
}

// SetBackendStatus updates the backend availability status
func (t *TrayApp) SetBackendStatus(online bool, details string) {
	t.backendOnline = online
	if t.menuBackendStatus != nil {
		if online {
			if details != "" {
				// Show service details (e.g., "5/5 Services" or "4/5 (babbage)")
				t.menuBackendStatus.SetTitle("Backend: " + details + " ✓")
			} else {
				t.menuBackendStatus.SetTitle("Backend: Verfügbar ✓")
			}
		} else {
			status := "Nicht verfügbar"
			if details != "" {
				status = details
			}
			t.menuBackendStatus.SetTitle("Backend: " + status + " ✗")
		}
	}
	// Update icon if currently idle
	if t.currentIcon == IconStateOffline || t.currentIcon == IconStateIdle {
		if online {
			t.SetIconState(IconStateIdle)
		} else {
			t.SetIconState(IconStateOffline)
		}
	}
}

// SetIcon sets the tray icon based on state (legacy method for compatibility)
func (t *TrayApp) SetIcon(state string) {
	switch state {
	case "recording":
		t.SetIconState(IconStateRecording)
	case "processing":
		t.SetIconState(IconStateProcessing)
	case "error":
		t.SetIconState(IconStateError)
	case "idle":
		if t.backendOnline {
			t.SetIconState(IconStateIdle)
		} else {
			t.SetIconState(IconStateOffline)
		}
	default:
		if t.backendOnline {
			t.SetIconState(IconStateIdle)
		} else {
			t.SetIconState(IconStateOffline)
		}
	}
}

// SetIconState sets the tray icon based on IconState
func (t *TrayApp) SetIconState(state IconState) {
	t.currentIcon = state
	systray.SetIcon(createTextIconBytes(state))
}

// Quit quits the system tray
func (t *TrayApp) Quit() {
	systray.Quit()
}

// createTextIconBytes creates a PNG icon with "mDW" text in the specified color
func createTextIconBytes(state IconState) []byte {
	// macOS menu bar: use 44x22 for wider text (retina-ready height)
	width := 44
	height := 22
	img := image.NewRGBA(image.Rect(0, 0, width, height))

	// Get color based on state
	var c color.RGBA
	switch state {
	case IconStateOffline:
		c = color.RGBA{128, 128, 128, 255} // Gray
	case IconStateIdle:
		c = color.RGBA{255, 255, 255, 255} // White
	case IconStateRecording:
		c = color.RGBA{255, 59, 48, 255} // Red
	case IconStateProcessing:
		c = color.RGBA{0, 122, 255, 255} // Blue
	case IconStateError:
		c = color.RGBA{255, 149, 0, 255} // Orange
	default:
		c = color.RGBA{128, 128, 128, 255}
	}

	// Draw "mDW" text using bitmap font
	drawText(img, "mDW", 2, 4, c)

	// Encode to PNG
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return minimalPNG()
	}
	return buf.Bytes()
}

// Bitmap font data for characters (5x7 pixels each)
// Each character is defined as 7 rows of 5 bits
var bitmapFont = map[rune][]byte{
	'm': {
		0b00000,
		0b00000,
		0b11011,
		0b10101,
		0b10101,
		0b10101,
		0b10101,
	},
	'D': {
		0b11100,
		0b10010,
		0b10001,
		0b10001,
		0b10001,
		0b10010,
		0b11100,
	},
	'W': {
		0b10001,
		0b10001,
		0b10001,
		0b10101,
		0b10101,
		0b11011,
		0b10001,
	},
}

// drawText draws text on the image using bitmap font
func drawText(img *image.RGBA, text string, startX, startY int, c color.RGBA) {
	x := startX
	charWidth := 6  // 5 pixels + 1 spacing
	charHeight := 7

	// Scale factor for better visibility (2x)
	scale := 2

	for _, ch := range text {
		if pattern, ok := bitmapFont[ch]; ok {
			for row := 0; row < charHeight; row++ {
				for col := 0; col < 5; col++ {
					if pattern[row]&(1<<(4-col)) != 0 {
						// Draw scaled pixel
						for sy := 0; sy < scale; sy++ {
							for sx := 0; sx < scale; sx++ {
								px := x + col*scale + sx
								py := startY + row*scale + sy
								if px >= 0 && px < img.Bounds().Max.X && py >= 0 && py < img.Bounds().Max.Y {
									img.SetRGBA(px, py, c)
								}
							}
						}
					}
				}
			}
		}
		x += charWidth * scale
	}
}

// minimalPNG returns a minimal valid 1x1 PNG as fallback
func minimalPNG() []byte {
	img := image.NewRGBA(image.Rect(0, 0, 1, 1))
	img.Set(0, 0, color.Black)
	var buf bytes.Buffer
	png.Encode(&buf, img)
	return buf.Bytes()
}
