// Package pipeline provides the pipeline engine for Aristoteles service
package pipeline

import (
	"context"
	"time"

	pb "github.com/msto63/mDW/api/gen/aristoteles"
)

// Stage represents a pipeline stage
type Stage interface {
	// Name returns the stage name
	Name() string
	// Execute runs the stage with the given context
	Execute(ctx context.Context, pctx *Context) error
}

// Context holds the pipeline state during execution
type Context struct {
	// Request data
	RequestID      string
	Prompt         string
	ConversationID string
	Metadata       map[string]string
	Options        *pb.ProcessOptions

	// Pipeline state
	Intent     *pb.IntentResult
	Strategy   *pb.StrategyInfo
	Enrichments []*pb.EnrichmentStep
	Response   string

	// Routing info
	Route *pb.RouteInfo

	// Metrics
	Metrics *pb.PipelineMetrics

	// Control
	Blocked     bool
	BlockReason string
	Cancelled   bool

	// Timing
	StartTime time.Time

	// Internal state
	state map[string]interface{}
}

// NewContext creates a new pipeline context
func NewContext(requestID, prompt, conversationID string, metadata map[string]string, options *pb.ProcessOptions) *Context {
	if metadata == nil {
		metadata = make(map[string]string)
	}
	if options == nil {
		options = &pb.ProcessOptions{}
	}
	return &Context{
		RequestID:      requestID,
		Prompt:         prompt,
		ConversationID: conversationID,
		Metadata:       metadata,
		Options:        options,
		Enrichments:    make([]*pb.EnrichmentStep, 0),
		Metrics: &pb.PipelineMetrics{},
		StartTime:      time.Now(),
		state:          make(map[string]interface{}),
	}
}

// Set stores a value in the context state
func (c *Context) Set(key string, value interface{}) {
	c.state[key] = value
}

// Get retrieves a value from the context state
func (c *Context) Get(key string) (interface{}, bool) {
	val, ok := c.state[key]
	return val, ok
}

// GetString retrieves a string value from the context state
func (c *Context) GetString(key string) string {
	val, ok := c.state[key]
	if !ok {
		return ""
	}
	str, ok := val.(string)
	if !ok {
		return ""
	}
	return str
}

// ElapsedTime returns the time since the context was created
func (c *Context) ElapsedTime() time.Duration {
	return time.Since(c.StartTime)
}

// AddEnrichment adds an enrichment step to the context
func (c *Context) AddEnrichment(enrichment *pb.EnrichmentStep) {
	c.Enrichments = append(c.Enrichments, enrichment)
}

// GetEnrichedPrompt returns the prompt with all enrichments applied
func (c *Context) GetEnrichedPrompt() string {
	if len(c.Enrichments) == 0 {
		return c.Prompt
	}

	enrichedPrompt := c.Prompt
	for _, e := range c.Enrichments {
		if e.Success && e.Content != "" {
			enrichedPrompt = enrichedPrompt + "\n\n[Context from " + e.Source + "]:\n" + e.Content
		}
	}
	return enrichedPrompt
}

// ToProcessResponse converts the context to a ProcessResponse
func (c *Context) ToProcessResponse() *pb.ProcessResponse {
	c.Metrics.TotalDurationMs = c.ElapsedTime().Milliseconds()

	return &pb.ProcessResponse{
		RequestId:   c.RequestID,
		Response:    c.Response,
		Intent:      c.Intent,
		Strategy:    c.Strategy,
		Enrichments: c.Enrichments,
		Route:       c.Route,
		Metrics:     c.Metrics,
		Metadata:    c.Metadata,
		Blocked:     c.Blocked,
		BlockReason: c.BlockReason,
	}
}
