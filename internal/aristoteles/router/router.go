// Package router provides service routing for the Aristoteles service
package router

import (
	"context"
	"fmt"
	"strings"
	"time"

	pb "github.com/msto63/mDW/api/gen/aristoteles"
	babbagepb "github.com/msto63/mDW/api/gen/babbage"
	hypatiapb "github.com/msto63/mDW/api/gen/hypatia"
	leibnizpb "github.com/msto63/mDW/api/gen/leibniz"
	turingpb "github.com/msto63/mDW/api/gen/turing"
	"github.com/msto63/mDW/internal/aristoteles/orchestrator"
	"github.com/msto63/mDW/internal/aristoteles/pipeline"
	"github.com/msto63/mDW/pkg/core/logging"
)

// TuringClient is the interface for Turing service calls
type TuringClient interface {
	Chat(ctx context.Context, req *turingpb.ChatRequest) (*turingpb.ChatResponse, error)
}

// LeibnizClient is the interface for Leibniz service calls
type LeibnizClient interface {
	Execute(ctx context.Context, req *leibnizpb.ExecuteRequest) (*leibnizpb.ExecuteResponse, error)
	FindBestAgent(ctx context.Context, req *leibnizpb.FindAgentRequest) (*leibnizpb.AgentMatchResponse, error)
	FindTopAgents(ctx context.Context, req *leibnizpb.FindTopAgentsRequest) (*leibnizpb.AgentMatchListResponse, error)
}

// HypatiaClient is the interface for Hypatia service calls
type HypatiaClient interface {
	AugmentPrompt(ctx context.Context, req *hypatiapb.AugmentPromptRequest) (*hypatiapb.AugmentPromptResponse, error)
}

// BabbageClient is the interface for Babbage service calls
type BabbageClient interface {
	Summarize(ctx context.Context, req *babbagepb.SummarizeRequest) (*babbagepb.SummarizeResponse, error)
	Translate(ctx context.Context, req *babbagepb.TranslateRequest) (*babbagepb.TranslateResponse, error)
}

// Router routes requests to the appropriate service
type Router struct {
	turingClient         TuringClient
	leibnizClient        LeibnizClient
	hypatiaClient        HypatiaClient
	babbageClient        BabbageClient
	orchestrator         *orchestrator.Orchestrator
	logger               *logging.Logger
	timeout              time.Duration
	enableAutoAgentMatch bool    // Enable RAG-style agent matching
	minAgentConfidence   float64 // Minimum confidence for agent matching
	enableOrchestrator   bool    // Enable multi-task orchestration
}

// Config holds router configuration
type Config struct {
	DefaultTimeout       time.Duration
	EnableAutoAgentMatch bool    // Enable RAG-style agent matching
	MinAgentConfidence   float64 // Minimum confidence for agent matching (0.0-1.0)
	EnableOrchestrator   bool    // Enable multi-task orchestration for complex prompts
}

// DefaultConfig returns default router configuration
func DefaultConfig() *Config {
	return &Config{
		DefaultTimeout:       180 * time.Second, // Increased for agent tasks like web research
		EnableAutoAgentMatch: true,              // Enable automatic agent selection
		MinAgentConfidence:   0.3,               // 30% minimum confidence
		EnableOrchestrator:   true,              // Enable task decomposition and multi-agent orchestration
	}
}

// NewRouter creates a new router
func NewRouter(cfg *Config) *Router {
	if cfg == nil {
		cfg = DefaultConfig()
	}

	r := &Router{
		logger:               logging.New("aristoteles-router"),
		timeout:              cfg.DefaultTimeout,
		enableAutoAgentMatch: cfg.EnableAutoAgentMatch,
		minAgentConfidence:   cfg.MinAgentConfidence,
		enableOrchestrator:   cfg.EnableOrchestrator,
	}

	// Initialize orchestrator if enabled
	if cfg.EnableOrchestrator {
		r.orchestrator = orchestrator.NewOrchestrator(orchestrator.Config{
			DefaultAgentID: "default",
			MinConfidence:  cfg.MinAgentConfidence,
		})
	}

	return r
}

