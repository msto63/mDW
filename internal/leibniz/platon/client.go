// ============================================================================
// meinDENKWERK (mDW) - Lokale KI-Plattform
// ============================================================================
//
// Package:     platon
// Description: gRPC client for Platon pipeline processing service
// Author:      Mike Stoffels with Claude
// Created:     2025-12-10
// License:     MIT
// ============================================================================

package platon

import (
	"context"
	"fmt"
	"time"

	commonpb "github.com/msto63/mDW/api/gen/common"
	pb "github.com/msto63/mDW/api/gen/platon"
	"github.com/msto63/mDW/pkg/core/logging"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// Client is a gRPC client for the Platon service
type Client struct {
	conn    *grpc.ClientConn
	client  pb.PlatonServiceClient
	logger  *logging.Logger
	timeout time.Duration
}

// Config holds client configuration
type Config struct {
	Host    string
	Port    int
	Timeout time.Duration
}

// DefaultConfig returns default client configuration
func DefaultConfig() Config {
	return Config{
		Host:    "localhost",
		Port:    9130,
		Timeout: 30 * time.Second,
	}
}

// NewClient creates a new Platon gRPC client
func NewClient(cfg Config) (*Client, error) {
	logger := logging.New("platon-client")

	addr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)

	conn, err := grpc.NewClient(addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Platon at %s: %w", addr, err)
	}

	client := &Client{
		conn:    conn,
		client:  pb.NewPlatonServiceClient(conn),
		logger:  logger,
		timeout: cfg.Timeout,
	}

	logger.Info("Connected to Platon service", "address", addr)
	return client, nil
}

// Close closes the client connection
func (c *Client) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

// ProcessRequest represents a processing request
type ProcessRequest struct {
	RequestID  string
	PipelineID string
	Prompt     string
	Response   string            // For post-processing
	Metadata   map[string]string
	Options    *ProcessOptions
}

// ProcessOptions holds processing options
type ProcessOptions struct {
	SkipPreProcessing  bool
	SkipPostProcessing bool
	DryRun             bool
	TimeoutSeconds     int32
	Debug              bool
}

// ProcessResponse represents a processing response
type ProcessResponse struct {
	RequestID         string
	ProcessedPrompt   string
	ProcessedResponse string
	Blocked           bool
	BlockReason       string
	Modified          bool
	AuditLog          []AuditEntry
	Metadata          map[string]string
	DurationMs        int64
}

// AuditEntry represents an audit log entry
type AuditEntry struct {
	Handler    string
	Phase      string
	DurationMs int64
	Error      string
	Modified   bool
	Details    map[string]string
}

// Process performs full pipeline processing (pre + post)
func (c *Client) Process(ctx context.Context, req *ProcessRequest) (*ProcessResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	pbReq := c.toProtoRequest(req)
	resp, err := c.client.Process(ctx, pbReq)
	if err != nil {
		c.logger.Error("Process failed", "error", err, "request_id", req.RequestID)
		return nil, fmt.Errorf("platon process failed: %w", err)
	}

	return c.fromProtoResponse(resp), nil
}

// ProcessPre performs only pre-processing
func (c *Client) ProcessPre(ctx context.Context, req *ProcessRequest) (*ProcessResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	pbReq := c.toProtoRequest(req)
	resp, err := c.client.ProcessPre(ctx, pbReq)
	if err != nil {
		c.logger.Error("ProcessPre failed", "error", err, "request_id", req.RequestID)
		return nil, fmt.Errorf("platon pre-process failed: %w", err)
	}

	return c.fromProtoResponse(resp), nil
}

// ProcessPost performs only post-processing
func (c *Client) ProcessPost(ctx context.Context, req *ProcessRequest) (*ProcessResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	pbReq := c.toProtoRequest(req)
	resp, err := c.client.ProcessPost(ctx, pbReq)
	if err != nil {
		c.logger.Error("ProcessPost failed", "error", err, "request_id", req.RequestID)
		return nil, fmt.Errorf("platon post-process failed: %w", err)
	}

	return c.fromProtoResponse(resp), nil
}

// HealthCheck checks if the Platon service is healthy
func (c *Client) HealthCheck(ctx context.Context) (bool, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	resp, err := c.client.HealthCheck(ctx, &commonpb.HealthCheckRequest{})
	if err != nil {
		return false, err
	}

	return resp.Status == "healthy", nil
}

// toProtoRequest converts internal request to proto request
func (c *Client) toProtoRequest(req *ProcessRequest) *pb.ProcessRequest {
	pbReq := &pb.ProcessRequest{
		RequestId:  req.RequestID,
		PipelineId: req.PipelineID,
		Prompt:     req.Prompt,
		Response:   req.Response,
		Metadata:   req.Metadata,
	}

	if req.Options != nil {
		pbReq.Options = &pb.ProcessOptions{
			SkipPreProcessing:  req.Options.SkipPreProcessing,
			SkipPostProcessing: req.Options.SkipPostProcessing,
			DryRun:             req.Options.DryRun,
			TimeoutSeconds:     req.Options.TimeoutSeconds,
			Debug:              req.Options.Debug,
		}
	}

	return pbReq
}

// fromProtoResponse converts proto response to internal response
func (c *Client) fromProtoResponse(resp *pb.ProcessResponse) *ProcessResponse {
	result := &ProcessResponse{
		RequestID:         resp.RequestId,
		ProcessedPrompt:   resp.ProcessedPrompt,
		ProcessedResponse: resp.ProcessedResponse,
		Blocked:           resp.Blocked,
		BlockReason:       resp.BlockReason,
		Modified:          resp.Modified,
		Metadata:          resp.Metadata,
		DurationMs:        resp.DurationMs,
	}

	for _, entry := range resp.AuditLog {
		result.AuditLog = append(result.AuditLog, AuditEntry{
			Handler:    entry.Handler,
			Phase:      entry.Phase,
			DurationMs: entry.DurationMs,
			Error:      entry.Error,
			Modified:   entry.Modified,
			Details:    entry.Details,
		})
	}

	return result
}

// IsConnected returns whether the client is connected
func (c *Client) IsConnected() bool {
	return c.conn != nil
}
