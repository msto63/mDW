// File: integration_examples.go
// Title: mDW Foundation Module Integration Examples
// Description: Practical, runnable examples demonstrating how to integrate
//              mDW Foundation modules effectively in real-world scenarios.
// Author: msto63 with Claude Sonnet 4.0
// Version: v0.1.0
// Created: 2025-01-25
// Modified: 2025-01-25
//
// Change History:
// - 2025-01-25 v0.1.0: Initial implementation of integration examples

package examples

import (
	"fmt"
	"strings"

	mdwerror "github.com/msto63/mDW/foundation/core/error"
	mdwvalidation "github.com/msto63/mDW/foundation/core/validation"
	mdwmathx "github.com/msto63/mDW/foundation/utils/mathx"
	mdwstringx "github.com/msto63/mDW/foundation/utils/stringx"
	mdwvalidationx "github.com/msto63/mDW/foundation/utils/validationx"
)

// ===============================
// Example 1: E-commerce Order Processing
// ===============================

// OrderProcessor demonstrates a complete e-commerce order processing pipeline
type OrderProcessor struct {
	taxRate      mdwmathx.Decimal
	shippingRate mdwmathx.Decimal
}

// Order represents an e-commerce order
type Order struct {
	CustomerEmail string
	Items         []OrderItem
	ShippingAddr  string
	DiscountCode  string
}

// OrderItem represents a single item in an order
type OrderItem struct {
	Name     string
	Price    string
	Quantity int
}

// OrderResult contains the processed order details
type OrderResult struct {
	Subtotal     string
	Tax          string
	Shipping     string
	Discount     string
	Total        string
	FormattedMsg string
}

// NewOrderProcessor creates a new order processor with tax and shipping rates
func NewOrderProcessor(taxRate, shippingRate string) (*OrderProcessor, error) {
	tax, err := mdwmathx.NewDecimal(taxRate)
	if err != nil {
		return nil, mdwerror.Wrap(err, "failed to parse tax rate").WithCode(mdwerror.CodeValidationFailed)
	}

	shipping, err := mdwmathx.NewDecimal(shippingRate)
	if err != nil {
		return nil, mdwerror.Wrap(err, "failed to parse shipping rate").WithCode(mdwerror.CodeValidationFailed)
	}

	return &OrderProcessor{
		taxRate:      tax,
		shippingRate: shipping,
	}, nil
}

// ProcessOrder demonstrates integration of stringx, mathx, and validationx modules
func (op *OrderProcessor) ProcessOrder(order Order) (*OrderResult, error) {
	// Phase 1: Input Validation (stringx + validationx)
	if err := op.validateOrder(order); err != nil {
		return nil, err
	}

	// Phase 2: Financial Calculations (mathx)
	subtotal, err := op.calculateSubtotal(order.Items)
	if err != nil {
		return nil, err
	}

	tax := subtotal.Multiply(op.taxRate)
	shipping := op.shippingRate

	// Apply discount if valid
	discount, err := op.calculateDiscount(order.DiscountCode, subtotal)
	if err != nil {
		return nil, err
	}

	// Calculate final total
	total := subtotal.Add(tax)
	total = total.Add(shipping)
	total = total.Subtract(discount)

	// Phase 3: Output Formatting (stringx)
	result := &OrderResult{
		Subtotal: subtotal.Round(2, mdwmathx.RoundingModeHalfUp).String(),
		Tax:      tax.Round(2, mdwmathx.RoundingModeHalfUp).String(),
		Shipping: shipping.Round(2, mdwmathx.RoundingModeHalfUp).String(),
		Discount: discount.Round(2, mdwmathx.RoundingModeHalfUp).String(),
		Total:    total.Round(2, mdwmathx.RoundingModeHalfUp).String(),
	}

	// Create formatted confirmation message
	result.FormattedMsg = op.formatConfirmationMessage(order.CustomerEmail, result)

	return result, nil
}

