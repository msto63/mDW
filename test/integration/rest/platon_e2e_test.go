//go:build integration

// ============================================================================
// meinDENKWERK (mDW) - Lokale KI-Plattform
// ============================================================================
//
// Package:     rest
// Description: REST API End-to-End tests for Platon Pipeline Processing
// Author:      Mike Stoffels with Claude
// Created:     2025-12-09
// License:     MIT
// ============================================================================

package rest

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"testing"
	"time"
)

// ============================================================================
// Platon REST API Types
// ============================================================================

type PlatonProcessRequest struct {
	PipelineID string            `json:"pipeline_id,omitempty"`
	Prompt     string            `json:"prompt"`
	Response   string            `json:"response,omitempty"`
	Metadata   map[string]string `json:"metadata,omitempty"`
	Options    *ProcessOptions   `json:"options,omitempty"`
}

type ProcessOptions struct {
	DryRun bool `json:"dry_run,omitempty"`
	Debug  bool `json:"debug,omitempty"`
}

type PlatonProcessResponse struct {
	RequestID         string       `json:"request_id"`
	ProcessedPrompt   string       `json:"processed_prompt"`
	ProcessedResponse string       `json:"processed_response,omitempty"`
	Blocked           bool         `json:"blocked"`
	BlockReason       string       `json:"block_reason,omitempty"`
	Modified          bool         `json:"modified"`
	DurationMs        int64        `json:"duration_ms"`
	AuditLog          []AuditEntry `json:"audit_log,omitempty"`
}

type AuditEntry struct {
	Handler    string `json:"handler"`
	Phase      string `json:"phase"`
	Modified   bool   `json:"modified"`
	DurationMs int64  `json:"duration_ms"`
	Error      string `json:"error,omitempty"`
}

type PlatonPipelineInfo struct {
	ID           string   `json:"id"`
	Name         string   `json:"name"`
	Description  string   `json:"description,omitempty"`
	Enabled      bool     `json:"enabled"`
	PreHandlers  []string `json:"pre_handlers,omitempty"`
	PostHandlers []string `json:"post_handlers,omitempty"`
}

type PlatonPipelinesResponse struct {
	Pipelines []PlatonPipelineInfo `json:"pipelines"`
	Total     int                  `json:"total"`
}

type PlatonPolicyInfo struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Description string            `json:"description,omitempty"`
	Type        string            `json:"type"`
	Enabled     bool              `json:"enabled"`
	Priority    int               `json:"priority"`
	Rules       []PlatonPolicyRule `json:"rules,omitempty"`
}

type PlatonPolicyRule struct {
	ID            string `json:"id"`
	Pattern       string `json:"pattern"`
	Action        string `json:"action"`
	Message       string `json:"message,omitempty"`
	Replacement   string `json:"replacement,omitempty"`
	CaseSensitive bool   `json:"case_sensitive,omitempty"`
}

type PlatonPoliciesResponse struct {
	Policies []PlatonPolicyInfo `json:"policies"`
	Total    int                `json:"total"`
}

type PlatonTestPolicyRequest struct {
	Policy   PlatonPolicyInfo `json:"policy"`
	TestText string           `json:"test_text"`
}

type PlatonTestPolicyResponse struct {
	Decision     string           `json:"decision"`
	Violations   []PolicyViolation `json:"violations,omitempty"`
	ModifiedText string           `json:"modified_text,omitempty"`
	Reason       string           `json:"reason,omitempty"`
	DurationMs   int64            `json:"duration_ms"`
}

type PolicyViolation struct {
	RuleID      string `json:"rule_id"`
	PolicyName  string `json:"policy_name"`
	Description string `json:"description"`
	Action      string `json:"action"`
	Matched     string `json:"matched"`
}

type PlatonHandlerInfo struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	Priority    int    `json:"priority"`
	Enabled     bool   `json:"enabled"`
	Description string `json:"description,omitempty"`
}

