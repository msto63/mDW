// File: registry_simple.go
// Title: Simplified TCOL Command Registry (Using Standard Errors)
// Description: A simplified version of the TCOL registry using standard Go
//              errors for faster development and testing. Will be enhanced
//              with foundation error handling later.
// Author: msto63 with Claude Sonnet 4.0
// Version: v0.1.0
// Created: 2025-01-25
// Modified: 2025-01-25

package registry

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"

	"github.com/msto63/mDW/foundation/core/log"
	mdwstringx "github.com/msto63/mDW/foundation/utils/stringx"
)

// SimpleRegistry is a simplified version of the TCOL registry
type SimpleRegistry struct {
	objects       map[string]*ObjectDefinition
	abbreviations map[string]string
	aliases       map[string]string
	services      map[string]string
	logger        *log.Logger
	mutex         sync.RWMutex
	options       Options
}

// NewSimple creates a new simplified TCOL registry
func NewSimple(opts Options) (*SimpleRegistry, error) {
	// Set defaults
	if opts.Logger == nil {
		opts.Logger = log.GetDefault()
	}

	registry := &SimpleRegistry{
		objects:       make(map[string]*ObjectDefinition),
		abbreviations: make(map[string]string),
		aliases:       make(map[string]string),
		services:      make(map[string]string),
		logger:        opts.Logger.WithField("component", "tcol-registry"),
		options:       opts,
	}

	// Register built-in objects and commands
	if err := registry.registerBuiltinCommands(); err != nil {
		return nil, fmt.Errorf("failed to register builtin commands: %w", err)
	}

	// Initialize abbreviations if enabled
	if opts.EnableAbbreviations {
		registry.initializeAbbreviations()
	}

	registry.logger.Info("TCOL registry initialized", log.Fields{
		"objectCount":          len(registry.objects),
		"serviceCount":         len(opts.Services),
		"enableAbbreviations": opts.EnableAbbreviations,
		"enableAliases":       opts.EnableAliases,
	})

	return registry, nil
}

// RegisterObject registers a new TCOL object
func (r *SimpleRegistry) RegisterObject(obj *ObjectDefinition) error {
	if obj == nil {
		return errors.New("object definition cannot be nil")
	}

	if mdwstringx.IsBlank(obj.Name) {
		return errors.New("object name cannot be empty")
	}

	r.mutex.Lock()
	defer r.mutex.Unlock()

	// Normalize object name to uppercase
	objName := strings.ToUpper(obj.Name)
	obj.Name = objName

	// Check if object already exists
	if _, exists := r.objects[objName]; exists {
		return fmt.Errorf("object %s already registered", objName)
	}

	// Validate methods
	if obj.Methods == nil {
		obj.Methods = make(map[string]*MethodDefinition)
	}

	for methodName, method := range obj.Methods {
		if method.Parameters == nil {
			method.Parameters = make(map[string]*ParameterDefinition)
		}
		
		// Normalize method name
		normalizedMethodName := strings.ToUpper(methodName)
		delete(obj.Methods, methodName)
		obj.Methods[normalizedMethodName] = method
		method.Name = normalizedMethodName
	}

	// Register object
	r.objects[objName] = obj

	// Register service mapping if provided
	if !mdwstringx.IsBlank(obj.Service) {
		r.services[objName] = obj.Service
	}

	r.logger.Info("TCOL object registered", log.Fields{
		"objectName":  objName,
		"service":     obj.Service,
		"methodCount": len(obj.Methods),
		"fieldCount":  len(obj.Fields),
	})

	// Update abbreviations
	if r.options.EnableAbbreviations {
		r.updateAbbreviations()
	}

	return nil
}

// RegisterAlias registers a command alias
func (r *SimpleRegistry) RegisterAlias(alias, command string) error {
	if !r.options.EnableAliases {
		return errors.New("aliases are disabled in this registry")
	}

	if mdwstringx.IsBlank(alias) {
		return errors.New("alias name cannot be empty")
	}

	if mdwstringx.IsBlank(command) {
		return errors.New("command cannot be empty")
	}

	r.mutex.Lock()
	defer r.mutex.Unlock()

	// Normalize alias
	alias = strings.ToUpper(alias)

	r.aliases[alias] = command

	r.logger.Debug("TCOL alias registered", log.Fields{
		"alias":   alias,
		"command": command,
	})

	return nil
}

// HasObject checks if an object is registered
func (r *SimpleRegistry) HasObject(objectName string) bool {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	_, exists := r.objects[strings.ToUpper(objectName)]
	return exists
}

