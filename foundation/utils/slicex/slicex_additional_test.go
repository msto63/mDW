// File: slicex_additional_test.go
// Title: Additional Slice Tests for Coverage Improvement
// Description: Tests for previously untested functions to achieve 90%+ coverage
// Author: msto63 with Claude Sonnet 4.0
// Version: v0.1.0
// Created: 2025-01-25

package slicex

import (
	"strconv"
	"testing"
)

// ===============================
// Tests for 0% Coverage Functions
// ===============================

func TestReduceWithIndex(t *testing.T) {
	t.Run("sum with index", func(t *testing.T) {
		input := []int{1, 2, 3, 4}
		result := ReduceWithIndex(input, 0, func(acc int, idx int, x int) int {
			return acc + x*idx // Multiply by index
		})
		expected := 0 + 1*0 + 2*1 + 3*2 + 4*3 // = 0 + 0 + 2 + 6 + 12 = 20
		
		if result != expected {
			t.Errorf("ReduceWithIndex() = %d, want %d", result, expected)
		}
	})
	
	t.Run("concatenate with index", func(t *testing.T) {
		input := []string{"a", "b", "c"}
		result := ReduceWithIndex(input, "", func(acc string, idx int, x string) string {
			return acc + strconv.Itoa(idx) + x
		})
		expected := "0a1b2c"
		
		if result != expected {
			t.Errorf("ReduceWithIndex() = %s, want %s", result, expected)
		}
	})
	
	t.Run("empty slice", func(t *testing.T) {
		var input []int
		result := ReduceWithIndex(input, 100, func(acc int, idx int, x int) int {
			return acc + x
		})
		expected := 100
		
		if result != expected {
			t.Errorf("ReduceWithIndex() = %d, want %d", result, expected)
		}
	})
}

func TestForEachWithIndex(t *testing.T) {
	t.Run("collect indices", func(t *testing.T) {
		input := []string{"a", "b", "c"}
		var indices []int
		var values []string
		
		ForEachWithIndex(input, func(idx int, x string) {
			indices = append(indices, idx)
			values = append(values, x)
		})
		
		expectedIndices := []int{0, 1, 2}
		expectedValues := []string{"a", "b", "c"}
		
		if !Equal(indices, expectedIndices) {
			t.Errorf("ForEachWithIndex indices = %v, want %v", indices, expectedIndices)
		}
		
		if !Equal(values, expectedValues) {
			t.Errorf("ForEachWithIndex values = %v, want %v", values, expectedValues)
		}
	})
	
	t.Run("empty slice", func(t *testing.T) {
		var input []int
		called := false
		
		ForEachWithIndex(input, func(idx, x int) {
			called = true
		})
		
		if called {
			t.Error("ForEachWithIndex should not call function for empty slice")
		}
	})
}

func TestIndexOfBy(t *testing.T) {
	t.Run("find by predicate", func(t *testing.T) {
		input := []int{1, 3, 5, 8, 10}
		result := IndexOfBy(input, func(x int) bool { return x%2 == 0 })
		expected := 3 // Index of first even number (8)
		
		if result != expected {
			t.Errorf("IndexOfBy() = %d, want %d", result, expected)
		}
	})
	
	t.Run("not found", func(t *testing.T) {
		input := []int{1, 3, 5, 7}
		result := IndexOfBy(input, func(x int) bool { return x > 10 })
		expected := -1
		
		if result != expected {
			t.Errorf("IndexOfBy() = %d, want %d", result, expected)
		}
	})
	
	t.Run("empty slice", func(t *testing.T) {
		var input []int
		result := IndexOfBy(input, func(x int) bool { return true })
		expected := -1
		
		if result != expected {
			t.Errorf("IndexOfBy() = %d, want %d", result, expected)
		}
	})
}

