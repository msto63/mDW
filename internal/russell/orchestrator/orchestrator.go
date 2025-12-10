// ============================================================================
// meinDENKWERK (mDW) - Lokale KI-Plattform
// ============================================================================
//
// Package:     orchestrator
// Description: Service orchestrator for managing mDW services lifecycle
// Author:      Mike Stoffels with Claude
// Created:     2025-12-07
// License:     MIT
// ============================================================================

package orchestrator

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/msto63/mDW/pkg/core/logging"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/health/grpc_health_v1"
)

// ServiceStatus represents the status of a managed service
type ServiceStatus int

const (
	StatusUnknown ServiceStatus = iota
	StatusStopped
	StatusStarting
	StatusRunning
	StatusStopping
	StatusFailed
)

func (s ServiceStatus) String() string {
	switch s {
	case StatusStopped:
		return "stopped"
	case StatusStarting:
		return "starting"
	case StatusRunning:
		return "running"
	case StatusStopping:
		return "stopping"
	case StatusFailed:
		return "failed"
	default:
		return "unknown"
	}
}

// OrchestratorStatus represents the overall orchestrator status
type OrchestratorStatus string

const (
	OrchestratorStarting OrchestratorStatus = "starting"
	OrchestratorRunning  OrchestratorStatus = "running"
	OrchestratorStopping OrchestratorStatus = "stopping"
	OrchestratorStopped  OrchestratorStatus = "stopped"
)

// ManagedService represents a service managed by the orchestrator
type ManagedService struct {
	Config          ServiceConfig
	Status          ServiceStatus
	PID             int
	StartedAt       time.Time
	RestartCount    int
	LastError       string
	LastHealthCheck time.Time
	Healthy         bool
	cmd             *exec.Cmd
	mu              sync.RWMutex
}

// ServiceEvent represents a service lifecycle event
type ServiceEvent struct {
	Type        EventType
	ServiceName string
	Message     string
	Timestamp   time.Time
}

// EventType defines the type of service event
type EventType int

const (
	EventStarted EventType = iota
	EventStopped
	EventFailed
	EventHealthCheckPassed
	EventHealthCheckFailed
	EventRestarting
)

// PortConflict holds information about a port conflict
type PortConflict struct {
	Port    int
	PID     int
	Service string
}

// Orchestrator manages the lifecycle of all mDW services
type Orchestrator struct {
	mu           sync.RWMutex
	services     map[string]*ManagedService
	config       *ServicesConfig
	logger       *logging.Logger
	status       OrchestratorStatus
	startedAt    time.Time

	// Channels
	stopCh      chan struct{}
	eventCh     chan ServiceEvent
	subscribers []chan ServiceEvent
	subMu       sync.RWMutex
}

