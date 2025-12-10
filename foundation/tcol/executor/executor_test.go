// File: executor_test.go
// Title: TCOL Executor Unit Tests
// Description: Comprehensive unit tests for the TCOL execution engine including
//              command routing, service communication, permission checking,
//              built-in command execution, error handling, and audit logging.
//              Tests cover all command types with mock service clients.
// Author: msto63 with Claude Sonnet 4.0
// Version: v0.1.0
// Created: 2025-01-25
// Modified: 2025-01-25
//
// Change History:
// - 2025-01-25 v0.1.0: Initial comprehensive executor test suite

package executor

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	mdwlog "github.com/msto63/mDW/foundation/core/log"
	mdwast "github.com/msto63/mDW/foundation/tcol/ast"
	mdwregistry "github.com/msto63/mDW/foundation/tcol/registry"
)

// Mock implementations for testing

type MockServiceClient struct {
	responses    map[string]*ServiceResponse
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
	Context     *ExecutionContext
}

func NewMockServiceClient() *MockServiceClient {
	return &MockServiceClient{
		responses:    make(map[string]*ServiceResponse),
		errors:       make(map[string]error),
		callHistory:  make([]MockCall, 0),
		healthStatus: make(map[string]error),
	}
}

func (m *MockServiceClient) Execute(ctx context.Context, serviceName, objectName, methodName string, 
	params map[string]interface{}, execCtx *ExecutionContext) (*ServiceResponse, error) {
	
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
	
	// Default success response
	return &ServiceResponse{
		Success: true,
		Data:    fmt.Sprintf("Mock response for %s", key),
		Metadata: map[string]interface{}{
			"mock": true,
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

func (m *MockServiceClient) SetResponse(serviceName, objectName, methodName string, response *ServiceResponse) {
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

type MockPermissionChecker struct {
	permissions map[string]bool
	errors      map[string]error
}

func NewMockPermissionChecker() *MockPermissionChecker {
	return &MockPermissionChecker{
		permissions: make(map[string]bool),
		errors:      make(map[string]error),
	}
}

func (p *MockPermissionChecker) CheckPermission(ctx context.Context, userID, objectName, methodName string, execCtx *ExecutionContext) error {
	key := fmt.Sprintf("%s.%s.%s", userID, objectName, methodName)
	
	if err, exists := p.errors[key]; exists {
		return err
	}
	
	if allowed, exists := p.permissions[key]; exists && !allowed {
		return fmt.Errorf("permission denied for user %s to execute %s.%s", userID, objectName, methodName)
	}
	
	return nil
}

func (p *MockPermissionChecker) GetUserPermissions(ctx context.Context, userID string) ([]string, error) {
	var perms []string
	for key, allowed := range p.permissions {
		if allowed && strings.HasPrefix(key, userID+".") {
			perms = append(perms, strings.TrimPrefix(key, userID+"."))
		}
	}
	return perms, nil
}

func (p *MockPermissionChecker) SetPermission(userID, objectName, methodName string, allowed bool) {
	key := fmt.Sprintf("%s.%s.%s", userID, objectName, methodName)
	p.permissions[key] = allowed
}

func (p *MockPermissionChecker) SetError(userID, objectName, methodName string, err error) {
	key := fmt.Sprintf("%s.%s.%s", userID, objectName, methodName)
	p.errors[key] = err
}

// Test helper functions

func createTestRegistry() *mdwregistry.Registry {
	reg, _ := mdwregistry.NewSimple(mdwregistry.Options{
		Logger:        mdwlog.GetDefault(),
		EnableAliases: true,
	})
	
	// Register test object
	testObj := &mdwregistry.ObjectDefinition{
		Name:        "CUSTOMER",
		Description: "Customer management",
		Service:     "customer-service",
		Methods: map[string]*mdwregistry.MethodDefinition{
			"CREATE": {Name: "CREATE"},
			"LIST":   {Name: "LIST"},
			"UPDATE": {Name: "UPDATE"},
			"DELETE": {Name: "DELETE"},
		},
		Fields: map[string]*mdwregistry.FieldDefinition{
			"name":  {Name: "name", Type: "string"},
			"email": {Name: "email", Type: "string"},
		},
	}
	reg.RegisterObject(testObj)
	
	return reg
}

func createTestCommand(object, method string) *mdwast.Command {
	return &mdwast.Command{
		Object:     object,
		Method:     method,
		Parameters: make(map[string]mdwast.Value),
	}
}

func createTestContext() *ExecutionContext {
	return &ExecutionContext{
		RequestID: "test-request",
		UserID:    "test-user",
		SessionID: "test-session",
		Timestamp: time.Now(),
		Metadata:  make(map[string]interface{}),
	}
}

// Test cases

func TestNew(t *testing.T) {
	mockClient := NewMockServiceClient()
	mockPermissions := NewMockPermissionChecker()

	tests := []struct {
		name      string
		options   Options
		expectErr bool
		errMsg    string
		checkFunc func(*Engine) bool
	}{
		{
			name: "Valid options",
			options: Options{
				Logger:            mdwlog.GetDefault(),
				ServiceClient:     mockClient,
				PermissionChecker: mockPermissions,
				ServiceTimeout:    10 * time.Second,
				MaxChainDepth:     5,
				EnableAuditLog:    true,
			},
			expectErr: false,
			checkFunc: func(e *Engine) bool {
				return e.client == mockClient && 
					   e.permissions == mockPermissions &&
					   e.options.ServiceTimeout == 10*time.Second &&
					   e.options.MaxChainDepth == 5 &&
					   e.options.EnableAuditLog
			},
		},
		{
			name: "Default options",
			options: Options{
				ServiceClient: mockClient,
			},
			expectErr: false,
			checkFunc: func(e *Engine) bool {
				return e.options.ServiceTimeout == 30*time.Second &&
					   e.options.MaxChainDepth == 10 &&
					   !e.options.EnableAuditLog
			},
		},
		{
			name: "Missing service client",
			options: Options{
				Logger: mdwlog.GetDefault(),
			},
			expectErr: true,
			errMsg:    "ServiceClient is required",
		},
		{
			name: "Nil logger (should use default)",
			options: Options{
				ServiceClient: mockClient,
				Logger:        nil,
			},
			expectErr: false,
			checkFunc: func(e *Engine) bool {
				return e.logger != nil
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			engine, err := New(tt.options)

			if tt.expectErr {
				if err == nil {
					t.Errorf("Expected error but got none")
				} else if tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("Expected error containing %q, got %q", tt.errMsg, err.Error())
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if engine == nil {
				t.Fatal("Expected engine but got nil")
			}

			if tt.checkFunc != nil && !tt.checkFunc(engine) {
				t.Error("Engine check function failed")
			}
		})
	}
}

func TestEngine_SetRegistry(t *testing.T) {
	mockClient := NewMockServiceClient()
	engine, err := New(Options{
		ServiceClient: mockClient,
		Logger:        mdwlog.GetDefault(),
	})
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}

	registry := createTestRegistry()
	engine.SetRegistry(registry)

	if engine.registry != registry {
		t.Error("Registry was not set correctly")
	}
}

func TestEngine_Execute_MethodCall(t *testing.T) {
	mockClient := NewMockServiceClient()
	mockPermissions := NewMockPermissionChecker()
	
	engine, err := New(Options{
		ServiceClient:     mockClient,
		PermissionChecker: mockPermissions,
		Logger:            mdwlog.GetDefault(),
		EnableAuditLog:    true,
	})
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}

	registry := createTestRegistry()
	engine.SetRegistry(registry)

	tests := []struct {
		name           string
		command        *mdwast.Command
		context        *ExecutionContext
		setupMocks     func()
		expectErr      bool
		errMsg         string
		checkResult    func(*ExecutionResult) bool
		checkMockCalls func([]MockCall) bool
	}{
		{
			name:    "Simple method call",
			command: createTestCommand("CUSTOMER", "LIST"),
			context: createTestContext(),
			setupMocks: func() {
				mockPermissions.SetPermission("test-user", "CUSTOMER", "LIST", true)
				mockClient.SetResponse("customer-service", "CUSTOMER", "LIST", &ServiceResponse{
					Success: true,
					Data:    []string{"customer1", "customer2"},
					Metadata: map[string]interface{}{"count": 2},
				})
			},
			expectErr: false,
			checkResult: func(result *ExecutionResult) bool {
				return result.Success && 
					   result.ServiceName == "customer-service" &&
					   result.CommandType == "METHOD_CALL"
			},
			checkMockCalls: func(calls []MockCall) bool {
				return len(calls) == 1 && 
					   calls[0].ServiceName == "customer-service" &&
					   calls[0].ObjectName == "CUSTOMER" &&
					   calls[0].MethodName == "LIST"
			},
		},
		{
			name: "Method call with parameters",
			command: &mdwast.Command{
				Object: "CUSTOMER",
				Method: "CREATE",
				Parameters: map[string]mdwast.Value{
					"name": {Type: mdwast.ValueTypeString, Value: "John Doe"},
					"email": {Type: mdwast.ValueTypeString, Value: "john@example.com"},
				},
			},
			context: createTestContext(),
			setupMocks: func() {
				mockPermissions.SetPermission("test-user", "CUSTOMER", "CREATE", true)
				mockClient.SetResponse("customer-service", "CUSTOMER", "CREATE", &ServiceResponse{
					Success: true,
					Data:    map[string]interface{}{"id": "12345", "name": "John Doe"},
				})
			},
			expectErr: false,
			checkResult: func(result *ExecutionResult) bool {
				return result.Success
			},
			checkMockCalls: func(calls []MockCall) bool {
				if len(calls) != 1 {
					return false
				}
				call := calls[0]
				return call.Params["name"] == "John Doe" && 
					   call.Params["email"] == "john@example.com"
			},
		},
		{
			name: "Method call with filter",
			command: &mdwast.Command{
				Object: "CUSTOMER",
				Method: "LIST",
				Filter: &mdwast.FilterExpr{
					Condition: &mdwast.BinaryExpr{
						Op: "=",
						Left: &mdwast.IdentifierExpr{Name: "active"},
						Right: &mdwast.LiteralExpr{Value: mdwast.Value{Type: mdwast.ValueTypeBoolean, Value: true}},
					},
				},
			},
			context: createTestContext(),
			setupMocks: func() {
				mockPermissions.SetPermission("test-user", "CUSTOMER", "LIST", true)
				mockClient.SetResponse("customer-service", "CUSTOMER", "LIST", &ServiceResponse{
					Success: true,
					Data:    []string{"active-customer1"},
				})
			},
			expectErr: false,
			checkResult: func(result *ExecutionResult) bool {
				return result.Success
			},
			checkMockCalls: func(calls []MockCall) bool {
				if len(calls) != 1 {
					return false
				}
				filter, exists := calls[0].Params["_filter"]
				return exists && filter != nil
			},
		},
		{
			name:    "Permission denied",
			command: createTestCommand("CUSTOMER", "DELETE"),
			context: createTestContext(),
			setupMocks: func() {
				mockPermissions.SetPermission("test-user", "CUSTOMER", "DELETE", false)
			},
			expectErr: true,
			errMsg:    "permission denied",
		},
		{
			name:    "Unknown object",
			command: createTestCommand("UNKNOWN", "LIST"),
			context: createTestContext(),
			setupMocks: func() {
				mockPermissions.SetPermission("test-user", "UNKNOWN", "LIST", true)
			},
			expectErr: true,
			errMsg:    "unknown object",
		},
		{
			name:    "Service call error",
			command: createTestCommand("CUSTOMER", "LIST"),
			context: createTestContext(),
			setupMocks: func() {
				mockPermissions.SetPermission("test-user", "CUSTOMER", "LIST", true)
				mockClient.SetError("customer-service", "CUSTOMER", "LIST", errors.New("service unavailable"))
			},
			expectErr: true,
			errMsg:    "service call failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear mock history
			mockClient.ClearHistory()
			
			// Setup mocks
			if tt.setupMocks != nil {
				tt.setupMocks()
			}

			// Execute command
			result, err := engine.Execute(context.Background(), tt.command, tt.context)

			if tt.expectErr {
				if err == nil {
					t.Errorf("Expected error but got none")
				} else if tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("Expected error containing %q, got %q", tt.errMsg, err.Error())
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

			if tt.checkResult != nil && !tt.checkResult(result) {
				t.Error("Result check function failed")
			}

			if tt.checkMockCalls != nil && !tt.checkMockCalls(mockClient.GetCallHistory()) {
				t.Error("Mock calls check function failed")
			}
		})
	}
}

func TestEngine_Execute_ObjectAccess(t *testing.T) {
	mockClient := NewMockServiceClient()
	mockPermissions := NewMockPermissionChecker()
	
	engine, err := New(Options{
		ServiceClient:     mockClient,
		PermissionChecker: mockPermissions,
		Logger:            mdwlog.GetDefault(),
	})
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}

	registry := createTestRegistry()
	engine.SetRegistry(registry)

	tests := []struct {
		name        string
		objectID    string
		setupMocks  func()
		expectErr   bool
		checkResult func(*ExecutionResult) bool
	}{
		{
			name:     "Valid object access",
			objectID: "12345",
			setupMocks: func() {
				mockPermissions.SetPermission("test-user", "CUSTOMER", "READ", true)
				mockClient.SetResponse("customer-service", "CUSTOMER", "GET", &ServiceResponse{
					Success: true,
					Data:    map[string]interface{}{"id": "12345", "name": "John Doe"},
				})
			},
			expectErr: false,
			checkResult: func(result *ExecutionResult) bool {
				return result.Success && result.CommandType == "OBJECT_ACCESS"
			},
		},
		{
			name:     "Permission denied for read",
			objectID: "12345",
			setupMocks: func() {
				mockPermissions.SetPermission("test-user", "CUSTOMER", "READ", false)
			},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient.ClearHistory()
			
			if tt.setupMocks != nil {
				tt.setupMocks()
			}

			command := &mdwast.Command{
				Object:   "CUSTOMER",
				ObjectID: tt.objectID,
			}

			result, err := engine.Execute(context.Background(), command, createTestContext())

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

			if tt.checkResult != nil && !tt.checkResult(result) {
				t.Error("Result check function failed")
			}
		})
	}
}

func TestEngine_Execute_FieldOperation(t *testing.T) {
	mockClient := NewMockServiceClient()
	mockPermissions := NewMockPermissionChecker()
	
	engine, err := New(Options{
		ServiceClient:     mockClient,
		PermissionChecker: mockPermissions,
		Logger:            mdwlog.GetDefault(),
	})
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}

	registry := createTestRegistry()
	engine.SetRegistry(registry)

	tests := []struct {
		name        string
		fieldOp     *mdwast.FieldOperation
		expectedMethod string
		setupMocks  func()
		expectErr   bool
		checkResult func(*ExecutionResult) bool
	}{
		{
			name: "Field read operation",
			fieldOp: &mdwast.FieldOperation{
				Field: "name",
				Op:    "",
			},
			expectedMethod: "GET_FIELD",
			setupMocks: func() {
				mockPermissions.SetPermission("test-user", "CUSTOMER", "READ", true)
				mockClient.SetResponse("customer-service", "CUSTOMER", "GET_FIELD", &ServiceResponse{
					Success: true,
					Data:    "John Doe",
				})
			},
			expectErr: false,
			checkResult: func(result *ExecutionResult) bool {
				return result.Success && result.CommandType == "FIELD_OPERATION"
			},
		},
		{
			name: "Field write operation",
			fieldOp: &mdwast.FieldOperation{
				Field: "email",
				Op:    "=",
				Value: mdwast.Value{Type: mdwast.ValueTypeString, Value: "new@example.com"},
			},
			expectedMethod: "SET_FIELD",
			setupMocks: func() {
				mockPermissions.SetPermission("test-user", "CUSTOMER", "UPDATE", true)
				mockClient.SetResponse("customer-service", "CUSTOMER", "SET_FIELD", &ServiceResponse{
					Success: true,
					Data:    "Field updated successfully",
				})
			},
			expectErr: false,
			checkResult: func(result *ExecutionResult) bool {
				return result.Success
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient.ClearHistory()
			
			if tt.setupMocks != nil {
				tt.setupMocks()
			}

			command := &mdwast.Command{
				Object:   "CUSTOMER",
				ObjectID: "12345",
				FieldOp:  tt.fieldOp,
			}

			result, err := engine.Execute(context.Background(), command, createTestContext())

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

			if tt.checkResult != nil && !tt.checkResult(result) {
				t.Error("Result check function failed")
			}

			// Check that correct method was called
			calls := mockClient.GetCallHistory()
			if len(calls) == 1 && calls[0].MethodName != tt.expectedMethod {
				t.Errorf("Expected method %s, got %s", tt.expectedMethod, calls[0].MethodName)
			}
		})
	}
}

