// ============================================================================
// meinDENKWERK (mDW) - Lokale KI-Plattform
// ============================================================================
//
// Package:     logviewer
// Description: Message types for async operations in LogViewer
// Author:      Mike Stoffels with Claude
// Created:     2025-12-07
// License:     MIT
// ============================================================================

package logviewer

import (
	"time"
)

// LogEntry represents a log entry from Bayes
type LogEntry struct {
	ID        string
	Timestamp time.Time
	Service   string
	Level     string
	Message   string
	RequestID string
	Fields    map[string]string
}

// LogLevel constants matching the proto definition
const (
	LevelDebug = "DEBUG"
	LevelInfo  = "INFO"
	LevelWarn  = "WARN"
	LevelError = "ERROR"
	LevelFatal = "FATAL"
)

// Message types for tea.Cmd async operations

// logsLoadedMsg is sent when logs are loaded from Bayes
type logsLoadedMsg struct {
	entries []LogEntry
	total   int
	err     error
}

// logStreamMsg is sent for each streamed log entry
type logStreamMsg struct {
	entry LogEntry
	err   error
}

// serviceStatusMsg is sent when service status is checked
type serviceStatusMsg struct {
	bayesOnline bool
	err         error
}

// statsLoadedMsg is sent when log stats are loaded
type statsLoadedMsg struct {
	totalLogs     int64
	logsByLevel   map[string]int64
	logsByService map[string]int64
	err           error
}

// tickMsg is used for periodic updates
type tickMsg time.Time

// refreshMsg signals a log refresh
type refreshMsg struct{}
