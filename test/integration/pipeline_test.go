package integration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"testing"
)

// ============================================================================
// Pipeline Integration Tests
// ============================================================================

// TestKant_Pipeline_ListPipelines tests the list pipelines endpoint
func TestKant_Pipeline_ListPipelines(t *testing.T) {
	cfg := getTestConfig()
	skipIfServiceUnavailable(t, cfg.KantAddr, "Kant")
	skipIfServiceUnavailable(t, cfg.LeibnizAddr, "Leibniz")
	logTestStart(t, "Kant", "Pipeline_ListPipelines")

	url := fmt.Sprintf("http://%s/api/v1/pipeline/pipelines", cfg.KantAddr)
	resp, err := httpClient.Get(url)
	requireNoError(t, err, "GET /pipeline/pipelines failed")
	defer resp.Body.Close()

	requireEqual(t, http.StatusOK, resp.StatusCode, "Expected 200 OK")

	var result struct {
		Pipelines []struct {
			ID      string `json:"id"`
			Name    string `json:"name"`
			Enabled bool   `json:"enabled"`
		} `json:"pipelines"`
		Total int `json:"total"`
	}
	err = json.NewDecoder(resp.Body).Decode(&result)
	requireNoError(t, err, "Failed to decode response")

	t.Logf("Found %d pipelines (total: %d)", len(result.Pipelines), result.Total)
	for _, p := range result.Pipelines {
		t.Logf("  - Pipeline: %s (%s), enabled: %v", p.ID, p.Name, p.Enabled)
	}
}

// TestKant_Pipeline_CreateAndDeletePipeline tests pipeline CRUD operations
func TestKant_Pipeline_CreateAndDeletePipeline(t *testing.T) {
	cfg := getTestConfig()
	skipIfServiceUnavailable(t, cfg.KantAddr, "Kant")
	skipIfServiceUnavailable(t, cfg.LeibnizAddr, "Leibniz")
	logTestStart(t, "Kant", "Pipeline_CreateAndDeletePipeline")

	baseURL := fmt.Sprintf("http://%s/api/v1/pipeline/pipelines", cfg.KantAddr)

	// 1. Create a new pipeline
	createReq := map[string]interface{}{
		"id":          "test-integration-pipeline",
		"name":        "Integration Test Pipeline",
		"description": "Pipeline created by integration test",
		"enabled":     true,
		"settings": map[string]interface{}{
			"max_stages":            5,
			"stage_timeout_seconds": 30,
			"total_timeout_seconds": 120,
			"fail_open":             false,
		},
	}
	jsonBody, _ := json.Marshal(createReq)

	resp, err := httpClient.Post(baseURL, "application/json", bytes.NewReader(jsonBody))
	requireNoError(t, err, "POST /pipeline/pipelines failed")
	defer resp.Body.Close()

	// Accept both 201 Created and 200 OK (in case pipeline already exists)
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("Expected 201 or 200, got %d: %s", resp.StatusCode, string(body))
	}

	var createResult struct {
		ID      string `json:"id"`
		Name    string `json:"name"`
		Enabled bool   `json:"enabled"`
	}
	err = json.NewDecoder(resp.Body).Decode(&createResult)
	requireNoError(t, err, "Failed to decode create response")
	t.Logf("Created pipeline: %s (%s)", createResult.ID, createResult.Name)

	// 2. Get the pipeline
	getURL := fmt.Sprintf("%s/%s", baseURL, "test-integration-pipeline")
	resp2, err := httpClient.Get(getURL)
	requireNoError(t, err, "GET /pipeline/pipelines/{id} failed")
	defer resp2.Body.Close()

	requireEqual(t, http.StatusOK, resp2.StatusCode, "Expected 200 OK for GET")

	var getResult struct {
		ID          string `json:"id"`
		Name        string `json:"name"`
		Description string `json:"description"`
		Enabled     bool   `json:"enabled"`
	}
	err = json.NewDecoder(resp2.Body).Decode(&getResult)
	requireNoError(t, err, "Failed to decode get response")
	requireEqual(t, "test-integration-pipeline", getResult.ID, "Pipeline ID mismatch")
	t.Logf("Retrieved pipeline: %s - %s", getResult.ID, getResult.Description)

	// 3. Delete the pipeline
	deleteReq, _ := http.NewRequest(http.MethodDelete, getURL, nil)
	resp3, err := httpClient.Do(deleteReq)
	requireNoError(t, err, "DELETE /pipeline/pipelines/{id} failed")
	defer resp3.Body.Close()

	requireEqual(t, http.StatusOK, resp3.StatusCode, "Expected 200 OK for DELETE")
	t.Logf("Deleted pipeline: test-integration-pipeline")
}

