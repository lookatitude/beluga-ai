package logger

import (
	"bytes"
	"context"
	"log"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewLogger(t *testing.T) {
	tests := []struct {
		name string
	}{
		{"basic logger"},
		{"another logger"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := NewLogger(tt.name)
			assert.NotNil(t, logger)
			assert.Equal(t, tt.name, logger.name)
			assert.Equal(t, INFO, logger.level)
			assert.True(t, logger.useColors)
		})
	}
}

func TestNewLoggerWithConfig(t *testing.T) {
	tests := []struct {
		name   string
		config LoggerConfig
		check  func(t *testing.T, logger *Logger)
	}{
		{
			name: "with debug level",
			config: LoggerConfig{
				Level:       DEBUG,
				EnableColor: false,
				UseConsole:  true,
			},
			check: func(t *testing.T, logger *Logger) {
				assert.Equal(t, DEBUG, logger.level)
				assert.False(t, logger.useColors)
			},
		},
		{
			name: "with file output",
			config: LoggerConfig{
				Level:       INFO,
				EnableColor: true,
				OutputFile:  "/tmp/test.log",
				UseConsole:  false,
			},
			check: func(t *testing.T, logger *Logger) {
				assert.Equal(t, INFO, logger.level)
				assert.True(t, logger.useColors)
				assert.NotNil(t, logger.fileOut)
				assert.Nil(t, logger.stdout)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := NewLoggerWithConfig(tt.name, tt.config)
			assert.NotNil(t, logger)
			assert.Equal(t, tt.name, logger.name)
			tt.check(t, logger)

			// Cleanup
			if logger.file != nil {
				logger.Close()
				os.Remove(tt.config.OutputFile)
			}
		})
	}
}

func TestLoggerSetLevel(t *testing.T) {
	logger := NewLogger("test")
	assert.Equal(t, INFO, logger.level)

	logger.SetLevel(DEBUG)
	assert.Equal(t, DEBUG, logger.level)

	logger.SetLevel(ERROR)
	assert.Equal(t, ERROR, logger.level)
}

func TestLoggerLogging(t *testing.T) {
	// Create a buffer to capture output
	var buf bytes.Buffer
	logger := &Logger{
		name:      "test",
		level:     DEBUG,
		stdout:    log.New(&buf, "", 0),
		useColors: false,
	}

	t.Run("debug logging", func(t *testing.T) {
		buf.Reset()
		logger.Debug("Debug message", "arg1", "arg2")

		output := buf.String()
		assert.Contains(t, output, "DEBUG")
		assert.Contains(t, output, "test")
		assert.Contains(t, output, "Debug message arg1 arg2")
	})

	t.Run("info logging", func(t *testing.T) {
		buf.Reset()
		logger.Info("Info message")

		output := buf.String()
		assert.Contains(t, output, "INFO")
		assert.Contains(t, output, "test")
		assert.Contains(t, output, "Info message")
	})

	t.Run("warning logging", func(t *testing.T) {
		buf.Reset()
		logger.Warning("Warning message")

		output := buf.String()
		assert.Contains(t, output, "WARN")
		assert.Contains(t, output, "test")
		assert.Contains(t, output, "Warning message")
	})

	t.Run("error logging", func(t *testing.T) {
		buf.Reset()
		logger.Error("Error message")

		output := buf.String()
		assert.Contains(t, output, "ERROR")
		assert.Contains(t, output, "test")
		assert.Contains(t, output, "Error message")
	})

	t.Run("level filtering", func(t *testing.T) {
		logger.SetLevel(WARNING)
		buf.Reset()
		logger.Debug("This should not appear")

		output := buf.String()
		assert.Empty(t, output, "Debug message should be filtered out")

		logger.Info("This should also not appear")
		output = buf.String()
		assert.Empty(t, output, "Info message should be filtered out")

		logger.Warning("This should appear")
		output = buf.String()
		assert.Contains(t, output, "This should appear")
	})
}

func TestLoggerColorize(t *testing.T) {
	logger := &Logger{
		name:      "test",
		useColors: true,
	}

	// Test colorization
	redText := logger.colorize(ERROR, "error message")
	assert.Contains(t, redText, "\033[31m") // Red color code
	assert.Contains(t, redText, "\033[0m")  // Reset code

	greenText := logger.colorize(INFO, "info message")
	assert.Contains(t, greenText, "\033[32m") // Green color code

	// Test no colorization
	logger.useColors = false
	plainText := logger.colorize(ERROR, "error message")
	assert.NotContains(t, plainText, "\033[")
}

func TestLoggerClose(t *testing.T) {
	// Test with file
	tempFile, err := os.CreateTemp("", "test_log_*.txt")
	require.NoError(t, err)
	defer os.Remove(tempFile.Name())

	logger := NewLoggerWithConfig("test", LoggerConfig{
		OutputFile: tempFile.Name(),
		UseConsole: false,
	})

	assert.NotNil(t, logger.file)
	assert.NotNil(t, logger.fileOut)

	err = logger.Close()
	assert.NoError(t, err)

	// Test without file
	logger2 := NewLogger("test2")
	err = logger2.Close()
	assert.NoError(t, err)
}

