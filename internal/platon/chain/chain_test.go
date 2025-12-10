package chain

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/msto63/mDW/pkg/core/logging"
)

// MockHandler is a test handler
type MockHandler struct {
	name        string
	htype       HandlerType
	priority    int
	processFunc func(ctx *ProcessingContext) error
	shouldProc  bool
	callCount   atomic.Int32
}

func NewMockHandler(name string, htype HandlerType, priority int) *MockHandler {
	return &MockHandler{
		name:       name,
		htype:      htype,
		priority:   priority,
		shouldProc: true,
	}
}

func (h *MockHandler) Name() string         { return h.name }
func (h *MockHandler) Type() HandlerType    { return h.htype }
func (h *MockHandler) Priority() int        { return h.priority }
func (h *MockHandler) ShouldProcess(*ProcessingContext) bool { return h.shouldProc }

func (h *MockHandler) Process(ctx *ProcessingContext) error {
	h.callCount.Add(1)
	if h.processFunc != nil {
		return h.processFunc(ctx)
	}
	return nil
}

func (h *MockHandler) CallCount() int32 {
	return h.callCount.Load()
}

func TestNewChain(t *testing.T) {
	logger := *logging.New("test")
	chain := NewChain(logger)

	if chain == nil {
		t.Fatal("NewChain returned nil")
	}

	if chain.PreHandlerCount() != 0 {
		t.Errorf("expected 0 pre handlers, got %d", chain.PreHandlerCount())
	}

	if chain.PostHandlerCount() != 0 {
		t.Errorf("expected 0 post handlers, got %d", chain.PostHandlerCount())
	}
}

func TestChain_Register(t *testing.T) {
	tests := []struct {
		name          string
		handlers      []*MockHandler
		expectedPre   int
		expectedPost  int
		expectedTotal int
	}{
		{
			name: "register pre handler",
			handlers: []*MockHandler{
				NewMockHandler("pre1", HandlerTypePre, 1),
			},
			expectedPre:   1,
			expectedPost:  0,
			expectedTotal: 1,
		},
		{
			name: "register post handler",
			handlers: []*MockHandler{
				NewMockHandler("post1", HandlerTypePost, 1),
			},
			expectedPre:   0,
			expectedPost:  1,
			expectedTotal: 1,
		},
		{
			name: "register both handler",
			handlers: []*MockHandler{
				NewMockHandler("both1", HandlerTypeBoth, 1),
			},
			expectedPre:   1,
			expectedPost:  1,
			expectedTotal: 1,
		},
		{
			name: "register multiple handlers",
			handlers: []*MockHandler{
				NewMockHandler("pre1", HandlerTypePre, 1),
				NewMockHandler("post1", HandlerTypePost, 2),
				NewMockHandler("both1", HandlerTypeBoth, 3),
			},
			expectedPre:   2,
			expectedPost:  2,
			expectedTotal: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := *logging.New("test")
			chain := NewChain(logger)

			for _, h := range tt.handlers {
				chain.Register(h)
			}

			if chain.PreHandlerCount() != tt.expectedPre {
				t.Errorf("expected %d pre handlers, got %d", tt.expectedPre, chain.PreHandlerCount())
			}

			if chain.PostHandlerCount() != tt.expectedPost {
				t.Errorf("expected %d post handlers, got %d", tt.expectedPost, chain.PostHandlerCount())
			}

			if chain.TotalHandlerCount() != tt.expectedTotal {
				t.Errorf("expected %d total handlers, got %d", tt.expectedTotal, chain.TotalHandlerCount())
			}
		})
	}
}

func TestChain_Priority(t *testing.T) {
	logger := *logging.New("test")
	chain := NewChain(logger)

	var executionOrder []string

	h1 := NewMockHandler("high", HandlerTypePre, 10)
	h1.processFunc = func(_ *ProcessingContext) error {
		executionOrder = append(executionOrder, "high")
		return nil
	}

	h2 := NewMockHandler("low", HandlerTypePre, 1)
	h2.processFunc = func(_ *ProcessingContext) error {
		executionOrder = append(executionOrder, "low")
		return nil
	}

	h3 := NewMockHandler("medium", HandlerTypePre, 5)
	h3.processFunc = func(_ *ProcessingContext) error {
		executionOrder = append(executionOrder, "medium")
		return nil
	}

	// Register in random order
	chain.Register(h1)
	chain.Register(h2)
	chain.Register(h3)

	ctx := NewProcessingContext(context.Background(), "req1", "pipe1", "test prompt")
	err := chain.ProcessPre(ctx)
	if err != nil {
		t.Fatalf("ProcessPre failed: %v", err)
	}

	// Should be sorted by priority (lower first)
	expected := []string{"low", "medium", "high"}
	if len(executionOrder) != len(expected) {
		t.Fatalf("expected %d executions, got %d", len(expected), len(executionOrder))
	}

	for i, name := range expected {
		if executionOrder[i] != name {
			t.Errorf("position %d: expected %s, got %s", i, name, executionOrder[i])
		}
	}
}

