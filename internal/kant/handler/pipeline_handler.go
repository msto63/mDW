// ============================================================================
// meinDENKWERK (mDW) - Lokale KI-Plattform
// ============================================================================
//
// Package:     handler
// Description: REST API handlers for pipeline processing via Platon
// Author:      Mike Stoffels with Claude
// Created:     2025-12-08
// License:     MIT
// ============================================================================

package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/msto63/mDW/api/gen/common"
	platonpb "github.com/msto63/mDW/api/gen/platon"
)

// ============================================================================
// Pipeline Processing REST Types
// ============================================================================

// ProcessPipelineRequest represents a pipeline processing request
type ProcessPipelineRequest struct {
	PipelineID string            `json:"pipeline_id,omitempty"`
	Prompt     string            `json:"prompt"`
	Response   string            `json:"response,omitempty"` // For post-processing
	Metadata   map[string]string `json:"metadata,omitempty"`
	Options    *PipelineOptions  `json:"options,omitempty"`
}

// PipelineOptions represents processing options
type PipelineOptions struct {
	SkipPreProcessing  bool `json:"skip_pre_processing,omitempty"`
	SkipPostProcessing bool `json:"skip_post_processing,omitempty"`
	DryRun             bool `json:"dry_run,omitempty"`
	TimeoutSeconds     int  `json:"timeout_seconds,omitempty"`
	Debug              bool `json:"debug,omitempty"`
}

// ProcessPipelineResponse represents a pipeline processing response
type ProcessPipelineResponse struct {
	RequestID         string       `json:"request_id"`
	ProcessedPrompt   string       `json:"processed_prompt,omitempty"`
	ProcessedResponse string       `json:"processed_response,omitempty"`
	Blocked           bool         `json:"blocked"`
	BlockReason       string       `json:"block_reason,omitempty"`
	Modified          bool         `json:"modified"`
	AuditLog          []AuditEntry `json:"audit_log,omitempty"`
	DurationMs        int64        `json:"duration_ms"`
}

// AuditEntry represents an audit entry
type AuditEntry struct {
	Handler    string            `json:"handler"`
	Phase      string            `json:"phase"`
	DurationMs int64             `json:"duration_ms"`
	Error      string            `json:"error,omitempty"`
	Modified   bool              `json:"modified"`
	Details    map[string]string `json:"details,omitempty"`
}

// PipelineDefinitionRequest represents a pipeline definition create/update request
type PipelineDefinitionRequest struct {
	ID           string            `json:"id,omitempty"`
	Name         string            `json:"name"`
	Description  string            `json:"description,omitempty"`
	Enabled      bool              `json:"enabled"`
	PreHandlers  []string          `json:"pre_handlers,omitempty"`
	PostHandlers []string          `json:"post_handlers,omitempty"`
	Config       map[string]string `json:"config,omitempty"`
}

// PipelineDefinitionResponse represents a pipeline definition response
type PipelineDefinitionResponse struct {
	ID           string            `json:"id"`
	Name         string            `json:"name"`
	Description  string            `json:"description,omitempty"`
	Enabled      bool              `json:"enabled"`
	PreHandlers  []string          `json:"pre_handlers,omitempty"`
	PostHandlers []string          `json:"post_handlers,omitempty"`
	Config       map[string]string `json:"config,omitempty"`
	CreatedAt    string            `json:"created_at,omitempty"`
	UpdatedAt    string            `json:"updated_at,omitempty"`
}

// PipelineDefinitionsResponse represents a list of pipeline definitions
type PipelineDefinitionsResponse struct {
	Pipelines []PipelineDefinitionResponse `json:"pipelines"`
	Total     int                          `json:"total"`
}

// PolicyDefinitionRequest represents a policy create/update request
type PolicyDefinitionRequest struct {
	ID          string           `json:"id,omitempty"`
	Name        string           `json:"name"`
	Description string           `json:"description,omitempty"`
	PolicyType  string           `json:"policy_type"`
	Enabled     bool             `json:"enabled"`
	Priority    int              `json:"priority,omitempty"`
	Rules       []PolicyRuleInput `json:"rules,omitempty"`
	LLMCheck    *LLMCheckConfig  `json:"llm_check,omitempty"`
}

