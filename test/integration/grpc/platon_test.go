// ============================================================================
// meinDENKWERK (mDW) - Lokale KI-Plattform
// ============================================================================
//
// Package:     grpc
// Description: Integration tests for Platon gRPC service (Pipeline Processing)
// Author:      Mike Stoffels with Claude
// Created:     2025-12-08
// License:     MIT
// ============================================================================

//go:build integration

package grpc

import (
	"context"
	"testing"
	"time"

	"github.com/msto63/mDW/api/gen/common"
	platonpb "github.com/msto63/mDW/api/gen/platon"
)

// PlatonTestClient wraps the Platon gRPC client for testing
type PlatonTestClient struct {
	conn   *TestConnection
	client platonpb.PlatonServiceClient
}

// NewPlatonTestClient creates a new Platon test client
func NewPlatonTestClient() (*PlatonTestClient, error) {
	configs := DefaultServiceConfigs()
	cfg := configs["platon"]

	conn, err := NewTestConnection(cfg)
	if err != nil {
		return nil, err
	}

	return &PlatonTestClient{
		conn:   conn,
		client: platonpb.NewPlatonServiceClient(conn.Conn()),
	}, nil
}

// Close closes the test client connection
func (pc *PlatonTestClient) Close() error {
	return pc.conn.Close()
}

// Client returns the underlying gRPC client
func (pc *PlatonTestClient) Client() platonpb.PlatonServiceClient {
	return pc.client
}

// ContextWithTimeout returns a context with a custom timeout
func (pc *PlatonTestClient) ContextWithTimeout(timeout time.Duration) (context.Context, context.CancelFunc) {
	return pc.conn.ContextWithTimeout(timeout)
}

// ============================================================================
// Health Check Tests
// ============================================================================

