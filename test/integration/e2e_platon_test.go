// ============================================================================
// meinDENKWERK (mDW) - Lokale KI-Plattform
// ============================================================================
//
// Package:     integration
// Description: End-to-End tests for Platon Pipeline Processing Service
// Author:      Mike Stoffels with Claude
// Created:     2025-12-09
// License:     MIT
// ============================================================================

package integration

import (
	"context"
	"fmt"
	"testing"
	"time"

	commonpb "github.com/msto63/mDW/api/gen/common"
	platonpb "github.com/msto63/mDW/api/gen/platon"
	turingpb "github.com/msto63/mDW/api/gen/turing"
)

// ============================================================================
// Platon Pipeline E2E Tests
// ============================================================================

// TestE2E_PlatonPipeline_BasicProcessing tests basic pipeline processing
func TestE2E_PlatonPipeline_BasicProcessing(t *testing.T) {
	cfg := getTestConfig()
	skipIfServiceUnavailable(t, cfg.PlatonAddr, "Platon")
	logTestStart(t, "E2E", "Platon Basic Processing")

	conn := dialGRPC(t, cfg.PlatonAddr)
	client := platonpb.NewPlatonServiceClient(conn)

	ctx, cancel := testContext(t, 30*time.Second)
	defer cancel()

	// Test basic processing
	req := &platonpb.ProcessRequest{
		RequestId:  fmt.Sprintf("e2e-basic-%d", time.Now().UnixNano()),
		PipelineId: "default",
		Prompt:     "What is the capital of France?",
		Metadata: map[string]string{
			"source": "e2e_test",
			"type":   "basic",
		},
		Options: &platonpb.ProcessOptions{
			Debug:  true,
			DryRun: false,
		},
	}

	resp, err := client.Process(ctx, req)
	requireNoError(t, err, "Process failed")

	t.Logf("Process Result:")
	t.Logf("  Request ID: %s", resp.GetRequestId())
	t.Logf("  Blocked: %v", resp.GetBlocked())
	t.Logf("  Modified: %v", resp.GetModified())
	t.Logf("  Duration: %dms", resp.GetDurationMs())
	t.Logf("  Audit Entries: %d", len(resp.GetAuditLog()))

	requireEqual(t, req.RequestId, resp.GetRequestId(), "Request ID should match")

	if resp.GetProcessedPrompt() == "" {
		t.Error("Processed prompt should not be empty")
	}

	t.Log("E2E Platon Basic Processing completed successfully!")
}

