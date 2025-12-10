// File: interfaces.go
// Title: Core Validation Interfaces and Types
// Description: Defines standard interfaces, types, and constants for unified
//              validation across all mDW Foundation modules. Provides the
//              foundation for consistent validation patterns and error handling.
// Author: msto63 with Claude Sonnet 4.0
// Version: v0.1.0
// Created: 2025-01-25
// Modified: 2025-01-25
//
// Change History:
// - 2025-01-25 v0.1.0: Initial validation interfaces implementation

package validation

import (
	"context"
	"fmt"
	"strings"

	mdwerror "github.com/msto63/mDW/foundation/core/error"
)

// Standard validation error codes used across all modules
const (
	// Core validation failures
	CodeRequired     = "VALIDATION_REQUIRED"     // Field is required but missing
	CodeFormat       = "VALIDATION_FORMAT"       // Invalid format (email, URL, etc.)
	CodeLength       = "VALIDATION_LENGTH"       // String/slice length validation
	CodeRange        = "VALIDATION_RANGE"        // Numeric range validation
	CodeType         = "VALIDATION_TYPE"         // Type validation (string, int, etc.)
	CodePattern      = "VALIDATION_PATTERN"      // Regex pattern validation  
	CodeCustom       = "VALIDATION_CUSTOM"       // Custom validation rules
	
	// Specific validation types
	CodeEmail        = "VALIDATION_EMAIL"        // Email address format
	CodeURL          = "VALIDATION_URL"          // URL format validation
	CodePhoneNumber  = "VALIDATION_PHONE"        // Phone number format
	CodePassword     = "VALIDATION_PASSWORD"     // Password strength
	CodeNumeric      = "VALIDATION_NUMERIC"      // Numeric value validation
	CodeDate         = "VALIDATION_DATE"         // Date format validation
	CodeTime         = "VALIDATION_TIME"         // Time format validation
	CodeJSON         = "VALIDATION_JSON"         // JSON format validation
	CodeXML          = "VALIDATION_XML"          // XML format validation
	
	// File and path validation
	CodePath         = "VALIDATION_PATH"         // File path validation
	CodeFileExists   = "VALIDATION_FILE_EXISTS"  // File existence check
	CodeFileType     = "VALIDATION_FILE_TYPE"    // File type validation
	CodePermission   = "VALIDATION_PERMISSION"   // File permission validation
	
	// Locale and internationalization
	CodeLocale       = "VALIDATION_LOCALE"       // Locale format validation
	CodeLanguage     = "VALIDATION_LANGUAGE"     // Language code validation
	CodeCountry      = "VALIDATION_COUNTRY"      // Country code validation
	CodeCurrency     = "VALIDATION_CURRENCY"     // Currency code validation
)

// Validator defines the interface for all validation functions
type Validator interface {
	// Validate performs validation on a value and returns structured result
	Validate(value interface{}) ValidationResult
	
	// ValidateWithContext performs validation with context for tracing/cancellation
	ValidateWithContext(ctx context.Context, value interface{}) ValidationResult
}

// ValidatorFunc is a function type that implements the Validator interface
type ValidatorFunc func(value interface{}) ValidationResult

// Validate implements the Validator interface for ValidatorFunc
func (f ValidatorFunc) Validate(value interface{}) ValidationResult {
	return f(value)
}

// ValidateWithContext implements context-aware validation for ValidatorFunc
func (f ValidatorFunc) ValidateWithContext(ctx context.Context, value interface{}) ValidationResult {
	// For function validators, context is passed through the result
	result := f(value)
	if ctx != nil {
		if result.Context == nil {
			result.Context = make(map[string]interface{})
		}
		// Add context information if available
		if requestID := ctx.Value("requestId"); requestID != nil {
			result.Context["requestId"] = requestID
		}
		if userID := ctx.Value("userId"); userID != nil {
			result.Context["userId"] = userID
		}
	}
	return result
}

// ValidationResult represents the result of a validation operation
type ValidationResult struct {
	Valid   bool                    `json:"valid"`   // Whether validation passed
	Errors  []ValidationError      `json:"errors,omitempty"`  // Detailed error information  
	Context map[string]interface{} `json:"context,omitempty"` // Additional context data
}

// ValidationError represents a single validation error with rich context
type ValidationError struct {
	Code     string                 `json:"code"`              // Standardized error code
	Field    string                 `json:"field,omitempty"`   // Field name being validated
	Message  string                 `json:"message"`           // Human-readable error message
	Value    interface{}           `json:"value,omitempty"`   // Actual value that failed validation
	Context  map[string]interface{} `json:"context,omitempty"` // Additional error context
	Expected interface{}           `json:"expected,omitempty"` // Expected value or format
}

// NewValidationResult creates a successful validation result
func NewValidationResult() ValidationResult {
	return ValidationResult{
		Valid:   true,
		Errors:  nil,
		Context: nil,
	}
}

// NewValidationError creates a failed validation result with a single error
func NewValidationError(code, message string) ValidationResult {
	return ValidationResult{
		Valid: false,
		Errors: []ValidationError{
			{
				Code:    code,
				Message: message,
			},
		},
	}
}