// TestPlatonHealthCheck tests the health check endpoint
func TestPlatonHealthCheck(t *testing.T) {
	client, err := NewPlatonTestClient()
	if err != nil {
		t.Fatalf("Failed to create Platon client: %v", err)
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

	if resp.GetService() != "platon" {
		t.Errorf("Expected service 'platon', got '%s'", resp.GetService())
	}
}

// ============================================================================
// Handler Management Tests
// ============================================================================

// TestPlatonListHandlers tests listing all registered handlers
func TestPlatonListHandlers(t *testing.T) {
	client, err := NewPlatonTestClient()
	if err != nil {
		t.Fatalf("Failed to create Platon client: %v", err)
	}
	defer client.Close()

	ctx, cancel := client.ContextWithTimeout(10 * time.Second)
	defer cancel()

	resp, err := client.Client().ListHandlers(ctx, &common.Empty{})
	if err != nil {
		t.Fatalf("ListHandlers failed: %v", err)
	}

	t.Logf("Listed %d handlers (total: %d):", len(resp.GetHandlers()), resp.GetTotal())
	for _, h := range resp.GetHandlers() {
		t.Logf("  - %s: type=%s, priority=%d, enabled=%v",
			h.GetName(), h.GetType().String(), h.GetPriority(), h.GetEnabled())
	}
}

// TestPlatonRegisterHandler tests handler registration
func TestPlatonRegisterHandler(t *testing.T) {
	client, err := NewPlatonTestClient()
	if err != nil {
		t.Fatalf("Failed to create Platon client: %v", err)
	}
	defer client.Close()

	ctx, cancel := client.ContextWithTimeout(10 * time.Second)
	defer cancel()

	req := &platonpb.RegisterHandlerRequest{
		Name:        "test-handler-integration",
		Type:        platonpb.HandlerType_HANDLER_TYPE_PRE,
		Priority:    100,
		Description: "Test handler for integration tests",
		Config: &platonpb.HandlerConfig{
			Enabled: true,
			Settings: map[string]string{
				"test_key": "test_value",
			},
		},
	}

	resp, err := client.Client().RegisterHandler(ctx, req)
	if err != nil {
		t.Fatalf("RegisterHandler failed: %v", err)
	}

	t.Logf("Registered Handler:")
	t.Logf("  Name: %s", resp.GetName())
	t.Logf("  Type: %s", resp.GetType().String())
	t.Logf("  Priority: %d", resp.GetPriority())
	t.Logf("  Enabled: %v", resp.GetEnabled())

	if resp.GetName() != req.Name {
		t.Errorf("Expected name '%s', got '%s'", req.Name, resp.GetName())
	}

	// Cleanup: unregister the test handler
	_, err = client.Client().UnregisterHandler(ctx, &platonpb.UnregisterHandlerRequest{
		Name: req.Name,
	})
	if err != nil {
		t.Logf("Warning: Failed to cleanup test handler: %v", err)
	}
}

// TestPlatonGetHandler tests getting a specific handler
func TestPlatonGetHandler(t *testing.T) {
	client, err := NewPlatonTestClient()
	if err != nil {
		t.Fatalf("Failed to create Platon client: %v", err)
	}
	defer client.Close()

	ctx, cancel := client.ContextWithTimeout(10 * time.Second)
	defer cancel()

	// First list handlers to get an existing handler name
	listResp, err := client.Client().ListHandlers(ctx, &common.Empty{})
	if err != nil {
		t.Fatalf("ListHandlers failed: %v", err)
	}

	if len(listResp.GetHandlers()) == 0 {
		t.Skip("No handlers registered, skipping GetHandler test")
	}

	handlerName := listResp.GetHandlers()[0].GetName()

	resp, err := client.Client().GetHandler(ctx, &platonpb.GetHandlerRequest{
		Name: handlerName,
	})
	if err != nil {
		t.Fatalf("GetHandler failed: %v", err)
	}

	t.Logf("Retrieved Handler:")
	t.Logf("  Name: %s", resp.GetName())
	t.Logf("  Type: %s", resp.GetType().String())
	t.Logf("  Priority: %d", resp.GetPriority())
	t.Logf("  Description: %s", resp.GetDescription())
	t.Logf("  Enabled: %v", resp.GetEnabled())

	if resp.GetName() != handlerName {
		t.Errorf("Expected name '%s', got '%s'", handlerName, resp.GetName())
	}
}

// ============================================================================
// Pipeline Management Tests
// ============================================================================

// TestPlatonListPipelines tests listing all pipelines
func TestPlatonListPipelines(t *testing.T) {
	client, err := NewPlatonTestClient()
	if err != nil {
		t.Fatalf("Failed to create Platon client: %v", err)
	}
	defer client.Close()

	ctx, cancel := client.ContextWithTimeout(10 * time.Second)
	defer cancel()

	resp, err := client.Client().ListPipelines(ctx, &common.Empty{})
	if err != nil {
		t.Fatalf("ListPipelines failed: %v", err)
	}

	t.Logf("Listed %d pipelines (total: %d):", len(resp.GetPipelines()), resp.GetTotal())
	for _, p := range resp.GetPipelines() {
		t.Logf("  - %s (%s): enabled=%v, pre_handlers=%d, post_handlers=%d",
			p.GetId(), p.GetName(), p.GetEnabled(),
			len(p.GetPreHandlers()), len(p.GetPostHandlers()))
	}
}

// TestPlatonCreatePipeline tests creating a new pipeline
func TestPlatonCreatePipeline(t *testing.T) {
	client, err := NewPlatonTestClient()
	if err != nil {
		t.Fatalf("Failed to create Platon client: %v", err)
	}
	defer client.Close()

	ctx, cancel := client.ContextWithTimeout(10 * time.Second)
	defer cancel()

	pipelineID := "test-pipeline-integration"
	req := &platonpb.CreatePipelineRequest{
		Id:          pipelineID,
		Name:        "Integration Test Pipeline",
		Description: "Pipeline created for integration testing",
		Enabled:     true,
		PreHandlers: []string{},
		PostHandlers: []string{},
		Config: map[string]string{
			"test_mode": "true",
		},
	}

	resp, err := client.Client().CreatePipeline(ctx, req)
	if err != nil {
		t.Fatalf("CreatePipeline failed: %v", err)
	}

	t.Logf("Created Pipeline:")
	t.Logf("  ID: %s", resp.GetId())
	t.Logf("  Name: %s", resp.GetName())
	t.Logf("  Description: %s", resp.GetDescription())
	t.Logf("  Enabled: %v", resp.GetEnabled())
	t.Logf("  Pre-Handlers: %v", resp.GetPreHandlers())
	t.Logf("  Post-Handlers: %v", resp.GetPostHandlers())

	if resp.GetId() != pipelineID {
		t.Errorf("Expected ID '%s', got '%s'", pipelineID, resp.GetId())
	}

	// Cleanup: delete the test pipeline
	_, err = client.Client().DeletePipeline(ctx, &platonpb.DeletePipelineRequest{
		Id: pipelineID,
	})
	if err != nil {
		t.Logf("Warning: Failed to cleanup test pipeline: %v", err)
	}
}

// TestPlatonGetPipeline tests getting a specific pipeline
func TestPlatonGetPipeline(t *testing.T) {
	client, err := NewPlatonTestClient()
	if err != nil {
		t.Fatalf("Failed to create Platon client: %v", err)
	}
	defer client.Close()

	ctx, cancel := client.ContextWithTimeout(10 * time.Second)
	defer cancel()

	// First list pipelines to get an existing pipeline ID
	listResp, err := client.Client().ListPipelines(ctx, &common.Empty{})
	if err != nil {
		t.Fatalf("ListPipelines failed: %v", err)
	}

	if len(listResp.GetPipelines()) == 0 {
		t.Skip("No pipelines configured, skipping GetPipeline test")
	}

	pipelineID := listResp.GetPipelines()[0].GetId()

	resp, err := client.Client().GetPipeline(ctx, &platonpb.GetPipelineRequest{
		Id: pipelineID,
	})
	if err != nil {
		t.Fatalf("GetPipeline failed: %v", err)
	}

	t.Logf("Retrieved Pipeline:")
	t.Logf("  ID: %s", resp.GetId())
	t.Logf("  Name: %s", resp.GetName())
	t.Logf("  Description: %s", resp.GetDescription())
	t.Logf("  Enabled: %v", resp.GetEnabled())
	t.Logf("  Pre-Handlers: %v", resp.GetPreHandlers())
	t.Logf("  Post-Handlers: %v", resp.GetPostHandlers())

	if resp.GetId() != pipelineID {
		t.Errorf("Expected ID '%s', got '%s'", pipelineID, resp.GetId())
	}
}

// TestPlatonUpdatePipeline tests updating a pipeline
func TestPlatonUpdatePipeline(t *testing.T) {
	client, err := NewPlatonTestClient()
	if err != nil {
		t.Fatalf("Failed to create Platon client: %v", err)
	}
	defer client.Close()

	ctx, cancel := client.ContextWithTimeout(10 * time.Second)
	defer cancel()

	// Create a test pipeline first
	pipelineID := "test-pipeline-update"
	createReq := &platonpb.CreatePipelineRequest{
		Id:          pipelineID,
		Name:        "Original Name",
		Description: "Original description",
		Enabled:     true,
	}

	_, err = client.Client().CreatePipeline(ctx, createReq)
	if err != nil {
		t.Fatalf("CreatePipeline failed: %v", err)
	}

	// Update the pipeline
	updateReq := &platonpb.UpdatePipelineRequest{
		Id:          pipelineID,
		Name:        "Updated Name",
		Description: "Updated description",
		Enabled:     false,
	}

	resp, err := client.Client().UpdatePipeline(ctx, updateReq)
	if err != nil {
		// Cleanup even on failure
		client.Client().DeletePipeline(ctx, &platonpb.DeletePipelineRequest{Id: pipelineID})
		t.Fatalf("UpdatePipeline failed: %v", err)
	}

	t.Logf("Updated Pipeline:")
	t.Logf("  ID: %s", resp.GetId())
	t.Logf("  Name: %s", resp.GetName())
	t.Logf("  Description: %s", resp.GetDescription())
	t.Logf("  Enabled: %v", resp.GetEnabled())

	if resp.GetName() != "Updated Name" {
		t.Errorf("Expected name 'Updated Name', got '%s'", resp.GetName())
	}

	// Cleanup
	_, err = client.Client().DeletePipeline(ctx, &platonpb.DeletePipelineRequest{
		Id: pipelineID,
	})
	if err != nil {
		t.Logf("Warning: Failed to cleanup test pipeline: %v", err)
	}
}

// ============================================================================
// Policy Management Tests
// ============================================================================

// TestPlatonListPolicies tests listing all policies
func TestPlatonListPolicies(t *testing.T) {
	client, err := NewPlatonTestClient()
	if err != nil {
		t.Fatalf("Failed to create Platon client: %v", err)
	}
	defer client.Close()

	ctx, cancel := client.ContextWithTimeout(10 * time.Second)
	defer cancel()

	resp, err := client.Client().ListPolicies(ctx, &common.Empty{})
	if err != nil {
		t.Fatalf("ListPolicies failed: %v", err)
	}

	t.Logf("Listed %d policies (total: %d):", len(resp.GetPolicies()), resp.GetTotal())
	for _, p := range resp.GetPolicies() {
		t.Logf("  - %s (%s): type=%s, enabled=%v, priority=%d",
			p.GetId(), p.GetName(), p.GetType().String(), p.GetEnabled(), p.GetPriority())
	}
}

// TestPlatonCreatePolicy tests creating a new policy
func TestPlatonCreatePolicy(t *testing.T) {
	client, err := NewPlatonTestClient()
	if err != nil {
		t.Fatalf("Failed to create Platon client: %v", err)
	}
	defer client.Close()

	ctx, cancel := client.ContextWithTimeout(10 * time.Second)
	defer cancel()

	policyID := "test-policy-integration"
	req := &platonpb.CreatePolicyRequest{
		Id:          policyID,
		Name:        "Integration Test Policy",
		Description: "Policy created for integration testing",
		Type:        platonpb.PolicyType_POLICY_TYPE_CONTENT,
		Enabled:     true,
		Priority:    50,
		Rules: []*platonpb.PolicyRule{
			{
				Id:            "rule-1",
				Pattern:       "test-pattern",
				Action:        platonpb.PolicyAction_POLICY_ACTION_LOG,
				Message:       "Test pattern matched",
				CaseSensitive: false,
			},
		},
	}

	resp, err := client.Client().CreatePolicy(ctx, req)
	if err != nil {
		t.Fatalf("CreatePolicy failed: %v", err)
	}

	t.Logf("Created Policy:")
	t.Logf("  ID: %s", resp.GetId())
	t.Logf("  Name: %s", resp.GetName())
	t.Logf("  Type: %s", resp.GetType().String())
	t.Logf("  Enabled: %v", resp.GetEnabled())
	t.Logf("  Priority: %d", resp.GetPriority())
	t.Logf("  Rules: %d", len(resp.GetRules()))

	if resp.GetId() != policyID {
		t.Errorf("Expected ID '%s', got '%s'", policyID, resp.GetId())
	}

	// Cleanup: delete the test policy
	_, err = client.Client().DeletePolicy(ctx, &platonpb.DeletePolicyRequest{
		Id: policyID,
	})
	if err != nil {
		t.Logf("Warning: Failed to cleanup test policy: %v", err)
	}
}

// TestPlatonGetPolicy tests getting a specific policy
func TestPlatonGetPolicy(t *testing.T) {
	client, err := NewPlatonTestClient()
	if err != nil {
		t.Fatalf("Failed to create Platon client: %v", err)
	}
	defer client.Close()

	ctx, cancel := client.ContextWithTimeout(10 * time.Second)
	defer cancel()

	// First list policies to get an existing policy ID
	listResp, err := client.Client().ListPolicies(ctx, &common.Empty{})
	if err != nil {
		t.Fatalf("ListPolicies failed: %v", err)
	}

	if len(listResp.GetPolicies()) == 0 {
		t.Skip("No policies configured, skipping GetPolicy test")
	}

	policyID := listResp.GetPolicies()[0].GetId()

	resp, err := client.Client().GetPolicy(ctx, &platonpb.GetPolicyRequest{
		Id: policyID,
	})
	if err != nil {
		t.Fatalf("GetPolicy failed: %v", err)
	}

	t.Logf("Retrieved Policy:")
	t.Logf("  ID: %s", resp.GetId())
	t.Logf("  Name: %s", resp.GetName())
	t.Logf("  Type: %s", resp.GetType().String())
	t.Logf("  Enabled: %v", resp.GetEnabled())
	t.Logf("  Priority: %d", resp.GetPriority())
	t.Logf("  Rules: %d", len(resp.GetRules()))

	if resp.GetId() != policyID {
		t.Errorf("Expected ID '%s', got '%s'", policyID, resp.GetId())
	}
}

// TestPlatonTestPolicy tests the policy testing functionality
func TestPlatonTestPolicy(t *testing.T) {
	client, err := NewPlatonTestClient()
	if err != nil {
		t.Fatalf("Failed to create Platon client: %v", err)
	}
	defer client.Close()

	ctx, cancel := client.ContextWithTimeout(30 * time.Second)
	defer cancel()

	// Create a policy for testing
	policy := &platonpb.PolicyInfo{
		Id:          "test-policy-temp",
		Name:        "Test Policy",
		Description: "Temporary policy for testing",
		Type:        platonpb.PolicyType_POLICY_TYPE_CONTENT,
		Enabled:     true,
		Priority:    100,
		Rules: []*platonpb.PolicyRule{
			{
				Id:            "block-rule",
				Pattern:       "forbidden",
				Action:        platonpb.PolicyAction_POLICY_ACTION_BLOCK,
				Message:       "Forbidden content detected",
				CaseSensitive: false,
			},
		},
	}

	testCases := []struct {
		name     string
		testText string
		wantBlock bool
	}{
		{
			name:      "Clean text",
			testText:  "This is a normal message",
			wantBlock: false,
		},
		{
			name:      "Forbidden text",
			testText:  "This contains forbidden content",
			wantBlock: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := &platonpb.TestPolicyRequest{
				Policy:   policy,
				TestText: tc.testText,
			}

			resp, err := client.Client().TestPolicy(ctx, req)
			if err != nil {
				t.Fatalf("TestPolicy failed: %v", err)
			}

			t.Logf("Test Policy Result for '%s':", tc.name)
			t.Logf("  Decision: %s", resp.GetDecision().String())
			t.Logf("  Violations: %d", len(resp.GetViolations()))
			t.Logf("  Modified Text: %s", resp.GetModifiedText())
			t.Logf("  Reason: %s", resp.GetReason())
			t.Logf("  Duration: %d ms", resp.GetDurationMs())

			isBlocked := resp.GetDecision() == platonpb.PolicyDecision_POLICY_DECISION_BLOCK
			if isBlocked != tc.wantBlock {
				t.Errorf("Expected blocked=%v, got %v", tc.wantBlock, isBlocked)
			}
		})
	}
}

