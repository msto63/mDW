// ============================================================================
// meinDENKWERK (mDW) - Lokale KI-Plattform
// ============================================================================
//
// Package:     leibniz
// Description: Integration tests for Leibniz-Platon communication
// Author:      Mike Stoffels with Claude
// Created:     2025-12-10
// License:     MIT
// ============================================================================

//go:build integration
// +build integration

package leibniz_test

import (
	"context"
	"net"
	"testing"
	"time"

	commonpb "github.com/msto63/mDW/api/gen/common"
	pb "github.com/msto63/mDW/api/gen/platon"
	"github.com/msto63/mDW/internal/leibniz/platon"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// ============================================================================
// Mock Platon Server for Integration Testing
// ============================================================================

type mockPlatonServer struct {
	pb.UnimplementedPlatonServiceServer
	processPreFunc  func(context.Context, *pb.ProcessRequest) (*pb.ProcessResponse, error)
	processPostFunc func(context.Context, *pb.ProcessRequest) (*pb.ProcessResponse, error)
	processFunc     func(context.Context, *pb.ProcessRequest) (*pb.ProcessResponse, error)
	healthFunc      func(context.Context, *commonpb.HealthCheckRequest) (*commonpb.HealthCheckResponse, error)
}

func (s *mockPlatonServer) ProcessPre(ctx context.Context, req *pb.ProcessRequest) (*pb.ProcessResponse, error) {
	if s.processPreFunc != nil {
		return s.processPreFunc(ctx, req)
	}
	return &pb.ProcessResponse{
		RequestId:       req.RequestId,
		ProcessedPrompt: req.Prompt,
		Modified:        false,
	}, nil
}

func (s *mockPlatonServer) ProcessPost(ctx context.Context, req *pb.ProcessRequest) (*pb.ProcessResponse, error) {
	if s.processPostFunc != nil {
		return s.processPostFunc(ctx, req)
	}
	return &pb.ProcessResponse{
		RequestId:         req.RequestId,
		ProcessedResponse: req.Response,
		Modified:          false,
	}, nil
}

func (s *mockPlatonServer) Process(ctx context.Context, req *pb.ProcessRequest) (*pb.ProcessResponse, error) {
	if s.processFunc != nil {
		return s.processFunc(ctx, req)
	}
	return &pb.ProcessResponse{
		RequestId:         req.RequestId,
		ProcessedPrompt:   req.Prompt,
		ProcessedResponse: req.Response,
		Modified:          false,
	}, nil
}

func (s *mockPlatonServer) HealthCheck(ctx context.Context, req *commonpb.HealthCheckRequest) (*commonpb.HealthCheckResponse, error) {
	if s.healthFunc != nil {
		return s.healthFunc(ctx, req)
	}
	return &commonpb.HealthCheckResponse{
		Status:  "healthy",
		Service: "mock-platon",
	}, nil
}

// startMockPlatonServer starts a mock Platon gRPC server and returns cleanup function
func startMockPlatonServer(t *testing.T, mock *mockPlatonServer) (string, func()) {
	listener, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		t.Fatalf("Failed to create listener: %v", err)
	}

	server := grpc.NewServer()
	pb.RegisterPlatonServiceServer(server, mock)

	go func() {
		if err := server.Serve(listener); err != nil {
			// Server stopped, ignore error
		}
	}()

	cleanup := func() {
		server.GracefulStop()
		listener.Close()
	}

	return listener.Addr().String(), cleanup
}

// ============================================================================
// Integration Tests - Client Connection
// ============================================================================

func TestIntegration_PlatonClient_Connect(t *testing.T) {
	mock := &mockPlatonServer{}
	addr, cleanup := startMockPlatonServer(t, mock)
	defer cleanup()

	// Parse address
	host, portStr, _ := net.SplitHostPort(addr)
	var port int
	_, _ = parsePort(portStr, &port)

	cfg := platon.Config{
		Host:    host,
		Port:    port,
		Timeout: 5 * time.Second,
	}

	client, err := platon.NewClient(cfg)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	if !client.IsConnected() {
		t.Error("Client should be connected")
	}
}

