// ============================================================================
// meinDENKWERK (mDW) - Lokale KI-Plattform
// ============================================================================
//
// Package:     platon
// Description: Unit tests for Platon gRPC client
// Author:      Mike Stoffels with Claude
// Created:     2025-12-10
// License:     MIT
// ============================================================================

package platon

import (
	"context"
	"testing"
	"time"

	pb "github.com/msto63/mDW/api/gen/platon"
)

// ============================================================================
// Unit Tests - Configuration
// ============================================================================

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	tests := []struct {
		name     string
		got      interface{}
		expected interface{}
	}{
		{"Host", cfg.Host, "localhost"},
		{"Port", cfg.Port, 9130},
		{"Timeout", cfg.Timeout, 30 * time.Second},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.expected {
				t.Errorf("DefaultConfig().%s = %v, expected %v", tt.name, tt.got, tt.expected)
			}
		})
	}
}

// ============================================================================
// Unit Tests - ProcessRequest
// ============================================================================

func TestProcessRequest_Validation(t *testing.T) {
	tests := []struct {
		name     string
		request  ProcessRequest
		hasID    bool
		hasMeta  bool
	}{
		{
			name: "complete request",
			request: ProcessRequest{
				RequestID:  "test-123",
				PipelineID: "default",
				Prompt:     "test prompt",
				Metadata:   map[string]string{"key": "value"},
			},
			hasID:   true,
			hasMeta: true,
		},
		{
			name: "minimal request",
			request: ProcessRequest{
				Prompt: "test prompt",
			},
			hasID:   false,
			hasMeta: false,
		},
		{
			name: "post-processing request",
			request: ProcessRequest{
				RequestID:  "test-456",
				PipelineID: "content-filter",
				Response:   "response to process",
			},
			hasID:   true,
			hasMeta: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if (tt.request.RequestID != "") != tt.hasID {
				t.Errorf("RequestID presence mismatch")
			}
			if (tt.request.Metadata != nil) != tt.hasMeta {
				t.Errorf("Metadata presence mismatch")
			}
		})
	}
}

func TestProcessOptions_Defaults(t *testing.T) {
	opts := &ProcessOptions{}

	if opts.SkipPreProcessing != false {
		t.Error("SkipPreProcessing should default to false")
	}
	if opts.SkipPostProcessing != false {
		t.Error("SkipPostProcessing should default to false")
	}
	if opts.DryRun != false {
		t.Error("DryRun should default to false")
	}
	if opts.Debug != false {
		t.Error("Debug should default to false")
	}
	if opts.TimeoutSeconds != 0 {
		t.Error("TimeoutSeconds should default to 0")
	}
}

// ============================================================================
// Unit Tests - ProcessResponse
// ============================================================================

func TestProcessResponse_Fields(t *testing.T) {
	resp := &ProcessResponse{
		RequestID:         "test-123",
		ProcessedPrompt:   "processed prompt",
		ProcessedResponse: "processed response",
		Blocked:           false,
		Modified:          true,
		DurationMs:        150,
		Metadata:          map[string]string{"processed": "true"},
		AuditLog: []AuditEntry{
			{Handler: "pii-filter", Phase: "pre", DurationMs: 50, Modified: true},
		},
	}

	if resp.RequestID != "test-123" {
		t.Errorf("RequestID = %s, expected test-123", resp.RequestID)
	}
	if !resp.Modified {
		t.Error("Modified should be true")
	}
	if resp.Blocked {
		t.Error("Blocked should be false")
	}
	if len(resp.AuditLog) != 1 {
		t.Errorf("AuditLog length = %d, expected 1", len(resp.AuditLog))
	}
}

func TestProcessResponse_Blocked(t *testing.T) {
	resp := &ProcessResponse{
		RequestID:   "blocked-123",
		Blocked:     true,
		BlockReason: "Content violates policy",
	}

	if !resp.Blocked {
		t.Error("Blocked should be true")
	}
	if resp.BlockReason == "" {
		t.Error("BlockReason should not be empty when blocked")
	}
}

// ============================================================================
// Unit Tests - AuditEntry
// ============================================================================

func TestAuditEntry_Fields(t *testing.T) {
	entry := AuditEntry{
		Handler:    "content-filter",
		Phase:      "post",
		DurationMs: 25,
		Error:      "",
		Modified:   true,
		Details:    map[string]string{"filtered": "pii"},
	}

	if entry.Handler != "content-filter" {
		t.Errorf("Handler = %s, expected content-filter", entry.Handler)
	}
	if entry.Error != "" {
		t.Error("Error should be empty")
	}
	if !entry.Modified {
		t.Error("Modified should be true")
	}
}

// ============================================================================
// Unit Tests - Proto Conversion (toProtoRequest)
// ============================================================================

