// Package validationx implements comprehensive input validation utilities for the mDW platform.
//
// Package: validationx
// Title: Extended Input Validation for Go
// Description: This package provides a comprehensive collection of validation functions
//              for input validation, data integrity checking, format validation, and
//              business rule enforcement. Built on the mDW core validation framework,
//              it provides concrete validators with consistent error handling and 
//              integration patterns for enterprise applications.
// Author: msto63 with Claude Sonnet 4.0
// Version: v0.2.0
// Created: 2025-01-25
// Modified: 2025-01-26
//
// Change History:
// - 2025-01-25 v0.1.0: Initial implementation with comprehensive validation utilities
// - 2025-01-26 v0.1.1: Enhanced documentation with comprehensive examples and mDW integration
// - 2025-01-26 v0.2.0: Refactored to use core validation framework with standardized error codes
//
// Package Overview:
//
// The validationx package provides over 40 validation functions organized into logical categories.
// It is built on top of the mDW core validation framework (pkg/core/validation) and provides
// concrete validator implementations that use standardized error codes and result types.
//
// # Core Framework Integration
//
// This package uses the mDW core validation framework for consistency:
//   - All validators implement validation.ValidatorFunc
//   - Uses standardized error codes (validation.CodeRequired, validation.CodeEmail, etc.)
//   - Returns validation.ValidationResult with rich error information
//   - Integrates with validation.ValidatorChain for composition
//   - Supports context-aware validation through the core framework
//
// Type aliases are provided for backwards compatibility:
//   - ValidationResult = validation.ValidationResult
//   - ValidationError = validation.ValidationError  
//   - ValidatorChain = validation.ValidatorChain
//
// # Validation Categories
//
// # Basic Validation Functions
//
// Core validation functions for common requirements:
//   - Required: Validates that a value is not empty/nil/zero
//   - Optional: Wraps validators to only run on non-empty values
//
// # String Validation Functions
//
// Comprehensive string validation capabilities:
//   - MinLength/MaxLength/Length: String length validation
//   - Contains/StartsWith/EndsWith: Substring validation
//   - AlphaOnly/AlphaNumeric/NumericOnly: Character type validation
//
// # Pattern Validation Functions
//
// Format validation using patterns and standards:
//   - Pattern: Custom regular expression validation
//   - Email: RFC-compliant email address validation
//   - URL: Valid URL format validation
//   - IP/IPv4/IPv6: IP address validation
//   - UUID: UUID format validation (versions 1-5)
//
// # Numeric Validation Functions
//
// Number validation and range checking:
//   - IsNumber/IsInteger: Type validation
//   - Min/Max/Range: Value range validation
//   - Support for various numeric types (int, float, string numbers)
//
// # Date/Time Validation Functions
//
// Temporal data validation:
//   - IsDate: Date format validation
//   - DateAfter/DateBefore: Date comparison validation
//   - Multiple date format support
//
// # Collection Validation Functions
//
// Value set validation:
//   - In: Validates value is in allowed set
//   - NotIn: Validates value is not in forbidden set
//
// # Business Validation Functions
//
// Common business data validation:
//   - CreditCard: Credit card number validation with Luhn algorithm
//   - Phone: Phone number format validation
//   - Extensible for custom business rules
//
// # Validator Chains
//
// Powerful composition system for combining validators:
//   - ValidatorChain: Compose multiple validators
//   - Field-specific error reporting
//   - Short-circuit evaluation on first failure
//
// # Custom Validation
//
// Extensible validation system:
//   - Custom: Create validators with custom logic
//   - Function-based validation
//   - Integration with existing validator chains
//
// # Error Handling and Reporting
//
// Comprehensive error reporting system:
//   - ValidationError: Detailed error information
//   - ValidationResult: Complete validation results
//   - Field-specific error messages
//   - Multiple error aggregation
//
// # Usage Examples
//
// Basic field validation:
//
//	// Simple validation
//	result := validationx.Required("value")
//	if !result.Valid {
//		fmt.Printf("Validation failed: %s\n", result.FirstError().Message)
//	}
//
//	// Email validation
//	result = validationx.Email("user@example.com")
//	if result.Valid {
//		fmt.Println("Valid email address")
//	}
//
// Validator chains for complex validation:
//
//	// Create a validator chain for user registration
//	emailChain := validationx.NewValidatorChain("email").
//		Add(validationx.Required).
//		Add(validationx.Email).
//		Add(validationx.MaxLength(100))
//
//	passwordChain := validationx.NewValidatorChain("password").
//		Add(validationx.Required).
//		Add(validationx.MinLength(8)).
//		Add(validationx.Pattern(`^(?=.*[a-z])(?=.*[A-Z])(?=.*\d).+$`)) // Must contain upper, lower, digit
//
//	ageChain := validationx.NewValidatorChain("age").
//		Add(validationx.Optional(validationx.Range(13, 120)))
//
//	// Validate individual fields
//	emailResult := emailChain.Validate("user@example.com")
//	passwordResult := passwordChain.Validate("SecurePass123")
//	ageResult := ageChain.Validate(25)
//
// Complete form validation:
//
//	// Define validation rules for a form
//	rules := map[string]*validationx.ValidatorChain{
//		"name": validationx.NewValidatorChain("name").
//			Add(validationx.Required).
//			Add(validationx.MinLength(2)).
//			Add(validationx.MaxLength(50)),
//		"email": validationx.NewValidatorChain("email").
//			Add(validationx.Required).
//			Add(validationx.Email),
//		"phone": validationx.NewValidatorChain("phone").
//			Add(validationx.Optional(validationx.Phone)),
//		"age": validationx.NewValidatorChain("age").
//			Add(validationx.Optional(validationx.Range(18, 100))),
//	}
//
//	// Form data
//	formData := map[string]interface{}{
//		"name":  "John Doe",
//		"email": "john@example.com",
//		"phone": "555-123-4567",
//		"age":   30,
//	}
//
//	// Validate all fields
//	result := validationx.Validate(formData, rules)
//	if !result.Valid {
//		for _, err := range result.Errors {
//			fmt.Printf("Field '%s': %s\n", err.Field, err.Message)
//		}
//	}
//
// Custom validation:
//
//	// Create a custom validator for business logic
//	uniqueUsername := validationx.Custom(func(value interface{}) (bool, string) {
//		username, ok := value.(string)
//		if !ok {
//			return false, "username must be a string"
//		}
//
//		// Check against database or service
//		if isUsernameTaken(username) {
//			return false, "username is already taken"
//		}
//
//		return true, ""
//	})
//
//	// Use in validator chain
//	usernameChain := validationx.NewValidatorChain("username").
//		Add(validationx.Required).
//		Add(validationx.MinLength(3)).
//		Add(validationx.AlphaNumeric).
//		Add(uniqueUsername)
//
// Optional field validation:
//
//	// Optional fields that validate only when present
//	optionalUrl := validationx.NewValidatorChain("website").
//		Add(validationx.Optional(validationx.URL))
//
//	optionalPhone := validationx.NewValidatorChain("phone").
//		Add(validationx.Optional(validationx.Phone))
//
//	// These will pass validation for empty/nil values
//	result1 := optionalUrl.Validate("")      // Valid (empty)
//	result2 := optionalUrl.Validate(nil)     // Valid (nil)
//	result3 := optionalUrl.Validate("https://example.com") // Valid (proper URL)
//	result4 := optionalUrl.Validate("not-url")             // Invalid (malformed URL)
//
// Business validation examples:
//
//	// Credit card validation
//	ccValidator := validationx.NewValidatorChain("credit_card").
//		Add(validationx.Required).
//		Add(validationx.CreditCard)
//
//	// Phone number validation
//	phoneValidator := validationx.NewValidatorChain("phone").
//		Add(validationx.Required).
//		Add(validationx.Phone)
//
//	// UUID validation
//	idValidator := validationx.NewValidatorChain("id").
//		Add(validationx.Required).
//		Add(validationx.UUID)
//
// Convenience functions for quick validation:
//
//	// Quick validation without detailed error information
//	isValid := validationx.IsValidEmail("user@example.com")
//	isValid = validationx.IsValidURL("https://example.com")
//	isValid = validationx.IsValidIP("192.168.1.1")
//	isValid = validationx.IsValidUUID("550e8400-e29b-41d4-a716-446655440000")
//	isValid = validationx.IsValidCreditCard("4532015112830366")
//	isValid = validationx.IsValidPhone("555-123-4567")
//
// Error handling and reporting:
//
//	result := validationx.Validate(formData, rules)
//	if !result.Valid {
//		// Get all error messages
//		messages := result.ErrorMessages()
//		
//		// Get first error
//		firstError := result.FirstError()
//		if firstError != nil {
//			fmt.Printf("First error in field '%s': %s\n", firstError.Field, firstError.Message)
//		}
//		
//		// Process each error
//		for _, err := range result.Errors {
//			fmt.Printf("Field: %s, Rule: %s, Value: %v, Message: %s\n",
//				err.Field, err.Rule, err.Value, err.Message)
//		}
//	}
//
// # Validation Types
//
// The package supports validation of various data types:
//
// String validation:
//   - Length constraints (min, max, exact)
//   - Content validation (alpha, numeric, alphanumeric)
//   - Pattern matching (regex)
//   - Format validation (email, URL, UUID)
//
// Numeric validation:
//   - Type checking (number, integer)
//   - Range validation (min, max, range)
//   - Support for int, float, and string numbers
//
// Date/Time validation:
//   - Format validation (multiple formats supported)
//   - Temporal comparisons (before, after)
//   - Support for time.Time and string dates
//
// Collection validation:
//   - Membership testing (in, not in)
//   - Custom set validation
//
// Business data validation:
//   - Credit card numbers (with Luhn algorithm)
//   - Phone numbers (basic format validation)
//   - IP addresses (IPv4 and IPv6)
//   - UUIDs (versions 1-5)
//
// # Advanced Features
//
// Validator Composition:
//   - Chain multiple validators together
//   - Short-circuit evaluation
//   - Field-specific error reporting
//   - Optional validation (skip when empty)
//
// Custom Validation:
//   - Function-based custom validators
//   - Integration with business logic
//   - Database validation support
//   - Complex rule composition
//
// Error Management:
//   - Detailed error information
//   - Multiple error aggregation
//   - Field-specific error mapping
//   - Human-readable error messages
//
// # Performance Characteristics
//
// All validation functions are optimized for performance:
//   - Minimal memory allocations
//   - Early termination on validation failure
//   - Efficient regular expression compilation
//   - Type-safe validation without reflection overhead
//
// # Thread Safety
//
// All validation functions are thread-safe and can be used concurrently.
// Validator chains are immutable after creation and can be safely shared
// across goroutines.
//
// # Integration with mDW Platform
//
// This package is designed as part of the mDW (Trusted Business Platform)
// foundation library and follows mDW coding standards:
//   - Comprehensive documentation and examples
//   - Extensive test coverage (>95%)
//   - Consistent error handling
//   - English-only code and comments
//
// The package provides the input validation capabilities needed for TCOL
// (Terminal Command Object Language) processing, user input validation,
// API request validation, and general data integrity checking in enterprise
// applications.
//
// # Error Types and Framework Integration
//
// This package uses the mDW core validation framework types:
//
//	// From pkg/core/validation
//	type ValidationError struct {
//		Code     string                 // Standardized error code
//		Field    string                 // Field name that failed validation
//		Message  string                 // Human-readable error message
//		Value    interface{}           // Value that failed validation
//		Context  map[string]interface{} // Additional error context
//		Expected interface{}           // Expected value or format
//	}
//
//	type ValidationResult struct {
//		Valid   bool                    // Whether validation passed
//		Errors  []ValidationError       // List of validation errors
//		Context map[string]interface{}  // Additional validation context
//	}
//
// Standardized error codes used by validators:
//   - validation.CodeRequired: Field is required but missing
//   - validation.CodeEmail: Invalid email format
//   - validation.CodeURL: Invalid URL format
//   - validation.CodePhoneNumber: Invalid phone number format
//   - validation.CodeLength: String/array length validation failed
//   - validation.CodeRange: Numeric range validation failed
//   - validation.CodeType: Type validation failed
//   - validation.CodePattern: Pattern/regex validation failed
//   - validation.CodeFormat: General format validation failed
//   - validation.CodeCustom: Custom validation rule failed
//
// # Common Use Cases
//
// 1. API Request Validation
//
//	// Define API request validation
//	createCustomerRules := map[string]*validationx.ValidatorChain{
//		"name": validationx.NewValidatorChain("name").
//			Add(validationx.Required).
//			Add(validationx.MinLength(2)).
//			Add(validationx.MaxLength(100)),
//		"email": validationx.NewValidatorChain("email").
//			Add(validationx.Required).
//			Add(validationx.Email),
//		"phone": validationx.NewValidatorChain("phone").
//			Add(validationx.Optional(validationx.Phone)),
//		"type": validationx.NewValidatorChain("type").
//			Add(validationx.Required).
//			Add(validationx.In([]string{"individual", "business"})),
//		"vatNumber": validationx.NewValidatorChain("vatNumber").
//			Add(validationx.Optional(validationx.Pattern(`^[A-Z]{2}\d{9}$`))),
//	}
//	
//	// Validate request
//	result := validationx.Validate(requestData, createCustomerRules)
//	if !result.Valid {
//		return NewBadRequestError(result.ErrorMessages())
//	}
//
// 2. Configuration Validation
//
//	// Validate application configuration
//	configRules := map[string]*validationx.ValidatorChain{
//		"port": validationx.NewValidatorChain("port").
//			Add(validationx.Required).
//			Add(validationx.Range(1, 65535)),
//		"host": validationx.NewValidatorChain("host").
//			Add(validationx.Required).
//			Add(validationx.Pattern(`^[\w\-\.]+$`)),
//		"database.url": validationx.NewValidatorChain("database.url").
//			Add(validationx.Required).
//			Add(validationx.URL),
//		"database.pool_size": validationx.NewValidatorChain("database.pool_size").
//			Add(validationx.Optional(validationx.Range(1, 100))),
//		"api.key": validationx.NewValidatorChain("api.key").
//			Add(validationx.Required).
//			Add(validationx.MinLength(32)),
//	}
//	
//	// Validate config
//	result := validationx.Validate(config, configRules)
//	if !result.Valid {
//		log.Fatal("Invalid configuration", result.Errors)
//	}
//
// 3. Form Processing
//
//	// Multi-step form validation
//	step1Rules := map[string]*validationx.ValidatorChain{
//		"firstName": validationx.NewValidatorChain("firstName").
//			Add(validationx.Required).
//			Add(validationx.AlphaOnly),
//		"lastName": validationx.NewValidatorChain("lastName").
//			Add(validationx.Required).
//			Add(validationx.AlphaOnly),
//		"dateOfBirth": validationx.NewValidatorChain("dateOfBirth").
//			Add(validationx.Required).
//			Add(validationx.IsDate).
//			Add(validationx.DateBefore(time.Now().AddDate(-18, 0, 0))), // 18 years old
//	}
//	
//	step2Rules := map[string]*validationx.ValidatorChain{
//		"address": validationx.NewValidatorChain("address").
//			Add(validationx.Required).
//			Add(validationx.MinLength(10)),
//		"city": validationx.NewValidatorChain("city").
//			Add(validationx.Required),
//		"zipCode": validationx.NewValidatorChain("zipCode").
//			Add(validationx.Required).
//			Add(validationx.Pattern(`^\d{5}(-\d{4})?$`)),
//	}
//
// 4. Business Rule Validation
//
//	// Complex business validation
//	orderValidator := validationx.Custom(func(value interface{}) (bool, string) {
//		order, ok := value.(*Order)
//		if !ok {
//			return false, "invalid order type"
//		}
//		
//		// Validate business rules
//		if order.TotalAmount < 0 {
//			return false, "order total cannot be negative"
//		}
//		
//		if len(order.Items) == 0 {
//			return false, "order must contain at least one item"
//		}
//		
//		// Check inventory
//		for _, item := range order.Items {
//			if !checkInventory(item.ProductID, item.Quantity) {
//				return false, fmt.Sprintf("insufficient inventory for product %s", item.ProductID)
//			}
//		}
//		
//		return true, ""
//	})
//
// 5. Data Import Validation
//
//	// CSV import validation
//	csvRowRules := map[string]*validationx.ValidatorChain{
//		"product_code": validationx.NewValidatorChain("product_code").
//			Add(validationx.Required).
//			Add(validationx.Pattern(`^[A-Z]{3}-\d{4}$`)),
//		"price": validationx.NewValidatorChain("price").
//			Add(validationx.Required).
//			Add(validationx.IsNumber).
//			Add(validationx.Min(0)),
//		"quantity": validationx.NewValidatorChain("quantity").
//			Add(validationx.Required).
//			Add(validationx.IsInteger).
//			Add(validationx.Min(0)),
//		"category": validationx.NewValidatorChain("category").
//			Add(validationx.Required).
//			Add(validationx.In(validCategories)),
//	}
//	
//	// Process each row
//	for i, row := range csvRows {
//		result := validationx.Validate(row, csvRowRules)
//		if !result.Valid {
//			importErrors = append(importErrors, fmt.Sprintf("Row %d: %s", i+1, result.FirstError().Message))
//		}
//	}
//
// # Best Practices
//
// 1. Reuse validator chains for consistent validation
// 2. Use Optional() for fields that may be empty
// 3. Provide clear, user-friendly error messages
// 4. Validate early and fail fast
// 5. Use custom validators for complex business rules
// 6. Consider performance for large datasets
//
// # mDW Integration Examples
//
// 1. TCOL Command Validation
//
//	// Validate TCOL command parameters
//	tcolParamRules := map[string]*validationx.ValidatorChain{
//		"object": validationx.NewValidatorChain("object").
//			Add(validationx.Required).
//			Add(validationx.In([]string{"CUSTOMER", "INVOICE", "ORDER", "REPORT"})),
//		"method": validationx.NewValidatorChain("method").
//			Add(validationx.Required).
//			Add(validationx.Pattern(`^[A-Z][A-Z\-]*$`)),
//		"id": validationx.NewValidatorChain("id").
//			Add(validationx.Optional(validationx.UUID)),
//		"filter": validationx.NewValidatorChain("filter").
//			Add(validationx.Optional(validationx.MaxLength(500))),
//	}
//	
//	// Validate command
//	result := validationx.Validate(tcolParams, tcolParamRules)
//	if !result.Valid {
//		return NewTCOLError("INVALID_PARAMETERS", result.ErrorMessages())
//	}
//
// 2. Permission Validation
//
//	// Validate permission strings
//	permissionValidator := validationx.NewValidatorChain("permission").
//		Add(validationx.Required).
//		Add(validationx.Pattern(`^[A-Z]+:[A-Z]+:(all|team|self)$`)).
//		Add(validationx.Custom(func(value interface{}) (bool, string) {
//			perm := value.(string)
//			parts := strings.Split(perm, ":")
//			
//			// Validate object exists
//			if !isValidObject(parts[0]) {
//				return false, fmt.Sprintf("invalid object: %s", parts[0])
//			}
//			
//			// Validate method for object
//			if !isValidMethod(parts[0], parts[1]) {
//				return false, fmt.Sprintf("invalid method %s for object %s", parts[1], parts[0])
//			}
//			
//			return true, ""
//		}))
//
// 3. Audit Log Validation
//
//	// Validate audit log entries
//	auditEntryRules := map[string]*validationx.ValidatorChain{
//		"timestamp": validationx.NewValidatorChain("timestamp").
//			Add(validationx.Required).
//			Add(validationx.IsDate),
//		"user_id": validationx.NewValidatorChain("user_id").
//			Add(validationx.Required).
//			Add(validationx.UUID),
//		"action": validationx.NewValidatorChain("action").
//			Add(validationx.Required).
//			Add(validationx.Pattern(`^[A-Z]+\.[A-Z]+$`)),
//		"ip_address": validationx.NewValidatorChain("ip_address").
//			Add(validationx.Required).
//			Add(validationx.IP),
//		"result": validationx.NewValidatorChain("result").
//			Add(validationx.Required).
//			Add(validationx.In([]string{"SUCCESS", "FAILURE", "ERROR"})),
//	}
//
// 4. Business Object Validation
//
//	// Invoice validation with business rules
//	invoiceRules := map[string]*validationx.ValidatorChain{
//		"invoice_number": validationx.NewValidatorChain("invoice_number").
//			Add(validationx.Required).
//			Add(validationx.Pattern(`^INV-\d{4}-\d{6}$`)),
//		"customer_id": validationx.NewValidatorChain("customer_id").
//			Add(validationx.Required).
//			Add(validationx.UUID),
//		"issue_date": validationx.NewValidatorChain("issue_date").
//			Add(validationx.Required).
//			Add(validationx.IsDate),
//		"due_date": validationx.NewValidatorChain("due_date").
//			Add(validationx.Required).
//			Add(validationx.IsDate).
//			Add(validationx.Custom(func(value interface{}) (bool, string) {
//				// Due date must be after issue date
//				dueDate := value.(time.Time)
//				if dueDate.Before(invoice.IssueDate) {
//					return false, "due date must be after issue date"
//				}
//				return true, ""
//			})),
//		"total_amount": validationx.NewValidatorChain("total_amount").
//			Add(validationx.Required).
//			Add(validationx.Min(0.01)),
//		"status": validationx.NewValidatorChain("status").
//			Add(validationx.Required).
//			Add(validationx.In([]string{"DRAFT", "SENT", "PAID", "OVERDUE", "CANCELLED"})),
//	}
//
// # Performance Considerations
//
// 1. Validator Chain Caching
//   - Create validator chains once and reuse them
//   - Store chains as package-level variables for frequently used validations
//   - Avoid creating chains in hot paths
//
// 2. Large Dataset Validation
//   - Use goroutines for parallel validation of independent items
//   - Implement streaming validation for very large files
//   - Consider batch validation with progress reporting
//
// 3. Optimization Tips
//   - Order validators from most to least likely to fail
//   - Use simple validators before complex ones
//   - Cache regex patterns for repeated use
//
// # Error Handling
//
// The package follows Go best practices for error handling:
//   - Descriptive error messages with context
//   - Field-specific error reporting
//   - Aggregated error results
//   - No panics from normal usage
//
// # Thread Safety
//
// All validation functions are thread-safe and can be used concurrently.
// Validator chains are immutable after creation and can be safely shared
// across goroutines.
//
// # Related Packages
//
//   - core/validation: Core validation framework (REQUIRED DEPENDENCY)
//   - core/error: Error handling for validation failures
//   - core/log: Logging validation events
//   - utils/stringx: String manipulation for validation  
//   - utils/mathx: Mathematical utilities for numeric validation
//
// # Extensibility
//
// The package is designed for extensibility:
//   - Custom validators can be easily created
//   - Validator chains support composition
//   - Business-specific validation rules can be added
//   - Integration with external validation services
//
// This makes it suitable for complex enterprise applications where
// validation requirements may vary significantly between different
// domains and use cases.
package validationx