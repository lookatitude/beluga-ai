package monitoring

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"sync"
)

// LogLevel represents the severity of a log entry.
type LogLevel int

const (
	// Log levels
	DEBUG LogLevel = iota
	INFO
	WARNING
	ERROR
	FATAL
)

// String returns the string representation of the log level.
func (l LogLevel) String() string {
	switch l {
	case DEBUG:
		return "DEBUG"
	case INFO:
		return "INFO"
	case WARNING:
		return "WARN"
	case ERROR:
		return "ERROR"
	case FATAL:
		return "FATAL"
	default:
		return "UNKNOWN"
	}
}

// Logger provides a standardized logging mechanism.
type Logger struct {
	name      string
	mutex     sync.Mutex
	level     LogLevel
	stdout    *log.Logger
	fileOut   *log.Logger
	file      *os.File
	useColors bool
}

// LoggerConfig contains configuration options for creating a logger.
type LoggerConfig struct {
	Level       LogLevel
	EnableColor bool
	OutputFile  string
	UseConsole  bool
}

// DefaultLoggerConfig returns a default configuration for the logger.
func DefaultLoggerConfig() LoggerConfig {
	return LoggerConfig{
		Level:       INFO,
		EnableColor: true,
		UseConsole:  true,
	}
}

// NewLogger creates a new instance of Logger with the given name.
func NewLogger(name string) *Logger {
	config := DefaultLoggerConfig()
	return NewLoggerWithConfig(name, config)
}

// NewLoggerWithConfig creates a new instance of Logger with the given name and configuration.
func NewLoggerWithConfig(name string, config LoggerConfig) *Logger {
	logger := &Logger{
		name:      name,
		level:     config.Level,
		stdout:    log.New(os.Stdout, "", 0),
		useColors: config.EnableColor,
	}

	if config.OutputFile != "" {
		if err := os.MkdirAll(filepath.Dir(config.OutputFile), 0755); err != nil {
			fmt.Fprintf(os.Stderr, "Error creating log directory: %v\n", err)
		} else {
			file, err := os.OpenFile(config.OutputFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error opening log file: %v\n", err)
			} else {
				logger.file = file
				logger.fileOut = log.New(file, "", log.LstdFlags)
			}
		}
	}

	if !config.UseConsole {
		logger.stdout = nil
	}

	return logger
}

// formatLogMessage formats a log entry with timestamp, level, and caller information.
func (l *Logger) formatLogMessage(level LogLevel, message string, args ...interface{}) string {
	var caller string
	if _, file, line, ok := runtime.Caller(2); ok {
		caller = fmt.Sprintf("%s:%d", filepath.Base(file), line)
	} else {
		caller = "unknown:0"
	}

	prefix := fmt.Sprintf("[%s][%s][%s] ", l.name, level.String(), caller)

	// Format the message if there are arguments
	if len(args) > 0 {
		message = fmt.Sprintf(message, args...)
	}

	return prefix + message
}

// colorize adds ANSI color to a string based on log level.
func (l *Logger) colorize(level LogLevel, text string) string {
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

// log outputs a log entry to stdout and file if enabled.
func (l *Logger) log(level LogLevel, message string, args ...interface{}) {
	if level < l.level {
		return
	}

	l.mutex.Lock()
	defer l.mutex.Unlock()

	formattedMsg := l.formatLogMessage(level, message, args...)

	if l.stdout != nil {
		colorizedMsg := l.colorize(level, formattedMsg)
		l.stdout.Println(colorizedMsg)
	}

	if l.fileOut != nil {
		l.fileOut.Println(formattedMsg)
	}

	if level == FATAL {
		os.Exit(1)
	}
}

// Debug logs debug information.
func (l *Logger) Debug(message string, args ...interface{}) {
	l.log(DEBUG, message, args...)
}

// Info logs informational messages.
func (l *Logger) Info(message string, args ...interface{}) {
	l.log(INFO, message, args...)
}

// Warning logs warning messages.
func (l *Logger) Warning(message string, args ...interface{}) {
	l.log(WARNING, message, args...)
}

// Error logs error messages.
func (l *Logger) Error(message string, args ...interface{}) {
	l.log(ERROR, message, args...)
}

// Fatal logs fatal messages and exits the application with status code 1.
func (l *Logger) Fatal(message string, args ...interface{}) {
	l.log(FATAL, message, args...)
}

// SetLevel changes the current log level.
func (l *Logger) SetLevel(level LogLevel) {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	l.level = level
}

// Close closes the logger's file if one is open.
func (l *Logger) Close() error {
	l.mutex.Lock()
	defer l.mutex.Unlock()

	if l.file != nil {
		return l.file.Close()
	}
	return nil
}

// GetWriter returns an io.Writer interface for the logger at the specified level.
func (l *Logger) GetWriter(level LogLevel) io.Writer {
	return &logWriter{
		logger: l,
		level:  level,
	}
}

// logWriter is an io.Writer implementation that writes to the logger.
type logWriter struct {
	logger *Logger
	level  LogLevel
}

// Write implements the io.Writer interface.
func (w *logWriter) Write(p []byte) (n int, err error) {
	w.logger.log(w.level, string(p))
	return len(p), nil
}