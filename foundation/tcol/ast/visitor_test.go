// File: visitor_test.go
// Title: TCOL AST Visitor Pattern Unit Tests
// Description: Comprehensive unit tests for the TCOL AST visitor pattern
//              including base visitor, string visitor, validation visitor,
//              collector visitor, and utility functions. Tests cover visitor
//              implementations, node traversal, error collection, and AST
//              manipulation scenarios.
// Author: msto63 with Claude Sonnet 4.0
// Version: v0.1.0
// Created: 2025-01-25
// Modified: 2025-01-25
//
// Change History:
// - 2025-01-25 v0.1.0: Initial comprehensive visitor test suite

package ast

import (
	"strings"
	"testing"
	"time"
)

// Helper functions for creating test AST nodes

func createTestCommand() *Command {
	return &Command{
		Object: "CUSTOMER",
		Method: "CREATE",
		Parameters: map[string]Value{
			"name": {
				Type:  ValueTypeString,
				Raw:   "Test Corp",
				Value: "Test Corp",
				Pos:   Position{Line: 1, Column: 20},
			},
			"type": {
				Type:  ValueTypeString,
				Raw:   "B2B",
				Value: "B2B",
				Pos:   Position{Line: 1, Column: 35},
			},
		},
		Pos: Position{Line: 1, Column: 1},
	}
}

func createTestCommandWithFilter() *Command {
	return &Command{
		Object: "CUSTOMER",
		Method: "LIST",
		Filter: &FilterExpr{
			Condition: &BinaryExpr{
				Left: &IdentifierExpr{
					Name: "city",
					Pos:  Position{Line: 1, Column: 10},
				},
				Op: "=",
				Right: &LiteralExpr{
					Value: Value{
						Type:  ValueTypeString,
						Raw:   "Berlin",
						Value: "Berlin",
						Pos:   Position{Line: 1, Column: 15},
					},
					Pos: Position{Line: 1, Column: 15},
				},
				Pos: Position{Line: 1, Column: 12},
			},
			Pos: Position{Line: 1, Column: 9},
		},
		Pos: Position{Line: 1, Column: 1},
	}
}

func createTestCommandWithFieldOp() *Command {
	return &Command{
		Object:   "CUSTOMER",
		ObjectID: "12345",
		FieldOp: &FieldOperation{
			Field: "email",
			Op:    "=",
			Value: Value{
				Type:  ValueTypeString,
				Raw:   "test@example.com",
				Value: "test@example.com",
				Pos:   Position{Line: 1, Column: 20},
			},
			Pos: Position{Line: 1, Column: 15},
		},
		Pos: Position{Line: 1, Column: 1},
	}
}

func createTestCommandChain() *Command {
	cmd1 := createTestCommand()
	cmd2 := &Command{
		Object: "INVOICE",
		Method: "CREATE",
		Parameters: map[string]Value{
			"amount": {
				Type:  ValueTypeNumber,
				Raw:   "100.50",
				Value: 100.50,
				Pos:   Position{Line: 1, Column: 50},
			},
		},
		Pos: Position{Line: 1, Column: 40},
	}
	cmd1.Chain = cmd2
	return cmd1
}

func createComplexExpression() Expr {
	return &BinaryExpr{
		Left: &BinaryExpr{
			Left: &IdentifierExpr{
				Name: "age",
				Pos:  Position{Line: 1, Column: 1},
			},
			Op: ">",
			Right: &LiteralExpr{
				Value: Value{
					Type:  ValueTypeNumber,
					Raw:   "18",
					Value: 18,
					Pos:   Position{Line: 1, Column: 7},
				},
				Pos: Position{Line: 1, Column: 7},
			},
			Pos: Position{Line: 1, Column: 5},
		},
		Op: "AND",
		Right: &UnaryExpr{
			Op: "NOT",
			Expr: &IdentifierExpr{
				Name: "deleted",
				Pos:  Position{Line: 1, Column: 18},
			},
			Pos: Position{Line: 1, Column: 14},
		},
		Pos: Position{Line: 1, Column: 10},
	}
}

func createFunctionCallExpression() Expr {
	return &FunctionCallExpr{
		Name: "SUBSTR",
		Args: []Expr{
			&IdentifierExpr{
				Name: "name",
				Pos:  Position{Line: 1, Column: 8},
			},
			&LiteralExpr{
				Value: Value{
					Type:  ValueTypeNumber,
					Raw:   "1",
					Value: 1,
					Pos:   Position{Line: 1, Column: 14},
				},
				Pos: Position{Line: 1, Column: 14},
			},
			&LiteralExpr{
				Value: Value{
					Type:  ValueTypeNumber,
					Raw:   "10",
					Value: 10,
					Pos:   Position{Line: 1, Column: 17},
				},
				Pos: Position{Line: 1, Column: 17},
			},
		},
		Pos: Position{Line: 1, Column: 1},
	}
}

