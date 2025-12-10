// ============================================================================
// meinDENKWERK (mDW) - Lokale KI-Plattform
// ============================================================================
//
// Package:     logviewer
// Description: Main Bubbletea model for mDW LogViewer
// Author:      Mike Stoffels with Claude
// Created:     2025-12-07
// License:     MIT
// ============================================================================

package logviewer

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	bayespb "github.com/msto63/mDW/api/gen/bayes"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// Version is set during build
var Version = "0.1.0"

// LevelFilter tracks which log levels are enabled
type LevelFilter struct {
	Debug bool
	Info  bool
	Warn  bool
	Error bool
	Fatal bool
}

// Model is the main Bubbletea model for LogViewer
type Model struct {
	// State
	width       int
	height      int
	ready       bool
	loading     bool
	paused      bool
	bayesOnline bool
	autoScroll  bool
	err         error

	// Components
	viewport viewport.Model
	spinner  spinner.Model

	// Log state
	allLogs      []LogEntry
	filteredLogs []LogEntry
	levelFilter  LevelFilter
	serviceFilter string
	searchFilter  string

	// Stats
	totalLogs     int64
	logsByLevel   map[string]int64
	logsByService map[string]int64

	// Configuration
	bayesAddr   string
	maxLogCount int
}

// Config holds LogViewer configuration
type Config struct {
	BayesAddr   string
	MaxLogCount int
}

// DefaultConfig returns default configuration
func DefaultConfig() Config {
	return Config{
		BayesAddr:   "localhost:9120",
		MaxLogCount: 1000,
	}
}

// New creates a new LogViewer model
func New(cfg Config) Model {
	// Setup spinner
	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = lipgloss.NewStyle().Foreground(ColorPrimary)

	return Model{
		spinner:     sp,
		allLogs:     []LogEntry{},
		levelFilter: LevelFilter{
			Debug: true,
			Info:  true,
			Warn:  true,
			Error: true,
			Fatal: true,
		},
		autoScroll:    true,
		bayesAddr:     cfg.BayesAddr,
		maxLogCount:   cfg.MaxLogCount,
		logsByLevel:   make(map[string]int64),
		logsByService: make(map[string]int64),
	}
}

// Init initializes the model
func (m Model) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.Tick,
		m.checkBayesStatus,
		m.loadLogs,
		tea.EnterAltScreen,
		tea.Tick(2*time.Second, func(t time.Time) tea.Msg {
			return tickMsg(t)
		}),
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

		headerHeight := 4  // Title + filter bar
		footerHeight := 4  // Status bar + help
		viewportHeight := msg.Height - headerHeight - footerHeight

		if !m.ready {
			m.viewport = viewport.New(msg.Width-4, viewportHeight)
			m.viewport.YPosition = headerHeight
			m.ready = true
		} else {
			m.viewport.Width = msg.Width - 4
			m.viewport.Height = viewportHeight
		}
		m.updateViewportContent()

	case spinner.TickMsg:
		if m.loading {
			m.spinner, cmd = m.spinner.Update(msg)
			cmds = append(cmds, cmd)
		}

	case logsLoadedMsg:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err
		} else {
			m.allLogs = msg.entries
			m.applyFilters()
			m.updateViewportContent()
			if m.autoScroll {
				m.viewport.GotoBottom()
			}
		}

	case serviceStatusMsg:
		m.bayesOnline = msg.bayesOnline

	case statsLoadedMsg:
		if msg.err == nil {
			m.totalLogs = msg.totalLogs
			m.logsByLevel = msg.logsByLevel
			m.logsByService = msg.logsByService
		}

	case tickMsg:
		if !m.paused {
			cmds = append(cmds, m.loadLogs)
		}
		cmds = append(cmds, tea.Tick(2*time.Second, func(t time.Time) tea.Msg {
			return tickMsg(t)
		}))

	case refreshMsg:
		m.loading = true
		cmds = append(cmds, m.loadLogs)
	}

	// Update viewport
	m.viewport, cmd = m.viewport.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

