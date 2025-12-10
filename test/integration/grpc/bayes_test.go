// ============================================================================
// meinDENKWERK (mDW) - Lokale KI-Plattform
// ============================================================================
//
// Package:     grpc
// Description: Integration tests for Bayes gRPC service (Logging & Metrics)
// Author:      Mike Stoffels with Claude
// Created:     2025-12-06
// License:     MIT
// ============================================================================

//go:build integration

package grpc

import (
	"context"
	"testing"
	"time"

	"github.com/msto63/mDW/api/gen/common"
	bayespb "github.com/msto63/mDW/api/gen/bayes"
)

// BayesTestClient wraps the Bayes gRPC client for testing
type BayesTestClient struct {
	conn   *TestConnection
	client bayespb.BayesServiceClient
}

// NewBayesTestClient creates a new Bayes test client
func NewBayesTestClient() (*BayesTestClient, error) {
	configs := DefaultServiceConfigs()
	cfg := configs["bayes"]

	conn, err := NewTestConnection(cfg)
	if err != nil {
		return nil, err
	}

	return &BayesTestClient{
		conn:   conn,
		client: bayespb.NewBayesServiceClient(conn.Conn()),
	}, nil
}

// Close closes the test client connection
func (bc *BayesTestClient) Close() error {
	return bc.conn.Close()
}

// Client returns the underlying gRPC client
func (bc *BayesTestClient) Client() bayespb.BayesServiceClient {
	return bc.client
}

// ContextWithTimeout returns a context with a custom timeout
func (bc *BayesTestClient) ContextWithTimeout(timeout time.Duration) (context.Context, context.CancelFunc) {
	return bc.conn.ContextWithTimeout(timeout)
}

// TestBayesHealthCheck tests the health check endpoint
func TestBayesHealthCheck(t *testing.T) {
	client, err := NewBayesTestClient()
	if err != nil {
		t.Fatalf("Failed to create Bayes client: %v", err)
	}
	defer client.Close()

	ctx, cancel := client.ContextWithTimeout(10 * time.Second)
	defer cancel()

	resp, err := client.Client().HealthCheck(ctx, &common.HealthCheckRequest{})
	if err != nil {
		t.Fatalf("HealthCheck failed: %v", err)
	}

	t.Logf("Health Check Response:")
	t.Logf("  Status: %s", resp.GetStatus())
	t.Logf("  Service: %s", resp.GetService())
	t.Logf("  Version: %s", resp.GetVersion())
	t.Logf("  Uptime: %d seconds", resp.GetUptimeSeconds())

	if resp.GetStatus() != "healthy" {
		t.Errorf("Expected status 'healthy', got '%s'", resp.GetStatus())
	}

	if resp.GetService() != "bayes" {
		t.Errorf("Expected service 'bayes', got '%s'", resp.GetService())
	}
}

// TestBayesGetStats tests getting logging statistics
func TestBayesGetStats(t *testing.T) {
	client, err := NewBayesTestClient()
	if err != nil {
		t.Fatalf("Failed to create Bayes client: %v", err)
	}
	defer client.Close()

	ctx, cancel := client.ContextWithTimeout(10 * time.Second)
	defer cancel()

	resp, err := client.Client().GetStats(ctx, &common.Empty{})
	if err != nil {
		t.Fatalf("GetStats failed: %v", err)
	}

	t.Logf("Log Statistics:")
	t.Logf("  Total Logs: %d", resp.GetTotalLogs())
	t.Logf("  Total Metrics: %d", resp.GetTotalMetrics())
	t.Logf("  Storage Bytes: %d", resp.GetStorageBytes())
	t.Logf("  Logs by Service: %v", resp.GetLogsByService())
	t.Logf("  Logs by Level: %v", resp.GetLogsByLevel())
}

