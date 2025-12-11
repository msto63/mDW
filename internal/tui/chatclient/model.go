// ============================================================================
// meinDENKWERK (mDW) - Lokale KI-Plattform
// ============================================================================
//
// Package:     chatclient
// Description: Main Bubbletea model for mDW ChatClient
// Author:      Mike Stoffels with Claude
// Created:     2025-12-07
// License:     MIT
// ============================================================================

package chatclient

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	aristotelepb "github.com/msto63/mDW/api/gen/aristoteles"
	"github.com/msto63/mDW/api/gen/common"
	leibnizpb "github.com/msto63/mDW/api/gen/leibniz"
	turingpb "github.com/msto63/mDW/api/gen/turing"
	"github.com/msto63/mDW/internal/turing/ollama"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// FocusArea represents which area has focus
type FocusArea int

const (
	FocusChat FocusArea = iota
	FocusModelSelector
)

// Model is the main Bubbletea model for ChatClient
type Model struct {
	// State
	width          int
	height         int
	ready          bool
	loading        bool
	streaming      bool
	turingOnline   bool
	focus          FocusArea
	showModelList  bool
	showAgentList  bool // Agent-Auswahl-Overlay
	err            error

	// Components
	textarea textarea.Model
	viewport viewport.Model
	spinner  spinner.Model

	// Chat state
	messages        []ChatMessage
	currentModel    string
	availableModels []ModelInfo
	modelIndex      int
	streamBuffer    *strings.Builder

	// Input history
	inputHistory      []string // Liste der bisherigen Eingaben
	historyIndex      int      // Aktuelle Position in der Historie (-1 = neue Eingabe)
	currentInput      string   // Zwischenspeicher für aktuelle Eingabe beim Navigieren

	// Streaming state
	streamRespCh  <-chan *ollama.ChatResponse
	streamErrCh   <-chan error
	streamStart   time.Time

	// Configuration
	turingAddr   string
	ollamaClient *ollama.Client
	useGRPC      bool

	// Aristoteles Pipeline state
	useAristoteles    bool   // Toggle für Aristoteles-Pipeline
	aristotelesOnline bool   // Aristoteles-Service-Status
	aristotelesAddr   string // Aristoteles gRPC address
	lastIntentType    string // Letzter erkannter Intent-Typ
	lastStrategyName  string // Letzte verwendete Strategie
	lastQualityScore  float32 // Letzter Quality-Score
	currentStep       string // Aktueller Verarbeitungsschritt (für Anzeige)

	// Agent Pipeline state (für UI-Anzeige)
	lastAgentID       string  // Zuletzt verwendeter Agent
	lastAgentName     string  // Name des letzten Agents
	lastAgentConfidence float64 // Confidence des Agent-Matchings
	pipelineAgents    []string // Liste der Agents in der Pipeline (für Multi-Agent)

	// Agent selection state
	availableAgents   []AgentInfo // Liste verfügbarer Agents
	agentIndex        int         // Index des aktuell ausgewählten Agents in der Liste
	selectedAgentID   string      // Manuell ausgewählter Agent-ID ("" = auto)
	selectedAgentName string      // Name des manuell ausgewählten Agents
	leibnizAddr       string      // Leibniz gRPC address für Agent-Abfragen
}

// Config holds ChatClient configuration
type Config struct {
	TuringAddr      string
	AristotelesAddr string
	LeibnizAddr     string // Leibniz address für Agent-Liste
	Model           string
	UseAristoteles  bool // Default: use Aristoteles pipeline
}

// DefaultConfig returns default configuration
func DefaultConfig() Config {
	return Config{
		TuringAddr:      "localhost:9200",
		AristotelesAddr: "localhost:9160",
		LeibnizAddr:     "localhost:9140",
		Model:           "mistral:7b",
		UseAristoteles:  false, // Default: direct Turing (toggle mit Ctrl+A)
	}
}

// New creates a new ChatClient model
func New(cfg Config) Model {
	// Setup textarea
	ta := textarea.New()
	ta.Placeholder = "Nachricht eingeben... (Enter zum Senden, Shift+Enter für neue Zeile)"
	ta.Focus()
	ta.CharLimit = 8000
	ta.SetWidth(80)
	ta.SetHeight(3)
	ta.ShowLineNumbers = false
	ta.FocusedStyle.CursorLine = lipgloss.NewStyle()
	ta.FocusedStyle.Base = FocusedInputStyle
	ta.BlurredStyle.Base = InputStyle

	// Setup spinner
	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = SpinnerStyle

	// Default models
	defaultModels := []ModelInfo{
		{Name: "mistral:7b", Size: "4.1 GB", Description: "Schnell und effizient", Available: true},
		{Name: "qwen2.5:7b", Size: "4.7 GB", Description: "Gut für Code", Available: true},
		{Name: "llama3.2:latest", Size: "2.0 GB", Description: "Meta's neuestes Modell", Available: true},
	}

	// Load saved model, fallback to config default
	model := cfg.Model
	if savedModel := LoadLastModel(); savedModel != "" {
		model = savedModel
	}

	// Load input history
	inputHistory := LoadInputHistory()

	// Apply defaults for empty addresses
	turingAddr := cfg.TuringAddr
	if turingAddr == "" {
		turingAddr = DefaultConfig().TuringAddr
	}
	aristotelesAddr := cfg.AristotelesAddr
	if aristotelesAddr == "" {
		aristotelesAddr = DefaultConfig().AristotelesAddr
	}
	leibnizAddr := cfg.LeibnizAddr
	if leibnizAddr == "" {
		leibnizAddr = DefaultConfig().LeibnizAddr
	}

	// Default agents list with "auto" option
	defaultAgents := []AgentInfo{
		{ID: "", Name: "Auto", Description: "Automatische Agent-Auswahl basierend auf Anfrage"},
	}

	return Model{
		textarea:          ta,
		spinner:           sp,
		messages:          []ChatMessage{},
		currentModel:      model,
		availableModels:   defaultModels,
		streamBuffer:      &strings.Builder{},
		inputHistory:      inputHistory,
		historyIndex:      -1, // -1 bedeutet: keine Historie-Navigation aktiv
		turingAddr:        turingAddr,
		aristotelesAddr:   aristotelesAddr,
		leibnizAddr:       leibnizAddr,
		ollamaClient:      ollama.NewClient(ollama.DefaultConfig()),
		useGRPC:           true,
		useAristoteles:    cfg.UseAristoteles,
		focus:             FocusChat,
		availableAgents:   defaultAgents,
		selectedAgentID:   "", // "" = auto
		selectedAgentName: "Auto",
	}
}

