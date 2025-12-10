// File: registry_test.go
// Title: TCOL Registry Unit Tests
// Description: Comprehensive unit tests for the TCOL registry system including
//              object registration, method definitions, abbreviations, aliases,
//              service mappings, and validation. Tests cover both positive and
//              negative scenarios with comprehensive error handling.
// Author: msto63 with Claude Sonnet 4.0
// Version: v0.1.0
// Created: 2025-01-25
// Modified: 2025-01-25
//
// Change History:
// - 2025-01-25 v0.1.0: Initial comprehensive registry test suite

package registry

import (
	"fmt"
	"sort"
	"strings"
	"testing"

	mdwlog "github.com/msto63/mDW/foundation/core/log"
)

func TestNewSimple(t *testing.T) {
	tests := []struct {
		name      string
		options   Options
		expectErr bool
		checkFunc func(*SimpleRegistry) bool
	}{
		{
			name: "Default options",
			options: Options{
				Logger: mdwlog.GetDefault(),
			},
			expectErr: false,
			checkFunc: func(r *SimpleRegistry) bool {
				return len(r.objects) >= 2 // ALIAS and HELP built-in objects
			},
		},
		{
			name: "With abbreviations enabled",
			options: Options{
				Logger:              mdwlog.GetDefault(),
				EnableAbbreviations: true,
			},
			expectErr: false,
			checkFunc: func(r *SimpleRegistry) bool {
				return len(r.abbreviations) > 0
			},
		},
		{
			name: "With aliases enabled",
			options: Options{
				Logger:        mdwlog.GetDefault(),
				EnableAliases: true,
			},
			expectErr: false,
			checkFunc: func(r *SimpleRegistry) bool {
				return r.options.EnableAliases
			},
		},
		{
			name: "With services list",
			options: Options{
				Logger:   mdwlog.GetDefault(),
				Services: []string{"customer-service", "invoice-service"},
			},
			expectErr: false,
			checkFunc: func(r *SimpleRegistry) bool {
				return len(r.options.Services) == 2
			},
		},
		{
			name: "Nil logger (should use default)",
			options: Options{
				Logger: nil,
			},
			expectErr: false,
			checkFunc: func(r *SimpleRegistry) bool {
				return r.logger != nil
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			registry, err := NewSimple(tt.options)

			if tt.expectErr {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if registry == nil {
				t.Fatal("Expected registry but got nil")
			}

			if tt.checkFunc != nil && !tt.checkFunc(registry) {
				t.Error("Registry check function failed")
			}
		})
	}
}

func TestSimpleRegistry_RegisterObject(t *testing.T) {
	registry, err := NewSimple(Options{
		Logger:              mdwlog.GetDefault(),
		EnableAbbreviations: true,
	})
	if err != nil {
		t.Fatalf("Failed to create registry: %v", err)
	}

	tests := []struct {
		name      string
		object    *ObjectDefinition
		expectErr bool
		errMsg    string
	}{
		{
			name: "Valid object registration",
			object: &ObjectDefinition{
				Name:        "CUSTOMER",
				Description: "Customer management",
				Service:     "customer-service",
				Methods: map[string]*MethodDefinition{
					"CREATE": {
						Name:        "CREATE",
						Description: "Create customer",
						Parameters: map[string]*ParameterDefinition{
							"name": {
								Name:        "name",
								Type:        "string",
								Required:    true,
								Description: "Customer name",
							},
						},
					},
				},
				Fields: map[string]*FieldDefinition{
					"id": {
						Name:        "id",
						Type:        "string",
						Description: "Customer ID",
					},
				},
			},
			expectErr: false,
		},
		{
			name:      "Nil object",
			object:    nil,
			expectErr: true,
			errMsg:    "object definition cannot be nil",
		},
		{
			name: "Empty object name",
			object: &ObjectDefinition{
				Name: "",
			},
			expectErr: true,
			errMsg:    "object name cannot be empty",
		},
		{
			name: "Blank object name",
			object: &ObjectDefinition{
				Name: "   ",
			},
			expectErr: true,
			errMsg:    "object name cannot be empty",
		},
		{
			name: "Object with minimal info",
			object: &ObjectDefinition{
				Name: "PRODUCT",
			},
			expectErr: false,
		},
		{
			name: "Object with mixed case name (should normalize)",
			object: &ObjectDefinition{
				Name: "InVoIcE",
				Methods: map[string]*MethodDefinition{
					"create": {
						Name: "create",
					},
				},
			},
			expectErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := registry.RegisterObject(tt.object)

			if tt.expectErr {
				if err == nil {
					t.Errorf("Expected error but got none")
				} else if tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("Expected error containing %q, got %q", tt.errMsg, err.Error())
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			// Verify object was registered
			if tt.object != nil {
				expectedName := strings.ToUpper(tt.object.Name)
				if !registry.HasObject(expectedName) {
					t.Errorf("Object %s was not registered", expectedName)
				}

				// Verify methods were normalized
				obj, _ := registry.GetObject(expectedName)
				for methodName := range obj.Methods {
					if methodName != strings.ToUpper(methodName) {
						t.Errorf("Method name %s was not normalized to uppercase", methodName)
					}
				}
			}
		})
	}

	// Test duplicate registration
	t.Run("Duplicate object registration", func(t *testing.T) {
		obj := &ObjectDefinition{
			Name: "DUPLICATE",
		}

		// First registration should succeed
		err := registry.RegisterObject(obj)
		if err != nil {
			t.Fatalf("First registration failed: %v", err)
		}

		// Second registration should fail
		err = registry.RegisterObject(obj)
		if err == nil {
			t.Error("Expected error for duplicate registration")
		} else if !strings.Contains(err.Error(), "already registered") {
			t.Errorf("Expected 'already registered' error, got: %v", err)
		}
	})
}

func TestSimpleRegistry_RegisterAlias(t *testing.T) {
	tests := []struct {
		name      string
		enableAliases bool
		alias     string
		command   string
		expectErr bool
		errMsg    string
	}{
		{
			name:          "Valid alias with aliases enabled",
			enableAliases: true,
			alias:         "uc",
			command:       "CUSTOMER.LIST status=unpaid",
			expectErr:     false,
		},
		{
			name:          "Aliases disabled",
			enableAliases: false,
			alias:         "uc",
			command:       "CUSTOMER.LIST",
			expectErr:     true,
			errMsg:        "aliases are disabled",
		},
		{
			name:          "Empty alias name",
			enableAliases: true,
			alias:         "",
			command:       "CUSTOMER.LIST",
			expectErr:     true,
			errMsg:        "alias name cannot be empty",
		},
		{
			name:          "Blank alias name",
			enableAliases: true,
			alias:         "   ",
			command:       "CUSTOMER.LIST",
			expectErr:     true,
			errMsg:        "alias name cannot be empty",
		},
		{
			name:          "Empty command",
			enableAliases: true,
			alias:         "empty",
			command:       "",
			expectErr:     true,
			errMsg:        "command cannot be empty",
		},
		{
			name:          "Blank command",
			enableAliases: true,
			alias:         "blank",
			command:       "   ",
			expectErr:     true,
			errMsg:        "command cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			registry, err := NewSimple(Options{
				Logger:        mdwlog.GetDefault(),
				EnableAliases: tt.enableAliases,
			})
			if err != nil {
				t.Fatalf("Failed to create registry: %v", err)
			}

			err = registry.RegisterAlias(tt.alias, tt.command)

			if tt.expectErr {
				if err == nil {
					t.Errorf("Expected error but got none")
				} else if tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("Expected error containing %q, got %q", tt.errMsg, err.Error())
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			// Verify alias was registered
			resolved := registry.ResolveAlias(tt.alias)
			if resolved != tt.command {
				t.Errorf("Expected resolved command %q, got %q", tt.command, resolved)
			}
		})
	}
}

