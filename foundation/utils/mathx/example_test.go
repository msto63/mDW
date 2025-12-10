// File: example_test.go
// Title: Example Tests for MathX Package Documentation
// Description: Executable examples that serve as both documentation and tests.
//              These examples demonstrate typical usage patterns for business
//              calculations and appear in the generated documentation.
// Author: msto63 with Claude Sonnet 4.0
// Version: v0.1.0
// Created: 2025-01-24
// Modified: 2025-01-24
//
// Change History:
// - 2025-01-24 v0.1.0: Initial example implementation

package mathx_test

import (
	"fmt"
	mdwmathx "github.com/msto63/mDW/foundation/utils/mathx"
)

func ExampleNewDecimal() {
	d1, _ := mdwmathx.NewDecimal("123.45")
	d2, _ := mdwmathx.NewDecimal("-67.89")
	
	fmt.Println(d1.String())
	fmt.Println(d2.String())
	// Output:
	// 123.45
	// -67.89
}

func ExampleDecimal_Add() {
	d1 := mdwmathx.MustNewDecimal("123.45")
	d2 := mdwmathx.MustNewDecimal("67.89")
	
	result := d1.Add(d2)
	fmt.Println(result.String())
	// Output:
	// 191.34
}

func ExampleDecimal_Multiply() {
	price := mdwmathx.MustNewDecimal("19.99")
	quantity := mdwmathx.MustNewDecimal("3")
	
	total := price.Multiply(quantity)
	fmt.Println(total.StringFixed(2))
	// Output:
	// 59.97
}

func ExampleDecimal_Round() {
	d := mdwmathx.MustNewDecimal("123.456789")
	
	rounded := d.Round(2, mdwmathx.RoundingModeHalfUp)
	fmt.Println(rounded.StringFixed(2))
	// Output:
	// 123.46
}

func ExampleNewMoney() {
	amount := mdwmathx.MustNewDecimal("19.99")
	money := mdwmathx.NewMoney(amount, mdwmathx.USD)
	
	fmt.Println(money.Format())
	fmt.Println(money.FormatWithCode())
	// Output:
	// $19.99
	// 19.99 USD
}

func ExampleNewMoneyFromString() {
	money, _ := mdwmathx.NewMoneyFromString("25.50", "EUR")
	
	fmt.Println(money.Format())
	fmt.Println(money.FormatLong())
	// Output:
	// €25.50
	// 25.50 Euro
}

func ExampleMoney_Add() {
	price1 := mdwmathx.MustNewMoneyFromString("15.99", "USD")
	price2 := mdwmathx.MustNewMoneyFromString("8.50", "USD")
	
	total, _ := price1.Add(price2)
	fmt.Println(total.Format())
	// Output:
	// $24.49
}

func ExampleMoney_Allocate() {
	bill := mdwmathx.MustNewMoneyFromString("100.00", "USD")
	
	// Split bill: 50% to person A, 30% to person B, 20% to person C
	ratios := []mdwmathx.Decimal{
		mdwmathx.MustNewDecimal("50"),
		mdwmathx.MustNewDecimal("30"),
		mdwmathx.MustNewDecimal("20"),
	}
	
	splits := bill.Allocate(ratios...)
	for i, split := range splits {
		fmt.Printf("Person %c: %s\n", 'A'+i, split.Format())
	}
	// Output:
	// Person A: $50.00
	// Person B: $30.00
	// Person C: $20.00
}

func ExampleCalculatePercentage() {
	price := mdwmathx.MustNewDecimal("100.00")
	discountPercent := mdwmathx.MustNewDecimal("15")
	
	discount := mdwmathx.CalculatePercentage(price, discountPercent)
	fmt.Printf("Discount: $%s\n", discount.StringFixed(2))
	
	finalPrice := mdwmathx.ApplyDiscount(price, discountPercent)
	fmt.Printf("Final price: $%s\n", finalPrice.StringFixed(2))
	// Output:
	// Discount: $15.00
	// Final price: $85.00
}

func ExampleCalculateTax() {
	netAmount := mdwmathx.MustNewDecimal("100.00")
	taxRate := mdwmathx.MustNewDecimal("19") // 19% VAT
	
	tax := mdwmathx.CalculateTax(netAmount, taxRate)
	grossAmount := mdwmathx.CalculateTaxInclusivePrice(netAmount, taxRate)
	
	fmt.Printf("Net: $%s\n", netAmount.StringFixed(2))
	fmt.Printf("Tax: $%s\n", tax.StringFixed(2))
	fmt.Printf("Gross: $%s\n", grossAmount.StringFixed(2))
	// Output:
	// Net: $100.00
	// Tax: $19.00
	// Gross: $119.00
}

func ExampleCalculateSimpleInterest() {
	principal := mdwmathx.MustNewDecimal("1000")
	rate := mdwmathx.MustNewDecimal("0.05") // 5% annual rate
	time := mdwmathx.MustNewDecimal("2")    // 2 years
	
	interest := mdwmathx.CalculateSimpleInterest(principal, rate, time)
	fmt.Printf("Simple interest: $%s\n", interest.StringFixed(2))
	// Output:
	// Simple interest: $100.00
}

func ExampleCalculateLoanPayment() {
	principal := mdwmathx.MustNewDecimal("200000") // $200,000 loan
	annualRate := mdwmathx.MustNewDecimal("4.5")   // 4.5% annual rate
	months := int64(360)                        // 30 years
	
	payment, _ := mdwmathx.CalculateLoanPayment(principal, annualRate, months)
	fmt.Printf("Monthly payment: $%s\n", payment.StringFixed(2))
	// Output:
	// Monthly payment: $1013.37
}

