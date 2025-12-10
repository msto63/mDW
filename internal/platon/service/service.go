package service

import (
	"context"
	"fmt"
	"regexp"
	"sync"
	"time"

	"github.com/google/uuid"
	mdwerror "github.com/msto63/mDW/foundation/core/error"
	"github.com/msto63/mDW/internal/platon/chain"
	"github.com/msto63/mDW/internal/platon/handlers"
	"github.com/msto63/mDW/pkg/core/logging"
)

// Config holds service configuration
type Config struct {
	DefaultPipeline string
	MaxHandlers     int
	HandlerTimeout  time.Duration
}

// DefaultConfig returns default service configuration
func DefaultConfig() Config {
	return Config{
		DefaultPipeline: "default",
		MaxHandlers:     100,
		HandlerTimeout:  30 * time.Second,
	}
}

// Policy represents a policy configuration
type Policy struct {
	ID          string
	Name        string
	Description string
	Type        string
	Enabled     bool
	Priority    int
	Rules       []PolicyRule
	LLMCheck    *LLMCheckConfig
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// PolicyRule represents a single rule within a policy
type PolicyRule struct {
	ID            string
	Pattern       string
	Action        string
	Message       string
	Replacement   string
	CaseSensitive bool
}

// LLMCheckConfig holds configuration for LLM-based policy checks
type LLMCheckConfig struct {
	Enabled        bool
	Model          string
	Prompt         string
	TimeoutSeconds int
	Temperature    float32
}

// Service is the Platon business logic layer
type Service struct {
	config    Config
	chain     *chain.Chain
	pipelines map[string]*chain.Pipeline
	policies  map[string]*Policy
	logger    logging.Logger
	mu        sync.RWMutex
}

// NewService creates a new Platon service
func NewService(cfg Config, logger logging.Logger) *Service {
	return &Service{
		config:    cfg,
		chain:     chain.NewChain(logger),
		pipelines: make(map[string]*chain.Pipeline),
		policies:  make(map[string]*Policy),
		logger:    logger,
	}
}

// RegisterHandler registers a handler with the chain
func (s *Service) RegisterHandler(h chain.Handler) error {
	if s.chain.TotalHandlerCount() >= s.config.MaxHandlers {
		return mdwerror.New("maximum number of handlers reached").
			WithCode(mdwerror.CodeQuotaExceeded).
			WithOperation("service.RegisterHandler")
	}

	s.chain.Register(h)
	return nil
}

// DynamicHandlerConfig holds configuration for registering a dynamic handler via gRPC
type DynamicHandlerConfig struct {
	Name        string
	Type        chain.HandlerType
	Priority    int
	Description string
	Enabled     bool
	Settings    map[string]string
}

// RegisterDynamicHandler creates and registers a dynamic handler
func (s *Service) RegisterDynamicHandler(cfg DynamicHandlerConfig) (*handlers.DynamicHandler, error) {
	// Check if handler already exists
	if _, exists := s.chain.GetHandler(cfg.Name); exists {
		return nil, mdwerror.New("handler already exists").
			WithCode(mdwerror.CodeDuplicateEntry).
			WithOperation("service.RegisterDynamicHandler").
			WithDetail("handler_name", cfg.Name)
	}

	// Check handler limit
	if s.chain.TotalHandlerCount() >= s.config.MaxHandlers {
		return nil, mdwerror.New("maximum number of handlers reached").
			WithCode(mdwerror.CodeQuotaExceeded).
			WithOperation("service.RegisterDynamicHandler")
	}

	// Create the dynamic handler
	h := handlers.NewDynamicHandler(handlers.DynamicHandlerConfig{
		Name:        cfg.Name,
		Type:        cfg.Type,
		Priority:    cfg.Priority,
		Description: cfg.Description,
		Enabled:     cfg.Enabled,
		Settings:    cfg.Settings,
	})

	// Register it
	s.chain.Register(h)

	s.logger.Info("Dynamic handler registered",
		"name", cfg.Name,
		"type", cfg.Type.String(),
		"priority", cfg.Priority)

	return h, nil
}

// UnregisterHandler removes a handler from the chain
func (s *Service) UnregisterHandler(name string) bool {
	return s.chain.Unregister(name)
}

// GetHandler returns a handler by name
func (s *Service) GetHandler(name string) (chain.Handler, bool) {
	return s.chain.GetHandler(name)
}

// ListHandlers returns all registered handlers
func (s *Service) ListHandlers() []chain.HandlerInfo {
	return s.chain.ListHandlers()
}

// CreatePipeline creates a new pipeline configuration
func (s *Service) CreatePipeline(p *chain.Pipeline) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.pipelines[p.ID]; exists {
		return mdwerror.New("pipeline already exists").
			WithCode(mdwerror.CodeDuplicateEntry).
			WithOperation("service.CreatePipeline").
			WithDetail("pipeline_id", p.ID)
	}

	p.CreatedAt = time.Now()
	p.UpdatedAt = time.Now()
	s.pipelines[p.ID] = p

	s.logger.Info("Pipeline created",
		"pipeline_id", p.ID,
		"name", p.Name)

	return nil
}