func TestSimpleRegistry_HasObject(t *testing.T) {
	registry, err := NewSimple(Options{
		Logger: mdwlog.GetDefault(),
	})
	if err != nil {
		t.Fatalf("Failed to create registry: %v", err)
	}

	// Register test object
	testObj := &ObjectDefinition{
		Name: "TESTOBJECT",
	}
	registry.RegisterObject(testObj)

	tests := []struct {
		name       string
		objectName string
		expected   bool
	}{
		{
			name:       "Existing object (uppercase)",
			objectName: "TESTOBJECT",
			expected:   true,
		},
		{
			name:       "Existing object (lowercase)",
			objectName: "testobject",
			expected:   true,
		},
		{
			name:       "Existing object (mixed case)",
			objectName: "TestObject",
			expected:   true,
		},
		{
			name:       "Non-existing object",
			objectName: "NONEXISTENT",
			expected:   false,
		},
		{
			name:       "Built-in ALIAS object",
			objectName: "ALIAS",
			expected:   true,
		},
		{
			name:       "Built-in HELP object",
			objectName: "HELP",
			expected:   true,
		},
		{
			name:       "Empty object name",
			objectName: "",
			expected:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := registry.HasObject(tt.objectName)
			if result != tt.expected {
				t.Errorf("HasObject(%q) = %v, expected %v", tt.objectName, result, tt.expected)
			}
		})
	}
}

