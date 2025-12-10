// ============================================================================
// meinDENKWERK (mDW) - Lokale KI-Plattform
// ============================================================================
//
// Package:     ui
// Description: Native macOS settings dialog using osascript
// Author:      Mike Stoffels with Claude
// Created:     2025-12-07
// License:     MIT
// ============================================================================

package ui

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// Settings holds the voice assistant settings
type Settings struct {
	Model             string
	OllamaURL         string
	UseMDW            bool
	MDWURL            string
	Language          string
	WhisperModel      string
	SilenceDurationMs int
	TTSEnabled        bool
	TTSVoice          string
	TTSRate           int
	VADThreshold      float64
	DialogMode        bool
	DialogTimeout     int
	InputDevice       string
	WakeWordEnabled   bool
	WakeWord          string
	StreamingSTT      bool
	StreamingInterval int
	// v0.3.2 Voxtral fields
	STTEngine    string
	VoxtralURL   string
	VoxtralModel string
}

// NativeSettingsDialog shows settings using native macOS dialogs
type NativeSettingsDialog struct {
	settings  Settings
	onApply   func(Settings)
	ollamaURL string
}

// NewNativeSettingsDialog creates a new native settings dialog
func NewNativeSettingsDialog(settings Settings, onApply func(Settings)) *NativeSettingsDialog {
	return &NativeSettingsDialog{
		settings:  settings,
		onApply:   onApply,
		ollamaURL: settings.OllamaURL,
	}
}

// Show displays the settings dialog
func (d *NativeSettingsDialog) Show() {
	go d.showMainMenu()
}

// showMainMenu shows the main settings menu
func (d *NativeSettingsDialog) showMainMenu() {
	backendMode := "Ollama direkt"
	if d.settings.UseMDW {
		backendMode = "mDW Backend"
	}

	options := []string{
		"Backend: " + backendMode,
		"LLM-Modell: " + d.settings.Model,
		"Ollama URL: " + d.settings.OllamaURL,
		"mDW URL: " + d.settings.MDWURL,
		"Sprache: " + d.settings.Language,
		"Whisper-Modell: " + d.settings.WhisperModel,
		"Stille-Erkennung: " + strconv.Itoa(d.settings.SilenceDurationMs) + "ms",
		"TTS: " + boolToGerman(d.settings.TTSEnabled),
		"TTS-Stimme: " + d.settings.TTSVoice,
		"TTS-Geschwindigkeit: " + strconv.Itoa(d.settings.TTSRate),
		"---",
		"Einstellungen speichern",
		"Abbrechen",
	}

	choice, err := d.showListDialog("mDW Voice Assistant - Einstellungen", "Einstellung auswählen:", options)
	if err != nil || choice == "" || choice == "Abbrechen" {
		return
	}

	switch {
	case strings.HasPrefix(choice, "Backend:"):
		d.showBackendSelector()
	case strings.HasPrefix(choice, "LLM-Modell:"):
		d.showModelSelector()
	case strings.HasPrefix(choice, "Ollama URL:"):
		d.showOllamaURLDialog()
	case strings.HasPrefix(choice, "mDW URL:"):
		d.showMDWURLDialog()
	case strings.HasPrefix(choice, "Sprache:"):
		d.showLanguageSelector()
	case strings.HasPrefix(choice, "Whisper-Modell:"):
		d.showWhisperModelSelector()
	case strings.HasPrefix(choice, "Stille-Erkennung:"):
		d.showSilenceDurationDialog()
	case strings.HasPrefix(choice, "TTS:"):
		d.toggleTTS()
	case strings.HasPrefix(choice, "TTS-Stimme:"):
		d.showVoiceSelector()
	case strings.HasPrefix(choice, "TTS-Geschwindigkeit:"):
		d.showTTSRateDialog()
	case choice == "Einstellungen speichern":
		if d.onApply != nil {
			d.onApply(d.settings)
		}
		d.showAlert("Einstellungen gespeichert", "Die Einstellungen wurden übernommen.")
		return
	}

	// Show main menu again
	d.showMainMenu()
}

