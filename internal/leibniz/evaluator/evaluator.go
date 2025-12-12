// ============================================================================
// meinDENKWERK (mDW) - Lokale KI-Plattform
// ============================================================================
//
// Package:     evaluator
// Description: Self-evaluation service for agent results with iterative improvement
// Author:      Mike Stoffels with Claude
// Created:     2025-12-12
// License:     MIT
// ============================================================================

package evaluator

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/msto63/mDW/internal/leibniz/agentloader"
	"github.com/msto63/mDW/internal/turing/ollama"
	"github.com/msto63/mDW/pkg/core/logging"
)

// Evaluator performs self-evaluation of agent results
type Evaluator struct {
	llmClient *ollama.Client
	logger    *logging.Logger
}

// New creates a new Evaluator
func New(llmClient *ollama.Client) *Evaluator {
	return &Evaluator{
		llmClient: llmClient,
		logger:    logging.New("leibniz-evaluator"),
	}
}

// EvaluateResult evaluates an agent's result against its criteria
func (e *Evaluator) EvaluateResult(
	ctx context.Context,
	agent *agentloader.AgentYAML,
	originalTask string,
	result string,
) (*agentloader.EvaluationResult, error) {
	if agent.Evaluation == nil || !agent.Evaluation.Enabled {
		// No evaluation configured - auto-pass
		return &agentloader.EvaluationResult{
			Passed:    true,
			Score:     1.0,
			Feedback:  "Evaluation not configured - auto-pass",
			Iteration: 1,
		}, nil
	}

	eval := agent.Evaluation

	// Build criteria list for prompt
	criteriaList := e.buildCriteriaList(eval.Criteria)

	// Build evaluation prompt
	prompt := eval.EvaluationPrompt
	prompt = strings.ReplaceAll(prompt, "{{ORIGINAL_TASK}}", originalTask)
	prompt = strings.ReplaceAll(prompt, "{{RESULT}}", result)
	prompt = strings.ReplaceAll(prompt, "{{CRITERIA_LIST}}", criteriaList)

	// Determine model to use
	model := agent.Model
	if eval.EvaluationModel != "" {
		model = eval.EvaluationModel
	}

	e.logger.Debug("Running evaluation",
		"agent", agent.ID,
		"model", model,
		"criteria_count", len(eval.Criteria),
	)

	// Call LLM for evaluation
	resp, err := e.llmClient.Chat(ctx, &ollama.ChatRequest{
		Model: model,
		Messages: []ollama.ChatMessage{
			{
				Role:    "system",
				Content: "Du bist ein Evaluator. Bewerte das Ergebnis objektiv und präzise. Antworte ausschließlich im angegebenen JSON-Format.",
			},
			{
				Role:    "user",
				Content: prompt,
			},
		},
		Options: map[string]interface{}{
			"temperature": 0.1, // Low temperature for consistent evaluation
		},
	})
	if err != nil {
		return nil, fmt.Errorf("evaluation LLM call failed: %w", err)
	}

	// Parse response
	evalResult, err := e.parseEvaluationResponse(resp.Message.Content, eval.Criteria)
	if err != nil {
		e.logger.Warn("Failed to parse evaluation response, falling back to heuristics",
			"error", err,
			"response", resp.Message.Content,
		)
		// Fallback: Use heuristic evaluation
		evalResult = e.heuristicEvaluation(result, eval.Criteria)
	}

	// Check against minimum quality score
	if evalResult.Score < eval.MinQualityScore {
		evalResult.Passed = false
	}

	// Check required criteria
	for _, cr := range evalResult.CriteriaResults {
		if cr.Required && !cr.Passed {
			evalResult.Passed = false
			break
		}
	}

	return evalResult, nil
}

// BuildImprovementPrompt creates the prompt for an improvement iteration
func (e *Evaluator) BuildImprovementPrompt(
	agent *agentloader.AgentYAML,
	originalTask string,
	previousResult string,
	evalResult *agentloader.EvaluationResult,
) string {
	if agent.Evaluation == nil {
		return originalTask
	}

	prompt := agent.Evaluation.ImprovementPrompt
	prompt = strings.ReplaceAll(prompt, "{{ORIGINAL_TASK}}", originalTask)
	prompt = strings.ReplaceAll(prompt, "{{PREVIOUS_RESULT}}", previousResult)
	prompt = strings.ReplaceAll(prompt, "{{EVALUATION_FEEDBACK}}", evalResult.Feedback)

	// Build failed criteria list
	var failedCriteria []string
	for _, cr := range evalResult.CriteriaResults {
		if !cr.Passed {
			failedCriteria = append(failedCriteria, fmt.Sprintf("- %s: %s", cr.Name, cr.Feedback))
		}
	}
	prompt = strings.ReplaceAll(prompt, "{{FAILED_CRITERIA}}", strings.Join(failedCriteria, "\n"))

	return prompt
}

