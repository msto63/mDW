// ============================================================================
// meinDENKWERK (mDW) - Lokale KI-Plattform
// ============================================================================
//
// Package:     logging
// Description: BayesWriter sends log entries to the Bayes logging service
// Author:      Mike Stoffels with Claude
// Created:     2025-12-06
// License:     MIT
// ============================================================================

package logging

import (
	"context"
	"encoding/json"
	"io"
	"os"
	"sync"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// BayesLogEntry represents the structure of a log entry from Foundation
type BayesLogEntry struct {
	Timestamp     string                 `json:"timestamp"`
	Level         string                 `json:"level"`
	Message       string                 `json:"message"`
	Logger        string                 `json:"logger"`
	RequestID     string                 `json:"request_id,omitempty"`
	UserID        string                 `json:"user_id,omitempty"`
	CorrelationID string                 `json:"correlation_id,omitempty"`
	Error         string                 `json:"error,omitempty"`
	Caller        string                 `json:"caller,omitempty"`
	Fields        map[string]interface{} `json:"fields,omitempty"`
}

// BayesWriter implements io.Writer and sends logs to Bayes service
type BayesWriter struct {
	// Configuration
	address     string
	serviceName string
	batchSize   int
	flushPeriod time.Duration

	// Connection
	conn   *grpc.ClientConn
	client BayesLogClient

	// Batching
	buffer    []BayesLogEntry
	bufferMu  sync.Mutex
	flushCh   chan struct{}
	stopCh    chan struct{}
	doneCh    chan struct{}

	// Fallback
	fallback io.Writer
	enabled  bool
}

// BayesLogClient is a minimal interface for the Bayes log service
// This allows us to avoid importing the generated proto code here
type BayesLogClient interface {
	LogBatch(ctx context.Context, entries []BayesLogEntry) (int, error)
}

// BayesWriterConfig holds configuration for BayesWriter
type BayesWriterConfig struct {
	Address     string        // Bayes service address (e.g., "localhost:9120")
	ServiceName string        // Name of the service sending logs
	BatchSize   int           // Number of entries to batch (default: 100)
	FlushPeriod time.Duration // How often to flush (default: 5s)
	Fallback    io.Writer     // Fallback writer on failure (default: os.Stdout)
}

// DefaultBayesWriterConfig returns default configuration
func DefaultBayesWriterConfig() BayesWriterConfig {
	return BayesWriterConfig{
		Address:     "localhost:9120",
		BatchSize:   100,
		FlushPeriod: 5 * time.Second,
		Fallback:    os.Stdout,
	}
}

// NewBayesWriter creates a new BayesWriter
func NewBayesWriter(cfg BayesWriterConfig) (*BayesWriter, error) {
	if cfg.BatchSize <= 0 {
		cfg.BatchSize = 100
	}
	if cfg.FlushPeriod <= 0 {
		cfg.FlushPeriod = 5 * time.Second
	}
	if cfg.Fallback == nil {
		cfg.Fallback = os.Stdout
	}

	w := &BayesWriter{
		address:     cfg.Address,
		serviceName: cfg.ServiceName,
		batchSize:   cfg.BatchSize,
		flushPeriod: cfg.FlushPeriod,
		buffer:      make([]BayesLogEntry, 0, cfg.BatchSize),
		flushCh:     make(chan struct{}, 1),
		stopCh:      make(chan struct{}),
		doneCh:      make(chan struct{}),
		fallback:    cfg.Fallback,
		enabled:     false, // Disabled until connected
	}

	// Try to connect asynchronously
	go w.connect()

	// Start flush worker
	go w.flushWorker()

	return w, nil
}

// connect attempts to connect to the Bayes service
func (w *BayesWriter) connect() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	conn, err := grpc.DialContext(ctx, w.address,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	if err != nil {
		// Connection failed, logs will go to fallback
		return
	}

	w.bufferMu.Lock()
	w.conn = conn
	w.client = &grpcBayesClient{conn: conn}
	w.enabled = true
	w.bufferMu.Unlock()
}

// Write implements io.Writer
func (w *BayesWriter) Write(p []byte) (n int, err error) {
	// Always write to fallback first (for local visibility)
	n, err = w.fallback.Write(p)
	if err != nil {
		return n, err
	}

	// If not enabled, we're done
	w.bufferMu.Lock()
	if !w.enabled {
		w.bufferMu.Unlock()
		return n, nil
	}
	w.bufferMu.Unlock()

	// Parse the JSON log entry
	var entry BayesLogEntry
	if jsonErr := json.Unmarshal(p, &entry); jsonErr != nil {
		// Not valid JSON, skip sending to Bayes
		return n, nil
	}

	// Override service name if set
	if w.serviceName != "" {
		entry.Logger = w.serviceName
	}

	// Add to buffer
	w.bufferMu.Lock()
	w.buffer = append(w.buffer, entry)
	shouldFlush := len(w.buffer) >= w.batchSize
	w.bufferMu.Unlock()

	// Trigger flush if buffer is full
	if shouldFlush {
		select {
		case w.flushCh <- struct{}{}:
		default:
		}
	}

	return n, nil
}

// flushWorker periodically flushes the buffer
func (w *BayesWriter) flushWorker() {
	defer close(w.doneCh)

	ticker := time.NewTicker(w.flushPeriod)
	defer ticker.Stop()

	for {
		select {
		case <-w.stopCh:
			// Final flush
			w.flush()
			return
		case <-w.flushCh:
			w.flush()
		case <-ticker.C:
			w.flush()
		}
	}
}

// flush sends buffered entries to Bayes
func (w *BayesWriter) flush() {
	w.bufferMu.Lock()
	if len(w.buffer) == 0 || w.client == nil {
		w.bufferMu.Unlock()
		return
	}

	// Copy and clear buffer
	entries := make([]BayesLogEntry, len(w.buffer))
	copy(entries, w.buffer)
	w.buffer = w.buffer[:0]
	client := w.client
	w.bufferMu.Unlock()

	// Send to Bayes (with timeout)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := client.LogBatch(ctx, entries)
	if err != nil {
		// Log send failed - entries are lost but were already written to fallback
		// Could implement retry logic here if needed
	}
}

// Close gracefully shuts down the BayesWriter
func (w *BayesWriter) Close() error {
	close(w.stopCh)
	<-w.doneCh // Wait for final flush

	w.bufferMu.Lock()
	defer w.bufferMu.Unlock()

	if w.conn != nil {
		return w.conn.Close()
	}
	return nil
}

// IsEnabled returns whether the Bayes connection is active
func (w *BayesWriter) IsEnabled() bool {
	w.bufferMu.Lock()
	defer w.bufferMu.Unlock()
	return w.enabled
}

// grpcBayesClient implements BayesLogClient using gRPC
type grpcBayesClient struct {
	conn *grpc.ClientConn
}

// LogBatch sends a batch of log entries to Bayes
// Note: This is a simplified implementation. In production,
// you would use the generated proto client.
func (c *grpcBayesClient) LogBatch(ctx context.Context, entries []BayesLogEntry) (int, error) {
	// For now, this is a placeholder that would call the actual gRPC service
	// Once proto is generated, this will use the real client:
	//
	// client := bayespb.NewBayesServiceClient(c.conn)
	// req := &bayespb.LogBatchRequest{
	//     Entries: convertEntries(entries),
	// }
	// resp, err := client.LogBatch(ctx, req)
	// return int(resp.Accepted), err

	// Placeholder: just return success
	return len(entries), nil
}

// MultiWriter creates an io.Writer that writes to multiple destinations
func MultiWriter(writers ...io.Writer) io.Writer {
	return io.MultiWriter(writers...)
}
