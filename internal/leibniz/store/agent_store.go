package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// AgentDefinition represents a stored agent definition
type AgentDefinition struct {
	ID           string            `json:"id"`
	Name         string            `json:"name"`
	Description  string            `json:"description"`
	SystemPrompt string            `json:"system_prompt"`
	Tools        []string          `json:"tools,omitempty"`
	Model        string            `json:"model"`
	MaxSteps     int               `json:"max_steps"`
	Timeout      time.Duration     `json:"timeout"`
	Metadata     map[string]string `json:"metadata,omitempty"`
	CreatedAt    time.Time         `json:"created_at"`
	UpdatedAt    time.Time         `json:"updated_at"`
}

// ExecutionRecord represents an execution history record
type ExecutionRecord struct {
	ID          string     `json:"id"`
	AgentID     string     `json:"agent_id"`
	Message     string     `json:"message"`
	Status      string     `json:"status"` // running, completed, error, cancelled
	Result      string     `json:"result"`
	Error       string     `json:"error,omitempty"`
	Steps       []StepInfo `json:"steps,omitempty"`
	ToolsUsed   []string   `json:"tools_used,omitempty"`
	StartedAt   time.Time  `json:"started_at"`
	CompletedAt time.Time  `json:"completed_at,omitempty"`
	Duration    int64      `json:"duration_ms"`
}

// StepInfo represents a single execution step
type StepInfo struct {
	Index      int       `json:"index"`
	Thought    string    `json:"thought"`
	Action     string    `json:"action"`
	ToolName   string    `json:"tool_name,omitempty"`
	ToolInput  string    `json:"tool_input,omitempty"`
	ToolOutput string    `json:"tool_output,omitempty"`
	Timestamp  time.Time `json:"timestamp"`
}

// AgentStore defines the interface for agent persistence
type AgentStore interface {
	// Agent operations
	CreateAgent(ctx context.Context, agent *AgentDefinition) error
	GetAgent(ctx context.Context, id string) (*AgentDefinition, error)
	UpdateAgent(ctx context.Context, agent *AgentDefinition) error
	DeleteAgent(ctx context.Context, id string) error
	ListAgents(ctx context.Context) ([]*AgentDefinition, error)

	// Execution operations
	CreateExecution(ctx context.Context, exec *ExecutionRecord) error
	UpdateExecution(ctx context.Context, exec *ExecutionRecord) error
	GetExecution(ctx context.Context, id string) (*ExecutionRecord, error)
	ListExecutions(ctx context.Context, agentID string, limit, offset int) ([]*ExecutionRecord, error)
	DeleteExecution(ctx context.Context, id string) error

	// Utility
	Close() error
	Statistics(ctx context.Context) (map[string]interface{}, error)
}

// SQLiteAgentStore implements AgentStore using SQLite
type SQLiteAgentStore struct {
	db *sql.DB
	mu sync.RWMutex
}

// SQLiteAgentConfig holds configuration for SQLite store
type SQLiteAgentConfig struct {
	Path string
}

// DefaultAgentConfig returns default configuration
func DefaultAgentConfig() SQLiteAgentConfig {
	return SQLiteAgentConfig{
		Path: "./data/agents.db",
	}
}

// NewSQLiteAgentStore creates a new SQLite-based agent store
func NewSQLiteAgentStore(cfg SQLiteAgentConfig) (*SQLiteAgentStore, error) {
	// Ensure directory exists
	dir := filepath.Dir(cfg.Path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create directory: %w", err)
	}

	// Open database with WAL mode
	db, err := sql.Open("sqlite3", cfg.Path+"?_journal_mode=WAL&_synchronous=NORMAL")
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	store := &SQLiteAgentStore{db: db}

	if err := store.initSchema(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to initialize schema: %w", err)
	}

	return store, nil
}

