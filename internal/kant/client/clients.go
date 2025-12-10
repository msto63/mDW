package client

import (
	"context"
	"fmt"
	"sync"
	"time"

	babbagepb "github.com/msto63/mDW/api/gen/babbage"
	hypatiapb "github.com/msto63/mDW/api/gen/hypatia"
	leibnizpb "github.com/msto63/mDW/api/gen/leibniz"
	platonpb "github.com/msto63/mDW/api/gen/platon"
	russellpb "github.com/msto63/mDW/api/gen/russell"
	turingpb "github.com/msto63/mDW/api/gen/turing"
	"github.com/msto63/mDW/pkg/core/logging"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// ServiceClients manages gRPC client connections to all services
type ServiceClients struct {
	mu     sync.RWMutex
	logger *logging.Logger

	// Service addresses
	russellAddr string
	turingAddr  string
	hypatiaAddr string
	leibnizAddr string
	babbageAddr string
	platonAddr  string

	// gRPC connections
	russellConn *grpc.ClientConn
	turingConn  *grpc.ClientConn
	hypatiaConn *grpc.ClientConn
	leibnizConn *grpc.ClientConn
	babbageConn *grpc.ClientConn
	platonConn  *grpc.ClientConn

	// Service clients
	Russell russellpb.RussellServiceClient
	Turing  turingpb.TuringServiceClient
	Hypatia hypatiapb.HypatiaServiceClient
	Leibniz leibnizpb.LeibnizServiceClient
	Babbage babbagepb.BabbageServiceClient
	Platon  platonpb.PlatonServiceClient
}

// Config holds client configuration
type Config struct {
	RussellAddr string
	TuringAddr  string
	HypatiaAddr string
	LeibnizAddr string
	BabbageAddr string
	PlatonAddr  string
}

// DefaultConfig returns default client configuration
func DefaultConfig() Config {
	return Config{
		RussellAddr: "localhost:9100",
		TuringAddr:  "localhost:9200",
		HypatiaAddr: "localhost:9220",
		LeibnizAddr: "localhost:9140",
		BabbageAddr: "localhost:9150",
		PlatonAddr:  "localhost:9130",
	}
}

// NewServiceClients creates a new service client manager
func NewServiceClients(cfg Config) *ServiceClients {
	return &ServiceClients{
		logger:      logging.New("kant-clients"),
		russellAddr: cfg.RussellAddr,
		turingAddr:  cfg.TuringAddr,
		hypatiaAddr: cfg.HypatiaAddr,
		leibnizAddr: cfg.LeibnizAddr,
		babbageAddr: cfg.BabbageAddr,
		platonAddr:  cfg.PlatonAddr,
	}
}

// Connect establishes connections to all services
func (c *ServiceClients) Connect(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	opts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	}

	// Use a timeout context for connection attempts
	connectCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	var err error

	// Connect to Russell (Service Discovery)
	c.logger.Info("Connecting to Russell", "addr", c.russellAddr)
	c.russellConn, err = grpc.DialContext(connectCtx, c.russellAddr, opts...)
	if err != nil {
		c.logger.Warn("Failed to connect to Russell", "error", err)
	} else {
		c.Russell = russellpb.NewRussellServiceClient(c.russellConn)
	}

	// Connect to Turing (LLM)
	c.logger.Info("Connecting to Turing", "addr", c.turingAddr)
	c.turingConn, err = grpc.DialContext(connectCtx, c.turingAddr, opts...)
	if err != nil {
		c.logger.Warn("Failed to connect to Turing", "error", err)
	} else {
		c.Turing = turingpb.NewTuringServiceClient(c.turingConn)
	}

	// Connect to Hypatia (RAG)
	c.logger.Info("Connecting to Hypatia", "addr", c.hypatiaAddr)
	c.hypatiaConn, err = grpc.DialContext(connectCtx, c.hypatiaAddr, opts...)
	if err != nil {
		c.logger.Warn("Failed to connect to Hypatia", "error", err)
	} else {
		c.Hypatia = hypatiapb.NewHypatiaServiceClient(c.hypatiaConn)
	}

	// Connect to Leibniz (Agent)
	c.logger.Info("Connecting to Leibniz", "addr", c.leibnizAddr)
	c.leibnizConn, err = grpc.DialContext(connectCtx, c.leibnizAddr, opts...)
	if err != nil {
		c.logger.Warn("Failed to connect to Leibniz", "error", err)
	} else {
		c.Leibniz = leibnizpb.NewLeibnizServiceClient(c.leibnizConn)
	}

	// Connect to Babbage (NLP)
	c.logger.Info("Connecting to Babbage", "addr", c.babbageAddr)
	c.babbageConn, err = grpc.DialContext(connectCtx, c.babbageAddr, opts...)
	if err != nil {
		c.logger.Warn("Failed to connect to Babbage", "error", err)
	} else {
		c.Babbage = babbagepb.NewBabbageServiceClient(c.babbageConn)
	}

	// Connect to Platon (Pipeline)
	c.logger.Info("Connecting to Platon", "addr", c.platonAddr)
	c.platonConn, err = grpc.DialContext(connectCtx, c.platonAddr, opts...)
	if err != nil {
		c.logger.Warn("Failed to connect to Platon", "error", err)
	} else {
		c.Platon = platonpb.NewPlatonServiceClient(c.platonConn)
	}

	c.logger.Info("Service client connections initialized")
	return nil
}

