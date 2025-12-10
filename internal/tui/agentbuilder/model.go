// ============================================================================
// meinDENKWERK (mDW) - Lokale KI-Plattform
// ============================================================================
//
// Package:     agentbuilder
// Description: Main Bubble Tea model for Agent Builder TUI
// Author:      Mike Stoffels with Claude
// Created:     2025-12-07
// License:     MIT
// ============================================================================

package agentbuilder

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/msto63/mDW/api/gen/common"
	pb "github.com/msto63/mDW/api/gen/leibniz"
)

// ViewMode represents the current view
type ViewMode int

const (
	ViewList ViewMode = iota
	ViewEditor
	ViewToolPicker
	ViewTest
)

// EditorField represents the currently focused field in editor
type EditorField int

const (
	FieldName EditorField = iota
	FieldDescription
	FieldModel
	FieldTemperature
	FieldMaxIterations
	FieldTimeout
	FieldSystemPrompt
	FieldTools
	FieldSave
	FieldCancel
)

// Config holds the configuration for Agent Builder
type Config struct {
	LeibnizAddr string
	TuringAddr  string
}

// Model is the main Bubble Tea model
type Model struct {
	// Dimensions
	width, height int
	ready         bool

	// State
	viewMode      ViewMode
	loading       bool
	leibnizOnline bool
	turingOnline  bool
	err           error

	// Data
	agents        []AgentData
	tools         []ToolData
	models        []string
	selectedAgent int
	editingAgent  *AgentData
	isNewAgent    bool

	// Components
	spinner   spinner.Model
	agentList list.Model
	viewport  viewport.Model

	// Editor fields
	editorField    EditorField
	nameInput      textinput.Model
	descInput      textinput.Model
	systemPrompt   textarea.Model
	selectedModel  int
	temperature    float32
	maxIterations  int
	timeout        int
	selectedTools  map[string]bool

	// Test view
	testInput    textinput.Model
	testMessages []string
	testRunning  bool
	testStats    ExecutionData

	// Config
	leibnizAddr string
	turingAddr  string
}

// agentItem implements list.Item for agent list
type agentItem struct {
	agent AgentData
}

func (i agentItem) Title() string       { return i.agent.Name }
func (i agentItem) Description() string { return i.agent.Description }
func (i agentItem) FilterValue() string { return i.agent.Name }

// NewModel creates a new Agent Builder model
func NewModel(cfg Config) Model {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(ColorPrimary)

	// Initialize text inputs
	nameInput := textinput.New()
	nameInput.Placeholder = "Agent Name"
	nameInput.CharLimit = 64
	nameInput.Width = 40

	descInput := textinput.New()
	descInput.Placeholder = "Beschreibung"
	descInput.CharLimit = 200
	descInput.Width = 40

	// Initialize textarea for system prompt
	systemPrompt := textarea.New()
	systemPrompt.Placeholder = "System Prompt..."
	systemPrompt.CharLimit = 10000
	systemPrompt.SetWidth(60)
	systemPrompt.SetHeight(10)

	// Initialize test input
	testInput := textinput.New()
	testInput.Placeholder = "Test-Nachricht eingeben..."
	testInput.Width = 60

	// Initialize agent list
	delegate := list.NewDefaultDelegate()
	delegate.Styles.SelectedTitle = SelectedItemStyle
	delegate.Styles.NormalTitle = ListItemStyle

	agentList := list.New([]list.Item{}, delegate, 0, 0)
	agentList.Title = "Agenten"
	agentList.SetShowHelp(false)
	agentList.SetFilteringEnabled(true)

	// Default models
	defaultModels := []string{
		"llama3.2:3b",
		"llama3.2:1b",
		"qwen2.5:7b",
		"mistral:7b",
	}

	return Model{
		spinner:       s,
		agentList:     agentList,
		nameInput:     nameInput,
		descInput:     descInput,
		systemPrompt:  systemPrompt,
		testInput:     testInput,
		viewMode:      ViewList,
		models:        defaultModels,
		temperature:   0.7,
		maxIterations: 10,
		timeout:       120,
		selectedTools: make(map[string]bool),
		leibnizAddr:   cfg.LeibnizAddr,
		turingAddr:    cfg.TuringAddr,
	}
}

// Init implements tea.Model
func (m Model) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.Tick,
		checkServiceStatus(m.leibnizAddr),
		loadAgents(m.leibnizAddr),
		loadTools(m.leibnizAddr),
	)
}