func TestToProtoRequest_Basic(t *testing.T) {
	// Create a mock client for testing the conversion
	c := &Client{}

	req := &ProcessRequest{
		RequestID:  "test-request-1",
		PipelineID: "default",
		Prompt:     "Hello world",
		Metadata:   map[string]string{"source": "test"},
	}

	pbReq := c.toProtoRequest(req)

	if pbReq.RequestId != req.RequestID {
		t.Errorf("RequestId = %s, expected %s", pbReq.RequestId, req.RequestID)
	}
	if pbReq.PipelineId != req.PipelineID {
		t.Errorf("PipelineId = %s, expected %s", pbReq.PipelineId, req.PipelineID)
	}
	if pbReq.Prompt != req.Prompt {
		t.Errorf("Prompt = %s, expected %s", pbReq.Prompt, req.Prompt)
	}
	if pbReq.Metadata["source"] != "test" {
		t.Error("Metadata not properly converted")
	}
	if pbReq.Options != nil {
		t.Error("Options should be nil when not provided")
	}
}

func TestToProtoRequest_WithOptions(t *testing.T) {
	c := &Client{}

	req := &ProcessRequest{
		RequestID:  "test-request-2",
		PipelineID: "custom",
		Prompt:     "Test prompt",
		Options: &ProcessOptions{
			SkipPreProcessing:  true,
			SkipPostProcessing: false,
			DryRun:             true,
			TimeoutSeconds:     60,
			Debug:              true,
		},
	}

	pbReq := c.toProtoRequest(req)

	if pbReq.Options == nil {
		t.Fatal("Options should not be nil")
	}
	if !pbReq.Options.SkipPreProcessing {
		t.Error("SkipPreProcessing should be true")
	}
	if pbReq.Options.SkipPostProcessing {
		t.Error("SkipPostProcessing should be false")
	}
	if !pbReq.Options.DryRun {
		t.Error("DryRun should be true")
	}
	if pbReq.Options.TimeoutSeconds != 60 {
		t.Errorf("TimeoutSeconds = %d, expected 60", pbReq.Options.TimeoutSeconds)
	}
	if !pbReq.Options.Debug {
		t.Error("Debug should be true")
	}
}

func TestToProtoRequest_WithResponse(t *testing.T) {
	c := &Client{}

	req := &ProcessRequest{
		RequestID:  "test-request-3",
		PipelineID: "post-only",
		Response:   "Response to process",
	}

	pbReq := c.toProtoRequest(req)

	if pbReq.Response != req.Response {
		t.Errorf("Response = %s, expected %s", pbReq.Response, req.Response)
	}
}

// ============================================================================
// Unit Tests - Proto Conversion (fromProtoResponse)
// ============================================================================

func TestFromProtoResponse_Basic(t *testing.T) {
	c := &Client{}

	pbResp := &pb.ProcessResponse{
		RequestId:         "test-123",
		ProcessedPrompt:   "processed prompt",
		ProcessedResponse: "processed response",
		Blocked:           false,
		Modified:          true,
		DurationMs:        100,
		Metadata:          map[string]string{"key": "value"},
	}

	resp := c.fromProtoResponse(pbResp)

	if resp.RequestID != pbResp.RequestId {
		t.Errorf("RequestID = %s, expected %s", resp.RequestID, pbResp.RequestId)
	}
	if resp.ProcessedPrompt != pbResp.ProcessedPrompt {
		t.Errorf("ProcessedPrompt mismatch")
	}
	if resp.ProcessedResponse != pbResp.ProcessedResponse {
		t.Errorf("ProcessedResponse mismatch")
	}
	if resp.Blocked != pbResp.Blocked {
		t.Errorf("Blocked mismatch")
	}
	if resp.Modified != pbResp.Modified {
		t.Errorf("Modified mismatch")
	}
	if resp.DurationMs != pbResp.DurationMs {
		t.Errorf("DurationMs mismatch")
	}
}

func TestFromProtoResponse_WithAuditLog(t *testing.T) {
	c := &Client{}

	pbResp := &pb.ProcessResponse{
		RequestId: "test-456",
		AuditLog: []*pb.AuditEntry{
			{
				Handler:    "handler-1",
				Phase:      "pre",
				DurationMs: 10,
				Modified:   true,
				Details:    map[string]string{"action": "filter"},
			},
			{
				Handler:    "handler-2",
				Phase:      "post",
				DurationMs: 20,
				Error:      "some error",
			},
		},
	}

	resp := c.fromProtoResponse(pbResp)

	if len(resp.AuditLog) != 2 {
		t.Fatalf("AuditLog length = %d, expected 2", len(resp.AuditLog))
	}

	// Check first entry
	if resp.AuditLog[0].Handler != "handler-1" {
		t.Errorf("First handler = %s, expected handler-1", resp.AuditLog[0].Handler)
	}
	if resp.AuditLog[0].Phase != "pre" {
		t.Errorf("First phase = %s, expected pre", resp.AuditLog[0].Phase)
	}
	if !resp.AuditLog[0].Modified {
		t.Error("First entry should be modified")
	}

	// Check second entry
	if resp.AuditLog[1].Handler != "handler-2" {
		t.Errorf("Second handler = %s, expected handler-2", resp.AuditLog[1].Handler)
	}
	if resp.AuditLog[1].Error != "some error" {
		t.Errorf("Second error = %s, expected 'some error'", resp.AuditLog[1].Error)
	}
}

