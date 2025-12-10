// ============================================================================
// meinDENKWERK (mDW) - Lokale KI-Plattform
// ============================================================================
//
// Package:     handler
// Description: REST API handlers for pipeline processing
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
	leibnizpb "github.com/msto63/mDW/api/gen/leibniz"
)

// ============================================================================
// Pipeline Processing REST Types
// ============================================================================

// ProcessPipelineRequest represents a pipeline processing request
type ProcessPipelineRequest struct {
	PipelineID string            `json:"pipeline_id,omitempty"`
	Prompt     string            `json:"prompt"`
	Metadata   map[string]string `json:"metadata,omitempty"`
	UserID     string            `json:"user_id,omitempty"`
	SessionID  string            `json:"session_id,omitempty"`
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
	RequestID       string           `json:"request_id"`
	Success         bool             `json:"success"`
	Response        string           `json:"response,omitempty"`
	ProcessedPrompt string           `json:"processed_prompt,omitempty"`
	Flags           *PipelineFlags   `json:"flags,omitempty"`
	StageResults    []StageResult    `json:"stage_results,omitempty"`
	DurationMs      int64            `json:"duration_ms"`
	Error           string           `json:"error,omitempty"`
}

// PipelineFlags represents processing flags
type PipelineFlags struct {
	Blocked        bool   `json:"blocked"`
	Modified       bool   `json:"modified"`
	Escalated      bool   `json:"escalated"`
	RequiresReview bool   `json:"requires_review"`
	BlockReason    string `json:"block_reason,omitempty"`
	ModifyReason   string `json:"modify_reason,omitempty"`
}

// StageResult represents a pipeline stage result
type StageResult struct {
	StageName  string            `json:"stage_name"`
	AgentID    string            `json:"agent_id,omitempty"`
	Role       string            `json:"role,omitempty"`
	Success    bool              `json:"success"`
	Decision   string            `json:"decision,omitempty"`
	Error      string            `json:"error,omitempty"`
	DurationMs int64             `json:"duration_ms"`
	Input      string            `json:"input,omitempty"`
	Output     map[string]string `json:"output,omitempty"`
	Skipped    bool              `json:"skipped,omitempty"`
	SkipReason string            `json:"skip_reason,omitempty"`
}

// PipelineDefinitionRequest represents a pipeline definition create/update request
type PipelineDefinitionRequest struct {
	ID             string             `json:"id,omitempty"`
	Name           string             `json:"name"`
	Description    string             `json:"description,omitempty"`
	Enabled        bool               `json:"enabled"`
	Inherit        string             `json:"inherit,omitempty"`
	PreProcessing  []StageConfigInput `json:"pre_processing,omitempty"`
	PostProcessing []StageConfigInput `json:"post_processing,omitempty"`
	Settings       *PipelineSettings  `json:"settings,omitempty"`
}

// StageConfigInput represents stage configuration input
type StageConfigInput struct {
	Name           string            `json:"name"`
	AgentID        string            `json:"agent_id"`
	Role           string            `json:"role"`
	Required       bool              `json:"required,omitempty"`
	OnFail         string            `json:"on_fail,omitempty"`
	Condition      string            `json:"condition,omitempty"`
	Priority       int               `json:"priority,omitempty"`
	TimeoutSeconds int               `json:"timeout_seconds,omitempty"`
	RetryCount     int               `json:"retry_count,omitempty"`
	Input          map[string]string `json:"input,omitempty"`
	OutputMapping  map[string]string `json:"output_mapping,omitempty"`
}

// PipelineSettings represents pipeline settings
type PipelineSettings struct {
	MaxStages           int  `json:"max_stages,omitempty"`
	StageTimeoutSeconds int  `json:"stage_timeout_seconds,omitempty"`
	TotalTimeoutSeconds int  `json:"total_timeout_seconds,omitempty"`
	FailOpen            bool `json:"fail_open,omitempty"`
}