// initSchema creates the necessary tables
func (s *SQLiteAgentStore) initSchema() error {
	schema := `
	-- Agent definitions table
	CREATE TABLE IF NOT EXISTS agents (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		description TEXT NOT NULL DEFAULT '',
		system_prompt TEXT NOT NULL DEFAULT '',
		tools TEXT,
		model TEXT NOT NULL DEFAULT '',
		max_steps INTEGER NOT NULL DEFAULT 10,
		timeout_ms INTEGER NOT NULL DEFAULT 120000,
		metadata TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	-- Execution records table
	CREATE TABLE IF NOT EXISTS executions (
		id TEXT PRIMARY KEY,
		agent_id TEXT NOT NULL,
		message TEXT NOT NULL,
		status TEXT NOT NULL DEFAULT 'running',
		result TEXT,
		error TEXT,
		steps TEXT,
		tools_used TEXT,
		started_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		completed_at DATETIME,
		duration_ms INTEGER DEFAULT 0,
		FOREIGN KEY (agent_id) REFERENCES agents(id) ON DELETE CASCADE
	);

	-- Indices
	CREATE INDEX IF NOT EXISTS idx_executions_agent ON executions(agent_id);
	CREATE INDEX IF NOT EXISTS idx_executions_status ON executions(status);
	CREATE INDEX IF NOT EXISTS idx_executions_started ON executions(started_at DESC);
	`

	_, err := s.db.Exec(schema)
	return err
}

// CreateAgent creates a new agent definition
func (s *SQLiteAgentStore) CreateAgent(ctx context.Context, agent *AgentDefinition) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if agent.ID == "" {
		return fmt.Errorf("agent ID is required")
	}

	now := time.Now()
	if agent.CreatedAt.IsZero() {
		agent.CreatedAt = now
	}
	agent.UpdatedAt = now

	toolsJSON, _ := json.Marshal(agent.Tools)
	metadataJSON, _ := json.Marshal(agent.Metadata)

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO agents (id, name, description, system_prompt, tools, model, max_steps, timeout_ms, metadata, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, agent.ID, agent.Name, agent.Description, agent.SystemPrompt, toolsJSON, agent.Model,
		agent.MaxSteps, agent.Timeout.Milliseconds(), metadataJSON, agent.CreatedAt, agent.UpdatedAt)

	if err != nil {
		return fmt.Errorf("failed to create agent: %w", err)
	}

	return nil
}

// GetAgent retrieves an agent by ID
func (s *SQLiteAgentStore) GetAgent(ctx context.Context, id string) (*AgentDefinition, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	row := s.db.QueryRowContext(ctx, `
		SELECT id, name, description, system_prompt, tools, model, max_steps, timeout_ms, metadata, created_at, updated_at
		FROM agents WHERE id = ?
	`, id)

	var agent AgentDefinition
	var toolsJSON, metadataJSON sql.NullString
	var timeoutMs int64

	err := row.Scan(&agent.ID, &agent.Name, &agent.Description, &agent.SystemPrompt,
		&toolsJSON, &agent.Model, &agent.MaxSteps, &timeoutMs, &metadataJSON,
		&agent.CreatedAt, &agent.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get agent: %w", err)
	}

	agent.Timeout = time.Duration(timeoutMs) * time.Millisecond

	if toolsJSON.Valid {
		json.Unmarshal([]byte(toolsJSON.String), &agent.Tools)
	}
	if metadataJSON.Valid {
		json.Unmarshal([]byte(metadataJSON.String), &agent.Metadata)
	}

	return &agent, nil
}

// UpdateAgent updates an agent definition
func (s *SQLiteAgentStore) UpdateAgent(ctx context.Context, agent *AgentDefinition) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	agent.UpdatedAt = time.Now()

	toolsJSON, _ := json.Marshal(agent.Tools)
	metadataJSON, _ := json.Marshal(agent.Metadata)

	result, err := s.db.ExecContext(ctx, `
		UPDATE agents
		SET name = ?, description = ?, system_prompt = ?, tools = ?, model = ?,
			max_steps = ?, timeout_ms = ?, metadata = ?, updated_at = ?
		WHERE id = ?
	`, agent.Name, agent.Description, agent.SystemPrompt, toolsJSON, agent.Model,
		agent.MaxSteps, agent.Timeout.Milliseconds(), metadataJSON, agent.UpdatedAt, agent.ID)

	if err != nil {
		return fmt.Errorf("failed to update agent: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("agent not found: %s", agent.ID)
	}

	return nil
}