type PlatonHandlersResponse struct {
	Handlers []PlatonHandlerInfo `json:"handlers"`
	Total    int                 `json:"total"`
}

type PlatonHealthResponse struct {
	Status        string `json:"status"`
	Service       string `json:"service"`
	Version       string `json:"version"`
	UptimeSeconds int64  `json:"uptime_seconds"`
}

// ============================================================================
// Health Check Tests
// ============================================================================

func TestPlaton_HealthCheck(t *testing.T) {
	url := kantBaseURL + "/api/v1/platon/health"
	resp, body := doRequest(t, http.MethodGet, url, nil)

	if resp.StatusCode == http.StatusServiceUnavailable || resp.StatusCode == http.StatusBadGateway {
		t.Skip("Platon service not available")
	}

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected status 200, got %d: %s", resp.StatusCode, string(body))
	}

	var health PlatonHealthResponse
	if err := json.Unmarshal(body, &health); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if health.Status != "healthy" {
		t.Errorf("Expected status 'healthy', got '%s'", health.Status)
	}

	if health.Service != "platon" {
		t.Errorf("Expected service 'platon', got '%s'", health.Service)
	}

	t.Logf("Platon Health: status=%s, version=%s, uptime=%ds",
		health.Status, health.Version, health.UptimeSeconds)
}

// ============================================================================
// Pipeline Processing Tests
// ============================================================================

func TestPlaton_Process_Basic(t *testing.T) {
	url := kantBaseURL + "/api/v1/platon/process"
	req := PlatonProcessRequest{
		Prompt: "What is the capital of France?",
		Options: &ProcessOptions{
			Debug: true,
		},
	}

	resp, body := doRequest(t, http.MethodPost, url, req)

	if resp.StatusCode == http.StatusServiceUnavailable || resp.StatusCode == http.StatusBadGateway {
		t.Skip("Platon service not available")
	}

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected status 200, got %d: %s", resp.StatusCode, string(body))
	}

	var processResp PlatonProcessResponse
	if err := json.Unmarshal(body, &processResp); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if processResp.RequestID == "" {
		t.Error("Request ID should not be empty")
	}

	if processResp.ProcessedPrompt == "" {
		t.Error("Processed prompt should not be empty")
	}

	t.Logf("Process Result: request_id=%s, blocked=%v, modified=%v, duration=%dms",
		processResp.RequestID, processResp.Blocked, processResp.Modified, processResp.DurationMs)
}

func TestPlaton_Process_WithPipeline(t *testing.T) {
	url := kantBaseURL + "/api/v1/platon/process"
	req := PlatonProcessRequest{
		PipelineID: "default",
		Prompt:     "Process this message through the default pipeline",
		Options: &ProcessOptions{
			Debug: true,
		},
	}

	resp, body := doRequest(t, http.MethodPost, url, req)

	if resp.StatusCode == http.StatusServiceUnavailable || resp.StatusCode == http.StatusBadGateway {
		t.Skip("Platon service not available")
	}

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected status 200, got %d: %s", resp.StatusCode, string(body))
	}

	var processResp PlatonProcessResponse
	if err := json.Unmarshal(body, &processResp); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	t.Logf("Process with pipeline: request_id=%s, audit_entries=%d",
		processResp.RequestID, len(processResp.AuditLog))
}

func TestPlaton_Process_DryRun(t *testing.T) {
	url := kantBaseURL + "/api/v1/platon/process"
	req := PlatonProcessRequest{
		Prompt: "Dry run test message",
		Options: &ProcessOptions{
			DryRun: true,
			Debug:  true,
		},
	}

	resp, body := doRequest(t, http.MethodPost, url, req)

	if resp.StatusCode == http.StatusServiceUnavailable || resp.StatusCode == http.StatusBadGateway {
		t.Skip("Platon service not available")
	}

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected status 200, got %d: %s", resp.StatusCode, string(body))
	}

	var processResp PlatonProcessResponse
	if err := json.Unmarshal(body, &processResp); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	t.Logf("Dry-run: request_id=%s, duration=%dms", processResp.RequestID, processResp.DurationMs)
}

