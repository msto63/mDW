package handlers

import (
	"github.com/msto63/mDW/internal/platon/chain"
)

// BaseHandler provides a base implementation for handlers
type BaseHandler struct {
	name     string
	htype    chain.HandlerType
	priority int
	enabled  bool
}

// NewBaseHandler creates a new base handler
func NewBaseHandler(name string, htype chain.HandlerType, priority int) *BaseHandler {
	return &BaseHandler{
		name:     name,
		htype:    htype,
		priority: priority,
		enabled:  true,
	}
}

// Name returns the handler name
func (h *BaseHandler) Name() string {
	return h.name
}

// Type returns the handler type
func (h *BaseHandler) Type() chain.HandlerType {
	return h.htype
}

// Priority returns the handler priority
func (h *BaseHandler) Priority() int {
	return h.priority
}

// SetEnabled enables or disables the handler
func (h *BaseHandler) SetEnabled(enabled bool) {
	h.enabled = enabled
}

// IsEnabled returns whether the handler is enabled
func (h *BaseHandler) IsEnabled() bool {
	return h.enabled
}

// ShouldProcess returns true if the handler should process (default implementation)
func (h *BaseHandler) ShouldProcess(ctx *chain.ProcessingContext) bool {
	return h.enabled
}

// DynamicHandler is a configurable handler that can be registered via gRPC
type DynamicHandler struct {
	*BaseHandler
	description string
	config      map[string]string
}

// DynamicHandlerConfig holds configuration for creating a dynamic handler
type DynamicHandlerConfig struct {
	Name        string
	Type        chain.HandlerType
	Priority    int
	Description string
	Enabled     bool
	Settings    map[string]string
}

// NewDynamicHandler creates a new dynamic handler with the given configuration
func NewDynamicHandler(cfg DynamicHandlerConfig) *DynamicHandler {
	base := NewBaseHandler(cfg.Name, cfg.Type, cfg.Priority)
	base.enabled = cfg.Enabled

	config := cfg.Settings
	if config == nil {
		config = make(map[string]string)
	}

	return &DynamicHandler{
		BaseHandler: base,
		description: cfg.Description,
		config:      config,
	}
}

// Description returns the handler description
func (h *DynamicHandler) Description() string {
	return h.description
}

// Config returns the handler configuration
func (h *DynamicHandler) Config() map[string]string {
	return h.config
}

// Process implements the Handler interface for dynamic handlers
// Dynamic handlers are pass-through by default (they don't modify the context)
func (h *DynamicHandler) Process(ctx *chain.ProcessingContext) error {
	// Dynamic handlers are configurable placeholders
	// They can be extended with specific behavior based on config
	return nil
}
