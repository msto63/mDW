// File: doc.go
// Title: Package Documentation for mathx
// Description: Package mathx provides extended mathematical operations for business
//              applications with precise decimal arithmetic, currency handling, and
//              financial calculations. This package is essential for the mDW platform's
//              financial operations.
// Author: msto63 with Claude Opus 4.0
// Version: v0.2.0
// Created: 2025-01-24
// Modified: 2025-01-26
//
// Change History:
// - 2025-01-24 v0.1.0: Initial implementation with decimal arithmetic and business functions
// - 2025-01-26 v0.2.0: Enhanced documentation with comprehensive structure and examples

// Package mathx provides extended mathematical operations for business applications.
//
// Package: mathx
// Title: Extended Mathematical Operations for Business Applications
// Description: This package provides precise decimal arithmetic, currency operations,
//              and business calculations essential for financial applications.
//              All operations avoid floating-point precision issues through decimal
//              arithmetic, making it suitable for financial and business contexts
//              where accuracy is critical.
// Author: msto63 with Claude Opus 4.0
// Version: v0.2.0
// Created: 2025-01-24
// Modified: 2025-01-26
//
// Overview
//
// The mathx package extends Go's standard math package with business-focused operations
// that require precise decimal arithmetic. It provides a comprehensive solution for
// financial calculations, currency handling, and business mathematics without the
// floating-point precision issues that can lead to rounding errors in monetary contexts.
//
// Key capabilities include:
//   - Arbitrary-precision decimal arithmetic with configurable precision
//   - Multiple rounding modes for different business requirements
//   - Currency operations with proper formatting and conversion
//   - Business calculations including interest, tax, and discount computations
//   - Statistical functions optimized for business data analysis
//   - Performance-optimized implementations with object pooling
//
// Architecture
//
// The package is structured around the Decimal type, which provides the foundation
// for all mathematical operations:
//
//   - Decimal: Core type for arbitrary-precision decimal numbers
//   - Currency: Specialized type for monetary values with currency codes
//   - Business: Functions for common business calculations
//   - Statistics: Statistical operations on decimal datasets
//
// The implementation uses the math/big package internally for precision while
// providing a developer-friendly API that feels natural for business logic.
//
// Usage Examples
//
// Basic decimal arithmetic:
//
//	// Create decimal values from strings for exact precision
//	price := mathx.NewDecimal("19.99")
//	taxRate := mathx.NewDecimal("0.19")
//	
//	// Calculate tax amount
//	tax := price.Multiply(taxRate)
//	total := price.Add(tax)
//	
//	// Format for display
//	fmt.Printf("Price: %s, Tax: %s, Total: %s\n",
//	    price.String(), tax.String(), total.String())
//
// Currency operations:
//
//	// Create currency values
//	amount := mathx.NewCurrency("1234.56", "EUR")
//	
//	// Format with locale-specific formatting
//	formatted := amount.Format() // "€1,234.56"
//	
//	// Convert between currencies
//	usdAmount, err := amount.Convert("USD", mathx.NewDecimal("1.18"))
//	if err != nil {
//	    // Handle conversion error
//	}
//
// Business calculations:
//
//	// Calculate loan payment
//	principal := mathx.NewDecimal("100000")
//	annualRate := mathx.NewDecimal("0.05")
//	months := 360
//	
//	payment := mathx.CalculateLoanPayment(principal, annualRate, months)
//	
//	// Calculate compound interest
//	futureValue := mathx.CompoundInterest(
//	    principal,
//	    annualRate,
//	    12, // compounds per year
//	    5,  // years
//	)
//	
//	// Apply discount with tax
//	originalPrice := mathx.NewDecimal("99.99")
//	discountRate := mathx.NewDecimal("0.15")
//	taxRate := mathx.NewDecimal("0.08")
//	
//	finalPrice := mathx.ApplyDiscountWithTax(
//	    originalPrice,
//	    discountRate,
//	    taxRate,
//	)
//
// Rounding modes:
//
//	value := mathx.NewDecimal("10.555")
//	
//	// Commercial rounding (half up)
//	commercial := value.Round(2, mathx.RoundingModeHalfUp) // 10.56
//	
//	// Banker's rounding (half even)
//	bankers := value.Round(2, mathx.RoundingModeHalfEven) // 10.56
//	
//	// Always round down (floor)
//	floor := value.Round(2, mathx.RoundingModeDown) // 10.55
//
// Performance Considerations
//
// The package is optimized for business applications with several performance features:
//
//   - Object pooling for big.Rat instances to reduce GC pressure
//   - Lazy evaluation of string representations
//   - Efficient algorithms for common operations
//   - Minimal memory allocations in hot paths
//
// Benchmark results show that decimal operations are approximately 10-20x slower
// than native float64 operations, which is acceptable for the precision gained.
// For high-frequency calculations, consider caching intermediate results.
//
// Best Practices
//
// 1. Always use string literals when creating decimal values for exact precision:
//
//	// Good - exact representation
//	exact := mathx.NewDecimal("0.1")
//	
//	// Bad - may have precision issues
//	approx := mathx.NewDecimalFromFloat(0.1)
//
// 2. Choose appropriate rounding modes for your business context:
//
//	// Financial calculations often use banker's rounding
//	financial := value.Round(2, mathx.RoundingModeHalfEven)
//	
//	// Customer-facing prices often use commercial rounding
//	display := value.Round(2, mathx.RoundingModeHalfUp)
//
// 3. Use Currency type for monetary values to maintain currency information:
//
//	// Good - maintains currency context
//	price := mathx.NewCurrency("99.99", "USD")
//	
//	// Less ideal - loses currency information
//	amount := mathx.NewDecimal("99.99")
//
// Integration with mDW
//
// The mathx package integrates seamlessly with other mDW Foundation components:
//
//   - Error handling uses the standard mDW error package
//   - Logging integration for debugging calculations
//   - JSON marshaling/unmarshaling for API communication
//   - Validation support for input constraints
//
// Example TCOL command usage:
//
//	INVOICE.CALCULATE tax_rate="0.19" discount="0.10"
//	PAYMENT.SCHEDULE principal="50000" rate="0.045" term="60"
//
// Error Handling
//
// All operations that can fail return errors using the mDW error package:
//
//	result, err := mathx.Divide(a, b)
//	if err != nil {
//	    // Check for specific error types
//	    if errors.Is(err, mathx.ErrDivisionByZero) {
//	        // Handle division by zero
//	    }
//	    return errors.Wrap(err, "failed to calculate ratio")
//	}
//
// Common error conditions include:
//   - Division by zero
//   - Invalid decimal string format
//   - Currency conversion errors
//   - Overflow in calculations
//
// Thread Safety
//
// All Decimal operations are thread-safe and can be used concurrently.
// The package uses sync.Pool for object pooling, which is also thread-safe.
// Currency conversion rates should be managed externally with appropriate
// synchronization if updated dynamically.
//
// Future Enhancements
//
// Planned additions to the package include:
//   - Matrix operations for financial modeling
//   - Time-value-of-money calculations
//   - Statistical distributions for risk analysis
//   - Integration with external currency rate providers
//   - Performance optimizations using SIMD instructions
//
// See Also
//
//   - Package errors: For error handling and wrapping
//   - Package log: For calculation logging and debugging
//   - Package validationx: For input validation
//   - math/big: Underlying precision arithmetic
//
package mathx