// Update implements tea.Model
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Global keys
		switch msg.String() {
		case "ctrl+c", "q":
			if m.viewMode == ViewList {
				return m, tea.Quit
			}
			// Go back to list from other views
			m.viewMode = ViewList
			return m, nil

		case "esc":
			if m.viewMode != ViewList {
				m.viewMode = ViewList
				return m, nil
			}
		}

		// View-specific handling
		switch m.viewMode {
		case ViewList:
			return m.updateListView(msg)
		case ViewEditor:
			return m.updateEditorView(msg)
		case ViewToolPicker:
			return m.updateToolPickerView(msg)
		case ViewTest:
			return m.updateTestView(msg)
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.ready = true

		// Update component sizes
		m.agentList.SetSize(30, m.height-6)
		m.viewport = viewport.New(m.width-35, m.height-6)
		m.systemPrompt.SetWidth(m.width - 45)
		m.systemPrompt.SetHeight(8)

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		cmds = append(cmds, cmd)

	case serviceStatusMsg:
		m.leibnizOnline = msg.leibnizOnline
		m.turingOnline = msg.turingOnline
		if msg.err != nil {
			m.err = msg.err
		}

	case agentsLoadedMsg:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err
		} else {
			m.agents = msg.agents
			m.updateAgentList()
		}

	case toolsLoadedMsg:
		if msg.err != nil {
			m.err = msg.err
		} else {
			m.tools = msg.tools
		}

	case agentCreatedMsg:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err
		} else {
			m.agents = append(m.agents, msg.agent)
			m.updateAgentList()
			m.viewMode = ViewList
		}

	case agentUpdatedMsg:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err
		} else {
			// Update agent in list
			for i, a := range m.agents {
				if a.ID == msg.agent.ID {
					m.agents[i] = msg.agent
					break
				}
			}
			m.updateAgentList()
			m.viewMode = ViewList
		}

	case agentDeletedMsg:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err
		} else {
			// Remove agent from list
			for i, a := range m.agents {
				if a.ID == msg.id {
					m.agents = append(m.agents[:i], m.agents[i+1:]...)
					break
				}
			}
			m.updateAgentList()
		}

	case testChunkMsg:
		m.testMessages = append(m.testMessages, formatTestChunk(msg))
		m.viewport.SetContent(strings.Join(m.testMessages, "\n"))
		m.viewport.GotoBottom()

	case testCompletedMsg:
		m.testRunning = false
		if msg.err != nil {
			m.testMessages = append(m.testMessages, TestErrorStyle.Render("Fehler: "+msg.err.Error()))
		} else {
			m.testStats = msg.execution
			m.testMessages = append(m.testMessages,
				fmt.Sprintf("\n%s", TestStatsStyle.Render(
					fmt.Sprintf("Iterations: %d | Tokens: %d | Duration: %v",
						msg.execution.Iterations,
						msg.execution.TotalTokens,
						msg.execution.Duration))))
		}
		m.viewport.SetContent(strings.Join(m.testMessages, "\n"))
		m.viewport.GotoBottom()

	case tickMsg:
		cmds = append(cmds, tea.Tick(5*time.Second, func(t time.Time) tea.Msg {
			return tickMsg(t)
		}))
	}

	return m, tea.Batch(cmds...)
}

