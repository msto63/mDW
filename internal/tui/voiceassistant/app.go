// ============================================================================
// meinDENKWERK (mDW) - Lokale KI-Plattform
// ============================================================================
//
// Package:     voiceassistant
// Description: Main application controller
// Author:      Mike Stoffels with Claude
// Created:     2025-12-07
// License:     MIT
// ============================================================================

package voiceassistant

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"strings"
	"sync"
	"time"

	"golang.design/x/hotkey"

	"github.com/msto63/mDW/internal/tui/voiceassistant/audio"
	"github.com/msto63/mDW/internal/tui/voiceassistant/client"
	"github.com/msto63/mDW/internal/tui/voiceassistant/stt"
	"github.com/msto63/mDW/internal/tui/voiceassistant/tts"
	"github.com/msto63/mDW/internal/tui/voiceassistant/ui"
	"github.com/msto63/mDW/internal/tui/voiceassistant/vad"
	"github.com/msto63/mDW/internal/tui/voiceassistant/wakeword"
	"github.com/msto63/mDW/pkg/core/logging"
)

// App is the main voice assistant application
type App struct {
	mu     sync.RWMutex
	config Config
	logger *logging.Logger

	// State
	state    *StateMachine
	ctx      context.Context
	cancel   context.CancelFunc
	running  bool

	// Components
	audioCapture *audio.Capture
	audioBuffer  *audio.AudioBuffer
	playback     *audio.Playback
	vadDetector  *vad.WebRTCVAD
	vadTracker   *vad.SpeechTracker
	transcriber  stt.Transcriber
	synthesizer  tts.Synthesizer
	mdwClient    *client.MDWClient
	ollamaClient *client.OllamaClient
	wsClient     *client.WSClient

	// UI
	tray           *ui.TrayApp
	popup          *ui.PopupWindow
	settingsServer *ui.WebSettingsServer
	hotkey         *hotkey.Hotkey

	// Wake word
	wakeWordDetector *wakeword.Detector

	// Recording state
	recordingCtx    context.Context
	recordingCancel context.CancelFunc

	// Dialog mode state
	dialogCtx    context.Context
	dialogCancel context.CancelFunc

	// Streaming STT state
	lastStreamingTranscription time.Time
	streamingText              string

	// Idle timeout state (30s timeout after response)
	idleTimeoutCancel context.CancelFunc

	// Conversation history
	history []client.Message
}

// New creates a new voice assistant application
func New(cfg Config) (*App, error) {
	logger := logging.New("voice-assistant")

	ctx, cancel := context.WithCancel(context.Background())

	app := &App{
		config:      cfg,
		logger:      logger,
		state:       NewStateMachine(),
		ctx:         ctx,
		cancel:      cancel,
		audioBuffer: audio.NewAudioBuffer(),
		history:     make([]client.Message, 0),
	}

	// Initialize components
	if err := app.initComponents(); err != nil {
		cancel()
		return nil, fmt.Errorf("failed to initialize components: %w", err)
	}

	return app, nil
}

