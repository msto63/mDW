// File: doc.go
// Title: TCOL Command Registry Package Documentation
// Description: Implements the command registry system for managing available
//              TCOL objects, methods, abbreviations, and aliases. Provides
//              registration, lookup, and validation services for the TCOL
//              execution engine.
// Author: msto63 with Claude Sonnet 4.0
// Version: v0.1.0
// Created: 2025-01-25
// Modified: 2025-01-25
//
// Change History:
// - 2025-01-25 v0.1.0: Initial registry implementation

/*
Package registry provides command registration and lookup services for TCOL.

This package manages the registry of available TCOL objects, methods,
abbreviations, and aliases. It provides:

  • Object and method registration
  • Command abbreviation expansion
  • Alias resolution and management
  • Service routing information
  • Validation of command availability

The registry serves as the central authority for what commands are available
in the TCOL system and how they should be resolved and routed.
*/
package registry