// SetTuringClient sets the Turing client
func (r *Router) SetTuringClient(client TuringClient) {
	r.turingClient = client

	// Configure orchestrator's decomposer with LLM function
	if r.orchestrator != nil && client != nil {
		r.orchestrator.SetDecomposerLLM(func(ctx context.Context, systemPrompt, userPrompt string) (string, error) {
			resp, err := client.Chat(ctx, &turingpb.ChatRequest{
				Model: "mistral:7b", // Use fast model for decomposition
				Messages: []*turingpb.Message{
					{Role: "system", Content: systemPrompt},
					{Role: "user", Content: userPrompt},
				},
				Temperature: 0.1, // Low temperature for structured output
				MaxTokens:   1000,
			})
			if err != nil {
				return "", err
			}
			return resp.Content, nil
		})
		r.logger.Debug("Orchestrator decomposer configured with Turing LLM")
	}
}

// SetLeibnizClient sets the Leibniz client
func (r *Router) SetLeibnizClient(client LeibnizClient) {
	r.leibnizClient = client

	// Configure orchestrator with Leibniz functions
	if r.orchestrator != nil && client != nil {
		// Agent Matcher: Find best agent for a task description
		r.orchestrator.SetAgentMatcher(func(ctx context.Context, taskDescription string) (*orchestrator.AgentMatch, error) {
			resp, err := client.FindBestAgent(ctx, &leibnizpb.FindAgentRequest{
				TaskDescription: taskDescription,
			})
			if err != nil {
				return nil, err
			}
			return &orchestrator.AgentMatch{
				AgentID:    resp.AgentId,
				AgentName:  resp.AgentName,
				Similarity: resp.Similarity,
			}, nil
		})

		// Agent Executor: Execute a task with a specific agent
		r.orchestrator.SetAgentExecutor(func(ctx context.Context, agentID, prompt string) (string, error) {
			resp, err := client.Execute(ctx, &leibnizpb.ExecuteRequest{
				AgentId: agentID,
				Message: prompt,
			})
			if err != nil {
				return "", err
			}
			return resp.Response, nil
		})

		r.logger.Info("Orchestrator configured with Leibniz client")
	}
}

// SetHypatiaClient sets the Hypatia client
func (r *Router) SetHypatiaClient(client HypatiaClient) {
	r.hypatiaClient = client
}

// SetBabbageClient sets the Babbage client
func (r *Router) SetBabbageClient(client BabbageClient) {
	r.babbageClient = client
}

// Route executes the request against the appropriate service
func (r *Router) Route(ctx context.Context, pctx *pipeline.Context) error {
	if pctx.Strategy == nil {
		return fmt.Errorf("no strategy selected")
	}

	start := time.Now()
	var err error

	switch pctx.Strategy.Target {
	case pb.TargetService_TARGET_TURING:
		err = r.routeToTuring(ctx, pctx)
	case pb.TargetService_TARGET_LEIBNIZ:
		err = r.routeToLeibniz(ctx, pctx)
	case pb.TargetService_TARGET_HYPATIA:
		err = r.routeToHypatia(ctx, pctx)
	case pb.TargetService_TARGET_BABBAGE:
		err = r.routeToBabbage(ctx, pctx)
	case pb.TargetService_TARGET_MULTI:
		err = r.routeMulti(ctx, pctx)
	default:
		err = r.routeToTuring(ctx, pctx) // Default fallback
	}

	if err != nil {
		return err
	}

	pctx.Route.DurationMs = time.Since(start).Milliseconds()
	return nil
}

