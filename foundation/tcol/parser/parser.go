// File: parser.go
// Title: TCOL Recursive Descent Parser
// Description: Implements the parsing phase of TCOL command processing.
//              Converts token streams into Abstract Syntax Trees using
//              recursive descent parsing. Handles all TCOL grammar rules
//              with comprehensive error reporting and recovery.
// Author: msto63 with Claude Sonnet 4.0
// Version: v0.1.0
// Created: 2025-01-25
// Modified: 2025-01-25
//
// Change History:
// - 2025-01-25 v0.1.0: Initial parser implementation

package parser

import (
	"fmt"
	"strconv"
	"strings"

	mdwlog "github.com/msto63/mDW/foundation/core/log"
	mdwast "github.com/msto63/mDW/foundation/tcol/ast"
	mdwregistry "github.com/msto63/mDW/foundation/tcol/registry"
)

// Parser implements recursive descent parsing for TCOL
type Parser struct {
	lexer    *Lexer
	current  Token  // Current token
	previous Token  // Previous token
	logger   *mdwlog.Logger
	registry *mdwregistry.Registry
	options  Options
}

// Options configures parser behavior
type Options struct {
	Logger         *mdwlog.Logger
	MaxInputLength int
	EnableChaining bool
	Registry       *mdwregistry.Registry
}

// ParseError represents a parsing error with position information
type ParseError struct {
	Message  string
	Position int
	Line     int
	Column   int
	Token    Token
}

func (pe *ParseError) Error() string {
	return fmt.Sprintf("parse error at line %d, column %d: %s (near '%s')",
		pe.Line, pe.Column, pe.Message, pe.Token.Value)
}

// New creates a new TCOL parser with the given options
func New(opts Options) (*Parser, error) {
	// Set defaults
	if opts.Logger == nil {
		opts.Logger = mdwlog.GetDefault()
	}
	if opts.MaxInputLength == 0 {
		opts.MaxInputLength = 4096
	}

	return &Parser{
		logger:  opts.Logger.WithField("component", "tcol-parser"),
		options: opts,
		registry: opts.Registry,
	}, nil
}

// Parse parses a TCOL command string and returns an AST
func (p *Parser) Parse(input string) (*mdwast.Command, error) {
	// Validate input length
	if len(input) > p.options.MaxInputLength {
		return nil, fmt.Errorf("input exceeds maximum length: %d > %d", 
			len(input), p.options.MaxInputLength)
	}

	// Initialize lexer
	p.lexer = NewLexer(input)
	p.advance() // Load first token

	p.logger.Debug("Starting TCOL parsing", mdwlog.Fields{
		"input":  input,
		"length": len(input),
	})

	// Parse the command
	cmd, err := p.parseCommand()
	if err != nil {
		p.logger.Warn("TCOL parsing failed", mdwlog.Fields{
			"input": input,
			"error": err.Error(),
		})
		return nil, err
	}

	// Ensure we've consumed all input
	if p.current.Type != TokenEOF {
		return nil, p.parseError(fmt.Sprintf("unexpected token after command: %s", p.current.Value))
	}

	p.logger.Debug("TCOL parsing completed successfully", mdwlog.Fields{
		"input":  input,
		"object": cmd.Object,
		"method": cmd.Method,
	})

	return cmd, nil
}

// parseCommand parses a complete TCOL command
func (p *Parser) parseCommand() (*mdwast.Command, error) {
	pos := p.currentPosition()
	
	// Parse the main command
	cmd, err := p.parseMainCommand()
	if err != nil {
		return nil, err
	}
	cmd.Pos = pos

	// Parse optional command chain
	if p.options.EnableChaining && p.current.Type == TokenPipe {
		p.advance() // consume '|'
		
		chainCmd, err := p.parseCommand()
		if err != nil {
			return nil, fmt.Errorf("chain command: %w", err)
		}
		cmd.Chain = chainCmd
	}

	return cmd, nil
}