func ExampleCalculateROI() {
	initialInvestment := mdwmathx.MustNewDecimal("10000")
	currentValue := mdwmathx.MustNewDecimal("12500")
	
	roi, _ := mdwmathx.CalculateROI(initialInvestment, currentValue)
	fmt.Printf("ROI: %s%%\n", roi.StringFixed(1))
	// Output:
	// ROI: 25.0%
}

func ExampleCalculateBreakEvenPoint() {
	fixedCosts := mdwmathx.MustNewDecimal("50000")
	pricePerUnit := mdwmathx.MustNewDecimal("25")
	variableCostPerUnit := mdwmathx.MustNewDecimal("15")
	
	breakEven, _ := mdwmathx.CalculateBreakEvenPoint(fixedCosts, pricePerUnit, variableCostPerUnit)
	fmt.Printf("Break-even point: %s units\n", breakEven.StringFixed(0))
	// Output:
	// Break-even point: 5000 units
}

func ExampleCalculateAverageDecimal() {
	salesFigures := []mdwmathx.Decimal{
		mdwmathx.MustNewDecimal("1250.00"),
		mdwmathx.MustNewDecimal("1890.50"),
		mdwmathx.MustNewDecimal("2100.25"),
		mdwmathx.MustNewDecimal("1675.75"),
		mdwmathx.MustNewDecimal("1980.00"),
	}
	
	average, _ := mdwmathx.CalculateAverageDecimal(salesFigures...)
	fmt.Printf("Average sales: $%s\n", average.StringFixed(2))
	// Output:
	// Average sales: $1779.30
}

func ExampleFindMinDecimal() {
	prices := []mdwmathx.Decimal{
		mdwmathx.MustNewDecimal("19.99"),
		mdwmathx.MustNewDecimal("15.50"),
		mdwmathx.MustNewDecimal("22.75"),
		mdwmathx.MustNewDecimal("18.25"),
	}
	
	lowest, _ := mdwmathx.FindMinDecimal(prices...)
	fmt.Printf("Lowest price: $%s\n", lowest.StringFixed(2))
	// Output:
	// Lowest price: $15.50
}

func ExampleSumDecimal() {
	expenses := []mdwmathx.Decimal{
		mdwmathx.MustNewDecimal("1250.00"),
		mdwmathx.MustNewDecimal("890.50"),
		mdwmathx.MustNewDecimal("2100.25"),
		mdwmathx.MustNewDecimal("675.75"),
	}
	
	total := mdwmathx.SumDecimal(expenses...)
	fmt.Printf("Total expenses: $%s\n", total.StringFixed(2))
	// Output:
	// Total expenses: $4916.50
}

func ExampleDecimal_Sqrt() {
	area := mdwmathx.MustNewDecimal("144")
	
	side, _ := area.Sqrt()
	fmt.Printf("Side length: %s\n", side.StringFixed(0))
	// Output:
	// Side length: 12
}

func ExampleFormatCurrency() {
	amount := mdwmathx.MustNewDecimal("1234.56")
	
	fmt.Println(mdwmathx.FormatCurrency(amount, "USD", 2))
	fmt.Println(mdwmathx.FormatCurrency(amount, "EUR", 2))
	fmt.Println(mdwmathx.FormatCurrency(amount, "JPY", 0))
	// Output:
	// $1234.56
	// €1234.56
	// ¥1235
}

// Example of a complete financial scenario
func Example_completeFinancialScenario() {
	// Investment scenario
	fmt.Println("=== Investment Analysis ===")
	
	initialInvestment := mdwmathx.MustNewDecimal("50000")
	currentValue := mdwmathx.MustNewDecimal("62500")
	
	roi, _ := mdwmathx.CalculateROI(initialInvestment, currentValue)
	fmt.Printf("Initial Investment: $%s\n", initialInvestment.StringFixed(2))
	fmt.Printf("Current Value: $%s\n", currentValue.StringFixed(2))
	fmt.Printf("ROI: %s%%\n", roi.StringFixed(1))
	
	// Loan scenario
	fmt.Println("\n=== Loan Analysis ===")
	
	loanAmount := mdwmathx.MustNewDecimal("300000")
	interestRate := mdwmathx.MustNewDecimal("3.75")
	loanTermMonths := int64(360)
	
	monthlyPayment, _ := mdwmathx.CalculateLoanPayment(loanAmount, interestRate, loanTermMonths)
	fmt.Printf("Loan Amount: $%s\n", loanAmount.StringFixed(2))
	fmt.Printf("Interest Rate: %s%%\n", interestRate.StringFixed(2))
	fmt.Printf("Monthly Payment: $%s\n", monthlyPayment.StringFixed(2))
	
	// Business scenario
	fmt.Println("\n=== Business Analysis ===")
	
	revenue := mdwmathx.MustNewDecimal("125000")
	taxRate := mdwmathx.MustNewDecimal("21")
	
	tax := mdwmathx.CalculateTax(revenue, taxRate)
	netRevenue := revenue.Subtract(tax)
	
	fmt.Printf("Gross Revenue: $%s\n", revenue.StringFixed(2))
	fmt.Printf("Tax (%s%%): $%s\n", taxRate.StringFixed(0), tax.StringFixed(2))
	fmt.Printf("Net Revenue: $%s\n", netRevenue.StringFixed(2))
	
	// Output:
	// === Investment Analysis ===
	// Initial Investment: $50000.00
	// Current Value: $62500.00
	// ROI: 25.0%
	//
	// === Loan Analysis ===
	// Loan Amount: $300000.00
	// Interest Rate: 3.75%
	// Monthly Payment: $1389.35
	//
	// === Business Analysis ===
	// Gross Revenue: $125000.00
	// Tax (21%): $26250.00
	// Net Revenue: $98750.00
}