func TestSimpleRegistry_HasMethod(t *testing.T) {
	registry, err := NewSimple(Options{
		Logger: mdwlog.GetDefault(),
	})
	if err != nil {
		t.Fatalf("Failed to create registry: %v", err)
	}

	// Register test object with methods
	testObj := &ObjectDefinition{
		Name: "TESTOBJECT",
		Methods: map[string]*MethodDefinition{
			"CREATE": {Name: "CREATE"},
			"LIST":   {Name: "LIST"},
		},
	}
	registry.RegisterObject(testObj)

	tests := []struct {
		name       string
		objectName string
		methodName string
		expected   bool
	}{
		{
			name:       "Existing method (uppercase)",
			objectName: "TESTOBJECT",
			methodName: "CREATE",
			expected:   true,
		},
		{
			name:       "Existing method (lowercase)",
			objectName: "testobject",
			methodName: "create",
			expected:   true,
		},
		{
			name:       "Existing method (mixed case)",
			objectName: "TestObject",
			methodName: "Create",
			expected:   true,
		},
		{
			name:       "Non-existing method",
			objectName: "TESTOBJECT",
			methodName: "DELETE",
			expected:   false,
		},
		{
			name:       "Non-existing object",
			objectName: "NONEXISTENT",
			methodName: "CREATE",
			expected:   false,
		},
		{
			name:       "Built-in ALIAS.CREATE",
			objectName: "ALIAS",
			methodName: "CREATE",
			expected:   true,
		},
		{
			name:       "Built-in HELP.LIST",
			objectName: "HELP",
			methodName: "LIST",
			expected:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := registry.HasMethod(tt.objectName, tt.methodName)
			if result != tt.expected {
				t.Errorf("HasMethod(%q, %q) = %v, expected %v", tt.objectName, tt.methodName, result, tt.expected)
			}
		})
	}
}