// parseMainCommand parses the main command (without chaining)
func (p *Parser) parseMainCommand() (*mdwast.Command, error) {
	// Parse object name
	if p.current.Type != TokenIdentifier {
		return nil, p.parseError("expected object name")
	}

	object := p.current.Value
	p.advance()

	// Check for object ID access (OBJECT:ID)
	if p.current.Type == TokenColon {
		return p.parseObjectAccess(object)
	}

	// Check for filter (OBJECT[filter])
	var filter *mdwast.FilterExpr
	if p.current.Type == TokenLeftBracket {
		var err error
		filter, err = p.parseFilter()
		if err != nil {
			return nil, fmt.Errorf("filter: %w", err)
		}
	}

	// Parse method call (OBJECT.METHOD)
	if p.current.Type != TokenDot {
		return nil, p.parseError("expected '.' after object name")
	}
	p.advance() // consume '.'

	if p.current.Type != TokenIdentifier {
		return nil, p.parseError("expected method name after '.'")
	}

	method := p.current.Value
	p.advance()

	// Parse optional parameters
	parameters := make(map[string]mdwast.Value)
	for p.current.Type != TokenEOF && p.current.Type != TokenPipe && p.current.Type != TokenSemicolon {
		param, err := p.parseParameter()
		if err != nil {
			return nil, fmt.Errorf("parameter: %w", err)
		}
		parameters[param.Name] = param.Value
	}

	return &mdwast.Command{
		Object:     object,
		Method:     method,
		Parameters: parameters,
		Filter:     filter,
	}, nil
}

// parseObjectAccess parses object access patterns (OBJECT:ID or OBJECT:ID:field=value)
func (p *Parser) parseObjectAccess(object string) (*mdwast.Command, error) {
	p.advance() // consume ':'

	// Parse object ID
	if p.current.Type != TokenIdentifier && p.current.Type != TokenNumber {
		return nil, p.parseError("expected object ID after ':'")
	}

	objectID := p.current.Value
	p.advance()

	cmd := &mdwast.Command{
		Object:   object,
		ObjectID: objectID,
	}

	// Check for field operation (OBJECT:ID:field=value)
	if p.current.Type == TokenColon {
		p.advance() // consume second ':'

		if p.current.Type != TokenIdentifier {
			return nil, p.parseError("expected field name after ':'")
		}

		fieldName := p.current.Value
		p.advance()

		fieldOp := &mdwast.FieldOperation{
			Field: fieldName,
			Pos:   p.currentPosition(),
		}

		// Check for assignment
		if p.current.Type == TokenEquals {
			p.advance() // consume '='
			
			value, err := p.parseValue()
			if err != nil {
				return nil, fmt.Errorf("field value: %w", err)
			}
			
			fieldOp.Op = "="
			fieldOp.Value = value
		}

		cmd.FieldOp = fieldOp
	}

	return cmd, nil
}

// parseFilter parses a filter expression [condition]
func (p *Parser) parseFilter() (*mdwast.FilterExpr, error) {
	pos := p.currentPosition()
	
	if p.current.Type != TokenLeftBracket {
		return nil, p.parseError("expected '[' to start filter")
	}
	p.advance() // consume '['

	// Parse the filter condition expression
	condition, err := p.parseExpression()
	if err != nil {
		return nil, fmt.Errorf("filter condition: %w", err)
	}

	if p.current.Type != TokenRightBracket {
		return nil, p.parseError("expected ']' to end filter")
	}
	p.advance() // consume ']'

	return &mdwast.FilterExpr{
		Condition: condition,
		Pos:       pos,
	}, nil
}

// parseExpression parses an expression (handles precedence)
func (p *Parser) parseExpression() (mdwast.Expr, error) {
	return p.parseOrExpression()
}

