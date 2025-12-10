// File: benchmark_test.go
// Title: Performance Benchmarks for MathX Functions
// Description: Comprehensive benchmarks for all mathx functions to measure
//              performance and ensure optimal implementations for financial
//              calculations. These benchmarks help identify bottlenecks.
// Author: msto63 with Claude Sonnet 4.0
// Version: v0.1.0
// Created: 2025-01-24
// Modified: 2025-01-24
//
// Change History:
// - 2025-01-24 v0.1.0: Initial benchmark implementation

package mathx

import (
	"testing"
)

// Benchmark decimal creation
func BenchmarkNewDecimal(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, _ = NewDecimal("123.456789")
	}
}

func BenchmarkNewDecimalFromInt(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = NewDecimalFromInt(123456)
	}
}

func BenchmarkNewDecimalFromFloat(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = NewDecimalFromFloat(123.456789)
	}
}

// Benchmark basic arithmetic operations
func BenchmarkDecimalAdd(b *testing.B) {
	d1 := MustNewDecimal("123.456")
	d2 := MustNewDecimal("789.123")
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = d1.Add(d2)
	}
}

func BenchmarkDecimalSubtract(b *testing.B) {
	d1 := MustNewDecimal("123.456")
	d2 := MustNewDecimal("789.123")
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = d1.Subtract(d2)
	}
}

func BenchmarkDecimalMultiply(b *testing.B) {
	d1 := MustNewDecimal("123.456")
	d2 := MustNewDecimal("789.123")
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = d1.Multiply(d2)
	}
}

func BenchmarkDecimalDivide(b *testing.B) {
	d1 := MustNewDecimal("123.456")
	d2 := MustNewDecimal("789.123")
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = d1.Divide(d2)
	}
}

// Benchmark comparison operations
func BenchmarkDecimalCompare(b *testing.B) {
	d1 := MustNewDecimal("123.456")
	d2 := MustNewDecimal("789.123")
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = d1.Compare(d2)
	}
}

func BenchmarkDecimalEqual(b *testing.B) {
	d1 := MustNewDecimal("123.456")
	d2 := MustNewDecimal("123.456")
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = d1.Equal(d2)
	}
}

// Benchmark rounding operations
func BenchmarkDecimalRound(b *testing.B) {
	d := MustNewDecimal("123.456789")
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = d.Round(2, RoundingModeHalfUp)
	}
}

func BenchmarkDecimalStringFixed(b *testing.B) {
	d := MustNewDecimal("123.456789")
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = d.StringFixed(2)
	}
}

// Benchmark advanced operations
func BenchmarkDecimalPow(b *testing.B) {
	d := MustNewDecimal("2.5")
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = d.Pow(3)
	}
}

func BenchmarkDecimalSqrt(b *testing.B) {
	d := MustNewDecimal("123.456")
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = d.Sqrt()
	}
}

// Benchmark currency operations
func BenchmarkNewMoney(b *testing.B) {
	amount := MustNewDecimal("123.456")
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = NewMoney(amount, USD)
	}
}

func BenchmarkNewMoneyFromString(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, _ = NewMoneyFromString("123.45", "USD")
	}
}

func BenchmarkMoneyAdd(b *testing.B) {
	m1 := MustNewMoneyFromString("123.45", "USD")
	m2 := MustNewMoneyFromString("67.89", "USD")
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = m1.Add(m2)
	}
}

func BenchmarkMoneyMultiply(b *testing.B) {
	money := MustNewMoneyFromString("123.45", "USD")
	factor := MustNewDecimal("1.05")
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = money.Multiply(factor)
	}
}

func BenchmarkMoneyAllocate(b *testing.B) {
	money := MustNewMoneyFromString("1000.00", "USD")
	ratios := []Decimal{
		MustNewDecimal("1"),
		MustNewDecimal("2"),
		MustNewDecimal("3"),
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = money.Allocate(ratios...)
	}
}

func BenchmarkMoneyFormat(b *testing.B) {
	money := MustNewMoneyFromString("123.45", "USD")
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = money.Format()
	}
}

// Benchmark business calculations
func BenchmarkCalculatePercentage(b *testing.B) {
	value := MustNewDecimal("1000")
	percentage := MustNewDecimal("15")
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = CalculatePercentage(value, percentage)
	}
}

func BenchmarkCalculateTax(b *testing.B) {
	amount := MustNewDecimal("1000")
	taxRate := MustNewDecimal("19")
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = CalculateTax(amount, taxRate)
	}
}

