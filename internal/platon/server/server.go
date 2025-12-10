package server

import (
	"context"
	"fmt"
	"time"

	"github.com/msto63/mDW/api/gen/common"
	pb "github.com/msto63/mDW/api/gen/platon"
	mdwerror "github.com/msto63/mDW/foundation/core/error"
	"github.com/msto63/mDW/internal/platon/chain"
	"github.com/msto63/mDW/internal/platon/service"
	coreGrpc "github.com/msto63/mDW/pkg/core/grpc"
	"github.com/msto63/mDW/pkg/core/health"
	"github.com/msto63/mDW/pkg/core/logging"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Server is the Platon gRPC server
type Server struct {
	pb.UnimplementedPlatonServiceServer
	service   *service.Service
	grpc      *coreGrpc.Server
	health    *health.Registry
	logger    logging.Logger
	config    Config
	startTime time.Time
}

// Config holds server configuration
type Config struct {
	Host     string
	Port     int
	HTTPPort int
}

// DefaultConfig returns default server configuration
func DefaultConfig() Config {
	return Config{
		Host:     "0.0.0.0",
		Port:     9130,
		HTTPPort: 9131,
	}
}

// New creates a new Platon server
func New(cfg Config) (*Server, error) {
	logger := logging.New("platon-server")

	// Create service
	svcCfg := service.DefaultConfig()
	svc := service.NewService(svcCfg, *logging.New("platon-service"))

	// Load default pipeline
	if err := svc.LoadDefaultPipeline(); err != nil {
		return nil, mdwerror.Wrap(err, "failed to load default pipeline").
			WithCode(mdwerror.CodeServiceInitialization).
			WithOperation("server.New")
	}

	// Create gRPC server
	grpcCfg := coreGrpc.DefaultServerConfig()
	grpcCfg.Host = cfg.Host
	grpcCfg.Port = cfg.Port

	grpcServer := coreGrpc.NewServer(grpcCfg)

	// Create health registry
	healthRegistry := health.NewRegistry("platon", "1.0.0")
	healthRegistry.RegisterFunc("service", func(ctx context.Context) health.CheckResult {
		stats := svc.Stats()
		return health.CheckResult{
			Name:    "service",
			Status:  health.StatusHealthy,
			Message: "Platon pipeline service is operational",
			Details: stats,
		}
	})

	server := &Server{
		service:   svc,
		grpc:      grpcServer,
		health:    healthRegistry,
		logger:    *logger,
		config:    cfg,
		startTime: time.Now(),
	}

	// Register gRPC service
	pb.RegisterPlatonServiceServer(grpcServer.GRPCServer(), server)

	return server, nil
}

// ============================================================================
// gRPC Processing Methods
// ============================================================================

// Process executes the full pipeline processing
func (s *Server) Process(ctx context.Context, req *pb.ProcessRequest) (*pb.ProcessResponse, error) {
	s.logger.Debug("Processing request",
		"request_id", req.RequestId,
		"pipeline_id", req.PipelineId)

	chainReq := s.protoToChainRequest(req)

	// For full processing, we need a main processor - return error if not available via gRPC
	// Full processing is typically called programmatically with a callback
	result, err := s.service.ProcessPre(ctx, chainReq)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "processing failed: %v", err)
	}

	return s.chainResultToProto(result), nil
}

// ProcessPre executes pre-processing
func (s *Server) ProcessPre(ctx context.Context, req *pb.ProcessRequest) (*pb.ProcessResponse, error) {
	s.logger.Debug("Pre-processing request",
		"request_id", req.RequestId,
		"pipeline_id", req.PipelineId)

	chainReq := s.protoToChainRequest(req)

	result, err := s.service.ProcessPre(ctx, chainReq)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "pre-processing failed: %v", err)
	}

	return s.chainResultToProto(result), nil
}

// ProcessPost executes post-processing
func (s *Server) ProcessPost(ctx context.Context, req *pb.ProcessRequest) (*pb.ProcessResponse, error) {
	s.logger.Debug("Post-processing request",
		"request_id", req.RequestId,
		"pipeline_id", req.PipelineId)

	chainReq := s.protoToChainRequest(req)

	result, err := s.service.ProcessPost(ctx, chainReq)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "post-processing failed: %v", err)
	}

	return s.chainResultToProto(result), nil
}

