// ============================================================================
// meinDENKWERK (mDW) - Lokale KI-Plattform
// ============================================================================
//
// Package:     handler
// Description: REST API handlers for Platon pipeline processing
// Author:      Mike Stoffels with Claude
// Created:     2025-12-08
// License:     MIT
// ============================================================================

package handler

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/msto63/mDW/api/gen/common"
	platonpb "github.com/msto63/mDW/api/gen/platon"
)

// ============================================================================
// Platon REST Types
// ============================================================================

// PlatonProcessRequest represents a Platon processing request
type PlatonProcessRequest struct {
	RequestID  string            `json:"request_id,omitempty"`
	PipelineID string            `json:"pipeline_id,omitempty"`
	Prompt     string            `json:"prompt,omitempty"`
	Response   string            `json:"response,omitempty"`
	Metadata   map[string]string `json:"metadata,omitempty"`
}

// PlatonProcessResponse represents a Platon processing response
type PlatonProcessResponse struct {
	RequestID         string              `json:"request_id"`
	Success           bool                `json:"success"`
	ProcessedPrompt   string              `json:"processed_prompt,omitempty"`
	ProcessedResponse string              `json:"processed_response,omitempty"`
	Blocked           bool                `json:"blocked"`
	BlockReason       string              `json:"block_reason,omitempty"`
	Modified          bool                `json:"modified"`
	AuditLog          []PlatonAuditEntry  `json:"audit_log,omitempty"`
	Metadata          map[string]string   `json:"metadata,omitempty"`
	DurationMs        int64               `json:"duration_ms"`
	Error             string              `json:"error,omitempty"`
}

// PlatonAuditEntry represents a Platon audit log entry
type PlatonAuditEntry struct {
	Handler    string            `json:"handler"`
	Phase      string            `json:"phase"`
	DurationMs int64             `json:"duration_ms"`
	Modified   bool              `json:"modified"`
	Error      string            `json:"error,omitempty"`
	Details    map[string]string `json:"details,omitempty"`
}

// PlatonHandlerInfo represents handler information
type PlatonHandlerInfo struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	Priority    int    `json:"priority"`
	Enabled     bool   `json:"enabled"`
	Description string `json:"description,omitempty"`
}

// PlatonPipelineInfo represents pipeline information
type PlatonPipelineInfo struct {
	ID           string            `json:"id"`
	Name         string            `json:"name"`
	Description  string            `json:"description,omitempty"`
	Enabled      bool              `json:"enabled"`
	PreHandlers  []string          `json:"pre_handlers,omitempty"`
	PostHandlers []string          `json:"post_handlers,omitempty"`
	Config       map[string]string `json:"config,omitempty"`
	CreatedAt    int64             `json:"created_at,omitempty"`
	UpdatedAt    int64             `json:"updated_at,omitempty"`
}

// ============================================================================
// Platon Processing Handlers
// ============================================================================

// HandlePlatonProcessPre handles POST /api/v1/platon/process/pre
func (h *Handler) HandlePlatonProcessPre(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Use POST", "")
		return
	}

	var req PlatonProcessRequest
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
		RequestId:  req.RequestID,
		PipelineId: req.PipelineID,
		Prompt:     req.Prompt,
		Response:   req.Response,
		Metadata:   req.Metadata,
	}

	grpcResp, err := h.clients.Platon.ProcessPre(ctx, grpcReq)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, "internal_error", "Pre-processing failed", err.Error())
		return
	}

	resp := convertPlatonResponse(grpcResp)
	h.writeJSON(w, http.StatusOK, resp)
}

