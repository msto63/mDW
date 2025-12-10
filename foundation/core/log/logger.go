// File: logger.go
// Title: Core Logger Implementation
// Description: Implements the main Logger type that provides structured logging
//              with contextual information, multiple output formats, and
//              integration with the mDW error system.
// Author: msto63 with Claude Sonnet 4.0
// Version: v0.1.0
// Created: 2025-01-24
// Modified: 2025-01-24
//
// Change History:
// - 2025-01-24 v0.1.0: Initial implementation with structured logging

package log

import (
	"io"
	"os"
	"runtime"
	"strings"
	"sync"
	
	mdwerror "github.com/msto63/mDW/foundation/core/error"
)

// Logger represents a structured logger with contextual information
type Logger struct {
	// Configuration
	level     Level
	formatter Formatter
	output    io.Writer
	name      string
	
	// Context fields that are added to all log entries
	contextFields Fields
	requestID     string
	userID        string
	correlationID string
	
	// Options
	enableCaller    bool
	callerSkipFrames int
	
	// Async logging support
	asyncEnabled bool
	asyncBuffer  chan *Entry
	asyncDone    chan struct{}
	asyncOnce    sync.Once
	
	// Thread safety
	mutex sync.RWMutex
}

// Config represents logger configuration
type Config struct {
	Level           Level
	Format          Format
	Output          io.Writer
	Name            string
	EnableCaller    bool
	CallerSkipFrames int
	AsyncEnabled    bool
	AsyncBufferSize int
}

// New creates a new logger with default configuration
func New() *Logger {
	return &Logger{
		level:            DefaultLevel(),
		formatter:        NewJSONFormatter(),
		output:           os.Stdout,
		contextFields:    make(Fields),
		enableCaller:     false,
		callerSkipFrames: 0,
		mutex:           sync.RWMutex{},
	}
}

// NewWithConfig creates a new logger with the specified configuration
func NewWithConfig(config Config) *Logger {
	logger := &Logger{
		level:            config.Level,
		output:           config.Output,
		name:             config.Name,
		contextFields:    make(Fields),
		enableCaller:     config.EnableCaller,
		callerSkipFrames: config.CallerSkipFrames,
		asyncEnabled:     config.AsyncEnabled,
		mutex:           sync.RWMutex{},
	}
	
	if config.Output == nil {
		logger.output = os.Stdout
	}
	
	logger.formatter = GetFormatter(config.Format)
	
	// Initialize async logging if enabled
	if config.AsyncEnabled {
		bufferSize := config.AsyncBufferSize
		if bufferSize <= 0 {
			bufferSize = 1000 // Default buffer size
		}
		logger.asyncBuffer = make(chan *Entry, bufferSize)
		logger.asyncDone = make(chan struct{})
		logger.startAsyncWorker()
	}
	
	return logger
}

// WithLevel sets the minimum log level
func (l *Logger) WithLevel(level Level) *Logger {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	
	clone := l.clone()
	clone.level = level
	return clone
}

// WithFormat sets the log format
func (l *Logger) WithFormat(format Format) *Logger {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	
	clone := l.clone()
	clone.formatter = GetFormatter(format)
	return clone
}

// WithOutput sets the output destination
func (l *Logger) WithOutput(output io.Writer) *Logger {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	
	clone := l.clone()
	clone.output = output
	return clone
}

// WithName sets the logger name
func (l *Logger) WithName(name string) *Logger {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	
	clone := l.clone()
	clone.name = name
	return clone
}

// WithField adds a persistent field to all log entries
func (l *Logger) WithField(key string, value interface{}) *Logger {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	
	clone := l.clone()
	clone.contextFields[key] = value
	return clone
}

// WithFields adds persistent fields to all log entries
func (l *Logger) WithFields(fields Fields) *Logger {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	
	clone := l.clone()
	for k, v := range fields {
		clone.contextFields[k] = v
	}
	return clone
}

// WithRequestID sets the request ID context
func (l *Logger) WithRequestID(requestID string) *Logger {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	
	clone := l.clone()
	clone.requestID = requestID
	return clone
}

// WithUserID sets the user ID context
func (l *Logger) WithUserID(userID string) *Logger {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	
	clone := l.clone()
	clone.userID = userID
	return clone
}

// WithCorrelationID sets the correlation ID context
func (l *Logger) WithCorrelationID(correlationID string) *Logger {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	
	clone := l.clone()
	clone.correlationID = correlationID
	return clone
}

// WithCaller enables caller information in log entries
func (l *Logger) WithCaller(skip int) *Logger {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	
	clone := l.clone()
	clone.enableCaller = true
	clone.callerSkipFrames = skip
	return clone
}