func TestEngine_Execute_BuiltinCommands(t *testing.T) {
	mockClient := NewMockServiceClient()
	
	engine, err := New(Options{
		ServiceClient: mockClient,
		Logger:        mdwlog.GetDefault(),
	})
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}

	registry := createTestRegistry()
	engine.SetRegistry(registry)

	tests := []struct {
		name        string
		command     *mdwast.Command
		expectErr   bool
		checkResult func(*ExecutionResult) bool
	}{
		{
			name: "ALIAS.LIST command",
			command: &mdwast.Command{
				Object: "ALIAS",
				Method: "LIST",
			},
			expectErr: false,
			checkResult: func(result *ExecutionResult) bool {
				return result.Success && result.CommandType == "BUILTIN"
			},
		},
		{
			name: "ALIAS.CREATE command",
			command: &mdwast.Command{
				Object: "ALIAS",
				Method: "CREATE",
				Parameters: map[string]mdwast.Value{
					"name":    {Type: mdwast.ValueTypeString, Value: "uc"},
					"command": {Type: mdwast.ValueTypeString, Value: "CUSTOMER.LIST status=unpaid"},
				},
			},
			expectErr: false,
			checkResult: func(result *ExecutionResult) bool {
				return result.Success && result.CommandType == "BUILTIN"
			},
		},
		{
			name: "HELP.LIST command",
			command: &mdwast.Command{
				Object: "HELP",
				Method: "LIST",
			},
			expectErr: false,
			checkResult: func(result *ExecutionResult) bool {
				return result.Success && result.CommandType == "BUILTIN"
			},
		},
		{
			name: "HELP.OBJECT command",
			command: &mdwast.Command{
				Object: "HELP",
				Method: "OBJECT",
				Parameters: map[string]mdwast.Value{
					"name": {Type: mdwast.ValueTypeString, Value: "CUSTOMER"},
				},
			},
			expectErr: false,
			checkResult: func(result *ExecutionResult) bool {
				return result.Success && result.CommandType == "BUILTIN"
			},
		},
		{
			name: "ALIAS.CREATE missing parameters",
			command: &mdwast.Command{
				Object: "ALIAS",
				Method: "CREATE",
				Parameters: map[string]mdwast.Value{
					"name": {Type: mdwast.ValueTypeString, Value: "uc"},
					// Missing command parameter
				},
			},
			expectErr: true,
		},
		{
			name: "Unknown builtin method",
			command: &mdwast.Command{
				Object: "ALIAS",
				Method: "UNKNOWN",
			},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient.ClearHistory()

			result, err := engine.Execute(context.Background(), tt.command, createTestContext())

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

			if tt.checkResult != nil && !tt.checkResult(result) {
				t.Error("Result check function failed")
			}

			// Builtin commands should not call external services
			calls := mockClient.GetCallHistory()
			if len(calls) > 0 {
				t.Errorf("Builtin commands should not call external services, but got %d calls", len(calls))
			}
		})
	}
}

