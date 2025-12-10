// File: lexer_test.go
// Title: TCOL Lexer Unit Tests
// Description: Comprehensive unit tests for the TCOL lexical analyzer.
//              Tests cover tokenization of all TCOL syntax elements,
//              error handling, position tracking, and edge cases.
// Author: msto63 with Claude Sonnet 4.0
// Version: v0.1.0
// Created: 2025-01-25
// Modified: 2025-01-25
//
// Change History:
// - 2025-01-25 v0.1.0: Initial comprehensive test suite

package parser

import (
	"testing"
)

func TestLexer_NextToken(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []Token
	}{
		{
			name:  "Simple command",
			input: "CUSTOMER.CREATE",
			expected: []Token{
				{Type: TokenIdentifier, Value: "CUSTOMER", Position: 0, Line: 1, Column: 1},
				{Type: TokenDot, Value: ".", Position: 8, Line: 1, Column: 9},
				{Type: TokenIdentifier, Value: "CREATE", Position: 9, Line: 1, Column: 10},
				{Type: TokenEOF, Value: "", Position: 15, Line: 1, Column: 16},
			},
		},
		{
			name:  "Command with parameters",
			input: `CUSTOMER.CREATE name="John Doe" age=30`,
			expected: []Token{
				{Type: TokenIdentifier, Value: "CUSTOMER", Position: 0, Line: 1, Column: 1},
				{Type: TokenDot, Value: ".", Position: 8, Line: 1, Column: 9},
				{Type: TokenIdentifier, Value: "CREATE", Position: 9, Line: 1, Column: 10},
				{Type: TokenIdentifier, Value: "name", Position: 16, Line: 1, Column: 17},
				{Type: TokenEquals, Value: "=", Position: 20, Line: 1, Column: 21},
				{Type: TokenString, Value: "John Doe", Position: 21, Line: 1, Column: 22},
				{Type: TokenIdentifier, Value: "age", Position: 32, Line: 1, Column: 33},
				{Type: TokenEquals, Value: "=", Position: 35, Line: 1, Column: 36},
				{Type: TokenNumber, Value: "30", Position: 36, Line: 1, Column: 37},
				{Type: TokenEOF, Value: "", Position: 38, Line: 1, Column: 39},
			},
		},
		{
			name:  "Object access with ID",
			input: "CUSTOMER:12345",
			expected: []Token{
				{Type: TokenIdentifier, Value: "CUSTOMER", Position: 0, Line: 1, Column: 1},
				{Type: TokenColon, Value: ":", Position: 8, Line: 1, Column: 9},
				{Type: TokenNumber, Value: "12345", Position: 9, Line: 1, Column: 10},
				{Type: TokenEOF, Value: "", Position: 14, Line: 1, Column: 15},
			},
		},
		{
			name:  "Field operation",
			input: `CUSTOMER:123:email="new@example.com"`,
			expected: []Token{
				{Type: TokenIdentifier, Value: "CUSTOMER", Position: 0, Line: 1, Column: 1},
				{Type: TokenColon, Value: ":", Position: 8, Line: 1, Column: 9},
				{Type: TokenNumber, Value: "123", Position: 9, Line: 1, Column: 10},
				{Type: TokenColon, Value: ":", Position: 12, Line: 1, Column: 13},
				{Type: TokenIdentifier, Value: "email", Position: 13, Line: 1, Column: 14},
				{Type: TokenEquals, Value: "=", Position: 18, Line: 1, Column: 19},
				{Type: TokenString, Value: "new@example.com", Position: 19, Line: 1, Column: 20},
				{Type: TokenEOF, Value: "", Position: 36, Line: 1, Column: 37},
			},
		},
		{
			name:  "Filter expression",
			input: `CUSTOMER[city="Berlin" AND age>30].LIST`,
			expected: []Token{
				{Type: TokenIdentifier, Value: "CUSTOMER", Position: 0, Line: 1, Column: 1},
				{Type: TokenLeftBracket, Value: "[", Position: 8, Line: 1, Column: 9},
				{Type: TokenIdentifier, Value: "city", Position: 9, Line: 1, Column: 10},
				{Type: TokenEquals, Value: "=", Position: 13, Line: 1, Column: 14},
				{Type: TokenString, Value: "Berlin", Position: 14, Line: 1, Column: 15},
				{Type: TokenAnd, Value: "AND", Position: 23, Line: 1, Column: 24},
				{Type: TokenIdentifier, Value: "age", Position: 27, Line: 1, Column: 28},
				{Type: TokenGreater, Value: ">", Position: 30, Line: 1, Column: 31},
				{Type: TokenNumber, Value: "30", Position: 31, Line: 1, Column: 32},
				{Type: TokenRightBracket, Value: "]", Position: 33, Line: 1, Column: 34},
				{Type: TokenDot, Value: ".", Position: 34, Line: 1, Column: 35},
				{Type: TokenIdentifier, Value: "LIST", Position: 35, Line: 1, Column: 36},
				{Type: TokenEOF, Value: "", Position: 39, Line: 1, Column: 40},
			},
		},
		{
			name:  "Command chaining",
			input: "CUSTOMER.LIST | EXPORT.CSV",
			expected: []Token{
				{Type: TokenIdentifier, Value: "CUSTOMER", Position: 0, Line: 1, Column: 1},
				{Type: TokenDot, Value: ".", Position: 8, Line: 1, Column: 9},
				{Type: TokenIdentifier, Value: "LIST", Position: 9, Line: 1, Column: 10},
				{Type: TokenPipe, Value: "|", Position: 14, Line: 1, Column: 15},
				{Type: TokenIdentifier, Value: "EXPORT", Position: 16, Line: 1, Column: 17},
				{Type: TokenDot, Value: ".", Position: 22, Line: 1, Column: 23},
				{Type: TokenIdentifier, Value: "CSV", Position: 23, Line: 1, Column: 24},
				{Type: TokenEOF, Value: "", Position: 26, Line: 1, Column: 27},
			},
		},
		{
			name:  "Boolean values",
			input: "CUSTOMER.CREATE active=true verified=false",
			expected: []Token{
				{Type: TokenIdentifier, Value: "CUSTOMER", Position: 0, Line: 1, Column: 1},
				{Type: TokenDot, Value: ".", Position: 8, Line: 1, Column: 9},
				{Type: TokenIdentifier, Value: "CREATE", Position: 9, Line: 1, Column: 10},
				{Type: TokenIdentifier, Value: "active", Position: 16, Line: 1, Column: 17},
				{Type: TokenEquals, Value: "=", Position: 22, Line: 1, Column: 23},
				{Type: TokenBoolean, Value: "true", Position: 23, Line: 1, Column: 24},
				{Type: TokenIdentifier, Value: "verified", Position: 28, Line: 1, Column: 29},
				{Type: TokenEquals, Value: "=", Position: 36, Line: 1, Column: 37},
				{Type: TokenBoolean, Value: "false", Position: 37, Line: 1, Column: 38},
				{Type: TokenEOF, Value: "", Position: 42, Line: 1, Column: 43},
			},
		},
		{
			name:  "Null value",
			input: "CUSTOMER.UPDATE phone=null",
			expected: []Token{
				{Type: TokenIdentifier, Value: "CUSTOMER", Position: 0, Line: 1, Column: 1},
				{Type: TokenDot, Value: ".", Position: 8, Line: 1, Column: 9},
				{Type: TokenIdentifier, Value: "UPDATE", Position: 9, Line: 1, Column: 10},
				{Type: TokenIdentifier, Value: "phone", Position: 16, Line: 1, Column: 17},
				{Type: TokenEquals, Value: "=", Position: 21, Line: 1, Column: 22},
				{Type: TokenNull, Value: "null", Position: 22, Line: 1, Column: 23},
				{Type: TokenEOF, Value: "", Position: 26, Line: 1, Column: 27},
			},
		},
		{
			name:  "Comparison operators",
			input: "age>=18 price<=100 count!=0",
			expected: []Token{
				{Type: TokenIdentifier, Value: "age", Position: 0, Line: 1, Column: 1},
				{Type: TokenGreaterEq, Value: ">=", Position: 3, Line: 1, Column: 4},
				{Type: TokenNumber, Value: "18", Position: 5, Line: 1, Column: 6},
				{Type: TokenIdentifier, Value: "price", Position: 8, Line: 1, Column: 9},
				{Type: TokenLessEq, Value: "<=", Position: 13, Line: 1, Column: 14},
				{Type: TokenNumber, Value: "100", Position: 15, Line: 1, Column: 16},
				{Type: TokenIdentifier, Value: "count", Position: 19, Line: 1, Column: 20},
				{Type: TokenNotEquals, Value: "!=", Position: 24, Line: 1, Column: 25},
				{Type: TokenNumber, Value: "0", Position: 26, Line: 1, Column: 27},
				{Type: TokenEOF, Value: "", Position: 27, Line: 1, Column: 28},
			},
		},
		{
			name:  "Logical operators",
			input: "active=true AND age>18 OR status=premium",
			expected: []Token{
				{Type: TokenIdentifier, Value: "active", Position: 0, Line: 1, Column: 1},
				{Type: TokenEquals, Value: "=", Position: 6, Line: 1, Column: 7},
				{Type: TokenBoolean, Value: "true", Position: 7, Line: 1, Column: 8},
				{Type: TokenAnd, Value: "AND", Position: 12, Line: 1, Column: 13},
				{Type: TokenIdentifier, Value: "age", Position: 16, Line: 1, Column: 17},
				{Type: TokenGreater, Value: ">", Position: 19, Line: 1, Column: 20},
				{Type: TokenNumber, Value: "18", Position: 20, Line: 1, Column: 21},
				{Type: TokenOr, Value: "OR", Position: 23, Line: 1, Column: 24},
				{Type: TokenIdentifier, Value: "status", Position: 26, Line: 1, Column: 27},
				{Type: TokenEquals, Value: "=", Position: 32, Line: 1, Column: 33},
				{Type: TokenIdentifier, Value: "premium", Position: 33, Line: 1, Column: 34},
				{Type: TokenEOF, Value: "", Position: 40, Line: 1, Column: 41},
			},
		},
		{
			name:  "Float numbers",
			input: "price=123.45 discount=0.15",
			expected: []Token{
				{Type: TokenIdentifier, Value: "price", Position: 0, Line: 1, Column: 1},
				{Type: TokenEquals, Value: "=", Position: 5, Line: 1, Column: 6},
				{Type: TokenNumber, Value: "123.45", Position: 6, Line: 1, Column: 7},
				{Type: TokenIdentifier, Value: "discount", Position: 13, Line: 1, Column: 14},
				{Type: TokenEquals, Value: "=", Position: 21, Line: 1, Column: 22},
				{Type: TokenNumber, Value: "0.15", Position: 22, Line: 1, Column: 23},
				{Type: TokenEOF, Value: "", Position: 26, Line: 1, Column: 27},
			},
		},
		{
			name:  "Identifiers with hyphens and underscores",
			input: "user-name=john_doe item-count=10",
			expected: []Token{
				{Type: TokenIdentifier, Value: "user-name", Position: 0, Line: 1, Column: 1},
				{Type: TokenEquals, Value: "=", Position: 9, Line: 1, Column: 10},
				{Type: TokenIdentifier, Value: "john_doe", Position: 10, Line: 1, Column: 11},
				{Type: TokenIdentifier, Value: "item-count", Position: 19, Line: 1, Column: 20},
				{Type: TokenEquals, Value: "=", Position: 29, Line: 1, Column: 30},
				{Type: TokenNumber, Value: "10", Position: 30, Line: 1, Column: 31},
				{Type: TokenEOF, Value: "", Position: 32, Line: 1, Column: 33},
			},
		},
		{
			name:  "Single quoted strings",
			input: `name='John Doe'`,
			expected: []Token{
				{Type: TokenIdentifier, Value: "name", Position: 0, Line: 1, Column: 1},
				{Type: TokenEquals, Value: "=", Position: 4, Line: 1, Column: 5},
				{Type: TokenString, Value: "John Doe", Position: 5, Line: 1, Column: 6},
				{Type: TokenEOF, Value: "", Position: 15, Line: 1, Column: 16},
			},
		},
		{
			name:  "Parentheses for grouping",
			input: "(age>18 AND active=true)",
			expected: []Token{
				{Type: TokenLeftParen, Value: "(", Position: 0, Line: 1, Column: 1},
				{Type: TokenIdentifier, Value: "age", Position: 1, Line: 1, Column: 2},
				{Type: TokenGreater, Value: ">", Position: 4, Line: 1, Column: 5},
				{Type: TokenNumber, Value: "18", Position: 5, Line: 1, Column: 6},
				{Type: TokenAnd, Value: "AND", Position: 8, Line: 1, Column: 9},
				{Type: TokenIdentifier, Value: "active", Position: 12, Line: 1, Column: 13},
				{Type: TokenEquals, Value: "=", Position: 18, Line: 1, Column: 19},
				{Type: TokenBoolean, Value: "true", Position: 19, Line: 1, Column: 20},
				{Type: TokenRightParen, Value: ")", Position: 23, Line: 1, Column: 24},
				{Type: TokenEOF, Value: "", Position: 24, Line: 1, Column: 25},
			},
		},
		{
			name:  "LIKE operator",
			input: `name LIKE "John%"`,
			expected: []Token{
				{Type: TokenIdentifier, Value: "name", Position: 0, Line: 1, Column: 1},
				{Type: TokenLike, Value: "LIKE", Position: 5, Line: 1, Column: 6},
				{Type: TokenString, Value: "John%", Position: 10, Line: 1, Column: 11},
				{Type: TokenEOF, Value: "", Position: 17, Line: 1, Column: 18},
			},
		},
		{
			name:  "IN operator",
			input: `status IN ["active", "pending"]`,
			expected: []Token{
				{Type: TokenIdentifier, Value: "status", Position: 0, Line: 1, Column: 1},
				{Type: TokenIn, Value: "IN", Position: 7, Line: 1, Column: 8},
				{Type: TokenLeftBracket, Value: "[", Position: 10, Line: 1, Column: 11},
				{Type: TokenString, Value: "active", Position: 11, Line: 1, Column: 12},
				{Type: TokenComma, Value: ",", Position: 19, Line: 1, Column: 20},
				{Type: TokenString, Value: "pending", Position: 21, Line: 1, Column: 22},
				{Type: TokenRightBracket, Value: "]", Position: 30, Line: 1, Column: 31},
				{Type: TokenEOF, Value: "", Position: 31, Line: 1, Column: 32},
			},
		},
		{
			name:  "NOT operator",
			input: "NOT active",
			expected: []Token{
				{Type: TokenNot, Value: "NOT", Position: 0, Line: 1, Column: 1},
				{Type: TokenIdentifier, Value: "active", Position: 4, Line: 1, Column: 5},
				{Type: TokenEOF, Value: "", Position: 10, Line: 1, Column: 11},
			},
		},
		{
			name:  "Empty input",
			input: "",
			expected: []Token{
				{Type: TokenEOF, Value: "", Position: 0, Line: 1, Column: 1},
			},
		},
		{
			name:  "Whitespace handling",
			input: "  CUSTOMER  .  CREATE  ",
			expected: []Token{
				{Type: TokenIdentifier, Value: "CUSTOMER", Position: 2, Line: 1, Column: 3},
				{Type: TokenDot, Value: ".", Position: 12, Line: 1, Column: 13},
				{Type: TokenIdentifier, Value: "CREATE", Position: 15, Line: 1, Column: 16},
				{Type: TokenEOF, Value: "", Position: 23, Line: 1, Column: 24},
			},
		},
		{
			name:  "Multiline input",
			input: "CUSTOMER.CREATE\nname=\"John\"",
			expected: []Token{
				{Type: TokenIdentifier, Value: "CUSTOMER", Position: 0, Line: 1, Column: 1},
				{Type: TokenDot, Value: ".", Position: 8, Line: 1, Column: 9},
				{Type: TokenIdentifier, Value: "CREATE", Position: 9, Line: 1, Column: 10},
				{Type: TokenIdentifier, Value: "name", Position: 16, Line: 2, Column: 1},
				{Type: TokenEquals, Value: "=", Position: 20, Line: 2, Column: 5},
				{Type: TokenString, Value: "John", Position: 21, Line: 2, Column: 6},
				{Type: TokenEOF, Value: "", Position: 27, Line: 2, Column: 12},
			},
		},
		{
			name:  "Object literal",
			input: `{name: "John", age: 30}`,
			expected: []Token{
				{Type: TokenLeftBrace, Value: "{", Position: 0, Line: 1, Column: 1},
				{Type: TokenIdentifier, Value: "name", Position: 1, Line: 1, Column: 2},
				{Type: TokenColon, Value: ":", Position: 5, Line: 1, Column: 6},
				{Type: TokenString, Value: "John", Position: 7, Line: 1, Column: 8},
				{Type: TokenComma, Value: ",", Position: 13, Line: 1, Column: 14},
				{Type: TokenIdentifier, Value: "age", Position: 15, Line: 1, Column: 16},
				{Type: TokenColon, Value: ":", Position: 18, Line: 1, Column: 19},
				{Type: TokenNumber, Value: "30", Position: 20, Line: 1, Column: 21},
				{Type: TokenRightBrace, Value: "}", Position: 22, Line: 1, Column: 23},
				{Type: TokenEOF, Value: "", Position: 23, Line: 1, Column: 24},
			},
		},
		{
			name:  "Case insensitive keywords",
			input: "name like \"john\" and age>18 or status=active",
			expected: []Token{
				{Type: TokenIdentifier, Value: "name", Position: 0, Line: 1, Column: 1},
				{Type: TokenLike, Value: "like", Position: 5, Line: 1, Column: 6},
				{Type: TokenString, Value: "john", Position: 10, Line: 1, Column: 11},
				{Type: TokenAnd, Value: "and", Position: 17, Line: 1, Column: 18},
				{Type: TokenIdentifier, Value: "age", Position: 21, Line: 1, Column: 22},
				{Type: TokenGreater, Value: ">", Position: 24, Line: 1, Column: 25},
				{Type: TokenNumber, Value: "18", Position: 25, Line: 1, Column: 26},
				{Type: TokenOr, Value: "or", Position: 28, Line: 1, Column: 29},
				{Type: TokenIdentifier, Value: "status", Position: 31, Line: 1, Column: 32},
				{Type: TokenEquals, Value: "=", Position: 37, Line: 1, Column: 38},
				{Type: TokenIdentifier, Value: "active", Position: 38, Line: 1, Column: 39},
				{Type: TokenEOF, Value: "", Position: 44, Line: 1, Column: 45},
			},
		},
		{
			name:  "Escaped strings",
			input: `name="John \"The Boss\" Doe"`,
			expected: []Token{
				{Type: TokenIdentifier, Value: "name", Position: 0, Line: 1, Column: 1},
				{Type: TokenEquals, Value: "=", Position: 4, Line: 1, Column: 5},
				{Type: TokenString, Value: `John \"The Boss\" Doe`, Position: 5, Line: 1, Column: 6},
				{Type: TokenEOF, Value: "", Position: 28, Line: 1, Column: 29},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lexer := NewLexer(tt.input)
			
			for i, expected := range tt.expected {
				token := lexer.NextToken()
				
				if token.Type != expected.Type {
					t.Errorf("Token %d: expected type %s, got %s", i, expected.Type.String(), token.Type.String())
				}
				
				if token.Value != expected.Value {
					t.Errorf("Token %d: expected value %q, got %q", i, expected.Value, token.Value)
				}
				
				if token.Position != expected.Position {
					t.Errorf("Token %d: expected position %d, got %d", i, expected.Position, token.Position)
				}
				
				if token.Line != expected.Line {
					t.Errorf("Token %d: expected line %d, got %d", i, expected.Line, token.Line)
				}
				
				if token.Column != expected.Column {
					t.Errorf("Token %d: expected column %d, got %d", i, expected.Column, token.Column)
				}
			}
		})
	}
}

