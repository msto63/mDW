# Voice Assistant Rust Migration - Konzept

**Dokumentversion**: 1.0.0
**Erstellt**: 2025-12-11
**Autor**: Mike Stoffels mit Claude
**Status**: Konzeptphase

---

## 1. Executive Summary

Dieses Dokument beschreibt das Migrationskonzept des mDW Voice Assistants von Go nach Rust. Die Migration zielt auf:

- **Performance-Optimierung**: Native Rust-Performance ohne CGO-Overhead
- **Single-Source Cross-Compilation**: Ein Codebase fuer Windows (x64), Linux (x64) und macOS (ARM64)
- **Kleinere Binaries**: Rust produziert kompaktere Executables als Go mit CGO
- **Memory Safety**: Rusts Ownership-System verhindert Memory-Leaks
- **Bessere ML-Integration**: Direkte Anbindung an whisper.cpp und andere native Libraries

### Zielplattformen

| Plattform | Architektur | Rust Target Triple |
|-----------|-------------|-------------------|
| Windows | x86_64 | `x86_64-pc-windows-msvc` |
| Linux | x86_64 | `x86_64-unknown-linux-gnu` |
| macOS | ARM64 (Apple Silicon) | `aarch64-apple-darwin` |

---

## 2. Architektur-Ueberblick

### 2.1 Aktuelle Go-Architektur

```
Voice Assistant (Go)
+-- Audio Pipeline (PortAudio via CGO)
+-- VAD (WebRTC via CGO)
+-- STT (Whisper CLI Subprocess)
+-- TTS (Piper Subprocess / macOS say)
+-- UI (fyne.io/systray)
+-- State Machine
+-- Backend Clients (HTTP/WebSocket)
```

### 2.2 Ziel Rust-Architektur

```
Voice Assistant (Rust)
+-- Audio Pipeline (cpal - Pure Rust)
+-- VAD (webrtc-vad-rs / silero-rs)
+-- STT (whisper-rs - Native Bindings)
+-- TTS (piper-rs / platform-native)
+-- UI (tray-icon + rfd/native-dialog)
+-- State Machine (tokio + channels)
+-- Backend Clients (reqwest/tokio-tungstenite)
```

### 2.3 Komponenten-Mapping

| Go-Komponente | Rust-Aequivalent | Typ |
|---------------|------------------|-----|
| `gordonklaus/portaudio` | `cpal` | Pure Rust |
| `go-webrtcvad` | `webrtc-vad` | Native Bindings |
| `whisper.cpp (CLI)` | `whisper-rs` | Native Bindings |
| `Piper (Subprocess)` | `piper-rs` / Native | Hybrid |
| `fyne.io/systray` | `tray-icon` | Pure Rust |
| `golang-design/hotkey` | `global-hotkey` | Pure Rust |
| `fyne (Popup)` | `rfd` + `native-dialog` | Pure Rust |
| `gorilla/websocket` | `tokio-tungstenite` | Pure Rust |
| `net/http` | `reqwest` | Pure Rust |

---

## 3. Plattform-spezifische Unterschiede

### 3.1 Audio-Subsystem

#### Windows (x64)

```rust
// Windows: WASAPI als Backend (Standard in cpal)
#[cfg(target_os = "windows")]
mod audio {
    use cpal::traits::{DeviceTrait, HostTrait, StreamTrait};

    pub fn get_default_host() -> cpal::Host {
        // WASAPI ist Standard auf Windows
        cpal::default_host()
    }

    pub fn get_audio_devices() -> Vec<String> {
        let host = cpal::default_host();
        host.input_devices()
            .unwrap()
            .filter_map(|d| d.name().ok())
            .collect()
    }
}
```

**Besonderheiten Windows:**
- WASAPI als Audio-Backend (low-latency)
- Keine zusaetzlichen Treiber erforderlich
- Exklusive und Shared-Mode unterstuetzt

#### Linux (x64)

```rust
// Linux: ALSA oder PulseAudio
#[cfg(target_os = "linux")]
mod audio {
    use cpal::traits::{DeviceTrait, HostTrait};

    pub fn get_default_host() -> cpal::Host {
        // Versuche PulseAudio, fallback zu ALSA
        #[cfg(feature = "jack")]
        if let Ok(host) = cpal::host_from_id(cpal::HostId::Jack) {
            return host;
        }
        cpal::default_host() // ALSA
    }
}
```

**Besonderheiten Linux:**
- ALSA als Basis-Backend
- PulseAudio/PipeWire fuer Desktop-Integration
- Optional: JACK fuer professionelle Audio-Setups
- Build-Dependency: `libasound2-dev`

#### macOS (ARM64)

```rust
// macOS: CoreAudio
#[cfg(target_os = "macos")]
mod audio {
    use cpal::traits::{DeviceTrait, HostTrait};

    pub fn get_default_host() -> cpal::Host {
        // CoreAudio ist Standard auf macOS
        cpal::default_host()
    }

    // macOS-spezifisch: Audio Unit fuer TTS
    pub fn use_system_tts(text: &str, voice: &str, rate: f32) {
        use std::process::Command;
        Command::new("say")
            .args(["-v", voice, "-r", &rate.to_string(), text])
            .spawn()
            .expect("Failed to execute say command");
    }
}
```

**Besonderheiten macOS:**
- CoreAudio als Backend (exzellente Latenz)
- Native `say` Befehl fuer TTS
- Code Signing erforderlich fuer Distribution
- Notarization fuer Gatekeeper

