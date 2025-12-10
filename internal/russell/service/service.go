package service

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/msto63/mDW/internal/russell/admin"
	"github.com/msto63/mDW/pkg/core/discovery"
	"github.com/msto63/mDW/pkg/core/logging"
)

// ServiceType represents the type of AI service
type ServiceType string

const (
	ServiceTypeTuring  ServiceType = "turing"   // LLM
	ServiceTypeHypatia ServiceType = "hypatia"  // RAG
	ServiceTypeLeibniz ServiceType = "leibniz"  // Agentic
	ServiceTypeBabbage ServiceType = "babbage"  // NLP
)

// PipelineStep represents a single step in a processing pipeline
type PipelineStep struct {
	ID          string
	ServiceType ServiceType
	Operation   string
	Parameters  map[string]interface{}
	DependsOn   []string
}

// Pipeline represents a multi-step processing workflow
type Pipeline struct {
	ID          string
	Name        string
	Description string
	Steps       []PipelineStep
	CreatedAt   time.Time
}

// PipelineExecution represents an execution of a pipeline
type PipelineExecution struct {
	ID          string
	PipelineID  string
	Status      ExecutionStatus
	StartedAt   time.Time
	CompletedAt time.Time
	StepResults map[string]*StepResult
	Error       string
}

// ExecutionStatus represents pipeline execution status
type ExecutionStatus string

const (
	ExecutionStatusPending   ExecutionStatus = "pending"
	ExecutionStatusRunning   ExecutionStatus = "running"
	ExecutionStatusCompleted ExecutionStatus = "completed"
	ExecutionStatusFailed    ExecutionStatus = "failed"
	ExecutionStatusCancelled ExecutionStatus = "cancelled"
)

// StepResult represents the result of a pipeline step
type StepResult struct {
	StepID      string
	Status      ExecutionStatus
	StartedAt   time.Time
	CompletedAt time.Time
	Output      interface{}
	Error       string
}

// Request represents a generic service request
type Request struct {
	ID          string
	ServiceType ServiceType
	Operation   string
	Input       interface{}
	Parameters  map[string]interface{}
	Timeout     time.Duration
}

// Response represents a generic service response
type Response struct {
	RequestID string
	Success   bool
	Output    interface{}
	Error     string
	Duration  time.Duration
	Metadata  map[string]interface{}
}

// Service is the Russell orchestration service
type Service struct {
	logger     *logging.Logger
	locator    *discovery.ServiceLocator
	admin      *admin.Admin
	pipelines  map[string]*Pipeline
	executions map[string]*PipelineExecution
	mu         sync.RWMutex
}

// Config holds configuration for the Russell service
type Config struct {
	DiscoveryClient discovery.Client
	CacheTTL        time.Duration
	MaxErrorHistory int
}

// NewService creates a new Russell orchestration service
func NewService(cfg Config) (*Service, error) {
	logger := logging.New("russell")

	locator := discovery.NewServiceLocator(cfg.DiscoveryClient, cfg.CacheTTL)

	adminCfg := admin.Config{
		DiscoveryClient: cfg.DiscoveryClient,
		MaxErrorHistory: cfg.MaxErrorHistory,
	}
	adminInstance := admin.NewAdmin(adminCfg)

	return &Service{
		logger:     logger,
		locator:    locator,
		admin:      adminInstance,
		pipelines:  make(map[string]*Pipeline),
		executions: make(map[string]*PipelineExecution),
	}, nil
}

// Execute executes a single service request
func (s *Service) Execute(ctx context.Context, req *Request) (*Response, error) {
	start := time.Now()

	s.logger.Info("Executing request",
		"id", req.ID,
		"service", req.ServiceType,
		"operation", req.Operation,
	)

	// Find service instance
	svcInfo, err := s.locator.Locate(ctx, string(req.ServiceType))
	if err != nil {
		duration := time.Since(start)
		s.admin.RecordRequest(string(req.ServiceType), req.Operation, false, duration, req.ID)
		s.admin.RecordError(string(req.ServiceType), req.Operation, "SERVICE_DISCOVERY_FAILED", err.Error(), req.ID)
		return nil, fmt.Errorf("service discovery failed: %w", err)
	}

	s.logger.Debug("Found service instance",
		"service", req.ServiceType,
		"address", svcInfo.FullAddress(),
	)

	// TODO: Actually call the service via gRPC
	// For now, return a placeholder response
	response := &Response{
		RequestID: req.ID,
		Success:   true,
		Output:    fmt.Sprintf("Executed %s on %s", req.Operation, req.ServiceType),
		Duration:  time.Since(start),
		Metadata: map[string]interface{}{
			"service_instance": svcInfo.FullAddress(),
		},
	}

	// Record metrics
	s.admin.RecordRequest(string(req.ServiceType), req.Operation, response.Success, response.Duration, req.ID)

	s.logger.Info("Request completed",
		"id", req.ID,
		"success", response.Success,
		"duration", response.Duration,
	)

	return response, nil
}

