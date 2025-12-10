package health

import (
	"context"
	"sync/atomic"
	"testing"
	"time"
)

func TestStatus_Constants(t *testing.T) {
	if StatusHealthy != "healthy" {
		t.Errorf("StatusHealthy = %v, want healthy", StatusHealthy)
	}
	if StatusUnhealthy != "unhealthy" {
		t.Errorf("StatusUnhealthy = %v, want unhealthy", StatusUnhealthy)
	}
	if StatusDegraded != "degraded" {
		t.Errorf("StatusDegraded = %v, want degraded", StatusDegraded)
	}
	if StatusUnknown != "unknown" {
		t.Errorf("StatusUnknown = %v, want unknown", StatusUnknown)
	}
}

func TestNewChecker(t *testing.T) {
	checker := NewChecker("test-checker", func(ctx context.Context) CheckResult {
		return CheckResult{
			Status:  StatusHealthy,
			Message: "test passed",
		}
	})

	if checker.Name() != "test-checker" {
		t.Errorf("Name() = %v, want test-checker", checker.Name())
	}

	result := checker.Check(context.Background())
	if result.Status != StatusHealthy {
		t.Errorf("Status = %v, want healthy", result.Status)
	}
	if result.Message != "test passed" {
		t.Errorf("Message = %v, want 'test passed'", result.Message)
	}
}

func TestCheckFunc(t *testing.T) {
	fn := CheckFunc(func(ctx context.Context) CheckResult {
		return CheckResult{Status: StatusHealthy}
	})

	if fn.Name() != "unknown" {
		t.Errorf("Name() = %v, want unknown", fn.Name())
	}

	result := fn.Check(context.Background())
	if result.Status != StatusHealthy {
		t.Errorf("Status = %v, want healthy", result.Status)
	}
}

func TestRegistry_RegisterAndCheck(t *testing.T) {
	registry := NewRegistry("test-service", "1.0.0")

	checker1 := NewChecker("db", func(ctx context.Context) CheckResult {
		return CheckResult{Status: StatusHealthy, Message: "DB connected"}
	})
	checker2 := NewChecker("cache", func(ctx context.Context) CheckResult {
		return CheckResult{Status: StatusHealthy, Message: "Cache available"}
	})

	registry.Register(checker1)
	registry.Register(checker2)

	report := registry.Check(context.Background())

	if report.Service != "test-service" {
		t.Errorf("Service = %v, want test-service", report.Service)
	}
	if report.Version != "1.0.0" {
		t.Errorf("Version = %v, want 1.0.0", report.Version)
	}
	if report.Status != StatusHealthy {
		t.Errorf("Status = %v, want healthy", report.Status)
	}
	if len(report.Checks) != 2 {
		t.Errorf("Checks count = %v, want 2", len(report.Checks))
	}
}

func TestRegistry_RegisterFunc(t *testing.T) {
	registry := NewRegistry("test-service", "1.0.0")

	registry.RegisterFunc("memory", func(ctx context.Context) CheckResult {
		return CheckResult{Status: StatusHealthy, Message: "Memory OK"}
	})

	report := registry.Check(context.Background())

	if len(report.Checks) != 1 {
		t.Errorf("Checks count = %v, want 1", len(report.Checks))
	}
	if report.Checks[0].Name != "memory" {
		t.Errorf("Check name = %v, want memory", report.Checks[0].Name)
	}
}

func TestRegistry_Unregister(t *testing.T) {
	registry := NewRegistry("test-service", "1.0.0")

	registry.RegisterFunc("temp", func(ctx context.Context) CheckResult {
		return CheckResult{Status: StatusHealthy}
	})

	report1 := registry.Check(context.Background())
	if len(report1.Checks) != 1 {
		t.Errorf("Before unregister: Checks count = %v, want 1", len(report1.Checks))
	}

	registry.Unregister("temp")

	report2 := registry.Check(context.Background())
	if len(report2.Checks) != 0 {
		t.Errorf("After unregister: Checks count = %v, want 0", len(report2.Checks))
	}
}

func TestRegistry_OverallStatus_Unhealthy(t *testing.T) {
	registry := NewRegistry("test-service", "1.0.0")

	registry.RegisterFunc("healthy-check", func(ctx context.Context) CheckResult {
		return CheckResult{Status: StatusHealthy}
	})
	registry.RegisterFunc("unhealthy-check", func(ctx context.Context) CheckResult {
		return CheckResult{Status: StatusUnhealthy}
	})

	report := registry.Check(context.Background())

	if report.Status != StatusUnhealthy {
		t.Errorf("Status = %v, want unhealthy", report.Status)
	}
}

