// File: standards_fixed.go
// Title: Fixed Error Standards for mDW Foundation
// Description: Provides standardized error handling patterns and codes for all
//              mDW foundation modules to ensure consistency and integration.
// Author: msto63 with Claude Sonnet 4.0
// Version: v0.1.1
// Created: 2025-01-25
// Modified: 2025-01-25
//
// Change History:
// - 2025-01-25 v0.1.0: Initial implementation for error standardization
// - 2025-01-25 v0.1.1: Fixed import and type reference issues

package errors

import (
	"fmt"
	"strings"

	mdwerror "github.com/msto63/mDW/foundation/core/error"
)

// Module identifiers for error categorization
const (
	ModuleStringx     = "stringx"
	ModuleMathx       = "mathx"
	ModuleMapx        = "mapx"
	ModuleSlicex      = "slicex"
	ModuleTimex       = "timex"
	ModuleValidationx = "validationx"
	ModuleFilex       = "filex"
)

// Standardized error codes for all modules
const (
	// Common error codes
	CodeInvalidInput     = "INVALID_INPUT"
	CodeInvalidFormat    = "INVALID_FORMAT"
	CodeOutOfRange       = "OUT_OF_RANGE"
	CodeNotFound         = "NOT_FOUND"
	CodePermissionDenied = "PERMISSION_DENIED"
	CodeOperationFailed  = "OPERATION_FAILED"

	// Module-specific error codes - stringx
	CodeStringxInvalidFormat   = "STRINGX_INVALID_FORMAT"
	CodeStringxLengthExceeded  = "STRINGX_LENGTH_EXCEEDED"
	CodeStringxEncodingError   = "STRINGX_ENCODING_ERROR"
	CodeStringxInvalidPattern  = "STRINGX_INVALID_PATTERN"

	// Module-specific error codes - mathx
	CodeMathxPrecisionLoss     = "MATHX_PRECISION_LOSS"
	CodeMathxDivisionByZero    = "MATHX_DIVISION_BY_ZERO"
	CodeMathxOverflow          = "MATHX_OVERFLOW"
	CodeMathxUnderflow         = "MATHX_UNDERFLOW"
	CodeMathxInvalidDecimal    = "MATHX_INVALID_DECIMAL"
	CodeMathxOperationFailed   = "MATHX_OPERATION_FAILED"

	// Module-specific error codes - mapx
	CodeMapxKeyNotFound        = "MAPX_KEY_NOT_FOUND"
	CodeMapxInvalidType        = "MAPX_INVALID_TYPE"
	CodeMapxOperationFailed    = "MAPX_OPERATION_FAILED"

	// Module-specific error codes - slicex
	CodeSlicexIndexOutOfRange  = "SLICEX_INDEX_OUT_OF_RANGE"
	CodeSlicexInvalidFunction  = "SLICEX_INVALID_FUNCTION"
	CodeSlicexOperationFailed  = "SLICEX_OPERATION_FAILED"

	// Module-specific error codes - timex
	CodeTimexInvalidFormat     = "TIMEX_INVALID_FORMAT"
	CodeTimexInvalidTimeZone   = "TIMEX_INVALID_TIMEZONE"
	CodeTimexParseError        = "TIMEX_PARSE_ERROR"
	CodeTimexCalculationError  = "TIMEX_CALCULATION_ERROR"
	CodeTimexOperationFailed   = "TIMEX_OPERATION_FAILED"

	// Module-specific error codes - validationx
	CodeValidationxRuleFailed  = "VALIDATIONX_RULE_FAILED"
	CodeValidationxChainFailed = "VALIDATIONX_CHAIN_FAILED"
	CodeValidationxInvalidRule = "VALIDATIONX_INVALID_RULE"

	// Module-specific error codes - filex
	CodeFilexNotFound          = "FILEX_NOT_FOUND"
	CodeFilexPermissionDenied  = "FILEX_PERMISSION_DENIED"
	CodeFilexOperationFailed   = "FILEX_OPERATION_FAILED"
	CodeFilexInvalidPath       = "FILEX_INVALID_PATH"
	CodeFilexReadFailed        = "FILEX_READ_FAILED"
	CodeFilexWriteFailed       = "FILEX_WRITE_FAILED"
)

// StandardError creates a standardized error with module context
func StandardError(module, operation, message string) *mdwerror.Error {
	return mdwerror.New(message).
		WithCode(mdwerror.Code(getModuleErrorCode(module, operation))).
		WithDetails(map[string]interface{}{
			"module":    module,
			"operation": operation,
		}).
		WithSeverity(mdwerror.SeverityMedium)
}

