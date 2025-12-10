// File: benchmark_test.go
// Title: Map Utilities Benchmarks
// Description: Performance benchmarks for map utility functions to ensure
//              optimal performance in enterprise applications.
// Author: msto63 with Claude Sonnet 4.0
// Version: v0.1.0
// Created: 2025-01-24
// Modified: 2025-01-24
//
// Change History:
// - 2025-01-24 v0.1.0: Initial implementation with comprehensive benchmarks

package mapx

import (
	"strconv"
	"testing"
)

// Helper function to create test maps of various sizes
func createTestMap(size int) map[string]int {
	m := make(map[string]int, size)
	for i := 0; i < size; i++ {
		m["key"+strconv.Itoa(i)] = i
	}
	return m
}

func BenchmarkKeys(b *testing.B) {
	sizes := []int{10, 100, 1000, 10000}
	
	for _, size := range sizes {
		b.Run("size_"+strconv.Itoa(size), func(b *testing.B) {
			m := createTestMap(size)
			b.ResetTimer()
			
			for i := 0; i < b.N; i++ {
				_ = Keys(m)
			}
		})
	}
}

func BenchmarkValues(b *testing.B) {
	sizes := []int{10, 100, 1000, 10000}
	
	for _, size := range sizes {
		b.Run("size_"+strconv.Itoa(size), func(b *testing.B) {
			m := createTestMap(size)
			b.ResetTimer()
			
			for i := 0; i < b.N; i++ {
				_ = Values(m)
			}
		})
	}
}

func BenchmarkClone(b *testing.B) {
	sizes := []int{10, 100, 1000, 10000}
	
	for _, size := range sizes {
		b.Run("size_"+strconv.Itoa(size), func(b *testing.B) {
			m := createTestMap(size)
			b.ResetTimer()
			
			for i := 0; i < b.N; i++ {
				_ = Clone(m)
			}
		})
	}
}

func BenchmarkMerge(b *testing.B) {
	m1 := createTestMap(1000)
	m2 := createTestMap(1000)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = Merge(m1, m2)
	}
}

func BenchmarkMergeMultiple(b *testing.B) {
	maps := make([]map[string]int, 10)
	for i := range maps {
		maps[i] = createTestMap(100)
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = Merge(maps...)
	}
}

func BenchmarkFilter(b *testing.B) {
	m := createTestMap(1000)
	predicate := func(k string, v int) bool {
		return v%2 == 0
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = Filter(m, predicate)
	}
}

func BenchmarkFilterKeys(b *testing.B) {
	m := createTestMap(1000)
	predicate := func(k string) bool {
		return len(k) > 4
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = FilterKeys(m, predicate)
	}
}

func BenchmarkFilterValues(b *testing.B) {
	m := createTestMap(1000)
	predicate := func(v int) bool {
		return v%2 == 0
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = FilterValues(m, predicate)
	}
}

func BenchmarkTransform(b *testing.B) {
	m := createTestMap(1000)
	transformer := func(k string, v int) string {
		return k + strconv.Itoa(v)
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = Transform(m, transformer)
	}
}

func BenchmarkTransformValues(b *testing.B) {
	m := createTestMap(1000)
	transformer := func(v int) int {
		return v * 2
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = TransformValues(m, transformer)
	}
}

func BenchmarkPick(b *testing.B) {
	m := createTestMap(1000)
	keys := []string{"key100", "key200", "key300", "key400", "key500"}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = Pick(m, keys...)
	}
}

func BenchmarkOmit(b *testing.B) {
	m := createTestMap(1000)
	keys := []string{"key100", "key200", "key300", "key400", "key500"}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = Omit(m, keys...)
	}
}

func BenchmarkIntersect(b *testing.B) {
	m1 := createTestMap(1000)
	m2 := createTestMap(1000)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = Intersect(m1, m2)
	}
}

func BenchmarkDifference(b *testing.B) {
	m1 := createTestMap(1000)
	m2 := createTestMap(500)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = Difference(m1, m2)
	}
}

func BenchmarkUnion(b *testing.B) {
	m1 := createTestMap(1000)
	m2 := createTestMap(1000)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = Union(m1, m2)
	}
}

func BenchmarkEqual(b *testing.B) {
	m1 := createTestMap(1000)
	m2 := Clone(m1)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = Equal(m1, m2)
	}
}

func BenchmarkHasKey(b *testing.B) {
	m := createTestMap(1000)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = HasKey(m, "key500")
	}
}

func BenchmarkHasValue(b *testing.B) {
	m := createTestMap(1000)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = HasValue(m, 500)
	}
}

func BenchmarkToSlice(b *testing.B) {
	m := createTestMap(1000)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ToSlice(m)
	}
}

func BenchmarkFromSlice(b *testing.B) {
	entries := make([]Entry[string, int], 1000)
	for i := 0; i < 1000; i++ {
		entries[i] = Entry[string, int]{
			Key:   "key" + strconv.Itoa(i),
			Value: i,
		}
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = FromSlice(entries)
	}
}

func BenchmarkToJSON(b *testing.B) {
	m := createTestMap(100) // Smaller size for JSON operations
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = ToJSON(m)
	}
}

func BenchmarkFromJSON(b *testing.B) {
	m := createTestMap(100)
	jsonStr, _ := ToJSON(m)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = FromJSON[string, int](jsonStr)
	}
}

func BenchmarkForEach(b *testing.B) {
	m := createTestMap(1000)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ForEach(m, func(k string, v int) {
			// Minimal operation to avoid optimizing away the loop
			_ = k + strconv.Itoa(v)
		})
	}
}

func BenchmarkClear(b *testing.B) {
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		m := createTestMap(1000)
		b.StartTimer()
		
		Clear(m)
	}
}

// Memory allocation benchmarks
func BenchmarkCloneMemory(b *testing.B) {
	m := createTestMap(1000)
	
	b.ReportAllocs()
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		result := Clone(m)
		_ = result
	}
}

func BenchmarkMergeMemory(b *testing.B) {
	m1 := createTestMap(500)
	m2 := createTestMap(500)
	
	b.ReportAllocs()
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		result := Merge(m1, m2)
		_ = result
	}
}

func BenchmarkFilterMemory(b *testing.B) {
	m := createTestMap(1000)
	predicate := func(k string, v int) bool {
		return v%2 == 0
	}
	
	b.ReportAllocs()
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		result := Filter(m, predicate)
		_ = result
	}
}