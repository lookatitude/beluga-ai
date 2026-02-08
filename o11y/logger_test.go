package o11y

import (
	"context"
	"testing"
)

func TestNewLogger(t *testing.T) {
	t.Run("default logger", func(t *testing.T) {
		logger := NewLogger()
		if logger == nil {
			t.Fatal("expected non-nil logger")
		}
		if logger.Slog() == nil {
			t.Fatal("expected non-nil underlying slog.Logger")
		}
	})

	t.Run("with debug level", func(t *testing.T) {
		logger := NewLogger(WithLogLevel("debug"))
		if logger == nil {
			t.Fatal("expected non-nil logger")
		}
	})

	t.Run("with JSON output", func(t *testing.T) {
		logger := NewLogger(WithLogLevel("info"), WithJSON())
		if logger == nil {
			t.Fatal("expected non-nil logger")
		}
	})

	t.Run("with warn level", func(t *testing.T) {
		logger := NewLogger(WithLogLevel("warn"))
		if logger == nil {
			t.Fatal("expected non-nil logger")
		}
	})

	t.Run("with error level", func(t *testing.T) {
		logger := NewLogger(WithLogLevel("error"))
		if logger == nil {
			t.Fatal("expected non-nil logger")
		}
	})

	t.Run("unknown level defaults to info", func(t *testing.T) {
		logger := NewLogger(WithLogLevel("unknown"))
		if logger == nil {
			t.Fatal("expected non-nil logger")
		}
	})
}

func TestLoggerMethods(t *testing.T) {
	// Verify the log methods do not panic. We cannot easily capture slog
	// output in tests without a custom handler, but we can ensure no panics.
	logger := NewLogger(WithLogLevel("debug"))
	ctx := context.Background()

	logger.Info(ctx, "info message", "key", "value")
	logger.Error(ctx, "error message", "err", "something")
	logger.Debug(ctx, "debug message")
	logger.Warn(ctx, "warn message", "count", 42)
}

func TestLoggerWith(t *testing.T) {
	logger := NewLogger()
	derived := logger.With("component", "test")
	if derived == nil {
		t.Fatal("expected non-nil derived logger")
	}
	ctx := context.Background()
	derived.Info(ctx, "from derived logger")
}

func TestLoggerContext(t *testing.T) {
	t.Run("round-trip through context", func(t *testing.T) {
		logger := NewLogger(WithLogLevel("debug"))
		ctx := WithLogger(context.Background(), logger)

		got := FromContext(ctx)
		if got != logger {
			t.Error("expected same logger from context")
		}
	})

	t.Run("missing logger returns default", func(t *testing.T) {
		got := FromContext(context.Background())
		if got == nil {
			t.Fatal("expected non-nil default logger")
		}
	})
}
