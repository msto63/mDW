# mDW Foundation Module Integration Patterns

## Overview

This document provides comprehensive integration patterns for combining mDW Foundation modules effectively. These patterns demonstrate real-world scenarios where multiple modules work together to provide complete solutions.

## Core Integration Principles

### 1. Error Context Preservation
When errors flow between modules, context from each module is preserved:

```go
// stringx validation error
validationErr := errorutils.ValidationFailed("stringx", "email", "invalid@", "must be valid email")

// mathx processing error that wraps the validation error
processingErr := errorutils.OperationFailed("mathx", "calculate", validationErr)

// Final error preserves full context chain
fmt.Println(processingErr.Error())
// Output: "mathx.calculate operation failed: stringx.validate_email: validation failed for field email: must be valid email"
```

### 2. Type-Safe Data Flow
Data flows between modules using type-safe interfaces:

```go
// stringx processes input
cleanInput := stringx.Trim(rawInput)
validated := stringx.ValidateEmail(cleanInput)

// mathx processes numerical data
if amount, err := mathx.ParseDecimal(validated); err == nil {
    result := mathx.Add(amount, mathx.NewDecimal("10.50"))
}
```

### 3. Consistent Error Handling
All modules use standardized mDW error patterns:

```go
func ProcessUserData(input string) (*Result, error) {
    // stringx validation with consistent error format
    if err := stringx.ValidateRequired(input); err != nil {
        return nil, errorutils.ValidationFailed("processor", "input", input, "required field")
    }
    
    // mathx conversion with error wrapping
    decimal, err := mathx.ParseDecimal(input)
    if err != nil {
        return nil, errorutils.OperationFailed("processor", "parse_amount", err)
    }
    
    return &Result{Amount: decimal}, nil
}
```

## Common Integration Patterns

### Pattern 1: Input Validation Pipeline

Combines multiple modules for comprehensive input validation:

```go
package integration

import (
    "github.com/msto63/mDW/foundation/pkg/utils/stringx"
    "github.com/msto63/mDW/foundation/pkg/utils/mathx"
    "github.com/msto63/mDW/foundation/pkg/utils/validationx"
    errorutils "github.com/msto63/mDW/foundation/pkg/core/errors"
)

type ValidationPipeline struct {
    strictMode bool
}

func NewValidationPipeline(strict bool) *ValidationPipeline {
    return &ValidationPipeline{strictMode: strict}
}

func (vp *ValidationPipeline) ValidateUserInput(input map[string]string) error {
    // 1. stringx: Clean and validate string formats
    email := stringx.Trim(input["email"])
    if !stringx.IsValidEmail(email) {
        return errorutils.ValidationFailed("stringx", "email", email, "invalid email format")
    }
    
    name := stringx.Trim(input["name"])
    if len(name) < 2 {
        return errorutils.ValidationFailed("stringx", "name", name, "name too short")
    }
    
    // 2. mathx: Validate numerical inputs
    if amountStr, exists := input["amount"]; exists {
        amount, err := mathx.ParseDecimal(amountStr)
        if err != nil {
            return errorutils.OperationFailed("mathx", "parse_amount", err)
        }
        
        if mathx.IsNegative(amount) {
            return errorutils.ValidationFailed("mathx", "amount", amount.String(), "must be positive")
        }
    }
    
    // 3. validationx: Advanced validation rules
    result := validationx.ValidateStruct(struct {
        Email string `validate:"required,email"`
        Name  string `validate:"required,min_length:2"`
    }{
        Email: email,
        Name:  name,
    })
    
    if result.HasErrors() {
        return errorutils.ValidationFailed("validationx", "struct", input, 
            "validation failed: " + result.ErrorMessages()[0])
    }
    
    return nil
}
```

### Pattern 2: Financial Calculation Workflow

Demonstrates precise financial calculations using multiple modules:

```go
package integration

import (
    "github.com/msto63/mDW/foundation/pkg/utils/stringx"
    "github.com/msto63/mDW/foundation/pkg/utils/mathx"
    errorutils "github.com/msto63/mDW/foundation/pkg/core/errors"
)

type FinancialProcessor struct {
    taxRate    mathx.Decimal
    currency   string
}

func NewFinancialProcessor(taxRate string, currency string) (*FinancialProcessor, error) {
    rate, err := mathx.ParseDecimal(taxRate)
    if err != nil {
        return nil, errorutils.OperationFailed("financial", "parse_tax_rate", err)
    }
    
    return &FinancialProcessor{
        taxRate:  rate,
        currency: currency,
    }, nil
}

func (fp *FinancialProcessor) CalculateInvoiceTotal(lineItems []string) (string, error) {
    var subtotal mathx.Decimal
    
    // 1. stringx: Parse and clean input
    for i, item := range lineItems {
        cleanItem := stringx.Trim(item)
        if stringx.IsEmpty(cleanItem) {
            return "", errorutils.ValidationFailed("stringx", "line_item", i, "empty line item")
        }
        
        // 2. mathx: Parse and accumulate amounts
        amount, err := mathx.ParseDecimal(cleanItem)
        if err != nil {
            return "", errorutils.OperationFailed("mathx", "parse_line_item", err)
        }
        
        subtotal = mathx.Add(subtotal, amount)
    }
    
    // 3. mathx: Calculate tax and total
    tax := mathx.Multiply(subtotal, fp.taxRate)
    total := mathx.Add(subtotal, tax)
    
    // 4. stringx: Format output
    formattedTotal := mathx.RoundToDecimalPlaces(total, 2).String()
    return stringx.PadLeft(formattedTotal, 10, " "), nil
}
```