// updateListView handles list view updates
func (m Model) updateListView(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "n":
		// New agent
		m.isNewAgent = true
		m.editingAgent = &AgentData{
			Temperature:    0.7,
			MaxIterations:  10,
			TimeoutSeconds: 120,
			Model:          "llama3.2:3b",
		}
		m.loadAgentToEditor()
		m.viewMode = ViewEditor
		m.editorField = FieldName
		m.nameInput.Focus()
		return m, nil

	case "enter", "e":
		// Edit selected agent
		if len(m.agents) > 0 && m.selectedAgent < len(m.agents) {
			m.isNewAgent = false
			agent := m.agents[m.selectedAgent]
			m.editingAgent = &agent
			m.loadAgentToEditor()
			m.viewMode = ViewEditor
			m.editorField = FieldName
			m.nameInput.Focus()
		}
		return m, nil

	case "d":
		// Delete selected agent
		if len(m.agents) > 0 && m.selectedAgent < len(m.agents) {
			agent := m.agents[m.selectedAgent]
			return m, deleteAgent(m.leibnizAddr, agent.ID)
		}
		return m, nil

	case "c":
		// Clone selected agent
		if len(m.agents) > 0 && m.selectedAgent < len(m.agents) {
			m.isNewAgent = true
			agent := m.agents[m.selectedAgent]
			clone := agent
			clone.ID = ""
			clone.Name = agent.Name + " (Kopie)"
			m.editingAgent = &clone
			m.loadAgentToEditor()
			m.viewMode = ViewEditor
			m.editorField = FieldName
			m.nameInput.Focus()
		}
		return m, nil

	case "t":
		// Test selected agent
		if len(m.agents) > 0 && m.selectedAgent < len(m.agents) {
			m.editingAgent = &m.agents[m.selectedAgent]
			m.testMessages = []string{
				HeaderStyle.Render(fmt.Sprintf("Test: %s", m.editingAgent.Name)),
				"",
			}
			m.viewport.SetContent(strings.Join(m.testMessages, "\n"))
			m.viewMode = ViewTest
			m.testInput.Focus()
		}
		return m, nil

	case "r":
		// Refresh
		m.loading = true
		return m, loadAgents(m.leibnizAddr)

	case "1", "2", "3":
		// Load template
		idx := int(msg.String()[0] - '1')
		if idx < len(DefaultAgentTemplates) {
			m.isNewAgent = true
			template := DefaultAgentTemplates[idx]
			m.editingAgent = &template
			m.editingAgent.ID = ""
			m.loadAgentToEditor()
			m.viewMode = ViewEditor
			m.editorField = FieldName
			m.nameInput.Focus()
		}
		return m, nil
	}

	// Update list navigation
	var cmd tea.Cmd
	m.agentList, cmd = m.agentList.Update(msg)
	if m.agentList.Index() != m.selectedAgent {
		m.selectedAgent = m.agentList.Index()
	}
	return m, cmd
}

// updateEditorView handles editor view updates
func (m Model) updateEditorView(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg.String() {
	case "tab", "down":
		m.editorField = (m.editorField + 1) % (FieldCancel + 1)
		m.updateEditorFocus()
		return m, nil

	case "shift+tab", "up":
		if m.editorField == 0 {
			m.editorField = FieldCancel
		} else {
			m.editorField--
		}
		m.updateEditorFocus()
		return m, nil

	case "ctrl+s", "enter":
		if m.editorField == FieldSave || msg.String() == "ctrl+s" {
			return m, m.saveAgent()
		}
		if m.editorField == FieldCancel {
			m.viewMode = ViewList
			return m, nil
		}
		if m.editorField == FieldTools {
			m.viewMode = ViewToolPicker
			return m, nil
		}

	case "left":
		if m.editorField == FieldTemperature {
			m.temperature = max(0, m.temperature-0.1)
		} else if m.editorField == FieldMaxIterations {
			m.maxIterations = max(1, m.maxIterations-1)
		} else if m.editorField == FieldTimeout {
			m.timeout = max(10, m.timeout-10)
		} else if m.editorField == FieldModel && m.selectedModel > 0 {
			m.selectedModel--
		}
		return m, nil

	case "right":
		if m.editorField == FieldTemperature {
			m.temperature = min(2.0, m.temperature+0.1)
		} else if m.editorField == FieldMaxIterations {
			m.maxIterations = min(50, m.maxIterations+1)
		} else if m.editorField == FieldTimeout {
			m.timeout = min(600, m.timeout+10)
		} else if m.editorField == FieldModel && m.selectedModel < len(m.models)-1 {
			m.selectedModel++
		}
		return m, nil
	}

	// Update focused input
	switch m.editorField {
	case FieldName:
		m.nameInput, cmd = m.nameInput.Update(msg)
	case FieldDescription:
		m.descInput, cmd = m.descInput.Update(msg)
	case FieldSystemPrompt:
		m.systemPrompt, cmd = m.systemPrompt.Update(msg)
	}

	return m, cmd
}

// updateToolPickerView handles tool picker view updates
func (m Model) updateToolPickerView(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter", "esc":
		m.viewMode = ViewEditor
		return m, nil

	case "space":
		// Toggle selected tool
		if m.selectedAgent < len(m.tools) {
			tool := m.tools[m.selectedAgent]
			m.selectedTools[tool.Name] = !m.selectedTools[tool.Name]
		}
		return m, nil

	case "a":
		// Select all
		for _, t := range m.tools {
			m.selectedTools[t.Name] = true
		}
		return m, nil

	case "n":
		// Select none
		m.selectedTools = make(map[string]bool)
		return m, nil

	case "j", "down":
		if m.selectedAgent < len(m.tools)-1 {
			m.selectedAgent++
		}
		return m, nil

	case "k", "up":
		if m.selectedAgent > 0 {
			m.selectedAgent--
		}
		return m, nil
	}

	return m, nil
}