// GetPipeline returns a pipeline by ID
func (s *Service) GetPipeline(id string) (*chain.Pipeline, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	p, exists := s.pipelines[id]
	if !exists {
		return nil, mdwerror.New("pipeline not found").
			WithCode(mdwerror.CodeNotFound).
			WithOperation("service.GetPipeline").
			WithDetail("pipeline_id", id)
	}

	return p, nil
}

// UpdatePipeline updates an existing pipeline
func (s *Service) UpdatePipeline(p *chain.Pipeline) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	existing, exists := s.pipelines[p.ID]
	if !exists {
		return mdwerror.New("pipeline not found").
			WithCode(mdwerror.CodeNotFound).
			WithOperation("service.UpdatePipeline").
			WithDetail("pipeline_id", p.ID)
	}

	p.CreatedAt = existing.CreatedAt
	p.UpdatedAt = time.Now()
	s.pipelines[p.ID] = p

	s.logger.Info("Pipeline updated",
		"pipeline_id", p.ID,
		"name", p.Name)

	return nil
}

// DeletePipeline removes a pipeline
func (s *Service) DeletePipeline(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.pipelines[id]; !exists {
		return mdwerror.New("pipeline not found").
			WithCode(mdwerror.CodeNotFound).
			WithOperation("service.DeletePipeline").
			WithDetail("pipeline_id", id)
	}

	delete(s.pipelines, id)

	s.logger.Info("Pipeline deleted", "pipeline_id", id)

	return nil
}

// ListPipelines returns all pipelines
func (s *Service) ListPipelines() []*chain.Pipeline {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]*chain.Pipeline, 0, len(s.pipelines))
	for _, p := range s.pipelines {
		result = append(result, p)
	}

	return result
}

// ProcessPre executes pre-processing on a request
func (s *Service) ProcessPre(ctx context.Context, req *chain.ProcessRequest) (*chain.ProcessResult, error) {
	if req.RequestID == "" {
		req.RequestID = uuid.New().String()
	}

	pctx := chain.NewProcessingContext(ctx, req.RequestID, req.PipelineID, req.Prompt)

	// Copy metadata
	for k, v := range req.Metadata {
		pctx.SetMetadata(k, v)
	}

	if err := s.chain.ProcessPre(pctx); err != nil {
		return nil, mdwerror.Wrap(err, "pre-processing failed").
			WithCode(mdwerror.CodeInternal).
			WithOperation("service.ProcessPre")
	}

	return pctx.ToResult(), nil
}

