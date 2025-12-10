// ============================================================================
// meinDENKWERK (mDW) - Lokale KI-Plattform
// ============================================================================
//
// Package:     controlcenter
// Description: Service management for mDW Control Center
// Author:      Mike Stoffels with Claude
// Created:     2025-12-06
// License:     MIT
// ============================================================================

package controlcenter

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	commonpb "github.com/msto63/mDW/api/gen/common"
	russellpb "github.com/msto63/mDW/api/gen/russell"
)

// ServiceStatus represents the status of a service
type ServiceStatus int

const (
	ServiceStopped ServiceStatus = iota
	ServiceStarting
	ServiceRunning
	ServiceStopping
	ServiceError
	ServiceUnknown
)

// String returns the status as a string
func (s ServiceStatus) String() string {
	switch s {
	case ServiceStopped:
		return "stopped"
	case ServiceStarting:
		return "starting..."
	case ServiceRunning:
		return "running"
	case ServiceStopping:
		return "stopping..."
	case ServiceError:
		return "error"
	default:
		return "unknown"
	}
}

// Service represents a mDW service
type Service struct {
	Name         string
	ShortName    string
	Description  string
	Port         int
	GRPCPort     int
	HTTPPort     int
	Status       ServiceStatus
	Managed      bool // true if registered with Russell
	PID          int
	Uptime       time.Duration
	LastCheck    time.Time
	Error        string
	Version      string
	RestartCount int
	StartedAt    time.Time
}

// ServiceManager manages all mDW services
type ServiceManager struct {
	Services []Service
	mu       sync.RWMutex
}

// NewServiceManager creates a new service manager
func NewServiceManager() *ServiceManager {
	return &ServiceManager{
		Services: []Service{
			{
				Name:        "Kant",
				ShortName:   "kant",
				Description: "API Gateway (HTTP/REST)",
				HTTPPort:    8080,
				Status:      ServiceUnknown,
				Version:     "1.0.0",
			},
			{
				Name:        "Russell",
				ShortName:   "russell",
				Description: "Service Discovery & Orchestration",
				GRPCPort:    9100,
				HTTPPort:    9101,
				Status:      ServiceUnknown,
				Version:     "1.0.0",
			},
			{
				Name:        "Turing",
				ShortName:   "turing",
				Description: "LLM Management & Inference",
				GRPCPort:    9200,
				HTTPPort:    9201,
				Status:      ServiceUnknown,
				Version:     "1.0.0",
			},
			{
				Name:        "Hypatia",
				ShortName:   "hypatia",
				Description: "RAG & Vector Search + Datasources",
				GRPCPort:    9220,
				HTTPPort:    9221,
				Status:      ServiceUnknown,
				Version:     "1.0.0",
			},
			{
				Name:        "Babbage",
				ShortName:   "babbage",
				Description: "NLP Processing",
				GRPCPort:    9150,
				HTTPPort:    9151,
				Status:      ServiceUnknown,
				Version:     "1.0.0",
			},
			{
				Name:        "Leibniz",
				ShortName:   "leibniz",
				Description: "Agentic AI & MCP",
				GRPCPort:    9140,
				HTTPPort:    9141,
				Status:      ServiceUnknown,
				Version:     "1.0.0",
			},
			{
				Name:        "Bayes",
				ShortName:   "bayes",
				Description: "Logging & Metrics",
				GRPCPort:    9120,
				HTTPPort:    9121,
				Status:      ServiceUnknown,
				Version:     "1.0.0",
			},
		},
	}
}

// RefreshStatus updates the status of all services
func (sm *ServiceManager) RefreshStatus() {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	// Try to get status from Russell's control API first
	if sm.checkRussellControlStatus() {
		return
	}

	// Fall back to direct port checking
	for i := range sm.Services {
		sm.checkServiceDirect(&sm.Services[i])
	}
}