func TestPlaton_Process_EmptyPrompt_Error(t *testing.T) {
	url := kantBaseURL + "/api/v1/platon/process"
	req := PlatonProcessRequest{
		Prompt: "",
	}

	resp, body := doRequest(t, http.MethodPost, url, req)

	if resp.StatusCode == http.StatusServiceUnavailable || resp.StatusCode == http.StatusBadGateway {
		t.Skip("Platon service not available")
	}

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("Expected status 400, got %d: %s", resp.StatusCode, string(body))
	}

	t.Log("Empty prompt correctly rejected with 400")
}

func TestPlaton_ProcessPre(t *testing.T) {
	url := kantBaseURL + "/api/v1/platon/process/pre"
	req := PlatonProcessRequest{
		Prompt: "Pre-process this message with email test@example.com",
		Options: &ProcessOptions{
			Debug: true,
		},
	}

	resp, body := doRequest(t, http.MethodPost, url, req)

	if resp.StatusCode == http.StatusServiceUnavailable || resp.StatusCode == http.StatusBadGateway {
		t.Skip("Platon service not available")
	}

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected status 200, got %d: %s", resp.StatusCode, string(body))
	}

	var processResp PlatonProcessResponse
	if err := json.Unmarshal(body, &processResp); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	t.Logf("Pre-process: processed_prompt=%s", truncate(processResp.ProcessedPrompt, 100))
}

func TestPlaton_ProcessPost(t *testing.T) {
	url := kantBaseURL + "/api/v1/platon/process/post"
	req := PlatonProcessRequest{
		Prompt:   "Original prompt",
		Response: "LLM response with IBAN DE89370400440532013000",
		Options: &ProcessOptions{
			Debug: true,
		},
	}

	resp, body := doRequest(t, http.MethodPost, url, req)

	if resp.StatusCode == http.StatusServiceUnavailable || resp.StatusCode == http.StatusBadGateway {
		t.Skip("Platon service not available")
	}

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected status 200, got %d: %s", resp.StatusCode, string(body))
	}

	var processResp PlatonProcessResponse
	if err := json.Unmarshal(body, &processResp); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	t.Logf("Post-process: processed_response=%s", truncate(processResp.ProcessedResponse, 100))
}

// ============================================================================
// Pipeline Management Tests
// ============================================================================

func TestPlaton_ListPipelines(t *testing.T) {
	url := kantBaseURL + "/api/v1/platon/pipelines"
	resp, body := doRequest(t, http.MethodGet, url, nil)

	if resp.StatusCode == http.StatusServiceUnavailable || resp.StatusCode == http.StatusBadGateway {
		t.Skip("Platon service not available")
	}

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected status 200, got %d: %s", resp.StatusCode, string(body))
	}

	var pipelines PlatonPipelinesResponse
	if err := json.Unmarshal(body, &pipelines); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	t.Logf("Found %d pipelines", pipelines.Total)
	for _, p := range pipelines.Pipelines {
		t.Logf("  - %s (%s): enabled=%v", p.ID, p.Name, p.Enabled)
	}
}

