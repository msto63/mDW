package procmgr

import (
	"context"
	"testing"
	"time"
)

func TestServiceStatus_String(t *testing.T) {
	tests := []struct {
		status   ServiceStatus
		expected string
	}{
		{StatusUnknown, "unknown"},
		{StatusStopped, "stopped"},
		{StatusStarting, "starting"},
		{StatusRunning, "running"},
		{StatusStopping, "stopping"},
		{StatusFailed, "failed"},
		{ServiceStatus(99), "unknown"}, // Invalid status
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := tt.status.String()
			if result != tt.expected {
				t.Errorf("ServiceStatus(%d).String() = %q, want %q", tt.status, result, tt.expected)
			}
		})
	}
}

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.BinaryPath != "./bin/mdw" {
		t.Errorf("BinaryPath = %q, want %q", cfg.BinaryPath, "./bin/mdw")
	}
	if cfg.ConfigPath != "./configs/config.toml" {
		t.Errorf("ConfigPath = %q, want %q", cfg.ConfigPath, "./configs/config.toml")
	}
}

func TestNew_RegistersKnownServices(t *testing.T) {
	pm := New(DefaultConfig())
	defer pm.Close()

	expectedServices := []string{"turing", "hypatia", "babbage", "leibniz", "kant", "bayes"}

	for _, name := range expectedServices {
		svc, err := pm.GetServiceStatus(name)
		if err != nil {
			t.Errorf("Expected service %q to be registered, got error: %v", name, err)
			continue
		}
		if svc == nil {
			t.Errorf("Expected service %q to be registered, got nil", name)
			continue
		}

		status, _, _, _, _ := svc.GetStatus()
		if status != StatusStopped {
			t.Errorf("Service %q initial status = %v, want %v", name, status, StatusStopped)
		}
	}
}

func TestNew_ServiceVersions(t *testing.T) {
	pm := New(DefaultConfig())
	defer pm.Close()

	services := pm.GetAllServiceStatus()

	for name, svc := range services {
		cfg := svc.GetConfig()
		if cfg.Version == "" {
			t.Errorf("Service %q has empty version", name)
		}
		// All versions should be semver format
		if len(cfg.Version) < 5 { // Minimum "x.y.z"
			t.Errorf("Service %q version %q seems invalid", name, cfg.Version)
		}
	}
}

func TestNew_ServicePorts(t *testing.T) {
	pm := New(DefaultConfig())
	defer pm.Close()

	expectedPorts := map[string]int{
		"turing":  9200,
		"hypatia": 9220,
		"babbage": 9150,
		"leibniz": 9140,
		"kant":    8080,
		"bayes":   9120,
	}

	for name, expectedPort := range expectedPorts {
		svc, err := pm.GetServiceStatus(name)
		if err != nil {
			t.Errorf("Service %q not found: %v", name, err)
			continue
		}

		cfg := svc.GetConfig()
		if cfg.Port != expectedPort {
			t.Errorf("Service %q port = %d, want %d", name, cfg.Port, expectedPort)
		}
	}
}

func TestGetServiceStatus_NotFound(t *testing.T) {
	pm := New(DefaultConfig())
	defer pm.Close()

	_, err := pm.GetServiceStatus("nonexistent")
	if err == nil {
		t.Error("Expected error for nonexistent service")
	}
}

func TestGetAllServiceStatus(t *testing.T) {
	pm := New(DefaultConfig())
	defer pm.Close()

	services := pm.GetAllServiceStatus()

	if len(services) == 0 {
		t.Error("Expected at least some services")
	}

	// Should return a copy, not the original map
	services["test"] = &ManagedService{}
	services2 := pm.GetAllServiceStatus()
	if _, exists := services2["test"]; exists {
		t.Error("GetAllServiceStatus should return a copy")
	}
}

func TestManagedService_GetStatus(t *testing.T) {
	svc := &ManagedService{
		Config: ServiceConfig{
			Name:    "test",
			Version: "1.0.0",
		},
		Status:       StatusRunning,
		PID:          1234,
		StartedAt:    time.Now().Add(-1 * time.Hour),
		RestartCount: 3,
		LastError:    "some error",
	}

	status, pid, startedAt, restartCount, lastError := svc.GetStatus()

	if status != StatusRunning {
		t.Errorf("status = %v, want %v", status, StatusRunning)
	}
	if pid != 1234 {
		t.Errorf("pid = %d, want %d", pid, 1234)
	}
	if startedAt.IsZero() {
		t.Error("startedAt should not be zero")
	}
	if restartCount != 3 {
		t.Errorf("restartCount = %d, want %d", restartCount, 3)
	}
	if lastError != "some error" {
		t.Errorf("lastError = %q, want %q", lastError, "some error")
	}
}