### Pattern 3: Data Transformation Chain

Shows how data flows through multiple transformation modules:

```go
package integration

import (
    "github.com/msto63/mDW/foundation/pkg/utils/stringx"
    "github.com/msto63/mDW/foundation/pkg/utils/slicex"
    "github.com/msto63/mDW/foundation/pkg/utils/mapx"
    errorutils "github.com/msto63/mDW/foundation/pkg/core/errors"
)

type DataTransformer struct {
    config map[string]interface{}
}

func NewDataTransformer(config map[string]interface{}) *DataTransformer {
    return &DataTransformer{config: config}
}

func (dt *DataTransformer) ProcessDataSet(rawData []map[string]string) ([]map[string]interface{}, error) {
    var results []map[string]interface{}
    
    for i, record := range rawData {
        // 1. mapx: Validate and clean record structure
        if mapx.IsEmpty(record) {
            return nil, errorutils.ValidationFailed("mapx", "record", i, "empty record")
        }
        
        cleanRecord := make(map[string]interface{})
        
        // 2. stringx: Clean and transform string values
        for key, value := range record {
            cleanKey := stringx.ToSnakeCase(strings.Trim(key))
            cleanValue := stringx.Trim(value)
            
            // Apply type conversion based on field patterns
            if stringx.IsNumeric(cleanValue) {
                if converted, err := mathx.ParseDecimal(cleanValue); err == nil {
                    cleanRecord[cleanKey] = converted
                } else {
                    cleanRecord[cleanKey] = cleanValue
                }
            } else {
                cleanRecord[cleanKey] = cleanValue
            }
        }
        
        // 3. mapx: Filter and transform based on configuration
        if requiredFields, exists := dt.config["required_fields"].([]string); exists {
            for _, field := range requiredFields {
                if !mapx.HasKey(cleanRecord, field) {
                    return nil, errorutils.ValidationFailed("mapx", "required_field", field, "missing required field")
                }
            }
        }
        
        results = append(results, cleanRecord)
    }
    
    // 4. slicex: Apply final transformations to the result set
    if sortField, exists := dt.config["sort_by"].(string); exists {
        slicex.SortBy(results, func(a, b map[string]interface{}) bool {
            aVal := fmt.Sprintf("%v", a[sortField])
            bVal := fmt.Sprintf("%v", b[sortField])
            return aVal < bVal
        })
    }
    
    return results, nil
}
```

### Pattern 4: Error Recovery and Fallback

Demonstrates graceful error handling across modules:

```go
package integration

import (
    "github.com/msto63/mDW/foundation/pkg/utils/stringx"
    "github.com/msto63/mDW/foundation/pkg/utils/mathx"
    mdwerror "github.com/msto63/mDW/foundation/pkg/core/error"
    errorutils "github.com/msto63/mDW/foundation/pkg/core/errors"
)

type RobustProcessor struct {
    fallbackValues map[string]interface{}
    maxRetries     int
}

func NewRobustProcessor(fallbacks map[string]interface{}, retries int) *RobustProcessor {
    return &RobustProcessor{
        fallbackValues: fallbacks,
        maxRetries:     retries,
    }
}

func (rp *RobustProcessor) ProcessWithFallback(input string) (interface{}, error) {
    var lastErr error
    
    for attempt := 0; attempt < rp.maxRetries; attempt++ {
        result, err := rp.attemptProcessing(input)
        if err == nil {
            return result, nil
        }
        
        lastErr = err
        
        // Apply recovery strategies based on error type
        if mdwErr, ok := err.(*mdwerror.Error); ok {
            switch mdwErr.Code() {
            case "STRINGX_INVALID_FORMAT":
                // Try to clean and retry
                input = stringx.RemoveNonAlphanumeric(input)
                continue
                
            case "MATHX_INVALID_DECIMAL":
                // Try fallback value
                if fallback, exists := rp.fallbackValues["default_amount"]; exists {
                    return fallback, nil
                }
                
            default:
                // For unknown errors, log and continue with next attempt
                continue
            }
        }
    }
    
    return nil, errorutils.OperationFailed("robust_processor", "process_with_fallback", lastErr)
}

func (rp *RobustProcessor) attemptProcessing(input string) (interface{}, error) {
    // 1. stringx: Basic validation and cleaning
    cleaned := stringx.Trim(input)
    if stringx.IsEmpty(cleaned) {
        return nil, errorutils.ValidationFailed("stringx", "input", input, "empty input")
    }
    
    // 2. Try different parsing strategies
    if stringx.IsNumeric(cleaned) {
        // mathx: Attempt decimal parsing
        if decimal, err := mathx.ParseDecimal(cleaned); err == nil {
            return decimal, nil
        } else {
            return nil, errorutils.OperationFailed("mathx", "parse_decimal", err)
        }
    }
    
    // 3. Fallback to string processing
    if stringx.IsValidEmail(cleaned) {
        return map[string]string{"email": cleaned}, nil
    }
    
    return cleaned, nil
}
```

