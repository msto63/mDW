package server

import (
	"context"
	"fmt"
	"time"

	mdwerror "github.com/msto63/mDW/foundation/core/error"
	pb "github.com/msto63/mDW/api/gen/hypatia"
	"github.com/msto63/mDW/internal/hypatia/service"
	"github.com/msto63/mDW/internal/hypatia/vectorstore"
	coreGrpc "github.com/msto63/mDW/pkg/core/grpc"
	"github.com/msto63/mDW/pkg/core/health"
	"github.com/msto63/mDW/pkg/core/logging"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Server is the Hypatia gRPC server
type Server struct {
	pb.UnimplementedHypatiaServiceServer
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
	ChunkSize      int
	ChunkOverlap   int
	DefaultTopK    int
	MinRelevance   float64
	VectorStoreType string
	VectorStorePath string
}

// DefaultConfig returns default server configuration
func DefaultConfig() Config {
	return Config{
		Host:           "0.0.0.0",
		Port:           9004,
		ChunkSize:      1000,
		ChunkOverlap:   200,
		DefaultTopK:    5,
		MinRelevance:   0.7,
		VectorStoreType: "memory",
		VectorStorePath: "./data/vectors",
	}
}

// New creates a new Hypatia server
func New(cfg Config) (*Server, error) {
	logger := logging.New("hypatia-server")

	// Create vector store
	var store vectorstore.Store
	var err error

	switch cfg.VectorStoreType {
	case "sqlite", "sqlite3", "sqlite-vec":
		store, err = vectorstore.NewSQLiteStore(vectorstore.SQLiteConfig{
			Path:       cfg.VectorStorePath + ".db",
			Dimensions: 768, // nomic-embed-text default
		})
		if err != nil {
			return nil, mdwerror.Wrap(err, "failed to create SQLite store").
				WithCode(mdwerror.CodeDatabaseError).
				WithOperation("server.New")
		}
		logger.Info("Using SQLite vector store", "path", cfg.VectorStorePath)
	case "memory", "":
		store = vectorstore.NewMemoryStore()
		logger.Info("Using in-memory vector store")
	default:
		logger.Warn("Unknown store type, using memory", "type", cfg.VectorStoreType)
		store = vectorstore.NewMemoryStore()
	}

	// Create service
	svcCfg := service.Config{
		ChunkSize:    cfg.ChunkSize,
		ChunkOverlap: cfg.ChunkOverlap,
		DefaultTopK:  cfg.DefaultTopK,
		MinRelevance: cfg.MinRelevance,
	}

	svc, err := service.NewService(svcCfg, store)
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
	healthRegistry := health.NewRegistry("hypatia", "1.0.0")
	healthRegistry.RegisterFunc("store", func(ctx context.Context) health.CheckResult {
		if err := svc.HealthCheck(ctx); err != nil {
			return health.CheckResult{
				Name:    "store",
				Status:  health.StatusUnhealthy,
				Message: fmt.Sprintf("Store not available: %v", err),
			}
		}
		return health.CheckResult{
			Name:    "store",
			Status:  health.StatusHealthy,
			Message: "Vector store is available",
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
	pb.RegisterHypatiaServiceServer(grpcServer.GRPCServer(), server)

	return server, nil
}

// SetEmbeddingFunc sets the embedding function
func (s *Server) SetEmbeddingFunc(fn service.EmbeddingFunc) {
	s.service.SetEmbeddingFunc(fn)
}

// Index indexes a document
func (s *Server) Index(ctx context.Context, id, content, collection string, metadata map[string]string) error {
	if id == "" {
		return status.Error(codes.InvalidArgument, "id is required")
	}
	if content == "" {
		return status.Error(codes.InvalidArgument, "content is required")
	}

	req := &service.IndexRequest{
		ID:         id,
		Content:    content,
		Collection: collection,
		Metadata:   metadata,
	}

	if err := s.service.Index(ctx, req); err != nil {
		s.logger.Error("Index failed", "error", err)
		return status.Error(codes.Internal, err.Error())
	}

	return nil
}

// SearchDirect performs semantic search directly (not via gRPC)
func (s *Server) SearchDirect(ctx context.Context, query, collection string, topK int, minScore float64) ([]service.SearchResult, error) {
	if query == "" {
		return nil, status.Error(codes.InvalidArgument, "query is required")
	}

	req := &service.SearchRequest{
		Query:      query,
		Collection: collection,
		TopK:       topK,
		MinScore:   minScore,
	}

	results, err := s.service.Search(ctx, req)
	if err != nil {
		s.logger.Error("Search failed", "error", err)
		return nil, status.Error(codes.Internal, err.Error())
	}

	return results, nil
}

// Delete deletes a document
func (s *Server) Delete(ctx context.Context, id string) error {
	if id == "" {
		return status.Error(codes.InvalidArgument, "id is required")
	}
	return s.service.Delete(ctx, id)
}

// ListCollectionsDirect lists all collections directly (not via gRPC)
func (s *Server) ListCollectionsDirect(ctx context.Context) ([]service.CollectionInfo, error) {
	return s.service.ListCollections(ctx)
}

// DeleteCollectionDirect deletes a collection directly (not via gRPC)
func (s *Server) DeleteCollectionDirect(ctx context.Context, collection string) error {
	return s.service.DeleteCollection(ctx, collection)
}

// Start starts the server
func (s *Server) Start() error {
	s.logger.Info("Starting Hypatia server", "host", s.config.Host, "port", s.config.Port)
	return s.grpc.Start()
}

// StartAsync starts the server asynchronously
func (s *Server) StartAsync() error {
	s.logger.Info("Starting Hypatia server (async)", "host", s.config.Host, "port", s.config.Port)
	return s.grpc.StartAsync()
}

// Stop stops the server
func (s *Server) Stop(ctx context.Context) error {
	s.logger.Info("Stopping Hypatia server")
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
