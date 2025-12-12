// ============================================================================
// meinDENKWERK (mDW) - Lokale KI-Plattform
// ============================================================================
//
// Package:     agentloader
// Description: Tests for agent types including evaluation configuration
// Author:      Mike Stoffels with Claude
// Created:     2025-12-12
// License:     MIT
// ============================================================================

package agentloader

import (
	"strings"
	"testing"
	"time"
)

// TestAgentYAMLDefaults tests the default value application
func TestAgentYAMLDefaults(t *testing.T) {
	tests := []struct {
		name     string
		agent    AgentYAML
		checkFn  func(*testing.T, *AgentYAML)
	}{
		{
			name:  "empty agent gets defaults",
			agent: AgentYAML{},
			checkFn: func(t *testing.T, a *AgentYAML) {
				if a.MaxSteps != 10 {
					t.Errorf("Expected MaxSteps=10, got %d", a.MaxSteps)
				}
				if a.Timeout != 120*time.Second {
					t.Errorf("Expected Timeout=120s, got %v", a.Timeout)
				}
				if a.Temperature != 0.7 {
					t.Errorf("Expected Temperature=0.7, got %f", a.Temperature)
				}
				if a.Model != "mistral:7b" {
					t.Errorf("Expected Model=mistral:7b, got %s", a.Model)
				}
			},
		},
		{
			name: "custom values preserved",
			agent: AgentYAML{
				MaxSteps:    20,
				Timeout:     60 * time.Second,
				Temperature: 0.5,
				Model:       "llama2:7b",
			},
			checkFn: func(t *testing.T, a *AgentYAML) {
				if a.MaxSteps != 20 {
					t.Errorf("Expected MaxSteps=20, got %d", a.MaxSteps)
				}
				if a.Timeout != 60*time.Second {
					t.Errorf("Expected Timeout=60s, got %v", a.Timeout)
				}
				if a.Temperature != 0.5 {
					t.Errorf("Expected Temperature=0.5, got %f", a.Temperature)
				}
				if a.Model != "llama2:7b" {
					t.Errorf("Expected Model=llama2:7b, got %s", a.Model)
				}
			},
		},
		{
			name: "tools get enabled by default",
			agent: AgentYAML{
				Tools: []ToolConfig{
					{Name: "tool1"},
					{Name: "tool2", Enabled: false},
				},
			},
			checkFn: func(t *testing.T, a *AgentYAML) {
				// Note: Current implementation sets all to true
				for i, tool := range a.Tools {
					if !tool.Enabled {
						t.Errorf("Expected tool[%d] to be enabled", i)
					}
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			agent := tt.agent
			agent.Defaults()
			tt.checkFn(t, &agent)
		})
	}
}

// TestAgentYAMLDefaults_WithEvaluation tests defaults when evaluation is enabled
func TestAgentYAMLDefaults_WithEvaluation(t *testing.T) {
	agent := AgentYAML{
		ID: "test",
		Evaluation: &EvaluationConfig{
			Enabled: true,
			Criteria: []EvaluationCriterion{
				{Name: "Quality", Check: "High quality"},
			},
		},
	}

	agent.Defaults()

	if agent.Evaluation.MaxIterations != 2 {
		t.Errorf("Expected MaxIterations=2, got %d", agent.Evaluation.MaxIterations)
	}
	if agent.Evaluation.MinQualityScore != 0.7 {
		t.Errorf("Expected MinQualityScore=0.7, got %f", agent.Evaluation.MinQualityScore)
	}
	if agent.Evaluation.Criteria[0].Weight != 1.0 {
		t.Errorf("Expected Weight=1.0, got %f", agent.Evaluation.Criteria[0].Weight)
	}
	if agent.Evaluation.EvaluationPrompt == "" {
		t.Error("Expected default EvaluationPrompt")
	}
	if agent.Evaluation.ImprovementPrompt == "" {
		t.Error("Expected default ImprovementPrompt")
	}
}

// TestAgentYAMLDefaults_DisabledEvaluation tests that disabled evaluation is not processed
func TestAgentYAMLDefaults_DisabledEvaluation(t *testing.T) {
	agent := AgentYAML{
		ID: "test",
		Evaluation: &EvaluationConfig{
			Enabled: false, // Disabled
		},
	}

	agent.Defaults()

	// Should not apply defaults when disabled
	if agent.Evaluation.MaxIterations != 0 {
		t.Errorf("Expected MaxIterations=0 (unchanged), got %d", agent.Evaluation.MaxIterations)
	}
}

// TestEvaluationConfigDefaults tests EvaluationConfig default values
func TestEvaluationConfigDefaults(t *testing.T) {
	tests := []struct {
		name    string
		config  EvaluationConfig
		checkFn func(*testing.T, *EvaluationConfig)
	}{
		{
			name:   "empty config",
			config: EvaluationConfig{},
			checkFn: func(t *testing.T, c *EvaluationConfig) {
				if c.MaxIterations != 2 {
					t.Errorf("Expected MaxIterations=2, got %d", c.MaxIterations)
				}
				if c.MinQualityScore != 0.7 {
					t.Errorf("Expected MinQualityScore=0.7, got %f", c.MinQualityScore)
				}
			},
		},
		{
			name: "custom values preserved",
			config: EvaluationConfig{
				MaxIterations:   5,
				MinQualityScore: 0.9,
			},
			checkFn: func(t *testing.T, c *EvaluationConfig) {
				if c.MaxIterations != 5 {
					t.Errorf("Expected MaxIterations=5, got %d", c.MaxIterations)
				}
				if c.MinQualityScore != 0.9 {
					t.Errorf("Expected MinQualityScore=0.9, got %f", c.MinQualityScore)
				}
			},
		},
		{
			name: "criteria weights set to 1.0",
			config: EvaluationConfig{
				Criteria: []EvaluationCriterion{
					{Name: "A", Check: "Check A"},
					{Name: "B", Check: "Check B", Weight: 2.0},
					{Name: "C", Check: "Check C"},
				},
			},
			checkFn: func(t *testing.T, c *EvaluationConfig) {
				if c.Criteria[0].Weight != 1.0 {
					t.Errorf("Expected Criteria[0].Weight=1.0, got %f", c.Criteria[0].Weight)
				}
				if c.Criteria[1].Weight != 2.0 {
					t.Errorf("Expected Criteria[1].Weight=2.0 (preserved), got %f", c.Criteria[1].Weight)
				}
				if c.Criteria[2].Weight != 1.0 {
					t.Errorf("Expected Criteria[2].Weight=1.0, got %f", c.Criteria[2].Weight)
				}
			},
		},
		{
			name: "default prompts applied",
			config: EvaluationConfig{},
			checkFn: func(t *testing.T, c *EvaluationConfig) {
				if c.EvaluationPrompt != DefaultEvaluationPrompt {
					t.Error("Expected default EvaluationPrompt")
				}
				if c.ImprovementPrompt != DefaultImprovementPrompt {
					t.Error("Expected default ImprovementPrompt")
				}
			},
		},
		{
			name: "custom prompts preserved",
			config: EvaluationConfig{
				EvaluationPrompt:  "Custom eval",
				ImprovementPrompt: "Custom improve",
			},
			checkFn: func(t *testing.T, c *EvaluationConfig) {
				if c.EvaluationPrompt != "Custom eval" {
					t.Errorf("Expected custom EvaluationPrompt, got %s", c.EvaluationPrompt)
				}
				if c.ImprovementPrompt != "Custom improve" {
					t.Errorf("Expected custom ImprovementPrompt, got %s", c.ImprovementPrompt)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := tt.config
			config.Defaults()
			tt.checkFn(t, &config)
		})
	}
}

// TestAgentYAMLValidate tests the validation logic
func TestAgentYAMLValidate(t *testing.T) {
	tests := []struct {
		name      string
		agent     AgentYAML
		expectErr error
	}{
		{
			name: "valid agent",
			agent: AgentYAML{
				ID:           "test-agent",
				Name:         "Test Agent",
				SystemPrompt: "You are a test agent",
			},
			expectErr: nil,
		},
		{
			name: "missing ID",
			agent: AgentYAML{
				Name:         "Test Agent",
				SystemPrompt: "You are a test agent",
			},
			expectErr: ErrMissingID,
		},
		{
			name: "missing Name",
			agent: AgentYAML{
				ID:           "test-agent",
				SystemPrompt: "You are a test agent",
			},
			expectErr: ErrMissingName,
		},
		{
			name: "missing SystemPrompt",
			agent: AgentYAML{
				ID:   "test-agent",
				Name: "Test Agent",
			},
			expectErr: ErrMissingSystemPrompt,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.agent.Validate()

			if tt.expectErr == nil {
				if err != nil {
					t.Errorf("Expected no error, got %v", err)
				}
			} else {
				if err != tt.expectErr {
					t.Errorf("Expected error %v, got %v", tt.expectErr, err)
				}
			}
		})
	}
}

// TestAgentYAMLGetToolNames tests the tool names extraction
func TestAgentYAMLGetToolNames(t *testing.T) {
	tests := []struct {
		name     string
		agent    AgentYAML
		expected []string
	}{
		{
			name:     "no tools",
			agent:    AgentYAML{},
			expected: nil,
		},
		{
			name: "all enabled",
			agent: AgentYAML{
				Tools: []ToolConfig{
					{Name: "tool1", Enabled: true},
					{Name: "tool2", Enabled: true},
				},
			},
			expected: []string{"tool1", "tool2"},
		},
		{
			name: "mixed enabled/disabled",
			agent: AgentYAML{
				Tools: []ToolConfig{
					{Name: "tool1", Enabled: true},
					{Name: "tool2", Enabled: false},
					{Name: "tool3", Enabled: true},
				},
			},
			expected: []string{"tool1", "tool3"},
		},
		{
			name: "all disabled",
			agent: AgentYAML{
				Tools: []ToolConfig{
					{Name: "tool1", Enabled: false},
					{Name: "tool2", Enabled: false},
				},
			},
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.agent.GetToolNames()

			if len(result) != len(tt.expected) {
				t.Errorf("Expected %d tool names, got %d", len(tt.expected), len(result))
				return
			}

			for i, name := range result {
				if name != tt.expected[i] {
					t.Errorf("Expected tool[%d]=%s, got %s", i, tt.expected[i], name)
				}
			}
		})
	}
}

