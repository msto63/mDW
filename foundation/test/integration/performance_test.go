// File: performance_test.go
// Title: mDW Foundation Performance Integration Tests
// Description: Benchmarks and performance tests for cross-module operations
//              to ensure consistent performance characteristics across modules.
// Author: msto63 with Claude Sonnet 4.0
// Version: v0.1.0
// Created: 2025-01-25
// Modified: 2025-01-25
//
// Change History:
// - 2025-01-25 v0.1.0: Initial implementation of performance integration tests

package integration

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/msto63/mDW/foundation/utils/mathx"
	"github.com/msto63/mDW/foundation/utils/stringx"
	"github.com/msto63/mDW/foundation/utils/timex"
)

// BenchmarkStringToDecimalConversion benchmarks the common pattern of string validation to decimal conversion
func BenchmarkStringToDecimalConversion(b *testing.B) {
	testCases := []string{
		"123.45",
		"0.01",
		"999999.99",
		"0.123456789",
		"12345678901234567890.12",
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		input := testCases[i%len(testCases)]
		
		// Step 1: String validation
		if err := stringx.ValidateRequired(input); err != nil {
			b.Fatal(err)
		}
		
		// Step 2: Decimal conversion
		if _, err := mathx.NewDecimal(input); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkStringProcessingChain benchmarks chained string operations
func BenchmarkStringProcessingChain(b *testing.B) {
	input := "  Hello, World! This is a test string for processing.  "
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Chain of string operations
		s := strings.TrimSpace(input)
		
		// Validation
		if err := stringx.ValidateLength(s, 10, 100); err != nil {
			b.Fatal(err)
		}
		
		// Processing
		s = stringx.Truncate(s, 30, "...")
		s = stringx.PadRight(s, 40, ' ')
		s = strings.ToUpper(s)
		
		// Prevent optimization
		_ = s
	}
}

// BenchmarkDecimalCalculations benchmarks chained decimal operations
func BenchmarkDecimalCalculations(b *testing.B) {
	d1, _ := mathx.NewDecimal("123.45")
	d2, _ := mathx.NewDecimal("67.89")
	taxRate := mathx.NewDecimalFromFloat(0.085)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Financial calculation chain
		subtotal := d1.Add(d2)
		tax := subtotal.Multiply(taxRate)
		total := subtotal.Add(tax)
		
		// Convert back to string (common in real usage)
		_ = total.String()
	}
}

// BenchmarkTimeCalculations benchmarks time-related operations
func BenchmarkTimeCalculations(b *testing.B) {
	now := time.Now()
	pastDate := now.AddDate(-5, -3, -15)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Common time calculations
		age := timex.Age(pastDate, now)
		businessDays := timex.BusinessDaysBetween(pastDate, now)
		
		// Prevent optimization
		_ = age + businessDays
	}
}

// BenchmarkErrorCreation benchmarks error creation patterns
func BenchmarkErrorCreation(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Create different types of errors
		err1 := fmt.Errorf("stringx validation error: %s", "invalid input")
		err2 := fmt.Errorf("mathx calculation error: %s", "division by zero")
		err3 := fmt.Errorf("timex parsing error: %s", "invalid format")
		
		// Use errors to prevent optimization
		_ = err1.Error() + err2.Error() + err3.Error()
	}
}

// BenchmarkCrossModuleDataFlow benchmarks realistic data flow scenarios
func BenchmarkCrossModuleDataFlow(b *testing.B) {
	testData := []struct {
		dateStr   string
		amountStr string
	}{
		{"2023-01-15", "1234.56"},
		{"2023-06-30", "987.65"},
		{"2023-12-25", "555.55"},
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		data := testData[i%len(testData)]
		
		// Step 1: Validate strings
		if err := stringx.ValidateNotBlank(data.dateStr); err != nil {
			b.Fatal(err)
		}
		if err := stringx.ValidateNotBlank(data.amountStr); err != nil {
			b.Fatal(err)
		}
		
		// Step 2: Parse date
		date, err := time.Parse("2006-01-02", data.dateStr)
		if err != nil {
			b.Fatal(err)
		}
		
		// Step 3: Parse amount
		amount, err := mathx.NewDecimal(data.amountStr)
		if err != nil {
			b.Fatal(err)
		}
		
		// Step 4: Perform calculations
		now := time.Now()
		daysOld := int(now.Sub(date).Hours() / 24)
		
		// Apply age-based discount (silly example but realistic pattern)
		discountRate := mathx.NewDecimalFromFloat(float64(daysOld) * 0.001)
		if discountRate.GreaterThan(mathx.NewDecimalFromFloat(0.1)) {
			discountRate = mathx.NewDecimalFromFloat(0.1) // Max 10% discount
		}
		
		discount := amount.Multiply(discountRate)
		finalAmount := amount.Subtract(discount)
		
		// Convert result to string
		_ = finalAmount.String()
	}
}

// Memory allocation benchmarks