## Integration Testing Strategies

### Strategy 1: End-to-End Workflow Testing

```go
func TestFinancialWorkflowIntegration(t *testing.T) {
    // Test complete workflow from raw input to final output
    processor, err := NewFinancialProcessor("0.08", "USD")
    require.NoError(t, err)
    
    lineItems := []string{
        " 123.45 ",
        "67.89",
        " 234.56 ",
    }
    
    total, err := processor.CalculateInvoiceTotal(lineItems)
    require.NoError(t, err)
    
    // Verify the complete chain: stringx -> mathx -> stringx
    expected := "   459.85" // formatted with padding
    assert.Equal(t, expected, total)
}
```

### Strategy 2: Error Propagation Testing

```go
func TestErrorPropagationAcrossModules(t *testing.T) {
    pipeline := NewValidationPipeline(true)
    
    invalidInput := map[string]string{
        "email": "invalid-email",
        "name":  "X", // too short
        "amount": "not-a-number",
    }
    
    err := pipeline.ValidateUserInput(invalidInput)
    require.Error(t, err)
    
    // Verify error contains module context
    assert.Contains(t, err.Error(), "stringx")
    
    // Verify error details are preserved
    if mdwErr, ok := err.(*mdwerror.Error); ok {
        details := mdwErr.Details()
        assert.NotNil(t, details)
        assert.Contains(t, details, "module")
    }
}
```

## Best Practices for Module Integration

### 1. Error Handling Guidelines

- **Always preserve context**: Use `errorutils.OperationFailed()` to wrap errors from other modules
- **Use appropriate severity**: Match error severity to business impact
- **Include relevant details**: Add module, operation, and contextual information

### 2. Data Flow Guidelines

- **Validate early**: Use stringx for input cleaning before processing
- **Transform appropriately**: Convert data types as close to usage as possible
- **Handle edge cases**: Plan for empty inputs, null values, and boundary conditions

### 3. Performance Considerations

- **Minimize allocations**: Reuse objects where possible
- **Batch operations**: Group similar operations for efficiency
- **Use appropriate data structures**: Choose the right utility for the task

### 4. Testing Integration Points

- **Test error boundaries**: Verify error handling between modules
- **Test data transformations**: Ensure data integrity through the pipeline
- **Test performance**: Verify integrated workflows meet performance requirements

## Common Anti-Patterns to Avoid

### 1. Tight Coupling
```go
// AVOID: Direct dependency on specific implementation
func BadExample(input string) {
    // Directly calling stringx internals
    result := stringx.privateFunction(input) // Don't do this
}

// PREFER: Use public APIs
func GoodExample(input string) {
    result := stringx.Trim(input)
}
```

### 2. Error Information Loss
```go
// AVOID: Losing error context
func BadErrorHandling(input string) error {
    _, err := mathx.ParseDecimal(input)
    if err != nil {
        return fmt.Errorf("parsing failed") // Context lost
    }
    return nil
}

// PREFER: Preserve error context
func GoodErrorHandling(input string) error {
    _, err := mathx.ParseDecimal(input)
    if err != nil {
        return errorutils.OperationFailed("processor", "parse", err)
    }
    return nil
}
```

### 3. Inconsistent Error Patterns
```go
// AVOID: Mixed error patterns
func InconsistentErrors() error {
    if someCondition {
        return errors.New("standard error") // Inconsistent
    }
    return errorutils.ValidationFailed("module", "field", value, "reason") // mDW pattern
}

// PREFER: Consistent mDW error patterns
func ConsistentErrors() error {
    if someCondition {
        return errorutils.ValidationFailed("module", "condition", someCondition, "failed check")
    }
    return nil
}
```

## Conclusion

These integration patterns provide a foundation for building robust, maintainable applications using mDW Foundation modules. By following these patterns and best practices, you can create solutions that are:

- **Reliable**: Consistent error handling and data validation
- **Maintainable**: Clear separation of concerns and predictable interfaces
- **Testable**: Well-defined integration points and error boundaries
- **Performant**: Efficient data flow and minimal overhead

For more specific integration examples, see the integration tests in `test/integration/` directory.