// New creates a new orchestrator instance
func New(configPath string) (*Orchestrator, error) {
	config, err := LoadServicesConfig(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	o := &Orchestrator{
		services:    make(map[string]*ManagedService),
		config:      config,
		logger:      logging.New("orchestrator"),
		status:      OrchestratorStopped,
		stopCh:      make(chan struct{}),
		eventCh:     make(chan ServiceEvent, 100),
		subscribers: make([]chan ServiceEvent, 0),
	}

	// Initialize managed services from config
	for _, svc := range config.Services {
		if svc.Enabled {
			o.services[svc.ShortName] = &ManagedService{
				Config: svc,
				Status: StatusStopped,
			}
		}
	}

	// Start event dispatcher
	go o.dispatchEvents()

	return o, nil
}

// StartAll starts all services in dependency order with retry logic
func (o *Orchestrator) StartAll(ctx context.Context) error {
	o.mu.Lock()
	// Reset stopCh if it was closed (from a previous StopAll call)
	select {
	case <-o.stopCh:
		// Channel was closed, create a new one
		o.stopCh = make(chan struct{})
	default:
		// Channel is still open, nothing to do
	}
	o.status = OrchestratorStarting
	o.startedAt = time.Now()
	o.mu.Unlock()

	o.logger.Info("Starting all services")

	// Check external dependencies first
	if err := o.checkExternalDependencies(ctx); err != nil {
		o.mu.Lock()
		o.status = OrchestratorStopped
		o.mu.Unlock()
		return fmt.Errorf("external dependency check failed: %w", err)
	}

	// Get services sorted by start order
	sortedServices := o.config.GetServicesSortedByStartOrder()

	for _, svcConfig := range sortedServices {
		// Check if we should stop
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-o.stopCh:
			return fmt.Errorf("orchestrator stopped")
		default:
		}

		// Wait for internal dependencies
		if err := o.waitForDependencies(ctx, svcConfig); err != nil {
			o.logger.Error("Dependency check failed",
				"service", svcConfig.Name,
				"error", err)
			return fmt.Errorf("dependency check failed for %s: %w", svcConfig.Name, err)
		}

		// Check for port conflict
		if conflict := o.checkPortConflict(svcConfig); conflict != nil {
			if err := o.handlePortConflict(ctx, svcConfig, conflict); err != nil {
				return fmt.Errorf("failed to handle port conflict for %s: %w", svcConfig.Name, err)
			}
			// If we adopted an existing service, continue to next
			o.mu.RLock()
			svc := o.services[svcConfig.ShortName]
			o.mu.RUnlock()
			if svc != nil && svc.Status == StatusRunning {
				continue
			}
		}

		// Start service with retry
		if err := o.startServiceWithRetry(ctx, svcConfig); err != nil {
			o.logger.Error("Failed to start service",
				"service", svcConfig.Name,
				"error", err)
			return fmt.Errorf("failed to start %s: %w", svcConfig.Name, err)
		}

		o.logger.Info("Service started successfully", "service", svcConfig.Name)
	}

	o.mu.Lock()
	o.status = OrchestratorRunning
	o.mu.Unlock()

	// Start health monitoring
	go o.runHealthMonitor(ctx)

	o.logger.Info("All services started successfully")
	return nil
}

// checkExternalDependencies verifies external dependencies are available
func (o *Orchestrator) checkExternalDependencies(ctx context.Context) error {
	for name, dep := range o.config.Dependencies {
		if !dep.Required {
			continue
		}

		o.logger.Info("Checking external dependency", "name", dep.Name, "type", dep.Type)

		switch dep.Type {
		case "http":
			if err := o.checkHTTPDependency(ctx, dep.URL); err != nil {
				return fmt.Errorf("%s not available: %w", dep.Name, err)
			}
		default:
			o.logger.Warn("Unknown dependency type", "name", name, "type", dep.Type)
		}

		o.logger.Info("External dependency available", "name", dep.Name)
	}
	return nil
}

// checkHTTPDependency checks if an HTTP endpoint is reachable
func (o *Orchestrator) checkHTTPDependency(ctx context.Context, url string) error {
	client := &http.Client{Timeout: 5 * time.Second}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	return nil
}

// waitForDependencies waits for service dependencies to be running
func (o *Orchestrator) waitForDependencies(ctx context.Context, svc ServiceConfig) error {
	if len(svc.Dependencies) == 0 {
		return nil
	}

	timeout := o.config.Orchestrator.GetStartupTimeout()
	deadline := time.Now().Add(timeout)

	for _, depName := range svc.Dependencies {
		o.logger.Debug("Waiting for dependency", "service", svc.Name, "dependency", depName)

		for time.Now().Before(deadline) {
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
			}

			o.mu.RLock()
			dep, exists := o.services[depName]
			o.mu.RUnlock()

			if !exists {
				return fmt.Errorf("unknown dependency: %s", depName)
			}

			dep.mu.RLock()
			status := dep.Status
			healthy := dep.Healthy
			dep.mu.RUnlock()

			if status == StatusRunning && healthy {
				break
			}

			if time.Now().After(deadline) {
				return fmt.Errorf("timeout waiting for dependency %s (status: %s, healthy: %v)",
					depName, status, healthy)
			}

			time.Sleep(500 * time.Millisecond)
		}
	}

	return nil
}

// checkPortConflict checks if the service's port is already in use
func (o *Orchestrator) checkPortConflict(svc ServiceConfig) *PortConflict {
	port := svc.GetPrimaryPort()
	if port == 0 {
		return nil
	}

	conn, err := net.DialTimeout("tcp", fmt.Sprintf("localhost:%d", port), time.Second)
	if err != nil {
		return nil // Port is free
	}
	conn.Close()

	// Port is in use - try to find the PID
	pid := o.findProcessOnPort(port)

	return &PortConflict{
		Port:    port,
		PID:     pid,
		Service: svc.ShortName,
	}
}

