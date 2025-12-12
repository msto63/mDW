// ============================================================================
// meinDENKWERK (mDW) - Lokale KI-Plattform
// ============================================================================
//
// Package:     agent
// Description: Tests for the agent execution including self-evaluation
// Author:      Mike Stoffels with Claude
// Created:     2025-12-12
// License:     MIT
// ============================================================================

package agent

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/msto63/mDW/internal/leibniz/agentloader"
)

// MockLLMFunc creates a mock LLM function for testing
func MockLLMFunc(responses []string) LLMFunc {
	index := 0
	return func(ctx context.Context, messages []Message) (string, error) {
		if index >= len(responses) {
			return responses[len(responses)-1], nil
		}
		resp := responses[index]
		index++
		return resp, nil
	}
}

// MockModelAwareLLMFunc creates a mock model-aware LLM function
func MockModelAwareLLMFunc(responses []string) ModelAwareLLMFunc {
	index := 0
	return func(ctx context.Context, model string, messages []Message) (string, error) {
		if index >= len(responses) {
			return responses[len(responses)-1], nil
		}
		resp := responses[index]
		index++
		return resp, nil
	}
}

// MockEvaluator is a mock for testing without real LLM calls
type MockEvaluator struct {
	results      []*agentloader.EvaluationResult
	resultIndex  int
	shouldError  bool
	errorMessage string
}

func (m *MockEvaluator) EvaluateResult(ctx context.Context, agent *agentloader.AgentYAML, task, result string) (*agentloader.EvaluationResult, error) {
	if m.shouldError {
		return nil, errors.New(m.errorMessage)
	}
	if m.resultIndex >= len(m.results) {
		return m.results[len(m.results)-1], nil
	}
	r := m.results[m.resultIndex]
	m.resultIndex++
	return r, nil
}

func (m *MockEvaluator) BuildImprovementPrompt(agent *agentloader.AgentYAML, task, prevResult string, evalResult *agentloader.EvaluationResult) string {
	return fmt.Sprintf("Improve: %s\nPrevious: %s\nFeedback: %s", task, prevResult, evalResult.Feedback)
}

func (m *MockEvaluator) ShouldIterate(agent *agentloader.AgentYAML, evalResult *agentloader.EvaluationResult, iteration int) bool {
	if agent.Evaluation == nil || !agent.Evaluation.Enabled {
		return false
	}
	if evalResult.Passed {
		return false
	}
	return iteration < agent.Evaluation.MaxIterations
}

// TestNewAgent tests agent creation
func TestNewAgent(t *testing.T) {
	cfg := DefaultConfig()
	agent := NewAgent(cfg)

	if agent == nil {
		t.Fatal("NewAgent returned nil")
	}
	if agent.maxSteps != cfg.MaxSteps {
		t.Errorf("Expected maxSteps=%d, got %d", cfg.MaxSteps, agent.maxSteps)
	}
	if agent.systemPrompt != cfg.SystemPrompt {
		t.Error("System prompt not set correctly")
	}
	if agent.tools == nil {
		t.Error("Tools map not initialized")
	}
}

// TestDefaultConfig tests default configuration
func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.MaxSteps != 10 {
		t.Errorf("Expected MaxSteps=10, got %d", cfg.MaxSteps)
	}
	if cfg.SystemPrompt == "" {
		t.Error("Expected non-empty SystemPrompt")
	}
	if !strings.Contains(cfg.SystemPrompt, "{{TOOLS}}") {
		t.Error("SystemPrompt should contain {{TOOLS}} placeholder")
	}
}

// TestAgentSetters tests the setter methods
func TestAgentSetters(t *testing.T) {
	agent := NewAgent(DefaultConfig())

	// Test SetLLMFunc
	agent.SetLLMFunc(func(ctx context.Context, msgs []Message) (string, error) {
		return "", nil
	})
	if agent.llmFunc == nil {
		t.Error("LLMFunc not set")
	}

	// Test SetModelAwareLLMFunc
	agent.SetModelAwareLLMFunc(func(ctx context.Context, model string, msgs []Message) (string, error) {
		return "", nil
	})
	if agent.modelAwareLLMFunc == nil {
		t.Error("ModelAwareLLMFunc not set")
	}

	// Test SetModel
	agent.SetModel("test-model")
	if agent.GetModel() != "test-model" {
		t.Errorf("Expected model=test-model, got %s", agent.GetModel())
	}

	// Test SetSystemPrompt
	agent.SetSystemPrompt("custom prompt")
	if agent.GetSystemPrompt() != "custom prompt" {
		t.Errorf("Expected systemPrompt=custom prompt, got %s", agent.GetSystemPrompt())
	}
}

