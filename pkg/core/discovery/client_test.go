package discovery

import (
	"context"
	"testing"
	"time"
)

func TestServiceStatus_Constants(t *testing.T) {
	if ServiceStatusHealthy != "healthy" {
		t.Errorf("ServiceStatusHealthy = %v, want healthy", ServiceStatusHealthy)
	}
	if ServiceStatusUnhealthy != "unhealthy" {
		t.Errorf("ServiceStatusUnhealthy = %v, want unhealthy", ServiceStatusUnhealthy)
	}
	if ServiceStatusStarting != "starting" {
		t.Errorf("ServiceStatusStarting = %v, want starting", ServiceStatusStarting)
	}
	if ServiceStatusStopping != "stopping" {
		t.Errorf("ServiceStatusStopping = %v, want stopping", ServiceStatusStopping)
	}
}

func TestServiceInfo_FullAddress(t *testing.T) {
	info := &ServiceInfo{
		Address: "localhost",
		Port:    8080,
	}

	if info.FullAddress() != "localhost:8080" {
		t.Errorf("FullAddress() = %v, want localhost:8080", info.FullAddress())
	}
}

func TestLocalRegistry_Register(t *testing.T) {
	registry := NewLocalRegistry()
	ctx := context.Background()

	info := &ServiceInfo{
		Name:    "test-service",
		Address: "localhost",
		Port:    9000,
	}

	err := registry.Register(ctx, info)
	if err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	// ID should be auto-generated
	if info.ID == "" {
		t.Error("ID should be auto-generated")
	}

	// Status should be set to healthy
	if info.Status != ServiceStatusHealthy {
		t.Errorf("Status = %v, want healthy", info.Status)
	}

	// RegisteredAt should be set
	if info.RegisteredAt.IsZero() {
		t.Error("RegisteredAt should be set")
	}
}

func TestLocalRegistry_RegisterWithID(t *testing.T) {
	registry := NewLocalRegistry()
	ctx := context.Background()

	info := &ServiceInfo{
		ID:      "custom-id",
		Name:    "test-service",
		Address: "localhost",
		Port:    9000,
	}

	err := registry.Register(ctx, info)
	if err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	// ID should remain unchanged
	if info.ID != "custom-id" {
		t.Errorf("ID = %v, want custom-id", info.ID)
	}
}

func TestLocalRegistry_Deregister(t *testing.T) {
	registry := NewLocalRegistry()
	ctx := context.Background()

	info := &ServiceInfo{
		ID:      "test-id",
		Name:    "test-service",
		Address: "localhost",
		Port:    9000,
	}

	registry.Register(ctx, info)
	err := registry.Deregister(ctx, "test-id")
	if err != nil {
		t.Fatalf("Deregister() error = %v", err)
	}

	_, err = registry.Get(ctx, "test-id")
	if err == nil {
		t.Error("Get() should return error after Deregister")
	}
}

func TestLocalRegistry_Heartbeat(t *testing.T) {
	registry := NewLocalRegistry()
	ctx := context.Background()

	info := &ServiceInfo{
		ID:      "test-id",
		Name:    "test-service",
		Address: "localhost",
		Port:    9000,
	}

	registry.Register(ctx, info)
	initialHeartbeat := info.LastHeartbeat

	time.Sleep(10 * time.Millisecond)

	err := registry.Heartbeat(ctx, "test-id")
	if err != nil {
		t.Fatalf("Heartbeat() error = %v", err)
	}

	svc, _ := registry.Get(ctx, "test-id")
	if !svc.LastHeartbeat.After(initialHeartbeat) {
		t.Error("LastHeartbeat should be updated after Heartbeat()")
	}
}

func TestLocalRegistry_Heartbeat_NotFound(t *testing.T) {
	registry := NewLocalRegistry()
	ctx := context.Background()

	err := registry.Heartbeat(ctx, "nonexistent")
	if err == nil {
		t.Error("Heartbeat() should return error for nonexistent service")
	}
}

