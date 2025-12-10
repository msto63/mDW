package integration

import (
	"io"
	"testing"
	"time"

	commonpb "github.com/msto63/mDW/api/gen/common"
	leibnizpb "github.com/msto63/mDW/api/gen/leibniz"
)

func TestLeibniz_HealthCheck(t *testing.T) {
	cfg := getTestConfig()
	skipIfServiceUnavailable(t, cfg.LeibnizAddr, "Leibniz")
	logTestStart(t, "Leibniz", "HealthCheck")

	conn := dialGRPC(t, cfg.LeibnizAddr)
	client := leibnizpb.NewLeibnizServiceClient(conn)

	ctx, cancel := testContext(t, 10*time.Second)
	defer cancel()

	resp, err := client.HealthCheck(ctx, &commonpb.HealthCheckRequest{})
	requireNoError(t, err, "HealthCheck failed")
	requireEqual(t, "healthy", resp.Status, "Service should be healthy")

	t.Logf("Leibniz health: status=%s version=%s", resp.Status, resp.Version)
}

func TestLeibniz_ListTools(t *testing.T) {
	cfg := getTestConfig()
	skipIfServiceUnavailable(t, cfg.LeibnizAddr, "Leibniz")
	logTestStart(t, "Leibniz", "ListTools")

	conn := dialGRPC(t, cfg.LeibnizAddr)
	client := leibnizpb.NewLeibnizServiceClient(conn)

	ctx, cancel := testContext(t, 10*time.Second)
	defer cancel()

	resp, err := client.ListTools(ctx, &commonpb.Empty{})
	requireNoError(t, err, "ListTools failed")

	t.Logf("Found %d tools", len(resp.Tools))
	for _, tool := range resp.Tools {
		t.Logf("  - %s: %s", tool.Name, tool.Description)
	}
}

func TestLeibniz_AgentLifecycle(t *testing.T) {
	cfg := getTestConfig()
	skipIfServiceUnavailable(t, cfg.LeibnizAddr, "Leibniz")
	logTestStart(t, "Leibniz", "AgentLifecycle")

	conn := dialGRPC(t, cfg.LeibnizAddr)
	client := leibnizpb.NewLeibnizServiceClient(conn)

	ctx, cancel := testContext(t, 30*time.Second)
	defer cancel()

	// Create agent
	t.Log("Creating agent...")
	createResp, err := client.CreateAgent(ctx, &leibnizpb.CreateAgentRequest{
		Name:         "test-agent",
		Description:  "Integration test agent",
		SystemPrompt: "Du bist ein hilfreicher Assistent für Tests.",
		Config: &leibnizpb.AgentConfig{
			Model:         "mistral:7b",
			MaxIterations: 5,
		},
	})
	requireNoError(t, err, "CreateAgent failed")
	requireNotEmpty(t, createResp.Id, "Agent ID should not be empty")

	agentID := createResp.Id
	t.Logf("Created agent: %s", agentID)

	// Get agent
	t.Log("Getting agent...")
	getResp, err := client.GetAgent(ctx, &leibnizpb.GetAgentRequest{
		Id: agentID,
	})
	requireNoError(t, err, "GetAgent failed")
	requireEqual(t, "test-agent", getResp.Name, "Agent name mismatch")

	// List agents
	t.Log("Listing agents...")
	listResp, err := client.ListAgents(ctx, &commonpb.Empty{})
	requireNoError(t, err, "ListAgents failed")

	found := false
	for _, agent := range listResp.Agents {
		if agent.Id == agentID {
			found = true
			t.Logf("Found agent: %s (%s)", agent.Name, agent.Id)
		}
	}
	requireTrue(t, found, "Created agent not found in list")

	// Update agent
	t.Log("Updating agent...")
	_, err = client.UpdateAgent(ctx, &leibnizpb.UpdateAgentRequest{
		Id:          agentID,
		Description: "Updated integration test agent",
	})
	requireNoError(t, err, "UpdateAgent failed")

	// Delete agent
	t.Log("Deleting agent...")
	_, err = client.DeleteAgent(ctx, &leibnizpb.DeleteAgentRequest{
		Id: agentID,
	})
	requireNoError(t, err, "DeleteAgent failed")
	t.Log("Agent deleted successfully")
}

