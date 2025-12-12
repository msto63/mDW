package cmd

import (
	"os"
	"time"

	aristotelespb "github.com/msto63/mDW/api/gen/aristoteles"
	babbagepb "github.com/msto63/mDW/api/gen/babbage"
	bayespb "github.com/msto63/mDW/api/gen/bayes"
	hypatiapb "github.com/msto63/mDW/api/gen/hypatia"
	leibnizpb "github.com/msto63/mDW/api/gen/leibniz"
	platonpb "github.com/msto63/mDW/api/gen/platon"
	russellpb "github.com/msto63/mDW/api/gen/russell"
	turingpb "github.com/msto63/mDW/api/gen/turing"
	coreGrpc "github.com/msto63/mDW/pkg/core/grpc"
	"google.golang.org/grpc"
)

// ServiceAddresses holds the addresses for all gRPC services
type ServiceAddresses struct {
	Russell     string
	Bayes       string
	Platon      string
	Leibniz     string
	Babbage     string
	Aristoteles string
	Turing      string
	Hypatia     string
}

// DefaultServiceAddresses returns default service addresses based on port convention
func DefaultServiceAddresses() ServiceAddresses {
	return ServiceAddresses{
		Russell:     getEnvOrDefault("MDW_RUSSELL_ADDR", "localhost:9100"),
		Bayes:       getEnvOrDefault("MDW_BAYES_ADDR", "localhost:9120"),
		Platon:      getEnvOrDefault("MDW_PLATON_ADDR", "localhost:9130"),
		Leibniz:     getEnvOrDefault("MDW_LEIBNIZ_ADDR", "localhost:9140"),
		Babbage:     getEnvOrDefault("MDW_BABBAGE_ADDR", "localhost:9150"),
		Aristoteles: getEnvOrDefault("MDW_ARISTOTELES_ADDR", "localhost:9160"),
		Turing:      getEnvOrDefault("MDW_TURING_ADDR", "localhost:9200"),
		Hypatia:     getEnvOrDefault("MDW_HYPATIA_ADDR", "localhost:9220"),
	}
}

// ServiceClients holds gRPC clients for all services
type ServiceClients struct {
	Russell     russellpb.RussellServiceClient
	Bayes       bayespb.BayesServiceClient
	Platon      platonpb.PlatonServiceClient
	Leibniz     leibnizpb.LeibnizServiceClient
	Babbage     babbagepb.BabbageServiceClient
	Aristoteles aristotelespb.AristotelesServiceClient
	Turing      turingpb.TuringServiceClient
	Hypatia     hypatiapb.HypatiaServiceClient

	conns []*grpc.ClientConn
}

// NewServiceClients creates gRPC clients for all services using the global pool
func NewServiceClients(addrs ServiceAddresses) (*ServiceClients, error) {
	pool := coreGrpc.GetGlobalPool()
	clients := &ServiceClients{}

	// Connect to Russell (Service Discovery)
	russellConn, err := pool.Get(addrs.Russell)
	if err != nil {
		return nil, err
	}
	clients.Russell = russellpb.NewRussellServiceClient(russellConn)

	// Connect to Bayes (Logging)
	bayesConn, err := pool.Get(addrs.Bayes)
	if err != nil {
		return nil, err
	}
	clients.Bayes = bayespb.NewBayesServiceClient(bayesConn)

	// Connect to Platon (Pipeline Processing)
	platonConn, err := pool.Get(addrs.Platon)
	if err != nil {
		return nil, err
	}
	clients.Platon = platonpb.NewPlatonServiceClient(platonConn)

	// Connect to Leibniz (Agentic AI)
	leibnizConn, err := pool.Get(addrs.Leibniz)
	if err != nil {
		return nil, err
	}
	clients.Leibniz = leibnizpb.NewLeibnizServiceClient(leibnizConn)

	// Connect to Babbage (NLP)
	babbageConn, err := pool.Get(addrs.Babbage)
	if err != nil {
		return nil, err
	}
	clients.Babbage = babbagepb.NewBabbageServiceClient(babbageConn)

	// Connect to Aristoteles (Agentic Pipeline)
	aristotelesConn, err := pool.Get(addrs.Aristoteles)
	if err != nil {
		return nil, err
	}
	clients.Aristoteles = aristotelespb.NewAristotelesServiceClient(aristotelesConn)

	// Connect to Turing (LLM)
	turingConn, err := pool.Get(addrs.Turing)
	if err != nil {
		return nil, err
	}
	clients.Turing = turingpb.NewTuringServiceClient(turingConn)

	// Connect to Hypatia (RAG)
	hypatiaConn, err := pool.Get(addrs.Hypatia)
	if err != nil {
		return nil, err
	}
	clients.Hypatia = hypatiapb.NewHypatiaServiceClient(hypatiaConn)

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

// NewBayesClient creates a gRPC client for Bayes service using the global pool
func NewBayesClient(addr string) (bayespb.BayesServiceClient, *grpc.ClientConn, error) {
	pool := coreGrpc.GetGlobalPool()
	conn, err := pool.Get(addr)
	if err != nil {
		return nil, nil, err
	}
	return bayespb.NewBayesServiceClient(conn), conn, nil
}

// NewPlatonClient creates a gRPC client for Platon service using the global pool
func NewPlatonClient(addr string) (platonpb.PlatonServiceClient, *grpc.ClientConn, error) {
	pool := coreGrpc.GetGlobalPool()
	conn, err := pool.Get(addr)
	if err != nil {
		return nil, nil, err
	}
	return platonpb.NewPlatonServiceClient(conn), conn, nil
}

// NewAristotelesClient creates a gRPC client for Aristoteles service using the global pool
func NewAristotelesClient(addr string) (aristotelespb.AristotelesServiceClient, *grpc.ClientConn, error) {
	pool := coreGrpc.GetGlobalPool()
	conn, err := pool.Get(addr)
	if err != nil {
		return nil, nil, err
	}
	return aristotelespb.NewAristotelesServiceClient(conn), conn, nil
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

// ServicePorts returns the gRPC ports for all services (in port order)
var ServicePorts = struct {
	Kant        int
	Russell     int
	Bayes       int
	Platon      int
	Leibniz     int
	Babbage     int
	Aristoteles int
	Turing      int
	Hypatia     int
}{
	Kant:        8080,
	Russell:     9100,
	Bayes:       9120,
	Platon:      9130,
	Leibniz:     9140,
	Babbage:     9150,
	Aristoteles: 9160,
	Turing:      9200,
	Hypatia:     9220,
}

// gRPCTimeout is the default timeout for gRPC calls
const gRPCTimeout = 120 * time.Second