// Trace logs a trace level message
func (l *Logger) Trace(message string, fields ...Fields) {
	l.log(LevelTrace, message, nil, fields...)
}

// Debug logs a debug level message
func (l *Logger) Debug(message string, fields ...Fields) {
	l.log(LevelDebug, message, nil, fields...)
}

// Info logs an info level message
func (l *Logger) Info(message string, fields ...Fields) {
	l.log(LevelInfo, message, nil, fields...)
}

// Warn logs a warning level message
func (l *Logger) Warn(message string, fields ...Fields) {
	l.log(LevelWarn, message, nil, fields...)
}

// Error logs an error level message
func (l *Logger) Error(message string, fields ...Fields) {
	l.log(LevelError, message, nil, fields...)
}

// Fatal logs a fatal level message and exits the program
func (l *Logger) Fatal(message string, fields ...Fields) {
	l.log(LevelFatal, message, nil, fields...)
	os.Exit(1)
}

// Audit logs an audit level message (always logged regardless of level)
func (l *Logger) Audit(message string, fields ...Fields) {
	l.log(LevelAudit, message, nil, fields...)
}

// ErrorWithErr logs an error with an error object
func (l *Logger) ErrorWithErr(message string, err error, fields ...Fields) {
	l.log(LevelError, message, err, fields...)
}

// WarnWithErr logs a warning with an error object
func (l *Logger) WarnWithErr(message string, err error, fields ...Fields) {
	l.log(LevelWarn, message, err, fields...)
}

// LogError logs a mDW error with full context
func (l *Logger) LogError(err error) {
	if err == nil {
		return
	}
	
	// Extract additional fields if it's a mDW error
	var fields Fields
	if mdwErr, ok := err.(*mdwerror.Error); ok {
		fields = Fields{
			"error_code":      mdwErr.Code(),
			"error_severity":  mdwErr.Severity().String(),
			"error_context":   mdwErr.Context(),
			"error_operation": mdwErr.Operation(),
		}
		
		// Add error details
		for k, v := range mdwErr.Details() {
			fields["error_"+k] = v
		}
		
		// Use appropriate log level based on error severity
		switch mdwErr.Severity() {
		case mdwerror.SeverityLow:
			l.log(LevelInfo, err.Error(), err, fields)
		case mdwerror.SeverityMedium:
			l.log(LevelWarn, err.Error(), err, fields)
		case mdwerror.SeverityHigh:
			l.log(LevelError, err.Error(), err, fields)
		case mdwerror.SeverityCritical:
			l.log(LevelError, err.Error(), err, fields)
		default:
			l.log(LevelError, err.Error(), err, fields)
		}
	} else {
		// Standard error
		l.log(LevelError, err.Error(), err)
	}
}

// StartTimer creates and starts a new performance timer
func (l *Logger) StartTimer(operation string) *Timer {
	return NewTimer(l, operation)
}

// IsLevelEnabled returns true if the given level is enabled
func (l *Logger) IsLevelEnabled(level Level) bool {
	l.mutex.RLock()
	defer l.mutex.RUnlock()
	
	return level.ShouldLog(l.level)
}

// GetLevel returns the current log level
func (l *Logger) GetLevel() Level {
	l.mutex.RLock()
	defer l.mutex.RUnlock()
	
	return l.level
}

// SetLevel sets the log level (not thread-safe, use WithLevel for safety)
func (l *Logger) SetLevel(level Level) {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	
	l.level = level
}

// log is the internal logging method
func (l *Logger) log(level Level, message string, err error, fields ...Fields) {
	l.mutex.RLock()
	
	// Check if level is enabled
	if !level.ShouldLog(l.level) {
		l.mutex.RUnlock()
		return
	}
	
	// Create entry
	entry := NewEntry(level, message)
	entry.Logger = l.name
	entry.RequestID = l.requestID
	entry.UserID = l.userID
	entry.CorrelationID = l.correlationID
	entry.Error = err
	
	// Add context fields
	for k, v := range l.contextFields {
		entry.Fields[k] = v
	}
	
	// Add provided fields
	for _, fieldSet := range fields {
		for k, v := range fieldSet {
			entry.Fields[k] = v
		}
	}
	
	// Add caller information if enabled
	if l.enableCaller {
		if function, file, line, ok := l.getCaller(); ok {
			entry.WithCaller(function, file, line)
		}
	}
	
	// Check if async logging is enabled
	if l.asyncEnabled && l.asyncBuffer != nil {
		// Send to async buffer (non-blocking)
		select {
		case l.asyncBuffer <- entry:
			// Successfully queued
		default:
			// Buffer full, fallback to synchronous logging
			formatter := l.formatter
			output := l.output
			l.mutex.RUnlock()
			
			if formatted, formatErr := formatter.Format(entry); formatErr == nil {
				output.Write(formatted)
			}
			return
		}
		l.mutex.RUnlock()
		return
	}
	
	// Synchronous logging
	formatter := l.formatter
	output := l.output
	l.mutex.RUnlock()
	
	// Format and write the log entry
	if formatted, formatErr := formatter.Format(entry); formatErr == nil {
		output.Write(formatted)
	}
}

