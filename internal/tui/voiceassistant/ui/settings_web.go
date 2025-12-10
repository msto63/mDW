// ============================================================================
// meinDENKWERK (mDW) - Lokale KI-Plattform
// ============================================================================
//
// Package:     ui
// Description: Web-based settings dialog for Voice Assistant
// Author:      Mike Stoffels with Claude
// Created:     2025-12-07
// License:     MIT
// ============================================================================

package ui

import (
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"net"
	"net/http"
	"os/exec"
	"runtime"
	"strings"
	"sync"
	"time"
)

// WebSettingsServer provides a web-based settings UI
type WebSettingsServer struct {
	mu         sync.RWMutex
	settings   Settings
	onApply    func(Settings)
	server     *http.Server
	port       int
	ollamaURL  string
	done       chan struct{}
}

// NewWebSettingsServer creates a new web-based settings server
func NewWebSettingsServer(settings Settings, onApply func(Settings)) *WebSettingsServer {
	return &WebSettingsServer{
		settings:  settings,
		onApply:   onApply,
		ollamaURL: settings.OllamaURL,
		done:      make(chan struct{}),
	}
}

// Show starts the web server and opens the settings page in browser
func (s *WebSettingsServer) Show() {
	go s.startServer()
}

// startServer starts the HTTP server for settings
func (s *WebSettingsServer) startServer() {
	// Find an available port
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		fmt.Printf("Failed to start settings server: %v\n", err)
		return
	}
	s.port = listener.Addr().(*net.TCPAddr).Port

	mux := http.NewServeMux()
	mux.HandleFunc("/", s.handleIndex)
	mux.HandleFunc("/api/settings", s.handleSettings)
	mux.HandleFunc("/api/models", s.handleModels)
	mux.HandleFunc("/api/voices", s.handleVoices)
	mux.HandleFunc("/api/devices", s.handleAudioDevices)
	mux.HandleFunc("/api/save", s.handleSave)
	mux.HandleFunc("/api/close", s.handleClose)

	s.server = &http.Server{
		Handler: mux,
	}

	// Open browser after a short delay
	go func() {
		time.Sleep(200 * time.Millisecond)
		url := fmt.Sprintf("http://127.0.0.1:%d", s.port)
		openBrowser(url)
	}()

	// Serve until closed
	if err := s.server.Serve(listener); err != nil && err != http.ErrServerClosed {
		fmt.Printf("Settings server error: %v\n", err)
	}
}

// Stop stops the settings server
func (s *WebSettingsServer) Stop() {
	if s.server != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		s.server.Shutdown(ctx)
	}
}

// handleIndex serves the main settings page
func (s *WebSettingsServer) handleIndex(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	tmpl := template.Must(template.New("settings").Parse(settingsHTML))
	tmpl.Execute(w, nil)
}

// handleSettings returns current settings as JSON
func (s *WebSettingsServer) handleSettings(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(s.settings)
}

// handleModels returns available Ollama models
func (s *WebSettingsServer) handleModels(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	ollamaURL := s.ollamaURL
	s.mu.RUnlock()

	models := fetchOllamaModelsList(ollamaURL)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(models)
}

// handleVoices returns available TTS voices
func (s *WebSettingsServer) handleVoices(w http.ResponseWriter, r *http.Request) {
	voices := fetchSystemVoices()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(voices)
}

// handleAudioDevices returns available audio input devices
func (s *WebSettingsServer) handleAudioDevices(w http.ResponseWriter, r *http.Request) {
	devices := fetchAudioInputDevices()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(devices)
}

// handleSave saves the settings
func (s *WebSettingsServer) handleSave(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var newSettings Settings
	if err := json.NewDecoder(r.Body).Decode(&newSettings); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	s.mu.Lock()
	s.settings = newSettings
	s.ollamaURL = newSettings.OllamaURL
	s.mu.Unlock()

	if s.onApply != nil {
		s.onApply(newSettings)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// handleClose closes the settings window
func (s *WebSettingsServer) handleClose(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})

	// Close server after response
	go func() {
		time.Sleep(100 * time.Millisecond)
		s.Stop()
	}()
}