// ============================================================================
// gRPC Handler Management Methods
// ============================================================================

// RegisterHandler registers a new dynamic handler
func (s *Server) RegisterHandler(ctx context.Context, req *pb.RegisterHandlerRequest) (*pb.HandlerInfo, error) {
	s.logger.Debug("Registering handler",
		"name", req.Name,
		"type", req.Type.String(),
		"priority", req.Priority)

	// Convert proto type to chain type
	handlerType := handlerTypeFromProto(req.Type)

	// Build settings from config
	var settings map[string]string
	var enabled bool = true
	if req.Config != nil {
		settings = req.Config.Settings
		enabled = req.Config.Enabled
	}

	// Register the dynamic handler
	cfg := service.DynamicHandlerConfig{
		Name:        req.Name,
		Type:        handlerType,
		Priority:    int(req.Priority),
		Description: req.Description,
		Enabled:     enabled,
		Settings:    settings,
	}

	h, err := s.service.RegisterDynamicHandler(cfg)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to register handler: %v", err)
	}

	return &pb.HandlerInfo{
		Name:        h.Name(),
		Type:        req.Type,
		Priority:    req.Priority,
		Description: h.Description(),
		Enabled:     h.IsEnabled(),
		Config:      h.Config(),
	}, nil
}

// UnregisterHandler removes a handler
func (s *Server) UnregisterHandler(ctx context.Context, req *pb.UnregisterHandlerRequest) (*common.Empty, error) {
	if s.service.UnregisterHandler(req.Name) {
		return &common.Empty{}, nil
	}
	return nil, status.Errorf(codes.NotFound, "handler not found: %s", req.Name)
}

// GetHandler returns handler info
func (s *Server) GetHandler(ctx context.Context, req *pb.GetHandlerRequest) (*pb.HandlerInfo, error) {
	h, found := s.service.GetHandler(req.Name)
	if !found {
		return nil, status.Errorf(codes.NotFound, "handler not found: %s", req.Name)
	}

	return &pb.HandlerInfo{
		Name:     h.Name(),
		Type:     handlerTypeToProto(h.Type()),
		Priority: int32(h.Priority()),
		Enabled:  true, // Handler is registered, so it's enabled
	}, nil
}

// ListHandlers returns all handlers
func (s *Server) ListHandlers(ctx context.Context, _ *common.Empty) (*pb.HandlerListResponse, error) {
	handlers := s.service.ListHandlers()

	pbHandlers := make([]*pb.HandlerInfo, len(handlers))
	for i, h := range handlers {
		pbHandlers[i] = &pb.HandlerInfo{
			Name:     h.Name,
			Type:     handlerTypeToProto(h.Type),
			Priority: int32(h.Priority),
			Enabled:  h.Enabled,
		}
	}

	return &pb.HandlerListResponse{
		Handlers: pbHandlers,
		Total:    int32(len(handlers)),
	}, nil
}

// ============================================================================
// gRPC Pipeline Management Methods
// ============================================================================

// CreatePipeline creates a new pipeline
func (s *Server) CreatePipeline(ctx context.Context, req *pb.CreatePipelineRequest) (*pb.PipelineInfo, error) {
	pipeline := &chain.Pipeline{
		ID:           req.Id,
		Name:         req.Name,
		Description:  req.Description,
		Enabled:      req.Enabled,
		PreHandlers:  req.PreHandlers,
		PostHandlers: req.PostHandlers,
		Config:       req.Config,
	}

	if err := s.service.CreatePipeline(pipeline); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create pipeline: %v", err)
	}

	return s.pipelineToProto(pipeline), nil
}

// UpdatePipeline updates an existing pipeline
func (s *Server) UpdatePipeline(ctx context.Context, req *pb.UpdatePipelineRequest) (*pb.PipelineInfo, error) {
	pipeline := &chain.Pipeline{
		ID:           req.Id,
		Name:         req.Name,
		Description:  req.Description,
		Enabled:      req.Enabled,
		PreHandlers:  req.PreHandlers,
		PostHandlers: req.PostHandlers,
		Config:       req.Config,
	}

	if err := s.service.UpdatePipeline(pipeline); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to update pipeline: %v", err)
	}

	return s.pipelineToProto(pipeline), nil
}