// parseOrExpression parses OR expressions (lowest precedence)
func (p *Parser) parseOrExpression() (mdwast.Expr, error) {
	left, err := p.parseAndExpression()
	if err != nil {
		return nil, err
	}

	for p.current.Type == TokenOr {
		op := p.current.Value
		pos := p.currentPosition()
		p.advance()

		right, err := p.parseAndExpression()
		if err != nil {
			return nil, err
		}

		left = &mdwast.BinaryExpr{
			Left:  left,
			Op:    op,
			Right: right,
			Pos:   pos,
		}
	}

	return left, nil
}

// parseAndExpression parses AND expressions
func (p *Parser) parseAndExpression() (mdwast.Expr, error) {
	left, err := p.parseEqualityExpression()
	if err != nil {
		return nil, err
	}

	for p.current.Type == TokenAnd {
		op := p.current.Value
		pos := p.currentPosition()
		p.advance()

		right, err := p.parseEqualityExpression()
		if err != nil {
			return nil, err
		}

		left = &mdwast.BinaryExpr{
			Left:  left,
			Op:    op,
			Right: right,
			Pos:   pos,
		}
	}

	return left, nil
}

// parseEqualityExpression parses equality expressions (=, !=)
func (p *Parser) parseEqualityExpression() (mdwast.Expr, error) {
	left, err := p.parseComparisonExpression()
	if err != nil {
		return nil, err
	}

	for p.current.Type == TokenEquals || p.current.Type == TokenNotEquals {
		op := p.current.Value
		pos := p.currentPosition()
		p.advance()

		right, err := p.parseComparisonExpression()
		if err != nil {
			return nil, err
		}

		left = &mdwast.BinaryExpr{
			Left:  left,
			Op:    op,
			Right: right,
			Pos:   pos,
		}
	}

	return left, nil
}

// parseComparisonExpression parses comparison expressions (<, >, <=, >=, LIKE, IN)
func (p *Parser) parseComparisonExpression() (mdwast.Expr, error) {
	left, err := p.parseUnaryExpression()
	if err != nil {
		return nil, err
	}

	for p.current.Type == TokenLess || p.current.Type == TokenLessEq ||
		p.current.Type == TokenGreater || p.current.Type == TokenGreaterEq ||
		p.current.Type == TokenLike || p.current.Type == TokenIn {
		
		op := p.current.Value
		pos := p.currentPosition()
		p.advance()

		right, err := p.parseUnaryExpression()
		if err != nil {
			return nil, err
		}

		left = &mdwast.BinaryExpr{
			Left:  left,
			Op:    op,
			Right: right,
			Pos:   pos,
		}
	}

	return left, nil
}

// parseUnaryExpression parses unary expressions (NOT, -)
func (p *Parser) parseUnaryExpression() (mdwast.Expr, error) {
	if p.current.Type == TokenNot {
		op := p.current.Value
		pos := p.currentPosition()
		p.advance()

		expr, err := p.parseUnaryExpression()
		if err != nil {
			return nil, err
		}

		return &mdwast.UnaryExpr{
			Op:   op,
			Expr: expr,
			Pos:  pos,
		}, nil
	}

	return p.parsePrimaryExpression()
}

// parsePrimaryExpression parses primary expressions (literals, identifiers, function calls, parentheses)
func (p *Parser) parsePrimaryExpression() (mdwast.Expr, error) {
	pos := p.currentPosition()

	switch p.current.Type {
	case TokenIdentifier:
		name := p.current.Value
		p.advance()

		// Check for function call
		if p.current.Type == TokenLeftParen {
			return p.parseFunctionCall(name, pos)
		}

		// Regular identifier
		return &mdwast.IdentifierExpr{
			Name: name,
			Pos:  pos,
		}, nil

	case TokenString, TokenNumber, TokenBoolean, TokenNull:
		value, err := p.parseValue()
		if err != nil {
			return nil, err
		}

		return &mdwast.LiteralExpr{
			Value: value,
			Pos:   pos,
		}, nil

	case TokenLeftParen:
		p.advance() // consume '('
		
		expr, err := p.parseExpression()
		if err != nil {
			return nil, err
		}

		if p.current.Type != TokenRightParen {
			return nil, p.parseError("expected ')' after expression")
		}
		p.advance() // consume ')'

		return expr, nil

	case TokenLeftBracket:
		return p.parseArrayExpression()

	case TokenLeftBrace:
		return p.parseObjectExpression()

	default:
		return nil, p.parseError(fmt.Sprintf("unexpected token in expression: %s", p.current.Value))
	}
}

