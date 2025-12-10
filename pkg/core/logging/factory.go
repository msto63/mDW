// ============================================================================
// meinDENKWERK (mDW) - Lokale KI-Plattform
// ============================================================================
//
// Package:     logging
// Description: Factory functions for creating loggers with Bayes integration
// Author:      Mike Stoffels with Claude
// Created:     2025-12-06
// License:     MIT
// ============================================================================

package logging

import (
	"io"
	"os"
	"sync"
	"time"

	mdwlog "github.com/msto63/mDW/foundation/core/log"
)

var (
	// Global BayesWriter instance (singleton)
	globalBayesWriter *BayesWriter
	bayesWriterOnce   sync.Once
	bayesWriterMu     sync.RWMutex
)

// LoggerConfig holds configuration for creating loggers
type LoggerConfig struct {
	// Service name
	ServiceName string

	// Log level (debug, info, warn, error)
	Level string

	// Bayes configuration (optional)
	BayesAddress string // If empty, Bayes integration is disabled

	// Output format
	Format string // "json" or "text" (default: json)

	// Additional outputs (besides stdout and Bayes)
	AdditionalOutputs []io.Writer
}

// DefaultLoggerConfig returns a default configuration
func DefaultLoggerConfig(serviceName string) LoggerConfig {
	return LoggerConfig{
		ServiceName: serviceName,
		Level:       "info",
		Format:      "json",
	}
}

// NewLogger creates a new Foundation logger with optional Bayes integration
func NewLogger(cfg LoggerConfig) *mdwlog.Logger {
	// Determine log level
	level := parseLevel(cfg.Level)

	// Build output writer
	var output io.Writer = os.Stdout

	// Add Bayes writer if configured
	if cfg.BayesAddress != "" {
		bayesWriter := getOrCreateBayesWriter(cfg.BayesAddress, cfg.ServiceName)
		if bayesWriter != nil {
			// BayesWriter already writes to stdout internally, so just use it
			output = bayesWriter
		}
	}

	// Add additional outputs if specified
	if len(cfg.AdditionalOutputs) > 0 {
		writers := append([]io.Writer{output}, cfg.AdditionalOutputs...)
		output = io.MultiWriter(writers...)
	}

	// Determine format
	format := mdwlog.FormatJSON
	if cfg.Format == "text" {
		format = mdwlog.FormatText
	}

	// Create logger
	logger := mdwlog.NewWithConfig(mdwlog.Config{
		Level:        level,
		Format:       format,
		Output:       output,
		Name:         cfg.ServiceName,
		EnableCaller: true,
	})

	return logger
}

// NewServiceLogger creates a logger for a service with standard configuration
func NewServiceLogger(serviceName string, bayesAddress string) *mdwlog.Logger {
	cfg := DefaultLoggerConfig(serviceName)
	cfg.BayesAddress = bayesAddress
	return NewLogger(cfg)
}

// NewSimpleLogger creates a simple logger without Bayes integration
func NewSimpleLogger(serviceName string) *mdwlog.Logger {
	return NewLogger(DefaultLoggerConfig(serviceName))
}

// getOrCreateBayesWriter returns the global BayesWriter, creating it if necessary
func getOrCreateBayesWriter(address string, serviceName string) *BayesWriter {
	bayesWriterOnce.Do(func() {
		writer, err := NewBayesWriter(BayesWriterConfig{
			Address:     address,
			ServiceName: serviceName,
			BatchSize:   100,
			FlushPeriod: 5 * time.Second,
			Fallback:    os.Stdout,
		})
		if err != nil {
			return
		}
		globalBayesWriter = writer
	})

	return globalBayesWriter
}

// GetGlobalBayesWriter returns the global BayesWriter instance
func GetGlobalBayesWriter() *BayesWriter {
	bayesWriterMu.RLock()
	defer bayesWriterMu.RUnlock()
	return globalBayesWriter
}

// CloseGlobalBayesWriter closes the global BayesWriter
func CloseGlobalBayesWriter() error {
	bayesWriterMu.Lock()
	defer bayesWriterMu.Unlock()

	if globalBayesWriter != nil {
		err := globalBayesWriter.Close()
		globalBayesWriter = nil
		return err
	}
	return nil
}

// parseLevel converts a string level to mdwlog.Level
func parseLevel(level string) mdwlog.Level {
	switch level {
	case "trace":
		return mdwlog.LevelTrace
	case "debug":
		return mdwlog.LevelDebug
	case "info":
		return mdwlog.LevelInfo
	case "warn", "warning":
		return mdwlog.LevelWarn
	case "error":
		return mdwlog.LevelError
	case "fatal":
		return mdwlog.LevelFatal
	default:
		return mdwlog.LevelInfo
	}
}

// Compatibility layer for existing code using the simple Logger

// Logger wraps the Foundation logger for compatibility
type Logger struct {
	*mdwlog.Logger
	name string
}

// New creates a new simple logger (compatibility with existing code)
func New(name string) *Logger {
	return &Logger{
		Logger: NewSimpleLogger(name),
		name:   name,
	}
}

// WithLevel returns a new logger with the specified level (compatibility)
func (l *Logger) WithLevel(level Level) *Logger {
	mdwLevel := mdwlog.LevelInfo
	switch level {
	case LevelDebug:
		mdwLevel = mdwlog.LevelDebug
	case LevelInfo:
		mdwLevel = mdwlog.LevelInfo
	case LevelWarn:
		mdwLevel = mdwlog.LevelWarn
	case LevelError:
		mdwLevel = mdwlog.LevelError
	}

	return &Logger{
		Logger: l.Logger.WithLevel(mdwLevel),
		name:   l.name,
	}
}

// Debug logs a debug message (compatibility with key-value pairs)
func (l *Logger) Debug(msg string, keysAndValues ...interface{}) {
	l.Logger.Debug(msg, toFields(keysAndValues...))
}

// Info logs an info message (compatibility with key-value pairs)
func (l *Logger) Info(msg string, keysAndValues ...interface{}) {
	l.Logger.Info(msg, toFields(keysAndValues...))
}

// Warn logs a warning message (compatibility with key-value pairs)
func (l *Logger) Warn(msg string, keysAndValues ...interface{}) {
	l.Logger.Warn(msg, toFields(keysAndValues...))
}

// Error logs an error message (compatibility with key-value pairs)
func (l *Logger) Error(msg string, keysAndValues ...interface{}) {
	l.Logger.Error(msg, toFields(keysAndValues...))
}

// toFields converts key-value pairs to mdwlog.Fields
func toFields(keysAndValues ...interface{}) mdwlog.Fields {
	if len(keysAndValues) == 0 {
		return nil
	}

	fields := make(mdwlog.Fields)
	for i := 0; i < len(keysAndValues)-1; i += 2 {
		key, ok := keysAndValues[i].(string)
		if !ok {
			continue
		}
		fields[key] = keysAndValues[i+1]
	}
	return fields
}
