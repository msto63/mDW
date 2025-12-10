// File: validationx.go
// Title: Core Validation Utilities
// Description: Implements comprehensive input validation functions including
//              string validation, format validation, business rule validation,
//              and custom validator chains for the mDW platform.
// Author: msto63 with Claude Sonnet 4.0
// Version: v0.1.0
// Created: 2025-01-25
// Modified: 2025-01-25
//
// Change History:
// - 2025-01-25 v0.1.0: Initial implementation with comprehensive validation utilities

package validationx

import (
	"fmt"
	"net"
	"net/mail"
	"net/url"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/msto63/mDW/foundation/core/validation"
)

// Regex cache for compiled patterns to avoid recompilation
var (
	regexCache = make(map[string]*regexp.Regexp)
	regexMu    sync.RWMutex
)

// getCompiledRegex returns a cached compiled regex or compiles and caches it
func getCompiledRegex(pattern string) (*regexp.Regexp, error) {
	regexMu.RLock()
	if regex, exists := regexCache[pattern]; exists {
		regexMu.RUnlock()
		return regex, nil
	}
	regexMu.RUnlock()
	
	// Compile and cache
	regex, err := regexp.Compile(pattern)
	if err != nil {
		return nil, err
	}
	
	regexMu.Lock()
	regexCache[pattern] = regex
	regexMu.Unlock()
	
	return regex, nil
}

// Type aliases for backwards compatibility and convenience
type (
	// ValidationResult is an alias to the core validation result type
	ValidationResult = validation.ValidationResult
	// ValidationError is an alias to the core validation error type
	ValidationError = validation.ValidationError
	// ValidatorChain is an alias to the core validator chain type
	ValidatorChain = validation.ValidatorChain
)

// NewValidatorChain creates a new validator chain using the core framework
func NewValidatorChain(name string) *ValidatorChain {
	return validation.NewValidatorChain(name)
}

// ===============================
// Basic Validation Functions
// ===============================

// Required validates that a value is not empty/nil/zero
var Required validation.ValidatorFunc = func(value interface{}) validation.ValidationResult {
	if validation.IsNilOrEmpty(value) {
		return validation.NewValidationError(validation.CodeRequired, "value is required")
	}
	
	switch v := value.(type) {
	case string:
		if strings.TrimSpace(v) == "" {
			return validation.NewValidationError(validation.CodeRequired, "value is required")
		}
	case []interface{}:
		if len(v) == 0 {
			return validation.NewValidationError(validation.CodeRequired, "value is required")
		}
	case map[string]interface{}:
		if len(v) == 0 {
			return validation.NewValidationError(validation.CodeRequired, "value is required")
		}
	}
	
	return validation.NewValidationResult()
}

// Optional creates a validator that only runs if the value is not empty
func Optional(validator validation.Validator) validation.ValidatorFunc {
	return func(value interface{}) validation.ValidationResult {
		// Check if value is empty using core framework utility
		if validation.IsNilOrEmpty(value) {
			return validation.NewValidationResult()
		}
		
		if str, ok := value.(string); ok && strings.TrimSpace(str) == "" {
			return validation.NewValidationResult()
		}
		
		return validator.Validate(value)
	}
}

// ===============================
// String Validation Functions
// ===============================

// MinLength validates minimum string length
func MinLength(min int) validation.ValidatorFunc {
	return func(value interface{}) validation.ValidationResult {
		str, ok := value.(string)
		if !ok {
			return validation.NewValidationError(validation.CodeType, "value must be a string")
		}
		
		if utf8.RuneCountInString(str) < min {
			return validation.NewValidationError(validation.CodeLength, fmt.Sprintf("must be at least %d characters long", min))
		}
		
		return validation.NewValidationResult()
	}
}

// MaxLength validates maximum string length
func MaxLength(max int) validation.ValidatorFunc {
	return func(value interface{}) validation.ValidationResult {
		str, ok := value.(string)
		if !ok {
			return validation.NewValidationError(validation.CodeType, "value must be a string")
		}
		
		if utf8.RuneCountInString(str) > max {
			return validation.NewValidationError(validation.CodeLength, fmt.Sprintf("must be at most %d characters long", max))
		}
		
		return validation.NewValidationResult()
	}
}

// Length validates exact string length
func Length(length int) validation.ValidatorFunc {
	return func(value interface{}) validation.ValidationResult {
		str, ok := value.(string)
		if !ok {
			return validation.NewValidationError(validation.CodeType, "value must be a string")
		}
		
		if utf8.RuneCountInString(str) != length {
			return validation.NewValidationError(validation.CodeLength, fmt.Sprintf("must be exactly %d characters long", length))
		}
		
		return validation.NewValidationResult()
	}
}

