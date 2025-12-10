// ============================================================================
// meinDENKWERK (mDW) - Lokale KI-Plattform
// ============================================================================
//
// Package:     procmgr
// Description: Process Manager for service lifecycle management
// Author:      Mike Stoffels with Claude
// Created:     2025-12-06
// License:     MIT
// ============================================================================

package procmgr

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"sync"
	"syscall"
	"time"

	"github.com/msto63/mDW/pkg/core/logging"
	"github.com/msto63/mDW/pkg/core/version"
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

// ServiceConfig holds configuration for a managed service
type ServiceConfig struct {
	Name        string
	Version     string
	Command     string
	Args        []string
	Env         map[string]string
	Port        int
	HealthCheck func(ctx context.Context) error
	StartupTime time.Duration // Time to wait for service to become healthy
}

// ManagedService represents a service managed by the process manager
type ManagedService struct {
	Config       ServiceConfig
	Status       ServiceStatus
	PID          int
	StartedAt    time.Time
	RestartCount int
	LastError    string
	cmd          *exec.Cmd
	mu           sync.RWMutex
}

// ProcessManager manages service processes
type ProcessManager struct {
	services     map[string]*ManagedService
	binaryPath   string
	configPath   string
	logger       *logging.Logger
	mu           sync.RWMutex
	statusCh     chan StatusEvent
	subscribers  []chan StatusEvent
	subscriberMu sync.RWMutex
}

// StatusEvent represents a status change event
type StatusEvent struct {
	ServiceName    string
	PreviousStatus ServiceStatus
	CurrentStatus  ServiceStatus
	Message        string
	Timestamp      time.Time
}

// Config holds process manager configuration
type Config struct {
	BinaryPath string // Path to mdw binary
	ConfigPath string // Path to config file
}

// DefaultConfig returns default configuration
func DefaultConfig() Config {
	return Config{
		BinaryPath: "./bin/mdw",
		ConfigPath: "./configs/config.toml",
	}
}

// New creates a new process manager
func New(cfg Config) *ProcessManager {
	pm := &ProcessManager{
		services:    make(map[string]*ManagedService),
		binaryPath:  cfg.BinaryPath,
		configPath:  cfg.ConfigPath,
		logger:      logging.New("procmgr"),
		statusCh:    make(chan StatusEvent, 100),
		subscribers: make([]chan StatusEvent, 0),
	}

	// Register known services
	pm.registerKnownServices()

	// Start event dispatcher
	go pm.dispatchEvents()

	return pm
}

// registerKnownServices registers all known mDW services
func (pm *ProcessManager) registerKnownServices() {
	services := []ServiceConfig{
		{Name: "turing", Version: version.Turing, Port: 9200, StartupTime: 10 * time.Second},
		{Name: "hypatia", Version: version.Hypatia, Port: 9220, StartupTime: 10 * time.Second},
		{Name: "babbage", Version: version.Babbage, Port: 9150, StartupTime: 5 * time.Second},
		{Name: "leibniz", Version: version.Leibniz, Port: 9140, StartupTime: 5 * time.Second},
		{Name: "kant", Version: version.Kant, Port: 8080, StartupTime: 5 * time.Second},
		{Name: "bayes", Version: version.Bayes, Port: 9120, StartupTime: 5 * time.Second},
	}

	for _, cfg := range services {
		pm.services[cfg.Name] = &ManagedService{
			Config: cfg,
			Status: StatusStopped,
		}
	}
}