func TestFindLast(t *testing.T) {
	t.Run("find last even number", func(t *testing.T) {
		input := []int{1, 2, 3, 4, 5, 6}
		result, found := FindLast(input, func(x int) bool { return x%2 == 0 })
		expectedValue := 6
		expectedFound := true
		
		if result != expectedValue || found != expectedFound {
			t.Errorf("FindLast() = %d, %t, want %d, %t", result, found, expectedValue, expectedFound)
		}
	})
	
	t.Run("not found", func(t *testing.T) {
		input := []int{1, 3, 5}
		_, found := FindLast(input, func(x int) bool { return x%2 == 0 })
		expectedFound := false
		
		if found != expectedFound {
			t.Errorf("FindLast() found = %t, want %t", found, expectedFound)
		}
	})
	
	t.Run("empty slice", func(t *testing.T) {
		var input []int
		_, found := FindLast(input, func(x int) bool { return true })
		expectedFound := false
		
		if found != expectedFound {
			t.Errorf("FindLast() found = %t, want %t", found, expectedFound)
		}
	})
}

func TestIsNotEmpty(t *testing.T) {
	t.Run("non-empty slice", func(t *testing.T) {
		input := []int{1}
		result := IsNotEmpty(input)
		expected := true
		
		if result != expected {
			t.Errorf("IsNotEmpty() = %t, want %t", result, expected)
		}
	})
	
	t.Run("empty slice", func(t *testing.T) {
		var input []int
		result := IsNotEmpty(input)
		expected := false
		
		if result != expected {
			t.Errorf("IsNotEmpty() = %t, want %t", result, expected)
		}
	})
	
	t.Run("nil slice", func(t *testing.T) {
		var input []int = nil
		result := IsNotEmpty(input)
		expected := false
		
		if result != expected {
			t.Errorf("IsNotEmpty() = %t, want %t", result, expected)
		}
	})
}

func TestMinBy(t *testing.T) {
	t.Run("find shortest string", func(t *testing.T) {
		input := []string{"hello", "hi", "world", "a"}
		result, found := MinBy(input, func(a, b string) bool { return len(a) < len(b) })
		expectedValue := "a"
		expectedFound := true
		
		if result != expectedValue || found != expectedFound {
			t.Errorf("MinBy() = %s, %t, want %s, %t", result, found, expectedValue, expectedFound)
		}
	})
	
	t.Run("find min number", func(t *testing.T) {
		input := []int{-1, -5, -2, -8}
		result, found := MinBy(input, func(a, b int) bool { return a < b })
		expectedValue := -8 // Minimum value
		expectedFound := true
		
		if result != expectedValue || found != expectedFound {
			t.Errorf("MinBy() = %d, %t, want %d, %t", result, found, expectedValue, expectedFound)
		}
	})
	
	t.Run("empty slice", func(t *testing.T) {
		var input []int
		_, found := MinBy(input, func(a, b int) bool { return a < b })
		expectedFound := false
		
		if found != expectedFound {
			t.Errorf("MinBy() found = %t, want %t", found, expectedFound)
		}
	})
}

func TestMaxBy(t *testing.T) {
	t.Run("find longest string", func(t *testing.T) {
		input := []string{"hi", "hello", "world", "a"}
		result, found := MaxBy(input, func(a, b string) bool { return len(a) < len(b) })
		expectedValue := "hello" // First string with max length
		expectedFound := true
		
		if result != expectedValue || found != expectedFound {
			t.Errorf("MaxBy() = %s, %t, want %s, %t", result, found, expectedValue, expectedFound)
		}
	})
	
	t.Run("find max number", func(t *testing.T) {
		input := []int{1, -5, 2, -8, 3}
		result, found := MaxBy(input, func(a, b int) bool { return a < b })
		expectedValue := 3 // Maximum value
		expectedFound := true
		
		if result != expectedValue || found != expectedFound {
			t.Errorf("MaxBy() = %d, %t, want %d, %t", result, found, expectedValue, expectedFound)
		}
	})
	
	t.Run("empty slice", func(t *testing.T) {
		var input []int
		_, found := MaxBy(input, func(a, b int) bool { return a < b })
		expectedFound := false
		
		if found != expectedFound {
			t.Errorf("MaxBy() found = %t, want %t", found, expectedFound)
		}
	})
}

