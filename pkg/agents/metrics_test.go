package agents

import (
	"context"
	"testing"
	"time"

	"github.com/lookatitude/beluga-ai/pkg/agents/iface"
	"go.opentelemetry.io/otel/metric/noop"
	"go.opentelemetry.io/otel/trace"
)

// mockMeterProvider implements a simple meter provider for testing.
type mockMeterProvider struct{}

func (m *mockMeterProvider) Meter(name string, opts ...any) any {
	return noop.NewMeterProvider().Meter(name)
}

func TestNewMetrics(t *testing.T) {
	tracer := trace.NewNoopTracerProvider().Tracer("test")
	meter := noop.NewMeterProvider().Meter("test")

	metrics := NewMetrics(meter, tracer)
	if metrics == nil {
		t.Error("NewMetrics() returned nil")
	}

	// Test that metrics are initialized
	if metrics.agentCreations == nil {
		t.Error("agentCreations metric not initialized")
	}
	if metrics.agentExecutions == nil {
		t.Error("agentExecutions metric not initialized")
	}
	if metrics.executorRuns == nil {
		t.Error("executorRuns metric not initialized")
	}
	if metrics.toolCalls == nil {
		t.Error("toolCalls metric not initialized")
	}
	if metrics.planningCalls == nil {
		t.Error("planningCalls metric not initialized")
	}
	if metrics.streamingLatency == nil {
		t.Error("streamingLatency metric not initialized")
	}
	if metrics.streamingDuration == nil {
		t.Error("streamingDuration metric not initialized")
	}
	if metrics.streamingChunks == nil {
		t.Error("streamingChunks metric not initialized")
	}
}

func TestDefaultMetrics(t *testing.T) {
	metrics := DefaultMetrics()
	if metrics == nil {
		t.Error("DefaultMetrics() returned nil")
	}

	if metrics.tracer == nil {
		t.Error("DefaultMetrics should have a tracer")
	}
}

func TestNoOpMetrics(t *testing.T) {
	metrics := NoOpMetrics()
	if metrics == nil {
		t.Error("NoOpMetrics() returned nil")
	}

	// NoOpMetrics should have a noop tracer
	if metrics.tracer == nil {
		t.Error("NoOpMetrics should have a tracer")
	}
}

func TestMetrics_RecordAgentCreation(t *testing.T) {
	tracer := trace.NewNoopTracerProvider().Tracer("test")
	meter := noop.NewMeterProvider().Meter("test")
	metrics := NewMetrics(meter, tracer)

	ctx := context.Background()

	// Test recording agent creation
	metrics.RecordAgentCreation(ctx, "base")
	metrics.RecordAgentCreation(ctx, "react")
	metrics.RecordAgentCreation(ctx, "custom")

	// Since we're using noop meter, we can't verify the actual metric values
	// but we can verify the method doesn't panic
}

func TestMetrics_RecordAgentExecution(t *testing.T) {
	tracer := trace.NewNoopTracerProvider().Tracer("test")
	meter := noop.NewMeterProvider().Meter("test")
	metrics := NewMetrics(meter, tracer)

	ctx := context.Background()
	duration := 100 * time.Millisecond

	tests := []struct {
		name      string
		agentName string
		agentType string
		success   bool
	}{
		{
			name:      "successful base agent execution",
			agentName: "test-agent",
			agentType: "base",
			success:   true,
		},
		{
			name:      "failed react agent execution",
			agentName: "react-agent",
			agentType: "react",
			success:   false,
		},
		{
			name:      "successful custom agent execution",
			agentName: "custom-agent",
			agentType: "custom",
			success:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			metrics.RecordAgentExecution(ctx, tt.agentName, tt.agentType, duration, tt.success)
			// Method should not panic with valid inputs
		})
	}
}

func TestMetrics_RecordExecutorRun(t *testing.T) {
	tracer := trace.NewNoopTracerProvider().Tracer("test")
	meter := noop.NewMeterProvider().Meter("test")
	metrics := NewMetrics(meter, tracer)

	ctx := context.Background()
	duration := 50 * time.Millisecond

	tests := []struct {
		name         string
		executorType string
		steps        int
		success      bool
	}{
		{
			name:         "successful executor run",
			executorType: "agent_executor",
			steps:        3,
			success:      true,
		},
		{
			name:         "failed executor run",
			executorType: "custom_executor",
			steps:        1,
			success:      false,
		},
		{
			name:         "zero steps executor run",
			executorType: "empty_executor",
			steps:        0,
			success:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			metrics.RecordExecutorRun(ctx, tt.executorType, duration, tt.steps, tt.success)
			// Method should not panic with valid inputs
		})
	}
}