### 3.2 System Tray

#### Plattformunabhaengige Abstraktion

```rust
use tray_icon::{TrayIconBuilder, menu::{Menu, MenuItem, PredefinedMenuItem}};

pub struct SystemTray {
    tray: tray_icon::TrayIcon,
}

impl SystemTray {
    pub fn new() -> Result<Self, Box<dyn std::error::Error>> {
        let menu = Menu::new();

        let status_item = MenuItem::new("Status: Bereit", false, None);
        let separator = PredefinedMenuItem::separator();
        let model_item = MenuItem::new("Modell: mistral:7b", true, None);
        let settings_item = MenuItem::new("Einstellungen...", true, None);
        let quit_item = MenuItem::new("Beenden", true, None);

        menu.append(&status_item)?;
        menu.append(&separator)?;
        menu.append(&model_item)?;
        menu.append(&settings_item)?;
        menu.append(&separator)?;
        menu.append(&quit_item)?;

        let icon = Self::load_icon();

        let tray = TrayIconBuilder::new()
            .with_menu(Box::new(menu))
            .with_tooltip("mDW Voice Assistant")
            .with_icon(icon)
            .build()?;

        Ok(Self { tray })
    }

    #[cfg(target_os = "windows")]
    fn load_icon() -> tray_icon::Icon {
        // Windows: ICO Format
        tray_icon::Icon::from_resource(1, None)
            .expect("Failed to load icon")
    }

    #[cfg(target_os = "linux")]
    fn load_icon() -> tray_icon::Icon {
        // Linux: PNG Format (fuer verschiedene DE's)
        let icon_data = include_bytes!("../assets/icon.png");
        tray_icon::Icon::from_rgba(
            image::load_from_memory(icon_data).unwrap().to_rgba8().into_raw(),
            32, 32
        ).unwrap()
    }

    #[cfg(target_os = "macos")]
    fn load_icon() -> tray_icon::Icon {
        // macOS: Template Image (automatisch hell/dunkel)
        let icon_data = include_bytes!("../assets/icon_template.png");
        tray_icon::Icon::from_rgba(
            image::load_from_memory(icon_data).unwrap().to_rgba8().into_raw(),
            22, 22  // macOS Menu Bar Standard
        ).unwrap()
    }
}
```

#### Plattform-Unterschiede

| Aspekt | Windows | Linux | macOS |
|--------|---------|-------|-------|
| Icon-Format | ICO (multi-res) | PNG (verschiedene Groessen) | PNG Template (22x22) |
| Icon-Groesse | 16x16, 32x32, 48x48 | 22x22, 24x24 | 22x22 (Retina: 44x44) |
| Tray-Position | Rechts unten | Variiert (DE-abhaengig) | Rechts oben |
| Dark Mode | Automatisch (Win10+) | DE-abhaengig | Template Images |

### 3.3 Global Hotkeys

```rust
use global_hotkey::{GlobalHotKeyManager, HotKey, hotkey::Code, hotkey::Modifiers};

pub struct HotkeyManager {
    manager: GlobalHotKeyManager,
    activation_hotkey: HotKey,
}

impl HotkeyManager {
    pub fn new() -> Result<Self, Box<dyn std::error::Error>> {
        let manager = GlobalHotKeyManager::new()?;

        // Plattformspezifische Modifier
        #[cfg(target_os = "macos")]
        let modifiers = Modifiers::SUPER | Modifiers::SHIFT;  // Cmd+Shift

        #[cfg(not(target_os = "macos"))]
        let modifiers = Modifiers::CONTROL | Modifiers::SHIFT;  // Ctrl+Shift

        let activation_hotkey = HotKey::new(Some(modifiers), Code::KeyM);
        manager.register(activation_hotkey)?;

        Ok(Self { manager, activation_hotkey })
    }

    pub fn get_hotkey_description(&self) -> &'static str {
        #[cfg(target_os = "macos")]
        return "Cmd+Shift+M";

        #[cfg(not(target_os = "macos"))]
        return "Ctrl+Shift+M";
    }
}
```

### 3.4 Dateisystem-Pfade

```rust
use directories::ProjectDirs;

pub struct AppPaths {
    pub config_dir: std::path::PathBuf,
    pub data_dir: std::path::PathBuf,
    pub models_dir: std::path::PathBuf,
    pub settings_file: std::path::PathBuf,
}

impl AppPaths {
    pub fn new() -> Option<Self> {
        let proj_dirs = ProjectDirs::from("de", "meinDENKWERK", "VoiceAssistant")?;

        let config_dir = proj_dirs.config_dir().to_path_buf();
        let data_dir = proj_dirs.data_dir().to_path_buf();
        let models_dir = data_dir.join("models");
        let settings_file = config_dir.join("settings.json");

        // Verzeichnisse erstellen falls nicht vorhanden
        std::fs::create_dir_all(&config_dir).ok()?;
        std::fs::create_dir_all(&models_dir).ok()?;

        Some(Self {
            config_dir,
            data_dir,
            models_dir,
            settings_file,
        })
    }
}
```

**Resultierende Pfade:**