func TestFromProtoResponse_Blocked(t *testing.T) {
	c := &Client{}

	pbResp := &pb.ProcessResponse{
		RequestId:   "blocked-test",
		Blocked:     true,
		BlockReason: "Policy violation detected",
	}

	resp := c.fromProtoResponse(pbResp)

	if !resp.Blocked {
		t.Error("Blocked should be true")
	}
	if resp.BlockReason != "Policy violation detected" {
		t.Errorf("BlockReason = %s, expected 'Policy violation detected'", resp.BlockReason)
	}
}

// ============================================================================
// Unit Tests - IsConnected
// ============================================================================

func TestIsConnected_Nil(t *testing.T) {
	c := &Client{conn: nil}

	if c.IsConnected() {
		t.Error("IsConnected should return false when conn is nil")
	}
}

// ============================================================================
// Unit Tests - Close
// ============================================================================

func TestClose_NilConnection(t *testing.T) {
	c := &Client{conn: nil}

	err := c.Close()
	if err != nil {
		t.Errorf("Close() with nil conn should return nil, got %v", err)
	}
}

// ============================================================================
// Benchmark Tests
// ============================================================================

func BenchmarkToProtoRequest(b *testing.B) {
	c := &Client{}
	req := &ProcessRequest{
		RequestID:  "bench-request",
		PipelineID: "default",
		Prompt:     "This is a benchmark test prompt for conversion testing",
		Metadata:   map[string]string{"source": "bench", "type": "test"},
		Options: &ProcessOptions{
			SkipPreProcessing: false,
			Debug:             true,
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		c.toProtoRequest(req)
	}
}

func BenchmarkFromProtoResponse(b *testing.B) {
	c := &Client{}
	pbResp := &pb.ProcessResponse{
		RequestId:         "bench-response",
		ProcessedPrompt:   "Processed benchmark prompt",
		ProcessedResponse: "Processed benchmark response",
		Modified:          true,
		DurationMs:        50,
		AuditLog: []*pb.AuditEntry{
			{Handler: "h1", Phase: "pre", DurationMs: 10},
			{Handler: "h2", Phase: "post", DurationMs: 20},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		c.fromProtoResponse(pbResp)
	}
}

// ============================================================================
// Table-Driven Tests
// ============================================================================

func TestProcessRequest_AllCombinations(t *testing.T) {
	tests := []struct {
		name        string
		requestID   string
		pipelineID  string
		prompt      string
		response    string
		wantPreProc bool
		wantPostProc bool
	}{
		{
			name:         "pre-processing only",
			requestID:    "pre-1",
			pipelineID:   "default",
			prompt:       "Process this prompt",
			response:     "",
			wantPreProc:  true,
			wantPostProc: false,
		},
		{
			name:         "post-processing only",
			requestID:    "post-1",
			pipelineID:   "default",
			prompt:       "",
			response:     "Process this response",
			wantPreProc:  false,
			wantPostProc: true,
		},
		{
			name:         "both pre and post",
			requestID:    "both-1",
			pipelineID:   "full",
			prompt:       "Input prompt",
			response:     "Output response",
			wantPreProc:  true,
			wantPostProc: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := &ProcessRequest{
				RequestID:  tt.requestID,
				PipelineID: tt.pipelineID,
				Prompt:     tt.prompt,
				Response:   tt.response,
			}

			hasPrompt := req.Prompt != ""
			hasResponse := req.Response != ""

			if hasPrompt != tt.wantPreProc {
				t.Errorf("pre-processing availability mismatch: got %v, want %v", hasPrompt, tt.wantPreProc)
			}
			if hasResponse != tt.wantPostProc {
				t.Errorf("post-processing availability mismatch: got %v, want %v", hasResponse, tt.wantPostProc)
			}
		})
	}
}

// ============================================================================
// Context Tests
// ============================================================================

func TestContextTimeout(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
	defer cancel()

	// Wait for context to expire
	time.Sleep(5 * time.Millisecond)

	select {
	case <-ctx.Done():
		if ctx.Err() != context.DeadlineExceeded {
			t.Errorf("Expected DeadlineExceeded, got %v", ctx.Err())
		}
	default:
		t.Error("Context should have expired")
	}
}