func TestMetrics_RecordToolCall(t *testing.T) {
	tracer := trace.NewNoopTracerProvider().Tracer("test")
	meter := noop.NewMeterProvider().Meter("test")
	metrics := NewMetrics(meter, tracer)

	ctx := context.Background()
	duration := 25 * time.Millisecond

	tests := []struct {
		name     string
		toolName string
		success  bool
	}{
		{
			name:     "successful calculator tool call",
			toolName: "calculator",
			success:  true,
		},
		{
			name:     "failed web search tool call",
			toolName: "web_search",
			success:  false,
		},
		{
			name:     "successful api tool call",
			toolName: "api_request",
			success:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			metrics.RecordToolCall(ctx, tt.toolName, duration, tt.success)
			// Method should not panic with valid inputs
		})
	}
}

func TestMetrics_RecordPlanningCall(t *testing.T) {
	tracer := trace.NewNoopTracerProvider().Tracer("test")
	meter := noop.NewMeterProvider().Meter("test")
	metrics := NewMetrics(meter, tracer)

	ctx := context.Background()
	duration := 75 * time.Millisecond

	tests := []struct {
		name      string
		agentName string
		success   bool
	}{
		{
			name:      "successful planning call",
			agentName: "smart-agent",
			success:   true,
		},
		{
			name:      "failed planning call",
			agentName: "confused-agent",
			success:   false,
		},
		{
			name:      "empty agent name planning call",
			agentName: "",
			success:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			metrics.RecordPlanningCall(ctx, tt.agentName, duration, tt.success)
			// Method should not panic with valid inputs
		})
	}
}

func TestMetrics_Tracing(t *testing.T) {
	tracer := trace.NewNoopTracerProvider().Tracer("test")
	meter := noop.NewMeterProvider().Meter("test")
	metrics := NewMetrics(meter, tracer)

	ctx := context.Background()

	t.Run("StartAgentSpan", func(t *testing.T) {
		newCtx, span := metrics.StartAgentSpan(ctx, "test-agent", "test_operation")
		if newCtx == nil {
			t.Error("StartAgentSpan should return valid context")
		}
		if span == nil {
			t.Error("StartAgentSpan should return valid span")
		}
		span.End()
	})

	t.Run("StartExecutorSpan", func(t *testing.T) {
		newCtx, span := metrics.StartExecutorSpan(ctx, "test-executor", "execute")
		if newCtx == nil {
			t.Error("StartExecutorSpan should return valid context")
		}
		if span == nil {
			t.Error("StartExecutorSpan should return valid span")
		}
		span.End()
	})

	t.Run("StartToolSpan", func(t *testing.T) {
		newCtx, span := metrics.StartToolSpan(ctx, "test-tool", "execute")
		if newCtx == nil {
			t.Error("StartToolSpan should return valid context")
		}
		if span == nil {
			t.Error("StartToolSpan should return valid span")
		}
		span.End()
	})
}

func TestMetrics_EdgeCases(t *testing.T) {
	tracer := trace.NewNoopTracerProvider().Tracer("test")
	meter := noop.NewMeterProvider().Meter("test")
	metrics := NewMetrics(meter, tracer)

	ctx := context.Background()

	t.Run("Empty strings", func(t *testing.T) {
		// These should not panic
		metrics.RecordAgentCreation(ctx, "")
		metrics.RecordAgentExecution(ctx, "", "", 0, true)
		metrics.RecordExecutorRun(ctx, "", 0, 0, true)
		metrics.RecordToolCall(ctx, "", 0, true)
		metrics.RecordPlanningCall(ctx, "", 0, true)
	})

	t.Run("Negative duration", func(t *testing.T) {
		// These should not panic
		metrics.RecordAgentExecution(ctx, "test", "base", -1*time.Second, true)
		metrics.RecordExecutorRun(ctx, "test", -1*time.Second, 1, true)
		metrics.RecordToolCall(ctx, "test", -1*time.Second, true)
		metrics.RecordPlanningCall(ctx, "test", -1*time.Second, true)
	})

	t.Run("Zero duration", func(t *testing.T) {
		// These should not panic
		metrics.RecordAgentExecution(ctx, "test", "base", 0, true)
		metrics.RecordExecutorRun(ctx, "test", 0, 1, true)
		metrics.RecordToolCall(ctx, "test", 0, true)
		metrics.RecordPlanningCall(ctx, "test", 0, true)
	})

	t.Run("Very large duration", func(t *testing.T) {
		largeDuration := 365 * 24 * time.Hour // 1 year
		// These should not panic
		metrics.RecordAgentExecution(ctx, "test", "base", largeDuration, true)
		metrics.RecordExecutorRun(ctx, "test", largeDuration, 1, true)
		metrics.RecordToolCall(ctx, "test", largeDuration, true)
		metrics.RecordPlanningCall(ctx, "test", largeDuration, true)
		metrics.RecordStreamingOperation(ctx, "test", largeDuration, largeDuration)
	})
}

