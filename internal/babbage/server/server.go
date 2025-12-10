package server

import (
	"context"
	"time"

	mdwerror "github.com/msto63/mDW/foundation/core/error"
	pb "github.com/msto63/mDW/api/gen/babbage"
	"github.com/msto63/mDW/internal/babbage/service"
	coreGrpc "github.com/msto63/mDW/pkg/core/grpc"
	"github.com/msto63/mDW/pkg/core/health"
	"github.com/msto63/mDW/pkg/core/logging"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Server is the Babbage gRPC server
type Server struct {
	pb.UnimplementedBabbageServiceServer
	service   *service.Service
	grpc      *coreGrpc.Server
	health    *health.Registry
	logger    *logging.Logger
	config    Config
	startTime time.Time
}

// Config holds server configuration
type Config struct {
	Host string
	Port int
}

// DefaultConfig returns default server configuration
func DefaultConfig() Config {
	return Config{
		Host: "0.0.0.0",
		Port: 9005,
	}
}

// New creates a new Babbage server
func New(cfg Config) (*Server, error) {
	logger := logging.New("babbage-server")

	// Create service
	svc, err := service.NewService(service.Config{})
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
	healthRegistry := health.NewRegistry("babbage", "1.0.0")
	healthRegistry.RegisterFunc("service", func(ctx context.Context) health.CheckResult {
		return health.CheckResult{
			Name:    "service",
			Status:  health.StatusHealthy,
			Message: "Babbage NLP service is operational",
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
	pb.RegisterBabbageServiceServer(grpcServer.GRPCServer(), server)

	return server, nil
}

// SetLLMFunc sets the LLM function for LLM-based operations
func (s *Server) SetLLMFunc(fn service.LLMFunc) {
	s.service.SetLLMFunc(fn)
}

// AnalyzeDirect performs text analysis directly (not via gRPC)
func (s *Server) AnalyzeDirect(ctx context.Context, text string) (*service.AnalysisResult, error) {
	if text == "" {
		return nil, status.Error(codes.InvalidArgument, "text is required")
	}

	result, err := s.service.Analyze(ctx, text)
	if err != nil {
		s.logger.Error("Analyze failed", "error", err)
		return nil, status.Error(codes.Internal, err.Error())
	}

	return result, nil
}

// SummarizeDirect generates a summary directly (not via gRPC)
func (s *Server) SummarizeDirect(ctx context.Context, text string, maxLength int, style string) (string, error) {
	if text == "" {
		return "", status.Error(codes.InvalidArgument, "text is required")
	}

	req := &service.SummarizeRequest{
		Text:      text,
		MaxLength: maxLength,
		Style:     style,
	}

	result, err := s.service.Summarize(ctx, req)
	if err != nil {
		s.logger.Error("Summarize failed", "error", err)
		return "", status.Error(codes.Internal, err.Error())
	}

	return result, nil
}

// ClassifyDirect classifies text directly (not via gRPC)
func (s *Server) ClassifyDirect(ctx context.Context, text string, labels []string) (*service.ClassifyResult, error) {
	if text == "" {
		return nil, status.Error(codes.InvalidArgument, "text is required")
	}
	if len(labels) == 0 {
		return nil, status.Error(codes.InvalidArgument, "labels are required")
	}

	req := &service.ClassifyRequest{
		Text:   text,
		Labels: labels,
	}

	result, err := s.service.Classify(ctx, req)
	if err != nil {
		s.logger.Error("Classify failed", "error", err)
		return nil, status.Error(codes.Internal, err.Error())
	}

	return result, nil
}

// ExtractKeywordsDirect extracts keywords from text directly (not via gRPC)
func (s *Server) ExtractKeywordsDirect(ctx context.Context, text string, maxKeywords int) ([]string, error) {
	if text == "" {
		return nil, status.Error(codes.InvalidArgument, "text is required")
	}

	return s.service.ExtractKeywords(ctx, text, maxKeywords)
}

// DetectLanguageDirect detects the language directly (not via gRPC)
func (s *Server) DetectLanguageDirect(ctx context.Context, text string) (string, error) {
	if text == "" {
		return "", status.Error(codes.InvalidArgument, "text is required")
	}

	return s.service.DetectLanguage(ctx, text)
}

// Start starts the server
func (s *Server) Start() error {
	s.logger.Info("Starting Babbage server", "host", s.config.Host, "port", s.config.Port)
	return s.grpc.Start()
}

// StartAsync starts the server asynchronously
func (s *Server) StartAsync() error {
	s.logger.Info("Starting Babbage server (async)", "host", s.config.Host, "port", s.config.Port)
	return s.grpc.StartAsync()
}

// Stop stops the server
func (s *Server) Stop(ctx context.Context) {
	s.logger.Info("Stopping Babbage server")
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