// Contains validates that string contains a substring
func Contains(substring string) validation.ValidatorFunc {
	return func(value interface{}) validation.ValidationResult {
		str, ok := value.(string)
		if !ok {
			return validation.NewValidationError(validation.CodeType, "value must be a string")
		}
		
		if !strings.Contains(str, substring) {
			return validation.NewValidationError(validation.CodePattern, fmt.Sprintf("must contain '%s'", substring))
		}
		
		return validation.NewValidationResult()
	}
}

// StartsWith validates that string starts with a prefix
func StartsWith(prefix string) validation.ValidatorFunc {
	return func(value interface{}) validation.ValidationResult {
		str, ok := value.(string)
		if !ok {
			return validation.NewValidationError(validation.CodeType, "value must be a string")
		}
		
		if !strings.HasPrefix(str, prefix) {
			return validation.NewValidationError(validation.CodePattern, fmt.Sprintf("must start with '%s'", prefix))
		}
		
		return validation.NewValidationResult()
	}
}

// EndsWith validates that string ends with a suffix
func EndsWith(suffix string) validation.ValidatorFunc {
	return func(value interface{}) validation.ValidationResult {
		str, ok := value.(string)
		if !ok {
			return validation.NewValidationError(validation.CodeType, "value must be a string")
		}
		
		if !strings.HasSuffix(str, suffix) {
			return validation.NewValidationError(validation.CodePattern, fmt.Sprintf("must end with '%s'", suffix))
		}
		
		return validation.NewValidationResult()
	}
}

// AlphaOnly validates that string contains only alphabetic characters
var AlphaOnly validation.ValidatorFunc = func(value interface{}) validation.ValidationResult {
	str, ok := value.(string)
	if !ok {
		return validation.NewValidationError(validation.CodeType, "value must be a string")
	}
	
	for _, r := range str {
		if !unicode.IsLetter(r) {
			return validation.NewValidationError(validation.CodePattern, "must contain only alphabetic characters")
		}
	}
	
	return validation.NewValidationResult()
}

// AlphaNumeric validates that string contains only alphanumeric characters
var AlphaNumeric validation.ValidatorFunc = func(value interface{}) validation.ValidationResult {
	str, ok := value.(string)
	if !ok {
		return validation.NewValidationError(validation.CodeType, "value must be a string")
	}
	
	for _, r := range str {
		if !unicode.IsLetter(r) && !unicode.IsNumber(r) {
			return validation.NewValidationError(validation.CodePattern, "must contain only alphanumeric characters")
		}
	}
	
	return validation.NewValidationResult()
}

// NumericOnly validates that string contains only numeric characters
var NumericOnly validation.ValidatorFunc = func(value interface{}) validation.ValidationResult {
	str, ok := value.(string)
	if !ok {
		return validation.NewValidationError(validation.CodeType, "value must be a string")
	}
	
	for _, r := range str {
		if !unicode.IsNumber(r) {
			return validation.NewValidationError(validation.CodePattern, "must contain only numeric characters")
		}
	}
	
	return validation.NewValidationResult()
}

// ===============================
// Pattern Validation Functions
// ===============================

// Pattern validates that string matches a regular expression
func Pattern(pattern string) validation.ValidatorFunc {
	return func(value interface{}) validation.ValidationResult {
		str, ok := value.(string)
		if !ok {
			return validation.NewValidationError(validation.CodeType, "value must be a string")
		}
		
		regex, err := getCompiledRegex(pattern)
		if err != nil {
			return validation.NewValidationError(validation.CodePattern, fmt.Sprintf("invalid pattern: %v", err))
		}
		
		if !regex.MatchString(str) {
			return validation.NewValidationError(validation.CodePattern, "does not match required pattern")
		}
		
		return validation.NewValidationResult()
	}
}

// Email validates email address format
var Email validation.ValidatorFunc = func(value interface{}) validation.ValidationResult {
	str, ok := value.(string)
	if !ok {
		return validation.NewValidationError(validation.CodeType, "value must be a string")
	}
	
	_, err := mail.ParseAddress(str)
	if err != nil {
		return validation.NewValidationError(validation.CodeEmail, "must be a valid email address")
	}
	
	return validation.NewValidationResult()
}

