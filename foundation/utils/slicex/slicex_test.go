// File: slicex_test.go
// Title: Slice Utilities Tests
// Description: Comprehensive test suite for all slicex utility functions including
//              unit tests, edge cases, and integration scenarios.
// Author: msto63 with Claude Sonnet 4.0
// Version: v0.1.0
// Created: 2025-01-25
// Modified: 2025-01-25
//
// Change History:
// - 2025-01-25 v0.1.0: Initial test implementation with comprehensive coverage

package slicex

import (
	"strconv"
	"testing"
)

// ===============================
// Core Transformation Tests
// ===============================

func TestFilter(t *testing.T) {
	t.Run("filter even numbers", func(t *testing.T) {
		input := []int{1, 2, 3, 4, 5, 6}
		result := Filter(input, func(x int) bool { return x%2 == 0 })
		expected := []int{2, 4, 6}
		
		if !Equal(result, expected) {
			t.Errorf("Filter() = %v, want %v", result, expected)
		}
	})
	
	t.Run("filter strings by length", func(t *testing.T) {
		input := []string{"a", "ab", "abc", "abcd"}
		result := Filter(input, func(s string) bool { return len(s) > 2 })
		expected := []string{"abc", "abcd"}
		
		if !Equal(result, expected) {
			t.Errorf("Filter() = %v, want %v", result, expected)
		}
	})
	
	t.Run("filter empty slice", func(t *testing.T) {
		var input []int
		result := Filter(input, func(x int) bool { return x > 0 })
		
		if result != nil {
			t.Errorf("Filter() = %v, want nil", result)
		}
	})
	
	t.Run("filter nil slice", func(t *testing.T) {
		result := Filter(nil, func(x int) bool { return x > 0 })
		
		if result != nil {
			t.Errorf("Filter() = %v, want nil", result)
		}
	})
	
	t.Run("filter with nil predicate", func(t *testing.T) {
		input := []int{1, 2, 3}
		result := Filter(input, nil)
		
		if result != nil {
			t.Errorf("Filter() = %v, want nil", result)
		}
	})
}

func TestMap(t *testing.T) {
	t.Run("map int to string", func(t *testing.T) {
		input := []int{1, 2, 3}
		result := Map(input, func(x int) string { return strconv.Itoa(x) })
		expected := []string{"1", "2", "3"}
		
		if !Equal(result, expected) {
			t.Errorf("Map() = %v, want %v", result, expected)
		}
	})
	
	t.Run("map string to length", func(t *testing.T) {
		input := []string{"a", "ab", "abc"}
		result := Map(input, func(s string) int { return len(s) })
		expected := []int{1, 2, 3}
		
		if !Equal(result, expected) {
			t.Errorf("Map() = %v, want %v", result, expected)
		}
	})
	
	t.Run("map empty slice", func(t *testing.T) {
		var input []int
		result := Map(input, func(x int) string { return strconv.Itoa(x) })
		
		if result != nil {
			t.Errorf("Map() = %v, want nil", result)
		}
	})
	
	t.Run("map nil slice", func(t *testing.T) {
		result := Map(nil, func(x int) string { return strconv.Itoa(x) })
		
		if result != nil {
			t.Errorf("Map() = %v, want nil", result)
		}
	})
}

func TestMapWithIndex(t *testing.T) {
	t.Run("map with index", func(t *testing.T) {
		input := []string{"a", "b", "c"}
		result := MapWithIndex(input, func(i int, s string) string {
			return strconv.Itoa(i) + ":" + s
		})
		expected := []string{"0:a", "1:b", "2:c"}
		
		if !Equal(result, expected) {
			t.Errorf("MapWithIndex() = %v, want %v", result, expected)
		}
	})
}

func TestReduce(t *testing.T) {
	t.Run("reduce sum", func(t *testing.T) {
		input := []int{1, 2, 3, 4, 5}
		result := Reduce(input, 0, func(acc, x int) int { return acc + x })
		expected := 15
		
		if result != expected {
			t.Errorf("Reduce() = %v, want %v", result, expected)
		}
	})
	
	t.Run("reduce string concatenation", func(t *testing.T) {
		input := []string{"a", "b", "c"}
		result := Reduce(input, "", func(acc, s string) string { return acc + s })
		expected := "abc"
		
		if result != expected {
			t.Errorf("Reduce() = %v, want %v", result, expected)
		}
	})
	
	t.Run("reduce empty slice", func(t *testing.T) {
		var input []int
		result := Reduce(input, 42, func(acc, x int) int { return acc + x })
		expected := 42
		
		if result != expected {
			t.Errorf("Reduce() = %v, want %v", result, expected)
		}
	})
}

