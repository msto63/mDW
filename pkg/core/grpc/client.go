package grpc

import (
	"context"
	"fmt"
	"sync"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"
)

// ClientConfig holds gRPC client configuration
type ClientConfig struct {
	Target            string
	Timeout           time.Duration
	MaxRecvMsgSize    int
	MaxSendMsgSize    int
	KeepaliveInterval time.Duration
	KeepaliveTimeout  time.Duration
	Block             bool // Block until connection is established
}

// DefaultClientConfig returns a default client configuration
func DefaultClientConfig(target string) ClientConfig {
	return ClientConfig{
		Target:            target,
		Timeout:           30 * time.Second,
		MaxRecvMsgSize:    16 * 1024 * 1024, // 16MB
		MaxSendMsgSize:    16 * 1024 * 1024, // 16MB
		KeepaliveInterval: 30 * time.Second,
		KeepaliveTimeout:  10 * time.Second,
		Block:             false,
	}
}

// Dial creates a new gRPC client connection
func Dial(cfg ClientConfig, opts ...grpc.DialOption) (*grpc.ClientConn, error) {
	dialOpts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultCallOptions(
			grpc.MaxCallRecvMsgSize(cfg.MaxRecvMsgSize),
			grpc.MaxCallSendMsgSize(cfg.MaxSendMsgSize),
		),
		grpc.WithKeepaliveParams(keepalive.ClientParameters{
			Time:                cfg.KeepaliveInterval,
			Timeout:             cfg.KeepaliveTimeout,
			PermitWithoutStream: true,
		}),
		grpc.WithChainUnaryInterceptor(
			ClientRequestIDInterceptor(),
			ClientLoggingInterceptor(),
		),
		grpc.WithChainStreamInterceptor(
			ClientStreamLoggingInterceptor(),
		),
	}

	// Append custom options
	dialOpts = append(dialOpts, opts...)

	ctx, cancel := context.WithTimeout(context.Background(), cfg.Timeout)
	defer cancel()

	conn, err := grpc.DialContext(ctx, cfg.Target, dialOpts...)
	if err != nil {
		return nil, fmt.Errorf("failed to dial %s: %w", cfg.Target, err)
	}

	return conn, nil
}

// DialSimple creates a simple gRPC client connection with minimal configuration
func DialSimple(target string) (*grpc.ClientConn, error) {
	return Dial(DefaultClientConfig(target))
}

// DialWithTimeout creates a gRPC client connection with a custom timeout
func DialWithTimeout(target string, timeout time.Duration) (*grpc.ClientConn, error) {
	cfg := DefaultClientConfig(target)
	cfg.Timeout = timeout
	return Dial(cfg)
}

// ConnectionPool manages a pool of gRPC connections (thread-safe)
type ConnectionPool struct {
	mu          sync.RWMutex
	connections map[string]*grpc.ClientConn
	config      ClientConfig
}

// NewConnectionPool creates a new connection pool
func NewConnectionPool(cfg ClientConfig) *ConnectionPool {
	return &ConnectionPool{
		connections: make(map[string]*grpc.ClientConn),
		config:      cfg,
	}
}

// Get returns a connection to the target, creating one if necessary
// The connection is checked for health before returning
func (p *ConnectionPool) Get(target string) (*grpc.ClientConn, error) {
	p.mu.RLock()
	conn, exists := p.connections[target]
	p.mu.RUnlock()

	if exists && isConnectionHealthy(conn) {
		return conn, nil
	}

	// Need to create or recreate connection
	p.mu.Lock()
	defer p.mu.Unlock()

	// Double-check after acquiring write lock
	if conn, exists := p.connections[target]; exists {
		if isConnectionHealthy(conn) {
			return conn, nil
		}
		// Connection unhealthy, close and recreate
		conn.Close()
		delete(p.connections, target)
	}

	cfg := p.config
	cfg.Target = target
	newConn, err := Dial(cfg)
	if err != nil {
		return nil, err
	}

	p.connections[target] = newConn
	return newConn, nil
}

// isConnectionHealthy checks if the connection is in a usable state
func isConnectionHealthy(conn *grpc.ClientConn) bool {
	state := conn.GetState()
	return state == connectivity.Ready || state == connectivity.Idle
}

// GetStatus returns the connection status for all targets
func (p *ConnectionPool) GetStatus() map[string]string {
	p.mu.RLock()
	defer p.mu.RUnlock()

	status := make(map[string]string, len(p.connections))
	for target, conn := range p.connections {
		status[target] = conn.GetState().String()
	}
	return status
}

// Close closes all connections in the pool
func (p *ConnectionPool) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	var lastErr error
	for target, conn := range p.connections {
		if err := conn.Close(); err != nil {
			lastErr = fmt.Errorf("failed to close connection to %s: %w", target, err)
		}
		delete(p.connections, target)
	}
	return lastErr
}

// ServiceAddresses holds all service addresses for mDW
type ServiceAddresses struct {
	Russell string
	Turing  string
	Hypatia string
	Leibniz string
	Babbage string
	Bayes   string
}

// DefaultServiceAddresses returns the default local service addresses
func DefaultServiceAddresses() ServiceAddresses {
	return ServiceAddresses{
		Russell: "localhost:9100",
		Turing:  "localhost:9200",
		Hypatia: "localhost:9220",
		Leibniz: "localhost:9140",
		Babbage: "localhost:9150",
		Bayes:   "localhost:9120",
	}
}

// GlobalPool is a singleton connection pool for all services
var (
	globalPool     *ConnectionPool
	globalPoolOnce sync.Once
	globalAddrs    ServiceAddresses
)

// InitGlobalPool initializes the global connection pool
func InitGlobalPool(addrs ServiceAddresses) {
	globalPoolOnce.Do(func() {
		globalAddrs = addrs
		globalPool = NewConnectionPool(DefaultClientConfig(""))
	})
}

// GetGlobalPool returns the global connection pool, initializing if needed
func GetGlobalPool() *ConnectionPool {
	globalPoolOnce.Do(func() {
		globalAddrs = DefaultServiceAddresses()
		globalPool = NewConnectionPool(DefaultClientConfig(""))
	})
	return globalPool
}

// GetGlobalAddresses returns the global service addresses
func GetGlobalAddresses() ServiceAddresses {
	GetGlobalPool() // Ensure initialized
	return globalAddrs
}

// CloseGlobalPool closes the global connection pool
func CloseGlobalPool() error {
	if globalPool != nil {
		return globalPool.Close()
	}
	return nil
}