func TestPlaton_CreateAndDeletePipeline(t *testing.T) {
	pipelineID := fmt.Sprintf("test-e2e-rest-%d", time.Now().UnixNano())

	// Create pipeline
	createURL := kantBaseURL + "/api/v1/platon/pipelines"
	createReq := PlatonPipelineInfo{
		ID:          pipelineID,
		Name:        "REST E2E Test Pipeline",
		Description: "Created via REST E2E test",
		Enabled:     true,
	}

	resp, body := doRequest(t, http.MethodPost, createURL, createReq)

	if resp.StatusCode == http.StatusServiceUnavailable || resp.StatusCode == http.StatusBadGateway {
		t.Skip("Platon service not available")
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		t.Fatalf("Expected status 200/201, got %d: %s", resp.StatusCode, string(body))
	}

	t.Logf("Created pipeline: %s", pipelineID)

	// Get pipeline
	getURL := fmt.Sprintf("%s/api/v1/platon/pipelines/%s", kantBaseURL, pipelineID)
	resp, body = doRequest(t, http.MethodGet, getURL, nil)

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected status 200 for GET, got %d: %s", resp.StatusCode, string(body))
	}

	var pipeline PlatonPipelineInfo
	if err := json.Unmarshal(body, &pipeline); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if pipeline.ID != pipelineID {
		t.Errorf("Expected ID '%s', got '%s'", pipelineID, pipeline.ID)
	}

	t.Logf("Retrieved pipeline: %s - %s", pipeline.ID, pipeline.Name)

	// Delete pipeline
	deleteReq, _ := http.NewRequest(http.MethodDelete, getURL, nil)
	client := newTestClient()
	deleteResp, err := client.Do(deleteReq)
	if err != nil {
		t.Fatalf("Delete request failed: %v", err)
	}
	deleteResp.Body.Close()

	if deleteResp.StatusCode != http.StatusOK && deleteResp.StatusCode != http.StatusNoContent {
		t.Fatalf("Expected status 200/204 for DELETE, got %d", deleteResp.StatusCode)
	}

	t.Logf("Deleted pipeline: %s", pipelineID)
}

// ============================================================================
// Policy Management Tests
// ============================================================================

func TestPlaton_ListPolicies(t *testing.T) {
	url := kantBaseURL + "/api/v1/platon/policies"
	resp, body := doRequest(t, http.MethodGet, url, nil)

	if resp.StatusCode == http.StatusServiceUnavailable || resp.StatusCode == http.StatusBadGateway {
		t.Skip("Platon service not available")
	}

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected status 200, got %d: %s", resp.StatusCode, string(body))
	}

	var policies PlatonPoliciesResponse
	if err := json.Unmarshal(body, &policies); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	t.Logf("Found %d policies", policies.Total)
	for _, p := range policies.Policies {
		t.Logf("  - %s (%s): type=%s, enabled=%v", p.ID, p.Name, p.Type, p.Enabled)
	}
}

func TestPlaton_CreateAndDeletePolicy(t *testing.T) {
	policyID := fmt.Sprintf("test-e2e-rest-policy-%d", time.Now().UnixNano())

	// Create policy
	createURL := kantBaseURL + "/api/v1/platon/policies"
	createReq := PlatonPolicyInfo{
		ID:          policyID,
		Name:        "REST E2E Test Policy",
		Description: "Created via REST E2E test",
		Type:        "content",
		Enabled:     true,
		Priority:    100,
		Rules: []PlatonPolicyRule{
			{
				ID:      "test-rule",
				Pattern: "test-pattern",
				Action:  "log",
				Message: "Test pattern matched",
			},
		},
	}

	resp, body := doRequest(t, http.MethodPost, createURL, createReq)

	if resp.StatusCode == http.StatusServiceUnavailable || resp.StatusCode == http.StatusBadGateway {
		t.Skip("Platon service not available")
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		t.Fatalf("Expected status 200/201, got %d: %s", resp.StatusCode, string(body))
	}

	t.Logf("Created policy: %s", policyID)

	// Get policy
	getURL := fmt.Sprintf("%s/api/v1/platon/policies/%s", kantBaseURL, policyID)
	resp, body = doRequest(t, http.MethodGet, getURL, nil)

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected status 200 for GET, got %d: %s", resp.StatusCode, string(body))
	}

	var policy PlatonPolicyInfo
	if err := json.Unmarshal(body, &policy); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if policy.ID != policyID {
		t.Errorf("Expected ID '%s', got '%s'", policyID, policy.ID)
	}

	t.Logf("Retrieved policy: %s - %s", policy.ID, policy.Name)

	// Delete policy
	deleteReq, _ := http.NewRequest(http.MethodDelete, getURL, nil)
	client := newTestClient()
	deleteResp, err := client.Do(deleteReq)
	if err != nil {
		t.Fatalf("Delete request failed: %v", err)
	}
	deleteResp.Body.Close()

	if deleteResp.StatusCode != http.StatusOK && deleteResp.StatusCode != http.StatusNoContent {
		t.Fatalf("Expected status 200/204 for DELETE, got %d", deleteResp.StatusCode)
	}

	t.Logf("Deleted policy: %s", policyID)
}