func TestEngine_Execute_CommandChaining(t *testing.T) {
	mockClient := NewMockServiceClient()
	mockPermissions := NewMockPermissionChecker()
	
	engine, err := New(Options{
		ServiceClient:     mockClient,
		PermissionChecker: mockPermissions,
		Logger:            mdwlog.GetDefault(),
		MaxChainDepth:     3,
	})
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}

	reg := createTestRegistry()
	engine.SetRegistry(reg)

	// Create chained command: CUSTOMER.LIST | EXPORT.CSV
	exportObj := &mdwregistry.ObjectDefinition{
		Name:    "EXPORT",
		Service: "export-service",
		Methods: map[string]*mdwregistry.MethodDefinition{
			"CSV": {Name: "CSV"},
		},
	}
	reg.RegisterObject(exportObj)

	chainedCommand := &mdwast.Command{
		Object: "CUSTOMER",
		Method: "LIST",
		Chain: &mdwast.Command{
			Object: "EXPORT",
			Method: "CSV",
		},
	}

	// Setup mocks
	mockPermissions.SetPermission("test-user", "CUSTOMER", "LIST", true)
	mockPermissions.SetPermission("test-user", "EXPORT", "CSV", true)
	
	mockClient.SetResponse("customer-service", "CUSTOMER", "LIST", &ServiceResponse{
		Success: true,
		Data:    []string{"customer1", "customer2"},
	})
	
	mockClient.SetResponse("export-service", "EXPORT", "CSV", &ServiceResponse{
		Success: true,
		Data:    "CSV export completed",
	})

	result, err := engine.Execute(context.Background(), chainedCommand, createTestContext())

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
		return
	}

	if !result.Success {
		t.Error("Expected successful result")
	}

	// Check that both commands were executed
	calls := mockClient.GetCallHistory()
	if len(calls) != 2 {
		t.Errorf("Expected 2 service calls, got %d", len(calls))
		return
	}

	// Check chain result in metadata
	if result.Metadata == nil || result.Metadata["chainResult"] == nil {
		t.Error("Expected chain result in metadata")
	}
}