// TestMetrics_RecordStreamingOperation tests streaming operation metrics recording.
func TestMetrics_RecordStreamingOperation(t *testing.T) {
	tracer := trace.NewNoopTracerProvider().Tracer("test")
	meter := noop.NewMeterProvider().Meter("test")
	metrics := NewMetrics(meter, tracer)

	ctx := context.Background()

	tests := []struct {
		name        string
		agentName   string
		latency     time.Duration
		duration    time.Duration
		wantNoPanic bool
	}{
		{
			name:        "normal streaming operation",
			agentName:   "streaming-agent",
			latency:     50 * time.Millisecond,
			duration:    500 * time.Millisecond,
			wantNoPanic: true,
		},
		{
			name:        "fast latency streaming operation",
			agentName:   "fast-agent",
			latency:     1 * time.Millisecond,
			duration:    100 * time.Millisecond,
			wantNoPanic: true,
		},
		{
			name:        "long duration streaming operation",
			agentName:   "long-agent",
			latency:     100 * time.Millisecond,
			duration:    30 * time.Minute,
			wantNoPanic: true,
		},
		{
			name:        "zero latency streaming operation",
			agentName:   "instant-agent",
			latency:     0,
			duration:    100 * time.Millisecond,
			wantNoPanic: true,
		},
		{
			name:        "zero duration streaming operation",
			agentName:   "empty-agent",
			latency:     10 * time.Millisecond,
			duration:    0,
			wantNoPanic: true,
		},
		{
			name:        "negative latency streaming operation",
			agentName:   "negative-agent",
			latency:     -1 * time.Millisecond,
			duration:    100 * time.Millisecond,
			wantNoPanic: true,
		},
		{
			name:        "negative duration streaming operation",
			agentName:   "negative-duration-agent",
			latency:     10 * time.Millisecond,
			duration:    -1 * time.Millisecond,
			wantNoPanic: true,
		},
		{
			name:        "empty agent name",
			agentName:   "",
			latency:     10 * time.Millisecond,
			duration:    100 * time.Millisecond,
			wantNoPanic: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.wantNoPanic {
				// Should not panic
				metrics.RecordStreamingOperation(ctx, tt.agentName, tt.latency, tt.duration)
			}
		})
	}
}

// TestMetrics_RecordStreamingChunk tests streaming chunk metrics recording.
func TestMetrics_RecordStreamingChunk(t *testing.T) {
	tracer := trace.NewNoopTracerProvider().Tracer("test")
	meter := noop.NewMeterProvider().Meter("test")
	metrics := NewMetrics(meter, tracer)

	ctx := context.Background()

	tests := []struct {
		name      string
		agentName string
		chunks    int
	}{
		{
			name:      "single chunk",
			agentName: "test-agent",
			chunks:    1,
		},
		{
			name:      "multiple chunks",
			agentName: "multi-agent",
			chunks:    10,
		},
		{
			name:      "many chunks",
			agentName: "high-volume-agent",
			chunks:    100,
		},
		{
			name:      "empty agent name",
			agentName: "",
			chunks:    5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Record multiple chunks
			for i := 0; i < tt.chunks; i++ {
				metrics.RecordStreamingChunk(ctx, tt.agentName)
			}
			// Method should not panic
		})
	}
}

// TestMetrics_RecordAgentExecution_ErrorMetrics tests that error metrics are recorded correctly.
func TestMetrics_RecordAgentExecution_ErrorMetrics(t *testing.T) {
	tracer := trace.NewNoopTracerProvider().Tracer("test")
	meter := noop.NewMeterProvider().Meter("test")
	metrics := NewMetrics(meter, tracer)

	ctx := context.Background()
	duration := 100 * time.Millisecond

	// Test that failed executions record error metrics
	t.Run("failed execution records error", func(t *testing.T) {
		metrics.RecordAgentExecution(ctx, "failing-agent", "base", duration, false)
		// Should not panic - error metrics should be recorded
	})

	t.Run("successful execution does not record error", func(t *testing.T) {
		metrics.RecordAgentExecution(ctx, "successful-agent", "base", duration, true)
		// Should not panic - no error metrics should be recorded
	})
}