// ===============================
// Slice Manipulation Tests
// ===============================

func TestChunk(t *testing.T) {
	t.Run("chunk normal case", func(t *testing.T) {
		input := []int{1, 2, 3, 4, 5, 6, 7}
		result := Chunk(input, 3)
		expected := [][]int{{1, 2, 3}, {4, 5, 6}, {7}}
		
		if len(result) != len(expected) {
			t.Errorf("Chunk() length = %v, want %v", len(result), len(expected))
			return
		}
		
		for i, chunk := range result {
			if !Equal(chunk, expected[i]) {
				t.Errorf("Chunk()[%d] = %v, want %v", i, chunk, expected[i])
			}
		}
	})
	
	t.Run("chunk exact division", func(t *testing.T) {
		input := []int{1, 2, 3, 4}
		result := Chunk(input, 2)
		expected := [][]int{{1, 2}, {3, 4}}
		
		if len(result) != len(expected) {
			t.Errorf("Chunk() length = %v, want %v", len(result), len(expected))
		}
	})
	
	t.Run("chunk size larger than slice", func(t *testing.T) {
		input := []int{1, 2, 3}
		result := Chunk(input, 5)
		expected := [][]int{{1, 2, 3}}
		
		if len(result) != 1 || !Equal(result[0], expected[0]) {
			t.Errorf("Chunk() = %v, want %v", result, expected)
		}
	})
	
	t.Run("chunk invalid size", func(t *testing.T) {
		input := []int{1, 2, 3}
		result := Chunk(input, 0)
		
		if result != nil {
			t.Errorf("Chunk() = %v, want nil", result)
		}
	})
}

func TestFlatten(t *testing.T) {
	t.Run("flatten normal case", func(t *testing.T) {
		input := [][]int{{1, 2}, {3, 4}, {5}}
		result := Flatten(input)
		expected := []int{1, 2, 3, 4, 5}
		
		if !Equal(result, expected) {
			t.Errorf("Flatten() = %v, want %v", result, expected)
		}
	})
	
	t.Run("flatten empty slices", func(t *testing.T) {
		input := [][]int{{}, {1, 2}, {}}
		result := Flatten(input)
		expected := []int{1, 2}
		
		if !Equal(result, expected) {
			t.Errorf("Flatten() = %v, want %v", result, expected)
		}
	})
	
	t.Run("flatten nil slice", func(t *testing.T) {
		result := Flatten(([][]int)(nil))
		
		if result != nil {
			t.Errorf("Flatten() = %v, want nil", result)
		}
	})
}

func TestUnique(t *testing.T) {
	t.Run("unique with duplicates", func(t *testing.T) {
		input := []int{1, 2, 2, 3, 1, 4}
		result := Unique(input)
		expected := []int{1, 2, 3, 4}
		
		if !Equal(result, expected) {
			t.Errorf("Unique() = %v, want %v", result, expected)
		}
	})
	
	t.Run("unique no duplicates", func(t *testing.T) {
		input := []int{1, 2, 3, 4}
		result := Unique(input)
		expected := []int{1, 2, 3, 4}
		
		if !Equal(result, expected) {
			t.Errorf("Unique() = %v, want %v", result, expected)
		}
	})
	
	t.Run("unique strings", func(t *testing.T) {
		input := []string{"a", "b", "a", "c", "b"}
		result := Unique(input)
		expected := []string{"a", "b", "c"}
		
		if !Equal(result, expected) {
			t.Errorf("Unique() = %v, want %v", result, expected)
		}
	})
}

