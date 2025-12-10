# TCOL Developer Guide

**Technical Implementation and Integration Guide for mDW Platform**

Version: 1.0.0  
Date: 2025-07-26  
Author: mDW Foundation Development Team

---

## Table of Contents

1. [Architecture Overview](#architecture-overview)
2. [Parser Implementation](#parser-implementation)
3. [Command Execution Engine](#command-execution-engine)
4. [Foundation Integration](#foundation-integration)
5. [Error Handling](#error-handling)
6. [Performance Optimization](#performance-optimization)
7. [Security Implementation](#security-implementation)
8. [Extension Development](#extension-development)
9. [Testing Strategies](#testing-strategies)
10. [Deployment and Operations](#deployment-and-operations)

---

## Architecture Overview

### System Architecture

TCOL operates within the mDW microservice architecture:

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   TUI Client    │    │   Web Client    │    │   API Client    │
└─────────────────┘    └─────────────────┘    └─────────────────┘
         │                       │                       │
         └───────────────────────┼───────────────────────┘
                                 │
         ┌─────────────────────────────────────────────────┐
         │            TCOL Command Router                  │
         │  ┌─────────────┐  ┌─────────────┐  ┌─────────┐  │
         │  │   Parser    │  │  Validator  │  │ Executor│  │
         │  └─────────────┘  └─────────────┘  └─────────┘  │
         └─────────────────────────────────────────────────┘
                                 │
    ┌────────────────────────────┼────────────────────────────┐
    │                            │                            │
┌───▼───┐                 ┌──────▼──────┐                ┌───▼───┐
│Service│                 │ Application │                │Service│
│   A   │                 │   Server    │                │   C   │
└───────┘                 └─────────────┘                └───────┘
                                 │
                          ┌──────▼──────┐
                          │ Foundation  │
                          │  Modules    │
                          └─────────────┘
```

### Core Components

#### 1. TCOL Parser
- **Lexical Analysis**: Tokenizes TCOL command strings
- **Syntax Parsing**: Builds Abstract Syntax Trees (AST)
- **Semantic Analysis**: Validates command structure and types
- **Error Recovery**: Provides meaningful error messages

#### 2. Command Router
- **Object Resolution**: Maps objects to microservices
- **Method Dispatch**: Routes methods to appropriate handlers
- **Parameter Binding**: Maps parameters to service call arguments
- **Response Aggregation**: Combines results from multiple services

#### 3. Execution Engine
- **Context Management**: Maintains session and transaction context
- **Pipeline Processing**: Handles command chains and pipes
- **Async Operations**: Manages background and scheduled commands
- **Result Caching**: Optimizes repeated operations

### Technology Stack

```go
// Core dependencies
go 1.21+
grpc 1.58+
protobuf 3.21+

// mDW Foundation modules
github.com/foundation/pkg/core/error
github.com/foundation/pkg/core/log
github.com/foundation/pkg/utils/stringx
github.com/foundation/pkg/utils/mathx
github.com/foundation/pkg/utils/mapx
```

---

## Parser Implementation

### Lexical Analysis

The TCOL lexer tokenizes input using a state machine:

```go
// File: pkg/tcol/lexer.go
package tcol

import (
    "fmt"
    "unicode"
    
    "github.com/foundation/pkg/core/error"
    "github.com/foundation/pkg/core/log"
)

type TokenType int

const (
    TokenEOF TokenType = iota
    TokenObject
    TokenMethod
    TokenIdentifier
    TokenColon
    TokenDot
    TokenEquals
    TokenString
    TokenNumber
    TokenBoolean
    TokenLBracket
    TokenRBracket
    TokenComma
    TokenPipe
    TokenSemicolon
    TokenComment
)

type Token struct {
    Type     TokenType
    Value    string
    Position Position
}

type Position struct {
    Line   int
    Column int
    Offset int
}

type Lexer struct {
    input    string
    position int
    line     int
    column   int
    logger   log.Logger
}

func NewLexer(input string, logger log.Logger) *Lexer {
    return &Lexer{
        input:  input,
        line:   1,
        column: 1,
        logger: logger,
    }
}

func (l *Lexer) NextToken() (Token, error) {
    l.skipWhitespace()
    
    if l.position >= len(l.input) {
        return Token{Type: TokenEOF, Position: l.currentPosition()}, nil
    }
    
    char := l.current()
    
    switch {
    case isLetter(char):
        return l.readIdentifier()
    case isDigit(char):
        return l.readNumber()
    case char == '"':
        return l.readString()
    case char == ':':
        return l.singleCharToken(TokenColon), nil
    case char == '.':
        return l.singleCharToken(TokenDot), nil
    case char == '=':
        return l.singleCharToken(TokenEquals), nil
    case char == '[':
        return l.singleCharToken(TokenLBracket), nil
    case char == ']':
        return l.singleCharToken(TokenRBracket), nil
    case char == ',':
        return l.singleCharToken(TokenComma), nil
    case char == '|':
        return l.singleCharToken(TokenPipe), nil
    case char == ';':
        return l.singleCharToken(TokenSemicolon), nil
    case char == '/' && l.peek() == '/':
        return l.readComment()
    default:
        return Token{}, error.New(
            error.ParseError,
            "LEX_INVALID_CHAR",
            fmt.Sprintf("Invalid character: %c", char),
        ).WithContext("position", l.currentPosition())
    }
}

func (l *Lexer) readIdentifier() (Token, error) {
    start := l.position
    startPos := l.currentPosition()
    
    for l.position < len(l.input) && (isAlphaNumeric(l.current()) || l.current() == '_') {
        l.advance()
    }
    
    value := l.input[start:l.position]
    tokenType := TokenIdentifier
    
    // Classify as object, method, or keyword
    if isObjectName(value) {
        tokenType = TokenObject
    } else if isBooleanLiteral(value) {
        tokenType = TokenBoolean
    }
    
    return Token{
        Type:     tokenType,
        Value:    value,
        Position: startPos,
    }, nil
}

func (l *Lexer) readString() (Token, error) {
    startPos := l.currentPosition()
    l.advance() // Skip opening quote
    
    var value []rune
    
    for l.position < len(l.input) && l.current() != '"' {
        if l.current() == '\\' {
            l.advance()
            if l.position >= len(l.input) {
                return Token{}, error.New(
                    error.ParseError,
                    "LEX_UNTERMINATED_STRING",
                    "Unterminated string literal",
                ).WithContext("position", startPos)
            }
            
            switch l.current() {
            case 'n':
                value = append(value, '\n')
            case 't':
                value = append(value, '\t')
            case 'r':
                value = append(value, '\r')
            case '\\':
                value = append(value, '\\')
            case '"':
                value = append(value, '"')
            default:
                value = append(value, l.current())
            }
        } else {
            value = append(value, l.current())
        }
        l.advance()
    }
    
    if l.position >= len(l.input) {
        return Token{}, error.New(
            error.ParseError,
            "LEX_UNTERMINATED_STRING",
            "Unterminated string literal",
        ).WithContext("position", startPos)
    }
    
    l.advance() // Skip closing quote
    
    return Token{
        Type:     TokenString,
        Value:    string(value),
        Position: startPos,
    }, nil
}
```

### Syntax Parser

The parser builds an Abstract Syntax Tree (AST):

```go
// File: pkg/tcol/parser.go
package tcol

import (
    "github.com/foundation/pkg/core/error"
    "github.com/foundation/pkg/core/log"
)

type ASTNode interface {
    Type() string
    Position() Position
    Validate() error
}

type Command struct {
    Object     string
    Method     string
    Parameters map[string]interface{}
    Filter     *FilterExpression
    Options    map[string]interface{}
    Pos        Position
}

func (c *Command) Type() string { return "Command" }
func (c *Command) Position() Position { return c.Pos }

type FilterExpression struct {
    Conditions []Condition
    Operator   string // AND, OR
    Pos        Position
}

type Condition struct {
    Field    string
    Operator string
    Value    interface{}
    Pos      Position
}

type Parser struct {
    lexer   *Lexer
    current Token
    peek    Token
    logger  log.Logger
}

func NewParser(input string, logger log.Logger) (*Parser, error) {
    lexer := NewLexer(input, logger)
    parser := &Parser{
        lexer:  lexer,
        logger: logger,
    }
    
    // Initialize current and peek tokens
    var err error
    parser.current, err = lexer.NextToken()
    if err != nil {
        return nil, err
    }
    
    parser.peek, err = lexer.NextToken()
    if err != nil {
        return nil, err
    }
    
    return parser, nil
}

func (p *Parser) ParseCommand() (*Command, error) {
    if p.current.Type != TokenObject {
        return nil, error.New(
            error.ParseError,
            "PARSE_EXPECTED_OBJECT",
            "Expected object name",
        ).WithContext("position", p.current.Position)
    }
    
    cmd := &Command{
        Object:     p.current.Value,
        Parameters: make(map[string]interface{}),
        Options:    make(map[string]interface{}),
        Pos:        p.current.Position,
    }
    
    if err := p.advance(); err != nil {
        return nil, err
    }
    
    // Parse object identifier or method
    if p.current.Type == TokenColon {
        // Object access: CUSTOMER:12345
        if err := p.advance(); err != nil {
            return nil, err
        }
        
        if p.current.Type != TokenIdentifier && p.current.Type != TokenNumber {
            return nil, error.New(
                error.ParseError,
                "PARSE_EXPECTED_ID",
                "Expected object identifier",
            ).WithContext("position", p.current.Position)
        }
        
        cmd.Parameters["id"] = p.current.Value
        
        if err := p.advance(); err != nil {
            return nil, err
        }
        
        // Parse field updates: CUSTOMER:12345:field=value
        for p.current.Type == TokenColon {
            if err := p.parseFieldUpdate(cmd); err != nil {
                return nil, err
            }
        }
        
        return cmd, nil
    }
    
    if p.current.Type != TokenDot {
        return nil, error.New(
            error.ParseError,
            "PARSE_EXPECTED_DOT",
            "Expected '.' after object name",
        ).WithContext("position", p.current.Position)
    }
    
    if err := p.advance(); err != nil {
        return nil, err
    }
    
    if p.current.Type != TokenIdentifier {
        return nil, error.New(
            error.ParseError,
            "PARSE_EXPECTED_METHOD",
            "Expected method name",
        ).WithContext("position", p.current.Position)
    }
    
    cmd.Method = p.current.Value
    
    if err := p.advance(); err != nil {
        return nil, err
    }
    
    // Parse parameters
    for p.current.Type == TokenIdentifier {
        if err := p.parseParameter(cmd); err != nil {
            return nil, err
        }
    }
    
    // Parse filter
    if p.current.Type == TokenLBracket {
        filter, err := p.parseFilter()
        if err != nil {
            return nil, err
        }
        cmd.Filter = filter
    }
    
    return cmd, nil
}

func (p *Parser) parseParameter(cmd *Command) error {
    if p.current.Type != TokenIdentifier {
        return error.New(
            error.ParseError,
            "PARSE_EXPECTED_PARAM_NAME",
            "Expected parameter name",
        ).WithContext("position", p.current.Position)
    }
    
    paramName := p.current.Value
    
    if err := p.advance(); err != nil {
        return err
    }
    
    if p.current.Type != TokenEquals {
        return error.New(
            error.ParseError,
            "PARSE_EXPECTED_EQUALS",
            "Expected '=' after parameter name",
        ).WithContext("position", p.current.Position)
    }
    
    if err := p.advance(); err != nil {
        return err
    }
    
    value, err := p.parseValue()
    if err != nil {
        return err
    }
    
    cmd.Parameters[paramName] = value
    return nil
}

func (p *Parser) parseValue() (interface{}, error) {
    switch p.current.Type {
    case TokenString:
        value := p.current.Value
        if err := p.advance(); err != nil {
            return nil, err
        }
        return value, nil
        
    case TokenNumber:
        value := p.current.Value
        if err := p.advance(); err != nil {
            return nil, err
        }
        // Convert to appropriate numeric type
        return parseNumber(value)
        
    case TokenBoolean:
        value := p.current.Value == "true"
        if err := p.advance(); err != nil {
            return nil, err
        }
        return value, nil
        
    default:
        return nil, error.New(
            error.ParseError,
            "PARSE_EXPECTED_VALUE",
            "Expected parameter value",
        ).WithContext("position", p.current.Position)
    }
}
```

### AST Validation

```go
// File: pkg/tcol/validator.go
package tcol

import (
    "github.com/foundation/pkg/core/error"
    "github.com/foundation/pkg/utils/stringx"
)

type Validator struct {
    objectRegistry map[string]ObjectDefinition
    logger         log.Logger
}

type ObjectDefinition struct {
    Name        string
    Methods     map[string]MethodDefinition
    Fields      map[string]FieldDefinition
    Permissions map[string]Permission
}

type MethodDefinition struct {
    Name       string
    Parameters map[string]ParameterDefinition
    Returns    string
    Permission string
}

type ParameterDefinition struct {
    Name     string
    Type     string
    Required bool
    Default  interface{}
    Validate func(interface{}) error
}

func NewValidator(logger log.Logger) *Validator {
    return &Validator{
        objectRegistry: make(map[string]ObjectDefinition),
        logger:         logger,
    }
}

func (v *Validator) RegisterObject(def ObjectDefinition) {
    v.objectRegistry[def.Name] = def
}

func (v *Validator) ValidateCommand(cmd *Command) error {
    // Validate object exists
    objDef, exists := v.objectRegistry[cmd.Object]
    if !exists {
        return error.New(
            error.ValidationError,
            "VAL_OBJECT_NOT_FOUND",
            "Unknown object type",
        ).WithContext("object", cmd.Object).
          WithContext("position", cmd.Position())
    }
    
    // For object access (no method), validate ID parameter
    if cmd.Method == "" {
        return v.validateObjectAccess(cmd, objDef)
    }
    
    // Validate method exists
    methodDef, exists := objDef.Methods[cmd.Method]
    if !exists {
        return error.New(
            error.ValidationError,
            "VAL_METHOD_NOT_FOUND",
            "Unknown method for object",
        ).WithContext("object", cmd.Object).
          WithContext("method", cmd.Method).
          WithContext("position", cmd.Position())
    }
    
    // Validate parameters
    if err := v.validateParameters(cmd.Parameters, methodDef); err != nil {
        return err
    }
    
    // Validate filter
    if cmd.Filter != nil {
        if err := v.validateFilter(cmd.Filter, objDef); err != nil {
            return err
        }
    }
    
    return nil
}

func (v *Validator) validateParameters(params map[string]interface{}, methodDef MethodDefinition) error {
    // Check required parameters
    for paramName, paramDef := range methodDef.Parameters {
        if paramDef.Required {
            if _, exists := params[paramName]; !exists {
                return error.New(
                    error.ValidationError,
                    "VAL_REQUIRED_PARAM_MISSING",
                    "Required parameter missing",
                ).WithContext("parameter", paramName)
            }
        }
    }
    
    // Validate parameter types and values
    for paramName, value := range params {
        paramDef, exists := methodDef.Parameters[paramName]
        if !exists {
            return error.New(
                error.ValidationError,
                "VAL_UNKNOWN_PARAMETER",
                "Unknown parameter",
            ).WithContext("parameter", paramName)
        }
        
        if err := v.validateParameterValue(value, paramDef); err != nil {
            return error.Wrap(err, "Parameter validation failed").
                WithContext("parameter", paramName)
        }
    }
    
    return nil
}

func (v *Validator) validateParameterValue(value interface{}, paramDef ParameterDefinition) error {
    // Type validation
    if err := v.validateType(value, paramDef.Type); err != nil {
        return err
    }
    
    // Custom validation
    if paramDef.Validate != nil {
        if err := paramDef.Validate(value); err != nil {
            return err
        }
    }
    
    return nil
}

func (v *Validator) validateType(value interface{}, expectedType string) error {
    switch expectedType {
    case "string":
        if _, ok := value.(string); !ok {
            return error.New(
                error.ValidationError,
                "VAL_TYPE_MISMATCH",
                "Expected string value",
            )
        }
    case "number":
        switch value.(type) {
        case int, int32, int64, float32, float64:
            // Valid numeric types
        default:
            return error.New(
                error.ValidationError,
                "VAL_TYPE_MISMATCH",
                "Expected numeric value",
            )
        }
    case "boolean":
        if _, ok := value.(bool); !ok {
            return error.New(
                error.ValidationError,
                "VAL_TYPE_MISMATCH",
                "Expected boolean value",
            )
        }
    case "email":
        str, ok := value.(string)
        if !ok {
            return error.New(
                error.ValidationError,
                "VAL_TYPE_MISMATCH",
                "Expected string value for email",
            )
        }
        if !stringx.IsEmail(str) {
            return error.New(
                error.ValidationError,
                "VAL_INVALID_EMAIL",
                "Invalid email format",
            )
        }
    }
    
    return nil
}
```

---

## Command Execution Engine

### Execution Context

```go
// File: pkg/tcol/executor.go
package tcol

import (
    "context"
    "time"
    
    "github.com/foundation/pkg/core/error"
    "github.com/foundation/pkg/core/log"
    "github.com/foundation/pkg/utils/mapx"
)

type ExecutionContext struct {
    SessionID     string
    UserID        string
    TenantID      string
    CorrelationID string
    RequestID     string
    Timestamp     time.Time
    Variables     map[string]interface{}
    Permissions   []string
    Transaction   *TransactionContext
    Logger        log.Logger
    Cancel        context.CancelFunc
}

type TransactionContext struct {
    ID       string
    Timeout  time.Duration
    ReadOnly bool
    Commands []ExecutedCommand
}

type ExecutedCommand struct {
    Command   *Command
    Result    interface{}
    Error     error
    Duration  time.Duration
    Timestamp time.Time
}

type Executor struct {
    serviceRegistry map[string]ServiceClient
    middleware      []Middleware
    logger          log.Logger
    metrics         MetricsCollector
}

type ServiceClient interface {
    Execute(ctx context.Context, cmd *Command) (interface{}, error)
    Health() error
}

type Middleware interface {
    Execute(ctx *ExecutionContext, cmd *Command, next func() (interface{}, error)) (interface{}, error)
}

func NewExecutor(logger log.Logger, metrics MetricsCollector) *Executor {
    return &Executor{
        serviceRegistry: make(map[string]ServiceClient),
        middleware:      make([]Middleware, 0),
        logger:          logger,
        metrics:         metrics,
    }
}

func (e *Executor) RegisterService(objectType string, client ServiceClient) {
    e.serviceRegistry[objectType] = client
}

func (e *Executor) AddMiddleware(mw Middleware) {
    e.middleware = append(e.middleware, mw)
}

func (e *Executor) Execute(ctx *ExecutionContext, cmd *Command) (interface{}, error) {
    startTime := time.Now()
    
    // Log command execution start
    ctx.Logger.WithContext(context.Background()).Info("Executing TCOL command", log.Fields{
        "object":         cmd.Object,
        "method":         cmd.Method,
        "parameters":     cmd.Parameters,
        "session_id":     ctx.SessionID,
        "user_id":        ctx.UserID,
        "correlation_id": ctx.CorrelationID,
    })
    
    // Execute middleware chain
    result, err := e.executeWithMiddleware(ctx, cmd, 0)
    
    duration := time.Since(startTime)
    
    // Log execution result
    if err != nil {
        ctx.Logger.WithContext(context.Background()).Error("TCOL command failed", log.Fields{
            "object":     cmd.Object,
            "method":     cmd.Method,
            "duration":   duration,
            "error":      err.Error(),
        })
    } else {
        ctx.Logger.WithContext(context.Background()).Info("TCOL command completed", log.Fields{
            "object":   cmd.Object,
            "method":   cmd.Method,
            "duration": duration,
        })
    }
    
    // Record metrics
    e.metrics.RecordExecution(cmd.Object, cmd.Method, duration, err)
    
    // Record in transaction context
    if ctx.Transaction != nil {
        ctx.Transaction.Commands = append(ctx.Transaction.Commands, ExecutedCommand{
            Command:   cmd,
            Result:    result,
            Error:     err,
            Duration:  duration,
            Timestamp: startTime,
        })
    }
    
    return result, err
}

func (e *Executor) executeWithMiddleware(ctx *ExecutionContext, cmd *Command, index int) (interface{}, error) {
    if index >= len(e.middleware) {
        return e.executeCommand(ctx, cmd)
    }
    
    return e.middleware[index].Execute(ctx, cmd, func() (interface{}, error) {
        return e.executeWithMiddleware(ctx, cmd, index+1)
    })
}

func (e *Executor) executeCommand(ctx *ExecutionContext, cmd *Command) (interface{}, error) {
    // Find service for object type
    client, exists := e.serviceRegistry[cmd.Object]
    if !exists {
        return nil, error.New(
            error.SystemError,
            "SYS_SERVICE_NOT_FOUND",
            "Service not available for object type",
        ).WithContext("object", cmd.Object)
    }
    
    // Create context with timeout
    execCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()
    
    // Add execution context metadata
    execCtx = context.WithValue(execCtx, "session_id", ctx.SessionID)
    execCtx = context.WithValue(execCtx, "user_id", ctx.UserID)
    execCtx = context.WithValue(execCtx, "correlation_id", ctx.CorrelationID)
    
    // Execute command
    result, err := client.Execute(execCtx, cmd)
    if err != nil {
        return nil, error.Wrap(err, "Service execution failed").
            WithContext("object", cmd.Object).
            WithContext("method", cmd.Method)
    }
    
    return result, nil
}
```

### Middleware Implementation

```go
// File: pkg/tcol/middleware.go
package tcol

import (
    "time"
    
    "github.com/foundation/pkg/core/error"
    "github.com/foundation/pkg/core/log"
)

// AuthorizationMiddleware handles permission checking
type AuthorizationMiddleware struct {
    authService AuthorizationService
    logger      log.Logger
}

func NewAuthorizationMiddleware(authService AuthorizationService, logger log.Logger) *AuthorizationMiddleware {
    return &AuthorizationMiddleware{
        authService: authService,
        logger:      logger,
    }
}

func (m *AuthorizationMiddleware) Execute(ctx *ExecutionContext, cmd *Command, next func() (interface{}, error)) (interface{}, error) {
    // Check permissions
    permission := fmt.Sprintf("%s:%s", cmd.Object, cmd.Method)
    if cmd.Method == "" {
        permission = fmt.Sprintf("%s:READ", cmd.Object)
    }
    
    authorized, err := m.authService.CheckPermission(ctx.UserID, permission, cmd.Parameters)
    if err != nil {
        return nil, error.Wrap(err, "Authorization check failed")
    }
    
    if !authorized {
        return nil, error.New(
            error.SecurityError,
            "SEC_INSUFFICIENT_PERMISSIONS",
            "Insufficient permissions for operation",
        ).WithContext("user_id", ctx.UserID).
          WithContext("permission", permission)
    }
    
    return next()
}

// ValidationMiddleware handles input validation
type ValidationMiddleware struct {
    validator *Validator
    logger    log.Logger
}

func NewValidationMiddleware(validator *Validator, logger log.Logger) *ValidationMiddleware {
    return &ValidationMiddleware{
        validator: validator,
        logger:    logger,
    }
}

func (m *ValidationMiddleware) Execute(ctx *ExecutionContext, cmd *Command, next func() (interface{}, error)) (interface{}, error) {
    if err := m.validator.ValidateCommand(cmd); err != nil {
        return nil, error.Wrap(err, "Command validation failed")
    }
    
    return next()
}

// CachingMiddleware handles result caching
type CachingMiddleware struct {
    cache  CacheService
    logger log.Logger
}

func NewCachingMiddleware(cache CacheService, logger log.Logger) *CachingMiddleware {
    return &CachingMiddleware{
        cache:  cache,
        logger: logger,
    }
}

func (m *CachingMiddleware) Execute(ctx *ExecutionContext, cmd *Command, next func() (interface{}, error)) (interface{}, error) {
    // Only cache read operations
    if !isReadOperation(cmd) {
        return next()
    }
    
    // Generate cache key
    cacheKey := generateCacheKey(cmd, ctx.TenantID)
    
    // Try to get from cache
    if result, found := m.cache.Get(cacheKey); found {
        m.logger.Debug("Cache hit", log.Fields{
            "key":    cacheKey,
            "object": cmd.Object,
            "method": cmd.Method,
        })
        return result, nil
    }
    
    // Execute and cache result
    result, err := next()
    if err == nil {
        m.cache.Set(cacheKey, result, determineCacheTTL(cmd))
        m.logger.Debug("Result cached", log.Fields{
            "key":    cacheKey,
            "object": cmd.Object,
            "method": cmd.Method,
        })
    }
    
    return result, err
}

// MetricsMiddleware collects execution metrics
type MetricsMiddleware struct {
    metrics MetricsCollector
    logger  log.Logger
}

func (m *MetricsMiddleware) Execute(ctx *ExecutionContext, cmd *Command, next func() (interface{}, error)) (interface{}, error) {
    startTime := time.Now()
    
    result, err := next()
    
    duration := time.Since(startTime)
    
    // Record metrics
    m.metrics.RecordCommandExecution(CommandMetrics{
        Object:        cmd.Object,
        Method:        cmd.Method,
        Duration:      duration,
        Success:       err == nil,
        UserID:        ctx.UserID,
        TenantID:      ctx.TenantID,
        CorrelationID: ctx.CorrelationID,
    })
    
    return result, err
}
```

---

## Foundation Integration

### Error Handling Integration

```go
// File: pkg/tcol/integration/errors.go
package integration

import (
    "github.com/foundation/pkg/core/error"
    "github.com/foundation/pkg/core/log"
)

// TCOLErrorHandler integrates TCOL with Foundation error module
type TCOLErrorHandler struct {
    logger log.Logger
}

func NewTCOLErrorHandler(logger log.Logger) *TCOLErrorHandler {
    return &TCOLErrorHandler{
        logger: logger,
    }
}

func (h *TCOLErrorHandler) HandleParseError(err error, input string, position Position) error {
    foundationErr := error.New(
        error.ParseError,
        "TCOL_PARSE_ERROR",
        "Failed to parse TCOL command",
    ).WithContext("input", input).
      WithContext("line", position.Line).
      WithContext("column", position.Column).
      WithSeverity(error.SeverityMedium)
    
    if err != nil {
        foundationErr = error.Wrap(foundationErr, err.Error())
    }
    
    h.logger.Error("TCOL parse error", error.Fields(foundationErr))
    
    return foundationErr
}

func (h *TCOLErrorHandler) HandleValidationError(err error, cmd *Command) error {
    foundationErr := error.New(
        error.ValidationError,
        "TCOL_VALIDATION_ERROR",
        "TCOL command validation failed",
    ).WithContext("object", cmd.Object).
      WithContext("method", cmd.Method).
      WithContext("parameters", cmd.Parameters).
      WithSeverity(error.SeverityMedium)
    
    if err != nil {
        foundationErr = error.Wrap(foundationErr, err.Error())
    }
    
    h.logger.Warn("TCOL validation error", error.Fields(foundationErr))
    
    return foundationErr
}

func (h *TCOLErrorHandler) HandleExecutionError(err error, cmd *Command, ctx *ExecutionContext) error {
    severity := error.SeverityHigh
    code := "TCOL_EXECUTION_ERROR"
    
    // Classify error severity based on type
    if businessErr, ok := err.(*error.BusinessLogicError); ok {
        severity = error.SeverityMedium
        code = "TCOL_BUSINESS_ERROR"
    } else if sysErr, ok := err.(*error.SystemError); ok {
        severity = error.SeverityCritical
        code = "TCOL_SYSTEM_ERROR"
    }
    
    foundationErr := error.New(
        error.SystemError,
        code,
        "TCOL command execution failed",
    ).WithContext("object", cmd.Object).
      WithContext("method", cmd.Method).
      WithContext("session_id", ctx.SessionID).
      WithContext("user_id", ctx.UserID).
      WithContext("correlation_id", ctx.CorrelationID).
      WithSeverity(severity)
    
    if err != nil {
        foundationErr = error.Wrap(foundationErr, err.Error())
    }
    
    h.logger.Error("TCOL execution error", error.Fields(foundationErr))
    
    return foundationErr
}
```

### Logging Integration

```go
// File: pkg/tcol/integration/logging.go
package integration

import (
    "context"
    "time"
    
    "github.com/foundation/pkg/core/log"
)

// TCOLLogger integrates TCOL with Foundation logging module
type TCOLLogger struct {
    logger log.Logger
}

func NewTCOLLogger(baseLogger log.Logger) *TCOLLogger {
    return &TCOLLogger{
        logger: baseLogger.WithComponent("tcol"),
    }
}

func (l *TCOLLogger) LogCommandStart(ctx *ExecutionContext, cmd *Command) {
    l.logger.WithContext(context.Background()).Info("TCOL command started", log.Fields{
        "command_type":   "execution_start",
        "object":         cmd.Object,
        "method":         cmd.Method,
        "parameters":     cmd.Parameters,
        "session_id":     ctx.SessionID,
        "user_id":        ctx.UserID,
        "tenant_id":      ctx.TenantID,
        "correlation_id": ctx.CorrelationID,
        "request_id":     ctx.RequestID,
        "timestamp":      ctx.Timestamp,
    })
}

func (l *TCOLLogger) LogCommandComplete(ctx *ExecutionContext, cmd *Command, result interface{}, duration time.Duration) {
    l.logger.WithContext(context.Background()).Info("TCOL command completed", log.Fields{
        "command_type":   "execution_complete",
        "object":         cmd.Object,
        "method":         cmd.Method,
        "duration_ms":    duration.Milliseconds(),
        "result_type":    getResultType(result),
        "session_id":     ctx.SessionID,
        "user_id":        ctx.UserID,
        "correlation_id": ctx.CorrelationID,
    })
}

func (l *TCOLLogger) LogCommandError(ctx *ExecutionContext, cmd *Command, err error, duration time.Duration) {
    l.logger.WithContext(context.Background()).Error("TCOL command failed", log.Fields{
        "command_type":   "execution_error",
        "object":         cmd.Object,
        "method":         cmd.Method,
        "duration_ms":    duration.Milliseconds(),
        "error":          err.Error(),
        "error_type":     getErrorType(err),
        "session_id":     ctx.SessionID,
        "user_id":        ctx.UserID,
        "correlation_id": ctx.CorrelationID,
    })
}

func (l *TCOLLogger) LogAuditEvent(ctx *ExecutionContext, cmd *Command, eventType string, details map[string]interface{}) {
    fields := log.Fields{
        "event_type":     eventType,
        "object":         cmd.Object,
        "method":         cmd.Method,
        "session_id":     ctx.SessionID,
        "user_id":        ctx.UserID,
        "tenant_id":      ctx.TenantID,
        "correlation_id": ctx.CorrelationID,
        "timestamp":      time.Now(),
    }
    
    // Add event-specific details
    for key, value := range details {
        fields[key] = value
    }
    
    l.logger.WithContext(context.Background()).Audit("TCOL audit event", fields)
}

func (l *TCOLLogger) LogSecurityEvent(ctx *ExecutionContext, cmd *Command, eventType string, severity log.Level) {
    l.logger.WithContext(context.Background()).Log(severity, "TCOL security event", log.Fields{
        "security_event": eventType,
        "object":         cmd.Object,
        "method":         cmd.Method,
        "session_id":     ctx.SessionID,
        "user_id":        ctx.UserID,
        "tenant_id":      ctx.TenantID,
        "ip_address":     ctx.Variables["ip_address"],
        "user_agent":     ctx.Variables["user_agent"],
        "timestamp":      time.Now(),
    })
}

func (l *TCOLLogger) LogPerformanceMetrics(ctx *ExecutionContext, cmd *Command, metrics PerformanceMetrics) {
    l.logger.WithContext(context.Background()).Info("TCOL performance metrics", log.Fields{
        "metrics_type":     "performance",
        "object":           cmd.Object,
        "method":           cmd.Method,
        "execution_time":   metrics.ExecutionTime,
        "memory_usage":     metrics.MemoryUsage,
        "db_queries":       metrics.DatabaseQueries,
        "cache_hits":       metrics.CacheHits,
        "cache_misses":     metrics.CacheMisses,
        "session_id":       ctx.SessionID,
        "correlation_id":   ctx.CorrelationID,
    })
}
```

### Utility Integration

```go
// File: pkg/tcol/integration/utilities.go
package integration

import (
    "github.com/foundation/pkg/utils/stringx"
    "github.com/foundation/pkg/utils/mathx"
    "github.com/foundation/pkg/utils/mapx"
)

// TCOLUtilities provides Foundation utility integration for TCOL
type TCOLUtilities struct {
    stringUtils *stringx.Utils
    mathUtils   *mathx.Utils
}

func NewTCOLUtilities() *TCOLUtilities {
    return &TCOLUtilities{
        stringUtils: stringx.New(),
        mathUtils:   mathx.New(),
    }
}

// String utility functions for TCOL
func (u *TCOLUtilities) NormalizeString(input string) string {
    return u.stringUtils.Normalize(input)
}

func (u *TCOLUtilities) GenerateID(length int) string {
    return u.stringUtils.Random(length, stringx.AlphaNumeric)
}

func (u *TCOLUtilities) ValidateEmail(email string) bool {
    return u.stringUtils.IsEmail(email)
}

func (u *TCOLUtilities) FormatTitle(input string) string {
    return u.stringUtils.ToTitle(input)
}

// Math utility functions for TCOL
func (u *TCOLUtilities) CalculateDecimal(amount string, operation string, operand string) (string, error) {
    decimal1, err := u.mathUtils.NewDecimal(amount)
    if err != nil {
        return "", err
    }
    
    decimal2, err := u.mathUtils.NewDecimal(operand)
    if err != nil {
        return "", err
    }
    
    var result mathx.Decimal
    switch operation {
    case "add":
        result = decimal1.Add(decimal2)
    case "subtract":
        result = decimal1.Subtract(decimal2)
    case "multiply":
        result = decimal1.Multiply(decimal2)
    case "divide":
        result = decimal1.Divide(decimal2)
    default:
        return "", fmt.Errorf("unsupported operation: %s", operation)
    }
    
    return result.String(), nil
}

// Map utility functions for TCOL
func (u *TCOLUtilities) TransformData(data map[string]interface{}, operations map[string]string) map[string]interface{} {
    result := mapx.Clone(data)
    
    for field, operation := range operations {
        switch operation {
        case "upper":
            if str, ok := result[field].(string); ok {
                result[field] = strings.ToUpper(str)
            }
        case "lower":
            if str, ok := result[field].(string); ok {
                result[field] = strings.ToLower(str)
            }
        case "title":
            if str, ok := result[field].(string); ok {
                result[field] = u.stringUtils.ToTitle(str)
            }
        }
    }
    
    return result
}

func (u *TCOLUtilities) FilterData(data map[string]interface{}, filters map[string]interface{}) bool {
    for key, expectedValue := range filters {
        actualValue, exists := data[key]
        if !exists {
            return false
        }
        
        if !mapx.Equal(actualValue, expectedValue) {
            return false
        }
    }
    
    return true
}

func (u *TCOLUtilities) ExtractFields(data map[string]interface{}, fields []string) map[string]interface{} {
    return mapx.Pick(data, fields)
}

func (u *TCOLUtilities) MergeData(base map[string]interface{}, updates map[string]interface{}) map[string]interface{} {
    return mapx.Merge(base, updates)
}
```

---

## Performance Optimization

### Caching Strategy

```go
// File: pkg/tcol/cache.go
package tcol

import (
    "crypto/md5"
    "fmt"
    "time"
    
    "github.com/foundation/pkg/core/log"
)

type CacheService interface {
    Get(key string) (interface{}, bool)
    Set(key string, value interface{}, ttl time.Duration) error
    Delete(key string) error
    Clear() error
    Stats() CacheStats
}

type CacheStats struct {
    Hits        int64
    Misses      int64
    Evictions   int64
    KeyCount    int64
    MemoryUsage int64
}

type MemoryCache struct {
    store   map[string]cacheEntry
    stats   CacheStats
    maxSize int
    logger  log.Logger
}

type cacheEntry struct {
    value     interface{}
    expiry    time.Time
    lastAccess time.Time
}

func NewMemoryCache(maxSize int, logger log.Logger) *MemoryCache {
    return &MemoryCache{
        store:   make(map[string]cacheEntry),
        maxSize: maxSize,
        logger:  logger,
    }
}

func (c *MemoryCache) Get(key string) (interface{}, bool) {
    entry, exists := c.store[key]
    if !exists {
        c.stats.Misses++
        return nil, false
    }
    
    if time.Now().After(entry.expiry) {
        delete(c.store, key)
        c.stats.Misses++
        c.stats.Evictions++
        return nil, false
    }
    
    entry.lastAccess = time.Now()
    c.store[key] = entry
    c.stats.Hits++
    
    return entry.value, true
}

func (c *MemoryCache) Set(key string, value interface{}, ttl time.Duration) error {
    // Check if cache is full
    if len(c.store) >= c.maxSize {
        c.evictLRU()
    }
    
    c.store[key] = cacheEntry{
        value:      value,
        expiry:     time.Now().Add(ttl),
        lastAccess: time.Now(),
    }
    
    c.stats.KeyCount = int64(len(c.store))
    
    return nil
}

func (c *MemoryCache) evictLRU() {
    var oldestKey string
    var oldestTime time.Time
    
    for key, entry := range c.store {
        if oldestKey == "" || entry.lastAccess.Before(oldestTime) {
            oldestKey = key
            oldestTime = entry.lastAccess
        }
    }
    
    if oldestKey != "" {
        delete(c.store, oldestKey)
        c.stats.Evictions++
    }
}

// Cache key generation
func GenerateCacheKey(cmd *Command, tenantID string) string {
    data := fmt.Sprintf("%s:%s:%v:%v:%s", 
        cmd.Object, 
        cmd.Method, 
        cmd.Parameters, 
        cmd.Filter, 
        tenantID)
    
    hash := md5.Sum([]byte(data))
    return fmt.Sprintf("tcol:%x", hash)
}

// Cache TTL determination
func DetermineCacheTTL(cmd *Command) time.Duration {
    // Different TTL based on object type and operation
    switch cmd.Object {
    case "CUSTOMER":
        if cmd.Method == "LIST" {
            return 5 * time.Minute
        }
        return 1 * time.Minute
    case "INVOICE":
        return 30 * time.Second
    case "REPORT":
        return 15 * time.Minute
    default:
        return 1 * time.Minute
    }
}
```

### Batch Processing

```go
// File: pkg/tcol/batch.go
package tcol

import (
    "context"
    "sync"
    "time"
    
    "github.com/foundation/pkg/core/error"
    "github.com/foundation/pkg/core/log"
)

type BatchProcessor struct {
    executor     *Executor
    maxBatchSize int
    maxWaitTime  time.Duration
    workers      int
    logger       log.Logger
}

type BatchJob struct {
    Commands []BatchCommand
    Options  BatchOptions
    Result   chan BatchResult
}

type BatchCommand struct {
    ID      string
    Command *Command
    Context *ExecutionContext
}

type BatchResult struct {
    Results []CommandResult
    Errors  []error
    Stats   BatchStats
}

type CommandResult struct {
    ID       string
    Result   interface{}
    Error    error
    Duration time.Duration
}

type BatchStats struct {
    TotalCommands     int
    SuccessfulCommands int
    FailedCommands    int
    TotalDuration     time.Duration
    AverageDuration   time.Duration
}

type BatchOptions struct {
    MaxConcurrency   int
    ContinueOnError  bool
    TimeoutPerCommand time.Duration
    RetryCount       int
}

func NewBatchProcessor(executor *Executor, workers int, logger log.Logger) *BatchProcessor {
    return &BatchProcessor{
        executor:     executor,
        maxBatchSize: 1000,
        maxWaitTime:  5 * time.Second,
        workers:      workers,
        logger:       logger,
    }
}

func (bp *BatchProcessor) ProcessBatch(job BatchJob) BatchResult {
    startTime := time.Now()
    
    bp.logger.Info("Starting batch processing", log.Fields{
        "command_count": len(job.Commands),
        "max_concurrency": job.Options.MaxConcurrency,
    })
    
    results := make([]CommandResult, len(job.Commands))
    var wg sync.WaitGroup
    
    // Create worker pool
    concurrency := job.Options.MaxConcurrency
    if concurrency == 0 {
        concurrency = bp.workers
    }
    
    commandChan := make(chan BatchCommand, len(job.Commands))
    resultChan := make(chan CommandResult, len(job.Commands))
    
    // Start workers
    for i := 0; i < concurrency; i++ {
        wg.Add(1)
        go bp.worker(&wg, commandChan, resultChan, job.Options)
    }
    
    // Send commands to workers
    go func() {
        for _, cmd := range job.Commands {
            commandChan <- cmd
        }
        close(commandChan)
    }()
    
    // Collect results
    go func() {
        wg.Wait()
        close(resultChan)
    }()
    
    // Process results
    resultMap := make(map[string]CommandResult)
    for result := range resultChan {
        resultMap[result.ID] = result
    }
    
    // Order results according to original command order
    var errors []error
    successCount := 0
    failCount := 0
    
    for i, cmd := range job.Commands {
        result, exists := resultMap[cmd.ID]
        if !exists {
            result = CommandResult{
                ID:    cmd.ID,
                Error: error.New(error.SystemError, "BATCH_RESULT_MISSING", "Result missing for command"),
            }
        }
        
        results[i] = result
        
        if result.Error != nil {
            errors = append(errors, result.Error)
            failCount++
        } else {
            successCount++
        }
    }
    
    totalDuration := time.Since(startTime)
    avgDuration := totalDuration / time.Duration(len(job.Commands))
    
    stats := BatchStats{
        TotalCommands:      len(job.Commands),
        SuccessfulCommands: successCount,
        FailedCommands:     failCount,
        TotalDuration:      totalDuration,
        AverageDuration:    avgDuration,
    }
    
    bp.logger.Info("Batch processing completed", log.Fields{
        "total_commands":     stats.TotalCommands,
        "successful":         stats.SuccessfulCommands,
        "failed":            stats.FailedCommands,
        "total_duration":    stats.TotalDuration,
        "average_duration":  stats.AverageDuration,
    })
    
    return BatchResult{
        Results: results,
        Errors:  errors,
        Stats:   stats,
    }
}

func (bp *BatchProcessor) worker(wg *sync.WaitGroup, commandChan <-chan BatchCommand, resultChan chan<- CommandResult, options BatchOptions) {
    defer wg.Done()
    
    for batchCmd := range commandChan {
        startTime := time.Now()
        
        // Set timeout for individual command
        if options.TimeoutPerCommand > 0 {
            ctx, cancel := context.WithTimeout(context.Background(), options.TimeoutPerCommand)
            batchCmd.Context.Cancel = cancel
            defer cancel()
        }
        
        // Execute command with retry
        var result interface{}
        var err error
        
        for attempt := 0; attempt <= options.RetryCount; attempt++ {
            result, err = bp.executor.Execute(batchCmd.Context, batchCmd.Command)
            
            if err == nil {
                break // Success, no retry needed
            }
            
            // Check if error is retryable
            if !isRetryableError(err) {
                break // Don't retry non-retryable errors
            }
            
            if attempt < options.RetryCount {
                bp.logger.Warn("Retrying command", log.Fields{
                    "command_id": batchCmd.ID,
                    "attempt":    attempt + 1,
                    "error":      err.Error(),
                })
                
                // Exponential backoff
                time.Sleep(time.Duration(attempt+1) * 100 * time.Millisecond)
            }
        }
        
        duration := time.Since(startTime)
        
        resultChan <- CommandResult{
            ID:       batchCmd.ID,
            Result:   result,
            Error:    err,
            Duration: duration,
        }
    }
}

func isRetryableError(err error) bool {
    // Determine if an error is retryable based on error type
    switch err.(type) {
    case *error.SystemError:
        return true // System errors might be transient
    case *error.NetworkError:
        return true // Network errors are often transient
    case *error.TimeoutError:
        return true // Timeouts can be retried
    default:
        return false // Business logic and validation errors shouldn't be retried
    }
}
```

---

## Deployment and Operations

### Docker Configuration

```dockerfile
# File: Dockerfile
FROM golang:1.21-alpine AS builder

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o tcol-server ./cmd/tcol-server

# Production image
FROM alpine:latest

RUN apk --no-cache add ca-certificates
WORKDIR /root/

# Copy binary from builder
COPY --from=builder /app/tcol-server .

# Copy configuration files
COPY --from=builder /app/configs/ ./configs/

# Expose port
EXPOSE 8080 9090

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

# Run the application
CMD ["./tcol-server"]
```

### Kubernetes Deployment

```yaml
# File: k8s/deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: tcol-server
  labels:
    app: tcol-server
    version: v1.0.0
spec:
  replicas: 3
  selector:
    matchLabels:
      app: tcol-server
  template:
    metadata:
      labels:
        app: tcol-server
        version: v1.0.0
    spec:
      containers:
      - name: tcol-server
        image: mdw/tcol-server:v1.0.0
        ports:
        - containerPort: 8080
          name: http
        - containerPort: 9090
          name: grpc
        env:
        - name: LOG_LEVEL
          value: "info"
        - name: LOG_FORMAT
          value: "json"
        - name: DATABASE_URL
          valueFrom:
            secretKeyRef:
              name: tcol-secrets
              key: database-url
        - name: REDIS_URL
          valueFrom:
            secretKeyRef:
              name: tcol-secrets
              key: redis-url
        resources:
          requests:
            memory: "256Mi"
            cpu: "100m"
          limits:
            memory: "512Mi"
            cpu: "500m"
        livenessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 30
          periodSeconds: 10
        readinessProbe:
          httpGet:
            path: /ready
            port: 8080
          initialDelaySeconds: 5
          periodSeconds: 5
        volumeMounts:
        - name: config
          mountPath: /root/configs
          readOnly: true
      volumes:
      - name: config
        configMap:
          name: tcol-config
---
apiVersion: v1
kind: Service
metadata:
  name: tcol-service
  labels:
    app: tcol-server
spec:
  selector:
    app: tcol-server
  ports:
  - name: http
    port: 80
    targetPort: 8080
  - name: grpc
    port: 9090
    targetPort: 9090
  type: ClusterIP
```

### Monitoring Configuration

```yaml
# File: monitoring/prometheus.yaml
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  name: tcol-server-metrics
  labels:
    app: tcol-server
spec:
  selector:
    matchLabels:
      app: tcol-server
  endpoints:
  - port: http
    path: /metrics
    interval: 30s
    scrapeTimeout: 10s
---
apiVersion: monitoring.coreos.com/v1
kind: PrometheusRule
metadata:
  name: tcol-server-alerts
  labels:
    app: tcol-server
spec:
  groups:
  - name: tcol-server
    rules:
    - alert: TCOLHighErrorRate
      expr: rate(tcol_command_errors_total[5m]) > 0.1
      for: 2m
      labels:
        severity: warning
      annotations:
        summary: "High TCOL command error rate"
        description: "TCOL error rate is above 10% for the last 5 minutes"
    
    - alert: TCOLHighLatency
      expr: histogram_quantile(0.95, rate(tcol_command_duration_seconds_bucket[5m])) > 2
      for: 5m
      labels:
        severity: warning
      annotations:
        summary: "High TCOL command latency"
        description: "95th percentile latency is above 2 seconds"
    
    - alert: TCOLServiceDown
      expr: up{job="tcol-server"} == 0
      for: 1m
      labels:
        severity: critical
      annotations:
        summary: "TCOL service is down"
        description: "TCOL service has been down for more than 1 minute"
```

---

This developer guide provides comprehensive technical documentation for implementing and extending TCOL within the mDW platform. It covers architecture, implementation details, Foundation integration, and operational considerations for production deployment.

The guide demonstrates how TCOL leverages mDW Foundation modules for robust error handling, structured logging, and utility functions while providing a scalable, performant command execution engine.