func TestLocalRegistry_Discover(t *testing.T) {
	registry := NewLocalRegistry()
	ctx := context.Background()

	// Register multiple services
	registry.Register(ctx, &ServiceInfo{
		ID:      "svc1",
		Name:    "api",
		Address: "localhost",
		Port:    9001,
		Status:  ServiceStatusHealthy,
	})
	registry.Register(ctx, &ServiceInfo{
		ID:      "svc2",
		Name:    "api",
		Address: "localhost",
		Port:    9002,
		Status:  ServiceStatusHealthy,
	})
	registry.Register(ctx, &ServiceInfo{
		ID:      "svc3",
		Name:    "db",
		Address: "localhost",
		Port:    9003,
		Status:  ServiceStatusHealthy,
	})
	registry.Register(ctx, &ServiceInfo{
		ID:      "svc4",
		Name:    "api",
		Address: "localhost",
		Port:    9004,
		Status:  ServiceStatusUnhealthy, // Unhealthy should be excluded
	})

	services, err := registry.Discover(ctx, "api")
	if err != nil {
		t.Fatalf("Discover() error = %v", err)
	}

	if len(services) != 2 {
		t.Errorf("Discover() returned %v services, want 2", len(services))
	}
}

func TestLocalRegistry_Get(t *testing.T) {
	registry := NewLocalRegistry()
	ctx := context.Background()

	info := &ServiceInfo{
		ID:      "test-id",
		Name:    "test-service",
		Address: "localhost",
		Port:    9000,
	}

	registry.Register(ctx, info)

	svc, err := registry.Get(ctx, "test-id")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	if svc.Name != "test-service" {
		t.Errorf("Name = %v, want test-service", svc.Name)
	}
}

func TestLocalRegistry_Get_NotFound(t *testing.T) {
	registry := NewLocalRegistry()
	ctx := context.Background()

	_, err := registry.Get(ctx, "nonexistent")
	if err == nil {
		t.Error("Get() should return error for nonexistent service")
	}
}

func TestLocalRegistry_List(t *testing.T) {
	registry := NewLocalRegistry()
	ctx := context.Background()

	registry.Register(ctx, &ServiceInfo{ID: "svc1", Name: "service1"})
	registry.Register(ctx, &ServiceInfo{ID: "svc2", Name: "service2"})
	registry.Register(ctx, &ServiceInfo{ID: "svc3", Name: "service3"})

	services, err := registry.List(ctx)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}

	if len(services) != 3 {
		t.Errorf("List() returned %v services, want 3", len(services))
	}
}

func TestLocalRegistry_Close(t *testing.T) {
	registry := NewLocalRegistry()
	err := registry.Close()
	if err != nil {
		t.Errorf("Close() error = %v", err)
	}
}

func TestNewRegistration(t *testing.T) {
	registry := NewLocalRegistry()
	info := &ServiceInfo{
		Name:    "test-service",
		Address: "localhost",
		Port:    9000,
	}

	reg := NewRegistration(registry, info, 10*time.Second)

	if reg.interval != 10*time.Second {
		t.Errorf("interval = %v, want 10s", reg.interval)
	}
}

func TestRegistration_StartAndStop(t *testing.T) {
	registry := NewLocalRegistry()
	ctx := context.Background()

	info := &ServiceInfo{
		Name:    "test-service",
		Address: "localhost",
		Port:    9000,
	}

	reg := NewRegistration(registry, info, 50*time.Millisecond)

	err := reg.Start(ctx)
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	// Verify service is registered
	if info.ID == "" {
		t.Error("Service ID should be set after Start()")
	}

	services, _ := registry.List(ctx)
	if len(services) != 1 {
		t.Errorf("Expected 1 service, got %d", len(services))
	}

	// Wait for at least one heartbeat
	time.Sleep(60 * time.Millisecond)

	err = reg.Stop(ctx)
	if err != nil {
		t.Fatalf("Stop() error = %v", err)
	}

	// Verify service is deregistered
	services, _ = registry.List(ctx)
	if len(services) != 0 {
		t.Errorf("Expected 0 services after Stop(), got %d", len(services))
	}
}

func TestRegistration_ServiceID(t *testing.T) {
	registry := NewLocalRegistry()
	ctx := context.Background()

	info := &ServiceInfo{
		ID:      "my-service-id",
		Name:    "test-service",
		Address: "localhost",
		Port:    9000,
	}

	reg := NewRegistration(registry, info, 10*time.Second)
	reg.Start(ctx)
	defer reg.Stop(ctx)

	if reg.ServiceID() != "my-service-id" {
		t.Errorf("ServiceID() = %v, want my-service-id", reg.ServiceID())
	}
}

func TestServiceLocator_Locate(t *testing.T) {
	registry := NewLocalRegistry()
	ctx := context.Background()

	registry.Register(ctx, &ServiceInfo{
		ID:      "svc1",
		Name:    "api",
		Address: "localhost",
		Port:    9001,
		Status:  ServiceStatusHealthy,
	})

	locator := NewServiceLocator(registry, 10*time.Second)

	svc, err := locator.Locate(ctx, "api")
	if err != nil {
		t.Fatalf("Locate() error = %v", err)
	}

	if svc.ID != "svc1" {
		t.Errorf("ID = %v, want svc1", svc.ID)
	}
}

