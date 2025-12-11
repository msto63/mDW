// ============================================================================
// meinDENKWERK (mDW) - Aristoteles Router Tests
// ============================================================================

package router

import (
	"context"
	"testing"
	"time"

	pb "github.com/msto63/mDW/api/gen/aristoteles"
	leibnizpb "github.com/msto63/mDW/api/gen/leibniz"
	turingpb "github.com/msto63/mDW/api/gen/turing"
	"github.com/msto63/mDW/internal/aristoteles/pipeline"
)

// Mock clients for testing

type mockTuringClient struct {
	chatResponse *turingpb.ChatResponse
	chatErr      error
	chatCalls    int
}

func (m *mockTuringClient) Chat(ctx context.Context, req *turingpb.ChatRequest) (*turingpb.ChatResponse, error) {
	m.chatCalls++
	if m.chatErr != nil {
		return nil, m.chatErr
	}
	return m.chatResponse, nil
}

type mockLeibnizClient struct {
	executeResponse       *leibnizpb.ExecuteResponse
	executeErr            error
	executeCalls          int
	findBestAgentResponse *leibnizpb.AgentMatchResponse
	findBestAgentErr      error
	findBestAgentCalls    int
	findTopAgentsResponse *leibnizpb.AgentMatchListResponse
	findTopAgentsErr      error
}

func (m *mockLeibnizClient) Execute(ctx context.Context, req *leibnizpb.ExecuteRequest) (*leibnizpb.ExecuteResponse, error) {
	m.executeCalls++
	if m.executeErr != nil {
		return nil, m.executeErr
	}
	return m.executeResponse, nil
}

func (m *mockLeibnizClient) FindBestAgent(ctx context.Context, req *leibnizpb.FindAgentRequest) (*leibnizpb.AgentMatchResponse, error) {
	m.findBestAgentCalls++
	if m.findBestAgentErr != nil {
		return nil, m.findBestAgentErr
	}
	return m.findBestAgentResponse, nil
}

func (m *mockLeibnizClient) FindTopAgents(ctx context.Context, req *leibnizpb.FindTopAgentsRequest) (*leibnizpb.AgentMatchListResponse, error) {
	if m.findTopAgentsErr != nil {
		return nil, m.findTopAgentsErr
	}
	return m.findTopAgentsResponse, nil
}

func TestNewRouter(t *testing.T) {
	router := NewRouter(nil)

	if router == nil {
		t.Fatal("NewRouter returned nil")
	}

	// Check defaults
	if router.timeout != 180*time.Second {
		t.Errorf("Expected timeout 180s, got %v", router.timeout)
	}

	if !router.enableAutoAgentMatch {
		t.Error("Expected enableAutoAgentMatch to be true by default")
	}

	if router.minAgentConfidence != 0.3 {
		t.Errorf("Expected minAgentConfidence 0.3, got %v", router.minAgentConfidence)
	}
}

func TestNewRouter_WithConfig(t *testing.T) {
	cfg := &Config{
		DefaultTimeout:       60 * time.Second,
		EnableAutoAgentMatch: false,
		MinAgentConfidence:   0.5,
	}

	router := NewRouter(cfg)

	if router.timeout != 60*time.Second {
		t.Errorf("Expected timeout 60s, got %v", router.timeout)
	}

	if router.enableAutoAgentMatch {
		t.Error("Expected enableAutoAgentMatch to be false")
	}

	if router.minAgentConfidence != 0.5 {
		t.Errorf("Expected minAgentConfidence 0.5, got %v", router.minAgentConfidence)
	}
}

func TestRouter_SetClients(t *testing.T) {
	router := NewRouter(nil)

	turingClient := &mockTuringClient{}
	leibnizClient := &mockLeibnizClient{}

	router.SetTuringClient(turingClient)
	router.SetLeibnizClient(leibnizClient)

	if router.turingClient == nil {
		t.Error("Turing client should be set")
	}

	if router.leibnizClient == nil {
		t.Error("Leibniz client should be set")
	}
}

func TestRouter_Route_NoStrategy(t *testing.T) {
	router := NewRouter(nil)

	pctx := &pipeline.Context{
		Prompt:   "Test prompt",
		Strategy: nil, // No strategy
	}

	err := router.Route(context.Background(), pctx)

	if err == nil {
		t.Error("Expected error when no strategy is set")
	}
}