// initComponents initializes all components
func (a *App) initComponents() error {
	var err error

	// Audio capture
	a.audioCapture, err = audio.NewCapture(audio.CaptureConfig{
		SampleRate: float64(a.config.SampleRate),
		BufferSize: a.config.BufferSize,
		Channels:   1,
	})
	if err != nil {
		return fmt.Errorf("failed to create audio capture: %w", err)
	}

	// Audio playback
	a.playback = audio.NewPlayback(audio.PlaybackConfig{
		SampleRate: 22050, // Piper default
		Channels:   1,
	})

	// VAD
	a.vadDetector, err = vad.NewWebRTCVAD(vad.Config{
		SampleRate:        a.config.SampleRate,
		Mode:              2,
		SilenceDuration:   time.Duration(a.config.SilenceDurationMs) * time.Millisecond,
		MinSpeechDuration: time.Duration(a.config.MinSpeechDurationMs) * time.Millisecond,
	})
	if err != nil {
		return fmt.Errorf("failed to create VAD: %w", err)
	}

	a.vadTracker = vad.NewSpeechTracker(vad.Config{
		SilenceDuration:   time.Duration(a.config.SilenceDurationMs) * time.Millisecond,
		MinSpeechDuration: time.Duration(a.config.MinSpeechDurationMs) * time.Millisecond,
	})

	// STT - Select engine based on config
	switch a.config.STTEngine {
	case "voxtral":
		a.logger.Info("Using Voxtral STT engine", "url", a.config.VoxtralURL)
		a.transcriber = stt.NewVoxtralClient(stt.VoxtralConfig{
			BaseURL:        a.config.VoxtralURL,
			Model:          a.config.VoxtralModel,
			Language:       a.config.Language,
			SampleRate:     a.config.SampleRate,
			TimeoutSeconds: 60,
		})
	default: // "whisper" or empty
		sttCfg := stt.Config{
			ModelPath:  a.config.WhisperModelPath,
			Language:   a.config.Language,
			SampleRate: a.config.SampleRate,
		}
		a.transcriber, err = stt.NewWhisperCLI(sttCfg)
		if err != nil {
			a.logger.Warn("Whisper CLI not available, using HTTP fallback", "error", err)
			// Use HTTP client as fallback (requires running whisper server)
			a.transcriber = stt.NewWhisperHTTP("http://localhost:8000", sttCfg)
		}
	}

	// TTS
	if a.config.TTSEnabled {
		a.synthesizer, err = tts.NewPiperTTS(tts.Config{
			BinaryPath: a.config.PiperBinary,
			ModelPath:  a.config.PiperModelPath,
			SampleRate: 22050,
		})
		if err != nil {
			a.logger.Warn("Piper TTS not available", "error", err)
			// TTS is optional, continue without it
		}
	}

	// Ollama client (direct connection)
	a.ollamaClient = client.NewOllamaClient(client.OllamaConfig{
		BaseURL:        a.config.OllamaURL,
		Model:          a.config.MDWModel,
		TimeoutSeconds: a.config.TimeoutSeconds,
	})

	// mDW client (via backend)
	a.mdwClient = client.NewMDWClient(client.Config{
		BaseURL:        a.config.MDWAPIURL,
		WebSocketURL:   a.config.MDWWebSocketURL,
		Model:          a.config.MDWModel,
		TimeoutSeconds: a.config.TimeoutSeconds,
	})

	// WebSocket client for streaming (mDW backend)
	a.wsClient = client.NewWSClient(a.config.MDWWebSocketURL, a.config.MDWModel)

	// UI Popup (creates the Fyne app)
	a.popup = ui.NewPopupWindow(ui.PopupConfig{
		Width:  float32(a.config.PopupWidth),
		Height: float32(a.config.PopupHeight),
	})

	// System tray (shares the same Fyne app)
	a.tray = ui.NewTrayApp(ui.TrayCallbacks{
		OnActivate: a.Activate,
		OnSettings: a.ShowSettings,
		OnQuit:     a.Quit,
	})

	// Set initial model display
	backendInfo := "Ollama"
	if !a.config.UseDirect {
		backendInfo = "mDW"
	}
	a.tray.SetModel(a.config.MDWModel + " (" + backendInfo + ")")

	// Wake word detector
	a.wakeWordDetector = wakeword.NewDetector(
		wakeword.Config{
			Keywords:       []string{a.config.WakeWord},
			Sensitivity:    a.config.WakeWordSensitivity,
			SampleRate:     a.config.SampleRate,
			BufferDuration: 2 * time.Second,
			CheckInterval:  500 * time.Millisecond,
		},
		a.transcriber,
	)
	a.wakeWordDetector.SetOnDetected(func() {
		// Only activate if idle
		if a.state.Current() == StateIdle {
			a.logger.Info("Wake word detected, activating...")
			a.Activate()
		}
	})

	// State change listener
	a.state.AddListener(func(oldState, newState State) {
		a.logger.Debug("State changed", "from", oldState.String(), "to", newState.String())
		a.tray.SetStatus(newState.String())

		switch newState {
		case StateIdle:
			a.tray.SetIcon("idle")
		case StateListening:
			a.tray.SetIcon("recording")
		case StateProcessing:
			a.tray.SetIcon("processing")
		case StateError:
			a.tray.SetIcon("error")
		}
	})

	return nil
}

// Run starts the voice assistant
func (a *App) Run() error {
	a.mu.Lock()
	if a.running {
		a.mu.Unlock()
		return fmt.Errorf("already running")
	}
	a.running = true
	a.mu.Unlock()

	// Show version on console
	fmt.Printf("mDW Voice Assistant v%s\n", Version)
	fmt.Println("────────────────────────────")

	a.logger.Info("Starting Voice Assistant", "version", Version)

	// Register global hotkey (skipped on macOS)
	if err := a.registerHotkey(); err != nil {
		a.logger.Warn("Failed to register hotkey", "error", err)
	}

	// Initialize popup (uses native macOS notifications instead of Fyne)
	a.popup.Initialize()

	// Initialize web settings server
	a.settingsServer = ui.NewWebSettingsServer(a.getCurrentSettings(), a.applySettings)

	// Start periodic backend health check
	go a.startBackendHealthCheck()

	// Start wake word detection if enabled
	if a.config.WakeWordEnabled {
		go a.startWakeWordDetection()
	}

	// Run system tray (blocking)
	// The systray library runs its own event loop
	a.tray.Run()

	return nil
}