// PipelineDefinitionResponse represents a pipeline definition response
type PipelineDefinitionResponse struct {
	ID             string             `json:"id"`
	Name           string             `json:"name"`
	Description    string             `json:"description,omitempty"`
	Enabled        bool               `json:"enabled"`
	Inherit        string             `json:"inherit,omitempty"`
	PreProcessing  []StageConfigInput `json:"pre_processing,omitempty"`
	PostProcessing []StageConfigInput `json:"post_processing,omitempty"`
	Settings       *PipelineSettings  `json:"settings,omitempty"`
	CreatedAt      string             `json:"created_at,omitempty"`
	UpdatedAt      string             `json:"updated_at,omitempty"`
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

// AuditLogRequest represents an audit log query request
type AuditLogRequest struct {
	RequestID string `json:"request_id,omitempty"`
	StartTime int64  `json:"start_time,omitempty"`
	EndTime   int64  `json:"end_time,omitempty"`
	UserID    string `json:"user_id,omitempty"`
	Decision  string `json:"decision,omitempty"`
	Limit     int    `json:"limit,omitempty"`
	Offset    int    `json:"offset,omitempty"`
}

// AuditLogResponse represents an audit log response
type AuditLogResponse struct {
	RequestID string       `json:"request_id"`
	Entries   []AuditEntry `json:"entries"`
}

// AuditEntry represents an audit entry
type AuditEntry struct {
	Timestamp   int64             `json:"timestamp"`
	Stage       string            `json:"stage"`
	Action      string            `json:"action"`
	Decision    string            `json:"decision"`
	Details     map[string]string `json:"details,omitempty"`
	UserID      string            `json:"user_id,omitempty"`
	RequestHash string            `json:"request_hash,omitempty"`
}

// AuditLogListResponse represents a list of audit log summaries
type AuditLogListResponse struct {
	Logs  []AuditLogSummary `json:"logs"`
	Total int               `json:"total"`
}

// AuditLogSummary represents an audit log summary
type AuditLogSummary struct {
	RequestID   string `json:"request_id"`
	Timestamp   int64  `json:"timestamp"`
	UserID      string `json:"user_id,omitempty"`
	Decision    string `json:"decision"`
	StageCount  int    `json:"stage_count"`
	RequestHash string `json:"request_hash,omitempty"`
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

	if h.clients.Leibniz == nil {
		h.writeError(w, http.StatusServiceUnavailable, "service_unavailable", "Leibniz service not available", "")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 120*time.Second)
	defer cancel()

	// Build gRPC request
	grpcReq := &leibnizpb.ProcessPipelineRequest{
		PipelineId: req.PipelineID,
		Prompt:     req.Prompt,
		Metadata:   req.Metadata,
		UserId:     req.UserID,
		SessionId:  req.SessionID,
	}

	if req.Options != nil {
		grpcReq.Options = &leibnizpb.PipelineOptions{
			SkipPreProcessing:  req.Options.SkipPreProcessing,
			SkipPostProcessing: req.Options.SkipPostProcessing,
			DryRun:             req.Options.DryRun,
			TimeoutSeconds:     int32(req.Options.TimeoutSeconds),
			Debug:              req.Options.Debug,
		}
	}

	grpcResp, err := h.clients.Leibniz.ProcessPipeline(ctx, grpcReq)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, "internal_error", "Pipeline processing failed", err.Error())
		return
	}

	// Convert response
	resp := ProcessPipelineResponse{
		RequestID:       grpcResp.RequestId,
		Success:         grpcResp.Success,
		Response:        grpcResp.Response,
		ProcessedPrompt: grpcResp.ProcessedPrompt,
		DurationMs:      grpcResp.DurationMs,
		Error:           grpcResp.Error,
	}

	if grpcResp.Flags != nil {
		resp.Flags = &PipelineFlags{
			Blocked:        grpcResp.Flags.Blocked,
			Modified:       grpcResp.Flags.Modified,
			Escalated:      grpcResp.Flags.Escalated,
			RequiresReview: grpcResp.Flags.RequiresReview,
			BlockReason:    grpcResp.Flags.BlockReason,
			ModifyReason:   grpcResp.Flags.ModifyReason,
		}
	}

	resp.StageResults = make([]StageResult, len(grpcResp.StageResults))
	for i, sr := range grpcResp.StageResults {
		resp.StageResults[i] = StageResult{
			StageName:  sr.StageName,
			AgentID:    sr.AgentId,
			Role:       sr.Role,
			Success:    sr.Success,
			Decision:   sr.Decision,
			Error:      sr.Error,
			DurationMs: sr.DurationMs,
			Input:      sr.Input,
			Output:     sr.Output,
			Skipped:    sr.Skipped,
			SkipReason: sr.SkipReason,
		}
	}

	h.writeJSON(w, http.StatusOK, resp)
}

