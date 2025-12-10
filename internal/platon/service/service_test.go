package service

import (
	"context"
	"testing"

	"github.com/msto63/mDW/internal/platon/chain"
	"github.com/msto63/mDW/pkg/core/logging"
)

// MockHandler for testing
type mockHandler struct {
	name        string
	htype       chain.HandlerType
	priority    int
	processFunc func(ctx *chain.ProcessingContext) error
}

func newMockHandler(name string, htype chain.HandlerType, priority int) *mockHandler {
	return &mockHandler{
		name:     name,
		htype:    htype,
		priority: priority,
	}
}

func (h *mockHandler) Name() string                                 { return h.name }
func (h *mockHandler) Type() chain.HandlerType                      { return h.htype }
func (h *mockHandler) Priority() int                                { return h.priority }
func (h *mockHandler) ShouldProcess(*chain.ProcessingContext) bool  { return true }

func (h *mockHandler) Process(ctx *chain.ProcessingContext) error {
	if h.processFunc != nil {
		return h.processFunc(ctx)
	}
	return nil
}

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.DefaultPipeline != "default" {
		t.Errorf("expected DefaultPipeline 'default', got '%s'", cfg.DefaultPipeline)
	}

	if cfg.MaxHandlers != 100 {
		t.Errorf("expected MaxHandlers 100, got %d", cfg.MaxHandlers)
	}
}

func TestNewService(t *testing.T) {
	cfg := DefaultConfig()
	logger := *logging.New("test")
	svc := NewService(cfg, logger)

	if svc == nil {
		t.Fatal("NewService returned nil")
	}

	if svc.chain == nil {
		t.Error("expected chain to be initialized")
	}
}

func TestService_RegisterHandler(t *testing.T) {
	cfg := DefaultConfig()
	cfg.MaxHandlers = 2
	logger := *logging.New("test")
	svc := NewService(cfg, logger)

	h1 := newMockHandler("handler1", chain.HandlerTypePre, 1)
	h2 := newMockHandler("handler2", chain.HandlerTypePre, 2)
	h3 := newMockHandler("handler3", chain.HandlerTypePre, 3)

	// Register first two handlers
	if err := svc.RegisterHandler(h1); err != nil {
		t.Errorf("failed to register handler1: %v", err)
	}

	if err := svc.RegisterHandler(h2); err != nil {
		t.Errorf("failed to register handler2: %v", err)
	}

	// Third handler should fail (max handlers reached)
	if err := svc.RegisterHandler(h3); err == nil {
		t.Error("expected error when registering third handler (max exceeded)")
	}
}

func TestService_UnregisterHandler(t *testing.T) {
	cfg := DefaultConfig()
	logger := *logging.New("test")
	svc := NewService(cfg, logger)

	h := newMockHandler("test", chain.HandlerTypePre, 1)
	_ = svc.RegisterHandler(h)

	// Unregister existing
	if !svc.UnregisterHandler("test") {
		t.Error("expected UnregisterHandler to return true")
	}

	// Unregister non-existing
	if svc.UnregisterHandler("nonexistent") {
		t.Error("expected UnregisterHandler to return false for non-existent handler")
	}
}

func TestService_GetHandler(t *testing.T) {
	cfg := DefaultConfig()
	logger := *logging.New("test")
	svc := NewService(cfg, logger)

	h := newMockHandler("findme", chain.HandlerTypePre, 1)
	_ = svc.RegisterHandler(h)

	// Get existing
	found, ok := svc.GetHandler("findme")
	if !ok {
		t.Error("expected to find handler")
	}
	if found.Name() != "findme" {
		t.Errorf("expected handler name 'findme', got '%s'", found.Name())
	}

	// Get non-existing
	_, ok = svc.GetHandler("notfound")
	if ok {
		t.Error("expected not to find non-existent handler")
	}
}

