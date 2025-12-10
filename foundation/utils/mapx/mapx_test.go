// File: mapx_test.go
// Title: Map Utilities Tests
// Description: Comprehensive tests for map utility functions including
//              transformation, manipulation, validation, and set operations.
// Author: msto63 with Claude Sonnet 4.0
// Version: v0.1.0
// Created: 2025-01-24
// Modified: 2025-01-24
//
// Change History:
// - 2025-01-24 v0.1.0: Initial implementation with comprehensive test coverage

package mapx

import (
	"strconv"
	"strings"
	"testing"
)

func TestKeys(t *testing.T) {
	tests := []struct {
		name     string
		input    map[string]int
		expected int // length of expected keys
	}{
		{
			name:     "nil map",
			input:    nil,
			expected: 0,
		},
		{
			name:     "empty map",
			input:    map[string]int{},
			expected: 0,
		},
		{
			name:     "single key",
			input:    map[string]int{"a": 1},
			expected: 1,
		},
		{
			name:     "multiple keys",
			input:    map[string]int{"a": 1, "b": 2, "c": 3},
			expected: 3,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Keys(tt.input)
			if (result == nil && tt.expected != 0) || (result != nil && len(result) != tt.expected) {
				t.Errorf("Keys() = %v, want length %d", result, tt.expected)
			}
			
			// Verify all keys are present if not nil
			if tt.input != nil && result != nil {
				for _, key := range result {
					if _, exists := tt.input[key]; !exists {
						t.Errorf("Keys() returned non-existent key: %v", key)
					}
				}
			}
		})
	}
}

func TestValues(t *testing.T) {
	tests := []struct {
		name     string
		input    map[string]int
		expected int // length of expected values
	}{
		{
			name:     "nil map",
			input:    nil,
			expected: 0,
		},
		{
			name:     "empty map",
			input:    map[string]int{},
			expected: 0,
		},
		{
			name:     "single value",
			input:    map[string]int{"a": 1},
			expected: 1,
		},
		{
			name:     "multiple values",
			input:    map[string]int{"a": 1, "b": 2, "c": 3},
			expected: 3,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Values(tt.input)
			if (result == nil && tt.expected != 0) || (result != nil && len(result) != tt.expected) {
				t.Errorf("Values() = %v, want length %d", result, tt.expected)
			}
		})
	}
}

