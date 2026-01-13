package session

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel/metric/noop"
	"go.opentelemetry.io/otel/trace"
)

func TestNewMetrics(t *testing.T) {
	meter := noop.NewMeterProvider().Meter("test")
	tracer := trace.NewNoopTracerProvider().Tracer("test")
	metrics := NewMetrics(meter, tracer)
	assert.NotNil(t, metrics)
}

func TestMetrics_RecordSessionStart(t *testing.T) {
	meter := noop.NewMeterProvider().Meter("test")
	tracer := trace.NewNoopTracerProvider().Tracer("test")
	metrics := NewMetrics(meter, tracer)
	ctx := context.Background()

	// Should not panic
	metrics.RecordSessionStart(ctx, "test-session", 10*time.Millisecond)
}

func TestMetrics_RecordSessionStop(t *testing.T) {
	meter := noop.NewMeterProvider().Meter("test")
	tracer := trace.NewNoopTracerProvider().Tracer("test")
	metrics := NewMetrics(meter, tracer)
	ctx := context.Background()

	// Should not panic
	metrics.RecordSessionStop(ctx, "test-session", 10*time.Millisecond)
}

func TestMetrics_RecordSessionError(t *testing.T) {
	meter := noop.NewMeterProvider().Meter("test")
	tracer := trace.NewNoopTracerProvider().Tracer("test")
	metrics := NewMetrics(meter, tracer)
	ctx := context.Background()

	// Should not panic
	metrics.RecordSessionError(ctx, "test-session", ErrCodeInternalError, 10*time.Millisecond)
}

func TestNoOpMetrics(t *testing.T) {
	noOp := NoOpMetrics()
	ctx := context.Background()

	// Should not panic
	noOp.RecordSessionStart(ctx, "test-session", 10*time.Millisecond)
	noOp.RecordSessionStop(ctx, "test-session", 10*time.Millisecond)
	noOp.RecordSessionError(ctx, "test-session", ErrCodeInternalError, 10*time.Millisecond)
	noOp.IncrementActiveSessions(ctx)
	noOp.DecrementActiveSessions(ctx)
}

// TestMetrics_RecordAgentOperation tests agent operation latency recording.
func TestMetrics_RecordAgentOperation(t *testing.T) {
	meter := noop.NewMeterProvider().Meter("test")
	tracer := trace.NewNoopTracerProvider().Tracer("test")
	metrics := NewMetrics(meter, tracer)
	ctx := context.Background()

	tests := []struct {
		name      string
		sessionID string
		latency   time.Duration
	}{
		{
			name:      "normal latency",
			sessionID: "test-session-1",
			latency:   100 * time.Millisecond,
		},
		{
			name:      "fast latency",
			sessionID: "test-session-2",
			latency:   10 * time.Millisecond,
		},
		{
			name:      "slow latency",
			sessionID: "test-session-3",
			latency:   1 * time.Second,
		},
		{
			name:      "zero latency",
			sessionID: "test-session-4",
			latency:   0,
		},
		{
			name:      "empty session ID",
			sessionID: "",
			latency:   50 * time.Millisecond,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Should not panic
			metrics.RecordAgentOperation(ctx, tt.sessionID, tt.latency)
		})
	}
}

// TestMetrics_RecordAgentStreamingChunk tests agent streaming chunk duration recording.
func TestMetrics_RecordAgentStreamingChunk(t *testing.T) {
	meter := noop.NewMeterProvider().Meter("test")
	tracer := trace.NewNoopTracerProvider().Tracer("test")
	metrics := NewMetrics(meter, tracer)
	ctx := context.Background()

	tests := []struct {
		name      string
		sessionID string
		duration  time.Duration
	}{
		{
			name:      "normal streaming duration",
			sessionID: "test-session-1",
			duration:  500 * time.Millisecond,
		},
		{
			name:      "short streaming duration",
			sessionID: "test-session-2",
			duration:  50 * time.Millisecond,
		},
		{
			name:      "long streaming duration",
			sessionID: "test-session-3",
			duration:  5 * time.Minute,
		},
		{
			name:      "zero duration",
			sessionID: "test-session-4",
			duration:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Should not panic
			metrics.RecordAgentStreamingChunk(ctx, tt.sessionID, tt.duration)
		})
	}
}

// TestMetrics_RecordAgentToolExecution tests agent tool execution time recording.
func TestMetrics_RecordAgentToolExecution(t *testing.T) {
	meter := noop.NewMeterProvider().Meter("test")
	tracer := trace.NewNoopTracerProvider().Tracer("test")
	metrics := NewMetrics(meter, tracer)
	ctx := context.Background()

	tests := []struct {
		name      string
		sessionID string
		toolName  string
		duration  time.Duration
	}{
		{
			name:      "calculator tool execution",
			sessionID: "test-session-1",
			toolName:  "calculator",
			duration:  25 * time.Millisecond,
		},
		{
			name:      "web search tool execution",
			sessionID: "test-session-2",
			toolName:  "web_search",
			duration:  200 * time.Millisecond,
		},
		{
			name:      "fast tool execution",
			sessionID: "test-session-3",
			toolName:  "fast_tool",
			duration:  1 * time.Millisecond,
		},
		{
			name:      "slow tool execution",
			sessionID: "test-session-4",
			toolName:  "slow_tool",
			duration:  2 * time.Second,
		},
		{
			name:      "empty tool name",
			sessionID: "test-session-5",
			toolName:  "",
			duration:  50 * time.Millisecond,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Should not panic
			metrics.RecordAgentToolExecution(ctx, tt.sessionID, tt.toolName, tt.duration)
		})
	}
}

