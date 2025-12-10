// File: business.go
// Title: Business Calculation Functions
// Description: Implements common business calculations including interest,
//              loan payments, tax calculations, discounts, and financial
//              formulas commonly used in enterprise applications.
// Author: msto63 with Claude Sonnet 4.0
// Version: v0.1.0
// Created: 2025-01-24
// Modified: 2025-01-24
//
// Change History:
// - 2025-01-24 v0.1.0: Initial implementation with core business calculations

package mathx

import (
	"errors"
)

// CalculatePercentage calculates the percentage of a value
// Example: CalculatePercentage(100, 20) returns 20 (20% of 100)
func CalculatePercentage(value, percentage Decimal) Decimal {
	hundred := NewDecimalFromInt(100)
	return value.Multiply(percentage).MustDivide(hundred)
}

// CalculatePercentageOf calculates what percentage one value is of another
// Example: CalculatePercentageOf(25, 100) returns 25 (25 is 25% of 100)
func CalculatePercentageOf(part, whole Decimal) (Decimal, error) {
	if whole.IsZero() {
		return Decimal{}, errors.New("cannot calculate percentage of zero")
	}
	
	hundred := NewDecimalFromInt(100)
	return part.MustDivide(whole).Multiply(hundred), nil
}

// MustCalculatePercentageOf calculates percentage, panicking on zero denominator
func MustCalculatePercentageOf(part, whole Decimal) Decimal {
	result, err := CalculatePercentageOf(part, whole)
	if err != nil {
		panic(err)
	}
	return result
}

// CalculateDiscount calculates the discount amount
func CalculateDiscount(originalPrice, discountPercent Decimal) Decimal {
	return CalculatePercentage(originalPrice, discountPercent)
}

// ApplyDiscount applies a percentage discount to a price
func ApplyDiscount(originalPrice, discountPercent Decimal) Decimal {
	discount := CalculateDiscount(originalPrice, discountPercent)
	return originalPrice.Subtract(discount)
}

// CalculateMarkup calculates the markup amount
func CalculateMarkup(cost, markupPercent Decimal) Decimal {
	return CalculatePercentage(cost, markupPercent)
}

// ApplyMarkup applies a percentage markup to a cost
func ApplyMarkup(cost, markupPercent Decimal) Decimal {
	markup := CalculateMarkup(cost, markupPercent)
	return cost.Add(markup)
}

// CalculateTax calculates tax amount
func CalculateTax(amount, taxRate Decimal) Decimal {
	return CalculatePercentage(amount, taxRate)
}

// CalculateTaxInclusivePrice calculates the total price including tax
func CalculateTaxInclusivePrice(netAmount, taxRate Decimal) Decimal {
	tax := CalculateTax(netAmount, taxRate)
	return netAmount.Add(tax)
}

// CalculateNetFromGross calculates the net amount from gross (tax-inclusive) amount
func CalculateNetFromGross(grossAmount, taxRate Decimal) Decimal {
	hundred := NewDecimalFromInt(100)
	divisor := hundred.Add(taxRate)
	return grossAmount.Multiply(hundred).MustDivide(divisor)
}

// CalculateSimpleInterest calculates simple interest
// Formula: Interest = Principal × Rate × Time
func CalculateSimpleInterest(principal, rate, time Decimal) Decimal {
	return principal.Multiply(rate).Multiply(time)
}

// CalculateCompoundInterest calculates compound interest
// Formula: A = P(1 + r/n)^(nt)
// Where: P = principal, r = annual rate, n = compounding frequency, t = time in years
func CalculateCompoundInterest(principal, annualRate Decimal, compoundingFrequency int64, timeInYears Decimal) (Decimal, error) {
	if compoundingFrequency <= 0 {
		return Decimal{}, errors.New("compounding frequency must be positive")
	}
	
	// Convert to decimal
	n := NewDecimalFromInt(compoundingFrequency)
	
	// Calculate r/n
	ratePerPeriod, err := annualRate.Divide(n)
	if err != nil {
		return Decimal{}, err
	}
	
	// Calculate 1 + r/n
	onePlusRate := One().Add(ratePerPeriod)
	
	// Calculate nt
	totalPeriods := n.Multiply(timeInYears)
	
	// For simplicity, we'll use an approximation for fractional exponents
	// In a production system, you might want to use a more sophisticated power function
	exponent, err := totalPeriods.Int64()
	if err != nil {
		return Decimal{}, errors.New("exponent too large for calculation")
	}
	
	// Calculate (1 + r/n)^(nt)
	compoundFactor := onePlusRate.Pow(exponent)
	
	// Calculate final amount
	return principal.Multiply(compoundFactor), nil
}