func TestRegistry_OverallStatus_Degraded(t *testing.T) {
	registry := NewRegistry("test-service", "1.0.0")

	registry.RegisterFunc("healthy-check", func(ctx context.Context) CheckResult {
		return CheckResult{Status: StatusHealthy}
	})
	registry.RegisterFunc("degraded-check", func(ctx context.Context) CheckResult {
		return CheckResult{Status: StatusDegraded}
	})

	report := registry.Check(context.Background())

	if report.Status != StatusDegraded {
		t.Errorf("Status = %v, want degraded", report.Status)
	}
}

func TestRegistry_CheckWithTimeout(t *testing.T) {
	registry := NewRegistry("test-service", "1.0.0")

	registry.RegisterFunc("fast-check", func(ctx context.Context) CheckResult {
		return CheckResult{Status: StatusHealthy}
	})

	report := registry.CheckWithTimeout(5 * time.Second)

	if report.Status != StatusHealthy {
		t.Errorf("Status = %v, want healthy", report.Status)
	}
}

func TestRegistry_ConcurrentChecks(t *testing.T) {
	registry := NewRegistry("test-service", "1.0.0")

	var counter int32

	for i := 0; i < 5; i++ {
		registry.RegisterFunc("check"+string(rune('A'+i)), func(ctx context.Context) CheckResult {
			atomic.AddInt32(&counter, 1)
			time.Sleep(10 * time.Millisecond) // Simulate work
			return CheckResult{Status: StatusHealthy}
		})
	}

	start := time.Now()
	report := registry.Check(context.Background())
	duration := time.Since(start)

	if atomic.LoadInt32(&counter) != 5 {
		t.Errorf("Counter = %v, want 5", counter)
	}

	// Checks should run concurrently, so total time should be close to 10ms, not 50ms
	if duration > 100*time.Millisecond {
		t.Errorf("Duration = %v, expected concurrent execution", duration)
	}

	if len(report.Checks) != 5 {
		t.Errorf("Checks count = %v, want 5", len(report.Checks))
	}
}

func TestRegistry_Uptime(t *testing.T) {
	registry := NewRegistry("test-service", "1.0.0")

	time.Sleep(10 * time.Millisecond)

	report := registry.Check(context.Background())

	if report.Uptime < 10*time.Millisecond {
		t.Errorf("Uptime = %v, expected >= 10ms", report.Uptime)
	}
}

func TestReport_String(t *testing.T) {
	report := &Report{
		Service: "test-service",
		Status:  StatusHealthy,
		Uptime:  1 * time.Hour,
		Checks:  []CheckResult{{}, {}},
	}

	str := report.String()

	if str == "" {
		t.Error("String() returned empty")
	}
	if len(str) < 10 {
		t.Errorf("String() too short: %v", str)
	}
}

func TestAlwaysHealthy(t *testing.T) {
	checker := AlwaysHealthy("always-healthy")

	if checker.Name() != "always-healthy" {
		t.Errorf("Name() = %v, want always-healthy", checker.Name())
	}

	result := checker.Check(context.Background())
	if result.Status != StatusHealthy {
		t.Errorf("Status = %v, want healthy", result.Status)
	}
}

func TestTCPCheck(t *testing.T) {
	checker := TCPCheck("tcp-test", "localhost:8080", time.Second)

	if checker.Name() != "tcp-test" {
		t.Errorf("Name() = %v, want tcp-test", checker.Name())
	}

	result := checker.Check(context.Background())
	if result.Details["address"] != "localhost:8080" {
		t.Errorf("Details[address] = %v, want localhost:8080", result.Details["address"])
	}
}

func TestHTTPCheck(t *testing.T) {
	checker := HTTPCheck("http-test", "http://example.com/health", time.Second)

	if checker.Name() != "http-test" {
		t.Errorf("Name() = %v, want http-test", checker.Name())
	}

	result := checker.Check(context.Background())
	if result.Details["url"] != "http://example.com/health" {
		t.Errorf("Details[url] = %v, want http://example.com/health", result.Details["url"])
	}
}

func TestGRPCCheck(t *testing.T) {
	checker := GRPCCheck("grpc-test", "localhost:9090", time.Second)

	if checker.Name() != "grpc-test" {
		t.Errorf("Name() = %v, want grpc-test", checker.Name())
	}

	result := checker.Check(context.Background())
	if result.Details["address"] != "localhost:9090" {
		t.Errorf("Details[address] = %v, want localhost:9090", result.Details["address"])
	}
}
