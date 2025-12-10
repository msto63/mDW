// File: executor.go
// Title: TCOL Command Execution Engine
// Description: Implements the execution engine for TCOL commands. Handles
//              command routing, service communication, response processing,
//              and execution context management with comprehensive error
//              handling and audit logging.
// Author: msto63 with Claude Sonnet 4.0
// Version: v0.1.0
// Created: 2025-01-25
// Modified: 2025-01-25
//
// Change History:
// - 2025-01-25 v0.1.0: Initial executor implementation

package executor

import (
	"context"
	"fmt"
	"sync"
	"time"

	mdwlog "github.com/msto63/mDW/foundation/core/log"
	mdwast "github.com/msto63/mDW/foundation/tcol/ast"
	mdwregistry "github.com/msto63/mDW/foundation/tcol/registry"
)

// Engine executes TCOL commands by routing them to appropriate services
type Engine struct {
	registry    *mdwregistry.Registry
	client      ServiceClient
	permissions PermissionChecker
	logger      *mdwlog.Logger
	options     Options
	mutex       sync.RWMutex
}

// Options configures executor behavior
type Options struct {
	Logger           *mdwlog.Logger
	ServiceTimeout   time.Duration
	MaxChainDepth    int
	EnableAuditLog   bool
	PermissionChecker PermissionChecker
	ServiceClient    ServiceClient
}

// ExecutionContext provides context for command execution
type ExecutionContext struct {
	RequestID      string
	UserID         string
	SessionID      string
	CorrelationID  string
	ClientIP       string
	Timestamp      time.Time
	Permissions    []string
	Metadata       map[string]interface{}
	ChainDepth     int
	ParentCommand  *mdwast.Command
}

