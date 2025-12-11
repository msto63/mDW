// ============================================================================
// meinDENKWERK (mDW) - Lokale KI-Plattform
// ============================================================================
//
// Package:     agentloader
// Description: YAML agent loader with hot-reload support
// Author:      Mike Stoffels with Claude
// Created:     2025-12-11
// License:     MIT
// ============================================================================

package agentloader

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/msto63/mDW/pkg/core/logging"
	"gopkg.in/yaml.v3"
)

// Loader manages loading and hot-reloading of agent definitions from YAML files
type Loader struct {
	mu                sync.RWMutex
	agents            map[string]*AgentYAML // id -> agent
	agentsDir         string
	watcher           *fsnotify.Watcher
	logger            *logging.Logger
	onChange          func(agentID string, agent *AgentYAML) // Callback when agent changes
	onDelete          func(agentID string)                   // Callback when agent is deleted
	stopCh            chan struct{}
	running           bool
	embeddingRegistry *EmbeddingRegistry // Embedding-Registry für Agent-Matching
}

// NewLoader creates a new agent loader
func NewLoader(agentsDir string) *Loader {
	l := &Loader{
		agents:            make(map[string]*AgentYAML),
		agentsDir:         agentsDir,
		logger:            logging.New("agentloader"),
		stopCh:            make(chan struct{}),
		embeddingRegistry: NewEmbeddingRegistry(),
	}
	// Verbinde EmbeddingRegistry mit Loader für Persistierung
	l.embeddingRegistry.SetLoader(l)
	return l
}

// EmbeddingRegistry returns the embedding registry
func (l *Loader) EmbeddingRegistry() *EmbeddingRegistry {
	return l.embeddingRegistry
}

// SetEmbeddingFunc sets the embedding function for agent matching
func (l *Loader) SetEmbeddingFunc(fn EmbeddingFunc) {
	l.embeddingRegistry.SetEmbeddingFunc(fn)
}

// UpdateAllEmbeddings updates embeddings for all loaded agents
func (l *Loader) UpdateAllEmbeddings(ctx context.Context) error {
	l.mu.RLock()
	agents := make([]*AgentYAML, 0, len(l.agents))
	for _, agent := range l.agents {
		agents = append(agents, agent)
	}
	l.mu.RUnlock()

	for _, agent := range agents {
		if err := l.embeddingRegistry.UpdateAgentEmbedding(ctx, agent); err != nil {
			l.logger.Warn("Failed to update embedding for agent", "agent", agent.ID, "error", err)
			// Continue with other agents
		}
	}

	l.logger.Info("Agent embeddings updated", "count", len(agents))
	return nil
}

// FindBestAgentForTask finds the best matching agent for a task description
func (l *Loader) FindBestAgentForTask(ctx context.Context, taskDescription string) (*AgentMatch, error) {
	return l.embeddingRegistry.FindBestAgentForTask(ctx, taskDescription)
}

// FindTopAgentsForTask finds the top N matching agents for a task
func (l *Loader) FindTopAgentsForTask(ctx context.Context, taskDescription string, topN int) ([]*AgentMatch, error) {
	return l.embeddingRegistry.FindTopAgentsForTask(ctx, taskDescription, topN)
}

// SetOnChange sets the callback for when an agent is loaded or updated
func (l *Loader) SetOnChange(fn func(agentID string, agent *AgentYAML)) {
	l.onChange = fn
}

// SetOnDelete sets the callback for when an agent is deleted
func (l *Loader) SetOnDelete(fn func(agentID string)) {
	l.onDelete = fn
}

