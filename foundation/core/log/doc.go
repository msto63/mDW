// Package log provides structured logging capabilities for the mDW platform.
//
// Package: log
// Title: mDW Structured Logging Framework
// Description: This package implements a comprehensive structured logging system with
//              contextual information, multiple output formats, log levels, and tight
//              integration with the mDW error handling system. It supports performance
//              monitoring, audit trails, and distributed tracing for microservices.
// Author: msto63 with Claude Sonnet 4.0
// Version: v0.1.0
// Created: 2025-01-24
// Modified: 2025-01-24
//
// Change History:
// - 2025-01-24 v0.1.0: Initial implementation with structured logging and error integration
//
// Features:
// - Structured logging with JSON and text formats
// - Multiple log levels with filtering capabilities
// - Contextual logging with request IDs, user IDs, and custom fields
// - Integration with mDW error system for automatic error logging
// - Performance metrics and timing measurements
// - Audit trail capabilities for TCOL commands
// - Multiple output destinations (console, file, remote)
// - Log sampling and rate limiting for high-volume scenarios
// - Correlation IDs for distributed tracing
//
// Usage:
//   import mdwlog "github.com/msto63/mDW/foundation/core/log"
//
//   // Create a logger with context
//   logger := log.New().
//     WithLevel(log.LevelInfo).
//     WithFormat(log.FormatJSON).
//     WithField("service", "user-service").
//     WithRequestID("req-123")
//
//   // Log messages with different levels
//   logger.Info("User created successfully", log.Field("user_id", "user123"))
//   logger.Error("Database connection failed", log.Err(err))
//   logger.Debug("Processing request", log.Fields{
//     "method": "POST",
//     "path":   "/api/users",
//     "body_size": 1024,
//   })
//
//   // Log performance metrics
//   timer := logger.StartTimer("database_query")
//   // ... perform database operation
//   timer.Stop()
//
//   // Audit logging for TCOL commands
//   logger.Audit("TCOL command executed", log.Fields{
//     "command": "CUSTOMER.CREATE",
//     "user_id": "admin123",
//     "success": true,
//   })
package log