func TestSimpleRegistry_GetObject(t *testing.T) {
	registry, err := NewSimple(Options{
		Logger: mdwlog.GetDefault(),
	})
	if err != nil {
		t.Fatalf("Failed to create registry: %v", err)
	}

	// Register test object
	testObj := &ObjectDefinition{
		Name:        "CUSTOMER",
		Description: "Customer management",
		Service:     "customer-service",
	}
	registry.RegisterObject(testObj)

	tests := []struct {
		name       string
		objectName string
		expectErr  bool
		checkFunc  func(*ObjectDefinition) bool
	}{
		{
			name:       "Get existing object",
			objectName: "CUSTOMER",
			expectErr:  false,
			checkFunc: func(obj *ObjectDefinition) bool {
				return obj.Name == "CUSTOMER" && obj.Service == "customer-service"
			},
		},
		{
			name:       "Get existing object (case insensitive)",
			objectName: "customer",
			expectErr:  false,
			checkFunc: func(obj *ObjectDefinition) bool {
				return obj.Name == "CUSTOMER"
			},
		},
		{
			name:       "Get non-existing object",
			objectName: "NONEXISTENT",
			expectErr:  true,
		},
		{
			name:       "Get built-in ALIAS object",
			objectName: "ALIAS",
			expectErr:  false,
			checkFunc: func(obj *ObjectDefinition) bool {
				return obj.Name == "ALIAS" && len(obj.Methods) >= 3
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			obj, err := registry.GetObject(tt.objectName)

			if tt.expectErr {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if obj == nil {
				t.Fatal("Expected object but got nil")
			}

			if tt.checkFunc != nil && !tt.checkFunc(obj) {
				t.Error("Object check function failed")
			}
		})
	}
}

func TestSimpleRegistry_GetMethod(t *testing.T) {
	registry, err := NewSimple(Options{
		Logger: mdwlog.GetDefault(),
	})
	if err != nil {
		t.Fatalf("Failed to create registry: %v", err)
	}

	// Register test object with methods
	testObj := &ObjectDefinition{
		Name: "CUSTOMER",
		Methods: map[string]*MethodDefinition{
			"CREATE": {
				Name:        "CREATE",
				Description: "Create customer",
				Parameters: map[string]*ParameterDefinition{
					"name": {
						Name:     "name",
						Type:     "string",
						Required: true,
					},
				},
			},
		},
	}
	registry.RegisterObject(testObj)

	tests := []struct {
		name       string
		objectName string
		methodName string
		expectErr  bool
		checkFunc  func(*MethodDefinition) bool
	}{
		{
			name:       "Get existing method",
			objectName: "CUSTOMER",
			methodName: "CREATE",
			expectErr:  false,
			checkFunc: func(method *MethodDefinition) bool {
				return method.Name == "CREATE" && len(method.Parameters) == 1
			},
		},
		{
			name:       "Get existing method (case insensitive)",
			objectName: "customer",
			methodName: "create",
			expectErr:  false,
			checkFunc: func(method *MethodDefinition) bool {
				return method.Name == "CREATE"
			},
		},
		{
			name:       "Get non-existing method",
			objectName: "CUSTOMER",
			methodName: "DELETE",
			expectErr:  true,
		},
		{
			name:       "Get method from non-existing object",
			objectName: "NONEXISTENT",
			methodName: "CREATE",
			expectErr:  true,
		},
		{
			name:       "Get built-in ALIAS.CREATE method",
			objectName: "ALIAS",
			methodName: "CREATE",
			expectErr:  false,
			checkFunc: func(method *MethodDefinition) bool {
				return method.Name == "CREATE" && len(method.Parameters) == 2
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			method, err := registry.GetMethod(tt.objectName, tt.methodName)

			if tt.expectErr {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if method == nil {
				t.Fatal("Expected method but got nil")
			}

			if tt.checkFunc != nil && !tt.checkFunc(method) {
				t.Error("Method check function failed")
			}
		})
	}
}

func TestSimpleRegistry_Abbreviations(t *testing.T) {
	registry, err := NewSimple(Options{
		Logger:              mdwlog.GetDefault(),
		EnableAbbreviations: true,
	})
	if err != nil {
		t.Fatalf("Failed to create registry: %v", err)
	}

	// Register test object to generate abbreviations
	testObj := &ObjectDefinition{
		Name: "CUSTOMER",
		Methods: map[string]*MethodDefinition{
			"CREATE": {Name: "CREATE"},
			"LIST":   {Name: "LIST"},
		},
	}
	registry.RegisterObject(testObj)

	tests := []struct {
		name     string
		abbrev   string
		expected string
	}{
		{
			name:     "Pre-defined abbreviation CUST.CR",
			abbrev:   "CUST.CR",
			expected: "CUSTOMER.CREATE",
		},
		{
			name:     "Pre-defined abbreviation CUST.LS",  
			abbrev:   "CUST.LS",
			expected: "CUSTOMER.LIST",
		},
		{
			name:     "Non-existing abbreviation",
			abbrev:   "NONEXIST.TEST",
			expected: "NONEXIST.TEST",
		},
		{
			name:     "Case insensitive abbreviation",
			abbrev:   "cust.cr",
			expected: "CUSTOMER.CREATE",
		},
		{
			name:     "Empty abbreviation",
			abbrev:   "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := registry.ExpandAbbreviation(tt.abbrev)
			if result != tt.expected {
				t.Errorf("ExpandAbbreviation(%q) = %q, expected %q", tt.abbrev, result, tt.expected)
			}
		})
	}

	// Test GetAbbreviations
	t.Run("GetAbbreviations", func(t *testing.T) {
		abbrevs := registry.GetAbbreviations()
		if len(abbrevs) == 0 {
			t.Error("Expected abbreviations but got empty map")
		}

		// Check that we got a copy (not the original map)
		originalLen := len(abbrevs)
		abbrevs["TEST"] = "MODIFIED"
		newAbbrevs := registry.GetAbbreviations()
		if len(newAbbrevs) != originalLen {
			t.Error("GetAbbreviations should return a copy, not the original map")
		}
	})

	// Test with abbreviations disabled
	t.Run("Abbreviations disabled", func(t *testing.T) {
		disabledRegistry, err := NewSimple(Options{
			Logger:              mdwlog.GetDefault(),
			EnableAbbreviations: false,
		})
		if err != nil {
			t.Fatalf("Failed to create registry: %v", err)
		}

		result := disabledRegistry.ExpandAbbreviation("CUST.CR")
		if result != "CUST.CR" {
			t.Errorf("Expected no expansion when abbreviations disabled, got %q", result)
		}
	})
}

func TestSimpleRegistry_Aliases(t *testing.T) {
	registry, err := NewSimple(Options{
		Logger:        mdwlog.GetDefault(),
		EnableAliases: true,
	})
	if err != nil {
		t.Fatalf("Failed to create registry: %v", err)
	}

	// Register some test aliases
	testAliases := map[string]string{
		"uc":       "CUSTOMER.LIST status=unpaid",
		"newcust":  "CUSTOMER.CREATE",
		"listall":  "CUSTOMER.LIST",
	}

	for alias, command := range testAliases {
		err := registry.RegisterAlias(alias, command)
		if err != nil {
			t.Fatalf("Failed to register alias %s: %v", alias, err)
		}
	}

	tests := []struct {
		name     string
		alias    string
		expected string
	}{
		{
			name:     "Resolve existing alias",
			alias:    "uc",
			expected: "CUSTOMER.LIST status=unpaid",
		},
		{
			name:     "Resolve existing alias (case insensitive)",
			alias:    "UC",
			expected: "CUSTOMER.LIST status=unpaid",
		},
		{
			name:     "Resolve non-existing alias",
			alias:    "nonexistent",
			expected: "nonexistent",
		},
		{
			name:     "Empty alias",
			alias:    "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := registry.ResolveAlias(tt.alias)
			if result != tt.expected {
				t.Errorf("ResolveAlias(%q) = %q, expected %q", tt.alias, result, tt.expected)
			}
		})
	}

	// Test GetAliases
	t.Run("GetAliases", func(t *testing.T) {
		aliases := registry.GetAliases()
		if len(aliases) != len(testAliases) {
			t.Errorf("Expected %d aliases, got %d", len(testAliases), len(aliases))
		}

		// Check that we got a copy
		originalLen := len(aliases)
		aliases["TEST"] = "MODIFIED"
		newAliases := registry.GetAliases()
		if len(newAliases) != originalLen {
			t.Error("GetAliases should return a copy, not the original map")
		}

		// Verify alias normalization (should be uppercase)
		for alias := range aliases {
			if alias != strings.ToUpper(alias) {
				t.Errorf("Alias %q should be normalized to uppercase", alias)
			}
		}
	})

	// Test with aliases disabled
	t.Run("Aliases disabled", func(t *testing.T) {
		disabledRegistry, err := NewSimple(Options{
			Logger:        mdwlog.GetDefault(),
			EnableAliases: false,
		})
		if err != nil {
			t.Fatalf("Failed to create registry: %v", err)
		}

		result := disabledRegistry.ResolveAlias("uc")
		if result != "uc" {
			t.Errorf("Expected no resolution when aliases disabled, got %q", result)
		}
	})
}

