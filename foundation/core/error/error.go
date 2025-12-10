// File: error.go
// Title: Core Error Implementation
// Description: Implements the main Error type with contextual information, stack traces,
//              and metadata. This provides a rich error handling system that maintains
//              compatibility with Go's standard error interface while adding powerful
//              debugging and monitoring capabilities.
// Author: msto63 with Claude Sonnet 4.0
// Version: v0.1.0
// Created: 2025-01-24
// Modified: 2025-01-24
//
// Change History:
// - 2025-01-24 v0.1.0: Initial implementation with contextual errors

package error

import (
	"encoding/json"
	"fmt"
	"runtime"
	"strings"
	"sync"
	"time"
)

// Error represents a structured error with context, codes, and metadata
type Error struct {
	// Core error information
	message   string
	cause     error
	code      Code
	severity  Severity
	timestamp time.Time
	
	// Context and metadata
	details   map[string]interface{}
	context   string
	operation string
	userID    string
	requestID string
	
	// Stack trace information
	stackTrace []StackFrame
	
	// Localization
	messageKey string
	messageArgs map[string]interface{}
}

// StackFrame represents a single frame in the stack trace
type StackFrame struct {
	Function string `json:"function"`
	File     string `json:"file"`
	Line     int    `json:"line"`
}

// Error chain and stack trace optimization constants and pools
const (
	// MaxErrorChainDepth limits the depth of error wrapping to prevent memory leaks
	MaxErrorChainDepth = 15
	
	// MaxStackFrames limits the number of stack frames captured
	MaxStackFrames = 20
	
	// StackTracePoolSize is the size of the stack trace slice pool
	StackTracePoolSize = 32
)

var (
	// stackFramePool pools stack frame slices for efficient memory reuse
	stackFramePool = sync.Pool{
		New: func() interface{} {
			return make([]StackFrame, 0, MaxStackFrames)
		},
	}
)

// getStackFrameSlice gets a stack frame slice from the pool
func getStackFrameSlice() []StackFrame {
	slice := stackFramePool.Get().([]StackFrame)
	return slice[:0] // Reset length but keep capacity
}

// putStackFrameSlice returns a stack frame slice to the pool
func putStackFrameSlice(slice []StackFrame) {
	if slice != nil && cap(slice) >= MaxStackFrames {
		stackFramePool.Put(slice)
	}
}

// New creates a new Error with the given message
func New(message string) *Error {
	return &Error{
		message:    message,
		code:       CodeUnknown,
		severity:   SeverityMedium,
		timestamp:  time.Now(),
		details:    make(map[string]interface{}),
		stackTrace: captureStackTrace(2), // Skip New and caller
	}
}

// getErrorChainDepth calculates the depth of an error chain
func getErrorChainDepth(err error) int {
	depth := 0
	current := err
	
	for current != nil && depth < MaxErrorChainDepth*2 { // Safety limit
		depth++
		if mdwErr, ok := current.(*Error); ok {
			current = mdwErr.cause
		} else {
			break
		}
	}
	
	return depth
}

// Wrap wraps an existing error with additional context
func Wrap(err error, message string) *Error {
	if err == nil {
		return nil
	}
	
	// Check error chain depth to prevent memory leaks
	if depth := getErrorChainDepth(err); depth >= MaxErrorChainDepth {
		// Create a flattened error instead of continuing the chain
		rootCause := getRootCause(err)
		return &Error{
			message:    fmt.Sprintf("%s (chain truncated at depth %d): %s", message, MaxErrorChainDepth, rootCause.Error()),
			cause:      nil, // Break the chain
			code:       CodeUnknown,
			severity:   SeverityHigh, // High severity for truncated chains
			timestamp:  time.Now(),
			details:    map[string]interface{}{"truncated": true, "original_depth": depth},
			stackTrace: captureStackTrace(2),
		}
	}
	
	// If err is already our Error type, preserve its information
	if mdwErr, ok := err.(*Error); ok {
		wrapped := &Error{
			message:     message,
			cause:       mdwErr,
			code:        mdwErr.code,
			severity:    mdwErr.severity,
			timestamp:   time.Now(),
			details:     make(map[string]interface{}),
			stackTrace:  captureStackTrace(2),
			messageKey:  mdwErr.messageKey,
			messageArgs: mdwErr.messageArgs,
		}
		// Copy details from the original error
		for k, v := range mdwErr.details {
			wrapped.details[k] = v
		}
		return wrapped
	}
	
	// Wrap standard error
	return &Error{
		message:    message,
		cause:      err,
		code:       CodeUnknown,
		severity:   SeverityMedium,
		timestamp:  time.Now(),
		details:    make(map[string]interface{}),
		stackTrace: captureStackTrace(2),
	}
}