// getCaller returns caller information
func (l *Logger) getCaller() (function, file string, line int, ok bool) {
	// Skip frames: getCaller, log, public method, user code
	skip := 3 + l.callerSkipFrames
	
	pc, file, line, ok := runtime.Caller(skip)
	if !ok {
		return "", "", 0, false
	}
	
	function = "unknown"
	if fn := runtime.FuncForPC(pc); fn != nil {
		function = fn.Name()
		// Simplify function name
		if idx := strings.LastIndex(function, "."); idx != -1 {
			function = function[idx+1:]
		}
	}
	
	// Simplify file path
	if idx := strings.LastIndex(file, "/"); idx != -1 {
		file = file[idx+1:]
	}
	
	return function, file, line, true
}

// clone creates a copy of the logger for immutable operations
func (l *Logger) clone() *Logger {
	clone := &Logger{
		level:            l.level,
		formatter:        l.formatter,
		output:           l.output,
		name:             l.name,
		requestID:        l.requestID,
		userID:           l.userID,
		correlationID:    l.correlationID,
		enableCaller:     l.enableCaller,
		callerSkipFrames: l.callerSkipFrames,
		contextFields:    make(Fields),
		mutex:           sync.RWMutex{},
	}
	
	// Copy context fields
	for k, v := range l.contextFields {
		clone.contextFields[k] = v
	}
	
	return clone
}

// startAsyncWorker starts the background goroutine for async logging
func (l *Logger) startAsyncWorker() {
	l.asyncOnce.Do(func() {
		go l.asyncWorker()
	})
}

// asyncWorker processes log entries from the async buffer
func (l *Logger) asyncWorker() {
	for {
		select {
		case entry := <-l.asyncBuffer:
			// Process the log entry synchronously in the worker
			l.mutex.RLock()
			formatter := l.formatter
			output := l.output
			l.mutex.RUnlock()
			
			if formatted, formatErr := formatter.Format(entry); formatErr == nil {
				output.Write(formatted)
			}
			
		case <-l.asyncDone:
			// Drain remaining entries before shutting down
			for {
				select {
				case entry := <-l.asyncBuffer:
					l.mutex.RLock()
					formatter := l.formatter
					output := l.output
					l.mutex.RUnlock()
					
					if formatted, formatErr := formatter.Format(entry); formatErr == nil {
						output.Write(formatted)
					}
				default:
					return
				}
			}
		}
	}
}

// Close gracefully shuts down async logging and flushes remaining entries
func (l *Logger) Close() {
	if l.asyncEnabled && l.asyncDone != nil {
		close(l.asyncDone)
	}
}

// Default logger instance
var defaultLogger = New()

// GetDefault returns the default logger instance
func GetDefault() *Logger {
	return defaultLogger
}

// SetDefault sets the default logger instance
func SetDefault(logger *Logger) {
	defaultLogger = logger
}

// Global convenience functions using the default logger

// Trace logs a trace message using the default logger
func Trace(message string, fields ...Fields) {
	defaultLogger.Trace(message, fields...)
}

// Debug logs a debug message using the default logger
func Debug(message string, fields ...Fields) {
	defaultLogger.Debug(message, fields...)
}

// Info logs an info message using the default logger
func Info(message string, fields ...Fields) {
	defaultLogger.Info(message, fields...)
}

// Warn logs a warning message using the default logger
func Warn(message string, fields ...Fields) {
	defaultLogger.Warn(message, fields...)
}

// Error logs an error message using the default logger
func Error(message string, fields ...Fields) {
	defaultLogger.Error(message, fields...)
}

// Fatal logs a fatal message using the default logger and exits
func Fatal(message string, fields ...Fields) {
	defaultLogger.Fatal(message, fields...)
}

// Audit logs an audit message using the default logger
func Audit(message string, fields ...Fields) {
	defaultLogger.Audit(message, fields...)
}