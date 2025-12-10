// File: lexer.go
// Title: TCOL Lexical Analyzer (Tokenizer)
// Description: Implements the lexical analysis phase of TCOL parsing.
//              Converts TCOL command strings into streams of tokens for
//              the parser. Handles all TCOL syntax elements and provides
//              detailed position information for error reporting.
// Author: msto63 with Claude Sonnet 4.0
// Version: v0.1.0
// Created: 2025-01-25
// Modified: 2025-01-25
//
// Change History:
// - 2025-01-25 v0.1.0: Initial lexer implementation

package parser

import (
	"fmt"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	mdwstringx "github.com/msto63/mDW/foundation/utils/stringx"
)

// TokenType represents the type of a lexical token
type TokenType int

const (
	// Special tokens
	TokenEOF TokenType = iota
	TokenIllegal

	// Identifiers and literals
	TokenIdentifier // CUSTOMER, CREATE, field_name
	TokenString     // "string literal"
	TokenNumber     // 123, 123.45
	TokenBoolean    // true, false

	// Operators
	TokenDot       // .
	TokenColon     // :
	TokenEquals    // =
	TokenNotEquals // !=
	TokenLess      // <
	TokenLessEq    // <=
	TokenGreater   // >
	TokenGreaterEq // >=
	TokenAnd       // AND
	TokenOr        // OR
	TokenNot       // NOT
	TokenLike      // LIKE
	TokenIn        // IN

	// Delimiters
	TokenLeftBracket  // [
	TokenRightBracket // ]
	TokenLeftParen    // (
	TokenRightParen   // )
	TokenLeftBrace    // {
	TokenRightBrace   // }
	TokenComma        // ,
	TokenPipe         // |
	TokenSemicolon    // ;

	// Keywords
	TokenNull // null
)

// Token represents a lexical token with position information
type Token struct {
	Type     TokenType // Token type
	Value    string    // Token text
	Position int       // Byte position in input
	Line     int       // Line number (1-based)
	Column   int       // Column number (1-based)
}

// String returns a string representation of the token
func (t Token) String() string {
	switch t.Type {
	case TokenEOF:
		return "EOF"
	case TokenIllegal:
		return fmt.Sprintf("ILLEGAL(%s)", t.Value)
	default:
		return fmt.Sprintf("%s(%s)", t.Type.String(), t.Value)
	}
}

// String returns a string representation of the token type
func (tt TokenType) String() string {
	switch tt {
	case TokenEOF:
		return "EOF"
	case TokenIllegal:
		return "ILLEGAL"
	case TokenIdentifier:
		return "IDENTIFIER"
	case TokenString:
		return "STRING"
	case TokenNumber:
		return "NUMBER"
	case TokenBoolean:
		return "BOOLEAN"
	case TokenDot:
		return "DOT"
	case TokenColon:
		return "COLON"
	case TokenEquals:
		return "EQUALS"
	case TokenNotEquals:
		return "NOT_EQUALS"
	case TokenLess:
		return "LESS"
	case TokenLessEq:
		return "LESS_EQ"
	case TokenGreater:
		return "GREATER"
	case TokenGreaterEq:
		return "GREATER_EQ"
	case TokenAnd:
		return "AND"
	case TokenOr:
		return "OR"
	case TokenNot:
		return "NOT"
	case TokenLike:
		return "LIKE"
	case TokenIn:
		return "IN"
	case TokenLeftBracket:
		return "LEFT_BRACKET"
	case TokenRightBracket:
		return "RIGHT_BRACKET"
	case TokenLeftParen:
		return "LEFT_PAREN"
	case TokenRightParen:
		return "RIGHT_PAREN"
	case TokenLeftBrace:
		return "LEFT_BRACE"
	case TokenRightBrace:
		return "RIGHT_BRACE"
	case TokenComma:
		return "COMMA"
	case TokenPipe:
		return "PIPE"
	case TokenSemicolon:
		return "SEMICOLON"
	case TokenNull:
		return "NULL"
	default:
		return "UNKNOWN"
	}
}

// Lexer performs lexical analysis of TCOL input
type Lexer struct {
	input    string // Input string
	position int    // Current position in input (points to current char)
	readPos  int    // Current reading position (after current char)
	ch       byte   // Current char under examination
	line     int    // Current line number (1-based)
	column   int    // Current column number (1-based)
}

// NewLexer creates a new lexer for the given input
func NewLexer(input string) *Lexer {
	l := &Lexer{
		input:  input,
		line:   1,
		column: 0,
	}
	l.readChar() // Initialize first character
	return l
}

