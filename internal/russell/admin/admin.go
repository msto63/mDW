// ============================================================================
// meinDENKWERK (mDW) - Lokale KI-Plattform
// ============================================================================
//
// Package:     admin
// Description: Administration and monitoring for Russell Service Orchestrator
// Author:      Mike Stoffels with Claude
// Created:     2025-12-06
// License:     MIT
// ============================================================================

package admin

import (
	"context"
	"sort"
	"sync"
	"time"

	"github.com/msto63/mDW/pkg/core/discovery"
	"github.com/msto63/mDW/pkg/core/logging"
)

// ServiceStatus represents the status of a service
type ServiceStatus struct {
	Name           string                 `json:"name"`
	Type           string                 `json:"type"`
	Status         HealthStatus           `json:"status"`
	Address        string                 `json:"address"`
	Version        string                 `json:"version,omitempty"`
	LastSeen       time.Time              `json:"last_seen"`
	Uptime         time.Duration          `json:"uptime,omitempty"`
	Metrics        map[string]interface{} `json:"metrics,omitempty"`
	HealthDetails  map[string]string      `json:"health_details,omitempty"`
	LastError      string                 `json:"last_error,omitempty"`
	LastErrorTime  time.Time              `json:"last_error_time,omitempty"`
}

// HealthStatus represents the health status of a service
type HealthStatus string

const (
	HealthStatusHealthy   HealthStatus = "healthy"
	HealthStatusDegraded  HealthStatus = "degraded"
	HealthStatusUnhealthy HealthStatus = "unhealthy"
	HealthStatusUnknown   HealthStatus = "unknown"
)

// SystemOverview represents an overview of the entire system
type SystemOverview struct {
	Timestamp          time.Time                  `json:"timestamp"`
	TotalServices      int                        `json:"total_services"`
	HealthyServices    int                        `json:"healthy_services"`
	DegradedServices   int                        `json:"degraded_services"`
	UnhealthyServices  int                        `json:"unhealthy_services"`
	Services           map[string]*ServiceStatus  `json:"services"`
	SystemMetrics      *SystemMetrics             `json:"system_metrics"`
	RecentErrors       []ErrorEntry               `json:"recent_errors,omitempty"`
}

// SystemMetrics represents system-wide metrics
type SystemMetrics struct {
	TotalRequests       int64         `json:"total_requests"`
	SuccessfulRequests  int64         `json:"successful_requests"`
	FailedRequests      int64         `json:"failed_requests"`
	AverageResponseTime time.Duration `json:"average_response_time"`
	RequestsPerSecond   float64       `json:"requests_per_second"`
	PipelineExecutions  int64         `json:"pipeline_executions"`
	ActiveConnections   int           `json:"active_connections"`
}

// ErrorEntry represents an error log entry
type ErrorEntry struct {
	Timestamp   time.Time `json:"timestamp"`
	Service     string    `json:"service"`
	Operation   string    `json:"operation"`
	ErrorCode   string    `json:"error_code,omitempty"`
	Message     string    `json:"message"`
	RequestID   string    `json:"request_id,omitempty"`
}

// ServiceConfig represents configuration for a service
type ServiceConfig struct {
	ServiceName    string                 `json:"service_name"`
	Enabled        bool                   `json:"enabled"`
	Priority       int                    `json:"priority"`
	MaxConcurrency int                    `json:"max_concurrency"`
	Timeout        time.Duration          `json:"timeout"`
	RetryPolicy    *RetryPolicy           `json:"retry_policy,omitempty"`
	Settings       map[string]interface{} `json:"settings,omitempty"`
}

// RetryPolicy defines retry behavior
type RetryPolicy struct {
	MaxRetries     int           `json:"max_retries"`
	InitialBackoff time.Duration `json:"initial_backoff"`
	MaxBackoff     time.Duration `json:"max_backoff"`
	BackoffFactor  float64       `json:"backoff_factor"`
}

// Admin provides administration and monitoring capabilities
type Admin struct {
	logger        *logging.Logger
	discovery     discovery.Client
	services      map[string]*ServiceStatus
	configs       map[string]*ServiceConfig
	errors        []ErrorEntry
	metrics       *SystemMetrics
	metricsStart  time.Time
	requestCount  int64
	successCount  int64
	failureCount  int64
	totalLatency  time.Duration
	mu            sync.RWMutex
	maxErrors     int
}

// Config holds configuration for Admin
type Config struct {
	DiscoveryClient discovery.Client
	MaxErrorHistory int
}