func BenchmarkCalculateSimpleInterest(b *testing.B) {
	principal := MustNewDecimal("10000")
	rate := MustNewDecimal("0.05")
	time := MustNewDecimal("2")
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = CalculateSimpleInterest(principal, rate, time)
	}
}

func BenchmarkCalculateCompoundInterest(b *testing.B) {
	principal := MustNewDecimal("10000")
	annualRate := MustNewDecimal("0.05")
	compoundingFrequency := int64(12)
	timeInYears := MustNewDecimal("5")
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = CalculateCompoundInterest(principal, annualRate, compoundingFrequency, timeInYears)
	}
}

func BenchmarkCalculateLoanPayment(b *testing.B) {
	principal := MustNewDecimal("250000")
	annualRate := MustNewDecimal("4.5")
	months := int64(360)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = CalculateLoanPayment(principal, annualRate, months)
	}
}

func BenchmarkCalculateROI(b *testing.B) {
	initialInvestment := MustNewDecimal("10000")
	currentValue := MustNewDecimal("12500")
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = CalculateROI(initialInvestment, currentValue)
	}
}

// Benchmark statistical functions
func BenchmarkCalculateAverageDecimal(b *testing.B) {
	values := []Decimal{
		MustNewDecimal("10.5"),
		MustNewDecimal("20.3"),
		MustNewDecimal("30.7"),
		MustNewDecimal("40.1"),
		MustNewDecimal("50.9"),
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = CalculateAverageDecimal(values...)
	}
}

func BenchmarkFindMinDecimal(b *testing.B) {
	values := []Decimal{
		MustNewDecimal("50.9"),
		MustNewDecimal("10.5"),
		MustNewDecimal("40.1"),
		MustNewDecimal("20.3"),
		MustNewDecimal("30.7"),
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = FindMinDecimal(values...)
	}
}

func BenchmarkSumDecimal(b *testing.B) {
	values := []Decimal{
		MustNewDecimal("10.5"),
		MustNewDecimal("20.3"),
		MustNewDecimal("30.7"),
		MustNewDecimal("40.1"),
		MustNewDecimal("50.9"),
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = SumDecimal(values...)
	}
}

// Memory allocation benchmarks
func BenchmarkDecimalAddAllocs(b *testing.B) {
	d1 := MustNewDecimal("123.456")
	d2 := MustNewDecimal("789.123")
	
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = d1.Add(d2)
	}
}

func BenchmarkDecimalMultiplyAllocs(b *testing.B) {
	d1 := MustNewDecimal("123.456")
	d2 := MustNewDecimal("789.123")
	
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = d1.Multiply(d2)
	}
}

func BenchmarkMoneyFormatAllocs(b *testing.B) {
	money := MustNewMoneyFromString("123.45", "USD")
	
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = money.Format()
	}
}

// Benchmark with different decimal sizes
func BenchmarkDecimalSmallNumbers(b *testing.B) {
	d1 := MustNewDecimal("1.23")
	d2 := MustNewDecimal("4.56")
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = d1.Add(d2)
	}
}

func BenchmarkDecimalLargeNumbers(b *testing.B) {
	d1 := MustNewDecimal("123456789.123456789")
	d2 := MustNewDecimal("987654321.987654321")
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = d1.Add(d2)
	}
}

func BenchmarkDecimalHighPrecision(b *testing.B) {
	d1 := MustNewDecimal("123.123456789012345678901234567890")
	d2 := MustNewDecimal("456.987654321098765432109876543210")
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = d1.Multiply(d2)
	}
}

// Benchmark currency registry operations
func BenchmarkGetCurrency(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, _ = GetCurrency("USD")
	}
}

func BenchmarkFormatCurrency(b *testing.B) {
	amount := MustNewDecimal("123.45")
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = FormatCurrency(amount, "USD", 2)
	}
}

// Benchmark complex financial calculations
func BenchmarkComplexFinancialScenario(b *testing.B) {
	// Simulate a complex scenario with multiple calculations
	principal := MustNewDecimal("100000")
	rate := MustNewDecimal("0.05")
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Calculate compound interest
		amount, _ := CalculateCompoundInterest(principal, rate, 12, MustNewDecimal("5"))
		
		// Calculate tax on gains
		gains := amount.Subtract(principal)
		tax := CalculateTax(gains, MustNewDecimal("25"))
		
		// Calculate final amount after tax
		_ = amount.Subtract(tax)
	}
}