// NextToken returns the next token from the input
func (l *Lexer) NextToken() Token {
	var tok Token

	l.skipWhitespace()

	// Save current position for token
	pos := l.position
	line := l.line
	column := l.column

	switch l.ch {
	case '=':
		if l.peekChar() == '=' {
			ch := l.ch
			l.readChar()
			tok = Token{Type: TokenEquals, Value: string(ch) + string(l.ch), Position: pos, Line: line, Column: column}
		} else {
			tok = newToken(TokenEquals, l.ch, pos, line, column)
		}
	case '!':
		if l.peekChar() == '=' {
			ch := l.ch
			l.readChar()
			tok = Token{Type: TokenNotEquals, Value: string(ch) + string(l.ch), Position: pos, Line: line, Column: column}
		} else {
			tok = newToken(TokenIllegal, l.ch, pos, line, column)
		}
	case '<':
		if l.peekChar() == '=' {
			ch := l.ch
			l.readChar()
			tok = Token{Type: TokenLessEq, Value: string(ch) + string(l.ch), Position: pos, Line: line, Column: column}
		} else {
			tok = newToken(TokenLess, l.ch, pos, line, column)
		}
	case '>':
		if l.peekChar() == '=' {
			ch := l.ch
			l.readChar()
			tok = Token{Type: TokenGreaterEq, Value: string(ch) + string(l.ch), Position: pos, Line: line, Column: column}
		} else {
			tok = newToken(TokenGreater, l.ch, pos, line, column)
		}
	case '.':
		tok = newToken(TokenDot, l.ch, pos, line, column)
	case ':':
		tok = newToken(TokenColon, l.ch, pos, line, column)
	case '[':
		tok = newToken(TokenLeftBracket, l.ch, pos, line, column)
	case ']':
		tok = newToken(TokenRightBracket, l.ch, pos, line, column)
	case '(':
		tok = newToken(TokenLeftParen, l.ch, pos, line, column)
	case ')':
		tok = newToken(TokenRightParen, l.ch, pos, line, column)
	case '{':
		tok = newToken(TokenLeftBrace, l.ch, pos, line, column)
	case '}':
		tok = newToken(TokenRightBrace, l.ch, pos, line, column)
	case ',':
		tok = newToken(TokenComma, l.ch, pos, line, column)
	case '|':
		tok = newToken(TokenPipe, l.ch, pos, line, column)
	case ';':
		tok = newToken(TokenSemicolon, l.ch, pos, line, column)
	case '"':
		tok.Type = TokenString
		tok.Value = l.readString()
		tok.Position = pos
		tok.Line = line
		tok.Column = column
	case '\'':
		tok.Type = TokenString
		tok.Value = l.readSingleQuotedString()
		tok.Position = pos
		tok.Line = line
		tok.Column = column
	case 0:
		tok = Token{Type: TokenEOF, Value: "", Position: pos, Line: line, Column: column}
	default:
		if isLetter(l.ch) {
			tok.Position = pos
			tok.Line = line
			tok.Column = column
			tok.Value = l.readIdentifier()
			tok.Type = lookupIdent(tok.Value)
			return tok // Early return to avoid readChar()
		} else if isDigit(l.ch) {
			tok.Type = TokenNumber
			tok.Value = l.readNumber()
			tok.Position = pos
			tok.Line = line
			tok.Column = column
			return tok // Early return to avoid readChar()
		} else {
			tok = newToken(TokenIllegal, l.ch, pos, line, column)
		}
	}

	l.readChar()
	return tok
}

// Tokenize returns all tokens from the input as a slice
func (l *Lexer) Tokenize() ([]Token, error) {
	var tokens []Token

	for {
		tok := l.NextToken()
		tokens = append(tokens, tok)

		if tok.Type == TokenEOF {
			break
		}

		if tok.Type == TokenIllegal {
			return tokens, fmt.Errorf("illegal character '%s' at line %d, column %d (position %d)", 
				tok.Value, tok.Line, tok.Column, tok.Position)
		}
	}

	return tokens, nil
}

// readChar reads the next character and advances position
func (l *Lexer) readChar() {
	if l.readPos >= len(l.input) {
		l.ch = 0 // ASCII NUL character represents EOF
	} else {
		l.ch = l.input[l.readPos]
	}

	l.position = l.readPos
	l.readPos++

	// Update line and column tracking
	if l.ch == '\n' {
		l.line++
		l.column = 0
	} else {
		l.column++
	}
}

