// File: example_test.go
// Title: Error Module Examples
// Description: Example usage patterns for the mDW error handling system.
//              These examples demonstrate common use cases and best practices.
// Author: msto63 with Claude Sonnet 4.0
// Version: v0.1.0
// Created: 2025-01-24
// Modified: 2025-01-24
//
// Change History:
// - 2025-01-24 v0.1.0: Initial implementation with comprehensive examples

package error

import (
	"database/sql"
	"fmt"
)

// ExampleNew demonstrates creating a new error with context
func ExampleNew() {
	err := New("failed to connect to database").
		WithCode(CodeDatabaseError).
		WithDetail("host", "localhost:5432").
		WithDetail("database", "mdw_users").
		WithSeverity(SeverityHigh)
	
	fmt.Println("Error:", err.Error())
	fmt.Println("Code:", err.Code())
	fmt.Println("Severity:", err.Severity())
	
	// Output:
	// Error: failed to connect to database
	// Code: DATABASE_ERROR
	// Severity: high
}

// ExampleWrap demonstrates wrapping an existing error with context
func ExampleWrap() {
	// Simulate a database error
	dbErr := sql.ErrNoRows
	
	// Wrap with business context
	err := Wrap(dbErr, "user not found during authentication").
		WithCode(CodeNotFound).
		WithDetail("user_id", "12345").
		WithOperation("authenticate_user")
	
	fmt.Println("Error:", err.Error())
	fmt.Println("Code:", err.Code())
	
	// Output:
	// Error: user not found during authentication: sql: no rows in result set
	// Code: NOT_FOUND
}

// ExampleError_WithDetails demonstrates adding multiple details to an error
func ExampleError_WithDetails() {
	details := map[string]interface{}{
		"user_id":    "user_12345",
		"operation":  "transfer_funds",
		"amount":     1000.50,
		"from_account": "ACC_001",
		"to_account":   "ACC_002",
	}
	
	err := New("insufficient funds for transfer").
		WithCode(CodeInsufficientFunds).
		WithDetails(details).
		WithSeverity(SeverityMedium)
	
	fmt.Println("Error:", err.Error())
	fmt.Println("Details count:", len(err.Details()))
	fmt.Println("Amount:", err.Details()["amount"])
	
	// Output:
	// Error: insufficient funds for transfer
	// Details count: 5
	// Amount: 1000.5
}

// ExampleError_WithContext demonstrates adding context information
func ExampleError_WithContext() {
	err := New("validation failed").
		WithCode(CodeValidationFailed).
		WithContext("user-service.CreateUser").
		WithOperation("INSERT INTO users").
		WithUserID("user_789").
		WithRequestID("req_abc123").
		WithDetail("field", "email").
		WithDetail("value", "invalid-email")
	
	fmt.Println("Context:", err.Context())
	fmt.Println("Operation:", err.Operation())
	fmt.Println("User ID:", err.UserID())
	fmt.Println("Request ID:", err.RequestID())
	
	// Output:
	// Context: user-service.CreateUser
	// Operation: INSERT INTO users
	// User ID: user_789
	// Request ID: req_abc123
}

// ExampleHasCode demonstrates checking for specific error codes
func ExampleHasCode() {
	err := New("database connection timeout").
		WithCode(CodeTimeout)
	
	if HasCode(err, CodeTimeout) {
		fmt.Println("This is a timeout error")
	}
	
	if HasCode(err, CodeDatabaseError) {
		fmt.Println("This is a database error")
	} else {
		fmt.Println("This is not a database error") 
	}
	
	// Output:
	// This is a timeout error
	// This is not a database error
}