// showModelSelector shows a dialog to select the LLM model
func (d *NativeSettingsDialog) showModelSelector() {
	models := d.fetchOllamaModels()
	if len(models) == 0 {
		// Manual input if no models available
		model, err := d.showInputDialog("LLM-Modell", "Modellname eingeben:", d.settings.Model)
		if err == nil && model != "" {
			d.settings.Model = model
		}
		return
	}

	choice, err := d.showListDialog("LLM-Modell auswählen", "Verfügbare Modelle:", models)
	if err == nil && choice != "" {
		d.settings.Model = choice
	}
}

// showBackendSelector shows a dialog to select the backend mode
func (d *NativeSettingsDialog) showBackendSelector() {
	options := []string{
		"Ollama direkt (schneller, kein mDW nötig)",
		"mDW Backend (über Kant → Turing, ermöglicht Agenten)",
	}
	choice, err := d.showListDialog("Backend auswählen", "Verarbeitungsmodus:", options)
	if err == nil && choice != "" {
		d.settings.UseMDW = strings.HasPrefix(choice, "mDW")
	}
}

// showOllamaURLDialog shows a dialog to enter Ollama URL
func (d *NativeSettingsDialog) showOllamaURLDialog() {
	url, err := d.showInputDialog("Ollama URL", "URL eingeben:", d.settings.OllamaURL)
	if err == nil && url != "" {
		d.settings.OllamaURL = url
		d.ollamaURL = url
	}
}

// showMDWURLDialog shows a dialog to enter mDW API URL
func (d *NativeSettingsDialog) showMDWURLDialog() {
	url, err := d.showInputDialog("mDW API URL", "URL eingeben (z.B. http://localhost:8080):", d.settings.MDWURL)
	if err == nil && url != "" {
		d.settings.MDWURL = url
	}
}

// showLanguageSelector shows a dialog to select the language
func (d *NativeSettingsDialog) showLanguageSelector() {
	languages := []string{"de", "en", "auto"}
	choice, err := d.showListDialog("Sprache auswählen", "Sprache für Spracherkennung:", languages)
	if err == nil && choice != "" {
		d.settings.Language = choice
	}
}

// showWhisperModelSelector shows a dialog to select Whisper model
func (d *NativeSettingsDialog) showWhisperModelSelector() {
	models := []string{"tiny", "base", "small", "medium"}
	choice, err := d.showListDialog("Whisper-Modell auswählen", "Modellgröße:", models)
	if err == nil && choice != "" {
		d.settings.WhisperModel = choice
	}
}

// showSilenceDurationDialog shows a dialog to set silence duration
func (d *NativeSettingsDialog) showSilenceDurationDialog() {
	options := []string{"1500", "2000", "2500", "3000", "3500", "4000", "5000"}
	choice, err := d.showListDialog("Stille-Erkennung", "Millisekunden bis Aufnahme endet:", options)
	if err == nil && choice != "" {
		if ms, err := strconv.Atoi(choice); err == nil {
			d.settings.SilenceDurationMs = ms
		}
	}
}

// toggleTTS toggles TTS on/off
func (d *NativeSettingsDialog) toggleTTS() {
	options := []string{"Aktiviert", "Deaktiviert"}
	choice, err := d.showListDialog("Sprachausgabe (TTS)", "Status:", options)
	if err == nil {
		d.settings.TTSEnabled = (choice == "Aktiviert")
	}
}

// showVoiceSelector shows a dialog to select TTS voice
func (d *NativeSettingsDialog) showVoiceSelector() {
	voices := d.fetchMacOSVoices()
	if len(voices) == 0 {
		voices = []string{"Anna", "Petra", "Yannick", "Markus", "Alex", "Samantha", "Daniel"}
	}

	choice, err := d.showListDialog("TTS-Stimme auswählen", "Verfügbare Stimmen:", voices)
	if err == nil && choice != "" {
		// Extract voice name (before the first space or language indicator)
		voiceName := strings.Split(choice, " ")[0]
		d.settings.TTSVoice = voiceName
	}
}

