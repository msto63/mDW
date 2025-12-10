// ============================================================================
// meinDENKWERK (mDW) - Lokale KI-Plattform
// ============================================================================
//
// Package:     grpc
// Description: Integration tests for Russell gRPC service (Service Discovery)
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
	russellpb "github.com/msto63/mDW/api/gen/russell"
)

// RussellTestClient wraps the Russell gRPC client for testing
type RussellTestClient struct {
	conn   *TestConnection
	client russellpb.RussellServiceClient
}

// NewRussellTestClient creates a new Russell test client
func NewRussellTestClient() (*RussellTestClient, error) {
	configs := DefaultServiceConfigs()
	cfg := configs["russell"]

	conn, err := NewTestConnection(cfg)
	if err != nil {
		return nil, err
	}

	return &RussellTestClient{
		conn:   conn,
		client: russellpb.NewRussellServiceClient(conn.Conn()),
	}, nil
}

// Close closes the test client connection
func (rc *RussellTestClient) Close() error {
	return rc.conn.Close()
}

// Client returns the underlying gRPC client
func (rc *RussellTestClient) Client() russellpb.RussellServiceClient {
	return rc.client
}

// Context returns a context with the configured timeout
func (rc *RussellTestClient) Context() context.Context {
	return rc.conn.Context()
}

// ContextWithTimeout returns a context with a custom timeout
func (rc *RussellTestClient) ContextWithTimeout(timeout time.Duration) (context.Context, context.CancelFunc) {
	return rc.conn.ContextWithTimeout(timeout)
}

// TestRussellHealthCheck tests the health check endpoint
func TestRussellHealthCheck(t *testing.T) {
	client, err := NewRussellTestClient()
	if err != nil {
		t.Fatalf("Failed to create Russell client: %v", err)
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

	if resp.GetService() != "russell" {
		t.Errorf("Expected service 'russell', got '%s'", resp.GetService())
	}
}

// TestRussellListServices tests listing registered services
func TestRussellListServices(t *testing.T) {
	client, err := NewRussellTestClient()
	if err != nil {
		t.Fatalf("Failed to create Russell client: %v", err)
	}
	defer client.Close()

	ctx, cancel := client.ContextWithTimeout(10 * time.Second)
	defer cancel()

	resp, err := client.Client().ListServices(ctx, &common.Empty{})
	if err != nil {
		t.Fatalf("ListServices failed: %v", err)
	}

	t.Logf("Found %d services (total: %d):", len(resp.GetServices()), resp.GetTotal())
	for _, svc := range resp.GetServices() {
		t.Logf("  - %s (%s:%d) - status: %s",
			svc.GetName(), svc.GetAddress(), svc.GetPort(), svc.GetStatus().String())
	}
}

// TestRussellGetSystemHealth tests getting overall system health
func TestRussellGetSystemHealth(t *testing.T) {
	client, err := NewRussellTestClient()
	if err != nil {
		t.Fatalf("Failed to create Russell client: %v", err)
	}
	defer client.Close()

	ctx, cancel := client.ContextWithTimeout(10 * time.Second)
	defer cancel()

	resp, err := client.Client().GetSystemHealth(ctx, &common.Empty{})
	if err != nil {
		t.Fatalf("GetSystemHealth failed: %v", err)
	}

	t.Logf("System Health:")
	t.Logf("  Overall Status: %s", resp.GetOverallStatus())
	t.Logf("  Total Services: %d", len(resp.GetServices()))
	t.Logf("  Timestamp: %d", resp.GetTimestamp())

	for _, svcHealth := range resp.GetServices() {
		t.Logf("  - %s: %s", svcHealth.GetName(), svcHealth.GetStatus())
	}
}

// TestRussellGetSystemOverview tests getting system overview
func TestRussellGetSystemOverview(t *testing.T) {
	client, err := NewRussellTestClient()
	if err != nil {
		t.Fatalf("Failed to create Russell client: %v", err)
	}
	defer client.Close()

	ctx, cancel := client.ContextWithTimeout(10 * time.Second)
	defer cancel()

	resp, err := client.Client().GetSystemOverview(ctx, &common.Empty{})
	if err != nil {
		t.Fatalf("GetSystemOverview failed: %v", err)
	}

	t.Logf("System Overview:")
	t.Logf("  Timestamp: %s", resp.GetTimestamp())
	t.Logf("  Total Services: %d", resp.GetTotalServices())
	t.Logf("  Healthy Services: %d", resp.GetHealthyServices())
	t.Logf("  Degraded Services: %d", resp.GetDegradedServices())
	t.Logf("  Unhealthy Services: %d", resp.GetUnhealthyServices())
}

// TestRussellDiscover tests service discovery
func TestRussellDiscover(t *testing.T) {
	client, err := NewRussellTestClient()
	if err != nil {
		t.Fatalf("Failed to create Russell client: %v", err)
	}
	defer client.Close()

	tests := []struct {
		name        string
		serviceName string
	}{
		{"Discover Turing", "turing"},
		{"Discover Kant", "kant"},
		{"Discover Russell", "russell"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := client.ContextWithTimeout(10 * time.Second)
			defer cancel()

			req := &russellpb.DiscoverRequest{
				Name: tt.serviceName,
			}

			resp, err := client.Client().Discover(ctx, req)
			if err != nil {
				t.Logf("Discover %s: not found or error: %v", tt.serviceName, err)
				return
			}

			t.Logf("Discovered %s:", tt.serviceName)
			for _, svc := range resp.GetServices() {
				t.Logf("  - %s:%d (status: %s)", svc.GetAddress(), svc.GetPort(), svc.GetStatus().String())
			}
		})
	}
}