// ============================================================================
// Policy Integration Tests
// ============================================================================

// TestKant_Pipeline_ListPolicies tests the list policies endpoint
func TestKant_Pipeline_ListPolicies(t *testing.T) {
	cfg := getTestConfig()
	skipIfServiceUnavailable(t, cfg.KantAddr, "Kant")
	skipIfServiceUnavailable(t, cfg.LeibnizAddr, "Leibniz")
	logTestStart(t, "Kant", "Pipeline_ListPolicies")

	url := fmt.Sprintf("http://%s/api/v1/pipeline/policies", cfg.KantAddr)
	resp, err := httpClient.Get(url)
	requireNoError(t, err, "GET /pipeline/policies failed")
	defer resp.Body.Close()

	requireEqual(t, http.StatusOK, resp.StatusCode, "Expected 200 OK")

	var result struct {
		Policies []struct {
			ID         string `json:"id"`
			Name       string `json:"name"`
			PolicyType string `json:"policy_type"`
			Enabled    bool   `json:"enabled"`
		} `json:"policies"`
		Total int `json:"total"`
	}
	err = json.NewDecoder(resp.Body).Decode(&result)
	requireNoError(t, err, "Failed to decode response")

	t.Logf("Found %d policies (total: %d)", len(result.Policies), result.Total)
	for _, p := range result.Policies {
		t.Logf("  - Policy: %s (%s), type: %s, enabled: %v", p.ID, p.Name, p.PolicyType, p.Enabled)
	}
}

// TestKant_Pipeline_CreateAndDeletePolicy tests policy CRUD operations
func TestKant_Pipeline_CreateAndDeletePolicy(t *testing.T) {
	cfg := getTestConfig()
	skipIfServiceUnavailable(t, cfg.KantAddr, "Kant")
	skipIfServiceUnavailable(t, cfg.LeibnizAddr, "Leibniz")
	logTestStart(t, "Kant", "Pipeline_CreateAndDeletePolicy")

	baseURL := fmt.Sprintf("http://%s/api/v1/pipeline/policies", cfg.KantAddr)

	// 1. Create a new policy
	createReq := map[string]interface{}{
		"id":          "test-integration-policy",
		"name":        "Integration Test Policy",
		"description": "Policy created by integration test",
		"policy_type": "content",
		"enabled":     true,
		"priority":    100,
		"rules": []map[string]interface{}{
			{
				"pattern":     "test-blocked-word",
				"action":      "block",
				"message":     "Test blocked word detected",
				"replacement": "",
			},
			{
				"pattern":     "test-redact-\\d+",
				"action":      "redact",
				"message":     "Test redaction pattern",
				"replacement": "[REDACTED]",
			},
		},
	}
	jsonBody, _ := json.Marshal(createReq)

	resp, err := httpClient.Post(baseURL, "application/json", bytes.NewReader(jsonBody))
	requireNoError(t, err, "POST /pipeline/policies failed")
	defer resp.Body.Close()

	// Accept both 201 Created and 200 OK
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("Expected 201 or 200, got %d: %s", resp.StatusCode, string(body))
	}

	var createResult struct {
		ID         string `json:"id"`
		Name       string `json:"name"`
		PolicyType string `json:"policy_type"`
	}
	err = json.NewDecoder(resp.Body).Decode(&createResult)
	requireNoError(t, err, "Failed to decode create response")
	t.Logf("Created policy: %s (%s), type: %s", createResult.ID, createResult.Name, createResult.PolicyType)

	// 2. Get the policy
	getURL := fmt.Sprintf("%s/%s", baseURL, "test-integration-policy")
	resp2, err := httpClient.Get(getURL)
	requireNoError(t, err, "GET /pipeline/policies/{id} failed")
	defer resp2.Body.Close()

	requireEqual(t, http.StatusOK, resp2.StatusCode, "Expected 200 OK for GET")

	var getResult struct {
		ID          string `json:"id"`
		Name        string `json:"name"`
		Description string `json:"description"`
		PolicyType  string `json:"policy_type"`
		Rules       []struct {
			Pattern string `json:"pattern"`
			Action  string `json:"action"`
		} `json:"rules"`
	}
	err = json.NewDecoder(resp2.Body).Decode(&getResult)
	requireNoError(t, err, "Failed to decode get response")
	requireEqual(t, "test-integration-policy", getResult.ID, "Policy ID mismatch")
	t.Logf("Retrieved policy: %s with %d rules", getResult.ID, len(getResult.Rules))

	// 3. Delete the policy
	deleteReq, _ := http.NewRequest(http.MethodDelete, getURL, nil)
	resp3, err := httpClient.Do(deleteReq)
	requireNoError(t, err, "DELETE /pipeline/policies/{id} failed")
	defer resp3.Body.Close()

	requireEqual(t, http.StatusOK, resp3.StatusCode, "Expected 200 OK for DELETE")
	t.Logf("Deleted policy: test-integration-policy")
}

