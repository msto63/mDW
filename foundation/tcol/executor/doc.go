// File: doc.go
// Title: TCOL Executor Package Documentation
// Description: Implements the execution engine for TCOL commands. Takes parsed
//              AST nodes and executes them by routing commands to appropriate
//              services, handling responses, and managing execution context.
// Author: msto63 with Claude Sonnet 4.0
// Version: v0.1.0
// Created: 2025-01-25
// Modified: 2025-01-25
//
// Change History:
// - 2025-01-25 v0.1.0: Initial executor implementation

/*
Package executor provides command execution capabilities for TCOL.

This package implements the execution engine that takes parsed TCOL commands
(represented as AST nodes) and executes them by:

  • Routing commands to appropriate microservices
  • Managing execution context and permissions
  • Handling command responses and errors
  • Supporting command chaining and composition
  • Providing audit logging and monitoring

The executor integrates with the mDW Foundation's error handling, logging,
and service communication infrastructure to provide secure and reliable
command execution.
*/
package executor