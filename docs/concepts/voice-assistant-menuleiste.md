# Voice Assistant für mDW - Konzept

**Dokumentversion**: 2.0.0
**Erstellt**: 2025-12-07
**Aktualisiert**: 2025-12-07
**Autor**: Mike Stoffels mit Claude
**Status**: Phase 1 + 2 implementiert, Phase 3 geplant

---

## 1. Executive Summary

Dieses Dokument beschreibt das Konzept für einen Voice-Assistant, der als Menüleisten-Anwendung (System Tray) auf macOS, Linux und Windows läuft. Die Anwendung ermöglicht Sprachinteraktion mit der mDW-Plattform durch:

- **Wake-Word-Erkennung** oder **Keyboard-Shortcut** zur Aktivierung
- **Speech-to-Text (STT)** für die Spracherkennung
- **Voice Activity Detection (VAD)** zur automatischen Erkennung von Sprechpausen
- **Text-to-Speech (TTS)** für natürlichsprachige Antworten
- **Full-Duplex Streaming** für Echtzeit-Dialog

---

## 2. Anforderungen

### 2.1 Funktionale Anforderungen

| ID | Anforderung | Priorität |
|----|-------------|-----------|
| F01 | Menüleisten-Icon mit Kontextmenü | Must |
| F02 | Aktivierung via Keyboard-Shortcut (z.B. Cmd+Shift+M) | Must |
| F03 | Aktivierung via Wake-Word ("mein DW") | Should |
| F04 | Audio-Aufnahme vom Mikrofon | Must |
| F05 | VAD mit 3 Sekunden Sprechpause-Erkennung | Must |
| F06 | STT - Lokale Spracherkennung | Must |
| F07 | Popup-Fenster für Antwort-Anzeige | Must |
| F08 | TTS - Natürlichsprachige Ausgabe | Should |
| F09 | Streaming-Integration mit mDW (Kant → Babbage) | Must |
| F10 | Full-Duplex-Kommunikation | Could |
| F11 | Dialog-Modus (Hin-und-Her-Gespräch) | Could |

### 2.2 Nicht-funktionale Anforderungen

| ID | Anforderung | Zielwert |
|----|-------------|----------|
| NF01 | Plattformunabhängigkeit | macOS, Linux, Windows |
| NF02 | Digitale Souveränität | MIT/Apache 2.0 Lizenzen |
| NF03 | Offline-Fähigkeit (STT/TTS) | Vollständig offline |
| NF04 | Latenz STT | < 500ms nach Sprechende |
| NF05 | Latenz TTS | < 300ms bis erste Silbe |
| NF06 | Ressourcenverbrauch | < 500MB RAM, < 10% CPU idle |

---

## 3. Architektur-Übersicht

### 3.1 Komponenten-Diagramm

```
┌─────────────────────────────────────────────────────────────────────┐
│                    Voice Assistant (Go)                              │
├─────────────────────────────────────────────────────────────────────┤
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐  ┌────────────┐ │
│  │  System     │  │   Audio     │  │   Speech    │  │    UI      │ │
│  │   Tray      │  │  Capture    │  │  Pipeline   │  │  Manager   │ │
│  │  (systray)  │  │ (PortAudio) │  │             │  │            │ │
│  └──────┬──────┘  └──────┬──────┘  └──────┬──────┘  └─────┬──────┘ │
│         │                │                │                │        │
│         │         ┌──────┴──────┐         │                │        │
│         │         │     VAD     │─────────┤                │        │
│         │         │ (WebRTC/    │         │                │        │
│         │         │  Silero)    │         │                │        │
│         │         └─────────────┘         │                │        │
│         │                          ┌──────┴──────┐         │        │
│         │                          │     STT     │         │        │
│         │                          │  (Whisper)  │         │        │
│         │                          └──────┬──────┘         │        │
│         │                                 │                │        │
│         │                          ┌──────┴──────┐         │        │
│         │                          │     TTS     │         │        │
│         │                          │   (Piper)   │         │        │
│         │                          └──────┬──────┘         │        │
│         │                                 │                │        │
│  ┌──────┴─────────────────────────────────┴────────────────┴──────┐ │
│  │                      Event Bus / State Machine                  │ │
│  └─────────────────────────────────┬───────────────────────────────┘ │
└────────────────────────────────────┼────────────────────────────────┘
                                     │
                              ┌──────┴──────┐
                              │   mDW API   │
                              │   (Kant)    │
                              └──────┬──────┘
                                     │
                    ┌────────────────┼────────────────┐
                    │                │                │
             ┌──────┴──────┐  ┌──────┴──────┐  ┌──────┴──────┐
             │   Babbage   │  │   Turing    │  │   Leibniz   │
             │    (NLP)    │  │    (LLM)    │  │   (Agent)   │
             └─────────────┘  └─────────────┘  └─────────────┘
```