func TestIntegration_PlatonClient_HealthCheck(t *testing.T) {
	mock := &mockPlatonServer{
		healthFunc: func(_ context.Context, _ *commonpb.HealthCheckRequest) (*commonpb.HealthCheckResponse, error) {
			return &commonpb.HealthCheckResponse{
				Status:  "healthy",
				Service: "platon",
			}, nil
		},
	}
	addr, cleanup := startMockPlatonServer(t, mock)
	defer cleanup()

	client := createTestClient(t, addr)
	defer client.Close()

	ctx := context.Background()
	healthy, err := client.HealthCheck(ctx)
	if err != nil {
		t.Fatalf("HealthCheck failed: %v", err)
	}

	if !healthy {
		t.Error("Server should be healthy")
	}
}

func TestIntegration_PlatonClient_HealthCheck_Unhealthy(t *testing.T) {
	mock := &mockPlatonServer{
		healthFunc: func(_ context.Context, _ *commonpb.HealthCheckRequest) (*commonpb.HealthCheckResponse, error) {
			return &commonpb.HealthCheckResponse{
				Status:  "unhealthy",
				Service: "platon",
			}, nil
		},
	}
	addr, cleanup := startMockPlatonServer(t, mock)
	defer cleanup()

	client := createTestClient(t, addr)
	defer client.Close()

	ctx := context.Background()
	healthy, err := client.HealthCheck(ctx)
	if err != nil {
		t.Fatalf("HealthCheck failed: %v", err)
	}

	if healthy {
		t.Error("Server should be unhealthy")
	}
}

// ============================================================================
// Integration Tests - Pre-Processing
// ============================================================================

func TestIntegration_PlatonClient_ProcessPre(t *testing.T) {
	mock := &mockPlatonServer{
		processPreFunc: func(_ context.Context, req *pb.ProcessRequest) (*pb.ProcessResponse, error) {
			return &pb.ProcessResponse{
				RequestId:       req.RequestId,
				ProcessedPrompt: "[FILTERED] " + req.Prompt,
				Modified:        true,
				DurationMs:      10,
				AuditLog: []*pb.AuditEntry{
					{
						Handler:    "content-filter",
						Phase:      "pre",
						DurationMs: 5,
						Modified:   true,
					},
				},
			}, nil
		},
	}
	addr, cleanup := startMockPlatonServer(t, mock)
	defer cleanup()

	client := createTestClient(t, addr)
	defer client.Close()

	ctx := context.Background()
	req := &platon.ProcessRequest{
		RequestID:  "test-pre-1",
		PipelineID: "default",
		Prompt:     "Hello world",
	}

	resp, err := client.ProcessPre(ctx, req)
	if err != nil {
		t.Fatalf("ProcessPre failed: %v", err)
	}

	if resp.RequestID != "test-pre-1" {
		t.Errorf("RequestID = %s, expected 'test-pre-1'", resp.RequestID)
	}
	if resp.ProcessedPrompt != "[FILTERED] Hello world" {
		t.Errorf("ProcessedPrompt = %s", resp.ProcessedPrompt)
	}
	if !resp.Modified {
		t.Error("Modified should be true")
	}
	if len(resp.AuditLog) != 1 {
		t.Errorf("AuditLog length = %d, expected 1", len(resp.AuditLog))
	}
}

func TestIntegration_PlatonClient_ProcessPre_Blocked(t *testing.T) {
	mock := &mockPlatonServer{
		processPreFunc: func(_ context.Context, req *pb.ProcessRequest) (*pb.ProcessResponse, error) {
			return &pb.ProcessResponse{
				RequestId:   req.RequestId,
				Blocked:     true,
				BlockReason: "Content violates policy",
			}, nil
		},
	}
	addr, cleanup := startMockPlatonServer(t, mock)
	defer cleanup()

	client := createTestClient(t, addr)
	defer client.Close()

	ctx := context.Background()
	req := &platon.ProcessRequest{
		RequestID:  "blocked-1",
		PipelineID: "strict",
		Prompt:     "Forbidden content",
	}

	resp, err := client.ProcessPre(ctx, req)
	if err != nil {
		t.Fatalf("ProcessPre failed: %v", err)
	}

	if !resp.Blocked {
		t.Error("Response should be blocked")
	}
	if resp.BlockReason != "Content violates policy" {
		t.Errorf("BlockReason = %s", resp.BlockReason)
	}
}