// TestE2E_PlatonPipeline_PIIDetection tests PII detection capabilities
func TestE2E_PlatonPipeline_PIIDetection(t *testing.T) {
	cfg := getTestConfig()
	skipIfServiceUnavailable(t, cfg.PlatonAddr, "Platon")
	logTestStart(t, "E2E", "Platon PII Detection")

	conn := dialGRPC(t, cfg.PlatonAddr)
	client := platonpb.NewPlatonServiceClient(conn)

	ctx, cancel := testContext(t, 30*time.Second)
	defer cancel()

	testCases := []struct {
		name          string
		prompt        string
		expectPII     bool
		piiType       string
	}{
		{
			name:      "Email Detection",
			prompt:    "Contact me at john.doe@example.com for more information.",
			expectPII: true,
			piiType:   "email",
		},
		{
			name:      "Phone Detection",
			prompt:    "Call me at +49 170 1234567 for assistance.",
			expectPII: true,
			piiType:   "phone",
		},
		{
			name:      "IBAN Detection",
			prompt:    "Transfer to DE89370400440532013000 please.",
			expectPII: true,
			piiType:   "iban",
		},
		{
			name:      "Clean Text",
			prompt:    "This is a normal message without any personal data.",
			expectPII: false,
			piiType:   "",
		},
	}

	// Create a PII policy for testing
	piiPolicy := &platonpb.PolicyInfo{
		Id:          "test-pii-e2e",
		Name:        "E2E PII Test Policy",
		Description: "Policy for E2E PII detection tests",
		Type:        platonpb.PolicyType_POLICY_TYPE_PII,
		Enabled:     true,
		Priority:    10,
		Rules: []*platonpb.PolicyRule{
			{
				Id:            "email",
				Pattern:       `\b[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Z|a-z]{2,}\b`,
				Action:        platonpb.PolicyAction_POLICY_ACTION_REDACT,
				Message:       "Email address detected",
				CaseSensitive: false,
			},
			{
				Id:            "phone_de",
				Pattern:       `(\+49|0049|0)\s*[1-9][\d\s]{1,20}`,
				Action:        platonpb.PolicyAction_POLICY_ACTION_REDACT,
				Message:       "Phone number detected",
				CaseSensitive: false,
			},
			{
				Id:            "iban",
				Pattern:       `\b[A-Z]{2}\d{2}[A-Z0-9]{4}\d{7}([A-Z0-9]?){0,16}\b`,
				Action:        platonpb.PolicyAction_POLICY_ACTION_REDACT,
				Message:       "IBAN detected",
				CaseSensitive: false,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			testReq := &platonpb.TestPolicyRequest{
				Policy:   piiPolicy,
				TestText: tc.prompt,
			}

			resp, err := client.TestPolicy(ctx, testReq)
			requireNoError(t, err, "TestPolicy failed")

			hasPII := len(resp.GetViolations()) > 0

			t.Logf("  Test '%s':", tc.name)
			t.Logf("    Decision: %s", resp.GetDecision().String())
			t.Logf("    Violations: %d", len(resp.GetViolations()))
			t.Logf("    Duration: %dms", resp.GetDurationMs())

			if hasPII != tc.expectPII {
				t.Errorf("Expected PII detected=%v, got %v", tc.expectPII, hasPII)
			}

			if tc.expectPII && resp.GetModifiedText() == tc.prompt {
				t.Logf("    Warning: Text was not modified despite PII detection")
			}
		})
	}

	t.Log("E2E Platon PII Detection completed successfully!")
}

// TestE2E_PlatonPipeline_ContentModeration tests content moderation
func TestE2E_PlatonPipeline_ContentModeration(t *testing.T) {
	cfg := getTestConfig()
	skipIfServiceUnavailable(t, cfg.PlatonAddr, "Platon")
	logTestStart(t, "E2E", "Platon Content Moderation")

	conn := dialGRPC(t, cfg.PlatonAddr)
	client := platonpb.NewPlatonServiceClient(conn)

	ctx, cancel := testContext(t, 30*time.Second)
	defer cancel()

	// Create a content moderation policy
	contentPolicy := &platonpb.PolicyInfo{
		Id:          "test-content-e2e",
		Name:        "E2E Content Moderation Policy",
		Description: "Policy for E2E content moderation tests",
		Type:        platonpb.PolicyType_POLICY_TYPE_CONTENT,
		Enabled:     true,
		Priority:    20,
		Rules: []*platonpb.PolicyRule{
			{
				Id:            "block-forbidden",
				Pattern:       `(?i)forbidden|prohibited|banned`,
				Action:        platonpb.PolicyAction_POLICY_ACTION_BLOCK,
				Message:       "Forbidden content detected",
				CaseSensitive: false,
			},
			{
				Id:            "warn-sensitive",
				Pattern:       `(?i)sensitive|confidential`,
				Action:        platonpb.PolicyAction_POLICY_ACTION_WARN,
				Message:       "Sensitive content detected",
				CaseSensitive: false,
			},
		},
	}

	testCases := []struct {
		name           string
		text           string
		expectedAction platonpb.PolicyDecision
	}{
		{
			name:           "Clean content",
			text:           "This is a normal message.",
			expectedAction: platonpb.PolicyDecision_POLICY_DECISION_ALLOW,
		},
		{
			name:           "Blocked content",
			text:           "This contains forbidden words.",
			expectedAction: platonpb.PolicyDecision_POLICY_DECISION_BLOCK,
		},
		{
			name:           "Escalated content",
			text:           "This is sensitive information.",
			expectedAction: platonpb.PolicyDecision_POLICY_DECISION_ESCALATE,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			testReq := &platonpb.TestPolicyRequest{
				Policy:   contentPolicy,
				TestText: tc.text,
			}

			resp, err := client.TestPolicy(ctx, testReq)
			requireNoError(t, err, "TestPolicy failed")

			t.Logf("  Test '%s':", tc.name)
			t.Logf("    Input: %s", tc.text)
			t.Logf("    Decision: %s", resp.GetDecision().String())
			t.Logf("    Reason: %s", resp.GetReason())

			if resp.GetDecision() != tc.expectedAction {
				t.Errorf("Expected decision %s, got %s", tc.expectedAction.String(), resp.GetDecision().String())
			}
		})
	}

	t.Log("E2E Platon Content Moderation completed successfully!")
}

