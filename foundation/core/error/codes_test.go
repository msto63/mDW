// File: codes_test.go
// Title: Error Code Tests
// Description: Tests for error code functionality including validation,
//              categorization, and HTTP status mapping.
// Author: msto63 with Claude Sonnet 4.0
// Version: v0.1.0
// Created: 2025-01-24
// Modified: 2025-01-24
//
// Change History:
// - 2025-01-24 v0.1.0: Initial implementation with comprehensive code tests

package error

import (
	"testing"
)

func TestCodeString(t *testing.T) {
	tests := []struct {
		code Code
		want string
	}{
		{CodeUnknown, "UNKNOWN"},
		{CodeDatabaseError, "DATABASE_ERROR"},
		{CodeNotFound, "NOT_FOUND"},
		{CodeTCOLSyntax, "TCOL_SYNTAX"},
	}
	
	for _, tt := range tests {
		t.Run(string(tt.code), func(t *testing.T) {
			if got := tt.code.String(); got != tt.want {
				t.Errorf("Code.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCodeIsValid(t *testing.T) {
	tests := []struct {
		name string
		code Code
		want bool
	}{
		{"known code", CodeDatabaseError, true},
		{"unknown code", Code("INVALID_CODE"), false},
		{"empty code", Code(""), false},
		{"TCOL code", CodeTCOLSyntax, true},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.code.IsValid(); got != tt.want {
				t.Errorf("Code.IsValid() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCodeCategory(t *testing.T) {
	tests := []struct {
		code     Code
		category string
	}{
		{CodeUnauthorized, "authentication"},
		{CodeForbidden, "authentication"},
		{CodeInvalidToken, "authentication"},
		{CodeDatabaseError, "database"},
		{CodeConnectionFailed, "database"},
		{CodeBusinessRule, "business"},
		{CodeInsufficientFunds, "business"},
		{CodeServiceUnavailable, "service"},
		{CodeNetworkError, "service"},
		{CodeTCOLSyntax, "tcol"},
		{CodeTCOLPermission, "tcol"},
		{CodeConfigError, "configuration"},
		{CodeMissingConfig, "configuration"},
		{CodeValidationFailed, "validation"},
		{CodeRequiredField, "validation"},
		{CodeUnknown, "generic"},
		{CodeInternal, "generic"},
	}
	
	for _, tt := range tests {
		t.Run(string(tt.code), func(t *testing.T) {
			if got := tt.code.Category(); got != tt.category {
				t.Errorf("Code.Category() = %v, want %v", got, tt.category)
			}
		})
	}
}

func TestCodeHTTPStatus(t *testing.T) {
	tests := []struct {
		code       Code
		httpStatus int
	}{
		// 400 Bad Request
		{CodeInvalidInput, 400},
		{CodeValidationFailed, 400},
		{CodeTCOLSyntax, 400},
		{CodeTCOLSemantic, 400},
		
		// 401 Unauthorized
		{CodeUnauthorized, 401},
		{CodeInvalidCredentials, 401},
		{CodeInvalidToken, 401},
		
		// 403 Forbidden
		{CodeForbidden, 403},
		{CodeTCOLPermission, 403},
		
		// 404 Not Found
		{CodeNotFound, 404},
		{CodeTCOLObjectNotFound, 404},
		
		// 408 Timeout
		{CodeTimeout, 408},
		{CodeServiceTimeout, 408},
		
		// 409 Conflict
		{CodeDuplicateEntry, 409},
		{CodeResourceLocked, 409},
		{CodeInvalidOperation, 409},
		
		// 429 Too Many Requests
		{CodeQuotaExceeded, 429},
		
		// 500 Internal Server Error
		{CodeUnknown, 500},
		{CodeInternal, 500},
		
		// 503 Service Unavailable
		{CodeServiceUnavailable, 503},
		{CodeDatabaseError, 503},
		{CodeConnectionFailed, 503},
	}
	
	for _, tt := range tests {
		t.Run(string(tt.code), func(t *testing.T) {
			if got := tt.code.HTTPStatus(); got != tt.httpStatus {
				t.Errorf("Code.HTTPStatus() = %v, want %v", got, tt.httpStatus)
			}
		})
	}
}

func TestAllDefinedCodesAreValid(t *testing.T) {
	// Test that all defined codes are considered valid
	codes := []Code{
		// Generic codes
		CodeUnknown, CodeInternal, CodeNotFound, CodeInvalidInput, CodeTimeout,
		
		// Authentication and authorization
		CodeUnauthorized, CodeForbidden, CodeInvalidToken, CodeExpiredToken, CodeInvalidCredentials,
		
		// Database and storage
		CodeDatabaseError, CodeConnectionFailed, CodeDataCorruption, CodeConstraintViolation, CodeDuplicateEntry,
		
		// Business logic
		CodeBusinessRule, CodeInsufficientFunds, CodeInvalidOperation, CodeResourceLocked, CodeQuotaExceeded,
		
		// Service and network
		CodeServiceUnavailable, CodeNetworkError, CodeServiceTimeout, CodeServiceInitialization, CodeExternalServiceError,
		
		// TCOL specific
		CodeTCOLSyntax, CodeTCOLSemantic, CodeTCOLPermission, CodeTCOLExecution, CodeTCOLObjectNotFound,
		
		// Configuration and environment
		CodeConfigError, CodeMissingConfig, CodeInvalidConfig, CodeEnvironmentError,
		
		// Validation
		CodeValidationFailed, CodeRequiredField, CodeInvalidFormat, CodeValueOutOfRange, CodeInvalidLength,
	}
	
	for _, code := range codes {
		t.Run(string(code), func(t *testing.T) {
			if !code.IsValid() {
				t.Errorf("Code %v should be valid", code)
			}
		})
	}
}

func TestCodeCategoryCoverage(t *testing.T) {
	// Ensure all categories are covered
	expectedCategories := map[string]bool{
		"authentication": false,
		"database":       false,
		"business":       false,
		"service":        false,
		"tcol":          false,
		"configuration": false,  
		"validation":    false,
		"generic":       false,
	}
	
	// Test a representative sample from each category
	testCodes := []Code{
		CodeUnauthorized,    // authentication
		CodeDatabaseError,   // database
		CodeBusinessRule,    // business
		CodeServiceUnavailable, // service
		CodeTCOLSyntax,     // tcol
		CodeConfigError,    // configuration
		CodeValidationFailed, // validation
		CodeUnknown,        // generic
	}
	
	for _, code := range testCodes {
		category := code.Category()
		if _, exists := expectedCategories[category]; !exists {
			t.Errorf("Unexpected category %q for code %v", category, code)
		} else {
			expectedCategories[category] = true
		}
	}
	
	// Ensure all categories were covered
	for category, covered := range expectedCategories {
		if !covered {
			t.Errorf("Category %q was not covered by test codes", category)
		}
	}
}

func TestHTTPStatusRanges(t *testing.T) {
	// Test that HTTP status codes are within expected ranges
	tests := []struct {
		name       string
		code       Code
		minStatus  int
		maxStatus  int
	}{
		{"client error codes", CodeInvalidInput, 400, 499},
		{"server error codes", CodeInternal, 500, 599},
		{"auth codes", CodeUnauthorized, 401, 403},
		{"not found codes", CodeNotFound, 404, 404},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status := tt.code.HTTPStatus()
			if status < tt.minStatus || status > tt.maxStatus {
				t.Errorf("HTTP status %d for code %v is outside expected range [%d, %d]", 
					status, tt.code, tt.minStatus, tt.maxStatus)
			}
		})
	}
}