func TestLeibniz_Execute(t *testing.T) {
	cfg := getTestConfig()
	skipIfServiceUnavailable(t, cfg.LeibnizAddr, "Leibniz")
	skipIfServiceUnavailable(t, cfg.OllamaAddr, "Ollama")
	logTestStart(t, "Leibniz", "Execute")

	conn := dialGRPC(t, cfg.LeibnizAddr)
	client := leibnizpb.NewLeibnizServiceClient(conn)

	// Retry up to 5 times for flaky LLM responses (GPU contention during full test suite)
	var resp *leibnizpb.ExecuteResponse
	var err error
	for attempt := 1; attempt <= 5; attempt++ {
		ctx, cancel := testContext(t, 150*time.Second)

		t.Logf("Executing task (attempt %d/5)...", attempt)
		// Use a simple prompt that doesn't require tool usage to avoid max iterations issue
		resp, err = client.Execute(ctx, &leibnizpb.ExecuteRequest{
			AgentId: "default",
			Message: "Antworte nur mit 'OK'.",
		})
		cancel()

		if err == nil && resp.Response != "" {
			break
		}
		if attempt < 5 {
			t.Logf("Empty response or error, retrying... (err=%v)", err)
			time.Sleep(3 * time.Second)
		}
	}

	requireNoError(t, err, "Execute failed after 5 attempts")
	requireNotEmpty(t, resp.Response, "Response should not be empty after 5 attempts")

	t.Logf("Agent response: %s", resp.Response)
	t.Logf("Status: %s, Actions: %d", resp.Status, len(resp.Actions))
}

func TestLeibniz_StreamExecute(t *testing.T) {
	cfg := getTestConfig()
	skipIfServiceUnavailable(t, cfg.LeibnizAddr, "Leibniz")
	skipIfServiceUnavailable(t, cfg.OllamaAddr, "Ollama")
	logTestStart(t, "Leibniz", "StreamExecute")

	conn := dialGRPC(t, cfg.LeibnizAddr)
	client := leibnizpb.NewLeibnizServiceClient(conn)

	ctx, cancel := testContext(t, 120*time.Second)
	defer cancel()

	// Stream execute
	t.Log("Starting streaming execution...")
	stream, err := client.StreamExecute(ctx, &leibnizpb.ExecuteRequest{
		AgentId: "default",
		Message: "Zähle von 1 bis 3.",
	})
	requireNoError(t, err, "StreamExecute failed")

	chunkCount := 0
	var lastChunk *leibnizpb.AgentChunk
	for {
		chunk, err := stream.Recv()
		if err == io.EOF {
			break
		}
		requireNoError(t, err, "Stream receive failed")
		chunkCount++
		lastChunk = chunk

		switch chunk.Type {
		case leibnizpb.ChunkType_CHUNK_TYPE_THINKING:
			t.Logf("  [THINKING] %s", chunk.Content)
		case leibnizpb.ChunkType_CHUNK_TYPE_TOOL_CALL:
			t.Logf("  [TOOL] %s", chunk.Content)
		case leibnizpb.ChunkType_CHUNK_TYPE_TOOL_RESULT:
			t.Logf("  [RESULT] %s", chunk.Content)
		case leibnizpb.ChunkType_CHUNK_TYPE_RESPONSE:
			t.Logf("  [RESPONSE] %s", chunk.Content)
		case leibnizpb.ChunkType_CHUNK_TYPE_FINAL:
			t.Logf("  [FINAL] %s", chunk.Content)
		}
	}

	requireTrue(t, chunkCount > 0, "Should receive at least one chunk")
	if lastChunk != nil {
		requireEqual(t, leibnizpb.ChunkType_CHUNK_TYPE_FINAL, lastChunk.Type, "Last chunk should be FINAL")
	}

	t.Logf("Received %d chunks", chunkCount)
}

func TestLeibniz_RegisterTool(t *testing.T) {
	cfg := getTestConfig()
	skipIfServiceUnavailable(t, cfg.LeibnizAddr, "Leibniz")
	logTestStart(t, "Leibniz", "RegisterTool")

	conn := dialGRPC(t, cfg.LeibnizAddr)
	client := leibnizpb.NewLeibnizServiceClient(conn)

	ctx, cancel := testContext(t, 10*time.Second)
	defer cancel()

	toolName := "test_tool"

	// Register custom tool
	t.Log("Registering tool...")
	_, err := client.RegisterTool(ctx, &leibnizpb.RegisterToolRequest{
		Name:            toolName,
		Description:     "A test tool for integration testing",
		ParameterSchema: `{"type":"object","properties":{"input":{"type":"string"}}}`,
	})
	requireNoError(t, err, "RegisterTool failed")
	t.Log("Tool registered")

	// Verify in list
	listResp, err := client.ListTools(ctx, &commonpb.Empty{})
	requireNoError(t, err, "ListTools failed")

	found := false
	for _, tool := range listResp.Tools {
		if tool.Name == toolName {
			found = true
			break
		}
	}
	requireTrue(t, found, "Registered tool not found in list")

	// Unregister tool
	t.Log("Unregistering tool...")
	_, err = client.UnregisterTool(ctx, &leibnizpb.UnregisterToolRequest{
		Name: toolName,
	})
	requireNoError(t, err, "UnregisterTool failed")
	t.Log("Tool unregistered successfully")
}
