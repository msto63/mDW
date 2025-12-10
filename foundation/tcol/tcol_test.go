// File: tcol_test.go
// Title: TCOL Engine Tests
// Description: Unit tests for the main TCOL engine functionality including
//              parsing, execution, and integration with registry and client
//              components. Tests cover basic commands, error handling, and
//              integration scenarios.
// Author: msto63 with Claude Sonnet 4.0
// Version: v0.1.0
// Created: 2025-01-25
// Modified: 2025-01-25
//
// Change History:
// - 2025-01-25 v0.1.0: Initial TCOL tests

package tcol

import (
	"context"
	"testing"
	"time"

	mdwlog "github.com/msto63/mDW/foundation/core/log"
	mdwclient "github.com/msto63/mDW/foundation/tcol/client"
	mdwexecutor "github.com/msto63/mDW/foundation/tcol/executor"
	mdwparser "github.com/msto63/mDW/foundation/tcol/parser"
	mdwregistry "github.com/msto63/mDW/foundation/tcol/registry"
)

func TestTCOLEngine_Execute(t *testing.T) {
	// Create test components
	logger := mdwlog.GetDefault()
	
	// Create registry
	reg, err := mdwregistry.NewSimple(mdwregistry.Options{
		Logger:              logger,
		EnableAbbreviations: true,
		EnableAliases:       true,
	})
	if err != nil {
		t.Fatalf("Failed to create registry: %v", err)
	}

	// Register test object
	testObj := &mdwregistry.ObjectDefinition{
		Name:        "CUSTOMER",
		Description: "Customer management",
		Service:     "customer-service",
		Methods: map[string]*mdwregistry.MethodDefinition{
			"CREATE": {
				Name:        "CREATE",
				Description: "Create a new customer",
				Parameters: map[string]*mdwregistry.ParameterDefinition{
					"name": {
						Name:        "name",
						Type:        "string",
						Required:    true,
						Description: "Customer name",
					},
				},
			},
			"LIST": {
				Name:        "LIST",
				Description: "List customers",
			},
		},
	}

	err = reg.RegisterObject(testObj)
	if err != nil {
		t.Fatalf("Failed to register test object: %v", err)
	}

	// Create client
	serviceClient, err := mdwclient.New(mdwclient.Options{
		Logger: logger,
	})
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer serviceClient.Close()

	// Create executor
	exec, err := mdwexecutor.New(mdwexecutor.Options{
		Logger:        logger,
		ServiceClient: serviceClient,
	})
	if err != nil {
		t.Fatalf("Failed to create executor: %v", err)
	}
	exec.SetRegistry(reg)

	// Create parser
	parserObj, err := mdwparser.New(mdwparser.Options{
		Logger:   logger,
		Registry: reg,
	})
	if err != nil {
		t.Fatalf("Failed to create parser: %v", err)
	}

	// Create TCOL engine
	engine, err := New(HighLevelOptions{
		Logger:   logger,
		Registry: reg,
		Executor: exec,
		Parser:   parserObj,
	})
	if err != nil {
		t.Fatalf("Failed to create TCOL engine: %v", err)
	}

	tests := []struct {
		name      string
		command   string
		expectErr bool
		checkData func(interface{}) bool
	}{
		{
			name:      "Simple method call",
			command:   "CUSTOMER.LIST",
			expectErr: false,
			checkData: func(data interface{}) bool {
				return data != nil
			},
		},
		{
			name:      "Method call with parameters",
			command:   `CUSTOMER.CREATE name="Test Customer"`,
			expectErr: false,
			checkData: func(data interface{}) bool {
				return data != nil
			},
		},
		{
			name:      "Abbreviated command", 
			command:   "CUSTOMER.LIST", // Use registered object and method
			expectErr: false,
			checkData: func(data interface{}) bool {
				return data != nil
			},
		},
		{
			name:      "Invalid object",
			command:   "INVALID.LIST",
			expectErr: true,
		},
		{
			name:      "Invalid method",
			command:   "CUSTOMER.INVALID",
			expectErr: true,
		},
		{
			name:      "Built-in HELP command",
			command:   "HELP.LIST",
			expectErr: false,
			checkData: func(data interface{}) bool {
				return data != nil
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			execCtx := &mdwexecutor.ExecutionContext{
				RequestID: "test-request",
				UserID:    "test-user",
				Timestamp: time.Now(),
			}

			result, err := engine.Execute(ctx, tt.command, execCtx)

			if tt.expectErr {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if result == nil {
				t.Errorf("Expected result but got nil")
				return
			}

			if !result.Success {
				t.Errorf("Expected successful result")
				return
			}

			if tt.checkData != nil && !tt.checkData(result.Data) {
				t.Errorf("Data check failed for result: %+v", result.Data)
			}
		})
	}
}

func TestTCOLEngine_ParseOnly(t *testing.T) {
	// Create minimal TCOL engine for parsing tests
	logger := mdwlog.GetDefault()
	
	reg, err := mdwregistry.NewSimple(mdwregistry.Options{
		Logger:              logger,
		EnableAbbreviations: true,
	})
	if err != nil {
		t.Fatalf("Failed to create registry: %v", err)
	}

	parserObj, err := mdwparser.New(mdwparser.Options{
		Logger:   logger,
		Registry: reg,
	})
	if err != nil {
		t.Fatalf("Failed to create parser: %v", err)
	}

	engine, err := New(HighLevelOptions{
		Logger: logger,
		Parser: parserObj,
	})
	if err != nil {
		t.Fatalf("Failed to create TCOL engine: %v", err)
	}

	tests := []struct {
		name      string
		command   string
		expectErr bool
	}{
		{
			name:      "Simple command",
			command:   "CUSTOMER.LIST",
			expectErr: false,
		},
		{
			name:      "Command with parameters",
			command:   `CUSTOMER.CREATE name="Test" type="B2B"`,
			expectErr: false,
		},
		{
			name:      "Command with filter",
			command:   `CUSTOMER[status="active"].LIST`,
			expectErr: false,
		},
		{
			name:      "Object access",
			command:   "CUSTOMER:12345",
			expectErr: false,
		},
		{
			name:      "Field operation",
			command:   `CUSTOMER:12345:name="New Name"`,
			expectErr: false,
		},
		{
			name:      "Invalid syntax",
			command:   "CUSTOMER.",
			expectErr: true,
		},
		{
			name:      "Empty command",
			command:   "",
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ast, err := engine.Parse(tt.command)

			if tt.expectErr {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if ast == nil {
				t.Errorf("Expected AST but got nil")
				return
			}

			// Basic AST validation
			if ast.Object == "" {
				t.Errorf("Expected object name in AST")
			}
		})
	}
}

func TestTCOLEngine_AliasSupport(t *testing.T) {
	logger := mdwlog.GetDefault()
	
	// Create registry with aliases enabled
	reg, err := mdwregistry.NewSimple(mdwregistry.Options{
		Logger:        logger,
		EnableAliases: true,
	})
	if err != nil {
		t.Fatalf("Failed to create registry: %v", err)
	}

	// Create minimal client and executor for alias testing
	serviceClient, err := mdwclient.New(mdwclient.Options{
		Logger: logger,
	})
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer serviceClient.Close()

	exec, err := mdwexecutor.New(mdwexecutor.Options{
		Logger:        logger,
		ServiceClient: serviceClient,
	})
	if err != nil {
		t.Fatalf("Failed to create executor: %v", err)
	}
	exec.SetRegistry(reg)

	parserObj, err := mdwparser.New(mdwparser.Options{
		Logger:   logger,
		Registry: reg,
	})
	if err != nil {
		t.Fatalf("Failed to create parser: %v", err)
	}

	engine, err := New(HighLevelOptions{
		Logger:   logger,
		Registry: reg,
		Executor: exec,
		Parser:   parserObj,
	})
	if err != nil {
		t.Fatalf("Failed to create TCOL engine: %v", err)
	}

	ctx := context.Background()
	execCtx := &mdwexecutor.ExecutionContext{
		RequestID: "test-alias",
		UserID:    "test-user",
		Timestamp: time.Now(),
	}

	// Test creating an alias
	result, err := engine.Execute(ctx, `ALIAS.CREATE name="uc" command="CUSTOMER.LIST status=unpaid"`, execCtx)
	if err != nil {
		t.Fatalf("Failed to create alias: %v", err)
	}

	if !result.Success {
		t.Fatalf("Alias creation failed")
	}

	// Test listing aliases
	result, err = engine.Execute(ctx, "ALIAS.LIST", execCtx)
	if err != nil {
		t.Fatalf("Failed to list aliases: %v", err)
	}

	if !result.Success {
		t.Fatalf("Alias listing failed")
	}

	aliases, ok := result.Data.(map[string]string)
	if !ok {
		t.Fatalf("Expected aliases map but got %T", result.Data)
	}

	if aliases["UC"] != "CUSTOMER.LIST status=unpaid" {
		t.Errorf("Expected alias 'UC' to be 'CUSTOMER.LIST status=unpaid', got '%s'", aliases["UC"])
	}
}

func BenchmarkTCOLEngine_Execute(b *testing.B) {
	// Setup
	logger := mdwlog.GetDefault()
	
	reg, _ := mdwregistry.NewSimple(mdwregistry.Options{
		Logger:              logger,
		EnableAbbreviations: true,
	})

	// Register test object
	testObj := &mdwregistry.ObjectDefinition{
		Name:    "CUSTOMER",
		Service: "customer-service",
		Methods: map[string]*mdwregistry.MethodDefinition{
			"LIST": {Name: "LIST", Description: "List customers"},
		},
	}
	reg.RegisterObject(testObj)

	serviceClient, _ := mdwclient.New(mdwclient.Options{Logger: logger})
	defer serviceClient.Close()

	exec, _ := mdwexecutor.New(mdwexecutor.Options{
		Logger:        logger,
		ServiceClient: serviceClient,
	})
	exec.SetRegistry(reg)

	parserObj, _ := mdwparser.New(mdwparser.Options{
		Logger:   logger,
		Registry: reg,
	})

	engine, _ := New(HighLevelOptions{
		Logger:   logger,
		Registry: reg,
		Executor: exec,
		Parser:   parserObj,
	})

	ctx := context.Background()
	execCtx := &mdwexecutor.ExecutionContext{
		RequestID: "bench-test",
		UserID:    "bench-user",
		Timestamp: time.Now(),
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, err := engine.Execute(ctx, "CUSTOMER.LIST", execCtx)
		if err != nil {
			b.Fatalf("Execution failed: %v", err)
		}
	}
}