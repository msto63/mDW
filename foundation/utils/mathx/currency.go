// File: currency.go
// Title: Currency Operations and Formatting
// Description: Implements currency-specific operations including formatting,
//              conversion, and currency-aware arithmetic with proper rounding
//              rules for different currencies and locales.
// Author: msto63 with Claude Sonnet 4.0
// Version: v0.1.0
// Created: 2025-01-24
// Modified: 2025-01-24
//
// Change History:
// - 2025-01-24 v0.1.0: Initial implementation with currency formatting and operations

package mathx

import (
	"fmt"
	"strings"
)

// Currency represents a currency with its properties
type Currency struct {
	Code         string // ISO 4217 code (e.g., "USD", "EUR")
	Symbol       string // Currency symbol (e.g., "$", "€")
	DecimalPlaces int    // Number of decimal places
	Name         string // Full name (e.g., "US Dollar")
}

// Common currencies
var (
	USD = Currency{Code: "USD", Symbol: "$", DecimalPlaces: 2, Name: "US Dollar"}
	EUR = Currency{Code: "EUR", Symbol: "€", DecimalPlaces: 2, Name: "Euro"}
	GBP = Currency{Code: "GBP", Symbol: "£", DecimalPlaces: 2, Name: "British Pound"}
	JPY = Currency{Code: "JPY", Symbol: "¥", DecimalPlaces: 0, Name: "Japanese Yen"}
	CHF = Currency{Code: "CHF", Symbol: "CHF", DecimalPlaces: 2, Name: "Swiss Franc"}
	CAD = Currency{Code: "CAD", Symbol: "C$", DecimalPlaces: 2, Name: "Canadian Dollar"}
	AUD = Currency{Code: "AUD", Symbol: "A$", DecimalPlaces: 2, Name: "Australian Dollar"}
	CNY = Currency{Code: "CNY", Symbol: "¥", DecimalPlaces: 2, Name: "Chinese Yuan"}
	INR = Currency{Code: "INR", Symbol: "₹", DecimalPlaces: 2, Name: "Indian Rupee"}
	BTC = Currency{Code: "BTC", Symbol: "₿", DecimalPlaces: 8, Name: "Bitcoin"}
)

// CurrencyRegistry holds all known currencies
var CurrencyRegistry = map[string]Currency{
	"USD": USD,
	"EUR": EUR,
	"GBP": GBP,
	"JPY": JPY,
	"CHF": CHF,
	"CAD": CAD,
	"AUD": AUD,
	"CNY": CNY,
	"INR": INR,
	"BTC": BTC,
}

// Money represents a monetary amount with currency
type Money struct {
	Amount   Decimal
	Currency Currency
}

// NewMoney creates a new Money instance
func NewMoney(amount Decimal, currency Currency) Money {
	// Round to currency's decimal places
	rounded := amount.Round(currency.DecimalPlaces, RoundingModeHalfUp)
	return Money{
		Amount:   rounded,
		Currency: currency,
	}
}

// NewMoneyFromString creates Money from string amount and currency code
func NewMoneyFromString(amount, currencyCode string) (Money, error) {
	dec, err := NewDecimal(amount)
	if err != nil {
		return Money{}, err
	}
	
	currency, exists := CurrencyRegistry[strings.ToUpper(currencyCode)]
	if !exists {
		return Money{}, fmt.Errorf("unknown currency code: %s", currencyCode)
	}
	
	return NewMoney(dec, currency), nil
}

// MustNewMoneyFromString creates Money from string, panicking on error
func MustNewMoneyFromString(amount, currencyCode string) Money {
	m, err := NewMoneyFromString(amount, currencyCode)
	if err != nil {
		panic(err)
	}
	return m
}

// Add adds another Money amount (must be same currency)
func (m Money) Add(other Money) (Money, error) {
	if m.Currency.Code != other.Currency.Code {
		return Money{}, fmt.Errorf("cannot add different currencies: %s and %s", 
			m.Currency.Code, other.Currency.Code)
	}
	
	result := m.Amount.Add(other.Amount)
	return NewMoney(result, m.Currency), nil
}

// MustAdd adds another Money amount, panicking on currency mismatch
func (m Money) MustAdd(other Money) Money {
	result, err := m.Add(other)
	if err != nil {
		panic(err)
	}
	return result
}

// Subtract subtracts another Money amount (must be same currency)
func (m Money) Subtract(other Money) (Money, error) {
	if m.Currency.Code != other.Currency.Code {
		return Money{}, fmt.Errorf("cannot subtract different currencies: %s and %s", 
			m.Currency.Code, other.Currency.Code)
	}
	
	result := m.Amount.Subtract(other.Amount)
	return NewMoney(result, m.Currency), nil
}

