// File: client_test.go
// Title: TCOL Service Client Unit Tests
// Description: Comprehensive unit tests for the TCOL service client including
//              connection management, service discovery, health checking,
//              circuit breaker patterns, retry logic, and mock service
//              interactions. Tests cover reliability and resilience features.
// Author: msto63 with Claude Sonnet 4.0
// Version: v0.1.0
// Created: 2025-01-25
// Modified: 2025-01-25
//
// Change History:
// - 2025-01-25 v0.1.0: Initial comprehensive client test suite

package client

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	mdwlog "github.com/msto63/mDW/foundation/core/log"
	mdwexecutor "github.com/msto63/mDW/foundation/tcol/executor"
)

// Test helper structures

type TestServiceDiscovery struct {
	services     map[string]string
	errors       map[string]error
	listResponse []string
	listError    error
	callCount    map[string]int
	mutex        sync.RWMutex
}

func NewTestServiceDiscovery() *TestServiceDiscovery {
	return &TestServiceDiscovery{
		services:  make(map[string]string),
		errors:    make(map[string]error),
		callCount: make(map[string]int),
	}
}

func (tsd *TestServiceDiscovery) GetServiceAddress(serviceName string) (string, error) {
	tsd.mutex.Lock()
	defer tsd.mutex.Unlock()
	
	tsd.callCount[serviceName]++
	
	if err, exists := tsd.errors[serviceName]; exists {
		return "", err
	}
	
	if address, exists := tsd.services[serviceName]; exists {
		return address, nil
	}
	
	// Return mock address
	address := fmt.Sprintf("localhost:50%03d", len(tsd.services)+1)
	tsd.services[serviceName] = address
	return address, nil
}

func (tsd *TestServiceDiscovery) ListServices() ([]string, error) {
	tsd.mutex.RLock()
	defer tsd.mutex.RUnlock()
	
	if tsd.listError != nil {
		return nil, tsd.listError
	}
	
	if tsd.listResponse != nil {
		return tsd.listResponse, nil
	}
	
	services := make([]string, 0, len(tsd.services))
	for name := range tsd.services {
		services = append(services, name)
	}
	return services, nil
}

func (tsd *TestServiceDiscovery) RegisterService(name, address string) error {
	tsd.mutex.Lock()
	defer tsd.mutex.Unlock()
	
	tsd.services[name] = address
	return nil
}

func (tsd *TestServiceDiscovery) UnregisterService(name string) error {
	tsd.mutex.Lock()
	defer tsd.mutex.Unlock()
	
	delete(tsd.services, name)
	return nil
}

func (tsd *TestServiceDiscovery) SetServiceAddress(name, address string) {
	tsd.mutex.Lock()
	defer tsd.mutex.Unlock()
	tsd.services[name] = address
}

func (tsd *TestServiceDiscovery) SetServiceError(name string, err error) {
	tsd.mutex.Lock()
	defer tsd.mutex.Unlock()
	tsd.errors[name] = err
}

func (tsd *TestServiceDiscovery) GetCallCount(serviceName string) int {
	tsd.mutex.RLock()
	defer tsd.mutex.RUnlock()
	return tsd.callCount[serviceName]
}

func createTestClient(opts ...Options) (*Client, error) {
	defaultOpts := Options{
		Logger:              mdwlog.GetDefault(),
		ServiceDiscovery:    NewTestServiceDiscovery(),
		ConnectionTimeout:   5 * time.Second,
		RequestTimeout:      10 * time.Second,
		MaxRetries:          2,
		HealthCheckInterval: 100 * time.Millisecond, // Fast for testing
		CircuitBreakerConfig: CircuitBreakerConfig{
			FailureThreshold:   3,
			RecoveryTimeout:    1 * time.Second,
			HalfOpenRequests:   2,
			MinRequestsToTrip:  3,
		},
	}
	
	if len(opts) > 0 {
		provided := opts[0]
		if provided.Logger != nil {
			defaultOpts.Logger = provided.Logger
		}
		if provided.ServiceDiscovery != nil {
			defaultOpts.ServiceDiscovery = provided.ServiceDiscovery
		}
		if provided.ConnectionTimeout > 0 {
			defaultOpts.ConnectionTimeout = provided.ConnectionTimeout
		}
		if provided.RequestTimeout > 0 {
			defaultOpts.RequestTimeout = provided.RequestTimeout
		}
		if provided.MaxRetries > 0 {
			defaultOpts.MaxRetries = provided.MaxRetries
		}
		if provided.HealthCheckInterval > 0 {
			defaultOpts.HealthCheckInterval = provided.HealthCheckInterval
		}
		if provided.CircuitBreakerConfig.FailureThreshold > 0 {
			defaultOpts.CircuitBreakerConfig = provided.CircuitBreakerConfig
		}
	}
	
	return New(defaultOpts)
}