// LoadAll loads all agent YAML files from the directory
func (l *Loader) LoadAll() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	// Ensure directory exists
	if err := os.MkdirAll(l.agentsDir, 0755); err != nil {
		return fmt.Errorf("failed to create agents directory: %w", err)
	}

	// Find all YAML files
	files, err := filepath.Glob(filepath.Join(l.agentsDir, "*.yaml"))
	if err != nil {
		return fmt.Errorf("failed to list agent files: %w", err)
	}

	// Also check .yml extension
	ymlFiles, _ := filepath.Glob(filepath.Join(l.agentsDir, "*.yml"))
	files = append(files, ymlFiles...)

	if len(files) == 0 {
		l.logger.Info("No agent files found in directory", "dir", l.agentsDir)
		return nil
	}

	// Load each file
	loadedCount := 0
	for _, file := range files {
		agent, err := l.loadFile(file)
		if err != nil {
			l.logger.Warn("Failed to load agent file", "file", file, "error", err)
			continue
		}

		l.agents[agent.ID] = agent
		loadedCount++
		l.logger.Info("Agent loaded", "id", agent.ID, "name", agent.Name, "file", filepath.Base(file))
	}

	l.logger.Info("Agents loaded from directory", "count", loadedCount, "dir", l.agentsDir)
	return nil
}

// loadFile loads a single YAML file
func (l *Loader) loadFile(path string) (*AgentYAML, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	var agent AgentYAML
	if err := yaml.Unmarshal(data, &agent); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidYAML, err)
	}

	// Apply defaults
	agent.Defaults()

	// Process dynamic placeholders ({{DATE}}, {{YEAR}}, etc.)
	agent.ProcessPlaceholders()

	// Validate
	if err := agent.Validate(); err != nil {
		return nil, err
	}

	// Set internal tracking
	agent.SourceFile = path
	agent.LoadedAt = time.Now()

	return &agent, nil
}

// Get returns an agent by ID
func (l *Loader) Get(id string) (*AgentYAML, bool) {
	l.mu.RLock()
	defer l.mu.RUnlock()

	agent, ok := l.agents[id]
	return agent, ok
}

// GetAll returns all loaded agents
func (l *Loader) GetAll() []*AgentYAML {
	l.mu.RLock()
	defer l.mu.RUnlock()

	agents := make([]*AgentYAML, 0, len(l.agents))
	for _, agent := range l.agents {
		agents = append(agents, agent)
	}
	return agents
}

// StartWatching starts the file watcher for hot-reload
func (l *Loader) StartWatching(ctx context.Context) error {
	if l.running {
		return nil
	}

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("failed to create watcher: %w", err)
	}

	l.watcher = watcher

	// Watch the agents directory
	if err := l.watcher.Add(l.agentsDir); err != nil {
		l.watcher.Close()
		return fmt.Errorf("failed to watch directory: %w", err)
	}

	l.running = true
	l.logger.Info("Started watching for agent changes", "dir", l.agentsDir)

	go l.watchLoop(ctx)

	return nil
}

// watchLoop handles file system events
func (l *Loader) watchLoop(ctx context.Context) {
	defer func() {
		l.running = false
		if l.watcher != nil {
			l.watcher.Close()
		}
	}()

	// Debounce map to prevent multiple reloads for the same file
	debounce := make(map[string]time.Time)
	debounceDelay := 500 * time.Millisecond

	for {
		select {
		case <-ctx.Done():
			l.logger.Info("Stopping file watcher (context cancelled)")
			return

		case <-l.stopCh:
			l.logger.Info("Stopping file watcher (stop signal)")
			return

		case event, ok := <-l.watcher.Events:
			if !ok {
				return
			}

			// Only process YAML files
			if !isYAMLFile(event.Name) {
				continue
			}

			// Debounce: skip if we just processed this file
			if lastTime, exists := debounce[event.Name]; exists {
				if time.Since(lastTime) < debounceDelay {
					continue
				}
			}
			debounce[event.Name] = time.Now()

			l.handleFileEvent(event)

		case err, ok := <-l.watcher.Errors:
			if !ok {
				return
			}
			l.logger.Error("Watcher error", "error", err)
		}
	}
}