// RegisterPipeline registers a new pipeline
func (s *Service) RegisterPipeline(pipeline *Pipeline) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if pipeline.ID == "" {
		return fmt.Errorf("pipeline ID is required")
	}

	pipeline.CreatedAt = time.Now()
	s.pipelines[pipeline.ID] = pipeline

	s.logger.Info("Pipeline registered",
		"id", pipeline.ID,
		"name", pipeline.Name,
		"steps", len(pipeline.Steps),
	)

	return nil
}

// GetPipeline returns a pipeline by ID
func (s *Service) GetPipeline(id string) (*Pipeline, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	pipeline, ok := s.pipelines[id]
	if !ok {
		return nil, fmt.Errorf("pipeline not found: %s", id)
	}

	return pipeline, nil
}

// ListPipelines returns all registered pipelines
func (s *Service) ListPipelines() []*Pipeline {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]*Pipeline, 0, len(s.pipelines))
	for _, p := range s.pipelines {
		result = append(result, p)
	}
	return result
}

// ExecutePipeline executes a pipeline and returns the execution result
func (s *Service) ExecutePipeline(ctx context.Context, pipelineID string, input interface{}) (*PipelineExecution, error) {
	pipeline, err := s.GetPipeline(pipelineID)
	if err != nil {
		return nil, err
	}

	execution := &PipelineExecution{
		ID:          fmt.Sprintf("exec-%d", time.Now().UnixNano()),
		PipelineID:  pipelineID,
		Status:      ExecutionStatusRunning,
		StartedAt:   time.Now(),
		StepResults: make(map[string]*StepResult),
	}

	s.logger.Info("Starting pipeline execution",
		"execution_id", execution.ID,
		"pipeline_id", pipelineID,
		"pipeline_name", pipeline.Name,
	)

	// Build dependency graph and execute steps
	stepOutputs := make(map[string]interface{})
	stepOutputs["input"] = input

	for _, step := range pipeline.Steps {
		// Check if dependencies are satisfied
		for _, dep := range step.DependsOn {
			if _, ok := stepOutputs[dep]; !ok {
				execution.Status = ExecutionStatusFailed
				execution.Error = fmt.Sprintf("dependency not satisfied: %s", dep)
				execution.CompletedAt = time.Now()
				// Store failed execution
				s.mu.Lock()
				s.executions[execution.ID] = execution
				s.mu.Unlock()
				return execution, nil
			}
		}

		stepResult := &StepResult{
			StepID:    step.ID,
			Status:    ExecutionStatusRunning,
			StartedAt: time.Now(),
		}

		// Prepare step input from dependencies
		stepInput := make(map[string]interface{})
		for _, dep := range step.DependsOn {
			stepInput[dep] = stepOutputs[dep]
		}
		if len(step.DependsOn) == 0 {
			stepInput["input"] = input
		}

		// Execute step
		req := &Request{
			ID:          fmt.Sprintf("%s-%s", execution.ID, step.ID),
			ServiceType: step.ServiceType,
			Operation:   step.Operation,
			Input:       stepInput,
			Parameters:  step.Parameters,
		}

		resp, err := s.Execute(ctx, req)
		if err != nil {
			stepResult.Status = ExecutionStatusFailed
			stepResult.Error = err.Error()
			stepResult.CompletedAt = time.Now()
			execution.StepResults[step.ID] = stepResult

			execution.Status = ExecutionStatusFailed
			execution.Error = fmt.Sprintf("step %s failed: %v", step.ID, err)
			execution.CompletedAt = time.Now()
			// Store failed execution
			s.mu.Lock()
			s.executions[execution.ID] = execution
			s.mu.Unlock()
			return execution, nil
		}

		if !resp.Success {
			stepResult.Status = ExecutionStatusFailed
			stepResult.Error = resp.Error
		} else {
			stepResult.Status = ExecutionStatusCompleted
			stepResult.Output = resp.Output
			stepOutputs[step.ID] = resp.Output
		}
		stepResult.CompletedAt = time.Now()
		execution.StepResults[step.ID] = stepResult

		if stepResult.Status == ExecutionStatusFailed {
			execution.Status = ExecutionStatusFailed
			execution.Error = fmt.Sprintf("step %s failed: %s", step.ID, stepResult.Error)
			execution.CompletedAt = time.Now()
			// Store failed execution
			s.mu.Lock()
			s.executions[execution.ID] = execution
			s.mu.Unlock()
			return execution, nil
		}
	}

	execution.Status = ExecutionStatusCompleted
	execution.CompletedAt = time.Now()

	// Store execution for history
	s.mu.Lock()
	s.executions[execution.ID] = execution
	s.mu.Unlock()

	s.logger.Info("Pipeline execution completed",
		"execution_id", execution.ID,
		"duration", execution.CompletedAt.Sub(execution.StartedAt),
	)

	return execution, nil
}