func createTestExecutionContext() *mdwexecutor.ExecutionContext {
	return &mdwexecutor.ExecutionContext{
		RequestID:     "test-request-id",
		UserID:        "test-user",
		SessionID:     "test-session",
		Timestamp:     time.Now(),
		Metadata:      make(map[string]interface{}),
	}
}

// Test cases

func TestNew(t *testing.T) {
	tests := []struct {
		name      string
		options   Options
		checkFunc func(*Client) bool
	}{
		{
			name: "Default options",
			options: Options{
				Logger: mdwlog.GetDefault(),
			},
			checkFunc: func(c *Client) bool {
				return c.options.ConnectionTimeout == 10*time.Second &&
					   c.options.RequestTimeout == 30*time.Second &&
					   c.options.MaxRetries == 3 &&
					   c.options.HealthCheckInterval == 30*time.Second
			},
		},
		{
			name: "Custom options",
			options: Options{
				Logger:            mdwlog.GetDefault(),
				ConnectionTimeout: 5 * time.Second,
				RequestTimeout:    15 * time.Second,
				MaxRetries:        5,
				CircuitBreakerConfig: CircuitBreakerConfig{
					FailureThreshold: 10,
					RecoveryTimeout:  30 * time.Second,
				},
			},
			checkFunc: func(c *Client) bool {
				return c.options.ConnectionTimeout == 5*time.Second &&
					   c.options.RequestTimeout == 15*time.Second &&
					   c.options.MaxRetries == 5 &&
					   c.options.CircuitBreakerConfig.FailureThreshold == 10
			},
		},
		{
			name: "Nil logger (should use default)",
			options: Options{
				Logger: nil,
			},
			checkFunc: func(c *Client) bool {
				return c.logger != nil
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := New(tt.options)
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if client == nil {
				t.Fatal("Expected client but got nil")
			}

			if tt.checkFunc != nil && !tt.checkFunc(client) {
				t.Error("Client check function failed")
			}

			// Clean up
			client.Close()
		})
	}
}

func TestClient_Execute_Success(t *testing.T) {
	discovery := NewTestServiceDiscovery()
	discovery.SetServiceAddress("test-service", "localhost:50001")
	
	client, err := createTestClient(Options{
		ServiceDiscovery: discovery,
	})
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	ctx := context.Background()
	execCtx := createTestExecutionContext()
	params := map[string]interface{}{
		"name": "Test Object",
		"id":   123,
	}

	response, err := client.Execute(ctx, "test-service", "OBJECT", "METHOD", params, execCtx)

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
		return
	}

	if response == nil {
		t.Fatal("Expected response but got nil")
	}

	if !response.Success {
		t.Error("Expected successful response")
	}

	// Check response data
	if data, ok := response.Data.(map[string]interface{}); ok {
		if data["object"] != "OBJECT" {
			t.Errorf("Expected object 'OBJECT', got %v", data["object"])
		}
		if data["method"] != "METHOD" {
			t.Errorf("Expected method 'METHOD', got %v", data["method"])
		}
		if data["requestID"] != execCtx.RequestID {
			t.Errorf("Expected requestID %s, got %v", execCtx.RequestID, data["requestID"])
		}
	} else {
		t.Error("Expected response data to be a map")
	}
}