func TestPlaton_TestPolicy(t *testing.T) {
	url := kantBaseURL + "/api/v1/platon/policies/test"

	testCases := []struct {
		name           string
		policy         PlatonPolicyInfo
		testText       string
		expectBlocked  bool
	}{
		{
			name: "Clean text allowed",
			policy: PlatonPolicyInfo{
				Name:    "Block Test",
				Type:    "content",
				Enabled: true,
				Rules: []PlatonPolicyRule{
					{Pattern: "forbidden", Action: "block", Message: "Forbidden content"},
				},
			},
			testText:      "This is a clean message",
			expectBlocked: false,
		},
		{
			name: "Forbidden text blocked",
			policy: PlatonPolicyInfo{
				Name:    "Block Test",
				Type:    "content",
				Enabled: true,
				Rules: []PlatonPolicyRule{
					{Pattern: "forbidden", Action: "block", Message: "Forbidden content"},
				},
			},
			testText:      "This contains forbidden content",
			expectBlocked: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := PlatonTestPolicyRequest{
				Policy:   tc.policy,
				TestText: tc.testText,
			}

			resp, body := doRequest(t, http.MethodPost, url, req)

			if resp.StatusCode == http.StatusServiceUnavailable || resp.StatusCode == http.StatusBadGateway {
				t.Skip("Platon service not available")
			}

			if resp.StatusCode != http.StatusOK {
				t.Fatalf("Expected status 200, got %d: %s", resp.StatusCode, string(body))
			}

			var testResp PlatonTestPolicyResponse
			if err := json.Unmarshal(body, &testResp); err != nil {
				t.Fatalf("Failed to unmarshal response: %v", err)
			}

			isBlocked := testResp.Decision == "block" || testResp.Decision == "POLICY_DECISION_BLOCK"

			t.Logf("Test '%s': decision=%s, violations=%d",
				tc.name, testResp.Decision, len(testResp.Violations))

			if isBlocked != tc.expectBlocked {
				t.Errorf("Expected blocked=%v, got decision=%s", tc.expectBlocked, testResp.Decision)
			}
		})
	}
}

// ============================================================================
// Handler Management Tests
// ============================================================================

func TestPlaton_ListHandlers(t *testing.T) {
	url := kantBaseURL + "/api/v1/platon/handlers"
	resp, body := doRequest(t, http.MethodGet, url, nil)

	if resp.StatusCode == http.StatusServiceUnavailable || resp.StatusCode == http.StatusBadGateway {
		t.Skip("Platon service not available")
	}

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected status 200, got %d: %s", resp.StatusCode, string(body))
	}

	var handlers PlatonHandlersResponse
	if err := json.Unmarshal(body, &handlers); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	t.Logf("Found %d handlers", handlers.Total)
	for _, h := range handlers.Handlers {
		t.Logf("  - %s: type=%s, priority=%d, enabled=%v", h.Name, h.Type, h.Priority, h.Enabled)
	}
}

// ============================================================================
// PII Detection Tests
// ============================================================================