func TestLoggerGetWriter(t *testing.T) {
	logger := NewLogger("test")
	writer := logger.GetWriter(DEBUG)
	assert.NotNil(t, writer)

	// Test writing to the writer
	data := []byte("test message")
	n, err := writer.Write(data)
	assert.NoError(t, err)
	assert.Equal(t, len(data), n)
}

func TestLogWriter_Write(t *testing.T) {
	var buf bytes.Buffer
	logger := &Logger{
		name:      "test",
		level:     INFO,
		stdout:    log.New(&buf, "", 0),
		useColors: false,
	}

	writer := &logWriter{
		logger: logger,
		level:  INFO,
	}

	data := []byte("test writer message")
	n, err := writer.Write(data)
	assert.NoError(t, err)
	assert.Equal(t, len(data), n)

	output := buf.String()
	assert.Contains(t, output, "test writer message")
}

func TestStructuredLogger(t *testing.T) {
	var buf bytes.Buffer
	logger := &StructuredLogger{
		name:      "structured_test",
		level:     DEBUG,
		stdout:    log.New(&buf, "", 0),
		useColors: false,
		useJSON:   true,
	}

	ctx := context.Background()

	t.Run("JSON logging", func(t *testing.T) {
		buf.Reset()
		logger.Info(ctx, "Test message", map[string]interface{}{
			"key": "value",
			"count": 42,
		})

		output := buf.String()
		assert.Contains(t, output, `"level":"INFO"`)
		assert.Contains(t, output, `"logger":"structured_test"`)
		assert.Contains(t, output, `"message":"Test message"`)
		assert.Contains(t, output, `"key":"value"`)
		assert.Contains(t, output, `"count":42`)
	})

	t.Run("Plain text logging", func(t *testing.T) {
		logger.useJSON = false
		buf.Reset()
		logger.Info(ctx, "Plain text message", map[string]interface{}{
			"action": "test",
		})

		output := buf.String()
		assert.Contains(t, output, "INFO")
		assert.Contains(t, output, "structured_test")
		assert.Contains(t, output, "Plain text message")
		assert.Contains(t, output, "action=test")
	})
}

func TestStructuredLoggerWithFields(t *testing.T) {
	logger := NewStructuredLogger("test", WithColors(false))
	ctx := context.Background()

	contextLogger := logger.WithFields(map[string]interface{}{
		"component": "test_component",
		"version":   "1.0.0",
	})

	assert.NotNil(t, contextLogger)

	// Test that context logger merges fields
	var buf bytes.Buffer
	structuredLogger := logger.(*StructuredLogger)
	structuredLogger.stdout = log.New(&buf, "", 0)
	structuredLogger.useJSON = true

	contextLogger.Info(ctx, "Context message", map[string]interface{}{
		"action": "test_action",
	})

	output := buf.String()
	assert.Contains(t, output, `"component":"test_component"`)
	assert.Contains(t, output, `"version":"1.0.0"`)
	assert.Contains(t, output, `"action":"test_action"`)
}

func TestStructuredLoggerContext(t *testing.T) {
	logger := NewStructuredLogger("test", WithColors(false), WithJSONOutput())
	ctx := context.Background()

	// Add trace context
	ctx = context.WithValue(ctx, "trace_id", "test-trace-123")
	ctx = context.WithValue(ctx, "span_id", "test-span-456")

	var buf bytes.Buffer
	structuredLogger := logger.(*StructuredLogger)
	structuredLogger.stdout = log.New(&buf, "", 0)

	structuredLogger.Info(ctx, "Context test message", nil)

	output := buf.String()
	assert.Contains(t, output, `"trace_id":"test-trace-123"`)
	assert.Contains(t, output, `"span_id":"test-span-456"`)
}

func TestLogLevel(t *testing.T) {
	tests := []struct {
		level    LogLevel
		expected string
	}{
		{DEBUG, "DEBUG"},
		{INFO, "INFO"},
		{WARNING, "WARNING"},
		{ERROR, "ERROR"},
		{FATAL, "FATAL"},
		{LogLevel(999), "UNKNOWN"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.level.String())
		})
	}
}

func TestDefaultLoggerConfig(t *testing.T) {
	config := DefaultLoggerConfig()
	assert.Equal(t, INFO, config.Level)
	assert.True(t, config.EnableColor)
	assert.True(t, config.UseConsole)
	assert.Empty(t, config.OutputFile)
}

// Benchmark tests
func BenchmarkLogger_Debug(b *testing.B) {
	logger := NewLogger("bench")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logger.Debug("Benchmark debug message", "arg", i)
	}
}

func BenchmarkLogger_Info(b *testing.B) {
	logger := NewLogger("bench")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logger.Info("Benchmark info message %d", i)
	}
}

func BenchmarkStructuredLogger_JSON(b *testing.B) {
	logger := NewStructuredLogger("bench", WithJSONOutput(), WithColors(false))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logger.Info(context.Background(), "Benchmark JSON message", map[string]interface{}{
			"iteration": i,
			"timestamp": time.Now().Unix(),
		})
	}
}

func BenchmarkStructuredLogger_PlainText(b *testing.B) {
	logger := NewStructuredLogger("bench", WithColors(false))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logger.Info(context.Background(), "Benchmark plain text message", map[string]interface{}{
			"iteration": i,
			"timestamp": time.Now().Unix(),
		})
	}
}