// URL validates URL format
var URL validation.ValidatorFunc = func(value interface{}) validation.ValidationResult {
	str, ok := value.(string)
	if !ok {
		return validation.NewValidationError(validation.CodeType, "value must be a string")
	}
	
	if strings.TrimSpace(str) == "" {
		return validation.NewValidationError(validation.CodeURL, "must be a valid URL")
	}
	
	parsedURL, err := url.ParseRequestURI(str)
	if err != nil || parsedURL.Scheme == "" || parsedURL.Host == "" {
		return validation.NewValidationError(validation.CodeURL, "must be a valid URL")
	}
	
	return validation.NewValidationResult()
}

// IP validates IP address format (IPv4 or IPv6)
var IP validation.ValidatorFunc = func(value interface{}) validation.ValidationResult {
	str, ok := value.(string)
	if !ok {
		return validation.NewValidationError(validation.CodeType, "value must be a string")
	}
	
	if net.ParseIP(str) == nil {
		return validation.NewValidationError(validation.CodeFormat, "must be a valid IP address")
	}
	
	return validation.NewValidationResult()
}

// IPv4 validates IPv4 address format
var IPv4 validation.ValidatorFunc = func(value interface{}) validation.ValidationResult {
	str, ok := value.(string)
	if !ok {
		return validation.NewValidationError(validation.CodeType, "value must be a string")
	}
	
	ip := net.ParseIP(str)
	if ip == nil || ip.To4() == nil {
		return validation.NewValidationError(validation.CodeFormat, "must be a valid IPv4 address")
	}
	
	return validation.NewValidationResult()
}

// IPv6 validates IPv6 address format
var IPv6 validation.ValidatorFunc = func(value interface{}) validation.ValidationResult {
	str, ok := value.(string)
	if !ok {
		return validation.NewValidationError(validation.CodeType, "value must be a string")
	}
	
	ip := net.ParseIP(str)
	if ip == nil || ip.To4() != nil {
		return validation.NewValidationError(validation.CodeFormat, "must be a valid IPv6 address")
	}
	
	return validation.NewValidationResult()
}

// UUID validates UUID format (versions 1-5)
var UUID validation.ValidatorFunc = func(value interface{}) validation.ValidationResult {
	str, ok := value.(string)
	if !ok {
		return validation.NewValidationError(validation.CodeType, "value must be a string")
	}
	
	// UUID regex pattern
	uuidPattern := `^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[1-5][0-9a-fA-F]{3}-[89abAB][0-9a-fA-F]{3}-[0-9a-fA-F]{12}$`
	regex, err := getCompiledRegex(uuidPattern)
	if err != nil {
		return validation.NewValidationError(validation.CodePattern, "invalid regex pattern")
	}
	
	if !regex.MatchString(str) {
		return validation.NewValidationError(validation.CodeFormat, "must be a valid UUID")
	}
	
	return validation.NewValidationResult()
}

// ===============================
// Numeric Validation Functions
// ===============================

// IsNumber validates that value is a valid number
var IsNumber validation.ValidatorFunc = func(value interface{}) validation.ValidationResult {
	switch value.(type) {
	case int, int8, int16, int32, int64:
		return validation.NewValidationResult()
	case uint, uint8, uint16, uint32, uint64:
		return validation.NewValidationResult()
	case float32, float64:
		return validation.NewValidationResult()
	case string:
		str := value.(string)
		if _, err := strconv.ParseFloat(str, 64); err != nil {
			return validation.NewValidationError(validation.CodeNumeric, "must be a valid number")
		}
		return validation.NewValidationResult()
	default:
		return validation.NewValidationError(validation.CodeType, "must be a number")
	}
}

// IsInteger validates that value is a valid integer
var IsInteger validation.ValidatorFunc = func(value interface{}) validation.ValidationResult {
	switch value.(type) {
	case int, int8, int16, int32, int64:
		return validation.NewValidationResult()
	case uint, uint8, uint16, uint32, uint64:
		return validation.NewValidationResult()
	case string:
		str := value.(string)
		if _, err := strconv.ParseInt(str, 10, 64); err != nil {
			return validation.NewValidationError(validation.CodeNumeric, "must be a valid integer")
		}
		return validation.NewValidationResult()
	default:
		return validation.NewValidationError(validation.CodeType, "must be an integer")
	}
}