### 3.2 Datenfluss

```
┌─────────┐    ┌─────┐    ┌─────┐    ┌──────┐    ┌──────┐    ┌─────────┐
│Mikrofon │───►│ VAD │───►│ STT │───►│ Kant │───►│Turing│───►│ Popup   │
│         │    │     │    │     │    │      │    │      │    │ Window  │
└─────────┘    └─────┘    └─────┘    └──────┘    └──────┘    └────┬────┘
                                                                  │
                                          ┌───────────────────────┘
                                          ▼
                                     ┌─────────┐    ┌─────────┐
                                     │   TTS   │───►│ Speaker │
                                     │         │    │         │
                                     └─────────┘    └─────────┘
```

### 3.3 State Machine

```
                    ┌─────────────────┐
                    │      IDLE       │◄────────────────────────┐
                    │  (Listening for │                         │
                    │   Wake-Word or  │                         │
                    │    Shortcut)    │                         │
                    └────────┬────────┘                         │
                             │                                  │
            Wake-Word oder   │                                  │
            Shortcut erkannt │                                  │
                             ▼                                  │
                    ┌─────────────────┐                         │
                    │   LISTENING     │                         │
                    │  (Recording     │                         │
                    │   User Speech)  │                         │
                    └────────┬────────┘                         │
                             │                                  │
            3s Pause erkannt │                                  │
            (VAD)            │                                  │
                             ▼                                  │
                    ┌─────────────────┐                         │
                    │  PROCESSING     │                         │
                    │  (STT + Send    │                         │
                    │   to mDW)       │                         │
                    └────────┬────────┘                         │
                             │                                  │
            Antwort erhalten │                                  │
                             ▼                                  │
                    ┌─────────────────┐         Timeout oder    │
                    │   RESPONDING    │────────────────────────►│
                    │  (Show Popup +  │         User schließt   │
                    │   Optional TTS) │                         │
                    └────────┬────────┘                         │
                             │                                  │
            Dialog-Modus:    │                                  │
            User spricht     │                                  │
            erneut           │                                  │
                             ▼                                  │
                    ┌─────────────────┐                         │
                    │    DIALOG       │─────────────────────────┘
                    │  (Continuous    │      Ende des Dialogs
                    │   Conversation) │
                    └─────────────────┘
```

---

## 4. Technologie-Stack

### 4.1 Empfohlene Bibliotheken

