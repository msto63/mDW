// ============================================================================
// meinDENKWERK (mDW) - Lokale KI-Plattform
// ============================================================================
//
// Package:     grpc
// Description: Integration test helpers for gRPC services
// Author:      Mike Stoffels with Claude
// Created:     2025-12-06
// License:     MIT
// ============================================================================

package grpc

import (
	"context"
	"fmt"
	"os"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// ServiceConfig holds configuration for a gRPC service
type ServiceConfig struct {
	Name    string
	Host    string
	Port    int
	Timeout time.Duration
}

// DefaultServiceConfigs returns default configurations for all services
func DefaultServiceConfigs() map[string]ServiceConfig {
	return map[string]ServiceConfig{
		"turing": {
			Name:    "turing",
			Host:    getEnvOrDefault("TURING_HOST", "localhost"),
			Port:    getEnvOrDefaultInt("TURING_PORT", 9200),
			Timeout: 30 * time.Second,
		},
		"russell": {
			Name:    "russell",
			Host:    getEnvOrDefault("RUSSELL_HOST", "localhost"),
			Port:    getEnvOrDefaultInt("RUSSELL_PORT", 9100),
			Timeout: 10 * time.Second,
		},
		"hypatia": {
			Name:    "hypatia",
			Host:    getEnvOrDefault("HYPATIA_HOST", "localhost"),
			Port:    getEnvOrDefaultInt("HYPATIA_PORT", 9220),
			Timeout: 30 * time.Second,
		},
		"babbage": {
			Name:    "babbage",
			Host:    getEnvOrDefault("BABBAGE_HOST", "localhost"),
			Port:    getEnvOrDefaultInt("BABBAGE_PORT", 9150),
			Timeout: 30 * time.Second,
		},
		"leibniz": {
			Name:    "leibniz",
			Host:    getEnvOrDefault("LEIBNIZ_HOST", "localhost"),
			Port:    getEnvOrDefaultInt("LEIBNIZ_PORT", 9140),
			Timeout: 60 * time.Second,
		},
		"bayes": {
			Name:    "bayes",
			Host:    getEnvOrDefault("BAYES_HOST", "localhost"),
			Port:    getEnvOrDefaultInt("BAYES_PORT", 9120),
			Timeout: 10 * time.Second,
		},
		"platon": {
			Name:    "platon",
			Host:    getEnvOrDefault("PLATON_HOST", "localhost"),
			Port:    getEnvOrDefaultInt("PLATON_PORT", 9130),
			Timeout: 30 * time.Second,
		},
	}
}

// TestConnection represents a gRPC connection for testing
type TestConnection struct {
	conn    *grpc.ClientConn
	config  ServiceConfig
	ctx     context.Context
	cancel  context.CancelFunc
}

// NewTestConnection creates a new test connection to a gRPC service
func NewTestConnection(cfg ServiceConfig) (*TestConnection, error) {
	ctx, cancel := context.WithTimeout(context.Background(), cfg.Timeout)

	addr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)
	conn, err := grpc.DialContext(ctx, addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to connect to %s at %s: %w", cfg.Name, addr, err)
	}

	return &TestConnection{
		conn:   conn,
		config: cfg,
		ctx:    ctx,
		cancel: cancel,
	}, nil
}

// Conn returns the underlying gRPC connection
func (tc *TestConnection) Conn() *grpc.ClientConn {
	return tc.conn
}

// Context returns a context with the configured timeout
func (tc *TestConnection) Context() context.Context {
	return tc.ctx
}

// ContextWithTimeout returns a new context with a custom timeout
func (tc *TestConnection) ContextWithTimeout(timeout time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), timeout)
}

// Close closes the connection and cancels the context
func (tc *TestConnection) Close() error {
	tc.cancel()
	return tc.conn.Close()
}

// ServiceName returns the service name
func (tc *TestConnection) ServiceName() string {
	return tc.config.Name
}

// Address returns the service address
func (tc *TestConnection) Address() string {
	return fmt.Sprintf("%s:%d", tc.config.Host, tc.config.Port)
}

// Helper functions

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvOrDefaultInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		var intValue int
		if _, err := fmt.Sscanf(value, "%d", &intValue); err == nil {
			return intValue
		}
	}
	return defaultValue
}

// TestResult holds the result of a test
type TestResult struct {
	Name     string
	Passed   bool
	Duration time.Duration
	Error    error
	Details  map[string]interface{}
}

// TestSuite represents a collection of tests for a service
type TestSuite struct {
	ServiceName string
	Results     []TestResult
	StartTime   time.Time
	EndTime     time.Time
}

// NewTestSuite creates a new test suite
func NewTestSuite(serviceName string) *TestSuite {
	return &TestSuite{
		ServiceName: serviceName,
		Results:     make([]TestResult, 0),
		StartTime:   time.Now(),
	}
}

// AddResult adds a test result to the suite
func (ts *TestSuite) AddResult(result TestResult) {
	ts.Results = append(ts.Results, result)
}

// Finish marks the test suite as finished
func (ts *TestSuite) Finish() {
	ts.EndTime = time.Now()
}

// Summary returns a summary of the test results
func (ts *TestSuite) Summary() string {
	passed := 0
	failed := 0
	for _, r := range ts.Results {
		if r.Passed {
			passed++
		} else {
			failed++
		}
	}

	return fmt.Sprintf("[%s] %d passed, %d failed, duration: %v",
		ts.ServiceName, passed, failed, ts.EndTime.Sub(ts.StartTime).Round(time.Millisecond))
}

// AllPassed returns true if all tests passed
func (ts *TestSuite) AllPassed() bool {
	for _, r := range ts.Results {
		if !r.Passed {
			return false
		}
	}
	return true
}
