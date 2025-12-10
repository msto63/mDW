# mDW Foundation Validation System Architecture

## Overview

The mDW Foundation implements a dual-module validation system with clear separation of concerns between framework infrastructure and concrete implementations. This document explains the architecture, boundaries, and proper usage patterns.

## Architecture Components

### 1. Core Validation Framework (`pkg/core/validation`)

**Purpose**: Provides validation infrastructure, interfaces, types, and framework utilities.

**Contains**:
- `Validator` interface - Standard interface for all validators
- `ValidatorFunc` type - Function-based validator implementation
- `ValidationResult` type - Structured validation results
- `ValidationError` type - Rich error information
- Standardized error codes (`CodeRequired`, `CodeEmail`, etc.)
- Framework utilities (`GetValueLength`, `ConvertToFloat64`, etc.)
- Result composition functions (`Combine`, `NewValidationResult`, etc.)

**Does NOT contain**:
- Concrete validator implementations
- Business logic validation
- Format-specific validation logic

### 2. Concrete Validators (`pkg/utils/validationx`)

**Purpose**: Implements actual validation logic using the core framework.

**Contains**:
- `ValidateStruct` - Struct field validation with tags
- `IsValidEmail`, `IsValidURL`, `IsValidIP` - Format validators
- `MinLength`, `MaxLength`, `Range` - Constraint validators
- `Pattern`, `Contains`, `StartsWith` - String validators
- `ValidatorChain` - Chain composition utility
- `Custom` - Custom validation function wrapper

**Uses**:
- Core validation types for all results
- Framework interfaces for validator composition
- Standardized error codes from core module

## Clear Boundaries

### What Goes Where

| Component | Core Validation | Utils Validationx |
|-----------|----------------|-------------------|
| Interfaces | ✅ Validator interface | ❌ No interfaces |
| Types | ✅ ValidationResult, ValidationError | ❌ No custom types |
| Error Codes | ✅ All standard codes | ❌ Uses core codes |
| Utilities | ✅ Framework utilities | ❌ Uses framework utils |
| Concrete Validators | ❌ No implementations | ✅ All validators |
| Business Logic | ❌ No business rules | ✅ Domain-specific validation |
| Chain Composition | ✅ Framework patterns | ✅ Implementation helpers |

### Import Patterns

#### Correct Usage

```go
import (
    // Core framework for types and interfaces
    "github.com/msto63/mDW/foundation/pkg/core/validation"
    
    // Concrete validators for actual validation
    "github.com/msto63/mDW/foundation/pkg/utils/validationx"
)

// Use core types for structure
func ValidateUser(user User) validation.ValidationResult {
    result := validation.NewValidationResult()
    
    // Use concrete validators for logic
    if !validationx.IsValidEmail(user.Email) {
        emailError := validation.NewValidationErrorWithField(
            validation.CodeEmail, "email", "invalid email", user.Email)
        result = validation.Combine(result, emailError)
    }
    
    return result
}
```

#### Incorrect Usage

```go
// DON'T import validation for concrete validators
import validation "pkg/core/validation"
result := validation.ValidateEmail(email) // This doesn't exist!

// DON'T create custom result types in validationx
type CustomResult struct { ... } // Should use validation.ValidationResult
```

## Integration Patterns

### 1. Framework-Based Custom Validators

```go
// Create validators that implement the framework interface
type EmailValidator struct{}

func (v EmailValidator) Validate(value interface{}) validation.ValidationResult {
    str, ok := value.(string)
    if !ok {
        return validation.NewValidationError(validation.CodeType, "must be string")
    }
    
    if !validationx.IsValidEmail(str) {
        return validation.NewValidationError(validation.CodeEmail, "invalid email")
    }
    
    return validation.NewValidationResult()
}

func (v EmailValidator) ValidateWithContext(ctx context.Context, value interface{}) validation.ValidationResult {
    // Add context-aware logic here
    return v.Validate(value)
}
```

### 2. Struct Validation with Tags

```go
// Use validationx for tag-based validation
type User struct {
    Email string `validate:"required,email"`
    Age   int    `validate:"required,min:0,max:150"`
    Name  string `validate:"required,min_length:2"`
}

func ValidateUserStruct(user User) validation.ValidationResult {
    return validationx.ValidateStruct(user)
}
```

### 3. Complex Validation Chains

```go
// Combine framework patterns with concrete validators
func ValidateComplexData(data interface{}) validation.ValidationResult {
    // Use framework for orchestration
    chain := validationx.NewValidatorChain("complex-validation").
        Add(validation.ValidatorFunc(validateRequired)).
        Add(validation.ValidatorFunc(validateFormat)).
        Add(validation.ValidatorFunc(validateBusinessRules))
    
    return chain.Validate(data)
}
```

### 4. Error Handling Integration

```go
import (
    "github.com/msto63/mDW/foundation/pkg/core/error"
    "github.com/msto63/mDW/foundation/pkg/core/validation"
)

func HandleValidationResult(result validation.ValidationResult) error {
    if result.Valid {
        return nil
    }
    
    // Convert to mDW error system
    if err := result.ToError(); err != nil {
        return err
    }
    
    // Or create custom error
    return mdwerror.New("validation failed").
        WithCode(mdwerror.CodeValidationFailed).
        WithDetail("errors", result.ErrorMessages())
}
```

