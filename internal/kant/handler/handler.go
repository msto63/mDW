package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	babbagepb "github.com/msto63/mDW/api/gen/babbage"
	"github.com/msto63/mDW/api/gen/common"
	hypatiapb "github.com/msto63/mDW/api/gen/hypatia"
	leibnizpb "github.com/msto63/mDW/api/gen/leibniz"
	turingpb "github.com/msto63/mDW/api/gen/turing"
	"github.com/msto63/mDW/internal/kant/client"
	"github.com/msto63/mDW/pkg/core/logging"
)

// Note: Russell import is used via clients.Russell which is already typed

// ChatRequest represents a chat completion request
type ChatRequest struct {
	Messages    []Message         `json:"messages"`
	Model       string            `json:"model,omitempty"`
	MaxTokens   int               `json:"max_tokens,omitempty"`
	Temperature float64           `json:"temperature,omitempty"`
	Stream      bool              `json:"stream,omitempty"`
	Context     map[string]string `json:"context,omitempty"`
}

// Message represents a chat message
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ChatResponse represents a chat completion response
type ChatResponse struct {
	ID      string  `json:"id"`
	Model   string  `json:"model"`
	Created int64   `json:"created"`
	Message Message `json:"message"`
	Usage   Usage   `json:"usage,omitempty"`
}

// Usage represents token usage
type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// SearchRequest represents a RAG search request
type SearchRequest struct {
	Query      string  `json:"query"`
	Collection string  `json:"collection,omitempty"`
	TopK       int     `json:"top_k,omitempty"`
	MinScore   float64 `json:"min_score,omitempty"`
}

// SearchResult represents a search result
type SearchResult struct {
	ID       string            `json:"id"`
	Content  string            `json:"content"`
	Score    float64           `json:"score"`
	Metadata map[string]string `json:"metadata,omitempty"`
}

// SearchResponse represents a RAG search response
type SearchResponse struct {
	Query   string         `json:"query"`
	Results []SearchResult `json:"results"`
	Total   int            `json:"total"`
}

// IngestRequest represents a document ingest request
type IngestRequest struct {
	Content    string            `json:"content"`
	Title      string            `json:"title,omitempty"`
	Source     string            `json:"source,omitempty"`
	Collection string            `json:"collection,omitempty"`
	Metadata   map[string]string `json:"metadata,omitempty"`
}

// IngestResponse represents a document ingest response
type IngestResponse struct {
	DocumentID string `json:"document_id"`
	Success    bool   `json:"success"`
}