// openBrowser opens the URL in the default browser
func openBrowser(url string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "linux":
		cmd = exec.Command("xdg-open", url)
	case "windows":
		cmd = exec.Command("cmd", "/c", "start", url)
	}
	if cmd != nil {
		cmd.Start()
	}
}

// fetchOllamaModelsList fetches models from Ollama API
func fetchOllamaModelsList(ollamaURL string) []string {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	url := ollamaURL + "/api/tags"
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return []string{}
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return []string{}
	}
	defer resp.Body.Close()

	var result struct {
		Models []struct {
			Name string `json:"name"`
		} `json:"models"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return []string{}
	}

	models := make([]string, len(result.Models))
	for i, m := range result.Models {
		models[i] = m.Name
	}
	return models
}

// fetchSystemVoices fetches available macOS voices
func fetchSystemVoices() []Voice {
	cmd := exec.Command("say", "-v", "?")
	output, err := cmd.Output()
	if err != nil {
		return defaultVoices()
	}

	var voices []Voice
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue
		}

		name := parts[0]
		var lang string

		// Find language code
		for _, part := range parts[1:] {
			if len(part) == 5 && (part[2] == '_' || part[2] == '-') {
				lang = part
				break
			}
		}

		if lang == "" {
			continue
		}

		// Only German and English
		if strings.HasPrefix(lang, "de") || strings.HasPrefix(lang, "en") {
			voices = append(voices, Voice{
				Name:     name,
				Language: lang,
			})
		}
	}

	if len(voices) == 0 {
		return defaultVoices()
	}
	return voices
}

// Voice represents a TTS voice
type Voice struct {
	Name     string `json:"name"`
	Language string `json:"language"`
}

func defaultVoices() []Voice {
	return []Voice{
		{Name: "Anna", Language: "de_DE"},
		{Name: "Petra", Language: "de_DE"},
		{Name: "Markus", Language: "de_DE"},
		{Name: "Yannick", Language: "de_DE"},
		{Name: "Alex", Language: "en_US"},
		{Name: "Samantha", Language: "en_US"},
	}
}

// AudioDevice represents an audio input device
type AudioDevice struct {
	Name      string `json:"name"`
	IsDefault bool   `json:"isDefault"`
}

// fetchAudioInputDevices returns available audio input devices
func fetchAudioInputDevices() []AudioDevice {
	// Try to use system commands to list audio devices
	var devices []AudioDevice

	// Add default option
	devices = append(devices, AudioDevice{
		Name:      "default",
		IsDefault: true,
	})

	// On macOS, use system_profiler to list audio devices
	cmd := exec.Command("system_profiler", "SPAudioDataType", "-json")
	output, err := cmd.Output()
	if err == nil {
		var audioData struct {
			SPAudioDataType []struct {
				Items []struct {
					Name             string `json:"_name"`
					DefaultInputDevice string `json:"coreaudio_default_audio_input_device,omitempty"`
				} `json:"_items"`
			} `json:"SPAudioDataType"`
		}
		if json.Unmarshal(output, &audioData) == nil {
			for _, dataType := range audioData.SPAudioDataType {
				for _, item := range dataType.Items {
					if item.Name != "" && item.Name != "default" {
						devices = append(devices, AudioDevice{
							Name:      item.Name,
							IsDefault: item.DefaultInputDevice == "yes",
						})
					}
				}
			}
		}
	}

	// Fallback: try listing common macOS audio inputs
	if len(devices) == 1 {
		// Add common devices as fallback
		devices = append(devices, AudioDevice{Name: "MacBook Pro Microphone", IsDefault: false})
		devices = append(devices, AudioDevice{Name: "External Microphone", IsDefault: false})
	}

	return devices
}

// HTML template for settings page
const settingsHTML = `<!DOCTYPE html>
<html lang="de">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>mDW Voice Assistant - Einstellungen</title>
    <style>
        * {
            box-sizing: border-box;
            margin: 0;
            padding: 0;
        }
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
            background: #fff;
            color: #333;
            min-height: 100vh;
            padding: 24px;
        }
        .container {
            max-width: 560px;
            margin: 0 auto;
        }
        h1 {
            color: #333;
            margin-bottom: 4px;
            font-size: 22px;
            font-weight: 600;
        }
        .subtitle {
            color: #888;
            margin-bottom: 24px;
            font-size: 13px;
        }
        .section {
            border: 1px solid #e0e0e0;
            border-radius: 8px;
            padding: 16px 20px;
            margin-bottom: 16px;
        }
        .section-title {
            color: #666;
            font-size: 12px;
            font-weight: 600;
            text-transform: uppercase;
            letter-spacing: 0.5px;
            margin-bottom: 14px;
        }
        .form-group {
            margin-bottom: 14px;
        }
        .form-group:last-child {
            margin-bottom: 0;
        }
        label {
            display: block;
            font-size: 13px;
            color: #555;
            margin-bottom: 5px;
        }
        input[type="text"], input[type="number"], select {
            width: 100%;
            padding: 8px 10px;
            border: 1px solid #ccc;
            border-radius: 4px;
            background: #fff;
            color: #333;
            font-size: 13px;
        }
        input[type="text"]:focus, input[type="number"]:focus, select:focus {
            outline: none;
            border-color: #007AFF;
        }
        select {
            cursor: pointer;
        }
        .checkbox-group {
            display: flex;
            align-items: center;
            gap: 8px;
        }
        input[type="checkbox"] {
            width: 16px;
            height: 16px;
            cursor: pointer;
        }
        .checkbox-label {
            color: #333;
            font-size: 13px;
            cursor: pointer;
        }
        .row {
            display: grid;
            grid-template-columns: 1fr 1fr;
            gap: 14px;
        }
        .buttons {
            display: flex;
            gap: 10px;
            margin-top: 20px;
        }
        button {
            flex: 1;
            padding: 10px 20px;
            border-radius: 6px;
            font-size: 13px;
            font-weight: 500;
            cursor: pointer;
        }
        .btn-primary {
            background: #007AFF;
            color: #fff;
            border: none;
        }
        .btn-primary:hover {
            background: #0066DD;
        }
        .btn-secondary {
            background: #fff;
            color: #333;
            border: 1px solid #ccc;
        }
        .btn-secondary:hover {
            background: #f5f5f5;
        }
        .status {
            text-align: center;
            padding: 10px;
            border-radius: 6px;
            margin-top: 14px;
            display: none;
            font-size: 13px;
        }
        .status.success {
            display: block;
            background: #e8f5e9;
            color: #2e7d32;
        }
        .status.error {
            display: block;
            background: #ffebee;
            color: #c62828;
        }
        .hint {
            font-size: 11px;
            color: #888;
            margin-top: 4px;
        }
        @media (max-width: 500px) {
            .row {
                grid-template-columns: 1fr;
            }
        }
    </style>
