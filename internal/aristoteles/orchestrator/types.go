// ============================================================================
// meinDENKWERK (mDW) - Orchestrator Types
// ============================================================================
//
// Typen für die Agent-Orchestrierung und Pipeline-Ausführung
// ============================================================================

package orchestrator

import (
	"time"
)

// PlannedTask ist eine Task mit zugewiesenem Agent
type PlannedTask struct {
	ID              string   `json:"id"`
	Description     string   `json:"description"`
	Dependencies    []string `json:"dependencies,omitempty"`
	AssignedAgentID string   `json:"assigned_agent_id"`
	AgentName       string   `json:"agent_name"`
	MatchConfidence float64  `json:"match_confidence"`
}

// ExecutionPlan ist der Plan für die Pipeline-Ausführung
type ExecutionPlan struct {
	ID             string         `json:"id"`
	OriginalPrompt string         `json:"original_prompt"`
	Tasks          []*PlannedTask `json:"tasks"`
	IsSequential   bool           `json:"is_sequential"`
	CreatedAt      time.Time      `json:"created_at"`
}

// GetAgentSequence gibt die Sequenz der Agents zurück
func (p *ExecutionPlan) GetAgentSequence() []string {
	agents := make([]string, len(p.Tasks))
	for i, task := range p.Tasks {
		agents[i] = task.AssignedAgentID
	}
	return agents
}

// TaskResult ist das Ergebnis einer einzelnen Task-Ausführung
type TaskResult struct {
	TaskID    string        `json:"task_id"`
	AgentID   string        `json:"agent_id"`
	AgentName string        `json:"agent_name"`
	Input     string        `json:"input"`
	Output    string        `json:"output"`
	Duration  time.Duration `json:"duration_ms"`
	Success   bool          `json:"success"`
	Error     string        `json:"error,omitempty"`
}

// PipelineResult ist das Gesamtergebnis der Pipeline
type PipelineResult struct {
	PlanID        string                 `json:"plan_id"`
	FinalOutput   string                 `json:"final_output"`
	TaskResults   map[string]*TaskResult `json:"task_results"`
	AgentSequence []string               `json:"agent_sequence"`
	TotalDuration time.Duration          `json:"total_duration_ms"`
	Success       bool                   `json:"success"`
	Error         string                 `json:"error,omitempty"`
}

// PipelineContext hält den Zustand während der Pipeline-Ausführung
type PipelineContext struct {
	OriginalPrompt    string
	AccumulatedOutput string
	TaskResults       map[string]*TaskResult
	CurrentTaskIndex  int
	StartTime         time.Time
}

// NewPipelineContext erstellt einen neuen Pipeline-Kontext
func NewPipelineContext(originalPrompt string) *PipelineContext {
	return &PipelineContext{
		OriginalPrompt: originalPrompt,
		TaskResults:    make(map[string]*TaskResult),
		StartTime:      time.Now(),
	}
}

// GetOutputForDependencies sammelt die Outputs der abhängigen Tasks
func (c *PipelineContext) GetOutputForDependencies(dependencies []string) string {
	if len(dependencies) == 0 {
		return ""
	}

	var outputs []string
	for _, depID := range dependencies {
		if result, ok := c.TaskResults[depID]; ok && result.Success {
			outputs = append(outputs, result.Output)
		}
	}

	if len(outputs) == 0 {
		return ""
	}

	return joinOutputs(outputs)
}

// joinOutputs kombiniert mehrere Outputs
func joinOutputs(outputs []string) string {
	if len(outputs) == 1 {
		return outputs[0]
	}

	result := ""
	for i, output := range outputs {
		if i > 0 {
			result += "\n\n---\n\n"
		}
		result += output
	}
	return result
}

// OrchestrationResult ist das finale Ergebnis der Orchestrierung
type OrchestrationResult struct {
	RequestID     string                 `json:"request_id"`
	FinalOutput   string                 `json:"final_output"`
	Plan          *ExecutionPlan         `json:"plan"`
	TaskResults   map[string]*TaskResult `json:"task_results"`
	TotalDuration time.Duration          `json:"total_duration_ms"`
	Success       bool                   `json:"success"`
	Error         string                 `json:"error,omitempty"`
}