// Min validates minimum numeric value
func Min(min float64) validation.ValidatorFunc {
	return func(value interface{}) validation.ValidationResult {
		num, err := validation.ConvertToFloat64(value)
		if err != nil {
			return validation.NewValidationError(validation.CodeType, "must be a valid number")
		}
		
		if num < min {
			return validation.NewValidationError(validation.CodeRange, fmt.Sprintf("must be at least %g", min))
		}
		
		return validation.NewValidationResult()
	}
}

// Max validates maximum numeric value
func Max(max float64) validation.ValidatorFunc {
	return func(value interface{}) validation.ValidationResult {
		num, err := validation.ConvertToFloat64(value)
		if err != nil {
			return validation.NewValidationError(validation.CodeType, "must be a valid number")
		}
		
		if num > max {
			return validation.NewValidationError(validation.CodeRange, fmt.Sprintf("must be at most %g", max))
		}
		
		return validation.NewValidationResult()
	}
}

// Range validates that numeric value is within a range
func Range(min, max float64) validation.ValidatorFunc {
	return func(value interface{}) validation.ValidationResult {
		num, err := validation.ConvertToFloat64(value)
		if err != nil {
			return validation.NewValidationError(validation.CodeType, "must be a valid number")
		}
		
		if num < min || num > max {
			return validation.NewValidationError(validation.CodeRange, fmt.Sprintf("must be between %g and %g", min, max))
		}
		
		return validation.NewValidationResult()
	}
}

// ===============================
// Date/Time Validation Functions
// ===============================

// IsDate validates that string is a valid date
var IsDate validation.ValidatorFunc = func(value interface{}) validation.ValidationResult {
	str, ok := value.(string)
	if !ok {
		return validation.NewValidationError(validation.CodeType, "value must be a string")
	}
	
	formats := []string{
		"2006-01-02",
		"01/02/2006",
		"2006-01-02T15:04:05Z07:00",
		"2006-01-02 15:04:05",
		time.RFC3339,
	}
	
	for _, format := range formats {
		if _, err := time.Parse(format, str); err == nil {
			return validation.NewValidationResult()
		}
	}
	
	return validation.NewValidationError(validation.CodeDate, "must be a valid date")
}

// DateAfter validates that date is after a specific date
func DateAfter(after time.Time) validation.ValidatorFunc {
	return func(value interface{}) validation.ValidationResult {
		var t time.Time
		var err error
		
		switch v := value.(type) {
		case time.Time:
			t = v
		case string:
			formats := []string{
				"2006-01-02",
				"01/02/2006",
				"2006-01-02T15:04:05Z07:00",
				"2006-01-02 15:04:05",
				time.RFC3339,
			}
			
			for _, format := range formats {
				if t, err = time.Parse(format, v); err == nil {
					break
				}
			}
			
			if err != nil {
				return validation.NewValidationError(validation.CodeDate, "must be a valid date")
			}
		default:
			return validation.NewValidationError(validation.CodeType, "must be a date")
		}
		
		if !t.After(after) {
			return validation.NewValidationError(validation.CodeDate, fmt.Sprintf("must be after %s", after.Format("2006-01-02")))
		}
		
		return validation.NewValidationResult()
	}
}

// DateBefore validates that date is before a specific date
func DateBefore(before time.Time) validation.ValidatorFunc {
	return func(value interface{}) validation.ValidationResult {
		var t time.Time
		var err error
		
		switch v := value.(type) {
		case time.Time:
			t = v
		case string:
			formats := []string{
				"2006-01-02",
				"01/02/2006",
				"2006-01-02T15:04:05Z07:00",
				"2006-01-02 15:04:05",
				time.RFC3339,
			}
			
			for _, format := range formats {
				if t, err = time.Parse(format, v); err == nil {
					break
				}
			}
			
			if err != nil {
				return validation.NewValidationError(validation.CodeDate, "must be a valid date")
			}
		default:
			return validation.NewValidationError(validation.CodeType, "must be a date")
		}
		
		if !t.Before(before) {
			return validation.NewValidationError(validation.CodeDate, fmt.Sprintf("must be before %s", before.Format("2006-01-02")))
		}
		
		return validation.NewValidationResult()
	}
}

// ===============================
// Collection Validation Functions
// ===============================

// In validates that value is in a list of allowed values
func In(allowed ...interface{}) validation.ValidatorFunc {
	return func(value interface{}) validation.ValidationResult {
		for _, item := range allowed {
			if value == item {
				return validation.NewValidationResult()
			}
		}
		
		return validation.NewValidationError(validation.CodeCustom, fmt.Sprintf("must be one of: %v", allowed))
	}
}