// TestE2E_PlatonPipeline_HandlerChain tests the handler chain execution
func TestE2E_PlatonPipeline_HandlerChain(t *testing.T) {
	cfg := getTestConfig()
	skipIfServiceUnavailable(t, cfg.PlatonAddr, "Platon")
	logTestStart(t, "E2E", "Platon Handler Chain")

	conn := dialGRPC(t, cfg.PlatonAddr)
	client := platonpb.NewPlatonServiceClient(conn)

	ctx, cancel := testContext(t, 60*time.Second)
	defer cancel()

	// List available handlers
	handlersResp, err := client.ListHandlers(ctx, &commonpb.Empty{})
	requireNoError(t, err, "ListHandlers failed")

	t.Logf("Available handlers: %d", len(handlersResp.GetHandlers()))
	for _, h := range handlersResp.GetHandlers() {
		t.Logf("  - %s (type: %s, priority: %d, enabled: %v)",
			h.GetName(), h.GetType().String(), h.GetPriority(), h.GetEnabled())
	}

	// Process with debug to see handler chain execution
	processReq := &platonpb.ProcessRequest{
		RequestId:  fmt.Sprintf("e2e-chain-%d", time.Now().UnixNano()),
		PipelineId: "default",
		Prompt:     "Test message for handler chain validation",
		Options: &platonpb.ProcessOptions{
			Debug:  true,
			DryRun: false,
		},
	}

	processResp, err := client.Process(ctx, processReq)
	requireNoError(t, err, "Process failed")

	t.Logf("Handler Chain Execution:")
	t.Logf("  Total Duration: %dms", processResp.GetDurationMs())
	t.Logf("  Audit Entries: %d", len(processResp.GetAuditLog()))

	for i, entry := range processResp.GetAuditLog() {
		t.Logf("  %d. Handler: %s", i+1, entry.GetHandler())
		t.Logf("     Phase: %s", entry.GetPhase())
		t.Logf("     Modified: %v", entry.GetModified())
		t.Logf("     Duration: %dms", entry.GetDurationMs())
		if entry.GetError() != "" {
			t.Logf("     Error: %s", entry.GetError())
		}
	}

	t.Log("E2E Platon Handler Chain completed successfully!")
}

