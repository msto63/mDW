package tui

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
	commonpb "github.com/msto63/mDW/api/gen/common"
	hypatiapb "github.com/msto63/mDW/api/gen/hypatia"
	leibnizpb "github.com/msto63/mDW/api/gen/leibniz"
	russellpb "github.com/msto63/mDW/api/gen/russell"
	turingpb "github.com/msto63/mDW/api/gen/turing"
	"github.com/msto63/mDW/internal/turing/ollama"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// View represents different views in the TUI
type View int

const (
	ViewChat View = iota
	ViewSearch
	ViewAgent
	ViewStatus
)

// Service addresses (configurable via environment)
type ServiceAddresses struct {
	Turing  string
	Hypatia string
	Leibniz string
	Russell string
}

func defaultAddresses() ServiceAddresses {
	return ServiceAddresses{
		Turing:  "localhost:9200",
		Hypatia: "localhost:9220",
		Leibniz: "localhost:9140",
		Russell: "localhost:9100",
	}
}

// Message represents a chat message
type Message struct {
	Role    string
	Content string
}

// SearchResult for display
type SearchResult struct {
	Title   string
	Score   float32
	Content string
}

// ServiceStatus for display
type ServiceStatus struct {
	Name    string
	Status  string
	Healthy bool
}

// Model is the main TUI model
type Model struct {
	// State
	view    View
	width   int
	height  int
	ready   bool
	loading bool
	err     error

	// Components
	textarea textarea.Model
	viewport viewport.Model
	spinner  spinner.Model

	// Chat state
	messages     []Message
	currentModel string

	// Search state
	searchQuery   string
	searchResults []SearchResult

	// Agent state
	agentTask    string
	agentOutput  string
	agentRunning bool

	// Status state
	services []ServiceStatus

	// Ollama client (fallback)
	ollamaClient *ollama.Client

	// gRPC addresses
	addrs ServiceAddresses

	// Content buffer
	content string

	// Use gRPC (vs direct Ollama)
	useGRPC bool
}

// NewModel creates a new TUI model
func NewModel() Model {
	ta := textarea.New()
	ta.Placeholder = "Nachricht eingeben..."
	ta.Focus()
	ta.CharLimit = 4000
	ta.SetWidth(80)
	ta.SetHeight(3)
	ta.ShowLineNumbers = false
	ta.KeyMap.InsertNewline.SetEnabled(false)

	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = lipgloss.NewStyle().Foreground(colorPrimary)

	return Model{
		view:          ViewChat,
		textarea:      ta,
		spinner:       sp,
		messages:      []Message{},
		searchResults: []SearchResult{},
		services:      []ServiceStatus{},
		currentModel:  "mistral:7b",
		ollamaClient:  ollama.NewClient(ollama.DefaultConfig()),
		addrs:         defaultAddresses(),
		useGRPC:       true, // Try gRPC first
	}
}

// Init initializes the model
func (m Model) Init() tea.Cmd {
	return tea.Batch(
		textarea.Blink,
		m.spinner.Tick,
		m.checkServices(), // Initial status check
	)
}

// Update handles messages
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			return m, tea.Quit

		case "tab":
			// Switch views
			m.view = (m.view + 1) % 4
			m.textarea.Reset()
			m.updatePlaceholder()
			if m.view == ViewStatus {
				cmds = append(cmds, m.checkServices())
			}
			return m, tea.Batch(cmds...)

		case "enter":
			if !m.loading {
				input := strings.TrimSpace(m.textarea.Value())
				if input != "" {
					switch m.view {
					case ViewChat:
						m.messages = append(m.messages, Message{Role: "user", Content: input})
						m.textarea.Reset()
						m.loading = true
						m.updateContent()
						return m, m.sendChatMessage(input)
					case ViewSearch:
						m.searchQuery = input
						m.textarea.Reset()
						m.loading = true
						return m, m.performSearch(input)
					case ViewAgent:
						m.agentTask = input
						m.agentOutput = ""
						m.textarea.Reset()
						m.loading = true
						m.agentRunning = true
						return m, m.executeAgent(input)
					}
				}
			}

		case "ctrl+l":
			// Clear current view
			switch m.view {
			case ViewChat:
				m.messages = []Message{}
			case ViewSearch:
				m.searchResults = []SearchResult{}
				m.searchQuery = ""
			case ViewAgent:
				m.agentOutput = ""
				m.agentTask = ""
			}
			m.updateContent()
			return m, nil

		case "ctrl+r":
			// Refresh status
			if m.view == ViewStatus {
				return m, m.checkServices()
			}
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		if !m.ready {
			m.viewport = viewport.New(msg.Width, msg.Height-10)
			m.viewport.YPosition = 3
			m.ready = true
		} else {
			m.viewport.Width = msg.Width
			m.viewport.Height = msg.Height - 10
		}
		m.textarea.SetWidth(msg.Width - 4)
		m.updateContent()

	case chatResponseMsg:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err
			m.messages = append(m.messages, Message{Role: "system", Content: "Fehler: " + msg.err.Error()})
		} else {
			m.messages = append(m.messages, Message{Role: "assistant", Content: msg.content})
		}
		m.updateContent()

	case searchResponseMsg:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err
		} else {
			m.searchResults = msg.results
		}
		m.updateContent()

	case agentResponseMsg:
		m.loading = false
		m.agentRunning = false
		if msg.err != nil {
			m.err = msg.err
			m.agentOutput = "Fehler: " + msg.err.Error()
		} else {
			m.agentOutput = msg.output
		}
		m.updateContent()

	case statusResponseMsg:
		if msg.err == nil {
			m.services = msg.services
		}
		m.updateContent()

	case spinner.TickMsg:
		if m.loading {
			m.spinner, cmd = m.spinner.Update(msg)
			cmds = append(cmds, cmd)
		}
	}

	// Update components
	m.textarea, cmd = m.textarea.Update(msg)
	cmds = append(cmds, cmd)

	m.viewport, cmd = m.viewport.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