// validateOrder demonstrates stringx and validationx integration for input validation
func (op *OrderProcessor) validateOrder(order Order) error {
	// stringx: Clean and validate email  
	cleanEmail := strings.TrimSpace(order.CustomerEmail)
	if mdwstringx.IsEmpty(cleanEmail) || !strings.Contains(cleanEmail, "@") {
		return mdwerror.New("invalid email format").WithCode(mdwerror.CodeValidationFailed).WithDetail("field", "customer_email").WithDetail("value", cleanEmail)
	}

	// stringx: Validate shipping address
	cleanAddr := strings.TrimSpace(order.ShippingAddr)
	if mdwstringx.IsEmpty(cleanAddr) {
		return mdwerror.New("shipping address required").WithCode(mdwerror.CodeValidationFailed).WithDetail("field", "shipping_address")
	}

	// validationx: Comprehensive order validation using concrete validators
	result := mdwvalidationx.ValidateStruct(struct {
		Email   string `validate:"required,email"`
		Address string `validate:"required,min_length:10"`
	}{
		Email:   cleanEmail,
		Address: cleanAddr,
	})

	if !result.Valid {
		return mdwerror.New("order validation failed").WithCode(mdwerror.CodeValidationFailed).WithDetail("errors", strings.Join(result.ErrorMessages(), ", "))
	}

	// Validate items
	if len(order.Items) == 0 {
		return mdwerror.New("order must contain at least one item").WithCode(mdwerror.CodeValidationFailed).WithDetail("item_count", len(order.Items))
	}

	return nil
}

// calculateSubtotal demonstrates mathx integration for financial calculations
func (op *OrderProcessor) calculateSubtotal(items []OrderItem) (mdwmathx.Decimal, error) {
	var subtotal mdwmathx.Decimal

	for i, item := range items {
		// stringx: Clean item name
		cleanName := strings.TrimSpace(item.Name)
		if mdwstringx.IsEmpty(cleanName) {
			return subtotal, mdwerror.New("item name cannot be empty").WithCode(mdwerror.CodeValidationFailed).WithDetail("item_index", i)
		}

		// mathx: Parse and validate price
		price, err := mdwmathx.NewDecimal(item.Price)
		if err != nil {
			return subtotal, mdwerror.Wrap(err, "failed to parse item price").WithCode(mdwerror.CodeValidationFailed)
		}

		if price.IsNegative() {
			return subtotal, mdwerror.New("item price must be positive").WithCode(mdwerror.CodeValidationFailed).WithDetail("price", price.String())
		}

		// mathx: Calculate line total
		quantityDecimal := mdwmathx.NewDecimalFromInt(int64(item.Quantity))
		lineTotal := price.Multiply(quantityDecimal)
		subtotal = subtotal.Add(lineTotal)
	}

	return subtotal, nil
}

// calculateDiscount demonstrates conditional logic with mathx calculations
func (op *OrderProcessor) calculateDiscount(discountCode string, subtotal mdwmathx.Decimal) (mdwmathx.Decimal, error) {
	// stringx: Clean and validate discount code
	cleanCode := strings.ToUpper(strings.TrimSpace(discountCode))
	
	if mdwstringx.IsEmpty(cleanCode) {
		zero, _ := mdwmathx.NewDecimal("0")
		return zero, nil
	}

	// Simple discount logic (in real apps, this would query a database)
	discountRate, _ := mdwmathx.NewDecimal("0")
	switch cleanCode {
	case "SAVE10":
		discountRate, _ = mdwmathx.NewDecimal("0.10")
	case "SAVE20":
		discountRate, _ = mdwmathx.NewDecimal("0.20")
	case "WELCOME":
		discountRate, _ = mdwmathx.NewDecimal("0.05")
	default:
		zero, _ := mdwmathx.NewDecimal("0")
		return zero, mdwerror.New("invalid discount code").WithCode(mdwerror.CodeValidationFailed).WithDetail("code", cleanCode)
	}

	return subtotal.Multiply(discountRate), nil
}

