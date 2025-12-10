// File: visitor.go
// Title: TCOL AST Visitor Pattern Implementation
// Description: Implements the visitor pattern for traversing and processing
//              TCOL AST nodes. Provides base visitor interface and common
//              visitor implementations for analysis and transformation.
// Author: msto63 with Claude Sonnet 4.0
// Version: v0.1.0
// Created: 2025-01-25
// Modified: 2025-01-25
//
// Change History:
// - 2025-01-25 v0.1.0: Initial visitor pattern implementation

package ast

import (
	"fmt"
	"strings"
)

// Visitor interface for traversing AST nodes using the visitor pattern
type Visitor interface {
	// Visit command nodes
	VisitCommand(cmd *Command) interface{}
	VisitFilter(filter *FilterExpr) interface{}
	VisitFieldOperation(fieldOp *FieldOperation) interface{}
	VisitValue(value *Value) interface{}

	// Visit expression nodes
	VisitBinaryExpr(expr *BinaryExpr) interface{}
	VisitUnaryExpr(expr *UnaryExpr) interface{}
	VisitIdentifier(expr *IdentifierExpr) interface{}
	VisitLiteral(expr *LiteralExpr) interface{}
	VisitFunctionCall(expr *FunctionCallExpr) interface{}
	VisitArray(expr *ArrayExpr) interface{}
	VisitObject(expr *ObjectExpr) interface{}
}

// BaseVisitor provides default implementations for all visitor methods
// Embed this in concrete visitors to only override needed methods
type BaseVisitor struct{}

func (bv *BaseVisitor) VisitCommand(cmd *Command) interface{} {
	// Visit filter if present
	if cmd.Filter != nil {
		cmd.Filter.Accept(bv)
	}

	// Visit field operation if present
	if cmd.FieldOp != nil {
		cmd.FieldOp.Accept(bv)
	}

	// Visit parameters
	for _, value := range cmd.Parameters {
		value.Accept(bv)
	}

	// Visit chain if present
	if cmd.Chain != nil {
		cmd.Chain.Accept(bv)
	}

	return nil
}

func (bv *BaseVisitor) VisitFilter(filter *FilterExpr) interface{} {
	if filter.Condition != nil {
		return filter.Condition.Accept(bv)
	}
	return nil
}

func (bv *BaseVisitor) VisitFieldOperation(fieldOp *FieldOperation) interface{} {
	if fieldOp.Op == "=" {
		return fieldOp.Value.Accept(bv)
	}
	return nil
}

func (bv *BaseVisitor) VisitValue(value *Value) interface{} {
	return nil // Terminal node
}

func (bv *BaseVisitor) VisitBinaryExpr(expr *BinaryExpr) interface{} {
	if expr.Left != nil {
		expr.Left.Accept(bv)
	}
	if expr.Right != nil {
		expr.Right.Accept(bv)
	}
	return nil
}

func (bv *BaseVisitor) VisitUnaryExpr(expr *UnaryExpr) interface{} {
	if expr.Expr != nil {
		return expr.Expr.Accept(bv)
	}
	return nil
}

func (bv *BaseVisitor) VisitIdentifier(expr *IdentifierExpr) interface{} {
	return nil // Terminal node
}

func (bv *BaseVisitor) VisitLiteral(expr *LiteralExpr) interface{} {
	return expr.Value.Accept(bv)
}

func (bv *BaseVisitor) VisitFunctionCall(expr *FunctionCallExpr) interface{} {
	for _, arg := range expr.Args {
		arg.Accept(bv)
	}
	return nil
}

func (bv *BaseVisitor) VisitArray(expr *ArrayExpr) interface{} {
	for _, elem := range expr.Elements {
		elem.Accept(bv)
	}
	return nil
}

func (bv *BaseVisitor) VisitObject(expr *ObjectExpr) interface{} {
	for _, value := range expr.Fields {
		value.Accept(bv)
	}
	return nil
}

// StringVisitor creates a string representation of the AST
type StringVisitor struct {
	BaseVisitor
	buffer strings.Builder
	indent int
}

// NewStringVisitor creates a new string visitor
func NewStringVisitor() *StringVisitor {
	return &StringVisitor{}
}

