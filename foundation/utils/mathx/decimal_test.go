// File: decimal_test.go
// Title: Unit Tests for Decimal Arithmetic
// Description: Comprehensive unit tests for the Decimal type and its operations.
//              Tests cover precision, rounding modes, edge cases, and mathematical
//              properties to ensure reliability in financial calculations.
// Author: msto63 with Claude Sonnet 4.0
// Version: v0.1.0
// Created: 2025-01-24
// Modified: 2025-01-24
//
// Change History:
// - 2025-01-24 v0.1.0: Initial test implementation for decimal arithmetic

package mathx

import (
	"testing"
)

func TestNewDecimal(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
		want    string
	}{
		{"positive integer", "123", false, "123"},
		{"negative integer", "-456", false, "-456"},
		{"positive decimal", "123.45", false, "123.45"},
		{"negative decimal", "-67.89", false, "-67.89"},
		{"zero", "0", false, "0"},
		{"zero decimal", "0.00", false, "0"},
		{"leading zeros", "000123.450", false, "123.45"},
		{"fraction", "1/2", false, "1/2"},
		{"invalid format", "abc", true, ""},
		{"empty string", "", true, ""},
		{"multiple decimals", "12.34.56", true, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := NewDecimal(tt.input)
			
			if tt.wantErr {
				if err == nil {
					t.Errorf("NewDecimal(%q) expected error, got nil", tt.input)
				}
				return
			}
			
			if err != nil {
				t.Errorf("NewDecimal(%q) unexpected error: %v", tt.input, err)
				return
			}
			
			if result.String() != tt.want {
				t.Errorf("NewDecimal(%q) = %q, want %q", tt.input, result.String(), tt.want)
			}
		})
	}
}

func TestMustNewDecimal(t *testing.T) {
	// Test successful case
	result := MustNewDecimal("123.45")
	expected := "123.45"
	if result.String() != expected {
		t.Errorf("MustNewDecimal(\"123.45\") = %q, want %q", result.String(), expected)
	}
	
	// Test panic case
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("MustNewDecimal(\"invalid\") expected panic")
		}
	}()
	MustNewDecimal("invalid")
}

func TestNewDecimalFromInt(t *testing.T) {
	tests := []struct {
		input int64
		want  string
	}{
		{0, "0"},
		{123, "123"},
		{-456, "-456"},
		{9223372036854775807, "9223372036854775807"}, // max int64
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			result := NewDecimalFromInt(tt.input)
			if result.String() != tt.want {
				t.Errorf("NewDecimalFromInt(%d) = %q, want %q", tt.input, result.String(), tt.want)
			}
		})
	}
}

func TestDecimalArithmetic(t *testing.T) {
	a := MustNewDecimal("10.50")
	b := MustNewDecimal("3.25")
	
	// Test addition
	sum := a.Add(b)
	if sum.String() != "13.75" {
		t.Errorf("10.50 + 3.25 = %s, want 13.75", sum.String())
	}
	
	// Test subtraction
	diff := a.Subtract(b)
	if diff.String() != "7.25" {
		t.Errorf("10.50 - 3.25 = %s, want 7.25", diff.String())
	}
	
	// Test multiplication
	product := a.Multiply(b)
	expected := "34.125"
	if product.String() != expected {
		t.Errorf("10.50 * 3.25 = %s, want %s", product.String(), expected)
	}
	
	// Test division
	quotient, err := a.Divide(b)
	if err != nil {
		t.Errorf("10.50 / 3.25 unexpected error: %v", err)
	}
	// Division result may have many decimal places, so we'll check if it's approximately correct
	expectedFloat := 10.50 / 3.25
	actualFloat := quotient.Float64()
	if abs(actualFloat-expectedFloat) > 0.0001 {
		t.Errorf("10.50 / 3.25 = %f, want approximately %f", actualFloat, expectedFloat)
	}
}

func TestDecimalDivisionByZero(t *testing.T) {
	a := MustNewDecimal("10")
	zero := Zero()
	
	_, err := a.Divide(zero)
	if err == nil {
		t.Error("Division by zero should return error")
	}
	
	// Test panic version
	defer func() {
		if r := recover(); r == nil {
			t.Error("MustDivide by zero should panic")
		}
	}()
	a.MustDivide(zero)
}

func TestDecimalComparison(t *testing.T) {
	a := MustNewDecimal("10.50")
	b := MustNewDecimal("3.25")
	c := MustNewDecimal("10.50")
	
	// Test Equal
	if !a.Equal(c) {
		t.Error("10.50 should equal 10.50")
	}
	if a.Equal(b) {
		t.Error("10.50 should not equal 3.25")
	}
	
	// Test GreaterThan
	if !a.GreaterThan(b) {
		t.Error("10.50 should be greater than 3.25")
	}
	if b.GreaterThan(a) {
		t.Error("3.25 should not be greater than 10.50")
	}
	
	// Test LessThan
	if !b.LessThan(a) {
		t.Error("3.25 should be less than 10.50")
	}
	if a.LessThan(b) {
		t.Error("10.50 should not be less than 3.25")
	}
	
	// Test Compare
	if a.Compare(b) != 1 {
		t.Error("10.50 compared to 3.25 should return 1")
	}
	if b.Compare(a) != -1 {
		t.Error("3.25 compared to 10.50 should return -1")
	}
	if a.Compare(c) != 0 {
		t.Error("10.50 compared to 10.50 should return 0")
	}
}