// DeletePipeline removes a pipeline
func (s *Server) DeletePipeline(ctx context.Context, req *pb.DeletePipelineRequest) (*common.Empty, error) {
	if err := s.service.DeletePipeline(req.Id); err != nil {
		return nil, status.Errorf(codes.NotFound, "pipeline not found: %s", req.Id)
	}
	return &common.Empty{}, nil
}

// GetPipeline returns a pipeline by ID
func (s *Server) GetPipeline(ctx context.Context, req *pb.GetPipelineRequest) (*pb.PipelineInfo, error) {
	pipeline, err := s.service.GetPipeline(req.Id)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "pipeline not found: %s", req.Id)
	}
	return s.pipelineToProto(pipeline), nil
}

// ListPipelines returns all pipelines
func (s *Server) ListPipelines(ctx context.Context, _ *common.Empty) (*pb.PipelineListResponse, error) {
	pipelines := s.service.ListPipelines()

	pbPipelines := make([]*pb.PipelineInfo, len(pipelines))
	for i, p := range pipelines {
		pbPipelines[i] = s.pipelineToProto(p)
	}

	return &pb.PipelineListResponse{
		Pipelines: pbPipelines,
		Total:     int32(len(pipelines)),
	}, nil
}

// ============================================================================
// gRPC Policy Management Methods
// ============================================================================

// CreatePolicy creates a new policy
func (s *Server) CreatePolicy(ctx context.Context, req *pb.CreatePolicyRequest) (*pb.PolicyInfo, error) {
	policy := s.protoToPolicy(req.Id, req.Name, req.Description, req.Type, req.Enabled, req.Priority, req.Rules, req.LlmCheck)

	if err := s.service.CreatePolicy(policy); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create policy: %v", err)
	}

	return s.policyToProto(policy), nil
}

// UpdatePolicy updates an existing policy
func (s *Server) UpdatePolicy(ctx context.Context, req *pb.UpdatePolicyRequest) (*pb.PolicyInfo, error) {
	policy := s.protoToPolicy(req.Id, req.Name, req.Description, req.Type, req.Enabled, req.Priority, req.Rules, req.LlmCheck)

	if err := s.service.UpdatePolicy(policy); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to update policy: %v", err)
	}

	return s.policyToProto(policy), nil
}

// DeletePolicy removes a policy
func (s *Server) DeletePolicy(ctx context.Context, req *pb.DeletePolicyRequest) (*common.Empty, error) {
	if err := s.service.DeletePolicy(req.Id); err != nil {
		return nil, status.Errorf(codes.NotFound, "policy not found: %s", req.Id)
	}
	return &common.Empty{}, nil
}

// GetPolicy returns a policy by ID
func (s *Server) GetPolicy(ctx context.Context, req *pb.GetPolicyRequest) (*pb.PolicyInfo, error) {
	policy, err := s.service.GetPolicy(req.Id)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "policy not found: %s", req.Id)
	}
	return s.policyToProto(policy), nil
}

// ListPolicies returns all policies
func (s *Server) ListPolicies(ctx context.Context, _ *common.Empty) (*pb.PolicyListResponse, error) {
	policies := s.service.ListPolicies()

	pbPolicies := make([]*pb.PolicyInfo, len(policies))
	for i, p := range policies {
		pbPolicies[i] = s.policyToProto(p)
	}

	return &pb.PolicyListResponse{
		Policies: pbPolicies,
		Total:    int32(len(policies)),
	}, nil
}

