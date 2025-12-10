// File: benchmark_test.go
// Title: Slice Utilities Benchmarks
// Description: Performance benchmarks for slicex utility functions to ensure
//              optimal performance and identify potential bottlenecks.
// Author: msto63 with Claude Sonnet 4.0
// Version: v0.1.0
// Created: 2025-01-25
// Modified: 2025-01-25
//
// Change History:
// - 2025-01-25 v0.1.0: Initial benchmark implementation

package slicex

import (
	"strconv"
	"testing"
)

// ===============================
// Core Transformation Benchmarks
// ===============================

func BenchmarkFilter(b *testing.B) {
	sizes := []int{100, 1000, 10000}
	
	for _, size := range sizes {
		input := Range(1, size+1)
		b.Run("size_"+strconv.Itoa(size), func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				Filter(input, func(x int) bool { return x%2 == 0 })
			}
		})
	}
}

func BenchmarkMap(b *testing.B) {
	sizes := []int{100, 1000, 10000}
	
	for _, size := range sizes {
		input := Range(1, size+1)
		b.Run("size_"+strconv.Itoa(size), func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				Map(input, func(x int) string { return strconv.Itoa(x) })
			}
		})
	}
}

func BenchmarkMapWithIndex(b *testing.B) {
	sizes := []int{100, 1000, 10000}
	
	for _, size := range sizes {
		input := Range(1, size+1)
		b.Run("size_"+strconv.Itoa(size), func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				MapWithIndex(input, func(i, x int) string { return strconv.Itoa(i) + ":" + strconv.Itoa(x) })
			}
		})
	}
}

func BenchmarkReduce(b *testing.B) {
	sizes := []int{100, 1000, 10000}
	
	for _, size := range sizes {
		input := Range(1, size+1)
		b.Run("size_"+strconv.Itoa(size), func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				Reduce(input, 0, func(acc, x int) int { return acc + x })
			}
		})
	}
}

// ===============================
// Slice Manipulation Benchmarks
// ===============================

func BenchmarkChunk(b *testing.B) {
	input := Range(1, 10001)
	chunkSizes := []int{10, 100, 1000}
	
	for _, chunkSize := range chunkSizes {
		b.Run("chunk_size_"+strconv.Itoa(chunkSize), func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				Chunk(input, chunkSize)
			}
		})
	}
}

func BenchmarkFlatten(b *testing.B) {
	// Create nested slices for testing
	input := make([][]int, 100)
	for i := range input {
		input[i] = Range(i*10, (i+1)*10)
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Flatten(input)
	}
}

func BenchmarkUnique(b *testing.B) {
	sizes := []int{100, 1000, 10000}
	
	for _, size := range sizes {
		// Create slice with some duplicates
		input := make([]int, size)
		for i := 0; i < size; i++ {
			input[i] = i % (size / 10) // 10% unique values
		}
		
		b.Run("size_"+strconv.Itoa(size), func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				Unique(input)
			}
		})
	}
}

func BenchmarkUniqueBy(b *testing.B) {
	type Person struct {
		ID   int
		Name string
	}
	
	sizes := []int{100, 1000, 10000}
	
	for _, size := range sizes {
		input := make([]Person, size)
		for i := 0; i < size; i++ {
			input[i] = Person{ID: i % (size / 10), Name: "Person" + strconv.Itoa(i)}
		}
		
		b.Run("size_"+strconv.Itoa(size), func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				UniqueBy(input, func(p Person) int { return p.ID })
			}
		})
	}
}

func BenchmarkUnion(b *testing.B) {
	sizes := []int{100, 1000, 10000}
	
	for _, size := range sizes {
		slice1 := Range(1, size/2+1)
		slice2 := Range(size/4, size+1)
		
		b.Run("size_"+strconv.Itoa(size), func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				Union(slice1, slice2)
			}
		})
	}
}

func BenchmarkIntersect(b *testing.B) {
	sizes := []int{100, 1000, 10000}
	
	for _, size := range sizes {
		slice1 := Range(1, size+1)
		slice2 := Range(size/2, size+size/2+1)
		
		b.Run("size_"+strconv.Itoa(size), func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				Intersect(slice1, slice2)
			}
		})
	}
}

func BenchmarkDifference(b *testing.B) {
	sizes := []int{100, 1000, 10000}
	
	for _, size := range sizes {
		slice1 := Range(1, size+1)
		slice2 := Range(size/2, size+1)
		
		b.Run("size_"+strconv.Itoa(size), func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				Difference(slice1, slice2)
			}
		})
	}
}

