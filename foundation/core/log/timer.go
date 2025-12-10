// File: timer.go
// Title: Performance Timer
// Description: Provides timing functionality for measuring and logging
//              performance metrics. Integrates with the logging system
//              to automatically log timing information.
// Author: msto63 with Claude Sonnet 4.0
// Version: v0.1.0
// Created: 2025-01-24
// Modified: 2025-01-24
//
// Change History:
// - 2025-01-24 v0.1.0: Initial implementation with performance timing

package log

import (
	"time"
)

// Timer represents a performance timer for measuring operation duration
type Timer struct {
	logger    *Logger
	operation string
	startTime time.Time
	fields    Fields
	level     Level
	stopped   bool
}

// NewTimer creates a new timer for the given operation
func NewTimer(logger *Logger, operation string) *Timer {
	return &Timer{
		logger:    logger,
		operation: operation,
		startTime: time.Now(),
		fields:    make(Fields),
		level:     LevelDebug,
		stopped:   false,
	}
}

// WithLevel sets the log level for the timer completion message
func (t *Timer) WithLevel(level Level) *Timer {
	t.level = level
	return t
}

// WithField adds a field to be logged when the timer completes
func (t *Timer) WithField(key string, value interface{}) *Timer {
	if t.fields == nil {
		t.fields = make(Fields)
	}
	t.fields[key] = value
	return t
}

// WithFields adds multiple fields to be logged when the timer completes
func (t *Timer) WithFields(fields Fields) *Timer {
	if t.fields == nil {
		t.fields = make(Fields)
	}
	for k, v := range fields {
		t.fields[k] = v
	}
	return t
}

// Elapsed returns the elapsed time since the timer was started
func (t *Timer) Elapsed() time.Duration {
	return time.Since(t.startTime)
}

// Stop stops the timer and logs the elapsed time
func (t *Timer) Stop() time.Duration {
	if t.stopped {
		return 0
	}
	
	elapsed := t.Elapsed()
	t.stopped = true
	
	// Create log message
	message := t.operation + " completed"
	
	// Add timing fields
	if t.fields == nil {
		t.fields = make(Fields)
	}
	t.fields["operation"] = t.operation
	t.fields["duration_ms"] = float64(elapsed.Nanoseconds()) / 1000000
	t.fields["duration"] = elapsed.String()
	
	// Log the completion
	if t.logger != nil {
		switch t.level {
		case LevelTrace:
			t.logger.Trace(message, t.fields)
		case LevelDebug:
			t.logger.Debug(message, t.fields)
		case LevelInfo:
			t.logger.Info(message, t.fields)
		case LevelWarn:
			t.logger.Warn(message, t.fields)
		case LevelError:
			t.logger.Error(message, t.fields)
		}
	}
	
	return elapsed
}

// StopWithError stops the timer and logs an error with the elapsed time
func (t *Timer) StopWithError(err error) time.Duration {
	if t.stopped {
		return 0
	}
	
	elapsed := t.Elapsed()
	t.stopped = true
	
	// Create error message
	message := t.operation + " failed"
	
	// Add timing and error fields
	if t.fields == nil {
		t.fields = make(Fields)
	}
	t.fields["operation"] = t.operation
	t.fields["duration_ms"] = float64(elapsed.Nanoseconds()) / 1000000
	t.fields["duration"] = elapsed.String()
	t.fields["success"] = false
	
	// Log the error
	if t.logger != nil {
		t.logger.ErrorWithErr(message, err, t.fields)
	}
	
	return elapsed
}

// StopWithResult stops the timer and logs the result with elapsed time
func (t *Timer) StopWithResult(success bool, result interface{}) time.Duration {
	if t.stopped {
		return 0
	}
	
	elapsed := t.Elapsed()
	t.stopped = true
	
	// Create result message
	var message string
	if success {
		message = t.operation + " completed successfully"
	} else {
		message = t.operation + " completed with errors"
	}
	
	// Add timing and result fields
	if t.fields == nil {
		t.fields = make(Fields)
	}
	t.fields["operation"] = t.operation
	t.fields["duration_ms"] = float64(elapsed.Nanoseconds()) / 1000000
	t.fields["duration"] = elapsed.String()
	t.fields["success"] = success
	
	if result != nil {
		t.fields["result"] = result
	}
	
	// Log the result
	if t.logger != nil {
		level := t.level
		if !success && level < LevelWarn {
			level = LevelWarn
		}
		
		switch level {
		case LevelTrace:
			t.logger.Trace(message, t.fields)
		case LevelDebug:
			t.logger.Debug(message, t.fields)
		case LevelInfo:
			t.logger.Info(message, t.fields)
		case LevelWarn:
			t.logger.Warn(message, t.fields)
		case LevelError:
			t.logger.Error(message, t.fields)
		}
	}
	
	return elapsed
}

// Checkpoint logs an intermediate timing checkpoint
func (t *Timer) Checkpoint(name string, fields ...Fields) {
	if t.stopped {
		return
	}
	
	elapsed := t.Elapsed()
	message := t.operation + " checkpoint: " + name
	
	// Combine fields
	combinedFields := Fields{
		"operation":   t.operation,
		"checkpoint":  name,
		"elapsed_ms":  float64(elapsed.Nanoseconds()) / 1000000,
		"elapsed":     elapsed.String(),
	}
	
	// Add base fields
	for k, v := range t.fields {
		combinedFields[k] = v
	}
	
	// Add checkpoint fields
	for _, f := range fields {
		for k, v := range f {
			combinedFields[k] = v
		}
	}
	
	// Log the checkpoint
	if t.logger != nil {
		t.logger.Debug(message, combinedFields)
	}
}

// Cancel cancels the timer without logging completion
func (t *Timer) Cancel() {
	t.stopped = true
}

// IsRunning returns true if the timer is still running
func (t *Timer) IsRunning() bool {
	return !t.stopped
}

// Reset resets the timer to start counting from now
func (t *Timer) Reset() {
	t.startTime = time.Now()
	t.stopped = false
}

// StartTime returns the time when the timer was started
func (t *Timer) StartTime() time.Time {
	return t.startTime
}