// TestAgentYAMLProcessPlaceholders tests placeholder replacement
func TestAgentYAMLProcessPlaceholders(t *testing.T) {
	agent := AgentYAML{
		SystemPrompt: "Today is {{DATE}}. Year: {{YEAR}}. Time: {{TIME}}.",
	}

	agent.ProcessPlaceholders()

	// Check that placeholders were replaced
	if strings.Contains(agent.SystemPrompt, "{{DATE}}") {
		t.Error("{{DATE}} placeholder not replaced")
	}
	if strings.Contains(agent.SystemPrompt, "{{YEAR}}") {
		t.Error("{{YEAR}} placeholder not replaced")
	}
	if strings.Contains(agent.SystemPrompt, "{{TIME}}") {
		t.Error("{{TIME}} placeholder not replaced")
	}

	// Check that current year is present
	currentYear := time.Now().Format("2006")
	if !strings.Contains(agent.SystemPrompt, currentYear) {
		t.Errorf("Expected year %s in prompt: %s", currentYear, agent.SystemPrompt)
	}
}

// TestDefaultEvaluationPromptPlaceholders tests that default prompts have expected placeholders
func TestDefaultEvaluationPromptPlaceholders(t *testing.T) {
	expectedPlaceholders := []string{
		"{{ORIGINAL_TASK}}",
		"{{RESULT}}",
		"{{CRITERIA_LIST}}",
	}

	for _, placeholder := range expectedPlaceholders {
		if !strings.Contains(DefaultEvaluationPrompt, placeholder) {
			t.Errorf("DefaultEvaluationPrompt missing placeholder: %s", placeholder)
		}
	}
}