| Plattform | Config | Data | Models |
|-----------|--------|------|--------|
| Windows | `%APPDATA%\meinDENKWERK\VoiceAssistant\config` | `%LOCALAPPDATA%\meinDENKWERK\VoiceAssistant\data` | `..\data\models` |
| Linux | `~/.config/voice-assistant` | `~/.local/share/voice-assistant` | `..\data\models` |
| macOS | `~/Library/Application Support/de.meinDENKWERK.VoiceAssistant` | gleich | `..\models` |

### 3.5 TTS-Abstraktionsschicht

```rust
pub trait TextToSpeech: Send + Sync {
    fn speak(&self, text: &str) -> Result<(), TtsError>;
    fn speak_async(&self, text: &str) -> impl std::future::Future<Output = Result<(), TtsError>>;
    fn stop(&self);
    fn set_voice(&mut self, voice: &str) -> Result<(), TtsError>;
    fn set_rate(&mut self, rate: f32);
    fn available_voices(&self) -> Vec<String>;
}

// macOS Native TTS
#[cfg(target_os = "macos")]
pub struct MacOsTts {
    voice: String,
    rate: f32,
}

#[cfg(target_os = "macos")]
impl TextToSpeech for MacOsTts {
    fn speak(&self, text: &str) -> Result<(), TtsError> {
        use std::process::Command;

        let status = Command::new("say")
            .args(["-v", &self.voice, "-r", &self.rate.to_string(), text])
            .status()?;

        if status.success() {
            Ok(())
        } else {
            Err(TtsError::SpeakFailed)
        }
    }

    fn available_voices(&self) -> Vec<String> {
        // macOS: say -v ? listet alle Stimmen
        let output = std::process::Command::new("say")
            .args(["-v", "?"])
            .output()
            .expect("Failed to list voices");

        String::from_utf8_lossy(&output.stdout)
            .lines()
            .filter_map(|line| line.split_whitespace().next())
            .map(String::from)
            .collect()
    }

    // ... weitere Implementierungen
}

// Cross-Platform Piper TTS
pub struct PiperTts {
    binary_path: std::path::PathBuf,
    model_path: std::path::PathBuf,
    voice: String,
}

impl TextToSpeech for PiperTts {
    fn speak(&self, text: &str) -> Result<(), TtsError> {
        use std::process::{Command, Stdio};
        use std::io::Write;

        let mut child = Command::new(&self.binary_path)
            .args([
                "--model", self.model_path.to_str().unwrap(),
                "--output_raw",
            ])
            .stdin(Stdio::piped())
            .stdout(Stdio::piped())
            .spawn()?;

        if let Some(mut stdin) = child.stdin.take() {
            stdin.write_all(text.as_bytes())?;
        }

        let output = child.wait_with_output()?;

        // Audio abspielen via cpal
        self.play_raw_audio(&output.stdout)?;

        Ok(())
    }

    // ... weitere Implementierungen
}

// Factory fuer plattformspezifische TTS
pub fn create_tts(config: &TtsConfig) -> Box<dyn TextToSpeech> {
    match config.engine.as_str() {
        #[cfg(target_os = "macos")]
        "macos" | "native" => Box::new(MacOsTts::new(&config.voice, config.rate)),

        "piper" | _ => Box::new(PiperTts::new(
            &config.piper_binary,
            &config.piper_model,
            &config.voice,
        )),
    }
}
```

---

## 4. Build-System und Cross-Compilation

### 4.1 Cargo.toml Konfiguration

```toml
[package]
name = "mdw-voice-assistant"
version = "0.1.0"
edition = "2021"
authors = ["Mike Stoffels <mike@meindenkwerk.de>"]
description = "Voice Assistant fuer meinDENKWERK"
license = "MIT"
repository = "https://github.com/msto63/mDW"

[features]
default = ["webrtc-vad"]
silero-vad = ["ort"]
webrtc-vad = ["webrtc-vad-rs"]
gpu = ["whisper-rs/cuda"]  # Optional: GPU-Beschleunigung

[dependencies]
# Async Runtime
tokio = { version = "1.35", features = ["full"] }

# Audio
cpal = "0.15"
hound = "3.5"  # WAV-Dateien

# STT
whisper-rs = "0.11"

# VAD
webrtc-vad-rs = { version = "0.1", optional = true }
ort = { version = "2.0", optional = true }  # ONNX Runtime fuer Silero

# TTS (Piper als Subprocess, keine direkte Dependency)

# System Tray & UI
tray-icon = "0.14"
global-hotkey = "0.5"
rfd = "0.14"  # Native File Dialogs
image = "0.25"

# Networking
reqwest = { version = "0.11", features = ["json", "stream"] }
tokio-tungstenite = "0.21"
url = "2.5"

# Serialization
serde = { version = "1.0", features = ["derive"] }
serde_json = "1.0"
toml = "0.8"

# Utilities
directories = "5.0"
log = "0.4"
env_logger = "0.11"
thiserror = "1.0"
anyhow = "1.0"

# Platform-specific
[target.'cfg(target_os = "windows")'.dependencies]
winapi = { version = "0.3", features = ["winuser", "shellapi"] }

[target.'cfg(target_os = "macos")'.dependencies]
cocoa = "0.25"
objc = "0.2"

[target.'cfg(target_os = "linux")'.dependencies]
# Keine zusaetzlichen Dependencies

[build-dependencies]
# Fuer Icon-Embedding auf Windows
winres = "0.1"

[profile.release]
opt-level = 3
lto = true
codegen-units = 1
strip = true
panic = "abort"
```

