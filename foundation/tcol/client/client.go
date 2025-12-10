// File: client.go
// Title: TCOL Service Client Implementation
// Description: Implements the service client for communicating with mDW
//              microservices via gRPC. Provides connection management,
//              service discovery, health checking, and circuit breaker
//              patterns for reliable service communication.
// Author: msto63 with Claude Sonnet 4.0
// Version: v0.1.0
// Created: 2025-01-25
// Modified: 2025-01-25
//
// Change History:
// - 2025-01-25 v0.1.0: Initial client implementation

package client

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	mdwlog "github.com/msto63/mDW/foundation/core/log"
	mdwexecutor "github.com/msto63/mDW/foundation/tcol/executor"
)

// Client implements the ServiceClient interface for TCOL
type Client struct {
	connections map[string]*ServiceConnection
	discovery   ServiceDiscovery
	logger      *mdwlog.Logger
	options     Options
	mutex       sync.RWMutex
}

// Options configures client behavior
type Options struct {
	Logger              *mdwlog.Logger
	ServiceDiscovery    ServiceDiscovery
	ConnectionTimeout   time.Duration
	RequestTimeout      time.Duration
	MaxRetries          int
	HealthCheckInterval time.Duration
	CircuitBreakerConfig CircuitBreakerConfig
}

// ServiceConnection represents a connection to a microservice
type ServiceConnection struct {
	ServiceName   string
	Address       string
	Connected     bool
	HealthStatus  HealthStatus
	LastUsed      time.Time
	RequestCount  int64
	ErrorCount    int64
	CircuitBreaker *CircuitBreaker
	mutex         sync.RWMutex
}

// ServiceDiscovery interface for discovering service endpoints
type ServiceDiscovery interface {
	GetServiceAddress(serviceName string) (string, error)
	ListServices() ([]string, error)
	RegisterService(name, address string) error
	UnregisterService(name string) error
}

// HealthStatus represents the health status of a service
type HealthStatus int

const (
	HealthUnknown HealthStatus = iota
	HealthHealthy
	HealthUnhealthy
	HealthDegraded
)

func (hs HealthStatus) String() string {
	switch hs {
	case HealthHealthy:
		return "HEALTHY"
	case HealthUnhealthy:
		return "UNHEALTHY"
	case HealthDegraded:
		return "DEGRADED"
	default:
		return "UNKNOWN"
	}
}

// CircuitBreakerConfig configures circuit breaker behavior
type CircuitBreakerConfig struct {
	FailureThreshold   int
	RecoveryTimeout    time.Duration
	HalfOpenRequests   int
	MinRequestsToTrip  int
}

// CircuitBreakerState represents circuit breaker states
type CircuitBreakerState int

const (
	StateClosed CircuitBreakerState = iota
	StateOpen
	StateHalfOpen
)

// CircuitBreaker implements circuit breaker pattern for service calls
type CircuitBreaker struct {
	config       CircuitBreakerConfig
	state        CircuitBreakerState
	failures     int
	requests     int
	lastFailTime time.Time
	mutex        sync.RWMutex
}

// MockServiceDiscovery provides a simple in-memory service discovery
type MockServiceDiscovery struct {
	services map[string]string
	mutex    sync.RWMutex
}

