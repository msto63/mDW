// ============================================================================
// meinDENKWERK (mDW) - Agent Orchestrator
// ============================================================================
//
// Koordiniert die intelligente Agent-Auswahl und Pipeline-Ausführung
// ============================================================================

package orchestrator

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/msto63/mDW/internal/aristoteles/decomposer"
	"github.com/msto63/mDW/pkg/core/logging"
)

// AgentInfo enthält Informationen über einen verfügbaren Agent
type AgentInfo struct {
	ID          string
	Name        string
	Description string
}

// AgentMatch repräsentiert ein Matching-Ergebnis
type AgentMatch struct {
	AgentID    string
	AgentName  string
	Similarity float64
}

// AgentMatcherFunc ist die Funktion zum Finden des besten Agents
type AgentMatcherFunc func(ctx context.Context, taskDescription string) (*AgentMatch, error)

// AgentExecutorFunc ist die Funktion zum Ausführen eines Agents
type AgentExecutorFunc func(ctx context.Context, agentID string, input string) (string, error)

// AgentListFunc ist die Funktion zum Auflisten aller Agents
type AgentListFunc func() []*AgentInfo

// Orchestrator koordiniert Task-Zerlegung, Agent-Matching und Ausführung
type Orchestrator struct {
	decomposer    *decomposer.Decomposer
	agentMatcher  AgentMatcherFunc
	agentExecutor AgentExecutorFunc
	agentList     AgentListFunc
	logger        *logging.Logger

	// Konfiguration
	defaultAgentID    string
	minConfidence     float64
	maxParallelTasks  int
}

// Config für den Orchestrator
type Config struct {
	DefaultAgentID   string
	MinConfidence    float64 // Mindest-Confidence für Agent-Matching
	MaxParallelTasks int
}

// DefaultConfig gibt die Standard-Konfiguration zurück
func DefaultConfig() Config {
	return Config{
		DefaultAgentID:   "default",
		MinConfidence:    0.3, // 30% Mindest-Ähnlichkeit
		MaxParallelTasks: 3,
	}
}

// NewOrchestrator erstellt einen neuen Orchestrator
func NewOrchestrator(cfg Config) *Orchestrator {
	return &Orchestrator{
		decomposer:       decomposer.NewDecomposer(),
		logger:           logging.New("orchestrator"),
		defaultAgentID:   cfg.DefaultAgentID,
		minConfidence:    cfg.MinConfidence,
		maxParallelTasks: cfg.MaxParallelTasks,
	}
}

// SetDecomposerLLM setzt die LLM-Funktion für den Decomposer
func (o *Orchestrator) SetDecomposerLLM(fn decomposer.LLMFunc) {
	o.decomposer.SetLLMFunc(fn)
}

// SetAgentMatcher setzt die Agent-Matching-Funktion
func (o *Orchestrator) SetAgentMatcher(fn AgentMatcherFunc) {
	o.agentMatcher = fn
}

// SetAgentExecutor setzt die Agent-Ausführungs-Funktion
func (o *Orchestrator) SetAgentExecutor(fn AgentExecutorFunc) {
	o.agentExecutor = fn
}

// SetAgentList setzt die Funktion zum Auflisten der Agents
func (o *Orchestrator) SetAgentList(fn AgentListFunc) {
	o.agentList = fn
}