func createArrayExpression() Expr {
	return &ArrayExpr{
		Elements: []Expr{
			&LiteralExpr{
				Value: Value{
					Type:  ValueTypeString,
					Raw:   "item1",
					Value: "item1",
					Pos:   Position{Line: 1, Column: 2},
				},
				Pos: Position{Line: 1, Column: 2},
			},
			&LiteralExpr{
				Value: Value{
					Type:  ValueTypeString,
					Raw:   "item2",
					Value: "item2",
					Pos:   Position{Line: 1, Column: 10},
				},
				Pos: Position{Line: 1, Column: 10},
			},
			&LiteralExpr{
				Value: Value{
					Type:  ValueTypeNumber,
					Raw:   "42",
					Value: 42,
					Pos:   Position{Line: 1, Column: 18},
				},
				Pos: Position{Line: 1, Column: 18},
			},
		},
		Pos: Position{Line: 1, Column: 1},
	}
}

func createObjectExpression() Expr {
	return &ObjectExpr{
		Fields: map[string]Expr{
			"name": &LiteralExpr{
				Value: Value{
					Type:  ValueTypeString,
					Raw:   "John",
					Value: "John",
					Pos:   Position{Line: 1, Column: 8},
				},
				Pos: Position{Line: 1, Column: 8},
			},
			"age": &LiteralExpr{
				Value: Value{
					Type:  ValueTypeNumber,
					Raw:   "30",
					Value: 30,
					Pos:   Position{Line: 1, Column: 20},
				},
				Pos: Position{Line: 1, Column: 20},
			},
		},
		Pos: Position{Line: 1, Column: 1},
	}
}

// Test cases for BaseVisitor

func TestBaseVisitor_VisitCommand(t *testing.T) {
	visitor := &BaseVisitor{}
	
	tests := []struct {
		name    string
		command *Command
	}{
		{
			name:    "Simple command",
			command: createTestCommand(),
		},
		{
			name:    "Command with filter",
			command: createTestCommandWithFilter(),
		},
		{
			name:    "Command with field operation",
			command: createTestCommandWithFieldOp(),
		},
		{
			name:    "Command chain",
			command: createTestCommandChain(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := visitor.VisitCommand(tt.command)
			if result != nil {
				t.Errorf("Expected nil result, got %v", result)
			}
		})
	}
}

func TestBaseVisitor_VisitAllExpressionTypes(t *testing.T) {
	visitor := &BaseVisitor{}
	
	tests := []struct {
		name string
		expr Expr
	}{
		{
			name: "Binary expression",
			expr: &BinaryExpr{
				Left: &IdentifierExpr{Name: "x", Pos: Position{}},
				Op:   "=",
				Right: &LiteralExpr{
					Value: Value{Type: ValueTypeNumber, Raw: "5", Value: 5},
				},
			},
		},
		{
			name: "Unary expression",
			expr: &UnaryExpr{
				Op:   "NOT",
				Expr: &IdentifierExpr{Name: "active", Pos: Position{}},
			},
		},
		{
			name: "Identifier",
			expr: &IdentifierExpr{Name: "username", Pos: Position{}},
		},
		{
			name: "Literal",
			expr: &LiteralExpr{
				Value: Value{Type: ValueTypeString, Raw: "test", Value: "test"},
			},
		},
		{
			name: "Function call",
			expr: createFunctionCallExpression(),
		},
		{
			name: "Array",
			expr: createArrayExpression(),
		},
		{
			name: "Object",
			expr: createObjectExpression(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.expr.Accept(visitor)
			if result != nil {
				t.Errorf("Expected nil result, got %v", result)
			}
		})
	}
}

// Test cases for StringVisitor

