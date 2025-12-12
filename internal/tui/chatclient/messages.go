// ============================================================================
// meinDENKWERK (mDW) - Lokale KI-Plattform
// ============================================================================
//
// Package:     chatclient
// Description: Message types for async operations in ChatClient
// Author:      Mike Stoffels with Claude
// Created:     2025-12-07
// License:     MIT
// ============================================================================

package chatclient

import (
	"time"
)

// ChatMessage represents a message in the conversation
type ChatMessage struct {
	Role      string        // user, assistant, system
	Content   string        // message content
	Model     string        // model used (for assistant messages)
	Timestamp time.Time     // when the message was sent/received
	Duration  time.Duration // how long the response took (for assistant messages)
}

// ModelInfo represents an available model
type ModelInfo struct {
	Name        string
	Size        string
	Description string
	Provider    string // ollama, openai, anthropic, mistral
	Available   bool
}

// AgentInfo represents an available agent
type AgentInfo struct {
	ID          string
	Name        string
	Description string
	Tools       []string
}

// Message types for tea.Cmd async operations

// chatResponseMsg is sent when chat response is received
type chatResponseMsg struct {
	content  string
	model    string
	duration time.Duration
	err      error
}

// streamChunkMsg is sent for each streaming chunk
type streamChunkMsg struct {
	delta    string
	done     bool
	err      error
	duration time.Duration // Set when done
}

// streamSession holds the active streaming state
type streamSession struct {
	respCh    <-chan interface{} // *ollama.ChatResponse or gRPC chunk
	errCh     <-chan error
	startTime time.Time
}

// modelsLoadedMsg is sent when models are loaded
type modelsLoadedMsg struct {
	models []ModelInfo
	err    error
}

// serviceStatusMsg is sent when service status is checked
type serviceStatusMsg struct {
	turingOnline bool
	err          error
}

// tickMsg is used for periodic updates
type tickMsg time.Time

// clearInputMsg signals to clear the input
type clearInputMsg struct{}

// focusInputMsg signals to focus the input
type focusInputMsg struct{}

// scrollToBottomMsg signals to scroll viewport to bottom
type scrollToBottomMsg struct{}

// aristotelesPipelineMsg is sent when Aristoteles pipeline response is received
type aristotelesPipelineMsg struct {
	content      string
	intentType   string
	strategyName string
	qualityScore float32
	duration     time.Duration
	enrichments  []string
	err          error
	// Agent Pipeline Info
	agentID         string  // Agent ID wenn Leibniz verwendet wurde
	agentName       string  // Agent Name aus Metadata
	agentConfidence float64 // Agent Match Confidence aus Metadata
	targetService   string  // Ziel-Service (Turing, Leibniz, etc.)
	// Orchestrator Info (Multi-Task)
	isOrchestrated bool     // True wenn Orchestrator verwendet wurde
	taskCount      int      // Anzahl der Tasks
	agentsUsed     []string // Liste der verwendeten Agent-Namen
	executionMode  string   // "sequential" oder "parallel"
}

// aristotelesStatusMsg is sent when Aristoteles status is checked
type aristotelesStatusMsg struct {
	online bool
	err    error
}

// stepUpdateMsg is sent to update the current processing step display
type stepUpdateMsg struct {
	step string
}

// agentListMsg is sent when agent list is received from Leibniz
type agentListMsg struct {
	agents []AgentInfo
	err    error
}

// Pipeline step names for display
const (
	StepAnalyzing     = "Analysiere Anfrage"
	StepDecomposing   = "Zerlege in Teilaufgaben"
	StepMatching      = "Suche passende Agents"
	StepSearching     = "Web-Suche"
	StepFetching      = "Lade Inhalte"
	StepProcessing    = "Verarbeite Daten"
	StepGenerating    = "Generiere Antwort"
	StepOrchestrating = "Orchestriere Tasks"
)

// OrchestratorTask represents a task in the orchestrator pipeline
type OrchestratorTask struct {
	ID          string // Task ID (task_1, task_2, etc.)
	Description string // Task description
	AgentID     string // Assigned agent ID
	AgentName   string // Assigned agent name
	Status      string // pending, running, completed, failed
	Output      string // Task output (when completed)
}

// orchestratorProgressMsg is sent to update orchestrator progress in real-time
type orchestratorProgressMsg struct {
	currentTaskIndex int                 // Index of currently running task (0-based)
	totalTasks       int                 // Total number of tasks
	tasks            []*OrchestratorTask // All tasks with their status
	currentAgentName string              // Name of currently active agent
	phase            string              // Current phase (decomposing, matching, executing)
}