// startWakeWordDetection starts the wake word detection loop
func (a *App) startWakeWordDetection() {
	a.logger.Info("Starting wake word detection", "keyword", a.config.WakeWord)

	// Create a dedicated audio capture for wake word detection
	wakeWordCapture, err := audio.NewCapture(audio.CaptureConfig{
		SampleRate: float64(a.config.SampleRate),
		BufferSize: a.config.BufferSize,
		Channels:   1,
		DeviceName: a.config.InputDevice,
	})
	if err != nil {
		a.logger.Error("Failed to create wake word audio capture", "error", err)
		return
	}

	// Start the detector
	if err := a.wakeWordDetector.Start(a.ctx, wakeWordCapture); err != nil {
		a.logger.Error("Failed to start wake word detection", "error", err)
		wakeWordCapture.Close()
		return
	}

	// Wait for context cancellation
	<-a.ctx.Done()
	a.wakeWordDetector.Stop()
	wakeWordCapture.Close()
}

// registerHotkey registers the global hotkey
// Note: On macOS, the golang.design/x/hotkey library can cause SIGTRAP crashes
// due to CGO and Objective-C runtime issues. We skip hotkey registration on macOS
// and rely on menu bar activation instead.
func (a *App) registerHotkey() error {
	// Skip hotkey registration on macOS due to known SIGTRAP crashes
	// The user can activate via clicking the menu bar icon instead
	if runtime.GOOS == "darwin" {
		a.logger.Info("Hotkey disabled on macOS (use menu bar to activate)")
		return nil
	}

	var mods []hotkey.Modifier

	// Ctrl+Shift+M on Linux/Windows
	mods = []hotkey.Modifier{hotkey.ModCtrl, hotkey.ModShift}

	a.hotkey = hotkey.New(mods, hotkey.KeyM)

	if err := a.hotkey.Register(); err != nil {
		return fmt.Errorf("failed to register hotkey: %w", err)
	}

	// Listen for hotkey events
	go func() {
		for range a.hotkey.Keydown() {
			a.logger.Debug("Hotkey pressed")
			a.Activate()
		}
	}()

	a.logger.Info("Hotkey registered", "shortcut", a.config.GetShortcutDescription())
	return nil
}

// Activate activates voice recording
func (a *App) Activate() {
	a.mu.Lock()
	currentState := a.state.Current()
	a.mu.Unlock()

	// If already listening, stop
	if currentState == StateListening {
		a.stopRecording()
		return
	}

	// If not idle, ignore
	if currentState != StateIdle {
		return
	}

	a.startRecording()
}

// startRecording starts audio recording
func (a *App) startRecording() {
	if !a.state.Transition(StateListening) {
		return
	}

	// Cancel any pending timeouts from previous round
	a.mu.Lock()
	if a.dialogCancel != nil {
		a.dialogCancel()
		a.dialogCancel = nil
	}
	if a.idleTimeoutCancel != nil {
		a.idleTimeoutCancel()
		a.idleTimeoutCancel = nil
	}
	// Reset streaming STT state
	a.lastStreamingTranscription = time.Time{}
	a.streamingText = ""
	a.mu.Unlock()

	a.audioBuffer.Clear()
	a.vadTracker.Reset()

	// Show listening UI
	a.popup.ShowListening()

	// Create recording context
	a.recordingCtx, a.recordingCancel = context.WithCancel(a.ctx)

	// Start audio capture
	if err := a.audioCapture.Start(a.recordingCtx); err != nil {
		a.logger.Error("Failed to start audio capture", "error", err)
		a.state.Transition(StateError)
		a.popup.ShowError(err)
		return
	}

	// Process audio in goroutine
	go a.processAudio()
}

// processAudio processes incoming audio
func (a *App) processAudio() {
	for {
		select {
		case <-a.recordingCtx.Done():
			return

		case samples, ok := <-a.audioCapture.Output():
			if !ok {
				return
			}

			// Add to buffer
			a.audioBuffer.Append(samples)

			// Check VAD
			isSpeech, err := a.vadDetector.Process(samples)
			if err != nil {
				a.logger.Warn("VAD error", "error", err)
				continue
			}

			// Update tracker
			a.vadTracker.Update(isSpeech)

			// Check if we should stop recording
			if a.vadTracker.ShouldEndRecording() {
				a.logger.Debug("Speech ended (silence detected)")
				a.stopRecording()
				return
			}

			// Update UI with speech state
			state := a.vadTracker.State()
			if state.IsSpeaking {
				a.popup.SetStatus(fmt.Sprintf("Aufnahme... %.1fs", state.SpeechDuration.Seconds()))
			} else if state.SilenceDuration > 0 {
				remaining := time.Duration(a.config.SilenceDurationMs)*time.Millisecond - state.SilenceDuration
				if remaining > 0 {
					a.popup.SetStatus(fmt.Sprintf("Pause erkannt... %.1fs", remaining.Seconds()))
				}
			}

			// Streaming STT: periodically transcribe while recording
			a.tryStreamingTranscription()
		}
	}
}