// StartService starts a service by name
func (pm *ProcessManager) StartService(ctx context.Context, name string) error {
	pm.mu.Lock()
	svc, exists := pm.services[name]
	pm.mu.Unlock()

	if !exists {
		return fmt.Errorf("service %s not found", name)
	}

	svc.mu.Lock()
	if svc.Status == StatusRunning || svc.Status == StatusStarting {
		svc.mu.Unlock()
		return fmt.Errorf("service %s is already running or starting", name)
	}

	prevStatus := svc.Status
	svc.Status = StatusStarting
	svc.mu.Unlock()

	pm.emitEvent(name, prevStatus, StatusStarting, "Starting service")

	// Prepare command
	args := []string{"serve", "--service", name}
	if pm.configPath != "" {
		args = append(args, "--config", pm.configPath)
	}

	cmd := exec.CommandContext(ctx, pm.binaryPath, args...)
	cmd.Env = os.Environ()

	// Add custom environment
	svc.mu.RLock()
	for k, v := range svc.Config.Env {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
	}
	svc.mu.RUnlock()

	// Set process group for clean termination
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	// Start the process
	if err := cmd.Start(); err != nil {
		svc.mu.Lock()
		svc.Status = StatusFailed
		svc.LastError = err.Error()
		svc.mu.Unlock()
		pm.emitEvent(name, StatusStarting, StatusFailed, err.Error())
		return fmt.Errorf("failed to start service %s: %w", name, err)
	}

	svc.mu.Lock()
	svc.cmd = cmd
	svc.PID = cmd.Process.Pid
	svc.StartedAt = time.Now()
	svc.mu.Unlock()

	pm.logger.Info("Service process started", "service", name, "pid", svc.PID)

	// Wait for startup in background
	go pm.waitForStartup(ctx, name, svc)

	// Monitor process in background
	go pm.monitorProcess(name, svc)

	return nil
}

// waitForStartup waits for a service to become healthy
func (pm *ProcessManager) waitForStartup(ctx context.Context, name string, svc *ManagedService) {
	startupTime := svc.Config.StartupTime
	if startupTime == 0 {
		startupTime = 10 * time.Second
	}

	// Simple wait - in production you'd check health endpoint
	select {
	case <-time.After(startupTime):
		svc.mu.Lock()
		if svc.Status == StatusStarting {
			svc.Status = StatusRunning
			svc.mu.Unlock()
			pm.emitEvent(name, StatusStarting, StatusRunning, "Service is running")
			pm.logger.Info("Service is now running", "service", name)
		} else {
			svc.mu.Unlock()
		}
	case <-ctx.Done():
		return
	}
}

// monitorProcess monitors a running process and handles termination
func (pm *ProcessManager) monitorProcess(name string, svc *ManagedService) {
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
		pm.logger.Warn("Service exited with error", "service", name, "error", err)
	} else {
		svc.Status = StatusStopped
		pm.logger.Info("Service exited normally", "service", name)
	}
	svc.PID = 0
	svc.cmd = nil
	svc.mu.Unlock()

	pm.emitEvent(name, prevStatus, svc.Status, "Process terminated")
}