func TestEngine_Execute_ChainDepthLimit(t *testing.T) {
	mockClient := NewMockServiceClient()
	
	engine, err := New(Options{
		ServiceClient: mockClient,
		Logger:        mdwlog.GetDefault(),
		MaxChainDepth: 2,
	})
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}

	registry := createTestRegistry()
	engine.SetRegistry(registry)

	command := createTestCommand("CUSTOMER", "LIST")
	execCtx := createTestContext()
	execCtx.ChainDepth = 2 // At the limit

	result, err := engine.Execute(context.Background(), command, execCtx)

	if err == nil {
		t.Error("Expected error for chain depth limit exceeded")
	} else if !strings.Contains(err.Error(), "chain exceeds maximum depth") {
		t.Errorf("Expected chain depth error, got: %v", err)
	}

	if result != nil {
		t.Error("Expected nil result when chain depth exceeded")
	}
}

func TestEngine_ExecuteBatch(t *testing.T) {
	mockClient := NewMockServiceClient()
	mockPermissions := NewMockPermissionChecker()
	
	engine, err := New(Options{
		ServiceClient:     mockClient,
		PermissionChecker: mockPermissions,
		Logger:            mdwlog.GetDefault(),
	})
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}

	registry := createTestRegistry()
	engine.SetRegistry(registry)

	// Setup permissions and responses
	mockPermissions.SetPermission("test-user", "CUSTOMER", "LIST", true)
	mockPermissions.SetPermission("test-user", "CUSTOMER", "CREATE", true)
	
	mockClient.SetResponse("customer-service", "CUSTOMER", "LIST", &ServiceResponse{
		Success: true,
		Data:    []string{"customer1"},
	})
	
	mockClient.SetResponse("customer-service", "CUSTOMER", "CREATE", &ServiceResponse{
		Success: true,
		Data:    map[string]interface{}{"id": "12345"},
	})

	commands := []*mdwast.Command{
		createTestCommand("CUSTOMER", "LIST"),
		createTestCommand("CUSTOMER", "CREATE"),
	}

	results, err := engine.ExecuteBatch(context.Background(), commands, createTestContext())

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
		return
	}

	if len(results) != 2 {
		t.Errorf("Expected 2 results, got %d", len(results))
		return
	}

	for i, result := range results {
		if result == nil {
			t.Errorf("Result %d is nil", i)
			continue
		}
		if !result.Success {
			t.Errorf("Result %d is not successful", i)
		}
	}

	// Check that both commands were executed
	calls := mockClient.GetCallHistory()
	if len(calls) != 2 {
		t.Errorf("Expected 2 service calls, got %d", len(calls))
	}
}

