package handlers

import (
	"context"
	"testing"

	"github.com/msto63/mDW/internal/platon/chain"
)

func TestBaseHandler(t *testing.T) {
	h := NewBaseHandler("test-handler", chain.HandlerTypePre, 10)

	if h.Name() != "test-handler" {
		t.Errorf("expected name 'test-handler', got '%s'", h.Name())
	}

	if h.Type() != chain.HandlerTypePre {
		t.Errorf("expected type HandlerTypePre, got %v", h.Type())
	}

	if h.Priority() != 10 {
		t.Errorf("expected priority 10, got %d", h.Priority())
	}

	if !h.IsEnabled() {
		t.Error("expected handler to be enabled by default")
	}

	ctx := chain.NewProcessingContext(context.Background(), "req", "pipe", "prompt")
	if !h.ShouldProcess(ctx) {
		t.Error("expected ShouldProcess to return true when enabled")
	}

	h.SetEnabled(false)
	if h.IsEnabled() {
		t.Error("expected handler to be disabled after SetEnabled(false)")
	}

	if h.ShouldProcess(ctx) {
		t.Error("expected ShouldProcess to return false when disabled")
	}
}

func TestBaseHandler_Types(t *testing.T) {
	tests := []struct {
		name     string
		htype    chain.HandlerType
		expected string
	}{
		{"pre handler", chain.HandlerTypePre, "pre"},
		{"post handler", chain.HandlerTypePost, "post"},
		{"both handler", chain.HandlerTypeBoth, "both"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := NewBaseHandler("test", tt.htype, 1)
			if h.Type() != tt.htype {
				t.Errorf("expected type %v, got %v", tt.htype, h.Type())
			}
		})
	}
}