// HasMethod checks if a method exists for an object
func (r *SimpleRegistry) HasMethod(objectName, methodName string) bool {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	obj, exists := r.objects[strings.ToUpper(objectName)]
	if !exists {
		return false
	}

	_, exists = obj.Methods[strings.ToUpper(methodName)]
	return exists
}

// GetObject returns an object definition
func (r *SimpleRegistry) GetObject(objectName string) (*ObjectDefinition, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	obj, exists := r.objects[strings.ToUpper(objectName)]
	if !exists {
		return nil, fmt.Errorf("object %s not found in registry", objectName)
	}

	return obj, nil
}

// GetMethod returns a method definition
func (r *SimpleRegistry) GetMethod(objectName, methodName string) (*MethodDefinition, error) {
	obj, err := r.GetObject(objectName)
	if err != nil {
		return nil, err
	}

	method, exists := obj.Methods[strings.ToUpper(methodName)]
	if !exists {
		return nil, fmt.Errorf("method %s not found for object %s", methodName, objectName)
	}

	return method, nil
}

// ExpandAbbreviation expands a command abbreviation to full form
func (r *SimpleRegistry) ExpandAbbreviation(abbrev string) string {
	if !r.options.EnableAbbreviations {
		return abbrev
	}

	r.mutex.RLock()
	defer r.mutex.RUnlock()

	if expanded, exists := r.abbreviations[strings.ToUpper(abbrev)]; exists {
		return expanded
	}

	return abbrev
}

// ResolveAlias resolves a command alias to the actual command
func (r *SimpleRegistry) ResolveAlias(alias string) string {
	if !r.options.EnableAliases {
		return alias
	}

	r.mutex.RLock()
	defer r.mutex.RUnlock()

	if command, exists := r.aliases[strings.ToUpper(alias)]; exists {
		return command
	}

	return alias
}

// GetServiceForObject returns the service name for an object
func (r *SimpleRegistry) GetServiceForObject(objectName string) (string, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	service, exists := r.services[strings.ToUpper(objectName)]
	if !exists {
		return "", fmt.Errorf("no service registered for object %s", objectName)
	}

	return service, nil
}

// GetAbbreviations returns all registered abbreviations
func (r *SimpleRegistry) GetAbbreviations() map[string]string {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	// Return a copy to prevent external modification
	abbrevs := make(map[string]string, len(r.abbreviations))
	for k, v := range r.abbreviations {
		abbrevs[k] = v
	}

	return abbrevs
}

// GetAliases returns all registered aliases
func (r *SimpleRegistry) GetAliases() map[string]string {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	// Return a copy to prevent external modification
	aliases := make(map[string]string, len(r.aliases))
	for k, v := range r.aliases {
		aliases[k] = v
	}

	return aliases
}

// GetObjects returns all registered objects
func (r *SimpleRegistry) GetObjects() map[string]*ObjectDefinition {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	// Return a copy to prevent external modification
	objects := make(map[string]*ObjectDefinition, len(r.objects))
	for k, v := range r.objects {
		objects[k] = v
	}

	return objects
}

// ValidateCommand validates that a command exists in the registry
func (r *SimpleRegistry) ValidateCommand(objectName, methodName string) error {
	// Expand abbreviations first
	if r.options.EnableAbbreviations {
		fullCommand := r.ExpandAbbreviation(fmt.Sprintf("%s.%s", objectName, methodName))
		if fullCommand != fmt.Sprintf("%s.%s", objectName, methodName) {
			parts := strings.Split(fullCommand, ".")
			if len(parts) == 2 {
				objectName = parts[0]
				methodName = parts[1]
			}
		}
	}

	// Check if object exists
	if !r.HasObject(objectName) {
		return fmt.Errorf("unknown object: %s", objectName)
	}

	// Check if method exists
	if !r.HasMethod(objectName, methodName) {
		return fmt.Errorf("unknown method %s for object %s", methodName, objectName)
	}

	return nil
}

// GetObjectNames returns sorted list of object names
func (r *SimpleRegistry) GetObjectNames() []string {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	names := make([]string, 0, len(r.objects))
	for name := range r.objects {
		names = append(names, name)
	}

	sort.Strings(names)
	return names
}

// GetMethodNames returns sorted list of method names for an object
func (r *SimpleRegistry) GetMethodNames(objectName string) []string {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	obj, exists := r.objects[strings.ToUpper(objectName)]
	if !exists {
		return []string{}
	}

	names := make([]string, 0, len(obj.Methods))
	for name := range obj.Methods {
		names = append(names, name)
	}

	sort.Strings(names)
	return names
}

// Helper methods from original implementation

