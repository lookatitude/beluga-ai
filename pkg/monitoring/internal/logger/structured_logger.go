// Package logger provides structured logging implementations
package logger

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"time"

	"github.com/lookatitude/beluga-ai/pkg/monitoring/iface"
)

// StructuredLogger provides structured logging with context support
type StructuredLogger struct {
	name      string
	mutex     sync.Mutex
	level     LogLevel
	stdout    *log.Logger
	fileOut   *log.Logger
	file      *os.File
	useColors bool
	useJSON   bool
}

// LogEntry represents a structured log entry
type LogEntry struct {
	Timestamp string                 `json:"timestamp"`
	Level     string                 `json:"level"`
	Logger    string                 `json:"logger"`
	Message   string                 `json:"message"`
	Fields    map[string]interface{} `json:"fields,omitempty"`
	Caller    string                 `json:"caller,omitempty"`
	TraceID   string                 `json:"trace_id,omitempty"`
	SpanID    string                 `json:"span_id,omitempty"`
}

// LoggerOption represents functional options for logger configuration
type LoggerOption func(*StructuredLogger)

// Ensure StructuredLogger implements iface.Logger
var _ iface.Logger = (*StructuredLogger)(nil)

// WithJSONOutput enables JSON structured output
func WithJSONOutput() LoggerOption {
	return func(l *StructuredLogger) {
		l.useJSON = true
	}
}

// WithColors enables colored output
func WithColors(enabled bool) LoggerOption {
	return func(l *StructuredLogger) {
		l.useColors = enabled
	}
}

// WithFileOutput sets file output destination
func WithFileOutput(path string) LoggerOption {
	return func(l *StructuredLogger) {
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			fmt.Fprintf(os.Stderr, "Error creating log directory: %v\n", err)
			return
		}

		file, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error opening log file: %v\n", err)
			return
		}

		l.file = file
		l.fileOut = log.New(file, "", 0)
	}
}

// SetLevel sets the minimum log level
func (l *StructuredLogger) SetLevel(level LogLevel) {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	l.level = level
}

// log creates a log entry with the specified level and fields
func (l *StructuredLogger) log(ctx context.Context, level LogLevel, message string, fields map[string]interface{}) {
	if level < l.level {
		return
	}

	entry := l.createLogEntry(ctx, level, message, fields)
	output := l.formatEntry(entry)

	l.mutex.Lock()
	defer l.mutex.Unlock()

	if l.stdout != nil {
		if l.useColors && !l.useJSON {
			output = l.colorize(level, output)
		}
		l.stdout.Println(output)
	}

	if l.fileOut != nil {
		// Always use structured format for file output
		if l.useJSON {
			l.fileOut.Println(output)
		} else {
			// Use plain text for files if not JSON
			plainOutput := l.formatPlainText(entry)
			l.fileOut.Println(plainOutput)
		}
	}

	if level == FATAL {
		os.Exit(1)
	}
}

// createLogEntry creates a structured log entry
func (l *StructuredLogger) createLogEntry(ctx context.Context, level LogLevel, message string, fields map[string]interface{}) LogEntry {
	entry := LogEntry{
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Level:     level.String(),
		Logger:    l.name,
		Message:   message,
		Fields:    fields,
	}

	// Add caller information
	if _, file, line, ok := runtime.Caller(3); ok {
		entry.Caller = fmt.Sprintf("%s:%d", filepath.Base(file), line)
	}

	// Extract tracing information from context
	if ctx != nil {
		if traceID, ok := ctx.Value("trace_id").(string); ok {
			entry.TraceID = traceID
		}
		if spanID, ok := ctx.Value("span_id").(string); ok {
			entry.SpanID = spanID
		}
	}

	return entry
}

// formatEntry formats the log entry based on configuration
func (l *StructuredLogger) formatEntry(entry LogEntry) string {
	if l.useJSON {
		return l.formatJSON(entry)
	}
	return l.formatPlainText(entry)
}

// formatJSON formats the entry as JSON
func (l *StructuredLogger) formatJSON(entry LogEntry) string {
	data, err := json.Marshal(entry)
	if err != nil {
		return fmt.Sprintf(`{"error":"failed to marshal log entry","message":"%s"}`, entry.Message)
	}
	return string(data)
}

// formatPlainText formats the entry as plain text
func (l *StructuredLogger) formatPlainText(entry LogEntry) string {
	result := fmt.Sprintf("[%s] %s %s: %s",
		entry.Timestamp,
		entry.Level,
		entry.Logger,
		entry.Message)

	if entry.Caller != "" {
		result += fmt.Sprintf(" (%s)", entry.Caller)
	}

	if len(entry.Fields) > 0 {
		result += " |"
		for key, value := range entry.Fields {
			result += fmt.Sprintf(" %s=%v", key, value)
		}
	}

	if entry.TraceID != "" {
		result += fmt.Sprintf(" trace_id=%s", entry.TraceID)
	}

	return result
}