### 4.2 Build-Scripts

#### build.rs (Windows Icon Embedding)

```rust
fn main() {
    #[cfg(target_os = "windows")]
    {
        let mut res = winres::WindowsResource::new();
        res.set_icon("assets/icon.ico");
        res.set("ProductName", "mDW Voice Assistant");
        res.set("FileDescription", "meinDENKWERK Voice Assistant");
        res.set("LegalCopyright", "Copyright 2025 Mike Stoffels");
        res.compile().unwrap();
    }
}
```

#### Makefile fuer Cross-Compilation

```makefile
# Makefile fuer mDW Voice Assistant (Rust)

VERSION := $(shell grep '^version' Cargo.toml | head -1 | cut -d'"' -f2)
BINARY_NAME := mdw-voice

# Targets
WINDOWS_TARGET := x86_64-pc-windows-msvc
LINUX_TARGET := x86_64-unknown-linux-gnu
MACOS_TARGET := aarch64-apple-darwin

# Output Directories
BUILD_DIR := target
DIST_DIR := dist

.PHONY: all clean build-all build-windows build-linux build-macos

all: build-all

clean:
	cargo clean
	rm -rf $(DIST_DIR)

# === Native Build (aktuelle Plattform) ===
build:
	cargo build --release

# === Cross-Compilation ===

# Windows (von Linux/macOS)
build-windows:
	@echo "Building for Windows x64..."
	cargo build --release --target $(WINDOWS_TARGET)
	@mkdir -p $(DIST_DIR)/windows
	@cp $(BUILD_DIR)/$(WINDOWS_TARGET)/release/$(BINARY_NAME).exe $(DIST_DIR)/windows/

# Linux (von macOS/Windows)
build-linux:
	@echo "Building for Linux x64..."
	cargo build --release --target $(LINUX_TARGET)
	@mkdir -p $(DIST_DIR)/linux
	@cp $(BUILD_DIR)/$(LINUX_TARGET)/release/$(BINARY_NAME) $(DIST_DIR)/linux/

# macOS ARM64 (von Linux/Windows)
build-macos:
	@echo "Building for macOS ARM64..."
	cargo build --release --target $(MACOS_TARGET)
	@mkdir -p $(DIST_DIR)/macos
	@cp $(BUILD_DIR)/$(MACOS_TARGET)/release/$(BINARY_NAME) $(DIST_DIR)/macos/

# Alle Plattformen
build-all: build-windows build-linux build-macos
	@echo "All platforms built successfully"
	@ls -la $(DIST_DIR)/*/

# === Packaging ===

package-windows: build-windows
	@echo "Creating Windows installer..."
	# NSIS oder WiX hier

package-linux: build-linux
	@echo "Creating Linux packages..."
	# .deb und .rpm erstellen

package-macos: build-macos
	@echo "Creating macOS app bundle..."
	@mkdir -p $(DIST_DIR)/macos/mDW\ Voice.app/Contents/MacOS
	@mkdir -p $(DIST_DIR)/macos/mDW\ Voice.app/Contents/Resources
	@cp $(DIST_DIR)/macos/$(BINARY_NAME) "$(DIST_DIR)/macos/mDW Voice.app/Contents/MacOS/"
	@cp assets/Info.plist "$(DIST_DIR)/macos/mDW Voice.app/Contents/"
	@cp assets/icon.icns "$(DIST_DIR)/macos/mDW Voice.app/Contents/Resources/"

# === Development ===

dev:
	cargo run

test:
	cargo test

lint:
	cargo clippy -- -D warnings

fmt:
	cargo fmt

# === CI/CD Integration ===

ci: lint test build-all
```

### 4.3 Cross-Compilation Setup

#### Voraussetzungen

```bash
# Rust Targets installieren
rustup target add x86_64-pc-windows-msvc
rustup target add x86_64-unknown-linux-gnu
rustup target add aarch64-apple-darwin

# Cross-Compiler (Linux)
# Fuer Windows:
sudo apt install mingw-w64

# Fuer macOS (erfordert osxcross):
# https://github.com/tpoechtrager/osxcross
```

#### .cargo/config.toml

```toml
[target.x86_64-pc-windows-msvc]
linker = "lld-link"
# Oder mit MinGW:
# linker = "x86_64-w64-mingw32-gcc"

[target.x86_64-unknown-linux-gnu]
linker = "x86_64-linux-gnu-gcc"

[target.aarch64-apple-darwin]
linker = "aarch64-apple-darwin-clang"

[build]
# Parallele Jobs
jobs = 8

[profile.release]
# Optimierungen
lto = true
codegen-units = 1
opt-level = 3
```

### 4.4 GitHub Actions CI/CD