// getRootCause helper function to get the deepest error in a chain
func getRootCause(err error) error {
	current := err
	var last error = err
	
	for current != nil {
		last = current
		if mdwErr, ok := current.(*Error); ok {
			current = mdwErr.cause
		} else {
			break
		}
	}
	
	return last
}

// Error implements the standard error interface
func (e *Error) Error() string {
	if e.cause != nil {
		return fmt.Sprintf("%s: %s", e.message, e.cause.Error())
	}
	return e.message
}

// Unwrap returns the underlying cause for error unwrapping
func (e *Error) Unwrap() error {
	return e.cause
}

// WithCode sets the error code
func (e *Error) WithCode(code Code) *Error {
	e.code = code
	if e.severity == SeverityMedium { // Only auto-set if not explicitly set
		e.severity = GetSeverityFromCode(code)
	}
	return e
}

// WithSeverity sets the error severity
func (e *Error) WithSeverity(severity Severity) *Error {
	e.severity = severity
	return e
}

// WithDetail adds a key-value detail to the error
func (e *Error) WithDetail(key string, value interface{}) *Error {
	e.details[key] = value
	return e
}

// WithDetails adds multiple key-value details to the error
func (e *Error) WithDetails(details map[string]interface{}) *Error {
	for k, v := range details {
		e.details[k] = v
	}
	return e
}

// WithContext sets the context information
func (e *Error) WithContext(context string) *Error {
	e.context = context
	return e
}

// WithOperation sets the operation that caused the error
func (e *Error) WithOperation(operation string) *Error {
	e.operation = operation
	return e
}

// WithUserID sets the user ID associated with the error
func (e *Error) WithUserID(userID string) *Error {
	e.userID = userID
	return e
}

// WithRequestID sets the request ID associated with the error
func (e *Error) WithRequestID(requestID string) *Error {
	e.requestID = requestID
	return e
}

// WithMessage sets localization information for the error message
func (e *Error) WithMessage(key string, args map[string]interface{}) *Error {
	e.messageKey = key
	e.messageArgs = args
	return e
}

// Code returns the error code
func (e *Error) Code() Code {
	return e.code
}

// Severity returns the error severity
func (e *Error) Severity() Severity {
	return e.severity
}

// Timestamp returns when the error occurred
func (e *Error) Timestamp() time.Time {
	return e.timestamp
}

// Details returns the error details
func (e *Error) Details() map[string]interface{} {
	result := make(map[string]interface{})
	for k, v := range e.details {
		result[k] = v
	}
	return result
}

// Context returns the error context
func (e *Error) Context() string {
	return e.context
}

// Operation returns the operation that caused the error
func (e *Error) Operation() string {
	return e.operation
}

// UserID returns the user ID associated with the error
func (e *Error) UserID() string {
	return e.userID
}

// RequestID returns the request ID associated with the error
func (e *Error) RequestID() string {
	return e.requestID
}

// StackTrace returns the stack trace
func (e *Error) StackTrace() []StackFrame {
	result := make([]StackFrame, len(e.stackTrace))
	copy(result, e.stackTrace)
	return result
}

// MessageKey returns the localization message key
func (e *Error) MessageKey() string {
	return e.messageKey
}

// MessageArgs returns the localization message arguments
func (e *Error) MessageArgs() map[string]interface{} {
	if e.messageArgs == nil {
		return nil
	}
	result := make(map[string]interface{})
	for k, v := range e.messageArgs {
		result[k] = v
	}
	return result
}