func TestDecimalProperties(t *testing.T) {
	positive := MustNewDecimal("10.50")
	negative := MustNewDecimal("-5.25")
	zero := Zero()
	
	// Test IsZero
	if !zero.IsZero() {
		t.Error("Zero should be zero")
	}
	if positive.IsZero() {
		t.Error("10.50 should not be zero")
	}
	
	// Test IsPositive
	if !positive.IsPositive() {
		t.Error("10.50 should be positive")
	}
	if negative.IsPositive() {
		t.Error("-5.25 should not be positive")
	}
	if zero.IsPositive() {
		t.Error("0 should not be positive")
	}
	
	// Test IsNegative
	if !negative.IsNegative() {
		t.Error("-5.25 should be negative")
	}
	if positive.IsNegative() {
		t.Error("10.50 should not be negative")
	}
	if zero.IsNegative() {
		t.Error("0 should not be negative")
	}
	
	// Test Sign
	if positive.Sign() != 1 {
		t.Error("Sign of 10.50 should be 1")
	}
	if negative.Sign() != -1 {
		t.Error("Sign of -5.25 should be -1")
	}
	if zero.Sign() != 0 {
		t.Error("Sign of 0 should be 0")
	}
}

func TestDecimalAbs(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"10.50", "10.50"},
		{"-5.25", "5.25"},
		{"0", "0"},
		{"-0.001", "0.001"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			d := MustNewDecimal(tt.input)
			result := d.Abs()
			if result.String() != tt.want {
				t.Errorf("Abs(%s) = %s, want %s", tt.input, result.String(), tt.want)
			}
		})
	}
}

func TestDecimalNeg(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"10.50", "-10.50"},
		{"-5.25", "5.25"},
		{"0", "0"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			d := MustNewDecimal(tt.input)
			result := d.Neg()
			if result.String() != tt.want {
				t.Errorf("Neg(%s) = %s, want %s", tt.input, result.String(), tt.want)
			}
		})
	}
}

func TestDecimalRounding(t *testing.T) {
	d := MustNewDecimal("123.456789")
	
	tests := []struct {
		places int
		mode   RoundingMode
		want   string
	}{
		{2, RoundingModeHalfUp, "123.46"},
		{1, RoundingModeHalfUp, "123.5"},
		{0, RoundingModeHalfUp, "123"},
		{2, RoundingModeDown, "123.45"},
		{4, RoundingModeHalfUp, "123.4568"},
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			result := d.Round(tt.places, tt.mode)
			if result.StringFixed(tt.places) != tt.want {
				t.Errorf("Round(%d, %v) = %s, want %s", 
					tt.places, tt.mode, result.StringFixed(tt.places), tt.want)
			}
		})
	}
}

func TestDecimalStringFixed(t *testing.T) {
	tests := []struct {
		input  string
		places int
		want   string
	}{
		{"123.456", 2, "123.46"},
		{"123.4", 2, "123.40"},
		{"123", 2, "123.00"},
		{"123.999", 1, "124.0"},
		{"0", 3, "0.000"},
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			d := MustNewDecimal(tt.input)
			result := d.StringFixed(tt.places)
			if result != tt.want {
				t.Errorf("StringFixed(%s, %d) = %s, want %s", 
					tt.input, tt.places, result, tt.want)
			}
		})
	}
}

func TestDecimalMinMax(t *testing.T) {
	a := MustNewDecimal("10.5")
	b := MustNewDecimal("3.2")
	
	min := a.Min(b)
	if !min.Equal(b) {
		t.Errorf("Min(10.5, 3.2) = %s, want %s", min.String(), b.String())
	}
	
	max := a.Max(b)
	if !max.Equal(a) {
		t.Errorf("Max(10.5, 3.2) = %s, want %s", max.String(), a.String())
	}
}

func TestDecimalPow(t *testing.T) {
	tests := []struct {
		base string
		exp  int64
		want string
	}{
		{"2", 0, "1"},
		{"2", 1, "2"},
		{"2", 3, "8"},
		{"0.5", 2, "0.25"},
		{"10", -1, "0.1"},
		{"-2", 2, "4"},
		{"-2", 3, "-8"},
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			base := MustNewDecimal(tt.base)
			result := base.Pow(tt.exp)
			
			// For approximate comparison due to potential precision differences
			expected := MustNewDecimal(tt.want)
			if !result.Equal(expected) {
				// Allow small differences for complex calculations
				diff := result.Subtract(expected).Abs()
				tolerance := MustNewDecimal("0.0001")
				if diff.GreaterThan(tolerance) {
					t.Errorf("Pow(%s, %d) = %s, want %s", 
						tt.base, tt.exp, result.String(), tt.want)
				}
			}
		})
	}
}

func TestDecimalSqrt(t *testing.T) {
	tests := []struct {
		input   string
		want    string
		wantErr bool
	}{
		{"0", "0", false},
		{"1", "1", false},
		{"4", "2", false},
		{"0.25", "0.5", false},
		{"-1", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			d := MustNewDecimal(tt.input)
			result, err := d.Sqrt()
			
			if tt.wantErr {
				if err == nil {
					t.Errorf("Sqrt(%s) expected error", tt.input)
				}
				return
			}
			
			if err != nil {
				t.Errorf("Sqrt(%s) unexpected error: %v", tt.input, err)
				return
			}
			
			expected := MustNewDecimal(tt.want)
			diff := result.Subtract(expected).Abs()
			tolerance := MustNewDecimal("0.0001")
			
			if diff.GreaterThan(tolerance) {
				t.Errorf("Sqrt(%s) = %s, want %s", tt.input, result.String(), tt.want)
			}
		})
	}
}

// Helper function for float comparison
func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}