// ExampleGetSeverityFromCode demonstrates automatic severity assignment
func ExampleGetSeverityFromCode() {
	codes := []Code{
		CodeDataCorruption,
		CodeDatabaseError,
		CodeBusinessRule,
		CodeValidationFailed,
	}
	
	for _, code := range codes {
		severity := GetSeverityFromCode(code)
		fmt.Printf("Code: %s -> Severity: %s (Should Alert: %t)\n", 
			code, severity, severity.ShouldAlert())
	}
	
	// Output:
	// Code: DATA_CORRUPTION -> Severity: critical (Should Alert: true)
	// Code: DATABASE_ERROR -> Severity: high (Should Alert: true)
	// Code: BUSINESS_RULE -> Severity: medium (Should Alert: false)
	// Code: VALIDATION_FAILED -> Severity: low (Should Alert: false)
}

// ExampleError_RootCause demonstrates finding the root cause of error chains
func ExampleError_RootCause() {
	// Create an error chain
	original := New("connection refused").WithCode(CodeConnectionFailed)
	middle := Wrap(original, "database initialization failed")
	top := Wrap(middle, "service startup failed")
	
	fmt.Println("Top error:", top.Error())
	fmt.Println("Root cause:", top.RootCause().Error())
	fmt.Println("Root cause code:", GetCode(top.RootCause()))
	
	// Output:
	// Top error: service startup failed: database initialization failed: connection refused
	// Root cause: connection refused
	// Root cause code: CONNECTION_FAILED
}

// ExampleError_MarshalJSON demonstrates JSON serialization for logging
func ExampleError_MarshalJSON() {
	err := New("TCOL command execution failed").
		WithCode(CodeTCOLExecution).
		WithContext("command-processor").
		WithDetail("command", "CUSTOMER.CREATE").
		WithDetail("user_id", "admin_123").
		WithSeverity(SeverityMedium)
	
	// This would typically be used with a JSON logger
	data, _ := err.MarshalJSON()
	fmt.Printf("JSON length: %d bytes\n", len(data))
	fmt.Println("Contains command:", string(data)[:50]+"...")
	
	// Output:
	// JSON length: 942 bytes
	// Contains command: {"code":"TCOL_EXECUTION","context":"command-proces...
}

// Example_businessLogicError demonstrates error handling in business logic
func Example_businessLogicError() {
	// Simulate a business rule violation
	processTransfer := func(amount float64, balance float64) error {
		if amount <= 0 {
			return New("invalid transfer amount").
				WithCode(CodeInvalidInput).
				WithDetail("amount", amount).
				WithDetail("rule", "amount must be positive")
		}
		
		if amount > balance {
			return New("insufficient funds for transfer").
				WithCode(CodeInsufficientFunds).
				WithDetail("requested", amount).
				WithDetail("available", balance).
				WithSeverity(SeverityMedium)
		}
		
		return nil
	}
	
	// Test with insufficient funds
	err := processTransfer(1000.00, 500.00)
	if err != nil {
		fmt.Println("Transfer failed:", err.Error())
		fmt.Println("Error code:", GetCode(err))
		
		if HasCode(err, CodeInsufficientFunds) {
			fmt.Println("Reason: Not enough money in account")
		}
	}
	
	// Output:
	// Transfer failed: insufficient funds for transfer
	// Error code: INSUFFICIENT_FUNDS
	// Reason: Not enough money in account
}

// Example_tcolError demonstrates TCOL-specific error handling
func Example_tcolError() {
	// Simulate TCOL command parsing error
	parseTCOLCommand := func(command string) error {
		if command == "" {
			return New("empty TCOL command").
				WithCode(CodeTCOLSyntax).
				WithDetail("command", command).
				WithDetail("position", 0)
		}
		
		if !contains(command, ".") {
			return New("TCOL command missing method separator").
				WithCode(CodeTCOLSyntax).
				WithDetail("command", command).
				WithDetail("expected", "OBJECT.METHOD format")
		}
		
		return nil
	}
	
	// Test invalid command
	err := parseTCOLCommand("CUSTOMER_CREATE")
	if err != nil {
		fmt.Println("Parse error:", err.Error())
		fmt.Println("Category:", GetCode(err).Category())
		fmt.Println("HTTP Status:", GetCode(err).HTTPStatus())
	}
	
	// Output:
	// Parse error: TCOL command missing method separator
	// Category: tcol
	// HTTP Status: 400
}

// Helper function for the example
func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}