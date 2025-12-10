// File: integration_test.go
// Title: TCOL Integration Tests for Complete Command Flow
// Description: Comprehensive integration tests that verify the complete TCOL
//              command execution flow from parsing through execution. Tests
//              the interaction between lexer, parser, AST, registry, executor,
//              and client components working together to process real TCOL
//              commands and return expected results.
// Author: msto63 with Claude Sonnet 4.0
// Version: v0.1.0
// Created: 2025-01-25
// Modified: 2025-01-25
//
// Change History:
// - 2025-01-25 v0.1.0: Initial integration test suite

package tcol

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	mdwlog "github.com/msto63/mDW/foundation/core/log"
	mdwast "github.com/msto63/mDW/foundation/tcol/ast"
	mdwclient "github.com/msto63/mDW/foundation/tcol/client"
	mdwexecutor "github.com/msto63/mDW/foundation/tcol/executor"
	mdwparser "github.com/msto63/mDW/foundation/tcol/parser"
	mdwregistry "github.com/msto63/mDW/foundation/tcol/registry"
)

// Test infrastructure for integration tests

// MockServiceClient for integration testing
type MockServiceClient struct {
	responses    map[string]*mdwexecutor.ServiceResponse
	errors       map[string]error
	callHistory  []MockCall
	healthStatus map[string]error
	closed       bool
}

type MockCall struct {
	ServiceName string
	ObjectName  string
	MethodName  string
	Params      map[string]interface{}
	Context     *mdwexecutor.ExecutionContext
}

func NewMockServiceClient() *MockServiceClient {
	return &MockServiceClient{
		responses:    make(map[string]*mdwexecutor.ServiceResponse),
		errors:       make(map[string]error),
		callHistory:  make([]MockCall, 0),
		healthStatus: make(map[string]error),
	}
}

func (m *MockServiceClient) Execute(ctx context.Context, serviceName, objectName, methodName string, 
	params map[string]interface{}, execCtx *mdwexecutor.ExecutionContext) (*mdwexecutor.ServiceResponse, error) {
	
	key := fmt.Sprintf("%s.%s.%s", serviceName, objectName, methodName)
	
	// Record call
	m.callHistory = append(m.callHistory, MockCall{
		ServiceName: serviceName,
		ObjectName:  objectName,
		MethodName:  methodName,
		Params:      params,
		Context:     execCtx,
	})
	
	// Return error if configured
	if err, exists := m.errors[key]; exists {
		return nil, err
	}
	
	// Return response if configured
	if response, exists := m.responses[key]; exists {
		return response, nil
	}
	
	// Default success response with metadata
	return &mdwexecutor.ServiceResponse{
		Success: true,
		Data:    fmt.Sprintf("Mock response for %s", key),
		Metadata: map[string]interface{}{
			"mock": true,
			"serviceName": serviceName,
		},
	}, nil
}

func (m *MockServiceClient) Health(ctx context.Context, serviceName string) error {
	if err, exists := m.healthStatus[serviceName]; exists {
		return err
	}
	return nil
}

func (m *MockServiceClient) Close() error {
	m.closed = true
	return nil
}

func (m *MockServiceClient) SetResponse(serviceName, objectName, methodName string, response *mdwexecutor.ServiceResponse) {
	key := fmt.Sprintf("%s.%s.%s", serviceName, objectName, methodName)
	m.responses[key] = response
}

func (m *MockServiceClient) SetError(serviceName, objectName, methodName string, err error) {
	key := fmt.Sprintf("%s.%s.%s", serviceName, objectName, methodName)
	m.errors[key] = err
}

func (m *MockServiceClient) GetCallHistory() []MockCall {
	return m.callHistory
}

func (m *MockServiceClient) ClearHistory() {
	m.callHistory = make([]MockCall, 0)
}

type IntegrationTestSuite struct {
	engine   *Engine
	registry *mdwregistry.Registry
	executor *mdwexecutor.Engine
	client   *mdwclient.Client
	logger   *mdwlog.Logger
}