// findProcessOnPort attempts to find the PID of the process using a port
func (o *Orchestrator) findProcessOnPort(port int) int {
	// Try lsof first (macOS and Linux)
	cmd := exec.Command("lsof", "-t", fmt.Sprintf("-i:%d", port), "-sTCP:LISTEN")
	output, err := cmd.Output()
	if err == nil {
		pidStr := strings.TrimSpace(string(output))
		if pid, err := strconv.Atoi(strings.Split(pidStr, "\n")[0]); err == nil {
			return pid
		}
	}
	return 0
}

// handlePortConflict handles a detected port conflict
func (o *Orchestrator) handlePortConflict(ctx context.Context, svc ServiceConfig, conflict *PortConflict) error {
	o.logger.Warn("Port conflict detected",
		"service", svc.Name,
		"port", conflict.Port,
		"existingPID", conflict.PID)

	// First, try to adopt the existing service (check if it's healthy)
	if o.tryAdoptService(ctx, svc, conflict.Port) {
		o.logger.Info("Adopted existing service",
			"service", svc.Name,
			"port", conflict.Port)
		return nil
	}

	// Can't adopt - need to kill the existing process
	if conflict.PID > 0 {
		o.logger.Info("Terminating conflicting process",
			"service", svc.Name,
			"pid", conflict.PID)

		// Try graceful shutdown first
		if err := syscall.Kill(conflict.PID, syscall.SIGTERM); err != nil {
			o.logger.Warn("Failed to send SIGTERM", "pid", conflict.PID, "error", err)
		}

		// Wait for process to terminate
		for i := 0; i < 10; i++ {
			time.Sleep(500 * time.Millisecond)
			if !o.isProcessRunning(conflict.PID) {
				break
			}
		}

		// Force kill if still running
		if o.isProcessRunning(conflict.PID) {
			o.logger.Warn("Force killing process", "pid", conflict.PID)
			if err := syscall.Kill(conflict.PID, syscall.SIGKILL); err != nil {
				return fmt.Errorf("failed to kill process %d: %w", conflict.PID, err)
			}
		}

		// Wait for port to become free
		time.Sleep(2 * time.Second)
	}

	return nil
}

// tryAdoptService tries to adopt an existing service on a port
func (o *Orchestrator) tryAdoptService(ctx context.Context, svc ServiceConfig, port int) bool {
	// Try to health check the existing service
	healthy := false

	switch svc.HealthCheck.Type {
	case "grpc":
		healthy = o.checkGRPCHealth(ctx, port)
	case "http":
		endpoint := svc.HealthCheck.Endpoint
		if endpoint == "" {
			endpoint = "/health"
		}
		healthy = o.checkHTTPHealth(ctx, port, endpoint)
	}

	if healthy {
		o.mu.Lock()
		if managedSvc, exists := o.services[svc.ShortName]; exists {
			managedSvc.mu.Lock()
			managedSvc.Status = StatusRunning
			managedSvc.Healthy = true
			managedSvc.StartedAt = time.Now() // We don't know actual start time
			managedSvc.LastHealthCheck = time.Now()
			managedSvc.mu.Unlock()
		}
		o.mu.Unlock()

		o.emitEvent(EventStarted, svc.ShortName, "Adopted existing service")
		return true
	}

	return false
}

// isProcessRunning checks if a process is running
func (o *Orchestrator) isProcessRunning(pid int) bool {
	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	err = process.Signal(syscall.Signal(0))
	return err == nil
}