// String returns the built string representation
func (sv *StringVisitor) String() string {
	return sv.buffer.String()
}

// Reset clears the internal buffer
func (sv *StringVisitor) Reset() {
	sv.buffer.Reset()
	sv.indent = 0
}

func (sv *StringVisitor) writeIndent() {
	for i := 0; i < sv.indent; i++ {
		sv.buffer.WriteString("  ")
	}
}

func (sv *StringVisitor) VisitCommand(cmd *Command) interface{} {
	sv.writeIndent()
	sv.buffer.WriteString("Command:\n")
	sv.indent++

	sv.writeIndent()
	sv.buffer.WriteString(fmt.Sprintf("Object: %s\n", cmd.Object))

	if cmd.ObjectID != "" {
		sv.writeIndent()
		sv.buffer.WriteString(fmt.Sprintf("ObjectID: %s\n", cmd.ObjectID))
	}

	if cmd.Method != "" {
		sv.writeIndent()
		sv.buffer.WriteString(fmt.Sprintf("Method: %s\n", cmd.Method))
	}

	if len(cmd.Parameters) > 0 {
		sv.writeIndent()
		sv.buffer.WriteString("Parameters:\n")
		sv.indent++
		for key, value := range cmd.Parameters {
			sv.writeIndent()
			sv.buffer.WriteString(fmt.Sprintf("%s: ", key))
			value.Accept(sv)
			sv.buffer.WriteString("\n")
		}
		sv.indent--
	}

	if cmd.Filter != nil {
		sv.writeIndent()
		sv.buffer.WriteString("Filter: ")
		cmd.Filter.Accept(sv)
		sv.buffer.WriteString("\n")
	}

	if cmd.FieldOp != nil {
		sv.writeIndent()
		sv.buffer.WriteString("FieldOperation: ")
		cmd.FieldOp.Accept(sv)
		sv.buffer.WriteString("\n")
	}

	if cmd.Chain != nil {
		sv.writeIndent()
		sv.buffer.WriteString("Chain:\n")
		sv.indent++
		cmd.Chain.Accept(sv)
		sv.indent--
	}

	sv.indent--
	return nil
}

func (sv *StringVisitor) VisitFilter(filter *FilterExpr) interface{} {
	sv.buffer.WriteString("[")
	if filter.Condition != nil {
		filter.Condition.Accept(sv)
	}
	sv.buffer.WriteString("]")
	return nil
}

func (sv *StringVisitor) VisitFieldOperation(fieldOp *FieldOperation) interface{} {
	sv.buffer.WriteString(fieldOp.Field)
	if fieldOp.Op == "=" {
		sv.buffer.WriteString("=")
		fieldOp.Value.Accept(sv)
	}
	return nil
}

func (sv *StringVisitor) VisitValue(value *Value) interface{} {
	sv.buffer.WriteString(fmt.Sprintf("%s(%s)", value.Type.String(), value.Raw))
	return nil
}

func (sv *StringVisitor) VisitBinaryExpr(expr *BinaryExpr) interface{} {
	sv.buffer.WriteString("(")
	expr.Left.Accept(sv)
	sv.buffer.WriteString(fmt.Sprintf(" %s ", expr.Op))
	expr.Right.Accept(sv)
	sv.buffer.WriteString(")")
	return nil
}

func (sv *StringVisitor) VisitUnaryExpr(expr *UnaryExpr) interface{} {
	sv.buffer.WriteString(fmt.Sprintf("(%s ", expr.Op))
	expr.Expr.Accept(sv)
	sv.buffer.WriteString(")")
	return nil
}

func (sv *StringVisitor) VisitIdentifier(expr *IdentifierExpr) interface{} {
	sv.buffer.WriteString(expr.Name)
	return nil
}

func (sv *StringVisitor) VisitLiteral(expr *LiteralExpr) interface{} {
	return expr.Value.Accept(sv)
}

func (sv *StringVisitor) VisitFunctionCall(expr *FunctionCallExpr) interface{} {
	sv.buffer.WriteString(fmt.Sprintf("%s(", expr.Name))
	for i, arg := range expr.Args {
		if i > 0 {
			sv.buffer.WriteString(", ")
		}
		arg.Accept(sv)
	}
	sv.buffer.WriteString(")")
	return nil
}