// Process verarbeitet einen User-Prompt vollständig
func (o *Orchestrator) Process(ctx context.Context, prompt string) (*OrchestrationResult, error) {
	startTime := time.Now()
	requestID := uuid.New().String()[:8]

	o.logger.Info("Starting orchestration", "request_id", requestID, "prompt_length", len(prompt))

	// Prüfe ob alle Funktionen gesetzt sind
	if o.agentMatcher == nil || o.agentExecutor == nil {
		return nil, fmt.Errorf("orchestrator not fully configured: missing agent matcher or executor")
	}

	// Phase 1: Task Decomposition
	o.logger.Debug("Phase 1: Decomposing prompt")
	decomposition, err := o.decomposer.Decompose(ctx, prompt)
	if err != nil {
		return nil, fmt.Errorf("decomposition failed: %w", err)
	}

	o.logger.Info("Prompt decomposed",
		"task_count", len(decomposition.Tasks),
		"sequential", decomposition.IsSequential)

	// Einfacher Prompt? Direkt ausführen
	if len(decomposition.Tasks) == 1 {
		return o.executeSingleTask(ctx, requestID, decomposition.Tasks[0], prompt, startTime)
	}

	// Phase 2: Agent Matching für jede Task
	o.logger.Debug("Phase 2: Matching agents to tasks")
	plan, err := o.createExecutionPlan(ctx, requestID, prompt, decomposition)
	if err != nil {
		return nil, fmt.Errorf("planning failed: %w", err)
	}

	// Phase 3: Pipeline Execution
	o.logger.Debug("Phase 3: Executing pipeline")
	result, err := o.executePipeline(ctx, plan)
	if err != nil {
		return nil, fmt.Errorf("pipeline execution failed: %w", err)
	}

	result.RequestID = requestID
	result.TotalDuration = time.Since(startTime)

	o.logger.Info("Orchestration completed",
		"request_id", requestID,
		"tasks", len(plan.Tasks),
		"duration_ms", result.TotalDuration.Milliseconds(),
		"success", result.Success)

	return result, nil
}

// executeSingleTask führt eine einzelne Task direkt aus
func (o *Orchestrator) executeSingleTask(ctx context.Context, requestID string, task *decomposer.Task, originalPrompt string, startTime time.Time) (*OrchestrationResult, error) {
	// Agent finden
	match, err := o.agentMatcher(ctx, task.Description)
	if err != nil {
		o.logger.Warn("Agent matching failed, using default", "error", err)
		match = &AgentMatch{
			AgentID:    o.defaultAgentID,
			AgentName:  "Default Agent",
			Similarity: 0,
		}
	}

	// Confidence-Check
	if match.Similarity < o.minConfidence {
		o.logger.Debug("Low confidence match, using default agent",
			"matched", match.AgentID,
			"confidence", match.Similarity,
			"threshold", o.minConfidence)
		match.AgentID = o.defaultAgentID
	}

	o.logger.Info("Single task execution",
		"agent", match.AgentID,
		"confidence", match.Similarity)

	// Ausführen
	taskStart := time.Now()
	output, err := o.agentExecutor(ctx, match.AgentID, originalPrompt)

	taskResult := &TaskResult{
		TaskID:    task.ID,
		AgentID:   match.AgentID,
		AgentName: match.AgentName,
		Input:     originalPrompt,
		Output:    output,
		Duration:  time.Since(taskStart),
		Success:   err == nil,
	}
	if err != nil {
		taskResult.Error = err.Error()
	}

	// Plan für Transparenz erstellen
	plan := &ExecutionPlan{
		ID:             requestID,
		OriginalPrompt: originalPrompt,
		Tasks: []*PlannedTask{
			{
				ID:              task.ID,
				Description:     task.Description,
				AssignedAgentID: match.AgentID,
				AgentName:       match.AgentName,
				MatchConfidence: match.Similarity,
			},
		},
		IsSequential: false,
		CreatedAt:    startTime,
	}

	return &OrchestrationResult{
		RequestID:     requestID,
		FinalOutput:   output,
		Plan:          plan,
		TaskResults:   map[string]*TaskResult{task.ID: taskResult},
		TotalDuration: time.Since(startTime),
		Success:       err == nil,
		Error:         errorString(err),
	}, nil
}

// createExecutionPlan erstellt den Ausführungsplan mit Agent-Zuweisungen
func (o *Orchestrator) createExecutionPlan(ctx context.Context, requestID, originalPrompt string, decomposition *decomposer.DecompositionResult) (*ExecutionPlan, error) {
	plan := &ExecutionPlan{
		ID:             requestID,
		OriginalPrompt: originalPrompt,
		Tasks:          make([]*PlannedTask, len(decomposition.Tasks)),
		IsSequential:   decomposition.IsSequential,
		CreatedAt:      time.Now(),
	}

	for i, task := range decomposition.Tasks {
		// Agent für diese Task finden
		match, err := o.agentMatcher(ctx, task.Description)
		if err != nil {
			o.logger.Warn("Agent matching failed for task, using default",
				"task", task.ID,
				"error", err)
			match = &AgentMatch{
				AgentID:    o.defaultAgentID,
				AgentName:  "Default Agent",
				Similarity: 0,
			}
		}

		// Confidence-Check
		if match.Similarity < o.minConfidence {
			o.logger.Debug("Low confidence for task, using default",
				"task", task.ID,
				"matched", match.AgentID,
				"confidence", match.Similarity)
			match.AgentID = o.defaultAgentID
		}

		plan.Tasks[i] = &PlannedTask{
			ID:              task.ID,
			Description:     task.Description,
			Dependencies:    task.Dependencies,
			AssignedAgentID: match.AgentID,
			AgentName:       match.AgentName,
			MatchConfidence: match.Similarity,
		}

		o.logger.Debug("Agent assigned to task",
			"task", task.ID,
			"agent", match.AgentID,
			"confidence", match.Similarity)
	}

	return plan, nil
}