func TestService_ListHandlers(t *testing.T) {
	cfg := DefaultConfig()
	logger := *logging.New("test")
	svc := NewService(cfg, logger)

	_ = svc.RegisterHandler(newMockHandler("h1", chain.HandlerTypePre, 1))
	_ = svc.RegisterHandler(newMockHandler("h2", chain.HandlerTypePost, 2))

	handlers := svc.ListHandlers()

	if len(handlers) != 2 {
		t.Errorf("expected 2 handlers, got %d", len(handlers))
	}
}

func TestService_CreatePipeline(t *testing.T) {
	cfg := DefaultConfig()
	logger := *logging.New("test")
	svc := NewService(cfg, logger)

	p := &chain.Pipeline{
		ID:          "test-pipeline",
		Name:        "Test Pipeline",
		Description: "A test pipeline",
		Enabled:     true,
	}

	// Create pipeline
	err := svc.CreatePipeline(p)
	if err != nil {
		t.Fatalf("failed to create pipeline: %v", err)
	}

	// Check timestamps were set
	if p.CreatedAt.IsZero() {
		t.Error("expected CreatedAt to be set")
	}
	if p.UpdatedAt.IsZero() {
		t.Error("expected UpdatedAt to be set")
	}

	// Create duplicate should fail
	err = svc.CreatePipeline(p)
	if err == nil {
		t.Error("expected error when creating duplicate pipeline")
	}
}

func TestService_GetPipeline(t *testing.T) {
	cfg := DefaultConfig()
	logger := *logging.New("test")
	svc := NewService(cfg, logger)

	p := &chain.Pipeline{ID: "get-test", Name: "Get Test"}
	_ = svc.CreatePipeline(p)

	// Get existing
	found, err := svc.GetPipeline("get-test")
	if err != nil {
		t.Fatalf("failed to get pipeline: %v", err)
	}
	if found.Name != "Get Test" {
		t.Errorf("expected name 'Get Test', got '%s'", found.Name)
	}

	// Get non-existing
	_, err = svc.GetPipeline("nonexistent")
	if err == nil {
		t.Error("expected error when getting non-existent pipeline")
	}
}

func TestService_UpdatePipeline(t *testing.T) {
	cfg := DefaultConfig()
	logger := *logging.New("test")
	svc := NewService(cfg, logger)

	p := &chain.Pipeline{ID: "update-test", Name: "Original"}
	_ = svc.CreatePipeline(p)

	// Update
	updated := &chain.Pipeline{ID: "update-test", Name: "Updated"}
	err := svc.UpdatePipeline(updated)
	if err != nil {
		t.Fatalf("failed to update pipeline: %v", err)
	}

	// Verify
	found, _ := svc.GetPipeline("update-test")
	if found.Name != "Updated" {
		t.Errorf("expected name 'Updated', got '%s'", found.Name)
	}

	// Update non-existing
	err = svc.UpdatePipeline(&chain.Pipeline{ID: "nonexistent"})
	if err == nil {
		t.Error("expected error when updating non-existent pipeline")
	}
}

func TestService_DeletePipeline(t *testing.T) {
	cfg := DefaultConfig()
	logger := *logging.New("test")
	svc := NewService(cfg, logger)

	p := &chain.Pipeline{ID: "delete-test", Name: "To Delete"}
	_ = svc.CreatePipeline(p)

	// Delete existing
	err := svc.DeletePipeline("delete-test")
	if err != nil {
		t.Fatalf("failed to delete pipeline: %v", err)
	}

	// Verify deleted
	_, err = svc.GetPipeline("delete-test")
	if err == nil {
		t.Error("expected error when getting deleted pipeline")
	}

	// Delete non-existing
	err = svc.DeletePipeline("nonexistent")
	if err == nil {
		t.Error("expected error when deleting non-existent pipeline")
	}
}

func TestService_ListPipelines(t *testing.T) {
	cfg := DefaultConfig()
	logger := *logging.New("test")
	svc := NewService(cfg, logger)

	_ = svc.CreatePipeline(&chain.Pipeline{ID: "p1", Name: "Pipeline 1"})
	_ = svc.CreatePipeline(&chain.Pipeline{ID: "p2", Name: "Pipeline 2"})

	pipelines := svc.ListPipelines()

	if len(pipelines) != 2 {
		t.Errorf("expected 2 pipelines, got %d", len(pipelines))
	}
}

