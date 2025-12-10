// File: level.go
// Title: Log Level Definitions
// Description: Defines log levels for filtering and controlling log output.
//              Provides a structured approach to categorizing log messages by
//              importance and verbosity.
// Author: msto63 with Claude Sonnet 4.0
// Version: v0.1.0
// Created: 2025-01-24
// Modified: 2025-01-24
//
// Change History:
// - 2025-01-24 v0.1.0: Initial implementation with standard log levels

package log

import (
	"strings"
)

// Level represents the importance level of a log message
type Level int

const (
	// LevelTrace is the most verbose level, used for very detailed debugging
	// Should only be used in development environments
	LevelTrace Level = iota
	
	// LevelDebug provides detailed information for debugging purposes
	// Typically disabled in production
	LevelDebug
	
	// LevelInfo represents general informational messages
	// Standard level for normal operation logging
	LevelInfo
	
	// LevelWarn indicates potentially harmful situations
	// Operations continue but attention may be required
	LevelWarn
	
	// LevelError represents error conditions that need attention
	// Operations may fail but the system continues
	LevelError
	
	// LevelFatal represents critical errors that cause program termination
	// System cannot continue operating
	LevelFatal
	
	// LevelAudit represents audit trail events
	// Special level for compliance and security logging
	LevelAudit
)

// String returns the string representation of the log level
func (l Level) String() string {
	switch l {
	case LevelTrace:
		return "trace"
	case LevelDebug:
		return "debug"
	case LevelInfo:
		return "info"
	case LevelWarn:
		return "warn"
	case LevelError:
		return "error"
	case LevelFatal:
		return "fatal"
	case LevelAudit:
		return "audit"
	default:
		return "unknown"
	}
}

// ShortString returns a short string representation of the log level
func (l Level) ShortString() string {
	switch l {
	case LevelTrace:
		return "TRC"
	case LevelDebug:
		return "DBG"
	case LevelInfo:
		return "INF"
	case LevelWarn:
		return "WRN"
	case LevelError:
		return "ERR"
	case LevelFatal:
		return "FTL"
	case LevelAudit:
		return "AUD"
	default:
		return "???"
	}
}

// Color returns the ANSI color code for the log level (for console output)
func (l Level) Color() string {
	switch l {
	case LevelTrace:
		return "\033[37m" // White
	case LevelDebug:
		return "\033[36m" // Cyan
	case LevelInfo:
		return "\033[32m" // Green
	case LevelWarn:
		return "\033[33m" // Yellow
	case LevelError:
		return "\033[31m" // Red
	case LevelFatal:
		return "\033[35m" // Magenta
	case LevelAudit:
		return "\033[34m" // Blue
	default:
		return "\033[0m"  // Reset
	}
}

// Priority returns the numeric priority of the level (higher = more important)
func (l Level) Priority() int {
	return int(l)
}

// ShouldLog returns true if this level should be logged given the minimum level
func (l Level) ShouldLog(minLevel Level) bool {
	// Audit logs are always logged regardless of minimum level
	if l == LevelAudit {
		return true
	}
	return l >= minLevel
}

// IsEnabled returns true if the level is enabled for the given minimum level
func (l Level) IsEnabled(minLevel Level) bool {
	return l.ShouldLog(minLevel)
}

// ParseLevel parses a string into a log level
func ParseLevel(level string) (Level, error) {
	switch strings.ToLower(strings.TrimSpace(level)) {
	case "trace", "trc":
		return LevelTrace, nil
	case "debug", "dbg":
		return LevelDebug, nil
	case "info", "inf", "information":
		return LevelInfo, nil
	case "warn", "wrn", "warning":
		return LevelWarn, nil
	case "error", "err":
		return LevelError, nil
	case "fatal", "ftl":
		return LevelFatal, nil
	case "audit", "aud":
		return LevelAudit, nil
	default:
		return LevelInfo, &ParseError{
			Input: level,
			Type:  "level",
		}
	}
}

// ParseError represents an error parsing a log configuration value
type ParseError struct {
	Input string
	Type  string
}

// Error implements the error interface
func (e *ParseError) Error() string {
	return "invalid " + e.Type + ": " + e.Input
}

// AllLevels returns all available log levels
func AllLevels() []Level {
	return []Level{
		LevelTrace,
		LevelDebug,
		LevelInfo,
		LevelWarn,
		LevelError,
		LevelFatal,
		LevelAudit,
	}
}

// DefaultLevel returns the default log level for production
func DefaultLevel() Level {
	return LevelInfo
}

// DevelopmentLevel returns the default log level for development
func DevelopmentLevel() Level {
	return LevelDebug
}