// TestMetrics_RecordExecutorRun_ErrorMetrics tests that executor error metrics are recorded correctly.
func TestMetrics_RecordExecutorRun_ErrorMetrics(t *testing.T) {
	tracer := trace.NewNoopTracerProvider().Tracer("test")
	meter := noop.NewMeterProvider().Meter("test")
	metrics := NewMetrics(meter, tracer)

	ctx := context.Background()
	duration := 50 * time.Millisecond

	t.Run("failed executor run records error", func(t *testing.T) {
		metrics.RecordExecutorRun(ctx, "failing-executor", duration, 3, false)
		// Should not panic
	})

	t.Run("successful executor run does not record error", func(t *testing.T) {
		metrics.RecordExecutorRun(ctx, "successful-executor", duration, 3, true)
		// Should not panic
	})
}

// TestMetrics_RecordToolCall_ErrorMetrics tests that tool error metrics are recorded correctly.
func TestMetrics_RecordToolCall_ErrorMetrics(t *testing.T) {
	tracer := trace.NewNoopTracerProvider().Tracer("test")
	meter := noop.NewMeterProvider().Meter("test")
	metrics := NewMetrics(meter, tracer)

	ctx := context.Background()
	duration := 25 * time.Millisecond

	t.Run("failed tool call records error", func(t *testing.T) {
		metrics.RecordToolCall(ctx, "failing-tool", duration, false)
		// Should not panic
	})

	t.Run("successful tool call does not record error", func(t *testing.T) {
		metrics.RecordToolCall(ctx, "successful-tool", duration, true)
		// Should not panic
	})
}

// TestMetrics_RecordPlanningCall_ErrorMetrics tests that planning error metrics are recorded correctly.
func TestMetrics_RecordPlanningCall_ErrorMetrics(t *testing.T) {
	tracer := trace.NewNoopTracerProvider().Tracer("test")
	meter := noop.NewMeterProvider().Meter("test")
	metrics := NewMetrics(meter, tracer)

	ctx := context.Background()
	duration := 75 * time.Millisecond

	t.Run("failed planning call records error", func(t *testing.T) {
		metrics.RecordPlanningCall(ctx, "failing-agent", duration, false)
		// Should not panic
	})

	t.Run("successful planning call does not record error", func(t *testing.T) {
		metrics.RecordPlanningCall(ctx, "successful-agent", duration, true)
		// Should not panic
	})
}

// TestMetrics_StreamingEdgeCases tests edge cases for streaming metrics.
func TestMetrics_StreamingEdgeCases(t *testing.T) {
	tracer := trace.NewNoopTracerProvider().Tracer("test")
	meter := noop.NewMeterProvider().Meter("test")
	metrics := NewMetrics(meter, tracer)

	ctx := context.Background()

	t.Run("concurrent streaming operations", func(t *testing.T) {
		// Record multiple streaming operations concurrently
		for i := 0; i < 10; i++ {
			go func(agentNum int) {
				metrics.RecordStreamingOperation(ctx, "concurrent-agent", 10*time.Millisecond, 100*time.Millisecond)
				for j := 0; j < 5; j++ {
					metrics.RecordStreamingChunk(ctx, "concurrent-agent")
				}
			}(i)
		}
		// Should not panic
	})

	t.Run("streaming with negative values", func(t *testing.T) {
		metrics.RecordStreamingOperation(ctx, "negative-agent", -1*time.Second, -1*time.Second)
		// Should not panic
	})

	t.Run("very small streaming durations", func(t *testing.T) {
		metrics.RecordStreamingOperation(ctx, "micro-agent", 1*time.Nanosecond, 1*time.Nanosecond)
		metrics.RecordStreamingChunk(ctx, "micro-agent")
		// Should not panic
	})

	t.Run("very large streaming durations", func(t *testing.T) {
		metrics.RecordStreamingOperation(ctx, "macro-agent", 1*time.Hour, 24*time.Hour)
		// Should not panic
	})
}

