// File: business_test.go
// Title: Unit Tests for Business Calculations
// Description: Comprehensive unit tests for business calculation functions
//              including interest, loans, taxes, discounts, and financial
//              formulas commonly used in enterprise applications.
// Author: msto63 with Claude Sonnet 4.0
// Version: v0.1.0
// Created: 2025-01-24
// Modified: 2025-01-24
//
// Change History:
// - 2025-01-24 v0.1.0: Initial test implementation for business calculations

package mathx

import (
	"testing"
)

func TestCalculatePercentage(t *testing.T) {
	tests := []struct {
		value      string
		percentage string
		want       string
	}{
		{"100", "20", "20"},
		{"150", "10", "15"},
		{"50", "50", "25"},
		{"0", "10", "0"},
		{"100", "0", "0"},
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			value := MustNewDecimal(tt.value)
			percentage := MustNewDecimal(tt.percentage)
			result := CalculatePercentage(value, percentage)
			
			if result.String() != tt.want {
				t.Errorf("CalculatePercentage(%s, %s) = %s, want %s", 
					tt.value, tt.percentage, result.String(), tt.want)
			}
		})
	}
}

func TestCalculatePercentageOf(t *testing.T) {
	tests := []struct {
		part    string
		whole   string
		want    string
		wantErr bool
	}{
		{"25", "100", "25", false},
		{"30", "150", "20", false},
		{"0", "100", "0", false},
		{"50", "0", "", true}, // Division by zero
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			part := MustNewDecimal(tt.part)
			whole := MustNewDecimal(tt.whole)
			result, err := CalculatePercentageOf(part, whole)
			
			if tt.wantErr {
				if err == nil {
					t.Errorf("CalculatePercentageOf(%s, %s) expected error", tt.part, tt.whole)
				}
				return
			}
			
			if err != nil {
				t.Errorf("CalculatePercentageOf(%s, %s) unexpected error: %v", 
					tt.part, tt.whole, err)
				return
			}
			
			if result.String() != tt.want {
				t.Errorf("CalculatePercentageOf(%s, %s) = %s, want %s", 
					tt.part, tt.whole, result.String(), tt.want)
			}
		})
	}
}

func TestDiscountCalculations(t *testing.T) {
	originalPrice := MustNewDecimal("100")
	discountPercent := MustNewDecimal("20")
	
	// Test discount amount
	discount := CalculateDiscount(originalPrice, discountPercent)
	if discount.String() != "20" {
		t.Errorf("CalculateDiscount(100, 20) = %s, want 20", discount.String())
	}
	
	// Test apply discount
	finalPrice := ApplyDiscount(originalPrice, discountPercent)
	if finalPrice.String() != "80" {
		t.Errorf("ApplyDiscount(100, 20) = %s, want 80", finalPrice.String())
	}
}

func TestMarkupCalculations(t *testing.T) {
	cost := MustNewDecimal("80")
	markupPercent := MustNewDecimal("25")
	
	// Test markup amount
	markup := CalculateMarkup(cost, markupPercent)
	if markup.String() != "20" {
		t.Errorf("CalculateMarkup(80, 25) = %s, want 20", markup.String())
	}
	
	// Test apply markup
	sellingPrice := ApplyMarkup(cost, markupPercent)
	if sellingPrice.String() != "100" {
		t.Errorf("ApplyMarkup(80, 25) = %s, want 100", sellingPrice.String())
	}
}

func TestTaxCalculations(t *testing.T) {
	netAmount := MustNewDecimal("100")
	taxRate := MustNewDecimal("19")
	
	// Test tax calculation
	tax := CalculateTax(netAmount, taxRate)
	if tax.String() != "19" {
		t.Errorf("CalculateTax(100, 19) = %s, want 19", tax.String())
	}
	
	// Test tax-inclusive price
	grossPrice := CalculateTaxInclusivePrice(netAmount, taxRate)
	if grossPrice.String() != "119" {
		t.Errorf("CalculateTaxInclusivePrice(100, 19) = %s, want 119", grossPrice.String())
	}
	
	// Test calculate net from gross
	grossAmount := MustNewDecimal("119")
	calculatedNet := CalculateNetFromGross(grossAmount, taxRate)
	
	// Should be approximately 100 (allowing for rounding)
	diff := calculatedNet.Subtract(netAmount).Abs()
	tolerance := MustNewDecimal("0.01")
	if diff.GreaterThan(tolerance) {
		t.Errorf("CalculateNetFromGross(119, 19) = %s, want approximately 100", 
			calculatedNet.String())
	}
}