// Route routes a request to the appropriate service based on intent
func (s *Service) Route(ctx context.Context, intent string, input interface{}) (*Response, error) {
	// Simple intent-based routing
	var serviceType ServiceType

	switch intent {
	case "generate", "complete", "chat":
		serviceType = ServiceTypeTuring
	case "search", "retrieve", "query":
		serviceType = ServiceTypeHypatia
	case "analyze", "extract", "classify", "summarize":
		serviceType = ServiceTypeBabbage
	case "execute", "agent", "workflow":
		serviceType = ServiceTypeLeibniz
	default:
		return nil, fmt.Errorf("unknown intent: %s", intent)
	}

	req := &Request{
		ID:          fmt.Sprintf("route-%d", time.Now().UnixNano()),
		ServiceType: serviceType,
		Operation:   intent,
		Input:       input,
	}

	return s.Execute(ctx, req)
}

// HealthCheck returns the health status of downstream services
func (s *Service) HealthCheck(ctx context.Context) map[ServiceType]bool {
	result := make(map[ServiceType]bool)
	services := []ServiceType{ServiceTypeTuring, ServiceTypeHypatia, ServiceTypeLeibniz, ServiceTypeBabbage}

	for _, svc := range services {
		_, err := s.locator.Locate(ctx, string(svc))
		result[svc] = err == nil
	}

	return result
}

// ============================================================================
// Admin & Orchestration Methods
// ============================================================================

// GetSystemOverview returns a comprehensive system overview
func (s *Service) GetSystemOverview(ctx context.Context) (*admin.SystemOverview, error) {
	return s.admin.GetSystemOverview(ctx)
}

// GetServiceStatus returns the status of a specific service
func (s *Service) GetServiceStatus(ctx context.Context, serviceName string) (*admin.ServiceStatus, error) {
	return s.admin.GetServiceStatus(ctx, serviceName)
}

// ListAllServices returns all registered services with their status
func (s *Service) ListAllServices(ctx context.Context) ([]*admin.ServiceStatus, error) {
	return s.admin.ListServices(ctx)
}

// GetServiceConfig returns configuration for a service
func (s *Service) GetServiceConfig(serviceName string) (*admin.ServiceConfig, error) {
	return s.admin.GetServiceConfig(serviceName)
}

// UpdateServiceConfig updates configuration for a service
func (s *Service) UpdateServiceConfig(config *admin.ServiceConfig) error {
	return s.admin.UpdateServiceConfig(config)
}

// GetSystemMetrics returns current system metrics
func (s *Service) GetSystemMetrics() *admin.SystemMetrics {
	return s.admin.GetMetrics()
}

// GetRecentErrors returns recent error entries
func (s *Service) GetRecentErrors(limit int) []admin.ErrorEntry {
	return s.admin.GetErrors(limit)
}

// GetHealthSummary returns a brief health summary of all services
func (s *Service) GetHealthSummary(ctx context.Context) map[string]admin.HealthStatus {
	return s.admin.HealthSummary(ctx)
}

// ResetMetrics resets all metrics counters
func (s *Service) ResetMetrics() {
	s.admin.ResetMetrics()
}

// ListExecutions returns all pipeline executions
func (s *Service) ListExecutions() []*PipelineExecution {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]*PipelineExecution, 0, len(s.executions))
	for _, exec := range s.executions {
		result = append(result, exec)
	}
	return result
}

// GetExecution returns a specific pipeline execution by ID
func (s *Service) GetExecution(id string) (*PipelineExecution, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	exec, ok := s.executions[id]
	if !ok {
		return nil, fmt.Errorf("execution not found: %s", id)
	}
	return exec, nil
}

// DeletePipeline deletes a pipeline by ID
func (s *Service) DeletePipeline(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.pipelines[id]; !ok {
		return fmt.Errorf("pipeline not found: %s", id)
	}

	delete(s.pipelines, id)
	s.logger.Info("Pipeline deleted", "id", id)
	return nil
}

// GetPipelineStats returns statistics for a pipeline
func (s *Service) GetPipelineStats(pipelineID string) (map[string]interface{}, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	pipeline, ok := s.pipelines[pipelineID]
	if !ok {
		return nil, fmt.Errorf("pipeline not found: %s", pipelineID)
	}

	// Count executions for this pipeline
	total, succeeded, failed := 0, 0, 0
	var totalDuration time.Duration

	for _, exec := range s.executions {
		if exec.PipelineID == pipelineID {
			total++
			switch exec.Status {
			case ExecutionStatusCompleted:
				succeeded++
				totalDuration += exec.CompletedAt.Sub(exec.StartedAt)
			case ExecutionStatusFailed:
				failed++
			}
		}
	}

	avgDuration := time.Duration(0)
	if succeeded > 0 {
		avgDuration = totalDuration / time.Duration(succeeded)
	}

	return map[string]interface{}{
		"pipeline_id":      pipelineID,
		"pipeline_name":    pipeline.Name,
		"step_count":       len(pipeline.Steps),
		"total_executions": total,
		"succeeded":        succeeded,
		"failed":           failed,
		"success_rate":     float64(succeeded) / float64(max(total, 1)) * 100,
		"avg_duration":     avgDuration.String(),
	}, nil
}

// max returns the larger of two integers
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
