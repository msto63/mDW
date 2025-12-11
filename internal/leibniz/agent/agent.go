package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/msto63/mDW/pkg/core/logging"
)

// Tool represents a callable tool
type Tool struct {
	Name        string
	Description string
	Parameters  map[string]ParameterDef
	Handler     ToolHandler
}

// ParameterDef defines a tool parameter
type ParameterDef struct {
	Type        string
	Description string
	Required    bool
}

// ToolHandler is a function that handles tool execution
type ToolHandler func(ctx context.Context, params map[string]interface{}) (interface{}, error)

// ToolCall represents a tool invocation
type ToolCall struct {
	Name   string
	Params map[string]interface{}
}

// ToolResult represents the result of a tool call
type ToolResult struct {
	Tool   string
	Result interface{}
	Error  string
}

// Step represents a single agent step
type Step struct {
	Index     int
	Thought   string
	Action    string
	ToolCall  *ToolCall
	ToolResult *ToolResult
	Timestamp time.Time
}

// ExecutionStatus represents the status of agent execution
type ExecutionStatus string

const (
	StatusPending   ExecutionStatus = "pending"
	StatusRunning   ExecutionStatus = "running"
	StatusCompleted ExecutionStatus = "completed"
	StatusFailed    ExecutionStatus = "failed"
	StatusCancelled ExecutionStatus = "cancelled"
)

// Execution represents an agent execution
type Execution struct {
	ID        string
	Task      string
	Status    ExecutionStatus
	Steps     []Step
	Result    string
	Error     string
	StartedAt time.Time
	EndedAt   time.Time
	ToolsUsed []string
}

// LLMFunc is a function that generates LLM responses
type LLMFunc func(ctx context.Context, messages []Message) (string, error)

// ModelAwareLLMFunc is a function that generates LLM responses with model selection
type ModelAwareLLMFunc func(ctx context.Context, model string, messages []Message) (string, error)

// Message represents a chat message
type Message struct {
	Role    string
	Content string
}

// Agent is an AI agent that can use tools
type Agent struct {
	tools             map[string]*Tool
	llmFunc           LLMFunc
	modelAwareLLMFunc ModelAwareLLMFunc
	logger            *logging.Logger
	maxSteps          int
	systemPrompt      string
	model             string // Model to use for this execution
}

// Config holds agent configuration
type Config struct {
	MaxSteps     int
	SystemPrompt string
	LLMFunc      LLMFunc
}

// DefaultConfig returns default agent configuration
func DefaultConfig() Config {
	return Config{
		MaxSteps: 10,
		SystemPrompt: `Du bist ein hilfreicher KI-Assistent, der Aufgaben schrittweise löst.

Für jede Aufgabe:
1. Überlege, welche Schritte nötig sind (THOUGHT)
2. Entscheide, welche Aktion oder welches Tool du verwenden willst (ACTION)
3. Führe die Aktion aus und werte das Ergebnis aus (OBSERVATION)
4. Wiederhole, bis die Aufgabe erledigt ist

Verfügbare Tools:
{{TOOLS}}

Antworte im folgenden Format:
THOUGHT: [Deine Überlegung]
ACTION: [tool_name] oder FINAL_ANSWER
ACTION_INPUT: [Parameter als JSON]

Wenn du fertig bist:
THOUGHT: [Abschließende Überlegung]
ACTION: FINAL_ANSWER
ACTION_INPUT: [Deine finale Antwort]`,
	}
}

// NewAgent creates a new agent
func NewAgent(cfg Config) *Agent {
	return &Agent{
		tools:        make(map[string]*Tool),
		llmFunc:      cfg.LLMFunc,
		logger:       logging.New("leibniz-agent"),
		maxSteps:     cfg.MaxSteps,
		systemPrompt: cfg.SystemPrompt,
	}
}

// SetLLMFunc sets the LLM function
func (a *Agent) SetLLMFunc(fn LLMFunc) {
	a.llmFunc = fn
}

// SetModelAwareLLMFunc sets the model-aware LLM function
func (a *Agent) SetModelAwareLLMFunc(fn ModelAwareLLMFunc) {
	a.modelAwareLLMFunc = fn
}

// SetModel sets the model to use for execution
func (a *Agent) SetModel(model string) {
	a.model = model
}

// SetSystemPrompt sets the system prompt for this agent
func (a *Agent) SetSystemPrompt(prompt string) {
	a.systemPrompt = prompt
}

// GetSystemPrompt returns the current system prompt
func (a *Agent) GetSystemPrompt() string {
	return a.systemPrompt
}