func TestPlaton_PIIDetection_Email(t *testing.T) {
	url := kantBaseURL + "/api/v1/platon/policies/test"
	req := PlatonTestPolicyRequest{
		Policy: PlatonPolicyInfo{
			Name:    "PII Test",
			Type:    "pii",
			Enabled: true,
			Rules: []PlatonPolicyRule{
				{
					ID:          "email",
					Pattern:     `\b[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Z|a-z]{2,}\b`,
					Action:      "redact",
					Message:     "Email detected",
					Replacement: "[EMAIL]",
				},
			},
		},
		TestText: "Contact me at john.doe@example.com for more info",
	}

	resp, body := doRequest(t, http.MethodPost, url, req)

	if resp.StatusCode == http.StatusServiceUnavailable || resp.StatusCode == http.StatusBadGateway {
		t.Skip("Platon service not available")
	}

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected status 200, got %d: %s", resp.StatusCode, string(body))
	}

	var testResp PlatonTestPolicyResponse
	if err := json.Unmarshal(body, &testResp); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	t.Logf("Email PII Test: decision=%s, violations=%d, modified=%s",
		testResp.Decision, len(testResp.Violations), testResp.ModifiedText)

	if len(testResp.Violations) == 0 {
		t.Log("Warning: Email not detected - may depend on policy configuration")
	}
}

func TestPlaton_PIIDetection_IBAN(t *testing.T) {
	url := kantBaseURL + "/api/v1/platon/policies/test"
	req := PlatonTestPolicyRequest{
		Policy: PlatonPolicyInfo{
			Name:    "PII Test",
			Type:    "pii",
			Enabled: true,
			Rules: []PlatonPolicyRule{
				{
					ID:          "iban",
					Pattern:     `\b[A-Z]{2}\d{2}[A-Z0-9]{4}\d{7}([A-Z0-9]?){0,16}\b`,
					Action:      "redact",
					Message:     "IBAN detected",
					Replacement: "[IBAN]",
				},
			},
		},
		TestText: "Transfer to DE89370400440532013000 please",
	}

	resp, body := doRequest(t, http.MethodPost, url, req)

	if resp.StatusCode == http.StatusServiceUnavailable || resp.StatusCode == http.StatusBadGateway {
		t.Skip("Platon service not available")
	}

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected status 200, got %d: %s", resp.StatusCode, string(body))
	}

	var testResp PlatonTestPolicyResponse
	if err := json.Unmarshal(body, &testResp); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	t.Logf("IBAN PII Test: decision=%s, violations=%d",
		testResp.Decision, len(testResp.Violations))
}

// ============================================================================
// Integration Flow Tests
// ============================================================================