// handleKeyPress handles keyboard input
func (m Model) handleKeyPress(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyCtrlC:
		return m, tea.Quit

	case tea.KeyRunes:
		switch string(msg.Runes) {
		// Log level filters - number keys
		case "1":
			m.levelFilter.Debug = !m.levelFilter.Debug
			m.applyFilters()
			m.updateViewportContent()
			return m, nil
		case "2":
			m.levelFilter.Info = !m.levelFilter.Info
			m.applyFilters()
			m.updateViewportContent()
			return m, nil
		case "3":
			m.levelFilter.Warn = !m.levelFilter.Warn
			m.applyFilters()
			m.updateViewportContent()
			return m, nil
		case "4":
			m.levelFilter.Error = !m.levelFilter.Error
			m.applyFilters()
			m.updateViewportContent()
			return m, nil
		case "5":
			m.levelFilter.Fatal = !m.levelFilter.Fatal
			m.applyFilters()
			m.updateViewportContent()
			return m, nil

		// Show all levels
		case "0":
			m.levelFilter = LevelFilter{
				Debug: true,
				Info:  true,
				Warn:  true,
				Error: true,
				Fatal: true,
			}
			m.applyFilters()
			m.updateViewportContent()
			return m, nil

		// Pause/Resume
		case "p", " ":
			m.paused = !m.paused
			return m, nil

		// Refresh
		case "r":
			m.loading = true
			return m, m.loadLogs

		// Auto-scroll toggle
		case "a":
			m.autoScroll = !m.autoScroll
			if m.autoScroll {
				m.viewport.GotoBottom()
			}
			return m, nil

		// Clear logs display
		case "c":
			m.allLogs = []LogEntry{}
			m.filteredLogs = []LogEntry{}
			m.updateViewportContent()
			return m, nil

		// Go to top
		case "g":
			m.viewport.GotoTop()
			m.autoScroll = false
			return m, nil

		// Go to bottom
		case "G":
			m.viewport.GotoBottom()
			m.autoScroll = true
			return m, nil
		}

	case tea.KeyPgUp:
		m.viewport.ViewUp()
		m.autoScroll = false
		return m, nil

	case tea.KeyPgDown:
		m.viewport.ViewDown()
		return m, nil

	case tea.KeyUp:
		m.viewport.LineUp(1)
		m.autoScroll = false
		return m, nil

	case tea.KeyDown:
		m.viewport.LineDown(1)
		return m, nil
	}

	return m, nil
}

// View renders the UI
func (m Model) View() string {
	if !m.ready {
		return "Lade LogViewer..."
	}

	var b strings.Builder

	// Header with logo
	b.WriteString(m.renderHeader())
	b.WriteString("\n")

	// Filter bar
	b.WriteString(m.renderFilterBar())
	b.WriteString("\n")

	// Log viewport
	b.WriteString(m.renderLogArea())
	b.WriteString("\n")

	// Status bar
	b.WriteString(m.renderStatusBar())
	b.WriteString("\n")

	// Help bar
	b.WriteString(m.renderHelpBar())

	return b.String()
}

// renderHeader renders the header with logo and status
func (m Model) renderHeader() string {
	logo := LogoStyle.Render(Logo)

	// Status indicator
	var status string
	if m.bayesOnline {
		status = StatusOnlineStyle.Render(IconOnline + "Bayes verbunden")
	} else {
		status = StatusOfflineStyle.Render(IconOffline + "Bayes offline")
	}

	// Pause indicator
	pauseStatus := ""
	if m.paused {
		pauseStatus = "  " + StatusPausedStyle.Render(IconPaused + "PAUSIERT")
	}

	header := lipgloss.JoinHorizontal(lipgloss.Center,
		logo,
		strings.Repeat(" ", 3),
		status,
		pauseStatus,
	)

	return TitlePanelStyle.Width(m.width - 4).Render(header)
}

