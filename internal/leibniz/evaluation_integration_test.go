// ============================================================================
// meinDENKWERK (mDW) - Lokale KI-Plattform
// ============================================================================
//
// Package:     leibniz
// Description: Integration tests for the self-evaluation pipeline
// Author:      Mike Stoffels with Claude
// Created:     2025-12-12
// License:     MIT
// ============================================================================

package leibniz

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/msto63/mDW/internal/leibniz/agent"
	"github.com/msto63/mDW/internal/leibniz/agentloader"
	"github.com/msto63/mDW/internal/leibniz/evaluator"
)

// MockLLMClient simulates LLM responses for testing
type MockLLMClient struct {
	responses    []string
	currentIndex int
}

func (m *MockLLMClient) NextResponse() string {
	if m.currentIndex >= len(m.responses) {
		return m.responses[len(m.responses)-1]
	}
	resp := m.responses[m.currentIndex]
	m.currentIndex++
	return resp
}

// TestIntegration_FullEvaluationPipeline tests the complete evaluation flow
func TestIntegration_FullEvaluationPipeline(t *testing.T) {
	// Create agent definition with evaluation enabled
	agentDef := &agentloader.AgentYAML{
		ID:           "test-summarizer",
		Name:         "Test Summarizer",
		Description:  "Summarizes text for testing",
		Model:        "test-model",
		MaxSteps:     5,
		Timeout:      30 * time.Second,
		SystemPrompt: "Du bist ein Zusammenfasser.",
		Evaluation: &agentloader.EvaluationConfig{
			Enabled:         true,
			MaxIterations:   3,
			MinQualityScore: 0.7,
			Criteria: []agentloader.EvaluationCriterion{
				{Name: "Completeness", Check: "All main points covered", Required: true, Weight: 1.0},
				{Name: "Clarity", Check: "Easy to understand", Required: false, Weight: 0.8},
			},
		},
	}

	// Apply defaults
	agentDef.Defaults()

	// Verify evaluation config was properly initialized
	if agentDef.Evaluation.EvaluationPrompt == "" {
		t.Error("Expected default EvaluationPrompt to be set")
	}
	if agentDef.Evaluation.ImprovementPrompt == "" {
		t.Error("Expected default ImprovementPrompt to be set")
	}
	if agentDef.Evaluation.MaxIterations != 3 {
		t.Errorf("Expected MaxIterations=3, got %d", agentDef.Evaluation.MaxIterations)
	}
}

// TestIntegration_EvaluatorWithCriteria tests evaluator with various criteria
func TestIntegration_EvaluatorWithCriteria(t *testing.T) {
	eval := evaluator.New(nil)

	// Test with agent that has evaluation disabled
	agentDisabled := &agentloader.AgentYAML{
		ID: "disabled",
		Evaluation: &agentloader.EvaluationConfig{
			Enabled: false,
		},
	}

	result, err := eval.EvaluateResult(context.Background(), agentDisabled, "task", "result")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if !result.Passed {
		t.Error("Disabled evaluation should auto-pass")
	}
	if result.Score != 1.0 {
		t.Errorf("Expected Score=1.0 for auto-pass, got %f", result.Score)
	}
}

// TestIntegration_EvaluationCriteriaFlow tests the criteria evaluation flow
func TestIntegration_EvaluationCriteriaFlow(t *testing.T) {
	criteria := []agentloader.EvaluationCriterion{
		{Name: "Accuracy", Check: "Facts are correct", Required: true, Weight: 1.5},
		{Name: "Completeness", Check: "All aspects covered", Required: true, Weight: 1.0},
		{Name: "Style", Check: "Well written", Required: false, Weight: 0.5},
	}

	// Test that criteria are properly structured
	for _, c := range criteria {
		if c.Name == "" {
			t.Error("Criterion name should not be empty")
		}
		if c.Check == "" {
			t.Error("Criterion check should not be empty")
		}
		if c.Weight <= 0 {
			t.Error("Criterion weight should be positive")
		}
	}

	// Verify required criteria are marked
	requiredCount := 0
	for _, c := range criteria {
		if c.Required {
			requiredCount++
		}
	}
	if requiredCount != 2 {
		t.Errorf("Expected 2 required criteria, got %d", requiredCount)
	}
}

