// ============================================================================
// meinDENKWERK (mDW) - Task Decomposer
// ============================================================================
//
// Zerlegt komplexe User-Prompts in atomare Aufgaben für Agent-Verkettung
// ============================================================================

package decomposer

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/msto63/mDW/pkg/core/logging"
)

// LLMFunc ist die Funktion zum Aufrufen des LLMs
type LLMFunc func(ctx context.Context, systemPrompt, userPrompt string) (string, error)

// Task repräsentiert eine einzelne, atomare Aufgabe
type Task struct {
	ID           string   `json:"id"`
	Description  string   `json:"description"`
	Dependencies []string `json:"dependencies,omitempty"` // IDs der vorherigen Tasks
	Priority     int      `json:"priority,omitempty"`
}

// DecompositionResult enthält das Ergebnis der Prompt-Zerlegung
type DecompositionResult struct {
	OriginalPrompt string   `json:"original_prompt"`
	Tasks          []*Task  `json:"tasks"`
	IsSequential   bool     `json:"is_sequential"`
	CanParallelize []string `json:"can_parallelize,omitempty"` // Task-IDs die parallel laufen können
	Reasoning      string   `json:"reasoning,omitempty"`
}

// Decomposer zerlegt User-Prompts in Einzelaufgaben
type Decomposer struct {
	llmFunc LLMFunc
	logger  *logging.Logger
}

// NewDecomposer erstellt einen neuen Decomposer
func NewDecomposer() *Decomposer {
	return &Decomposer{
		logger: logging.New("decomposer"),
	}
}

// SetLLMFunc setzt die LLM-Funktion
func (d *Decomposer) SetLLMFunc(fn LLMFunc) {
	d.llmFunc = fn
}

// Decompose zerlegt einen Prompt in einzelne Aufgaben
func (d *Decomposer) Decompose(ctx context.Context, prompt string) (*DecompositionResult, error) {
	if d.llmFunc == nil {
		// Fallback: Einfache Heuristik ohne LLM
		return d.decomposeHeuristic(prompt), nil
	}

	start := time.Now()

	// LLM-basierte Zerlegung
	result, err := d.decomposeLLM(ctx, prompt)
	if err != nil {
		d.logger.Warn("LLM decomposition failed, using heuristic", "error", err)
		return d.decomposeHeuristic(prompt), nil
	}

	d.logger.Info("Prompt decomposed",
		"tasks", len(result.Tasks),
		"sequential", result.IsSequential,
		"duration_ms", time.Since(start).Milliseconds())

	return result, nil
}

// decomposeLLM nutzt das LLM zur Prompt-Zerlegung
func (d *Decomposer) decomposeLLM(ctx context.Context, prompt string) (*DecompositionResult, error) {
	systemPrompt := `Du bist ein Experte für die Analyse und Zerlegung von Aufgaben.

Analysiere den folgenden User-Prompt und zerlege ihn in einzelne, atomare Aufgaben.
Identifiziere Abhängigkeiten zwischen den Aufgaben.

WICHTIG:
- Jede Task sollte eine einzelne, klar definierte Aufgabe sein
- Erkenne Verben wie "recherchiere", "fasse zusammen", "übersetze", "analysiere", "schreibe", "berechne"
- Wenn Aufgaben aufeinander aufbauen, setze Dependencies entsprechend
- Einfache Prompts ohne mehrere Schritte → Nur eine Task

Antworte NUR mit einem JSON-Objekt im folgenden Format:
{
  "tasks": [
    {"id": "task_1", "description": "Beschreibung der ersten Aufgabe", "dependencies": []},
    {"id": "task_2", "description": "Beschreibung der zweiten Aufgabe", "dependencies": ["task_1"]}
  ],
  "is_sequential": true,
  "reasoning": "Kurze Erklärung der Zerlegung"
}

Gib NUR das JSON zurück, keinen anderen Text.`

	response, err := d.llmFunc(ctx, systemPrompt, prompt)
	if err != nil {
		return nil, fmt.Errorf("LLM call failed: %w", err)
	}

	// JSON extrahieren (falls LLM zusätzlichen Text generiert)
	jsonStr := extractJSON(response)
	if jsonStr == "" {
		return nil, fmt.Errorf("no valid JSON in response")
	}

	// Parsen
	var llmResult struct {
		Tasks        []*Task `json:"tasks"`
		IsSequential bool    `json:"is_sequential"`
		Reasoning    string  `json:"reasoning"`
	}

	if err := json.Unmarshal([]byte(jsonStr), &llmResult); err != nil {
		return nil, fmt.Errorf("failed to parse LLM response: %w", err)
	}

	// Validierung
	if len(llmResult.Tasks) == 0 {
		// Fallback: Eine Task für den gesamten Prompt
		llmResult.Tasks = []*Task{
			{ID: "task_1", Description: prompt, Dependencies: []string{}},
		}
	}

	return &DecompositionResult{
		OriginalPrompt: prompt,
		Tasks:          llmResult.Tasks,
		IsSequential:   llmResult.IsSequential,
		Reasoning:      llmResult.Reasoning,
	}, nil
}