// ModuleError creates an error specific to a module operation
func ModuleError(module, operation string, cause error, details map[string]interface{}) *mdwerror.Error {
	if details == nil {
		details = make(map[string]interface{})
	}
	details["module"] = module
	details["operation"] = operation

	code := mdwerror.Code(getModuleErrorCode(module, operation))
	severity := getSeverityFromError(cause)

	if cause != nil {
		return mdwerror.Wrap(cause, fmt.Sprintf("%s.%s failed", module, operation)).
			WithCode(code).
			WithDetails(details).
			WithSeverity(severity)
	}

	return mdwerror.New(fmt.Sprintf("%s.%s failed", module, operation)).
		WithCode(code).
		WithDetails(details).
		WithSeverity(severity)
}

// ValidationError creates a standardized validation error
func ValidationError(module, field string, value interface{}, message string) *mdwerror.Error {
	return mdwerror.New(message).
		WithCode(mdwerror.Code(fmt.Sprintf("%s_VALIDATION_FAILED", strings.ToUpper(module)))).
		WithDetails(map[string]interface{}{
			"module": module,
			"field":  field,
			"value":  value,
		}).
		WithSeverity(mdwerror.SeverityLow)
}

// InputError creates a standardized input validation error
func InputError(module, operation string, input interface{}, expected string) *mdwerror.Error {
	return mdwerror.New(fmt.Sprintf("invalid input for %s.%s", module, operation)).
		WithCode(mdwerror.Code(CodeInvalidInput)).
		WithDetails(map[string]interface{}{
			"module":    module,
			"operation": operation,
			"input":     input,
			"expected":  expected,
		}).
		WithSeverity(mdwerror.SeverityMedium)
}

// FormatError creates a standardized format error
func FormatError(module string, input interface{}, expectedFormat string) *mdwerror.Error {
	return mdwerror.New(fmt.Sprintf("invalid format in %s", module)).
		WithCode(mdwerror.Code(getFormatErrorCode(module))).
		WithDetails(map[string]interface{}{
			"module":          module,
			"input":           input,
			"expected_format": expectedFormat,
		}).
		WithSeverity(mdwerror.SeverityMedium)
}

// OperationError creates a standardized operation failure error
func OperationError(module, operation string, cause error, context map[string]interface{}) *mdwerror.Error {
	if context == nil {
		context = make(map[string]interface{})
	}
	context["module"] = module
	context["operation"] = operation

	return mdwerror.Wrap(cause, fmt.Sprintf("%s.%s operation failed", module, operation)).
		WithCode(mdwerror.Code(getOperationErrorCode(module))).
		WithDetails(context).
		WithSeverity(mdwerror.SeverityHigh)
}

// getModuleErrorCode returns the appropriate error code for a module operation
func getModuleErrorCode(module, operation string) string {
	switch module {
	case ModuleStringx:
		return getStringxErrorCode(operation)
	case ModuleMathx:
		return getMathxErrorCode(operation)
	case ModuleMapx:
		return getMapxErrorCode(operation)
	case ModuleSlicex:
		return getSlicexErrorCode(operation)
	case ModuleTimex:
		return getTimexErrorCode(operation)
	case ModuleValidationx:
		return getValidationxErrorCode(operation)
	case ModuleFilex:
		return getFilexErrorCode(operation)
	default:
		return CodeOperationFailed
	}
}

// Module-specific error code getters
func getStringxErrorCode(operation string) string {
	switch {
	case strings.Contains(operation, "format"):
		return CodeStringxInvalidFormat
	case strings.Contains(operation, "length"):
		return CodeStringxLengthExceeded
	case strings.Contains(operation, "encoding"):
		return CodeStringxEncodingError
	case strings.Contains(operation, "pattern"):
		return CodeStringxInvalidPattern
	default:
		return CodeInvalidInput
	}
}

func getMathxErrorCode(operation string) string {
	switch {
	case strings.Contains(operation, "divide") || strings.Contains(operation, "div"):
		return CodeMathxDivisionByZero
	case strings.Contains(operation, "precision"):
		return CodeMathxPrecisionLoss
	case strings.Contains(operation, "overflow"):
		return CodeMathxOverflow
	case strings.Contains(operation, "underflow"):
		return CodeMathxUnderflow
	case strings.Contains(operation, "decimal") || strings.Contains(operation, "parse"):
		return CodeMathxInvalidDecimal
	default:
		return CodeInvalidInput
	}
}

func getMapxErrorCode(operation string) string {
	switch {
	case strings.Contains(operation, "key") || strings.Contains(operation, "get"):
		return CodeMapxKeyNotFound
	case strings.Contains(operation, "type"):
		return CodeMapxInvalidType
	default:
		return CodeMapxOperationFailed
	}
}

