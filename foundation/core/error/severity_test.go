// File: severity_test.go
// Title: Severity Tests
// Description: Tests for error severity functionality including string representation,
//              alerting rules, and automatic severity determination from error codes.
// Author: msto63 with Claude Sonnet 4.0
// Version: v0.1.0
// Created: 2025-01-24
// Modified: 2025-01-24
//
// Change History:
// - 2025-01-24 v0.1.0: Initial implementation with comprehensive severity tests

package error

import (
	"testing"
)

func TestSeverityString(t *testing.T) {
	tests := []struct {
		severity Severity
		want     string
	}{
		{SeverityLow, "low"},
		{SeverityMedium, "medium"},
		{SeverityHigh, "high"},
		{SeverityCritical, "critical"},
		{Severity(999), "unknown"}, // Invalid severity
	}
	
	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := tt.severity.String(); got != tt.want {
				t.Errorf("Severity.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSeverityLevel(t *testing.T) {
	tests := []struct {
		severity Severity
		want     int
	}{
		{SeverityLow, 0},
		{SeverityMedium, 1},
		{SeverityHigh, 2},
		{SeverityCritical, 3},
	}
	
	for _, tt := range tests {
		t.Run(tt.severity.String(), func(t *testing.T) {
			if got := tt.severity.Level(); got != tt.want {
				t.Errorf("Severity.Level() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSeverityShouldAlert(t *testing.T) {
	tests := []struct {
		severity    Severity
		shouldAlert bool
	}{
		{SeverityLow, false},
		{SeverityMedium, false},
		{SeverityHigh, true},
		{SeverityCritical, true},
	}
	
	for _, tt := range tests {
		t.Run(tt.severity.String(), func(t *testing.T) {
			if got := tt.severity.ShouldAlert(); got != tt.shouldAlert {
				t.Errorf("Severity.ShouldAlert() = %v, want %v", got, tt.shouldAlert)
			}
		})
	}
}

func TestSeverityShouldLog(t *testing.T) {
	// All severities should be logged
	severities := []Severity{SeverityLow, SeverityMedium, SeverityHigh, SeverityCritical}
	
	for _, severity := range severities {
		t.Run(severity.String(), func(t *testing.T) {
			if !severity.ShouldLog() {
				t.Errorf("Severity.ShouldLog() = false, want true for %v", severity)
			}
		})
	}
}

func TestSeverityPriority(t *testing.T) {
	tests := []struct {
		severity Severity
		want     int
	}{
		{SeverityLow, 0},
		{SeverityMedium, 1},
		{SeverityHigh, 2},
		{SeverityCritical, 3},
	}
	
	for _, tt := range tests {
		t.Run(tt.severity.String(), func(t *testing.T) {
			if got := tt.severity.Priority(); got != tt.want {
				t.Errorf("Severity.Priority() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSeverityOrdering(t *testing.T) {
	// Test that severities are properly ordered
	if SeverityLow >= SeverityMedium {
		t.Error("SeverityLow should be less than SeverityMedium")
	}
	
	if SeverityMedium >= SeverityHigh {
		t.Error("SeverityMedium should be less than SeverityHigh")
	}
	
	if SeverityHigh >= SeverityCritical {
		t.Error("SeverityHigh should be less than SeverityCritical")
	}
}

func TestGetSeverityFromCode(t *testing.T) {
	tests := []struct {
		name     string
		code     Code
		severity Severity
	}{
		// Critical severity
		{"data corruption", CodeDataCorruption, SeverityCritical},
		{"service unavailable", CodeServiceUnavailable, SeverityCritical},
		{"environment error", CodeEnvironmentError, SeverityCritical},
		
		// High severity
		{"database error", CodeDatabaseError, SeverityHigh},
		{"connection failed", CodeConnectionFailed, SeverityHigh},
		{"service initialization", CodeServiceInitialization, SeverityHigh},
		{"unauthorized", CodeUnauthorized, SeverityHigh},
		{"forbidden", CodeForbidden, SeverityHigh},
		{"invalid token", CodeInvalidToken, SeverityHigh},
		{"expired token", CodeExpiredToken, SeverityHigh},
		
		// Medium severity
		{"business rule", CodeBusinessRule, SeverityMedium},
		{"insufficient funds", CodeInsufficientFunds, SeverityMedium},
		{"resource locked", CodeResourceLocked, SeverityMedium},
		{"quota exceeded", CodeQuotaExceeded, SeverityMedium},
		{"service timeout", CodeServiceTimeout, SeverityMedium},
		{"network error", CodeNetworkError, SeverityMedium},
		{"external service error", CodeExternalServiceError, SeverityMedium},
		{"TCOL permission", CodeTCOLPermission, SeverityMedium},
		{"TCOL execution", CodeTCOLExecution, SeverityMedium},
		
		// Low severity
		{"invalid input", CodeInvalidInput, SeverityLow},
		{"not found", CodeNotFound, SeverityLow},
		{"validation failed", CodeValidationFailed, SeverityLow},
		{"required field", CodeRequiredField, SeverityLow},
		{"invalid format", CodeInvalidFormat, SeverityLow},
		{"value out of range", CodeValueOutOfRange, SeverityLow},
		{"invalid length", CodeInvalidLength, SeverityLow},
		{"TCOL syntax", CodeTCOLSyntax, SeverityLow},
		{"TCOL semantic", CodeTCOLSemantic, SeverityLow},
		{"TCOL object not found", CodeTCOLObjectNotFound, SeverityLow},
		
		// Default case
		{"unknown code", Code("UNKNOWN_CODE"), SeverityMedium},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetSeverityFromCode(tt.code); got != tt.severity {
				t.Errorf("GetSeverityFromCode(%v) = %v, want %v", tt.code, got, tt.severity)
			}
		})
	}
}

func TestSeverityConsistency(t *testing.T) {
	// Test that severity mappings are consistent across different functions
	codes := []Code{
		CodeDatabaseError,
		CodeNotFound,
		CodeUnauthorized,
		CodeDataCorruption,
		CodeValidationFailed,
	}
	
	for _, code := range codes {
		t.Run(string(code), func(t *testing.T) {
			severity := GetSeverityFromCode(code)
			
			// Test that severity methods return expected values
			if severity.Level() < 0 || severity.Level() > 3 {
				t.Errorf("Severity level %d is out of valid range [0-3]", severity.Level())
			}
			
			if severity.Priority() != severity.Level() {
				t.Errorf("Priority() and Level() should return the same value, got %d and %d", 
					severity.Priority(), severity.Level())
			}
			
			// Test string representation
			str := severity.String()
			if str == "" || str == "unknown" {
				t.Errorf("Severity string should not be empty or unknown for valid severity, got %q", str)
			}
		})
	}
}

func BenchmarkGetSeverityFromCode(b *testing.B) {
	codes := []Code{
		CodeDatabaseError,
		CodeNotFound,
		CodeUnauthorized,
		CodeValidationFailed,
		CodeTCOLSyntax,
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		code := codes[i%len(codes)]
		_ = GetSeverityFromCode(code)
	}
}

func BenchmarkSeverityString(b *testing.B) {
	severities := []Severity{SeverityLow, SeverityMedium, SeverityHigh, SeverityCritical}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		severity := severities[i%len(severities)]
		_ = severity.String()
	}
}