// updateTestView handles test view updates
func (m Model) updateTestView(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg.String() {
	case "enter":
		if !m.testRunning && m.testInput.Value() != "" {
			message := m.testInput.Value()
			m.testInput.SetValue("")
			m.testMessages = append(m.testMessages,
				TestUserStyle.Render("USER: ")+message,
				"")
			m.viewport.SetContent(strings.Join(m.testMessages, "\n"))
			m.viewport.GotoBottom()
			m.testRunning = true
			return m, executeTest(m.leibnizAddr, m.editingAgent.ID, message)
		}

	case "ctrl+r":
		// Reset conversation
		m.testMessages = []string{
			HeaderStyle.Render(fmt.Sprintf("Test: %s", m.editingAgent.Name)),
			"",
		}
		m.viewport.SetContent(strings.Join(m.testMessages, "\n"))
		return m, nil

	case "ctrl+x":
		// Cancel test
		m.testRunning = false
		m.testMessages = append(m.testMessages, TestErrorStyle.Render("Test abgebrochen"))
		m.viewport.SetContent(strings.Join(m.testMessages, "\n"))
		return m, nil
	}

	m.testInput, cmd = m.testInput.Update(msg)
	return m, cmd
}

// View implements tea.Model
func (m Model) View() string {
	if !m.ready {
		return m.spinner.View() + " Initialisiere..."
	}

	var content string
	switch m.viewMode {
	case ViewList:
		content = m.viewList()
	case ViewEditor:
		content = m.viewEditor()
	case ViewToolPicker:
		content = m.viewToolPicker()
	case ViewTest:
		content = m.viewTest()
	}

	return content
}

// viewList renders the agent list view
func (m Model) viewList() string {
	var b strings.Builder

	// Header
	header := TitlePanelStyle.Render(
		LogoStyle.Render(Logo) + "  " +
			SubHeaderStyle.Render("[Leibniz]"))
	b.WriteString(header)
	b.WriteString("\n")

	// Status bar
	status := "Status: "
	if m.leibnizOnline {
		status += StatusOnlineStyle.Render("Leibniz Online")
	} else {
		status += StatusOfflineStyle.Render("Leibniz Offline")
	}
	b.WriteString(StatusBarStyle.Render(status))
	b.WriteString("\n\n")

	// Main content - two columns
	leftPanel := FocusedPanelStyle.Width(30).Height(m.height - 10).Render(m.agentList.View())

	// Right panel - agent details or templates
	var rightContent string
	if len(m.agents) > 0 && m.selectedAgent < len(m.agents) {
		agent := m.agents[m.selectedAgent]
		rightContent = m.renderAgentDetails(agent)
	} else {
		rightContent = m.renderTemplates()
	}
	rightPanel := PanelStyle.Width(m.width - 35).Height(m.height - 10).Render(rightContent)

	b.WriteString(lipgloss.JoinHorizontal(lipgloss.Top, leftPanel, " ", rightPanel))
	b.WriteString("\n")

	// Help
	help := []string{
		RenderKeyHint("n", "Neu"),
		RenderKeyHint("e", "Edit"),
		RenderKeyHint("d", "Delete"),
		RenderKeyHint("c", "Clone"),
		RenderKeyHint("t", "Test"),
		RenderKeyHint("1-3", "Template"),
		RenderKeyHint("q", "Beenden"),
	}
	b.WriteString(HelpStyle.Render(strings.Join(help, "  ")))

	return b.String()
}

