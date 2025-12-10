package chain

import (
	"time"
)

// HandlerType defines when a handler is executed
type HandlerType int

const (
	// HandlerTypePre executes before main processing
	HandlerTypePre HandlerType = iota
	// HandlerTypePost executes after main processing
	HandlerTypePost
	// HandlerTypeBoth executes both before and after
	HandlerTypeBoth
)

// String returns the string representation of HandlerType
func (t HandlerType) String() string {
	switch t {
	case HandlerTypePre:
		return "pre"
	case HandlerTypePost:
		return "post"
	case HandlerTypeBoth:
		return "both"
	default:
		return "unknown"
	}
}

// ProcessingPhase indicates current processing phase
type ProcessingPhase int

const (
	// PhasePre is before main processing
	PhasePre ProcessingPhase = iota
	// PhasePost is after main processing
	PhasePost
)

// String returns the string representation of ProcessingPhase
func (p ProcessingPhase) String() string {
	switch p {
	case PhasePre:
		return "pre"
	case PhasePost:
		return "post"
	default:
		return "unknown"
	}
}

// Handler is the central interface for pipeline steps
type Handler interface {
	// Name returns the unique handler name
	Name() string

	// Type returns when this handler executes (pre, post, both)
	Type() HandlerType

	// Priority determines execution order (lower = earlier)
	Priority() int

	// Process handles the request/response
	Process(ctx *ProcessingContext) error

	// ShouldProcess decides if this handler should run
	ShouldProcess(ctx *ProcessingContext) bool
}

// AuditEntry records a single handler execution
type AuditEntry struct {
	Handler   string           `json:"handler"`
	Phase     ProcessingPhase  `json:"phase"`
	Duration  time.Duration    `json:"duration"`
	Error     error            `json:"error,omitempty"`
	Modified  bool             `json:"modified"`
	Details   map[string]any   `json:"details,omitempty"`
}

// Pipeline represents a configured collection of handlers
type Pipeline struct {
	ID           string            `json:"id"`
	Name         string            `json:"name"`
	Description  string            `json:"description"`
	Enabled      bool              `json:"enabled"`
	PreHandlers  []string          `json:"pre_handlers"`
	PostHandlers []string          `json:"post_handlers"`
	Config       map[string]string `json:"config"`
	CreatedAt    time.Time         `json:"created_at"`
	UpdatedAt    time.Time         `json:"updated_at"`
}

// ProcessRequest represents an incoming processing request
type ProcessRequest struct {
	RequestID  string            `json:"request_id"`
	PipelineID string            `json:"pipeline_id"`
	Prompt     string            `json:"prompt"`
	Response   string            `json:"response"`
	Metadata   map[string]any    `json:"metadata"`
	Options    map[string]string `json:"options"`
}

// ProcessResult represents the result of pipeline processing
type ProcessResult struct {
	RequestID       string            `json:"request_id"`
	ProcessedPrompt string            `json:"processed_prompt"`
	ProcessedResponse string          `json:"processed_response"`
	Blocked         bool              `json:"blocked"`
	BlockReason     string            `json:"block_reason,omitempty"`
	Modified        bool              `json:"modified"`
	AuditLog        []AuditEntry      `json:"audit_log"`
	Metadata        map[string]any    `json:"metadata"`
	Duration        time.Duration     `json:"duration"`
}

// HandlerInfo provides metadata about a handler
type HandlerInfo struct {
	Name        string            `json:"name"`
	Type        HandlerType       `json:"type"`
	Priority    int               `json:"priority"`
	Description string            `json:"description"`
	Enabled     bool              `json:"enabled"`
	Config      map[string]string `json:"config"`
}
