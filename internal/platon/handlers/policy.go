package handlers

import (
	"context"
	"fmt"
	"regexp"
	"sync"
	"time"

	"github.com/msto63/mDW/internal/platon/chain"
	"github.com/msto63/mDW/pkg/core/logging"
)

// PolicyType defines the type of policy
type PolicyType string

const (
	PolicyTypeContent PolicyType = "content"
	PolicyTypeSafety  PolicyType = "safety"
	PolicyTypeScope   PolicyType = "scope"
	PolicyTypePII     PolicyType = "pii"
	PolicyTypeCustom  PolicyType = "custom"
)

// PolicyAction defines the action to take when a policy is violated
type PolicyAction string

const (
	PolicyActionBlock  PolicyAction = "block"
	PolicyActionAllow  PolicyAction = "allow"
	PolicyActionRedact PolicyAction = "redact"
	PolicyActionWarn   PolicyAction = "warn"
	PolicyActionLog    PolicyAction = "log"
)

// PolicyRule defines a single rule within a policy
type PolicyRule struct {
	ID            string       `json:"id,omitempty"`
	Pattern       string       `json:"pattern"`
	Action        PolicyAction `json:"action"`
	Message       string       `json:"message"`
	Replacement   string       `json:"replacement,omitempty"`
	CaseSensitive bool         `json:"case_sensitive,omitempty"`
}

// PolicyConfig holds configuration for a policy
type PolicyConfig struct {
	ID          string       `json:"id"`
	Name        string       `json:"name"`
	Description string       `json:"description"`
	Type        PolicyType   `json:"type"`
	Enabled     bool         `json:"enabled"`
	Priority    int          `json:"priority"`
	Rules       []PolicyRule `json:"rules,omitempty"`
}

// PolicyViolation represents a detected policy violation
type PolicyViolation struct {
	PolicyID    string       `json:"policy_id"`
	PolicyName  string       `json:"policy_name"`
	RuleID      string       `json:"rule_id,omitempty"`
	Severity    string       `json:"severity"`
	Description string       `json:"description"`
	Location    string       `json:"location,omitempty"`
	Action      PolicyAction `json:"action"`
	Matched     string       `json:"matched,omitempty"`
}

// compiledRule holds a compiled regex pattern and its rule
type compiledRule struct {
	rule    PolicyRule
	pattern *regexp.Regexp
}

// PolicyHandler handles policy-based validation of prompts and responses
type PolicyHandler struct {
	*BaseHandler
	config        PolicyConfig
	compiledRules []*compiledRule
	logger        logging.Logger
	mu            sync.RWMutex
}

// NewPolicyHandler creates a new policy handler
func NewPolicyHandler(config PolicyConfig, logger logging.Logger) (*PolicyHandler, error) {
	h := &PolicyHandler{
		BaseHandler: NewBaseHandler(
			fmt.Sprintf("policy_%s", config.ID),
			chain.HandlerTypeBoth, // Policy checks both pre and post
			config.Priority,
		),
		config:        config,
		compiledRules: make([]*compiledRule, 0),
		logger:        logger,
	}

	// Compile regex patterns
	if err := h.compileRules(); err != nil {
		return nil, err
	}

	h.SetEnabled(config.Enabled)
	return h, nil
}

// compileRules compiles all regex patterns
func (h *PolicyHandler) compileRules() error {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.compiledRules = make([]*compiledRule, 0, len(h.config.Rules))

	for _, rule := range h.config.Rules {
		flags := ""
		if !rule.CaseSensitive {
			flags = "(?i)"
		}
		pattern, err := regexp.Compile(flags + rule.Pattern)
		if err != nil {
			return fmt.Errorf("invalid pattern in rule %s: %w", rule.ID, err)
		}
		h.compiledRules = append(h.compiledRules, &compiledRule{
			rule:    rule,
			pattern: pattern,
		})
	}

	return nil
}

// Process evaluates the policy against the current text
func (h *PolicyHandler) Process(ctx *chain.ProcessingContext) error {
	h.mu.RLock()
	rules := make([]*compiledRule, len(h.compiledRules))
	copy(rules, h.compiledRules)
	h.mu.RUnlock()

	text := ctx.CurrentText()
	modifiedText := text
	violations := make([]PolicyViolation, 0)

	for _, cr := range rules {
		matches := cr.pattern.FindAllString(text, -1)
		if len(matches) > 0 {
			for _, match := range matches {
				violation := PolicyViolation{
					PolicyID:    h.config.ID,
					PolicyName:  h.config.Name,
					RuleID:      cr.rule.ID,
					Severity:    getSeverity(cr.rule.Action),
					Description: cr.rule.Message,
					Action:      cr.rule.Action,
					Matched:     match,
				}
				violations = append(violations, violation)

				switch cr.rule.Action {
				case PolicyActionBlock:
					ctx.Block(cr.rule.Message)
					h.storeViolations(ctx, violations)
					h.logger.Info("Request blocked by policy",
						"policy_id", h.config.ID,
						"request_id", ctx.RequestID,
						"reason", cr.rule.Message)
					return nil

				case PolicyActionRedact:
					replacement := cr.rule.Replacement
					if replacement == "" {
						replacement = "[REDACTED]"
					}
					modifiedText = cr.pattern.ReplaceAllString(modifiedText, replacement)
					ctx.MarkModified()
				}
			}
		}
	}

	// Store violations in context state
	h.storeViolations(ctx, violations)

	// Update text if modified
	if modifiedText != text {
		ctx.SetCurrentText(modifiedText)
		h.logger.Debug("Text modified by policy",
			"policy_id", h.config.ID,
			"request_id", ctx.RequestID,
			"violations", len(violations))
	}

	return nil
}