// GetModel returns the current model
func (a *Agent) GetModel() string {
	return a.model
}

// RegisterTool registers a tool
func (a *Agent) RegisterTool(tool *Tool) {
	a.tools[tool.Name] = tool
	a.logger.Info("Tool registered", "name", tool.Name)
}

// UnregisterTool removes a tool
func (a *Agent) UnregisterTool(name string) {
	delete(a.tools, name)
}

// ListTools returns all registered tools
func (a *Agent) ListTools() []*Tool {
	tools := make([]*Tool, 0, len(a.tools))
	for _, t := range a.tools {
		tools = append(tools, t)
	}
	return tools
}

// Execute runs the agent with a task
func (a *Agent) Execute(ctx context.Context, task string) (*Execution, error) {
	if a.llmFunc == nil && a.modelAwareLLMFunc == nil {
		return nil, fmt.Errorf("LLM function not set")
	}

	execution := &Execution{
		ID:        fmt.Sprintf("exec-%d", time.Now().UnixNano()),
		Task:      task,
		Status:    StatusRunning,
		Steps:     []Step{},
		StartedAt: time.Now(),
		ToolsUsed: []string{},
	}

	a.logger.Info("Starting agent execution",
		"id", execution.ID,
		"task", task,
	)

	// Build system prompt with tools
	systemPrompt := a.buildSystemPrompt()

	// Initialize conversation
	messages := []Message{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: task},
	}

	// Execute steps
	for step := 0; step < a.maxSteps; step++ {
		select {
		case <-ctx.Done():
			execution.Status = StatusCancelled
			execution.Error = "cancelled"
			execution.EndedAt = time.Now()
			return execution, ctx.Err()
		default:
		}

		// Get LLM response - prefer model-aware function if available
		var response string
		var err error
		if a.modelAwareLLMFunc != nil {
			response, err = a.modelAwareLLMFunc(ctx, a.model, messages)
		} else {
			response, err = a.llmFunc(ctx, messages)
		}
		if err != nil {
			execution.Status = StatusFailed
			execution.Error = fmt.Sprintf("LLM error: %v", err)
			execution.EndedAt = time.Now()
			return execution, err
		}

		// Parse response
		stepResult := a.parseResponse(response)
		stepResult.Index = step
		stepResult.Timestamp = time.Now()

		// If no structured response detected, treat the whole response as final answer
		if stepResult.Action == "" && stepResult.Thought == "" {
			// LLM returned a direct answer without ReAct format
			a.logger.Info("Direct response detected, treating as final answer")
			stepResult.Action = "FINAL_ANSWER"
			stepResult.ToolCall = &ToolCall{
				Name:   "FINAL_ANSWER",
				Params: map[string]interface{}{"input": strings.TrimSpace(response)},
			}
		}

		// Check for final answer
		if stepResult.Action == "FINAL_ANSWER" {
			execution.Steps = append(execution.Steps, stepResult)
			execution.Status = StatusCompleted
			execution.Result = extractFinalAnswer(stepResult.ToolCall)
			execution.EndedAt = time.Now()

			a.logger.Info("Agent execution completed",
				"id", execution.ID,
				"steps", len(execution.Steps),
			)
			return execution, nil
		}

		// Execute tool
		if stepResult.ToolCall != nil {
			tool, exists := a.tools[stepResult.ToolCall.Name]
			if !exists {
				stepResult.ToolResult = &ToolResult{
					Tool:  stepResult.ToolCall.Name,
					Error: fmt.Sprintf("Tool not found: %s", stepResult.ToolCall.Name),
				}
			} else {
				result, err := tool.Handler(ctx, stepResult.ToolCall.Params)
				stepResult.ToolResult = &ToolResult{
					Tool:   stepResult.ToolCall.Name,
					Result: result,
				}
				if err != nil {
					stepResult.ToolResult.Error = err.Error()
				}

				// Track tool usage
				found := false
				for _, used := range execution.ToolsUsed {
					if used == tool.Name {
						found = true
						break
					}
				}
				if !found {
					execution.ToolsUsed = append(execution.ToolsUsed, tool.Name)
				}
			}

			// Add observation to conversation
			observation := formatObservation(stepResult.ToolResult)
			messages = append(messages, Message{Role: "assistant", Content: response})

			// Count how many search tool calls we've done
			searchCount := 0
			for _, s := range execution.Steps {
				if s.ToolCall != nil && (s.ToolCall.Name == "web_search" || s.ToolCall.Name == "search_news") {
					searchCount++
				}
			}
			// Include current call if it's a search
			if stepResult.ToolCall.Name == "web_search" || stepResult.ToolCall.Name == "search_news" {
				searchCount++
			}

			// Add urgency hint after first search to encourage FINAL_ANSWER
			observationMsg := fmt.Sprintf("OBSERVATION: %s", observation)
			if searchCount >= 1 && (stepResult.ToolCall.Name == "web_search" || stepResult.ToolCall.Name == "search_news") {
				observationMsg += "\n\nHINWEIS: Du hast bereits Suchergebnisse erhalten. Fasse diese jetzt mit FINAL_ANSWER zusammen! Führe KEINE weitere Suche durch."
			}
			messages = append(messages, Message{Role: "user", Content: observationMsg})
		}

		execution.Steps = append(execution.Steps, stepResult)
	}

	// Max steps reached - try to provide a result based on collected observations
	execution.Status = StatusCompleted
	execution.Error = "max steps reached, result based on partial observations"
	execution.EndedAt = time.Now()
	execution.Result = a.synthesizePartialResult(execution.Steps)

	a.logger.Warn("Agent reached max steps without FINAL_ANSWER, synthesizing partial result",
		"id", execution.ID,
		"steps", len(execution.Steps),
		"tools_used", execution.ToolsUsed,
	)

	return execution, nil
}