// handleFileEvent processes a single file event
func (l *Loader) handleFileEvent(event fsnotify.Event) {
	fileName := filepath.Base(event.Name)

	switch {
	case event.Op&fsnotify.Create == fsnotify.Create || event.Op&fsnotify.Write == fsnotify.Write:
		// File created or modified - reload it
		l.logger.Info("Agent file changed, reloading", "file", fileName, "op", event.Op.String())

		agent, err := l.loadFile(event.Name)
		if err != nil {
			l.logger.Error("Failed to reload agent", "file", fileName, "error", err)
			return
		}

		l.mu.Lock()
		l.agents[agent.ID] = agent
		l.mu.Unlock()

		l.logger.Info("Agent reloaded", "id", agent.ID, "name", agent.Name)

		// Automatisch Embedding aktualisieren (async)
		go func(a *AgentYAML) {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			if err := l.embeddingRegistry.UpdateAgentEmbedding(ctx, a); err != nil {
				l.logger.Warn("Failed to update embedding on hot-reload", "agent", a.ID, "error", err)
			} else {
				l.logger.Info("Agent embedding updated on hot-reload", "agent", a.ID)
			}
		}(agent)

		if l.onChange != nil {
			l.onChange(agent.ID, agent)
		}

	case event.Op&fsnotify.Remove == fsnotify.Remove || event.Op&fsnotify.Rename == fsnotify.Rename:
		// File removed - find and remove the agent
		l.mu.Lock()
		var removedID string
		for id, agent := range l.agents {
			if agent.SourceFile == event.Name {
				removedID = id
				delete(l.agents, id)
				break
			}
		}
		l.mu.Unlock()

		if removedID != "" {
			l.logger.Info("Agent removed", "id", removedID, "file", fileName)

			// Embedding entfernen
			l.embeddingRegistry.RemoveAgentEmbedding(removedID)

			if l.onDelete != nil {
				l.onDelete(removedID)
			}
		}
	}
}

// Stop stops the file watcher
func (l *Loader) Stop() {
	if l.running {
		close(l.stopCh)
	}
}

// SaveAgent saves an agent definition to a YAML file
// Uses the existing SourceFile if present, otherwise creates a new file
func (l *Loader) SaveAgent(agent *AgentYAML) error {
	// Determine file path - use existing SourceFile if available
	var filePath string
	if agent.SourceFile != "" {
		filePath = agent.SourceFile
	} else {
		fileName := fmt.Sprintf("%s.yaml", sanitizeFileName(agent.ID))
		filePath = filepath.Join(l.agentsDir, fileName)
	}

	// Marshal to YAML
	data, err := yaml.Marshal(agent)
	if err != nil {
		return fmt.Errorf("failed to marshal agent: %w", err)
	}

	// Write file
	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write agent file: %w", err)
	}

	// Update internal state
	agent.SourceFile = filePath
	agent.LoadedAt = time.Now()

	l.mu.Lock()
	l.agents[agent.ID] = agent
	l.mu.Unlock()

	l.logger.Info("Agent saved", "id", agent.ID, "file", filepath.Base(filePath))
	return nil
}

// DeleteAgent deletes an agent file
func (l *Loader) DeleteAgent(id string) error {
	l.mu.Lock()
	agent, exists := l.agents[id]
	if !exists {
		l.mu.Unlock()
		return ErrAgentNotFound
	}

	filePath := agent.SourceFile
	delete(l.agents, id)
	l.mu.Unlock()

	if filePath != "" {
		if err := os.Remove(filePath); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("failed to delete agent file: %w", err)
		}
	}

	l.logger.Info("Agent deleted", "id", id)
	return nil
}

// GetDirectory returns the agents directory path
func (l *Loader) GetDirectory() string {
	return l.agentsDir
}

// Helper functions

func isYAMLFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	return ext == ".yaml" || ext == ".yml"
}

func sanitizeFileName(s string) string {
	// Replace unsafe characters
	replacer := strings.NewReplacer(
		"/", "-",
		"\\", "-",
		":", "-",
		"*", "-",
		"?", "-",
		"\"", "-",
		"<", "-",
		">", "-",
		"|", "-",
		" ", "_",
	)
	return replacer.Replace(s)
}