// viewEditor renders the editor view
func (m Model) viewEditor() string {
	var b strings.Builder

	title := "Neuer Agent"
	if !m.isNewAgent {
		title = "Agent bearbeiten"
	}
	b.WriteString(TitlePanelStyle.Render(LogoStyle.Render(title)))
	b.WriteString("\n\n")

	// Form fields
	fields := []struct {
		label   string
		content string
		field   EditorField
	}{
		{"Name:", m.nameInput.View(), FieldName},
		{"Beschreibung:", m.descInput.View(), FieldDescription},
		{"Modell:", m.renderModelSelector(), FieldModel},
		{"Temperature:", m.renderSlider(m.temperature, 0, 2, 20), FieldTemperature},
		{"Max Steps:", m.renderSlider(float32(m.maxIterations), 1, 50, 20), FieldMaxIterations},
		{"Timeout (s):", m.renderSlider(float32(m.timeout), 10, 600, 20), FieldTimeout},
	}

	for _, f := range fields {
		label := LabelStyle.Render(f.label)
		if m.editorField == f.field {
			label = LabelStyle.Copy().Foreground(ColorPrimary).Bold(true).Render(f.label)
		}
		b.WriteString(label + " " + f.content + "\n")
	}

	// System Prompt
	b.WriteString("\n")
	promptLabel := "System Prompt:"
	if m.editorField == FieldSystemPrompt {
		promptLabel = LabelStyle.Copy().Foreground(ColorPrimary).Bold(true).Render(promptLabel)
	} else {
		promptLabel = LabelStyle.Render(promptLabel)
	}
	b.WriteString(promptLabel + "\n")
	b.WriteString(m.systemPrompt.View())
	b.WriteString("\n\n")

	// Tools
	toolsLabel := "Tools:"
	if m.editorField == FieldTools {
		toolsLabel = LabelStyle.Copy().Foreground(ColorPrimary).Bold(true).Render(toolsLabel)
	} else {
		toolsLabel = LabelStyle.Render(toolsLabel)
	}
	selectedTools := m.getSelectedToolNames()
	if len(selectedTools) == 0 {
		b.WriteString(toolsLabel + " " + HelpDescStyle.Render("(keine ausgewaehlt - Enter zum Auswaehlen)") + "\n")
	} else {
		b.WriteString(toolsLabel + " " + strings.Join(selectedTools, ", ") + "\n")
	}

	// Buttons
	b.WriteString("\n")
	saveBtn := ButtonStyle.Render("[ Speichern ]")
	cancelBtn := ButtonStyle.Render("[ Abbrechen ]")
	if m.editorField == FieldSave {
		saveBtn = ButtonFocusedStyle.Render("[ Speichern ]")
	}
	if m.editorField == FieldCancel {
		cancelBtn = ButtonFocusedStyle.Render("[ Abbrechen ]")
	}
	b.WriteString(saveBtn + " " + cancelBtn + "\n")

	// Help
	b.WriteString("\n")
	b.WriteString(HelpStyle.Render(
		RenderKeyHint("Tab", "Naechstes") + "  " +
			RenderKeyHint("Ctrl+S", "Speichern") + "  " +
			RenderKeyHint("Esc", "Abbrechen")))

	return b.String()
}

// viewToolPicker renders the tool picker view
func (m Model) viewToolPicker() string {
	var b strings.Builder

	b.WriteString(TitlePanelStyle.Render(LogoStyle.Render("Tool-Auswahl")))
	b.WriteString("\n\n")

	// Group tools by source
	builtinTools := []ToolData{}
	mcpTools := []ToolData{}
	customTools := []ToolData{}

	for _, t := range m.tools {
		switch t.Source {
		case "builtin":
			builtinTools = append(builtinTools, t)
		case "mcp":
			mcpTools = append(mcpTools, t)
		default:
			customTools = append(customTools, t)
		}
	}

	idx := 0
	renderTools := func(title string, tools []ToolData) {
		if len(tools) == 0 {
			return
		}
		b.WriteString(ToolCategoryStyle.Render(title))
		b.WriteString("\n")
		for _, t := range tools {
			checkbox := RenderCheckbox(m.selectedTools[t.Name])
			style := ToolItemStyle
			if idx == m.selectedAgent {
				style = ToolSelectedStyle
			}
			b.WriteString(fmt.Sprintf("  %s %s - %s\n",
				checkbox,
				style.Render(t.Name),
				HelpDescStyle.Render(t.Description)))
			idx++
		}
	}

	renderTools("Builtin Tools", builtinTools)
	renderTools("MCP Tools", mcpTools)
	renderTools("Custom Tools", customTools)

	// If no tools available, show message
	if len(m.tools) == 0 {
		b.WriteString(HelpDescStyle.Render("Keine Tools verfuegbar"))
	}

	b.WriteString("\n")
	b.WriteString(HelpStyle.Render(
		RenderKeyHint("Space", "Toggle") + "  " +
			RenderKeyHint("a", "Alle") + "  " +
			RenderKeyHint("n", "Keine") + "  " +
			RenderKeyHint("Enter", "Fertig")))

	return b.String()
}