// NotIn validates that value is not in a list of forbidden values
func NotIn(forbidden ...interface{}) validation.ValidatorFunc {
	return func(value interface{}) validation.ValidationResult {
		for _, item := range forbidden {
			if value == item {
				return validation.NewValidationError(validation.CodeCustom, fmt.Sprintf("must not be one of: %v", forbidden))
			}
		}
		
		return validation.NewValidationResult()
	}
}

// ===============================
// Business Validation Functions
// ===============================

// CreditCard validates credit card number using Luhn algorithm
var CreditCard validation.ValidatorFunc = func(value interface{}) validation.ValidationResult {
	str, ok := value.(string)
	if !ok {
		return validation.NewValidationError(validation.CodeType, "value must be a string")
	}
	
	// Remove spaces and dashes
	cleaned := strings.ReplaceAll(strings.ReplaceAll(str, " ", ""), "-", "")
	
	// Check if all characters are digits
	for _, r := range cleaned {
		if !unicode.IsDigit(r) {
			return validation.NewValidationError(validation.CodeFormat, "must contain only digits")
		}
	}
	
	// Check length (most credit cards are 13-19 digits)
	if len(cleaned) < 13 || len(cleaned) > 19 {
		return validation.NewValidationError(validation.CodeLength, "must be between 13 and 19 digits")
	}
	
	// Luhn algorithm validation
	if !luhnCheck(cleaned) {
		return validation.NewValidationError(validation.CodeFormat, "must be a valid credit card number")
	}
	
	return validation.NewValidationResult()
}

// luhnCheck implements the Luhn algorithm for credit card validation
func luhnCheck(number string) bool {
	var sum int
	alternate := false
	
	// Process digits from right to left
	for i := len(number) - 1; i >= 0; i-- {
		digit := int(number[i] - '0')
		
		if alternate {
			digit *= 2
			if digit > 9 {
				digit = (digit % 10) + 1
			}
		}
		
		sum += digit
		alternate = !alternate
	}
	
	return sum%10 == 0
}

// Phone validates phone number format (basic validation)
var Phone validation.ValidatorFunc = func(value interface{}) validation.ValidationResult {
	str, ok := value.(string)
	if !ok {
		return validation.NewValidationError(validation.CodeType, "value must be a string")
	}
	
	// Remove common formatting characters
	cleaned := strings.ReplaceAll(str, " ", "")
	cleaned = strings.ReplaceAll(cleaned, "-", "")
	cleaned = strings.ReplaceAll(cleaned, "(", "")
	cleaned = strings.ReplaceAll(cleaned, ")", "")
	cleaned = strings.ReplaceAll(cleaned, ".", "")
	cleaned = strings.ReplaceAll(cleaned, "+", "")
	
	// Check if remaining characters are digits
	for _, r := range cleaned {
		if !unicode.IsDigit(r) {
			return validation.NewValidationError(validation.CodePhoneNumber, "must contain only digits and formatting characters")
		}
	}
	
	// Check length (most phone numbers are 7-15 digits)
	if len(cleaned) < 7 || len(cleaned) > 15 {
		return validation.NewValidationError(validation.CodePhoneNumber, "must be between 7 and 15 digits")
	}
	
	return validation.NewValidationResult()
}

// ===============================
// Custom Validation Functions
// ===============================

// Custom creates a custom validator with a validation function
func Custom(fn func(interface{}) (bool, string)) validation.ValidatorFunc {
	return func(value interface{}) validation.ValidationResult {
		valid, message := fn(value)
		if !valid {
			return validation.NewValidationError(validation.CodeCustom, message)
		}
		
		return validation.NewValidationResult()
	}
}

// ===============================
// Utility Functions
// ===============================

// Validate runs validation on a map of field values
func Validate(data map[string]interface{}, rules map[string]*ValidatorChain) validation.ValidationResult {
	var results []validation.ValidationResult
	
	for field, chain := range rules {
		value, exists := data[field]
		
		// If field doesn't exist, treat as nil
		if !exists {
			value = nil
		}
		
		fieldResult := chain.Validate(value)
		// Add field context to errors if not already present
		for i := range fieldResult.Errors {
			if fieldResult.Errors[i].Field == "" {
				fieldResult.Errors[i].Field = field
			}
		}
		results = append(results, fieldResult)
	}
	
	return validation.Combine(results...)
}

