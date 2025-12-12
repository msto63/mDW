// ============================================================================
// meinDENKWERK (mDW) - Lokale KI-Plattform
// ============================================================================
//
// Package:     service
// Description: Tests for the Leibniz service including evaluation features
// Author:      Mike Stoffels with Claude
// Created:     2025-12-12
// License:     MIT
// ============================================================================

package service

import (
	"context"
	"testing"
	"time"
)

// TestDefaultConfig tests the default configuration
func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.MaxSteps != 10 {
		t.Errorf("Expected MaxSteps=10, got %d", cfg.MaxSteps)
	}
	if cfg.EnablePersistence != true {
		t.Error("Expected EnablePersistence=true")
	}
	if cfg.EnableBuiltinTools != true {
		t.Error("Expected EnableBuiltinTools=true")
	}
	if cfg.EnableWebResearchAgent != true {
		t.Error("Expected EnableWebResearchAgent=true")
	}
	if cfg.EnablePlaton != true {
		t.Error("Expected EnablePlaton=true")
	}
	if cfg.PlatonHost != "localhost" {
		t.Errorf("Expected PlatonHost=localhost, got %s", cfg.PlatonHost)
	}
	if cfg.PlatonPort != 9130 {
		t.Errorf("Expected PlatonPort=9130, got %d", cfg.PlatonPort)
	}
	if cfg.AgentsDir != "./configs/agents" {
		t.Errorf("Expected AgentsDir=./configs/agents, got %s", cfg.AgentsDir)
	}
	if cfg.EnableHotReload != true {
		t.Error("Expected EnableHotReload=true")
	}
}

// TestEvaluationOptions tests the EvaluationOptions struct
func TestEvaluationOptions(t *testing.T) {
	tests := []struct {
		name string
		opts EvaluationOptions
	}{
		{
			name: "skip evaluation",
			opts: EvaluationOptions{SkipEvaluation: true, MaxIterations: 0},
		},
		{
			name: "custom max iterations",
			opts: EvaluationOptions{SkipEvaluation: false, MaxIterations: 5},
		},
		{
			name: "default values",
			opts: EvaluationOptions{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Just verify the struct can be created and accessed
			_ = tt.opts.SkipEvaluation
			_ = tt.opts.MaxIterations
		})
	}
}

// TestEvaluationInfo tests the EvaluationInfo struct
func TestEvaluationInfo(t *testing.T) {
	info := EvaluationInfo{
		Performed:      true,
		IterationsUsed: 2,
		FinalScore:     0.85,
		Passed:         true,
		Feedback:       "Good result",
		Criteria: []CriterionResultInfo{
			{Name: "Quality", Passed: true, Required: true, Feedback: "Excellent"},
			{Name: "Format", Passed: true, Required: false, Feedback: "Good format"},
		},
	}

	if !info.Performed {
		t.Error("Expected Performed=true")
	}
	if info.IterationsUsed != 2 {
		t.Errorf("Expected IterationsUsed=2, got %d", info.IterationsUsed)
	}
	if info.FinalScore != 0.85 {
		t.Errorf("Expected FinalScore=0.85, got %f", info.FinalScore)
	}
	if !info.Passed {
		t.Error("Expected Passed=true")
	}
	if info.Feedback != "Good result" {
		t.Errorf("Expected Feedback='Good result', got %s", info.Feedback)
	}
	if len(info.Criteria) != 2 {
		t.Errorf("Expected 2 criteria, got %d", len(info.Criteria))
	}
}

// TestCriterionResultInfo tests the CriterionResultInfo struct
func TestCriterionResultInfo(t *testing.T) {
	cri := CriterionResultInfo{
		Name:     "Accuracy",
		Passed:   true,
		Required: true,
		Feedback: "All facts verified",
	}

	if cri.Name != "Accuracy" {
		t.Errorf("Expected Name=Accuracy, got %s", cri.Name)
	}
	if !cri.Passed {
		t.Error("Expected Passed=true")
	}
	if !cri.Required {
		t.Error("Expected Required=true")
	}
	if cri.Feedback != "All facts verified" {
		t.Errorf("Expected Feedback='All facts verified', got %s", cri.Feedback)
	}
}