// viewTest renders the test view
func (m Model) viewTest() string {
	var b strings.Builder

	b.WriteString(TitlePanelStyle.Render(
		LogoStyle.Render("Agent Test") + "  " +
			SubHeaderStyle.Render(m.editingAgent.Name)))
	b.WriteString("\n\n")

	// Test conversation viewport
	b.WriteString(FocusedPanelStyle.Width(m.width - 4).Height(m.height - 12).Render(m.viewport.View()))
	b.WriteString("\n\n")

	// Input
	if m.testRunning {
		b.WriteString(m.spinner.View() + " Agent denkt...")
	} else {
		b.WriteString("> " + m.testInput.View())
	}
	b.WriteString("\n\n")

	// Help
	b.WriteString(HelpStyle.Render(
		RenderKeyHint("Enter", "Senden") + "  " +
			RenderKeyHint("Ctrl+R", "Reset") + "  " +
			RenderKeyHint("Ctrl+X", "Abbruch") + "  " +
			RenderKeyHint("Esc", "Zurueck")))

	return b.String()
}

// Helper methods

func (m *Model) updateAgentList() {
	items := make([]list.Item, len(m.agents))
	for i, a := range m.agents {
		items[i] = agentItem{agent: a}
	}
	m.agentList.SetItems(items)
}

func (m *Model) loadAgentToEditor() {
	if m.editingAgent == nil {
		return
	}
	m.nameInput.SetValue(m.editingAgent.Name)
	m.descInput.SetValue(m.editingAgent.Description)
	m.systemPrompt.SetValue(m.editingAgent.SystemPrompt)
	m.temperature = m.editingAgent.Temperature
	m.maxIterations = m.editingAgent.MaxIterations
	m.timeout = m.editingAgent.TimeoutSeconds

	// Find model index
	for i, model := range m.models {
		if model == m.editingAgent.Model {
			m.selectedModel = i
			break
		}
	}

	// Load tools
	m.selectedTools = make(map[string]bool)
	for _, t := range m.editingAgent.Tools {
		m.selectedTools[t] = true
	}
}

func (m *Model) updateEditorFocus() {
	m.nameInput.Blur()
	m.descInput.Blur()
	m.systemPrompt.Blur()

	switch m.editorField {
	case FieldName:
		m.nameInput.Focus()
	case FieldDescription:
		m.descInput.Focus()
	case FieldSystemPrompt:
		m.systemPrompt.Focus()
	}
}

func (m *Model) getSelectedToolNames() []string {
	var names []string
	for name, selected := range m.selectedTools {
		if selected {
			names = append(names, name)
		}
	}
	return names
}

func (m Model) renderAgentDetails(agent AgentData) string {
	var b strings.Builder

	b.WriteString(HeaderStyle.Render(agent.Name) + "\n\n")
	b.WriteString(LabelStyle.Render("Beschreibung: ") + agent.Description + "\n")
	b.WriteString(LabelStyle.Render("Modell: ") + agent.Model + "\n")
	b.WriteString(LabelStyle.Render("Temperature: ") + fmt.Sprintf("%.1f", agent.Temperature) + "\n")
	b.WriteString(LabelStyle.Render("Max Steps: ") + fmt.Sprintf("%d", agent.MaxIterations) + "\n")
	b.WriteString(LabelStyle.Render("Timeout: ") + fmt.Sprintf("%ds", agent.TimeoutSeconds) + "\n")
	b.WriteString(LabelStyle.Render("Tools: ") + strings.Join(agent.Tools, ", ") + "\n")
	b.WriteString("\n")
	b.WriteString(LabelStyle.Render("Erstellt: ") + agent.CreatedAt.Format("2006-01-02 15:04") + "\n")

	return b.String()
}

func (m Model) renderTemplates() string {
	var b strings.Builder

	b.WriteString(HeaderStyle.Render("Vorlagen") + "\n\n")
	for i, t := range DefaultAgentTemplates {
		b.WriteString(fmt.Sprintf("%s %s\n",
			HelpKeyStyle.Render(fmt.Sprintf("[%d]", i+1)),
			t.Name))
		b.WriteString("    " + HelpDescStyle.Render(t.Description) + "\n\n")
	}

	return b.String()
}

func (m Model) renderModelSelector() string {
	if m.selectedModel < len(m.models) {
		return fmt.Sprintf("< %s >", m.models[m.selectedModel])
	}
	return "< - >"
}

func (m Model) renderSlider(value, min, max float32, width int) string {
	ratio := (value - min) / (max - min)
	filled := int(ratio * float32(width))
	if filled < 0 {
		filled = 0
	}
	if filled > width {
		filled = width
	}

	track := strings.Repeat("=", filled) + "o" + strings.Repeat("-", width-filled)
	return SliderTrackStyle.Render("["+track+"]") + " " + SliderValueStyle.Render(fmt.Sprintf("%.1f", value))
}

