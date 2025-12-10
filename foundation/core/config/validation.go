// File: validation.go
// Title: Configuration Validation Implementation
// Description: Implements comprehensive validation for configuration values
//              including type checking, range validation, required fields,
//              and custom validation rules.
// Author: msto63 with Claude Sonnet 4.0
// Version: v0.1.0
// Created: 2025-01-25
// Modified: 2025-01-25
//
// Change History:
// - 2025-01-25 v0.1.0: Initial implementation of validation

package config

import (
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"

	mdwerror "github.com/msto63/mDW/foundation/core/error"
)

// ValidationResult contains the results of configuration validation
type ValidationResult struct {
	Valid  bool     `json:"valid"`
	Errors []string `json:"errors,omitempty"`
}

// Validate validates the configuration against the provided rules
func (c *Config) Validate(rules ValidationRules) *ValidationResult {
	c.mu.RLock()
	defer c.mu.RUnlock()

	result := &ValidationResult{
		Valid:  true,
		Errors: make([]string, 0),
	}

	for key, rule := range rules {
		if err := c.validateField(key, rule); err != nil {
			result.Valid = false
			result.Errors = append(result.Errors, err.Error())
		}
	}

	return result
}

// validateField validates a single configuration field
func (c *Config) validateField(key string, rule ValidationRule) error {
	value := c.getValue(key)

	// Check if required field is missing
	if rule.Required && value == nil {
		return fmt.Errorf("required field '%s' is missing", key)
	}

	// If value is nil and not required, apply default if available
	if value == nil {
		if rule.Default != nil {
			c.Set(key, rule.Default)
		}
		return nil
	}

	// Validate type
	if rule.Type != "" {
		if err := c.validateType(key, value, rule.Type); err != nil {
			return err
		}
	}

	// Validate range/bounds
	if err := c.validateBounds(key, value, rule); err != nil {
		return err
	}

	// Validate pattern (for strings)
	if rule.Pattern != "" {
		if err := c.validatePattern(key, value, rule.Pattern); err != nil {
			return err
		}
	}

	return nil
}

// validateType validates the type of a configuration value
func (c *Config) validateType(key string, value interface{}, expectedType string) error {
	actualType := reflect.TypeOf(value)

	switch expectedType {
	case "string":
		if actualType.Kind() != reflect.String {
			return fmt.Errorf("field '%s' must be a string, got %s", key, actualType.Kind())
		}

	case "int":
		switch actualType.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			// Valid integer types
		case reflect.Float64:
			// TOML numbers are float64, check if it's a whole number
			if f, ok := value.(float64); ok && f == float64(int64(f)) {
				// It's a whole number, convert it
				c.Set(key, int64(f))
			} else {
				return fmt.Errorf("field '%s' must be an integer, got float with decimal places", key)
			}
		default:
			return fmt.Errorf("field '%s' must be an integer, got %s", key, actualType.Kind())
		}

	case "float":
		switch actualType.Kind() {
		case reflect.Float32, reflect.Float64:
			// Valid float types
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			// Integers can be converted to floats
		default:
			return fmt.Errorf("field '%s' must be a float, got %s", key, actualType.Kind())
		}

	case "bool":
		if actualType.Kind() != reflect.Bool {
			return fmt.Errorf("field '%s' must be a boolean, got %s", key, actualType.Kind())
		}

	case "duration":
		if actualType.Kind() == reflect.String {
			if _, err := time.ParseDuration(value.(string)); err != nil {
				return fmt.Errorf("field '%s' must be a valid duration string, got '%v'", key, value)
			}
		} else if actualType != reflect.TypeOf(time.Duration(0)) {
			return fmt.Errorf("field '%s' must be a duration, got %s", key, actualType.Kind())
		}

	case "[]string":
		if actualType.Kind() == reflect.Slice {
			// Check if it's a slice of strings or interfaces that can be converted
			if slice, ok := value.([]interface{}); ok {
				// Convert to []string
				stringSlice := make([]string, len(slice))
				for i, item := range slice {
					stringSlice[i] = fmt.Sprintf("%v", item)
				}
				c.Set(key, stringSlice)
			} else if _, ok := value.([]string); !ok {
				return fmt.Errorf("field '%s' must be a slice of strings", key)
			}
		} else {
			return fmt.Errorf("field '%s' must be a slice of strings, got %s", key, actualType.Kind())
		}

	default:
		return fmt.Errorf("unknown validation type: %s", expectedType)
	}

	return nil
}

