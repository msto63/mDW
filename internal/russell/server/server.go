package server

import (
	"context"
	"fmt"
	"time"

	mdwerror "github.com/msto63/mDW/foundation/core/error"
	pb "github.com/msto63/mDW/api/gen/russell"
	"github.com/msto63/mDW/internal/russell/orchestrator"
	"github.com/msto63/mDW/internal/russell/procmgr"
	"github.com/msto63/mDW/internal/russell/service"
	"github.com/msto63/mDW/pkg/core/discovery"
	coreGrpc "github.com/msto63/mDW/pkg/core/grpc"
	"github.com/msto63/mDW/pkg/core/health"
	"github.com/msto63/mDW/pkg/core/logging"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// ExecuteRequest represents an execution request
type ExecuteRequest struct {
	ServiceType string
	Operation   string
	Input       string // JSON encoded
	Parameters  map[string]string
	Timeout     int64 // seconds
}

// ExecuteResponse represents an execution response
type ExecuteResponse struct {
	Success  bool
	Output   string // JSON encoded
	Error    string
	Duration int64 // milliseconds
}

// RouteRequest represents a routing request
type RouteRequest struct {
	Intent string
	Input  string // JSON encoded
}

// RouteResponse represents a routing response
type RouteResponse struct {
	Success     bool
	Output      string // JSON encoded
	Error       string
	ServiceUsed string
}

// PipelineRequest represents a pipeline execution request
type PipelineRequest struct {
	PipelineID string
	Input      string // JSON encoded
}

// PipelineResponse represents a pipeline execution response
type PipelineResponse struct {
	ExecutionID string
	Status      string
	Output      string // JSON encoded
	Error       string
	Duration    int64 // milliseconds
}

// HealthResponse represents service health status
type HealthResponse struct {
	Status   string
	Services map[string]bool
}

// Server is the Russell gRPC server
type Server struct {
	pb.UnimplementedRussellServiceServer
	service      *service.Service
	grpc         *coreGrpc.Server
	health       *health.Registry
	logger       *logging.Logger
	config       Config
	registry     *discovery.LocalRegistry
	procMgr      *procmgr.ProcessManager
	orchestrator *orchestrator.Orchestrator
	startTime    time.Time
}

// Config holds server configuration
type Config struct {
	Host               string
	Port               int
	CacheTTL           time.Duration
	BinaryPath         string // Path to mdw binary for process management
	ConfigPath         string // Path to config file
	ServicesConfigPath string // Path to services.toml for orchestrator (optional)
}

// DefaultConfig returns default server configuration
func DefaultConfig() Config {
	return Config{
		Host:               "0.0.0.0",
		Port:               9002,
		CacheTTL:           30 * time.Second,
		BinaryPath:         "./bin/mdw",
		ConfigPath:         "./configs/config.toml",
		ServicesConfigPath: "./configs/services.toml",
	}
}

// New creates a new Russell server
func New(cfg Config) (*Server, error) {
	logger := logging.New("russell-server")

	// Create local registry for development
	registry := discovery.NewLocalRegistry()

	// Create process manager
	procMgrCfg := procmgr.DefaultConfig()
	if cfg.BinaryPath != "" {
		procMgrCfg.BinaryPath = cfg.BinaryPath
	}
	if cfg.ConfigPath != "" {
		procMgrCfg.ConfigPath = cfg.ConfigPath
	}
	procManager := procmgr.New(procMgrCfg)

	// Try to create orchestrator if services.toml exists
	var orch *orchestrator.Orchestrator
	if cfg.ServicesConfigPath != "" {
		var err error
		orch, err = orchestrator.New(cfg.ServicesConfigPath)
		if err != nil {
			logger.Warn("Failed to initialize orchestrator, using procmgr fallback",
				"error", err,
				"configPath", cfg.ServicesConfigPath)
			// Don't fail - just use procMgr as fallback
		} else {
			logger.Info("Orchestrator initialized",
				"configPath", cfg.ServicesConfigPath)
		}
	}

	// Create service
	svcCfg := service.Config{
		DiscoveryClient: registry,
		CacheTTL:        cfg.CacheTTL,
	}

	svc, err := service.NewService(svcCfg)
	if err != nil {
		return nil, mdwerror.Wrap(err, "failed to create service").
			WithCode(mdwerror.CodeServiceInitialization).
			WithOperation("server.New")
	}

	// Create gRPC server
	grpcCfg := coreGrpc.DefaultServerConfig()
	grpcCfg.Host = cfg.Host
	grpcCfg.Port = cfg.Port

	grpcServer := coreGrpc.NewServer(grpcCfg)

	// Create health registry
	healthRegistry := health.NewRegistry("russell", "1.0.0")
	healthRegistry.RegisterFunc("service", func(ctx context.Context) health.CheckResult {
		return health.CheckResult{
			Name:    "service",
			Status:  health.StatusHealthy,
			Message: "Russell orchestration service is operational",
		}
	})

	server := &Server{
		service:      svc,
		grpc:         grpcServer,
		health:       healthRegistry,
		logger:       logger,
		config:       cfg,
		registry:     registry,
		procMgr:      procManager,
		orchestrator: orch,
		startTime:    time.Now(),
	}

	// Register gRPC service
	pb.RegisterRussellServiceServer(grpcServer.GRPCServer(), server)

	// Register built-in pipelines
	server.registerDefaultPipelines()

	return server, nil
}

// registerDefaultPipelines registers default processing pipelines
func (s *Server) registerDefaultPipelines() {
	// RAG Pipeline: Search -> Generate
	ragPipeline := &service.Pipeline{
		ID:          "rag-default",
		Name:        "RAG Pipeline",
		Description: "Retrieval-Augmented Generation pipeline",
		Steps: []service.PipelineStep{
			{
				ID:          "retrieve",
				ServiceType: service.ServiceTypeHypatia,
				Operation:   "search",
				Parameters:  map[string]interface{}{"top_k": 5},
			},
			{
				ID:          "generate",
				ServiceType: service.ServiceTypeTuring,
				Operation:   "generate",
				DependsOn:   []string{"retrieve"},
			},
		},
	}
	s.service.RegisterPipeline(ragPipeline)

	// Analysis Pipeline: NLP -> Summarize
	analysisPipeline := &service.Pipeline{
		ID:          "analysis-default",
		Name:        "Analysis Pipeline",
		Description: "Text analysis and summarization pipeline",
		Steps: []service.PipelineStep{
			{
				ID:          "analyze",
				ServiceType: service.ServiceTypeBabbage,
				Operation:   "analyze",
			},
			{
				ID:          "summarize",
				ServiceType: service.ServiceTypeTuring,
				Operation:   "summarize",
				DependsOn:   []string{"analyze"},
			},
		},
	}
	s.service.RegisterPipeline(analysisPipeline)
}

// Execute handles execute requests
func (s *Server) Execute(ctx context.Context, req *ExecuteRequest) (*ExecuteResponse, error) {
	if req.ServiceType == "" {
		return nil, status.Error(codes.InvalidArgument, "service_type is required")
	}
	if req.Operation == "" {
		return nil, status.Error(codes.InvalidArgument, "operation is required")
	}

	timeout := time.Duration(req.Timeout) * time.Second
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	svcReq := &service.Request{
		ID:          fmt.Sprintf("req-%d", time.Now().UnixNano()),
		ServiceType: service.ServiceType(req.ServiceType),
		Operation:   req.Operation,
		Input:       req.Input,
		Parameters:  convertParams(req.Parameters),
		Timeout:     timeout,
	}

	resp, err := s.service.Execute(ctx, svcReq)
	if err != nil {
		s.logger.Error("Execute failed", "error", err)
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &ExecuteResponse{
		Success:  resp.Success,
		Output:   fmt.Sprintf("%v", resp.Output),
		Error:    resp.Error,
		Duration: resp.Duration.Milliseconds(),
	}, nil
}

// Route handles route requests
func (s *Server) Route(ctx context.Context, req *RouteRequest) (*RouteResponse, error) {
	if req.Intent == "" {
		return nil, status.Error(codes.InvalidArgument, "intent is required")
	}

	resp, err := s.service.Route(ctx, req.Intent, req.Input)
	if err != nil {
		s.logger.Error("Route failed", "error", err)
		return nil, status.Error(codes.Internal, err.Error())
	}

	serviceUsed := ""
	if meta, ok := resp.Metadata["service_instance"].(string); ok {
		serviceUsed = meta
	}

	return &RouteResponse{
		Success:     resp.Success,
		Output:      fmt.Sprintf("%v", resp.Output),
		Error:       resp.Error,
		ServiceUsed: serviceUsed,
	}, nil
}

// ExecutePipelineInternal handles internal pipeline execution requests (non-gRPC)
func (s *Server) ExecutePipelineInternal(ctx context.Context, req *PipelineRequest) (*PipelineResponse, error) {
	if req.PipelineID == "" {
		return nil, status.Error(codes.InvalidArgument, "pipeline_id is required")
	}

	execution, err := s.service.ExecutePipeline(ctx, req.PipelineID, req.Input)
	if err != nil {
		s.logger.Error("Pipeline execution failed", "error", err)
		return nil, status.Error(codes.Internal, err.Error())
	}

	output := ""
	if execution.Status == service.ExecutionStatusCompleted {
		// Get output from last step
		for _, step := range execution.StepResults {
			if step.Output != nil {
				output = fmt.Sprintf("%v", step.Output)
			}
		}
	}

	return &PipelineResponse{
		ExecutionID: execution.ID,
		Status:      string(execution.Status),
		Output:      output,
		Error:       execution.Error,
		Duration:    execution.CompletedAt.Sub(execution.StartedAt).Milliseconds(),
	}, nil
}

// GetHealth returns the health status
func (s *Server) GetHealth(ctx context.Context) (*HealthResponse, error) {
	serviceHealth := s.service.HealthCheck(ctx)

	services := make(map[string]bool)
	for svc, healthy := range serviceHealth {
		services[string(svc)] = healthy
	}

	return &HealthResponse{
		Status:   "healthy",
		Services: services,
	}, nil
}

// RegisterService registers a service with the local discovery registry
func (s *Server) RegisterService(info *discovery.ServiceInfo) error {
	return s.registry.Register(context.Background(), info)
}

// Start starts the server
func (s *Server) Start() error {
	s.logger.Info("Starting Russell server", "host", s.config.Host, "port", s.config.Port)
	return s.grpc.Start()
}

// StartAsync starts the server asynchronously
func (s *Server) StartAsync() error {
	s.logger.Info("Starting Russell server (async)", "host", s.config.Host, "port", s.config.Port)
	return s.grpc.StartAsync()
}

// Stop stops the server
func (s *Server) Stop(ctx context.Context) {
	s.logger.Info("Stopping Russell server")
	s.grpc.StopWithTimeout(ctx)
}

// GRPCServer returns the underlying gRPC server
func (s *Server) GRPCServer() *grpc.Server {
	return s.grpc.GRPCServer()
}

// HealthRegistry returns the health check registry
func (s *Server) HealthRegistry() *health.Registry {
	return s.health
}

// Helper to convert string map to interface map
func convertParams(m map[string]string) map[string]interface{} {
	if m == nil {
		return nil
	}
	result := make(map[string]interface{}, len(m))
	for k, v := range m {
		result[k] = v
	}
	return result
}