// Init initializes the model
func (m Model) Init() tea.Cmd {
	return tea.Batch(
		textarea.Blink,
		m.spinner.Tick,
		m.checkTuringStatus,
		m.checkAristotelesStatus,
		m.loadModels,
		m.loadAgents,
		tea.EnterAltScreen,
	)
}

// Update handles messages
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m.handleKeyPress(msg)

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		headerHeight := 4  // Logo + model selector
		footerHeight := 8  // Input + status bar + help
		viewportHeight := msg.Height - headerHeight - footerHeight

		if !m.ready {
			m.viewport = viewport.New(msg.Width-4, viewportHeight)
			m.viewport.YPosition = headerHeight
			m.ready = true
		} else {
			m.viewport.Width = msg.Width - 4
			m.viewport.Height = viewportHeight
		}
		m.textarea.SetWidth(msg.Width - 4)
		m.updateViewportContent()

	case spinner.TickMsg:
		if m.loading || m.streaming {
			m.spinner, cmd = m.spinner.Update(msg)
			cmds = append(cmds, cmd)
		}

	case chatResponseMsg:
		m.loading = false
		m.currentStep = "" // Reset step display
		if msg.err != nil {
			m.err = msg.err
			m.messages = append(m.messages, ChatMessage{
				Role:      "system",
				Content:   "Fehler: " + msg.err.Error(),
				Timestamp: time.Now(),
			})
		} else {
			m.messages = append(m.messages, ChatMessage{
				Role:      "assistant",
				Content:   msg.content,
				Model:     m.currentModel,
				Timestamp: time.Now(),
				Duration:  msg.duration,
			})
		}
		m.updateViewportContent()
		m.viewport.GotoBottom()

	case streamChunkMsg:
		if msg.err != nil {
			m.streaming = false
			m.err = msg.err
			m.messages = append(m.messages, ChatMessage{
				Role:      "system",
				Content:   "Fehler: " + msg.err.Error(),
				Timestamp: time.Now(),
			})
		} else if msg.done {
			m.streaming = false
			// Add any final content
			if msg.delta != "" {
				m.streamBuffer.WriteString(msg.delta)
			}
			// Finalize the streamed message with duration
			if m.streamBuffer.Len() > 0 {
				m.messages = append(m.messages, ChatMessage{
					Role:      "assistant",
					Content:   m.streamBuffer.String(),
					Model:     m.currentModel,
					Timestamp: time.Now(),
					Duration:  msg.duration,
				})
				m.streamBuffer.Reset()
			}
		} else {
			// Append chunk and wait for next
			m.streamBuffer.WriteString(msg.delta)
			m.updateViewportContent()
			m.viewport.GotoBottom()
			return m, m.waitForNextChunk()
		}
		m.updateViewportContent()
		m.viewport.GotoBottom()

	case modelsLoadedMsg:
		if msg.err == nil && len(msg.models) > 0 {
			m.availableModels = msg.models
			// Update current model if it's in the list
			for i, model := range m.availableModels {
				if model.Name == m.currentModel {
					m.modelIndex = i
					break
				}
			}
		}

	case serviceStatusMsg:
		m.turingOnline = msg.turingOnline
		if !m.turingOnline {
			m.useGRPC = false
		}

	case aristotelesStatusMsg:
		m.aristotelesOnline = msg.online
		// Wenn Aristoteles offline ist und aktiviert war, deaktivieren
		if !m.aristotelesOnline && m.useAristoteles {
			m.useAristoteles = false
		}

	case aristotelesPipelineMsg:
		m.loading = false
		m.currentStep = "" // Reset step display
		if msg.err != nil {
			m.err = msg.err
			m.messages = append(m.messages, ChatMessage{
				Role:      "system",
				Content:   "Aristoteles-Fehler: " + msg.err.Error(),
				Timestamp: time.Now(),
			})
		} else {
			// Pipeline-Metadaten speichern
			m.lastIntentType = msg.intentType
			m.lastStrategyName = msg.strategyName
			m.lastQualityScore = msg.qualityScore

			// Agent Pipeline Info speichern
			m.lastAgentID = msg.agentID
			m.lastAgentName = msg.agentName
			m.lastAgentConfidence = msg.agentConfidence

			// Model-Info mit Agent-Details formatieren
			modelInfo := m.currentModel
			if msg.agentName != "" {
				// Agent wurde verwendet - zeige Agent-Info
				modelInfo = fmt.Sprintf("%s [Agent: %s, %.0f%%]", msg.targetService, msg.agentName, msg.agentConfidence*100)
			} else {
				// Direkter Service-Aufruf
				modelInfo = fmt.Sprintf("%s [%s, Q:%.0f%%]", m.currentModel, msg.strategyName, msg.qualityScore*100)
			}

			// Antwort mit Pipeline-Info anzeigen
			m.messages = append(m.messages, ChatMessage{
				Role:      "assistant",
				Content:   msg.content,
				Model:     modelInfo,
				Timestamp: time.Now(),
				Duration:  msg.duration,
			})
		}
		m.updateViewportContent()
		m.viewport.GotoBottom()

	case stepUpdateMsg:
		m.currentStep = msg.step

	case agentListMsg:
		if msg.err != nil {
			// Bei Fehler behalten wir die Default-Liste
		} else {
			// Auto-Option beibehalten + geladene Agents
			m.availableAgents = []AgentInfo{
				{ID: "", Name: "Auto", Description: "Automatische Agent-Auswahl basierend auf Anfrage"},
			}
			m.availableAgents = append(m.availableAgents, msg.agents...)
		}

	case tickMsg:
		// Periodic status check
		return m, tea.Batch(
			m.checkTuringStatus,
			m.checkAristotelesStatus,
			tea.Tick(10*time.Second, func(t time.Time) tea.Msg {
				return tickMsg(t)
			}),
		)
	}

	// Update components
	if m.focus == FocusChat && !m.showModelList && !m.showAgentList {
		m.textarea, cmd = m.textarea.Update(msg)
		cmds = append(cmds, cmd)
	}

	m.viewport, cmd = m.viewport.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