// routeToTuring routes the request to Turing (LLM)
func (r *Router) routeToTuring(ctx context.Context, pctx *pipeline.Context) error {
	if r.turingClient == nil {
		return fmt.Errorf("turing client not available")
	}

	// Build messages
	messages := []*turingpb.Message{
		{Role: "user", Content: pctx.GetEnrichedPrompt()},
	}

	req := &turingpb.ChatRequest{
		Model:       pctx.Strategy.Model,
		Messages:    messages,
		Temperature: pctx.Strategy.Temperature,
		MaxTokens:   pctx.Strategy.MaxTokens,
	}

	resp, err := r.turingClient.Chat(ctx, req)
	if err != nil {
		// Try fallback model
		if pctx.Strategy.FallbackModel != "" {
			r.logger.Warn("Primary model failed, trying fallback",
				"primary", pctx.Strategy.Model,
				"fallback", pctx.Strategy.FallbackModel,
				"error", err)
			req.Model = pctx.Strategy.FallbackModel
			resp, err = r.turingClient.Chat(ctx, req)
		}
		if err != nil {
			return fmt.Errorf("turing chat failed: %w", err)
		}
	}

	pctx.Response = resp.Content
	pctx.Route = &pb.RouteInfo{
		Service:   pb.TargetService_TARGET_TURING,
		Endpoint:  "Chat",
		Model:     resp.Model,
		TokensIn:  resp.PromptTokens,
		TokensOut: resp.CompletionTokens,
	}

	return nil
}

// routeToLeibniz routes the request to Leibniz (Agent)
func (r *Router) routeToLeibniz(ctx context.Context, pctx *pipeline.Context) error {
	if r.leibnizClient == nil {
		return fmt.Errorf("leibniz client not available")
	}

	// Check if this is a multi-step task that should use the orchestrator
	// Only use orchestrator if:
	// 1. Orchestrator is enabled and configured
	// 2. Intent is TASK_DECOMPOSITION or MULTI_STEP
	// 3. No forced agent is specified (manual selection bypasses orchestration)
	useOrchestrator := r.shouldUseOrchestrator(pctx)

	if useOrchestrator {
		return r.routeViaOrchestrator(ctx, pctx)
	}

	// Standard single-agent routing
	return r.routeToLeibnizSingle(ctx, pctx)
}

// shouldUseOrchestrator checks if the orchestrator should be used for this request
func (r *Router) shouldUseOrchestrator(pctx *pipeline.Context) bool {
	// Orchestrator must be enabled and configured
	if !r.enableOrchestrator || r.orchestrator == nil {
		return false
	}

	// Manual agent selection bypasses orchestration
	if pctx.Options != nil && pctx.Options.ForceAgent != "" {
		return false
	}

	// Check intent type
	if pctx.Intent == nil {
		return false
	}

	// Use orchestrator for multi-step and task decomposition intents
	switch pctx.Intent.Primary {
	case pb.IntentType_INTENT_TYPE_TASK_DECOMPOSITION,
		pb.IntentType_INTENT_TYPE_MULTI_STEP:
		return true
	}

	return false
}

// routeViaOrchestrator uses the orchestrator for multi-task execution
func (r *Router) routeViaOrchestrator(ctx context.Context, pctx *pipeline.Context) error {
	r.logger.Info("Using orchestrator for multi-task execution",
		"intent", pctx.Intent.Primary.String())

	result, err := r.orchestrator.Process(ctx, pctx.GetEnrichedPrompt())
	if err != nil {
		// Fallback to single-agent if orchestration fails
		r.logger.Warn("Orchestration failed, falling back to single-agent",
			"error", err)
		return r.routeToLeibnizSingle(ctx, pctx)
	}

	// Build response from orchestration result
	pctx.Response = result.FinalOutput

	// Collect agent info from all tasks
	var agentIDs, agentNames []string
	for _, task := range result.Plan.Tasks {
		agentIDs = append(agentIDs, task.AssignedAgentID)
		agentNames = append(agentNames, task.AgentName)
	}

	pctx.Route = &pb.RouteInfo{
		Service:    pb.TargetService_TARGET_LEIBNIZ,
		Endpoint:   "Orchestrator",
		AgentId:    strings.Join(agentIDs, ","),
		DurationMs: result.TotalDuration.Milliseconds(),
	}

	// Store orchestration info in metadata
	if pctx.Metadata == nil {
		pctx.Metadata = make(map[string]string)
	}
	pctx.Metadata["orchestration_mode"] = "multi_task"
	pctx.Metadata["task_count"] = fmt.Sprintf("%d", len(result.Plan.Tasks))
	pctx.Metadata["agents_used"] = strings.Join(agentNames, ", ")
	if result.Plan.IsSequential {
		pctx.Metadata["execution_mode"] = "sequential"
	} else {
		pctx.Metadata["execution_mode"] = "parallel"
	}

	r.logger.Info("Orchestration completed",
		"tasks", len(result.Plan.Tasks),
		"agents", strings.Join(agentNames, ", "),
		"duration_ms", result.TotalDuration.Milliseconds())

	return nil
}