func TestEngine_ExecuteBatch_EmptyList(t *testing.T) {
	mockClient := NewMockServiceClient()
	
	engine, err := New(Options{
		ServiceClient: mockClient,
		Logger:        mdwlog.GetDefault(),
	})
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}

	results, err := engine.ExecuteBatch(context.Background(), []*mdwast.Command{}, createTestContext())

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if len(results) != 0 {
		t.Errorf("Expected empty results, got %d", len(results))
	}
}

func TestEngine_Execute_NilCommand(t *testing.T) {
	mockClient := NewMockServiceClient()
	
	engine, err := New(Options{
		ServiceClient: mockClient,
		Logger:        mdwlog.GetDefault(),
	})
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}

	result, err := engine.Execute(context.Background(), nil, createTestContext())

	if err == nil {
		t.Error("Expected error for nil command")
	} else if !strings.Contains(err.Error(), "command cannot be nil") {
		t.Errorf("Expected 'command cannot be nil' error, got: %v", err)
	}

	if result != nil {
		t.Error("Expected nil result for nil command")
	}
}

func TestEngine_Execute_NilContext(t *testing.T) {
	mockClient := NewMockServiceClient()
	
	engine, err := New(Options{
		ServiceClient: mockClient,
		Logger:        mdwlog.GetDefault(),
	})
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}

	registry := createTestRegistry()
	engine.SetRegistry(registry)

	command := createTestCommand("CUSTOMER", "LIST")

	// Should create default context if nil is passed
	_, err = engine.Execute(context.Background(), command, nil)

	// This might fail due to service call, but should not fail due to nil context
	if err != nil && strings.Contains(err.Error(), "context") {
		t.Errorf("Should handle nil context gracefully, got: %v", err)
	}
}

