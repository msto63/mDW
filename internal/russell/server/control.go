package server

import (
	"context"
	"strings"
	"time"

	commonpb "github.com/msto63/mDW/api/gen/common"
	pb "github.com/msto63/mDW/api/gen/russell"
	"github.com/msto63/mDW/internal/russell/orchestrator"
	"github.com/msto63/mDW/internal/russell/procmgr"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// StartService starts a service
func (s *Server) StartService(ctx context.Context, req *pb.StartServiceRequest) (*pb.StartServiceResponse, error) {
	if req.Name == "" {
		return nil, status.Error(codes.InvalidArgument, "service name is required")
	}

	s.logger.Info("Starting service", "service", req.Name)

	// Use orchestrator if available
	if s.orchestrator != nil {
		if err := s.orchestrator.StartService(ctx, req.Name); err != nil {
			s.logger.Error("Failed to start service", "service", req.Name, "error", err)
			return &pb.StartServiceResponse{
				Success: false,
				Message: err.Error(),
			}, nil
		}

		svc, _ := s.orchestrator.GetServiceStatus(req.Name)
		pid := int32(0)
		if svc != nil {
			info := svc.GetInfo()
			pid = int32(info.PID)
		}

		return &pb.StartServiceResponse{
			Success:   true,
			Message:   "Service started",
			ServiceId: req.Name,
			Pid:       pid,
		}, nil
	}

	// Fallback to procMgr
	if err := s.procMgr.StartService(ctx, req.Name); err != nil {
		s.logger.Error("Failed to start service", "service", req.Name, "error", err)
		return &pb.StartServiceResponse{
			Success: false,
			Message: err.Error(),
		}, nil
	}

	svc, _ := s.procMgr.GetServiceStatus(req.Name)
	pid := int32(0)
	if svc != nil {
		_, pidVal, _, _, _ := svc.GetStatus()
		pid = int32(pidVal)
	}

	return &pb.StartServiceResponse{
		Success:   true,
		Message:   "Service started",
		ServiceId: req.Name,
		Pid:       pid,
	}, nil
}

// StopService stops a service
func (s *Server) StopService(ctx context.Context, req *pb.StopServiceRequest) (*pb.StopServiceResponse, error) {
	if req.Name == "" {
		return nil, status.Error(codes.InvalidArgument, "service name is required")
	}

	s.logger.Info("Stopping service", "service", req.Name, "force", req.Force)

	// Use orchestrator if available
	if s.orchestrator != nil {
		if err := s.orchestrator.StopService(ctx, req.Name); err != nil {
			s.logger.Error("Failed to stop service", "service", req.Name, "error", err)
			return &pb.StopServiceResponse{
				Success: false,
				Message: err.Error(),
			}, nil
		}

		return &pb.StopServiceResponse{
			Success: true,
			Message: "Service stopped",
		}, nil
	}

	// Fallback to procMgr
	if err := s.procMgr.StopService(ctx, req.Name, req.Force); err != nil {
		s.logger.Error("Failed to stop service", "service", req.Name, "error", err)
		return &pb.StopServiceResponse{
			Success: false,
			Message: err.Error(),
		}, nil
	}

	return &pb.StopServiceResponse{
		Success: true,
		Message: "Service stopped",
	}, nil
}

// RestartService restarts a service
func (s *Server) RestartService(ctx context.Context, req *pb.RestartServiceRequest) (*pb.RestartServiceResponse, error) {
	if req.Name == "" {
		return nil, status.Error(codes.InvalidArgument, "service name is required")
	}

	s.logger.Info("Restarting service", "service", req.Name)

	// Use orchestrator if available
	if s.orchestrator != nil {
		if err := s.orchestrator.RestartService(ctx, req.Name); err != nil {
			s.logger.Error("Failed to restart service", "service", req.Name, "error", err)
			return &pb.RestartServiceResponse{
				Success: false,
				Message: err.Error(),
			}, nil
		}

		svc, _ := s.orchestrator.GetServiceStatus(req.Name)
		pid := int32(0)
		if svc != nil {
			info := svc.GetInfo()
			pid = int32(info.PID)
		}

		return &pb.RestartServiceResponse{
			Success:   true,
			Message:   "Service restarted",
			ServiceId: req.Name,
			Pid:       pid,
		}, nil
	}

	// Fallback to procMgr
	if err := s.procMgr.RestartService(ctx, req.Name); err != nil {
		s.logger.Error("Failed to restart service", "service", req.Name, "error", err)
		return &pb.RestartServiceResponse{
			Success: false,
			Message: err.Error(),
		}, nil
	}

	svc, _ := s.procMgr.GetServiceStatus(req.Name)
	pid := int32(0)
	if svc != nil {
		_, pidVal, _, _, _ := svc.GetStatus()
		pid = int32(pidVal)
	}

	return &pb.RestartServiceResponse{
		Success:   true,
		Message:   "Service restarted",
		ServiceId: req.Name,
		Pid:       pid,
	}, nil
}

// GetServiceStatus returns the status of a service
func (s *Server) GetServiceStatus(ctx context.Context, req *pb.GetServiceStatusRequest) (*pb.ServiceStatusResponse, error) {
	if req.Name == "" {
		return nil, status.Error(codes.InvalidArgument, "service name is required")
	}

	svc, err := s.procMgr.GetServiceStatus(req.Name)
	if err != nil {
		return nil, status.Error(codes.NotFound, err.Error())
	}

	return s.buildServiceStatusResponse(svc), nil
}

// GetAllServiceStatus returns status of all services
func (s *Server) GetAllServiceStatus(ctx context.Context, _ *commonpb.Empty) (*pb.AllServiceStatusResponse, error) {
	services := s.procMgr.GetAllServiceStatus()

	resp := &pb.AllServiceStatusResponse{
		Services: make([]*pb.ServiceStatusResponse, 0, len(services)),
	}

	var running, stopped, unhealthy int32

	for _, svc := range services {
		resp.Services = append(resp.Services, s.buildServiceStatusResponse(svc))

		svcStatus, _, _, _, _ := svc.GetStatus()
		switch svcStatus {
		case procmgr.StatusRunning:
			running++
		case procmgr.StatusStopped:
			stopped++
		case procmgr.StatusFailed:
			unhealthy++
		}
	}

	resp.Total = int32(len(services))
	resp.Running = running
	resp.Stopped = stopped
	resp.Unhealthy = unhealthy

	return resp, nil
}

// StreamServiceStatus streams service status events
func (s *Server) StreamServiceStatus(_ *commonpb.Empty, stream pb.RussellService_StreamServiceStatusServer) error {
	ch := s.procMgr.Subscribe()
	defer s.procMgr.Unsubscribe(ch)

	// Send initial status for all services
	services := s.procMgr.GetAllServiceStatus()
	for _, svc := range services {
		svcStatus, _, _, _, _ := svc.GetStatus()
		cfg := svc.GetConfig()
		event := &pb.ServiceStatusEvent{
			Name:           cfg.Name,
			CurrentStatus:  convertStatus(svcStatus),
			PreviousStatus: pb.ServiceStatus_SERVICE_STATUS_UNKNOWN,
			Message:        "Initial status",
			Timestamp:      time.Now().Unix(),
		}

		if err := stream.Send(event); err != nil {
			return err
		}
	}

	// Stream events
	for event := range ch {
		pbEvent := &pb.ServiceStatusEvent{
			Name:           event.ServiceName,
			PreviousStatus: convertStatus(event.PreviousStatus),
			CurrentStatus:  convertStatus(event.CurrentStatus),
			Message:        event.Message,
			Timestamp:      event.Timestamp.Unix(),
		}

		if err := stream.Send(pbEvent); err != nil {
			return err
		}
	}

	return nil
}

// buildServiceStatusResponse builds a ServiceStatusResponse from a ManagedService
func (s *Server) buildServiceStatusResponse(svc *procmgr.ManagedService) *pb.ServiceStatusResponse {
	svcStatus, pid, startedAt, restartCount, lastError := svc.GetStatus()
	cfg := svc.GetConfig()

	var startedAtUnix int64
	if !startedAt.IsZero() {
		startedAtUnix = startedAt.Unix()
	}

	return &pb.ServiceStatusResponse{
		Name:          cfg.Name,
		Status:        convertStatus(svcStatus),
		Pid:           int32(pid),
		Port:          int32(cfg.Port),
		Address:       "localhost",
		UptimeSeconds: svc.Uptime(),
		StartedAt:     startedAtUnix,
		RestartCount:  int32(restartCount),
		HealthMessage: lastError,
		Version:       cfg.Version,
	}
}

// convertStatus converts procmgr.ServiceStatus to pb.ServiceStatus
func convertStatus(s procmgr.ServiceStatus) pb.ServiceStatus {
	switch s {
	case procmgr.StatusStopped:
		return pb.ServiceStatus_SERVICE_STATUS_STOPPED
	case procmgr.StatusStarting:
		return pb.ServiceStatus_SERVICE_STATUS_STARTING
	case procmgr.StatusRunning:
		return pb.ServiceStatus_SERVICE_STATUS_HEALTHY
	case procmgr.StatusStopping:
		return pb.ServiceStatus_SERVICE_STATUS_STOPPING
	case procmgr.StatusFailed:
		return pb.ServiceStatus_SERVICE_STATUS_FAILED
	default:
		return pb.ServiceStatus_SERVICE_STATUS_UNKNOWN
	}
}

// convertOrchestratorStatus converts orchestrator.ServiceStatus to pb.ServiceStatus
func convertOrchestratorStatus(s orchestrator.ServiceStatus) pb.ServiceStatus {
	switch s {
	case orchestrator.StatusStopped:
		return pb.ServiceStatus_SERVICE_STATUS_STOPPED
	case orchestrator.StatusStarting:
		return pb.ServiceStatus_SERVICE_STATUS_STARTING
	case orchestrator.StatusRunning:
		return pb.ServiceStatus_SERVICE_STATUS_HEALTHY
	case orchestrator.StatusStopping:
		return pb.ServiceStatus_SERVICE_STATUS_STOPPING
	case orchestrator.StatusFailed:
		return pb.ServiceStatus_SERVICE_STATUS_FAILED
	default:
		return pb.ServiceStatus_SERVICE_STATUS_UNKNOWN
	}
}

// StartAllServices starts all services in dependency order
func (s *Server) StartAllServices(ctx context.Context, req *pb.StartAllRequest) (*pb.StartAllResponse, error) {
	s.logger.Info("Starting all services", "force", req.Force)

	// Use orchestrator if available
	if s.orchestrator != nil {
		if err := s.orchestrator.StartAll(ctx); err != nil {
			s.logger.Error("Failed to start all services", "error", err)
			return &pb.StartAllResponse{
				Success: false,
				Message: err.Error(),
			}, nil
		}

		// Build results from orchestrator status
		services := s.orchestrator.GetAllServiceStatus()
		results := make([]*pb.ServiceStartResult, 0, len(services))
		for name, svc := range services {
			info := svc.GetInfo()
			result := &pb.ServiceStartResult{
				Name:     name,
				Success:  info.Status == orchestrator.StatusRunning,
				Attempts: int32(info.RestartCount + 1),
			}
			if info.LastError != "" {
				result.Error = info.LastError
			}
			results = append(results, result)
		}

		return &pb.StartAllResponse{
			Success: true,
			Message: "All services started successfully",
			Results: results,
		}, nil
	}

	// Fallback to procMgr
	if err := s.procMgr.StartAll(ctx); err != nil {
		return &pb.StartAllResponse{
			Success: false,
			Message: err.Error(),
		}, nil
	}

	return &pb.StartAllResponse{
		Success: true,
		Message: "All services started",
	}, nil
}

// StopAllServices stops all services in reverse dependency order
func (s *Server) StopAllServices(ctx context.Context, req *pb.StopAllRequest) (*pb.StopAllResponse, error) {
	s.logger.Info("Stopping all services", "force", req.Force)

	// Use orchestrator if available
	if s.orchestrator != nil {
		if err := s.orchestrator.StopAll(ctx); err != nil {
			s.logger.Error("Failed to stop all services", "error", err)
			return &pb.StopAllResponse{
				Success: false,
				Message: err.Error(),
			}, nil
		}

		return &pb.StopAllResponse{
			Success: true,
			Message: "All services stopped successfully",
		}, nil
	}

	// Fallback to procMgr
	if err := s.procMgr.StopAll(ctx); err != nil {
		return &pb.StopAllResponse{
			Success: false,
			Message: err.Error(),
		}, nil
	}

	return &pb.StopAllResponse{
		Success: true,
		Message: "All services stopped",
	}, nil
}

// GetOrchestratorStatus returns the orchestrator status
func (s *Server) GetOrchestratorStatus(ctx context.Context, _ *commonpb.Empty) (*pb.OrchestratorStatusResponse, error) {
	// Use orchestrator if available
	if s.orchestrator != nil {
		status := s.orchestrator.GetStatus()
		services := s.orchestrator.GetAllServiceStatus()
		startedAt := s.orchestrator.GetStartedAt()

		resp := &pb.OrchestratorStatusResponse{
			State:         convertOrchestratorState(status),
			TotalServices: int32(len(services)),
			StartedAt:     startedAt.Unix(),
			Services:      make([]*pb.OrchestratorServiceStatus, 0, len(services)),
		}

		if !startedAt.IsZero() {
			resp.UptimeSeconds = int64(time.Since(startedAt).Seconds())
		}

		var running, healthy, failed int32
		for _, svc := range services {
			info := svc.GetInfo()
			svcStatus := &pb.OrchestratorServiceStatus{
				Name:         info.Name,
				ShortName:    info.ShortName,
				Status:       convertOrchestratorStatus(info.Status),
				Healthy:      info.Healthy,
				Pid:          int32(info.PID),
				Port:         int32(info.Port),
				RestartCount: int32(info.RestartCount),
				LastError:    info.LastError,
				Dependencies: info.Dependencies,
				Version:      info.Version,
			}
			if !info.StartedAt.IsZero() {
				svcStatus.UptimeSeconds = int64(time.Since(info.StartedAt).Seconds())
			}

			resp.Services = append(resp.Services, svcStatus)

			if info.Status == orchestrator.StatusRunning {
				running++
				if info.Healthy {
					healthy++
				}
			} else if info.Status == orchestrator.StatusFailed {
				failed++
			}
		}

		resp.RunningServices = running
		resp.HealthyServices = healthy
		resp.FailedServices = failed

		return resp, nil
	}

	// Fallback: Return status based on procMgr
	services := s.procMgr.GetAllServiceStatus()
	resp := &pb.OrchestratorStatusResponse{
		State:         pb.OrchestratorState_ORCHESTRATOR_STATE_RUNNING,
		TotalServices: int32(len(services)),
		StartedAt:     s.startTime.Unix(),
		UptimeSeconds: int64(time.Since(s.startTime).Seconds()),
		Services:      make([]*pb.OrchestratorServiceStatus, 0, len(services)),
	}

	var running, healthy, failed int32
	for _, svc := range services {
		svcStatus, pid, startedAt, restartCount, lastError := svc.GetStatus()
		cfg := svc.GetConfig()

		orchSvc := &pb.OrchestratorServiceStatus{
			Name:         cfg.Name,
			ShortName:    strings.ToLower(cfg.Name), // Use lowercase name as short_name for matching
			Status:       convertStatus(svcStatus),
			Healthy:      svcStatus == procmgr.StatusRunning,
			Pid:          int32(pid),
			Port:         int32(cfg.Port),
			RestartCount: int32(restartCount),
			LastError:    lastError,
			Version:      cfg.Version,
		}
		if !startedAt.IsZero() {
			orchSvc.UptimeSeconds = int64(time.Since(startedAt).Seconds())
		}

		resp.Services = append(resp.Services, orchSvc)

		if svcStatus == procmgr.StatusRunning {
			running++
			healthy++
		} else if svcStatus == procmgr.StatusFailed {
			failed++
		}
	}

	resp.RunningServices = running
	resp.HealthyServices = healthy
	resp.FailedServices = failed

	return resp, nil
}

// convertOrchestratorState converts orchestrator.OrchestratorStatus to pb.OrchestratorState
func convertOrchestratorState(s orchestrator.OrchestratorStatus) pb.OrchestratorState {
	switch s {
	case orchestrator.OrchestratorStarting:
		return pb.OrchestratorState_ORCHESTRATOR_STATE_STARTING
	case orchestrator.OrchestratorRunning:
		return pb.OrchestratorState_ORCHESTRATOR_STATE_RUNNING
	case orchestrator.OrchestratorStopping:
		return pb.OrchestratorState_ORCHESTRATOR_STATE_STOPPING
	case orchestrator.OrchestratorStopped:
		return pb.OrchestratorState_ORCHESTRATOR_STATE_STOPPED
	default:
		return pb.OrchestratorState_ORCHESTRATOR_STATE_UNKNOWN
	}
}