// NewAdmin creates a new Admin instance
func NewAdmin(cfg Config) *Admin {
	maxErrors := cfg.MaxErrorHistory
	if maxErrors <= 0 {
		maxErrors = 100
	}

	return &Admin{
		logger:       logging.New("russell-admin"),
		discovery:    cfg.DiscoveryClient,
		services:     make(map[string]*ServiceStatus),
		configs:      make(map[string]*ServiceConfig),
		errors:       make([]ErrorEntry, 0, maxErrors),
		metrics:      &SystemMetrics{},
		metricsStart: time.Now(),
		maxErrors:    maxErrors,
	}
}

// GetSystemOverview returns a comprehensive system overview
func (a *Admin) GetSystemOverview(ctx context.Context) (*SystemOverview, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	// Update service statuses
	a.refreshServiceStatuses(ctx)

	// Count health statuses
	healthy, degraded, unhealthy := 0, 0, 0
	for _, svc := range a.services {
		switch svc.Status {
		case HealthStatusHealthy:
			healthy++
		case HealthStatusDegraded:
			degraded++
		case HealthStatusUnhealthy:
			unhealthy++
		}
	}

	// Calculate metrics
	a.updateMetrics()

	// Get recent errors
	recentErrors := make([]ErrorEntry, 0)
	if len(a.errors) > 10 {
		recentErrors = a.errors[len(a.errors)-10:]
	} else {
		recentErrors = a.errors
	}

	return &SystemOverview{
		Timestamp:         time.Now(),
		TotalServices:     len(a.services),
		HealthyServices:   healthy,
		DegradedServices:  degraded,
		UnhealthyServices: unhealthy,
		Services:          a.services,
		SystemMetrics:     a.metrics,
		RecentErrors:      recentErrors,
	}, nil
}

// GetServiceStatus returns the status of a specific service
func (a *Admin) GetServiceStatus(ctx context.Context, serviceName string) (*ServiceStatus, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	// Refresh this specific service
	a.refreshServiceStatus(ctx, serviceName)

	status, ok := a.services[serviceName]
	if !ok {
		return &ServiceStatus{
			Name:   serviceName,
			Status: HealthStatusUnknown,
		}, nil
	}

	return status, nil
}

// ListServices returns all registered services
func (a *Admin) ListServices(ctx context.Context) ([]*ServiceStatus, error) {
	a.mu.Lock()
	a.refreshServiceStatuses(ctx)
	a.mu.Unlock()

	a.mu.RLock()
	defer a.mu.RUnlock()

	services := make([]*ServiceStatus, 0, len(a.services))
	for _, svc := range a.services {
		services = append(services, svc)
	}

	// Sort by name
	sort.Slice(services, func(i, j int) bool {
		return services[i].Name < services[j].Name
	})

	return services, nil
}

// GetServiceConfig returns configuration for a service
func (a *Admin) GetServiceConfig(serviceName string) (*ServiceConfig, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	config, ok := a.configs[serviceName]
	if !ok {
		// Return default config
		return &ServiceConfig{
			ServiceName:    serviceName,
			Enabled:        true,
			Priority:       1,
			MaxConcurrency: 10,
			Timeout:        30 * time.Second,
			RetryPolicy: &RetryPolicy{
				MaxRetries:     3,
				InitialBackoff: 100 * time.Millisecond,
				MaxBackoff:     5 * time.Second,
				BackoffFactor:  2.0,
			},
		}, nil
	}

	return config, nil
}

// UpdateServiceConfig updates configuration for a service
func (a *Admin) UpdateServiceConfig(config *ServiceConfig) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.configs[config.ServiceName] = config
	a.logger.Info("Service configuration updated",
		"service", config.ServiceName,
		"enabled", config.Enabled,
	)

	return nil
}

// RecordRequest records a request for metrics
func (a *Admin) RecordRequest(service, operation string, success bool, latency time.Duration, requestID string) {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.requestCount++
	if success {
		a.successCount++
	} else {
		a.failureCount++
	}
	a.totalLatency += latency
}