// ============================================================================
// Integration Tests - Post-Processing
// ============================================================================

func TestIntegration_PlatonClient_ProcessPost(t *testing.T) {
	mock := &mockPlatonServer{
		processPostFunc: func(_ context.Context, req *pb.ProcessRequest) (*pb.ProcessResponse, error) {
			return &pb.ProcessResponse{
				RequestId:         req.RequestId,
				ProcessedResponse: req.Response + " [SANITIZED]",
				Modified:          true,
				DurationMs:        15,
			}, nil
		},
	}
	addr, cleanup := startMockPlatonServer(t, mock)
	defer cleanup()

	client := createTestClient(t, addr)
	defer client.Close()

	ctx := context.Background()
	req := &platon.ProcessRequest{
		RequestID:  "test-post-1",
		PipelineID: "output-filter",
		Response:   "Response with PII: test@example.com",
	}

	resp, err := client.ProcessPost(ctx, req)
	if err != nil {
		t.Fatalf("ProcessPost failed: %v", err)
	}

	if resp.ProcessedResponse != "Response with PII: test@example.com [SANITIZED]" {
		t.Errorf("ProcessedResponse = %s", resp.ProcessedResponse)
	}
	if !resp.Modified {
		t.Error("Modified should be true")
	}
}

// ============================================================================
// Integration Tests - Full Pipeline
// ============================================================================

func TestIntegration_PlatonClient_Process(t *testing.T) {
	mock := &mockPlatonServer{
		processFunc: func(_ context.Context, req *pb.ProcessRequest) (*pb.ProcessResponse, error) {
			return &pb.ProcessResponse{
				RequestId:         req.RequestId,
				ProcessedPrompt:   "[PRE] " + req.Prompt,
				ProcessedResponse: req.Response + " [POST]",
				Modified:          true,
				DurationMs:        25,
				Metadata:          map[string]string{"processed": "true"},
			}, nil
		},
	}
	addr, cleanup := startMockPlatonServer(t, mock)
	defer cleanup()

	client := createTestClient(t, addr)
	defer client.Close()

	ctx := context.Background()
	req := &platon.ProcessRequest{
		RequestID:  "test-full-1",
		PipelineID: "full",
		Prompt:     "Input",
		Response:   "Output",
		Metadata:   map[string]string{"source": "test"},
	}

	resp, err := client.Process(ctx, req)
	if err != nil {
		t.Fatalf("Process failed: %v", err)
	}

	if resp.ProcessedPrompt != "[PRE] Input" {
		t.Errorf("ProcessedPrompt = %s", resp.ProcessedPrompt)
	}
	if resp.ProcessedResponse != "Output [POST]" {
		t.Errorf("ProcessedResponse = %s", resp.ProcessedResponse)
	}
	if resp.Metadata["processed"] != "true" {
		t.Error("Metadata should contain 'processed: true'")
	}
}

// ============================================================================
// Integration Tests - Options
// ============================================================================

func TestIntegration_PlatonClient_ProcessWithOptions(t *testing.T) {
	var receivedOpts *pb.ProcessOptions
	mock := &mockPlatonServer{
		processPreFunc: func(_ context.Context, req *pb.ProcessRequest) (*pb.ProcessResponse, error) {
			receivedOpts = req.Options
			return &pb.ProcessResponse{
				RequestId:       req.RequestId,
				ProcessedPrompt: req.Prompt,
			}, nil
		},
	}
	addr, cleanup := startMockPlatonServer(t, mock)
	defer cleanup()

	client := createTestClient(t, addr)
	defer client.Close()

	ctx := context.Background()
	req := &platon.ProcessRequest{
		RequestID:  "opts-test",
		PipelineID: "default",
		Prompt:     "Test",
		Options: &platon.ProcessOptions{
			SkipPreProcessing: true,
			DryRun:            true,
			Debug:             true,
			TimeoutSeconds:    30,
		},
	}

	_, err := client.ProcessPre(ctx, req)
	if err != nil {
		t.Fatalf("ProcessPre failed: %v", err)
	}

	if receivedOpts == nil {
		t.Fatal("Options should be received")
	}
	if !receivedOpts.SkipPreProcessing {
		t.Error("SkipPreProcessing should be true")
	}
	if !receivedOpts.DryRun {
		t.Error("DryRun should be true")
	}
	if !receivedOpts.Debug {
		t.Error("Debug should be true")
	}
	if receivedOpts.TimeoutSeconds != 30 {
		t.Errorf("TimeoutSeconds = %d, expected 30", receivedOpts.TimeoutSeconds)
	}
}