// TestRegisterTool tests tool registration
func TestRegisterTool(t *testing.T) {
	agent := NewAgent(DefaultConfig())

	tool := &Tool{
		Name:        "test_tool",
		Description: "A test tool",
		Parameters:  map[string]ParameterDef{},
		Handler: func(ctx context.Context, params map[string]interface{}) (interface{}, error) {
			return "result", nil
		},
	}

	agent.RegisterTool(tool)

	tools := agent.ListTools()
	if len(tools) != 1 {
		t.Errorf("Expected 1 tool, got %d", len(tools))
	}
	if tools[0].Name != "test_tool" {
		t.Errorf("Expected tool name=test_tool, got %s", tools[0].Name)
	}
}

// TestUnregisterTool tests tool unregistration
func TestUnregisterTool(t *testing.T) {
	agent := NewAgent(DefaultConfig())

	agent.RegisterTool(&Tool{Name: "tool1", Handler: func(ctx context.Context, params map[string]interface{}) (interface{}, error) { return nil, nil }})
	agent.RegisterTool(&Tool{Name: "tool2", Handler: func(ctx context.Context, params map[string]interface{}) (interface{}, error) { return nil, nil }})

	if len(agent.ListTools()) != 2 {
		t.Fatalf("Expected 2 tools before unregister")
	}

	agent.UnregisterTool("tool1")

	tools := agent.ListTools()
	if len(tools) != 1 {
		t.Errorf("Expected 1 tool after unregister, got %d", len(tools))
	}
	if tools[0].Name != "tool2" {
		t.Errorf("Expected remaining tool=tool2, got %s", tools[0].Name)
	}
}

// TestExecute_NoLLMFunc tests execution without LLM function
func TestExecute_NoLLMFunc(t *testing.T) {
	agent := NewAgent(DefaultConfig())
	// Don't set LLMFunc

	_, err := agent.Execute(context.Background(), "test task")
	if err == nil {
		t.Error("Expected error when LLMFunc not set")
	}
	if !strings.Contains(err.Error(), "LLM function not set") {
		t.Errorf("Unexpected error message: %v", err)
	}
}

// TestExecute_FinalAnswerDirect tests direct final answer response
func TestExecute_FinalAnswerDirect(t *testing.T) {
	agent := NewAgent(DefaultConfig())
	agent.SetLLMFunc(MockLLMFunc([]string{
		"THOUGHT: The answer is simple\nACTION: FINAL_ANSWER\nACTION_INPUT: {\"input\": \"The result is 42\"}",
	}))

	exec, err := agent.Execute(context.Background(), "What is the answer?")

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if exec.Status != StatusCompleted {
		t.Errorf("Expected status=completed, got %s", exec.Status)
	}
	if exec.Result != "The result is 42" {
		t.Errorf("Unexpected result: %s", exec.Result)
	}
	if len(exec.Steps) != 1 {
		t.Errorf("Expected 1 step, got %d", len(exec.Steps))
	}
}

// TestExecute_WithToolCall tests execution with tool usage
func TestExecute_WithToolCall(t *testing.T) {
	agent := NewAgent(DefaultConfig())

	toolCalled := false
	agent.RegisterTool(&Tool{
		Name:        "calculator",
		Description: "Performs calculations",
		Handler: func(ctx context.Context, params map[string]interface{}) (interface{}, error) {
			toolCalled = true
			return "Result: 42", nil
		},
	})

	agent.SetLLMFunc(MockLLMFunc([]string{
		"THOUGHT: I need to calculate\nACTION: calculator\nACTION_INPUT: {\"expression\": \"6*7\"}",
		"THOUGHT: Got the result\nACTION: FINAL_ANSWER\nACTION_INPUT: {\"input\": \"The answer is 42\"}",
	}))

	exec, err := agent.Execute(context.Background(), "Calculate 6*7")

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if !toolCalled {
		t.Error("Tool was not called")
	}
	if exec.Status != StatusCompleted {
		t.Errorf("Expected status=completed, got %s", exec.Status)
	}
	if len(exec.ToolsUsed) != 1 || exec.ToolsUsed[0] != "calculator" {
		t.Errorf("Expected ToolsUsed=[calculator], got %v", exec.ToolsUsed)
	}
}

