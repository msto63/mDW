// Package error provides comprehensive error handling capabilities for the mDW platform.
//
// Package: error
// Title: mDW Error Handling Framework
// Description: This package implements a structured error handling system with contextual
//              information, error codes, stack traces, and integration with logging and
//              monitoring systems. It provides a foundation for consistent error handling
//              across all mDW services and supports multi-language error messages.
// Author: msto63 with Claude Sonnet 4.0
// Version: v0.1.0
// Created: 2025-01-24
// Modified: 2025-01-24
//
// Change History:
// - 2025-01-24 v0.1.0: Initial implementation with contextual errors and codes
//
// Features:
// - Contextual error wrapping with additional metadata
// - Structured error codes for consistent API responses
// - Stack trace capture for debugging
// - Integration with logging and monitoring systems
// - Multi-language error message support
// - Error severity levels and categorization
// - Custom error types for specific business domains
//
// Usage:
//   import "github.com/msto63/mDW/foundation/core/error"
//
//   // Create a new error with context
//   err := error.New("database connection failed").
//     WithCode(error.CodeDatabaseError).
//     WithDetail("host", "localhost:5432").
//     WithSeverity(error.SeverityHigh)
//
//   // Wrap an existing error with context
//   wrapped := error.Wrap(err, "failed to initialize user service").
//     WithCode(error.CodeServiceInitialization).
//     WithDetail("service", "user")
//
//   // Check error type and code
//   if error.HasCode(err, error.CodeDatabaseError) {
//     // Handle database errors specifically
//   }
package error