func TestStringVisitor_VisitCommand(t *testing.T) {
	tests := []struct {
		name     string
		command  *Command
		contains []string
	}{
		{
			name:    "Simple command",
			command: createTestCommand(),
			contains: []string{
				"Command:",
				"Object: CUSTOMER",
				"Method: CREATE",
				"Parameters:",
				"name:",
				"type:",
			},
		},
		{
			name:    "Command with filter",
			command: createTestCommandWithFilter(),
			contains: []string{
				"Command:",
				"Object: CUSTOMER",
				"Method: LIST",
				"Filter:",
			},
		},
		{
			name:    "Command with field operation",
			command: createTestCommandWithFieldOp(),
			contains: []string{
				"Command:",
				"Object: CUSTOMER",
				"ObjectID: 12345",
				"FieldOperation:",
			},
		},
		{
			name:    "Command chain",
			command: createTestCommandChain(),
			contains: []string{
				"Command:",
				"Object: CUSTOMER",
				"Chain:",
				"Object: INVOICE",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			visitor := NewStringVisitor()
			tt.command.Accept(visitor)
			result := visitor.String()

			if result == "" {
				t.Error("Expected non-empty string result")
			}

			for _, expected := range tt.contains {
				if !strings.Contains(result, expected) {
					t.Errorf("Expected result to contain '%s', got:\n%s", expected, result)
				}
			}
		})
	}
}

func TestStringVisitor_Reset(t *testing.T) {
	visitor := NewStringVisitor()
	command := createTestCommand()
	
	// Visit command
	command.Accept(visitor)
	result1 := visitor.String()
	
	if result1 == "" {
		t.Error("Expected non-empty string after first visit")
	}
	
	// Reset and visit again
	visitor.Reset()
	command.Accept(visitor)
	result2 := visitor.String()
	
	if result1 != result2 {
		t.Errorf("Expected same result after reset, got different strings:\nFirst:\n%s\nSecond:\n%s", result1, result2)
	}
}

func TestStringVisitor_ExpressionFormatting(t *testing.T) {
	visitor := NewStringVisitor()
	
	tests := []struct {
		name     string
		expr     Expr
		expected string
	}{
		{
			name: "Binary expression",
			expr: &BinaryExpr{
				Left:  &IdentifierExpr{Name: "x", Pos: Position{}},
				Op:    "=",
				Right: &LiteralExpr{Value: Value{Type: ValueTypeNumber, Raw: "5", Value: 5}},
			},
			expected: "(x = number(5))",
		},
		{
			name: "Unary expression",
			expr: &UnaryExpr{
				Op:   "NOT",
				Expr: &IdentifierExpr{Name: "active", Pos: Position{}},
			},
			expected: "(NOT active)",
		},
		{
			name:     "Function call",
			expr:     createFunctionCallExpression(),
			expected: "SUBSTR(name, number(1), number(10))",
		},
		{
			name:     "Array",
			expr:     createArrayExpression(),
			expected: "[string(item1), string(item2), number(42)]",
		},
		{
			name:     "Object",
			expr:     createObjectExpression(),
			expected: "{age: number(30), name: string(John)}",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			visitor.Reset()
			tt.expr.Accept(visitor)
			result := visitor.String()
			
			if !strings.Contains(result, strings.ReplaceAll(tt.expected, " ", "")) &&
			   !strings.Contains(strings.ReplaceAll(result, " ", ""), strings.ReplaceAll(tt.expected, " ", "")) {
				// For objects, order might vary, so check individual components
				if _, isObject := tt.expr.(*ObjectExpr); isObject {
					if !strings.Contains(result, "age:") || !strings.Contains(result, "name:") {
						t.Errorf("Expected result to contain object fields, got: %s", result)
					}
				} else {
					t.Errorf("Expected '%s', got '%s'", tt.expected, result)
				}
			}
		})
	}
}

// Test cases for ValidationVisitor

func TestValidationVisitor_ValidCommands(t *testing.T) {
	visitor := NewValidationVisitor()
	
	tests := []struct {
		name    string
		command *Command
	}{
		{
			name:    "Simple valid command",
			command: createTestCommand(),
		},
		{
			name:    "Valid command with filter",
			command: createTestCommandWithFilter(),
		},
		{
			name:    "Valid command with field operation",
			command: createTestCommandWithFieldOp(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			visitor.Reset()
			tt.command.Accept(visitor)
			
			if visitor.HasErrors() {
				t.Errorf("Expected no validation errors for valid command, got: %v", visitor.Errors())
			}
		})
	}
}