// showTTSRateDialog shows a dialog to set TTS rate
func (d *NativeSettingsDialog) showTTSRateDialog() {
	options := []string{"150", "175", "200", "220", "250", "275", "300"}
	choice, err := d.showListDialog("TTS-Geschwindigkeit", "Wörter pro Minute:", options)
	if err == nil && choice != "" {
		if rate, err := strconv.Atoi(choice); err == nil {
			d.settings.TTSRate = rate
		}
	}
}

// showListDialog shows a list dialog and returns the selected item
func (d *NativeSettingsDialog) showListDialog(title, prompt string, items []string) (string, error) {
	// Build AppleScript list
	var quotedItems []string
	for _, item := range items {
		quotedItems = append(quotedItems, fmt.Sprintf(`"%s"`, item))
	}
	itemList := strings.Join(quotedItems, ", ")

	script := fmt.Sprintf(`
		set theList to {%s}
		choose from list theList with prompt "%s" with title "%s" default items {item 1 of theList}
	`, itemList, prompt, title)

	result, err := d.runAppleScript(script)
	if err != nil {
		return "", err
	}

	// "false" is returned when cancelled
	if result == "false" {
		return "", nil
	}

	return result, nil
}

// showInputDialog shows an input dialog and returns the entered text
func (d *NativeSettingsDialog) showInputDialog(title, prompt, defaultValue string) (string, error) {
	script := fmt.Sprintf(`
		display dialog "%s" with title "%s" default answer "%s" buttons {"Abbrechen", "OK"} default button "OK"
		if button returned of result is "OK" then
			return text returned of result
		else
			return ""
		end if
	`, prompt, title, defaultValue)

	return d.runAppleScript(script)
}

// showAlert shows an alert dialog
func (d *NativeSettingsDialog) showAlert(title, message string) {
	script := fmt.Sprintf(`display dialog "%s" with title "%s" buttons {"OK"} default button "OK"`, message, title)
	d.runAppleScript(script)
}

// runAppleScript executes an AppleScript and returns the result
func (d *NativeSettingsDialog) runAppleScript(script string) (string, error) {
	cmd := exec.Command("osascript", "-e", script)
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

// fetchOllamaModels fetches available models from Ollama
func (d *NativeSettingsDialog) fetchOllamaModels() []string {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	url := d.ollamaURL + "/api/tags"
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()

	var result struct {
		Models []struct {
			Name string `json:"name"`
		} `json:"models"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil
	}

	var models []string
	for _, m := range result.Models {
		models = append(models, m.Name)
	}
	return models
}

// fetchMacOSVoices fetches available macOS voices
func (d *NativeSettingsDialog) fetchMacOSVoices() []string {
	cmd := exec.Command("say", "-v", "?")
	output, err := cmd.Output()
	if err != nil {
		return nil
	}

	var voices []string
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		// Format: "Anna (Deutsch (Deutschland)) de_DE    # Hallo, ich heiße Anna."
		// Find the language code pattern (xx_XX or xx-XX)
		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue
		}

		name := parts[0]
		var lang string

		// Find the language code (format: de_DE, en_US, etc.)
		for _, part := range parts[1:] {
			if len(part) >= 2 && (strings.Contains(part, "_") || strings.Contains(part, "-")) {
				// Check if it looks like a language code
				if len(part) == 5 && (part[2] == '_' || part[2] == '-') {
					lang = part
					break
				}
			}
		}

		if lang == "" {
			continue
		}

		// Only German and English voices
		if strings.HasPrefix(lang, "de") || strings.HasPrefix(lang, "en") {
			voices = append(voices, fmt.Sprintf("%s (%s)", name, lang))
		}
	}
	return voices
}

// boolToGerman converts a bool to German text
func boolToGerman(b bool) string {
	if b {
		return "Aktiviert"
	}
	return "Deaktiviert"
}
