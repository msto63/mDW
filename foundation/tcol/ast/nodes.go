// File: nodes.go
// Title: TCOL AST Node Definitions
// Description: Defines all AST node types for representing TCOL commands
//              including commands, expressions, filters, and parameters.
//              Provides string representations and validation methods.
// Author: msto63 with Claude Sonnet 4.0
// Version: v0.1.0
// Created: 2025-01-25
// Modified: 2025-01-25
//
// Change History:
// - 2025-01-25 v0.1.0: Initial AST node definitions

package ast

import (
	"fmt"
	"strings"
	"time"

	mdwstringx "github.com/msto63/mDW/foundation/utils/stringx"
)

// Node represents the base interface for all AST nodes
type Node interface {
	// String returns a string representation of the node
	String() string

	// Accept implements the visitor pattern
	Accept(visitor Visitor) interface{}

	// Position returns the source position of the node
	Position() Position

	// Validate performs basic validation of the node
	Validate() error
}

// Position represents a position in the source code
type Position struct {
	Line   int // Line number (1-based)
	Column int // Column number (1-based)
	Offset int // Byte offset (0-based)
}

// Command represents a complete TCOL command
type Command struct {
	Object     string            // Object name (e.g., "CUSTOMER")
	Method     string            // Method name (e.g., "CREATE")
	Parameters map[string]Value  // Method parameters
	Filter     *FilterExpr       // Optional filter expression
	ObjectID   string            // Direct object ID access (OBJECT:ID)
	FieldOp    *FieldOperation   // Field operation (OBJECT:ID:field=value)
	Chain      *Command          // Next command in chain
	Pos        Position          // Source position
}

// FilterExpr represents a filter expression [condition]
type FilterExpr struct {
	Condition Expr     // Filter condition expression
	Pos       Position // Source position
}

// FieldOperation represents a field operation (assignment or access)
type FieldOperation struct {
	Field string    // Field name
	Op    string    // Operation ("=" for assignment, empty for access)
	Value Value     // Value for assignment operations
	Pos   Position  // Source position
}

// Expr represents the base interface for all expressions
type Expr interface {
	Node
	exprNode() // marker method
}

// Value represents a value in TCOL (string, number, boolean, etc.)
type Value struct {
	Type  ValueType   // Type of the value
	Raw   string      // Raw string representation
	Value interface{} // Parsed value
	Pos   Position    // Source position
}

// ValueType represents the type of a value
type ValueType int

const (
	ValueTypeString ValueType = iota
	ValueTypeNumber
	ValueTypeBoolean
	ValueTypeDate
	ValueTypeTime
	ValueTypeNull
	ValueTypeArray
	ValueTypeObject
)

// String returns string representation of ValueType
func (vt ValueType) String() string {
	switch vt {
	case ValueTypeString:
		return "string"
	case ValueTypeNumber:
		return "number"
	case ValueTypeBoolean:
		return "boolean"
	case ValueTypeDate:
		return "date"
	case ValueTypeTime:
		return "time"
	case ValueTypeNull:
		return "null"
	case ValueTypeArray:
		return "array"
	case ValueTypeObject:
		return "object"
	default:
		return "unknown"
	}
}

// Expression types

// BinaryExpr represents a binary expression (a AND b, a = b, etc.)
type BinaryExpr struct {
	Left  Expr     // Left operand
	Op    string   // Operator (AND, OR, =, !=, <, >, etc.)
	Right Expr     // Right operand
	Pos   Position // Source position
}

// UnaryExpr represents a unary expression (NOT a, -a, etc.)
type UnaryExpr struct {
	Op   string   // Operator (NOT, -, etc.)
	Expr Expr     // Operand expression
	Pos  Position // Source position
}

// IdentifierExpr represents an identifier (field name, variable, etc.)
type IdentifierExpr struct {
	Name string   // Identifier name
	Pos  Position // Source position
}

// LiteralExpr represents a literal value
type LiteralExpr struct {
	Value Value    // Literal value
	Pos   Position // Source position
}

// FunctionCallExpr represents a function call
type FunctionCallExpr struct {
	Name string   // Function name
	Args []Expr   // Function arguments
	Pos  Position // Source position
}

// ArrayExpr represents an array literal [1, 2, 3]
type ArrayExpr struct {
	Elements []Expr   // Array elements
	Pos      Position // Source position
}

