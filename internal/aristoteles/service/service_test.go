// Package service provides tests for the Aristoteles service
package service

import (
	"context"
	"testing"
	"time"

	pb "github.com/msto63/mDW/api/gen/aristoteles"
)

func TestNewService(t *testing.T) {
	cfg := DefaultConfig()
	svc := NewService(cfg, nil) // nil clients for unit test

	if svc == nil {
		t.Fatal("NewService returned nil")
	}

	if svc.config == nil {
		t.Error("Service config is nil")
	}

	if svc.engine == nil {
		t.Error("Service engine is nil")
	}

	if svc.intent == nil {
		t.Error("Service intent analyzer is nil")
	}

	if svc.strategy == nil {
		t.Error("Service strategy selector is nil")
	}
}

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.MaxIterations != 3 {
		t.Errorf("Expected MaxIterations=3, got %d", cfg.MaxIterations)
	}

	if cfg.QualityThreshold != 0.8 {
		t.Errorf("Expected QualityThreshold=0.8, got %f", cfg.QualityThreshold)
	}

	if !cfg.EnableWebSearch {
		t.Error("Expected EnableWebSearch=true")
	}

	if !cfg.EnableRAG {
		t.Error("Expected EnableRAG=true")
	}
}

func TestServiceStats(t *testing.T) {
	cfg := DefaultConfig()
	svc := NewService(cfg, nil)

	stats := svc.Stats()

	if stats == nil {
		t.Fatal("Stats returned nil")
	}

	if stats["max_iterations"] != cfg.MaxIterations {
		t.Error("Stats max_iterations mismatch")
	}

	if stats["enable_web_search"] != cfg.EnableWebSearch {
		t.Error("Stats enable_web_search mismatch")
	}
}

func TestServiceGetConfig(t *testing.T) {
	cfg := DefaultConfig()
	svc := NewService(cfg, nil)

	configResp := svc.GetConfig()

	if configResp == nil {
		t.Fatal("GetConfig returned nil")
	}

	if configResp.MaxIterations != int32(cfg.MaxIterations) {
		t.Errorf("Expected MaxIterations=%d, got %d", cfg.MaxIterations, configResp.MaxIterations)
	}

	if configResp.QualityThreshold != cfg.QualityThreshold {
		t.Errorf("Expected QualityThreshold=%f, got %f", cfg.QualityThreshold, configResp.QualityThreshold)
	}
}

func TestServiceUpdateConfig(t *testing.T) {
	cfg := DefaultConfig()
	svc := NewService(cfg, nil)

	newMaxIterations := int32(5)
	newQualityThreshold := float32(0.9)

	req := &pb.UpdateConfigRequest{
		MaxIterations:    &newMaxIterations,
		QualityThreshold: &newQualityThreshold,
	}

	resp := svc.UpdateConfig(req)

	if resp.MaxIterations != newMaxIterations {
		t.Errorf("Expected MaxIterations=%d, got %d", newMaxIterations, resp.MaxIterations)
	}

	if resp.QualityThreshold != newQualityThreshold {
		t.Errorf("Expected QualityThreshold=%f, got %f", newQualityThreshold, resp.QualityThreshold)
	}
}

func TestServiceListStrategies(t *testing.T) {
	cfg := DefaultConfig()
	svc := NewService(cfg, nil)

	strategies := svc.ListStrategies()

	if len(strategies) == 0 {
		t.Error("Expected at least one strategy")
	}

	// Check that we have the basic strategies
	foundDirectLLM := false
	foundCodeGen := false

	for _, s := range strategies {
		if s.Id == "direct_llm" {
			foundDirectLLM = true
		}
		if s.Id == "code_generation" {
			foundCodeGen = true
		}
	}

	if !foundDirectLLM {
		t.Error("Expected direct_llm strategy")
	}

	if !foundCodeGen {
		t.Error("Expected code_generation strategy")
	}
}

func TestServiceGetStrategy(t *testing.T) {
	cfg := DefaultConfig()
	svc := NewService(cfg, nil)

	// Test existing strategy
	strategy, found := svc.GetStrategy("direct_llm")
	if !found {
		t.Error("Expected to find direct_llm strategy")
	}
	if strategy == nil {
		t.Error("Strategy should not be nil")
	}

	// Test non-existing strategy
	_, found = svc.GetStrategy("non_existent")
	if found {
		t.Error("Should not find non_existent strategy")
	}
}

func TestServiceClose(t *testing.T) {
	cfg := DefaultConfig()
	svc := NewService(cfg, nil)

	// Should not panic with nil clients
	err := svc.Close()
	if err != nil {
		t.Errorf("Close returned error: %v", err)
	}
}

func TestServiceProcessWithoutClients(t *testing.T) {
	cfg := DefaultConfig()
	svc := NewService(cfg, nil)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req := &pb.ProcessRequest{
		RequestId: "test-123",
		Prompt:    "Hello, how are you?",
	}

	// Without clients, this should fail gracefully or return an error
	resp, err := svc.Process(ctx, req)

	// The service should handle missing clients gracefully
	// Either return an error or a default response
	if err != nil {
		// This is acceptable - no clients configured
		t.Logf("Process returned expected error: %v", err)
	} else if resp != nil {
		t.Logf("Process returned response: %s", resp.Response)
	}
}

func TestServiceGetPipelineStatusNotFound(t *testing.T) {
	cfg := DefaultConfig()
	svc := NewService(cfg, nil)

	_, found := svc.GetPipelineStatus("non-existent-id")
	if found {
		t.Error("Should not find non-existent pipeline")
	}
}

func TestServiceCancelPipelineNotFound(t *testing.T) {
	cfg := DefaultConfig()
	svc := NewService(cfg, nil)

	cancelled := svc.CancelPipeline("non-existent-id")
	if cancelled {
		t.Error("Should not cancel non-existent pipeline")
	}
}
