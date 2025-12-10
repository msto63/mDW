// Package errors provides THE STANDARD error handling interface for all mDW foundation
// modules. This is the primary error handling API that all modules should use.
//
// Package: errors  
// Title: Standard Error Handling API for mDW Foundation
// Description: This package provides common error patterns, standardized error
//              codes, and utilities for creating consistent errors across all
//              mDW foundation modules. It integrates with the core error package
//              to provide module-specific error handling while maintaining
//              consistency and enabling better error analysis and monitoring.
// Author: msto63 with Claude Sonnet 4.0
// Version: v0.1.0
// Created: 2025-01-25
// Modified: 2025-01-25
//
// Change History:
// - 2025-01-25 v0.1.0: Initial implementation for cross-module error standardization
//
// Package Overview:
//
// The errors package serves as the foundation for consistent error handling
// across all mDW modules, providing:
//
// # Standardized Error Codes
//
// Module-specific error codes for consistent error categorization:
//   - Common codes: INVALID_INPUT, INVALID_FORMAT, OUT_OF_RANGE, etc.
//   - stringx codes: STRINGX_INVALID_FORMAT, STRINGX_LENGTH_EXCEEDED, etc.
//   - mathx codes: MATHX_PRECISION_LOSS, MATHX_DIVISION_BY_ZERO, etc.
//   - timex codes: TIMEX_INVALID_FORMAT, TIMEX_PARSE_ERROR, etc.
//   - filex codes: FILEX_NOT_FOUND, FILEX_PERMISSION_DENIED, etc.
//   - And codes for all other foundation modules
//
// # Error Creation Utilities
//
// Standardized functions for creating module-specific errors:
//   - StandardError: Basic module error with automatic code assignment
//   - ModuleError: Wraps errors with module context and details
//   - ValidationError: Specialized for validation failures
//   - InputError: For invalid input parameters
//   - FormatError: For format-related errors
//   - OperationError: For operation failures
//
// # Error Analysis Functions
//
// Utilities for analyzing and working with standardized errors:
//   - IsModuleError: Check if error belongs to specific module
//   - GetErrorModule: Extract module name from error
//   - GetErrorOperation: Extract operation name from error
//
// # Usage Examples
//
// Creating standardized module errors:
//
//	// Basic module error
//	err := errors.StandardError("stringx", "format", "invalid string format")
//
//	// Error with wrapped cause
//	err = errors.ModuleError("mathx", "parse_decimal", parseErr, map[string]interface{}{
//		"input": "invalid_decimal",
//		"expected": "valid decimal string",
//	})
//
//	// Validation error
//	err = errors.ValidationError("validationx", "email", "invalid@", "must be valid email")
//
//	// Input validation error
//	err = errors.InputError("slicex", "filter", nil, "non-nil slice")
//
//	// Format error
//	err = errors.FormatError("timex", "2023-13-45", "YYYY-MM-DD")
//
// Using in module functions:
//
//	func ParseDecimal(s string) (Decimal, error) {
//		if strings.TrimSpace(s) == "" {
//			return Decimal{}, errors.InputError("mathx", "parse_decimal", s, "non-empty string")
//		}
//		
//		rat := new(big.Rat)
//		if _, ok := rat.SetString(s); !ok {
//			return Decimal{}, errors.FormatError("mathx", s, "valid decimal string")
//		}
//		
//		return Decimal{value: rat}, nil
//	}
//
//	func ReadFile(path string) ([]byte, error) {
//		if path == "" {
//			return nil, errors.InputError("filex", "read_file", path, "non-empty file path")
//		}
//		
//		data, err := os.ReadFile(path)
//		if err != nil {
//			return nil, errors.ModuleError("filex", "read_file", err, map[string]interface{}{
//				"path": path,
//			})
//		}
//		
//		return data, nil
//	}
//
// Error analysis and handling:
//
//	if err != nil {
//		// Check if error is from specific module
//		if errors.IsModuleError(err, "filex") {
//			log.Error("File operation failed", "error", err)
//		}
//		
//		// Extract error context
//		module := errors.GetErrorModule(err)
//		operation := errors.GetErrorOperation(err)
//		log.Info("Error details", "module", module, "operation", operation)
//	}
//
// # Integration with Core Error Package
//
// This package builds on the core error package to provide:
//   - Automatic error code assignment based on module and operation
//   - Consistent severity levels based on error type
//   - Standardized error details structure
//   - Module-specific error categorization
//
// # Error Code Patterns
//
// Error codes follow consistent patterns:
//   - Format: {MODULE}_{CATEGORY} (e.g., STRINGX_INVALID_FORMAT)
//   - Common categories: INVALID_FORMAT, OPERATION_FAILED, NOT_FOUND
//   - Module-specific categories based on domain (e.g., MATHX_DIVISION_BY_ZERO)
//
// # Benefits
//
// Using this standardized error handling provides:
//   - Consistent error messages and codes across modules
//   - Better error monitoring and alerting capabilities
//   - Easier debugging with structured error information
//   - Uniform error handling patterns for developers
//   - Integration with logging and observability systems
//
// # Module Integration
//
// All mDW foundation modules should use these error utilities:
//   - stringx: String manipulation errors
//   - mathx: Mathematical calculation errors
//   - mapx: Map operation errors
//   - slicex: Slice operation errors
//   - timex: Time parsing and calculation errors
//   - validationx: Input validation errors
//   - filex: File operation errors
//
// This ensures consistent error handling across the entire mDW platform
// and enables better error analysis and system monitoring.
package errors