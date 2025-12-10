// File: format.go
// Title: Log Format Definitions
// Description: Defines output formats for log messages including JSON, text,
//              and console formats. Provides formatters for different output
//              destinations and use cases.
// Author: msto63 with Claude Sonnet 4.0
// Version: v0.1.0
// Created: 2025-01-24
// Modified: 2025-01-24
//
// Change History:
// - 2025-01-24 v0.1.0: Initial implementation with multiple output formats

package log

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// Format represents the output format for log messages
type Format int

const (
	// FormatJSON outputs structured JSON logs (recommended for production)
	FormatJSON Format = iota
	
	// FormatText outputs human-readable text logs
	FormatText
	
	// FormatConsole outputs colored console logs for development
	FormatConsole
	
	// FormatLogfmt outputs logfmt structured logs (key=value pairs)
	FormatLogfmt
)

// String returns the string representation of the format
func (f Format) String() string {
	switch f {
	case FormatJSON:
		return "json"
	case FormatText:
		return "text"
	case FormatConsole:
		return "console"
	case FormatLogfmt:
		return "logfmt"
	default:
		return "unknown"
	}
}

// ParseFormat parses a string into a log format
func ParseFormat(format string) (Format, error) {
	switch strings.ToLower(strings.TrimSpace(format)) {
	case "json":
		return FormatJSON, nil
	case "text":
		return FormatText, nil
	case "console":
		return FormatConsole, nil
	case "logfmt":
		return FormatLogfmt, nil
	default:
		return FormatJSON, &ParseError{
			Input: format,
			Type:  "format",
		}
	}
}

// Formatter defines the interface for log formatters
type Formatter interface {
	Format(entry *Entry) ([]byte, error)
}

// JSONFormatter formats log entries as JSON
type JSONFormatter struct {
	// PrettyPrint enables indented JSON output
	PrettyPrint bool
	
	// TimestampFormat specifies the timestamp format
	TimestampFormat string
}

// NewJSONFormatter creates a new JSON formatter
func NewJSONFormatter() *JSONFormatter {
	return &JSONFormatter{
		PrettyPrint:     false,
		TimestampFormat: time.RFC3339,
	}
}

// Format formats a log entry as JSON
func (f *JSONFormatter) Format(entry *Entry) ([]byte, error) {
	data := make(map[string]interface{})
	
	// Standard fields
	data["timestamp"] = entry.Timestamp.Format(f.TimestampFormat)
	data["level"] = entry.Level.String()
	data["message"] = entry.Message
	
	// Context fields
	if entry.Logger != "" {
		data["logger"] = entry.Logger
	}
	
	if entry.RequestID != "" {
		data["request_id"] = entry.RequestID
	}
	
	if entry.UserID != "" {
		data["user_id"] = entry.UserID
	}
	
	if entry.CorrelationID != "" {
		data["correlation_id"] = entry.CorrelationID
	}
	
	// Custom fields
	for k, v := range entry.Fields {
		data[k] = v
	}
	
	// Error information
	if entry.Error != nil {
		data["error"] = entry.Error.Error()
		// If it's a mDW error, include additional context
		if mdwErr, ok := entry.Error.(interface{ MarshalJSON() ([]byte, error) }); ok {
			if errData, err := mdwErr.MarshalJSON(); err == nil {
				var errorObj map[string]interface{}
				if json.Unmarshal(errData, &errorObj) == nil {
					data["error_details"] = errorObj
				}
			}
		}
	}
	
	// Performance metrics
	if entry.Duration > 0 {
		data["duration_ms"] = float64(entry.Duration.Nanoseconds()) / 1000000
	}
	
	if f.PrettyPrint {
		return json.MarshalIndent(data, "", "  ")
	}
	
	return json.Marshal(data)
}

// TextFormatter formats log entries as human-readable text
type TextFormatter struct {
	// TimestampFormat specifies the timestamp format
	TimestampFormat string
	
	// FullTimestamp enables full timestamp instead of just time
	FullTimestamp bool
	
	// DisableTimestamp disables timestamp output
	DisableTimestamp bool
}

// NewTextFormatter creates a new text formatter
func NewTextFormatter() *TextFormatter {
	return &TextFormatter{
		TimestampFormat: "15:04:05",
		FullTimestamp:   false,
		DisableTimestamp: false,
	}
}