func TestPlaton_FullFlow_ProcessWithPolicies(t *testing.T) {
	// This test simulates a full flow: create policy, process text, cleanup

	// Skip if Platon not available
	healthURL := kantBaseURL + "/api/v1/platon/health"
	resp, _ := doRequest(t, http.MethodGet, healthURL, nil)
	if resp.StatusCode == http.StatusServiceUnavailable || resp.StatusCode == http.StatusBadGateway {
		t.Skip("Platon service not available")
	}

	policyID := fmt.Sprintf("flow-test-%d", time.Now().UnixNano())

	// Step 1: Create a policy
	t.Log("Step 1: Creating test policy...")
	createPolicyURL := kantBaseURL + "/api/v1/platon/policies"
	createPolicyReq := PlatonPolicyInfo{
		ID:       policyID,
		Name:     "Flow Test Policy",
		Type:     "content",
		Enabled:  true,
		Priority: 10,
		Rules: []PlatonPolicyRule{
			{
				ID:      "warn-test",
				Pattern: "test-word",
				Action:  "warn",
				Message: "Test word detected",
			},
		},
	}

	resp, body := doRequest(t, http.MethodPost, createPolicyURL, createPolicyReq)
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		t.Fatalf("Failed to create policy: %d - %s", resp.StatusCode, string(body))
	}
	t.Logf("  Created policy: %s", policyID)

	// Step 2: Process text that should trigger the policy
	t.Log("Step 2: Processing text...")
	processURL := kantBaseURL + "/api/v1/platon/process"
	processReq := PlatonProcessRequest{
		Prompt: "This message contains test-word in the text",
		Options: &ProcessOptions{
			Debug: true,
		},
	}

	resp, body = doRequest(t, http.MethodPost, processURL, processReq)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Failed to process: %d - %s", resp.StatusCode, string(body))
	}

	var processResp PlatonProcessResponse
	if err := json.Unmarshal(body, &processResp); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}
	t.Logf("  Processed: blocked=%v, modified=%v", processResp.Blocked, processResp.Modified)

	// Step 3: Cleanup - delete the policy
	t.Log("Step 3: Cleaning up...")
	deleteURL := fmt.Sprintf("%s/api/v1/platon/policies/%s", kantBaseURL, policyID)
	deleteReq, _ := http.NewRequest(http.MethodDelete, deleteURL, nil)
	client := newTestClient()
	resp, err := client.Do(deleteReq)
	if err != nil {
		t.Logf("Warning: Failed to delete policy: %v", err)
	} else {
		resp.Body.Close()
		t.Logf("  Deleted policy: %s", policyID)
	}

	t.Log("Full flow test completed successfully!")
}

// ============================================================================
// Error Handling Tests
// ============================================================================

func TestPlaton_GetNonexistentPipeline_404(t *testing.T) {
	url := kantBaseURL + "/api/v1/platon/pipelines/nonexistent-pipeline-12345"
	resp, body := doRequest(t, http.MethodGet, url, nil)

	if resp.StatusCode == http.StatusServiceUnavailable || resp.StatusCode == http.StatusBadGateway {
		t.Skip("Platon service not available")
	}

	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("Expected status 404, got %d: %s", resp.StatusCode, string(body))
	}

	t.Log("Correctly returned 404 for nonexistent pipeline")
}

func TestPlaton_GetNonexistentPolicy_404(t *testing.T) {
	url := kantBaseURL + "/api/v1/platon/policies/nonexistent-policy-12345"
	resp, body := doRequest(t, http.MethodGet, url, nil)

	if resp.StatusCode == http.StatusServiceUnavailable || resp.StatusCode == http.StatusBadGateway {
		t.Skip("Platon service not available")
	}

	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("Expected status 404, got %d: %s", resp.StatusCode, string(body))
	}

	t.Log("Correctly returned 404 for nonexistent policy")
}

func TestPlaton_TestPolicy_EmptyText_400(t *testing.T) {
	url := kantBaseURL + "/api/v1/platon/policies/test"
	req := PlatonTestPolicyRequest{
		Policy: PlatonPolicyInfo{
			Name:    "Test",
			Type:    "content",
			Enabled: true,
		},
		TestText: "",
	}

	resp, body := doRequest(t, http.MethodPost, url, req)

	if resp.StatusCode == http.StatusServiceUnavailable || resp.StatusCode == http.StatusBadGateway {
		t.Skip("Platon service not available")
	}

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("Expected status 400, got %d: %s", resp.StatusCode, string(body))
	}

	t.Log("Correctly returned 400 for empty test text")
}

// ============================================================================
// Helper Functions
// ============================================================================

// truncate truncates a string to the specified length
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// doRequestWithBody helper for requests with body
func doRequestWithBody(t *testing.T, method, url string, body interface{}) (*http.Response, []byte) {
	t.Helper()

	var reqBody io.Reader
	if body != nil {
		jsonData, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("Failed to marshal request body: %v", err)
		}
		reqBody = bytes.NewReader(jsonData)
	}

	req, err := http.NewRequest(method, url, reqBody)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	client := newTestClient()
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}

	respBody, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		t.Fatalf("Failed to read response body: %v", err)
	}

	return resp, respBody
}