// validateBounds validates numeric bounds and string/slice lengths
func (c *Config) validateBounds(key string, value interface{}, rule ValidationRule) error {
	// Validate minimum value/length
	if rule.Min != nil {
		if err := c.validateMin(key, value, rule.Min); err != nil {
			return err
		}
	}

	// Validate maximum value/length
	if rule.Max != nil {
		if err := c.validateMax(key, value, rule.Max); err != nil {
			return err
		}
	}

	return nil
}

// validateMin validates minimum values or lengths
func (c *Config) validateMin(key string, value interface{}, min interface{}) error {
	switch v := value.(type) {
	case int, int8, int16, int32, int64:
		intVal := reflect.ValueOf(v).Int()
		minVal := reflect.ValueOf(min).Int()
		if intVal < minVal {
			return fmt.Errorf("field '%s' value %d is less than minimum %d", key, intVal, minVal)
		}

	case float32, float64:
		floatVal := reflect.ValueOf(v).Float()
		minVal := reflect.ValueOf(min).Float()
		if floatVal < minVal {
			return fmt.Errorf("field '%s' value %g is less than minimum %g", key, floatVal, minVal)
		}

	case string:
		if minLen, ok := min.(int); ok && len(v) < minLen {
			return fmt.Errorf("field '%s' length %d is less than minimum %d", key, len(v), minLen)
		}

	case []string:
		if minLen, ok := min.(int); ok && len(v) < minLen {
			return fmt.Errorf("field '%s' length %d is less than minimum %d", key, len(v), minLen)
		}

	case []interface{}:
		if minLen, ok := min.(int); ok && len(v) < minLen {
			return fmt.Errorf("field '%s' length %d is less than minimum %d", key, len(v), minLen)
		}
	}

	return nil
}

// validateMax validates maximum values or lengths
func (c *Config) validateMax(key string, value interface{}, max interface{}) error {
	switch v := value.(type) {
	case int, int8, int16, int32, int64:
		intVal := reflect.ValueOf(v).Int()
		maxVal := reflect.ValueOf(max).Int()
		if intVal > maxVal {
			return fmt.Errorf("field '%s' value %d is greater than maximum %d", key, intVal, maxVal)
		}

	case float32, float64:
		floatVal := reflect.ValueOf(v).Float()
		maxVal := reflect.ValueOf(max).Float()
		if floatVal > maxVal {
			return fmt.Errorf("field '%s' value %g is greater than maximum %g", key, floatVal, maxVal)
		}

	case string:
		if maxLen, ok := max.(int); ok && len(v) > maxLen {
			return fmt.Errorf("field '%s' length %d is greater than maximum %d", key, len(v), maxLen)
		}

	case []string:
		if maxLen, ok := max.(int); ok && len(v) > maxLen {
			return fmt.Errorf("field '%s' length %d is greater than maximum %d", key, len(v), maxLen)
		}

	case []interface{}:
		if maxLen, ok := max.(int); ok && len(v) > maxLen {
			return fmt.Errorf("field '%s' length %d is greater than maximum %d", key, len(v), maxLen)
		}
	}

	return nil
}

// validatePattern validates string values against regex patterns
func (c *Config) validatePattern(key string, value interface{}, pattern string) error {
	strValue, ok := value.(string)
	if !ok {
		return fmt.Errorf("field '%s' pattern validation requires string value", key)
	}

	regex, err := regexp.Compile(pattern)
	if err != nil {
		return fmt.Errorf("invalid regex pattern for field '%s': %w", key, err)
	}

	if !regex.MatchString(strValue) {
		return fmt.Errorf("field '%s' value '%s' does not match pattern '%s'", key, strValue, pattern)
	}

	return nil
}