// PolicyRuleInput represents a policy rule input
type PolicyRuleInput struct {
	ID            string `json:"id,omitempty"`
	Pattern       string `json:"pattern"`
	Action        string `json:"action"`
	Message       string `json:"message,omitempty"`
	Replacement   string `json:"replacement,omitempty"`
	CaseSensitive bool   `json:"case_sensitive,omitempty"`
}

// LLMCheckConfig represents LLM check configuration
type LLMCheckConfig struct {
	Enabled        bool    `json:"enabled"`
	Model          string  `json:"model,omitempty"`
	Prompt         string  `json:"prompt,omitempty"`
	TimeoutSeconds int     `json:"timeout_seconds,omitempty"`
	Temperature    float32 `json:"temperature,omitempty"`
}

// PolicyDefinitionResponse represents a policy definition response
type PolicyDefinitionResponse struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Description string            `json:"description,omitempty"`
	PolicyType  string            `json:"policy_type"`
	Enabled     bool              `json:"enabled"`
	Priority    int               `json:"priority"`
	Rules       []PolicyRuleInput `json:"rules,omitempty"`
	LLMCheck    *LLMCheckConfig   `json:"llm_check,omitempty"`
	CreatedAt   string            `json:"created_at,omitempty"`
	UpdatedAt   string            `json:"updated_at,omitempty"`
}

// PolicyDefinitionsResponse represents a list of policy definitions
type PolicyDefinitionsResponse struct {
	Policies []PolicyDefinitionResponse `json:"policies"`
	Total    int                        `json:"total"`
}

// TestPolicyRequest represents a policy test request
type TestPolicyRequest struct {
	Policy   PolicyDefinitionRequest `json:"policy,omitempty"`
	PolicyID string                  `json:"policy_id,omitempty"`
	TestText string                  `json:"test_text"`
}

// TestPolicyResponse represents a policy test response
type TestPolicyResponse struct {
	Decision     string            `json:"decision"`
	Violations   []PolicyViolation `json:"violations,omitempty"`
	ModifiedText string            `json:"modified_text,omitempty"`
	Reason       string            `json:"reason,omitempty"`
	DurationMs   int64             `json:"duration_ms"`
}

// PolicyViolation represents a policy violation
type PolicyViolation struct {
	PolicyID    string `json:"policy_id"`
	PolicyName  string `json:"policy_name"`
	RuleID      string `json:"rule_id,omitempty"`
	Severity    string `json:"severity,omitempty"`
	Description string `json:"description"`
	Location    string `json:"location,omitempty"`
	Action      string `json:"action"`
	Matched     string `json:"matched,omitempty"`
}

// HandlerDefinitionResponse represents a handler definition response
type HandlerDefinitionResponse struct {
	Name        string            `json:"name"`
	Type        string            `json:"type"`
	Priority    int               `json:"priority"`
	Description string            `json:"description,omitempty"`
	Enabled     bool              `json:"enabled"`
	Config      map[string]string `json:"config,omitempty"`
}

// HandlersListResponse represents a list of handlers
type HandlersListResponse struct {
	Handlers []HandlerDefinitionResponse `json:"handlers"`
	Total    int                         `json:"total"`
}

// ============================================================================
// Pipeline Processing Handlers
// ============================================================================