func BenchmarkReverse(b *testing.B) {
	sizes := []int{100, 1000, 10000}
	
	for _, size := range sizes {
		input := Range(1, size+1)
		
		b.Run("size_"+strconv.Itoa(size), func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				Reverse(input)
			}
		})
	}
}

// ===============================
// Search and Validation Benchmarks
// ===============================

func BenchmarkContains(b *testing.B) {
	sizes := []int{100, 1000, 10000}
	
	for _, size := range sizes {
		input := Range(1, size+1)
		target := size / 2 // Element in the middle
		
		b.Run("size_"+strconv.Itoa(size), func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				Contains(input, target)
			}
		})
	}
}

func BenchmarkContainsBy(b *testing.B) {
	sizes := []int{100, 1000, 10000}
	
	for _, size := range sizes {
		input := Range(1, size+1)
		
		b.Run("size_"+strconv.Itoa(size), func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				ContainsBy(input, func(x int) bool { return x == size/2 })
			}
		})
	}
}

func BenchmarkIndexOf(b *testing.B) {
	sizes := []int{100, 1000, 10000}
	
	for _, size := range sizes {
		input := Range(1, size+1)
		target := size / 2
		
		b.Run("size_"+strconv.Itoa(size), func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				IndexOf(input, target)
			}
		})
	}
}

func BenchmarkFind(b *testing.B) {
	sizes := []int{100, 1000, 10000}
	
	for _, size := range sizes {
		input := Range(1, size+1)
		
		b.Run("size_"+strconv.Itoa(size), func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				Find(input, func(x int) bool { return x == size/2 })
			}
		})
	}
}

func BenchmarkEvery(b *testing.B) {
	sizes := []int{100, 1000, 10000}
	
	for _, size := range sizes {
		input := Range(1, size+1)
		
		b.Run("size_"+strconv.Itoa(size), func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				Every(input, func(x int) bool { return x > 0 })
			}
		})
	}
}

func BenchmarkSome(b *testing.B) {
	sizes := []int{100, 1000, 10000}
	
	for _, size := range sizes {
		input := Range(1, size+1)
		
		b.Run("size_"+strconv.Itoa(size), func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				Some(input, func(x int) bool { return x > size/2 })
			}
		})
	}
}

// ===============================
// Utility Benchmarks
// ===============================

func BenchmarkCount(b *testing.B) {
	sizes := []int{100, 1000, 10000}
	
	for _, size := range sizes {
		input := Range(1, size+1)
		
		b.Run("size_"+strconv.Itoa(size), func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				Count(input, func(x int) bool { return x%2 == 0 })
			}
		})
	}
}

func BenchmarkMin(b *testing.B) {
	sizes := []int{100, 1000, 10000}
	
	for _, size := range sizes {
		input := make([]int, size)
		for i := 0; i < size; i++ {
			input[i] = size - i // Reverse order for worst case
		}
		
		b.Run("size_"+strconv.Itoa(size), func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				Min(input)
			}
		})
	}
}

func BenchmarkMax(b *testing.B) {
	sizes := []int{100, 1000, 10000}
	
	for _, size := range sizes {
		input := Range(1, size+1)
		
		b.Run("size_"+strconv.Itoa(size), func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				Max(input)
			}
		})
	}
}

func BenchmarkSum(b *testing.B) {
	sizes := []int{100, 1000, 10000}
	
	for _, size := range sizes {
		input := Range(1, size+1)
		
		b.Run("size_"+strconv.Itoa(size), func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				Sum(input)
			}
		})
	}
}

// ===============================
// Creation and Conversion Benchmarks
// ===============================

func BenchmarkRange(b *testing.B) {
	sizes := []int{100, 1000, 10000}
	
	for _, size := range sizes {
		b.Run("size_"+strconv.Itoa(size), func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				Range(1, size+1)
			}
		})
	}
}

func BenchmarkRepeat(b *testing.B) {
	sizes := []int{100, 1000, 10000}
	
	for _, size := range sizes {
		b.Run("size_"+strconv.Itoa(size), func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				Repeat(42, size)
			}
		})
	}
}

func BenchmarkClone(b *testing.B) {
	sizes := []int{100, 1000, 10000}
	
	for _, size := range sizes {
		input := Range(1, size+1)
		
		b.Run("size_"+strconv.Itoa(size), func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				Clone(input)
			}
		})
	}
}

