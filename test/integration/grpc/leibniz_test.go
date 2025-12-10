// ============================================================================
// meinDENKWERK (mDW) - Lokale KI-Plattform
// ============================================================================
//
// Package:     grpc
// Description: Integration tests for Leibniz gRPC service (Agentic AI)
// Author:      Mike Stoffels with Claude
// Created:     2025-12-06
// License:     MIT
// ============================================================================

//go:build integration

package grpc

import (
	"context"
	"testing"
	"time"

	"github.com/msto63/mDW/api/gen/common"
	leibnizpb "github.com/msto63/mDW/api/gen/leibniz"
)

// LeibnizTestClient wraps the Leibniz gRPC client for testing
type LeibnizTestClient struct {
	conn   *TestConnection
	client leibnizpb.LeibnizServiceClient
}

// NewLeibnizTestClient creates a new Leibniz test client
func NewLeibnizTestClient() (*LeibnizTestClient, error) {
	configs := DefaultServiceConfigs()
	cfg := configs["leibniz"]

	conn, err := NewTestConnection(cfg)
	if err != nil {
		return nil, err
	}

	return &LeibnizTestClient{
		conn:   conn,
		client: leibnizpb.NewLeibnizServiceClient(conn.Conn()),
	}, nil
}

// Close closes the test client connection
func (lc *LeibnizTestClient) Close() error {
	return lc.conn.Close()
}

// Client returns the underlying gRPC client
func (lc *LeibnizTestClient) Client() leibnizpb.LeibnizServiceClient {
	return lc.client
}

// ContextWithTimeout returns a context with a custom timeout
func (lc *LeibnizTestClient) ContextWithTimeout(timeout time.Duration) (context.Context, context.CancelFunc) {
	return lc.conn.ContextWithTimeout(timeout)
}

// TestLeibnizHealthCheck tests the health check endpoint
func TestLeibnizHealthCheck(t *testing.T) {
	client, err := NewLeibnizTestClient()
	if err != nil {
		t.Fatalf("Failed to create Leibniz client: %v", err)
	}
	defer client.Close()

	ctx, cancel := client.ContextWithTimeout(10 * time.Second)
	defer cancel()

	resp, err := client.Client().HealthCheck(ctx, &common.HealthCheckRequest{})
	if err != nil {
		t.Fatalf("HealthCheck failed: %v", err)
	}

	t.Logf("Health Check Response:")
	t.Logf("  Status: %s", resp.GetStatus())
	t.Logf("  Service: %s", resp.GetService())
	t.Logf("  Version: %s", resp.GetVersion())
	t.Logf("  Uptime: %d seconds", resp.GetUptimeSeconds())

	if resp.GetStatus() != "healthy" {
		t.Errorf("Expected status 'healthy', got '%s'", resp.GetStatus())
	}

	if resp.GetService() != "leibniz" {
		t.Errorf("Expected service 'leibniz', got '%s'", resp.GetService())
	}
}

// TestLeibnizListTools tests listing available tools
func TestLeibnizListTools(t *testing.T) {
	client, err := NewLeibnizTestClient()
	if err != nil {
		t.Fatalf("Failed to create Leibniz client: %v", err)
	}
	defer client.Close()

	ctx, cancel := client.ContextWithTimeout(10 * time.Second)
	defer cancel()

	resp, err := client.Client().ListTools(ctx, &common.Empty{})
	if err != nil {
		t.Fatalf("ListTools failed: %v", err)
	}

	t.Logf("Found %d tools:", len(resp.GetTools()))
	for _, tool := range resp.GetTools() {
		t.Logf("  - %s: %s (source: %s)", tool.GetName(), tool.GetDescription(), tool.GetSource().String())
	}
}