// decomposeHeuristic nutzt einfache Regeln zur Zerlegung (Fallback)
func (d *Decomposer) decomposeHeuristic(prompt string) *DecompositionResult {
	lower := strings.ToLower(prompt)
	var tasks []*Task
	taskID := 0

	// Schlüsselwörter für verschiedene Aufgabentypen
	patterns := []struct {
		keywords    []string
		description string
	}{
		{[]string{"recherchiere", "suche", "finde heraus", "search", "research"}, "Informationen recherchieren"},
		{[]string{"zusammenfass", "fasse zusammen", "summary", "summarize"}, "Informationen zusammenfassen"},
		{[]string{"übersetz", "translate"}, "Text übersetzen"},
		{[]string{"analysiere", "analyze", "prüfe", "check"}, "Analyse durchführen"},
		{[]string{"schreibe", "erstelle", "write", "create"}, "Text erstellen"},
		{[]string{"berechne", "calculate", "rechne"}, "Berechnung durchführen"},
		{[]string{"vergleiche", "compare"}, "Vergleich erstellen"},
		{[]string{"erkläre", "explain"}, "Erklärung geben"},
	}

	// Suche nach Schlüsselwörtern
	foundPatterns := make([]string, 0)
	for _, p := range patterns {
		for _, kw := range p.keywords {
			if strings.Contains(lower, kw) {
				foundPatterns = append(foundPatterns, p.description)
				break
			}
		}
	}

	// Suche nach Konjunktionen die auf mehrere Aufgaben hindeuten
	hasMultipleIndicators := strings.Contains(lower, " und dann ") ||
		strings.Contains(lower, " danach ") ||
		strings.Contains(lower, ", dann ") ||
		strings.Contains(lower, " anschließend ")

	if len(foundPatterns) > 1 || hasMultipleIndicators {
		// Mehrere Aufgaben erkannt
		deps := []string{}
		for _, desc := range foundPatterns {
			taskID++
			task := &Task{
				ID:           fmt.Sprintf("task_%d", taskID),
				Description:  desc + " basierend auf: " + truncate(prompt, 100),
				Dependencies: append([]string{}, deps...),
			}
			tasks = append(tasks, task)
			deps = []string{task.ID}
		}
	}

	// Fallback: Eine einzelne Task
	if len(tasks) == 0 {
		tasks = []*Task{
			{
				ID:           "task_1",
				Description:  prompt,
				Dependencies: []string{},
			},
		}
	}

	return &DecompositionResult{
		OriginalPrompt: prompt,
		Tasks:          tasks,
		IsSequential:   len(tasks) > 1,
		Reasoning:      "Heuristische Zerlegung basierend auf Schlüsselwörtern",
	}
}

// IsSimplePrompt prüft ob ein Prompt einfach ist (keine Zerlegung nötig)
func (d *Decomposer) IsSimplePrompt(prompt string) bool {
	lower := strings.ToLower(prompt)

	// Kurze Prompts sind meist einfach
	if len(prompt) < 100 {
		return true
	}

	// Keine Konjunktionen die auf mehrere Schritte hindeuten
	multiStepIndicators := []string{
		" und dann ", " danach ", ", dann ", " anschließend ",
		" zuerst ", " erstens ", " zweitens ",
		" schritt ", " step ",
	}

	for _, indicator := range multiStepIndicators {
		if strings.Contains(lower, indicator) {
			return false
		}
	}

	// Zähle verschiedene Aktionsverben
	actionCount := 0
	actionVerbs := []string{
		"recherchiere", "suche", "finde",
		"fasse zusammen", "zusammenfass",
		"übersetz",
		"analysiere", "prüfe",
		"schreibe", "erstelle",
		"berechne", "rechne",
		"vergleiche",
	}

	for _, verb := range actionVerbs {
		if strings.Contains(lower, verb) {
			actionCount++
		}
	}

	return actionCount <= 1
}

// Helper Functions

// extractJSON extrahiert JSON aus einer LLM-Antwort
func extractJSON(s string) string {
	// Finde den ersten '{' und letzten '}'
	start := strings.Index(s, "{")
	end := strings.LastIndex(s, "}")

	if start == -1 || end == -1 || end <= start {
		return ""
	}

	return s[start : end+1]
}

// truncate kürzt einen String
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