// CalculateLoanPayment calculates monthly loan payment using the standard amortization formula
// Formula: M = P * [r(1+r)^n] / [(1+r)^n - 1]
// Where: P = principal, r = monthly interest rate, n = number of payments
func CalculateLoanPayment(principal, annualRate Decimal, months int64) (Decimal, error) {
	if months <= 0 {
		return Decimal{}, errors.New("number of months must be positive")
	}
	
	// Handle zero interest rate case
	if annualRate.IsZero() {
		monthsDecimal := NewDecimalFromInt(months)
		return principal.MustDivide(monthsDecimal), nil
	}
	
	// Convert annual rate to monthly rate
	twelve := NewDecimalFromInt(12)
	hundred := NewDecimalFromInt(100)
	monthlyRate := annualRate.MustDivide(hundred).MustDivide(twelve)
	
	// Calculate (1 + r)
	onePlusRate := One().Add(monthlyRate)
	
	// Calculate (1 + r)^n
	compound := onePlusRate.Pow(months)
	
	// Calculate numerator: r * (1 + r)^n
	numerator := monthlyRate.Multiply(compound)
	
	// Calculate denominator: (1 + r)^n - 1
	denominator := compound.Subtract(One())
	
	if denominator.IsZero() {
		return Decimal{}, errors.New("invalid calculation: denominator is zero")
	}
	
	// Calculate payment factor
	paymentFactor := numerator.MustDivide(denominator)
	
	// Calculate monthly payment
	return principal.Multiply(paymentFactor), nil
}

// CalculateROI calculates Return on Investment as a percentage
// Formula: ROI = (Gain - Cost) / Cost * 100
func CalculateROI(initialInvestment, currentValue Decimal) (Decimal, error) {
	if initialInvestment.IsZero() {
		return Decimal{}, errors.New("initial investment cannot be zero")
	}
	
	gain := currentValue.Subtract(initialInvestment)
	return MustCalculatePercentageOf(gain, initialInvestment), nil
}

// MustCalculateROI calculates ROI, panicking on zero investment
func MustCalculateROI(initialInvestment, currentValue Decimal) Decimal {
	result, err := CalculateROI(initialInvestment, currentValue)
	if err != nil {
		panic(err)
	}
	return result
}

// CalculateBreakEvenPoint calculates break-even point in units
// Formula: Break-even = Fixed Costs / (Price per Unit - Variable Cost per Unit)
func CalculateBreakEvenPoint(fixedCosts, pricePerUnit, variableCostPerUnit Decimal) (Decimal, error) {
	contributionMargin := pricePerUnit.Subtract(variableCostPerUnit)
	
	if contributionMargin.IsZero() {
		return Decimal{}, errors.New("contribution margin cannot be zero")
	}
	
	return fixedCosts.MustDivide(contributionMargin), nil
}

// CalculatePresentValue calculates present value of future cash flow
// Formula: PV = FV / (1 + r)^n
func CalculatePresentValue(futureValue, discountRate Decimal, periods int64) Decimal {
	onePlusRate := One().Add(discountRate)
	discountFactor := onePlusRate.Pow(periods)
	return futureValue.MustDivide(discountFactor)
}

// CalculateFutureValue calculates future value of present investment
// Formula: FV = PV * (1 + r)^n
func CalculateFutureValue(presentValue, interestRate Decimal, periods int64) Decimal {
	onePlusRate := One().Add(interestRate)
	growthFactor := onePlusRate.Pow(periods)
	return presentValue.Multiply(growthFactor)
}

// CalculateAverageDecimal calculates the average of decimal values
func CalculateAverageDecimal(values ...Decimal) (Decimal, error) {
	if len(values) == 0 {
		return Decimal{}, errors.New("cannot calculate average of empty slice")
	}
	
	sum := Zero()
	for _, value := range values {
		sum = sum.Add(value)
	}
	
	count := NewDecimalFromInt(int64(len(values)))
	return sum.MustDivide(count), nil
}

// MustCalculateAverageDecimal calculates average, panicking on empty slice
func MustCalculateAverageDecimal(values ...Decimal) Decimal {
	result, err := CalculateAverageDecimal(values...)
	if err != nil {
		panic(err)
	}
	return result
}

// FindMinDecimal finds the minimum value in a slice of decimals
func FindMinDecimal(values ...Decimal) (Decimal, error) {
	if len(values) == 0 {
		return Decimal{}, errors.New("cannot find minimum of empty slice")
	}
	
	min := values[0]
	for _, value := range values[1:] {
		if value.LessThan(min) {
			min = value
		}
	}
	
	return min, nil
}

// FindMaxDecimal finds the maximum value in a slice of decimals
func FindMaxDecimal(values ...Decimal) (Decimal, error) {
	if len(values) == 0 {
		return Decimal{}, errors.New("cannot find maximum of empty slice")
	}
	
	max := values[0]
	for _, value := range values[1:] {
		if value.GreaterThan(max) {
			max = value
		}
	}
	
	return max, nil
}

// SumDecimal calculates the sum of decimal values
func SumDecimal(values ...Decimal) Decimal {
	sum := Zero()
	for _, value := range values {
		sum = sum.Add(value)
	}
	return sum
}