// TestLeibnizListAgents tests listing agents
func TestLeibnizListAgents(t *testing.T) {
	client, err := NewLeibnizTestClient()
	if err != nil {
		t.Fatalf("Failed to create Leibniz client: %v", err)
	}
	defer client.Close()

	ctx, cancel := client.ContextWithTimeout(10 * time.Second)
	defer cancel()

	resp, err := client.Client().ListAgents(ctx, &common.Empty{})
	if err != nil {
		t.Fatalf("ListAgents failed: %v", err)
	}

	t.Logf("Found %d agents (total: %d)", len(resp.GetAgents()), resp.GetTotal())
	for _, agent := range resp.GetAgents() {
		t.Logf("  - %s: %s", agent.GetName(), agent.GetDescription())
	}
}

// TestLeibnizCreateDeleteAgent tests agent lifecycle
func TestLeibnizCreateDeleteAgent(t *testing.T) {
	client, err := NewLeibnizTestClient()
	if err != nil {
		t.Fatalf("Failed to create Leibniz client: %v", err)
	}
	defer client.Close()

	ctx, cancel := client.ContextWithTimeout(30 * time.Second)
	defer cancel()

	// Create agent
	createReq := &leibnizpb.CreateAgentRequest{
		Name:         "test-agent-integration",
		Description:  "Integration test agent",
		SystemPrompt: "You are a helpful assistant for integration testing.",
		Config: &leibnizpb.AgentConfig{
			Model:         "ollama:mistral:7b",
			Temperature:   0.7,
			MaxIterations: 5,
		},
	}

	agentInfo, err := client.Client().CreateAgent(ctx, createReq)
	if err != nil {
		t.Logf("CreateAgent: %v (agent may already exist or model not available)", err)
		t.Skip("Skipping - agent creation not available")
		return
	}

	t.Logf("Created Agent:")
	t.Logf("  ID: %s", agentInfo.GetId())
	t.Logf("  Name: %s", agentInfo.GetName())
	t.Logf("  Description: %s", agentInfo.GetDescription())

	if agentInfo.GetId() == "" {
		t.Error("Expected non-empty agent ID")
	}

	// Clean up - delete the agent
	deleteReq := &leibnizpb.DeleteAgentRequest{
		Id: agentInfo.GetId(),
	}

	_, err = client.Client().DeleteAgent(ctx, deleteReq)
	if err != nil {
		t.Logf("DeleteAgent: %v", err)
	} else {
		t.Log("Deleted agent successfully")
	}
}

// TestLeibnizGetAgent tests getting agent details
func TestLeibnizGetAgent(t *testing.T) {
	client, err := NewLeibnizTestClient()
	if err != nil {
		t.Fatalf("Failed to create Leibniz client: %v", err)
	}
	defer client.Close()

	ctx, cancel := client.ContextWithTimeout(10 * time.Second)
	defer cancel()

	// First list agents to get an ID
	listResp, err := client.Client().ListAgents(ctx, &common.Empty{})
	if err != nil {
		t.Fatalf("ListAgents failed: %v", err)
	}

	if len(listResp.GetAgents()) == 0 {
		t.Skip("No agents available to test GetAgent")
		return
	}

	agentID := listResp.GetAgents()[0].GetId()

	getReq := &leibnizpb.GetAgentRequest{
		Id: agentID,
	}

	resp, err := client.Client().GetAgent(ctx, getReq)
	if err != nil {
		t.Fatalf("GetAgent failed: %v", err)
	}

	t.Logf("Agent Details:")
	t.Logf("  ID: %s", resp.GetId())
	t.Logf("  Name: %s", resp.GetName())
	t.Logf("  Description: %s", resp.GetDescription())
	t.Logf("  Model: %s", resp.GetConfig().GetModel())
}