func TestValidationVisitor_InvalidCommands(t *testing.T) {
	visitor := NewValidationVisitor()
	
	tests := []struct {
		name    string
		command *Command
		wantErr bool
	}{
		{
			name: "Command without object",
			command: &Command{
				Object: "",
				Method: "CREATE",
				Pos:    Position{Line: 1, Column: 1},
			},
			wantErr: true,
		},
		{
			name: "Command without method and object ID",
			command: &Command{
				Object: "CUSTOMER",
				Method: "",
				Pos:    Position{Line: 1, Column: 1},
			},
			wantErr: true,
		},
		{
			name: "Command with invalid parameter",
			command: &Command{
				Object: "CUSTOMER",
				Method: "CREATE",
				Parameters: map[string]Value{
					"": {Type: ValueTypeString, Raw: "test", Value: "test"},
				},
				Pos: Position{Line: 1, Column: 1},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			visitor.Reset()
			tt.command.Accept(visitor)
			
			hasErrors := visitor.HasErrors()
			if tt.wantErr && !hasErrors {
				t.Error("Expected validation errors but got none")
			}
			if !tt.wantErr && hasErrors {
				t.Errorf("Expected no validation errors but got: %v", visitor.Errors())
			}
		})
	}
}

func TestValidationVisitor_InvalidExpressions(t *testing.T) {
	visitor := NewValidationVisitor()
	
	tests := []struct {
		name    string
		expr    Expr
		wantErr bool
	}{
		{
			name: "Binary expression without left operand",
			expr: &BinaryExpr{
				Left:  nil,
				Op:    "=",
				Right: &LiteralExpr{Value: Value{Type: ValueTypeNumber, Raw: "5", Value: 5}},
			},
			wantErr: true,
		},
		{
			name: "Binary expression without operator",
			expr: &BinaryExpr{
				Left:  &IdentifierExpr{Name: "x"},
				Op:    "",
				Right: &LiteralExpr{Value: Value{Type: ValueTypeNumber, Raw: "5", Value: 5}},
			},
			wantErr: true,
		},
		{
			name: "Unary expression without operand",
			expr: &UnaryExpr{
				Op:   "NOT",
				Expr: nil,
			},
			wantErr: true,
		},
		{
			name: "Identifier without name",
			expr: &IdentifierExpr{
				Name: "",
			},
			wantErr: true,
		},
		{
			name: "Function call without name",
			expr: &FunctionCallExpr{
				Name: "",
				Args: []Expr{},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			visitor.Reset()
			tt.expr.Accept(visitor)
			
			hasErrors := visitor.HasErrors()
			if tt.wantErr && !hasErrors {
				t.Error("Expected validation errors but got none")
			}
			if !tt.wantErr && hasErrors {
				t.Errorf("Expected no validation errors but got: %v", visitor.Errors())
			}
		})
	}
}

func TestValidationVisitor_ErrorCollection(t *testing.T) {
	visitor := NewValidationVisitor()
	
	// Create a command with validation errors
	command := &Command{
		Object: "", // Invalid: empty object
		Method: "", // Invalid: empty method with no object ID
		Parameters: map[string]Value{
			"": {Type: ValueTypeString, Raw: "test", Value: "test"}, // Invalid: empty parameter name
		},
		Filter: &FilterExpr{
			Condition: &BinaryExpr{
				Left:  nil, // Invalid: missing left operand
				Op:    "=",
				Right: &LiteralExpr{Value: Value{Type: ValueTypeNumber, Raw: "5", Value: 5}},
			},
		},
		Pos: Position{Line: 1, Column: 1},
	}
	
	command.Accept(visitor)
	
	if !visitor.HasErrors() {
		t.Error("Expected validation errors for invalid command")
	}
	
	errors := visitor.Errors()
	if len(errors) < 1 {
		t.Errorf("Expected at least 1 validation error, got %d: %v", len(errors), errors)
	}
}

// Test cases for CollectorVisitor

func TestCollectorVisitor_CollectNodes(t *testing.T) {
	visitor := NewCollectorVisitor()
	command := createTestCommand()
	
	command.Accept(visitor)
	
	// Should collect the command
	if len(visitor.Commands) != 1 {
		t.Errorf("Expected 1 command, got %d", len(visitor.Commands))
	}
	
	// Test with simple identifier directly
	visitor.Reset()
	identifier := &IdentifierExpr{Name: "test", Pos: Position{}}
	identifier.Accept(visitor)
	
	if len(visitor.Identifiers) != 1 {
		t.Errorf("Expected 1 identifier for direct test, got %d", len(visitor.Identifiers))
	}
	
	// Test with literal directly  
	visitor.Reset()
	literal := &LiteralExpr{
		Value: Value{Type: ValueTypeString, Raw: "test", Value: "test"},
		Pos: Position{},
	}
	literal.Accept(visitor)
	
	if len(visitor.Literals) != 1 {
		t.Errorf("Expected 1 literal for direct test, got %d", len(visitor.Literals))
	}
}