// tryStreamingTranscription attempts a streaming transcription if enabled and interval has passed
func (a *App) tryStreamingTranscription() {
	a.mu.RLock()
	streamingEnabled := a.config.StreamingSTT
	streamingInterval := a.config.StreamingInterval
	lastTranscription := a.lastStreamingTranscription
	a.mu.RUnlock()

	if !streamingEnabled {
		return
	}

	// Check if enough time has passed since last transcription
	if time.Since(lastTranscription) < time.Duration(streamingInterval)*time.Millisecond {
		return
	}

	// Need minimum audio to transcribe (at least 0.5 seconds)
	minSamples := a.config.SampleRate / 2
	if a.audioBuffer.Len() < minSamples {
		return
	}

	// Update last transcription time before starting (to prevent concurrent transcriptions)
	a.mu.Lock()
	a.lastStreamingTranscription = time.Now()
	a.mu.Unlock()

	// Do transcription in goroutine to not block audio processing
	go func() {
		samples := a.audioBuffer.Get()

		ctx, cancel := context.WithTimeout(a.recordingCtx, 5*time.Second)
		defer cancel()

		result, err := a.transcriber.Transcribe(ctx, samples)
		if err != nil {
			a.logger.Debug("Streaming transcription failed", "error", err)
			return
		}

		if result.Text == "" {
			return
		}

		// Update streaming text and UI
		a.mu.Lock()
		a.streamingText = result.Text
		a.mu.Unlock()

		// Show intermediate result in popup
		a.popup.SetContent("**Erkannt:** " + result.Text + "\n\n_(Aufnahme läuft...)_")
		a.logger.Debug("Streaming transcription", "text", result.Text)
	}()
}

// stopRecording stops audio recording and processes the result
func (a *App) stopRecording() {
	if a.recordingCancel != nil {
		a.recordingCancel()
	}

	if err := a.audioCapture.Stop(); err != nil {
		a.logger.Warn("Failed to stop audio capture", "error", err)
	}

	// Check if we have enough audio
	if !a.vadTracker.IsValidSpeech() {
		a.logger.Debug("Not enough speech captured")
		a.state.Reset()
		a.popup.Hide()
		return
	}

	// Process the recorded audio
	a.processRecordedAudio()
}

// processRecordedAudio processes the recorded audio
func (a *App) processRecordedAudio() {
	if !a.state.Transition(StateProcessing) {
		return
	}

	samples := a.audioBuffer.Get()
	a.logger.Debug("Processing audio", "samples", len(samples), "duration", a.audioBuffer.DurationSeconds(float64(a.config.SampleRate)))

	// Transcribe
	a.popup.SetStatus("Transkribiere...")

	ctx, cancel := context.WithTimeout(a.ctx, 30*time.Second)
	defer cancel()

	result, err := a.transcriber.Transcribe(ctx, samples)
	if err != nil {
		a.logger.Error("Transcription failed", "error", err)
		a.state.Transition(StateError)
		a.popup.ShowError(fmt.Errorf("Transkription fehlgeschlagen: %w", err))
		return
	}

	userText := result.Text
	if userText == "" {
		a.logger.Debug("Empty transcription")
		a.state.Reset()
		a.popup.Hide()
		return
	}

	a.logger.Info("Transcription", "text", userText)

	// Show processing state
	a.popup.ShowProcessing(userText)

	// Send to mDW
	a.sendToMDW(userText)
}