func (sv *StringVisitor) VisitArray(expr *ArrayExpr) interface{} {
	sv.buffer.WriteString("[")
	for i, elem := range expr.Elements {
		if i > 0 {
			sv.buffer.WriteString(", ")
		}
		elem.Accept(sv)
	}
	sv.buffer.WriteString("]")
	return nil
}

func (sv *StringVisitor) VisitObject(expr *ObjectExpr) interface{} {
	sv.buffer.WriteString("{")
	i := 0
	for key, value := range expr.Fields {
		if i > 0 {
			sv.buffer.WriteString(", ")
		}
		sv.buffer.WriteString(fmt.Sprintf("%s: ", key))
		value.Accept(sv)
		i++
	}
	sv.buffer.WriteString("}")
	return nil
}

// ValidationVisitor validates AST nodes and collects errors
type ValidationVisitor struct {
	BaseVisitor
	errors []error
}

// NewValidationVisitor creates a new validation visitor
func NewValidationVisitor() *ValidationVisitor {
	return &ValidationVisitor{
		errors: make([]error, 0),
	}
}

// Errors returns all validation errors found
func (vv *ValidationVisitor) Errors() []error {
	return vv.errors
}

// HasErrors returns true if any validation errors were found
func (vv *ValidationVisitor) HasErrors() bool {
	return len(vv.errors) > 0
}

// Reset clears all collected errors
func (vv *ValidationVisitor) Reset() {
	vv.errors = vv.errors[:0]
}

func (vv *ValidationVisitor) addError(err error) {
	vv.errors = append(vv.errors, err)
}

func (vv *ValidationVisitor) VisitCommand(cmd *Command) interface{} {
	if err := cmd.Validate(); err != nil {
		vv.addError(fmt.Errorf("command validation failed: %w", err))
	}

	// Continue with base visitor behavior
	vv.BaseVisitor.VisitCommand(cmd)
	return nil
}

func (vv *ValidationVisitor) VisitFilter(filter *FilterExpr) interface{} {
	if err := filter.Validate(); err != nil {
		vv.addError(fmt.Errorf("filter validation failed: %w", err))
	}

	return vv.BaseVisitor.VisitFilter(filter)
}

func (vv *ValidationVisitor) VisitFieldOperation(fieldOp *FieldOperation) interface{} {
	if err := fieldOp.Validate(); err != nil {
		vv.addError(fmt.Errorf("field operation validation failed: %w", err))
	}

	return vv.BaseVisitor.VisitFieldOperation(fieldOp)
}

func (vv *ValidationVisitor) VisitValue(value *Value) interface{} {
	if err := value.Validate(); err != nil {
		vv.addError(fmt.Errorf("value validation failed: %w", err))
	}

	return vv.BaseVisitor.VisitValue(value)
}

func (vv *ValidationVisitor) VisitBinaryExpr(expr *BinaryExpr) interface{} {
	if err := expr.Validate(); err != nil {
		vv.addError(fmt.Errorf("binary expression validation failed: %w", err))
	}

	return vv.BaseVisitor.VisitBinaryExpr(expr)
}

func (vv *ValidationVisitor) VisitUnaryExpr(expr *UnaryExpr) interface{} {
	if err := expr.Validate(); err != nil {
		vv.addError(fmt.Errorf("unary expression validation failed: %w", err))
	}

	return vv.BaseVisitor.VisitUnaryExpr(expr)
}

func (vv *ValidationVisitor) VisitIdentifier(expr *IdentifierExpr) interface{} {
	if err := expr.Validate(); err != nil {
		vv.addError(fmt.Errorf("identifier validation failed: %w", err))
	}

	return vv.BaseVisitor.VisitIdentifier(expr)
}

func (vv *ValidationVisitor) VisitLiteral(expr *LiteralExpr) interface{} {
	if err := expr.Validate(); err != nil {
		vv.addError(fmt.Errorf("literal validation failed: %w", err))
	}

	return vv.BaseVisitor.VisitLiteral(expr)
}

