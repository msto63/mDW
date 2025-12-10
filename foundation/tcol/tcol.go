// File: tcol.go
// Title: TCOL Main Interface and Engine
// Description: Provides the main TCOL engine interface and high-level API
//              for parsing and executing TCOL commands. Integrates parser,
//              AST, executor, and registry components.
// Author: msto63 with Claude Sonnet 4.0
// Version: v0.1.0
// Created: 2025-01-25
// Modified: 2025-01-25
//
// Change History:
// - 2025-01-25 v0.1.0: Initial TCOL engine implementation

package tcol

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	mdwlog "github.com/msto63/mDW/foundation/core/log"
	mdwast "github.com/msto63/mDW/foundation/tcol/ast"
	mdwexecutor "github.com/msto63/mDW/foundation/tcol/executor"
	mdwparser "github.com/msto63/mDW/foundation/tcol/parser"
	mdwregistry "github.com/msto63/mDW/foundation/tcol/registry"
	mdwstringx "github.com/msto63/mDW/foundation/utils/stringx"
)

// Engine represents the main TCOL engine that coordinates parsing and execution
type Engine struct {
	parser   *mdwparser.Parser
	executor *mdwexecutor.Engine
	registry *mdwregistry.Registry
	logger   *mdwlog.Logger
	options  Options
}

// Options configures the TCOL engine behavior
type Options struct {
	// Logger for TCOL operations (optional, defaults to default logger)
	Logger *mdwlog.Logger

	// LogLevel for TCOL-specific logging
	LogLevel mdwlog.Level

	// MaxCommandLength limits input command length (default: 4096)
	MaxCommandLength int

	// Services lists available service names for command routing
	Services []string

	// EnableAbbreviations allows command abbreviation expansion (default: true)
	EnableAbbreviations bool

	// EnableAliases allows user-defined command aliases (default: true)
	EnableAliases bool

	// EnableChaining allows command chaining with pipes (default: true)
	EnableChaining bool

	// ExecutionTimeout sets maximum command execution time (default: 30s)
	ExecutionTimeout time.Duration

	// PermissionChecker validates user permissions for commands
	PermissionChecker PermissionChecker

	// AuditLogger logs all command executions for compliance
	AuditLogger AuditLogger

	// ServiceClient for communicating with microservices (optional for testing)
	ServiceClient mdwexecutor.ServiceClient
}

// Result represents the result of a TCOL command execution
type Result struct {
	// Success indicates if the command executed successfully
	Success bool

	// Data contains the command result data
	Data []interface{}

	// Message contains human-readable result message
	Message string

	// ExecutionTime is the time taken to execute the command
	ExecutionTime time.Duration

	// Command is the original command that was executed
	Command string

	// ParsedCommand is the parsed AST representation
	ParsedCommand *mdwast.Command

	// Metadata contains additional result information
	Metadata map[string]interface{}
}

// Error represents a TCOL-specific error with additional context
type Error struct {
	Err      error
	command  string
	position int
	object   string
	method   string
}

// Error implements the error interface
func (e *Error) Error() string {
	if e.Err != nil {
		return e.Err.Error()
	}
	return "TCOL error"
}

// PermissionChecker interface for validating user permissions
type PermissionChecker interface {
	// HasPermission checks if user has permission for object.method
	HasPermission(ctx context.Context, userID, object, method string) bool
}

// AuditLogger interface for logging command executions
type AuditLogger interface {
	// LogExecution logs a command execution for audit purposes
	LogExecution(ctx context.Context, cmd *mdwast.Command, result *Result, err error)
}

// Command creates a new TCOL command result
func (r *Result) AddData(item interface{}) {
	if r.Data == nil {
		r.Data = make([]interface{}, 0)
	}
	r.Data = append(r.Data, item)
}

// SetMetadata sets metadata value
func (r *Result) SetMetadata(key string, value interface{}) {
	if r.Metadata == nil {
		r.Metadata = make(map[string]interface{})
	}
	r.Metadata[key] = value
}

// Error methods for TCOL-specific error information
func (e *Error) Command() string  { return e.command }
func (e *Error) Position() int    { return e.position }
func (e *Error) Object() string   { return e.object }
func (e *Error) Method() string   { return e.method }