// CollectionRequest represents a collection creation request
type CollectionRequest struct {
	Name        string            `json:"name"`
	Description string            `json:"description,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

// CollectionResponse represents a collection response
type CollectionResponse struct {
	Name        string            `json:"name"`
	Description string            `json:"description,omitempty"`
	Count       int64             `json:"count"`
	Metadata    map[string]string `json:"metadata,omitempty"`
	CreatedAt   string            `json:"created_at,omitempty"`
}

// CollectionsResponse represents a list of collections
type CollectionsResponse struct {
	Collections []CollectionResponse `json:"collections"`
	Total       int                  `json:"total"`
}

// CollectionStatsResponse represents collection statistics
type CollectionStatsResponse struct {
	Name          string  `json:"name"`
	DocumentCount int64   `json:"document_count"`
	ChunkCount    int64   `json:"chunk_count"`
	VectorCount   int64   `json:"vector_count"`
	StorageSize   int64   `json:"storage_size"`
	AvgChunkSize  float64 `json:"avg_chunk_size"`
}

// DocumentResponse represents a document
type DocumentResponse struct {
	ID         string            `json:"id"`
	Title      string            `json:"title,omitempty"`
	Content    string            `json:"content,omitempty"`
	Source     string            `json:"source,omitempty"`
	Collection string            `json:"collection,omitempty"`
	Metadata   map[string]string `json:"metadata,omitempty"`
	ChunkCount int               `json:"chunk_count,omitempty"`
	CreatedAt  string            `json:"created_at,omitempty"`
}

// DocumentsResponse represents a list of documents
type DocumentsResponse struct {
	Documents []DocumentResponse `json:"documents"`
	Total     int                `json:"total"`
}

// HybridSearchRequest represents a hybrid search request
type HybridSearchRequest struct {
	Query       string  `json:"query"`
	Collection  string  `json:"collection,omitempty"`
	TopK        int     `json:"top_k,omitempty"`
	MinScore    float64 `json:"min_score,omitempty"`
	AlphaVector float64 `json:"alpha_vector,omitempty"` // Weight for vector search (0-1)
}

// RAGAugmentRequest represents a RAG augmentation request
type RAGAugmentRequest struct {
	Query      string `json:"query"`
	Collection string `json:"collection,omitempty"`
	TopK       int    `json:"top_k,omitempty"`
	Model      string `json:"model,omitempty"`
}

// RAGAugmentResponse represents a RAG augmentation response
type RAGAugmentResponse struct {
	Query         string         `json:"query"`
	Answer        string         `json:"answer"`
	Sources       []SearchResult `json:"sources"`
	Model         string         `json:"model"`
	PromptTokens  int            `json:"prompt_tokens,omitempty"`
	OutputTokens  int            `json:"output_tokens,omitempty"`
}

// EmbedRequest represents an embedding request
type EmbedRequest struct {
	Text  string `json:"text,omitempty"`
	Texts []string `json:"texts,omitempty"`
	Model string `json:"model,omitempty"`
}

// EmbedResponse represents an embedding response
type EmbedResponse struct {
	Embeddings [][]float64 `json:"embeddings"`
	Model      string      `json:"model"`
	Dimensions int         `json:"dimensions"`
}

// ModelPullRequest represents a model pull request
type ModelPullRequest struct {
	Model string `json:"model"`
}

// ModelPullResponse represents a model pull response
type ModelPullResponse struct {
	Model   string `json:"model"`
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
}

// ConversationRequest represents a conversation creation request
type ConversationRequest struct {
	Title    string            `json:"title,omitempty"`
	Model    string            `json:"model,omitempty"`
	Metadata map[string]string `json:"metadata,omitempty"`
}

// ConversationResponse represents a conversation
type ConversationResponse struct {
	ID           string    `json:"id"`
	Title        string    `json:"title,omitempty"`
	Model        string    `json:"model,omitempty"`
	MessageCount int       `json:"message_count"`
	CreatedAt    string    `json:"created_at,omitempty"`
	UpdatedAt    string    `json:"updated_at,omitempty"`
	Messages     []Message `json:"messages,omitempty"`
}

// ConversationsResponse represents a list of conversations
type ConversationsResponse struct {
	Conversations []ConversationResponse `json:"conversations"`
	Total         int                    `json:"total"`
}

// AdminOverviewResponse represents system overview
type AdminOverviewResponse struct {
	Timestamp         string                    `json:"timestamp"`
	TotalServices     int                       `json:"total_services"`
	HealthyServices   int                       `json:"healthy_services"`
	DegradedServices  int                       `json:"degraded_services"`
	UnhealthyServices int                       `json:"unhealthy_services"`
	Services          map[string]ServiceStatus  `json:"services"`
	Metrics           *SystemMetricsResponse    `json:"metrics"`
	RecentErrors      []ErrorEntryResponse      `json:"recent_errors,omitempty"`
}

// ServiceStatus represents service status
type ServiceStatus struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	Status   string `json:"status"`
	Address  string `json:"address,omitempty"`
	Version  string `json:"version,omitempty"`
	LastSeen string `json:"last_seen,omitempty"`
}

// SystemMetricsResponse represents system metrics
type SystemMetricsResponse struct {
	TotalRequests       int64   `json:"total_requests"`
	SuccessfulRequests  int64   `json:"successful_requests"`
	FailedRequests      int64   `json:"failed_requests"`
	AverageResponseTime string  `json:"average_response_time"`
	RequestsPerSecond   float64 `json:"requests_per_second"`
}

// ErrorEntryResponse represents an error entry
type ErrorEntryResponse struct {
	Timestamp string `json:"timestamp"`
	Service   string `json:"service"`
	Operation string `json:"operation"`
	ErrorCode string `json:"error_code,omitempty"`
	Message   string `json:"message"`
	RequestID string `json:"request_id,omitempty"`
}

// PipelineRequest represents a pipeline creation request
type PipelineRequest struct {
	ID          string              `json:"id,omitempty"`
	Name        string              `json:"name"`
	Description string              `json:"description,omitempty"`
	Steps       []PipelineStepInput `json:"steps"`
}

// PipelineStepInput represents a pipeline step input
type PipelineStepInput struct {
	ID          string                 `json:"id"`
	ServiceType string                 `json:"service_type"`
	Operation   string                 `json:"operation"`
	Parameters  map[string]interface{} `json:"parameters,omitempty"`
	DependsOn   []string               `json:"depends_on,omitempty"`
}

// PipelineResponse represents a pipeline
type PipelineResponse struct {
	ID          string              `json:"id"`
	Name        string              `json:"name"`
	Description string              `json:"description,omitempty"`
	Steps       []PipelineStepInput `json:"steps"`
	CreatedAt   string              `json:"created_at,omitempty"`
}

// PipelinesResponse represents a list of pipelines
type PipelinesResponse struct {
	Pipelines []PipelineResponse `json:"pipelines"`
	Total     int                `json:"total"`
}

// PipelineExecuteRequest represents pipeline execution request
type PipelineExecuteRequest struct {
	Input interface{} `json:"input"`
}

// PipelineExecutionResponse represents pipeline execution result
type PipelineExecutionResponse struct {
	ID          string                 `json:"id"`
	PipelineID  string                 `json:"pipeline_id"`
	Status      string                 `json:"status"`
	StartedAt   string                 `json:"started_at"`
	CompletedAt string                 `json:"completed_at,omitempty"`
	StepResults map[string]interface{} `json:"step_results,omitempty"`
	Error       string                 `json:"error,omitempty"`
}

// AnalyzeRequest represents an NLP analysis request
type AnalyzeRequest struct {
	Text string `json:"text"`
}

// AnalyzeResponse represents an NLP analysis response
type AnalyzeResponse struct {
	Language  string           `json:"language,omitempty"`
	Sentiment *SentimentResult `json:"sentiment,omitempty"`
	Entities  []Entity         `json:"entities,omitempty"`
	Keywords  []Keyword        `json:"keywords,omitempty"`
}

// SentimentResult represents sentiment analysis result
type SentimentResult struct {
	Label      string  `json:"label"`
	Confidence float64 `json:"confidence"`
}

// Entity represents a named entity
type Entity struct {
	Text  string `json:"text"`
	Type  string `json:"type"`
	Start int    `json:"start"`
	End   int    `json:"end"`
}

// Keyword represents a keyword
type Keyword struct {
	Word  string  `json:"word"`
	Score float64 `json:"score"`
}

// SummarizeRequest represents a summarization request
type SummarizeRequest struct {
	Text      string `json:"text"`
	MaxLength int    `json:"max_length,omitempty"`
	Style     string `json:"style,omitempty"` // brief, detailed, bullet
}

// SummarizeResponse represents a summarization response
type SummarizeResponse struct {
	Summary        string `json:"summary"`
	OriginalLength int    `json:"original_length"`
	SummaryLength  int    `json:"summary_length"`
}

// AgentRequest represents an agent execution request
type AgentRequest struct {
	Task     string   `json:"task"`
	Message  string   `json:"message"`  // Alias for Task
	AgentID  string   `json:"agent_id"` // Optional agent ID
	Tools    []string `json:"tools,omitempty"`
	MaxSteps int      `json:"max_steps,omitempty"`
}

// AgentResponse represents an agent execution response
type AgentResponse struct {
	ID        string      `json:"id"`
	Status    string      `json:"status"`
	Result    string      `json:"result"`
	Response  string      `json:"response"` // Alias for Result
	Steps     []AgentStep `json:"steps,omitempty"`
	ToolsUsed []string    `json:"tools_used,omitempty"`
}

// AgentStep represents a single step in agent execution
type AgentStep struct {
	Step      int    `json:"step"`
	Action    string `json:"action"`
	Tool      string `json:"tool,omitempty"`
	Input     string `json:"input,omitempty"`
	Output    string `json:"output,omitempty"`
	Reasoning string `json:"reasoning,omitempty"`
}

// ModelInfo represents LLM model information
type ModelInfo struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Provider    string `json:"provider"`
	Size        int64  `json:"size,omitempty"`
	Description string `json:"description,omitempty"`
}

// ModelsResponse represents a list of models
type ModelsResponse struct {
	Models []ModelInfo `json:"models"`
}

// ErrorResponse represents an API error
type ErrorResponse struct {
	Error   string `json:"error"`
	Code    string `json:"code,omitempty"`
	Details string `json:"details,omitempty"`
}

// HealthResponse represents health check response
type HealthResponse struct {
	Status   string            `json:"status"`
	Version  string            `json:"version"`
	Uptime   string            `json:"uptime"`
	Services map[string]string `json:"services,omitempty"`
}

// ServiceInfo represents information about a registered service
type ServiceInfo struct {
	ID       string            `json:"id"`
	Name     string            `json:"name"`
	Address  string            `json:"address"`
	Port     int               `json:"port"`
	Status   string            `json:"status"`
	Metadata map[string]string `json:"metadata,omitempty"`
}

// ServicesResponse represents a list of registered services
type ServicesResponse struct {
	Services []ServiceInfo `json:"services"`
	Total    int           `json:"total"`
}

// Handler handles HTTP requests for the API Gateway
type Handler struct {
	clients   *client.ServiceClients
	logger    *logging.Logger
	startTime time.Time
	version   string
}

// NewHandler creates a new API handler
func NewHandler(version string, clients *client.ServiceClients) *Handler {
	return &Handler{
		clients:   clients,
		logger:    logging.New("kant-handler"),
		startTime: time.Now(),
		version:   version,
	}
}

// ServeHTTP implements http.Handler
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Add CORS headers
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	// Route requests
	path := strings.TrimPrefix(r.URL.Path, "/api/v1")
	path = strings.TrimPrefix(path, "/")

	switch {
	case path == "" || path == "/":
		h.handleRoot(w, r)
	case path == "health" || path == "health/":
		h.handleHealth(w, r)
	case path == "services" || path == "services/":
		h.handleServices(w, r)
	case path == "models" || path == "models/":
		h.handleModels(w, r)
	case strings.HasPrefix(path, "models/pull"):
		h.handleModelPull(w, r)
	case strings.HasPrefix(path, "models/"):
		h.handleModelDelete(w, r, strings.TrimPrefix(path, "models/"))
	case path == "chat" || path == "chat/":
		h.handleChat(w, r)
	case path == "chat/stream" || path == "chat/stream/":
		h.handleChatStream(w, r)
	case path == "embed" || path == "embed/":
		h.handleEmbed(w, r)
	case path == "conversations" || path == "conversations/":
		h.handleConversations(w, r)
	case strings.HasPrefix(path, "conversations/"):
		h.handleConversation(w, r, strings.TrimPrefix(path, "conversations/"))
	case path == "search" || path == "search/":
		h.handleSearch(w, r)
	case path == "search/hybrid" || path == "search/hybrid/":
		h.handleHybridSearch(w, r)
	case path == "ingest" || path == "ingest/":
		h.handleIngest(w, r)
	case path == "collections" || path == "collections/":
		h.handleCollections(w, r)
	case strings.HasPrefix(path, "collections/") && strings.HasSuffix(path, "/stats"):
		name := strings.TrimSuffix(strings.TrimPrefix(path, "collections/"), "/stats")
		h.handleCollectionStats(w, r, name)
	case strings.HasPrefix(path, "collections/"):
		h.handleCollection(w, r, strings.TrimPrefix(path, "collections/"))
	case path == "documents" || path == "documents/":
		h.handleDocuments(w, r)
	case strings.HasPrefix(path, "documents/"):
		h.handleDocument(w, r, strings.TrimPrefix(path, "documents/"))
	case path == "rag/augment" || path == "rag/augment/":
		h.handleRAGAugment(w, r)
	case path == "analyze" || path == "analyze/":
		h.handleAnalyze(w, r)
	case path == "summarize" || path == "summarize/":
		h.handleSummarize(w, r)
	case path == "agent" || path == "agent/" || path == "agent/execute" || path == "agent/execute/":
		h.handleAgent(w, r)
	case path == "agent/stream" || path == "agent/stream/":
		h.handleAgentStream(w, r)
	case path == "agent/tools" || path == "agent/tools/":
		h.handleAgentTools(w, r)
	case path == "admin/overview" || path == "admin/overview/":
		h.handleAdminOverview(w, r)
	case path == "admin/metrics" || path == "admin/metrics/":
		h.handleAdminMetrics(w, r)
	case path == "admin/errors" || path == "admin/errors/":
		h.handleAdminErrors(w, r)
	case path == "pipelines" || path == "pipelines/":
		h.handlePipelines(w, r)
	case strings.HasPrefix(path, "pipelines/") && strings.HasSuffix(path, "/execute"):
		id := strings.TrimSuffix(strings.TrimPrefix(path, "pipelines/"), "/execute")
		h.handlePipelineExecute(w, r, id)
	case strings.HasPrefix(path, "pipelines/"):
		h.handlePipeline(w, r, strings.TrimPrefix(path, "pipelines/"))
	// Pipeline Processing API
	case path == "pipeline/process" || path == "pipeline/process/":
		h.HandlePipelineProcess(w, r)
	case path == "pipeline/process/stream" || path == "pipeline/process/stream/":
		h.HandlePipelineProcessStream(w, r)
	case path == "pipeline/pipelines" || path == "pipeline/pipelines/":
		h.HandlePipelineDefinitions(w, r)
	case strings.HasPrefix(path, "pipeline/pipelines/"):
		h.HandlePipelineDefinition(w, r, strings.TrimPrefix(path, "pipeline/pipelines/"))
	case path == "pipeline/policies" || path == "pipeline/policies/":
		h.HandlePolicyDefinitions(w, r)
	case path == "pipeline/policies/test" || path == "pipeline/policies/test/":
		h.HandlePolicyTest(w, r)
	case strings.HasPrefix(path, "pipeline/policies/"):
		h.HandlePolicyDefinition(w, r, strings.TrimPrefix(path, "pipeline/policies/"))
	case path == "pipeline/audit" || path == "pipeline/audit/":
		h.HandleAuditLogs(w, r)
	case strings.HasPrefix(path, "pipeline/audit/"):
		h.HandleAuditLog(w, r, strings.TrimPrefix(path, "pipeline/audit/"))
	// Platon Pipeline Processing API
	case path == "platon/process" || path == "platon/process/":
		h.HandlePlatonProcess(w, r)
	case path == "platon/process/pre" || path == "platon/process/pre/":
		h.HandlePlatonProcessPre(w, r)
	case path == "platon/process/post" || path == "platon/process/post/":
		h.HandlePlatonProcessPost(w, r)
	case path == "platon/handlers" || path == "platon/handlers/":
		h.HandlePlatonListHandlers(w, r)
	case strings.HasPrefix(path, "platon/handlers/"):
		h.HandlePlatonGetHandler(w, r)
	case path == "platon/pipelines" || path == "platon/pipelines/":
		if r.Method == http.MethodPost {
			h.HandlePlatonCreatePipeline(w, r)
		} else {
			h.HandlePlatonListPipelines(w, r)
		}
	case strings.HasPrefix(path, "platon/pipelines/"):
		if r.Method == http.MethodDelete {
			h.HandlePlatonDeletePipeline(w, r)
		} else {
			h.HandlePlatonGetPipeline(w, r)
		}
	case path == "platon/stats" || path == "platon/stats/":
		h.HandlePlatonStats(w, r)
	default:
		h.writeError(w, http.StatusNotFound, "not_found", "Endpoint not found", "")
	}
}

// handleRoot handles the root endpoint
func (h *Handler) handleRoot(w http.ResponseWriter, r *http.Request) {
	info := map[string]interface{}{
		"name":    "meinDENKWERK API",
		"version": h.version,
		"endpoints": map[string][]string{
			"core": {
				"GET  /api/v1/health",
				"GET  /api/v1/services",
			},
			"llm": {
				"GET  /api/v1/models",
				"POST /api/v1/models/pull",
				"DELETE /api/v1/models/{name}",
				"POST /api/v1/chat",
				"POST /api/v1/chat/stream",
				"POST /api/v1/embed",
				"GET  /api/v1/conversations",
				"POST /api/v1/conversations",
				"GET  /api/v1/conversations/{id}",
				"DELETE /api/v1/conversations/{id}",
			},
			"rag": {
				"POST /api/v1/search",
				"POST /api/v1/search/hybrid",
				"POST /api/v1/ingest",
				"POST /api/v1/rag/augment",
				"GET  /api/v1/collections",
				"POST /api/v1/collections",
				"GET  /api/v1/collections/{name}",
				"DELETE /api/v1/collections/{name}",
				"GET  /api/v1/collections/{name}/stats",
				"GET  /api/v1/documents",
				"GET  /api/v1/documents/{id}",
				"DELETE /api/v1/documents/{id}",
			},
			"nlp": {
				"POST /api/v1/analyze",
				"POST /api/v1/summarize",
			},
			"agent": {
				"POST /api/v1/agent",
				"POST /api/v1/agent/stream",
				"GET  /api/v1/agent/tools",
			},
			"admin": {
				"GET  /api/v1/admin/overview",
				"GET  /api/v1/admin/metrics",
				"GET  /api/v1/admin/errors",
				"GET  /api/v1/pipelines",
				"POST /api/v1/pipelines",
				"GET  /api/v1/pipelines/{id}",
				"DELETE /api/v1/pipelines/{id}",
				"POST /api/v1/pipelines/{id}/execute",
			},
			"pipeline": {
				"POST /api/v1/pipeline/process",
				"POST /api/v1/pipeline/process/stream",
				"GET  /api/v1/pipeline/pipelines",
				"POST /api/v1/pipeline/pipelines",
				"GET  /api/v1/pipeline/pipelines/{id}",
				"PUT  /api/v1/pipeline/pipelines/{id}",
				"DELETE /api/v1/pipeline/pipelines/{id}",
				"GET  /api/v1/pipeline/policies",
				"POST /api/v1/pipeline/policies",
				"GET  /api/v1/pipeline/policies/{id}",
				"PUT  /api/v1/pipeline/policies/{id}",
				"DELETE /api/v1/pipeline/policies/{id}",
				"POST /api/v1/pipeline/policies/test",
				"GET  /api/v1/pipeline/audit",
				"GET  /api/v1/pipeline/audit/{request_id}",
			},
		},
	}
	h.writeJSON(w, http.StatusOK, info)
}

// handleHealth handles health check requests
func (h *Handler) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Use GET", "")
		return
	}

	services := map[string]string{"kant": "healthy"}

	// Check Turing health
	if h.clients.Turing != nil {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		resp, err := h.clients.Turing.HealthCheck(ctx, &common.HealthCheckRequest{})
		cancel()
		if err != nil {
			services["turing"] = "unhealthy"
		} else {
			services["turing"] = resp.Status
		}
	} else {
		services["turing"] = "disconnected"
	}

	// Check Hypatia health
	if h.clients.Hypatia != nil {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		resp, err := h.clients.Hypatia.HealthCheck(ctx, &common.HealthCheckRequest{})
		cancel()
		if err != nil {
			services["hypatia"] = "unhealthy"
		} else {
			services["hypatia"] = resp.Status
		}
	} else {
		services["hypatia"] = "disconnected"
	}

	// Check Leibniz health
	if h.clients.Leibniz != nil {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		resp, err := h.clients.Leibniz.HealthCheck(ctx, &common.HealthCheckRequest{})
		cancel()
		if err != nil {
			services["leibniz"] = "unhealthy"
		} else {
			services["leibniz"] = resp.Status
		}
	} else {
		services["leibniz"] = "disconnected"
	}

	// Check Babbage health
	if h.clients.Babbage != nil {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		resp, err := h.clients.Babbage.HealthCheck(ctx, &common.HealthCheckRequest{})
		cancel()
		if err != nil {
			services["babbage"] = "unhealthy"
		} else {
			services["babbage"] = resp.Status
		}
	} else {
		services["babbage"] = "disconnected"
	}

	resp := HealthResponse{
		Status:   "healthy",
		Version:  h.version,
		Uptime:   time.Since(h.startTime).String(),
		Services: services,
	}
	h.writeJSON(w, http.StatusOK, resp)
}

// handleServices handles service discovery requests
func (h *Handler) handleServices(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Use GET", "")
		return
	}

	if h.clients.Russell == nil {
		h.writeError(w, http.StatusServiceUnavailable, "service_unavailable", "Russell service not available", "")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	grpcResp, err := h.clients.Russell.ListServices(ctx, &common.Empty{})
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, "internal_error", "Failed to list services", err.Error())
		return
	}

	services := make([]ServiceInfo, len(grpcResp.Services))
	for i, s := range grpcResp.Services {
		services[i] = ServiceInfo{
			ID:       s.Id,
			Name:     s.Name,
			Address:  s.Address,
			Port:     int(s.Port),
			Status:   s.Status.String(),
			Metadata: s.Metadata,
		}
	}

	resp := ServicesResponse{
		Services: services,
		Total:    len(services),
	}
	h.writeJSON(w, http.StatusOK, resp)
}

// handleModels handles model listing
func (h *Handler) handleModels(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Use GET", "")
		return
	}

	if h.clients.Turing == nil {
		h.writeError(w, http.StatusServiceUnavailable, "service_unavailable", "Turing service not available", "")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	resp, err := h.clients.Turing.ListModels(ctx, &common.Empty{})
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, "internal_error", "Failed to list models", err.Error())
		return
	}

	models := make([]ModelInfo, len(resp.Models))
	for i, m := range resp.Models {
		models[i] = ModelInfo{
			ID:       m.Name,
			Name:     m.Name,
			Provider: m.Provider,
			Size:     m.Size,
		}
	}

	h.writeJSON(w, http.StatusOK, ModelsResponse{Models: models})
}

// handleChat handles chat completion requests
func (h *Handler) handleChat(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Use POST", "")
		return
	}

	var req ChatRequest
	if err := h.readJSON(r, &req); err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON", err.Error())
		return
	}

	if len(req.Messages) == 0 {
		h.writeError(w, http.StatusBadRequest, "invalid_request", "Messages required", "")
		return
	}

	if h.clients.Turing == nil {
		h.writeError(w, http.StatusServiceUnavailable, "service_unavailable", "Turing service not available", "")
		return
	}

	// Convert messages to protobuf format
	pbMessages := make([]*turingpb.Message, len(req.Messages))
	for i, m := range req.Messages {
		pbMessages[i] = &turingpb.Message{
			Role:    m.Role,
			Content: m.Content,
		}
	}

	ctx, cancel := context.WithTimeout(r.Context(), 120*time.Second)
	defer cancel()

	grpcReq := &turingpb.ChatRequest{
		Messages:    pbMessages,
		Model:       req.Model,
		MaxTokens:   int32(req.MaxTokens),
		Temperature: float32(req.Temperature),
	}

	grpcResp, err := h.clients.Turing.Chat(ctx, grpcReq)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, "internal_error", "Chat failed", err.Error())
		return
	}

	resp := ChatResponse{
		ID:      fmt.Sprintf("chat-%d", time.Now().UnixNano()),
		Model:   grpcResp.Model,
		Created: time.Now().Unix(),
		Message: Message{
			Role:    "assistant",
			Content: grpcResp.Content,
		},
		Usage: Usage{
			PromptTokens:     int(grpcResp.PromptTokens),
			CompletionTokens: int(grpcResp.CompletionTokens),
			TotalTokens:      int(grpcResp.TotalTokens),
		},
	}
	h.writeJSON(w, http.StatusOK, resp)
}

// handleChatStream handles streaming chat requests via SSE
func (h *Handler) handleChatStream(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Use POST", "")
		return
	}

	var req ChatRequest
	if err := h.readJSON(r, &req); err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON", err.Error())
		return
	}

	if len(req.Messages) == 0 {
		h.writeError(w, http.StatusBadRequest, "invalid_request", "Messages required", "")
		return
	}

	if h.clients.Turing == nil {
		h.writeError(w, http.StatusServiceUnavailable, "service_unavailable", "Turing service not available", "")
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

	// Convert messages to protobuf format
	pbMessages := make([]*turingpb.Message, len(req.Messages))
	for i, m := range req.Messages {
		pbMessages[i] = &turingpb.Message{
			Role:    m.Role,
			Content: m.Content,
		}
	}

	ctx, cancel := context.WithTimeout(r.Context(), 120*time.Second)
	defer cancel()

	grpcReq := &turingpb.ChatRequest{
		Messages:    pbMessages,
		Model:       req.Model,
		MaxTokens:   int32(req.MaxTokens),
		Temperature: float32(req.Temperature),
	}

	stream, err := h.clients.Turing.StreamChat(ctx, grpcReq)
	if err != nil {
		fmt.Fprintf(w, "event: error\ndata: %s\n\n", err.Error())
		flusher.Flush()
		return
	}

	for {
		chunk, err := stream.Recv()
		if err == io.EOF {
			fmt.Fprintf(w, "event: done\ndata: [DONE]\n\n")
			flusher.Flush()
			break
		}
		if err != nil {
			fmt.Fprintf(w, "event: error\ndata: %s\n\n", err.Error())
			flusher.Flush()
			break
		}

		data := map[string]interface{}{
			"content": chunk.Delta,
			"done":    chunk.Done,
		}
		jsonData, _ := json.Marshal(data)
		fmt.Fprintf(w, "data: %s\n\n", jsonData)
		flusher.Flush()
	}
}

// handleSearch handles RAG search requests
func (h *Handler) handleSearch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Use POST", "")
		return
	}

	var req SearchRequest
	if err := h.readJSON(r, &req); err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON", err.Error())
		return
	}

	if req.Query == "" {
		h.writeError(w, http.StatusBadRequest, "invalid_request", "Query required", "")
		return
	}

	if h.clients.Hypatia == nil {
		h.writeError(w, http.StatusServiceUnavailable, "service_unavailable", "Hypatia service not available", "")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	grpcReq := &hypatiapb.SearchRequest{
		Query:      req.Query,
		Collection: req.Collection,
		TopK:       int32(req.TopK),
		MinScore:   float32(req.MinScore),
	}

	grpcResp, err := h.clients.Hypatia.Search(ctx, grpcReq)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, "internal_error", "Search failed", err.Error())
		return
	}

	results := make([]SearchResult, len(grpcResp.Results))
	for i, r := range grpcResp.Results {
		results[i] = SearchResult{
			ID:      r.DocumentId,
			Content: r.Content,
			Score:   float64(r.Score),
		}
	}

	resp := SearchResponse{
		Query:   req.Query,
		Results: results,
		Total:   len(results),
	}
	h.writeJSON(w, http.StatusOK, resp)
}

// handleIngest handles document ingestion
func (h *Handler) handleIngest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Use POST", "")
		return
	}

	var req IngestRequest
	if err := h.readJSON(r, &req); err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON", err.Error())
		return
	}

	if req.Content == "" {
		h.writeError(w, http.StatusBadRequest, "invalid_request", "Content required", "")
		return
	}

	if h.clients.Hypatia == nil {
		h.writeError(w, http.StatusServiceUnavailable, "service_unavailable", "Hypatia service not available", "")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 60*time.Second)
	defer cancel()

	grpcReq := &hypatiapb.IngestDocumentRequest{
		Content:    req.Content,
		Title:      req.Title,
		Source:     req.Source,
		Collection: req.Collection,
		Metadata:   req.Metadata,
	}

	grpcResp, err := h.clients.Hypatia.IngestDocument(ctx, grpcReq)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, "internal_error", "Ingest failed", err.Error())
		return
	}

	resp := IngestResponse{
		DocumentID: grpcResp.DocumentId,
		Success:    grpcResp.Success,
	}
	h.writeJSON(w, http.StatusOK, resp)
}

// handleAnalyze handles NLP analysis requests
func (h *Handler) handleAnalyze(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Use POST", "")
		return
	}

	var req AnalyzeRequest
	if err := h.readJSON(r, &req); err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON", err.Error())
		return
	}

	if req.Text == "" {
		h.writeError(w, http.StatusBadRequest, "invalid_request", "Text required", "")
		return
	}

	if h.clients.Babbage == nil {
		h.writeError(w, http.StatusServiceUnavailable, "service_unavailable", "Babbage service not available", "")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	grpcReq := &babbagepb.AnalyzeRequest{
		Text: req.Text,
	}

	grpcResp, err := h.clients.Babbage.Analyze(ctx, grpcReq)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, "internal_error", "Analysis failed", err.Error())
		return
	}

	// Convert entities
	entities := make([]Entity, len(grpcResp.Entities))
	for i, e := range grpcResp.Entities {
		entities[i] = Entity{
			Text:  e.Text,
			Type:  e.Type.String(),
			Start: int(e.Start),
			End:   int(e.End),
		}
	}

	// Convert keywords
	keywords := make([]Keyword, len(grpcResp.Keywords))
	for i, k := range grpcResp.Keywords {
		keywords[i] = Keyword{
			Word:  k.Word,
			Score: float64(k.Score),
		}
	}

	// Convert sentiment
	var sentiment *SentimentResult
	if grpcResp.Sentiment != nil {
		sentiment = &SentimentResult{
			Label:      grpcResp.Sentiment.Sentiment.String(),
			Confidence: float64(grpcResp.Sentiment.Confidence),
		}
	}

	resp := AnalyzeResponse{
		Language:  grpcResp.Language,
		Sentiment: sentiment,
		Entities:  entities,
		Keywords:  keywords,
	}
	h.writeJSON(w, http.StatusOK, resp)
}

// handleSummarize handles summarization requests
func (h *Handler) handleSummarize(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Use POST", "")
		return
	}

	var req SummarizeRequest
	if err := h.readJSON(r, &req); err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON", err.Error())
		return
	}

	if req.Text == "" {
		h.writeError(w, http.StatusBadRequest, "invalid_request", "Text required", "")
		return
	}

	if h.clients.Babbage == nil {
		h.writeError(w, http.StatusServiceUnavailable, "service_unavailable", "Babbage service not available", "")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 60*time.Second)
	defer cancel()

	// Convert style string to enum
	style := babbagepb.SummarizationStyle_SUMMARIZATION_STYLE_BRIEF
	switch req.Style {
	case "detailed":
		style = babbagepb.SummarizationStyle_SUMMARIZATION_STYLE_DETAILED
	case "bullet":
		style = babbagepb.SummarizationStyle_SUMMARIZATION_STYLE_BULLET_POINTS
	case "headline":
		style = babbagepb.SummarizationStyle_SUMMARIZATION_STYLE_HEADLINE
	}

	grpcReq := &babbagepb.SummarizeRequest{
		Text:      req.Text,
		MaxLength: int32(req.MaxLength),
		Style:     style,
	}

	grpcResp, err := h.clients.Babbage.Summarize(ctx, grpcReq)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, "internal_error", "Summarization failed", err.Error())
		return
	}

	resp := SummarizeResponse{
		Summary:        grpcResp.Summary,
		OriginalLength: int(grpcResp.OriginalLength),
		SummaryLength:  int(grpcResp.SummaryLength),
	}
	h.writeJSON(w, http.StatusOK, resp)
}

// handleAgent handles agent execution requests
func (h *Handler) handleAgent(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Use POST", "")
		return
	}

	var req AgentRequest
	if err := h.readJSON(r, &req); err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON", err.Error())
		return
	}

	// Accept either "task" or "message" field
	task := req.Task
	if task == "" {
		task = req.Message
	}
	if task == "" {
		h.writeError(w, http.StatusBadRequest, "invalid_request", "Task or message required", "")
		return
	}

	if h.clients.Leibniz == nil {
		h.writeError(w, http.StatusServiceUnavailable, "service_unavailable", "Leibniz service not available", "")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 300*time.Second)
	defer cancel()

	grpcReq := &leibnizpb.ExecuteRequest{
		AgentId: req.AgentID,
		Message: task,
	}

	grpcResp, err := h.clients.Leibniz.Execute(ctx, grpcReq)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, "internal_error", "Agent execution failed", err.Error())
		return
	}

	resp := AgentResponse{
		ID:       grpcResp.ExecutionId,
		Status:   grpcResp.Status.String(),
		Result:   grpcResp.Response,
		Response: grpcResp.Response, // Alias for compatibility
		Steps:    []AgentStep{},
	}
	h.writeJSON(w, http.StatusOK, resp)
}

// handleAgentStream handles streaming agent execution via SSE
func (h *Handler) handleAgentStream(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Use POST", "")
		return
	}

	var req AgentRequest
	if err := h.readJSON(r, &req); err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON", err.Error())
		return
	}

	if req.Task == "" {
		h.writeError(w, http.StatusBadRequest, "invalid_request", "Task required", "")
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

	ctx, cancel := context.WithTimeout(r.Context(), 300*time.Second)
	defer cancel()

	grpcReq := &leibnizpb.ExecuteRequest{
		Message: req.Task,
	}

	stream, err := h.clients.Leibniz.StreamExecute(ctx, grpcReq)
	if err != nil {
		fmt.Fprintf(w, "event: error\ndata: %s\n\n", err.Error())
		flusher.Flush()
		return
	}

	for {
		chunk, err := stream.Recv()
		if err == io.EOF {
			fmt.Fprintf(w, "event: done\ndata: [DONE]\n\n")
			flusher.Flush()
			break
		}
		if err != nil {
			fmt.Fprintf(w, "event: error\ndata: %s\n\n", err.Error())
			flusher.Flush()
			break
		}

		data := map[string]interface{}{
			"type":      chunk.Type.String(),
			"content":   chunk.Content,
			"iteration": chunk.Iteration,
		}
		jsonData, _ := json.Marshal(data)
		fmt.Fprintf(w, "data: %s\n\n", jsonData)
		flusher.Flush()
	}
}

// handleAgentTools handles listing available agent tools
func (h *Handler) handleAgentTools(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Use GET", "")
		return
	}

	if h.clients.Leibniz == nil {
		h.writeError(w, http.StatusServiceUnavailable, "service_unavailable", "Leibniz service not available", "")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	grpcResp, err := h.clients.Leibniz.ListTools(ctx, &common.Empty{})
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, "internal_error", "Failed to list tools", err.Error())
		return
	}

	tools := make([]map[string]interface{}, len(grpcResp.Tools))
	for i, t := range grpcResp.Tools {
		tools[i] = map[string]interface{}{
			"name":        t.Name,
			"description": t.Description,
			"enabled":     t.Enabled,
		}
	}

	h.writeJSON(w, http.StatusOK, map[string]interface{}{"tools": tools})
}

// ============================================================================
// Hypatia (RAG) Endpoints
// ============================================================================

// handleCollections handles collection listing and creation
func (h *Handler) handleCollections(w http.ResponseWriter, r *http.Request) {
	if h.clients.Hypatia == nil {
		h.writeError(w, http.StatusServiceUnavailable, "service_unavailable", "Hypatia service not available", "")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	switch r.Method {
	case http.MethodGet:
		grpcResp, err := h.clients.Hypatia.ListCollections(ctx, &common.Empty{})
		if err != nil {
			h.writeError(w, http.StatusInternalServerError, "internal_error", "Failed to list collections", err.Error())
			return
		}

		collections := make([]CollectionResponse, len(grpcResp.Collections))
		for i, c := range grpcResp.Collections {
			collections[i] = CollectionResponse{
				Name:  c.Name,
				Count: int64(c.DocumentCount),
			}
		}

		h.writeJSON(w, http.StatusOK, CollectionsResponse{
			Collections: collections,
			Total:       len(collections),
		})

	case http.MethodPost:
		var req CollectionRequest
		if err := h.readJSON(r, &req); err != nil {
			h.writeError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON", err.Error())
			return
		}

		if req.Name == "" {
			h.writeError(w, http.StatusBadRequest, "invalid_request", "Collection name required", "")
			return
		}

		grpcReq := &hypatiapb.CreateCollectionRequest{
			Name: req.Name,
		}

		grpcResp, err := h.clients.Hypatia.CreateCollection(ctx, grpcReq)
		if err != nil {
			h.writeError(w, http.StatusInternalServerError, "internal_error", "Failed to create collection", err.Error())
			return
		}

		h.writeJSON(w, http.StatusCreated, CollectionResponse{
			Name:  grpcResp.Name,
			Count: 0,
		})

	default:
		h.writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Use GET or POST", "")
	}
}

// handleCollection handles single collection operations
func (h *Handler) handleCollection(w http.ResponseWriter, r *http.Request, name string) {
	if h.clients.Hypatia == nil {
		h.writeError(w, http.StatusServiceUnavailable, "service_unavailable", "Hypatia service not available", "")
		return
	}

	name = strings.TrimSuffix(name, "/")
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	switch r.Method {
	case http.MethodGet:
		// Use GetCollectionStats to get collection info
		grpcReq := &hypatiapb.GetCollectionStatsRequest{Name: name}
		grpcResp, err := h.clients.Hypatia.GetCollectionStats(ctx, grpcReq)
		if err != nil {
			h.writeError(w, http.StatusNotFound, "not_found", "Collection not found", err.Error())
			return
		}

		h.writeJSON(w, http.StatusOK, CollectionResponse{
			Name:  grpcResp.Name,
			Count: int64(grpcResp.DocumentCount),
		})

	case http.MethodDelete:
		grpcReq := &hypatiapb.DeleteCollectionRequest{Name: name}
		_, err := h.clients.Hypatia.DeleteCollection(ctx, grpcReq)
		if err != nil {
			h.writeError(w, http.StatusInternalServerError, "internal_error", "Failed to delete collection", err.Error())
			return
		}

		h.writeJSON(w, http.StatusOK, map[string]interface{}{
			"success": true,
			"message": fmt.Sprintf("Collection '%s' deleted", name),
		})

	default:
		h.writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Use GET or DELETE", "")
	}
}

// handleCollectionStats handles collection statistics
func (h *Handler) handleCollectionStats(w http.ResponseWriter, r *http.Request, name string) {
	if r.Method != http.MethodGet {
		h.writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Use GET", "")
		return
	}

	if h.clients.Hypatia == nil {
		h.writeError(w, http.StatusServiceUnavailable, "service_unavailable", "Hypatia service not available", "")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	grpcReq := &hypatiapb.GetCollectionStatsRequest{Name: name}
	grpcResp, err := h.clients.Hypatia.GetCollectionStats(ctx, grpcReq)
	if err != nil {
		h.writeError(w, http.StatusNotFound, "not_found", "Collection not found", err.Error())
		return
	}

	h.writeJSON(w, http.StatusOK, CollectionStatsResponse{
		Name:          name,
		DocumentCount: int64(grpcResp.DocumentCount),
		ChunkCount:    int64(grpcResp.ChunkCount),
		VectorCount:   grpcResp.TotalTokens, // Use TotalTokens as proxy for vector count
		StorageSize:   grpcResp.StorageBytes,
	})
}

// handleDocuments handles document listing
func (h *Handler) handleDocuments(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Use GET", "")
		return
	}

	if h.clients.Hypatia == nil {
		h.writeError(w, http.StatusServiceUnavailable, "service_unavailable", "Hypatia service not available", "")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	collection := r.URL.Query().Get("collection")
	grpcReq := &hypatiapb.ListDocumentsRequest{
		Collection: collection,
	}

	grpcResp, err := h.clients.Hypatia.ListDocuments(ctx, grpcReq)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, "internal_error", "Failed to list documents", err.Error())
		return
	}

	documents := make([]DocumentResponse, len(grpcResp.Documents))
	for i, d := range grpcResp.Documents {
		// Convert DocumentMetadata to map[string]string
		var metadata map[string]string
		if d.Metadata != nil && d.Metadata.Custom != nil {
			metadata = d.Metadata.Custom
		}
		documents[i] = DocumentResponse{
			ID:         d.Id,
			Title:      d.Title,
			Source:     d.Source,
			Collection: d.Collection,
			Metadata:   metadata,
			ChunkCount: int(d.ChunkCount),
		}
	}

	h.writeJSON(w, http.StatusOK, DocumentsResponse{
		Documents: documents,
		Total:     len(documents),
	})
}

// handleDocument handles single document operations
func (h *Handler) handleDocument(w http.ResponseWriter, r *http.Request, id string) {
	if h.clients.Hypatia == nil {
		h.writeError(w, http.StatusServiceUnavailable, "service_unavailable", "Hypatia service not available", "")
		return
	}

	id = strings.TrimSuffix(id, "/")
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	switch r.Method {
	case http.MethodGet:
		grpcReq := &hypatiapb.GetDocumentRequest{DocumentId: id}
		grpcResp, err := h.clients.Hypatia.GetDocument(ctx, grpcReq)
		if err != nil {
			h.writeError(w, http.StatusNotFound, "not_found", "Document not found", err.Error())
			return
		}

		// Convert DocumentMetadata to map[string]string
		var metadata map[string]string
		if grpcResp.Metadata != nil && grpcResp.Metadata.Custom != nil {
			metadata = grpcResp.Metadata.Custom
		}

		h.writeJSON(w, http.StatusOK, DocumentResponse{
			ID:         grpcResp.Id,
			Title:      grpcResp.Title,
			Source:     grpcResp.Source,
			Collection: grpcResp.Collection,
			Metadata:   metadata,
			ChunkCount: int(grpcResp.ChunkCount),
		})

	case http.MethodDelete:
		grpcReq := &hypatiapb.DeleteDocumentRequest{DocumentId: id}
		_, err := h.clients.Hypatia.DeleteDocument(ctx, grpcReq)
		if err != nil {
			h.writeError(w, http.StatusInternalServerError, "internal_error", "Failed to delete document", err.Error())
			return
		}

		h.writeJSON(w, http.StatusOK, map[string]interface{}{
			"success": true,
			"message": fmt.Sprintf("Document '%s' deleted", id),
		})

	default:
		h.writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Use GET or DELETE", "")
	}
}

// handleHybridSearch handles hybrid search requests
func (h *Handler) handleHybridSearch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Use POST", "")
		return
	}

	var req HybridSearchRequest
	if err := h.readJSON(r, &req); err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON", err.Error())
		return
	}

	if req.Query == "" {
		h.writeError(w, http.StatusBadRequest, "invalid_request", "Query required", "")
		return
	}

	if h.clients.Hypatia == nil {
		h.writeError(w, http.StatusServiceUnavailable, "service_unavailable", "Hypatia service not available", "")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	// Default vector weight if not specified
	vectorWeight := float32(0.7)
	if req.AlphaVector > 0 {
		vectorWeight = float32(req.AlphaVector)
	}

	grpcReq := &hypatiapb.HybridSearchRequest{
		Query:         req.Query,
		Collection:    req.Collection,
		TopK:          int32(req.TopK),
		MinScore:      float32(req.MinScore),
		VectorWeight:  vectorWeight,
		KeywordWeight: 1.0 - vectorWeight,
	}

	grpcResp, err := h.clients.Hypatia.HybridSearch(ctx, grpcReq)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, "internal_error", "Hybrid search failed", err.Error())
		return
	}

	results := make([]SearchResult, len(grpcResp.Results))
	for i, r := range grpcResp.Results {
		results[i] = SearchResult{
			ID:      r.DocumentId,
			Content: r.Content,
			Score:   float64(r.Score),
		}
	}

	h.writeJSON(w, http.StatusOK, SearchResponse{
		Query:   req.Query,
		Results: results,
		Total:   len(results),
	})
}

// handleRAGAugment handles RAG augmentation requests
func (h *Handler) handleRAGAugment(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Use POST", "")
		return
	}

	var req RAGAugmentRequest
	if err := h.readJSON(r, &req); err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON", err.Error())
		return
	}

	if req.Query == "" {
		h.writeError(w, http.StatusBadRequest, "invalid_request", "Query required", "")
		return
	}

	if h.clients.Hypatia == nil {
		h.writeError(w, http.StatusServiceUnavailable, "service_unavailable", "Hypatia service not available", "")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 60*time.Second)
	defer cancel()

	// Use AugmentPrompt RPC
	grpcReq := &hypatiapb.AugmentPromptRequest{
		Prompt:     req.Query,
		Collection: req.Collection,
		TopK:       int32(req.TopK),
	}

	grpcResp, err := h.clients.Hypatia.AugmentPrompt(ctx, grpcReq)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, "internal_error", "RAG augmentation failed", err.Error())
		return
	}

	sources := make([]SearchResult, len(grpcResp.Sources))
	for i, s := range grpcResp.Sources {
		sources[i] = SearchResult{
			ID:      s.DocumentId,
			Content: s.Content,
			Score:   float64(s.Score),
		}
	}

	h.writeJSON(w, http.StatusOK, RAGAugmentResponse{
		Query:        req.Query,
		Answer:       grpcResp.AugmentedPrompt,
		Sources:      sources,
		PromptTokens: int(grpcResp.ContextTokens),
	})
}

// ============================================================================
// Turing (LLM) Additional Endpoints
// ============================================================================

// handleEmbed handles embedding requests
func (h *Handler) handleEmbed(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Use POST", "")
		return
	}

	var req EmbedRequest
	if err := h.readJSON(r, &req); err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON", err.Error())
		return
	}

	// Handle both single text and multiple texts
	texts := req.Texts
	if req.Text != "" && len(texts) == 0 {
		texts = []string{req.Text}
	}

	if len(texts) == 0 {
		h.writeError(w, http.StatusBadRequest, "invalid_request", "Text or texts required", "")
		return
	}

	if h.clients.Turing == nil {
		h.writeError(w, http.StatusServiceUnavailable, "service_unavailable", "Turing service not available", "")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 60*time.Second)
	defer cancel()

	// Use BatchEmbed for multiple texts, Embed for single text
	if len(texts) == 1 {
		grpcReq := &turingpb.EmbedRequest{
			Input: texts[0],
			Model: req.Model,
		}

		grpcResp, err := h.clients.Turing.Embed(ctx, grpcReq)
		if err != nil {
			h.writeError(w, http.StatusInternalServerError, "internal_error", "Embedding failed", err.Error())
			return
		}

		// Convert single embedding
		embedding := make([]float64, len(grpcResp.Embedding))
		for i, v := range grpcResp.Embedding {
			embedding[i] = float64(v)
		}

		h.writeJSON(w, http.StatusOK, EmbedResponse{
			Embeddings: [][]float64{embedding},
			Model:      grpcResp.Model,
			Dimensions: int(grpcResp.Dimensions),
		})
	} else {
		grpcReq := &turingpb.BatchEmbedRequest{
			Inputs: texts,
			Model:  req.Model,
		}

		grpcResp, err := h.clients.Turing.BatchEmbed(ctx, grpcReq)
		if err != nil {
			h.writeError(w, http.StatusInternalServerError, "internal_error", "Batch embedding failed", err.Error())
			return
		}

		embeddings := make([][]float64, len(grpcResp.Embeddings))
		for i, emb := range grpcResp.Embeddings {
			embeddings[i] = make([]float64, len(emb.Embedding))
			for j, v := range emb.Embedding {
				embeddings[i][j] = float64(v)
			}
		}

		h.writeJSON(w, http.StatusOK, EmbedResponse{
			Embeddings: embeddings,
			Model:      grpcResp.Model,
			Dimensions: len(embeddings[0]),
		})
	}
}

// handleModelPull handles model pull requests (streams progress via SSE)
func (h *Handler) handleModelPull(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Use POST", "")
		return
	}

	var req ModelPullRequest
	if err := h.readJSON(r, &req); err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON", err.Error())
		return
	}

	if req.Model == "" {
		h.writeError(w, http.StatusBadRequest, "invalid_request", "Model name required", "")
		return
	}

	if h.clients.Turing == nil {
		h.writeError(w, http.StatusServiceUnavailable, "service_unavailable", "Turing service not available", "")
		return
	}

	// Set SSE headers for streaming progress
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	flusher, ok := w.(http.Flusher)
	if !ok {
		h.writeError(w, http.StatusInternalServerError, "internal_error", "Streaming not supported", "")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 600*time.Second) // Long timeout for model pull
	defer cancel()

	grpcReq := &turingpb.PullModelRequest{
		Name: req.Model,
	}

	stream, err := h.clients.Turing.PullModel(ctx, grpcReq)
	if err != nil {
		fmt.Fprintf(w, "event: error\ndata: %s\n\n", err.Error())
		flusher.Flush()
		return
	}

	var lastStatus string
	for {
		progress, err := stream.Recv()
		if err == io.EOF {
			data := map[string]interface{}{
				"model":   req.Model,
				"status":  "completed",
				"percent": 100.0,
			}
			jsonData, _ := json.Marshal(data)
			fmt.Fprintf(w, "event: done\ndata: %s\n\n", jsonData)
			flusher.Flush()
			break
		}
		if err != nil {
			fmt.Fprintf(w, "event: error\ndata: %s\n\n", err.Error())
			flusher.Flush()
			break
		}

		lastStatus = progress.Status
		data := map[string]interface{}{
			"model":     req.Model,
			"status":    progress.Status,
			"completed": progress.Completed,
			"total":     progress.Total,
			"percent":   progress.Percent,
		}
		jsonData, _ := json.Marshal(data)
		fmt.Fprintf(w, "data: %s\n\n", jsonData)
		flusher.Flush()
	}

	_ = lastStatus // Suppress unused variable warning
}

// handleModelDelete handles model deletion
// TODO: Add DeleteModel to Turing proto when Ollama deletion support is needed
func (h *Handler) handleModelDelete(w http.ResponseWriter, r *http.Request, name string) {
	if r.Method != http.MethodDelete {
		h.writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Use DELETE", "")
		return
	}

	// Model deletion not yet implemented in gRPC service
	h.writeError(w, http.StatusNotImplemented, "not_implemented", "Model deletion not yet implemented", "")
}

// handleConversations handles conversation listing and creation
// TODO: Add conversation management to Turing proto for persistent chat history
func (h *Handler) handleConversations(w http.ResponseWriter, r *http.Request) {
	// Conversation management not yet implemented in gRPC service
	h.writeError(w, http.StatusNotImplemented, "not_implemented", "Conversation management not yet implemented", "")
}

// handleConversation handles single conversation operations
// TODO: Add conversation management to Turing proto for persistent chat history
func (h *Handler) handleConversation(w http.ResponseWriter, r *http.Request, id string) {
	// Conversation management not yet implemented in gRPC service
	h.writeError(w, http.StatusNotImplemented, "not_implemented", "Conversation management not yet implemented", "")
}

// ============================================================================
// Russell (Admin/Orchestration) Endpoints
// ============================================================================

// handleAdminOverview handles system overview requests
// TODO: Implement once Russell proto is regenerated with admin endpoints
func (h *Handler) handleAdminOverview(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Use GET", "")
		return
	}

	// Admin overview requires proto regeneration - returning stub data for now
	h.writeJSON(w, http.StatusOK, AdminOverviewResponse{
		Timestamp:       time.Now().Format(time.RFC3339),
		TotalServices:   5,
		HealthyServices: 0,
		Services:        make(map[string]ServiceStatus),
		Metrics: &SystemMetricsResponse{
			TotalRequests: 0,
		},
	})
}

// handleAdminMetrics handles system metrics requests
// TODO: Implement once Russell proto is regenerated with admin endpoints
func (h *Handler) handleAdminMetrics(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Use GET", "")
		return
	}

	// Metrics requires proto regeneration - returning stub data for now
	h.writeJSON(w, http.StatusOK, SystemMetricsResponse{
		TotalRequests:       0,
		SuccessfulRequests:  0,
		FailedRequests:      0,
		AverageResponseTime: "0ms",
		RequestsPerSecond:   0,
	})
}

// handleAdminErrors handles error listing requests
// TODO: Implement once Russell proto is regenerated with admin endpoints
func (h *Handler) handleAdminErrors(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Use GET", "")
		return
	}

	// Errors requires proto regeneration - returning empty for now
	h.writeJSON(w, http.StatusOK, map[string]interface{}{
		"errors": []ErrorEntryResponse{},
		"total":  0,
	})
}

// handlePipelines handles pipeline listing and creation
// TODO: Implement once Russell proto is regenerated with pipeline endpoints
func (h *Handler) handlePipelines(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		// Pipelines requires proto regeneration - returning empty for now
		h.writeJSON(w, http.StatusOK, PipelinesResponse{
			Pipelines: []PipelineResponse{},
			Total:     0,
		})
	case http.MethodPost:
		h.writeError(w, http.StatusNotImplemented, "not_implemented", "Pipeline creation requires proto regeneration", "")
	default:
		h.writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Use GET or POST", "")
	}
}

// handlePipeline handles single pipeline operations
// TODO: Implement once Russell proto is regenerated with pipeline endpoints
func (h *Handler) handlePipeline(w http.ResponseWriter, r *http.Request, id string) {
	h.writeError(w, http.StatusNotImplemented, "not_implemented", "Pipeline operations require proto regeneration", "")
}

// handlePipelineExecute handles pipeline execution
// TODO: Implement once Russell proto is regenerated with pipeline endpoints
func (h *Handler) handlePipelineExecute(w http.ResponseWriter, r *http.Request, id string) {
	h.writeError(w, http.StatusNotImplemented, "not_implemented", "Pipeline execution requires proto regeneration", "")
}

// Helper methods

func (h *Handler) readJSON(r *http.Request, v interface{}) error {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return err
	}
	return json.Unmarshal(body, v)
}

func (h *Handler) writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func (h *Handler) writeError(w http.ResponseWriter, status int, code, message, details string) {
	resp := ErrorResponse{
		Error:   message,
		Code:    code,
		Details: details,
	}
	h.writeJSON(w, status, resp)
}