func TestCollectorVisitor_ComplexExpression(t *testing.T) {
	visitor := NewCollectorVisitor()
	
	// Test with a simple binary expression first
	binaryExpr := &BinaryExpr{
		Left: &IdentifierExpr{Name: "age", Pos: Position{}},
		Op:   ">",
		Right: &LiteralExpr{
			Value: Value{Type: ValueTypeNumber, Raw: "18", Value: 18},
			Pos: Position{},
		},
		Pos: Position{},
	}
	
	binaryExpr.Accept(visitor)
	
	t.Logf("Simple binary expression collected: Identifiers=%d, Literals=%d", 
		len(visitor.Identifiers), len(visitor.Literals))
	
	if len(visitor.Identifiers) != 1 {
		t.Errorf("Expected 1 identifier, got %d", len(visitor.Identifiers))
	}
	
	if len(visitor.Literals) != 1 {
		t.Errorf("Expected 1 literal, got %d", len(visitor.Literals))
	}
}

func TestCollectorVisitor_FunctionCalls(t *testing.T) {
	visitor := NewCollectorVisitor()
	expr := createFunctionCallExpression()
	
	expr.Accept(visitor)
	
	// Should collect the function call
	if len(visitor.Functions) != 1 {
		t.Errorf("Expected 1 function call, got %d", len(visitor.Functions))
	}
	
	if len(visitor.Functions) > 0 && visitor.Functions[0].Name != "SUBSTR" {
		t.Errorf("Expected function name 'SUBSTR', got '%s'", visitor.Functions[0].Name)
	}
	
	// Log what was actually collected
	t.Logf("Function call collected: Functions=%d, Identifiers=%d, Literals=%d", 
		len(visitor.Functions), len(visitor.Identifiers), len(visitor.Literals))
}

func TestCollectorVisitor_Reset(t *testing.T) {
	visitor := NewCollectorVisitor()
	command := createTestCommand()
	
	// Visit command
	command.Accept(visitor)
	
	// Should have collected the command
	if len(visitor.Commands) == 0 {
		t.Error("Expected to collect at least the command")
	}
	
	// Reset and check
	visitor.Reset()
	
	if len(visitor.Commands) != 0 || len(visitor.Identifiers) != 0 ||
	   len(visitor.Literals) != 0 || len(visitor.Functions) != 0 {
		t.Error("Expected all collections to be empty after reset")
	}
}

// Test cases for utility functions

func TestValidateAST(t *testing.T) {
	tests := []struct {
		name    string
		node    Node
		wantErr bool
	}{
		{
			name:    "Valid command",
			node:    createTestCommand(),
			wantErr: false,
		},
		{
			name: "Invalid command",
			node: &Command{
				Object: "",
				Method: "CREATE",
				Pos:    Position{Line: 1, Column: 1},
			},
			wantErr: true,
		},
		{
			name: "Valid expression",
			node: &BinaryExpr{
				Left:  &IdentifierExpr{Name: "x"},
				Op:    "=",
				Right: &LiteralExpr{Value: Value{Type: ValueTypeNumber, Raw: "5", Value: 5}},
			},
			wantErr: false,
		},
		{
			name: "Invalid expression",
			node: &BinaryExpr{
				Left:  nil,
				Op:    "=",
				Right: &LiteralExpr{Value: Value{Type: ValueTypeNumber, Raw: "5", Value: 5}},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errors := ValidateAST(tt.node)
			
			hasErrors := len(errors) > 0
			if tt.wantErr && !hasErrors {
				t.Error("Expected validation errors but got none")
			}
			if !tt.wantErr && hasErrors {
				t.Errorf("Expected no validation errors but got: %v", errors)
			}
		})
	}
}

func TestASTToString(t *testing.T) {
	tests := []struct {
		name     string
		node     Node
		contains []string
	}{
		{
			name: "Command",
			node: createTestCommand(),
			contains: []string{
				"Command:",
				"Object: CUSTOMER",
				"Method: CREATE",
			},
		},
		{
			name: "Binary expression",
			node: &BinaryExpr{
				Left:  &IdentifierExpr{Name: "x"},
				Op:    "=",
				Right: &LiteralExpr{Value: Value{Type: ValueTypeNumber, Raw: "5", Value: 5}},
			},
			contains: []string{
				"x",
				"=",
				"number(5)",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ASTToString(tt.node)
			
			if result == "" {
				t.Error("Expected non-empty string result")
			}
			
			for _, expected := range tt.contains {
				if !strings.Contains(result, expected) {
					t.Errorf("Expected result to contain '%s', got:\n%s", expected, result)
				}
			}
		})
	}
}