func TestClient_Execute_ServiceDiscoveryError(t *testing.T) {
	discovery := NewTestServiceDiscovery()
	discovery.SetServiceError("unknown-service", errors.New("service not found"))
	
	client, err := createTestClient(Options{
		ServiceDiscovery: discovery,
	})
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	ctx := context.Background()
	execCtx := createTestExecutionContext()

	response, err := client.Execute(ctx, "unknown-service", "OBJECT", "METHOD", nil, execCtx)

	if err == nil {
		t.Error("Expected error but got none")
	}

	if response != nil {
		t.Error("Expected nil response on error")
	}

	if !strings.Contains(err.Error(), "failed to discover service address") {
		t.Errorf("Expected service discovery error, got: %v", err)
	}
}

func TestClient_Execute_ConnectionReuse(t *testing.T) {
	discovery := NewTestServiceDiscovery()
	discovery.SetServiceAddress("reuse-service", "localhost:50002")
	
	client, err := createTestClient(Options{
		ServiceDiscovery: discovery,
	})
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	ctx := context.Background()
	execCtx := createTestExecutionContext()

	// First request
	_, err = client.Execute(ctx, "reuse-service", "OBJECT", "METHOD1", nil, execCtx)
	if err != nil {
		t.Errorf("First request failed: %v", err)
	}

	// Second request to same service
	_, err = client.Execute(ctx, "reuse-service", "OBJECT", "METHOD2", nil, execCtx)
	if err != nil {
		t.Errorf("Second request failed: %v", err)
	}

	// Check that service discovery was only called once
	callCount := discovery.GetCallCount("reuse-service")
	if callCount != 1 {
		t.Errorf("Expected service discovery to be called once, got %d calls", callCount)
	}

	// Check that connection exists
	client.mutex.RLock()
	conn, exists := client.connections["reuse-service"]
	client.mutex.RUnlock()

	if !exists {
		t.Error("Expected connection to exist")
	}

	if conn != nil {
		conn.mutex.RLock()
		requestCount := conn.RequestCount
		conn.mutex.RUnlock()
		
		if requestCount != 2 {
			t.Errorf("Expected 2 requests on connection, got %d", requestCount)
		}
	}
}

func TestClient_Health(t *testing.T) {
	discovery := NewTestServiceDiscovery()
	discovery.SetServiceAddress("health-service", "localhost:50003")
	
	client, err := createTestClient(Options{
		ServiceDiscovery: discovery,
	})
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	ctx := context.Background()

	// Health check should succeed
	err = client.Health(ctx, "health-service")
	if err != nil {
		t.Errorf("Health check failed: %v", err)
	}

	// Check connection status
	client.mutex.RLock()
	conn, exists := client.connections["health-service"]
	client.mutex.RUnlock()

	if !exists {
		t.Error("Expected connection to exist after health check")
	}

	if conn != nil {
		conn.mutex.RLock()
		status := conn.HealthStatus
		conn.mutex.RUnlock()
		
		if status != HealthHealthy {
			t.Errorf("Expected healthy status, got %v", status)
		}
	}
}

func TestClient_Close(t *testing.T) {
	discovery := NewTestServiceDiscovery()
	discovery.SetServiceAddress("close-service", "localhost:50004")
	
	client, err := createTestClient(Options{
		ServiceDiscovery: discovery,
	})
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	// Create a connection
	ctx := context.Background()
	execCtx := createTestExecutionContext()
	_, err = client.Execute(ctx, "close-service", "OBJECT", "METHOD", nil, execCtx)
	if err != nil {
		t.Errorf("Failed to create connection: %v", err)
	}

	// Verify connection exists
	client.mutex.RLock()
	connCount := len(client.connections)
	client.mutex.RUnlock()
	
	if connCount != 1 {
		t.Errorf("Expected 1 connection, got %d", connCount)
	}

	// Close client
	err = client.Close()
	if err != nil {
		t.Errorf("Close failed: %v", err)
	}

	// Verify connections are cleaned up
	client.mutex.RLock()
	connCount = len(client.connections)
	client.mutex.RUnlock()
	
	if connCount != 0 {
		t.Errorf("Expected 0 connections after close, got %d", connCount)
	}
}