// renderFilterBar renders the log level filter bar
func (m Model) renderFilterBar() string {
	filters := []string{
		fmt.Sprintf("1:%s", RenderFilterStatus("DEBUG", m.levelFilter.Debug)),
		fmt.Sprintf("2:%s", RenderFilterStatus("INFO", m.levelFilter.Info)),
		fmt.Sprintf("3:%s", RenderFilterStatus("WARN", m.levelFilter.Warn)),
		fmt.Sprintf("4:%s", RenderFilterStatus("ERROR", m.levelFilter.Error)),
		fmt.Sprintf("5:%s", RenderFilterStatus("FATAL", m.levelFilter.Fatal)),
	}

	// Count visible logs
	visibleCount := len(m.filteredLogs)
	totalCount := len(m.allLogs)

	filterStr := IconFilter + strings.Join(filters, "  ")
	countStr := HelpDescStyle.Render(fmt.Sprintf("[%d/%d Logs]", visibleCount, totalCount))

	// Auto-scroll indicator
	scrollStr := ""
	if m.autoScroll {
		scrollStr = "  " + FilterActiveStyle.Render("[Auto-Scroll]")
	}

	content := filterStr + "  " + countStr + scrollStr

	return FilterBarStyle.Width(m.width - 2).Render(content)
}

// renderLogArea renders the main log viewport
func (m Model) renderLogArea() string {
	style := LogPanelStyle.Width(m.width - 2).Height(m.viewport.Height + 2)
	return style.Render(m.viewport.View())
}

// renderStatusBar renders the status bar
func (m Model) renderStatusBar() string {
	// Left: Log count info
	leftPart := HelpDescStyle.Render(fmt.Sprintf("Logs: %d", m.totalLogs))

	// Center: Version
	centerPart := HelpDescStyle.Render("v" + Version)

	// Right: Connection status
	var rightPart string
	if m.loading {
		rightPart = m.spinner.View() + " Lade..."
	} else if m.bayesOnline {
		rightPart = StatusOnlineStyle.Render("Bayes:9120")
	} else {
		rightPart = StatusOfflineStyle.Render("Offline")
	}

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
	items := []string{
		RenderKeyHint("1-5", "Level"),
		RenderKeyHint("0", "Alle"),
		RenderKeyHint("p", "Pause"),
		RenderKeyHint("r", "Refresh"),
		RenderKeyHint("a", "AutoScroll"),
		RenderKeyHint("g/G", "Top/Bottom"),
		RenderKeyHint("Ctrl+C", "Beenden"),
	}

	return HelpStyle.Render(strings.Join(items, "  "))
}

// updateViewportContent updates the viewport with filtered logs
func (m *Model) updateViewportContent() {
	var content strings.Builder

	for _, log := range m.filteredLogs {
		// Format: [TIME] [LEVEL] [SERVICE] Message
		timeStr := LogTimestampStyle.Render(log.Timestamp.Format("15:04:05"))
		levelStr := RenderLevelBadge(log.Level)
		serviceStr := LogServiceStyle.Render(fmt.Sprintf("[%-10s]", truncateString(log.Service, 10)))
		msgStr := LogMessageStyle.Render(log.Message)

		line := fmt.Sprintf("%s %s %s %s", timeStr, levelStr, serviceStr, msgStr)
		content.WriteString(line)
		content.WriteString("\n")
	}

	m.viewport.SetContent(content.String())
}

// applyFilters filters logs based on current filter settings
func (m *Model) applyFilters() {
	m.filteredLogs = make([]LogEntry, 0)

	for _, log := range m.allLogs {
		// Check level filter
		switch log.Level {
		case LevelDebug:
			if !m.levelFilter.Debug {
				continue
			}
		case LevelInfo:
			if !m.levelFilter.Info {
				continue
			}
		case LevelWarn, "WARNING":
			if !m.levelFilter.Warn {
				continue
			}
		case LevelError:
			if !m.levelFilter.Error {
				continue
			}
		case LevelFatal:
			if !m.levelFilter.Fatal {
				continue
			}
		}

		// Check service filter
		if m.serviceFilter != "" && !strings.Contains(strings.ToLower(log.Service), strings.ToLower(m.serviceFilter)) {
			continue
		}

		// Check search filter
		if m.searchFilter != "" && !strings.Contains(strings.ToLower(log.Message), strings.ToLower(m.searchFilter)) {
			continue
		}

		m.filteredLogs = append(m.filteredLogs, log)
	}
}