// ShouldIterate determines if another iteration should be attempted
func (e *Evaluator) ShouldIterate(
	agent *agentloader.AgentYAML,
	evalResult *agentloader.EvaluationResult,
	currentIteration int,
) bool {
	if agent.Evaluation == nil || !agent.Evaluation.Enabled {
		return false
	}

	// Already passed - no more iterations needed
	if evalResult.Passed {
		return false
	}

	// Check iteration limit
	if currentIteration >= agent.Evaluation.MaxIterations {
		e.logger.Info("Max iterations reached",
			"agent", agent.ID,
			"iterations", currentIteration,
			"max", agent.Evaluation.MaxIterations,
		)
		return false
	}

	return true
}

// buildCriteriaList formats criteria for the evaluation prompt
func (e *Evaluator) buildCriteriaList(criteria []agentloader.EvaluationCriterion) string {
	var lines []string
	for i, c := range criteria {
		required := ""
		if c.Required {
			required = " [PFLICHT]"
		}
		lines = append(lines, fmt.Sprintf("%d. %s%s: %s", i+1, c.Name, required, c.Check))
	}
	return strings.Join(lines, "\n")
}

// parseEvaluationResponse parses the LLM's JSON response
func (e *Evaluator) parseEvaluationResponse(response string, criteria []agentloader.EvaluationCriterion) (*agentloader.EvaluationResult, error) {
	// Try to extract JSON from response
	response = extractJSON(response)

	var parsed struct {
		Passed          bool    `json:"passed"`
		Score           float32 `json:"score"`
		CriteriaResults []struct {
			Name     string `json:"name"`
			Passed   bool   `json:"passed"`
			Feedback string `json:"feedback"`
		} `json:"criteria_results"`
		Feedback     string   `json:"feedback"`
		Improvements []string `json:"improvements"`
	}

	if err := json.Unmarshal([]byte(response), &parsed); err != nil {
		return nil, fmt.Errorf("JSON parse error: %w", err)
	}

	// Build result
	result := &agentloader.EvaluationResult{
		Passed:       parsed.Passed,
		Score:        parsed.Score,
		Feedback:     parsed.Feedback,
		Improvements: parsed.Improvements,
		Iteration:    1,
	}

	// Map criteria results with required flag from config
	criteriaMap := make(map[string]bool)
	for _, c := range criteria {
		criteriaMap[c.Name] = c.Required
	}

	for _, cr := range parsed.CriteriaResults {
		result.CriteriaResults = append(result.CriteriaResults, agentloader.CriterionResult{
			Name:     cr.Name,
			Passed:   cr.Passed,
			Required: criteriaMap[cr.Name],
			Feedback: cr.Feedback,
		})
	}

	return result, nil
}

// heuristicEvaluation provides a fallback when JSON parsing fails
func (e *Evaluator) heuristicEvaluation(result string, criteria []agentloader.EvaluationCriterion) *agentloader.EvaluationResult {
	// Simple heuristic: check if result is non-empty and reasonably long
	passed := len(result) > 100
	score := float32(0.5)
	if passed {
		score = 0.7
	}

	evalResult := &agentloader.EvaluationResult{
		Passed:    passed,
		Score:     score,
		Feedback:  "Heuristic evaluation (JSON parse failed)",
		Iteration: 1,
	}

	// Mark all criteria as requiring manual review
	for _, c := range criteria {
		evalResult.CriteriaResults = append(evalResult.CriteriaResults, agentloader.CriterionResult{
			Name:     c.Name,
			Passed:   true, // Assume pass for heuristic
			Required: c.Required,
			Feedback: "Manual review recommended",
		})
	}

	return evalResult
}

// extractJSON tries to extract JSON from a response that may contain markdown or other text
func extractJSON(response string) string {
	// Look for JSON block in markdown
	if start := strings.Index(response, "```json"); start != -1 {
		start += 7
		if end := strings.Index(response[start:], "```"); end != -1 {
			return strings.TrimSpace(response[start : start+end])
		}
	}

	// Look for plain JSON block
	if start := strings.Index(response, "```"); start != -1 {
		start += 3
		// Skip optional language identifier
		if nl := strings.Index(response[start:], "\n"); nl != -1 && nl < 20 {
			start += nl + 1
		}
		if end := strings.Index(response[start:], "```"); end != -1 {
			return strings.TrimSpace(response[start : start+end])
		}
	}

	// Look for JSON object
	if start := strings.Index(response, "{"); start != -1 {
		// Find matching closing brace
		depth := 0
		for i := start; i < len(response); i++ {
			switch response[i] {
			case '{':
				depth++
			case '}':
				depth--
				if depth == 0 {
					return response[start : i+1]
				}
			}
		}
	}

	return response
}