// formatConfirmationMessage demonstrates stringx formatting capabilities
func (op *OrderProcessor) formatConfirmationMessage(email string, result *OrderResult) string {
	// stringx: Format customer name from email
	customerName := mdwstringx.ToTitleCase(strings.Split(email, "@")[0])
	
	// stringx: Build formatted message
	var msg strings.Builder
	msg.WriteString(fmt.Sprintf("Dear %s,\n\n", customerName))
	msg.WriteString("Your order has been processed successfully!\n\n")
	msg.WriteString("Order Summary:\n")
	msg.WriteString(fmt.Sprintf("  Subtotal: $%s\n", mdwstringx.PadLeft(result.Subtotal, 8, ' ')))
	msg.WriteString(fmt.Sprintf("  Tax:      $%s\n", mdwstringx.PadLeft(result.Tax, 8, ' ')))
	msg.WriteString(fmt.Sprintf("  Shipping: $%s\n", mdwstringx.PadLeft(result.Shipping, 8, ' ')))
	
	if result.Discount != "0" {
		msg.WriteString(fmt.Sprintf("  Discount: -$%s\n", mdwstringx.PadLeft(result.Discount, 7, ' ')))
	}
	
	msg.WriteString(fmt.Sprintf("  Total:    $%s\n\n", mdwstringx.PadLeft(result.Total, 8, ' ')))
	msg.WriteString("Thank you for your business!")

	return msg.String()
}

// ===============================
// Example 2: Data Validation and Transformation Pipeline
// ===============================

// DataPipeline demonstrates a complete data processing pipeline
type DataPipeline struct {
	validationRules map[string][]string
	transformRules  map[string]string
}

// DataRecord represents an input data record
type DataRecord struct {
	ID       string
	Name     string
	Email    string
	Amount   string
	Category string
	Tags     []string
}

// ProcessedRecord represents a processed and validated data record
type ProcessedRecord struct {
	ID              string
	NormalizedName  string
	ValidatedEmail  string
	ParsedAmount    mdwmathx.Decimal
	CategoryCode    string
	ProcessedTags   []string
	ValidationScore int
}

// NewDataPipeline creates a new data processing pipeline
func NewDataPipeline() *DataPipeline {
	return &DataPipeline{
		validationRules: map[string][]string{
			"email":    {"required", "email"},
			"name":     {"required", "min_length:2"},
			"amount":   {"required", "numeric"},
			"category": {"required"},
		},
		transformRules: map[string]string{
			"name":     "title_case",
			"email":    "lower_case",
			"category": "upper_case",
		},
	}
}

// ProcessRecord demonstrates comprehensive data processing using multiple modules
func (dp *DataPipeline) ProcessRecord(record DataRecord) (*ProcessedRecord, error) {
	processed := &ProcessedRecord{
		ID: strings.TrimSpace(record.ID),
	}

	// Phase 1: stringx cleaning and basic validation
	if err := dp.cleanAndValidateFields(&record); err != nil {
		return nil, err
	}

	// Phase 2: Advanced validation with mdwvalidation
	if err := dp.performAdvancedValidation(record); err != nil {
		return nil, err
	}

	// Phase 3: Data transformation
	if err := dp.transformFields(record, processed); err != nil {
		return nil, err
	}

	// Phase 4: Calculate validation score
	processed.ValidationScore = dp.calculateValidationScore(record)

	return processed, nil
}