func (m *Model) updatePlaceholder() {
	switch m.view {
	case ViewChat:
		m.textarea.Placeholder = "Nachricht eingeben..."
	case ViewSearch:
		m.textarea.Placeholder = "Suchanfrage eingeben..."
	case ViewAgent:
		m.textarea.Placeholder = "Aufgabe für den Agenten eingeben..."
	case ViewStatus:
		m.textarea.Placeholder = "Ctrl+R zum Aktualisieren..."
	}
}

// View renders the UI
func (m Model) View() string {
	if !m.ready {
		return "Lade..."
	}

	var s strings.Builder

	// Header
	s.WriteString(m.renderHeader())
	s.WriteString("\n")

	// Main content
	switch m.view {
	case ViewChat:
		s.WriteString(m.renderChatView())
	case ViewSearch:
		s.WriteString(m.renderSearchView())
	case ViewAgent:
		s.WriteString(m.renderAgentView())
	case ViewStatus:
		s.WriteString(m.renderStatusView())
	}

	// Footer
	s.WriteString("\n")
	s.WriteString(m.renderFooter())

	return s.String()
}

func (m *Model) renderHeader() string {
	tabs := []string{"Chat", "Suche", "Agent", "Status"}
	var renderedTabs []string

	for i, tab := range tabs {
		if View(i) == m.view {
			renderedTabs = append(renderedTabs, ActiveTabStyle.Render(tab))
		} else {
			renderedTabs = append(renderedTabs, TabStyle.Render(tab))
		}
	}

	title := TitleStyle.Render("meinDENKWERK")
	tabLine := lipgloss.JoinHorizontal(lipgloss.Top, renderedTabs...)

	return lipgloss.JoinVertical(lipgloss.Left, title, tabLine)
}

func (m *Model) renderChatView() string {
	var s strings.Builder

	// Messages viewport
	s.WriteString(m.viewport.View())
	s.WriteString("\n")

	// Loading indicator
	if m.loading {
		s.WriteString(m.spinner.View())
		s.WriteString(" Generiere Antwort...\n")
	}

	// Input area
	s.WriteString(FocusedInputStyle.Render(m.textarea.View()))

	return s.String()
}

func (m *Model) renderSearchView() string {
	var s strings.Builder

	s.WriteString(SubtitleStyle.Render("RAG-Suche"))
	s.WriteString("\n\n")

	if m.loading {
		s.WriteString(m.spinner.View())
		s.WriteString(" Suche läuft...\n")
	} else if len(m.searchResults) > 0 {
		s.WriteString(fmt.Sprintf("Ergebnisse für: %s\n\n", m.searchQuery))
		for i, r := range m.searchResults {
			s.WriteString(fmt.Sprintf("%d. %s (Score: %.2f)\n", i+1, r.Title, r.Score))
			content := r.Content
			if len(content) > 100 {
				content = content[:100] + "..."
			}
			s.WriteString(fmt.Sprintf("   %s\n\n", content))
		}
	} else if m.searchQuery != "" {
		s.WriteString("Keine Ergebnisse gefunden.\n")
	} else {
		s.WriteString("Gib eine Suchanfrage ein und drücke Enter.\n\n")
		s.WriteString("Beispiele:\n")
		s.WriteString("  - 'Wie funktioniert das System?'\n")
		s.WriteString("  - 'API Dokumentation'\n")
	}

	s.WriteString("\n")
	s.WriteString(FocusedInputStyle.Render(m.textarea.View()))

	return BoxStyle.Render(s.String())
}