func (m Model) saveAgent() tea.Cmd {
	agent := AgentData{
		ID:             m.editingAgent.ID,
		Name:           m.nameInput.Value(),
		Description:    m.descInput.Value(),
		SystemPrompt:   m.systemPrompt.Value(),
		Model:          m.models[m.selectedModel],
		Temperature:    m.temperature,
		MaxIterations:  m.maxIterations,
		TimeoutSeconds: m.timeout,
		Tools:          m.getSelectedToolNames(),
	}

	if m.isNewAgent {
		agent.ID = uuid.New().String()
		return createAgent(m.leibnizAddr, agent)
	}
	return updateAgent(m.leibnizAddr, agent)
}

func formatTestChunk(chunk testChunkMsg) string {
	switch chunk.chunkType {
	case "thinking":
		return TestThinkingStyle.Render("THINKING: " + chunk.content)
	case "tool_call":
		return TestToolCallStyle.Render("TOOL: " + chunk.content)
	case "tool_result":
		return TestToolResultStyle.Render("RESULT: " + chunk.content)
	case "response":
		return TestAgentStyle.Render("AGENT: ") + chunk.content
	case "final":
		return TestAgentStyle.Render("AGENT: ") + chunk.content
	default:
		return chunk.content
	}
}

// Commands

func checkServiceStatus(leibnizAddr string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		conn, err := grpc.DialContext(ctx, leibnizAddr,
			grpc.WithTransportCredentials(insecure.NewCredentials()),
			grpc.WithBlock())
		if err != nil {
			return serviceStatusMsg{leibnizOnline: false, err: err}
		}
		defer conn.Close()

		return serviceStatusMsg{leibnizOnline: true}
	}
}

func loadAgents(leibnizAddr string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		conn, err := grpc.DialContext(ctx, leibnizAddr,
			grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			return agentsLoadedMsg{err: err}
		}
		defer conn.Close()

		client := pb.NewLeibnizServiceClient(conn)
		resp, err := client.ListAgents(ctx, &common.Empty{})
		if err != nil {
			return agentsLoadedMsg{err: err}
		}

		agents := make([]AgentData, len(resp.Agents))
		for i, a := range resp.Agents {
			agents[i] = AgentData{
				ID:             a.Id,
				Name:           a.Name,
				Description:    a.Description,
				SystemPrompt:   a.SystemPrompt,
				Model:          a.Config.Model,
				Temperature:    a.Config.Temperature,
				MaxIterations:  int(a.Config.MaxIterations),
				TimeoutSeconds: int(a.Config.TimeoutSeconds),
				Tools:          a.Tools,
				CreatedAt:      time.Unix(a.CreatedAt, 0),
				UpdatedAt:      time.Unix(a.UpdatedAt, 0),
			}
		}

		return agentsLoadedMsg{agents: agents}
	}
}

func loadTools(leibnizAddr string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		conn, err := grpc.DialContext(ctx, leibnizAddr,
			grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			return toolsLoadedMsg{err: err}
		}
		defer conn.Close()

		client := pb.NewLeibnizServiceClient(conn)
		resp, err := client.ListTools(ctx, &common.Empty{})
		if err != nil {
			// Return default tools if service unavailable
			return toolsLoadedMsg{tools: []ToolData{
				{Name: "calculator", Description: "Mathematische Berechnungen", Source: "builtin"},
				{Name: "datetime", Description: "Datum und Uhrzeit", Source: "builtin"},
				{Name: "web_search", Description: "Web-Suche", Source: "builtin"},
			}}
		}

		tools := make([]ToolData, len(resp.Tools))
		for i, t := range resp.Tools {
			source := "builtin"
			switch t.Source {
			case pb.ToolSource_TOOL_SOURCE_MCP:
				source = "mcp"
			case pb.ToolSource_TOOL_SOURCE_CUSTOM:
				source = "custom"
			}
			tools[i] = ToolData{
				Name:                 t.Name,
				Description:          t.Description,
				Source:               source,
				Enabled:              t.Enabled,
				RequiresConfirmation: t.RequiresConfirmation,
			}
		}

		return toolsLoadedMsg{tools: tools}
	}
}