// BindToStruct binds configuration values to a Go struct
func (c *Config) BindToStruct(keyPrefix string, target interface{}) error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	targetValue := reflect.ValueOf(target)
	if targetValue.Kind() != reflect.Ptr || targetValue.Elem().Kind() != reflect.Struct {
		return mdwerror.New("target must be a pointer to struct").
			WithCode(mdwerror.CodeValidationFailed).
			WithOperation("config.BindToStruct")
	}

	targetStruct := targetValue.Elem()
	targetType := targetStruct.Type()

	// Get the configuration section
	var configData map[string]interface{}
	if keyPrefix == "" {
		configData = c.data
	} else {
		if value := c.getValue(keyPrefix); value != nil {
			if data, ok := value.(map[string]interface{}); ok {
				configData = data
			} else {
				return mdwerror.New(fmt.Sprintf("configuration key '%s' is not a section", keyPrefix)).
					WithCode(mdwerror.CodeValidationFailed).
					WithOperation("config.BindToStruct").
					WithDetail("keyPrefix", keyPrefix)
			}
		} else {
			return mdwerror.New(fmt.Sprintf("configuration section '%s' not found", keyPrefix)).
				WithCode(mdwerror.CodeNotFound).
				WithOperation("config.BindToStruct").
				WithDetail("keyPrefix", keyPrefix)
		}
	}

	// Bind fields
	for i := 0; i < targetStruct.NumField(); i++ {
		field := targetStruct.Field(i)
		fieldType := targetType.Field(i)

		if !field.CanSet() {
			continue
		}

		// Get field name from tag or use field name
		configKey := fieldType.Tag.Get("config")
		if configKey == "" {
			configKey = strings.ToLower(fieldType.Name)
		}

		// Skip fields marked with "-"
		if configKey == "-" {
			continue
		}

		// Get value from configuration
		configValue := configData[configKey]
		if configValue == nil {
			// Check if field is required
			validate := fieldType.Tag.Get("validate")
			if strings.Contains(validate, "required") {
				return mdwerror.New(fmt.Sprintf("required field '%s' not found in configuration", configKey)).
					WithCode(mdwerror.CodeValidationFailed).
					WithOperation("config.BindToStruct").
					WithDetail("configKey", configKey)
			}
			continue
		}

		// Set field value
		if err := c.setFieldValue(field, configValue, fieldType); err != nil {
			return mdwerror.Wrap(err, fmt.Sprintf("error setting field '%s'", fieldType.Name)).
			WithCode(mdwerror.CodeInvalidOperation).
			WithOperation("config.BindToStruct").
			WithDetail("fieldName", fieldType.Name)
		}
	}

	return nil
}

// setFieldValue sets a struct field value from configuration data
func (c *Config) setFieldValue(field reflect.Value, configValue interface{}, fieldType reflect.StructField) error {
	switch field.Kind() {
	case reflect.String:
		if str, ok := configValue.(string); ok {
			field.SetString(str)
		} else {
			field.SetString(fmt.Sprintf("%v", configValue))
		}

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		var intVal int64
		switch v := configValue.(type) {
		case int:
			intVal = int64(v)
		case int64:
			intVal = v
		case float64:
			intVal = int64(v)
		case string:
			var err error
			intVal, err = strconv.ParseInt(v, 10, 64)
			if err != nil {
				return fmt.Errorf("cannot convert '%v' to integer", v)
			}
		default:
			return fmt.Errorf("cannot convert '%v' to integer", v)
		}
		field.SetInt(intVal)

	case reflect.Float32, reflect.Float64:
		var floatVal float64
		switch v := configValue.(type) {
		case float64:
			floatVal = v
		case float32:
			floatVal = float64(v)
		case int:
			floatVal = float64(v)
		case int64:
			floatVal = float64(v)
		case string:
			var err error
			floatVal, err = strconv.ParseFloat(v, 64)
			if err != nil {
				return fmt.Errorf("cannot convert '%v' to float", v)
			}
		default:
			return fmt.Errorf("cannot convert '%v' to float", v)
		}
		field.SetFloat(floatVal)

	case reflect.Bool:
		var boolVal bool
		switch v := configValue.(type) {
		case bool:
			boolVal = v
		case string:
			var err error
			boolVal, err = strconv.ParseBool(v)
			if err != nil {
				return fmt.Errorf("cannot convert '%v' to boolean", v)
			}
		default:
			return fmt.Errorf("cannot convert '%v' to boolean", v)
		}
		field.SetBool(boolVal)

	case reflect.Slice:
		if field.Type().Elem().Kind() == reflect.String {
			// Handle []string
			var stringSlice []string
			switch v := configValue.(type) {
			case []string:
				stringSlice = v
			case []interface{}:
				stringSlice = make([]string, len(v))
				for i, item := range v {
					stringSlice[i] = fmt.Sprintf("%v", item)
				}
			case string:
				stringSlice = []string{v}
			default:
				return fmt.Errorf("cannot convert '%v' to []string", v)
			}
			field.Set(reflect.ValueOf(stringSlice))
		}

	default:
		return fmt.Errorf("unsupported field type: %s", field.Kind())
	}

	return nil
}