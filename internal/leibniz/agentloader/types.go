// ============================================================================
// meinDENKWERK (mDW) - Lokale KI-Plattform
// ============================================================================
//
// Package:     agentloader
// Description: YAML-based agent definitions with hot-reload support
// Author:      Mike Stoffels with Claude
// Created:     2025-12-11
// License:     MIT
// ============================================================================

package agentloader

import (
	"time"
)

// AgentYAML represents an agent definition loaded from YAML
type AgentYAML struct {
	// Core identification
	ID          string `yaml:"id"`
	Name        string `yaml:"name"`
	Description string `yaml:"description"`

	// Model configuration
	Model       string  `yaml:"model"`
	Temperature float32 `yaml:"temperature,omitempty"`

	// Execution limits
	MaxSteps int           `yaml:"max_steps,omitempty"`
	Timeout  time.Duration `yaml:"timeout,omitempty"`

	// Prompts
	SystemPrompt string `yaml:"system_prompt"`

	// Tools configuration
	Tools []ToolConfig `yaml:"tools,omitempty"`

	// Optional: Platon pipeline integration
	PlatonEnabled    bool   `yaml:"platon_enabled,omitempty"`
	PlatonPipelineID string `yaml:"platon_pipeline_id,omitempty"`

	// Self-Evaluation configuration
	Evaluation *EvaluationConfig `yaml:"evaluation,omitempty"`

	// Metadata for extensibility
	Metadata map[string]string `yaml:"metadata,omitempty"`

	// Embedding for vector similarity matching (persisted in YAML)
	Embedding     []float64 `yaml:"embedding,omitempty"`      // Vector embedding für Agent-Matching
	EmbeddingHash string    `yaml:"embedding_hash,omitempty"` // Hash des Textes für Cache-Validierung

	// Internal tracking (not from YAML)
	SourceFile string    `yaml:"-"`
	LoadedAt   time.Time `yaml:"-"`
}

// EvaluationConfig defines self-evaluation settings for an agent
type EvaluationConfig struct {
	// Enabled activates self-evaluation for this agent
	Enabled bool `yaml:"enabled"`

	// MaxIterations limits the number of improvement cycles (1 = no iteration)
	MaxIterations int `yaml:"max_iterations,omitempty"`

	// Criteria defines the KPIs that must be fulfilled
	Criteria []EvaluationCriterion `yaml:"criteria,omitempty"`

	// EvaluationPrompt is the prompt template for self-evaluation
	// Available placeholders: {{ORIGINAL_TASK}}, {{RESULT}}, {{CRITERIA_LIST}}
	EvaluationPrompt string `yaml:"evaluation_prompt,omitempty"`

	// ImprovementPrompt is the prompt template for improvement iterations
	// Available placeholders: {{ORIGINAL_TASK}}, {{PREVIOUS_RESULT}}, {{EVALUATION_FEEDBACK}}, {{FAILED_CRITERIA}}
	ImprovementPrompt string `yaml:"improvement_prompt,omitempty"`

	// MinQualityScore is the minimum score (0.0-1.0) to pass evaluation
	MinQualityScore float32 `yaml:"min_quality_score,omitempty"`

	// EvaluationModel allows using a different model for evaluation (optional)
	EvaluationModel string `yaml:"evaluation_model,omitempty"`
}

// EvaluationCriterion defines a single KPI for evaluation
type EvaluationCriterion struct {
	// Name is the criterion identifier
	Name string `yaml:"name"`

	// Check describes what to verify (used in evaluation prompt)
	Check string `yaml:"check"`

	// Required indicates if this criterion must pass
	Required bool `yaml:"required,omitempty"`

	// Weight for scoring (default: 1.0)
	Weight float32 `yaml:"weight,omitempty"`
}

// EvaluationResult represents the result of a self-evaluation
type EvaluationResult struct {
	// Passed indicates if all required criteria were met
	Passed bool

	// Score is the overall quality score (0.0-1.0)
	Score float32

	// CriteriaResults contains individual criterion results
	CriteriaResults []CriterionResult

	// Feedback is the LLM's explanation
	Feedback string

	// Improvements lists suggested improvements (if not passed)
	Improvements []string

	// Iteration is the current iteration number
	Iteration int
}

// CriterionResult represents the result of a single criterion check
type CriterionResult struct {
	Name     string
	Passed   bool
	Required bool
	Feedback string
}

// ToolConfig allows per-agent tool configuration
type ToolConfig struct {
	Name    string                 `yaml:"name"`
	Enabled bool                   `yaml:"enabled,omitempty"` // Default: true if listed
	Config  map[string]interface{} `yaml:"config,omitempty"`  // Tool-specific config
}