func TestRouter_RouteToTuring(t *testing.T) {
	router := NewRouter(nil)

	mockClient := &mockTuringClient{
		chatResponse: &turingpb.ChatResponse{
			Content:          "Test response",
			Model:            "test-model",
			PromptTokens:     10,
			CompletionTokens: 20,
		},
	}
	router.SetTuringClient(mockClient)

	pctx := &pipeline.Context{
		Prompt: "Test prompt",
		Strategy: &pb.StrategyInfo{
			Target:      pb.TargetService_TARGET_TURING,
			Model:       "test-model",
			Temperature: 0.7,
			MaxTokens:   100,
		},
		Metrics: &pb.PipelineMetrics{},
	}

	err := router.Route(context.Background(), pctx)

	if err != nil {
		t.Fatalf("Route failed: %v", err)
	}

	if mockClient.chatCalls != 1 {
		t.Errorf("Expected 1 chat call, got %d", mockClient.chatCalls)
	}

	if pctx.Response != "Test response" {
		t.Errorf("Unexpected response: %s", pctx.Response)
	}

	if pctx.Route == nil {
		t.Fatal("Route info should be set")
	}

	if pctx.Route.Service != pb.TargetService_TARGET_TURING {
		t.Error("Route service should be Turing")
	}
}

func TestRouter_RouteToLeibniz_WithForceAgent(t *testing.T) {
	router := NewRouter(nil)

	mockClient := &mockLeibnizClient{
		executeResponse: &leibnizpb.ExecuteResponse{
			Response:    "Agent response",
			TotalTokens: 50,
		},
	}
	router.SetLeibnizClient(mockClient)

	pctx := &pipeline.Context{
		Prompt: "Test prompt",
		Strategy: &pb.StrategyInfo{
			Target: pb.TargetService_TARGET_LEIBNIZ,
		},
		Options: &pb.ProcessOptions{
			ForceAgent: "web-researcher", // Force specific agent
		},
		Metrics: &pb.PipelineMetrics{},
	}

	err := router.Route(context.Background(), pctx)

	if err != nil {
		t.Fatalf("Route failed: %v", err)
	}

	// Should NOT call FindBestAgent when ForceAgent is set
	if mockClient.findBestAgentCalls != 0 {
		t.Errorf("FindBestAgent should not be called when ForceAgent is set, got %d calls", mockClient.findBestAgentCalls)
	}

	if mockClient.executeCalls != 1 {
		t.Errorf("Expected 1 execute call, got %d", mockClient.executeCalls)
	}

	if pctx.Response != "Agent response" {
		t.Errorf("Unexpected response: %s", pctx.Response)
	}

	// Check metadata for forced agent
	if pctx.Metadata == nil {
		t.Fatal("Metadata should be set")
	}

	if pctx.Metadata["matched_agent_id"] != "web-researcher" {
		t.Errorf("Expected agent_id 'web-researcher', got %s", pctx.Metadata["matched_agent_id"])
	}

	if pctx.Metadata["agent_confidence"] != "1.00" {
		t.Errorf("Expected confidence '1.00' for forced agent, got %s", pctx.Metadata["agent_confidence"])
	}
}

func TestRouter_RouteToLeibniz_WithAutoAgentMatch(t *testing.T) {
	cfg := &Config{
		EnableAutoAgentMatch: true,
		MinAgentConfidence:   0.3,
	}
	router := NewRouter(cfg)

	mockClient := &mockLeibnizClient{
		executeResponse: &leibnizpb.ExecuteResponse{
			Response:    "Agent response",
			TotalTokens: 50,
		},
		findBestAgentResponse: &leibnizpb.AgentMatchResponse{
			AgentId:    "auto-selected-agent",
			AgentName:  "Auto Selected Agent",
			Similarity: 0.85,
		},
	}
	router.SetLeibnizClient(mockClient)

	pctx := &pipeline.Context{
		Prompt: "Research this topic on the web",
		Strategy: &pb.StrategyInfo{
			Target: pb.TargetService_TARGET_LEIBNIZ,
		},
		Metrics: &pb.PipelineMetrics{},
	}

	err := router.Route(context.Background(), pctx)

	if err != nil {
		t.Fatalf("Route failed: %v", err)
	}

	// Should call FindBestAgent for auto-selection
	if mockClient.findBestAgentCalls != 1 {
		t.Errorf("Expected 1 FindBestAgent call, got %d", mockClient.findBestAgentCalls)
	}

	// Check metadata for auto-selected agent
	if pctx.Metadata == nil {
		t.Fatal("Metadata should be set")
	}

	if pctx.Metadata["matched_agent_id"] != "auto-selected-agent" {
		t.Errorf("Expected agent_id 'auto-selected-agent', got %s", pctx.Metadata["matched_agent_id"])
	}
}