func TestServiceLocator_Locate_NotFound(t *testing.T) {
	registry := NewLocalRegistry()
	ctx := context.Background()

	locator := NewServiceLocator(registry, 10*time.Second)

	_, err := locator.Locate(ctx, "nonexistent")
	if err == nil {
		t.Error("Locate() should return error for nonexistent service")
	}
}

func TestServiceLocator_LocateAll(t *testing.T) {
	registry := NewLocalRegistry()
	ctx := context.Background()

	registry.Register(ctx, &ServiceInfo{
		ID:     "svc1",
		Name:   "api",
		Status: ServiceStatusHealthy,
	})
	registry.Register(ctx, &ServiceInfo{
		ID:     "svc2",
		Name:   "api",
		Status: ServiceStatusHealthy,
	})

	locator := NewServiceLocator(registry, 10*time.Second)

	services, err := locator.LocateAll(ctx, "api")
	if err != nil {
		t.Fatalf("LocateAll() error = %v", err)
	}

	if len(services) != 2 {
		t.Errorf("LocateAll() returned %d services, want 2", len(services))
	}
}

func TestServiceLocator_Cache(t *testing.T) {
	registry := NewLocalRegistry()
	ctx := context.Background()

	registry.Register(ctx, &ServiceInfo{
		ID:     "svc1",
		Name:   "api",
		Status: ServiceStatusHealthy,
	})

	locator := NewServiceLocator(registry, 1*time.Second)

	// First call should hit registry
	services1, _ := locator.LocateAll(ctx, "api")
	if len(services1) != 1 {
		t.Fatalf("Expected 1 service, got %d", len(services1))
	}

	// Add another service
	registry.Register(ctx, &ServiceInfo{
		ID:     "svc2",
		Name:   "api",
		Status: ServiceStatusHealthy,
	})

	// Second call should hit cache (still 1 service)
	services2, _ := locator.LocateAll(ctx, "api")
	if len(services2) != 1 {
		t.Errorf("Expected 1 service from cache, got %d", len(services2))
	}

	// Wait for cache to expire
	time.Sleep(1100 * time.Millisecond)

	// Third call should hit registry again
	services3, _ := locator.LocateAll(ctx, "api")
	if len(services3) != 2 {
		t.Errorf("Expected 2 services after cache expiry, got %d", len(services3))
	}
}

func TestServiceLocator_InvalidateCache(t *testing.T) {
	registry := NewLocalRegistry()
	ctx := context.Background()

	registry.Register(ctx, &ServiceInfo{
		ID:     "svc1",
		Name:   "api",
		Status: ServiceStatusHealthy,
	})

	locator := NewServiceLocator(registry, 10*time.Second)

	// First call
	locator.LocateAll(ctx, "api")

	// Add another service
	registry.Register(ctx, &ServiceInfo{
		ID:     "svc2",
		Name:   "api",
		Status: ServiceStatusHealthy,
	})

	// Invalidate cache
	locator.InvalidateCache("api")

	// Should now see 2 services
	services, _ := locator.LocateAll(ctx, "api")
	if len(services) != 2 {
		t.Errorf("Expected 2 services after InvalidateCache, got %d", len(services))
	}
}

func TestServiceLocator_ClearCache(t *testing.T) {
	registry := NewLocalRegistry()
	ctx := context.Background()

	registry.Register(ctx, &ServiceInfo{ID: "svc1", Name: "api", Status: ServiceStatusHealthy})
	registry.Register(ctx, &ServiceInfo{ID: "svc2", Name: "db", Status: ServiceStatusHealthy})

	locator := NewServiceLocator(registry, 10*time.Second)

	// Populate cache
	locator.LocateAll(ctx, "api")
	locator.LocateAll(ctx, "db")

	// Clear entire cache
	locator.ClearCache()

	// Add more services
	registry.Register(ctx, &ServiceInfo{ID: "svc3", Name: "api", Status: ServiceStatusHealthy})
	registry.Register(ctx, &ServiceInfo{ID: "svc4", Name: "db", Status: ServiceStatusHealthy})

	// Should see updated counts
	apiServices, _ := locator.LocateAll(ctx, "api")
	dbServices, _ := locator.LocateAll(ctx, "db")

	if len(apiServices) != 2 {
		t.Errorf("Expected 2 api services, got %d", len(apiServices))
	}
	if len(dbServices) != 2 {
		t.Errorf("Expected 2 db services, got %d", len(dbServices))
	}
}
