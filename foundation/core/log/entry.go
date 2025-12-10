// File: entry.go
// Title: Log Entry Structure
// Description: Defines the log entry structure that holds all information
//              about a single log message including metadata, context, and
//              performance measurements.
// Author: msto63 with Claude Sonnet 4.0
// Version: v0.1.0
// Created: 2025-01-24
// Modified: 2025-01-24
//
// Change History:
// - 2025-01-24 v0.1.0: Initial implementation with comprehensive log entry structure

package log

import (
	"time"
)

// Entry represents a single log entry with all its metadata
type Entry struct {
	// Core log information
	Timestamp time.Time
	Level     Level
	Message   string
	Logger    string
	
	// Context information
	RequestID     string
	UserID        string
	CorrelationID string
	
	// Custom fields
	Fields Fields
	
	// Error information
	Error error
	
	// Performance metrics
	Duration time.Duration
	
	// Caller information (optional, for debugging)
	Caller *CallerInfo
}

// CallerInfo contains information about where the log was called from
type CallerInfo struct {
	Function string
	File     string
	Line     int
}

// Fields represents custom key-value pairs for structured logging
type Fields map[string]interface{}

// Field creates a single field for logging
func Field(key string, value interface{}) Fields {
	return Fields{key: value}
}

// Err creates an error field for logging
func Err(err error) Fields {
	return Fields{"error": err}
}

// Duration creates a duration field for logging
func Duration(key string, duration time.Duration) Fields {
	return Fields{key: duration}
}

// Int creates an integer field for logging
func Int(key string, value int) Fields {
	return Fields{key: value}
}

// Int64 creates an int64 field for logging
func Int64(key string, value int64) Fields {
	return Fields{key: value}
}

// Float64 creates a float64 field for logging
func Float64(key string, value float64) Fields {
	return Fields{key: value}
}

// String creates a string field for logging
func String(key string, value string) Fields {
	return Fields{key: value}
}

// Bool creates a boolean field for logging
func Bool(key string, value bool) Fields {
	return Fields{key: value}
}

// Time creates a time field for logging
func Time(key string, value time.Time) Fields {
	return Fields{key: value}
}

// Any creates a field with any value type for logging
func Any(key string, value interface{}) Fields {
	return Fields{key: value}
}

// Merge combines multiple Fields into one
func (f Fields) Merge(other Fields) Fields {
	result := make(Fields)
	for k, v := range f {
		result[k] = v
	}
	for k, v := range other {
		result[k] = v
	}
	return result
}

// With adds a field to the existing Fields
func (f Fields) With(key string, value interface{}) Fields {
	if f == nil {
		f = make(Fields)
	}
	f[key] = value
	return f
}

// Clone creates a copy of the Fields
func (f Fields) Clone() Fields {
	if f == nil {
		return nil
	}
	result := make(Fields, len(f))
	for k, v := range f {
		result[k] = v
	}
	return result
}

// NewEntry creates a new log entry with the given level and message
func NewEntry(level Level, message string) *Entry {
	return &Entry{
		Timestamp: time.Now(),
		Level:     level,
		Message:   message,
		Fields:    make(Fields),
	}
}

// WithFields adds custom fields to the entry
func (e *Entry) WithFields(fields Fields) *Entry {
	if e.Fields == nil {
		e.Fields = make(Fields)
	}
	for k, v := range fields {
		e.Fields[k] = v
	}
	return e
}

// WithField adds a single custom field to the entry
func (e *Entry) WithField(key string, value interface{}) *Entry {
	if e.Fields == nil {
		e.Fields = make(Fields)
	}
	e.Fields[key] = value
	return e
}

// WithError adds error information to the entry
func (e *Entry) WithError(err error) *Entry {
	e.Error = err
	return e
}

// WithDuration adds duration information to the entry
func (e *Entry) WithDuration(duration time.Duration) *Entry {
	e.Duration = duration
	return e
}

// WithRequestID adds request ID context to the entry
func (e *Entry) WithRequestID(requestID string) *Entry {
	e.RequestID = requestID
	return e
}

// WithUserID adds user ID context to the entry
func (e *Entry) WithUserID(userID string) *Entry {
	e.UserID = userID
	return e
}

// WithCorrelationID adds correlation ID context to the entry
func (e *Entry) WithCorrelationID(correlationID string) *Entry {
	e.CorrelationID = correlationID
	return e
}

// WithLogger sets the logger name for the entry
func (e *Entry) WithLogger(logger string) *Entry {
	e.Logger = logger
	return e
}

// WithCaller adds caller information to the entry
func (e *Entry) WithCaller(function, file string, line int) *Entry {
	e.Caller = &CallerInfo{
		Function: function,
		File:     file,
		Line:     line,
	}
	return e
}

// Clone creates a copy of the entry
func (e *Entry) Clone() *Entry {
	if e == nil {
		return nil
	}
	
	clone := &Entry{
		Timestamp:     e.Timestamp,
		Level:         e.Level,
		Message:       e.Message,
		Logger:        e.Logger,
		RequestID:     e.RequestID,
		UserID:        e.UserID,
		CorrelationID: e.CorrelationID,
		Fields:        e.Fields.Clone(),
		Error:         e.Error,
		Duration:      e.Duration,
	}
	
	if e.Caller != nil {
		clone.Caller = &CallerInfo{
			Function: e.Caller.Function,
			File:     e.Caller.File,
			Line:     e.Caller.Line,
		}
	}
	
	return clone
}