// HandlePipelineProcessStream handles POST /api/v1/pipeline/process/stream (SSE)
func (h *Handler) HandlePipelineProcessStream(w http.ResponseWriter, r *http.Request) {
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

	if h.clients.Leibniz == nil {
		h.writeError(w, http.StatusServiceUnavailable, "service_unavailable", "Leibniz service not available", "")
		return
	}

	// Set SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	flusher, ok := w.(http.Flusher)
	if !ok {
		h.writeError(w, http.StatusInternalServerError, "internal_error", "Streaming not supported", "")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 120*time.Second)
	defer cancel()

	// Build gRPC request
	grpcReq := &leibnizpb.ProcessPipelineRequest{
		PipelineId: req.PipelineID,
		Prompt:     req.Prompt,
		Metadata:   req.Metadata,
		UserId:     req.UserID,
		SessionId:  req.SessionID,
	}

	if req.Options != nil {
		grpcReq.Options = &leibnizpb.PipelineOptions{
			SkipPreProcessing:  req.Options.SkipPreProcessing,
			SkipPostProcessing: req.Options.SkipPostProcessing,
			DryRun:             req.Options.DryRun,
			TimeoutSeconds:     int32(req.Options.TimeoutSeconds),
			Debug:              req.Options.Debug,
		}
	}

	stream, err := h.clients.Leibniz.StreamProcessPipeline(ctx, grpcReq)
	if err != nil {
		fmt.Fprintf(w, "event: error\ndata: %s\n\n", err.Error())
		flusher.Flush()
		return
	}

	for {
		result, err := stream.Recv()
		if err != nil {
			if err.Error() == "EOF" {
				fmt.Fprintf(w, "event: done\ndata: [DONE]\n\n")
			} else {
				fmt.Fprintf(w, "event: error\ndata: %s\n\n", err.Error())
			}
			flusher.Flush()
			break
		}

		// Send stage result
		data := StageResult{
			StageName:  result.StageName,
			AgentID:    result.AgentId,
			Role:       result.Role,
			Success:    result.Success,
			Decision:   result.Decision,
			Error:      result.Error,
			DurationMs: result.DurationMs,
			Input:      result.Input,
			Output:     result.Output,
			Skipped:    result.Skipped,
			SkipReason: result.SkipReason,
		}

		h.writeSSEEvent(w, "stage", data)
		flusher.Flush()
	}
}

// ============================================================================
// Pipeline Definition Handlers
// ============================================================================