func (vv *ValidationVisitor) VisitFunctionCall(expr *FunctionCallExpr) interface{} {
	if err := expr.Validate(); err != nil {
		vv.addError(fmt.Errorf("function call validation failed: %w", err))
	}

	return vv.BaseVisitor.VisitFunctionCall(expr)
}

func (vv *ValidationVisitor) VisitArray(expr *ArrayExpr) interface{} {
	if err := expr.Validate(); err != nil {
		vv.addError(fmt.Errorf("array validation failed: %w", err))
	}

	return vv.BaseVisitor.VisitArray(expr)
}

func (vv *ValidationVisitor) VisitObject(expr *ObjectExpr) interface{} {
	if err := expr.Validate(); err != nil {
		vv.addError(fmt.Errorf("object validation failed: %w", err))
	}

	return vv.BaseVisitor.VisitObject(expr)
}

// CollectorVisitor collects specific types of nodes from the AST
type CollectorVisitor struct {
	BaseVisitor
	Commands    []*Command
	Identifiers []*IdentifierExpr
	Literals    []*LiteralExpr
	Functions   []*FunctionCallExpr
}

// NewCollectorVisitor creates a new collector visitor
func NewCollectorVisitor() *CollectorVisitor {
	return &CollectorVisitor{
		Commands:    make([]*Command, 0),
		Identifiers: make([]*IdentifierExpr, 0),
		Literals:    make([]*LiteralExpr, 0),
		Functions:   make([]*FunctionCallExpr, 0),
	}
}

// Reset clears all collected nodes
func (cv *CollectorVisitor) Reset() {
	cv.Commands = cv.Commands[:0]
	cv.Identifiers = cv.Identifiers[:0]
	cv.Literals = cv.Literals[:0]
	cv.Functions = cv.Functions[:0]
}

func (cv *CollectorVisitor) VisitCommand(cmd *Command) interface{} {
	cv.Commands = append(cv.Commands, cmd)
	return cv.BaseVisitor.VisitCommand(cmd)
}

func (cv *CollectorVisitor) VisitIdentifier(expr *IdentifierExpr) interface{} {
	cv.Identifiers = append(cv.Identifiers, expr)
	return cv.BaseVisitor.VisitIdentifier(expr)
}

func (cv *CollectorVisitor) VisitLiteral(expr *LiteralExpr) interface{} {
	cv.Literals = append(cv.Literals, expr)
	return cv.BaseVisitor.VisitLiteral(expr)
}

func (cv *CollectorVisitor) VisitFunctionCall(expr *FunctionCallExpr) interface{} {
	cv.Functions = append(cv.Functions, expr)
	for _, arg := range expr.Args {
		arg.Accept(cv)
	}
	return nil
}

// Override all expression visitor methods to ensure collection
func (cv *CollectorVisitor) VisitBinaryExpr(expr *BinaryExpr) interface{} {
	if expr.Left != nil {
		expr.Left.Accept(cv)
	}
	if expr.Right != nil {
		expr.Right.Accept(cv)
	}
	return nil
}

func (cv *CollectorVisitor) VisitUnaryExpr(expr *UnaryExpr) interface{} {
	if expr.Expr != nil {
		return expr.Expr.Accept(cv)
	}
	return nil
}

func (cv *CollectorVisitor) VisitArray(expr *ArrayExpr) interface{} {
	for _, elem := range expr.Elements {
		elem.Accept(cv)
	}
	return nil
}

func (cv *CollectorVisitor) VisitObject(expr *ObjectExpr) interface{} {
	for _, value := range expr.Fields {
		value.Accept(cv)
	}
	return nil
}

// Utility functions for working with visitors

// ValidateAST validates an AST node and returns any validation errors
func ValidateAST(node Node) []error {
	visitor := NewValidationVisitor()
	node.Accept(visitor)
	return visitor.Errors()
}

// ASTToString converts an AST node to a formatted string representation
func ASTToString(node Node) string {
	visitor := NewStringVisitor()
	node.Accept(visitor)
	return visitor.String()
}

// CollectNodes collects specific types of nodes from an AST
func CollectNodes(node Node) *CollectorVisitor {
	visitor := NewCollectorVisitor()
	node.Accept(visitor)
	return visitor
}