// sendToMDW sends the transcribed text to mDW or directly to Ollama
func (a *App) sendToMDW(userText string) {
	// Add to history
	a.history = append(a.history, client.Message{
		Role:    "user",
		Content: userText,
	})

	ctx, cancel := context.WithTimeout(a.ctx, time.Duration(a.config.TimeoutSeconds)*time.Second)
	defer cancel()

	var response string
	var err error

	// Print user input to console
	fmt.Printf("\n[Sie] %s\n", userText)

	if a.config.UseDirect {
		// Direct Ollama connection (no mDW backend required)
		var responseBuilder string

		fmt.Print("[mDW] ")
		err = a.ollamaClient.ChatStreamWithHistory(ctx, a.history, func(chunk string, done bool) {
			responseBuilder += chunk
			fmt.Print(chunk) // Stream to console
			a.popup.SetContent("**Sie:** " + userText + "\n\n**Antwort:** " + responseBuilder)
			if done {
				response = responseBuilder
				fmt.Println() // Newline after response
			}
		})

		if err != nil {
			a.logger.Debug("Ollama streaming failed, using non-streaming", "error", err)
			response, err = a.ollamaClient.ChatWithHistory(ctx, a.history)
			if err == nil {
				fmt.Printf("%s\n", response)
			}
		}
	} else {
		// Use mDW backend (Kant → Turing)
		var responseBuilder string

		fmt.Print("[mDW] ")
		// Use WebSocket with full conversation history
		err = a.wsClient.ChatStreamWithHistory(ctx, a.history, func(chunk string, done bool) {
			responseBuilder += chunk
			fmt.Print(chunk) // Stream to console
			a.popup.SetContent("**Sie:** " + userText + "\n\n**mDW:** " + responseBuilder)
			if done {
				response = responseBuilder
				fmt.Println() // Newline after response
			}
		})

		if err != nil {
			a.logger.Debug("WebSocket streaming failed, using HTTP", "error", err)
			response, err = a.mdwClient.ChatWithHistory(ctx, a.history)
			if err == nil {
				fmt.Printf("%s\n", response)
			}
		}
	}

	if err != nil {
		a.logger.Error("Chat request failed", "error", err)
		a.state.Transition(StateError)
		a.popup.ShowError(fmt.Errorf("Anfrage fehlgeschlagen: %w", err))
		fmt.Printf("[Fehler] %v\n", err)
		return
	}

	// Add response to history
	a.history = append(a.history, client.Message{
		Role:    "assistant",
		Content: response,
	})

	// Show response
	a.state.Transition(StateResponding)
	a.popup.ShowResponse(userText, response)

	// Speak response if TTS is enabled
	if a.config.TTSEnabled {
		go a.speakResponse(response)
	}

	// Return to idle after a delay (cancellable)
	a.mu.Lock()
	// Cancel any previous idle timeout
	if a.idleTimeoutCancel != nil {
		a.idleTimeoutCancel()
	}
	idleCtx, idleCancel := context.WithCancel(a.ctx)
	a.idleTimeoutCancel = idleCancel
	a.mu.Unlock()

	go func() {
		select {
		case <-idleCtx.Done():
			// Timeout was cancelled (dialog continued or new recording started)
			return
		case <-time.After(30 * time.Second):
			a.mu.RLock()
			if a.state.Current() == StateResponding {
				a.state.Reset()
				a.popup.Hide()
			}
			a.mu.RUnlock()
		}
	}()
}

// speakResponse speaks the response using TTS
func (a *App) speakResponse(text string) {
	// Use a separate context with longer timeout for TTS
	// Don't tie to app context so TTS can continue even if state changes
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	// Try Piper TTS first
	if a.synthesizer != nil {
		audioData, err := a.synthesizer.Synthesize(ctx, text)
		if err != nil {
			a.logger.Debug("Piper TTS not available, using macOS say", "error", err)
		} else if len(audioData) > 0 {
			// Play audio
			if err := a.playback.PlayRaw(audioData, float64(a.synthesizer.SampleRate())); err != nil {
				a.logger.Debug("Audio playback failed", "error", err)
			} else {
				// TTS completed successfully - check for dialog mode
				a.continueDialogIfEnabled()
				return // Success with Piper
			}
		}
	}

	// Fallback to macOS say command (sentence by sentence for faster feedback)
	voice := a.config.TTSVoice
	if voice == "" {
		voice = "Anna"
	}
	macosSay := tts.NewMacOSSayWithRate(voice, a.config.TTSRate)
	if macosSay.IsAvailable() {
		if err := macosSay.SpeakStreaming(ctx, text); err != nil {
			// Only log if not a context cancellation (which is expected when interrupted)
			if ctx.Err() == nil {
				a.logger.Debug("macOS say error", "error", err)
			}
		}
	}

	// TTS completed - check for dialog mode
	a.continueDialogIfEnabled()
}

