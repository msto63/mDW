// Package server provides the gRPC server for the Aristoteles service
package server

import (
	"context"
	"fmt"
	"time"

	"github.com/msto63/mDW/api/gen/common"
	pb "github.com/msto63/mDW/api/gen/aristoteles"
	"github.com/msto63/mDW/internal/aristoteles"
	"github.com/msto63/mDW/internal/aristoteles/clients"
	"github.com/msto63/mDW/internal/aristoteles/service"
	coreGrpc "github.com/msto63/mDW/pkg/core/grpc"
	"github.com/msto63/mDW/pkg/core/health"
	"github.com/msto63/mDW/pkg/core/logging"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Server is the Aristoteles gRPC server
type Server struct {
	pb.UnimplementedAristotelesServiceServer
	service   *service.Service
	grpc      *coreGrpc.Server
	health    *health.Registry
	logger    *logging.Logger
	config    Config
	startTime time.Time
}

// Config holds server configuration
type Config struct {
	Host        string
	Port        int
	HTTPPort    int
	TuringAddr  string
	LeibnizAddr string
	HypatiaAddr string
	BabbageAddr string
	PlatonAddr  string
}

// DefaultConfig returns default server configuration
func DefaultConfig() Config {
	return Config{
		Host:        "0.0.0.0",
		Port:        9160,
		HTTPPort:    9161,
		TuringAddr:  "localhost:9200",
		LeibnizAddr: "localhost:9140",
		HypatiaAddr: "localhost:9220",
		BabbageAddr: "localhost:9150",
		PlatonAddr:  "localhost:9130",
	}
}

// New creates a new Aristoteles server
func New(cfg Config) (*Server, error) {
	logger := logging.New("aristoteles-server")

	// Create service clients
	clientsCfg := &clients.Config{
		TuringAddr:  cfg.TuringAddr,
		LeibnizAddr: cfg.LeibnizAddr,
		HypatiaAddr: cfg.HypatiaAddr,
		BabbageAddr: cfg.BabbageAddr,
		PlatonAddr:  cfg.PlatonAddr,
	}
	serviceClients, err := clients.NewServiceClients(clientsCfg)
	if err != nil {
		logger.Warn("Some service connections failed", "error", err)
	}

	// Create service
	svcCfg := service.DefaultConfig()
	svc := service.NewService(svcCfg, serviceClients)

	// Create gRPC server
	grpcCfg := coreGrpc.DefaultServerConfig()
	grpcCfg.Host = cfg.Host
	grpcCfg.Port = cfg.Port
	grpcServer := coreGrpc.NewServer(grpcCfg)

	// Create health registry
	healthRegistry := health.NewRegistry("aristoteles", aristoteles.Version)
	healthRegistry.RegisterFunc("service", func(ctx context.Context) health.CheckResult {
		stats := svc.Stats()
		return health.CheckResult{
			Name:    "service",
			Status:  health.StatusHealthy,
			Message: "Aristoteles pipeline service is operational",
			Details: stats,
		}
	})

	server := &Server{
		service:   svc,
		grpc:      grpcServer,
		health:    healthRegistry,
		logger:    logger,
		config:    cfg,
		startTime: time.Now(),
	}

	// Register gRPC service
	pb.RegisterAristotelesServiceServer(grpcServer.GRPCServer(), server)

	return server, nil
}

// ============================================================================
// gRPC Processing Methods
// ============================================================================

// Process executes the full pipeline
func (s *Server) Process(ctx context.Context, req *pb.ProcessRequest) (*pb.ProcessResponse, error) {
	s.logger.Debug("Processing request",
		"request_id", req.RequestId)

	resp, err := s.service.Process(ctx, req)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "processing failed: %v", err)
	}

	return resp, nil
}

// StreamProcess executes the pipeline with streaming output
func (s *Server) StreamProcess(req *pb.ProcessRequest, stream pb.AristotelesService_StreamProcessServer) error {
	s.logger.Debug("Starting stream processing",
		"request_id", req.RequestId)

	chunkCh := make(chan *pb.ProcessChunk, 10)

	// Run pipeline in goroutine
	errCh := make(chan error, 1)
	go func() {
		errCh <- s.service.StreamProcess(stream.Context(), req, chunkCh)
	}()

	// Stream chunks to client
	for chunk := range chunkCh {
		if err := stream.Send(chunk); err != nil {
			return status.Errorf(codes.Internal, "failed to send chunk: %v", err)
		}
	}

	// Check for pipeline errors
	if err := <-errCh; err != nil {
		return status.Errorf(codes.Internal, "pipeline failed: %v", err)
	}

	return nil
}

// ============================================================================
// Intent Analysis
// ============================================================================