// parseFunctionCall parses a function call expression
func (p *Parser) parseFunctionCall(name string, pos mdwast.Position) (mdwast.Expr, error) {
	p.advance() // consume '('

	var args []mdwast.Expr

	if p.current.Type != TokenRightParen {
		// Parse first argument
		arg, err := p.parseExpression()
		if err != nil {
			return nil, fmt.Errorf("function argument: %w", err)
		}
		args = append(args, arg)

		// Parse additional arguments
		for p.current.Type == TokenComma {
			p.advance() // consume ','
			
			arg, err := p.parseExpression()
			if err != nil {
				return nil, fmt.Errorf("function argument: %w", err)
			}
			args = append(args, arg)
		}
	}

	if p.current.Type != TokenRightParen {
		return nil, p.parseError("expected ')' after function arguments")
	}
	p.advance() // consume ')'

	return &mdwast.FunctionCallExpr{
		Name: name,
		Args: args,
		Pos:  pos,
	}, nil
}

// parseArrayExpression parses an array literal [elem1, elem2, ...]
func (p *Parser) parseArrayExpression() (mdwast.Expr, error) {
	pos := p.currentPosition()
	p.advance() // consume '['

	var elements []mdwast.Expr

	if p.current.Type != TokenRightBracket {
		// Parse first element
		elem, err := p.parseExpression()
		if err != nil {
			return nil, fmt.Errorf("array element: %w", err)
		}
		elements = append(elements, elem)

		// Parse additional elements
		for p.current.Type == TokenComma {
			p.advance() // consume ','
			
			elem, err := p.parseExpression()
			if err != nil {
				return nil, fmt.Errorf("array element: %w", err)
			}
			elements = append(elements, elem)
		}
	}

	if p.current.Type != TokenRightBracket {
		return nil, p.parseError("expected ']' after array elements")
	}
	p.advance() // consume ']'

	return &mdwast.ArrayExpr{
		Elements: elements,
		Pos:      pos,
	}, nil
}

// parseObjectExpression parses an object literal {key: value, ...}
func (p *Parser) parseObjectExpression() (mdwast.Expr, error) {
	pos := p.currentPosition()
	p.advance() // consume '{'

	fields := make(map[string]mdwast.Expr)

	if p.current.Type != TokenRightBrace {
		// Parse first field
		key, value, err := p.parseObjectField()
		if err != nil {
			return nil, err
		}
		fields[key] = value

		// Parse additional fields
		for p.current.Type == TokenComma {
			p.advance() // consume ','
			
			key, value, err := p.parseObjectField()
			if err != nil {
				return nil, err
			}
			fields[key] = value
		}
	}

	if p.current.Type != TokenRightBrace {
		return nil, p.parseError("expected '}' after object fields")
	}
	p.advance() // consume '}'

	return &mdwast.ObjectExpr{
		Fields: fields,
		Pos:    pos,
	}, nil
}

// parseObjectField parses a single object field (key: value)
func (p *Parser) parseObjectField() (string, mdwast.Expr, error) {
	// Parse key
	if p.current.Type != TokenIdentifier && p.current.Type != TokenString {
		return "", nil, p.parseError("expected object key")
	}

	key := p.current.Value
	p.advance()

	if p.current.Type != TokenColon {
		return "", nil, p.parseError("expected ':' after object key")
	}
	p.advance() // consume ':'

	// Parse value
	value, err := p.parseExpression()
	if err != nil {
		return "", nil, fmt.Errorf("object value: %w", err)
	}

	return key, value, nil
}