func createAgent(leibnizAddr string, agent AgentData) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		conn, err := grpc.DialContext(ctx, leibnizAddr,
			grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			return agentCreatedMsg{err: err}
		}
		defer conn.Close()

		client := pb.NewLeibnizServiceClient(conn)
		req := &pb.CreateAgentRequest{
			Name:         agent.Name,
			Description:  agent.Description,
			SystemPrompt: agent.SystemPrompt,
			Tools:        agent.Tools,
			Config: &pb.AgentConfig{
				Model:          agent.Model,
				Temperature:    agent.Temperature,
				MaxIterations:  int32(agent.MaxIterations),
				TimeoutSeconds: int32(agent.TimeoutSeconds),
			},
		}

		resp, err := client.CreateAgent(ctx, req)
		if err != nil {
			return agentCreatedMsg{err: err}
		}

		return agentCreatedMsg{agent: AgentData{
			ID:             resp.Id,
			Name:           resp.Name,
			Description:    resp.Description,
			SystemPrompt:   resp.SystemPrompt,
			Model:          resp.Config.Model,
			Temperature:    resp.Config.Temperature,
			MaxIterations:  int(resp.Config.MaxIterations),
			TimeoutSeconds: int(resp.Config.TimeoutSeconds),
			Tools:          resp.Tools,
			CreatedAt:      time.Unix(resp.CreatedAt, 0),
			UpdatedAt:      time.Unix(resp.UpdatedAt, 0),
		}}
	}
}

func updateAgent(leibnizAddr string, agent AgentData) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		conn, err := grpc.DialContext(ctx, leibnizAddr,
			grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			return agentUpdatedMsg{err: err}
		}
		defer conn.Close()

		client := pb.NewLeibnizServiceClient(conn)
		req := &pb.UpdateAgentRequest{
			Id:           agent.ID,
			Name:         agent.Name,
			Description:  agent.Description,
			SystemPrompt: agent.SystemPrompt,
			Tools:        agent.Tools,
			Config: &pb.AgentConfig{
				Model:          agent.Model,
				Temperature:    agent.Temperature,
				MaxIterations:  int32(agent.MaxIterations),
				TimeoutSeconds: int32(agent.TimeoutSeconds),
			},
		}

		resp, err := client.UpdateAgent(ctx, req)
		if err != nil {
			return agentUpdatedMsg{err: err}
		}

		return agentUpdatedMsg{agent: AgentData{
			ID:             resp.Id,
			Name:           resp.Name,
			Description:    resp.Description,
			SystemPrompt:   resp.SystemPrompt,
			Model:          resp.Config.Model,
			Temperature:    resp.Config.Temperature,
			MaxIterations:  int(resp.Config.MaxIterations),
			TimeoutSeconds: int(resp.Config.TimeoutSeconds),
			Tools:          resp.Tools,
			CreatedAt:      time.Unix(resp.CreatedAt, 0),
			UpdatedAt:      time.Unix(resp.UpdatedAt, 0),
		}}
	}
}

func deleteAgent(leibnizAddr string, id string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		conn, err := grpc.DialContext(ctx, leibnizAddr,
			grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			return agentDeletedMsg{err: err}
		}
		defer conn.Close()

		client := pb.NewLeibnizServiceClient(conn)
		_, err = client.DeleteAgent(ctx, &pb.DeleteAgentRequest{Id: id})
		if err != nil {
			return agentDeletedMsg{err: err}
		}

		return agentDeletedMsg{id: id}
	}
}

func executeTest(leibnizAddr, agentID, message string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
		defer cancel()

		conn, err := grpc.DialContext(ctx, leibnizAddr,
			grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			return testCompletedMsg{err: err}
		}
		defer conn.Close()

		client := pb.NewLeibnizServiceClient(conn)
		resp, err := client.Execute(ctx, &pb.ExecuteRequest{
			AgentId: agentID,
			Message: message,
		})
		if err != nil {
			return testCompletedMsg{err: err}
		}

		actions := make([]ActionData, len(resp.Actions))
		for i, a := range resp.Actions {
			actions[i] = ActionData{
				Tool:     a.Tool,
				Input:    a.Input,
				Output:   a.Output,
				Success:  a.Success,
				Duration: time.Duration(a.DurationMs) * time.Millisecond,
			}
		}

		return testCompletedMsg{execution: ExecutionData{
			ID:          resp.ExecutionId,
			Status:      "completed",
			Response:    resp.Response,
			Iterations:  int(resp.Iterations),
			Duration:    time.Duration(resp.DurationMs) * time.Millisecond,
			TotalTokens: int(resp.TotalTokens),
			Actions:     actions,
		}}
	}
}

// Run starts the Agent Builder TUI
func Run(cfg Config) error {
	p := tea.NewProgram(NewModel(cfg), tea.WithAltScreen())
	_, err := p.Run()
	return err
}

// Empty is the gRPC Empty message
type Empty struct{}