// Format formats a log entry as text
func (f *TextFormatter) Format(entry *Entry) ([]byte, error) {
	var parts []string
	
	// Timestamp
	if !f.DisableTimestamp {
		timestampFormat := f.TimestampFormat
		if f.FullTimestamp {
			timestampFormat = time.RFC3339
		}
		parts = append(parts, entry.Timestamp.Format(timestampFormat))
	}
	
	// Level
	parts = append(parts, fmt.Sprintf("[%s]", strings.ToUpper(entry.Level.ShortString())))
	
	// Logger name
	if entry.Logger != "" {
		parts = append(parts, fmt.Sprintf("{%s}", entry.Logger))
	}
	
	// Request/User context
	var contextParts []string
	if entry.RequestID != "" {
		contextParts = append(contextParts, fmt.Sprintf("req=%s", entry.RequestID))
	}
	if entry.UserID != "" {
		contextParts = append(contextParts, fmt.Sprintf("user=%s", entry.UserID))
	}
	if len(contextParts) > 0 {
		parts = append(parts, fmt.Sprintf("(%s)", strings.Join(contextParts, ",")))
	}
	
	// Message
	parts = append(parts, entry.Message)
	
	// Fields
	if len(entry.Fields) > 0 {
		var fieldParts []string
		for k, v := range entry.Fields {
			fieldParts = append(fieldParts, fmt.Sprintf("%s=%v", k, v))
		}
		parts = append(parts, fmt.Sprintf("[%s]", strings.Join(fieldParts, " ")))
	}
	
	// Error
	if entry.Error != nil {
		parts = append(parts, fmt.Sprintf("error=\"%s\"", entry.Error.Error()))
	}
	
	// Duration
	if entry.Duration > 0 {
		parts = append(parts, fmt.Sprintf("duration=%s", entry.Duration))
	}
	
	return []byte(strings.Join(parts, " ") + "\n"), nil
}

// ConsoleFormatter formats log entries for console output with colors
type ConsoleFormatter struct {
	// DisableColors disables color output
	DisableColors bool
	
	// TextFormatter embedded for basic formatting
	*TextFormatter
}

// NewConsoleFormatter creates a new console formatter
func NewConsoleFormatter() *ConsoleFormatter {
	return &ConsoleFormatter{
		DisableColors: false,
		TextFormatter: NewTextFormatter(),
	}
}

// Format formats a log entry for console output
func (f *ConsoleFormatter) Format(entry *Entry) ([]byte, error) {
	// Get basic text format
	data, err := f.TextFormatter.Format(entry)
	if err != nil {
		return nil, err
	}
	
	// Apply colors if enabled
	if !f.DisableColors {
		level := entry.Level
		colorCode := level.Color()
		resetCode := "\033[0m"
		
		// Colorize the entire line
		colored := fmt.Sprintf("%s%s%s", colorCode, strings.TrimSpace(string(data)), resetCode)
		return []byte(colored + "\n"), nil
	}
	
	return data, nil
}

// LogfmtFormatter formats log entries in logfmt format (key=value pairs)
type LogfmtFormatter struct {
	// TimestampFormat specifies the timestamp format
	TimestampFormat string
}

// NewLogfmtFormatter creates a new logfmt formatter
func NewLogfmtFormatter() *LogfmtFormatter {
	return &LogfmtFormatter{
		TimestampFormat: time.RFC3339,
	}
}

// Format formats a log entry in logfmt format
func (f *LogfmtFormatter) Format(entry *Entry) ([]byte, error) {
	var parts []string
	
	// Standard fields
	parts = append(parts, fmt.Sprintf("timestamp=%s", entry.Timestamp.Format(f.TimestampFormat)))
	parts = append(parts, fmt.Sprintf("level=%s", entry.Level.String()))
	parts = append(parts, fmt.Sprintf("message=%q", entry.Message))
	
	// Context fields
	if entry.Logger != "" {
		parts = append(parts, fmt.Sprintf("logger=%s", entry.Logger))
	}
	
	if entry.RequestID != "" {
		parts = append(parts, fmt.Sprintf("request_id=%s", entry.RequestID))
	}
	
	if entry.UserID != "" {
		parts = append(parts, fmt.Sprintf("user_id=%s", entry.UserID))
	}
	
	if entry.CorrelationID != "" {
		parts = append(parts, fmt.Sprintf("correlation_id=%s", entry.CorrelationID))
	}
	
	// Custom fields
	for k, v := range entry.Fields {
		if str, ok := v.(string); ok {
			parts = append(parts, fmt.Sprintf("%s=%q", k, str))
		} else {
			parts = append(parts, fmt.Sprintf("%s=%v", k, v))
		}
	}
	
	// Error
	if entry.Error != nil {
		parts = append(parts, fmt.Sprintf("error=%q", entry.Error.Error()))
	}
	
	// Duration
	if entry.Duration > 0 {
		parts = append(parts, fmt.Sprintf("duration_ms=%.3f", float64(entry.Duration.Nanoseconds())/1000000))
	}
	
	return []byte(strings.Join(parts, " ") + "\n"), nil
}

// GetFormatter returns a formatter for the specified format
func GetFormatter(format Format) Formatter {
	switch format {
	case FormatJSON:
		return NewJSONFormatter()
	case FormatText:
		return NewTextFormatter()
	case FormatConsole:
		return NewConsoleFormatter()
	case FormatLogfmt:
		return NewLogfmtFormatter()
	default:
		return NewJSONFormatter()
	}
}