```yaml
# .github/workflows/build.yml
name: Build Voice Assistant

on:
  push:
    branches: [main]
    tags: ['v*']
  pull_request:
    branches: [main]

env:
  CARGO_TERM_COLOR: always

jobs:
  build-windows:
    runs-on: windows-latest
    steps:
      - uses: actions/checkout@v4

      - name: Install Rust
        uses: dtolnay/rust-toolchain@stable

      - name: Build
        run: cargo build --release

      - name: Upload Artifact
        uses: actions/upload-artifact@v4
        with:
          name: windows-x64
          path: target/release/mdw-voice.exe

  build-linux:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Install Dependencies
        run: |
          sudo apt-get update
          sudo apt-get install -y libasound2-dev libayatana-appindicator3-dev

      - name: Install Rust
        uses: dtolnay/rust-toolchain@stable

      - name: Build
        run: cargo build --release

      - name: Upload Artifact
        uses: actions/upload-artifact@v4
        with:
          name: linux-x64
          path: target/release/mdw-voice

  build-macos:
    runs-on: macos-14  # M1 Runner
    steps:
      - uses: actions/checkout@v4

      - name: Install Rust
        uses: dtolnay/rust-toolchain@stable
        with:
          targets: aarch64-apple-darwin

      - name: Build
        run: cargo build --release --target aarch64-apple-darwin

      - name: Create App Bundle
        run: |
          mkdir -p "dist/mDW Voice.app/Contents/MacOS"
          mkdir -p "dist/mDW Voice.app/Contents/Resources"
          cp target/aarch64-apple-darwin/release/mdw-voice "dist/mDW Voice.app/Contents/MacOS/"
          cp assets/Info.plist "dist/mDW Voice.app/Contents/"
          cp assets/icon.icns "dist/mDW Voice.app/Contents/Resources/"

      - name: Upload Artifact
        uses: actions/upload-artifact@v4
        with:
          name: macos-arm64
          path: dist/mDW Voice.app

  release:
    needs: [build-windows, build-linux, build-macos]
    runs-on: ubuntu-latest
    if: startsWith(github.ref, 'refs/tags/v')
    steps:
      - name: Download Artifacts
        uses: actions/download-artifact@v4

      - name: Create Release
        uses: softprops/action-gh-release@v1
        with:
          files: |
            windows-x64/mdw-voice.exe
            linux-x64/mdw-voice
            macos-arm64/mDW Voice.app
```

---

## 5. Rust-Projektstruktur

```
mdw-voice-assistant/
+-- Cargo.toml
+-- Cargo.lock
+-- build.rs
+-- .cargo/
|   +-- config.toml
+-- src/
|   +-- main.rs                 # Entry Point
|   +-- lib.rs                  # Library Root
|   +-- app.rs                  # Application Controller
|   +-- state.rs                # State Machine
|   +-- config.rs               # Configuration
|   +-- error.rs                # Error Types
|   |
|   +-- audio/
|   |   +-- mod.rs
|   |   +-- capture.rs          # Audio Input (cpal)
|   |   +-- playback.rs         # Audio Output (cpal)
|   |   +-- buffer.rs           # Ring Buffer
|   |
|   +-- vad/
|   |   +-- mod.rs
|   |   +-- traits.rs           # VAD Interface
|   |   +-- webrtc.rs           # WebRTC VAD
|   |   +-- silero.rs           # Silero VAD (optional)
|   |
|   +-- stt/
|   |   +-- mod.rs
|   |   +-- traits.rs           # STT Interface
|   |   +-- whisper.rs          # Whisper Implementation
|   |   +-- streaming.rs        # Streaming STT
|   |
|   +-- tts/
|   |   +-- mod.rs
|   |   +-- traits.rs           # TTS Interface
|   |   +-- piper.rs            # Piper TTS
|   |   +-- macos.rs            # macOS Native TTS
|   |
|   +-- ui/
|   |   +-- mod.rs
|   |   +-- tray.rs             # System Tray
|   |   +-- popup.rs            # Response Popup
|   |   +-- settings.rs         # Settings Dialog
|   |   +-- hotkey.rs           # Global Hotkeys
|   |
|   +-- client/
|   |   +-- mod.rs
|   |   +-- ollama.rs           # Ollama Client
|   |   +-- mdw.rs              # mDW API Client
|   |   +-- websocket.rs        # WebSocket Handler
|   |
|   +-- platform/
|       +-- mod.rs
|       +-- windows.rs          # Windows-spezifisch
|       +-- linux.rs            # Linux-spezifisch
|       +-- macos.rs            # macOS-spezifisch
|
+-- assets/
|   +-- icon.ico                # Windows Icon
|   +-- icon.png                # Linux Icon
|   +-- icon_template.png       # macOS Template
|   +-- icon.icns               # macOS App Icon
|   +-- Info.plist              # macOS App Bundle
|
+-- models/                     # ML-Modelle (gitignored)
|   +-- whisper/
|   +-- piper/
|
+-- tests/
|   +-- integration/
|   +-- audio_tests.rs
|   +-- stt_tests.rs
|
+-- benches/
    +-- stt_benchmark.rs
```

---

## 6. Kern-Implementierungen

### 6.1 Main Entry Point

```rust
// src/main.rs
use mdw_voice_assistant::{App, Config, Error};
use log::{info, error};

#[tokio::main]
async fn main() -> Result<(), Error> {
    // Logger initialisieren
    env_logger::Builder::from_env(
        env_logger::Env::default().default_filter_or("info")
    ).init();

    info!("mDW Voice Assistant v{}", env!("CARGO_PKG_VERSION"));

    // Konfiguration laden
    let config = Config::load().unwrap_or_else(|e| {
        error!("Failed to load config: {}, using defaults", e);
        Config::default()
    });

    // Anwendung starten
    let mut app = App::new(config).await?;

    // Event Loop starten
    app.run().await?;

    Ok(())
}
```

### 6.2 Application Controller