// continueDialogIfEnabled starts listening again if dialog mode is enabled
func (a *App) continueDialogIfEnabled() {
	a.mu.RLock()
	dialogMode := a.config.DialogMode
	dialogTimeout := a.config.DialogTimeout
	a.mu.RUnlock()

	if !dialogMode {
		return
	}

	// Short delay before starting to listen again
	time.Sleep(300 * time.Millisecond)

	// Check if we're still in responding state
	if a.state.Current() != StateResponding {
		return
	}

	a.logger.Debug("Dialog mode: auto-continuing conversation")

	// Cancel any previous dialog timeout
	a.mu.Lock()
	if a.dialogCancel != nil {
		a.dialogCancel()
	}
	// Create new dialog context for this round
	a.dialogCtx, a.dialogCancel = context.WithCancel(a.ctx)
	dialogCtx := a.dialogCtx
	a.mu.Unlock()

	// Update UI to show we're ready for next input
	a.popup.SetStatus("Warte auf Ihre Antwort...")

	// Set up timeout if configured
	if dialogTimeout > 0 {
		go func() {
			select {
			case <-dialogCtx.Done():
				// Dialog was cancelled (new recording started), do nothing
				return
			case <-time.After(time.Duration(dialogTimeout) * time.Second):
				// Timeout reached
				a.mu.RLock()
				currentState := a.state.Current()
				a.mu.RUnlock()

				// If still responding (not listening/processing), reset to idle
				if currentState == StateResponding {
					a.logger.Debug("Dialog mode: timeout reached, returning to idle")
					a.state.Reset()
					a.popup.Hide()
				}
			}
		}()
	}

	// Start listening for next input
	a.startRecording()
}

// ShowSettings shows the settings dialog
func (a *App) ShowSettings() {
	a.logger.Debug("Settings requested")
	// Stop old server if running
	if a.settingsServer != nil {
		a.settingsServer.Stop()
	}
	// Create new server with current settings
	a.settingsServer = ui.NewWebSettingsServer(a.getCurrentSettings(), a.applySettings)
	a.settingsServer.Show()
}

// getCurrentSettings returns the current settings for the settings window
func (a *App) getCurrentSettings() ui.Settings {
	ttsRate := a.config.TTSRate
	if ttsRate == 0 {
		ttsRate = 220 // Default rate
	}
	dialogTimeout := a.config.DialogTimeout
	if dialogTimeout == 0 {
		dialogTimeout = 10 // Default timeout
	}
	streamingInterval := a.config.StreamingInterval
	if streamingInterval == 0 {
		streamingInterval = 2000 // Default interval
	}
	sttEngine := a.config.STTEngine
	if sttEngine == "" {
		sttEngine = "whisper"
	}
	voxtralURL := a.config.VoxtralURL
	if voxtralURL == "" {
		voxtralURL = "http://localhost:8100"
	}
	voxtralModel := a.config.VoxtralModel
	if voxtralModel == "" {
		voxtralModel = "mistralai/Voxtral-Mini-3B-2507"
	}
	return ui.Settings{
		Model:             a.config.MDWModel,
		OllamaURL:         a.config.OllamaURL,
		UseMDW:            !a.config.UseDirect,
		MDWURL:            a.config.MDWAPIURL,
		Language:          a.config.Language,
		WhisperModel:      a.config.WhisperModel,
		SilenceDurationMs: a.config.SilenceDurationMs,
		TTSEnabled:        a.config.TTSEnabled,
		TTSVoice:          a.config.TTSVoice,
		TTSRate:           ttsRate,
		VADThreshold:      float64(a.config.VADThreshold),
		DialogMode:        a.config.DialogMode,
		DialogTimeout:     dialogTimeout,
		InputDevice:       a.config.InputDevice,
		WakeWordEnabled:   a.config.WakeWordEnabled,
		WakeWord:          a.config.WakeWord,
		StreamingSTT:      a.config.StreamingSTT,
		StreamingInterval: streamingInterval,
		STTEngine:         sttEngine,
		VoxtralURL:        voxtralURL,
		VoxtralModel:      voxtralModel,
	}
}