// routeToLeibnizSingle routes to a single agent (original behavior)
func (r *Router) routeToLeibnizSingle(ctx context.Context, pctx *pipeline.Context) error {
	agentID := "default"
	agentName := "Default Agent"
	confidence := float64(0)

	// Priority 1: Use forced agent from options (UI manual selection)
	if pctx.Options != nil && pctx.Options.ForceAgent != "" {
		agentID = pctx.Options.ForceAgent
		agentName = "Manuell ausgewählt"
		confidence = 1.0 // 100% confidence for manual selection
		r.logger.Info("Using forced agent from options", "agent", agentID)
	} else if len(pctx.Strategy.Agents) > 0 {
		// Priority 2: Use explicit agent if specified in strategy
		agentID = pctx.Strategy.Agents[0]
		r.logger.Debug("Using strategy-specified agent", "agent", agentID)
	} else if r.enableAutoAgentMatch {
		// Priority 3: RAG-style automatic agent selection
		matchResp, err := r.leibnizClient.FindBestAgent(ctx, &leibnizpb.FindAgentRequest{
			TaskDescription: pctx.Prompt,
		})
		if err != nil {
			r.logger.Warn("Agent matching failed, using default", "error", err)
		} else if matchResp.Similarity >= r.minAgentConfidence {
			agentID = matchResp.AgentId
			agentName = matchResp.AgentName
			confidence = matchResp.Similarity
			r.logger.Info("Agent auto-selected via embedding similarity",
				"agent", agentID,
				"name", agentName,
				"confidence", fmt.Sprintf("%.2f%%", confidence*100))
		} else {
			r.logger.Debug("Agent confidence too low, using default",
				"matched_agent", matchResp.AgentId,
				"confidence", matchResp.Similarity,
				"threshold", r.minAgentConfidence)
		}
	}

	req := &leibnizpb.ExecuteRequest{
		AgentId:        agentID,
		Message:        pctx.GetEnrichedPrompt(),
		ConversationId: pctx.ConversationID,
	}

	resp, err := r.leibnizClient.Execute(ctx, req)
	if err != nil {
		return fmt.Errorf("leibniz execute failed: %w", err)
	}

	pctx.Response = resp.Response
	pctx.Route = &pb.RouteInfo{
		Service:   pb.TargetService_TARGET_LEIBNIZ,
		Endpoint:  "Execute",
		AgentId:   agentID,
		TokensOut: resp.TotalTokens,
	}

	// Store agent match info in metadata for UI display
	if confidence > 0 {
		if pctx.Metadata == nil {
			pctx.Metadata = make(map[string]string)
		}
		pctx.Metadata["matched_agent_id"] = agentID
		pctx.Metadata["matched_agent_name"] = agentName
		pctx.Metadata["agent_confidence"] = fmt.Sprintf("%.2f", confidence)
	}

	return nil
}