// cleanAndValidateFields demonstrates stringx cleaning operations
func (dp *DataPipeline) cleanAndValidateFields(record *DataRecord) error {
	// Clean all string fields
	record.Name = strings.TrimSpace(record.Name)
	record.Email = strings.TrimSpace(record.Email)
	record.Amount = strings.TrimSpace(record.Amount)
	record.Category = strings.TrimSpace(record.Category)

	// Basic validation
	if mdwstringx.IsEmpty(record.Name) {
		return mdwerror.New("name is required").WithCode(mdwerror.CodeValidationFailed).WithDetail("field", "name")
	}

	if mdwstringx.IsEmpty(record.Email) {
		return mdwerror.New("email is required").WithCode(mdwerror.CodeValidationFailed).WithDetail("field", "email")
	}

	if !strings.Contains(record.Email, "@") {
		return mdwerror.New("invalid email format").WithCode(mdwerror.CodeValidationFailed).WithDetail("field", "email").WithDetail("value", record.Email)
	}

	if _, err := mdwmathx.NewDecimal(record.Amount); err != nil {
		return mdwerror.New("amount must be numeric").WithCode(mdwerror.CodeValidationFailed).WithDetail("field", "amount").WithDetail("value", record.Amount)
	}

	return nil
}

// performAdvancedValidation demonstrates mdwvalidation integration
func (dp *DataPipeline) performAdvancedValidation(record DataRecord) error {
	// Use mdwvalidation for comprehensive validation
	result := mdwvalidationx.ValidateStruct(struct {
		Name     string `validate:"required,min_length:2,max_length:100"`
		Email    string `validate:"required,email"`
		Amount   string `validate:"required,min:0"`
		Category string `validate:"required,min_length:1"`
	}{
		Name:     record.Name,
		Email:    record.Email,
		Amount:   record.Amount,
		Category: record.Category,
	})

	if !result.Valid {
		return mdwerror.New("advanced validation failed").WithCode(mdwerror.CodeValidationFailed).WithDetail("errors", strings.Join(result.ErrorMessages(), ", "))
	}

	return nil
}

// transformFields demonstrates data transformation using multiple modules
func (dp *DataPipeline) transformFields(record DataRecord, processed *ProcessedRecord) error {
	// stringx: Transform name
	processed.NormalizedName = mdwstringx.ToTitleCase(record.Name)

	// stringx: Transform email
	processed.ValidatedEmail = strings.ToLower(record.Email)

	// mathx: Parse and validate amount
	amount, err := mdwmathx.NewDecimal(record.Amount)
	if err != nil {
		return mdwerror.Wrap(err, "failed to parse amount").WithCode(mdwerror.CodeValidationFailed)
	}
	processed.ParsedAmount = amount

	// stringx: Transform category
	processed.CategoryCode = strings.ToUpper(record.Category)

	// stringx: Process tags
	processed.ProcessedTags = make([]string, len(record.Tags))
	for i, tag := range record.Tags {
		processed.ProcessedTags[i] = mdwstringx.ToSnakeCase(strings.TrimSpace(tag))
	}

	return nil
}

// calculateValidationScore demonstrates complex logic using utility modules
func (dp *DataPipeline) calculateValidationScore(record DataRecord) int {
	score := 0

	// Email validation score
	if strings.Contains(record.Email, "@") {
		score += 20
	}

	// Name quality score
	nameWords := len(strings.Fields(record.Name))
	if nameWords >= 2 {
		score += 15
	}

	// Amount precision score
	if amount, err := mdwmathx.NewDecimal(record.Amount); err == nil {
		if amount.IsPositive() {
			score += 20
		}
		// Bonus for reasonable amounts
		threshold, _ := mdwmathx.NewDecimal("1000000") // 1 million threshold
		if amount.LessThan(threshold) {
			score += 10
		}
	}

	// Category specificity score
	if len(record.Category) > 3 {
		score += 10
	}

	// Tags richness score
	if len(record.Tags) > 0 {
		score += 5
		if len(record.Tags) >= 3 {
			score += 10
		}
	}

	// Bonus for comprehensive data
	if score >= 70 {
		score += 10
	}

	return score
}

// ===============================
// Example 3: Error Recovery and Resilience
// ===============================

// ResilientProcessor demonstrates error recovery patterns across modules
type ResilientProcessor struct {
	maxRetries    int
	fallbackMode  bool
	errorAnalyzer *ErrorAnalyzer
}

// ErrorAnalyzer helps analyze and categorize errors for recovery strategies
type ErrorAnalyzer struct {
	errorCounts map[string]int
}

