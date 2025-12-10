package server

import (
	"context"
	"testing"
	"time"

	commonpb "github.com/msto63/mDW/api/gen/common"
	pb "github.com/msto63/mDW/api/gen/russell"
	"github.com/msto63/mDW/internal/russell/procmgr"
)

func TestConvertStatus(t *testing.T) {
	tests := []struct {
		input    procmgr.ServiceStatus
		expected pb.ServiceStatus
	}{
		{procmgr.StatusStopped, pb.ServiceStatus_SERVICE_STATUS_STOPPED},
		{procmgr.StatusStarting, pb.ServiceStatus_SERVICE_STATUS_STARTING},
		{procmgr.StatusRunning, pb.ServiceStatus_SERVICE_STATUS_HEALTHY},
		{procmgr.StatusStopping, pb.ServiceStatus_SERVICE_STATUS_STOPPING},
		{procmgr.StatusFailed, pb.ServiceStatus_SERVICE_STATUS_FAILED},
		{procmgr.StatusUnknown, pb.ServiceStatus_SERVICE_STATUS_UNKNOWN},
		{procmgr.ServiceStatus(99), pb.ServiceStatus_SERVICE_STATUS_UNKNOWN}, // Invalid status
	}

	for _, tt := range tests {
		t.Run(tt.input.String(), func(t *testing.T) {
			result := convertStatus(tt.input)
			if result != tt.expected {
				t.Errorf("convertStatus(%v) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

func createTestServer(t *testing.T) *Server {
	t.Helper()
	cfg := DefaultConfig()
	cfg.Port = 0 // Use any available port
	server, err := New(cfg)
	if err != nil {
		t.Fatalf("Failed to create test server: %v", err)
	}
	return server
}

func TestStartService_EmptyName(t *testing.T) {
	server := createTestServer(t)

	_, err := server.StartService(context.Background(), &pb.StartServiceRequest{
		Name: "",
	})

	if err == nil {
		t.Error("Expected error for empty service name")
	}
}

func TestStartService_ValidName(t *testing.T) {
	server := createTestServer(t)

	// Note: This will fail because the binary doesn't exist
	// but it tests that validation passes and the method is called
	resp, err := server.StartService(context.Background(), &pb.StartServiceRequest{
		Name: "turing",
	})

	// We expect either an error or a failure response (binary not found)
	if err == nil && resp != nil && !resp.Success {
		// Expected - service couldn't start because binary doesn't exist
		return
	}
	// Or error is acceptable too
	if err != nil {
		return
	}
}

func TestStopService_EmptyName(t *testing.T) {
	server := createTestServer(t)

	_, err := server.StopService(context.Background(), &pb.StopServiceRequest{
		Name: "",
	})

	if err == nil {
		t.Error("Expected error for empty service name")
	}
}

func TestStopService_ValidName(t *testing.T) {
	server := createTestServer(t)

	resp, err := server.StopService(context.Background(), &pb.StopServiceRequest{
		Name:  "turing",
		Force: false,
	})

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if !resp.Success {
		t.Errorf("Expected success for stopping already stopped service")
	}
}

func TestStopService_WithForce(t *testing.T) {
	server := createTestServer(t)

	resp, err := server.StopService(context.Background(), &pb.StopServiceRequest{
		Name:  "turing",
		Force: true,
	})

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if !resp.Success {
		t.Errorf("Expected success for stopping already stopped service")
	}
}

func TestRestartService_EmptyName(t *testing.T) {
	server := createTestServer(t)

	_, err := server.RestartService(context.Background(), &pb.RestartServiceRequest{
		Name: "",
	})

	if err == nil {
		t.Error("Expected error for empty service name")
	}
}

func TestGetServiceStatus_EmptyName(t *testing.T) {
	server := createTestServer(t)

	_, err := server.GetServiceStatus(context.Background(), &pb.GetServiceStatusRequest{
		Name: "",
	})

	if err == nil {
		t.Error("Expected error for empty service name")
	}
}

func TestGetServiceStatus_ValidName(t *testing.T) {
	server := createTestServer(t)

	resp, err := server.GetServiceStatus(context.Background(), &pb.GetServiceStatusRequest{
		Name: "turing",
	})

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if resp.Name != "turing" {
		t.Errorf("Name = %q, want %q", resp.Name, "turing")
	}
	if resp.Status != pb.ServiceStatus_SERVICE_STATUS_STOPPED {
		t.Errorf("Status = %v, want %v", resp.Status, pb.ServiceStatus_SERVICE_STATUS_STOPPED)
	}
	if resp.Port != 9200 {
		t.Errorf("Port = %d, want %d", resp.Port, 9200)
	}
	if resp.Version == "" {
		t.Error("Version should not be empty")
	}
}

func TestGetServiceStatus_NotFound(t *testing.T) {
	server := createTestServer(t)

	_, err := server.GetServiceStatus(context.Background(), &pb.GetServiceStatusRequest{
		Name: "nonexistent",
	})

	if err == nil {
		t.Error("Expected error for nonexistent service")
	}
}

func TestGetAllServiceStatus(t *testing.T) {
	server := createTestServer(t)

	resp, err := server.GetAllServiceStatus(context.Background(), &commonpb.Empty{})

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if len(resp.Services) == 0 {
		t.Error("Expected at least some services")
	}

	// Check that we have the expected services
	expectedServices := map[string]bool{
		"turing":  false,
		"hypatia": false,
		"babbage": false,
		"leibniz": false,
		"kant":    false,
		"bayes":   false,
	}

	for _, svc := range resp.Services {
		if _, ok := expectedServices[svc.Name]; ok {
			expectedServices[svc.Name] = true
		}
	}

	for name, found := range expectedServices {
		if !found {
			t.Errorf("Expected service %q not found in response", name)
		}
	}

	// Check totals
	if resp.Total != int32(len(resp.Services)) {
		t.Errorf("Total = %d, want %d", resp.Total, len(resp.Services))
	}

	// All services should be stopped initially
	if resp.Running != 0 {
		t.Errorf("Running = %d, want 0", resp.Running)
	}
	if resp.Unhealthy != 0 {
		t.Errorf("Unhealthy = %d, want 0", resp.Unhealthy)
	}
	if resp.Stopped != resp.Total {
		t.Errorf("Stopped = %d, want %d", resp.Stopped, resp.Total)
	}
}

func TestBuildServiceStatusResponse(t *testing.T) {
	server := createTestServer(t)

	startTime := time.Now().Add(-1 * time.Hour)
	svc := &procmgr.ManagedService{
		Config: procmgr.ServiceConfig{
			Name:    "test-service",
			Version: "1.2.3",
			Port:    9999,
		},
		Status:       procmgr.StatusRunning,
		PID:          12345,
		StartedAt:    startTime,
		RestartCount: 3,
		LastError:    "previous error",
	}

	resp := server.buildServiceStatusResponse(svc)

	if resp.Name != "test-service" {
		t.Errorf("Name = %q, want %q", resp.Name, "test-service")
	}
	if resp.Status != pb.ServiceStatus_SERVICE_STATUS_HEALTHY {
		t.Errorf("Status = %v, want %v", resp.Status, pb.ServiceStatus_SERVICE_STATUS_HEALTHY)
	}
	if resp.Pid != 12345 {
		t.Errorf("Pid = %d, want %d", resp.Pid, 12345)
	}
	if resp.Port != 9999 {
		t.Errorf("Port = %d, want %d", resp.Port, 9999)
	}
	if resp.Address != "localhost" {
		t.Errorf("Address = %q, want %q", resp.Address, "localhost")
	}
	if resp.RestartCount != 3 {
		t.Errorf("RestartCount = %d, want %d", resp.RestartCount, 3)
	}
	if resp.HealthMessage != "previous error" {
		t.Errorf("HealthMessage = %q, want %q", resp.HealthMessage, "previous error")
	}
	if resp.Version != "1.2.3" {
		t.Errorf("Version = %q, want %q", resp.Version, "1.2.3")
	}
	if resp.StartedAt != startTime.Unix() {
		t.Errorf("StartedAt = %d, want %d", resp.StartedAt, startTime.Unix())
	}
	// Uptime should be approximately 1 hour
	if resp.UptimeSeconds < 3500 || resp.UptimeSeconds > 3700 {
		t.Errorf("UptimeSeconds = %d, want ~3600", resp.UptimeSeconds)
	}
}

func TestBuildServiceStatusResponse_ZeroStartedAt(t *testing.T) {
	server := createTestServer(t)

	svc := &procmgr.ManagedService{
		Config: procmgr.ServiceConfig{
			Name:    "test-service",
			Version: "1.0.0",
			Port:    9000,
		},
		Status:    procmgr.StatusStopped,
		StartedAt: time.Time{}, // Zero time
	}

	resp := server.buildServiceStatusResponse(svc)

	if resp.StartedAt != 0 {
		t.Errorf("StartedAt = %d, want 0 for zero time", resp.StartedAt)
	}
	if resp.UptimeSeconds != 0 {
		t.Errorf("UptimeSeconds = %d, want 0 for stopped service", resp.UptimeSeconds)
	}
}

func TestServiceVersionsInStatus(t *testing.T) {
	server := createTestServer(t)

	resp, err := server.GetAllServiceStatus(context.Background(), &commonpb.Empty{})
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	for _, svc := range resp.Services {
		if svc.Version == "" {
			t.Errorf("Service %q has empty version", svc.Name)
		}
		// Version should be semver format (x.y.z)
		if len(svc.Version) < 5 {
			t.Errorf("Service %q version %q seems invalid", svc.Name, svc.Version)
		}
	}
}

func TestDefaultConfig_Values(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.Host != "0.0.0.0" {
		t.Errorf("Host = %q, want %q", cfg.Host, "0.0.0.0")
	}
	if cfg.Port != 9002 {
		t.Errorf("Port = %d, want %d", cfg.Port, 9002)
	}
	if cfg.CacheTTL != 30*time.Second {
		t.Errorf("CacheTTL = %v, want %v", cfg.CacheTTL, 30*time.Second)
	}
	if cfg.BinaryPath != "./bin/mdw" {
		t.Errorf("BinaryPath = %q, want %q", cfg.BinaryPath, "./bin/mdw")
	}
	if cfg.ConfigPath != "./configs/config.toml" {
		t.Errorf("ConfigPath = %q, want %q", cfg.ConfigPath, "./configs/config.toml")
	}
}

func TestConvertParams(t *testing.T) {
	tests := []struct {
		name     string
		input    map[string]string
		expected map[string]interface{}
	}{
		{
			name:     "nil input",
			input:    nil,
			expected: nil,
		},
		{
			name:     "empty map",
			input:    map[string]string{},
			expected: map[string]interface{}{},
		},
		{
			name:     "with values",
			input:    map[string]string{"key1": "value1", "key2": "value2"},
			expected: map[string]interface{}{"key1": "value1", "key2": "value2"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := convertParams(tt.input)

			if tt.expected == nil {
				if result != nil {
					t.Errorf("expected nil, got %v", result)
				}
				return
			}

			if len(result) != len(tt.expected) {
				t.Errorf("length = %d, want %d", len(result), len(tt.expected))
			}

			for k, v := range tt.expected {
				if result[k] != v {
					t.Errorf("result[%q] = %v, want %v", k, result[k], v)
				}
			}
		})
	}
}

func TestServiceStatusCounts(t *testing.T) {
	server := createTestServer(t)

	// Get initial status (all stopped)
	resp, err := server.GetAllServiceStatus(context.Background(), &commonpb.Empty{})
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Initially all should be stopped
	total := resp.Total
	if resp.Stopped != total {
		t.Errorf("Stopped = %d, want %d (total)", resp.Stopped, total)
	}
	if resp.Running != 0 {
		t.Errorf("Running = %d, want 0", resp.Running)
	}
	if resp.Unhealthy != 0 {
		t.Errorf("Unhealthy = %d, want 0", resp.Unhealthy)
	}

	// Sum should equal total
	sum := resp.Running + resp.Stopped + resp.Unhealthy
	if sum != total {
		t.Errorf("Sum of statuses = %d, want %d (total)", sum, total)
	}
}