</head>
<body>
    <div class="container">
        <h1>mDW Voice Assistant</h1>
        <p class="subtitle">Einstellungen</p>

        <div class="section">
            <div class="section-title">Backend & Modell</div>
            <div class="form-group">
                <label>Backend</label>
                <select id="backend">
                    <option value="ollama">Ollama direkt (schneller)</option>
                    <option value="mdw">mDW Backend (Kant → Turing)</option>
                </select>
                <div class="hint">mDW ermöglicht Agenten und Pre-/Post-Processing</div>
            </div>
            <div class="form-group">
                <label>LLM-Modell</label>
                <select id="model"></select>
            </div>
            <div class="row">
                <div class="form-group">
                    <label>Ollama URL</label>
                    <input type="text" id="ollamaUrl" placeholder="http://localhost:11434">
                </div>
                <div class="form-group">
                    <label>mDW URL</label>
                    <input type="text" id="mdwUrl" placeholder="http://localhost:8080">
                </div>
            </div>
        </div>

        <div class="section">
            <div class="section-title">Spracherkennung (STT)</div>
            <div class="form-group">
                <label>STT-Engine</label>
                <select id="sttEngine" onchange="toggleVoxtralSettings()">
                    <option value="whisper">Whisper (lokal, schnell)</option>
                    <option value="voxtral">Voxtral (via vLLM, GPU-beschleunigt)</option>
                </select>
                <div class="hint">Whisper: Lokales Modell | Voxtral: Mistral Audio-Modell via vLLM</div>
            </div>
            <div class="row">
                <div class="form-group">
                    <label>Sprache</label>
                    <select id="language">
                        <option value="de">Deutsch</option>
                        <option value="en">Englisch</option>
                        <option value="auto">Automatisch</option>
                    </select>
                </div>
                <div class="form-group" id="whisperModelGroup">
                    <label>Whisper-Modell</label>
                    <select id="whisperModel">
                        <option value="tiny">tiny (schnell)</option>
                        <option value="base">base (empfohlen)</option>
                        <option value="small">small (genauer)</option>
                        <option value="medium">medium (beste Qualität)</option>
                    </select>
                </div>
            </div>
            <div id="voxtralSettings" style="display: none;">
                <div class="row">
                    <div class="form-group">
                        <label>Voxtral Server URL</label>
                        <input type="text" id="voxtralUrl" placeholder="http://localhost:8100">
                        <div class="hint">vLLM Server mit Voxtral-Modell</div>
                    </div>
                    <div class="form-group">
                        <label>Voxtral-Modell</label>
                        <input type="text" id="voxtralModel" placeholder="mistralai/Voxtral-Mini-3B-2507">
                    </div>
                </div>
            </div>
            <div class="form-group" style="margin-top: 14px;">
                <div class="checkbox-group">
                    <input type="checkbox" id="streamingSTT">
                    <label class="checkbox-label" for="streamingSTT">Echtzeit-Transkription (Streaming STT)</label>
                </div>
                <div class="hint">Zeigt die Transkription in Echtzeit während des Sprechens. Hinweis: Höhere CPU-Last!</div>
            </div>
            <div class="form-group">
                <label>Streaming-Intervall</label>
                <select id="streamingInterval">
                    <option value="1000">1 Sekunde (schnell)</option>
                    <option value="1500">1,5 Sekunden</option>
                    <option value="2000">2 Sekunden (empfohlen)</option>
                    <option value="2500">2,5 Sekunden</option>
                    <option value="3000">3 Sekunden (genauer)</option>
                </select>
                <div class="hint">Wie oft die Zwischentranskription aktualisiert wird</div>
            </div>
        </div>

        <div class="section">
            <div class="section-title">Sprachausgabe (TTS)</div>
            <div class="form-group">
                <div class="checkbox-group">
                    <input type="checkbox" id="ttsEnabled">
                    <label class="checkbox-label" for="ttsEnabled">Sprachausgabe aktivieren</label>
                </div>
            </div>
            <div class="row">
                <div class="form-group">
                    <label>Stimme</label>
                    <select id="ttsVoice"></select>
                </div>
                <div class="form-group">
                    <label>Geschwindigkeit (WPM)</label>
                    <select id="ttsRate">
                        <option value="150">150 (langsam)</option>
                        <option value="175">175</option>
                        <option value="200">200</option>
                        <option value="220">220 (normal)</option>
                        <option value="250">250</option>
                        <option value="275">275</option>
                        <option value="300">300 (schnell)</option>
                    </select>
                </div>
            </div>
        </div>

        <div class="section">
            <div class="section-title">Aufnahme</div>
            <div class="form-group">
                <label>Mikrofon</label>
                <select id="inputDevice"></select>
                <div class="hint">Audio-Eingangsgerät für die Sprachaufnahme</div>
            </div>
            <div class="form-group">
                <label>Stille bis Aufnahme endet (ms)</label>
                <select id="silenceDuration">
                    <option value="1500">1500 ms (schnell)</option>
                    <option value="2000">2000 ms</option>
                    <option value="2500">2500 ms</option>
                    <option value="3000">3000 ms (Standard)</option>
                    <option value="3500">3500 ms</option>
                    <option value="4000">4000 ms</option>
                    <option value="5000">5000 ms (lang)</option>
                </select>
                <div class="hint">Wie lange Stille erkannt werden muss, bis die Aufnahme automatisch endet</div>
            </div>
        </div>

        <div class="section">
            <div class="section-title">Dialog-Modus</div>
            <div class="form-group">
                <div class="checkbox-group">
                    <input type="checkbox" id="dialogMode">
                    <label class="checkbox-label" for="dialogMode">Dialog-Modus aktivieren</label>
                </div>
                <div class="hint">Nach der Antwort automatisch auf weitere Eingabe warten</div>
            </div>
            <div class="form-group">
                <label>Timeout (Sekunden)</label>
                <select id="dialogTimeout">
                    <option value="5">5 Sekunden</option>
                    <option value="10">10 Sekunden (Standard)</option>
                    <option value="15">15 Sekunden</option>
                    <option value="20">20 Sekunden</option>
                    <option value="30">30 Sekunden</option>
                    <option value="0">Unbegrenzt</option>
                </select>
                <div class="hint">Zeit bis automatisch zum Idle-Modus zurückgekehrt wird</div>
            </div>
        </div>

        <div class="section">
            <div class="section-title">Wake Word</div>
            <div class="form-group">
                <div class="checkbox-group">
                    <input type="checkbox" id="wakeWordEnabled">
                    <label class="checkbox-label" for="wakeWordEnabled">Wake Word Erkennung aktivieren</label>
                </div>
                <div class="hint">Aktivierung durch Sprachbefehl (z.B. "Mein Denkwerk"). Hinweis: Erhöhter CPU-Verbrauch!</div>
            </div>
            <div class="form-group">
                <label>Wake Word</label>
                <input type="text" id="wakeWord" placeholder="mein denkwerk">
                <div class="hint">Erkennungsphrase für die Aktivierung (z.B. "mein denkwerk", "hey denkwerk")</div>
            </div>
        </div>

        <div class="buttons">
            <button class="btn-secondary" onclick="closeWindow()">Abbrechen</button>
            <button class="btn-primary" onclick="saveSettings()">Speichern</button>
        </div>

        <div id="status" class="status"></div>
    </div>

    <script>
        let currentSettings = {};

        async function loadSettings() {
            try {
                const resp = await fetch('/api/settings');
                currentSettings = await resp.json();

                document.getElementById('backend').value = currentSettings.UseMDW ? 'mdw' : 'ollama';
                document.getElementById('ollamaUrl').value = currentSettings.OllamaURL || 'http://localhost:11434';
                document.getElementById('mdwUrl').value = currentSettings.MDWURL || 'http://localhost:8080';
                document.getElementById('language').value = currentSettings.Language || 'de';
                document.getElementById('whisperModel').value = currentSettings.WhisperModel || 'base';
                document.getElementById('ttsEnabled').checked = currentSettings.TTSEnabled;
                document.getElementById('ttsRate').value = String(currentSettings.TTSRate || 220);
                document.getElementById('silenceDuration').value = String(currentSettings.SilenceDurationMs || 3000);
                document.getElementById('dialogMode').checked = currentSettings.DialogMode || false;
                document.getElementById('dialogTimeout').value = String(currentSettings.DialogTimeout || 10);

                // Load models
                loadModels(currentSettings.Model);

                // Load voices
                loadVoices(currentSettings.TTSVoice);

                // Load audio devices
                loadAudioDevices(currentSettings.InputDevice);

                // Wake word settings
                document.getElementById('wakeWordEnabled').checked = currentSettings.WakeWordEnabled || false;
                document.getElementById('wakeWord').value = currentSettings.WakeWord || 'mein denkwerk';

                // Streaming STT settings
                document.getElementById('streamingSTT').checked = currentSettings.StreamingSTT || false;
                document.getElementById('streamingInterval').value = String(currentSettings.StreamingInterval || 2000);

                // Voxtral settings
                document.getElementById('sttEngine').value = currentSettings.STTEngine || 'whisper';
                document.getElementById('voxtralUrl').value = currentSettings.VoxtralURL || 'http://localhost:8100';
                document.getElementById('voxtralModel').value = currentSettings.VoxtralModel || 'mistralai/Voxtral-Mini-3B-2507';
                toggleVoxtralSettings();
            } catch (e) {
                showStatus('Fehler beim Laden der Einstellungen', 'error');
            }
        }

        function toggleVoxtralSettings() {
            const engine = document.getElementById('sttEngine').value;
            const voxtralSettings = document.getElementById('voxtralSettings');
            const whisperModelGroup = document.getElementById('whisperModelGroup');

            if (engine === 'voxtral') {
                voxtralSettings.style.display = 'block';
                whisperModelGroup.style.display = 'none';
            } else {
                voxtralSettings.style.display = 'none';
                whisperModelGroup.style.display = 'block';
            }
        }

        async function loadModels(currentModel) {
            try {
                const resp = await fetch('/api/models');
                const models = await resp.json();
                const select = document.getElementById('model');
                select.innerHTML = '';

                if (models.length === 0) {
                    const opt = document.createElement('option');
                    opt.value = currentModel || 'mistral:7b';
                    opt.textContent = currentModel || 'mistral:7b';
                    select.appendChild(opt);
                } else {
                    models.forEach(m => {
                        const opt = document.createElement('option');
                        opt.value = m;
                        opt.textContent = m;
                        if (m === currentModel) opt.selected = true;
                        select.appendChild(opt);
                    });
                }
            } catch (e) {
                console.error('Failed to load models:', e);
            }
        }

        async function loadVoices(currentVoice) {
            try {
                const resp = await fetch('/api/voices');
                const voices = await resp.json();
                const select = document.getElementById('ttsVoice');
                select.innerHTML = '';

                voices.forEach(v => {
                    const opt = document.createElement('option');
                    opt.value = v.name;
                    opt.textContent = v.name + ' (' + v.language + ')';
                    if (v.name === currentVoice) opt.selected = true;
                    select.appendChild(opt);
                });
            } catch (e) {
                console.error('Failed to load voices:', e);
            }
        }

        async function loadAudioDevices(currentDevice) {
            try {
                const resp = await fetch('/api/devices');
                const devices = await resp.json();
                const select = document.getElementById('inputDevice');
                select.innerHTML = '';

                devices.forEach(d => {
                    const opt = document.createElement('option');
                    opt.value = d.name;
                    opt.textContent = d.name + (d.isDefault ? ' (Standard)' : '');
                    if (d.name === currentDevice || (currentDevice === '' && d.isDefault)) opt.selected = true;
                    select.appendChild(opt);
                });
            } catch (e) {
                console.error('Failed to load audio devices:', e);
            }
        }

        async function saveSettings() {
            const settings = {
                Model: document.getElementById('model').value,
                OllamaURL: document.getElementById('ollamaUrl').value,
                UseMDW: document.getElementById('backend').value === 'mdw',
                MDWURL: document.getElementById('mdwUrl').value,
                Language: document.getElementById('language').value,
                WhisperModel: document.getElementById('whisperModel').value,
                SilenceDurationMs: parseInt(document.getElementById('silenceDuration').value),
                TTSEnabled: document.getElementById('ttsEnabled').checked,
                TTSVoice: document.getElementById('ttsVoice').value,
                TTSRate: parseInt(document.getElementById('ttsRate').value),
                VADThreshold: currentSettings.VADThreshold || 0.5,
                DialogMode: document.getElementById('dialogMode').checked,
                DialogTimeout: parseInt(document.getElementById('dialogTimeout').value),
                InputDevice: document.getElementById('inputDevice').value,
                WakeWordEnabled: document.getElementById('wakeWordEnabled').checked,
                WakeWord: document.getElementById('wakeWord').value,
                StreamingSTT: document.getElementById('streamingSTT').checked,
                StreamingInterval: parseInt(document.getElementById('streamingInterval').value),
                STTEngine: document.getElementById('sttEngine').value,
                VoxtralURL: document.getElementById('voxtralUrl').value,
                VoxtralModel: document.getElementById('voxtralModel').value
            };

            try {
                const resp = await fetch('/api/save', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify(settings)
                });

                if (resp.ok) {
                    showStatus('Einstellungen gespeichert!', 'success');
                    setTimeout(closeWindow, 1000);
                } else {
                    showStatus('Fehler beim Speichern', 'error');
                }
            } catch (e) {
                showStatus('Fehler beim Speichern: ' + e.message, 'error');
            }
        }

        function closeWindow() {
            fetch('/api/close').then(() => {
                window.close();
            }).catch(() => {
                window.close();
            });
        }

        function showStatus(message, type) {
            const el = document.getElementById('status');
            el.textContent = message;
            el.className = 'status ' + type;
        }

        // Reload models when Ollama URL changes
        document.getElementById('ollamaUrl').addEventListener('blur', function() {
            loadModels(document.getElementById('model').value);
        });

        // Load settings on page load
        loadSettings();
    </script>
</body>
</html>`