func TestChain_Unregister(t *testing.T) {
	logger := *logging.New("test")
	chain := NewChain(logger)

	h1 := NewMockHandler("handler1", HandlerTypePre, 1)
	h2 := NewMockHandler("handler2", HandlerTypePre, 2)

	chain.Register(h1)
	chain.Register(h2)

	if chain.PreHandlerCount() != 2 {
		t.Errorf("expected 2 handlers, got %d", chain.PreHandlerCount())
	}

	// Unregister existing handler
	removed := chain.Unregister("handler1")
	if !removed {
		t.Error("expected Unregister to return true")
	}

	if chain.PreHandlerCount() != 1 {
		t.Errorf("expected 1 handler after unregister, got %d", chain.PreHandlerCount())
	}

	// Unregister non-existing handler
	removed = chain.Unregister("nonexistent")
	if removed {
		t.Error("expected Unregister to return false for non-existent handler")
	}
}

func TestChain_GetHandler(t *testing.T) {
	logger := *logging.New("test")
	chain := NewChain(logger)

	h := NewMockHandler("testhandler", HandlerTypePre, 1)
	chain.Register(h)

	// Get existing handler
	found, ok := chain.GetHandler("testhandler")
	if !ok {
		t.Error("expected to find handler")
	}
	if found.Name() != "testhandler" {
		t.Errorf("expected handler name 'testhandler', got '%s'", found.Name())
	}

	// Get non-existing handler
	_, ok = chain.GetHandler("nonexistent")
	if ok {
		t.Error("expected not to find non-existent handler")
	}
}

func TestChain_ListHandlers(t *testing.T) {
	logger := *logging.New("test")
	chain := NewChain(logger)

	chain.Register(NewMockHandler("h1", HandlerTypePre, 1))
	chain.Register(NewMockHandler("h2", HandlerTypePost, 2))
	chain.Register(NewMockHandler("h3", HandlerTypeBoth, 3))

	handlers := chain.ListHandlers()

	if len(handlers) != 3 {
		t.Errorf("expected 3 handlers, got %d", len(handlers))
	}

	// Check that both handler appears only once
	names := make(map[string]bool)
	for _, h := range handlers {
		if names[h.Name] {
			t.Errorf("handler %s appears more than once", h.Name)
		}
		names[h.Name] = true
	}
}

func TestChain_ProcessPre(t *testing.T) {
	logger := *logging.New("test")
	chain := NewChain(logger)

	h := NewMockHandler("pre", HandlerTypePre, 1)
	h.processFunc = func(ctx *ProcessingContext) error {
		ctx.Prompt = ctx.Prompt + " modified"
		ctx.MarkModified()
		return nil
	}
	chain.Register(h)

	ctx := NewProcessingContext(context.Background(), "req1", "pipe1", "original")
	err := chain.ProcessPre(ctx)
	if err != nil {
		t.Fatalf("ProcessPre failed: %v", err)
	}

	if ctx.Prompt != "original modified" {
		t.Errorf("expected 'original modified', got '%s'", ctx.Prompt)
	}

	if !ctx.Modified {
		t.Error("expected Modified to be true")
	}
}

func TestChain_ProcessPost(t *testing.T) {
	logger := *logging.New("test")
	chain := NewChain(logger)

	h := NewMockHandler("post", HandlerTypePost, 1)
	h.processFunc = func(ctx *ProcessingContext) error {
		ctx.Response = ctx.Response + " processed"
		ctx.MarkModified()
		return nil
	}
	chain.Register(h)

	ctx := NewProcessingContext(context.Background(), "req1", "pipe1", "prompt")
	ctx.Response = "response"
	err := chain.ProcessPost(ctx)
	if err != nil {
		t.Fatalf("ProcessPost failed: %v", err)
	}

	if ctx.Response != "response processed" {
		t.Errorf("expected 'response processed', got '%s'", ctx.Response)
	}
}

func TestChain_BlockingHandler(t *testing.T) {
	logger := *logging.New("test")
	chain := NewChain(logger)

	blocker := NewMockHandler("blocker", HandlerTypePre, 1)
	blocker.processFunc = func(ctx *ProcessingContext) error {
		ctx.Block("blocked by test")
		return nil
	}
	chain.Register(blocker)

	afterBlocker := NewMockHandler("after", HandlerTypePre, 2)
	chain.Register(afterBlocker)

	ctx := NewProcessingContext(context.Background(), "req1", "pipe1", "test")
	err := chain.ProcessPre(ctx)
	if err != nil {
		t.Fatalf("ProcessPre failed: %v", err)
	}

	if !ctx.Blocked {
		t.Error("expected request to be blocked")
	}

	if ctx.BlockReason != "blocked by test" {
		t.Errorf("expected block reason 'blocked by test', got '%s'", ctx.BlockReason)
	}

	// After-blocker should not have been called
	if afterBlocker.CallCount() != 0 {
		t.Errorf("expected afterBlocker not to be called, but it was called %d times", afterBlocker.CallCount())
	}
}

