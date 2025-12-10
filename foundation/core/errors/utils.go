// File: utils.go
// Title: Shared Error Handling Utilities
// Description: Provides common error handling utilities that can be used
//              across all mDW foundation modules for consistent error patterns.
// Author: msto63 with Claude Sonnet 4.0
// Version: v0.1.1
// Created: 2025-01-25
// Modified: 2025-07-26
//
// Change History:
// - 2025-01-25 v0.1.0: Initial implementation of shared error utilities
// - 2025-07-26 v0.1.1: Enhanced OutOfRange function with "validation failed:" prefix
//                       for better integration test compatibility

package errors

import (
	"fmt"
	"reflect"
	"strings"

	mdwerror "github.com/msto63/mDW/foundation/core/error"
)

// ErrorBuilder provides a fluent interface for building standardized errors
type ErrorBuilder struct {
	module    string
	operation string
	message   string
	cause     error
	details   map[string]interface{}
	severity  mdwerror.Severity
	code      string
}

// NewErrorBuilder creates a new error builder for the specified module
func NewErrorBuilder(module string) *ErrorBuilder {
	return &ErrorBuilder{
		module:   module,
		details:  make(map[string]interface{}),
		severity: mdwerror.SeverityMedium,
	}
}

// Operation sets the operation name for the error
func (eb *ErrorBuilder) Operation(operation string) *ErrorBuilder {
	eb.operation = operation
	return eb
}

// Message sets the error message
func (eb *ErrorBuilder) Message(message string) *ErrorBuilder {
	eb.message = message
	return eb
}

// Messagef sets the error message with formatting
func (eb *ErrorBuilder) Messagef(format string, args ...interface{}) *ErrorBuilder {
	eb.message = fmt.Sprintf(format, args...)
	return eb
}

// Cause sets the underlying cause of the error
func (eb *ErrorBuilder) Cause(cause error) *ErrorBuilder {
	eb.cause = cause
	return eb
}

// Detail adds a detail key-value pair to the error
func (eb *ErrorBuilder) Detail(key string, value interface{}) *ErrorBuilder {
	eb.details[key] = value
	return eb
}

// Details sets multiple details at once
func (eb *ErrorBuilder) Details(details map[string]interface{}) *ErrorBuilder {
	for k, v := range details {
		eb.details[k] = v
	}
	return eb
}

// Severity sets the error severity
func (eb *ErrorBuilder) Severity(severity mdwerror.Severity) *ErrorBuilder {
	eb.severity = severity
	return eb
}

// Code sets the error code
func (eb *ErrorBuilder) Code(code string) *ErrorBuilder {
	eb.code = code
	return eb
}

// Build creates the final error
func (eb *ErrorBuilder) Build() *mdwerror.Error {
	// Auto-generate code if not set
	if eb.code == "" {
		eb.code = getModuleErrorCode(eb.module, eb.operation)
	}
	
	// Auto-generate message if not set
	if eb.message == "" {
		if eb.operation != "" {
			eb.message = fmt.Sprintf("%s.%s failed", eb.module, eb.operation)
		} else {
			eb.message = fmt.Sprintf("%s operation failed", eb.module)
		}
	}
	
	// Add module and operation to details
	eb.details["module"] = eb.module
	if eb.operation != "" {
		eb.details["operation"] = eb.operation
	}
	
	// Create the error
	var err *mdwerror.Error
	if eb.cause != nil {
		err = mdwerror.Wrap(eb.cause, eb.message)
	} else {
		err = mdwerror.New(eb.message)
	}
	
	return err.
		WithCode(mdwerror.Code(eb.code)).
		WithDetails(eb.details).
		WithSeverity(eb.severity)
}

// =============================================================================
// STANDARD ERROR CREATION FUNCTIONS FOR ALL mDW MODULES
// =============================================================================
// These functions provide a consistent interface for creating errors across
// all mDW foundation modules. Use these instead of fmt.Errorf() or errors.New()

// InvalidInput creates a standardized invalid input error
func InvalidInput(module, operation string, input interface{}, expected string) *mdwerror.Error {
	return NewErrorBuilder(module).
		Operation(operation).
		Message(fmt.Sprintf("invalid input for %s.%s", module, operation)).
		Code(CodeInvalidInput).
		Detail("input", input).
		Detail("expected", expected).
		Severity(mdwerror.SeverityMedium).
		Build()
}