// handleKeyPress handles keyboard input
func (m Model) handleKeyPress(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Model selector navigation - handle FIRST when list is shown
	if m.showModelList {
		switch msg.Type {
		case tea.KeyUp:
			if m.modelIndex > 0 {
				m.modelIndex--
			}
			return m, nil

		case tea.KeyDown:
			if m.modelIndex < len(m.availableModels)-1 {
				m.modelIndex++
			}
			return m, nil

		case tea.KeyEnter:
			if m.modelIndex < len(m.availableModels) {
				m.currentModel = m.availableModels[m.modelIndex].Name
				_ = SaveLastModel(m.currentModel)
			}
			m.showModelList = false
			m.focus = FocusChat
			m.textarea.Focus()
			return m, nil

		case tea.KeyEsc:
			m.showModelList = false
			m.focus = FocusChat
			m.textarea.Focus()
			return m, nil

		case tea.KeyRunes:
			// Handle j/k for vim-style navigation
			switch string(msg.Runes) {
			case "k":
				if m.modelIndex > 0 {
					m.modelIndex--
				}
				return m, nil
			case "j":
				if m.modelIndex < len(m.availableModels)-1 {
					m.modelIndex++
				}
				return m, nil
			case " ":
				if m.modelIndex < len(m.availableModels) {
					m.currentModel = m.availableModels[m.modelIndex].Name
					_ = SaveLastModel(m.currentModel)
				}
				m.showModelList = false
				m.focus = FocusChat
				m.textarea.Focus()
				return m, nil
			}
		}
		// Ignore all other keys when model list is open
		return m, nil
	}

	// Agent selector navigation - handle when agent list is shown
	if m.showAgentList {
		switch msg.Type {
		case tea.KeyUp:
			if m.agentIndex > 0 {
				m.agentIndex--
			}
			return m, nil

		case tea.KeyDown:
			if m.agentIndex < len(m.availableAgents)-1 {
				m.agentIndex++
			}
			return m, nil

		case tea.KeyEnter:
			if m.agentIndex < len(m.availableAgents) {
				m.selectedAgentID = m.availableAgents[m.agentIndex].ID
				m.selectedAgentName = m.availableAgents[m.agentIndex].Name
			}
			m.showAgentList = false
			m.focus = FocusChat
			m.textarea.Focus()
			return m, nil

		case tea.KeyEsc:
			m.showAgentList = false
			m.focus = FocusChat
			m.textarea.Focus()
			return m, nil

		case tea.KeyRunes:
			// Handle j/k for vim-style navigation
			switch string(msg.Runes) {
			case "k":
				if m.agentIndex > 0 {
					m.agentIndex--
				}
				return m, nil
			case "j":
				if m.agentIndex < len(m.availableAgents)-1 {
					m.agentIndex++
				}
				return m, nil
			case " ":
				if m.agentIndex < len(m.availableAgents) {
					m.selectedAgentID = m.availableAgents[m.agentIndex].ID
					m.selectedAgentName = m.availableAgents[m.agentIndex].Name
				}
				m.showAgentList = false
				m.focus = FocusChat
				m.textarea.Focus()
				return m, nil
			}
		}
		// Ignore all other keys when agent list is open
		return m, nil
	}

	// Global shortcuts
	switch msg.Type {
	case tea.KeyCtrlC:
		return m, tea.Quit

	case tea.KeyCtrlL:
		// Clear chat
		m.messages = []ChatMessage{}
		m.updateViewportContent()
		return m, nil
	}

	// Check for Ctrl+O (model selector)
	if msg.String() == "ctrl+o" {
		m.showModelList = true
		m.focus = FocusModelSelector
		m.textarea.Blur()
		// Set modelIndex to current model
		for i, model := range m.availableModels {
			if model.Name == m.currentModel {
				m.modelIndex = i
				break
			}
		}
		return m, nil
	}

	// Check for Ctrl+G (Agent selector) - nur im Aristoteles-Modus
	if msg.String() == "ctrl+g" {
		if m.useAristoteles && m.aristotelesOnline {
			m.showAgentList = true
			m.textarea.Blur()
			// Set agentIndex to current agent
			for i, agent := range m.availableAgents {
				if agent.ID == m.selectedAgentID {
					m.agentIndex = i
					break
				}
			}
		} else {
			m.messages = append(m.messages, ChatMessage{
				Role:      "system",
				Content:   "Agent-Auswahl nur im Aristoteles-Modus verfügbar (Ctrl+A zum Aktivieren)",
				Timestamp: time.Now(),
			})
			m.updateViewportContent()
			m.viewport.GotoBottom()
		}
		return m, nil
	}

	// Check for Ctrl+A (Aristoteles toggle)
	if msg.String() == "ctrl+a" {
		if m.aristotelesOnline {
			m.useAristoteles = !m.useAristoteles
			var statusMsg string
			if m.useAristoteles {
				statusMsg = "Aristoteles-Pipeline aktiviert (intelligentes Prompt-Routing)"
			} else {
				statusMsg = "Aristoteles-Pipeline deaktiviert (direkt zu Turing)"
			}
			m.messages = append(m.messages, ChatMessage{
				Role:      "system",
				Content:   statusMsg,
				Timestamp: time.Now(),
			})
			m.updateViewportContent()
			m.viewport.GotoBottom()
		} else {
			m.messages = append(m.messages, ChatMessage{
				Role:      "system",
				Content:   "Aristoteles-Service nicht verfügbar",
				Timestamp: time.Now(),
			})
			m.updateViewportContent()
			m.viewport.GotoBottom()
		}
		return m, nil
	}

	// Chat input handling
	if m.focus == FocusChat && !m.loading && !m.streaming {
		switch msg.Type {
		case tea.KeyEnter:
			// Send message with streaming
			input := strings.TrimSpace(m.textarea.Value())
			if input != "" {
				// Zur Historie hinzufügen (nur wenn nicht identisch mit letztem Eintrag)
				if len(m.inputHistory) == 0 || m.inputHistory[len(m.inputHistory)-1] != input {
					m.inputHistory = append(m.inputHistory, input)
					// Historie auf maximal 100 Einträge begrenzen
					if len(m.inputHistory) > 100 {
						m.inputHistory = m.inputHistory[len(m.inputHistory)-100:]
					}
					// Historie speichern
					_ = SaveInputHistory(m.inputHistory)
				}
				// Historie-Index zurücksetzen
				m.historyIndex = -1
				m.currentInput = ""

				m.messages = append(m.messages, ChatMessage{
					Role:      "user",
					Content:   input,
					Timestamp: time.Now(),
				})
				m.textarea.Reset()
				m.updateViewportContent()
				m.viewport.GotoBottom()

				// Entscheide zwischen Aristoteles-Pipeline und direktem Streaming
				if m.useAristoteles && m.aristotelesOnline {
					m.loading = true
					m.currentStep = StepAnalyzing
					// Reset Agent-Info bis neue Antwort kommt
					m.lastAgentID = ""
					m.lastAgentName = ""
					m.lastAgentConfidence = 0
					return m, tea.Batch(
						m.spinner.Tick,
						m.sendMessageViaAristoteles(input),
						m.scheduleStepUpdates(),
					)
				}
				// Standard: Direktes Streaming via Turing/Ollama
				m.streaming = true
				m.streamBuffer.Reset()
				return m, tea.Batch(
					m.spinner.Tick,
					m.sendMessageWithStreaming(input),
				)
			}
			return m, nil

		case tea.KeyUp:
			// Nach oben in der Historie navigieren
			if len(m.inputHistory) > 0 {
				if m.historyIndex == -1 {
					// Erste Navigation: aktuelle Eingabe speichern
					m.currentInput = m.textarea.Value()
					m.historyIndex = len(m.inputHistory) - 1
				} else if m.historyIndex > 0 {
					m.historyIndex--
				}
				m.textarea.SetValue(m.inputHistory[m.historyIndex])
				// Cursor ans Ende setzen
				m.textarea.CursorEnd()
			}
			return m, nil

		case tea.KeyDown:
			// Nach unten in der Historie navigieren
			if m.historyIndex != -1 {
				if m.historyIndex < len(m.inputHistory)-1 {
					m.historyIndex++
					m.textarea.SetValue(m.inputHistory[m.historyIndex])
				} else {
					// Zurück zur aktuellen Eingabe
					m.historyIndex = -1
					m.textarea.SetValue(m.currentInput)
				}
				// Cursor ans Ende setzen
				m.textarea.CursorEnd()
			}
			return m, nil

		case tea.KeyPgUp:
			m.viewport.ViewUp()
			return m, nil

		case tea.KeyPgDown:
			m.viewport.ViewDown()
			return m, nil
		}
	}

	// Pass other keys to textarea
	var cmd tea.Cmd
	m.textarea, cmd = m.textarea.Update(msg)
	return m, cmd
}