// ============================================================================
// Processing Tests
// ============================================================================

// TestPlatonProcess tests the main processing functionality
func TestPlatonProcess(t *testing.T) {
	client, err := NewPlatonTestClient()
	if err != nil {
		t.Fatalf("Failed to create Platon client: %v", err)
	}
	defer client.Close()

	ctx, cancel := client.ContextWithTimeout(30 * time.Second)
	defer cancel()

	req := &platonpb.ProcessRequest{
		RequestId:  "test-req-1",
		PipelineId: "default",
		Prompt:     "What is the capital of France?",
		Metadata: map[string]string{
			"source": "integration-test",
		},
		Options: &platonpb.ProcessOptions{
			Debug:   true,
			DryRun:  false,
		},
	}

	resp, err := client.Client().Process(ctx, req)
	if err != nil {
		t.Fatalf("Process failed: %v", err)
	}

	t.Logf("Process Result:")
	t.Logf("  Request ID: %s", resp.GetRequestId())
	t.Logf("  Blocked: %v", resp.GetBlocked())
	t.Logf("  Block Reason: %s", resp.GetBlockReason())
	t.Logf("  Modified: %v", resp.GetModified())
	t.Logf("  Processed Prompt: %s", truncate(resp.GetProcessedPrompt(), 100))
	t.Logf("  Duration: %d ms", resp.GetDurationMs())
	t.Logf("  Audit Entries: %d", len(resp.GetAuditLog()))

	for _, entry := range resp.GetAuditLog() {
		t.Logf("    - %s (%s): modified=%v, duration=%dms",
			entry.GetHandler(), entry.GetPhase(), entry.GetModified(), entry.GetDurationMs())
	}

	if resp.GetRequestId() != req.RequestId {
		t.Errorf("Expected request ID '%s', got '%s'", req.RequestId, resp.GetRequestId())
	}
}

