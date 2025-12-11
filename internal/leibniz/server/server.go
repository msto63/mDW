package server

import (
	"context"
	"time"

	mdwerror "github.com/msto63/mDW/foundation/core/error"
	pb "github.com/msto63/mDW/api/gen/leibniz"
	"github.com/msto63/mDW/internal/leibniz/agent"
	"github.com/msto63/mDW/internal/leibniz/mcp"
	"github.com/msto63/mDW/internal/leibniz/service"
	coreGrpc "github.com/msto63/mDW/pkg/core/grpc"
	"github.com/msto63/mDW/pkg/core/health"
	"github.com/msto63/mDW/pkg/core/logging"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Server is the Leibniz gRPC server
type Server struct {
	pb.UnimplementedLeibnizServiceServer
	service   *service.Service
	grpc      *coreGrpc.Server
	health    *health.Registry
	logger    *logging.Logger
	config    Config
	startTime time.Time
}

// Config holds server configuration
type Config struct {
	Host       string
	Port       int
	MaxSteps   int
	Timeout    time.Duration
	MCPServers []MCPServerConfig
	// Platon integration
	PlatonHost    string
	PlatonPort    int
	EnablePlaton  bool
	PlatonTimeout time.Duration
	// Web Research Agent
	EnableWebResearchAgent bool
	SearXNGInstances       []string
	// YAML-based agent configuration
	AgentsDir       string // Directory for YAML agent definitions
	EnableHotReload bool   // Enable hot-reload of agent definitions
}

// MCPServerConfig holds MCP server configuration
type MCPServerConfig struct {
	Name    string
	Command string
	Args    []string
	Env     map[string]string
}

// DefaultConfig returns default server configuration
func DefaultConfig() Config {
	return Config{
		Host:                   "0.0.0.0",
		Port:                   9140,
		MaxSteps:               10,
		Timeout:                120 * time.Second,
		MCPServers:             []MCPServerConfig{},
		PlatonHost:             "localhost",
		PlatonPort:             9130,
		EnablePlaton:           true,
		PlatonTimeout:          30 * time.Second,
		EnableWebResearchAgent: true,              // Enable web-researcher agent by default
		SearXNGInstances:       []string{},
		AgentsDir:              "./configs/agents", // YAML agent definitions
		EnableHotReload:        true,               // Enable hot-reload by default
	}
}

// New creates a new Leibniz server
func New(cfg Config) (*Server, error) {
	logger := logging.New("leibniz-server")

	// Convert MCP configs
	mcpConfigs := make([]service.MCPServerConfig, len(cfg.MCPServers))
	for i, mcpCfg := range cfg.MCPServers {
		mcpConfigs[i] = service.MCPServerConfig{
			Name:    mcpCfg.Name,
			Command: mcpCfg.Command,
			Args:    mcpCfg.Args,
			Env:     mcpCfg.Env,
		}
	}

	// Create service config with Platon and Web Research settings
	svcCfg := service.Config{
		MaxSteps:               cfg.MaxSteps,
		MCPServers:             mcpConfigs,
		EnablePlaton:           cfg.EnablePlaton,
		PlatonHost:             cfg.PlatonHost,
		PlatonPort:             cfg.PlatonPort,
		PlatonTimeout:          cfg.PlatonTimeout,
		EnableWebResearchAgent: cfg.EnableWebResearchAgent,
		SearXNGInstances:       cfg.SearXNGInstances,
		AgentsDir:              cfg.AgentsDir,
		EnableHotReload:        cfg.EnableHotReload,
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
	healthRegistry := health.NewRegistry("leibniz", "1.0.0")
	healthRegistry.RegisterFunc("service", func(ctx context.Context) health.CheckResult {
		return health.CheckResult{
			Name:    "service",
			Status:  health.StatusHealthy,
			Message: "Leibniz agentic AI service is operational",
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
	pb.RegisterLeibnizServiceServer(grpcServer.GRPCServer(), server)

	return server, nil
}

// SetLLMFunc sets the LLM function
func (s *Server) SetLLMFunc(fn agent.LLMFunc) {
	s.service.SetLLMFunc(fn)
}

// SetModelAwareLLMFunc sets the model-aware LLM function
func (s *Server) SetModelAwareLLMFunc(fn agent.ModelAwareLLMFunc) {
	s.service.SetModelAwareLLMFunc(fn)
}

// ConnectMCPServer connects to an MCP server
func (s *Server) ConnectMCPServer(ctx context.Context, name, command string, args []string, env map[string]string) error {
	cfg := mcp.ServerConfig{
		Command: command,
		Args:    args,
		Env:     env,
	}
	return s.service.ConnectMCPServer(ctx, name, cfg)
}

// ExecuteDirect executes an agent task directly (not via gRPC)
func (s *Server) ExecuteDirect(ctx context.Context, task string, tools []string, maxSteps int, timeout time.Duration) (*service.ExecuteResponse, error) {
	if task == "" {
		return nil, status.Error(codes.InvalidArgument, "task is required")
	}

	if maxSteps <= 0 {
		maxSteps = s.config.MaxSteps
	}
	if timeout <= 0 {
		timeout = s.config.Timeout
	}

	req := &service.ExecuteRequest{
		Task:     task,
		Tools:    tools,
		MaxSteps: maxSteps,
		Timeout:  timeout,
	}

	resp, err := s.service.Execute(ctx, req)
	if err != nil {
		s.logger.Error("Execute failed", "error", err)
		// Don't return error for completed but failed executions
		if resp != nil {
			return resp, nil
		}
		return nil, status.Error(codes.Internal, err.Error())
	}

	return resp, nil
}

// ListToolsDirect lists available tools directly (not via gRPC)
func (s *Server) ListToolsDirect(ctx context.Context) ([]service.ToolInfo, error) {
	return s.service.ListTools(), nil
}

// Start starts the server
func (s *Server) Start() error {
	s.logger.Info("Starting Leibniz server", "host", s.config.Host, "port", s.config.Port)
	return s.grpc.Start()
}

// StartAsync starts the server asynchronously
func (s *Server) StartAsync() error {
	s.logger.Info("Starting Leibniz server (async)", "host", s.config.Host, "port", s.config.Port)
	return s.grpc.StartAsync()
}

// Stop stops the server
func (s *Server) Stop(ctx context.Context) error {
	s.logger.Info("Stopping Leibniz server")
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

// Agent returns the underlying agent for tool registration
func (s *Server) Agent() *agent.Agent {
	return s.service.Agent()
}