// Defaults applies default values to the agent definition
func (a *AgentYAML) Defaults() {
	if a.MaxSteps == 0 {
		a.MaxSteps = 10
	}
	if a.Timeout == 0 {
		a.Timeout = 120 * time.Second
	}
	if a.Temperature == 0 {
		a.Temperature = 0.7
	}
	if a.Model == "" {
		a.Model = "mistral:7b"
	}

	// Set enabled=true for all tools if not explicitly set
	for i := range a.Tools {
		if !a.Tools[i].Enabled {
			a.Tools[i].Enabled = true
		}
	}

	// Apply evaluation defaults if enabled
	if a.Evaluation != nil && a.Evaluation.Enabled {
		a.Evaluation.Defaults()
	}
}

// Defaults applies default values to the evaluation config
func (e *EvaluationConfig) Defaults() {
	if e.MaxIterations == 0 {
		e.MaxIterations = 2 // Default: 1 initial + 1 improvement
	}
	if e.MinQualityScore == 0 {
		e.MinQualityScore = 0.7 // 70% quality threshold
	}

	// Set default weights for criteria
	for i := range e.Criteria {
		if e.Criteria[i].Weight == 0 {
			e.Criteria[i].Weight = 1.0
		}
	}

	// Default evaluation prompt if not set
	if e.EvaluationPrompt == "" {
		e.EvaluationPrompt = DefaultEvaluationPrompt
	}

	// Default improvement prompt if not set
	if e.ImprovementPrompt == "" {
		e.ImprovementPrompt = DefaultImprovementPrompt
	}
}

// DefaultEvaluationPrompt is the default prompt for self-evaluation
const DefaultEvaluationPrompt = `Überprüfe das folgende Ergebnis anhand der gegebenen Kriterien.

URSPRÜNGLICHE AUFGABE:
{{ORIGINAL_TASK}}

ERGEBNIS:
{{RESULT}}

ZU PRÜFENDE KRITERIEN:
{{CRITERIA_LIST}}

Antworte im folgenden JSON-Format:
{
  "passed": true/false,
  "score": 0.0-1.0,
  "criteria_results": [
    {"name": "Kriterium1", "passed": true/false, "feedback": "Begründung"}
  ],
  "feedback": "Gesamtbewertung",
  "improvements": ["Verbesserung 1", "Verbesserung 2"]
}
`

// DefaultImprovementPrompt is the default prompt for improvement iterations
const DefaultImprovementPrompt = `Verbessere dein vorheriges Ergebnis basierend auf dem Feedback.

URSPRÜNGLICHE AUFGABE:
{{ORIGINAL_TASK}}

VORHERIGES ERGEBNIS:
{{PREVIOUS_RESULT}}

FEEDBACK ZUR VERBESSERUNG:
{{EVALUATION_FEEDBACK}}

NICHT ERFÜLLTE KRITERIEN:
{{FAILED_CRITERIA}}

Erstelle eine verbesserte Version, die die genannten Probleme behebt.
`

// GetToolNames returns a list of enabled tool names
func (a *AgentYAML) GetToolNames() []string {
	var names []string
	for _, t := range a.Tools {
		if t.Enabled {
			names = append(names, t.Name)
		}
	}
	return names
}

// Validate checks if the agent definition is valid
func (a *AgentYAML) Validate() error {
	if a.ID == "" {
		return ErrMissingID
	}
	if a.Name == "" {
		return ErrMissingName
	}
	if a.SystemPrompt == "" {
		return ErrMissingSystemPrompt
	}
	return nil
}

// ProcessPlaceholders replaces dynamic placeholders in the system prompt
func (a *AgentYAML) ProcessPlaceholders() {
	now := time.Now()

	// Replace date/time placeholders
	replacements := map[string]string{
		"{{DATE}}":     now.Format("02.01.2006"),
		"{{YEAR}}":     now.Format("2006"),
		"{{MONTH}}":    now.Format("01"),
		"{{DAY}}":      now.Format("02"),
		"{{TIME}}":     now.Format("15:04"),
		"{{DATETIME}}": now.Format("02.01.2006 15:04"),
	}

	for placeholder, value := range replacements {
		a.SystemPrompt = replaceAll(a.SystemPrompt, placeholder, value)
	}
}

// replaceAll is a helper to replace all occurrences
func replaceAll(s, old, new string) string {
	for {
		idx := indexOf(s, old)
		if idx == -1 {
			break
		}
		s = s[:idx] + new + s[idx+len(old):]
	}
	return s
}

func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
