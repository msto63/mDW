package grpc

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/msto63/mDW/pkg/core/logging"
	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/reflection"
)

var serverLogger = logging.New("grpc-server")

// ServerConfig holds gRPC server configuration
type ServerConfig struct {
	Host              string
	Port              int
	MaxRecvMsgSize    int
	MaxSendMsgSize    int
	EnableReflection  bool
	KeepaliveInterval time.Duration
	KeepaliveTimeout  time.Duration
}

// DefaultServerConfig returns a default server configuration
func DefaultServerConfig() ServerConfig {
	return ServerConfig{
		Host:              "0.0.0.0",
		Port:              9000,
		MaxRecvMsgSize:    16 * 1024 * 1024, // 16MB
		MaxSendMsgSize:    16 * 1024 * 1024, // 16MB
		EnableReflection:  true,
		KeepaliveInterval: 30 * time.Second,
		KeepaliveTimeout:  10 * time.Second,
	}
}

// Server wraps a gRPC server with additional functionality
type Server struct {
	server   *grpc.Server
	config   ServerConfig
	listener net.Listener
}

// NewServer creates a new gRPC server
func NewServer(cfg ServerConfig, opts ...grpc.ServerOption) *Server {
	// Build server options
	serverOpts := []grpc.ServerOption{
		grpc.MaxRecvMsgSize(cfg.MaxRecvMsgSize),
		grpc.MaxSendMsgSize(cfg.MaxSendMsgSize),
		grpc.KeepaliveParams(keepalive.ServerParameters{
			Time:    cfg.KeepaliveInterval,
			Timeout: cfg.KeepaliveTimeout,
		}),
		grpc.KeepaliveEnforcementPolicy(keepalive.EnforcementPolicy{
			MinTime:             5 * time.Second,
			PermitWithoutStream: true,
		}),
		grpc.ChainUnaryInterceptor(
			RecoveryInterceptor(),
			LoggingInterceptor(),
			RequestIDInterceptor(),
		),
		grpc.ChainStreamInterceptor(
			StreamRecoveryInterceptor(),
			StreamLoggingInterceptor(),
		),
	}

	// Append custom options
	serverOpts = append(serverOpts, opts...)

	server := grpc.NewServer(serverOpts...)

	// Enable reflection for debugging
	if cfg.EnableReflection {
		reflection.Register(server)
	}

	return &Server{
		server: server,
		config: cfg,
	}
}

// GRPCServer returns the underlying gRPC server for service registration
func (s *Server) GRPCServer() *grpc.Server {
	return s.server
}

// Start starts the gRPC server
func (s *Server) Start() error {
	addr := fmt.Sprintf("%s:%d", s.config.Host, s.config.Port)

	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", addr, err)
	}
	s.listener = listener

	return s.server.Serve(listener)
}

// StartAsync starts the gRPC server in a goroutine
func (s *Server) StartAsync() error {
	addr := fmt.Sprintf("%s:%d", s.config.Host, s.config.Port)

	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", addr, err)
	}
	s.listener = listener

	go func() {
		if err := s.server.Serve(listener); err != nil {
			// Log error but don't panic - server might be shutting down
			serverLogger.Error("gRPC server error", "error", err)
		}
	}()

	return nil
}

// Stop gracefully stops the gRPC server
func (s *Server) Stop() {
	s.server.GracefulStop()
}

// StopWithTimeout stops the server with a timeout
func (s *Server) StopWithTimeout(ctx context.Context) {
	done := make(chan struct{})
	go func() {
		s.server.GracefulStop()
		close(done)
	}()

	select {
	case <-done:
		return
	case <-ctx.Done():
		s.server.Stop()
	}
}

// Address returns the server address
func (s *Server) Address() string {
	if s.listener != nil {
		return s.listener.Addr().String()
	}
	return fmt.Sprintf("%s:%d", s.config.Host, s.config.Port)
}
