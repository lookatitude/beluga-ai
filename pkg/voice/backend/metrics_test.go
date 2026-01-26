package backend

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/metric/noop"
	"go.opentelemetry.io/otel/trace"
)

func TestNewMetrics(t *testing.T) {
	meter := noop.NewMeterProvider().Meter("test")
	tracer := trace.NewNoopTracerProvider().Tracer("test")

	metrics, err := NewMetrics(meter, tracer)

	require.NoError(t, err)
	require.NotNil(t, metrics)
	assert.NotNil(t, metrics.requestsTotal)
	assert.NotNil(t, metrics.errorsTotal)
	assert.NotNil(t, metrics.latencySeconds)
	assert.NotNil(t, metrics.sessionsActive)
	assert.NotNil(t, metrics.sessionsTotal)
	assert.NotNil(t, metrics.throughputBytes)
	assert.NotNil(t, metrics.concurrentOps)
	assert.NotNil(t, metrics.sessionCreationTime)
	assert.NotNil(t, metrics.throughputPerSession)
	assert.NotNil(t, metrics.tracer)
}

func TestNewMetricsWithNilTracer(t *testing.T) {
	meter := noop.NewMeterProvider().Meter("test")

	metrics, err := NewMetrics(meter, nil)

	require.NoError(t, err)
	require.NotNil(t, metrics)
	assert.NotNil(t, metrics.tracer)
}

func TestNoOpMetrics(t *testing.T) {
	metrics := NoOpMetrics()

	require.NotNil(t, metrics)
	assert.NotNil(t, metrics.tracer)
	assert.Nil(t, metrics.requestsTotal)
	assert.Nil(t, metrics.errorsTotal)
}

func TestMetricsRecordRequest(t *testing.T) {
	t.Run("nil metrics", func(t *testing.T) {
		var m *Metrics
		// Should not panic
		m.RecordRequest(context.Background(), "test", time.Second)
	})

	t.Run("valid metrics", func(t *testing.T) {
		meter := noop.NewMeterProvider().Meter("test")
		tracer := trace.NewNoopTracerProvider().Tracer("test")
		m, err := NewMetrics(meter, tracer)
		require.NoError(t, err)

		// Should not panic
		m.RecordRequest(context.Background(), "livekit", 100*time.Millisecond)
	})
}

func TestMetricsRecordError(t *testing.T) {
	t.Run("nil metrics", func(t *testing.T) {
		var m *Metrics
		// Should not panic
		m.RecordError(context.Background(), "test", "error_code", time.Second)
	})

	t.Run("valid metrics", func(t *testing.T) {
		meter := noop.NewMeterProvider().Meter("test")
		tracer := trace.NewNoopTracerProvider().Tracer("test")
		m, err := NewMetrics(meter, tracer)
		require.NoError(t, err)

		// Should not panic
		m.RecordError(context.Background(), "livekit", ErrCodeConnectionFailed, 50*time.Millisecond)
	})

	t.Run("zero duration", func(t *testing.T) {
		meter := noop.NewMeterProvider().Meter("test")
		tracer := trace.NewNoopTracerProvider().Tracer("test")
		m, err := NewMetrics(meter, tracer)
		require.NoError(t, err)

		// Should not panic even with zero duration
		m.RecordError(context.Background(), "livekit", ErrCodeTimeout, 0)
	})
}

func TestMetricsRecordLatency(t *testing.T) {
	t.Run("nil metrics", func(t *testing.T) {
		var m *Metrics
		// Should not panic
		m.RecordLatency(context.Background(), "test", time.Second)
	})

	t.Run("noop metrics", func(t *testing.T) {
		m := NoOpMetrics()
		// Should not panic
		m.RecordLatency(context.Background(), "test", time.Second)
	})

	t.Run("valid metrics", func(t *testing.T) {
		meter := noop.NewMeterProvider().Meter("test")
		tracer := trace.NewNoopTracerProvider().Tracer("test")
		m, err := NewMetrics(meter, tracer)
		require.NoError(t, err)

		// Should not panic
		m.RecordLatency(context.Background(), "livekit", 200*time.Millisecond)
	})
}