// peekChar returns the next character without advancing position
func (l *Lexer) peekChar() byte {
	if l.readPos >= len(l.input) {
		return 0
	}
	return l.input[l.readPos]
}

// readIdentifier reads an identifier (letters, digits, underscores, hyphens)
func (l *Lexer) readIdentifier() string {
	start := l.position
	for isLetter(l.ch) || isDigit(l.ch) || l.ch == '_' || l.ch == '-' {
		l.readChar()
	}
	return l.input[start:l.position]
}

// readNumber reads a numeric literal (integer or float)
func (l *Lexer) readNumber() string {
	start := l.position
	
	// Read integer part
	for isDigit(l.ch) {
		l.readChar()
	}
	
	// Check for decimal point
	if l.ch == '.' && isDigit(l.peekChar()) {
		l.readChar() // consume '.'
		for isDigit(l.ch) {
			l.readChar()
		}
	}
	
	return l.input[start:l.position]
}

// readString reads a double-quoted string literal
func (l *Lexer) readString() string {
	start := l.position + 1 // Skip opening quote
	
	for {
		l.readChar()
		if l.ch == '"' || l.ch == 0 {
			break
		}
		// Handle escape sequences
		if l.ch == '\\' {
			l.readChar() // Skip escape character
		}
	}
	
	return l.input[start:l.position]
}

// readSingleQuotedString reads a single-quoted string literal
func (l *Lexer) readSingleQuotedString() string {
	start := l.position + 1 // Skip opening quote
	
	for {
		l.readChar()
		if l.ch == '\'' || l.ch == 0 {
			break
		}
		// Handle escape sequences
		if l.ch == '\\' {
			l.readChar() // Skip escape character
		}
	}
	
	return l.input[start:l.position]
}

// skipWhitespace skips whitespace characters
func (l *Lexer) skipWhitespace() {
	for l.ch == ' ' || l.ch == '\t' || l.ch == '\n' || l.ch == '\r' {
		l.readChar()
	}
}

// Utility functions

// newToken creates a new token with the given parameters
func newToken(tokenType TokenType, ch byte, pos, line, column int) Token {
	return Token{
		Type:     tokenType,
		Value:    string(ch),
		Position: pos,
		Line:     line,
		Column:   column,
	}
}

// isLetter checks if the character is a letter (including Unicode)
func isLetter(ch byte) bool {
	return 'a' <= ch && ch <= 'z' || 'A' <= ch && ch <= 'Z' || ch == '_' || ch > 127
}

// isDigit checks if the character is a digit
func isDigit(ch byte) bool {
	return '0' <= ch && ch <= '9'
}

// Keywords map for identifier lookup
var keywords = map[string]TokenType{
	"AND":   TokenAnd,
	"OR":    TokenOr,
	"NOT":   TokenNot,
	"LIKE":  TokenLike,
	"IN":    TokenIn,
	"true":  TokenBoolean,
	"false": TokenBoolean,
	"null":  TokenNull,
	"NULL":  TokenNull,
}

// lookupIdent determines if an identifier is a keyword or regular identifier
func lookupIdent(ident string) TokenType {
	// Check exact match first
	if tok, ok := keywords[ident]; ok {
		return tok
	}
	
	// Check case-insensitive match for keywords
	upperIdent := strings.ToUpper(ident)
	if tok, ok := keywords[upperIdent]; ok {
		return tok
	}
	
	return TokenIdentifier
}

// IsValidNumber checks if a string represents a valid number
func IsValidNumber(s string) bool {
	if mdwstringx.IsBlank(s) {
		return false
	}
	
	_, err := strconv.ParseFloat(s, 64)
	return err == nil
}

// IsValidIdentifier checks if a string is a valid TCOL identifier
func IsValidIdentifier(s string) bool {
	if mdwstringx.IsBlank(s) {
		return false
	}
	
	// Must start with letter or underscore
	r, _ := utf8.DecodeRuneInString(s)
	if !unicode.IsLetter(r) && r != '_' {
		return false
	}
	
	// Rest can be letters, digits, underscores, or hyphens
	for _, r := range s[1:] {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '_' && r != '-' {
			return false
		}
	}
	
	return true
}

// IsKeyword checks if a string is a TCOL keyword
func IsKeyword(s string) bool {
	_, isKeyword := keywords[strings.ToUpper(s)]
	return isKeyword
}

// TokenizeInput is a convenience function that tokenizes input and returns tokens or error
func TokenizeInput(input string) ([]Token, error) {
	lexer := NewLexer(input)
	return lexer.Tokenize()
}