// HandlePlatonProcessPost handles POST /api/v1/platon/process/post
func (h *Handler) HandlePlatonProcessPost(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Use POST", "")
		return
	}

	var req PlatonProcessRequest
	if err := h.readJSON(r, &req); err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON", err.Error())
		return
	}

	if req.Response == "" {
		h.writeError(w, http.StatusBadRequest, "invalid_request", "Response required", "")
		return
	}

	if h.clients.Platon == nil {
		h.writeError(w, http.StatusServiceUnavailable, "service_unavailable", "Platon service not available", "")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 60*time.Second)
	defer cancel()

	grpcReq := &platonpb.ProcessRequest{
		RequestId:  req.RequestID,
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

	resp := convertPlatonResponse(grpcResp)
	h.writeJSON(w, http.StatusOK, resp)
}

// HandlePlatonProcess handles POST /api/v1/platon/process
func (h *Handler) HandlePlatonProcess(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Use POST", "")
		return
	}

	var req PlatonProcessRequest
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
		RequestId:  req.RequestID,
		PipelineId: req.PipelineID,
		Prompt:     req.Prompt,
		Response:   req.Response,
		Metadata:   req.Metadata,
	}

	grpcResp, err := h.clients.Platon.Process(ctx, grpcReq)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, "internal_error", "Processing failed", err.Error())
		return
	}

	resp := convertPlatonResponse(grpcResp)
	h.writeJSON(w, http.StatusOK, resp)
}

// ============================================================================
// Platon Handler Management
// ============================================================================

// HandlePlatonListHandlers handles GET /api/v1/platon/handlers
func (h *Handler) HandlePlatonListHandlers(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Use GET", "")
		return
	}

	if h.clients.Platon == nil {
		h.writeError(w, http.StatusServiceUnavailable, "service_unavailable", "Platon service not available", "")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	grpcResp, err := h.clients.Platon.ListHandlers(ctx, &common.Empty{})
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, "internal_error", "Failed to list handlers", err.Error())
		return
	}

	handlers := make([]PlatonHandlerInfo, len(grpcResp.Handlers))
	for i, hi := range grpcResp.Handlers {
		handlers[i] = PlatonHandlerInfo{
			Name:        hi.Name,
			Type:        handlerTypeToString(hi.Type),
			Priority:    int(hi.Priority),
			Enabled:     hi.Enabled,
			Description: hi.Description,
		}
	}

	h.writeJSON(w, http.StatusOK, map[string]interface{}{
		"handlers": handlers,
		"total":    len(handlers),
	})
}

// HandlePlatonGetHandler handles GET /api/v1/platon/handlers/{name}
func (h *Handler) HandlePlatonGetHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Use GET", "")
		return
	}

	// Extract handler name from path
	name := strings.TrimPrefix(r.URL.Path, "/api/v1/platon/handlers/")
	if name == "" {
		h.writeError(w, http.StatusBadRequest, "invalid_request", "Handler name required", "")
		return
	}

	if h.clients.Platon == nil {
		h.writeError(w, http.StatusServiceUnavailable, "service_unavailable", "Platon service not available", "")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	// GetHandler returns HandlerInfo directly
	hi, err := h.clients.Platon.GetHandler(ctx, &platonpb.GetHandlerRequest{Name: name})
	if err != nil {
		h.writeError(w, http.StatusNotFound, "not_found", "Handler not found", err.Error())
		return
	}

	handler := PlatonHandlerInfo{
		Name:        hi.Name,
		Type:        handlerTypeToString(hi.Type),
		Priority:    int(hi.Priority),
		Enabled:     hi.Enabled,
		Description: hi.Description,
	}

	h.writeJSON(w, http.StatusOK, handler)
}

// ============================================================================
// Platon Pipeline Management
// ============================================================================