// DeleteAgent deletes an agent and its executions
func (s *SQLiteAgentStore) DeleteAgent(ctx context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	_, err := s.db.ExecContext(ctx, `DELETE FROM agents WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("failed to delete agent: %w", err)
	}

	return nil
}

// ListAgents returns all agents
func (s *SQLiteAgentStore) ListAgents(ctx context.Context) ([]*AgentDefinition, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	rows, err := s.db.QueryContext(ctx, `
		SELECT id, name, description, system_prompt, tools, model, max_steps, timeout_ms, metadata, created_at, updated_at
		FROM agents ORDER BY name
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to list agents: %w", err)
	}
	defer rows.Close()

	var agents []*AgentDefinition
	for rows.Next() {
		var agent AgentDefinition
		var toolsJSON, metadataJSON sql.NullString
		var timeoutMs int64

		if err := rows.Scan(&agent.ID, &agent.Name, &agent.Description, &agent.SystemPrompt,
			&toolsJSON, &agent.Model, &agent.MaxSteps, &timeoutMs, &metadataJSON,
			&agent.CreatedAt, &agent.UpdatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan agent: %w", err)
		}

		agent.Timeout = time.Duration(timeoutMs) * time.Millisecond

		if toolsJSON.Valid {
			json.Unmarshal([]byte(toolsJSON.String), &agent.Tools)
		}
		if metadataJSON.Valid {
			json.Unmarshal([]byte(metadataJSON.String), &agent.Metadata)
		}

		agents = append(agents, &agent)
	}

	return agents, nil
}

// CreateExecution creates a new execution record
func (s *SQLiteAgentStore) CreateExecution(ctx context.Context, exec *ExecutionRecord) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if exec.ID == "" {
		return fmt.Errorf("execution ID is required")
	}

	if exec.StartedAt.IsZero() {
		exec.StartedAt = time.Now()
	}

	stepsJSON, _ := json.Marshal(exec.Steps)
	toolsUsedJSON, _ := json.Marshal(exec.ToolsUsed)

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO executions (id, agent_id, message, status, result, error, steps, tools_used, started_at, completed_at, duration_ms)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, exec.ID, exec.AgentID, exec.Message, exec.Status, exec.Result, exec.Error,
		stepsJSON, toolsUsedJSON, exec.StartedAt, nullTime(exec.CompletedAt), exec.Duration)

	if err != nil {
		return fmt.Errorf("failed to create execution: %w", err)
	}

	return nil
}

// UpdateExecution updates an execution record
func (s *SQLiteAgentStore) UpdateExecution(ctx context.Context, exec *ExecutionRecord) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	stepsJSON, _ := json.Marshal(exec.Steps)
	toolsUsedJSON, _ := json.Marshal(exec.ToolsUsed)

	result, err := s.db.ExecContext(ctx, `
		UPDATE executions
		SET status = ?, result = ?, error = ?, steps = ?, tools_used = ?, completed_at = ?, duration_ms = ?
		WHERE id = ?
	`, exec.Status, exec.Result, exec.Error, stepsJSON, toolsUsedJSON,
		nullTime(exec.CompletedAt), exec.Duration, exec.ID)

	if err != nil {
		return fmt.Errorf("failed to update execution: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("execution not found: %s", exec.ID)
	}

	return nil
}

// GetExecution retrieves an execution by ID
func (s *SQLiteAgentStore) GetExecution(ctx context.Context, id string) (*ExecutionRecord, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	row := s.db.QueryRowContext(ctx, `
		SELECT id, agent_id, message, status, result, error, steps, tools_used, started_at, completed_at, duration_ms
		FROM executions WHERE id = ?
	`, id)

	return s.scanExecution(row)
}

// ListExecutions returns executions for an agent
func (s *SQLiteAgentStore) ListExecutions(ctx context.Context, agentID string, limit, offset int) ([]*ExecutionRecord, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if limit <= 0 {
		limit = 50
	}

	var query string
	var args []interface{}

	if agentID != "" {
		query = `
			SELECT id, agent_id, message, status, result, error, steps, tools_used, started_at, completed_at, duration_ms
			FROM executions
			WHERE agent_id = ?
			ORDER BY started_at DESC
			LIMIT ? OFFSET ?
		`
		args = []interface{}{agentID, limit, offset}
	} else {
		query = `
			SELECT id, agent_id, message, status, result, error, steps, tools_used, started_at, completed_at, duration_ms
			FROM executions
			ORDER BY started_at DESC
			LIMIT ? OFFSET ?
		`
		args = []interface{}{limit, offset}
	}

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list executions: %w", err)
	}
	defer rows.Close()

	var executions []*ExecutionRecord
	for rows.Next() {
		exec, err := s.scanExecutionRow(rows)
		if err != nil {
			return nil, err
		}
		executions = append(executions, exec)
	}

	return executions, nil
}