// TestKant_Pipeline_TestPolicy tests the policy test endpoint
func TestKant_Pipeline_TestPolicy(t *testing.T) {
	cfg := getTestConfig()
	skipIfServiceUnavailable(t, cfg.KantAddr, "Kant")
	skipIfServiceUnavailable(t, cfg.LeibnizAddr, "Leibniz")
	logTestStart(t, "Kant", "Pipeline_TestPolicy")

	url := fmt.Sprintf("http://%s/api/v1/pipeline/policies/test", cfg.KantAddr)

	// Test with an inline policy
	testReq := map[string]interface{}{
		"policy": map[string]interface{}{
			"name":        "Test Policy",
			"policy_type": "content",
			"enabled":     true,
			"rules": []map[string]interface{}{
				{
					"pattern": "forbidden",
					"action":  "block",
					"message": "Forbidden word detected",
				},
			},
		},
		"test_text": "This text contains a forbidden word",
	}
	jsonBody, _ := json.Marshal(testReq)

	resp, err := httpClient.Post(url, "application/json", bytes.NewReader(jsonBody))
	requireNoError(t, err, "POST /pipeline/policies/test failed")
	defer resp.Body.Close()

	requireEqual(t, http.StatusOK, resp.StatusCode, "Expected 200 OK")

	var result struct {
		Decision     string `json:"decision"`
		Violations   []struct {
			PolicyName  string `json:"policy_name"`
			Description string `json:"description"`
			Action      string `json:"action"`
			Matched     string `json:"matched"`
		} `json:"violations"`
		ModifiedText string `json:"modified_text"`
		Reason       string `json:"reason"`
		DurationMs   int64  `json:"duration_ms"`
	}
	err = json.NewDecoder(resp.Body).Decode(&result)
	requireNoError(t, err, "Failed to decode response")

	t.Logf("Policy test result: decision=%s, violations=%d, duration=%dms",
		result.Decision, len(result.Violations), result.DurationMs)

	// The policy should detect the "forbidden" word
	if len(result.Violations) > 0 {
		t.Logf("Violation detected: %s - %s", result.Violations[0].Action, result.Violations[0].Description)
	}
}

// TestKant_Pipeline_TestPolicy_AllowedText tests policy with allowed text
func TestKant_Pipeline_TestPolicy_AllowedText(t *testing.T) {
	cfg := getTestConfig()
	skipIfServiceUnavailable(t, cfg.KantAddr, "Kant")
	skipIfServiceUnavailable(t, cfg.LeibnizAddr, "Leibniz")
	logTestStart(t, "Kant", "Pipeline_TestPolicy_AllowedText")

	url := fmt.Sprintf("http://%s/api/v1/pipeline/policies/test", cfg.KantAddr)

	// Test with text that should be allowed
	testReq := map[string]interface{}{
		"policy": map[string]interface{}{
			"name":        "Test Policy",
			"policy_type": "content",
			"enabled":     true,
			"rules": []map[string]interface{}{
				{
					"pattern": "forbidden",
					"action":  "block",
					"message": "Forbidden word detected",
				},
			},
		},
		"test_text": "This is a perfectly fine text without any issues",
	}
	jsonBody, _ := json.Marshal(testReq)

	resp, err := httpClient.Post(url, "application/json", bytes.NewReader(jsonBody))
	requireNoError(t, err, "POST /pipeline/policies/test failed")
	defer resp.Body.Close()

	requireEqual(t, http.StatusOK, resp.StatusCode, "Expected 200 OK")

	var result struct {
		Decision   string `json:"decision"`
		Violations []struct {
			PolicyName string `json:"policy_name"`
		} `json:"violations"`
		DurationMs int64 `json:"duration_ms"`
	}
	err = json.NewDecoder(resp.Body).Decode(&result)
	requireNoError(t, err, "Failed to decode response")

	t.Logf("Policy test result: decision=%s, violations=%d, duration=%dms",
		result.Decision, len(result.Violations), result.DurationMs)

	// Text should be allowed (no violations or allow decision)
	if result.Decision == "allow" || len(result.Violations) == 0 {
		t.Logf("Text correctly allowed")
	}
}