// routeToHypatia routes the request to Hypatia (RAG)
func (r *Router) routeToHypatia(ctx context.Context, pctx *pipeline.Context) error {
	if r.hypatiaClient == nil {
		return fmt.Errorf("hypatia client not available")
	}

	req := &hypatiapb.AugmentPromptRequest{
		Prompt: pctx.GetEnrichedPrompt(),
		TopK:   5,
	}

	resp, err := r.hypatiaClient.AugmentPrompt(ctx, req)
	if err != nil {
		return fmt.Errorf("hypatia augment failed: %w", err)
	}

	// If there's an augmented prompt, we still need to call Turing
	if r.turingClient != nil {
		messages := []*turingpb.Message{
			{Role: "user", Content: resp.AugmentedPrompt},
		}

		chatReq := &turingpb.ChatRequest{
			Model:       pctx.Strategy.Model,
			Messages:    messages,
			Temperature: pctx.Strategy.Temperature,
			MaxTokens:   pctx.Strategy.MaxTokens,
		}

		chatResp, err := r.turingClient.Chat(ctx, chatReq)
		if err != nil {
			return fmt.Errorf("turing chat after RAG failed: %w", err)
		}

		pctx.Response = chatResp.Content
		pctx.Route = &pb.RouteInfo{
			Service:   pb.TargetService_TARGET_HYPATIA,
			Endpoint:  "AugmentPrompt+Chat",
			Model:     chatResp.Model,
			TokensIn:  chatResp.PromptTokens,
			TokensOut: chatResp.CompletionTokens,
		}
	} else {
		pctx.Response = resp.AugmentedPrompt
		pctx.Route = &pb.RouteInfo{
			Service:  pb.TargetService_TARGET_HYPATIA,
			Endpoint: "AugmentPrompt",
		}
	}

	return nil
}

// routeToBabbage routes the request to Babbage (NLP)
func (r *Router) routeToBabbage(ctx context.Context, pctx *pipeline.Context) error {
	if r.babbageClient == nil {
		return fmt.Errorf("babbage client not available")
	}

	// Determine operation based on intent
	if pctx.Intent != nil && pctx.Intent.Primary == pb.IntentType_INTENT_TYPE_TRANSLATION {
		// Translation
		req := &babbagepb.TranslateRequest{
			Text: pctx.Prompt,
			// Target language would need to be detected from prompt
		}

		resp, err := r.babbageClient.Translate(ctx, req)
		if err != nil {
			return fmt.Errorf("babbage translate failed: %w", err)
		}

		pctx.Response = resp.TranslatedText
		pctx.Route = &pb.RouteInfo{
			Service:  pb.TargetService_TARGET_BABBAGE,
			Endpoint: "Translate",
		}
	} else {
		// Summarization
		req := &babbagepb.SummarizeRequest{
			Text: pctx.Prompt,
		}

		resp, err := r.babbageClient.Summarize(ctx, req)
		if err != nil {
			return fmt.Errorf("babbage summarize failed: %w", err)
		}

		pctx.Response = resp.Summary
		pctx.Route = &pb.RouteInfo{
			Service:  pb.TargetService_TARGET_BABBAGE,
			Endpoint: "Summarize",
		}
	}

	return nil
}

// routeMulti orchestrates multiple services
func (r *Router) routeMulti(ctx context.Context, pctx *pipeline.Context) error {
	// For now, fall back to Turing
	// Future: parallel execution of multiple services
	r.logger.Info("Multi-service routing not fully implemented, using Turing fallback")
	return r.routeToTuring(ctx, pctx)
}

// Stage is the routing pipeline stage
type Stage struct {
	router *Router
}

// NewStage creates a new router stage
func NewStage(router *Router) *Stage {
	return &Stage{router: router}
}

// Name returns the stage name
func (s *Stage) Name() string {
	return "router"
}

// Execute runs the routing stage
func (s *Stage) Execute(ctx context.Context, pctx *pipeline.Context) error {
	start := time.Now()

	// Get timeout from options or use default
	timeout := s.router.timeout
	if pctx.Options != nil && pctx.Options.TimeoutSeconds > 0 {
		timeout = time.Duration(pctx.Options.TimeoutSeconds) * time.Second
	}

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	if err := s.router.Route(ctx, pctx); err != nil {
		return err
	}

	pctx.Metrics.RoutingDurationMs = time.Since(start).Milliseconds()
	if pctx.Route != nil {
		pctx.Metrics.TotalTokens = pctx.Route.TokensIn + pctx.Route.TokensOut
	}

	return nil
}