func TestSimpleInterest(t *testing.T) {
	principal := MustNewDecimal("1000")
	rate := MustNewDecimal("0.05") // 5%
	time := MustNewDecimal("2")    // 2 years
	
	interest := CalculateSimpleInterest(principal, rate, time)
	expected := "100" // 1000 * 0.05 * 2
	
	if interest.String() != expected {
		t.Errorf("CalculateSimpleInterest(1000, 0.05, 2) = %s, want %s", 
			interest.String(), expected)
	}
}

func TestCompoundInterest(t *testing.T) {
	principal := MustNewDecimal("1000")
	annualRate := MustNewDecimal("0.05") // 5%
	compoundingFrequency := int64(12)    // Monthly
	timeInYears := MustNewDecimal("1")   // 1 year
	
	amount, err := CalculateCompoundInterest(principal, annualRate, compoundingFrequency, timeInYears)
	if err != nil {
		t.Errorf("CalculateCompoundInterest unexpected error: %v", err)
		return
	}
	
	// Result should be approximately 1051.16 for monthly compounding
	// We'll check if it's reasonable (between 1050 and 1052)
	min := MustNewDecimal("1050")
	max := MustNewDecimal("1052")
	
	if amount.LessThan(min) || amount.GreaterThan(max) {
		t.Errorf("CalculateCompoundInterest result %s should be between 1050 and 1052", 
			amount.String())
	}
}

func TestLoanPayment(t *testing.T) {
	principal := MustNewDecimal("100000") // $100,000
	annualRate := MustNewDecimal("5")     // 5%
	months := int64(360)                  // 30 years
	
	payment, err := CalculateLoanPayment(principal, annualRate, months)
	if err != nil {
		t.Errorf("CalculateLoanPayment unexpected error: %v", err)
		return
	}
	
	// Monthly payment should be approximately $536.82
	// We'll check if it's reasonable (between 530 and 550)
	min := MustNewDecimal("530")
	max := MustNewDecimal("550")
	
	if payment.LessThan(min) || payment.GreaterThan(max) {
		t.Errorf("CalculateLoanPayment result %s should be between 530 and 550", 
			payment.String())
	}
}

func TestLoanPaymentZeroInterest(t *testing.T) {
	principal := MustNewDecimal("12000")
	annualRate := Zero() // 0%
	months := int64(12)  // 1 year
	
	payment, err := CalculateLoanPayment(principal, annualRate, months)
	if err != nil {
		t.Errorf("CalculateLoanPayment with zero interest unexpected error: %v", err)
		return
	}
	
	// With zero interest, payment should be principal / months
	expected := MustNewDecimal("1000") // 12000 / 12
	if !payment.Equal(expected) {
		t.Errorf("CalculateLoanPayment with zero interest = %s, want %s", 
			payment.String(), expected.String())
	}
}

func TestROI(t *testing.T) {
	initialInvestment := MustNewDecimal("1000")
	currentValue := MustNewDecimal("1200")
	
	roi, err := CalculateROI(initialInvestment, currentValue)
	if err != nil {
		t.Errorf("CalculateROI unexpected error: %v", err)
		return
	}
	
	// ROI should be 20% ((1200 - 1000) / 1000 * 100)
	expected := MustNewDecimal("20")
	if !roi.Equal(expected) {
		t.Errorf("CalculateROI(1000, 1200) = %s, want %s", roi.String(), expected.String())
	}
	
	// Test negative ROI
	lowerValue := MustNewDecimal("800")
	negativeROI, err := CalculateROI(initialInvestment, lowerValue)
	if err != nil {
		t.Errorf("CalculateROI with loss unexpected error: %v", err)
		return
	}
	
	expected = MustNewDecimal("-20")
	if !negativeROI.Equal(expected) {
		t.Errorf("CalculateROI(1000, 800) = %s, want %s", 
			negativeROI.String(), expected.String())
	}
}

func TestBreakEvenPoint(t *testing.T) {
	fixedCosts := MustNewDecimal("10000")
	pricePerUnit := MustNewDecimal("50")
	variableCostPerUnit := MustNewDecimal("30")
	
	breakEven, err := CalculateBreakEvenPoint(fixedCosts, pricePerUnit, variableCostPerUnit)
	if err != nil {
		t.Errorf("CalculateBreakEvenPoint unexpected error: %v", err)
		return
	}
	
	// Break-even should be 500 units (10000 / (50 - 30))
	expected := MustNewDecimal("500")
	if !breakEven.Equal(expected) {
		t.Errorf("CalculateBreakEvenPoint = %s, want %s", 
			breakEven.String(), expected.String())
	}
}