// storeViolations stores violations in the context state
func (h *PolicyHandler) storeViolations(ctx *chain.ProcessingContext, violations []PolicyViolation) {
	if len(violations) == 0 {
		return
	}

	// Get existing violations
	existingVal, _ := ctx.GetState("policy_violations")
	existing, ok := existingVal.([]PolicyViolation)
	if !ok {
		existing = make([]PolicyViolation, 0)
	}

	// Append new violations
	existing = append(existing, violations...)
	ctx.SetState("policy_violations", existing)
}

// getSeverity returns severity based on action
func getSeverity(action PolicyAction) string {
	switch action {
	case PolicyActionBlock:
		return "critical"
	case PolicyActionRedact:
		return "high"
	case PolicyActionWarn:
		return "medium"
	case PolicyActionLog:
		return "low"
	default:
		return "info"
	}
}

// Config returns the policy configuration
func (h *PolicyHandler) Config() PolicyConfig {
	return h.config
}

// UpdateRules updates the policy rules
func (h *PolicyHandler) UpdateRules(rules []PolicyRule) error {
	h.config.Rules = rules
	return h.compileRules()
}

// Default PII patterns for common personal data
var DefaultPIIRules = []PolicyRule{
	{
		ID:          "email",
		Pattern:     `\b[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Z|a-z]{2,}\b`,
		Action:      PolicyActionRedact,
		Message:     "Email address detected",
		Replacement: "[EMAIL]",
	},
	{
		ID:          "phone_de",
		Pattern:     `\b(\+49|0049|0)[1-9][0-9]{1,14}\b`,
		Action:      PolicyActionRedact,
		Message:     "German phone number detected",
		Replacement: "[PHONE]",
	},
	{
		ID:          "iban",
		Pattern:     `\b[A-Z]{2}\d{2}[A-Z0-9]{4}\d{7}([A-Z0-9]?){0,16}\b`,
		Action:      PolicyActionRedact,
		Message:     "IBAN detected",
		Replacement: "[IBAN]",
	},
	{
		ID:          "credit_card",
		Pattern:     `\b(?:\d{4}[- ]?){3}\d{4}\b`,
		Action:      PolicyActionRedact,
		Message:     "Credit card number detected",
		Replacement: "[CREDIT_CARD]",
	},
}

// NewDefaultPIIHandler creates a handler with default PII detection rules
func NewDefaultPIIHandler(logger logging.Logger) (*PolicyHandler, error) {
	config := PolicyConfig{
		ID:          "default_pii",
		Name:        "Default PII Detection",
		Description: "Detects and redacts common personal identifiable information",
		Type:        PolicyTypePII,
		Enabled:     true,
		Priority:    100,
		Rules:       DefaultPIIRules,
	}
	return NewPolicyHandler(config, logger)
}

// LLMExecutor interface for LLM-based policy checks
type LLMExecutor interface {
	Execute(ctx context.Context, model string, prompt string, temperature float32) (string, error)
}

// LLMPolicyHandler extends PolicyHandler with LLM-based checking
type LLMPolicyHandler struct {
	*PolicyHandler
	llmExecutor LLMExecutor
	llmPrompt   string
	llmModel    string
	llmTimeout  time.Duration
}

// NewLLMPolicyHandler creates a policy handler with LLM-based checking
func NewLLMPolicyHandler(config PolicyConfig, executor LLMExecutor, llmPrompt string, logger logging.Logger) (*LLMPolicyHandler, error) {
	base, err := NewPolicyHandler(config, logger)
	if err != nil {
		return nil, err
	}

	return &LLMPolicyHandler{
		PolicyHandler: base,
		llmExecutor:   executor,
		llmPrompt:     llmPrompt,
		llmModel:      "llama3.2:3b",
		llmTimeout:    30 * time.Second,
	}, nil
}

// Process extends the base process with LLM checking
func (h *LLMPolicyHandler) Process(ctx *chain.ProcessingContext) error {
	// First run regex-based rules
	if err := h.PolicyHandler.Process(ctx); err != nil {
		return err
	}

	// If already blocked, skip LLM check
	if ctx.Blocked {
		return nil
	}

	// Run LLM check if executor is available
	if h.llmExecutor != nil && h.llmPrompt != "" {
		return h.runLLMCheck(ctx)
	}

	return nil
}

// runLLMCheck performs LLM-based content checking
func (h *LLMPolicyHandler) runLLMCheck(ctx *chain.ProcessingContext) error {
	llmCtx, cancel := context.WithTimeout(ctx.Context(), h.llmTimeout)
	defer cancel()

	text := ctx.CurrentText()
	prompt := fmt.Sprintf("%s\n\nText to analyze:\n%s", h.llmPrompt, text)

	response, err := h.llmExecutor.Execute(llmCtx, h.llmModel, prompt, 0.1)
	if err != nil {
		h.logger.Warn("LLM policy check failed", "error", err)
		return nil // Don't fail the pipeline on LLM errors
	}

	// Simple response parsing - look for "UNSAFE" or "BLOCK"
	if containsUnsafe(response) {
		ctx.Block("Content flagged as unsafe by LLM analysis")
		h.logger.Info("Request blocked by LLM policy check",
			"request_id", ctx.RequestID,
			"policy_id", h.config.ID)
	}

	return nil
}

// containsUnsafe checks if response indicates unsafe content
func containsUnsafe(response string) bool {
	lower := response
	return containsIgnoreCase(lower, "unsafe") ||
		containsIgnoreCase(lower, "block") ||
		containsIgnoreCase(lower, "harmful")
}

// containsIgnoreCase does case-insensitive contains check
func containsIgnoreCase(s, substr string) bool {
	return len(s) >= len(substr) &&
		(s == substr || len(regexp.MustCompile("(?i)"+regexp.QuoteMeta(substr)).FindString(s)) > 0)
}