// TestPlatonProcessPre tests pre-processing only
func TestPlatonProcessPre(t *testing.T) {
	client, err := NewPlatonTestClient()
	if err != nil {
		t.Fatalf("Failed to create Platon client: %v", err)
	}
	defer client.Close()

	ctx, cancel := client.ContextWithTimeout(30 * time.Second)
	defer cancel()

	req := &platonpb.ProcessRequest{
		RequestId:  "test-pre-1",
		PipelineId: "default",
		Prompt:     "Translate this to German: Hello world",
		Options: &platonpb.ProcessOptions{
			Debug: true,
		},
	}

	resp, err := client.Client().ProcessPre(ctx, req)
	if err != nil {
		t.Fatalf("ProcessPre failed: %v", err)
	}

	t.Logf("Pre-Process Result:")
	t.Logf("  Request ID: %s", resp.GetRequestId())
	t.Logf("  Blocked: %v", resp.GetBlocked())
	t.Logf("  Modified: %v", resp.GetModified())
	t.Logf("  Processed Prompt: %s", truncate(resp.GetProcessedPrompt(), 100))
	t.Logf("  Duration: %d ms", resp.GetDurationMs())
}

// TestPlatonProcessPost tests post-processing only
func TestPlatonProcessPost(t *testing.T) {
	client, err := NewPlatonTestClient()
	if err != nil {
		t.Fatalf("Failed to create Platon client: %v", err)
	}
	defer client.Close()

	ctx, cancel := client.ContextWithTimeout(30 * time.Second)
	defer cancel()

	req := &platonpb.ProcessRequest{
		RequestId:  "test-post-1",
		PipelineId: "default",
		Prompt:     "Original prompt",
		Response:   "This is the LLM response that needs post-processing",
		Options: &platonpb.ProcessOptions{
			Debug: true,
		},
	}

	resp, err := client.Client().ProcessPost(ctx, req)
	if err != nil {
		t.Fatalf("ProcessPost failed: %v", err)
	}

	t.Logf("Post-Process Result:")
	t.Logf("  Request ID: %s", resp.GetRequestId())
	t.Logf("  Blocked: %v", resp.GetBlocked())
	t.Logf("  Modified: %v", resp.GetModified())
	t.Logf("  Processed Response: %s", truncate(resp.GetProcessedResponse(), 100))
	t.Logf("  Duration: %d ms", resp.GetDurationMs())
}

