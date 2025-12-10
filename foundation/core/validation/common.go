// File: common.go
// Title: Validation Framework Utilities
// Description: Provides utility functions and helpers for the validation framework.
//              Contains only framework infrastructure code - no concrete validators.
// Author: msto63 with Claude Sonnet 4.0
// Version: v0.1.0
// Created: 2025-01-25
// Modified: 2025-01-25
//
// Change History:
// - 2025-01-25 v0.1.0: Initial validation framework utilities

package validation

import (
	"fmt"
	"reflect"
	"strconv"
	"unicode/utf8"
)

// Framework Utility Functions

// GetValueLength returns the length of strings, slices, arrays, or maps.
// This is a utility function used by validation frameworks and concrete validators.
// Returns -1 for unsupported types.
func GetValueLength(value interface{}) int {
	if value == nil {
		return 0
	}
	
	switch v := value.(type) {
	case string:
		return utf8.RuneCountInString(v)
	case []interface{}:
		return len(v)
	case []string:
		return len(v)
	case map[string]interface{}:
		return len(v)
	default:
		// Use reflection for other types
		rv := reflect.ValueOf(value)
		switch rv.Kind() {
		case reflect.Slice, reflect.Array, reflect.Map, reflect.Chan:
			return rv.Len()
		case reflect.String:
			return utf8.RuneCountInString(rv.String())
		default:
			return -1 // Invalid type
		}
	}
}

// ConvertToFloat64 converts various numeric types to float64.
// This is a utility function used by validation frameworks and numeric validators.
func ConvertToFloat64(value interface{}) (float64, error) {
	switch v := value.(type) {
	case float64:
		return v, nil
	case float32:
		return float64(v), nil
	case int:
		return float64(v), nil
	case int32:
		return float64(v), nil
	case int64:
		return float64(v), nil
	case uint:
		return float64(v), nil
	case uint32:
		return float64(v), nil
	case uint64:
		return float64(v), nil
	case string:
		return strconv.ParseFloat(v, 64)
	default:
		return 0, fmt.Errorf("cannot convert %T to float64", value)
	}
}

// IsNilOrEmpty checks if a value is nil or considered empty based on its type.
// This is a utility function used by validation frameworks.
func IsNilOrEmpty(value interface{}) bool {
	if value == nil {
		return true
	}
	
	// Use reflection to check various types
	rv := reflect.ValueOf(value)
	switch rv.Kind() {
	case reflect.String:
		return rv.Len() == 0
	case reflect.Slice, reflect.Array, reflect.Map, reflect.Chan:
		return rv.Len() == 0
	case reflect.Ptr, reflect.Interface:
		return rv.IsNil()
	default:
		return false
	}
}

