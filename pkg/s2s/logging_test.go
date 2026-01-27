package s2s

import (
	"bytes"
	"context"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSetLogger(t *testing.T) {
	// Create a custom logger
	var buf bytes.Buffer
	customLogger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))

	// Set the logger
	SetLogger(customLogger)

	// Verify logger is set
	assert.Equal(t, customLogger, Logger)
}

func TestLogProcess(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))
	SetLogger(logger)

	ctx := context.Background()

	// Test Info level
	buf.Reset()
	LogProcess(ctx, slog.LevelInfo, "Processing audio", "test-provider", "test-model", "session-123", nil)
	output := buf.String()
	assert.Contains(t, output, "Processing audio")
	assert.Contains(t, output, "test-provider")
	assert.Contains(t, output, "test-model")
	assert.Contains(t, output, "session-123")

	// Test Error level with error
	buf.Reset()
	err := assert.AnError
	LogProcess(ctx, slog.LevelError, "Processing failed", "test-provider", "test-model", "session-123", err)
	output = buf.String()
	assert.Contains(t, output, "Processing failed")
	assert.Contains(t, output, "error")
}

func TestLogStreaming(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))
	SetLogger(logger)

	ctx := context.Background()

	// Test Info level
	buf.Reset()
	LogStreaming(ctx, slog.LevelInfo, "Starting stream", "test-provider", "test-model", "session-123", nil)
	output := buf.String()
	assert.Contains(t, output, "Starting stream")
	assert.Contains(t, output, "test-provider")
	assert.Contains(t, output, "streaming")

	// Test Error level with error
	buf.Reset()
	err := assert.AnError
	LogStreaming(ctx, slog.LevelError, "Stream failed", "test-provider", "test-model", "session-123", err)
	output = buf.String()
	assert.Contains(t, output, "Stream failed")
	assert.Contains(t, output, "error")
}

func TestLogError(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{
		Level: slog.LevelError,
	}))
	SetLogger(logger)

	ctx := context.Background()

	buf.Reset()
	err := assert.AnError
	LogError(ctx, "S2S error occurred", "test-provider", "test-model", "ERR_CODE_001", "session-123", err)
	output := buf.String()
	assert.Contains(t, output, "S2S error occurred")
	assert.Contains(t, output, "test-provider")
	assert.Contains(t, output, "ERR_CODE_001")
	assert.Contains(t, output, "error")
}

func TestLogFallback(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	SetLogger(logger)

	ctx := context.Background()

	buf.Reset()
	LogFallback(ctx, "primary-provider", "fallback-provider", "session-123")
	output := buf.String()
	assert.Contains(t, output, "S2S provider fallback")
	assert.Contains(t, output, "primary-provider")
	assert.Contains(t, output, "fallback-provider")
	assert.Contains(t, output, "session-123")
}

func TestLogProcessLevels(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))
	SetLogger(logger)

	ctx := context.Background()

	// Test Debug level
	buf.Reset()
	LogProcess(ctx, slog.LevelDebug, "Debug message", "provider", "model", "session", nil)
	output := buf.String()
	assert.Contains(t, output, "Debug message")

	// Test Warn level
	buf.Reset()
	LogProcess(ctx, slog.LevelWarn, "Warning message", "provider", "model", "session", nil)
	output = buf.String()
	assert.Contains(t, output, "Warning message")

	// Test default level (Info)
	buf.Reset()
	LogProcess(ctx, slog.Level(999), "Default message", "provider", "model", "session", nil)
	output = buf.String()
	assert.Contains(t, output, "Default message")
}

func TestLoggingWithNilError(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	SetLogger(logger)

	ctx := context.Background()

	// Test that nil error doesn't cause issues
	buf.Reset()
	LogProcess(ctx, slog.LevelInfo, "Success", "provider", "model", "session", nil)
	output := buf.String()
	assert.Contains(t, output, "Success")
	assert.NotContains(t, output, "error")
}

func TestLoggingContext(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	SetLogger(logger)

	ctx := context.WithValue(context.Background(), "test-key", "test-value")

	// Test that context is passed through
	buf.Reset()
	LogProcess(ctx, slog.LevelInfo, "Context test", "provider", "model", "session", nil)
	output := buf.String()
	assert.Contains(t, output, "Context test")
}

func TestLoggingIntegration(t *testing.T) {
	// Test logging functions work together in a typical flow
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	SetLogger(logger)

	ctx := context.Background()
	provider := "amazon_nova"
	model := "nova-2-sonic"
	sessionID := "session-123"

	// Simulate a typical S2S flow
	LogProcess(ctx, slog.LevelInfo, "Starting S2S processing", provider, model, sessionID, nil)
	LogStreaming(ctx, slog.LevelInfo, "Streaming started", provider, model, sessionID, nil)
	LogFallback(ctx, "primary", "fallback", sessionID)
	LogError(ctx, "Processing error", provider, model, "ERR_001", sessionID, assert.AnError)

	output := buf.String()
	assert.Contains(t, output, "Starting S2S processing")
	assert.Contains(t, output, "Streaming started")
	assert.Contains(t, output, "S2S provider fallback")
	assert.Contains(t, output, "Processing error")
}