// loadLogs loads logs from Bayes service
func (m Model) loadLogs() tea.Msg {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn, err := grpc.DialContext(ctx, m.bayesAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	if err != nil {
		// Try to load from local log file as fallback
		return m.loadLogsFromFile()
	}
	defer conn.Close()

	client := bayespb.NewBayesServiceClient(conn)

	resp, err := client.QueryLogs(ctx, &bayespb.QueryLogsRequest{
		Limit: int32(m.maxLogCount),
		Sort:  bayespb.SortOrder_SORT_ORDER_DESC,
	})
	if err != nil {
		return logsLoadedMsg{err: err}
	}

	entries := make([]LogEntry, 0, len(resp.Entries))
	for _, e := range resp.Entries {
		level := levelFromProto(e.Level)
		entries = append(entries, LogEntry{
			ID:        fmt.Sprintf("%d", e.Timestamp),
			Timestamp: time.Unix(0, e.Timestamp*int64(time.Millisecond)),
			Service:   e.Service,
			Level:     level,
			Message:   e.Message,
			RequestID: e.RequestId,
			Fields:    e.Fields,
		})
	}

	// Reverse to get oldest first
	for i, j := 0, len(entries)-1; i < j; i, j = i+1, j-1 {
		entries[i], entries[j] = entries[j], entries[i]
	}

	return logsLoadedMsg{entries: entries, total: int(resp.Total)}
}

// loadLogsFromFile loads logs from local file as fallback
func (m Model) loadLogsFromFile() tea.Msg {
	// For now, return empty if Bayes is not available
	// This could be extended to read from a local log file
	return logsLoadedMsg{
		entries: []LogEntry{
			{
				ID:        "demo-1",
				Timestamp: time.Now().Add(-5 * time.Minute),
				Service:   "bayes",
				Level:     LevelInfo,
				Message:   "LogViewer gestartet - warte auf Bayes-Service...",
			},
			{
				ID:        "demo-2",
				Timestamp: time.Now(),
				Service:   "logviewer",
				Level:     LevelWarn,
				Message:   "Bayes-Service nicht erreichbar auf " + m.bayesAddr,
			},
		},
		total: 2,
	}
}

// checkBayesStatus checks if Bayes service is available
func (m Model) checkBayesStatus() tea.Msg {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	conn, err := grpc.DialContext(ctx, m.bayesAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	if err != nil {
		return serviceStatusMsg{bayesOnline: false, err: err}
	}
	conn.Close()
	return serviceStatusMsg{bayesOnline: true}
}

// levelFromProto converts proto LogLevel to string
func levelFromProto(level bayespb.LogLevel) string {
	switch level {
	case bayespb.LogLevel_LOG_LEVEL_DEBUG:
		return LevelDebug
	case bayespb.LogLevel_LOG_LEVEL_INFO:
		return LevelInfo
	case bayespb.LogLevel_LOG_LEVEL_WARN:
		return LevelWarn
	case bayespb.LogLevel_LOG_LEVEL_ERROR:
		return LevelError
	case bayespb.LogLevel_LOG_LEVEL_FATAL:
		return LevelFatal
	default:
		return LevelInfo
	}
}

// truncateString truncates a string to max length
func truncateString(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-1] + "~"
}

// Run starts the LogViewer TUI
func Run(cfg Config) error {
	p := tea.NewProgram(New(cfg), tea.WithAltScreen())
	_, err := p.Run()
	return err
}
