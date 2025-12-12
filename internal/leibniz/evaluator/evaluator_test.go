// ============================================================================
// meinDENKWERK (mDW) - Lokale KI-Plattform
// ============================================================================
//
// Package:     evaluator
// Description: Tests for the self-evaluation service
// Author:      Mike Stoffels with Claude
// Created:     2025-12-12
// License:     MIT
// ============================================================================

package evaluator

import (
	"context"
	"testing"

	"github.com/msto63/mDW/internal/leibniz/agentloader"
)

// TestNew tests the creation of a new Evaluator
func TestNew(t *testing.T) {
	eval := New(nil)

	if eval == nil {
		t.Fatal("New() returned nil")
	}
	if eval.logger == nil {
		t.Error("Logger not initialized")
	}
}

// TestEvaluateResult_DisabledEvaluation tests auto-pass when evaluation is disabled
func TestEvaluateResult_DisabledEvaluation(t *testing.T) {
	tests := []struct {
		name  string
		agent *agentloader.AgentYAML
	}{
		{
			name:  "nil evaluation config",
			agent: &agentloader.AgentYAML{ID: "test"},
		},
		{
			name: "evaluation disabled",
			agent: &agentloader.AgentYAML{
				ID: "test",
				Evaluation: &agentloader.EvaluationConfig{
					Enabled: false,
				},
			},
		},
	}

	eval := New(nil)
	ctx := context.Background()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := eval.EvaluateResult(ctx, tt.agent, "task", "result")

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
			if result == nil {
				t.Fatal("Expected result, got nil")
			}
			if !result.Passed {
				t.Error("Expected Passed=true for disabled evaluation")
			}
			if result.Score != 1.0 {
				t.Errorf("Expected Score=1.0, got %f", result.Score)
			}
			if result.Feedback != "Evaluation not configured - auto-pass" {
				t.Errorf("Unexpected feedback: %s", result.Feedback)
			}
		})
	}
}

// TestBuildImprovementPrompt tests the improvement prompt generation
func TestBuildImprovementPrompt(t *testing.T) {
	eval := New(nil)

	tests := []struct {
		name           string
		agent          *agentloader.AgentYAML
		originalTask   string
		previousResult string
		evalResult     *agentloader.EvaluationResult
		expectContains []string
	}{
		{
			name:           "nil evaluation config returns original task",
			agent:          &agentloader.AgentYAML{ID: "test"},
			originalTask:   "Original task",
			previousResult: "Previous result",
			evalResult:     &agentloader.EvaluationResult{},
			expectContains: []string{"Original task"},
		},
		{
			name: "with evaluation config",
			agent: &agentloader.AgentYAML{
				ID: "test",
				Evaluation: &agentloader.EvaluationConfig{
					Enabled:           true,
					ImprovementPrompt: "Task: {{ORIGINAL_TASK}}\nPrev: {{PREVIOUS_RESULT}}\nFeedback: {{EVALUATION_FEEDBACK}}\nFailed: {{FAILED_CRITERIA}}",
				},
			},
			originalTask:   "Write a summary",
			previousResult: "Short summary",
			evalResult: &agentloader.EvaluationResult{
				Feedback: "Needs more detail",
				CriteriaResults: []agentloader.CriterionResult{
					{Name: "Length", Passed: false, Feedback: "Too short"},
					{Name: "Quality", Passed: true, Feedback: "Good"},
				},
			},
			expectContains: []string{
				"Task: Write a summary",
				"Prev: Short summary",
				"Feedback: Needs more detail",
				"Length: Too short",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := eval.BuildImprovementPrompt(tt.agent, tt.originalTask, tt.previousResult, tt.evalResult)

			for _, expected := range tt.expectContains {
				if !contains(result, expected) {
					t.Errorf("Expected prompt to contain %q, got: %s", expected, result)
				}
			}
		})
	}
}

