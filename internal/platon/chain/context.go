package chain

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"
)

// ProcessingContext contains all information during pipeline processing
type ProcessingContext struct {
	// Go context for cancellation and timeouts
	ctx context.Context

	// Request identification
	RequestID  string
	PipelineID string

	// Content being processed
	Prompt   string
	Response string

	// Metadata from the original request
	Metadata map[string]any

	// Processing state
	Phase       ProcessingPhase
	Blocked     bool
	BlockReason string
	Modified    bool

	// Shared state between handlers
	State map[string]any
	mu    sync.RWMutex

	// Audit trail
	AuditLog  []AuditEntry
	StartTime time.Time
}

// NewProcessingContext creates a new processing context
func NewProcessingContext(ctx context.Context, requestID, pipelineID, prompt string) *ProcessingContext {
	if requestID == "" {
		requestID = uuid.New().String()
	}

	return &ProcessingContext{
		ctx:        ctx,
		RequestID:  requestID,
		PipelineID: pipelineID,
		Prompt:     prompt,
		Metadata:   make(map[string]any),
		State:      make(map[string]any),
		AuditLog:   make([]AuditEntry, 0),
		StartTime:  time.Now(),
	}
}

// Context returns the Go context
func (c *ProcessingContext) Context() context.Context {
	return c.ctx
}

// SetState sets a value in the shared state (thread-safe)
func (c *ProcessingContext) SetState(key string, value any) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.State[key] = value
}

// GetState gets a value from the shared state (thread-safe)
func (c *ProcessingContext) GetState(key string) (any, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	val, ok := c.State[key]
	return val, ok
}

// GetStateString gets a string value from state
func (c *ProcessingContext) GetStateString(key string) string {
	val, ok := c.GetState(key)
	if !ok {
		return ""
	}
	s, ok := val.(string)
	if !ok {
		return ""
	}
	return s
}

// GetStateBool gets a boolean value from state
func (c *ProcessingContext) GetStateBool(key string) bool {
	val, ok := c.GetState(key)
	if !ok {
		return false
	}
	b, ok := val.(bool)
	if !ok {
		return false
	}
	return b
}

// SetMetadata sets a metadata value
func (c *ProcessingContext) SetMetadata(key string, value any) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.Metadata[key] = value
}

// GetMetadata gets a metadata value
func (c *ProcessingContext) GetMetadata(key string) (any, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	val, ok := c.Metadata[key]
	return val, ok
}

// Block marks the request as blocked
func (c *ProcessingContext) Block(reason string) {
	c.Blocked = true
	c.BlockReason = reason
}

// MarkModified marks the content as modified
func (c *ProcessingContext) MarkModified() {
	c.Modified = true
}

// AddAuditEntry adds an entry to the audit log
func (c *ProcessingContext) AddAuditEntry(entry AuditEntry) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.AuditLog = append(c.AuditLog, entry)
}

// Duration returns the time since processing started
func (c *ProcessingContext) Duration() time.Duration {
	return time.Since(c.StartTime)
}

// CurrentText returns the current text based on the phase
func (c *ProcessingContext) CurrentText() string {
	if c.Phase == PhasePre {
		return c.Prompt
	}
	return c.Response
}

// SetCurrentText sets the current text based on the phase
func (c *ProcessingContext) SetCurrentText(text string) {
	if c.Phase == PhasePre {
		c.Prompt = text
	} else {
		c.Response = text
	}
}

// ToResult converts the context to a ProcessResult
func (c *ProcessingContext) ToResult() *ProcessResult {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return &ProcessResult{
		RequestID:         c.RequestID,
		ProcessedPrompt:   c.Prompt,
		ProcessedResponse: c.Response,
		Blocked:           c.Blocked,
		BlockReason:       c.BlockReason,
		Modified:          c.Modified,
		AuditLog:          c.AuditLog,
		Metadata:          c.Metadata,
		Duration:          c.Duration(),
	}
}

// Clone creates a copy of the context for parallel processing
func (c *ProcessingContext) Clone() *ProcessingContext {
	c.mu.RLock()
	defer c.mu.RUnlock()

	stateCopy := make(map[string]any)
	for k, v := range c.State {
		stateCopy[k] = v
	}

	metaCopy := make(map[string]any)
	for k, v := range c.Metadata {
		metaCopy[k] = v
	}

	return &ProcessingContext{
		ctx:         c.ctx,
		RequestID:   c.RequestID,
		PipelineID:  c.PipelineID,
		Prompt:      c.Prompt,
		Response:    c.Response,
		Metadata:    metaCopy,
		Phase:       c.Phase,
		Blocked:     c.Blocked,
		BlockReason: c.BlockReason,
		Modified:    c.Modified,
		State:       stateCopy,
		AuditLog:    make([]AuditEntry, len(c.AuditLog)),
		StartTime:   c.StartTime,
	}
}