// TestRussellGetService tests getting a specific service
func TestRussellGetService(t *testing.T) {
	client, err := NewRussellTestClient()
	if err != nil {
		t.Fatalf("Failed to create Russell client: %v", err)
	}
	defer client.Close()

	// First, list services to get a valid service ID
	ctx, cancel := client.ContextWithTimeout(10 * time.Second)
	defer cancel()

	listResp, err := client.Client().ListServices(ctx, &common.Empty{})
	if err != nil {
		t.Skipf("ListServices failed: %v", err)
	}

	if len(listResp.GetServices()) == 0 {
		t.Skip("No services registered to test GetService")
	}

	// Get the first service by ID
	firstService := listResp.GetServices()[0]
	req := &russellpb.GetServiceRequest{
		Id: firstService.GetId(),
	}

	resp, err := client.Client().GetService(ctx, req)
	if err != nil {
		t.Logf("GetService %s: error: %v", firstService.GetId(), err)
		return
	}

	t.Logf("Service Info:")
	t.Logf("  ID: %s", resp.GetId())
	t.Logf("  Name: %s", resp.GetName())
	t.Logf("  Address: %s", resp.GetAddress())
	t.Logf("  Port: %d", resp.GetPort())
	t.Logf("  Status: %s", resp.GetStatus().String())
}

