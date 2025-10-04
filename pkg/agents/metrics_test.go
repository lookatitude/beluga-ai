package agents

import (
	"context"
	"testing"
	"time"

	"github.com/lookatitude/beluga-ai/pkg/agents/iface"
	"go.opentelemetry.io/otel/metric/noop"
	"go.opentelemetry.io/otel/trace"
)

// mockMeterProvider implements a simple meter provider for testing
type mockMeterProvider struct{}

func (m *mockMeterProvider) Meter(name string, opts ...interface{}) interface{} {
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
	})
}

// Test that Metrics implements the MetricsRecorder interface
func TestMetrics_InterfaceCompliance(t *testing.T) {
	tracer := trace.NewNoopTracerProvider().Tracer("test")
	meter := noop.NewMeterProvider().Meter("test")
	metrics := NewMetrics(meter, tracer)

	// This should compile without issues
	var _ iface.MetricsRecorder = metrics
}

// Benchmark tests
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