// ProcessingResult contains the result and any recovered errors
type ProcessingResult struct {
	Success      bool
	Result       interface{}
	AttemptCount int
	Errors       []error
	RecoveryUsed bool
}

// NewResilientProcessor creates a processor with error recovery capabilities
func NewResilientProcessor(maxRetries int, enableFallback bool) *ResilientProcessor {
	return &ResilientProcessor{
		maxRetries:   maxRetries,
		fallbackMode: enableFallback,
		errorAnalyzer: &ErrorAnalyzer{
			errorCounts: make(map[string]int),
		},
	}
}

// ProcessWithRecovery demonstrates error recovery across module boundaries
func (rp *ResilientProcessor) ProcessWithRecovery(input string) *ProcessingResult {
	result := &ProcessingResult{
		Errors: make([]error, 0),
	}

	for attempt := 1; attempt <= rp.maxRetries; attempt++ {
		result.AttemptCount = attempt

		processedResult, err := rp.attemptProcessing(input)
		if err == nil {
			result.Success = true
			result.Result = processedResult
			return result
		}

		result.Errors = append(result.Errors, err)

		// Analyze error and attempt recovery
		recoveryStrategy := rp.analyzeAndRecoverFromError(err, &input)
		if recoveryStrategy != "" {
			result.RecoveryUsed = true
			// Continue with modified input
			continue
		}

		// If no recovery possible and fallback enabled, try fallback
		if attempt == rp.maxRetries && rp.fallbackMode {
			fallbackResult := rp.performFallbackProcessing(input)
			if fallbackResult != nil {
				result.Success = true
				result.Result = fallbackResult
				result.RecoveryUsed = true
				return result
			}
		}
	}

	return result
}

// attemptProcessing demonstrates multi-module processing with error points
func (rp *ResilientProcessor) attemptProcessing(input string) (interface{}, error) {
	// Step 1: stringx validation and cleaning
	cleaned := strings.TrimSpace(input)
	if mdwstringx.IsEmpty(cleaned) {
		return nil, mdwerror.New("empty input after cleaning").WithCode(mdwerror.CodeValidationFailed).WithDetail("original_input", input)
	}

	// Step 2: Determine data type and process accordingly
	if decimal, err := mdwmathx.NewDecimal(cleaned); err == nil {
		// mathx processing for numeric data succeeded
		
		// Perform some calculation to test mathx error recovery
		if decimal.IsZero() {
			return nil, mdwerror.New("zero values not allowed").WithCode(mdwerror.CodeValidationFailed).WithDetail("value", decimal.String())
		}
		
		// Return processed numeric result
		return map[string]interface{}{
			"type":  "numeric",
			"value": decimal,
			"formatted": mdwstringx.PadLeft(decimal.String(), 10, '0'),
		}, nil
	} else if strings.Contains(cleaned, "@") && len(strings.Split(cleaned, "@")) == 2 {
		// Email processing
		normalized := strings.ToLower(cleaned)
		return map[string]interface{}{
			"type":       "email",
			"value":      normalized,
			"domain":     strings.Split(normalized, "@")[1],
			"formatted":  mdwstringx.ToTitleCase(strings.Split(normalized, "@")[0]),
		}, nil
	} else {
		// Generic string processing
		if len(cleaned) < 3 {
			return nil, mdwerror.New("input too short").WithCode(mdwerror.CodeValidationFailed).WithDetail("length", len(cleaned))
		}
		
		return map[string]interface{}{
			"type":      "text",
			"value":     cleaned,
			"formatted": mdwstringx.ToTitleCase(cleaned),
			"length":    len(cleaned),
		}, nil
	}
}

