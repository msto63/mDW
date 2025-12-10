// ============================================================================
// meinDENKWERK (mDW) - Lokale KI-Plattform
// ============================================================================
//
// Package:     controlcenter
// Description: Main Bubbletea model for mDW Control Center
// Author:      Mike Stoffels with Claude
// Created:     2025-12-06
// License:     MIT
// ============================================================================

package controlcenter

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ViewState represents the current view
type ViewState int

const (
	ViewStartup ViewState = iota
	ViewMain
	ViewHelp
)

// Model is the main Bubbletea model
type Model struct {
	// State
	viewState      ViewState
	selectedIndex  int
	width          int
	height         int
	startTime      time.Time
	lastRefresh    time.Time
	refreshTicker  *time.Ticker

	// Components
	depChecker     *DependencyChecker
	serviceManager *ServiceManager
	spinner        spinner.Model

	// Status
	startupComplete bool
	startupError    string
	statusMessage   string
	statusExpiry    time.Time
}

// Message types
type tickMsg time.Time
type refreshMsg time.Time
type startupCompleteMsg struct{}
type statusClearMsg struct{}

// New creates a new ControlCenter model
func New() Model {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = SpinnerStyle

	return Model{
		viewState:      ViewStartup,
		depChecker:     NewDependencyChecker(),
		serviceManager: NewServiceManager(),
		spinner:        s,
		startTime:      time.Now(),
	}
}

// Init initializes the model
func (m Model) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.Tick,
		m.checkDependencies,
		tea.EnterAltScreen,
	)
}

// checkDependencies runs dependency checks
func (m Model) checkDependencies() tea.Msg {
	m.depChecker.CheckAll()
	return startupCompleteMsg{}
}

// Update handles messages
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m.handleKeyPress(msg)

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case startupCompleteMsg:
		m.startupComplete = true
		if !m.depChecker.AllRequiredOK() {
			m.startupError = "Some required dependencies are missing"
		}
		// Transition to main view after a brief delay
		return m, tea.Tick(time.Second, func(t time.Time) tea.Msg {
			return tickMsg(t)
		})

	case tickMsg:
		if m.viewState == ViewStartup && m.startupComplete {
			m.viewState = ViewMain
			m.serviceManager.RefreshStatus()
			// Start refresh ticker
			return m, tea.Tick(3*time.Second, func(t time.Time) tea.Msg {
				return refreshMsg(t)
			})
		}
		return m, nil

	case refreshMsg:
		m.serviceManager.RefreshStatus()
		m.lastRefresh = time.Now()
		// Continue refreshing
		return m, tea.Tick(3*time.Second, func(t time.Time) tea.Msg {
			return refreshMsg(t)
		})

	case statusClearMsg:
		m.statusMessage = ""
		return m, nil
	}

	return m, nil
}

// handleKeyPress handles keyboard input
func (m Model) handleKeyPress(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch m.viewState {
	case ViewStartup:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "enter", " ":
			if m.startupComplete {
				m.viewState = ViewMain
				m.serviceManager.RefreshStatus()
				return m, tea.Tick(3*time.Second, func(t time.Time) tea.Msg {
					return refreshMsg(t)
				})
			}
		}

	case ViewMain:
		switch msg.String() {
		case "q", "ctrl+c":
			// Don't stop services when exiting ControlCenter
			// Services should keep running in the background
			return m, tea.Quit

		case "up", "k":
			if m.selectedIndex > 0 {
				m.selectedIndex--
			}

		case "down", "j":
			if m.selectedIndex < len(m.serviceManager.Services)-1 {
				m.selectedIndex++
			}

		case "a":
			// Start all services
			if err := m.serviceManager.StartAll(); err != nil {
				m.setStatus("Error: " + err.Error())
			} else {
				m.setStatus("Starting all services...")
			}

		case "s":
			// Stop all services
			if err := m.serviceManager.StopAll(); err != nil {
				m.setStatus("Error: " + err.Error())
			} else {
				m.setStatus("Stopping all services...")
			}

		case "enter", " ":
			// Toggle selected service
			svc := &m.serviceManager.Services[m.selectedIndex]
			if svc.Status == ServiceRunning {
				if err := m.serviceManager.StopService(m.selectedIndex); err != nil {
					m.setStatus("Error: " + err.Error())
				} else {
					m.setStatus(fmt.Sprintf("Stopping %s...", svc.Name))
				}
			} else {
				if err := m.serviceManager.StartService(m.selectedIndex); err != nil {
					m.setStatus("Error: " + err.Error())
				} else {
					m.setStatus(fmt.Sprintf("Starting %s...", svc.Name))
				}
			}

		case "r":
			// Refresh status
			m.serviceManager.RefreshStatus()
			m.lastRefresh = time.Now()
			m.setStatus("Refreshed service status")

		case "d":
			// Re-check dependencies
			m.depChecker.CheckAll()
			m.setStatus("Re-checked dependencies")

		case "?", "h":
			m.viewState = ViewHelp
		}

	case ViewHelp:
		switch msg.String() {
		case "q", "ctrl+c", "escape", "?", "h", "enter", " ":
			m.viewState = ViewMain
		}
	}

	return m, nil
}