// HandlePlatonListPipelines handles GET /api/v1/platon/pipelines
func (h *Handler) HandlePlatonListPipelines(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Use GET", "")
		return
	}

	if h.clients.Platon == nil {
		h.writeError(w, http.StatusServiceUnavailable, "service_unavailable", "Platon service not available", "")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	grpcResp, err := h.clients.Platon.ListPipelines(ctx, &common.Empty{})
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, "internal_error", "Failed to list pipelines", err.Error())
		return
	}

	pipelines := make([]PlatonPipelineInfo, len(grpcResp.Pipelines))
	for i, p := range grpcResp.Pipelines {
		pipelines[i] = PlatonPipelineInfo{
			ID:           p.Id,
			Name:         p.Name,
			Description:  p.Description,
			Enabled:      p.Enabled,
			PreHandlers:  p.PreHandlers,
			PostHandlers: p.PostHandlers,
			Config:       p.Config,
			CreatedAt:    p.CreatedAt,
			UpdatedAt:    p.UpdatedAt,
		}
	}

	h.writeJSON(w, http.StatusOK, map[string]interface{}{
		"pipelines": pipelines,
		"total":     len(pipelines),
	})
}

// HandlePlatonGetPipeline handles GET /api/v1/platon/pipelines/{id}
func (h *Handler) HandlePlatonGetPipeline(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Use GET", "")
		return
	}

	// Extract pipeline ID from path
	id := strings.TrimPrefix(r.URL.Path, "/api/v1/platon/pipelines/")
	if id == "" {
		h.writeError(w, http.StatusBadRequest, "invalid_request", "Pipeline ID required", "")
		return
	}

	if h.clients.Platon == nil {
		h.writeError(w, http.StatusServiceUnavailable, "service_unavailable", "Platon service not available", "")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	// GetPipeline returns PipelineInfo directly
	p, err := h.clients.Platon.GetPipeline(ctx, &platonpb.GetPipelineRequest{Id: id})
	if err != nil {
		h.writeError(w, http.StatusNotFound, "not_found", "Pipeline not found", err.Error())
		return
	}

	pipeline := PlatonPipelineInfo{
		ID:           p.Id,
		Name:         p.Name,
		Description:  p.Description,
		Enabled:      p.Enabled,
		PreHandlers:  p.PreHandlers,
		PostHandlers: p.PostHandlers,
		Config:       p.Config,
		CreatedAt:    p.CreatedAt,
		UpdatedAt:    p.UpdatedAt,
	}

	h.writeJSON(w, http.StatusOK, pipeline)
}

// HandlePlatonCreatePipeline handles POST /api/v1/platon/pipelines
func (h *Handler) HandlePlatonCreatePipeline(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Use POST", "")
		return
	}

	var req PlatonPipelineInfo
	if err := h.readJSON(r, &req); err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON", err.Error())
		return
	}

	if req.ID == "" || req.Name == "" {
		h.writeError(w, http.StatusBadRequest, "invalid_request", "ID and Name required", "")
		return
	}

	if h.clients.Platon == nil {
		h.writeError(w, http.StatusServiceUnavailable, "service_unavailable", "Platon service not available", "")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	grpcReq := &platonpb.CreatePipelineRequest{
		Id:           req.ID,
		Name:         req.Name,
		Description:  req.Description,
		Enabled:      req.Enabled,
		PreHandlers:  req.PreHandlers,
		PostHandlers: req.PostHandlers,
		Config:       req.Config,
	}

	// CreatePipeline returns PipelineInfo directly
	p, err := h.clients.Platon.CreatePipeline(ctx, grpcReq)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, "internal_error", "Failed to create pipeline", err.Error())
		return
	}

	pipeline := PlatonPipelineInfo{
		ID:           p.Id,
		Name:         p.Name,
		Description:  p.Description,
		Enabled:      p.Enabled,
		PreHandlers:  p.PreHandlers,
		PostHandlers: p.PostHandlers,
		Config:       p.Config,
		CreatedAt:    p.CreatedAt,
		UpdatedAt:    p.UpdatedAt,
	}

	h.writeJSON(w, http.StatusCreated, pipeline)
}