// analyzeAndRecoverFromError demonstrates intelligent error recovery
func (rp *ResilientProcessor) analyzeAndRecoverFromError(err error, input *string) string {
	if mdwErr, ok := err.(*mdwerror.Error); ok {
		code := string(mdwErr.Code())
		rp.errorAnalyzer.errorCounts[code]++

		switch code {
		case "STRINGX_VALIDATION_FAILED":
			// Try to clean input more aggressively - remove special chars
			cleaned := ""
			for _, r := range *input {
				if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
					cleaned += string(r)
				}
			}
			*input = cleaned
			return "aggressive_cleaning"

		case "MATHX_INVALID_DECIMAL":
			// Try to extract numbers from string
			cleaned := ""
			for _, r := range *input {
				if (r >= '0' && r <= '9') || r == '.' || r == '-' {
					cleaned += string(r)
				}
			}
			if cleaned != "" {
				*input = cleaned
				return "number_extraction"
			}

		case "STRINGX_LENGTH_TOO_SHORT":
			// Try to pad short strings
			if len(*input) < 3 {
				*input = mdwstringx.PadRight(*input, 3, 'X')
				return "padding"
			}
		}
	}

	return "" // No recovery strategy available
}

// performFallbackProcessing provides a last-resort processing option
func (rp *ResilientProcessor) performFallbackProcessing(input string) interface{} {
	// Always return a safe, minimal result - simple cleaning
	cleaned := ""
	for _, r := range input {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == ' ' {
			cleaned += string(r)
		}
	}
	cleaned = strings.TrimSpace(cleaned)
	if mdwstringx.IsEmpty(cleaned) {
		cleaned = "fallback_value"
	}

	return map[string]interface{}{
		"type":     "fallback",
		"value":    cleaned,
		"original": input,
		"note":     "Processed using fallback mode due to errors",
	}
}

// GetErrorStatistics returns error analysis for monitoring
func (rp *ResilientProcessor) GetErrorStatistics() map[string]int {
	return rp.errorAnalyzer.errorCounts
}

// ===============================
// Example 4: Validation System Architecture Demonstration
// ===============================

// ValidationBoundariesDemo demonstrates the proper use of both validation modules
type ValidationBoundariesDemo struct {
	logger interface{} // Placeholder for logger interface
}

// NewValidationBoundariesDemo creates a new demonstration instance
func NewValidationBoundariesDemo() *ValidationBoundariesDemo {
	return &ValidationBoundariesDemo{}
}

// DemonstrateValidationArchitecture shows the clear boundaries between validation modules
func (vbd *ValidationBoundariesDemo) DemonstrateValidationArchitecture() {
	fmt.Println("=== Validation System Architecture Demonstration ===")
	
	// Phase 1: Using core/validation framework interfaces and types
	fmt.Println("\n1. Core Validation Framework (pkg/core/validation):")
	fmt.Println("   - Provides validation interfaces and types")
	fmt.Println("   - Contains NO concrete validators")
	fmt.Println("   - Used for building validation infrastructure")
	
	// Create validation results using core framework
	successResult := mdwvalidation.NewValidationResult()
	fmt.Printf("   Success result: %s\n", successResult.String())
	
	errorResult := mdwvalidation.NewValidationError(
		mdwvalidation.CodeEmail, "invalid email format")
	fmt.Printf("   Error result: %s\n", errorResult.String())
	
	fieldErrorResult := mdwvalidation.NewValidationErrorWithField(
		mdwvalidation.CodeRequired, "name", "name is required", "")
	fmt.Printf("   Field error result: %s\n", fieldErrorResult.String())
	
	// Phase 2: Using validationx concrete validators
	fmt.Println("\n2. Concrete Validators (pkg/utils/validationx):")
	fmt.Println("   - Implements actual validation logic")
	fmt.Println("   - Uses core/validation types for results")
	fmt.Println("   - Provides ready-to-use validators")
	
	// Demonstrate concrete validators
	testData := struct {
		Email string `validate:"required,email"`
		Age   int    `validate:"required,min:0,max:150"`
	}{
		Email: "invalid-email",
		Age:   -5,
	}
	
	concreteResult := mdwvalidationx.ValidateStruct(testData)
	fmt.Printf("   Concrete validation result: valid=%t, errors=%d\n", 
		concreteResult.Valid, len(concreteResult.Errors))
	
	for _, err := range concreteResult.Errors {
		fmt.Printf("     - %s: %s\n", err.Field, err.Message)
	}
	
	// Phase 3: Demonstrate proper integration
	fmt.Println("\n3. Proper Integration Pattern:")
	vbd.demonstrateProperIntegration()
}

