// ============================================================================
// meinDENKWERK (mDW) - Agent Embedding System
// ============================================================================
//
// Dynamische Embedding-Generierung für Agents zur RAG-ähnlichen Agent-Auswahl
// ============================================================================

package agentloader

import (
	"context"
	"fmt"
	"math"
	"strings"
	"sync"

	"github.com/msto63/mDW/pkg/core/logging"
)

// EmbeddingFunc ist die Funktion zum Generieren von Embeddings
// Wird von Turing via BatchEmbed bereitgestellt
type EmbeddingFunc func(ctx context.Context, texts []string) ([][]float64, error)

// AgentEmbedding enthält das Embedding eines Agents
type AgentEmbedding struct {
	AgentID   string    `json:"agent_id"`
	AgentName string    `json:"agent_name"`
	Embedding []float64 `json:"embedding"`
	TextHash  string    `json:"text_hash"` // Hash des Textes für Cache-Invalidierung
}

// EmbeddingRegistry verwaltet Agent-Embeddings mit automatischer Aktualisierung
type EmbeddingRegistry struct {
	mu           sync.RWMutex
	agents       map[string]*AgentYAML // AgentID -> Agent (mit Embedding)
	embeddingFn  EmbeddingFunc
	logger       *logging.Logger
	initialized  bool
	loader       *Loader // Optional: Loader für Persistierung
}

// NewEmbeddingRegistry erstellt eine neue Embedding-Registry
func NewEmbeddingRegistry() *EmbeddingRegistry {
	return &EmbeddingRegistry{
		agents: make(map[string]*AgentYAML),
		logger: logging.New("agent-embedding"),
	}
}

// SetLoader setzt den Loader für Embedding-Persistierung
func (r *EmbeddingRegistry) SetLoader(loader *Loader) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.loader = loader
}

// SetEmbeddingFunc setzt die Embedding-Funktion (von Turing)
func (r *EmbeddingRegistry) SetEmbeddingFunc(fn EmbeddingFunc) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.embeddingFn = fn
	r.initialized = fn != nil
	if r.initialized {
		r.logger.Info("Embedding function configured")
	}
}

// IsInitialized prüft, ob die Embedding-Funktion konfiguriert ist
func (r *EmbeddingRegistry) IsInitialized() bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.initialized
}

// UpdateAgentEmbedding aktualisiert das Embedding eines Agents
// Wird automatisch beim Hot-Reload aufgerufen
// Das Embedding wird direkt im Agent gespeichert und optional persistiert
func (r *EmbeddingRegistry) UpdateAgentEmbedding(ctx context.Context, agent *AgentYAML) error {
	if !r.IsInitialized() {
		// Auch ohne Embedding-Funktion können wir Agents mit bestehenden Embeddings registrieren
		if len(agent.Embedding) > 0 {
			r.mu.Lock()
			r.agents[agent.ID] = agent
			r.mu.Unlock()
			r.logger.Debug("Agent with existing embedding registered", "agent", agent.ID)
		}
		return nil
	}

	// Text für Embedding zusammenstellen
	embeddingText := buildAgentEmbeddingText(agent)
	textHash := hashString(embeddingText)

	// Prüfen ob Embedding im Agent noch aktuell ist
	if len(agent.Embedding) > 0 && agent.EmbeddingHash == textHash {
		// Embedding ist aktuell, nur in Registry registrieren
		r.mu.Lock()
		r.agents[agent.ID] = agent
		r.mu.Unlock()
		r.logger.Debug("Agent embedding unchanged, using cached", "agent", agent.ID)
		return nil
	}

	// Neues Embedding generieren
	embeddings, err := r.embeddingFn(ctx, []string{embeddingText})
	if err != nil {
		return fmt.Errorf("failed to generate embedding for agent %s: %w", agent.ID, err)
	}

	if len(embeddings) == 0 || len(embeddings[0]) == 0 {
		return fmt.Errorf("empty embedding returned for agent %s", agent.ID)
	}

	// Embedding direkt im Agent speichern
	agent.Embedding = embeddings[0]
	agent.EmbeddingHash = textHash

	// In Registry registrieren
	r.mu.Lock()
	r.agents[agent.ID] = agent
	r.mu.Unlock()

	r.logger.Info("Agent embedding updated",
		"agent", agent.ID,
		"name", agent.Name,
		"dimensions", len(embeddings[0]))

	// Optional: Embedding in YAML-Datei persistieren
	if r.loader != nil && agent.SourceFile != "" {
		if err := r.loader.SaveAgent(agent); err != nil {
			r.logger.Warn("Failed to persist agent embedding",
				"agent", agent.ID,
				"error", err)
		} else {
			r.logger.Debug("Agent embedding persisted", "agent", agent.ID)
		}
	}

	return nil
}