// buildSystemPrompt builds the system prompt with tool descriptions
func (a *Agent) buildSystemPrompt() string {
	var toolDescs []string
	for _, tool := range a.tools {
		params, _ := json.Marshal(tool.Parameters)
		desc := fmt.Sprintf("- %s: %s\n  Parameters: %s", tool.Name, tool.Description, string(params))
		toolDescs = append(toolDescs, desc)
	}

	toolList := "Keine Tools verfügbar"
	if len(toolDescs) > 0 {
		toolList = strings.Join(toolDescs, "\n")
	}

	return strings.Replace(a.systemPrompt, "{{TOOLS}}", toolList, 1)
}

// parseResponse parses the LLM response into a Step
func (a *Agent) parseResponse(response string) Step {
	step := Step{}

	// Extract THOUGHT
	if idx := strings.Index(response, "THOUGHT:"); idx != -1 {
		end := strings.Index(response[idx:], "\nACTION:")
		if end == -1 {
			end = len(response) - idx
		}
		step.Thought = strings.TrimSpace(response[idx+8 : idx+end])
	}

	// Extract ACTION
	if idx := strings.Index(response, "ACTION:"); idx != -1 {
		end := strings.Index(response[idx:], "\nACTION_INPUT:")
		if end == -1 {
			end = len(response) - idx
		}
		step.Action = strings.TrimSpace(response[idx+7 : idx+end])
	}

	// Extract ACTION_INPUT
	if idx := strings.Index(response, "ACTION_INPUT:"); idx != -1 {
		inputStr := strings.TrimSpace(response[idx+13:])

		step.ToolCall = &ToolCall{
			Name:   step.Action,
			Params: make(map[string]interface{}),
		}

		// Try to parse as JSON
		var params map[string]interface{}
		if err := json.Unmarshal([]byte(inputStr), &params); err == nil {
			step.ToolCall.Params = params
		} else {
			// Treat as plain text input
			step.ToolCall.Params["input"] = inputStr
		}
	}

	return step
}

func formatObservation(result *ToolResult) string {
	if result == nil {
		return "No result"
	}
	if result.Error != "" {
		return fmt.Sprintf("Error: %s", result.Error)
	}
	if s, ok := result.Result.(string); ok {
		return s
	}
	data, _ := json.Marshal(result.Result)
	return string(data)
}

func extractFinalAnswer(call *ToolCall) string {
	if call == nil {
		return ""
	}
	if input, ok := call.Params["input"].(string); ok {
		return input
	}
	data, _ := json.Marshal(call.Params)
	return string(data)
}

// synthesizePartialResult creates a result from tool observations when max steps is reached
func (a *Agent) synthesizePartialResult(steps []Step) string {
	var sb strings.Builder
	sb.WriteString("Basierend auf der durchgeführten Recherche:\n\n")

	hasResults := false
	for _, step := range steps {
		if step.ToolResult != nil && step.ToolResult.Error == "" && step.ToolResult.Result != nil {
			// Include tool results
			resultStr := formatObservation(step.ToolResult)
			if len(resultStr) > 0 && resultStr != "No result" {
				hasResults = true
				sb.WriteString(resultStr)
				sb.WriteString("\n\n")
			}
		}
	}

	if !hasResults {
		return "Die Recherche konnte keine Ergebnisse liefern. Bitte versuchen Sie es mit einer anderen Suchanfrage."
	}

	return sb.String()
}