// NewEngine creates a new TCOL engine with the specified options
func NewEngine(opts ...Options) (*Engine, error) {
	// Default options
	options := Options{
		Logger:              mdwlog.GetDefault(),
		LogLevel:            mdwlog.LevelInfo,
		MaxCommandLength:    4096,
		Services:            []string{},
		EnableAbbreviations: true,
		EnableAliases:       true,
		EnableChaining:      true,
		ExecutionTimeout:    30 * time.Second,
	}

	// Apply provided options
	if len(opts) > 0 {
		provided := opts[0]
		if provided.Logger != nil {
			options.Logger = provided.Logger
		}
		if provided.LogLevel != 0 {
			options.LogLevel = provided.LogLevel
		}
		if provided.MaxCommandLength > 0 {
			options.MaxCommandLength = provided.MaxCommandLength
		}
		if len(provided.Services) > 0 {
			options.Services = provided.Services
		}
		if provided.ExecutionTimeout > 0 {
			options.ExecutionTimeout = provided.ExecutionTimeout
		}
		options.EnableAbbreviations = provided.EnableAbbreviations
		options.EnableAliases = provided.EnableAliases
		options.EnableChaining = provided.EnableChaining
		options.PermissionChecker = provided.PermissionChecker
		options.AuditLogger = provided.AuditLogger
		options.ServiceClient = provided.ServiceClient
	}

	// Create logger with TCOL context
	logger := options.Logger.WithField("component", "tcol-engine")

	// Create registry
	reg, err := mdwregistry.NewSimple(mdwregistry.Options{
		Logger:              logger,
		Services:            options.Services,
		EnableAbbreviations: options.EnableAbbreviations,
		EnableAliases:       options.EnableAliases,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to initialize TCOL registry: %w", err)
	}

	// Create parser
	p, err := mdwparser.New(mdwparser.Options{
		Logger:           logger,
		MaxInputLength:   options.MaxCommandLength,
		EnableChaining:   options.EnableChaining,
		Registry:         reg,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to initialize TCOL parser: %w", err)
	}

	// Create executor with service client if provided
	exec, err := mdwexecutor.New(mdwexecutor.Options{
		Logger:        logger,
		ServiceClient: options.ServiceClient,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to initialize TCOL executor: %w", err)
	}
	exec.SetRegistry(reg)

	engine := &Engine{
		parser:   p,
		executor: exec,
		registry: reg,
		logger:   logger,
		options:  options,
	}

	logger.Info("TCOL engine initialized", mdwlog.Fields{
		"maxCommandLength":    options.MaxCommandLength,
		"servicesCount":       len(options.Services),
		"enableAbbreviations": options.EnableAbbreviations,
		"enableAliases":       options.EnableAliases,
		"enableChaining":      options.EnableChaining,
		"executionTimeout":    options.ExecutionTimeout,
	})

	return engine, nil
}

// Execute parses and executes a TCOL command
func (e *Engine) Execute(ctx context.Context, command string) (*Result, error) {
	// Create execution timer
	timer := e.logger.StartTimer("tcol_command_execution")
	defer timer.Stop()

	// Log command execution start
	e.logger.Info("Executing TCOL command", mdwlog.Fields{
		"command":   command,
		"requestId": ctx.Value("requestId"),
		"userId":    ctx.Value("userId"),
	})

	// Validate input
	if err := e.validateInput(command); err != nil {
		timer.StopWithError(err)
		return nil, err
	}

	timer.Checkpoint("input_validated")

	// Parse command
	parsedCmd, err := e.parser.Parse(command)
	if err != nil {
		timer.StopWithError(err)
		e.logger.Warn("TCOL parsing failed", mdwlog.Fields{
			"command": command,
			"error":   err.Error(),
		})
		return nil, e.wrapParseError(err, command)
	}

	timer.Checkpoint("command_parsed")

	// Check permissions if checker is configured
	if e.options.PermissionChecker != nil {
		userID, ok := ctx.Value("userId").(string)
		if !ok {
			err := errors.New("user ID not found in context")
			timer.StopWithError(err)
			return nil, err
		}

		if !e.options.PermissionChecker.HasPermission(ctx, userID, parsedCmd.Object, parsedCmd.Method) {
			err := fmt.Errorf("insufficient permissions for command - user: %s, object: %s, method: %s", userID, parsedCmd.Object, parsedCmd.Method)

			timer.StopWithError(err)
			return nil, err
		}
	}

	timer.Checkpoint("permissions_checked")

	// Execute command
	execCtx := &mdwexecutor.ExecutionContext{
		RequestID: fmt.Sprintf("req-%d", time.Now().UnixNano()),
		Timestamp: time.Now(),
		Metadata:  make(map[string]interface{}),
	}
	result, err := e.executor.Execute(ctx, parsedCmd, execCtx)
	if err != nil {
		timer.StopWithError(err)
		e.logger.Error("TCOL execution failed", mdwlog.Fields{
			"command": command,
			"object":  parsedCmd.Object,
			"method":  parsedCmd.Method,
			"error":   err.Error(),
		})

		// Audit log the failed execution
		if e.options.AuditLogger != nil {
			e.options.AuditLogger.LogExecution(ctx, parsedCmd, nil, err)
		}

		return nil, e.wrapExecutionError(err, command, parsedCmd)
	}

	timer.Checkpoint("command_executed")

	// Create result
	var resultData []interface{}
	if result.Data != nil {
		if slice, ok := result.Data.([]interface{}); ok {
			resultData = slice
		} else {
			resultData = []interface{}{result.Data}
		}
	}
	
	tcolResult := &Result{
		Success:       true,
		Data:          resultData,
		Message:       "Command executed successfully",
		ExecutionTime: time.Since(timer.StartTime()),
		Command:       command,
		ParsedCommand: parsedCmd,
		Metadata:      result.Metadata,
	}

	// Log successful execution
	e.logger.Info("TCOL command executed successfully", mdwlog.Fields{
		"command":     command,
		"object":      parsedCmd.Object,
		"method":      parsedCmd.Method,
		"resultCount": len(tcolResult.Data),
		"duration":    tcolResult.ExecutionTime,
	})

	// Audit log the successful execution
	if e.options.AuditLogger != nil {
		e.options.AuditLogger.LogExecution(ctx, parsedCmd, tcolResult, nil)
	}

	return tcolResult, nil
}

// Parse parses a TCOL command without executing it
func (e *Engine) Parse(command string) (*mdwast.Command, error) {
	// Validate input
	if err := e.validateInput(command); err != nil {
		return nil, err
	}

	// Parse command
	return e.parser.Parse(command)
}

// Registry returns the command registry for registration of custom objects and methods
func (e *Engine) Registry() *mdwregistry.Registry {
	return e.registry
}

// ValidateCommand checks if a command is syntactically valid
func (e *Engine) ValidateCommand(command string) error {
	_, err := e.Parse(command)
	return err
}

// GetAbbreviations returns all available command abbreviations
func (e *Engine) GetAbbreviations() map[string]string {
	return e.registry.GetAbbreviations()
}

// GetAliases returns all available command aliases
func (e *Engine) GetAliases() map[string]string {
	return e.registry.GetAliases()
}

// validateInput validates the input command string
func (e *Engine) validateInput(command string) error {
	// Check for empty input
	if mdwstringx.IsBlank(command) {
		return errors.New("command input cannot be empty")
	}

	// Check length limits
	if len(command) > e.options.MaxCommandLength {
		return fmt.Errorf("command input exceeds maximum length: %d > %d", len(command), e.options.MaxCommandLength)
	}

	// Basic security checks
	if strings.Contains(command, ";DROP TABLE") ||
		strings.Contains(command, "'; DROP TABLE") ||
		strings.Contains(command, "UNION SELECT") {
		return fmt.Errorf("potential security violation detected in command: %s", command)
	}

	return nil
}

// wrapParseError wraps parsing errors with TCOL-specific context
func (e *Engine) wrapParseError(err error, command string) error {
	if tcolErr, ok := err.(*Error); ok {
		return tcolErr
	}

	return &Error{
		Err:     fmt.Errorf("failed to parse TCOL command: %w", err),
		command: command,
	}
}

// wrapExecutionError wraps execution errors with TCOL-specific context
func (e *Engine) wrapExecutionError(err error, command string, cmd *mdwast.Command) error {
	if tcolErr, ok := err.(*Error); ok {
		return tcolErr
	}

	return &Error{
		Err:     fmt.Errorf("failed to execute TCOL command: %w", err),
		command: command,
		object:  cmd.Object,
		method:  cmd.Method,
	}
}

// String returns a string representation of the result
func (r *Result) String() string {
	if !r.Success {
		return fmt.Sprintf("FAILED: %s", r.Message)
	}

	return fmt.Sprintf("SUCCESS: %s (Items: %d, Duration: %v)",
		r.Message, len(r.Data), r.ExecutionTime)
}

// IsEmpty returns true if the result contains no data
func (r *Result) IsEmpty() bool {
	return len(r.Data) == 0
}

// Count returns the number of items in the result
func (r *Result) Count() int {
	return len(r.Data)
}