// TestPolicy tests a policy against sample text
func (s *Server) TestPolicy(ctx context.Context, req *pb.TestPolicyRequest) (*pb.TestPolicyResponse, error) {
	if req.Policy == nil {
		return nil, status.Errorf(codes.InvalidArgument, "policy is required")
	}

	policy := s.protoToPolicy(
		req.Policy.Id,
		req.Policy.Name,
		req.Policy.Description,
		req.Policy.Type,
		req.Policy.Enabled,
		req.Policy.Priority,
		req.Policy.Rules,
		req.Policy.LlmCheck,
	)

	result, err := s.service.TestPolicy(policy, req.TestText)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to test policy: %v", err)
	}

	// Convert violations
	violations := make([]*pb.PolicyViolation, len(result.Violations))
	for i, v := range result.Violations {
		violations[i] = &pb.PolicyViolation{
			PolicyId:    v.PolicyID,
			PolicyName:  v.PolicyName,
			RuleId:      v.RuleID,
			Severity:    v.Severity,
			Description: v.Description,
			Location:    v.Location,
			Action:      policyActionStringToProto(v.Action),
			Matched:     v.Matched,
		}
	}

	return &pb.TestPolicyResponse{
		Decision:     policyDecisionStringToProto(result.Decision),
		Violations:   violations,
		ModifiedText: result.ModifiedText,
		Reason:       result.Reason,
		DurationMs:   result.Duration.Milliseconds(),
	}, nil
}

// ============================================================================
// gRPC Health Method
// ============================================================================

// HealthCheck performs a health check
func (s *Server) HealthCheck(ctx context.Context, req *common.HealthCheckRequest) (*common.HealthCheckResponse, error) {
	report := s.health.Check(ctx)

	// Convert checks to details map
	details := make(map[string]string)
	for _, c := range report.Checks {
		details[c.Name] = fmt.Sprintf("%s: %s", c.Status, c.Message)
	}
	details["uptime"] = report.Uptime.String()

	return &common.HealthCheckResponse{
		Status:        string(report.Status),
		Service:       report.Service,
		Version:       report.Version,
		UptimeSeconds: int64(report.Uptime.Seconds()),
		Details:       details,
	}, nil
}

// ============================================================================
// Server Lifecycle Methods
// ============================================================================

// RegisterHandlerDirect registers a handler with the pipeline
func (s *Server) RegisterHandlerDirect(h chain.Handler) error {
	return s.service.RegisterHandler(h)
}

// Start starts the server
func (s *Server) Start() error {
	s.logger.Info("Starting Platon server", "host", s.config.Host, "port", s.config.Port)
	return s.grpc.Start()
}

// StartAsync starts the server asynchronously
func (s *Server) StartAsync() error {
	s.logger.Info("Starting Platon server (async)", "host", s.config.Host, "port", s.config.Port)
	return s.grpc.StartAsync()
}

// Stop stops the server
func (s *Server) Stop(ctx context.Context) error {
	s.logger.Info("Stopping Platon server")
	s.grpc.StopWithTimeout(ctx)
	return s.service.Close()
}

// GRPCServer returns the underlying gRPC server
func (s *Server) GRPCServer() *grpc.Server {
	return s.grpc.GRPCServer()
}

// HealthRegistry returns the health check registry
func (s *Server) HealthRegistry() *health.Registry {
	return s.health
}

// Service returns the underlying service
func (s *Server) Service() *service.Service {
	return s.service
}

// Chain returns the handler chain
func (s *Server) Chain() *chain.Chain {
	return s.service.Chain()
}

// Stats returns server statistics
func (s *Server) Stats() map[string]interface{} {
	stats := s.service.Stats()
	stats["uptime"] = time.Since(s.startTime).String()
	stats["host"] = s.config.Host
	stats["port"] = s.config.Port
	return stats
}

// ============================================================================
// Helper Methods
// ============================================================================

// protoToChainRequest converts protobuf request to chain request
func (s *Server) protoToChainRequest(req *pb.ProcessRequest) *chain.ProcessRequest {
	metadata := make(map[string]any)
	for k, v := range req.Metadata {
		metadata[k] = v
	}

	return &chain.ProcessRequest{
		RequestID:  req.RequestId,
		PipelineID: req.PipelineId,
		Prompt:     req.Prompt,
		Response:   req.Response,
		Metadata:   metadata,
	}
}