// setStatus sets a temporary status message
func (m *Model) setStatus(msg string) {
	m.statusMessage = msg
	m.statusExpiry = time.Now().Add(3 * time.Second)
}

// View renders the UI
func (m Model) View() string {
	switch m.viewState {
	case ViewStartup:
		return m.renderStartup()
	case ViewHelp:
		return m.renderHelp()
	default:
		return m.renderMain()
	}
}

// renderStartup renders the startup/dependency check view
func (m Model) renderStartup() string {
	var b strings.Builder

	// Logo
	b.WriteString(LogoStyle.Render(Logo))
	b.WriteString("\n")

	// Title
	title := HeaderStyle.Render("mDW Control Center")
	subtitle := SubHeaderStyle.Render("System Dependency Check")
	b.WriteString(lipgloss.JoinHorizontal(lipgloss.Center, title))
	b.WriteString("\n")
	b.WriteString(lipgloss.JoinHorizontal(lipgloss.Center, subtitle))
	b.WriteString("\n\n")

	// Dependency list
	for _, dep := range m.depChecker.Dependencies {
		icon := GetDependencyIcon(dep.Status)

		var statusStyle lipgloss.Style
		switch dep.Status {
		case StatusOK:
			statusStyle = StatusOKStyle
		case StatusFailed:
			statusStyle = StatusFailStyle
		case StatusWarning:
			statusStyle = StatusStartingStyle
		case StatusChecking:
			statusStyle = StatusCheckingStyle
		default:
			statusStyle = StatusUnknownStyle
		}

		name := DependencyNameStyle.Render(dep.Name)
		status := statusStyle.Render(icon + " " + dep.Status.String())

		var detail string
		if dep.Version != "" {
			detail = VersionStyle.Render(" (" + dep.Version + ")")
		} else if dep.Message != "" {
			detail = VersionStyle.Render(" - " + dep.Message)
		}

		required := ""
		if dep.Required {
			required = HelpDescStyle.Render(" [required]")
		}

		b.WriteString(DependencyStyle.Render(name + status + detail + required))
		b.WriteString("\n")
	}

	b.WriteString("\n")

	// Status message
	if m.startupComplete {
		if m.depChecker.AllRequiredOK() {
			b.WriteString(StatusRunningStyle.Render(IconOK + " All required dependencies OK"))
			b.WriteString("\n\n")
			b.WriteString(HelpStyle.Render("Press ENTER to continue..."))
		} else {
			b.WriteString(StatusStoppedStyle.Render(IconError + " Some required dependencies are missing!"))
			b.WriteString("\n\n")
			b.WriteString(HelpStyle.Render("Press ENTER to continue anyway, or q to quit"))
		}
	} else {
		b.WriteString(m.spinner.View() + " Checking dependencies...")
	}

	return b.String()
}