// RecordError records an error
func (a *Admin) RecordError(service, operation, errorCode, message, requestID string) {
	a.mu.Lock()
	defer a.mu.Unlock()

	entry := ErrorEntry{
		Timestamp:   time.Now(),
		Service:     service,
		Operation:   operation,
		ErrorCode:   errorCode,
		Message:     message,
		RequestID:   requestID,
	}

	a.errors = append(a.errors, entry)

	// Trim to max size
	if len(a.errors) > a.maxErrors {
		a.errors = a.errors[len(a.errors)-a.maxErrors:]
	}

	// Update service status with last error
	if svc, ok := a.services[service]; ok {
		svc.LastError = message
		svc.LastErrorTime = entry.Timestamp
	}

	a.logger.Warn("Error recorded",
		"service", service,
		"operation", operation,
		"error", message,
	)
}

// GetErrors returns error history
func (a *Admin) GetErrors(limit int) []ErrorEntry {
	a.mu.RLock()
	defer a.mu.RUnlock()

	if limit <= 0 || limit > len(a.errors) {
		limit = len(a.errors)
	}

	// Return most recent errors
	result := make([]ErrorEntry, limit)
	copy(result, a.errors[len(a.errors)-limit:])

	// Reverse to get newest first
	for i, j := 0, len(result)-1; i < j; i, j = i+1, j-1 {
		result[i], result[j] = result[j], result[i]
	}

	return result
}

// GetMetrics returns current system metrics
func (a *Admin) GetMetrics() *SystemMetrics {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.updateMetrics()
	return a.metrics
}

// ResetMetrics resets all metrics counters
func (a *Admin) ResetMetrics() {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.requestCount = 0
	a.successCount = 0
	a.failureCount = 0
	a.totalLatency = 0
	a.metricsStart = time.Now()
	a.metrics = &SystemMetrics{}

	a.logger.Info("Metrics reset")
}

// refreshServiceStatuses updates status for all known services
func (a *Admin) refreshServiceStatuses(ctx context.Context) {
	knownServices := []string{"kant", "russell", "turing", "hypatia", "leibniz", "babbage", "bayes"}

	for _, svc := range knownServices {
		a.refreshServiceStatus(ctx, svc)
	}
}

// refreshServiceStatus updates status for a single service
func (a *Admin) refreshServiceStatus(ctx context.Context, serviceName string) {
	status := &ServiceStatus{
		Name:     serviceName,
		Type:     a.getServiceType(serviceName),
		Status:   HealthStatusUnknown,
		LastSeen: time.Now(),
	}

	// Try to discover the service
	if a.discovery != nil {
		services, err := a.discovery.Discover(ctx, serviceName)
		if err == nil && len(services) > 0 {
			svc := services[0]
			status.Address = svc.FullAddress()
			status.Version = svc.Metadata["version"]
			status.Status = HealthStatusHealthy

			// Preserve existing error info
			if existing, ok := a.services[serviceName]; ok {
				status.LastError = existing.LastError
				status.LastErrorTime = existing.LastErrorTime
				if existing.Status == HealthStatusUnhealthy {
					// Service might be recovering
					status.Status = HealthStatusDegraded
				}
			}
		} else {
			status.Status = HealthStatusUnhealthy
		}
	}

	a.services[serviceName] = status
}

// updateMetrics calculates current metrics
func (a *Admin) updateMetrics() {
	elapsed := time.Since(a.metricsStart).Seconds()

	avgLatency := time.Duration(0)
	if a.requestCount > 0 {
		avgLatency = time.Duration(int64(a.totalLatency) / a.requestCount)
	}

	rps := float64(0)
	if elapsed > 0 {
		rps = float64(a.requestCount) / elapsed
	}

	a.metrics = &SystemMetrics{
		TotalRequests:       a.requestCount,
		SuccessfulRequests:  a.successCount,
		FailedRequests:      a.failureCount,
		AverageResponseTime: avgLatency,
		RequestsPerSecond:   rps,
	}
}

// getServiceType returns the type/category of a service
func (a *Admin) getServiceType(serviceName string) string {
	types := map[string]string{
		"kant":    "gateway",
		"russell": "orchestrator",
		"turing":  "llm",
		"hypatia": "rag",
		"leibniz": "agent",
		"babbage": "nlp",
		"bayes":   "logging",
	}

	if t, ok := types[serviceName]; ok {
		return t
	}
	return "unknown"
}

// HealthSummary returns a brief health summary
func (a *Admin) HealthSummary(ctx context.Context) map[string]HealthStatus {
	a.mu.Lock()
	a.refreshServiceStatuses(ctx)
	a.mu.Unlock()

	a.mu.RLock()
	defer a.mu.RUnlock()

	result := make(map[string]HealthStatus)
	for name, status := range a.services {
		result[name] = status.Status
	}

	return result
}