func TestEqualBy(t *testing.T) {
	t.Run("equal by length", func(t *testing.T) {
		slice1 := []string{"ab", "cd", "ef"}
		slice2 := []string{"xy", "zw", "uv"}
		result := EqualBy(slice1, slice2, func(a, b string) bool { return len(a) == len(b) })
		expected := true
		
		if result != expected {
			t.Errorf("EqualBy() = %t, want %t", result, expected)
		}
	})
	
	t.Run("not equal by length", func(t *testing.T) {
		slice1 := []string{"ab", "cd"}
		slice2 := []string{"xyz", "uv"}
		result := EqualBy(slice1, slice2, func(a, b string) bool { return len(a) == len(b) })
		expected := false
		
		if result != expected {
			t.Errorf("EqualBy() = %t, want %t", result, expected)
		}
	})
	
	t.Run("different lengths", func(t *testing.T) {
		slice1 := []int{1, 2}
		slice2 := []int{1, 2, 3}
		result := EqualBy(slice1, slice2, func(a, b int) bool { return a == b })
		expected := false
		
		if result != expected {
			t.Errorf("EqualBy() = %t, want %t", result, expected)
		}
	})
	
	t.Run("both empty", func(t *testing.T) {
		var slice1, slice2 []int
		result := EqualBy(slice1, slice2, func(a, b int) bool { return a == b })
		expected := true
		
		if result != expected {
			t.Errorf("EqualBy() = %t, want %t", result, expected)
		}
	})
}

func TestZip(t *testing.T) {
	t.Run("zip equal length slices", func(t *testing.T) {
		slice1 := []int{1, 2, 3}
		slice2 := []string{"a", "b", "c"}
		result := Zip(slice1, slice2)
		expected := [][2]interface{}{
			{1, "a"},
			{2, "b"},
			{3, "c"},
		}
		
		if len(result) != len(expected) {
			t.Errorf("Zip() length = %d, want %d", len(result), len(expected))
			return
		}
		
		for i, pair := range result {
			if pair.First != expected[i][0] || pair.Second != expected[i][1] {
				t.Errorf("Zip()[%d] = %v, want %v", i, pair, expected[i])
			}
		}
	})
	
	t.Run("zip different length slices", func(t *testing.T) {
		slice1 := []int{1, 2, 3, 4}
		slice2 := []string{"a", "b"}
		result := Zip(slice1, slice2)
		expected := [][2]interface{}{
			{1, "a"},
			{2, "b"},
		}
		
		if len(result) != len(expected) {
			t.Errorf("Zip() length = %d, want %d", len(result), len(expected))
			return
		}
		
		for i, pair := range result {
			if pair.First != expected[i][0] || pair.Second != expected[i][1] {
				t.Errorf("Zip()[%d] = %v, want %v", i, pair, expected[i])
			}
		}
	})
	
	t.Run("zip empty slices", func(t *testing.T) {
		var slice1 []int
		var slice2 []string
		result := Zip(slice1, slice2)
		
		if result != nil {
			t.Errorf("Zip() = %v, want nil", result)
		}
	})
}

func TestIsSortedBy(t *testing.T) {
	t.Run("sorted by length", func(t *testing.T) {
		input := []string{"a", "ab", "abc", "abcd"}
		result := IsSortedBy(input, func(a, b string) bool { return len(a) < len(b) })
		expected := true
		
		if result != expected {
			t.Errorf("IsSortedBy() = %t, want %t", result, expected)
		}
	})
	
	t.Run("not sorted by length", func(t *testing.T) {
		input := []string{"abc", "a", "ab"}
		result := IsSortedBy(input, func(a, b string) bool { return len(a) < len(b) })
		expected := false
		
		if result != expected {
			t.Errorf("IsSortedBy() = %t, want %t", result, expected)
		}
	})
	
	t.Run("empty slice", func(t *testing.T) {
		var input []string
		result := IsSortedBy(input, func(a, b string) bool { return len(a) < len(b) })
		expected := true
		
		if result != expected {
			t.Errorf("IsSortedBy() = %t, want %t", result, expected)
		}
	})
	
	t.Run("single element", func(t *testing.T) {
		input := []int{42}
		result := IsSortedBy(input, func(a, b int) bool { return a < b })
		expected := true
		
		if result != expected {
			t.Errorf("IsSortedBy() = %t, want %t", result, expected)
		}
	})
}