// demonstrateProperIntegration shows how to use both modules together
func (vbd *ValidationBoundariesDemo) demonstrateProperIntegration() {
	// Use case: Building a custom validator that uses both modules
	
	// Step 1: Use core framework types for structure
	fmt.Println("   Creating custom validator using framework interfaces...")
	
	// Step 2: Use concrete validators for actual validation
	customValidator := func(value interface{}) mdwvalidation.ValidationResult {
		// This demonstrates the proper pattern:
		// - Use core/validation types for structure and interfaces
		// - Use utils/validationx for concrete validation logic
		
		if value == nil {
			return mdwvalidation.NewValidationError(
				mdwvalidation.CodeRequired, "value is required")
		}
		
		// Convert to string for email validation
		str, ok := value.(string)
		if !ok {
			return mdwvalidation.NewValidationError(
				mdwvalidation.CodeType, "value must be a string")
		}
		
		// Use concrete validator from validationx
		if !mdwvalidationx.IsValidEmail(str) {
			return mdwvalidation.NewValidationError(
				mdwvalidation.CodeEmail, "invalid email format")
		}
		
		return mdwvalidation.NewValidationResult()
	}
	
	// Test the custom validator
	validEmail := customValidator("user@example.com")
	invalidEmail := customValidator("invalid-email")
	
	fmt.Printf("   Valid email result: %s\n", validEmail.String())
	fmt.Printf("   Invalid email result: %s\n", invalidEmail.String())
	
	// Step 3: Demonstrate chain composition using framework
	fmt.Println("   Building validator chain using framework infrastructure...")
	
	// This shows how you would build complex validation using both:
	// - Framework types and patterns from core/validation
	// - Concrete validation logic from utils/validationx
	combinedResult := vbd.validateUserData(UserData{
		Email: "test@example.com",
		Name:  "John Doe",
		Age:   25,
	})
	
	fmt.Printf("   Combined validation result: %s\n", combinedResult.String())
}

// UserData represents data to be validated
type UserData struct {
	Email string
	Name  string
	Age   int
}

// validateUserData demonstrates proper integration of both validation modules
func (vbd *ValidationBoundariesDemo) validateUserData(data UserData) mdwvalidation.ValidationResult {
	// Use framework types to build result
	result := mdwvalidation.NewValidationResult()
	
	// Use concrete validators for actual validation
	if !mdwvalidationx.IsValidEmail(data.Email) {
		emailError := mdwvalidation.NewValidationErrorWithField(
			mdwvalidation.CodeEmail, "email", "invalid email format", data.Email)
		result = mdwvalidation.Combine(result, emailError)
	}
	
	if mdwstringx.IsEmpty(data.Name) {
		nameError := mdwvalidation.NewValidationErrorWithField(
			mdwvalidation.CodeRequired, "name", "name is required", data.Name)
		result = mdwvalidation.Combine(result, nameError)
	}
	
	// Add custom validation using framework types
	if data.Age < 0 || data.Age > 150 {
		ageError := mdwvalidation.NewValidationErrorWithField(
			mdwvalidation.CodeRange, "age", "age must be between 0 and 150", data.Age)
		result = mdwvalidation.Combine(result, ageError)
	}
	
	return result
}

// ===============================
// Example Usage Functions
// ===============================