// startServiceWithRetry starts a service with retry logic
func (o *Orchestrator) startServiceWithRetry(ctx context.Context, svc ServiceConfig) error {
	var lastErr error

	for attempt := 1; attempt <= svc.MaxRetries; attempt++ {
		o.logger.Info("Starting service",
			"service", svc.Name,
			"attempt", attempt,
			"maxRetries", svc.MaxRetries)

		if err := o.startService(ctx, svc); err != nil {
			lastErr = err
			o.logger.Warn("Service start failed, retrying",
				"service", svc.Name,
				"attempt", attempt,
				"error", err)

			// Update service status
			o.mu.Lock()
			if managedSvc, exists := o.services[svc.ShortName]; exists {
				managedSvc.mu.Lock()
				managedSvc.RestartCount++
				managedSvc.LastError = err.Error()
				managedSvc.mu.Unlock()
			}
			o.mu.Unlock()

			// Wait before retry
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(2 * time.Second):
			}
			continue
		}

		// Wait for service to become healthy
		timeout := o.config.Orchestrator.GetStartupTimeout()
		if err := o.waitForHealthy(ctx, svc, timeout); err != nil {
			lastErr = err
			o.logger.Warn("Service not healthy, retrying",
				"service", svc.Name,
				"attempt", attempt,
				"error", err)
			o.stopService(ctx, svc.ShortName)
			continue
		}

		return nil
	}

	return fmt.Errorf("failed after %d attempts: %w", svc.MaxRetries, lastErr)
}

// startService starts a single service
func (o *Orchestrator) startService(ctx context.Context, svc ServiceConfig) error {
	o.mu.Lock()
	managedSvc, exists := o.services[svc.ShortName]
	if !exists {
		o.mu.Unlock()
		return fmt.Errorf("service %s not found", svc.ShortName)
	}
	o.mu.Unlock()

	managedSvc.mu.Lock()
	if managedSvc.Status == StatusRunning || managedSvc.Status == StatusStarting {
		managedSvc.mu.Unlock()
		return fmt.Errorf("service %s is already running or starting", svc.Name)
	}
	managedSvc.Status = StatusStarting
	managedSvc.mu.Unlock()

	o.emitEvent(EventStarted, svc.ShortName, "Starting service")

	// Build command
	// IMPORTANT: Use exec.Command instead of exec.CommandContext!
	// The child process should NOT be tied to the request context.
	// When the gRPC request context ends, the child would be killed.
	var cmd *exec.Cmd
	if len(svc.Command) > 0 {
		cmd = exec.Command(svc.Command[0], svc.Command[1:]...)
	} else {
		cmd = exec.Command(o.config.Orchestrator.BinaryPath, "serve", svc.ShortName)
	}

	cmd.Env = os.Environ()
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	// Start the process
	if err := cmd.Start(); err != nil {
		managedSvc.mu.Lock()
		managedSvc.Status = StatusFailed
		managedSvc.LastError = err.Error()
		managedSvc.mu.Unlock()
		o.emitEvent(EventFailed, svc.ShortName, err.Error())
		return fmt.Errorf("failed to start: %w", err)
	}

	managedSvc.mu.Lock()
	managedSvc.cmd = cmd
	managedSvc.PID = cmd.Process.Pid
	managedSvc.StartedAt = time.Now()
	managedSvc.mu.Unlock()

	o.logger.Info("Service process started",
		"service", svc.Name,
		"pid", cmd.Process.Pid)

	// Monitor process in background
	go o.monitorProcess(svc.ShortName, managedSvc)

	return nil
}

// waitForHealthy waits for a service to become healthy
func (o *Orchestrator) waitForHealthy(ctx context.Context, svc ServiceConfig, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	checkInterval := 500 * time.Millisecond

	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if o.performHealthCheck(ctx, svc) {
			o.mu.Lock()
			if managedSvc, exists := o.services[svc.ShortName]; exists {
				managedSvc.mu.Lock()
				managedSvc.Status = StatusRunning
				managedSvc.Healthy = true
				managedSvc.LastHealthCheck = time.Now()
				managedSvc.mu.Unlock()
			}
			o.mu.Unlock()

			o.emitEvent(EventHealthCheckPassed, svc.ShortName, "Service is healthy")
			return nil
		}

		time.Sleep(checkInterval)
	}

	return fmt.Errorf("timeout waiting for service to become healthy")
}

// performHealthCheck performs a health check on a service
func (o *Orchestrator) performHealthCheck(ctx context.Context, svc ServiceConfig) bool {
	port := svc.GetPrimaryPort()
	if port == 0 {
		return true // No port to check
	}

	switch svc.HealthCheck.Type {
	case "grpc":
		return o.checkGRPCHealth(ctx, port)
	case "http":
		endpoint := svc.HealthCheck.Endpoint
		if endpoint == "" {
			endpoint = "/health"
		}
		return o.checkHTTPHealth(ctx, port, endpoint)
	case "tcp":
		return o.checkTCPHealth(port)
	default:
		// Default to TCP check
		return o.checkTCPHealth(port)
	}
}

