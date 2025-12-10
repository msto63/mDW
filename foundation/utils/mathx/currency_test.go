// File: currency_test.go
// Title: Unit Tests for Currency Operations
// Description: Comprehensive unit tests for currency handling, Money type,
//              and currency-specific operations including formatting,
//              arithmetic, and allocation functions.
// Author: msto63 with Claude Sonnet 4.0
// Version: v0.1.0
// Created: 2025-01-24
// Modified: 2025-01-24
//
// Change History:
// - 2025-01-24 v0.1.0: Initial test implementation for currency operations

package mathx

import (
	"testing"
)

func TestNewMoney(t *testing.T) {
	amount := MustNewDecimal("19.999")
	money := NewMoney(amount, USD)
	
	// Should round to 2 decimal places for USD
	expected := "20.00"
	if money.Amount.StringFixed(2) != expected {
		t.Errorf("NewMoney should round to currency precision, got %s, want %s", 
			money.Amount.StringFixed(2), expected)
	}
	
	if money.Currency.Code != "USD" {
		t.Errorf("Currency should be USD, got %s", money.Currency.Code)
	}
}

func TestNewMoneyFromString(t *testing.T) {
	tests := []struct {
		name         string
		amount       string
		currencyCode string
		wantErr      bool
		wantAmount   string
		wantCurrency string
	}{
		{"valid USD", "19.99", "USD", false, "19.99", "USD"},
		{"valid EUR", "25.50", "EUR", false, "25.50", "EUR"},
		{"lowercase code", "10.00", "usd", false, "10.00", "USD"},
		{"invalid amount", "abc", "USD", true, "", ""},
		{"invalid currency", "10.00", "XXX", true, "", ""},
		{"JPY no decimals", "1000", "JPY", false, "1000", "JPY"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			money, err := NewMoneyFromString(tt.amount, tt.currencyCode)
			
			if tt.wantErr {
				if err == nil {
					t.Errorf("NewMoneyFromString(%s, %s) expected error", tt.amount, tt.currencyCode)
				}
				return
			}
			
			if err != nil {
				t.Errorf("NewMoneyFromString(%s, %s) unexpected error: %v", 
					tt.amount, tt.currencyCode, err)
				return
			}
			
			if money.Currency.Code != tt.wantCurrency {
				t.Errorf("Currency code = %s, want %s", money.Currency.Code, tt.wantCurrency)
			}
			
			actualAmount := money.Amount.StringFixed(money.Currency.DecimalPlaces)
			if actualAmount != tt.wantAmount {
				t.Errorf("Amount = %s, want %s", actualAmount, tt.wantAmount)
			}
		})
	}
}

func TestMoneyArithmetic(t *testing.T) {
	money1 := MustNewMoneyFromString("10.50", "USD")
	money2 := MustNewMoneyFromString("5.25", "USD")
	
	// Test addition
	sum, err := money1.Add(money2)
	if err != nil {
		t.Errorf("Add unexpected error: %v", err)
	}
	expected := "15.75"
	if sum.Amount.StringFixed(2) != expected {
		t.Errorf("Add: got %s, want %s", sum.Amount.StringFixed(2), expected)
	}
	
	// Test subtraction
	diff, err := money1.Subtract(money2)
	if err != nil {
		t.Errorf("Subtract unexpected error: %v", err)
	}
	expected = "5.25"
	if diff.Amount.StringFixed(2) != expected {
		t.Errorf("Subtract: got %s, want %s", diff.Amount.StringFixed(2), expected)
	}
	
	// Test multiplication
	factor := MustNewDecimal("2")
	product := money1.Multiply(factor)
	expected = "21.00"
	if product.Amount.StringFixed(2) != expected {
		t.Errorf("Multiply: got %s, want %s", product.Amount.StringFixed(2), expected)
	}
	
	// Test division
	divisor := MustNewDecimal("2")
	quotient, err := money1.Divide(divisor)
	if err != nil {
		t.Errorf("Divide unexpected error: %v", err)
	}
	expected = "5.25"
	if quotient.Amount.StringFixed(2) != expected {
		t.Errorf("Divide: got %s, want %s", quotient.Amount.StringFixed(2), expected)
	}
}

func TestMoneyDifferentCurrencies(t *testing.T) {
	usd := MustNewMoneyFromString("10.00", "USD")
	eur := MustNewMoneyFromString("8.50", "EUR")
	
	// Test addition with different currencies
	_, err := usd.Add(eur)
	if err == nil {
		t.Error("Adding different currencies should return error")
	}
	
	// Test subtraction with different currencies
	_, err = usd.Subtract(eur)
	if err == nil {
		t.Error("Subtracting different currencies should return error")
	}
	
	// Test comparison with different currencies
	_, err = usd.Compare(eur)
	if err == nil {
		t.Error("Comparing different currencies should return error")
	}
	
	// Test equality with different currencies
	if usd.Equal(eur) {
		t.Error("Different currencies should not be equal")
	}
}

func TestMoneyAllocate(t *testing.T) {
	money := MustNewMoneyFromString("100.00", "USD")
	
	// Test equal allocation
	ratios := []Decimal{
		MustNewDecimal("1"),
		MustNewDecimal("1"),
		MustNewDecimal("1"),
	}
	
	allocated := money.Allocate(ratios...)
	
	if len(allocated) != 3 {
		t.Errorf("Expected 3 allocations, got %d", len(allocated))
	}
	
	// Sum should equal original amount (accounting for rounding)
	sum := Zero()
	for _, alloc := range allocated {
		sum = sum.Add(alloc.Amount)
	}
	
	if !sum.Equal(money.Amount) {
		t.Errorf("Allocated sum %s should equal original amount %s", 
			sum.StringFixed(2), money.Amount.StringFixed(2))
	}
	
	// Test proportional allocation
	proportions := []Decimal{
		MustNewDecimal("50"), // 50%
		MustNewDecimal("30"), // 30%
		MustNewDecimal("20"), // 20%
	}
	
	proportionalAlloc := money.Allocate(proportions...)
	
	// Check approximate allocations (allowing for rounding)
	expected := []string{"50.00", "30.00", "20.00"}
	for i, alloc := range proportionalAlloc {
		actual := alloc.Amount.StringFixed(2)
		if actual != expected[i] {
			t.Errorf("Allocation %d: got %s, want %s", i, actual, expected[i])
		}
	}
}