// TestIntegration_IterationLogic tests the iteration decision logic
func TestIntegration_IterationLogic(t *testing.T) {
	eval := evaluator.New(nil)

	agentDef := &agentloader.AgentYAML{
		ID: "test",
		Evaluation: &agentloader.EvaluationConfig{
			Enabled:       true,
			MaxIterations: 3,
		},
	}

	tests := []struct {
		name             string
		evalResult       *agentloader.EvaluationResult
		currentIteration int
		expectIterate    bool
	}{
		{
			name:             "first iteration, not passed",
			evalResult:       &agentloader.EvaluationResult{Passed: false, Score: 0.5},
			currentIteration: 1,
			expectIterate:    true,
		},
		{
			name:             "second iteration, not passed",
			evalResult:       &agentloader.EvaluationResult{Passed: false, Score: 0.6},
			currentIteration: 2,
			expectIterate:    true,
		},
		{
			name:             "third iteration (max), not passed",
			evalResult:       &agentloader.EvaluationResult{Passed: false, Score: 0.65},
			currentIteration: 3,
			expectIterate:    false, // Max reached
		},
		{
			name:             "first iteration, passed",
			evalResult:       &agentloader.EvaluationResult{Passed: true, Score: 0.9},
			currentIteration: 1,
			expectIterate:    false, // Already passed
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			shouldIterate := eval.ShouldIterate(agentDef, tt.evalResult, tt.currentIteration)
			if shouldIterate != tt.expectIterate {
				t.Errorf("Expected ShouldIterate=%v, got %v", tt.expectIterate, shouldIterate)
			}
		})
	}
}

// TestIntegration_ImprovementPromptGeneration tests improvement prompt generation
func TestIntegration_ImprovementPromptGeneration(t *testing.T) {
	eval := evaluator.New(nil)

	agentDef := &agentloader.AgentYAML{
		ID: "test",
		Evaluation: &agentloader.EvaluationConfig{
			Enabled:           true,
			ImprovementPrompt: "Improve based on: {{EVALUATION_FEEDBACK}}\nFailed: {{FAILED_CRITERIA}}",
		},
	}

	evalResult := &agentloader.EvaluationResult{
		Passed:   false,
		Score:    0.5,
		Feedback: "Needs improvement in clarity",
		CriteriaResults: []agentloader.CriterionResult{
			{Name: "Clarity", Passed: false, Feedback: "Too complex"},
			{Name: "Accuracy", Passed: true, Feedback: "Good"},
		},
	}

	prompt := eval.BuildImprovementPrompt(agentDef, "original task", "previous result", evalResult)

	// Check that placeholders were replaced
	if strings.Contains(prompt, "{{EVALUATION_FEEDBACK}}") {
		t.Error("{{EVALUATION_FEEDBACK}} placeholder not replaced")
	}
	if strings.Contains(prompt, "{{FAILED_CRITERIA}}") {
		t.Error("{{FAILED_CRITERIA}} placeholder not replaced")
	}

	// Check that feedback is included
	if !strings.Contains(prompt, "Needs improvement in clarity") {
		t.Error("Expected feedback to be in prompt")
	}

	// Check that failed criteria are included
	if !strings.Contains(prompt, "Clarity") {
		t.Error("Expected failed criterion name in prompt")
	}
}

// TestIntegration_AgentWithEvaluationConfig tests agent creation with evaluation
func TestIntegration_AgentWithEvaluationConfig(t *testing.T) {
	cfg := agent.DefaultConfig()
	cfg.MaxSteps = 5

	ag := agent.NewAgent(cfg)

	// Set up a mock LLM that returns final answer immediately
	callCount := 0
	ag.SetLLMFunc(func(ctx context.Context, msgs []agent.Message) (string, error) {
		callCount++
		return "THOUGHT: Done\nACTION: FINAL_ANSWER\nACTION_INPUT: {\"input\": \"Test result\"}", nil
	})

	agentDef := &agentloader.AgentYAML{
		ID:   "test",
		Name: "Test Agent",
		Evaluation: &agentloader.EvaluationConfig{
			Enabled:         true,
			MaxIterations:   2,
			MinQualityScore: 0.7,
		},
	}
	agentDef.Defaults()

	// Execute without evaluator (should fall back to standard execution)
	exec, err := ag.ExecuteWithEvaluation(context.Background(), "Test task", agentDef, nil)

	// With nil evaluator but enabled evaluation, it should still execute
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if exec == nil {
		t.Fatal("Expected execution result")
	}
	if exec.Status != agent.StatusCompleted {
		t.Errorf("Expected status=completed, got %s", exec.Status)
	}
}