// HandlePipelineProcess handles POST /api/v1/pipeline/process
func (h *Handler) HandlePipelineProcess(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Use POST", "")
		return
	}

	var req ProcessPipelineRequest
	if err := h.readJSON(r, &req); err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON", err.Error())
		return
	}

	if req.Prompt == "" {
		h.writeError(w, http.StatusBadRequest, "invalid_request", "Prompt required", "")
		return
	}

	if h.clients.Platon == nil {
		h.writeError(w, http.StatusServiceUnavailable, "service_unavailable", "Platon service not available", "")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 120*time.Second)
	defer cancel()

	// Build gRPC request
	grpcReq := &platonpb.ProcessRequest{
		RequestId:  fmt.Sprintf("kant-%d", time.Now().UnixNano()),
		PipelineId: req.PipelineID,
		Prompt:     req.Prompt,
		Response:   req.Response,
		Metadata:   req.Metadata,
	}

	if req.Options != nil {
		grpcReq.Options = &platonpb.ProcessOptions{
			SkipPreProcessing:  req.Options.SkipPreProcessing,
			SkipPostProcessing: req.Options.SkipPostProcessing,
			DryRun:             req.Options.DryRun,
			TimeoutSeconds:     int32(req.Options.TimeoutSeconds),
			Debug:              req.Options.Debug,
		}
	}

	grpcResp, err := h.clients.Platon.Process(ctx, grpcReq)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, "internal_error", "Pipeline processing failed", err.Error())
		return
	}

	// Convert response
	resp := ProcessPipelineResponse{
		RequestID:         grpcResp.RequestId,
		ProcessedPrompt:   grpcResp.ProcessedPrompt,
		ProcessedResponse: grpcResp.ProcessedResponse,
		Blocked:           grpcResp.Blocked,
		BlockReason:       grpcResp.BlockReason,
		Modified:          grpcResp.Modified,
		DurationMs:        grpcResp.DurationMs,
	}

	resp.AuditLog = make([]AuditEntry, len(grpcResp.AuditLog))
	for i, entry := range grpcResp.AuditLog {
		resp.AuditLog[i] = AuditEntry{
			Handler:    entry.Handler,
			Phase:      entry.Phase,
			DurationMs: entry.DurationMs,
			Error:      entry.Error,
			Modified:   entry.Modified,
			Details:    entry.Details,
		}
	}

	h.writeJSON(w, http.StatusOK, resp)
}

// HandlePipelineProcessPre handles POST /api/v1/pipeline/process/pre
func (h *Handler) HandlePipelineProcessPre(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Use POST", "")
		return
	}

	var req ProcessPipelineRequest
	if err := h.readJSON(r, &req); err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON", err.Error())
		return
	}

	if req.Prompt == "" {
		h.writeError(w, http.StatusBadRequest, "invalid_request", "Prompt required", "")
		return
	}

	if h.clients.Platon == nil {
		h.writeError(w, http.StatusServiceUnavailable, "service_unavailable", "Platon service not available", "")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 60*time.Second)
	defer cancel()

	grpcReq := &platonpb.ProcessRequest{
		RequestId:  fmt.Sprintf("kant-%d", time.Now().UnixNano()),
		PipelineId: req.PipelineID,
		Prompt:     req.Prompt,
		Metadata:   req.Metadata,
	}

	grpcResp, err := h.clients.Platon.ProcessPre(ctx, grpcReq)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, "internal_error", "Pre-processing failed", err.Error())
		return
	}

	resp := ProcessPipelineResponse{
		RequestID:       grpcResp.RequestId,
		ProcessedPrompt: grpcResp.ProcessedPrompt,
		Blocked:         grpcResp.Blocked,
		BlockReason:     grpcResp.BlockReason,
		Modified:        grpcResp.Modified,
		DurationMs:      grpcResp.DurationMs,
	}

	resp.AuditLog = make([]AuditEntry, len(grpcResp.AuditLog))
	for i, entry := range grpcResp.AuditLog {
		resp.AuditLog[i] = AuditEntry{
			Handler:    entry.Handler,
			Phase:      entry.Phase,
			DurationMs: entry.DurationMs,
			Error:      entry.Error,
			Modified:   entry.Modified,
			Details:    entry.Details,
		}
	}

	h.writeJSON(w, http.StatusOK, resp)
}