func TestSimpleRegistry_ServiceMapping(t *testing.T) {
	registry, err := NewSimple(Options{
		Logger: mdwlog.GetDefault(),
	})
	if err != nil {
		t.Fatalf("Failed to create registry: %v", err)
	}

	// Register objects with different service mappings
	testObjects := []*ObjectDefinition{
		{
			Name:    "CUSTOMER",
			Service: "customer-service",
		},
		{
			Name:    "INVOICE",
			Service: "billing-service",
		},
		{
			Name:    "NOSVC",
			// No service specified
		},
	}

	for _, obj := range testObjects {
		err := registry.RegisterObject(obj)
		if err != nil {
			t.Fatalf("Failed to register object %s: %v", obj.Name, err)
		}
	}

	tests := []struct {
		name       string
		objectName string
		expectErr  bool
		expected   string
	}{
		{
			name:       "Get service for object with service",
			objectName: "CUSTOMER",
			expectErr:  false,
			expected:   "customer-service",
		},
		{
			name:       "Get service for object with service (case insensitive)",
			objectName: "customer",
			expectErr:  false,
			expected:   "customer-service",
		},
		{
			name:       "Get service for object without service",
			objectName: "NOSVC",
			expectErr:  true,
		},
		{
			name:       "Get service for non-existing object",
			objectName: "NONEXISTENT",
			expectErr:  true,
		},
		{
			name:       "Get service for built-in object",
			objectName: "ALIAS",
			expectErr:  false,
			expected:   "tcol-internal",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service, err := registry.GetServiceForObject(tt.objectName)

			if tt.expectErr {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if service != tt.expected {
				t.Errorf("GetServiceForObject(%q) = %q, expected %q", tt.objectName, service, tt.expected)
			}
		})
	}
}