// RootCause returns the root cause of the error chain
func (e *Error) RootCause() error {
	cause := e.cause
	for cause != nil {
		if mdwErr, ok := cause.(*Error); ok {
			if mdwErr.cause == nil {
				return mdwErr
			}
			cause = mdwErr.cause
		} else {
			return cause
		}
	}
	return e
}

// String returns a detailed string representation of the error
func (e *Error) String() string {
	var parts []string
	
	parts = append(parts, fmt.Sprintf("Error: %s", e.message))
	parts = append(parts, fmt.Sprintf("Code: %s", e.code))
	parts = append(parts, fmt.Sprintf("Severity: %s", e.severity))
	parts = append(parts, fmt.Sprintf("Timestamp: %s", e.timestamp.Format(time.RFC3339)))
	
	if e.context != "" {
		parts = append(parts, fmt.Sprintf("Context: %s", e.context))
	}
	
	if e.operation != "" {
		parts = append(parts, fmt.Sprintf("Operation: %s", e.operation))
	}
	
	if e.userID != "" {
		parts = append(parts, fmt.Sprintf("UserID: %s", e.userID))
	}
	
	if e.requestID != "" {
		parts = append(parts, fmt.Sprintf("RequestID: %s", e.requestID))
	}
	
	if len(e.details) > 0 {
		detailStrs := make([]string, 0, len(e.details))
		for k, v := range e.details {
			detailStrs = append(detailStrs, fmt.Sprintf("%s=%v", k, v))
		}
		parts = append(parts, fmt.Sprintf("Details: {%s}", strings.Join(detailStrs, ", ")))
	}
	
	if e.cause != nil {
		parts = append(parts, fmt.Sprintf("Cause: %s", e.cause.Error()))
	}
	
	return strings.Join(parts, "\n")
}

// MarshalJSON implements json.Marshaler for structured logging
func (e *Error) MarshalJSON() ([]byte, error) {
	data := map[string]interface{}{
		"message":   e.message,
		"code":      e.code,
		"severity":  e.severity.String(),
		"timestamp": e.timestamp.Format(time.RFC3339),
		"details":   e.details,
	}
	
	if e.context != "" {
		data["context"] = e.context
	}
	
	if e.operation != "" {
		data["operation"] = e.operation
	}
	
	if e.userID != "" {
		data["user_id"] = e.userID
	}
	
	if e.requestID != "" {
		data["request_id"] = e.requestID
	}
	
	if e.cause != nil {
		data["cause"] = e.cause.Error()
	}
	
	if len(e.stackTrace) > 0 {
		data["stack_trace"] = e.stackTrace
	}
	
	if e.messageKey != "" {
		data["message_key"] = e.messageKey
		if e.messageArgs != nil {
			data["message_args"] = e.messageArgs
		}
	}
	
	return json.Marshal(data)
}

// captureStackTrace captures the current stack trace with pooling optimization
func captureStackTrace(skip int) []StackFrame {
	frames := getStackFrameSlice()
	
	for i := skip; i < MaxStackFrames+skip; i++ {
		pc, file, line, ok := runtime.Caller(i)
		if !ok {
			break
		}
		
		fn := runtime.FuncForPC(pc)
		if fn == nil {
			continue
		}
		
		frames = append(frames, StackFrame{
			Function: fn.Name(),
			File:     file,
			Line:     line,
		})
	}
	
	// Create a copy for the error since we'll return the slice to the pool
	result := make([]StackFrame, len(frames))
	copy(result, frames)
	
	// Return pooled slice
	putStackFrameSlice(frames)
	
	return result
}

// HasCode checks if an error has a specific code
func HasCode(err error, code Code) bool {
	if mdwErr, ok := err.(*Error); ok {
		return mdwErr.code == code
	}
	return false
}

// GetCode returns the error code from an error, or CodeUnknown if not a mDW error
func GetCode(err error) Code {
	if mdwErr, ok := err.(*Error); ok {
		return mdwErr.code
	}
	return CodeUnknown
}

// GetSeverity returns the error severity from an error, or SeverityMedium if not a mDW error
func GetSeverity(err error) Severity {
	if mdwErr, ok := err.(*Error); ok {
		return mdwErr.severity
	}
	return SeverityMedium
}