func TestUniqueBy(t *testing.T) {
	type Person struct {
		Name string
		Age  int
	}
	
	t.Run("unique by field", func(t *testing.T) {
		input := []Person{
			{"Alice", 25},
			{"Bob", 30},
			{"Alice", 35}, // duplicate name
			{"Charlie", 25}, // duplicate age, but different name
		}
		result := UniqueBy(input, func(p Person) string { return p.Name })
		expected := []Person{
			{"Alice", 25},
			{"Bob", 30},
			{"Charlie", 25},
		}
		
		if len(result) != len(expected) {
			t.Errorf("UniqueBy() length = %v, want %v", len(result), len(expected))
			return
		}
		
		// Check each person (can't use Equal directly with structs)
		for i, person := range result {
			if person.Name != expected[i].Name {
				t.Errorf("UniqueBy()[%d].Name = %v, want %v", i, person.Name, expected[i].Name)
			}
		}
	})
}

func TestUnion(t *testing.T) {
	t.Run("union with overlap", func(t *testing.T) {
		slice1 := []int{1, 2, 3}
		slice2 := []int{3, 4, 5}
		result := Union(slice1, slice2)
		
		// Union should contain all unique elements
		if !Contains(result, 1) || !Contains(result, 2) || !Contains(result, 3) || !Contains(result, 4) || !Contains(result, 5) {
			t.Errorf("Union() = %v, should contain all elements 1,2,3,4,5", result)
		}
		
		// Check no duplicates
		unique := Unique(result)
		if len(result) != len(unique) {
			t.Errorf("Union() = %v contains duplicates", result)
		}
	})
}

func TestIntersect(t *testing.T) {
	t.Run("intersect with overlap", func(t *testing.T) {
		slice1 := []int{1, 2, 3, 4}
		slice2 := []int{3, 4, 5, 6}
		result := Intersect(slice1, slice2)
		expected := []int{3, 4}
		
		if !Equal(result, expected) {
			t.Errorf("Intersect() = %v, want %v", result, expected)
		}
	})
	
	t.Run("intersect no overlap", func(t *testing.T) {
		slice1 := []int{1, 2}
		slice2 := []int{3, 4}
		result := Intersect(slice1, slice2)
		
		if len(result) != 0 {
			t.Errorf("Intersect() = %v, want empty slice", result)
		}
	})
}

func TestDifference(t *testing.T) {
	t.Run("difference normal case", func(t *testing.T) {
		slice1 := []int{1, 2, 3, 4}
		slice2 := []int{2, 4}
		result := Difference(slice1, slice2)
		expected := []int{1, 3}
		
		if !Equal(result, expected) {
			t.Errorf("Difference() = %v, want %v", result, expected)
		}
	})
	
	t.Run("difference no overlap", func(t *testing.T) {
		slice1 := []int{1, 2, 3}
		slice2 := []int{4, 5, 6}
		result := Difference(slice1, slice2)
		expected := []int{1, 2, 3}
		
		if !Equal(result, expected) {
			t.Errorf("Difference() = %v, want %v", result, expected)
		}
	})
}

func TestReverse(t *testing.T) {
	t.Run("reverse normal case", func(t *testing.T) {
		input := []int{1, 2, 3, 4, 5}
		result := Reverse(input)
		expected := []int{5, 4, 3, 2, 1}
		
		if !Equal(result, expected) {
			t.Errorf("Reverse() = %v, want %v", result, expected)
		}
	})
	
	t.Run("reverse single element", func(t *testing.T) {
		input := []int{42}
		result := Reverse(input)
		expected := []int{42}
		
		if !Equal(result, expected) {
			t.Errorf("Reverse() = %v, want %v", result, expected)
		}
	})
	
	t.Run("reverse empty slice", func(t *testing.T) {
		var input []int
		result := Reverse(input)
		
		if result != nil {
			t.Errorf("Reverse() = %v, want nil", result)
		}
	})
}

// ===============================
// Search and Validation Tests
// ===============================

func TestContains(t *testing.T) {
	t.Run("contains existing element", func(t *testing.T) {
		input := []int{1, 2, 3, 4, 5}
		if !Contains(input, 3) {
			t.Error("Contains() = false, want true")
		}
	})
	
	t.Run("contains non-existing element", func(t *testing.T) {
		input := []int{1, 2, 3, 4, 5}
		if Contains(input, 6) {
			t.Error("Contains() = true, want false")
		}
	})
	
	t.Run("contains in empty slice", func(t *testing.T) {
		var input []int
		if Contains(input, 1) {
			t.Error("Contains() = true, want false")
		}
	})
}