func TestService_ProcessPre(t *testing.T) {
	cfg := DefaultConfig()
	logger := *logging.New("test")
	svc := NewService(cfg, logger)

	h := newMockHandler("modifier", chain.HandlerTypePre, 1)
	h.processFunc = func(ctx *chain.ProcessingContext) error {
		ctx.Prompt = ctx.Prompt + " modified"
		ctx.MarkModified()
		return nil
	}
	_ = svc.RegisterHandler(h)

	req := &chain.ProcessRequest{
		Prompt: "test",
	}

	result, err := svc.ProcessPre(context.Background(), req)
	if err != nil {
		t.Fatalf("ProcessPre failed: %v", err)
	}

	if result.ProcessedPrompt != "test modified" {
		t.Errorf("expected 'test modified', got '%s'", result.ProcessedPrompt)
	}

	if !result.Modified {
		t.Error("expected Modified to be true")
	}

	// Check request ID was generated
	if result.RequestID == "" {
		t.Error("expected RequestID to be generated")
	}
}

func TestService_ProcessPost(t *testing.T) {
	cfg := DefaultConfig()
	logger := *logging.New("test")
	svc := NewService(cfg, logger)

	h := newMockHandler("postmod", chain.HandlerTypePost, 1)
	h.processFunc = func(ctx *chain.ProcessingContext) error {
		ctx.Response = ctx.Response + " processed"
		return nil
	}
	_ = svc.RegisterHandler(h)

	req := &chain.ProcessRequest{
		Prompt:   "prompt",
		Response: "response",
	}

	result, err := svc.ProcessPost(context.Background(), req)
	if err != nil {
		t.Fatalf("ProcessPost failed: %v", err)
	}

	if result.ProcessedResponse != "response processed" {
		t.Errorf("expected 'response processed', got '%s'", result.ProcessedResponse)
	}
}

func TestService_Process(t *testing.T) {
	cfg := DefaultConfig()
	logger := *logging.New("test")
	svc := NewService(cfg, logger)

	preHandler := newMockHandler("pre", chain.HandlerTypePre, 1)
	preHandler.processFunc = func(ctx *chain.ProcessingContext) error {
		ctx.Prompt = "[PRE] " + ctx.Prompt
		return nil
	}
	_ = svc.RegisterHandler(preHandler)

	postHandler := newMockHandler("post", chain.HandlerTypePost, 1)
	postHandler.processFunc = func(ctx *chain.ProcessingContext) error {
		ctx.Response = ctx.Response + " [POST]"
		return nil
	}
	_ = svc.RegisterHandler(postHandler)

	mainProcessor := func(_ context.Context, prompt string) (string, error) {
		return "Echo: " + prompt, nil
	}

	req := &chain.ProcessRequest{
		RequestID: "test-req",
		Prompt:    "Hello",
	}

	result, err := svc.Process(context.Background(), req, mainProcessor)
	if err != nil {
		t.Fatalf("Process failed: %v", err)
	}

	expectedPrompt := "[PRE] Hello"
	if result.ProcessedPrompt != expectedPrompt {
		t.Errorf("expected prompt '%s', got '%s'", expectedPrompt, result.ProcessedPrompt)
	}

	expectedResponse := "Echo: [PRE] Hello [POST]"
	if result.ProcessedResponse != expectedResponse {
		t.Errorf("expected response '%s', got '%s'", expectedResponse, result.ProcessedResponse)
	}
}

