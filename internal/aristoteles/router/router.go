// Package router provides service routing for the Aristoteles service
package router

import (
	"context"
	"fmt"
	"time"

	pb "github.com/msto63/mDW/api/gen/aristoteles"
	babbagepb "github.com/msto63/mDW/api/gen/babbage"
	hypatiapb "github.com/msto63/mDW/api/gen/hypatia"
	leibnizpb "github.com/msto63/mDW/api/gen/leibniz"
	turingpb "github.com/msto63/mDW/api/gen/turing"
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
	turingClient  TuringClient
	leibnizClient LeibnizClient
	hypatiaClient HypatiaClient
	babbageClient BabbageClient
	logger        *logging.Logger
	timeout       time.Duration
}

// Config holds router configuration
type Config struct {
	DefaultTimeout time.Duration
}

// DefaultConfig returns default router configuration
func DefaultConfig() *Config {
	return &Config{
		DefaultTimeout: 180 * time.Second, // Increased for agent tasks like web research
	}
}

// NewRouter creates a new router
func NewRouter(cfg *Config) *Router {
	if cfg == nil {
		cfg = DefaultConfig()
	}
	return &Router{
		logger:  logging.New("aristoteles-router"),
		timeout: cfg.DefaultTimeout,
	}
}

// SetTuringClient sets the Turing client
func (r *Router) SetTuringClient(client TuringClient) {
	r.turingClient = client
}

// SetLeibnizClient sets the Leibniz client
func (r *Router) SetLeibnizClient(client LeibnizClient) {
	r.leibnizClient = client
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

	agentID := "default"
	if len(pctx.Strategy.Agents) > 0 {
		agentID = pctx.Strategy.Agents[0]
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