// checkRussellControlStatus checks services via Russell's orchestrator API
// Returns true if Russell is reachable
func (sm *ServiceManager) checkRussellControlStatus() bool {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	conn, err := grpc.DialContext(ctx, "localhost:9100",
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	if err != nil {
		return false
	}
	defer conn.Close()

	client := russellpb.NewRussellServiceClient(conn)

	// Mark Russell as running since we connected
	for i := range sm.Services {
		if sm.Services[i].ShortName == "russell" {
			sm.Services[i].Status = ServiceRunning
			sm.Services[i].Managed = true // Russell manages itself
			sm.Services[i].LastCheck = time.Now()
		}
	}

	// Try the new GetOrchestratorStatus API first
	orchResp, err := client.GetOrchestratorStatus(ctx, &commonpb.Empty{})

	// Track which services were found in orchestrator response
	foundInOrchestrator := make(map[string]bool)

	if err == nil && orchResp != nil {
		// Update service statuses based on orchestrator response
		for _, svcStatus := range orchResp.GetServices() {
			foundInOrchestrator[svcStatus.GetShortName()] = true

			for i := range sm.Services {
				if sm.Services[i].ShortName == svcStatus.GetShortName() {
					// Mark as managed by orchestrator
					sm.Services[i].Managed = true

					// Convert proto status to ServiceStatus
					// Also check the healthy flag - if status is HEALTHY but not healthy, show as error
					switch svcStatus.GetStatus() {
					case russellpb.ServiceStatus_SERVICE_STATUS_HEALTHY:
						if svcStatus.GetHealthy() {
							sm.Services[i].Status = ServiceRunning
						} else {
							sm.Services[i].Status = ServiceError
						}
					case russellpb.ServiceStatus_SERVICE_STATUS_UNHEALTHY:
						sm.Services[i].Status = ServiceError
					case russellpb.ServiceStatus_SERVICE_STATUS_STOPPED:
						sm.Services[i].Status = ServiceStopped
					case russellpb.ServiceStatus_SERVICE_STATUS_FAILED:
						sm.Services[i].Status = ServiceError
					case russellpb.ServiceStatus_SERVICE_STATUS_STARTING:
						sm.Services[i].Status = ServiceStarting
					case russellpb.ServiceStatus_SERVICE_STATUS_STOPPING:
						sm.Services[i].Status = ServiceStopping
					default:
						sm.Services[i].Status = ServiceUnknown
					}

					// Update additional info from orchestrator
					sm.Services[i].PID = int(svcStatus.GetPid())
					// Only show uptime if service is running, otherwise reset to 0
					if sm.Services[i].Status == ServiceRunning {
						sm.Services[i].Uptime = time.Duration(svcStatus.GetUptimeSeconds()) * time.Second
					} else {
						sm.Services[i].Uptime = 0
						sm.Services[i].StartedAt = time.Time{} // Reset start time
					}
					sm.Services[i].RestartCount = int(svcStatus.GetRestartCount())
					sm.Services[i].LastCheck = time.Now()
					if svcStatus.GetVersion() != "" {
						sm.Services[i].Version = svcStatus.GetVersion()
					}
					if svcStatus.GetLastError() != "" {
						sm.Services[i].Error = svcStatus.GetLastError()
					} else {
						sm.Services[i].Error = "" // Clear error when no error
					}
				}
			}
		}
	} else {
		// Fallback to GetAllServiceStatus if orchestrator API not available
		resp, fallbackErr := client.GetAllServiceStatus(ctx, &commonpb.Empty{})
		if fallbackErr == nil && resp != nil {
			for _, svcStatus := range resp.GetServices() {
				foundInOrchestrator[svcStatus.GetName()] = true

				for i := range sm.Services {
					if sm.Services[i].ShortName == svcStatus.GetName() {
						sm.Services[i].Managed = true

						switch svcStatus.GetStatus() {
						case russellpb.ServiceStatus_SERVICE_STATUS_HEALTHY:
							sm.Services[i].Status = ServiceRunning
						case russellpb.ServiceStatus_SERVICE_STATUS_UNHEALTHY:
							sm.Services[i].Status = ServiceError
						case russellpb.ServiceStatus_SERVICE_STATUS_STOPPED:
							sm.Services[i].Status = ServiceStopped
						case russellpb.ServiceStatus_SERVICE_STATUS_FAILED:
							sm.Services[i].Status = ServiceError
						case russellpb.ServiceStatus_SERVICE_STATUS_STARTING:
							sm.Services[i].Status = ServiceStarting
						case russellpb.ServiceStatus_SERVICE_STATUS_STOPPING:
							sm.Services[i].Status = ServiceStopping
						default:
							sm.Services[i].Status = ServiceUnknown
						}

						sm.Services[i].PID = int(svcStatus.GetPid())
						// Only show uptime if service is running, otherwise reset to 0
						if sm.Services[i].Status == ServiceRunning {
							sm.Services[i].Uptime = time.Duration(svcStatus.GetUptimeSeconds()) * time.Second
						} else {
							sm.Services[i].Uptime = 0
							sm.Services[i].StartedAt = time.Time{} // Reset start time
						}
						sm.Services[i].RestartCount = int(svcStatus.GetRestartCount())
						sm.Services[i].LastCheck = time.Now()
						if svcStatus.GetVersion() != "" {
							sm.Services[i].Version = svcStatus.GetVersion()
						}
						if svcStatus.GetHealthMessage() != "" {
							sm.Services[i].Error = svcStatus.GetHealthMessage()
						} else {
							sm.Services[i].Error = "" // Clear error when no error
						}
					}
				}
			}
		}
	}

	// Mark services NOT found in orchestrator response as not managed
	for i := range sm.Services {
		if sm.Services[i].ShortName == "russell" {
			continue
		}
		if !foundInOrchestrator[sm.Services[i].ShortName] {
			sm.Services[i].Managed = false
		}
	}

	return true
}

// IsRussellRunning returns true if Russell is running
func (sm *ServiceManager) IsRussellRunning() bool {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	for _, svc := range sm.Services {
		if svc.ShortName == "russell" {
			return svc.Status == ServiceRunning
		}
	}
	return false
}

// checkServiceDirect checks a service by connecting to its port
func (sm *ServiceManager) checkServiceDirect(svc *Service) {
	svc.LastCheck = time.Now()

	// Check if service is in grace period (recently started)
	inGracePeriod := svc.Status == ServiceStarting &&
		!svc.StartedAt.IsZero() &&
		time.Since(svc.StartedAt) < startupGracePeriod

	// Kant uses HTTP only
	if svc.ShortName == "kant" {
		if sm.checkHTTP(svc.HTTPPort) {
			svc.Status = ServiceRunning
			if !svc.StartedAt.IsZero() {
				svc.Uptime = time.Since(svc.StartedAt)
			}
		} else if !inGracePeriod {
			// Only mark as stopped if not in grace period
			svc.Status = ServiceStopped
		}
		return
	}

	// All other services use gRPC
	port := svc.GRPCPort
	if port == 0 {
		port = svc.HTTPPort
	}
	if port == 0 {
		svc.Status = ServiceUnknown
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	conn, err := grpc.DialContext(ctx, fmt.Sprintf("localhost:%d", port),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	if err != nil {
		if !inGracePeriod {
			// Only mark as stopped if not in grace period
			svc.Status = ServiceStopped
		}
		return
	}
	conn.Close()

	svc.Status = ServiceRunning
	if !svc.StartedAt.IsZero() {
		svc.Uptime = time.Since(svc.StartedAt)
	}
}

// checkHTTP checks if an HTTP endpoint is responding
func (sm *ServiceManager) checkHTTP(port int) bool {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	url := fmt.Sprintf("http://localhost:%d/api/v1/health", port)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return false
	}

	client := &http.Client{Timeout: 1 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	return resp.StatusCode == http.StatusOK
}

// StartAll starts all services via Russell's orchestrator
func (sm *ServiceManager) StartAll() error {
	// First check if Russell is running
	if sm.IsRussellRunning() {
		// Try to use Russell's StartAllServices API
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()

		conn, err := grpc.DialContext(ctx, "localhost:9100",
			grpc.WithTransportCredentials(insecure.NewCredentials()),
			grpc.WithBlock(),
		)
		if err == nil {
			defer conn.Close()
			client := russellpb.NewRussellServiceClient(conn)

			// Mark all as starting
			sm.mu.Lock()
			for i := range sm.Services {
				if sm.Services[i].Status != ServiceRunning {
					sm.Services[i].Status = ServiceStarting
					sm.Services[i].StartedAt = time.Now()
				}
			}
			sm.mu.Unlock()

			// Call StartAllServices - Russell handles ordering, retries, dependencies
			resp, err := client.StartAllServices(ctx, &russellpb.StartAllRequest{})
			if err == nil && resp.GetSuccess() {
				return nil
			}
			// Fall through to manual start if orchestrator fails
		}
	}

	// Fallback: manual start (Russell first, then others)
	sm.RefreshStatus()

	sm.mu.Lock()

	// Count running and stopped services
	runningCount := 0
	stoppedIndices := []int{}
	for i, svc := range sm.Services {
		if svc.Status == ServiceRunning {
			runningCount++
		} else if svc.Status == ServiceStopped || svc.Status == ServiceUnknown || svc.Status == ServiceError {
			stoppedIndices = append(stoppedIndices, i)
		}
	}
	sm.mu.Unlock()

	if runningCount == len(sm.Services) {
		return fmt.Errorf("all services already running")
	}

	if len(stoppedIndices) == 0 {
		return fmt.Errorf("no stopped services to start")
	}

	startOrder := sm.getStartOrder(stoppedIndices)

	startedCount := 0
	for _, idx := range startOrder {
		if err := sm.startServiceInternal(idx); err == nil {
			startedCount++
		}
		time.Sleep(500 * time.Millisecond)
	}

	if startedCount == 0 {
		return fmt.Errorf("failed to start any services")
	}

	return nil
}

// getStartOrder returns indices in the correct startup order
// Russell should start first as the service discovery
func (sm *ServiceManager) getStartOrder(indices []int) []int {
	// Find Russell's index first
	var russellIdx = -1
	var otherIndices []int

	for _, idx := range indices {
		if sm.Services[idx].ShortName == "russell" {
			russellIdx = idx
		} else {
			otherIndices = append(otherIndices, idx)
		}
	}

	// Russell first, then others
	result := []int{}
	if russellIdx >= 0 {
		result = append(result, russellIdx)
	}
	result = append(result, otherIndices...)
	return result
}

// startupGracePeriod is how long to preserve "Starting" status before allowing refresh to override
const startupGracePeriod = 10 * time.Second

// startServiceInternal starts a single service (internal, without lock)
func (sm *ServiceManager) startServiceInternal(index int) error {
	sm.mu.Lock()
	svc := &sm.Services[index]
	svc.Status = ServiceStarting
	svc.StartedAt = time.Now() // Track when we started for grace period
	shortName := svc.ShortName
	sm.mu.Unlock()

	// Start the individual service
	cmd := exec.Command(findMDWBinary(), "serve", shortName)
	cmd.Stdout = nil
	cmd.Stderr = nil

	if err := cmd.Start(); err != nil {
		sm.mu.Lock()
		sm.Services[index].Status = ServiceError
		sm.Services[index].Error = err.Error()
		sm.mu.Unlock()
		return err
	}

	// Don't wait for the process - let RefreshStatus detect the status
	return nil
}

// StopAll stops all services via Russell's orchestrator
func (sm *ServiceManager) StopAll() error {
	// First check if Russell is running
	if sm.IsRussellRunning() {
		// Try to use Russell's StopAllServices API
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		conn, err := grpc.DialContext(ctx, "localhost:9100",
			grpc.WithTransportCredentials(insecure.NewCredentials()),
			grpc.WithBlock(),
		)
		if err == nil {
			defer conn.Close()
			client := russellpb.NewRussellServiceClient(conn)

			// Mark all as stopping
			sm.mu.Lock()
			for i := range sm.Services {
				sm.Services[i].Status = ServiceStopping
			}
			sm.mu.Unlock()

			// Call StopAllServices - Russell handles ordering, dependencies
			resp, err := client.StopAllServices(ctx, &russellpb.StopAllRequest{})
			if err == nil && resp.GetSuccess() {
				sm.mu.Lock()
				for i := range sm.Services {
					if sm.Services[i].ShortName != "russell" {
						sm.Services[i].Status = ServiceStopped
					}
				}
				sm.mu.Unlock()
				return nil
			}
			// Fall through to manual stop if orchestrator fails
		}
	}

	// Fallback: manual stop
	sm.mu.Lock()
	// Mark all as stopping
	for i := range sm.Services {
		sm.Services[i].Status = ServiceStopping
	}
	sm.mu.Unlock()

	// Kill all mdw serve processes (graceful with SIGTERM first)
	exec.Command("pkill", "-TERM", "-f", findMDWBinary()+" serve").Run()

	// Wait briefly for graceful shutdown
	time.Sleep(2 * time.Second)

	// Force kill any remaining processes
	exec.Command("pkill", "-KILL", "-f", findMDWBinary()+" serve").Run()

	sm.mu.Lock()
	for i := range sm.Services {
		sm.Services[i].Status = ServiceStopped
	}
	sm.mu.Unlock()

	return nil
}

// StartService starts a single service
func (sm *ServiceManager) StartService(index int) error {
	if index < 0 || index >= len(sm.Services) {
		return fmt.Errorf("invalid service index")
	}

	// Check current status first
	sm.mu.RLock()
	currentStatus := sm.Services[index].Status
	svcName := sm.Services[index].Name
	shortName := sm.Services[index].ShortName
	sm.mu.RUnlock()

	// If already running, don't start again
	if currentStatus == ServiceRunning {
		return fmt.Errorf("%s is already running", svcName)
	}

	// Try to use Russell's StartService API if available
	if sm.IsRussellRunning() && shortName != "russell" {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		conn, err := grpc.DialContext(ctx, "localhost:9100",
			grpc.WithTransportCredentials(insecure.NewCredentials()),
			grpc.WithBlock(),
		)
		if err == nil {
			defer conn.Close()
			client := russellpb.NewRussellServiceClient(conn)

			sm.mu.Lock()
			sm.Services[index].Status = ServiceStarting
			sm.Services[index].StartedAt = time.Now()
			sm.mu.Unlock()

			resp, err := client.StartService(ctx, &russellpb.StartServiceRequest{
				Name: shortName,
			})
			if err == nil && resp.GetSuccess() {
				return nil // Status will be updated by RefreshStatus
			}
			// Fall through to manual start
		}
	}

	return sm.startServiceInternal(index)
}

// StopService stops a single service
func (sm *ServiceManager) StopService(index int) error {
	if index < 0 || index >= len(sm.Services) {
		return fmt.Errorf("invalid service index")
	}

	sm.mu.RLock()
	shortName := sm.Services[index].ShortName
	sm.mu.RUnlock()

	// Try to use Russell's StopService API if available
	if sm.IsRussellRunning() && shortName != "russell" {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		conn, err := grpc.DialContext(ctx, "localhost:9100",
			grpc.WithTransportCredentials(insecure.NewCredentials()),
			grpc.WithBlock(),
		)
		if err == nil {
			defer conn.Close()
			client := russellpb.NewRussellServiceClient(conn)

			sm.mu.Lock()
			sm.Services[index].Status = ServiceStopping
			sm.mu.Unlock()

			resp, err := client.StopService(ctx, &russellpb.StopServiceRequest{
				Name: shortName,
			})
			if err == nil && resp.GetSuccess() {
				sm.mu.Lock()
				sm.Services[index].Status = ServiceStopped
				sm.mu.Unlock()
				return nil
			}
			// Fall through to manual stop
		}
	}

	// Fallback: manual stop
	sm.mu.Lock()
	sm.Services[index].Status = ServiceStopping
	sm.mu.Unlock()

	// Try to kill the specific service process
	exec.Command("pkill", "-f", fmt.Sprintf("%s serve %s", findMDWBinary(), shortName)).Run()

	sm.mu.Lock()
	sm.Services[index].Status = ServiceStopped
	sm.mu.Unlock()

	return nil
}

// IsAllRunning returns true if all services are running
func (sm *ServiceManager) IsAllRunning() bool {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	for _, svc := range sm.Services {
		if svc.Status != ServiceRunning {
			return false
		}
	}
	return true
}

// IsAnyRunning returns true if any service is running
func (sm *ServiceManager) IsAnyRunning() bool {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	for _, svc := range sm.Services {
		if svc.Status == ServiceRunning || svc.Status == ServiceStarting {
			return true
		}
	}
	return false
}

// GetRunningCount returns the number of running services
func (sm *ServiceManager) GetRunningCount() int {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	count := 0
	for _, svc := range sm.Services {
		if svc.Status == ServiceRunning {
			count++
		}
	}
	return count
}

// GetServicePort returns the main port for a service
func (svc *Service) GetPort() int {
	if svc.GRPCPort != 0 {
		return svc.GRPCPort
	}
	return svc.HTTPPort
}

// GetStatusIcon returns the appropriate icon for the service status
func (svc *Service) GetStatusIcon() string {
	switch svc.Status {
	case ServiceRunning:
		return IconRunning
	case ServiceStopped:
		return IconStopped
	case ServiceStarting:
		return IconSpinner
	case ServiceStopping:
		return IconSpinner
	case ServiceError:
		return IconError
	default:
		return IconBullet
	}
}

// findMDWBinary finds the mdw binary path
func findMDWBinary() string {
	// First, get the working directory
	wd, _ := os.Getwd()

	// Try common locations with absolute paths
	paths := []string{
		filepath.Join(wd, "bin", "mdw"),
		filepath.Join(wd, "mdw"),
		"./bin/mdw",
		"bin/mdw",
	}

	for _, path := range paths {
		if _, err := os.Stat(path); err == nil {
			// Return absolute path if possible
			if absPath, err := filepath.Abs(path); err == nil {
				return absPath
			}
			return path
		}
	}

	// Check if it's in PATH
	if path, err := exec.LookPath("mdw"); err == nil {
		return path
	}

	return "./bin/mdw"
}