// TestKant_Pipeline_TestPolicy_Redaction tests policy redaction
func TestKant_Pipeline_TestPolicy_Redaction(t *testing.T) {
	cfg := getTestConfig()
	skipIfServiceUnavailable(t, cfg.KantAddr, "Kant")
	skipIfServiceUnavailable(t, cfg.LeibnizAddr, "Leibniz")
	logTestStart(t, "Kant", "Pipeline_TestPolicy_Redaction")

	url := fmt.Sprintf("http://%s/api/v1/pipeline/policies/test", cfg.KantAddr)

	// Test with text that should be redacted
	testReq := map[string]interface{}{
		"policy": map[string]interface{}{
			"name":        "PII Policy",
			"policy_type": "pii",
			"enabled":     true,
			"rules": []map[string]interface{}{
				{
					"pattern":     `\b[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Z|a-z]{2,}\b`,
					"action":      "redact",
					"message":     "Email address detected",
					"replacement": "[EMAIL]",
				},
			},
		},
		"test_text": "Contact me at john.doe@example.com for more info",
	}
	jsonBody, _ := json.Marshal(testReq)

	resp, err := httpClient.Post(url, "application/json", bytes.NewReader(jsonBody))
	requireNoError(t, err, "POST /pipeline/policies/test failed")
	defer resp.Body.Close()

	requireEqual(t, http.StatusOK, resp.StatusCode, "Expected 200 OK")

	var result struct {
		Decision     string `json:"decision"`
		Violations   []struct {
			Action  string `json:"action"`
			Matched string `json:"matched"`
		} `json:"violations"`
		ModifiedText string `json:"modified_text"`
		DurationMs   int64  `json:"duration_ms"`
	}
	err = json.NewDecoder(resp.Body).Decode(&result)
	requireNoError(t, err, "Failed to decode response")

	t.Logf("Policy test result: decision=%s, violations=%d, duration=%dms",
		result.Decision, len(result.Violations), result.DurationMs)

	if result.ModifiedText != "" {
		t.Logf("Modified text: %s", result.ModifiedText)
	}
}

// ============================================================================
// Pipeline Processing Tests
// ============================================================================

// TestKant_Pipeline_Process tests the pipeline process endpoint
func TestKant_Pipeline_Process(t *testing.T) {
	cfg := getTestConfig()
	skipIfServiceUnavailable(t, cfg.KantAddr, "Kant")
	skipIfServiceUnavailable(t, cfg.LeibnizAddr, "Leibniz")
	logTestStart(t, "Kant", "Pipeline_Process")

	url := fmt.Sprintf("http://%s/api/v1/pipeline/process", cfg.KantAddr)

	// Test basic pipeline processing
	processReq := map[string]interface{}{
		"prompt": "Hello, this is a test message for pipeline processing.",
		"metadata": map[string]string{
			"source": "integration_test",
		},
		"options": map[string]interface{}{
			"dry_run": true,
			"debug":   true,
		},
	}
	jsonBody, _ := json.Marshal(processReq)

	resp, err := httpClient.Post(url, "application/json", bytes.NewReader(jsonBody))
	requireNoError(t, err, "POST /pipeline/process failed")
	defer resp.Body.Close()

	requireEqual(t, http.StatusOK, resp.StatusCode, "Expected 200 OK")

	var result struct {
		RequestID       string `json:"request_id"`
		Success         bool   `json:"success"`
		Response        string `json:"response"`
		ProcessedPrompt string `json:"processed_prompt"`
		Flags           *struct {
			Blocked        bool   `json:"blocked"`
			Modified       bool   `json:"modified"`
			RequiresReview bool   `json:"requires_review"`
			BlockReason    string `json:"block_reason"`
		} `json:"flags"`
		StageResults []struct {
			StageName  string `json:"stage_name"`
			Success    bool   `json:"success"`
			Decision   string `json:"decision"`
			DurationMs int64  `json:"duration_ms"`
		} `json:"stage_results"`
		DurationMs int64  `json:"duration_ms"`
		Error      string `json:"error"`
	}
	err = json.NewDecoder(resp.Body).Decode(&result)
	requireNoError(t, err, "Failed to decode response")

	t.Logf("Pipeline process result:")
	t.Logf("  - Request ID: %s", result.RequestID)
	t.Logf("  - Success: %v", result.Success)
	t.Logf("  - Duration: %dms", result.DurationMs)
	t.Logf("  - Stages: %d", len(result.StageResults))

	if result.Flags != nil {
		t.Logf("  - Blocked: %v", result.Flags.Blocked)
		t.Logf("  - Modified: %v", result.Flags.Modified)
	}

	for _, stage := range result.StageResults {
		t.Logf("    Stage '%s': success=%v, decision=%s, duration=%dms",
			stage.StageName, stage.Success, stage.Decision, stage.DurationMs)
	}

	if result.Error != "" {
		t.Logf("  - Error: %s", result.Error)
	}
}