// View renders the UI
func (m Model) View() string {
	if !m.ready {
		return "Lade ChatClient..."
	}

	var b strings.Builder

	// Header with logo and model selector
	b.WriteString(m.renderHeader())
	b.WriteString("\n")

	// If model list is shown, show dropdown instead of chat area
	if m.showModelList {
		b.WriteString(m.renderModelDropdown())
		b.WriteString("\n")
	} else if m.showAgentList {
		// Agent selection dropdown
		b.WriteString(m.renderAgentDropdown())
		b.WriteString("\n")
	} else {
		// Chat viewport
		b.WriteString(m.renderChatArea())
		b.WriteString("\n")

		// Input area
		b.WriteString(m.renderInputArea())
		b.WriteString("\n")
	}

	// Status bar
	b.WriteString(m.renderStatusBar())
	b.WriteString("\n")

	// Help bar
	b.WriteString(m.renderHelpBar())

	return b.String()
}

// renderHeader renders the header with logo and model selector
func (m Model) renderHeader() string {
	// Logo
	logo := LogoStyle.Render(Logo)

	// Status indicator
	var status string
	if m.turingOnline {
		status = StatusOnlineStyle.Render(IconOnline + "Turing verbunden")
	} else {
		status = StatusOfflineStyle.Render(IconOffline + "Turing offline (Ollama direkt)")
	}

	// Model selector
	modelStr := ModelLabelStyle.Render("Modell: ") + SelectedModelItemStyle.Render(m.currentModel)
	if m.showModelList {
		modelStr = FocusedModelSelectorStyle.Render(modelStr)
	} else {
		modelStr = ModelSelectorStyle.Render(modelStr)
	}

	header := lipgloss.JoinHorizontal(lipgloss.Center,
		logo,
		strings.Repeat(" ", 3),
		status,
	)

	titlePanel := TitlePanelStyle.Width(m.width - 4).Render(header)

	// When model list is shown, just return title panel (dropdown rendered separately)
	if m.showModelList {
		return titlePanel
	}

	return titlePanel + "\n" + modelStr
}

