// File: severity.go
// Title: Error Severity Levels
// Description: Defines severity levels for errors to enable proper prioritization,
//              monitoring, and alerting. Severity levels help operations teams
//              respond appropriately to different types of errors.
// Author: msto63 with Claude Sonnet 4.0
// Version: v0.1.0
// Created: 2025-01-24
// Modified: 2025-01-24
//
// Change History:
// - 2025-01-24 v0.1.0: Initial implementation with severity levels

package error

// Severity represents the severity level of an error
type Severity int

const (
	// SeverityLow indicates a minor error that doesn't affect core functionality
	// Examples: invalid user input, missing optional fields, cosmetic issues
	SeverityLow Severity = iota
	
	// SeverityMedium indicates an error that affects functionality but has workarounds
	// Examples: degraded performance, non-critical service unavailable, data inconsistency
	SeverityMedium
	
	// SeverityHigh indicates a serious error that significantly impacts functionality
	// Examples: database connection issues, critical service failures, security violations
	SeverityHigh
	
	// SeverityCritical indicates a critical error that makes the system unusable
	// Examples: data corruption, complete service outage, security breaches
	SeverityCritical
)

// String returns the string representation of the severity level
func (s Severity) String() string {
	switch s {
	case SeverityLow:
		return "low"
	case SeverityMedium:
		return "medium"
	case SeverityHigh:
		return "high"
	case SeverityCritical:
		return "critical"
	default:
		return "unknown"
	}
}

// Level returns the numeric level of the severity (0-3)
func (s Severity) Level() int {
	return int(s)
}

// ShouldAlert returns true if this severity level should trigger alerts
func (s Severity) ShouldAlert() bool {
	return s >= SeverityHigh
}

// ShouldLog returns true if this severity level should be logged
func (s Severity) ShouldLog() bool {
	return true // All severities should be logged
}

// Priority returns a priority value for sorting (higher number = higher priority)
func (s Severity) Priority() int {
	return int(s)
}

// GetSeverityFromCode determines appropriate severity level based on error code
func GetSeverityFromCode(code Code) Severity {
	switch code {
	// Critical system errors
	case CodeDataCorruption, CodeServiceUnavailable, CodeEnvironmentError:
		return SeverityCritical
		
	// High severity errors
	case CodeDatabaseError, CodeConnectionFailed, CodeServiceInitialization,
		 CodeUnauthorized, CodeForbidden, CodeInvalidToken, CodeExpiredToken:
		return SeverityHigh
		
	// Medium severity errors
	case CodeBusinessRule, CodeInsufficientFunds, CodeResourceLocked, CodeQuotaExceeded,
		 CodeServiceTimeout, CodeNetworkError, CodeExternalServiceError,
		 CodeTCOLPermission, CodeTCOLExecution:
		return SeverityMedium
		
	// Low severity errors
	case CodeInvalidInput, CodeNotFound, CodeValidationFailed, CodeRequiredField,
		 CodeInvalidFormat, CodeValueOutOfRange, CodeInvalidLength,
		 CodeTCOLSyntax, CodeTCOLSemantic, CodeTCOLObjectNotFound:
		return SeverityLow
		
	default:
		return SeverityMedium
	}
}