// TestKant_Pipeline_Process_WithPipelineID tests processing with specific pipeline
func TestKant_Pipeline_Process_WithPipelineID(t *testing.T) {
	cfg := getTestConfig()
	skipIfServiceUnavailable(t, cfg.KantAddr, "Kant")
	skipIfServiceUnavailable(t, cfg.LeibnizAddr, "Leibniz")
	logTestStart(t, "Kant", "Pipeline_Process_WithPipelineID")

	url := fmt.Sprintf("http://%s/api/v1/pipeline/process", cfg.KantAddr)

	// Test with default pipeline
	processReq := map[string]interface{}{
		"pipeline_id": "default",
		"prompt":      "Process this with the default pipeline",
		"options": map[string]interface{}{
			"dry_run": true,
		},
	}
	jsonBody, _ := json.Marshal(processReq)

	resp, err := httpClient.Post(url, "application/json", bytes.NewReader(jsonBody))
	requireNoError(t, err, "POST /pipeline/process failed")
	defer resp.Body.Close()

	requireEqual(t, http.StatusOK, resp.StatusCode, "Expected 200 OK")

	var result struct {
		RequestID  string `json:"request_id"`
		Success    bool   `json:"success"`
		DurationMs int64  `json:"duration_ms"`
	}
	err = json.NewDecoder(resp.Body).Decode(&result)
	requireNoError(t, err, "Failed to decode response")

	t.Logf("Pipeline process with ID: request_id=%s, success=%v, duration=%dms",
		result.RequestID, result.Success, result.DurationMs)
}

// ============================================================================
// Audit Log Tests
// ============================================================================

// TestKant_Pipeline_AuditLogs tests the audit logs endpoint
func TestKant_Pipeline_AuditLogs(t *testing.T) {
	cfg := getTestConfig()
	skipIfServiceUnavailable(t, cfg.KantAddr, "Kant")
	skipIfServiceUnavailable(t, cfg.LeibnizAddr, "Leibniz")
	logTestStart(t, "Kant", "Pipeline_AuditLogs")

	url := fmt.Sprintf("http://%s/api/v1/pipeline/audit", cfg.KantAddr)
	resp, err := httpClient.Get(url)
	requireNoError(t, err, "GET /pipeline/audit failed")
	defer resp.Body.Close()

	requireEqual(t, http.StatusOK, resp.StatusCode, "Expected 200 OK")

	var result struct {
		Logs []struct {
			RequestID  string `json:"request_id"`
			Timestamp  int64  `json:"timestamp"`
			Decision   string `json:"decision"`
			StageCount int    `json:"stage_count"`
		} `json:"logs"`
		Total int `json:"total"`
	}
	err = json.NewDecoder(resp.Body).Decode(&result)
	requireNoError(t, err, "Failed to decode response")

	t.Logf("Found %d audit logs (total: %d)", len(result.Logs), result.Total)
	for i, log := range result.Logs {
		if i >= 5 {
			t.Logf("  ... and %d more", len(result.Logs)-5)
			break
		}
		t.Logf("  - %s: decision=%s, stages=%d", log.RequestID, log.Decision, log.StageCount)
	}
}

// ============================================================================
// Edge Cases and Error Handling Tests
// ============================================================================