// ConnectLazy establishes connections lazily (non-blocking)
func (c *ServiceClients) ConnectLazy() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	opts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	}

	var err error

	// Connect to Russell
	c.russellConn, err = grpc.Dial(c.russellAddr, opts...)
	if err != nil {
		return fmt.Errorf("failed to dial russell: %w", err)
	}
	c.Russell = russellpb.NewRussellServiceClient(c.russellConn)

	// Connect to Turing
	c.turingConn, err = grpc.Dial(c.turingAddr, opts...)
	if err != nil {
		return fmt.Errorf("failed to dial turing: %w", err)
	}
	c.Turing = turingpb.NewTuringServiceClient(c.turingConn)

	// Connect to Hypatia
	c.hypatiaConn, err = grpc.Dial(c.hypatiaAddr, opts...)
	if err != nil {
		return fmt.Errorf("failed to dial hypatia: %w", err)
	}
	c.Hypatia = hypatiapb.NewHypatiaServiceClient(c.hypatiaConn)

	// Connect to Leibniz
	c.leibnizConn, err = grpc.Dial(c.leibnizAddr, opts...)
	if err != nil {
		return fmt.Errorf("failed to dial leibniz: %w", err)
	}
	c.Leibniz = leibnizpb.NewLeibnizServiceClient(c.leibnizConn)

	// Connect to Babbage
	c.babbageConn, err = grpc.Dial(c.babbageAddr, opts...)
	if err != nil {
		return fmt.Errorf("failed to dial babbage: %w", err)
	}
	c.Babbage = babbagepb.NewBabbageServiceClient(c.babbageConn)

	// Connect to Platon
	c.platonConn, err = grpc.Dial(c.platonAddr, opts...)
	if err != nil {
		return fmt.Errorf("failed to dial platon: %w", err)
	}
	c.Platon = platonpb.NewPlatonServiceClient(c.platonConn)

	c.logger.Info("Service client connections initialized (lazy)")
	return nil
}

// Close closes all connections
func (c *ServiceClients) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	var errs []error

	if c.russellConn != nil {
		if err := c.russellConn.Close(); err != nil {
			errs = append(errs, err)
		}
	}
	if c.turingConn != nil {
		if err := c.turingConn.Close(); err != nil {
			errs = append(errs, err)
		}
	}
	if c.hypatiaConn != nil {
		if err := c.hypatiaConn.Close(); err != nil {
			errs = append(errs, err)
		}
	}
	if c.leibnizConn != nil {
		if err := c.leibnizConn.Close(); err != nil {
			errs = append(errs, err)
		}
	}
	if c.babbageConn != nil {
		if err := c.babbageConn.Close(); err != nil {
			errs = append(errs, err)
		}
	}
	if c.platonConn != nil {
		if err := c.platonConn.Close(); err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("errors closing connections: %v", errs)
	}
	return nil
}

// IsConnected checks if a specific service is connected
func (c *ServiceClients) IsConnected(service string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	switch service {
	case "russell":
		return c.Russell != nil
	case "turing":
		return c.Turing != nil
	case "hypatia":
		return c.Hypatia != nil
	case "leibniz":
		return c.Leibniz != nil
	case "babbage":
		return c.Babbage != nil
	case "platon":
		return c.Platon != nil
	default:
		return false
	}
}

// GetServiceStatus returns connection status for all services
func (c *ServiceClients) GetServiceStatus() map[string]string {
	c.mu.RLock()
	defer c.mu.RUnlock()

	status := make(map[string]string)

	if c.Russell != nil {
		status["russell"] = "connected"
	} else {
		status["russell"] = "disconnected"
	}

	if c.Turing != nil {
		status["turing"] = "connected"
	} else {
		status["turing"] = "disconnected"
	}

	if c.Hypatia != nil {
		status["hypatia"] = "connected"
	} else {
		status["hypatia"] = "disconnected"
	}

	if c.Leibniz != nil {
		status["leibniz"] = "connected"
	} else {
		status["leibniz"] = "disconnected"
	}

	if c.Babbage != nil {
		status["babbage"] = "connected"
	} else {
		status["babbage"] = "disconnected"
	}

	if c.Platon != nil {
		status["platon"] = "connected"
	} else {
		status["platon"] = "disconnected"
	}

	return status
}
