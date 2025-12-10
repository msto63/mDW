// File: chain.go
// Title: Validator Chain Implementation
// Description: Provides composable validator chains that allow combining multiple
//              validation rules into a single validator. Supports fluent API for
//              building complex validation pipelines with proper error handling.
// Author: msto63 with Claude Sonnet 4.0
// Version: v0.1.0
// Created: 2025-01-25
// Modified: 2025-01-25
//
// Change History:
// - 2025-01-25 v0.1.0: Initial validator chain implementation

package validation

import (
	"context"
	"fmt"
)

// ValidatorChain represents a chain of validators that can be executed sequentially
type ValidatorChain struct {
	validators []Validator
	name       string
	stopOnFirstError bool
	context    map[string]interface{}
}

// NewValidatorChain creates a new validator chain with an optional name
func NewValidatorChain(name ...string) *ValidatorChain {
	chainName := ""
	if len(name) > 0 {
		chainName = name[0]
	}
	
	return &ValidatorChain{
		validators: make([]Validator, 0),
		name:       chainName,
		stopOnFirstError: false,
		context:    make(map[string]interface{}),
	}
}

// Add adds a validator to the chain
func (c *ValidatorChain) Add(validator Validator) *ValidatorChain {
	c.validators = append(c.validators, validator)
	return c
}

// AddFunc adds a validator function to the chain
func (c *ValidatorChain) AddFunc(fn ValidatorFunc) *ValidatorChain {
	c.validators = append(c.validators, fn)
	return c
}

// StopOnFirstError configures the chain to stop on the first validation error
// By default, chains collect all validation errors
func (c *ValidatorChain) StopOnFirstError(stop bool) *ValidatorChain {
	c.stopOnFirstError = stop
	return c
}

// WithContext adds context information that will be passed to all validators
func (c *ValidatorChain) WithContext(key string, value interface{}) *ValidatorChain {
	c.context[key] = value
	return c
}

// WithName sets or updates the chain name for better error reporting
func (c *ValidatorChain) WithName(name string) *ValidatorChain {
	c.name = name
	return c
}

// Validate executes all validators in the chain and returns combined results
func (c *ValidatorChain) Validate(value interface{}) ValidationResult {
	return c.ValidateWithContext(context.Background(), value)
}

// ValidateWithContext executes all validators with context support
func (c *ValidatorChain) ValidateWithContext(ctx context.Context, value interface{}) ValidationResult {
	// Create context with chain information
	chainCtx := context.WithValue(ctx, "validatorChain", c.name)
	for key, val := range c.context {
		chainCtx = context.WithValue(chainCtx, key, val)
	}
	
	var allResults []ValidationResult
	
	// Execute each validator in sequence
	for i, validator := range c.validators {
		result := validator.ValidateWithContext(chainCtx, value)
		
		// Add chain context to result
		if result.Context == nil {
			result.Context = make(map[string]interface{})
		}
		if c.name != "" {
			result.Context["validatorChain"] = c.name
		}
		result.Context["validatorIndex"] = i
		
		allResults = append(allResults, result)
		
		// Stop on first error if configured
		if c.stopOnFirstError && !result.Valid {
			break
		}
	}
	
	// Combine all results
	combined := Combine(allResults...)
	
	// Add chain-level context
	if c.name != "" {
		combined.WithContext("validatorChain", c.name)
	}
	combined.WithContext("totalValidators", len(c.validators))
	combined.WithContext("executedValidators", len(allResults))
	
	return combined
}

// Length returns the number of validators in the chain
func (c *ValidatorChain) Length() int {
	return len(c.validators)
}

// Name returns the chain name
func (c *ValidatorChain) Name() string {
	return c.name
}

// String returns a string representation of the validator chain
func (c *ValidatorChain) String() string {
	name := c.name
	if name == "" {
		name = "unnamed"
	}
	return fmt.Sprintf("ValidatorChain{name: %s, validators: %d, stopOnFirstError: %v}", 
		name, len(c.validators), c.stopOnFirstError)
}