// InvalidFormat creates a standardized format error
func InvalidFormat(module string, input interface{}, expectedFormat string) *mdwerror.Error {
	return NewErrorBuilder(module).
		Message(fmt.Sprintf("invalid format in %s", module)).
		Code(getFormatErrorCode(module)).
		Detail("input", input).
		Detail("expected_format", expectedFormat).
		Severity(mdwerror.SeverityMedium).
		Build()
}

// OperationFailed creates a standardized operation failure error
func OperationFailed(module, operation string, cause error) *mdwerror.Error {
	return NewErrorBuilder(module).
		Operation(operation).
		Message(fmt.Sprintf("%s.%s operation failed", module, operation)).
		Cause(cause).
		Code(getOperationErrorCode(module)).
		Severity(mdwerror.SeverityHigh).
		Build()
}

// ValidationFailed creates a standardized validation error
func ValidationFailed(module, field string, value interface{}, reason string) *mdwerror.Error {
	return NewErrorBuilder(module).
		Message(fmt.Sprintf("%s.validate_%s: validation failed for field %s: %s", module, field, field, reason)).
		Code(fmt.Sprintf("%s_VALIDATION_FAILED", strings.ToUpper(module))).
		Detail("field", field).
		Detail("value", value).
		Detail("reason", reason).
		Severity(mdwerror.SeverityLow).
		Build()
}

// OutOfRange creates a standardized out of range error
func OutOfRange(module, operation string, value, min, max interface{}) *mdwerror.Error {
	return NewErrorBuilder(module).
		Operation(operation).
		Message(fmt.Sprintf("validation failed: value out of range in %s.%s", module, operation)).
		Code(CodeOutOfRange).
		Detail("value", value).
		Detail("min", min).
		Detail("max", max).
		Severity(mdwerror.SeverityMedium).
		Build()
}

// NotFound creates a standardized not found error
func NotFound(module, operation string, identifier interface{}) *mdwerror.Error {
	return NewErrorBuilder(module).
		Operation(operation).
		Message(fmt.Sprintf("item not found in %s.%s", module, operation)).
		Code(CodeNotFound).
		Detail("identifier", identifier).
		Severity(mdwerror.SeverityMedium).
		Build()
}

// Utility functions for error analysis

// ExtractDetails extracts all details from a mDW error
func ExtractDetails(err error) map[string]interface{} {
	if mdwErr, ok := err.(*mdwerror.Error); ok {
		return mdwErr.Details()
	}
	return nil
}

// ExtractModule extracts the module name from an error
func ExtractModule(err error) string {
	details := ExtractDetails(err)
	if details != nil {
		if module, ok := details["module"].(string); ok {
			return module
		}
	}
	return ""
}

// ExtractOperation extracts the operation name from an error
func ExtractOperation(err error) string {
	details := ExtractDetails(err)
	if details != nil {
		if operation, ok := details["operation"].(string); ok {
			return operation
		}
	}
	return ""
}

// IsModuleOperation checks if error is from specific module and operation
func IsModuleOperation(err error, module, operation string) bool {
	return ExtractModule(err) == module && ExtractOperation(err) == operation
}

// ValidateRequired validates that a value is not nil/empty using reflection
func ValidateRequired(module, field string, value interface{}) error {
	if value == nil {
		return ValidationFailed(module, field, value, "cannot be nil")
	}
	
	v := reflect.ValueOf(value)
	switch v.Kind() {
	case reflect.String:
		if v.String() == "" {
			return ValidationFailed(module, field, value, "cannot be empty")
		}
	case reflect.Slice, reflect.Map, reflect.Array:
		if v.Len() == 0 {
			return ValidationFailed(module, field, value, "cannot be empty")
		}
	case reflect.Ptr, reflect.Interface:
		if v.IsNil() {
			return ValidationFailed(module, field, value, "cannot be nil")
		}
	}
	
	return nil
}

// ValidateRange validates that a numeric value is within range
func ValidateRange(module, field string, value, min, max interface{}) error {
	// Convert to float64 for comparison
	val, err := toFloat64(value)
	if err != nil {
		return InvalidInput(module, "validate_range", value, "numeric value")
	}
	
	minVal, err := toFloat64(min)
	if err != nil {
		return InvalidInput(module, "validate_range", min, "numeric min value")
	}
	
	maxVal, err := toFloat64(max)
	if err != nil {
		return InvalidInput(module, "validate_range", max, "numeric max value")
	}
	
	if val < minVal || val > maxVal {
		return OutOfRange(module, "validate_range", value, min, max)
	}
	
	return nil
}