// TestRussellRegisterDeregister tests service registration and deregistration
func TestRussellRegisterDeregister(t *testing.T) {
	client, err := NewRussellTestClient()
	if err != nil {
		t.Fatalf("Failed to create Russell client: %v", err)
	}
	defer client.Close()

	ctx, cancel := client.ContextWithTimeout(10 * time.Second)
	defer cancel()

	// Register a test service
	registerReq := &russellpb.RegisterRequest{
		Name:    "test-service",
		Address: "localhost",
		Port:    9999,
		Tags:    []string{"test", "integration"},
	}

	registerResp, err := client.Client().Register(ctx, registerReq)
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	t.Logf("Registered service:")
	t.Logf("  Service ID: %s", registerResp.GetId())
	t.Logf("  Success: %v", registerResp.GetSuccess())
	t.Logf("  Message: %s", registerResp.GetMessage())

	if !registerResp.GetSuccess() {
		t.Errorf("Registration was not successful")
	}

	// Verify the service is registered
	listResp, err := client.Client().ListServices(ctx, &common.Empty{})
	if err != nil {
		t.Fatalf("ListServices failed: %v", err)
	}

	found := false
	for _, svc := range listResp.GetServices() {
		if svc.GetName() == "test-service" {
			found = true
			t.Logf("Found test-service in service list")
			break
		}
	}

	if !found {
		t.Error("test-service not found in service list after registration")
	}

	// Deregister the test service
	deregisterReq := &russellpb.DeregisterRequest{
		Id: registerResp.GetId(),
	}

	_, err = client.Client().Deregister(ctx, deregisterReq)
	if err != nil {
		t.Fatalf("Deregister failed: %v", err)
	}

	t.Log("Deregistered test-service successfully")

	// Verify the service is no longer registered
	listResp, err = client.Client().ListServices(ctx, &common.Empty{})
	if err != nil {
		t.Fatalf("ListServices failed: %v", err)
	}

	for _, svc := range listResp.GetServices() {
		if svc.GetName() == "test-service" {
			t.Error("test-service still found after deregistration")
		}
	}
}

// TestRussellHeartbeat tests the heartbeat mechanism
func TestRussellHeartbeat(t *testing.T) {
	client, err := NewRussellTestClient()
	if err != nil {
		t.Fatalf("Failed to create Russell client: %v", err)
	}
	defer client.Close()

	ctx, cancel := client.ContextWithTimeout(10 * time.Second)
	defer cancel()

	// First register a service
	registerReq := &russellpb.RegisterRequest{
		Name:    "heartbeat-test-service",
		Address: "localhost",
		Port:    9998,
	}

	registerResp, err := client.Client().Register(ctx, registerReq)
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	defer func() {
		// Clean up: deregister the service
		deregisterReq := &russellpb.DeregisterRequest{
			Id: registerResp.GetId(),
		}
		client.Client().Deregister(ctx, deregisterReq)
	}()

	// Send a heartbeat
	heartbeatReq := &russellpb.HeartbeatRequest{
		Id: registerResp.GetId(),
	}

	heartbeatResp, err := client.Client().Heartbeat(ctx, heartbeatReq)
	if err != nil {
		t.Fatalf("Heartbeat failed: %v", err)
	}

	t.Logf("Heartbeat Response:")
	t.Logf("  Acknowledged: %v", heartbeatResp.GetAcknowledged())
	t.Logf("  Next Heartbeat In: %d ms", heartbeatResp.GetNextHeartbeatMs())

	if !heartbeatResp.GetAcknowledged() {
		t.Error("Heartbeat was not acknowledged")
	}
}

// TestRussellGetMetrics tests getting system metrics
func TestRussellGetMetrics(t *testing.T) {
	client, err := NewRussellTestClient()
	if err != nil {
		t.Fatalf("Failed to create Russell client: %v", err)
	}
	defer client.Close()

	ctx, cancel := client.ContextWithTimeout(10 * time.Second)
	defer cancel()

	resp, err := client.Client().GetMetrics(ctx, &common.Empty{})
	if err != nil {
		t.Logf("GetMetrics: %v (metrics may not be implemented)", err)
		return
	}

	t.Logf("System Metrics: %v", resp)
}

// TestRussellListPipelines tests listing pipelines
func TestRussellListPipelines(t *testing.T) {
	client, err := NewRussellTestClient()
	if err != nil {
		t.Fatalf("Failed to create Russell client: %v", err)
	}
	defer client.Close()

	ctx, cancel := client.ContextWithTimeout(10 * time.Second)
	defer cancel()

	resp, err := client.Client().ListPipelines(ctx, &common.Empty{})
	if err != nil {
		t.Logf("ListPipelines: %v (pipelines may not be implemented)", err)
		return
	}

	t.Logf("Found %d pipelines", len(resp.GetPipelines()))
	for _, pipeline := range resp.GetPipelines() {
		t.Logf("  - %s (ID: %s)", pipeline.GetName(), pipeline.GetId())
	}
}