func TestCircuitBreaker_StateMachine(t *testing.T) {
	config := CircuitBreakerConfig{
		FailureThreshold:   3,
		RecoveryTimeout:    100 * time.Millisecond,
		HalfOpenRequests:   2,
		MinRequestsToTrip:  3,
	}

	cb := NewCircuitBreaker(config)

	// Initially closed - should allow requests
	if !cb.AllowRequest() {
		t.Error("Circuit breaker should allow requests when closed")
	}

	// Record failures to trip the breaker
	for i := 0; i < 3; i++ {
		cb.RecordFailure()
	}

	// Should now be open
	if cb.AllowRequest() {
		t.Error("Circuit breaker should not allow requests when open")
	}

	// Wait for recovery timeout
	time.Sleep(150 * time.Millisecond)

	// Should now be half-open
	if !cb.AllowRequest() {
		t.Error("Circuit breaker should allow requests when half-open")
	}

	// Record successes to close the breaker
	for i := 0; i < 2; i++ {
		cb.RecordSuccess()
	}

	// Should now be closed again
	if !cb.AllowRequest() {
		t.Error("Circuit breaker should allow requests when closed again")
	}
}

func TestCircuitBreaker_Integration(t *testing.T) {
	discovery := NewTestServiceDiscovery()
	discovery.SetServiceAddress("circuit-service", "localhost:50005")
	
	client, err := createTestClient(Options{
		ServiceDiscovery: discovery,
		CircuitBreakerConfig: CircuitBreakerConfig{
			FailureThreshold:   2,
			RecoveryTimeout:    100 * time.Millisecond,
			HalfOpenRequests:   1,
			MinRequestsToTrip:  2,
		},
	})
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	ctx := context.Background()
	execCtx := createTestExecutionContext()

	// First, trigger failures to open the circuit breaker
	// The mock service fails every 10th request starting at 7
	// We need to get the request count to 7, 17, 27, etc.
	
	// Get connection first
	conn, err := client.getConnection("circuit-service")
	if err != nil {
		t.Fatalf("Failed to get connection: %v", err)
	}

	// Manually set request count to trigger failures
	conn.mutex.Lock()
	conn.RequestCount = 6 // Next request (7) will fail
	conn.mutex.Unlock()

	// This should fail (request count 7)
	_, err = client.Execute(ctx, "circuit-service", "OBJECT", "METHOD", nil, execCtx)
	if err == nil {
		t.Error("Expected first request to fail")
	}

	// This should fail (request count 8) but will succeed due to mock logic
	_, err = client.Execute(ctx, "circuit-service", "OBJECT", "METHOD", nil, execCtx)
	if err != nil {
		t.Errorf("Second request failed unexpectedly: %v", err)
	}

	// Manually trigger failures to open circuit breaker
	for i := 0; i < 3; i++ {
		conn.CircuitBreaker.RecordFailure()
	}

	// Circuit breaker should now be open
	_, err = client.Execute(ctx, "circuit-service", "OBJECT", "METHOD", nil, execCtx)
	if err == nil {
		t.Error("Expected request to fail when circuit breaker is open")
	}

	if !strings.Contains(err.Error(), "circuit breaker is open") {
		t.Errorf("Expected circuit breaker error, got: %v", err)
	}
}