```rust
// src/app.rs
use crate::{
    audio::{AudioCapture, AudioPlayback},
    vad::VoiceActivityDetector,
    stt::SpeechToText,
    tts::TextToSpeech,
    ui::{SystemTray, HotkeyManager},
    client::{OllamaClient, MdwClient},
    state::{State, StateMachine},
    config::Config,
    Error,
};
use tokio::sync::mpsc;

pub struct App {
    config: Config,
    state_machine: StateMachine,
    audio_capture: AudioCapture,
    audio_playback: AudioPlayback,
    vad: Box<dyn VoiceActivityDetector>,
    stt: Box<dyn SpeechToText>,
    tts: Box<dyn TextToSpeech>,
    tray: SystemTray,
    hotkey_manager: HotkeyManager,
    client: Box<dyn LlmClient>,

    // Channels
    audio_tx: mpsc::Sender<Vec<f32>>,
    audio_rx: mpsc::Receiver<Vec<f32>>,
    event_tx: mpsc::Sender<AppEvent>,
    event_rx: mpsc::Receiver<AppEvent>,
}

impl App {
    pub async fn new(config: Config) -> Result<Self, Error> {
        let (audio_tx, audio_rx) = mpsc::channel(100);
        let (event_tx, event_rx) = mpsc::channel(100);

        // Audio initialisieren
        let audio_capture = AudioCapture::new(
            config.audio.sample_rate,
            config.audio.buffer_size,
            audio_tx.clone(),
        )?;

        let audio_playback = AudioPlayback::new(
            config.audio.sample_rate,
        )?;

        // VAD initialisieren
        let vad = crate::vad::create_vad(&config.vad)?;

        // STT initialisieren
        let stt = crate::stt::create_stt(&config.stt)?;

        // TTS initialisieren
        let tts = crate::tts::create_tts(&config.tts)?;

        // UI initialisieren
        let tray = SystemTray::new(event_tx.clone())?;
        let hotkey_manager = HotkeyManager::new(event_tx.clone())?;

        // Client initialisieren
        let client: Box<dyn LlmClient> = if config.backend.use_mdw {
            Box::new(MdwClient::new(&config.backend.mdw_url)?)
        } else {
            Box::new(OllamaClient::new(&config.backend.ollama_url)?)
        };

        Ok(Self {
            config,
            state_machine: StateMachine::new(),
            audio_capture,
            audio_playback,
            vad,
            stt,
            tts,
            tray,
            hotkey_manager,
            client,
            audio_tx,
            audio_rx,
            event_tx,
            event_rx,
        })
    }

    pub async fn run(&mut self) -> Result<(), Error> {
        loop {
            tokio::select! {
                // Audio-Daten verarbeiten
                Some(audio_data) = self.audio_rx.recv() => {
                    self.process_audio(audio_data).await?;
                }

                // Events verarbeiten
                Some(event) = self.event_rx.recv() => {
                    match event {
                        AppEvent::Activate => self.start_listening().await?,
                        AppEvent::Deactivate => self.stop_listening().await?,
                        AppEvent::Quit => break,
                        AppEvent::SettingsChanged(new_config) => {
                            self.update_config(new_config).await?;
                        }
                    }
                }
            }
        }

        Ok(())
    }

    async fn process_audio(&mut self, audio_data: Vec<f32>) -> Result<(), Error> {
        match self.state_machine.current_state() {
            State::Listening => {
                // VAD pruefen
                let is_speech = self.vad.process(&audio_data)?;

                if is_speech {
                    self.state_machine.add_audio(&audio_data);
                } else if self.state_machine.silence_duration() > self.config.vad.silence_duration_ms {
                    // Stille erkannt -> STT starten
                    self.state_machine.transition(State::Processing);
                    self.process_speech().await?;
                }
            }
            _ => {}
        }

        Ok(())
    }

    async fn process_speech(&mut self) -> Result<(), Error> {
        let audio = self.state_machine.get_recorded_audio();

        // STT
        let text = self.stt.transcribe(&audio).await?;

        if text.is_empty() {
            self.state_machine.transition(State::Idle);
            return Ok(());
        }

        // LLM-Anfrage
        self.state_machine.transition(State::Processing);

        let response = self.client.chat(&text, &self.config.backend.model).await?;

        // Antwort anzeigen
        self.state_machine.transition(State::Responding);
        self.tray.show_popup(&response)?;

        // TTS (optional)
        if self.config.tts.enabled {
            self.tts.speak(&response).await?;
        }

        // Zurueck zu Idle oder Dialog-Modus
        if self.config.dialog.enabled {
            self.state_machine.transition(State::Dialog);
        } else {
            self.state_machine.transition(State::Idle);
        }

        Ok(())
    }

    async fn start_listening(&mut self) -> Result<(), Error> {
        self.state_machine.transition(State::Listening);
        self.audio_capture.start()?;
        self.tray.update_status("Hoere zu...");
        Ok(())
    }

    async fn stop_listening(&mut self) -> Result<(), Error> {
        self.audio_capture.stop()?;
        self.state_machine.transition(State::Idle);
        self.tray.update_status("Bereit");
        Ok(())
    }
}
```

### 6.3 Audio Capture (cpal)