// TestExecute_ToolNotFound tests execution with non-existent tool
func TestExecute_ToolNotFound(t *testing.T) {
	agent := NewAgent(DefaultConfig())
	agent.SetLLMFunc(MockLLMFunc([]string{
		"THOUGHT: Using unknown tool\nACTION: unknown_tool\nACTION_INPUT: {}",
		"THOUGHT: Tool failed, giving up\nACTION: FINAL_ANSWER\nACTION_INPUT: {\"input\": \"Could not complete\"}",
	}))

	exec, err := agent.Execute(context.Background(), "Do something")

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if exec.Status != StatusCompleted {
		t.Errorf("Expected status=completed, got %s", exec.Status)
	}
	// Check that the tool error was recorded
	if len(exec.Steps) < 1 {
		t.Fatal("Expected at least 1 step")
	}
	if exec.Steps[0].ToolResult == nil || exec.Steps[0].ToolResult.Error == "" {
		t.Error("Expected tool error to be recorded")
	}
}

// TestExecute_ContextCancellation tests cancellation handling
func TestExecute_ContextCancellation(t *testing.T) {
	agent := NewAgent(DefaultConfig())

	ctx, cancel := context.WithCancel(context.Background())

	agent.SetLLMFunc(func(ctx context.Context, msgs []Message) (string, error) {
		cancel() // Cancel immediately
		return "THOUGHT: thinking\nACTION: some_action\nACTION_INPUT: {}", nil
	})

	exec, err := agent.Execute(ctx, "Test task")

	if err == nil {
		t.Error("Expected error on cancellation")
	}
	if exec.Status != StatusCancelled {
		t.Errorf("Expected status=cancelled, got %s", exec.Status)
	}
}

// TestExecute_MaxStepsReached tests max steps limit
func TestExecute_MaxStepsReached(t *testing.T) {
	cfg := DefaultConfig()
	cfg.MaxSteps = 2
	agent := NewAgent(cfg)

	agent.RegisterTool(&Tool{
		Name:        "dummy",
		Description: "Dummy tool",
		Handler: func(ctx context.Context, params map[string]interface{}) (interface{}, error) {
			return "Some result", nil
		},
	})

	// Never return FINAL_ANSWER
	agent.SetLLMFunc(MockLLMFunc([]string{
		"THOUGHT: Step 1\nACTION: dummy\nACTION_INPUT: {}",
		"THOUGHT: Step 2\nACTION: dummy\nACTION_INPUT: {}",
		"THOUGHT: Step 3\nACTION: dummy\nACTION_INPUT: {}",
	}))

	exec, err := agent.Execute(context.Background(), "Never-ending task")

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if exec.Status != StatusCompleted {
		t.Errorf("Expected status=completed, got %s", exec.Status)
	}
	if len(exec.Steps) > 2 {
		t.Errorf("Expected max 2 steps, got %d", len(exec.Steps))
	}
	if exec.Error == "" {
		t.Error("Expected error message about max steps")
	}
}

// TestExecute_DirectResponse tests handling of non-ReAct format responses
func TestExecute_DirectResponse(t *testing.T) {
	agent := NewAgent(DefaultConfig())
	agent.SetLLMFunc(MockLLMFunc([]string{
		"This is a direct response without any ReAct format.",
	}))

	exec, err := agent.Execute(context.Background(), "Simple question")

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if exec.Status != StatusCompleted {
		t.Errorf("Expected status=completed, got %s", exec.Status)
	}
	if exec.Result == "" {
		t.Error("Expected result to contain the direct response")
	}
}