// ProcessPost executes post-processing on a response
func (s *Service) ProcessPost(ctx context.Context, req *chain.ProcessRequest) (*chain.ProcessResult, error) {
	if req.RequestID == "" {
		req.RequestID = uuid.New().String()
	}

	pctx := chain.NewProcessingContext(ctx, req.RequestID, req.PipelineID, req.Prompt)
	pctx.Response = req.Response

	// Copy metadata
	for k, v := range req.Metadata {
		pctx.SetMetadata(k, v)
	}

	if err := s.chain.ProcessPost(pctx); err != nil {
		return nil, mdwerror.Wrap(err, "post-processing failed").
			WithCode(mdwerror.CodeInternal).
			WithOperation("service.ProcessPost")
	}

	return pctx.ToResult(), nil
}

// Process executes the complete pipeline (pre + main + post)
func (s *Service) Process(ctx context.Context, req *chain.ProcessRequest, mainProcessor func(context.Context, string) (string, error)) (*chain.ProcessResult, error) {
	if req.RequestID == "" {
		req.RequestID = uuid.New().String()
	}

	result, err := s.chain.Process(ctx, req, mainProcessor)
	if err != nil {
		return nil, mdwerror.Wrap(err, "pipeline processing failed").
			WithCode(mdwerror.CodeInternal).
			WithOperation("service.Process")
	}

	return result, nil
}

// Chain returns the underlying chain for direct access
func (s *Service) Chain() *chain.Chain {
	return s.chain
}

// Stats returns service statistics
func (s *Service) Stats() map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return map[string]interface{}{
		"total_handlers":    s.chain.TotalHandlerCount(),
		"pre_handlers":      s.chain.PreHandlerCount(),
		"post_handlers":     s.chain.PostHandlerCount(),
		"pipeline_count":    len(s.pipelines),
		"default_pipeline":  s.config.DefaultPipeline,
	}
}

// Close cleans up service resources
func (s *Service) Close() error {
	s.logger.Info("Platon service closed")
	return nil
}

// LoadDefaultPipeline creates a default pipeline if not exists
func (s *Service) LoadDefaultPipeline() error {
	defaultPipeline := &chain.Pipeline{
		ID:           "default",
		Name:         "Default Pipeline",
		Description:  "Default processing pipeline with standard handlers",
		Enabled:      true,
		PreHandlers:  []string{},
		PostHandlers: []string{},
		Config:       make(map[string]string),
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.pipelines["default"]; !exists {
		defaultPipeline.CreatedAt = time.Now()
		defaultPipeline.UpdatedAt = time.Now()
		s.pipelines["default"] = defaultPipeline

		s.logger.Info("Default pipeline loaded", "pipeline_id", "default")
	}

	return nil
}

// ValidateRequest validates a process request
func (s *Service) ValidateRequest(req *chain.ProcessRequest) error {
	if req.Prompt == "" && req.Response == "" {
		return fmt.Errorf("either prompt or response must be provided")
	}
	return nil
}

// ============================================================================
// Policy Management
// ============================================================================

// CreatePolicy creates a new policy
func (s *Service) CreatePolicy(p *Policy) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.policies[p.ID]; exists {
		return mdwerror.New("policy already exists").
			WithCode(mdwerror.CodeDuplicateEntry).
			WithOperation("service.CreatePolicy").
			WithDetail("policy_id", p.ID)
	}

	p.CreatedAt = time.Now()
	p.UpdatedAt = time.Now()
	s.policies[p.ID] = p

	s.logger.Info("Policy created",
		"policy_id", p.ID,
		"name", p.Name,
		"type", p.Type)

	return nil
}

// GetPolicy returns a policy by ID
func (s *Service) GetPolicy(id string) (*Policy, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	p, exists := s.policies[id]
	if !exists {
		return nil, mdwerror.New("policy not found").
			WithCode(mdwerror.CodeNotFound).
			WithOperation("service.GetPolicy").
			WithDetail("policy_id", id)
	}

	return p, nil
}