// HandlePipelineProcessPost handles POST /api/v1/pipeline/process/post
func (h *Handler) HandlePipelineProcessPost(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Use POST", "")
		return
	}

	var req ProcessPipelineRequest
	if err := h.readJSON(r, &req); err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON", err.Error())
		return
	}

	if req.Response == "" {
		h.writeError(w, http.StatusBadRequest, "invalid_request", "Response required for post-processing", "")
		return
	}

	if h.clients.Platon == nil {
		h.writeError(w, http.StatusServiceUnavailable, "service_unavailable", "Platon service not available", "")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 60*time.Second)
	defer cancel()

	grpcReq := &platonpb.ProcessRequest{
		RequestId:  fmt.Sprintf("kant-%d", time.Now().UnixNano()),
		PipelineId: req.PipelineID,
		Prompt:     req.Prompt,
		Response:   req.Response,
		Metadata:   req.Metadata,
	}

	grpcResp, err := h.clients.Platon.ProcessPost(ctx, grpcReq)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, "internal_error", "Post-processing failed", err.Error())
		return
	}

	resp := ProcessPipelineResponse{
		RequestID:         grpcResp.RequestId,
		ProcessedResponse: grpcResp.ProcessedResponse,
		Blocked:           grpcResp.Blocked,
		BlockReason:       grpcResp.BlockReason,
		Modified:          grpcResp.Modified,
		DurationMs:        grpcResp.DurationMs,
	}

	resp.AuditLog = make([]AuditEntry, len(grpcResp.AuditLog))
	for i, entry := range grpcResp.AuditLog {
		resp.AuditLog[i] = AuditEntry{
			Handler:    entry.Handler,
			Phase:      entry.Phase,
			DurationMs: entry.DurationMs,
			Error:      entry.Error,
			Modified:   entry.Modified,
			Details:    entry.Details,
		}
	}

	h.writeJSON(w, http.StatusOK, resp)
}

// ============================================================================
// Handler Management
// ============================================================================

// HandleHandlers handles GET /api/v1/pipeline/handlers
func (h *Handler) HandleHandlers(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Use GET", "")
		return
	}

	if h.clients.Platon == nil {
		h.writeError(w, http.StatusServiceUnavailable, "service_unavailable", "Platon service not available", "")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	grpcResp, err := h.clients.Platon.ListHandlers(ctx, &common.Empty{})
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, "internal_error", "Failed to list handlers", err.Error())
		return
	}

	handlers := make([]HandlerDefinitionResponse, len(grpcResp.Handlers))
	for i, handler := range grpcResp.Handlers {
		handlers[i] = HandlerDefinitionResponse{
			Name:        handler.Name,
			Type:        handler.Type.String(),
			Priority:    int(handler.Priority),
			Description: handler.Description,
			Enabled:     handler.Enabled,
			Config:      handler.Config,
		}
	}

	h.writeJSON(w, http.StatusOK, HandlersListResponse{
		Handlers: handlers,
		Total:    int(grpcResp.Total),
	})
}

// ============================================================================
// Pipeline Definition Handlers
// ============================================================================