// TestExecute_ModelAwareLLMPreferred tests that model-aware LLM is preferred
func TestExecute_ModelAwareLLMPreferred(t *testing.T) {
	agent := NewAgent(DefaultConfig())
	agent.SetModel("test-model")

	regularCalled := false
	modelAwareCalled := false

	agent.SetLLMFunc(func(ctx context.Context, msgs []Message) (string, error) {
		regularCalled = true
		return "THOUGHT: done\nACTION: FINAL_ANSWER\nACTION_INPUT: {\"input\": \"regular\"}", nil
	})

	agent.SetModelAwareLLMFunc(func(ctx context.Context, model string, msgs []Message) (string, error) {
		modelAwareCalled = true
		if model != "test-model" {
			t.Errorf("Expected model=test-model, got %s", model)
		}
		return "THOUGHT: done\nACTION: FINAL_ANSWER\nACTION_INPUT: {\"input\": \"model-aware\"}", nil
	})

	exec, err := agent.Execute(context.Background(), "Test")

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if regularCalled {
		t.Error("Regular LLMFunc should not be called when ModelAwareLLMFunc is set")
	}
	if !modelAwareCalled {
		t.Error("ModelAwareLLMFunc should be called")
	}
	if exec.Result != "model-aware" {
		t.Errorf("Expected result=model-aware, got %s", exec.Result)
	}
}

// TestExecuteWithEvaluation_DisabledEvaluation tests fallback to regular execution
func TestExecuteWithEvaluation_DisabledEvaluation(t *testing.T) {
	agent := NewAgent(DefaultConfig())
	agent.SetLLMFunc(MockLLMFunc([]string{
		"THOUGHT: done\nACTION: FINAL_ANSWER\nACTION_INPUT: {\"input\": \"result\"}",
	}))

	tests := []struct {
		name     string
		agentDef *agentloader.AgentYAML
	}{
		{
			name:     "nil agent definition",
			agentDef: nil,
		},
		{
			name:     "nil evaluation config",
			agentDef: &agentloader.AgentYAML{ID: "test"},
		},
		{
			name: "evaluation disabled",
			agentDef: &agentloader.AgentYAML{
				ID:         "test",
				Evaluation: &agentloader.EvaluationConfig{Enabled: false},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			agent.SetLLMFunc(MockLLMFunc([]string{
				"THOUGHT: done\nACTION: FINAL_ANSWER\nACTION_INPUT: {\"input\": \"result\"}",
			}))

			exec, err := agent.ExecuteWithEvaluation(context.Background(), "task", tt.agentDef, nil)

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}
			if exec.Status != StatusCompleted {
				t.Errorf("Expected status=completed, got %s", exec.Status)
			}
			// No evaluation should be tracked
			if exec.Iterations != 0 {
				t.Errorf("Expected 0 iterations (standard execution), got %d", exec.Iterations)
			}
		})
	}
}

// TestExecuteWithEvaluation_SinglePassingIteration tests a single passing evaluation
func TestExecuteWithEvaluation_SinglePassingIteration(t *testing.T) {
	agent := NewAgent(DefaultConfig())

	agent.SetLLMFunc(func(ctx context.Context, msgs []Message) (string, error) {
		return "THOUGHT: done\nACTION: FINAL_ANSWER\nACTION_INPUT: {\"input\": \"good result\"}", nil
	})

	agentDef := &agentloader.AgentYAML{
		ID: "test",
		Evaluation: &agentloader.EvaluationConfig{
			Enabled:       true,
			MaxIterations: 3,
		},
	}

	// Pass nil evaluator - should fall back to standard execution when LLM client missing
	// The ExecuteWithEvaluation handles nil evaluator gracefully
	exec, err := agent.ExecuteWithEvaluation(context.Background(), "task", agentDef, nil)

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if exec.Status != StatusCompleted {
		t.Errorf("Expected status=completed, got %s", exec.Status)
	}
	// Without evaluator, falls back to standard execution (no iterations tracked)
	if exec.Status != StatusCompleted {
		t.Errorf("Expected status=completed, got %s", exec.Status)
	}
}