// checkGRPCHealth performs a gRPC health check
func (o *Orchestrator) checkGRPCHealth(ctx context.Context, port int) bool {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	conn, err := grpc.DialContext(ctx,
		fmt.Sprintf("localhost:%d", port),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	if err != nil {
		return false
	}
	defer conn.Close()

	client := grpc_health_v1.NewHealthClient(conn)
	resp, err := client.Check(ctx, &grpc_health_v1.HealthCheckRequest{})
	if err != nil {
		return false
	}

	return resp.Status == grpc_health_v1.HealthCheckResponse_SERVING
}

// checkHTTPHealth performs an HTTP health check
func (o *Orchestrator) checkHTTPHealth(ctx context.Context, port int, endpoint string) bool {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	url := fmt.Sprintf("http://localhost:%d%s", port, endpoint)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return false
	}

	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	return resp.StatusCode >= 200 && resp.StatusCode < 400
}

// checkTCPHealth performs a TCP connectivity check
func (o *Orchestrator) checkTCPHealth(port int) bool {
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("localhost:%d", port), 3*time.Second)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

// monitorProcess monitors a running process and handles termination
func (o *Orchestrator) monitorProcess(name string, svc *ManagedService) {
	svc.mu.RLock()
	cmd := svc.cmd
	svc.mu.RUnlock()

	if cmd == nil || cmd.Process == nil {
		return
	}

	// Wait for process to exit
	err := cmd.Wait()

	svc.mu.Lock()
	prevStatus := svc.Status
	if err != nil {
		svc.Status = StatusFailed
		svc.LastError = err.Error()
		o.logger.Warn("Service exited with error", "service", name, "error", err)
	} else {
		svc.Status = StatusStopped
		o.logger.Info("Service exited normally", "service", name)
	}
	svc.PID = 0
	svc.cmd = nil
	svc.Healthy = false
	svc.mu.Unlock()

	if prevStatus == StatusRunning {
		o.emitEvent(EventStopped, name, "Process terminated")
	}
}

// runHealthMonitor continuously monitors service health
func (o *Orchestrator) runHealthMonitor(ctx context.Context) {
	interval := o.config.Orchestrator.GetHealthCheckInterval()
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-o.stopCh:
			return
		case <-ticker.C:
			o.checkAllServices(ctx)
		}
	}
}

// checkAllServices performs health checks on all running services
func (o *Orchestrator) checkAllServices(ctx context.Context) {
	o.mu.RLock()
	services := make([]*ManagedService, 0, len(o.services))
	configs := make([]ServiceConfig, 0, len(o.services))
	for _, svc := range o.services {
		services = append(services, svc)
		configs = append(configs, svc.Config)
	}
	o.mu.RUnlock()

	for i, svc := range services {
		svc.mu.RLock()
		status := svc.Status
		svc.mu.RUnlock()

		if status != StatusRunning {
			continue
		}

		healthy := o.performHealthCheck(ctx, configs[i])

		svc.mu.Lock()
		svc.LastHealthCheck = time.Now()
		prevHealthy := svc.Healthy
		svc.Healthy = healthy
		svc.mu.Unlock()

		if !healthy && prevHealthy {
			o.logger.Warn("Service became unhealthy",
				"service", configs[i].Name,
				"restartCount", svc.RestartCount)
			o.emitEvent(EventHealthCheckFailed, configs[i].ShortName, "Health check failed")

			// Auto-restart if under max retries
			if svc.RestartCount < configs[i].MaxRetries {
				go o.restartService(ctx, configs[i].ShortName)
			}
		} else if healthy && !prevHealthy {
			o.emitEvent(EventHealthCheckPassed, configs[i].ShortName, "Health check passed")
		}
	}
}

