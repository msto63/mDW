package server

import (
	"context"
	"time"

	"github.com/msto63/mDW/api/gen/common"
	pb "github.com/msto63/mDW/api/gen/russell"
	"github.com/msto63/mDW/internal/russell/service"
	"github.com/msto63/mDW/pkg/core/discovery"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Ensure Server implements RussellServiceServer
var _ pb.RussellServiceServer = (*Server)(nil)

// Register implements RussellServiceServer.Register
func (s *Server) Register(ctx context.Context, req *pb.RegisterRequest) (*pb.RegisterResponse, error) {
	if req.Name == "" {
		return nil, status.Error(codes.InvalidArgument, "name is required")
	}
	if req.Address == "" {
		return nil, status.Error(codes.InvalidArgument, "address is required")
	}

	info := &discovery.ServiceInfo{
		Name:     req.Name,
		Version:  req.Version,
		Address:  req.Address,
		Port:     int(req.Port),
		Metadata: req.Metadata,
		Tags:     req.Tags,
	}

	if err := s.registry.Register(ctx, info); err != nil {
		s.logger.Error("Failed to register service", "name", req.Name, "error", err)
		return nil, status.Error(codes.Internal, err.Error())
	}

	s.logger.Info("Service registered", "id", info.ID, "name", req.Name, "address", req.Address, "port", req.Port)

	return &pb.RegisterResponse{
		Success: true,
		Id:      info.ID, // Return the actual generated ID from the registry
		Message: "Service registered successfully",
	}, nil
}

// Deregister implements RussellServiceServer.Deregister
func (s *Server) Deregister(ctx context.Context, req *pb.DeregisterRequest) (*common.Empty, error) {
	if req.Id == "" {
		return nil, status.Error(codes.InvalidArgument, "id is required")
	}

	if err := s.registry.Deregister(ctx, req.Id); err != nil {
		s.logger.Error("Failed to deregister service", "id", req.Id, "error", err)
		return nil, status.Error(codes.Internal, err.Error())
	}

	s.logger.Info("Service deregistered", "id", req.Id)

	return &common.Empty{}, nil
}

// Heartbeat implements RussellServiceServer.Heartbeat
func (s *Server) Heartbeat(ctx context.Context, req *pb.HeartbeatRequest) (*pb.HeartbeatResponse, error) {
	if req.Id == "" {
		return nil, status.Error(codes.InvalidArgument, "id is required")
	}

	// Update service status in registry
	if err := s.registry.Heartbeat(ctx, req.Id); err != nil {
		s.logger.Warn("Heartbeat failed", "id", req.Id, "error", err)
		return &pb.HeartbeatResponse{
			Acknowledged:    false,
			NextHeartbeatMs: 30000,
		}, nil
	}

	return &pb.HeartbeatResponse{
		Acknowledged:    true,
		NextHeartbeatMs: 30000, // 30 seconds until next heartbeat
	}, nil
}

// Discover implements RussellServiceServer.Discover
func (s *Server) Discover(ctx context.Context, req *pb.DiscoverRequest) (*pb.DiscoverResponse, error) {
	services, err := s.registry.Discover(ctx, req.Name)
	if err != nil {
		s.logger.Error("Discovery failed", "name", req.Name, "error", err)
		return nil, status.Error(codes.Internal, err.Error())
	}

	pbServices := make([]*pb.ServiceInfo, len(services))
	for i, svc := range services {
		pbServices[i] = convertToProtoServiceInfo(svc)
	}

	return &pb.DiscoverResponse{
		Services: pbServices,
	}, nil
}

// GetService implements RussellServiceServer.GetService
func (s *Server) GetService(ctx context.Context, req *pb.GetServiceRequest) (*pb.ServiceInfo, error) {
	if req.Id == "" {
		return nil, status.Error(codes.InvalidArgument, "id is required")
	}

	// Use Discover to find the service by name
	services, err := s.registry.Discover(ctx, req.Id)
	if err != nil || len(services) == 0 {
		return nil, status.Error(codes.NotFound, "service not found")
	}

	return convertToProtoServiceInfo(services[0]), nil
}

// ListServices implements RussellServiceServer.ListServices
func (s *Server) ListServices(ctx context.Context, _ *common.Empty) (*pb.ServiceListResponse, error) {
	services, err := s.registry.List(ctx)
	if err != nil {
		s.logger.Error("ListServices failed", "error", err)
		return nil, status.Error(codes.Internal, err.Error())
	}

	pbServices := make([]*pb.ServiceInfo, len(services))
	for i, svc := range services {
		pbServices[i] = convertToProtoServiceInfo(svc)
	}

	return &pb.ServiceListResponse{
		Services: pbServices,
		Total:    int32(len(services)),
	}, nil
}

// GetSystemHealth implements RussellServiceServer.GetSystemHealth
func (s *Server) GetSystemHealth(ctx context.Context, _ *common.Empty) (*pb.SystemHealthResponse, error) {
	serviceHealth := s.service.HealthCheck(ctx)

	var serviceHealthList []*pb.ServiceHealth
	allHealthy := true

	for svc, healthy := range serviceHealth {
		healthStatus := "healthy"
		if !healthy {
			healthStatus = "unhealthy"
			allHealthy = false
		}
		serviceHealthList = append(serviceHealthList, &pb.ServiceHealth{
			Name:   string(svc),
			Status: healthStatus,
		})
	}

	overallStatus := "healthy"
	if !allHealthy {
		overallStatus = "degraded"
	}

	return &pb.SystemHealthResponse{
		OverallStatus: overallStatus,
		Services:      serviceHealthList,
		Timestamp:     time.Now().Unix(),
	}, nil
}

// HealthCheck implements RussellServiceServer.HealthCheck
func (s *Server) HealthCheck(ctx context.Context, _ *common.HealthCheckRequest) (*common.HealthCheckResponse, error) {
	result := s.health.Check(ctx)

	details := make(map[string]string)
	for _, check := range result.Checks {
		details[check.Name] = string(check.Status)
	}

	return &common.HealthCheckResponse{
		Status:        string(result.Status),
		Service:       "russell",
		Version:       "1.0.0",
		UptimeSeconds: int64(time.Since(s.startTime).Seconds()),
		Details:       details,
	}, nil
}

// ============================================================================
// Admin & Orchestration Methods
// ============================================================================

// GetSystemOverview implements RussellServiceServer.GetSystemOverview
func (s *Server) GetSystemOverview(ctx context.Context, _ *common.Empty) (*pb.SystemOverviewResponse, error) {
	overview, err := s.service.GetSystemOverview(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	services := make(map[string]*pb.AdminServiceStatus)
	for name, svc := range overview.Services {
		services[name] = &pb.AdminServiceStatus{
			Name:    svc.Name,
			Type:    svc.Type,
			Status:  string(svc.Status),
			Address: svc.Address,
			Version: svc.Version,
		}
	}

	var metrics *pb.MetricsResponse
	if overview.SystemMetrics != nil {
		metrics = &pb.MetricsResponse{
			TotalRequests:         overview.SystemMetrics.TotalRequests,
			SuccessfulRequests:    overview.SystemMetrics.SuccessfulRequests,
			FailedRequests:        overview.SystemMetrics.FailedRequests,
			AverageResponseTimeMs: overview.SystemMetrics.AverageResponseTime.Milliseconds(),
			RequestsPerSecond:     overview.SystemMetrics.RequestsPerSecond,
		}
	}

	errors := make([]*pb.ErrorEntry, len(overview.RecentErrors))
	for i, e := range overview.RecentErrors {
		errors[i] = &pb.ErrorEntry{
			Timestamp: e.Timestamp.Format(time.RFC3339),
			Service:   e.Service,
			Operation: e.Operation,
			ErrorCode: e.ErrorCode,
			Message:   e.Message,
			RequestId: e.RequestID,
		}
	}

	return &pb.SystemOverviewResponse{
		Timestamp:         overview.Timestamp.Format(time.RFC3339),
		TotalServices:     int32(overview.TotalServices),
		HealthyServices:   int32(overview.HealthyServices),
		DegradedServices:  int32(overview.DegradedServices),
		UnhealthyServices: int32(overview.UnhealthyServices),
		Services:          services,
		Metrics:           metrics,
		RecentErrors:      errors,
	}, nil
}

// GetMetrics implements RussellServiceServer.GetMetrics
func (s *Server) GetMetrics(ctx context.Context, _ *common.Empty) (*pb.MetricsResponse, error) {
	metrics := s.service.GetSystemMetrics()

	return &pb.MetricsResponse{
		TotalRequests:         metrics.TotalRequests,
		SuccessfulRequests:    metrics.SuccessfulRequests,
		FailedRequests:        metrics.FailedRequests,
		AverageResponseTimeMs: metrics.AverageResponseTime.Milliseconds(),
		RequestsPerSecond:     metrics.RequestsPerSecond,
	}, nil
}

// GetErrors implements RussellServiceServer.GetErrors
func (s *Server) GetErrors(ctx context.Context, req *pb.GetErrorsRequest) (*pb.ErrorsResponse, error) {
	limit := int(req.Limit)
	if limit <= 0 {
		limit = 50
	}

	errorEntries := s.service.GetRecentErrors(limit)
	errors := make([]*pb.ErrorEntry, len(errorEntries))
	for i, e := range errorEntries {
		errors[i] = &pb.ErrorEntry{
			Timestamp: e.Timestamp.Format(time.RFC3339),
			Service:   e.Service,
			Operation: e.Operation,
			ErrorCode: e.ErrorCode,
			Message:   e.Message,
			RequestId: e.RequestID,
		}
	}

	return &pb.ErrorsResponse{
		Errors: errors,
	}, nil
}

// ============================================================================
// Pipeline Management Methods
// ============================================================================

// CreatePipeline implements RussellServiceServer.CreatePipeline
func (s *Server) CreatePipeline(ctx context.Context, req *pb.CreatePipelineRequest) (*pb.Pipeline, error) {
	if req.Name == "" {
		return nil, status.Error(codes.InvalidArgument, "pipeline name is required")
	}

	// Convert proto steps to service steps
	steps := make([]service.PipelineStep, len(req.Steps))
	for i, step := range req.Steps {
		steps[i] = service.PipelineStep{
			ID:          step.Id,
			ServiceType: service.ServiceType(step.ServiceType),
			Operation:   step.Operation,
			DependsOn:   step.DependsOn,
		}
	}

	pipeline := &service.Pipeline{
		ID:          req.Id,
		Name:        req.Name,
		Description: req.Description,
		Steps:       steps,
	}

	if err := s.service.RegisterPipeline(pipeline); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return convertPipelineToProto(pipeline), nil
}

// GetPipeline implements RussellServiceServer.GetPipeline
func (s *Server) GetPipeline(ctx context.Context, req *pb.GetPipelineRequest) (*pb.Pipeline, error) {
	if req.Id == "" {
		return nil, status.Error(codes.InvalidArgument, "pipeline id is required")
	}

	pipeline, err := s.service.GetPipeline(req.Id)
	if err != nil {
		return nil, status.Error(codes.NotFound, err.Error())
	}

	return convertPipelineToProto(pipeline), nil
}

// ListPipelines implements RussellServiceServer.ListPipelines
func (s *Server) ListPipelines(ctx context.Context, _ *common.Empty) (*pb.PipelineListResponse, error) {
	pipelines := s.service.ListPipelines()

	pbPipelines := make([]*pb.Pipeline, len(pipelines))
	for i, p := range pipelines {
		pbPipelines[i] = convertPipelineToProto(p)
	}

	return &pb.PipelineListResponse{
		Pipelines: pbPipelines,
	}, nil
}

// DeletePipeline implements RussellServiceServer.DeletePipeline
func (s *Server) DeletePipeline(ctx context.Context, req *pb.DeletePipelineRequest) (*common.Empty, error) {
	if req.Id == "" {
		return nil, status.Error(codes.InvalidArgument, "pipeline id is required")
	}

	if err := s.service.DeletePipeline(req.Id); err != nil {
		return nil, status.Error(codes.NotFound, err.Error())
	}

	return &common.Empty{}, nil
}

// ExecutePipeline implements RussellServiceServer.ExecutePipeline
func (s *Server) ExecutePipeline(ctx context.Context, req *pb.ExecutePipelineRequest) (*pb.PipelineExecutionResponse, error) {
	if req.PipelineId == "" {
		return nil, status.Error(codes.InvalidArgument, "pipeline id is required")
	}

	execution, err := s.service.ExecutePipeline(ctx, req.PipelineId, req.Input)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &pb.PipelineExecutionResponse{
		ExecutionId: execution.ID,
		PipelineId:  execution.PipelineID,
		Status:      string(execution.Status),
		StartedAt:   execution.StartedAt.Format(time.RFC3339),
		CompletedAt: execution.CompletedAt.Format(time.RFC3339),
		Error:       execution.Error,
	}, nil
}

// Helper function to convert service.Pipeline to proto
func convertPipelineToProto(p *service.Pipeline) *pb.Pipeline {
	steps := make([]*pb.PipelineStep, len(p.Steps))
	for i, step := range p.Steps {
		steps[i] = &pb.PipelineStep{
			Id:          step.ID,
			ServiceType: string(step.ServiceType),
			Operation:   step.Operation,
			DependsOn:   step.DependsOn,
		}
	}

	return &pb.Pipeline{
		Id:          p.ID,
		Name:        p.Name,
		Description: p.Description,
		Steps:       steps,
		CreatedAt:   p.CreatedAt.Format(time.RFC3339),
	}
}

// convertToProtoServiceInfo converts discovery.ServiceInfo to proto ServiceInfo
func convertToProtoServiceInfo(svc *discovery.ServiceInfo) *pb.ServiceInfo {
	pbStatus := pb.ServiceStatus_SERVICE_STATUS_UNKNOWN
	switch svc.Status {
	case discovery.ServiceStatusHealthy:
		pbStatus = pb.ServiceStatus_SERVICE_STATUS_HEALTHY
	case discovery.ServiceStatusUnhealthy:
		pbStatus = pb.ServiceStatus_SERVICE_STATUS_UNHEALTHY
	case discovery.ServiceStatusStarting:
		pbStatus = pb.ServiceStatus_SERVICE_STATUS_STARTING
	case discovery.ServiceStatusStopping:
		pbStatus = pb.ServiceStatus_SERVICE_STATUS_STOPPING
	}

	return &pb.ServiceInfo{
		Id:       svc.ID,
		Name:     svc.Name,
		Version:  svc.Version,
		Address:  svc.Address,
		Port:     int32(svc.Port),
		Status:   pbStatus,
		Metadata: svc.Metadata,
		Tags:     svc.Tags,
	}
}
