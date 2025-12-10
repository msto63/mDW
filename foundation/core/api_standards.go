// File: api_standards.go
// Title: API Standards and Consistency Guidelines
// Description: Defines standard patterns and interfaces for consistent APIs
//              across all mDW foundation modules. This ensures predictable
//              developer experience and maintainable codebase.
// Author: msto63 with Claude Sonnet 4.0
// Version: v0.1.0
// Created: 2025-01-25
// Modified: 2025-01-25
//
// Change History:
// - 2025-01-25 v0.1.0: Initial API standards implementation

package core

import (
	"context"
	"time"
)

// StandardOptions defines the common options pattern used across modules
type StandardOptions struct {
	// Context for cancellation and timeouts
	Context context.Context
	// Timeout for operations (if applicable)
	Timeout time.Duration
	// Strict mode for validation and operations
	Strict bool
	// MaxRetries for operations that can be retried
	MaxRetries int
}

// DefaultOptions returns sensible defaults for StandardOptions
func DefaultOptions() StandardOptions {
	return StandardOptions{
		Context:    context.Background(),
		Timeout:    30 * time.Second,
		Strict:     false,
		MaxRetries: 3,
	}
}

// ValidationResult represents a standardized validation result
type ValidationResult struct {
	Valid   bool
	Message string
	Code    string
	Details map[string]interface{}
}

// NewValidationResult creates a new validation result
func NewValidationResult(valid bool, code, message string) ValidationResult {
	return ValidationResult{
		Valid:   valid,
		Message: message,
		Code:    code,
		Details: make(map[string]interface{}),
	}
}

// ValidationError creates an error from validation result
func (vr ValidationResult) Error() string {
	if vr.Valid {
		return ""
	}
	return vr.Message
}

// StandardValidator defines the interface for all validators
type StandardValidator interface {
	Validate(value interface{}) ValidationResult
	ValidateString(value string) ValidationResult
}

// TransformFunc represents a generic transformation function
type TransformFunc[T, R any] func(T) R

// PredicateFunc represents a generic predicate function  
type PredicateFunc[T any] func(T) bool

// CompareFunc represents a generic comparison function
type CompareFunc[T any] func(T, T) int

// StandardNaming defines naming conventions for consistent APIs
type StandardNaming struct{}

// Naming conventions documentation:
// 
// Function Prefixes:
// - Is* / Has* / Can*: Boolean predicates (IsEmpty, HasKey, CanWrite)
// - To* / From*: Conversions (ToString, ToLower, FromJSON, FromString)
// - Must*: Panic versions of functions (MustParse, MustCreate)
// - Validate*: Validation functions (ValidateEmail, ValidateRequired)
// - Get* / Set*: Property accessors (GetSize, SetPermissions)
// - Create* / New*: Constructors (CreateFile, NewDecimal)
// - Find* / Search*: Search operations (FindFirst, SearchIn)
// - Filter* / Map* / Reduce*: Collection operations (FilterBy, MapTo, ReduceSum)
//
// Parameter Order Conventions:
// 1. Subject/Source (what we're operating on)
// 2. Target/Destination (where result goes)
// 3. Options/Configuration (how to do it)
// 4. Context (cancellation/metadata)
//
// Return Value Conventions:
// - (result, error): Standard Go pattern for operations that can fail
// - (result, bool): For operations with optional results (found/not found)
// - result: For operations that always succeed
// - error: For validation-only operations
//
// Generic Type Parameter Conventions:
// - T, U, V: Generic types in order of appearance
// - K, V: Key-Value pairs (maps)
// - R: Result type for transformations
// - E: Element type for collections
// - C: Constraint types
//
// Error Handling Conventions:
// - Always wrap errors with context: fmt.Errorf("operation failed: %w", err) 
// - Use consistent error messages: "{operation} failed for {context}: {reason}"
// - Include relevant details in error context
// - Use error codes from standards.go for categorization

// ParsingOptions defines standard options for parsing operations
type ParsingOptions struct {
	StandardOptions
	// StrictMode requires exact format matching
	StrictMode bool
	// AllowEmpty allows empty/nil inputs without error
	AllowEmpty bool
	// DefaultValue used when input is empty (if AllowEmpty is true)
	DefaultValue interface{}
}

// DefaultParsingOptions returns defaults for parsing operations
func DefaultParsingOptions() ParsingOptions {
	return ParsingOptions{
		StandardOptions: DefaultOptions(),
		StrictMode:      false,
		AllowEmpty:      false,
		DefaultValue:    nil,
	}
}

// CollectionOptions defines standard options for collection operations
type CollectionOptions struct {
	StandardOptions
	// PreserveOrder maintains input order in output
	PreserveOrder bool
	// RemoveDuplicates eliminates duplicate values
	RemoveDuplicates bool
	// MaxSize limits the size of result collections
	MaxSize int
}

// DefaultCollectionOptions returns defaults for collection operations
func DefaultCollectionOptions() CollectionOptions {
	return CollectionOptions{
		StandardOptions:  DefaultOptions(),
		PreserveOrder:    true,
		RemoveDuplicates: false,
		MaxSize:          -1, // No limit
	}
}

// FileOperationOptions defines standard options for file operations
type FileOperationOptions struct {
	StandardOptions
	// CreateDirs creates parent directories if they don't exist
	CreateDirs bool
	// OverwriteExisting allows overwriting existing files
	OverwriteExisting bool
	// PreservePermissions maintains original file permissions
	PreservePermissions bool
	// BufferSize for I/O operations
	BufferSize int
}

// DefaultFileOperationOptions returns defaults for file operations
func DefaultFileOperationOptions() FileOperationOptions {
	return FileOperationOptions{
		StandardOptions:     DefaultOptions(),
		CreateDirs:          false,
		OverwriteExisting:   false,
		PreservePermissions: true,
		BufferSize:          64 * 1024, // 64KB
	}
}