// TestPlatonProcessDryRun tests dry-run processing
func TestPlatonProcessDryRun(t *testing.T) {
	client, err := NewPlatonTestClient()
	if err != nil {
		t.Fatalf("Failed to create Platon client: %v", err)
	}
	defer client.Close()

	ctx, cancel := client.ContextWithTimeout(30 * time.Second)
	defer cancel()

	req := &platonpb.ProcessRequest{
		RequestId:  "test-dryrun-1",
		PipelineId: "default",
		Prompt:     "Test prompt for dry run",
		Options: &platonpb.ProcessOptions{
			DryRun: true,
			Debug:  true,
		},
	}

	resp, err := client.Client().Process(ctx, req)
	if err != nil {
		t.Fatalf("Process (dry-run) failed: %v", err)
	}

	t.Logf("Dry-Run Process Result:")
	t.Logf("  Request ID: %s", resp.GetRequestId())
	t.Logf("  Blocked: %v", resp.GetBlocked())
	t.Logf("  Modified: %v", resp.GetModified())
	t.Logf("  Audit Entries: %d", len(resp.GetAuditLog()))
	t.Logf("  Duration: %d ms", resp.GetDurationMs())
}

// ============================================================================
// Test Suite
// ============================================================================

// RunPlatonTestSuite runs all Platon tests
func RunPlatonTestSuite(t *testing.T) *TestSuite {
	suite := NewTestSuite("platon")

	tests := []struct {
		name string
		fn   func(*testing.T)
	}{
		{"HealthCheck", TestPlatonHealthCheck},
		{"ListHandlers", TestPlatonListHandlers},
		{"RegisterHandler", TestPlatonRegisterHandler},
		{"GetHandler", TestPlatonGetHandler},
		{"ListPipelines", TestPlatonListPipelines},
		{"CreatePipeline", TestPlatonCreatePipeline},
		{"GetPipeline", TestPlatonGetPipeline},
		{"UpdatePipeline", TestPlatonUpdatePipeline},
		{"ListPolicies", TestPlatonListPolicies},
		{"CreatePolicy", TestPlatonCreatePolicy},
		{"GetPolicy", TestPlatonGetPolicy},
		{"TestPolicy", TestPlatonTestPolicy},
		{"Process", TestPlatonProcess},
		{"ProcessPre", TestPlatonProcessPre},
		{"ProcessPost", TestPlatonProcessPost},
		{"ProcessDryRun", TestPlatonProcessDryRun},
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
