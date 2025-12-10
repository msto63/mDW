// ============================================================================
// meinDENKWERK (mDW) - Lokale KI-Plattform
// ============================================================================
//
// Package:     grpc
// Description: Main integration test runner for all gRPC services
// Author:      Mike Stoffels with Claude
// Created:     2025-12-06
// License:     MIT
// ============================================================================

//go:build integration

package grpc

import (
	"flag"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/msto63/mDW/api/gen/common"
	turingpb "github.com/msto63/mDW/api/gen/turing"
)

var (
	// Command line flags for controlling test behavior
	verbose     = flag.Bool("verbose", false, "Enable verbose output")
	skipSlowTests = flag.Bool("skip-slow", false, "Skip slow tests (e.g., chat tests)")
	serviceFilter = flag.String("service", "", "Run tests only for specific service (turing, russell)")
)

// TestMain is the entry point for integration tests
func TestMain(m *testing.M) {
	flag.Parse()

	// Print test environment info
	if *verbose {
		fmt.Println("=== mDW Integration Tests ===")
		fmt.Printf("Time: %s\n", time.Now().Format(time.RFC3339))
		fmt.Println()
	}

	// Run the tests
	code := m.Run()

	os.Exit(code)
}

// TestAllServices runs a quick connectivity test for all services
func TestAllServices(t *testing.T) {
	configs := DefaultServiceConfigs()

	for name, cfg := range configs {
		t.Run(name, func(t *testing.T) {
			conn, err := NewTestConnection(cfg)
			if err != nil {
				t.Logf("Service %s not reachable: %v", name, err)
				return
			}
			defer conn.Close()

			t.Logf("Service %s is reachable at %s", name, conn.Address())
		})
	}
}

// TestTuringService runs all Turing service integration tests
func TestTuringService(t *testing.T) {
	if *serviceFilter != "" && *serviceFilter != "turing" {
		t.Skip("Skipping turing tests (filtered)")
	}

	// Check if service is available
	client, err := NewTuringTestClient()
	if err != nil {
		t.Skipf("Turing service not available: %v", err)
	}
	client.Close()

	// Run the test suite
	RunTuringTestSuite(t)
}

// TestRussellService runs all Russell service integration tests
func TestRussellService(t *testing.T) {
	if *serviceFilter != "" && *serviceFilter != "russell" {
		t.Skip("Skipping russell tests (filtered)")
	}

	// Check if service is available
	client, err := NewRussellTestClient()
	if err != nil {
		t.Skipf("Russell service not available: %v", err)
	}
	client.Close()

	// Run the test suite
	RunRussellTestSuite(t)
}

// TestPlatonService runs all Platon service integration tests
func TestPlatonService(t *testing.T) {
	if *serviceFilter != "" && *serviceFilter != "platon" {
		t.Skip("Skipping platon tests (filtered)")
	}

	// Check if service is available
	client, err := NewPlatonTestClient()
	if err != nil {
		t.Skipf("Platon service not available: %v", err)
	}
	client.Close()

	// Run the test suite
	RunPlatonTestSuite(t)
}

// TestBabbageService runs all Babbage service integration tests
func TestBabbageService(t *testing.T) {
	if *serviceFilter != "" && *serviceFilter != "babbage" {
		t.Skip("Skipping babbage tests (filtered)")
	}

	// Check if service is available
	client, err := NewBabbageTestClient()
	if err != nil {
		t.Skipf("Babbage service not available: %v", err)
	}
	client.Close()

	// Run the test suite
	RunBabbageTestSuite(t)
}

// TestQuickHealthChecks runs just the health check tests for all services
func TestQuickHealthChecks(t *testing.T) {
	t.Run("TuringHealth", TestTuringHealthCheck)
	t.Run("RussellHealth", TestRussellHealthCheck)
	t.Run("PlatonHealth", TestPlatonHealthCheck)
	t.Run("BabbageHealth", TestBabbageHealthCheck)
}

// BenchmarkTuringChat benchmarks the Turing chat endpoint
func BenchmarkTuringChat(b *testing.B) {
	client, err := NewTuringTestClient()
	if err != nil {
		b.Skipf("Turing service not available: %v", err)
	}
	defer client.Close()

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		ctx, cancel := client.ContextWithTimeout(60 * time.Second)

		// Simple chat request for benchmarking
		_, err := client.Client().Chat(ctx, &turingpb.ChatRequest{
			Model: "ollama:mistral:7b",
			Messages: []*turingpb.Message{
				{Role: "user", Content: "Say hello in one word."},
			},
			Temperature: 0.7,
			MaxTokens:   10,
		})

		cancel()

		if err != nil {
			b.Fatalf("Chat failed: %v", err)
		}
	}
}

// BenchmarkRussellListServices benchmarks Russell's ListServices endpoint
func BenchmarkRussellListServices(b *testing.B) {
	client, err := NewRussellTestClient()
	if err != nil {
		b.Skipf("Russell service not available: %v", err)
	}
	defer client.Close()

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		ctx, cancel := client.ContextWithTimeout(10 * time.Second)

		_, err := client.Client().ListServices(ctx, &common.Empty{})
		cancel()

		if err != nil {
			b.Fatalf("ListServices failed: %v", err)
		}
	}
}
