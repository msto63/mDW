package cmd

import (
	"os"
	"time"

	babbagepb "github.com/msto63/mDW/api/gen/babbage"
	hypatiapb "github.com/msto63/mDW/api/gen/hypatia"
	leibnizpb "github.com/msto63/mDW/api/gen/leibniz"
	russellpb "github.com/msto63/mDW/api/gen/russell"
	turingpb "github.com/msto63/mDW/api/gen/turing"
	coreGrpc "github.com/msto63/mDW/pkg/core/grpc"
	"google.golang.org/grpc"
)

// ServiceAddresses holds the addresses for all gRPC services
type ServiceAddresses struct {
	Russell string
	Turing  string
	Hypatia string
	Leibniz string
	Babbage string
}

// DefaultServiceAddresses returns default service addresses based on port convention
func DefaultServiceAddresses() ServiceAddresses {
	return ServiceAddresses{
		Russell: getEnvOrDefault("MDW_RUSSELL_ADDR", "localhost:9100"),
		Turing:  getEnvOrDefault("MDW_TURING_ADDR", "localhost:9200"),
		Hypatia: getEnvOrDefault("MDW_HYPATIA_ADDR", "localhost:9220"),
		Leibniz: getEnvOrDefault("MDW_LEIBNIZ_ADDR", "localhost:9140"),
		Babbage: getEnvOrDefault("MDW_BABBAGE_ADDR", "localhost:9150"),
	}
}

// ServiceClients holds gRPC clients for all services
type ServiceClients struct {
	Russell russellpb.RussellServiceClient
	Turing  turingpb.TuringServiceClient
	Hypatia hypatiapb.HypatiaServiceClient
	Leibniz leibnizpb.LeibnizServiceClient
	Babbage babbagepb.BabbageServiceClient

	conns []*grpc.ClientConn
}

// NewServiceClients creates gRPC clients for all services using the global pool
func NewServiceClients(addrs ServiceAddresses) (*ServiceClients, error) {
	pool := coreGrpc.GetGlobalPool()
	clients := &ServiceClients{}

	// Connect to Russell
	russellConn, err := pool.Get(addrs.Russell)
	if err != nil {
		return nil, err
	}
	clients.Russell = russellpb.NewRussellServiceClient(russellConn)

	// Connect to Turing
	turingConn, err := pool.Get(addrs.Turing)
	if err != nil {
		return nil, err
	}
	clients.Turing = turingpb.NewTuringServiceClient(turingConn)

	// Connect to Hypatia
	hypatiaConn, err := pool.Get(addrs.Hypatia)
	if err != nil {
		return nil, err
	}
	clients.Hypatia = hypatiapb.NewHypatiaServiceClient(hypatiaConn)

	// Connect to Leibniz
	leibnizConn, err := pool.Get(addrs.Leibniz)
	if err != nil {
		return nil, err
	}
	clients.Leibniz = leibnizpb.NewLeibnizServiceClient(leibnizConn)

	// Connect to Babbage
	babbageConn, err := pool.Get(addrs.Babbage)
	if err != nil {
		return nil, err
	}
	clients.Babbage = babbagepb.NewBabbageServiceClient(babbageConn)

	return clients, nil
}

// NewTuringClient creates a gRPC client for Turing service using the global pool
func NewTuringClient(addr string) (turingpb.TuringServiceClient, *grpc.ClientConn, error) {
	pool := coreGrpc.GetGlobalPool()
	conn, err := pool.Get(addr)
	if err != nil {
		return nil, nil, err
	}
	return turingpb.NewTuringServiceClient(conn), conn, nil
}

// NewHypatiaClient creates a gRPC client for Hypatia service using the global pool
func NewHypatiaClient(addr string) (hypatiapb.HypatiaServiceClient, *grpc.ClientConn, error) {
	pool := coreGrpc.GetGlobalPool()
	conn, err := pool.Get(addr)
	if err != nil {
		return nil, nil, err
	}
	return hypatiapb.NewHypatiaServiceClient(conn), conn, nil
}

// NewBabbageClient creates a gRPC client for Babbage service using the global pool
func NewBabbageClient(addr string) (babbagepb.BabbageServiceClient, *grpc.ClientConn, error) {
	pool := coreGrpc.GetGlobalPool()
	conn, err := pool.Get(addr)
	if err != nil {
		return nil, nil, err
	}
	return babbagepb.NewBabbageServiceClient(conn), conn, nil
}

// NewLeibnizClient creates a gRPC client for Leibniz service using the global pool
func NewLeibnizClient(addr string) (leibnizpb.LeibnizServiceClient, *grpc.ClientConn, error) {
	pool := coreGrpc.GetGlobalPool()
	conn, err := pool.Get(addr)
	if err != nil {
		return nil, nil, err
	}
	return leibnizpb.NewLeibnizServiceClient(conn), conn, nil
}

// NewRussellClient creates a gRPC client for Russell service using the global pool
func NewRussellClient(addr string) (russellpb.RussellServiceClient, *grpc.ClientConn, error) {
	pool := coreGrpc.GetGlobalPool()
	conn, err := pool.Get(addr)
	if err != nil {
		return nil, nil, err
	}
	return russellpb.NewRussellServiceClient(conn), conn, nil
}

// Close is now a no-op since connections are managed by the global pool
// Keeping for backwards compatibility
func (c *ServiceClients) Close() error {
	// Connections are managed by the global pool, don't close individually
	return nil
}

// ClosePool closes the global connection pool (call on program exit)
func ClosePool() error {
	return coreGrpc.CloseGlobalPool()
}

// getEnvOrDefault returns environment variable value or default
func getEnvOrDefault(key, defaultValue string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultValue
}

// ServicePorts returns the gRPC ports for all services
var ServicePorts = struct {
	Kant    int
	Russell int
	Turing  int
	Hypatia int
	Leibniz int
	Babbage int
	Bayes   int
}{
	Kant:    8080,
	Russell: 9100,
	Turing:  9200,
	Hypatia: 9220,
	Leibniz: 9140,
	Babbage: 9150,
	Bayes:   9120,
}

// gRPCTimeout is the default timeout for gRPC calls
const gRPCTimeout = 120 * time.Second