func TestManagedService_GetConfig(t *testing.T) {
	svc := &ManagedService{
		Config: ServiceConfig{
			Name:        "test-service",
			Version:     "2.0.0",
			Port:        9999,
			StartupTime: 30 * time.Second,
		},
	}

	cfg := svc.GetConfig()

	if cfg.Name != "test-service" {
		t.Errorf("Name = %q, want %q", cfg.Name, "test-service")
	}
	if cfg.Version != "2.0.0" {
		t.Errorf("Version = %q, want %q", cfg.Version, "2.0.0")
	}
	if cfg.Port != 9999 {
		t.Errorf("Port = %d, want %d", cfg.Port, 9999)
	}
	if cfg.StartupTime != 30*time.Second {
		t.Errorf("StartupTime = %v, want %v", cfg.StartupTime, 30*time.Second)
	}
}

func TestManagedService_Uptime_Stopped(t *testing.T) {
	svc := &ManagedService{
		Status:    StatusStopped,
		StartedAt: time.Now().Add(-1 * time.Hour),
	}

	uptime := svc.Uptime()
	if uptime != 0 {
		t.Errorf("Uptime for stopped service = %d, want 0", uptime)
	}
}

func TestManagedService_Uptime_Running(t *testing.T) {
	svc := &ManagedService{
		Status:    StatusRunning,
		StartedAt: time.Now().Add(-10 * time.Second),
	}

	uptime := svc.Uptime()
	if uptime < 9 || uptime > 11 {
		t.Errorf("Uptime = %d, want ~10", uptime)
	}
}

func TestManagedService_Uptime_ZeroStartedAt(t *testing.T) {
	svc := &ManagedService{
		Status:    StatusRunning,
		StartedAt: time.Time{}, // Zero time
	}

	uptime := svc.Uptime()
	if uptime != 0 {
		t.Errorf("Uptime with zero StartedAt = %d, want 0", uptime)
	}
}

func TestSubscribe_Unsubscribe(t *testing.T) {
	pm := New(DefaultConfig())
	defer pm.Close()

	ch := pm.Subscribe()
	if ch == nil {
		t.Fatal("Subscribe returned nil channel")
	}

	// Check subscriber was added
	pm.subscriberMu.RLock()
	subCount := len(pm.subscribers)
	pm.subscriberMu.RUnlock()

	if subCount != 1 {
		t.Errorf("subscriber count = %d, want 1", subCount)
	}

	// Unsubscribe
	pm.Unsubscribe(ch)

	pm.subscriberMu.RLock()
	subCount = len(pm.subscribers)
	pm.subscriberMu.RUnlock()

	if subCount != 0 {
		t.Errorf("subscriber count after unsubscribe = %d, want 0", subCount)
	}
}

func TestUnsubscribe_NotSubscribed(t *testing.T) {
	pm := New(DefaultConfig())
	defer pm.Close()

	ch := make(chan StatusEvent, 10)

	// Should not panic
	pm.Unsubscribe(ch)
}

func TestEmitEvent(t *testing.T) {
	pm := New(DefaultConfig())
	defer pm.Close()

	ch := pm.Subscribe()

	// Emit an event
	pm.emitEvent("test-service", StatusStopped, StatusStarting, "Starting service")

	// Should receive the event
	select {
	case event := <-ch:
		if event.ServiceName != "test-service" {
			t.Errorf("ServiceName = %q, want %q", event.ServiceName, "test-service")
		}
		if event.PreviousStatus != StatusStopped {
			t.Errorf("PreviousStatus = %v, want %v", event.PreviousStatus, StatusStopped)
		}
		if event.CurrentStatus != StatusStarting {
			t.Errorf("CurrentStatus = %v, want %v", event.CurrentStatus, StatusStarting)
		}
		if event.Message != "Starting service" {
			t.Errorf("Message = %q, want %q", event.Message, "Starting service")
		}
		if event.Timestamp.IsZero() {
			t.Error("Timestamp should not be zero")
		}
	case <-time.After(1 * time.Second):
		t.Error("Did not receive event within timeout")
	}

	pm.Unsubscribe(ch)
}

func TestStartService_NotFound(t *testing.T) {
	pm := New(DefaultConfig())
	defer pm.Close()

	err := pm.StartService(context.Background(), "nonexistent")
	if err == nil {
		t.Error("Expected error for nonexistent service")
	}
}