func TestHealthStatus_String(t *testing.T) {
	tests := []struct {
		status   HealthStatus
		expected string
	}{
		{HealthHealthy, "HEALTHY"},
		{HealthUnhealthy, "UNHEALTHY"},
		{HealthDegraded, "DEGRADED"},
		{HealthUnknown, "UNKNOWN"},
		{HealthStatus(999), "UNKNOWN"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := tt.status.String()
			if result != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestMockServiceDiscovery(t *testing.T) {
	discovery := NewMockServiceDiscovery()

	// Test GetServiceAddress with new service
	address, err := discovery.GetServiceAddress("new-service")
	if err != nil {
		t.Errorf("GetServiceAddress failed: %v", err)
	}

	if !strings.HasPrefix(address, "localhost:50") {
		t.Errorf("Expected localhost address, got %s", address)
	}

	// Test GetServiceAddress with existing service
	address2, err := discovery.GetServiceAddress("new-service")
	if err != nil {
		t.Errorf("GetServiceAddress failed: %v", err)
	}

	if address != address2 {
		t.Errorf("Expected same address for same service, got %s and %s", address, address2)
	}

	// Test RegisterService
	err = discovery.RegisterService("registered-service", "localhost:60001")
	if err != nil {
		t.Errorf("RegisterService failed: %v", err)
	}

	address3, err := discovery.GetServiceAddress("registered-service")
	if err != nil {
		t.Errorf("GetServiceAddress failed: %v", err)
	}

	if address3 != "localhost:60001" {
		t.Errorf("Expected localhost:60001, got %s", address3)
	}

	// Test ListServices
	services, err := discovery.ListServices()
	if err != nil {
		t.Errorf("ListServices failed: %v", err)
	}

	if len(services) != 2 {
		t.Errorf("Expected 2 services, got %d", len(services))
	}

	// Test UnregisterService
	err = discovery.UnregisterService("new-service")
	if err != nil {
		t.Errorf("UnregisterService failed: %v", err)
	}

	services, err = discovery.ListServices()
	if err != nil {
		t.Errorf("ListServices failed: %v", err)
	}

	if len(services) != 1 {
		t.Errorf("Expected 1 service after unregister, got %d", len(services))
	}
}

func TestClient_Execute_Retry(t *testing.T) {
	discovery := NewTestServiceDiscovery()
	discovery.SetServiceAddress("retry-service", "localhost:50006")
	
	client, err := createTestClient(Options{
		ServiceDiscovery: discovery,
		MaxRetries:       3,
		RequestTimeout:   50 * time.Millisecond, // Short timeout for testing
	})
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	ctx := context.Background()
	execCtx := createTestExecutionContext()

	// Get connection and set request count to trigger failures
	conn, err := client.getConnection("retry-service")
	if err != nil {
		t.Fatalf("Failed to get connection: %v", err)
	}

	// Set request count so that the first few requests will fail
	conn.mutex.Lock()
	conn.RequestCount = 6 // Next request (7) will fail, then 8, 9 succeed, 17 fails
	conn.mutex.Unlock()

	// This should eventually succeed after retries
	response, err := client.Execute(ctx, "retry-service", "OBJECT", "METHOD", nil, execCtx)

	// Since we have retries, and the mock fails on request 7 but succeeds on 8,
	// the retry should succeed
	if err != nil {
		t.Errorf("Expected request to succeed after retry, got: %v", err)
	}

	if response == nil {
		t.Error("Expected response after successful retry")
	}
}

func TestClient_Execute_ContextCancellation(t *testing.T) {
	discovery := NewTestServiceDiscovery()
	discovery.SetServiceAddress("cancel-service", "localhost:50007")
	
	client, err := createTestClient(Options{
		ServiceDiscovery: discovery,
		RequestTimeout:   5 * time.Second, // Long timeout to test cancellation
	})
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	ctx, cancel := context.WithCancel(context.Background())
	execCtx := createTestExecutionContext()

	// Cancel context immediately
	cancel()

	_, err = client.Execute(ctx, "cancel-service", "OBJECT", "METHOD", nil, execCtx)

	if err == nil {
		t.Error("Expected error due to context cancellation")
	}

	if err != context.Canceled && !strings.Contains(err.Error(), "context canceled") {
		t.Errorf("Expected context cancellation error, got: %v", err)
	}
}

func TestClient_ConcurrentAccess(t *testing.T) {
	discovery := NewTestServiceDiscovery()
	discovery.SetServiceAddress("concurrent-service", "localhost:50008")
	
	client, err := createTestClient(Options{
		ServiceDiscovery: discovery,
	})
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	ctx := context.Background()
	execCtx := createTestExecutionContext()

	// Run multiple concurrent requests
	const numRequests = 10
	results := make(chan error, numRequests)

	for i := 0; i < numRequests; i++ {
		go func(id int) {
			execCtxCopy := *execCtx
			execCtxCopy.RequestID = fmt.Sprintf("concurrent-request-%d", id)
			
			_, err := client.Execute(ctx, "concurrent-service", "OBJECT", "METHOD", 
				map[string]interface{}{"id": id}, &execCtxCopy)
			results <- err
		}(i)
	}

	// Collect results
	var errors []error
	for i := 0; i < numRequests; i++ {
		if err := <-results; err != nil {
			errors = append(errors, err)
		}
	}

	if len(errors) > 0 {
		t.Errorf("Some concurrent requests failed: %v", errors)
	}

	// Check that only one connection was created
	client.mutex.RLock()
	connCount := len(client.connections)
	client.mutex.RUnlock()

	if connCount != 1 {
		t.Errorf("Expected 1 connection for concurrent requests, got %d", connCount)
	}
}

func TestServiceConnection_UpdateStats(t *testing.T) {
	conn := &ServiceConnection{
		ServiceName: "test-service",
		Address:     "localhost:50009",
	}

	// Test successful update
	conn.updateStats(true)
	
	conn.mutex.RLock()
	requestCount := conn.RequestCount
	errorCount := conn.ErrorCount
	conn.mutex.RUnlock()

	if requestCount != 1 {
		t.Errorf("Expected request count 1, got %d", requestCount)
	}

	if errorCount != 0 {
		t.Errorf("Expected error count 0, got %d", errorCount)
	}

	// Test error update
	conn.updateStats(false)
	
	conn.mutex.RLock()
	requestCount = conn.RequestCount
	errorCount = conn.ErrorCount
	conn.mutex.RUnlock()

	if requestCount != 2 {
		t.Errorf("Expected request count 2, got %d", requestCount)
	}

	if errorCount != 1 {
		t.Errorf("Expected error count 1, got %d", errorCount)
	}
}

// Benchmarks

func BenchmarkClient_Execute(b *testing.B) {
	discovery := NewTestServiceDiscovery()
	discovery.SetServiceAddress("bench-service", "localhost:50010")
	
	client, err := createTestClient(Options{
		ServiceDiscovery:    discovery,
		HealthCheckInterval: 1 * time.Hour, // Disable for benchmark
	})
	if err != nil {
		b.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	ctx := context.Background()
	execCtx := createTestExecutionContext()
	params := map[string]interface{}{
		"test": "benchmark",
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, err := client.Execute(ctx, "bench-service", "OBJECT", "METHOD", params, execCtx)
		if err != nil {
			b.Fatalf("Execute failed: %v", err)
		}
	}
}

func BenchmarkCircuitBreaker_AllowRequest(b *testing.B) {
	config := CircuitBreakerConfig{
		FailureThreshold:   5,
		RecoveryTimeout:    60 * time.Second,
		HalfOpenRequests:   3,
		MinRequestsToTrip:  10,
	}

	cb := NewCircuitBreaker(config)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		cb.AllowRequest()
	}
}

func BenchmarkServiceDiscovery_GetAddress(b *testing.B) {
	discovery := NewMockServiceDiscovery()
	discovery.RegisterService("test-service", "localhost:50011")

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, err := discovery.GetServiceAddress("test-service")
		if err != nil {
			b.Fatalf("GetServiceAddress failed: %v", err)
		}
	}
}

func BenchmarkClient_ConcurrentExecute(b *testing.B) {
	discovery := NewTestServiceDiscovery()
	discovery.SetServiceAddress("concurrent-bench-service", "localhost:50012")
	
	client, err := createTestClient(Options{
		ServiceDiscovery:    discovery,
		HealthCheckInterval: 1 * time.Hour, // Disable for benchmark
	})
	if err != nil {
		b.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	ctx := context.Background()
	execCtx := createTestExecutionContext()

	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, err := client.Execute(ctx, "concurrent-bench-service", "OBJECT", "METHOD", nil, execCtx)
			if err != nil {
				b.Fatalf("Execute failed: %v", err)
			}
		}
	})
}