// HandlePipelineDefinitions handles GET/POST /api/v1/pipeline/pipelines
func (h *Handler) HandlePipelineDefinitions(w http.ResponseWriter, r *http.Request) {
	if h.clients.Leibniz == nil {
		h.writeError(w, http.StatusServiceUnavailable, "service_unavailable", "Leibniz service not available", "")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	switch r.Method {
	case http.MethodGet:
		grpcResp, err := h.clients.Leibniz.ListPipelines(ctx, &common.Empty{})
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

		grpcReq := pipelineRequestToProto(&req)
		grpcResp, err := h.clients.Leibniz.CreatePipeline(ctx, grpcReq)
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
	if h.clients.Leibniz == nil {
		h.writeError(w, http.StatusServiceUnavailable, "service_unavailable", "Leibniz service not available", "")
		return
	}

	id = strings.TrimSuffix(id, "/")
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	switch r.Method {
	case http.MethodGet:
		grpcResp, err := h.clients.Leibniz.GetPipeline(ctx, &leibnizpb.GetPipelineRequest{Id: id})
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

		grpcReq := &leibnizpb.UpdatePipelineRequest{
			Id:             id,
			Name:           req.Name,
			Description:    req.Description,
			Enabled:        req.Enabled,
			PreProcessing:  stageConfigsToProto(req.PreProcessing),
			PostProcessing: stageConfigsToProto(req.PostProcessing),
		}
		if req.Settings != nil {
			grpcReq.Settings = &leibnizpb.PipelineSettings{
				MaxStages:           int32(req.Settings.MaxStages),
				StageTimeoutSeconds: int32(req.Settings.StageTimeoutSeconds),
				TotalTimeoutSeconds: int32(req.Settings.TotalTimeoutSeconds),
				FailOpen:            req.Settings.FailOpen,
			}
		}

		grpcResp, err := h.clients.Leibniz.UpdatePipeline(ctx, grpcReq)
		if err != nil {
			h.writeError(w, http.StatusInternalServerError, "internal_error", "Failed to update pipeline", err.Error())
			return
		}
		h.writeJSON(w, http.StatusOK, pipelineInfoToResponse(grpcResp))

	case http.MethodDelete:
		_, err := h.clients.Leibniz.DeletePipeline(ctx, &leibnizpb.DeletePipelineRequest{Id: id})
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
	if h.clients.Leibniz == nil {
		h.writeError(w, http.StatusServiceUnavailable, "service_unavailable", "Leibniz service not available", "")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	switch r.Method {
	case http.MethodGet:
		grpcResp, err := h.clients.Leibniz.ListPolicies(ctx, &common.Empty{})
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
		grpcResp, err := h.clients.Leibniz.CreatePolicy(ctx, grpcReq)
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
	if h.clients.Leibniz == nil {
		h.writeError(w, http.StatusServiceUnavailable, "service_unavailable", "Leibniz service not available", "")
		return
	}

	id = strings.TrimSuffix(id, "/")
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	switch r.Method {
	case http.MethodGet:
		grpcResp, err := h.clients.Leibniz.GetPolicy(ctx, &leibnizpb.GetPolicyRequest{Id: id})
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

		grpcReq := &leibnizpb.UpdatePolicyRequest{
			Id:          id,
			Name:        req.Name,
			Description: req.Description,
			PolicyType:  req.PolicyType,
			Enabled:     req.Enabled,
			Priority:    int32(req.Priority),
			Rules:       policyRulesToProto(req.Rules),
		}
		if req.LLMCheck != nil {
			grpcReq.LlmCheck = &leibnizpb.LLMCheckConfig{
				Enabled:        req.LLMCheck.Enabled,
				Model:          req.LLMCheck.Model,
				Prompt:         req.LLMCheck.Prompt,
				TimeoutSeconds: int32(req.LLMCheck.TimeoutSeconds),
				Temperature:    req.LLMCheck.Temperature,
			}
		}

		grpcResp, err := h.clients.Leibniz.UpdatePolicy(ctx, grpcReq)
		if err != nil {
			h.writeError(w, http.StatusInternalServerError, "internal_error", "Failed to update policy", err.Error())
			return
		}
		h.writeJSON(w, http.StatusOK, policyInfoToResponse(grpcResp))

	case http.MethodDelete:
		_, err := h.clients.Leibniz.DeletePolicy(ctx, &leibnizpb.DeletePolicyRequest{Id: id})
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

	if h.clients.Leibniz == nil {
		h.writeError(w, http.StatusServiceUnavailable, "service_unavailable", "Leibniz service not available", "")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	grpcReq := &leibnizpb.TestPolicyRequest{
		TestText: req.TestText,
	}

	// If policy is provided inline, include it
	if req.Policy.Name != "" {
		grpcReq.Policy = &leibnizpb.PolicyInfo{
			Id:          req.Policy.ID,
			Name:        req.Policy.Name,
			Description: req.Policy.Description,
			PolicyType:  req.Policy.PolicyType,
			Enabled:     req.Policy.Enabled,
			Priority:    int32(req.Policy.Priority),
			Rules:       policyRulesToProto(req.Policy.Rules),
		}
		if req.Policy.LLMCheck != nil {
			grpcReq.Policy.LlmCheck = &leibnizpb.LLMCheckConfig{
				Enabled:        req.Policy.LLMCheck.Enabled,
				Model:          req.Policy.LLMCheck.Model,
				Prompt:         req.Policy.LLMCheck.Prompt,
				TimeoutSeconds: int32(req.Policy.LLMCheck.TimeoutSeconds),
				Temperature:    req.Policy.LLMCheck.Temperature,
			}
		}
	}

	grpcResp, err := h.clients.Leibniz.TestPolicy(ctx, grpcReq)
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
			Action:      v.Action,
			Matched:     v.Matched,
		}
	}

	resp := TestPolicyResponse{
		Decision:     grpcResp.Decision,
		Violations:   violations,
		ModifiedText: grpcResp.ModifiedText,
		Reason:       grpcResp.Reason,
		DurationMs:   grpcResp.DurationMs,
	}

	h.writeJSON(w, http.StatusOK, resp)
}

// ============================================================================
// Audit Handlers
// ============================================================================

// HandleAuditLogs handles GET /api/v1/pipeline/audit
func (h *Handler) HandleAuditLogs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Use GET", "")
		return
	}

	if h.clients.Leibniz == nil {
		h.writeError(w, http.StatusServiceUnavailable, "service_unavailable", "Leibniz service not available", "")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	// Parse query parameters
	q := r.URL.Query()
	grpcReq := &leibnizpb.ListAuditLogsRequest{
		UserId:   q.Get("user_id"),
		Decision: q.Get("decision"),
	}

	grpcResp, err := h.clients.Leibniz.ListAuditLogs(ctx, grpcReq)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, "internal_error", "Failed to list audit logs", err.Error())
		return
	}

	logs := make([]AuditLogSummary, len(grpcResp.Logs))
	for i, l := range grpcResp.Logs {
		logs[i] = AuditLogSummary{
			RequestID:   l.RequestId,
			Timestamp:   l.Timestamp,
			UserID:      l.UserId,
			Decision:    l.Decision,
			StageCount:  int(l.StageCount),
			RequestHash: l.RequestHash,
		}
	}

	h.writeJSON(w, http.StatusOK, AuditLogListResponse{
		Logs:  logs,
		Total: int(grpcResp.Total),
	})
}

// HandleAuditLog handles GET /api/v1/pipeline/audit/{request_id}
func (h *Handler) HandleAuditLog(w http.ResponseWriter, r *http.Request, requestID string) {
	if r.Method != http.MethodGet {
		h.writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Use GET", "")
		return
	}

	if h.clients.Leibniz == nil {
		h.writeError(w, http.StatusServiceUnavailable, "service_unavailable", "Leibniz service not available", "")
		return
	}

	requestID = strings.TrimSuffix(requestID, "/")
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	grpcResp, err := h.clients.Leibniz.GetAuditLog(ctx, &leibnizpb.GetAuditLogRequest{
		RequestId: requestID,
	})
	if err != nil {
		h.writeError(w, http.StatusNotFound, "not_found", "Audit log not found", err.Error())
		return
	}

	entries := make([]AuditEntry, len(grpcResp.Entries))
	for i, e := range grpcResp.Entries {
		entries[i] = AuditEntry{
			Timestamp:   e.Timestamp,
			Stage:       e.Stage,
			Action:      e.Action,
			Decision:    e.Decision,
			Details:     e.Details,
			UserID:      e.UserId,
			RequestHash: e.RequestHash,
		}
	}

	h.writeJSON(w, http.StatusOK, AuditLogResponse{
		RequestID: grpcResp.RequestId,
		Entries:   entries,
	})
}

// ============================================================================
// Helper Functions
// ============================================================================

func (h *Handler) writeSSEEvent(w http.ResponseWriter, event string, data interface{}) {
	jsonData, _ := json.Marshal(data)
	fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event, jsonData)
}

func pipelineInfoToResponse(p *leibnizpb.PipelineInfo) PipelineDefinitionResponse {
	resp := PipelineDefinitionResponse{
		ID:          p.Id,
		Name:        p.Name,
		Description: p.Description,
		Enabled:     p.Enabled,
		Inherit:     p.Inherit,
	}

	if p.Settings != nil {
		resp.Settings = &PipelineSettings{
			MaxStages:           int(p.Settings.MaxStages),
			StageTimeoutSeconds: int(p.Settings.StageTimeoutSeconds),
			TotalTimeoutSeconds: int(p.Settings.TotalTimeoutSeconds),
			FailOpen:            p.Settings.FailOpen,
		}
	}

	resp.PreProcessing = stageConfigsFromProto(p.PreProcessing)
	resp.PostProcessing = stageConfigsFromProto(p.PostProcessing)

	if p.CreatedAt > 0 {
		resp.CreatedAt = time.Unix(p.CreatedAt, 0).Format(time.RFC3339)
	}
	if p.UpdatedAt > 0 {
		resp.UpdatedAt = time.Unix(p.UpdatedAt, 0).Format(time.RFC3339)
	}

	return resp
}

func pipelineRequestToProto(req *PipelineDefinitionRequest) *leibnizpb.CreatePipelineRequest {
	grpcReq := &leibnizpb.CreatePipelineRequest{
		Id:             req.ID,
		Name:           req.Name,
		Description:    req.Description,
		Enabled:        req.Enabled,
		Inherit:        req.Inherit,
		PreProcessing:  stageConfigsToProto(req.PreProcessing),
		PostProcessing: stageConfigsToProto(req.PostProcessing),
	}

	if req.Settings != nil {
		grpcReq.Settings = &leibnizpb.PipelineSettings{
			MaxStages:           int32(req.Settings.MaxStages),
			StageTimeoutSeconds: int32(req.Settings.StageTimeoutSeconds),
			TotalTimeoutSeconds: int32(req.Settings.TotalTimeoutSeconds),
			FailOpen:            req.Settings.FailOpen,
		}
	}

	return grpcReq
}

func stageConfigsToProto(stages []StageConfigInput) []*leibnizpb.StageConfig {
	result := make([]*leibnizpb.StageConfig, len(stages))
	for i, s := range stages {
		result[i] = &leibnizpb.StageConfig{
			Name:           s.Name,
			AgentId:        s.AgentID,
			Role:           s.Role,
			Required:       s.Required,
			OnFail:         s.OnFail,
			Condition:      s.Condition,
			Priority:       int32(s.Priority),
			TimeoutSeconds: int32(s.TimeoutSeconds),
			RetryCount:     int32(s.RetryCount),
			Input:          s.Input,
			OutputMapping:  s.OutputMapping,
		}
	}
	return result
}

func stageConfigsFromProto(stages []*leibnizpb.StageConfig) []StageConfigInput {
	result := make([]StageConfigInput, len(stages))
	for i, s := range stages {
		result[i] = StageConfigInput{
			Name:           s.Name,
			AgentID:        s.AgentId,
			Role:           s.Role,
			Required:       s.Required,
			OnFail:         s.OnFail,
			Condition:      s.Condition,
			Priority:       int(s.Priority),
			TimeoutSeconds: int(s.TimeoutSeconds),
			RetryCount:     int(s.RetryCount),
			Input:          s.Input,
			OutputMapping:  s.OutputMapping,
		}
	}
	return result
}

func policyInfoToResponse(p *leibnizpb.PolicyInfo) PolicyDefinitionResponse {
	resp := PolicyDefinitionResponse{
		ID:          p.Id,
		Name:        p.Name,
		Description: p.Description,
		PolicyType:  p.PolicyType,
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

func policyRequestToProto(req *PolicyDefinitionRequest) *leibnizpb.CreatePolicyRequest {
	grpcReq := &leibnizpb.CreatePolicyRequest{
		Id:          req.ID,
		Name:        req.Name,
		Description: req.Description,
		PolicyType:  req.PolicyType,
		Enabled:     req.Enabled,
		Priority:    int32(req.Priority),
		Rules:       policyRulesToProto(req.Rules),
	}

	if req.LLMCheck != nil {
		grpcReq.LlmCheck = &leibnizpb.LLMCheckConfig{
			Enabled:        req.LLMCheck.Enabled,
			Model:          req.LLMCheck.Model,
			Prompt:         req.LLMCheck.Prompt,
			TimeoutSeconds: int32(req.LLMCheck.TimeoutSeconds),
			Temperature:    req.LLMCheck.Temperature,
		}
	}

	return grpcReq
}

func policyRulesToProto(rules []PolicyRuleInput) []*leibnizpb.PolicyRule {
	result := make([]*leibnizpb.PolicyRule, len(rules))
	for i, r := range rules {
		result[i] = &leibnizpb.PolicyRule{
			Id:            r.ID,
			Pattern:       r.Pattern,
			Action:        r.Action,
			Message:       r.Message,
			Replacement:   r.Replacement,
			CaseSensitive: r.CaseSensitive,
		}
	}
	return result
}

func policyRulesFromProto(rules []*leibnizpb.PolicyRule) []PolicyRuleInput {
	result := make([]PolicyRuleInput, len(rules))
	for i, r := range rules {
		result[i] = PolicyRuleInput{
			ID:            r.Id,
			Pattern:       r.Pattern,
			Action:        r.Action,
			Message:       r.Message,
			Replacement:   r.Replacement,
			CaseSensitive: r.CaseSensitive,
		}
	}
	return result
}