// ValidateStruct validates struct fields using tags (basic implementation)
func ValidateStruct(s interface{}) validation.ValidationResult {
	result := &validation.ValidationResult{Valid: true}
	
	// Use reflection to iterate through struct fields
	v := reflect.ValueOf(s)
	t := reflect.TypeOf(s)
	
	// Handle pointer to struct
	if v.Kind() == reflect.Ptr {
		if v.IsNil() {
			return *result
		}
		v = v.Elem()
		t = t.Elem()
	}
	
	// Only process structs
	if v.Kind() != reflect.Struct {
		return *result
	}
	
	// Iterate through fields
	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		fieldType := t.Field(i)
		
		// Skip unexported fields
		if !field.CanInterface() {
			continue
		}
		
		fieldValue := field.Interface()
		
		// Check validate tag
		validateTag := fieldType.Tag.Get("validate")
		if validateTag == "" {
			continue
		}
		
		// Parse comma-separated validation rules
		rules := strings.Split(validateTag, ",")
		
		for _, rule := range rules {
			rule = strings.TrimSpace(rule)
			
			// Handle required validation
			if rule == "required" {
				if isFieldEmpty(fieldValue) {
					result.AddError("REQUIRED", fmt.Sprintf("%s is required", fieldType.Name))
				}
			}
			
			// Handle min_length validation
			if strings.HasPrefix(rule, "min_length:") {
				if str, ok := fieldValue.(string); ok {
					minStr := strings.TrimPrefix(rule, "min_length:")
					if min, err := strconv.Atoi(minStr); err == nil {
						if len(str) < min {
							result.AddError("MIN_LENGTH", fmt.Sprintf("%s must be at least %d characters", fieldType.Name, min))
						}
					}
				}
			}
			
			// Handle min value validation
			if strings.HasPrefix(rule, "min:") {
				if age, ok := fieldValue.(int); ok {
					minStr := strings.TrimPrefix(rule, "min:")
					if min, err := strconv.Atoi(minStr); err == nil {
						if age < min {
							result.AddError("MIN_VALUE", fmt.Sprintf("%s must be at least %d", fieldType.Name, min))
						}
					}
				}
			}
			
			// Handle max value validation  
			if strings.HasPrefix(rule, "max:") {
				if age, ok := fieldValue.(int); ok {
					maxStr := strings.TrimPrefix(rule, "max:")
					if max, err := strconv.Atoi(maxStr); err == nil {
						if age > max {
							result.AddError("MAX_VALUE", fmt.Sprintf("%s must be at most %d", fieldType.Name, max))
						}
					}
				}
			}
			
			// Handle email validation
			if rule == "email" {
				if str, ok := fieldValue.(string); ok && str != "" {
					if !IsValidEmail(str) {
						result.AddError("EMAIL_FORMAT", fmt.Sprintf("%s must be a valid email", fieldType.Name))
					}
				}
			}
		}
	}
	
	return *result
}

// isFieldEmpty checks if a field value is considered empty
func isFieldEmpty(value interface{}) bool {
	if value == nil {
		return true
	}
	
	switch v := value.(type) {
	case string:
		return strings.TrimSpace(v) == ""
	case int, int8, int16, int32, int64:
		return v == 0
	case uint, uint8, uint16, uint32, uint64:
		return v == 0
	case float32, float64:
		return v == 0
	case bool:
		return !v
	default:
		val := reflect.ValueOf(value)
		switch val.Kind() {
		case reflect.Array, reflect.Slice, reflect.Map, reflect.Chan:
			return val.Len() == 0
		case reflect.Ptr, reflect.Interface:
			return val.IsNil()
		}
		return false
	}
}

// ===============================
// Convenience Functions
// ===============================

// IsValidEmail is a convenience function for email validation
func IsValidEmail(email string) bool {
	result := Email.Validate(email)
	return result.Valid
}

// IsValidURL is a convenience function for URL validation
func IsValidURL(urlStr string) bool {
	result := URL.Validate(urlStr)
	return result.Valid
}

// IsValidIP is a convenience function for IP validation
func IsValidIP(ip string) bool {
	result := IP.Validate(ip)
	return result.Valid
}

// IsValidUUID is a convenience function for UUID validation
func IsValidUUID(uuid string) bool {
	result := UUID.Validate(uuid)
	return result.Valid
}

// IsValidCreditCard is a convenience function for credit card validation
func IsValidCreditCard(number string) bool {
	result := CreditCard.Validate(number)
	return result.Valid
}

// IsValidPhone is a convenience function for phone validation
func IsValidPhone(phone string) bool {
	result := Phone.Validate(phone)
	return result.Valid
}