// NewValidationErrorWithField creates a validation error for a specific field
func NewValidationErrorWithField(code, field, message string, value interface{}) ValidationResult {
	return ValidationResult{
		Valid: false,
		Errors: []ValidationError{
			{
				Code:    code,
				Field:   field,
				Message: message,
				Value:   value,
			},
		},
	}
}

// AddError adds an error to an existing validation result
func (r *ValidationResult) AddError(code, message string) *ValidationResult {
	r.Valid = false
	r.Errors = append(r.Errors, ValidationError{
		Code:    code,
		Message: message,
	})
	return r
}

// AddFieldError adds a field-specific error to the validation result
func (r *ValidationResult) AddFieldError(code, field, message string, value interface{}) *ValidationResult {
	r.Valid = false
	r.Errors = append(r.Errors, ValidationError{
		Code:    code,
		Field:   field,
		Message: message,
		Value:   value,
	})
	return r
}

// WithContext adds context information to the validation result
func (r *ValidationResult) WithContext(key string, value interface{}) *ValidationResult {
	if r.Context == nil {
		r.Context = make(map[string]interface{})
	}
	r.Context[key] = value
	return r
}

// FirstError returns the first validation error, or nil if validation passed
func (r ValidationResult) FirstError() *ValidationError {
	if len(r.Errors) == 0 {
		return nil
	}
	return &r.Errors[0]
}

// ErrorMessages returns all error messages as a slice of strings
func (r ValidationResult) ErrorMessages() []string {
	messages := make([]string, len(r.Errors))
	for i, err := range r.Errors {
		messages[i] = err.Message
	}
	return messages
}

// ErrorCodes returns all error codes as a slice of strings
func (r ValidationResult) ErrorCodes() []string {
	codes := make([]string, len(r.Errors))
	for i, err := range r.Errors {
		codes[i] = err.Code
	}
	return codes
}

// HasError checks if the result contains a specific error code
func (r ValidationResult) HasError(code string) bool {
	for _, err := range r.Errors {
		if err.Code == code {
			return true
		}
	}
	return false
}

// ToError converts the validation result to a standard error
// Returns nil if validation passed, or an error with detailed information
func (r ValidationResult) ToError() error {
	if r.Valid {
		return nil
	}
	
	if len(r.Errors) == 0 {
		return mdwerror.New("validation failed").
			WithCode(mdwerror.CodeValidationFailed)
	}
	
	firstError := r.Errors[0]
	err := mdwerror.New(firstError.Message).
		WithCode(mdwerror.Code(firstError.Code))
	
	// Add field information if available
	if firstError.Field != "" {
		err = err.WithDetail("field", firstError.Field)
	}
	
	// Add actual value if available
	if firstError.Value != nil {
		err = err.WithDetail("value", firstError.Value)
	}
	
	// Add expected value if available
	if firstError.Expected != nil {
		err = err.WithDetail("expected", firstError.Expected)
	}
	
	// Add context information
	for key, value := range firstError.Context {
		err = err.WithDetail(key, value)
	}
	
	// If multiple errors, add them as details
	if len(r.Errors) > 1 {
		err = err.WithDetail("totalErrors", len(r.Errors))
		err = err.WithDetail("allMessages", r.ErrorMessages())
	}
	
	return err
}

// String returns a human-readable representation of the validation result
func (r ValidationResult) String() string {
	if r.Valid {
		return "ValidationResult{valid: true}"
	}
	
	var parts []string
	parts = append(parts, "ValidationResult{valid: false")
	
	if len(r.Errors) > 0 {
		parts = append(parts, fmt.Sprintf("errors: %d", len(r.Errors)))
		
		// Add first error details
		firstError := r.Errors[0]
		parts = append(parts, fmt.Sprintf("first: %s", firstError.Message))
		if firstError.Field != "" {
			parts = append(parts, fmt.Sprintf("field: %s", firstError.Field))
		}
	}
	
	parts = append(parts, "}")
	return strings.Join(parts, ", ")
}

// String returns a human-readable representation of a validation error
func (e ValidationError) String() string {
	var parts []string
	
	if e.Field != "" {
		parts = append(parts, fmt.Sprintf("field:%s", e.Field))
	}
	
	parts = append(parts, fmt.Sprintf("code:%s", e.Code))
	parts = append(parts, fmt.Sprintf("message:%s", e.Message))
	
	if e.Value != nil {
		parts = append(parts, fmt.Sprintf("value:%v", e.Value))
	}
	
	if e.Expected != nil {
		parts = append(parts, fmt.Sprintf("expected:%v", e.Expected))
	}
	
	return fmt.Sprintf("ValidationError{%s}", strings.Join(parts, ", "))
}

// Combine merges multiple validation results into a single result
func Combine(results ...ValidationResult) ValidationResult {
	combined := NewValidationResult()
	
	for _, result := range results {
		if !result.Valid {
			combined.Valid = false
			combined.Errors = append(combined.Errors, result.Errors...)
		}
		
		// Merge context information
		for key, value := range result.Context {
			combined.WithContext(key, value)
		}
	}
	
	return combined
}