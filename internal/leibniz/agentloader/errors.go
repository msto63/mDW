// ============================================================================
// meinDENKWERK (mDW) - Lokale KI-Plattform
// ============================================================================
//
// Package:     agentloader
// Description: Error definitions for agent loader
// Author:      Mike Stoffels with Claude
// Created:     2025-12-11
// License:     MIT
// ============================================================================

package agentloader

import "errors"

var (
	// Validation errors
	ErrMissingID           = errors.New("agent ID is required")
	ErrMissingName         = errors.New("agent name is required")
	ErrMissingSystemPrompt = errors.New("agent system_prompt is required")

	// Loading errors
	ErrAgentNotFound  = errors.New("agent not found")
	ErrInvalidYAML    = errors.New("invalid YAML syntax")
	ErrAgentExists    = errors.New("agent with this ID already exists")
	ErrLoadFailed     = errors.New("failed to load agent")
	ErrDirectoryEmpty = errors.New("agents directory is empty or does not exist")
)
