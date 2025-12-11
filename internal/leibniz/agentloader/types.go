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

	// Metadata for extensibility
	Metadata map[string]string `yaml:"metadata,omitempty"`

	// Embedding for vector similarity matching (persisted in YAML)
	Embedding     []float64 `yaml:"embedding,omitempty"`      // Vector embedding für Agent-Matching
	EmbeddingHash string    `yaml:"embedding_hash,omitempty"` // Hash des Textes für Cache-Validierung

	// Internal tracking (not from YAML)
	SourceFile string    `yaml:"-"`
	LoadedAt   time.Time `yaml:"-"`
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
}

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