// restartService restarts a failed service
func (o *Orchestrator) restartService(ctx context.Context, name string) {
	o.logger.Info("Restarting service", "service", name)
	o.emitEvent(EventRestarting, name, "Restarting service")

	o.mu.RLock()
	svc, exists := o.services[name]
	o.mu.RUnlock()

	if !exists {
		return
	}

	svc.mu.Lock()
	svc.RestartCount++
	config := svc.Config
	svc.mu.Unlock()

	// Stop if running
	o.stopService(ctx, name)

	// Wait a bit
	time.Sleep(time.Second)

	// Try to start
	if err := o.startServiceWithRetry(ctx, config); err != nil {
		o.logger.Error("Failed to restart service", "service", name, "error", err)
	}
}

// StopAll stops all services in reverse order
func (o *Orchestrator) StopAll(ctx context.Context) error {
	o.mu.Lock()
	o.status = OrchestratorStopping
	o.mu.Unlock()

	// Signal health monitor to stop
	close(o.stopCh)

	o.logger.Info("Stopping all services")

	// Get services in reverse start order
	sortedServices := o.config.GetServicesSortedByStartOrder()
	for i, j := 0, len(sortedServices)-1; i < j; i, j = i+1, j-1 {
		sortedServices[i], sortedServices[j] = sortedServices[j], sortedServices[i]
	}

	var lastErr error
	for _, svc := range sortedServices {
		if err := o.stopService(ctx, svc.ShortName); err != nil {
			o.logger.Error("Failed to stop service",
				"service", svc.Name,
				"error", err)
			lastErr = err
		}
	}

	o.mu.Lock()
	o.status = OrchestratorStopped
	o.mu.Unlock()

	o.logger.Info("All services stopped")
	return lastErr
}

// stopService stops a single service
func (o *Orchestrator) stopService(ctx context.Context, name string) error {
	o.mu.RLock()
	svc, exists := o.services[name]
	o.mu.RUnlock()

	if !exists {
		return fmt.Errorf("service %s not found", name)
	}

	svc.mu.Lock()
	if svc.Status == StatusStopped || svc.Status == StatusStopping {
		svc.mu.Unlock()
		return nil
	}

	if svc.cmd == nil || svc.cmd.Process == nil {
		svc.Status = StatusStopped
		svc.mu.Unlock()
		return nil
	}

	svc.Status = StatusStopping
	proc := svc.cmd.Process
	pid := svc.PID
	svc.mu.Unlock()

	o.logger.Info("Stopping service", "service", name, "pid", pid)

	// Send SIGTERM to process group
	if err := syscall.Kill(-proc.Pid, syscall.SIGTERM); err != nil {
		o.logger.Warn("Failed to send SIGTERM", "service", name, "error", err)
	}

	// Wait for graceful shutdown
	timeout := o.config.Orchestrator.GetShutdownTimeout()
	done := make(chan struct{})
	go func() {
		for i := 0; i < int(timeout.Seconds()*10); i++ {
			svc.mu.RLock()
			status := svc.Status
			svc.mu.RUnlock()
			if status == StatusStopped || status == StatusFailed {
				close(done)
				return
			}
			time.Sleep(100 * time.Millisecond)
		}
		close(done)
	}()

	select {
	case <-done:
		svc.mu.RLock()
		status := svc.Status
		svc.mu.RUnlock()
		if status == StatusStopped || status == StatusFailed {
			o.emitEvent(EventStopped, name, "Service stopped gracefully")
			return nil
		}
	case <-ctx.Done():
		return ctx.Err()
	}

	// Force kill
	o.logger.Warn("Force killing service", "service", name)
	if err := syscall.Kill(-proc.Pid, syscall.SIGKILL); err != nil {
		return fmt.Errorf("failed to kill service %s: %w", name, err)
	}

	svc.mu.Lock()
	svc.Status = StatusStopped
	svc.PID = 0
	svc.cmd = nil
	svc.Healthy = false
	svc.mu.Unlock()

	o.emitEvent(EventStopped, name, "Service killed")
	return nil
}

// StartService starts a single service
func (o *Orchestrator) StartService(ctx context.Context, name string) error {
	o.mu.RLock()
	svc, exists := o.services[name]
	o.mu.RUnlock()

	if !exists {
		return fmt.Errorf("service %s not found", name)
	}

	return o.startServiceWithRetry(ctx, svc.Config)
}

// StopService stops a single service
func (o *Orchestrator) StopService(ctx context.Context, name string) error {
	return o.stopService(ctx, name)
}