// chainResultToProto converts chain result to protobuf response
func (s *Server) chainResultToProto(result *chain.ProcessResult) *pb.ProcessResponse {
	auditLog := make([]*pb.AuditEntry, len(result.AuditLog))
	for i, entry := range result.AuditLog {
		details := make(map[string]string)
		for k, v := range entry.Details {
			details[k] = fmt.Sprintf("%v", v)
		}

		errStr := ""
		if entry.Error != nil {
			errStr = entry.Error.Error()
		}

		auditLog[i] = &pb.AuditEntry{
			Handler:    entry.Handler,
			Phase:      entry.Phase.String(),
			DurationMs: entry.Duration.Milliseconds(),
			Error:      errStr,
			Modified:   entry.Modified,
			Details:    details,
		}
	}

	metadata := make(map[string]string)
	for k, v := range result.Metadata {
		metadata[k] = fmt.Sprintf("%v", v)
	}

	return &pb.ProcessResponse{
		RequestId:         result.RequestID,
		ProcessedPrompt:   result.ProcessedPrompt,
		ProcessedResponse: result.ProcessedResponse,
		Blocked:           result.Blocked,
		BlockReason:       result.BlockReason,
		Modified:          result.Modified,
		AuditLog:          auditLog,
		Metadata:          metadata,
		DurationMs:        result.Duration.Milliseconds(),
	}
}

// pipelineToProto converts pipeline to protobuf
func (s *Server) pipelineToProto(p *chain.Pipeline) *pb.PipelineInfo {
	return &pb.PipelineInfo{
		Id:           p.ID,
		Name:         p.Name,
		Description:  p.Description,
		Enabled:      p.Enabled,
		PreHandlers:  p.PreHandlers,
		PostHandlers: p.PostHandlers,
		Config:       p.Config,
		CreatedAt:    p.CreatedAt.Unix(),
		UpdatedAt:    p.UpdatedAt.Unix(),
	}
}

// handlerTypeToProto converts handler type to protobuf enum
func handlerTypeToProto(t chain.HandlerType) pb.HandlerType {
	switch t {
	case chain.HandlerTypePre:
		return pb.HandlerType_HANDLER_TYPE_PRE
	case chain.HandlerTypePost:
		return pb.HandlerType_HANDLER_TYPE_POST
	case chain.HandlerTypeBoth:
		return pb.HandlerType_HANDLER_TYPE_BOTH
	default:
		return pb.HandlerType_HANDLER_TYPE_UNKNOWN
	}
}

func handlerTypeFromProto(t pb.HandlerType) chain.HandlerType {
	switch t {
	case pb.HandlerType_HANDLER_TYPE_PRE:
		return chain.HandlerTypePre
	case pb.HandlerType_HANDLER_TYPE_POST:
		return chain.HandlerTypePost
	case pb.HandlerType_HANDLER_TYPE_BOTH:
		return chain.HandlerTypeBoth
	default:
		return chain.HandlerTypePre
	}
}

// protoToPolicy converts proto request to service policy
func (s *Server) protoToPolicy(id, name, description string, pType pb.PolicyType, enabled bool, priority int32, rules []*pb.PolicyRule, llmCheck *pb.LLMCheckConfig) *service.Policy {
	policy := &service.Policy{
		ID:          id,
		Name:        name,
		Description: description,
		Type:        policyTypeFromProto(pType),
		Enabled:     enabled,
		Priority:    int(priority),
		Rules:       make([]service.PolicyRule, len(rules)),
	}

	for i, r := range rules {
		policy.Rules[i] = service.PolicyRule{
			ID:            r.Id,
			Pattern:       r.Pattern,
			Action:        policyActionFromProto(r.Action),
			Message:       r.Message,
			Replacement:   r.Replacement,
			CaseSensitive: r.CaseSensitive,
		}
	}

	if llmCheck != nil {
		policy.LLMCheck = &service.LLMCheckConfig{
			Enabled:        llmCheck.Enabled,
			Model:          llmCheck.Model,
			Prompt:         llmCheck.Prompt,
			TimeoutSeconds: int(llmCheck.TimeoutSeconds),
			Temperature:    llmCheck.Temperature,
		}
	}

	return policy
}