```rust
// src/audio/capture.rs
use cpal::traits::{DeviceTrait, HostTrait, StreamTrait};
use tokio::sync::mpsc;
use std::sync::{Arc, atomic::{AtomicBool, Ordering}};

pub struct AudioCapture {
    stream: Option<cpal::Stream>,
    running: Arc<AtomicBool>,
    sample_rate: u32,
    buffer_size: usize,
    sender: mpsc::Sender<Vec<f32>>,
}

impl AudioCapture {
    pub fn new(
        sample_rate: u32,
        buffer_size: usize,
        sender: mpsc::Sender<Vec<f32>>,
    ) -> Result<Self, crate::Error> {
        Ok(Self {
            stream: None,
            running: Arc::new(AtomicBool::new(false)),
            sample_rate,
            buffer_size,
            sender,
        })
    }

    pub fn start(&mut self) -> Result<(), crate::Error> {
        let host = cpal::default_host();
        let device = host.default_input_device()
            .ok_or(crate::Error::NoInputDevice)?;

        let config = cpal::StreamConfig {
            channels: 1,
            sample_rate: cpal::SampleRate(self.sample_rate),
            buffer_size: cpal::BufferSize::Fixed(self.buffer_size as u32),
        };

        let sender = self.sender.clone();
        let running = self.running.clone();
        running.store(true, Ordering::SeqCst);

        let stream = device.build_input_stream(
            &config,
            move |data: &[f32], _: &cpal::InputCallbackInfo| {
                if running.load(Ordering::SeqCst) {
                    let _ = sender.try_send(data.to_vec());
                }
            },
            |err| {
                log::error!("Audio capture error: {}", err);
            },
            None,
        )?;

        stream.play()?;
        self.stream = Some(stream);

        Ok(())
    }

    pub fn stop(&mut self) -> Result<(), crate::Error> {
        self.running.store(false, Ordering::SeqCst);
        if let Some(stream) = self.stream.take() {
            stream.pause()?;
        }
        Ok(())
    }

    pub fn list_devices() -> Vec<String> {
        let host = cpal::default_host();
        host.input_devices()
            .map(|devices| {
                devices
                    .filter_map(|d| d.name().ok())
                    .collect()
            })
            .unwrap_or_default()
    }
}
```

### 6.4 Whisper STT

```rust
// src/stt/whisper.rs
use whisper_rs::{WhisperContext, WhisperContextParameters, FullParams, SamplingStrategy};
use std::path::Path;

pub struct WhisperStt {
    ctx: WhisperContext,
    language: String,
}

impl WhisperStt {
    pub fn new(model_path: &Path, language: &str) -> Result<Self, crate::Error> {
        let params = WhisperContextParameters::default();
        let ctx = WhisperContext::new_with_params(
            model_path.to_str().unwrap(),
            params,
        )?;

        Ok(Self {
            ctx,
            language: language.to_string(),
        })
    }

    pub async fn transcribe(&self, audio: &[f32]) -> Result<String, crate::Error> {
        let mut params = FullParams::new(SamplingStrategy::Greedy { best_of: 1 });

        params.set_language(Some(&self.language));
        params.set_print_special(false);
        params.set_print_progress(false);
        params.set_print_realtime(false);
        params.set_print_timestamps(false);

        // Whisper erwartet 16kHz Mono Float32
        let mut state = self.ctx.create_state()?;
        state.full(params, audio)?;

        let num_segments = state.full_n_segments()?;
        let mut result = String::new();

        for i in 0..num_segments {
            let segment = state.full_get_segment_text(i)?;
            result.push_str(&segment);
            result.push(' ');
        }

        Ok(result.trim().to_string())
    }
}

#[async_trait::async_trait]
impl super::SpeechToText for WhisperStt {
    async fn transcribe(&self, audio: &[f32]) -> Result<String, crate::Error> {
        self.transcribe(audio).await
    }

    fn set_language(&mut self, language: &str) {
        self.language = language.to_string();
    }

    fn supported_languages(&self) -> Vec<&'static str> {
        vec!["de", "en", "fr", "es", "it", "pt", "nl", "pl", "ru", "zh", "ja", "ko", "auto"]
    }
}
```

---

## 7. Migration-Strategie

### 7.1 Phasenplan

#### Phase 1: Foundation (2-3 Wochen)

1. **Projekt-Setup**
   - Cargo.toml mit allen Dependencies
   - CI/CD Pipeline fuer alle Plattformen
   - Basis-Projektstruktur

2. **Audio-Subsystem**
   - cpal Integration
   - Cross-Platform Audio Capture
   - Ring Buffer Implementation

3. **State Machine**
   - Zustandsuebergaenge portieren
   - Event-System mit tokio channels

#### Phase 2: Core Features (3-4 Wochen)

1. **VAD Integration**
   - WebRTC VAD Bindings
   - Silero VAD (optional)
   - Sprechpausen-Erkennung

2. **STT Integration**
   - whisper-rs Einbindung
   - Modell-Management
   - Streaming-Transkription

3. **Backend Clients**
   - Ollama HTTP Client
   - mDW WebSocket Client
   - Streaming-Unterstuetzung

#### Phase 3: UI & Platform (2-3 Wochen)

1. **System Tray**
   - tray-icon Integration
   - Plattformspezifische Icons
   - Kontextmenue

2. **Global Hotkeys**
   - Cross-Platform Hotkey Registration
   - Plattformspezifische Modifier

3. **TTS**
   - Piper Integration
   - macOS Native TTS
   - Satz-Streaming

#### Phase 4: Polish & Testing (2 Wochen)

1. **Testing**
   - Unit Tests fuer alle Module
   - Integration Tests
   - Cross-Platform Testing

