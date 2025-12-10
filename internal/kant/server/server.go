package server

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/msto63/mDW/internal/kant/client"
	"github.com/msto63/mDW/internal/kant/handler"
	"github.com/msto63/mDW/pkg/core/health"
	"github.com/msto63/mDW/pkg/core/logging"
)

// Server is the Kant API Gateway server
type Server struct {
	httpServer *http.Server
	handler    *handler.Handler
	clients    *client.ServiceClients
	health     *health.Registry
	logger     *logging.Logger
	config     Config
}

// Config holds server configuration
type Config struct {
	Host         string
	HTTPPort     int
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	Version      string

	// Service addresses
	RussellAddr string
	TuringAddr  string
	HypatiaAddr string
	LeibnizAddr string
	BabbageAddr string
}

// DefaultConfig returns default server configuration
func DefaultConfig() Config {
	return Config{
		Host:         "0.0.0.0",
		HTTPPort:     8080,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 120 * time.Second,
		Version:      "1.0.0",

		// Default service addresses
		RussellAddr: "localhost:9100",
		TuringAddr:  "localhost:9200",
		HypatiaAddr: "localhost:9220",
		LeibnizAddr: "localhost:9140",
		BabbageAddr: "localhost:9150",
	}
}

// New creates a new Kant server
func New(cfg Config) (*Server, error) {
	logger := logging.New("kant-server")

	// Create service clients
	clientCfg := client.Config{
		RussellAddr: cfg.RussellAddr,
		TuringAddr:  cfg.TuringAddr,
		HypatiaAddr: cfg.HypatiaAddr,
		LeibnizAddr: cfg.LeibnizAddr,
		BabbageAddr: cfg.BabbageAddr,
	}
	clients := client.NewServiceClients(clientCfg)

	// Connect to services (lazy - non-blocking)
	if err := clients.ConnectLazy(); err != nil {
		logger.Warn("Failed to initialize service clients", "error", err)
	}

	// Create handler with clients
	h := handler.NewHandler(cfg.Version, clients)

	// Create WebSocket handler
	wsHandler := handler.NewWebSocketHandler(clients)

	// Create HTTP server
	mux := http.NewServeMux()

	// WebSocket route
	mux.Handle("/api/v1/chat/ws", wsHandler)

	// API routes
	mux.Handle("/", h)
	mux.Handle("/api/", h)
	mux.Handle("/api/v1/", h)

	httpServer := &http.Server{
		Addr:         fmt.Sprintf("%s:%d", cfg.Host, cfg.HTTPPort),
		Handler:      loggingMiddleware(logger, mux),
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
	}

	// Create health registry
	healthRegistry := health.NewRegistry("kant", cfg.Version)
	healthRegistry.RegisterFunc("http", func(ctx context.Context) health.CheckResult {
		return health.CheckResult{
			Name:    "http",
			Status:  health.StatusHealthy,
			Message: "HTTP server is running",
		}
	})

	// Register service client health checks
	healthRegistry.RegisterFunc("services", func(ctx context.Context) health.CheckResult {
		status := clients.GetServiceStatus()
		connected := 0
		for _, s := range status {
			if s == "connected" {
				connected++
			}
		}
		if connected == 0 {
			return health.CheckResult{
				Name:    "services",
				Status:  health.StatusUnhealthy,
				Message: "No backend services connected",
			}
		}
		return health.CheckResult{
			Name:    "services",
			Status:  health.StatusHealthy,
			Message: fmt.Sprintf("%d/%d services connected", connected, len(status)),
		}
	})

	return &Server{
		httpServer: httpServer,
		handler:    h,
		clients:    clients,
		health:     healthRegistry,
		logger:     logger,
		config:     cfg,
	}, nil
}

// loggingMiddleware adds request logging
func loggingMiddleware(logger *logging.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Wrap response writer to capture status code
		wrapper := &responseWrapper{ResponseWriter: w, statusCode: http.StatusOK}

		next.ServeHTTP(wrapper, r)

		logger.Info("HTTP request",
			"method", r.Method,
			"path", r.URL.Path,
			"status", wrapper.statusCode,
			"duration", time.Since(start),
		)
	})
}

// responseWrapper wraps http.ResponseWriter to capture status code
type responseWrapper struct {
	http.ResponseWriter
	statusCode int
}

func (w *responseWrapper) WriteHeader(code int) {
	w.statusCode = code
	w.ResponseWriter.WriteHeader(code)
}

// Flush implements http.Flusher for SSE streaming support
func (w *responseWrapper) Flush() {
	if flusher, ok := w.ResponseWriter.(http.Flusher); ok {
		flusher.Flush()
	}
}

// Start starts the server
func (s *Server) Start() error {
	s.logger.Info("Starting Kant API Gateway",
		"host", s.config.Host,
		"port", s.config.HTTPPort,
	)
	return s.httpServer.ListenAndServe()
}

// StartAsync starts the server asynchronously
func (s *Server) StartAsync() error {
	s.logger.Info("Starting Kant API Gateway (async)",
		"host", s.config.Host,
		"port", s.config.HTTPPort,
	)

	go func() {
		if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			s.logger.Error("HTTP server error", "error", err)
		}
	}()

	return nil
}

// Stop gracefully stops the server
func (s *Server) Stop(ctx context.Context) error {
	s.logger.Info("Stopping Kant API Gateway")

	// Close service clients
	if s.clients != nil {
		if err := s.clients.Close(); err != nil {
			s.logger.Warn("Error closing service clients", "error", err)
		}
	}

	return s.httpServer.Shutdown(ctx)
}

// Address returns the server address
func (s *Server) Address() string {
	return fmt.Sprintf("%s:%d", s.config.Host, s.config.HTTPPort)
}

// HealthRegistry returns the health check registry
func (s *Server) HealthRegistry() *health.Registry {
	return s.health
}

// Clients returns the service clients
func (s *Server) Clients() *client.ServiceClients {
	return s.clients
}