// TestExecuteResponse tests the ExecuteResponse struct
func TestExecuteResponse(t *testing.T) {
	resp := ExecuteResponse{
		ID:        "exec-123",
		Status:    "completed",
		Result:    "Task completed successfully",
		Steps:     []StepInfo{{Index: 0, Thought: "thinking"}},
		ToolsUsed: []string{"calculator"},
		Duration:  5 * time.Second,
		Error:     "",
		Evaluation: &EvaluationInfo{
			Performed:      true,
			IterationsUsed: 1,
			FinalScore:     0.9,
			Passed:         true,
		},
	}

	if resp.ID != "exec-123" {
		t.Errorf("Expected ID=exec-123, got %s", resp.ID)
	}
	if resp.Status != "completed" {
		t.Errorf("Expected Status=completed, got %s", resp.Status)
	}
	if resp.Evaluation == nil {
		t.Fatal("Expected Evaluation to be set")
	}
	if !resp.Evaluation.Performed {
		t.Error("Expected Evaluation.Performed=true")
	}
	if resp.Evaluation.FinalScore != 0.9 {
		t.Errorf("Expected Evaluation.FinalScore=0.9, got %f", resp.Evaluation.FinalScore)
	}
}

// TestExecuteResponse_NoEvaluation tests ExecuteResponse without evaluation
func TestExecuteResponse_NoEvaluation(t *testing.T) {
	resp := ExecuteResponse{
		ID:         "exec-456",
		Status:     "completed",
		Result:     "Done",
		Evaluation: nil, // No evaluation performed
	}

	if resp.Evaluation != nil {
		t.Error("Expected Evaluation to be nil")
	}
}

// TestStepInfo tests the StepInfo struct
func TestStepInfo(t *testing.T) {
	now := time.Now()
	step := StepInfo{
		Index:      0,
		Thought:    "I need to calculate",
		Action:     "calculator",
		ToolName:   "calculator",
		ToolInput:  `{"expression": "2+2"}`,
		ToolOutput: "4",
		Timestamp:  now,
	}

	if step.Index != 0 {
		t.Errorf("Expected Index=0, got %d", step.Index)
	}
	if step.Thought != "I need to calculate" {
		t.Errorf("Expected Thought='I need to calculate', got %s", step.Thought)
	}
	if step.ToolName != "calculator" {
		t.Errorf("Expected ToolName=calculator, got %s", step.ToolName)
	}
	if step.ToolOutput != "4" {
		t.Errorf("Expected ToolOutput='4', got %s", step.ToolOutput)
	}
}

// TestExecuteRequest tests the ExecuteRequest struct
func TestExecuteRequest(t *testing.T) {
	req := ExecuteRequest{
		Task:     "Calculate 2+2",
		Tools:    []string{"calculator"},
		MaxSteps: 5,
		Timeout:  30 * time.Second,
		Context:  map[string]string{"user": "test"},
	}

	if req.Task != "Calculate 2+2" {
		t.Errorf("Expected Task='Calculate 2+2', got %s", req.Task)
	}
	if len(req.Tools) != 1 || req.Tools[0] != "calculator" {
		t.Errorf("Expected Tools=['calculator'], got %v", req.Tools)
	}
	if req.MaxSteps != 5 {
		t.Errorf("Expected MaxSteps=5, got %d", req.MaxSteps)
	}
	if req.Timeout != 30*time.Second {
		t.Errorf("Expected Timeout=30s, got %v", req.Timeout)
	}
}

// TestToolInfo tests the ToolInfo struct
func TestToolInfo(t *testing.T) {
	tool := ToolInfo{
		Name:        "web_search",
		Description: "Search the web",
		Source:      "builtin",
	}

	if tool.Name != "web_search" {
		t.Errorf("Expected Name=web_search, got %s", tool.Name)
	}
	if tool.Description != "Search the web" {
		t.Errorf("Expected Description='Search the web', got %s", tool.Description)
	}
	if tool.Source != "builtin" {
		t.Errorf("Expected Source=builtin, got %s", tool.Source)
	}
}