func TestRouter_RouteToLeibniz_LowConfidenceFallback(t *testing.T) {
	cfg := &Config{
		EnableAutoAgentMatch: true,
		MinAgentConfidence:   0.5, // High threshold
	}
	router := NewRouter(cfg)

	mockClient := &mockLeibnizClient{
		executeResponse: &leibnizpb.ExecuteResponse{
			Response: "Default agent response",
		},
		findBestAgentResponse: &leibnizpb.AgentMatchResponse{
			AgentId:    "some-agent",
			AgentName:  "Some Agent",
			Similarity: 0.3, // Below threshold
		},
	}
	router.SetLeibnizClient(mockClient)

	pctx := &pipeline.Context{
		Prompt: "Vague task",
		Strategy: &pb.StrategyInfo{
			Target: pb.TargetService_TARGET_LEIBNIZ,
		},
		Metrics: &pb.PipelineMetrics{},
	}

	err := router.Route(context.Background(), pctx)

	if err != nil {
		t.Fatalf("Route failed: %v", err)
	}

	// Should use default agent when confidence is too low
	// Metadata should NOT contain the low-confidence match
	if pctx.Metadata != nil && pctx.Metadata["matched_agent_id"] != "" {
		t.Error("Should not set agent metadata for low-confidence matches")
	}
}

func TestRouter_RouteToLeibniz_NoClient(t *testing.T) {
	router := NewRouter(nil)
	// Not setting leibniz client

	pctx := &pipeline.Context{
		Prompt: "Test prompt",
		Strategy: &pb.StrategyInfo{
			Target: pb.TargetService_TARGET_LEIBNIZ,
		},
		Metrics: &pb.PipelineMetrics{},
	}

	err := router.Route(context.Background(), pctx)

	if err == nil {
		t.Error("Expected error when Leibniz client is not set")
	}
}

func TestRouter_DefaultFallback(t *testing.T) {
	router := NewRouter(nil)

	mockClient := &mockTuringClient{
		chatResponse: &turingpb.ChatResponse{
			Content: "Fallback response",
		},
	}
	router.SetTuringClient(mockClient)

	pctx := &pipeline.Context{
		Prompt: "Test prompt",
		Strategy: &pb.StrategyInfo{
			Target: pb.TargetService_TARGET_UNKNOWN, // Unknown target
		},
		Metrics: &pb.PipelineMetrics{},
	}

	err := router.Route(context.Background(), pctx)

	if err != nil {
		t.Fatalf("Route failed: %v", err)
	}

	// Should fallback to Turing
	if mockClient.chatCalls != 1 {
		t.Errorf("Expected fallback to Turing, got %d calls", mockClient.chatCalls)
	}
}

func TestRouterStage_Execute(t *testing.T) {
	router := NewRouter(nil)

	mockClient := &mockTuringClient{
		chatResponse: &turingpb.ChatResponse{
			Content: "Test response",
		},
	}
	router.SetTuringClient(mockClient)

	stage := NewStage(router)

	if stage.Name() != "router" {
		t.Errorf("Expected stage name 'router', got %s", stage.Name())
	}

	pctx := &pipeline.Context{
		Prompt: "Test prompt",
		Strategy: &pb.StrategyInfo{
			Target: pb.TargetService_TARGET_TURING,
		},
		Metrics: &pb.PipelineMetrics{},
	}

	err := stage.Execute(context.Background(), pctx)

	if err != nil {
		t.Fatalf("Stage execute failed: %v", err)
	}

	// RoutingDurationMs can be 0 for very fast operations (under 1ms)
	// Just verify it's not negative
	if pctx.Metrics.RoutingDurationMs < 0 {
		t.Error("RoutingDurationMs should not be negative")
	}
}

func TestRouterStage_Execute_WithTimeout(t *testing.T) {
	router := NewRouter(nil)

	mockClient := &mockTuringClient{
		chatResponse: &turingpb.ChatResponse{
			Content: "Test response",
		},
	}
	router.SetTuringClient(mockClient)

	stage := NewStage(router)

	pctx := &pipeline.Context{
		Prompt: "Test prompt",
		Strategy: &pb.StrategyInfo{
			Target: pb.TargetService_TARGET_TURING,
		},
		Options: &pb.ProcessOptions{
			TimeoutSeconds: 30, // Custom timeout
		},
		Metrics: &pb.PipelineMetrics{},
	}

	err := stage.Execute(context.Background(), pctx)

	if err != nil {
		t.Fatalf("Stage execute failed: %v", err)
	}
}