// TestBayesLog tests logging a single entry
func TestBayesLog(t *testing.T) {
	client, err := NewBayesTestClient()
	if err != nil {
		t.Fatalf("Failed to create Bayes client: %v", err)
	}
	defer client.Close()

	ctx, cancel := client.ContextWithTimeout(10 * time.Second)
	defer cancel()

	req := &bayespb.LogRequest{
		Entry: &bayespb.LogEntry{
			Service:   "integration-test",
			Level:     bayespb.LogLevel_LOG_LEVEL_INFO,
			Message:   "Integration test log entry",
			Timestamp: time.Now().UnixNano(),
			Fields: map[string]string{
				"test": "true",
				"env":  "integration",
			},
			RequestId: "test-request-123",
		},
	}

	_, err = client.Client().Log(ctx, req)
	if err != nil {
		t.Fatalf("Log failed: %v", err)
	}

	t.Log("Successfully logged entry")
}

// TestBayesLogBatch tests batch logging
func TestBayesLogBatch(t *testing.T) {
	client, err := NewBayesTestClient()
	if err != nil {
		t.Fatalf("Failed to create Bayes client: %v", err)
	}
	defer client.Close()

	ctx, cancel := client.ContextWithTimeout(10 * time.Second)
	defer cancel()

	entries := []*bayespb.LogEntry{
		{
			Service:   "integration-test",
			Level:     bayespb.LogLevel_LOG_LEVEL_DEBUG,
			Message:   "Batch log entry 1",
			Timestamp: time.Now().UnixNano(),
		},
		{
			Service:   "integration-test",
			Level:     bayespb.LogLevel_LOG_LEVEL_INFO,
			Message:   "Batch log entry 2",
			Timestamp: time.Now().UnixNano(),
		},
		{
			Service:   "integration-test",
			Level:     bayespb.LogLevel_LOG_LEVEL_WARN,
			Message:   "Batch log entry 3",
			Timestamp: time.Now().UnixNano(),
		},
	}

	req := &bayespb.LogBatchRequest{
		Entries: entries,
	}

	resp, err := client.Client().LogBatch(ctx, req)
	if err != nil {
		t.Fatalf("LogBatch failed: %v", err)
	}

	t.Logf("Batch Log Response:")
	t.Logf("  Accepted: %d", resp.GetAccepted())
	t.Logf("  Rejected: %d", resp.GetRejected())

	if resp.GetAccepted() != 3 {
		t.Errorf("Expected 3 accepted logs, got %d", resp.GetAccepted())
	}
}

// TestBayesQueryLogs tests querying logs
func TestBayesQueryLogs(t *testing.T) {
	client, err := NewBayesTestClient()
	if err != nil {
		t.Fatalf("Failed to create Bayes client: %v", err)
	}
	defer client.Close()

	ctx, cancel := client.ContextWithTimeout(10 * time.Second)
	defer cancel()

	req := &bayespb.QueryLogsRequest{
		Service:  "integration-test",
		MinLevel: bayespb.LogLevel_LOG_LEVEL_DEBUG,
		Limit:    10,
		Sort:     bayespb.SortOrder_SORT_ORDER_DESC,
	}

	resp, err := client.Client().QueryLogs(ctx, req)
	if err != nil {
		t.Fatalf("QueryLogs failed: %v", err)
	}

	t.Logf("Query Logs Response:")
	t.Logf("  Total: %d", resp.GetTotal())
	t.Logf("  Has More: %v", resp.GetHasMore())
	t.Logf("  Entries: %d", len(resp.GetEntries()))

	for i, entry := range resp.GetEntries() {
		if i >= 5 {
			break
		}
		t.Logf("  - [%s] %s: %s", entry.GetLevel().String(), entry.GetService(), entry.GetMessage())
	}
}

// TestBayesRecordMetric tests recording a metric
func TestBayesRecordMetric(t *testing.T) {
	client, err := NewBayesTestClient()
	if err != nil {
		t.Fatalf("Failed to create Bayes client: %v", err)
	}
	defer client.Close()

	ctx, cancel := client.ContextWithTimeout(10 * time.Second)
	defer cancel()

	req := &bayespb.MetricRequest{
		Entry: &bayespb.MetricEntry{
			Service:   "integration-test",
			Name:      "test_counter",
			Value:     42.0,
			Type:      bayespb.MetricType_METRIC_TYPE_COUNTER,
			Timestamp: time.Now().UnixNano(),
			Labels: map[string]string{
				"env": "integration",
			},
		},
	}

	_, err = client.Client().RecordMetric(ctx, req)
	if err != nil {
		t.Fatalf("RecordMetric failed: %v", err)
	}

	t.Log("Successfully recorded metric")
}