| Komponente | Bibliothek | Lizenz | Plattformen | Begründung |
|------------|------------|--------|-------------|------------|
| **System Tray** | [fyne.io/systray](https://pkg.go.dev/fyne.io/systray) | BSD-3 | macOS, Linux, Windows | Aktiv gepflegt, weniger Abhängigkeiten als Original |
| **Audio I/O** | [gordonklaus/portaudio](https://github.com/gordonklaus/portaudio) | MIT | Alle (via PortAudio) | Bewährte Go-Bindings für PortAudio |
| **VAD** | [silero-vad-go](https://pkg.go.dev/github.com/skypro1111/silero-vad-go) | MIT | Alle | Moderne ML-basierte VAD, hohe Genauigkeit |
| **VAD (Alternative)** | [go-webrtcvad](https://github.com/maxhawkins/go-webrtcvad) | Apache 2.0 | Alle | WebRTC's VAD, schnell und leichtgewichtig |
| **STT** | [whisper.cpp/bindings/go](https://github.com/ggerganov/whisper.cpp) | MIT | Alle | Beste Qualität, offline, Multi-Sprache |
| **STT (Alternative)** | [go-whisper](https://github.com/mutablelogic/go-whisper) | Apache 2.0 | Alle | Higher-level API, HTTP-Server integriert |
| **TTS** | Piper via CLI/HTTP | MIT | Alle | Schnell, offline, gute deutsche Stimmen |
| **Wake-Word** | [Porcupine](https://github.com/Picovoice/porcupine) | Apache 2.0* | Alle | Beste Genauigkeit (*teilweise proprietär) |
| **GUI (Popup)** | [fyne.io/fyne](https://fyne.io/) | BSD-3 | Alle | Native Look, Go-native |
| **Hotkey** | [golang-design/hotkey](https://github.com/golang-design/hotkey) | MIT | macOS, Linux, Windows | Cross-Platform Global Hotkeys |

### 4.2 Abhängigkeiten nach Plattform

#### macOS
```bash
# Homebrew
brew install portaudio
# Für Whisper.cpp mit Metal-Beschleunigung
brew install cmake
```

#### Linux (Ubuntu/Debian)
```bash
sudo apt-get install portaudio19-dev
sudo apt-get install libayatana-appindicator3-dev  # Für System Tray
```

#### Windows
```bash
# Via vcpkg oder vorkompilierte Binaries
vcpkg install portaudio
```

---

## 5. Detaillierte Komponenten-Beschreibung

### 5.1 Audio-Subsystem

#### Audio Capture (PortAudio)

```go
// Beispiel-Konfiguration
const (
    SampleRate      = 16000  // 16kHz für Whisper
    FramesPerBuffer = 512
    Channels        = 1      // Mono
    BitDepth        = 16     // 16-bit PCM
)

type AudioCapture struct {
    stream     *portaudio.Stream
    buffer     []int16
    outputChan chan []int16
    running    bool
}
```

**Wichtige Aspekte:**
- **Sample Rate**: 16kHz ist optimal für Whisper
- **Buffer Size**: 512 Samples (~32ms bei 16kHz) für niedrige Latenz
- **Format**: 16-bit PCM Mono

#### Voice Activity Detection (VAD)

**Option A: Silero VAD (empfohlen für Genauigkeit)**
```go
type SileroVAD struct {
    model         *ort.Session  // ONNX Runtime
    threshold     float32       // 0.5 default
    minSilenceMs  int           // 3000ms
    windowSize    int           // 512 samples
}
```

**Option B: WebRTC VAD (empfohlen für Geschwindigkeit)**
```go
type WebRTCVAD struct {
    vad           *webrtcvad.VAD
    aggressiveness int  // 0-3, höher = mehr Filterung
    frameDuration  int  // 10, 20 oder 30 ms
}
```

**Vergleich:**

| Kriterium | Silero VAD | WebRTC VAD |
|-----------|------------|------------|
| Genauigkeit | Sehr hoch | Gut |
| CPU-Verbrauch | Mittel | Sehr niedrig |
| Latenz | ~10ms | ~1ms |
| Modellgröße | ~2MB | ~100KB |
| Rauschresistenz | Ausgezeichnet | Gut |

### 5.2 Speech-to-Text (STT)

#### Whisper.cpp Integration

```go
type WhisperSTT struct {
    ctx       whisper.Context
    model     whisper.Model
    modelPath string
    language  string  // "de", "en", "auto"
}

func (w *WhisperSTT) Transcribe(audio []float32) (string, error) {
    // Audio in Whisper-Format konvertieren
    // Transkription durchführen
    // Text zurückgeben
}
```

**Modell-Auswahl:**

| Modell | Größe | RAM | Genauigkeit | Geschwindigkeit |
|--------|-------|-----|-------------|-----------------|
| tiny | 75MB | ~400MB | Gut | Sehr schnell |
| base | 142MB | ~500MB | Besser | Schnell |
| small | 466MB | ~1GB | Sehr gut | Mittel |
| medium | 1.5GB | ~2.5GB | Exzellent | Langsamer |
| large-v3 | 3GB | ~5GB | Beste | Langsam |

**Empfehlung**: `small` oder `base` für Balance zwischen Qualität und Geschwindigkeit.

#### Streaming-STT (Inkrementell)

Für Full-Duplex wird inkrementelle Transkription benötigt:

```go
type StreamingSTT struct {
    whisper    *WhisperSTT
    buffer     *RingBuffer
    windowMs   int  // 2000ms sliding window
    overlapMs  int  // 500ms overlap
}

func (s *StreamingSTT) ProcessChunk(audio []float32) (string, bool) {
    // Audio zum Buffer hinzufügen
    // Wenn genug Daten: Transkribieren
    // Delta-Text zurückgeben
}
```

### 5.3 Text-to-Speech (TTS)

#### Piper TTS Integration

Da es keine native Go-Library für Piper gibt, empfehlen wir:

**Option A: Piper als Subprocess**
```go
type PiperTTS struct {
    binaryPath string
    modelPath  string
    voiceName  string  // z.B. "de_DE-thorsten-high"
}

func (p *PiperTTS) Speak(text string) ([]byte, error) {
    cmd := exec.Command(p.binaryPath,
        "--model", p.modelPath,
        "--output_raw",
    )
    cmd.Stdin = strings.NewReader(text)
    return cmd.Output()
}
```

**Option B: HTTP-API (via LocalAI oder OpenTTS)**
```go
type HTTPTTSClient struct {
    baseURL string  // http://localhost:5500
    voice   string
}

func (c *HTTPTTSClient) Speak(text string) (io.Reader, error) {
    resp, err := http.Get(fmt.Sprintf(
        "%s/api/tts?voice=%s&text=%s",
        c.baseURL, c.voice, url.QueryEscape(text),
    ))
    return resp.Body, err
}
```

**Deutsche Stimmen für Piper:**
- `de_DE-thorsten-high` - Männlich, natürlich
- `de_DE-thorsten-medium` - Männlich, schneller
- `de_DE-eva_k-x_low` - Weiblich

### 5.4 Wake-Word-Erkennung

#### Porcupine (Beste Qualität, aber eingeschränkt)

```go
type PorcupineWakeWord struct {
    handle     *porcupine.Porcupine
    keywords   []string  // ["mein DENKWERK"]
    accessKey  string    // API Key (kostenlos für dev)
}
```

**Einschränkung**: Custom Wake Words benötigen Training über Picovoice Console (kostenlos, aber Account nötig).

#### Alternative: Einfache Keyword-Erkennung via Whisper

```go
type WhisperKeywordDetector struct {
    stt       *WhisperSTT
    keywords  []string
    threshold float32  // Ähnlichkeitsschwelle
}

func (d *WhisperKeywordDetector) Detect(audio []float32) bool {
    text := d.stt.TranscribeFast(audio)
    for _, kw := range d.keywords {
        if strings.Contains(strings.ToLower(text), kw) {
            return true
        }
    }
    return false
}
```

**Vor-/Nachteile:**

| Aspekt | Porcupine | Whisper-basiert |
|--------|-----------|-----------------|
| Stromverbrauch | Sehr niedrig | Hoch |
| Genauigkeit | Sehr hoch | Gut |
| Latenz | ~10ms | ~200ms |
| Custom Keywords | Account nötig | Frei definierbar |
| Offline | Ja | Ja |

### 5.5 GUI-Komponenten

#### System Tray (fyne.io/systray)

```go
func setupSystray() {
    systray.SetIcon(iconData)
    systray.SetTitle("mDW Voice")
    systray.SetTooltip("meinDENKWERK Voice Assistant")

    mStatus := systray.AddMenuItem("Status: Bereit", "")
    mStatus.Disable()

    systray.AddSeparator()

    mModel := systray.AddMenuItem("Modell: mistral:7b", "")
    mSettings := systray.AddMenuItem("Einstellungen...", "")

    systray.AddSeparator()

    mQuit := systray.AddMenuItem("Beenden", "")
}
```

#### Popup-Fenster (Fyne)

```go
type ResponsePopup struct {
    window   fyne.Window
    content  *widget.RichText
    spinner  *widget.Activity
}

func NewResponsePopup(app fyne.App) *ResponsePopup {
    w := app.NewWindow("mDW Antwort")
    w.Resize(fyne.NewSize(400, 300))
    w.SetFixedSize(false)

    content := widget.NewRichText()
    spinner := widget.NewActivity()

    container := container.NewVBox(
        spinner,
        container.NewScroll(content),
    )

    w.SetContent(container)
    return &ResponsePopup{window: w, content: content, spinner: spinner}
}
```

#### Global Hotkey

```go
import "github.com/golang-design/hotkey"

func registerHotkey() {
    hk := hotkey.New([]hotkey.Modifier{hotkey.ModCtrl, hotkey.ModShift}, hotkey.KeyM)

    if err := hk.Register(); err != nil {
        log.Fatal(err)
    }

    go func() {
        for range hk.Keydown() {
            // Aktivierung auslösen
            triggerListening()
        }
    }()
}
```

---

## 6. Integration mit mDW

### 6.1 API-Erweiterung für Babbage

Aktuell ist Babbage rein Text-basiert. Für Voice-Support sind folgende Erweiterungen möglich:

#### Option A: Audio-Upload an Kant (Einfach)

```protobuf
// api/proto/babbage.proto - Erweiterung

message TranscribeRequest {
    bytes audio_data = 1;           // WAV/PCM Audio
    string format = 2;              // "wav", "pcm16"
    int32 sample_rate = 3;          // 16000
    string language = 4;            // "de", "auto"
}

message TranscribeResponse {
    string text = 1;
    string detected_language = 2;
    float confidence = 3;
    int64 duration_ms = 4;
}

service BabbageService {
    // Bestehende RPCs...

    // Neu: Audio-Transkription
    rpc Transcribe(TranscribeRequest) returns (TranscribeResponse);
    rpc TranscribeStream(stream TranscribeRequest) returns (stream TranscribeResponse);
}
```

#### Option B: Lokale STT im Voice Assistant (Empfohlen)

Vorteile:
- Keine Netzwerk-Latenz für STT
- Geringere Bandbreite
- Offline-Fähigkeit
- mDW-Services bleiben unverändert

```
Voice Assistant (lokal)
    │
    ├── Audio Capture
    ├── VAD
    ├── STT (Whisper lokal)
    │
    └── Text ──────────────────► Kant ──► Turing/Leibniz
                                   │
         Antwort (Text) ◄──────────┘
              │
              ├── Popup anzeigen
              └── TTS (Piper lokal)
```

### 6.2 Kant API-Nutzung

Der Voice Assistant nutzt bestehende Kant-Endpoints:

```go
type MdwClient struct {
    baseURL    string  // http://localhost:8080
    httpClient *http.Client
    wsConn     *websocket.Conn
}

// Für einfache Anfragen
func (c *MdwClient) Chat(ctx context.Context, message string) (string, error) {
    req := ChatRequest{
        Messages: []Message{{Role: "user", Content: message}},
        Model:    "mistral:7b",
    }
    // POST /api/v1/chat
}

// Für Streaming
func (c *MdwClient) ChatStream(ctx context.Context, message string) (<-chan string, error) {
    // WebSocket: /api/v1/chat/ws
}
```

### 6.3 Full-Duplex-Architektur

```
┌──────────────────────────────────────────────────────────────────┐
│                     Voice Assistant                               │
│                                                                   │
│  ┌─────────────┐       ┌─────────────┐       ┌─────────────┐    │
│  │   Audio     │──────►│    STT      │──────►│   Text      │    │
│  │   Input     │       │  (Stream)   │       │   Buffer    │    │
│  └─────────────┘       └─────────────┘       └──────┬──────┘    │
│                                                      │           │
│                                          Partial Text│           │
│                                                      ▼           │
│  ┌─────────────┐       ┌─────────────┐       ┌─────────────┐    │
│  │   Audio     │◄──────│    TTS      │◄──────│  WebSocket  │◄───┼──┐
│  │   Output    │       │  (Stream)   │       │   Client    │    │  │
│  └─────────────┘       └─────────────┘       └─────────────┘    │  │
│                                                                   │  │
└──────────────────────────────────────────────────────────────────┘  │
                                                                       │
                               ┌───────────────────────────────────────┘
                               │
                               ▼
                        ┌─────────────┐
                        │    Kant     │
                        │  WebSocket  │
                        │  /chat/ws   │
                        └──────┬──────┘
                               │
                               ▼
                        ┌─────────────┐
                        │   Turing    │
                        │  (Stream)   │
                        └─────────────┘
```

---

## 7. Projekt-Struktur

```
internal/tui/voiceassistant/
├── main.go                 # Entry Point
├── version.go              # Version Info
├── app.go                  # Application Controller
├── state.go                # State Machine
├── config.go               # Configuration
│
├── audio/
│   ├── capture.go          # PortAudio Capture
│   ├── playback.go         # Audio Playback
│   └── buffer.go           # Ring Buffer
│
├── vad/
│   ├── vad.go              # VAD Interface
│   ├── silero.go           # Silero VAD Implementation
│   └── webrtc.go           # WebRTC VAD Implementation
│
├── stt/
│   ├── stt.go              # STT Interface
│   ├── whisper.go          # Whisper Implementation
│   └── streaming.go        # Streaming STT
│
├── tts/
│   ├── tts.go              # TTS Interface
│   ├── piper.go            # Piper Implementation
│   └── http.go             # HTTP TTS Client
│
├── wakeword/
│   ├── detector.go         # Wake Word Interface
│   ├── porcupine.go        # Porcupine Implementation
│   └── keyword.go          # Simple Keyword Detection
│
├── ui/
│   ├── systray.go          # System Tray
│   ├── popup.go            # Response Popup
│   ├── settings.go         # Settings Dialog
│   └── icons/              # Icon Assets
│
├── client/
│   ├── mdw.go              # mDW API Client
│   └── websocket.go        # WebSocket Handler
│
└── assets/
    ├── models/             # Whisper/Piper Models
    └── sounds/             # Notification Sounds
```

---

## 8. Konfiguration

```toml
# configs/voiceassistant.toml

[general]
language = "de"
log_level = "info"

[activation]
mode = "shortcut"           # "shortcut", "wakeword", "both"
shortcut = "ctrl+shift+m"
wakeword = "mein denkwerk"
wakeword_sensitivity = 0.5

[audio]
input_device = "default"
output_device = "default"
sample_rate = 16000
buffer_size = 512

[vad]
engine = "silero"           # "silero", "webrtc"
threshold = 0.5
silence_duration_ms = 3000
min_speech_duration_ms = 500

[stt]
engine = "whisper"
model = "base"              # "tiny", "base", "small", "medium"
model_path = "./models/whisper"

[tts]
enabled = true
engine = "piper"            # "piper", "http"
voice = "de_DE-thorsten-high"
model_path = "./models/piper"
# Für HTTP-Engine:
# http_url = "http://localhost:5500"

[mdw]
api_url = "http://localhost:8080"
websocket_url = "ws://localhost:8080/api/v1/chat/ws"
model = "mistral:7b"
timeout_seconds = 60

[ui]
popup_width = 400
popup_height = 300
notification_sounds = true
```

---

## 9. Vor- und Nachteile

### 9.1 Vorteile

| Aspekt | Beschreibung |
|--------|--------------|
| **Digitale Souveränität** | Alle Kernkomponenten (STT, TTS, VAD) laufen lokal ohne Cloud-Abhängigkeit |
| **Datenschutz** | Keine Audiodaten verlassen das Gerät (außer transkribierter Text an lokales mDW) |
| **Offline-Fähigkeit** | Voice-Funktionen funktionieren ohne Internet |
| **Plattformunabhängig** | Go + CGO ermöglicht Builds für macOS, Linux, Windows |
| **Integration** | Nahtlose Integration in bestehendes mDW-Ökosystem |
| **Erweiterbar** | Modularer Aufbau ermöglicht Austausch einzelner Komponenten |
| **Kostenlos** | Keine API-Kosten für Cloud-Services |

### 9.2 Nachteile

| Aspekt | Beschreibung | Mitigation |
|--------|--------------|------------|
| **CGO-Abhängigkeit** | PortAudio, Whisper.cpp benötigen CGO | Cross-Compilation erschwert, aber möglich |
| **Modell-Größe** | Whisper + Piper Modelle ~500MB-2GB | Kleinere Modelle wählen, Download bei Erststart |
| **RAM-Verbrauch** | ~500MB-1.5GB je nach Modellen | Konfigurierbare Modellgröße |
| **Build-Komplexität** | Native Libraries pro Plattform | Makefile/Scripts für Build-Automatisierung |
| **Wake-Word-Qualität** | Open-Source Alternativen weniger genau | Porcupine oder Shortcut-Aktivierung bevorzugen |
| **GPU-Beschleunigung** | Erfordert plattformspezifischen Setup | CPU-Only als Fallback, Metal/CUDA optional |

### 9.3 Risiken

| Risiko | Wahrscheinlichkeit | Auswirkung | Mitigation |
|--------|-------------------|------------|------------|
| Performance-Probleme auf älteren Geräten | Mittel | Hoch | Modell-Auswahl anpassbar |
| Whisper.cpp API-Änderungen | Niedrig | Mittel | Version pinnen |
| PortAudio Bugs auf spezifischen Geräten | Niedrig | Mittel | Alternative Audio-Backends |
| Porcupine Lizenzänderungen | Niedrig | Mittel | Keyword-Detection als Fallback |

---

## 10. Implementierungsplan

### Phase 1: Grundgerüst (MVP) ✅ ABGESCHLOSSEN

1. **System Tray App** ✅
   - Icon in Menüleiste (fyne.io/systray)
   - Kontextmenü (Status, Modell, Aktivieren, Einstellungen, Beenden)
   - Global Hotkey Registration (deaktiviert auf macOS wegen SIGTRAP-Crash)
   - Aktivierung über Menüleisten-Klick

2. **Audio Capture** ✅
   - PortAudio Integration
   - Mikrofon-Auswahl
   - PCM-Aufnahme (16kHz, Mono)

3. **Basic STT** ✅
   - Whisper.cpp CLI Integration
   - HTTP-Fallback zu Whisper-Server
   - Deutsche und englische Sprache

4. **mDW Integration** ✅
   - HTTP-Client für Kant (`/api/v1/chat`)
   - Direkte Ollama-Verbindung als Alternative

5. **Popup-Fenster** ✅
   - Native macOS-Benachrichtigungen (via osascript)
   - Status-Updates während Verarbeitung

### Phase 2: Erweiterte Features ✅ ABGESCHLOSSEN

1. **VAD Integration** ✅
   - WebRTC VAD implementiert
   - Automatische Aufnahme-Ende nach Stille
   - Konfigurierbare Stille-Dauer (Standard: 3s)

2. **TTS** ✅
   - macOS `say` Befehl (nativ, gute Qualität)
   - Piper TTS als Fallback
   - Satz-für-Satz-Streaming für schnelle Antwort
   - Sprachauswahl (Anna, Petra, Markus, etc.)
   - Konfigurierbare Sprechgeschwindigkeit

3. **Streaming** ✅
   - WebSocket-Client für Kant (`/api/v1/chat/ws`)
   - Inkrementelle Anzeige von Antworten
   - Vollständige Chat-History-Unterstützung

4. **Backend-Auswahl** ✅
   - Ollama direkt (schneller, kein mDW nötig)
   - mDW Backend (über Kant → Turing, ermöglicht Agenten)
   - Umschaltbar in den Einstellungen

5. **Settings UI** ✅ (vorgezogen aus Phase 3)
   - Native macOS-Dialoge (via osascript)
   - Modell-Auswahl (automatisch von Ollama)
   - Backend-Auswahl (Ollama/mDW)
   - Sprache, Whisper-Modell, VAD-Einstellungen
   - TTS-Stimme und Geschwindigkeit

### Phase 3: Polish & Optimierung (GEPLANT)

1. **Wake-Word**
   - Porcupine oder Keyword-Detection
   - Always-On Modus

2. **Full-Duplex**
   - Streaming STT
   - Parallele Ein-/Ausgabe

3. **Dialog-Modus**
   - Konversations-Kontext ✅ (bereits implementiert)
   - Automatische Weiterführung

4. **Weitere Optimierungen**
   - Persistente Einstellungen (Datei-basiert)
   - Audio-Geräte-Auswahl
   - Fortschrittsanzeige für Modell-Download

---

## 11. Alternativen

### 11.1 Electron/Web-basiert

**Technologie**: Electron + Web Speech API + Node.js

**Vorteile:**
- Einfachere UI-Entwicklung
- Web Speech API für STT (Chrome)
- Große Community

**Nachteile:**
- Kein Offline-STT (Web Speech API ist Cloud)
- Hoher RAM-Verbrauch (~200MB+)
- Nicht Go-basiert (passt nicht ins mDW-Ökosystem)

**Bewertung**: Nicht empfohlen wegen Cloud-Abhängigkeit

### 11.2 Python-basiert

**Technologie**: Python + PyQt/tkinter + Whisper + Piper

**Vorteile:**
- Einfachere ML-Integration
- Große Auswahl an Libraries
- Schnelle Entwicklung

**Nachteile:**
- Deployment komplexer (Python Runtime)
- Passt nicht ins Go-Ökosystem
- Größere Binaries/Installation

**Bewertung**: Möglich als Prototyp, aber Go bevorzugt

### 11.3 Rust-basiert

**Technologie**: Rust + Tauri + whisper-rs

**Vorteile:**
- Sehr performant
- Kleine Binaries
- Gute Whisper-Bindings

**Nachteile:**
- Passt nicht ins Go-Ökosystem
- Steilere Lernkurve
- Kleinere Community für Desktop-Apps

**Bewertung**: Gute Alternative, aber Go bevorzugt für Konsistenz

---

## 12. Fazit und Empfehlung

### Empfohlener Ansatz

**Go-basierte Implementierung** mit:

1. **fyne.io/systray** für System Tray
2. **gordonklaus/portaudio** für Audio I/O
3. **whisper.cpp Go Bindings** für STT (lokal)
4. **Silero VAD** für Sprechpausen-Erkennung
5. **Piper TTS** (als Subprocess) für Sprachausgabe
6. **Global Hotkey** als primäre Aktivierung
7. **Fyne** für Popup-Fenster

### Nächste Schritte

1. Prototyp mit Hotkey → Aufnahme → STT → mDW → Popup
2. VAD-Integration für automatisches Aufnahme-Ende
3. TTS-Integration für Sprachausgabe
4. Streaming-Optimierung
5. Optional: Wake-Word für Hands-Free-Betrieb

### Geschätzter Aufwand

| Phase | Beschreibung | Komplexität |
|-------|--------------|-------------|
| Phase 1 | MVP (Hotkey, STT, Popup) | Mittel |
| Phase 2 | VAD, TTS, Streaming | Mittel-Hoch |
| Phase 3 | Wake-Word, Full-Duplex, Polish | Hoch |

---

## 13. Referenzen

### Libraries

- [fyne.io/systray](https://pkg.go.dev/fyne.io/systray) - System Tray
- [gordonklaus/portaudio](https://github.com/gordonklaus/portaudio) - Audio I/O
- [whisper.cpp](https://github.com/ggerganov/whisper.cpp) - STT Engine
- [go-whisper](https://github.com/mutablelogic/go-whisper) - Go Whisper Wrapper
- [silero-vad-go](https://pkg.go.dev/github.com/skypro1111/silero-vad-go) - VAD
- [go-webrtcvad](https://github.com/maxhawkins/go-webrtcvad) - WebRTC VAD
- [Piper TTS](https://github.com/rhasspy/piper) - Text-to-Speech
- [Porcupine](https://github.com/Picovoice/porcupine) - Wake Word
- [golang-design/hotkey](https://github.com/golang-design/hotkey) - Global Hotkeys
- [Fyne](https://fyne.io/) - GUI Toolkit

### Artikel

- [Offline Speech-to-Text with Whisper and Golang](https://dev.to/sfundomhlungu/how-to-set-up-offline-speech-to-text-with-whisper-and-golang-399n)
- [Local STT with Go, Vosk, and gRPC](https://medium.com/@etolkachev93/local-continuous-speech-to-text-recognition-with-go-vosk-and-grpc-streaming-fcc9d87b6eff)
- [Building a System Tray App with Go](https://dev.to/osuka42/building-a-simple-system-tray-app-with-go-899)
- [Voice Activity Detection Guide](https://picovoice.ai/blog/complete-guide-voice-activity-detection-vad/)

### mDW Dokumentation

- [Babbage Proto](../api/proto/babbage.proto)
- [Kant Handler](../internal/kant/handler/handler.go)
- [WebSocket Implementation](../internal/kant/handler/websocket.go)

---

*Dokument erstellt im Rahmen des meinDENKWERK (mDW) Projekts*