// New creates a new TCOL service client
func New(opts Options) (*Client, error) {
	// Set defaults
	if opts.Logger == nil {
		opts.Logger = mdwlog.GetDefault()
	}
	if opts.ConnectionTimeout == 0 {
		opts.ConnectionTimeout = 10 * time.Second
	}
	if opts.RequestTimeout == 0 {
		opts.RequestTimeout = 30 * time.Second
	}
	if opts.MaxRetries == 0 {
		opts.MaxRetries = 3
	}
	if opts.HealthCheckInterval == 0 {
		opts.HealthCheckInterval = 30 * time.Second
	}
	if opts.ServiceDiscovery == nil {
		opts.ServiceDiscovery = NewMockServiceDiscovery()
	}

	// Set circuit breaker defaults
	if opts.CircuitBreakerConfig.FailureThreshold == 0 {
		opts.CircuitBreakerConfig.FailureThreshold = 5
	}
	if opts.CircuitBreakerConfig.RecoveryTimeout == 0 {
		opts.CircuitBreakerConfig.RecoveryTimeout = 60 * time.Second
	}
	if opts.CircuitBreakerConfig.HalfOpenRequests == 0 {
		opts.CircuitBreakerConfig.HalfOpenRequests = 3
	}
	if opts.CircuitBreakerConfig.MinRequestsToTrip == 0 {
		opts.CircuitBreakerConfig.MinRequestsToTrip = 10
	}

	client := &Client{
		connections: make(map[string]*ServiceConnection),
		discovery:   opts.ServiceDiscovery,
		logger:      opts.Logger.WithField("component", "tcol-client"),
		options:     opts,
	}

	// Start health check routine
	go client.healthCheckLoop()

	client.logger.Info("TCOL service client initialized", mdwlog.Fields{
		"connectionTimeout":   opts.ConnectionTimeout,
		"requestTimeout":      opts.RequestTimeout,
		"maxRetries":          opts.MaxRetries,
		"healthCheckInterval": opts.HealthCheckInterval,
	})

	return client, nil
}

// Execute executes a command on a microservice
func (c *Client) Execute(ctx context.Context, serviceName, objectName, methodName string,
	params map[string]interface{}, execCtx *mdwexecutor.ExecutionContext) (*mdwexecutor.ServiceResponse, error) {

	// Get or create connection
	conn, err := c.getConnection(serviceName)
	if err != nil {
		return nil, err
	}

	// Check circuit breaker
	if !conn.CircuitBreaker.AllowRequest() {
		return nil, fmt.Errorf("circuit breaker is open for service %s (state: %v)", serviceName, conn.CircuitBreaker.state)
	}

	// Create request context with timeout
	reqCtx, cancel := context.WithTimeout(ctx, c.options.RequestTimeout)
	defer cancel()

	// Execute with retries
	var lastErr error
	for attempt := 0; attempt <= c.options.MaxRetries; attempt++ {
		if attempt > 0 {
			c.logger.Debug("Retrying service request", mdwlog.Fields{
				"serviceName": serviceName,
				"attempt":     attempt,
				"maxRetries":  c.options.MaxRetries,
			})
			
			// Wait before retry
			select {
			case <-time.After(time.Duration(attempt) * time.Second):
			case <-reqCtx.Done():
				return nil, reqCtx.Err()
			}
		}

		response, err := c.executeRequest(reqCtx, conn, objectName, methodName, params, execCtx)
		if err == nil {
			conn.CircuitBreaker.RecordSuccess()
			conn.updateStats(true)
			return response, nil
		}

		lastErr = err
		conn.CircuitBreaker.RecordFailure()
		conn.updateStats(false)

		// Don't retry on certain errors
		if c.shouldNotRetry(err) {
			break
		}
	}

	return nil, fmt.Errorf("service request failed after %d retries for service %s: %w", c.options.MaxRetries, serviceName, lastErr)
}

// Health checks the health of a service
func (c *Client) Health(ctx context.Context, serviceName string) error {
	conn, err := c.getConnection(serviceName)
	if err != nil {
		return err
	}

	// Create request context with timeout
	reqCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	_ = reqCtx // Used in real implementation

	// Mock health check - in real implementation, this would be a gRPC health check
	if !conn.Connected {
		return fmt.Errorf("service %s is not connected (address: %s)", serviceName, conn.Address)
	}

	conn.mutex.Lock()
	conn.HealthStatus = HealthHealthy
	conn.mutex.Unlock()

	return nil
}

// Close closes all connections and resources
func (c *Client) Close() error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	var errors []error
	for serviceName, conn := range c.connections {
		if err := c.closeConnection(conn); err != nil {
			errors = append(errors, fmt.Errorf("failed to close connection to %s: %w", serviceName, err))
		}
	}

	c.connections = make(map[string]*ServiceConnection)

	if len(errors) > 0 {
		return fmt.Errorf("errors closing connections: %v", errors)
	}

	c.logger.Info("TCOL service client closed")
	return nil
}