func TestInvert(t *testing.T) {
	tests := []struct {
		name     string
		input    map[string]int
		expected map[int]string
	}{
		{
			name:     "nil map",
			input:    nil,
			expected: nil,
		},
		{
			name:     "empty map",
			input:    map[string]int{},
			expected: map[int]string{},
		},
		{
			name:     "single entry",
			input:    map[string]int{"a": 1},
			expected: map[int]string{1: "a"},
		},
		{
			name:     "multiple entries",
			input:    map[string]int{"a": 1, "b": 2, "c": 3},
			expected: map[int]string{1: "a", 2: "b", 3: "c"},
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Invert(tt.input)
			if !Equal(result, tt.expected) {
				t.Errorf("Invert() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestFilterKeys(t *testing.T) {
	input := map[string]int{"apple": 1, "banana": 2, "cherry": 3, "date": 4}
	
	// Filter keys that start with 'a' or 'b'
	result := FilterKeys(input, func(k string) bool {
		return strings.HasPrefix(k, "a") || strings.HasPrefix(k, "b")
	})
	
	expected := map[string]int{"apple": 1, "banana": 2}
	if !Equal(result, expected) {
		t.Errorf("FilterKeys() = %v, want %v", result, expected)
	}
	
	// Test with nil map
	nilResult := FilterKeys(map[string]int(nil), func(k string) bool { return true })
	if nilResult != nil {
		t.Errorf("FilterKeys() with nil map = %v, want nil", nilResult)
	}
}

func TestFilterValues(t *testing.T) {
	input := map[string]int{"a": 1, "b": 2, "c": 3, "d": 4}
	
	// Filter values greater than 2
	result := FilterValues(input, func(v int) bool {
		return v > 2
	})
	
	expected := map[string]int{"c": 3, "d": 4}
	if !Equal(result, expected) {
		t.Errorf("FilterValues() = %v, want %v", result, expected)
	}
}

func TestFilter(t *testing.T) {
	input := map[string]int{"a": 1, "b": 2, "c": 3, "d": 4}
	
	// Filter entries where key comes after 'b' and value is even
	result := Filter(input, func(k string, v int) bool {
		return k > "b" && v%2 == 0
	})
	
	expected := map[string]int{"d": 4}
	if !Equal(result, expected) {
		t.Errorf("Filter() = %v, want %v", result, expected)
	}
}

func TestMerge(t *testing.T) {
	tests := []struct {
		name     string
		maps     []map[string]int
		expected map[string]int
	}{
		{
			name:     "no maps",
			maps:     []map[string]int{},
			expected: map[string]int{},
		},
		{
			name:     "single map",
			maps:     []map[string]int{{"a": 1, "b": 2}},
			expected: map[string]int{"a": 1, "b": 2},
		},
		{
			name:     "two maps no overlap",
			maps:     []map[string]int{{"a": 1}, {"b": 2}},
			expected: map[string]int{"a": 1, "b": 2},
		},
		{
			name:     "two maps with overlap",
			maps:     []map[string]int{{"a": 1, "b": 2}, {"b": 3, "c": 4}},
			expected: map[string]int{"a": 1, "b": 3, "c": 4},
		},
		{
			name:     "with nil map",
			maps:     []map[string]int{{"a": 1}, nil, {"b": 2}},
			expected: map[string]int{"a": 1, "b": 2},
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Merge(tt.maps...)
			if !Equal(result, tt.expected) {
				t.Errorf("Merge() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestClone(t *testing.T) {
	tests := []struct {
		name  string
		input map[string]int
	}{
		{
			name:  "nil map",
			input: nil,
		},
		{
			name:  "empty map",
			input: map[string]int{},
		},
		{
			name:  "non-empty map",
			input: map[string]int{"a": 1, "b": 2, "c": 3},
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Clone(tt.input)
			
			if tt.input == nil {
				if result != nil {
					t.Errorf("Clone() of nil map = %v, want nil", result)
				}
				return
			}
			
			if !Equal(result, tt.input) {
				t.Errorf("Clone() = %v, want %v", result, tt.input)
			}
			
			// Verify it's a different instance
			if len(tt.input) > 0 {
				// Modify original to ensure clone is independent
				for k := range tt.input {
					tt.input[k] = 999
					break
				}
				
				// Clone should not be affected
				found999 := false
				for _, v := range result {
					if v == 999 {
						found999 = true
						break
					}
				}
				if found999 {
					t.Error("Clone() should create independent copy")
				}
			}
		})
	}
}

func TestPick(t *testing.T) {
	input := map[string]int{"a": 1, "b": 2, "c": 3, "d": 4}
	
	tests := []struct {
		name     string
		keys     []string
		expected map[string]int
	}{
		{
			name:     "pick existing keys",
			keys:     []string{"a", "c"},
			expected: map[string]int{"a": 1, "c": 3},
		},
		{
			name:     "pick non-existing keys",
			keys:     []string{"x", "y"},
			expected: map[string]int{},
		},
		{
			name:     "pick mixed keys",
			keys:     []string{"a", "x", "c"},
			expected: map[string]int{"a": 1, "c": 3},
		},
		{
			name:     "pick no keys",
			keys:     []string{},
			expected: map[string]int{},
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Pick(input, tt.keys...)
			if !Equal(result, tt.expected) {
				t.Errorf("Pick() = %v, want %v", result, tt.expected)
			}
		})
	}
	
	// Test with nil map
	nilResult := Pick(map[string]int(nil), "a")
	if nilResult != nil {
		t.Errorf("Pick() with nil map = %v, want nil", nilResult)
	}
}

func TestOmit(t *testing.T) {
	input := map[string]int{"a": 1, "b": 2, "c": 3, "d": 4}
	
	tests := []struct {
		name     string
		keys     []string
		expected map[string]int
	}{
		{
			name:     "omit existing keys",
			keys:     []string{"a", "c"},
			expected: map[string]int{"b": 2, "d": 4},
		},
		{
			name:     "omit non-existing keys",
			keys:     []string{"x", "y"},
			expected: map[string]int{"a": 1, "b": 2, "c": 3, "d": 4},
		},
		{
			name:     "omit mixed keys",
			keys:     []string{"a", "x", "c"},
			expected: map[string]int{"b": 2, "d": 4},
		},
		{
			name:     "omit no keys",
			keys:     []string{},
			expected: map[string]int{"a": 1, "b": 2, "c": 3, "d": 4},
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Omit(input, tt.keys...)
			if !Equal(result, tt.expected) {
				t.Errorf("Omit() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestRename(t *testing.T) {
	input := map[string]int{"a": 1, "b": 2, "c": 3}
	keyMapping := map[string]string{"a": "alpha", "b": "beta"}
	
	result := Rename(input, keyMapping)
	expected := map[string]int{"alpha": 1, "beta": 2, "c": 3}
	
	if !Equal(result, expected) {
		t.Errorf("Rename() = %v, want %v", result, expected)
	}
	
	// Test with nil map
	nilResult := Rename(map[string]int(nil), keyMapping)
	if nilResult != nil {
		t.Errorf("Rename() with nil map = %v, want nil", nilResult)
	}
}

func TestHasKey(t *testing.T) {
	input := map[string]int{"a": 1, "b": 2, "c": 3}
	
	tests := []struct {
		name     string
		key      string
		expected bool
	}{
		{"existing key", "a", true},
		{"non-existing key", "x", false},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := HasKey(input, tt.key)
			if result != tt.expected {
				t.Errorf("HasKey() = %v, want %v", result, tt.expected)
			}
		})
	}
	
	// Test with nil map
	if HasKey(map[string]int(nil), "a") {
		t.Error("HasKey() with nil map should return false")
	}
}

func TestHasValue(t *testing.T) {
	input := map[string]int{"a": 1, "b": 2, "c": 3}
	
	tests := []struct {
		name     string
		value    int
		expected bool
	}{
		{"existing value", 1, true},
		{"non-existing value", 99, false},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := HasValue(input, tt.value)
			if result != tt.expected {
				t.Errorf("HasValue() = %v, want %v", result, tt.expected)
			}
		})
	}
	
	// Test with nil map
	if HasValue(map[string]int(nil), 1) {
		t.Error("HasValue() with nil map should return false")
	}
}

func TestIsEmpty(t *testing.T) {
	tests := []struct {
		name     string
		input    map[string]int
		expected bool
	}{
		{"nil map", nil, true},
		{"empty map", map[string]int{}, true},
		{"non-empty map", map[string]int{"a": 1}, false},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsEmpty(tt.input)
			if result != tt.expected {
				t.Errorf("IsEmpty() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestEqual(t *testing.T) {
	tests := []struct {
		name     string
		m1       map[string]int
		m2       map[string]int
		expected bool
	}{
		{
			name:     "both nil",
			m1:       nil,
			m2:       nil,
			expected: true,
		},
		{
			name:     "one nil",
			m1:       nil,
			m2:       map[string]int{},
			expected: false,
		},
		{
			name:     "both empty",
			m1:       map[string]int{},
			m2:       map[string]int{},
			expected: true,
		},
		{
			name:     "identical",
			m1:       map[string]int{"a": 1, "b": 2},
			m2:       map[string]int{"a": 1, "b": 2},
			expected: true,
		},
		{
			name:     "different values",
			m1:       map[string]int{"a": 1, "b": 2},
			m2:       map[string]int{"a": 1, "b": 3},
			expected: false,
		},
		{
			name:     "different keys",
			m1:       map[string]int{"a": 1, "b": 2},
			m2:       map[string]int{"a": 1, "c": 2},
			expected: false,
		},
		{
			name:     "different sizes",
			m1:       map[string]int{"a": 1},
			m2:       map[string]int{"a": 1, "b": 2},
			expected: false,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Equal(tt.m1, tt.m2)
			if result != tt.expected {
				t.Errorf("Equal() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestToSliceAndFromSlice(t *testing.T) {
	input := map[string]int{"a": 1, "b": 2, "c": 3}
	
	// Convert to slice
	slice := ToSlice(input)
	if len(slice) != 3 {
		t.Errorf("ToSlice() length = %d, want 3", len(slice))
	}
	
	// Convert back to map
	result := FromSlice(slice)
	if !Equal(result, input) {
		t.Errorf("FromSlice(ToSlice()) = %v, want %v", result, input)
	}
	
	// Test with nil
	nilSlice := ToSlice(map[string]int(nil))
	if nilSlice != nil {
		t.Errorf("ToSlice() with nil map = %v, want nil", nilSlice)
	}
	
	nilMap := FromSlice([]Entry[string, int](nil))
	if nilMap != nil {
		t.Errorf("FromSlice() with nil slice = %v, want nil", nilMap)
	}
}

func TestJSONConversion(t *testing.T) {
	input := map[string]int{"a": 1, "b": 2, "c": 3}
	
	// Convert to JSON
	jsonStr, err := ToJSON(input)
	if err != nil {
		t.Fatalf("ToJSON() error = %v", err)
	}
	
	// Convert back from JSON
	result, err := FromJSON[string, int](jsonStr)
	if err != nil {
		t.Fatalf("FromJSON() error = %v", err)
	}
	
	if !Equal(result, input) {
		t.Errorf("FromJSON(ToJSON()) = %v, want %v", result, input)
	}
	
	// Test with nil
	nilJSON, err := ToJSON(map[string]int(nil))
	if err != nil {
		t.Errorf("ToJSON() with nil map error = %v", err)
	}
	if nilJSON != "null" {
		t.Errorf("ToJSON() with nil map = %s, want 'null'", nilJSON)
	}
	
	nilMap, err := FromJSON[string, int]("null")
	if err != nil {
		t.Errorf("FromJSON() with 'null' error = %v", err)
	}
	if nilMap != nil {
		t.Errorf("FromJSON() with 'null' = %v, want nil", nilMap)
	}
}

func TestIntersect(t *testing.T) {
	m1 := map[string]int{"a": 1, "b": 2, "c": 3}
	m2 := map[string]int{"b": 4, "c": 5, "d": 6}
	
	result := Intersect(m1, m2)
	expected := map[string]int{"b": 2, "c": 3} // Values from m1
	
	if !Equal(result, expected) {
		t.Errorf("Intersect() = %v, want %v", result, expected)
	}
	
	// Test with nil maps
	nilResult := Intersect(map[string]int(nil), m2)
	if !IsEmpty(nilResult) {
		t.Errorf("Intersect() with nil first map = %v, want empty", nilResult)
	}
}

func TestDifference(t *testing.T) {
	m1 := map[string]int{"a": 1, "b": 2, "c": 3}
	m2 := map[string]int{"b": 4, "c": 5, "d": 6}
	
	result := Difference(m1, m2)
	expected := map[string]int{"a": 1}
	
	if !Equal(result, expected) {
		t.Errorf("Difference() = %v, want %v", result, expected)
	}
	
	// Test with nil maps
	nilResult := Difference(map[string]int(nil), m2)
	if !IsEmpty(nilResult) {
		t.Errorf("Difference() with nil first map = %v, want empty", nilResult)
	}
	
	cloneResult := Difference(m1, map[string]int(nil))
	if !Equal(cloneResult, m1) {
		t.Errorf("Difference() with nil second map should clone first map")
	}
}

func TestUnion(t *testing.T) {
	m1 := map[string]int{"a": 1, "b": 2}
	m2 := map[string]int{"b": 3, "c": 4}
	
	result := Union(m1, m2)
	expected := map[string]int{"a": 1, "b": 3, "c": 4}
	
	if !Equal(result, expected) {
		t.Errorf("Union() = %v, want %v", result, expected)
	}
}

func TestTransform(t *testing.T) {
	input := map[string]int{"a": 1, "b": 2, "c": 3}
	
	// Transform values to strings
	result := Transform(input, func(k string, v int) string {
		return k + strconv.Itoa(v)
	})
	
	expected := map[string]string{"a": "a1", "b": "b2", "c": "c3"}
	if !Equal(result, expected) {
		t.Errorf("Transform() = %v, want %v", result, expected)
	}
	
	// Test with nil map
	nilResult := Transform(map[string]int(nil), func(k string, v int) string {
		return k + strconv.Itoa(v)
	})
	if nilResult != nil {
		t.Errorf("Transform() with nil map = %v, want nil", nilResult)
	}
}

func TestTransformKeys(t *testing.T) {
	input := map[string]int{"a": 1, "b": 2, "c": 3}
	
	// Transform keys to uppercase
	result := TransformKeys(input, func(k string) string {
		return strings.ToUpper(k)
	})
	
	expected := map[string]int{"A": 1, "B": 2, "C": 3}
	if !Equal(result, expected) {
		t.Errorf("TransformKeys() = %v, want %v", result, expected)
	}
}

func TestTransformValues(t *testing.T) {
	input := map[string]int{"a": 1, "b": 2, "c": 3}
	
	// Transform values by doubling
	result := TransformValues(input, func(v int) int {
		return v * 2
	})
	
	expected := map[string]int{"a": 2, "b": 4, "c": 6}
	if !Equal(result, expected) {
		t.Errorf("TransformValues() = %v, want %v", result, expected)
	}
}

func TestForEach(t *testing.T) {
	input := map[string]int{"a": 1, "b": 2, "c": 3}
	
	sum := 0
	keys := make([]string, 0)
	
	ForEach(input, func(k string, v int) {
		sum += v
		keys = append(keys, k)
	})
	
	if sum != 6 {
		t.Errorf("ForEach() sum = %d, want 6", sum)
	}
	
	if len(keys) != 3 {
		t.Errorf("ForEach() collected %d keys, want 3", len(keys))
	}
	
	// Test with nil map (should not panic)
	ForEach(map[string]int(nil), func(k string, v int) {
		t.Error("ForEach() with nil map should not call function")
	})
}

func TestSize(t *testing.T) {
	tests := []struct {
		name     string
		input    map[string]int
		expected int
	}{
		{"nil map", nil, 0},
		{"empty map", map[string]int{}, 0},
		{"single entry", map[string]int{"a": 1}, 1},
		{"multiple entries", map[string]int{"a": 1, "b": 2, "c": 3}, 3},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Size(tt.input)
			if result != tt.expected {
				t.Errorf("Size() = %d, want %d", result, tt.expected)
			}
		})
	}
}

func TestClear(t *testing.T) {
	input := map[string]int{"a": 1, "b": 2, "c": 3}
	Clear(input)
	
	if len(input) != 0 {
		t.Errorf("Clear() should empty the map, got length %d", len(input))
	}
	
	// Test with nil map (should not panic)
	Clear(map[string]int(nil))
}