func TestEngine_Close(t *testing.T) {
	mockClient := NewMockServiceClient()
	
	engine, err := New(Options{
		ServiceClient: mockClient,
		Logger:        mdwlog.GetDefault(),
	})
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}

	err = engine.Close()
	if err != nil {
		t.Errorf("Unexpected error closing engine: %v", err)
	}

	if !mockClient.closed {
		t.Error("Expected service client to be closed")
	}
}

func TestEngine_SerializeFilter(t *testing.T) {
	mockClient := NewMockServiceClient()
	
	engine, err := New(Options{
		ServiceClient: mockClient,
		Logger:        mdwlog.GetDefault(),
	})
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}

	filter := &mdwast.FilterExpr{
		Condition: &mdwast.BinaryExpr{
			Op: "=",
			Left: &mdwast.IdentifierExpr{Name: "status"},
			Right: &mdwast.LiteralExpr{Value: mdwast.Value{Type: mdwast.ValueTypeString, Value: "active"}},
		},
	}

	serialized := engine.serializeFilter(filter)

	if serialized == nil {
		t.Fatal("Expected serialized filter, got nil")
	}

	condition, exists := serialized["condition"]
	if !exists {
		t.Error("Expected condition in serialized filter")
	}

	conditionMap, ok := condition.(map[string]interface{})
	if !ok {
		t.Error("Expected condition to be a map")
	} else {
		if conditionMap["type"] != "binary" {
			t.Errorf("Expected binary type, got %v", conditionMap["type"])
		}
		if conditionMap["op"] != "=" {
			t.Errorf("Expected = operator, got %v", conditionMap["op"])
		}
	}
}