// renderMain renders the main service view
func (m Model) renderMain() string {
	var b strings.Builder

	// Compact header
	header := TitlePanelStyle.Render(
		LogoStyle.Render(LogoCompact) + "  " +
			VersionStyle.Render("v0.1.0") + "  " +
			m.renderRunningCount(),
	)
	b.WriteString(header)
	b.WriteString("\n\n")

	// Services panel
	servicesContent := m.renderServiceList()
	servicesPanel := FocusedPanelStyle.Width(m.width - 4).Render(servicesContent)
	b.WriteString(servicesPanel)

	// Status bar
	if m.statusMessage != "" && time.Now().Before(m.statusExpiry) {
		b.WriteString("\n")
		b.WriteString(StatusStartingStyle.Render(IconArrow + " " + m.statusMessage))
	}

	// Quick dependency summary
	b.WriteString("\n\n")
	b.WriteString(m.renderDependencySummary())

	// Help bar
	b.WriteString("\n\n")
	b.WriteString(m.renderHelpBar())

	return b.String()
}

// renderServiceList renders the list of services
func (m Model) renderServiceList() string {
	var b strings.Builder

	russellRunning := m.serviceManager.IsRussellRunning()

	// Header row - show MANAGED column only when Russell is running
	headerName := ServiceNameStyle.Bold(true).Render("SERVICE")
	headerPort := ServicePortStyle.Render("PORT")
	headerStatus := lipgloss.NewStyle().Width(15).Render("STATUS")
	headerManaged := ""
	if russellRunning {
		headerManaged = VersionStyle.Width(10).Render("MANAGED")
	}
	headerVersion := VersionStyle.Width(10).Render("VERSION")
	headerUptime := VersionStyle.Width(12).Render("UPTIME")

	b.WriteString(HelpDescStyle.Render(headerName + headerPort + headerStatus + headerManaged + headerVersion + headerUptime))
	b.WriteString("\n")
	lineWidth := 75
	if russellRunning {
		lineWidth = 85
	}
	b.WriteString(HelpDescStyle.Render(strings.Repeat("─", lineWidth)))
	b.WriteString("\n")

	// Service rows
	for i, svc := range m.serviceManager.Services {
		isSelected := i == m.selectedIndex

		// Status styling
		var statusStyle lipgloss.Style
		switch svc.Status {
		case ServiceRunning:
			statusStyle = StatusRunningStyle
		case ServiceStopped:
			statusStyle = StatusStoppedStyle
		case ServiceStarting, ServiceStopping:
			statusStyle = StatusStartingStyle
		case ServiceError:
			statusStyle = StatusFailStyle
		default:
			statusStyle = StatusUnknownStyle
		}

		icon := svc.GetStatusIcon()
		name := ServiceNameStyle.Render(svc.Name)
		port := ServicePortStyle.Render(fmt.Sprintf(":%d", svc.GetPort()))
		status := statusStyle.Width(15).Render(icon + " " + svc.Status.String())

		// Managed column - only show when Russell is running
		managed := ""
		if russellRunning {
			if svc.ShortName == "russell" {
				managed = VersionStyle.Width(10).Render("—") // Russell itself
			} else if svc.Status == ServiceRunning && svc.Managed {
				managed = StatusOKStyle.Width(10).Render(IconOK) // Running and managed
			} else if svc.Status == ServiceRunning && !svc.Managed {
				managed = StatusStartingStyle.Width(10).Render("✗") // Running but not managed
			} else {
				managed = VersionStyle.Width(10).Render("—") // Stopped
			}
		}

		version := VersionStyle.Width(10).Render("v" + svc.Version)
		uptime := VersionStyle.Width(12).Render(m.formatUptime(svc.Uptime))

		row := name + port + status + managed + version + uptime

		if isSelected {
			row = ServiceSelectedStyle.Render(IconArrow + " " + row)
		} else {
			row = ServiceRowStyle.Render("  " + row)
		}

		b.WriteString(row)
		b.WriteString("\n")
	}

	return b.String()
}

// formatUptime formats a duration as a human-readable string
func (m Model) formatUptime(d time.Duration) string {
	if d == 0 {
		return "-"
	}

	hours := int(d.Hours())
	minutes := int(d.Minutes()) % 60
	seconds := int(d.Seconds()) % 60

	if hours > 0 {
		return fmt.Sprintf("%dh%dm", hours, minutes)
	} else if minutes > 0 {
		return fmt.Sprintf("%dm%ds", minutes, seconds)
	}
	return fmt.Sprintf("%ds", seconds)
}

