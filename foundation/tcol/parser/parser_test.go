// File: parser_test.go
// Title: TCOL Parser Unit Tests
// Description: Comprehensive unit tests for the TCOL recursive descent parser.
//              Tests cover all command structures, expression parsing, error
//              handling, and edge cases in TCOL syntax parsing.
// Author: msto63 with Claude Sonnet 4.0
// Version: v0.1.0
// Created: 2025-01-25
// Modified: 2025-01-25
//
// Change History:
// - 2025-01-25 v0.1.0: Initial comprehensive parser test suite

package parser

import (
	"fmt"
	"testing"

	mdwlog "github.com/msto63/mDW/foundation/core/log"
	mdwast "github.com/msto63/mDW/foundation/tcol/ast"
	mdwregistry "github.com/msto63/mDW/foundation/tcol/registry"
)

func TestParser_Parse(t *testing.T) {
	parser, _ := New(Options{
		Logger:         mdwlog.GetDefault(),
		EnableChaining: true,
	})

	tests := []struct {
		name    string
		input   string
		wantErr bool
		errMsg  string
		check   func(t *testing.T, cmd *mdwast.Command)
	}{
		{
			name:  "Simple command",
			input: "CUSTOMER.CREATE",
			check: func(t *testing.T, cmd *mdwast.Command) {
				if cmd.Object != "CUSTOMER" {
					t.Errorf("Expected object CUSTOMER, got %s", cmd.Object)
				}
				if cmd.Method != "CREATE" {
					t.Errorf("Expected method CREATE, got %s", cmd.Method)
				}
				if len(cmd.Parameters) != 0 {
					t.Errorf("Expected no parameters, got %d", len(cmd.Parameters))
				}
			},
		},
		{
			name:  "Command with string parameter",
			input: `CUSTOMER.CREATE name="John Doe"`,
			check: func(t *testing.T, cmd *mdwast.Command) {
				if cmd.Object != "CUSTOMER" {
					t.Errorf("Expected object CUSTOMER, got %s", cmd.Object)
				}
				if cmd.Method != "CREATE" {
					t.Errorf("Expected method CREATE, got %s", cmd.Method)
				}
				if len(cmd.Parameters) != 1 {
					t.Errorf("Expected 1 parameter, got %d", len(cmd.Parameters))
					return
				}
				param, exists := cmd.Parameters["name"]
				if !exists {
					t.Error("Expected parameter 'name' not found")
					return
				}
				if param.Type != mdwast.ValueTypeString {
					t.Errorf("Expected string type, got %v", param.Type)
				}
				if param.Value != "John Doe" {
					t.Errorf("Expected value 'John Doe', got %v", param.Value)
				}
			},
		},
		{
			name:  "Command with multiple parameters",
			input: `CUSTOMER.CREATE name="John Doe" age=30 active=true`,
			check: func(t *testing.T, cmd *mdwast.Command) {
				if len(cmd.Parameters) != 3 {
					t.Errorf("Expected 3 parameters, got %d", len(cmd.Parameters))
					return
				}

				// Check name parameter
				name, exists := cmd.Parameters["name"]
				if !exists || name.Value != "John Doe" {
					t.Error("Name parameter incorrect")
				}

				// Check age parameter
				age, exists := cmd.Parameters["age"]
				if !exists || age.Value != int64(30) {
					t.Errorf("Age parameter incorrect: %v", age.Value)
				}

				// Check active parameter
				active, exists := cmd.Parameters["active"]
				if !exists || active.Value != true {
					t.Error("Active parameter incorrect")
				}
			},
		},
		{
			name:  "Object access",
			input: "CUSTOMER:12345",
			check: func(t *testing.T, cmd *mdwast.Command) {
				if cmd.Object != "CUSTOMER" {
					t.Errorf("Expected object CUSTOMER, got %s", cmd.Object)
				}
				if cmd.ObjectID != "12345" {
					t.Errorf("Expected object ID 12345, got %s", cmd.ObjectID)
				}
				if cmd.Method != "" {
					t.Errorf("Expected no method, got %s", cmd.Method)
				}
			},
		},
		{
			name:  "Field read operation",
			input: "CUSTOMER:123:email",
			check: func(t *testing.T, cmd *mdwast.Command) {
				if cmd.Object != "CUSTOMER" {
					t.Errorf("Expected object CUSTOMER, got %s", cmd.Object)
				}
				if cmd.ObjectID != "123" {
					t.Errorf("Expected object ID 123, got %s", cmd.ObjectID)
				}
				if cmd.FieldOp == nil {
					t.Fatal("Expected field operation, got nil")
				}
				if cmd.FieldOp.Field != "email" {
					t.Errorf("Expected field 'email', got %s", cmd.FieldOp.Field)
				}
				if cmd.FieldOp.Op != "" {
					t.Errorf("Expected no operator for read, got %s", cmd.FieldOp.Op)
				}
			},
		},
		{
			name:  "Field write operation",
			input: `CUSTOMER:123:email="new@example.com"`,
			check: func(t *testing.T, cmd *mdwast.Command) {
				if cmd.FieldOp == nil {
					t.Fatal("Expected field operation, got nil")
				}
				if cmd.FieldOp.Field != "email" {
					t.Errorf("Expected field 'email', got %s", cmd.FieldOp.Field)
				}
				if cmd.FieldOp.Op != "=" {
					t.Errorf("Expected operator '=', got %s", cmd.FieldOp.Op)
				}
				if cmd.FieldOp.Value.Value != "new@example.com" {
					t.Errorf("Expected value 'new@example.com', got %v", cmd.FieldOp.Value.Value)
				}
			},
		},
		{
			name:  "Simple filter",
			input: `CUSTOMER[active=true].LIST`,
			check: func(t *testing.T, cmd *mdwast.Command) {
				if cmd.Filter == nil {
					t.Fatal("Expected filter, got nil")
				}
				// Check filter condition
				binExpr, ok := cmd.Filter.Condition.(*mdwast.BinaryExpr)
				if !ok {
					t.Fatal("Expected binary expression in filter")
				}
				if binExpr.Op != "=" {
					t.Errorf("Expected operator '=', got %s", binExpr.Op)
				}

				// Check left side
				leftIdent, ok := binExpr.Left.(*mdwast.IdentifierExpr)
				if !ok || leftIdent.Name != "active" {
					t.Error("Expected identifier 'active' on left")
				}

				// Check right side
				rightLit, ok := binExpr.Right.(*mdwast.LiteralExpr)
				if !ok || rightLit.Value.Value != true {
					t.Error("Expected literal 'true' on right")
				}
			},
		},
		{
			name:  "Complex filter with AND",
			input: `CUSTOMER[city="Berlin" AND age>30].LIST`,
			check: func(t *testing.T, cmd *mdwast.Command) {
				if cmd.Filter == nil {
					t.Fatal("Expected filter, got nil")
				}
				// Check that it's an AND expression
				andExpr, ok := cmd.Filter.Condition.(*mdwast.BinaryExpr)
				if !ok {
					t.Fatal("Expected binary expression for AND")
				}
				if andExpr.Op != "AND" {
					t.Errorf("Expected AND operator, got %s", andExpr.Op)
				}
			},
		},
		{
			name:  "Command chain",
			input: "CUSTOMER.LIST | EXPORT.CSV",
			check: func(t *testing.T, cmd *mdwast.Command) {
				if cmd.Object != "CUSTOMER" {
					t.Errorf("Expected object CUSTOMER, got %s", cmd.Object)
				}
				if cmd.Method != "LIST" {
					t.Errorf("Expected method LIST, got %s", cmd.Method)
				}
				if cmd.Chain == nil {
					t.Fatal("Expected chain command, got nil")
				}
				if cmd.Chain.Object != "EXPORT" {
					t.Errorf("Expected chain object EXPORT, got %s", cmd.Chain.Object)
				}
				if cmd.Chain.Method != "CSV" {
					t.Errorf("Expected chain method CSV, got %s", cmd.Chain.Method)
				}
			},
		},
		{
			name:  "Function call in filter",
			input: `CUSTOMER[age>AVG(age)].LIST`,
			check: func(t *testing.T, cmd *mdwast.Command) {
				if cmd.Filter == nil {
					t.Fatal("Expected filter, got nil")
				}
				// Check that filter contains function call
				binExpr, ok := cmd.Filter.Condition.(*mdwast.BinaryExpr)
				if !ok {
					t.Fatal("Expected binary expression")
				}
				// Right side should be function call
				funcCall, ok := binExpr.Right.(*mdwast.FunctionCallExpr)
				if !ok {
					t.Fatal("Expected function call on right side")
				}
				if funcCall.Name != "AVG" {
					t.Errorf("Expected function AVG, got %s", funcCall.Name)
				}
			},
		},
		{
			name:  "Array in filter",
			input: `CUSTOMER[status IN ["active", "pending"]].LIST`,
			check: func(t *testing.T, cmd *mdwast.Command) {
				if cmd.Filter == nil {
					t.Fatal("Expected filter, got nil")
				}
				// Check IN expression
				binExpr, ok := cmd.Filter.Condition.(*mdwast.BinaryExpr)
				if !ok {
					t.Fatal("Expected binary expression")
				}
				if binExpr.Op != "IN" {
					t.Errorf("Expected IN operator, got %s", binExpr.Op)
				}
				// Right side should be array
				arrayExpr, ok := binExpr.Right.(*mdwast.ArrayExpr)
				if !ok {
					t.Fatal("Expected array expression on right")
				}
				if len(arrayExpr.Elements) != 2 {
					t.Errorf("Expected 2 elements in array, got %d", len(arrayExpr.Elements))
				}
			},
		},
		{
			name:  "Nested parentheses in filter",
			input: `CUSTOMER[(age>18 AND active=true) OR premium=true].LIST`,
			check: func(t *testing.T, cmd *mdwast.Command) {
				if cmd.Filter == nil {
					t.Fatal("Expected filter, got nil")
				}
				// Root should be OR
				orExpr, ok := cmd.Filter.Condition.(*mdwast.BinaryExpr)
				if !ok || orExpr.Op != "OR" {
					t.Fatal("Expected OR at root")
				}
				// Left side should be AND (from parentheses)
				andExpr, ok := orExpr.Left.(*mdwast.BinaryExpr)
				if !ok || andExpr.Op != "AND" {
					t.Fatal("Expected AND on left side")
				}
			},
		},
		{
			name:  "NOT expression",
			input: `CUSTOMER[NOT active].LIST`,
			check: func(t *testing.T, cmd *mdwast.Command) {
				if cmd.Filter == nil {
					t.Fatal("Expected filter, got nil")
				}
				// Should be unary NOT
				unaryExpr, ok := cmd.Filter.Condition.(*mdwast.UnaryExpr)
				if !ok {
					t.Fatal("Expected unary expression")
				}
				if unaryExpr.Op != "NOT" {
					t.Errorf("Expected NOT operator, got %s", unaryExpr.Op)
				}
			},
		},
		{
			name:  "LIKE operator",
			input: `CUSTOMER[name LIKE "John%"].LIST`,
			check: func(t *testing.T, cmd *mdwast.Command) {
				if cmd.Filter == nil {
					t.Fatal("Expected filter, got nil")
				}
				binExpr, ok := cmd.Filter.Condition.(*mdwast.BinaryExpr)
				if !ok {
					t.Fatal("Expected binary expression")
				}
				if binExpr.Op != "LIKE" {
					t.Errorf("Expected LIKE operator, got %s", binExpr.Op)
				}
			},
		},
		{
			name:  "Null value parameter",
			input: "CUSTOMER.UPDATE phone=null",
			check: func(t *testing.T, cmd *mdwast.Command) {
				phone, exists := cmd.Parameters["phone"]
				if !exists {
					t.Fatal("Expected phone parameter")
				}
				if phone.Type != mdwast.ValueTypeNull {
					t.Errorf("Expected null type, got %v", phone.Type)
				}
				if phone.Value != nil {
					t.Errorf("Expected nil value, got %v", phone.Value)
				}
			},
		},
		{
			name:  "Float parameter",
			input: "PRODUCT.CREATE price=123.45",
			check: func(t *testing.T, cmd *mdwast.Command) {
				price, exists := cmd.Parameters["price"]
				if !exists {
					t.Fatal("Expected price parameter")
				}
				if price.Type != mdwast.ValueTypeNumber {
					t.Errorf("Expected number type, got %v", price.Type)
				}
				if price.Value != float64(123.45) {
					t.Errorf("Expected 123.45, got %v", price.Value)
				}
			},
		},
		{
			name:    "Object expression parameter",
			input:   `USER.CREATE profile={name: "John", age: 30}`,
			wantErr: true,
			errMsg:  "expected value",
		},
		{
			name:    "Empty input",
			input:   "",
			wantErr: true,
			errMsg:  "expected object name",
		},
		{
			name:    "Invalid syntax - missing method",
			input:   "CUSTOMER.",
			wantErr: true,
			errMsg:  "expected method name",
		},
		{
			name:    "Invalid syntax - missing object",
			input:   ".CREATE",
			wantErr: true,
			errMsg:  "expected object name",
		},
		{
			name:    "Invalid filter - unclosed bracket",
			input:   "CUSTOMER[active=true.LIST",
			wantErr: true,
			errMsg:  "expected ']'",
		},
		{
			name:    "Invalid parameter syntax",
			input:   "CUSTOMER.CREATE name=",
			wantErr: true,
			errMsg:  "expected value",
		},
		{
			name:    "Invalid operator",
			input:   "CUSTOMER[age @ 30].LIST",
			wantErr: true,
			errMsg:  "expected ']'",
		},
		{
			name:    "Unexpected token after command",
			input:   "CUSTOMER.CREATE extra",
			wantErr: true,
			errMsg:  "expected '='",
		},
		{
			name:  "Command with abbreviations",
			input: "CUST.CR",
			check: func(t *testing.T, cmd *mdwast.Command) {
				if cmd.Object != "CUST" {
					t.Errorf("Expected object CUST, got %s", cmd.Object)
				}
				if cmd.Method != "CR" {
					t.Errorf("Expected method CR, got %s", cmd.Method)
				}
			},
		},
		{
			name:    "Chained filters",
			input:   "CUSTOMER.LIST | FILTER[age>30] | EXPORT.CSV",
			wantErr: true,
			errMsg:  "expected '.'",
		},
		{
			name:  "Comparison operators",
			input: `CUSTOMER[age>=18 AND age<=65].LIST`,
			check: func(t *testing.T, cmd *mdwast.Command) {
				if cmd.Filter == nil {
					t.Fatal("Expected filter")
				}
				andExpr, ok := cmd.Filter.Condition.(*mdwast.BinaryExpr)
				if !ok || andExpr.Op != "AND" {
					t.Fatal("Expected AND expression")
				}
				// Check left comparison
				left, ok := andExpr.Left.(*mdwast.BinaryExpr)
				if !ok || left.Op != ">=" {
					t.Error("Expected >= on left")
				}
				// Check right comparison
				right, ok := andExpr.Right.(*mdwast.BinaryExpr)
				if !ok || right.Op != "<=" {
					t.Error("Expected <= on right")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd, err := parser.Parse(tt.input)

			if tt.wantErr {
				if err == nil {
					t.Errorf("Expected error, got nil")
				} else if tt.errMsg != "" && !containsString(err.Error(), tt.errMsg) {
					t.Errorf("Expected error containing %q, got %q", tt.errMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
					return
				}
				if cmd == nil {
					t.Fatal("Expected command, got nil")
					return
				}
				if tt.check != nil {
					tt.check(t, cmd)
				}
			}
		})
	}
}

func TestParser_WithRegistry(t *testing.T) {
	// Create registry with test objects
	reg, _ := mdwregistry.NewSimple(mdwregistry.Options{
		Logger:              mdwlog.GetDefault(),
		EnableAbbreviations: true,
	})

	// Register test object
	reg.RegisterObject(&mdwregistry.ObjectDefinition{
		Name:    "CUSTOMER",
		Service: "customer-service",
		Methods: map[string]*mdwregistry.MethodDefinition{
			"CREATE": {
				Name: "CREATE",
				Parameters: map[string]*mdwregistry.ParameterDefinition{
					"name": {Name: "name", Type: "string", Required: true},
					"age":  {Name: "age", Type: "number", Required: false},
				},
			},
		},
	})

	parser, _ := New(Options{
		Logger:   mdwlog.GetDefault(),
		Registry: reg,
	})

	tests := []struct {
		name    string
		input   string
		wantErr bool
		errMsg  string
	}{
		{
			name:    "Valid command with registry",
			input:   `CUSTOMER.CREATE name="John"`,
			wantErr: false,
		},
		{
			name:  "Abbreviation expansion",
			input: "CUST.CR",
			// Note: abbreviations are handled by registry, not parser
		},
		{
			name:    "Unknown object",
			input:   "UNKNOWN.METHOD",
			wantErr: false, // Parser doesn't validate objects, executor does
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd, err := parser.Parse(tt.input)

			if tt.wantErr {
				if err == nil {
					t.Errorf("Expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if cmd == nil {
					t.Fatal("Expected command, got nil")
				}
			}
		})
	}
}

func TestParser_MaxInputLength(t *testing.T) {
	parser, _ := New(Options{
		Logger:         mdwlog.GetDefault(),
		MaxInputLength: 50,
	})

	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{
			name:    "Within limit",
			input:   "CUSTOMER.CREATE name=\"John\"",
			wantErr: false,
		},
		{
			name:    "Exceeds limit",
			input:   "CUSTOMER.CREATE name=\"This is a very long name that exceeds the limit\"",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := parser.Parse(tt.input)
			if tt.wantErr && err == nil {
				t.Error("Expected error for input exceeding max length")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}

func TestParser_DisableChaining(t *testing.T) {
	parser, _ := New(Options{
		Logger:         mdwlog.GetDefault(),
		EnableChaining: false,
	})

	_, err := parser.Parse("CUSTOMER.LIST | EXPORT.CSV")
	if err != nil {
		t.Errorf("Parser should ignore pipe when chaining disabled: %v", err)
	}
}

func TestParseError_Error(t *testing.T) {
	err := &ParseError{
		Message:  "test error",
		Position: 10,
		Line:     2,
		Column:   5,
		Token:    Token{Type: TokenIdentifier, Value: "TEST"},
	}

	expected := "parse error at line 2, column 5: test error (near 'TEST')"
	if err.Error() != expected {
		t.Errorf("Expected error %q, got %q", expected, err.Error())
	}
}

// Helper function for string contains
func containsString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		match := true
		for j := 0; j < len(substr); j++ {
			if s[i+j] != substr[j] {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}

// Benchmarks

func BenchmarkParser_SimpleCommand(b *testing.B) {
	parser, _ := New(Options{
		Logger: mdwlog.GetDefault(),
	})

	input := `CUSTOMER.CREATE name="John Doe" age=30`

	for i := 0; i < b.N; i++ {
		_, err := parser.Parse(input)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkParser_ComplexFilter(b *testing.B) {
	parser, _ := New(Options{
		Logger:         mdwlog.GetDefault(),
		EnableChaining: true,
	})

	input := `CUSTOMER[city="Berlin" AND age>30 AND (status="active" OR premium=true)].LIST | EXPORT.CSV`

	for i := 0; i < b.N; i++ {
		_, err := parser.Parse(input)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkParser_ManyParameters(b *testing.B) {
	parser, _ := New(Options{
		Logger: mdwlog.GetDefault(),
	})

	input := `CUSTOMER.CREATE name="John" email="john@example.com" phone="+123" address="123 Main" city="Berlin" country="Germany" age=30 active=true`

	for i := 0; i < b.N; i++ {
		_, err := parser.Parse(input)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// Test helper to print AST structure (for debugging)
func printAST(cmd *mdwast.Command, indent string) {
	fmt.Printf("%sCommand:\n", indent)
	fmt.Printf("%s  Object: %s\n", indent, cmd.Object)
	fmt.Printf("%s  Method: %s\n", indent, cmd.Method)
	fmt.Printf("%s  ObjectID: %s\n", indent, cmd.ObjectID)
	
	if len(cmd.Parameters) > 0 {
		fmt.Printf("%s  Parameters:\n", indent)
		for name, value := range cmd.Parameters {
			fmt.Printf("%s    %s = %v (%v)\n", indent, name, value.Value, value.Type)
		}
	}
	
	if cmd.Filter != nil {
		fmt.Printf("%s  Filter:\n", indent)
		printExpression(cmd.Filter.Condition, indent+"    ")
	}
	
	if cmd.FieldOp != nil {
		fmt.Printf("%s  FieldOp: %s %s %v\n", indent, cmd.FieldOp.Field, cmd.FieldOp.Op, cmd.FieldOp.Value)
	}
	
	if cmd.Chain != nil {
		fmt.Printf("%s  Chain:\n", indent)
		printAST(cmd.Chain, indent+"    ")
	}
}

func printExpression(expr mdwast.Expr, indent string) {
	switch e := expr.(type) {
	case *mdwast.BinaryExpr:
		fmt.Printf("%sBinary: %s\n", indent, e.Op)
		fmt.Printf("%s  Left:\n", indent)
		printExpression(e.Left, indent+"    ")
		fmt.Printf("%s  Right:\n", indent)
		printExpression(e.Right, indent+"    ")
	case *mdwast.UnaryExpr:
		fmt.Printf("%sUnary: %s\n", indent, e.Op)
		printExpression(e.Expr, indent+"  ")
	case *mdwast.IdentifierExpr:
		fmt.Printf("%sIdentifier: %s\n", indent, e.Name)
	case *mdwast.LiteralExpr:
		fmt.Printf("%sLiteral: %v (%v)\n", indent, e.Value.Value, e.Value.Type)
	case *mdwast.FunctionCallExpr:
		fmt.Printf("%sFunction: %s\n", indent, e.Name)
		for i, arg := range e.Args {
			fmt.Printf("%s  Arg %d:\n", indent, i)
			printExpression(arg, indent+"    ")
		}
	case *mdwast.ArrayExpr:
		fmt.Printf("%sArray:\n", indent)
		for i, elem := range e.Elements {
			fmt.Printf("%s  [%d]:\n", indent, i)
			printExpression(elem, indent+"    ")
		}
	case *mdwast.ObjectExpr:
		fmt.Printf("%sObject:\n", indent)
		for key, value := range e.Fields {
			fmt.Printf("%s  %s:\n", indent, key)
			printExpression(value, indent+"    ")
		}
	}
}