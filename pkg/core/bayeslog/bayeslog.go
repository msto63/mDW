// ============================================================================
// meinDENKWERK (mDW) - Lokale KI-Plattform
// ============================================================================
//
// Package:     bayeslog
// Description: Central logging to Bayes service
// Author:      Mike Stoffels with Claude
// Created:     2025-12-06
// License:     MIT
// ============================================================================

package bayeslog

import (
	"context"
	"sync"
	"time"

	bayespb "github.com/msto63/mDW/api/gen/bayes"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// LogLevel represents the log level
type LogLevel int32

const (
	LevelDebug LogLevel = 1
	LevelInfo  LogLevel = 2
	LevelWarn  LogLevel = 3
	LevelError LogLevel = 4
)

// Client is a Bayes logging client
type Client struct {
	bayesAddr   string
	serviceName string
	conn        *grpc.ClientConn
	client      bayespb.BayesServiceClient
	mu          sync.Mutex
	buffer      []*bayespb.LogEntry
	bufferSize  int
	flushInt    time.Duration
	stopCh      chan struct{}
}

// Config holds client configuration
type Config struct {
	BayesAddr   string
	ServiceName string
	BufferSize  int
	FlushInterval time.Duration
}

// DefaultConfig returns default configuration
func DefaultConfig() Config {
	return Config{
		BayesAddr:     "localhost:9120",
		BufferSize:    100,
		FlushInterval: 5 * time.Second,
	}
}

// NewClient creates a new Bayes logging client
func NewClient(cfg Config) *Client {
	if cfg.BufferSize <= 0 {
		cfg.BufferSize = 100
	}
	if cfg.FlushInterval <= 0 {
		cfg.FlushInterval = 5 * time.Second
	}

	return &Client{
		bayesAddr:   cfg.BayesAddr,
		serviceName: cfg.ServiceName,
		buffer:      make([]*bayespb.LogEntry, 0, cfg.BufferSize),
		bufferSize:  cfg.BufferSize,
		flushInt:    cfg.FlushInterval,
		stopCh:      make(chan struct{}),
	}
}

// Connect establishes connection to Bayes
func (c *Client) Connect(ctx context.Context) error {
	dialCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	conn, err := grpc.DialContext(dialCtx, c.bayesAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	if err != nil {
		return err
	}

	c.conn = conn
	c.client = bayespb.NewBayesServiceClient(conn)

	// Start flush goroutine
	go c.flushLoop(ctx)

	return nil
}

// Close closes the client
func (c *Client) Close() error {
	close(c.stopCh)
	c.Flush(context.Background())
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

// Log logs a message at a specific level
func (c *Client) Log(level LogLevel, message string, fields map[string]string) {
	entry := &bayespb.LogEntry{
		Service:   c.serviceName,
		Level:     bayespb.LogLevel(level),
		Message:   message,
		Timestamp: time.Now().UnixNano(),
		Fields:    fields,
	}

	c.mu.Lock()
	c.buffer = append(c.buffer, entry)
	shouldFlush := len(c.buffer) >= c.bufferSize
	c.mu.Unlock()

	if shouldFlush {
		go c.Flush(context.Background())
	}
}

// Debug logs at debug level
func (c *Client) Debug(message string, fields map[string]string) {
	c.Log(LevelDebug, message, fields)
}

// Info logs at info level
func (c *Client) Info(message string, fields map[string]string) {
	c.Log(LevelInfo, message, fields)
}

// Warn logs at warn level
func (c *Client) Warn(message string, fields map[string]string) {
	c.Log(LevelWarn, message, fields)
}

// Error logs at error level
func (c *Client) Error(message string, fields map[string]string) {
	c.Log(LevelError, message, fields)
}

// Flush sends buffered logs to Bayes
func (c *Client) Flush(ctx context.Context) error {
	c.mu.Lock()
	if len(c.buffer) == 0 {
		c.mu.Unlock()
		return nil
	}

	entries := c.buffer
	c.buffer = make([]*bayespb.LogEntry, 0, c.bufferSize)
	c.mu.Unlock()

	if c.client == nil {
		return nil
	}

	_, err := c.client.LogBatch(ctx, &bayespb.LogBatchRequest{
		Entries: entries,
	})

	return err
}

// flushLoop periodically flushes the buffer
func (c *Client) flushLoop(ctx context.Context) {
	ticker := time.NewTicker(c.flushInt)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-c.stopCh:
			return
		case <-ticker.C:
			c.Flush(ctx)
		}
	}
}

// Global client for convenience
var (
	globalClient *Client
	globalMu     sync.Mutex
)

// Init initializes the global Bayes logging client
func Init(ctx context.Context, serviceName, bayesAddr string) error {
	globalMu.Lock()
	defer globalMu.Unlock()

	cfg := DefaultConfig()
	cfg.ServiceName = serviceName
	cfg.BayesAddr = bayesAddr

	globalClient = NewClient(cfg)
	return globalClient.Connect(ctx)
}

// GetClient returns the global client
func GetClient() *Client {
	globalMu.Lock()
	defer globalMu.Unlock()
	return globalClient
}

// LogToGlobal logs to the global client if initialized
func LogToGlobal(level LogLevel, message string, fields map[string]string) {
	c := GetClient()
	if c != nil {
		c.Log(level, message, fields)
	}
}