// ObjectExpr represents an object literal {key: value}
type ObjectExpr struct {
	Fields map[string]Expr // Object fields
	Pos    Position        // Source position
}

// Implementation of Node interface for Command

func (c *Command) String() string {
	var parts []string

	// Object and method
	if !mdwstringx.IsBlank(c.ObjectID) {
		parts = append(parts, fmt.Sprintf("%s:%s", c.Object, c.ObjectID))
		
		// Field operation
		if c.FieldOp != nil {
			if c.FieldOp.Op == "=" {
				parts = append(parts, fmt.Sprintf(":%s=%s", c.FieldOp.Field, c.FieldOp.Value.String()))
			} else {
				parts = append(parts, fmt.Sprintf(":%s", c.FieldOp.Field))
			}
		}
	} else {
		objectPart := c.Object
		
		// Add filter if present
		if c.Filter != nil {
			objectPart += c.Filter.String()
		}
		
		parts = append(parts, fmt.Sprintf("%s.%s", objectPart, c.Method))
	}

	// Parameters
	if len(c.Parameters) > 0 {
		var paramParts []string
		for key, value := range c.Parameters {
			paramParts = append(paramParts, fmt.Sprintf("%s=%s", key, value.String()))
		}
		parts = append(parts, strings.Join(paramParts, " "))
	}

	result := strings.Join(parts, " ")

	// Chain
	if c.Chain != nil {
		result += " | " + c.Chain.String()
	}

	return result
}

func (c *Command) Accept(visitor Visitor) interface{} {
	return visitor.VisitCommand(c)
}

func (c *Command) Position() Position {
	return c.Pos
}

func (c *Command) Validate() error {
	// Object name is required
	if mdwstringx.IsBlank(c.Object) {
		return fmt.Errorf("object name is required")
	}

	// Method is required unless it's a direct object access
	if mdwstringx.IsBlank(c.Method) && mdwstringx.IsBlank(c.ObjectID) {
		return fmt.Errorf("method name is required")
	}

	// Validate parameters
	for key, value := range c.Parameters {
		if mdwstringx.IsBlank(key) {
			return fmt.Errorf("parameter name cannot be empty")
		}
		if err := value.Validate(); err != nil {
			return fmt.Errorf("parameter %s: %w", key, err)
		}
	}

	// Validate filter
	if c.Filter != nil {
		if err := c.Filter.Validate(); err != nil {
			return fmt.Errorf("filter: %w", err)
		}
	}

	// Validate field operation
	if c.FieldOp != nil {
		if err := c.FieldOp.Validate(); err != nil {
			return fmt.Errorf("field operation: %w", err)
		}
	}

	// Validate chain
	if c.Chain != nil {
		if err := c.Chain.Validate(); err != nil {
			return fmt.Errorf("chain: %w", err)
		}
	}

	return nil
}

// IsDirectAccess returns true if this is a direct object access (OBJECT:ID)
func (c *Command) IsDirectAccess() bool {
	return !mdwstringx.IsBlank(c.ObjectID)
}

// IsFieldOperation returns true if this is a field operation (OBJECT:ID:field=value)
func (c *Command) IsFieldOperation() bool {
	return c.FieldOp != nil
}

// IsChained returns true if this command is part of a chain
func (c *Command) IsChained() bool {
	return c.Chain != nil
}

// HasFilter returns true if this command has a filter
func (c *Command) HasFilter() bool {
	return c.Filter != nil
}

// Implementation of Node interface for FilterExpr

func (f *FilterExpr) String() string {
	return fmt.Sprintf("[%s]", f.Condition.String())
}

func (f *FilterExpr) Accept(visitor Visitor) interface{} {
	return visitor.VisitFilter(f)
}

func (f *FilterExpr) Position() Position {
	return f.Pos
}

func (f *FilterExpr) Validate() error {
	if f.Condition == nil {
		return fmt.Errorf("filter condition is required")
	}
	return f.Condition.Validate()
}

// Implementation of Node interface for FieldOperation

func (fo *FieldOperation) String() string {
	if fo.Op == "=" {
		return fmt.Sprintf("%s=%s", fo.Field, fo.Value.String())
	}
	return fo.Field
}

func (fo *FieldOperation) Accept(visitor Visitor) interface{} {
	return visitor.VisitFieldOperation(fo)
}