// TestBayesRecordMetricBatch tests batch metric recording
func TestBayesRecordMetricBatch(t *testing.T) {
	client, err := NewBayesTestClient()
	if err != nil {
		t.Fatalf("Failed to create Bayes client: %v", err)
	}
	defer client.Close()

	ctx, cancel := client.ContextWithTimeout(10 * time.Second)
	defer cancel()

	entries := []*bayespb.MetricEntry{
		{
			Service:   "integration-test",
			Name:      "batch_gauge_1",
			Value:     100.0,
			Type:      bayespb.MetricType_METRIC_TYPE_GAUGE,
			Timestamp: time.Now().UnixNano(),
		},
		{
			Service:   "integration-test",
			Name:      "batch_gauge_2",
			Value:     200.0,
			Type:      bayespb.MetricType_METRIC_TYPE_GAUGE,
			Timestamp: time.Now().UnixNano(),
		},
		{
			Service:   "integration-test",
			Name:      "batch_histogram",
			Value:     1.5,
			Type:      bayespb.MetricType_METRIC_TYPE_HISTOGRAM,
			Timestamp: time.Now().UnixNano(),
		},
	}

	req := &bayespb.MetricBatchRequest{
		Entries: entries,
	}

	_, err = client.Client().RecordMetricBatch(ctx, req)
	if err != nil {
		t.Fatalf("RecordMetricBatch failed: %v", err)
	}

	t.Log("Successfully recorded batch metrics")
}

// TestBayesQueryMetrics tests querying metrics
func TestBayesQueryMetrics(t *testing.T) {
	client, err := NewBayesTestClient()
	if err != nil {
		t.Fatalf("Failed to create Bayes client: %v", err)
	}
	defer client.Close()

	ctx, cancel := client.ContextWithTimeout(10 * time.Second)
	defer cancel()

	req := &bayespb.QueryMetricsRequest{
		Service:       "integration-test",
		Name:          "test_counter",
		FromTimestamp: time.Now().Add(-1 * time.Hour).UnixNano(),
		ToTimestamp:   time.Now().UnixNano(),
		Aggregation:   bayespb.AggregationType_AGGREGATION_TYPE_NONE,
	}

	resp, err := client.Client().QueryMetrics(ctx, req)
	if err != nil {
		t.Fatalf("QueryMetrics failed: %v", err)
	}

	t.Logf("Query Metrics Response:")
	t.Logf("  Service: %s", resp.GetService())
	t.Logf("  Name: %s", resp.GetName())
	t.Logf("  Data Points: %d", len(resp.GetDataPoints()))

	for i, dp := range resp.GetDataPoints() {
		if i >= 5 {
			break
		}
		t.Logf("  - Value: %.2f at %d", dp.GetValue(), dp.GetTimestamp())
	}
}

// RunBayesTestSuite runs all Bayes tests
func RunBayesTestSuite(t *testing.T) *TestSuite {
	suite := NewTestSuite("bayes")

	tests := []struct {
		name string
		fn   func(*testing.T)
	}{
		{"HealthCheck", TestBayesHealthCheck},
		{"GetStats", TestBayesGetStats},
		{"Log", TestBayesLog},
		{"LogBatch", TestBayesLogBatch},
		{"QueryLogs", TestBayesQueryLogs},
		{"RecordMetric", TestBayesRecordMetric},
		{"RecordMetricBatch", TestBayesRecordMetricBatch},
		{"QueryMetrics", TestBayesQueryMetrics},
	}

	for _, tt := range tests {
		start := time.Now()
		passed := t.Run(tt.name, tt.fn)
		duration := time.Since(start)

		result := TestResult{
			Name:     tt.name,
			Passed:   passed,
			Duration: duration,
		}
		suite.AddResult(result)
	}

	suite.Finish()
	t.Log(suite.Summary())

	return suite
}