func TestContainsBy(t *testing.T) {
	t.Run("contains by predicate", func(t *testing.T) {
		input := []int{1, 2, 3, 4, 5}
		if !ContainsBy(input, func(x int) bool { return x > 4 }) {
			t.Error("ContainsBy() = false, want true")
		}
	})
	
	t.Run("not contains by predicate", func(t *testing.T) {
		input := []int{1, 2, 3, 4, 5}
		if ContainsBy(input, func(x int) bool { return x > 10 }) {
			t.Error("ContainsBy() = true, want false")
		}
	})
}

func TestIndexOf(t *testing.T) {
	t.Run("index of existing element", func(t *testing.T) {
		input := []string{"a", "b", "c", "b"}
		result := IndexOf(input, "b")
		expected := 1 // first occurrence
		
		if result != expected {
			t.Errorf("IndexOf() = %v, want %v", result, expected)
		}
	})
	
	t.Run("index of non-existing element", func(t *testing.T) {
		input := []string{"a", "b", "c"}
		result := IndexOf(input, "d")
		expected := -1
		
		if result != expected {
			t.Errorf("IndexOf() = %v, want %v", result, expected)
		}
	})
}

func TestLastIndexOf(t *testing.T) {
	t.Run("last index of existing element", func(t *testing.T) {
		input := []string{"a", "b", "c", "b"}
		result := LastIndexOf(input, "b")
		expected := 3 // last occurrence
		
		if result != expected {
			t.Errorf("LastIndexOf() = %v, want %v", result, expected)
		}
	})
}

func TestFind(t *testing.T) {
	t.Run("find existing element", func(t *testing.T) {
		input := []int{1, 2, 3, 4, 5}
		result, found := Find(input, func(x int) bool { return x > 3 })
		
		if !found || result != 4 {
			t.Errorf("Find() = (%v, %v), want (4, true)", result, found)
		}
	})
	
	t.Run("find non-existing element", func(t *testing.T) {
		input := []int{1, 2, 3, 4, 5}
		result, found := Find(input, func(x int) bool { return x > 10 })
		
		if found || result != 0 {
			t.Errorf("Find() = (%v, %v), want (0, false)", result, found)
		}
	})
}

func TestEvery(t *testing.T) {
	t.Run("every positive", func(t *testing.T) {
		input := []int{2, 4, 6, 8}
		result := Every(input, func(x int) bool { return x%2 == 0 })
		
		if !result {
			t.Error("Every() = false, want true")
		}
	})
	
	t.Run("every negative", func(t *testing.T) {
		input := []int{2, 4, 5, 8}
		result := Every(input, func(x int) bool { return x%2 == 0 })
		
		if result {
			t.Error("Every() = true, want false")
		}
	})
}

func TestSome(t *testing.T) {
	t.Run("some positive", func(t *testing.T) {
		input := []int{1, 3, 4, 7}
		result := Some(input, func(x int) bool { return x%2 == 0 })
		
		if !result {
			t.Error("Some() = false, want true")
		}
	})
	
	t.Run("some negative", func(t *testing.T) {
		input := []int{1, 3, 5, 7}
		result := Some(input, func(x int) bool { return x%2 == 0 })
		
		if result {
			t.Error("Some() = true, want false")
		}
	})
}

// ===============================
// Utility Tests
// ===============================

func TestIsEmpty(t *testing.T) {
	t.Run("empty slice", func(t *testing.T) {
		var input []int
		if !IsEmpty(input) {
			t.Error("IsEmpty() = false, want true")
		}
	})
	
	t.Run("non-empty slice", func(t *testing.T) {
		input := []int{1}
		if IsEmpty(input) {
			t.Error("IsEmpty() = true, want false")
		}
	})
	
	t.Run("nil slice", func(t *testing.T) {
		if !IsEmpty(([]int)(nil)) {
			t.Error("IsEmpty() = false, want true")
		}
	})
}

func TestCount(t *testing.T) {
	t.Run("count matching elements", func(t *testing.T) {
		input := []int{1, 2, 3, 4, 5, 6}
		result := Count(input, func(x int) bool { return x%2 == 0 })
		expected := 3
		
		if result != expected {
			t.Errorf("Count() = %v, want %v", result, expected)
		}
	})
}