// policyToProto converts service policy to protobuf
func (s *Server) policyToProto(p *service.Policy) *pb.PolicyInfo {
	rules := make([]*pb.PolicyRule, len(p.Rules))
	for i, r := range p.Rules {
		rules[i] = &pb.PolicyRule{
			Id:            r.ID,
			Pattern:       r.Pattern,
			Action:        policyActionStringToProto(r.Action),
			Message:       r.Message,
			Replacement:   r.Replacement,
			CaseSensitive: r.CaseSensitive,
		}
	}

	info := &pb.PolicyInfo{
		Id:          p.ID,
		Name:        p.Name,
		Description: p.Description,
		Type:        policyTypeToProto(p.Type),
		Enabled:     p.Enabled,
		Priority:    int32(p.Priority),
		Rules:       rules,
		CreatedAt:   p.CreatedAt.Unix(),
		UpdatedAt:   p.UpdatedAt.Unix(),
	}

	if p.LLMCheck != nil {
		info.LlmCheck = &pb.LLMCheckConfig{
			Enabled:        p.LLMCheck.Enabled,
			Model:          p.LLMCheck.Model,
			Prompt:         p.LLMCheck.Prompt,
			TimeoutSeconds: int32(p.LLMCheck.TimeoutSeconds),
			Temperature:    p.LLMCheck.Temperature,
		}
	}

	return info
}

// policyTypeFromProto converts protobuf policy type to string
func policyTypeFromProto(t pb.PolicyType) string {
	switch t {
	case pb.PolicyType_POLICY_TYPE_CONTENT:
		return "content"
	case pb.PolicyType_POLICY_TYPE_SAFETY:
		return "safety"
	case pb.PolicyType_POLICY_TYPE_SCOPE:
		return "scope"
	case pb.PolicyType_POLICY_TYPE_PII:
		return "pii"
	case pb.PolicyType_POLICY_TYPE_CUSTOM:
		return "custom"
	default:
		return "unknown"
	}
}

// policyTypeToProto converts string policy type to protobuf
func policyTypeToProto(t string) pb.PolicyType {
	switch t {
	case "content":
		return pb.PolicyType_POLICY_TYPE_CONTENT
	case "safety":
		return pb.PolicyType_POLICY_TYPE_SAFETY
	case "scope":
		return pb.PolicyType_POLICY_TYPE_SCOPE
	case "pii":
		return pb.PolicyType_POLICY_TYPE_PII
	case "custom":
		return pb.PolicyType_POLICY_TYPE_CUSTOM
	default:
		return pb.PolicyType_POLICY_TYPE_UNKNOWN
	}
}

// policyActionFromProto converts protobuf policy action to string
func policyActionFromProto(a pb.PolicyAction) string {
	switch a {
	case pb.PolicyAction_POLICY_ACTION_BLOCK:
		return "block"
	case pb.PolicyAction_POLICY_ACTION_ALLOW:
		return "allow"
	case pb.PolicyAction_POLICY_ACTION_REDACT:
		return "redact"
	case pb.PolicyAction_POLICY_ACTION_WARN:
		return "warn"
	case pb.PolicyAction_POLICY_ACTION_LOG:
		return "log"
	default:
		return "unknown"
	}
}

// policyActionStringToProto converts string policy action to protobuf
func policyActionStringToProto(a string) pb.PolicyAction {
	switch a {
	case "block":
		return pb.PolicyAction_POLICY_ACTION_BLOCK
	case "allow":
		return pb.PolicyAction_POLICY_ACTION_ALLOW
	case "redact":
		return pb.PolicyAction_POLICY_ACTION_REDACT
	case "warn":
		return pb.PolicyAction_POLICY_ACTION_WARN
	case "log":
		return pb.PolicyAction_POLICY_ACTION_LOG
	default:
		return pb.PolicyAction_POLICY_ACTION_UNKNOWN
	}
}

// policyDecisionStringToProto converts string decision to protobuf
func policyDecisionStringToProto(d string) pb.PolicyDecision {
	switch d {
	case "allow":
		return pb.PolicyDecision_POLICY_DECISION_ALLOW
	case "block":
		return pb.PolicyDecision_POLICY_DECISION_BLOCK
	case "modify":
		return pb.PolicyDecision_POLICY_DECISION_MODIFY
	case "escalate":
		return pb.PolicyDecision_POLICY_DECISION_ESCALATE
	default:
		return pb.PolicyDecision_POLICY_DECISION_UNKNOWN
	}
}