func TestStopService_NotFound(t *testing.T) {
	pm := New(DefaultConfig())
	defer pm.Close()

	err := pm.StopService(context.Background(), "nonexistent", false)
	if err == nil {
		t.Error("Expected error for nonexistent service")
	}
}

func TestStopService_AlreadyStopped(t *testing.T) {
	pm := New(DefaultConfig())
	defer pm.Close()

	// All services start as stopped
	err := pm.StopService(context.Background(), "turing", false)
	if err != nil {
		t.Errorf("Stopping already stopped service should not error: %v", err)
	}
}

func TestRestartService_NotFound(t *testing.T) {
	pm := New(DefaultConfig())
	defer pm.Close()

	err := pm.RestartService(context.Background(), "nonexistent")
	if err == nil {
		t.Error("Expected error for nonexistent service")
	}
}

func TestStartService_AlreadyRunning(t *testing.T) {
	pm := New(DefaultConfig())
	defer pm.Close()

	// Manually set a service to running
	pm.mu.Lock()
	svc := pm.services["turing"]
	pm.mu.Unlock()

	svc.mu.Lock()
	svc.Status = StatusRunning
	svc.mu.Unlock()

	err := pm.StartService(context.Background(), "turing")
	if err == nil {
		t.Error("Expected error when starting already running service")
	}
}

func TestStartService_AlreadyStarting(t *testing.T) {
	pm := New(DefaultConfig())
	defer pm.Close()

	// Manually set a service to starting
	pm.mu.Lock()
	svc := pm.services["turing"]
	pm.mu.Unlock()

	svc.mu.Lock()
	svc.Status = StatusStarting
	svc.mu.Unlock()

	err := pm.StartService(context.Background(), "turing")
	if err == nil {
		t.Error("Expected error when starting already starting service")
	}
}

func TestClose(t *testing.T) {
	pm := New(DefaultConfig())

	ch := pm.Subscribe()

	pm.Close()

	// Channel should be closed
	select {
	case _, ok := <-ch:
		if ok {
			t.Error("Channel should be closed after Close()")
		}
	default:
		// Wait a moment for async close
		time.Sleep(100 * time.Millisecond)
	}
}

func TestStatusEvent_Fields(t *testing.T) {
	event := StatusEvent{
		ServiceName:    "test",
		PreviousStatus: StatusStopped,
		CurrentStatus:  StatusRunning,
		Message:        "Service started",
		Timestamp:      time.Now(),
	}

	if event.ServiceName != "test" {
		t.Errorf("ServiceName = %q, want %q", event.ServiceName, "test")
	}
	if event.PreviousStatus != StatusStopped {
		t.Errorf("PreviousStatus = %v, want %v", event.PreviousStatus, StatusStopped)
	}
	if event.CurrentStatus != StatusRunning {
		t.Errorf("CurrentStatus = %v, want %v", event.CurrentStatus, StatusRunning)
	}
	if event.Message != "Service started" {
		t.Errorf("Message = %q, want %q", event.Message, "Service started")
	}
}

func TestServiceConfig_Fields(t *testing.T) {
	cfg := ServiceConfig{
		Name:        "my-service",
		Version:     "1.2.3",
		Command:     "/usr/bin/myservice",
		Args:        []string{"--flag", "value"},
		Env:         map[string]string{"KEY": "VALUE"},
		Port:        8080,
		StartupTime: 15 * time.Second,
	}

	if cfg.Name != "my-service" {
		t.Errorf("Name = %q, want %q", cfg.Name, "my-service")
	}
	if cfg.Version != "1.2.3" {
		t.Errorf("Version = %q, want %q", cfg.Version, "1.2.3")
	}
	if cfg.Command != "/usr/bin/myservice" {
		t.Errorf("Command = %q, want %q", cfg.Command, "/usr/bin/myservice")
	}
	if len(cfg.Args) != 2 {
		t.Errorf("Args length = %d, want 2", len(cfg.Args))
	}
	if cfg.Env["KEY"] != "VALUE" {
		t.Errorf("Env[KEY] = %q, want %q", cfg.Env["KEY"], "VALUE")
	}
	if cfg.Port != 8080 {
		t.Errorf("Port = %d, want %d", cfg.Port, 8080)
	}
	if cfg.StartupTime != 15*time.Second {
		t.Errorf("StartupTime = %v, want %v", cfg.StartupTime, 15*time.Second)
	}
}