func TestMinMax(t *testing.T) {
	t.Run("min normal case", func(t *testing.T) {
		input := []int{3, 1, 4, 1, 5}
		result, found := Min(input)
		
		if !found || result != 1 {
			t.Errorf("Min() = (%v, %v), want (1, true)", result, found)
		}
	})
	
	t.Run("max normal case", func(t *testing.T) {
		input := []int{3, 1, 4, 1, 5}
		result, found := Max(input)
		
		if !found || result != 5 {
			t.Errorf("Max() = (%v, %v), want (5, true)", result, found)
		}
	})
	
	t.Run("min empty slice", func(t *testing.T) {
		var input []int
		_, found := Min(input)
		
		if found {
			t.Error("Min() found = true, want false")
		}
	})
}

func TestSum(t *testing.T) {
	t.Run("sum integers", func(t *testing.T) {
		input := []int{1, 2, 3, 4, 5}
		result := Sum(input)
		expected := 15
		
		if result != expected {
			t.Errorf("Sum() = %v, want %v", result, expected)
		}
	})
	
	t.Run("sum floats", func(t *testing.T) {
		input := []float64{1.1, 2.2, 3.3}
		result := Sum(input)
		expected := 6.6
		
		if result < expected-0.001 || result > expected+0.001 {
			t.Errorf("Sum() = %v, want %v", result, expected)
		}
	})
}

// ===============================
// Creation and Conversion Tests
// ===============================

func TestRange(t *testing.T) {
	t.Run("range normal case", func(t *testing.T) {
		result := Range(1, 5)
		expected := []int{1, 2, 3, 4}
		
		if !Equal(result, expected) {
			t.Errorf("Range() = %v, want %v", result, expected)
		}
	})
	
	t.Run("range start equals end", func(t *testing.T) {
		result := Range(5, 5)
		
		if result != nil {
			t.Errorf("Range() = %v, want nil", result)
		}
	})
	
	t.Run("range start greater than end", func(t *testing.T) {
		result := Range(5, 3)
		
		if result != nil {
			t.Errorf("Range() = %v, want nil", result)
		}
	})
}

func TestRangeStep(t *testing.T) {
	t.Run("range step positive", func(t *testing.T) {
		result := RangeStep(0, 10, 2)
		expected := []int{0, 2, 4, 6, 8}
		
		if !Equal(result, expected) {
			t.Errorf("RangeStep() = %v, want %v", result, expected)
		}
	})
	
	t.Run("range step negative", func(t *testing.T) {
		result := RangeStep(10, 0, -2)
		expected := []int{10, 8, 6, 4, 2}
		
		if !Equal(result, expected) {
			t.Errorf("RangeStep() = %v, want %v", result, expected)
		}
	})
}

func TestRepeat(t *testing.T) {
	t.Run("repeat normal case", func(t *testing.T) {
		result := Repeat("a", 3)
		expected := []string{"a", "a", "a"}
		
		if !Equal(result, expected) {
			t.Errorf("Repeat() = %v, want %v", result, expected)
		}
	})
	
	t.Run("repeat zero times", func(t *testing.T) {
		result := Repeat(42, 0)
		
		if result != nil {
			t.Errorf("Repeat() = %v, want nil", result)
		}
	})
}

func TestClone(t *testing.T) {
	t.Run("clone normal case", func(t *testing.T) {
		input := []int{1, 2, 3}
		result := Clone(input)
		
		if !Equal(input, result) {
			t.Errorf("Clone() = %v, want %v", result, input)
		}
		
		// Verify it's a different slice
		result[0] = 99
		if input[0] == 99 {
			t.Error("Clone() created reference, not copy")
		}
	})
}

func TestEqual(t *testing.T) {
	t.Run("equal slices", func(t *testing.T) {
		slice1 := []int{1, 2, 3}
		slice2 := []int{1, 2, 3}
		
		if !Equal(slice1, slice2) {
			t.Error("Equal() = false, want true")
		}
	})
	
	t.Run("unequal slices", func(t *testing.T) {
		slice1 := []int{1, 2, 3}
		slice2 := []int{1, 2, 4}
		
		if Equal(slice1, slice2) {
			t.Error("Equal() = true, want false")
		}
	})
	
	t.Run("different length slices", func(t *testing.T) {
		slice1 := []int{1, 2, 3}
		slice2 := []int{1, 2}
		
		if Equal(slice1, slice2) {
			t.Error("Equal() = true, want false")
		}
	})
}