func TestString(t *testing.T) {
	t.Run("integer slice", func(t *testing.T) {
		input := []int{1, 2, 3}
		result := String(input)
		expected := "[1 2 3]"
		
		if result != expected {
			t.Errorf("String() = %s, want %s", result, expected)
		}
	})
	
	t.Run("string slice", func(t *testing.T) {
		input := []string{"hello", "world"}
		result := String(input)
		expected := "[hello world]"
		
		if result != expected {
			t.Errorf("String() = %s, want %s", result, expected)
		}
	})
	
	t.Run("empty slice", func(t *testing.T) {
		var input []int
		result := String(input)
		expected := "[]"
		
		if result != expected {
			t.Errorf("String() = %s, want %s", result, expected)
		}
	})
	
	t.Run("single element", func(t *testing.T) {
		input := []string{"only"}
		result := String(input)
		expected := "[only]"
		
		if result != expected {
			t.Errorf("String() = %s, want %s", result, expected)
		}
	})
}

// ===============================
// Tests for Functions with Low Coverage
// ===============================

func TestMapWithIndexEdgeCases(t *testing.T) {
	t.Run("empty slice edge case", func(t *testing.T) {
		var input []int
		result := MapWithIndex(input, func(idx int, x int) string {
			return strconv.Itoa(x) + "@" + strconv.Itoa(idx)
		})
		
		if result != nil {
			t.Errorf("MapWithIndex() = %v, want nil", result)
		}
	})
}

func TestForEachEdgeCases(t *testing.T) {
	t.Run("nil function handling", func(t *testing.T) {
		// ForEach with nil function should return early, not panic
		input := []int{1, 2, 3}
		// This should not panic based on the implementation
		ForEach(input, nil)
		// If we get here without panic, the test passes
	})
}

func TestCountEdgeCases(t *testing.T) {
	t.Run("no matches", func(t *testing.T) {
		input := []int{1, 3, 5}
		result := Count(input, func(x int) bool { return x%2 == 0 })
		expected := 0
		
		if result != expected {
			t.Errorf("Count() = %d, want %d", result, expected)
		}
	})
}

func TestMaxEdgeCases(t *testing.T) {
	t.Run("single element", func(t *testing.T) {
		input := []int{42}
		result, found := Max(input)
		expected := 42
		expectedFound := true
		
		if result != expected || found != expectedFound {
			t.Errorf("Max() = %d, %t, want %d, %t", result, found, expected, expectedFound)
		}
	})
}

func TestFillEdgeCases(t *testing.T) {
	t.Run("generator function test", func(t *testing.T) {
		result := Fill(3, func(i int) int { return i * 2 })
		expected := []int{0, 2, 4}
		
		if !Equal(result, expected) {
			t.Errorf("Fill() = %v, want %v", result, expected)
		}
	})
}

func TestCloneEdgeCases(t *testing.T) {
	t.Run("nil slice", func(t *testing.T) {
		var input []int
		result := Clone(input)
		
		if result != nil {
			t.Errorf("Clone() = %v, want nil", result)
		}
	})
}

// Additional edge case tests for better coverage
func TestDropEdgeCases(t *testing.T) {
	t.Run("drop more than length", func(t *testing.T) {
		input := []int{1, 2}
		result := Drop(input, 5)
		
		if result != nil {
			t.Errorf("Drop() = %v, want nil", result)
		}
	})
	
	t.Run("drop zero", func(t *testing.T) {
		input := []int{1, 2, 3}
		result := Drop(input, 0)
		expected := []int{1, 2, 3}
		
		if !Equal(result, expected) {
			t.Errorf("Drop() = %v, want %v", result, expected)
		}
	})
}

func TestJoinEdgeCases(t *testing.T) {
	t.Run("single element", func(t *testing.T) {
		input := []string{"only"}
		result := Join(input, ", ")
		expected := "only"
		
		if result != expected {
			t.Errorf("Join() = %s, want %s", result, expected)
		}
	})
}