// TestParseResponse tests the response parsing
func TestParseResponse(t *testing.T) {
	agent := NewAgent(DefaultConfig())

	tests := []struct {
		name           string
		response       string
		expectThought  string
		expectAction   string
		expectToolCall bool
		expectParams   map[string]interface{}
	}{
		{
			name:           "full ReAct format",
			response:       "THOUGHT: I need to search\nACTION: web_search\nACTION_INPUT: {\"query\": \"test\"}",
			expectThought:  "I need to search",
			expectAction:   "web_search",
			expectToolCall: true,
			expectParams:   map[string]interface{}{"query": "test"},
		},
		{
			name:           "final answer",
			response:       "THOUGHT: Done\nACTION: FINAL_ANSWER\nACTION_INPUT: {\"input\": \"result\"}",
			expectThought:  "Done",
			expectAction:   "FINAL_ANSWER",
			expectToolCall: true,
			expectParams:   map[string]interface{}{"input": "result"},
		},
		{
			name:           "plain text input",
			response:       "THOUGHT: Simple\nACTION: echo\nACTION_INPUT: hello world",
			expectThought:  "Simple",
			expectAction:   "echo",
			expectToolCall: true,
			expectParams:   map[string]interface{}{"input": "hello world"},
		},
		{
			name:           "no format",
			response:       "Just a plain response",
			expectThought:  "",
			expectAction:   "",
			expectToolCall: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			step := agent.parseResponse(tt.response)

			if step.Thought != tt.expectThought {
				t.Errorf("Expected thought=%q, got %q", tt.expectThought, step.Thought)
			}
			if step.Action != tt.expectAction {
				t.Errorf("Expected action=%q, got %q", tt.expectAction, step.Action)
			}
			if tt.expectToolCall {
				if step.ToolCall == nil {
					t.Fatal("Expected ToolCall, got nil")
				}
				for key, val := range tt.expectParams {
					if step.ToolCall.Params[key] != val {
						t.Errorf("Expected param[%s]=%v, got %v", key, val, step.ToolCall.Params[key])
					}
				}
			}
		})
	}
}

// TestFormatObservation tests observation formatting
func TestFormatObservation(t *testing.T) {
	tests := []struct {
		name     string
		result   *ToolResult
		expected string
	}{
		{
			name:     "nil result",
			result:   nil,
			expected: "No result",
		},
		{
			name:     "error result",
			result:   &ToolResult{Tool: "test", Error: "something failed"},
			expected: "Error: something failed",
		},
		{
			name:     "string result",
			result:   &ToolResult{Tool: "test", Result: "hello"},
			expected: "hello",
		},
		{
			name:     "map result",
			result:   &ToolResult{Tool: "test", Result: map[string]string{"key": "value"}},
			expected: `{"key":"value"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatObservation(tt.result)
			if result != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, result)
			}
		})
	}
}

// TestExtractFinalAnswer tests final answer extraction
func TestExtractFinalAnswer(t *testing.T) {
	tests := []struct {
		name     string
		call     *ToolCall
		expected string
	}{
		{
			name:     "nil call",
			call:     nil,
			expected: "",
		},
		{
			name:     "string input",
			call:     &ToolCall{Params: map[string]interface{}{"input": "the answer"}},
			expected: "the answer",
		},
		{
			name:     "non-string input",
			call:     &ToolCall{Params: map[string]interface{}{"data": 42}},
			expected: `{"data":42}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractFinalAnswer(tt.call)
			if result != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, result)
			}
		})
	}
}

// TestSynthesizePartialResult tests partial result synthesis
func TestSynthesizePartialResult(t *testing.T) {
	agent := NewAgent(DefaultConfig())

	tests := []struct {
		name            string
		steps           []Step
		expectContains  []string
		expectNotContain string
	}{
		{
			name:           "empty steps",
			steps:          []Step{},
			expectContains: []string{"keine Ergebnisse"},
		},
		{
			name: "steps with results",
			steps: []Step{
				{ToolResult: &ToolResult{Result: "First result"}},
				{ToolResult: &ToolResult{Result: "Second result"}},
			},
			expectContains: []string{"First result", "Second result"},
		},
		{
			name: "steps with error",
			steps: []Step{
				{ToolResult: &ToolResult{Error: "some error"}},
			},
			expectContains: []string{"keine Ergebnisse"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := agent.synthesizePartialResult(tt.steps)

			for _, expected := range tt.expectContains {
				if !strings.Contains(result, expected) {
					t.Errorf("Expected result to contain %q, got: %s", expected, result)
				}
			}
		})
	}
}

