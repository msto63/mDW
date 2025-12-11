package server

import (
	"context"
	"time"

	"github.com/msto63/mDW/api/gen/common"
	pb "github.com/msto63/mDW/api/gen/leibniz"
	"github.com/msto63/mDW/internal/leibniz/service"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Ensure Server implements LeibnizServiceServer
var _ pb.LeibnizServiceServer = (*Server)(nil)

// CreateAgent implements LeibnizServiceServer.CreateAgent
func (s *Server) CreateAgent(ctx context.Context, req *pb.CreateAgentRequest) (*pb.AgentInfo, error) {
	if req.Name == "" {
		return nil, status.Error(codes.InvalidArgument, "name is required")
	}

	// Build agent definition
	def := &service.AgentDefinition{
		Name:         req.Name,
		Description:  req.Description,
		SystemPrompt: req.SystemPrompt,
		Tools:        req.Tools,
	}

	if req.Config != nil {
		def.Model = req.Config.Model
		def.MaxSteps = int(req.Config.MaxIterations)
		if req.Config.TimeoutSeconds > 0 {
			def.Timeout = time.Duration(req.Config.TimeoutSeconds) * time.Second
		}
	}

	created, err := s.service.CreateAgent(def)
	if err != nil {
		s.logger.Error("CreateAgent failed", "error", err)
		return nil, status.Error(codes.Internal, err.Error())
	}

	// Save as YAML file if requested (enables hot-reload)
	if req.SaveAsYaml {
		if err := s.service.SaveAgentAsYAML(created); err != nil {
			s.logger.Warn("Failed to save agent as YAML", "id", created.ID, "error", err)
			// Don't fail the request, just log the warning
		} else {
			s.logger.Info("Agent saved as YAML for hot-reload", "id", created.ID)
		}
	}

	return agentToProto(created), nil
}

// UpdateAgent implements LeibnizServiceServer.UpdateAgent
func (s *Server) UpdateAgent(ctx context.Context, req *pb.UpdateAgentRequest) (*pb.AgentInfo, error) {
	if req.Id == "" {
		return nil, status.Error(codes.InvalidArgument, "id is required")
	}

	updates := &service.AgentDefinition{
		Name:         req.Name,
		Description:  req.Description,
		SystemPrompt: req.SystemPrompt,
		Tools:        req.Tools,
	}

	if req.Config != nil {
		updates.Model = req.Config.Model
		updates.MaxSteps = int(req.Config.MaxIterations)
		if req.Config.TimeoutSeconds > 0 {
			updates.Timeout = time.Duration(req.Config.TimeoutSeconds) * time.Second
		}
	}

	updated, err := s.service.UpdateAgent(req.Id, updates)
	if err != nil {
		s.logger.Error("UpdateAgent failed", "error", err)
		return nil, status.Error(codes.NotFound, err.Error())
	}

	// Save as YAML file if requested (enables hot-reload)
	if req.SaveAsYaml {
		if err := s.service.SaveAgentAsYAML(updated); err != nil {
			s.logger.Warn("Failed to save agent as YAML", "id", updated.ID, "error", err)
		} else {
			s.logger.Info("Agent saved as YAML for hot-reload", "id", updated.ID)
		}
	}

	return agentToProto(updated), nil
}

// DeleteAgent implements LeibnizServiceServer.DeleteAgent
func (s *Server) DeleteAgent(ctx context.Context, req *pb.DeleteAgentRequest) (*common.Empty, error) {
	if req.Id == "" {
		return nil, status.Error(codes.InvalidArgument, "id is required")
	}

	if err := s.service.DeleteAgent(req.Id); err != nil {
		s.logger.Error("DeleteAgent failed", "error", err)
		return nil, status.Error(codes.NotFound, err.Error())
	}

	return &common.Empty{}, nil
}

// GetAgent implements LeibnizServiceServer.GetAgent
func (s *Server) GetAgent(ctx context.Context, req *pb.GetAgentRequest) (*pb.AgentInfo, error) {
	if req.Id == "" {
		return nil, status.Error(codes.InvalidArgument, "id is required")
	}

	agent, err := s.service.GetAgent(req.Id)
	if err != nil {
		return nil, status.Error(codes.NotFound, err.Error())
	}

	return agentToProto(agent), nil
}

// ListAgents implements LeibnizServiceServer.ListAgents
func (s *Server) ListAgents(ctx context.Context, _ *common.Empty) (*pb.AgentListResponse, error) {
	agents := s.service.ListAgents()

	pbAgents := make([]*pb.AgentInfo, len(agents))
	for i, agent := range agents {
		pbAgents[i] = agentToProto(agent)
	}

	return &pb.AgentListResponse{
		Agents: pbAgents,
		Total:  int32(len(agents)),
	}, nil
}

// Execute implements LeibnizServiceServer.Execute
func (s *Server) Execute(ctx context.Context, req *pb.ExecuteRequest) (*pb.ExecuteResponse, error) {
	if req.Message == "" {
		return nil, status.Error(codes.InvalidArgument, "message is required")
	}

	agentID := req.AgentId
	if agentID == "" {
		agentID = "default"
	}

	resp, err := s.service.ExecuteWithAgent(ctx, agentID, req.Message)
	if err != nil && resp == nil {
		s.logger.Error("Execute failed", "error", err)
		return nil, status.Error(codes.Internal, err.Error())
	}

	return executeResponseToProto(resp), nil
}

// StreamExecute implements LeibnizServiceServer.StreamExecute
func (s *Server) StreamExecute(req *pb.ExecuteRequest, stream grpc.ServerStreamingServer[pb.AgentChunk]) error {
	if req.Message == "" {
		return status.Error(codes.InvalidArgument, "message is required")
	}

	ctx := stream.Context()

	// Send thinking chunk
	if err := stream.Send(&pb.AgentChunk{
		Type:    pb.ChunkType_CHUNK_TYPE_THINKING,
		Content: "Analysiere Aufgabe...",
	}); err != nil {
		return err
	}

	// Execute the task
	resp, err := s.Execute(ctx, req)
	if err != nil {
		return err
	}

	// Send response chunk
	if err := stream.Send(&pb.AgentChunk{
		Type:    pb.ChunkType_CHUNK_TYPE_RESPONSE,
		Content: resp.Response,
	}); err != nil {
		return err
	}

	// Send final chunk
	return stream.Send(&pb.AgentChunk{
		Type:    pb.ChunkType_CHUNK_TYPE_FINAL,
		Content: resp.Response,
	})
}

// ContinueExecution implements LeibnizServiceServer.ContinueExecution
func (s *Server) ContinueExecution(ctx context.Context, req *pb.ContinueRequest) (*pb.ExecuteResponse, error) {
	if req.ExecutionId == "" {
		return nil, status.Error(codes.InvalidArgument, "execution_id is required")
	}

	// Get execution record
	record, err := s.service.GetExecution(req.ExecutionId)
	if err != nil {
		return nil, status.Error(codes.NotFound, err.Error())
	}

	// Check if execution can be continued
	if record.Status != "awaiting_confirmation" {
		return nil, status.Error(codes.FailedPrecondition, "execution cannot be continued")
	}

	// For now, return the current state
	return &pb.ExecuteResponse{
		ExecutionId: record.ID,
		Status:      stringToExecutionStatus(record.Status),
		Response:    record.Result,
	}, nil
}

// CancelExecution implements LeibnizServiceServer.CancelExecution
func (s *Server) CancelExecution(ctx context.Context, req *pb.CancelRequest) (*common.Empty, error) {
	if req.ExecutionId == "" {
		return nil, status.Error(codes.InvalidArgument, "execution_id is required")
	}

	if err := s.service.CancelExecution(req.ExecutionId); err != nil {
		s.logger.Error("CancelExecution failed", "error", err)
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &common.Empty{}, nil
}

// GetExecution implements LeibnizServiceServer.GetExecution
func (s *Server) GetExecution(ctx context.Context, req *pb.GetExecutionRequest) (*pb.ExecutionInfo, error) {
	if req.ExecutionId == "" {
		return nil, status.Error(codes.InvalidArgument, "execution_id is required")
	}

	record, err := s.service.GetExecution(req.ExecutionId)
	if err != nil {
		return nil, status.Error(codes.NotFound, err.Error())
	}

	// Convert actions
	actions := make([]*pb.AgentAction, len(record.Steps))
	for i, step := range record.Steps {
		actions[i] = &pb.AgentAction{
			Tool:       step.ToolName,
			Input:      step.ToolInput,
			Output:     step.ToolOutput,
			DurationMs: 0,
		}
	}

	return &pb.ExecutionInfo{
		Id:            record.ID,
		AgentId:       record.AgentID,
		Status:        stringToExecutionStatus(record.Status),
		Actions:       actions,
		FinalResponse: record.Result,
		Iterations:    int32(len(record.Steps)),
		DurationMs:    record.Duration.Milliseconds(),
		StartedAt:     record.StartedAt.Unix(),
		CompletedAt:   record.CompletedAt.Unix(),
	}, nil
}

// ListTools implements LeibnizServiceServer.ListTools
func (s *Server) ListTools(ctx context.Context, _ *common.Empty) (*pb.ToolListResponse, error) {
	tools := s.service.ListTools()
	customTools := s.service.GetCustomTools()

	pbTools := make([]*pb.ToolInfo, 0, len(tools)+len(customTools))

	// Add agent tools
	for _, t := range tools {
		source := pb.ToolSource_TOOL_SOURCE_BUILTIN
		if t.Source != "builtin" {
			source = pb.ToolSource_TOOL_SOURCE_MCP
		}
		pbTools = append(pbTools, &pb.ToolInfo{
			Name:        t.Name,
			Description: t.Description,
			Enabled:     true,
			Source:      source,
		})
	}

	// Add custom tools
	for _, t := range customTools {
		pbTools = append(pbTools, &pb.ToolInfo{
			Name:                 t.Name,
			Description:          t.Description,
			ParameterSchema:      t.ParameterSchema,
			RequiresConfirmation: t.RequiresConfirmation,
			Enabled:              true,
			Source:               pb.ToolSource_TOOL_SOURCE_CUSTOM,
		})
	}

	return &pb.ToolListResponse{
		Tools: pbTools,
	}, nil
}

// RegisterTool implements LeibnizServiceServer.RegisterTool
func (s *Server) RegisterTool(ctx context.Context, req *pb.RegisterToolRequest) (*pb.ToolInfo, error) {
	if req.Name == "" {
		return nil, status.Error(codes.InvalidArgument, "name is required")
	}

	tool := &service.CustomTool{
		Name:                 req.Name,
		Description:          req.Description,
		ParameterSchema:      req.ParameterSchema,
		RequiresConfirmation: req.RequiresConfirmation,
	}

	if err := s.service.RegisterCustomTool(tool); err != nil {
		s.logger.Error("RegisterTool failed", "error", err)
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &pb.ToolInfo{
		Name:                 req.Name,
		Description:          req.Description,
		ParameterSchema:      req.ParameterSchema,
		RequiresConfirmation: req.RequiresConfirmation,
		Enabled:              true,
		Source:               pb.ToolSource_TOOL_SOURCE_CUSTOM,
	}, nil
}

// UnregisterTool implements LeibnizServiceServer.UnregisterTool
func (s *Server) UnregisterTool(ctx context.Context, req *pb.UnregisterToolRequest) (*common.Empty, error) {
	if req.Name == "" {
		return nil, status.Error(codes.InvalidArgument, "name is required")
	}

	if err := s.service.UnregisterCustomTool(req.Name); err != nil {
		s.logger.Error("UnregisterTool failed", "error", err)
		return nil, status.Error(codes.NotFound, err.Error())
	}

	return &common.Empty{}, nil
}

// HealthCheck implements LeibnizServiceServer.HealthCheck
func (s *Server) HealthCheck(ctx context.Context, _ *common.HealthCheckRequest) (*common.HealthCheckResponse, error) {
	result := s.health.Check(ctx)

	details := make(map[string]string)
	for _, check := range result.Checks {
		details[check.Name] = string(check.Status)
	}

	return &common.HealthCheckResponse{
		Status:        string(result.Status),
		Service:       "leibniz",
		Version:       "1.0.0",
		UptimeSeconds: int64(time.Since(s.startTime).Seconds()),
		Details:       details,
	}, nil
}

// FindBestAgent implements LeibnizServiceServer.FindBestAgent
// Uses RAG-style vector similarity to find the best matching agent for a task
func (s *Server) FindBestAgent(ctx context.Context, req *pb.FindAgentRequest) (*pb.AgentMatchResponse, error) {
	if req.TaskDescription == "" {
		return nil, status.Error(codes.InvalidArgument, "task_description is required")
	}

	match, err := s.service.FindBestAgentForTask(ctx, req.TaskDescription)
	if err != nil {
		s.logger.Warn("FindBestAgent failed", "error", err)
		// Return default agent as fallback
		return &pb.AgentMatchResponse{
			AgentId:    "default",
			AgentName:  "Default Agent",
			Similarity: 0,
		}, nil
	}

	return &pb.AgentMatchResponse{
		AgentId:    match.AgentID,
		AgentName:  match.AgentName,
		Similarity: match.Similarity,
	}, nil
}

// FindTopAgents implements LeibnizServiceServer.FindTopAgents
// Returns the top N matching agents for a task based on vector similarity
func (s *Server) FindTopAgents(ctx context.Context, req *pb.FindTopAgentsRequest) (*pb.AgentMatchListResponse, error) {
	if req.TaskDescription == "" {
		return nil, status.Error(codes.InvalidArgument, "task_description is required")
	}

	topN := int(req.TopN)
	if topN <= 0 {
		topN = 3 // Default to top 3
	}

	matches, err := s.service.FindTopAgentsForTask(ctx, req.TaskDescription, topN)
	if err != nil {
		s.logger.Warn("FindTopAgents failed", "error", err)
		return &pb.AgentMatchListResponse{
			Matches: []*pb.AgentMatchResponse{},
		}, nil
	}

	pbMatches := make([]*pb.AgentMatchResponse, len(matches))
	for i, m := range matches {
		pbMatches[i] = &pb.AgentMatchResponse{
			AgentId:    m.AgentID,
			AgentName:  m.AgentName,
			Similarity: m.Similarity,
		}
	}

	return &pb.AgentMatchListResponse{
		Matches: pbMatches,
	}, nil
}

// Helper functions

func agentToProto(agent *service.AgentDefinition) *pb.AgentInfo {
	return &pb.AgentInfo{
		Id:           agent.ID,
		Name:         agent.Name,
		Description:  agent.Description,
		SystemPrompt: agent.SystemPrompt,
		Tools:        agent.Tools,
		Config: &pb.AgentConfig{
			Model:         agent.Model,
			MaxIterations: int32(agent.MaxSteps),
		},
		CreatedAt: agent.CreatedAt.Unix(),
		UpdatedAt: agent.UpdatedAt.Unix(),
	}
}

func executeResponseToProto(resp *service.ExecuteResponse) *pb.ExecuteResponse {
	if resp == nil {
		return &pb.ExecuteResponse{
			Status: pb.ExecutionStatus_EXECUTION_STATUS_ERROR,
		}
	}

	actions := make([]*pb.AgentAction, len(resp.Steps))
	for i, step := range resp.Steps {
		actions[i] = &pb.AgentAction{
			Tool:   step.ToolName,
			Input:  step.ToolInput,
			Output: step.ToolOutput,
		}
	}

	return &pb.ExecuteResponse{
		ExecutionId: resp.ID,
		Status:      stringToExecutionStatus(resp.Status),
		Response:    resp.Result,
		Actions:     actions,
		Iterations:  int32(len(resp.Steps)),
		DurationMs:  resp.Duration.Milliseconds(),
	}
}

func stringToExecutionStatus(s string) pb.ExecutionStatus {
	switch s {
	case "running":
		return pb.ExecutionStatus_EXECUTION_STATUS_RUNNING
	case "completed":
		return pb.ExecutionStatus_EXECUTION_STATUS_COMPLETED
	case "awaiting_confirmation":
		return pb.ExecutionStatus_EXECUTION_STATUS_AWAITING_CONFIRMATION
	case "error", "failed":
		return pb.ExecutionStatus_EXECUTION_STATUS_ERROR
	case "cancelled":
		return pb.ExecutionStatus_EXECUTION_STATUS_CANCELLED
	default:
		return pb.ExecutionStatus_EXECUTION_STATUS_UNKNOWN
	}
}