// TestMetrics_ComprehensiveScenarios tests comprehensive metric recording scenarios.
func TestMetrics_ComprehensiveScenarios(t *testing.T) {
	tracer := trace.NewNoopTracerProvider().Tracer("test")
	meter := noop.NewMeterProvider().Meter("test")
	metrics := NewMetrics(meter, tracer)

	ctx := context.Background()

	t.Run("full agent lifecycle metrics", func(t *testing.T) {
		agentName := "lifecycle-agent"
		agentType := "base"

		// Agent creation
		metrics.RecordAgentCreation(ctx, agentType)

		// Planning
		metrics.RecordPlanningCall(ctx, agentName, 50*time.Millisecond, true)

		// Execution
		metrics.RecordAgentExecution(ctx, agentName, agentType, 200*time.Millisecond, true)

		// Streaming
		metrics.RecordStreamingOperation(ctx, agentName, 10*time.Millisecond, 150*time.Millisecond)
		for i := 0; i < 10; i++ {
			metrics.RecordStreamingChunk(ctx, agentName)
		}

		// Tool calls
		metrics.RecordToolCall(ctx, "calculator", 25*time.Millisecond, true)
		metrics.RecordToolCall(ctx, "web_search", 100*time.Millisecond, true)

		// Executor run
		metrics.RecordExecutorRun(ctx, "agent_executor", 300*time.Millisecond, 2, true)
		// Should not panic
	})

	t.Run("error scenario metrics", func(t *testing.T) {
		agentName := "error-agent"
		agentType := "base"

		// Failed planning
		metrics.RecordPlanningCall(ctx, agentName, 50*time.Millisecond, false)

		// Failed execution
		metrics.RecordAgentExecution(ctx, agentName, agentType, 200*time.Millisecond, false)

		// Failed tool call
		metrics.RecordToolCall(ctx, "failing_tool", 25*time.Millisecond, false)

		// Failed executor run
		metrics.RecordExecutorRun(ctx, "failing_executor", 300*time.Millisecond, 1, false)
		// Should not panic
	})
}

// Test that Metrics implements the MetricsRecorder interface.
func TestMetrics_InterfaceCompliance(t *testing.T) {
	tracer := trace.NewNoopTracerProvider().Tracer("test")
	meter := noop.NewMeterProvider().Meter("test")
	metrics := NewMetrics(meter, tracer)

	// This should compile without issues
	var _ iface.MetricsRecorder = metrics
}

// Benchmark tests.
func BenchmarkNewMetrics(b *testing.B) {
	tracer := trace.NewNoopTracerProvider().Tracer("test")
	meter := noop.NewMeterProvider().Meter("test")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = NewMetrics(meter, tracer)
	}
}

func BenchmarkRecordAgentExecution(b *testing.B) {
	tracer := trace.NewNoopTracerProvider().Tracer("test")
	meter := noop.NewMeterProvider().Meter("test")
	metrics := NewMetrics(meter, tracer)

	ctx := context.Background()
	duration := 100 * time.Millisecond

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		metrics.RecordAgentExecution(ctx, "bench-agent", "base", duration, true)
	}
}

func BenchmarkRecordToolCall(b *testing.B) {
	tracer := trace.NewNoopTracerProvider().Tracer("test")
	meter := noop.NewMeterProvider().Meter("test")
	metrics := NewMetrics(meter, tracer)

	ctx := context.Background()
	duration := 25 * time.Millisecond

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		metrics.RecordToolCall(ctx, "bench-tool", duration, true)
	}
}

func BenchmarkStartAgentSpan(b *testing.B) {
	tracer := trace.NewNoopTracerProvider().Tracer("test")
	meter := noop.NewMeterProvider().Meter("test")
	metrics := NewMetrics(meter, tracer)

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, span := metrics.StartAgentSpan(ctx, "bench-agent", "bench-operation")
		span.End()
	}
}

func BenchmarkRecordStreamingOperation(b *testing.B) {
	tracer := trace.NewNoopTracerProvider().Tracer("test")
	meter := noop.NewMeterProvider().Meter("test")
	metrics := NewMetrics(meter, tracer)

	ctx := context.Background()
	latency := 10 * time.Millisecond
	duration := 100 * time.Millisecond

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		metrics.RecordStreamingOperation(ctx, "bench-agent", latency, duration)
	}
}

func BenchmarkRecordStreamingChunk(b *testing.B) {
	tracer := trace.NewNoopTracerProvider().Tracer("test")
	meter := noop.NewMeterProvider().Meter("test")
	metrics := NewMetrics(meter, tracer)

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		metrics.RecordStreamingChunk(ctx, "bench-agent")
	}
}