// HandlePipelineDefinitions handles GET/POST /api/v1/pipeline/pipelines
func (h *Handler) HandlePipelineDefinitions(w http.ResponseWriter, r *http.Request) {
	if h.clients.Platon == nil {
		h.writeError(w, http.StatusServiceUnavailable, "service_unavailable", "Platon service not available", "")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	switch r.Method {
	case http.MethodGet:
		grpcResp, err := h.clients.Platon.ListPipelines(ctx, &common.Empty{})
		if err != nil {
			h.writeError(w, http.StatusInternalServerError, "internal_error", "Failed to list pipelines", err.Error())
			return
		}

		pipelines := make([]PipelineDefinitionResponse, len(grpcResp.Pipelines))
		for i, p := range grpcResp.Pipelines {
			pipelines[i] = pipelineInfoToResponse(p)
		}

		h.writeJSON(w, http.StatusOK, PipelineDefinitionsResponse{
			Pipelines: pipelines,
			Total:     int(grpcResp.Total),
		})

	case http.MethodPost:
		var req PipelineDefinitionRequest
		if err := h.readJSON(r, &req); err != nil {
			h.writeError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON", err.Error())
			return
		}

		grpcReq := &platonpb.CreatePipelineRequest{
			Id:           req.ID,
			Name:         req.Name,
			Description:  req.Description,
			Enabled:      req.Enabled,
			PreHandlers:  req.PreHandlers,
			PostHandlers: req.PostHandlers,
			Config:       req.Config,
		}

		grpcResp, err := h.clients.Platon.CreatePipeline(ctx, grpcReq)
		if err != nil {
			h.writeError(w, http.StatusInternalServerError, "internal_error", "Failed to create pipeline", err.Error())
			return
		}

		h.writeJSON(w, http.StatusCreated, pipelineInfoToResponse(grpcResp))

	default:
		h.writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Use GET or POST", "")
	}
}

// HandlePipelineDefinition handles GET/PUT/DELETE /api/v1/pipeline/pipelines/{id}
func (h *Handler) HandlePipelineDefinition(w http.ResponseWriter, r *http.Request, id string) {
	if h.clients.Platon == nil {
		h.writeError(w, http.StatusServiceUnavailable, "service_unavailable", "Platon service not available", "")
		return
	}

	id = strings.TrimSuffix(id, "/")
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	switch r.Method {
	case http.MethodGet:
		grpcResp, err := h.clients.Platon.GetPipeline(ctx, &platonpb.GetPipelineRequest{Id: id})
		if err != nil {
			h.writeError(w, http.StatusNotFound, "not_found", "Pipeline not found", err.Error())
			return
		}
		h.writeJSON(w, http.StatusOK, pipelineInfoToResponse(grpcResp))

	case http.MethodPut:
		var req PipelineDefinitionRequest
		if err := h.readJSON(r, &req); err != nil {
			h.writeError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON", err.Error())
			return
		}
		req.ID = id

		grpcReq := &platonpb.UpdatePipelineRequest{
			Id:           id,
			Name:         req.Name,
			Description:  req.Description,
			Enabled:      req.Enabled,
			PreHandlers:  req.PreHandlers,
			PostHandlers: req.PostHandlers,
			Config:       req.Config,
		}

		grpcResp, err := h.clients.Platon.UpdatePipeline(ctx, grpcReq)
		if err != nil {
			h.writeError(w, http.StatusInternalServerError, "internal_error", "Failed to update pipeline", err.Error())
			return
		}
		h.writeJSON(w, http.StatusOK, pipelineInfoToResponse(grpcResp))

	case http.MethodDelete:
		_, err := h.clients.Platon.DeletePipeline(ctx, &platonpb.DeletePipelineRequest{Id: id})
		if err != nil {
			h.writeError(w, http.StatusInternalServerError, "internal_error", "Failed to delete pipeline", err.Error())
			return
		}
		h.writeJSON(w, http.StatusOK, map[string]interface{}{
			"success": true,
			"message": fmt.Sprintf("Pipeline '%s' deleted", id),
		})

	default:
		h.writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Use GET, PUT, or DELETE", "")
	}
}

// ============================================================================
// Policy Definition Handlers
// ============================================================================

// HandlePolicyDefinitions handles GET/POST /api/v1/pipeline/policies
func (h *Handler) HandlePolicyDefinitions(w http.ResponseWriter, r *http.Request) {
	if h.clients.Platon == nil {
		h.writeError(w, http.StatusServiceUnavailable, "service_unavailable", "Platon service not available", "")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	switch r.Method {
	case http.MethodGet:
		grpcResp, err := h.clients.Platon.ListPolicies(ctx, &common.Empty{})
		if err != nil {
			h.writeError(w, http.StatusInternalServerError, "internal_error", "Failed to list policies", err.Error())
			return
		}

		policies := make([]PolicyDefinitionResponse, len(grpcResp.Policies))
		for i, p := range grpcResp.Policies {
			policies[i] = policyInfoToResponse(p)
		}

		h.writeJSON(w, http.StatusOK, PolicyDefinitionsResponse{
			Policies: policies,
			Total:    int(grpcResp.Total),
		})

	case http.MethodPost:
		var req PolicyDefinitionRequest
		if err := h.readJSON(r, &req); err != nil {
			h.writeError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON", err.Error())
			return
		}

		grpcReq := policyRequestToProto(&req)
		grpcResp, err := h.clients.Platon.CreatePolicy(ctx, grpcReq)
		if err != nil {
			h.writeError(w, http.StatusInternalServerError, "internal_error", "Failed to create policy", err.Error())
			return
		}

		h.writeJSON(w, http.StatusCreated, policyInfoToResponse(grpcResp))

	default:
		h.writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Use GET or POST", "")
	}
}

// HandlePolicyDefinition handles GET/PUT/DELETE /api/v1/pipeline/policies/{id}
func (h *Handler) HandlePolicyDefinition(w http.ResponseWriter, r *http.Request, id string) {
	if h.clients.Platon == nil {
		h.writeError(w, http.StatusServiceUnavailable, "service_unavailable", "Platon service not available", "")
		return
	}

	id = strings.TrimSuffix(id, "/")
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	switch r.Method {
	case http.MethodGet:
		grpcResp, err := h.clients.Platon.GetPolicy(ctx, &platonpb.GetPolicyRequest{Id: id})
		if err != nil {
			h.writeError(w, http.StatusNotFound, "not_found", "Policy not found", err.Error())
			return
		}
		h.writeJSON(w, http.StatusOK, policyInfoToResponse(grpcResp))

	case http.MethodPut:
		var req PolicyDefinitionRequest
		if err := h.readJSON(r, &req); err != nil {
			h.writeError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON", err.Error())
			return
		}
		req.ID = id

		grpcReq := &platonpb.UpdatePolicyRequest{
			Id:          id,
			Name:        req.Name,
			Description: req.Description,
			Type:        stringToPolicyType(req.PolicyType),
			Enabled:     req.Enabled,
			Priority:    int32(req.Priority),
			Rules:       policyRulesToProto(req.Rules),
		}
		if req.LLMCheck != nil {
			grpcReq.LlmCheck = &platonpb.LLMCheckConfig{
				Enabled:        req.LLMCheck.Enabled,
				Model:          req.LLMCheck.Model,
				Prompt:         req.LLMCheck.Prompt,
				TimeoutSeconds: int32(req.LLMCheck.TimeoutSeconds),
				Temperature:    req.LLMCheck.Temperature,
			}
		}

		grpcResp, err := h.clients.Platon.UpdatePolicy(ctx, grpcReq)
		if err != nil {
			h.writeError(w, http.StatusInternalServerError, "internal_error", "Failed to update policy", err.Error())
			return
		}
		h.writeJSON(w, http.StatusOK, policyInfoToResponse(grpcResp))

	case http.MethodDelete:
		_, err := h.clients.Platon.DeletePolicy(ctx, &platonpb.DeletePolicyRequest{Id: id})
		if err != nil {
			h.writeError(w, http.StatusInternalServerError, "internal_error", "Failed to delete policy", err.Error())
			return
		}
		h.writeJSON(w, http.StatusOK, map[string]interface{}{
			"success": true,
			"message": fmt.Sprintf("Policy '%s' deleted", id),
		})

	default:
		h.writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Use GET, PUT, or DELETE", "")
	}
}

// HandlePolicyTest handles POST /api/v1/pipeline/policies/test
func (h *Handler) HandlePolicyTest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Use POST", "")
		return
	}

	var req TestPolicyRequest
	if err := h.readJSON(r, &req); err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON", err.Error())
		return
	}

	if req.TestText == "" {
		h.writeError(w, http.StatusBadRequest, "invalid_request", "Test text required", "")
		return
	}

	if h.clients.Platon == nil {
		h.writeError(w, http.StatusServiceUnavailable, "service_unavailable", "Platon service not available", "")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	grpcReq := &platonpb.TestPolicyRequest{
		TestText: req.TestText,
	}

	// If policy is provided inline, include it
	if req.Policy.Name != "" {
		grpcReq.Policy = &platonpb.PolicyInfo{
			Id:          req.Policy.ID,
			Name:        req.Policy.Name,
			Description: req.Policy.Description,
			Type:        stringToPolicyType(req.Policy.PolicyType),
			Enabled:     req.Policy.Enabled,
			Priority:    int32(req.Policy.Priority),
			Rules:       policyRulesToProto(req.Policy.Rules),
		}
		if req.Policy.LLMCheck != nil {
			grpcReq.Policy.LlmCheck = &platonpb.LLMCheckConfig{
				Enabled:        req.Policy.LLMCheck.Enabled,
				Model:          req.Policy.LLMCheck.Model,
				Prompt:         req.Policy.LLMCheck.Prompt,
				TimeoutSeconds: int32(req.Policy.LLMCheck.TimeoutSeconds),
				Temperature:    req.Policy.LLMCheck.Temperature,
			}
		}
	}

	grpcResp, err := h.clients.Platon.TestPolicy(ctx, grpcReq)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, "internal_error", "Policy test failed", err.Error())
		return
	}

	violations := make([]PolicyViolation, len(grpcResp.Violations))
	for i, v := range grpcResp.Violations {
		violations[i] = PolicyViolation{
			PolicyID:    v.PolicyId,
			PolicyName:  v.PolicyName,
			RuleID:      v.RuleId,
			Severity:    v.Severity,
			Description: v.Description,
			Location:    v.Location,
			Action:      v.Action.String(),
			Matched:     v.Matched,
		}
	}

	resp := TestPolicyResponse{
		Decision:     grpcResp.Decision.String(),
		Violations:   violations,
		ModifiedText: grpcResp.ModifiedText,
		Reason:       grpcResp.Reason,
		DurationMs:   grpcResp.DurationMs,
	}

	h.writeJSON(w, http.StatusOK, resp)
}