// executePipeline führt die Pipeline sequentiell aus
func (o *Orchestrator) executePipeline(ctx context.Context, plan *ExecutionPlan) (*OrchestrationResult, error) {
	pipelineCtx := NewPipelineContext(plan.OriginalPrompt)

	for i, task := range plan.Tasks {
		pipelineCtx.CurrentTaskIndex = i

		// Kontext aus vorherigen Tasks zusammenstellen
		previousContext := pipelineCtx.GetOutputForDependencies(task.Dependencies)

		// Input für den Agent bauen
		agentInput := o.buildAgentInput(task, previousContext, plan.OriginalPrompt)

		o.logger.Debug("Executing task",
			"task", task.ID,
			"agent", task.AssignedAgentID,
			"has_context", previousContext != "")

		// Agent ausführen
		taskStart := time.Now()
		output, err := o.agentExecutor(ctx, task.AssignedAgentID, agentInput)

		taskResult := &TaskResult{
			TaskID:    task.ID,
			AgentID:   task.AssignedAgentID,
			AgentName: task.AgentName,
			Input:     agentInput,
			Output:    output,
			Duration:  time.Since(taskStart),
			Success:   err == nil,
		}

		if err != nil {
			taskResult.Error = err.Error()
			o.logger.Error("Task execution failed",
				"task", task.ID,
				"agent", task.AssignedAgentID,
				"error", err)

			// Bei Fehler: Pipeline abbrechen oder fortfahren?
			// Aktuell: Fortfahren mit leerem Output
			taskResult.Output = ""
		}

		pipelineCtx.TaskResults[task.ID] = taskResult
		pipelineCtx.AccumulatedOutput = taskResult.Output

		o.logger.Info("Task completed",
			"task", task.ID,
			"agent", task.AssignedAgentID,
			"duration_ms", taskResult.Duration.Milliseconds(),
			"success", taskResult.Success)
	}

	// Finales Ergebnis
	return &OrchestrationResult{
		FinalOutput:   pipelineCtx.AccumulatedOutput,
		Plan:          plan,
		TaskResults:   pipelineCtx.TaskResults,
		TotalDuration: time.Since(pipelineCtx.StartTime),
		Success:       o.allTasksSuccessful(pipelineCtx.TaskResults),
	}, nil
}

// buildAgentInput baut den Input für einen Agent
func (o *Orchestrator) buildAgentInput(task *PlannedTask, previousContext, originalPrompt string) string {
	var parts []string

	// Ursprünglicher Prompt als Kontext
	parts = append(parts, fmt.Sprintf("Ursprüngliche Anfrage: %s", originalPrompt))

	// Aktuelle Aufgabe
	parts = append(parts, fmt.Sprintf("\nDeine Aufgabe: %s", task.Description))

	// Kontext aus vorherigen Schritten
	if previousContext != "" {
		parts = append(parts, fmt.Sprintf("\nErgebnisse aus vorherigen Schritten:\n%s", previousContext))
	}

	return strings.Join(parts, "\n")
}

// allTasksSuccessful prüft ob alle Tasks erfolgreich waren
func (o *Orchestrator) allTasksSuccessful(results map[string]*TaskResult) bool {
	for _, result := range results {
		if !result.Success {
			return false
		}
	}
	return true
}

// errorString konvertiert einen Error zu einem String (oder leer)
func errorString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

// GetAvailableAgents gibt die Liste der verfügbaren Agents zurück
func (o *Orchestrator) GetAvailableAgents() []*AgentInfo {
	if o.agentList == nil {
		return nil
	}
	return o.agentList()
}