// ============================================================================
// Service Control Integration Tests (Process Management)
// ============================================================================

// TestRussellGetAllServiceStatus tests getting status of all services
func TestRussellGetAllServiceStatus(t *testing.T) {
	client, err := NewRussellTestClient()
	if err != nil {
		t.Fatalf("Failed to create Russell client: %v", err)
	}
	defer client.Close()

	ctx, cancel := client.ContextWithTimeout(10 * time.Second)
	defer cancel()

	resp, err := client.Client().GetAllServiceStatus(ctx, &common.Empty{})
	if err != nil {
		t.Fatalf("GetAllServiceStatus failed: %v", err)
	}

	t.Logf("Service Status Overview:")
	t.Logf("  Total: %d", resp.GetTotal())
	t.Logf("  Running: %d", resp.GetRunning())
	t.Logf("  Stopped: %d", resp.GetStopped())
	t.Logf("  Unhealthy: %d", resp.GetUnhealthy())
	t.Logf("")

	expectedServices := map[string]bool{
		"turing":  false,
		"hypatia": false,
		"babbage": false,
		"leibniz": false,
		"kant":    false,
		"bayes":   false,
	}

	for _, svc := range resp.GetServices() {
		t.Logf("  %s:", svc.GetName())
		t.Logf("    Status: %s", svc.GetStatus().String())
		t.Logf("    Version: %s", svc.GetVersion())
		t.Logf("    Port: %d", svc.GetPort())
		t.Logf("    PID: %d", svc.GetPid())
		t.Logf("    Uptime: %d seconds", svc.GetUptimeSeconds())
		t.Logf("    RestartCount: %d", svc.GetRestartCount())

		if _, ok := expectedServices[svc.GetName()]; ok {
			expectedServices[svc.GetName()] = true
		}

		// Verify version is present
		if svc.GetVersion() == "" {
			t.Errorf("Service %s has empty version", svc.GetName())
		}
	}

	// Check all expected services are present
	for name, found := range expectedServices {
		if !found {
			t.Errorf("Expected service %q not found in status response", name)
		}
	}

	// Verify totals match
	if int(resp.GetTotal()) != len(resp.GetServices()) {
		t.Errorf("Total (%d) doesn't match service count (%d)", resp.GetTotal(), len(resp.GetServices()))
	}
}

// TestRussellGetServiceStatus tests getting status of a specific service
func TestRussellGetServiceStatus(t *testing.T) {
	client, err := NewRussellTestClient()
	if err != nil {
		t.Fatalf("Failed to create Russell client: %v", err)
	}
	defer client.Close()

	tests := []struct {
		name        string
		serviceName string
		wantPort    int32
	}{
		{"Turing status", "turing", 9200},
		{"Hypatia status", "hypatia", 9220},
		{"Babbage status", "babbage", 9150},
		{"Leibniz status", "leibniz", 9140},
		{"Kant status", "kant", 8080},
		{"Bayes status", "bayes", 9120},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := client.ContextWithTimeout(10 * time.Second)
			defer cancel()

			resp, err := client.Client().GetServiceStatus(ctx, &russellpb.GetServiceStatusRequest{
				Name: tt.serviceName,
			})
			if err != nil {
				t.Fatalf("GetServiceStatus failed for %s: %v", tt.serviceName, err)
			}

			if resp.GetName() != tt.serviceName {
				t.Errorf("Name = %q, want %q", resp.GetName(), tt.serviceName)
			}
			if resp.GetPort() != tt.wantPort {
				t.Errorf("Port = %d, want %d", resp.GetPort(), tt.wantPort)
			}
			if resp.GetVersion() == "" {
				t.Error("Version should not be empty")
			}

			t.Logf("Service %s: status=%s, version=%s, port=%d",
				resp.GetName(), resp.GetStatus().String(), resp.GetVersion(), resp.GetPort())
		})
	}
}

