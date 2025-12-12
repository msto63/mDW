package server

import (
	"context"
	"fmt"
	"time"

	mdwerror "github.com/msto63/mDW/foundation/core/error"
	pb "github.com/msto63/mDW/api/gen/turing"
	"github.com/msto63/mDW/internal/turing/service"
	coreGrpc "github.com/msto63/mDW/pkg/core/grpc"
	"github.com/msto63/mDW/pkg/core/health"
	"github.com/msto63/mDW/pkg/core/logging"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Server is the Turing gRPC server
type Server struct {
	pb.UnimplementedTuringServiceServer
	service   *service.Service
	grpc      *coreGrpc.Server
	health    *health.Registry
	logger    *logging.Logger
	config    Config
	startTime time.Time
}

// Config holds server configuration
type Config struct {
	Host           string
	Port           int
	OllamaURL      string
	OllamaTimeout  time.Duration
	DefaultModel   string
	EmbeddingModel string
}

// DefaultConfig returns default server configuration
func DefaultConfig() Config {
	return Config{
		Host:           "0.0.0.0",
		Port:           9200,
		OllamaURL:      "http://localhost:11434",
		OllamaTimeout:  120 * time.Second,
		DefaultModel:   "mistral:7b",
		EmbeddingModel: "nomic-embed-text",
	}
}

// New creates a new Turing server
func New(cfg Config) (*Server, error) {
	logger := logging.New("turing-server")

	// Create service
	svcCfg := service.Config{
		OllamaURL:      cfg.OllamaURL,
		OllamaTimeout:  cfg.OllamaTimeout,
		DefaultModel:   cfg.DefaultModel,
		EmbeddingModel: cfg.EmbeddingModel,
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
	healthRegistry := health.NewRegistry("turing", "1.0.0")
	healthRegistry.RegisterFunc("ollama", func(ctx context.Context) health.CheckResult {
		if err := svc.HealthCheck(ctx); err != nil {
			return health.CheckResult{
				Name:    "ollama",
				Status:  health.StatusUnhealthy,
				Message: fmt.Sprintf("Ollama not available: %v", err),
			}
		}
		return health.CheckResult{
			Name:    "ollama",
			Status:  health.StatusHealthy,
			Message: "Ollama is available",
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
	pb.RegisterTuringServiceServer(grpcServer.GRPCServer(), server)

	return server, nil
}

// Generate handles generate requests
func (s *Server) Generate(ctx context.Context, prompt, system, model string, maxTokens int, temperature float64) (string, error) {
	if prompt == "" {
		return "", status.Error(codes.InvalidArgument, "prompt is required")
	}

	req := &service.GenerateRequest{
		Prompt:      prompt,
		System:      system,
		Model:       model,
		MaxTokens:   maxTokens,
		Temperature: temperature,
	}

	resp, err := s.service.Generate(ctx, req)
	if err != nil {
		s.logger.Error("Generate failed", "error", err)
		return "", status.Error(codes.Internal, err.Error())
	}

	return resp.Text, nil
}

// ChatDirect handles chat requests directly (not via gRPC)
func (s *Server) ChatDirect(ctx context.Context, messages []service.Message, model string, maxTokens int, temperature float64) (*service.ChatResponse, error) {
	if len(messages) == 0 {
		return nil, status.Error(codes.InvalidArgument, "messages are required")
	}

	req := &service.ChatRequest{
		Messages:    messages,
		Model:       model,
		MaxTokens:   maxTokens,
		Temperature: temperature,
	}

	resp, err := s.service.Chat(ctx, req)
	if err != nil {
		s.logger.Error("Chat failed", "error", err)
		return nil, status.Error(codes.Internal, err.Error())
	}

	return resp, nil
}

// EmbedDirect handles embedding requests directly (not via gRPC)
func (s *Server) EmbedDirect(ctx context.Context, input []string, model string) ([][]float64, error) {
	if len(input) == 0 {
		return nil, status.Error(codes.InvalidArgument, "input is required")
	}

	req := &service.EmbeddingRequest{
		Input: input,
		Model: model,
	}

	resp, err := s.service.Embed(ctx, req)
	if err != nil {
		s.logger.Error("Embed failed", "error", err)
		return nil, status.Error(codes.Internal, err.Error())
	}

	return resp.Embeddings, nil
}

// ListModelsDirect lists available models directly (not via gRPC)
func (s *Server) ListModelsDirect(ctx context.Context) ([]service.ModelInfo, error) {
	return s.service.ListModels(ctx)
}

// Summarize generates a summary
func (s *Server) Summarize(ctx context.Context, text string, maxLength int) (string, error) {
	if text == "" {
		return "", status.Error(codes.InvalidArgument, "text is required")
	}

	return s.service.Summarize(ctx, text, maxLength)
}

// Start starts the server
func (s *Server) Start() error {
	s.logger.Info("Starting Turing server", "host", s.config.Host, "port", s.config.Port)
	return s.grpc.Start()
}

// StartAsync starts the server asynchronously
func (s *Server) StartAsync() error {
	s.logger.Info("Starting Turing server (async)", "host", s.config.Host, "port", s.config.Port)
	return s.grpc.StartAsync()
}

// Stop stops the server
func (s *Server) Stop(ctx context.Context) {
	s.logger.Info("Stopping Turing server")
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

// Service returns the underlying service (for direct access)
func (s *Server) Service() *service.Service {
	return s.service
}