// ===============================
// Advanced Operations Tests
// ===============================

func TestGroupBy(t *testing.T) {
	t.Run("group by length", func(t *testing.T) {
		input := []string{"a", "bb", "ccc", "dd", "e"}
		result := GroupBy(input, func(s string) int { return len(s) })
		
		if len(result[1]) != 2 { // "a", "e"
			t.Errorf("GroupBy()[1] length = %v, want 2", len(result[1]))
		}
		if len(result[2]) != 2 { // "bb", "dd"
			t.Errorf("GroupBy()[2] length = %v, want 2", len(result[2]))
		}
		if len(result[3]) != 1 { // "ccc"
			t.Errorf("GroupBy()[3] length = %v, want 1", len(result[3]))
		}
	})
}

func TestPartition(t *testing.T) {
	t.Run("partition even odd", func(t *testing.T) {
		input := []int{1, 2, 3, 4, 5, 6}
		even, odd := Partition(input, func(x int) bool { return x%2 == 0 })
		
		expectedEven := []int{2, 4, 6}
		expectedOdd := []int{1, 3, 5}
		
		if !Equal(even, expectedEven) {
			t.Errorf("Partition() even = %v, want %v", even, expectedEven)
		}
		if !Equal(odd, expectedOdd) {
			t.Errorf("Partition() odd = %v, want %v", odd, expectedOdd)
		}
	})
}

func TestTake(t *testing.T) {
	t.Run("take normal case", func(t *testing.T) {
		input := []int{1, 2, 3, 4, 5}
		result := Take(input, 3)
		expected := []int{1, 2, 3}
		
		if !Equal(result, expected) {
			t.Errorf("Take() = %v, want %v", result, expected)
		}
	})
	
	t.Run("take more than length", func(t *testing.T) {
		input := []int{1, 2, 3}
		result := Take(input, 5)
		expected := []int{1, 2, 3}
		
		if !Equal(result, expected) {
			t.Errorf("Take() = %v, want %v", result, expected)
		}
	})
}

func TestTakeWhile(t *testing.T) {
	t.Run("take while less than 4", func(t *testing.T) {
		input := []int{1, 2, 3, 5, 6}
		result := TakeWhile(input, func(x int) bool { return x < 4 })
		expected := []int{1, 2, 3}
		
		if !Equal(result, expected) {
			t.Errorf("TakeWhile() = %v, want %v", result, expected)
		}
	})
}

func TestDrop(t *testing.T) {
	t.Run("drop normal case", func(t *testing.T) {
		input := []int{1, 2, 3, 4, 5}
		result := Drop(input, 2)
		expected := []int{3, 4, 5}
		
		if !Equal(result, expected) {
			t.Errorf("Drop() = %v, want %v", result, expected)
		}
	})
}

func TestDropWhile(t *testing.T) {
	t.Run("drop while less than 4", func(t *testing.T) {
		input := []int{1, 2, 3, 5, 6}
		result := DropWhile(input, func(x int) bool { return x < 4 })
		expected := []int{5, 6}
		
		if !Equal(result, expected) {
			t.Errorf("DropWhile() = %v, want %v", result, expected)
		}
	})
}

// ===============================
// Sorting Tests
// ===============================

func TestSort(t *testing.T) {
	t.Run("sort integers", func(t *testing.T) {
		input := []int{3, 1, 4, 1, 5, 9}
		result := Sort(input)
		expected := []int{1, 1, 3, 4, 5, 9}
		
		if !Equal(result, expected) {
			t.Errorf("Sort() = %v, want %v", result, expected)
		}
		
		// Verify original is unchanged
		if Equal(input, result) {
			t.Error("Sort() modified original slice")
		}
	})
}