// colorize adds ANSI colors to plain text output
func (l *StructuredLogger) colorize(level LogLevel, text string) string {
	if !l.useColors {
		return text
	}

	var color string
	switch level {
	case DEBUG:
		color = "\033[37m" // White
	case INFO:
		color = "\033[32m" // Green
	case WARNING:
		color = "\033[33m" // Yellow
	case ERROR:
		color = "\033[31m" // Red
	case FATAL:
		color = "\033[35m" // Magenta
	default:
		return text
	}

	reset := "\033[0m"
	return color + text + reset
}

// Public logging methods with context support

// Debug logs a debug message with optional fields
func (l *StructuredLogger) Debug(ctx context.Context, message string, fields ...map[string]interface{}) {
	var fieldMap map[string]interface{}
	if len(fields) > 0 {
		fieldMap = fields[0]
	}
	l.log(ctx, DEBUG, message, fieldMap)
}

// Info logs an info message with optional fields
func (l *StructuredLogger) Info(ctx context.Context, message string, fields ...map[string]interface{}) {
	var fieldMap map[string]interface{}
	if len(fields) > 0 {
		fieldMap = fields[0]
	}
	l.log(ctx, INFO, message, fieldMap)
}

// Warning logs a warning message with optional fields
func (l *StructuredLogger) Warning(ctx context.Context, message string, fields ...map[string]interface{}) {
	var fieldMap map[string]interface{}
	if len(fields) > 0 {
		fieldMap = fields[0]
	}
	l.log(ctx, WARNING, message, fieldMap)
}

// Error logs an error message with optional fields
func (l *StructuredLogger) Error(ctx context.Context, message string, fields ...map[string]interface{}) {
	var fieldMap map[string]interface{}
	if len(fields) > 0 {
		fieldMap = fields[0]
	}
	l.log(ctx, ERROR, message, fieldMap)
}

// Fatal logs a fatal message and exits
func (l *StructuredLogger) Fatal(ctx context.Context, message string, fields ...map[string]interface{}) {
	var fieldMap map[string]interface{}
	if len(fields) > 0 {
		fieldMap = fields[0]
	}
	l.log(ctx, FATAL, message, fieldMap)
}

// WithFields returns a logger with additional context fields
func (l *StructuredLogger) WithFields(fields map[string]interface{}) iface.ContextLogger {
	return &ContextLogger{
		logger: l,
		fields: fields,
	}
}

// ContextLogger provides logging with persistent context fields
type ContextLogger struct {
	logger *StructuredLogger
	fields map[string]interface{}
}

// Ensure ContextLogger implements iface.ContextLogger
var _ iface.ContextLogger = (*ContextLogger)(nil)

// mergeFields merges context fields with additional fields
func (cl *ContextLogger) mergeFields(additional map[string]interface{}) map[string]interface{} {
	if cl.fields == nil {
		return additional
	}

	merged := make(map[string]interface{})
	for k, v := range cl.fields {
		merged[k] = v
	}

	if additional != nil {
		for k, v := range additional {
			merged[k] = v
		}
	}

	return merged
}

// Debug logs with context fields
func (cl *ContextLogger) Debug(ctx context.Context, message string, fields ...map[string]interface{}) {
	var additional map[string]interface{}
	if len(fields) > 0 {
		additional = fields[0]
	}
	cl.logger.log(ctx, DEBUG, message, cl.mergeFields(additional))
}

// Info logs with context fields
func (cl *ContextLogger) Info(ctx context.Context, message string, fields ...map[string]interface{}) {
	var additional map[string]interface{}
	if len(fields) > 0 {
		additional = fields[0]
	}
	cl.logger.log(ctx, INFO, message, cl.mergeFields(additional))
}

// Error logs with context fields
func (cl *ContextLogger) Error(ctx context.Context, message string, fields ...map[string]interface{}) {
	var additional map[string]interface{}
	if len(fields) > 0 {
		additional = fields[0]
	}
	cl.logger.log(ctx, ERROR, message, cl.mergeFields(additional))
}

// Close closes the logger
func (l *StructuredLogger) Close() error {
	l.mutex.Lock()
	defer l.mutex.Unlock()

	if l.file != nil {
		return l.file.Close()
	}
	return nil
}

// GetWriter returns an io.Writer for the specified log level
func (l *StructuredLogger) GetWriter(level LogLevel) io.Writer {
	return &structuredLogWriter{
		logger: l,
		level:  level,
	}
}

// structuredLogWriter implements io.Writer for structured logging
type structuredLogWriter struct {
	logger *StructuredLogger
	level  LogLevel
}

func (w *structuredLogWriter) Write(p []byte) (n int, err error) {
	w.logger.log(context.Background(), w.level, string(p), nil)
	return len(p), nil
}

// NewStructuredLogger creates a new structured logger with the given name and options
func NewStructuredLogger(name string, opts ...LoggerOption) iface.Logger {
	logger := &StructuredLogger{
		name:      name,
		level:     INFO, // Default level
		stdout:    log.New(os.Stdout, "", 0),
		useColors: true,
		useJSON:   false,
	}

	for _, opt := range opts {
		opt(logger)
	}

	return logger
}