// TestE2E_PlatonPipeline_PipelineManagement tests pipeline CRUD operations
func TestE2E_PlatonPipeline_PipelineManagement(t *testing.T) {
	cfg := getTestConfig()
	skipIfServiceUnavailable(t, cfg.PlatonAddr, "Platon")
	logTestStart(t, "E2E", "Platon Pipeline Management")

	conn := dialGRPC(t, cfg.PlatonAddr)
	client := platonpb.NewPlatonServiceClient(conn)

	ctx, cancel := testContext(t, 60*time.Second)
	defer cancel()

	pipelineID := fmt.Sprintf("e2e-pipeline-%d", time.Now().UnixNano())

	// Step 1: Create pipeline
	t.Log("Step 1: Creating pipeline...")
	createResp, err := client.CreatePipeline(ctx, &platonpb.CreatePipelineRequest{
		Id:          pipelineID,
		Name:        "E2E Test Pipeline",
		Description: "Pipeline created for E2E testing",
		Enabled:     true,
		Config: map[string]string{
			"test_mode": "true",
		},
	})
	requireNoError(t, err, "CreatePipeline failed")
	t.Logf("  Created: %s", createResp.GetId())

	// Step 2: Get pipeline
	t.Log("Step 2: Getting pipeline...")
	getResp, err := client.GetPipeline(ctx, &platonpb.GetPipelineRequest{
		Id: pipelineID,
	})
	requireNoError(t, err, "GetPipeline failed")
	requireEqual(t, pipelineID, getResp.GetId(), "Pipeline ID should match")
	t.Logf("  Retrieved: %s - %s", getResp.GetId(), getResp.GetName())

	// Step 3: Update pipeline
	t.Log("Step 3: Updating pipeline...")
	updateResp, err := client.UpdatePipeline(ctx, &platonpb.UpdatePipelineRequest{
		Id:          pipelineID,
		Name:        "E2E Test Pipeline (Updated)",
		Description: "Updated description",
		Enabled:     false,
	})
	requireNoError(t, err, "UpdatePipeline failed")
	requireEqual(t, "E2E Test Pipeline (Updated)", updateResp.GetName(), "Name should be updated")
	t.Logf("  Updated: %s", updateResp.GetName())

	// Step 4: List pipelines
	t.Log("Step 4: Listing pipelines...")
	listResp, err := client.ListPipelines(ctx, &commonpb.Empty{})
	requireNoError(t, err, "ListPipelines failed")

	found := false
	for _, p := range listResp.GetPipelines() {
		if p.GetId() == pipelineID {
			found = true
			break
		}
	}
	requireTrue(t, found, "Created pipeline should be in list")
	t.Logf("  Found pipeline in list of %d pipelines", len(listResp.GetPipelines()))

	// Step 5: Delete pipeline
	t.Log("Step 5: Deleting pipeline...")
	_, err = client.DeletePipeline(ctx, &platonpb.DeletePipelineRequest{
		Id: pipelineID,
	})
	requireNoError(t, err, "DeletePipeline failed")
	t.Logf("  Deleted: %s", pipelineID)

	// Verify deletion
	_, err = client.GetPipeline(ctx, &platonpb.GetPipelineRequest{
		Id: pipelineID,
	})
	if err == nil {
		t.Error("Pipeline should not exist after deletion")
	}

	t.Log("E2E Platon Pipeline Management completed successfully!")
}