// ============================================================================
// Helper Functions
// ============================================================================

func (h *Handler) writeSSEEvent(w http.ResponseWriter, event string, data interface{}) {
	jsonData, _ := json.Marshal(data)
	fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event, jsonData)
}

func pipelineInfoToResponse(p *platonpb.PipelineInfo) PipelineDefinitionResponse {
	resp := PipelineDefinitionResponse{
		ID:           p.Id,
		Name:         p.Name,
		Description:  p.Description,
		Enabled:      p.Enabled,
		PreHandlers:  p.PreHandlers,
		PostHandlers: p.PostHandlers,
		Config:       p.Config,
	}

	if p.CreatedAt > 0 {
		resp.CreatedAt = time.Unix(p.CreatedAt, 0).Format(time.RFC3339)
	}
	if p.UpdatedAt > 0 {
		resp.UpdatedAt = time.Unix(p.UpdatedAt, 0).Format(time.RFC3339)
	}

	return resp
}

func policyInfoToResponse(p *platonpb.PolicyInfo) PolicyDefinitionResponse {
	resp := PolicyDefinitionResponse{
		ID:          p.Id,
		Name:        p.Name,
		Description: p.Description,
		PolicyType:  p.Type.String(),
		Enabled:     p.Enabled,
		Priority:    int(p.Priority),
	}

	resp.Rules = policyRulesFromProto(p.Rules)

	if p.LlmCheck != nil {
		resp.LLMCheck = &LLMCheckConfig{
			Enabled:        p.LlmCheck.Enabled,
			Model:          p.LlmCheck.Model,
			Prompt:         p.LlmCheck.Prompt,
			TimeoutSeconds: int(p.LlmCheck.TimeoutSeconds),
			Temperature:    p.LlmCheck.Temperature,
		}
	}

	if p.CreatedAt > 0 {
		resp.CreatedAt = time.Unix(p.CreatedAt, 0).Format(time.RFC3339)
	}
	if p.UpdatedAt > 0 {
		resp.UpdatedAt = time.Unix(p.UpdatedAt, 0).Format(time.RFC3339)
	}

	return resp
}