// toFloat64 converts various numeric types to float64
func toFloat64(value interface{}) (float64, error) {
	switch v := value.(type) {
	case int:
		return float64(v), nil
	case int32:
		return float64(v), nil
	case int64:
		return float64(v), nil
	case float32:
		return float64(v), nil
	case float64:
		return v, nil
	default:
		return 0, fmt.Errorf("cannot convert %T to float64", value)
	}
}

// =============================================================================
// MODULE-SPECIFIC CONVENIENCE FUNCTIONS
// =============================================================================
// These functions provide direct, easy-to-use error creation for common
// scenarios in each foundation module

// StringX convenience functions
func StringxValidationError(operation, input, expected string) *mdwerror.Error {
	return ValidationFailed("stringx", operation, input, expected)
}

func StringxInvalidInput(operation string, input interface{}) *mdwerror.Error {
	return InvalidInput("stringx", operation, input, "valid string")
}

func StringxFormatError(input, expectedFormat string) *mdwerror.Error {
	return InvalidFormat("stringx", input, expectedFormat)
}

// MathX convenience functions  
func MathxDivisionByZero(operation string) *mdwerror.Error {
	return NewErrorBuilder("mathx").
		Operation(operation).
		Message("division by zero").
		Code("MATHX_DIVISION_BY_ZERO").
		Severity(mdwerror.SeverityHigh).
		Build()
}

func MathxPrecisionLoss(operation string, input interface{}) *mdwerror.Error {
	return NewErrorBuilder("mathx").
		Operation(operation).
		Message("precision loss in calculation").
		Code("MATHX_PRECISION_LOSS").
		Detail("input", input).
		Severity(mdwerror.SeverityMedium).
		Build()
}

func MathxInvalidDecimal(input string) *mdwerror.Error {
	return InvalidFormat("mathx", input, "valid decimal string")
}

// SliceX convenience functions
func SlicexIndexOutOfRange(operation string, index, length int) *mdwerror.Error {
	return OutOfRange("slicex", operation, index, 0, length-1)
}

func SlicexEmptySlice(operation string) *mdwerror.Error {
	return InvalidInput("slicex", operation, "empty slice", "non-empty slice")
}

// MapX convenience functions
func MapxKeyNotFound(operation, key string) *mdwerror.Error {
	return NewErrorBuilder("mapx").
		Operation(operation).
		Message(fmt.Sprintf("key '%s' not found", key)).
		Code("MAPX_KEY_NOT_FOUND").
		Detail("key", key).
		Severity(mdwerror.SeverityLow).
		Build()
}

func MapxEmptyMap(operation string) *mdwerror.Error {
	return InvalidInput("mapx", operation, "empty map", "non-empty map")
}

// TimeX convenience functions
func TimexParseError(input, expectedFormat string) *mdwerror.Error {
	return InvalidFormat("timex", input, expectedFormat)
}

func TimexInvalidTimezone(timezone string) *mdwerror.Error {
	return InvalidInput("timex", "set_timezone", timezone, "valid timezone")
}

// ValidationX convenience functions
func ValidationxRuleFailed(rule, field string, value interface{}, message string) *mdwerror.Error {
	return ValidationFailed("validationx", field, value, fmt.Sprintf("rule '%s': %s", rule, message))
}

// FileX convenience functions
func FilexNotFound(path string) *mdwerror.Error {
	return NewErrorBuilder("filex").
		Operation("access").
		Message(fmt.Sprintf("file not found: %s", path)).
		Code("FILEX_NOT_FOUND").
		Detail("path", path).
		Severity(mdwerror.SeverityMedium).
		Build()
}

func FilexPermissionDenied(path, operation string) *mdwerror.Error {
	return NewErrorBuilder("filex").
		Operation(operation).
		Message(fmt.Sprintf("permission denied: %s", path)).
		Code("FILEX_PERMISSION_DENIED").
		Detail("path", path).
		Detail("operation", operation).
		Severity(mdwerror.SeverityHigh).
		Build()
}