func (m *Model) renderAgentView() string {
	var s strings.Builder

	s.WriteString(SubtitleStyle.Render("AI Agent"))
	s.WriteString("\n\n")

	if m.agentRunning {
		s.WriteString(m.spinner.View())
		s.WriteString(fmt.Sprintf(" Bearbeite Aufgabe: %s\n", m.agentTask))
	} else if m.agentOutput != "" {
		s.WriteString(fmt.Sprintf("Aufgabe: %s\n\n", m.agentTask))
		s.WriteString("Ergebnis:\n")
		s.WriteString(m.agentOutput)
		s.WriteString("\n")
	} else {
		s.WriteString("Gib eine Aufgabe für den Agenten ein.\n\n")
		s.WriteString("Beispiele:\n")
		s.WriteString("  - 'Analysiere die Datei README.md'\n")
		s.WriteString("  - 'Berechne 25 * 17'\n")
		s.WriteString("  - 'Erstelle eine Zusammenfassung'\n")
	}

	s.WriteString("\n")
	s.WriteString(FocusedInputStyle.Render(m.textarea.View()))

	return BoxStyle.Render(s.String())
}

func (m *Model) renderStatusView() string {
	var s strings.Builder

	s.WriteString(SubtitleStyle.Render("Service Status"))
	s.WriteString("\n\n")

	if len(m.services) == 0 {
		s.WriteString("Lade Status...\n")
	} else {
		for _, svc := range m.services {
			icon := "[+]"
			style := StatusOKStyle
			if !svc.Healthy {
				icon = "[-]"
				style = StatusErrorStyle
			}
			s.WriteString(fmt.Sprintf("  %s %-25s %s\n",
				style.Render(icon),
				svc.Name,
				svc.Status))
		}
	}

	s.WriteString("\n")
	s.WriteString(HelpStyle.Render("Ctrl+R: Aktualisieren"))

	return BoxStyle.Render(s.String())
}

func (m *Model) renderFooter() string {
	help := "Tab: Wechseln • Ctrl+L: Leeren • Ctrl+C: Beenden"
	model := fmt.Sprintf("Modell: %s", m.currentModel)

	return StatusBarStyle.Width(m.width).Render(
		lipgloss.JoinHorizontal(
			lipgloss.Top,
			help,
			strings.Repeat(" ", max(0, m.width-len(help)-len(model)-4)),
			model,
		),
	)
}

func (m *Model) updateContent() {
	var content strings.Builder

	for _, msg := range m.messages {
		switch msg.Role {
		case "user":
			content.WriteString(UserMessageStyle.Render("Du: "))
			content.WriteString(msg.Content)
		case "assistant":
			content.WriteString(AssistantMessageStyle.Render("Assistent: "))
			content.WriteString(msg.Content)
		case "system":
			content.WriteString(SystemMessageStyle.Render("[System] "))
			content.WriteString(msg.Content)
		}
		content.WriteString("\n\n")
	}

	m.content = content.String()
	m.viewport.SetContent(m.content)
	m.viewport.GotoBottom()
}

// Message types for async operations
type chatResponseMsg struct {
	content string
	err     error
}

type searchResponseMsg struct {
	results []SearchResult
	err     error
}

type agentResponseMsg struct {
	output string
	err    error
}

type statusResponseMsg struct {
	services []ServiceStatus
	err      error
}

// gRPC dial helper
func dialGRPC(addr string) (*grpc.ClientConn, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	return grpc.DialContext(ctx, addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
}

// sendChatMessage sends a chat message via gRPC or Ollama
func (m *Model) sendChatMessage(input string) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()

		// Try gRPC first
		if m.useGRPC {
			conn, err := dialGRPC(m.addrs.Turing)
			if err == nil {
				defer conn.Close()
				client := turingpb.NewTuringServiceClient(conn)

				// Build messages
				var messages []*turingpb.Message
				for _, msg := range m.messages {
					messages = append(messages, &turingpb.Message{
						Role:    msg.Role,
						Content: msg.Content,
					})
				}

				grpcCtx, cancel := context.WithTimeout(ctx, 120*time.Second)
				defer cancel()

				// Use streaming for better UX
				stream, err := client.StreamChat(grpcCtx, &turingpb.ChatRequest{
					Messages: messages,
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
					return chatResponseMsg{content: fullContent.String()}
				}
			}
		}

		// Fallback to Ollama
		messages := make([]ollama.ChatMessage, len(m.messages))
		for i, msg := range m.messages {
			messages[i] = ollama.ChatMessage{
				Role:    msg.Role,
				Content: msg.Content,
			}
		}

		resp, err := m.ollamaClient.Chat(ctx, &ollama.ChatRequest{
			Model:    m.currentModel,
			Messages: messages,
		})

		if err != nil {
			return chatResponseMsg{err: err}
		}

		return chatResponseMsg{content: resp.Message.Content}
	}
}