// DeleteExecution deletes an execution record
func (s *SQLiteAgentStore) DeleteExecution(ctx context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	_, err := s.db.ExecContext(ctx, `DELETE FROM executions WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("failed to delete execution: %w", err)
	}

	return nil
}

// Close closes the database connection
func (s *SQLiteAgentStore) Close() error {
	return s.db.Close()
}

// Statistics returns store statistics
func (s *SQLiteAgentStore) Statistics(ctx context.Context) (map[string]interface{}, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	stats := make(map[string]interface{})

	// Total agents
	var totalAgents int64
	s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM agents`).Scan(&totalAgents)
	stats["total_agents"] = totalAgents

	// Total executions
	var totalExecs int64
	s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM executions`).Scan(&totalExecs)
	stats["total_executions"] = totalExecs

	// Executions by status
	statusRows, _ := s.db.QueryContext(ctx, `SELECT status, COUNT(*) FROM executions GROUP BY status`)
	if statusRows != nil {
		defer statusRows.Close()
		statusCounts := make(map[string]int64)
		for statusRows.Next() {
			var status string
			var count int64
			statusRows.Scan(&status, &count)
			statusCounts[status] = count
		}
		stats["executions_by_status"] = statusCounts
	}

	// Average duration
	var avgDuration sql.NullFloat64
	s.db.QueryRowContext(ctx, `SELECT AVG(duration_ms) FROM executions WHERE status = 'completed'`).Scan(&avgDuration)
	if avgDuration.Valid {
		stats["avg_duration_ms"] = avgDuration.Float64
	}

	return stats, nil
}

// Helper functions

func (s *SQLiteAgentStore) scanExecution(row *sql.Row) (*ExecutionRecord, error) {
	var exec ExecutionRecord
	var stepsJSON, toolsUsedJSON sql.NullString
	var completedAt sql.NullTime

	err := row.Scan(&exec.ID, &exec.AgentID, &exec.Message, &exec.Status, &exec.Result, &exec.Error,
		&stepsJSON, &toolsUsedJSON, &exec.StartedAt, &completedAt, &exec.Duration)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get execution: %w", err)
	}

	if completedAt.Valid {
		exec.CompletedAt = completedAt.Time
	}
	if stepsJSON.Valid {
		json.Unmarshal([]byte(stepsJSON.String), &exec.Steps)
	}
	if toolsUsedJSON.Valid {
		json.Unmarshal([]byte(toolsUsedJSON.String), &exec.ToolsUsed)
	}

	return &exec, nil
}

func (s *SQLiteAgentStore) scanExecutionRow(rows *sql.Rows) (*ExecutionRecord, error) {
	var exec ExecutionRecord
	var stepsJSON, toolsUsedJSON sql.NullString
	var completedAt sql.NullTime

	err := rows.Scan(&exec.ID, &exec.AgentID, &exec.Message, &exec.Status, &exec.Result, &exec.Error,
		&stepsJSON, &toolsUsedJSON, &exec.StartedAt, &completedAt, &exec.Duration)
	if err != nil {
		return nil, fmt.Errorf("failed to scan execution: %w", err)
	}

	if completedAt.Valid {
		exec.CompletedAt = completedAt.Time
	}
	if stepsJSON.Valid {
		json.Unmarshal([]byte(stepsJSON.String), &exec.Steps)
	}
	if toolsUsedJSON.Valid {
		json.Unmarshal([]byte(toolsUsedJSON.String), &exec.ToolsUsed)
	}

	return &exec, nil
}

func nullTime(t time.Time) interface{} {
	if t.IsZero() {
		return nil
	}
	return t
}

// MemoryAgentStore is an in-memory implementation for testing
type MemoryAgentStore struct {
	mu         sync.RWMutex
	agents     map[string]*AgentDefinition
	executions map[string]*ExecutionRecord
}

// NewMemoryAgentStore creates a new in-memory agent store
func NewMemoryAgentStore() *MemoryAgentStore {
	return &MemoryAgentStore{
		agents:     make(map[string]*AgentDefinition),
		executions: make(map[string]*ExecutionRecord),
	}
}

// CreateAgent creates a new agent definition
func (s *MemoryAgentStore) CreateAgent(ctx context.Context, agent *AgentDefinition) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if agent.ID == "" {
		return fmt.Errorf("agent ID is required")
	}

	now := time.Now()
	if agent.CreatedAt.IsZero() {
		agent.CreatedAt = now
	}
	agent.UpdatedAt = now

	s.agents[agent.ID] = agent
	return nil
}

// GetAgent retrieves an agent by ID
func (s *MemoryAgentStore) GetAgent(ctx context.Context, id string) (*AgentDefinition, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	agent, ok := s.agents[id]
	if !ok {
		return nil, nil
	}
	return agent, nil
}

// UpdateAgent updates an agent definition
func (s *MemoryAgentStore) UpdateAgent(ctx context.Context, agent *AgentDefinition) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.agents[agent.ID]; !ok {
		return fmt.Errorf("agent not found: %s", agent.ID)
	}

	agent.UpdatedAt = time.Now()
	s.agents[agent.ID] = agent
	return nil
}

// DeleteAgent deletes an agent
func (s *MemoryAgentStore) DeleteAgent(ctx context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.agents, id)
	// Also delete related executions
	for execID, exec := range s.executions {
		if exec.AgentID == id {
			delete(s.executions, execID)
		}
	}
	return nil
}

// ListAgents returns all agents
func (s *MemoryAgentStore) ListAgents(ctx context.Context) ([]*AgentDefinition, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	agents := make([]*AgentDefinition, 0, len(s.agents))
	for _, agent := range s.agents {
		agents = append(agents, agent)
	}
	return agents, nil
}

// CreateExecution creates a new execution record
func (s *MemoryAgentStore) CreateExecution(ctx context.Context, exec *ExecutionRecord) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if exec.ID == "" {
		return fmt.Errorf("execution ID is required")
	}

	if exec.StartedAt.IsZero() {
		exec.StartedAt = time.Now()
	}

	s.executions[exec.ID] = exec
	return nil
}

// UpdateExecution updates an execution record
func (s *MemoryAgentStore) UpdateExecution(ctx context.Context, exec *ExecutionRecord) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.executions[exec.ID]; !ok {
		return fmt.Errorf("execution not found: %s", exec.ID)
	}

	s.executions[exec.ID] = exec
	return nil
}

// GetExecution retrieves an execution by ID
func (s *MemoryAgentStore) GetExecution(ctx context.Context, id string) (*ExecutionRecord, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	exec, ok := s.executions[id]
	if !ok {
		return nil, nil
	}
	return exec, nil
}

// ListExecutions returns executions for an agent
func (s *MemoryAgentStore) ListExecutions(ctx context.Context, agentID string, limit, offset int) ([]*ExecutionRecord, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var executions []*ExecutionRecord
	for _, exec := range s.executions {
		if agentID == "" || exec.AgentID == agentID {
			executions = append(executions, exec)
		}
	}

	// Sort by started_at descending (simple sort)
	for i := 0; i < len(executions)-1; i++ {
		for j := i + 1; j < len(executions); j++ {
			if executions[j].StartedAt.After(executions[i].StartedAt) {
				executions[i], executions[j] = executions[j], executions[i]
			}
		}
	}

	// Apply pagination
	if offset >= len(executions) {
		return []*ExecutionRecord{}, nil
	}
	if limit <= 0 {
		limit = 50
	}
	end := offset + limit
	if end > len(executions) {
		end = len(executions)
	}

	return executions[offset:end], nil
}

// DeleteExecution deletes an execution record
func (s *MemoryAgentStore) DeleteExecution(ctx context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.executions, id)
	return nil
}

// Close is a no-op for memory store
func (s *MemoryAgentStore) Close() error {
	return nil
}

// Statistics returns store statistics
func (s *MemoryAgentStore) Statistics(ctx context.Context) (map[string]interface{}, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	statusCounts := make(map[string]int64)
	for _, exec := range s.executions {
		statusCounts[exec.Status]++
	}

	return map[string]interface{}{
		"total_agents":         len(s.agents),
		"total_executions":     len(s.executions),
		"executions_by_status": statusCounts,
	}, nil
}