// TestKant_Pipeline_Process_EmptyPrompt tests error handling for empty prompt
func TestKant_Pipeline_Process_EmptyPrompt(t *testing.T) {
	cfg := getTestConfig()
	skipIfServiceUnavailable(t, cfg.KantAddr, "Kant")
	skipIfServiceUnavailable(t, cfg.LeibnizAddr, "Leibniz")
	logTestStart(t, "Kant", "Pipeline_Process_EmptyPrompt")

	url := fmt.Sprintf("http://%s/api/v1/pipeline/process", cfg.KantAddr)

	// Test with empty prompt (should fail)
	processReq := map[string]interface{}{
		"prompt": "",
	}
	jsonBody, _ := json.Marshal(processReq)

	resp, err := httpClient.Post(url, "application/json", bytes.NewReader(jsonBody))
	requireNoError(t, err, "POST /pipeline/process failed")
	defer resp.Body.Close()

	// Should return 400 Bad Request
	requireEqual(t, http.StatusBadRequest, resp.StatusCode, "Expected 400 Bad Request for empty prompt")

	t.Logf("Correctly rejected empty prompt with status %d", resp.StatusCode)
}

// TestKant_Pipeline_GetNonexistentPipeline tests error handling for missing pipeline
func TestKant_Pipeline_GetNonexistentPipeline(t *testing.T) {
	cfg := getTestConfig()
	skipIfServiceUnavailable(t, cfg.KantAddr, "Kant")
	skipIfServiceUnavailable(t, cfg.LeibnizAddr, "Leibniz")
	logTestStart(t, "Kant", "Pipeline_GetNonexistentPipeline")

	url := fmt.Sprintf("http://%s/api/v1/pipeline/pipelines/nonexistent-pipeline-12345", cfg.KantAddr)
	resp, err := httpClient.Get(url)
	requireNoError(t, err, "GET /pipeline/pipelines/{id} failed")
	defer resp.Body.Close()

	// Should return 404 Not Found
	requireEqual(t, http.StatusNotFound, resp.StatusCode, "Expected 404 Not Found for nonexistent pipeline")

	t.Logf("Correctly returned 404 for nonexistent pipeline")
}

// TestKant_Pipeline_GetNonexistentPolicy tests error handling for missing policy
func TestKant_Pipeline_GetNonexistentPolicy(t *testing.T) {
	cfg := getTestConfig()
	skipIfServiceUnavailable(t, cfg.KantAddr, "Kant")
	skipIfServiceUnavailable(t, cfg.LeibnizAddr, "Leibniz")
	logTestStart(t, "Kant", "Pipeline_GetNonexistentPolicy")

	url := fmt.Sprintf("http://%s/api/v1/pipeline/policies/nonexistent-policy-12345", cfg.KantAddr)
	resp, err := httpClient.Get(url)
	requireNoError(t, err, "GET /pipeline/policies/{id} failed")
	defer resp.Body.Close()

	// Should return 404 Not Found
	requireEqual(t, http.StatusNotFound, resp.StatusCode, "Expected 404 Not Found for nonexistent policy")

	t.Logf("Correctly returned 404 for nonexistent policy")
}

// TestKant_Pipeline_TestPolicy_EmptyText tests error handling for empty test text
func TestKant_Pipeline_TestPolicy_EmptyText(t *testing.T) {
	cfg := getTestConfig()
	skipIfServiceUnavailable(t, cfg.KantAddr, "Kant")
	skipIfServiceUnavailable(t, cfg.LeibnizAddr, "Leibniz")
	logTestStart(t, "Kant", "Pipeline_TestPolicy_EmptyText")

	url := fmt.Sprintf("http://%s/api/v1/pipeline/policies/test", cfg.KantAddr)

	testReq := map[string]interface{}{
		"policy": map[string]interface{}{
			"name":        "Test Policy",
			"policy_type": "content",
			"enabled":     true,
		},
		"test_text": "",
	}
	jsonBody, _ := json.Marshal(testReq)

	resp, err := httpClient.Post(url, "application/json", bytes.NewReader(jsonBody))
	requireNoError(t, err, "POST /pipeline/policies/test failed")
	defer resp.Body.Close()

	// Should return 400 Bad Request
	requireEqual(t, http.StatusBadRequest, resp.StatusCode, "Expected 400 Bad Request for empty test text")

	t.Logf("Correctly rejected empty test text with status %d", resp.StatusCode)
}