// RemoveAgentEmbedding entfernt das Embedding eines Agents
func (r *EmbeddingRegistry) RemoveAgentEmbedding(agentID string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.agents[agentID]; exists {
		delete(r.agents, agentID)
		r.logger.Info("Agent embedding removed", "agent", agentID)
	}
}

// GetAgentEmbedding gibt das Embedding eines Agents zurück
func (r *EmbeddingRegistry) GetAgentEmbedding(agentID string) (*AgentEmbedding, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	agent, exists := r.agents[agentID]
	if !exists || len(agent.Embedding) == 0 {
		return nil, false
	}

	return &AgentEmbedding{
		AgentID:   agent.ID,
		AgentName: agent.Name,
		Embedding: agent.Embedding,
		TextHash:  agent.EmbeddingHash,
	}, true
}

// GetAllEmbeddings gibt alle Agent-Embeddings zurück
func (r *EmbeddingRegistry) GetAllEmbeddings() map[string]*AgentEmbedding {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make(map[string]*AgentEmbedding, len(r.agents))
	for id, agent := range r.agents {
		if len(agent.Embedding) > 0 {
			result[id] = &AgentEmbedding{
				AgentID:   agent.ID,
				AgentName: agent.Name,
				Embedding: agent.Embedding,
				TextHash:  agent.EmbeddingHash,
			}
		}
	}
	return result
}

// AgentMatch repräsentiert ein Agent-Matching-Ergebnis
type AgentMatch struct {
	AgentID    string  `json:"agent_id"`
	AgentName  string  `json:"agent_name"`
	Similarity float64 `json:"similarity"`
}

// FindBestAgentForTask findet den besten Agent für eine Aufgabe
func (r *EmbeddingRegistry) FindBestAgentForTask(ctx context.Context, taskDescription string) (*AgentMatch, error) {
	if !r.IsInitialized() {
		return nil, fmt.Errorf("embedding function not configured")
	}

	// Task-Embedding generieren
	embeddings, err := r.embeddingFn(ctx, []string{taskDescription})
	if err != nil {
		return nil, fmt.Errorf("failed to generate task embedding: %w", err)
	}

	if len(embeddings) == 0 || len(embeddings[0]) == 0 {
		return nil, fmt.Errorf("empty embedding returned for task")
	}

	taskEmbedding := embeddings[0]

	// Beste Übereinstimmung finden
	r.mu.RLock()
	defer r.mu.RUnlock()

	var bestMatch *AgentMatch
	for _, agent := range r.agents {
		if len(agent.Embedding) == 0 {
			continue
		}
		similarity := cosineSimilarity(taskEmbedding, agent.Embedding)

		if bestMatch == nil || similarity > bestMatch.Similarity {
			bestMatch = &AgentMatch{
				AgentID:    agent.ID,
				AgentName:  agent.Name,
				Similarity: similarity,
			}
		}
	}

	if bestMatch == nil {
		return nil, fmt.Errorf("no agents available for matching")
	}

	r.logger.Debug("Best agent match found",
		"task", truncateString(taskDescription, 50),
		"agent", bestMatch.AgentID,
		"similarity", bestMatch.Similarity)

	return bestMatch, nil
}