// Parameter represents a parsed parameter
type Parameter struct {
	Name  string
	Value mdwast.Value
}

// parseParameter parses a parameter (name=value)
func (p *Parser) parseParameter() (*Parameter, error) {
	if p.current.Type != TokenIdentifier {
		return nil, p.parseError("expected parameter name")
	}

	name := p.current.Value
	p.advance()

	if p.current.Type != TokenEquals {
		return nil, p.parseError("expected '=' after parameter name")
	}
	p.advance() // consume '='

	value, err := p.parseValue()
	if err != nil {
		return nil, fmt.Errorf("parameter value: %w", err)
	}

	return &Parameter{
		Name:  name,
		Value: value,
	}, nil
}

// parseValue parses a value literal
func (p *Parser) parseValue() (mdwast.Value, error) {
	pos := p.currentPosition()

	switch p.current.Type {
	case TokenString:
		value := mdwast.Value{
			Type:  mdwast.ValueTypeString,
			Raw:   p.current.Value,
			Value: p.current.Value,
			Pos:   pos,
		}
		p.advance()
		return value, nil

	case TokenNumber:
		raw := p.current.Value
		var parsedValue interface{}
		var valueType mdwast.ValueType

		if strings.Contains(raw, ".") {
			// Float
			f, err := strconv.ParseFloat(raw, 64)
			if err != nil {
				return mdwast.Value{}, p.parseError(fmt.Sprintf("invalid number: %s", raw))
			}
			parsedValue = f
			valueType = mdwast.ValueTypeNumber
		} else {
			// Integer
			i, err := strconv.ParseInt(raw, 10, 64)
			if err != nil {
				return mdwast.Value{}, p.parseError(fmt.Sprintf("invalid number: %s", raw))
			}
			parsedValue = i
			valueType = mdwast.ValueTypeNumber
		}

		value := mdwast.Value{
			Type:  valueType,
			Raw:   raw,
			Value: parsedValue,
			Pos:   pos,
		}
		p.advance()
		return value, nil

	case TokenBoolean:
		raw := p.current.Value
		boolValue := strings.ToLower(raw) == "true"
		
		value := mdwast.Value{
			Type:  mdwast.ValueTypeBoolean,
			Raw:   raw,
			Value: boolValue,
			Pos:   pos,
		}
		p.advance()
		return value, nil

	case TokenNull:
		value := mdwast.Value{
			Type:  mdwast.ValueTypeNull,
			Raw:   p.current.Value,
			Value: nil,
			Pos:   pos,
		}
		p.advance()
		return value, nil

	case TokenIdentifier:
		// Handle unquoted string values
		raw := p.current.Value
		value := mdwast.Value{
			Type:  mdwast.ValueTypeString,
			Raw:   raw,
			Value: raw,
			Pos:   pos,
		}
		p.advance()
		return value, nil

	default:
		return mdwast.Value{}, p.parseError(fmt.Sprintf("expected value, got %s", p.current.Type.String()))
	}
}

// Utility methods

// advance moves to the next token
func (p *Parser) advance() {
	p.previous = p.current
	p.current = p.lexer.NextToken()
}

// currentPosition returns the current AST position
func (p *Parser) currentPosition() mdwast.Position {
	return mdwast.Position{
		Line:   p.current.Line,
		Column: p.current.Column,
		Offset: p.current.Position,
	}
}

// parseError creates a parse error with current position
func (p *Parser) parseError(message string) error {
	return fmt.Errorf("parse error at line %d, column %d: %s (token: %s, type: %s)", 
		p.current.Line, p.current.Column, message, p.current.Value, p.current.Type.String())
}