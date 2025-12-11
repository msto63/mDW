// ============================================================================
// meinDENKWERK (mDW) - Aristoteles Router Integration Tests
// ============================================================================
//
// These tests verify the full agent selection pipeline.
// Run with: go test -tags=integration ./internal/aristoteles/router/...
// ============================================================================

//go:build integration

package router

import (
	"context"
	"testing"
	"time"

	pb "github.com/msto63/mDW/api/gen/aristoteles"
	leibnizpb "github.com/msto63/mDW/api/gen/leibniz"
	"github.com/msto63/mDW/api/gen/common"
	"github.com/msto63/mDW/internal/aristoteles/pipeline"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const (
	leibnizAddr = "localhost:9140"
)

// TestIntegration_AgentAutoSelection verifies that the router correctly
// selects agents based on task descriptions using the Leibniz service.
func TestIntegration_AgentAutoSelection(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Connect to Leibniz
	conn, err := grpc.DialContext(ctx, leibnizAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		t.Skipf("Leibniz not available at %s: %v", leibnizAddr, err)
	}
	defer conn.Close()

	client := leibnizpb.NewLeibnizServiceClient(conn)

	// Verify Leibniz is healthy
	_, err = client.HealthCheck(ctx, &common.HealthCheckRequest{})
	if err != nil {
		t.Skipf("Leibniz health check failed: %v", err)
	}

	// List available agents
	agentList, err := client.ListAgents(ctx, &common.Empty{})
	if err != nil {
		t.Fatalf("ListAgents failed: %v", err)
	}

	if len(agentList.Agents) == 0 {
		t.Skip("No agents available for testing")
	}

	t.Logf("Found %d agents", len(agentList.Agents))
	for _, agent := range agentList.Agents {
		t.Logf("  - %s: %s", agent.Id, agent.Name)
	}

	// Test agent matching
	testCases := []struct {
		name        string
		task        string
		expectAgent string // empty means any agent is ok
	}{
		{
			name: "Web research task",
			task: "Search the internet for the latest news about AI developments",
		},
		{
			name: "Code review task",
			task: "Review this Python code for bugs and security issues",
		},
		{
			name: "General question",
			task: "What is the capital of France?",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			match, err := client.FindBestAgent(ctx, &leibnizpb.FindAgentRequest{
				TaskDescription: tc.task,
			})

			if err != nil {
				t.Fatalf("FindBestAgent failed: %v", err)
			}

			t.Logf("Task: %s", tc.task)
			t.Logf("Matched Agent: %s (%s) with similarity %.2f%%",
				match.AgentId, match.AgentName, match.Similarity*100)

			if match.AgentId == "" {
				t.Error("AgentId should not be empty")
			}

			if match.Similarity < 0 || match.Similarity > 1 {
				t.Errorf("Similarity should be between 0 and 1, got %f", match.Similarity)
			}
		})
	}
}

// TestIntegration_AgentExecution verifies that agents can be executed
// through the Leibniz service.
func TestIntegration_AgentExecution(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Connect to Leibniz
	conn, err := grpc.DialContext(ctx, leibnizAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		t.Skipf("Leibniz not available at %s: %v", leibnizAddr, err)
	}
	defer conn.Close()

	client := leibnizpb.NewLeibnizServiceClient(conn)

	// Execute with default agent
	resp, err := client.Execute(ctx, &leibnizpb.ExecuteRequest{
		AgentId: "default",
		Message: "What is 2 + 2?",
	})

	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	t.Logf("Response: %s", resp.Response)
	t.Logf("Execution ID: %s", resp.ExecutionId)
	t.Logf("Iterations: %d", resp.Iterations)

	if resp.Response == "" {
		t.Error("Response should not be empty")
	}
}