// renderRunningCount renders the running service count
func (m Model) renderRunningCount() string {
	running := m.serviceManager.GetRunningCount()
	total := len(m.serviceManager.Services)

	var style lipgloss.Style
	if running == total {
		style = StatusRunningStyle
	} else if running > 0 {
		style = StatusStartingStyle
	} else {
		style = StatusStoppedStyle
	}

	return style.Render(fmt.Sprintf("%d/%d running", running, total))
}

// renderDependencySummary renders a compact dependency summary
func (m Model) renderDependencySummary() string {
	var items []string

	for _, dep := range m.depChecker.Dependencies {
		if !dep.Required {
			continue
		}

		var icon string
		var style lipgloss.Style
		switch dep.Status {
		case StatusOK:
			icon = IconOK
			style = StatusOKStyle
		case StatusFailed:
			icon = IconError
			style = StatusFailStyle
		default:
			icon = IconBullet
			style = StatusUnknownStyle
		}

		items = append(items, style.Render(icon+" "+dep.Name))
	}

	return HelpDescStyle.Render("Dependencies: ") + strings.Join(items, "  ")
}

// renderHelpBar renders the help shortcuts bar
func (m Model) renderHelpBar() string {
	items := []string{
		RenderKeyHint("a", "start all"),
		RenderKeyHint("s", "stop all"),
		RenderKeyHint("Enter", "toggle"),
		RenderKeyHint("r", "refresh"),
		RenderKeyHint("d", "check deps"),
		RenderKeyHint("?", "help"),
		RenderKeyHint("q", "quit"),
	}
	return strings.Join(items, "  ")
}

// renderHelp renders the help view
func (m Model) renderHelp() string {
	var b strings.Builder

	b.WriteString(HeaderStyle.Render("mDW Control Center - Help"))
	b.WriteString("\n\n")

	// Key bindings
	bindings := []struct {
		key  string
		desc string
	}{
		{"a", "Start all services"},
		{"s", "Stop all services"},
		{"Enter/Space", "Start/Stop selected service"},
		{"j/↓", "Move selection down"},
		{"k/↑", "Move selection up"},
		{"r", "Refresh service status"},
		{"d", "Re-check dependencies"},
		{"?/h", "Show/hide this help"},
		{"q/Ctrl+C", "Quit Control Center"},
	}

	for _, binding := range bindings {
		b.WriteString("  ")
		b.WriteString(HelpKeyStyle.Width(15).Render(binding.key))
		b.WriteString(HelpDescStyle.Render(binding.desc))
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(HeaderStyle.Render("Services Overview"))
	b.WriteString("\n\n")

	for _, svc := range m.serviceManager.Services {
		// Service name with version
		b.WriteString("  ")
		b.WriteString(ServiceNameStyle.Render(svc.Name))
		b.WriteString(" ")
		b.WriteString(VersionStyle.Render("v" + svc.Version))
		b.WriteString("\n")

		// Description
		b.WriteString("    ")
		b.WriteString(HelpDescStyle.Render(svc.Description))
		b.WriteString("\n")

		// Ports
		b.WriteString("    ")
		if svc.GRPCPort != 0 {
			b.WriteString(VersionStyle.Render(fmt.Sprintf("gRPC: %d", svc.GRPCPort)))
			b.WriteString("  ")
		}
		if svc.HTTPPort != 0 {
			b.WriteString(VersionStyle.Render(fmt.Sprintf("HTTP: %d", svc.HTTPPort)))
		}

		// Status details for running services
		if svc.Status == ServiceRunning {
			b.WriteString("  ")
			b.WriteString(StatusRunningStyle.Render(fmt.Sprintf("PID: %d", svc.PID)))
			if svc.Uptime > 0 {
				b.WriteString("  ")
				b.WriteString(StatusRunningStyle.Render("Uptime: " + m.formatUptime(svc.Uptime)))
			}
		}
		b.WriteString("\n\n")
	}

	b.WriteString(HelpStyle.Render("Press any key to return..."))

	return b.String()
}

// Run starts the Control Center TUI
func Run() error {
	p := tea.NewProgram(New(), tea.WithAltScreen())
	_, err := p.Run()
	return err
}