// RestartService restarts a single service
func (o *Orchestrator) RestartService(ctx context.Context, name string) error {
	if err := o.stopService(ctx, name); err != nil {
		o.logger.Warn("Error stopping service for restart", "service", name, "error", err)
	}
	time.Sleep(500 * time.Millisecond)
	return o.StartService(ctx, name)
}

// GetStatus returns the orchestrator status
func (o *Orchestrator) GetStatus() OrchestratorStatus {
	o.mu.RLock()
	defer o.mu.RUnlock()
	return o.status
}

// GetServiceStatus returns the status of a specific service
func (o *Orchestrator) GetServiceStatus(name string) (*ManagedService, error) {
	o.mu.RLock()
	defer o.mu.RUnlock()

	svc, exists := o.services[name]
	if !exists {
		return nil, fmt.Errorf("service %s not found", name)
	}
	return svc, nil
}

// GetAllServiceStatus returns status of all services
func (o *Orchestrator) GetAllServiceStatus() map[string]*ManagedService {
	o.mu.RLock()
	defer o.mu.RUnlock()

	result := make(map[string]*ManagedService, len(o.services))
	for k, v := range o.services {
		result[k] = v
	}
	return result
}

// Subscribe returns a channel for service events
func (o *Orchestrator) Subscribe() chan ServiceEvent {
	o.subMu.Lock()
	defer o.subMu.Unlock()

	ch := make(chan ServiceEvent, 10)
	o.subscribers = append(o.subscribers, ch)
	return ch
}

// Unsubscribe removes a subscriber
func (o *Orchestrator) Unsubscribe(ch chan ServiceEvent) {
	o.subMu.Lock()
	defer o.subMu.Unlock()

	for i, sub := range o.subscribers {
		if sub == ch {
			o.subscribers = append(o.subscribers[:i], o.subscribers[i+1:]...)
			close(ch)
			return
		}
	}
}

// emitEvent sends a service event
func (o *Orchestrator) emitEvent(eventType EventType, service, message string) {
	event := ServiceEvent{
		Type:        eventType,
		ServiceName: service,
		Message:     message,
		Timestamp:   time.Now(),
	}

	select {
	case o.eventCh <- event:
	default:
		o.logger.Warn("Event channel full, dropping event")
	}
}

// dispatchEvents dispatches events to all subscribers
func (o *Orchestrator) dispatchEvents() {
	for event := range o.eventCh {
		o.subMu.RLock()
		for _, ch := range o.subscribers {
			select {
			case ch <- event:
			default:
				// Subscriber channel full, skip
			}
		}
		o.subMu.RUnlock()
	}
}

// Close closes the orchestrator
func (o *Orchestrator) Close() {
	close(o.eventCh)

	o.subMu.Lock()
	for _, ch := range o.subscribers {
		close(ch)
	}
	o.subscribers = nil
	o.subMu.Unlock()
}

// GetStartedAt returns when the orchestrator started
func (o *Orchestrator) GetStartedAt() time.Time {
	o.mu.RLock()
	defer o.mu.RUnlock()
	return o.startedAt
}

// GetConfig returns the loaded configuration
func (o *Orchestrator) GetConfig() *ServicesConfig {
	return o.config
}

// ManagedServiceInfo contains thread-safe access to service info
type ManagedServiceInfo struct {
	Name         string
	ShortName    string
	Status       ServiceStatus
	Healthy      bool
	PID          int
	Port         int
	StartedAt    time.Time
	RestartCount int
	LastError    string
	Dependencies []string
	Version      string
}

// GetInfo returns a thread-safe snapshot of the service info
func (svc *ManagedService) GetInfo() ManagedServiceInfo {
	svc.mu.RLock()
	defer svc.mu.RUnlock()
	return ManagedServiceInfo{
		Name:         svc.Config.Name,
		ShortName:    svc.Config.ShortName,
		Status:       svc.Status,
		Healthy:      svc.Healthy,
		PID:          svc.PID,
		Port:         svc.Config.GetPrimaryPort(),
		StartedAt:    svc.StartedAt,
		RestartCount: svc.RestartCount,
		LastError:    svc.LastError,
		Dependencies: svc.Config.Dependencies,
		Version:      svc.Config.Version,
	}
}