// FindTopAgentsForTask findet die Top-N Agents für eine Aufgabe
func (r *EmbeddingRegistry) FindTopAgentsForTask(ctx context.Context, taskDescription string, topN int) ([]*AgentMatch, error) {
	if !r.IsInitialized() {
		return nil, fmt.Errorf("embedding function not configured")
	}

	// Task-Embedding generieren
	embeddings, err := r.embeddingFn(ctx, []string{taskDescription})
	if err != nil {
		return nil, fmt.Errorf("failed to generate task embedding: %w", err)
	}

	if len(embeddings) == 0 || len(embeddings[0]) == 0 {
		return nil, fmt.Errorf("empty embedding returned for task")
	}

	taskEmbedding := embeddings[0]

	// Alle Similarities berechnen
	r.mu.RLock()
	matches := make([]*AgentMatch, 0, len(r.agents))
	for _, agent := range r.agents {
		if len(agent.Embedding) == 0 {
			continue
		}
		similarity := cosineSimilarity(taskEmbedding, agent.Embedding)
		matches = append(matches, &AgentMatch{
			AgentID:    agent.ID,
			AgentName:  agent.Name,
			Similarity: similarity,
		})
	}
	r.mu.RUnlock()

	// Sortieren nach Similarity (absteigend)
	sortAgentMatches(matches)

	// Top-N zurückgeben
	if topN > len(matches) {
		topN = len(matches)
	}

	return matches[:topN], nil
}

// buildAgentEmbeddingText erstellt den Text für das Agent-Embedding
func buildAgentEmbeddingText(agent *AgentYAML) string {
	var parts []string

	// Name und Beschreibung
	parts = append(parts, fmt.Sprintf("Agent-Name: %s", agent.Name))
	if agent.Description != "" {
		parts = append(parts, fmt.Sprintf("Beschreibung: %s", agent.Description))
	}

	// Tools als Fähigkeiten
	if len(agent.Tools) > 0 {
		toolNames := make([]string, len(agent.Tools))
		for i, t := range agent.Tools {
			toolNames[i] = t.Name
		}
		parts = append(parts, fmt.Sprintf("Verfügbare Tools: %s", strings.Join(toolNames, ", ")))
	}

	// Metadata-Tags
	if tags, ok := agent.Metadata["tags"]; ok {
		parts = append(parts, fmt.Sprintf("Schlüsselwörter: %s", tags))
	}
	if category, ok := agent.Metadata["category"]; ok {
		parts = append(parts, fmt.Sprintf("Kategorie: %s", category))
	}

	// Spezialisierung aus System-Prompt extrahieren (erste 500 Zeichen)
	if agent.SystemPrompt != "" {
		// Erste Zeilen des System-Prompts für Kontext
		promptPreview := extractPromptEssence(agent.SystemPrompt)
		if promptPreview != "" {
			parts = append(parts, fmt.Sprintf("Spezialisierung: %s", promptPreview))
		}
	}

	return strings.Join(parts, "\n")
}

// extractPromptEssence extrahiert die wesentlichen Teile des System-Prompts
func extractPromptEssence(prompt string) string {
	// Erste 500 Zeichen, aber nur bis zum letzten vollständigen Satz
	if len(prompt) <= 500 {
		return prompt
	}

	excerpt := prompt[:500]
	// Finde letzten Satzende-Punkt
	lastDot := strings.LastIndex(excerpt, ".")
	if lastDot > 100 {
		return excerpt[:lastDot+1]
	}

	// Fallback: Letzte Newline
	lastNewline := strings.LastIndex(excerpt, "\n")
	if lastNewline > 100 {
		return excerpt[:lastNewline]
	}

	return excerpt + "..."
}

// cosineSimilarity berechnet die Kosinus-Ähnlichkeit zwischen zwei Vektoren
func cosineSimilarity(a, b []float64) float64 {
	if len(a) != len(b) || len(a) == 0 {
		return 0
	}

	var dotProduct, normA, normB float64
	for i := range a {
		dotProduct += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}

	if normA == 0 || normB == 0 {
		return 0
	}

	return dotProduct / (math.Sqrt(normA) * math.Sqrt(normB))
}

// hashString erstellt einen einfachen Hash eines Strings
func hashString(s string) string {
	// Einfacher FNV-1a Hash
	var hash uint64 = 14695981039346656037
	for i := 0; i < len(s); i++ {
		hash ^= uint64(s[i])
		hash *= 1099511628211
	}
	return fmt.Sprintf("%x", hash)
}

// truncateString kürzt einen String auf maxLen Zeichen
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// sortAgentMatches sortiert Matches nach Similarity (absteigend)
func sortAgentMatches(matches []*AgentMatch) {
	// Simple insertion sort (für kleine Listen ausreichend)
	for i := 1; i < len(matches); i++ {
		key := matches[i]
		j := i - 1
		for j >= 0 && matches[j].Similarity < key.Similarity {
			matches[j+1] = matches[j]
			j--
		}
		matches[j+1] = key
	}
}
