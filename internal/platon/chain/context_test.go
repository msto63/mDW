package chain

import (
	"context"
	"sync"
	"testing"
	"time"
)

func TestNewProcessingContext(t *testing.T) {
	ctx := context.Background()
	pctx := NewProcessingContext(ctx, "req123", "pipe456", "test prompt")

	if pctx.RequestID != "req123" {
		t.Errorf("expected RequestID 'req123', got '%s'", pctx.RequestID)
	}

	if pctx.PipelineID != "pipe456" {
		t.Errorf("expected PipelineID 'pipe456', got '%s'", pctx.PipelineID)
	}

	if pctx.Prompt != "test prompt" {
		t.Errorf("expected Prompt 'test prompt', got '%s'", pctx.Prompt)
	}

	if pctx.Metadata == nil {
		t.Error("expected Metadata to be initialized")
	}

	if pctx.State == nil {
		t.Error("expected State to be initialized")
	}

	if pctx.StartTime.IsZero() {
		t.Error("expected StartTime to be set")
	}
}

func TestNewProcessingContext_GeneratesRequestID(t *testing.T) {
	ctx := context.Background()
	pctx := NewProcessingContext(ctx, "", "pipe", "prompt")

	if pctx.RequestID == "" {
		t.Error("expected RequestID to be auto-generated")
	}
}

func TestProcessingContext_Context(t *testing.T) {
	ctx := context.Background()
	pctx := NewProcessingContext(ctx, "req", "pipe", "prompt")

	if pctx.Context() != ctx {
		t.Error("Context() returned different context")
	}
}

func TestProcessingContext_State(t *testing.T) {
	pctx := NewProcessingContext(context.Background(), "req", "pipe", "prompt")

	// Set and Get
	pctx.SetState("key1", "value1")
	val, ok := pctx.GetState("key1")
	if !ok {
		t.Error("expected to find key1")
	}
	if val != "value1" {
		t.Errorf("expected 'value1', got '%v'", val)
	}

	// Get non-existing
	_, ok = pctx.GetState("nonexistent")
	if ok {
		t.Error("expected not to find nonexistent key")
	}
}

func TestProcessingContext_GetStateString(t *testing.T) {
	pctx := NewProcessingContext(context.Background(), "req", "pipe", "prompt")

	// String value
	pctx.SetState("str", "hello")
	if pctx.GetStateString("str") != "hello" {
		t.Errorf("expected 'hello', got '%s'", pctx.GetStateString("str"))
	}

	// Non-string value
	pctx.SetState("int", 123)
	if pctx.GetStateString("int") != "" {
		t.Error("expected empty string for non-string value")
	}

	// Non-existing
	if pctx.GetStateString("missing") != "" {
		t.Error("expected empty string for missing key")
	}
}

func TestProcessingContext_GetStateBool(t *testing.T) {
	pctx := NewProcessingContext(context.Background(), "req", "pipe", "prompt")

	// Bool value
	pctx.SetState("bool", true)
	if !pctx.GetStateBool("bool") {
		t.Error("expected true")
	}

	// Non-bool value
	pctx.SetState("str", "true")
	if pctx.GetStateBool("str") {
		t.Error("expected false for non-bool value")
	}

	// Non-existing
	if pctx.GetStateBool("missing") {
		t.Error("expected false for missing key")
	}
}

func TestProcessingContext_Metadata(t *testing.T) {
	pctx := NewProcessingContext(context.Background(), "req", "pipe", "prompt")

	// Set and Get
	pctx.SetMetadata("user_id", "user123")
	val, ok := pctx.GetMetadata("user_id")
	if !ok {
		t.Error("expected to find user_id")
	}
	if val != "user123" {
		t.Errorf("expected 'user123', got '%v'", val)
	}

	// Get non-existing
	_, ok = pctx.GetMetadata("nonexistent")
	if ok {
		t.Error("expected not to find nonexistent key")
	}
}

func TestProcessingContext_Block(t *testing.T) {
	pctx := NewProcessingContext(context.Background(), "req", "pipe", "prompt")

	if pctx.Blocked {
		t.Error("expected Blocked to be false initially")
	}

	pctx.Block("security violation")

	if !pctx.Blocked {
		t.Error("expected Blocked to be true after Block()")
	}

	if pctx.BlockReason != "security violation" {
		t.Errorf("expected BlockReason 'security violation', got '%s'", pctx.BlockReason)
	}
}

func TestProcessingContext_MarkModified(t *testing.T) {
	pctx := NewProcessingContext(context.Background(), "req", "pipe", "prompt")

	if pctx.Modified {
		t.Error("expected Modified to be false initially")
	}

	pctx.MarkModified()

	if !pctx.Modified {
		t.Error("expected Modified to be true after MarkModified()")
	}
}

func TestProcessingContext_AuditLog(t *testing.T) {
	pctx := NewProcessingContext(context.Background(), "req", "pipe", "prompt")

	if len(pctx.AuditLog) != 0 {
		t.Error("expected empty audit log initially")
	}

	entry := AuditEntry{
		Handler:  "test-handler",
		Phase:    PhasePre,
		Duration: 10 * time.Millisecond,
	}
	pctx.AddAuditEntry(entry)

	if len(pctx.AuditLog) != 1 {
		t.Errorf("expected 1 audit entry, got %d", len(pctx.AuditLog))
	}

	if pctx.AuditLog[0].Handler != "test-handler" {
		t.Errorf("expected handler 'test-handler', got '%s'", pctx.AuditLog[0].Handler)
	}
}