func (fo *FieldOperation) Position() Position {
	return fo.Pos
}

func (fo *FieldOperation) Validate() error {
	if mdwstringx.IsBlank(fo.Field) {
		return fmt.Errorf("field name is required")
	}
	
	if fo.Op == "=" && fo.Value.Type == ValueTypeNull {
		return fmt.Errorf("assignment requires a value")
	}
	
	return nil
}

// Implementation of Node interface for Value

func (v *Value) String() string {
	switch v.Type {
	case ValueTypeString:
		if strings.Contains(v.Raw, " ") || strings.Contains(v.Raw, "\"") {
			return fmt.Sprintf(`"%s"`, strings.ReplaceAll(v.Raw, `"`, `\"`))
		}
		return v.Raw
	case ValueTypeNull:
		return "null"
	default:
		return v.Raw
	}
}

func (v *Value) Accept(visitor Visitor) interface{} {
	return visitor.VisitValue(v)
}

func (v *Value) Position() Position {
	return v.Pos
}

func (v *Value) Validate() error {
	// Basic type validation
	switch v.Type {
	case ValueTypeString:
		return nil // Strings are always valid
	case ValueTypeNumber:
		// Validate that Value is actually a number
		switch v.Value.(type) {
		case int, int64, float64:
			return nil
		default:
			return fmt.Errorf("invalid number value: %v", v.Value)
		}
	case ValueTypeBoolean:
		// Validate that Value is actually a boolean
		if _, ok := v.Value.(bool); !ok {
			return fmt.Errorf("invalid boolean value: %v", v.Value)
		}
		return nil
	case ValueTypeDate, ValueTypeTime:
		// Validate that Value is actually a time.Time
		if _, ok := v.Value.(time.Time); !ok {
			return fmt.Errorf("invalid date/time value: %v", v.Value)
		}
		return nil
	case ValueTypeNull:
		return nil // Null is always valid
	case ValueTypeArray:
		// Validate array elements
		if arr, ok := v.Value.([]interface{}); ok {
			for i, elem := range arr {
				if val, ok := elem.(Value); ok {
					if err := val.Validate(); err != nil {
						return fmt.Errorf("array element %d: %w", i, err)
					}
				}
			}
		}
		return nil
	case ValueTypeObject:
		// Validate object fields
		if obj, ok := v.Value.(map[string]interface{}); ok {
			for key, value := range obj {
				if val, ok := value.(Value); ok {
					if err := val.Validate(); err != nil {
						return fmt.Errorf("object field %s: %w", key, err)
					}
				}
			}
		}
		return nil
	default:
		return fmt.Errorf("unknown value type: %v", v.Type)
	}
}

// GetStringValue returns the string value, converting if necessary
func (v *Value) GetStringValue() string {
	switch v.Type {
	case ValueTypeString:
		return v.Raw
	case ValueTypeNumber, ValueTypeBoolean:
		return fmt.Sprintf("%v", v.Value)
	case ValueTypeNull:
		return ""
	default:
		return v.Raw
	}
}

// GetNumberValue returns the numeric value if possible
func (v *Value) GetNumberValue() (float64, bool) {
	switch val := v.Value.(type) {
	case int:
		return float64(val), true
	case int64:
		return float64(val), true
	case float64:
		return val, true
	default:
		return 0, false
	}
}

// GetBoolValue returns the boolean value if possible
func (v *Value) GetBoolValue() (bool, bool) {
	if val, ok := v.Value.(bool); ok {
		return val, true
	}
	return false, false
}

// Expression implementations

func (be *BinaryExpr) String() string {
	return fmt.Sprintf("(%s %s %s)", be.Left.String(), be.Op, be.Right.String())
}

func (be *BinaryExpr) Accept(visitor Visitor) interface{} {
	return visitor.VisitBinaryExpr(be)
}

func (be *BinaryExpr) Position() Position {
	return be.Pos
}

func (be *BinaryExpr) Validate() error {
	if be.Left == nil {
		return fmt.Errorf("left operand is required")
	}
	if be.Right == nil {
		return fmt.Errorf("right operand is required")
	}
	if mdwstringx.IsBlank(be.Op) {
		return fmt.Errorf("operator is required")
	}
	
	if err := be.Left.Validate(); err != nil {
		return fmt.Errorf("left operand: %w", err)
	}
	if err := be.Right.Validate(); err != nil {
		return fmt.Errorf("right operand: %w", err)
	}
	
	return nil
}