// TestDefaultImprovementPromptPlaceholders tests that improvement prompt has expected placeholders
func TestDefaultImprovementPromptPlaceholders(t *testing.T) {
	expectedPlaceholders := []string{
		"{{ORIGINAL_TASK}}",
		"{{PREVIOUS_RESULT}}",
		"{{EVALUATION_FEEDBACK}}",
		"{{FAILED_CRITERIA}}",
	}

	for _, placeholder := range expectedPlaceholders {
		if !strings.Contains(DefaultImprovementPrompt, placeholder) {
			t.Errorf("DefaultImprovementPrompt missing placeholder: %s", placeholder)
		}
	}
}

// TestEvaluationCriterion tests the criterion structure
func TestEvaluationCriterion(t *testing.T) {
	criterion := EvaluationCriterion{
		Name:     "Accuracy",
		Check:    "The answer must be factually correct",
		Required: true,
		Weight:   1.5,
	}

	if criterion.Name != "Accuracy" {
		t.Errorf("Expected Name=Accuracy, got %s", criterion.Name)
	}
	if criterion.Check != "The answer must be factually correct" {
		t.Errorf("Unexpected Check value: %s", criterion.Check)
	}
	if !criterion.Required {
		t.Error("Expected Required=true")
	}
	if criterion.Weight != 1.5 {
		t.Errorf("Expected Weight=1.5, got %f", criterion.Weight)
	}
}

// TestEvaluationResult tests the result structure
func TestEvaluationResult(t *testing.T) {
	result := EvaluationResult{
		Passed: true,
		Score:  0.85,
		CriteriaResults: []CriterionResult{
			{Name: "A", Passed: true, Required: true, Feedback: "Good"},
			{Name: "B", Passed: false, Required: false, Feedback: "Needs work"},
		},
		Feedback:     "Overall good",
		Improvements: []string{"Improve B"},
		Iteration:    2,
	}

	if !result.Passed {
		t.Error("Expected Passed=true")
	}
	if result.Score != 0.85 {
		t.Errorf("Expected Score=0.85, got %f", result.Score)
	}
	if len(result.CriteriaResults) != 2 {
		t.Errorf("Expected 2 criteria results, got %d", len(result.CriteriaResults))
	}
	if result.Iteration != 2 {
		t.Errorf("Expected Iteration=2, got %d", result.Iteration)
	}
}

// TestCriterionResult tests the criterion result structure
func TestCriterionResult(t *testing.T) {
	cr := CriterionResult{
		Name:     "Quality",
		Passed:   true,
		Required: true,
		Feedback: "Excellent quality",
	}

	if cr.Name != "Quality" {
		t.Errorf("Expected Name=Quality, got %s", cr.Name)
	}
	if !cr.Passed {
		t.Error("Expected Passed=true")
	}
	if !cr.Required {
		t.Error("Expected Required=true")
	}
	if cr.Feedback != "Excellent quality" {
		t.Errorf("Unexpected Feedback: %s", cr.Feedback)
	}
}