// getConnection gets or creates a connection to a service
func (c *Client) getConnection(serviceName string) (*ServiceConnection, error) {
	c.mutex.RLock()
	conn, exists := c.connections[serviceName]
	c.mutex.RUnlock()

	if exists && conn.Connected {
		conn.mutex.Lock()
		conn.LastUsed = time.Now()
		conn.mutex.Unlock()
		return conn, nil
	}

	// Create new connection
	return c.createConnection(serviceName)
}

// createConnection creates a new connection to a service
func (c *Client) createConnection(serviceName string) (*ServiceConnection, error) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	// Double-check pattern
	if conn, exists := c.connections[serviceName]; exists && conn.Connected {
		return conn, nil
	}

	// Get service address from discovery
	address, err := c.discovery.GetServiceAddress(serviceName)
	if err != nil {
		return nil, fmt.Errorf("failed to discover service address for %s: %w", serviceName, err)
	}

	// Create connection
	conn := &ServiceConnection{
		ServiceName:    serviceName,
		Address:        address,
		Connected:      false,
		HealthStatus:   HealthUnknown,
		LastUsed:       time.Now(),
		CircuitBreaker: NewCircuitBreaker(c.options.CircuitBreakerConfig),
	}

	// Mock connection - in real implementation, this would establish gRPC connection
	if err := c.connectToService(conn); err != nil {
		return nil, fmt.Errorf("failed to connect to service %s at %s: %w", serviceName, address, err)
	}

	c.connections[serviceName] = conn

	c.logger.Info("Connected to service", mdwlog.Fields{
		"serviceName": serviceName,
		"address":     address,
	})

	return conn, nil
}

// executeRequest executes a request to a service
func (c *Client) executeRequest(ctx context.Context, conn *ServiceConnection,
	objectName, methodName string, params map[string]interface{},
	execCtx *mdwexecutor.ExecutionContext) (*mdwexecutor.ServiceResponse, error) {

	c.logger.Debug("Executing service request", mdwlog.Fields{
		"serviceName": conn.ServiceName,
		"objectName":  objectName,
		"methodName":  methodName,
		"requestID":   execCtx.RequestID,
	})

	// Mock service call - in real implementation, this would be a gRPC call
	response := &mdwexecutor.ServiceResponse{
		Success: true,
		Data: map[string]interface{}{
			"object":    objectName,
			"method":    methodName,
			"params":    params,
			"requestID": execCtx.RequestID,
			"result":    "Mock service response",
		},
		Metadata: map[string]interface{}{
			"serviceName": conn.ServiceName,
			"executionTime": "10ms",
		},
	}

	// Simulate occasional failures for circuit breaker testing
	if conn.RequestCount%10 == 7 { // Fail every 10th request starting at 7
		return nil, errors.New("mock service error for testing")
	}

	return response, nil
}

// connectToService establishes connection to a service
func (c *Client) connectToService(conn *ServiceConnection) error {
	// Mock connection - in real implementation, this would establish gRPC connection
	conn.mutex.Lock()
	defer conn.mutex.Unlock()

	conn.Connected = true
	conn.HealthStatus = HealthHealthy

	return nil
}

// closeConnection closes a service connection
func (c *Client) closeConnection(conn *ServiceConnection) error {
	conn.mutex.Lock()
	defer conn.mutex.Unlock()

	conn.Connected = false
	conn.HealthStatus = HealthUnknown

	// In real implementation, this would close gRPC connection
	return nil
}

// updateStats updates connection statistics
func (conn *ServiceConnection) updateStats(success bool) {
	conn.mutex.Lock()
	defer conn.mutex.Unlock()

	conn.RequestCount++
	if !success {
		conn.ErrorCount++
	}
	conn.LastUsed = time.Now()
}

// shouldNotRetry determines if an error should not be retried
func (c *Client) shouldNotRetry(err error) bool {
	// Don't retry on certain error conditions
	if err == context.Canceled || err == context.DeadlineExceeded {
		return true
	}
	return false
}

