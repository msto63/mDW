package chain

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/msto63/mDW/pkg/core/logging"
)

// Chain manages the handler chain using Chain-of-Responsibility pattern
type Chain struct {
	preHandlers  []Handler
	postHandlers []Handler
	logger       logging.Logger
	mu           sync.RWMutex
}

// NewChain creates a new Chain
func NewChain(logger logging.Logger) *Chain {
	return &Chain{
		preHandlers:  make([]Handler, 0),
		postHandlers: make([]Handler, 0),
		logger:       logger,
	}
}

// Register adds a handler to the chain
func (c *Chain) Register(h Handler) {
	c.mu.Lock()
	defer c.mu.Unlock()

	switch h.Type() {
	case HandlerTypePre:
		c.preHandlers = append(c.preHandlers, h)
		c.sortByPriority(c.preHandlers)
	case HandlerTypePost:
		c.postHandlers = append(c.postHandlers, h)
		c.sortByPriority(c.postHandlers)
	case HandlerTypeBoth:
		c.preHandlers = append(c.preHandlers, h)
		c.postHandlers = append(c.postHandlers, h)
		c.sortByPriority(c.preHandlers)
		c.sortByPriority(c.postHandlers)
	}

	c.logger.Info("Handler registered",
		"name", h.Name(),
		"type", h.Type().String(),
		"priority", h.Priority())
}

// Unregister removes a handler from the chain by name
func (c *Chain) Unregister(name string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	removed := false

	c.preHandlers = c.removeHandler(c.preHandlers, name, &removed)
	c.postHandlers = c.removeHandler(c.postHandlers, name, &removed)

	if removed {
		c.logger.Info("Handler unregistered", "name", name)
	}

	return removed
}

func (c *Chain) removeHandler(handlers []Handler, name string, removed *bool) []Handler {
	result := make([]Handler, 0, len(handlers))
	for _, h := range handlers {
		if h.Name() != name {
			result = append(result, h)
		} else {
			*removed = true
		}
	}
	return result
}

// GetHandler returns a handler by name
func (c *Chain) GetHandler(name string) (Handler, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	for _, h := range c.preHandlers {
		if h.Name() == name {
			return h, true
		}
	}
	for _, h := range c.postHandlers {
		if h.Name() == name {
			return h, true
		}
	}
	return nil, false
}

// ListHandlers returns information about all registered handlers
func (c *Chain) ListHandlers() []HandlerInfo {
	c.mu.RLock()
	defer c.mu.RUnlock()

	seen := make(map[string]bool)
	result := make([]HandlerInfo, 0)

	addHandler := func(h Handler) {
		if seen[h.Name()] {
			return
		}
		seen[h.Name()] = true
		result = append(result, HandlerInfo{
			Name:     h.Name(),
			Type:     h.Type(),
			Priority: h.Priority(),
			Enabled:  true,
		})
	}

	for _, h := range c.preHandlers {
		addHandler(h)
	}
	for _, h := range c.postHandlers {
		addHandler(h)
	}

	return result
}

// ProcessPre executes the pre-processing chain
func (c *Chain) ProcessPre(ctx *ProcessingContext) error {
	ctx.Phase = PhasePre
	return c.processChain(ctx, c.preHandlers)
}

// ProcessPost executes the post-processing chain
func (c *Chain) ProcessPost(ctx *ProcessingContext) error {
	ctx.Phase = PhasePost
	return c.processChain(ctx, c.postHandlers)
}

// Process executes the full pipeline (pre -> main -> post)
// The mainProcessor is called between pre and post if not blocked
func (c *Chain) Process(ctx context.Context, req *ProcessRequest, mainProcessor func(context.Context, string) (string, error)) (*ProcessResult, error) {
	pctx := NewProcessingContext(ctx, req.RequestID, req.PipelineID, req.Prompt)

	// Copy metadata
	for k, v := range req.Metadata {
		pctx.SetMetadata(k, v)
	}

	// Pre-processing
	if err := c.ProcessPre(pctx); err != nil {
		return nil, fmt.Errorf("pre-processing failed: %w", err)
	}

	// Check if blocked
	if pctx.Blocked {
		c.logger.Info("Request blocked during pre-processing",
			"request_id", pctx.RequestID,
			"reason", pctx.BlockReason)
		return pctx.ToResult(), nil
	}

	// Main processing (if provided)
	if mainProcessor != nil {
		response, err := mainProcessor(ctx, pctx.Prompt)
		if err != nil {
			return nil, fmt.Errorf("main processing failed: %w", err)
		}
		pctx.Response = response
	} else if req.Response != "" {
		// Use provided response for post-processing only
		pctx.Response = req.Response
	}

	// Post-processing
	if err := c.ProcessPost(pctx); err != nil {
		return nil, fmt.Errorf("post-processing failed: %w", err)
	}

	c.logger.Info("Pipeline processing completed",
		"request_id", pctx.RequestID,
		"duration_ms", pctx.Duration().Milliseconds(),
		"blocked", pctx.Blocked,
		"modified", pctx.Modified,
		"handlers_executed", len(pctx.AuditLog))

	return pctx.ToResult(), nil
}

func (c *Chain) processChain(ctx *ProcessingContext, handlers []Handler) error {
	c.mu.RLock()
	handlersCopy := make([]Handler, len(handlers))
	copy(handlersCopy, handlers)
	c.mu.RUnlock()

	for _, h := range handlersCopy {
		// Check context cancellation
		select {
		case <-ctx.Context().Done():
			return ctx.Context().Err()
		default:
		}

		// Check if blocked
		if ctx.Blocked {
			c.logger.Debug("Chain aborted - request blocked",
				"handler", h.Name(),
				"reason", ctx.BlockReason)
			return nil
		}

		// Check if handler should process
		if !h.ShouldProcess(ctx) {
			c.logger.Debug("Handler skipped",
				"handler", h.Name(),
				"phase", ctx.Phase.String())
			continue
		}

		// Execute handler
		start := time.Now()
		wasModified := ctx.Modified
		err := h.Process(ctx)
		duration := time.Since(start)

		// Record audit entry
		entry := AuditEntry{
			Handler:  h.Name(),
			Phase:    ctx.Phase,
			Duration: duration,
			Error:    err,
			Modified: ctx.Modified && !wasModified,
		}
		ctx.AddAuditEntry(entry)

		if err != nil {
			c.logger.Error("Handler failed",
				"handler", h.Name(),
				"phase", ctx.Phase.String(),
				"error", err)
			return fmt.Errorf("handler %s failed: %w", h.Name(), err)
		}

		c.logger.Debug("Handler executed",
			"handler", h.Name(),
			"phase", ctx.Phase.String(),
			"duration_ms", duration.Milliseconds(),
			"modified", entry.Modified)
	}

	return nil
}

func (c *Chain) sortByPriority(handlers []Handler) {
	sort.Slice(handlers, func(i, j int) bool {
		return handlers[i].Priority() < handlers[j].Priority()
	})
}

// PreHandlerCount returns the number of pre-processing handlers
func (c *Chain) PreHandlerCount() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.preHandlers)
}

// PostHandlerCount returns the number of post-processing handlers
func (c *Chain) PostHandlerCount() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.postHandlers)
}

// TotalHandlerCount returns the total number of unique handlers
func (c *Chain) TotalHandlerCount() int {
	return len(c.ListHandlers())
}