// renderModelDropdown renders the model selection dropdown as a full panel
func (m Model) renderModelDropdown() string {
	var content strings.Builder

	// Title
	content.WriteString(HeaderStyle.Render("  Modell auswählen"))
	content.WriteString("\n\n")

	// Model list
	for i, model := range m.availableModels {
		var line string
		if i == m.modelIndex {
			// Selected item with cursor and highlighting
			line = SelectedModelItemStyle.Render(fmt.Sprintf(" ▶ %s ", model.Name))
			if model.Description != "" && model.Description != model.Name {
				line += HelpDescStyle.Render(" - " + model.Description)
			}
			line += HelpDescStyle.Render(fmt.Sprintf(" (%s)", model.Size))
		} else {
			// Normal item
			line = ModelItemStyle.Render(fmt.Sprintf("   %s ", model.Name))
			if model.Description != "" && model.Description != model.Name {
				line += HelpDescStyle.Render(" - " + model.Description)
			}
			line += HelpDescStyle.Render(fmt.Sprintf(" (%s)", model.Size))
		}
		content.WriteString(line)
		content.WriteString("\n")
	}

	// Footer with count and hints
	content.WriteString("\n")
	content.WriteString(HelpDescStyle.Render(fmt.Sprintf("  [%d von %d Modellen]", m.modelIndex+1, len(m.availableModels))))
	content.WriteString("\n\n")
	content.WriteString(HelpStyle.Render("  ↑/↓ navigieren • Enter auswählen • Esc schließen"))

	// Render in a prominent panel
	panelHeight := m.viewport.Height + 2
	return FocusedModelSelectorStyle.
		Width(m.width - 2).
		Height(panelHeight).
		Render(content.String())
}

// renderAgentDropdown renders the agent selection dropdown as a full panel
func (m Model) renderAgentDropdown() string {
	var content strings.Builder

	// Title
	content.WriteString(HeaderStyle.Render("  Agent auswählen"))
	content.WriteString("\n\n")

	// Agent list
	for i, agent := range m.availableAgents {
		var line string
		isSelected := i == m.agentIndex
		isCurrent := agent.ID == m.selectedAgentID

		if isSelected {
			// Selected item with cursor and highlighting
			line = SelectedModelItemStyle.Render(fmt.Sprintf(" ▶ %s ", agent.Name))
			if agent.Description != "" {
				// Truncate long descriptions
				desc := agent.Description
				if len(desc) > 50 {
					desc = desc[:47] + "..."
				}
				line += HelpDescStyle.Render(" - " + desc)
			}
			if isCurrent {
				line += StatusOnlineStyle.Render(" ✓")
			}
		} else {
			// Normal item
			line = ModelItemStyle.Render(fmt.Sprintf("   %s ", agent.Name))
			if agent.Description != "" {
				desc := agent.Description
				if len(desc) > 50 {
					desc = desc[:47] + "..."
				}
				line += HelpDescStyle.Render(" - " + desc)
			}
			if isCurrent {
				line += StatusOnlineStyle.Render(" ✓")
			}
		}
		content.WriteString(line)
		content.WriteString("\n")
	}

	// Footer with count and hints
	content.WriteString("\n")
	if m.selectedAgentID == "" {
		content.WriteString(HelpDescStyle.Render(fmt.Sprintf("  Aktuell: Auto (automatische Auswahl)")))
	} else {
		content.WriteString(HelpDescStyle.Render(fmt.Sprintf("  Aktuell: %s", m.selectedAgentName)))
	}
	content.WriteString("\n")
	content.WriteString(HelpDescStyle.Render(fmt.Sprintf("  [%d von %d Agents]", m.agentIndex+1, len(m.availableAgents))))
	content.WriteString("\n\n")
	content.WriteString(HelpStyle.Render("  ↑/↓ navigieren • Enter auswählen • Esc schließen"))

	// Render in a prominent panel
	panelHeight := m.viewport.Height + 2
	return FocusedModelSelectorStyle.
		Width(m.width - 2).
		Height(panelHeight).
		Render(content.String())
}