func TestMetricsActiveSessions(t *testing.T) {
	t.Run("nil metrics", func(t *testing.T) {
		var m *Metrics
		// Should not panic
		m.IncrementActiveSessions(context.Background(), "test")
		m.DecrementActiveSessions(context.Background(), "test")
	})

	t.Run("noop metrics", func(t *testing.T) {
		m := NoOpMetrics()
		// Should not panic
		m.IncrementActiveSessions(context.Background(), "test")
		m.DecrementActiveSessions(context.Background(), "test")
	})

	t.Run("valid metrics", func(t *testing.T) {
		meter := noop.NewMeterProvider().Meter("test")
		tracer := trace.NewNoopTracerProvider().Tracer("test")
		m, err := NewMetrics(meter, tracer)
		require.NoError(t, err)

		// Should not panic
		m.IncrementActiveSessions(context.Background(), "livekit")
		m.DecrementActiveSessions(context.Background(), "livekit")
	})
}

func TestMetricsRecordThroughput(t *testing.T) {
	t.Run("nil metrics", func(t *testing.T) {
		var m *Metrics
		// Should not panic
		m.RecordThroughput(context.Background(), "test", 1024)
	})

	t.Run("noop metrics", func(t *testing.T) {
		m := NoOpMetrics()
		// Should not panic
		m.RecordThroughput(context.Background(), "test", 1024)
	})

	t.Run("valid metrics", func(t *testing.T) {
		meter := noop.NewMeterProvider().Meter("test")
		tracer := trace.NewNoopTracerProvider().Tracer("test")
		m, err := NewMetrics(meter, tracer)
		require.NoError(t, err)

		// Should not panic
		m.RecordThroughput(context.Background(), "livekit", 4096)
	})
}

func TestMetricsConcurrentOps(t *testing.T) {
	t.Run("nil metrics", func(t *testing.T) {
		var m *Metrics
		// Should not panic
		m.IncrementConcurrentOps(context.Background(), "test")
		m.DecrementConcurrentOps(context.Background(), "test")
	})

	t.Run("noop metrics", func(t *testing.T) {
		m := NoOpMetrics()
		// Should not panic
		m.IncrementConcurrentOps(context.Background(), "test")
		m.DecrementConcurrentOps(context.Background(), "test")
	})

	t.Run("valid metrics", func(t *testing.T) {
		meter := noop.NewMeterProvider().Meter("test")
		tracer := trace.NewNoopTracerProvider().Tracer("test")
		m, err := NewMetrics(meter, tracer)
		require.NoError(t, err)

		// Should not panic
		m.IncrementConcurrentOps(context.Background(), "livekit")
		m.DecrementConcurrentOps(context.Background(), "livekit")
	})
}

func TestMetricsRecordSessionCreationTime(t *testing.T) {
	t.Run("nil metrics", func(t *testing.T) {
		var m *Metrics
		// Should not panic
		m.RecordSessionCreationTime(context.Background(), "test", time.Second)
	})

	t.Run("noop metrics", func(t *testing.T) {
		m := NoOpMetrics()
		// Should not panic
		m.RecordSessionCreationTime(context.Background(), "test", time.Second)
	})

	t.Run("valid metrics", func(t *testing.T) {
		meter := noop.NewMeterProvider().Meter("test")
		tracer := trace.NewNoopTracerProvider().Tracer("test")
		m, err := NewMetrics(meter, tracer)
		require.NoError(t, err)

		// Should not panic
		m.RecordSessionCreationTime(context.Background(), "livekit", 500*time.Millisecond)
	})
}

func TestMetricsRecordThroughputPerSession(t *testing.T) {
	t.Run("nil metrics", func(t *testing.T) {
		var m *Metrics
		// Should not panic
		m.RecordThroughputPerSession(context.Background(), "test", "session-123", 2048)
	})

	t.Run("noop metrics", func(t *testing.T) {
		m := NoOpMetrics()
		// Should not panic
		m.RecordThroughputPerSession(context.Background(), "test", "session-123", 2048)
	})

	t.Run("valid metrics", func(t *testing.T) {
		meter := noop.NewMeterProvider().Meter("test")
		tracer := trace.NewNoopTracerProvider().Tracer("test")
		m, err := NewMetrics(meter, tracer)
		require.NoError(t, err)

		// Should not panic
		m.RecordThroughputPerSession(context.Background(), "livekit", "session-456", 8192)
	})
}