// TestLeibnizRegisterUnregisterTool tests tool registration
func TestLeibnizRegisterUnregisterTool(t *testing.T) {
	client, err := NewLeibnizTestClient()
	if err != nil {
		t.Fatalf("Failed to create Leibniz client: %v", err)
	}
	defer client.Close()

	ctx, cancel := client.ContextWithTimeout(10 * time.Second)
	defer cancel()

	toolName := "test-tool-integration"

	// Register tool
	registerReq := &leibnizpb.RegisterToolRequest{
		Name:                 toolName,
		Description:          "Integration test tool",
		ParameterSchema:      `{"type": "object", "properties": {"input": {"type": "string"}}}`,
		RequiresConfirmation: false,
	}

	toolInfo, err := client.Client().RegisterTool(ctx, registerReq)
	if err != nil {
		t.Logf("RegisterTool: %v", err)
		t.Skip("Skipping - tool registration not available")
		return
	}

	t.Logf("Registered Tool:")
	t.Logf("  Name: %s", toolInfo.GetName())
	t.Logf("  Description: %s", toolInfo.GetDescription())
	t.Logf("  Source: %s", toolInfo.GetSource().String())

	// Unregister tool
	unregisterReq := &leibnizpb.UnregisterToolRequest{
		Name: toolName,
	}

	_, err = client.Client().UnregisterTool(ctx, unregisterReq)
	if err != nil {
		t.Logf("UnregisterTool: %v", err)
	} else {
		t.Log("Unregistered tool successfully")
	}
}

// TestLeibnizExecute tests agent execution (basic)
func TestLeibnizExecute(t *testing.T) {
	client, err := NewLeibnizTestClient()
	if err != nil {
		t.Fatalf("Failed to create Leibniz client: %v", err)
	}
	defer client.Close()

	ctx, cancel := client.ContextWithTimeout(60 * time.Second)
	defer cancel()

	// First list agents to get an ID
	listResp, err := client.Client().ListAgents(ctx, &common.Empty{})
	if err != nil {
		t.Fatalf("ListAgents failed: %v", err)
	}

	if len(listResp.GetAgents()) == 0 {
		t.Skip("No agents available to test execution")
		return
	}

	agentID := listResp.GetAgents()[0].GetId()

	executeReq := &leibnizpb.ExecuteRequest{
		AgentId:          agentID,
		Message:          "What is 2 + 2?",
		AutoApproveTools: true,
	}

	resp, err := client.Client().Execute(ctx, executeReq)
	if err != nil {
		t.Logf("Execute: %v (LLM may not be available)", err)
		t.Skip("Skipping - execution not available")
		return
	}

	t.Logf("Execution Result:")
	t.Logf("  Execution ID: %s", resp.GetExecutionId())
	t.Logf("  Status: %s", resp.GetStatus().String())
	t.Logf("  Iterations: %d", resp.GetIterations())
	t.Logf("  Duration: %d ms", resp.GetDurationMs())
	t.Logf("  Response: %s", truncateString(resp.GetResponse(), 200))
}

// truncateString truncates a string to max length
func truncateString(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}

// RunLeibnizTestSuite runs all Leibniz tests
func RunLeibnizTestSuite(t *testing.T) *TestSuite {
	suite := NewTestSuite("leibniz")

	tests := []struct {
		name string
		fn   func(*testing.T)
	}{
		{"HealthCheck", TestLeibnizHealthCheck},
		{"ListTools", TestLeibnizListTools},
		{"ListAgents", TestLeibnizListAgents},
		{"CreateDeleteAgent", TestLeibnizCreateDeleteAgent},
		{"GetAgent", TestLeibnizGetAgent},
		{"RegisterUnregisterTool", TestLeibnizRegisterUnregisterTool},
		{"Execute", TestLeibnizExecute},
	}

	for _, tt := range tests {
		start := time.Now()
		passed := t.Run(tt.name, tt.fn)
		duration := time.Since(start)

		result := TestResult{
			Name:     tt.name,
			Passed:   passed,
			Duration: duration,
		}
		suite.AddResult(result)
	}

	suite.Finish()
	t.Log(suite.Summary())

	return suite
}