// applySettings applies new settings from the settings window
func (a *App) applySettings(settings ui.Settings) {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.logger.Info("Applying new settings",
		"model", settings.Model,
		"language", settings.Language,
		"ttsEnabled", settings.TTSEnabled,
		"ttsRate", settings.TTSRate,
		"silenceDurationMs", settings.SilenceDurationMs,
		"useMDW", settings.UseMDW,
		"dialogMode", settings.DialogMode,
		"dialogTimeout", settings.DialogTimeout,
		"streamingSTT", settings.StreamingSTT,
		"streamingInterval", settings.StreamingInterval,
		"sttEngine", settings.STTEngine,
		"voxtralURL", settings.VoxtralURL,
	)

	// Update config
	a.config.MDWModel = settings.Model
	a.config.OllamaURL = settings.OllamaURL
	a.config.UseDirect = !settings.UseMDW
	a.config.MDWAPIURL = settings.MDWURL
	a.config.Language = settings.Language
	a.config.WhisperModel = settings.WhisperModel
	a.config.SilenceDurationMs = settings.SilenceDurationMs
	a.config.TTSEnabled = settings.TTSEnabled
	a.config.TTSVoice = settings.TTSVoice
	a.config.TTSRate = settings.TTSRate
	a.config.VADThreshold = float32(settings.VADThreshold)
	a.config.DialogMode = settings.DialogMode
	a.config.DialogTimeout = settings.DialogTimeout
	a.config.InputDevice = settings.InputDevice
	a.config.WakeWordEnabled = settings.WakeWordEnabled
	a.config.WakeWord = settings.WakeWord
	a.config.StreamingSTT = settings.StreamingSTT
	a.config.StreamingInterval = settings.StreamingInterval
	a.config.STTEngine = settings.STTEngine
	a.config.VoxtralURL = settings.VoxtralURL
	a.config.VoxtralModel = settings.VoxtralModel

	// Update STT engine if changed
	if settings.STTEngine == "voxtral" {
		a.logger.Info("Switching to Voxtral STT engine", "url", settings.VoxtralURL)
		if a.transcriber != nil {
			a.transcriber.Close()
		}
		a.transcriber = stt.NewVoxtralClient(stt.VoxtralConfig{
			BaseURL:        settings.VoxtralURL,
			Model:          settings.VoxtralModel,
			Language:       settings.Language,
			SampleRate:     a.config.SampleRate,
			TimeoutSeconds: 60,
		})
	} else if settings.STTEngine == "whisper" || settings.STTEngine == "" {
		// Only switch if we're not already using Whisper
		if _, isVoxtral := a.transcriber.(*stt.VoxtralClient); isVoxtral {
			a.logger.Info("Switching to Whisper STT engine")
			if a.transcriber != nil {
				a.transcriber.Close()
			}
			var err error
			sttCfg := stt.Config{
				ModelPath:  a.config.WhisperModelPath,
				Language:   settings.Language,
				SampleRate: a.config.SampleRate,
			}
			a.transcriber, err = stt.NewWhisperCLI(sttCfg)
			if err != nil {
				a.logger.Warn("Whisper CLI not available, using HTTP fallback", "error", err)
				a.transcriber = stt.NewWhisperHTTP("http://localhost:8000", sttCfg)
			}
		}
	}

	// Update audio capture device
	if a.audioCapture != nil {
		a.audioCapture.SetDeviceName(settings.InputDevice)
		a.logger.Debug("Updated audio input device", "device", settings.InputDevice)
	}

	// Update wake word detector
	if a.wakeWordDetector != nil {
		a.wakeWordDetector.SetKeywords([]string{settings.WakeWord})
		if settings.WakeWordEnabled && !a.wakeWordDetector.IsRunning() {
			go a.startWakeWordDetection()
		} else if !settings.WakeWordEnabled && a.wakeWordDetector.IsRunning() {
			a.wakeWordDetector.Stop()
		}
		a.logger.Debug("Updated wake word settings", "enabled", settings.WakeWordEnabled, "word", settings.WakeWord)
	}

	// Update Ollama client
	if a.ollamaClient != nil {
		a.ollamaClient.SetModel(settings.Model)
		a.ollamaClient.SetBaseURL(settings.OllamaURL)
	}

	// Update WebSocket client for mDW
	if a.wsClient != nil {
		a.wsClient.Close()
	}
	wsURL := settings.MDWURL + "/api/v1/chat/ws"
	// Convert http:// to ws://
	wsURL = strings.Replace(wsURL, "http://", "ws://", 1)
	wsURL = strings.Replace(wsURL, "https://", "wss://", 1)
	a.config.MDWWebSocketURL = wsURL
	a.wsClient = client.NewWSClient(wsURL, settings.Model)

	// Update mDW HTTP client
	if a.mdwClient != nil {
		// Recreate with new settings
		a.mdwClient = client.NewMDWClient(client.Config{
			BaseURL:        settings.MDWURL,
			WebSocketURL:   wsURL,
			Model:          settings.Model,
			TimeoutSeconds: a.config.TimeoutSeconds,
		})
	}

	// Update STT language
	if a.transcriber != nil {
		// Try to set language if the transcriber supports it
		if whisperCLI, ok := a.transcriber.(*stt.WhisperCLI); ok {
			whisperCLI.SetLanguage(settings.Language)
			a.logger.Debug("Updated WhisperCLI language", "language", settings.Language)
		} else if whisperHTTP, ok := a.transcriber.(*stt.WhisperHTTP); ok {
			whisperHTTP.SetLanguage(settings.Language)
			a.logger.Debug("Updated WhisperHTTP language", "language", settings.Language)
		} else if voxtralClient, ok := a.transcriber.(*stt.VoxtralClient); ok {
			voxtralClient.SetLanguage(settings.Language)
			a.logger.Debug("Updated Voxtral language", "language", settings.Language)
		}
	}

	// Update VAD silence duration
	if a.vadTracker != nil {
		a.vadTracker.SetSilenceDuration(time.Duration(settings.SilenceDurationMs) * time.Millisecond)
		a.logger.Debug("Updated VAD silence duration", "ms", settings.SilenceDurationMs)
	}

	// Update tray model display
	backendInfo := "Ollama"
	if settings.UseMDW {
		backendInfo = "mDW"
	}
	a.tray.SetModel(settings.Model + " (" + backendInfo + ")")

	// Save settings to file
	if err := a.saveSettingsToFile(); err != nil {
		a.logger.Warn("Failed to save settings", "error", err)
	}

	a.logger.Info("Settings applied successfully")
}