// ConditionalValidator allows conditional execution of validators based on a predicate
type ConditionalValidator struct {
	condition func(interface{}) bool
	validator Validator
	name      string
}

// NewConditionalValidator creates a validator that only executes if the condition is true
func NewConditionalValidator(condition func(interface{}) bool, validator Validator, name ...string) *ConditionalValidator {
	condName := ""
	if len(name) > 0 {
		condName = name[0]
	}
	
	return &ConditionalValidator{
		condition: condition,
		validator: validator,
		name:      condName,
	}
}

// Validate executes the validator only if the condition is met
func (c *ConditionalValidator) Validate(value interface{}) ValidationResult {
	return c.ValidateWithContext(context.Background(), value)
}

// ValidateWithContext executes conditional validation with context
func (c *ConditionalValidator) ValidateWithContext(ctx context.Context, value interface{}) ValidationResult {
	// Check condition
	if !c.condition(value) {
		// Condition not met, validation passes
		result := NewValidationResult()
		result.WithContext("conditionalValidator", c.name)
		result.WithContext("conditionMet", false)
		return result
	}
	
	// Condition met, execute validator
	result := c.validator.ValidateWithContext(ctx, value)
	
	// Add conditional context
	if result.Context == nil {
		result.Context = make(map[string]interface{})
	}
	result.Context["conditionalValidator"] = c.name
	result.Context["conditionMet"] = true
	
	return result
}

// String returns a string representation of the conditional validator
func (c *ConditionalValidator) String() string {
	name := c.name
	if name == "" {
		name = "unnamed"
	}
	return fmt.Sprintf("ConditionalValidator{name: %s}", name)
}

// ParallelValidator executes multiple validators concurrently
type ParallelValidator struct {
	validators []Validator
	name       string
}

// NewParallelValidator creates a validator that executes validators in parallel
func NewParallelValidator(name ...string) *ParallelValidator {
	validatorName := ""
	if len(name) > 0 {
		validatorName = name[0]
	}
	
	return &ParallelValidator{
		validators: make([]Validator, 0),
		name:       validatorName,
	}
}

// Add adds a validator to the parallel execution group
func (p *ParallelValidator) Add(validator Validator) *ParallelValidator {
	p.validators = append(p.validators, validator)
	return p
}

// Validate executes all validators in parallel
func (p *ParallelValidator) Validate(value interface{}) ValidationResult {
	return p.ValidateWithContext(context.Background(), value)
}

// ValidateWithContext executes validators in parallel with context support
func (p *ParallelValidator) ValidateWithContext(ctx context.Context, value interface{}) ValidationResult {
	if len(p.validators) == 0 {
		return NewValidationResult()
	}
	
	// Create channels for results
	results := make(chan ValidationResult, len(p.validators))
	
	// Execute validators concurrently
	for i, validator := range p.validators {
		go func(idx int, v Validator) {
			result := v.ValidateWithContext(ctx, value)
			
			// Add parallel context
			if result.Context == nil {
				result.Context = make(map[string]interface{})
			}
			result.Context["parallelValidator"] = p.name
			result.Context["validatorIndex"] = idx
			
			results <- result
		}(i, validator)
	}
	
	// Collect all results
	var allResults []ValidationResult
	for i := 0; i < len(p.validators); i++ {
		allResults = append(allResults, <-results)
	}
	
	// Combine results
	combined := Combine(allResults...)
	
	// Add parallel execution context
	if p.name != "" {
		combined.WithContext("parallelValidator", p.name)
	}
	combined.WithContext("parallelExecution", true)
	combined.WithContext("totalValidators", len(p.validators))
	
	return combined
}

// String returns a string representation of the parallel validator
func (p *ParallelValidator) String() string {
	name := p.name
	if name == "" {
		name = "unnamed"
	}
	return fmt.Sprintf("ParallelValidator{name: %s, validators: %d}", name, len(p.validators))
}