// StopService stops a service by name
func (pm *ProcessManager) StopService(ctx context.Context, name string, force bool) error {
	pm.mu.Lock()
	svc, exists := pm.services[name]
	pm.mu.Unlock()

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

	prevStatus := svc.Status
	svc.Status = StatusStopping
	proc := svc.cmd.Process
	svc.mu.Unlock()

	pm.emitEvent(name, prevStatus, StatusStopping, "Stopping service")
	pm.logger.Info("Stopping service", "service", name, "pid", proc.Pid, "force", force)

	// Try graceful shutdown first
	if !force {
		// Send SIGTERM to process group
		if err := syscall.Kill(-proc.Pid, syscall.SIGTERM); err != nil {
			pm.logger.Warn("Failed to send SIGTERM", "service", name, "error", err)
		}

		// Wait up to 10 seconds for graceful shutdown
		done := make(chan struct{})
		go func() {
			for i := 0; i < 100; i++ {
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
				return nil
			}
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	// Force kill
	pm.logger.Warn("Force killing service", "service", name)
	if err := syscall.Kill(-proc.Pid, syscall.SIGKILL); err != nil {
		pm.logger.Error("Failed to kill service", "service", name, "error", err)
		return fmt.Errorf("failed to kill service %s: %w", name, err)
	}

	svc.mu.Lock()
	svc.Status = StatusStopped
	svc.PID = 0
	svc.cmd = nil
	svc.mu.Unlock()

	pm.emitEvent(name, StatusStopping, StatusStopped, "Service stopped")
	return nil
}

// RestartService restarts a service
func (pm *ProcessManager) RestartService(ctx context.Context, name string) error {
	if err := pm.StopService(ctx, name, false); err != nil {
		pm.logger.Warn("Error stopping service for restart", "service", name, "error", err)
	}

	// Wait a bit for cleanup
	time.Sleep(500 * time.Millisecond)

	return pm.StartService(ctx, name)
}

// GetServiceStatus returns the status of a service
func (pm *ProcessManager) GetServiceStatus(name string) (*ManagedService, error) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	svc, exists := pm.services[name]
	if !exists {
		return nil, fmt.Errorf("service %s not found", name)
	}

	return svc, nil
}

// GetAllServiceStatus returns status of all services
func (pm *ProcessManager) GetAllServiceStatus() map[string]*ManagedService {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	result := make(map[string]*ManagedService, len(pm.services))
	for k, v := range pm.services {
		result[k] = v
	}
	return result
}

// Uptime returns uptime in seconds for a service
func (svc *ManagedService) Uptime() int64 {
	svc.mu.RLock()
	defer svc.mu.RUnlock()

	if svc.Status != StatusRunning || svc.StartedAt.IsZero() {
		return 0
	}
	return int64(time.Since(svc.StartedAt).Seconds())
}

// GetStatus returns the current status and related info (thread-safe)
func (svc *ManagedService) GetStatus() (ServiceStatus, int, time.Time, int, string) {
	svc.mu.RLock()
	defer svc.mu.RUnlock()
	return svc.Status, svc.PID, svc.StartedAt, svc.RestartCount, svc.LastError
}

// GetConfig returns the service config
func (svc *ManagedService) GetConfig() ServiceConfig {
	svc.mu.RLock()
	defer svc.mu.RUnlock()
	return svc.Config
}

// Subscribe returns a channel for status events
func (pm *ProcessManager) Subscribe() chan StatusEvent {
	pm.subscriberMu.Lock()
	defer pm.subscriberMu.Unlock()

	ch := make(chan StatusEvent, 10)
	pm.subscribers = append(pm.subscribers, ch)
	return ch
}

// Unsubscribe removes a subscriber
func (pm *ProcessManager) Unsubscribe(ch chan StatusEvent) {
	pm.subscriberMu.Lock()
	defer pm.subscriberMu.Unlock()

	for i, sub := range pm.subscribers {
		if sub == ch {
			pm.subscribers = append(pm.subscribers[:i], pm.subscribers[i+1:]...)
			close(ch)
			return
		}
	}
}

// emitEvent sends a status event
func (pm *ProcessManager) emitEvent(name string, prev, curr ServiceStatus, msg string) {
	event := StatusEvent{
		ServiceName:    name,
		PreviousStatus: prev,
		CurrentStatus:  curr,
		Message:        msg,
		Timestamp:      time.Now(),
	}

	select {
	case pm.statusCh <- event:
	default:
		pm.logger.Warn("Status event channel full, dropping event")
	}
}

// dispatchEvents dispatches events to all subscribers
func (pm *ProcessManager) dispatchEvents() {
	for event := range pm.statusCh {
		pm.subscriberMu.RLock()
		for _, ch := range pm.subscribers {
			select {
			case ch <- event:
			default:
				// Subscriber channel full, skip
			}
		}
		pm.subscriberMu.RUnlock()
	}
}

// Close closes the process manager
func (pm *ProcessManager) Close() {
	close(pm.statusCh)

	pm.subscriberMu.Lock()
	for _, ch := range pm.subscribers {
		close(ch)
	}
	pm.subscribers = nil
	pm.subscriberMu.Unlock()
}

// StartAll starts all registered services
func (pm *ProcessManager) StartAll(ctx context.Context) error {
	pm.mu.RLock()
	names := make([]string, 0, len(pm.services))
	for name := range pm.services {
		names = append(names, name)
	}
	pm.mu.RUnlock()

	var lastErr error
	for _, name := range names {
		if err := pm.StartService(ctx, name); err != nil {
			pm.logger.Error("Failed to start service", "service", name, "error", err)
			lastErr = err
		}
		// Small delay between service starts
		time.Sleep(500 * time.Millisecond)
	}

	return lastErr
}

// StopAll stops all running services
func (pm *ProcessManager) StopAll(ctx context.Context) error {
	pm.mu.RLock()
	names := make([]string, 0, len(pm.services))
	for name := range pm.services {
		names = append(names, name)
	}
	pm.mu.RUnlock()

	var lastErr error
	for _, name := range names {
		if err := pm.StopService(ctx, name, false); err != nil {
			pm.logger.Error("Failed to stop service", "service", name, "error", err)
			lastErr = err
		}
	}

	return lastErr
}
