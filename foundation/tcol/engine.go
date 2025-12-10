// File: engine.go
// Title: TCOL High-Level Engine Interface
// Description: Provides a high-level interface for the TCOL engine that
//              integrates parser, executor, and registry components for
//              command processing. Compatible with the existing tcol.go API.
// Author: msto63 with Claude Sonnet 4.0
// Version: v0.1.0
// Created: 2025-01-25
// Modified: 2025-01-25
//
// Change History:
// - 2025-01-25 v0.1.0: Initial high-level engine implementation

package tcol

import (
	"context"
	"fmt"

	mdwlog "github.com/msto63/mDW/foundation/core/log"
	mdwast "github.com/msto63/mDW/foundation/tcol/ast"
	mdwexecutor "github.com/msto63/mDW/foundation/tcol/executor"
	mdwparser "github.com/msto63/mDW/foundation/tcol/parser"
	mdwregistry "github.com/msto63/mDW/foundation/tcol/registry"
	mdwstringx "github.com/msto63/mDW/foundation/utils/stringx"
)

// HighLevelEngine provides a simplified interface to the TCOL system
type HighLevelEngine struct {
	parser   *mdwparser.Parser
	executor *mdwexecutor.Engine
	registry *mdwregistry.Registry
	logger   *mdwlog.Logger
	options  HighLevelOptions
}

// HighLevelOptions configures the high-level TCOL engine
type HighLevelOptions struct {
	Logger              *mdwlog.Logger
	Registry            *mdwregistry.Registry
	Executor            *mdwexecutor.Engine
	Parser              *mdwparser.Parser
	MaxCommandLength    int
	EnableAbbreviations bool
	EnableAliases       bool
	EnableChaining      bool
}

// ExecutionContext provides context for command execution
type ExecutionContext = mdwexecutor.ExecutionContext

// ExecutionResult represents the result of command execution
type ExecutionResult = mdwexecutor.ExecutionResult

// New creates a new high-level TCOL engine
func New(opts HighLevelOptions) (*HighLevelEngine, error) {
	// Set defaults
	if opts.Logger == nil {
		opts.Logger = mdwlog.GetDefault()
	}
	if opts.MaxCommandLength == 0 {
		opts.MaxCommandLength = 4096
	}

	logger := opts.Logger.WithField("component", "tcol-engine")

	// Create registry if not provided
	if opts.Registry == nil {
		reg, err := mdwregistry.NewSimple(mdwregistry.Options{
			Logger:              logger,
			EnableAbbreviations: opts.EnableAbbreviations,
			EnableAliases:       opts.EnableAliases,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to initialize TCOL registry: %w", err)
		}
		opts.Registry = reg
	}

	// Create parser if not provided
	if opts.Parser == nil {
		p, err := mdwparser.New(mdwparser.Options{
			Logger:         logger,
			MaxInputLength: opts.MaxCommandLength,
			EnableChaining: opts.EnableChaining,
			Registry:       opts.Registry,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to initialize TCOL parser: %w", err)
		}
		opts.Parser = p
	}

	// Executor is optional for parse-only usage
	if opts.Executor != nil {
		opts.Executor.SetRegistry(opts.Registry)
	}

	engine := &HighLevelEngine{
		parser:   opts.Parser,
		executor: opts.Executor,
		registry: opts.Registry,
		logger:   logger,
		options:  opts,
	}

	logger.Info("High-level TCOL engine initialized", mdwlog.Fields{
		"maxCommandLength":    opts.MaxCommandLength,
		"enableAbbreviations": opts.EnableAbbreviations,
		"enableAliases":       opts.EnableAliases,
		"enableChaining":      opts.EnableChaining,
		"hasExecutor":         opts.Executor != nil,
	})

	return engine, nil
}

// Execute parses and executes a TCOL command
func (e *HighLevelEngine) Execute(ctx context.Context, command string, execCtx *ExecutionContext) (*ExecutionResult, error) {
	if mdwstringx.IsBlank(command) {
		return nil, fmt.Errorf("command cannot be empty")
	}

	e.logger.Debug("Executing TCOL command", mdwlog.Fields{
		"command":   command,
		"requestID": execCtx.RequestID,
		"userID":    execCtx.UserID,
	})

	// Parse the command
	cmd, err := e.parser.Parse(command)
	if err != nil {
		return nil, fmt.Errorf("failed to parse TCOL command: %w", err)
	}

	// Execute the command if executor is available
	if e.executor != nil {
		result, err := e.executor.Execute(ctx, cmd, execCtx)
		if err != nil {
			return nil, fmt.Errorf("failed to execute TCOL command: %w", err)
		}
		return result, nil
	}

	// If no executor, return parse-only result
	return &ExecutionResult{
		Success:     true,
		Data:        cmd,
		CommandType: "PARSE_ONLY",
	}, nil
}

// Parse parses a TCOL command without executing it
func (e *HighLevelEngine) Parse(command string) (*mdwast.Command, error) {
	if mdwstringx.IsBlank(command) {
		return nil, fmt.Errorf("command cannot be empty")
	}

	return e.parser.Parse(command)
}

// Registry returns the command registry
func (e *HighLevelEngine) Registry() *mdwregistry.Registry {
	return e.registry
}

// ValidateCommand checks if a command is syntactically valid
func (e *HighLevelEngine) ValidateCommand(command string) error {
	_, err := e.Parse(command)
	return err
}

// GetAbbreviations returns all available command abbreviations
func (e *HighLevelEngine) GetAbbreviations() map[string]string {
	if e.registry == nil {
		return make(map[string]string)
	}
	return e.registry.GetAbbreviations()
}

// GetAliases returns all available command aliases
func (e *HighLevelEngine) GetAliases() map[string]string {
	if e.registry == nil {
		return make(map[string]string)
	}
	return e.registry.GetAliases()
}

// Close closes the engine and releases resources
func (e *HighLevelEngine) Close() error {
	if e.executor != nil {
		return e.executor.Close()
	}
	return nil
}