// UpdatePolicy updates an existing policy
func (s *Service) UpdatePolicy(p *Policy) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	existing, exists := s.policies[p.ID]
	if !exists {
		return mdwerror.New("policy not found").
			WithCode(mdwerror.CodeNotFound).
			WithOperation("service.UpdatePolicy").
			WithDetail("policy_id", p.ID)
	}

	p.CreatedAt = existing.CreatedAt
	p.UpdatedAt = time.Now()
	s.policies[p.ID] = p

	s.logger.Info("Policy updated",
		"policy_id", p.ID,
		"name", p.Name)

	return nil
}

// DeletePolicy removes a policy
func (s *Service) DeletePolicy(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.policies[id]; !exists {
		return mdwerror.New("policy not found").
			WithCode(mdwerror.CodeNotFound).
			WithOperation("service.DeletePolicy").
			WithDetail("policy_id", id)
	}

	delete(s.policies, id)

	s.logger.Info("Policy deleted", "policy_id", id)

	return nil
}

// ListPolicies returns all policies
func (s *Service) ListPolicies() []*Policy {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]*Policy, 0, len(s.policies))
	for _, p := range s.policies {
		result = append(result, p)
	}

	return result
}

// TestPolicyResult holds the result of testing a policy
type TestPolicyResult struct {
	Decision     string
	Violations   []PolicyViolationResult
	ModifiedText string
	Reason       string
	Duration     time.Duration
}

// PolicyViolationResult holds a single policy violation
type PolicyViolationResult struct {
	PolicyID    string
	PolicyName  string
	RuleID      string
	Severity    string
	Description string
	Location    string
	Action      string
	Matched     string
}

// TestPolicy tests a policy against sample text
func (s *Service) TestPolicy(p *Policy, testText string) (*TestPolicyResult, error) {
	startTime := time.Now()

	result := &TestPolicyResult{
		Decision:     "allow",
		Violations:   make([]PolicyViolationResult, 0),
		ModifiedText: testText,
	}

	// Evaluate rules against test text
	for _, rule := range p.Rules {
		pattern, err := compilePattern(rule.Pattern, rule.CaseSensitive)
		if err != nil {
			return nil, mdwerror.Wrap(err, "invalid rule pattern").
				WithCode(mdwerror.CodeInvalidInput).
				WithOperation("service.TestPolicy").
				WithDetail("rule_id", rule.ID)
		}

		matches := pattern.FindAllString(testText, -1)
		for _, match := range matches {
			violation := PolicyViolationResult{
				PolicyID:    p.ID,
				PolicyName:  p.Name,
				RuleID:      rule.ID,
				Severity:    getSeverityForAction(rule.Action),
				Description: rule.Message,
				Action:      rule.Action,
				Matched:     match,
			}
			result.Violations = append(result.Violations, violation)

			switch rule.Action {
			case "block":
				result.Decision = "block"
				result.Reason = rule.Message
			case "redact":
				if result.Decision != "block" {
					result.Decision = "modify"
				}
				replacement := rule.Replacement
				if replacement == "" {
					replacement = "[REDACTED]"
				}
				result.ModifiedText = pattern.ReplaceAllString(result.ModifiedText, replacement)
			case "warn":
				if result.Decision != "block" && result.Decision != "modify" {
					result.Decision = "escalate"
					result.Reason = rule.Message
				}
			}
		}
	}

	result.Duration = time.Since(startTime)
	return result, nil
}

// compilePattern compiles a regex pattern with optional case sensitivity
func compilePattern(pattern string, caseSensitive bool) (*regexp.Regexp, error) {
	if !caseSensitive {
		pattern = "(?i)" + pattern
	}
	return regexp.Compile(pattern)
}

// getSeverityForAction returns severity based on action
func getSeverityForAction(action string) string {
	switch action {
	case "block":
		return "critical"
	case "redact":
		return "high"
	case "warn":
		return "medium"
	case "log":
		return "low"
	default:
		return "info"
	}
}
