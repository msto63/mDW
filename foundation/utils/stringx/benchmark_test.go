// File: benchmark_test.go
// Title: Performance Benchmarks for StringX Functions
// Description: Comprehensive benchmarks for all stringx functions to measure
//              performance and ensure optimal implementations. These benchmarks
//              help identify performance regressions and optimization opportunities.
// Author: msto63 with Claude Sonnet 4.0
// Version: v0.1.0
// Created: 2025-01-24
// Modified: 2025-01-24
//
// Change History:
// - 2025-01-24 v0.1.0: Initial benchmark implementation

package stringx

import (
	"strings"
	"testing"
)

// Benchmark core string utility functions
func BenchmarkIsEmpty(b *testing.B) {
	testStrings := []string{"", "hello", "hello world with some text"}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = IsEmpty(testStrings[i%len(testStrings)])
	}
}

func BenchmarkIsBlank(b *testing.B) {
	testStrings := []string{"", "   ", "hello", "  hello  ", "hello world with some text"}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = IsBlank(testStrings[i%len(testStrings)])
	}
}

func BenchmarkTruncate(b *testing.B) {
	text := "This is a longer text that will be truncated in the benchmark test"
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = Truncate(text, 20, "...")
	}
}

func BenchmarkTruncateUnicode(b *testing.B) {
	text := "これは日本語のテキストで、ベンチマークテストで切り捨てられます"
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = Truncate(text, 10, "...")
	}
}

func BenchmarkReverse(b *testing.B) {
	text := "hello world this is a test string"
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = Reverse(text)
	}
}

func BenchmarkReverseUnicode(b *testing.B) {
	text := "こんにちは世界、これはテスト文字列です"
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = Reverse(text)
	}
}

func BenchmarkContainsIgnoreCase(b *testing.B) {
	text := "Hello World This Is A Test String With Mixed Case"
	substr := "test string"
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ContainsIgnoreCase(text, substr)
	}
}

// Compare with standard library approach
func BenchmarkContainsIgnoreCaseStdLib(b *testing.B) {
	text := "Hello World This Is A Test String With Mixed Case"
	substr := "test string"
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = strings.Contains(strings.ToLower(text), strings.ToLower(substr))
	}
}

func BenchmarkPadLeft(b *testing.B) {
	text := "hello"
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = PadLeft(text, 20, ' ')
	}
}

func BenchmarkPadRight(b *testing.B) {
	text := "hello"
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = PadRight(text, 20, ' ')
	}
}

func BenchmarkCenter(b *testing.B) {
	text := "hello"
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = Center(text, 20, ' ')
	}
}

func BenchmarkSplitLines(b *testing.B) {
	text := "line1\nline2\r\nline3\rline4\nline5\r\nline6"
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = SplitLines(text)
	}
}

// Benchmark case conversion functions
func BenchmarkToSnakeCase(b *testing.B) {
	text := "ThisIsAVeryLongVariableNameInPascalCase"
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ToSnakeCase(text)
	}
}

func BenchmarkToCamelCase(b *testing.B) {
	text := "this_is_a_very_long_variable_name_in_snake_case"
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ToCamelCase(text)
	}
}

func BenchmarkToPascalCase(b *testing.B) {
	text := "this_is_a_very_long_variable_name_in_snake_case"
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ToPascalCase(text)
	}
}

func BenchmarkToKebabCase(b *testing.B) {
	text := "ThisIsAVeryLongVariableNameInPascalCase"
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ToKebabCase(text)
	}
}

func BenchmarkToTitleCase(b *testing.B) {
	text := "this is a long sentence that will be converted to title case"
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ToTitleCase(text)
	}
}

// Memory allocation benchmarks
func BenchmarkTruncateAllocs(b *testing.B) {
	text := "This is a text that will be truncated"
	
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = Truncate(text, 10, "...")
	}
}

func BenchmarkReverseAllocs(b *testing.B) {
	text := "hello world test string"
	
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = Reverse(text)
	}
}

func BenchmarkToSnakeCaseAllocs(b *testing.B) {
	text := "ThisIsALongVariableName"
	
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ToSnakeCase(text)
	}
}

// Benchmark with different string sizes
func BenchmarkTruncateSmall(b *testing.B) {
	text := "small"
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = Truncate(text, 10, "...")
	}
}

func BenchmarkTruncateMedium(b *testing.B) {
	text := "This is a medium-sized text for benchmarking"
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = Truncate(text, 20, "...")
	}
}

func BenchmarkTruncateLarge(b *testing.B) {
	text := strings.Repeat("This is a large text for benchmarking purposes ", 10)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = Truncate(text, 50, "...")
	}
}

// Benchmark FirstNonEmpty and FirstNonBlank with different scenarios
func BenchmarkFirstNonEmptyAllEmpty(b *testing.B) {
	strings := []string{"", "", "", ""}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = FirstNonEmpty(strings...)
	}
}

func BenchmarkFirstNonEmptyFirstFound(b *testing.B) {
	strings := []string{"found", "", "", ""}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = FirstNonEmpty(strings...)
	}
}

func BenchmarkFirstNonEmptyLastFound(b *testing.B) {
	strings := []string{"", "", "", "found"}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = FirstNonEmpty(strings...)
	}
}

func BenchmarkFirstNonBlankAllBlank(b *testing.B) {
	strings := []string{"", "  ", "\t", "\n"}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = FirstNonBlank(strings...)
	}
}

func BenchmarkFirstNonBlankFirstFound(b *testing.B) {
	strings := []string{"found", "", "  ", "\t"}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = FirstNonBlank(strings...)
	}
}