// TestE2E_PlatonPipeline_PolicyManagement tests policy CRUD operations
func TestE2E_PlatonPipeline_PolicyManagement(t *testing.T) {
	cfg := getTestConfig()
	skipIfServiceUnavailable(t, cfg.PlatonAddr, "Platon")
	logTestStart(t, "E2E", "Platon Policy Management")

	conn := dialGRPC(t, cfg.PlatonAddr)
	client := platonpb.NewPlatonServiceClient(conn)

	ctx, cancel := testContext(t, 60*time.Second)
	defer cancel()

	policyID := fmt.Sprintf("e2e-policy-%d", time.Now().UnixNano())

	// Step 1: Create policy
	t.Log("Step 1: Creating policy...")
	createResp, err := client.CreatePolicy(ctx, &platonpb.CreatePolicyRequest{
		Id:          policyID,
		Name:        "E2E Test Policy",
		Description: "Policy created for E2E testing",
		Type:        platonpb.PolicyType_POLICY_TYPE_CONTENT,
		Enabled:     true,
		Priority:    100,
		Rules: []*platonpb.PolicyRule{
			{
				Id:      "test-rule",
				Pattern: "test-pattern",
				Action:  platonpb.PolicyAction_POLICY_ACTION_LOG,
				Message: "Test pattern matched",
			},
		},
	})
	requireNoError(t, err, "CreatePolicy failed")
	t.Logf("  Created: %s", createResp.GetId())

	// Step 2: Get policy
	t.Log("Step 2: Getting policy...")
	getResp, err := client.GetPolicy(ctx, &platonpb.GetPolicyRequest{
		Id: policyID,
	})
	requireNoError(t, err, "GetPolicy failed")
	requireEqual(t, policyID, getResp.GetId(), "Policy ID should match")
	t.Logf("  Retrieved: %s - %s", getResp.GetId(), getResp.GetName())

	// Step 3: Update policy
	t.Log("Step 3: Updating policy...")
	updateResp, err := client.UpdatePolicy(ctx, &platonpb.UpdatePolicyRequest{
		Id:          policyID,
		Name:        "E2E Test Policy (Updated)",
		Description: "Updated description",
		Enabled:     false,
		Priority:    50,
	})
	requireNoError(t, err, "UpdatePolicy failed")
	requireEqual(t, "E2E Test Policy (Updated)", updateResp.GetName(), "Name should be updated")
	t.Logf("  Updated: %s", updateResp.GetName())

	// Step 4: List policies
	t.Log("Step 4: Listing policies...")
	listResp, err := client.ListPolicies(ctx, &commonpb.Empty{})
	requireNoError(t, err, "ListPolicies failed")

	found := false
	for _, p := range listResp.GetPolicies() {
		if p.GetId() == policyID {
			found = true
			break
		}
	}
	requireTrue(t, found, "Created policy should be in list")
	t.Logf("  Found policy in list of %d policies", len(listResp.GetPolicies()))

	// Step 5: Delete policy
	t.Log("Step 5: Deleting policy...")
	_, err = client.DeletePolicy(ctx, &platonpb.DeletePolicyRequest{
		Id: policyID,
	})
	requireNoError(t, err, "DeletePolicy failed")
	t.Logf("  Deleted: %s", policyID)

	// Verify deletion
	_, err = client.GetPolicy(ctx, &platonpb.GetPolicyRequest{
		Id: policyID,
	})
	if err == nil {
		t.Error("Policy should not exist after deletion")
	}

	t.Log("E2E Platon Policy Management completed successfully!")
}

// TestE2E_PlatonPipeline_PrePostProcessing tests separate pre and post processing
func TestE2E_PlatonPipeline_PrePostProcessing(t *testing.T) {
	cfg := getTestConfig()
	skipIfServiceUnavailable(t, cfg.PlatonAddr, "Platon")
	logTestStart(t, "E2E", "Platon Pre/Post Processing")

	conn := dialGRPC(t, cfg.PlatonAddr)
	client := platonpb.NewPlatonServiceClient(conn)

	ctx, cancel := testContext(t, 60*time.Second)
	defer cancel()

	prompt := "Original user prompt with email test@example.com"
	response := "LLM generated response with IBAN DE89370400440532013000"

	// Test Pre-Processing
	t.Log("Testing Pre-Processing...")
	preReq := &platonpb.ProcessRequest{
		RequestId:  fmt.Sprintf("e2e-pre-%d", time.Now().UnixNano()),
		PipelineId: "default",
		Prompt:     prompt,
		Options: &platonpb.ProcessOptions{
			Debug: true,
		},
	}

	preResp, err := client.ProcessPre(ctx, preReq)
	requireNoError(t, err, "ProcessPre failed")

	t.Logf("  Pre-Processing Result:")
	t.Logf("    Original: %s", prompt)
	t.Logf("    Processed: %s", preResp.GetProcessedPrompt())
	t.Logf("    Modified: %v", preResp.GetModified())
	t.Logf("    Duration: %dms", preResp.GetDurationMs())

	// Test Post-Processing
	t.Log("Testing Post-Processing...")
	postReq := &platonpb.ProcessRequest{
		RequestId:  fmt.Sprintf("e2e-post-%d", time.Now().UnixNano()),
		PipelineId: "default",
		Prompt:     prompt,
		Response:   response,
		Options: &platonpb.ProcessOptions{
			Debug: true,
		},
	}

	postResp, err := client.ProcessPost(ctx, postReq)
	requireNoError(t, err, "ProcessPost failed")

	t.Logf("  Post-Processing Result:")
	t.Logf("    Original Response: %s", response)
	t.Logf("    Processed Response: %s", postResp.GetProcessedResponse())
	t.Logf("    Modified: %v", postResp.GetModified())
	t.Logf("    Duration: %dms", postResp.GetDurationMs())

	t.Log("E2E Platon Pre/Post Processing completed successfully!")
}