func TestLexer_Tokenize(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantErr   bool
		errMsg    string
		tokenLen  int
	}{
		{
			name:     "Valid command",
			input:    "CUSTOMER.CREATE",
			wantErr:  false,
			tokenLen: 4, // CUSTOMER, ., CREATE, EOF
		},
		{
			name:     "Illegal character",
			input:    "CUSTOMER@CREATE",
			wantErr:  true,
			errMsg:   "illegal character",
		},
		{
			name:     "Another illegal character",
			input:    "CUSTOMER#123",
			wantErr:  true,
			errMsg:   "illegal character",
		},
		{
			name:     "Empty string",
			input:    "",
			wantErr:  false,
			tokenLen: 1, // EOF
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lexer := NewLexer(tt.input)
			tokens, err := lexer.Tokenize()

			if tt.wantErr {
				if err == nil {
					t.Errorf("Expected error, got nil")
				} else if tt.errMsg != "" && !contains(err.Error(), tt.errMsg) {
					t.Errorf("Expected error containing %q, got %q", tt.errMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if len(tokens) != tt.tokenLen {
					t.Errorf("Expected %d tokens, got %d", tt.tokenLen, len(tokens))
				}
			}
		})
	}
}

func TestTokenType_String(t *testing.T) {
	tests := []struct {
		tokenType TokenType
		expected  string
	}{
		{TokenEOF, "EOF"},
		{TokenIllegal, "ILLEGAL"},
		{TokenIdentifier, "IDENTIFIER"},
		{TokenString, "STRING"},
		{TokenNumber, "NUMBER"},
		{TokenBoolean, "BOOLEAN"},
		{TokenDot, "DOT"},
		{TokenColon, "COLON"},
		{TokenEquals, "EQUALS"},
		{TokenNotEquals, "NOT_EQUALS"},
		{TokenLess, "LESS"},
		{TokenLessEq, "LESS_EQ"},
		{TokenGreater, "GREATER"},
		{TokenGreaterEq, "GREATER_EQ"},
		{TokenAnd, "AND"},
		{TokenOr, "OR"},
		{TokenNot, "NOT"},
		{TokenLike, "LIKE"},
		{TokenIn, "IN"},
		{TokenLeftBracket, "LEFT_BRACKET"},
		{TokenRightBracket, "RIGHT_BRACKET"},
		{TokenLeftParen, "LEFT_PAREN"},
		{TokenRightParen, "RIGHT_PAREN"},
		{TokenLeftBrace, "LEFT_BRACE"},
		{TokenRightBrace, "RIGHT_BRACE"},
		{TokenComma, "COMMA"},
		{TokenPipe, "PIPE"},
		{TokenSemicolon, "SEMICOLON"},
		{TokenNull, "NULL"},
		{TokenType(999), "UNKNOWN"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := tt.tokenType.String()
			if result != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestToken_String(t *testing.T) {
	tests := []struct {
		token    Token
		expected string
	}{
		{
			Token{Type: TokenEOF, Value: ""},
			"EOF",
		},
		{
			Token{Type: TokenIllegal, Value: "@"},
			"ILLEGAL(@)",
		},
		{
			Token{Type: TokenIdentifier, Value: "CUSTOMER"},
			"IDENTIFIER(CUSTOMER)",
		},
		{
			Token{Type: TokenString, Value: "hello"},
			"STRING(hello)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := tt.token.String()
			if result != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestIsValidNumber(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"123", true},
		{"123.45", true},
		{"0", true},
		{"0.0", true},
		{"-123", true},
		{"-123.45", true},
		{"+123", true},
		{"", false},
		{"   ", false},
		{"abc", false},
		{"12.34.56", false},
		{"12a", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := IsValidNumber(tt.input)
			if result != tt.expected {
				t.Errorf("IsValidNumber(%q) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestIsValidIdentifier(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"customer", true},
		{"CUSTOMER", true},
		{"customer_name", true},
		{"customer-name", true},
		{"_private", true},
		{"item123", true},
		{"", false},
		{"   ", false},
		{"123item", false},
		{"-item", false},
		{"item@name", false},
		{"item name", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := IsValidIdentifier(tt.input)
			if result != tt.expected {
				t.Errorf("IsValidIdentifier(%q) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestIsKeyword(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"AND", true},
		{"and", true},
		{"OR", true},
		{"or", true},
		{"NOT", true},
		{"not", true},
		{"LIKE", true},
		{"like", true},
		{"IN", true},
		{"in", true},
		{"true", true},
		{"false", true},
		{"null", true},
		{"NULL", true},
		{"customer", false},
		{"CUSTOMER", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := IsKeyword(tt.input)
			if result != tt.expected {
				t.Errorf("IsKeyword(%q) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestTokenizeInput(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantErr  bool
		checkLen int
	}{
		{
			name:     "Simple command",
			input:    "CUSTOMER.LIST",
			wantErr:  false,
			checkLen: 4, // CUSTOMER, ., LIST, EOF
		},
		{
			name:    "Invalid character",
			input:   "CUSTOMER@LIST",
			wantErr: true,
		},
		{
			name:     "Complex filter",
			input:    `CUSTOMER[age>18 AND status="active"].LIST`,
			wantErr:  false,
			checkLen: 14,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokens, err := TokenizeInput(tt.input)

			if tt.wantErr {
				if err == nil {
					t.Errorf("Expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if tt.checkLen > 0 && len(tokens) != tt.checkLen {
					t.Errorf("Expected %d tokens, got %d", tt.checkLen, len(tokens))
				}
			}
		})
	}
}

// Helper function
func contains(s, substr string) bool {
	return len(s) >= len(substr) && s[:len(substr)] == substr || 
		len(s) > len(substr) && containsHelper(s[1:], substr)
}

func containsHelper(s, substr string) bool {
	if len(s) < len(substr) {
		return false
	}
	if s[:len(substr)] == substr {
		return true
	}
	return containsHelper(s[1:], substr)
}

// Benchmarks

func BenchmarkLexer_SimpleCommand(b *testing.B) {
	input := "CUSTOMER.CREATE name=\"John Doe\" age=30"
	
	for i := 0; i < b.N; i++ {
		lexer := NewLexer(input)
		for {
			token := lexer.NextToken()
			if token.Type == TokenEOF {
				break
			}
		}
	}
}

func BenchmarkLexer_ComplexFilter(b *testing.B) {
	input := `CUSTOMER[city="Berlin" AND age>30 AND (status="active" OR premium=true)].LIST`
	
	for i := 0; i < b.N; i++ {
		lexer := NewLexer(input)
		for {
			token := lexer.NextToken()
			if token.Type == TokenEOF {
				break
			}
		}
	}
}

func BenchmarkLexer_LongCommand(b *testing.B) {
	input := `CUSTOMER.CREATE name="John Doe" email="john@example.com" phone="+1234567890" address="123 Main St" city="Berlin" country="Germany" age=30 active=true verified=false`
	
	for i := 0; i < b.N; i++ {
		lexer := NewLexer(input)
		for {
			token := lexer.NextToken()
			if token.Type == TokenEOF {
				break
			}
		}
	}
}