func TestService_Stats(t *testing.T) {
	cfg := DefaultConfig()
	logger := *logging.New("test")
	svc := NewService(cfg, logger)

	_ = svc.RegisterHandler(newMockHandler("h1", chain.HandlerTypePre, 1))
	_ = svc.RegisterHandler(newMockHandler("h2", chain.HandlerTypePost, 1))
	_ = svc.CreatePipeline(&chain.Pipeline{ID: "p1"})

	stats := svc.Stats()

	if stats["total_handlers"].(int) != 2 {
		t.Errorf("expected total_handlers 2, got %v", stats["total_handlers"])
	}

	if stats["pre_handlers"].(int) != 1 {
		t.Errorf("expected pre_handlers 1, got %v", stats["pre_handlers"])
	}

	if stats["post_handlers"].(int) != 1 {
		t.Errorf("expected post_handlers 1, got %v", stats["post_handlers"])
	}

	if stats["pipeline_count"].(int) != 1 {
		t.Errorf("expected pipeline_count 1, got %v", stats["pipeline_count"])
	}
}

func TestService_LoadDefaultPipeline(t *testing.T) {
	cfg := DefaultConfig()
	logger := *logging.New("test")
	svc := NewService(cfg, logger)

	// Load default pipeline
	err := svc.LoadDefaultPipeline()
	if err != nil {
		t.Fatalf("LoadDefaultPipeline failed: %v", err)
	}

	// Verify it exists
	p, err := svc.GetPipeline("default")
	if err != nil {
		t.Fatalf("failed to get default pipeline: %v", err)
	}

	if p.Name != "Default Pipeline" {
		t.Errorf("expected name 'Default Pipeline', got '%s'", p.Name)
	}

	// Call again should not error (idempotent)
	err = svc.LoadDefaultPipeline()
	if err != nil {
		t.Fatalf("second LoadDefaultPipeline failed: %v", err)
	}
}