func TestProcessingContext_Duration(t *testing.T) {
	pctx := NewProcessingContext(context.Background(), "req", "pipe", "prompt")

	time.Sleep(50 * time.Millisecond)

	d := pctx.Duration()
	if d < 50*time.Millisecond {
		t.Errorf("expected duration >= 50ms, got %v", d)
	}
}

func TestProcessingContext_CurrentText(t *testing.T) {
	pctx := NewProcessingContext(context.Background(), "req", "pipe", "my prompt")
	pctx.Response = "my response"

	// Pre phase
	pctx.Phase = PhasePre
	if pctx.CurrentText() != "my prompt" {
		t.Errorf("expected 'my prompt' in pre phase, got '%s'", pctx.CurrentText())
	}

	// Post phase
	pctx.Phase = PhasePost
	if pctx.CurrentText() != "my response" {
		t.Errorf("expected 'my response' in post phase, got '%s'", pctx.CurrentText())
	}
}

func TestProcessingContext_SetCurrentText(t *testing.T) {
	pctx := NewProcessingContext(context.Background(), "req", "pipe", "original prompt")
	pctx.Response = "original response"

	// Pre phase
	pctx.Phase = PhasePre
	pctx.SetCurrentText("modified prompt")
	if pctx.Prompt != "modified prompt" {
		t.Errorf("expected Prompt 'modified prompt', got '%s'", pctx.Prompt)
	}

	// Post phase
	pctx.Phase = PhasePost
	pctx.SetCurrentText("modified response")
	if pctx.Response != "modified response" {
		t.Errorf("expected Response 'modified response', got '%s'", pctx.Response)
	}
}

func TestProcessingContext_ToResult(t *testing.T) {
	pctx := NewProcessingContext(context.Background(), "req123", "pipe456", "prompt")
	pctx.Response = "response"
	pctx.Block("blocked")
	pctx.MarkModified()
	pctx.SetMetadata("key", "value")
	pctx.AddAuditEntry(AuditEntry{Handler: "h1"})

	result := pctx.ToResult()

	if result.RequestID != "req123" {
		t.Errorf("expected RequestID 'req123', got '%s'", result.RequestID)
	}

	if result.ProcessedPrompt != "prompt" {
		t.Errorf("expected ProcessedPrompt 'prompt', got '%s'", result.ProcessedPrompt)
	}

	if result.ProcessedResponse != "response" {
		t.Errorf("expected ProcessedResponse 'response', got '%s'", result.ProcessedResponse)
	}

	if !result.Blocked {
		t.Error("expected Blocked to be true")
	}

	if result.BlockReason != "blocked" {
		t.Errorf("expected BlockReason 'blocked', got '%s'", result.BlockReason)
	}

	if !result.Modified {
		t.Error("expected Modified to be true")
	}

	if len(result.AuditLog) != 1 {
		t.Errorf("expected 1 audit entry, got %d", len(result.AuditLog))
	}

	if result.Metadata["key"] != "value" {
		t.Error("expected metadata to be copied")
	}
}

func TestProcessingContext_Clone(t *testing.T) {
	original := NewProcessingContext(context.Background(), "req", "pipe", "prompt")
	original.Response = "response"
	original.Phase = PhasePost
	original.Block("reason")
	original.SetState("key", "value")
	original.SetMetadata("meta", "data")

	cloned := original.Clone()

	// Verify values are copied
	if cloned.RequestID != original.RequestID {
		t.Error("RequestID not cloned")
	}

	if cloned.Prompt != original.Prompt {
		t.Error("Prompt not cloned")
	}

	if cloned.Response != original.Response {
		t.Error("Response not cloned")
	}

	if cloned.Phase != original.Phase {
		t.Error("Phase not cloned")
	}

	if cloned.Blocked != original.Blocked {
		t.Error("Blocked not cloned")
	}

	// Verify state is independent
	cloned.SetState("key", "modified")
	val, _ := original.GetState("key")
	if val != "value" {
		t.Error("modifying cloned state affected original")
	}

	// Verify metadata is independent
	cloned.SetMetadata("meta", "modified")
	mval, _ := original.GetMetadata("meta")
	if mval != "data" {
		t.Error("modifying cloned metadata affected original")
	}
}

func TestProcessingContext_ConcurrentAccess(t *testing.T) {
	pctx := NewProcessingContext(context.Background(), "req", "pipe", "prompt")

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(3)

		// Writer for state
		go func(i int) {
			defer wg.Done()
			pctx.SetState("key", i)
		}(i)

		// Reader for state
		go func() {
			defer wg.Done()
			pctx.GetState("key")
		}()

		// Writer for audit log
		go func() {
			defer wg.Done()
			pctx.AddAuditEntry(AuditEntry{Handler: "concurrent"})
		}()
	}

	wg.Wait()
	// Test passes if no race conditions or panics
}