func TestIsSorted(t *testing.T) {
	t.Run("is sorted true", func(t *testing.T) {
		input := []int{1, 2, 3, 4, 5}
		if !IsSorted(input) {
			t.Error("IsSorted() = false, want true")
		}
	})
	
	t.Run("is sorted false", func(t *testing.T) {
		input := []int{1, 3, 2, 4, 5}
		if IsSorted(input) {
			t.Error("IsSorted() = true, want false")
		}
	})
}

// ===============================
// String Conversion Tests
// ===============================

func TestJoin(t *testing.T) {
	t.Run("join strings", func(t *testing.T) {
		input := []string{"a", "b", "c"}
		result := Join(input, ",")
		expected := "a,b,c"
		
		if result != expected {
			t.Errorf("Join() = %v, want %v", result, expected)
		}
	})
	
	t.Run("join integers", func(t *testing.T) {
		input := []int{1, 2, 3}
		result := Join(input, "-")
		expected := "1-2-3"
		
		if result != expected {
			t.Errorf("Join() = %v, want %v", result, expected)
		}
	})
	
	t.Run("join empty slice", func(t *testing.T) {
		var input []string
		result := Join(input, ",")
		expected := ""
		
		if result != expected {
			t.Errorf("Join() = %v, want %v", result, expected)
		}
	})
	
	t.Run("join single element", func(t *testing.T) {
		input := []string{"alone"}
		result := Join(input, ",")
		expected := "alone"
		
		if result != expected {
			t.Errorf("Join() = %v, want %v", result, expected)
		}
	})
}

// ===============================
// Edge Cases and Error Handling
// ===============================

func TestNilSliceHandling(t *testing.T) {
	t.Run("nil slice operations", func(t *testing.T) {
		var nilSlice []int
		
		// These should return nil without panicking
		if Filter(nilSlice, func(x int) bool { return true }) != nil {
			t.Error("Filter() with nil slice should return nil")
		}
		
		if Map(nilSlice, func(x int) string { return "" }) != nil {
			t.Error("Map() with nil slice should return nil")
		}
		
		if Unique(nilSlice) != nil {
			t.Error("Unique() with nil slice should return nil")
		}
		
		if Reverse(nilSlice) != nil {
			t.Error("Reverse() with nil slice should return nil")
		}
		
		// These should return false/zero values without panicking
		if Contains(nilSlice, 1) {
			t.Error("Contains() with nil slice should return false")
		}
		
		if !IsEmpty(nilSlice) {
			t.Error("IsEmpty() with nil slice should return true")
		}
		
		if Some(nilSlice, func(x int) bool { return true }) {
			t.Error("Some() with nil slice should return false")
		}
	})
}

func TestNilFunctionHandling(t *testing.T) {
	t.Run("nil function operations", func(t *testing.T) {
		input := []int{1, 2, 3}
		
		// These should return nil when function is nil
		if Filter(input, nil) != nil {
			t.Error("Filter() with nil function should return nil")
		}
		
		if Map(input, (func(int) string)(nil)) != nil {
			t.Error("Map() with nil function should return nil")
		}
		
		if ContainsBy(input, nil) {
			t.Error("ContainsBy() with nil function should return false")
		}
		
		if Some(input, nil) {
			t.Error("Some() with nil function should return false")
		}
		
		if Every(input, nil) {
			t.Error("Every() with nil function should return false")
		}
	})
}

// Test helper function to verify our Equal function works correctly
func TestEqualFunction(t *testing.T) {
	t.Run("equal function correctness", func(t *testing.T) {
		// Test various scenarios to ensure our Equal function is reliable
		testCases := []struct {
			name     string
			slice1   []int
			slice2   []int
			expected bool
		}{
			{"both nil", nil, nil, true},
			{"one nil", nil, []int{}, true}, // Both represent empty
			{"same content", []int{1, 2, 3}, []int{1, 2, 3}, true},
			{"different content", []int{1, 2, 3}, []int{1, 2, 4}, false},
			{"different length", []int{1, 2}, []int{1, 2, 3}, false},
			{"both empty", []int{}, []int{}, true},
		}
		
		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				result := Equal(tc.slice1, tc.slice2)
				if result != tc.expected {
					t.Errorf("Equal(%v, %v) = %v, want %v", tc.slice1, tc.slice2, result, tc.expected)
				}
			})
		}
	})
}