// AnalyzeIntent analyzes a prompt for intent
func (s *Server) AnalyzeIntent(ctx context.Context, req *pb.IntentRequest) (*pb.IntentResponse, error) {
	s.logger.Debug("Analyzing intent",
		"prompt_length", len(req.Prompt))

	resp, err := s.service.AnalyzeIntent(ctx, req)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "intent analysis failed: %v", err)
	}

	return resp, nil
}

// ============================================================================
// Pipeline Management
// ============================================================================

// GetPipelineStatus returns the status of an active pipeline
func (s *Server) GetPipelineStatus(ctx context.Context, req *pb.PipelineStatusRequest) (*pb.PipelineStatusResponse, error) {
	resp, found := s.service.GetPipelineStatus(req.RequestId)
	if !found {
		return nil, status.Errorf(codes.NotFound, "pipeline not found: %s", req.RequestId)
	}
	return resp, nil
}

// CancelPipeline cancels an active pipeline
func (s *Server) CancelPipeline(ctx context.Context, req *pb.CancelPipelineRequest) (*common.Empty, error) {
	if !s.service.CancelPipeline(req.RequestId) {
		return nil, status.Errorf(codes.NotFound, "pipeline not found: %s", req.RequestId)
	}
	return &common.Empty{}, nil
}

// ============================================================================
// Configuration
// ============================================================================

// GetConfig returns the current configuration
func (s *Server) GetConfig(ctx context.Context, _ *common.Empty) (*pb.ConfigResponse, error) {
	return s.service.GetConfig(), nil
}

// UpdateConfig updates the configuration
func (s *Server) UpdateConfig(ctx context.Context, req *pb.UpdateConfigRequest) (*pb.ConfigResponse, error) {
	return s.service.UpdateConfig(req), nil
}

// ============================================================================
// Strategy Management
// ============================================================================

// ListStrategies returns all available strategies
func (s *Server) ListStrategies(ctx context.Context, _ *common.Empty) (*pb.StrategyListResponse, error) {
	strategies := s.service.ListStrategies()
	return &pb.StrategyListResponse{
		Strategies: strategies,
		Total:      int32(len(strategies)),
	}, nil
}

// GetStrategy returns a strategy by ID
func (s *Server) GetStrategy(ctx context.Context, req *pb.GetStrategyRequest) (*pb.StrategyInfo, error) {
	strategy, found := s.service.GetStrategy(req.StrategyId)
	if !found {
		return nil, status.Errorf(codes.NotFound, "strategy not found: %s", req.StrategyId)
	}
	return strategy, nil
}

// ============================================================================
// Health Check
// ============================================================================

// HealthCheck performs a health check
func (s *Server) HealthCheck(ctx context.Context, req *common.HealthCheckRequest) (*common.HealthCheckResponse, error) {
	report := s.health.Check(ctx)

	details := make(map[string]string)
	for _, c := range report.Checks {
		details[c.Name] = fmt.Sprintf("%s: %s", c.Status, c.Message)
	}
	details["uptime"] = report.Uptime.String()

	return &common.HealthCheckResponse{
		Status:        string(report.Status),
		Service:       report.Service,
		Version:       report.Version,
		UptimeSeconds: int64(report.Uptime.Seconds()),
		Details:       details,
	}, nil
}

// ============================================================================
// Server Lifecycle
// ============================================================================

// Start starts the server
func (s *Server) Start() error {
	s.logger.Info("Starting Aristoteles server",
		"host", s.config.Host,
		"port", s.config.Port)
	return s.grpc.Start()
}

// StartAsync starts the server asynchronously
func (s *Server) StartAsync() error {
	s.logger.Info("Starting Aristoteles server (async)",
		"host", s.config.Host,
		"port", s.config.Port)
	return s.grpc.StartAsync()
}

// Stop stops the server
func (s *Server) Stop(ctx context.Context) error {
	s.logger.Info("Stopping Aristoteles server")
	s.grpc.StopWithTimeout(ctx)
	return s.service.Close()
}

// GRPCServer returns the underlying gRPC server
func (s *Server) GRPCServer() *grpc.Server {
	return s.grpc.GRPCServer()
}

// HealthRegistry returns the health check registry
func (s *Server) HealthRegistry() *health.Registry {
	return s.health
}

// Service returns the underlying service
func (s *Server) Service() *service.Service {
	return s.service
}

// Stats returns server statistics
func (s *Server) Stats() map[string]interface{} {
	stats := s.service.Stats()
	stats["uptime"] = time.Since(s.startTime).String()
	stats["host"] = s.config.Host
	stats["port"] = s.config.Port
	return stats
}