// HandlePlatonDeletePipeline handles DELETE /api/v1/platon/pipelines/{id}
func (h *Handler) HandlePlatonDeletePipeline(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		h.writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Use DELETE", "")
		return
	}

	// Extract pipeline ID from path
	id := strings.TrimPrefix(r.URL.Path, "/api/v1/platon/pipelines/")
	if id == "" {
		h.writeError(w, http.StatusBadRequest, "invalid_request", "Pipeline ID required", "")
		return
	}

	if h.clients.Platon == nil {
		h.writeError(w, http.StatusServiceUnavailable, "service_unavailable", "Platon service not available", "")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	_, err := h.clients.Platon.DeletePipeline(ctx, &platonpb.DeletePipelineRequest{Id: id})
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, "internal_error", "Failed to delete pipeline", err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// HandlePlatonStats handles GET /api/v1/platon/stats
func (h *Handler) HandlePlatonStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Use GET", "")
		return
	}

	if h.clients.Platon == nil {
		h.writeError(w, http.StatusServiceUnavailable, "service_unavailable", "Platon service not available", "")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	// Get handler count from ListHandlers
	handlersResp, err := h.clients.Platon.ListHandlers(ctx, &common.Empty{})
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, "internal_error", "Failed to get stats", err.Error())
		return
	}

	// Get pipeline count from ListPipelines
	pipelinesResp, err := h.clients.Platon.ListPipelines(ctx, &common.Empty{})
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, "internal_error", "Failed to get stats", err.Error())
		return
	}

	// Count handler types
	preHandlers := 0
	postHandlers := 0
	bothHandlers := 0
	for _, hi := range handlersResp.Handlers {
		switch hi.Type {
		case platonpb.HandlerType_HANDLER_TYPE_PRE:
			preHandlers++
		case platonpb.HandlerType_HANDLER_TYPE_POST:
			postHandlers++
		case platonpb.HandlerType_HANDLER_TYPE_BOTH:
			bothHandlers++
		}
	}

	stats := map[string]interface{}{
		"total_handlers":  len(handlersResp.Handlers),
		"pre_handlers":    preHandlers,
		"post_handlers":   postHandlers,
		"both_handlers":   bothHandlers,
		"pipeline_count":  len(pipelinesResp.Pipelines),
	}

	h.writeJSON(w, http.StatusOK, stats)
}

// ============================================================================
// Helper Functions
// ============================================================================

// convertPlatonResponse converts a gRPC ProcessResponse to REST response
func convertPlatonResponse(grpcResp *platonpb.ProcessResponse) PlatonProcessResponse {
	resp := PlatonProcessResponse{
		RequestID:         grpcResp.RequestId,
		Success:           !grpcResp.Blocked, // Derive success from not being blocked
		ProcessedPrompt:   grpcResp.ProcessedPrompt,
		ProcessedResponse: grpcResp.ProcessedResponse,
		Blocked:           grpcResp.Blocked,
		BlockReason:       grpcResp.BlockReason,
		Modified:          grpcResp.Modified,
		Metadata:          grpcResp.Metadata,
		DurationMs:        grpcResp.DurationMs,
	}

	if len(grpcResp.AuditLog) > 0 {
		resp.AuditLog = make([]PlatonAuditEntry, len(grpcResp.AuditLog))
		for i, entry := range grpcResp.AuditLog {
			resp.AuditLog[i] = PlatonAuditEntry{
				Handler:    entry.Handler,
				Phase:      entry.Phase,
				DurationMs: entry.DurationMs,
				Modified:   entry.Modified,
				Error:      entry.Error,
				Details:    entry.Details,
			}
		}
	}

	return resp
}

// handlerTypeToString converts HandlerType enum to string
func handlerTypeToString(t platonpb.HandlerType) string {
	switch t {
	case platonpb.HandlerType_HANDLER_TYPE_PRE:
		return "pre"
	case platonpb.HandlerType_HANDLER_TYPE_POST:
		return "post"
	case platonpb.HandlerType_HANDLER_TYPE_BOTH:
		return "both"
	default:
		return "unknown"
	}
}