func policyRequestToProto(req *PolicyDefinitionRequest) *platonpb.CreatePolicyRequest {
	grpcReq := &platonpb.CreatePolicyRequest{
		Id:          req.ID,
		Name:        req.Name,
		Description: req.Description,
		Type:        stringToPolicyType(req.PolicyType),
		Enabled:     req.Enabled,
		Priority:    int32(req.Priority),
		Rules:       policyRulesToProto(req.Rules),
	}

	if req.LLMCheck != nil {
		grpcReq.LlmCheck = &platonpb.LLMCheckConfig{
			Enabled:        req.LLMCheck.Enabled,
			Model:          req.LLMCheck.Model,
			Prompt:         req.LLMCheck.Prompt,
			TimeoutSeconds: int32(req.LLMCheck.TimeoutSeconds),
			Temperature:    req.LLMCheck.Temperature,
		}
	}

	return grpcReq
}

func policyRulesToProto(rules []PolicyRuleInput) []*platonpb.PolicyRule {
	result := make([]*platonpb.PolicyRule, len(rules))
	for i, r := range rules {
		result[i] = &platonpb.PolicyRule{
			Id:            r.ID,
			Pattern:       r.Pattern,
			Action:        stringToPolicyAction(r.Action),
			Message:       r.Message,
			Replacement:   r.Replacement,
			CaseSensitive: r.CaseSensitive,
		}
	}
	return result
}