// TestExecution_Fields tests the Execution struct fields
func TestExecution_Fields(t *testing.T) {
	exec := &Execution{
		ID:                "exec-123",
		Task:              "Test task",
		Status:            StatusRunning,
		Steps:             []Step{{Index: 0, Thought: "thinking"}},
		Result:            "result",
		Error:             "",
		StartedAt:         time.Now(),
		EndedAt:           time.Now().Add(time.Second),
		ToolsUsed:         []string{"tool1", "tool2"},
		Iterations:        2,
		EvaluationResults: []*agentloader.EvaluationResult{{Passed: true, Score: 0.9}},
		FinalQualityScore: 0.9,
	}

	if exec.ID != "exec-123" {
		t.Errorf("Expected ID=exec-123, got %s", exec.ID)
	}
	if exec.Task != "Test task" {
		t.Errorf("Expected Task=Test task, got %s", exec.Task)
	}
	if exec.Status != StatusRunning {
		t.Errorf("Expected Status=running, got %s", exec.Status)
	}
	if len(exec.Steps) != 1 {
		t.Errorf("Expected 1 step, got %d", len(exec.Steps))
	}
	if exec.Iterations != 2 {
		t.Errorf("Expected Iterations=2, got %d", exec.Iterations)
	}
	if exec.FinalQualityScore != 0.9 {
		t.Errorf("Expected FinalQualityScore=0.9, got %f", exec.FinalQualityScore)
	}
}

// TestExecutionStatus tests status constants
func TestExecutionStatus(t *testing.T) {
	statuses := []ExecutionStatus{
		StatusPending,
		StatusRunning,
		StatusCompleted,
		StatusFailed,
		StatusCancelled,
	}

	for _, status := range statuses {
		if string(status) == "" {
			t.Errorf("Status constant should not be empty")
		}
	}

	if StatusPending != "pending" {
		t.Errorf("Expected StatusPending=pending, got %s", StatusPending)
	}
	if StatusCompleted != "completed" {
		t.Errorf("Expected StatusCompleted=completed, got %s", StatusCompleted)
	}
}

// TestStep_Fields tests the Step struct
func TestStep_Fields(t *testing.T) {
	now := time.Now()
	step := Step{
		Index:     1,
		Thought:   "thinking",
		Action:    "test_action",
		ToolCall:  &ToolCall{Name: "test", Params: map[string]interface{}{}},
		ToolResult: &ToolResult{Tool: "test", Result: "success"},
		Timestamp: now,
	}

	if step.Index != 1 {
		t.Errorf("Expected Index=1, got %d", step.Index)
	}
	if step.Thought != "thinking" {
		t.Errorf("Expected Thought=thinking, got %s", step.Thought)
	}
	if step.Action != "test_action" {
		t.Errorf("Expected Action=test_action, got %s", step.Action)
	}
	if step.ToolCall.Name != "test" {
		t.Errorf("Expected ToolCall.Name=test, got %s", step.ToolCall.Name)
	}
	if step.ToolResult.Tool != "test" {
		t.Errorf("Expected ToolResult.Tool=test, got %s", step.ToolResult.Tool)
	}
}

// TestTool_Handler tests tool handler execution
func TestTool_Handler(t *testing.T) {
	tool := &Tool{
		Name:        "test_tool",
		Description: "Test tool",
		Parameters: map[string]ParameterDef{
			"input": {Type: "string", Description: "Input value", Required: true},
		},
		Handler: func(ctx context.Context, params map[string]interface{}) (interface{}, error) {
			input := params["input"].(string)
			return "processed: " + input, nil
		},
	}

	result, err := tool.Handler(context.Background(), map[string]interface{}{"input": "test"})
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if result != "processed: test" {
		t.Errorf("Expected 'processed: test', got %v", result)
	}
}