func TestPresentValue(t *testing.T) {
	futureValue := MustNewDecimal("1000")
	discountRate := MustNewDecimal("0.05") // 5%
	periods := int64(2)                    // 2 periods
	
	pv := CalculatePresentValue(futureValue, discountRate, periods)
	
	// PV = 1000 / (1.05)^2 ≈ 907.03
	expected := MustNewDecimal("907.029478458")
	
	// Allow for small rounding differences
	diff := pv.Subtract(expected).Abs()
	tolerance := MustNewDecimal("0.01")
	
	if diff.GreaterThan(tolerance) {
		t.Errorf("CalculatePresentValue = %s, want approximately %s", 
			pv.String(), expected.String())
	}
}

func TestFutureValue(t *testing.T) {
	presentValue := MustNewDecimal("1000")
	interestRate := MustNewDecimal("0.05") // 5%
	periods := int64(2)                    // 2 periods
	
	fv := CalculateFutureValue(presentValue, interestRate, periods)
	
	// FV = 1000 * (1.05)^2 = 1102.5
	expected := MustNewDecimal("1102.5")
	
	if !fv.Equal(expected) {
		t.Errorf("CalculateFutureValue = %s, want %s", fv.String(), expected.String())
	}
}

func TestStatisticalFunctions(t *testing.T) {
	values := []Decimal{
		MustNewDecimal("10"),
		MustNewDecimal("20"),
		MustNewDecimal("30"),
		MustNewDecimal("40"),
		MustNewDecimal("50"),
	}
	
	// Test average
	avg, err := CalculateAverageDecimal(values...)
	if err != nil {
		t.Errorf("CalculateAverageDecimal unexpected error: %v", err)
	}
	expected := MustNewDecimal("30")
	if !avg.Equal(expected) {
		t.Errorf("CalculateAverageDecimal = %s, want %s", avg.String(), expected.String())
	}
	
	// Test min
	min, err := FindMinDecimal(values...)
	if err != nil {
		t.Errorf("FindMinDecimal unexpected error: %v", err)
	}
	expected = MustNewDecimal("10")
	if !min.Equal(expected) {
		t.Errorf("FindMinDecimal = %s, want %s", min.String(), expected.String())
	}
	
	// Test max
	max, err := FindMaxDecimal(values...)
	if err != nil {
		t.Errorf("FindMaxDecimal unexpected error: %v", err)
	}
	expected = MustNewDecimal("50")
	if !max.Equal(expected) {
		t.Errorf("FindMaxDecimal = %s, want %s", max.String(), expected.String())
	}
	
	// Test sum
	sum := SumDecimal(values...)
	expected = MustNewDecimal("150")
	if !sum.Equal(expected) {
		t.Errorf("SumDecimal = %s, want %s", sum.String(), expected.String())
	}
}

func TestStatisticalFunctionsEmptySlice(t *testing.T) {
	// Test average with empty slice
	_, err := CalculateAverageDecimal()
	if err == nil {
		t.Error("CalculateAverageDecimal with empty slice should return error")
	}
	
	// Test min with empty slice
	_, err = FindMinDecimal()
	if err == nil {
		t.Error("FindMinDecimal with empty slice should return error")
	}
	
	// Test max with empty slice
	_, err = FindMaxDecimal()
	if err == nil {
		t.Error("FindMaxDecimal with empty slice should return error")
	}
	
	// Test sum with empty slice (should return zero)
	sum := SumDecimal()
	if !sum.IsZero() {
		t.Errorf("SumDecimal with empty slice should return zero, got %s", sum.String())
	}
}

func TestPanicVersions(t *testing.T) {
	// Test MustCalculatePercentageOf panic
	defer func() {
		if r := recover(); r == nil {
			t.Error("MustCalculatePercentageOf should panic on division by zero")
		}
	}()
	
	part := MustNewDecimal("10")
	whole := Zero()
	MustCalculatePercentageOf(part, whole)
}

func TestFinancialEdgeCases(t *testing.T) {
	// Test break-even with zero contribution margin
	fixedCosts := MustNewDecimal("1000")
	pricePerUnit := MustNewDecimal("10")
	variableCostPerUnit := MustNewDecimal("10") // Same as price
	
	_, err := CalculateBreakEvenPoint(fixedCosts, pricePerUnit, variableCostPerUnit)
	if err == nil {
		t.Error("CalculateBreakEvenPoint with zero contribution margin should return error")
	}
	
	// Test ROI with zero investment
	zeroInvestment := Zero()
	currentValue := MustNewDecimal("100")
	
	_, err = CalculateROI(zeroInvestment, currentValue)
	if err == nil {
		t.Error("CalculateROI with zero investment should return error")
	}
}