func TestSimpleRegistry_ValidateCommand(t *testing.T) {
	registry, err := NewSimple(Options{
		Logger:              mdwlog.GetDefault(),
		EnableAbbreviations: true,
	})
	if err != nil {
		t.Fatalf("Failed to create registry: %v", err)
	}

	// Register test object
	testObj := &ObjectDefinition{
		Name: "CUSTOMER",
		Methods: map[string]*MethodDefinition{
			"CREATE": {Name: "CREATE"},
			"LIST":   {Name: "LIST"},
		},
	}
	registry.RegisterObject(testObj)

	tests := []struct {
		name       string
		objectName string
		methodName string
		expectErr  bool
		errMsg     string
	}{
		{
			name:       "Valid command",
			objectName: "CUSTOMER",
			methodName: "CREATE",
			expectErr:  false,
		},
		{
			name:       "Valid command (case insensitive)",
			objectName: "customer",
			methodName: "create",
			expectErr:  false,
		},
		{
			name:       "Valid abbreviation",
			objectName: "CUST",
			methodName: "CR",
			expectErr:  false,
		},
		{
			name:       "Unknown object",
			objectName: "UNKNOWN",
			methodName: "CREATE",
			expectErr:  true,
			errMsg:     "unknown object",
		},
		{
			name:       "Unknown method",
			objectName: "CUSTOMER",
			methodName: "DELETE",
			expectErr:  true,
			errMsg:     "unknown method",
		},
		{
			name:       "Built-in command",
			objectName: "ALIAS",
			methodName: "CREATE",
			expectErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := registry.ValidateCommand(tt.objectName, tt.methodName)

			if tt.expectErr {
				if err == nil {
					t.Errorf("Expected error but got none")
				} else if tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("Expected error containing %q, got %q", tt.errMsg, err.Error())
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}

func TestSimpleRegistry_GetObjectNames(t *testing.T) {
	registry, err := NewSimple(Options{
		Logger: mdwlog.GetDefault(),
	})
	if err != nil {
		t.Fatalf("Failed to create registry: %v", err)
	}

	// Register additional objects
	testObjects := []string{"CUSTOMER", "PRODUCT", "INVOICE"}
	for _, name := range testObjects {
		obj := &ObjectDefinition{Name: name}
		registry.RegisterObject(obj)
	}

	names := registry.GetObjectNames()

	// Check that all registered objects are included
	expectedNames := append(testObjects, "ALIAS", "HELP") // Built-in objects
	if len(names) != len(expectedNames) {
		t.Errorf("Expected %d object names, got %d", len(expectedNames), len(names))
	}

	// Check that names are sorted
	if !sort.StringsAreSorted(names) {
		t.Error("Object names should be sorted")
	}

	// Check that all expected names are present
	nameMap := make(map[string]bool)
	for _, name := range names {
		nameMap[name] = true
	}

	for _, expectedName := range expectedNames {
		if !nameMap[expectedName] {
			t.Errorf("Expected object name %q not found in result", expectedName)
		}
	}
}

func TestSimpleRegistry_GetMethodNames(t *testing.T) {
	registry, err := NewSimple(Options{
		Logger: mdwlog.GetDefault(),
	})
	if err != nil {
		t.Fatalf("Failed to create registry: %v", err)
	}

	// Register test object with methods
	testObj := &ObjectDefinition{
		Name: "CUSTOMER",
		Methods: map[string]*MethodDefinition{
			"CREATE": {Name: "CREATE"},
			"UPDATE": {Name: "UPDATE"},
			"DELETE": {Name: "DELETE"},
			"LIST":   {Name: "LIST"},
		},
	}
	registry.RegisterObject(testObj)

	tests := []struct {
		name       string
		objectName string
		expected   []string
	}{
		{
			name:       "Get methods for existing object",
			objectName: "CUSTOMER",
			expected:   []string{"CREATE", "DELETE", "LIST", "UPDATE"}, // Should be sorted
		},
		{
			name:       "Get methods for existing object (case insensitive)",
			objectName: "customer",
			expected:   []string{"CREATE", "DELETE", "LIST", "UPDATE"},
		},
		{
			name:       "Get methods for non-existing object",
			objectName: "NONEXISTENT",
			expected:   []string{},
		},
		{
			name:       "Get methods for built-in ALIAS object",
			objectName: "ALIAS",
			expected:   []string{"CREATE", "DELETE", "LIST"}, // Built-in methods
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			methods := registry.GetMethodNames(tt.objectName)

			if len(methods) != len(tt.expected) {
				t.Errorf("Expected %d methods, got %d", len(tt.expected), len(methods))
				t.Errorf("Expected: %v", tt.expected)
				t.Errorf("Got: %v", methods)
				return
			}

			// Check that methods are sorted
			if !sort.StringsAreSorted(methods) {
				t.Error("Method names should be sorted")
			}

			// Check that all expected methods are present
			for i, expected := range tt.expected {
				if i >= len(methods) || methods[i] != expected {
					t.Errorf("Expected method %q at position %d, got %q", expected, i, methods[i])
				}
			}
		})
	}
}

func TestSimpleRegistry_GetObjects(t *testing.T) {
	registry, err := NewSimple(Options{
		Logger: mdwlog.GetDefault(),
	})
	if err != nil {
		t.Fatalf("Failed to create registry: %v", err)
	}

	// Register test object
	testObj := &ObjectDefinition{
		Name:        "TESTOBJ",
		Description: "Test object",
		Service:     "test-service",
	}
	registry.RegisterObject(testObj)

	objects := registry.GetObjects()

	// Check that our test object is included
	if testObjResult, exists := objects["TESTOBJ"]; !exists {
		t.Error("Test object not found in results")
	} else {
		if testObjResult.Description != "Test object" {
			t.Errorf("Expected description 'Test object', got %q", testObjResult.Description)
		}
	}

	// Check that built-in objects are included
	if _, exists := objects["ALIAS"]; !exists {
		t.Error("Built-in ALIAS object not found")
	}

	if _, exists := objects["HELP"]; !exists {
		t.Error("Built-in HELP object not found")
	}

	// Check that we got a copy (not the original map)
	originalLen := len(objects)
	objects["MODIFIED"] = &ObjectDefinition{Name: "MODIFIED"}
	newObjects := registry.GetObjects()
	if len(newObjects) != originalLen {
		t.Error("GetObjects should return a copy, not the original map")
	}
}

func TestSimpleRegistry_ConcurrentAccess(t *testing.T) {
	registry, err := NewSimple(Options{
		Logger:              mdwlog.GetDefault(),
		EnableAbbreviations: true,
		EnableAliases:       true,
	})
	if err != nil {
		t.Fatalf("Failed to create registry: %v", err)
	}

	// Test concurrent registration and access
	done := make(chan bool, 10)

	// Concurrent object registration
	for i := 0; i < 5; i++ {
		go func(id int) {
			defer func() { done <- true }()
			obj := &ObjectDefinition{
				Name: fmt.Sprintf("OBJECT%d", id),
				Methods: map[string]*MethodDefinition{
					"METHOD1": {Name: "METHOD1"},
				},
			}
			registry.RegisterObject(obj)
		}(i)
	}

	// Concurrent alias registration
	for i := 0; i < 5; i++ {
		go func(id int) {
			defer func() { done <- true }()
			registry.RegisterAlias(fmt.Sprintf("alias%d", id), fmt.Sprintf("OBJECT%d.METHOD1", id))
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < 10; i++ {
		<-done
	}

	// Verify that all objects were registered
	objects := registry.GetObjects()
	if len(objects) < 7 { // 5 test objects + 2 built-in objects
		t.Errorf("Expected at least 7 objects, got %d", len(objects))
	}

	// Verify that all aliases were registered
	aliases := registry.GetAliases()
	if len(aliases) < 5 {
		t.Errorf("Expected at least 5 aliases, got %d", len(aliases))
	}
}

// Benchmarks

func BenchmarkSimpleRegistry_RegisterObject(b *testing.B) {
	registry, _ := NewSimple(Options{
		Logger:              mdwlog.GetDefault(),
		EnableAbbreviations: true,
	})

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		obj := &ObjectDefinition{
			Name: fmt.Sprintf("OBJECT%d", i),
			Methods: map[string]*MethodDefinition{
				"CREATE": {Name: "CREATE"},
				"LIST":   {Name: "LIST"},
			},
		}
		registry.RegisterObject(obj)
	}
}

func BenchmarkSimpleRegistry_HasObject(b *testing.B) {
	registry, _ := NewSimple(Options{
		Logger: mdwlog.GetDefault(),
	})

	// Register some test objects
	for i := 0; i < 100; i++ {
		obj := &ObjectDefinition{
			Name: fmt.Sprintf("OBJECT%d", i),
		}
		registry.RegisterObject(obj)
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		registry.HasObject(fmt.Sprintf("OBJECT%d", i%100))
	}
}

func BenchmarkSimpleRegistry_ExpandAbbreviation(b *testing.B) {
	registry, _ := NewSimple(Options{
		Logger:              mdwlog.GetDefault(),
		EnableAbbreviations: true,
	})

	// Register test objects to generate abbreviations
	for i := 0; i < 10; i++ {
		obj := &ObjectDefinition{
			Name: fmt.Sprintf("OBJECT%d", i),
			Methods: map[string]*MethodDefinition{
				"CREATE": {Name: "CREATE"},
				"LIST":   {Name: "LIST"},
			},
		}
		registry.RegisterObject(obj)
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		registry.ExpandAbbreviation("CUST.CR")
	}
}

func BenchmarkSimpleRegistry_ValidateCommand(b *testing.B) {
	registry, _ := NewSimple(Options{
		Logger: mdwlog.GetDefault(),
	})

	// Register test object
	obj := &ObjectDefinition{
		Name: "CUSTOMER",
		Methods: map[string]*MethodDefinition{
			"CREATE": {Name: "CREATE"},
			"LIST":   {Name: "LIST"},
		},
	}
	registry.RegisterObject(obj)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		registry.ValidateCommand("CUSTOMER", "CREATE")
	}
}