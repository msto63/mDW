// File: doc.go
// Title: TCOL Parser Package Documentation
// Description: Implements the lexical analyzer and parser for TCOL commands.
//              Converts TCOL command strings into structured AST representations
//              with comprehensive error reporting and syntax validation.
// Author: msto63 with Claude Sonnet 4.0
// Version: v0.1.0
// Created: 2025-01-25
// Modified: 2025-01-25
//
// Change History:
// - 2025-01-25 v0.1.0: Initial parser implementation

/*
Package parser provides lexical analysis and parsing capabilities for TCOL commands.

This package implements a recursive descent parser that converts TCOL command
strings into Abstract Syntax Tree (AST) representations. It includes:

  • Lexical analyzer (tokenizer) for TCOL syntax
  • Recursive descent parser for TCOL grammar
  • Comprehensive error reporting with position information
  • Support for all TCOL language constructs

The parser follows TCOL grammar rules and produces well-formed AST nodes
that can be analyzed, optimized, and executed by other components.
*/
package parser