// renderChatArea renders the main chat viewport
func (m Model) renderChatArea() string {
	style := ChatPanelStyle.Width(m.width - 2).Height(m.viewport.Height + 2)
	if m.focus == FocusChat {
		style = FocusedChatPanelStyle.Width(m.width - 2).Height(m.viewport.Height + 2)
	}
	return style.Render(m.viewport.View())
}

// renderInputArea renders the input textarea
func (m Model) renderInputArea() string {
	var input string

	if m.loading {
		statusText := " Generiere Antwort..."
		if m.currentStep != "" {
			statusText = fmt.Sprintf(" %s...", m.currentStep)
		}
		input = m.spinner.View() + ThinkingStyle.Render(statusText)
	} else if m.streaming {
		input = m.spinner.View() + ThinkingStyle.Render(" Empfange Antwort...")
	} else {
		input = m.textarea.View()
	}

	style := InputStyle.Width(m.width - 2)
	if m.focus == FocusChat && !m.loading && !m.streaming {
		style = FocusedInputStyle.Width(m.width - 2)
	}

	return style.Render(input)
}

// renderStatusBar renders the status bar with current model and version
func (m Model) renderStatusBar() string {
	// Left: Model info with Aristoteles indicator
	modelInfo := IconModel + SelectedModelItemStyle.Render(m.currentModel)
	if m.useAristoteles {
		modelInfo += HelpDescStyle.Render(" [Pipeline]")
	}

	// Center: Version info + Pipeline status + Agent info
	var centerInfo string
	if m.useAristoteles {
		if m.lastAgentName != "" {
			// Agent wurde verwendet - zeige Agent-Info
			centerInfo = HelpDescStyle.Render(fmt.Sprintf("v%s | Agent: %s (%.0f%%)", Version, m.lastAgentName, m.lastAgentConfidence*100))
		} else if m.lastStrategyName != "" {
			// Keine Agent-Info, aber Strategie vorhanden
			centerInfo = HelpDescStyle.Render(fmt.Sprintf("v%s | %s Q:%.0f%%", Version, m.lastStrategyName, m.lastQualityScore*100))
		} else {
			centerInfo = HelpDescStyle.Render("v" + Version)
		}
	} else {
		centerInfo = HelpDescStyle.Render("v" + Version)
	}

	// Right: Connection status with Aristoteles
	var status string
	if m.useAristoteles && m.aristotelesOnline {
		status = StatusOnlineStyle.Render(IconOnline + "Aristoteles")
	} else if m.turingOnline {
		status = StatusOnlineStyle.Render(IconOnline + "Turing")
	} else {
		status = StatusOfflineStyle.Render(IconOffline + "Ollama")
	}

	// Build the status bar
	leftPart := ModelLabelStyle.Render("Modell: ") + modelInfo
	centerPart := centerInfo
	rightPart := status

	// Calculate padding
	leftLen := lipgloss.Width(leftPart)
	centerLen := lipgloss.Width(centerPart)
	rightLen := lipgloss.Width(rightPart)
	totalLen := leftLen + centerLen + rightLen
	availableSpace := m.width - totalLen - 4
	if availableSpace < 2 {
		availableSpace = 2
	}
	leftPadding := availableSpace / 2
	rightPadding := availableSpace - leftPadding

	content := leftPart + strings.Repeat(" ", leftPadding) + centerPart + strings.Repeat(" ", rightPadding) + rightPart

	return StatusBarStyle.Width(m.width - 2).Render(content)
}

// renderHelpBar renders the help shortcuts bar
func (m Model) renderHelpBar() string {
	var items []string

	if m.showModelList {
		items = []string{
			RenderKeyHint("↑/↓", "navigieren"),
			RenderKeyHint("Enter", "auswählen"),
			RenderKeyHint("Esc", "schließen"),
		}
	} else if m.showAgentList {
		items = []string{
			RenderKeyHint("↑/↓", "navigieren"),
			RenderKeyHint("Enter", "auswählen"),
			RenderKeyHint("Esc", "schließen"),
		}
	} else {
		// Aristoteles-Status im Hint anzeigen
		var aristotelesHint string
		if m.useAristoteles {
			aristotelesHint = "Pipeline aus"
		} else {
			aristotelesHint = "Pipeline an"
		}
		items = []string{
			RenderKeyHint("Enter", "senden"),
			RenderKeyHint("↑/↓", "Historie"),
			RenderKeyHint("Ctrl+O", "Modell"),
			RenderKeyHint("Ctrl+A", aristotelesHint),
		}
		// Zeige Ctrl+G nur im Pipeline-Modus
		if m.useAristoteles {
			items = append(items, RenderKeyHint("Ctrl+G", "Agent"))
		}
		items = append(items, RenderKeyHint("Ctrl+L", "leeren"))
		items = append(items, RenderKeyHint("Ctrl+C", "beenden"))
	}

	return HelpStyle.Render(strings.Join(items, "  "))
}