func TestService_ValidateRequest(t *testing.T) {
	cfg := DefaultConfig()
	logger := *logging.New("test")
	svc := NewService(cfg, logger)

	tests := []struct {
		name    string
		req     *chain.ProcessRequest
		wantErr bool
	}{
		{
			name:    "valid with prompt",
			req:     &chain.ProcessRequest{Prompt: "test"},
			wantErr: false,
		},
		{
			name:    "valid with response",
			req:     &chain.ProcessRequest{Response: "test"},
			wantErr: false,
		},
		{
			name:    "valid with both",
			req:     &chain.ProcessRequest{Prompt: "p", Response: "r"},
			wantErr: false,
		},
		{
			name:    "invalid - empty",
			req:     &chain.ProcessRequest{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := svc.ValidateRequest(tt.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateRequest() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestService_Chain(t *testing.T) {
	cfg := DefaultConfig()
	logger := *logging.New("test")
	svc := NewService(cfg, logger)

	if svc.Chain() == nil {
		t.Error("Chain() returned nil")
	}
}

func TestService_Close(t *testing.T) {
	cfg := DefaultConfig()
	logger := *logging.New("test")
	svc := NewService(cfg, logger)

	err := svc.Close()
	if err != nil {
		t.Errorf("Close() returned error: %v", err)
	}
}

// ============================================================================
// Policy Management Tests
// ============================================================================

func TestService_CreatePolicy(t *testing.T) {
	cfg := DefaultConfig()
	logger := *logging.New("test")
	svc := NewService(cfg, logger)

	policy := &Policy{
		ID:          "test-policy",
		Name:        "Test Policy",
		Description: "A test policy",
		Type:        "content",
		Enabled:     true,
		Priority:    10,
	}

	err := svc.CreatePolicy(policy)
	if err != nil {
		t.Fatalf("CreatePolicy failed: %v", err)
	}

	if policy.CreatedAt.IsZero() {
		t.Error("CreatedAt should be set")
	}
	if policy.UpdatedAt.IsZero() {
		t.Error("UpdatedAt should be set")
	}

	// Duplicate should fail
	err = svc.CreatePolicy(policy)
	if err == nil {
		t.Error("expected error when creating duplicate policy")
	}
}

func TestService_GetPolicy(t *testing.T) {
	cfg := DefaultConfig()
	logger := *logging.New("test")
	svc := NewService(cfg, logger)

	policy := &Policy{
		ID:   "get-policy",
		Name: "Policy to Get",
		Type: "pii",
	}
	svc.CreatePolicy(policy)

	got, err := svc.GetPolicy("get-policy")
	if err != nil {
		t.Fatalf("GetPolicy failed: %v", err)
	}
	if got.Type != "pii" {
		t.Errorf("Type = %s, expected 'pii'", got.Type)
	}

	_, err = svc.GetPolicy("nonexistent")
	if err == nil {
		t.Error("expected error for non-existent policy")
	}
}

func TestService_UpdatePolicy(t *testing.T) {
	cfg := DefaultConfig()
	logger := *logging.New("test")
	svc := NewService(cfg, logger)

	original := &Policy{
		ID:   "update-policy",
		Name: "Original Policy",
	}
	svc.CreatePolicy(original)

	updated := &Policy{
		ID:   "update-policy",
		Name: "Updated Policy",
	}

	err := svc.UpdatePolicy(updated)
	if err != nil {
		t.Fatalf("UpdatePolicy failed: %v", err)
	}

	got, _ := svc.GetPolicy("update-policy")
	if got.Name != "Updated Policy" {
		t.Errorf("Name = %s, expected 'Updated Policy'", got.Name)
	}

	err = svc.UpdatePolicy(&Policy{ID: "nonexistent"})
	if err == nil {
		t.Error("expected error for non-existent policy")
	}
}

func TestService_DeletePolicy(t *testing.T) {
	cfg := DefaultConfig()
	logger := *logging.New("test")
	svc := NewService(cfg, logger)

	policy := &Policy{
		ID:   "delete-policy",
		Name: "Policy to Delete",
	}
	svc.CreatePolicy(policy)

	err := svc.DeletePolicy("delete-policy")
	if err != nil {
		t.Fatalf("DeletePolicy failed: %v", err)
	}

	_, err = svc.GetPolicy("delete-policy")
	if err == nil {
		t.Error("policy should be deleted")
	}

	err = svc.DeletePolicy("nonexistent")
	if err == nil {
		t.Error("expected error for non-existent policy")
	}
}

func TestService_ListPolicies(t *testing.T) {
	cfg := DefaultConfig()
	logger := *logging.New("test")
	svc := NewService(cfg, logger)

	for i := 0; i < 3; i++ {
		svc.CreatePolicy(&Policy{
			ID:   "policy-" + string(rune('a'+i)),
			Name: "Policy " + string(rune('A'+i)),
		})
	}

	list := svc.ListPolicies()
	if len(list) != 3 {
		t.Errorf("ListPolicies length = %d, expected 3", len(list))
	}
}

// ============================================================================
// Policy Testing Tests
// ============================================================================

func TestService_TestPolicy_NoViolations(t *testing.T) {
	cfg := DefaultConfig()
	logger := *logging.New("test")
	svc := NewService(cfg, logger)

	policy := &Policy{
		ID:      "clean-policy",
		Name:    "Clean Policy",
		Enabled: true,
		Rules:   []PolicyRule{},
	}

	result, err := svc.TestPolicy(policy, "This is clean text")
	if err != nil {
		t.Fatalf("TestPolicy failed: %v", err)
	}

	if result.Decision != "allow" {
		t.Errorf("Decision = %s, expected 'allow'", result.Decision)
	}
	if len(result.Violations) != 0 {
		t.Error("Should have no violations")
	}
}

func TestService_TestPolicy_Block(t *testing.T) {
	cfg := DefaultConfig()
	logger := *logging.New("test")
	svc := NewService(cfg, logger)

	policy := &Policy{
		ID:      "block-policy",
		Name:    "Block Policy",
		Enabled: true,
		Rules: []PolicyRule{
			{
				ID:      "block-forbidden",
				Pattern: "forbidden",
				Action:  "block",
				Message: "Forbidden word detected",
			},
		},
	}

	result, err := svc.TestPolicy(policy, "This contains forbidden content")
	if err != nil {
		t.Fatalf("TestPolicy failed: %v", err)
	}

	if result.Decision != "block" {
		t.Errorf("Decision = %s, expected 'block'", result.Decision)
	}
	if len(result.Violations) != 1 {
		t.Errorf("Violations count = %d, expected 1", len(result.Violations))
	}
	if result.Violations[0].Action != "block" {
		t.Errorf("Violation action = %s, expected 'block'", result.Violations[0].Action)
	}
}

func TestService_TestPolicy_Redact(t *testing.T) {
	cfg := DefaultConfig()
	logger := *logging.New("test")
	svc := NewService(cfg, logger)

	policy := &Policy{
		ID:      "redact-policy",
		Name:    "Redact Policy",
		Enabled: true,
		Rules: []PolicyRule{
			{
				ID:          "redact-email",
				Pattern:     `[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}`,
				Action:      "redact",
				Message:     "Email address detected",
				Replacement: "[EMAIL]",
			},
		},
	}

	result, err := svc.TestPolicy(policy, "Contact me at test@example.com please")
	if err != nil {
		t.Fatalf("TestPolicy failed: %v", err)
	}

	if result.Decision != "modify" {
		t.Errorf("Decision = %s, expected 'modify'", result.Decision)
	}
	if result.ModifiedText != "Contact me at [EMAIL] please" {
		t.Errorf("ModifiedText = %s, expected redacted email", result.ModifiedText)
	}
}

func TestService_TestPolicy_Warn(t *testing.T) {
	cfg := DefaultConfig()
	logger := *logging.New("test")
	svc := NewService(cfg, logger)

	policy := &Policy{
		ID:      "warn-policy",
		Name:    "Warn Policy",
		Enabled: true,
		Rules: []PolicyRule{
			{
				ID:      "warn-sensitive",
				Pattern: "sensitive",
				Action:  "warn",
				Message: "Sensitive content detected",
			},
		},
	}

	result, err := svc.TestPolicy(policy, "This contains sensitive information")
	if err != nil {
		t.Fatalf("TestPolicy failed: %v", err)
	}

	if result.Decision != "escalate" {
		t.Errorf("Decision = %s, expected 'escalate'", result.Decision)
	}
}

func TestService_TestPolicy_CaseInsensitive(t *testing.T) {
	cfg := DefaultConfig()
	logger := *logging.New("test")
	svc := NewService(cfg, logger)

	policy := &Policy{
		ID:      "case-policy",
		Name:    "Case Policy",
		Enabled: true,
		Rules: []PolicyRule{
			{
				ID:            "case-rule",
				Pattern:       "secret",
				Action:        "block",
				Message:       "Secret detected",
				CaseSensitive: false,
			},
		},
	}

	result, _ := svc.TestPolicy(policy, "This is SECRET")
	if result.Decision != "block" {
		t.Error("Should match case-insensitively")
	}

	result, _ = svc.TestPolicy(policy, "This is SeCrEt")
	if result.Decision != "block" {
		t.Error("Should match mixed case")
	}
}

func TestService_TestPolicy_CaseSensitive(t *testing.T) {
	cfg := DefaultConfig()
	logger := *logging.New("test")
	svc := NewService(cfg, logger)

	policy := &Policy{
		ID:      "case-sensitive-policy",
		Name:    "Case Sensitive Policy",
		Enabled: true,
		Rules: []PolicyRule{
			{
				ID:            "case-rule",
				Pattern:       "Secret",
				Action:        "block",
				Message:       "Secret detected",
				CaseSensitive: true,
			},
		},
	}

	result, _ := svc.TestPolicy(policy, "This is Secret")
	if result.Decision != "block" {
		t.Error("Should match exact case")
	}

	result, _ = svc.TestPolicy(policy, "This is secret")
	if result.Decision != "allow" {
		t.Error("Should not match lowercase when case-sensitive")
	}
}

func TestService_TestPolicy_InvalidPattern(t *testing.T) {
	cfg := DefaultConfig()
	logger := *logging.New("test")
	svc := NewService(cfg, logger)

	policy := &Policy{
		ID:      "invalid-policy",
		Name:    "Invalid Policy",
		Enabled: true,
		Rules: []PolicyRule{
			{
				ID:      "invalid-regex",
				Pattern: "[invalid(regex",
				Action:  "block",
			},
		},
	}

	_, err := svc.TestPolicy(policy, "test text")
	if err == nil {
		t.Error("Should return error for invalid regex pattern")
	}
}

func TestService_TestPolicy_DefaultRedaction(t *testing.T) {
	cfg := DefaultConfig()
	logger := *logging.New("test")
	svc := NewService(cfg, logger)

	policy := &Policy{
		ID:      "default-redact-policy",
		Name:    "Default Redact Policy",
		Enabled: true,
		Rules: []PolicyRule{
			{
				ID:          "redact-no-replacement",
				Pattern:     "secret",
				Action:      "redact",
				Message:     "Secret detected",
				Replacement: "", // Empty replacement
			},
		},
	}

	result, err := svc.TestPolicy(policy, "This is secret data")
	if err != nil {
		t.Fatalf("TestPolicy failed: %v", err)
	}

	if result.ModifiedText != "This is [REDACTED] data" {
		t.Errorf("ModifiedText = %s, expected default [REDACTED]", result.ModifiedText)
	}
}

// ============================================================================
// Severity Mapping Tests
// ============================================================================

func TestGetSeverityForAction(t *testing.T) {
	tests := []struct {
		action   string
		expected string
	}{
		{"block", "critical"},
		{"redact", "high"},
		{"warn", "medium"},
		{"log", "low"},
		{"unknown", "info"},
		{"", "info"},
	}

	for _, tt := range tests {
		t.Run(tt.action, func(t *testing.T) {
			result := getSeverityForAction(tt.action)
			if result != tt.expected {
				t.Errorf("getSeverityForAction(%s) = %s, expected %s", tt.action, result, tt.expected)
			}
		})
	}
}

// ============================================================================
// Dynamic Handler Tests
// ============================================================================

func TestService_RegisterDynamicHandler(t *testing.T) {
	cfg := DefaultConfig()
	logger := *logging.New("test")
	svc := NewService(cfg, logger)

	dynCfg := DynamicHandlerConfig{
		Name:        "dynamic-handler",
		Type:        chain.HandlerTypePre,
		Priority:    50,
		Description: "A dynamic handler",
		Enabled:     true,
		Settings:    map[string]string{"key": "value"},
	}

	h, err := svc.RegisterDynamicHandler(dynCfg)
	if err != nil {
		t.Fatalf("RegisterDynamicHandler failed: %v", err)
	}

	if h == nil {
		t.Fatal("Handler should not be nil")
	}

	// Verify registered
	found, ok := svc.GetHandler("dynamic-handler")
	if !ok {
		t.Error("Handler should be found after registration")
	}
	if found.Priority() != 50 {
		t.Errorf("Priority = %d, expected 50", found.Priority())
	}
}

func TestService_RegisterDynamicHandler_Duplicate(t *testing.T) {
	cfg := DefaultConfig()
	logger := *logging.New("test")
	svc := NewService(cfg, logger)

	dynCfg := DynamicHandlerConfig{
		Name:     "dup-handler",
		Type:     chain.HandlerTypePre,
		Priority: 10,
	}

	_, err := svc.RegisterDynamicHandler(dynCfg)
	if err != nil {
		t.Fatalf("First registration failed: %v", err)
	}

	_, err = svc.RegisterDynamicHandler(dynCfg)
	if err == nil {
		t.Error("Should fail for duplicate handler")
	}
}

func TestService_RegisterDynamicHandler_MaxLimit(t *testing.T) {
	logger := *logging.New("test")
	cfg := Config{
		DefaultPipeline: "default",
		MaxHandlers:     1,
	}
	svc := NewService(cfg, logger)

	// Register first
	_, err := svc.RegisterDynamicHandler(DynamicHandlerConfig{
		Name:     "first",
		Type:     chain.HandlerTypePre,
		Priority: 10,
	})
	if err != nil {
		t.Fatalf("First registration failed: %v", err)
	}

	// Second should fail
	_, err = svc.RegisterDynamicHandler(DynamicHandlerConfig{
		Name:     "second",
		Type:     chain.HandlerTypePre,
		Priority: 20,
	})
	if err == nil {
		t.Error("Should fail when max handlers reached")
	}
}

// ============================================================================
// Benchmark Tests
// ============================================================================

func BenchmarkService_TestPolicy_Simple(b *testing.B) {
	cfg := DefaultConfig()
	logger := *logging.New("bench")
	svc := NewService(cfg, logger)

	policy := &Policy{
		ID:      "bench-policy",
		Enabled: true,
		Rules: []PolicyRule{
			{ID: "r1", Pattern: "test", Action: "warn"},
		},
	}

	text := "This is a test message for benchmarking."

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		svc.TestPolicy(policy, text)
	}
}

func BenchmarkService_TestPolicy_Complex(b *testing.B) {
	cfg := DefaultConfig()
	logger := *logging.New("bench")
	svc := NewService(cfg, logger)

	policy := &Policy{
		ID:      "bench-policy",
		Enabled: true,
		Rules: []PolicyRule{
			{ID: "r1", Pattern: `[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}`, Action: "redact"},
			{ID: "r2", Pattern: `\b\d{3}-\d{2}-\d{4}\b`, Action: "redact"},
			{ID: "r3", Pattern: `\b\d{4}[- ]?\d{4}[- ]?\d{4}[- ]?\d{4}\b`, Action: "redact"},
		},
	}

	text := "Contact john@example.com or 123-45-6789 for credit card 4111-1111-1111-1111"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		svc.TestPolicy(policy, text)
	}
}

func BenchmarkService_ProcessPre(b *testing.B) {
	cfg := DefaultConfig()
	logger := *logging.New("bench")
	svc := NewService(cfg, logger)

	h := newMockHandler("bench", chain.HandlerTypePre, 1)
	h.processFunc = func(ctx *chain.ProcessingContext) error {
		ctx.Prompt = ctx.Prompt + " processed"
		return nil
	}
	svc.RegisterHandler(h)

	req := &chain.ProcessRequest{
		Prompt: "benchmark test prompt",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		svc.ProcessPre(context.Background(), req)
	}
}

// ============================================================================
// Concurrency Tests
// ============================================================================

func TestService_ConcurrentPipelineAccess(t *testing.T) {
	cfg := DefaultConfig()
	logger := *logging.New("test")
	svc := NewService(cfg, logger)

	done := make(chan bool, 100)

	// Concurrent creates
	for i := 0; i < 50; i++ {
		go func(id int) {
			svc.CreatePipeline(&chain.Pipeline{
				ID:   "concurrent-" + string(rune(id%26+'a')),
				Name: "Concurrent Pipeline",
			})
			done <- true
		}(i)
	}

	// Concurrent reads
	for i := 0; i < 50; i++ {
		go func() {
			svc.ListPipelines()
			done <- true
		}()
	}

	for i := 0; i < 100; i++ {
		<-done
	}
}

func TestService_ConcurrentPolicyTesting(t *testing.T) {
	cfg := DefaultConfig()
	logger := *logging.New("test")
	svc := NewService(cfg, logger)

	policy := &Policy{
		ID:      "concurrent-policy",
		Enabled: true,
		Rules: []PolicyRule{
			{ID: "r1", Pattern: "test", Action: "warn"},
		},
	}

	done := make(chan bool, 100)

	for i := 0; i < 100; i++ {
		go func() {
			_, err := svc.TestPolicy(policy, "This is a test message")
			done <- (err == nil)
		}()
	}

	successCount := 0
	for i := 0; i < 100; i++ {
		if <-done {
			successCount++
		}
	}

	if successCount != 100 {
		t.Errorf("Expected 100 successful tests, got %d", successCount)
	}
}