func policyRulesFromProto(rules []*platonpb.PolicyRule) []PolicyRuleInput {
	result := make([]PolicyRuleInput, len(rules))
	for i, r := range rules {
		result[i] = PolicyRuleInput{
			ID:            r.Id,
			Pattern:       r.Pattern,
			Action:        r.Action.String(),
			Message:       r.Message,
			Replacement:   r.Replacement,
			CaseSensitive: r.CaseSensitive,
		}
	}
	return result
}

func stringToPolicyType(s string) platonpb.PolicyType {
	s = strings.ToUpper(strings.TrimSpace(s))
	switch s {
	case "CONTENT", "POLICY_TYPE_CONTENT":
		return platonpb.PolicyType_POLICY_TYPE_CONTENT
	case "SAFETY", "POLICY_TYPE_SAFETY":
		return platonpb.PolicyType_POLICY_TYPE_SAFETY
	case "SCOPE", "POLICY_TYPE_SCOPE":
		return platonpb.PolicyType_POLICY_TYPE_SCOPE
	case "PII", "POLICY_TYPE_PII":
		return platonpb.PolicyType_POLICY_TYPE_PII
	case "CUSTOM", "POLICY_TYPE_CUSTOM":
		return platonpb.PolicyType_POLICY_TYPE_CUSTOM
	default:
		return platonpb.PolicyType_POLICY_TYPE_UNKNOWN
	}
}

func stringToPolicyAction(s string) platonpb.PolicyAction {
	s = strings.ToUpper(strings.TrimSpace(s))
	switch s {
	case "BLOCK", "POLICY_ACTION_BLOCK":
		return platonpb.PolicyAction_POLICY_ACTION_BLOCK
	case "ALLOW", "POLICY_ACTION_ALLOW":
		return platonpb.PolicyAction_POLICY_ACTION_ALLOW
	case "REDACT", "POLICY_ACTION_REDACT":
		return platonpb.PolicyAction_POLICY_ACTION_REDACT
	case "WARN", "POLICY_ACTION_WARN":
		return platonpb.PolicyAction_POLICY_ACTION_WARN
	case "LOG", "POLICY_ACTION_LOG":
		return platonpb.PolicyAction_POLICY_ACTION_LOG
	default:
		return platonpb.PolicyAction_POLICY_ACTION_UNKNOWN
	}
}
