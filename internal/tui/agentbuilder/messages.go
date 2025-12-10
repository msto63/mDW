// ============================================================================
// meinDENKWERK (mDW) - Lokale KI-Plattform
// ============================================================================
//
// Package:     agentbuilder
// Description: Message types for Agent Builder TUI
// Author:      Mike Stoffels with Claude
// Created:     2025-12-07
// License:     MIT
// ============================================================================

package agentbuilder

import (
	"time"
)

// AgentData represents an agent for display
type AgentData struct {
	ID                  string
	Name                string
	Description         string
	SystemPrompt        string
	Model               string
	Temperature         float32
	MaxIterations       int
	TimeoutSeconds      int
	Tools               []string
	UseKnowledgeBase    bool
	KnowledgeCollection string
	StreamingEnabled    bool
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

// ToolData represents a tool for display
type ToolData struct {
	Name                 string
	Description          string
	Source               string // builtin, mcp, custom
	Enabled              bool
	RequiresConfirmation bool
}

// ExecutionData represents a test execution result
type ExecutionData struct {
	ID         string
	Status     string // running, completed, error
	Response   string
	Iterations int
	Duration   time.Duration
	TotalTokens int
	Actions    []ActionData
}

// ActionData represents a tool action in execution
type ActionData struct {
	Tool     string
	Input    string
	Output   string
	Success  bool
	Duration time.Duration
}

// Message types for tea.Cmd async operations

// agentsLoadedMsg is sent when agents are loaded from Leibniz
type agentsLoadedMsg struct {
	agents []AgentData
	err    error
}

// agentCreatedMsg is sent when an agent is created
type agentCreatedMsg struct {
	agent AgentData
	err   error
}

// agentUpdatedMsg is sent when an agent is updated
type agentUpdatedMsg struct {
	agent AgentData
	err   error
}

// agentDeletedMsg is sent when an agent is deleted
type agentDeletedMsg struct {
	id  string
	err error
}

// toolsLoadedMsg is sent when tools are loaded
type toolsLoadedMsg struct {
	tools []ToolData
	err   error
}

// modelsLoadedMsg is sent when models are loaded from Turing
type modelsLoadedMsg struct {
	models []string
	err    error
}

// serviceStatusMsg is sent when service status is checked
type serviceStatusMsg struct {
	leibnizOnline bool
	turingOnline  bool
	err           error
}

// testStartedMsg is sent when a test execution starts
type testStartedMsg struct {
	executionID string
	err         error
}

// testChunkMsg is sent for streaming test output
type testChunkMsg struct {
	chunkType string // thinking, tool_call, tool_result, response, final
	content   string
	action    *ActionData
	iteration int
}

// testCompletedMsg is sent when a test execution completes
type testCompletedMsg struct {
	execution ExecutionData
	err       error
}

// tickMsg is used for periodic updates
type tickMsg time.Time

// refreshMsg signals a data refresh
type refreshMsg struct{}

// clearTestMsg clears the test conversation
type clearTestMsg struct{}

// Default agent templates
var DefaultAgentTemplates = []AgentData{
	{
		Name:           "Allgemeiner Assistent",
		Description:    "Hilfreicher KI-Assistent fuer allgemeine Aufgaben",
		Model:          "llama3.2:3b",
		Temperature:    0.7,
		MaxIterations:  10,
		TimeoutSeconds: 120,
		Tools:          []string{"calculator", "datetime"},
		SystemPrompt: `Du bist ein hilfreicher KI-Assistent, der Aufgaben schrittweise loest.

Fuer jede Aufgabe:
1. Ueberlege, welche Schritte noetig sind (THOUGHT)
2. Entscheide, welche Aktion oder welches Tool du verwenden willst (ACTION)
3. Fuehre die Aktion aus und werte das Ergebnis aus (OBSERVATION)
4. Wiederhole, bis die Aufgabe erledigt ist

Antworte im folgenden Format:
THOUGHT: [Deine Ueberlegung]
ACTION: [tool_name] oder FINAL_ANSWER
ACTION_INPUT: [Parameter als JSON]

Wenn du fertig bist:
THOUGHT: [Abschliessende Ueberlegung]
ACTION: FINAL_ANSWER
ACTION_INPUT: [Deine finale Antwort]`,
	},
	{
		Name:           "Recherche Agent",
		Description:    "Agent fuer Web-Recherche und Informationssuche",
		Model:          "llama3.2:3b",
		Temperature:    0.5,
		MaxIterations:  15,
		TimeoutSeconds: 180,
		Tools:          []string{"web_search", "datetime"},
		SystemPrompt: `Du bist ein Recherche-Agent, der Informationen im Web sucht und zusammenfasst.

Deine Aufgaben:
- Suche relevante Informationen zu einem Thema
- Fasse die Ergebnisse zusammen
- Gib Quellen an, wo moeglich

Nutze das web_search Tool um Informationen zu finden.`,
	},
	{
		Name:           "Code Assistent",
		Description:    "Agent fuer Programmierung und Code-Analyse",
		Model:          "llama3.2:3b",
		Temperature:    0.3,
		MaxIterations:  20,
		TimeoutSeconds: 240,
		Tools:          []string{"file_read", "file_write"},
		SystemPrompt: `Du bist ein erfahrener Programmierer und Code-Assistent.

Deine Faehigkeiten:
- Code schreiben und analysieren
- Bugs finden und beheben
- Code erklaeren und dokumentieren
- Best Practices empfehlen

Antworte praezise und mit Code-Beispielen wo sinnvoll.`,
	},
}

// GetTemplateByName returns a template by name
func GetTemplateByName(name string) *AgentData {
	for _, t := range DefaultAgentTemplates {
		if t.Name == name {
			return &t
		}
	}
	return nil
}