// TestAgentDefinition tests the AgentDefinition struct
func TestAgentDefinition(t *testing.T) {
	now := time.Now()
	agent := AgentDefinition{
		ID:           "test-agent",
		Name:         "Test Agent",
		Description:  "A test agent",
		SystemPrompt: "You are a test agent",
		Tools:        []string{"calculator", "web_search"},
		Model:        "mistral:7b",
		MaxSteps:     10,
		Timeout:      60 * time.Second,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if agent.ID != "test-agent" {
		t.Errorf("Expected ID=test-agent, got %s", agent.ID)
	}
	if agent.Name != "Test Agent" {
		t.Errorf("Expected Name='Test Agent', got %s", agent.Name)
	}
	if agent.Model != "mistral:7b" {
		t.Errorf("Expected Model=mistral:7b, got %s", agent.Model)
	}
	if len(agent.Tools) != 2 {
		t.Errorf("Expected 2 tools, got %d", len(agent.Tools))
	}
}

// TestExecutionRecord tests the ExecutionRecord struct
func TestExecutionRecord(t *testing.T) {
	now := time.Now()
	_, cancel := context.WithCancel(context.Background())

	record := ExecutionRecord{
		ID:          "exec-789",
		AgentID:     "test-agent",
		Message:     "Do something",
		Status:      "running",
		Result:      "",
		Error:       "",
		Steps:       []StepInfo{},
		ToolsUsed:   []string{},
		StartedAt:   now,
		CompletedAt: time.Time{},
		Duration:    0,
		Cancel:      cancel,
	}

	if record.ID != "exec-789" {
		t.Errorf("Expected ID=exec-789, got %s", record.ID)
	}
	if record.AgentID != "test-agent" {
		t.Errorf("Expected AgentID=test-agent, got %s", record.AgentID)
	}
	if record.Status != "running" {
		t.Errorf("Expected Status=running, got %s", record.Status)
	}
	if record.Cancel == nil {
		t.Error("Expected Cancel function to be set")
	}

	// Clean up
	cancel()
}

// TestCustomTool tests the CustomTool struct
func TestCustomTool(t *testing.T) {
	tool := CustomTool{
		Name:                 "custom_tool",
		Description:          "A custom tool",
		ParameterSchema:      `{"type": "object", "properties": {"input": {"type": "string"}}}`,
		RequiresConfirmation: true,
	}

	if tool.Name != "custom_tool" {
		t.Errorf("Expected Name=custom_tool, got %s", tool.Name)
	}
	if tool.Description != "A custom tool" {
		t.Errorf("Expected Description='A custom tool', got %s", tool.Description)
	}
	if !tool.RequiresConfirmation {
		t.Error("Expected RequiresConfirmation=true")
	}
}

// TestMCPServerConfig tests the MCPServerConfig struct
func TestMCPServerConfig(t *testing.T) {
	cfg := MCPServerConfig{
		Name:    "test-server",
		Command: "npx",
		Args:    []string{"-y", "@test/mcp-server"},
		Env:     map[string]string{"API_KEY": "secret"},
	}

	if cfg.Name != "test-server" {
		t.Errorf("Expected Name=test-server, got %s", cfg.Name)
	}
	if cfg.Command != "npx" {
		t.Errorf("Expected Command=npx, got %s", cfg.Command)
	}
	if len(cfg.Args) != 2 {
		t.Errorf("Expected 2 args, got %d", len(cfg.Args))
	}
	if cfg.Env["API_KEY"] != "secret" {
		t.Errorf("Expected Env[API_KEY]=secret, got %s", cfg.Env["API_KEY"])
	}
}

// TestAgentMatch tests the AgentMatch struct
func TestAgentMatch(t *testing.T) {
	match := AgentMatch{
		AgentID:    "web-researcher",
		AgentName:  "Web Researcher",
		Similarity: 0.95,
	}

	if match.AgentID != "web-researcher" {
		t.Errorf("Expected AgentID=web-researcher, got %s", match.AgentID)
	}
	if match.AgentName != "Web Researcher" {
		t.Errorf("Expected AgentName='Web Researcher', got %s", match.AgentName)
	}
	if match.Similarity != 0.95 {
		t.Errorf("Expected Similarity=0.95, got %f", match.Similarity)
	}
}

// TestAgentInfo tests the AgentInfo struct
func TestAgentInfo(t *testing.T) {
	info := AgentInfo{
		ID:          "summarizer",
		Name:        "Summarizer",
		Description: "Summarizes text",
	}

	if info.ID != "summarizer" {
		t.Errorf("Expected ID=summarizer, got %s", info.ID)
	}
	if info.Name != "Summarizer" {
		t.Errorf("Expected Name='Summarizer', got %s", info.Name)
	}
	if info.Description != "Summarizes text" {
		t.Errorf("Expected Description='Summarizes text', got %s", info.Description)
	}
}

// TestGetString tests the getString helper function
func TestGetString(t *testing.T) {
	tests := []struct {
		name     string
		m        map[string]interface{}
		key      string
		expected string
	}{
		{
			name:     "existing string key",
			m:        map[string]interface{}{"key": "value"},
			key:      "key",
			expected: "value",
		},
		{
			name:     "missing key",
			m:        map[string]interface{}{"other": "value"},
			key:      "key",
			expected: "",
		},
		{
			name:     "non-string value",
			m:        map[string]interface{}{"key": 123},
			key:      "key",
			expected: "",
		},
		{
			name:     "nil map value",
			m:        map[string]interface{}{"key": nil},
			key:      "key",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getString(tt.m, tt.key)
			if result != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, result)
			}
		})
	}
}