// MustSubtract subtracts another Money amount, panicking on currency mismatch
func (m Money) MustSubtract(other Money) Money {
	result, err := m.Subtract(other)
	if err != nil {
		panic(err)
	}
	return result
}

// Multiply multiplies the amount by a decimal factor
func (m Money) Multiply(factor Decimal) Money {
	result := m.Amount.Multiply(factor)
	return NewMoney(result, m.Currency)
}

// Divide divides the amount by a decimal divisor
func (m Money) Divide(divisor Decimal) (Money, error) {
	result, err := m.Amount.Divide(divisor)
	if err != nil {
		return Money{}, err
	}
	return NewMoney(result, m.Currency), nil
}

// MustDivide divides the amount by a decimal divisor, panicking on error
func (m Money) MustDivide(divisor Decimal) Money {
	result, err := m.Divide(divisor)
	if err != nil {
		panic(err)
	}
	return result
}

// Allocate divides the money into parts according to the given ratios
// This is useful for splitting bills, calculating commissions, etc.
// The ratios don't need to sum to 1.0 - they're normalized automatically
func (m Money) Allocate(ratios ...Decimal) []Money {
	if len(ratios) == 0 {
		return []Money{}
	}
	
	// Calculate total ratio
	totalRatio := Zero()
	for _, ratio := range ratios {
		totalRatio = totalRatio.Add(ratio)
	}
	
	if totalRatio.IsZero() {
		// If all ratios are zero, return zero amounts
		result := make([]Money, len(ratios))
		zeroMoney := NewMoney(Zero(), m.Currency)
		for i := range result {
			result[i] = zeroMoney
		}
		return result
	}
	
	// Calculate allocated amounts
	result := make([]Money, len(ratios))
	remainder := m.Amount
	
	for i, ratio := range ratios {
		if i == len(ratios)-1 {
			// Last allocation gets the remainder to avoid rounding errors
			result[i] = NewMoney(remainder, m.Currency)
		} else {
			// Calculate proportional amount
			proportion := ratio.MustDivide(totalRatio)
			allocated := m.Amount.Multiply(proportion)
			result[i] = NewMoney(allocated, m.Currency)
			remainder = remainder.Subtract(result[i].Amount)
		}
	}
	
	return result
}

// IsZero returns true if the amount is zero
func (m Money) IsZero() bool {
	return m.Amount.IsZero()
}

// IsPositive returns true if the amount is positive
func (m Money) IsPositive() bool {
	return m.Amount.IsPositive()
}

// IsNegative returns true if the amount is negative
func (m Money) IsNegative() bool {
	return m.Amount.IsNegative()
}

// Compare compares this Money with another (must be same currency)
func (m Money) Compare(other Money) (int, error) {
	if m.Currency.Code != other.Currency.Code {
		return 0, fmt.Errorf("cannot compare different currencies: %s and %s", 
			m.Currency.Code, other.Currency.Code)
	}
	
	return m.Amount.Compare(other.Amount), nil
}

// Equal returns true if both Money instances have the same amount and currency
func (m Money) Equal(other Money) bool {
	if m.Currency.Code != other.Currency.Code {
		return false
	}
	return m.Amount.Equal(other.Amount)
}

// String returns a formatted string representation of the money
func (m Money) String() string {
	return m.Format()
}

// Format formats the money according to the currency's conventions
func (m Money) Format() string {
	amount := m.Amount.StringFixed(m.Currency.DecimalPlaces)
	return fmt.Sprintf("%s%s", m.Currency.Symbol, amount)
}

// FormatWithCode formats the money with the currency code
func (m Money) FormatWithCode() string {
	amount := m.Amount.StringFixed(m.Currency.DecimalPlaces)
	return fmt.Sprintf("%s %s", amount, m.Currency.Code)
}

// FormatLong formats the money with full currency name
func (m Money) FormatLong() string {
	amount := m.Amount.StringFixed(m.Currency.DecimalPlaces)
	return fmt.Sprintf("%s %s", amount, m.Currency.Name)
}

// RegisterCurrency adds a new currency to the registry
func RegisterCurrency(currency Currency) {
	CurrencyRegistry[strings.ToUpper(currency.Code)] = currency
}

// GetCurrency retrieves a currency by code
func GetCurrency(code string) (Currency, bool) {
	currency, exists := CurrencyRegistry[strings.ToUpper(code)]
	return currency, exists
}

// FormatCurrency formats a decimal amount as currency
func FormatCurrency(amount Decimal, currencyCode string, places int) string {
	currency, exists := CurrencyRegistry[strings.ToUpper(currencyCode)]
	if !exists {
		// Fallback formatting
		return fmt.Sprintf("%s %s", amount.StringFixed(places), currencyCode)
	}
	
	money := NewMoney(amount, currency)
	return money.Format()
}