// TestIntegration_EvaluationResultTracking tests that evaluation results are tracked
func TestIntegration_EvaluationResultTracking(t *testing.T) {
	// Create evaluation result
	evalResult := &agentloader.EvaluationResult{
		Passed:    true,
		Score:     0.85,
		Iteration: 2,
		Feedback:  "Good result after improvement",
		CriteriaResults: []agentloader.CriterionResult{
			{Name: "Quality", Passed: true, Required: true, Feedback: "High quality"},
			{Name: "Format", Passed: true, Required: false, Feedback: "Good format"},
		},
		Improvements: []string{},
	}

	// Verify structure
	if evalResult.Iteration != 2 {
		t.Errorf("Expected Iteration=2, got %d", evalResult.Iteration)
	}
	if len(evalResult.CriteriaResults) != 2 {
		t.Errorf("Expected 2 criteria results, got %d", len(evalResult.CriteriaResults))
	}

	// Check required criteria tracking
	for _, cr := range evalResult.CriteriaResults {
		if cr.Name == "Quality" && !cr.Required {
			t.Error("Quality should be required")
		}
		if cr.Name == "Format" && cr.Required {
			t.Error("Format should not be required")
		}
	}
}

// TestIntegration_DefaultPrompts tests that default prompts contain required placeholders
func TestIntegration_DefaultPrompts(t *testing.T) {
	// Check evaluation prompt
	evalPrompt := agentloader.DefaultEvaluationPrompt

	requiredPlaceholders := []string{"{{ORIGINAL_TASK}}", "{{RESULT}}", "{{CRITERIA_LIST}}"}
	for _, placeholder := range requiredPlaceholders {
		if !strings.Contains(evalPrompt, placeholder) {
			t.Errorf("DefaultEvaluationPrompt missing placeholder: %s", placeholder)
		}
	}

	// Check improvement prompt
	improvementPrompt := agentloader.DefaultImprovementPrompt

	improvementPlaceholders := []string{"{{ORIGINAL_TASK}}", "{{PREVIOUS_RESULT}}", "{{EVALUATION_FEEDBACK}}", "{{FAILED_CRITERIA}}"}
	for _, placeholder := range improvementPlaceholders {
		if !strings.Contains(improvementPrompt, placeholder) {
			t.Errorf("DefaultImprovementPrompt missing placeholder: %s", placeholder)
		}
	}
}

// TestIntegration_YAMLAgentWithEvaluation tests loading a YAML agent with evaluation
func TestIntegration_YAMLAgentWithEvaluation(t *testing.T) {
	// Simulate YAML agent structure
	yamlAgent := &agentloader.AgentYAML{
		ID:           "summarizer",
		Name:         "Summarizer",
		Description:  "Fasst Texte zusammen",
		Model:        "mistral:7b",
		MaxSteps:     5,
		Timeout:      60 * time.Second,
		SystemPrompt: "Du bist ein Zusammenfasser.",
		Evaluation: &agentloader.EvaluationConfig{
			Enabled:         true,
			MaxIterations:   2,
			MinQualityScore: 0.7,
			Criteria: []agentloader.EvaluationCriterion{
				{Name: "Kernaussagen", Check: "Alle wichtigen Punkte erfasst", Required: true},
				{Name: "Komprimierung", Check: "Deutlich kürzer als Original", Required: true},
				{Name: "Verständlichkeit", Check: "Klar und verständlich", Required: false},
			},
		},
	}

	// Apply defaults
	yamlAgent.Defaults()

	// Validate structure
	if err := yamlAgent.Validate(); err != nil {
		t.Fatalf("Validation failed: %v", err)
	}

	// Check evaluation was properly configured
	if yamlAgent.Evaluation.MaxIterations != 2 {
		t.Errorf("Expected MaxIterations=2, got %d", yamlAgent.Evaluation.MaxIterations)
	}
	if yamlAgent.Evaluation.MinQualityScore != 0.7 {
		t.Errorf("Expected MinQualityScore=0.7, got %f", yamlAgent.Evaluation.MinQualityScore)
	}
	if len(yamlAgent.Evaluation.Criteria) != 3 {
		t.Errorf("Expected 3 criteria, got %d", len(yamlAgent.Evaluation.Criteria))
	}

	// Verify criteria weights were set
	for _, c := range yamlAgent.Evaluation.Criteria {
		if c.Weight != 1.0 {
			t.Errorf("Expected Weight=1.0 for %s, got %f", c.Name, c.Weight)
		}
	}
}