// ExecutionResult represents the result of command execution
type ExecutionResult struct {
	Success       bool                   `json:"success"`
	Data          interface{}            `json:"data,omitempty"`
	Error         error                  `json:"error,omitempty"`
	ExecutionTime time.Duration          `json:"execution_time"`
	ServiceName   string                 `json:"service_name,omitempty"`
	CommandType   string                 `json:"command_type"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
}

// ServiceClient interface for communicating with microservices
type ServiceClient interface {
	Execute(ctx context.Context, serviceName, objectName, methodName string, 
			params map[string]interface{}, execCtx *ExecutionContext) (*ServiceResponse, error)
	Health(ctx context.Context, serviceName string) error
	Close() error
}

// ServiceResponse represents a response from a microservice
type ServiceResponse struct {
	Success   bool                   `json:"success"`
	Data      interface{}            `json:"data"`
	Error     string                 `json:"error,omitempty"`
	ErrorCode string                 `json:"error_code,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// PermissionChecker interface for checking command permissions
type PermissionChecker interface {
	CheckPermission(ctx context.Context, userID, objectName, methodName string, execCtx *ExecutionContext) error
	GetUserPermissions(ctx context.Context, userID string) ([]string, error)
}

// New creates a new TCOL execution engine
func New(opts Options) (*Engine, error) {
	// Set defaults
	if opts.Logger == nil {
		opts.Logger = mdwlog.GetDefault()
	}
	if opts.ServiceTimeout == 0 {
		opts.ServiceTimeout = 30 * time.Second
	}
	if opts.MaxChainDepth == 0 {
		opts.MaxChainDepth = 10
	}

	// Validate required dependencies
	if opts.ServiceClient == nil {
		return nil, fmt.Errorf("ServiceClient is required")
	}

	engine := &Engine{
		client:      opts.ServiceClient,
		permissions: opts.PermissionChecker,
		logger:      opts.Logger.WithField("component", "tcol-executor"),
		options:     opts,
	}

	engine.logger.Info("TCOL executor initialized", mdwlog.Fields{
		"serviceTimeout": opts.ServiceTimeout,
		"maxChainDepth":  opts.MaxChainDepth,
		"auditEnabled":   opts.EnableAuditLog,
	})

	return engine, nil
}

// SetRegistry sets the command registry for the executor
func (e *Engine) SetRegistry(registry *mdwregistry.Registry) {
	e.mutex.Lock()
	defer e.mutex.Unlock()
	e.registry = registry
}

// Execute executes a TCOL command with the given context
func (e *Engine) Execute(ctx context.Context, cmd *mdwast.Command, execCtx *ExecutionContext) (*ExecutionResult, error) {
	if cmd == nil {
		return nil, fmt.Errorf("command cannot be nil")
	}

	if execCtx == nil {
		execCtx = &ExecutionContext{
			RequestID:     fmt.Sprintf("req-%d", time.Now().UnixNano()),
			Timestamp:     time.Now(),
			Metadata:      make(map[string]interface{}),
		}
	}

	// Check chain depth
	if execCtx.ChainDepth >= e.options.MaxChainDepth {
		return nil, fmt.Errorf("command chain exceeds maximum depth: current=%d, max=%d", 
			execCtx.ChainDepth, e.options.MaxChainDepth)
	}

	startTime := time.Now()

	e.logger.Debug("Executing TCOL command", mdwlog.Fields{
		"requestID":   execCtx.RequestID,
		"userID":      execCtx.UserID,
		"object":      cmd.Object,
		"method":      cmd.Method,
		"chainDepth":  execCtx.ChainDepth,
	})

	// Audit log
	if e.options.EnableAuditLog {
		e.auditCommand(cmd, execCtx, "STARTED")
	}

	// Execute main command
	result, err := e.executeCommand(ctx, cmd, execCtx)
	if err != nil {
		if e.options.EnableAuditLog {
			e.auditCommand(cmd, execCtx, "FAILED")
		}
		return nil, err
	}

	result.ExecutionTime = time.Since(startTime)

	// Execute chain if present
	if cmd.Chain != nil {
		chainCtx := *execCtx // Copy context
		chainCtx.ChainDepth++
		chainCtx.ParentCommand = cmd

		chainResult, err := e.Execute(ctx, cmd.Chain, &chainCtx)
		if err != nil {
			e.logger.Warn("Chain command execution failed", mdwlog.Fields{
				"requestID": execCtx.RequestID,
				"error":     err.Error(),
			})
			// Don't fail the main command if chain fails
		} else {
			// Merge chain result with main result
			if result.Metadata == nil {
				result.Metadata = make(map[string]interface{})
			}
			result.Metadata["chainResult"] = chainResult
		}
	}

	if e.options.EnableAuditLog {
		e.auditCommand(cmd, execCtx, "COMPLETED")
	}

	e.logger.Debug("TCOL command execution completed", mdwlog.Fields{
		"requestID":     execCtx.RequestID,
		"success":       result.Success,
		"executionTime": result.ExecutionTime,
	})

	return result, nil
}

// ExecuteBatch executes multiple commands in sequence
func (e *Engine) ExecuteBatch(ctx context.Context, commands []*mdwast.Command, execCtx *ExecutionContext) ([]*ExecutionResult, error) {
	if len(commands) == 0 {
		return []*ExecutionResult{}, nil
	}

	results := make([]*ExecutionResult, len(commands))
	
	for i, cmd := range commands {
		// Create separate context for each command
		cmdCtx := *execCtx
		cmdCtx.RequestID = fmt.Sprintf("%s-%d", execCtx.RequestID, i)
		
		result, err := e.Execute(ctx, cmd, &cmdCtx)
		if err != nil {
			return results[:i], err
		}
		results[i] = result
	}

	return results, nil
}

// executeCommand executes a single command
func (e *Engine) executeCommand(ctx context.Context, cmd *mdwast.Command, execCtx *ExecutionContext) (*ExecutionResult, error) {
	// Handle different command types
	switch {
	case cmd.ObjectID != "" && cmd.FieldOp == nil:
		// Object access: OBJECT:ID
		return e.executeObjectAccess(ctx, cmd, execCtx)
	
	case cmd.ObjectID != "" && cmd.FieldOp != nil:
		// Field operation: OBJECT:ID:field=value
		return e.executeFieldOperation(ctx, cmd, execCtx)
	
	case cmd.Method != "":
		// Method call: OBJECT.METHOD or OBJECT[filter].METHOD
		return e.executeMethodCall(ctx, cmd, execCtx)
	
	default:
		return nil, fmt.Errorf("invalid command structure: object=%s, method=%s, objectID=%s", 
			cmd.Object, cmd.Method, cmd.ObjectID)
	}
}

// executeObjectAccess executes object access commands (OBJECT:ID)
func (e *Engine) executeObjectAccess(ctx context.Context, cmd *mdwast.Command, execCtx *ExecutionContext) (*ExecutionResult, error) {
	// Check permissions
	if err := e.checkPermission(ctx, cmd.Object, "READ", execCtx); err != nil {
		return nil, err
	}

	// Get service for object
	serviceName, err := e.getServiceForObject(cmd.Object)
	if err != nil {
		return nil, err
	}

	// Prepare parameters
	params := map[string]interface{}{
		"id": cmd.ObjectID,
	}

	// Execute service call
	response, err := e.client.Execute(ctx, serviceName, cmd.Object, "GET", params, execCtx)
	if err != nil {
		return nil, e.wrapServiceError(err, serviceName, cmd.Object, "GET")
	}

	return &ExecutionResult{
		Success:     response.Success,
		Data:        response.Data,
		ServiceName: serviceName,
		CommandType: "OBJECT_ACCESS",
		Metadata:    response.Metadata,
	}, nil
}

// executeFieldOperation executes field operations (OBJECT:ID:field=value)
func (e *Engine) executeFieldOperation(ctx context.Context, cmd *mdwast.Command, execCtx *ExecutionContext) (*ExecutionResult, error) {
	// Check permissions
	method := "READ"
	if cmd.FieldOp.Op == "=" {
		method = "UPDATE"
	}
	
	if err := e.checkPermission(ctx, cmd.Object, method, execCtx); err != nil {
		return nil, err
	}

	// Get service for object
	serviceName, err := e.getServiceForObject(cmd.Object)
	if err != nil {
		return nil, err
	}

	// Prepare parameters
	params := map[string]interface{}{
		"id":    cmd.ObjectID,
		"field": cmd.FieldOp.Field,
	}

	if cmd.FieldOp.Op == "=" {
		params["value"] = cmd.FieldOp.Value
		method = "SET_FIELD"
	} else {
		method = "GET_FIELD"
	}

	// Execute service call
	response, err := e.client.Execute(ctx, serviceName, cmd.Object, method, params, execCtx)
	if err != nil {
		return nil, e.wrapServiceError(err, serviceName, cmd.Object, method)
	}

	return &ExecutionResult{
		Success:     response.Success,
		Data:        response.Data,
		ServiceName: serviceName,
		CommandType: "FIELD_OPERATION",
		Metadata:    response.Metadata,
	}, nil
}

// executeMethodCall executes method calls (OBJECT.METHOD)
func (e *Engine) executeMethodCall(ctx context.Context, cmd *mdwast.Command, execCtx *ExecutionContext) (*ExecutionResult, error) {
	// Handle built-in commands
	if cmd.Object == "ALIAS" || cmd.Object == "HELP" {
		return e.executeBuiltinCommand(ctx, cmd, execCtx)
	}

	// Check permissions
	if err := e.checkPermission(ctx, cmd.Object, cmd.Method, execCtx); err != nil {
		return nil, err
	}

	// Validate command exists in registry
	if e.registry != nil {
		if err := e.registry.ValidateCommand(cmd.Object, cmd.Method); err != nil {
			return nil, err
		}
	}

	// Get service for object
	serviceName, err := e.getServiceForObject(cmd.Object)
	if err != nil {
		return nil, err
	}

	// Convert AST values to interface{}
	params := make(map[string]interface{})
	for key, value := range cmd.Parameters {
		params[key] = value.Value
	}

	// Add filter if present
	if cmd.Filter != nil {
		params["_filter"] = e.serializeFilter(cmd.Filter)
	}

	// Execute service call
	response, err := e.client.Execute(ctx, serviceName, cmd.Object, cmd.Method, params, execCtx)
	if err != nil {
		return nil, e.wrapServiceError(err, serviceName, cmd.Object, cmd.Method)
	}

	return &ExecutionResult{
		Success:     response.Success,
		Data:        response.Data,
		ServiceName: serviceName,
		CommandType: "METHOD_CALL",
		Metadata:    response.Metadata,
	}, nil
}

// executeBuiltinCommand executes built-in TCOL commands
func (e *Engine) executeBuiltinCommand(ctx context.Context, cmd *mdwast.Command, execCtx *ExecutionContext) (*ExecutionResult, error) {
	switch cmd.Object {
	case "ALIAS":
		return e.executeAliasCommand(ctx, cmd, execCtx)
	case "HELP":
		return e.executeHelpCommand(ctx, cmd, execCtx)
	default:
		return nil, fmt.Errorf("unknown built-in command: %s", cmd.Object)
	}
}

// executeAliasCommand executes ALIAS commands
func (e *Engine) executeAliasCommand(ctx context.Context, cmd *mdwast.Command, execCtx *ExecutionContext) (*ExecutionResult, error) {
	if e.registry == nil {
		return nil, fmt.Errorf("registry not available for alias operations")
	}

	switch cmd.Method {
	case "CREATE":
		name, hasName := cmd.Parameters["name"]
		command, hasCommand := cmd.Parameters["command"]
		
		if !hasName || !hasCommand {
			return nil, fmt.Errorf("ALIAS.CREATE requires 'name' and 'command' parameters")
		}
		
		err := e.registry.RegisterAlias(name.Value.(string), command.Value.(string))
		if err != nil {
			return nil, err
		}
		
		return &ExecutionResult{
			Success:     true,
			Data:        fmt.Sprintf("Alias '%s' created successfully", name.Value),
			CommandType: "BUILTIN",
		}, nil

	case "LIST":
		aliases := e.registry.GetAliases()
		return &ExecutionResult{
			Success:     true,
			Data:        aliases,
			CommandType: "BUILTIN",
		}, nil

	default:
		return nil, fmt.Errorf("unknown ALIAS method: %s", cmd.Method)
	}
}

// executeHelpCommand executes HELP commands
func (e *Engine) executeHelpCommand(ctx context.Context, cmd *mdwast.Command, execCtx *ExecutionContext) (*ExecutionResult, error) {
	if e.registry == nil {
		return nil, fmt.Errorf("registry not available for help operations")
	}

	switch cmd.Method {
	case "LIST":
		objects := e.registry.GetObjectNames()
		return &ExecutionResult{
			Success:     true,
			Data:        objects,
			CommandType: "BUILTIN",
		}, nil

	case "OBJECT":
		name, hasName := cmd.Parameters["name"]
		if !hasName {
			return nil, fmt.Errorf("HELP.OBJECT requires 'name' parameter")
		}

		obj, err := e.registry.GetObject(name.Value.(string))
		if err != nil {
			return nil, err
		}

		return &ExecutionResult{
			Success:     true,
			Data:        obj,
			CommandType: "BUILTIN",
		}, nil

	default:
		return nil, fmt.Errorf("unknown HELP method: %s", cmd.Method)
	}
}

// Utility methods

// checkPermission checks if the user has permission to execute the command
func (e *Engine) checkPermission(ctx context.Context, objectName, methodName string, execCtx *ExecutionContext) error {
	if e.permissions == nil {
		return nil // No permission checker configured
	}

	return e.permissions.CheckPermission(ctx, execCtx.UserID, objectName, methodName, execCtx)
}

// getServiceForObject gets the service name for an object
func (e *Engine) getServiceForObject(objectName string) (string, error) {
	if e.registry == nil {
		return "", fmt.Errorf("registry not available")
	}

	return e.registry.GetServiceForObject(objectName)
}

// wrapServiceError wraps service errors with TCOL context
func (e *Engine) wrapServiceError(err error, serviceName, objectName, methodName string) error {
	return fmt.Errorf("service call failed: service=%s, object=%s, method=%s: %w", 
		serviceName, objectName, methodName, err)
}

// serializeFilter serializes a filter expression for service calls
func (e *Engine) serializeFilter(filter *mdwast.FilterExpr) map[string]interface{} {
	return map[string]interface{}{
		"condition": e.serializeExpression(filter.Condition),
	}
}

// serializeExpression serializes an AST expression
func (eng *Engine) serializeExpression(expr mdwast.Expr) interface{} {
	switch e := expr.(type) {
	case *mdwast.BinaryExpr:
		return map[string]interface{}{
			"type":  "binary",
			"op":    e.Op,
			"left":  eng.serializeExpression(e.Left),
			"right": eng.serializeExpression(e.Right),
		}
	case *mdwast.UnaryExpr:
		return map[string]interface{}{
			"type": "unary",
			"op":   e.Op,
			"expr": eng.serializeExpression(e.Expr),
		}
	case *mdwast.IdentifierExpr:
		return map[string]interface{}{
			"type": "identifier",
			"name": e.Name,
		}
	case *mdwast.LiteralExpr:
		return map[string]interface{}{
			"type":  "literal",
			"value": e.Value.Value,
		}
	default:
		return map[string]interface{}{
			"type": "unknown",
		}
	}
}

// auditCommand logs command execution for audit purposes
func (e *Engine) auditCommand(cmd *mdwast.Command, execCtx *ExecutionContext, status string) {
	e.logger.Audit("TCOL command execution", mdwlog.Fields{
		"requestID":   execCtx.RequestID,
		"userID":      execCtx.UserID,
		"sessionID":   execCtx.SessionID,
		"clientIP":    execCtx.ClientIP,
		"object":      cmd.Object,
		"method":      cmd.Method,
		"objectID":    cmd.ObjectID,
		"status":      status,
		"timestamp":   execCtx.Timestamp,
		"chainDepth":  execCtx.ChainDepth,
	})
}

// Close closes the executor and releases resources
func (e *Engine) Close() error {
	if e.client != nil {
		return e.client.Close()
	}
	return nil
}