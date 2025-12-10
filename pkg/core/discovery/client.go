package discovery

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/msto63/mDW/pkg/core/logging"
)

var discoveryLogger = logging.New("discovery")

// ServiceStatus represents the status of a service
type ServiceStatus string

const (
	ServiceStatusHealthy   ServiceStatus = "healthy"
	ServiceStatusUnhealthy ServiceStatus = "unhealthy"
	ServiceStatusStarting  ServiceStatus = "starting"
	ServiceStatusStopping  ServiceStatus = "stopping"
	ServiceStatusUnknown   ServiceStatus = "unknown"
)

// ServiceInfo represents information about a registered service
type ServiceInfo struct {
	ID            string
	Name          string
	Version       string
	Address       string
	Port          int
	Status        ServiceStatus
	Metadata      map[string]string
	Tags          []string
	LastHeartbeat time.Time
	RegisteredAt  time.Time
}

// FullAddress returns the full address of the service
func (s *ServiceInfo) FullAddress() string {
	return fmt.Sprintf("%s:%d", s.Address, s.Port)
}

// Client is the service discovery client interface
type Client interface {
	// Register registers the current service with the registry
	Register(ctx context.Context, info *ServiceInfo) error

	// Deregister removes the current service from the registry
	Deregister(ctx context.Context, id string) error

	// Heartbeat sends a heartbeat to keep the registration alive
	Heartbeat(ctx context.Context, id string) error

	// Discover finds services by name
	Discover(ctx context.Context, name string) ([]*ServiceInfo, error)

	// Get returns a specific service by ID
	Get(ctx context.Context, id string) (*ServiceInfo, error)

	// List returns all registered services
	List(ctx context.Context) ([]*ServiceInfo, error)

	// Close closes the client connection
	Close() error
}

// LocalRegistry is an in-memory service registry for local development
type LocalRegistry struct {
	mu       sync.RWMutex
	services map[string]*ServiceInfo
}

// NewLocalRegistry creates a new local registry
func NewLocalRegistry() *LocalRegistry {
	return &LocalRegistry{
		services: make(map[string]*ServiceInfo),
	}
}

// Register registers a service
func (r *LocalRegistry) Register(ctx context.Context, info *ServiceInfo) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if info.ID == "" {
		info.ID = uuid.New().String()
	}
	info.RegisteredAt = time.Now()
	info.LastHeartbeat = time.Now()
	if info.Status == "" {
		info.Status = ServiceStatusHealthy
	}

	r.services[info.ID] = info
	return nil
}

// Deregister removes a service
func (r *LocalRegistry) Deregister(ctx context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	delete(r.services, id)
	return nil
}

// Heartbeat updates the heartbeat timestamp
func (r *LocalRegistry) Heartbeat(ctx context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if svc, ok := r.services[id]; ok {
		svc.LastHeartbeat = time.Now()
		return nil
	}
	return fmt.Errorf("service not found: %s", id)
}

// Discover finds services by name
func (r *LocalRegistry) Discover(ctx context.Context, name string) ([]*ServiceInfo, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var results []*ServiceInfo
	for _, svc := range r.services {
		if svc.Name == name && svc.Status == ServiceStatusHealthy {
			results = append(results, svc)
		}
	}
	return results, nil
}

// Get returns a specific service
func (r *LocalRegistry) Get(ctx context.Context, id string) (*ServiceInfo, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if svc, ok := r.services[id]; ok {
		return svc, nil
	}
	return nil, fmt.Errorf("service not found: %s", id)
}

// List returns all services
func (r *LocalRegistry) List(ctx context.Context) ([]*ServiceInfo, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	results := make([]*ServiceInfo, 0, len(r.services))
	for _, svc := range r.services {
		results = append(results, svc)
	}
	return results, nil
}

// Close closes the registry (no-op for local)
func (r *LocalRegistry) Close() error {
	return nil
}

// Registration handles automatic service registration and heartbeat
type Registration struct {
	client   Client
	info     *ServiceInfo
	interval time.Duration
	stopCh   chan struct{}
	doneCh   chan struct{}
}

// NewRegistration creates a new registration
func NewRegistration(client Client, info *ServiceInfo, heartbeatInterval time.Duration) *Registration {
	return &Registration{
		client:   client,
		info:     info,
		interval: heartbeatInterval,
		stopCh:   make(chan struct{}),
		doneCh:   make(chan struct{}),
	}
}

// Start starts the registration and heartbeat loop
func (r *Registration) Start(ctx context.Context) error {
	// Register the service
	if err := r.client.Register(ctx, r.info); err != nil {
		return fmt.Errorf("failed to register service: %w", err)
	}

	// Start heartbeat loop
	go r.heartbeatLoop()

	return nil
}

// Stop stops the registration and deregisters the service
func (r *Registration) Stop(ctx context.Context) error {
	close(r.stopCh)
	<-r.doneCh

	return r.client.Deregister(ctx, r.info.ID)
}

// heartbeatLoop sends periodic heartbeats
func (r *Registration) heartbeatLoop() {
	defer close(r.doneCh)

	ticker := time.NewTicker(r.interval)
	defer ticker.Stop()

	for {
		select {
		case <-r.stopCh:
			return
		case <-ticker.C:
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			if err := r.client.Heartbeat(ctx, r.info.ID); err != nil {
				discoveryLogger.Warn("Heartbeat failed", "service_id", r.info.ID, "error", err)
			}
			cancel()
		}
	}
}

// ServiceID returns the registered service ID
func (r *Registration) ServiceID() string {
	return r.info.ID
}

// ServiceLocator provides service lookup functionality
type ServiceLocator struct {
	client Client
	cache  map[string][]*ServiceInfo
	mu     sync.RWMutex
	ttl    time.Duration
	lastUpdate map[string]time.Time
}

// NewServiceLocator creates a new service locator
func NewServiceLocator(client Client, cacheTTL time.Duration) *ServiceLocator {
	return &ServiceLocator{
		client:     client,
		cache:      make(map[string][]*ServiceInfo),
		ttl:        cacheTTL,
		lastUpdate: make(map[string]time.Time),
	}
}

// Locate finds a service by name
func (l *ServiceLocator) Locate(ctx context.Context, name string) (*ServiceInfo, error) {
	services, err := l.LocateAll(ctx, name)
	if err != nil {
		return nil, err
	}
	if len(services) == 0 {
		return nil, fmt.Errorf("no healthy instances found for service: %s", name)
	}
	// Return first healthy instance (simple round-robin could be added)
	return services[0], nil
}

// LocateAll finds all instances of a service
func (l *ServiceLocator) LocateAll(ctx context.Context, name string) ([]*ServiceInfo, error) {
	l.mu.RLock()
	cached, ok := l.cache[name]
	lastUpdate := l.lastUpdate[name]
	l.mu.RUnlock()

	// Return cached if still valid
	if ok && time.Since(lastUpdate) < l.ttl {
		return cached, nil
	}

	// Fetch from registry
	services, err := l.client.Discover(ctx, name)
	if err != nil {
		return nil, err
	}

	// Update cache
	l.mu.Lock()
	l.cache[name] = services
	l.lastUpdate[name] = time.Now()
	l.mu.Unlock()

	return services, nil
}

// InvalidateCache clears the cache for a service
func (l *ServiceLocator) InvalidateCache(name string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	delete(l.cache, name)
	delete(l.lastUpdate, name)
}

// ClearCache clears all cached entries
func (l *ServiceLocator) ClearCache() {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.cache = make(map[string][]*ServiceInfo)
	l.lastUpdate = make(map[string]time.Time)
}