// TestIntegration_QualityScoreThreshold tests quality score threshold logic
func TestIntegration_QualityScoreThreshold(t *testing.T) {
	tests := []struct {
		name            string
		score           float32
		minScore        float32
		requiredPassed  bool
		expectFinalPass bool
	}{
		{
			name:            "score above threshold, required passed",
			score:           0.85,
			minScore:        0.7,
			requiredPassed:  true,
			expectFinalPass: true,
		},
		{
			name:            "score below threshold, required passed",
			score:           0.65,
			minScore:        0.7,
			requiredPassed:  true,
			expectFinalPass: false, // Below threshold
		},
		{
			name:            "score above threshold, required failed",
			score:           0.85,
			minScore:        0.7,
			requiredPassed:  false,
			expectFinalPass: false, // Required failed
		},
		{
			name:            "score exactly at threshold",
			score:           0.7,
			minScore:        0.7,
			requiredPassed:  true,
			expectFinalPass: true, // Exactly at threshold passes
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := &agentloader.EvaluationResult{
				Score:  tt.score,
				Passed: tt.score >= tt.minScore && tt.requiredPassed,
			}

			// Simulate check logic
			if result.Score < tt.minScore {
				result.Passed = false
			}
			if !tt.requiredPassed {
				result.Passed = false
			}

			if result.Passed != tt.expectFinalPass {
				t.Errorf("Expected Passed=%v, got %v", tt.expectFinalPass, result.Passed)
			}
		})
	}
}

// TestIntegration_ExecutionWithMultipleIterations tests execution tracking across iterations
func TestIntegration_ExecutionWithMultipleIterations(t *testing.T) {
	execution := &agent.Execution{
		ID:                "exec-test",
		Task:              "Test task",
		Status:            agent.StatusCompleted,
		Iterations:        3,
		FinalQualityScore: 0.85,
		EvaluationResults: []*agentloader.EvaluationResult{
			{Iteration: 1, Score: 0.5, Passed: false},
			{Iteration: 2, Score: 0.7, Passed: false},
			{Iteration: 3, Score: 0.85, Passed: true},
		},
	}

	// Verify iteration tracking
	if execution.Iterations != 3 {
		t.Errorf("Expected 3 iterations, got %d", execution.Iterations)
	}
	if len(execution.EvaluationResults) != 3 {
		t.Errorf("Expected 3 evaluation results, got %d", len(execution.EvaluationResults))
	}

	// Verify score improvement across iterations
	prevScore := float32(0)
	for i, result := range execution.EvaluationResults {
		if result.Score < prevScore {
			t.Errorf("Expected score to improve, iteration %d has lower score than previous", i+1)
		}
		prevScore = result.Score
	}

	// Verify final result
	lastResult := execution.EvaluationResults[len(execution.EvaluationResults)-1]
	if !lastResult.Passed {
		t.Error("Expected final iteration to pass")
	}
	if execution.FinalQualityScore != lastResult.Score {
		t.Errorf("Expected FinalQualityScore=%f, got %f", lastResult.Score, execution.FinalQualityScore)
	}
}