// ============================================================================
// Integration Tests - Context Cancellation
// ============================================================================

func TestIntegration_PlatonClient_ContextCancellation(t *testing.T) {
	mock := &mockPlatonServer{
		processPreFunc: func(ctx context.Context, req *pb.ProcessRequest) (*pb.ProcessResponse, error) {
			// Simulate slow processing
			select {
			case <-time.After(5 * time.Second):
				return &pb.ProcessResponse{RequestId: req.RequestId}, nil
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		},
	}
	addr, cleanup := startMockPlatonServer(t, mock)
	defer cleanup()

	client := createTestClient(t, addr)
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	req := &platon.ProcessRequest{
		RequestID: "timeout-test",
		Prompt:    "Test",
	}

	_, err := client.ProcessPre(ctx, req)
	if err == nil {
		t.Error("Expected timeout error")
	}
}

// ============================================================================
// Integration Tests - Concurrent Requests
// ============================================================================

func TestIntegration_PlatonClient_ConcurrentRequests(t *testing.T) {
	requestCount := 0
	mock := &mockPlatonServer{
		processPreFunc: func(_ context.Context, req *pb.ProcessRequest) (*pb.ProcessResponse, error) {
			requestCount++
			return &pb.ProcessResponse{
				RequestId:       req.RequestId,
				ProcessedPrompt: req.Prompt,
			}, nil
		},
	}
	addr, cleanup := startMockPlatonServer(t, mock)
	defer cleanup()

	client := createTestClient(t, addr)
	defer client.Close()

	ctx := context.Background()
	numRequests := 50
	done := make(chan error, numRequests)

	for i := 0; i < numRequests; i++ {
		go func(id int) {
			req := &platon.ProcessRequest{
				RequestID:  "concurrent-" + string(rune('a'+id%26)),
				PipelineID: "default",
				Prompt:     "Concurrent test",
			}
			_, err := client.ProcessPre(ctx, req)
			done <- err
		}(i)
	}

	errorCount := 0
	for i := 0; i < numRequests; i++ {
		if err := <-done; err != nil {
			errorCount++
		}
	}

	if errorCount > 0 {
		t.Errorf("%d concurrent requests failed", errorCount)
	}
}

// ============================================================================
// Helper Functions
// ============================================================================

func createTestClient(t *testing.T, addr string) *platon.Client {
	host, portStr, err := net.SplitHostPort(addr)
	if err != nil {
		t.Fatalf("Failed to parse address: %v", err)
	}

	var port int
	parsePort(portStr, &port)

	cfg := platon.Config{
		Host:    host,
		Port:    port,
		Timeout: 5 * time.Second,
	}

	client, err := platon.NewClient(cfg)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	return client
}

func parsePort(s string, port *int) (int, error) {
	*port = 0
	for _, c := range s {
		if c < '0' || c > '9' {
			return 0, nil
		}
		*port = *port*10 + int(c-'0')
	}
	return *port, nil
}

// ============================================================================
// Benchmark Tests
// ============================================================================

func BenchmarkIntegration_ProcessPre(b *testing.B) {
	mock := &mockPlatonServer{}
	listener, _ := net.Listen("tcp", "localhost:0")
	server := grpc.NewServer()
	pb.RegisterPlatonServiceServer(server, mock)
	go server.Serve(listener)
	defer server.GracefulStop()

	addr := listener.Addr().String()
	host, portStr, _ := net.SplitHostPort(addr)
	var port int
	parsePort(portStr, &port)

	conn, _ := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	defer conn.Close()

	cfg := platon.Config{Host: host, Port: port, Timeout: 5 * time.Second}
	client, _ := platon.NewClient(cfg)
	defer client.Close()

	ctx := context.Background()
	req := &platon.ProcessRequest{
		RequestID:  "bench",
		PipelineID: "default",
		Prompt:     "Benchmark test prompt",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		client.ProcessPre(ctx, req)
	}
}