func TestChain_HandlerError(t *testing.T) {
	logger := *logging.New("test")
	chain := NewChain(logger)

	expectedErr := errors.New("handler error")
	h := NewMockHandler("failing", HandlerTypePre, 1)
	h.processFunc = func(_ *ProcessingContext) error {
		return expectedErr
	}
	chain.Register(h)

	ctx := NewProcessingContext(context.Background(), "req1", "pipe1", "test")
	err := chain.ProcessPre(ctx)

	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !errors.Is(err, expectedErr) {
		t.Errorf("expected wrapped error to contain original, got: %v", err)
	}
}

func TestChain_ContextCancellation(t *testing.T) {
	logger := *logging.New("test")
	chain := NewChain(logger)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	pctx := NewProcessingContext(ctx, "req1", "pipe1", "test")

	h := NewMockHandler("slow", HandlerTypePre, 1)
	chain.Register(h)

	err := chain.ProcessPre(pctx)
	if err == nil {
		t.Error("expected context cancellation error")
	}
}

func TestChain_Process_FullPipeline(t *testing.T) {
	logger := *logging.New("test")
	chain := NewChain(logger)

	preHandler := NewMockHandler("pre", HandlerTypePre, 1)
	preHandler.processFunc = func(ctx *ProcessingContext) error {
		ctx.Prompt = ctx.Prompt + " [pre-processed]"
		return nil
	}
	chain.Register(preHandler)

	postHandler := NewMockHandler("post", HandlerTypePost, 1)
	postHandler.processFunc = func(ctx *ProcessingContext) error {
		ctx.Response = ctx.Response + " [post-processed]"
		return nil
	}
	chain.Register(postHandler)

	mainProcessor := func(_ context.Context, prompt string) (string, error) {
		return "Response to: " + prompt, nil
	}

	req := &ProcessRequest{
		RequestID:  "req1",
		PipelineID: "pipe1",
		Prompt:     "Hello",
	}

	result, err := chain.Process(context.Background(), req, mainProcessor)
	if err != nil {
		t.Fatalf("Process failed: %v", err)
	}

	expectedPrompt := "Hello [pre-processed]"
	if result.ProcessedPrompt != expectedPrompt {
		t.Errorf("expected prompt '%s', got '%s'", expectedPrompt, result.ProcessedPrompt)
	}

	expectedResponse := "Response to: Hello [pre-processed] [post-processed]"
	if result.ProcessedResponse != expectedResponse {
		t.Errorf("expected response '%s', got '%s'", expectedResponse, result.ProcessedResponse)
	}
}

func TestChain_AuditLog(t *testing.T) {
	logger := *logging.New("test")
	chain := NewChain(logger)

	h1 := NewMockHandler("handler1", HandlerTypePre, 1)
	h1.processFunc = func(ctx *ProcessingContext) error {
		time.Sleep(10 * time.Millisecond)
		return nil
	}
	chain.Register(h1)

	h2 := NewMockHandler("handler2", HandlerTypePre, 2)
	chain.Register(h2)

	ctx := NewProcessingContext(context.Background(), "req1", "pipe1", "test")
	err := chain.ProcessPre(ctx)
	if err != nil {
		t.Fatalf("ProcessPre failed: %v", err)
	}

	if len(ctx.AuditLog) != 2 {
		t.Fatalf("expected 2 audit entries, got %d", len(ctx.AuditLog))
	}

	// Check first handler
	if ctx.AuditLog[0].Handler != "handler1" {
		t.Errorf("expected handler name 'handler1', got '%s'", ctx.AuditLog[0].Handler)
	}
	if ctx.AuditLog[0].Phase != PhasePre {
		t.Errorf("expected phase PhasePre, got %v", ctx.AuditLog[0].Phase)
	}
	if ctx.AuditLog[0].Duration < 10*time.Millisecond {
		t.Errorf("expected duration >= 10ms, got %v", ctx.AuditLog[0].Duration)
	}
}

func TestChain_ShouldProcess(t *testing.T) {
	logger := *logging.New("test")
	chain := NewChain(logger)

	skipHandler := NewMockHandler("skipper", HandlerTypePre, 1)
	skipHandler.shouldProc = false
	chain.Register(skipHandler)

	ctx := NewProcessingContext(context.Background(), "req1", "pipe1", "test")
	err := chain.ProcessPre(ctx)
	if err != nil {
		t.Fatalf("ProcessPre failed: %v", err)
	}

	if skipHandler.CallCount() != 0 {
		t.Errorf("expected handler not to be called, but it was called %d times", skipHandler.CallCount())
	}
}