func (be *BinaryExpr) exprNode() {}

func (ue *UnaryExpr) String() string {
	return fmt.Sprintf("(%s %s)", ue.Op, ue.Expr.String())
}

func (ue *UnaryExpr) Accept(visitor Visitor) interface{} {
	return visitor.VisitUnaryExpr(ue)
}

func (ue *UnaryExpr) Position() Position {
	return ue.Pos
}

func (ue *UnaryExpr) Validate() error {
	if ue.Expr == nil {
		return fmt.Errorf("operand is required")
	}
	if mdwstringx.IsBlank(ue.Op) {
		return fmt.Errorf("operator is required")
	}
	return ue.Expr.Validate()
}

func (ue *UnaryExpr) exprNode() {}

func (ie *IdentifierExpr) String() string {
	return ie.Name
}

func (ie *IdentifierExpr) Accept(visitor Visitor) interface{} {
	return visitor.VisitIdentifier(ie)
}

func (ie *IdentifierExpr) Position() Position {
	return ie.Pos
}

func (ie *IdentifierExpr) Validate() error {
	if mdwstringx.IsBlank(ie.Name) {
		return fmt.Errorf("identifier name is required")
	}
	return nil
}

func (ie *IdentifierExpr) exprNode() {}

func (le *LiteralExpr) String() string {
	return le.Value.String()
}

func (le *LiteralExpr) Accept(visitor Visitor) interface{} {
	return visitor.VisitLiteral(le)
}

func (le *LiteralExpr) Position() Position {
	return le.Pos
}

func (le *LiteralExpr) Validate() error {
	return le.Value.Validate()
}

func (le *LiteralExpr) exprNode() {}

func (fce *FunctionCallExpr) String() string {
	var args []string
	for _, arg := range fce.Args {
		args = append(args, arg.String())
	}
	return fmt.Sprintf("%s(%s)", fce.Name, strings.Join(args, ", "))
}

func (fce *FunctionCallExpr) Accept(visitor Visitor) interface{} {
	return visitor.VisitFunctionCall(fce)
}

func (fce *FunctionCallExpr) Position() Position {
	return fce.Pos
}

func (fce *FunctionCallExpr) Validate() error {
	if mdwstringx.IsBlank(fce.Name) {
		return fmt.Errorf("function name is required")
	}
	
	for i, arg := range fce.Args {
		if err := arg.Validate(); err != nil {
			return fmt.Errorf("argument %d: %w", i, err)
		}
	}
	
	return nil
}

func (fce *FunctionCallExpr) exprNode() {}

func (ae *ArrayExpr) String() string {
	var elements []string
	for _, elem := range ae.Elements {
		elements = append(elements, elem.String())
	}
	return fmt.Sprintf("[%s]", strings.Join(elements, ", "))
}

func (ae *ArrayExpr) Accept(visitor Visitor) interface{} {
	return visitor.VisitArray(ae)
}

func (ae *ArrayExpr) Position() Position {
	return ae.Pos
}

func (ae *ArrayExpr) Validate() error {
	for i, elem := range ae.Elements {
		if err := elem.Validate(); err != nil {
			return fmt.Errorf("element %d: %w", i, err)
		}
	}
	return nil
}

func (ae *ArrayExpr) exprNode() {}

func (oe *ObjectExpr) String() string {
	var fields []string
	for key, value := range oe.Fields {
		fields = append(fields, fmt.Sprintf("%s: %s", key, value.String()))
	}
	return fmt.Sprintf("{%s}", strings.Join(fields, ", "))
}

func (oe *ObjectExpr) Accept(visitor Visitor) interface{} {
	return visitor.VisitObject(oe)
}

func (oe *ObjectExpr) Position() Position {
	return oe.Pos
}

func (oe *ObjectExpr) Validate() error {
	for key, value := range oe.Fields {
		if mdwstringx.IsBlank(key) {
			return fmt.Errorf("object key cannot be empty")
		}
		if err := value.Validate(); err != nil {
			return fmt.Errorf("field %s: %w", key, err)
		}
	}
	return nil
}

func (oe *ObjectExpr) exprNode() {}