// Quit quits the application
func (a *App) Quit() {
	a.logger.Info("Shutting down Voice Assistant")

	a.mu.Lock()
	a.running = false
	a.mu.Unlock()

	// Cancel context
	if a.cancel != nil {
		a.cancel()
	}

	// Unregister hotkey
	if a.hotkey != nil {
		a.hotkey.Unregister()
	}

	// Close components
	if a.audioCapture != nil {
		a.audioCapture.Close()
	}
	if a.vadDetector != nil {
		a.vadDetector.Close()
	}
	if a.transcriber != nil {
		a.transcriber.Close()
	}
	if a.synthesizer != nil {
		a.synthesizer.Close()
	}
	if a.wsClient != nil {
		a.wsClient.Close()
	}
	if a.settingsServer != nil {
		a.settingsServer.Stop()
	}

	// Exit the process - systray.Quit() alone doesn't terminate on macOS
	a.logger.Info("Exiting process")
	os.Exit(0)
}

// ClearHistory clears the conversation history
func (a *App) ClearHistory() {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.history = make([]client.Message, 0)
}

// startBackendHealthCheck starts periodic health checking of the backend
func (a *App) startBackendHealthCheck() {
	// Initial delay
	time.Sleep(500 * time.Millisecond)

	// Check immediately, then periodically
	a.checkBackendHealth()

	// Periodic check every 30 seconds
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-a.ctx.Done():
			return
		case <-ticker.C:
			a.checkBackendHealth()
		}
	}
}

// checkBackendHealth checks if the backend (Ollama or mDW) is available
func (a *App) checkBackendHealth() {
	a.mu.RLock()
	useDirect := a.config.UseDirect
	a.mu.RUnlock()

	ctx, cancel := context.WithTimeout(a.ctx, 5*time.Second)
	defer cancel()

	if useDirect {
		// Check Ollama directly
		if err := a.ollamaClient.HealthCheck(ctx); err != nil {
			a.logger.Debug("Ollama health check failed", "error", err)
			a.tray.SetBackendStatus(false, "Ollama offline")
			a.tray.SetStatus("Offline")
		} else {
			a.logger.Debug("Ollama health check OK")
			a.tray.SetBackendStatus(true, "")
			a.tray.SetStatus("Bereit")
		}
	} else {
		// Check mDW backend (Kant) with detailed service status
		status := a.mdwClient.GetHealthStatus(ctx)
		if !status.Online {
			a.logger.Debug("mDW health check failed", "error", status.ErrorMessage)
			a.tray.SetBackendStatus(false, "mDW offline")
			a.tray.SetStatus("Offline")
		} else {
			// Log detailed service status
			a.logger.Debug("mDW health check OK",
				"status", status.Status,
				"version", status.Version,
				"services", status.Services,
			)

			// Build status details for menu
			details := a.formatServiceDetails(status)
			if status.Status == "degraded" {
				a.tray.SetBackendStatus(true, details)
				a.tray.SetStatus("Eingeschränkt")
			} else {
				a.tray.SetBackendStatus(true, details)
				a.tray.SetStatus("Bereit")
			}
		}
	}
}

// formatServiceDetails formats the service status for display
func (a *App) formatServiceDetails(status client.HealthStatus) string {
	if len(status.Services) == 0 {
		return ""
	}

	healthyCount := 0
	var unhealthyServices []string

	for name, state := range status.Services {
		if state == "healthy" {
			healthyCount++
		} else {
			unhealthyServices = append(unhealthyServices, name)
		}
	}

	if len(unhealthyServices) == 0 {
		return fmt.Sprintf("%d/%d Services", healthyCount, len(status.Services))
	}

	return fmt.Sprintf("%d/%d (%s)", healthyCount, len(status.Services), strings.Join(unhealthyServices, ", "))
}