## Best Practices

### Do's

1. **Use core types for all validation results**
   ```go
   func MyValidator(value interface{}) validation.ValidationResult { ... }
   ```

2. **Use concrete validators for actual validation logic**
   ```go
   if !validationx.IsValidEmail(email) { ... }
   ```

3. **Compose validation using framework patterns**
   ```go
   result1 := validator1.Validate(value1)
   result2 := validator2.Validate(value2)
   combined := validation.Combine(result1, result2)
   ```

4. **Use standardized error codes**
   ```go
   validation.NewValidationError(validation.CodeEmail, "invalid email")
   ```

### Don'ts

1. **Don't create validation types in validationx**
   ```go
   // DON'T - types belong in core/validation
   type MyValidationResult struct { ... }
   ```

2. **Don't implement concrete validators in core/validation**
   ```go
   // DON'T - concrete logic belongs in utils/validationx
   func (core) ValidateEmail(email string) bool { ... }
   ```

3. **Don't bypass the framework types**
   ```go
   // DON'T - use validation.ValidationResult instead
   func Validate(value interface{}) (bool, []string) { ... }
   ```

4. **Don't duplicate error codes**
   ```go
   // DON'T - use validation.CodeEmail
   const MyEmailCode = "EMAIL_INVALID"
   ```

## Migration Guide

If you have existing code that doesn't follow these boundaries:

### Step 1: Update Imports

```go
// Old
import "pkg/some/validation"

// New
import (
    "pkg/core/validation"           // For types and interfaces
    "pkg/utils/validationx"         // For concrete validators
)
```

### Step 2: Update Function Signatures

```go
// Old
func Validate(value interface{}) (bool, error)

// New
func Validate(value interface{}) validation.ValidationResult
```

### Step 3: Update Error Handling

```go
// Old
if err := validateEmail(email); err != nil {
    return fmt.Errorf("validation failed: %w", err)
}

// New
result := validationx.ValidateStruct(data)
if !result.Valid {
    return result.ToError()
}
```

### Step 4: Update Validator Creation

```go
// Old - custom result types
type MyResult struct { Valid bool; Message string }

// New - use framework types
func MyValidator(value interface{}) validation.ValidationResult {
    if /* validation logic */ {
        return validation.NewValidationResult()
    }
    return validation.NewValidationError(validation.CodeCustom, "validation failed")
}
```

## Testing Validation Systems

### Testing Framework Components

```go
func TestValidationResult(t *testing.T) {
    result := validation.NewValidationResult()
    assert.True(t, result.Valid)
    
    errorResult := validation.NewValidationError(validation.CodeRequired, "required")
    assert.False(t, errorResult.Valid)
    assert.Len(t, errorResult.Errors, 1)
}
```

### Testing Concrete Validators

```go
func TestEmailValidation(t *testing.T) {
    tests := []struct {
        email string
        valid bool
    }{
        {"test@example.com", true},
        {"invalid-email", false},
    }
    
    for _, tt := range tests {
        result := validationx.IsValidEmail(tt.email)
        assert.Equal(t, tt.valid, result)
    }
}
```

### Testing Integration

```go
func TestCompleteValidation(t *testing.T) {
    user := User{Email: "test@example.com", Age: 25}
    result := ValidateUser(user)
    
    assert.True(t, result.Valid)
    assert.Empty(t, result.Errors)
}
```

## Performance Considerations

### Framework Overhead

- ValidationResult creation: ~20-30 ns/op
- Error result creation: ~30-50 ns/op
- Result combination: ~10-20 ns/op per result
- Context processing: ~10 ns/op overhead

### Optimization Tips

1. **Reuse validator chains**
   ```go
   var userChain = validationx.NewValidatorChain("user").
       Add(emailValidator).
       Add(ageValidator)
   ```

2. **Cache compiled regexes**
   ```go
   // validationx already does this internally
   ```

3. **Use struct validation for batch operations**
   ```go
   // More efficient than individual field validation
   result := validationx.ValidateStruct(data)
   ```

## Future Extensions

The validation system is designed for extension:

### Adding New Validators

1. Implement in `pkg/utils/validationx`
2. Use core validation types for results
3. Follow existing naming conventions
4. Add comprehensive tests

### Adding New Error Codes

1. Add constants to `pkg/core/validation/interfaces.go`
2. Document in the error code reference
3. Use in concrete validators

### Creating Domain-Specific Validators

1. Create separate packages (e.g., `pkg/business/validation`)
2. Import and use both core and utils validation modules
3. Implement domain-specific logic while following the same patterns

## Conclusion

The mDW Foundation validation system provides a clean separation between framework infrastructure and concrete implementation. By following these boundaries and patterns, you can build robust, maintainable validation systems that integrate seamlessly with the rest of the mDW Foundation.

For examples of proper usage, see `examples/integration_examples.go` and the test files in both validation modules.