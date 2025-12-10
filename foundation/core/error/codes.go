// File: codes.go
// Title: Error Code Definitions
// Description: Defines standardized error codes for consistent error classification
//              across the mDW platform. These codes enable structured error handling,
//              API response formatting, and error monitoring.
// Author: msto63 with Claude Sonnet 4.0
// Version: v0.1.0
// Created: 2025-01-24
// Modified: 2025-01-24
//
// Change History:
// - 2025-01-24 v0.1.0: Initial implementation with core error codes

package error

// Code represents a structured error code for categorizing errors
type Code string

// Core error codes for the mDW platform
const (
	// Generic codes
	CodeUnknown     Code = "UNKNOWN"
	CodeInternal    Code = "INTERNAL"
	CodeNotFound    Code = "NOT_FOUND"
	CodeInvalidInput Code = "INVALID_INPUT"
	CodeTimeout     Code = "TIMEOUT"
	
	// Authentication and authorization
	CodeUnauthorized    Code = "UNAUTHORIZED"
	CodeForbidden       Code = "FORBIDDEN"
	CodeInvalidToken    Code = "INVALID_TOKEN"
	CodeExpiredToken    Code = "EXPIRED_TOKEN"
	CodeInvalidCredentials Code = "INVALID_CREDENTIALS"
	
	// Database and storage
	CodeDatabaseError      Code = "DATABASE_ERROR"
	CodeConnectionFailed   Code = "CONNECTION_FAILED"
	CodeDataCorruption     Code = "DATA_CORRUPTION"
	CodeConstraintViolation Code = "CONSTRAINT_VIOLATION"
	CodeDuplicateEntry     Code = "DUPLICATE_ENTRY"
	
	// Business logic
	CodeBusinessRule       Code = "BUSINESS_RULE"
	CodeInsufficientFunds  Code = "INSUFFICIENT_FUNDS"
	CodeInvalidOperation   Code = "INVALID_OPERATION"
	CodeResourceLocked     Code = "RESOURCE_LOCKED"
	CodeQuotaExceeded      Code = "QUOTA_EXCEEDED"
	
	// Service and network
	CodeServiceUnavailable Code = "SERVICE_UNAVAILABLE"
	CodeNetworkError       Code = "NETWORK_ERROR"
	CodeServiceTimeout     Code = "SERVICE_TIMEOUT"
	CodeServiceInitialization Code = "SERVICE_INITIALIZATION"
	CodeExternalServiceError Code = "EXTERNAL_SERVICE_ERROR"
	
	// TCOL specific
	CodeTCOLSyntax         Code = "TCOL_SYNTAX"
	CodeTCOLSemantic       Code = "TCOL_SEMANTIC"
	CodeTCOLPermission     Code = "TCOL_PERMISSION"
	CodeTCOLExecution      Code = "TCOL_EXECUTION"
	CodeTCOLObjectNotFound Code = "TCOL_OBJECT_NOT_FOUND"
	
	// Configuration and environment
	CodeConfigError     Code = "CONFIG_ERROR"
	CodeMissingConfig   Code = "MISSING_CONFIG"
	CodeInvalidConfig   Code = "INVALID_CONFIG"
	CodeEnvironmentError Code = "ENVIRONMENT_ERROR"
	
	// Validation
	CodeValidationFailed   Code = "VALIDATION_FAILED"
	CodeRequiredField      Code = "REQUIRED_FIELD"
	CodeInvalidFormat      Code = "INVALID_FORMAT"
	CodeValueOutOfRange    Code = "VALUE_OUT_OF_RANGE"
	CodeInvalidLength      Code = "INVALID_LENGTH"
)

// String returns the string representation of the error code
func (c Code) String() string {
	return string(c)
}

// IsValid checks if the error code is a known valid code
func (c Code) IsValid() bool {
	switch c {
	case CodeUnknown, CodeInternal, CodeNotFound, CodeInvalidInput, CodeTimeout,
		 CodeUnauthorized, CodeForbidden, CodeInvalidToken, CodeExpiredToken, CodeInvalidCredentials,
		 CodeDatabaseError, CodeConnectionFailed, CodeDataCorruption, CodeConstraintViolation, CodeDuplicateEntry,
		 CodeBusinessRule, CodeInsufficientFunds, CodeInvalidOperation, CodeResourceLocked, CodeQuotaExceeded,
		 CodeServiceUnavailable, CodeNetworkError, CodeServiceTimeout, CodeServiceInitialization, CodeExternalServiceError,
		 CodeTCOLSyntax, CodeTCOLSemantic, CodeTCOLPermission, CodeTCOLExecution, CodeTCOLObjectNotFound,
		 CodeConfigError, CodeMissingConfig, CodeInvalidConfig, CodeEnvironmentError,
		 CodeValidationFailed, CodeRequiredField, CodeInvalidFormat, CodeValueOutOfRange, CodeInvalidLength:
		return true
	default:
		return false
	}
}

// Category returns the high-level category of the error code
func (c Code) Category() string {
	switch c {
	case CodeUnauthorized, CodeForbidden, CodeInvalidToken, CodeExpiredToken, CodeInvalidCredentials:
		return "authentication"
	case CodeDatabaseError, CodeConnectionFailed, CodeDataCorruption, CodeConstraintViolation, CodeDuplicateEntry:
		return "database"
	case CodeBusinessRule, CodeInsufficientFunds, CodeInvalidOperation, CodeResourceLocked, CodeQuotaExceeded:
		return "business"
	case CodeServiceUnavailable, CodeNetworkError, CodeServiceTimeout, CodeServiceInitialization, CodeExternalServiceError:
		return "service"
	case CodeTCOLSyntax, CodeTCOLSemantic, CodeTCOLPermission, CodeTCOLExecution, CodeTCOLObjectNotFound:
		return "tcol"
	case CodeConfigError, CodeMissingConfig, CodeInvalidConfig, CodeEnvironmentError:
		return "configuration"
	case CodeValidationFailed, CodeRequiredField, CodeInvalidFormat, CodeValueOutOfRange, CodeInvalidLength:
		return "validation"
	default:
		return "generic"
	}
}

// HTTPStatus returns the appropriate HTTP status code for this error code
func (c Code) HTTPStatus() int {
	switch c {
	case CodeNotFound, CodeTCOLObjectNotFound:
		return 404
	case CodeUnauthorized, CodeInvalidToken, CodeExpiredToken, CodeInvalidCredentials:
		return 401
	case CodeForbidden, CodeTCOLPermission:
		return 403
	case CodeInvalidInput, CodeValidationFailed, CodeRequiredField, CodeInvalidFormat, 
		 CodeValueOutOfRange, CodeInvalidLength, CodeTCOLSyntax, CodeTCOLSemantic:
		return 400
	case CodeDuplicateEntry, CodeResourceLocked, CodeInvalidOperation:
		return 409
	case CodeQuotaExceeded:
		return 429
	case CodeTimeout, CodeServiceTimeout:
		return 408
	case CodeServiceUnavailable, CodeDatabaseError, CodeConnectionFailed:
		return 503
	default:
		return 500
	}
}