func BenchmarkEqual(b *testing.B) {
	sizes := []int{100, 1000, 10000}
	
	for _, size := range sizes {
		slice1 := Range(1, size+1)
		slice2 := Range(1, size+1)
		
		b.Run("size_"+strconv.Itoa(size), func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				Equal(slice1, slice2)
			}
		})
	}
}

// ===============================
// Advanced Operations Benchmarks
// ===============================

func BenchmarkGroupBy(b *testing.B) {
	sizes := []int{100, 1000, 10000}
	
	for _, size := range sizes {
		input := Range(1, size+1)
		
		b.Run("size_"+strconv.Itoa(size), func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				GroupBy(input, func(x int) int { return x % 10 })
			}
		})
	}
}

func BenchmarkPartition(b *testing.B) {
	sizes := []int{100, 1000, 10000}
	
	for _, size := range sizes {
		input := Range(1, size+1)
		
		b.Run("size_"+strconv.Itoa(size), func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				Partition(input, func(x int) bool { return x%2 == 0 })
			}
		})
	}
}

func BenchmarkTake(b *testing.B) {
	input := Range(1, 10001)
	takeSizes := []int{10, 100, 1000}
	
	for _, takeSize := range takeSizes {
		b.Run("take_"+strconv.Itoa(takeSize), func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				Take(input, takeSize)
			}
		})
	}
}

func BenchmarkDrop(b *testing.B) {
	input := Range(1, 10001)
	dropSizes := []int{10, 100, 1000}
	
	for _, dropSize := range dropSizes {
		b.Run("drop_"+strconv.Itoa(dropSize), func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				Drop(input, dropSize)
			}
		})
	}
}

// ===============================
// Sorting Benchmarks
// ===============================

func BenchmarkSort(b *testing.B) {
	sizes := []int{100, 1000, 10000}
	
	for _, size := range sizes {
		// Create reverse-sorted slice for worst case
		input := make([]int, size)
		for i := 0; i < size; i++ {
			input[i] = size - i
		}
		
		b.Run("size_"+strconv.Itoa(size), func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				b.StopTimer()
				testSlice := Clone(input) // Reset for each iteration
				b.StartTimer()
				Sort(testSlice)
			}
		})
	}
}

func BenchmarkIsSorted(b *testing.B) {
	sizes := []int{100, 1000, 10000}
	
	for _, size := range sizes {
		input := Range(1, size+1) // Already sorted
		
		b.Run("size_"+strconv.Itoa(size), func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				IsSorted(input)
			}
		})
	}
}

// ===============================
// String Conversion Benchmarks
// ===============================

func BenchmarkJoin(b *testing.B) {
	sizes := []int{100, 1000, 10000}
	
	for _, size := range sizes {
		input := Map(Range(1, size+1), func(x int) string { return strconv.Itoa(x) })
		
		b.Run("size_"+strconv.Itoa(size), func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				Join(input, ",")
			}
		})
	}
}

// ===============================
// Memory Allocation Benchmarks
// ===============================

func BenchmarkMemoryUsage(b *testing.B) {
	// Test memory usage of common operations
	input := Range(1, 1001)
	
	b.Run("Filter_Memory", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			Filter(input, func(x int) bool { return x%2 == 0 })
		}
	})
	
	b.Run("Map_Memory", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			Map(input, func(x int) string { return strconv.Itoa(x) })
		}
	})
	
	b.Run("Unique_Memory", func(b *testing.B) {
		// Create slice with duplicates
		duplicateInput := make([]int, 1000)
		for i := 0; i < 1000; i++ {
			duplicateInput[i] = i % 100
		}
		
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			Unique(duplicateInput)
		}
	})
	
	b.Run("Clone_Memory", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			Clone(input)
		}
	})
}

// ===============================
// Comparison Benchmarks
// ===============================

func BenchmarkStandardLibraryComparison(b *testing.B) {
	input := make([]int, 10000)
	for i := 0; i < 10000; i++ {
		input[i] = i
	}
	
	// Compare our implementations with standard library where applicable
	b.Run("SliceX_Contains", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			Contains(input, 5000)
		}
	})
	
	// Note: Go standard library doesn't have a direct Contains equivalent for slices,
	// but we can simulate manual search
	b.Run("Manual_Contains", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			found := false
			for _, v := range input {
				if v == 5000 {
					found = true
					break
				}
			}
			_ = found
		}
	})
}