// TestIntegration_RouterWithLeibnizClient tests the full router
// integration with a real Leibniz client.
func TestIntegration_RouterWithLeibnizClient(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Connect to Leibniz
	conn, err := grpc.DialContext(ctx, leibnizAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		t.Skipf("Leibniz not available at %s: %v", leibnizAddr, err)
	}
	defer conn.Close()

	// Create wrapper
	leibnizClient := newLeibnizClientWrapper(leibnizpb.NewLeibnizServiceClient(conn))

	// Create router with auto-agent-match enabled
	cfg := &Config{
		EnableAutoAgentMatch: true,
		MinAgentConfidence:   0.3,
		DefaultTimeout:       60 * time.Second,
	}
	router := NewRouter(cfg)
	router.SetLeibnizClient(leibnizClient)

	// Test routing to Leibniz with auto-selection
	pctx := &pipeline.Context{
		Prompt: "Search the web for information about Go programming",
		Strategy: &pb.StrategyInfo{
			Target: pb.TargetService_TARGET_LEIBNIZ,
		},
		Metrics: &pb.PipelineMetrics{},
	}

	err = router.Route(ctx, pctx)
	if err != nil {
		t.Fatalf("Route failed: %v", err)
	}

	t.Logf("Response: %s", truncateForLog(pctx.Response, 200))

	if pctx.Route == nil {
		t.Fatal("Route info should be set")
	}

	t.Logf("Agent ID: %s", pctx.Route.AgentId)

	if pctx.Metadata != nil {
		t.Logf("Matched Agent: %s", pctx.Metadata["matched_agent_name"])
		t.Logf("Confidence: %s", pctx.Metadata["agent_confidence"])
	}
}

// TestIntegration_ForceAgentOverride tests that force_agent option
// correctly overrides automatic agent selection.
func TestIntegration_ForceAgentOverride(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Connect to Leibniz
	conn, err := grpc.DialContext(ctx, leibnizAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		t.Skipf("Leibniz not available at %s: %v", leibnizAddr, err)
	}
	defer conn.Close()

	client := leibnizpb.NewLeibnizServiceClient(conn)

	// Get first available agent
	agentList, err := client.ListAgents(ctx, &common.Empty{})
	if err != nil || len(agentList.Agents) == 0 {
		t.Skip("No agents available")
	}

	forcedAgentId := agentList.Agents[0].Id

	// Create wrapper
	leibnizClient := newLeibnizClientWrapper(client)

	// Create router
	cfg := &Config{
		EnableAutoAgentMatch: true,
		MinAgentConfidence:   0.3,
	}
	router := NewRouter(cfg)
	router.SetLeibnizClient(leibnizClient)

	// Test with forced agent
	pctx := &pipeline.Context{
		Prompt: "This task would normally match a different agent",
		Strategy: &pb.StrategyInfo{
			Target: pb.TargetService_TARGET_LEIBNIZ,
		},
		Options: &pb.ProcessOptions{
			ForceAgent: forcedAgentId,
		},
		Metrics: &pb.PipelineMetrics{},
	}

	err = router.Route(ctx, pctx)
	if err != nil {
		t.Fatalf("Route failed: %v", err)
	}

	// Verify the forced agent was used
	if pctx.Route.AgentId != forcedAgentId {
		t.Errorf("Expected forced agent %s, got %s", forcedAgentId, pctx.Route.AgentId)
	}

	// Verify confidence is 1.0 for forced agent
	if pctx.Metadata != nil && pctx.Metadata["agent_confidence"] != "1.00" {
		t.Errorf("Expected confidence 1.00 for forced agent, got %s", pctx.Metadata["agent_confidence"])
	}
}

// leibnizClientWrapper wraps the gRPC client to implement the LeibnizClient interface
type leibnizClientWrapper struct {
	client leibnizpb.LeibnizServiceClient
}

func newLeibnizClientWrapper(client leibnizpb.LeibnizServiceClient) *leibnizClientWrapper {
	return &leibnizClientWrapper{client: client}
}

func (w *leibnizClientWrapper) Execute(ctx context.Context, req *leibnizpb.ExecuteRequest) (*leibnizpb.ExecuteResponse, error) {
	return w.client.Execute(ctx, req)
}

func (w *leibnizClientWrapper) FindBestAgent(ctx context.Context, req *leibnizpb.FindAgentRequest) (*leibnizpb.AgentMatchResponse, error) {
	return w.client.FindBestAgent(ctx, req)
}

func (w *leibnizClientWrapper) FindTopAgents(ctx context.Context, req *leibnizpb.FindTopAgentsRequest) (*leibnizpb.AgentMatchListResponse, error) {
	return w.client.FindTopAgents(ctx, req)
}

func truncateForLog(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