// DemonstrateOrderProcessing shows how to use the OrderProcessor
func DemonstrateOrderProcessing() {
	fmt.Println("=== E-commerce Order Processing Example ===")
	
	processor, err := NewOrderProcessor("0.08", "5.99")
	if err != nil {
		fmt.Printf("Failed to create processor: %v\n", err)
		return
	}

	order := Order{
		CustomerEmail: "john.doe@example.com",
		Items: []OrderItem{
			{Name: "Widget A", Price: "29.99", Quantity: 2},
			{Name: "Widget B", Price: "15.50", Quantity: 1},
			{Name: "Widget C", Price: "42.75", Quantity: 3},
		},
		ShippingAddr: "123 Main Street, Anytown, AN 12345",
		DiscountCode: "SAVE10",
	}

	result, err := processor.ProcessOrder(order)
	if err != nil {
		fmt.Printf("Order processing failed: %v\n", err)
		return
	}

	fmt.Printf("Order processed successfully!\n")
	fmt.Printf("Subtotal: $%s\n", result.Subtotal)
	fmt.Printf("Tax: $%s\n", result.Tax)
	fmt.Printf("Shipping: $%s\n", result.Shipping)
	fmt.Printf("Discount: $%s\n", result.Discount)
	fmt.Printf("Total: $%s\n\n", result.Total)
	fmt.Printf("Confirmation Message:\n%s\n", result.FormattedMsg)
}

// DemonstrateDataPipeline shows how to use the DataPipeline
func DemonstrateDataPipeline() {
	fmt.Println("=== Data Validation and Transformation Pipeline Example ===")
	
	pipeline := NewDataPipeline()

	record := DataRecord{
		ID:       "  12345  ",
		Name:     "john doe",
		Email:    " JOHN.DOE@EXAMPLE.COM ",
		Amount:   "1234.56",
		Category: "premium",
		Tags:     []string{"VIP Customer", "Frequent Buyer", "High Value"},
	}

	processed, err := pipeline.ProcessRecord(record)
	if err != nil {
		fmt.Printf("Data processing failed: %v\n", err)
		return
	}

	fmt.Printf("Data processed successfully!\n")
	fmt.Printf("ID: %s\n", processed.ID)
	fmt.Printf("Normalized Name: %s\n", processed.NormalizedName)
	fmt.Printf("Validated Email: %s\n", processed.ValidatedEmail)
	fmt.Printf("Parsed Amount: %s\n", processed.ParsedAmount.String())
	fmt.Printf("Category Code: %s\n", processed.CategoryCode)
	fmt.Printf("Processed Tags: %v\n", processed.ProcessedTags)
	fmt.Printf("Validation Score: %d/100\n", processed.ValidationScore)
}

// DemonstrateErrorRecovery shows how to use the ResilientProcessor
func DemonstrateErrorRecovery() {
	fmt.Println("=== Error Recovery and Resilience Example ===")
	
	processor := NewResilientProcessor(3, true)

	testInputs := []string{
		"123.45",           // Valid numeric
		"user@example.com", // Valid email
		"  ",               // Empty after trim (will need recovery)
		"abc123def",        // Mixed content (will need recovery)
		"xy",               // Too short (will need recovery)
	}

	for i, input := range testInputs {
		fmt.Printf("\nProcessing input %d: '%s'\n", i+1, input)
		result := processor.ProcessWithRecovery(input)
		
		if result.Success {
			fmt.Printf("✓ Success after %d attempt(s)\n", result.AttemptCount)
			if result.RecoveryUsed {
				fmt.Printf("  Recovery strategy used\n")
			}
			fmt.Printf("  Result: %+v\n", result.Result)
		} else {
			fmt.Printf("✗ Failed after %d attempt(s)\n", result.AttemptCount)
			fmt.Printf("  Errors: %v\n", result.Errors)
		}
	}

	fmt.Printf("\nError Statistics:\n")
	for code, count := range processor.GetErrorStatistics() {
		fmt.Printf("  %s: %d occurrences\n", code, count)
	}
}

// DemonstrateValidationBoundaries shows the proper validation architecture
func DemonstrateValidationBoundaries() {
	demo := NewValidationBoundariesDemo()
	demo.DemonstrateValidationArchitecture()
}