// TestToStoreAgent tests the toStoreAgent conversion
func TestToStoreAgent(t *testing.T) {
	now := time.Now()
	agent := &AgentDefinition{
		ID:           "test",
		Name:         "Test",
		Description:  "Test agent",
		SystemPrompt: "You are a test",
		Tools:        []string{"tool1", "tool2"},
		Model:        "mistral:7b",
		MaxSteps:     10,
		Timeout:      60 * time.Second,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	storeAgent := toStoreAgent(agent)

	if storeAgent.ID != agent.ID {
		t.Errorf("Expected ID=%s, got %s", agent.ID, storeAgent.ID)
	}
	if storeAgent.Name != agent.Name {
		t.Errorf("Expected Name=%s, got %s", agent.Name, storeAgent.Name)
	}
	if storeAgent.Model != agent.Model {
		t.Errorf("Expected Model=%s, got %s", agent.Model, storeAgent.Model)
	}
	if len(storeAgent.Tools) != len(agent.Tools) {
		t.Errorf("Expected %d tools, got %d", len(agent.Tools), len(storeAgent.Tools))
	}
}

// TestToStoreExecution tests the toStoreExecution conversion
func TestToStoreExecution(t *testing.T) {
	now := time.Now()
	record := &ExecutionRecord{
		ID:        "exec-123",
		AgentID:   "test-agent",
		Message:   "Do something",
		Status:    "completed",
		Result:    "Done",
		Error:     "",
		Steps: []StepInfo{
			{Index: 0, Thought: "thinking", Action: "test"},
		},
		ToolsUsed:   []string{"tool1"},
		StartedAt:   now,
		CompletedAt: now.Add(5 * time.Second),
		Duration:    5 * time.Second,
	}

	storeExec := toStoreExecution(record)

	if storeExec.ID != record.ID {
		t.Errorf("Expected ID=%s, got %s", record.ID, storeExec.ID)
	}
	if storeExec.AgentID != record.AgentID {
		t.Errorf("Expected AgentID=%s, got %s", record.AgentID, storeExec.AgentID)
	}
	if storeExec.Status != record.Status {
		t.Errorf("Expected Status=%s, got %s", record.Status, storeExec.Status)
	}
	if len(storeExec.Steps) != len(record.Steps) {
		t.Errorf("Expected %d steps, got %d", len(record.Steps), len(storeExec.Steps))
	}
	if storeExec.Duration != record.Duration.Milliseconds() {
		t.Errorf("Expected Duration=%d, got %d", record.Duration.Milliseconds(), storeExec.Duration)
	}
}

// TestEvaluationOptions_Defaults tests default behavior of EvaluationOptions
func TestEvaluationOptions_Defaults(t *testing.T) {
	// Empty options should not skip evaluation and use agent defaults
	opts := EvaluationOptions{}

	if opts.SkipEvaluation {
		t.Error("Default SkipEvaluation should be false")
	}
	if opts.MaxIterations != 0 {
		t.Errorf("Default MaxIterations should be 0 (use agent default), got %d", opts.MaxIterations)
	}
}

// TestEvaluationOptions_SkipEvaluation tests skip evaluation option
func TestEvaluationOptions_SkipEvaluation(t *testing.T) {
	opts := EvaluationOptions{
		SkipEvaluation: true,
	}

	if !opts.SkipEvaluation {
		t.Error("Expected SkipEvaluation=true")
	}
}

// TestEvaluationOptions_CustomMaxIterations tests custom max iterations
func TestEvaluationOptions_CustomMaxIterations(t *testing.T) {
	opts := EvaluationOptions{
		MaxIterations: 5,
	}

	if opts.MaxIterations != 5 {
		t.Errorf("Expected MaxIterations=5, got %d", opts.MaxIterations)
	}
}

// TestConfig_EvaluationRelated tests configuration fields related to evaluation
func TestConfig_EvaluationRelated(t *testing.T) {
	cfg := Config{
		MaxSteps:        10,
		AgentsDir:       "./configs/agents",
		EnableHotReload: true,
	}

	// Verify that evaluation-related configuration works
	if cfg.AgentsDir != "./configs/agents" {
		t.Errorf("Expected AgentsDir='./configs/agents', got %s", cfg.AgentsDir)
	}
	if !cfg.EnableHotReload {
		t.Error("Expected EnableHotReload=true")
	}
}
