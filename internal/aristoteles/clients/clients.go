// Package clients provides gRPC client wrappers for Aristoteles service
package clients

import (
	"context"
	"fmt"
	"time"

	babbagepb "github.com/msto63/mDW/api/gen/babbage"
	hypatiapb "github.com/msto63/mDW/api/gen/hypatia"
	leibnizpb "github.com/msto63/mDW/api/gen/leibniz"
	platonpb "github.com/msto63/mDW/api/gen/platon"
	turingpb "github.com/msto63/mDW/api/gen/turing"
	"github.com/msto63/mDW/pkg/core/logging"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// ServiceClients holds all service client connections
type ServiceClients struct {
	Turing  turingpb.TuringServiceClient
	Leibniz leibnizpb.LeibnizServiceClient
	Hypatia hypatiapb.HypatiaServiceClient
	Babbage babbagepb.BabbageServiceClient
	Platon  platonpb.PlatonServiceClient

	conns  []*grpc.ClientConn
	logger *logging.Logger
}

// Config holds client configuration
type Config struct {
	TuringAddr  string
	LeibnizAddr string
	HypatiaAddr string
	BabbageAddr string
	PlatonAddr  string
	Timeout     time.Duration
}

// DefaultConfig returns default client configuration
func DefaultConfig() *Config {
	return &Config{
		TuringAddr:  "localhost:9200",
		LeibnizAddr: "localhost:9140",
		HypatiaAddr: "localhost:9220",
		BabbageAddr: "localhost:9150",
		PlatonAddr:  "localhost:9130",
		Timeout:     10 * time.Second,
	}
}

// NewServiceClients creates new service clients
func NewServiceClients(cfg *Config) (*ServiceClients, error) {
	if cfg == nil {
		cfg = DefaultConfig()
	}

	clients := &ServiceClients{
		conns:  make([]*grpc.ClientConn, 0),
		logger: logging.New("aristoteles-clients"),
	}

	ctx, cancel := context.WithTimeout(context.Background(), cfg.Timeout)
	defer cancel()

	// Connect to Turing
	if cfg.TuringAddr != "" {
		conn, err := grpc.DialContext(ctx, cfg.TuringAddr,
			grpc.WithTransportCredentials(insecure.NewCredentials()),
		)
		if err != nil {
			clients.logger.Warn("Failed to connect to Turing", "addr", cfg.TuringAddr, "error", err)
		} else {
			clients.Turing = turingpb.NewTuringServiceClient(conn)
			clients.conns = append(clients.conns, conn)
			clients.logger.Info("Connected to Turing", "addr", cfg.TuringAddr)
		}
	}

	// Connect to Leibniz
	if cfg.LeibnizAddr != "" {
		conn, err := grpc.DialContext(ctx, cfg.LeibnizAddr,
			grpc.WithTransportCredentials(insecure.NewCredentials()),
		)
		if err != nil {
			clients.logger.Warn("Failed to connect to Leibniz", "addr", cfg.LeibnizAddr, "error", err)
		} else {
			clients.Leibniz = leibnizpb.NewLeibnizServiceClient(conn)
			clients.conns = append(clients.conns, conn)
			clients.logger.Info("Connected to Leibniz", "addr", cfg.LeibnizAddr)
		}
	}

	// Connect to Hypatia
	if cfg.HypatiaAddr != "" {
		conn, err := grpc.DialContext(ctx, cfg.HypatiaAddr,
			grpc.WithTransportCredentials(insecure.NewCredentials()),
		)
		if err != nil {
			clients.logger.Warn("Failed to connect to Hypatia", "addr", cfg.HypatiaAddr, "error", err)
		} else {
			clients.Hypatia = hypatiapb.NewHypatiaServiceClient(conn)
			clients.conns = append(clients.conns, conn)
			clients.logger.Info("Connected to Hypatia", "addr", cfg.HypatiaAddr)
		}
	}

	// Connect to Babbage
	if cfg.BabbageAddr != "" {
		conn, err := grpc.DialContext(ctx, cfg.BabbageAddr,
			grpc.WithTransportCredentials(insecure.NewCredentials()),
		)
		if err != nil {
			clients.logger.Warn("Failed to connect to Babbage", "addr", cfg.BabbageAddr, "error", err)
		} else {
			clients.Babbage = babbagepb.NewBabbageServiceClient(conn)
			clients.conns = append(clients.conns, conn)
			clients.logger.Info("Connected to Babbage", "addr", cfg.BabbageAddr)
		}
	}

	// Connect to Platon
	if cfg.PlatonAddr != "" {
		conn, err := grpc.DialContext(ctx, cfg.PlatonAddr,
			grpc.WithTransportCredentials(insecure.NewCredentials()),
		)
		if err != nil {
			clients.logger.Warn("Failed to connect to Platon", "addr", cfg.PlatonAddr, "error", err)
		} else {
			clients.Platon = platonpb.NewPlatonServiceClient(conn)
			clients.conns = append(clients.conns, conn)
			clients.logger.Info("Connected to Platon", "addr", cfg.PlatonAddr)
		}
	}

	return clients, nil
}

// Close closes all client connections
func (c *ServiceClients) Close() error {
	var errs []error
	for _, conn := range c.conns {
		if err := conn.Close(); err != nil {
			errs = append(errs, err)
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("failed to close %d connections", len(errs))
	}
	return nil
}

// TuringWrapper wraps Turing client for router interface
type TuringWrapper struct {
	client turingpb.TuringServiceClient
}

// NewTuringWrapper creates a new Turing wrapper
func NewTuringWrapper(client turingpb.TuringServiceClient) *TuringWrapper {
	return &TuringWrapper{client: client}
}

// Chat calls Turing Chat
func (w *TuringWrapper) Chat(ctx context.Context, req *turingpb.ChatRequest) (*turingpb.ChatResponse, error) {
	return w.client.Chat(ctx, req)
}

// LeibnizWrapper wraps Leibniz client for router interface
type LeibnizWrapper struct {
	client leibnizpb.LeibnizServiceClient
}

// NewLeibnizWrapper creates a new Leibniz wrapper
func NewLeibnizWrapper(client leibnizpb.LeibnizServiceClient) *LeibnizWrapper {
	return &LeibnizWrapper{client: client}
}

// Execute calls Leibniz Execute
func (w *LeibnizWrapper) Execute(ctx context.Context, req *leibnizpb.ExecuteRequest) (*leibnizpb.ExecuteResponse, error) {
	return w.client.Execute(ctx, req)
}

// FindBestAgent calls Leibniz FindBestAgent for RAG-style agent selection
func (w *LeibnizWrapper) FindBestAgent(ctx context.Context, req *leibnizpb.FindAgentRequest) (*leibnizpb.AgentMatchResponse, error) {
	return w.client.FindBestAgent(ctx, req)
}

// FindTopAgents calls Leibniz FindTopAgents for RAG-style agent selection
func (w *LeibnizWrapper) FindTopAgents(ctx context.Context, req *leibnizpb.FindTopAgentsRequest) (*leibnizpb.AgentMatchListResponse, error) {
	return w.client.FindTopAgents(ctx, req)
}

// HypatiaWrapper wraps Hypatia client for router interface
type HypatiaWrapper struct {
	client hypatiapb.HypatiaServiceClient
}

// NewHypatiaWrapper creates a new Hypatia wrapper
func NewHypatiaWrapper(client hypatiapb.HypatiaServiceClient) *HypatiaWrapper {
	return &HypatiaWrapper{client: client}
}

// Search calls Hypatia Search
func (w *HypatiaWrapper) Search(ctx context.Context, req *hypatiapb.SearchRequest) (*hypatiapb.SearchResponse, error) {
	return w.client.Search(ctx, req)
}

// AugmentPrompt calls Hypatia AugmentPrompt
func (w *HypatiaWrapper) AugmentPrompt(ctx context.Context, req *hypatiapb.AugmentPromptRequest) (*hypatiapb.AugmentPromptResponse, error) {
	return w.client.AugmentPrompt(ctx, req)
}

// BabbageWrapper wraps Babbage client for router interface
type BabbageWrapper struct {
	client babbagepb.BabbageServiceClient
}

// NewBabbageWrapper creates a new Babbage wrapper
func NewBabbageWrapper(client babbagepb.BabbageServiceClient) *BabbageWrapper {
	return &BabbageWrapper{client: client}
}

// Summarize calls Babbage Summarize
func (w *BabbageWrapper) Summarize(ctx context.Context, req *babbagepb.SummarizeRequest) (*babbagepb.SummarizeResponse, error) {
	return w.client.Summarize(ctx, req)
}

// Translate calls Babbage Translate
func (w *BabbageWrapper) Translate(ctx context.Context, req *babbagepb.TranslateRequest) (*babbagepb.TranslateResponse, error) {
	return w.client.Translate(ctx, req)
}