// performSearch executes a search query via gRPC
func (m *Model) performSearch(query string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		conn, err := dialGRPC(m.addrs.Hypatia)
		if err != nil {
			return searchResponseMsg{err: fmt.Errorf("Hypatia nicht erreichbar: %v", err)}
		}
		defer conn.Close()

		client := hypatiapb.NewHypatiaServiceClient(conn)

		resp, err := client.Search(ctx, &hypatiapb.SearchRequest{
			Query: query,
			TopK:  10,
		})
		if err != nil {
			return searchResponseMsg{err: err}
		}

		var results []SearchResult
		for _, r := range resp.Results {
			title := ""
			if r.Metadata != nil {
				title = r.Metadata.Title
			}
			if title == "" {
				title = r.DocumentId
			}
			results = append(results, SearchResult{
				Title:   title,
				Score:   r.Score,
				Content: r.Content,
			})
		}

		return searchResponseMsg{results: results}
	}
}

// executeAgent runs an agent task via gRPC
func (m *Model) executeAgent(task string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
		defer cancel()

		conn, err := dialGRPC(m.addrs.Leibniz)
		if err != nil {
			return agentResponseMsg{err: fmt.Errorf("Leibniz nicht erreichbar: %v", err)}
		}
		defer conn.Close()

		client := leibnizpb.NewLeibnizServiceClient(conn)

		resp, err := client.Execute(ctx, &leibnizpb.ExecuteRequest{
			AgentId: "default",
			Message: task,
		})
		if err != nil {
			return agentResponseMsg{err: err}
		}

		return agentResponseMsg{output: resp.Response}
	}
}

// checkServices checks all service statuses via gRPC
func (m *Model) checkServices() tea.Cmd {
	return func() tea.Msg {
		services := []ServiceStatus{
			{Name: "Kant (API Gateway)", Status: "prüfe...", Healthy: false},
			{Name: "Russell (Discovery)", Status: "prüfe...", Healthy: false},
			{Name: "Turing (LLM)", Status: "prüfe...", Healthy: false},
			{Name: "Hypatia (RAG)", Status: "prüfe...", Healthy: false},
			{Name: "Leibniz (Agent)", Status: "prüfe...", Healthy: false},
			{Name: "Babbage (NLP)", Status: "prüfe...", Healthy: false},
			{Name: "Bayes (Logging)", Status: "prüfe...", Healthy: false},
		}

		// Check Russell for comprehensive status
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		conn, err := dialGRPC(m.addrs.Russell)
		if err != nil {
			// Russell not available, do direct checks
			return statusResponseMsg{services: m.checkServicesDirect()}
		}
		defer conn.Close()

		client := russellpb.NewRussellServiceClient(conn)

		resp, err := client.ListServices(ctx, &commonpb.Empty{})
		if err != nil {
			return statusResponseMsg{services: m.checkServicesDirect()}
		}

		// Update from Russell response
		for _, svc := range resp.Services {
			for i := range services {
				if strings.Contains(strings.ToLower(services[i].Name), strings.ToLower(svc.Name)) {
					services[i].Status = svc.Status.String()
					services[i].Healthy = svc.Status == russellpb.ServiceStatus_SERVICE_STATUS_HEALTHY
				}
			}
		}

		// Mark Russell as healthy since we connected
		for i := range services {
			if strings.Contains(services[i].Name, "Russell") {
				services[i].Status = "healthy"
				services[i].Healthy = true
			}
		}

		return statusResponseMsg{services: services}
	}
}

// checkServicesDirect checks services without Russell
func (m *Model) checkServicesDirect() []ServiceStatus {
	services := []ServiceStatus{
		{Name: "Kant (API Gateway) :8080", Status: "stopped", Healthy: false},
		{Name: "Russell (Discovery) :9100", Status: "stopped", Healthy: false},
		{Name: "Turing (LLM) :9200", Status: "stopped", Healthy: false},
		{Name: "Hypatia (RAG) :9220", Status: "stopped", Healthy: false},
		{Name: "Leibniz (Agent) :9140", Status: "stopped", Healthy: false},
		{Name: "Babbage (NLP) :9150", Status: "stopped", Healthy: false},
		{Name: "Bayes (Logging) :9120", Status: "stopped", Healthy: false},
	}

	checks := []struct {
		index int
		addr  string
	}{
		{1, "localhost:9100"}, // Russell
		{2, "localhost:9200"}, // Turing
		{3, "localhost:9220"}, // Hypatia
		{4, "localhost:9140"}, // Leibniz
		{5, "localhost:9150"}, // Babbage
		{6, "localhost:9120"}, // Bayes
	}

	for _, check := range checks {
		conn, err := dialGRPC(check.addr)
		if err == nil {
			conn.Close()
			services[check.index].Status = "running"
			services[check.index].Healthy = true
		}
	}

	return services
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