func TestMoneyAllocateEdgeCases(t *testing.T) {
	money := MustNewMoneyFromString("10.00", "USD")
	
	// Test empty ratios
	allocated := money.Allocate()
	if len(allocated) != 0 {
		t.Errorf("Empty ratios should return empty slice, got %d items", len(allocated))
	}
	
	// Test zero ratios
	zeroRatios := []Decimal{Zero(), Zero()}
	allocated = money.Allocate(zeroRatios...)
	
	for i, alloc := range allocated {
		if !alloc.Amount.IsZero() {
			t.Errorf("Zero ratio allocation %d should be zero, got %s", i, alloc.Amount.String())
		}
	}
}

func TestMoneyFormatting(t *testing.T) {
	tests := []struct {
		amount       string
		currency     string
		wantFormat   string
		wantWithCode string
		wantLong     string
	}{
		{"19.99", "USD", "$19.99", "19.99 USD", "19.99 US Dollar"},
		{"25.50", "EUR", "€25.50", "25.50 EUR", "25.50 Euro"},
		{"1000", "JPY", "¥1000", "1000 JPY", "1000 Japanese Yen"},
		{"15.75", "GBP", "£15.75", "15.75 GBP", "15.75 British Pound"},
	}

	for _, tt := range tests {
		t.Run(tt.currency, func(t *testing.T) {
			money := MustNewMoneyFromString(tt.amount, tt.currency)
			
			if money.Format() != tt.wantFormat {
				t.Errorf("Format() = %s, want %s", money.Format(), tt.wantFormat)
			}
			
			if money.FormatWithCode() != tt.wantWithCode {
				t.Errorf("FormatWithCode() = %s, want %s", 
					money.FormatWithCode(), tt.wantWithCode)
			}
			
			if money.FormatLong() != tt.wantLong {
				t.Errorf("FormatLong() = %s, want %s", money.FormatLong(), tt.wantLong)
			}
			
			// Test String() method
			if money.String() != tt.wantFormat {
				t.Errorf("String() = %s, want %s", money.String(), tt.wantFormat)
			}
		})
	}
}

func TestMoneyProperties(t *testing.T) {
	positive := MustNewMoneyFromString("10.50", "USD")
	negative := MustNewMoneyFromString("-5.25", "USD")
	zero := MustNewMoneyFromString("0", "USD")
	
	// Test IsZero
	if !zero.IsZero() {
		t.Error("Zero money should be zero")
	}
	if positive.IsZero() {
		t.Error("Positive money should not be zero")
	}
	
	// Test IsPositive
	if !positive.IsPositive() {
		t.Error("Positive money should be positive")
	}
	if negative.IsPositive() {
		t.Error("Negative money should not be positive")
	}
	if zero.IsPositive() {
		t.Error("Zero money should not be positive")
	}
	
	// Test IsNegative
	if !negative.IsNegative() {
		t.Error("Negative money should be negative")
	}
	if positive.IsNegative() {
		t.Error("Positive money should not be negative")
	}
	if zero.IsNegative() {
		t.Error("Zero money should not be negative")
	}
}

func TestCurrencyRegistry(t *testing.T) {
	// Test getting existing currency
	currency, exists := GetCurrency("USD")
	if !exists {
		t.Error("USD should exist in currency registry")
	}
	if currency.Code != "USD" {
		t.Errorf("Currency code should be USD, got %s", currency.Code)
	}
	
	// Test case insensitive lookup
	currency, exists = GetCurrency("usd")
	if !exists {
		t.Error("Lowercase 'usd' should find USD in registry")
	}
	
	// Test non-existent currency
	_, exists = GetCurrency("XXX")
	if exists {
		t.Error("XXX should not exist in currency registry")
	}
	
	// Test registering new currency
	testCurrency := Currency{
		Code:          "TEST",
		Symbol:        "T$",
		DecimalPlaces: 3,
		Name:          "Test Currency",
	}
	
	RegisterCurrency(testCurrency)
	
	retrieved, exists := GetCurrency("TEST")
	if !exists {
		t.Error("TEST currency should exist after registration")
	}
	if retrieved.DecimalPlaces != 3 {
		t.Errorf("TEST currency should have 3 decimal places, got %d", retrieved.DecimalPlaces)
	}
}

func TestFormatCurrency(t *testing.T) {
	tests := []struct {
		amount   string
		currency string
		places   int
		want     string
	}{
		{"19.99", "USD", 2, "$19.99"},
		{"25.5", "EUR", 2, "€25.50"},
		{"1000", "JPY", 0, "¥1000"},
		{"15.755", "UNKNOWN", 2, "15.76 UNKNOWN"},
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			amount := MustNewDecimal(tt.amount)
			result := FormatCurrency(amount, tt.currency, tt.places)
			if result != tt.want {
				t.Errorf("FormatCurrency(%s, %s, %d) = %s, want %s", 
					tt.amount, tt.currency, tt.places, result, tt.want)
			}
		})
	}
}