2. **Packaging**
   - Windows Installer (NSIS/WiX)
   - Linux Packages (.deb, .rpm, AppImage)
   - macOS App Bundle + Notarization

3. **Dokumentation**
   - API-Dokumentation
   - Benutzerhandbuch
   - Build-Anleitung

### 7.2 Risiken und Mitigationen

| Risiko | Wahrscheinlichkeit | Auswirkung | Mitigation |
|--------|-------------------|------------|------------|
| whisper-rs API-Inkompatibilitaet | Mittel | Hoch | Version pinnen, Abstraktionsschicht |
| cpal Bugs auf spezifischen Geraeten | Niedrig | Mittel | Fallback-Audio-Backend |
| Cross-Compilation Probleme | Mittel | Mittel | Native Builds auf CI |
| macOS Notarization Aenderungen | Niedrig | Mittel | Apple Developer Account |
| Performance-Regression | Niedrig | Hoch | Benchmarks, Profiling |

### 7.3 Kompatibilitaet mit bestehendem mDW

Die Rust-Version kommuniziert mit dem Go-Backend ueber dieselben APIs:

- **HTTP**: `/api/v1/chat` (Kant)
- **WebSocket**: `ws://localhost:8080/api/v1/chat/ws`
- **gRPC**: Keine direkte Nutzung (Kant als Gateway)

Keine Aenderungen am Backend erforderlich.

---

## 8. Vergleich Go vs. Rust

### 8.1 Performance-Erwartungen

| Metrik | Go (aktuell) | Rust (erwartet) | Verbesserung |
|--------|--------------|-----------------|--------------|
| Binary-Groesse | ~25 MB | ~8 MB | -68% |
| RAM (Idle) | ~50 MB | ~15 MB | -70% |
| Startup Zeit | ~200 ms | ~50 ms | -75% |
| STT Latenz | ~800 ms | ~600 ms | -25% |
| CPU (Recording) | ~15% | ~10% | -33% |

### 8.2 Feature-Paritaet

| Feature | Go | Rust |
|---------|:--:|:----:|
| Audio Capture | OK | OK |
| WebRTC VAD | OK | OK |
| Whisper STT | OK (CLI) | OK (Native) |
| Piper TTS | OK | OK |
| macOS TTS | OK | OK |
| System Tray | OK | OK |
| Global Hotkeys | OK | OK |
| WebSocket Streaming | OK | OK |
| Dialog Mode | OK | OK |
| Settings Persistence | OK | OK |

### 8.3 Vorteile der Migration

1. **Performance**: Native Bindings statt CLI-Subprocesses
2. **Memory Safety**: Keine Memory Leaks durch Ownership
3. **Kleinere Binaries**: LTO und bessere Optimierung
4. **Bessere Async**: tokio ist ausgereifter als Go-Goroutines fuer I/O
5. **Type Safety**: Strengeres Typsystem verhindert Runtime-Fehler
6. **Cross-Compilation**: Einfacher ohne CGO

### 8.4 Nachteile der Migration

1. **Lernkurve**: Rust erfordert mehr Einarbeitung
2. **Kompilierzeit**: Laenger als Go
3. **Ecosystem**: Go hat mehr Libraries fuer Enterprise
4. **Team-Alignment**: Restlicher mDW-Stack ist Go

---

## 9. Fazit und Empfehlung

### Empfehlung

Die Migration nach Rust ist **empfohlen** fuer den Voice Assistant, da:

1. **Performance kritisch**: Audio-Verarbeitung profitiert von nativen Bindings
2. **Standalone-Anwendung**: Kein enger Coupling mit Go-Backend
3. **Cross-Platform**: Rust's Build-System ist besser fuer Multi-Platform
4. **Zukunftssicher**: Rust-Ecosystem waechst schnell im ML-Bereich

### Naechste Schritte

1. **Proof of Concept**: Audio Capture + Whisper in Rust (1 Woche)
2. **Entscheidung**: Go-to-Production oder Weiterentwicklung
3. **Vollstaendige Migration**: Nach erfolgreichem PoC

### Geschaetzter Aufwand

| Phase | Dauer | Personen |
|-------|-------|----------|
| Phase 1: Foundation | 2-3 Wochen | 1 |
| Phase 2: Core | 3-4 Wochen | 1 |
| Phase 3: UI | 2-3 Wochen | 1 |
| Phase 4: Polish | 2 Wochen | 1 |
| **Gesamt** | **9-12 Wochen** | **1** |

---

## 10. Referenzen

### Rust Crates

- [cpal](https://crates.io/crates/cpal) - Cross-Platform Audio
- [whisper-rs](https://crates.io/crates/whisper-rs) - Whisper Bindings
- [tray-icon](https://crates.io/crates/tray-icon) - System Tray
- [global-hotkey](https://crates.io/crates/global-hotkey) - Hotkeys
- [tokio](https://crates.io/crates/tokio) - Async Runtime
- [tokio-tungstenite](https://crates.io/crates/tokio-tungstenite) - WebSocket

### Dokumentation

- [Rust Cross-Compilation Guide](https://rust-lang.github.io/rustup/cross-compilation.html)
- [whisper.cpp](https://github.com/ggerganov/whisper.cpp)
- [Piper TTS](https://github.com/rhasspy/piper)

---

*Dokument erstellt im Rahmen des meinDENKWERK (mDW) Projekts*