// TestE2E_PlatonPipeline_DryRun tests dry-run mode
func TestE2E_PlatonPipeline_DryRun(t *testing.T) {
	cfg := getTestConfig()
	skipIfServiceUnavailable(t, cfg.PlatonAddr, "Platon")
	logTestStart(t, "E2E", "Platon Dry Run")

	conn := dialGRPC(t, cfg.PlatonAddr)
	client := platonpb.NewPlatonServiceClient(conn)

	ctx, cancel := testContext(t, 30*time.Second)
	defer cancel()

	// Test with dry-run enabled
	req := &platonpb.ProcessRequest{
		RequestId:  fmt.Sprintf("e2e-dryrun-%d", time.Now().UnixNano()),
		PipelineId: "default",
		Prompt:     "Test prompt for dry run validation",
		Options: &platonpb.ProcessOptions{
			DryRun: true,
			Debug:  true,
		},
	}

	resp, err := client.Process(ctx, req)
	requireNoError(t, err, "Process (dry-run) failed")

	t.Logf("Dry-Run Result:")
	t.Logf("  Request ID: %s", resp.GetRequestId())
	t.Logf("  Audit Entries: %d", len(resp.GetAuditLog()))
	t.Logf("  Duration: %dms", resp.GetDurationMs())

	// In dry-run mode, we should still get audit entries
	if len(resp.GetAuditLog()) == 0 {
		t.Logf("  Warning: No audit entries in dry-run mode")
	}

	t.Log("E2E Platon Dry Run completed successfully!")
}

// ============================================================================
// Full Pipeline Integration Tests
// ============================================================================

