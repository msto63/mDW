// File: mapx.go
// Title: Core Map Utilities
// Description: Implements core map utility functions including transformation,
//              manipulation, validation, and conversion operations for Go maps.
// Author: msto63 with Claude Sonnet 4.0
// Version: v0.1.0
// Created: 2025-01-24
// Modified: 2025-01-24
//
// Change History:
// - 2025-01-24 v0.1.0: Initial implementation with comprehensive map utilities

package mapx

import (
	"encoding/json"
	"fmt"
	"reflect"
)

// Keys returns a slice of all keys from the map
func Keys[K comparable, V any](m map[K]V) []K {
	if m == nil {
		return nil
	}
	
	keys := make([]K, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

// Values returns a slice of all values from the map
func Values[K comparable, V any](m map[K]V) []V {
	if m == nil {
		return nil
	}
	
	values := make([]V, 0, len(m))
	for _, v := range m {
		values = append(values, v)
	}
	return values
}

// Invert creates a new map by swapping keys and values
// Note: V must be comparable for this to work
func Invert[K, V comparable](m map[K]V) map[V]K {
	if m == nil {
		return nil
	}
	
	inverted := make(map[V]K, len(m))
	for k, v := range m {
		inverted[v] = k
	}
	return inverted
}

// FilterKeys returns a new map containing only entries where the key matches the predicate
func FilterKeys[K comparable, V any](m map[K]V, predicate func(K) bool) map[K]V {
	if m == nil {
		return nil
	}
	
	result := make(map[K]V)
	for k, v := range m {
		if predicate(k) {
			result[k] = v
		}
	}
	return result
}

// FilterValues returns a new map containing only entries where the value matches the predicate
func FilterValues[K comparable, V any](m map[K]V, predicate func(V) bool) map[K]V {
	if m == nil {
		return nil
	}
	
	result := make(map[K]V)
	for k, v := range m {
		if predicate(v) {
			result[k] = v
		}
	}
	return result
}

// Filter returns a new map containing only entries where both key and value match the predicate
func Filter[K comparable, V any](m map[K]V, predicate func(K, V) bool) map[K]V {
	if m == nil {
		return nil
	}
	
	result := make(map[K]V)
	for k, v := range m {
		if predicate(k, v) {
			result[k] = v
		}
	}
	return result
}

// Merge creates a new map by merging multiple maps
// Later maps override values from earlier maps for duplicate keys
func Merge[K comparable, V any](maps ...map[K]V) map[K]V {
	if len(maps) == 0 {
		return make(map[K]V)
	}
	
	// Calculate total capacity
	totalSize := 0
	for _, m := range maps {
		if m != nil {
			totalSize += len(m)
		}
	}
	
	result := make(map[K]V, totalSize)
	for _, m := range maps {
		for k, v := range m {
			result[k] = v
		}
	}
	return result
}

// Clone creates a shallow copy of the map
func Clone[K comparable, V any](m map[K]V) map[K]V {
	if m == nil {
		return nil
	}
	
	clone := make(map[K]V, len(m))
	for k, v := range m {
		clone[k] = v
	}
	return clone
}

// Pick creates a new map containing only the specified keys
func Pick[K comparable, V any](m map[K]V, keys ...K) map[K]V {
	if m == nil {
		return nil
	}
	
	result := make(map[K]V)
	for _, key := range keys {
		if value, exists := m[key]; exists {
			result[key] = value
		}
	}
	return result
}

// Omit creates a new map excluding the specified keys
func Omit[K comparable, V any](m map[K]V, keys ...K) map[K]V {
	if m == nil {
		return nil
	}
	
	// Create a set of keys to omit for O(1) lookup
	omitSet := make(map[K]bool, len(keys))
	for _, key := range keys {
		omitSet[key] = true
	}
	
	result := make(map[K]V)
	for k, v := range m {
		if !omitSet[k] {
			result[k] = v
		}
	}
	return result
}

// Rename creates a new map with renamed keys based on the mapping
func Rename[K comparable, V any](m map[K]V, keyMapping map[K]K) map[K]V {
	if m == nil {
		return nil
	}
	
	result := make(map[K]V, len(m))
	for k, v := range m {
		if newKey, exists := keyMapping[k]; exists {
			result[newKey] = v
		} else {
			result[k] = v
		}
	}
	return result
}

// HasKey checks if the map contains the specified key
func HasKey[K comparable, V any](m map[K]V, key K) bool {
	if m == nil {
		return false
	}
	_, exists := m[key]
	return exists
}

// HasValue checks if the map contains the specified value
func HasValue[K comparable, V comparable](m map[K]V, value V) bool {
	if m == nil {
		return false
	}
	
	for _, v := range m {
		if v == value {
			return true
		}
	}
	return false
}

// IsEmpty checks if the map is empty or nil
func IsEmpty[K comparable, V any](m map[K]V) bool {
	return m == nil || len(m) == 0
}

// Equal checks if two maps are equal (same keys with same values)
func Equal[K, V comparable](m1, m2 map[K]V) bool {
	if m1 == nil && m2 == nil {
		return true
	}
	if m1 == nil || m2 == nil {
		return false
	}
	if len(m1) != len(m2) {
		return false
	}
	
	for k, v1 := range m1 {
		if v2, exists := m2[k]; !exists || v1 != v2 {
			return false
		}
	}
	return true
}

// DeepEqual checks if two maps are deeply equal using reflection
func DeepEqual[K comparable, V any](m1, m2 map[K]V) bool {
	return reflect.DeepEqual(m1, m2)
}

// ToSlice converts a map to a slice of key-value pairs
func ToSlice[K comparable, V any](m map[K]V) []Entry[K, V] {
	if m == nil {
		return nil
	}
	
	result := make([]Entry[K, V], 0, len(m))
	for k, v := range m {
		result = append(result, Entry[K, V]{Key: k, Value: v})
	}
	return result
}

// FromSlice creates a map from a slice of key-value pairs
func FromSlice[K comparable, V any](entries []Entry[K, V]) map[K]V {
	if entries == nil {
		return nil
	}
	
	result := make(map[K]V, len(entries))
	for _, entry := range entries {
		result[entry.Key] = entry.Value
	}
	return result
}

// Entry represents a key-value pair
type Entry[K comparable, V any] struct {
	Key   K `json:"key"`
	Value V `json:"value"`
}

// ToJSON converts a map to JSON string
func ToJSON[K comparable, V any](m map[K]V) (string, error) {
	if m == nil {
		return "null", nil
	}
	
	data, err := json.Marshal(m)
	if err != nil {
		return "", fmt.Errorf("failed to marshal map to JSON: %w", err)
	}
	return string(data), nil
}

// FromJSON creates a map from JSON string
func FromJSON[K comparable, V any](jsonStr string) (map[K]V, error) {
	if jsonStr == "null" || jsonStr == "" {
		return nil, nil
	}
	
	var result map[K]V
	if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to map: %w", err)
	}
	return result, nil
}

// Intersect returns a new map containing entries that exist in both maps
// Values are taken from the first map
func Intersect[K comparable, V any](m1, m2 map[K]V) map[K]V {
	if m1 == nil || m2 == nil {
		return make(map[K]V)
	}
	
	result := make(map[K]V)
	for k, v := range m1 {
		if _, exists := m2[k]; exists {
			result[k] = v
		}
	}
	return result
}

// Difference returns a new map containing entries that exist in the first map but not in the second
func Difference[K comparable, V any](m1, m2 map[K]V) map[K]V {
	if m1 == nil {
		return make(map[K]V)
	}
	if m2 == nil {
		return Clone(m1)
	}
	
	result := make(map[K]V)
	for k, v := range m1 {
		if _, exists := m2[k]; !exists {
			result[k] = v
		}
	}
	return result
}

// Union returns a new map containing all entries from both maps
// Values from the second map override values from the first map for duplicate keys
func Union[K comparable, V any](m1, m2 map[K]V) map[K]V {
	return Merge(m1, m2)
}

// Transform applies a transformation function to all values in the map
func Transform[K comparable, V, R any](m map[K]V, transformer func(K, V) R) map[K]R {
	if m == nil {
		return nil
	}
	
	result := make(map[K]R, len(m))
	for k, v := range m {
		result[k] = transformer(k, v)
	}
	return result
}

// TransformKeys applies a transformation function to all keys in the map
func TransformKeys[K, R comparable, V any](m map[K]V, transformer func(K) R) map[R]V {
	if m == nil {
		return nil
	}
	
	result := make(map[R]V, len(m))
	for k, v := range m {
		newKey := transformer(k)
		result[newKey] = v
	}
	return result
}

// TransformValues applies a transformation function to all values in the map
func TransformValues[K comparable, V, R any](m map[K]V, transformer func(V) R) map[K]R {
	if m == nil {
		return nil
	}
	
	result := make(map[K]R, len(m))
	for k, v := range m {
		result[k] = transformer(v)
	}
	return result
}

// ForEach iterates over all key-value pairs in the map
func ForEach[K comparable, V any](m map[K]V, fn func(K, V)) {
	if m == nil {
		return
	}
	
	for k, v := range m {
		fn(k, v)
	}
}

// Size returns the number of entries in the map
func Size[K comparable, V any](m map[K]V) int {
	if m == nil {
		return 0
	}
	return len(m)
}

// Clear removes all entries from the map (modifies the original map)
func Clear[K comparable, V any](m map[K]V) {
	if m == nil {
		return
	}
	
	for k := range m {
		delete(m, k)
	}
}