// updateViewportContent updates the viewport with current messages
func (m *Model) updateViewportContent() {
	var content strings.Builder

	for _, msg := range m.messages {
		switch msg.Role {
		case "user":
			// User label with timestamp
			timeStr := msg.Timestamp.Format("15:04")
			content.WriteString(RenderUserLabel() + "  " + HelpDescStyle.Render(timeStr))
			content.WriteString("\n")
			content.WriteString(UserMessageStyle.Width(m.width - 6).Render(msg.Content))
			content.WriteString("\n\n")

		case "assistant":
			modelLabel := m.currentModel
			if msg.Model != "" {
				modelLabel = msg.Model
			}
			// Assistant label with timestamp and duration
			timeStr := msg.Timestamp.Format("15:04")
			durationStr := ""
			if msg.Duration > 0 {
				durationStr = fmt.Sprintf(" (%.1fs)", msg.Duration.Seconds())
			}
			content.WriteString(RenderAssistantLabel(modelLabel) + "  " + HelpDescStyle.Render(timeStr+durationStr))
			content.WriteString("\n")
			content.WriteString(AssistantMessageStyle.Width(m.width - 6).Render(msg.Content))
			content.WriteString("\n\n")

		case "system":
			content.WriteString(SystemMessageStyle.Render(msg.Content))
			content.WriteString("\n\n")
		}
	}

	// Show streaming content
	if m.streaming && m.streamBuffer.Len() > 0 {
		timeStr := time.Now().Format("15:04")
		content.WriteString(RenderAssistantLabel(m.currentModel) + "  " + HelpDescStyle.Render(timeStr))
		content.WriteString("\n")
		content.WriteString(AssistantMessageStyle.Width(m.width - 6).Render(m.streamBuffer.String()))
		content.WriteString(ThinkingStyle.Render("..."))
		content.WriteString("\n")
	}

	m.viewport.SetContent(content.String())
}

// sendMessageWithStreaming initiates streaming and returns waitForNextChunk command
func (m *Model) sendMessageWithStreaming(input string) tea.Cmd {
	// Build message history
	var messages []ollama.ChatMessage
	for _, msg := range m.messages {
		if msg.Role == "user" || msg.Role == "assistant" {
			messages = append(messages, ollama.ChatMessage{
				Role:    msg.Role,
				Content: msg.Content,
			})
		}
	}

	// Start streaming via Ollama
	ctx := context.Background() // No timeout for streaming - we handle it via chunks
	respCh, errCh := m.ollamaClient.ChatStream(ctx, &ollama.ChatRequest{
		Model:    m.currentModel,
		Messages: messages,
	})

	// Store channels in model for later use
	m.streamRespCh = respCh
	m.streamErrCh = errCh
	m.streamStart = time.Now()

	// Return command to wait for first chunk
	return m.waitForNextChunk()
}

// waitForNextChunk returns a command that waits for the next streaming chunk
func (m *Model) waitForNextChunk() tea.Cmd {
	respCh := m.streamRespCh
	errCh := m.streamErrCh
	startTime := m.streamStart

	return func() tea.Msg {
		select {
		case resp, ok := <-respCh:
			if !ok {
				// Channel closed, streaming done
				return streamChunkMsg{done: true, duration: time.Since(startTime)}
			}
			if resp.Done {
				return streamChunkMsg{delta: resp.Message.Content, done: true, duration: time.Since(startTime)}
			}
			return streamChunkMsg{delta: resp.Message.Content, done: false}

		case err, ok := <-errCh:
			if ok && err != nil {
				return streamChunkMsg{err: err, done: true}
			}
			// Error channel closed without error
			return streamChunkMsg{done: true, duration: time.Since(startTime)}
		}
	}
}

// sendMessage sends a message via gRPC or Ollama (non-streaming fallback)
func (m *Model) sendMessage(input string) tea.Cmd {
	return func() tea.Msg {
		startTime := time.Now()

		ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
		defer cancel()

		// Build message history
		var messages []ollama.ChatMessage
		for _, msg := range m.messages {
			if msg.Role == "user" || msg.Role == "assistant" {
				messages = append(messages, ollama.ChatMessage{
					Role:    msg.Role,
					Content: msg.Content,
				})
			}
		}

		// Try gRPC first
		if m.useGRPC {
			conn, err := m.dialGRPC()
			if err == nil {
				defer conn.Close()
				client := turingpb.NewTuringServiceClient(conn)

				// Build gRPC messages
				var grpcMessages []*turingpb.Message
				for _, msg := range messages {
					grpcMessages = append(grpcMessages, &turingpb.Message{
						Role:    msg.Role,
						Content: msg.Content,
					})
				}

				// Use streaming for better UX
				stream, err := client.StreamChat(ctx, &turingpb.ChatRequest{
					Messages: grpcMessages,
					Model:    m.currentModel,
				})
				if err == nil {
					var fullContent strings.Builder
					for {
						chunk, err := stream.Recv()
						if err == io.EOF {
							break
						}
						if err != nil {
							return chatResponseMsg{err: err}
						}
						fullContent.WriteString(chunk.Delta)
						if chunk.Done {
							break
						}
					}
					return chatResponseMsg{content: fullContent.String(), model: m.currentModel, duration: time.Since(startTime)}
				}
			}
		}

		// Fallback to Ollama directly
		resp, err := m.ollamaClient.Chat(ctx, &ollama.ChatRequest{
			Model:    m.currentModel,
			Messages: messages,
		})

		if err != nil {
			return chatResponseMsg{err: err}
		}

		return chatResponseMsg{content: resp.Message.Content, model: m.currentModel, duration: time.Since(startTime)}
	}
}