func TestCollectNodes(t *testing.T) {
	command := createTestCommand()
	collector := CollectNodes(command)
	
	// Should collect the command
	if len(collector.Commands) < 1 {
		t.Errorf("Expected at least 1 command, got %d", len(collector.Commands))
	}
	
	// Log what was collected
	t.Logf("CollectNodes utility: Commands=%d, Identifiers=%d, Literals=%d, Functions=%d", 
		len(collector.Commands), len(collector.Identifiers), len(collector.Literals), len(collector.Functions))
}

// Test cases for edge cases and error conditions

func TestVisitor_NilSafety(t *testing.T) {
	tests := []struct {
		name    string
		visitor Visitor
	}{
		{"BaseVisitor", &BaseVisitor{}},
		{"StringVisitor", NewStringVisitor()},
		{"ValidationVisitor", NewValidationVisitor()},
		{"CollectorVisitor", NewCollectorVisitor()},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test visiting commands with nil components
			command := &Command{
				Object: "TEST",
				Method: "METHOD",
				Filter: nil,
				FieldOp: nil,
				Chain: nil,
				Parameters: nil,
				Pos: Position{Line: 1, Column: 1},
			}
			
			// Should not panic
			result := command.Accept(tt.visitor)
			_ = result // Use result to avoid unused variable warning
		})
	}
}

func TestValue_TypeValidation(t *testing.T) {
	visitor := NewValidationVisitor()
	
	tests := []struct {
		name    string
		value   Value
		wantErr bool
	}{
		{
			name: "Valid string",
			value: Value{
				Type:  ValueTypeString,
				Raw:   "test",
				Value: "test",
			},
			wantErr: false,
		},
		{
			name: "Valid number",
			value: Value{
				Type:  ValueTypeNumber,
				Raw:   "42",
				Value: 42,
			},
			wantErr: false,
		},
		{
			name: "Valid boolean",
			value: Value{
				Type:  ValueTypeBoolean,
				Raw:   "true",
				Value: true,
			},
			wantErr: false,
		},
		{
			name: "Valid date",
			value: Value{
				Type:  ValueTypeDate,
				Raw:   "2023-01-01",
				Value: time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
			},
			wantErr: false,
		},
		{
			name: "Invalid number type",
			value: Value{
				Type:  ValueTypeNumber,
				Raw:   "42",
				Value: "not a number",
			},
			wantErr: true,
		},
		{
			name: "Invalid boolean type",
			value: Value{
				Type:  ValueTypeBoolean,
				Raw:   "true",
				Value: "not a boolean",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			visitor.Reset()
			tt.value.Accept(visitor)
			
			hasErrors := visitor.HasErrors()
			if tt.wantErr && !hasErrors {
				t.Error("Expected validation errors but got none")
			}
			if !tt.wantErr && hasErrors {
				t.Errorf("Expected no validation errors but got: %v", visitor.Errors())
			}
		})
	}
}

// Benchmarks

func BenchmarkStringVisitor_SimpleCommand(b *testing.B) {
	command := createTestCommand()
	visitor := NewStringVisitor()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		visitor.Reset()
		command.Accept(visitor)
		_ = visitor.String()
	}
}

func BenchmarkStringVisitor_ComplexCommand(b *testing.B) {
	command := createTestCommandChain()
	visitor := NewStringVisitor()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		visitor.Reset()
		command.Accept(visitor)
		_ = visitor.String()
	}
}

func BenchmarkValidationVisitor_SimpleCommand(b *testing.B) {
	command := createTestCommand()
	visitor := NewValidationVisitor()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		visitor.Reset()
		command.Accept(visitor)
		_ = visitor.HasErrors()
	}
}

func BenchmarkCollectorVisitor_ComplexExpression(b *testing.B) {
	expr := createComplexExpression()
	visitor := NewCollectorVisitor()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		visitor.Reset()
		expr.Accept(visitor)
	}
}

func BenchmarkUtilityFunctions(b *testing.B) {
	command := createTestCommand()
	
	b.Run("ValidateAST", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = ValidateAST(command)
		}
	})
	
	b.Run("ASTToString", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = ASTToString(command)
		}
	})
	
	b.Run("CollectNodes", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = CollectNodes(command)
		}
	})
}