// TestE2E_FullPipeline_ChatWithPlaton tests complete chat flow with Platon processing
func TestE2E_FullPipeline_ChatWithPlaton(t *testing.T) {
	cfg := getTestConfig()
	skipIfServiceUnavailable(t, cfg.PlatonAddr, "Platon")
	skipIfServiceUnavailable(t, cfg.TuringAddr, "Turing")
	skipIfServiceUnavailable(t, cfg.OllamaAddr, "Ollama")
	logTestStart(t, "E2E", "Full Chat Pipeline with Platon")

	// Setup clients
	platonConn := dialGRPC(t, cfg.PlatonAddr)
	platonClient := platonpb.NewPlatonServiceClient(platonConn)

	turingConn := dialGRPC(t, cfg.TuringAddr)
	turingClient := turingpb.NewTuringServiceClient(turingConn)

	ctx, cancel := testContext(t, 180*time.Second)
	defer cancel()

	userPrompt := "What is the capital of France? My email is user@example.com for follow-up."

	// Step 1: Pre-process with Platon
	t.Log("Step 1: Pre-processing user prompt with Platon...")
	preReq := &platonpb.ProcessRequest{
		RequestId:  fmt.Sprintf("e2e-chat-%d", time.Now().UnixNano()),
		PipelineId: "default",
		Prompt:     userPrompt,
		Options: &platonpb.ProcessOptions{
			Debug: true,
		},
	}

	preResp, err := platonClient.ProcessPre(ctx, preReq)
	requireNoError(t, err, "Platon ProcessPre failed")

	processedPrompt := preResp.GetProcessedPrompt()
	if processedPrompt == "" {
		processedPrompt = userPrompt
	}

	t.Logf("  Original: %s", userPrompt)
	t.Logf("  Processed: %s", processedPrompt)
	t.Logf("  Modified: %v", preResp.GetModified())

	// Step 2: Check if blocked
	if preResp.GetBlocked() {
		t.Logf("  Request blocked: %s", preResp.GetBlockReason())
		t.Log("E2E Full Chat Pipeline completed (blocked by pre-processing)")
		return
	}

	// Step 3: Send to LLM via Turing
	t.Log("Step 2: Sending processed prompt to Turing LLM...")
	chatReq := &turingpb.ChatRequest{
		Messages: []*turingpb.Message{
			{Role: "user", Content: processedPrompt},
		},
		Model: "qwen2.5:7b",
	}

	chatResp, err := turingClient.Chat(ctx, chatReq)
	requireNoError(t, err, "Turing Chat failed")

	llmResponse := chatResp.GetContent()
	t.Logf("  LLM Response: %s", truncateString(llmResponse, 200))

	// Step 4: Post-process LLM response with Platon
	t.Log("Step 3: Post-processing LLM response with Platon...")
	postReq := &platonpb.ProcessRequest{
		RequestId:  preReq.RequestId,
		PipelineId: "default",
		Prompt:     processedPrompt,
		Response:   llmResponse,
		Options: &platonpb.ProcessOptions{
			Debug: true,
		},
	}

	postResp, err := platonClient.ProcessPost(ctx, postReq)
	requireNoError(t, err, "Platon ProcessPost failed")

	finalResponse := postResp.GetProcessedResponse()
	if finalResponse == "" {
		finalResponse = llmResponse
	}

	t.Logf("  Final Response: %s", truncateString(finalResponse, 200))
	t.Logf("  Modified: %v", postResp.GetModified())

	t.Log("E2E Full Chat Pipeline with Platon completed successfully!")
}

// TestE2E_FullPipeline_ServiceCommunication tests all services working together
func TestE2E_FullPipeline_ServiceCommunication(t *testing.T) {
	cfg := getTestConfig()
	logTestStart(t, "E2E", "Full Service Communication")

	services := []struct {
		name string
		addr string
	}{
		{"Platon", cfg.PlatonAddr},
		{"Russell", cfg.RussellAddr},
		{"Turing", cfg.TuringAddr},
		{"Hypatia", cfg.HypatiaAddr},
		{"Leibniz", cfg.LeibnizAddr},
		{"Babbage", cfg.BabbageAddr},
		{"Bayes", cfg.BayesAddr},
	}

	t.Log("Checking service availability:")
	availableCount := 0
	for _, svc := range services {
		available := isServiceAvailable(svc.addr)
		status := "available"
		if !available {
			status = "unavailable"
		} else {
			availableCount++
		}
		t.Logf("  %s at %s: %s", svc.name, svc.addr, status)
	}

	t.Logf("Services available: %d/%d", availableCount, len(services))

	if availableCount < 3 {
		t.Skip("Skipping: Need at least 3 services running")
	}

	// Test Platon health if available
	if isServiceAvailable(cfg.PlatonAddr) {
		conn := dialGRPC(t, cfg.PlatonAddr)
		client := platonpb.NewPlatonServiceClient(conn)

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		healthResp, err := client.HealthCheck(ctx, &commonpb.HealthCheckRequest{})
		if err == nil {
			t.Logf("  Platon Health: %s (version: %s)", healthResp.GetStatus(), healthResp.GetVersion())
		}
	}

	t.Log("E2E Full Service Communication check completed!")
}

// ============================================================================
// Helper Functions
// ============================================================================

// truncateString truncates a string to maxLen characters
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