// dialGRPC creates a gRPC connection
func (m *Model) dialGRPC() (*grpc.ClientConn, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	return grpc.DialContext(ctx, m.turingAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
}

// checkTuringStatus checks if Turing service is available
func (m Model) checkTuringStatus() tea.Msg {
	conn, err := m.dialGRPC()
	if err != nil {
		return serviceStatusMsg{turingOnline: false, err: err}
	}
	conn.Close()
	return serviceStatusMsg{turingOnline: true}
}

// loadModels loads available models from Ollama
func (m Model) loadModels() tea.Msg {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := m.ollamaClient.ListModels(ctx)
	if err != nil {
		return modelsLoadedMsg{err: err}
	}

	var modelInfos []ModelInfo
	for _, model := range resp.Models {
		size := "unknown"
		if model.Size > 0 {
			size = fmt.Sprintf("%.1f GB", float64(model.Size)/(1024*1024*1024))
		}
		modelInfos = append(modelInfos, ModelInfo{
			Name:        model.Name,
			Size:        size,
			Description: model.Name,
			Available:   true,
		})
	}

	return modelsLoadedMsg{models: modelInfos}
}

// loadAgents loads available agents from Leibniz service
func (m Model) loadAgents() tea.Msg {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Connect to Leibniz
	conn, err := grpc.DialContext(ctx, m.leibnizAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return agentListMsg{err: err}
	}
	defer conn.Close()

	client := leibnizpb.NewLeibnizServiceClient(conn)

	// List agents
	resp, err := client.ListAgents(ctx, &common.Empty{})
	if err != nil {
		return agentListMsg{err: err}
	}

	var agents []AgentInfo
	for _, agent := range resp.Agents {
		var tools []string
		for _, t := range agent.Tools {
			tools = append(tools, t)
		}
		agents = append(agents, AgentInfo{
			ID:          agent.Id,
			Name:        agent.Name,
			Description: agent.Description,
			Tools:       tools,
		})
	}

	return agentListMsg{agents: agents}
}

// checkAristotelesStatus checks if Aristoteles service is available
func (m Model) checkAristotelesStatus() tea.Msg {
	conn, err := m.dialAristotelesGRPC()
	if err != nil {
		return aristotelesStatusMsg{online: false, err: err}
	}
	conn.Close()
	return aristotelesStatusMsg{online: true}
}

// dialAristotelesGRPC creates a gRPC connection to Aristoteles
func (m *Model) dialAristotelesGRPC() (*grpc.ClientConn, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	return grpc.DialContext(ctx, m.aristotelesAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
}

// sendMessageViaAristoteles sends a message through the Aristoteles pipeline
func (m *Model) sendMessageViaAristoteles(input string) tea.Cmd {
	return func() tea.Msg {
		startTime := time.Now()

		ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
		defer cancel()

		conn, err := m.dialAristotelesGRPC()
		if err != nil {
			return aristotelesPipelineMsg{err: fmt.Errorf("Aristoteles-Verbindungsfehler: %w", err)}
		}
		defer conn.Close()

		client := aristotelepb.NewAristotelesServiceClient(conn)

		// Build request
		req := &aristotelepb.ProcessRequest{
			RequestId: fmt.Sprintf("chat-%d", time.Now().UnixNano()),
			Prompt:    input,
			Options: &aristotelepb.ProcessOptions{
				ForceModel: m.currentModel,
				ForceAgent: m.selectedAgentID, // "" = auto-selection, sonst explizite Agent-ID
			},
		}

		// Call Aristoteles
		resp, err := client.Process(ctx, req)
		if err != nil {
			return aristotelesPipelineMsg{err: fmt.Errorf("Pipeline-Fehler: %w", err)}
		}

		// Extract enrichments
		var enrichments []string
		for _, e := range resp.Enrichments {
			enrichments = append(enrichments, e.Type.String())
		}

		// Extract intent and strategy info
		intentType := "UNKNOWN"
		if resp.Intent != nil {
			intentType = resp.Intent.Primary.String()
		}
		strategyName := "direct"
		if resp.Strategy != nil {
			strategyName = resp.Strategy.Name
		}
		qualityScore := float32(0.0)
		if resp.Metrics != nil {
			qualityScore = resp.Metrics.QualityScore
		}

		// Extract agent pipeline info
		agentID := ""
		agentName := ""
		agentConfidence := 0.0
		targetService := "Turing"

		if resp.Route != nil {
			targetService = resp.Route.Service.String()
			agentID = resp.Route.AgentId
		}

		// Extract agent match info from metadata (set by router)
		if resp.Metadata != nil {
			if name, ok := resp.Metadata["matched_agent_name"]; ok {
				agentName = name
			}
			if conf, ok := resp.Metadata["agent_confidence"]; ok {
				// Parse confidence string to float
				if _, err := fmt.Sscanf(conf, "%f", &agentConfidence); err != nil {
					agentConfidence = 0.0
				}
			}
		}

		return aristotelesPipelineMsg{
			content:         resp.Response,
			intentType:      intentType,
			strategyName:    strategyName,
			qualityScore:    qualityScore,
			duration:        time.Since(startTime),
			enrichments:     enrichments,
			agentID:         agentID,
			agentName:       agentName,
			agentConfidence: agentConfidence,
			targetService:   targetService,
		}
	}
}

// scheduleStepUpdates returns a command that schedules step updates
// This simulates the pipeline steps since actual server steps aren't streamed
func (m Model) scheduleStepUpdates() tea.Cmd {
	return tea.Batch(
		tea.Tick(1500*time.Millisecond, func(t time.Time) tea.Msg {
			return stepUpdateMsg{step: StepSearching}
		}),
		tea.Tick(3500*time.Millisecond, func(t time.Time) tea.Msg {
			return stepUpdateMsg{step: StepFetching}
		}),
		tea.Tick(6000*time.Millisecond, func(t time.Time) tea.Msg {
			return stepUpdateMsg{step: StepProcessing}
		}),
		tea.Tick(9000*time.Millisecond, func(t time.Time) tea.Msg {
			return stepUpdateMsg{step: StepGenerating}
		}),
	)
}

// Run starts the ChatClient TUI
func Run(cfg Config) error {
	p := tea.NewProgram(New(cfg), tea.WithAltScreen())
	_, err := p.Run()
	return err
}