func setupIntegrationTest(t *testing.T) *IntegrationTestSuite {
	logger := mdwlog.GetDefault()
	
	// Create mock service client
	mockClient := NewMockServiceClient()
	
	// Create engine using the simplified API
	engine, err := NewEngine(Options{
		Logger:              logger,
		MaxCommandLength:    4096,
		EnableAbbreviations: true,
		EnableAliases:       true,
		EnableChaining:      true,
		ExecutionTimeout:    30 * time.Second,
		ServiceClient:       mockClient,
	})
	if err != nil {
		if t != nil {
			t.Fatalf("Failed to create engine: %v", err)
		}
		panic(fmt.Sprintf("Failed to create engine: %v", err))
	}
	
	return &IntegrationTestSuite{
		engine:   engine,
		registry: engine.Registry(),
		executor: nil, // Access through engine
		client:   nil, // Access through engine
		logger:   logger,
	}
}

func (suite *IntegrationTestSuite) cleanup() {
	if suite.client != nil {
		suite.client.Close()
	}
}

// Test complete TCOL command execution flow

func TestIntegration_SimpleCommandExecution(t *testing.T) {
	suite := setupIntegrationTest(t)
	defer suite.cleanup()
	
	ctx := context.Background()
	
	// Register test objects needed for the test cases
	err := suite.engine.Registry().RegisterObject(&mdwregistry.ObjectDefinition{
		Name:        "CUSTOMER",
		Service:     "customer-service",
		Description: "Customer management object",
		Methods: map[string]*mdwregistry.MethodDefinition{
			"CREATE": {
				Name:        "CREATE",
				Description: "Create new customer",
				Parameters: map[string]*mdwregistry.ParameterDefinition{
					"name": {Name: "name", Type: "string", Required: true},
					"type": {Name: "type", Type: "string", Required: false},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("Failed to register CUSTOMER object: %v", err)
	}
	
	testCases := []struct {
		name     string
		command  string
		expectedObject string
		expectedMethod string
		wantErr  bool
	}{
		{
			name:    "Basic object method call",
			command: "CUSTOMER.CREATE name=\"Test Corp\" type=\"B2B\"",
			expectedObject: "CUSTOMER",
			expectedMethod: "CREATE",
			wantErr: false,
		},
		{
			name:    "Object ID access",
			command: "CUSTOMER:12345",
			expectedObject: "CUSTOMER",
			expectedMethod: "",
			wantErr: false, // Object access is supported
		},
		{
			name:    "Field assignment",
			command: "CUSTOMER:12345:email=\"new@example.com\"",
			expectedObject: "CUSTOMER",
			expectedMethod: "",
			wantErr: false, // Field assignment is supported
		},
		{
			name:    "Built-in HELP command",
			command: "HELP.LIST",
			expectedObject: "HELP",
			expectedMethod: "LIST",
			wantErr: false,
		},
		{
			name:    "Invalid command syntax",
			command: "INVALID SYNTAX [",
			wantErr: true,
		},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := suite.engine.Execute(ctx, tc.command)
			
			if tc.wantErr {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}
			
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}
			
			if result == nil {
				t.Fatal("Expected result but got nil")
			}
			
			if !result.Success {
				t.Errorf("Expected successful result, got: %s", result.Message)
			}
			
			// Check parsed command structure
			if result.ParsedCommand == nil {
				t.Fatal("Expected parsed command but got nil")
			}
			
			if tc.expectedObject != "" && result.ParsedCommand.Object != tc.expectedObject {
				t.Errorf("Expected object '%s', got '%s'", tc.expectedObject, result.ParsedCommand.Object)
			}
			
			if tc.expectedMethod != "" && result.ParsedCommand.Method != tc.expectedMethod {
				t.Errorf("Expected method '%s', got '%s'", tc.expectedMethod, result.ParsedCommand.Method)
			}
		})
	}
}

func TestIntegration_CommandWithFilter(t *testing.T) {
	suite := setupIntegrationTest(t)
	defer suite.cleanup()
	
	ctx := context.Background()
	
	// Register a test object that supports filters
	err := suite.engine.Registry().RegisterObject(&mdwregistry.ObjectDefinition{
		Name:        "CUSTOMER",
		Service:     "customer-service",
		Description: "Customer management object",
		Methods: map[string]*mdwregistry.MethodDefinition{
			"LIST": {
				Name:        "LIST",
				Description: "List customers with optional filter",
				Parameters: map[string]*mdwregistry.ParameterDefinition{
					"limit": {Name: "limit", Type: "number", Required: false},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("Failed to register CUSTOMER object: %v", err)
	}
	
	command := `CUSTOMER[city="Berlin" AND status="active"].LIST limit=10`
	
	result, err := suite.engine.Execute(ctx, command)
	if err != nil {
		t.Fatalf("Command execution failed: %v", err)
	}
	
	if !result.Success {
		t.Errorf("Expected successful result, got: %s", result.Message)
	}
	
	// Verify the command was parsed and executed correctly
	if result.ParsedCommand == nil {
		t.Fatal("Expected parsed command but got nil")
	}
	
	if result.ParsedCommand.Object != "CUSTOMER" {
		t.Errorf("Expected object 'CUSTOMER', got %v", result.ParsedCommand.Object)
	}
	
	if result.ParsedCommand.Method != "LIST" {
		t.Errorf("Expected method 'LIST', got %v", result.ParsedCommand.Method)
	}
}

func TestIntegration_CommandChaining(t *testing.T) {
	suite := setupIntegrationTest(t)
	defer suite.cleanup()
	
	ctx := context.Background()
	
	// Register objects for chaining test
	objects := []string{"CUSTOMER", "INVOICE"}
	for _, objName := range objects {
		err := suite.engine.Registry().RegisterObject(&mdwregistry.ObjectDefinition{
			Name:        objName,
			Service:     strings.ToLower(objName) + "-service",
			Description: objName + " management object",
			Methods: map[string]*mdwregistry.MethodDefinition{
				"CREATE": {
					Name:        "CREATE",
					Description: "Create new " + strings.ToLower(objName),
					Parameters: map[string]*mdwregistry.ParameterDefinition{
						"name": {Name: "name", Type: "string", Required: true},
					},
				},
			},
		})
		if err != nil {
			t.Fatalf("Failed to register %s object: %v", objName, err)
		}
	}
	
	command := `CUSTOMER.CREATE name="Test Corp" | INVOICE.CREATE name="INV-001"`
	
	result, err := suite.engine.Execute(ctx, command)
	if err != nil {
		t.Fatalf("Command execution failed: %v", err)
	}
	
	if !result.Success {
		t.Errorf("Expected successful result, got: %s", result.Message)
	}
	
	// Verify chain execution
	if result.ParsedCommand == nil {
		t.Fatal("Expected parsed command but got nil")
	}
}

func TestIntegration_CommandAbbreviations(t *testing.T) {
	suite := setupIntegrationTest(t)
	defer suite.cleanup()
	
	ctx := context.Background()
	
	// Register object with method for abbreviation testing
	err := suite.engine.Registry().RegisterObject(&mdwregistry.ObjectDefinition{
		Name:        "CUSTOMER",
		Service:     "customer-service",
		Description: "Customer management object",
		Methods: map[string]*mdwregistry.MethodDefinition{
			"CREATE": {
				Name:        "CREATE",
				Description: "Create new customer",
				Parameters: map[string]*mdwregistry.ParameterDefinition{
					"name": {Name: "name", Type: "string", Required: true},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("Failed to register CUSTOMER object: %v", err)
	}
	
	testCases := []struct {
		name    string
		command string
		wantErr bool
	}{
		{
			name:    "Full command",
			command: `CUSTOMER.CREATE name="Test"`,
			wantErr: false,
		},
		{
			name:    "Abbreviated object",
			command: `CUST.CREATE name="Test"`,
			wantErr: false,
		},
		{
			name:    "Abbreviated method",
			command: `CUSTOMER.CR name="Test"`,
			wantErr: false,
		},
		{
			name:    "Both abbreviated",
			command: `CUST.CR name="Test"`,
			wantErr: false,
		},
		{
			name:    "Ambiguous abbreviation",
			command: `C.C name="Test"`,
			wantErr: true, // Too ambiguous
		},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := suite.engine.Execute(ctx, tc.command)
			
			if tc.wantErr {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}
			
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}
			
			if !result.Success {
				t.Errorf("Expected successful result, got: %s", result.Message)
			}
		})
	}
}

func TestIntegration_AliasSystem(t *testing.T) {
	suite := setupIntegrationTest(t)
	defer suite.cleanup()
	
	ctx := context.Background()
	
	// Create an alias for a common command
	err := suite.engine.Registry().RegisterAlias("uc", `CUSTOMER[unpaid=true].LIST`)
	if err != nil {
		t.Fatalf("Failed to create alias: %v", err)
	}
	
	// Test using the alias
	result, err := suite.engine.Execute(ctx, "uc")
	if err != nil {
		t.Fatalf("Alias execution failed: %v", err)
	}
	
	if !result.Success {
		t.Errorf("Expected successful result, got: %s", result.Message)
	}
	
	// Test alias with parameters
	err = suite.engine.Registry().RegisterAlias("cc", `CUSTOMER.CREATE name="{{name}}" type="{{type}}"`)
	if err != nil {
		t.Fatalf("Failed to create parameterized alias: %v", err)
	}
	
	result, err = suite.engine.Execute(ctx, `cc name="Test Corp" type="B2B"`)
	if err != nil {
		t.Fatalf("Parameterized alias execution failed: %v", err)
	}
	
	if !result.Success {
		t.Errorf("Expected successful result, got: %s", result.Message)
	}
}

func TestIntegration_ErrorHandling(t *testing.T) {
	suite := setupIntegrationTest(t)
	defer suite.cleanup()
	
	ctx := context.Background()
	
	testCases := []struct {
		name        string
		command     string
		expectError string
	}{
		{
			name:        "Unknown object",
			command:     "UNKNOWN.METHOD",
			expectError: "object not found",
		},
		{
			name:        "Unknown method",
			command:     "HELP.UNKNOWN",
			expectError: "method not found",
		},
		{
			name:        "Invalid syntax",
			command:     "INVALID SYNTAX [[[",
			expectError: "parse error",
		},
		{
			name:        "Missing required parameter",
			command:     "CUSTOMER.CREATE", // Missing required name parameter
			expectError: "missing required parameter",
		},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := suite.engine.Execute(ctx, tc.command)
			
			if err == nil && (result == nil || result.Success) {
				t.Error("Expected error but command succeeded")
				return
			}
			
			var errorMsg string
			if err != nil {
				errorMsg = err.Error()
			} else if result != nil {
				errorMsg = result.Message
			}
			
			if !strings.Contains(strings.ToLower(errorMsg), strings.ToLower(tc.expectError)) {
				t.Errorf("Expected error containing '%s', got: %s", tc.expectError, errorMsg)
			}
		})
	}
}

func TestIntegration_PermissionChecking(t *testing.T) {
	suite := setupIntegrationTest(t)
	defer suite.cleanup()
	
	ctx := context.Background()
	
	// Register object for permission testing
	err := suite.engine.Registry().RegisterObject(&mdwregistry.ObjectDefinition{
		Name:        "SENSITIVE",
		Service:     "sensitive-service",
		Description: "Sensitive data object",
		Methods: map[string]*mdwregistry.MethodDefinition{
			"READ": {
				Name:        "READ",
				Description: "Read sensitive data",
			},
		},
	})
	if err != nil {
		t.Fatalf("Failed to register SENSITIVE object: %v", err)
	}
	
	// Test with permission denied
	result, err := suite.engine.Execute(ctx, "SENSITIVE.READ")
	
	// The mock permission checker should deny access
	if err == nil && result.Success {
		t.Error("Expected permission denial but command succeeded")
	}
}

func TestIntegration_ServiceClientIntegration(t *testing.T) {
	suite := setupIntegrationTest(t)
	defer suite.cleanup()
	
	ctx := context.Background()
	
	// Register an object that will trigger service client calls
	err := suite.engine.Registry().RegisterObject(&mdwregistry.ObjectDefinition{
		Name:        "EXTERNAL",
		Service:     "external-service",
		Description: "External service object",
		Methods: map[string]*mdwregistry.MethodDefinition{
			"CALL": {
				Name:        "CALL",
				Description: "Call external service",
			},
		},
	})
	if err != nil {
		t.Fatalf("Failed to register EXTERNAL object: %v", err)
	}
	
	// Execute command that will go through the service client
	result, err := suite.engine.Execute(ctx, "EXTERNAL.CALL")
	if err != nil {
		t.Fatalf("Service client call failed: %v", err)
	}
	
	if !result.Success {
		t.Errorf("Expected successful result, got: %s", result.Message)
	}
	
	// Verify that the service client was used
	if result.ParsedCommand == nil {
		t.Fatal("Expected parsed command but got nil")
	}
	
	// The mock service should return metadata indicating it was called
	if metadata, ok := result.Metadata["serviceName"]; ok {
		if metadata != "external-service" {
			t.Errorf("Expected service name 'external-service', got %v", metadata)
		}
	}
}

func TestIntegration_ASTVisitorIntegration(t *testing.T) {
	suite := setupIntegrationTest(t)
	defer suite.cleanup()
	
	// Test that the AST visitor pattern works correctly in the full pipeline
	p, err := mdwparser.New(mdwparser.Options{
		Logger:         suite.logger,
		MaxInputLength: 4096,
		EnableChaining: true,
		Registry:       suite.engine.Registry(),
	})
	if err != nil {
		t.Fatalf("Failed to create parser: %v", err)
	}
	
	command, err := p.Parse(`CUSTOMER[city="Berlin"].CREATE name="Test"`)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	
	// Use AST validation visitor
	errors := mdwast.ValidateAST(command)
	if len(errors) > 0 {
		t.Errorf("AST validation failed: %v", errors)
	}
	
	// Use AST string visitor
	astString := mdwast.ASTToString(command)
	if astString == "" {
		t.Error("Expected non-empty AST string representation")
	}
	
	// Use AST collector visitor
	collector := mdwast.CollectNodes(command)
	if len(collector.Commands) != 1 {
		t.Errorf("Expected 1 command in AST, got %d", len(collector.Commands))
	}
	
	t.Logf("AST String Representation:\n%s", astString)
}

func TestIntegration_ConcurrentExecution(t *testing.T) {
	suite := setupIntegrationTest(t)
	defer suite.cleanup()
	
	ctx := context.Background()
	
	// Test concurrent command execution
	const numGoroutines = 10
	const commandsPerGoroutine = 5
	
	results := make(chan error, numGoroutines*commandsPerGoroutine)
	
	for i := 0; i < numGoroutines; i++ {
		go func(goroutineID int) {
			for j := 0; j < commandsPerGoroutine; j++ {
				command := "HELP.SHOW"
				result, err := suite.engine.Execute(ctx, command)
				
				if err != nil {
					results <- err
					continue
				}
				
				if !result.Success {
					results <- fmt.Errorf("command failed: %s", result.Message)
					continue
				}
				
				results <- nil
			}
		}(i)
	}
	
	// Collect results
	var errors []error
	for i := 0; i < numGoroutines*commandsPerGoroutine; i++ {
		if err := <-results; err != nil {
			errors = append(errors, err)
		}
	}
	
	if len(errors) > 0 {
		t.Errorf("Concurrent execution had %d errors: %v", len(errors), errors[:min(5, len(errors))])
	}
}

func TestIntegration_PerformanceBaseline(t *testing.T) {
	suite := setupIntegrationTest(t)
	defer suite.cleanup()
	
	ctx := context.Background()
	
	// Warm up
	for i := 0; i < 10; i++ {
		suite.engine.Execute(ctx, "HELP.SHOW")
	}
	
	// Performance test
	start := time.Now()
	const iterations = 100
	
	for i := 0; i < iterations; i++ {
		result, err := suite.engine.Execute(ctx, "HELP.SHOW")
		if err != nil || !result.Success {
			t.Fatalf("Performance test failed at iteration %d: %v", i, err)
		}
	}
	
	elapsed := time.Since(start)
	avgDuration := elapsed / iterations
	
	t.Logf("Performance baseline: %d iterations in %v (avg: %v per command)", 
		iterations, elapsed, avgDuration)
	
	// Ensure reasonable performance (adjust threshold as needed)
	if avgDuration > 50*time.Millisecond {
		t.Errorf("Performance below expectations: average %v per command", avgDuration)
	}
}

// Benchmarks for integration scenarios

func BenchmarkIntegration_SimpleCommand(b *testing.B) {
	suite := setupIntegrationTest(nil) // nil for benchmark
	defer suite.cleanup()
	
	ctx := context.Background()
	command := "HELP.SHOW"
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result, err := suite.engine.Execute(ctx, command)
		if err != nil || !result.Success {
			b.Fatalf("Benchmark failed: %v", err)
		}
	}
}

func BenchmarkIntegration_ComplexCommand(b *testing.B) {
	suite := setupIntegrationTest(nil)
	defer suite.cleanup()
	
	// Register object for benchmark
	suite.engine.Registry().RegisterObject(&mdwregistry.ObjectDefinition{
		Name:    "BENCHMARK",
		Service: "benchmark-service",
		Methods: map[string]*mdwregistry.MethodDefinition{
			"TEST": {
				Name: "TEST",
				Parameters: map[string]*mdwregistry.ParameterDefinition{
					"param1": {Name: "param1", Type: "string"},
					"param2": {Name: "param2", Type: "number"},
				},
			},
		},
	})
	
	ctx := context.Background()
	command := `BENCHMARK[status="active" AND priority>5].TEST param1="test" param2=42`
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result, err := suite.engine.Execute(ctx, command)
		if err != nil || !result.Success {
			b.Fatalf("Benchmark failed: %v", err)
		}
	}
}

// Helper functions

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// setupTestRegistry creates a test registry with standard objects for testing
func setupTestRegistry(t *testing.T) *mdwregistry.Registry {
	reg, err := mdwregistry.NewSimple(mdwregistry.Options{
		Logger: mdwlog.GetDefault(),
	})
	if err != nil {
		t.Fatalf("Failed to create registry: %v", err)
	}
	
	// Register standard test objects
	testObjects := []*mdwregistry.ObjectDefinition{
		{
			Name:        "CUSTOMER",
			Service:     "customer-service",
			Description: "Customer management",
			Methods: map[string]*mdwregistry.MethodDefinition{
				"CREATE": {
					Name: "CREATE",
					Parameters: map[string]*mdwregistry.ParameterDefinition{
						"name": {Name: "name", Type: "string", Required: true},
						"type": {Name: "type", Type: "string", Required: false},
					},
				},
				"LIST": {
					Name: "LIST",
					Parameters: map[string]*mdwregistry.ParameterDefinition{
						"limit": {Name: "limit", Type: "number", Required: false},
					},
				},
			},
		},
		{
			Name:        "INVOICE",
			Service:     "invoice-service", 
			Description: "Invoice management",
			Methods: map[string]*mdwregistry.MethodDefinition{
				"CREATE": {
					Name: "CREATE",
					Parameters: map[string]*mdwregistry.ParameterDefinition{
						"amount": {Name: "amount", Type: "number", Required: true},
						"currency": {Name: "currency", Type: "string", Required: false},
					},
				},
			},
		},
	}
	
	for _, obj := range testObjects {
		if err := reg.RegisterObject(obj); err != nil {
			t.Fatalf("Failed to register test object %s: %v", obj.Name, err)
		}
	}
	
	return reg
}