// TestRussellGetServiceStatus_InvalidName tests error handling for invalid service name
func TestRussellGetServiceStatus_InvalidName(t *testing.T) {
	client, err := NewRussellTestClient()
	if err != nil {
		t.Fatalf("Failed to create Russell client: %v", err)
	}
	defer client.Close()

	tests := []struct {
		name        string
		serviceName string
	}{
		{"Empty name", ""},
		{"Nonexistent service", "nonexistent-service"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := client.ContextWithTimeout(10 * time.Second)
			defer cancel()

			_, err := client.Client().GetServiceStatus(ctx, &russellpb.GetServiceStatusRequest{
				Name: tt.serviceName,
			})
			if err == nil {
				t.Errorf("Expected error for service name %q", tt.serviceName)
			} else {
				t.Logf("Got expected error for %q: %v", tt.serviceName, err)
			}
		})
	}
}

// TestRussellStopService_AlreadyStopped tests stopping an already stopped service
func TestRussellStopService_AlreadyStopped(t *testing.T) {
	client, err := NewRussellTestClient()
	if err != nil {
		t.Fatalf("Failed to create Russell client: %v", err)
	}
	defer client.Close()

	ctx, cancel := client.ContextWithTimeout(10 * time.Second)
	defer cancel()

	// Stopping an already stopped service should succeed
	resp, err := client.Client().StopService(ctx, &russellpb.StopServiceRequest{
		Name:  "turing",
		Force: false,
	})
	if err != nil {
		t.Fatalf("StopService failed: %v", err)
	}

	if !resp.GetSuccess() {
		t.Errorf("Expected success when stopping already stopped service")
	}

	t.Logf("StopService response: success=%v, message=%s", resp.GetSuccess(), resp.GetMessage())
}

// TestRussellStopService_InvalidName tests error handling for invalid service name
func TestRussellStopService_InvalidName(t *testing.T) {
	client, err := NewRussellTestClient()
	if err != nil {
		t.Fatalf("Failed to create Russell client: %v", err)
	}
	defer client.Close()

	ctx, cancel := client.ContextWithTimeout(10 * time.Second)
	defer cancel()

	_, err = client.Client().StopService(ctx, &russellpb.StopServiceRequest{
		Name: "",
	})
	if err == nil {
		t.Error("Expected error for empty service name")
	} else {
		t.Logf("Got expected error for empty name: %v", err)
	}
}

// TestRussellStartService_InvalidName tests error handling for invalid service name
func TestRussellStartService_InvalidName(t *testing.T) {
	client, err := NewRussellTestClient()
	if err != nil {
		t.Fatalf("Failed to create Russell client: %v", err)
	}
	defer client.Close()

	ctx, cancel := client.ContextWithTimeout(10 * time.Second)
	defer cancel()

	_, err = client.Client().StartService(ctx, &russellpb.StartServiceRequest{
		Name: "",
	})
	if err == nil {
		t.Error("Expected error for empty service name")
	} else {
		t.Logf("Got expected error for empty name: %v", err)
	}
}

// TestRussellRestartService_InvalidName tests error handling for invalid service name
func TestRussellRestartService_InvalidName(t *testing.T) {
	client, err := NewRussellTestClient()
	if err != nil {
		t.Fatalf("Failed to create Russell client: %v", err)
	}
	defer client.Close()

	ctx, cancel := client.ContextWithTimeout(10 * time.Second)
	defer cancel()

	_, err = client.Client().RestartService(ctx, &russellpb.RestartServiceRequest{
		Name: "",
	})
	if err == nil {
		t.Error("Expected error for empty service name")
	} else {
		t.Logf("Got expected error for empty name: %v", err)
	}
}