// TestMetrics_RecordSessionStart_AllScenarios tests session start recording with various scenarios.
func TestMetrics_RecordSessionStart_AllScenarios(t *testing.T) {
	meter := noop.NewMeterProvider().Meter("test")
	tracer := trace.NewNoopTracerProvider().Tracer("test")
	metrics := NewMetrics(meter, tracer)
	ctx := context.Background()

	tests := []struct {
		name      string
		sessionID string
		duration  time.Duration
	}{
		{
			name:      "normal session start",
			sessionID: "session-1",
			duration:  50 * time.Millisecond,
		},
		{
			name:      "fast session start",
			sessionID: "session-2",
			duration:  5 * time.Millisecond,
		},
		{
			name:      "zero duration",
			sessionID: "session-3",
			duration:  0,
		},
		{
			name:      "empty session ID",
			sessionID: "",
			duration:  10 * time.Millisecond,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			metrics.RecordSessionStart(ctx, tt.sessionID, tt.duration)
			// Should not panic
		})
	}
}

// TestMetrics_RecordSessionStop_AllScenarios tests session stop recording with various scenarios.
func TestMetrics_RecordSessionStop_AllScenarios(t *testing.T) {
	meter := noop.NewMeterProvider().Meter("test")
	tracer := trace.NewNoopTracerProvider().Tracer("test")
	metrics := NewMetrics(meter, tracer)
	ctx := context.Background()

	tests := []struct {
		name      string
		sessionID string
		duration  time.Duration
	}{
		{
			name:      "normal session stop",
			sessionID: "session-1",
			duration:  30 * time.Minute,
		},
		{
			name:      "short session stop",
			sessionID: "session-2",
			duration:  1 * time.Second,
		},
		{
			name:      "long session stop",
			sessionID: "session-3",
			duration:  2 * time.Hour,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			metrics.RecordSessionStop(ctx, tt.sessionID, tt.duration)
			// Should not panic
		})
	}
}

// TestMetrics_RecordSessionError_AllErrorCodes tests error recording with all error codes.
func TestMetrics_RecordSessionError_AllErrorCodes(t *testing.T) {
	meter := noop.NewMeterProvider().Meter("test")
	tracer := trace.NewNoopTracerProvider().Tracer("test")
	metrics := NewMetrics(meter, tracer)
	ctx := context.Background()

	errorCodes := []string{
		ErrCodeInvalidConfig,
		ErrCodeInternalError,
		ErrCodeInvalidState,
		ErrCodeTimeout,
		ErrCodeSessionNotFound,
		ErrCodeSessionAlreadyActive,
		ErrCodeSessionNotActive,
		ErrCodeSessionExpired,
		ErrCodeContextCanceled,
		ErrCodeContextTimeout,
		ErrCodeAgentNotSet,
		ErrCodeAgentInvalid,
		ErrCodeStreamError,
		ErrCodeContextError,
		ErrCodeInterruptionError,
	}

	for _, code := range errorCodes {
		t.Run(code, func(t *testing.T) {
			metrics.RecordSessionError(ctx, "test-session", code, 10*time.Millisecond)
			// Should not panic
		})
	}
}

// TestMetrics_ActiveSessionsCounter tests active sessions counter operations.
func TestMetrics_ActiveSessionsCounter(t *testing.T) {
	meter := noop.NewMeterProvider().Meter("test")
	tracer := trace.NewNoopTracerProvider().Tracer("test")
	metrics := NewMetrics(meter, tracer)
	ctx := context.Background()

	// Test increment
	metrics.IncrementActiveSessions(ctx)

	// Test decrement
	metrics.DecrementActiveSessions(ctx)

	// Test multiple increments/decrements
	for i := 0; i < 10; i++ {
		metrics.IncrementActiveSessions(ctx)
	}
	for i := 0; i < 5; i++ {
		metrics.DecrementActiveSessions(ctx)
	}

	// Should not panic
}

// TestMetrics_EdgeCases tests edge cases for all metrics recording.
func TestMetrics_EdgeCases(t *testing.T) {
	meter := noop.NewMeterProvider().Meter("test")
	tracer := trace.NewNoopTracerProvider().Tracer("test")
	metrics := NewMetrics(meter, tracer)
	ctx := context.Background()

	t.Run("very large durations", func(t *testing.T) {
		largeDuration := 365 * 24 * time.Hour
		metrics.RecordSessionStart(ctx, "test", largeDuration)
		metrics.RecordSessionStop(ctx, "test", largeDuration)
		metrics.RecordAgentOperation(ctx, "test", largeDuration)
		metrics.RecordAgentStreamingChunk(ctx, "test", largeDuration)
		metrics.RecordAgentToolExecution(ctx, "test", "tool", largeDuration)
		// Should not panic
	})

	t.Run("negative durations", func(t *testing.T) {
		metrics.RecordSessionStart(ctx, "test", -1*time.Second)
		metrics.RecordSessionStop(ctx, "test", -1*time.Second)
		metrics.RecordAgentOperation(ctx, "test", -1*time.Second)
		// Should not panic
	})

	t.Run("concurrent metrics recording", func(t *testing.T) {
		done := make(chan bool)
		for i := 0; i < 10; i++ {
			go func(id int) {
				sessionID := fmt.Sprintf("session-%d", id)
				metrics.RecordSessionStart(ctx, sessionID, 10*time.Millisecond)
				metrics.IncrementActiveSessions(ctx)
				metrics.RecordAgentOperation(ctx, sessionID, 50*time.Millisecond)
				done <- true
			}(i)
		}

		for i := 0; i < 10; i++ {
			<-done
		}
		// Should not panic
	})
}