// Benchmarks

func BenchmarkEngine_Execute_SimpleCommand(b *testing.B) {
	mockClient := NewMockServiceClient()
	mockPermissions := NewMockPermissionChecker()
	
	engine, _ := New(Options{
		ServiceClient:     mockClient,
		PermissionChecker: mockPermissions,
		Logger:            mdwlog.GetDefault(),
	})

	registry := createTestRegistry()
	engine.SetRegistry(registry)

	// Setup mocks
	mockPermissions.SetPermission("test-user", "CUSTOMER", "LIST", true)
	mockClient.SetResponse("customer-service", "CUSTOMER", "LIST", &ServiceResponse{
		Success: true,
		Data:    []string{"customer1", "customer2"},
	})

	command := createTestCommand("CUSTOMER", "LIST")
	execCtx := createTestContext()

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, err := engine.Execute(context.Background(), command, execCtx)
		if err != nil {
			b.Fatalf("Execution failed: %v", err)
		}
	}
}

func BenchmarkEngine_Execute_ChainedCommand(b *testing.B) {
	mockClient := NewMockServiceClient()
	mockPermissions := NewMockPermissionChecker()
	
	engine, _ := New(Options{
		ServiceClient:     mockClient,
		PermissionChecker: mockPermissions,
		Logger:            mdwlog.GetDefault(),
	})

	reg := createTestRegistry()
	engine.SetRegistry(reg)

	// Register export service
	exportObj := &mdwregistry.ObjectDefinition{
		Name:    "EXPORT",
		Service: "export-service",
		Methods: map[string]*mdwregistry.MethodDefinition{
			"CSV": {Name: "CSV"},
		},
	}
	reg.RegisterObject(exportObj)

	// Setup mocks
	mockPermissions.SetPermission("test-user", "CUSTOMER", "LIST", true)
	mockPermissions.SetPermission("test-user", "EXPORT", "CSV", true)
	
	mockClient.SetResponse("customer-service", "CUSTOMER", "LIST", &ServiceResponse{
		Success: true,
		Data:    []string{"customer1", "customer2"},
	})
	
	mockClient.SetResponse("export-service", "EXPORT", "CSV", &ServiceResponse{
		Success: true,
		Data:    "CSV export completed",
	})

	command := &mdwast.Command{
		Object: "CUSTOMER",
		Method: "LIST",
		Chain: &mdwast.Command{
			Object: "EXPORT",
			Method: "CSV",
		},
	}
	execCtx := createTestContext()

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, err := engine.Execute(context.Background(), command, execCtx)
		if err != nil {
			b.Fatalf("Execution failed: %v", err)
		}
	}
}

func BenchmarkEngine_ExecuteBatch(b *testing.B) {
	mockClient := NewMockServiceClient()
	mockPermissions := NewMockPermissionChecker()
	
	engine, _ := New(Options{
		ServiceClient:     mockClient,
		PermissionChecker: mockPermissions,
		Logger:            mdwlog.GetDefault(),
	})

	registry := createTestRegistry()
	engine.SetRegistry(registry)

	// Setup mocks
	mockPermissions.SetPermission("test-user", "CUSTOMER", "LIST", true)
	mockClient.SetResponse("customer-service", "CUSTOMER", "LIST", &ServiceResponse{
		Success: true,
		Data:    []string{"customer1"},
	})

	commands := []*mdwast.Command{
		createTestCommand("CUSTOMER", "LIST"),
		createTestCommand("CUSTOMER", "LIST"),
		createTestCommand("CUSTOMER", "LIST"),
	}
	execCtx := createTestContext()

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, err := engine.ExecuteBatch(context.Background(), commands, execCtx)
		if err != nil {
			b.Fatalf("Batch execution failed: %v", err)
		}
	}
}