// TestShouldIterate tests the iteration decision logic
func TestShouldIterate(t *testing.T) {
	eval := New(nil)

	tests := []struct {
		name             string
		agent            *agentloader.AgentYAML
		evalResult       *agentloader.EvaluationResult
		currentIteration int
		expected         bool
	}{
		{
			name:             "nil evaluation config",
			agent:            &agentloader.AgentYAML{ID: "test"},
			evalResult:       &agentloader.EvaluationResult{Passed: false},
			currentIteration: 1,
			expected:         false,
		},
		{
			name: "evaluation disabled",
			agent: &agentloader.AgentYAML{
				ID: "test",
				Evaluation: &agentloader.EvaluationConfig{
					Enabled: false,
				},
			},
			evalResult:       &agentloader.EvaluationResult{Passed: false},
			currentIteration: 1,
			expected:         false,
		},
		{
			name: "already passed",
			agent: &agentloader.AgentYAML{
				ID: "test",
				Evaluation: &agentloader.EvaluationConfig{
					Enabled:       true,
					MaxIterations: 3,
				},
			},
			evalResult:       &agentloader.EvaluationResult{Passed: true},
			currentIteration: 1,
			expected:         false,
		},
		{
			name: "max iterations reached",
			agent: &agentloader.AgentYAML{
				ID: "test",
				Evaluation: &agentloader.EvaluationConfig{
					Enabled:       true,
					MaxIterations: 2,
				},
			},
			evalResult:       &agentloader.EvaluationResult{Passed: false},
			currentIteration: 2,
			expected:         false,
		},
		{
			name: "should iterate",
			agent: &agentloader.AgentYAML{
				ID: "test",
				Evaluation: &agentloader.EvaluationConfig{
					Enabled:       true,
					MaxIterations: 3,
				},
			},
			evalResult:       &agentloader.EvaluationResult{Passed: false},
			currentIteration: 1,
			expected:         true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := eval.ShouldIterate(tt.agent, tt.evalResult, tt.currentIteration)

			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

// TestBuildCriteriaList tests the criteria list formatting
func TestBuildCriteriaList(t *testing.T) {
	eval := New(nil)

	criteria := []agentloader.EvaluationCriterion{
		{Name: "Completeness", Check: "All points covered", Required: true},
		{Name: "Clarity", Check: "Easy to understand", Required: false},
		{Name: "Accuracy", Check: "Factually correct", Required: true},
	}

	result := eval.buildCriteriaList(criteria)

	expectedSubstrings := []string{
		"1. Completeness [PFLICHT]: All points covered",
		"2. Clarity: Easy to understand",
		"3. Accuracy [PFLICHT]: Factually correct",
	}

	for _, expected := range expectedSubstrings {
		if !contains(result, expected) {
			t.Errorf("Expected list to contain %q, got: %s", expected, result)
		}
	}
}

// TestParseEvaluationResponse tests JSON parsing of evaluation responses
func TestParseEvaluationResponse(t *testing.T) {
	eval := New(nil)

	criteria := []agentloader.EvaluationCriterion{
		{Name: "Quality", Check: "High quality", Required: true},
		{Name: "Format", Check: "Proper format", Required: false},
	}

	tests := []struct {
		name           string
		response       string
		expectError    bool
		expectPassed   bool
		expectScore    float32
		expectFeedback string
	}{
		{
			name: "valid JSON",
			response: `{
				"passed": true,
				"score": 0.85,
				"criteria_results": [
					{"name": "Quality", "passed": true, "feedback": "Excellent"},
					{"name": "Format", "passed": true, "feedback": "Good format"}
				],
				"feedback": "Overall good",
				"improvements": ["Minor polish needed"]
			}`,
			expectError:    false,
			expectPassed:   true,
			expectScore:    0.85,
			expectFeedback: "Overall good",
		},
		{
			name: "JSON in markdown block",
			response: "Here is my evaluation:\n```json\n" + `{
				"passed": false,
				"score": 0.5,
				"criteria_results": [],
				"feedback": "Needs work"
			}` + "\n```",
			expectError:    false,
			expectPassed:   false,
			expectScore:    0.5,
			expectFeedback: "Needs work",
		},
		{
			name: "JSON with surrounding text",
			response: `Based on my analysis: {"passed": true, "score": 0.9, "criteria_results": [], "feedback": "Great"} That's my assessment.`,
			expectError:    false,
			expectPassed:   true,
			expectScore:    0.9,
			expectFeedback: "Great",
		},
		{
			name:        "invalid JSON",
			response:    "This is not valid JSON at all",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := eval.parseEvaluationResponse(tt.response, criteria)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}
			if result.Passed != tt.expectPassed {
				t.Errorf("Expected Passed=%v, got %v", tt.expectPassed, result.Passed)
			}
			if result.Score != tt.expectScore {
				t.Errorf("Expected Score=%f, got %f", tt.expectScore, result.Score)
			}
			if result.Feedback != tt.expectFeedback {
				t.Errorf("Expected Feedback=%q, got %q", tt.expectFeedback, result.Feedback)
			}
		})
	}
}

// TestHeuristicEvaluation tests the fallback heuristic evaluation
func TestHeuristicEvaluation(t *testing.T) {
	eval := New(nil)

	criteria := []agentloader.EvaluationCriterion{
		{Name: "Quality", Check: "High quality", Required: true},
		{Name: "Format", Check: "Proper format", Required: false},
	}

	tests := []struct {
		name         string
		result       string
		expectPassed bool
		expectScore  float32
	}{
		{
			name:         "short result fails",
			result:       "Short",
			expectPassed: false,
			expectScore:  0.5,
		},
		{
			name:         "long result passes",
			result:       "This is a much longer result that contains more than 100 characters. It should pass the basic heuristic check because it has sufficient length to be considered valid.",
			expectPassed: true,
			expectScore:  0.7,
		},
		{
			name:         "exactly 100 chars fails",
			result:       "1234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890",
			expectPassed: false,
			expectScore:  0.5,
		},
		{
			name:         "101 chars passes",
			result:       "12345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901",
			expectPassed: true,
			expectScore:  0.7,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := eval.heuristicEvaluation(tt.result, criteria)

			if result.Passed != tt.expectPassed {
				t.Errorf("Expected Passed=%v, got %v (result length: %d)", tt.expectPassed, result.Passed, len(tt.result))
			}
			if result.Score != tt.expectScore {
				t.Errorf("Expected Score=%f, got %f", tt.expectScore, result.Score)
			}
			if len(result.CriteriaResults) != len(criteria) {
				t.Errorf("Expected %d criteria results, got %d", len(criteria), len(result.CriteriaResults))
			}
			if result.Feedback != "Heuristic evaluation (JSON parse failed)" {
				t.Errorf("Unexpected feedback: %s", result.Feedback)
			}
		})
	}
}

// TestExtractJSON tests the JSON extraction from various formats
func TestExtractJSON(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "plain JSON",
			input:    `{"key": "value"}`,
			expected: `{"key": "value"}`,
		},
		{
			name:     "JSON in markdown code block",
			input:    "```json\n{\"key\": \"value\"}\n```",
			expected: `{"key": "value"}`,
		},
		{
			name:     "JSON in generic code block",
			input:    "```\n{\"key\": \"value\"}\n```",
			expected: `{"key": "value"}`,
		},
		{
			name:     "JSON with surrounding text",
			input:    "Here is the result: {\"key\": \"value\"} as requested.",
			expected: `{"key": "value"}`,
		},
		{
			name:     "nested JSON",
			input:    `{"outer": {"inner": "value"}}`,
			expected: `{"outer": {"inner": "value"}}`,
		},
		{
			name:     "no JSON",
			input:    "Just plain text",
			expected: "Just plain text",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractJSON(tt.input)

			if result != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, result)
			}
		})
	}
}

// Helper function
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsSubstring(s, substr))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