func getSlicexErrorCode(operation string) string {
	switch {
	case strings.Contains(operation, "index") || strings.Contains(operation, "range"):
		return CodeSlicexIndexOutOfRange
	case strings.Contains(operation, "function") || strings.Contains(operation, "func"):
		return CodeSlicexInvalidFunction
	default:
		return CodeSlicexOperationFailed
	}
}

func getTimexErrorCode(operation string) string {
	switch {
	case strings.Contains(operation, "parse"):
		return CodeTimexParseError
	case strings.Contains(operation, "format"):
		return CodeTimexInvalidFormat
	case strings.Contains(operation, "timezone") || strings.Contains(operation, "tz"):
		return CodeTimexInvalidTimeZone
	case strings.Contains(operation, "calc") || strings.Contains(operation, "compute"):
		return CodeTimexCalculationError
	default:
		return CodeInvalidInput
	}
}

func getValidationxErrorCode(operation string) string {
	switch {
	case strings.Contains(operation, "rule"):
		return CodeValidationxRuleFailed
	case strings.Contains(operation, "chain"):
		return CodeValidationxChainFailed
	default:
		return CodeValidationxInvalidRule
	}
}

func getFilexErrorCode(operation string) string {
	switch {
	case strings.Contains(operation, "read"):
		return CodeFilexReadFailed
	case strings.Contains(operation, "write"):
		return CodeFilexWriteFailed
	case strings.Contains(operation, "permission"):
		return CodeFilexPermissionDenied
	case strings.Contains(operation, "path"):
		return CodeFilexInvalidPath
	case strings.Contains(operation, "find") || strings.Contains(operation, "exist"):
		return CodeFilexNotFound
	default:
		return CodeFilexOperationFailed
	}
}

func getFormatErrorCode(module string) string {
	switch module {
	case ModuleStringx:
		return CodeStringxInvalidFormat
	case ModuleMathx:
		return CodeMathxInvalidDecimal
	case ModuleTimex:
		return CodeTimexInvalidFormat
	default:
		return CodeInvalidFormat
	}
}

func getOperationErrorCode(module string) string {
	switch module {
	case ModuleMathx:
		return CodeMathxOperationFailed
	case ModuleMapx:
		return CodeMapxOperationFailed
	case ModuleSlicex:
		return CodeSlicexOperationFailed
	case ModuleTimex:
		return CodeTimexOperationFailed
	case ModuleFilex:
		return CodeFilexOperationFailed
	default:
		return CodeOperationFailed
	}
}

// getSeverityFromError determines appropriate severity based on error type
func getSeverityFromError(cause error) mdwerror.Severity {
	if cause == nil {
		return mdwerror.SeverityLow
	}

	errStr := cause.Error()
	switch {
	case strings.Contains(errStr, "permission") || strings.Contains(errStr, "access"):
		return mdwerror.SeverityHigh
	case strings.Contains(errStr, "not found") || strings.Contains(errStr, "missing"):
		return mdwerror.SeverityMedium
	case strings.Contains(errStr, "invalid") || strings.Contains(errStr, "format"):
		return mdwerror.SeverityLow
	case strings.Contains(errStr, "overflow") || strings.Contains(errStr, "underflow"):
		return mdwerror.SeverityHigh
	case strings.Contains(errStr, "divide") || strings.Contains(errStr, "zero"):
		return mdwerror.SeverityHigh
	default:
		return mdwerror.SeverityMedium
	}
}

// IsModuleError checks if an error belongs to a specific module
func IsModuleError(err error, module string) bool {
	if mdwErr, ok := err.(*mdwerror.Error); ok {
		if details := mdwErr.Details(); details != nil {
			if mod, exists := details["module"]; exists {
				return mod == module
			}
		}
	}
	return false
}

// GetErrorModule extracts the module name from a standardized error
func GetErrorModule(err error) string {
	if mdwErr, ok := err.(*mdwerror.Error); ok {
		if details := mdwErr.Details(); details != nil {
			if mod, exists := details["module"]; exists {
				if modStr, ok := mod.(string); ok {
					return modStr
				}
			}
		}
	}
	return ""
}

// GetErrorOperation extracts the operation name from a standardized error
func GetErrorOperation(err error) string {
	if mdwErr, ok := err.(*mdwerror.Error); ok {
		if details := mdwErr.Details(); details != nil {
			if op, exists := details["operation"]; exists {
				if opStr, ok := op.(string); ok {
					return opStr
				}
			}
		}
	}
	return ""
}