package server

import (
	"context"
	"fmt"
	"time"

	mdwerror "github.com/msto63/mDW/foundation/core/error"
	pb "github.com/msto63/mDW/api/gen/bayes"
	"github.com/msto63/mDW/internal/bayes/service"
	coreGrpc "github.com/msto63/mDW/pkg/core/grpc"
	"github.com/msto63/mDW/pkg/core/health"
	"github.com/msto63/mDW/pkg/core/logging"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// LogLevel enum for gRPC
type LogLevel int32

const (
	LogLevel_DEBUG   LogLevel = 0
	LogLevel_INFO    LogLevel = 1
	LogLevel_WARNING LogLevel = 2
	LogLevel_ERROR   LogLevel = 3
)

// LogRequest represents a log request
type LogRequest struct {
	Service   string
	Level     LogLevel
	Message   string
	RequestID string
	Metadata  map[string]string
}

// LogResponse represents a log response
type LogResponse struct {
	ID        string
	Timestamp int64
}

// QueryRequest represents a query request
type QueryRequest struct {
	Service   string
	Level     LogLevel
	StartTime int64
	EndTime   int64
	RequestID string
	Limit     int32
	Offset    int32
}

// LogEntry represents a log entry in responses
type LogEntry struct {
	ID        string
	Timestamp int64
	Service   string
	Level     LogLevel
	Message   string
	RequestID string
	Metadata  map[string]string
}

// QueryResponse represents a query response
type QueryResponse struct {
	Entries []*LogEntry
	Total   int32
}

// StatsResponse represents stats response
type StatsResponse struct {
	TotalEntries     int64
	EntriesByLevel   map[int32]int64
	EntriesByService map[string]int64
	LastEntryTime    int64
}

// Server is the Bayes gRPC server
type Server struct {
	pb.UnimplementedBayesServiceServer
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
	Service service.Config
}

// DefaultConfig returns default server configuration
func DefaultConfig() Config {
	return Config{
		Host:    "0.0.0.0",
		Port:    9001,
		Service: service.DefaultConfig(),
	}
}

// New creates a new Bayes server
func New(cfg Config) (*Server, error) {
	logger := logging.New("bayes-server")

	svc, err := service.NewService(cfg.Service)
	if err != nil {
		return nil, mdwerror.Wrap(err, "failed to create service").
			WithCode(mdwerror.CodeServiceInitialization).
			WithOperation("server.New")
	}

	grpcCfg := coreGrpc.DefaultServerConfig()
	grpcCfg.Host = cfg.Host
	grpcCfg.Port = cfg.Port

	grpcServer := coreGrpc.NewServer(grpcCfg)

	healthRegistry := health.NewRegistry("bayes", "1.0.0")
	healthRegistry.RegisterFunc("service", func(ctx context.Context) health.CheckResult {
		return health.CheckResult{
			Name:    "service",
			Status:  health.StatusHealthy,
			Message: "Bayes logging service is operational",
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
	pb.RegisterBayesServiceServer(grpcServer.GRPCServer(), server)

	return server, nil
}


// LogDirect implements the Log RPC for direct (non-gRPC) calls
func (s *Server) LogDirect(ctx context.Context, req *LogRequest) (*LogResponse, error) {
	if req.Service == "" {
		return nil, status.Error(codes.InvalidArgument, "service name is required")
	}
	if req.Message == "" {
		return nil, status.Error(codes.InvalidArgument, "message is required")
	}

	entry := &service.LogEntry{
		Service:   req.Service,
		Level:     convertLevel(req.Level),
		Message:   req.Message,
		RequestID: req.RequestID,
		Metadata:  convertMetadata(req.Metadata),
	}

	if err := s.service.Log(ctx, entry); err != nil {
		s.logger.Error("Failed to log entry", "error", err)
		return nil, status.Error(codes.Internal, "failed to log entry")
	}

	return &LogResponse{
		ID:        entry.ID,
		Timestamp: entry.Timestamp.Unix(),
	}, nil
}

// Query implements the Query RPC
func (s *Server) Query(ctx context.Context, req *QueryRequest) (*QueryResponse, error) {
	filter := service.LogFilter{
		Service:   req.Service,
		Level:     convertLevel(req.Level),
		RequestID: req.RequestID,
		Limit:     int(req.Limit),
		Offset:    int(req.Offset),
	}

	if req.StartTime > 0 {
		filter.StartTime = time.Unix(req.StartTime, 0)
	}
	if req.EndTime > 0 {
		filter.EndTime = time.Unix(req.EndTime, 0)
	}

	entries, err := s.service.Query(ctx, filter)
	if err != nil {
		s.logger.Error("Failed to query logs", "error", err)
		return nil, status.Error(codes.Internal, "failed to query logs")
	}

	resp := &QueryResponse{
		Entries: make([]*LogEntry, len(entries)),
		Total:   int32(len(entries)),
	}

	for i, e := range entries {
		resp.Entries[i] = &LogEntry{
			ID:        e.ID,
			Timestamp: e.Timestamp.Unix(),
			Service:   e.Service,
			Level:     reverseConvertLevel(e.Level),
			Message:   e.Message,
			RequestID: e.RequestID,
			Metadata:  reverseConvertMetadata(e.Metadata),
		}
	}

	return resp, nil
}

// GetStatsDirect implements the GetStats RPC for direct (non-gRPC) calls
func (s *Server) GetStatsDirect(ctx context.Context) (*StatsResponse, error) {
	stats, err := s.service.GetStats(ctx)
	if err != nil {
		s.logger.Error("Failed to get stats", "error", err)
		return nil, status.Error(codes.Internal, "failed to get stats")
	}

	resp := &StatsResponse{
		TotalEntries:     stats.TotalEntries,
		EntriesByLevel:   make(map[int32]int64),
		EntriesByService: stats.EntriesByService,
		LastEntryTime:    stats.LastEntry.Unix(),
	}

	for level, count := range stats.EntriesByLevel {
		resp.EntriesByLevel[int32(reverseConvertLevel(level))] = count
	}

	return resp, nil
}

// Stream implements the Stream RPC for real-time log streaming
func (s *Server) Stream(ctx context.Context, req *QueryRequest, send func(*LogEntry) error) error {
	filter := service.LogFilter{
		Service:   req.Service,
		Level:     convertLevel(req.Level),
		RequestID: req.RequestID,
	}

	ch, err := s.service.Stream(ctx, filter)
	if err != nil {
		return status.Error(codes.Internal, "failed to start stream")
	}

	for entry := range ch {
		if err := send(&LogEntry{
			ID:        entry.ID,
			Timestamp: entry.Timestamp.Unix(),
			Service:   entry.Service,
			Level:     reverseConvertLevel(entry.Level),
			Message:   entry.Message,
			RequestID: entry.RequestID,
			Metadata:  reverseConvertMetadata(entry.Metadata),
		}); err != nil {
			return err
		}
	}

	return nil
}

// Start starts the server
func (s *Server) Start() error {
	s.logger.Info("Starting Bayes server", "host", s.config.Host, "port", s.config.Port)
	return s.grpc.Start()
}

// StartAsync starts the server asynchronously
func (s *Server) StartAsync() error {
	s.logger.Info("Starting Bayes server (async)", "host", s.config.Host, "port", s.config.Port)
	return s.grpc.StartAsync()
}

// Stop stops the server
func (s *Server) Stop(ctx context.Context) error {
	s.logger.Info("Stopping Bayes server")
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

// Helper functions for type conversion

func convertLevel(level LogLevel) service.LogLevel {
	switch level {
	case LogLevel_DEBUG:
		return service.LogLevelDebug
	case LogLevel_INFO:
		return service.LogLevelInfo
	case LogLevel_WARNING:
		return service.LogLevelWarning
	case LogLevel_ERROR:
		return service.LogLevelError
	default:
		return service.LogLevelInfo
	}
}

func reverseConvertLevel(level service.LogLevel) LogLevel {
	switch level {
	case service.LogLevelDebug:
		return LogLevel_DEBUG
	case service.LogLevelInfo:
		return LogLevel_INFO
	case service.LogLevelWarning:
		return LogLevel_WARNING
	case service.LogLevelError:
		return LogLevel_ERROR
	default:
		return LogLevel_INFO
	}
}

func convertMetadata(m map[string]string) map[string]interface{} {
	if m == nil {
		return nil
	}
	result := make(map[string]interface{}, len(m))
	for k, v := range m {
		result[k] = v
	}
	return result
}

func reverseConvertMetadata(m map[string]interface{}) map[string]string {
	if m == nil {
		return nil
	}
	result := make(map[string]string, len(m))
	for k, v := range m {
		result[k] = fmt.Sprintf("%v", v)
	}
	return result
}