// TestRussellServiceVersions tests that all services have valid version numbers
func TestRussellServiceVersions(t *testing.T) {
	client, err := NewRussellTestClient()
	if err != nil {
		t.Fatalf("Failed to create Russell client: %v", err)
	}
	defer client.Close()

	ctx, cancel := client.ContextWithTimeout(10 * time.Second)
	defer cancel()

	resp, err := client.Client().GetAllServiceStatus(ctx, &common.Empty{})
	if err != nil {
		t.Fatalf("GetAllServiceStatus failed: %v", err)
	}

	for _, svc := range resp.GetServices() {
		version := svc.GetVersion()

		// Version should not be empty
		if version == "" {
			t.Errorf("Service %s has empty version", svc.GetName())
			continue
		}

		// Version should be semver format (basic check: contains at least two dots)
		dots := 0
		for _, c := range version {
			if c == '.' {
				dots++
			}
		}
		if dots < 2 {
			t.Errorf("Service %s version %q doesn't look like semver", svc.GetName(), version)
		}

		t.Logf("Service %s version: %s", svc.GetName(), version)
	}
}

// TestRussellRegisterWithVersion tests registering a service with a version
func TestRussellRegisterWithVersion(t *testing.T) {
	client, err := NewRussellTestClient()
	if err != nil {
		t.Fatalf("Failed to create Russell client: %v", err)
	}
	defer client.Close()

	ctx, cancel := client.ContextWithTimeout(10 * time.Second)
	defer cancel()

	// Register a test service with version
	registerReq := &russellpb.RegisterRequest{
		Name:    "version-test-service",
		Address: "localhost",
		Port:    9997,
		Version: "2.3.4",
		Tags:    []string{"test", "version"},
	}

	registerResp, err := client.Client().Register(ctx, registerReq)
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	defer func() {
		// Clean up: deregister the service
		deregisterReq := &russellpb.DeregisterRequest{
			Id: registerResp.GetId(),
		}
		client.Client().Deregister(ctx, deregisterReq)
	}()

	t.Logf("Registered service with version:")
	t.Logf("  Service ID: %s", registerResp.GetId())
	t.Logf("  Success: %v", registerResp.GetSuccess())

	if !registerResp.GetSuccess() {
		t.Errorf("Registration was not successful")
	}

	// Verify the service is registered with the correct version
	listResp, err := client.Client().ListServices(ctx, &common.Empty{})
	if err != nil {
		t.Fatalf("ListServices failed: %v", err)
	}

	found := false
	for _, svc := range listResp.GetServices() {
		if svc.GetName() == "version-test-service" {
			found = true
			if svc.GetVersion() != "2.3.4" {
				t.Errorf("Version = %q, want %q", svc.GetVersion(), "2.3.4")
			} else {
				t.Logf("Found version-test-service with correct version: %s", svc.GetVersion())
			}
			break
		}
	}

	if !found {
		t.Error("version-test-service not found in service list after registration")
	}
}

// RunRussellTestSuite runs all Russell tests and returns a test suite with results
func RunRussellTestSuite(t *testing.T) *TestSuite {
	suite := NewTestSuite("russell")

	tests := []struct {
		name string
		fn   func(*testing.T)
	}{
		{"HealthCheck", TestRussellHealthCheck},
		{"ListServices", TestRussellListServices},
		{"GetSystemHealth", TestRussellGetSystemHealth},
		{"GetSystemOverview", TestRussellGetSystemOverview},
		{"Discover", TestRussellDiscover},
		{"GetService", TestRussellGetService},
		{"RegisterDeregister", TestRussellRegisterDeregister},
		{"Heartbeat", TestRussellHeartbeat},
		{"GetMetrics", TestRussellGetMetrics},
		{"ListPipelines", TestRussellListPipelines},
		// Service Control Tests
		{"GetAllServiceStatus", TestRussellGetAllServiceStatus},
		{"GetServiceStatus", TestRussellGetServiceStatus},
		{"GetServiceStatus_InvalidName", TestRussellGetServiceStatus_InvalidName},
		{"StopService_AlreadyStopped", TestRussellStopService_AlreadyStopped},
		{"StopService_InvalidName", TestRussellStopService_InvalidName},
		{"StartService_InvalidName", TestRussellStartService_InvalidName},
		{"RestartService_InvalidName", TestRussellRestartService_InvalidName},
		{"ServiceVersions", TestRussellServiceVersions},
		{"RegisterWithVersion", TestRussellRegisterWithVersion},
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