// healthCheckLoop runs periodic health checks
func (c *Client) healthCheckLoop() {
	ticker := time.NewTicker(c.options.HealthCheckInterval)
	defer ticker.Stop()

	for range ticker.C {
		c.performHealthChecks()
	}
}

// performHealthChecks performs health checks on all connections
func (c *Client) performHealthChecks() {
	c.mutex.RLock()
	connections := make([]*ServiceConnection, 0, len(c.connections))
	for _, conn := range c.connections {
		connections = append(connections, conn)
	}
	c.mutex.RUnlock()

	for _, conn := range connections {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		err := c.Health(ctx, conn.ServiceName)
		cancel()

		conn.mutex.Lock()
		if err != nil {
			conn.HealthStatus = HealthUnhealthy
			c.logger.Warn("Service health check failed", mdwlog.Fields{
				"serviceName": conn.ServiceName,
				"error":       err.Error(),
			})
		} else {
			conn.HealthStatus = HealthHealthy
		}
		conn.mutex.Unlock()
	}
}

// NewCircuitBreaker creates a new circuit breaker
func NewCircuitBreaker(config CircuitBreakerConfig) *CircuitBreaker {
	return &CircuitBreaker{
		config: config,
		state:  StateClosed,
	}
}

// AllowRequest checks if a request should be allowed
func (cb *CircuitBreaker) AllowRequest() bool {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()

	switch cb.state {
	case StateClosed:
		return true
	case StateOpen:
		if time.Since(cb.lastFailTime) > cb.config.RecoveryTimeout {
			cb.state = StateHalfOpen
			cb.requests = 0
			return true
		}
		return false
	case StateHalfOpen:
		return cb.requests < cb.config.HalfOpenRequests
	default:
		return false
	}
}

// RecordSuccess records a successful request
func (cb *CircuitBreaker) RecordSuccess() {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()

	cb.failures = 0
	cb.requests++

	if cb.state == StateHalfOpen && cb.requests >= cb.config.HalfOpenRequests {
		cb.state = StateClosed
	}
}

// RecordFailure records a failed request
func (cb *CircuitBreaker) RecordFailure() {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()

	cb.failures++
	cb.requests++
	cb.lastFailTime = time.Now()

	if cb.state == StateHalfOpen {
		cb.state = StateOpen
	} else if cb.state == StateClosed &&
		cb.failures >= cb.config.FailureThreshold &&
		cb.requests >= cb.config.MinRequestsToTrip {
		cb.state = StateOpen
	}
}

// NewMockServiceDiscovery creates a mock service discovery
func NewMockServiceDiscovery() *MockServiceDiscovery {
	return &MockServiceDiscovery{
		services: make(map[string]string),
	}
}

// GetServiceAddress returns the address for a service
func (msd *MockServiceDiscovery) GetServiceAddress(serviceName string) (string, error) {
	msd.mutex.RLock()
	defer msd.mutex.RUnlock()

	if address, exists := msd.services[serviceName]; exists {
		return address, nil
	}

	// Return mock address for any service
	address := fmt.Sprintf("localhost:50%03d", len(msd.services)+1)
	msd.services[serviceName] = address
	return address, nil
}

// ListServices returns all registered services
func (msd *MockServiceDiscovery) ListServices() ([]string, error) {
	msd.mutex.RLock()
	defer msd.mutex.RUnlock()

	services := make([]string, 0, len(msd.services))
	for name := range msd.services {
		services = append(services, name)
	}
	return services, nil
}

// RegisterService registers a service
func (msd *MockServiceDiscovery) RegisterService(name, address string) error {
	msd.mutex.Lock()
	defer msd.mutex.Unlock()

	msd.services[name] = address
	return nil
}

// UnregisterService unregisters a service
func (msd *MockServiceDiscovery) UnregisterService(name string) error {
	msd.mutex.Lock()
	defer msd.mutex.Unlock()

	delete(msd.services, name)
	return nil
}