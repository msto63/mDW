// File: interface.go
// Title: TCOL Registry Interface
// Description: Defines the common interface for TCOL registry implementations
//              to enable abstraction and testing with different registry types.
// Author: msto63 with Claude Sonnet 4.0
// Version: v0.1.0
// Created: 2025-01-25
// Modified: 2025-01-25

package registry

import (
	"github.com/msto63/mDW/foundation/core/log"
)

// Options configures registry behavior
type Options struct {
	Logger              *log.Logger
	Services            []string
	EnableAbbreviations bool
	EnableAliases       bool
}

// ObjectDefinition defines a TCOL object with its methods
type ObjectDefinition struct {
	Name        string                    // Object name (e.g., "CUSTOMER")
	Description string                    // Object description
	Service     string                    // Service that handles this object
	Methods     map[string]*MethodDefinition // Available methods
	Fields      map[string]*FieldDefinition  // Object fields
}

// MethodDefinition defines a TCOL method
type MethodDefinition struct {
	Name        string                     // Method name (e.g., "CREATE")
	Description string                     // Method description
	Parameters  map[string]*ParameterDefinition // Method parameters
	Returns     string                     // Return type description
	Examples    []string                   // Usage examples
}

// ParameterDefinition defines a method parameter
type ParameterDefinition struct {
	Name        string   // Parameter name
	Type        string   // Parameter type (string, number, boolean, etc.)
	Required    bool     // Whether parameter is required
	Description string   // Parameter description
	Default     string   // Default value (if any)
	Values      []string // Allowed values (for enums)
}

// FieldDefinition defines an object field
type FieldDefinition struct {
	Name        string // Field name
	Type        string // Field type
	Description string // Field description
	Readable    bool   // Can be read
	Writable    bool   // Can be written
}

// RegistryInterface defines the common interface for TCOL registries
type RegistryInterface interface {
	// Object management
	RegisterObject(obj *ObjectDefinition) error
	HasObject(objectName string) bool
	GetObject(objectName string) (*ObjectDefinition, error)
	GetObjects() map[string]*ObjectDefinition
	GetObjectNames() []string

	// Method management
	HasMethod(objectName, methodName string) bool
	GetMethod(objectName, methodName string) (*MethodDefinition, error)
	GetMethodNames(objectName string) []string

	// Command validation
	ValidateCommand(objectName, methodName string) error

	// Alias management
	RegisterAlias(alias, command string) error
	ResolveAlias(alias string) string
	GetAliases() map[string]string

	// Abbreviation management
	ExpandAbbreviation(abbrev string) string
	GetAbbreviations() map[string]string

	// Service routing
	GetServiceForObject(objectName string) (string, error)
}

// Registry is an alias to the default registry implementation
type Registry = SimpleRegistry