// BenchmarkStringOperationsAlloc benchmarks memory allocations in string operations
func BenchmarkStringOperationsAlloc(b *testing.B) {
	input := "test string for memory allocation testing"
	
	b.ReportAllocs()
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		// Operations that should minimize allocations
		result := stringx.PadLeft(input, 50, ' ')
		result = stringx.Truncate(result, 40, "...")
		result = stringx.Center(result, 60, '*')
		
		// Prevent optimization
		_ = result
	}
}

// BenchmarkDecimalOperationsAlloc benchmarks memory allocations in decimal operations
func BenchmarkDecimalOperationsAlloc(b *testing.B) {
	d1, _ := mathx.NewDecimal("123.45")
	d2, _ := mathx.NewDecimal("67.89")
	
	b.ReportAllocs()
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		// Operations that should minimize allocations
		result := d1.Add(d2)
		result = result.Multiply(mathx.NewDecimalFromFloat(1.1))
		result = result.Subtract(mathx.NewDecimalFromFloat(10.0))
		
		// Prevent optimization
		_ = result
	}
}

// Scalability tests

// BenchmarkLargeStringOperations tests performance with large strings
func BenchmarkLargeStringOperations(b *testing.B) {
	sizes := []int{100, 1000, 10000, 100000}
	
	for _, size := range sizes {
		b.Run(fmt.Sprintf("size_%d", size), func(b *testing.B) {
			input := strings.Repeat("A", size)
			
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				// Operations that should scale well
				if err := stringx.ValidateLength(input, 1, size*2); err != nil {
					b.Fatal(err)
				}
				
				truncated := stringx.Truncate(input, size/2, "...")
				_ = stringx.Reverse(truncated)
			}
		})
	}
}

// BenchmarkManyDecimalOperations tests performance with many decimal operations
func BenchmarkManyDecimalOperations(b *testing.B) {
	counts := []int{10, 100, 1000, 10000}
	
	for _, count := range counts {
		b.Run(fmt.Sprintf("count_%d", count), func(b *testing.B) {
			decimals := make([]mathx.Decimal, count)
			for i := 0; i < count; i++ {
				decimals[i], _ = mathx.NewDecimal(fmt.Sprintf("%d.%02d", i, i%100))
			}
			
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				sum := mathx.NewDecimalFromInt(0)
				for _, d := range decimals {
					sum = sum.Add(d)
				}
				
				// Prevent optimization
				_ = sum.String()
			}
		})
	}
}

// Concurrency benchmarks

// BenchmarkConcurrentStringOperations tests thread safety and performance under concurrency
func BenchmarkConcurrentStringOperations(b *testing.B) {
	input := "concurrent test string"
	
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			// Thread-safe operations
			if err := stringx.ValidateRequired(input); err != nil {
				b.Fatal(err)
			}
			
			result := stringx.PadLeft(input, 30, ' ')
			result = stringx.Truncate(result, 25, "...")
			
			// Prevent optimization
			_ = result
		}
	})
}

// BenchmarkConcurrentDecimalOperations tests decimal operations under concurrency
func BenchmarkConcurrentDecimalOperations(b *testing.B) {
	d1, _ := mathx.NewDecimal("123.45")
	d2, _ := mathx.NewDecimal("67.89")
	
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			// Thread-safe operations
			result := d1.Add(d2)
			result = result.Multiply(mathx.NewDecimalFromFloat(1.1))
			
			// Prevent optimization
			_ = result.String()
		}
	})
}

// Real-world scenario benchmarks

// BenchmarkFinancialProcessing benchmarks a realistic financial processing scenario
func BenchmarkFinancialProcessing(b *testing.B) {
	transactions := []struct {
		id     string
		amount string
		date   string
	}{
		{"TXN001", "1234.56", "2023-01-15"},
		{"TXN002", "987.65", "2023-02-20"},
		{"TXN003", "555.55", "2023-03-10"},
		{"TXN004", "2000.00", "2023-04-05"},
		{"TXN005", "750.25", "2023-05-12"},
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		txn := transactions[i%len(transactions)]
		
		// Full processing pipeline
		
		// 1. Validate transaction ID
		if err := stringx.ValidateLength(txn.id, 3, 20); err != nil {
			b.Fatal(err)
		}
		
		// 2. Process amount
		amount, err := mathx.NewDecimal(txn.amount)
		if err != nil {
			b.Fatal(err)
		}
		
		// 3. Parse date
		date, err := time.Parse("2006-01-02", txn.date)
		if err != nil {
			b.Fatal(err)
		}
		
		// 4. Calculate fees (2.5%)
		feeRate := mathx.NewDecimalFromFloat(0.025)
		fee := amount.Multiply(feeRate)
		
		// 5. Calculate net amount
		netAmount := amount.Subtract(fee)
		
		// 6. Check if transaction is old
		now := time.Now()
		age := timex.DaysBetween(date, now)
		
		// 7. Apply aging discount if applicable
		if age > 30 {
			discount := netAmount.Multiply(mathx.NewDecimalFromFloat(0.01)) // 1% discount
			netAmount = netAmount.Subtract(discount)
		}
		
		// 8. Format result
		result := fmt.Sprintf("Transaction %s: %s (net: %s, age: %d days)",
			txn.id, txn.amount, netAmount.String(), age)
		
		// Prevent optimization
		_ = result
	}
}