func (r *SimpleRegistry) registerBuiltinCommands() error {
	// Register ALIAS object for managing aliases
	aliasObj := &ObjectDefinition{
		Name:        "ALIAS",
		Description: "Manage command aliases",
		Service:     "tcol-internal",
		Methods: map[string]*MethodDefinition{
			"CREATE": {
				Name:        "CREATE",
				Description: "Create a new alias",
				Parameters: map[string]*ParameterDefinition{
					"name": {
						Name:        "name",
						Type:        "string",
						Required:    true,
						Description: "Alias name",
					},
					"command": {
						Name:        "command",
						Type:        "string",
						Required:    true,
						Description: "Command to alias",
					},
				},
				Examples: []string{
					`ALIAS.CREATE name="uc" command="CUSTOMER.LIST status=unpaid"`,
				},
			},
			"DELETE": {
				Name:        "DELETE",
				Description: "Delete an alias",
				Parameters: map[string]*ParameterDefinition{
					"name": {
						Name:        "name",
						Type:        "string",
						Required:    true,
						Description: "Alias name to delete",
					},
				},
			},
			"LIST": {
				Name:        "LIST",
				Description: "List all aliases",
			},
		},
	}

	if err := r.RegisterObject(aliasObj); err != nil {
		return fmt.Errorf("failed to register ALIAS object: %w", err)
	}

	// Register HELP object for documentation
	helpObj := &ObjectDefinition{
		Name:        "HELP",
		Description: "Get help information",
		Service:     "tcol-internal",
		Methods: map[string]*MethodDefinition{
			"OBJECT": {
				Name:        "OBJECT",
				Description: "Get help for an object",
				Parameters: map[string]*ParameterDefinition{
					"name": {
						Name:        "name",
						Type:        "string",
						Required:    true,
						Description: "Object name",
					},
				},
			},
			"METHOD": {
				Name:        "METHOD",
				Description: "Get help for a method",
				Parameters: map[string]*ParameterDefinition{
					"object": {
						Name:        "object",
						Type:        "string",
						Required:    true,
						Description: "Object name",
					},
					"method": {
						Name:        "method",
						Type:        "string",
						Required:    true,
						Description: "Method name",
					},
				},
			},
			"LIST": {
				Name:        "LIST",
				Description: "List all available objects",
			},
		},
	}

	if err := r.RegisterObject(helpObj); err != nil {
		return fmt.Errorf("failed to register HELP object: %w", err)
	}

	return nil
}

func (r *SimpleRegistry) initializeAbbreviations() {
	// Common object abbreviations
	objectAbbrevs := map[string]string{
		"CUST": "CUSTOMER",
		"INV":  "INVOICE",
		"ORD":  "ORDER",
		"PROD": "PRODUCT",
		"USR":  "USER",
	}

	// Common method abbreviations
	methodAbbrevs := map[string]string{
		"CR":  "CREATE",
		"LS":  "LIST",
		"UPD": "UPDATE",
		"DEL": "DELETE",
		"SH":  "SHOW",
		"GET": "GET",
		"SET": "SET",
	}

	// Generate combinations
	for objAbbrev, objFull := range objectAbbrevs {
		for methodAbbrev, methodFull := range methodAbbrevs {
			abbrev := fmt.Sprintf("%s.%s", objAbbrev, methodAbbrev)
			full := fmt.Sprintf("%s.%s", objFull, methodFull)
			r.abbreviations[abbrev] = full
		}
	}
}

func (r *SimpleRegistry) updateAbbreviations() {
	// Generate abbreviations for all registered objects and methods
	for objName, obj := range r.objects {
		// Generate object abbreviations (first 3-4 characters)
		objAbbrev := r.generateAbbreviation(objName)
		
		for methodName := range obj.Methods {
			// Generate method abbreviations
			methodAbbrev := r.generateAbbreviation(methodName)
			
			// Create full abbreviation
			abbrev := fmt.Sprintf("%s.%s", objAbbrev, methodAbbrev)
			full := fmt.Sprintf("%s.%s", objName, methodName)
			
			r.abbreviations[abbrev] = full
		}
	}
}

func (r *SimpleRegistry) generateAbbreviation(name string) string {
	if len(name) <= 3 {
		return name
	}

	// Try consonants first
	var abbrev strings.Builder
	for i, ch := range name {
		if i == 0 || !r.isVowel(ch) {
			abbrev.WriteRune(ch)
			if abbrev.Len() >= 3 {
				break
			}
		}
	}

	if abbrev.Len() >= 3 {
		return abbrev.String()
	}

	// Fallback to first N characters
	if len(name) >= 4 {
		return name[:4]
	}
	return name[:3]
}

func (r *SimpleRegistry) isVowel(ch rune) bool {
	vowels := "AEIOU"
	return strings.ContainsRune(vowels, ch)
}