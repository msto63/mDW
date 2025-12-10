package handlers

import (
	"encoding/json"
	"time"

	"github.com/msto63/mDW/internal/platon/chain"
	"github.com/msto63/mDW/pkg/core/logging"
)

// AuditConfig holds configuration for the audit handler
type AuditConfig struct {
	LogPrompts    bool `json:"log_prompts"`
	LogResponses  bool `json:"log_responses"`
	LogMetadata   bool `json:"log_metadata"`
	LogViolations bool `json:"log_violations"`
	MaxTextLength int  `json:"max_text_length"` // 0 = unlimited
}

// DefaultAuditConfig returns default audit configuration
func DefaultAuditConfig() AuditConfig {
	return AuditConfig{
		LogPrompts:    true,
		LogResponses:  true,
		LogMetadata:   true,
		LogViolations: true,
		MaxTextLength: 500, // Truncate long texts
	}
}

// AuditHandler logs all pipeline processing for audit purposes
type AuditHandler struct {
	*BaseHandler
	config AuditConfig
	logger logging.Logger
}

// NewAuditHandler creates a new audit handler
func NewAuditHandler(config AuditConfig, logger logging.Logger) *AuditHandler {
	return &AuditHandler{
		BaseHandler: NewBaseHandler(
			"audit",
			chain.HandlerTypeBoth, // Audit both pre and post
			1000,                  // Very low priority - runs last
		),
		config: config,
		logger: logger,
	}
}

// NewDefaultAuditHandler creates an audit handler with default config
func NewDefaultAuditHandler(logger logging.Logger) *AuditHandler {
	return NewAuditHandler(DefaultAuditConfig(), logger)
}

// Process logs audit information
func (h *AuditHandler) Process(ctx *chain.ProcessingContext) error {
	entry := h.buildAuditEntry(ctx)

	// Log based on phase
	if ctx.Phase == chain.PhasePre {
		h.logPreProcessing(entry)
	} else {
		h.logPostProcessing(entry)
	}

	return nil
}

// AuditEntry represents an audit log entry
type AuditEntry struct {
	Timestamp   time.Time              `json:"timestamp"`
	RequestID   string                 `json:"request_id"`
	PipelineID  string                 `json:"pipeline_id"`
	Phase       string                 `json:"phase"`
	Prompt      string                 `json:"prompt,omitempty"`
	Response    string                 `json:"response,omitempty"`
	Blocked     bool                   `json:"blocked"`
	BlockReason string                 `json:"block_reason,omitempty"`
	Modified    bool                   `json:"modified"`
	DurationMs  int64                  `json:"duration_ms"`
	Violations  []PolicyViolation      `json:"violations,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// buildAuditEntry creates an audit entry from the context
func (h *AuditHandler) buildAuditEntry(ctx *chain.ProcessingContext) AuditEntry {
	entry := AuditEntry{
		Timestamp:   time.Now(),
		RequestID:   ctx.RequestID,
		PipelineID:  ctx.PipelineID,
		Phase:       ctx.Phase.String(),
		Blocked:     ctx.Blocked,
		BlockReason: ctx.BlockReason,
		Modified:    ctx.Modified,
		DurationMs:  ctx.Duration().Milliseconds(),
	}

	// Add prompt/response based on config
	if h.config.LogPrompts && ctx.Prompt != "" {
		entry.Prompt = h.truncate(ctx.Prompt)
	}
	if h.config.LogResponses && ctx.Response != "" {
		entry.Response = h.truncate(ctx.Response)
	}

	// Add violations if configured
	if h.config.LogViolations {
		if violations, ok := ctx.GetState("policy_violations"); ok {
			if v, ok := violations.([]PolicyViolation); ok {
				entry.Violations = v
			}
		}
	}

	// Add metadata if configured
	if h.config.LogMetadata && len(ctx.Metadata) > 0 {
		entry.Metadata = make(map[string]interface{})
		for k, v := range ctx.Metadata {
			entry.Metadata[k] = v
		}
	}

	return entry
}

// truncate truncates text to max length
func (h *AuditHandler) truncate(text string) string {
	if h.config.MaxTextLength == 0 || len(text) <= h.config.MaxTextLength {
		return text
	}
	return text[:h.config.MaxTextLength] + "..."
}

// logPreProcessing logs pre-processing audit entry
func (h *AuditHandler) logPreProcessing(entry AuditEntry) {
	fields := h.entryToFields(entry)

	if entry.Blocked {
		h.logger.Warn("Pipeline pre-processing blocked", fields...)
	} else if entry.Modified {
		h.logger.Info("Pipeline pre-processing modified", fields...)
	} else {
		h.logger.Debug("Pipeline pre-processing completed", fields...)
	}
}

// logPostProcessing logs post-processing audit entry
func (h *AuditHandler) logPostProcessing(entry AuditEntry) {
	fields := h.entryToFields(entry)

	if entry.Blocked {
		h.logger.Warn("Pipeline post-processing blocked", fields...)
	} else if entry.Modified {
		h.logger.Info("Pipeline post-processing modified", fields...)
	} else {
		h.logger.Debug("Pipeline post-processing completed", fields...)
	}
}

// entryToFields converts audit entry to logger fields
func (h *AuditHandler) entryToFields(entry AuditEntry) []interface{} {
	fields := []interface{}{
		"request_id", entry.RequestID,
		"pipeline_id", entry.PipelineID,
		"phase", entry.Phase,
		"blocked", entry.Blocked,
		"modified", entry.Modified,
		"duration_ms", entry.DurationMs,
	}

	if entry.BlockReason != "" {
		fields = append(fields, "block_reason", entry.BlockReason)
	}

	if len(entry.Violations) > 0 {
		fields = append(fields, "violations", len(entry.Violations))
	}

	return fields
}

// EntryToJSON converts the audit entry to JSON
func (h *AuditHandler) EntryToJSON(entry AuditEntry) ([]byte, error) {
	return json.Marshal(entry)
}
